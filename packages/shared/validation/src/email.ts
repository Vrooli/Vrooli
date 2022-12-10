import { blankToUndefined, requiredErrorMessage } from './base';
import * as yup from 'yup';

const emailAddress = yup.string().transform(blankToUndefined).email()

export const emailCreate = yup.object().shape({
    emailAddress: emailAddress.required(requiredErrorMessage),
})

export const emailCreateButton = yup.object().shape({
    emailAddress: emailAddress.required(requiredErrorMessage),
})

export const emailsCreate = yup.array().of(emailCreate.required(requiredErrorMessage))