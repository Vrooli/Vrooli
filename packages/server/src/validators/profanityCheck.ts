import { GqlModelType } from "@local/shared";
import { isRelationshipArray, isRelationshipObject } from "../builders";
import { CustomError } from "../events";
import { ObjectMap } from "../models/base";
import { hasProfanity } from "../utils/censor";

/**
 * Helper function for profanity check. Recursively finds every field which must be checked for profanity, by: 
 * - Grabbing every field in a "translationsCreate" or "translationsUpdate" object
 * - Grabbing every field specified by the current object's validator's profanityFields array
 * @param input The input object to collect translations from
 * @param objectType The type of the input object
 * @returns An object with every field that must be checked for profanity
 */
const collectProfanities = (input: { [x: string]: any }, objectType?: `${GqlModelType}`): { [x: string]: string[] } => {
    // Initialize result
    const result: { [x: string]: string[] } = {};
    // Handle base case
    // Get current object's formatter and validator
    const format = objectType ? ObjectMap[objectType]?.format : undefined;
    const validate = objectType ? ObjectMap[objectType]?.validate : undefined;
    // If validator specifies profanityFields, add them to the result
    if (validate?.profanityFields) {
        for (const field of validate.profanityFields) {
            if (input[field]) result[field] = result[field] ? [...result[field], input[field]] : [input[field]];
        }
    }
    // Helper function to handle translations
    const handleTranslationsArray = (translationsArray: any[], result: { [x: string]: string[] }) => {
        for (const translation of translationsArray) {
            for (const field in translation) {
                if (field !== "id" && !field.endsWith("Id")) {
                    result[field] = result[field] ? [...result[field], translation[field]] : [translation[field]];
                }
            }
        }
    };
    // Add translationsCreate and translationsUpdate to the result
    if (isRelationshipArray(input.translationsCreate) && input.translationsCreate.length > 0) {
        handleTranslationsArray(input.translationsCreate, result);
    }
    if (isRelationshipArray(input.translationsUpdate) && input.translationsUpdate.length > 0) {
        handleTranslationsArray(input.translationsUpdate, result);
    }
    // Check for tags, which always appear as a list of strings being connected
    if (Array.isArray(input.tagsConnect) && input.tagsConnect.length > 0) {
        result.tagsConnect = input.tagsConnect as string[];
    }
    // Handle recursive case
    const processNestedFields = (nestedInput: any, nestedObjectType?: `${GqlModelType}`) => {
        const newFields = collectProfanities(nestedInput, nestedObjectType);
        for (const field in newFields) {
            result[field] = result[field] ? [...result[field], ...newFields[field]] : newFields[field];
        }
    };
    for (const key in input) {
        // Find next objectType, if any
        let nextObjectType: `${GqlModelType}` | undefined;
        // Strip "Create" and "Update" from the end of the key
        const strippedKey = key.endsWith("Create") || key.endsWith("Update") ? key.slice(0, -6) : key;
        // Check if stripped key is in validator's validateMap
        if (typeof format?.gqlRelMap?.[strippedKey] === "string") {
            nextObjectType = format?.gqlRelMap?.[strippedKey] as GqlModelType;
        }
        if (isRelationshipArray(input[key])) {
            for (const item of input[key]) {
                processNestedFields(item, nextObjectType);
            }
        } else if (isRelationshipObject(input[key])) {
            processNestedFields(input[key], nextObjectType);
        }
    }
    return result;
};

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
export const profanityCheck = (input: { [x: string]: any }[], objectType: `${GqlModelType}` | undefined, languages: string[]): void => {
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
            throw new CustomError("0115", "BannedWord", languages);
        }
    }
};
