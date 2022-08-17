import { description, id, idArray, language, tag } from './base';
import * as yup from 'yup';

const anonymous = yup.boolean() // Determines if the user will be credited for the tag

export const tagTranslationCreate = yup.object().shape({
    id: id.required(),
    language: language.required(),
    description: description.required(),
});
export const tagTranslationUpdate = yup.object().shape({
    id: id.required(),
    language: language.notRequired().default(undefined),
    description: description.notRequired().default(undefined),
});
export const tagTranslationsCreate = yup.array().of(tagTranslationCreate.required())
export const tagTranslationsUpdate = yup.array().of(tagTranslationUpdate.required())

/**
 * Information required when creating a tag
 */
export const tagCreate = yup.object().shape({
    anonymous: anonymous.notRequired().default(undefined),
    tag: tag.required(),
    translationsCreate: tagTranslationsCreate.notRequired().default(undefined),
})

/**
 * Information required when updating a tag
 */
export const tagUpdate = yup.object().shape({
    anonymous: anonymous.notRequired().default(undefined),
    tag: tag.required(),
    translationsDelete: idArray.notRequired().default(undefined),
    translationsCreate: tagTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: tagTranslationsUpdate.notRequired().default(undefined),
})

export const tagsCreate = yup.array().of(tagCreate.required())
export const tagsUpdate = yup.array().of(tagUpdate.required())
