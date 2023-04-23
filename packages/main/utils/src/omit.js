export function omit(obj, keysToOmit) {
    const result = { ...obj };
    for (const key of keysToOmit) {
        const path = key.split(".");
        let current = result;
        for (let i = 0; i < path.length - 1; i++) {
            current = current[path[i]];
        }
        delete current[path[path.length - 1]];
        let currentKey = path[0];
        for (let i = 1; i < path.length; i++) {
            if (Object.keys(current).length === 0) {
                delete result[currentKey];
                break;
            }
            current = current[path[i]];
            currentKey = currentKey + "." + path[i];
        }
    }
    return result;
}
//# sourceMappingURL=omit.js.map