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
    }
};