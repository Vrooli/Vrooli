import { description, idArray, id, name, title, language } from './base';
import { nodeLinksCreate, nodeLinksUpdate, nodesCreate, nodesUpdate } from './node';
import { resourcesCreate, resourcesUpdate } from './resource';
import { standardCreate } from './standard';
import { tagsCreate } from './tag';
import * as yup from 'yup';

const isAutomatable = yup.boolean().optional();
const isComplete = yup.boolean().optional();
const isInternal = yup.boolean().optional();
const isRequired = yup.boolean().optional();
const instructions = yup.string().max(8192).optional();
const version = yup.string().max(16).optional();

export const inputTranslationCreate = yup.object().shape({
    language,
    description,
});
export const inputTranslationUpdate = yup.object().shape({
    id: id.required(),
    language,
    description,
});
export const inputTranslationsCreate = yup.array().of(inputTranslationCreate.required()).optional();
export const inputTranslationsUpdate = yup.array().of(inputTranslationUpdate.required()).optional();
export const inputCreate = yup.object().shape({
    isRequired,
    name,
    standardConnect: id,
    standardCreate,
})
export const inputUpdate = yup.object().shape({
    isRequired,
    name,
    // There is purposely no option to delete or update a standard from here
    standardConnect: id,
    standardDisconnect: id,
    standardCreate,
})
export const inputsCreate = yup.array().of(inputCreate.required()).optional();
export const inputsUpdate = yup.array().of(inputUpdate.required()).optional();

export const outputTranslationCreate = yup.object().shape({
    language,
    description,
});
export const outputTranslationUpdate = yup.object().shape({
    id: id.required(),
    language,
    description,
});
export const outputTranslationsCreate = yup.array().of(outputTranslationCreate.required()).optional();
export const outputTranslationsUpdate = yup.array().of(outputTranslationUpdate.required()).optional();
export const outputCreate = yup.object().shape({
    name,
    standardConnect: id,
    standardCreate,
    translationsCreate: outputTranslationsCreate,
})
export const outputUpdate = yup.object().shape({
    name,
    // There is purposely no option to delete or update a standard from here
    standardConnect: id,
    standardDisconnect: id,
    standardCreate,
    translationsDelete: idArray,
    translationsCreate: outputTranslationsCreate,
    translationsUpdate: outputTranslationsUpdate,
})
export const outputsCreate = yup.array().of(outputCreate.required()).optional();
export const outputsUpdate = yup.array().of(outputUpdate.required()).optional();

export const routineTranslationCreate = yup.object().shape({
    language,
    description,
    instructions: instructions.required(),
    title: title.required(),
});
export const routineTranslationUpdate = yup.object().shape({
    id: id.required(),
    language,
    description,
    instructions,
    title,
});
export const routineTranslationsCreate = yup.array().of(routineTranslationCreate.required()).optional();
export const routineTranslationsUpdate = yup.array().of(routineTranslationUpdate.required()).optional();

/**
 * Information required when creating a routine. 
 */
export const routineCreate = yup.object().shape({
    isAutomatable,
    isComplete,
    isInternal,
    version,
    parentId: id, // If forked, the parent's id
    createdByUserId: id, // If associating with yourself, your own id. Cannot associate with another user
    createdByOrganizationId: id, // If associating with an organization you are an admin of, the organization's id
    nodesCreate,
    nodeLinksCreate,
    inputsCreate,
    outputsCreate,
    resourcesCreate,
    tagsConnect: idArray,
    tagsCreate,
    translationsCreate: routineTranslationsCreate,
}, [['createdByUserId', 'createdByOrganizationId']]) // Makes sure you can't associate with both a user and an organization

/**
 * Information required when updating a routine
 */
export const routineUpdate = yup.object().shape({
    isAutomatable,
    isComplete,
    isInternal,
    version,
    parentId: id, // If forked, the parent's id
    userId: id, // If associating with yourself, your own id. Cannot associate with another user
    organizationId: id, // If associating with an organization you are an admin of, the organization's id
    nodesDelete: idArray,
    nodesCreate,
    nodesUpdate,
    nodeLinksDelete: idArray,
    nodeLinksCreate,
    nodeLinksUpdate,
    inputsDelete: idArray,
    inputsCreate,
    inputsUpdate,
    outputsDelete: idArray,
    outputsCreate,
    outputsUpdate,
    resourcesDelete: idArray,
    resourcesCreate: resourcesCreate,
    resourcesUpdate: resourcesUpdate,
    tagsConnect: idArray,
    tagsDisconnect: idArray,
    tagsCreate,
    translationsDelete: idArray,
    translationsCreate: routineTranslationsCreate,
    translationsUpdate: routineTranslationsUpdate,
}, [['userId', 'organizationId']]) // Makes sure you can't transfer to both a user and an organization

export const routinesCreate = yup.array().of(routineCreate.required()).optional();
export const routinesUpdate = yup.array().of(routineUpdate.required()).optional();