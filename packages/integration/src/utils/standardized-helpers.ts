/**
 * Standardized Integration Test Helpers
 * 
 * This module provides a consistent interface for all integration tests
 * following the pattern: Form Data → Shape → Validation → Endpoint → Database → Response
 */

import { DbProvider } from "@vrooli/server";
import { generatePK } from "@vrooli/shared";
import { 
    auth,
    user,
    auth_emailSignUp,
    user_profileUpdate,
    mockLoggedOutSession, 
    mockAuthenticatedSession, 
    loggedInUserNoPremiumData, 
} from "@vrooli/server";

// Type definitions
export interface TestUser {
    id: bigint;
    publicId: string;
    name: string;
    handle: string;
    emails: Array<{
        id: bigint;
        emailAddress: string;
        verifiedAt: Date | null;
    }>;
}

export interface TestSession {
    isLoggedIn: boolean;
    languages: string[];
    users: Array<{
        id: string;
        handle: string;
        theme: string;
    }>;
}

export interface EndpointResult<T> {
    success: boolean;
    result?: T;
    error?: any;
    statusCode?: number;
}

export interface ValidationResult<T> {
    isValid: boolean;
    data?: T;
    errors?: any[];
}

/**
 * Centralized Test Helper Registry
 * Provides consistent interface for all integration test operations
 */
export class IntegrationTestHelpers {
    private prisma: any;

    constructor() {
        this.prisma = DbProvider.get();
    }

    /**
     * User Management Helpers
     */
    async createTestUser(overrides: Partial<{
        name: string;
        handle: string;
        email: string;
        isPrivate: boolean;
        theme: string;
    }> = {}): Promise<{ user: TestUser; sessionData: TestSession }> {
        const timestamp = Date.now();
        const userId = generatePK();
        
        const defaults = {
            name: "Test User",
            handle: `test-user-${timestamp}`,
            email: `test-${timestamp}@example.com`,
            isPrivate: false,
            theme: "light",
        };
        
        const userData = { ...defaults, ...overrides };
        
        const user = await this.prisma.user.create({
            data: {
                id: userId,
                publicId: String(userId),
                name: userData.name,
                handle: userData.handle,
                isPrivate: userData.isPrivate,
                theme: userData.theme,
                emails: {
                    create: {
                        id: generatePK(),
                        emailAddress: userData.email,
                        verifiedAt: new Date(),
                    },
                },
            },
            include: {
                emails: true,
            },
        });

        const sessionData: TestSession = {
            isLoggedIn: true,
            languages: ["en"],
            users: [{
                id: String(user.id),
                handle: user.handle,
                theme: userData.theme,
            }],
        };

        return { user, sessionData };
    }

    /**
     * Endpoint Calling Helpers
     */
    async callSignupEndpoint(input: any): Promise<EndpointResult<any>> {
        try {
            const { req, res } = await mockLoggedOutSession();
            
            const result = await auth.emailSignUp(
                { input },
                { req, res },
                auth_emailSignUp,
            );
            
            return { success: true, result };
        } catch (error) {
            return { 
                success: false, 
                error,
                statusCode: error.statusCode || 500,
            };
        }
    }

    async callProfileUpdateEndpoint(input: any, userId: string): Promise<EndpointResult<any>> {
        try {
            const testUser = { ...loggedInUserNoPremiumData(), id: userId };
            const { req, res } = await mockAuthenticatedSession(testUser);
            
            const result = await user.profileUpdate(
                { input },
                { req, res },
                user_profileUpdate,
            );
            
            return { success: true, result };
        } catch (error) {
            return { 
                success: false, 
                error,
                statusCode: error.statusCode || 500,
            };
        }
    }

    /**
     * Response Validation Helpers
     */
    validateSignupResponse(response: any): ValidationResult<any> {
        const errors: string[] = [];
        
        if (!response) {
            errors.push("Response is null or undefined");
            return { isValid: false, errors };
        }
        
        // Check required fields
        const requiredFields = ["id", "sessionToken", "languages", "users"];
        for (const field of requiredFields) {
            if (!response[field]) {
                errors.push(`Response missing required field: ${field}`);
            }
        }
        
        // Validate array fields
        if (response.languages && !Array.isArray(response.languages)) {
            errors.push("Response.languages should be an array");
        }
        
        if (response.users && !Array.isArray(response.users)) {
            errors.push("Response.users should be an array");
        }
        
        return {
            isValid: errors.length === 0,
            data: response,
            errors: errors.length > 0 ? errors : undefined,
        };
    }

    validateProfileUpdateResponse(response: any, expectedInput: any): ValidationResult<any> {
        const errors: string[] = [];
        
        if (!response) {
            errors.push("Response is null or undefined");
            return { isValid: false, errors };
        }
        
        if (!response.id) {
            errors.push("Response missing user ID");
        }
        
        // Validate data integrity - what we sent should be reflected
        const fieldsToCheck = ["name", "bio", "handle", "theme"];
        for (const field of fieldsToCheck) {
            if (expectedInput[field] !== undefined && response[field] !== expectedInput[field]) {
                errors.push(`${field} mismatch: expected '${expectedInput[field]}', got '${response[field]}'`);
            }
        }
        
        return {
            isValid: errors.length === 0,
            data: response,
            errors: errors.length > 0 ? errors : undefined,
        };
    }

    /**
     * Database Verification Helpers
     */
    async verifyUserExists(userId: string | bigint): Promise<TestUser | null> {
        return await this.prisma.user.findUnique({
            where: { id: typeof userId === "string" ? BigInt(userId) : userId },
            include: { emails: true },
        });
    }

