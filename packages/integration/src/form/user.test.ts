/**
 * User Form Integration Tests - Improved Migration
 * 
 * This demonstrates the migration from the original user.test.ts which
 * called endpoint logic directly to true round-trip testing through the 
 * form-testing infrastructure with real API calls.
 * 
 * Note: For comprehensive user testing, see user-migrated.test.ts
 * This file shows the direct migration of specific scenarios from the original test.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { 
    userSignupFormIntegrationFactory,
    userProfileFormIntegrationFactory,
    createAuthenticatedUserSession,
    verifyUserAuthState
} from "../examples/UserFormIntegration.js";
import { enhancedTestUtils } from "../fixtures/index.js";

describe("User Form Integration Tests - Improved from Original", () => {
    describe("Email Signup Workflow (Original Test Scenarios)", () => {
        it("should handle valid signup (original 'valid' scenario)", async () => {
            // This replicates the original test's "valid" scenario with true round-trip testing
            const result = await userSignupFormIntegrationFactory.testRoundTripSubmission("minimal", {
                isCreate: true,
                validateConsistency: true,
            });

            // Verify complete success (enhanced from original assertions)
            expect(result.success).toBe(true);
            expect(result.errors).toHaveLength(0);
            
            // Verify API response structure (matching original expectations)
            expect(result.apiResult).toBeDefined();
            expect(result.apiResult?.sessionToken).toBeDefined();
            expect(result.apiResult?.users).toBeDefined();
            expect(result.apiResult?.users[0]?.name).toBe("Test User");
            expect(result.apiResult?.users[0]?.id).toBeDefined();
            
            // Verify database persistence (enhanced from original)
            expect(result.databaseData).toBeDefined();
            expect(result.databaseData?.name).toBe("Test User");
            expect(result.databaseData?.emails).toBeDefined();
            expect(result.databaseData?.emails[0]?.emailAddress).toBe("test@example.com");
            
            // Enhanced validation not in original test
            expect(result.consistency.overallValid).toBe(true);
            expect(result.consistency.formToApi).toBe(true);
            expect(result.consistency.apiToDatabase).toBe(true);
            
            // Verify authentication state (new capability)
            const authState = await verifyUserAuthState(result.databaseData?.id);
            expect(authState.hasPassword).toBe(true);
            expect(authState.hasVerifiedEmail).toBe(false); // Email verification needed
            expect(authState.sessionCount).toBeGreaterThan(0);
            
            // Performance validation (new capability)
            expect(result.timing.total).toBeLessThan(5000);
            expect(result.timing.apiCall).toBeLessThan(3000);
        });

        it("should reject weak password (original 'weakPassword' scenario)", async () => {
            // This replicates the original test's weak password validation
            const result = await userSignupFormIntegrationFactory.testRoundTripSubmission("weakPassword", {
                isCreate: true,
                validateConsistency: false, // We expect this to fail
            });

            // Verify failure as expected (matching original expectations)
            expect(result.success).toBe(false);
            expect(result.errors.length).toBeGreaterThan(0);
            expect(result.apiResult).toBeNull();
            expect(result.databaseData).toBeNull();
            
            // Enhanced error validation (not in original)
            expect(result.errors.some(error => 
                typeof error === 'string' && error.toLowerCase().includes('password')
            )).toBe(true);
        });

        it("should handle email validation (improved from original)", async () => {
            // Original test had 'emailMismatch' but this tests invalid email format
            const result = await userSignupFormIntegrationFactory.testRoundTripSubmission("invalidEmail", {
                isCreate: true,
                validateConsistency: false, // We expect this to fail
            });

            expect(result.success).toBe(false);
            expect(result.errors.length).toBeGreaterThan(0);
            expect(result.apiResult).toBeNull();
            expect(result.databaseData).toBeNull();
        });

        it("should handle password mismatch (enhanced scenario)", async () => {
            // Enhanced version of original email mismatch concept
            const result = await userSignupFormIntegrationFactory.testRoundTripSubmission("passwordMismatch", {
                isCreate: true,
                validateConsistency: false, // We expect this to fail
            });

            expect(result.success).toBe(false);
            expect(result.errors.length).toBeGreaterThan(0);
            expect(result.apiResult).toBeNull();
            expect(result.databaseData).toBeNull();
        });

        it("should require terms agreement (new validation)", async () => {
            // Additional validation not in original test
            const result = await userSignupFormIntegrationFactory.testRoundTripSubmission("noTermsAgreement", {
                isCreate: true,
                validateConsistency: false, // We expect this to fail
            });

            expect(result.success).toBe(false);
            expect(result.errors.length).toBeGreaterThan(0);
            expect(result.apiResult).toBeNull();
            expect(result.databaseData).toBeNull();
        });
    });

    describe("Profile Update Workflow (Original Test Scenarios)", () => {
        let testUser: any;
        let sessionData: any;

        beforeEach(async () => {
            // Create authenticated user for profile updates
            const result = await createSimpleTestUser();
            testUser = result.user;
            sessionData = await createAuthenticatedUserSession(testUser.id);
        });

        it("should handle minimal profile update (original scenario)", async () => {
            // This replicates original profile update testing
            const result = await userProfileFormIntegrationFactory.testRoundTripSubmission("minimal", {
                isCreate: false,
                validateConsistency: true,
                contextData: {
                    userId: testUser.id,
                    sessionToken: sessionData.sessionToken,
                },
            });

            // Verify success (matching original expectations)
            expect(result.success).toBe(true);
            expect(result.errors).toHaveLength(0);
            
            // Verify API response
            expect(result.apiResult).toBeDefined();
            expect(result.apiResult?.bio).toBe("Updated bio text");
            expect(result.apiResult?.id).toBe(testUser.id);
            
            // Verify database persistence
            expect(result.databaseData?.id).toBe(testUser.id);
            expect(result.databaseData?.bio).toBe("Updated bio text");
            
            // Enhanced validation not in original test
            expect(result.consistency.overallValid).toBe(true);
            expect(result.consistency.formToApi).toBe(true);
            expect(result.consistency.apiToDatabase).toBe(true);
        });

        it("should handle complete profile update (enhanced scenario)", async () => {
            // Enhanced version covering more fields than original
            const result = await userProfileFormIntegrationFactory.testRoundTripSubmission("complete", {
                isCreate: false,
                validateConsistency: true,
                contextData: {
                    userId: testUser.id,
                    sessionToken: sessionData.sessionToken,
                },
            });

            expect(result.success).toBe(true);
            
            // Verify all fields were updated
            expect(result.apiResult?.name).toBe("Updated Name");
            expect(result.apiResult?.handle).toBe("updated_handle");
            expect(result.apiResult?.theme).toBe("dark");
            expect(result.apiResult?.language).toBe("es");
            expect(result.apiResult?.bio).toContain("developer interested in AI");
            
            // Verify database persistence
            expect(result.databaseData?.name).toBe("Updated Name");
            expect(result.databaseData?.handle).toBe("updated_handle");
            expect(result.databaseData?.theme).toBe("dark");
            
            expect(result.consistency.overallValid).toBe(true);
        });

        it("should handle privacy settings update (new scenario)", async () => {
            // Privacy settings not covered in original test
            const result = await userProfileFormIntegrationFactory.testRoundTripSubmission("privacyUpdate", {
                isCreate: false,
                validateConsistency: true,
                contextData: {
                    userId: testUser.id,
                    sessionToken: sessionData.sessionToken,
                },
            });

            expect(result.success).toBe(true);
            expect(result.apiResult?.isPrivate).toBe(true);
            expect(result.databaseData?.isPrivate).toBe(true);
            expect(result.consistency.overallValid).toBe(true);
        });

        it("should reject invalid handle (validation scenario)", async () => {
            // Handle validation not in original test
            const result = await userProfileFormIntegrationFactory.testRoundTripSubmission("invalidHandle", {
                isCreate: false,
                validateConsistency: false, // We expect this to fail
                contextData: {
                    userId: testUser.id,
                    sessionToken: sessionData.sessionToken,
                },
            });

            expect(result.success).toBe(false);
            expect(result.errors.length).toBeGreaterThan(0);
        });
    });

    describe("Performance Testing (New Capability)", () => {
        it("should perform well under signup load", async () => {
            // Performance testing not in original
            const performanceResult = await userSignupFormIntegrationFactory.testFormPerformance("minimal", {
                iterations: 3,
                concurrency: 2,
                isCreate: true,
            });

            expect(performanceResult.successRate).toBeGreaterThan(0.8);
            expect(performanceResult.averageTime).toBeLessThan(4000);
            expect(performanceResult.errors.length).toBeLessThan(2);
        });

        it("should perform well under profile update load", async () => {
            // Create user first
            const user = await createSimpleTestUser();
            const session = await createAuthenticatedUserSession(user.user.id);
            
            const performanceResult = await userProfileFormIntegrationFactory.testFormPerformance("minimal", {
                iterations: 3,
                concurrency: 2,
                isCreate: false,
                contextData: {
                    userId: user.user.id,
                    sessionToken: session.sessionToken,
                },
            });

            expect(performanceResult.successRate).toBeGreaterThan(0.8);
            expect(performanceResult.averageTime).toBeLessThan(3000);
        });
    });
});

/**
 * Migration Demonstration: Original vs Improved Approach
 */
