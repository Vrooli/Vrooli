/**
 * User Round-Trip Test
 * 
 * Tests the complete data flow for user operations:
 * Form Data → Shape Transform → Validation → Endpoint Logic → Database → API Response
 */
// Define the input types based on what we've seen in the API types
interface EmailSignUpInput {
    name: string;
    email: string;
    password: string;
    confirmPassword: string;
    marketingEmails: boolean;
}

interface ProfileUpdateInput {
    id: string;
    name?: string;
    bio?: string;
    handle?: string;
    theme?: string;
    language?: string;
}

// Import actual server endpoint logic
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { getPrisma } from "../setup/test-setup.js";
import { createSimpleTestUser } from "../utils/simple-helpers.js";
// Import from package roots
import { emailSignUpValidation, profileUpdateValidation } from "@vrooli/shared";
import { 
    auth,
    user,
    auth_emailSignUp,
    user_profileUpdate,
    mockLoggedOutSession, 
    mockAuthenticatedSession, 
    loggedInUserNoPremiumData, 
} from "@vrooli/server";

// Form input type to match the UI SignupView form structure
interface SignupViewFormInput {
    name: string;
    email: string;
    password: string;
    confirmPassword: string;
    marketingEmails: boolean;
    theme?: string;
    agreeToTerms: boolean;
}

interface ProfileUpdateFormData {
    name?: string;
    bio?: string;
    handle?: string;
    theme?: string;
    language?: string;
}

/**
 * Form Fixtures Layer
 * Simulates form interactions and generates form-shaped data
 */
class UserFormFixtures {
    createSignupViewFormData(scenario: "valid" | "weakPassword" | "emailMismatch"): SignupViewFormInput {
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

    createProfileUpdateFormData(scenario: "minimal" | "complete" | "invalidHandle"): ProfileUpdateFormData {
        switch (scenario) {
            case "minimal":
                return {
                    bio: "Updated bio text",
                };
            case "complete":
                return {
                    name: "Updated Name",
                    bio: "I am a developer interested in AI and automation.",
                    handle: "updated_handle",
                    theme: "dark",
                    language: "es",
                };
            case "invalidHandle":
                return {
                    handle: "a", // Too short
                };
            default:
                throw new Error(`Unknown scenario: ${scenario}`);
        }
    }
}

/**
 * Shape Fixtures Layer
 * Converts form data to API inputs (simplified without shape functions)
 */
class UserShapeFixtures {
    transformSignupToAPIInput(formData: SignupViewFormInput): EmailSignUpInput {
        // Transform form data to match API input structure
        return {
            name: formData.name,
            email: formData.email,
            password: formData.password,
            confirmPassword: formData.confirmPassword,
            marketingEmails: formData.marketingEmails,
        };
    }

    transformProfileUpdateToAPIInput(formData: ProfileUpdateFormData, userId: string): ProfileUpdateInput {
        // Transform form data to match API input structure
        return {
            id: userId,
            ...formData,
        };
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
            // Use actual shared validation logic
            const validatedData = await emailSignUpValidation.validateSync(apiInput);
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
            // Use actual shared validation logic
            const validatedData = await profileUpdateValidation.validateSync(apiInput);
            return { isValid: true, data: validatedData };
        } catch (error) {
            return { isValid: false, errors: error };
        }
    }
}

/**
 * Endpoint Fixtures Layer
 * Tests actual server endpoints with proper authentication
 */
class UserEndpointFixtures {
    async processSignup(input: EmailSignUpInput): Promise<{
        success: boolean;
        result?: any;
        error?: any;
    }> {
        try {
            // Create unauthenticated session for signup
            const { req, res } = await mockLoggedOutSession();
            
            // Call actual signup endpoint
            const result = await auth.emailSignUp(
                { input },
                { req, res },
                auth_emailSignUp,
            );
            
            return { success: true, result };
        } catch (error) {
            return { success: false, error };
        }
    }
    
    async processProfileUpdate(input: ProfileUpdateInput, userId: string): Promise<{
        success: boolean;
        result?: any;
        error?: any;
    }> {
        try {
            // Create authenticated session for profile update
            const testUser = { ...loggedInUserNoPremiumData(), id: userId };
            const { req, res } = await mockAuthenticatedSession(testUser);
            
            // Call actual profile update endpoint
            const result = await user.profileUpdate(
                { input },
                { req, res },
                user_profileUpdate,
            );
            
            return { success: true, result };
        } catch (error) {
            return { success: false, error };
        }
    }
    
    /**
     * Response validation helper
     */
    validateSignupResponse(response: any): {
        isValid: boolean;
        errors?: string[];
    } {
        const errors: string[] = [];
        
        // Validate response structure
        if (!response) {
            errors.push("Response is null or undefined");
        }
        
        if (!response.id) {
            errors.push("Response missing user ID");
        }
        
        if (!response.sessionToken) {
            errors.push("Response missing session token");
        }
        
        if (!response.languages || !Array.isArray(response.languages)) {
            errors.push("Response missing or invalid languages array");
        }
        
        if (!response.users || !Array.isArray(response.users)) {
            errors.push("Response missing or invalid users array");
        }
        
        return {
            isValid: errors.length === 0,
            errors: errors.length > 0 ? errors : undefined,
        };
    }
    
