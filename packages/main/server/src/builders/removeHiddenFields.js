export const removeHiddenFields = (obj, fields) => {
    if (!fields)
        return obj;
    const result = {};
    for (const [key, value] of Object.entries(obj)) {
        if (fields.includes(key))
            continue;
        if (typeof value === "object") {
            const nestedFields = fields.filter((field) => field.startsWith(`${key}.`)).map((field) => field.replace(`${key}.`, ""));
            if (nestedFields.length > 0) {
                result[key] = removeHiddenFields(value, nestedFields);
            }
            else {
                result[key] = value;
            }
        }
        else
            result[key] = value;
    }
    return result;
};
//# sourceMappingURL=removeHiddenFields.js.map