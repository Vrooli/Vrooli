export const setDotNotationValue = (object, notation, value) => {
    if (!object || !notation)
        return null;
    const keys = notation.split(".");
    const lastKey = keys.pop();
    const lastObj = keys.reduce((obj, key) => obj[key] = obj[key] || {}, object);
    lastObj[lastKey] = value;
    return object;
};
//# sourceMappingURL=setDotNotationValue.js.map