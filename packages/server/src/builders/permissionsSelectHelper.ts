import { getLogic } from "../getters";
import { GraphQLModelType, PermissionsMap } from "../models/types";
import { isRelationshipObject } from "./isRelationshipObject";

/**
 * Recursively converts a PermissionsMap object into a real Prisma select query
 * @param mapResolver The PermissionsMap object resolver, or a recursed value
 * @param userID Current user ID
 * @param languages Preferred languages for error messages
 * @returns A Prisma select query
 */
export const permissionsSelectHelper = <Select extends { [x: string]: any }>(
    mapResolver: PermissionsMap<Select> | ((userId: string | null, languages: string[]) => PermissionsMap<Select>),
    userId: string | null,
    languages: string[]
): Select => {
    const map = typeof mapResolver === 'function' ? mapResolver(userId, languages) : mapResolver;
    // Initialize result
    const result: { [x: string]: any } = {};
    // For every key in the PermissionsMap object
    for (const key of Object.keys(map)) {
        // Get the value of the key
        const value = map[key];
        // If the value is an array, recurse
        if (Array.isArray(value)) {
            result[key] = value.map((x) => permissionsSelectHelper(x, userId, languages));
        }
        // If the value is an object, recurse
        else if (isRelationshipObject(value)) {
            result[key] = permissionsSelectHelper(value, userId, languages);
        }
        // If the value is a GraphQLModelType, attempt to recurse using the validator for that type
        else if (typeof value === 'string') {
            // Check if the validator exists. If not, assume this is some other string and add it to the result
            const { validate } = getLogic([], value as GraphQLModelType, languages, 'permissionsSelectHelper');
            if (!validate) {
                result[key] = true;
            }
            // If the validator exists, recurse using the validator's permissionsSelect function
            else {
                const childMap = validate.permissionsSelect(userId, languages);
                if (childMap) {
                    result[key] = { select: permissionsSelectHelper(childMap, userId, languages) };
                }
            }
        }
        // Else, add the key to the result
        else {
            result[key] = true;
        }
    }
    // Return the result
    return result as Select;
}