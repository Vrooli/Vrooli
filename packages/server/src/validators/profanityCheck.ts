import { GqlModelType } from "@local/shared";
import { isRelationshipArray } from "../builders/isRelationshipArray";
import { isRelationshipObject } from "../builders/isRelationshipObject";
import { CustomError } from "../events/error";
import { getLogic } from "../getters/getLogic";
import { ObjectMapSingleton } from "../models/base";
import { authDataWithInput } from "../utils/authDataWithInput";
import { hasProfanity } from "../utils/censor";
import { AuthDataById } from "../utils/getAuthenticatedData";
import { getParentInfo } from "../utils/getParentInfo";
import { CudInputData, InputsById } from "../utils/types";

type ProfanityFieldsToCheck = {
    tagsConnect?: string[],
    translationsCreate?: object[],
    translationsUpdate?: object[],
}

/**
 * Helper function for profanity check. Recursively finds every field which must be checked for profanity, by: 
 * - Grabbing every field in a "translationsCreate" or "translationsUpdate" object
 * - Grabbing every field specified by the current object's validator's profanityFields array
 * @returns An object with every field that must be checked for profanity
 */
const collectProfanities = (
    input: ProfanityFieldsToCheck,
    objectType?: `${GqlModelType}`,
): Record<string, string[]> => {
    // Initialize result
    const result: Record<string, string[]> = {};
    // Handle base case
    // Get current object's formatter and validator
    const format = objectType ? ObjectMapSingleton.getInstance().map[objectType]?.format : undefined;
    const validate = objectType ? ObjectMapSingleton.getInstance().map[objectType]?.validate : undefined;
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
                // Ignore ID fields and language fields
                if (field === "id" || field === "language" || field.includes("Id")) continue;
                // If the field is an array, recurse
                if (Array.isArray(translation[field])) {
                    handleTranslationsArray(translation[field], result);
                }
                // If the field is an object, recurse
                else if (typeof translation[field] === "object") {
                    handleTranslationsArray([translation[field]], result);
                }
                // Otherwise, add string fields to the result
                else if (typeof translation[field] === "string") result[field] = result[field] ? [...result[field], translation[field]] : [translation[field]];
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
 */
export const profanityCheck = (inputData: CudInputData[], inputsById: InputsById, authDataById: AuthDataById, languages: string[]): void => {
    // Find all fields which must be checked for profanity
    const fieldsToCheck: { [x: string]: string[] } = {};
    for (const item of inputData) {
        // Only check objects being created or updated
        if (!["Create", "Update"].includes(item.actionType)) continue;
        // Only check for objects which are not private. 
        // NOTE: This means that a user could create a private object with profanity in it, and then change it to public. 
        // We'll have to rely on the reporting and reputation system to handle this.
        const { idField, validate } = getLogic(["idField", "validate"], item.objectType, languages, "profanityCheck");
        const existingData = authDataById[item.input[idField]];
        const input = item.input as object;
        const combinedData = authDataWithInput(input, existingData ?? {}, inputsById, authDataById);
        const isPublic = validate?.isPublic(combinedData, (...rest) => getParentInfo(...rest, inputsById), languages);
        if (isPublic === false) continue;
        const newFields = collectProfanities(item.input as ProfanityFieldsToCheck, item.objectType);
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
