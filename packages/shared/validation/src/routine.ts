import { description, idArray, id, name, language, tagArray, helpText, maxStringErrorMessage, requiredErrorMessage, version, blankToUndefined } from './base';
import { nodeLinksCreate, nodeLinksUpdate, nodesCreate, nodesUpdate } from './node';
import { resourceListsCreate, resourceListsUpdate } from './resourceList';
import { standardCreate } from './standard';
import { tagsCreate } from './tag';
import * as yup from 'yup';

const isAutomatable = yup.boolean()
const isComplete = yup.boolean()
const isInternal = yup.boolean()
const isPrivate = yup.boolean()
const isRequired = yup.boolean()
const instructions = yup.string().transform(blankToUndefined).max(8192, maxStringErrorMessage)

export const inputTranslationCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.required(requiredErrorMessage),
    description: description.notRequired().default(undefined),
    helpText: helpText.notRequired().default(undefined),
});
export const inputTranslationUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.notRequired().default(undefined),
    description: description.notRequired().default(undefined),
    helpText: helpText.notRequired().default(undefined),
});
export const inputTranslationsCreate = yup.array().of(inputTranslationCreate.required(requiredErrorMessage))
export const inputTranslationsUpdate = yup.array().of(inputTranslationUpdate.required(requiredErrorMessage))
export const inputCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    isRequired: isRequired.notRequired().default(undefined),
    name: name.required(requiredErrorMessage),
    standardConnect: id.notRequired().default(undefined),
    standardCreate: standardCreate.notRequired().default(undefined),
}, [['standardConnect', 'standardCreate']]) // Can only pick one standard
export const inputUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    isRequired: isRequired.notRequired().default(undefined),
    name: name.notRequired().default(undefined),
    // There is purposely no option to delete or update a standard from here
    standardConnect: id.notRequired().default(undefined),
    standardDisconnect: id.notRequired().default(undefined),
    standardCreate: standardCreate.notRequired().default(undefined),
})
export const inputsCreate = yup.array().of(inputCreate.required(requiredErrorMessage))
export const inputsUpdate = yup.array().of(inputUpdate.required(requiredErrorMessage))

export const outputTranslationCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.required(requiredErrorMessage),
    description: description.notRequired().default(undefined),
    helpText: helpText.notRequired().default(undefined),
});
export const outputTranslationUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.notRequired().default(undefined),
    description: description.notRequired().default(undefined),
    helpText: helpText.notRequired().default(undefined),
});
export const outputTranslationsCreate = yup.array().of(outputTranslationCreate.required(requiredErrorMessage))
export const outputTranslationsUpdate = yup.array().of(outputTranslationUpdate.required(requiredErrorMessage))
export const outputCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    name: name.notRequired().default(undefined),
    standardConnect: id.notRequired().default(undefined),
    standardCreate: standardCreate.notRequired().default(undefined),
    translationsCreate: outputTranslationsCreate.notRequired().default(undefined),
}, [['standardConnect', 'standardCreate']]) // Can only pick one standard
export const outputUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    name: name.notRequired().default(undefined),
    // There is purposely no option to delete or update a standard from here
    standardConnect: id.notRequired().default(undefined),
    standardDisconnect: id.notRequired().default(undefined),
    standardCreate: standardCreate.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
    translationsCreate: outputTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: outputTranslationsUpdate.notRequired().default(undefined),
})
export const outputsCreate = yup.array().of(outputCreate.required(requiredErrorMessage))
export const outputsUpdate = yup.array().of(outputUpdate.required(requiredErrorMessage))

export const routineTranslationCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.required(requiredErrorMessage),
    description: description.notRequired().default(undefined),
    instructions: instructions.required(requiredErrorMessage),
    name: name.required(requiredErrorMessage),
});
export const routineTranslationUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.notRequired().default(undefined),
    description: description.notRequired().default(undefined),
    instructions: instructions.notRequired().default(undefined),
    name: name.notRequired().default(undefined),
});
export const routineTranslationsCreate = yup.array().of(routineTranslationCreate.required(requiredErrorMessage))
export const routineTranslationsUpdate = yup.array().of(routineTranslationUpdate.required(requiredErrorMessage))

