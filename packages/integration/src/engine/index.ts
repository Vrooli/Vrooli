/**
 * Integration Form Testing Infrastructure
 * 
 * This module provides comprehensive round-trip form testing capabilities that work
 * with real API calls, databases, and testcontainers. It complements the UI-focused
 * form testing in the UI package by testing the complete data flow from form
 * submission through API endpoints to database persistence.
 * 
 * Key Features:
 * - Real API calls through endpoint logic
 * - Real database operations with testcontainers
 * - Data transformation validation across all layers
 * - Performance testing for complete data flow
 * - Consistency validation from form to database
 * - Dynamic test case generation
 * - Error handling and recovery testing
 * 
 * Usage:
 * 1. Define form fixtures for different test scenarios
 * 2. Configure the integration test factory with API endpoints and database functions
 * 3. Run comprehensive round-trip tests
 * 4. Validate data consistency across all layers
 * 5. Test performance under load
 * 
 * @example
 * ```typescript
 * // Create integration test factory
 * const factory = createIntegrationFormTestFactory({
 *   objectType: "Comment",
 *   validation: commentValidation,
 *   transformFunction: transformCommentValues,
 *   endpoints: { create: endpointsComment.createOne, update: endpointsComment.updateOne },
 *   formFixtures: commentFormFixtures,
 *   formToShape: commentFormToShape,
 *   findInDatabase: findCommentInDatabase,
 *   prismaModel: "comment",
 * });
 * 
 * // Test complete round-trip
 * const result = await factory.testRoundTripSubmission("minimal", {
 *   isCreate: true,
 *   validateConsistency: true,
 * });
 * 
 * expect(result.success).toBe(true);
 * expect(result.apiResult?.id).toBe(result.databaseData?.id);
 * ```
 */

// Core integration testing infrastructure
export {
    IntegrationFormTestFactory,
    createIntegrationFormTestFactory,
    type IntegrationFormTestConfig,
    type IntegrationFormTestResult,
} from "./IntegrationFormTestFactory.js";

// Example implementations
export {
    commentFormIntegrationFactory,
    commentIntegrationTestCases,
    commentFormFixtures,
    createTestCommentTarget,
} from "../examples/CommentFormIntegration.js";

// Re-export setup utilities for convenience
export {
    getPrisma,
    cleanDatabase,
    getSetupStatus,
} from "../setup/test-setup.js";

/**
 * Quick start guide for integration form testing:
 * 
 * 1. **Create Form Fixtures**: Define test data for different scenarios
 *    ```typescript
 *    const myFormFixtures = {
 *      minimal: { name: "Test", isPrivate: false },
 *      complete: { name: "Complete Test", description: "Full data", tags: ["test"] },
 *      invalid: { name: "", isPrivate: false }, // Should fail validation
 *      edgeCase: { name: "A".repeat(200), isPrivate: true }, // Edge case testing
 *    };
 *    ```
 * 
 * 2. **Configure Integration Test Factory**: Set up with real API endpoints
 *    ```typescript
 *    const factory = createIntegrationFormTestFactory({
 *      objectType: "MyObject",
 *      validation: myObjectValidation,
 *      transformFunction: transformMyObjectValues,
 *      endpoints: { create: endpointsMyObject.createOne, update: endpointsMyObject.updateOne },
 *      formFixtures: myFormFixtures,
 *      formToShape: myFormToShape,
 *      findInDatabase: findMyObjectInDatabase,
 *      prismaModel: "myObject",
 *    });
 *    ```
 * 
 * 3. **Write Integration Tests**: Test complete data flow
 *    ```typescript
 *    describe("MyObject Form Integration", () => {
 *      it("should complete round-trip form submission", async () => {
 *        const result = await factory.testRoundTripSubmission("complete", {
 *          isCreate: true,
 *          validateConsistency: true,
 *        });
 *        
 *        expect(result.success).toBe(true);
 *        expect(result.apiResult?.id).toBe(result.databaseData?.id);
 *        expect(result.consistency.overallValid).toBe(true);
 *      });
 *    });
 *    ```
 * 
 * 4. **Test Performance**: Validate performance under load
 *    ```typescript
 *    it("should maintain performance under load", async () => {
 *      const performanceResult = await factory.testFormPerformance("complete", {
 *        iterations: 10,
 *        concurrency: 3,
 *        isCreate: true,
 *      });
 *      
 *      expect(performanceResult.successRate).toBeGreaterThan(0.9);
 *      expect(performanceResult.averageTime).toBeLessThan(2000);
 *    });
 *    ```
 * 
 * 5. **Generate Dynamic Tests**: Let the factory create comprehensive test suites
 *    ```typescript
 *    const testCases = factory.generateIntegrationTestCases();
 *    testCases.forEach(testCase => {
 *      it(testCase.name, async () => {
 *        const result = await factory.testRoundTripSubmission(testCase.scenario, {
 *          isCreate: testCase.isCreate,
 *          validateConsistency: testCase.shouldSucceed,
 *        });
 *        
 *        expect(result.success).toBe(testCase.shouldSucceed);
 *      });
 *    });
 *    ```
 * 
 * **Key Differences from UI Testing:**
 * - Uses real API endpoints (not mocked)
 * - Uses real database with testcontainers
 * - Tests complete data flow through all layers
 * - Validates data persistence and consistency
 * - Tests performance of entire stack
 * - Runs in Node.js environment (not DOM)
 * 
 * **Performance Considerations:**
 * - Tests take longer due to real database operations
 * - Use appropriate timeouts (30s+ for complex operations)
 * - Consider test isolation and cleanup between tests
 * - Monitor resource usage with testcontainers
 */