import type { PhoneCreateInput } from "@vrooli/shared";

/**
 * Form data fixtures for phone-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * Phone creation form data (for adding new phone to account)
 */
export const minimalPhoneCreateFormInput: Partial<PhoneCreateInput> = {
    phoneNumber: "+1234567890",
};

export const completePhoneCreateFormInput = {
    phoneNumber: "+12025551234", // US format with proper area code
};

/**
 * Phone verification form data
 */
export const phoneVerificationFormInput = {
    phoneNumber: "+1234567890",
    code: "123456",
};

/**
 * Phone update notification preferences
 */
export const phoneNotificationPreferencesFormInput = {
    phoneNumber: "+1234567890",
    receiveSms: true,
    receiveAlerts: true,
    receiveReminders: false,
    marketingMessages: false,
};

/**
 * Phone-related form data for authentication flows
 */
export const phoneAuthFormInputs = {
    twoFactorSetup: {
        phoneNumber: "+1234567890",
    },
    twoFactorVerification: {
        phoneNumber: "+1234567890",
        code: "123456",
    },
    changePhone: {
        currentPhone: "+1234567890",
        newPhone: "+10987654321",
        password: "CurrentPassword123!",
    },
};

/**
 * International phone number formats for testing
 */
export const internationalPhoneFormInputs = {
    us: {
        phoneNumber: "+12025551234", // US
    },
    usWithFormatting: {
        phoneNumber: "+1 (202) 555-1234", // US with formatting
    },
    uk: {
        phoneNumber: "+442079460958", // UK
    },
    ukWithFormatting: {
        phoneNumber: "+44 20 7946 0958", // UK with formatting
    },
    germany: {
        phoneNumber: "+49301234567", // Germany
    },
    japan: {
        phoneNumber: "+81312345678", // Japan
    },
    australia: {
        phoneNumber: "+61212345678", // Australia
    },
    india: {
        phoneNumber: "+919876543210", // India
    },
};

/**
 * Edge case phone inputs for testing validation
 */
export const phoneEdgeCaseFormInputs = {
    withSpaces: {
        phoneNumber: "+1 234 567 8900",
    },
    withDashes: {
        phoneNumber: "+1-234-567-8900",
    },
    withParentheses: {
        phoneNumber: "+1(234)567-8900",
    },
    withDots: {
        phoneNumber: "+1.234.567.8900",
    },
    minimalLength: {
        phoneNumber: "+1", // Minimum valid format
    },
    maximalLength: {
        phoneNumber: "+12345678901234", // 16 characters exactly
    },
    withExtension: {
        phoneNumber: "+12345678900", // Basic format, extensions handled separately
        extension: "1234",
    },
};

/**
 * Invalid phone inputs for testing validation
 */
export const invalidPhoneFormInputs = {
    empty: {
        phoneNumber: "",
    },
    whitespaceOnly: {
        phoneNumber: "   ",
    },
    missingPlus: {
        phoneNumber: "1234567890", // Missing + prefix
    },
    tooShort: {
        phoneNumber: "+", // Just the plus sign
    },
    tooLong: {
        phoneNumber: "+123456789012345678901", // Exceeds 16 character limit
    },
    invalidCharacters: {
        phoneNumber: "+123-456-ABCD", // Contains letters
    },
    invalidFormat: {
        phoneNumber: "++1234567890", // Double plus
    },
    localOnly: {
        phoneNumber: "555-1234", // Local number without country code
    },
};

/**
 * Form validation states
 */
export const phoneFormValidationStates = {
    pristine: {
        values: {
            phoneNumber: "",
        },
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            phoneNumber: "1234567890", // Missing + prefix
        },
        errors: {
            phoneNumber: "Please enter a valid phone number with country code",
        },
        touched: {
            phoneNumber: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalPhoneCreateFormInput,
        errors: {},
        touched: {
            phoneNumber: true,
        },
        isValid: true,
        isSubmitting: false,
    },
    submitting: {
        values: minimalPhoneCreateFormInput,
        errors: {},
        touched: {
            phoneNumber: true,
        },
        isValid: true,
        isSubmitting: true,
    },
};

/**
 * Helper function to create phone form initial values
 */
export const createPhoneFormInitialValues = (phoneData?: Partial<any>) => ({
    phoneNumber: phoneData?.phoneNumber || "",
    ...phoneData,
});

/**
 * Helper function to validate phone format
 */
export const validatePhoneFormat = (phone: string): string | null => {
    if (!phone) return "Phone number is required";
    if (!phone.startsWith("+")) return "Phone number must start with country code (+)";
    if (phone.length < 2) return "Phone number is too short";
    if (phone.length > 16) return "Phone number is too long";
    const phoneRegex = /^\+[\d\s\-().]+$/;
    if (!phoneRegex.test(phone)) return "Please enter a valid phone number";
    return null;
};

/**
 * Helper function to transform form data to API format
 */
export const transformPhoneFormToApiInput = (formData: any) => ({
    phoneNumber: formData.phoneNumber?.replace(/[\s\-().]/g, "") || "", // Remove formatting
    // Add any additional transformations needed
});

/**
 * Helper function to format phone number for display
 */
export const formatPhoneForDisplay = (phone: string): string => {
    // Remove all formatting first
    const cleaned = phone.replace(/[\s\-().]/g, "");
    
    // US/Canada format
    if (cleaned.startsWith("+1") && cleaned.length === 12) {
        return `+1 (${cleaned.slice(2, 5)}) ${cleaned.slice(5, 8)}-${cleaned.slice(8)}`;
    }
    
    // Return with basic spacing for other formats
    return phone;
};

/**
 * Mock country code suggestions for autocomplete
 */
export const mockCountryCodeSuggestions = [
    { code: "+1", country: "United States / Canada" },
    { code: "+44", country: "United Kingdom" },
    { code: "+49", country: "Germany" },
    { code: "+33", country: "France" },
    { code: "+81", country: "Japan" },
    { code: "+86", country: "China" },
    { code: "+91", country: "India" },
    { code: "+61", country: "Australia" },
];

/**
 * Phone list form data (for managing multiple phones)
 */
export const phoneListFormData = {
    phones: [
        {
            id: "phone_001",
            phoneNumber: "+12025551234",
            verified: true,
            isPrimary: true,
        },
        {
            id: "phone_002",
            phoneNumber: "+14155552345",
            verified: false,
            isPrimary: false,
        },
        {
            id: "phone_003",
            phoneNumber: "+442079460958",
            verified: true,
            isPrimary: false,
        },
    ],
};