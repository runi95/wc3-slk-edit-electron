package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/asticode/go-astilectron"
	"github.com/asticode/go-astilectron-bootstrap"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
	"github.com/runi95/wts-parser/models"
	"github.com/runi95/wts-parser/parser"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var baseUnitMap map[string]*models.SLKUnit
var unitFuncMap map[string]*models.UnitFunc
var lastValidIndex int

var configuration *config = nil

const CONFIG_PATH = "./config.json"
const DISABLED_INPUTS_PATH = "./disabled-inputs.json"

/*
var Asset bootstrap.Asset
var AssetDir bootstrap.AssetDir
var RestoreAssets bootstrap.RestoreAssets
*/

// Vars
var (
	AppName string
	BuiltAt string
	debug   = flag.Bool("d", false, "enables the debug mode")
	w       *astilectron.Window
)

// const MAXINT = 2147483647

type config struct {
	InDir  string
	OutDir string
}

type UnitListData struct {
	UnitID string
	Name   string
}

type UnitData struct {
	SLKUnit  *models.SLKUnit
	UnitFunc *models.UnitFunc
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)

	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return true, err
}

func loadConfig() error {
	configExists, err := exists(CONFIG_PATH)
	if err == nil {
		if !configExists {
			return nil
		}

		configFile, err := ioutil.ReadFile(CONFIG_PATH)
		if err != nil {
			return err
		}

		err = json.Unmarshal(configFile, &configuration)
		if err != nil {
			return err
		}

		return nil
	} else if err != nil {
		return err
	}

	return fmt.Errorf("%s does not exist", CONFIG_PATH)
}

func makeConfigAbsolute() {
	if configuration == nil {
		return
	}

	absolutePathInDir, err := filepath.Abs(configuration.InDir)
	if err != nil {
		log.Println(err)

		return
	}

	configuration.InDir = absolutePathInDir

	absolutePathOutDir, err := filepath.Abs(configuration.OutDir)
	if err != nil {
		log.Println(err)

		return
	}

	configuration.OutDir = absolutePathOutDir
}

func setConfig() {
	err := loadConfig()
	if err != nil {
		log.Println(err)
	}

	makeConfigAbsolute()
}

func initializeConfiguration() {
	if len(os.Args) > 1 {
		inputDirectory := os.Args[1]
		outputDirectory := os.Args[2]

		configuration = &config{InDir: inputDirectory, OutDir: outputDirectory}

		makeConfigAbsolute()
	} else {
		setConfig()
	}
}

