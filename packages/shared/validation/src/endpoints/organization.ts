import { bio, id, idArray, language, name, opt, req, reqArr, tagArray } from './base';
import { resourceListsCreate, resourceListsUpdate } from './resourceList';
import { rolesCreate, rolesUpdate } from './role';
import { tagsCreate } from './tag';
import * as yup from 'yup';

const isOpenToNewMembers = yup.boolean()
const isPrivate = yup.boolean()

export const organizationTranslationCreate = yup.object().shape({
    id: req(id),
    language: req(language),
    bio: opt(bio),
    name: req(name),
});
export const organizationTranslationUpdate = yup.object().shape({
    id: req(id),
    language: opt(language),
    bio: opt(bio),
    name: opt(name),
});
export const organizationTranslationsCreate = reqArr(organizationTranslationCreate)
export const organizationTranslationsUpdate = reqArr(organizationTranslationUpdate)

/**
 * Information required when creating an organization. 
 * You are automatically created as an admin
 */
export const organizationCreate = yup.object().shape({
    id: req(id),
    isOpenToNewMembers: opt(isOpenToNewMembers),
    isPrivate: opt(isPrivate),
    // You are automatically added as an admin. IDs you add here will be requested to be added as a member
    membersConnect: opt(idArray),
    resourceListsCreate: opt(resourceListsCreate),
    rolesCreate: opt(rolesCreate),
    tagsConnect: opt(tagArray),
    tagsCreate: opt(tagsCreate),
    translationsCreate: req(organizationTranslationsCreate),
})

/**
 * Information required when updating an organization
 */
export const organizationUpdate = yup.object().shape({
    id: req(id),
    isOpenToNewMembers: opt(isOpenToNewMembers),
    isPrivate: opt(isPrivate),
    membersConnect: opt(idArray),
    membersDisconnect: opt(idArray),
    resourceListsDelete: opt(idArray),
    resourceListsCreate: opt(resourceListsCreate),
    resourceListsUpdate: opt(resourceListsUpdate),
    rolesDelete: opt(idArray),
    rolesCreate: opt(rolesCreate),
    rolesUpdate: opt(rolesUpdate),
    tagsConnect: opt(tagArray),
    tagsDisconnect: opt(tagArray),
    tagsCreate: opt(tagsCreate),
    translationsDelete: opt(idArray),
    translationsCreate: opt(organizationTranslationsCreate),
    translationsUpdate: opt(organizationTranslationsUpdate),
})

export const organizationsCreate = reqArr(organizationCreate)
export const organizationsUpdate = reqArr(organizationUpdate)