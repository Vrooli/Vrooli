import { getDotNotationValue, setDotNotationValue } from "@local/utils";
import { ObjectMap } from "../models";
function getKeyPaths(obj, parentKey) {
    let keys = [];
    for (const key in obj) {
        const currentKey = parentKey ? `${parentKey}.${key}` : key;
        if (typeof obj[key] === "object" && !Array.isArray(obj[key])) {
            keys = keys.concat(getKeyPaths(obj[key], currentKey));
        }
        else {
            keys.push(currentKey);
        }
    }
    return keys;
}
export const addSupplementalFieldsHelper = async ({ languages, objects, objectType, partial, prisma, userData }) => {
    if (!objects || objects.length === 0)
        return [];
    const supplementer = ObjectMap[objectType]?.format?.supplemental;
    if (!supplementer)
        return objects;
    const ids = objects.map(({ id }) => id);
    const supplementalData = await supplementer.toGraphQL({ ids, languages, objects, partial, prisma, userData });
    const supplementalDotFields = getKeyPaths(supplementalData);
    for (let i = 0; i < objects.length; i++) {
        for (const field of supplementalDotFields) {
            const dotField = `${field}.${i}`;
            const suppValue = getDotNotationValue(supplementalData, dotField);
            setDotNotationValue(objects[i], field, suppValue);
        }
    }
    return objects;
};
//# sourceMappingURL=addSupplementalFieldsHelper.js.map