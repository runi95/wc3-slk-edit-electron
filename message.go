package main

import (
	"archive/zip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/asticode/go-astilectron"
	bootstrap "github.com/asticode/go-astilectron-bootstrap"
	"github.com/runi95/wts-parser/models"
	"github.com/runi95/wts-parser/parser"
	"github.com/shibukawa/configdir"
	"gopkg.in/volatiletech/null.v6"
)

const (
	// const MAXINT = 2147483647
	VENDOR_NAME              = "wc3-slk-edit"
	CONFIG_FILENAME          = "config.json"
	DISABLED_INPUTS_FILENAME = "disabled-inputs.json"
	MODEL_DOWNLOAD_URL       = "https://codeload.github.com/runi95/wc3-slk-edit-electron-resources/zip/master"
)

var (
	// Private Variables
	unitMap               map[string]*models.SLKUnit
	itemMap               map[string]*models.SLKItem
	abilityMap            map[string]*models.SLKAbility
	baseAbilityMap        map[string]*models.SLKAbility
	abilityMetaDataMap    map[string]*models.AbilityMetaData
	lastValidUnitIndex    int
	lastValidItemIndex    int
	lastValidAbilityIndex int

	// Private Initialized Variables
	configDirs                             = configdir.New(VENDOR_NAME, "")
	configuration                          = &config{InDir: nil, OutDir: nil, ResourceETag: nil, IsLocked: false, IsRegexSearch: false}
	globalConfig         *configdir.Config = nil
	defaultDisabledUnits                   = []string{
		"Unit-Blend",
		"Unit-Castbsw",
		"Unit-Castpt",
		"Unit-Run",
		"Unit-Walk",
		"Unit-Casterupgradeart",
		"Unit-Death",
		"Unit-ElevPts",
		"Unit-ElevRad",
		"Unit-FogRad",
		"Unit-ShadowOnWater",
		"Unit-ScoreScreenIcon",
		"Unit-MaxPitch",
		"Unit-MaxRoll",
		"Unit-FileVerFlags",
		"Unit-OccH",
		"Unit-OrientInterp",
		"Unit-ImpactZ",
		"Unit-ImpactSwimZ",
		"Unit-LaunchX",
		"Unit-LaunchY",
		"Unit-LaunchZ",
		"Unit-LaunchSwimZ",
		"Unit-PropWin",
		"Unit-Animprops",
		"Unit-Attachmentanimprops",
		"Unit-SelZ",
		"Unit-SelCircOnWater",
		"Unit-Description",
		"Unit-Repulse",
		"Unit-RepulseGroup",
		"Unit-RepulseParam",
		"Unit-RepulsePrio",
		"Unit-Attachmentlinkprops",
		"Unit-Boneprops",
		"Unit-Special",
		"Unit-Targetart",
		"UseExtendedLineOfSight",
		"Unit-MaxSpd",
		"Unit-MinSpd",
		"AIPlacementRadius",
		"AIPlacementType",
		"Unit-Randomsoundlabel",
		"Unit-Formation",
		"Unit-Prio",
		"Unit-CargoSize",
		"DependencyEquivalents",
		"RequirementsLevels",
		"UpgradesUsed",
		"CasterUpgradeTip",
	}
)

/**
*    PUBLIC STRUCTURES
 */
type ListData struct {
	Id           string
	Name         string
	EditorSuffix null.String
}

type SaveField struct {
	Id    string
	Field string
	Value string
}

type ConfigurationDirectories struct {
	InDir  *string
	OutDir *string
}

type EventMessage struct {
	Name    string
	Payload interface{}
}

type NewUnit struct {
	UnitId     null.String
	GenerateId bool
	Name       string
	UnitType   string
	BaseUnitId null.String
	AttackType string
}

type NewItem struct {
	ItemId     null.String
	GenerateId bool
	Name       string
	BaseItemId null.String
}

type NewAbility struct {
	Alias         null.String
	GenerateId    bool
	Name          string
	BaseAbilityId null.String
}

type FileInfo struct {
	FileName        string
	StatusClass     string
	StatusIconClass string
}

type Model struct {
	Name string
	Path string
}

type Models []Model

type GroupedModels struct {
	Units     Models
	Abilities Models
	Missiles  Models
	Items     Models
}

/**
*    PRIVATE STRUCTURES
 */
type config struct {
	InDir         *string
	OutDir        *string
	ResourceETag  *string
	IsLocked      bool
	IsRegexSearch bool
}

func (models Models) Len() int {
	return len(models)
}

func (models Models) Less(i, j int) bool {
	return models[i].Name < models[j].Name
}

func (models Models) Swap(i, j int) {
	models[i], models[j] = models[j], models[i]
}

