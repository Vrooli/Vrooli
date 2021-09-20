import * as yup from 'yup';
import { ACCOUNT_STATUS, DEFAULT_PRONOUNS } from './modelConsts';

export const MIN_PASSWORD_LENGTH = 8;
export const MAX_PASSWORD_LENGTH = 50;
// See https://stackoverflow.com/a/21456918/10240279 for more options
export const PASSWORD_REGEX = /^(?=.*[A-Za-z])(?=.*\d)[A-Za-z\d@$!%*#?&]{8,}$/;
export const PASSWORD_REGEX_ERROR = "Must be at least 8 characters, with at least one character and one number";

export const passwordSchema = yup.string().min(MIN_PASSWORD_LENGTH).max(MAX_PASSWORD_LENGTH).matches(PASSWORD_REGEX, PASSWORD_REGEX_ERROR);

export const businessSchema = yup.object().shape({
    name: yup.string().max(128).required(),
    subscribedToNewsletters: yup.boolean().optional(),
    employeeIds: yup.array().of(yup.string().required()).optional(),
})

export const addressSchema = yup.object().shape({
    tag: yup.string().max(128).optional(),
    name: yup.string().max(128).optional(),
    country: yup.string().max(2).default('US').required(),
    administrativeArea: yup.string().max(64).required(),
    subAdministrativeArea: yup.string().max(64).optional(),
    locality: yup.string().max(64).required(),
    postalCode: yup.string().max(16).required(),
    throughfare: yup.string().max(256).required(),
    premise: yup.string().max(64).optional(),
    deliveryInstructions: yup.string().max(1024).optional(),
    businessId: yup.string().max(128).required(),
});

export const emailSchema = yup.object().shape({
    emailAddress: yup.string().max(128).required(),
    receivesDeliveryUpdates: yup.bool().default(true).optional(),
    customerId: yup.string().optional(),
    businessId: yup.string().optional(),
});

export const feedbackSchema = yup.object().shape({
    text: yup.string().max(4096).required(),
    customerId: yup.string().required(),
});

export const imageSchema = yup.object().shape({
    files: yup.array().of(yup.object().shape({
        src: yup.string().required(),
        alt: yup.string().max(256).optional(),
        description: yup.string().max(1024).optional(),
        labels: yup.array().of(yup.string().max(128).required()).required(),
    })).required(),
});

export const phoneSchema = yup.object().shape({
    number: yup.string().max(20).required(),
    receivesDeliveryUpdates: yup.bool().default(true).required(),
});

export const roleSchema = yup.object().shape({
    title: yup.string().max(128).required(),
    description: yup.string().max(2048).optional(),
    customerIds: yup.array().of(yup.string().required()).optional(),
});

export const customerSchema = yup.object().shape({
    id: yup.string().max(256).optional(),
    firstName: yup.string().max(128).required(),
    lastName: yup.string().max(128).required(),
    pronouns: yup.string().max(128).default(DEFAULT_PRONOUNS[0]).optional(),
    business: businessSchema,
    emails: yup.array().of(emailSchema).required(),
    phones: yup.array().of(phoneSchema).required(),
    status: yup.mixed().oneOf(Object.values(ACCOUNT_STATUS)).optional(),
});


// Schema for creating a new account
export const signUpSchema = yup.object().shape({
    firstName: yup.string().max(128).required(),
    lastName: yup.string().max(128).required(),
    pronouns: yup.string().max(128).default(DEFAULT_PRONOUNS[0]).optional(),
    business: yup.string().max(128).required(),
    email: yup.string().email().required(),
    phone: yup.string().max(20).required(),
    marketingEmails: yup.boolean().required(),
    password: passwordSchema.required(),
    passwordConfirmation: yup.string().oneOf([yup.ref('password'), null], 'Passwords must match')
});

// Schema for creating a new customer
export const addCustomerSchema = yup.object().shape({
    firstName: yup.string().max(128).required(),
    lastName: yup.string().max(128).required(),
    pronouns: yup.string().max(128).default(DEFAULT_PRONOUNS[0]).optional(),
    business: yup.string().max(128).required(),
    email: yup.string().email().required(),
    phone: yup.string().max(20).required(),
});

// Schema for updating a customer profile
export const profileSchema = yup.object().shape({
    firstName: yup.string().max(128).required(),
    lastName: yup.string().max(128).required(),
    pronouns: yup.string().max(128).default(DEFAULT_PRONOUNS[0]).optional(),
    business: yup.string().max(128).required(),
    email: yup.string().email().required(),
    phone: yup.string().max(20).required(),
    theme: yup.string().max(128).required(),
    // Don't apply validation to current password. If you change password requirements, customers would be unable to change their password
    currentPassword: yup.string().max(128).required(),
    newPassword: passwordSchema.optional(),
    newPasswordConfirmation: yup.string().oneOf([yup.ref('newPassword'), null], 'Passwords must match')
});

// Schema for logging in
export const logInSchema = yup.object().shape({
    email: yup.string().email().required(),
    password: yup.string().max(128).required()
})

// Schema for sending a password reset request
export const requestPasswordChangeSchema = yup.object().shape({
    email: yup.string().email().required()
})

// Schema for resetting password
export const resetPasswordSchema = yup.object().shape({
    newPassword: passwordSchema.required(),
    confirmNewPassword: yup.string().oneOf([yup.ref('newPassword'), null], 'Passwords must match')
})


// Schema for adding an employee to a business
export const employeeSchema = yup.object().shape({
    firstName: yup.string().max(128).required(),
    lastName: yup.string().max(128).required(),
    pronouns: yup.string().max(128).default(DEFAULT_PRONOUNS[0]).optional(),
    password: passwordSchema.required()
});