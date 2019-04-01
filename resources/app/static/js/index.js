let unitDataList = [];

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
                    if (message.payload.SLKUnit[slkUnitKey][key]) {
                        const elemList = $("#SLKUnit-" + slkUnitKey + "-" + key);
                        if (elemList.length > 0) {
                            if (elemList[0] instanceof HTMLInputElement) {
                                const type = elemList[0].type;

                                if (type === "text" || type === "select-one") {
                                    elemList[0].value = message.payload.SLKUnit[slkUnitKey][key];
                                } else if (type === "checkbox") {
                                    elemList[0].checked = message.payload.SLKUnit[slkUnitKey][key] === "1";
                                }
                            } else if (elemList[0].classList.contains("multi-check")) {
                                const childInputs = $("#SLKUnit-" + slkUnitKey + "-" + key + " :input");
                                for (let i = 0; i < childInputs.length; i++) {
                                    if (message.payload.SLKUnit[slkUnitKey][key].includes(childInputs[i].value)) {
                                        childInputs.checked = true;
                                    } else {
                                        childInputs.checked = false;
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

        const updatedValue = hiddenInput[0].value;

        if (!updatedValue)
            return;

        input[0].value = updatedValue;
    },
    saveUnit: function () {
        // TODO: Implement this
    }
};