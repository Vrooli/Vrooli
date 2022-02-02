import { description, id, title } from './base';
import * as yup from 'yup';
import { ResourceFor, ResourceUsedFor } from '../consts';

const createdFor = yup.string().oneOf(Object.values(ResourceFor)).optional();
const link = yup.string().max(1024).optional();
const usedFor = yup.string().oneOf(Object.values(ResourceUsedFor)).optional();

export const resourceAdd = yup.object().shape({
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

export const resourcesAdd = yup.array().of(resourceAdd.required()).optional();
export const resourcesUpdate = yup.array().of(resourceUpdate.required()).optional();