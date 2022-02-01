import { description, idArray, id, name, nodesAdd, nodesUpdate, resourcesAdd, resourcesUpdate, standardAdd, tagsAdd, tagsUpdate, title } from './';
import * as yup from 'yup';

const isAutomatable = yup.boolean().optional();
const isRequired = yup.boolean().optional();
const instructions = yup.string().max(8192).optional();
const version = yup.string().max(16).optional();

const inputOutputAdd = yup.object().shape({
    description,
    isRequired,
    name,
    standardConnect: id,
    standardAdd,
})
const inputOutputUpdate = yup.object().shape({
    description,
    isRequired,
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
    inputsAdd: inputOutputAdd.optional(),
    outputsAdd: inputOutputAdd.optional(),
    resourcesContextualConnect: idArray,
    resourcesContextualAdd: resourcesAdd,
    resourcesExternalConnect: idArray,
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
    inputsAdd: inputOutputAdd,
    inputsUpdate: inputOutputUpdate,
    outputsDelete: idArray,
    outpusAdd: inputOutputAdd,
    outputsUpdate: inputOutputUpdate,
    resourcesContextualConnect: idArray,
    resourcesContextualDisconnect: idArray,
    resourcesContextualDelete: idArray,
    resourcesContextualAdd: resourcesAdd,
    resourcesContextualUpdate: resourcesUpdate,
    esourcesExternalConnect: idArray,
    resourcesExternalDisconnect: idArray,
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