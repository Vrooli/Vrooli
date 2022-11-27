import { isRelationshipArray, isRelationshipObject } from "../builders";
import { PrismaUpdate } from "../builders/types";
import { CustomError } from "../events";
import { getDelegator, getValidator } from "../getters";
import { GraphQLModelType } from "../models/types";
import { PrismaType } from "../types";
import { QueryAction } from "./types";
// TODO was originally created for partialselect format. Now must parse data as full Prisma select

/**
 * Helper function to grab ids from an object, and map them to their action and object type. For ids which need 
 * to be authenticated but do not appear in the object (e.g. disconnects), a placeholder is added
 * 
 * Examples:
 * (ActionType.Read, relMap, {
 *     ...
 *     __typename: 'Project',
 *     id: 'abc123',
 *     parent: {
 *         ...
 *         __typename: 'Project',
 *         id: 'def456',
 *      }
 * }) => { 'Project': ['Read-abc123', 'Read-def456'] }
 * 
 * (ActionType.Update, relMap, {
 *     ...
 *     __typename: 'Routine',
 *     id: 'abc123',
 *     parentUpdate: {
 *         ...
 *         __typename: 'Routine',
 *         id: 'def456',
 *         organizationId: 'ghi789',
 *     }
 *     childrenUpdate: [
 *          {
 *              ...  
 *              __typename: 'Routine',
 *              id: 'jkl012',
 *              grandChildCreate: {
 *                  ... 
 *                  __typename: 'Routine',
 *                  id: 'mno345',
 *                  greatGrandchildId: 'pqr678',
 *              }
 *          }
 *    ]
 * }) => { 'Routine': ['Update-abc123', 'Update-def456', 'Update-jkl012', 'Connect-ghi789', 'Connect-pqr678', 'Disconnect-def456.organizationId', 'Disconnect-mno345.greatGrandchildId'] }
 */
const objectToIds = ( // TODO doesn't support versioned objects
    actionType: QueryAction,
    relMap: { [x: string]: GraphQLModelType } & { __typename: GraphQLModelType },
    object: PrismaUpdate | string,
    languages: string[]
): { [key in GraphQLModelType]?: string[] } => {
    // Initialize return object
    const ids: { [key in GraphQLModelType]?: string[] } = {};
    // If object is a string, this must be a 'Delete' action, and the string is the id
    if (typeof object === 'string') {
        ids[relMap.__typename] = [`Delete-${object}`];
        return ids;
    }
    // If not a 'Create', add id of this object to return object
    if (actionType !== 'Create' && object.id) { ids[relMap.__typename] = [`${actionType}-${object.id}`]; }
    // Loop through all keys in relationship map
    Object.keys(relMap).forEach(key => {
        // Loop through all key variations. Multiple variations can be set at once (e.g. resourcesCreate, resourcesUpdate)
        const variations = ['', 'Id', 'Ids', 'Connect', 'Create', 'Delete', 'Disconnect', 'Update']; // TODO this part won't work now that we're using PrismaUpdate type
        variations.forEach(variation => {
            // If key is in object
            if (`${key}${variation}` in object) {
                // Get child relationship map
                const childRelMap = getValidator(relMap[key], languages, 'objectToIds').validateMap; //TODO must account for dot notation
                // Get child action type
                let childActionType: QueryAction = 'Read';
                if (actionType !== 'Read') {
                    switch (variation) {
                        case 'Id':
                        case 'Ids':
                        case 'Connect':
                            childActionType = 'Connect';
                            break;
                        case 'Create':
                            childActionType = 'Create';
                            break;
                        case 'Delete':
                            childActionType = 'Delete';
                            break;
                        case 'Disconnect':
                            childActionType = 'Disconnect';
                            break;
                        case 'Update':
                            childActionType = 'Update';
                            break;
                        default:
                            childActionType = actionType;
                    }
                }
                // Get value of key
                const value = object[`${key}${variation}`];
                // If value is an array of objects
                if (isRelationshipArray(value)) {
                    // 'Connect', 'Delete', and 'Disconnect' are invalid for object arrays
                    if (['Connect', 'Delete', 'Disconnect'].includes(childActionType)) {
                        throw new CustomError('0282', 'InternalError', languages)
                    }
                    // Recursively call objectToIds on each object in array
                    value.forEach((item: any) => {
                        const nestedIds = objectToIds(childActionType as 'Create' | 'Read' | 'Update', childRelMap as any, item);
                        Object.keys(nestedIds).forEach(nestedKey => {
                            if (ids[nestedKey]) { ids[nestedKey] = [...ids[nestedKey], ...nestedIds[nestedKey]]; }
                            else { ids[nestedKey] = nestedIds[nestedKey]; }
                        });
                    });
                }
                // If value is an array of ids
                else if (Array.isArray(value)) {
                    // Only 'Connect', 'Delete', and 'Disconnect' are valid for id arrays
                    if (!['Connect', 'Delete', 'Disconnect'].includes(childActionType)) {
                        throw new CustomError('0283', 'InternalError', languages)
                    }
                    // Add ids to return object
                    if (ids[relMap[key]]) { ids[relMap[key]] = [...(ids[relMap[key]] ?? []), ...value.map(id => `${childActionType}-${id}`)]; }
                }
                // If value is a single object
                else if (isRelationshipObject(value)) {
                    // 'Connect', 'Delete', and 'Disconnect' are invalid for objects
                    if (['Connect', 'Delete', 'Disconnect'].includes(childActionType)) {
                        throw new CustomError('0284', 'InternalError', languages)
                    }
                    // Recursively call objectToIds on object
                    const nestedIds = objectToIds(childActionType as 'Create' | 'Read' | 'Update', childRelMap as any, value);
                    Object.keys(nestedIds).forEach(nestedKey => {
                        if (ids[nestedKey]) { ids[nestedKey] = [...ids[nestedKey], ...nestedIds[nestedKey]]; }
                        else { ids[nestedKey] = nestedIds[nestedKey]; }
                    });
                }
                // If value is a single id
                else if (typeof value === 'string') {
                    // Only 'Connect', 'Delete', and 'Disconnect' are valid for ids
                    if (!['Connect', 'Delete', 'Disconnect'].includes(childActionType)) {
                        throw new CustomError('0285', 'InternalError', languages)
                    }
                    // Add id to return object
                    if (ids[relMap[key]]) { ids[relMap[key]] = [...(ids[relMap[key]] ?? []), `${childActionType}-${value}`]; }
                    // If child action is a 'Connect' and parent action is 'Update', add 'Disconnect' placeholder. 
                    // This id will have to be queried for later. 
                    // NOTE: This is the only place where this check must be done. This is because one-to-one relationships 
                    // automatically disconnect the old object when a new one is connected.
                    else if (childActionType === 'Connect' && actionType === 'Update') {
                        ids[relMap[key]] = [`Disconnect-${value}.${key}${variation}`];
                    }
                }
            }
        });
    });
    return ids;
};

