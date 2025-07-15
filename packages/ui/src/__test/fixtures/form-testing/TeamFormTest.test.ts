import { describe, expect, it, beforeEach } from "vitest";
import { userEvent } from "@testing-library/user-event";
import { teamFormTestFactory, type TeamFormData } from "./TeamFormTest.js";

/**
 * Comprehensive Team Form Tests
 * 
 * This test suite demonstrates how to use the Team form testing infrastructure
 * to validate form behavior, user interactions, field validation, and API integration.
 * 
 * The tests use the actual validation schemas, transformation functions, and 
 * endpoints from the shared package, ensuring that form testing matches 
 * production behavior.
 */
describe("Team Form Testing", () => {
    
    describe("Form Validation Tests", () => {
        it("should validate minimal team data successfully", async () => {
            const result = await teamFormTestFactory.testFormValidation("minimal", {
                shouldPass: true,
            });
            
            expect(result.passed).toBe(true);
            expect(result.transformedData).toBeDefined();
            expect(result.transformedData?.translationsCreate?.[0]?.name).toBe("Test Team");
            expect(result.errors).toHaveLength(0);
        });
        
        it("should validate complete team data successfully", async () => {
            const result = await teamFormTestFactory.testFormValidation("complete", {
                shouldPass: true,
            });
            
            if (!result.passed) {
                console.error("Validation failed with errors:", result.errors);
                console.error("Transformed data:", JSON.stringify(result.transformedData, null, 2));
            }
            
            expect(result.passed).toBe(true);
            expect(result.transformedData).toBeDefined();
            expect(result.transformedData?.isOpenToNewMembers).toBe(true);
            expect(result.transformedData?.tagsConnect || result.transformedData?.tagsCreate).toBeDefined();
        });
        
        it("should reject invalid team data", async () => {
            const result = await teamFormTestFactory.testFormValidation("invalid", {
                shouldPass: false,
            });
            
            expect(result.passed).toBe(false);
            expect(result.errors.length).toBeGreaterThan(0);
            expect(result.errors.some(err => err.includes("name"))).toBe(true);
        });
        
        it("should handle edge case data appropriately", async () => {
            const result = await teamFormTestFactory.testFormValidation("edgeCase", {
                shouldPass: false, // Very long name should fail validation
            });
            
            expect(result.passed).toBe(false);
            expect(result.errors.some(err => err.includes("name") || err.includes("length"))).toBe(true);
        });
    });
    
    describe("Field-Specific Validation Tests", () => {
        it("should validate team name field correctly", async () => {
            const fieldResults = await teamFormTestFactory.testFieldBehavior("name", [
                "Valid Team Name",          // Should pass
                "Team-123",                 // Should pass
                "",                         // Should fail: required
                "A".repeat(101),            // Should fail: too long
                "Team with émojis 🚀",      // Should pass: unicode support
            ]);
            
            expect(fieldResults.length).toBe(5);
            expect(fieldResults[0].passed).toBe(true);   // Valid name
            expect(fieldResults[1].passed).toBe(true);   // Name with numbers
            expect(fieldResults[2].passed).toBe(false);  // Empty name
            expect(fieldResults[3].passed).toBe(false);  // Too long
            expect(fieldResults[4].passed).toBe(true);   // Unicode characters
        });
        
        it("should validate team bio field correctly", async () => {
            const fieldResults = await teamFormTestFactory.testFieldBehavior("bio", [
                "Valid bio description",     // Should pass
                "",                          // Should pass: bio is optional
                "A".repeat(2048),            // Should pass: at limit
                "A".repeat(2049),            // Should fail: too long
                "Bio with\nnewlines",        // Should pass: multiline
            ]);
            
            expect(fieldResults.length).toBe(5);
            expect(fieldResults[0].passed).toBe(true);   // Valid bio
            expect(fieldResults[1].passed).toBe(true);   // Empty bio (optional)
            expect(fieldResults[2].passed).toBe(true);   // At character limit
            expect(fieldResults[3].passed).toBe(false);  // Over character limit
            expect(fieldResults[4].passed).toBe(true);   // Multiline bio
        });
    });
    
    describe("User Interaction Tests", () => {
        let user: ReturnType<typeof userEvent.setup>;
        
        beforeEach(() => {
            user = userEvent.setup();
        });
        
        it("should handle complete user interaction flow for team creation", async () => {
            const result = await teamFormTestFactory.testUserInteraction("complete", { 
                user,
                isCreate: true, 
            });
            
            expect(result.success).toBe(true);
            expect(result.formData).toBeDefined();
            expect(result.submissionAttempted).toBe(true);
            expect(result.validationPassed).toBe(true);
        });
        
        it("should handle user interaction with invalid data", async () => {
            const result = await teamFormTestFactory.testUserInteraction("invalid", { 
                user,
                isCreate: true, 
            });
            
            expect(result.success).toBe(false);
            expect(result.submissionAttempted).toBe(true);
            expect(result.validationPassed).toBe(false);
            expect(result.errors?.length).toBeGreaterThan(0);
        });
        
        it("should handle form cancellation correctly", async () => {
            const result = await teamFormTestFactory.testUserCancellation({ user });
            
            expect(result.cancelled).toBe(true);
            expect(result.formCleared).toBe(true);
        });
    });
    
    describe("API Integration Tests", () => {
        it("should transform form data to correct API input for creation", async () => {
            const formData: TeamFormData = {
                name: "API Test Team",
                bio: "Testing API integration",
                isPrivate: false,
                isOpenToNewMembers: true,
                tags: ["api", "test"],
                language: "en",
            };
            
            const apiInput = teamFormTestFactory.transformToAPIInput(formData);
            
            expect(apiInput).toBeDefined();
            expect(apiInput.__typename).toBe("Team");
            expect(apiInput.translations[0].name).toBe("API Test Team");
            expect(apiInput.translations[0].bio).toBe("Testing API integration");
            expect(apiInput.isPrivate).toBe(false);
            expect(apiInput.isOpenToNewMembers).toBe(true);
        });
        
        it("should generate valid MSW handlers for testing", async () => {
            const handlers = teamFormTestFactory.createMSWHandlers();
            
            expect(handlers.create).toBeDefined();
            expect(handlers.update).toBeDefined();
            expect(handlers.success).toBeInstanceOf(Array);
            expect(handlers.error).toBeInstanceOf(Array);
        });
    });
    
    describe("Performance Tests", () => {
        it("should meet performance expectations for form rendering", async () => {
            const metrics = await teamFormTestFactory.measurePerformance("complete");
            
            expect(metrics.renderTime).toBeLessThan(100); // ms
            expect(metrics.validationTime).toBeLessThan(50); // ms
            expect(metrics.memoryUsage).toBeLessThan(5 * 1024 * 1024); // 5MB
        });
        
        it("should handle large datasets efficiently", async () => {
            const largeFormData: TeamFormData = {
                __typename: "Team" as const,
                id: "01234567890123456799",
                config: undefined,
                handle: "large_data_team",
                isPrivate: false,
                isOpenToNewMembers: true,
                bannerImage: undefined,
                profileImage: undefined,
                tags: Array.from({ length: 20 }, (_, i) => ({ 
                    __typename: "Tag" as const, 
                    id: `1234567890123456789${i.toString().padStart(1, '0')}` 
                })),
                translations: [{
                    __typename: "TeamTranslation" as const,
                    id: "01234567890123456784",
                    language: "en",
                    name: "Team with Large Data",
                    bio: "A".repeat(2000), // Near character limit
                }],
                memberInvites: undefined,
                members: undefined,
                membersDelete: undefined,
            };
            
            const startTime = Date.now();
            const result = await teamFormTestFactory.testFormValidation("custom", {
                shouldPass: true,
                customData: largeFormData,
            });
            const endTime = Date.now();
            
            expect(result.passed).toBe(true);
            expect(endTime - startTime).toBeLessThan(200); // Should validate quickly
        });
    });
    
    describe("Accessibility Tests", () => {
        it("should have proper form accessibility", async () => {
            const a11yResult = await teamFormTestFactory.testAccessibility("complete");
            
            expect(a11yResult.hasRequiredLabels).toBe(true);
            expect(a11yResult.hasProperRoles).toBe(true);
            expect(a11yResult.supportsKeyboardNavigation).toBe(true);
            expect(a11yResult.violations).toHaveLength(0);
        });
    });
    
    describe("Translation Tests", () => {
        it("should handle multiple languages correctly", async () => {
            const languages = ["en", "es", "fr", "de"];
            
            for (const language of languages) {
                const formData: TeamFormData = {
                    __typename: "Team" as const,
                    id: "01234567890123456798",
                    config: undefined,
                    handle: `team_${language}`,
                    isPrivate: false,
                    isOpenToNewMembers: true,
                    bannerImage: undefined,
                    profileImage: undefined,
                    tags: [{ __typename: "Tag" as const, id: "12345678901234567890" }],
                    translations: [{
                        __typename: "TeamTranslation" as const,
                        id: "01234567890123456785",
                        language,
                        name: `Team in ${language}`,
                        bio: `Description in ${language}`,
                    }],
                    memberInvites: undefined,
                    members: undefined,
                    membersDelete: undefined,
                };
                
                const result = await teamFormTestFactory.testFormValidation("custom", {
                        shouldPass: true,
                    customData: formData,
                });
                
                expect(result.passed).toBe(true);
                expect(result.transformedData?.translationsCreate?.[0]?.language).toBe(language);
            }
        });
    });
    
    describe("Edge Cases and Error Scenarios", () => {
        it("should handle network errors gracefully", async () => {
            const result = await teamFormTestFactory.testErrorScenarios([
                "NETWORK_ERROR",
                "TIMEOUT_ERROR", 
                "SERVER_ERROR",
                "VALIDATION_ERROR",
                "PERMISSION_ERROR",
            ]);
            
            result.forEach(scenarioResult => {
                expect(scenarioResult.errorHandledCorrectly).toBe(true);
                expect(scenarioResult.userFeedbackProvided).toBe(true);
                expect(scenarioResult.formStatePreserved).toBe(true);
            });
        });
        
        it("should handle concurrent form submissions", async () => {
            const result = await teamFormTestFactory.testConcurrentSubmissions("complete", 3);
            
            expect(result.allSubmissionsHandled).toBe(true);
            expect(result.noDataCorruption).toBe(true);
            expect(result.oneSuccessfulSubmission).toBe(true);
        });
    });
});

