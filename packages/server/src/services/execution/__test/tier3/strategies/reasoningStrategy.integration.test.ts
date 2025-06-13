import { describe, it, beforeEach, afterEach, vi, expect } from "vitest";
import { type Logger } from "winston";
import { type ExecutionContext, StrategyType } from "@vrooli/shared";
import { ReasoningStrategy } from "../reasoningStrategy.js";

// Mock LLM service for realistic testing
const mockLLMResponse = vi.fn().callsFake((request: any) => {
    // Simulate realistic LLM responses based on the request
    const content = request.messages[0].content.toLowerCase();
    
    if (content.includes("analyze this reasoning task")) {
        return Promise.resolve({
            content: "Analysis Results:\n1. Task involves data comparison and threshold analysis\n2. Numerical data requires statistical evaluation\n3. Boolean result expected based on comparison criteria\n4. Moderate complexity due to mathematical operations required",
        });
    }
    
    if (content.includes("create an execution plan")) {
        return Promise.resolve({
            content: "Execution Plan:\n1. Load and validate input data array\n2. Apply threshold comparison to each element\n3. Calculate percentage of elements meeting criteria\n4. Determine boolean result based on percentage threshold\n5. Generate comprehensive summary of findings",
        });
    }
    
    if (content.includes("execute this reasoning plan")) {
        return Promise.resolve({
            content: "Execution Results:\nAnalyzed data [1,2,3,4,5] against threshold 3\nElements above threshold: 4, 5 (count: 2)\nElements at/below threshold: 1, 2, 3 (count: 3)\nPercentage above: 40%\nPercentage at/below: 60%\nResult: false (majority does not exceed threshold)\nSummary: Data analysis complete - 60% of values are at or below the specified threshold of 3",
        });
    }
    
    // Default response
    return Promise.resolve({
        content: "Reasoning analysis completed with structured approach and logical conclusions.",
    });
});

