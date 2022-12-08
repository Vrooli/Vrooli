import { description, idArray, id, language, name, requiredErrorMessage } from './base';
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

export const roleCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    name: name.required(requiredErrorMessage),
    translationsCreate: roleTranslationsCreate.notRequired().default(undefined),
})

export const roleUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    name: name.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
    translationsCreate: roleTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: roleTranslationsUpdate.notRequired().default(undefined),
})

export const rolesCreate = yup.array().of(roleCreate.required(requiredErrorMessage))
export const rolesUpdate = yup.array().of(roleUpdate.required(requiredErrorMessage))