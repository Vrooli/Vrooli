import { isObject } from "@local/utils";
import { ObjectMap } from "../models";
import { constructUnions } from "./constructUnions";
import { isRelationshipObject } from "./isRelationshipObject";
import { removeCountFields } from "./removeCountFields";
import { removeHiddenFields } from "./removeHiddenFields";
import { removeJoinTables } from "./removeJoinTables";
export function modelToGql(data, partialInfo) {
    const type = partialInfo?.__typename;
    const format = typeof type === "string" ? ObjectMap[type]?.format : undefined;
    if (format) {
        const unionData = constructUnions(data, partialInfo, format.gqlRelMap);
        data = unionData.data;
        partialInfo = unionData.partialInfo;
        data = removeJoinTables(data, format.joinMap);
        data = removeCountFields(data, format.countFields);
        data = removeHiddenFields(data, format.hiddenFields);
    }
    for (const [key, value] of Object.entries(data)) {
        if (!isObject(partialInfo) || !(key in partialInfo)) {
            continue;
        }
        if (Array.isArray(value)) {
            data[key] = data[key].map((v) => modelToGql(v, partialInfo[key]));
        }
        else if (isRelationshipObject(value)) {
            data[key] = modelToGql(value, partialInfo[key]);
        }
    }
    return data;
}
//# sourceMappingURL=modelToGql.js.map