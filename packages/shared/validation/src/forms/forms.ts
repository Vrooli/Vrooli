import * as yup from 'yup';
import { blankToUndefined, description, details, email, id, language, maxStrErr, name, opt, password, reportCreatedFor, reportReason, req } from '../utils';

export const nodeEndFormValidation = yup.object().shape({
    wasSuccessful: opt(yup.boolean()),
    name: req(name),
    description: opt(description),
});

export const emailLogInFormValidation = yup.object().shape({
    email: req(email),
    password: req(yup.string().transform(blankToUndefined).max(128, maxStrErr)),
})

export const emailSignUpFormValidation = yup.object().shape({
    name: req(name),
    email: req(email),
    marketingEmails: req(yup.boolean()),
    password: req(password),
    passwordConfirmation: yup.string().transform(blankToUndefined).oneOf([yup.ref('password'), null], 'Passwords must match')
});

export const reportCreateForm = yup.object().shape({
    createdFor: req(reportCreatedFor),
    createdForId: req(id),
    details: opt(details),
    language: req(language),
    reason: req(reportReason),
    otherReason: opt(reportReason),
})