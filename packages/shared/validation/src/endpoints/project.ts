import { description, idArray, id, name, language, tagArray, req, reqArr, opt } from './base';
import { resourceListsCreate, resourceListsUpdate } from './resourceList';
import { tagsCreate } from './tag';
import * as yup from 'yup';

const isComplete = yup.boolean()
const isPrivate = yup.boolean()

export const projectTranslationCreate = yup.object().shape({
    id: req(id),
    language: req(language),
    description: req(description),
    name: req(name),
});
export const projectTranslationUpdate = yup.object().shape({
    id: req(id),
    language: opt(language),
    description: req(description),
    name: opt(name),
});
export const projectTranslationsCreate = reqArr(projectTranslationCreate)
export const projectTranslationsUpdate = reqArr(projectTranslationUpdate)

/**
 * Information required when creating a project. 
 */
export const projectCreate = yup.object().shape({
    id: req(id),
    isComplete: opt(isComplete),
    isPrivate: opt(isPrivate),
    parentId: opt(id), // If forked, the parent's id
    createdByUserId: opt(id), // If associating with yourself, your own id. Cannot associate with another user
    createdByOrganizationId: opt(id), // If associating with an organization you are an admin of, the organization's id
    resourceListsCreate: opt(resourceListsCreate),
    tagsConnect: opt(tagArray),
    tagsCreate: opt(tagsCreate),
    translationsCreate: req(projectTranslationsCreate),
}, [['createdByUserId', 'createdByOrganizationId']]) // Makes sure you can't associate with both a user and an organization

/**
 * Information required when updating a project
 */
export const projectUpdate = yup.object().shape({
    id: req(id),
    isComplete: opt(isComplete),
    isPrivate: opt(isPrivate),
    userId: opt(id), // Allows you to request transfer ownership of the project
    organizationId: opt(id), // Allows you to request transfer ownership of the project
    resourceListsDelete: opt(idArray),
    resourceListsCreate: opt(resourceListsCreate),
    resourceListsUpdate: opt(resourceListsUpdate),
    tagsConnect: opt(tagArray),
    tagsDisconnect: opt(tagArray),
    tagsCreate: opt(tagsCreate),
    translationsDelete: opt(idArray),
    translationsCreate: opt(projectTranslationsCreate),
    translationsUpdate: opt(projectTranslationsUpdate),
}, [['userId', 'organizationId']]) // Makes sure you can't transfer to both a user and an organization

export const projectsCreate = reqArr(projectCreate)
export const projectsUpdate = reqArr(projectUpdate)