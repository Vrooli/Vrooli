import { bio, blankToUndefined, description, id, idArray, language, minNumberErrorMessage, requiredErrorMessage, title } from './base';
import { resourcesCreate, resourcesUpdate } from './resource';
import * as yup from 'yup';
import { ResourceListUsedFor } from '@shared/consts';

const index = yup.number().integer().min(0, minNumberErrorMessage)
const usedFor = yup.string().transform(blankToUndefined).oneOf(Object.values(ResourceListUsedFor))

export const resourceListTranslationCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.required(requiredErrorMessage),
    description: description.notRequired().default(undefined),
    title: title.notRequired().default(undefined),
});
export const resourceListTranslationUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.notRequired().default(undefined),
    description: description.notRequired().default(undefined),
    title: title.notRequired().default(undefined),
});
export const resourceListTranslationsCreate = yup.array().of(resourceListTranslationCreate.required(requiredErrorMessage))
export const resourceListTranslationsUpdate = yup.array().of(resourceListTranslationUpdate.required(requiredErrorMessage))


export const resourceListCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    index: index.notRequired().default(undefined),
    organizationId: id.notRequired().default(undefined),
    projectId: id.notRequired().default(undefined),
    routineId: id.notRequired().default(undefined),
    userId: id.notRequired().default(undefined),
    usedFor: usedFor.notRequired().default(undefined),
    resourcesCreate: resourcesCreate.notRequired().default(undefined),
    translationsCreate: resourceListTranslationsCreate.notRequired().default(undefined),
})

/**
 * Information required when updating an resourceList
 */
export const resourceListUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    index: index.notRequired().default(undefined),
    organizationId: id.notRequired().default(undefined),
    projectId: id.notRequired().default(undefined),
    routineId: id.notRequired().default(undefined),
    userId: id.notRequired().default(undefined),
    usedFor: usedFor.notRequired().default(undefined),
    resourcesDelete: idArray.notRequired().default(undefined),
    resourcesCreate: resourcesCreate.notRequired().default(undefined),
    resourcesUpdate: resourcesUpdate.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
    translationsCreate: resourceListTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: resourceListTranslationsUpdate.notRequired().default(undefined),
})

export const resourceListsCreate = yup.array().of(resourceListCreate.required(requiredErrorMessage))
export const resourceListsUpdate = yup.array().of(resourceListUpdate.required(requiredErrorMessage))