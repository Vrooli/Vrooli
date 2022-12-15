import { description, id, idArray, language, opt, req, reqArr, tag } from './base';
import * as yup from 'yup';

const anonymous = yup.boolean() // Determines if the user will be credited for the tag

export const tagTranslationCreate = yup.object().shape({
    id: req(id),
    language: req(language),
    description: req(description),
});
export const tagTranslationUpdate = yup.object().shape({
    id: req(id),
    language: opt(language),
    description: req(description),
});
export const tagTranslationsCreate = reqArr(tagTranslationCreate)
export const tagTranslationsUpdate = reqArr(tagTranslationUpdate)

/**
 * Information required when creating a tag
 */
export const tagCreate = yup.object().shape({
    anonymous: opt(anonymous),
    tag: req(tag),
    translationsCreate: opt(tagTranslationsCreate),
})

/**
 * Information required when updating a tag
 */
export const tagUpdate = yup.object().shape({
    anonymous: opt(anonymous),
    tag: req(tag),
    translationsDelete: opt(idArray),
    translationsCreate: opt(tagTranslationsCreate),
    translationsUpdate: opt(tagTranslationsUpdate),
})

export const tagsCreate = reqArr(tagCreate)
export const tagsUpdate = reqArr(tagUpdate)
