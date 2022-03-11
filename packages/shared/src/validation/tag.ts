import { description, id, idArray, language } from './base';
import * as yup from 'yup';

const anonymous = yup.boolean().optional(); // Determines if the user will be credited for the tag
const tag = yup.string().min(2).max(128).optional();

export const tagTranslationCreate = yup.object().shape({
    language,
    description,
});
export const tagTranslationUpdate = yup.object().shape({
    id: id.required(),
    language,
    description,
});
export const tagTranslationsCreate = yup.array().of(tagTranslationCreate.required()).optional();
export const tagTranslationsUpdate = yup.array().of(tagTranslationUpdate.required()).optional();

/**
 * Information required when creating a tag
 */
export const tagCreate = yup.object().shape({
    anonymous,
    tag: tag.required(),
    translationsCreate: tagTranslationsCreate,
})

/**
 * Information required when updating a tag
 */
export const tagUpdate = yup.object().shape({
    id: id.required(),
    anonymous,
    tag,
    translationsDelete: idArray,
    translationsCreate: tagTranslationsCreate,
    translationsUpdate: tagTranslationsUpdate,
})

export const tagsCreate = yup.array().of(tagCreate.required()).optional();
export const tagsUpdate = yup.array().of(tagUpdate.required()).optional();
