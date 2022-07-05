import { StandardCreateInput, StandardUpdateInput } from "graphql/generated/globalTypes";
import { Standard } from "types";
import { formatForUpdate, hasObjectChanged, shapeResourceListsCreate, shapeResourceListsUpdate, shapeTagsCreate, shapeTagsUpdate } from "utils";
import { findCreatedItems, findRemovedItems, findUpdatedItems } from "./shapeTools";

type StandardTranslationCreate = Partial<Standard['translations'][0]> & { 
    id: string;
    language: Standard['translations'][0]['language'];
    jsonVariable: Standard['translations'][0]['jsonVariable'];
}
/**
 * Format a standard's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeStandardTranslationsCreate = (
    translations: StandardTranslationCreate[] | null | undefined
): StandardCreateInput['translationsCreate'] | undefined => {
    if (!translations) return undefined;
    const formatted: StandardCreateInput['translationsCreate'] = [];
    for (const translation of translations) {
        formatted.push({
            id: translation.id,
            language: translation.language,
            description: translation.description,
            jsonVariable: translation.jsonVariable,
        });
    }
    return formatted.length > 0 ? formatted : undefined;
}

/**
 * Format a standard's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeStandardTranslationsUpdate = (
    original: StandardTranslationCreate[] | null | undefined,
    updated: StandardTranslationCreate[] | null | undefined
): {
    translationsCreate?: StandardUpdateInput['translationsCreate'],
    translationsUpdate?: StandardUpdateInput['translationsUpdate'],
    translationsDelete?: StandardUpdateInput['translationsDelete'],
} => {
    if (!updated) return { };
    if (!original || !Array.isArray(original)) {
        return { translationsCreate: shapeStandardTranslationsCreate(updated) };
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        translationsCreate: findCreatedItems(original, updated, shapeStandardTranslationsCreate),
        translationsUpdate: findUpdatedItems(original, updated, hasObjectChanged, formatForUpdate) as StandardUpdateInput['translationsUpdate'],
        translationsDelete: findRemovedItems(original, updated),
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