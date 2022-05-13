import { id, idArray, language } from './base';
import * as yup from 'yup';

const createdFor = yup.string().oneOf(['Project', 'Routine', 'Standard'])
const text = yup.string().min(1).max(8192)

export const commentTranslationCreate = yup.object().shape({
    language: language.required(),
    text: text.required(),
});
export const commentTranslationUpdate = yup.object().shape({
    id: id.required(),
    language: language.notRequired().default(undefined),
    text: text.notRequired().default(undefined),
});
export const commentTranslationsCreate = yup.array().of(commentTranslationCreate.required())
export const commentTranslationsUpdate = yup.array().of(commentTranslationUpdate.required())

/**
 * Information required when creating a comment
 */
export const commentCreate = yup.object().shape({
    createdFor: createdFor.required(),
    forId: id.required(),
    translationsCreate: commentTranslationsCreate.required(),
})

/**
 * Information required when updating an organization
 */
export const commentUpdate = yup.object().shape({
    id: id.required(),
    translationsDelete: idArray.notRequired().default(undefined),
    translationsCreate: commentTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: commentTranslationsUpdate.notRequired().default(undefined),
})

export const commentsCreate = yup.array().of(commentCreate.required())
export const commentsUpdate = yup.array().of(commentUpdate.required())