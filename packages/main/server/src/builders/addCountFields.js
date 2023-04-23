export const addCountFields = (obj, countFields) => {
    if (!countFields)
        return obj;
    const result = {};
    for (const key of Object.keys(countFields)) {
        if (obj[key]) {
            if (!obj._count)
                obj._count = {};
            const value = key.slice(0, -5);
            obj._count[value] = true;
            delete obj[key];
        }
    }
    return {
        ...obj,
        ...result,
    };
};
//# sourceMappingURL=addCountFields.js.map