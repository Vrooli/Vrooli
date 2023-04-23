import { GqlModelType } from "@local/consts";
import { CustomError } from "../events";
import { getLogic } from "../getters";
import { isRelationshipObject } from "./isRelationshipObject";
const removeFirstDotLayer = (arr) => arr.map(x => x.split(".").slice(1).join(".")).filter(x => x !== "");
export const permissionsSelectHelper = (mapResolver, userId, languages, recursionDepth = 0, omitFields = []) => {
    if (recursionDepth > 100) {
        throw new CustomError("0386", "InternalError", languages ?? ["en"], { userId, recursionDepth });
    }
    const map = typeof mapResolver === "function" ? mapResolver(userId, languages) : mapResolver;
    const result = {};
    for (const key of Object.keys(map)) {
        if (omitFields.includes(key)) {
            continue;
        }
        const value = map[key];
        if (Array.isArray(value)) {
            if (value.length === 2 && typeof value[0] === "string" && value[0] in GqlModelType && Array.isArray(value[1])) {
                const { validate } = getLogic(["validate"], value[0], languages, "permissionsSelectHelper");
                if (!validate) {
                    result[key] = value;
                }
                else {
                    const childOmitFields = removeFirstDotLayer(omitFields).concat(value[1]);
                    const childMap = validate.permissionsSelect(userId, languages);
                    if (childMap) {
                        result[key] = { select: permissionsSelectHelper(childMap, userId, languages, recursionDepth + 1, childOmitFields) };
                    }
                }
            }
            else {
                const childOmitFields = removeFirstDotLayer(omitFields);
                result[key] = value.map((x) => permissionsSelectHelper(x, userId, languages, recursionDepth + 1), childOmitFields);
            }
        }
        else if (isRelationshipObject(value)) {
            const childOmitFields = removeFirstDotLayer(omitFields);
            result[key] = permissionsSelectHelper(value, userId, languages, recursionDepth + 1, childOmitFields);
        }
        else if (typeof value === "string" && value in GqlModelType) {
            const { validate } = getLogic(["validate"], value, languages, "permissionsSelectHelper");
            if (!validate) {
                result[key] = value;
            }
            else {
                const childOmitFields = removeFirstDotLayer(omitFields);
                const childMap = validate.permissionsSelect(userId, languages);
                if (childMap) {
                    result[key] = { select: permissionsSelectHelper(childMap, userId, languages, recursionDepth + 1, childOmitFields) };
                }
            }
        }
        else {
            result[key] = value;
        }
    }
    return result;
};
//# sourceMappingURL=permissionsSelectHelper.js.map