    validateProfileUpdateResponse(response: any, expectedInput: ProfileUpdateInput): {
        isValid: boolean;
        errors?: string[];
    } {
        const errors: string[] = [];
        
        // Validate response structure
        if (!response) {
            errors.push("Response is null or undefined");
        }
        
        if (!response.id) {
            errors.push("Response missing user ID");
        }
        
        // Validate data integrity - what we sent should be reflected
        if (expectedInput.name && response.name !== expectedInput.name) {
            errors.push(`Name mismatch: expected '${expectedInput.name}', got '${response.name}'`);
        }
        
        if (expectedInput.bio && response.bio !== expectedInput.bio) {
            errors.push(`Bio mismatch: expected '${expectedInput.bio}', got '${response.bio}'`);
        }
        
        if (expectedInput.handle && response.handle !== expectedInput.handle) {
            errors.push(`Handle mismatch: expected '${expectedInput.handle}', got '${response.handle}'`);
        }
        
        if (expectedInput.theme && response.theme !== expectedInput.theme) {
            errors.push(`Theme mismatch: expected '${expectedInput.theme}', got '${response.theme}'`);
        }
        
        return {
            isValid: errors.length === 0,
            errors: errors.length > 0 ? errors : undefined,
        };
    }
}

describe("User Round-Trip Tests", () => {
    let prisma: any;

    const formFixtures = new UserFormFixtures();
    const shapeFixtures = new UserShapeFixtures();
    const validationFixtures = new UserValidationFixtures();
    const endpointFixtures = new UserEndpointFixtures();

    beforeAll(async () => {
        // Get Prisma instance from integration test setup
        prisma = getPrisma();
    });

    afterAll(async () => {
        // Cleanup handled by test setup
    });

    describe("Signup Flow", () => {
        it("should complete full signup cycle with valid data", async () => {
            // 1. Form data
            const formData = formFixtures.createSignupViewFormData("valid");
            expect(formData.password).toBe(formData.confirmPassword);

            // 2. Shape transform
            const apiInput = shapeFixtures.transformSignupToAPIInput(formData);
            expect(apiInput.email).toBe(formData.email);
            expect(apiInput.name).toBe(formData.name);

            // 3. Validation
            const validationResult = await validationFixtures.validateSignupInput(apiInput);
            expect(validationResult.isValid).toBe(true);
            expect(validationResult.data).toBeDefined();

            // 4. Endpoint - test actual signup endpoint
            const endpointResult = await endpointFixtures.processSignup(validationResult.data!);
            expect(endpointResult.success).toBe(true);
            expect(endpointResult.result).toBeDefined();
            
            // 5. Response validation
            const responseValidation = endpointFixtures.validateSignupResponse(endpointResult.result);
            expect(responseValidation.isValid).toBe(true);
            if (!responseValidation.isValid) {
                console.error("Response validation errors:", responseValidation.errors);
            }
            
            // 6. Database verification - ensure user was actually created
            const createdUser = await prisma.user.findUnique({
                where: { id: endpointResult.result.users[0].id },
                include: { emails: true },
            });
            expect(createdUser).toBeDefined();
            expect(createdUser.name).toBe(formData.name);
            expect(createdUser.emails[0].emailAddress).toBe(formData.email);
            
            console.log("✅ Full signup pipeline completed successfully");
        });

        it("should fail validation with weak password", async () => {
            // 1. Form data
            const formData = formFixtures.createSignupViewFormData("weakPassword");

            // 2. Shape transform
            const apiInput = shapeFixtures.transformSignupToAPIInput(formData);

            // 3. Validation should fail
            const validationResult = await validationFixtures.validateSignupInput(apiInput);
            expect(validationResult.isValid).toBe(false);
            expect(validationResult.errors).toBeDefined();
            
            // 4. Endpoint should also fail if validation is bypassed
            const endpointResult = await endpointFixtures.processSignup(apiInput);
            expect(endpointResult.success).toBe(false);
            expect(endpointResult.error).toBeDefined();
            
            console.log("✅ Weak password properly rejected at validation and endpoint levels");
        });

        it("should fail validation with invalid email", async () => {
            // 1. Form data
            const formData = formFixtures.createSignupViewFormData("emailMismatch");

            // 2. Shape transform
            const apiInput = shapeFixtures.transformSignupToAPIInput(formData);

            // 3. Validation should fail
            const validationResult = await validationFixtures.validateSignupInput(apiInput);
            expect(validationResult.isValid).toBe(false);
            expect(validationResult.errors).toBeDefined();
            
            // 4. Endpoint should also fail if validation is bypassed
            const endpointResult = await endpointFixtures.processSignup(apiInput);
            expect(endpointResult.success).toBe(false);
            expect(endpointResult.error).toBeDefined();
            
            console.log("✅ Invalid email properly rejected at validation and endpoint levels");
        });
    });

    describe("Profile Update Flow", () => {
        it("should complete profile update with minimal data", async () => {
            // 0. Setup - create a test user first
            const { user: testUser } = await createSimpleTestUser();
            
            // 1. Form data
            const formData = formFixtures.createProfileUpdateFormData("minimal");

            // 2. Shape transform
            const apiInput = shapeFixtures.transformProfileUpdateToAPIInput(formData, testUser.id.toString());
            expect(apiInput.bio).toBe(formData.bio);
            expect(apiInput.id).toBe(testUser.id.toString());

            // 3. Validation
            const validationResult = await validationFixtures.validateProfileUpdateInput(apiInput);
            expect(validationResult.isValid).toBe(true);
            expect(validationResult.data).toBeDefined();

            // 4. Endpoint - test actual profile update
            const endpointResult = await endpointFixtures.processProfileUpdate(validationResult.data!, testUser.id.toString());
            expect(endpointResult.success).toBe(true);
            expect(endpointResult.result).toBeDefined();
            
            // 5. Response validation
            const responseValidation = endpointFixtures.validateProfileUpdateResponse(endpointResult.result, validationResult.data!);
            expect(responseValidation.isValid).toBe(true);
            if (!responseValidation.isValid) {
                console.error("Response validation errors:", responseValidation.errors);
            }
            
            // 6. Database verification
            const updatedUser = await prisma.user.findUnique({
                where: { id: testUser.id },
            });
            expect(updatedUser).toBeDefined();
            expect(updatedUser.bio).toBe(formData.bio);
            
            console.log("✅ Profile update (minimal) pipeline completed successfully");
        });

        it("should complete profile update with complete data", async () => {
            // 0. Setup - create a test user first
            const { user: testUser } = await createSimpleTestUser();
            
            // 1. Form data
            const formData = formFixtures.createProfileUpdateFormData("complete");

            // 2. Shape transform
            const apiInput = shapeFixtures.transformProfileUpdateToAPIInput(formData, testUser.id.toString());
            expect(apiInput.name).toBe(formData.name);
            expect(apiInput.handle).toBe(formData.handle);
            expect(apiInput.theme).toBe(formData.theme);
            expect(apiInput.id).toBe(testUser.id.toString());

            // 3. Validation
            const validationResult = await validationFixtures.validateProfileUpdateInput(apiInput);
            expect(validationResult.isValid).toBe(true);
            expect(validationResult.data).toBeDefined();
            
            // 4. Endpoint - test actual profile update
            const endpointResult = await endpointFixtures.processProfileUpdate(validationResult.data!, testUser.id.toString());
            expect(endpointResult.success).toBe(true);
            expect(endpointResult.result).toBeDefined();
            
            // 5. Response validation
            const responseValidation = endpointFixtures.validateProfileUpdateResponse(endpointResult.result, validationResult.data!);
            expect(responseValidation.isValid).toBe(true);
            if (!responseValidation.isValid) {
                console.error("Response validation errors:", responseValidation.errors);
            }
            
            // 6. Database verification - check all updated fields
            const updatedUser = await prisma.user.findUnique({
                where: { id: testUser.id },
            });
            expect(updatedUser).toBeDefined();
            expect(updatedUser.name).toBe(formData.name);
            expect(updatedUser.handle).toBe(formData.handle);
            expect(updatedUser.theme).toBe(formData.theme);
            
            console.log("✅ Profile update (complete) pipeline completed successfully");
        });

        it("should fail validation with invalid handle", async () => {
            // 0. Setup - create a test user first
            const { user: testUser } = await createSimpleTestUser();
            
            // 1. Form data
            const formData = formFixtures.createProfileUpdateFormData("invalidHandle");

            // 2. Shape transform
            const apiInput = shapeFixtures.transformProfileUpdateToAPIInput(formData, testUser.id.toString());

            // 3. Validation should fail
            const validationResult = await validationFixtures.validateProfileUpdateInput(apiInput);
            expect(validationResult.isValid).toBe(false);
            expect(validationResult.errors).toBeDefined();
            
            // 4. Endpoint should also fail if validation is bypassed
            const endpointResult = await endpointFixtures.processProfileUpdate(apiInput, testUser.id.toString());
            expect(endpointResult.success).toBe(false);
            expect(endpointResult.error).toBeDefined();
            
            console.log("✅ Invalid handle properly rejected at validation and endpoint levels");
        });
    });

    describe("Integration with Test Helpers", () => {
        it("should work with createSimpleTestUser", async () => {
            // Test that our helper functions work properly
            const { user, sessionData } = await createSimpleTestUser();
            
            expect(user).toBeDefined();
            expect(user.id).toBeDefined();
            expect(user.handle).toBeDefined();
            expect(user.emails).toBeDefined();
            expect(user.emails.length).toBeGreaterThan(0);
            
            expect(sessionData).toBeDefined();
            expect(sessionData.isLoggedIn).toBe(true);
            expect(sessionData.users).toBeDefined();
            expect(sessionData.users.length).toBeGreaterThan(0);
        });
    });
});
