import { idArray, id, language, bio, name, handle } from './base';
import { tagsCreate } from './tag';
import * as yup from 'yup';
import { tagHiddensCreate, tagHiddensUpdate } from './tagHidden';

export const theme = yup.string().max(128);

export const MIN_PASSWORD_LENGTH = 8;
export const MAX_PASSWORD_LENGTH = 50;
// See https://stackoverflow. com/a/21456918/10240279 for more options
export const PASSWORD_REGEX = /^(?=.*[A-Za-z])(?=.*\d)[A-Za-z\d@$!%*#?&]{8,}$/;
export const PASSWORD_REGEX_ERROR = "Must be at least 8 characters, with at least one character and one number";

export const passwordSchema = yup.string().min(MIN_PASSWORD_LENGTH).max(MAX_PASSWORD_LENGTH).matches(PASSWORD_REGEX, PASSWORD_REGEX_ERROR);

export const emailSchema = yup.object().shape({
    emailAddress: yup.string().max(128).required(),
    receivesAccountUpdates: yup.bool().default(true).notRequired().default(undefined),
    receivesBusinessUpdates: yup.bool().default(true).notRequired().default(undefined),
    userId: id.required(),
});


// Schema for creating a new account
export const emailSignUpSchema = yup.object().shape({
    name: name.required(),
    email: yup.string().email().required(),
    marketingEmails: yup.boolean().required(),
    password: passwordSchema.required(),
    passwordConfirmation: yup.string().oneOf([yup.ref('password'), null], 'Passwords must match')
});

export const userTranslationCreate = yup.object().shape({
    id: id.required(),
    language: language.required(),
    bio: bio.required(),
});
export const userTranslationUpdate = yup.object().shape({
    id: id.required(),
    language: language.notRequired().default(undefined),
    bio: bio.notRequired().default(undefined),
});
export const userTranslationsCreate = yup.array().of(userTranslationCreate.required())
export const userTranslationsUpdate = yup.array().of(userTranslationUpdate.required())

export const profileUpdateSchema = yup.object().shape({
    name: name.notRequired().default(undefined),
    handle: handle.notRequired().default(undefined),
    theme: theme.notRequired().default(undefined),
    starredTagsConnect: idArray.notRequired().default(undefined),
    starredTagsDisconnect: idArray.notRequired().default(undefined),
    starredTagsCreate: tagsCreate.notRequired().default(undefined),
    hiddenTagsDelete: idArray.notRequired().default(undefined),
    hiddenTagsCreate: tagHiddensCreate.notRequired().default(undefined),
    hiddenTagsUpdate: tagHiddensUpdate.notRequired().default(undefined),
    // Don't apply validation to current password. If you change password requirements, users would be unable to change their password. 
    // Also only require current password when changing new password
    currentPassword: yup.string().max(128).when('newPassword', {
        is: (newPassword: string | undefined) => !!newPassword,
        then: yup.string().required(),
        otherwise: yup.string().notRequired(),
    }),
    newPassword: passwordSchema.notRequired().default(undefined),
    newPasswordConfirmation: yup.string().oneOf([yup.ref('newPassword'), null], 'Passwords must match'),
    translationsDelete: idArray.notRequired().default(undefined),
    translationsCreate: userTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: userTranslationsUpdate.notRequired().default(undefined),
});

/**
 * Schema for traditional email/password log in FORM.
 */
export const emailLogInForm = yup.object().shape({
    email: yup.string().email().required(),
    password: yup.string().max(128).required(),
})

/**
 * Schema for traditional email/password log in.
 */
export const emailLogInSchema = yup.object().shape({
    email: yup.string().email().notRequired().default(undefined),
    password: yup.string().max(128).notRequired().default(undefined),
    verificationCode: yup.string().max(128).notRequired().default(undefined),
})

/**
 * Schema for sending a password reset request
 */
export const emailRequestPasswordChangeSchema = yup.object().shape({
    email: yup.string().email().required()
})

/**
 * Schema for resetting your password
 */
export const emailResetPasswordSchema = yup.object().shape({
    newPassword: passwordSchema.required(),
    confirmNewPassword: yup.string().oneOf([yup.ref('newPassword'), null], 'Passwords must match')
})