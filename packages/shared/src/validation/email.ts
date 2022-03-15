import { id } from './base';
import * as yup from 'yup';

const emailAddress = yup.string().email()
const receivesAccountUpdates = yup.boolean()
const receivesBusinessUpdates = yup.boolean()

/**
 * Information required when creating a comment
 */
export const emailCreate = yup.object().shape({
    emailAddress: emailAddress.required(),
    receivesAccountUpdates: receivesAccountUpdates.notRequired().default(undefined),
    receivesBusinessUpdates: receivesBusinessUpdates.notRequired().default(undefined),
})

/**
 * Information required when updating an organization
 */
export const emailUpdate = yup.object().shape({
    id: id.required(),
    receivesAccountUpdates: receivesAccountUpdates.notRequired().default(undefined),
    receivesBusinessUpdates: receivesBusinessUpdates.notRequired().default(undefined),
})

export const emailsCreate = yup.array().of(emailCreate.required())
export const emailsUpdate = yup.array().of(emailUpdate.required())