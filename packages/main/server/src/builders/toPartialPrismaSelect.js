import { ObjectMap } from "../models";
import { addCountFields } from "./addCountFields";
import { addJoinTables } from "./addJoinTables";
import { deconstructUnions } from "./deconstructUnions";
import { isRelationshipObject } from "./isRelationshipObject";
import { removeSupplementalFields } from "./removeSupplementalFields";
export const toPartialPrismaSelect = (partial) => {
    let result = {};
    for (const [key, value] of Object.entries(partial)) {
        if (isRelationshipObject(value)) {
            result[key] = toPartialPrismaSelect(value);
        }
        else {
            result[key] = value;
        }
    }
    const type = partial.__typename;
    const format = typeof type === "string" ? ObjectMap[type]?.format : undefined;
    if (type && format) {
        result = removeSupplementalFields(type, result);
        result = deconstructUnions(result, format.gqlRelMap);
        result = addJoinTables(result, format.joinMap);
        result = addCountFields(result, format.countFields);
    }
    return result;
};
//# sourceMappingURL=toPartialPrismaSelect.js.map