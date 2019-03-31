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
    init: function() {
        // Init
        asticode.loader.init();
        asticode.notifier.init();

        // Wait for astilectron to be ready
        document.addEventListener('astilectron-ready', function() {
            // Listen
            index.listen();

            index.loadUnitData();
        })
    },
    listen: function() {
        astilectron.onMessage(function(message) {

        });
    },
    removeUnit: function(unit) {
        const unitMessage = { name: "removeUnit", payload: unit };

        astilectron.sendMessage(unitMessage, function(message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return
            }

            document.getElementById("removeButton").innerHTML = message.payload;
        })
    },
    loadUnitData: function() {
        const message = { name: "loadUnitData", payload: null };
        asticode.loader.show();

        astilectron.sendMessage(message, function(message) {
            asticode.loader.hide();

            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return
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
        const message = { name: "selectUnit", payload: unitId };
        astilectron.sendMessage(message, function(message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return
            }

            $("#unitTableBody>tr").removeClass("active");
            unitTableRow.setAttribute("class", "active");
            Object.keys(message.payload.SLKUnit).forEach(slkUnitKey =>
                Object.keys(message.payload.SLKUnit[slkUnitKey]).forEach(key => {
                const elemList = $("#SLKUnit-" + slkUnitKey + "-" + key);
                if (elemList.length > 0) {
                    if (!elemList[0] instanceof HTMLInputElement)
                        return;

                    if (elemList[0].type === "text") {
                        elemList[0].value = message.payload.SLKUnit[slkUnitKey][key];
                    } else if (elemList[0].type === "checkbox") {
                        elemList[0].checked = message.payload.SLKUnit[slkUnitKey][key] === "1";
                    }
                }
            }));
        });
    }
};