import { id } from './base';
import * as yup from 'yup';

const createdFor = yup.string().oneOf(['Project', 'Routine', 'Standard']).optional();
const text = yup.string().min(1).max(8192).optional();

/**
 * Information required when creating a comment
 */
export const commentCreate = yup.object().shape({
    text: text.required(),
    createdFor: createdFor.required(),
    forId: id.required(),
})

/**
 * Information required when updating an organization
 */
export const commentUpdate = yup.object().shape({
    id: id.required(),
    text: text,
})