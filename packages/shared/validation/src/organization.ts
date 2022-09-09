import { bio, id, idArray, language, name, requiredErrorMessage, tagArray } from './base';
import { resourceListsCreate, resourceListsUpdate } from './resourceList';
import { rolesCreate, rolesUpdate } from './role';
import { tagsCreate } from './tag';
import * as yup from 'yup';

const isOpenToNewMembers = yup.boolean()
const isPrivate = yup.boolean()

export const organizationTranslationCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.required(requiredErrorMessage),
    bio: bio.notRequired().default(undefined),
    name: name.required(requiredErrorMessage),
});
export const organizationTranslationUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.notRequired().default(undefined),
    bio: bio.notRequired().default(undefined),
    name: name.notRequired().default(undefined),
});
export const organizationTranslationsCreate = yup.array().of(organizationTranslationCreate.required(requiredErrorMessage))
export const organizationTranslationsUpdate = yup.array().of(organizationTranslationUpdate.required(requiredErrorMessage))

export const organizationCreateForm = yup.object().shape({
    name: name.required(requiredErrorMessage),
    isOpenToNewMembers: isOpenToNewMembers.notRequired().default(undefined),
    isPrivate: isPrivate.notRequired().default(undefined),
})
export const organizationUpdateForm = organizationCreateForm;
/**
 * Information required when creating an organization. 
 * You are automatically created as an admin
 */
export const organizationCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    isOpenToNewMembers: isOpenToNewMembers.notRequired().default(undefined),
    isPrivate: isPrivate.notRequired().default(undefined),
    // You are automatically added as an admin. IDs you add here will be requested to be added as a member
    membersConnect: idArray.notRequired().default(undefined),
    resourceListsCreate: resourceListsCreate.notRequired().default(undefined),
    rolesCreate: rolesCreate.notRequired().default(undefined),
    tagsConnect: tagArray.notRequired().default(undefined),
    tagsCreate: tagsCreate.notRequired().default(undefined),
    translationsCreate: organizationTranslationsCreate.required(requiredErrorMessage),
})

/**
 * Information required when updating an organization
 */
export const organizationUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    isOpenToNewMembers: isOpenToNewMembers.notRequired().default(undefined),
    isPrivate: isPrivate.notRequired().default(undefined),
    membersConnect: idArray.notRequired().default(undefined),
    membersDisconnect: idArray.notRequired().default(undefined),
    resourceListsDelete: idArray.notRequired().default(undefined),
    resourceListsCreate: resourceListsCreate.notRequired().default(undefined),
    resourceListsUpdate: resourceListsUpdate.notRequired().default(undefined),
    rolesDelete: idArray.notRequired().default(undefined),
    rolesCreate: rolesCreate.notRequired().default(undefined),
    rolesUpdate: rolesUpdate.notRequired().default(undefined),
    tagsConnect: tagArray.notRequired().default(undefined),
    tagsDisconnect: tagArray.notRequired().default(undefined),
    tagsCreate: tagsCreate.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
    translationsCreate: organizationTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: organizationTranslationsUpdate.notRequired().default(undefined),
})

export const organizationsCreate = yup.array().of(organizationCreate.required(requiredErrorMessage))
export const organizationsUpdate = yup.array().of(organizationUpdate.required(requiredErrorMessage))