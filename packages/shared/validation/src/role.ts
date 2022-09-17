import { description, idArray, id, language, title, requiredErrorMessage } from './base';
import * as yup from 'yup';

export const roleTranslationCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.required(requiredErrorMessage),
    description: description.required(requiredErrorMessage),
});
export const roleTranslationUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.notRequired().default(undefined),
    description: description.notRequired().default(undefined),
});
export const roleTranslationsCreate = yup.array().of(roleTranslationCreate.required(requiredErrorMessage))
export const roleTranslationsUpdate = yup.array().of(roleTranslationUpdate.required(requiredErrorMessage))

export const roleCreateForm = yup.object().shape({
    title: title.required(requiredErrorMessage),
})
export const roleUpdateForm = yup.object().shape({
    title: title.notRequired().default(undefined),
})

export const roleCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    title: title.required(requiredErrorMessage),
    translationsCreate: roleTranslationsCreate.notRequired().default(undefined),
})

export const roleUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    title: title.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
    translationsCreate: roleTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: roleTranslationsUpdate.notRequired().default(undefined),
})

export const rolesCreate = yup.array().of(roleCreate.required(requiredErrorMessage))
export const rolesUpdate = yup.array().of(roleUpdate.required(requiredErrorMessage))