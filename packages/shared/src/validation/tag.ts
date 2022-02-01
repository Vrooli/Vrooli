import { description, id } from './';
import * as yup from 'yup';

const anonymous = yup.boolean().optional(); // Determines if the user will be credited for the tag
const tag = yup.string().max(128).optional();

/**
 * Information required when creating a tag
 */
export const tagAdd = yup.object().shape({
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

export const tagsAdd = yup.array().of(tagAdd.required()).optional();
export const tagsUpdate = yup.array().of(tagUpdate.required()).optional();