func saveConfig() error {
	confingInBytes, err := json.Marshal(configuration)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(CONFIG_PATH, confingInBytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

func loadSLK() {
	unitAbilitiesPath := filepath.Join(configuration.InDir, "UnitAbilities.slk")
	unitDataPath := filepath.Join(configuration.InDir, "UnitData.slk")
	unitUIPath := filepath.Join(configuration.InDir, "UnitUI.slk")
	unitWeaponsPath := filepath.Join(configuration.InDir, "UnitWeapons.slk")
	unitBalancePath := filepath.Join(configuration.InDir, "UnitBalance.slk")
	campaignUnitPath := filepath.Join(configuration.InDir, "CampaignUnitFunc.txt")

	if bool, err := exists(unitAbilitiesPath); err != nil || !bool {
		return
	}

	if bool, err := exists(unitDataPath); err != nil || !bool {
		return
	}

	if bool, err := exists(unitUIPath); err != nil || !bool {
		return
	}

	if bool, err := exists(unitWeaponsPath); err != nil || !bool {
		return
	}

	if bool, err := exists(unitBalancePath); err != nil || !bool {
		return
	}

	if bool, err := exists(campaignUnitPath); err != nil || !bool {
		return
	}

	log.Println("Reading UnitAbilities.slk...")

	unitAbilitiesBytes, err := ioutil.ReadFile(unitAbilitiesPath)
	if err != nil {
		log.Println(err)
		os.Exit(10)
	}

	unitAbilitiesMap := parser.SlkToUnitAbilities(unitAbilitiesBytes)

	log.Println("Reading UnitData.slk...")

	unitDataBytes, err := ioutil.ReadFile(unitDataPath)
	if err != nil {
		log.Println(err)
		os.Exit(10)
	}

	unitDataMap := parser.SlkToUnitData(unitDataBytes)

	log.Println("Reading UnitUI.slk...")

	unitUIBytes, err := ioutil.ReadFile(unitUIPath)
	if err != nil {
		log.Println(err)
		os.Exit(10)
	}

	unitUIMap := parser.SLKToUnitUI(unitUIBytes)

	log.Println("Reading UnitWeapons.slk...")

	unitWeaponsBytes, err := ioutil.ReadFile(unitWeaponsPath)
	if err != nil {
		log.Println(err)
		os.Exit(10)
	}

	unitWeaponsMap := parser.SLKToUnitWeapons(unitWeaponsBytes)

	log.Println("Reading UnitBalance.slk...")

	unitBalanceBytes, err := ioutil.ReadFile(unitBalancePath)
	if err != nil {
		log.Println(err)
		os.Exit(10)
	}

	unitBalanceMap := parser.SLKToUnitBalance(unitBalanceBytes)

	log.Println("Reading CampaignUnitFunc.txt...")

	campaignUnitFuncBytes, err := ioutil.ReadFile(campaignUnitPath)
	if err != nil {
		log.Println(err)
		os.Exit(10)
	}

	unitFuncMap = parser.TxtToUnitFunc(campaignUnitFuncBytes)

	baseUnitMap = make(map[string]*models.SLKUnit)
	i := 0
	for k := range unitDataMap {
		slkUnit := new(models.SLKUnit)
		slkUnit.UnitAbilities = unitAbilitiesMap[k]
		slkUnit.UnitData = unitDataMap[k]
		slkUnit.UnitUI = unitUIMap[k]
		slkUnit.UnitWeapons = unitWeaponsMap[k]
		slkUnit.UnitBalance = unitBalanceMap[k]

		unitId := strings.Replace(k, "\"", "", -1)
		baseUnitMap[unitId] = slkUnit
		i++
	}
}

func renderTemplate(w http.ResponseWriter, tmpl string) {
	t, err := template.ParseFiles("resources/app/" + tmpl)
	if err != nil {
		log.Println(err)
	}
	t.Execute(w, nil)
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Path[len("/"):]

	if fileName == "favicon.ico" {
		ico, _ := ioutil.ReadFile("/resources/app/favicon.ico")
		w.Write(ico)
	} else {
		renderTemplate(w, fileName)
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-web" {
		fs := http.FileServer(http.Dir("resources/app/static"))
		http.Handle("/static/", http.StripPrefix("/static/", fs))
		http.HandleFunc("/", viewHandler)
		log.Fatal(http.ListenAndServe(":8080", nil))
	} else {
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

				w.OpenDevTools()

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
}

func intToHex(i int) string {
	if i < 10 {
		return fmt.Sprint(i)
	} else if i < 16 {
		return fmt.Sprint(string(55 + i))
	} else {
		return ""
	}
}

func getNextValidUnitId(offset int) string {
	var str string

	if offset > 16383 {
		log.Println("Ran out of valid generated unit id's")
		return ""
	}

	var firstChar string

	switch offset / 4096 {
	case 0:
		firstChar = "u"
	case 1:
		firstChar = "n"
	case 2:
		firstChar = "h"
	case 3:
		firstChar = "o"
	case 4:
		firstChar = "e"
	default:
		firstChar = "u"
	}

	str = firstChar + intToHex(offset/256) + intToHex(int(offset/16)%16) + intToHex(offset%16)
	if _, ok := unitFuncMap[str]; !ok {
		lastValidIndex = offset
		return str
	}

	return getNextValidUnitId(offset + 1)
}

func saveUnitsToFile(location string) {
	customUnitFuncs := new(models.UnitFuncs)
	campaignUnitFuncs := make([]*models.UnitFunc, len(unitFuncMap))
	var campaignIndex = 0
	for _, k := range unitFuncMap {
		campaignUnitFuncs[campaignIndex] = k
		campaignIndex++
	}

	customUnitFuncs.CampaignUnitFuncs = campaignUnitFuncs

	unitMapLength := len(baseUnitMap)
	parsedSLKUnitsAbilities := make([]*models.UnitAbilities, unitMapLength)
	parsedSLKUnitsData := make([]*models.UnitData, unitMapLength)
	parsedSLKUnitsUI := make([]*models.UnitUI, unitMapLength)
	parsedSLKUnitsWeapons := make([]*models.UnitWeapons, unitMapLength)
	parsedSLKUnitsBalance := make([]*models.UnitBalance, unitMapLength)

	var i = 0
	for _, parsedSLKUnit := range baseUnitMap {
		parsedSLKUnitsAbilities[i] = parsedSLKUnit.UnitAbilities
		parsedSLKUnitsData[i] = parsedSLKUnit.UnitData
		parsedSLKUnitsUI[i] = parsedSLKUnit.UnitUI
		parsedSLKUnitsWeapons[i] = parsedSLKUnit.UnitWeapons
		parsedSLKUnitsBalance[i] = parsedSLKUnit.UnitBalance
		i++
	}

	parser.WriteToFilesAndSaveToFolder(customUnitFuncs, parsedSLKUnitsAbilities, parsedSLKUnitsData, parsedSLKUnitsUI, parsedSLKUnitsWeapons, parsedSLKUnitsBalance, location, true)
}
