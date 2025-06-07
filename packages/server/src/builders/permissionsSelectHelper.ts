import { ModelType } from "@vrooli/shared";
import { CustomError } from "../events/error.js";
import { ModelMap } from "../models/base/index.js";
import { type PermissionsMap } from "../models/types.js";
import { isRelationshipObject } from "./isOfType.js";

const MAX_RECURSION_DEPTH = 50;

/**
 * Helper function to remove the first layer of dot values form an array of
 * dot values. For example, if the array is ['a', 'b.c', 'd.e.f'], the result
 * will be ['c', 'e.f']
 */
function removeFirstDotLayer(arr: string[]): string[] {
    return arr.map(x => x.split(".").slice(1).join(".")).filter(x => x !== "");
}

/**
 * Recursively converts a PermissionsMap object into a real Prisma select query
 * @param mapResolver The PermissionsMap object resolver, or a recursed value
 * @param userID Current user ID
 * @param recursionDepth The current recursion depth. Used to detect infinite recursion
 * @param omitFields Fields to omit from the selection. Supports dot notation.
 * @returns A Prisma select query
 */
export function permissionsSelectHelper<Select extends { [x: string]: any }>(
    mapResolver: PermissionsMap<Select> | ((userId: string | null) => PermissionsMap<Select>),
    userId: string | null,
    recursionDepth = 0,
    omitFields: string[] = [],
): Select {
    // If recursion depth is too high, throw an error
    if (recursionDepth > MAX_RECURSION_DEPTH) {
        throw new CustomError("0386", "InternalError", { userId, recursionDepth });
    }
    const map = typeof mapResolver === "function" ? mapResolver(userId) : mapResolver;
    // Initialize result
    const result: { [x: string]: any } = {};
    // For every key in the PermissionsMap object
    for (const key of Object.keys(map)) {
        // If the key is in the omitFields array, skip it
        if (omitFields.includes(key)) {
            continue;
        }
        // Get the value of the key
        const value = map[key];
        // If the value is an array
        if (Array.isArray(value)) {
            // If array is of length 2, where the first element is a ModelType and the second is a string array, 
            // attempt to recurse using substitution
            if (value.length === 2 && typeof value[0] === "string" && value[0] in ModelType && Array.isArray(value[1])) {
                // Check if the validator exists. If not, assume this is not a substitution and add it to the result
                const validate = ModelMap.get(value[0] as ModelType, false)?.validate;
                if (!validate) {
                    result[key] = value;
                }
                // If the validator exists, recurse using the validator's permissionsSelect function
                else {
                    // Child omit is curr omit with first dot level removed, combined with value[1]
                    const childOmitFields = removeFirstDotLayer(omitFields).concat(value[1]);
                    // Child map is the validator's permissionsSelect function
                    const childMap = validate().permissionsSelect(userId);
                    if (childMap) {
                        result[key] = { select: permissionsSelectHelper(childMap, userId, recursionDepth + 1, childOmitFields) };
                    }
                }
            }
            // Otherwise, recurse normally
            else {
                // Child omit is curr omit with first dot level removed
                const childOmitFields = removeFirstDotLayer(omitFields);
                result[key] = value.map((x: any) => permissionsSelectHelper(x, userId, recursionDepth + 1), childOmitFields);
            }
        }
        // If the value is an object, recurse
        else if (isRelationshipObject(value)) {
            // Child omit is curr omit with first dot level removed
            const childOmitFields = removeFirstDotLayer(omitFields);
            result[key] = permissionsSelectHelper(value, userId, recursionDepth + 1, childOmitFields);
        }
        // If the value is a ModelType, attempt to recurse using substitution
        else if (typeof value === "string" && value in ModelType) {
            // Check if the validator exists. If not, assume this is some other string and add it to the result
            const validate = ModelMap.get(value as ModelType, false)?.validate;
            if (!validate) {
                result[key] = value;
            }
            // If the validator exists, recurse using the validator's permissionsSelect function
            else {
                // Child omit is curr omit with first dot level removed
                const childOmitFields = removeFirstDotLayer(omitFields);
                // Child map is the validator's permissionsSelect function
                const childMap = validate().permissionsSelect(userId);
                if (childMap) {
                    result[key] = { select: permissionsSelectHelper(childMap, userId, recursionDepth + 1, childOmitFields) };
                }
            }
        }
        // Else, add the key to the result
        else {
            result[key] = value;
        }
    }
    // Return the result
    return result as Select;
}
