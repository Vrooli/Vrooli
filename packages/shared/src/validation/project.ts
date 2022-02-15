import { description, idArray, id, name } from './base';
import { resourcesCreate, resourcesUpdate } from './resource';
import { tagsCreate } from './tag';
import * as yup from 'yup';

/**
 * Information required when creating a project. 
 */
export const projectCreate = yup.object().shape({
    description,
    name: name.required(),
    parentId: id, // If forked, the parent's id
    createdByUserId: id, // If associating with yourself, your own id. Cannot associate with another user
    createdByOrganizationId: id, // If associating with an organization you are an admin of, the organization's id
    resourcesCreate,
    tagsConnect: idArray,
    tagsCreate,
}, [['createdByUserId', 'createdByOrganizationId']]) // Makes sure you can't associate with both a user and an organization

/**
 * Information required when updating a project
 */
export const projectUpdate = yup.object().shape({
    description,
    name,
    userId: id, // Allows you to request transfer ownership of the project
    organizationId: id, // Allows you to request transfer ownership of the project
    resourcesDelete: idArray,
    resourcesCreate,
    resourcesUpdate,
    tagsConnect: idArray,
    tagsDisconnect: idArray,
    tagsCreate,
}, [['userId', 'organizationId']]) // Makes sure you can't transfer to both a user and an organization

export const projectsCreate = yup.array().of(projectCreate.required()).optional();
export const projectsUpdate = yup.array().of(projectUpdate.required()).optional();