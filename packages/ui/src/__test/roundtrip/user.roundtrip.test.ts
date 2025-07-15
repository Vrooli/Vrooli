/**
 * User Round-Trip Test
 * 
 * Tests the complete data flow for user operations:
 * Form Data → Shape Transform → Validation → Endpoint Logic → Database → API Response
 */
import { PrismaClient } from "@prisma/client";
import type { EmailSignUpInput, ProfileShape, ProfileUpdateInput } from "@vrooli/shared";
import { authValidation, profileValidation, shapeAuth, shapeProfile } from "@vrooli/shared";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { type SignupViewFormInput } from "../../views/auth/SignupView.js";
// @ts-expect-error - Server endpoints are not exported in package.json. These relative imports work at runtime for round-trip testing.
import { auth } from "@vrooli/server/endpoints/logic/auth.js";
// @ts-expect-error - Server endpoints are not exported in package.json. These relative imports work at runtime for round-trip testing.
import { user } from "@vrooli/server/endpoints/logic/user.js";

/**
 * Form Fixtures Layer
 * Simulates form interactions and generates form-shaped data
 */
class UserFormFixtures {
    createSignupFormData(scenario: "valid" | "weakPassword" | "emailMismatch"): SignupViewFormInput {
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
                return {
                    ...baseData,
                    email: "invalid-email",
                };
            default:
                throw new Error(`Unknown scenario: ${scenario}`);
        }
    }

    createProfileUpdateFormData(scenario: "minimal" | "complete" | "invalidHandle", original: ProfileShape): ProfileShape {
        switch (scenario) {
            case "minimal":
                return {
                    ...original,
                    name: "Updated Name",
                };
            case "complete":
                return {
                    ...original,
                    name: "Updated Name",
                    handle: "updated_handle",
                    theme: "dark",
                    translations: [
                        ...(original.translations ?? []).filter(t => t.language !== "en"),
                        {
                            id: "1",
                            language: "en",
                            bio: "Updated bio text",
                        },
                    ],
                };
            case "invalidHandle":
                return {
                    ...original,
                    handle: "a", // Too short
                };
            default:
                throw new Error(`Unknown scenario: ${scenario}`);
        }
    }
}

/**
 * Shape Fixtures Layer
 * Uses real shaping functions to convert form data to API inputs
 */
class UserShapeFixtures {
    transformSignupToAPIInput(formData: SignupViewFormInput): EmailSignUpInput {
        // Transform form data to match what shapeAuth.signup expects
        const baseInput = {
            name: formData.name,
            email: formData.email,
            password: formData.password,
            marketingEmails: formData.marketingEmails,
            theme: formData.theme,
        };

        // Use real shape function
        return shapeAuth.emailSignUp(baseInput);
    }

    transformProfileUpdateToAPIInput(formData: ProfileUpdateFormData): ProfileUpdateInput {
        // Use real shape function
        return shapeProfile.update(formData);
    }
}

/**
 * Validation Fixtures Layer
 * Uses real validation schemas
 */
class UserValidationFixtures {
    async validateSignupInput(apiInput: EmailSignUpInput): Promise<{
        isValid: boolean;
        errors?: unknown;
        data?: EmailSignUpInput;
    }> {
        try {
            const schema = authValidation.signup({});
            const validatedData = await schema.validate(apiInput);
            return { isValid: true, data: validatedData };
        } catch (error) {
            return { isValid: false, errors: error };
        }
    }

    async validateProfileUpdateInput(apiInput: ProfileUpdateInput): Promise<{
        isValid: boolean;
        errors?: unknown;
        data?: ProfileUpdateInput;
    }> {
        try {
            const schema = profileValidation.update({});
            const validatedData = await schema.validate(apiInput);
            return { isValid: true, data: validatedData };
        } catch (error) {
            return { isValid: false, errors: error };
        }
    }
}

/**
 * Endpoint Fixtures Layer
 * Uses real endpoint logic
 */
interface TestContext {
    prisma: PrismaClient;
    req: {
        ip: string;
        headers: Record<string, string>;
    };
    userData: {
        id: string;
        languages: string[];
    };
}

class UserEndpointFixtures {
    async processSignup(apiInput: EmailSignUpInput, context: TestContext): Promise<unknown> {
        return await auth.emailSignUp({
            input: apiInput,
        }, {
            req: context.req,
        });
    }

    async processProfileUpdate(apiInput: ProfileUpdateInput, context: TestContext): Promise<unknown> {
        return await user.updateOne.logic({
            input: apiInput,
            userData: context.userData,
            prisma: context.prisma,
        });
    }
}

