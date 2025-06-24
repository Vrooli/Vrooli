/**
 * Bookmark Form Integration Tests
 * 
 * Tests the complete bookmark submission workflow from form data through
 * API validation to database persistence using the integration testing engine.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { 
    bookmarkFormIntegrationFactory, 
    createTestBookmarkTarget, 
    createTestBookmarkList 
} from "../examples/BookmarkFormIntegration.js";
import { BookmarkFor } from "@vrooli/shared";
import { createSimpleTestUser } from "../utils/simple-helpers.js";

describe("Bookmark Round-Trip Tests - Improved from Original", () => {
    let testUser: any;
    let testTargetUserId: string;
    let testProjectId: string;
    let testListId: string;

    beforeEach(async () => {
        // Create test user
        const result = await createSimpleTestUser();
        testUser = result.user;
        
        // Create test targets for bookmarking
        testTargetUserId = await createTestBookmarkTarget(BookmarkFor.User);
        testProjectId = await createTestBookmarkTarget(BookmarkFor.Project);
        
        // Create test bookmark list
        testListId = await createTestBookmarkList("My Reading List", testUser.id);
    });

    it("should create a bookmark through complete round-trip (original test scenario)", async () => {
        // This replicates the original test scenario but with true round-trip testing
        // Original test: Created bookmark for user with new list
        const result = await bookmarkFormIntegrationFactory.testRoundTripSubmission("withNewList", {
            isCreate: true,
            validateConsistency: true,
            contextData: {
                userId: testUser.id,
                targetId: testProjectId,
                listLabel: "My Reading List",
            },
        });

        // Verify complete success (same expectations as original test)
        expect(result.success).toBe(true);
        expect(result.errors).toHaveLength(0);
        
        // Verify API response structure (matching original assertions)
        expect(result.apiResult).toBeDefined();
        expect(result.apiResult?.id).toBeDefined();
        expect(result.apiResult?.bookmarkFor).toBe(BookmarkFor.Project);
        expect(result.apiResult?.to?.id).toBe(testProjectId);
        expect(result.apiResult?.list).toBeDefined();
        expect(result.apiResult?.list?.label).toBe("My Favorite Projects");
        
        // Verify database persistence (enhanced from original)
        expect(result.databaseData).toBeDefined();
        expect(result.databaseData?.id).toBe(result.apiResult?.id);
        expect(result.databaseData?.list?.label).toBe("My Favorite Projects");
        expect(result.databaseData?.to?.id).toBe(testProjectId);
        
        // Enhanced validation not in original test
        expect(result.consistency.overallValid).toBe(true);
        expect(result.consistency.formToApi).toBe(true);
        expect(result.consistency.apiToDatabase).toBe(true);
        
        // Performance validation (new capability)
        expect(result.timing.total).toBeLessThan(5000);
        expect(result.timing.apiCall).toBeLessThan(2000);
    });

    it("should create bookmark with existing list (additional scenario)", async () => {
        // Test using existing list instead of creating new one
        const result = await bookmarkFormIntegrationFactory.testRoundTripSubmission("minimal", {
            isCreate: true,
            validateConsistency: true,
            contextData: {
                userId: testUser.id,
                targetId: testTargetUserId,
                listId: testListId,
            },
        });

        expect(result.success).toBe(true);
        expect(result.apiResult?.bookmarkFor).toBe(BookmarkFor.User);
        expect(result.apiResult?.to?.id).toBe(testTargetUserId);
        expect(result.apiResult?.list?.id).toBe(testListId);
        expect(result.databaseData?.list?.id).toBe(testListId);
        expect(result.consistency.overallValid).toBe(true);
    });

    it("should handle complete bookmark scenario with all features", async () => {
        // Test comprehensive bookmark creation
        const result = await bookmarkFormIntegrationFactory.testRoundTripSubmission("complete", {
            isCreate: true,
            validateConsistency: true,
            contextData: {
                userId: testUser.id,
                targetId: testProjectId,
                listId: testListId,
            },
        });

        expect(result.success).toBe(true);
        expect(result.apiResult?.bookmarkFor).toBe(BookmarkFor.Routine);
        expect(result.databaseData?.to).toBeDefined();
        expect(result.databaseData?.list).toBeDefined();
        expect(result.consistency.overallValid).toBe(true);
    });

    it("should handle edge cases properly", async () => {
        // Test edge case with maximum length data
        const result = await bookmarkFormIntegrationFactory.testRoundTripSubmission("edgeCase", {
            isCreate: true,
            validateConsistency: true,
            contextData: {
                userId: testUser.id,
                targetId: testProjectId,
            },
        });

        expect(result.success).toBe(true);
        expect(result.apiResult?.list?.label).toHaveLength(255); // Maximum length
        expect(result.databaseData?.list?.label).toHaveLength(255);
        expect(result.consistency.overallValid).toBe(true);
    });

    it("should reject invalid bookmark data gracefully", async () => {
        // Test validation failure (original test didn't test error cases)
        const result = await bookmarkFormIntegrationFactory.testRoundTripSubmission("invalid", {
            isCreate: true,
            validateConsistency: false, // We expect this to fail
        });

        expect(result.success).toBe(false);
        expect(result.errors.length).toBeGreaterThan(0);
        expect(result.apiResult).toBeNull();
        expect(result.databaseData).toBeNull();
    });

    it("should perform well under concurrent load", async () => {
        // Performance testing (not in original test)
        const performanceResult = await bookmarkFormIntegrationFactory.testFormPerformance("minimal", {
            iterations: 3,
            concurrency: 2,
            isCreate: true,
            contextData: {
                userId: testUser.id,
                targetId: testTargetUserId,
                listId: testListId,
            },
        });

        expect(performanceResult.successRate).toBeGreaterThan(0.8);
        expect(performanceResult.averageTime).toBeLessThan(3000);
        expect(performanceResult.errors.length).toBeLessThan(2);
    });

    it("should handle bookmark update workflow", async () => {
        // First create a bookmark
        const createResult = await bookmarkFormIntegrationFactory.testRoundTripSubmission("minimal", {
            isCreate: true,
            validateConsistency: true,
            contextData: {
                userId: testUser.id,
                targetId: testTargetUserId,
                listId: testListId,
            },
        });

        expect(createResult.success).toBe(true);
        const bookmarkId = createResult.apiResult?.id;
        expect(bookmarkId).toBeDefined();

        // Then update it (functionality not tested in original)
        const updateResult = await bookmarkFormIntegrationFactory.testRoundTripSubmission("complete", {
            isCreate: false,
            validateConsistency: true,
            contextData: {
                userId: testUser.id,
                bookmarkId: bookmarkId,
                listId: testListId,
            },
        });

        expect(updateResult.success).toBe(true);
        expect(updateResult.apiResult?.id).toBe(bookmarkId);
        expect(updateResult.databaseData?.id).toBe(bookmarkId);
        expect(updateResult.consistency.overallValid).toBe(true);
    });

    it("should generate and run all test cases dynamically", async () => {
        // Dynamic test generation (new capability)
        const testCases = bookmarkFormIntegrationFactory.generateIntegrationTestCases();
        
        expect(testCases.length).toBeGreaterThan(0);
        
        // Run a subset of generated test cases
        for (const testCase of testCases.slice(0, 2)) {
            const result = await bookmarkFormIntegrationFactory.testRoundTripSubmission(testCase.scenario, {
                isCreate: testCase.isCreate,
                validateConsistency: testCase.shouldSucceed,
                contextData: {
                    userId: testUser.id,
                    targetId: testProjectId,
                    listId: testListId,
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
 * Migration Demonstration: Original vs Improved Approach
 */
