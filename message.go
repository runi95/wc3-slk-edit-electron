package main

import (
	"encoding/json"
	"fmt"
	"github.com/asticode/go-astilectron"
	"github.com/asticode/go-astilectron-bootstrap"
	"github.com/runi95/wts-parser/models"
	"github.com/runi95/wts-parser/parser"
	"github.com/shibukawa/configdir"
	"gopkg.in/volatiletech/null.v6"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

const (
	// const MAXINT = 2147483647
	VENDOR_NAME              = "wc3-slk-edit"
	CONFIG_FILENAME          = "config.json"
	DISABLED_INPUTS_FILENAME = "disabled-inputs.json"
)

var (
	// Private Variables
	baseUnitMap    map[string]*models.SLKUnit
	unitFuncMap    map[string]*models.UnitFunc
	lastValidIndex int

	// Private Initialized Variables
	configDirs                   = configdir.New(VENDOR_NAME, "")
	configuration        *config = nil
	defaultDisabledUnits         = []string{
		"SLKUnit-UnitUI-Blend",
		"SLKUnit-UnitWeapons-Castbsw",
		"SLKUnit-UnitWeapons-Castpt",
		"SLKUnit-UnitUI-Run",
		"SLKUnit-UnitUI-Walk",
		"UnitFunc-Casterupgradeart",
		"SLKUnit-UnitData-Death",
		"SLKUnit-UnitUI-ElevPts",
		"SLKUnit-UnitUI-ElevRad",
		"SLKUnit-UnitUI-FogRad",
		"SLKUnit-UnitUI-ShadowOnWater",
		"UnitFunc-ScoreScreenIcon",
		"SLKUnit-UnitUI-MaxPitch",
		"SLKUnit-UnitUI-MaxRoll",
		"SLKUnit-UnitUI-FileVerFlags",
		"SLKUnit-UnitUI-OccH",
		"SLKUnit-UnitData-OrientInterp",
		"SLKUnit-UnitWeapons-ImpactZ",
		"SLKUnit-UnitWeapons-ImpactSwimZ",
		"SLKUnit-UnitWeapons-LaunchX",
		"SLKUnit-UnitWeapons-LaunchY",
		"SLKUnit-UnitWeapons-LaunchZ",
		"SLKUnit-UnitWeapons-LaunchSwimZ",
		"SLKUnit-UnitData-PropWin",
		"UnitFunc-Animprops",
		"UnitFunc-Attachmentanimprops",
		"SLKUnit-UnitUI-SelZ",
		"SLKUnit-UnitUI-SelCircOnWater",
		"UnitFunc-Description",
		"SLKUnit-UnitBalance-Repulse",
		"SLKUnit-UnitBalance-RepulseGroup",
		"SLKUnit-UnitBalance-RepulseParam",
		"SLKUnit-UnitBalance-RepulsePrio",
		"UnitFunc-Attachmentlinkprops",
		"UnitFunc-Boneprops",
		"SLKUnit-UnitUI-Special",
		"UnitFunc-Targetart",
		"UseExtendedLineOfSight",
		"SLKUnit-UnitBalance-MaxSpd",
		"SLKUnit-UnitBalance-MinSpd",
		"AIPlacementRadius",
		"AIPlacementType",
		"UnitFunc-Randomsoundlabel",
		"SLKUnit-UnitData-Formation",
		"SLKUnit-UnitData-Prio",
		"SLKUnit-UnitData-CargoSize",
		"DependencyEquivalents",
		"RequirementsLevels",
		"UpgradesUsed",
		"CasterUpgradeTip",
	}
)

type FieldToUnit struct {
	UnitId string
	Field  string
	Value  string
}

func HandleMessages(_ *astilectron.Window, m bootstrap.MessageIn) (payload interface{}, err error) {
	switch m.Name {
	case "saveFieldToUnit":
		var fieldToUnit FieldToUnit
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &fieldToUnit); err != nil {
				payload = err.Error()
				return
			}

			if _, ok := baseUnitMap[fieldToUnit.UnitId]; !ok {
				payload = "unsaved"
				return
			}

			split := strings.Split(fieldToUnit.Field, "-")
			nullString := new(null.String)
			nullString.SetValid(fieldToUnit.Value)
			if split[0] == "SLKUnit" {
				if split[1] == "UnitWeapons" {
					err = reflectUpdateValueOnFieldNullStruct(baseUnitMap[fieldToUnit.UnitId].UnitWeapons, *nullString, split[2])
				} else if split[1] == "UnitData" {
					err = reflectUpdateValueOnFieldNullStruct(baseUnitMap[fieldToUnit.UnitId].UnitData, *nullString, split[2])
				} else if split[1] == "UnitBalance" {
					err = reflectUpdateValueOnFieldNullStruct(baseUnitMap[fieldToUnit.UnitId].UnitBalance, *nullString, split[2])
				} else if split[1] == "UnitUI" {
					err = reflectUpdateValueOnFieldNullStruct(baseUnitMap[fieldToUnit.UnitId].UnitUI, *nullString, split[2])
				} else if split[1] == "UnitAbilities" {
					err = reflectUpdateValueOnFieldNullStruct(baseUnitMap[fieldToUnit.UnitId].UnitAbilities, *nullString, split[2])
				} else {
					err = fmt.Errorf("the field %s does not exist in SLKUnit", split[1])
				}
			} else if split[0] == "UnitFunc" {
				err = reflectUpdateValueOnFieldNullStruct(unitFuncMap[fieldToUnit.UnitId], *nullString, split[1])
			} else {
				err = fmt.Errorf("the field %s does not exist in UnitFunc", split[1])
			}

			if err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			payload = "success"
		}
	case "removeUnit":
		var unit string
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &unit); err != nil {
				payload = err.Error()
				return
			}

			delete(baseUnitMap, unit)
			delete(unitFuncMap, unit)
			payload = unit
		} else {
			payload = fmt.Errorf("invalid input")
		}
	case "getDisabledInputs":
		var isLocked bool
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &isLocked); err != nil {
				payload = err.Error()
				return
			}

			if isLocked != configuration.IsLocked {
				configuration.IsLocked = isLocked

				err = saveConfig()
				if err != nil {
					payload = err.Error()
					return
				}
			}

			config := loadConfigFile(DISABLED_INPUTS_FILENAME)
			if config != nil {
				var file []byte
				file, err = ioutil.ReadFile(config.Path + string(os.PathSeparator) + DISABLED_INPUTS_FILENAME)
				if err != nil {
					payload = err.Error()
					return
				}

				var disabledInputs []string
				err = json.Unmarshal([]byte(file), &disabledInputs)
				if err != nil {
					payload = err.Error()
					return
				}

				payload = disabledInputs
			} else {
				var file []byte
				file, err = json.Marshal(defaultDisabledUnits)
				if err != nil {
					payload = err.Error()
					return
				}

				err = saveConfigFile(DISABLED_INPUTS_FILENAME, file)
				if err != nil {
					payload = err.Error()
					return
				}

				payload = defaultDisabledUnits
			}
		}
	case "selectUnit":
		var unitId string
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &unitId); err != nil {
				payload = err.Error()
				return
			}

			var unitData = new(UnitData)
			unitData.UnitFunc = unitFuncMap[unitId]
			unitData.SLKUnit = baseUnitMap[unitId]

			payload = unitData
		} else {
			payload = fmt.Errorf("invalid input")
		}
	case "generateUnitId":
		payload = getNextValidUnitId(lastValidIndex)
	case "saveToFile":
		var outputDir string
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &outputDir); err != nil {
				payload = err.Error()
				return
			}

			configuration.OutDir = outputDir
			makeConfigAbsolute()

			saveUnitsToFile(configuration.OutDir)
			payload = "success" // TODO: Change this
		} else {
			payload = fmt.Errorf("invalid input")
		}
	case "saveUnit":
		var unit UnitData
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &unit); err != nil {
				log.Println(err.Error())
				payload = err.Error()
				return
			}

			baseUnitMap[unit.UnitFunc.UnitId] = unit.SLKUnit
			unitFuncMap[unit.UnitFunc.UnitId] = unit.UnitFunc

			payload = "success"
		} else {
			payload = fmt.Errorf("invalid input")
		}
	case "loadUnitData":
		loadSLK()
		var unitListData = make([]UnitListData, len(unitFuncMap))

		i := 0
		for k, v := range unitFuncMap {
			unitListData[i] = UnitListData{k, v.Name.String, v.Editorsuffix}
			i++
		}

		payload = unitListData
	case "initializeConfig":
		initializeConfiguration()
		payload = "success"
	case "loadConfig":
		payload = configuration
	case "setConfig":
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &configuration); err != nil {
				payload = err.Error()
				return
			}

			makeConfigAbsolute()
			err = saveConfig()
			if err != nil {
				payload = err.Error()
				return
			}

			payload = "success"
		} else {
			payload = fmt.Errorf("invalid input")
		}
	case "updateLock":
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &configuration.IsLocked); err != nil {
				payload = err.Error()
				return
			}

			err = saveConfig()
			if err != nil {
				payload = err.Error()
				return
			}

			payload = "success"
		}
	case "getOperatingSystem":
		payload = runtime.GOOS
	}

	return
}

