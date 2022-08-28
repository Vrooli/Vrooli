import { description, idArray, id, language, title } from './base';
import * as yup from 'yup';

export const roleTranslationCreate = yup.object().shape({
    id: id.required(),
    language: language.required(),
    description: description.required(),
});
export const roleTranslationUpdate = yup.object().shape({
    id: id.required(),
    language: language.notRequired().default(undefined),
    description: description.notRequired().default(undefined),
});
export const roleTranslationsCreate = yup.array().of(roleTranslationCreate.required())
export const roleTranslationsUpdate = yup.array().of(roleTranslationUpdate.required())

export const roleCreateForm = yup.object().shape({
    title: title.required(),
})
export const roleUpdateForm = yup.object().shape({
    title: title.notRequired().default(undefined),
})

export const roleCreate = yup.object().shape({
    id: id.required(),
    title: title.required(),
    translationsCreate: roleTranslationsCreate.notRequired().default(undefined),
})

export const roleUpdate = yup.object().shape({
    id: id.required(),
    title: title.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
    translationsCreate: roleTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: roleTranslationsUpdate.notRequired().default(undefined),
})

export const rolesCreate = yup.array().of(roleCreate.required())
export const rolesUpdate = yup.array().of(roleUpdate.required())