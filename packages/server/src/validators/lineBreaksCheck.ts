// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-02
import { type TranslationKeyError } from "@vrooli/shared";
import { CustomError } from "../events/error.js";

/**
 * Makes sure there are no more than k line breaks in the specified fields
 * @param input The input to check
 * @param fields The field names to check
 * @param error - The error to throw if failed
 * @param k - The maximum number of line breaks allowed
 */
export function lineBreaksCheck<T extends Record<string, unknown>>(
    input: T & { 
        translationsCreate?: Array<Record<string, unknown>> | null | undefined;
        translationsUpdate?: Array<Record<string, unknown>> | null | undefined;
    }, 
    fields: string[], 
    error: TranslationKeyError, 
    k = 2,
): void {
    // Helper function to check translations
    function checkTranslations(translations: Array<Record<string, unknown>>, fields: string[]): void {
        translations.forEach((translation) => {
            fields.forEach(field => {
                const value = translation[field];
                if (typeof value === "string" && value.split("\n").length > (k + 1)) {
                    throw new CustomError("0116", error);
                }
            });
        });
    }
    
    // Check translation arrays if they exist (filter out null values)
    if (input.translationsCreate && input.translationsCreate !== null) {
        checkTranslations(input.translationsCreate, fields);
    }
    if (input.translationsUpdate && input.translationsUpdate !== null) {
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
