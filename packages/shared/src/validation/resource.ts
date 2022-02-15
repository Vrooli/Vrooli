import { description, id, title } from './base';
import * as yup from 'yup';
import { ResourceFor, ResourceUsedFor } from '../consts';

const createdFor = yup.string().oneOf(Object.values(ResourceFor)).optional();
const link = yup.string().max(1024).optional();
const usedFor = yup.string().oneOf(Object.values(ResourceUsedFor)).optional();

export const resourceCreate = yup.object().shape({
    createdFor: createdFor.required(),
    createdForId: id.required(),
    description,
    link: link.required(),
    title,
    usedFor,
})

export const resourceUpdate = yup.object().shape({
    id: id.required(),
    createdFor,
    createdForId: id,
    description,
    link,
    title,
    usedFor,
})

// Resources created/updated through relationships don't need createdFor and createdForId,
// as the relationship handles that
export const resourcesCreate = yup.array().of(resourceCreate.omit(['createdFor', 'createdForId']).required()).optional();
export const resourcesUpdate = yup.array().of(resourceUpdate.omit(['createdFor', 'createdForId']).required()).optional();