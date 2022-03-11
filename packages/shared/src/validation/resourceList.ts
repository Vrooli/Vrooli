import { bio, description, id, idArray, language, title } from './base';
import { resourcesCreate, resourcesUpdate } from './resource';
import * as yup from 'yup';
import { ResourceListUsedFor } from '../consts';

const index = yup.number().integer().min(0).optional();
const usedFor = yup.string().oneOf(Object.values(ResourceListUsedFor)).optional();

export const resourceListTranslationCreate = yup.object().shape({
    language,
    description,
    title,
});
export const resourceListTranslationUpdate = yup.object().shape({
    id: id.required(),
    language,
    description,
    title,
});
export const resourceListTranslationsCreate = yup.array().of(resourceListTranslationCreate.required()).optional();
export const resourceListTranslationsUpdate = yup.array().of(resourceListTranslationUpdate.required()).optional();


export const resourceListCreate = yup.object().shape({
    index,
    organizationId: id,
    projectId: id,
    routineId: id,
    userId: id,
    usedFor,
    resourcesCreate,
    translationsCreate: resourceListTranslationsCreate,
})

/**
 * Information required when updating an resourceList
 */
export const resourceListUpdate = yup.object().shape({
    index,
    organizationId: id,
    projectId: id,
    routineId: id,
    userId: id,
    usedFor,
    resourcesDelete: idArray,
    resourcesCreate,
    resourcesUpdate,
    translationsDelete: idArray,
    translationsCreate: resourceListTranslationsCreate,
    translationsUpdate: resourceListTranslationsUpdate,
})

export const resourceListsCreate = yup.array().of(resourceListCreate.required()).optional();
export const resourceListsUpdate = yup.array().of(resourceListUpdate.required()).optional();