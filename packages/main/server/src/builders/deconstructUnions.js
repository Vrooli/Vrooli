import { isRelationshipObject } from "./isRelationshipObject";
export const deconstructUnions = (data, gqlRelMap) => {
    const result = data;
    const unionFields = Object.entries(gqlRelMap).filter(([_, value]) => isRelationshipObject(value));
    for (const [key, value] of unionFields) {
        if (!data[key])
            continue;
        const unionData = data[key];
        delete result[key];
        if (!isRelationshipObject(unionData))
            continue;
        for (const [prismaField, type] of Object.entries(value)) {
            if (unionData[type]) {
                result[prismaField] = unionData[type];
            }
        }
    }
    return result;
};
//# sourceMappingURL=deconstructUnions.js.map