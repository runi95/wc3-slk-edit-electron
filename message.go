package main

import (
	"encoding/json"
	"fmt"
	"github.com/asticode/go-astilectron"
	"github.com/asticode/go-astilectron-bootstrap"
	"io/ioutil"
	"log"
)

func HandleMessages(_ *astilectron.Window, m bootstrap.MessageIn) (payload interface{}, err error) {
	switch m.Name {
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
		var disabledInputExists bool
		disabledInputExists, err = exists(DISABLED_INPUTS_PATH)
		if err != nil {
			payload = err.Error()
			return
		}

		if disabledInputExists {
			var file []byte
			file, err = ioutil.ReadFile("disabled-inputs.json")
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
			var disabledInputs []string
			var file []byte
			file, err = json.Marshal(disabledInputs)
			if err != nil {
				payload = err.Error()
				return
			}

			err = ioutil.WriteFile(DISABLED_INPUTS_PATH, file, 0644)
			if err != nil {
				payload = err.Error()
				return
			}

			payload = nil
		}
	case "loadUnitData":
		loadSLK()
		var unitListData = make([]UnitListData, len(unitFuncMap))

		i := 0
		for k, v := range unitFuncMap {
			unitListData[i] = UnitListData{k, v.Name.String}
			i++
		}

		payload = unitListData
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
	}

	return
}
