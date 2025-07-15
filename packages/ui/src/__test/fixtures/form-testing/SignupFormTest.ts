import {
    emailSignUpFormValidation,
    endpointsAuth,
    type Session,
} from "@vrooli/shared";
import { createUIFormTestFactory, type UIFormTestConfig } from "./UIFormTestFactory.js";

/**
 * Form data interface for Signup forms
 */
interface SignupFormData {
    // Required fields
    name: string;
    email: string;
    password: string;
    confirmPassword: string;

    // Agreements
    agreeToTerms: boolean;
    marketingEmails: boolean;

    // Optional settings
    theme?: string;
}

/**
 * API input type for signup - used as "shape" since signup doesn't use Shape models
 */
interface SignupApiInput {
    name: string;
    email: string;
    password: string;
    marketingEmails: boolean;
    theme?: string;
}

/**
 * Configuration for Signup form testing with data-driven test scenarios
 */
const signupFormTestConfig: UIFormTestConfig<SignupFormData, SignupFormData, SignupApiInput, SignupApiInput, SignupFormData> = {
    // Form metadata
    objectType: "EmailSignUp",
    formFixtures: {
        minimal: {
            name: "Test User",
            email: "test@example.com",
            password: "SecurePassword123!",
            confirmPassword: "SecurePassword123!",
            agreeToTerms: true,
            marketingEmails: false,
        },
        complete: {
            name: "John Doe",
            email: "john.doe@example.com",
            password: "VerySecurePassword123!",
            confirmPassword: "VerySecurePassword123!",
            agreeToTerms: true,
            marketingEmails: true,
            theme: "light",
        },
        invalid: {
            name: "",
            email: "invalid-email",
            password: "123", // Too short
            confirmPassword: "456", // Doesn't match
            agreeToTerms: false, // Required to be true
            marketingEmails: false,
        },
        edgeCase: {
            name: "A".repeat(100), // At character limit
            email: "very.long.email.address.for.testing@example.com",
            password: "ComplexPassword123!@#$%",
            confirmPassword: "ComplexPassword123!@#$%",
            agreeToTerms: true,
            marketingEmails: true,
            theme: "dark",
        },
    },

    // Validation schemas from shared package
    validation: emailSignUpFormValidation,

    // API endpoints from shared package
    endpoints: {
        create: endpointsAuth.emailSignUp,
        update: endpointsAuth.emailSignUp, // Signup doesn't have update
    },

    // Transform functions - form data to shape
    formToShape: (formData: SignupFormData) => formData,

    transformFunction: (shape: SignupFormData, existing: SignupFormData, isCreate: boolean) => {
        // Convert form data to API input, excluding confirmPassword and agreeToTerms
        return {
            name: shape.name,
            email: shape.email,
            password: shape.password,
            marketingEmails: shape.marketingEmails,
            theme: shape.theme || "light",
        };
    },

    initialValuesFunction: (session?: Session, existing?: Partial<SignupFormData>): SignupFormData => {
        return {
            name: existing?.name || "",
            email: existing?.email || "",
            password: "",
            confirmPassword: "",
            agreeToTerms: false,
            marketingEmails: false,
            theme: existing?.theme || "light",
        };
    },

    // DATA-DRIVEN TEST SCENARIOS - replaces all custom wrapper methods
    testScenarios: {
        nameValidation: {
            description: "Test name field validation",
            testCases: [
                {
                    name: "Valid name",
                    field: "name",
                    value: "John Doe",
                    shouldPass: true,
                },
                {
                    name: "Empty name",
                    field: "name",
                    value: "",
                    shouldPass: false,
                },
                {
                    name: "Single character name",
                    field: "name",
                    value: "A",
                    shouldPass: true,
                },
                {
                    name: "Name at character limit",
                    field: "name",
                    value: "A".repeat(100),
                    shouldPass: true,
                },
                {
                    name: "Name over character limit",
                    field: "name",
                    value: "A".repeat(101),
                    shouldPass: false,
                },
                {
                    name: "Name with numbers",
                    field: "name",
                    value: "John123",
                    shouldPass: true,
                },
                {
                    name: "Name with special characters",
                    field: "name",
                    value: "John O'Connor-Smith",
                    shouldPass: true,
                },
            ],
        },

        emailValidation: {
            description: "Test email field validation",
            testCases: [
                {
                    name: "Valid email",
                    field: "email",
                    value: "user@example.com",
                    shouldPass: true,
                },
                {
                    name: "Email with subdomain",
                    field: "email",
                    value: "user@mail.example.com",
                    shouldPass: true,
                },
                {
                    name: "Email with plus sign",
                    field: "email",
                    value: "user+tag@example.com",
                    shouldPass: true,
                },
                {
                    name: "Empty email",
                    field: "email",
                    value: "",
                    shouldPass: false,
                },
                {
                    name: "Invalid email format",
                    field: "email",
                    value: "not-an-email",
                    shouldPass: false,
                },
                {
                    name: "Email missing @",
                    field: "email",
                    value: "userexample.com",
                    shouldPass: false,
                },
                {
                    name: "Email missing domain",
                    field: "email",
                    value: "user@",
                    shouldPass: false,
                },
            ],
        },

        passwordValidation: {
            description: "Test password field validation and strength",
            testCases: [
                {
                    name: "Strong password",
                    field: "password",
                    value: "SecurePassword123!",
                    shouldPass: true,
                },
                {
                    name: "Minimum length password",
                    field: "password",
                    value: "12345678",
                    shouldPass: true,
                },
                {
                    name: "Empty password",
                    field: "password",
                    value: "",
                    shouldPass: false,
                },
                {
                    name: "Password too short",
                    field: "password",
                    value: "123",
                    shouldPass: false,
                },
                {
                    name: "Very long password",
                    field: "password",
                    value: "A".repeat(1000),
                    shouldPass: true,
                },
                {
                    name: "Password with special characters",
                    field: "password",
                    value: "Pa$$w0rd!@#$%",
                    shouldPass: true,
                },
            ],
        },

        passwordConfirmation: {
            description: "Test password confirmation matching",
            testCases: [
                {
                    name: "Matching passwords",
                    data: {
                        password: "SecurePassword123!",
                        confirmPassword: "SecurePassword123!",
                    },
                    shouldPass: true,
                },
                {
                    name: "Non-matching passwords",
                    data: {
                        password: "SecurePassword123!",
                        confirmPassword: "DifferentPassword456!",
                    },
                    shouldPass: false,
                },
                {
                    name: "Empty confirmation",
                    data: {
                        password: "SecurePassword123!",
                        confirmPassword: "",
                    },
                    shouldPass: false,
                },
            ],
        },

        termsAgreement: {
            description: "Test terms and conditions agreement",
            testCases: [
                {
                    name: "Terms agreed",
                    field: "agreeToTerms",
                    value: true,
                    shouldPass: true,
                },
                {
                    name: "Terms not agreed",
                    field: "agreeToTerms",
                    value: false,
                    shouldPass: false,
                },
            ],
        },

        marketingPreferences: {
            description: "Test marketing email preferences",
            testCases: [
                {
                    name: "Marketing emails opted in",
                    field: "marketingEmails",
                    value: true,
                    shouldPass: true,
                },
                {
                    name: "Marketing emails opted out",
                    field: "marketingEmails",
                    value: false,
                    shouldPass: true,
                },
            ],
        },

        themeSelection: {
            description: "Test theme selection",
            testCases: [
                {
                    name: "Light theme",
                    field: "theme",
                    value: "light",
                    shouldPass: true,
                },
                {
                    name: "Dark theme",
                    field: "theme",
                    value: "dark",
                    shouldPass: true,
                },
                {
                    name: "No theme specified",
                    field: "theme",
                    value: undefined,
                    shouldPass: true,
                },
            ],
        },

        signupWorkflows: {
            description: "Test different signup workflow scenarios",
            testCases: [
                {
                    name: "Basic signup with minimum requirements",
                    data: {
                        name: "Test User",
                        email: "test@example.com",
                        password: "Password123",
                        confirmPassword: "Password123",
                        agreeToTerms: true,
                        marketingEmails: false,
                    },
                    shouldPass: true,
                },
                {
                    name: "Complete signup with all options",
                    data: {
                        name: "John Doe",
                        email: "john.doe@example.com",
                        password: "VerySecurePassword123!",
                        confirmPassword: "VerySecurePassword123!",
                        agreeToTerms: true,
                        marketingEmails: true,
                        theme: "dark",
                    },
                    shouldPass: true,
                },
                {
                    name: "Signup without agreeing to terms",
                    data: {
                        name: "Test User",
                        email: "test@example.com",
                        password: "Password123",
                        confirmPassword: "Password123",
                        agreeToTerms: false,
                        marketingEmails: false,
                    },
                    shouldPass: false,
                },
                {
                    name: "Signup with weak password",
                    data: {
                        name: "Test User",
                        email: "test@example.com",
                        password: "123",
                        confirmPassword: "123",
                        agreeToTerms: true,
                        marketingEmails: false,
                    },
                    shouldPass: false,
                },
                {
                    name: "Signup with invalid email",
                    data: {
                        name: "Test User",
                        email: "invalid-email",
                        password: "SecurePassword123!",
                        confirmPassword: "SecurePassword123!",
                        agreeToTerms: true,
                        marketingEmails: false,
                    },
                    shouldPass: false,
                },
            ],
        },
    },
};

/**
 * SIMPLIFIED: Direct factory export - no wrapper function needed!
 */
export const signupFormTestFactory = createUIFormTestFactory(signupFormTestConfig);

/**
 * Type exports for use in other test files
 */
export { signupFormTestConfig };
export type { SignupFormData };

