import { StandardCreateInput, StandardUpdateInput } from "graphql/generated/globalTypes";
import { Standard } from "types";
import { formatForUpdate, hasObjectChanged, shapeResourceListsCreate, shapeResourceListsUpdate, shapeTagsCreate, shapeTagsUpdate } from "utils";

/**
 * Format a standard's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeStandardTranslationsCreate =  (translations: { [x: string]: any }[] | null | undefined): { translationsCreate?: StandardCreateInput['translationsCreate'] } => {
    if (!translations) return { };
    const formatted: StandardCreateInput['translationsCreate'] = [];
    for (const translation of translations) {
        formatted.push({
            id: translation.id,
            language: translation.language,
            description: translation.description,
            jsonVariables: translation.jsonVariables,
        });
    }
    return formatted.length > 0 ? {
        translationsCreate: formatted,
    } : {};
}

/**
 * Format a standard's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeStandardTranslationsUpdate = (
    original: { [x: string]: any }[] | null | undefined,
    updated: { [x: string]: any }[] | null | undefined
): {
    translationsCreate?: StandardUpdateInput['translationsCreate'],
    translationsUpdate?: StandardUpdateInput['translationsUpdate'],
    translationsDelete?: StandardUpdateInput['translationsDelete'],
} => {
    if (!updated) return { };
    if (!original || !Array.isArray(original)) {
        return shapeStandardTranslationsCreate(updated);
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        ...(shapeStandardTranslationsCreate(updated.filter(t => !original.some(o => o.id === t.id)))),
        translationsUpdate: updated.map(t => {
            const ot = original.find(o => o.id === t.id);
            return (ot && hasObjectChanged(ot, t)) ? formatForUpdate(ot, t) : undefined;
        }).filter(t => Boolean(t)) as StandardUpdateInput['translationsUpdate'],
        translationsDelete: original.filter(o => !updated.some(u => u.id === o.id)).map(o => o.id),
    } : {}
}

type ShapeStandardCreateInput = Partial<Standard> & {
    props: Standard['props'],
    type: Standard['type'],
}
/**
 * Format a standard for create mutation.
 * @param standard The standard's information
 * @returns Standard shaped for create mutation
 */
export const shapeStandardCreate =  (standard: ShapeStandardCreateInput | null | undefined): StandardCreateInput | undefined => {
    if (!standard) return undefined;
    // const requiredFields = yupFields(standardCreate as any);
    // console.log('BOOP', requiredFields);
    // // Check standard for required fields TODO make this work
    // const hasRequired = yupObjectContainsRequiredFields(standard, requiredFields);
    // if (!hasRequired) {
    //     PubSub.publish(Pubs.Snack, { message: 'Missing required fields.', severity: 'error' });
    //     return undefined;
    // }
    return {
        default: standard.default,
        isInternal: standard.isInternal,
        name: standard.name,
        props: standard.props,
        yup: standard.yup,
        type: standard.type,
        version: standard.version,
        ...shapeStandardTranslationsCreate(standard.translations),
        ...shapeResourceListsCreate(standard.resourceLists),
        ...shapeTagsCreate(standard.tags ?? []),
    };
}

type ShapeStandardUpdateInput = Partial<Standard> & {
    id: Standard['id'],
};
/**
 * Format a standard for update mutation
 * @param original The original standard's information
 * @param updated The updated standard's information
 * @returns Standard shaped for update mutation
 */
export const shapeStandardUpdate = (original: ShapeStandardUpdateInput | null | undefined, updated: ShapeStandardUpdateInput | null | undefined): StandardUpdateInput | undefined => {
    if (!updated?.id) return undefined;
    if (!original) original = { id: updated.id };
    return {
        id: original.id,
        // makingAnonymous: updated.makingAnonymous, TODO
        ...shapeStandardTranslationsUpdate(original.translations, updated.translations),
        ...shapeResourceListsUpdate(original.resourceLists, updated.resourceLists),
        ...shapeTagsUpdate(original.tags ?? [], updated.tags ?? []),
    };
}