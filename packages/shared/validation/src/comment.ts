import { blankToUndefined, id, idArray, language, maxStrErr, minStrErr, opt, req, reqArr } from './base';
import * as yup from 'yup';

const createdFor = yup.string().transform(blankToUndefined).oneOf(['Project', 'Routine', 'Standard'])
const text = yup.string().transform(blankToUndefined).min(1, minStrErr).max(8192, maxStrErr)

export const commentTranslationCreate = yup.object().shape({
    id: req(id),
    language: req(language),
    text: req(text),
});
export const commentTranslationUpdate = yup.object().shape({
    id: req(id),
    language: opt(language),
    text: opt(text),
});
export const commentTranslationsCreate = reqArr(commentTranslationCreate)
export const commentTranslationsUpdate = reqArr(commentTranslationUpdate)

/**
 * Information required when creating a comment
 */
export const commentCreate = yup.object().shape({
    id: req(id),
    createdFor: req(createdFor),
    forId: req(id),
    translationsCreate: req(commentTranslationsCreate),
})

/**
 * Information required when updating an organization
 */
export const commentUpdate = yup.object().shape({
    id: req(id),
    translationsDelete: opt(idArray),
    translationsCreate: opt(commentTranslationsCreate),
    translationsUpdate: opt(commentTranslationsUpdate),
})

export const commentsCreate = reqArr(commentCreate)
export const commentsUpdate = reqArr(commentUpdate)