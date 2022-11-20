import { CODE } from "@shared/consts";
import { CustomError } from "../../events";

/**
 * Makes sure there are no more than k line breaks in the specified fields
 * @param input The input to check
 * @param fields The field names to check
 * @param error - The error to throw if failed
 * @param k - The maximum number of line breaks allowed
 */
export const lineBreaksCheck = (input: any, fields: string[], error: CODE, k: number = 2): void => {
    // First, check translations
    const checkTranslations = (translations: any[], fields: string[]): void => {
        translations.forEach((x: any) => {
            fields.forEach(field => {
                if (x[field] && x[field].split('\n').length > (k + 1)) {
                    throw new CustomError(error, 'Maximun number of line breaks exceeded', { trace: '0116' });
                }
            })
        })
    }
    if (input.translationsCreate) checkTranslations(input.translationsCreate, fields);
    if (input.translationsUpdate) checkTranslations(input.translationsUpdate, fields);
    // Then, check the main object
    fields.forEach(field => {
        if (input[field] && input[field].split('\n').length > (k + 1)) {
            throw new CustomError(error, 'Maximun number of line breaks exceeded', { trace: '0117' });
        }
    });
}