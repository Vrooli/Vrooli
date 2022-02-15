import { id } from './base';
import * as yup from 'yup';
import { ReportFor } from '../consts';

const createdFor = yup.string().oneOf(Object.values(ReportFor)).optional();
const details = yup.string().max(1024).optional();
const reason = yup.string().min(1).max(128).optional();

/**
 * Information required when creating a comment
 */
export const reportCreate = yup.object().shape({
    createdFor: createdFor.required(),
    createdForId: id.required(),
    details,
    reason: reason.required(),
})

/**
 * Information required when updating an organization
 */
export const reportUpdate = yup.object().shape({
    id: id.required(),
    details,
    reason,
})