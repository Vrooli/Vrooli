import { description, idArray, id, name, title } from './base';
import { nodesCreate, nodesUpdate } from './node';
import { resourcesCreate, resourcesUpdate } from './resource';
import { standardCreate } from './standard';
import { tagsCreate } from './tag';
import * as yup from 'yup';

const isAutomatable = yup.boolean().optional();
const isRequired = yup.boolean().optional();
const instructions = yup.string().max(8192).optional();
const version = yup.string().max(16).optional();

export const inputCreate = yup.object().shape({
    description,
    isRequired,
    name,
    standardConnect: id,
    standardCreate,
})
export const inputUpdate = yup.object().shape({
    description,
    isRequired,
    name,
    // There is purposely no option to delete or update a standard from here
    standardConnect: id,
    standardDisconnect: id,
    standardCreate,
})

export const outputCreate = yup.object().shape({
    description,
    name,
    standardConnect: id,
    standardCreate,
})
export const outputUpdate = yup.object().shape({
    description,
    name,
    // There is purposely no option to delete or update a standard from here
    standardConnect: id,
    standardDisconnect: id,
    standardCreate,
})

/**
 * Information required when creating a routine. 
 */
export const routineCreate = yup.object().shape({
    description,
    instructions,
    isAutomatable,
    title: title.required(),
    version,
    parentId: id, // If forked, the parent's id
    createdByUserId: id, // If associating with yourself, your own id. Cannot associate with another user
    createdByOrganizationId: id, // If associating with an organization you are an admin of, the organization's id
    nodesConnect: idArray,
    nodesCreate,
    inputsCreate: inputCreate.optional(),
    outputsCreate: outputCreate.optional(),
    resourcesContextualCreate: resourcesCreate,
    resourcesExternalCreate: resourcesCreate,
    tagsConnect: idArray,
    tagsCreate,
}, [['createdByUserId', 'createdByOrganizationId']]) // Makes sure you can't associate with both a user and an organization

/**
 * Information required when updating a routine
 */
export const routineUpdate = yup.object().shape({
    description,
    instructions,
    isAutomatable,
    title: title.required(),
    version,
    parentId: id, // If forked, the parent's id
    userId: id, // If associating with yourself, your own id. Cannot associate with another user
    organizationId: id, // If associating with an organization you are an admin of, the organization's id
    nodesConnect: idArray,
    nodesDisconnect: idArray,
    nodesDelete: idArray,
    nodesCreate,
    nodesUpdate,
    inputsDelete: idArray,
    inputsCreate: inputCreate,
    inputsUpdate: inputUpdate,
    outputsDelete: idArray,
    outpusCreate: outputCreate,
    outputsUpdate: outputUpdate,
    resourcesContextualDelete: idArray,
    resourcesContextualCreate: resourcesCreate,
    resourcesContextualUpdate: resourcesUpdate,
    resourcesExternalDelete: idArray,
    resourcesExternalCreate: resourcesCreate,
    resourcesExternalUpdate: resourcesUpdate,
    tagsConnect: idArray,
    tagsDisconnect: idArray,
    tagsCreate,
}, [['userId', 'organizationId']]) // Makes sure you can't transfer to both a user and an organization

export const routinesCreate = yup.array().of(routineCreate.required()).optional();
export const routinesUpdate = yup.array().of(routineUpdate.required()).optional();