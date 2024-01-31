import pkg from "lodash";
import { isRelationshipObject } from "./isOfType";

const { merge } = pkg;

/**
 * Recombines objects returned from the supplementalFields function into a shape that matches the requested info
 * @param data Original, unsupplemented data, where each object has an ID
 * @param objectsById Dictionary of objects with supplemental fields added, where each value contains at least an ID
 * @returns data with supplemental fields added
 */
export const combineSupplements = (data: { [x: string]: any }, objectsById: { [x: string]: any }) => {
    const result: { [x: string]: any } = {};
    // Loop through each key/value pair in data
    for (const [key, value] of Object.entries(data)) {
        // If value is an array, add each element to the correct key in objectDict
        if (Array.isArray(value)) {
            // Pass each element through combineSupplements
            result[key] = data[key].map((v: any) => combineSupplements(v, objectsById));
        }
        // If value is an object (and not a date), add it to the correct key in objectDict
        else if (isRelationshipObject(value)) {
            // Pass value through combineSupplements
            result[key] = combineSupplements(value, objectsById);
        }
    }
    // Handle base case (combining non-array and non-object values)
    return merge(result, objectsById[data.id]);
};
