import { description, idArray, id, name, title } from './base';
import { nodesAdd, nodesUpdate } from './node';
import { resourcesAdd, resourcesUpdate } from './resource';
import { standardAdd } from './standard';
import { tagsAdd, tagsUpdate } from './tag';
import * as yup from 'yup';

const isAutomatable = yup.boolean().optional();
const isRequired = yup.boolean().optional();
const instructions = yup.string().max(8192).optional();
const version = yup.string().max(16).optional();

const inputAdd = yup.object().shape({
    description,
    isRequired,
    name,
    standardConnect: id,
    standardAdd,
})
const inputUpdate = yup.object().shape({
    description,
    isRequired,
    name,
    // There is purposely no option to delete or update a standard from here
    standardConnect: id,
    standardDisconnect: id,
    standardAdd,
})

const outputAdd = yup.object().shape({
    description,
    name,
    standardConnect: id,
    standardAdd,
})
const outputUpdate = yup.object().shape({
    description,
    name,
    // There is purposely no option to delete or update a standard from here
    standardConnect: id,
    standardDisconnect: id,
    standardAdd,
})

/**
 * Information required when creating a routine. 
 */
export const routineAdd = yup.object().shape({
    description,
    instructions,
    isAutomatable,
    title: title.required(),
    version,
    parentId: id, // If forked, the parent's id
    createdByUserId: id, // If associating with yourself, your own id. Cannot associate with another user
    createdByOrganizationId: id, // If associating with an organization you are an admin of, the organization's id
    nodesConnect: idArray,
    nodesAdd,
    inputsAdd: inputAdd.optional(),
    outputsAdd: outputAdd.optional(),
    resourcesContextualAdd: resourcesAdd,
    resourcesExternalAdd: resourcesAdd,
    tagsConnect: idArray,
    tagsAdd,
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
    nodesAdd,
    nodesUpdate,
    inputsDelete: idArray,
    inputsAdd: inputAdd,
    inputsUpdate: inputUpdate,
    outputsDelete: idArray,
    outpusAdd: outputAdd,
    outputsUpdate: outputUpdate,
    resourcesContextualDelete: idArray,
    resourcesContextualAdd: resourcesAdd,
    resourcesContextualUpdate: resourcesUpdate,
    resourcesExternalDelete: idArray,
    resourcesExternalAdd: resourcesAdd,
    resourcesExternalUpdate: resourcesUpdate,
    tagsConnect: idArray,
    tagsDisconnect: idArray,
    tagsDelete: idArray,
    tagsAdd,
    tagsUpdate,
}, [['userId', 'organizationId']]) // Makes sure you can't transfer to both a user and an organization

export const routinesAdd = yup.array().of(routineAdd.required()).optional();
export const routinesUpdate = yup.array().of(routineUpdate.required()).optional();