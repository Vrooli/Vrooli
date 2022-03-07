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
        let fields: string[] = [];
        if (translationsCreate) {
            for (const [key, value] of Object.entries(translationsCreate)) {
                if (key !== 'id' && !key.endsWith('Id') && typeof value === 'string') {
                    fields.push(key);
                }
            }
        }
        if (translationsUpdate) {
            for (const [key, value] of Object.entries(translationsUpdate)) {
                if (key !== 'id' && !key.endsWith('Id') && typeof value === 'string') {
                    fields.push(key);
                }
            }
        }
        if (hasProfanity(...fields)) throw new CustomError(CODE.BannedWord)
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
            updateMany.forEach(input => validators.update.validateSync(input, { abortEarly: false }));
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