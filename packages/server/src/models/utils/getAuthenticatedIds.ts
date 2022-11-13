import { CODE } from "@shared/consts";
import { CustomError } from "../../events";
import { PrismaType } from "../../types";
import { isRelationshipArray, isRelationshipObject } from "../builder";
import { GraphQLModelType, PartialPrismaSelect } from "../types";
import { getValidator } from "./getValidator";

/**
 * Helper function to grab ids from an object, and map them to their object type
 * 
 * Examples:
 * (relMap, {
 *     ...
 *     id: 'abc123',
 *     parent: {
 *         ...
 *         id: 'def456',
 *      }
 * }) => { [objectType]: ['abc123'], [relMap.parent]: ['def456'] }
 * 
 * (relMap, {
 *     ...
 *     id: 'abc123',
 *     parent: {
 *         ...
 *         id: 'def456',
 *         organizationId: 'ghi789',
 *     }
 *     children: [
 *          {
 *              ...  
 *              id: 'jkl012',
 *              grandChild: {
 *                  ... 
 *                  id: 'mno345',
 *                  greatGrandchildId: 'pqr678',
 *              }
 *          }
 *    ]
 * }) => { 
 *  [objectType]: ['abc123'], 
 *  [relMap.parent]: ['def456'], 
 *  [relMap.parent.organization]: ['ghi789'], 
 *  [relMap.children]: ['jkl012'], 
 *  [relMap.children.grandChild]: ['mno345'], 
 *  [relMap.chilren.grandChild.greatGrandchild]: ['pqr678'] 
 * }
 */
// const objectToIds = <GQLCreate extends { [x: string]: any }, GQLUpdate extends { id?: string }>(
//     relMap: { [x: string]: GraphQLModelType } & { __typename: GraphQLModelType },
//     object: GQLCreate | GQLUpdate,
// ): { [x: string]: string[] } => {
//     // Initialize return object
//     const ids: { [x: string]: string[] } = {};
//     // Add id of this object to return object
//     if (object.id) { ids[relMap.__typename] = [object.id]; }
//     // Loop through all keys in object
//     Object.keys(object).forEach(key => {
//         // Skip __typename
//         if (key === '__typename') return;
//         // If key is a relationship
//         const keyOptions = [key, `${key}Id`, `${key}Ids`, `${key}Connect`, `${key}Create`, `${key}Update`, `${key}Upsert`, `${key}Delete`, `${key}Disconnect`];
//         if (keyOptions.some(keyOption => keyOption in relMap)) {
//             // Get relMap of this relationship
//             const childRelMap = getValidator(relMap[key], 'objectToIds').validatedRelationshipMap;
//             // If relationship object is an array of objects
//             if (isRelationshipArray(object[key])) {
//                 // Loop through all objects in array
//                 object[key].forEach((item: any) => {
//                     // Add id of relationship to return object
//                     if (item.id) { ids[relMap[key]] = [item.id]; }
//                     // Add nested ids to return object
//                     const nestedIds = objectToIds(childRelMap, item);
//                     Object.keys(nestedIds).forEach(nestedKey => {
//                         if (ids[nestedKey]) { ids[nestedKey] = [...ids[nestedKey], ...nestedIds[nestedKey]]; }
//                         else { ids[nestedKey] = nestedIds[nestedKey]; }
//                     });
//                 });
//             }
//             // If relationship object is a single object
//             else if (isRelationshipObject(object[key])) {
//                 // Add id of relationship to return object
//                 if (object[key].id) { ids[relMap[key]] = [object[key].id]; }
//                 // Add nested ids to return object
//                 const nestedIds = objectToIds(childRelMap, object[key]);
//                 Object.keys(nestedIds).forEach(nestedKey => {
//                     if (ids[nestedKey]) { ids[nestedKey] = [...ids[nestedKey], ...nestedIds[nestedKey]]; }
//                     else { ids[nestedKey] = nestedIds[nestedKey]; }
//                 });
//             }
//         }
//     });
//     return ids;
// };

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
const objectToIds = <GQLCreate extends { [x: string]: any }, GQLUpdate extends { id?: string }>(
    actionType: 'Connect' | 'Create' | 'Disconnect' | 'Read' | 'Update', // Don't pass Delete
    relMap: { [x: string]: GraphQLModelType } & { __typename: GraphQLModelType },
    object: GQLCreate | GQLUpdate,
): { [key in GraphQLModelType]?: string[] } => {
    // Initialize return object
    const ids: { [key in GraphQLModelType]?: string[] } = {};
    // If not a 'Create', add id of this object to return object
    if (actionType !== 'Create' && object.id) { ids[relMap.__typename] = [`${actionType}-${object.id}`]; }
    // Loop through all keys in relationship map
    Object.keys(relMap).forEach(key => {
        // Skip __typename
        if (key === '__typename') return;
        // Loop through all key variations. Multiple variations can be set at once (e.g. resourcesCreate, resourcesUpdate)
        const variations = ['', 'Id', 'Ids', 'Connect', 'Create', 'Delete', 'Disconnect', 'Update'];
        variations.forEach(variation => {
            // If key is in object
            if (`${key}${variation}` in object) {
                // Get child relationship map
                const childRelMap = getValidator(relMap[key], 'objectToIds').validatedRelationshipMap;
                // Get child action type
                let childActionType: 'Connect' | 'Create' | 'Delete' | 'Disconnect' | 'Read' | 'Update' = 'Read';
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
                        throw new CustomError(CODE.InvalidArgs, 'TODO');
                    }
                    // Recursively call objectToIds on each object in array
                    value.forEach((item: any) => {
                        const nestedIds = objectToIds(childActionType as 'Create' | 'Read' | 'Update', childRelMap, item);
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
                        throw new CustomError(CODE.InvalidArgs, 'TODO');
                    }
                    // Add ids to return object
                    if (ids[relMap[key]]) { ids[relMap[key]] = [...(ids[relMap[key]] ?? []), ...value.map(id => `${childActionType}-${id}`)]; }
                    // If child action is a 'Connect' and parent action is 'Update', add 'Disconnect' placeholders. 
                    // These ids will have to be queried for later
                    if (childActionType === 'Connect' && actionType === 'Update') {
                        ids[relMap[key]] = [...(ids[relMap[key]] ?? []), ...value.map(id => `${childActionType}-${id}.${key}${variation}`)];
                    }
                }
                // If value is a single object
                else if (isRelationshipObject(value)) {
                    // 'Connect', 'Delete', and 'Disconnect' are invalid for objects
                    if (['Connect', 'Delete', 'Disconnect'].includes(childActionType)) {
                        throw new CustomError(CODE.InvalidArgs, 'TODO');
                    }
                    // Recursively call objectToIds on object
                    const nestedIds = objectToIds(childActionType as 'Create' | 'Read' | 'Update', childRelMap, value);
                    Object.keys(nestedIds).forEach(nestedKey => {
                        if (ids[nestedKey]) { ids[nestedKey] = [...ids[nestedKey], ...nestedIds[nestedKey]]; }
                        else { ids[nestedKey] = nestedIds[nestedKey]; }
                    });
                }
                // If value is a single id
                else if (typeof value === 'string') {
                    // Only 'Connect', 'Delete', and 'Disconnect' are valid for ids
                    if (!['Connect', 'Delete', 'Disconnect'].includes(childActionType)) {
                        throw new CustomError(CODE.InvalidArgs, 'TODO');
                    }
                    // Add id to return object
                    if (ids[relMap[key]]) { ids[relMap[key]] = [...(ids[relMap[key]] ?? []), `${childActionType}-${value}`]; }
                    // If child action is a 'Connect' and parent action is 'Update', add 'Disconnect' placeholder. 
                    // This id will have to be queried for later
                    else if (childActionType === 'Connect' && actionType === 'Update') {
                        ids[relMap[key]] = [`${childActionType}-${value}.${key}${variation}`];
                    }
                }
            }
        });
    });
    return ids;
};

