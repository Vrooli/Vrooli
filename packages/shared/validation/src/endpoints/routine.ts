import { description, idArray, id, name, language, tagArray, helpText, maxStrErr, blankToUndefined, req, opt, YupModel, permissions, rel } from '../utils';
import * as yup from 'yup';
import { labelValidation } from './label';
import { tagValidation } from './tag';
import { routineVersionValidation } from './routineVersion';

const isAutomatable = yup.boolean()
const isComplete = yup.boolean()
const isInternal = yup.boolean()
const isPrivate = yup.boolean()
const isRequired = yup.boolean()
const instructions = yup.string().transform(blankToUndefined).max(8192, maxStrErr)

export const routineValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        isInternal: opt(isInternal),
        isPrivate: opt(isPrivate),
        permissions: req(permissions),
        ...rel('user', ['Connect'], 'one', 'opt'),
        ...rel('organization', ['Connect'], 'one', 'opt'),
        ...rel('labels', ['Connect', 'Create'], 'many', 'opt', labelValidation),
        ...rel('tags', ['Connect', 'Create'], 'many', 'opt', tagValidation),
        ...rel('versions', ['Create'], 'many', 'req', routineVersionValidation),
    }),
    update: () => yup.object().shape({
        id: req(id),
        isInternal: opt(isInternal),
        isPrivate: opt(isPrivate),
        permissions: opt(permissions),
        ...rel('user', ['Connect'], 'one', 'opt'),
        ...rel('organization', ['Connect'], 'one', 'opt'),
        ...rel('labels', ['Connect', 'Create', 'Disconnect'], 'many', 'opt', labelValidation),
        ...rel('tags', ['Connect', 'Create', 'Disconnect'], 'many', 'opt', tagValidation),
        ...rel('versions', ['Create', 'Update', 'Delete'], 'many', 'req', routineVersionValidation),
    }),
}

export const inputTranslationCreate = yup.object().shape({
    id: req(id),
    language: req(language),
    description: req(description),
    helpText: opt(helpText),
});
export const inputTranslationUpdate = yup.object().shape({
    id: req(id),
    language: opt(language),
    description: req(description),
    helpText: opt(helpText),
});
export const inputCreate = yup.object().shape({
    id: req(id),
    isRequired: opt(isRequired),
    name: req(name),
    standardConnect: opt(id),
    // standardCreate: opt(standardCreate),
}, [['standardConnect', 'standardCreate']]) // Can only pick one standard
export const inputUpdate = yup.object().shape({
    id: req(id),
    isRequired: opt(isRequired),
    name: opt(name),
    // There is purposely no option to delete or update a standard from here
    standardConnect: opt(id),
    standardDisconnect: opt(id),
    // standardCreate: opt(standardCreate),
})

export const outputTranslationCreate = yup.object().shape({
    id: req(id),
    language: req(language),
    description: req(description),
    helpText: opt(helpText),
});
export const outputTranslationUpdate = yup.object().shape({
    id: req(id),
    language: opt(language),
    description: req(description),
    helpText: opt(helpText),
});
export const outputCreate = yup.object().shape({
    id: req(id),
    name: opt(name),
    standardConnect: opt(id),
    // standardCreate: opt(standardCreate),
    // translationsCreate: opt(outputTranslationsCreate),
}, [['standardConnect', 'standardCreate']]) // Can only pick one standard
export const outputUpdate = yup.object().shape({
    id: req(id),
    name: opt(name),
    // There is purposely no option to delete or update a standard from here
    standardConnect: opt(id),
    standardDisconnect: opt(id),
    // standardCreate: opt(standardCreate),
    translationsDelete: opt(idArray),
    // translationsCreate: opt(outputTranslationsCreate),
    // translationsUpdate: opt(outputTranslationsUpdate),
})

export const routineTranslationCreate = yup.object().shape({
    id: req(id),
    language: req(language),
    description: req(description),
    instructions: req(instructions),
    name: req(name),
});
export const routineTranslationUpdate = yup.object().shape({
    id: req(id),
    language: opt(language),
    description: req(description),
    instructions: opt(instructions),
    name: opt(name),
});

/**
 * Information required when creating a routine. 
 */
export const routineCreate = yup.object().shape({
    id: req(id),
    isAutomatable: opt(isAutomatable.nullable()),
    isComplete: opt(isComplete),
    isInternal: opt(isInternal.nullable()),
    isPrivate: isPrivate.notRequired().default(false),
    // version: opt(version().nullable()),
    parentId: opt(id), // If forked, the parent's id
    createdByUserId: opt(id), // If associating with yourself, your own id. Cannot associate with another user
    createdByOrganizationId: opt(id), // If associating with an organization you are an admin of, the organization's id
    // nodesCreate: opt(nodesCreate),
    // nodeLinksCreate: opt(nodeLinksCreate),
    // inputsCreate: opt(inputsCreate),
    // outputsCreate: opt(outputsCreate),
    // resourceListsCreate: opt(resourceListsCreate),
    tagsConnect: opt(tagArray),
    // tagsCreate: opt(tagsCreate),
    // translationsCreate: opt(routineTranslationsCreate),
}, [['createdByUserId', 'createdByOrganizationId']]) // Makes sure you can't associate with both a user and an organization


type RoutineUpdateValidationProps = {
    minVersion?: string,
}

/**
 * Information required when updating a routine
 */
export const routineUpdate = ({ minVersion }: RoutineUpdateValidationProps) => yup.object().shape({
    id: req(id),
    isAutomatable: opt(isAutomatable.nullable()),
    isComplete: opt(isComplete),
    isInternal: opt(isInternal.nullable()),
    isPrivate: isPrivate.notRequired().default(false),
    // version: opt(version(minVersion).nullable()),
    parentId: opt(id), // If forked, the parent's id
    userId: opt(id), // If associating with yourself, your own id. Cannot associate with another user
    organizationId: opt(id), // If associating with an organization you are an admin of, the organization's id
    nodesDelete: opt(idArray),
    // nodesCreate: opt(nodesCreate),
    // nodesUpdate: opt(nodesUpdate),
    nodeLinksDelete: opt(idArray),
    // nodeLinksCreate: opt(nodeLinksCreate),
    // nodeLinksUpdate: opt(nodeLinksUpdate),
    inputsDelete: opt(idArray),
    // inputsCreate: opt(inputsCreate),
    // inputsUpdate: opt(inputsUpdate),
    // outputsDelete: opt(idArray),
    // outputsCreate: opt(outputsCreate),
    // outputsUpdate: opt(outputsUpdate),
    // resourceListsDelete: opt(idArray),
    // resourceListsCreate: opt(resourceListsCreate),
    // resourceListsUpdate: opt(resourceListsUpdate),
    // tagsConnect: opt(tagArray),
    // tagsDisconnect: opt(tagArray),
    // tagsCreate: opt(tagsCreate),
    // translationsDelete: opt(idArray),
    // translationsCreate: opt(routineTranslationsCreate),
    // translationsUpdate: opt(routineTranslationsUpdate),
}, [['userId', 'organizationId']]) // Makes sure you can't transfer to both a user and an organization