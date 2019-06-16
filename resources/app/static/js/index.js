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
    loadMdxModal: function (inputId) {
        modelInputId = inputId;
        initMdx();

        const currentModelPath = document.getElementById(inputId).value;
        const lowercaseModelPath = currentModelPath.toLowerCase().replace(new RegExp("\.mdl$"), ".mdx");
        const modelPathWithExtension = lowercaseModelPath.endsWith("mdx") ? lowercaseModelPath : lowercaseModelPath + ".mdx";
        if (!modelPathToName.hasOwnProperty(modelPathWithExtension)) {
            return;
        }

        $("#model-selector").typeahead("val", modelPathToName[modelPathWithExtension]);
        loadMdxModel(modelPathWithExtension);
    },
    loadIconModal: function (inputId) {
        const input = document.getElementById(inputId);
        const currentIconPath = input.value;
        if (!unitIconPathToName.hasOwnProperty(currentIconPath.toLowerCase())) {
            return;
        }

        iconInputId = inputId;
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
    removeAbility: function () {
        if (selectedAbilityId === null)
            return;

        const message = {name: "removeAbility", payload: selectedAbilityId};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            selectedAbilityId = null;
            abilityDataList = abilityDataList.filter(ability => ability.Id !== message.payload);
            index.abilitySearch(document.getElementById("abilitySearchInput"));
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
    loadAbilityData: function () {
        const message = {name: "loadAbilityData", payload: null};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            abilityDataList = message.payload;
            const abilityTableBody = document.getElementById("abilityTableBody");
            addTableData(abilityTableBody, "selectAbility", message.payload);
        });
    },
    loadAbilityMetaData: function () {
        const message = {name: "loadAbilityMetaData", payload: null};
        astilectron.sendMessage(message, function (message) {
            // Check for errors|
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            Object.keys(message.payload).forEach(key => {
                const {UseSpecific} = message.payload[key];
                if (!UseSpecific) {
                    console.log("Missing:", message.payload[key]);
                    return;
                }

                const useSpecificSplit = trimQuotes(UseSpecific).split(",");
                useSpecificSplit.forEach(useSpecific => {
                    if (!abilityMetaDataFields.hasOwnProperty(useSpecific)) {
                        abilityMetaDataFields[useSpecific] = {};
                    }

                    const {Data} = message.payload[key];
                    const addition = Data === "0" ? "" : String.fromCharCode(parseInt(trimQuotes(Data)) + 64);
                    abilityMetaDataFields[useSpecific][trimQuotes(message.payload[key].Field) + addition] = {
                        DisplayName: trimQuotes(message.payload[key].DisplayName)
                    };
                });
            });
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

                const {StatusClass, StatusIconClass, FileName} = fileInfo;
                fileInfoContainerString += '<li><span class="' + StatusClass + '">' + '<i class="fas ' + StatusIconClass + '"></i> ' + FileName + '</span></li>';
                i++;
            });

            fileInfoContainerString += '</ul>';
            document.getElementById("file-info-container").innerHTML = fileInfoContainerString;

            index.loadUnitData();
            index.loadItemData();
            index.loadAbilityData();
            index.loadAbilityMetaData();
        });
    },
    unitSearch: function (inputField) {
        let filteredUnitDataList;
        if (isUnitRegexSearch) {
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
    abilitySearch: function (inputField) {
        let filteredAbilityDataList;
        if (isAbilityRegexSearch) {
            const regex = new RegExp(inputField.value, "i");
            filteredAbilityDataList = abilityDataList.filter(abilityData => (abilityData.Name + abilityData.Id + abilityData.EditorSuffix).match(regex));
        } else {
            filteredAbilityDataList = abilityDataList.filter(abilityData => (abilityData.Name + abilityData.Id + abilityData.EditorSuffix).includes(inputField.value));
        }

        const abilityTableBody = document.getElementById("abilityTableBody");
        addTableData(abilityTableBody, "selectAbility", filteredAbilityDataList);
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
                message.payload.Ubertip = trimQuotes(message.payload.Ubertip).replace(new RegExp("\\|n", "g"), "\n");
            }

            const {Buttonpos, ButtonposX, ButtonposY} = message.payload;
            message.payload.Buttonpos = getSanitizedButtonpos(Buttonpos, ButtonposX, ButtonposY);

            Object.keys(message.payload).forEach(slkUnitKey => {
                const value = trimQuotes(message.payload[slkUnitKey] ? message.payload[slkUnitKey] : "");
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
                        elem.value = getSanitizedButtonpos(message.payload[slkUnitKey]);
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
            index.loadIcon("ImageUnit-Art", document.getElementById("Unit-Art"));
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
                message.payload.Ubertip = trimQuotes(message.payload.Ubertip).replace(new RegExp("\\|n", "g"), "\n");
            }

            const {Buttonpos, ButtonposX, ButtonposY} = message.payload;
            message.payload.Buttonpos = getSanitizedButtonpos(Buttonpos, ButtonposX, ButtonposY);

            Object.keys(message.payload).forEach(slkItemKey => {
                const value = trimQuotes(message.payload[slkItemKey] ? message.payload[slkItemKey] : "");
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
                        elem.value = getSanitizedButtonpos(message.payload[slkItemKey]);
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
            index.loadIcon("ImageItem-Art", document.getElementById("Item-Art"));
        });
    },
    selectAbility: function (abilityTableRow) {
        index.selectAbilityFromId(abilityTableRow.id);
    },
    selectAbilityFromId: function (abilityId) {
        selectedAbilityId = abilityId;
        const message = {name: "selectAbility", payload: abilityId};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            $("#abilityTableBody>tr").removeClass("active");
            document.getElementById(abilityId).setAttribute("class", "active");

            if (message.payload.Ubertip) {
                message.payload.Ubertip = trimQuotes(message.payload.Ubertip).replace(new RegExp("\\|n", "g"), "\n");
            }

            const {Buttonpos, ButtonposX, ButtonposY} = message.payload;
            message.payload.Buttonpos = getSanitizedButtonpos(Buttonpos, ButtonposX, ButtonposY);

            const {Unbuttonpos, UnbuttonposX, UnbuttonposY} = message.payload;
            message.payload.Unbuttonpos = getSanitizedButtonpos(Unbuttonpos, UnbuttonposX, UnbuttonposY);

            const levelDependentDataElem = document.getElementById("abilityLevelDependentData");
            while (levelDependentDataElem.firstElementChild) {
                levelDependentDataElem.firstElementChild.remove();
            }

            Object.keys(message.payload).forEach(slkAbilityKey => {
                if (slkAbilityKey.match(abilityKeyEndsWithNumberRegex) === null) {
                    const value = trimQuotes(message.payload[slkAbilityKey] ? message.payload[slkAbilityKey] : "");
                    const elem = document.getElementById("Ability-" + slkAbilityKey);
                    if (elem) {
                        if (elem instanceof HTMLInputElement || elem instanceof HTMLSelectElement || elem instanceof HTMLTextAreaElement) {
                            const type = elem.type;

                            if (type === "text" || type === "textarea" || type === "select-one") {
                                elem.value = value;
                            } else if (type === "checkbox") {
                                elem.checked = value === "1";
                            }
                        } else if (elem.id === "Ability-Buttonpos") {
                            elem.value = getSanitizedButtonpos(message.payload[slkAbilityKey]);
                        } else if (elem.classList.contains("multi-check")) {
                            const childInputs = $("#Ability-" + slkAbilityKey + " :input");
                            const valueSplit = value.split(",");
                            const valueLower = valueSplit.map(val => val.toLowerCase());
                            for (let i = 0; i < childInputs.length; i++) {
                                childInputs[i].checked = valueLower.includes(childInputs[i].value.toLowerCase());
                            }
                        }
                    }
                }
            });

            let updatedHtml = "";
            const {Levels, Code} = message.payload;
            const trimmedCode = trimQuotes(Code ? Code : "");
            const parsedLevels = parseInt(Levels);
            if (parsedLevels && typeof parsedLevels === "number" && Number.isNaN(parsedLevels) === false && parsedLevels > 0) {
                const updateHtmlForEachLevel = (fieldId, displayName) => {
                    for (let i = 1; i <= parsedLevels; i++) {
                        const value = trimQuotes(message.payload[fieldId + i] ? message.payload[fieldId + i] : "");
                        updatedHtml += '<li><label for="Ability-' + fieldId + i + '">' + displayName + ' - ' + i + '</label><input oninput="index.saveField(this, this.form.id)" type="text" class="form-control" id="Ability-' + fieldId + i + '" placeholder="' + displayName + '" value="' + value + '"/></li>';
                    }
                };

                if (abilityMetaDataFields[trimmedCode].hasOwnProperty("DataA"))
                    updateHtmlForEachLevel("DataA", abilityMetaDataFields[trimmedCode]["DataA"].DisplayName);
                if (abilityMetaDataFields[trimmedCode].hasOwnProperty("DataB"))
                    updateHtmlForEachLevel("DataB", abilityMetaDataFields[trimmedCode]["DataB"].DisplayName);
                if (abilityMetaDataFields[trimmedCode].hasOwnProperty("DataC"))
                    updateHtmlForEachLevel("DataC", abilityMetaDataFields[trimmedCode]["DataC"].DisplayName);
                if (abilityMetaDataFields[trimmedCode].hasOwnProperty("DataD"))
                    updateHtmlForEachLevel("DataD", abilityMetaDataFields[trimmedCode]["DataD"].DisplayName);
                if (abilityMetaDataFields[trimmedCode].hasOwnProperty("DataE"))
                    updateHtmlForEachLevel("DataE", abilityMetaDataFields[trimmedCode]["DataE"].DisplayName);
                if (abilityMetaDataFields[trimmedCode].hasOwnProperty("DataF"))
                    updateHtmlForEachLevel("DataF", abilityMetaDataFields[trimmedCode]["DataF"].DisplayName);
                if (abilityMetaDataFields[trimmedCode].hasOwnProperty("DataG"))
                    updateHtmlForEachLevel("DataG", abilityMetaDataFields[trimmedCode]["DataG"].DisplayName);
                if (abilityMetaDataFields[trimmedCode].hasOwnProperty("DataH"))
                    updateHtmlForEachLevel("DataH", abilityMetaDataFields[trimmedCode]["DataH"].DisplayName);
                if (abilityMetaDataFields[trimmedCode].hasOwnProperty("DataI"))
                    updateHtmlForEachLevel("DataI", abilityMetaDataFields[trimmedCode]["DataI"].DisplayName);
                updateHtmlForEachLevel("UnitID", "Summoned Unit Type");
                updateHtmlForEachLevel("BuffID", "Buffs");
                updateHtmlForEachLevel("EfctID", "Effects");
                updateHtmlForEachLevel("Targs", "Targets Allowed");
                updateHtmlForEachLevel("Cast", "Casting Time");
                updateHtmlForEachLevel("Dur", "Duration - Normal");
                updateHtmlForEachLevel("HeroDur", "Duration - Hero");
                updateHtmlForEachLevel("Cool", "Cooldown");
                updateHtmlForEachLevel("Cost", "Mana Cost");
                updateHtmlForEachLevel("Area", "Area of Effect");
                updateHtmlForEachLevel("Rng", "Cast Range");

                const {Tip, Ubertip, Untip, Unubertip} = message.payload;
                if (Tip) {
                    updatedHtml += '<li id="Ability-Tip" style="display: block; align-items: inherit; flex-flow: inherit; text-align: inherit;"><ul class="flex-outer multi-text">';

                    const tipSplit = Tip.split(",");
                    for (let i = 1; i <= parsedLevels; i++) {
                        const fieldId = "Tip-" + i;
                        const displayName = "Tooltip - Normal - " + i;
                        const value = trimQuotes(tipSplit[i - 1] ? tipSplit[i - 1] : "");
                        updatedHtml += '<li><label for="' + fieldId + '">' + displayName + '</label><input class="form-control sub-multi-text" oninput="index.saveField(this, this.form.id)" type="text" value="' + value + '"/></li>';
                    }

                    updatedHtml += '</ul></li>';
                }

                if (Ubertip) {
                    updatedHtml += '<li id="Ability-Ubertip" style="display: block; align-items: inherit; flex-flow: inherit; text-align: inherit;"><ul class="flex-outer multi-text">';

                    const tipSplit = Tip.split(",");
                    for (let i = 1; i <= parsedLevels; i++) {
                        const fieldId = "Ubertip-" + i;
                        const displayName = "Tooltip - Extended - " + i;
                        const value = trimQuotes(tipSplit[i - 1] ? tipSplit[i - 1] : "");
                        updatedHtml += '<li><label for="' + fieldId + '">' + displayName + '</label><input class="form-control sub-multi-text" oninput="index.saveField(this, this.form.id)" type="text" value="' + value + '"/></li>';
                    }

                    updatedHtml += '</ul></li>';
                }

                if (Untip) {
                    updatedHtml += '<li id="Ability-Untip" style="display: block; align-items: inherit; flex-flow: inherit; text-align: inherit;"><ul class="flex-outer multi-text">';

                    const tipSplit = Tip.split(",");
                    for (let i = 1; i <= parsedLevels; i++) {
                        const fieldId = "Untip-" + i;
                        const displayName = "Tooltip - Turn Off - " + i;
                        const value = trimQuotes(tipSplit[i - 1] ? tipSplit[i - 1] : "");
                        updatedHtml += '<li><label for="' + fieldId + '">' + displayName + '</label><input class="form-control sub-multi-text" oninput="index.saveField(this, this.form.id)" type="text" value="' + value + '"/></li>';
                    }

                    updatedHtml += '</ul></li>';
                }

                if (Unubertip) {
                    updatedHtml += '<li id="Ability-Unubertip" style="display: block; align-items: inherit; flex-flow: inherit; text-align: inherit;"><ul class="flex-outer multi-text">';

                    const tipSplit = Tip.split(",");
                    for (let i = 1; i <= parsedLevels; i++) {
                        const fieldId = "Unubertip-" + i;
                        const displayName = "Tooltip - Turn Off Extended - " + i;
                        const value = trimQuotes(tipSplit[i - 1] ? tipSplit[i - 1] : "");
                        updatedHtml += '<li><label for="' + fieldId + '">' + displayName + '</label><input class="form-control sub-multi-text" oninput="index.saveField(this, this.form.id)" type="text" value="' + value + '"/></li>';
                    }

                    updatedHtml += '</ul></li>';
                }
            }

            levelDependentDataElem.innerHTML = updatedHtml;
            index.loadIcon("ImageAbility-Art", document.getElementById("Ability-Art"));
            index.loadIcon("ImageAbility-Unart", document.getElementById("Ability-Unart"));
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
    saveField: function (input, idForm) {
        const formElem = document.getElementById(idForm);
        if (formElem && !formElem.checkValidity()) {
            console.log("Invalid field!");
            return;
        }

        let field = null;
        let value = null;
        const containsMultiCheck = input.classList.contains("sub-multi-check");
        const containsMultiText = input.classList.contains("sub-multi-text");
        if (containsMultiCheck || containsMultiText) {
            field = input.parentNode.parentNode.parentNode.id;
            input.parentNode.parentNode.parentNode.childNodes.forEach(child => {
                if (child instanceof HTMLUListElement) {
                    child.childNodes.forEach(listChild => {
                        if (listChild instanceof HTMLLIElement) {
                            listChild.childNodes.forEach(listElement => {
                                if (listElement instanceof HTMLInputElement) {
                                    if (containsMultiCheck && listElement.checked) {
                                        if (value === null) {
                                            value = "\"" + listElement.value;
                                        } else {
                                            value += "," + listElement.value;
                                        }
                                    } else if (containsMultiText) {
                                        if (value === null) {
                                            value += "\"" + listElement.value + "\"";
                                        } else {
                                            value += ",\"" + listElement.value + "\"";
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
            } else if (containsMultiCheck) {
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

        const id = document.getElementById(formElem.dataset.mapid).value;
        const fieldToSave = {Field: field, Value: value, Id: id};
        if (field != null && value != null) {
            const message = {name: "saveField", payload: fieldToSave};
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
            const iconInput = document.getElementById(iconInputId);
            index.saveField(iconInput, iconInput.form.id);
            index.loadIconValue(`Image${iconInputId}`, iconPath);

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

        unit.SortAbil = "\"z3\"";
        unit.SortBalance = "\"z3\"";
        unit.Sort2 = "\"zzm\"";
        if (!unit.Type) {
            unit.Type = "\"_\"";
        }

        const {HP} = unit;
        unit.RealHP = HP;
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
                const {Name, Editorsuffix} = unit;
                unitDataList.push({
                    Id: unitId,
                    Name,
                    EditorSuffix: Editorsuffix
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

        if (modelNameToPath.hasOwnProperty(inputValue)) {
            loadMdxModel(modelNameToPath[inputValue]);
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

            const {Units, Abilities, Missiles, Items} = message.payload;

            Units.forEach(unitModel => {
                modelNameToPath[unitModel.Name.toLowerCase()] = unitModel.Path;
                modelPathToName[unitModel.Path.toLowerCase()] = unitModel.Name.toLowerCase();
            });

            Abilities.forEach(unitModel => {
                modelNameToPath[unitModel.Name.toLowerCase()] = unitModel.Path;
                modelPathToName[unitModel.Path.toLowerCase()] = unitModel.Name.toLowerCase();
            });

            Missiles.forEach(missileModel => {
                modelNameToPath[missileModel.Name.toLowerCase()] = missileModel.Path;
                modelPathToName[missileModel.Path.toLowerCase()] = missileModel.Name.toLowerCase();
            });

            Items.forEach(itemModel => {
                modelNameToPath[itemModel.Name.toLowerCase()] = itemModel.Path;
                modelPathToName[itemModel.Path.toLowerCase()] = itemModel.Name.toLowerCase();
            });

            const unitModels = new Bloodhound({
                datumTokenizer: Bloodhound.tokenizers.obj.whitespace("Name"),
                queryTokenizer: Bloodhound.tokenizers.whitespace,
                identify: function (obj) {
                    return obj.Name;
                },
                local: Units
            });

            const abilityModels = new Bloodhound({
                datumTokenizer: Bloodhound.tokenizers.obj.whitespace("Name"),
                queryTokenizer: Bloodhound.tokenizers.whitespace,
                identify: function (obj) {
                    return obj.Name;
                },
                local: Abilities
            });

            const missileModels = new Bloodhound({
                datumTokenizer: Bloodhound.tokenizers.obj.whitespace("Name"),
                queryTokenizer: Bloodhound.tokenizers.whitespace,
                identify: function (obj) {
                    return obj.Name;
                },
                local: Missiles
            });

            const itemModels = new Bloodhound({
                datumTokenizer: Bloodhound.tokenizers.obj.whitespace("Name"),
                queryTokenizer: Bloodhound.tokenizers.whitespace,
                identify: function (obj) {
                    return obj.Name;
                },
                local: Items
            });

            const unitsWithDefaults = (q, sync) => {
                if (!isUnitModelFilterChecked) {
                    sync([]);
                } else if (q === "") {
                    sync(unitModels.all());
                } else {
                    unitModels.search(q, sync);
                }
            };

            const abilitiesWithDefaults = (q, sync) => {
                if (!isAbilityModelFilterChecked) {
                    sync([]);
                } else if (q === "") {
                    sync(abilityModels.all());
                } else {
                    abilityModels.search(q, sync);
                }
            };

            const missilesWithDefaults = (q, sync) => {
                if (!isMissileModelFilterChecked) {
                    sync([]);
                } else if (q === "") {
                    sync(missileModels.all());
                } else {
                    missileModels.search(q, sync);
                }
            };

            const itemsWithDefaults = (q, sync) => {
                if (!isItemModelFilterChecked) {
                    sync([]);
                } else if (q === "") {
                    sync(itemModels.all());
                } else {
                    itemModels.search(q, sync);
                }
            };

            $("#model-selector").typeahead({
                    hint: true,
                    highlight: true,
                    minLength: 0
                },
                {
                    name: "unit-models",
                    display: "Name",
                    limit: Units.length,
                    source: unitsWithDefaults,
                    templates: {
                        header: '<h3 class="model-group">Units</h3>'
                    }
                },
                {
                    name: "ability-models",
                    display: "Name",
                    limit: Abilities.length,
                    source: abilitiesWithDefaults,
                    templates: {
                        header: '<h3 class="model-group">Abilities</h3>'
                    }
                },
                {
                    name: "missile-models",
                    display: "Name",
                    limit: Missiles.length,
                    source: missilesWithDefaults,
                    templates: {
                        header: '<h3 class="model-group">Missiles</h3>'
                    }
                },
                {
                    name: "item-models",
                    display: "Name",
                    limit: Items.length,
                    source: itemsWithDefaults,
                    templates: {
                        header: '<h3 class="model-group">Items</h3>'
                    }
                }).bind("typeahead:select", (obj, datum) => {
                if (datum) {
                    loadMdxModel(datum.Path);
                }
            }).bind("typeahead:cursorchange", (obj, datum) => {
                if (datum) {
                    loadMdxModel(datum.Path);
                }
            });

            index.startMainWindow()
        });
    },
    selectMdxModel: function () {
        const inputValue = document.getElementById("model-selector").value.toLowerCase();
        if (modelNameToPath.hasOwnProperty(inputValue)) {
            const modelFileInput = document.getElementById(modelInputId);
            modelFileInput.value = modelNameToPath[inputValue];
            index.saveField(modelFileInput, modelFileInput.form.id);

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
            const {IsLocked, IsRegexSearch} = message.payload;
            index.disableInputs(IsLocked);
            index.setUnitRegexSearch(IsRegexSearch);
            index.setItemRegexSearch(IsRegexSearch);
            index.setAbilityRegexSearch(IsRegexSearch);

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
    generateAbilityId: function () {
        const message = {name: "generateAbilityId", payload: null};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            document.getElementById("Ability-Alias").value = message.payload;
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

            isUnitRegexSearch = bool;
            document.getElementById("Unit-Regex-Toggle").checked = isUnitRegexSearch;
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
    setAbilityRegexSearch: function (bool) {
        const message = {name: "setRegexSearch", payload: bool};
        astilectron.sendMessage(message, function (message) {
            // Check for errors
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return;
            }

            isAbilityRegexSearch = bool;
            document.getElementById("Ability-Regex-Toggle").checked = isAbilityRegexSearch;
            index.itemSearch(document.getElementById("abilitySearchInput"));
        });
    },
    sortName: function (elementPrefixId, tableBodyId, onclickFuncName, sortIdState, sortList) {
        if (sortIdState === 0) {
            document.getElementById(`${elementPrefixId}-name-sort-icon`).setAttribute("class", "fas fa-sort");
            document.getElementById(`${elementPrefixId}-id-sort-icon`).setAttribute("class", "fas fa-sort");
        } else if (sortIdState === 1) {
            document.getElementById(`${elementPrefixId}-name-sort-icon`).setAttribute("class", "fas fa-sort-up");
            document.getElementById(`${elementPrefixId}-id-sort-icon`).setAttribute("class", "fas fa-sort");
            sortList.sort(sortNameAlphabetical);
        } else {
            document.getElementById(`${elementPrefixId}-name-sort-icon`).setAttribute("class", "fas fa-sort-down");
            document.getElementById(`${elementPrefixId}-id-sort-icon`).setAttribute("class", "fas fa-sort");
            sortList.sort(sortNameInverse);
        }

        addTableData(document.getElementById(tableBodyId), onclickFuncName, sortList);
    },
    sortUnitName: function () {
        sortUnitNameState = (sortUnitNameState + 1) % 3;
        sortUnitIdState = 0;
        index.sortName("unit", "unitTableBody", "selectUnit", sortUnitNameState, unitDataList.slice(0));
    },
    sortItemName: function () {
        sortItemNameState = (sortItemNameState + 1) % 3;
        sortItemIdState = 0;
        index.sortName("item", "itemTableBody", "selectItem", sortItemNameState, itemDataList.slice(0));
    },
    sortAbilityName: function () {
        sortAbilityNameState = (sortAbilityNameState + 1) % 3;
        sortAbilityIdState = 0;
        index.sortName("ability", "abilityTableBody", "selectAbility", sortAbilityNameState, abilityDataList.slice(0));
    },
    sortId: function (elementPrefixId, tableBodyId, onclickFuncName, sortIdState, sortList) {
        if (sortIdState === 0) {
            document.getElementById(`${elementPrefixId}-id-sort-icon`).setAttribute("class", "fas fa-sort");
            document.getElementById(`${elementPrefixId}-name-sort-icon`).setAttribute("class", "fas fa-sort");
        } else if (sortIdState === 1) {
            document.getElementById(`${elementPrefixId}-id-sort-icon`).setAttribute("class", "fas fa-sort-up");
            document.getElementById(`${elementPrefixId}-name-sort-icon`).setAttribute("class", "fas fa-sort");
            sortList.sort(sortIdAlphabetical);
        } else {
            document.getElementById(`${elementPrefixId}-id-sort-icon`).setAttribute("class", "fas fa-sort-down");
            document.getElementById(`${elementPrefixId}-name-sort-icon`).setAttribute("class", "fas fa-sort");
            sortList.sort(sortIdInverse);
        }

        addTableData(document.getElementById(tableBodyId), onclickFuncName, sortList);
    },
    sortUnitId: function () {
        sortUnitIdState = (sortUnitIdState + 1) % 3;
        sortUnitNameState = 0;
        index.sortId("unit", "unitTableBody", "selectUnit", sortUnitIdState, unitDataList.slice(0));
    },
    sortItemId: function () {
        sortItemIdState = (sortItemIdState + 1) % 3;
        sortItemNameState = 0;
        index.sortId("item", "itemTableBody", "selectItem", sortItemIdState, itemDataList.slice(0));
    },
    sortAbilityId: function () {
        sortAbilityIdState = (sortAbilityIdState + 1) % 3;
        sortAbilityNameState = 0;
        index.sortId("ability", "abilityTableBody", "selectAbility", sortAbilityIdState, abilityDataList.slice(0));
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

            const {ItemID} = message.payload;
            index.selectItemFromId(ItemID);
            itemDataList.push({Id: ItemID, Name: message.payload.Name});
            index.itemSearch(document.getElementById("itemSearchInput"));

            $('#new-item-modal').modal('toggle');
        });
    },
    switchTab: function (containerId) {
        if (containerId === "units-container") {
            document.getElementById("items-container").hidden = true;
            document.getElementById("abilities-container").hidden = true;
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
            document.getElementById("abilities-container").hidden = true;
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
            document.getElementById("abilities-container").hidden = false;

            document.getElementById("units-tab").className = "nav-link";
            document.getElementById("items-tab").className = "nav-link";
            document.getElementById("buffs-tab").className = "nav-link";
            document.getElementById("abilities-tab").className = "nav-link active";
        } else if (containerId === "buffs-container") {
            document.getElementById("units-container").hidden = true;
            document.getElementById("items-container").hidden = true;
            document.getElementById("abilities-container").hidden = true;
            // document.getElementById("buffs-container").hidden = false;

            document.getElementById("units-tab").className = "nav-link";
            document.getElementById("items-tab").className = "nav-link";
            document.getElementById("abilities-tab").className = "nav-link";
            document.getElementById("buffs-tab").className = "nav-link active";
        }
    }
};