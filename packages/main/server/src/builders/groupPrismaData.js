import { isObject } from "@local/utils";
import pkg from "lodash";
import { isRelationshipObject } from "./isRelationshipObject";
const { merge } = pkg;
const combineDicts = (dict1, dict2) => {
    const result = dict1;
    for (const [childType, childObjects] of Object.entries(dict2.objectTypesIdsDict)) {
        result.objectTypesIdsDict[childType] = result.objectTypesIdsDict[childType] ?? [];
        result.objectTypesIdsDict[childType].push(...childObjects);
    }
    result.selectFieldsDict = merge(result.selectFieldsDict, dict2.selectFieldsDict);
    result.objectIdsDataDict = merge(result.objectIdsDataDict, dict2.objectIdsDataDict);
    return result;
};
export const groupPrismaData = (data, partialInfo) => {
    if (!data || !partialInfo)
        return {
            objectTypesIdsDict: {},
            selectFieldsDict: {},
            objectIdsDataDict: {},
        };
    let result = {
        objectTypesIdsDict: {},
        selectFieldsDict: {},
        objectIdsDataDict: {},
    };
    if (Array.isArray(data)) {
        for (let i = 0; i < data.length; i++) {
            const childDicts = groupPrismaData(data[i], Array.isArray(partialInfo) ? partialInfo[i] : partialInfo);
            result = combineDicts(result, childDicts);
        }
    }
    for (const [key, value] of Object.entries(data)) {
        let childPartialInfo = partialInfo[key];
        if (childPartialInfo)
            if (isObject(childPartialInfo) && Object.keys(childPartialInfo).every(k => k[0] === k[0].toUpperCase())) {
                let matchingType;
                for (const unionType of Object.keys(childPartialInfo)) {
                    if (value.__typename === unionType) {
                        matchingType = unionType;
                        break;
                    }
                }
                if (!matchingType)
                    continue;
                childPartialInfo = childPartialInfo[matchingType];
            }
        if (Array.isArray(value)) {
            for (const v of value) {
                const childDicts = groupPrismaData(v, childPartialInfo);
                result = combineDicts(result, childDicts);
            }
        }
        else if (isRelationshipObject(value)) {
            const childDicts = groupPrismaData(value, childPartialInfo);
            result = combineDicts(result, childDicts);
        }
        else if (key === "id" && partialInfo.__typename) {
            const type = partialInfo.__typename;
            result.objectTypesIdsDict[type] = result.objectTypesIdsDict[type] ?? [];
            result.objectTypesIdsDict[type].push(value);
        }
    }
    const currType = partialInfo?.__typename;
    if (currType) {
        result.selectFieldsDict[currType] = merge(result.selectFieldsDict[currType] ?? {}, partialInfo);
    }
    if (currType && data.id) {
        result.objectIdsDataDict[data.id] = merge(result.objectIdsDataDict[data.id] ?? {}, data);
    }
    for (const [type, ids] of Object.entries(result.objectTypesIdsDict)) {
        result.objectTypesIdsDict[type] = [...new Set(ids)];
    }
    return result;
};
//# sourceMappingURL=groupPrismaData.js.map