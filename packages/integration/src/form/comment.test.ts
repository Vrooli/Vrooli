/**
 * Comment Form Integration Tests
 * 
 * Tests the complete comment submission workflow from form data through
 * API validation to database persistence using the integration testing engine.
 */

import { beforeEach, describe, expect, it } from "vitest";
import {
    commentFormIntegrationFactory,
    createTestCommentTarget,
} from "../integration/CommentIntegrationConfig.js";
import { createSimpleTestUser } from "../utils/simple-helpers.js";

describe("Comment Round-Trip Tests - Form-Testing Infrastructure", () => {
    let testUser: any;
    let testProjectId: string;
    let testRoutineId: string;

    beforeEach(async () => {
        // Create test user
        const result = await createSimpleTestUser();
        testUser = result.user;

        // Create test targets for comments
        testProjectId = await createTestCommentTarget("Project");
        testRoutineId = await createTestCommentTarget("Routine");
    });

    it("should complete full comment creation workflow with minimal data", async () => {
        // Test creating a simple comment through complete round-trip
        const result = await commentFormIntegrationFactory.testRoundTripSubmission("valid.minimal", {
            isCreate: true,
            validateConsistency: true,
            contextData: {
                userId: testUser.id,
                projectId: testProjectId,
            },
        });

        // Verify complete success
        expect(result.success).toBe(true);
        expect(result.errors).toHaveLength(0);

        // Verify API response structure
        expect(result.dataFlow.apiResponse).toBeDefined();
        expect(result.dataFlow.apiResponse?.translations).toBeDefined();
        expect(result.dataFlow.apiResponse?.translations[0]?.text).toContain("test comment");

        // Verify database persistence
        expect(result.dataFlow.databaseData).toBeDefined();
        expect(result.dataFlow.databaseData?.id).toBe(result.dataFlow.apiResponse?.id);
        expect(result.dataFlow.databaseData?.translations).toBeDefined();

        // Verify data consistency across layers
        expect(result.consistency.overallValid).toBe(true);
        expect(result.consistency.formToApi).toBe(true);
        expect(result.consistency.apiToDatabase).toBe(true);

        // Verify performance is reasonable
        expect(result.timing.total).toBeLessThan(5000); // Less than 5 seconds
        expect(result.timing.apiCall).toBeLessThan(2000); // Less than 2 seconds for API call
    });

    it("should handle complete comment with markdown formatting", async () => {
        // Test comprehensive comment with markdown content
        const result = await commentFormIntegrationFactory.testRoundTripSubmission("valid.complete", {
            isCreate: true,
            validateConsistency: true,
            contextData: {
                userId: testUser.id,
                routineId: testRoutineId,
            },
        });

        expect(result.success).toBe(true);
        expect(result.dataFlow.apiResponse?.translations[0]?.text).toContain("comprehensive");
        expect(result.dataFlow.databaseData?.translations).toBeDefined();
        expect(result.dataFlow.databaseData?.commentedOn).toBeDefined();
        expect(result.consistency.overallValid).toBe(true);
    });

    it("should handle edge case with maximum length text", async () => {
        // Test edge case with very long comment text
        const result = await commentFormIntegrationFactory.testRoundTripSubmission("edge.maxValues", {
            isCreate: true,
            validateConsistency: true,
            contextData: {
                userId: testUser.id,
                projectId: testProjectId,
            },
        });

        expect(result.success).toBe(true);
        expect(result.dataFlow.apiResponse?.translations[0]?.text).toHaveLength(32768); // Maximum length
        expect(result.dataFlow.databaseData?.translations[0]?.text).toHaveLength(32768);
        expect(result.consistency.overallValid).toBe(true);
    });

    it("should reject invalid comment data gracefully", async () => {
        // Test validation failure with empty comment text
        const result = await commentFormIntegrationFactory.testRoundTripSubmission("invalid.missingRequired", {
            isCreate: true,
            validateConsistency: false, // We expect this to fail
        });

        expect(result.success).toBe(false);
        expect(result.errors.length).toBeGreaterThan(0);
        expect(result.dataFlow.apiResponse).toBeNull();
        expect(result.dataFlow.databaseData).toBeNull();
    });

    it("should perform well under concurrent load", async () => {
        // Test performance with multiple concurrent comment operations
        const performanceResult = await commentFormIntegrationFactory.testFormPerformance("valid.minimal", {
            iterations: 3,
            concurrency: 2,
            isCreate: true,
            contextData: {
                userId: testUser.id,
                projectId: testProjectId,
            },
        });

        expect(performanceResult.successRate).toBeGreaterThan(0.8); // 80% success rate
        expect(performanceResult.averageTime).toBeLessThan(3000); // Average under 3 seconds
        expect(performanceResult.errors.length).toBeLessThan(2); // Few errors expected
    });

    it("should complete update workflow for existing comments", async () => {
        // First create a comment
        const createResult = await commentFormIntegrationFactory.testRoundTripSubmission("valid.minimal", {
            isCreate: true,
            validateConsistency: true,
            contextData: {
                userId: testUser.id,
                projectId: testProjectId,
            },
        });

        expect(createResult.success).toBe(true);
        const commentId = createResult.dataFlow.apiResponse?.id;
        expect(commentId).toBeDefined();

        // Then update it
        const updateResult = await commentFormIntegrationFactory.testRoundTripSubmission("valid.complete", {
            isCreate: false,
            validateConsistency: true,
            contextData: {
                userId: testUser.id,
                commentId,
            },
        });

        expect(updateResult.success).toBe(true);
        expect(updateResult.dataFlow.apiResponse?.id).toBe(commentId);
        expect(updateResult.dataFlow.databaseData?.id).toBe(commentId);
        expect(updateResult.consistency.overallValid).toBe(true);
    });

    it("should generate and run all test cases dynamically", async () => {
        // Use the dynamic test case generation
        const testCases = commentFormIntegrationFactory.generateIntegrationTestCases();

        expect(testCases.length).toBeGreaterThan(0);

        // Run a subset of generated test cases
        for (const testCase of testCases.slice(0, 2)) {
            const result = await commentFormIntegrationFactory.testRoundTripSubmission(testCase.scenario, {
                isCreate: testCase.isCreate,
                validateConsistency: testCase.shouldSucceed,
                contextData: {
                    userId: testUser.id,
                    projectId: testProjectId,
                },
            });

            expect(result.success).toBe(testCase.shouldSucceed);

            if (testCase.shouldSucceed) {
                expect(result.consistency.overallValid).toBe(true);
            }
        }
    });
});