describe("ReasoningStrategy Integration Tests", () => {
    let strategy: ReasoningStrategy;
    let mockLogger: Logger;

    beforeEach(() => {
        mockLogger = {
            info: vi.fn(),
            debug: vi.fn(),
            warn: vi.fn(),
            error: vi.fn(),
        } as any;

        strategy = new ReasoningStrategy(mockLogger);
        
        // Mock the LLM service for each test
        if ((strategy as any).phaseExecutor) {
            (strategy as any).phaseExecutor.llmService = {
                executeRequest: mockLLMResponse
            };
        }
    });
    
    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("Complete Execution Flow", () => {
        it("should execute full 4-phase reasoning process successfully", async () => {
            const context: ExecutionContext = {
                stepId: "integration-test-001",
                stepType: "analyze_data_threshold",
                inputs: {
                    data: [1, 2, 3, 4, 5],
                    threshold: 3,
                    requireMajority: true,
                },
                config: {
                    model: "gpt-4o-mini",
                    expectedOutputs: {
                        result: { type: "Boolean" },
                        summary: { type: "Text" },
                        percentage: { type: "Number" },
                        count: { type: "Integer" },
                    },
                },
                resources: {
                    models: [
                        {
                            provider: "openai",
                            model: "gpt-4o-mini",
                            capabilities: ["reasoning", "analysis"],
                            cost: 0.00001,
                            available: true,
                        },
                    ],
                    tools: [
                        {
                            name: "calculator",
                            type: "math",
                            description: "Mathematical calculations",
                            parameters: {},
                        },
                    ],
                    apis: [],
                    credits: 5000,
                    timeLimit: 30000,
                },
                history: {
                    recentSteps: [
                        {
                            stepId: "prev-step-001",
                            strategy: StrategyType.REASONING,
                            result: "success",
                            duration: 5000,
                        },
                    ],
                    totalExecutions: 5,
                    successRate: 0.8,
                },
                constraints: {
                    maxTokens: 2000,
                    maxTime: 30000,
                    maxCost: 1.0,
                    requiredConfidence: 0.8,
                },
            };

            const result = await strategy.execute(context);

            // Verify successful execution
            expect(result.success).toBe(true);
            expect(result).toHaveProperty("outputs");
            expect(result).toHaveProperty("metadata");
            expect(result).toHaveProperty("resourceUsage");
            expect(result).toHaveProperty("confidence");
            expect(result).toHaveProperty("performanceScore");

            // Verify metadata structure
            expect(result.metadata).toHaveProperty("strategy", "reasoning");
            expect(result.metadata).toHaveProperty("version", "2.0.0-enhanced");
            expect(result.metadata).toHaveProperty("executionPhases");
            expect(result.metadata.executionPhases).toHaveProperty("analysis");
            expect(result.metadata.executionPhases).toHaveProperty("planning");
            expect(result.metadata.executionPhases).toHaveProperty("refinement");

            // Verify resource usage tracking
            expect(result.resourceUsage).toHaveProperty("tokens");
            expect(result.resourceUsage).toHaveProperty("apiCalls");
            expect(result.resourceUsage).toHaveProperty("computeTime");
            expect(result.resourceUsage).toHaveProperty("cost");

            expect(result.resourceUsage.tokens).toBeGreaterThan(0);
            expect(result.resourceUsage.apiCalls).toBeGreaterThan(0);
            expect(result.resourceUsage.computeTime).toBeGreaterThan(0);
            expect(result.resourceUsage.cost).toBeGreaterThan(0);

            // Verify performance metrics
            expect(result.confidence).toBeGreaterThan(0);
            expect(result.confidence).toBeLessThanOrEqual(1);
            expect(result.performanceScore).toBeGreaterThan(0);
            expect(result.performanceScore).toBeLessThanOrEqual(1);
        });

        it("should handle missing inputs by generating them intelligently", async () => {
            const contextMissingInputs: ExecutionContext = {
                stepId: "missing-inputs-test",
                stepType: "analyze_incomplete_data",
                inputs: {
                    data: [1, 2, 3], // Missing threshold
                },
                config: {
                    expectedInputs: {
                        data: { type: "Array" },
                        threshold: { type: "Number", defaultValue: 2.5 },
                        includeStats: { type: "Boolean" },
                        description: { type: "Text" },
                    },
                    expectedOutputs: {
                        result: { type: "Boolean" },
                        analysis: { type: "Text" },
                    },
                },
                resources: {
                    models: [],
                    tools: [],
                    apis: [],
                    credits: 3000,
                },
                history: {
                    recentSteps: [],
                    totalExecutions: 0,
                    successRate: 1.0,
                },
                constraints: {},
            };

            const result = await strategy.execute(contextMissingInputs);

            expect(result.success).toBe(true);
            
            // Verify that the strategy handled missing inputs
            expect(mockLogger.debug).toHaveBeenCalledWith(
                "[LegacyInputHandler] Generating missing inputs",
                expect.objectContaining({
                    stepId: "missing-inputs-test",
                    missingInputs: expect.arrayContaining(["threshold", "includeStats", "description"]),
                })
            );
        });

        it("should generate appropriate outputs for different types", async () => {
            const context: ExecutionContext = {
                stepId: "output-types-test",
                stepType: "comprehensive_analysis",
                inputs: {
                    numbers: [10, 20, 30],
                    text: "sample data",
                    flag: true,
                },
                config: {
                    expectedOutputs: {
                        booleanResult: { type: "Boolean" },
                        textSummary: { type: "Text" },
                        numericScore: { type: "Number" },
                        integerCount: { type: "Integer" },
                        arraySteps: { type: "Array" },
                        objectMetadata: { type: "Object" },
                    },
                },
                resources: {
                    models: [],
                    tools: [],
                    apis: [],
                    credits: 4000,
                },
                history: {
                    recentSteps: [],
                    totalExecutions: 0,
                    successRate: 1.0,
                },
                constraints: {},
            };

            const result = await strategy.execute(context);

            expect(result.success).toBe(true);
            expect(result.outputs).toHaveProperty("booleanResult");
            expect(result.outputs).toHaveProperty("textSummary");
            expect(result.outputs).toHaveProperty("numericScore");
            expect(result.outputs).toHaveProperty("integerCount");
            expect(result.outputs).toHaveProperty("arraySteps");
            expect(result.outputs).toHaveProperty("objectMetadata");

            // Verify output types
            expect(typeof result.outputs.booleanResult).toBe("boolean");
            expect(typeof result.outputs.textSummary).toBe("string");
            expect(typeof result.outputs.numericScore).toBe("number");
            expect(typeof result.outputs.integerCount).toBe("number");
            expect(Number.isInteger(result.outputs.integerCount as number)).toBe(true);
            expect(Array.isArray(result.outputs.arraySteps)).toBe(true);
            expect(typeof result.outputs.objectMetadata).toBe("object");
        });

        it("should apply comprehensive validation and provide improvement suggestions", async () => {
            const context: ExecutionContext = {
                stepId: "validation-test",
                stepType: "validate_reasoning_quality",
                inputs: {
                    hypothesis: "Most users prefer option A",
                    evidence: ["Survey data", "User feedback", "Analytics"],
                },
                config: {
                    expectedOutputs: {
                        validated: { type: "Boolean" },
                        confidence: { type: "Number" },
                        issues: { type: "Array" },
                    },
                },
                resources: {
                    models: [],
                    tools: [],
                    apis: [],
                    credits: 3500,
                },
                history: {
                    recentSteps: [],
                    totalExecutions: 0,
                    successRate: 1.0,
                },
                constraints: {
                    requiredConfidence: 0.9,
                },
            };

            const result = await strategy.execute(context);

            expect(result.success).toBe(true);
            
            // Check that validation was applied
            expect(mockLogger.info).toHaveBeenCalledWith(
                "[FourPhaseExecutor] Validation and refinement completed",
                expect.objectContaining({
                    stepId: "validation-test",
                    validationScore: sinon.match.number,
                    improvementsCount: sinon.match.number,
                    overallPassed: sinon.match.bool,
                })
            );
        });

        it("should handle errors gracefully and provide detailed error information", async () => {
            // Mock the LLM service to throw an error
            const mockError = new Error("LLM service temporarily unavailable");
            const mockLLMService = {
                executeRequest: vi.fn().rejects(mockError),
            };
            
            // Replace the LLM service with the failing mock
            (strategy as any).phaseExecutor.llmService = mockLLMService;

            const context: ExecutionContext = {
                stepId: "error-handling-test",
                stepType: "analyze_with_error",
                inputs: { data: "test" },
                config: {},
                resources: {
                    models: [],
                    tools: [],
                    apis: [],
                    credits: 1000,
                },
                history: {
                    recentSteps: [],
                    totalExecutions: 0,
                    successRate: 1.0,
                },
                constraints: {},
            };

            const result = await strategy.execute(context);

            expect(result.success).toBe(false);
            expect(result).toHaveProperty("error");
            expect(result.error).toHaveProperty("message", "LLM service temporarily unavailable");
            expect(result.error).toHaveProperty("type", "Error");
            expect(result.error).toHaveProperty("strategy", "reasoning");

            // Should still provide resource usage information
            expect(result).toHaveProperty("resourceUsage");
            expect(result).toHaveProperty("confidence", 0.0);
            expect(result).toHaveProperty("performanceScore", 0.0);

            // Should log the error appropriately
            expect(mockLogger.error).toHaveBeenCalledWith(
                "[ReasoningStrategy] 4-phase execution failed",
                expect.objectContaining({
                    stepId: "error-handling-test",
                    error: "LLM service temporarily unavailable",
                })
            );
        });
    });

    describe("Resource Estimation Accuracy", () => {
        it("should provide realistic resource estimates that match actual usage", async () => {
            const context: ExecutionContext = {
                stepId: "resource-estimation-test",
                stepType: "analyze_complex_data",
                inputs: {
                    dataset: Array.from({ length: 100 }, (_, i) => i),
                    parameters: { threshold: 50, mode: "statistical" },
                },
                config: {
                    expectedOutputs: {
                        result: { type: "Boolean" },
                        statistics: { type: "Object" },
                        summary: { type: "Text" },
                        recommendations: { type: "Array" },
                    },
                },
                resources: {
                    models: [],
                    tools: [
                        { name: "stats", type: "analysis", description: "Statistical analysis", parameters: {} },
                        { name: "chart", type: "visualization", description: "Chart generation", parameters: {} },
                    ],
                    apis: [],
                    credits: 10000,
                },
                history: {
                    recentSteps: Array.from({ length: 5 }, (_, i) => ({
                        stepId: `step-${i}`,
                        strategy: StrategyType.REASONING,
                        result: "success" as const,
                        duration: 2000,
                    })),
                    totalExecutions: 20,
                    successRate: 0.95,
                },
                constraints: {
                    maxTokens: 4000,
                    maxTime: 60000,
                    requiredConfidence: 0.85,
                },
            };

            // Get estimation
            const estimation = strategy.estimateResources(context);

            // Execute actual task
            const result = await strategy.execute(context);

            // Compare estimation vs actual usage
            expect(result.success).toBe(true);
            expect(result.resourceUsage.tokens).toBeGreaterThan(0);
            expect(result.resourceUsage.apiCalls).toBeGreaterThanOrEqual(4); // 4-phase minimum
            expect(result.resourceUsage.computeTime).toBeGreaterThan(0);

            // Estimation should be in reasonable range of actual usage
            const tokenRatio = result.resourceUsage.tokens / estimation.tokens;
            const timeRatio = result.resourceUsage.computeTime / estimation.computeTime;
            const callRatio = result.resourceUsage.apiCalls / estimation.apiCalls;

            expect(tokenRatio).toBeGreaterThan(0.5); // Not more than 2x underestimate
            expect(tokenRatio).toBeLessThan(3.0);    // Not more than 3x overestimate
            expect(timeRatio).toBeGreaterThan(0.1);  // Time estimation can vary more
            expect(timeRatio).toBeLessThan(5.0);
            expect(callRatio).toBeGreaterThan(0.5);  // API calls should be close
            expect(callRatio).toBeLessThan(2.0);
        });

        it("should scale estimation with context complexity", () => {
            const simpleContext: ExecutionContext = {
                stepId: "simple",
                stepType: "basic_check",
                inputs: { value: 5 },
                config: { expectedOutputs: { result: { type: "Boolean" } } },
                resources: { models: [], tools: [], apis: [], credits: 1000 },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                constraints: {},
            };

            const complexContext: ExecutionContext = {
                stepId: "complex",
                stepType: "comprehensive_analysis_with_optimization",
                inputs: Object.fromEntries(Array.from({ length: 20 }, (_, i) => [`input${i}`, `value${i}`])),
                config: {
                    expectedOutputs: Object.fromEntries(Array.from({ length: 10 }, (_, i) => [`output${i}`, { type: "Text" }])),
                },
                resources: {
                    models: [],
                    tools: Array.from({ length: 5 }, (_, i) => ({
                        name: `tool${i}`,
                        type: "analysis",
                        description: `Tool ${i}`,
                        parameters: {},
                    })),
                    apis: [],
                    credits: 10000,
                },
                history: {
                    recentSteps: Array.from({ length: 20 }, (_, i) => ({
                        stepId: `step${i}`,
                        strategy: StrategyType.REASONING,
                        result: "success" as const,
                        duration: 3000,
                    })),
                    totalExecutions: 100,
                    successRate: 0.9,
                },
                constraints: {
                    maxTokens: 8000,
                    maxTime: 120000,
                    requiredConfidence: 0.95,
                },
            };

            const simpleEstimate = strategy.estimateResources(simpleContext);
            const complexEstimate = strategy.estimateResources(complexContext);

            expect(complexEstimate.tokens).toBeGreaterThan(simpleEstimate.tokens);
            expect(complexEstimate.apiCalls).toBeGreaterThan(simpleEstimate.apiCalls);
            expect(complexEstimate.computeTime).toBeGreaterThan(simpleEstimate.computeTime);
            expect(complexEstimate.cost).toBeGreaterThan(simpleEstimate.cost);
        });
    });

    // Learning and Performance Tracking now happens through event-driven architecture
    // Tests removed: "should learn from feedback and track performance"

    describe("Strategy Selection Accuracy", () => {
        it("should correctly identify when it can handle different step types", () => {
            const testCases = [
                { stepType: "analyze_data", expected: true },
                { stepType: "reason_about_options", expected: true },
                { stepType: "evaluate_performance", expected: true },
                { stepType: "decide_best_approach", expected: true },
                { stepType: "validate_hypothesis", expected: true },
                { stepType: "calculate_metrics", expected: true },
                { stepType: "optimize_parameters", expected: true },
                { stepType: "justify_decision", expected: true },
                { stepType: "explain_reasoning", expected: true },
                { stepType: "format_output", expected: false },
                { stepType: "save_to_database", expected: false },
                { stepType: "send_notification", expected: false },
                { stepType: "upload_file", expected: false },
            ];

            testCases.forEach(({ stepType, expected }) => {
                const canHandle = strategy.canHandle(stepType);
                expect(canHandle).toBe(expected);
            });
        });

        it("should respect explicit strategy configuration", () => {
            expect(strategy.canHandle("any_step_type", { strategy: "reasoning" })).toBe(true);
            expect(strategy.canHandle("format_output", { strategy: "reasoning" })).toBe(true);
        });
    });

    describe("Reasoning Framework Selection", () => {
        it("should select appropriate reasoning frameworks for different tasks", () => {
            const testCases = [
                { stepType: "decide_best_option", expectedFramework: "decision_tree" },
                { stepType: "choose_algorithm", expectedFramework: "decision_tree" },
                { stepType: "analyze_performance", expectedFramework: "analytical" },
                { stepType: "evaluate_results", expectedFramework: "analytical" },
                { stepType: "validate_hypothesis", expectedFramework: "evidence_based" },
                { stepType: "verify_claims", expectedFramework: "evidence_based" },
                { stepType: "reason_logically", expectedFramework: "logical" },
                { stepType: "process_data", expectedFramework: "logical" },
            ];

            testCases.forEach(({ stepType, expectedFramework }) => {
                const context = {
                    stepId: "test",
                    stepType,
                    inputs: {},
                    config: {},
                    resources: { models: [], tools: [], apis: [], credits: 1000 },
                    history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                    constraints: {},
                };

                const framework = (strategy as any).selectReasoningFramework(context);
                expect(framework.type).toBe(expectedFramework);
            });
        });
    });
});