import { InputItemCreateInput, InputItemTranslationCreateInput, InputItemTranslationUpdateInput, InputItemUpdateInput } from "graphql/generated/globalTypes";
import { RoutineInput, RoutineInputTranslation } from "types";
import { formatForUpdate, hasObjectChanged, shapeStandardCreate } from "utils";
import { shapeCreateList, shapeUpdateList, ShapeWrapper } from "./shapeTools";

type InputTranslationCreate = ShapeWrapper<RoutineInputTranslation> &
    Pick<InputItemTranslationCreateInput, 'language' | 'description'>;
/**
 * Format an input's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeInputTranslationsCreate = (
    translations: InputTranslationCreate[] | null | undefined
): InputItemTranslationCreateInput[] | undefined => shapeCreateList(translations, (translation) => ({
    id: translation.id,
    language: translation.language,
    description: translation.description,
}))

interface InputTranslationUpdate extends InputTranslationCreate { id: string };
/**
 * Format an input's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeInputTranslationsUpdate = (
    original: InputTranslationUpdate[] | null | undefined,
    updated: InputTranslationUpdate[] | null | undefined
): {
    translationsCreate?: InputItemTranslationCreateInput[],
    translationsUpdate?: InputItemTranslationUpdateInput[],
    translationsDelete?: string[],
} => shapeUpdateList(
    original,
    updated,
    'translations',
    shapeInputTranslationsCreate,
    hasObjectChanged,
    formatForUpdate as (original: InputItemTranslationCreateInput, updated: InputItemTranslationCreateInput) => InputItemTranslationUpdateInput | undefined,
)

type InputCreate = ShapeWrapper<RoutineInput> &
    Pick<RoutineInput, 'translations' | 'standard'>;
/**
 * Format an input list for create mutation.
 * @param inputs The input list's information
 * @returns Input list shaped for create mutation
 */
export const shapeInputsCreate = (
    inputs: InputCreate[] | null | undefined
): InputItemCreateInput[] | undefined => shapeCreateList(inputs, (input) => ({
    id: input.id,
    isRequired: input.isRequired,
    name: input.name,
    ...shapeInputTranslationsCreate(input.translations),
    ...shapeStandardCreate(input.standard),
}))

interface InputUpdate extends InputCreate { id: string };
/**
 * Format an input list for update mutation.
 * @param original Original input list
 * @param updated Updated input list
 * @returns Formatted input list
 */
export const shapeInputsUpdate = (
    original: InputUpdate[] | null | undefined,
    updated: InputUpdate[] | null | undefined,
): {
    inputsCreate?: InputItemCreateInput[],
    inputsUpdate?: InputItemUpdateInput[],
    inputsDelete?: string[],
} => shapeUpdateList(
    original,
    updated,
    'inputs',
    shapeInputsCreate,
    hasObjectChanged,
    (o: InputUpdate, u: InputUpdate) => {
        // Connect to standard if it's marked as external. 
        // Otherwise, set as create. The backend will handle the rest, even if 
        // you're updating.
        const shouldConnectToStandard = u.standard && !u.standard.isInternal && u.standard.id;
        return {
            id: o.id,
            isRequired: u.isRequired,
            name: u.name,
            standardConnect: shouldConnectToStandard ? u.standard?.id as string : undefined,
            standardCreate: !shouldConnectToStandard ? shapeStandardCreate(u.standard) : undefined,
            ...shapeInputTranslationsUpdate(o.translations, u.translations),
        }
    },
);