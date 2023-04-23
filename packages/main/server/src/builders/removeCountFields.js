import { exists } from "@local/utils";
export const removeCountFields = (obj, countFields) => {
    if (!obj || !countFields)
        return obj;
    const result = {};
    if (!obj._count)
        return obj;
    for (const key of Object.keys(countFields)) {
        const value = key.slice(0, -5);
        if (exists(obj._count[value])) {
            obj[key] = obj._count[value];
        }
    }
    delete obj._count;
    return {
        ...obj,
        ...result,
    };
};
//# sourceMappingURL=removeCountFields.js.map