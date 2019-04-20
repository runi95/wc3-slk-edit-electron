package main

import (
	"flag"
	"github.com/asticode/go-astilectron"
	"github.com/asticode/go-astilectron-bootstrap"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
	"github.com/runi95/wts-parser/models"
	"log"
)

var (
	// Public Variables
	AppName string
	BuiltAt string

	// Private Variables
	w *astilectron.Window

	// Flags
	debug  = flag.Bool("d", false, "enables the debug mode")
	input  = flag.String("input", "", "sets the input folder where the SLK files are stored")
	output = flag.String("output", "", "sets the output folder where we'll save the resulting SLK files")
)

/**
*    PUBLIC STRUCTURES
 */
type UnitListData struct {
	UnitID string
	Name   string
}

type UnitData struct {
	SLKUnit  *models.SLKUnit
	UnitFunc *models.UnitFunc
}

/**
*    PRIVATE STRUCTURES
 */
type config struct {
	InDir    string
	OutDir   string
	IsLocked bool
	Version  string
}

/**
*    PRIVATE FUNCTIONS
 */
func main() {
	log.Println("Starting up...")

	// Init
	flag.Parse()
	astilog.FlagInit()

	// Run bootstrap
	astilog.Debugf("Running app built at %s", BuiltAt)
	if err := bootstrap.Run(bootstrap.Options{
		Asset:    Asset,
		AssetDir: AssetDir,
		AstilectronOptions: astilectron.Options{
			AppName:            AppName,
			AppIconDarwinPath:  "resources/icon.icns",
			AppIconDefaultPath: "resources/icon.png",
		},
		Debug:       *debug,
		MenuOptions: []*astilectron.MenuItemOptions{},
		OnWait: func(_ *astilectron.Astilectron, ws []*astilectron.Window, _ *astilectron.Menu, _ *astilectron.Tray, _ *astilectron.Menu) error {
			w = ws[0]

			if *debug {
				w.OpenDevTools()
			}

			return nil
		},
		RestoreAssets: RestoreAssets,
		Windows: []*bootstrap.Window{
			{
				Homepage:       "index.html",
				MessageHandler: HandleMessages,
				Options: &astilectron.WindowOptions{
					Center:          astilectron.PtrBool(true),
					AutoHideMenuBar: astilectron.PtrBool(true),
				},
			}},
	}); err != nil {
		astilog.Fatal(errors.Wrap(err, "running bootstrap failed"))
	}
}
