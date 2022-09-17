import { description, id, idArray, language, requiredErrorMessage, tag } from './base';
import * as yup from 'yup';

const anonymous = yup.boolean() // Determines if the user will be credited for the tag

export const tagTranslationCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.required(requiredErrorMessage),
    description: description.required(requiredErrorMessage),
});
export const tagTranslationUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.notRequired().default(undefined),
    description: description.notRequired().default(undefined),
});
export const tagTranslationsCreate = yup.array().of(tagTranslationCreate.required(requiredErrorMessage))
export const tagTranslationsUpdate = yup.array().of(tagTranslationUpdate.required(requiredErrorMessage))

/**
 * Information required when creating a tag
 */
export const tagCreate = yup.object().shape({
    anonymous: anonymous.notRequired().default(undefined),
    tag: tag.required(requiredErrorMessage),
    translationsCreate: tagTranslationsCreate.notRequired().default(undefined),
})

/**
 * Information required when updating a tag
 */
export const tagUpdate = yup.object().shape({
    anonymous: anonymous.notRequired().default(undefined),
    tag: tag.required(requiredErrorMessage),
    translationsDelete: idArray.notRequired().default(undefined),
    translationsCreate: tagTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: tagTranslationsUpdate.notRequired().default(undefined),
})

export const tagsCreate = yup.array().of(tagCreate.required(requiredErrorMessage))
export const tagsUpdate = yup.array().of(tagUpdate.required(requiredErrorMessage))