func HandleMessages(w *astilectron.Window, m bootstrap.MessageIn) (payload interface{}, err error) {
	switch m.Name {
	case "saveField":
		var saveField SaveField
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &saveField); err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			fieldSplit := strings.Split(saveField.Field, "-")
			var ok bool
			var v interface{}
			if fieldSplit[0] == "Unit" {
				v, ok = unitMap[saveField.Id]
			} else if fieldSplit[0] == "Item" {
				v, ok = itemMap[saveField.Id]
			} else if fieldSplit[0] == "Ability" {
				v, ok = abilityMap[saveField.Id]
			} else {
				err = fmt.Errorf("invalid field name %v does not belong anywhere", saveField.Field)
				log.Println(err)
				return
			}

			if !ok {
				log.Println("The given id does not exist, returning unsaved")
				payload = "unsaved"
				return
			}

			split := strings.Split(saveField.Field, "-")

			nullString := new(null.String)
			if saveField.Value == "" || saveField.Value == "_" || saveField.Value == "\"_\"" || saveField.Value == "-" || saveField.Value == "\"-\"" {
				nullString.Valid = false
			} else {
				nullString.SetValid(saveField.Value)
			}

			err = reflectUpdateValueOnFieldNullStruct(v, *nullString, split[1])
			if err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			payload = "success"
		}
	case "fetchMdxModel":
		var path string
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &path); err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			folders := configDirs.QueryFolders(configdir.Global)
			if len(folders) < 1 {
				err = fmt.Errorf("failed to load config directory")

				log.Println(err)
				return 0, err
			}

			data, err := ioutil.ReadFile(folders[0].Path + string(filepath.Separator) + path)
			if err != nil {
				log.Println(err)
				return 0, fmt.Errorf("failed to fetch mdx models")
			}

			encoded := base64.StdEncoding.EncodeToString(data)
			payload = encoded
		}
	case "removeUnit":
		var unit string
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &unit); err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			delete(unitMap, unit)
			payload = unit
		} else {
			err = fmt.Errorf("invalid input")

			log.Println(err)
			payload = err.Error()
		}
	case "removeItem":
		var item string
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &item); err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			delete(itemMap, item)
			payload = item
		} else {
			err = fmt.Errorf("invalid input")

			log.Println(err)
			payload = err.Error()
		}
	case "removeAbility":
		var ability string
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &ability); err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			delete(abilityMap, ability)
			payload = ability
		} else {
			err = fmt.Errorf("invalid input")

			log.Println(err)
			payload = err.Error()
		}
	case "getDisabledInputs":
		var isLocked bool
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &isLocked); err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			if isLocked != configuration.IsLocked {
				configuration.IsLocked = isLocked

				err = saveConfig()
				if err != nil {
					log.Println(err)
					payload = err.Error()
					return
				}
			}

			config := loadConfigFile(DISABLED_INPUTS_FILENAME)
			if config != nil {
				var file []byte
				file, err = ioutil.ReadFile(config.Path + string(os.PathSeparator) + DISABLED_INPUTS_FILENAME)
				if err != nil {
					log.Println(err)
					payload = err.Error()
					return
				}

				var disabledInputs []string
				err = json.Unmarshal([]byte(file), &disabledInputs)
				if err != nil {
					log.Println(err)
					payload = err.Error()
					return
				}

				payload = disabledInputs
			} else {
				var file []byte
				file, err = json.Marshal(defaultDisabledUnits)
				if err != nil {
					log.Println(err)
					payload = err.Error()
					return
				}

				err = saveConfigFile(DISABLED_INPUTS_FILENAME, file)
				if err != nil {
					log.Println(err)
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
				log.Println(err)
				payload = err.Error()
				return
			}

			payload = unitMap[unitId]
		} else {
			err = fmt.Errorf("invalid input")

			log.Println(err)
			payload = err.Error()
		}
	case "selectItem":
		var itemId string
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &itemId); err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			payload = itemMap[itemId]
		} else {
			err = fmt.Errorf("invalid input")

			log.Println(err)
			payload = err.Error()
		}
	case "selectAbility":
		var abilityId string
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &abilityId); err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			payload = abilityMap[abilityId]
		} else {
			err = fmt.Errorf("invalid input")

			log.Println(err)
			payload = err.Error()
		}
	case "generateUnitId":
		payload = getNextValidUnitId(lastValidUnitIndex)
	case "generateItemId":
		payload = getNextValidItemId(lastValidItemIndex)
	case "generateAbilityId":
		payload = getNextValidAbilityId(lastValidAbilityIndex)
	case "saveToFile":
		if configuration.OutDir != nil {
			saveUnitsToFile(*configuration.OutDir)
		}

		payload = configuration.OutDir
	case "loadIcon":
		var imagePath string
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &imagePath); err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			payload = images[strings.Replace(strings.Replace(imagePath, "ReplaceableTextures\\CommandButtons", "Command", 1), "ReplaceableTextures\\PassiveButtons", "Passive", 1)]
		}
	case "saveUnit":
		var unit models.SLKUnit
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &unit); err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			unitMap[unit.UnitID.String] = &unit

			payload = "success"
		} else {
			err = fmt.Errorf("invalid input")

			log.Println(err)
			payload = err.Error()
		}
	case "loadSlk":
		payload = loadSLK()
	case "loadData":
		err = loadData()
		if err != nil {
			payload = err.Error()
			return
		}

		payload = "success"
	case "loadBaseAbilityData":
		var baseAbilityKeyList = make([]string, len(baseAbilityMap))
		i := 0
		for k := range baseAbilityMap {
			baseAbilityKeyList[i] = k
			i++
		}

		payload = baseAbilityKeyList
	case "loadAbilityMetaData":
		payload = abilityMetaDataMap
	case "loadUnitData":
		var unitListData = make([]ListData, len(unitMap))

		i := 0
		for k, v := range unitMap {
			unitListData[i] = ListData{k, v.UnitString.Name.String, v.Editorsuffix}
			i++
		}

		payload = unitListData
	case "loadItemData":
		var itemListData = make([]ListData, len(itemMap))

		i := 0
		for k, v := range itemMap {
			itemListData[i] = ListData{k, v.Name.String, v.Editorsuffix}
			i++
		}

		payload = itemListData
	case "loadAbilityData":
		var abilityListData = make([]ListData, len(abilityMap))

		i := 0
		for k, v := range abilityMap {
			abilityListData[i] = ListData{k, v.Name.String, v.Editorsuffix}
			i++
		}

		payload = abilityListData
	case "loadIcons":
		iconModels := make(Models, 0, len(images))
		for k := range images {
			path := strings.Replace(strings.Replace(k, "Command", "ReplaceableTextures\\CommandButtons", 1), "Passive", "ReplaceableTextures\\PassiveButtons", 1)
			name := strings.Replace(strings.Replace(strings.Replace(k, "Command\\BTN", "", 1), "Passive\\PASBTN", "", 1), ".blp", "", 1)

			iconModels = append(iconModels, Model{name, path})
		}

		sort.Sort(iconModels)

		payload = iconModels
	case "loadConfig":
		queryResult := configDirs.QueryFolders(configdir.Global)
		if len(queryResult) < 1 {
			err = fmt.Errorf("failed to locate the configuration directory")
			log.Println(err)
			CrashWithMessage(w, err.Error())
			payload = err.Error()
			return
		}

		globalConfig = queryResult[0]
		configPath := globalConfig.Path + string(filepath.Separator) + CONFIG_FILENAME

		var flag bool
		if flag, err = exists(configPath); err != nil || !flag {
			err = saveConfig()
			if err != nil {
				log.Println(err)
				CrashWithMessage(w, err.Error())
				payload = err.Error()
				return
			}
		} else {
			var fileData []byte
			fileData, err = ioutil.ReadFile(configPath)
			if err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			if err = json.Unmarshal(fileData, &configuration); err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}
		}

		if input != nil && *input != "" {
			configuration.InDir = input
		}

		if output != nil && *output != "" {
			configuration.OutDir = output
		}

		payload = configuration
	case "saveOptions":
		if m.Payload != nil {
			var configurationDirectories ConfigurationDirectories

			if err = json.Unmarshal(m.Payload, &configurationDirectories); err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			configuration.InDir = configurationDirectories.InDir
			configuration.OutDir = configurationDirectories.OutDir

			if configuration.InDir != nil {
				absolutePathInDir, err := filepath.Abs(*configuration.InDir)
				if err != nil {
					log.Println(err)
				}

				configuration.InDir = &absolutePathInDir
			}

			if configuration.OutDir != nil {
				absolutePathOutDir, err := filepath.Abs(*configuration.OutDir)
				if err != nil {
					log.Println(err)
				}

				configuration.OutDir = &absolutePathOutDir
			}

			err = saveConfig()
			if err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			payload = configuration
		} else {
			err = fmt.Errorf("invalid configuration")

			log.Println(err)
			payload = err.Error()
		}
	case "updateLock":
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &configuration.IsLocked); err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			err = saveConfig()
			if err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			payload = "success"
		}
	case "createNewUnit":
		if m.Payload != nil {
			var newUnit NewUnit
			if err = json.Unmarshal(m.Payload, &newUnit); err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			unit := new(models.SLKUnit)
			unit.UnitUI = new(models.UnitUI)
			unit.UnitData = new(models.UnitData)
			unit.UnitBalance = new(models.UnitBalance)
			unit.UnitWeapons = new(models.UnitWeapons)
			unit.UnitAbilities = new(models.UnitAbilities)
			unit.UnitFunc = new(models.UnitFunc)
			unit.UnitString = new(models.UnitString)
			unit.UnitString.Name.SetValid(newUnit.Name)

			var unitId string
			if newUnit.GenerateId == true || !newUnit.UnitId.Valid {
				unitId = getNextValidUnitId(lastValidUnitIndex)
			} else {
				unitId = newUnit.UnitId.String
			}

			unit.UnitFuncId.SetValid(unitId)
			unit.UnitStringId.SetValid(unitId)
			unit.UnitAbilID.SetValid(unitId)
			unit.UnitBalanceID.SetValid(unitId)
			unit.UnitID.SetValid(unitId)
			unit.UnitUIID.SetValid(unitId)
			unit.UnitWeapID.SetValid(unitId)

			unitType := strings.ToLower(newUnit.UnitType)
			if unitType == "unit" {
				// UnitFunc
				unit.Art.SetValid("ReplaceableTextures\\CommandButtons\\BTNFootman.blp")
				unit.ButtonposX.SetValid("0")
				unit.ButtonposY.SetValid("0")
				unit.Buttonpos.SetValid("0,0")
				unit.Specialart.SetValid("Objects\\Spawnmodels\\Human\\HumanLargeDeathExplode\\HumanLargeDeathExplode.mdl")

				// UnitString
				unit.Hotkey.SetValid("F")
				unit.Tip.SetValid("Train " + newUnit.Name + " [|cffffcc00F|r]")
				unit.Ubertip.SetValid("Versatile foot soldier. Can learn the Defend ability. |n|n|cffffcc00Attacks land units.|r")

				// UnitAbilities
				unit.Auto.SetValid("_")
				unit.AbilList.SetValid("Adef,Aihn")

				// UnitBalance
				unit.Level.SetValid("2")
				unit.Type.SetValid("\"_\"")
				unit.Goldcost.SetValid("135")
				unit.Lumbercost.SetValid("0")
				unit.GoldRep.SetValid("135")
				unit.LumberRep.SetValid("0")
				unit.Fmade.SetValid("\"-\"")
				unit.Fused.SetValid("2")
				unit.Bountydice.SetValid("6")
				unit.Bountysides.SetValid("3")
				unit.Bountyplus.SetValid("20")
				unit.Lumberbountydice.SetValid("0")
				unit.Lumberbountysides.SetValid("0")
				unit.Lumberbountyplus.SetValid("0")
				unit.StockMax.SetValid("3")
				unit.StockRegen.SetValid("30")
				unit.StockStart.SetValid("0")
				unit.HP.SetValid("420")
				unit.RealHP.SetValid("420")
				unit.RegenHP.SetValid("0.25")
				unit.RegenType.SetValid("\"always\"")
				unit.ManaN.SetValid("\"-\"")
				unit.RealM.SetValid("\"-\"")
				unit.Mana0.SetValid("\"-\"")
				unit.Def.SetValid("2")
				unit.DefUp.SetValid("2")
				unit.Realdef.SetValid("2")
				unit.DefType.SetValid("\"large\"")
				unit.Spd.SetValid("270")
				unit.MinSpd.SetValid("0")
				unit.MaxSpd.SetValid("0")
				unit.Bldtm.SetValid("20")
				unit.Reptm.SetValid("20")
				unit.Sight.SetValid("1400")
				unit.Nsight.SetValid("800")
				unit.STR.SetValid("\"-\"")
				unit.INT.SetValid("\"-\"")
				unit.AGI.SetValid("\"-\"")
				unit.STRplus.SetValid("\"-\"")
				unit.INTplus.SetValid("\"-\"")
				unit.AGIplus.SetValid("\"-\"")
				unit.Primary.SetValid("\"_\"")
				unit.Primary.SetValid("\"Rhar,Rhme,Rhde,Rhpm,Rguv\"")
				unit.Isbldg.SetValid("0")
				unit.PreventPlace.SetValid("\"_\"")
				unit.RequirePlace.SetValid("\"_\"")
				unit.Collision.SetValid("31")

				// UnitData
				unit.Race.SetValid("\"human\"")
				unit.Prio.SetValid("6")
				unit.Threat.SetValid("1")
				unit.Valid.SetValid("1")
				unit.DeathType.SetValid("3")
				unit.Death.SetValid("3.04")
				unit.CanSleep.SetValid("0")
				unit.CargoSize.SetValid("1")
				unit.Movetp.SetValid("\"foot\"")
				unit.MoveHeight.SetValid("0")
				unit.MoveFloor.SetValid("0")
				unit.TurnRate.SetValid("0.6")
				unit.PropWin.SetValid("60")
				unit.OrientInterp.SetValid("0")
				unit.Formation.SetValid("0")
				unit.TargType.SetValid("\"ground\"")
				unit.PathTex.SetValid("\"_\"")
				unit.Points.SetValid("100")
				unit.CanFlee.SetValid("1")
				unit.RequireWaterRadius.SetValid("0")
				unit.IsBuildOn.SetValid("0")
				unit.CanBuildOn.SetValid("0")

				// UnitUI
				unit.File.SetValid("\"units\\human\\Footman\\Footman\"")
				unit.FileVerFlags.SetValid("0")
				unit.UnitSound.SetValid("\"Footman\"")
				unit.TilesetSpecific.SetValid("0")
				unit.UnitClass.SetValid("\"HUnit02\"")
				unit.Special.SetValid("0")
				unit.Campaign.SetValid("0")
				unit.InEditor.SetValid("1")
				unit.HiddenInEditor.SetValid("0")
				unit.HostilePal.SetValid("\"-\"")
				unit.DropItems.SetValid("1")
				unit.NbmmIcon.SetValid("\"-\"")
				unit.UseClickHelper.SetValid("0")
				unit.HideHeroBar.SetValid("0")
				unit.HideHeroMinimap.SetValid("0")
				unit.HideHeroDeathMsg.SetValid("0")
				unit.HideOnMinimap.SetValid("0")
				unit.Blend.SetValid("0.15")
				unit.Scale.SetValid("1")
				unit.ScaleBull.SetValid("1")
				unit.MaxPitch.SetValid("10")
				unit.MaxRoll.SetValid("10")
				unit.ElevPts.SetValid("\"-\"")
				unit.ElevRad.SetValid("20")
				unit.FogRad.SetValid("0")
				unit.Walk.SetValid("210")
				unit.Run.SetValid("210")
				unit.SelZ.SetValid("0")
				unit.Weap1.SetValid("\"MetalMediumSlice\"")
				unit.Weap2.SetValid("\"_\"")
				unit.TeamColor.SetValid("-1")
				unit.CustomTeamColor.SetValid("0")
				unit.Armor.SetValid("\"Metal\"")
				unit.ModelScale.SetValid("1")
				unit.Red.SetValid("255")
				unit.Green.SetValid("255")
				unit.Blue.SetValid("255")
				unit.UberSplat.SetValid("\"_\"")
				unit.UnitShadow.SetValid("\"Shadow\"")
				unit.BuildingShadow.SetValid("\"_\"")
				unit.ShadowW.SetValid("140")
				unit.ShadowH.SetValid("140")
				unit.ShadowX.SetValid("50")
				unit.ShadowY.SetValid("50")
				unit.ShadowOnWater.SetValid("1")
				unit.SelCircOnWater.SetValid("0")
				unit.OccH.SetValid("0")
			} else if unitType == "building" {
				// UnitFunc
				unit.Art.SetValid("ReplaceableTextures\\CommandButtons\\BTNFarm.blp")
				unit.ButtonposX.SetValid("0")
				unit.ButtonposY.SetValid("1")
				unit.Buttonpos.SetValid("0,1")
				unit.Buildingsoundlabel.SetValid("BuildingConstructionLoop")
				unit.Loopingsoundfadein.SetValid("512")
				unit.Loopingsoundfadeout.SetValid("512")
				unit.Specialart.SetValid("Objects\\Spawnmodels\\Human\\HCancelDeath\\HCancelDeath.mdl")

				// UnitAbilities
				unit.Auto.SetValid("_")
				unit.AbilList.SetValid("Abds")

				// UnitBalance
				unit.Level.SetValid("\"-\"")
				unit.Type.SetValid("\"Mechanical\"")
				unit.Goldcost.SetValid("80")
				unit.Lumbercost.SetValid("20")
				unit.GoldRep.SetValid("80")
				unit.LumberRep.SetValid("20")
				unit.Fmade.SetValid("6")
				unit.Fused.SetValid("\"-\"")
				unit.Bountydice.SetValid("0")
				unit.Bountysides.SetValid("0")
				unit.Bountyplus.SetValid("0")
				unit.Lumberbountydice.SetValid("0")
				unit.Lumberbountysides.SetValid("0")
				unit.Lumberbountyplus.SetValid("0")
				unit.StockMax.SetValid("\"-\"")
				unit.StockRegen.SetValid("\"-\"")
				unit.StockStart.SetValid("\"-\"")
				unit.HP.SetValid("500")
				unit.RealHP.SetValid("500")
				unit.RegenHP.SetValid("\"-\"")
				unit.RegenType.SetValid("\"none\"")
				unit.ManaN.SetValid("\"-\"")
				unit.RealM.SetValid("\"-\"")
				unit.Mana0.SetValid("\"-\"")
				unit.Def.SetValid("5")
				unit.DefUp.SetValid("1")
				unit.Realdef.SetValid("5")
				unit.DefType.SetValid("\"fort\"")
				unit.Spd.SetValid("\"-\"")
				unit.MinSpd.SetValid("0")
				unit.MaxSpd.SetValid("0")
				unit.Bldtm.SetValid("35")
				unit.Reptm.SetValid("35")
				unit.Sight.SetValid("900")
				unit.Nsight.SetValid("600")
				unit.STR.SetValid("\"-\"")
				unit.INT.SetValid("\"-\"")
				unit.AGI.SetValid("\"-\"")
				unit.STRplus.SetValid("\"-\"")
				unit.INTplus.SetValid("\"-\"")
				unit.AGIplus.SetValid("\"-\"")
				unit.Primary.SetValid("\"_\"")
				unit.Primary.SetValid("\"Rhac,Rgfo\"")
				unit.Isbldg.SetValid("1")
				unit.PreventPlace.SetValid("\"unbuildable\"")
				unit.RequirePlace.SetValid("\"_\"")
				unit.Collision.SetValid("72")

				// UnitData
				unit.Race.SetValid("\"human\"")
				unit.Prio.SetValid("1")
				unit.Threat.SetValid("1")
				unit.Valid.SetValid("1")
				unit.DeathType.SetValid("2")
				unit.Death.SetValid("2.34")
				unit.CanSleep.SetValid("0")
				unit.CargoSize.SetValid("\"-\"")
				unit.Movetp.SetValid("\"_\"")
				unit.MoveHeight.SetValid("0")
				unit.MoveFloor.SetValid("0")
				unit.TurnRate.SetValid("\"-\"")
				unit.PropWin.SetValid("60")
				unit.OrientInterp.SetValid("0")
				unit.Formation.SetValid("0")
				unit.TargType.SetValid("\"structure\"")
				unit.PathTex.SetValid("\"PathTextures\\4x4SimpleSolid.tga\"")
				unit.Points.SetValid("100")
				unit.CanFlee.SetValid("1")
				unit.RequireWaterRadius.SetValid("0")
				unit.IsBuildOn.SetValid("0")
				unit.CanBuildOn.SetValid("0")

				// UnitUI
				unit.File.SetValid("\"buildings\\human\\Farm\\Farm\"")
				unit.FileVerFlags.SetValid("0")
				unit.UnitSound.SetValid("\"Farm\"")
				unit.TilesetSpecific.SetValid("0")
				unit.UnitClass.SetValid("\"HBuilding04\"")
				unit.Special.SetValid("0")
				unit.Campaign.SetValid("0")
				unit.InEditor.SetValid("1")
				unit.HiddenInEditor.SetValid("0")
				unit.HostilePal.SetValid("\"-\"")
				unit.DropItems.SetValid("1")
				unit.NbmmIcon.SetValid("\"-\"")
				unit.UseClickHelper.SetValid("0")
				unit.HideHeroBar.SetValid("0")
				unit.HideHeroMinimap.SetValid("0")
				unit.HideHeroDeathMsg.SetValid("0")
				unit.HideOnMinimap.SetValid("0")
				unit.Blend.SetValid("0.15")
				unit.Scale.SetValid("2.5")
				unit.ScaleBull.SetValid("1")
				unit.MaxPitch.SetValid("15")
				unit.MaxRoll.SetValid("15")
				unit.ElevPts.SetValid("4")
				unit.ElevRad.SetValid("50")
				unit.FogRad.SetValid("0")
				unit.Walk.SetValid("200")
				unit.Run.SetValid("200")
				unit.SelZ.SetValid("0")
				unit.Weap1.SetValid("\"_\"")
				unit.Weap2.SetValid("\"_\"")
				unit.TeamColor.SetValid("-1")
				unit.CustomTeamColor.SetValid("0")
				unit.Armor.SetValid("\"Wood\"")
				unit.ModelScale.SetValid("1")
				unit.Red.SetValid("255")
				unit.Green.SetValid("255")
				unit.Blue.SetValid("255")
				unit.UberSplat.SetValid("\"HSMA\"")
				unit.UnitShadow.SetValid("\"_\"")
				unit.BuildingShadow.SetValid("\"ShadowHouse\"")
				unit.ShadowOnWater.SetValid("1")
				unit.SelCircOnWater.SetValid("0")
				unit.OccH.SetValid("0")
			} else if unitType == "hero" {
				// TODO: Implement
			}

			if newUnit.AttackType == "0" { // None
				unit.WeapsOn.SetValid("0")
				unit.Acquire.SetValid("\"-\"")
				unit.MinRange.SetValid("\"-\"")
				unit.Castpt.SetValid("\"-\"")
				unit.Castbsw.SetValid("0.51")
				unit.LaunchX.SetValid("0")
				unit.LaunchY.SetValid("0")
				unit.LaunchZ.SetValid("60")
				unit.LaunchSwimZ.SetValid("0")
				unit.ImpactZ.SetValid("120")
				unit.ImpactSwimZ.SetValid("0")
				unit.WeapType1.SetValid("\"_\"")
				unit.Targs1.SetValid("\"_\"")
				unit.ShowUI1.SetValid("1")
				unit.RangeN1.SetValid("\"-\"")
				unit.RngTst.SetValid("\"-\"")
				unit.RngBuff1.SetValid("\"-\"")
				unit.AtkType1.SetValid("\"normal\"")
				unit.WeapTp1.SetValid("\"-\"")
				unit.Cool1.SetValid("\"-\"")
				unit.Mincool1.SetValid("\"-\"")
				unit.Dice1.SetValid("\"-\"")
				unit.Sides1.SetValid("\"-\"")
				unit.Dmgplus1.SetValid("\"-\"")
				unit.DmgUp1.SetValid("\"-\"")
				unit.Mindmg1.SetValid("\"-\"")
				unit.Avgdmg1.SetValid("\"-\"")
				unit.Maxdmg1.SetValid("\"-\"")
				unit.Dmgpt1.SetValid("\"-\"")
				unit.BackSw1.SetValid("\"-\"")
				unit.Farea1.SetValid("\"-\"")
				unit.Harea1.SetValid("\"-\"")
				unit.Qarea1.SetValid("\"-\"")
				unit.Hfact1.SetValid("\"-\"")
				unit.Qfact1.SetValid("\"-\"")
				unit.SplashTargs1.SetValid("\"_\"")
				unit.TargCount1.SetValid("\"-\"")
				unit.DamageLoss1.SetValid("0")
				unit.SpillDist1.SetValid("0")
				unit.SpillRadius1.SetValid("0")
				unit.DmgUpg.SetValid("\"-\"")
				unit.Dmod1.SetValid("\"-\"")
				unit.DPS.SetValid("\"-\"")
				unit.WeapType2.SetValid("\"_\"")
				unit.Targs2.SetValid("\"_\"")
				unit.ShowUI2.SetValid("\"-\"")
				unit.RangeN2.SetValid("\"-\"")
				unit.RngTst2.SetValid("\"-\"")
				unit.RngBuff2.SetValid("\"-\"")
				unit.AtkType2.SetValid("\"normal\"")
				unit.WeapTp2.SetValid("\"_\"")
				unit.Cool2.SetValid("\"-\"")
				unit.Mincool2.SetValid("\"-\"")
				unit.Dice2.SetValid("\"-\"")
				unit.Sides2.SetValid("\"-\"")
				unit.Dmgplus2.SetValid("\"-\"")
				unit.DmgUp2.SetValid("\"-\"")
				unit.Mindmg2.SetValid("\"-\"")
				unit.Avgdmg2.SetValid("\"-\"")
				unit.Maxdmg2.SetValid("\"-\"")
				unit.Dmgpt2.SetValid("\"-\"")
				unit.BackSw2.SetValid("\"-\"")
				unit.Farea2.SetValid("\"-\"")
				unit.Harea2.SetValid("\"-\"")
				unit.Qarea2.SetValid("\"-\"")
				unit.Hfact2.SetValid("\"-\"")
				unit.Qfact2.SetValid("\"-\"")
				unit.SplashTargs2.SetValid("\"_\"")
				unit.TargCount2.SetValid("\"-\"")
				unit.DamageLoss2.SetValid("0")
				unit.SpillDist2.SetValid("0")
				unit.SpillRadius2.SetValid("0")
			} else if newUnit.AttackType == "1" { // Melee
				unit.WeapsOn.SetValid("1")
				unit.Acquire.SetValid("500")
				unit.MinRange.SetValid("\"-\"")
				unit.Castpt.SetValid("0.3")
				unit.Castbsw.SetValid("0.51")
				unit.LaunchX.SetValid("0")
				unit.LaunchY.SetValid("0")
				unit.LaunchZ.SetValid("60")
				unit.LaunchSwimZ.SetValid("0")
				unit.ImpactZ.SetValid("60")
				unit.ImpactSwimZ.SetValid("0")
				unit.WeapType1.SetValid("\"MetalMediumSlice\"")
				unit.Targs1.SetValid("\"ground,structure,debris,item,ward\"")
				unit.ShowUI1.SetValid("1")
				unit.RangeN1.SetValid("90")
				unit.RngTst.SetValid("\"-\"")
				unit.RngBuff1.SetValid("250")
				unit.AtkType1.SetValid("\"normal\"")
				unit.WeapTp1.SetValid("\"normal\"")
				unit.Cool1.SetValid("1.35")
				unit.Mincool1.SetValid("\"-\"")
				unit.Dice1.SetValid("1")
				unit.Sides1.SetValid("2")
				unit.Dmgplus1.SetValid("11")
				unit.DmgUp1.SetValid("\"-\"")
				unit.Mindmg1.SetValid("12")
				unit.Avgdmg1.SetValid("12.5")
				unit.Maxdmg1.SetValid("13")
				unit.Dmgpt1.SetValid("0.5")
				unit.BackSw1.SetValid("0.5")
				unit.Farea1.SetValid("\"-\"")
				unit.Harea1.SetValid("\"-\"")
				unit.Qarea1.SetValid("\"-\"")
				unit.Hfact1.SetValid("\"-\"")
				unit.Qfact1.SetValid("\"-\"")
				unit.SplashTargs1.SetValid("\"_\"")
				unit.TargCount1.SetValid("1")
				unit.DamageLoss1.SetValid("0")
				unit.SpillDist1.SetValid("0")
				unit.SpillRadius1.SetValid("0")
				unit.DmgUpg.SetValid("\"-\"")
				unit.Dmod1.SetValid("\"-\"")
				unit.DPS.SetValid("9.25925925925926")
				unit.WeapType2.SetValid("\"_\"")
				unit.Targs2.SetValid("\"_\"")
				unit.ShowUI2.SetValid("1")
				unit.RangeN2.SetValid("\"-\"")
				unit.RngTst2.SetValid("\"-\"")
				unit.RngBuff2.SetValid("\"-\"")
				unit.AtkType2.SetValid("\"normal\"")
				unit.WeapTp2.SetValid("\"_\"")
				unit.Cool2.SetValid("\"-\"")
				unit.Mincool2.SetValid("\"-\"")
				unit.Dice2.SetValid("\"-\"")
				unit.Sides2.SetValid("\"-\"")
				unit.Dmgplus2.SetValid("\"-\"")
				unit.DmgUp2.SetValid("\"-\"")
				unit.Mindmg2.SetValid("\"-\"")
				unit.Avgdmg2.SetValid("\"-\"")
				unit.Maxdmg2.SetValid("\"-\"")
				unit.Dmgpt2.SetValid("\"-\"")
				unit.BackSw2.SetValid("\"-\"")
				unit.Farea2.SetValid("\"-\"")
				unit.Harea2.SetValid("\"-\"")
				unit.Qarea2.SetValid("\"-\"")
				unit.Hfact2.SetValid("\"-\"")
				unit.Qfact2.SetValid("\"-\"")
				unit.SplashTargs2.SetValid("\"_\"")
				unit.TargCount2.SetValid("1")
				unit.DamageLoss2.SetValid("0")
				unit.SpillDist2.SetValid("0")
				unit.SpillRadius2.SetValid("0")
			} else if newUnit.AttackType == "2" { // Ranged
				unit.Missileart.SetValid("Abilities\\Weapons\\GuardTowerMissile\\GuardTowerMissile.mdl")
				unit.Missileart1.SetValid("Abilities\\Weapons\\GuardTowerMissile\\GuardTowerMissile.mdl")
				unit.Missilearc.SetValid("0.15")
				unit.Missilearc1.SetValid("0.15")
				unit.Missilespeed.SetValid("1800")
				unit.Missilespeed1.SetValid("1800")

				unit.WeapsOn.SetValid("1")
				unit.Acquire.SetValid("700")
				unit.MinRange.SetValid("\"-\"")
				unit.Castpt.SetValid("0.3")
				unit.Castbsw.SetValid("0.51")
				unit.LaunchX.SetValid("0")
				unit.LaunchY.SetValid("0")
				unit.LaunchZ.SetValid("145")
				unit.LaunchSwimZ.SetValid("0")
				unit.ImpactZ.SetValid("120")
				unit.ImpactSwimZ.SetValid("0")
				unit.WeapType1.SetValid("\"_\"")
				unit.Targs1.SetValid("\"ground,structure,debris,air,item,ward\"")
				unit.ShowUI1.SetValid("1")
				unit.RangeN1.SetValid("700")
				unit.RngTst.SetValid("\"-\"")
				unit.RngBuff1.SetValid("250")
				unit.AtkType1.SetValid("\"pierce\"")
				unit.WeapTp1.SetValid("\"missile\"")
				unit.Cool1.SetValid("0.9")
				unit.Mincool1.SetValid("\"-\"")
				unit.Dice1.SetValid("1")
				unit.Sides1.SetValid("5")
				unit.Dmgplus1.SetValid("22")
				unit.DmgUp1.SetValid("\"-\"")
				unit.Mindmg1.SetValid("23")
				unit.Avgdmg1.SetValid("25")
				unit.Maxdmg1.SetValid("27")
				unit.Dmgpt1.SetValid("0.3")
				unit.BackSw1.SetValid("0.3")
				unit.Farea1.SetValid("\"-\"")
				unit.Harea1.SetValid("\"-\"")
				unit.Qarea1.SetValid("\"-\"")
				unit.Hfact1.SetValid("\"-\"")
				unit.Qfact1.SetValid("\"-\"")
				unit.SplashTargs1.SetValid("\"_\"")
				unit.TargCount1.SetValid("1")
				unit.DamageLoss1.SetValid("0")
				unit.SpillDist1.SetValid("0")
				unit.SpillRadius1.SetValid("0")
				unit.DmgUpg.SetValid("\"-\"")
				unit.Dmod1.SetValid("\"-\"")
				unit.DPS.SetValid("27.7777777777778")
				unit.WeapType2.SetValid("\"_\"")
				unit.Targs2.SetValid("\"_\"")
				unit.ShowUI2.SetValid("1")
				unit.RangeN2.SetValid("\"-\"")
				unit.RngTst2.SetValid("\"-\"")
				unit.RngBuff2.SetValid("\"-\"")
				unit.AtkType2.SetValid("\"normal\"")
				unit.WeapTp2.SetValid("\"_\"")
				unit.Cool2.SetValid("\"-\"")
				unit.Mincool2.SetValid("\"-\"")
				unit.Dice2.SetValid("\"-\"")
				unit.Sides2.SetValid("\"-\"")
				unit.Dmgplus2.SetValid("\"-\"")
				unit.DmgUp2.SetValid("\"-\"")
				unit.Mindmg2.SetValid("\"-\"")
				unit.Avgdmg2.SetValid("\"-\"")
				unit.Maxdmg2.SetValid("\"-\"")
				unit.Dmgpt2.SetValid("\"-\"")
				unit.BackSw2.SetValid("\"-\"")
				unit.Farea2.SetValid("\"-\"")
				unit.Harea2.SetValid("\"-\"")
				unit.Qarea2.SetValid("\"-\"")
				unit.Hfact2.SetValid("\"-\"")
				unit.Qfact2.SetValid("\"-\"")
				unit.SplashTargs2.SetValid("\"_\"")
				unit.TargCount2.SetValid("1")
				unit.DamageLoss2.SetValid("0")
				unit.SpillDist2.SetValid("0")
				unit.SpillRadius2.SetValid("0")
			} else if newUnit.AttackType == "3" { // Ranged (Splash)
				unit.Missileart.SetValid("Abilities\\Weapons\\CannonTowerMissile\\CannonTowerMissile.mdl")
				unit.Missileart1.SetValid("Abilities\\Weapons\\CannonTowerMissile\\CannonTowerMissile.mdl")
				unit.Missilearc.SetValid("0.35")
				unit.Missilearc1.SetValid("0.35")
				unit.Missilespeed.SetValid("700")
				unit.Missilespeed1.SetValid("700")

				unit.WeapsOn.SetValid("3")
				unit.Acquire.SetValid("800")
				unit.MinRange.SetValid("\"-\"")
				unit.Castpt.SetValid("\"-\"")
				unit.Castbsw.SetValid("0.51")
				unit.LaunchX.SetValid("0")
				unit.LaunchY.SetValid("0")
				unit.LaunchZ.SetValid("160")
				unit.LaunchSwimZ.SetValid("0")
				unit.ImpactZ.SetValid("120")
				unit.ImpactSwimZ.SetValid("0")
				unit.WeapType1.SetValid("\"_\"")
				unit.Targs1.SetValid("\"ground,debris,tree,wall,ward,item\"")
				unit.ShowUI1.SetValid("1")
				unit.RangeN1.SetValid("800")
				unit.RngTst.SetValid("\"-\"")
				unit.RngBuff1.SetValid("250")
				unit.AtkType1.SetValid("\"siege\"")
				unit.WeapTp1.SetValid("\"msplash\"")
				unit.Cool1.SetValid("2.5")
				unit.Mincool1.SetValid("\"-\"")
				unit.Dice1.SetValid("1")
				unit.Sides1.SetValid("22")
				unit.Dmgplus1.SetValid("89")
				unit.DmgUp1.SetValid("\"-\"")
				unit.Mindmg1.SetValid("90")
				unit.Avgdmg1.SetValid("100.5")
				unit.Maxdmg1.SetValid("111")
				unit.Dmgpt1.SetValid("0.3")
				unit.BackSw1.SetValid("0.3")
				unit.Farea1.SetValid("50")
				unit.Harea1.SetValid("100")
				unit.Qarea1.SetValid("125")
				unit.Hfact1.SetValid("0.5")
				unit.Qfact1.SetValid("0.1")
				unit.SplashTargs1.SetValid("ground,structure,debris,tree,wall,notself")
				unit.TargCount1.SetValid("1")
				unit.DamageLoss1.SetValid("0")
				unit.SpillDist1.SetValid("0")
				unit.SpillRadius1.SetValid("0")
				unit.DmgUpg.SetValid("\"-\"")
				unit.Dmod1.SetValid("84")
				unit.DPS.SetValid("40.2")
				unit.WeapType2.SetValid("\"_\"")
				unit.Targs2.SetValid("\"_\"")
				unit.ShowUI2.SetValid("1")
				unit.RangeN2.SetValid("\"-\"")
				unit.RngTst2.SetValid("\"-\"")
				unit.RngBuff2.SetValid("\"-\"")
				unit.AtkType2.SetValid("\"normal\"")
				unit.WeapTp2.SetValid("\"_\"")
				unit.Cool2.SetValid("\"-\"")
				unit.Mincool2.SetValid("\"-\"")
				unit.Dice2.SetValid("\"-\"")
				unit.Sides2.SetValid("\"-\"")
				unit.Dmgplus2.SetValid("\"-\"")
				unit.DmgUp2.SetValid("\"-\"")
				unit.Mindmg2.SetValid("\"-\"")
				unit.Avgdmg2.SetValid("\"-\"")
				unit.Maxdmg2.SetValid("\"-\"")
				unit.Dmgpt2.SetValid("\"-\"")
				unit.BackSw2.SetValid("\"-\"")
				unit.Farea2.SetValid("\"-\"")
				unit.Harea2.SetValid("\"-\"")
				unit.Qarea2.SetValid("\"-\"")
				unit.Hfact2.SetValid("\"-\"")
				unit.Qfact2.SetValid("\"-\"")
				unit.SplashTargs2.SetValid("\"_\"")
				unit.TargCount2.SetValid("1")
				unit.DamageLoss2.SetValid("0")
				unit.SpillDist2.SetValid("0")
				unit.SpillRadius2.SetValid("0")
			}

			unitMap[unitId] = unit
			payload = unit
		}
	case "createNewItem":
		if m.Payload != nil {
			var newItem NewItem
			if err = json.Unmarshal(m.Payload, &newItem); err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			item := new(models.SLKItem)
			item.ItemData = new(models.ItemData)
			item.ItemFunc = new(models.ItemFunc)
			item.ItemString = new(models.ItemString)

			var itemId string
			if newItem.GenerateId == true || !newItem.ItemId.Valid {
				itemId = getNextValidItemId(lastValidItemIndex)
			} else {
				itemId = newItem.ItemId.String
			}

			item.ItemID.SetValid(itemId)
			item.ItemFuncId.SetValid(itemId)
			item.ItemStringId.SetValid(itemId)
			item.Name.SetValid(newItem.Name)

			item.AbilList.SetValid("\"Aret\"")
			item.Buttonpos.SetValid("0,0")
			item.ButtonposX.SetValid("0")
			item.ButtonposY.SetValid("0")
			item.Art.SetValid("ReplaceableTextures\\CommandButtons\\BTNTomeOfRetraining.blp")
			item.File.SetValid("\"Objects\\InventoryItems\\TreasureChest\\treasurechest.mdl\"")
			item.Scale.SetValid("1")
			item.SelSize.SetValid("0")
			item.ColorR.SetValid("255")
			item.ColorG.SetValid("255")
			item.ColorB.SetValid("255")
			item.Armor.SetValid("\"Wood\"")
			item.Uses.SetValid("1")
			item.Droppable.SetValid("1")
			item.Sellable.SetValid("1")
			item.Pawnable.SetValid("1")
			item.Class.SetValid("\"Purchasable\"")
			item.CooldownID.SetValid("\"Aret\"")
			item.Drop.SetValid("0")
			item.Goldcost.SetValid("300")
			item.HP.SetValid("75")
			item.IgnoreCD.SetValid("0")
			item.PickRandom.SetValid("0")
			item.Level.SetValid("3")
			item.OldLevel.SetValid("0")
			item.Uses.SetValid("1")
			item.Perishable.SetValid("1")
			item.Prio.SetValid("0")
			item.StockMax.SetValid("1")
			item.StockRegen.SetValid("440")
			item.Morph.SetValid("0")
			item.Description.SetValid("\"Unlearns a Hero's skills.\"")
			item.Hotkey.SetValid("\"O\"")
			item.Tip.SetValid("\"Purchase T|cffffcc00o|rme of Retraining\"")
			item.Ubertip.SetValid("\"Unlearns all of the Hero's spells, allowing the Hero to learn different skills.\"")

			itemMap[itemId] = item
			payload = item
		}
	case "createNewAbility":
		if m.Payload != nil {
			var newAbility NewAbility
			if err = json.Unmarshal(m.Payload, &newAbility); err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			ability := new(models.SLKAbility)
			ability.AbilityData = new(models.AbilityData)
			ability.AbilityFunc = new(models.AbilityFunc)
			ability.AbilityString = new(models.AbilityString)

			var alias string
			if newAbility.GenerateId == true || !newAbility.Alias.Valid {
				alias = getNextValidAbilityId(lastValidAbilityIndex)
			} else {
				alias = newAbility.Alias.String
			}

			ability.Alias.SetValid(alias)
			ability.AbilityFuncId.SetValid(alias)
			ability.AbilityStringId.SetValid(alias)
			ability.Name.SetValid(newAbility.Name)

			// TODO: Implement functionality to create new abilities here!

			abilityMap[alias] = ability
			payload = ability
		}
	case "loadMdx":
		if len(m.Payload) > 0 {
			folders := configDirs.QueryFolders(configdir.Global)
			if len(folders) < 1 {
				err = fmt.Errorf("failed to load config directory")
				log.Println(err)
				payload = err.Error()
				return
			}

			path := folders[0].Path

			err = startDownload(w, path)
			if err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			var unitModelList Models
			unitWalkPath := path + string(filepath.Separator) + "resources" + string(filepath.Separator) + "wc3-slk-edit-electron-resources-master" + string(filepath.Separator) + "units"
			var flag = false
			if flag, err = exists(unitWalkPath); err != nil || flag {
				err = filepath.Walk(unitWalkPath, func(currentPath string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					if !info.IsDir() {
						index := strings.LastIndex(info.Name(), ".")
						if index > -1 {
							if info.Name()[index:] == ".mdx" {
								unitModelList = append(unitModelList, Model{info.Name()[:index], "units" + currentPath[len(unitWalkPath):]})
							}
						}
					}
					return err
				})

				if err != nil {
					log.Println(err)
					payload = err.Error()
					return
				}
			}

			buildingWalkPath := path + string(filepath.Separator) + "resources" + string(filepath.Separator) + "wc3-slk-edit-electron-resources-master" + string(filepath.Separator) + "buildings"
			flag = false
			if flag, err = exists(buildingWalkPath); err != nil || flag {
				err = filepath.Walk(buildingWalkPath, func(currentPath string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					if !info.IsDir() {
						index := strings.LastIndex(info.Name(), ".")
						if index > -1 {
							if info.Name()[index:] == ".mdx" {
								unitModelList = append(unitModelList, Model{info.Name()[:index], "buildings" + currentPath[len(buildingWalkPath):]})
							}
						}
					}
					return err
				})

				if err != nil {
					log.Println(err)
					payload = err.Error()
					return
				}
			}

			var abilityModelList Models
			abilityWalkPath := path + string(filepath.Separator) + "resources" + string(filepath.Separator) + "wc3-slk-edit-electron-resources-master" + string(filepath.Separator) + "abilities" + string(filepath.Separator) + "spells"
			flag = false
			if flag, err = exists(abilityWalkPath); err != nil || flag {
				err = filepath.Walk(abilityWalkPath, func(currentPath string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					if !info.IsDir() {
						index := strings.LastIndex(info.Name(), ".")
						if index > -1 {
							if info.Name()[index:] == ".mdx" {
								abilityModelList = append(abilityModelList, Model{info.Name()[:index], "abilities" + string(filepath.Separator) + "spells" + currentPath[len(abilityWalkPath):]})
							}
						}
					}
					return err
				})

				if err != nil {
					log.Println(err)
					payload = err.Error()
					return
				}
			}

			var missileModelList Models
			missileWalkPath := path + string(filepath.Separator) + "resources" + string(filepath.Separator) + "wc3-slk-edit-electron-resources-master" + string(filepath.Separator) + "abilities" + string(filepath.Separator) + "weapons"
			flag = false
			if flag, err = exists(missileWalkPath); err != nil || flag {
				err = filepath.Walk(missileWalkPath, func(currentPath string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					if !info.IsDir() {
						index := strings.LastIndex(info.Name(), ".")
						if index > -1 {
							if info.Name()[index:] == ".mdx" {
								missileModelList = append(missileModelList, Model{info.Name()[:index], "abilities" + string(filepath.Separator) + "weapons" + currentPath[len(missileWalkPath):]})
							}
						}
					}
					return err
				})

				if err != nil {
					log.Println(err)
					payload = err.Error()
					return
				}
			}

			var itemModelList Models
			itemWalkPath := path + string(filepath.Separator) + "resources" + string(filepath.Separator) + "wc3-slk-edit-electron-resources-master" + string(filepath.Separator) + "objects" + string(filepath.Separator) + "inventoryitems"
			flag = false
			if flag, err = exists(itemWalkPath); err != nil || flag {
				err = filepath.Walk(itemWalkPath, func(currentPath string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					if !info.IsDir() {
						index := strings.LastIndex(info.Name(), ".")
						if index > -1 {
							if info.Name()[index:] == ".mdx" {
								itemModelList = append(itemModelList, Model{info.Name()[:index], "objects" + string(filepath.Separator) + "inventoryitems" + currentPath[len(itemWalkPath):]})
							}
						}
					}
					return err
				})

				if err != nil {
					log.Println(err)
					payload = err.Error()
					return
				}
			}

			var groupedModelList = new(GroupedModels)
			if unitModelList != nil {
				sort.Sort(unitModelList)
				groupedModelList.Units = unitModelList
			} else {
				groupedModelList.Units = []Model{}
			}

			if abilityModelList != nil {
				sort.Sort(abilityModelList)
				groupedModelList.Abilities = abilityModelList
			} else {
				groupedModelList.Abilities = []Model{}
			}

			if missileModelList != nil {
				sort.Sort(missileModelList)
				groupedModelList.Missiles = missileModelList
			} else {
				groupedModelList.Missiles = []Model{}
			}

			if itemModelList != nil {
				sort.Sort(itemModelList)
				groupedModelList.Items = itemModelList
			} else {
				groupedModelList.Items = []Model{}
			}

			payload = groupedModelList
		}
	case "setRegexSearch":
		var isRegexSearch bool
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &isRegexSearch); err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			if isRegexSearch != configuration.IsRegexSearch {
				configuration.IsRegexSearch = isRegexSearch

				err = saveConfig()
				if err != nil {
					log.Println(err)
					payload = err.Error()
					return
				}
			}

			payload = configuration.IsRegexSearch
		}
	case "getOperatingSystem":
		payload = runtime.GOOS
	case "hideWindow":
		err = w.Hide()
		if err != nil {
			panic(err)
		}
	case "closeWindow":
		err = w.Close()
		if err != nil {
			panic(err)
		}
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

