import { blankToUndefined, id, language, maxStrErr, minStrErr, opt, req } from '../utils';
import * as yup from 'yup';
import { ReportFor } from '@shared/consts';

const createdFor = yup.string().transform(blankToUndefined).oneOf(Object.values(ReportFor))
const details = yup.string().transform(blankToUndefined).max(1024, maxStrErr)
const reason = yup.string().transform(blankToUndefined).min(1, minStrErr).max(128, maxStrErr)

/**
 * Information required when creating a comment
 */
export const reportCreate = yup.object().shape({
    id: req(id),
    createdFor: req(createdFor),
    createdForId: req(id),
    details: opt(details),
    language: req(language),
    reason: req(reason),
})

export const reportCreateForm = yup.object().shape({
    createdFor: req(createdFor),
    createdForId: req(id),
    details: opt(details),
    language: req(language),
    reason: req(reason),
    otherReason: opt(reason),
})

/**
 * Information required when updating an organization
 */
export const reportUpdate = yup.object().shape({
    id: req(id),
    details: opt(details),
    language: opt(language),
    reason: opt(reason),
})