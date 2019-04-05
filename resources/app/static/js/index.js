let unitDataList = [];
const selectedUnitIndex = null;

const addUnitTableData = (unitTableBody, unitData) => {
    const tr = document.createElement("tr");
    const th = document.createElement("th");
    const td = document.createElement("td");
    th.setAttribute("scope", "row");
    th.appendChild(document.createTextNode(unitData.UnitID));
    td.appendChild(document.createTextNode(unitData.Name));
    tr.id = unitData.UnitID;
    tr.onclick = () => index.selectUnit(tr);
    tr.appendChild(th);
    tr.appendChild(td);
    unitTableBody.appendChild(tr);
};

const index = {
    init: function () {
        // Init
        asticode.loader.init();
        asticode.notifier.init();

        // Wait for astilectron to be ready
        document.addEventListener('astilectron-ready', function () {
            // Listen
            index.listen();

            index.disableInputs(true);
            index.loadConfig();
            index.loadUnitData();
        })
    },
    listen: function () {
        astilectron.onMessage(function (message) {

        });
    },
    removeUnit: function (unit) {
        const unitMessage = {name: "removeUnit", payload: unit};

        astilectron.sendMessage(unitMessage, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            document.getElementById("removeButton").innerHTML = message.payload;
        })
    },
    loadUnitData: function () {
        const message = {name: "loadUnitData", payload: null};
        asticode.loader.show();

        astilectron.sendMessage(message, function (message) {
            asticode.loader.hide();

            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            unitDataList = message.payload;
            const unitTableBody = $("#unitTableBody")[0];
            message.payload.forEach(unitData => {
                addUnitTableData(unitTableBody, unitData);
            });
        });
    },
    search: function (inputField) {
        const regex = new RegExp(inputField.value, "i");
        const filteredUnitDataList = unitDataList.filter(unitData => (unitData.Name + unitData.UnitID).match(regex));
        const unitTableBody = $("#unitTableBody")[0];
        unitTableBody.innerHTML = '';

        filteredUnitDataList.forEach(unitData => {
            addUnitTableData(unitTableBody, unitData);
        });
    },
    selectUnit: function (unitTableRow) {
        const unitId = unitTableRow.id;
        const message = {name: "selectUnit", payload: unitId};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            $("#unitTableBody>tr").removeClass("active");
            unitTableRow.setAttribute("class", "active");
            Object.keys(message.payload.SLKUnit).forEach(slkUnitKey =>
                Object.keys(message.payload.SLKUnit[slkUnitKey]).forEach(key => {
                    const rawValue = message.payload.SLKUnit[slkUnitKey][key];
                    if (rawValue) {
                        const trimmedRight = rawValue.endsWith("\"") ? rawValue.substr(0, rawValue.length - 1) : rawValue;
                        const value = trimmedRight.startsWith("\"") ? trimmedRight.substr(1) : trimmedRight;
                        const elemList = $("#SLKUnit-" + slkUnitKey + "-" + key);
                        if (elemList.length > 0) {
                            if (elemList[0] instanceof HTMLInputElement || elemList[0] instanceof HTMLSelectElement) {
                                const type = elemList[0].type;

                                if (type === "text" || type === "select-one") {
                                    elemList[0].value = value;
                                } else if (type === "checkbox") {
                                    elemList[0].checked = value === "1";
                                }
                            } else if (elemList[0].classList.contains("multi-check")) {
                                const childInputs = $("#SLKUnit-" + slkUnitKey + "-" + key + " :input");
                                for (let i = 0; i < childInputs.length; i++) {
                                    if (value.includes(childInputs[i].value)) {
                                        childInputs[i].checked = true;
                                    } else {
                                        childInputs[i].checked = false;
                                    }
                                }
                            }
                        }
                    }
                }));

            Object.keys(message.payload.UnitFunc).forEach(unitFuncKey => {
                const elemList = $("#UnitFunc-" + unitFuncKey);
                if (elemList.length > 0) {
                    if (!elemList[0] instanceof HTMLInputElement)
                        return;

                    const type = elemList[0].type;
                    if (type === "text" || type === "select-one") {
                        elemList[0].value = message.payload.UnitFunc[unitFuncKey];
                    } else if (type === "checkbox") {
                        elemList[0].checked = message.payload.UnitFunc[unitFuncKey] === "1";
                    }
                }
            });
        });
    },
    activateFileUploadButton: function () {
        $("#hiddenFileUploadInput").click();
    },
    saveToFile: function () {
        const input = $("#outputFolderInput");

        if (input.length < 1)
            return;

        const outputDir = input[0].value;
        const message = {name: "saveToFile", payload: outputDir};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }
        });
    },
    updateOutputFolder: function () {
        const hiddenInput = $("#hiddenFileUploadInput");
        const input = $("#outputFolderInput");

        if (input.length < 1 || hiddenInput.length < 1)
            return;

        const updatedValue = hiddenInput[0].files[0].path;

        if (!updatedValue)
            return;

        input[0].value = updatedValue;
    },
    saveUnit: function () {
        const forms = $("form");
        let formsValid = true;
        for (let i = 0; i < forms.length; i++) {
            if (!forms[i].checkValidity()) {
                formsValid = false;
            }
        }

        if (!formsValid)
            return;

        const unit = {
            SLKUnit: {UnitUI: {}, UnitData: {}, UnitBalance: {}, UnitWeapons: {}, UnitAbilities: {}},
            UnitFunc: {}
        };

        const inputs = $(":input");
        for (let i = 0; i < inputs.length; i++) {
            const type = inputs[i].type;
            const idSplit = inputs[i].id.split("-");
            if (inputs[i].id) {
                if (type === "checkbox") {
                    if (idSplit[0] === "SLKUnit") {
                        unit.SLKUnit[idSplit[1]][idSplit[2]] = inputs[i].checked ? "1" : "0"
                    } else if (idSplit[0] === "UnitFunc") {
                        unit.UnitFunc[idSplit[1]] = inputs[i].checked ? "1" : "0"
                    }
                } else if(inputs[i].value) {
                    if (idSplit[0] === "SLKUnit") {
                        unit.SLKUnit[idSplit[1]][idSplit[2]] = inputs[i].value
                    } else if (idSplit[0] === "UnitFunc") {
                        unit.UnitFunc[idSplit[1]] = inputs[i].value
                    }
                }
            } else if(type === "checkbox" && inputs[i].checked) {
                const parentNode = inputs[i].parentNode.parentNode.parentNode;
                const parentIdSplit = parentNode.id.split("-");
                const oldValue = unit.SLKUnit[parentIdSplit[1]][parentIdSplit[2]];
                if (oldValue) {
                    unit.SLKUnit[parentIdSplit[1]][parentIdSplit[2]] += "," + inputs[i].value;
                } else {
                    unit.SLKUnit[parentIdSplit[1]][parentIdSplit[2]] = inputs[i].value;
                }
            }
        }

        const unitId = unit.UnitFunc.UnitId;
        const quotedUnitId = "\"" + unitId + "\"";
        unit.SLKUnit.UnitData.UnitID = quotedUnitId;
        unit.SLKUnit.UnitBalance.UnitBalanceID = quotedUnitId;
        unit.SLKUnit.UnitUI.UnitUIID = quotedUnitId;
        unit.SLKUnit.UnitWeapons.UnitWeapID = quotedUnitId;
        unit.SLKUnit.UnitAbilities.UnitAbilID = quotedUnitId;

        const containsNumberRegex = new RegExp("^(?:(?:\\d*)|(?:\\d+\\.\\d+))$");
        Object.keys(unit.SLKUnit.UnitWeapons).forEach(key => {
            const val = unit.SLKUnit.UnitWeapons[key];
            const isNotQuoted = !(val.startsWith("\"") && val.endsWith("\""));
            if (isNotQuoted && !val.match(containsNumberRegex)) {
                unit.SLKUnit.UnitWeapons[key] = "\"" + val + "\"";
            }
        });
        Object.keys(unit.SLKUnit.UnitUI).forEach(key => {
            const val = unit.SLKUnit.UnitUI[key];
            const isNotQuoted = !(val.startsWith("\"") && val.endsWith("\""));
            if (isNotQuoted && !val.match(containsNumberRegex)) {
                unit.SLKUnit.UnitUI[key] = "\"" + val + "\"";
            }
        });
        Object.keys(unit.SLKUnit.UnitData).forEach(key => {
            const val = unit.SLKUnit.UnitData[key];
            const isNotQuoted = !(val.startsWith("\"") && val.endsWith("\""));
            if (isNotQuoted && !val.match(containsNumberRegex)) {
                unit.SLKUnit.UnitData[key] = "\"" + val + "\"";
            }
        });
        Object.keys(unit.SLKUnit.UnitBalance).forEach(key => {
            const val = unit.SLKUnit.UnitBalance[key];
            const isNotQuoted = !(val.startsWith("\"") && val.endsWith("\""));
            if (isNotQuoted && !val.match(containsNumberRegex)) {
                unit.SLKUnit.UnitBalance[key] = "\"" + val + "\"";
            }
        });
        Object.keys(unit.SLKUnit.UnitAbilities).forEach(key => {
            const val = unit.SLKUnit.UnitAbilities[key];
            const isNotQuoted = !(val.startsWith("\"") && val.endsWith("\""));
            if (isNotQuoted && !val.match(containsNumberRegex)) {
                unit.SLKUnit.UnitAbilities[key] = "\"" + val + "\"";
            }
        });
        unit.UnitFunc.Ubertip = "\"" + unit.UnitFunc.Ubertip + "\"";

        unit.SLKUnit.UnitAbilities.SortAbil = "\"z3\"";

        unit.SLKUnit.UnitBalance.SortBalance = "\"z3\"";
        unit.SLKUnit.UnitBalance.Sort2 = "\"zzm\"";
        if (!unit.SLKUnit.UnitBalance.Type) {
            unit.SLKUnit.UnitBalance.Type = "\"_\"";
        }
        unit.SLKUnit.UnitBalance.RealHP = unit.SLKUnit.UnitBalance.HP;
        if (!unit.SLKUnit.UnitBalance.Def) {
            unit.SLKUnit.UnitBalance.Def = "\"0\"";
        }
        unit.SLKUnit.UnitBalance.Realdef = unit.SLKUnit.UnitBalance.Def;
        if (!unit.SLKUnit.UnitBalance.STR) {
            unit.SLKUnit.UnitBalance.STR = "\"-\"";
        }
        if (!unit.SLKUnit.UnitBalance.AGI) {
            unit.SLKUnit.UnitBalance.AGI = "\"-\"";
        }
        if (!unit.SLKUnit.UnitBalance.INT) {
            unit.SLKUnit.UnitBalance.INT = "\"-\"";
        }
        unit.SLKUnit.UnitBalance.AbilTest = "\"-\"";
        if (!unit.SLKUnit.UnitBalance.Primary) {
            unit.SLKUnit.UnitBalance.Primary = "\"_\"";
        }
        if (!unit.SLKUnit.UnitBalance.Upgrades) {
            unit.SLKUnit.UnitBalance.Upgrades = "\"_\"";
        }
        unit.SLKUnit.UnitBalance.Nbrandom = "\"_\"";
        unit.SLKUnit.UnitBalance.InBeta= "0";

        unit.SLKUnit.UnitData.Sort = "\"z3\"";
        if (!unit.SLKUnit.UnitData.Threat) {
            unit.SLKUnit.UnitData.Threat = "1";
        }
        if (!unit.SLKUnit.UnitData.Valid) {
            unit.SLKUnit.UnitData.Valid = "1";
        }
        if (!unit.SLKUnit.UnitData.TargType) {
            unit.SLKUnit.UnitData.TargType = "\"_\"";
        }
        unit.SLKUnit.UnitData.FatLOS = "0";
        unit.SLKUnit.UnitData.BuffType = "\"_\"";
        unit.SLKUnit.UnitData.BuffRadius = "\"-\"";
        unit.SLKUnit.UnitData.NameCount = "\"-\"";
        if (!unit.SLKUnit.UnitData.RequireWaterRadius) {
            unit.SLKUnit.UnitData.RequireWaterRadius = "0";
        }
        if (!unit.SLKUnit.UnitData.RequireWaterRadius) {
            unit.SLKUnit.UnitData.RequireWaterRadius = "0";
        }
        if (!unit.SLKUnit.UnitData.RequireWaterRadius) {
            unit.SLKUnit.UnitData.RequireWaterRadius = "0";
        }
        if (!unit.SLKUnit.UnitData.RequireWaterRadius) {
            unit.SLKUnit.UnitData.RequireWaterRadius = "0";
        }
        unit.SLKUnit.UnitData.InBeta = "0";
        unit.SLKUnit.UnitData.Version = "1";

        unit.SLKUnit.UnitUI.SortUI = "\"z3\"";
        unit.SLKUnit.UnitUI.TilesetSpecific = "0";
        unit.SLKUnit.UnitUI.Name = "-";
        unit.SLKUnit.UnitUI.Campaign = "1";
        unit.SLKUnit.UnitUI.InEditor = "1";
        unit.SLKUnit.UnitUI.HiddenInEditor = "0";
        unit.SLKUnit.UnitUI.HostilePal = "0";
        unit.SLKUnit.UnitUI.DropItems = "1";
        unit.SLKUnit.UnitUI.NbmmIcon = "1";
        unit.SLKUnit.UnitUI.UseClickHelper = "0";
        unit.SLKUnit.UnitUI.HideHeroBar = "0";
        unit.SLKUnit.UnitUI.HideHeroMinimap = "0";
        unit.SLKUnit.UnitUI.HideHeroDeathMsg = "0";
        unit.SLKUnit.UnitUI.Weap1 = "\"_\"";
        unit.SLKUnit.UnitUI.Weap2 = "\"_\"";
        unit.SLKUnit.UnitUI.InBeta = "0";

        unit.SLKUnit.UnitWeapons.SortWeap = "\"n2\"";
        unit.SLKUnit.UnitWeapons.Sort2 = "\"zzm\"";
        unit.SLKUnit.UnitWeapons.RngTst = "\"-\"";
        unit.SLKUnit.UnitWeapons.Mincool1 = "\"-\"";
        unit.SLKUnit.UnitWeapons.Mindmg1 = "0";
        unit.SLKUnit.UnitWeapons.Mindmg2 = "0";
        unit.SLKUnit.UnitWeapons.Avgdmg1 = "0";
        unit.SLKUnit.UnitWeapons.Avgdmg2 = "0";
        unit.SLKUnit.UnitWeapons.Maxdmg1 = "0";
        unit.SLKUnit.UnitWeapons.Maxdmg2 = "0";
        if (!unit.SLKUnit.UnitWeapons.Targs1) {
            unit.SLKUnit.UnitWeapons.Targs1 = "\"-\"";
        }
        if (!unit.SLKUnit.UnitWeapons.Targs2) {
            unit.SLKUnit.UnitWeapons.Targs2 = "\"-\"";
        }
        if (!unit.SLKUnit.UnitWeapons.DmgUp1) {
            unit.SLKUnit.UnitWeapons.DmgUp1 = "\"-\"";
        }
        if (!unit.SLKUnit.UnitWeapons.DmgUp2) {
            unit.SLKUnit.UnitWeapons.DmgUp2 = "\"-\"";
        }
        if (!unit.SLKUnit.UnitWeapons.Hfact1) {
            unit.SLKUnit.UnitWeapons.Hfact1 = "\"-\"";
        }
        if (!unit.SLKUnit.UnitWeapons.Hfact2) {
            unit.SLKUnit.UnitWeapons.Hfact2 = "\"-\"";
        }
        if (!unit.SLKUnit.UnitWeapons.Qfact1) {
            unit.SLKUnit.UnitWeapons.Qfact1 = "\"-\"";
        }
        if (!unit.SLKUnit.UnitWeapons.Qfact2) {
            unit.SLKUnit.UnitWeapons.Qfact2 = "\"-\"";
        }
        if (!unit.SLKUnit.UnitWeapons.SplashTargs1) {
            unit.SLKUnit.UnitWeapons.SplashTargs1 = "\"_\"";
        }
        if (!unit.SLKUnit.UnitWeapons.SplashTargs2) {
            unit.SLKUnit.UnitWeapons.SplashTargs2 = "\"_\"";
        }
        if (!unit.SLKUnit.UnitWeapons.DmgUpg) {
            unit.SLKUnit.UnitWeapons.DmgUpg = "\"-\"";
        }
        unit.SLKUnit.UnitWeapons.InBeta = "0";
        unit.SLKUnit.UnitWeapons.RngTst2 = "\"-\"";

        const message = {name: "saveUnit", payload: unit};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            let oldUnit = null;
            for(let i = 0; i < unitDataList.length; i++) {
                if (unitDataList[i].UnitID === unitId) {
                    oldUnit = i;
                    break;
                }
            }

            if (oldUnit) {
                unitDataList[oldUnit] = { UnitID: unitId, Name: unit.UnitFunc.Name };
            } else {
                unitDataList.push({ UnitID: unitId, Name: unit.UnitFunc.Name });
            }

            index.search(document.getElementById("searchInput"));
        });
    },
    loadConfig: function () {
        const message = {name: "loadConfig", payload: null};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            const input = $("#outputFolderInput");
            if (input.length > 0) {
                input[0].value = message.payload.OutDir;
            }
        });
    },
    disableInputs: function (bool) {
        const message = {name: "getDisabledInputs", payload: null};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            message.payload.forEach(disabledInputId => {
                const elem = document.getElementById(disabledInputId);
                if (elem) {
                    elem.parentNode.hidden = bool;
                } else {
                    console.warn("Can't find the", disabledInputId,"element");
                }
            });
        });
    },
    generateUnitId: function () {
        const message = {name: "generateUnitId", payload: null};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            document.getElementById("UnitFunc-UnitId").value = message.payload;
        });
    }
};