import { StandardCreateInput, StandardTranslationCreateInput, StandardTranslationUpdateInput, StandardUpdateInput } from "graphql/generated/globalTypes";
import { Standard, StandardTranslation } from "types";
import { formatForUpdate, hasObjectChanged, shapeResourceListsCreate, shapeResourceListsUpdate, shapeTagsCreate, shapeTagsUpdate } from "utils";
import { shapeCreateList, shapeUpdate, shapeUpdateList, ShapeWrapper } from "./shapeTools";

type StandardTranslationCreate = ShapeWrapper<StandardTranslation> &
    Pick<StandardTranslation, 'language' | 'jsonVariable'>;
/**
 * Format a standard's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeStandardTranslationsCreate = (
    translations: StandardTranslationCreate[] | null | undefined
): StandardTranslationCreateInput[] | undefined => {
    return shapeCreateList(translations, (translation) => ({
        id: translation.id,
        language: translation.language,
        description: translation.description,
        jsonVariable: translation.jsonVariable,
    }))
}

interface StandardTranslationUpdate extends StandardTranslationCreate { id: string };
/**
 * Format a standard's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeStandardTranslationsUpdate = (
    original: StandardTranslationUpdate[] | null | undefined,
    updated: StandardTranslationUpdate[] | null | undefined
): {
    translationsCreate?: StandardTranslationCreateInput[],
    translationsUpdate?: StandardTranslationUpdateInput[],
    translationsDelete?: string[],
} => shapeUpdateList(
    original,
    updated,
    'translations',
    shapeStandardTranslationsCreate,
    hasObjectChanged,
    formatForUpdate as (original: StandardTranslationUpdate, updated: StandardTranslationUpdate) => StandardTranslationUpdateInput | undefined,
)

type StandardCreate = ShapeWrapper<Standard> &
    Pick<Standard, 'props' | 'type'>;
/**
 * Format a standard for create mutation.
 * @param standard The standard's information
 * @returns Standard shaped for create mutation
 */
export const shapeStandardCreate = (standard: StandardCreate | null | undefined): StandardCreateInput | undefined => {
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

interface StandardUpdate extends StandardCreate { id: string };
/**
 * Format a standard for update mutation
 * @param original The original standard's information
 * @param updated The updated standard's information
 * @returns Standard shaped for update mutation
 */
export const shapeStandardUpdate = (
    original: StandardUpdate,
    updated: StandardUpdate | null | undefined
): StandardUpdateInput | undefined => shapeUpdate(original, updated, (o, u) => ({
    id: o.id,
    // makingAnonymous: updated.makingAnonymous, TODO
    ...shapeStandardTranslationsUpdate(o.translations, u.translations),
    ...shapeResourceListsUpdate(o.resourceLists, u.resourceLists),
    ...shapeTagsUpdate(o.tags ?? [], u.tags ?? []),
}))