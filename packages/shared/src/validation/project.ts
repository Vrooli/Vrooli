import { description, idArray, id, name } from './base';
import { resourcesAdd, resourcesUpdate } from './resource';
import { tagsAdd, tagsUpdate } from './tag';
import * as yup from 'yup';

/**
 * Information required when creating a project. 
 */
export const projectAdd = yup.object().shape({
    description,
    name: name.required(),
    parentId: id, // If forked, the parent's id
    createdByUserId: id, // If associating with yourself, your own id. Cannot associate with another user
    createdByOrganizationId: id, // If associating with an organization you are an admin of, the organization's id
    resourcesConnect: idArray,
    resourcesAdd,
    tagsConnect: idArray,
    tagsAdd,
}, [['createdByUserId', 'createdByOrganizationId']]) // Makes sure you can't associate with both a user and an organization

/**
 * Information required when updating a project
 */
export const projectUpdate = yup.object().shape({
    description,
    name,
    userId: id, // Allows you to request transfer ownership of the project
    organizationId: id, // Allows you to request transfer ownership of the project
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
}, [['userId', 'organizationId']]) // Makes sure you can't transfer to both a user and an organization

export const projectsAdd = yup.array().of(projectAdd.required()).optional();
export const projectsUpdate = yup.array().of(projectUpdate.required()).optional();