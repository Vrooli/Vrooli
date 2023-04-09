import { getLogic } from "../getters";
import { PrismaRelMap } from "../models/types";
import { IdsByAction, IdsByType, InputsByType, QueryAction } from "./types";

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
 *     idsByAction: { 'Connect': ['ghi789'], 'Disconnect': ['Routine|jkl012.grandChild.org'], 'Update': ['abc123', 'jkl012'] },
 *     idsByType: { 'Routine': ['abc123', 'jkl012'], 'Organization': ['Routine|jkl012.grandChild.org'] }
 * }
 */
export const inputToMapWithPartials = <T extends Record<string, any>>(
    actionType: QueryAction,
    relMap: PrismaRelMap<T>,
    object: any,//PrismaUpdate | { where: { [key: string]: any }, data: PrismaUpdate } | string,
    languages: string[],
): {
    idsByAction: IdsByAction,
    idsByType: IdsByType,
    inputsByType: InputsByType,
} => {
    // Initialize return objects
    const idsByAction: IdsByAction = {};
    const idsByType: IdsByType = {};
    const inputsByType: InputsByType = {};
    // If action is Create, Update, or Delete, add input to inputsByType
    if (['Create', 'Update', 'Delete'].includes(actionType)) {
        inputsByType[relMap.__typename] = { Create: [], Update: [], Delete: [], [actionType]: [object as any] };
    }
    // If action is Connect, Disconnect, or Delete, add to maps and return. 
    // The rest of the function is for parsing objects
    if (['Connect', 'Disconnect', 'Delete'].includes(actionType)) {
        idsByAction[actionType] = [object as string];
        idsByType[relMap.__typename] = [object as string];
        return { idsByAction, idsByType, inputsByType };
    }
    // Can only be Create, Update, or Read past this point
    // If not a 'Create' (i.e. already exists in database), add id of this object to return object
    if (actionType !== 'Create') {
        const objectId = actionType === 'Update' ? object.where.id : object.id;
        idsByAction[actionType] = [objectId];
        idsByType[relMap.__typename] = [objectId];
    }
    // Loop through all keys in relationship map
    Object.keys(relMap).forEach((key) => {
        // Ignore __typename
        if (key === '__typename') return;

        // Get relationship type
        const relType = relMap[key]!;

        // Helper function to process relationship action
        const processRelationshipAction = (action, relationshipAction, relationshipType) => {
            // Get nested relMap data
            const { format } = getLogic(['format'], relationshipType, languages, 'inputToMapWithPartials loop');
            const childRelMap = format.gqlRelMap;

            if (object.hasOwnProperty(relationshipAction)) {
                // Get the nested object(s) corresponding to the relationship action
                const nestedObjects = Array.isArray(object[relationshipAction])
                    ? object[relationshipAction]
                    : [object[relationshipAction]];

                // Process each nested object
                nestedObjects.forEach((nestedObject) => {
                    // Handle Disconnect case (boolean or array of strings)
                    if (action === 'Disconnect' && typeof nestedObject === 'boolean') {
                        nestedObject = `${relationshipType}|${key}`;
                    }

                    // Recursive call for nested objects
                    const { idsByAction: nestedIdsByAction, idsByType: nestedIdsByType, inputsByType: nestedInputsByType } = inputToMapWithPartials(
                        action,
                        childRelMap as any,
                        nestedObject,
                        languages
                    );

                    // Merge results
                    Object.assign(idsByAction, nestedIdsByAction);
                    Object.assign(idsByType, nestedIdsByType);
                    Object.assign(inputsByType, nestedInputsByType);
                });
            }
        };

        // Check for each action type in the object
        (['Connect', 'Disconnect', 'Create', 'Update', 'Delete'] as const).forEach((action) => {
            if (typeof relType === 'object') {
                // Process union relationship actions
                Object.keys(relType).forEach((unionKey) => {
                    const unionRelType = relType[unionKey];
                    const relationshipAction = `${unionKey}${action}`;
                    processRelationshipAction(action, relationshipAction, unionRelType);
                });
            } else {
                // Process regular relationship actions
                const relationshipAction = `${key}${action}`;
                processRelationshipAction(action, relationshipAction, relType);
            }
        });
    });
    return { idsByAction, idsByType, inputsByType };
};