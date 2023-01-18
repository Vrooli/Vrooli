/**
 * Shared fields are defined to reduce bugs that may occur when 
 * there is a mismatch between the database and schemas. Every database 
 * field with a duplicate name has the name format, so as long as 
 * that format matches the fields below, there should be no errors.
 */
import { ReportFor } from '@shared/consts';
import * as yup from 'yup';
import { blankToUndefined } from './builders';
import { maxNumErr, maxStrErr, minNumErr, minStrErr, passwordErr, reqErr } from './errors';
import { adaHandleRegex, passwordRegex, urlRegex } from './regex';
import { minVersionTest } from './versions';

// db fields
export const id = yup.string().transform(blankToUndefined).max(256, maxStrErr)

// protocol fields
export const apiCallData = yup.string().transform(blankToUndefined).max(8192, maxStrErr)
export const email = yup.string().transform(blankToUndefined).email().max(256, maxStrErr);
export const handle = yup.string().transform(blankToUndefined).min(3, minStrErr).max(16, maxStrErr).test(
    'handle',
    'Must start with a \'$\', be from 3-16 characters, and only contain letters and numbers',
    (value: string | undefined) => {
        return value !== undefined ? adaHandleRegex.test(value) : true
    }
)
export const hexColor = yup.string().transform(blankToUndefined).max(16, maxStrErr)
export const pushNotificationKeys = yup.array().of(yup.string().transform(blankToUndefined).max(256, maxStrErr))
export const smartContractCallData = yup.string().transform(blankToUndefined).max(8192, maxStrErr)
export const url = yup.string().transform(blankToUndefined).max(1024, maxStrErr).test(
    'link',
    'Must be a URL',
    (value: string | undefined) => {
        return value !== undefined ? urlRegex.test(value) : true
    }
)

// numbers
const MAX_INT = 2 ** 32 - 1;
const MAX_DOUBLE = 2 ** 32 - 1;
const MIN_INT = -(2 ** 32);
const MIN_DOUBLE = -(2 ** 32);
export const doublePositiveOrZero = yup.number().min(0, minNumErr).max(MAX_DOUBLE, maxNumErr);
export const intPositiveOrZero = yup.number().min(0, minNumErr).max(MAX_INT, maxNumErr).integer();
export const intPositiveOrOne = yup.number().min(1, minNumErr).max(MAX_INT, maxNumErr).integer();
export const double = yup.number().min(MIN_DOUBLE, minNumErr).max(MAX_DOUBLE, maxNumErr);
export const int = yup.number().min(MIN_INT, minNumErr).max(MAX_INT, maxNumErr).integer();
export const index = intPositiveOrZero;

// dates
export const timeZone = yup.string().transform(blankToUndefined).max(64, maxStrErr)
export const eventStart = yup.date().min(new Date(), 'Start date must be in the future');
export const eventEnd = yup.date().min(yup.ref('eventStart'), 'End date must be after start date');
export const recurrStart = yup.date().min(new Date(), 'Start date must be in the future');
export const recurrEnd = yup.date().min(yup.ref('recurrStart'), 'End date must be after start date');

// strings
export const bio = yup.string().transform(blankToUndefined).max(2048, maxStrErr);
export const description = yup.string().transform(blankToUndefined).max(2048, maxStrErr)
export const helpText = yup.string().transform(blankToUndefined).max(2048, maxStrErr)
export const language = yup.string().transform(blankToUndefined).min(2, minStrErr).max(3, maxStrErr) // Language code
export const name = yup.string().transform(blankToUndefined).min(3, minStrErr).max(128, maxStrErr)
export const tag = yup.string().transform(blankToUndefined).min(2, minStrErr).max(64, maxStrErr)
export const versionLabel = (minVersion: string = '0.0.1') => yup.string().transform(blankToUndefined).max(16, maxStrErr).test(...minVersionTest(minVersion))
export const versionNotes = yup.string().transform(blankToUndefined).max(4092, maxStrErr)
export const idArray = yup.array().of(id.required(reqErr));
export const tagArray = yup.array().of(tag.required(reqErr));
export const nodeCondition = yup.string().transform(blankToUndefined).max(8192, maxStrErr)
export const nodeOperation = yup.string().transform(blankToUndefined).max(512, maxStrErr)
export const permissions = yup.string().transform(blankToUndefined).max(8192, maxStrErr)
export const response = yup.string().transform(blankToUndefined).max(8192, maxStrErr)
export const message = yup.string().transform(blankToUndefined).max(4096, maxStrErr)
export const theme = yup.string().transform(blankToUndefined).max(128, maxStrErr);
export const password = yup.string().min(8).max(50).matches(passwordRegex, passwordErr);
export const details = yup.string().transform(blankToUndefined).max(8192, maxStrErr)
export const summary = yup.string().transform(blankToUndefined).max(1024, maxStrErr)
export const reportReason = yup.string().transform(blankToUndefined).min(1, minStrErr).max(128, maxStrErr)
export const instructions = yup.string().transform(blankToUndefined).max(8192, maxStrErr)
export const jsonVariable = yup.string().transform(blankToUndefined).max(8192, maxStrErr)

// enums
export const reportCreatedFor = yup.string().transform(blankToUndefined).oneOf(Object.values(ReportFor))