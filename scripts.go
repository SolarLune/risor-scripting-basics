package main

import (
	"context"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/risor-io/risor"
	"github.com/risor-io/risor/builtins"
	"github.com/risor-io/risor/compiler"
	"github.com/risor-io/risor/importer"
	"github.com/risor-io/risor/object"
	"github.com/risor-io/risor/parser"
	"github.com/risor-io/risor/vm"
)

// The scripts file represents a script that can run.
var scriptImporter *importer.LocalImporter

// Set up the importer for importing libraries between modules
func init() {

	globalNames := []string{}
	sourceDir := "scripts/libraries"

	// builtins.Builtins() is built-ins that are normally automatically made available in an explicitly run Eval() function call (len(), keys(), etc.).
	for k := range builtins.Builtins() {
		globalNames = append(globalNames, k)
	}

	// Create the script importer; the source directory is the location of the source directory where modules can be found
	scriptImporter = importer.NewLocalImporter(importer.LocalImporterOptions{SourceDir: sourceDir, GlobalNames: globalNames})

	// Walk the source directory to import the modules by their module names (e.g. "Vec3"), not their filenames ("Vec3.rsr") and not their full paths ("assets/scripts/libraries/Vec3.rsr")
	filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil // Skip
		}
		_, file := filepath.Split(path)
		moduleName := strings.TrimSuffix(file, filepath.Ext(file))
		scriptImporter.Import(context.Background(), moduleName) // Import the module by its name
		return nil
	})

}

// Script represents an object that runs a script. The script (in this example) must be defined with some entry points (like OnInit(), OnUpdate(), etc)
// and has access to global Go variables. You could also just have the script run in its entirety if nothing were in functions.
type Script struct {
	Name        string
	VM          *vm.VirtualMachine
	Initialized bool
	Game        *Game
	LoadTime    time.Time

	data      map[string]any
	Functions map[string]*object.Function
	Running   bool
}

func NewScript(name string, game *Game) *Script {

	s := &Script{
		Name:      name,
		Game:      game,
		Functions: map[string]*object.Function{},
	}

	s.Reload()

	return s

}

// The reload function initializes the script; this can be called at will (i.e. when hotreloading detects a script changes) to reinitialize the script.
func (s *Script) Reload() {

	src, err := os.ReadFile("scripts/" + s.Name + ".rsr")

	if err != nil {
		log.Println(err.Error())
		return
	}

	ctx := context.Background()

	// A lot of the stuff we're doing below is done in the one-liner `risor.Eval()`, but
	// since we're going to be reusing almost all of the process (i.e. calling pre-compiled functions
	// frequently), we can do it ourselves, grab the functions, and then just call them when we want.

	// Parse the source code to create the AST
	ast, err := parser.Parse(ctx, string(src))
	if err != nil {
		log.Println(err)
		return
	}

	// We're gonna not clear out the data here on subsequent reloads to make hotreloading just a little bit hotter
	if s.data == nil {

		// By putting stuff in here, we make it accessible to our scripts
		s.data = map[string]any{
			"Lib":  s.Game.Library,
			"Data": &Data{data: map[string]any{}},
			"Self": &Self{
				script: s,
			},
		}
	}

	// We can do some Go <-> Risor interop by specifying globals here.
	cfg := risor.NewConfig(risor.WithImporter(scriptImporter), risor.WithGlobals(s.data))

	main, err := compiler.Compile(ast, cfg.CompilerOpts()...)
	if err != nil {
		log.Println(err)
		return
	}

	machine := vm.New(main, cfg.VMOpts()...)

	s.VM = machine

	// Run the compiled script. This allows us to get elements from the script that have run.
	if err := machine.Run(ctx); err != nil {
		log.Println(err)
		return
	}

	// Search for entry-point functions, store them.
	if update, err := machine.Get("OnUpdate"); err == nil {
		s.Functions["OnUpdate"] = update.(*object.Function)
	}

	if update, err := machine.Get("OnDraw"); err == nil {
		s.Functions["OnDraw"] = update.(*object.Function)
	}

	if init, err := machine.Get("OnInit"); err == nil {
		s.Functions["OnInit"] = init.(*object.Function)
	}

	if msg, err := machine.Get("OnMessage"); err == nil {
		s.Functions["OnMessage"] = msg.(*object.Function)
	}

	s.LoadTime = time.Now()

	s.Running = true

	// When reloaded, initialized is false
	// s.Initialized = false

}

func (s *Script) Loaded() bool {
	return s.VM != nil
}

func (s *Script) Update() {

	// s == nil if loading didn't work
	if s == nil {
		return
	}

	if !s.Initialized {
		s.Initialized = true
		s.RunFunc("OnInit")
	}

	s.RunFunc("OnUpdate")

}

func (s *Script) Draw(screen *ebiten.Image) {
	s.RunFunc("OnDraw")
}

func (s *Script) RunFunc(funcName string, args ...any) {

	if s.Running {

		if f, ok := s.Functions[funcName]; ok {

			var arguments []object.Object

			for _, a := range args {
				arguments = append(arguments, object.FromGoType(a))
			}

			_, err := s.VM.Call(context.Background(), f, arguments)
			if err != nil {
				log.Println(s.Name+".rsr:", err)
				s.Running = false
			}
		}

	}

}