/**
*    PRIVATE FUNCTIONS
*     - these are functions that are only called from within this file
 */
func reflectUpdateValueOnFieldNullStruct(iface interface{}, fieldValue interface{}, fieldName string) error {
	valueIface := reflect.ValueOf(iface)
	if valueIface.Type().Kind() != reflect.Ptr {
		return fmt.Errorf("can't swap values if the reflected interface is not a pointer")
	}

	// 'dereference' with Elem() and get the field by name
	field := valueIface.Elem().FieldByName(fieldName)
	if !field.IsValid() {
		return fmt.Errorf("interface `%s` does not have the field `%s`", valueIface.Type(), fieldName)
	}

	// A Value can be changed only if it is
	// addressable and was not obtained by
	// the use of unexported struct fields.
	if field.CanSet() {
		if field.Kind() == reflect.Struct {
			field.Set(reflect.ValueOf(fieldValue))
		} else {
			return fmt.Errorf("kind is not a struct")
		}
	} else {
		return fmt.Errorf("can't set value")
	}

	return nil
}

func initializeConfiguration() {
	configDirs.LocalPath, _ = filepath.Abs(".")

	err := loadConfig()
	if err != nil {
		log.Println(err)
	}

	if configuration.Version != "1.0.2" {
		configuration.IsLocked = false
		configuration.Version = "1.0.2"

		err = saveConfig()
		if err != nil {
			log.Println("An error occurred while updating the configuration to the newest version")
			if *debug {
				log.Println(err)
			}
		}
	}

	if *input != "" {
		configuration.InDir = *input
	}

	if *output != "" {
		configuration.OutDir = *output
	}

	makeConfigAbsolute()
}

