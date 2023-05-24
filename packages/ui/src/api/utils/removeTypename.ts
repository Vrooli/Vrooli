/**
 * Removes "__typename" and "type" fields from the object, as these are only used 
 * for caching and determining the type of the object. They must not be passed 
 * into queries or mutations.
 */
export const removeTypename = (value: any): any => {
    if (value === null || value === undefined) return value;
    if (Array.isArray(value)) {
        return value.map(v => removeTypename(v));
    }
    if (typeof value === "object") {
        const newObj = {};
        Object.keys(value).forEach(key => {
            if (key !== "__typename" && key !== "type") {
                newObj[key] = removeTypename(value[key]);
            }
        });
        return newObj;
    }
    return value;
};
