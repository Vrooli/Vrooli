import { relationshipToPrisma, RelationshipTypes } from "./builder";
import { CODE } from "@shared/consts";
import { CustomError, genErrorCode } from "../events";
import { hasProfanity } from "../utils/censor";
import { ValidateMutationsInput } from "./types";

export type TranslatableObject = {
    translationsCreate?: { [x: string]: any }[] | null;
    translationsUpdate?: { [x: string]: any }[] | null;
}

//==============================================================
/* #region Custom Components */
//==============================================================

export const translationMutater = () => ({
    /**
     * Throws an error if a object's translations contain any banned words.
     * Checks for censored words on any field that:
     * 1. Is not "id"
     * 2. Does not end with "Id"
     * 3. Has a string value
     * @params input An array of objects with translations
     */
    profanityCheck(input: (TranslatableObject | null | undefined)[]): void {
        const objects: TranslatableObject[] = input.filter(Boolean) as TranslatableObject[];
        // Collect all fields that are not an ID
        let fields: string[] = [];
        for (const obj of objects) {
            const { translationsCreate, translationsUpdate } = obj;
            if (translationsCreate) {
                for (let i = 0; i < translationsCreate.length; i++) {
                    const currTranslation = translationsCreate[i];
                    for (const [key, value] of Object.entries(currTranslation)) {
                        if (key !== 'id' && !key.endsWith('Id') && typeof value === 'string') {
                            fields.push(value);
                        }
                    }
                }
            }
            if (translationsUpdate) {
                for (let i = 0; i < translationsUpdate.length; i++) {
                    const currTranslation = translationsUpdate[i];
                    for (const [key, value] of Object.entries(currTranslation)) {
                        if (key !== 'id' && !key.endsWith('Id') && typeof value === 'string') {
                            fields.push(value);
                        }
                    }
                }
            }
        }
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
        userId: string,
        input: { [x: string]: any },
        validators: { create: any, update: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'translations', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate
        this.validateMutations({ userId, ...formattedInput }, validators);
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Validate adds, updates, and deletes
     */
    async validateMutations<CreateInput, UpdateInput>(
        { userId, createMany, updateMany, deleteMany }: ValidateMutationsInput<CreateInput, UpdateInput>,
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

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

// NOTE: Not a ModelLogic type because it does not map to a specific table
export const TranslationModel = ({
    ...(translationMutater()),
})

//==============================================================
/* #endregion Model */
//==============================================================