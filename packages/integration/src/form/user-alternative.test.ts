/**
 * User Form Integration Tests - Migrated to Form-Testing Infrastructure
 * 
 * This demonstrates the migration from standardized helpers to true round-trip
 * testing through the form-testing infrastructure with real API calls.
 * Tests both signup and profile update workflows.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { 
    userSignupFormIntegrationFactory,
    userProfileFormIntegrationFactory,
    createAuthenticatedUserSession,
    verifyUserAuthState
} from "../examples/UserFormIntegration.js";
import { createSimpleTestUser } from "../utils/simple-helpers.js";

describe("User Form Integration Tests - Form-Testing Infrastructure", () => {
    describe("User Signup Workflow", () => {
        it("should complete full user signup workflow with minimal data", async () => {
            // Test creating a new user through complete round-trip
            const result = await userSignupFormIntegrationFactory.testRoundTripSubmission("minimal", {
                isCreate: true,
                validateConsistency: true,
            });

            // Verify complete success
            expect(result.success).toBe(true);
            expect(result.errors).toHaveLength(0);
            
            // Verify API response structure
            expect(result.apiResult).toBeDefined();
            expect(result.apiResult?.sessionToken).toBeDefined();
            expect(result.apiResult?.users).toBeDefined();
            expect(result.apiResult?.users[0]?.name).toBe("Test User");
            
            // Verify database persistence
            expect(result.databaseData).toBeDefined();
            expect(result.databaseData?.name).toBe("Test User");
            expect(result.databaseData?.emails).toBeDefined();
            expect(result.databaseData?.emails[0]?.emailAddress).toBe("test@example.com");
            
            // Verify data consistency across layers
            expect(result.consistency.overallValid).toBe(true);
            expect(result.consistency.formToApi).toBe(true);
            expect(result.consistency.apiToDatabase).toBe(true);
            
            // Verify performance is reasonable
            expect(result.timing.total).toBeLessThan(5000); // Less than 5 seconds
            expect(result.timing.apiCall).toBeLessThan(3000); // Less than 3 seconds for API call
            
            // Verify authentication state
            const authState = await verifyUserAuthState(result.databaseData?.id);
            expect(authState.hasPassword).toBe(true);
            expect(authState.sessionCount).toBeGreaterThan(0);
        });

        it("should handle complete signup with all optional fields", async () => {
            // Test comprehensive signup with theme and marketing preferences
            const result = await userSignupFormIntegrationFactory.testRoundTripSubmission("complete", {
                isCreate: true,
                validateConsistency: true,
            });

            expect(result.success).toBe(true);
            expect(result.apiResult?.users[0]?.theme).toBe("dark");
            expect(result.databaseData?.marketingEmails).toBe(true);
            expect(result.consistency.overallValid).toBe(true);
        });

        it("should reject weak passwords gracefully", async () => {
            // Test validation failure with weak password
            const result = await userSignupFormIntegrationFactory.testRoundTripSubmission("weakPassword", {
                isCreate: true,
                validateConsistency: false, // We expect this to fail
            });

            expect(result.success).toBe(false);
            expect(result.errors.length).toBeGreaterThan(0);
            expect(result.apiResult).toBeNull();
            expect(result.databaseData).toBeNull();
        });

        it("should reject invalid email addresses", async () => {
            // Test validation failure with invalid email
            const result = await userSignupFormIntegrationFactory.testRoundTripSubmission("invalidEmail", {
                isCreate: true,
                validateConsistency: false, // We expect this to fail
            });

            expect(result.success).toBe(false);
            expect(result.errors.length).toBeGreaterThan(0);
            expect(result.apiResult).toBeNull();
            expect(result.databaseData).toBeNull();
        });

        it("should reject password mismatches", async () => {
            // Test validation failure when passwords don't match
            const result = await userSignupFormIntegrationFactory.testRoundTripSubmission("passwordMismatch", {
                isCreate: true,
                validateConsistency: false, // We expect this to fail
            });

            expect(result.success).toBe(false);
            expect(result.errors.length).toBeGreaterThan(0);
            expect(result.apiResult).toBeNull();
            expect(result.databaseData).toBeNull();
        });

        it("should require terms agreement", async () => {
            // Test validation failure when terms are not agreed to
            const result = await userSignupFormIntegrationFactory.testRoundTripSubmission("noTermsAgreement", {
                isCreate: true,
                validateConsistency: false, // We expect this to fail
            });

            expect(result.success).toBe(false);
            expect(result.errors.length).toBeGreaterThan(0);
            expect(result.apiResult).toBeNull();
            expect(result.databaseData).toBeNull();
        });

        it("should perform well under concurrent signup load", async () => {
            // Test performance with multiple concurrent signup operations
            const performanceResult = await userSignupFormIntegrationFactory.testFormPerformance("minimal", {
                iterations: 3,
                concurrency: 2,
                isCreate: true,
            });

            expect(performanceResult.successRate).toBeGreaterThan(0.8); // 80% success rate
            expect(performanceResult.averageTime).toBeLessThan(4000); // Average under 4 seconds
            expect(performanceResult.errors.length).toBeLessThan(2); // Few errors expected
        });
    });

    describe("User Profile Update Workflow", () => {
        let testUser: any;
        let sessionData: any;

        beforeEach(async () => {
            // Create authenticated user for profile updates
            const result = await createSimpleTestUser();
            testUser = result.user;
            sessionData = await createAuthenticatedUserSession(testUser.id);
        });

        it("should complete profile update workflow with minimal data", async () => {
            // Test updating user profile through complete round-trip
            const result = await userProfileFormIntegrationFactory.testRoundTripSubmission("minimal", {
                isCreate: false,
                validateConsistency: true,
                contextData: {
                    userId: testUser.id,
                    sessionToken: sessionData.sessionToken,
                },
            });

            expect(result.success).toBe(true);
            expect(result.errors).toHaveLength(0);
            
            // Verify API response
            expect(result.apiResult).toBeDefined();
            expect(result.apiResult?.bio).toBe("Updated bio text");
            
            // Verify database persistence
            expect(result.databaseData?.id).toBe(testUser.id);
            expect(result.databaseData?.bio).toBe("Updated bio text");
            
            expect(result.consistency.overallValid).toBe(true);
        });

        it("should handle complete profile update with all fields", async () => {
            // Test comprehensive profile update
            const result = await userProfileFormIntegrationFactory.testRoundTripSubmission("complete", {
                isCreate: false,
                validateConsistency: true,
                contextData: {
                    userId: testUser.id,
                    sessionToken: sessionData.sessionToken,
                },
            });

            expect(result.success).toBe(true);
            expect(result.apiResult?.name).toBe("Updated Name");
            expect(result.apiResult?.handle).toBe("updated_handle");
            expect(result.apiResult?.theme).toBe("dark");
            expect(result.apiResult?.language).toBe("es");
            
            expect(result.databaseData?.name).toBe("Updated Name");
            expect(result.databaseData?.handle).toBe("updated_handle");
            expect(result.consistency.overallValid).toBe(true);
        });

        it("should handle privacy settings update", async () => {
            // Test updating privacy settings
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

        it("should reject invalid handle", async () => {
            // Test validation failure with empty handle
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

        it("should handle long bio content", async () => {
            // Test edge case with very long bio
            const result = await userProfileFormIntegrationFactory.testRoundTripSubmission("longBio", {
                isCreate: false,
                validateConsistency: true,
                contextData: {
                    userId: testUser.id,
                    sessionToken: sessionData.sessionToken,
                },
            });

            expect(result.success).toBe(true);
            expect(result.apiResult?.bio).toHaveLength(5000);
            expect(result.apiResult?.handle).toBe("long_bio_user");
            expect(result.databaseData?.bio).toHaveLength(5000);
            expect(result.consistency.overallValid).toBe(true);
        });

        it("should perform well under concurrent update load", async () => {
            // Test performance with multiple concurrent profile updates
            const performanceResult = await userProfileFormIntegrationFactory.testFormPerformance("minimal", {
                iterations: 3,
                concurrency: 2,
                isCreate: false,
                contextData: {
                    userId: testUser.id,
                    sessionToken: sessionData.sessionToken,
                },
            });

            expect(performanceResult.successRate).toBeGreaterThan(0.8); // 80% success rate
            expect(performanceResult.averageTime).toBeLessThan(3000); // Average under 3 seconds
        });
    });

    describe("Dynamic Test Generation", () => {
        it("should generate and run signup test cases", async () => {
            const testCases = userSignupFormIntegrationFactory.generateIntegrationTestCases();
            
            expect(testCases.length).toBeGreaterThan(0);
            
            // Run a subset of generated test cases
            for (const testCase of testCases.slice(0, 2)) {
                if (testCase.shouldSucceed) {
                    const result = await userSignupFormIntegrationFactory.testRoundTripSubmission(testCase.scenario, {
                        isCreate: testCase.isCreate,
                        validateConsistency: testCase.shouldSucceed,
                    });
                    
                    expect(result.success).toBe(testCase.shouldSucceed);
                    if (testCase.shouldSucceed) {
                        expect(result.consistency.overallValid).toBe(true);
                    }
                }
            }
        });

        it("should generate and run profile update test cases", async () => {
            // Create user first
            const user = await createSimpleTestUser();
            const session = await createAuthenticatedUserSession(user.user.id);
            
            const testCases = userProfileFormIntegrationFactory.generateIntegrationTestCases();
            
            expect(testCases.length).toBeGreaterThan(0);
            
            // Run a subset of generated test cases
            for (const testCase of testCases.slice(0, 1)) {
                const result = await userProfileFormIntegrationFactory.testRoundTripSubmission(testCase.scenario, {
                    isCreate: false,
                    validateConsistency: testCase.shouldSucceed,
                    contextData: {
                        userId: user.user.id,
                        sessionToken: session.sessionToken,
                    },
                });
                
                expect(result.success).toBe(testCase.shouldSucceed);
                if (testCase.shouldSucceed) {
                    expect(result.consistency.overallValid).toBe(true);
                }
            }
        });
    });
});

/**
 * Migration Demonstration: Standardized Helpers vs Form-Testing
 */
