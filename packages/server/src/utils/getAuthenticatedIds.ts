import { isRelationshipArray, isRelationshipObject } from "../builders";
import { PrismaUpdate } from "../builders/types";
import { CustomError } from "../events";
import { PrismaRelMap } from "../models/types";
import { PrismaType } from "../types";
import { QueryAction } from "./types";
import pkg from 'lodash';
import { getLogic } from "../getters";
import { GqlModelType } from "@shared/consts";
const { merge } = pkg;
// TODO was originally created for partialselect format. Now must parse data as full Prisma select 
// (i.e. no "type" or "__typename" fields, and all relationships are padded with "select")

/**
 * Helper function to grab ids from an object, and map them to their action and object type. For ids which need 
 * to be authenticated but do not appear in the object (i.e. disconnects), a placeholder is added
 * 
 * Examples:
 * (ActionType.Read, relMap, {
 *     ...
 *     type: 'Project',
 *     id: 'abc123',
 *     parent: {
 *         ...
 *         type: 'Project',
 *         id: 'def456',
 *      }
 * }) => { 
 *     idsByAction: { 'Read': ['abc123', 'def456'] },
 *     idsByType: { 'Project': ['abc123', 'def456'] },
 * }
 * 
 * (ActionType.Update, relMap, {
 *     ...
 *     type: 'Routine',
 *     id: 'abc123',
 *     parentUpdate: {
 *         ...
 *         type: 'Routine',
 *         id: 'def456',
 *         organizationId: 'ghi789',
 *     }
 *     childrenUpdate: [
 *          {
 *              ...  
 *              type: 'Routine',
 *              id: 'jkl012',
 *              grandChildCreate: {
 *                  ... 
 *                  type: 'Routine',
 *                  id: 'mno345',
 *                  greatGrandchildId: 'pqr678',
 *              }
 *          }
 *    ]
 * }) => { 
 *     idsByAction: { 'Connect': ['ghi789', 'pqr678'], 'Disconnect': ['def456.organizationId', 'mo345.greatGrandchildId'], 'Update': ['abc123', 'def456', 'jkl012'] },
 *     idsByType: { 'Routine': ['abc123', 'def456', 'jkl012', 'pqr678', 'def456.organizationId', 'mno345.greatGrandchildId'], 'Organization': ['ghi789'] }
 * }
 */
