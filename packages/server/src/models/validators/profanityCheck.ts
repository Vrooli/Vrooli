import { CustomError } from "../../events";
import { hasProfanity } from "../../utils/censor";
import { isRelationshipArray, isRelationshipObject, ObjectMap } from "../builder";
import { GraphQLModelType, PartialGraphQLInfo, Validator } from "../types";

/**
 * Helper function for profanity check. Recursively finds every field which must be checked for profanity, by: 
 * - Grabbing every field in a "translationsCreate" or "translationsUpdate" object
 * - Grabbing every field specified by the current object's validator's profanityFields array
 * @param input The input object to collect translations from
 * @param objectType The type of the input object
 * @returns An object with every field that must be checked for profanity
 */
const collectProfanities = (input: { [x: string]: any }, objectType?: GraphQLModelType): { [x: string]: string[] } => {
    // Initialize result
    const result: { [x: string]: string[] } = {};
    // Handle base case
    // Get current object's validator
    const validator: Validator<any, any, any, any, any, any, any> | undefined = objectType ? ObjectMap[objectType]?.validate : undefined;
    // If validator specifies profanityFields, add them to the result
    if (validator?.profanityFields) {
        for (const field of validator.profanityFields) {
            if (input[field]) result[field] = result[field] ? [...result[field], input[field]] : [input[field]];
            // Also check "root" object, so we can support versioned objects without implementing dot notation
            else if (input.root?.[field]) result[field] = result[field] ? [...result[field], input.root[field]] : [input.root[field]];
        }
    }
    // Add translationsCreate and translationsUpdate to the result
    if (isRelationshipArray(input.translationsCreate) && input.translationsCreate.length > 0) {
        // Only add non-ID fields
        for (const field of Object.keys(input.translationsCreate[0])) {
            if (field !== 'id' && !field.endsWith('Id')) {
                const values = input.translationsCreate.map((x: any) => x[field]);
                result[field] = result[field] ? [...result[field], ...values] : values;
            }
        }
    }
    if (input.translationsUpdate) {
        // Only add non-ID fields
        for (const field of Object.keys(input.translationsUpdate)) {
            if (field !== 'id' && !field.endsWith('Id')) {
                const values = input.translationsUpdate.map((x: any) => x[field]);
                result[field] = result[field] ? [...result[field], ...values] : values;
            }
        }
    }
    // Handle recursive case
    for (const key in input) {
        // Find next objectType, if any
        let nextObjectType: GraphQLModelType | undefined;
        // Strip "Create" and "Update" from the end of the key
        const strippedKey = key.endsWith('Create') || key.endsWith('Update') ? key.slice(0, -6) : key;
        // Check if stripped key is in validator's validateMap
        if (typeof validator?.validateMap?.[strippedKey] === 'string')
            nextObjectType = validator?.validateMap?.[strippedKey] as GraphQLModelType;
        // Now we can validate translations objects
        // Check for array
        if (isRelationshipArray(input[key])) {
            for (const item of input[key]) {
                const newFields = collectProfanities(item, nextObjectType);
                for (const field in newFields) {
                    result[field] = result[field] ? [...result[field], ...newFields[field]] : newFields[field];
                }
            }
        }
        // Check for object
        else if (isRelationshipObject(input[key])) {
            const newFields = collectProfanities(input[key], nextObjectType);
            for (const field in newFields) {
                result[field] = result[field] ? [...result[field], ...newFields[field]] : newFields[field];
            }
        }
    }
    return result;
}

/**
 * Throws an error if a object's translations contain any banned words.
 * Recursively checks for censored words on any field that:
 * 1. Is not "id"
 * 2. Does not end with "Id"
 * 3. Has a string value
 * 
 * Additionally, finds the validator for the object's type and checks if any additional fields - besides 
 * those found in translation objects - should be checked for censored words (e.g. username, email).
 * @params input An array of objects with translations
 * @params objectType The type of the object
 * @params languages The languages to use for error messages
 */
export const profanityCheck = (input: { [x: string]: any }[], objectType: GraphQLModelType | undefined, languages: string[]): void => {
    // Find all fields which must be checked for profanity
    const fieldsToCheck: { [x: string]: string[] } = {};
    for (const item of input) {
        const newFields = collectProfanities(item, objectType);
        for (const field in newFields) {
            fieldsToCheck[field] = fieldsToCheck[field] ? [...fieldsToCheck[field], ...newFields[field]] : newFields[field];
        }
    }
    // Check each field for profanity
    for (const field in fieldsToCheck) {
        if (hasProfanity(...fieldsToCheck[field])) {
            throw new CustomError('0115', 'BannedWord', languages);
        }
    }
}