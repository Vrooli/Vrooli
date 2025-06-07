import { describe, it, beforeEach, vi, expect } from "vitest";
import { type Logger } from "winston";
import { ValidationEngine } from "../validationEngine.js";

describe("ValidationEngine", () => {
    let engine: ValidationEngine;
    let mockLogger: Logger;
    let mockReasoningResult: any;
    let mockFramework: any;

    beforeEach(() => {
        mockLogger = {
            debug: vi.fn(),
            info: vi.fn(),
            warn: vi.fn(),
            error: vi.fn(),
        } as any;

        mockReasoningResult = {
            conclusion: "Data analysis shows 60% of values exceed threshold",
            reasoning: [
                "Analyzed data array containing 5 elements",
                "Applied threshold comparison of 3 to each element",
                "Found elements 4 and 5 exceed the threshold",
                "Calculated percentage: 2/5 = 40% exceed threshold",
                "Concluded that minority of values exceed threshold",
            ],
            evidence: [
                {
                    type: "fact",
                    content: "Data contains values [1, 2, 3, 4, 5]",
                    confidence: 1.0,
                },
                {
                    type: "inference",
                    content: "Elements 4 and 5 are greater than 3",
                    confidence: 0.95,
                },
                {
                    type: "assumption",
                    content: "Threshold comparison is the appropriate method",
                    confidence: 0.8,
                },
            ],
            confidence: 0.85,
            assumptions: ["Threshold is correctly set", "Data is complete"],
        };

        mockFramework = {
            type: "analytical",
            steps: [
                { id: "data_gathering", type: "gather_data", description: "Collect data", inputs: ["context"], outputs: ["data_points"] },
                { id: "analysis", type: "analyze_patterns", description: "Analyze patterns", inputs: ["data_points"], outputs: ["patterns"] },
                { id: "conclusion", type: "draw_conclusion", description: "Draw conclusion", inputs: ["patterns"], outputs: ["conclusion"] },
            ],
        };

        engine = new ValidationEngine(mockLogger);
    });

    describe("validateReasoning", () => {
        it("should run all validation checks", async () => {
            const results = await engine.validateReasoning(mockReasoningResult, mockFramework);

            expect(results).toHaveLength(6);
            
            const validationTypes = results.map(r => r.type);
            expect(validationTypes).toContain("logic");
            expect(validationTypes).toContain("completeness");
            expect(validationTypes).toContain("bias");
            expect(validationTypes).toContain("confidence");
            expect(validationTypes).toContain("evidence");
            expect(validationTypes).toContain("assumptions");
        });

        it("should pass logic validation for consistent reasoning", async () => {
            const results = await engine.validateReasoning(mockReasoningResult, mockFramework);
            const logicValidation = results.find(r => r.type === "logic");

            expect(logicValidation).toBeDefined();
            expect(logicValidation?.passed).toBe(true);
            expect(logicValidation?.message).toBe("Logical consistency check");
            expect(logicValidation?.details).toHaveProperty("reasoningSteps", 5);
            expect(logicValidation?.details).toHaveProperty("contradictionsFound");
        });

        it("should fail logic validation for contradictory reasoning", async () => {
            const contradictoryResult = {
                ...mockReasoningResult,
                reasoning: [
                    "The result is true",
                    "The analysis contradicts earlier findings",
                    "Both true and false conditions apply",
                    "This is inconsistent with previous conclusions",
                ],
            };

            const results = await engine.validateReasoning(contradictoryResult, mockFramework);
            const logicValidation = results.find(r => r.type === "logic");

            expect(logicValidation?.passed).toBe(false);
        });

        it("should pass completeness validation for adequate reasoning", async () => {
            const results = await engine.validateReasoning(mockReasoningResult, mockFramework);
            const completenessValidation = results.find(r => r.type === "completeness");

            expect(completenessValidation?.passed).toBe(true);
            expect(completenessValidation?.details).toHaveProperty("expectedSteps", 3);
            expect(completenessValidation?.details).toHaveProperty("actualSteps", 5);
            expect(completenessValidation?.details).toHaveProperty("hasConclusion", true);
        });

        it("should fail completeness validation for insufficient reasoning", async () => {
            const incompleteResult = {
                ...mockReasoningResult,
                reasoning: ["Brief analysis"], // Only 1 step, framework expects 3
                evidence: [], // No evidence
            };

            const results = await engine.validateReasoning(incompleteResult, mockFramework);
            const completenessValidation = results.find(r => r.type === "completeness");

            expect(completenessValidation?.passed).toBe(false);
        });

        it("should detect various types of bias", async () => {
            const biasTestCases = [
                {
                    bias: "absolute thinking",
                    reasoning: ["This always works", "Never fails", "All cases are the same"],
                },
                {
                    bias: "overconfidence",
                    reasoning: ["Obviously the correct answer", "Clearly the best option", "Definitely true"],
                },
                {
                    bias: "negative bias",
                    reasoning: ["Probably not going to work", "Unlikely to succeed", "I doubt this approach"],
                },
                {
                    bias: "certainty bias",
                    reasoning: ["Surely this is correct", "Certainly the answer", "Without question true"],
                },
            ];

            for (const testCase of biasTestCases) {
                const biasedResult = {
                    ...mockReasoningResult,
                    reasoning: testCase.reasoning,
                };

                const results = await engine.validateReasoning(biasedResult, mockFramework);
                const biasValidation = results.find(r => r.type === "bias");

                expect(biasValidation?.passed).toBe(false);
                expect(biasValidation?.message).toContain(testCase.bias);
            }
        });

        it("should detect overconfidence without evidence bias", async () => {
            const overconfidentResult = {
                ...mockReasoningResult,
                confidence: 0.95,
                evidence: [{ type: "assumption", content: "Single weak assumption", confidence: 0.5 }], // Only 1 piece of low-quality evidence
            };

            const results = await engine.validateReasoning(overconfidentResult, mockFramework);
            const biasValidation = results.find(r => r.type === "bias");

            expect(biasValidation?.passed).toBe(false);
            expect(biasValidation?.message).toBe("Bias detected: overconfidence without evidence");
        });

        it("should validate confidence threshold", async () => {
            const highConfidenceResult = { ...mockReasoningResult, confidence: 0.9 };
            const lowConfidenceResult = { ...mockReasoningResult, confidence: 0.5 };

            const highResults = await engine.validateReasoning(highConfidenceResult, mockFramework);
            const lowResults = await engine.validateReasoning(lowConfidenceResult, mockFramework);

            const highConfidenceValidation = highResults.find(r => r.type === "confidence");
            const lowConfidenceValidation = lowResults.find(r => r.type === "confidence");

            expect(highConfidenceValidation?.passed).toBe(true);
            expect(lowConfidenceValidation?.passed).toBe(false);
        });

        it("should validate evidence sufficiency", async () => {
            const sufficientEvidenceResult = {
                ...mockReasoningResult,
                evidence: [
                    { type: "fact", content: "Fact 1", confidence: 0.9 },
                    { type: "inference", content: "Inference 1", confidence: 0.8 },
                ],
            };

            const insufficientEvidenceResult = {
                ...mockReasoningResult,
                evidence: [], // No evidence
            };

            const sufficientResults = await engine.validateReasoning(sufficientEvidenceResult, mockFramework);
            const insufficientResults = await engine.validateReasoning(insufficientEvidenceResult, mockFramework);

            const sufficientEvidenceValidation = sufficientResults.find(r => r.type === "evidence");
            const insufficientEvidenceValidation = insufficientResults.find(r => r.type === "evidence");

            expect(sufficientEvidenceValidation?.passed).toBe(true);
            expect(insufficientEvidenceValidation?.passed).toBe(false);
        });

        it("should validate assumptions count", async () => {
            const fewAssumptionsResult = {
                ...mockReasoningResult,
                assumptions: ["Assumption 1", "Assumption 2"],
            };

            const manyAssumptionsResult = {
                ...mockReasoningResult,
                assumptions: ["Assumption 1", "Assumption 2", "Assumption 3", "Assumption 4", "Assumption 5"],
            };

            const fewResults = await engine.validateReasoning(fewAssumptionsResult, mockFramework);
            const manyResults = await engine.validateReasoning(manyAssumptionsResult, mockFramework);

            const fewAssumptionsValidation = fewResults.find(r => r.type === "assumptions");
            const manyAssumptionsValidation = manyResults.find(r => r.type === "assumptions");

            expect(fewAssumptionsValidation?.passed).toBe(true);
            expect(manyAssumptionsValidation?.passed).toBe(false);
        });

        it("should log validation process", async () => {
            await engine.validateReasoning(mockReasoningResult, mockFramework);

            expect(mockLogger.debug).toHaveBeenCalledWith(
                "[ValidationEngine] Starting comprehensive validation",
                expect.objectContaining({
                    frameworkType: "analytical",
                    conclusionType: "string",
                    reasoningStepsCount: 5,
                    evidenceCount: 3,
                })
            );

            expect(mockLogger.info).toHaveBeenCalledWith(
                "[ValidationEngine] Validation completed",
                expect.objectContaining({
                    frameworkType: "analytical",
                    passedValidations: expect.any(Number),
                    totalValidations: 6,
                    overallPassed: expect.any(Boolean),
                })
            );
        });
    });

    describe("suggestImprovements", () => {
        it("should suggest improvements based on failed validations", () => {
            const failedValidations = [
                { type: "logic", passed: false, message: "Logic failed" },
                { type: "bias", passed: false, message: "Bias detected" },
                { type: "confidence", passed: true, message: "Confidence OK" },
            ];

            const improvements = engine.suggestImprovements(mockReasoningResult, failedValidations);

            expect(improvements).toContain("Address validation issues: logic, bias");
        });

        it("should suggest confidence improvements for low confidence", () => {
            const lowConfidenceResult = { ...mockReasoningResult, confidence: 0.5 };

            const improvements = engine.suggestImprovements(lowConfidenceResult, []);

            expect(improvements).toContain("Gather more evidence to increase confidence");
            expect(improvements).toContain("Consider alternative perspectives or approaches");
        });

        it("should suggest assumption reduction for many assumptions", () => {
            const manyAssumptionsResult = {
                ...mockReasoningResult,
                assumptions: ["Assumption 1", "Assumption 2", "Assumption 3", "Assumption 4"],
            };

            const improvements = engine.suggestImprovements(manyAssumptionsResult, []);

            expect(improvements).toContain("Reduce number of assumptions by gathering more facts");
            expect(improvements).toContain("Validate key assumptions with additional evidence");
        });

        it("should suggest evidence improvements for insufficient evidence", () => {
            const lowEvidenceResult = { ...mockReasoningResult, evidence: [] };

            const improvements = engine.suggestImprovements(lowEvidenceResult, []);

            expect(improvements).toContain("Collect more supporting evidence");
            expect(improvements).toContain("Include diverse types of evidence (facts, inferences, external sources)");
        });

        it("should suggest reasoning structure improvements for brief reasoning", () => {
            const briefReasoningResult = {
                ...mockReasoningResult,
                reasoning: ["Brief analysis", "Quick conclusion"],
            };

            const improvements = engine.suggestImprovements(briefReasoningResult, []);

            expect(improvements).toContain("Provide more detailed reasoning steps");
            expect(improvements).toContain("Include intermediate conclusions in reasoning chain");
        });

        it("should suggest bias-specific improvements", () => {
            const biasedResult = {
                ...mockReasoningResult,
                reasoning: ["Obviously correct", "Clearly the best"],
            };

            // Simulate bias detection
            const biasValidations = [
                { type: "bias", passed: false, message: "Bias detected: overconfidence" },
            ];

            const improvements = engine.suggestImprovements(biasedResult, biasValidations);

            expect(improvements).toContain("Address detected bias: overconfidence");
            expect(improvements).toContain("Consider alternative viewpoints and perspectives");
        });
    });

    describe("createValidationSummary", () => {
        it("should create comprehensive validation summary", () => {
            const validations = [
                { type: "logic", passed: true, message: "Logic OK" },
                { type: "completeness", passed: false, message: "Incomplete reasoning" },
                { type: "bias", passed: true, message: "No bias" },
                { type: "confidence", passed: false, message: "Low confidence" },
                { type: "evidence", passed: true, message: "Sufficient evidence" },
                { type: "assumptions", passed: true, message: "Reasonable assumptions" },
            ];

            const summary = engine.createValidationSummary(validations);

            expect(summary).toEqual({
                overallPassed: false,
                passedCount: 4,
                totalCount: 6,
                criticalFailures: ["Incomplete reasoning"], // completeness is critical
                warnings: ["Low confidence"], // confidence is not critical
                score: 4/6, // 4 passed out of 6
            });
        });

        it("should identify critical failures correctly", () => {
            const criticalFailures = [
                { type: "logic", passed: false, message: "Logic failed" },
                { type: "completeness", passed: false, message: "Incomplete" },
            ];

            const nonCriticalFailures = [
                { type: "confidence", passed: false, message: "Low confidence" },
                { type: "bias", passed: false, message: "Bias detected" },
            ];

            const summary = engine.createValidationSummary([...criticalFailures, ...nonCriticalFailures]);

            expect(summary.criticalFailures).toEqual(["Logic failed", "Incomplete"]);
            expect(summary.warnings).toEqual(["Low confidence", "Bias detected"]);
        });

        it("should handle all passed validations", () => {
            const allPassed = [
                { type: "logic", passed: true, message: "Logic OK" },
                { type: "completeness", passed: true, message: "Complete" },
                { type: "bias", passed: true, message: "No bias" },
            ];

            const summary = engine.createValidationSummary(allPassed);

            expect(summary.overallPassed).toBe(true);
            expect(summary.score).toBe(1.0);
            expect(summary.criticalFailures).toEqual([]);
            expect(summary.warnings).toEqual([]);
        });
    });

    describe("checkAdvancedConsistency", () => {
        it("should detect step-by-step contradictions", () => {
            const contradictoryResult = {
                ...mockReasoningResult,
                reasoning: [
                    "The result is true based on analysis",
                    "Further investigation shows the result is false",
                    "Data indicates positive outcome",
                    "However, the outcome is negative",
                ],
            };

            const consistencyCheck = engine.checkAdvancedConsistency(contradictoryResult);

            expect(consistencyCheck.consistent).toBe(false);
            expect(consistencyCheck.issues.length).toBeGreaterThan(0);
            expect(consistencyCheck.confidence).toBeLessThan(1.0);
        });

        it("should detect evidence contradictions", () => {
            const contradictoryEvidenceResult = {
                ...mockReasoningResult,
                evidence: [
                    { type: "fact", content: "Data supports the hypothesis", confidence: 0.9 },
                    { type: "fact", content: "Data refutes the hypothesis", confidence: 0.8 },
                    { type: "inference", content: "Analysis proves the claim", confidence: 0.7 },
                    { type: "inference", content: "Analysis disproves the claim", confidence: 0.6 },
                ],
            };

            const consistencyCheck = engine.checkAdvancedConsistency(contradictoryEvidenceResult);

            expect(consistencyCheck.consistent).toBe(false);
            expect(consistencyCheck.issues).toContain("Evidence 1 contradicts evidence 2");
            expect(consistencyCheck.issues).toContain("Evidence 3 contradicts evidence 4");
        });

        it("should identify low confidence evidence issues", () => {
            const lowConfidenceEvidenceResult = {
                ...mockReasoningResult,
                evidence: [
                    { type: "assumption", content: "Weak assumption 1", confidence: 0.3 },
                    { type: "assumption", content: "Weak assumption 2", confidence: 0.4 },
                    { type: "inference", content: "Weak inference", confidence: 0.2 },
                ],
            };

            const consistencyCheck = engine.checkAdvancedConsistency(lowConfidenceEvidenceResult);

            expect(consistencyCheck.issues).toContain("More than half of evidence has low confidence");
        });

        it("should return consistent result for good reasoning", () => {
            const consistencyCheck = engine.checkAdvancedConsistency(mockReasoningResult);

            expect(consistencyCheck.consistent).toBe(true);
            expect(consistencyCheck.issues).toEqual([]);
            expect(consistencyCheck.confidence).toBeCloseTo(1.0, 1);
        });

        it("should calculate consistency confidence correctly", () => {
            const someIssuesResult = {
                ...mockReasoningResult,
                reasoning: [
                    "Analysis shows positive result",
                    "However, some data indicates negative result", // 1 contradiction
                ],
            };

            const consistencyCheck = engine.checkAdvancedConsistency(someIssuesResult);

            // Should reduce confidence by 0.2 per issue: 1.0 - (1 * 0.2) = 0.8
            expect(consistencyCheck.confidence).toBeCloseTo(0.8, 1);
        });
    });

    describe("integration with reasoning strategies", () => {
        it("should work with different framework types", async () => {
            const frameworks = [
                { type: "logical", steps: [] },
                { type: "analytical", steps: [] },
                { type: "decision_tree", steps: [] },
                { type: "evidence_based", steps: [] },
            ];

            for (const framework of frameworks) {
                const results = await engine.validateReasoning(mockReasoningResult, framework as any);
                
                expect(results).toHaveLength(6);
                expect(results.every(r => typeof r.type === "string")).toBe(true);
                expect(results.every(r => typeof r.passed === "boolean")).toBe(true);
                expect(results.every(r => typeof r.message === "string")).toBe(true);
            }
        });

        it("should handle empty or minimal reasoning gracefully", async () => {
            const minimalResult = {
                conclusion: "Simple answer",
                reasoning: [],
                evidence: [],
                confidence: 0.5,
                assumptions: [],
            };

            const results = await engine.validateReasoning(minimalResult, mockFramework);

            expect(results).toHaveLength(6);
            
            // Should fail most validations due to lack of content
            const failedValidations = results.filter(r => !r.passed);
            expect(failedValidations.length).toBeGreaterThan(3);
        });

        it("should provide actionable validation details", async () => {
            const results = await engine.validateReasoning(mockReasoningResult, mockFramework);

            results.forEach(validation => {
                expect(validation).toHaveProperty("type");
                expect(validation).toHaveProperty("passed");
                expect(validation).toHaveProperty("message");
                expect(validation).toHaveProperty("details");
                
                // Details should provide useful information
                expect(typeof validation.details).toBe("object");
                expect(validation.details).not.toBeNull();
                expect(Object.keys(validation.details).length).toBeGreaterThan(0);
            });
        });
    });
});