describe("User Testing Migration Comparison", () => {
    it("should demonstrate the improved testing approach", async () => {
        // ❌ OLD APPROACH (from user-standardized.test.ts):
        // - Used standardized helpers but still bypassed HTTP layer
        // - Called endpoint logic directly via auth.emailSignUp.logic()
        // - Manual session mocking with mockLoggedOutSession()
        // - Custom response validation
        // - No built-in performance metrics
        //
        // const result = await testHelpers.runSignupPipeline(
        //     formData,
        //     shapeTransform,
        //     validator
        // );
        
        // ✅ NEW APPROACH (form-testing infrastructure):
        // - True round-trip through complete stack
        // - Real HTTP API endpoint calls
        // - Built-in authentication handling
        // - Comprehensive data consistency validation
        // - Performance metrics and load testing
        // - Both signup and profile update workflows
        
        const result = await userSignupFormIntegrationFactory.testRoundTripSubmission("minimal", {
            isCreate: true,
            validateConsistency: true,
        });
        
        // The new approach automatically tests:
        // 1. Form data transformation (UI → API format)
        // 2. API input validation (Yup schemas)
        // 3. HTTP endpoint processing (real API calls)
        // 4. Authentication and session management
        // 5. Business logic execution (server-side logic)
        // 6. Database persistence (real Prisma operations)
        // 7. Response formatting (database → API response)
        // 8. Data consistency validation (end-to-end integrity)
        // 9. Performance metrics (timing for each layer)
        // 10. Email verification and auth state management
        
        expect(result.success).toBe(true);
        expect(result.timing).toBeDefined();
        expect(result.consistency).toBeDefined();
        expect(result.apiResult).toBeDefined();
        expect(result.databaseData).toBeDefined();
        
        // Verify user-specific features
        expect(result.apiResult?.sessionToken).toBeDefined();
        expect(result.databaseData?.emails).toBeDefined();
        
        // Verify authentication was properly set up
        const authState = await verifyUserAuthState(result.databaseData?.id);
        expect(authState.hasPassword).toBe(true);
    });
});