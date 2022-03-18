import { description, id, idArray, language, title } from './base';
import * as yup from 'yup';
import { ResourceUsedFor } from '../consts';

const index = yup.number().integer().min(0)
const link = yup.string().max(1024)
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