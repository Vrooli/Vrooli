import { JoinMap } from "../models/types";
import { isRelationshipObject } from "./isOfType";

/**
 * Idempotent helper function for removing join tables between
 * many-to-many relationship parents and children
 * @param obj - DB-shaped object
 * @param map - Mapping of many-to-many relationship names to join table names
 */
export const removeJoinTables = (obj: any, map: JoinMap | undefined): any => {
    if (!obj || !map) return obj;
    // Create result object
    const result: any = {};
    // Iterate over join map
    for (const [key, value] of Object.entries(map)) {
        // If the key is in the object
        if (obj[key]) {
            // If the value is an array
            if (Array.isArray(obj[key])) {
                // Check if the join should be applied (i.e. elements are objects with one non-ID key)
                if (obj[key].every((o: any) => isRelationshipObject(o) && Object.keys(o).length === 1 && Object.keys(o)[0] !== "id")) {
                    // Remove the join table from each item in the array
                    result[key] = obj[key].map((item: any) => item[value]);
                }
            } else {
                // Check if the join should be applied (i.e. element is an object with one non-ID key)
                if (isRelationshipObject(obj[key]) && Object.keys(obj[key]).length === 1 && Object.keys(obj[key])[0] !== "id") {
                    // Otherwise, remove the join table from the object
                    result[key] = obj[key][value];
                }
            }
        }
    }
    return {
        ...obj,
        ...result,
    };
};
