import { OutputItemCreateInput, OutputItemUpdateInput, RoutineCreateInput, RoutineUpdateInput } from "graphql/generated/globalTypes";
import { RoutineOutput } from "types";
import { formatForUpdate, hasObjectChanged, shapeStandardCreate, shapeStandardUpdate } from "utils";
import { findCreatedItems, findRemovedItems, findUpdatedItems } from "./shapeTools";

type OutputTranslationCreate = Partial<RoutineOutput['translations'][0]> & { 
    id: string;
    language: RoutineOutput['translations'][0]['language'];
}
/**
 * Format an output's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeOutputTranslationsCreate = (
    translations: OutputTranslationCreate[] | null | undefined
): OutputItemCreateInput['translationsCreate'] | undefined => {
    if (!translations) return undefined;
    const formatted: OutputItemCreateInput['translationsCreate'] = [];
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
 * Format an output's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeOutputTranslationsUpdate = (
    original: OutputTranslationCreate[] | null | undefined,
    updated: OutputTranslationCreate[] | null | undefined
): {
    translationsCreate?: OutputItemCreateInput['translationsCreate'],
    translationsUpdate?: OutputItemUpdateInput['translationsUpdate'],
    translationsDelete?: OutputItemUpdateInput['translationsDelete'],
} => {
    if (!updated) return {};
    if (!original || !Array.isArray(original)) {
        return { translationsCreate: shapeOutputTranslationsCreate(updated) };
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        translationsCreate: findCreatedItems(original, updated, shapeOutputTranslationsCreate),
        translationsUpdate: findUpdatedItems(original, updated, hasObjectChanged, formatForUpdate) as OutputItemUpdateInput['translationsUpdate'],
        translationsDelete: findRemovedItems(original, updated),
    } : {}
}

type ShapeOutputCreateInput = Partial<RoutineOutput> & {
    id: string;
    translations: RoutineOutput['translations'];
    standard: RoutineOutput['standard'];
}
/**
 * Format an output list for create mutation.
 * @param outputs The output list's information
 * @returns Output list shaped for create mutation
 */
export const shapeOutputsCreate = (
    outputs: ShapeOutputCreateInput[] | null | undefined
): RoutineCreateInput['outputsCreate'] | undefined => {
    if (!outputs) return undefined;
    const formatted: ShapeOutputCreateInput[] = [];
    for (const input of outputs) {
        formatted.push({
            id: input.id,
            name: input.name,
            ...shapeOutputTranslationsCreate(input.translations),
            ...shapeStandardCreate(input.standard),
        } as any);
    }
    return formatted.length > 0 ? formatted : undefined;
}

type ShapeOutputUpdateInput = ShapeOutputCreateInput & {

}
/**
 * Format an output list for update mutation.
 * @param original Original output list
 * @param updated Updated output list
 * @returns Formatted output list
 */
export const shapeOutputsUpdate = (
    original: ShapeOutputUpdateInput[] | null | undefined,
    updated: ShapeOutputUpdateInput[] | null | undefined,
): {
    outputsCreate?: RoutineUpdateInput['outputsCreate'],
    outputsUpdate?: RoutineUpdateInput['outputsUpdate'],
    outputsDelete?: string[],
} => {
    if (!updated) return {};
    if (!original || !Array.isArray(original)) {
        return { outputsCreate: shapeOutputsCreate(updated) };
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        outputsCreate: findCreatedItems(original, updated, shapeOutputsCreate),
        outputsUpdate: findUpdatedItems(original, updated, hasObjectChanged, (o, u) => ({
            id: o.id,
            name: u.name,
            ...shapeOutputTranslationsUpdate(o.translations, u.translations),
            ...shapeStandardUpdate(o.standard, u.standard),
        })) as RoutineUpdateInput['outputsUpdate'],
        outputsDelete: findRemovedItems(original, updated),
    } : {}
}