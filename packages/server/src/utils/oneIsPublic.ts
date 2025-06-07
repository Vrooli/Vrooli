import { type ModelType } from "@vrooli/shared";
import { ModelMap } from "../models/base/index.js";

/**
 * Given permissions data and a list of fields and GraphQLModels which have validators, determines if one of the fields is public. 
 * This typically means that one of the fields has a non-null value with "isPrivate" and "isDeleted" set to false.
 */
export function oneIsPublic<PrismaSelect extends { [x: string]: any }>(
    list: [keyof PrismaSelect, `${ModelType}`][],
    permissionsData: { [key in keyof PrismaSelect]: any },
    getParentInfo: ((id: string, typename: `${ModelType}`) => any | undefined),
): boolean {
    // Loop through each field in the list
    for (let i = 0; i < list.length; i++) {
        const [field, type] = list[i];
        // Get the validator for this type
        const { idField, validate } = ModelMap.getLogic(["idField", "validate"], type);
        // Use validator to determine if this field is public
        if (permissionsData[field] && validate().isPublic(permissionsData[field] ?? getParentInfo(permissionsData.id ?? permissionsData[idField], type), getParentInfo)) {
            return true;
        }
    }
    return false;
}
