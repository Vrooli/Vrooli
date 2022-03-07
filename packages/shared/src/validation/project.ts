import { description, idArray, id, name, language } from './base';
import { resourcesCreate, resourcesUpdate } from './resource';
import { tagsCreate } from './tag';
import * as yup from 'yup';

const isComplete = yup.boolean().optional();

export const projectTranslationCreate = yup.object().shape({
    language,
    description,
    name: name.required(),
});
export const projectTranslationUpdate = yup.object().shape({
    id: id.required(),
    language,
    description,
    name,
});
export const projectTranslationsCreate = yup.array().of(projectTranslationCreate.required()).optional();
export const projectTranslationsUpdate = yup.array().of(projectTranslationUpdate.required()).optional();

/**
 * Information required when creating a project. 
 */
export const projectCreate = yup.object().shape({
    isComplete,
    parentId: id, // If forked, the parent's id
    createdByUserId: id, // If associating with yourself, your own id. Cannot associate with another user
    createdByOrganizationId: id, // If associating with an organization you are an admin of, the organization's id
    resourcesCreate,
    tagsConnect: idArray,
    tagsCreate,
    translationsCreate: projectTranslationsCreate,
}, [['createdByUserId', 'createdByOrganizationId']]) // Makes sure you can't associate with both a user and an organization

/**
 * Information required when updating a project
 */
export const projectUpdate = yup.object().shape({
    isComplete,
    userId: id, // Allows you to request transfer ownership of the project
    organizationId: id, // Allows you to request transfer ownership of the project
    resourcesDelete: idArray,
    resourcesCreate,
    resourcesUpdate,
    tagsConnect: idArray,
    tagsDisconnect: idArray,
    tagsCreate,
    translationsDelete: idArray,
    translationsCreate: projectTranslationsCreate,
    translationsUpdate: projectTranslationsUpdate,
}, [['userId', 'organizationId']]) // Makes sure you can't transfer to both a user and an organization

export const projectsCreate = yup.array().of(projectCreate.required()).optional();
export const projectsUpdate = yup.array().of(projectUpdate.required()).optional();