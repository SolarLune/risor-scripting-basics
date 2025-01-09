package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	Width, Height int

	Library *Library

	Scripts  []*Script
	ToRemove []*Script
}

func NewGame() *Game {
	ebiten.SetWindowTitle("Risor Scripting Basics")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	g := &Game{
		Width:  640,
		Height: 360,
		Library: &Library{
			screen:       &EbitenImage{},
			loadedAssets: map[string]any{},
		},
	}

	g.Library.game = g

	g.Init()

	return g
}

func (g *Game) Init() {
	g.Scripts = []*Script{}

	// Add one smiley so we know it's working
	g.Add("Smiley")
}

func (g *Game) Add(scriptName string) {
	g.Scripts = append(g.Scripts, NewScript(scriptName, g))
}

func (g *Game) Update() error {

	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		g.Add("Smiley")
	}

	// Restart
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.Init()
	}

	// Quit
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	t := time.Now()
	for _, s := range g.Scripts {
		s.Update()
	}

	fmt.Println("Script Count:", len(g.Scripts))
	fmt.Println("Processing Cost:", time.Since(t))

	g.Library.frame++

	return nil

}

// Draw implements ebiten.Game.
func (g *Game) Draw(screen *ebiten.Image) {

	g.Library.screen.img = screen

	for _, s := range g.Scripts {
		s.Draw(screen)
	}

	g.HandleHotreloading()

	for _, r := range g.ToRemove {
		for i, s := range g.Scripts {
			if s == r {
				g.Scripts[i] = nil
				g.Scripts = append(g.Scripts[:i], g.Scripts[i+1:]...)
				break
			}
		}
	}

}

func (g *Game) HandleHotreloading() {

	// Handle hot-reloading for scripts
	filepath.WalkDir("scripts", func(path string, d fs.DirEntry, err error) error {

		// Skip directories.
		if d.IsDir() {
			return nil
		}

		// If you can't get the loadtime of a file, skip.
		info, err := os.Stat(path)
		if err != nil {
			panic(err)
		}

		// If the file's size is 0, it's being written to disk (at least, hopefully that's what that means).
		if info.Size() == 0 {
			return nil
		}

		filename := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))

		for _, script := range g.Scripts {
			if script.Name == filename && script.LoadTime.Before(info.ModTime()) {
				// time.Sleep(time.Millisecond * 30) // Give some time to try to ensure the file (should) be written out to the disk
				// This may not be necessary with the "Size() == 0" check above
				script.Reload()
			}
		}
		return nil
	})

}

// Layout implements ebiten.Game.
func (g *Game) Layout(outsideWidth int, outsideHeight int) (screenWidth int, screenHeight int) {
	return g.Width, g.Height
}

func main() {

	if err := ebiten.RunGame(NewGame()); err != nil {
		panic(err)
	}

}
