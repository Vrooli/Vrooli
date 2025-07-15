// AI_CHECK: TYPE_SAFETY=2 | LAST: 2025-07-03
import { type ModelType } from "@vrooli/shared";
import { isRelationshipArray, isRelationshipObject } from "../builders/isOfType.js";
import { CustomError } from "../events/error.js";
import { ModelMap } from "../models/base/index.js";
import { authDataWithInput } from "../utils/authDataWithInput.js";
import { hasProfanity } from "../utils/censor.js";
import { type AuthDataById } from "../utils/getAuthenticatedData.js";
import { type CudInputData, type InputsById } from "../utils/types.js";
import { getParentInfo } from "./permissions.js";

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
function collectProfanities(
    input: ProfanityFieldsToCheck,
    objectType?: `${ModelType}`,
): Record<string, string[]> {
    // Initialize result
    const result: Record<string, string[]> = {};
    // Handle base case
    // Get current object's formatter and validator
    const format = ModelMap.get(objectType, false)?.format;
    const validator = ModelMap.get(objectType, false)?.validate();
    // If validator specifies profanityFields, add them to the result
    if (validator?.profanityFields) {
        for (const field of validator.profanityFields) {
            if (input[field]) result[field] = result[field] ? [...result[field], input[field]] : [input[field]];
        }
    }
    // Helper function to handle translations
    // AI_CHECK: TYPE_SAFETY=server-validators-profanity-type-safety-maintenance-1 | LAST: 2025-07-03 - Replaced any[] with unknown[], added proper type guards
    function handleTranslationsArray(translationsArray: unknown[], result: { [x: string]: string[] }): void {
        for (const translation of translationsArray) {
            if (!translation || typeof translation !== "object") continue;
            const translationObj = translation as Record<string, unknown>;
            for (const field in translationObj) {
                // Ignore ID fields and language fields
                if (field === "id" || field === "language" || field.includes("Id")) continue;
                // If the field is an array, recurse
                if (Array.isArray(translationObj[field])) {
                    handleTranslationsArray(translationObj[field] as unknown[], result);
                }
                // If the field is an object, recurse
                else if (typeof translationObj[field] === "object" && translationObj[field] !== null) {
                    handleTranslationsArray([translationObj[field]], result);
                }
                // Otherwise, add string fields to the result
                else if (typeof translationObj[field] === "string") {
                    const stringValue = translationObj[field] as string;
                    result[field] = result[field] ? [...result[field], stringValue] : [stringValue];
                }
            }
        }
    }
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
    // AI_CHECK: TYPE_SAFETY=server-validators-profanity-type-safety-maintenance-1 | LAST: 2025-07-03 - Replaced any parameter with unknown, added type guard
    function processNestedFields(nestedInput: unknown, nestedObjectType?: `${ModelType}`): void {
        if (!nestedInput || typeof nestedInput !== "object") return;
        const newFields = collectProfanities(nestedInput, nestedObjectType);
        for (const field in newFields) {
            result[field] = result[field] ? [...result[field], ...newFields[field]] : newFields[field];
        }
    }
    for (const key in input) {
        // Find next objectType, if any
        let nextObjectType: `${ModelType}` | undefined;
        // Strip "Create" and "Update" from the end of the key
        let strippedKey = key.endsWith("Create") ? key.slice(0, -"Create".length) : key;
        strippedKey = key.endsWith("Update") ? key.slice(0, -"Update".length) : strippedKey;
        // Translations were already handled above, so skip them here
        if (strippedKey === "translations") continue;
        // Check if stripped key is in validator's validateMap
        if (typeof format?.apiRelMap?.[strippedKey] === "string") {
            nextObjectType = format?.apiRelMap?.[strippedKey] as ModelType;
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
 */
export function profanityCheck(inputData: CudInputData[], inputsById: InputsById, authDataById: AuthDataById): void {
    // Find all fields which must be checked for profanity
    const fieldsToCheck: { [x: string]: string[] } = {};
    for (const item of inputData) {
        // Only check objects being created or updated
        if (!["Create", "Update"].includes(item.action)) continue;
        // Only check for objects which are not private. 
        // NOTE: This means that a user could create a private object with profanity in it, and then change it to public. 
        // We'll have to rely on the reporting and reputation system to handle this.
        const { idField, validate } = ModelMap.getLogic(["idField", "validate"], item.objectType);
        const existingData = authDataById[item.input[idField]];
        const input = item.input as object;
        const combinedData = authDataWithInput(input, existingData ?? {}, inputsById, authDataById);
        const isPublic = validate().isPublic(combinedData, (...rest) => getParentInfo(...rest, inputsById));
        if (isPublic === false) continue;
        const newFields = collectProfanities(item.input as ProfanityFieldsToCheck, item.objectType);
        for (const field in newFields) {
            fieldsToCheck[field] = fieldsToCheck[field] ? [...fieldsToCheck[field], ...newFields[field]] : newFields[field];
        }
    }
    // Check each field for profanity
    for (const field in fieldsToCheck) {
        if (hasProfanity(...fieldsToCheck[field])) {
            throw new CustomError("0115", "BannedWord");
        }
    }
}
