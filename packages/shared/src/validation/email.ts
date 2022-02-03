import { id } from './base';
import * as yup from 'yup';

const emailAddress = yup.string().email().optional();
const receivesAccountUpdates = yup.boolean().optional();
const receivesBusinessUpdates = yup.boolean().optional();

/**
 * Information required when creating a comment
 */
export const emailAdd = yup.object().shape({
    emailAddress: emailAddress.required(),
    receivesAccountUpdates,
    receivesBusinessUpdates,
})

/**
 * Information required when updating an organization
 */
export const emailUpdate = yup.object().shape({
    id: id.required(),
    receivesAccountUpdates,
    receivesBusinessUpdates,
})

export const emailsAdd = yup.array().of(emailAdd.required()).optional();
export const emailsUpdate = yup.array().of(emailUpdate.required()).optional();