/**
 * SignupView Form Fixtures
 * 
 * Provides form data fixtures for SignupView testing.
 * These fixtures simulate different user input scenarios for the signup form.
 */
import { type SignupViewFormInput } from "../../../views/auth/SignupView.js";

export type SignupScenario = "valid" | "weakPassword" | "emailMismatch" | "passwordMismatch" | "missingTerms" | "invalidEmail";

/**
 * Creates SignupView form data for different test scenarios
 * @param scenario The test scenario to generate data for
 * @returns Form data matching the SignupViewFormInput type
 */
export function createSignupViewFormData(scenario: SignupScenario): SignupViewFormInput {
    const baseData: SignupViewFormInput = {
        name: "Test User",
        email: "test@example.com",
        password: "SecurePassword123!",
        confirmPassword: "SecurePassword123!",
        marketingEmails: false,
        theme: "light",
        agreeToTerms: true,
    };

    switch (scenario) {
        case "valid":
            return baseData;

        case "weakPassword":
            return {
                ...baseData,
                password: "123",
                confirmPassword: "123",
            };

        case "emailMismatch":
            // This was originally "emailMismatch" but appears to be testing invalid email format
            return {
                ...baseData,
                email: "invalid-email",
            };

        case "passwordMismatch":
            return {
                ...baseData,
                confirmPassword: "DifferentPassword123!",
            };

        case "missingTerms":
            return {
                ...baseData,
                agreeToTerms: false,
            };

        case "invalidEmail":
            return {
                ...baseData,
                email: "not-an-email",
            };

        default:
            throw new Error(`Unknown scenario: ${scenario}`);
    }
}

/**
 * Helper to get expected validation errors for each scenario
 */
export function getExpectedErrors(scenario: SignupScenario): {
    field?: keyof SignupViewFormInput;
    message?: string;
    shouldSubmit: boolean;
} {
    switch (scenario) {
        case "valid":
            return { shouldSubmit: true };

        case "weakPassword":
            return {
                field: "password",
                message: "Password must be at least 8 characters",
                shouldSubmit: false,
            };

        case "emailMismatch":
        case "invalidEmail":
            return {
                field: "email",
                message: "Invalid email format",
                shouldSubmit: false,
            };

        case "passwordMismatch":
            return {
                field: "confirmPassword",
                message: "Passwords don't match",
                shouldSubmit: false,
            };

        case "missingTerms":
            return {
                field: "agreeToTerms",
                message: "Please accept the terms to continue",
                shouldSubmit: false,
            };

        default:
            return { shouldSubmit: false };
    }
}