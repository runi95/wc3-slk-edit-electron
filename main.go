package main

import (
	"flag"
	"fmt"
	"github.com/asticode/go-astilectron"
	"github.com/asticode/go-astilectron-bootstrap"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
	"github.com/runi95/wts-parser/models"
	"github.com/shibukawa/configdir"
	"gopkg.in/volatiletech/null.v6"
	"log"
	"os"
	"path/filepath"
)

var (
	// Public Variables
	AppName string
	BuiltAt string

	// Private Variables
	w *astilectron.Window

	// Flags
	debugFlag = flag.Bool("d", false, "enables the debug mode")
	input     = flag.String("input", "", "sets the input folder where the SLK files are stored")
	output    = flag.String("output", "", "sets the output folder where we'll save the resulting SLK files")
)

/**
*    PUBLIC STRUCTURES
 */
type UnitListData struct {
	UnitID       string
	Name         string
	EditorSuffix null.String
}

type UnitData struct {
	SLKUnit  *models.SLKUnit
	UnitFunc *models.UnitFunc
}

/**
*    PRIVATE STRUCTURES
 */
type config struct {
	InDir                   string
	OutDir                  string
	IsLocked                bool
	IsDoneDownloadingModels bool
	IsRegexSearch           bool
	Version                 string
}

type logWriter struct{}

/**
*    PRIVATE FUNCTIONS
 */
func (w *logWriter) Write(p []byte) (int, error) {
	var err error
	fmt.Println(string(p))

	folders := configDirs.QueryFolders(configdir.Global)
	if len(folders) < 1 {
		err = fmt.Errorf("failed to load output directories")

		fmt.Println(err)
		return 0, err
	}

	file, err := os.OpenFile(folders[0].Path+string(filepath.Separator)+"log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println(err)
		return 0, err
	}
	defer file.Close()

	return file.Write(p)
}

func main() {
	writer := new(logWriter)
	log.SetOutput(writer)

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
		Debug:       *debugFlag,
		MenuOptions: []*astilectron.MenuItemOptions{},
		OnWait: func(_ *astilectron.Astilectron, ws []*astilectron.Window, _ *astilectron.Menu, _ *astilectron.Tray, _ *astilectron.Menu) error {
			w = ws[0]

			if *debugFlag {
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
