import { description, idArray, id, name, language, tagArray, maxStrErr, blankToUndefined, req, opt } from '../utils';
import * as yup from 'yup';
import { InputType } from '@shared/consts';

const standardDefault = yup.string().transform(blankToUndefined).max(8192, maxStrErr);
const stringifiedJson = yup.string().transform(blankToUndefined).max(8192, maxStrErr);
const type = yup.string().transform(blankToUndefined).oneOf(Object.values(InputType))
const isPrivate = yup.boolean();

export const standardTranslationCreate = yup.object().shape({
    id: req(id),
    language: req(language),
    description: req(description),
    jsonVariable: opt(stringifiedJson.nullable()),
});
export const standardTranslationUpdate = yup.object().shape({
    id: req(id),
    language: opt(language),
    description: req(description),
    jsonVariable: opt(stringifiedJson.nullable()),
});

/**
 * Information required when creating a standard. 
 */
export const standardCreate = yup.object().shape({
    id: req(id),
    default: opt(standardDefault),
    isInternal: opt(yup.boolean()),
    isPrivate: opt(isPrivate),
    name: opt(name),
    type: req(type),
    props: req(stringifiedJson),
    yup: opt(stringifiedJson),
    // version: opt(version()),
    createdByUserId: opt(id), // If associating with yourself, your own id. Cannot associate with another user
    createdByOrganizationId: opt(id), // If associating with an organization you are an admin of, the organization's id
    // resourceListsCreate: opt(resourceListsCreate),
    tagsConnect: opt(tagArray),
    // tagsCreate: opt(tagsCreate),
    // translationsCreate: opt(standardTranslationsCreate),
})

/**
 * Information required when updating a routine
 */
export const standardUpdate = yup.object().shape({
    id: req(id),
    isPrivate: opt(isPrivate),
    makingAnonymous: opt(yup.boolean()), // If you want the standard to be made anonymous
    resourceListsDelete: opt(idArray),
    // resourceListsCreate: opt(resourceListsCreate),
    // resourceListsUpdate: opt(resourceListsUpdate),
    tagsConnect: opt(tagArray),
    tagsDisconnect: opt(tagArray),
    // tagsCreate: opt(tagsCreate),
    translationsDelete: opt(idArray),
    // translationsCreate: opt(standardTranslationsCreate),
    // translationsUpdate: opt(standardTranslationsUpdate),
})