/**
 * Helper function to convert disconnect placeholders to disconnect ids. 
 * Example: { 'Routine': ['Create-asd123', 'Disconnect-def456.organizationId', 'Disconnect-mno345.greatGrandchildId'] } => { 'Routine': ['Create-asd123', 'Disconnect-someid', 'Disconnect-someotherid'] }
 * @param idActions Map of GraphQLModelType to ${actionType}-${id}?.${key}${variation}. Anything with a '.' is a placeholder
 * @param prisma Prisma client
 * @param languages Array of languages to use for error messages
 * @returns Map with placeholders replaced with ids
 */
const placeholdersToIds = async (idActions: { [key in GraphQLModelType]?: string[] }, prisma: PrismaType, languages: string[]): Promise<{ [key in GraphQLModelType]?: string[] }> => {
    // Initialize object to hold prisma queries
    const queries: { [key in GraphQLModelType]?: { ids: string[], select: { [x: string]: any } } } = {};
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
    const queryData: { [key in GraphQLModelType]?: { [x: string]: any } } = {};
    // Loop through all keys in queries
    for (const key in Object.keys(queries)) {
        // If there are any no ids, skip
        if (queries[key as any].ids.length === 0) continue;
        // Query for ids
        const delegate = getDelegator(key as GraphQLModelType, prisma, languages, 'disconnectPlaceholdersToIds');
        queryData[key as any] = await delegate.findMany({
            where: { id: { in: queries[key as any].ids } },
            select: queries[key as any].select,
        });
    }
    // Initialize return object
    const result: { [key in GraphQLModelType]?: string[] } = {};
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
 * Finds all ids of objects in a crud request that need to be checked for permissions. In certain cases, 
 * ids are not included in the request, and must be queried for.
 * @returns IDs organized both by action type and GraphQLModelType
 */
export const getAuthenticatedIds = async (
    objects: { 
        actionType: QueryAction, 
        data: string | PrismaUpdate
    }[],
    objectType: GraphQLModelType,
    prisma: PrismaType,
    languages: string[]
): Promise<{
    idsByType: { [key in GraphQLModelType]?: string[] }
    idsByAction: { [key in QueryAction]?: string[] }
}> => {
    // Initialize return objects
    let idsByType: { [key in GraphQLModelType]?: string[] } = {};
    let idsByAction: { [key in QueryAction]?: string[] } = {};
    // Find validator and prisma delegate for this object type
    const validator = getValidator(objectType, languages, 'getAuthenticatedIds');
    // Filter out objects that are strings
    const filteredObjects = objects.filter(object => {
        const isString = typeof object.data === 'string';
        if (isString) {
            idsByType[objectType] = [...(idsByType[objectType] ?? []), object.data as string];
            idsByAction['Delete'] = [...(idsByAction['Delete'] ?? []), object.data as string];
        }
        return !isString;
    });
    // For each object
    filteredObjects.forEach(object => {
        // Call objectToIds to get ids of all objects requiring authentication
        const ids = objectToIds(object.actionType, validator.validateMap as any, object.data as PrismaUpdate, languages); //TODO validateMap must account  for dot notation
        // Add ids to return object
        Object.keys(ids).forEach(key => {
            if (result[key]) { result[key] = [...result[key], ...ids[key]]; }
            else { result[key] = ids[key]; }
        });
    });
    // Now we should have ALMOST all ids of objects requiring authentication. What's missing are the ids of 
    // objects that are being implicitly disconnected (i.e. being replaced by a new connect). We need to find
    // these ids by querying for them
    result = await placeholdersToIds(result, prisma, languages);
    // Return result
    return result;
}