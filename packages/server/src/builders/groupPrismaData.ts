import { GqlModelType } from "@shared/consts";
import { isObject } from "@shared/utils";
import pkg from 'lodash';
import { SingleOrArray } from "../types";
import { isRelationshipObject } from "./isRelationshipObject";
import { PartialGraphQLInfo } from "./types";
const { merge } = pkg;

type GroupPrismaDataReturn = {
    objectTypesIdsDict: { [x: string]: string[] },
    selectFieldsDict: { [x: string]: { [x: string]: any } },
    objectIdsDataDict: { [x: string]: ({ id: string } & { [x: string]: any }) },
}

/**
 * Helper method to combine dictionaries returned by groupPrismaData
 * @param dict1
 * @param dict2
 * @returns dict1 and dict2 combined
 */
const combineDicts = (dict1: GroupPrismaDataReturn, dict2: GroupPrismaDataReturn): GroupPrismaDataReturn => {
    // Initialize result
    let result: GroupPrismaDataReturn = dict1;
    // Update objectTypesIdsDict
    for (const [childType, childObjects] of Object.entries(dict2.objectTypesIdsDict)) {
        result.objectTypesIdsDict[childType] = result.objectTypesIdsDict[childType] ?? [];
        result.objectTypesIdsDict[childType].push(...childObjects);
    }
    // Update selectFieldsDict
    result.selectFieldsDict = merge(result.selectFieldsDict, dict2.selectFieldsDict);
    // Update objectIdsDataDict
    result.objectIdsDataDict = merge(result.objectIdsDataDict, dict2.objectIdsDataDict);
    return result;
}

/**
 * Combines fields from a Prisma object with arbitrarily nested relationships. Used to get 
 * the following information: 
 * 1. The IDs of all objects in the data array, grouped by type
 * 2. The select fields for each type, unioned if they show up in multiple places
 * 3. The known data for each object, by ID. This is helpful for: 
 *     a. Finding the data for a given object, when we use the ID to look it up
 *     b. Combining known data if the same object appears in multiple places in the data
 * @param data GraphQL-shaped data, where each object contains at least an ID
 * @param partialInfo PartialGraphQLInfo object
 */
export const groupPrismaData = (
    data: SingleOrArray<{ [x: string]: any }>,
    partialInfo: SingleOrArray<PartialGraphQLInfo>
): GroupPrismaDataReturn => {
    // Check for valid input
    if (!data || !partialInfo) return {
        objectTypesIdsDict: {},
        selectFieldsDict: {},
        objectIdsDataDict: {},
    }
    // Initialize dictionaries
    let result: GroupPrismaDataReturn = {
        objectTypesIdsDict: {},
        selectFieldsDict: {},
        objectIdsDataDict: {},
    }
    // If data is an array, loop through each element
    if (Array.isArray(data)) {
        for (let i = 0; i < data.length; i++) {
            // Pass each element through groupPrismaData
            const childDicts = groupPrismaData(data[i], Array.isArray(partialInfo) ? partialInfo[i] : partialInfo);
            // Update result with childDicts
            result = combineDicts(result, childDicts);
        }
    }
    // Loop through each key/value pair in data
    for (const [key, value] of Object.entries(data)) {
        let childPartialInfo: PartialGraphQLInfo = partialInfo[key] as any;
        if (childPartialInfo)
            // If every key in childPartialInfo starts with a capital letter, then it is a union.
            // In this case, we must determine which union to use based on the shape of value
            if (isObject(childPartialInfo) && Object.keys(childPartialInfo).every(k => k[0] === k[0].toUpperCase())) {
                console.log('groupprismadata union check', key, JSON.stringify(value), '\n\n', JSON.stringify(childPartialInfo), '\n\n');
                // Find the union type which matches the shape of value
                let matchingType: string | undefined;
                for (const unionType of Object.keys(childPartialInfo)) {
                    if (value.__typename === unionType) {
                        matchingType = unionType;
                        break;
                    }
                }
                // If no union type matches, skip
                if (!matchingType) continue;
                // If union type, update child partial
                childPartialInfo = childPartialInfo[matchingType] as PartialGraphQLInfo;
            }
        // If value is an array
        if (Array.isArray(value)) {
            // Pass each element through groupPrismaData
            for (const v of value) {
                const childDicts = groupPrismaData(v, childPartialInfo);
                // Update result with childDicts
                result = combineDicts(result, childDicts);
            }
        }
        // If value is an object (and not date)
        else if (isRelationshipObject(value)) {
            // Pass value through groupPrismaData
            const childDicts = groupPrismaData(value, childPartialInfo);
            // Update result with childDicts
            result = combineDicts(result, childDicts);
        }
        // If key is 'id'
        else if (key === 'id' && (partialInfo as PartialGraphQLInfo).__typename) {
            const type = (partialInfo as PartialGraphQLInfo).__typename as `${GqlModelType}`;
            // Add to objectTypesIdsDict
            result.objectTypesIdsDict[type] = result.objectTypesIdsDict[type] ?? [];
            result.objectTypesIdsDict[type].push(value);
        }
    }
    // Add keys to selectFieldsDict
    const currType = (partialInfo as PartialGraphQLInfo)?.__typename as `${GqlModelType}`;
    if (currType) {
        result.selectFieldsDict[currType] = merge(result.selectFieldsDict[currType] ?? {}, partialInfo);
    }
    // Add data to objectIdsDataDict
    if (currType && data.id) {
        result.objectIdsDataDict[data.id] = merge(result.objectIdsDataDict[data.id] ?? {}, data);
    }
    // Before returning, remove duplicate IDs from objectTypesIdsDict
    for (const [type, ids] of Object.entries(result.objectTypesIdsDict)) {
        result.objectTypesIdsDict[type] = [...new Set(ids)];
    }
    // Return dictionaries
    return result;
}