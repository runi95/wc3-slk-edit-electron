package main

import (
	"encoding/json"
	"fmt"
	"github.com/asticode/go-astilectron"
	"github.com/asticode/go-astilectron-bootstrap"
	"gopkg.in/volatiletech/null.v6"
	"io/ioutil"
	"log"
	"reflect"
	"strings"
)

type FieldToUnit struct {
	UnitId string
	Field string
	Value string
}

func reflectUpdateValueOnFieldNullStruct(iface interface{}, fieldValue interface{}, fieldName string) error {
	valueIface := reflect.ValueOf(iface)

	// Check if the passed interface is a pointer
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
		// change value of the field with name fieldName
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

func HandleMessages(_ *astilectron.Window, m bootstrap.MessageIn) (payload interface{}, err error) {
	switch m.Name {
	case "saveFieldToUnit":
		var fieldToUnit FieldToUnit
		if len(m.Payload) > 0 {
			if err = json.Unmarshal(m.Payload, &fieldToUnit); err != nil {
				payload = err.Error()
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
