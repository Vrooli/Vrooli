/**
 * Determines if an object is a relationship object, and not an array of relationship objects.
 * @param obj - object to check
 * @returns True if obj is a relationship object, false otherwise
 */
export function isRelationshipObject(obj: unknown): obj is object {
    return obj !== null && obj !== undefined && // Not null or undefined
        (typeof obj === "object" || typeof obj === "function") && // Is an object or function
        !Array.isArray(obj) && // Not an array
        Object.prototype.toString.call(obj) !== "[object Date]"; // Not a date
}

/**
* Determines if an object is an array of relationship objects, and not a relationship object.
* @param obj - object to check
* @returns True if obj is an array of relationship objects, false otherwise
*/
export function isRelationshipArray(obj: unknown): obj is object[] {
    return Array.isArray(obj) && obj.every(isRelationshipObject);
}
