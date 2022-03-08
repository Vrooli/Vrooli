import { bio, id, idArray, language, name } from './base';
import { resourceListsCreate, resourceListsUpdate } from './resourceList';
import { tagsCreate } from './tag';
import * as yup from 'yup';

const isOpenToNewMembers = yup.boolean().optional();

export const organizationTranslationCreate = yup.object().shape({
    language,
    bio,
    name: name.required(),
});
export const organizationTranslationUpdate = yup.object().shape({
    id: id.required(),
    language,
    bio,
    name,
});
export const organizationTranslationsCreate = yup.array().of(organizationTranslationCreate.required()).optional();
export const organizationTranslationsUpdate = yup.array().of(organizationTranslationUpdate.required()).optional();

/**
 * Information required when creating an organization. 
 * You are automatically created as an admin
 */
export const organizationCreate = yup.object().shape({
    isOpenToNewMembers,
    // You are automatically added as an admin. IDs you add here will be requested to be added as a member
    membersConnect: idArray,
    resourceListsCreate,
    tagsConnect: idArray,
    tagsCreate,
    translationsCreate: organizationTranslationsCreate,
})

/**
 * Information required when updating an organization
 */
export const organizationUpdate = yup.object().shape({
    isOpenToNewMembers,
    membersConnect: idArray,
    membersDisconnect: idArray,
    resourceListsDelete: idArray,
    resourceListsCreate,
    resourceListsUpdate,
    tagsConnect: idArray,
    tagsDisconnect: idArray,
    tagsCreate,
    translationsDelete: idArray,
    translationsCreate: organizationTranslationsCreate,
    translationsUpdate: organizationTranslationsUpdate,
})

export const organizationsCreate = yup.array().of(organizationCreate.required()).optional();
export const organizationsUpdate = yup.array().of(organizationUpdate.required()).optional();