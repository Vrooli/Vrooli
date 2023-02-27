import { exists } from "@shared/utils";

/**
 * Removes one or more properties from an object using dot notation (ex: 'parent.child.property'). 
 * NOTE 1: Supports lazy values, but removes the lazy part
 * NOTE 2: Modifies the original object
 */
export const removeValuesUsingDot = async (obj: Record<string | number | symbol, any>, ...keys: (string | number | symbol)[]) => {
    keys.forEach(async key => {
        const keyArr = typeof key === 'string' ? key.split('.') : [key]; // split the key into an array of keys
        // loop through the keys, checking if each level is lazy-loaded
        let currentObject = obj;
        let currentKey;
        for (let i = 0; i < keyArr.length - 1; i++) {
            currentKey = keyArr[i];
            if (typeof currentObject[currentKey] === 'function') {
                currentObject[currentKey] = await currentObject[currentKey]();
            }
            currentObject = currentObject[currentKey];
            if (!exists(currentObject)) break;
        }
        currentKey = keyArr[keyArr.length - 1];
        if (!exists(currentObject) || !exists(currentObject[currentKey])) return;
        delete currentObject[currentKey];
    });
}