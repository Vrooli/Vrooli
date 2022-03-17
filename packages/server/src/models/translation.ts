import { relationshipToPrisma, RelationshipTypes, ValidateMutationsInput } from "./base";
import { CustomError } from "../error";
import { CODE } from "@local/shared";
import { hasProfanity } from "../utils/censor";

//==============================================================
/* #region Custom Components */
//==============================================================

export const translationMutater = () => ({
    /**
     * Determines if a translation contains any banned words.
     * Checks for censored words on any field that:
     * 1. Is not "id"
     * 2. Does not end with "Id"
     * 3. Has a string value
     */
    profanityCheck({
        translationsCreate,
        translationsUpdate,
    }: { [x: string]: any }): void {
        console.log('translationMutater.profanityCheck()', translationsCreate, translationsUpdate);
        let fields: string[] = [];
        if (translationsCreate) {
            console.log('in translationsCreate');
            for (let i = 0; i < translationsCreate.length; i++) {
                const currTranslation = translationsCreate[i];
                console.log("currTranslation", currTranslation);
                for (const [key, value] of Object.entries(currTranslation)) {
                    console.log('key', key, 'value', value);
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
        console.log('in profanityCheck', fields);
        if (hasProfanity(...fields)) throw new CustomError(CODE.BannedWord)
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
                        throw new CustomError(error)
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
        if ((createMany || updateMany || deleteMany) && !userId) throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations');
        if (createMany) {
            createMany.forEach(input => validators.create.validateSync(input, { abortEarly: false }));
        }
        if (updateMany) {
            updateMany.forEach(input => validators.update.validateSync(input.data, { abortEarly: false }));
        }
        this.profanityCheck({ translationsCreate: createMany, translationsUpdate: updateMany });
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function TranslationModel() {
    const mutate = translationMutater();

    return {
        ...mutate,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================