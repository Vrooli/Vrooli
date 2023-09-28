import { id, optArr } from "@local/shared";
import * as yup from "yup";
import { description, email, maxStrErr, name, opt, password, req } from "../utils";

export const nodeEndFormValidation = yup.object().shape({
    wasSuccessful: opt(yup.boolean()),
    name: req(name),
    description: opt(description),
});

export const emailLogInFormValidation = yup.object().shape({
    email: req(email),
    password: req(yup.string().trim().removeEmptyString().max(128, maxStrErr)),
});

export const emailSignUpFormValidation = yup.object().shape({
    name: req(name),
    email: req(email),
    marketingEmails: req(yup.boolean()),
    password: req(password),
});

export const profileEmailUpdateFormValidation = yup.object().shape({
    currentPassword: req(password),
    newPassword: opt(password),
    emailsCreate: optArr(yup.object().shape({
        emailAddress: req(email),
    })),
    emailsDelete: optArr(id),
});
