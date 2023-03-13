import { PrismaUpdate } from "../builders/types";
import { PrismaRelMap } from "../models/types";
import { IdsByAction, IdsByType, QueryAction } from "./types";

/**
 * Helper function to grab ids from an object, and map them to their action and object type. For implicit ids (i.e. they 
 * don't appear in the mutation. This happens for some one-to-one and many-to-one relations) to be authenticated, a placeholder is added.
 * 
 * NOTE: Due to the way GraphQL schema is defined, updates always specify their ID (even relationships). You would 
 * think that we could use this to simplify the logic of this function, but doing so would open up a security hole. 
 * It would allow users to pass in incorrect IDs of objects that have full access to, to modify objects that they
 * don't have access to.
 * 
 * Simple example (no placeholders):
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
 * Complex example (with a nested placeholder):
 * (ActionType.Update, relMap, {
 *     ...
 *     type: 'Routine',
 *     id: 'abc123',
 *     parentCreate: {
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
 *              grandChildUpdate: {
 *                  ... 
 *                  type: 'Routine',
 *                  org: { disconnect: true },
 *              }
 *          }
 *    ]
 * }) => { 
 *     idsByAction: { 'Connect': ['ghi789'], 'Disconnect': ['Routine-jkl012.grandChild.org'], 'Update': ['abc123', 'jkl012'] },
 *     idsByType: { 'Routine': ['abc123', 'jkl012'], 'Organization': ['Routine-jkl012.grandChild.org'] }
 * }
 */
export const inputToMapWithPartials = <T extends Record<string, any>>(
    actionType: QueryAction,
    relMap: PrismaRelMap<T>,
    object: PrismaUpdate | string,
    languages: string[],
): {
    idsByAction: IdsByAction,
    idsByType: IdsByType,
} => {
    // Initialize return objects
    const idsByAction: IdsByAction = {};
    const idsByType: IdsByType = {};
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