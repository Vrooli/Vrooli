import { description, idArray, id, title } from './base';
import { tagsCreate } from './tag';
import * as yup from 'yup';

export const MIN_USERNAME_LENGTH = 3;
export const MAX_USERNAME_LENGTH = 25;
export const USERNAME_REGEX = /^[a-zA-Z0-9_.-]*$/;
export const USERNAME_REGEX_ERROR = `Characters allowed: letters, numbers, '-', '.', '_'`;
export const MIN_PASSWORD_LENGTH = 8;
export const MAX_PASSWORD_LENGTH = 50;
// See https://stackoverflow. com/a/21456918/10240279 for more options
export const PASSWORD_REGEX = /^(?=.*[A-Za-z])(?=.*\d)[A-Za-z\d@$!%*#?&]{8,}$/;
export const PASSWORD_REGEX_ERROR = "Must be at least 8 characters, with at least one character and one number";

export const usernameSchema = yup.string().min(MIN_USERNAME_LENGTH).max(MAX_USERNAME_LENGTH).matches(USERNAME_REGEX, USERNAME_REGEX_ERROR);
export const passwordSchema = yup.string().min(MIN_PASSWORD_LENGTH).max(MAX_PASSWORD_LENGTH).matches(PASSWORD_REGEX, PASSWORD_REGEX_ERROR);

export const emailSchema = yup.object().shape({
    emailAddress: yup.string().max(128).required(),
    receivesAccountUpdates: yup.bool().default(true).optional(),
    receivesBusinessUpdates: yup.bool().default(true).optional(),
    userId: id,
});

export const roleSchema = yup.object().shape({
    title,
    description,
    userIds: idArray,
});

export const userSchema = yup.object().shape({
    id,
    username: yup.string().max(128).required(),
    emails: yup.array().of(emailSchema).required(),
    status: yup.mixed().oneOf(["Deleted", "Unlocked", "SoftLocked", "HardLocked"]).optional(),
});


// Schema for creating a new account
export const emailSignUpSchema = yup.object().shape({
    username: usernameSchema.required(),
    email: yup.string().email().required(),
    marketingEmails: yup.boolean().required(),
    password: passwordSchema.required(),
    passwordConfirmation: yup.string().oneOf([yup.ref('password'), null], 'Passwords must match')
});

// Schema for updating a user profile
export const profileSchema = yup.object().shape({
    username: usernameSchema.required(),
    email: yup.string().email().required(),
    theme: yup.string().max(128).required(),
    starredTagsCreate: tagsCreate,
    starredTagsConnect: idArray,
    starredTagsDisconnect: idArray,
    hiddenTagsCreate: tagsCreate,
    hiddenTagsConnect: idArray,
    hiddenTagsDisconnect: idArray,
    // Don't apply validation to current password. If you change password requirements, users would be unable to change their password
    currentPassword: yup.string().max(128).required(),
    newPassword: passwordSchema.optional(),
    newPasswordConfirmation: yup.string().oneOf([yup.ref('newPassword'), null], 'Passwords must match')
});

/**
 * Schema for traditional email/password log in.
 * NOTE: Does not include verification code, since it is optional and
 * the schema is reused for the log in form.
 */
export const emailLogInSchema = yup.object().shape({
    email: yup.string().email().required(),
    password: yup.string().max(128).required()
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