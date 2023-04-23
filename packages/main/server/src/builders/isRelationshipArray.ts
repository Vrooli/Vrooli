import { isRelationshipObject } from ".";

/**
 * Determines if an object is an array of relationship objects, and not a relationship object.
 * @param obj - object to check
 * @returns True if obj is an array of relationship objects, false otherwise
 */
export const isRelationshipArray = (obj: any): obj is Object[] => Array.isArray(obj) && obj.every(isRelationshipObject);
