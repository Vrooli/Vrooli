import { OutputItemCreateInput, OutputItemUpdateInput, RoutineCreateInput, RoutineUpdateInput } from "graphql/generated/globalTypes";
import { RoutineOutput } from "types";
import { formatForUpdate, hasObjectChanged, shapeStandardCreate, shapeStandardUpdate } from "utils";

/**
 * Format an output's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeOutputTranslationsCreate = (translations: { [x: string]: any }[] | null | undefined): { translationsCreate?: OutputItemCreateInput['translationsCreate'] } => {
    if (!translations) return {};
    const formatted: OutputItemCreateInput['translationsCreate'] = [];
    for (const translation of translations) {
        formatted.push({
            id: translation.id,
            language: translation.language,
            description: translation.description,
        });
    }
    return formatted.length > 0 ? {
        translationsCreate: formatted,
    } : {};
}

/**
 * Format an output's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeOutputTranslationsUpdate = (
    original: { [x: string]: any }[] | null | undefined,
    updated: { [x: string]: any }[] | null | undefined
): {
    translationsCreate?: OutputItemCreateInput['translationsCreate'],
    translationsUpdate?: OutputItemUpdateInput['translationsUpdate'],
    translationsDelete?: OutputItemUpdateInput['translationsDelete'],
} => {
    if (!updated) return {};
    if (!original || !Array.isArray(original)) {
        return shapeOutputTranslationsCreate(updated);
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        ...(shapeOutputTranslationsCreate(updated.filter(t => !original.some(o => o.id === t.id)))),
        translationsUpdate: updated.map(t => {
            const ot = original.find(o => o.id === t.id);
            return (ot && hasObjectChanged(ot, t)) ? formatForUpdate(ot, t) : undefined;
        }).filter(t => Boolean(t)) as OutputItemUpdateInput['translationsUpdate'],
        translationsDelete: original.filter(o => !updated.some(u => u.id === o.id)).map(o => o.id),
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
): {
    outputsCreate?: RoutineCreateInput['outputsCreate'],
} => {
    if (!outputs) return {};
    const formatted: ShapeOutputCreateInput[] = [];
    for (const input of outputs) {
        formatted.push({
            id: input.id,
            name: input.name,
            ...shapeOutputTranslationsCreate(input.translations),
            ...shapeStandardCreate(input.standard),
        } as any);
    }
    return formatted.length > 0 ? {
        outputsCreate: formatted,
    } : {};
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
        return shapeOutputsCreate(updated);
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        ...(shapeOutputsCreate(updated.filter(t => !original.some(o => o.id === t.id)))),
        outputsUpdate: updated.map(t => {
            const ol = original.find(o => o.id === t.id);
            return (ol && hasObjectChanged(ol, t)) ? {
                id: ol.id,
                name: t.name,
                ...shapeOutputTranslationsUpdate(ol.translations, t.translations),
                ...shapeStandardUpdate(ol.standard, t.standard),
            } : undefined;
        }).filter(t => Boolean(t)) as RoutineUpdateInput['outputsUpdate'],
        outputsDelete: original.filter(o => !updated.some(u => u.id === o.id)).map(o => o.id),
    } : {}
}