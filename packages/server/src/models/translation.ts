import { isRelationshipArray, isRelationshipObject, relationshipToPrisma, RelationshipTypes } from "./builder";
import { CODE } from "@shared/consts";
import { CustomError, genErrorCode } from "../events";
import { hasProfanity } from "../utils/censor";
import { ValidateMutationsInput } from "./types";

export const translationMutater = () => ({
    /**
     * Recursively collects translation objects from input and returns them in a flat array.
     * A translation object is any object with stored in a field with the name "translationsCreate" or "translationsUpdate".
     * @param input The input object to collect translations from
     * @returns An array of translation objects
     */
    collectTranslations: (input: { [x: string]: any }): { [x: string]: any }[] => {
        const translations: { [x: string]: any }[] = [];
        // Handle base case
        if (input.translationsCreate) {
            translations.push(...input.translationsCreate);
        }
        if (input.translationsUpdate) {
            translations.push(...input.translationsUpdate);
        }
        // Handle recursive case
        for (const key in input) {
            if (isRelationshipArray(input[key])) {
                for (const item of input[key]) {
                    translations.push(...translationMutater().collectTranslations(item));
                }
            }
            else if (isRelationshipObject(input[key])) {
                translations.push(...translationMutater().collectTranslations(input[key]));
            }
        }
        return translations;
    },
    /**
     * Throws an error if a object's translations contain any banned words.
     * Recursively checks for censored words on any field that:
     * 1. Is not "id"
     * 2. Does not end with "Id"
     * 3. Has a string value
     * @params input An array of objects with translations
     */
    profanityCheck(input: { [x: string]: any }[]): void {
        // Find all translation objects
        const translations: { [x: string]: any }[] = [];
        for (const item of input) {
            translations.push(...translationMutater().collectTranslations(item));
        }
        // Collect all fields in translation objects that are not an ID
        let fields: string[] = [];
        for (const obj of translations) {
            for (const key in obj) {
                if (key !== 'id' && !key.endsWith('Id') && typeof obj[key] === 'string') {
                    fields.push(key);
                }
            }
        }
        // Remove duplicates
        fields = [...new Set(fields)];
        // Check for profanity
        if (hasProfanity(...fields))
            throw new CustomError(CODE.BannedWord, 'Banned word detected', { code: genErrorCode('0115'), fields });
    },
    /**
     * Makes sure there are no more than k line breaks in the specified fields
     * @param input The input to check
     * @param fields The field names to check
     * @param error - The error to throw if failed
     * @param k - The maximum number of line breaks allowed
     */
    validateLineBreaks: (input: any, fields: string[], error: any, k: number = 2): void => {
        // Helper method
        const checkTranslations = (translations: any[], fields: string[]): void => {
            translations.forEach((x: any) => {
                fields.forEach(field => {
                    if (x[field] && x[field].split('\n').length > (k + 1)) {
                        throw new CustomError(error, 'Maximun number of line breaks exceeded', { code: genErrorCode('0116') });
                    }
                })
            })
        }
        if (input.translationsCreate) checkTranslations(input.translationsCreate, fields);
        if (input.translationsUpdate) checkTranslations(input.translationsUpdate, fields);
    },
    /**
    * Add, update, or remove translation data for an object.
    */
    relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        validators: { create: any, update: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'translations', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate
        this.validateMutations({ userId: '', ...formattedInput }, validators);
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Validate adds, updates, and deletes
     */
    async validateMutations<CreateInput, UpdateInput>(
        { createMany, updateMany, deleteMany }: ValidateMutationsInput<CreateInput, UpdateInput>,
        validators: { create: any, update: any }
    ): Promise<void> {
        if (!createMany && !updateMany && !deleteMany) return;
        if (createMany) {
            createMany.forEach(input => validators.create.validateSync(input, { abortEarly: false }));
        }
        if (updateMany) {
            updateMany.forEach(input => validators.update.validateSync(input.data, { abortEarly: false }));
        }
        this.profanityCheck([...(createMany ?? []), ...(updateMany?.map(u => u.data) ?? [])] as any)
    },
})

// NOTE: Not a ModelLogic type because it does not map to a specific table
export const TranslationModel = ({
    ...(translationMutater()),
})