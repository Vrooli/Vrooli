import { description, id, idArray, language, title } from './base';
import * as yup from 'yup';
import { ResourceUsedFor } from '../consts';

// Regular expressions for validating links
// URL
export const urlRegex = /^((ftp|http|https):\/\/)?(www.)?(?!.*(ftp|http|https|www.))[a-zA-Z0-9_-]+(\.[a-zA-Z]+)+((\/)[\w#]+)*(\/\w+\?[a-zA-Z0-9_]+=\w+(&[a-zA-Z0-9_]+=\w+)*)?$/
// Cardano payment wallet address (i.e. starts with "addr1", and is 103 characters long in total)
export const walletAddressRegex = /^addr1[a-zA-Z0-9]{98}$/
// ADA Handle (i.e. starts with "$", and is 3-16 characters (not including the "$"))
export const adaHandleRegex = /^\$[a-zA-Z0-9]{3,16}$/

const index = yup.number().integer().min(0)
// Link must match one of the regex above
const link = yup.string().max(1024).test(
    'link',
    'Must be a URL, Cardano payment address, or ADA Handle',
    (value: string | undefined) => {
        return value !== undefined && (urlRegex.test(value) || walletAddressRegex.test(value) || adaHandleRegex.test(value))
    }
)
const usedFor = yup.string().oneOf(Object.values(ResourceUsedFor))

export const resourceTranslationCreate = yup.object().shape({
    language: language.required(),
    description: description.notRequired().default(undefined),
    title: title.notRequired().default(undefined),
});
export const resourceTranslationUpdate = yup.object().shape({
    id: id.required(),
    language: language.notRequired().default(undefined),
    description: description.notRequired().default(undefined),
    title: title.notRequired().default(undefined),
});
export const resourceTranslationsCreate = yup.array().of(resourceTranslationCreate.required())
export const resourceTranslationsUpdate = yup.array().of(resourceTranslationUpdate.required())

export const resourceCreateForm = yup.object().shape({
    link: link.required(),
    description: description.notRequired().default(undefined),
    title: title.notRequired().default(undefined),
    usedFor: usedFor.notRequired().default(undefined),
})
export const resourceUpdateForm = resourceCreateForm;

export const resourceCreate = yup.object().shape({
    listId: id.required(),
    index: index.notRequired().default(undefined),
    link: link.required(),
    usedFor: usedFor.notRequired().default(undefined),
    translations: resourceTranslationsCreate,
})

export const resourceUpdate = yup.object().shape({
    id: id.required(),
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
export const resourcesCreate = yup.array().of(resourceCreate.omit(['createdFor', 'createdForId']).required())
export const resourcesUpdate = yup.array().of(resourceUpdate.omit(['createdFor', 'createdForId']).required())