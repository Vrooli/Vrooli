import { InputItemCreateInput, InputItemUpdateInput, RoutineCreateInput, RoutineUpdateInput } from "graphql/generated/globalTypes";
import { RoutineInput } from "types";
import { formatForUpdate, hasObjectChanged, shapeStandardCreate, shapeStandardUpdate } from "utils";

/**
 * Format an input's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeInputTranslationsCreate = (translations: { [x: string]: any }[] | null | undefined): { translationsCreate?: InputItemCreateInput['translationsCreate'] } => {
    if (!translations) return {};
    const formatted: InputItemCreateInput['translationsCreate'] = [];
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
 * Format an input's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeInputTranslationsUpdate = (
    original: { [x: string]: any }[] | null | undefined,
    updated: { [x: string]: any }[] | null | undefined
): {
    translationsCreate?: InputItemCreateInput['translationsCreate'],
    translationsUpdate?: InputItemUpdateInput['translationsUpdate'],
    translationsDelete?: InputItemUpdateInput['translationsDelete'],
} => {
    if (!updated) return {};
    if (!original || !Array.isArray(original)) {
        return shapeInputTranslationsCreate(updated);
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        ...(shapeInputTranslationsCreate(updated.filter(t => !original.some(o => o.id === t.id)))),
        translationsUpdate: updated.map(t => {
            const ot = original.find(o => o.id === t.id);
            return (ot && hasObjectChanged(ot, t)) ? formatForUpdate(ot, t) : undefined;
        }).filter(t => Boolean(t)) as InputItemUpdateInput['translationsUpdate'],
        translationsDelete: original.filter(o => !updated.some(u => u.id === o.id)).map(o => o.id),
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
): {
    inputsCreate?: RoutineCreateInput['inputsCreate'],
} => {
    if (!inputs) return {};
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
    return formatted.length > 0 ? {
        inputsCreate: formatted,
    } : {};
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
        return shapeInputsCreate(updated);
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        ...(shapeInputsCreate(updated.filter(t => !original.some(o => o.id === t.id)))),
        inputsUpdate: updated.map(t => {
            const ol = original.find(o => o.id === t.id);
            return (ol && hasObjectChanged(ol, t)) ? {
                id: ol.id,
                isRequired: t.isRequired,
                name: t.name,
                ...shapeInputTranslationsUpdate(ol.translations, t.translations),
                ...shapeStandardUpdate(ol.standard, t.standard),
            } : undefined;
        }).filter(t => Boolean(t)) as RoutineUpdateInput['inputsUpdate'],
        inputsDelete: original.filter(o => !updated.some(u => u.id === o.id)).map(o => o.id),
    } : {}
}