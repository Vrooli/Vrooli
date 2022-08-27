import { description, idArray, id, name, language, tagArray } from './base';
import { resourceListsCreate, resourceListsUpdate } from './resourceList';
import { tagsCreate } from './tag';
import * as yup from 'yup';

const isComplete = yup.boolean()
const isPrivate = yup.boolean()

export const projectTranslationCreate = yup.object().shape({
    id: id.required(),
    language: language.required(),
    description: description.notRequired().default(undefined),
    name: name.required(),
});
export const projectTranslationUpdate = yup.object().shape({
    id: id.required(),
    language: language.notRequired().default(undefined),
    description: description.notRequired().default(undefined),
    name: name.notRequired().default(undefined),
});
export const projectTranslationsCreate = yup.array().of(projectTranslationCreate.required())
export const projectTranslationsUpdate = yup.array().of(projectTranslationUpdate.required())

export const projectCreateForm = yup.object().shape({
    description: description.notRequired().default(undefined),
    name: name.required(),
    isPrivate: isPrivate.notRequired().default(undefined),
})
export const projectUpdateForm = projectCreateForm;
/**
 * Information required when creating a project. 
 */
export const projectCreate = yup.object().shape({
    id: id.required(),
    isComplete: isComplete.notRequired().default(undefined),
    isPrivate: isPrivate.notRequired().default(undefined),
    parentId: id.notRequired().default(undefined), // If forked, the parent's id
    createdByUserId: id.notRequired().default(undefined), // If associating with yourself, your own id. Cannot associate with another user
    createdByOrganizationId: id.notRequired().default(undefined), // If associating with an organization you are an admin of, the organization's id
    resourceListsCreate: resourceListsCreate.notRequired().default(undefined),
    tagsConnect: tagArray.notRequired().default(undefined),
    tagsCreate: tagsCreate.notRequired().default(undefined),
    translationsCreate: projectTranslationsCreate.required(),
}, [['createdByUserId', 'createdByOrganizationId']]) // Makes sure you can't associate with both a user and an organization

/**
 * Information required when updating a project
 */
export const projectUpdate = yup.object().shape({
    id: id.required(),
    isComplete: isComplete.notRequired().default(undefined),
    isPrivate: isPrivate.notRequired().default(undefined),
    userId: id.notRequired().default(undefined), // Allows you to request transfer ownership of the project
    organizationId: id.notRequired().default(undefined), // Allows you to request transfer ownership of the project
    resourceListsDelete: idArray.notRequired().default(undefined),
    resourceListsCreate: resourceListsCreate.notRequired().default(undefined),
    resourceListsUpdate: resourceListsUpdate.notRequired().default(undefined),
    tagsConnect: tagArray.notRequired().default(undefined),
    tagsDisconnect: tagArray.notRequired().default(undefined),
    tagsCreate: tagsCreate.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
    translationsCreate: projectTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: projectTranslationsUpdate.notRequired().default(undefined),
}, [['userId', 'organizationId']]) // Makes sure you can't transfer to both a user and an organization

export const projectsCreate = yup.array().of(projectCreate.required())
export const projectsUpdate = yup.array().of(projectUpdate.required())