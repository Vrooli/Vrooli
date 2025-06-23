/**
 * User Round-Trip Tests - Standardized Approach
 * 
 * This demonstrates the recommended pattern for integration tests using
 * the standardized helper system for consistent testing across all endpoints.
 */
import { emailSignUpValidation, profileEmailUpdateValidation } from "@vrooli/shared";
import { type SettingsProfileFormInput, type SignupViewFormInput } from "@vrooli/ui";
import { describe, expect, it } from "vitest";
import { testHelpers } from "../utils/standardized-helpers.js";

/**
 * Form Data Fixtures
 * Simulates form interactions from the UI
 */
class StandardizedFormFixtures {
    createSignupForm(scenario: "valid" | "weakPassword" | "invalidEmail"): SignupViewFormInput {
        const base: SignupViewFormInput = {
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
                return base;
            case "weakPassword":
                return { ...base, password: "123", confirmPassword: "123" };
            case "invalidEmail":
                return { ...base, email: "invalid-email" };
            default:
                throw new Error(`Unknown scenario: ${scenario}`);
        }
    }

    createProfileUpdateForm(scenario: "minimal" | "complete" | "invalidHandle"): SettingsProfileFormInput {
        switch (scenario) {
            case "minimal":
                return { bio: "Updated bio text" };
            case "complete":
                return {
                    name: "Updated Name",
                    bio: "I am a developer interested in AI and automation.",
                    handle: "updated_handle",
                    theme: "dark",
                    language: "es",
                };
            case "invalidHandle":
                return { handle: "" }; // Empty handle should fail validation
            default:
                throw new Error(`Unknown scenario: ${scenario}`);
        }
    }
}

/**
 * Shape Transformation Fixtures
 * Converts form data to API input format
 */
class StandardizedShapeFixtures {
    signupFormToApi(formData: SignupViewFormInput) {
        return {
            name: formData.name,
            email: formData.email,
            password: formData.password,
            confirmPassword: formData.confirmPassword,
            marketingEmails: formData.marketingEmails,
        };
    }

    profileUpdateFormToApi(formData: SettingsProfileFormInput, userId: string) {
        return {
            id: userId,
            ...formData,
        };
    }
}

/**
 * Validation Fixtures
 * Uses actual shared validation logic
 */
class StandardizedValidationFixtures {
    async validateSignup(apiInput: any) {
        try {
            const validatedData = await emailSignUpValidation.validate(apiInput);
            return { isValid: true, data: validatedData };
        } catch (error) {
            return { isValid: false, errors: [error] };
        }
    }

    async validateProfileUpdate(apiInput: any) {
        try {
            const validatedData = await profileEmailUpdateValidation.validate(apiInput);
            return { isValid: true, data: validatedData };
        } catch (error) {
            return { isValid: false, errors: [error] };
        }
    }
}