/**
 * Migration Demonstration: Old vs New Approach
 */
describe("Comment Testing Migration Comparison", () => {
    it("should demonstrate the improved testing approach", async () => {
        // ❌ OLD APPROACH (from original comment.test.ts):
        // - Direct endpoint logic calling
        // - No HTTP layer testing
        // - Custom validation setup
        // 
        // const result = await comment.createOne.logic({
        //     input: apiInput,
        //     userData: context.userData,
        //     prisma: context.prisma,
        // });

        // ✅ NEW APPROACH (form-testing infrastructure):
        // - True round-trip through complete stack
        // - Real API endpoint calls
        // - Built-in performance metrics
        // - Comprehensive data consistency validation

        const result = await commentFormIntegrationFactory.testRoundTripSubmission("valid.minimal", {
            isCreate: true,
            validateConsistency: true,
        });

        // The new approach automatically tests:
        // 1. Form data transformation (UI → API format)
        // 2. API input validation (Yup schemas)
        // 3. HTTP endpoint processing (real API calls)
        // 4. Business logic execution (server-side logic)
        // 5. Database persistence (real Prisma operations)
        // 6. Response formatting (database → API response)
        // 7. Data consistency validation (end-to-end integrity)
        // 8. Performance metrics (timing for each layer)

        expect(result.success).toBe(true);
        expect(result.timing).toBeDefined();
        expect(result.consistency).toBeDefined();
        expect(result.dataFlow.apiResponse).toBeDefined();
        expect(result.dataFlow.databaseData).toBeDefined();
    });
});
