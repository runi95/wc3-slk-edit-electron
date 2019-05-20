package main

import (
	"archive/zip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/asticode/go-astilectron"
	"github.com/asticode/go-astilectron-bootstrap"
	"github.com/runi95/wts-parser/models"
	"github.com/runi95/wts-parser/parser"
	"github.com/shibukawa/configdir"
	"gopkg.in/volatiletech/null.v6"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	VERSION = "v1.0.4"

	// const MAXINT = 2147483647
	VENDOR_NAME              = "wc3-slk-edit"
	CONFIG_FILENAME          = "config.json"
	DISABLED_INPUTS_FILENAME = "disabled-inputs.json"
	MODEL_DOWNLOAD_URL       = "http://www.maulbot.com/media/slk-editor-required-files.zip"
)

var (
	// Private Variables
	baseUnitMap    map[string]*models.SLKUnit
	unitFuncMap    map[string]*models.UnitFunc
	lastValidIndex int

	// Private Initialized Variables
	configDirs                             = configdir.New(VENDOR_NAME, "")
	configuration                          = &config{InDir: nil, OutDir: nil, IsLocked: false, IsDoneDownloadingModels: false, IsRegexSearch: false}
	globalConfig         *configdir.Config = nil
	defaultDisabledUnits                   = []string{
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

type UnitModel struct {
	Name string
	Path string
}

type UnitModels []UnitModel

func (unitModels UnitModels) Len() int {
	return len(unitModels)
}

func (unitModels UnitModels) Less(i, j int) bool {
	return unitModels[i].Name < unitModels[j].Name
}

func (unitModels UnitModels) Swap(i, j int) {
	unitModels[i], unitModels[j] = unitModels[j], unitModels[i]
}

func HandleMessages(w *astilectron.Window, m bootstrap.MessageIn) (payload interface{}, err error) {
	switch m.Name {
	case "saveFieldToUnit":
		var fieldToUnit FieldToUnit
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &fieldToUnit); err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			if _, ok := baseUnitMap[fieldToUnit.UnitId]; !ok {
				log.Println("Unit does not exist, returning unsaved")
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

			delete(baseUnitMap, unit)
			delete(unitFuncMap, unit)
			payload = unit
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

			var unitData = new(UnitData)
			unitData.UnitFunc = unitFuncMap[unitId]
			unitData.SLKUnit = baseUnitMap[unitId]

			missileartSplit := strings.Split(unitData.UnitFunc.Missileart.String, ",")
			missilearcSplit := strings.Split(unitData.UnitFunc.Missilearc.String, ",")
			missilespeedSplit := strings.Split(unitData.UnitFunc.Missilespeed.String, ",")
			missilehomingSplit := strings.Split(unitData.UnitFunc.Missilehoming.String, ",")
			if !unitData.UnitFunc.Missileart1.Valid {
				if len(missileartSplit) > 0 && len(missileartSplit[0]) > 0 {
					unitData.UnitFunc.Missileart1.SetValid(missileartSplit[0])
				}
			}

			if !unitData.UnitFunc.Missileart2.Valid {
				if len(missileartSplit) > 1 && len(missileartSplit[1]) > 0 {
					unitData.UnitFunc.Missileart2.SetValid(missileartSplit[1])
				}
			}

			if !unitData.UnitFunc.Missilearc1.Valid {
				if len(missilearcSplit) > 0 && len(missilearcSplit[0]) > 0 {
					unitData.UnitFunc.Missilearc1.SetValid(missilearcSplit[0])
				}
			}

			if !unitData.UnitFunc.Missilearc2.Valid {
				if len(missilearcSplit) > 1 && len(missilearcSplit[1]) > 0 {
					unitData.UnitFunc.Missilearc2.SetValid(missilearcSplit[1])
				}
			}

			if !unitData.UnitFunc.Missilespeed1.Valid {
				if len(missilespeedSplit) > 0 && len(missilespeedSplit[0]) > 0 {
					unitData.UnitFunc.Missilespeed1.SetValid(missilespeedSplit[0])
				}
			}

			if !unitData.UnitFunc.Missilespeed2.Valid {
				if len(missilespeedSplit) > 1 && len(missilespeedSplit[1]) > 0 {
					unitData.UnitFunc.Missilespeed2.SetValid(missilespeedSplit[1])
				}
			}

			if !unitData.UnitFunc.Missilehoming1.Valid {
				if len(missilehomingSplit) > 0 && len(missilehomingSplit[0]) > 0 {
					unitData.UnitFunc.Missilehoming1.SetValid(missilehomingSplit[0])
				}
			}

			if !unitData.UnitFunc.Missilehoming2.Valid {
				if len(missilehomingSplit) > 1 && len(missilehomingSplit[1]) > 0 {
					unitData.UnitFunc.Missilehoming2.SetValid(missilehomingSplit[1])
				}
			}

			payload = unitData
		} else {
			err = fmt.Errorf("invalid input")

			log.Println(err)
			payload = err.Error()
		}
	case "generateUnitId":
		payload = getNextValidUnitId(lastValidIndex)
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
		var unit UnitData
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &unit); err != nil {
				log.Println(err)
				payload = err.Error()
				return
			}

			baseUnitMap[unit.UnitFunc.UnitId] = unit.SLKUnit
			unitFuncMap[unit.UnitFunc.UnitId] = unit.UnitFunc

			payload = "success"
		} else {
			err = fmt.Errorf("invalid input")

			log.Println(err)
			payload = err.Error()
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
	case "loadIcons":
		iconModels := make(UnitModels, 0, len(images))
		for k := range images {
			path := strings.Replace(strings.Replace(k, "Command", "ReplaceableTextures\\CommandButtons", 1), "Passive", "ReplaceableTextures\\PassiveButtons", 1)
			name := strings.Replace(strings.Replace(strings.Replace(k, "Command\\BTN", "", 1), "Passive\\PASBTN", "", 1), ".blp", "", 1)

			iconModels = append(iconModels, UnitModel{name, path})
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
			unitAbilities := new(models.UnitAbilities)
			unitBalance := new(models.UnitBalance)
			unitData := new(models.UnitData)
			unitUI := new(models.UnitUI)
			unitWeapons := new(models.UnitWeapons)

			unitFunc := new(models.UnitFunc)
			unitFunc.Name.SetValid(newUnit.Name)

			var unitId string
			if newUnit.GenerateId == true || !newUnit.UnitId.Valid {
				unitId = getNextValidUnitId(lastValidIndex)
			} else {
				unitId = newUnit.UnitId.String
			}

			unitFunc.UnitId = unitId
			unitAbilities.UnitAbilID.SetValid(unitId)
			unitBalance.UnitBalanceID.SetValid(unitId)
			unitData.UnitID.SetValid(unitId)
			unitUI.UnitUIID.SetValid(unitId)
			unitWeapons.UnitWeapID.SetValid(unitId)

			unitType := strings.ToLower(newUnit.UnitType)
			if unitType == "unit" {
				// UnitFunc
				unitFunc.Hotkey.SetValid("F")
				unitFunc.Tip.SetValid("Train " + newUnit.Name + " [|cffffcc00F|r]")
				unitFunc.Ubertip.SetValid("Versatile foot soldier. Can learn the Defend ability. |n|n|cffffcc00Attacks land units.|r")
				unitFunc.Art.SetValid("ReplaceableTextures\\CommandButtons\\BTNFootman.blp")
				unitFunc.ButtonposX.SetValid("0")
				unitFunc.ButtonposY.SetValid("0")
				unitFunc.Buttonpos.SetValid("0,0")
				unitFunc.Specialart.SetValid("Objects\\Spawnmodels\\Human\\HumanLargeDeathExplode\\HumanLargeDeathExplode.mdl")

				// UnitAbilities
				unitAbilities.Auto.SetValid("_")
				unitAbilities.AbilList.SetValid("Adef,Aihn")

				// UnitBalance
				unitBalance.Level.SetValid("2")
				unitBalance.Type.SetValid("\"_\"")
				unitBalance.Goldcost.SetValid("135")
				unitBalance.Lumbercost.SetValid("0")
				unitBalance.GoldRep.SetValid("135")
				unitBalance.LumberRep.SetValid("0")
				unitBalance.Fmade.SetValid("\"-\"")
				unitBalance.Fused.SetValid("2")
				unitBalance.Bountydice.SetValid("6")
				unitBalance.Bountysides.SetValid("3")
				unitBalance.Bountyplus.SetValid("20")
				unitBalance.Lumberbountydice.SetValid("0")
				unitBalance.Lumberbountysides.SetValid("0")
				unitBalance.Lumberbountyplus.SetValid("0")
				unitBalance.StockMax.SetValid("3")
				unitBalance.StockRegen.SetValid("30")
				unitBalance.StockStart.SetValid("0")
				unitBalance.HP.SetValid("420")
				unitBalance.RealHP.SetValid("420")
				unitBalance.RegenHP.SetValid("0.25")
				unitBalance.RegenType.SetValid("\"always\"")
				unitBalance.ManaN.SetValid("\"-\"")
				unitBalance.RealM.SetValid("\"-\"")
				unitBalance.Mana0.SetValid("\"-\"")
				unitBalance.Def.SetValid("2")
				unitBalance.DefUp.SetValid("2")
				unitBalance.Realdef.SetValid("2")
				unitBalance.DefType.SetValid("\"large\"")
				unitBalance.Spd.SetValid("270")
				unitBalance.MinSpd.SetValid("0")
				unitBalance.MaxSpd.SetValid("0")
				unitBalance.Bldtm.SetValid("20")
				unitBalance.Reptm.SetValid("20")
				unitBalance.Sight.SetValid("1400")
				unitBalance.Nsight.SetValid("800")
				unitBalance.STR.SetValid("\"-\"")
				unitBalance.INT.SetValid("\"-\"")
				unitBalance.AGI.SetValid("\"-\"")
				unitBalance.STRplus.SetValid("\"-\"")
				unitBalance.INTplus.SetValid("\"-\"")
				unitBalance.AGIplus.SetValid("\"-\"")
				unitBalance.Primary.SetValid("\"_\"")
				unitBalance.Primary.SetValid("\"Rhar,Rhme,Rhde,Rhpm,Rguv\"")
				unitBalance.Isbldg.SetValid("0")
				unitBalance.PreventPlace.SetValid("\"_\"")
				unitBalance.RequirePlace.SetValid("\"_\"")
				unitBalance.Collision.SetValid("31")

				// UnitData
				unitData.Race.SetValid("\"human\"")
				unitData.Prio.SetValid("6")
				unitData.Threat.SetValid("1")
				unitData.Valid.SetValid("1")
				unitData.DeathType.SetValid("3")
				unitData.Death.SetValid("3.04")
				unitData.CanSleep.SetValid("0")
				unitData.CargoSize.SetValid("1")
				unitData.Movetp.SetValid("\"foot\"")
				unitData.MoveHeight.SetValid("0")
				unitData.MoveFloor.SetValid("0")
				unitData.TurnRate.SetValid("0.6")
				unitData.PropWin.SetValid("60")
				unitData.OrientInterp.SetValid("0")
				unitData.Formation.SetValid("0")
				unitData.TargType.SetValid("\"ground\"")
				unitData.PathTex.SetValid("\"_\"")
				unitData.Points.SetValid("100")
				unitData.CanFlee.SetValid("1")
				unitData.RequireWaterRadius.SetValid("0")
				unitData.IsBuildOn.SetValid("0")
				unitData.CanBuildOn.SetValid("0")

				// UnitUI
				unitUI.File.SetValid("\"units\\human\\Footman\\Footman\"")
				unitUI.FileVerFlags.SetValid("0")
				unitUI.UnitSound.SetValid("\"Footman\"")
				unitUI.TilesetSpecific.SetValid("0")
				unitUI.UnitClass.SetValid("\"HUnit02\"")
				unitUI.Special.SetValid("0")
				unitUI.Campaign.SetValid("0")
				unitUI.InEditor.SetValid("1")
				unitUI.HiddenInEditor.SetValid("0")
				unitUI.HostilePal.SetValid("\"-\"")
				unitUI.DropItems.SetValid("1")
				unitUI.NbmmIcon.SetValid("\"-\"")
				unitUI.UseClickHelper.SetValid("0")
				unitUI.HideHeroBar.SetValid("0")
				unitUI.HideHeroMinimap.SetValid("0")
				unitUI.HideHeroDeathMsg.SetValid("0")
				unitUI.HideOnMinimap.SetValid("0")
				unitUI.Blend.SetValid("0.15")
				unitUI.Scale.SetValid("1")
				unitUI.ScaleBull.SetValid("1")
				unitUI.MaxPitch.SetValid("10")
				unitUI.MaxRoll.SetValid("10")
				unitUI.ElevPts.SetValid("\"-\"")
				unitUI.ElevRad.SetValid("20")
				unitUI.FogRad.SetValid("0")
				unitUI.Walk.SetValid("210")
				unitUI.Run.SetValid("210")
				unitUI.SelZ.SetValid("0")
				unitUI.Weap1.SetValid("\"MetalMediumSlice\"")
				unitUI.Weap2.SetValid("\"_\"")
				unitUI.TeamColor.SetValid("-1")
				unitUI.CustomTeamColor.SetValid("0")
				unitUI.Armor.SetValid("\"Metal\"")
				unitUI.ModelScale.SetValid("1")
				unitUI.Red.SetValid("255")
				unitUI.Green.SetValid("255")
				unitUI.Blue.SetValid("255")
				unitUI.UberSplat.SetValid("\"_\"")
				unitUI.UnitShadow.SetValid("\"Shadow\"")
				unitUI.BuildingShadow.SetValid("\"_\"")
				unitUI.ShadowW.SetValid("140")
				unitUI.ShadowH.SetValid("140")
				unitUI.ShadowX.SetValid("50")
				unitUI.ShadowY.SetValid("50")
				unitUI.ShadowOnWater.SetValid("1")
				unitUI.SelCircOnWater.SetValid("0")
				unitUI.OccH.SetValid("0")
			} else if unitType == "building" {
				// UnitFunc
				unitFunc.Art.SetValid("ReplaceableTextures\\CommandButtons\\BTNFarm.blp")
				unitFunc.ButtonposX.SetValid("0")
				unitFunc.ButtonposY.SetValid("1")
				unitFunc.Buttonpos.SetValid("0,1")
				unitFunc.Buildingsoundlabel.SetValid("BuildingConstructionLoop")
				unitFunc.Loopingsoundfadein.SetValid("512")
				unitFunc.Loopingsoundfadeout.SetValid("512")
				unitFunc.Specialart.SetValid("Objects\\Spawnmodels\\Human\\HCancelDeath\\HCancelDeath.mdl")

				// UnitAbilities
				unitAbilities.Auto.SetValid("_")
				unitAbilities.AbilList.SetValid("Abds")

				// UnitBalance
				unitBalance.Level.SetValid("\"-\"")
				unitBalance.Type.SetValid("\"Mechanical\"")
				unitBalance.Goldcost.SetValid("80")
				unitBalance.Lumbercost.SetValid("20")
				unitBalance.GoldRep.SetValid("80")
				unitBalance.LumberRep.SetValid("20")
				unitBalance.Fmade.SetValid("6")
				unitBalance.Fused.SetValid("\"-\"")
				unitBalance.Bountydice.SetValid("0")
				unitBalance.Bountysides.SetValid("0")
				unitBalance.Bountyplus.SetValid("0")
				unitBalance.Lumberbountydice.SetValid("0")
				unitBalance.Lumberbountysides.SetValid("0")
				unitBalance.Lumberbountyplus.SetValid("0")
				unitBalance.StockMax.SetValid("\"-\"")
				unitBalance.StockRegen.SetValid("\"-\"")
				unitBalance.StockStart.SetValid("\"-\"")
				unitBalance.HP.SetValid("500")
				unitBalance.RealHP.SetValid("500")
				unitBalance.RegenHP.SetValid("\"-\"")
				unitBalance.RegenType.SetValid("\"none\"")
				unitBalance.ManaN.SetValid("\"-\"")
				unitBalance.RealM.SetValid("\"-\"")
				unitBalance.Mana0.SetValid("\"-\"")
				unitBalance.Def.SetValid("5")
				unitBalance.DefUp.SetValid("1")
				unitBalance.Realdef.SetValid("5")
				unitBalance.DefType.SetValid("\"fort\"")
				unitBalance.Spd.SetValid("\"-\"")
				unitBalance.MinSpd.SetValid("0")
				unitBalance.MaxSpd.SetValid("0")
				unitBalance.Bldtm.SetValid("35")
				unitBalance.Reptm.SetValid("35")
				unitBalance.Sight.SetValid("900")
				unitBalance.Nsight.SetValid("600")
				unitBalance.STR.SetValid("\"-\"")
				unitBalance.INT.SetValid("\"-\"")
				unitBalance.AGI.SetValid("\"-\"")
				unitBalance.STRplus.SetValid("\"-\"")
				unitBalance.INTplus.SetValid("\"-\"")
				unitBalance.AGIplus.SetValid("\"-\"")
				unitBalance.Primary.SetValid("\"_\"")
				unitBalance.Primary.SetValid("\"Rhac,Rgfo\"")
				unitBalance.Isbldg.SetValid("1")
				unitBalance.PreventPlace.SetValid("\"unbuildable\"")
				unitBalance.RequirePlace.SetValid("\"_\"")
				unitBalance.Collision.SetValid("72")

				// UnitData
				unitData.Race.SetValid("\"human\"")
				unitData.Prio.SetValid("1")
				unitData.Threat.SetValid("1")
				unitData.Valid.SetValid("1")
				unitData.DeathType.SetValid("2")
				unitData.Death.SetValid("2.34")
				unitData.CanSleep.SetValid("0")
				unitData.CargoSize.SetValid("\"-\"")
				unitData.Movetp.SetValid("\"_\"")
				unitData.MoveHeight.SetValid("0")
				unitData.MoveFloor.SetValid("0")
				unitData.TurnRate.SetValid("\"-\"")
				unitData.PropWin.SetValid("60")
				unitData.OrientInterp.SetValid("0")
				unitData.Formation.SetValid("0")
				unitData.TargType.SetValid("\"structure\"")
				unitData.PathTex.SetValid("\"PathTextures\\4x4SimpleSolid.tga\"")
				unitData.Points.SetValid("100")
				unitData.CanFlee.SetValid("1")
				unitData.RequireWaterRadius.SetValid("0")
				unitData.IsBuildOn.SetValid("0")
				unitData.CanBuildOn.SetValid("0")

				// UnitUI
				unitUI.File.SetValid("\"buildings\\human\\Farm\\Farm\"")
				unitUI.FileVerFlags.SetValid("0")
				unitUI.UnitSound.SetValid("\"Farm\"")
				unitUI.TilesetSpecific.SetValid("0")
				unitUI.UnitClass.SetValid("\"HBuilding04\"")
				unitUI.Special.SetValid("0")
				unitUI.Campaign.SetValid("0")
				unitUI.InEditor.SetValid("1")
				unitUI.HiddenInEditor.SetValid("0")
				unitUI.HostilePal.SetValid("\"-\"")
				unitUI.DropItems.SetValid("1")
				unitUI.NbmmIcon.SetValid("\"-\"")
				unitUI.UseClickHelper.SetValid("0")
				unitUI.HideHeroBar.SetValid("0")
				unitUI.HideHeroMinimap.SetValid("0")
				unitUI.HideHeroDeathMsg.SetValid("0")
				unitUI.HideOnMinimap.SetValid("0")
				unitUI.Blend.SetValid("0.15")
				unitUI.Scale.SetValid("2.5")
				unitUI.ScaleBull.SetValid("1")
				unitUI.MaxPitch.SetValid("15")
				unitUI.MaxRoll.SetValid("15")
				unitUI.ElevPts.SetValid("4")
				unitUI.ElevRad.SetValid("50")
				unitUI.FogRad.SetValid("0")
				unitUI.Walk.SetValid("200")
				unitUI.Run.SetValid("200")
				unitUI.SelZ.SetValid("0")
				unitUI.Weap1.SetValid("\"_\"")
				unitUI.Weap2.SetValid("\"_\"")
				unitUI.TeamColor.SetValid("-1")
				unitUI.CustomTeamColor.SetValid("0")
				unitUI.Armor.SetValid("\"Wood\"")
				unitUI.ModelScale.SetValid("1")
				unitUI.Red.SetValid("255")
				unitUI.Green.SetValid("255")
				unitUI.Blue.SetValid("255")
				unitUI.UberSplat.SetValid("\"HSMA\"")
				unitUI.UnitShadow.SetValid("\"_\"")
				unitUI.BuildingShadow.SetValid("\"ShadowHouse\"")
				unitUI.ShadowOnWater.SetValid("1")
				unitUI.SelCircOnWater.SetValid("0")
				unitUI.OccH.SetValid("0")
			} else if unitType == "hero" {
				// TODO: Implement
			}

			if newUnit.AttackType == "0" { // None
				unitWeapons.WeapsOn.SetValid("0")
				unitWeapons.Acquire.SetValid("\"-\"")
				unitWeapons.MinRange.SetValid("\"-\"")
				unitWeapons.Castpt.SetValid("\"-\"")
				unitWeapons.Castbsw.SetValid("0.51")
				unitWeapons.LaunchX.SetValid("0")
				unitWeapons.LaunchY.SetValid("0")
				unitWeapons.LaunchZ.SetValid("60")
				unitWeapons.LaunchSwimZ.SetValid("0")
				unitWeapons.ImpactZ.SetValid("120")
				unitWeapons.ImpactSwimZ.SetValid("0")
				unitWeapons.WeapType1.SetValid("\"_\"")
				unitWeapons.Targs1.SetValid("\"_\"")
				unitWeapons.ShowUI1.SetValid("1")
				unitWeapons.RangeN1.SetValid("\"-\"")
				unitWeapons.RngTst.SetValid("\"-\"")
				unitWeapons.RngBuff1.SetValid("\"-\"")
				unitWeapons.AtkType1.SetValid("\"normal\"")
				unitWeapons.WeapTp1.SetValid("\"-\"")
				unitWeapons.Cool1.SetValid("\"-\"")
				unitWeapons.Mincool1.SetValid("\"-\"")
				unitWeapons.Dice1.SetValid("\"-\"")
				unitWeapons.Sides1.SetValid("\"-\"")
				unitWeapons.Dmgplus1.SetValid("\"-\"")
				unitWeapons.DmgUp1.SetValid("\"-\"")
				unitWeapons.Mindmg1.SetValid("\"-\"")
				unitWeapons.Avgdmg1.SetValid("\"-\"")
				unitWeapons.Maxdmg1.SetValid("\"-\"")
				unitWeapons.Dmgpt1.SetValid("\"-\"")
				unitWeapons.BackSw1.SetValid("\"-\"")
				unitWeapons.Farea1.SetValid("\"-\"")
				unitWeapons.Harea1.SetValid("\"-\"")
				unitWeapons.Qarea1.SetValid("\"-\"")
				unitWeapons.Hfact1.SetValid("\"-\"")
				unitWeapons.Qfact1.SetValid("\"-\"")
				unitWeapons.SplashTargs1.SetValid("\"_\"")
				unitWeapons.TargCount1.SetValid("\"-\"")
				unitWeapons.DamageLoss1.SetValid("0")
				unitWeapons.SpillDist1.SetValid("0")
				unitWeapons.SpillRadius1.SetValid("0")
				unitWeapons.DmgUpg.SetValid("\"-\"")
				unitWeapons.Dmod1.SetValid("\"-\"")
				unitWeapons.DPS.SetValid("\"-\"")
				unitWeapons.WeapType2.SetValid("\"_\"")
				unitWeapons.Targs2.SetValid("\"_\"")
				unitWeapons.ShowUI2.SetValid("\"-\"")
				unitWeapons.RangeN2.SetValid("\"-\"")
				unitWeapons.RngTst2.SetValid("\"-\"")
				unitWeapons.RngBuff2.SetValid("\"-\"")
				unitWeapons.AtkType2.SetValid("\"normal\"")
				unitWeapons.WeapTp2.SetValid("\"_\"")
				unitWeapons.Cool2.SetValid("\"-\"")
				unitWeapons.Mincool2.SetValid("\"-\"")
				unitWeapons.Dice2.SetValid("\"-\"")
				unitWeapons.Sides2.SetValid("\"-\"")
				unitWeapons.Dmgplus2.SetValid("\"-\"")
				unitWeapons.DmgUp2.SetValid("\"-\"")
				unitWeapons.Mindmg2.SetValid("\"-\"")
				unitWeapons.Avgdmg2.SetValid("\"-\"")
				unitWeapons.Maxdmg2.SetValid("\"-\"")
				unitWeapons.Dmgpt2.SetValid("\"-\"")
				unitWeapons.BackSw2.SetValid("\"-\"")
				unitWeapons.Farea2.SetValid("\"-\"")
				unitWeapons.Harea2.SetValid("\"-\"")
				unitWeapons.Qarea2.SetValid("\"-\"")
				unitWeapons.Hfact2.SetValid("\"-\"")
				unitWeapons.Qfact2.SetValid("\"-\"")
				unitWeapons.SplashTargs2.SetValid("\"_\"")
				unitWeapons.TargCount2.SetValid("\"-\"")
				unitWeapons.DamageLoss2.SetValid("0")
				unitWeapons.SpillDist2.SetValid("0")
				unitWeapons.SpillRadius2.SetValid("0")
			} else if newUnit.AttackType == "1" { // Melee
				unitWeapons.WeapsOn.SetValid("1")
				unitWeapons.Acquire.SetValid("500")
				unitWeapons.MinRange.SetValid("\"-\"")
				unitWeapons.Castpt.SetValid("0.3")
				unitWeapons.Castbsw.SetValid("0.51")
				unitWeapons.LaunchX.SetValid("0")
				unitWeapons.LaunchY.SetValid("0")
				unitWeapons.LaunchZ.SetValid("60")
				unitWeapons.LaunchSwimZ.SetValid("0")
				unitWeapons.ImpactZ.SetValid("60")
				unitWeapons.ImpactSwimZ.SetValid("0")
				unitWeapons.WeapType1.SetValid("\"MetalMediumSlice\"")
				unitWeapons.Targs1.SetValid("\"ground,structure,debris,item,ward\"")
				unitWeapons.ShowUI1.SetValid("1")
				unitWeapons.RangeN1.SetValid("90")
				unitWeapons.RngTst.SetValid("\"-\"")
				unitWeapons.RngBuff1.SetValid("250")
				unitWeapons.AtkType1.SetValid("\"normal\"")
				unitWeapons.WeapTp1.SetValid("\"normal\"")
				unitWeapons.Cool1.SetValid("1.35")
				unitWeapons.Mincool1.SetValid("\"-\"")
				unitWeapons.Dice1.SetValid("1")
				unitWeapons.Sides1.SetValid("2")
				unitWeapons.Dmgplus1.SetValid("11")
				unitWeapons.DmgUp1.SetValid("\"-\"")
				unitWeapons.Mindmg1.SetValid("12")
				unitWeapons.Avgdmg1.SetValid("12.5")
				unitWeapons.Maxdmg1.SetValid("13")
				unitWeapons.Dmgpt1.SetValid("0.5")
				unitWeapons.BackSw1.SetValid("0.5")
				unitWeapons.Farea1.SetValid("\"-\"")
				unitWeapons.Harea1.SetValid("\"-\"")
				unitWeapons.Qarea1.SetValid("\"-\"")
				unitWeapons.Hfact1.SetValid("\"-\"")
				unitWeapons.Qfact1.SetValid("\"-\"")
				unitWeapons.SplashTargs1.SetValid("\"_\"")
				unitWeapons.TargCount1.SetValid("1")
				unitWeapons.DamageLoss1.SetValid("0")
				unitWeapons.SpillDist1.SetValid("0")
				unitWeapons.SpillRadius1.SetValid("0")
				unitWeapons.DmgUpg.SetValid("\"-\"")
				unitWeapons.Dmod1.SetValid("\"-\"")
				unitWeapons.DPS.SetValid("9.25925925925926")
				unitWeapons.WeapType2.SetValid("\"_\"")
				unitWeapons.Targs2.SetValid("\"_\"")
				unitWeapons.ShowUI2.SetValid("1")
				unitWeapons.RangeN2.SetValid("\"-\"")
				unitWeapons.RngTst2.SetValid("\"-\"")
				unitWeapons.RngBuff2.SetValid("\"-\"")
				unitWeapons.AtkType2.SetValid("\"normal\"")
				unitWeapons.WeapTp2.SetValid("\"_\"")
				unitWeapons.Cool2.SetValid("\"-\"")
				unitWeapons.Mincool2.SetValid("\"-\"")
				unitWeapons.Dice2.SetValid("\"-\"")
				unitWeapons.Sides2.SetValid("\"-\"")
				unitWeapons.Dmgplus2.SetValid("\"-\"")
				unitWeapons.DmgUp2.SetValid("\"-\"")
				unitWeapons.Mindmg2.SetValid("\"-\"")
				unitWeapons.Avgdmg2.SetValid("\"-\"")
				unitWeapons.Maxdmg2.SetValid("\"-\"")
				unitWeapons.Dmgpt2.SetValid("\"-\"")
				unitWeapons.BackSw2.SetValid("\"-\"")
				unitWeapons.Farea2.SetValid("\"-\"")
				unitWeapons.Harea2.SetValid("\"-\"")
				unitWeapons.Qarea2.SetValid("\"-\"")
				unitWeapons.Hfact2.SetValid("\"-\"")
				unitWeapons.Qfact2.SetValid("\"-\"")
				unitWeapons.SplashTargs2.SetValid("\"_\"")
				unitWeapons.TargCount2.SetValid("1")
				unitWeapons.DamageLoss2.SetValid("0")
				unitWeapons.SpillDist2.SetValid("0")
				unitWeapons.SpillRadius2.SetValid("0")
			} else if newUnit.AttackType == "2" { // Ranged
				unitFunc.Missileart.SetValid("Abilities\\Weapons\\GuardTowerMissile\\GuardTowerMissile.mdl")
				unitFunc.Missileart1.SetValid("Abilities\\Weapons\\GuardTowerMissile\\GuardTowerMissile.mdl")
				unitFunc.Missilearc.SetValid("0.15")
				unitFunc.Missilearc1.SetValid("0.15")
				unitFunc.Missilespeed.SetValid("1800")
				unitFunc.Missilespeed1.SetValid("1800")

				unitWeapons.WeapsOn.SetValid("1")
				unitWeapons.Acquire.SetValid("700")
				unitWeapons.MinRange.SetValid("\"-\"")
				unitWeapons.Castpt.SetValid("0.3")
				unitWeapons.Castbsw.SetValid("0.51")
				unitWeapons.LaunchX.SetValid("0")
				unitWeapons.LaunchY.SetValid("0")
				unitWeapons.LaunchZ.SetValid("145")
				unitWeapons.LaunchSwimZ.SetValid("0")
				unitWeapons.ImpactZ.SetValid("120")
				unitWeapons.ImpactSwimZ.SetValid("0")
				unitWeapons.WeapType1.SetValid("\"_\"")
				unitWeapons.Targs1.SetValid("\"ground,structure,debris,air,item,ward\"")
				unitWeapons.ShowUI1.SetValid("1")
				unitWeapons.RangeN1.SetValid("700")
				unitWeapons.RngTst.SetValid("\"-\"")
				unitWeapons.RngBuff1.SetValid("250")
				unitWeapons.AtkType1.SetValid("\"pierce\"")
				unitWeapons.WeapTp1.SetValid("\"missile\"")
				unitWeapons.Cool1.SetValid("0.9")
				unitWeapons.Mincool1.SetValid("\"-\"")
				unitWeapons.Dice1.SetValid("1")
				unitWeapons.Sides1.SetValid("5")
				unitWeapons.Dmgplus1.SetValid("22")
				unitWeapons.DmgUp1.SetValid("\"-\"")
				unitWeapons.Mindmg1.SetValid("23")
				unitWeapons.Avgdmg1.SetValid("25")
				unitWeapons.Maxdmg1.SetValid("27")
				unitWeapons.Dmgpt1.SetValid("0.3")
				unitWeapons.BackSw1.SetValid("0.3")
				unitWeapons.Farea1.SetValid("\"-\"")
				unitWeapons.Harea1.SetValid("\"-\"")
				unitWeapons.Qarea1.SetValid("\"-\"")
				unitWeapons.Hfact1.SetValid("\"-\"")
				unitWeapons.Qfact1.SetValid("\"-\"")
				unitWeapons.SplashTargs1.SetValid("\"_\"")
				unitWeapons.TargCount1.SetValid("1")
				unitWeapons.DamageLoss1.SetValid("0")
				unitWeapons.SpillDist1.SetValid("0")
				unitWeapons.SpillRadius1.SetValid("0")
				unitWeapons.DmgUpg.SetValid("\"-\"")
				unitWeapons.Dmod1.SetValid("\"-\"")
				unitWeapons.DPS.SetValid("27.7777777777778")
				unitWeapons.WeapType2.SetValid("\"_\"")
				unitWeapons.Targs2.SetValid("\"_\"")
				unitWeapons.ShowUI2.SetValid("1")
				unitWeapons.RangeN2.SetValid("\"-\"")
				unitWeapons.RngTst2.SetValid("\"-\"")
				unitWeapons.RngBuff2.SetValid("\"-\"")
				unitWeapons.AtkType2.SetValid("\"normal\"")
				unitWeapons.WeapTp2.SetValid("\"_\"")
				unitWeapons.Cool2.SetValid("\"-\"")
				unitWeapons.Mincool2.SetValid("\"-\"")
				unitWeapons.Dice2.SetValid("\"-\"")
				unitWeapons.Sides2.SetValid("\"-\"")
				unitWeapons.Dmgplus2.SetValid("\"-\"")
				unitWeapons.DmgUp2.SetValid("\"-\"")
				unitWeapons.Mindmg2.SetValid("\"-\"")
				unitWeapons.Avgdmg2.SetValid("\"-\"")
				unitWeapons.Maxdmg2.SetValid("\"-\"")
				unitWeapons.Dmgpt2.SetValid("\"-\"")
				unitWeapons.BackSw2.SetValid("\"-\"")
				unitWeapons.Farea2.SetValid("\"-\"")
				unitWeapons.Harea2.SetValid("\"-\"")
				unitWeapons.Qarea2.SetValid("\"-\"")
				unitWeapons.Hfact2.SetValid("\"-\"")
				unitWeapons.Qfact2.SetValid("\"-\"")
				unitWeapons.SplashTargs2.SetValid("\"_\"")
				unitWeapons.TargCount2.SetValid("1")
				unitWeapons.DamageLoss2.SetValid("0")
				unitWeapons.SpillDist2.SetValid("0")
				unitWeapons.SpillRadius2.SetValid("0")
			} else if newUnit.AttackType == "3" { // Ranged (Splash)
				unitFunc.Missileart.SetValid("Abilities\\Weapons\\CannonTowerMissile\\CannonTowerMissile.mdl")
				unitFunc.Missileart1.SetValid("Abilities\\Weapons\\CannonTowerMissile\\CannonTowerMissile.mdl")
				unitFunc.Missilearc.SetValid("0.35")
				unitFunc.Missilearc1.SetValid("0.35")
				unitFunc.Missilespeed.SetValid("700")
				unitFunc.Missilespeed1.SetValid("700")

				unitWeapons.WeapsOn.SetValid("3")
				unitWeapons.Acquire.SetValid("800")
				unitWeapons.MinRange.SetValid("\"-\"")
				unitWeapons.Castpt.SetValid("\"-\"")
				unitWeapons.Castbsw.SetValid("0.51")
				unitWeapons.LaunchX.SetValid("0")
				unitWeapons.LaunchY.SetValid("0")
				unitWeapons.LaunchZ.SetValid("160")
				unitWeapons.LaunchSwimZ.SetValid("0")
				unitWeapons.ImpactZ.SetValid("120")
				unitWeapons.ImpactSwimZ.SetValid("0")
				unitWeapons.WeapType1.SetValid("\"_\"")
				unitWeapons.Targs1.SetValid("\"ground,debris,tree,wall,ward,item\"")
				unitWeapons.ShowUI1.SetValid("1")
				unitWeapons.RangeN1.SetValid("800")
				unitWeapons.RngTst.SetValid("\"-\"")
				unitWeapons.RngBuff1.SetValid("250")
				unitWeapons.AtkType1.SetValid("\"siege\"")
				unitWeapons.WeapTp1.SetValid("\"msplash\"")
				unitWeapons.Cool1.SetValid("2.5")
				unitWeapons.Mincool1.SetValid("\"-\"")
				unitWeapons.Dice1.SetValid("1")
				unitWeapons.Sides1.SetValid("22")
				unitWeapons.Dmgplus1.SetValid("89")
				unitWeapons.DmgUp1.SetValid("\"-\"")
				unitWeapons.Mindmg1.SetValid("90")
				unitWeapons.Avgdmg1.SetValid("100.5")
				unitWeapons.Maxdmg1.SetValid("111")
				unitWeapons.Dmgpt1.SetValid("0.3")
				unitWeapons.BackSw1.SetValid("0.3")
				unitWeapons.Farea1.SetValid("50")
				unitWeapons.Harea1.SetValid("100")
				unitWeapons.Qarea1.SetValid("125")
				unitWeapons.Hfact1.SetValid("0.5")
				unitWeapons.Qfact1.SetValid("0.1")
				unitWeapons.SplashTargs1.SetValid("ground,structure,debris,tree,wall,notself")
				unitWeapons.TargCount1.SetValid("1")
				unitWeapons.DamageLoss1.SetValid("0")
				unitWeapons.SpillDist1.SetValid("0")
				unitWeapons.SpillRadius1.SetValid("0")
				unitWeapons.DmgUpg.SetValid("\"-\"")
				unitWeapons.Dmod1.SetValid("84")
				unitWeapons.DPS.SetValid("40.2")
				unitWeapons.WeapType2.SetValid("\"_\"")
				unitWeapons.Targs2.SetValid("\"_\"")
				unitWeapons.ShowUI2.SetValid("1")
				unitWeapons.RangeN2.SetValid("\"-\"")
				unitWeapons.RngTst2.SetValid("\"-\"")
				unitWeapons.RngBuff2.SetValid("\"-\"")
				unitWeapons.AtkType2.SetValid("\"normal\"")
				unitWeapons.WeapTp2.SetValid("\"_\"")
				unitWeapons.Cool2.SetValid("\"-\"")
				unitWeapons.Mincool2.SetValid("\"-\"")
				unitWeapons.Dice2.SetValid("\"-\"")
				unitWeapons.Sides2.SetValid("\"-\"")
				unitWeapons.Dmgplus2.SetValid("\"-\"")
				unitWeapons.DmgUp2.SetValid("\"-\"")
				unitWeapons.Mindmg2.SetValid("\"-\"")
				unitWeapons.Avgdmg2.SetValid("\"-\"")
				unitWeapons.Maxdmg2.SetValid("\"-\"")
				unitWeapons.Dmgpt2.SetValid("\"-\"")
				unitWeapons.BackSw2.SetValid("\"-\"")
				unitWeapons.Farea2.SetValid("\"-\"")
				unitWeapons.Harea2.SetValid("\"-\"")
				unitWeapons.Qarea2.SetValid("\"-\"")
				unitWeapons.Hfact2.SetValid("\"-\"")
				unitWeapons.Qfact2.SetValid("\"-\"")
				unitWeapons.SplashTargs2.SetValid("\"_\"")
				unitWeapons.TargCount2.SetValid("1")
				unitWeapons.DamageLoss2.SetValid("0")
				unitWeapons.SpillDist2.SetValid("0")
				unitWeapons.SpillRadius2.SetValid("0")
			}

			unit.UnitAbilities = unitAbilities
			unit.UnitBalance = unitBalance
			unit.UnitData = unitData
			unit.UnitUI = unitUI
			unit.UnitWeapons = unitWeapons

			baseUnitMap[unitId] = unit
			unitFuncMap[unitId] = unitFunc

			var payloadUnit = new(UnitData)
			payloadUnit.UnitFunc = unitFunc
			payloadUnit.SLKUnit = unit

			payload = payloadUnit
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

			if !configuration.IsDoneDownloadingModels {
				err = startDownload(w, path)
				if err != nil {
					log.Println(err)
					payload = err.Error()
					return
				}

				configuration.IsDoneDownloadingModels = true

				err = saveConfig()
				if err != nil {
					log.Println(err)
					payload = err.Error()
					return
				}
			}

			var unitModelList UnitModels
			walkPath := path + string(filepath.Separator) + "resources" + string(filepath.Separator) + "units"

			err = filepath.Walk(walkPath, func(currentPath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if !info.IsDir() {
					index := strings.LastIndex(info.Name(), ".")
					if index > -1 {
						if info.Name()[index:] == ".mdx" {
							unitModelList = append(unitModelList, UnitModel{info.Name()[:index], "units" + currentPath[len(walkPath):]})
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

			sort.Sort(unitModelList)

			payload = unitModelList
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
	case "loadVersion":
		payload = VERSION
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
	var inputDirectory string

	if configuration.InDir == nil {
		log.Println("Input directory has not been set!")
		return
	}

	inputDirectory = *configuration.InDir
	unitAbilitiesPath := filepath.Join(inputDirectory, "UnitAbilities.slk")
	unitDataPath := filepath.Join(inputDirectory, "UnitData.slk")
	unitUIPath := filepath.Join(inputDirectory, "UnitUI.slk")
	unitWeaponsPath := filepath.Join(inputDirectory, "UnitWeapons.slk")
	unitBalancePath := filepath.Join(inputDirectory, "UnitBalance.slk")
	campaignUnitPath := filepath.Join(inputDirectory, "CampaignUnitFunc.txt")

	if flag, err := exists(unitAbilitiesPath); err != nil || !flag {
		return
	}

	if flag, err := exists(unitDataPath); err != nil || !flag {
		return
	}

	if flag, err := exists(unitUIPath); err != nil || !flag {
		return
	}

	if flag, err := exists(unitWeaponsPath); err != nil || !flag {
		return
	}

	if flag, err := exists(unitBalancePath); err != nil || !flag {
		return
	}

	if flag, err := exists(campaignUnitPath); err != nil || !flag {
		return
	}

	log.Println("Reading UnitAbilities.slk...")

	unitAbilitiesBytes, err := ioutil.ReadFile(unitAbilitiesPath)
	if err != nil {
		CrashWithMessage(w, err.Error())
	}

	unitAbilitiesMap := parser.SlkToUnitAbilities(unitAbilitiesBytes)

	log.Println("Reading UnitData.slk...")

	unitDataBytes, err := ioutil.ReadFile(unitDataPath)
	if err != nil {
		CrashWithMessage(w, err.Error())
	}

	unitDataMap := parser.SlkToUnitData(unitDataBytes)

	log.Println("Reading UnitUI.slk...")

	unitUIBytes, err := ioutil.ReadFile(unitUIPath)
	if err != nil {
		CrashWithMessage(w, err.Error())
	}

	unitUIMap := parser.SLKToUnitUI(unitUIBytes)

	log.Println("Reading UnitWeapons.slk...")

	unitWeaponsBytes, err := ioutil.ReadFile(unitWeaponsPath)
	if err != nil {
		CrashWithMessage(w, err.Error())
	}

	unitWeaponsMap := parser.SLKToUnitWeapons(unitWeaponsBytes)

	log.Println("Reading UnitBalance.slk...")

	unitBalanceBytes, err := ioutil.ReadFile(unitBalancePath)
	if err != nil {
		CrashWithMessage(w, err.Error())
	}

	unitBalanceMap := parser.SLKToUnitBalance(unitBalanceBytes)

	log.Println("Reading CampaignUnitFunc.txt...")

	campaignUnitFuncBytes, err := ioutil.ReadFile(campaignUnitPath)
	if err != nil {
		CrashWithMessage(w, err.Error())
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

	out, err := os.Create(file)
	if err != nil {
		return err
	}

	headResp, err := http.Head(url)
	if err != nil {
		return err
	}

	defer headResp.Body.Close()

	size, err := strconv.Atoi(headResp.Header.Get("Content-Length"))
	if err != nil {
		return err
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	w.SendMessage(EventMessage{"downloadTextUpdate", "Downloading..."})

	done := make(chan int64)

	go SendDownloadProgressMessage(w, done, file, int64(size))

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

	w.SendMessage(EventMessage{"downloadTextUpdate", "Extracting..."})

	unzipDestination := path + string(filepath.Separator) + "resources"

	err = os.Mkdir(unzipDestination, os.ModePerm)
	if err != nil {
		return err
	}

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