describe("User Round-Trip Tests - Standardized", () => {
    const formFixtures = new StandardizedFormFixtures();
    const shapeFixtures = new StandardizedShapeFixtures();
    const validationFixtures = new StandardizedValidationFixtures();

    describe("Complete Pipeline Tests", () => {
        it("should complete full signup pipeline with valid data", async () => {
            // 1. Create form data
            const formData = formFixtures.createSignupForm("valid");

            // 2. Run complete pipeline
            const pipelineResult = await testHelpers.runSignupPipeline(
                formData,
                (data) => shapeFixtures.signupFormToApi(data),
                (data) => validationFixtures.validateSignup(data),
            );

            // 3. Assert pipeline results
            expect(pipelineResult.validationResult.isValid).toBe(true);
            expect(pipelineResult.endpointResult.success).toBe(true);
            expect(pipelineResult.responseValidation.isValid).toBe(true);
            expect(pipelineResult.dbVerification.isValid).toBe(true);

            // 4. Log detailed results for debugging
            testHelpers.logPipelineResult("Valid Signup", pipelineResult);

            console.log("✅ Complete signup pipeline validated successfully");
        }, 30000); // Longer timeout for full pipeline

        it("should complete full profile update pipeline", async () => {
            // 0. Setup - create test user
            const { user } = await testHelpers.createTestUser({
                name: "Original Name",
                bio: "Original bio",
            });

            // 1. Create form data
            const formData = formFixtures.createProfileUpdateForm("complete");

            // 2. Run complete pipeline
            const pipelineResult = await testHelpers.runProfileUpdatePipeline(
                formData,
                user.id.toString(),
                (data, userId) => shapeFixtures.profileUpdateFormToApi(data, userId),
                (data) => validationFixtures.validateProfileUpdate(data),
            );

            // 3. Assert pipeline results
            expect(pipelineResult.validationResult.isValid).toBe(true);
            expect(pipelineResult.endpointResult.success).toBe(true);
            expect(pipelineResult.responseValidation.isValid).toBe(true);
            expect(pipelineResult.dbVerification.isValid).toBe(true);

            // 4. Log detailed results for debugging
            testHelpers.logPipelineResult("Profile Update", pipelineResult);

            console.log("✅ Complete profile update pipeline validated successfully");
        }, 30000);
    });

    describe("Validation Failure Tests", () => {
        it("should handle weak password validation failure", async () => {
            const formData = formFixtures.createSignupForm("weakPassword");

            const pipelineResult = await testHelpers.runSignupPipeline(
                formData,
                (data) => shapeFixtures.signupFormToApi(data),
                (data) => validationFixtures.validateSignup(data),
            );

            // Validation should fail
            expect(pipelineResult.validationResult.isValid).toBe(false);
            // Endpoint should also fail
            expect(pipelineResult.endpointResult.success).toBe(false);

            testHelpers.logPipelineResult("Weak Password", pipelineResult);

            console.log("✅ Weak password properly rejected");
        });

        it("should handle invalid email validation failure", async () => {
            const formData = formFixtures.createSignupForm("invalidEmail");

            const pipelineResult = await testHelpers.runSignupPipeline(
                formData,
                (data) => shapeFixtures.signupFormToApi(data),
                (data) => validationFixtures.validateSignup(data),
            );

            // Validation should fail
            expect(pipelineResult.validationResult.isValid).toBe(false);
            // Endpoint should also fail
            expect(pipelineResult.endpointResult.success).toBe(false);

            testHelpers.logPipelineResult("Invalid Email", pipelineResult);

            console.log("✅ Invalid email properly rejected");
        });

        it("should handle invalid handle in profile update", async () => {
            const { user } = await testHelpers.createTestUser();
            const formData = formFixtures.createProfileUpdateForm("invalidHandle");

            const pipelineResult = await testHelpers.runProfileUpdatePipeline(
                formData,
                user.id.toString(),
                (data, userId) => shapeFixtures.profileUpdateFormToApi(data, userId),
                (data) => validationFixtures.validateProfileUpdate(data),
            );

            // Validation should fail
            expect(pipelineResult.validationResult.isValid).toBe(false);
            // Endpoint should also fail
            expect(pipelineResult.endpointResult.success).toBe(false);

            testHelpers.logPipelineResult("Invalid Handle", pipelineResult);

            console.log("✅ Invalid handle properly rejected");
        });
    });

    describe("Individual Component Tests", () => {
        it("should test endpoint calling in isolation", async () => {
            const validSignupData = {
                name: "Test User",
                email: "isolated@example.com",
                password: "SecurePassword123!",
                confirmPassword: "SecurePassword123!",
                marketingEmails: false,
            };

            const result = await testHelpers.callSignupEndpoint(validSignupData);
            expect(result.success).toBe(true);
            expect(result.result).toBeDefined();

            const validation = testHelpers.validateSignupResponse(result.result);
            expect(validation.isValid).toBe(true);

            console.log("✅ Isolated endpoint testing successful");
        });

        it("should test response validation in isolation", async () => {
            const mockGoodResponse = {
                id: "user123",
                sessionToken: "token123",
                languages: ["en"],
                users: [{ id: "user123", handle: "testuser" }],
            };

            const validation = testHelpers.validateSignupResponse(mockGoodResponse);
            expect(validation.isValid).toBe(true);

            const mockBadResponse = {
                id: "user123",
                // missing sessionToken, languages, users
            };

            const badValidation = testHelpers.validateSignupResponse(mockBadResponse);
            expect(badValidation.isValid).toBe(false);
            expect(badValidation.errors).toBeDefined();
            expect(badValidation.errors!.length).toBeGreaterThan(0);

            console.log("✅ Response validation testing successful");
        });

        it("should test database verification in isolation", async () => {
            const { user } = await testHelpers.createTestUser({
                name: "Database Test User",
                handle: "db_test_user",
            });

            const verification = await testHelpers.verifyUserFields(user.id, {
                name: "Database Test User",
                handle: "db_test_user",
            });

            expect(verification.isValid).toBe(true);
            expect(verification.data).toBeDefined();

            // Test with wrong expected values
            const badVerification = await testHelpers.verifyUserFields(user.id, {
                name: "Wrong Name",
            });

            expect(badVerification.isValid).toBe(false);
            expect(badVerification.errors).toBeDefined();

            console.log("✅ Database verification testing successful");
        });
    });

    describe("Edge Cases and Error Handling", () => {
        it("should handle missing required fields gracefully", async () => {
            const incompleteData = {
                name: "Test User",
                // missing email and password
            };

            const result = await testHelpers.callSignupEndpoint(incompleteData);
            expect(result.success).toBe(false);
            expect(result.error).toBeDefined();
            expect(result.statusCode).toBeDefined();

            console.log("✅ Missing required fields handled gracefully");
        });

        it("should handle non-existent user ID in profile update", async () => {
            const formData = formFixtures.createProfileUpdateForm("minimal");
            const fakeUserId = "999999999999999999999999";

            const pipelineResult = await testHelpers.runProfileUpdatePipeline(
                formData,
                fakeUserId,
                (data, userId) => shapeFixtures.profileUpdateFormToApi(data, userId),
                (data) => validationFixtures.validateProfileUpdate(data),
            );

            expect(pipelineResult.endpointResult.success).toBe(false);

            console.log("✅ Non-existent user ID handled gracefully");
        });
    });
});
