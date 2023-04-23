export const unlazy = async (obj) => typeof obj === "function" ? await obj() : obj;
export const unlazyDeep = async (obj) => {
    const unlazyObj = await unlazy(obj);
    for (const key in unlazyObj) {
        const value = unlazyObj[key];
        if (Array.isArray(value)) {
            unlazyObj[key] = await Promise.all(value.map(unlazyDeep));
        }
        else if (typeof value === "function" || typeof value === "object") {
            unlazyObj[key] = await unlazyDeep(value);
        }
    }
    return unlazyObj;
};
//# sourceMappingURL=unlazy.js.map