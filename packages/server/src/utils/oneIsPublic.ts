import { getValidator } from "../getters";
import { GraphQLModelType } from "../models/types";

/**
 * Given permissions data and a list of fields and GraphQLModels which have validators, determines if one of the fields is public. 
 * This typically means that one of the fields has a non-null value with "isPrivate" and "isDeleted" set to false.
 */
export const oneIsPublic = <PrismaSelect extends { [x: string]: any }>(
    permissionsData: { [key in keyof PrismaSelect]: any },
    list: [keyof PrismaSelect, GraphQLModelType][],
    languages: string[],
): boolean => {
    // Loop through each field in the list
    for (let i = 0; i < list.length; i++) {
        const [field, type] = list[i];
        // Get the validator for this type
        const validator = getValidator(type, languages, 'oneIsPublic');
        // Use validator to determine if this field is public
        if (validator.isPublic(permissionsData[field], languages)) {
            return true;
        }
    }
    return false;
}