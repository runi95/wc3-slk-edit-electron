let unitDataList = [];
let itemDataList = [];
let isLocked = false;
let isRegexSearch = false;
let isItemRegexSearch = false;
let selectedUnitId = null;
let selectedItemId = null;
let isUnsaved = false;
let sortUnitNameState = 0;
let sortUnitIdState = 0;
let sortItemNameState = 0;
let sortItemIdState = 0;
let iconMode;
let modelMode;
const mdxModels = {};
const unitModelNameToPath = {};
const unitModelPathToName = {};
const unitIconNameToPath = {};
const unitIconPathToName = {};

const sortNameAlphabetical = (a, b) => {
    return a.Name > b.Name ? 1 : -1;
};

const sortNameInverse = (a, b) => {
    return b.Name > a.Name ? 1 : -1;
};

const sortUnitIdAlphabetical = (a, b) => {
    return a.UnitID > b.UnitID ? 1 : -1;
};

const sortUnitIdInverse = (a, b) => {
    return b.UnitID > a.UnitID ? 1 : -1;
};

const sortItemIdAlphabetical = (a, b) => {
    return a.ItemID > b.ItemID ? 1 : -1;
};

const sortItemIdInverse = (a, b) => {
    return b.ItemID > a.ItemID ? 1 : -1;
};

const addTableData = (tableBody, onclickFunc, dataList) => {
    let trList = "";
    dataList.forEach(data => {
        let str = '<tr id="' + data.Id + '" onclick="index.' + onclickFunc + '(this)"><th scope="row">' + data.Id + '</th><td>' + data.Name;
        if (data.EditorSuffix) {
            str += '<span class="text-secondary"> ' + data.EditorSuffix + '</span>';
        }
        str += '</td></tr>';
        trList += str;
    });

    tableBody.innerHTML = trList;
};

