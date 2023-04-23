export function getDotNotationValue(obj, keyPath) {
    const keys = keyPath.split(".");
    let currentValue = obj;
    for (const key of keys) {
        if (Array.isArray(currentValue) && /^\d+$/.test(key)) {
            const index = parseInt(key, 10);
            if (index < 0 || index >= currentValue.length) {
                return undefined;
            }
            currentValue = currentValue[index];
        }
        else {
            if (!currentValue || !(key in currentValue)) {
                return undefined;
            }
            currentValue = currentValue[key];
        }
    }
    return currentValue;
}
//# sourceMappingURL=getDotNotationValue.js.map