/**
 * Information required when creating a routine. 
 */
export const routineCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    isAutomatable: isAutomatable.nullable().notRequired().default(undefined),
    isComplete: isComplete.notRequired().default(undefined),
    isInternal: isInternal.nullable().notRequired().default(undefined),
    isPrivate: isPrivate.notRequired().default(false),
    version: version().nullable().notRequired().default(undefined),
    parentId: id.notRequired().default(undefined), // If forked, the parent's id
    createdByUserId: id.notRequired().default(undefined), // If associating with yourself, your own id. Cannot associate with another user
    createdByOrganizationId: id.notRequired().default(undefined), // If associating with an organization you are an admin of, the organization's id
    nodesCreate: nodesCreate.notRequired().default(undefined),
    nodeLinksCreate: nodeLinksCreate.notRequired().default(undefined),
    inputsCreate: inputsCreate.notRequired().default(undefined),
    outputsCreate: outputsCreate.notRequired().default(undefined),
    resourceListsCreate: resourceListsCreate.notRequired().default(undefined),
    tagsConnect: tagArray.notRequired().default(undefined),
    tagsCreate: tagsCreate.notRequired().default(undefined),
    translationsCreate: routineTranslationsCreate.notRequired().default(undefined),
}, [['createdByUserId', 'createdByOrganizationId']]) // Makes sure you can't associate with both a user and an organization


type RoutineUpdateValidationProps = {
    minVersion?: string,
}

/**
 * Information required when updating a routine
 */
export const routineUpdate = ({ minVersion }: RoutineUpdateValidationProps) => yup.object().shape({
    id: id.required(requiredErrorMessage),
    isAutomatable: isAutomatable.nullable().notRequired().default(undefined),
    isComplete: isComplete.notRequired().default(undefined),
    isInternal: isInternal.nullable().notRequired().default(undefined),
    isPrivate: isPrivate.notRequired().default(false),
    version: version(minVersion).nullable().notRequired().default(undefined),
    parentId: id.notRequired().default(undefined), // If forked, the parent's id
    userId: id.notRequired().default(undefined), // If associating with yourself, your own id. Cannot associate with another user
    organizationId: id.notRequired().default(undefined), // If associating with an organization you are an admin of, the organization's id
    nodesDelete: idArray.notRequired().default(undefined),
    nodesCreate: nodesCreate.notRequired().default(undefined),
    nodesUpdate: nodesUpdate.notRequired().default(undefined),
    nodeLinksDelete: idArray.notRequired().default(undefined),
    nodeLinksCreate: nodeLinksCreate.notRequired().default(undefined),
    nodeLinksUpdate: nodeLinksUpdate.notRequired().default(undefined),
    inputsDelete: idArray.notRequired().default(undefined),
    inputsCreate: inputsCreate.notRequired().default(undefined),
    inputsUpdate: inputsUpdate.notRequired().default(undefined),
    outputsDelete: idArray.notRequired().default(undefined),
    outputsCreate: outputsCreate.notRequired().default(undefined),
    outputsUpdate: outputsUpdate.notRequired().default(undefined),
    resourceListsDelete: idArray.notRequired().default(undefined),
    resourceListsCreate: resourceListsCreate.notRequired().default(undefined),
    resourceListsUpdate: resourceListsUpdate.notRequired().default(undefined),
    tagsConnect: tagArray.notRequired().default(undefined),
    tagsDisconnect: tagArray.notRequired().default(undefined),
    tagsCreate: tagsCreate.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
    translationsCreate: routineTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: routineTranslationsUpdate.notRequired().default(undefined),
}, [['userId', 'organizationId']]) // Makes sure you can't transfer to both a user and an organization

export const routinesCreate = yup.array().of(routineCreate.required(requiredErrorMessage))
export const routinesUpdate = yup.array().of(routineUpdate({}).required(requiredErrorMessage))