import { id, language } from './base';
import * as yup from 'yup';
import { ReportFor } from '@shared/consts';

const createdFor = yup.string().oneOf(Object.values(ReportFor))
const details = yup.string().max(1024)
const reason = yup.string().min(1).max(128)

/**
 * Information required when creating a comment
 */
export const reportCreate = yup.object().shape({
    id: id.required(),
    createdFor: createdFor.required(),
    createdForId: id.required(),
    details: details.notRequired().default(undefined),
    language: language.required(),
    reason: reason.required(),
})

export const reportCreateForm = yup.object().shape({
    createdFor: createdFor.required(),
    createdForId: id.required(),
    details: details.notRequired().default(undefined),
    language: language.required(),
    reason: reason.required(),
    otherReason: reason.notRequired().default(undefined),
})

/**
 * Information required when updating an organization
 */
export const reportUpdate = yup.object().shape({
    id: id.required(),
    details: details.notRequired().default(undefined),
    language: language.notRequired().default(undefined),
    reason: reason.notRequired().default(undefined),
})

export const reportsCreate = yup.array().of(reportCreate.required())
export const reportsUpdate = yup.array().of(reportUpdate.required())