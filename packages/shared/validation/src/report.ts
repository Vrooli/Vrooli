import { id, language, maxStringErrorMessage, minStringErrorMessage, requiredErrorMessage } from './base';
import * as yup from 'yup';
import { ReportFor } from '@shared/consts';

const createdFor = yup.string().oneOf(Object.values(ReportFor))
const details = yup.string().max(1024, maxStringErrorMessage)
const reason = yup.string().min(1, minStringErrorMessage).max(128, maxStringErrorMessage)

/**
 * Information required when creating a comment
 */
export const reportCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    createdFor: createdFor.required(requiredErrorMessage),
    createdForId: id.required(requiredErrorMessage),
    details: details.notRequired().default(undefined),
    language: language.required(requiredErrorMessage),
    reason: reason.required(requiredErrorMessage),
})

export const reportCreateForm = yup.object().shape({
    createdFor: createdFor.required(requiredErrorMessage),
    createdForId: id.required(requiredErrorMessage),
    details: details.notRequired().default(undefined),
    language: language.required(requiredErrorMessage),
    reason: reason.required(requiredErrorMessage),
    otherReason: reason.notRequired().default(undefined),
})

/**
 * Information required when updating an organization
 */
export const reportUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    details: details.notRequired().default(undefined),
    language: language.notRequired().default(undefined),
    reason: reason.notRequired().default(undefined),
})

export const reportsCreate = yup.array().of(reportCreate.required(requiredErrorMessage))
export const reportsUpdate = yup.array().of(reportUpdate.required(requiredErrorMessage))