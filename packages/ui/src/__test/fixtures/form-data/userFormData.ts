import type { ProfileUpdateInput } from "@vrooli/shared";

/**
 * Form data fixtures for user-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * User registration form data
 * Note: Registration forms have UI-specific fields not present in API types
 */
export const minimalRegistrationFormInput = {
    email: "newuser@example.com",
    password: "SecurePassword123!",
    confirmPassword: "SecurePassword123!",
    handle: "newuser",
    name: "New User",
    acceptTerms: true,
};

export const completeRegistrationFormInput = {
    email: "poweruser@example.com",
    password: "SuperSecure123!@#",
    confirmPassword: "SuperSecure123!@#",
    handle: "poweruser",
    name: "Power User",
    bio: "Passionate developer interested in AI and automation",
    theme: "dark",
    language: "en",
    timezone: "America/New_York",
    marketingEmails: true,
    acceptTerms: true,
    referralCode: "FRIEND2024",
};

/**
 * User login form data
 */
export const loginFormInput = {
    emailOrHandle: "testuser@example.com",
    password: "MyPassword123!",
    rememberMe: true,
};

export const loginWithHandleFormInput = {
    emailOrHandle: "testuser",
    password: "MyPassword123!",
    rememberMe: false,
};

/**
 * User profile update form data
 */
export const minimalProfileUpdateFormInput: Partial<ProfileUpdateInput> = {
    // Note: ProfileUpdateInput uses translations, but forms often work with simpler structure
    translationsCreate: [
        {
            language: "en",
            name: "Updated Name",
            bio: "Updated bio text",
        },
    ],
};

export const completeProfileUpdateFormInput = {
    name: "Professional User",
    bio: "Senior developer with 10+ years of experience in AI/ML. Building the future of automation.",
    theme: "light",
    language: "en",
    timezone: "UTC",
    isPrivate: false,
    showBookmarks: true,
    showProjects: true,
    profileImage: null, // File object would go here
    bannerImage: null, // File object would go here
    links: [
        { type: "website", url: "https://myportfolio.com" },
        { type: "github", url: "https://github.com/username" },
        { type: "linkedin", url: "https://linkedin.com/in/username" },
    ],
};

/**
 * Password reset form data
 */
export const forgotPasswordFormInput = {
    email: "user@example.com",
};

export const resetPasswordFormInput = {
    token: "reset_token_123456789",
    newPassword: "NewSecurePassword123!",
    confirmPassword: "NewSecurePassword123!",
};

/**
 * Account settings form data
 */
export const privacySettingsFormInput = {
    isPrivate: true,
    showBookmarks: false,
    showProjects: false,
    showRoutines: true,
    allowMessages: "contacts", // "anyone" | "contacts" | "nobody"
    allowInvites: "anyone",
};

export const notificationSettingsFormInput = {
    emailNotifications: {
        comments: true,
        mentions: true,
        updates: true,
        newsletter: false,
    },
    pushNotifications: {
        comments: true,
        mentions: true,
        updates: false,
        reminders: true,
    },
    notificationFrequency: "realtime", // "realtime" | "daily" | "weekly"
};

export const securitySettingsFormInput = {
    twoFactorEnabled: true,
    twoFactorMethod: "app", // "app" | "sms"
    sessionTimeout: 30, // minutes
    trustedDevices: [],
};

/**
 * Delete account form data
 */
export const deleteAccountFormInput = {
    confirmText: "DELETE",
    password: "CurrentPassword123!",
    reason: "No longer needed",
    feedback: "The service was great, but I'm moving to a different solution.",
};

/**
 * Form validation states
 */
export const formValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            email: "invalid-email",
            password: "123", // Too short
        },
        errors: {
            email: "Please enter a valid email address",
            password: "Password must be at least 8 characters",
            confirmPassword: "Passwords do not match",
        },
        touched: {
            email: true,
            password: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalRegistrationFormInput,
        errors: {},
        touched: {
            email: true,
            password: true,
            confirmPassword: true,
            handle: true,
            name: true,
            acceptTerms: true,
        },
        isValid: true,
        isSubmitting: false,
    },
    submitting: {
        values: minimalRegistrationFormInput,
        errors: {},
        touched: {},
        isValid: true,
        isSubmitting: true,
    },
};

/**
 * Helper function to create form initial values
 */
export const createUserFormInitialValues = (userData?: Partial<any>) => ({
    email: userData?.email || "",
    handle: userData?.handle || "",
    name: userData?.name || "",
    bio: userData?.bio || "",
    theme: userData?.theme || "light",
    language: userData?.language || "en",
    isPrivate: userData?.isPrivate || false,
    ...userData,
});

/**
 * Helper function to transform form data to API format
 */
export const transformUserFormToApiInput = (formData: any) => ({
    ...formData,
    // Convert file objects to upload format
    profileImage: formData.profileImage?.file || undefined,
    bannerImage: formData.bannerImage?.file || undefined,
    // Convert links array to proper format
    links: formData.links?.filter((link: any) => link.url) || [],
});

/**
 * Mock file inputs for testing
 */
export const mockFileInputs = {
    profileImage: {
        file: null, // Would be File object
        preview: "data:image/jpeg;base64,/9j/4AAQ...", // Base64 preview
        name: "profile.jpg",
        size: 45678,
        type: "image/jpeg",
    },
    bannerImage: {
        file: null, // Would be File object
        preview: "data:image/jpeg;base64,/9j/4BBQ...", // Base64 preview
        name: "banner.jpg",
        size: 123456,
        type: "image/jpeg",
    },
};