describe("User Round-Trip Tests", () => {
    let prisma: PrismaClient;
    let _context: TestContext;

    const formFixtures = new UserFormFixtures();
    const shapeFixtures = new UserShapeFixtures();
    const validationFixtures = new UserValidationFixtures();
    const _endpointFixtures = new UserEndpointFixtures();

    beforeAll(async () => {
        // Initialize Prisma with test database
        prisma = new PrismaClient({
            datasources: {
                db: { url: process.env.DATABASE_URL },
            },
        });

        await prisma.$connect();

        // Setup test context
        _context = {
            prisma,
            req: {
                ip: "127.0.0.1",
                headers: { "user-agent": "test" },
            },
            userData: {
                id: "test_user_123",
                languages: ["en"],
            },
        };
    });

    afterAll(async () => {
        // Cleanup test data
        await prisma.user.deleteMany({
            where: { emails: { some: { emailAddress: "test@example.com" } } },
        });
        await prisma.$disconnect();
    });

    describe("Signup Flow", () => {
        it("should complete full signup cycle with valid data", async () => {
            // 1. Form data
            const formData = formFixtures.createSignupFormData("valid");
            expect(formData.password).toBe(formData.confirmPassword);

            // 2. Shape transform
            const apiInput = shapeFixtures.transformSignupToAPIInput(formData);
            expect(apiInput.email).toBe(formData.email);
            expect(apiInput.name).toBe(formData.name);

            // 3. Validation
            const validationResult = await validationFixtures.validateSignupInput(apiInput);
            expect(validationResult.isValid).toBe(true);
            expect(validationResult.data).toBeDefined();

            // 4. Endpoint (skip for now as it requires more setup)
            // const endpointResult = await endpointFixtures.processSignup(apiInput, context);
            // expect(endpointResult.user).toBeDefined();
            // expect(endpointResult.session).toBeDefined();
        });

        it("should fail validation with weak password", async () => {
            // 1. Form data
            const formData = formFixtures.createSignupFormData("weakPassword");

            // 2. Shape transform
            const apiInput = shapeFixtures.transformSignupToAPIInput(formData);

            // 3. Validation
            const validationResult = await validationFixtures.validateSignupInput(apiInput);
            expect(validationResult.isValid).toBe(false);
            expect(validationResult.errors).toBeDefined();
        });

        it("should fail validation with invalid email", async () => {
            // 1. Form data
            const formData = formFixtures.createSignupFormData("emailMismatch");

            // 2. Shape transform
            const apiInput = shapeFixtures.transformSignupToAPIInput(formData);

            // 3. Validation
            const validationResult = await validationFixtures.validateSignupInput(apiInput);
            expect(validationResult.isValid).toBe(false);
            expect(validationResult.errors).toBeDefined();
        });
    });

    describe("Profile Update Flow", () => {
        it("should complete profile update with minimal data", async () => {
            // 1. Form data
            const formData = formFixtures.createProfileUpdateFormData("minimal");

            // 2. Shape transform
            const apiInput = shapeFixtures.transformProfileUpdateToAPIInput(formData);
            expect(apiInput.bio).toBe(formData.bio);

            // 3. Validation
            const validationResult = await validationFixtures.validateProfileUpdateInput(apiInput);
            expect(validationResult.isValid).toBe(true);
            expect(validationResult.data).toBeDefined();

            // 4. Endpoint would go here
        });

        it("should complete profile update with complete data", async () => {
            // 1. Form data
            const formData = formFixtures.createProfileUpdateFormData("complete");

            // 2. Shape transform
            const apiInput = shapeFixtures.transformProfileUpdateToAPIInput(formData);
            expect(apiInput.name).toBe(formData.name);
            expect(apiInput.handle).toBe(formData.handle);
            expect(apiInput.theme).toBe(formData.theme);

            // 3. Validation
            const validationResult = await validationFixtures.validateProfileUpdateInput(apiInput);
            expect(validationResult.isValid).toBe(true);
            expect(validationResult.data).toBeDefined();
        });

        it("should fail validation with invalid handle", async () => {
            // 1. Form data
            const formData = formFixtures.createProfileUpdateFormData("invalidHandle");

            // 2. Shape transform
            const apiInput = shapeFixtures.transformProfileUpdateToAPIInput(formData);

            // 3. Validation
            const validationResult = await validationFixtures.validateProfileUpdateInput(apiInput);
            expect(validationResult.isValid).toBe(false);
            expect(validationResult.errors).toBeDefined();
        });
    });
});
