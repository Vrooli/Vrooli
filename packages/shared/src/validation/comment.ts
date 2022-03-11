import { id, idArray, language } from './base';
import * as yup from 'yup';

const createdFor = yup.string().oneOf(['Project', 'Routine', 'Standard']).optional();
const text = yup.string().min(1).max(8192).optional();

export const commentTranslationCreate = yup.object().shape({
    language,
    text: text.required(),
});
export const commentTranslationUpdate = yup.object().shape({
    id: id.required(),
    language,
    text,
});
export const commentTranslationsCreate = yup.array().of(commentTranslationCreate.required()).optional();
export const commentTranslationsUpdate = yup.array().of(commentTranslationUpdate.required()).optional();

/**
 * Information required when creating a comment
 */
export const commentCreate = yup.object().shape({
    createdFor: createdFor.required(),
    forId: id.required(),
    translationsCreate: commentTranslationsCreate,
})

/**
 * Information required when updating an organization
 */
export const commentUpdate = yup.object().shape({
    id: id.required(),
    translationsDelete: idArray,
    translationsCreate: commentTranslationsCreate,
    translationsUpdate: commentTranslationsUpdate,
})