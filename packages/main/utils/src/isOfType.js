export const isOfType = (obj, ...types) => {
    if (!obj || !obj.__typename)
        return false;
    return types.includes(obj.__typename);
};
//# sourceMappingURL=isOfType.js.map