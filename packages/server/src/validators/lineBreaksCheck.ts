// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-02
import { type TranslationKeyError } from "@vrooli/shared";
import { CustomError } from "../events/error.js";

/**
 * Interface for translation objects that may contain string fields
 */
interface TranslationData {
    [key: string]: unknown;
}

/**
 * Interface for objects that may have translation arrays
 */
interface TranslatableInput {
    translationsCreate?: TranslationData[];
    translationsUpdate?: TranslationData[];
    [key: string]: unknown;
}

/**
 * Makes sure there are no more than k line breaks in the specified fields
 * @param input The input to check
 * @param fields The field names to check
 * @param error - The error to throw if failed
 * @param k - The maximum number of line breaks allowed
 */
export function lineBreaksCheck(input: TranslatableInput, fields: string[], error: TranslationKeyError, k = 2): void {
    // Helper function to check translations
    function checkTranslations(translations: TranslationData[], fields: string[]): void {
        translations.forEach((translation) => {
            fields.forEach(field => {
                const value = translation[field];
                if (typeof value === "string" && value.split("\n").length > (k + 1)) {
                    throw new CustomError("0116", error);
                }
            });
        });
    }
    
    // Check translation arrays if they exist
    if (input.translationsCreate) {
        checkTranslations(input.translationsCreate, fields);
    }
    if (input.translationsUpdate) {
        checkTranslations(input.translationsUpdate, fields);
    }
    
    // Check the main object fields
    fields.forEach(field => {
        const value = input[field];
        if (typeof value === "string" && value.split("\n").length > (k + 1)) {
            throw new CustomError("0117", error);
        }
    });
}
