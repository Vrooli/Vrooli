import { GqlModelType } from "@shared/consts";
import { isRelationshipObject } from "./isRelationshipObject";
import { removeSupplementalFields } from "./removeSupplementalFields";

/**
 * Determines if a queried object matches the shape of a GraphQL request object
 * @param obj - queried object
 * @param query - GraphQL request object
 * @returns True if obj matches query
 */
export const subsetsMatch = (obj: any, query: any): boolean => {
    // Check that both params are valid objects
    if (obj === null || typeof obj !== 'object' || query === null || typeof query !== 'object') return false;
    // Check if query type is in FormatterMap. 
    // This should hopefully always be the case for the main subsetsMatch call, 
    // but not necessarily for the recursive calls.
    let formattedQuery = query;
    if (query?.type === 'string') {
        // Remove calculated fields from query, since these will not be in obj
        formattedQuery = removeSupplementalFields(query.type as GqlModelType, query);
    }
    // First, check if obj is a join table. If this is the case, what we want to check 
    // is actually one layer down
    let formattedObj = obj;
    if (Object.keys(obj).length === 1 && isRelationshipObject(obj[Object.keys(obj)[0]])) {
        formattedObj = obj[Object.keys(obj)[0]];
    }
    // If query contains any fields which are not in obj, return false
    for (const key of Object.keys(formattedQuery)) {
        // Ignore type
        if (key === 'type') continue;
        // If union, check if any of the union types match formattedObj
        if (key[0] === key[0].toUpperCase()) {
            const unionTypes = Object.keys(formattedQuery);
            const unionSubsetsMatch = unionTypes.some(unionType => subsetsMatch(formattedObj, formattedQuery[unionType]));
            if (!unionSubsetsMatch) return false;
        }
        // If key is not in object, return false
        else if (!formattedObj.hasOwnProperty(key)) {
            return false;
        }
        // If formattedObj[key] is array, compare to first element of query[key]
        else if (Array.isArray(formattedObj[key])) {
            // Can't check if array is empty
            if (formattedObj[key].length === 0) continue;
            const firstElem = formattedObj[key][0];
            const matches = subsetsMatch(firstElem, formattedQuery[key]);
            if (!matches) return false;
        }
        // If formattedObj[key] is formattedObject, recurse
        else if (isRelationshipObject((formattedObj)[key])) {
            const matches = subsetsMatch(formattedObj[key], formattedQuery[key]);
            if (!matches) return false;
        }
    }
    return true;
}