    async verifyUserFields(userId: string | bigint, expectedFields: Record<string, any>): Promise<ValidationResult<TestUser>> {
        const user = await this.verifyUserExists(userId);
        const errors: string[] = [];
        
        if (!user) {
            errors.push("User not found in database");
            return { isValid: false, errors };
        }
        
        for (const [field, expectedValue] of Object.entries(expectedFields)) {
            if (user[field] !== expectedValue) {
                errors.push(`Database field ${field}: expected '${expectedValue}', got '${user[field]}'`);
            }
        }
        
        return {
            isValid: errors.length === 0,
            data: user,
            errors: errors.length > 0 ? errors : undefined,
        };
    }

    /**
     * Complete Pipeline Helpers
     * These methods run the entire Form → Shape → Validation → Endpoint → Database → Response pipeline
     */
    async runSignupPipeline(formData: any, shapeTransform: (data: any) => any, validator: (data: any) => Promise<ValidationResult<any>>): Promise<{
        formData: any;
        shapedData: any;
        validationResult: ValidationResult<any>;
        endpointResult: EndpointResult<any>;
        responseValidation: ValidationResult<any>;
        dbVerification: ValidationResult<TestUser>;
    }> {
        // 1. Shape transformation
        const shapedData = shapeTransform(formData);
        
        // 2. Validation
        const validationResult = await validator(shapedData);
        
        // 3. Endpoint call
        const endpointResult = validationResult.isValid 
            ? await this.callSignupEndpoint(validationResult.data)
            : await this.callSignupEndpoint(shapedData); // Test with invalid data too
        
        // 4. Response validation
        const responseValidation = endpointResult.success 
            ? this.validateSignupResponse(endpointResult.result)
            : { isValid: false, errors: ["Endpoint call failed"] };
        
        // 5. Database verification
        const dbVerification = endpointResult.success && responseValidation.isValid
            ? await this.verifyUserFields(endpointResult.result.users[0].id, {
                name: formData.name,
                // Add other fields as needed
            })
            : { isValid: false, errors: ["Cannot verify DB - endpoint failed"] };
        
        return {
            formData,
            shapedData,
            validationResult,
            endpointResult,
            responseValidation,
            dbVerification,
        };
    }

    async runProfileUpdatePipeline(
        formData: any, 
        userId: string,
        shapeTransform: (data: any, userId: string) => any, 
        validator: (data: any) => Promise<ValidationResult<any>>,
    ): Promise<{
        formData: any;
        shapedData: any;
        validationResult: ValidationResult<any>;
        endpointResult: EndpointResult<any>;
        responseValidation: ValidationResult<any>;
        dbVerification: ValidationResult<TestUser>;
    }> {
        // 1. Shape transformation
        const shapedData = shapeTransform(formData, userId);
        
        // 2. Validation
        const validationResult = await validator(shapedData);
        
        // 3. Endpoint call
        const endpointResult = validationResult.isValid 
            ? await this.callProfileUpdateEndpoint(validationResult.data, userId)
            : await this.callProfileUpdateEndpoint(shapedData, userId); // Test with invalid data too
        
        // 4. Response validation
        const responseValidation = endpointResult.success 
            ? this.validateProfileUpdateResponse(endpointResult.result, validationResult.data || shapedData)
            : { isValid: false, errors: ["Endpoint call failed"] };
        
        // 5. Database verification
        const dbVerification = endpointResult.success && responseValidation.isValid
            ? await this.verifyUserFields(userId, formData)
            : { isValid: false, errors: ["Cannot verify DB - endpoint failed"] };
        
        return {
            formData,
            shapedData,
            validationResult,
            endpointResult,
            responseValidation,
            dbVerification,
        };
    }

    /**
     * Test Cleanup Helpers
     */
    async cleanupTestUser(userId: string | bigint): Promise<void> {
        const id = typeof userId === "string" ? BigInt(userId) : userId;
        
        // Delete related records first
        await this.prisma.email.deleteMany({
            where: { userId: id },
        });
        
        // Delete user
        await this.prisma.user.delete({
            where: { id },
        });
    }

    /**
     * Debug and Logging Helpers
     */
    logPipelineResult(testName: string, pipelineResult: any): void {
        console.log(`\n🔍 Pipeline Result for: ${testName}`);
        console.log(`   Form Data: ${JSON.stringify(pipelineResult.formData, null, 2)}`);
        console.log(`   Shaped Data: ${JSON.stringify(pipelineResult.shapedData, null, 2)}`);
        console.log(`   Validation: ${pipelineResult.validationResult.isValid ? "✅" : "❌"}`);
        console.log(`   Endpoint: ${pipelineResult.endpointResult.success ? "✅" : "❌"}`);
        console.log(`   Response: ${pipelineResult.responseValidation.isValid ? "✅" : "❌"}`);
        console.log(`   Database: ${pipelineResult.dbVerification.isValid ? "✅" : "❌"}`);
        
        if (!pipelineResult.validationResult.isValid) {
            console.log(`   Validation Errors: ${JSON.stringify(pipelineResult.validationResult.errors)}`);
        }
        if (!pipelineResult.endpointResult.success) {
            console.log(`   Endpoint Error: ${JSON.stringify(pipelineResult.endpointResult.error)}`);
        }
        if (!pipelineResult.responseValidation.isValid) {
            console.log(`   Response Errors: ${JSON.stringify(pipelineResult.responseValidation.errors)}`);
        }
        if (!pipelineResult.dbVerification.isValid) {
            console.log(`   DB Errors: ${JSON.stringify(pipelineResult.dbVerification.errors)}`);
        }
    }
}

// Export singleton instance
export const testHelpers = new IntegrationTestHelpers();

// Re-export the simple helper for backward compatibility
export { createSimpleTestUser } from "./simple-helpers.js";
