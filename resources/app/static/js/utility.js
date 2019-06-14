let unitDataList = [];
let itemDataList = [];
let abilityDataList = [];
let isLocked = false;
let isUnitRegexSearch = false;
let isItemRegexSearch = false;
let isAbilityRegexSearch = false;
let isUnitModelFilterChecked = true;
let isItemModelFilterChecked = true;
let isAbilityModelFilterChecked = true;
let isMissileModelFilterChecked = true;
let selectedUnitId = null;
let selectedItemId = null;
let selectedAbilityId = null;
let isUnsaved = false;
let sortUnitNameState = 0;
let sortUnitIdState = 0;
let sortItemNameState = 0;
let sortItemIdState = 0;
let sortAbilityNameState = 0;
let sortAbilityIdState = 0;
let iconMode;
let modelMode;
const mdxModels = {};
const modelNameToPath = {};
const modelPathToName = {};
const unitIconNameToPath = {};
const unitIconPathToName = {};
const abilityMetaDataFields = {};

const trimQuotes = str => {
    if (str.startsWith("\"")) {
        str = str.substr(1);
    }

    if (str.endsWith("\"")) {
        return str.substr(0, str.length - 1);
    } else {
        return str;
    }
};

const sortNameAlphabetical = (a, b) => {
    return a.Name > b.Name ? 1 : -1;
};

const sortNameInverse = (a, b) => {
    return b.Name > a.Name ? 1 : -1;
};

const sortIdAlphabetical = (a, b) => {
    return a.Id > b.Id ? 1 : -1;
};

const sortIdInverse = (a, b) => {
    return b.Id > a.Id ? 1 : -1;
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

const getSanitizedButtonpos = (buttonpos, buttonposX, buttonposY) => {
    const buttonposInvalid = !buttonpos || buttonpos === "" || buttonpos === "_" || buttonpos === "-";
    if (buttonposInvalid === false) {
        return buttonpos;
    }

    const buttonXOrButtonYInvalid = !buttonposX || !buttonposY || buttonposX === "" || buttonposY === "" || buttonposX === "_" || buttonposY === "_" || buttonposX === "-" || buttonposY === "-";
    if (buttonposInvalid === true && buttonXOrButtonYInvalid === true) {
        return "0,0";
    } else {
        return buttonposX + "," + buttonposY;
    }
};