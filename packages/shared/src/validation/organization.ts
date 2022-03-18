import { bio, id, idArray, language, name } from './base';
import { resourceListsCreate, resourceListsUpdate } from './resourceList';
import { tagsCreate } from './tag';
import * as yup from 'yup';

const isOpenToNewMembers = yup.boolean()

export const organizationTranslationCreate = yup.object().shape({
    language: language.required(),
    bio: bio.notRequired().default(undefined),
    name: name.required(),
});
export const organizationTranslationUpdate = yup.object().shape({
    id: id.required(),
    language: language.notRequired().default(undefined),
    bio: bio.notRequired().default(undefined),
    name: name.notRequired().default(undefined),
});
export const organizationTranslationsCreate = yup.array().of(organizationTranslationCreate.required())
export const organizationTranslationsUpdate = yup.array().of(organizationTranslationUpdate.required())

export const organizationCreateForm = yup.object().shape({
    name: name.required(),
    isOpenToNewMembers: isOpenToNewMembers.notRequired().default(undefined),
})
export const organizationUpdateForm = organizationCreateForm;
/**
 * Information required when creating an organization. 
 * You are automatically created as an admin
 */
export const organizationCreate = yup.object().shape({
    isOpenToNewMembers: isOpenToNewMembers.notRequired().default(undefined),
    // You are automatically added as an admin. IDs you add here will be requested to be added as a member
    membersConnect: idArray.notRequired().default(undefined),
    resourceListsCreate: resourceListsCreate.notRequired().default(undefined),
    tagsConnect: idArray.notRequired().default(undefined),
    tagsCreate: tagsCreate.notRequired().default(undefined),
    translationsCreate: organizationTranslationsCreate.required(),
})

/**
 * Information required when updating an organization
 */
export const organizationUpdate = yup.object().shape({
    id: id.required(),
    isOpenToNewMembers: isOpenToNewMembers.notRequired().default(undefined),
    membersConnect: idArray.notRequired().default(undefined),
    membersDisconnect: idArray.notRequired().default(undefined),
    resourceListsDelete: idArray.notRequired().default(undefined),
    resourceListsCreate: resourceListsCreate.notRequired().default(undefined),
    resourceListsUpdate: resourceListsUpdate.notRequired().default(undefined),
    tagsConnect: idArray.notRequired().default(undefined),
    tagsDisconnect: idArray.notRequired().default(undefined),
    tagsCreate: tagsCreate.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
    translationsCreate: organizationTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: organizationTranslationsUpdate.notRequired().default(undefined),
})

export const organizationsCreate = yup.array().of(organizationCreate.required())
export const organizationsUpdate = yup.array().of(organizationUpdate.required())