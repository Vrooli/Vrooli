import { exists } from "@local/utils";
export const removeValuesUsingDot = async (obj, ...keys) => {
    keys.forEach(async (key) => {
        const keyArr = typeof key === "string" ? key.split(".") : [key];
        let currentObject = obj;
        let currentKey;
        for (let i = 0; i < keyArr.length - 1; i++) {
            currentKey = keyArr[i];
            if (typeof currentObject[currentKey] === "function") {
                currentObject[currentKey] = await currentObject[currentKey]();
            }
            currentObject = currentObject[currentKey];
            if (!exists(currentObject))
                break;
        }
        currentKey = keyArr[keyArr.length - 1];
        if (!exists(currentObject) || !exists(currentObject[currentKey]))
            return;
        delete currentObject[currentKey];
    });
};
//# sourceMappingURL=removeValuesUsingDot.js.map