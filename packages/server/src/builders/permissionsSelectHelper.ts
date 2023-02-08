import { GqlModelType } from "@shared/consts";
import { CustomError, logger } from "../events";
import { getLogic } from "../getters";
import { PermissionsMap } from "../models/types";
import { isRelationshipObject } from "./isRelationshipObject";

/**
 * Recursively converts a PermissionsMap object into a real Prisma select query
 * @param mapResolver The PermissionsMap object resolver, or a recursed value
 * @param userID Current user ID
 * @param languages Preferred languages for error messages
 * @param recursionDepth The current recursion depth. Used to detect infinite recursion
 * @returns A Prisma select query
 */
export const permissionsSelectHelper = <Select extends { [x: string]: any }>(
    mapResolver: PermissionsMap<Select> | ((userId: string | null, languages: string[]) => PermissionsMap<Select>),
    userId: string | null,
    languages: string[],
    recursionDepth = 0,
): Select => {
    console.log('permissionsselecthelper 1permissionsselecthelper 1', userId);
    // If recursion depth is too high, throw an error
    if (recursionDepth > 100) {
        throw new CustomError('0386', 'InternalError', languages ?? ['en'], { userId, recursionDepth });
    }
    const map = typeof mapResolver === 'function' ? mapResolver(userId, languages) : mapResolver;
    console.log('permissionsselecthelper 2', recursionDepth, JSON.stringify(map), '\n')
    // Initialize result
    const result: { [x: string]: any } = {};
    // For every key in the PermissionsMap object
    for (const key of Object.keys(map)) {
        // Get the value of the key
        const value = map[key];
        console.log('permissionsselecthelper 3', key, JSON.stringify(value), '\n')
        // If the value is an array, recurse
        if (Array.isArray(value)) {
            result[key] = value.map((x) => permissionsSelectHelper(x, userId, languages, recursionDepth + 1));
        }
        // If the value is an object, recurse
        else if (isRelationshipObject(value)) {
            result[key] = permissionsSelectHelper(value, userId, languages, recursionDepth + 1);
        }
        // If the value is a GqlModelType, attempt to recurse using the validator for that type
        else if (typeof value === 'string' && value in GqlModelType) {
            console.log('permissionsselecthelper string 1', key, value);
            // Check if the validator exists. If not, assume this is some other string and add it to the result
            const { validate } = getLogic(['validate'], value as GqlModelType, languages, 'permissionsSelectHelper');
            console.log('permissionsselecthelper string 2', key, validate)
            if (!validate) {
                result[key] = true;
            }
            // If the validator exists, recurse using the validator's permissionsSelect function
            else {
                console.log('permissionsselecthelper string 3', key, userId)
                const childMap = validate.permissionsSelect(userId, languages);
                console.log('permissionsselecthelper string 4', key, childMap)
                if (childMap) {
                    result[key] = { select: permissionsSelectHelper(childMap, userId, languages, recursionDepth + 1) };
                }
                console.log('permissionsselecthelper string 5', key)
            }
        }
        // Else, add the key to the result
        else {
            result[key] = value;
        }
        console.log('permissionsselecthelper 4', key)
    }
    console.log('permissionsselecthelper 5')
    // Return the result
    return result as Select;
}