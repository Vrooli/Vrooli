import { description, id, idArray, language, title } from './base';
import * as yup from 'yup';
import { ResourceFor, ResourceUsedFor } from '../consts';

const createdFor = yup.string().oneOf(Object.values(ResourceFor)).optional();
const index = yup.number().integer().min(0).optional();
const link = yup.string().max(1024).optional();
const usedFor = yup.string().oneOf(Object.values(ResourceUsedFor)).optional();

export const resourceTranslationCreate = yup.object().shape({
    language,
    description,
    title,
});
export const resourceTranslationUpdate = yup.object().shape({
    id: id.required(),
    language,
    description,
    title,
});
export const resourceTranslationsCreate = yup.array().of(resourceTranslationCreate.required()).optional();
export const resourceTranslationsUpdate = yup.array().of(resourceTranslationUpdate.required()).optional();

export const resourceCreate = yup.object().shape({
    createdFor: createdFor.required(),
    createdForId: id.required(),
    index,
    link: link.required(),
    usedFor,
    translations: resourceTranslationsCreate,
})

export const resourceUpdate = yup.object().shape({
    id: id.required(),
    createdFor,
    createdForId: id,
    index,
    link,
    usedFor,
    translationsDelete: idArray,
    translationsCreate: resourceTranslationsCreate,
    translationsUpdate: resourceTranslationsUpdate,
})

// Resources created/updated through relationships don't need createdFor and createdForId,
// as the relationship handles that
export const resourcesCreate = yup.array().of(resourceCreate.omit(['createdFor', 'createdForId']).required()).optional();
export const resourcesUpdate = yup.array().of(resourceUpdate.omit(['createdFor', 'createdForId']).required()).optional();