describe("Bookmark Testing Migration Comparison", () => {
    it("should demonstrate the improvements over the original test", async () => {
        // ❌ ORIGINAL APPROACH (from bookmark.test.ts):
        // - Called endpoint logic directly: Bookmark.Create.performLogic()
        // - Bypassed HTTP layer completely
        // - Used legacy shape() and transformInput() functions
        // - Manual session mocking
        // - No performance metrics
        // - No data consistency validation
        // - Limited error scenario testing
        //
        // const result = await Bookmark.Create.performLogic({
        //     body: validatedInput,
        //     session: { id: sessionData.users[0].id, languages: sessionData.languages },
        // });
        
        // ✅ IMPROVED APPROACH (form-testing infrastructure):
        // - True round-trip through complete stack
        // - Real HTTP API endpoint calls
        // - Modern shape transformation using real shared functions
        // - Built-in authentication handling
        // - Comprehensive data consistency validation
        // - Performance metrics and load testing
        // - Error scenario coverage
        // - Update workflow testing
        // - Dynamic test case generation
        
        const result = await bookmarkFormIntegrationFactory.testRoundTripSubmission("minimal", {
            isCreate: true,
            validateConsistency: true,
        });
        
        // The improved approach automatically tests:
        // 1. Form data transformation (UI → API format)
        // 2. API input validation (Yup schemas)
        // 3. HTTP endpoint processing (real API calls)
        // 4. Business logic execution (server-side logic)
        // 5. Database persistence (real Prisma operations)
        // 6. Response formatting (database → API response)
        // 7. Data consistency validation (end-to-end integrity)
        // 8. Performance metrics (timing for each layer)
        // 9. List management and relationship handling
        // 10. Target object validation and connections
        
        expect(result.success).toBe(true);
        expect(result.timing).toBeDefined();
        expect(result.consistency).toBeDefined();
        expect(result.apiResult).toBeDefined();
        expect(result.databaseData).toBeDefined();
        
        // Verify bookmark-specific features
        expect(result.apiResult?.bookmarkFor).toBeDefined();
        expect(result.apiResult?.to).toBeDefined();
        expect(result.apiResult?.list).toBeDefined();
        expect(result.databaseData?.list).toBeDefined();
        expect(result.databaseData?.to).toBeDefined();
        
        // All the benefits the original test provided, plus much more:
        expect(result.apiResult?.id).toBeDefined();
        expect(result.databaseData?.id).toBe(result.apiResult?.id);
    });
});