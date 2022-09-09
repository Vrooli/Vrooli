import { id, idArray, language, maxStringErrorMessage, minStringErrorMessage, requiredErrorMessage } from './base';
import * as yup from 'yup';

const createdFor = yup.string().oneOf(['Project', 'Routine', 'Standard'])
const text = yup.string().min(1, minStringErrorMessage).max(8192, maxStringErrorMessage)

export const commentTranslationCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.required(requiredErrorMessage),
    text: text.required(requiredErrorMessage),
});
export const commentTranslationUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.notRequired().default(undefined),
    text: text.notRequired().default(undefined),
});
export const commentTranslationsCreate = yup.array().of(commentTranslationCreate.required(requiredErrorMessage))
export const commentTranslationsUpdate = yup.array().of(commentTranslationUpdate.required(requiredErrorMessage))

/**
 * Information required when creating a comment
 */
export const commentCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    createdFor: createdFor.required(requiredErrorMessage),
    forId: id.required(requiredErrorMessage),
    translationsCreate: commentTranslationsCreate.required(requiredErrorMessage),
})

export const commentCreateForm = yup.object().shape({
    comment: text.required(requiredErrorMessage),
})

/**
 * Information required when updating an organization
 */
export const commentUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    translationsDelete: idArray.notRequired().default(undefined),
    translationsCreate: commentTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: commentTranslationsUpdate.notRequired().default(undefined),
})

export const commentsCreate = yup.array().of(commentCreate.required(requiredErrorMessage))
export const commentsUpdate = yup.array().of(commentUpdate.required(requiredErrorMessage))