const index = {
    multiColorTextarea: function (target, input) {
        const regex = new RegExp("\\|C([0-9A-F]{8})((?:(?!\\|C).)*)\\|R", "i");
        let result = input.value.split("|n").join("<br>").split("\n").join("<br>");
        let exec = regex.exec(result);
        while (exec !== null) {
            const index = exec.index;
            const color = "rgba(" + parseInt(exec[1].substr(2, 2), 16) + ", " + parseInt(exec[1].substr(4, 2), 16) + ", " + parseInt(exec[1].substr(6, 2), 16) + ", " + (parseInt(exec[1].substr(0, 2), 16) / 255) + ")";
            result = result.substr(0, index) + `<span style="color: ${color}">` + result.substr(index + 2 + exec[1].length, exec[2].length) + "</span>" + result.substr(index + 4 + exec[1].length + exec[2].length);
            exec = regex.exec(result);
        }

        document.getElementById(target).innerHTML = result;
    },
    multiColorTextareaScroll: function (target, input) {
        document.getElementById(target).scrollTop = input.scrollTop;
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
    loadMdxModal: function (modelInputId, mode) {
        modelMode = mode;
        initMdx();

        const currentModelPath = document.getElementById(modelInputId).value;
        const lowercaseModelPath = currentModelPath.toLowerCase().replace(new RegExp("\.mdl$"), ".mdx");
        const modelPathWithExtension = lowercaseModelPath.endsWith("mdx") ? lowercaseModelPath : lowercaseModelPath + ".mdx";
        if (!unitModelPathToName.hasOwnProperty(modelPathWithExtension)) {
            return;
        }

        $("#model-selector").typeahead("val", unitModelPathToName[modelPathWithExtension]);
        loadMdxModel(modelPathWithExtension);
    },
    loadIconModal: function (mode, artId) {
        iconMode = mode;
        const currentIconPath = document.getElementById(artId).value;
        if (!unitIconPathToName.hasOwnProperty(currentIconPath.toLowerCase())) {
            return;

        }
        $("#icon-selector").typeahead("val", unitIconPathToName[currentIconPath.toLowerCase()]);
        index.loadModalIcon(currentIconPath);
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

            index.loadConfig();
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
                case "crash":
                    astilectron.showErrorBox("Crash", message.Payload);
                    astilectron.sendMessage({name: "closeWindow", payload: null}, function (message) {
                    });
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
            unitDataList = unitDataList.filter(unit => unit.Id !== message.payload);
            index.unitSearch(document.getElementById("unitSearchInput"));
        })
    },
    removeItem: function () {
        if (selectedItemId === null)
            return;

        const message = {name: "removeItem", payload: selectedItemId};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            selectedItemId = null;
            itemDataList = itemDataList.filter(item => item.Id !== message.payload);
            index.itemSearch(document.getElementById("itemSearchInput"));
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
            const unitTableBody = document.getElementById("unitTableBody");
            addTableData(unitTableBody, "selectUnit", message.payload);
        });
    },
    loadItemData: function () {
        const message = {name: "loadItemData", payload: null};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            itemDataList = message.payload;
            const itemTableBody = document.getElementById("itemTableBody");
            addTableData(itemTableBody, "selectItem", message.payload);
        });
    },
    loadSlk: function () {
        const message = {name: "loadSlk", payload: null};
        astilectron.sendMessage(message, function (message) {
            // Check for errors|
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            let fileInfoContainerString = '<ul style="list-style: none; padding: 0;">';
            let i = 0;
            message.payload.forEach(fileInfo => {
                if (i === Math.floor(message.payload.length / 2)) {
                    fileInfoContainerString += '</ul><ul style="list-style: none; padding: 0;">';
                }

                fileInfoContainerString += '<li><span class="' + fileInfo.StatusClass + '">' + '<i class="fas ' + fileInfo.StatusIconClass + '"></i> ' + fileInfo.FileName + '</span></li>';
                i++;
            });

            fileInfoContainerString += '</ul>';
            document.getElementById("file-info-container").innerHTML = fileInfoContainerString;

            index.loadUnitData();
            index.loadItemData();
        });
    },
    unitSearch: function (inputField) {
        let filteredUnitDataList;
        if (isRegexSearch) {
            const regex = new RegExp(inputField.value, "i");
            filteredUnitDataList = unitDataList.filter(unitData => (unitData.Name + unitData.Id + unitData.EditorSuffix).match(regex));
        } else {
            filteredUnitDataList = unitDataList.filter(unitData => (unitData.Name + unitData.Id + unitData.EditorSuffix).includes(inputField.value));
        }

        const unitTableBody = document.getElementById("unitTableBody");
        addTableData(unitTableBody, "selectUnit", filteredUnitDataList);
    },
    itemSearch: function (inputField) {
        let filteredItemDataList;
        if (isItemRegexSearch) {
            const regex = new RegExp(inputField.value, "i");
            filteredItemDataList = itemDataList.filter(itemData => (itemData.Name + itemData.Id + itemData.EditorSuffix).match(regex));
        } else {
            filteredItemDataList = itemDataList.filter(itemData => (itemData.Name + itemData.Id + itemData.EditorSuffix).includes(inputField.value));
        }

        const itemTableBody = document.getElementById("itemTableBody");
        addTableData(itemTableBody, "selectItem", filteredItemDataList);
    },
    selectUnit: function (unitTableRow) {
        index.selectUnitFromId(unitTableRow.id);
    },
    selectUnitFromId: function (unitId) {
        selectedUnitId = unitId;
        const message = {name: "selectUnit", payload: unitId};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            $("#unitTableBody>tr").removeClass("active");
            document.getElementById(unitId).setAttribute("class", "active");

            if (message.payload.Ubertip) {
                const rawValue = message.payload.Ubertip;
                const trimmedRight = rawValue.endsWith("\"") ? rawValue.substr(0, rawValue.length - 1) : rawValue;
                const trimmedLeft = trimmedRight.startsWith("\"") ? trimmedRight.substr(1) : trimmedRight;

                message.payload.Ubertip = trimmedLeft.replace(new RegExp("\\|n", "g"), "\n");
            }

            if (!message.payload.Buttonpos || message.payload.Buttonpos === "" || message.payload.Buttonpos === "_" || message.payload.Buttonpos === "-") {
                if (!message.payload.ButtonposX || message.payload.ButtonposX === "" || message.payload.ButtonposX === "_" || message.payload.ButtonposX === "-" || !message.payload.ButtonposY || message.payload.ButtonposY === "" || message.payload.ButtonposY === "_" || message.payload.ButtonposY === "-") {
                    message.payload.Buttonpos = "0,0";
                } else {
                    message.payload.Buttonpos = message.payload.ButtonposX + "," + message.payload.ButtonposY;
                }
            }

            Object.keys(message.payload).forEach(slkUnitKey => {
                const rawValue = message.payload[slkUnitKey] ? message.payload[slkUnitKey] : "";
                const trimmedRight = rawValue.endsWith("\"") ? rawValue.substr(0, rawValue.length - 1) : rawValue;
                const value = trimmedRight.startsWith("\"") ? trimmedRight.substr(1) : trimmedRight;
                const elem = document.getElementById("Unit-" + slkUnitKey);
                if (elem) {
                    if (elem instanceof HTMLInputElement || elem instanceof HTMLSelectElement || elem instanceof HTMLTextAreaElement) {
                        const type = elem.type;

                        if (type === "text" || type === "textarea" || type === "select-one") {
                            elem.value = value;
                        } else if (type === "checkbox") {
                            elem.checked = value === "1";
                        }
                    } else if (elem.id === "Unit-Buttonpos") {
                        const buttonpos = message.payload[slkUnitKey];
                        if (!buttonpos || buttonpos === "-" || buttonpos === "_") {
                            elem.value = "0,0"
                        } else {
                            elem.value = buttonpos;
                        }
                    } else if (elem.classList.contains("multi-check")) {
                        const childInputs = $("#Unit-" + slkUnitKey + " :input");
                        const valueSplit = value.split(",");
                        const valueLower = valueSplit.map(val => val.toLowerCase());
                        for (let i = 0; i < childInputs.length; i++) {
                            childInputs[i].checked = valueLower.includes(childInputs[i].value.toLowerCase());
                        }
                    }
                }
            });

            index.multiColorTextarea("unit-preview", document.getElementById("Unit-Ubertip"));
            index.loadIcon("unitIconImage", document.getElementById("Unit-Art"));
        });
    },
    selectItem: function (itemTableRow) {
        index.selectItemFromId(itemTableRow.id);
    },
    selectItemFromId: function (itemId) {
        selectedItemId = itemId;
        const message = {name: "selectItem", payload: itemId};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            $("#itemTableBody>tr").removeClass("active");
            document.getElementById(itemId).setAttribute("class", "active");

            if (message.payload.Ubertip) {
                const rawValue = message.payload.Ubertip;
                const trimmedRight = rawValue.endsWith("\"") ? rawValue.substr(0, rawValue.length - 1) : rawValue;
                const trimmedLeft = trimmedRight.startsWith("\"") ? trimmedRight.substr(1) : trimmedRight;

                message.payload.Ubertip = trimmedLeft.replace(new RegExp("\\|n", "g"), "\n");
            }

            if (!message.payload.Buttonpos || message.payload.Buttonpos === "" || message.payload.Buttonpos === "_" || message.payload.Buttonpos === "-") {
                if (!message.payload.ButtonposX || message.payload.ButtonposX === "" || message.payload.ButtonposX === "_" || message.payload.ButtonposX === "-" || !message.payload.ButtonposY || message.payload.ButtonposY === "" || message.payload.ButtonposY === "_" || message.payload.ButtonposY === "-") {
                    message.payload.Buttonpos = "0,0";
                } else {
                    message.payload.Buttonpos = message.payload.ButtonposX + "," + message.payload.ButtonposY;
                }
            }

            Object.keys(message.payload).forEach(slkItemKey => {
                const rawValue = message.payload[slkItemKey] ? message.payload[slkItemKey] : "";
                const trimmedRight = rawValue.endsWith("\"") ? rawValue.substr(0, rawValue.length - 1) : rawValue;
                const value = trimmedRight.startsWith("\"") ? trimmedRight.substr(1) : trimmedRight;
                const elem = document.getElementById("Item-" + slkItemKey);
                if (elem) {
                    if (elem instanceof HTMLInputElement || elem instanceof HTMLSelectElement || elem instanceof HTMLTextAreaElement) {
                        const type = elem.type;

                        if (type === "text" || type === "textarea" || type === "select-one") {
                            elem.value = value;
                        } else if (type === "checkbox") {
                            elem.checked = value === "1";
                        }
                    } else if (elem.id === "Item-Buttonpos") {
                        const buttonpos = message.payload[slkItemKey];
                        if (!buttonpos || buttonpos === "-" || buttonpos === "_") {
                            elem.value = "0,0"
                        } else {
                            elem.value = buttonpos;
                        }
                    } else if (elem.classList.contains("multi-check")) {
                        const childInputs = $("#Item-" + slkItemKey + " :input");
                        const valueSplit = value.split(",");
                        const valueLower = valueSplit.map(val => val.toLowerCase());
                        for (let i = 0; i < childInputs.length; i++) {
                            childInputs[i].checked = valueLower.includes(childInputs[i].value.toLowerCase());
                        }
                    }
                }
            });

            index.multiColorTextarea("item-preview", document.getElementById("Item-Ubertip"));
            index.loadIcon("itemIconImage", document.getElementById("Item-Art"));
        });
    },
    saveToFile: function () {
        document.getElementById("savingSpan").hidden = false;
        document.getElementById("savedSpan").hidden = true;
        document.getElementById("unsavedSpan").hidden = true;

        const message = {name: "saveToFile", payload: null};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                document.getElementById("unsavedSpan").hidden = false;
                document.getElementById("savedSpan").hidden = true;
                document.getElementById("savingSpan").hidden = true;

                asticode.notifier.error(message.payload);
                return;
            }

            if (isUnsaved) {
                isUnsaved = false;
            }

            document.getElementById("savedSpan").hidden = false;
            document.getElementById("unsavedSpan").hidden = true;
            document.getElementById("savingSpan").hidden = true;
        });
    },
    saveFieldToUnit: function (input) {
        index.saveField(input, "unitId-form", "Unit-UnitID", (Field, Value, UnitId) => {
            if (Field === "Unit-Name" || Field === "Unit-Editorsuffix") {
                let oldUnit = null;
                for (let i = 0; i < unitDataList.length; i++) {
                    if (unitDataList[i].Id === UnitId) {
                        oldUnit = i;
                        break;
                    }
                }

                if (oldUnit) {
                    const oldUnitData = unitDataList[oldUnit];

                    if (Field === "Unit-Name") {
                        oldUnitData.Name = Value;
                    } else if (Field === "Unit-Editorsuffix") {
                        oldUnitData.EditorSuffix = Value;
                    }

                    index.unitSearch(document.getElementById("unitSearchInput"));
                }
            }
        });
    },
    saveFieldToItem: function (input) {
        index.saveField(input, "itemID-form", "Item-ItemID");
    },
    saveField: function (input, idForm, idInput, savedCallback) {
        if (!document.getElementById(idForm).checkValidity())
            return;

        let field = null;
        let value = null;

        if (input.classList.contains("sub-multi-check")) {
            field = input.parentNode.parentNode.parentNode.id;
            input.parentNode.parentNode.parentNode.childNodes.forEach(child => {
                if (child instanceof HTMLUListElement) {
                    child.childNodes.forEach(listChild => {
                        if (listChild instanceof HTMLLIElement) {
                            listChild.childNodes.forEach(listElement => {
                                if (listElement instanceof HTMLInputElement) {
                                    if (listElement.checked) {
                                        if (value === null) {
                                            value = "\"" + listElement.value;
                                        } else {
                                            value += "," + listElement.value;
                                        }
                                    }
                                }
                            });
                        }
                    });
                }
            });

            if (value === null) {
                value = "\"_\"";
            } else {
                value += "\"";
            }
        } else {
            const type = input.type;
            field = input.id;
            if (type === "text" || type === "textarea" || type === "select-one") {
                const containsNumberRegex = new RegExp("^-?(?:(?:\\d*)|(?:\\d+\\.\\d+))$");
                value = input.value.replace(new RegExp("\n", "g"), "|n");

                if (!value.match(containsNumberRegex) && !value.startsWith("\"") && !value.endsWith("\"")) {
                    value = "\"" + value + "\"";
                }
            } else if (type === "checkbox") {
                value = input.checked ? "1" : "0";
            }
        }

        const id = document.getElementById(idInput).value;
        const fieldToSave = {Field: field, Value: value, Id: id};
        if (field != null && value != null) {
            const message = {name: "saveFieldToUnit", payload: fieldToSave};
            astilectron.sendMessage(message, function (message) {
                // Check for errors
                if (message.name === "error") {
                    asticode.notifier.error(message.payload);
                    return;
                }

                if (!isUnsaved) {
                    isUnsaved = true;
                    document.getElementById("savedSpan").hidden = true;
                    document.getElementById("unsavedSpan").hidden = false;
                }

                if (message.payload === "unsaved") {
                    index.saveUnit();
                } else {
                    if (typeof savedCallback === "function") {
                        savedCallback(field, value, id);
                    }
                }
            });
        }
    },
    changeModalIcon: function (input) {
        const inputValue = input.value.toLowerCase();
        if (unitIconNameToPath.hasOwnProperty(inputValue)) {
            index.loadModalIcon(unitIconNameToPath[inputValue]);
        }
    },
    loadModalIcon: function (path) {
        const message = {name: "loadIcon", payload: path};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            if (message.payload !== null && message.payload !== "") {
                document.getElementById("iconModal").setAttribute("src", "data:image/png;base64," + message.payload);
            } else {
                document.getElementById("iconModal").setAttribute("src", "emptyicon.png");
            }
        });
    },
    selectIcon: function () {
        const inputValue = document.getElementById("icon-selector").value.toLowerCase();
        if (unitIconNameToPath.hasOwnProperty(inputValue)) {
            const iconPath = unitIconNameToPath[inputValue];
            let iconInput;
            if (iconMode === "unit") {
                iconInput = document.getElementById("Unit-Art");
                index.saveFieldToUnit(iconInput);
                index.loadIconValue("unitIconImage", iconPath);
            } else if (iconMode === "item") {
                iconInput = document.getElementById("Item-Art");
                index.saveFieldToItem(iconInput);
                index.loadIconValue("itemIconImage", iconPath);
            }
            iconInput.value = iconPath;
            $('#icon-modal').modal('toggle');
        }
    },
    loadIcon: function (iconImage, input) {
        index.loadIconValue(iconImage, input.value);
    },
    loadIconValue: function (iconImage, iconPath) {
        const message = {name: "loadIcon", payload: iconPath};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            if (message.payload !== null && message.payload !== "") {
                document.getElementById(iconImage).setAttribute("src", "data:image/png;base64," + message.payload);
            } else {
                document.getElementById(iconImage).setAttribute("src", "emptyicon.png");
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

        const unit = {};
        const inputs = $(":input");
        for (let i = 0; i < inputs.length; i++) {
            const type = inputs[i].type;
            if (inputs[i].id) {
                const idSplit = inputs[i].id.split("-");
                if (type === "checkbox") {
                    unit[idSplit[1]] = inputs[i].checked ? "1" : "0"
                } else if (inputs[i].value) {
                    unit[idSplit[1]] = inputs[i].value
                }
            } else if (type === "checkbox" && inputs[i].checked) {
                const parentNode = inputs[i].parentNode.parentNode.parentNode;
                const parentIdSplit = parentNode.id.split("-");
                const oldValue = unit[parentIdSplit[1]];
                if (oldValue) {
                    unit[parentIdSplit[1]] += "," + inputs[i].value;
                } else {
                    unit[parentIdSplit[1]] = inputs[i].value;
                }
            }
        }

        const unitId = unit.UnitID;
        const quotedUnitId = "\"" + unitId + "\"";
        unit.UnitID = quotedUnitId;
        unit.UnitBalanceID = quotedUnitId;
        unit.UnitUIID = quotedUnitId;
        unit.UnitWeapID = quotedUnitId;
        unit.UnitAbilID = quotedUnitId;

        const containsNumberRegex = new RegExp("^-?(?:(?:\\d*)|(?:\\d+\\.\\d+))$");
        Object.keys(unit).forEach(key => {
            const val = unit[key];
            const isNotQuoted = !(val.startsWith("\"") && val.endsWith("\""));
            if (isNotQuoted && !val.match(containsNumberRegex)) {
                unit[key] = "\"" + val + "\"";
            }
        });
        Object.keys(unit).forEach(key => {
            const val = unit[key];
            const isNotQuoted = !(val.startsWith("\"") && val.endsWith("\""));
            if (isNotQuoted && !val.match(containsNumberRegex)) {
                unit[key] = "\"" + val + "\"";
            }
        });
        Object.keys(unit).forEach(key => {
            const val = unit[key];
            const isNotQuoted = !(val.startsWith("\"") && val.endsWith("\""));
            if (isNotQuoted && !val.match(containsNumberRegex)) {
                unit[key] = "\"" + val + "\"";
            }
        });
        Object.keys(unit).forEach(key => {
            const val = unit[key];
            const isNotQuoted = !(val.startsWith("\"") && val.endsWith("\""));
            if (isNotQuoted && !val.match(containsNumberRegex)) {
                unit[key] = "\"" + val + "\"";
            }
        });
        Object.keys(unit).forEach(key => {
            const val = unit[key];
            const isNotQuoted = !(val.startsWith("\"") && val.endsWith("\""));
            if (isNotQuoted && !val.match(containsNumberRegex)) {
                unit[key] = "\"" + val + "\"";
            }
        });
        if (unit.Ubertip) {
            unit.Ubertip = "\"" + unit.Ubertip + "\"";
        }

        unit.SLKUnit.SortAbil = "\"z3\"";

        unit.SortBalance = "\"z3\"";
        unit.Sort2 = "\"zzm\"";
        if (!unit.Type) {
            unit.Type = "\"_\"";
        }
        unit.RealHP = unit.HP;
        if (!unit.Def) {
            unit.Def = "\"0\"";
        }
        unit.Realdef = unit.Def;
        if (!unit.STR) {
            unit.STR = "\"-\"";
        }
        if (!unit.AGI) {
            unit.AGI = "\"-\"";
        }
        if (!unit.INT) {
            unit.INT = "\"-\"";
        }
        unit.AbilTest = "\"-\"";
        if (!unit.Primary) {
            unit.Primary = "\"_\"";
        }
        if (!unit.Upgrades) {
            unit.Upgrades = "\"_\"";
        }
        unit.Nbrandom = "\"_\"";
        unit.InBeta = "0";

        unit.Sort = "\"z3\"";
        if (!unit.Threat) {
            unit.Threat = "1";
        }
        if (!unit.Valid) {
            unit.Valid = "1";
        }
        if (!unit.TargType) {
            unit.TargType = "\"_\"";
        }
        unit.FatLOS = "0";
        unit.BuffType = "\"_\"";
        unit.BuffRadius = "\"-\"";
        unit.NameCount = "\"-\"";
        if (!unit.RequireWaterRadius) {
            unit.RequireWaterRadius = "0";
        }
        if (!unit.RequireWaterRadius) {
            unit.RequireWaterRadius = "0";
        }
        if (!unit.RequireWaterRadius) {
            unit.RequireWaterRadius = "0";
        }
        if (!unit.RequireWaterRadius) {
            unit.RequireWaterRadius = "0";
        }
        unit.InBeta = "0";
        unit.Version = "1";

        unit.SortUI = "\"z3\"";
        unit.TilesetSpecific = "0";
        unit.Name = "-";
        unit.Campaign = "1";
        unit.InEditor = "1";
        unit.HiddenInEditor = "0";
        unit.HostilePal = "0";
        unit.DropItems = "1";
        unit.NbmmIcon = "1";
        unit.UseClickHelper = "0";
        unit.HideHeroBar = "0";
        unit.HideHeroMinimap = "0";
        unit.HideHeroDeathMsg = "0";
        unit.Weap1 = "\"_\"";
        unit.Weap2 = "\"_\"";
        unit.InBeta = "0";

        unit.SortWeap = "\"n2\"";
        unit.Sort2 = "\"zzm\"";
        unit.RngTst = "\"-\"";
        unit.Mincool1 = "\"-\"";
        unit.Mindmg1 = "0";
        unit.Mindmg2 = "0";
        unit.Avgdmg1 = "0";
        unit.Avgdmg2 = "0";
        unit.Maxdmg1 = "0";
        unit.Maxdmg2 = "0";
        if (!unit.Targs1) {
            unit.Targs1 = "\"-\"";
        }
        if (!unit.Targs2) {
            unit.Targs2 = "\"-\"";
        }
        if (!unit.DmgUp1) {
            unit.DmgUp1 = "\"-\"";
        }
        if (!unit.DmgUp2) {
            unit.DmgUp2 = "\"-\"";
        }
        if (!unit.Hfact1) {
            unit.Hfact1 = "\"-\"";
        }
        if (!unit.Hfact2) {
            unit.Hfact2 = "\"-\"";
        }
        if (!unit.Qfact1) {
            unit.Qfact1 = "\"-\"";
        }
        if (!unit.Qfact2) {
            unit.Qfact2 = "\"-\"";
        }
        if (!unit.SplashTargs1) {
            unit.SplashTargs1 = "\"_\"";
        }
        if (!unit.SplashTargs2) {
            unit.SplashTargs2 = "\"_\"";
        }
        if (!unit.DmgUpg) {
            unit.DmgUpg = "\"-\"";
        }
        unit.InBeta = "0";
        unit.RngTst2 = "\"-\"";

        const message = {name: "saveUnit", payload: unit};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            let oldUnit = null;
            for (let i = 0; i < unitDataList.length; i++) {
                if (unitDataList[i].Id === unitId) {
                    oldUnit = i;
                    break;
                }
            }

            if (oldUnit) {
                unitDataList[oldUnit] = {Id: unitId, Name: unit.Name};
            } else {
                unitDataList.push({
                    Id: unitId,
                    Name: unit.Name,
                    EditorSuffix: unit.Editorsuffix
                });
            }

            index.unitSearch(document.getElementById("unitSearchInput"));
        });
    },
    loadIcons: function () {
        const message = {name: "loadIcons", payload: null};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            message.payload.forEach(icon => {
                unitIconNameToPath[icon.Name.toLowerCase()] = icon.Path;
                unitIconPathToName[icon.Path.toLowerCase()] = icon.Name.toLowerCase();
            });

            const icons = new Bloodhound({
                datumTokenizer: Bloodhound.tokenizers.whitespace,
                queryTokenizer: Bloodhound.tokenizers.whitespace,
                local: message.payload.map(icon => icon.Name)
            });

            $("#icon-selector").typeahead({
                    hint: true,
                    highlight: true,
                    minLength: 0
                },
                {
                    name: "icons",
                    limit: Object.keys(message.payload).length,
                    source: (q, sync) => {
                        if (q === "") {
                            sync(icons.all());
                        } else {
                            icons.search(q, sync);
                        }
                    }
                }).bind("typeahead:selected", (obj, datum) => {
                    if (datum) {
                        index.loadModalIcon(unitIconNameToPath[datum.toLowerCase()]);
                    }
            }).bind("typeahead:cursorchange", (obj, datum) => {
                if (datum) {
                    index.loadModalIcon(unitIconNameToPath[datum.toLowerCase()]);
                }
            });

            index.loadMdx();
        });
    },
    changeMdxModel: function (input) {
        const inputValue = input.value.toLowerCase();

        if (unitModelNameToPath.hasOwnProperty(inputValue)) {
            loadMdxModel(unitModelNameToPath[inputValue]);
        }
    },
    loadMdx: function () {
        const message = {name: "loadMdx", payload: null};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            message.payload.forEach(unitModel => {
                unitModelNameToPath[unitModel.Name.toLowerCase()] = unitModel.Path;
                unitModelPathToName[unitModel.Path.toLowerCase()] = unitModel.Name.toLowerCase();
            });

            const models = new Bloodhound({
                datumTokenizer: Bloodhound.tokenizers.whitespace,
                queryTokenizer: Bloodhound.tokenizers.whitespace,
                local: message.payload.map(unitModel => unitModel.Name)
            });

            $("#model-selector").typeahead({
                    hint: true,
                    highlight: true,
                    minLength: 0
                },
                {
                    name: "models",
                    limit: Object.keys(message.payload).length,
                    source: (q, sync) => {
                        if (q === "") {
                            sync(models.all());
                        } else {
                            models.search(q, sync);
                        }
                    }
                }).bind("typeahead:select", (obj, datum) => {
                    if (datum) {
                        loadMdxModel(unitModelNameToPath[datum.toLowerCase()]);
                    }
            }).bind("typeahead:cursorchange", (obj, datum) => {
                if (datum) {
                    loadMdxModel(unitModelNameToPath[datum.toLowerCase()]);
                }
            });

            index.startMainWindow()
        });
    },
    selectMdxModel: function () {
        const inputValue = document.getElementById("model-selector").value.toLowerCase();
        if (unitModelNameToPath.hasOwnProperty(inputValue)) {
            let modelFileInput;
            if (modelMode === "unit") {
                modelFileInput = document.getElementById("Unit-File");
                modelFileInput.value = unitModelNameToPath[inputValue];
                index.saveFieldToUnit(modelFileInput);
            } else if (modelMode === "item") {
                modelFileInput = document.getElementById("Item-File");
                modelFileInput.value = unitModelNameToPath[inputValue];
                index.saveFieldToItem(modelFileInput);
            }

            $('#model-modal').modal('toggle');
        }
    },
    loadConfig: function () {
        const message = {name: "loadConfig", payload: null};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            document.getElementById("configInput").value = message.payload.InDir;
            document.getElementById("configOutput").value = message.payload.OutDir;
            index.disableInputs(message.payload.IsLocked);
            index.setUnitRegexSearch(message.payload.IsRegexSearch);

            index.loadVersion();
        });
    },
    loadVersion: function () {
        $.getJSON("version.json", (data) => {
            document.getElementById("version-info").innerText = data.version;
            index.loadIcons();
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

            document.getElementById("Unit-UnitID").value = message.payload;
        });
    },
    generateItemId: function () {
        const message = {name: "generateItemId", payload: null};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            document.getElementById("Item-ItemID").value = message.payload;
        });
    },
    generateUnitTooltip: function () {
        const attacksEnabled = document.getElementById("Unit-WeapsOn").value;
        let value = "";
        if (attacksEnabled === "1" || attacksEnabled === "3") {
            value += "|cffffcc00Attack:|r " + document.getElementById("Unit-AtkType1").value.charAt(0).toUpperCase() + document.getElementById("Unit-AtkType1").value.substr(1) + "|n";
            value += "|cffffcc00Cooldown:|r " + document.getElementById("Unit-Cool1").value + "|n";
            const baseDamage = parseInt(document.getElementById("Unit-Dmgplus1").value, 10);
            const damageNumberOfDice = parseInt(document.getElementById("Unit-Dice1").value, 10);
            const damageSidesPerDie = parseInt(document.getElementById("Unit-Sides1").value, 10);
            value += "|cffffcc00Damage:|r " + (baseDamage + damageNumberOfDice) + " - " + (baseDamage + damageNumberOfDice * damageSidesPerDie) + "|n";
            value += "|cffffcc00Range:|r " + document.getElementById("Unit-RangeN1").value + "|n";
        } else if (attacksEnabled === "2") {
            value += "|cffffcc00Attack:|r " + document.getElementById("Unit-AtkType2").value.charAt(0).toUpperCase() + "|n";
            value += "|cffffcc00Cooldown:|r " + document.getElementById("Unit-Cool2").value + "|n";
            const baseDamage = parseInt(document.getElementById("Unit-Dmgplus2").value, 10);
            const damageNumberOfDice = parseInt(document.getElementById("Unit-Dice2").value, 10);
            const damageSidesPerDie = parseInt(document.getElementById("Unit-Sides2").value, 10);
            value += "|cffffcc00Damage:|r " + (baseDamage + damageNumberOfDice) + " - " + (baseDamage + damageNumberOfDice * damageSidesPerDie) + "|n";
            value += "|cffffcc00Range:|r " + document.getElementById("Unit-RangeN2").value + "|n";
        } else if (attacksEnabled === "0") {
            value += "|cffffcc00Attack:|r None|n";
            value += "|cffffcc00Range:|r " + document.getElementById("Unit-RangeN1").value + "|n";
        }

        if (attacksEnabled === "3") {
            value += "|cffffcc00Attack(2):|r " + document.getElementById("Unit-AtkType2").value.charAt(0).toUpperCase() + document.getElementById("Unit-AtkType2").value.substr(1) + "|n";
            value += "|cffffcc00Cooldown(2):|r " + document.getElementById("Unit-Cool2").value + "|n";
            const baseDamage = parseInt(document.getElementById("Unit-Dmgplus2").value, 10);
            const damageNumberOfDice = parseInt(document.getElementById("Unit-Dice2").value, 10);
            const damageSidesPerDie = parseInt(document.getElementById("Unit-Sides2").value, 10);
            value += "|cffffcc00Damage(2):|r " + (baseDamage + damageNumberOfDice) + " - " + (baseDamage + damageNumberOfDice * damageSidesPerDie) + "|n";
            value += "|cffffcc00Range(2):|r " + document.getElementById("Unit-RangeN2").value + "|n";
        }

        const ubertipInput = document.getElementById("Unit-Ubertip");
        ubertipInput.value = value;
        index.multiColorTextarea('unit-preview', ubertipInput);
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
    startMainWindow: function () {
        document.getElementById("mainwindow").hidden = true;
        document.getElementById("downloadwindow").hidden = true;
        document.getElementById("loadingwindow").hidden = false;

        index.activateHotkeys();
        index.listenToModals();
        index.loadSlk();

        document.getElementById("downloadwindow").hidden = true;
        document.getElementById("loadingwindow").hidden = true;
        document.getElementById("mainwindow").hidden = false;
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
                        index.saveToFile();
                    } else if (e.metaKey && e.key === "f") {
                        document.getElementById("unitSearchInput").focus();
                    }
                }
            } else {
                document.onkeydown = function (e) {
                    if (e.ctrlKey && e.key === "s") {
                        index.saveToFile();
                    } else if (e.ctrlKey && e.key === "f") {
                        document.getElementById("unitSearchInput").focus();
                    }
                }
            }
        });
    },
    listenToModals: function () {
        $("#model-modal").on("shown.bs.modal", () => document.getElementById("model-selector").focus());
        $("#icon-modal").on("shown.bs.modal", () => document.getElementById("icon-selector").focus())
    },
    setUnitRegexSearch: function (bool) {
        const message = {name: "setRegexSearch", payload: bool};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            isRegexSearch = bool;
            document.getElementById("Unit-Regex-Toggle").checked = isRegexSearch;
            index.unitSearch(document.getElementById("unitSearchInput"));
        });
    },
    setItemRegexSearch: function (bool) {
        const message = {name: "setRegexSearch", payload: bool};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            isItemRegexSearch = bool;
            document.getElementById("Item-Regex-Toggle").checked = isItemRegexSearch;
            index.itemSearch(document.getElementById("itemSearchInput"));
        });
    },
    sortUnitName: function () {
        sortUnitNameState = (sortUnitNameState + 1) % 3;
        sortUnitIdState = 0;
        const sortList = unitDataList.slice(0);

        if (sortUnitNameState === 0) {
            document.getElementById("unit-name-sort-icon").setAttribute("class", "fas fa-sort");
            document.getElementById("unit-id-sort-icon").setAttribute("class", "fas fa-sort");
        } else if (sortUnitNameState === 1) {
            document.getElementById("unit-name-sort-icon").setAttribute("class", "fas fa-sort-up");
            document.getElementById("unit-id-sort-icon").setAttribute("class", "fas fa-sort");
            sortList.sort(sortNameAlphabetical);
        } else {
            document.getElementById("unit-name-sort-icon").setAttribute("class", "fas fa-sort-down");
            document.getElementById("unit-id-sort-icon").setAttribute("class", "fas fa-sort");
            sortList.sort(sortNameInverse);
        }

        addTableData(document.getElementById("unitTableBody"), "selectUnit", sortList);
    },
    sortUnitId: function () {
        sortUnitIdState = (sortUnitIdState + 1) % 3;
        sortUnitNameState = 0;
        const sortList = unitDataList.slice(0);

        if (sortUnitIdState === 0) {
            document.getElementById("unit-id-sort-icon").setAttribute("class", "fas fa-sort");
            document.getElementById("unit-name-sort-icon").setAttribute("class", "fas fa-sort");
        } else if (sortUnitIdState === 1) {
            document.getElementById("unit-id-sort-icon").setAttribute("class", "fas fa-sort-up");
            document.getElementById("unit-name-sort-icon").setAttribute("class", "fas fa-sort");
            sortList.sort(sortUnitIdAlphabetical);
        } else {
            document.getElementById("unit-id-sort-icon").setAttribute("class", "fas fa-sort-down");
            document.getElementById("unit-name-sort-icon").setAttribute("class", "fas fa-sort");
            sortList.sort(sortUnitIdInverse);
        }

        addTableData(document.getElementById("unitTableBody"), "selectUnit", sortList);
    },
    sortItemName: function () {
        sortItemNameState = (sortItemNameState + 1) % 3;
        sortItemIdState = 0;
        const sortList = itemDataList.slice(0);

        if (sortItemNameState === 0) {
            document.getElementById("item-name-sort-icon").setAttribute("class", "fas fa-sort");
            document.getElementById("item-id-sort-icon").setAttribute("class", "fas fa-sort");
        } else if (sortItemNameState === 1) {
            document.getElementById("item-name-sort-icon").setAttribute("class", "fas fa-sort-up");
            document.getElementById("item-id-sort-icon").setAttribute("class", "fas fa-sort");
            sortList.sort(sortNameAlphabetical);
        } else {
            document.getElementById("item-name-sort-icon").setAttribute("class", "fas fa-sort-down");
            document.getElementById("item-id-sort-icon").setAttribute("class", "fas fa-sort");
            sortList.sort(sortNameInverse);
        }

        addTableData(document.getElementById("itemTableBody"), "selectItem", sortList);
    },
    sortItemId: function () {
        sortItemIdState = (sortItemIdState + 1) % 3;
        sortItemNameState = 0;
        const sortList = itemDataList.slice(0);

        if (sortItemIdState === 0) {
            document.getElementById("item-id-sort-icon").setAttribute("class", "fas fa-sort");
            document.getElementById("item-name-sort-icon").setAttribute("class", "fas fa-sort");
        } else if (sortItemIdState === 1) {
            document.getElementById("item-id-sort-icon").setAttribute("class", "fas fa-sort-up");
            document.getElementById("item-name-sort-icon").setAttribute("class", "fas fa-sort");
            sortList.sort(sortItemIdAlphabetical);
        } else {
            document.getElementById("item-id-sort-icon").setAttribute("class", "fas fa-sort-down");
            document.getElementById("item-name-sort-icon").setAttribute("class", "fas fa-sort");
            sortList.sort(sortItemIdInverse);
        }

        addTableData(document.getElementById("itemTableBody"), "selectItem", sortList);
    },
    saveOptions: function () {
        const InDir = document.getElementById("configInput").value;
        const OutDir = document.getElementById("configOutput").value;
        const message = {name: "saveOptions", payload: {InDir, OutDir}};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            $('#options-modal').modal('toggle');
            index.loadSlk();
        });
    },
    updateNewUnitIsGenerated: function (input) {
        if (input.checked) {
            input.required = true;
            document.getElementById("NewUnit-UnitID").disabled = true;
        } else {
            input.required = false;
            document.getElementById("NewUnit-UnitID").disabled = false;
        }
    },
    updateNewItemIsGenerated: function (input) {
        if (input.checked) {
            input.required = true;
            document.getElementById("NewItem-ItemID").disabled = true;
        } else {
            input.required = false;
            document.getElementById("NewItem-ItemID").disabled = false;
        }
    },
    createNewUnit: function () {
        if (!document.getElementById("NewUnit-Form").checkValidity())
            return;

        const generateId = document.getElementById("NewUnit-Generated").checked;
        const unitId = generateId ? document.getElementById("NewUnit-UnitID").value : null;
        // const baseUnitValue = document.getElementById("NewUnit-BaseUnitId").value;
        const baseUnitId = null; // baseUnitValue.length > 0 ? baseUnitValue : null;
        const name = document.getElementById("NewUnit-Name").value;
        const attackType = document.getElementById("NewUnit-AttackType").value;
        const unitType = document.getElementById("NewUnit-TypeUnit").checked ? document.getElementById("NewUnit-TypeUnit").value :
            (document.getElementById("NewUnit-TypeBuilding").checked ? document.getElementById("NewUnit-TypeBuilding").value :
                (document.getElementById("NewUnit-TypeHero").checked ? document.getElementById("NewUnit-TypeHero").value :
                    "none"));
        const message = {
            name: "createNewUnit",
            payload: {
                UnitId: unitId,
                GenerateId: generateId,
                Name: name,
                UnitType: unitType,
                BaseUnitId: baseUnitId,
                AttackType: attackType
            }
        };
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            index.selectUnitFromId(message.payload.UnitID);
            unitDataList.push({Id: message.payload.UnitID, Name: message.payload.Name});
            index.unitSearch(document.getElementById("unitSearchInput"));

            $('#new-unit-modal').modal('toggle');
        });
    },
    createNewItem: function () {
        if (!document.getElementById("NewItem-Form").checkValidity())
            return;

        const generateId = document.getElementById("NewItem-Generated").checked;
        const itemId = generateId ? document.getElementById("NewItem-ItemID").value : null;
        // const baseItemValue = document.getElementById("NewItem-BaseItemId").value;
        const baseItemId = null; // baseItemValue.length > 0 ? baseItemValue : null;
        const name = document.getElementById("NewItem-Name").value;
        const message = {
            name: "createNewItem",
            payload: {
                ItemId: itemId,
                GenerateId: generateId,
                Name: name,
                BaseItemId: baseItemId
            }
        };
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            index.selectItemFromId(message.payload.ItemID);
            itemDataList.push({Id: message.payload.ItemID, Name: message.payload.Name});
            index.itemSearch(document.getElementById("itemSearchInput"));

            $('#new-item-modal').modal('toggle');
        });
    },
    switchTab: function (containerId) {
        if (containerId === "units-container") {
            document.getElementById("items-container").hidden = true;
            // document.getElementById("abilities-container").hidden = true;
            // document.getElementById("buffs-container").hidden = true;
            document.getElementById("new-item-button").hidden = true;
            document.getElementById("new-unit-button").hidden = false;
            document.getElementById("units-container").hidden = false;
            document.getElementById("unit-nav-pills").hidden = false;

            document.getElementById("items-tab").className = "nav-link";
            document.getElementById("abilities-tab").className = "nav-link";
            document.getElementById("buffs-tab").className = "nav-link";
            document.getElementById("units-tab").className = "nav-link active";
        } else if (containerId === "items-container") {
            document.getElementById("units-container").hidden = true;
            // document.getElementById("abilities-container").hidden = true;
            // document.getElementById("buffs-container").hidden = true;
            document.getElementById("new-unit-button").hidden = true;
            document.getElementById("unit-nav-pills").hidden = true;
            document.getElementById("new-item-button").hidden = false;
            document.getElementById("items-container").hidden = false;

            document.getElementById("units-tab").className = "nav-link";
            document.getElementById("abilities-tab").className = "nav-link";
            document.getElementById("buffs-tab").className = "nav-link";
            document.getElementById("items-tab").className = "nav-link active";
        } else if (containerId === "abilities-container") {
            document.getElementById("units-container").hidden = true;
            document.getElementById("items-container").hidden = true;
            // document.getElementById("buffs-container").hidden = true;
            // document.getElementById("abilities-container").hidden = false;

            document.getElementById("units-tab").className = "nav-link";
            document.getElementById("items-tab").className = "nav-link";
            document.getElementById("buffs-tab").className = "nav-link";
            document.getElementById("abilities-tab").className = "nav-link active";
        } else if (containerId === "buffs-container") {
            document.getElementById("units-container").hidden = true;
            document.getElementById("items-container").hidden = true;
            // document.getElementById("abilities-container").hidden = true;
            // document.getElementById("buffs-container").hidden = false;

            document.getElementById("units-tab").className = "nav-link";
            document.getElementById("items-tab").className = "nav-link";
            document.getElementById("abilities-tab").className = "nav-link";
            document.getElementById("buffs-tab").className = "nav-link active";
        }
    }
};