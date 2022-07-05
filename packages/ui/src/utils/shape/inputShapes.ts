import { InputItemCreateInput, InputItemUpdateInput, RoutineCreateInput, RoutineUpdateInput } from "graphql/generated/globalTypes";
import { RoutineInput } from "types";
import { formatForUpdate, hasObjectChanged, shapeStandardCreate, shapeStandardUpdate } from "utils";
import { findCreatedItems, findRemovedItems, findUpdatedItems } from "./shapeTools";

type InputTranslationCreate = Partial<RoutineInput['translations'][0]> & {
    id: string;
    language: RoutineInput['translations'][0]['language'];
    description: RoutineInput['translations'][0]['description'];
}
/**
 * Format an input's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeInputTranslationsCreate = (
    translations: InputTranslationCreate[] | null | undefined
): InputItemCreateInput['translationsCreate'] | undefined => {
    if (!translations) return undefined;
    const formatted: InputItemCreateInput['translationsCreate'] = [];
    for (const translation of translations) {
        formatted.push({
            id: translation.id,
            language: translation.language,
            description: translation.description,
        });
    }
    return formatted.length > 0 ? formatted : undefined;
}

/**
 * Format an input's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeInputTranslationsUpdate = (
    original: InputTranslationCreate[] | null | undefined,
    updated: InputTranslationCreate[] | null | undefined
): {
    translationsCreate?: InputItemCreateInput['translationsCreate'],
    translationsUpdate?: InputItemUpdateInput['translationsUpdate'],
    translationsDelete?: InputItemUpdateInput['translationsDelete'],
} => {
    if (!updated) return {};
    if (!original || !Array.isArray(original)) {
        return { translationsCreate: shapeInputTranslationsCreate(updated) };
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        translationsCreate: findCreatedItems(original, updated, shapeInputTranslationsCreate),
        translationsUpdate: findUpdatedItems(original, updated, hasObjectChanged, formatForUpdate) as InputItemUpdateInput['translationsUpdate'],
        translationsDelete: findRemovedItems(original, updated),
    } : {}
}

type ShapeInputCreateInput = Partial<RoutineInput> & {
    id: string;
    translations: RoutineInput['translations'];
    standard: RoutineInput['standard'];
}
/**
 * Format an input list for create mutation.
 * @param inputs The input list's information
 * @returns Input list shaped for create mutation
 */
export const shapeInputsCreate = (
    inputs: ShapeInputCreateInput[] | null | undefined
): RoutineCreateInput['inputsCreate'] | undefined => {
    if (!inputs) return undefined;
    const formatted: ShapeInputCreateInput[] = [];
    for (const input of inputs) {
        formatted.push({
            id: input.id,
            isRequired: input.isRequired,
            name: input.name,
            ...shapeInputTranslationsCreate(input.translations),
            ...shapeStandardCreate(input.standard),
        } as any);
    }
    return formatted.length > 0 ? formatted : undefined;
}

type ShapeInputUpdateInput = ShapeInputCreateInput & {

}
/**
 * Format an input list for update mutation.
 * @param original Original input list
 * @param updated Updated input list
 * @returns Formatted input list
 */
export const shapeInputsUpdate = (
    original: ShapeInputUpdateInput[] | null | undefined,
    updated: ShapeInputUpdateInput[] | null | undefined,
): {
    inputsCreate?: RoutineUpdateInput['inputsCreate'],
    inputsUpdate?: RoutineUpdateInput['inputsUpdate'],
    inputsDelete?: string[],
} => {
    if (!updated) return {};
    if (!original || !Array.isArray(original)) {
        return { inputsCreate: shapeInputsCreate(updated) };
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        inputsCreate: findCreatedItems(original, updated, shapeInputsCreate),
        inputsUpdate: findUpdatedItems(original, updated, hasObjectChanged, (o, u) => ({
            id: o.id,
            isRequired: u.isRequired,
            name: u.name,
            ...shapeInputTranslationsUpdate(o.translations, u.translations),
            ...shapeStandardUpdate(o.standard, u.standard),
        })) as RoutineUpdateInput['inputsUpdate'],
        inputsDelete: findRemovedItems(original, updated),
    } : {}
}