const objectToIds = <T extends Record<string, any>>(
    actionType: QueryAction,
    relMap: PrismaRelMap<T>,
    object: PrismaUpdate | string,
    languages: string[]
): {
    idsByAction: { [x in QueryAction]?: string[] },
    idsByType: { [key in GqlModelType]?: string[] },
} => {
    // Initialize return objects
    const idsByAction: { [x in QueryAction]?: string[] } = {};
    const idsByType: { [key in GqlModelType]?: string[] } = {};
    // If object is a string, this must be a 'Delete' action, and the string is the id
    if (typeof object === 'string') {
        idsByAction['Delete'] = [object];
        idsByType[relMap.__typename] = [object];
        return { idsByAction, idsByType };
    }
    // If not a 'Create' (i.e. already exists in database), add id of this object to return object
    if (actionType !== 'Create' && object.id) {
        idsByAction[actionType] = [object.id];
        idsByType[relMap.__typename] = [object.id];
    }
    console.log('in objectToIds a', JSON.stringify(object, null, 2));
    // Loop through all keys in relationship map
    Object.keys(relMap).forEach(key => {
        // // Loop through all key variations. Multiple variations can be set at once (e.g. resourcesCreate, resourcesUpdate)
        // const variations = ['', 'Id', 'Ids', 'Connect', 'Create', 'Delete', 'Disconnect', 'Update']; // TODO this part won't work now that we're using PrismaUpdate type
        // variations.forEach(variation => {
        //     // If key is in object
        //     if (`${key}${variation}` in object) {
        //         // Get child relationship map
        //         const childRelMap = getValidator(relMap[key], languages, 'objectToIds').validateMap; //TODO must account for dot notation
        //         // Get child action type
        //         let childActionType: QueryAction = 'Read';
        //         if (actionType !== 'Read') {
        //             switch (variation) {
        //                 case 'Id':
        //                 case 'Ids':
        //                 case 'Connect':
        //                     childActionType = 'Connect';
        //                     break;
        //                 case 'Create':
        //                     childActionType = 'Create';
        //                     break;
        //                 case 'Delete':
        //                     childActionType = 'Delete';
        //                     break;
        //                 case 'Disconnect':
        //                     childActionType = 'Disconnect';
        //                     break;
        //                 case 'Update':
        //                     childActionType = 'Update';
        //                     break;
        //                 default:
        //                     childActionType = actionType;
        //             }
        //         }
        //         // Get value of key
        //         const value = object[`${key}${variation}`];
        //         // If value is an array of objects
        //         if (isRelationshipArray(value)) {
        //             // 'Connect', 'Delete', and 'Disconnect' are invalid for object arrays
        //             if (['Connect', 'Delete', 'Disconnect'].includes(childActionType)) {
        //                 throw new CustomError('0282', 'InternalError', languages)
        //             }
        //             // Recursively call objectToIds on each object in array
        //             value.forEach((item: any) => {
        //                 const nestedIds = objectToIds(childActionType as 'Create' | 'Read' | 'Update', childRelMap as any, item, languages);
        //                 Object.keys(nestedIds).forEach(nestedKey => {
        //                     if (ids[nestedKey]) { ids[nestedKey] = [...ids[nestedKey], ...nestedIds[nestedKey]]; }
        //                     else { ids[nestedKey] = nestedIds[nestedKey]; }
        //                 });
        //             });
        //         }
        //         // If value is an array of ids
        //         else if (Array.isArray(value)) {
        //             // Only 'Connect', 'Delete', and 'Disconnect' are valid for id arrays
        //             if (!['Connect', 'Delete', 'Disconnect'].includes(childActionType)) {
        //                 throw new CustomError('0283', 'InternalError', languages)
        //             }
        //             // Add ids to return object
        //             if (ids[relMap[key]]) { ids[relMap[key]] = [...(ids[relMap[key]] ?? []), ...value.map(id => `${childActionType}-${id}`)]; }
        //         }
        //         // If value is a single object
        //         else if (isRelationshipObject(value)) {
        //             // 'Connect', 'Delete', and 'Disconnect' are invalid for objects
        //             if (['Connect', 'Delete', 'Disconnect'].includes(childActionType)) {
        //                 throw new CustomError('0284', 'InternalError', languages)
        //             }
        //             // Recursively call objectToIds on object
        //             const nestedIds = objectToIds(childActionType as 'Create' | 'Read' | 'Update', childRelMap as any, value as any, languages);
        //             Object.keys(nestedIds).forEach(nestedKey => {
        //                 if (ids[nestedKey]) { ids[nestedKey] = [...ids[nestedKey], ...nestedIds[nestedKey]]; }
        //                 else { ids[nestedKey] = nestedIds[nestedKey]; }
        //             });
        //         }
        //         // If value is a single id
        //         else if (typeof value === 'string') {
        //             // Only 'Connect', 'Delete', and 'Disconnect' are valid for ids
        //             if (!['Connect', 'Delete', 'Disconnect'].includes(childActionType)) {
        //                 throw new CustomError('0285', 'InternalError', languages)
        //             }
        //             // Add id to return object
        //             if (ids[relMap[key]]) { ids[relMap[key]] = [...(ids[relMap[key]] ?? []), `${childActionType}-${value}`]; }
        //             // If child action is a 'Connect' and parent action is 'Update', add 'Disconnect' placeholder. 
        //             // This id will have to be queried for later. 
        //             // NOTE: This is the only place where this check must be done. This is because one-to-one relationships 
        //             // automatically disconnect the old object when a new one is connected.
        //             else if (childActionType === 'Connect' && actionType === 'Update') {
        //                 ids[relMap[key]] = [`Disconnect-${value}.${key}${variation}`];
        //             }
        //         }
        //     }
        // });
        // Check if relation is in object
        if (object[key]) {
            
        }
    });
    return { idsByAction, idsByType };
};

/**
 * Finds all placeholders in an idsByAction object. 
 * Placeholder examples: 'Create-asd123', 'Disconnect-def456.organizationId', 'Disconnect-mno345.greatGrandchildId']
 */

/**
 * Helper function to convert disconnect placeholders to disconnect ids. 
 * Example: { 'Routine': ['Create-asd123', 'Disconnect-def456.organizationId', 'Disconnect-mno345.greatGrandchildId'] } => { 'Routine': ['Create-asd123', 'Disconnect-someid', 'Disconnect-someotherid'] }
 * @param idActions Map of GqlModelType to ${actionType}-${id}?.${key}${variation}. Anything with a '.' is a placeholder
 * @param prisma Prisma client
 * @param languages Array of languages to use for error messages
 * @returns Map with placeholders replaced with ids
 */
