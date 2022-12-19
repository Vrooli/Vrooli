/**
 * Shared fields are defined to reduce bugs that may occur when 
 * there is a mismatch between the database and schemas. Every database 
 * field with a duplicate name has the name format, so as long as 
 * that format matches the fields below, there should be no errors.
 */
import { ReportFor } from '@shared/consts';
import * as yup from 'yup';
import { blankToUndefined, reqArr } from './builders';
import { maxStrErr, minNumErr, minStrErr, passwordErr } from './errors';
import { passwordRegex, urlRegex } from './regex';
import { minVersionTest } from './versions';

export const id = yup.string().transform(blankToUndefined).max(256, maxStrErr)
export const bio = yup.string().transform(blankToUndefined).max(2048, maxStrErr);
export const email = yup.string().transform(blankToUndefined).email().max(256, maxStrErr);
export const description = yup.string().transform(blankToUndefined).max(2048, maxStrErr)
export const helpText = yup.string().transform(blankToUndefined).max(2048, maxStrErr)
export const language = yup.string().transform(blankToUndefined).min(2, minStrErr).max(3, maxStrErr) // Language code
export const name = yup.string().transform(blankToUndefined).min(3, minStrErr).max(128, maxStrErr)
export const handle = yup.string().transform(blankToUndefined).min(3, minStrErr).max(16, maxStrErr).nullable() // ADA Handle
export const tag = yup.string().transform(blankToUndefined).min(2, minStrErr).max(64, maxStrErr)
export const versionLabel = (minVersion: string = '0.0.1') => yup.string().transform(blankToUndefined).max(16, maxStrErr).test(...minVersionTest(minVersion))
export const versionNotes = yup.string().transform(blankToUndefined).max(4092, maxStrErr)
export const idArray = reqArr(id)
export const tagArray = reqArr(tag)
export const nodeCondition = yup.string().transform(blankToUndefined).max(8192, maxStrErr)
export const nodeOperation = yup.string().transform(blankToUndefined).max(512, maxStrErr)
export const permissions = yup.string().transform(blankToUndefined).max(8192, maxStrErr)
export const hexColor = yup.string().transform(blankToUndefined).max(16, maxStrErr)
export const timeZone = yup.string().transform(blankToUndefined).max(64, maxStrErr)
export const eventStart = yup.string().transform(blankToUndefined).max(64, maxStrErr)
export const eventEnd = yup.string().transform(blankToUndefined).max(64, maxStrErr)
export const recurrStart = yup.string().transform(blankToUndefined).max(64, maxStrErr)
export const recurrEnd = yup.string().transform(blankToUndefined).max(64, maxStrErr)
export const message = yup.string().transform(blankToUndefined).max(4096, maxStrErr)
export const theme = yup.string().transform(blankToUndefined).max(128, maxStrErr);
export const password = yup.string().min(8).max(50).matches(passwordRegex, passwordErr);
export const url = yup.string().transform(blankToUndefined).max(1024, maxStrErr).test(
    'link',
    'Must be a URL',
    (value: string | undefined) => {
        return value !== undefined ? urlRegex.test(value) : true
    }
)
export const details = yup.string().transform(blankToUndefined).max(8192, maxStrErr)
export const summary = yup.string().transform(blankToUndefined).max(1024, maxStrErr)
export const reportCreatedFor = yup.string().transform(blankToUndefined).oneOf(Object.values(ReportFor))
export const reportReason = yup.string().transform(blankToUndefined).min(1, minStrErr).max(128, maxStrErr)
export const apiCallData = yup.string().transform(blankToUndefined).max(8192, maxStrErr)
export const smartContractCallData = yup.string().transform(blankToUndefined).max(8192, maxStrErr)
export const instructions = yup.string().transform(blankToUndefined).max(8192, maxStrErr)
export const index = yup.number().integer().min(0, minNumErr).nullable();