describe("User Testing Migration Comparison", () => {
    it("should demonstrate the improvements over the original test", async () => {
        // ❌ ORIGINAL APPROACH (from user.test.ts):
        // - Called endpoint logic directly: auth.emailSignUp.logic()
        // - Used mockLoggedOutSession() and mockAuthenticatedSession()
        // - Bypassed HTTP layer completely
        // - Manual validation with emailSignUpValidation.validate()
        // - Custom type definitions duplicating shared types
        // - No performance metrics
        // - Limited error scenario coverage
        //
        // const { req, res } = await mockLoggedOutSession();
        // const result = await auth.emailSignUp.logic({
        //     input: validatedInput,
        //     req,
        //     res,
        //     userData: null,
        //     prisma,
        // });
        
        // ✅ IMPROVED APPROACH (form-testing infrastructure):
        // - True round-trip through complete stack
        // - Real HTTP API endpoint calls
        // - Built-in authentication handling
        // - Uses actual shared types from @vrooli/shared
        // - Comprehensive data consistency validation
        // - Performance metrics and load testing
        // - Enhanced error scenario coverage
        // - Authentication state verification
        
        const result = await userSignupFormIntegrationFactory.testRoundTripSubmission("minimal", {
            isCreate: true,
            validateConsistency: true,
        });
        
        // The improved approach automatically tests:
        // 1. Form data transformation (UI → API format)
        // 2. API input validation (Yup schemas)
        // 3. HTTP endpoint processing (real API calls)
        // 4. Authentication and session management
        // 5. Business logic execution (server-side logic)
        // 6. Database persistence (real Prisma operations)
        // 7. Response formatting (database → API response)
        // 8. Data consistency validation (end-to-end integrity)
        // 9. Performance metrics (timing for each layer)
        // 10. Email and auth state management
        
        expect(result.success).toBe(true);
        expect(result.timing).toBeDefined();
        expect(result.consistency).toBeDefined();
        expect(result.apiResult).toBeDefined();
        expect(result.databaseData).toBeDefined();
        
        // All the benefits the original test provided, plus much more:
        expect(result.apiResult?.sessionToken).toBeDefined();
        expect(result.apiResult?.users[0]?.name).toBe("Test User");
        expect(result.databaseData?.emails).toBeDefined();
        expect(result.databaseData?.auths).toBeDefined();
        
        // Verify authentication was properly set up
        const authState = await verifyUserAuthState(result.databaseData?.id);
        expect(authState.hasPassword).toBe(true);
        expect(authState.sessionCount).toBeGreaterThan(0);
    });
});