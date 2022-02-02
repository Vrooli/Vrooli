import { bio, idArray, name } from './base';
import { resourcesAdd, resourcesUpdate } from './resource';
import { tagsAdd, tagsUpdate } from './tag';
import * as yup from 'yup';

/**
 * Information required when creating an organization. 
 * You are automatically added as an admin
 */
export const organizationAdd = yup.object().shape({
    bio,
    name: name.required(),
    // You are automatically added as an admin. IDs you add here will be requested to be added as a member
    membersConnect: idArray,
    resourcesConnect: idArray,
    resourcesAdd,
    tagsConnect: idArray,
    tagsAdd,
})

/**
 * Information required when updating an organization
 */
export const organizationUpdate = yup.object().shape({
    bio,
    name,
    membersConnect: idArray,
    membersDisconnect: idArray,
    resourcesConnect: idArray,
    resourcesDisconnect: idArray,
    resourcesDelete: idArray,
    resourcesAdd,
    resourcesUpdate,
    tagsConnect: idArray,
    tagsDisconnect: idArray,
    tagsDelete: idArray,
    tagsAdd,
    tagsUpdate,
})

export const organizationsAdd = yup.array().of(organizationAdd.required()).optional();
export const organizationsUpdate = yup.array().of(organizationUpdate.required()).optional();