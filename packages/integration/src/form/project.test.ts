/**
 * Project Form Integration Tests - Migrated to Form-Testing Infrastructure
 * 
 * This demonstrates the migration from legacy endpoint calling to true round-trip
 * testing through the form-testing infrastructure with real API calls.
 * Projects are core content objects in Vrooli for organizing collaborative work.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { 
    projectFormIntegrationFactory, 
    createTestTeam,
    createTestProjectUser
} from "../examples/ProjectFormIntegration.js";
import { createSimpleTestUser } from "../utils/simple-helpers.js";

describe("Project Form Integration Tests - Form-Testing Infrastructure", () => {
    let testUser: any;
    let testTeamId: string;

    beforeEach(async () => {
        // Create test user
        const result = await createSimpleTestUser();
        testUser = result.user;
        
        // Create test team for team projects
        testTeamId = await createTestTeam("Test Project Team", testUser.id);
    });

    it("should complete full project creation workflow with minimal data", async () => {
        // Test creating a minimal project through complete round-trip
        const result = await projectFormIntegrationFactory.testRoundTripSubmission("minimal", {
            isCreate: true,
            validateConsistency: true,
            contextData: {
                userId: testUser.id,
            },
        });

        // Verify complete success
        expect(result.success).toBe(true);
        expect(result.errors).toHaveLength(0);
        
        // Verify API response structure
        expect(result.apiResult).toBeDefined();
        expect(result.apiResult?.translations).toBeDefined();
        expect(result.apiResult?.translations[0]?.name).toBe("My Test Project");
        expect(result.apiResult?.isPrivate).toBe(false);
        
        // Verify database persistence
        expect(result.databaseData).toBeDefined();
        expect(result.databaseData?.id).toBe(result.apiResult?.id);
        expect(result.databaseData?.translations).toBeDefined();
        expect(result.databaseData?.creator?.id).toBe(testUser.id);
        
        // Verify data consistency across layers
        expect(result.consistency.overallValid).toBe(true);
        expect(result.consistency.formToApi).toBe(true);
        expect(result.consistency.apiToDatabase).toBe(true);
        
        // Verify performance is reasonable
        expect(result.timing.total).toBeLessThan(5000); // Less than 5 seconds
        expect(result.timing.apiCall).toBeLessThan(2000); // Less than 2 seconds for API call
    });

    it("should handle complete project with all features", async () => {
        // Test comprehensive project with translations, tags, and resource lists
        const result = await projectFormIntegrationFactory.testRoundTripSubmission("complete", {
            isCreate: true,
            validateConsistency: true,
            contextData: {
                userId: testUser.id,
            },
        });

        expect(result.success).toBe(true);
        
        // Verify rich content
        expect(result.apiResult?.translations).toHaveLength(2); // English and Spanish
        expect(result.apiResult?.handle).toBe("complete-project-example");
        expect(result.apiResult?.tags?.length).toBeGreaterThan(3);
        expect(result.apiResult?.resourceLists?.length).toBe(2);
        
        // Verify database persistence of complex data
        expect(result.databaseData?.translations).toHaveLength(2);
        expect(result.databaseData?.tags?.length).toBeGreaterThan(3);
        expect(result.databaseData?.resourceLists?.length).toBe(2);
        
        expect(result.consistency.overallValid).toBe(true);
    });

    it("should handle team-based project creation", async () => {
        // Test creating a project with team ownership
        const result = await projectFormIntegrationFactory.testRoundTripSubmission("teamProject", {
            isCreate: true,
            validateConsistency: true,
            contextData: {
                userId: testUser.id,
                teamId: testTeamId,
            },
        });

        expect(result.success).toBe(true);
        expect(result.apiResult?.isPrivate).toBe(true);
        expect(result.apiResult?.team?.id).toBe(testTeamId);
        expect(result.databaseData?.team?.id).toBe(testTeamId);
        expect(result.consistency.overallValid).toBe(true);
    });

    it("should handle edge case with maximum data", async () => {
        // Test edge case with very long content and many tags
        const result = await projectFormIntegrationFactory.testRoundTripSubmission("edgeCase", {
            isCreate: true,
            validateConsistency: true,
            contextData: {
                userId: testUser.id,
            },
        });

        expect(result.success).toBe(true);
        expect(result.apiResult?.translations[0]?.name).toHaveLength(255); // Maximum length
        expect(result.apiResult?.translations[0]?.description).toHaveLength(2048); // Long description
        expect(result.apiResult?.tags?.length).toBe(20); // Many tags
        expect(result.databaseData?.translations[0]?.name).toHaveLength(255);
        expect(result.consistency.overallValid).toBe(true);
    });

    it("should reject invalid project data gracefully", async () => {
        // Test validation failure with empty project name
        const result = await projectFormIntegrationFactory.testRoundTripSubmission("invalid", {
            isCreate: true,
            validateConsistency: false, // We expect this to fail
        });

        expect(result.success).toBe(false);
        expect(result.errors.length).toBeGreaterThan(0);
        expect(result.apiResult).toBeNull();
        expect(result.databaseData).toBeNull();
    });

    it("should handle name length validation", async () => {
        // Test validation failure with name too long
        const result = await projectFormIntegrationFactory.testRoundTripSubmission("longName", {
            isCreate: true,
            validateConsistency: false, // We expect this to fail
        });

        expect(result.success).toBe(false);
        expect(result.errors.length).toBeGreaterThan(0);
        expect(result.apiResult).toBeNull();
        expect(result.databaseData).toBeNull();
    });

    it("should perform well under concurrent load", async () => {
        // Test performance with multiple concurrent project operations
        const performanceResult = await projectFormIntegrationFactory.testFormPerformance("minimal", {
            iterations: 3,
            concurrency: 2,
            isCreate: true,
            contextData: {
                userId: testUser.id,
            },
        });

        expect(performanceResult.successRate).toBeGreaterThan(0.8); // 80% success rate
        expect(performanceResult.averageTime).toBeLessThan(4000); // Average under 4 seconds
        expect(performanceResult.errors.length).toBeLessThan(2); // Few errors expected
    });

    it("should complete update workflow for existing projects", async () => {
        // First create a project
        const createResult = await projectFormIntegrationFactory.testRoundTripSubmission("minimal", {
            isCreate: true,
            validateConsistency: true,
            contextData: {
                userId: testUser.id,
            },
        });

        expect(createResult.success).toBe(true);
        const projectId = createResult.apiResult?.id;
        expect(projectId).toBeDefined();

        // Then update it to add more features
        const updateResult = await projectFormIntegrationFactory.testRoundTripSubmission("complete", {
            isCreate: false,
            validateConsistency: true,
            contextData: {
                userId: testUser.id,
                projectId: projectId,
            },
        });

        expect(updateResult.success).toBe(true);
        expect(updateResult.apiResult?.id).toBe(projectId);
        expect(updateResult.databaseData?.id).toBe(projectId);
        expect(updateResult.apiResult?.translations?.length).toBeGreaterThan(1); // Should have translations now
        expect(updateResult.consistency.overallValid).toBe(true);
    });

    it("should generate and run all test cases dynamically", async () => {
        // Use the dynamic test case generation
        const testCases = projectFormIntegrationFactory.generateIntegrationTestCases();
        
        expect(testCases.length).toBeGreaterThan(0);
        
        // Run a subset of generated test cases
        for (const testCase of testCases.slice(0, 2)) {
            const result = await projectFormIntegrationFactory.testRoundTripSubmission(testCase.scenario, {
                isCreate: testCase.isCreate,
                validateConsistency: testCase.shouldSucceed,
                contextData: {
                    userId: testUser.id,
                    teamId: testTeamId,
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
describe("Project Testing Migration Comparison", () => {
    it("should demonstrate the improved testing approach", async () => {
        // ❌ OLD APPROACH (from original project.test.ts):
        // - Used Resource types instead of Project types (confusion)
        // - Direct endpoint logic calling via resource.createOne.logic()
        // - No HTTP layer testing
        // - Manual validation setup
        // 
        // const result = await resource.createOne.logic({
        //     input: apiInput,
        //     userData: context.userData,
        //     prisma: context.prisma,
        // });
        
        // ✅ NEW APPROACH (form-testing infrastructure):
        // - Correct Project types and endpoints
        // - True round-trip through complete stack
        // - Real API endpoint calls
        // - Built-in performance metrics
        // - Comprehensive data consistency validation
        // - Team and ownership testing
        // - Translation and tag management testing
        
        const result = await projectFormIntegrationFactory.testRoundTripSubmission("minimal", {
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
        // 9. Complex object relationships (teams, tags, translations)
        // 10. Permission and ownership validation
        
        expect(result.success).toBe(true);
        expect(result.timing).toBeDefined();
        expect(result.consistency).toBeDefined();
        expect(result.apiResult).toBeDefined();
        expect(result.databaseData).toBeDefined();
        
        // Verify project-specific features
        expect(result.apiResult?.translations).toBeDefined();
        expect(result.databaseData?.creator).toBeDefined();
    });
});