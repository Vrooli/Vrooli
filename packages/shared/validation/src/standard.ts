import { description, idArray, id, name, version, language, tagArray, maxStringErrorMessage, requiredErrorMessage } from './base';
import { tagsCreate } from './tag';
import * as yup from 'yup';
import { InputType } from '@shared/consts';
import { resourceListsCreate, resourceListsUpdate } from './resourceList';

const standardDefault = yup.string().max(8192, maxStringErrorMessage);
const stringifiedJson = yup.string().max(8192, maxStringErrorMessage);
const type = yup.string().oneOf(Object.values(InputType))
const isPrivate = yup.boolean();

export const standardTranslationCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.required(requiredErrorMessage),
    description: description.notRequired().default(undefined),
    jsonVariable: stringifiedJson.nullable().notRequired().default(undefined),
});
export const standardTranslationUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.notRequired().default(undefined),
    description: description.notRequired().default(undefined),
    jsonVariable: stringifiedJson.nullable().notRequired().default(undefined),
});
export const standardTranslationsCreate = yup.array().of(standardTranslationCreate.required(requiredErrorMessage))
export const standardTranslationsUpdate = yup.array().of(standardTranslationUpdate.required(requiredErrorMessage))

/**
 * Information required when creating a standard. 
 */
export const standardCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    default: standardDefault.notRequired().default(undefined),
    isInternal: yup.boolean().notRequired().default(undefined),
    isPrivate: isPrivate.notRequired().default(undefined),
    name: name.notRequired().default(undefined),
    type: type.required(requiredErrorMessage),
    props: stringifiedJson.required(requiredErrorMessage),
    yup: stringifiedJson.notRequired().default(undefined),
    version: version().notRequired().default(undefined),
    createdByUserId: id.notRequired().default(undefined), // If associating with yourself, your own id. Cannot associate with another user
    createdByOrganizationId: id.notRequired().default(undefined), // If associating with an organization you are an admin of, the organization's id
    resourceListsCreate: resourceListsCreate.notRequired().default(undefined),
    tagsConnect: tagArray.notRequired().default(undefined),
    tagsCreate: tagsCreate.notRequired().default(undefined),
    translationsCreate: standardTranslationsCreate.notRequired().default(undefined),
})

/**
 * Information required when updating a routine
 */
export const standardUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    isPrivate: isPrivate.notRequired().default(undefined),
    makingAnonymous: yup.boolean().notRequired().default(undefined), // If you want the standard to be made anonymous
    resourceListsDelete: idArray.notRequired().default(undefined),
    resourceListsCreate: resourceListsCreate.notRequired().default(undefined),
    resourceListsUpdate: resourceListsUpdate.notRequired().default(undefined),
    tagsConnect: tagArray.notRequired().default(undefined),
    tagsDisconnect: tagArray.notRequired().default(undefined),
    tagsCreate: tagsCreate.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
    translationsCreate: standardTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: standardTranslationsUpdate.notRequired().default(undefined),
})

export const standardsCreate = yup.array().of(standardCreate.required(requiredErrorMessage))
export const standardsUpdate = yup.array().of(standardUpdate.required(requiredErrorMessage))