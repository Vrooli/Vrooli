import { bio, idArray, name } from './base';
import { resourcesCreate, resourcesUpdate } from './resource';
import { tagsCreate } from './tag';
import * as yup from 'yup';

/**
 * Information required when creating an organization. 
 * You are automatically created as an admin
 */
export const organizationCreate = yup.object().shape({
    bio,
    name: name.required(),
    // You are automatically added as an admin. IDs you add here will be requested to be added as a member
    membersConnect: idArray,
    resourcesCreate,
    tagsConnect: idArray,
    tagsCreate,
})

/**
 * Information required when updating an organization
 */
export const organizationUpdate = yup.object().shape({
    bio,
    name,
    membersConnect: idArray,
    membersDisconnect: idArray,
    resourcesDelete: idArray,
    resourcesCreate,
    resourcesUpdate,
    tagsConnect: idArray,
    tagsDisconnect: idArray,
    tagsCreate,
})

export const organizationsCreate = yup.array().of(organizationCreate.required()).optional();
export const organizationsUpdate = yup.array().of(organizationUpdate.required()).optional();