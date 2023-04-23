export const removeTypenames = (obj) => {
    return JSON.parse(JSON.stringify(obj, (k, v) => (k === "__typename") ? undefined : v));
};
//# sourceMappingURL=removeTypenames.js.map