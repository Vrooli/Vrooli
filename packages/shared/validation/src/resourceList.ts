import { blankToUndefined, description, id, idArray, language, minNumErr, name, opt, req, reqArr } from './base';
import { resourcesCreate, resourcesUpdate } from './resource';
import * as yup from 'yup';
import { ResourceListUsedFor } from '@shared/consts';

const index = yup.number().integer().min(0, minNumErr)
const usedFor = yup.string().transform(blankToUndefined).oneOf(Object.values(ResourceListUsedFor))

export const resourceListTranslationCreate = yup.object().shape({
    id: req(id),
    language: req(language),
    description: req(description),
    name: opt(name),
});
export const resourceListTranslationUpdate = yup.object().shape({
    id: req(id),
    language: opt(language),
    description: req(description),
    name: opt(name),
});
export const resourceListTranslationsCreate = reqArr(resourceListTranslationCreate)
export const resourceListTranslationsUpdate = reqArr(resourceListTranslationUpdate)


export const resourceListCreate = yup.object().shape({
    id: req(id),
    index: opt(index),
    organizationId: opt(id),
    projectId: opt(id),
    routineId: opt(id),
    userId: opt(id),
    usedFor: opt(usedFor),
    resourcesCreate: opt(resourcesCreate),
    translationsCreate: opt(resourceListTranslationsCreate),
})

/**
 * Information required when updating an resourceList
 */
export const resourceListUpdate = yup.object().shape({
    id: req(id),
    index: opt(index),
    organizationId: opt(id),
    projectId: opt(id),
    routineId: opt(id),
    userId: opt(id),
    usedFor: opt(usedFor),
    resourcesDelete: opt(idArray),
    resourcesCreate: opt(resourcesCreate),
    resourcesUpdate: opt(resourcesUpdate),
    translationsDelete: opt(idArray),
    translationsCreate: opt(resourceListTranslationsCreate),
    translationsUpdate: opt(resourceListTranslationsUpdate),
})

export const resourceListsCreate = reqArr(resourceListCreate)
export const resourceListsUpdate = reqArr(resourceListUpdate)