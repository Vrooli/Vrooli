import { isRelationshipObject } from "./isRelationshipObject";
export const removeJoinTables = (obj, map) => {
    if (!obj || !map)
        return obj;
    const result = {};
    for (const [key, value] of Object.entries(map)) {
        if (obj[key]) {
            if (Array.isArray(obj[key])) {
                if (obj[key].every((o) => isRelationshipObject(o) && Object.keys(o).length === 1 && Object.keys(o)[0] !== "id")) {
                    result[key] = obj[key].map((item) => item[value]);
                }
            }
            else {
                if (isRelationshipObject(obj[key]) && Object.keys(obj[key]).length === 1 && Object.keys(obj[key])[0] !== "id") {
                    result[key] = obj[key][value];
                }
            }
        }
    }
    return {
        ...obj,
        ...result,
    };
};
//# sourceMappingURL=removeJoinTables.js.map