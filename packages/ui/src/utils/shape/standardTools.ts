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
 * @param originalTranslations Original translations list
 * @param updatedTranslations Updated translations list
 * @returns Formatted translations
 */
export const shapeStandardTranslationsUpdate = (
    originalTranslations: { [x: string]: any }[] | null | undefined,
    updatedTranslations: { [x: string]: any }[] | null | undefined
): {
    translationsCreate?: StandardUpdateInput['translationsCreate'],
    translationsUpdate?: StandardUpdateInput['translationsUpdate'],
    translationsDelete?: StandardUpdateInput['translationsDelete'],
} => {
    if (!updatedTranslations) return { };
    if (!originalTranslations || !Array.isArray(originalTranslations)) {
        return shapeStandardTranslationsCreate(updatedTranslations);
    }
    return Array.isArray(updatedTranslations) && updatedTranslations.length > 0 ? {
        ...(shapeStandardTranslationsCreate(updatedTranslations.filter(t => !originalTranslations.some(o => o.id === t.id)))),
        translationsUpdate: updatedTranslations.map(t => {
            const original = originalTranslations.find(o => o.id === t.id);
            return (original && hasObjectChanged(original, t)) ? formatForUpdate(original, t) : undefined;
        }).filter(t => Boolean(t)) as StandardUpdateInput['translationsUpdate'],
        translationsDelete: originalTranslations.filter(o => !updatedTranslations.some(u => u.id === o.id)).map(o => o.id),
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
 * @param originalStandard The original standard's information
 * @param updatedStandard The updated standard's information
 * @returns Standard shaped for update mutation
 */
export const shapeStandardUpdate = (originalStandard: ShapeStandardUpdateInput | null | undefined, updatedStandard: ShapeStandardUpdateInput | null | undefined): StandardUpdateInput | undefined => {
    if (!updatedStandard?.id) return undefined;
    if (!originalStandard) originalStandard = { id: updatedStandard.id };
    return {
        id: originalStandard.id,
        // makingAnonymous: updatedStandard.makingAnonymous, TODO
        ...shapeStandardTranslationsUpdate(originalStandard.translations, updatedStandard.translations),
        ...shapeResourceListsUpdate(originalStandard.resourceLists, updatedStandard.resourceLists),
        ...shapeTagsUpdate(originalStandard.tags ?? [], updatedStandard.tags ?? []),
    };
}