/**
 * Integration test demonstrating the complete form lifecycle
 */
describe("Team Form Complete Integration", () => {
    it("should handle the complete team creation lifecycle", async () => {
        const user = userEvent.setup();
        
        // 1. Test form rendering and initial state
        const initialState = teamFormTestFactory.createFormData("minimal");
        expect(initialState.name).toBe("Test Team");
        
        // 2. Test user interaction and data entry
        const interactionResult = await teamFormTestFactory.testUserInteraction("complete", { 
            user,
            isCreate: true, 
        });
        expect(interactionResult.success).toBe(true);
        
        // 3. Test form validation
        const validationResult = await teamFormTestFactory.testFormValidation("complete", {
            isCreate: true,
            shouldPass: true,
        });
        expect(validationResult.passed).toBe(true);
        
        // 4. Test API transformation
        const apiInput = teamFormTestFactory.transformToAPIInput(
            teamFormTestFactory.createFormData("complete"),
        );
        expect(apiInput.__typename).toBe("Team");
        
        // 5. Test error recovery
        const errorResult = await teamFormTestFactory.testErrorScenarios(["SERVER_ERROR"]);
        expect(errorResult[0].errorHandledCorrectly).toBe(true);
        
        console.log("✅ Complete team form lifecycle test passed!");
    });
});