func loadConfigFile(fileName string) *configdir.Config {
	return configDirs.QueryFolderContainsFile(fileName)
}

func saveConfig() error {
	confingInBytes, err := json.MarshalIndent(configuration, "", "  ")
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

func getNextValidAbilityId(offset int) string {
	var str string

	if offset > 16383 {
		log.Println("Ran out of valid generated ability id's")
		return ""
	}

	var firstChar string
	firstChar = "A"

	str = firstChar + intToHex(offset/256) + intToHex(int(offset/16)%16) + intToHex(offset%16)
	if _, ok := abilityMap[str]; !ok {
		lastValidAbilityIndex = offset
		return str
	}

	return getNextValidAbilityId(offset + 1)
}

func getNextValidItemId(offset int) string {
	var str string

	if offset > 16383 {
		log.Println("Ran out of valid generated item id's")
		return ""
	}

	var firstChar string
	firstChar = "I"

	str = firstChar + intToHex(offset/256) + intToHex(int(offset/16)%16) + intToHex(offset%16)
	if _, ok := itemMap[str]; !ok {
		lastValidItemIndex = offset
		return str
	}

	return getNextValidItemId(offset + 1)
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
	if _, ok := unitMap[str]; !ok {
		lastValidItemIndex = offset
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

func saveUnitsToFile(location string) {
	unitList := make([]*models.SLKUnit, len(unitMap))

	i := 0
	for _, v := range unitMap {
		unitList[i] = v
		i++
	}

	abilityList := make([]*models.SLKAbility, len(abilityMap))
	i = 0
	for _, v := range abilityMap {
		abilityList[i] = v
		i++
	}

	parser.WriteToFilesAndSaveToFolder(unitList, []*models.SLKItem{}, abilityList, location, true)
}

func loadData() error {
	var err error

	folders := configDirs.QueryFolders(configdir.Global)
	if len(folders) < 1 {
		err = fmt.Errorf("failed to load config directory")

		log.Println(err)
		return err
	}

	var abilityMetaDataPath *string = nil
	var abilityDataPath *string = nil
	var campaignAbilityFuncPath *string = nil
	var campaignAbilityStringsPath *string = nil
	var commonAbilityFuncPath *string = nil
	var commonAbilityStringsPath *string = nil
	var humanAbilityFuncPath *string = nil
	var humanAbilityStringsPath *string = nil
	var itemAbilityFuncPath *string = nil
	var itemAbilityStringsPath *string = nil
	var neutralAbilityFuncPath *string = nil
	var neutralAbilityStringsPath *string = nil
	var nightElfAbilityFuncPath *string = nil
	var nightElfAbilityStringsPath *string = nil
	var orcAbilityFuncPath *string = nil
	var orcAbilityStringsPath *string = nil
	var undeadAbilityFuncPath *string = nil
	var undeadAbilityStringsPath *string = nil

	inputDirectory := folders[0].Path + string(filepath.Separator) + "resources" + string(filepath.Separator) + "wc3-slk-edit-electron-resources-master" + string(filepath.Separator) + "data"

	var filesInDirectory []os.FileInfo
	filesInDirectory, err = ioutil.ReadDir(inputDirectory)
	if err != nil {
		log.Fatal(err)
		return err
	}

	for _, file := range filesInDirectory {
		lowercaseFilename := strings.ToLower(file.Name())
		path := filepath.Join(inputDirectory, file.Name())

		switch lowercaseFilename {
		case "abilitymetadata.slk":
			abilityMetaDataPath = &path
		case "abilitydata.slk":
			abilityDataPath = &path
		case "campaignabilityfunc.txt":
			campaignAbilityFuncPath = &path
		case "campaignabilitystrings.txt":
			campaignAbilityStringsPath = &path
		case "commonabilityfunc.txt":
			commonAbilityFuncPath = &path
		case "commonabilitystrings.txt":
			commonAbilityStringsPath = &path
		case "humanabilityfunc.txt":
			humanAbilityFuncPath = &path
		case "humanabilitystrings.txt":
			humanAbilityStringsPath = &path
		case "neutralabilityfunc.txt":
			neutralAbilityFuncPath = &path
		case "neutralabilitystrings.txt":
			neutralAbilityStringsPath = &path
		case "nightelfabilityfunc.txt":
			nightElfAbilityFuncPath = &path
		case "nightelfabilitystrings.txt":
			nightElfAbilityStringsPath = &path
		case "orcabilityfunc.txt":
			orcAbilityFuncPath = &path
		case "orcabilitystrings.txt":
			orcAbilityStringsPath = &path
		case "undeadabilityfunc.txt":
			undeadAbilityFuncPath = &path
		case "undeadabilitystrings.txt":
			undeadAbilityStringsPath = &path
		case "itemabilityfunc.txt":
			itemAbilityFuncPath = &path
		case "itemabilitystrings.txt":
			itemAbilityStringsPath = &path
		default:
			log.Printf("%v is an unknown file and will be ignored!", lowercaseFilename)
		}
	}

	var abilityMetaDataBytes []byte = nil
	var abilityDataBytes []byte = nil
	var itemAbilityFuncBytes []byte = nil
	var itemAbilityStringsBytes []byte = nil
	var campaignAbilityFuncBytes []byte = nil
	var campaignAbilityStringsBytes []byte = nil
	var commonAbilityFuncBytes []byte = nil
	var commonAbilityStringsBytes []byte = nil
	var humanAbilityFuncBytes []byte = nil
	var humanAbilityStringsBytes []byte = nil
	var neutralAbilityFuncBytes []byte = nil
	var neutralAbilityStringsBytes []byte = nil
	var nightElfAbilityFuncBytes []byte = nil
	var nightElfAbilityStringsBytes []byte = nil
	var orcAbilityFuncBytes []byte = nil
	var orcAbilityStringsBytes []byte = nil
	var undeadAbilityFuncBytes []byte = nil
	var undeadAbilityStringsBytes []byte = nil
	var readFileWaitGroup sync.WaitGroup

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if abilityMetaDataPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*abilityMetaDataPath); err != nil || flag {
				log.Println("Reading AbilityMetaData.slk...")

				abilityMetaDataBytes, err = ioutil.ReadFile(*abilityMetaDataPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if abilityDataPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*abilityDataPath); err != nil || flag {
				log.Println("Reading AbilityData.slk...")

				abilityDataBytes, err = ioutil.ReadFile(*abilityDataPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if itemAbilityFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*itemAbilityFuncPath); err != nil || flag {
				log.Println("Reading CampaignAbilityFunc.txt...")

				itemAbilityFuncBytes, err = ioutil.ReadFile(*itemAbilityFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if itemAbilityStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*itemAbilityStringsPath); err != nil || flag {
				log.Println("Reading CampaignAbilityStrings.txt...")

				itemAbilityStringsBytes, err = ioutil.ReadFile(*itemAbilityStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if campaignAbilityFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*campaignAbilityFuncPath); err != nil || flag {
				log.Println("Reading ItemAbilityFunc.txt...")

				campaignAbilityFuncBytes, err = ioutil.ReadFile(*campaignAbilityFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if campaignAbilityStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*campaignAbilityStringsPath); err != nil || flag {
				log.Println("Reading ItemAbilityStrings.txt...")

				campaignAbilityStringsBytes, err = ioutil.ReadFile(*campaignAbilityStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if commonAbilityFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*commonAbilityFuncPath); err != nil || flag {
				log.Println("Reading CommonAbilityFunc.txt...")

				commonAbilityFuncBytes, err = ioutil.ReadFile(*commonAbilityFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if commonAbilityStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*commonAbilityStringsPath); err != nil || flag {
				log.Println("Reading CommonAbilityStrings.txt...")

				commonAbilityStringsBytes, err = ioutil.ReadFile(*commonAbilityStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if humanAbilityFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*humanAbilityFuncPath); err != nil || flag {
				log.Println("Reading HumanAbilityFunc.txt...")

				humanAbilityFuncBytes, err = ioutil.ReadFile(*humanAbilityFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if humanAbilityStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*humanAbilityStringsPath); err != nil || flag {
				log.Println("Reading HumanAbilityStrings.txt...")

				humanAbilityStringsBytes, err = ioutil.ReadFile(*humanAbilityStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if neutralAbilityFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*neutralAbilityFuncPath); err != nil || flag {
				log.Println("Reading NeutralAbilityFunc.txt...")

				neutralAbilityFuncBytes, err = ioutil.ReadFile(*neutralAbilityFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if neutralAbilityStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*neutralAbilityStringsPath); err != nil || flag {
				log.Println("Reading NeutralAbilityStrings.txt...")

				neutralAbilityStringsBytes, err = ioutil.ReadFile(*neutralAbilityStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if nightElfAbilityFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*nightElfAbilityFuncPath); err != nil || flag {
				log.Println("Reading NightElfAbilityFunc.txt...")

				nightElfAbilityFuncBytes, err = ioutil.ReadFile(*nightElfAbilityFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if nightElfAbilityStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*nightElfAbilityStringsPath); err != nil || flag {
				log.Println("Reading NightElfAbilityStrings.txt...")

				nightElfAbilityStringsBytes, err = ioutil.ReadFile(*nightElfAbilityStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if orcAbilityFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*orcAbilityFuncPath); err != nil || flag {
				log.Println("Reading OrcAbilityFunc.txt...")

				orcAbilityFuncBytes, err = ioutil.ReadFile(*orcAbilityFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if orcAbilityStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*orcAbilityStringsPath); err != nil || flag {
				log.Println("Reading OrcAbilityStrings.txt...")

				orcAbilityStringsBytes, err = ioutil.ReadFile(*orcAbilityStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if undeadAbilityFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*undeadAbilityFuncPath); err != nil || flag {
				log.Println("Reading UndeadAbilityFunc.txt...")

				undeadAbilityFuncBytes, err = ioutil.ReadFile(*undeadAbilityFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if undeadAbilityStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*undeadAbilityStringsPath); err != nil || flag {
				log.Println("Reading UndeadAbilityStrings.txt...")

				undeadAbilityStringsBytes, err = ioutil.ReadFile(*undeadAbilityStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Wait()

	abilityMetaDataMap = make(map[string]*models.AbilityMetaData)
	if abilityMetaDataBytes != nil {
		log.Println("Parsing abilityMetaDataBytes...")
		parser.PopulateAbilityMetaDataMapWithSlkFileData(abilityMetaDataBytes, abilityMetaDataMap)
	}

	baseAbilityMap = make(map[string]*models.SLKAbility)
	if abilityDataBytes != nil {
		log.Println("Parsing abilityDataBytes...")
		parser.PopulateAbilityMapWithSlkFileData(abilityDataBytes, baseAbilityMap)
	}

	if campaignAbilityFuncBytes != nil {
		log.Println("Parsing campaignAbilityFuncBytes...")
		parser.PopulateAbilityMapWithTxtFileData(campaignAbilityFuncBytes, baseAbilityMap)
	}

	if campaignAbilityStringsBytes != nil {
		log.Println("Parsing campaignAbilityStringsBytes...")
		parser.PopulateAbilityMapWithTxtFileData(campaignAbilityStringsBytes, baseAbilityMap)
	}

	if commonAbilityFuncBytes != nil {
		log.Println("Parsing commonAbilityFuncBytes...")
		parser.PopulateAbilityMapWithTxtFileData(commonAbilityFuncBytes, baseAbilityMap)
	}

	if commonAbilityStringsBytes != nil {
		log.Println("Parsing commonAbilityStringsBytes...")
		parser.PopulateAbilityMapWithTxtFileData(commonAbilityStringsBytes, baseAbilityMap)
	}

	if humanAbilityFuncBytes != nil {
		log.Println("Parsing humanAbilityFuncBytes...")
		parser.PopulateAbilityMapWithTxtFileData(humanAbilityFuncBytes, baseAbilityMap)
	}

	if humanAbilityStringsBytes != nil {
		log.Println("Parsing humanAbilityStringsBytes...")
		parser.PopulateAbilityMapWithTxtFileData(humanAbilityStringsBytes, baseAbilityMap)
	}

	if neutralAbilityFuncBytes != nil {
		log.Println("Parsing neutralAbilityFuncBytes...")
		parser.PopulateAbilityMapWithTxtFileData(neutralAbilityFuncBytes, baseAbilityMap)
	}

	if neutralAbilityStringsBytes != nil {
		log.Println("Parsing neutralAbilityStringsBytes...")
		parser.PopulateAbilityMapWithTxtFileData(neutralAbilityStringsBytes, baseAbilityMap)
	}

	if nightElfAbilityFuncBytes != nil {
		log.Println("Parsing nightElfAbilityFuncBytes...")
		parser.PopulateAbilityMapWithTxtFileData(nightElfAbilityFuncBytes, baseAbilityMap)
	}

	if nightElfAbilityStringsBytes != nil {
		log.Println("Parsing nightElfAbilityStringsBytes...")
		parser.PopulateAbilityMapWithTxtFileData(nightElfAbilityStringsBytes, baseAbilityMap)
	}

	if orcAbilityFuncBytes != nil {
		log.Println("Parsing orcAbilityFuncBytes...")
		parser.PopulateAbilityMapWithTxtFileData(orcAbilityFuncBytes, baseAbilityMap)
	}

	if orcAbilityStringsBytes != nil {
		log.Println("Parsing orcAbilityStringsBytes...")
		parser.PopulateAbilityMapWithTxtFileData(orcAbilityStringsBytes, baseAbilityMap)
	}

	if undeadAbilityFuncBytes != nil {
		log.Println("Parsing undeadAbilityFuncBytes...")
		parser.PopulateAbilityMapWithTxtFileData(undeadAbilityFuncBytes, baseAbilityMap)
	}

	if undeadAbilityStringsBytes != nil {
		log.Println("Parsing undeadAbilityStringsBytes...")
		parser.PopulateAbilityMapWithTxtFileData(undeadAbilityStringsBytes, baseAbilityMap)
	}

	if itemAbilityFuncBytes != nil {
		log.Println("Parsing itemAbilityFuncBytes...")
		parser.PopulateAbilityMapWithTxtFileData(itemAbilityFuncBytes, baseAbilityMap)
	}

	if itemAbilityStringsBytes != nil {
		log.Println("Parsing itemAbilityStringsBytes(2)...")
		parser.PopulateAbilityMapWithTxtFileData(itemAbilityStringsBytes, baseAbilityMap)
	}

	return nil
}

func loadSLK() []*FileInfo {
	var inputDirectory string
	var abilityDataFileInfo = &FileInfo{"AbilityData.slk", "color-secondary", "ga-genderless"}
	var campaignAbilityFuncFileInfo = &FileInfo{"CampaignAbilityFunc.txt", "color-secondary", "fa-genderless"}
	var campaignAbilityStringsFileInfo = &FileInfo{"CampaignAbilityStrings.txt", "color-secondary", "fa-genderless"}
	var campaignUnitFuncFileInfo = &FileInfo{"CampaignUnitFunc.txt", "color-secondary", "fa-genderless"}
	var campaignUnitStringsFileInfo = &FileInfo{"CampaignUnitStrings.txt", "color-secondary", "fa-genderless"}
	var commonAbilityFuncFileInfo = &FileInfo{"CommonAbilityFunc.txt", "color-secondary", "fa-genderless"}
	var commonAbilityStringsFileInfo = &FileInfo{"CommonAbilityStrings.txt", "color-secondary", "fa-genderless"}
	var humanAbilityFuncFileInfo = &FileInfo{"HumanAbilityFunc.txt", "color-secondary", "fa-genderless"}
	var humanAbilityStringsFileInfo = &FileInfo{"HumanAbilityStrings.txt", "color-secondary", "fa-genderless"}
	var humanUnitFuncFileInfo = &FileInfo{"HumanUnitFunc.txt", "color-secondary", "fa-genderless"}
	var humanUnitStringsFileInfo = &FileInfo{"HumanUnitStrings.txt", "color-secondary", "fa-genderless"}
	var neutralAbilityFuncFileInfo = &FileInfo{"NeutralAbilityFunc.txt", "color-secondary", "fa-genderless"}
	var neutralAbilityStringsFileInfo = &FileInfo{"NeutralAbilityStrings.txt", "color-secondary", "fa-genderless"}
	var neutralUnitFuncFileInfo = &FileInfo{"NeutralUnitFunc.txt", "color-secondary", "fa-genderless"}
	var neutralUnitStringsFileInfo = &FileInfo{"NeutralUnitStrings.txt", "color-secondary", "fa-genderless"}
	var nightElfAbilityFuncFileInfo = &FileInfo{"NightElfAbilityFunc.txt", "color-secondary", "fa-genderless"}
	var nightElfAbilityStringsFileInfo = &FileInfo{"NightElfAbilityStrings.txt", "color-secondary", "fa-genderless"}
	var nightElfUnitFuncFileInfo = &FileInfo{"NightElfUnitFunc.txt", "color-secondary", "fa-genderless"}
	var nightElfUnitStringsFileInfo = &FileInfo{"NightElfUnitStrings.txt", "color-secondary", "fa-genderless"}
	var orcAbilityFuncFileInfo = &FileInfo{"OrcAbilityFunc.txt", "color-secondary", "fa-genderless"}
	var orcAbilityStringsFileInfo = &FileInfo{"OrcAbilityStrings.txt", "color-secondary", "fa-genderless"}
	var orcUnitFuncFileInfo = &FileInfo{"OrcUnitFunc.txt", "color-secondary", "fa-genderless"}
	var orcUnitStringsFileInfo = &FileInfo{"OrcUnitStrings.txt", "color-secondary", "fa-genderless"}
	var undeadAbilityFuncFileInfo = &FileInfo{"UndeadAbilityFunc.txt", "color-secondary", "fa-genderless"}
	var undeadAbilityStringsFileInfo = &FileInfo{"UndeadAbilityStrings.txt", "color-secondary", "fa-genderless"}
	var undeadUnitFuncFileInfo = &FileInfo{"UndeadUnitFunc.txt", "color-secondary", "fa-genderless"}
	var undeadUnitStringsFileInfo = &FileInfo{"UndeadUnitStrings.txt", "color-secondary", "fa-genderless"}
	var unitAbilitiesFileInfo = &FileInfo{"UnitAbilities.slk", "color-secondary", "fa-genderless"}
	var unitBalanceFileInfo = &FileInfo{"UnitBalance.slk", "color-secondary", "fa-genderless"}
	var unitDataFileInfo = &FileInfo{"UnitData.slk", "color-secondary", "fa-genderless"}
	var unitUiFileInfo = &FileInfo{"UnitUI.slk", "color-secondary", "fa-genderless"}
	var unitWeaponsFileInfo = &FileInfo{"UnitWeapons.slk", "color-secondary", "fa-genderless"}
	var itemAbilityFuncFileInfo = &FileInfo{"ItemAbilityFunc.txt", "color-secondary", "fa-genderless"}
	var itemAbilityStringsFileInfo = &FileInfo{"ItemAbilityStrings.txt", "color-secondary", "fa-genderless"}
	var itemDataFileInfo = &FileInfo{"ItemData.slk", "color-secondary", "fa-genderless"}
	var itemFuncFileInfo = &FileInfo{"ItemFunc.txt", "color-secondary", "fa-genderless"}
	var itemStringsFileInfo = &FileInfo{"ItemStrings.txt", "color-secondary", "fa-genderless"}
	var fileInfoList = []*FileInfo{
		campaignAbilityFuncFileInfo,
		campaignAbilityStringsFileInfo,
		campaignUnitFuncFileInfo,
		campaignUnitStringsFileInfo,
		commonAbilityFuncFileInfo,
		commonAbilityStringsFileInfo,
		humanAbilityFuncFileInfo,
		humanUnitStringsFileInfo,
		humanUnitFuncFileInfo,
		humanUnitStringsFileInfo,
		neutralAbilityFuncFileInfo,
		neutralAbilityStringsFileInfo,
		neutralUnitFuncFileInfo,
		neutralUnitStringsFileInfo,
		nightElfAbilityFuncFileInfo,
		nightElfAbilityStringsFileInfo,
		nightElfUnitFuncFileInfo,
		nightElfUnitStringsFileInfo,
		orcAbilityFuncFileInfo,
		orcAbilityStringsFileInfo,
		orcUnitFuncFileInfo,
		orcUnitStringsFileInfo,
		undeadAbilityFuncFileInfo,
		undeadAbilityStringsFileInfo,
		undeadUnitFuncFileInfo,
		undeadUnitStringsFileInfo,
		unitAbilitiesFileInfo,
		unitBalanceFileInfo,
		unitDataFileInfo,
		unitUiFileInfo,
		unitWeaponsFileInfo,
		itemAbilityFuncFileInfo,
		itemAbilityStringsFileInfo,
		itemDataFileInfo,
		itemFuncFileInfo,
		itemStringsFileInfo,
	}

	if configuration.InDir == nil {
		log.Println("Input directory has not been set!")
		return fileInfoList
	}

	inputDirectory = *configuration.InDir
	if flag, err := exists(inputDirectory); err != nil || !flag {
		log.Println(inputDirectory + " does not exist!")
		return fileInfoList
	}

	filesInDirectory, err := ioutil.ReadDir(inputDirectory)
	if err != nil {
		log.Fatal(err)
		return fileInfoList
	}

	var abilityDataPath *string = nil
	var unitAbilitiesPath *string = nil
	var unitDataPath *string = nil
	var unitUIPath *string = nil
	var unitWeaponsPath *string = nil
	var unitBalancePath *string = nil
	var campaignAbilityFuncPath *string = nil
	var campaignAbilityStringsPath *string = nil
	var campaignUnitFuncPath *string = nil
	var campaignUnitStringsPath *string = nil
	var commonAbilityFuncPath *string = nil
	var commonAbilityStringsPath *string = nil
	var humanAbilityFuncPath *string = nil
	var humanAbilityStringsPath *string = nil
	var humanUnitFuncPath *string = nil
	var humanUnitStringsPath *string = nil
	var neutralAbilityFuncPath *string = nil
	var neutralAbilityStringsPath *string = nil
	var neutralUnitFuncPath *string = nil
	var nightElfAbilityFuncPath *string = nil
	var nightElfAbilityStringsPath *string = nil
	var neutralUnitStringsPath *string = nil
	var nightElfUnitFuncPath *string = nil
	var nightElfUnitStringsPath *string = nil
	var orcAbilityFuncPath *string = nil
	var orcAbilityStringsPath *string = nil
	var orcUnitFuncPath *string = nil
	var orcUnitStringsPath *string = nil
	var undeadAbilityFuncPath *string = nil
	var undeadAbilityStringsPath *string = nil
	var undeadUnitFuncPath *string = nil
	var undeadUnitStringsPath *string = nil
	var itemAbilityFuncPath *string = nil
	var itemAbilityStringsPath *string = nil
	var itemDataPath *string = nil
	var itemFuncPath *string = nil
	var itemStringsPath *string = nil

	for _, file := range filesInDirectory {
		lowercaseFilename := strings.ToLower(file.Name())
		path := filepath.Join(inputDirectory, file.Name())

		switch lowercaseFilename {
		case "abilitydata.slk":
			abilityDataPath = &path
		case "unitabilities.slk":
			unitAbilitiesPath = &path
		case "unitdata.slk":
			unitDataPath = &path
		case "unitui.slk":
			unitUIPath = &path
		case "unitweapons.slk":
			unitWeaponsPath = &path
		case "unitbalance.slk":
			unitBalancePath = &path
		case "campaignabilityfunc.txt":
			campaignAbilityFuncPath = &path
		case "campaignabilitystrings.txt":
			campaignAbilityStringsPath = &path
		case "campaignunitfunc.txt":
			campaignUnitFuncPath = &path
		case "campaignunitstrings.txt":
			campaignUnitStringsPath = &path
		case "commonabilityfunc.txt":
			commonAbilityFuncPath = &path
		case "commonabilitystrings.txt":
			commonAbilityStringsPath = &path
		case "humanabilityfunc.txt":
			humanAbilityFuncPath = &path
		case "humanabilitystrings.txt":
			humanAbilityStringsPath = &path
		case "humanunitfunc.txt":
			humanUnitFuncPath = &path
		case "humanunitstrings.txt":
			humanUnitStringsPath = &path
		case "neutralabilityfunc.txt":
			neutralAbilityFuncPath = &path
		case "neutralabilitystrings.txt":
			neutralAbilityStringsPath = &path
		case "neutralunitfunc.txt":
			neutralUnitFuncPath = &path
		case "neutralunitstrings.txt":
			neutralUnitStringsPath = &path
		case "nightelfabilityfunc.txt":
			nightElfAbilityFuncPath = &path
		case "nightelfabilitystrings.txt":
			nightElfAbilityStringsPath = &path
		case "nightelfunitfunc.txt":
			nightElfUnitFuncPath = &path
		case "nightelfunitstrings.txt":
			nightElfUnitStringsPath = &path
		case "orcabilityfunc.txt":
			orcAbilityFuncPath = &path
		case "orcabilitystrings.txt":
			orcAbilityStringsPath = &path
		case "orcunitfunc.txt":
			orcUnitFuncPath = &path
		case "orcunitstrings.txt":
			orcUnitStringsPath = &path
		case "undeadabilityfunc.txt":
			undeadAbilityFuncPath = &path
		case "undeadabilitystrings.txt":
			undeadAbilityStringsPath = &path
		case "undeadunitfunc.txt":
			undeadUnitFuncPath = &path
		case "undeadunitstrings.txt":
			undeadUnitStringsPath = &path
		case "itemabilityfunc.txt":
			itemAbilityFuncPath = &path
		case "itemabilitystrings.txt":
			itemAbilityStringsPath = &path
		case "itemdata.slk":
			itemDataPath = &path
		case "itemfunc.txt":
			itemFuncPath = &path
		case "itemstrings.txt":
			itemStringsPath = &path
		default:
			log.Printf("%v is an unknown file and will be ignored!", file)
		}
	}

	// Unused
	/*
		abilityBuffData := filepath.Join(inputDirectory, "AbilityBuffData.slk")
		campaignUpgradeFunc := filepath.Join(inputDirectory, "CampaignUpgradeFunc.txt")
		campaignUpgradeStrings := filepath.Join(inputDirectory, "CampaignUpgradeStrings.txt")
		commandFunc := filepath.Join(inputDirectory, "CommandFunc.txt")
		commandStrings := filepath.Join(inputDirectory, "CommandStrings.txt")
		humanUpgradeFunc := filepath.Join(inputDirectory, "HumanUpgradeFunc.txt")
		humanUpgradeStrings := filepath.Join(inputDirectory, "HumanUpgradeStrings.txt")
		itemAbilityStrings := filepath.Join(inputDirectory, "ItemAbilityStrings.txt")
		neutralUpgradeFunc := filepath.Join(inputDirectory, "NeutralUpgradeFunc.txt")
		neutralUpgradeStrings := filepath.Join(inputDirectory, "NeutralUpgradeStrings.txt")
		nightElfUpgradeFunc := filepath.Join(inputDirectory, "NightElfUpgradeFunc.txt")
		nightElfUpgradeStrings := filepath.Join(inputDirectory, "NightElfUpgradeStrings.txt")
		orcUpgradeFunc := filepath.Join(inputDirectory, "OrcUpgradeFunc.txt")
		orcUpgradeStrings := filepath.Join(inputDirectory, "OrcUpgradeStrings.txt")
		undeadUpgradeFunc := filepath.Join(inputDirectory, "UndeadUpgradeFunc.txt")
		undeadUpgradeStrings := filepath.Join(inputDirectory, "UndeadUpgradeStrings.txt")
		upgradeData := filepath.Join(inputDirectory, "UpgradeData.slk")
	*/

	var abilityDataBytes []byte = nil
	var unitDataBytes []byte = nil
	var unitAbilitiesBytes []byte = nil
	var unitUIBytes []byte = nil
	var unitWeaponsBytes []byte = nil
	var unitBalanceBytes []byte = nil
	var campaignAbilityFuncBytes []byte = nil
	var campaignAbilityStringsBytes []byte = nil
	var campaignUnitFuncBytes []byte = nil
	var campaignUnitStringsBytes []byte = nil
	var commonAbilityFuncBytes []byte = nil
	var commonAbilityStringsBytes []byte = nil
	var humanAbilityFuncBytes []byte = nil
	var humanAbilityStringsBytes []byte = nil
	var humanUnitFuncBytes []byte = nil
	var humanUnitStringsBytes []byte = nil
	var neutralAbilityFuncBytes []byte = nil
	var neutralAbilityStringsBytes []byte = nil
	var neutralUnitFuncBytes []byte = nil
	var neutralUnitStringsBytes []byte = nil
	var nightElfAbilityFuncBytes []byte = nil
	var nightElfAbilityStringsBytes []byte = nil
	var nightElfUnitFuncBytes []byte = nil
	var nightElfUnitStringsBytes []byte = nil
	var orcAbilityFuncBytes []byte = nil
	var orcAbilityStringsBytes []byte = nil
	var orcUnitFuncBytes []byte = nil
	var orcUnitStringsBytes []byte = nil
	var undeadAbilityFuncBytes []byte = nil
	var undeadAbilityStringsBytes []byte = nil
	var undeadUnitFuncBytes []byte = nil
	var undeadUnitStringsBytes []byte = nil
	var itemAbilityFuncBytes []byte = nil
	var itemAbilityStringsBytes []byte = nil
	var itemDataBytes []byte = nil
	var itemFuncBytes []byte = nil
	var itemStringsBytes []byte = nil
	var readFileWaitGroup sync.WaitGroup

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if abilityDataPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*abilityDataPath); err != nil || flag {
				log.Println("Reading AbilityData.slk...")

				abilityDataBytes, err = ioutil.ReadFile(*abilityDataPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if unitDataPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*unitDataPath); err != nil || flag {
				log.Println("Reading UnitData.slk...")

				unitDataBytes, err = ioutil.ReadFile(*unitDataPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if unitAbilitiesPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*unitAbilitiesPath); err != nil || flag {
				log.Println("Reading UnitAbilities.slk...")

				unitAbilitiesBytes, err = ioutil.ReadFile(*unitAbilitiesPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if unitUIPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*unitUIPath); err != nil || flag {
				log.Println("Reading UnitUI.slk...")

				unitUIBytes, err = ioutil.ReadFile(*unitUIPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if unitWeaponsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*unitWeaponsPath); err != nil || flag {
				log.Println("Reading UnitWeapons.slk...")

				unitWeaponsBytes, err = ioutil.ReadFile(*unitWeaponsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if unitBalancePath != nil {
			var flag bool
			var err error
			if flag, err = exists(*unitBalancePath); err != nil || flag {
				log.Println("Reading UnitBalance.slk...")

				unitBalanceBytes, err = ioutil.ReadFile(*unitBalancePath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if campaignAbilityFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*campaignAbilityFuncPath); err != nil || flag {
				log.Println("Reading CampaignAbilityFunc.txt...")

				campaignAbilityFuncBytes, err = ioutil.ReadFile(*campaignAbilityFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if campaignAbilityStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*campaignAbilityStringsPath); err != nil || flag {
				log.Println("Reading CampaignAbilityStrings.txt...")

				campaignAbilityStringsBytes, err = ioutil.ReadFile(*campaignAbilityStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if campaignUnitFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*campaignUnitFuncPath); err != nil || flag {
				log.Println("Reading CampaignUnitFunc.slk...")

				campaignUnitFuncBytes, err = ioutil.ReadFile(*campaignUnitFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if campaignUnitStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*campaignUnitStringsPath); err != nil || flag {
				log.Println("Reading CampaignUnitStrings.slk...")

				campaignUnitStringsBytes, err = ioutil.ReadFile(*campaignUnitStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if commonAbilityFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*commonAbilityFuncPath); err != nil || flag {
				log.Println("Reading CommonAbilityFunc.txt...")

				commonAbilityFuncBytes, err = ioutil.ReadFile(*commonAbilityFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if commonAbilityStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*commonAbilityStringsPath); err != nil || flag {
				log.Println("Reading CommonAbilityStrings.txt...")

				commonAbilityStringsBytes, err = ioutil.ReadFile(*commonAbilityStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if humanAbilityFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*humanAbilityFuncPath); err != nil || flag {
				log.Println("Reading HumanAbilityFunc.txt...")

				humanAbilityFuncBytes, err = ioutil.ReadFile(*humanAbilityFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if humanAbilityStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*humanAbilityStringsPath); err != nil || flag {
				log.Println("Reading HumanAbilityStrings.txt...")

				humanAbilityStringsBytes, err = ioutil.ReadFile(*humanAbilityStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if humanUnitFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*humanUnitFuncPath); err != nil || flag {
				log.Println("Reading HumanUnitFunc.slk...")

				humanUnitFuncBytes, err = ioutil.ReadFile(*humanUnitFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if humanUnitStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*humanUnitStringsPath); err != nil || flag {
				log.Println("Reading HumanUnitStrings.slk...")

				humanUnitStringsBytes, err = ioutil.ReadFile(*humanUnitStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if neutralAbilityFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*neutralAbilityFuncPath); err != nil || flag {
				log.Println("Reading NeutralAbilityFunc.txt...")

				neutralAbilityFuncBytes, err = ioutil.ReadFile(*neutralAbilityFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if neutralAbilityStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*neutralAbilityStringsPath); err != nil || flag {
				log.Println("Reading NeutralAbilityStrings.txt...")

				neutralAbilityStringsBytes, err = ioutil.ReadFile(*neutralAbilityStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if neutralUnitFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*neutralUnitFuncPath); err != nil || flag {
				log.Println("Reading NeutralUnitFunc.slk...")

				neutralUnitFuncBytes, err = ioutil.ReadFile(*neutralUnitFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if neutralUnitStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*neutralUnitStringsPath); err != nil || flag {
				log.Println("Reading NeutralUnitStrings.slk...")

				neutralUnitStringsBytes, err = ioutil.ReadFile(*neutralUnitStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if nightElfAbilityFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*nightElfAbilityFuncPath); err != nil || flag {
				log.Println("Reading NightElfAbilityFunc.txt...")

				nightElfAbilityFuncBytes, err = ioutil.ReadFile(*nightElfAbilityFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if nightElfAbilityStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*nightElfAbilityStringsPath); err != nil || flag {
				log.Println("Reading NightElfAbilityStrings.txt...")

				nightElfAbilityStringsBytes, err = ioutil.ReadFile(*nightElfAbilityStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if nightElfUnitFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*nightElfUnitFuncPath); err != nil || flag {
				log.Println("Reading NightElfUnitFunc.slk...")

				nightElfUnitFuncBytes, err = ioutil.ReadFile(*nightElfUnitFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if nightElfUnitStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*nightElfUnitStringsPath); err != nil || flag {
				log.Println("Reading NightElfUnitStrings.slk...")

				nightElfUnitStringsBytes, err = ioutil.ReadFile(*nightElfUnitStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if orcAbilityFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*orcAbilityFuncPath); err != nil || flag {
				log.Println("Reading OrcAbilityFunc.txt...")

				orcAbilityFuncBytes, err = ioutil.ReadFile(*orcAbilityFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if orcAbilityStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*orcAbilityStringsPath); err != nil || flag {
				log.Println("Reading OrcAbilityStrings.txt...")

				orcAbilityStringsBytes, err = ioutil.ReadFile(*orcAbilityStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if orcUnitFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*orcUnitFuncPath); err != nil || flag {
				log.Println("Reading OrcUnitFunc.slk...")

				orcUnitFuncBytes, err = ioutil.ReadFile(*orcUnitFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if orcUnitStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*orcUnitStringsPath); err != nil || flag {
				log.Println("Reading OrcUnitStrings.slk...")

				orcUnitStringsBytes, err = ioutil.ReadFile(*orcUnitStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if undeadAbilityFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*undeadAbilityFuncPath); err != nil || flag {
				log.Println("Reading UndeadAbilityFunc.txt...")

				undeadAbilityFuncBytes, err = ioutil.ReadFile(*undeadAbilityFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if undeadAbilityStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*undeadAbilityStringsPath); err != nil || flag {
				log.Println("Reading UndeadAbilityStrings.txt...")

				undeadAbilityStringsBytes, err = ioutil.ReadFile(*undeadAbilityStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if undeadUnitFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*undeadUnitFuncPath); err != nil || flag {
				log.Println("Reading UndeadUnitFunc.slk...")

				undeadUnitFuncBytes, err = ioutil.ReadFile(*undeadUnitFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if undeadUnitStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*undeadUnitStringsPath); err != nil || flag {
				log.Println("Reading UndeadUnitStrings.slk...")

				undeadUnitStringsBytes, err = ioutil.ReadFile(*undeadUnitStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if itemAbilityFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*itemAbilityFuncPath); err != nil || flag {
				log.Println("Reading ItemAbilityFunc.txt...")

				itemAbilityFuncBytes, err = ioutil.ReadFile(*itemAbilityFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if itemAbilityStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*itemAbilityStringsPath); err != nil || flag {
				log.Println("Reading ItemAbilityStrings.txt...")

				itemAbilityStringsBytes, err = ioutil.ReadFile(*itemAbilityStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if itemDataPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*itemDataPath); err != nil || flag {
				log.Println("Reading ItemData.slk...")

				itemDataBytes, err = ioutil.ReadFile(*itemDataPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if itemFuncPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*itemFuncPath); err != nil || flag {
				log.Println("Reading ItemFunc.txt...")

				itemFuncBytes, err = ioutil.ReadFile(*itemFuncPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Add(1)
	go func() {
		defer readFileWaitGroup.Done()
		if itemStringsPath != nil {
			var flag bool
			var err error
			if flag, err = exists(*itemStringsPath); err != nil || flag {
				log.Println("Reading ItemStrings.txt...")

				itemStringsBytes, err = ioutil.ReadFile(*itemStringsPath)
				if err != nil {
					CrashWithMessage(w, err.Error())
				}
			}
		}
	}()

	readFileWaitGroup.Wait()

	abilityMap = make(map[string]*models.SLKAbility)
	if abilityDataBytes != nil {
		log.Println("Parsing abilityDataBytes...")
		abilityDataFileInfo.StatusClass = "text-success"
		abilityDataFileInfo.StatusIconClass = "fa-check"
		parser.PopulateAbilityMapWithSlkFileData(abilityDataBytes, abilityMap)
	}

	unitMap = make(map[string]*models.SLKUnit)
	if unitDataBytes != nil {
		log.Println("Parsing unitDataBytes...")
		unitDataFileInfo.StatusClass = "text-success"
		unitDataFileInfo.StatusIconClass = "fa-check"
		parser.PopulateUnitMapWithSlkFileData(unitDataBytes, unitMap)
	}

	if unitAbilitiesBytes != nil {
		log.Println("Parsing unitAbilitiesBytes...")
		unitAbilitiesFileInfo.StatusClass = "text-success"
		unitAbilitiesFileInfo.StatusIconClass = "fa-check"
		parser.PopulateUnitMapWithSlkFileData(unitAbilitiesBytes, unitMap)
	}

	if unitUIBytes != nil {
		log.Println("Parsing unitUIBytes...")
		unitUiFileInfo.StatusClass = "text-success"
		unitUiFileInfo.StatusIconClass = "fa-check"
		parser.PopulateUnitMapWithSlkFileData(unitUIBytes, unitMap)
	}

	if unitWeaponsBytes != nil {
		log.Println("Parsing unitWeaponsBytes...")
		unitWeaponsFileInfo.StatusClass = "text-success"
		unitWeaponsFileInfo.StatusIconClass = "fa-check"
		parser.PopulateUnitMapWithSlkFileData(unitWeaponsBytes, unitMap)
	}

	if unitBalanceBytes != nil {
		log.Println("Parsing unitBalanceBytes...")
		unitBalanceFileInfo.StatusClass = "text-success"
		unitBalanceFileInfo.StatusIconClass = "fa-check"
		parser.PopulateUnitMapWithSlkFileData(unitBalanceBytes, unitMap)
	}

	if campaignAbilityFuncBytes != nil {
		log.Println("Parsing campaignAbilityFuncBytes...")
		campaignAbilityFuncFileInfo.StatusClass = "text-success"
		campaignAbilityFuncFileInfo.StatusIconClass = "fa-check"
		parser.PopulateAbilityMapWithTxtFileData(campaignAbilityFuncBytes, abilityMap)
	}

	if campaignAbilityStringsBytes != nil {
		log.Println("Parsing campaignAbilityStringsBytes...")
		campaignAbilityStringsFileInfo.StatusClass = "text-success"
		campaignAbilityStringsFileInfo.StatusIconClass = "fa-check"
		parser.PopulateAbilityMapWithTxtFileData(campaignAbilityStringsBytes, abilityMap)
	}

	if campaignUnitFuncBytes != nil {
		log.Println("Parsing campaignUnitFuncBytes...")
		campaignUnitFuncFileInfo.StatusClass = "text-success"
		campaignUnitFuncFileInfo.StatusIconClass = "fa-check"
		parser.PopulateUnitMapWithTxtFileData(campaignUnitFuncBytes, unitMap)
	}

	if campaignUnitStringsBytes != nil {
		log.Println("Parsing campaignUnitStringsBytes...")
		campaignUnitStringsFileInfo.StatusClass = "text-success"
		campaignUnitStringsFileInfo.StatusIconClass = "fa-check"
		parser.PopulateUnitMapWithTxtFileData(campaignUnitStringsBytes, unitMap)
	}

	if commonAbilityFuncBytes != nil {
		log.Println("Parsing commonAbilityFuncBytes...")
		commonAbilityFuncFileInfo.StatusClass = "text-success"
		commonAbilityFuncFileInfo.StatusIconClass = "fa-check"
		parser.PopulateAbilityMapWithTxtFileData(commonAbilityFuncBytes, abilityMap)
	}

	if commonAbilityStringsBytes != nil {
		log.Println("Parsing commonAbilityStringsBytes...")
		commonAbilityStringsFileInfo.StatusClass = "text-success"
		commonAbilityStringsFileInfo.StatusIconClass = "fa-check"
		parser.PopulateAbilityMapWithTxtFileData(commonAbilityStringsBytes, abilityMap)
	}

	if humanAbilityFuncBytes != nil {
		log.Println("Parsing humanAbilityFuncBytes...")
		humanAbilityFuncFileInfo.StatusClass = "text-success"
		humanAbilityFuncFileInfo.StatusIconClass = "fa-check"
		parser.PopulateAbilityMapWithTxtFileData(humanAbilityFuncBytes, abilityMap)
	}

	if humanAbilityStringsBytes != nil {
		log.Println("Parsing humanAbilityStringsBytes...")
		humanAbilityStringsFileInfo.StatusClass = "text-success"
		humanAbilityStringsFileInfo.StatusIconClass = "fa-check"
		parser.PopulateAbilityMapWithTxtFileData(humanAbilityStringsBytes, abilityMap)
	}

	if humanUnitFuncBytes != nil {
		log.Println("Parsing humanUnitFuncBytes...")
		humanUnitFuncFileInfo.StatusClass = "text-success"
		humanUnitFuncFileInfo.StatusIconClass = "fa-check"
		parser.PopulateUnitMapWithTxtFileData(humanUnitFuncBytes, unitMap)
	}

	if humanUnitStringsBytes != nil {
		log.Println("Parsing humanUnitStringsBytes...")
		humanUnitStringsFileInfo.StatusClass = "text-success"
		humanUnitStringsFileInfo.StatusIconClass = "fa-check"
		parser.PopulateUnitMapWithTxtFileData(humanUnitStringsBytes, unitMap)
	}

	if neutralAbilityFuncBytes != nil {
		log.Println("Parsing neutralAbilityFuncBytes...")
		neutralAbilityFuncFileInfo.StatusClass = "text-success"
		neutralAbilityFuncFileInfo.StatusIconClass = "fa-check"
		parser.PopulateAbilityMapWithTxtFileData(neutralAbilityFuncBytes, abilityMap)
	}

	if neutralAbilityStringsBytes != nil {
		log.Println("Parsing neutralAbilityStringsBytes...")
		neutralAbilityStringsFileInfo.StatusClass = "text-success"
		neutralAbilityStringsFileInfo.StatusIconClass = "fa-check"
		parser.PopulateAbilityMapWithTxtFileData(neutralAbilityStringsBytes, abilityMap)
	}

	if neutralUnitFuncBytes != nil {
		log.Println("Parsing neutralUnitFuncBytes...")
		neutralUnitFuncFileInfo.StatusClass = "text-success"
		neutralUnitFuncFileInfo.StatusIconClass = "fa-check"
		parser.PopulateUnitMapWithTxtFileData(neutralUnitFuncBytes, unitMap)
	}

	if neutralUnitStringsBytes != nil {
		log.Println("Parsing neutralUnitStringsBytes...")
		neutralUnitStringsFileInfo.StatusClass = "text-success"
		neutralUnitStringsFileInfo.StatusIconClass = "fa-check"
		parser.PopulateUnitMapWithTxtFileData(neutralUnitStringsBytes, unitMap)
	}

	if nightElfAbilityFuncBytes != nil {
		log.Println("Parsing nightElfAbilityFuncBytes...")
		nightElfAbilityFuncFileInfo.StatusClass = "text-success"
		nightElfAbilityFuncFileInfo.StatusIconClass = "fa-check"
		parser.PopulateAbilityMapWithTxtFileData(nightElfAbilityFuncBytes, abilityMap)
	}

	if nightElfAbilityStringsBytes != nil {
		log.Println("Parsing nightElfAbilityStringsBytes...")
		nightElfAbilityStringsFileInfo.StatusClass = "text-success"
		nightElfAbilityStringsFileInfo.StatusIconClass = "fa-check"
		parser.PopulateAbilityMapWithTxtFileData(nightElfAbilityStringsBytes, abilityMap)
	}

	if nightElfUnitFuncBytes != nil {
		log.Println("Parsing nightElfUnitFuncBytes...")
		nightElfUnitFuncFileInfo.StatusClass = "text-success"
		nightElfUnitFuncFileInfo.StatusIconClass = "fa-check"
		parser.PopulateUnitMapWithTxtFileData(nightElfUnitFuncBytes, unitMap)
	}

	if nightElfUnitStringsBytes != nil {
		log.Println("Parsing nightElfUnitStringsBytes...")
		nightElfUnitStringsFileInfo.StatusClass = "text-success"
		nightElfUnitStringsFileInfo.StatusIconClass = "fa-check"
		parser.PopulateUnitMapWithTxtFileData(nightElfUnitStringsBytes, unitMap)
	}

	if orcAbilityFuncBytes != nil {
		log.Println("Parsing orcAbilityFuncBytes...")
		orcAbilityFuncFileInfo.StatusClass = "text-success"
		orcAbilityFuncFileInfo.StatusIconClass = "fa-check"
		parser.PopulateAbilityMapWithTxtFileData(orcAbilityFuncBytes, abilityMap)
	}

	if orcAbilityStringsBytes != nil {
		log.Println("Parsing orcAbilityStringsBytes...")
		orcAbilityStringsFileInfo.StatusClass = "text-success"
		orcAbilityStringsFileInfo.StatusIconClass = "fa-check"
		parser.PopulateAbilityMapWithTxtFileData(orcAbilityStringsBytes, abilityMap)
	}

	if orcUnitFuncBytes != nil {
		log.Println("Parsing orcUnitFuncBytes...")
		orcUnitFuncFileInfo.StatusClass = "text-success"
		orcUnitFuncFileInfo.StatusIconClass = "fa-check"
		parser.PopulateUnitMapWithTxtFileData(orcUnitFuncBytes, unitMap)
	}

	if orcUnitStringsBytes != nil {
		log.Println("Parsing orcUnitStringsBytes...")
		orcUnitStringsFileInfo.StatusClass = "text-success"
		orcUnitStringsFileInfo.StatusIconClass = "fa-check"
		parser.PopulateUnitMapWithTxtFileData(orcUnitStringsBytes, unitMap)
	}

	if undeadAbilityFuncBytes != nil {
		log.Println("Parsing undeadAbilityFuncBytes...")
		undeadAbilityFuncFileInfo.StatusClass = "text-success"
		undeadAbilityFuncFileInfo.StatusIconClass = "fa-check"
		parser.PopulateAbilityMapWithTxtFileData(undeadAbilityFuncBytes, abilityMap)
	}

	if undeadAbilityStringsBytes != nil {
		log.Println("Parsing undeadAbilityStringsBytes...")
		undeadAbilityStringsFileInfo.StatusClass = "text-success"
		undeadAbilityStringsFileInfo.StatusIconClass = "fa-check"
		parser.PopulateAbilityMapWithTxtFileData(undeadAbilityStringsBytes, abilityMap)
	}

	if undeadUnitFuncBytes != nil {
		log.Println("Parsing undeadUnitFuncBytes...")
		undeadUnitFuncFileInfo.StatusClass = "text-success"
		undeadUnitFuncFileInfo.StatusIconClass = "fa-check"
		parser.PopulateUnitMapWithTxtFileData(undeadUnitFuncBytes, unitMap)
	}

	if undeadUnitStringsBytes != nil {
		log.Println("Parsing undeadUnitStringsBytes...")
		undeadUnitStringsFileInfo.StatusClass = "text-success"
		undeadUnitStringsFileInfo.StatusIconClass = "fa-check"
		parser.PopulateUnitMapWithTxtFileData(undeadUnitStringsBytes, unitMap)
	}

	if itemAbilityFuncBytes != nil {
		log.Println("Parsing itemAbilityFuncBytes...")
		itemAbilityFuncFileInfo.StatusClass = "text-success"
		itemAbilityFuncFileInfo.StatusIconClass = "fa-check"
		parser.PopulateAbilityMapWithTxtFileData(itemAbilityFuncBytes, abilityMap)
	}

	if itemAbilityStringsBytes != nil {
		log.Println("Parsing itemAbilityStringsBytes...")
		itemAbilityStringsFileInfo.StatusClass = "text-success"
		itemAbilityStringsFileInfo.StatusIconClass = "fa-check"
		parser.PopulateAbilityMapWithTxtFileData(itemAbilityStringsBytes, abilityMap)
	}

	itemMap = make(map[string]*models.SLKItem)
	if itemDataBytes != nil {
		log.Println("Parsing itemDataBytes...")
		itemDataFileInfo.StatusClass = "text-success"
		itemDataFileInfo.StatusIconClass = "fa-check"
		parser.PopulateItemMapWithSlkFileData(itemDataBytes, itemMap)
	}

	if itemFuncBytes != nil {
		log.Println("Parsing itemFuncBytes...")
		itemFuncFileInfo.StatusClass = "text-success"
		itemFuncFileInfo.StatusIconClass = "fa-check"
		parser.PopulateItemMapWithTxtFileData(itemFuncBytes, itemMap)
	}

	if itemStringsBytes != nil {
		log.Println("Parsing itemStringsBytes...")
		itemStringsFileInfo.StatusClass = "text-success"
		itemStringsFileInfo.StatusIconClass = "fa-check"
		parser.PopulateItemMapWithTxtFileData(itemStringsBytes, itemMap)
	}

	return fileInfoList
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

func SendDownloadProgressMessage(w *astilectron.Window, done chan int64, path string, total int64) {
	stop := false

	for {
		select {
		case <-done:
			stop = true
		default:
			file, err := os.Open(path)
			if err != nil {
				log.Println(err)
			}

			fileStat, err := file.Stat()
			if err != nil {
				log.Println(err)
			}

			size := fileStat.Size()

			if size == 0 {
				size = 1
			}

			percent := float64(size) / float64(total) * 50

			w.SendMessage(EventMessage{"downloadPercentUpdate", percent})
		}

		if stop {
			break
		}

		time.Sleep(time.Second)
	}
}

func startDownload(w *astilectron.Window, path string) error {
	w.SendMessage(EventMessage{"downloadStart", nil})

	url := MODEL_DOWNLOAD_URL

	start := time.Now()

	file := path + string(filepath.Separator) + "temp.zip"

	log.Printf("Download started for %s...\n", file)

	var err error
	var headResp *http.Response
	var size int64

	out, err := os.Create(file)
	if err != nil {
		return err
	}

	headResp, err = http.Head(url)
	if err != nil {
		return err
	}

	defer headResp.Body.Close()

	eTag := headResp.Header.Get("ETag")

	if configuration.ResourceETag != nil && *(configuration.ResourceETag) == eTag {
		return nil
	}

	size = headResp.ContentLength

	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	log.Printf("size(%v)\n", size)

	// If the HEAD request didn't receive any Content-Length header we'll have to grab it from the actual request
	if size < 0 {
		size = resp.ContentLength
		log.Printf("Size failed, new size(%v)\n", size)
	}

	w.SendMessage(EventMessage{"downloadTextUpdate", "Downloading..."})

	done := make(chan int64)

	go SendDownloadProgressMessage(w, done, file, size)

	n, err := io.Copy(out, resp.Body)
	if err != nil {
		done <- 0

		return err
	}

	done <- n

	err = out.Close()
	if err != nil {
		return err
	}

	elapsed := time.Since(start)
	log.Printf("Download completed in %s\n", elapsed)

	configuration.ResourceETag = &eTag

	err = saveConfig()
	if err != nil {
		return err
	}

	w.SendMessage(EventMessage{"downloadTextUpdate", "Extracting..."})

	unzipDestination := path + string(filepath.Separator) + "resources"

	os.Mkdir(unzipDestination, os.ModePerm)

	_, err = Unzip(w, file, unzipDestination)
	if err != nil {
		return err
	}

	w.SendMessage(EventMessage{"downloadTextUpdate", "Cleaning up..."})

	err = os.Remove(file)
	if err != nil {
		return err
	}

	return nil
}

func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

func SendExtractionProgressMessage(w *astilectron.Window, done chan int64, path string, total int64) {
	stop := false

	for {
		select {
		case <-done:
			stop = true
		default:

			size, err := DirSize(path)
			if err != nil {
				log.Println(err)
			}

			if size == 0 {
				size = 1
			}

			percent := 50 + (float64(size) / float64(total) * 50)

			w.SendMessage(EventMessage{"downloadPercentUpdate", percent})
		}

		if stop {
			break
		}

		time.Sleep(time.Second)
	}
}

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file to an output directory
func Unzip(w *astilectron.Window, src string, destination string) ([]string, error) {
	var fileNames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return fileNames, err
	}

	var totalSize int64
	totalSize = 0

	for _, f := range r.File {
		totalSize += f.FileInfo().Size()
	}

	done := make(chan int64)

	go SendExtractionProgressMessage(w, done, destination, totalSize)

	for _, f := range r.File {
		filePath := filepath.Join(destination, f.Name)

		if !strings.HasPrefix(filePath, filepath.Clean(destination)+string(os.PathSeparator)) {
			return fileNames, fmt.Errorf("%s: illegal file path", filePath)
		}

		fileNames = append(fileNames, filePath)

		if f.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return fileNames, err
		}

		outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fileNames, err
		}

		rc, err := f.Open()
		if err != nil {
			return fileNames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return fileNames, err
		}
	}

	err = r.Close()
	if err != nil {
		return fileNames, err
	}

	return fileNames, nil
}

func CrashWithMessage(w *astilectron.Window, message string) {
	w.SendMessage(EventMessage{"crash", message})
}
