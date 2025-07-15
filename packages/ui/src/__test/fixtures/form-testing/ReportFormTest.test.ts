import { describe, expect, it, beforeEach } from "vitest";
import { userEvent } from "@testing-library/user-event";
import { reportFormTestFactory, ReportOptions, type ReportFormData } from "./ReportFormTest.js";

/**
 * Comprehensive Report Form Tests
 * 
 * This test suite demonstrates how to use the Report form testing infrastructure
 * for simpler forms without translation validation. It focuses on:
 * - Business logic validation
 * - Conditional field behavior
 * - User workflow testing
 * - Different report scenarios
 */
describe("Report Form Testing", () => {
    
    describe("Form Validation Tests", () => {
        it("should validate minimal report data successfully", async () => {
            const result = await reportFormTestFactory.testFormValidation("minimal", {
                shouldPass: true,
            });
            
            expect(result.passed).toBe(true);
            expect(result.transformedData).toBeDefined();
            expect(result.transformedData?.reason).toBe(ReportOptions.Spam);
            expect(result.errors).toHaveLength(0);
        });
        
        it("should validate complete report data successfully", async () => {
            const result = await reportFormTestFactory.testFormValidation("complete", {
                shouldPass: true,
            });
            
            expect(result.passed).toBe(true);
            expect(result.transformedData).toBeDefined();
            expect(result.transformedData?.reason).toBe(ReportOptions.Inappropriate);
            expect(result.transformedData?.details).toContain("inappropriate language");
        });
        
        it("should reject invalid report data", async () => {
            const result = await reportFormTestFactory.testFormValidation("invalid", {
                shouldPass: false,
            });
            
            expect(result.passed).toBe(false);
            expect(result.errors.length).toBeGreaterThan(0);
            expect(result.errors.some(err => err.includes("reason"))).toBe(true);
        });
        
        it("should require otherReason when reason is 'Other'", async () => {
            const result = await reportFormTestFactory.testFormValidation("invalidOtherReason", {
                shouldPass: false,
            });
            
            expect(result.passed).toBe(false);
            expect(result.errors.some(err => 
                err.includes("otherReason") || err.includes("required"),
            )).toBe(true);
        });
    });
    
    describe("Field-Specific Validation Tests", () => {
        it("should validate reason field correctly", async () => {
            const fieldResults = await reportFormTestFactory.testFieldBehavior("reason", [
                ReportOptions.Spam,             // Should pass
                ReportOptions.Inappropriate,    // Should pass
                ReportOptions.PII,              // Should pass
                ReportOptions.Scam,             // Should pass
                ReportOptions.Other,            // Should pass
                "",                             // Should fail: required
                "InvalidReason",                // Should fail: invalid option
            ]);
            
            expect(fieldResults.length).toBe(7);
            expect(fieldResults[0].passed).toBe(true);   // Valid spam reason
            expect(fieldResults[1].passed).toBe(true);   // Valid inappropriate reason
            expect(fieldResults[2].passed).toBe(true);   // Valid PII reason
            expect(fieldResults[3].passed).toBe(true);   // Valid scam reason
            expect(fieldResults[4].passed).toBe(true);   // Valid other reason
            expect(fieldResults[5].passed).toBe(false);  // Empty reason
            expect(fieldResults[6].passed).toBe(false);  // Invalid reason
        });
        
        it("should validate details field correctly", async () => {
            const fieldResults = await reportFormTestFactory.testFieldBehavior("details", [
                "Valid detailed explanation",                        // Should pass
                "",                                                 // Should pass: optional
                "A".repeat(5000),                                   // Should pass: at limit
                "A".repeat(5001),                                   // Should fail: too long
                "Details with\nnewlines\nand formatting",          // Should pass: multiline
            ]);
            
            expect(fieldResults.length).toBe(5);
            expect(fieldResults[0].passed).toBe(true);   // Valid details
            expect(fieldResults[1].passed).toBe(true);   // Empty details (optional)
            expect(fieldResults[2].passed).toBe(true);   // At character limit
            expect(fieldResults[3].passed).toBe(false);  // Over character limit
            expect(fieldResults[4].passed).toBe(true);   // Multiline details
        });
    });
    
    describe("Conditional Field Tests", () => {
        it("should handle 'Other' reason requiring otherReason field", async () => {
            const result = await reportFormTestFactory.testConditionalFields({
                reason: ReportOptions.Other,
                expectRequired: ["otherReason"],
            });
            
            expect(result.conditionMet).toBe(true);
            expect(result.requiredFieldsValidated).toContain("otherReason");
        });
        
        it("should not require otherReason for non-Other reasons", async () => {
            const result = await reportFormTestFactory.testConditionalFields({
                reason: ReportOptions.Spam,
                expectRequired: [],
            });
            
            expect(result.conditionMet).toBe(false);
            expect(result.validationResult.passed).toBe(true);
        });
        
        it("should test all report reason workflows", async () => {
            const results = await reportFormTestFactory.testReportReasonWorkflows();
            
            expect(results).toHaveLength(5); // All ReportOptions
            
            results.forEach(result => {
                expect(result.passed).toBe(true);
                if (result.reason === ReportOptions.Other) {
                    expect(result.hasOtherReason).toBe(true);
                } else {
                    expect(result.hasOtherReason).toBe(false);
                }
            });
        });
    });
    
    describe("User Interaction Tests", () => {
        let user: ReturnType<typeof userEvent.setup>;
        
        beforeEach(() => {
            user = userEvent.setup();
        });
        
        it("should handle complete user interaction flow for report creation", async () => {
            const result = await reportFormTestFactory.testUserInteraction("complete", { 
                user,
            });
            
            expect(result.success).toBe(true);
            expect(result.formData).toBeDefined();
            expect(result.submissionAttempted).toBe(true);
            expect(result.validationPassed).toBe(true);
        });
        
        it("should handle user selecting 'Other' reason and filling otherReason", async () => {
            const result = await reportFormTestFactory.testUserInteraction("withOtherReason", { 
                user,
            });
            
            expect(result.success).toBe(true);
            expect(result.formData.reason).toBe(ReportOptions.Other);
            expect(result.formData.otherReason).toBeDefined();
            expect(result.formData.otherReason?.length).toBeGreaterThan(0);
        });
        
        it("should handle form cancellation correctly", async () => {
            const result = await reportFormTestFactory.testUserCancellation({ user });
            
            expect(result.cancelled).toBe(true);
            expect(result.formCleared).toBe(true);
        });
    });
    
    describe("Report Targeting Tests", () => {
        it("should handle reports for different object types", async () => {
            const results = await reportFormTestFactory.testReportTargeting();
            
            expect(results).toHaveLength(5); // Project, Comment, Team, User, Resource
            
            results.forEach(result => {
                expect(result.passed).toBe(true);
                expect(result.apiInput).toBeDefined();
                expect(result.apiInput?.createdFor.__typename).toBe(result.targetType);
            });
        });
        
        it("should create appropriate API input for each target type", async () => {
            const projectReport: ReportFormData = {
                reason: ReportOptions.Spam,
                details: "This project contains spam content",
                language: "en",
                createdFor: { __typename: "Project", id: "project_123" },
            };
            
            const apiInput = reportFormTestFactory.transformToAPIInput(projectReport);
            
            expect(apiInput).toBeDefined();
            expect(apiInput.__typename).toBe("Report");
            expect(apiInput.reason).toBe(ReportOptions.Spam);
            expect(apiInput.details).toBe("This project contains spam content");
            expect(apiInput.createdFor.__typename).toBe("Project");
            expect(apiInput.createdFor.id).toBe("project_123");
        });
    });
    
    describe("Performance Tests", () => {
        it("should meet performance expectations for simple form", async () => {
            const metrics = await reportFormTestFactory.measurePerformance("complete");
            
            expect(metrics.renderTime).toBeLessThan(50); // ms - Simple form
            expect(metrics.validationTime).toBeLessThan(30); // ms - Simple validation
            expect(metrics.memoryUsage).toBeLessThan(2 * 1024 * 1024); // 2MB
        });
        
        it("should handle edge case data efficiently", async () => {
            const startTime = Date.now();
            const result = await reportFormTestFactory.testFormValidation("edgeCase", {
                shouldPass: false, // Should fail due to long text
            });
            const endTime = Date.now();
            
            expect(result.passed).toBe(false);
            expect(endTime - startTime).toBeLessThan(100); // Should validate quickly even with long text
        });
    });
    
    describe("Error Scenario Tests", () => {
        it("should handle various error scenarios gracefully", async () => {
            const result = await reportFormTestFactory.testErrorScenarios([
                "NETWORK_ERROR",
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
        
        it("should prevent duplicate report submissions", async () => {
            const result = await reportFormTestFactory.testConcurrentSubmissions("complete", 3);
            
            expect(result.allSubmissionsHandled).toBe(true);
            expect(result.noDataCorruption).toBe(true);
            expect(result.oneSuccessfulSubmission).toBe(true);
        });
    });
    
    describe("Accessibility Tests", () => {
        it("should have proper form accessibility", async () => {
            const a11yResult = await reportFormTestFactory.testAccessibility("complete");
            
            expect(a11yResult.hasRequiredLabels).toBe(true);
            expect(a11yResult.hasProperRoles).toBe(true);
            expect(a11yResult.supportsKeyboardNavigation).toBe(true);
            expect(a11yResult.violations).toHaveLength(0);
        });
        
        it("should provide proper screen reader support for conditional fields", async () => {
            const a11yResult = await reportFormTestFactory.testAccessibility("withOtherReason");
            
            expect(a11yResult.hasRequiredLabels).toBe(true);
            expect(a11yResult.conditionalFieldsAccessible).toBe(true);
        });
    });
});

/**
 * Integration test demonstrating different report scenarios
 */
describe("Report Form Scenario Testing", () => {
    it("should handle the complete reporting workflow for different scenarios", async () => {
        const user = userEvent.setup();
        
        // Test spam report scenario
        const spamResult = await reportFormTestFactory.testUserInteraction("minimal", { 
            user,
            isCreate: true, 
        });
        expect(spamResult.success).toBe(true);
        expect(spamResult.formData.reason).toBe(ReportOptions.Spam);
        
        // Test inappropriate content scenario
        const inappropriateResult = await reportFormTestFactory.testUserInteraction("complete", { 
            user,
            isCreate: true, 
        });
        expect(inappropriateResult.success).toBe(true);
        expect(inappropriateResult.formData.reason).toBe(ReportOptions.Inappropriate);
        
        // Test PII report scenario
        const piiResult = await reportFormTestFactory.testUserInteraction("piiReport", { 
            user,
            isCreate: true, 
        });
        expect(piiResult.success).toBe(true);
        expect(piiResult.formData.reason).toBe(ReportOptions.PII);
        
        // Test scam report scenario
        const scamResult = await reportFormTestFactory.testUserInteraction("scamReport", { 
            user,
            isCreate: true, 
        });
        expect(scamResult.success).toBe(true);
        expect(scamResult.formData.reason).toBe(ReportOptions.Scam);
        
        // Test other reason scenario
        const otherResult = await reportFormTestFactory.testUserInteraction("withOtherReason", { 
            user,
            isCreate: true, 
        });
        expect(otherResult.success).toBe(true);
        expect(otherResult.formData.reason).toBe(ReportOptions.Other);
        expect(otherResult.formData.otherReason).toBeDefined();
        
        console.log("✅ All report form scenarios completed successfully!");
    });
});
