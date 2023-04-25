import { isObject } from "@local/shared";

/**
 * Determines if an object is a relationship object, and not an array of relationship objects.
 * @param obj - object to check
 * @returns True if obj is a relationship object, false otherwise
 */
export const isRelationshipObject = (obj: any): obj is Object => isObject(obj) && Object.prototype.toString.call(obj) !== "[object Date]";
