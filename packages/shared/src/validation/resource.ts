import { description, id, title } from './';
import * as yup from 'yup';
import { ResourceFor } from '../consts';

const link = yup.string().max(1024).optional();
const usedFor = yup.string().oneOf(Object.values(ResourceFor)).optional();

export const resourceAdd = yup.object().shape({
    description,
    link: link.required(),
    title,
    usedFor,
})

export const resourceUpdate = yup.object().shape({
    id: id.required(),
    description,
    link,
    title,
    usedFor,
})

export const resourcesAdd = yup.array().of(resourceAdd.required()).optional();
export const resourcesUpdate = yup.array().of(resourceUpdate.required()).optional();