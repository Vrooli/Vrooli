import type { EmailCreateInput } from "@vrooli/shared";

/**
 * Form data fixtures for email-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * Email creation form data (for adding new email to account)
 */
export const minimalEmailCreateFormInput: Partial<EmailCreateInput> = {
    emailAddress: "newemail@example.com",
};

export const completeEmailCreateFormInput = {
    emailAddress: "user.name+tag@example.co.uk",
};

/**
 * Email verification form data
 */
export const emailVerificationFormInput = {
    emailAddress: "verify@example.com",
    code: "123456",
};

/**
 * Email update notification preferences
 */
export const emailNotificationPreferencesFormInput = {
    emailAddress: "notifications@example.com",
    receiveMarketing: true,
    receiveUpdates: true,
    receiveComments: true,
    receiveMentions: true,
    receiveReminders: true,
};

/**
 * Email-related form data for authentication flows
 */
export const emailAuthFormInputs = {
    forgotPassword: {
        emailAddress: "forgot@example.com",
    },
    resendVerification: {
        emailAddress: "resend@example.com",
    },
    changeEmail: {
        currentEmail: "current@example.com",
        newEmail: "new@example.com",
        password: "CurrentPassword123!",
    },
};

/**
 * Edge case email inputs for testing validation
 */
export const emailEdgeCaseFormInputs = {
    withPlus: {
        emailAddress: "test+filter@gmail.com",
    },
    withDots: {
        emailAddress: "first.last@company.com",
    },
    withNumbers: {
        emailAddress: "user123@example456.com",
    },
    withHyphens: {
        emailAddress: "test@my-company.com",
    },
    shortValid: {
        emailAddress: "a@b.c",
    },
    longValid: {
        emailAddress: `${"a".repeat(50)}@${"b".repeat(50)}.com`,
    },
};

/**
 * Invalid email inputs for testing validation
 */
export const invalidEmailFormInputs = {
    empty: {
        emailAddress: "",
    },
    whitespaceOnly: {
        emailAddress: "   ",
    },
    missingAt: {
        emailAddress: "notanemail.com",
    },
    missingDomain: {
        emailAddress: "user@",
    },
    missingLocal: {
        emailAddress: "@example.com",
    },
    multipleAt: {
        emailAddress: "user@@example.com",
    },
    invalidCharacters: {
        emailAddress: "user<>@example.com",
    },
    tooLong: {
        emailAddress: `${"a".repeat(250)}@example.com`,
    },
};

/**
 * Form validation states
 */
export const emailFormValidationStates = {
    pristine: {
        values: {
            emailAddress: "",
        },
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            emailAddress: "invalid-email",
        },
        errors: {
            emailAddress: "Please enter a valid email address",
        },
        touched: {
            emailAddress: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalEmailCreateFormInput,
        errors: {},
        touched: {
            emailAddress: true,
        },
        isValid: true,
        isSubmitting: false,
    },
    submitting: {
        values: minimalEmailCreateFormInput,
        errors: {},
        touched: {
            emailAddress: true,
        },
        isValid: true,
        isSubmitting: true,
    },
};

/**
 * Helper function to create email form initial values
 */
export const createEmailFormInitialValues = (emailData?: Partial<any>) => ({
    emailAddress: emailData?.emailAddress || "",
    ...emailData,
});

/**
 * Helper function to validate email format
 */
export const validateEmailFormat = (email: string): string | null => {
    if (!email) return "Email is required";
    if (email.length > 256) return "Email is too long";
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(email)) return "Please enter a valid email address";
    return null;
};

/**
 * Helper function to transform form data to API format
 */
export const transformEmailFormToApiInput = (formData: any) => ({
    emailAddress: formData.emailAddress?.trim() || "",
    // Add any additional transformations needed
});

/**
 * Mock email suggestions for autocomplete
 */
export const mockEmailSuggestions = {
    domains: [
        "@gmail.com",
        "@yahoo.com",
        "@outlook.com",
        "@hotmail.com",
        "@icloud.com",
        "@protonmail.com",
    ],
    recentEmails: [
        "john.doe@example.com",
        "jane.smith@company.com",
        "test.user@gmail.com",
    ],
};

/**
 * Email list form data (for managing multiple emails)
 */
export const emailListFormData = {
    emails: [
        {
            id: "email_001",
            emailAddress: "primary@example.com",
            verified: true,
            isPrimary: true,
        },
        {
            id: "email_002",
            emailAddress: "secondary@example.com",
            verified: false,
            isPrimary: false,
        },
        {
            id: "email_003",
            emailAddress: "work@company.com",
            verified: true,
            isPrimary: false,
        },
    ],
};