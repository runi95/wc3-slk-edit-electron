let unitDataList = [];
let isLocked = false;
let selectedUnitId = null;
const mdxModels = {};
const unitModelNameToPath = {};

const addUnitTableData = (unitTableBody, unitData) => {
    const tr = document.createElement("tr");
    const th = document.createElement("th");
    const td = document.createElement("td");
    th.setAttribute("scope", "row");
    th.appendChild(document.createTextNode(unitData.UnitID));
    td.appendChild(document.createTextNode(unitData.Name));
    if (unitData.EditorSuffix) {
        const span = document.createElement("span");
        span.setAttribute("class", "text-secondary");
        span.appendChild(document.createTextNode(" " + unitData.EditorSuffix));
        td.appendChild(span);
    }
    tr.id = unitData.UnitID;
    tr.onclick = () => index.selectUnit(tr);
    tr.appendChild(th);
    tr.appendChild(td);
    unitTableBody.appendChild(tr);
};

const index = {
    multiColorTextarea: function (input) {
        const regex = new RegExp("\\|C([0-9A-F]{8})((?:(?!\\|C).)*)\\|R", "i");
        let result = input.value.split("|n").join("<br>").split("\n").join("<br>");
        let exec = regex.exec(result);
        while (exec !== null) {
            const index = exec.index;
            const color = "rgba(" + parseInt(exec[1].substr(2, 2), 16) + ", " + parseInt(exec[1].substr(4, 2), 16) + ", " + parseInt(exec[1].substr(6, 2), 16) + ", " + (parseInt(exec[1].substr(0, 2), 16) / 255) + ")";
            result = result.substr(0, index) + `<span style="color: ${color}">` + result.substr(index + 2 + exec[1].length, exec[2].length) + "</span>" + result.substr(index + 4 + exec[1].length + exec[2].length);
            exec = regex.exec(result);
        }

        document.getElementById("preview").innerHTML = result;
    },
    multiColorTextareaScroll: function (input) {
        document.getElementById("preview").scrollTop = input.scrollTop;
    },
    loadMdxModel: function (path) {
        if (mdxModels.hasOwnProperty(path)) {
            return {src: mdxModels[path], fetch: false};
        } else {
            const fetchPromise = new Promise((resolve, reject) => {
                astilectron.sendMessage({name: "fetchMdxModel", payload: path}, function (message) {
                    // Check for errors
                    if (message.name === "error") {
                        // asticode.notifier.error(message.payload);
                        reject(message.payload);
                    } else {
                        const binary_string = window.atob(message.payload);
                        const len = binary_string.length;
                        const bytes = new Uint8Array(len);
                        for (let i = 0; i < len; i++) {
                            bytes[i] = binary_string.charCodeAt(i);
                        }

                        const buf = bytes.buffer;
                        mdxModels[path] = buf;
                        resolve(buf);
                    }
                })
            });
            return {src: fetchPromise, fetch: true};
        }
    },
    init: function () {
        // Init
        asticode.loader.init();
        asticode.notifier.init();

        // Wait for astilectron to be ready
        document.addEventListener('astilectron-ready', function () {
            console.log("Initializing...");

            // Listen
            index.listen();

            index.initializeConfig();
        })
    },
    listen: function () {
        astilectron.onMessage(function (message) {
            switch (message.Name) {
                case "downloadStart":
                    document.getElementById("loadingwindow").hidden = true;
                    document.getElementById("downloadwindow").hidden = false;
                    break;
                case "downloadPercentUpdate":
                    document.getElementById("downloadbar").setAttribute("style", "width: " + message.Payload + "%");
                    break;
                case "downloadTextUpdate":
                    document.getElementById("downloadtext").innerText = message.Payload;
                    break;
            }
        });
    },
    removeUnit: function () {
        if (selectedUnitId === null)
            return;

        const unitMessage = {name: "removeUnit", payload: selectedUnitId};
        astilectron.sendMessage(unitMessage, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            selectedUnitId = null;
            unitDataList = unitDataList.filter(unit => unit.UnitID !== message.payload);
            index.search(document.getElementById("searchInput"));
        })
    },
    loadUnitData: function () {
        const message = {name: "loadUnitData", payload: null};
        astilectron.sendMessage(message, function (message) {
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

            document.getElementById("inputselectwindow").hidden = true;
            document.getElementById("outputselectwindow").hidden = true;
            document.getElementById("downloadwindow").hidden = true;
            document.getElementById("loadingwindow").hidden = true;
            document.getElementById("mainwindow").hidden = false;
        });
    },
    search: function (inputField) {
        const regex = new RegExp(inputField.value, "i");
        const filteredUnitDataList = unitDataList.filter(unitData => (unitData.Name + unitData.UnitID + unitData.EditorSuffix).match(regex));
        const unitTableBody = $("#unitTableBody")[0];
        unitTableBody.innerHTML = '';

        filteredUnitDataList.forEach(unitData => {
            addUnitTableData(unitTableBody, unitData);
        });
    },
    selectUnit: function (unitTableRow) {
        const unitId = unitTableRow.id;
        selectedUnitId = unitId;
        const message = {name: "selectUnit", payload: unitId};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            $("#unitTableBody>tr").removeClass("active");
            unitTableRow.setAttribute("class", "active");
            Object.keys(message.payload.SLKUnit).forEach(slkUnitKey => {
                Object.keys(message.payload.SLKUnit[slkUnitKey]).forEach(key => {
                    const rawValue = message.payload.SLKUnit[slkUnitKey][key] ? message.payload.SLKUnit[slkUnitKey][key] : "";
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
                            const valueSplit = value.split(",");
                            const valueLower = valueSplit.map(val => val.toLowerCase());
                            for (let i = 0; i < childInputs.length; i++) {
                                if (valueLower.includes(childInputs[i].value.toLowerCase())) {
                                    childInputs[i].checked = true;
                                } else {
                                    childInputs[i].checked = false;
                                }
                            }
                        }
                    }
                })
            });

            if (message.payload.UnitFunc.Ubertip) {
                const rawValue = message.payload.UnitFunc.Ubertip;
                const trimmedRight = rawValue.endsWith("\"") ? rawValue.substr(0, rawValue.length - 1) : rawValue;
                const trimmedLeft = trimmedRight.startsWith("\"") ? trimmedRight.substr(1) : trimmedRight;
                const value = trimmedLeft.replace(new RegExp("\\|n", "g"), "\n");

                message.payload.UnitFunc.Ubertip = value;
            }

            Object.keys(message.payload.UnitFunc).forEach(unitFuncKey => {
                const elemList = $("#UnitFunc-" + unitFuncKey);
                if (elemList.length > 0) {
                    if (elemList[0] instanceof HTMLInputElement || elemList[0] instanceof HTMLTextAreaElement) {
                        const type = elemList[0].type;
                        if (type === "text" || type === "textarea" || type === "select-one") {
                            elemList[0].value = message.payload.UnitFunc[unitFuncKey];
                        } else if (type === "checkbox") {
                            elemList[0].checked = message.payload.UnitFunc[unitFuncKey] === "1";
                        } else {
                            console.log("type is " + elemList[0].type);
                        }
                    }
                }
            });

            index.multiColorTextarea(document.getElementById("UnitFunc-Ubertip"));
            index.loadIcon(document.getElementById("UnitFunc-Art"));
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
        const hiddenInput = document.getElementById("hiddenFileUploadInput");
        const input = document.getElementById("outputFolderInput");

        if (!hiddenInput.files || hiddenInput.files.length < 1)
            return;

        const updatedValue = hiddenInput.files[0].path;

        if (!updatedValue)
            return;

        input.value = updatedValue;
    },
    saveFieldToUnit: function (input) {
        if (!document.getElementById("unitId-form").checkValidity())
            return;

        let Field = null;
        let Value = null;

        if (input.classList.contains("sub-multi-check")) {
            Field = input.parentNode.parentNode.parentNode.id;
            input.parentNode.parentNode.childNodes.forEach(child => {
                if (child instanceof HTMLLIElement) {
                    child.childNodes.forEach(listChild => {
                        if (listChild instanceof HTMLInputElement) {
                            if (listChild.checked) {
                                if (Value === null) {
                                    Value = "\"" + listChild.value;
                                } else {
                                    Value += "," + listChild.value;
                                }
                            }
                        }
                    });
                }
            });

            if (Value === null) {
                Value = "_";
            } else {
                Value += "\"";
            }
        } else {
            const type = input.type;
            Field = input.id;
            fieldSplit = Field.split("-");
            if (type === "text" || type === "textarea" || type === "select-one") {
                const containsNumberRegex = new RegExp("^-?(?:(?:\\d*)|(?:\\d+\\.\\d+))$");
                Value = input.value.replace(new RegExp("\n", "g"), "|n");

                if (fieldSplit[0] === "SLKUnit" || Field === "UnitFunc-Ubertip") {
                    if (!Value.match(containsNumberRegex)) {
                        Value = "\"" + Value + "\"";
                    }
                }
            } else if (type === "checkbox") {
                Value = input.checked ? "1" : "0";
            }
        }

        const UnitId = document.getElementById("UnitFunc-UnitId").value;
        const fieldToUnit = {Field, Value, UnitId};
        if (Field != null && Value != null) {
            const message = {name: "saveFieldToUnit", payload: fieldToUnit};
            astilectron.sendMessage(message, function (message) {
                // Check for errors
                if (message.name === "error") {
                    asticode.notifier.error(message.payload);
                    return;
                }

                if (message.payload === "unsaved") {
                    index.saveUnit();
                }
            });
        }
    },
    loadIcon: function (input) {
        const message = {name: "loadIcon", payload: input.value};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            if (message.payload !== null && message.payload !== "") {
                document.getElementById("iconImage").setAttribute("src", "data:image/png;base64," + message.payload);
            } else {
                document.getElementById("iconImage").setAttribute("src", "emptyicon.png");
            }
        });
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
                } else if (inputs[i].value) {
                    if (idSplit[0] === "SLKUnit") {
                        unit.SLKUnit[idSplit[1]][idSplit[2]] = inputs[i].value
                    } else if (idSplit[0] === "UnitFunc") {
                        unit.UnitFunc[idSplit[1]] = inputs[i].value
                    }
                }
            } else if (type === "checkbox" && inputs[i].checked) {
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

        const containsNumberRegex = new RegExp("^-?(?:(?:\\d*)|(?:\\d+\\.\\d+))$");
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
        if (unit.UnitFunc.Ubertip) {
            unit.UnitFunc.Ubertip = "\"" + unit.UnitFunc.Ubertip + "\"";
        }

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
        unit.SLKUnit.UnitBalance.InBeta = "0";

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
            for (let i = 0; i < unitDataList.length; i++) {
                if (unitDataList[i].UnitID === unitId) {
                    oldUnit = i;
                    break;
                }
            }

            if (oldUnit) {
                unitDataList[oldUnit] = {UnitID: unitId, Name: unit.UnitFunc.Name};
            } else {
                unitDataList.push({UnitID: unitId, Name: unit.UnitFunc.Name});
            }

            index.search(document.getElementById("searchInput"));
        });
    },
    initializeConfig: function () {
        const message = {name: "initializeConfig", payload: null};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            index.initializeIcons();
        });
    },
    initializeIcons: function () {
        const message = {name: "initializeIcons", payload: null};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            const dataList = document.getElementById("IconList");
            let options = "";
            message.payload.forEach(imagePath => {
                /*
                const option = document.createElement("option");
                option.setAttribute("value", imagePath);
                dataList.appendChild(option);
                */
                options += '<option value="' + imagePath + '" />';
            });
            dataList.innerHTML = options;

            index.loadMdx();
        });
    },
    changeMdxModel: function (input) {
        loadMdxModel(input.value);
    },
    loadMdx: function () {
        const message = {name: "loadMdx", payload: null};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            let options = "";
            message.payload.forEach(unitModel => {
                unitModelNameToPath[unitModel.Name] = unitModel.Path
                options += '<option value="' + unitModel.Path + '">' + unitModel.Name + '</option>';
            });

            document.getElementById("model-selector").innerHTML = options;

            index.loadConfig();
        });
    },
    selectMdxModel: function () {
        const modelFileInput = document.getElementById("SLKUnit-UnitUI-File");
        modelFileInput.value = document.getElementById("model-selector").value;

        $('#unit-model-modal').modal('toggle');
        index.saveFieldToUnit(modelFileInput);
    },
    loadConfig: function () {
        const message = {name: "loadConfig", payload: null};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            if (message.payload !== null) {
                document.getElementById("outputFolderInput").value = message.payload.OutDir;
                index.disableInputs(message.payload.IsLocked);
                index.startMainWindow();
            } else {
                document.getElementById("loadingwindow").hidden = true;
                document.getElementById("downloadwindow").hidden = true;
                document.getElementById("outputselectwindow").hidden = true;
                document.getElementById("mainwindow").hidden = true;
                document.getElementById("inputselectwindow").hidden = false;
            }
        });
    },
    disableInputs: function (bool) {
        if (bool === isLocked)
            return;

        const message = {name: "getDisabledInputs", payload: bool};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            if (message.payload === null)
                return;

            message.payload.forEach(disabledInputId => {
                const elem = document.getElementById(disabledInputId);
                if (elem) {
                    elem.parentNode.hidden = bool;
                } else {
                    console.warn("Can't find the", disabledInputId, "element");
                }
            });

            if (bool) {
                document.getElementById("modeLock").classList.replace("fa-unlock", "fa-lock");
            } else {
                document.getElementById("modeLock").classList.replace("fa-lock", "fa-unlock");
            }
            isLocked = bool;
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
    },
    generateUnitTooltip: function () {
        const attacksEnabled = document.getElementById("SLKUnit-UnitWeapons-WeapsOn").value;
        let value = "";
        if (attacksEnabled === "1" || attacksEnabled === "3") {
            value += "|cffffcc00Attack:|r " + document.getElementById("SLKUnit-UnitWeapons-AtkType1").value.charAt(0).toUpperCase() + document.getElementById("SLKUnit-UnitWeapons-AtkType1").value.substr(1) + "|n";
            value += "|cffffcc00Cooldown:|r " + document.getElementById("SLKUnit-UnitWeapons-Cool1").value + "|n";
            const baseDamage = parseInt(document.getElementById("SLKUnit-UnitWeapons-Dmgplus1").value, 10);
            const damageNumberOfDice = parseInt(document.getElementById("SLKUnit-UnitWeapons-Dice1").value, 10);
            const damageSidesPerDie = parseInt(document.getElementById("SLKUnit-UnitWeapons-Sides1").value, 10);
            value += "|cffffcc00Damage:|r " + (baseDamage + damageNumberOfDice) + " - " + (baseDamage + damageNumberOfDice * damageSidesPerDie) + "|n";
            value += "|cffffcc00Range:|r " + document.getElementById("SLKUnit-UnitWeapons-RangeN1").value + "|n";
        } else if (attacksEnabled === "2") {
            value += "|cffffcc00Attack:|r " + document.getElementById("SLKUnit-UnitWeapons-AtkType2").value.charAt(0).toUpperCase() + "|n";
            value += "|cffffcc00Cooldown:|r " + document.getElementById("SLKUnit-UnitWeapons-Cool2").value + "|n";
            const baseDamage = parseInt(document.getElementById("SLKUnit-UnitWeapons-Dmgplus2").value, 10);
            const damageNumberOfDice = parseInt(document.getElementById("SLKUnit-UnitWeapons-Dice2").value, 10);
            const damageSidesPerDie = parseInt(document.getElementById("SLKUnit-UnitWeapons-Sides2").value, 10);
            value += "|cffffcc00Damage:|r " + (baseDamage + damageNumberOfDice) + " - " + (baseDamage + damageNumberOfDice * damageSidesPerDie) + "|n";
            value += "|cffffcc00Range:|r " + document.getElementById("SLKUnit-UnitWeapons-RangeN2").value + "|n";
        } else if (attacksEnabled === "0") {
            value += "|cffffcc00Attack:|r None|n";
            value += "|cffffcc00Range:|r " + document.getElementById("SLKUnit-UnitWeapons-RangeN1").value + "|n";
        }

        if (attacksEnabled === "3") {
            value += "|cffffcc00Attack(2):|r " + document.getElementById("SLKUnit-UnitWeapons-AtkType2").value.charAt(0).toUpperCase() + document.getElementById("SLKUnit-UnitWeapons-AtkType2").value.substr(1) + "|n";
            value += "|cffffcc00Cooldown(2):|r " + document.getElementById("SLKUnit-UnitWeapons-Cool2").value + "|n";
            const baseDamage = parseInt(document.getElementById("SLKUnit-UnitWeapons-Dmgplus2").value, 10);
            const damageNumberOfDice = parseInt(document.getElementById("SLKUnit-UnitWeapons-Dice2").value, 10);
            const damageSidesPerDie = parseInt(document.getElementById("SLKUnit-UnitWeapons-Sides2").value, 10);
            value += "|cffffcc00Damage(2):|r " + (baseDamage + damageNumberOfDice) + " - " + (baseDamage + damageNumberOfDice * damageSidesPerDie) + "|n";
            value += "|cffffcc00Range(2):|r " + document.getElementById("SLKUnit-UnitWeapons-RangeN2").value + "|n";
        }

        document.getElementById("UnitFunc-Ubertip").value = value;
        index.multiColorTextarea(document.getElementById("UnitFunc-Ubertip"));
    },
    activateFileUploadButtonLoadInput: function () {
        $("#hiddenFileUploadInputLoadInput").click();
    },
    activateFileUploadButtonLoadOutput: function () {
        $("#hiddenFileUploadInputLoadOutput").click();
    },
    updateInputConfigPath: function () {
        const hiddenInput = document.getElementById("hiddenFileUploadInputLoadInput");
        const input = document.getElementById("configInput");

        if (!hiddenInput.files || hiddenInput.files.length < 1)
            return;

        const updatedValue = hiddenInput.files[0].path;

        if (!updatedValue)
            return;

        input.value = updatedValue;
    },
    updateOutputConfigPath: function () {
        const hiddenInput = document.getElementById("hiddenFileUploadInputLoadOutput");
        const input = document.getElementById("configOutput");

        if (!hiddenInput.files || hiddenInput.files.length < 1)
            return;

        const updatedValue = hiddenInput.files[0].path;

        if (!updatedValue)
            return;

        input.value = updatedValue;
    },
    submitConfigurationDifferent: function () {
        const inDir = document.getElementById("configInput").value;
        const outDir = document.getElementById("configOutput").value;

        index.submitConfiguration(inDir, outDir);
    },
    submitConfigurationSame: function () {
        const inDir = document.getElementById("configInput").value;
        const outDir = inDir;

        index.submitConfiguration(inDir, outDir);
    },
    submitConfiguration: function (inDir, outDir) {
        const message = {name: "setConfig", payload: {InDir: inDir, OutDir: outDir, IsLocked: false}};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            index.initializeIcons();
        });
    },
    newOutputFolder: function () {
        document.getElementById("inputselectwindow").hidden = true;
        document.getElementById("outputselectwindow").hidden = false;
    },
    startMainWindow: function () {
        document.getElementById("inputselectwindow").hidden = true;
        document.getElementById("outputselectwindow").hidden = true;
        document.getElementById("mainwindow").hidden = true;
        document.getElementById("downloadwindow").hidden = true;
        document.getElementById("loadingwindow").hidden = false;

        // initMdx();
        index.activateHotkeys();
        index.loadUnitData();
    },
    activateHotkeys: function () {
        const message = {name: "getOperatingSystem", payload: null};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            if (message.payload === "darwin") {
                document.onkeydown = function (e) {
                    if (e.metaKey && e.key === "s") {
                        index.saveUnit();
                    } else if (e.metaKey && e.key === "f") {
                        document.getElementById("searchInput").focus();
                    }
                }
            } else {
                document.onkeydown = function (e) {
                    if (e.ctrlKey && e.key === "s") {
                        index.saveUnit();
                    } else if (e.ctrlKey && e.key === "f") {
                        document.getElementById("searchInput").focus();
                    }
                }
            }
        });
    }
};