import { isRelationshipArray } from "./isRelationshipArray";
import { isRelationshipObject } from "./isRelationshipObject";
export const addJoinTables = (partialInfo, map) => {
    if (!map)
        return partialInfo;
    const result = {};
    for (const [key, value] of Object.entries(map)) {
        if (partialInfo[key]) {
            if (isRelationshipArray(partialInfo[key])) {
                if (partialInfo[key].every((o) => isRelationshipObject(o) && Object.keys(o).length === 1 && Object.keys(o)[0] !== "id")) {
                    result[key] = partialInfo[key];
                    continue;
                }
            }
            else if (isRelationshipObject(partialInfo[key])) {
                if (Object.keys(partialInfo[key]).length === 1 && Object.keys(partialInfo[key])[0] !== "id") {
                    result[key] = partialInfo[key];
                    continue;
                }
            }
            result[key] = { [value]: partialInfo[key] };
        }
    }
    return {
        ...partialInfo,
        ...result,
    };
};
//# sourceMappingURL=addJoinTables.js.map