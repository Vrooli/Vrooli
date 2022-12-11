import { blankToUndefined, req, reqArr } from './base';
import * as yup from 'yup';

const emailAddress = yup.string().transform(blankToUndefined).email()

export const emailCreate = yup.object().shape({
    emailAddress: req(emailAddress),
})

export const emailCreateButton = yup.object().shape({
    emailAddress: req(emailAddress),
})

export const emailsCreate = reqArr(emailCreate)