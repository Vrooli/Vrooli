import { description, idArray, id, name, version } from './base';
import { tagsAdd, tagsUpdate } from './tag';
import * as yup from 'yup';
import { StandardType } from '../consts';

const standardDefault = yup.string().max(1024).optional();
const schema = yup.string().max(8192).required();
const type = yup.string().oneOf(Object.values(StandardType)).optional();

/**
 * Information required when creating a standard. 
 */
export const standardAdd = yup.object().shape({
    default: standardDefault,
    description,
    isFile: yup.boolean().optional(),
    name: name.required(),
    schema,
    type,
    version,
    createdByUserId: id, // If associating with yourself, your own id. Cannot associate with another user
    createdByOrganizationId: id, // If associating with an organization you are an admin of, the organization's id
    tagsConnect: idArray,
    tagsAdd,
})

/**
 * Information required when updating a routine
 */
export const standardUpdate = yup.object().shape({
    default: standardDefault,
    description,
    isFile: yup.boolean().optional(),
    makingAnonymous: yup.boolean().optional(), // If you want the standard to be made anonymous
    name: name.required(),
    schema,
    type,
    version,
    tagsConnect: idArray,
    tagsDisconnect: idArray,
    tagsDelete: idArray,
    tagsAdd,
    tagsUpdate,
})

export const standardsAdd = yup.array().of(standardAdd.required()).optional();
export const standardsUpdate = yup.array().of(standardUpdate.required()).optional();