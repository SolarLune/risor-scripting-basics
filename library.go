package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// This file represents glue-code that can be used to call Go-code freely.

type Library struct {
	// Unexported (Uncapitalized) fields are not visible to Risor
	screen       *EbitenImage
	game         *Game
	loadedAssets map[string]any
	frame        int
}

func (l *Library) Frame() int {
	return l.frame
}

func (l *Library) Add(scriptName string) {
	l.game.Add(scriptName)
}

// Returns the game screen
func (l *Library) Screen() *EbitenImage {
	return l.screen
}

func (l *Library) GameWidth() int {
	return l.game.Width
}

func (l *Library) GameHeight() int {
	return l.game.Height
}

type EbitenImage struct {
	img *ebiten.Image
}

func (e *EbitenImage) Width() int {
	return e.img.Bounds().Dx()
}

func (e *EbitenImage) Height() int {
	return e.img.Bounds().Dy()
}

func (e *EbitenImage) Draw(source *EbitenImage, x, y float64) {
	if e.img == nil {
		return
	}
	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(x, y)
	e.img.DrawImage(source.img, opt)
}

func (l *Library) LoadPNG(imgPath string) *EbitenImage {
	if a, ok := l.loadedAssets[imgPath]; ok {
		return a.(*EbitenImage)
	}
	img, _, err := ebitenutil.NewImageFromFile("assets/smile.png")
	if err != nil {
		panic(err)
	}
	ebitenImage := &EbitenImage{img: img}
	l.loadedAssets[imgPath] = ebitenImage
	return ebitenImage
}

type Data struct {
	data map[string]any
}

func (d *Data) Get(key string) any {
	if v, ok := d.data[key]; ok {
		return v
	}
	return nil
}

func (d *Data) Set(key string, value any) {
	d.data[key] = value
}

func (d *Data) SetAll(data map[string]any) {
	d.data = data
}

type Self struct {
	script *Script
}

func (s *Self) Remove() {
	s.script.Game.ToRemove = append(s.script.Game.ToRemove, s.script)
}
