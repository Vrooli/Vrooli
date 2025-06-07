import { describe, it, beforeEach, vi, expect } from "vitest";
import { type Logger } from "winston";
import { type ExecutionContext } from "@vrooli/shared";
import { FourPhaseExecutor } from "../fourPhaseExecutor.js";

describe("FourPhaseExecutor", () => {
    let executor: FourPhaseExecutor;
    let mockLogger: Logger;
    let mockLLMService: any;
    let mockValidationEngine: any;
    let baseContext: ExecutionContext;

    beforeEach(() => {
        mockLogger = {
            info: vi.fn(),
            debug: vi.fn(),
            warn: vi.fn(),
            error: vi.fn(),
        } as any;

        mockLLMService = {
            executeRequest: vi.fn(),
        };

        mockValidationEngine = {
            validateReasoning: vi.fn(),
            createValidationSummary: vi.fn(),
            suggestImprovements: vi.fn(),
        };

        baseContext = {
            stepId: "test-step-123",
            stepType: "analyze_data",
            inputs: {
                data: [1, 2, 3, 4, 5],
                threshold: 3,
            },
            config: {
                model: "gpt-4o-mini",
                expectedOutputs: {
                    result: { type: "Boolean" },
                    summary: { type: "Text" },
                },
            },
            resources: {
                models: [],
                tools: [],
                apis: [],
                credits: 5000,
                timeLimit: 30000,
            },
            history: {
                recentSteps: [],
                totalExecutions: 0,
                successRate: 1.0,
            },
            constraints: {
                maxTokens: 2000,
                maxTime: 30000,
            },
        };

        executor = new FourPhaseExecutor(mockLogger);
        
        // Replace mocked services
        (executor as any).llmService = mockLLMService;
        (executor as any).validationEngine = mockValidationEngine;
    });

    describe("Phase 1: analyzeTask", () => {
        beforeEach(() => {
            mockLLMService.executeRequest.mockResolvedValue({
                content: "This task requires analyzing numerical data against a threshold. Key insights:\n1. Data contains 5 elements\n2. Threshold is set to 3\n3. Analysis will involve comparison operations\n4. Result should be boolean indicating if data meets criteria",
            });
        });

        it("should analyze task and extract insights", async () => {
            const result = await executor.analyzeTask(baseContext);

            expect(mockLLMService.executeRequest).toHaveBeenCalledWith(
                expect.objectContaining({
                    model: "gpt-4o-mini",
                    messages: expect.arrayContaining([
                        expect.objectContaining({
                            role: "user",
                            content: expect.stringContaining("Analyze this reasoning task"),
                        }),
                    ]),
                    systemMessage: "Analyze this task to understand its requirements and complexity.",
                    maxTokens: 500,
                    temperature: 0.3,
                }),
                expect.objectContaining({
                    maxCredits: 5000,
                    maxTokens: 500,
                    maxTime: 10000,
                    tools: [],
                })
            );

            expect(result).toHaveProperty("insights");
            expect(result).toHaveProperty("creditsUsed", 100);
            expect(result).toHaveProperty("toolCallsCount", 1);
            expect(Array.isArray(result.insights)).toBe(true);
            expect(result.insights.length).toBeGreaterThan(0);
        });

        it("should build appropriate analysis prompt", async () => {
            await executor.analyzeTask(baseContext);

            const callArgs = mockLLMService.executeRequest.mock.calls[0][0];
            const prompt = callArgs.messages[0].content;

            expect(prompt).toContain("Analyze this reasoning task:");
            expect(prompt).toContain("Step Type: analyze_data");
            expect(prompt).toContain("Step ID: test-step-123");
            expect(prompt).toContain("Available Inputs:");
            expect(prompt).toContain("data: [1,2,3,4,5]");
            expect(prompt).toContain("threshold: 3");
            expect(prompt).toContain("Expected Outputs: result, summary");
            expect(prompt).toContain("Please analyze:");
            expect(prompt).toContain("1. Task complexity and requirements");
            expect(prompt).toContain("2. Available information sufficiency");
            expect(prompt).toContain("3. Potential challenges or considerations");
            expect(prompt).toContain("4. Recommended reasoning approach");
        });

        it("should extract insights from structured responses", async () => {
            mockLLMService.executeRequest.mockResolvedValue({
                content: "Analysis Results:\n• Task is moderately complex\n• Sufficient data available\n• May require statistical analysis\n• Recommend threshold comparison approach",
            });

            const result = await executor.analyzeTask(baseContext);

            expect(result.insights).toContain("Task is moderately complex");
            expect(result.insights).toContain("Sufficient data available");
            expect(result.insights).toContain("May require statistical analysis");
            expect(result.insights).toContain("Recommend threshold comparison approach");
        });

        it("should handle unstructured responses with sentence fallback", async () => {
            mockLLMService.executeRequest.mockResolvedValue({
                content: "This task requires careful analysis of numerical data. The threshold comparison is straightforward. Results should be clear and actionable. Statistical methods may be helpful.",
            });

            const result = await executor.analyzeTask(baseContext);

            expect(result.insights.length).toBeGreaterThan(0);
            expect(result.insights.length).toBeLessThanOrEqual(4); // Fallback takes first 4 sentences
        });

        it("should log analysis phase appropriately", async () => {
            await executor.analyzeTask(baseContext);

            expect(mockLogger.info).toHaveBeenCalledWith(
                "[FourPhaseExecutor] Analyzing task with AI reasoning",
                {
                    stepId: "test-step-123",
                    stepType: "analyze_data",
                }
            );
        });
    });

    describe("Phase 2: planReasoningApproach", () => {
        const mockInsights = [
            "Task requires data comparison",
            "Threshold-based analysis needed",
            "Boolean result expected",
        ];

        beforeEach(() => {
            mockLLMService.executeRequest.mockResolvedValue({
                content: "Execution Plan:\n1. Load and validate input data\n2. Compare each element against threshold\n3. Determine overall result based on comparison criteria\n4. Generate clear summary of findings",
            });
        });

        it("should create execution plan based on insights", async () => {
            const result = await executor.planReasoningApproach(baseContext, mockInsights);

            expect(mockLLMService.executeRequest).toHaveBeenCalledWith(
                expect.objectContaining({
                    model: "gpt-4o-mini",
                    messages: expect.arrayContaining([
                        expect.objectContaining({
                            role: "user",
                            content: expect.stringContaining("Create an execution plan based on this analysis"),
                        }),
                    ]),
                    systemMessage: "Create a structured execution plan for this reasoning task.",
                    maxTokens: 750,
                    temperature: 0.3,
                }),
                expect.objectContaining({
                    maxCredits: 5000,
                    maxTokens: 750,
                    maxTime: 15000,
                })
            );

            expect(result).toHaveProperty("executionPlan");
            expect(result).toHaveProperty("creditsUsed", 150);
            expect(result).toHaveProperty("toolCallsCount", 1);
            expect(typeof result.executionPlan).toBe("string");
            expect(result.executionPlan.length).toBeGreaterThan(0);
        });

        it("should build planning prompt with insights", async () => {
            await executor.planReasoningApproach(baseContext, mockInsights);

            const callArgs = mockLLMService.executeRequest.mock.calls[0][0];
            const prompt = callArgs.messages[0].content;

            expect(prompt).toContain("Create an execution plan based on this analysis:");
            expect(prompt).toContain("Task: analyze_data (test-step-123)");
            expect(prompt).toContain("Analysis Insights:");
            expect(prompt).toContain("1. Task requires data comparison");
            expect(prompt).toContain("2. Threshold-based analysis needed");
            expect(prompt).toContain("3. Boolean result expected");
            expect(prompt).toContain("Plan Requirements:");
            expect(prompt).toContain("1. Structured step-by-step approach");
            expect(prompt).toContain("2. Reasoning methodology to use");
            expect(prompt).toContain("3. Expected deliverables");
            expect(prompt).toContain("4. Success criteria");
        });

        it("should log planning phase appropriately", async () => {
            await executor.planReasoningApproach(baseContext, mockInsights);

            expect(mockLogger.info).toHaveBeenCalledWith(
                "[FourPhaseExecutor] Planning reasoning approach",
                {
                    stepId: "test-step-123",
                    insights: mockInsights,
                }
            );
        });
    });

    describe("Phase 3: executeReasoningLogic", () => {
        const mockExecutionPlan = "1. Analyze data 2. Compare with threshold 3. Generate result";

        beforeEach(() => {
            mockLLMService.executeRequest.mockResolvedValue({
                content: "Execution Results:\nData analysis: [1,2,3,4,5] compared to threshold 3\nElements above threshold: 4, 5 (2 elements)\nElements at or below threshold: 1, 2, 3 (3 elements)\nConclusion: Most elements (60%) are at or below threshold\nResult: false",
            });
        });

        it("should execute reasoning logic based on plan", async () => {
            const result = await executor.executeReasoningLogic(baseContext, mockExecutionPlan);

            expect(mockLLMService.executeRequest).toHaveBeenCalledWith(
                expect.objectContaining({
                    model: "gpt-4o-mini",
                    messages: expect.arrayContaining([
                        expect.objectContaining({
                            role: "user",
                            content: expect.stringContaining("Execute this reasoning plan"),
                        }),
                    ]),
                    systemMessage: "Execute the reasoning plan and provide structured results.",
                    maxTokens: 1000,
                    temperature: 0.3,
                }),
                expect.objectContaining({
                    maxCredits: 5000,
                    maxTokens: 1000,
                    maxTime: 20000,
                })
            );

            expect(result).toHaveProperty("outputs");
            expect(result).toHaveProperty("creditsUsed", 300);
            expect(result).toHaveProperty("toolCallsCount", 2);
            expect(typeof result.outputs).toBe("object");
        });

        it("should build execution prompt with plan and context", async () => {
            await executor.executeReasoningLogic(baseContext, mockExecutionPlan);

            const callArgs = mockLLMService.executeRequest.mock.calls[0][0];
            const prompt = callArgs.messages[0].content;

            expect(prompt).toContain("Execute this reasoning plan:");
            expect(prompt).toContain("Plan: " + mockExecutionPlan);
            expect(prompt).toContain("Available Context:");
            expect(prompt).toContain("data: [1,2,3,4,5]");
            expect(prompt).toContain("threshold: 3");
            expect(prompt).toContain("Instructions:");
            expect(prompt).toContain("1. Follow the execution plan systematically");
            expect(prompt).toContain("2. Apply logical reasoning to the available context");
            expect(prompt).toContain("3. Generate appropriate outputs based on the plan");
            expect(prompt).toContain("4. Provide reasoning for your conclusions");
        });

        it("should parse outputs based on expected output configuration", async () => {
            const result = await executor.executeReasoningLogic(baseContext, mockExecutionPlan);

            // Should have outputs for expected output types
            expect(result.outputs).toHaveProperty("result");
            expect(result.outputs).toHaveProperty("summary");
        });

        it("should handle contexts with no expected outputs", async () => {
            const contextNoOutputs = {
                ...baseContext,
                config: {
                    ...baseContext.config,
                    expectedOutputs: {},
                },
            };

            const result = await executor.executeReasoningLogic(contextNoOutputs, mockExecutionPlan);

            expect(result.outputs).toHaveProperty("result");
            expect(result.outputs).toHaveProperty("reasoning");
            expect(result.outputs).toHaveProperty("timestamp");
            expect(result.outputs).toHaveProperty("stepId", "test-step-123");
        });

        it("should extract Boolean outputs correctly", async () => {
            mockLLMService.executeRequest.mockResolvedValue({
                content: "Analysis complete. The result is true based on the threshold comparison.",
            });

            const result = await executor.executeReasoningLogic(baseContext, mockExecutionPlan);

            expect(result.outputs.result).toBe(true);
        });

        it("should extract Number outputs correctly", async () => {
            const contextWithNumbers = {
                ...baseContext,
                config: {
                    ...baseContext.config,
                    expectedOutputs: {
                        score: { type: "Number" },
                        count: { type: "Integer" },
                    },
                },
            };

            mockLLMService.executeRequest.mockResolvedValue({
                content: "Analysis shows a score of 85.7 and count of 42 items processed.",
            });

            const result = await executor.executeReasoningLogic(contextWithNumbers, mockExecutionPlan);

            expect(result.outputs.score).toBe(85.7);
            expect(result.outputs.count).toBe(42);
        });

        it("should log execution phase appropriately", async () => {
            await executor.executeReasoningLogic(baseContext, mockExecutionPlan);

            expect(mockLogger.info).toHaveBeenCalledWith(
                "[FourPhaseExecutor] Executing reasoning logic",
                {
                    stepId: "test-step-123",
                }
            );
        });
    });

    describe("Phase 4: refineAndValidate", () => {
        const mockExecutionResult = {
            outputs: {
                result: true,
                summary: "Data analysis completed successfully",
            },
        };
        const mockExecutionPlan = "Systematic data analysis plan";

        beforeEach(() => {
            mockValidationEngine.validateReasoning.mockResolvedValue([
                { type: "logic", passed: true, message: "Logical consistency check" },
                { type: "completeness", passed: true, message: "Reasoning completeness check" },
                { type: "bias", passed: true, message: "No bias detected" },
            ]);

            mockValidationEngine.createValidationSummary.mockReturnValue({
                overallPassed: true,
                passedCount: 3,
                totalCount: 3,
                criticalFailures: [],
                warnings: [],
                score: 1.0,
            });

            mockValidationEngine.suggestImprovements.mockReturnValue([
                "Consider adding more evidence",
                "Validate assumptions with external sources",
            ]);
        });

        it("should apply comprehensive validation and refinement", async () => {
            const result = await executor.refineAndValidate(
                baseContext,
                mockExecutionResult,
                mockExecutionPlan
            );

            expect(mockValidationEngine.validateReasoning).toHaveBeenCalledWith(
                expect.objectContaining({
                    conclusion: mockExecutionResult.outputs,
                    reasoning: expect.arrayContaining([mockExecutionPlan]),
                    evidence: [],
                    confidence: 0.85,
                    assumptions: [],
                }),
                expect.objectContaining({
                    type: "logical",
                    steps: [],
                })
            );

            expect(result).toHaveProperty("outputs");
            expect(result).toHaveProperty("refinements");
            expect(result).toHaveProperty("validationResults");
            expect(result).toHaveProperty("validationSummary");
            expect(result).toHaveProperty("creditsUsed", 100);
            expect(result).toHaveProperty("toolCallsCount", 0);
        });

        it("should apply intelligent refinements with validation metadata", async () => {
            const result = await executor.refineAndValidate(
                baseContext,
                mockExecutionResult,
                mockExecutionPlan
            );

            // Check that refinements include validation improvements
            expect(result.refinements).toEqual([
                "Outputs are contextually appropriate",
                "All required fields are populated",
                "Results align with input parameters",
                "AI reasoning has been applied for optimization",
                "Consider adding more evidence",
                "Validate assumptions with external sources",
            ]);

            // Check that outputs have validation metadata
            const outputKeys = Object.keys(result.outputs);
            for (const key of outputKeys) {
                const output = result.outputs[key];
                if (typeof output === "object" && output !== null) {
                    expect(output).toHaveProperty("_validationPassed", true);
                    expect(output).toHaveProperty("_confidence", 1.0);
                    expect(output).toHaveProperty("_improvements");
                }
            }
        });

        it("should handle validation failures appropriately", async () => {
            mockValidationEngine.validateReasoning.mockResolvedValue([
                { type: "logic", passed: false, message: "Logical inconsistency detected" },
                { type: "completeness", passed: true, message: "Reasoning completeness check" },
                { type: "bias", passed: false, message: "Overconfidence bias detected" },
            ]);

            mockValidationEngine.createValidationSummary.mockReturnValue({
                overallPassed: false,
                passedCount: 1,
                totalCount: 3,
                criticalFailures: ["Logical inconsistency detected"],
                warnings: ["Overconfidence bias detected"],
                score: 0.33,
            });

            const result = await executor.refineAndValidate(
                baseContext,
                mockExecutionResult,
                mockExecutionPlan
            );

            expect(result.validationSummary.overallPassed).toBe(false);
            expect(result.validationSummary.score).toBe(0.33);

            // Check that confidence is reduced for failed validation
            const outputKeys = Object.keys(result.outputs);
            for (const key of outputKeys) {
                const output = result.outputs[key];
                if (typeof output === "object" && output !== null) {
                    expect(output).toHaveProperty("_validationPassed", false);
                    expect(output).toHaveProperty("_confidence", 0.33);
                }
            }
        });

        it("should log validation and refinement completion", async () => {
            await executor.refineAndValidate(
                baseContext,
                mockExecutionResult,
                mockExecutionPlan
            );

            expect(mockLogger.info).toHaveBeenCalledWith(
                "[FourPhaseExecutor] Validation and refinement completed",
                {
                    stepId: "test-step-123",
                    validationScore: 1.0,
                    improvementsCount: 2,
                    overallPassed: true,
                }
            );
        });

        it("should preserve original outputs while adding metadata", async () => {
            const result = await executor.refineAndValidate(
                baseContext,
                mockExecutionResult,
                mockExecutionPlan
            );

            // Original structure should be preserved
            expect(result.outputs).toHaveProperty("result");
            expect(result.outputs).toHaveProperty("summary");

            // But with added metadata for object values
            if (typeof result.outputs.result === "object") {
                expect(result.outputs.result).toHaveProperty("_stepId", "test-step-123");
                expect(result.outputs.result).toHaveProperty("_strategy", "reasoning");
            }
        });
    });

    describe("integration between phases", () => {
        it("should maintain data flow consistency across all phases", async () => {
            // Mock all phases
            mockLLMService.executeRequest
                .mockResolvedValueOnce({
                    content: "Analysis: Task requires comparison logic",
                })
                .mockResolvedValueOnce({
                    content: "Plan: 1. Compare 2. Decide 3. Report",
                })
                .mockResolvedValueOnce({
                    content: "Execution: Comparison complete, result is true",
                });

            mockValidationEngine.validateReasoning.mockResolvedValue([]);
            mockValidationEngine.createValidationSummary.mockReturnValue({
                overallPassed: true,
                score: 1.0,
            });
            mockValidationEngine.suggestImprovements.mockReturnValue([]);

            // Execute all phases in sequence
            const analysisResult = await executor.analyzeTask(baseContext);
            const planningResult = await executor.planReasoningApproach(baseContext, analysisResult.insights);
            const executionResult = await executor.executeReasoningLogic(baseContext, planningResult.executionPlan);
            const refinementResult = await executor.refineAndValidate(
                baseContext,
                executionResult,
                planningResult.executionPlan
            );

            // Verify data flows correctly
            expect(analysisResult.insights).toBeDefined();
            expect(planningResult.executionPlan).toBeDefined();
            expect(executionResult.outputs).toBeDefined();
            expect(refinementResult.outputs).toBeDefined();
            expect(refinementResult.validationResults).toBeDefined();
        });

        it("should accumulate resource usage correctly across phases", async () => {
            mockLLMService.executeRequest.mockResolvedValue({ content: "test response" });
            mockValidationEngine.validateReasoning.mockResolvedValue([]);
            mockValidationEngine.createValidationSummary.mockReturnValue({ overallPassed: true, score: 1.0 });
            mockValidationEngine.suggestImprovements.mockReturnValue([]);

            const analysisResult = await executor.analyzeTask(baseContext);
            const planningResult = await executor.planReasoningApproach(baseContext, analysisResult.insights);
            const executionResult = await executor.executeReasoningLogic(baseContext, planningResult.executionPlan);
            const refinementResult = await executor.refineAndValidate(baseContext, executionResult, planningResult.executionPlan);

            // Verify each phase returns expected resource usage
            expect(analysisResult.creditsUsed).toBe(100);
            expect(analysisResult.toolCallsCount).toBe(1);

            expect(planningResult.creditsUsed).toBe(150);
            expect(planningResult.toolCallsCount).toBe(1);

            expect(executionResult.creditsUsed).toBe(300);
            expect(executionResult.toolCallsCount).toBe(2);

            expect(refinementResult.creditsUsed).toBe(100);
            expect(refinementResult.toolCallsCount).toBe(0);

            // Total should be: 100 + 150 + 300 + 100 = 650 credits, 1 + 1 + 2 + 0 = 4 tool calls
        });
    });
});