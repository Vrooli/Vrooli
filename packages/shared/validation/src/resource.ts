import { blankToUndefined, description, id, idArray, language, maxStringErrorMessage, minNumberErrorMessage, requiredErrorMessage, name } from './base';
import * as yup from 'yup';
import { ResourceUsedFor } from '@shared/consts';

// Regular expressions for validating links
// URL (taken from here: https://stackoverflow.com/a/9284473/10240279)
export const urlRegex = /^(?:(?:https?|ftp):\/\/)(?:\S+(?::\S*)?@)?(?:(?!(?:10|127)(?:\.\d{1,3}){3})(?!(?:169\.254|192\.168)(?:\.\d{1,3}){2})(?!172\.(?:1[6-9]|2\d|3[0-1])(?:\.\d{1,3}){2})(?:[1-9]\d?|1\d\d|2[01]\d|22[0-3])(?:\.(?:1?\d{1,2}|2[0-4]\d|25[0-5])){2}(?:\.(?:[1-9]\d?|1\d\d|2[0-4]\d|25[0-4]))|(?:(?:[a-z\u00a1-\uffff0-9]-*)*[a-z\u00a1-\uffff0-9]+)(?:\.(?:[a-z\u00a1-\uffff0-9]-*)*[a-z\u00a1-\uffff0-9]+)*(?:\.(?:[a-z\u00a1-\uffff]{2,}))\.?)(?::\d{2,5})?(?:[/?#]\S*)?$/i
// Cardano payment wallet address (i.e. starts with "addr1", and is 103 characters long in total)
export const walletAddressRegex = /^addr1[a-zA-Z0-9]{98}$/
// ADA Handle (i.e. starts with "$", and is 3-16 characters (not including the "$"))
export const adaHandleRegex = /^\$[a-zA-Z0-9]{3,16}$/

const index = yup.number().integer().min(0, minNumberErrorMessage)
// Link must match one of the regex above
const link = yup.string().transform(blankToUndefined).max(1024, maxStringErrorMessage).test(
    'link',
    'Must be a URL, Cardano payment address, or ADA Handle',
    (value: string | undefined) => {
        return value !== undefined ? (urlRegex.test(value) || walletAddressRegex.test(value) || adaHandleRegex.test(value)) : true
    }
)
const usedFor = yup.string().transform(blankToUndefined).oneOf(Object.values(ResourceUsedFor))

export const resourceTranslationCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.required(requiredErrorMessage),
    description: description.notRequired().default(undefined),
    name: name.notRequired().default(undefined),
});
export const resourceTranslationUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.notRequired().default(undefined),
    description: description.notRequired().default(undefined),
    name: name.notRequired().default(undefined),
});
export const resourceTranslationsCreate = yup.array().of(resourceTranslationCreate.required(requiredErrorMessage))
export const resourceTranslationsUpdate = yup.array().of(resourceTranslationUpdate.required(requiredErrorMessage))

export const resourceCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    listId: id.required(requiredErrorMessage),
    index: index.notRequired().default(undefined),
    link: link.required(requiredErrorMessage),
    usedFor: usedFor.notRequired().default(undefined),
    translations: resourceTranslationsCreate,
})

export const resourceUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    listId: id.notRequired().default(undefined),
    index: index.notRequired().default(undefined),
    link: link.notRequired().default(undefined),
    usedFor: usedFor.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
    translationsCreate: resourceTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: resourceTranslationsUpdate.notRequired().default(undefined),
})

// Resources created/updated through relationships don't need createdFor and createdForId,
// as the relationship handles that
export const resourcesCreate = yup.array().of(resourceCreate.omit(['createdFor', 'createdForId']).required(requiredErrorMessage))
export const resourcesUpdate = yup.array().of(resourceUpdate.omit(['createdFor', 'createdForId']).required(requiredErrorMessage))