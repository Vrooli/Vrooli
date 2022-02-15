import { description, id } from './base';
import * as yup from 'yup';

const anonymous = yup.boolean().optional(); // Determines if the user will be credited for the tag
const tag = yup.string().min(2).max(128).optional();

/**
 * Information required when creating a tag
 */
export const tagCreate = yup.object().shape({
    anonymous,
    description,
    tag: tag.required(),
})

/**
 * Information required when updating a tag
 */
export const tagUpdate = yup.object().shape({
    id: id.required(),
    anonymous,
    description,
    tag,
})

export const tagsCreate = yup.array().of(tagCreate.required()).optional();
export const tagsUpdate = yup.array().of(tagUpdate.required()).optional();
