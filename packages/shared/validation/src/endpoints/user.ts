import { idArray, id, language, bio, name, handle, maxStrErr, blankToUndefined, email, req, opt, reqArr } from './base';
import { tagsCreate } from './tag';
import * as yup from 'yup';

export const theme = yup.string().transform(blankToUndefined).max(128, maxStrErr);

export const MIN_PASSWORD_LENGTH = 8;
export const MAX_PASSWORD_LENGTH = 50;
// See https://stackoverflow. com/a/21456918/10240279 for more options
export const PASSWORD_REGEX = /^(?=.*[A-Za-z])(?=.*\d)[A-Za-z\d@$!%*#?&]{8,}$/;
export const PASSWORD_REGEX_ERROR = "Must be at least 8 characters, with at least one character and one number";

export const passwordSchema = yup.string().min(MIN_PASSWORD_LENGTH).max(MAX_PASSWORD_LENGTH).matches(PASSWORD_REGEX, PASSWORD_REGEX_ERROR);

export const emailSchema = yup.object().shape({
    emailAddress: req(email),
    receivesAccountUpdates: opt(yup.bool().default(true)),
    receivesBusinessUpdates: opt(yup.bool().default(true)),
    userId: req(id),
});


// Schema for creating a new account
export const emailSignUpSchema = yup.object().shape({
    name: req(name),
    email: req(email),
    marketingEmails: req(yup.boolean()),
    password: req(passwordSchema),
    passwordConfirmation: yup.string().transform(blankToUndefined).oneOf([yup.ref('password'), null], 'Passwords must match')
});

export const userTranslationCreate = yup.object().shape({
    id: req(id),
    language: req(language),
    bio: opt(bio),
});
export const userTranslationUpdate = yup.object().shape({
    id: req(id),
    language: opt(language),
    bio: opt(bio),
});
export const userTranslationsCreate = reqArr(userTranslationCreate)
export const userTranslationsUpdate = reqArr(userTranslationUpdate)

export const profileUpdateSchema = yup.object().shape({
    name: opt(name),
    handle: opt(handle),
    theme: opt(theme),
    starredTagsConnect: opt(idArray),
    starredTagsDisconnect: opt(idArray),
    starredTagsCreate: opt(tagsCreate),
    // Don't apply validation to current password. If you change password requirements, users would be unable to change their password. 
    // Also only require current password when changing new password
    currentPassword: yup.string().transform(blankToUndefined).max(128, maxStrErr).when('newPassword', {
        is: (newPassword: string | undefined) => !!newPassword,
        then: req(yup.string().transform(blankToUndefined)),
        otherwise: yup.string().transform(blankToUndefined).notRequired(),
    }),
    newPassword: opt(passwordSchema),
    newPasswordConfirmation: yup.string().transform(blankToUndefined).oneOf([yup.ref('newPassword'), null], 'Passwords must match'),
    translationsDelete: opt(idArray),
    translationsCreate: opt(userTranslationsCreate),
    translationsUpdate: opt(userTranslationsUpdate),
});

export const profilesUpdate = reqArr(profileUpdateSchema)

/**
 * Schema for traditional email/password log in FORM.
 */
export const emailLogInForm = yup.object().shape({
    email: req(email),
    password: req(yup.string().transform(blankToUndefined).max(128, maxStrErr)),
})

/**
 * Schema for traditional email/password log in.
 */
export const emailLogInSchema = yup.object().shape({
    email: opt(email),
    password: opt(yup.string().transform(blankToUndefined).max(128, maxStrErr)),
    verificationCode: opt(yup.string().transform(blankToUndefined).max(128, maxStrErr)),
})

/**
 * Schema for traditional email/password log in FORM.
 */
 export const userDeleteOneSchema = yup.object().shape({
    password: req(yup.string().transform(blankToUndefined).max(128, maxStrErr)),
    deletePublicData: req(yup.boolean()),
})

/**
 * Schema for sending a password reset request
 */
export const emailRequestPasswordChangeSchema = yup.object().shape({
    email: req(email)
})

/**
 * Schema for resetting your password
 */
export const emailResetPasswordSchema = yup.object().shape({
    newPassword: req(passwordSchema),
    confirmNewPassword: yup.string().oneOf([yup.ref('newPassword'), null], 'Passwords must match')
})