import { relationshipBuilderHelper } from "./builder";

export const translationMutater = () => ({
    /**
    * Add, update, or remove translation data for an object.
    */
    relationshipBuilder(
        userId: string | null,
        data: { [x: string]: any },
        validators: { create: any, update: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        return relationshipBuilderHelper({
            data,
            relationshipName: 'translations',
            isAdd,
            isTransferable: false,
            shape: (_, cuData) => cuData,
            userId: '',
        });
    },
})

// NOTE: Not a ModelLogic type because it does not map to a specific table
export const TranslationModel = ({
    ...(translationMutater()),
})