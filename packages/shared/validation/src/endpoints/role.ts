import { description, idArray, id, language, name, req, opt, reqArr } from './base';
import * as yup from 'yup';

export const roleTranslationCreate = yup.object().shape({
    id: req(id),
    language: req(language),
    description: req(description),
});
export const roleTranslationUpdate = yup.object().shape({
    id: req(id),
    language: opt(language),
    description: req(description),
});
export const roleTranslationsCreate = reqArr(roleTranslationCreate)
export const roleTranslationsUpdate = reqArr(roleTranslationUpdate)

export const roleCreate = yup.object().shape({
    id: req(id),
    name: req(name),
    translationsCreate: opt(roleTranslationsCreate),
})

export const roleUpdate = yup.object().shape({
    id: req(id),
    name: opt(name),
    translationsDelete: opt(idArray),
    translationsCreate: opt(roleTranslationsCreate),
    translationsUpdate: opt(roleTranslationsUpdate),
})

export const rolesCreate = reqArr(roleCreate)
export const rolesUpdate = reqArr(roleUpdate)