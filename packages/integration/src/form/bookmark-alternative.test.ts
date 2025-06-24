/**
 * Bookmark Form Integration Tests - Migrated to Form-Testing Infrastructure
 * 
 * This demonstrates the migration from legacy round-trip testing to the modern
 * form-testing infrastructure. This test file shows the CORRECT approach for
 * true round-trip testing with real API endpoints and database persistence.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { 
    bookmarkFormIntegrationFactory, 
    createTestBookmarkTarget, 
    createTestBookmarkList 
} from "../examples/BookmarkFormIntegration.js";
import { BookmarkFor } from "@vrooli/shared";
import { createSimpleTestUser } from "../utils/simple-helpers.js";

describe("Bookmark Form Integration Tests - Form-Testing Infrastructure", () => {
    let testUser: any;
    let testTargetId: string;
    let testListId: string;

    beforeEach(async () => {
        // Create test user
        const result = await createSimpleTestUser();
        testUser = result.user;
        
        // Create test target to bookmark
        testTargetId = await createTestBookmarkTarget(BookmarkFor.Project);
        
        // Create test bookmark list
        testListId = await createTestBookmarkList("Test List", testUser.id);
    });

    it("should complete full bookmark creation workflow with minimal data", async () => {
        // Test the minimal scenario through complete round-trip
        const result = await bookmarkFormIntegrationFactory.testRoundTripSubmission("minimal", {
            isCreate: true,
            validateConsistency: true,
            contextData: {
                userId: testUser.id,
                targetId: testTargetId,
                listId: testListId,
            },
        });

        // Verify complete success
        expect(result.success).toBe(true);
        expect(result.errors).toHaveLength(0);
        
        // Verify API response
        expect(result.apiResult).toBeDefined();
        expect(result.apiResult?.bookmarkFor).toBe(BookmarkFor.User);
        
        // Verify database persistence
        expect(result.databaseData).toBeDefined();
        expect(result.databaseData?.id).toBe(result.apiResult?.id);
        
        // Verify data consistency across layers
        expect(result.consistency.overallValid).toBe(true);
        expect(result.consistency.formToApi).toBe(true);
        expect(result.consistency.apiToDatabase).toBe(true);
        
        // Verify performance is reasonable
        expect(result.timing.total).toBeLessThan(5000); // Less than 5 seconds
        expect(result.timing.apiCall).toBeLessThan(2000); // Less than 2 seconds for API call
    });

    it("should complete bookmark creation with new list", async () => {
        // Test creating a bookmark with a new list
        const result = await bookmarkFormIntegrationFactory.testRoundTripSubmission("withNewList", {
            isCreate: true,
            validateConsistency: true,
            contextData: {
                userId: testUser.id,
                targetId: testTargetId,
            },
        });

        expect(result.success).toBe(true);
        expect(result.apiResult?.list?.label).toBe("My Favorite Projects");
        expect(result.databaseData?.list).toBeDefined();
        expect(result.consistency.overallValid).toBe(true);
    });

    it("should handle validation failures gracefully", async () => {
        // Test invalid data handling
        const result = await bookmarkFormIntegrationFactory.testRoundTripSubmission("invalid", {
            isCreate: true,
            validateConsistency: false, // We expect this to fail
        });

        expect(result.success).toBe(false);
        expect(result.errors.length).toBeGreaterThan(0);
        expect(result.apiResult).toBeNull();
        expect(result.databaseData).toBeNull();
    });

    it("should perform well under load", async () => {
        // Test performance with multiple concurrent operations
        const performanceResult = await bookmarkFormIntegrationFactory.testFormPerformance("minimal", {
            iterations: 5,
            concurrency: 2,
            isCreate: true,
            contextData: {
                userId: testUser.id,
                targetId: testTargetId,
                listId: testListId,
            },
        });

        expect(performanceResult.successRate).toBeGreaterThan(0.8); // 80% success rate
        expect(performanceResult.averageTime).toBeLessThan(3000); // Average under 3 seconds
        expect(performanceResult.errors.length).toBeLessThan(2); // Few errors expected
    });

    it("should generate and run all test cases dynamically", async () => {
        // Use the dynamic test case generation
        const testCases = bookmarkFormIntegrationFactory.generateIntegrationTestCases();
        
        expect(testCases.length).toBeGreaterThan(0);
        
        // Run a subset of generated test cases
        for (const testCase of testCases.slice(0, 2)) {
            const result = await bookmarkFormIntegrationFactory.testRoundTripSubmission(testCase.scenario, {
                isCreate: testCase.isCreate,
                validateConsistency: testCase.shouldSucceed,
                contextData: {
                    userId: testUser.id,
                    targetId: testTargetId,
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
 * Comparison Test: Legacy vs Form-Testing Approach
 * 
 * This demonstrates the differences between the old and new approaches
 */
describe("Migration Comparison: Legacy vs Form-Testing", () => {
    it("should demonstrate the difference in testing approaches", async () => {
        // ❌ OLD APPROACH (DEPRECATED):
        // Direct Prisma access - bypasses entire API layer
        // const bookmark = await prisma.bookmark.create({ data: { ... } });
        
        // ✅ NEW APPROACH (RECOMMENDED):
        // True round-trip through complete stack
        const result = await bookmarkFormIntegrationFactory.testRoundTripSubmission("minimal", {
            isCreate: true,
            validateConsistency: true,
        });
        
        // The new approach tests:
        // 1. Form data transformation
        // 2. API endpoint validation
        // 3. Business logic execution
        // 4. Database persistence
        // 5. Response formatting
        // 6. Data consistency across all layers
        
        expect(result.success).toBe(true);
        expect(result.timing).toBeDefined(); // Performance metrics
        expect(result.consistency).toBeDefined(); // Data integrity validation
    });
});