/**
 * Helper function to convert disconnect placeholders to disconnect ids
 */
const disconnectPlaceholdersToIds = (...params: any) => {} //TODO this itself is a placeholder


/**
 * Finds all ids of objects in a crud request that need to be checked for permissions
 */
export const getAuthenticatedIds = <GQLCreate extends { [x: string]: any }, GQLUpdate extends { id?: string }>(
    actionType: 'Connect' | 'Create' | 'Delete' | 'Disconnect' | 'Read' | 'Update',
    objects: (GQLCreate | GQLUpdate | string | PartialPrismaSelect)[],
    objectType: GraphQLModelType,
    prisma: PrismaType,
    userId: string | null,
): { [key in GraphQLModelType]?: string[] } => {
    // Initialize return object
    const result: { [key in GraphQLModelType]?: string[] } = {};
    // Find validator and prisma delegate for this object type
    const validator = getValidator(objectType, 'getAuthenticatedIds');
    // Filter out objects that are strings
    const filteredObjects = objects.filter(object => {
        const isString = typeof object === 'string';
        if (isString) {
            result[objectType] = [...(result[objectType] ?? []), `${actionType}-${object}`];
        }
        return !isString;
    });
    // For each object
    filteredObjects.forEach(object => {
        // Call objectToIds to get ids of all objects requiring authentication
        const ids = objectToIds('Read', validator.validatedRelationshipMap, object);
        // Add ids to return object
        Object.keys(ids).forEach(key => {
            if (result[key]) { result[key] = [...result[key], ...ids[key]]; }
            else { result[key] = ids[key]; }
        });
    }
    // Now we should have ALMOST all ids of objects requiring authentication. What's missing are the ids of 
    // objects that are being implicitly disconnected (i.e. being replaced by a new connect). We need to find
    // these ids by querying for them
    fdsafdsafdsafd
    // Replace placeholders in result with actual ids we just found
    fdsafdsafds
    // Return result
    return result;
    // // Collect all top-level ids from the request. These always have to be checked for permissions. 
    // // NOTE: objects can include create inputs, which don't have an id. We still authenticate these because they 
    // // may have an authenticated relationship (e.g. parent, owner, etc.)
    // result[objectType] = objects.map(x => {
    //     if (typeof x === 'string') return x;
    //     return x.id;
    // }).filter(Boolean) as string[];
    // // Now the tricky part: if the request contains a nested object, we need to check if the user has permission to create/update that object.
    // // Nested objects are grouped by type, to reduce the number of database queries
    // const validatedRelationshipMap = validator.validatedRelationshipMap;
    // // Check all objects for validated relationship fields. 
    // for (const object of objects.filter(x => typeof x !== 'string') as (GQLCreate | GQLUpdate | PartialPrismaSelect)[]) {
    //     for (const [relField, relType] of Object.entries(validatedRelationshipMap)) {
    //         // Use helper function to get all ids of objects that need to be authenticated
    //         const authMap = getAuthenticatedIdsHelper(object, objectType, relField, relType);
    //         // Add ids to result
    //         for (const [type, ids] of Object.entries(authMap)) {
    //             result[type] = [...(result[type] || []), ...ids];
    //         }
    //     }
    // }
    // // Remove duplicates
    // for (const [type, ids] of Object.entries(result)) {
    //     result[type] = [...new Set(ids)];
    // }
    // return result;
}