const placeholdersToIds = async (idActions: { [key in GqlModelType]?: string[] }, prisma: PrismaType, languages: string[]): Promise<{ [key in GqlModelType]?: string[] }> => {
    // Initialize object to hold prisma queries
    const queries: { [key in GqlModelType]?: { ids: string[], select: { [x: string]: any } } } = {};
    // Loop through all keys in ids
    Object.keys(idActions).forEach(key => {
        // Loop through all ids in key
        idActions[key as any].forEach((idAction: string) => {
            // If id is a placeholder, add to placeholderIds
            if (idAction.includes('.')) {
                queries[key as any] = queries[key as any] ?? { ids: [], select: {} };
                // Get id, which is everything after the first '-' and before '.'
                const id = idAction.split('-')[1].split('.')[0];
                queries[key as any].ids.push(id);
                // Get placeholder, which is everything after '.'
                const placeholder = idAction.split('.')[1];
                // Add placeholder to select object
                queries[key as any].select[id] = { [placeholder]: true };
            }
        });
    });
    // If there are no placeholders, return ids to save time
    if (Object.keys(queries).length === 0) return idActions;
    // Initialize object to hold query results
    const queryData: { [key in GqlModelType]?: { [x: string]: any } } = {};
    // Loop through all keys in queries
    for (const key in Object.keys(queries)) {
        // If there are any no ids, skip
        if (queries[key as any].ids.length === 0) continue;
        // Query for ids
        const { delegate } = getLogic(['delegate'], key as GqlModelType, languages, 'disconnectPlaceholdersToIds');
        queryData[key as any] = await delegate(prisma).findMany({
            where: { id: { in: queries[key as any].ids } },
            select: queries[key as any].select,
        });
    }
    // Initialize return object
    const result: { [key in GqlModelType]?: string[] } = {};
    // Loop through all keys in idActions
    Object.keys(idActions).forEach(key => {
        // Get queryData for this key
        const data: { [x: string]: any }[] | undefined = queryData[key as any];
        // Loop through all ids in key
        idActions[key as any].forEach((idAction: string) => {
            // If id is a placeholder, replace with id from results
            if (data && idAction.includes('.')) {
                // Get id, which is everything after the first '-' and before '.'
                const id = idAction.split('-')[1].split('.')[0];
                // Get placeholder, which is everything after '.'
                const placeholder = idAction.split('.')[1];
                // Find data object with matching id, and use placeholder to get disconnect id
                const parent = data.find(d => d.id === id) as { [x: string]: any };
                const disconnectId = parent[placeholder];
                // Add disconnect id to result
                result[key as any] = result[key as any] ?? [];
                result[key as any].push(`Disconnect-${disconnectId}`);
            }
            // Otherwise, add to result
            else { result[key as any] = result[key as any] ?? []; result[key as any].push(idAction); }
        });
    });
    return result;
};

/**
 * Finds all ids of objects in a crud request that need to be checked for permissions. 
 * In certain cases (i.e. disconnects), 
 * ids are not included in the request, and must be queried for.
 * @returns IDs organized both by action type and GqlModelType
 */
export const getAuthenticatedIds = async (
    objects: {
        actionType: QueryAction,
        data: string | PrismaUpdate
    }[],
    objectType: `${GqlModelType}`,
    prisma: PrismaType,
    languages: string[]
): Promise<{
    idsByType: { [key in GqlModelType]?: string[] }
    idsByAction: { [key in QueryAction]?: string[] }
}> => {
    console.log('getauthenticatedids start', JSON.stringify(objects));
    // Initialize return objects
    let idsByType: { [key in GqlModelType]?: string[] } = {};
    let idsByAction: { [key in QueryAction]?: string[] } = {};
    // Find validator for this object type
    const { format } = getLogic(['format'], objectType, languages, 'getAuthenticatedIds');
    // Filter out objects that are strings, since the rest of the function only works with objects
    const filteredObjects = objects.filter(object => {
        const isString = typeof object.data === 'string';
        if (isString) {
            idsByType[objectType] = [...(idsByType[objectType] ?? []), object.data as string];
            // Every string must be a delete. This is because connect and disconnect are only 
            // used for relations (i.e. not top-level), and the other types always use objects.
            idsByAction['Delete'] = [...(idsByAction['Delete'] ?? []), object.data as string];
        }
        return !isString;
    });
    console.log('getauthenticatedids filteredobjects', JSON.stringify(filteredObjects));
    console.log('getauthenticatedids idsbytype', JSON.stringify(idsByType));
    console.log('getauthenticatedids idsbyaction', JSON.stringify(idsByAction));
    // For each object
    filteredObjects.forEach(object => {
        // Call objectToIds to get ids of all objects requiring authentication. 
        // For implicit IDs (i.e. 'Connect' and 'Disconnect'), this will return placeholders
        // that we can use to query for the actual ids.
        const { idsByAction: childIdsByAction, idsByType: childIdsByType } = objectToIds(object.actionType, format.prismaRelMap, object.data as PrismaUpdate, languages); //TODO validateMap must account  for dot notation
        // Merge idsByAction and idsByType with childIdsByAction and childIdsByType
        idsByAction = merge(idsByAction, childIdsByAction);
        idsByType = merge(idsByType, childIdsByType);
    });
    // Now we should have ALMOST all ids of objects requiring authentication. What's missing are the ids of 
    // objects that are being implicitly disconnected (i.e. being replaced by a new connect). We need to find
    // these ids by querying for them TODO
    // result = await placeholdersToIds(result, prisma, languages);
    // Return result
    console.log('getAuthenticatedIds end')
    return { idsByType, idsByAction };
}