func loadConfig() error {
	config := loadConfigFile(CONFIG_FILENAME)
	if config != nil {
		configFile, err := ioutil.ReadFile(config.Path + string(os.PathSeparator) + CONFIG_FILENAME)
		if err != nil {
			return err
		}

		err = json.Unmarshal(configFile, &configuration)
		if err != nil {
			return err
		}
	}

	return nil
}

func loadConfigFile(fileName string) *configdir.Config {
	return configDirs.QueryFolderContainsFile(fileName)
}

func saveConfig() error {
	confingInBytes, err := json.Marshal(configuration)
	if err != nil {
		return err
	}

	return saveConfigFile(CONFIG_FILENAME, confingInBytes)
}

func saveConfigFile(fileName string, data []byte) error {
	folders := configDirs.QueryFolders(configdir.Global)
	if len(folders) < 1 {
		return fmt.Errorf("failed to load global configuration")
	}

	return folders[0].WriteFile(fileName, data)
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

func intToHex(i int) string {
	if i < 10 {
		return fmt.Sprint(i)
	} else if i < 16 {
		return fmt.Sprint(string(55 + i))
	} else {
		return ""
	}
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
	for k := range unitFuncMap {
		unitFuncMap[k].Ubertip.SetValid(strings.Replace(unitFuncMap[k].Ubertip.String, "|n", "\n", -1))
	}

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
