import { describe, it, beforeEach, afterEach, vi, expect } from "vitest";
import { type Logger } from "winston";
import { type ExecutionContext, StrategyType } from "@vrooli/shared";
import { ReasoningStrategy } from "../reasoningStrategy.js";

describe("ReasoningStrategy", () => {
    let strategy: ReasoningStrategy;
    let mockLogger: Logger;
    let mockContext: ExecutionContext;

    beforeEach(() => {
        mockLogger = {
            info: vi.fn(),
            debug: vi.fn(),
            warn: vi.fn(),
            error: vi.fn(),
        } as any;

        mockContext = {
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
                    analysis: { type: "Text" },
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
                maxCost: 1.0,
                requiredConfidence: 0.8,
            },
        };

        strategy = new ReasoningStrategy(mockLogger);
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("constructor", () => {
        it("should initialize with correct properties", () => {
            expect(strategy.type).toBe(StrategyType.REASONING);
            expect(strategy.name).toBe("ReasoningStrategy");
            expect(strategy.version).toBe("2.0.0");
        });

        it("should initialize all modular components", () => {
            // Verify that all components are properly initialized
            expect(strategy).toHaveProperty("inputHandler");
            expect(strategy).toHaveProperty("phaseExecutor");
            expect(strategy).toHaveProperty("outputGenerator");
            expect(strategy).toHaveProperty("costEstimator");
        });
    });

    describe("canHandle", () => {
        it("should return true for explicit reasoning strategy request", () => {
            const config = { strategy: "reasoning" };
            expect(strategy.canHandle("any_step", config)).to.equal(true);
        });

        it("should return true for reasoning-related step types", () => {
            const reasoningSteps = [
                "analyze_data",
                "reason_about_options",
                "evaluate_results",
                "assess_quality",
                "decide_next_step",
                "compare_alternatives",
                "synthesize_findings",
                "deduce_conclusion",
                "infer_meaning",
                "conclude_analysis",
                "calculate_score",
                "optimize_parameters",
                "validate_results",
                "verify_accuracy",
                "justify_decision",
                "explain_reasoning",
            ];

            reasoningSteps.forEach(stepType => {
                expect(strategy.canHandle(stepType)).to.equal(true);
            });
        });

        it("should return false for non-reasoning step types", () => {
            const nonReasoningSteps = [
                "format_output",
                "save_file",
                "send_email",
                "upload_data",
                "download_file",
            ];

            nonReasoningSteps.forEach(stepType => {
                expect(strategy.canHandle(stepType)).to.equal(false);
            });
        });

        it("should be case insensitive", () => {
            expect(strategy.canHandle("ANALYZE_DATA")).to.equal(true);
            expect(strategy.canHandle("Reason_About_Options")).to.equal(true);
            expect(strategy.canHandle("evaluate_RESULTS")).to.equal(true);
        });
    });

    describe("estimateResources", () => {
        it("should delegate to cost estimator", () => {
            const mockResourceUsage = {
                tokens: 750,
                apiCalls: 4,
                computeTime: 20000,
                cost: 0.005,
            };

            // Mock the cost estimator
            const mockEstimateResources = vi.fn().returns(mockResourceUsage);
            (strategy as any).costEstimator = { estimateResources: mockEstimateResources };

            const result = strategy.estimateResources(mockContext);

            expect(mockEstimateResources).toHaveBeenCalledWith(mockContext);
            expect(result).toEqual(mockResourceUsage);
        });

        it("should return realistic resource estimates", () => {
            const result = strategy.estimateResources(mockContext);

            expect(result).toHaveProperty("tokens");
            expect(result).toHaveProperty("apiCalls");
            expect(result).toHaveProperty("computeTime");
            expect(result).toHaveProperty("cost");

            expect(typeof result.tokens).toBe("number");
            expect(typeof result.apiCalls).toBe("number");
            expect(typeof result.computeTime).toBe("number");
            expect(typeof result.cost).toBe("number");

            expect(result.tokens).toBeGreaterThan(0);
            expect(result.apiCalls).toBeGreaterThan(0);
            expect(result.computeTime).toBeGreaterThan(0);
            expect(result.cost).toBeGreaterThan(0);
        });
    });

    describe("execute", () => {
        it("should execute the complete 4-phase reasoning process", async () => {
            // Mock all the modular components
            const mockInputHandler = {
                handleMissingInputs: vi.fn().resolves(mockContext),
            };

            const mockPhaseExecutor = {
                analyzeTask: vi.fn().resolves({
                    insights: ["Task requires data analysis", "Threshold comparison needed"],
                    creditsUsed: 100,
                    toolCallsCount: 1,
                }),
                planReasoningApproach: vi.fn().resolves({
                    executionPlan: "Analyze data, compare with threshold, conclude",
                    creditsUsed: 150,
                    toolCallsCount: 1,
                }),
                executeReasoningLogic: vi.fn().resolves({
                    outputs: { result: true, analysis: "Data exceeds threshold" },
                    creditsUsed: 300,
                    toolCallsCount: 2,
                }),
                refineAndValidate: vi.fn().resolves({
                    outputs: { result: true, analysis: "Data exceeds threshold" },
                    refinements: ["Validated logic", "Enhanced clarity"],
                    creditsUsed: 100,
                    toolCallsCount: 0,
                }),
            };

            const mockOutputGenerator = {
                createLegacyCompatibleSuccessResult: vi.fn().returns({
                    success: true,
                    outputs: { result: true, analysis: "Data exceeds threshold" },
                    metadata: {
                        strategy: "reasoning",
                        version: "2.0.0-enhanced",
                    },
                    resourceUsage: {
                        tokens: 650,
                        apiCalls: 4,
                        computeTime: 15000,
                        cost: 0.0065,
                    },
                    confidence: 0.90,
                    performanceScore: 0.85,
                }),
            };

            // Replace the mocked components
            (strategy as any).inputHandler = mockInputHandler;
            (strategy as any).phaseExecutor = mockPhaseExecutor;
            (strategy as any).outputGenerator = mockOutputGenerator;

            const result = await strategy.execute(mockContext);

            // Verify the execution flow
            expect(mockInputHandler.handleMissingInputs).toHaveBeenCalledWith(mockContext);
            expect(mockPhaseExecutor.analyzeTask).toHaveBeenCalled();
            expect(mockPhaseExecutor.planReasoningApproach).toHaveBeenCalled();
            expect(mockPhaseExecutor.executeReasoningLogic).toHaveBeenCalled();
            expect(mockPhaseExecutor.refineAndValidate).toHaveBeenCalled();
            expect(mockOutputGenerator.createLegacyCompatibleSuccessResult).toHaveBeenCalled();

            // Verify the result structure
            expect(result).toHaveProperty("success", true);
            expect(result).toHaveProperty("outputs");
            expect(result).toHaveProperty("metadata");
            expect(result).toHaveProperty("resourceUsage");
            expect(result).toHaveProperty("confidence");
            expect(result).toHaveProperty("performanceScore");
        });

        it("should handle execution errors gracefully", async () => {
            const error = new Error("LLM service unavailable");

            // Mock input handler to throw error
            const mockInputHandler = {
                handleMissingInputs: vi.fn().rejects(error),
            };

            const mockOutputGenerator = {
                createLegacyCompatibleErrorResult: vi.fn().returns({
                    success: false,
                    error: {
                        message: "LLM service unavailable",
                        type: "Error",
                        strategy: "reasoning",
                    },
                    outputs: {},
                    metadata: {
                        strategy: "reasoning",
                        version: "2.0.0-enhanced",
                        executionFailed: true,
                    },
                    confidence: 0.0,
                    performanceScore: 0.0,
                }),
            };

            (strategy as any).inputHandler = mockInputHandler;
            (strategy as any).outputGenerator = mockOutputGenerator;

            const result = await strategy.execute(mockContext);

            expect(mockOutputGenerator.createLegacyCompatibleErrorResult).toHaveBeenCalledWith(
                error,
                0, // creditsUsed
                0, // toolCallsCount
                expect.any(Number) // executionTime
            );

            expect(result).toHaveProperty("success", false);
            expect(result).toHaveProperty("error");
            expect(result.error).toHaveProperty("message", "LLM service unavailable");
        });

        it("should track resource usage correctly", async () => {
            // Mock components with realistic resource usage
            const mockInputHandler = {
                handleMissingInputs: vi.fn().resolves(mockContext),
            };

            const mockPhaseExecutor = {
                analyzeTask: vi.fn().resolves({
                    insights: ["Insight 1"],
                    creditsUsed: 100,
                    toolCallsCount: 1,
                }),
                planReasoningApproach: vi.fn().resolves({
                    executionPlan: "Plan",
                    creditsUsed: 150,
                    toolCallsCount: 1,
                }),
                executeReasoningLogic: vi.fn().resolves({
                    outputs: { result: true },
                    creditsUsed: 300,
                    toolCallsCount: 2,
                }),
                refineAndValidate: vi.fn().resolves({
                    outputs: { result: true },
                    refinements: ["Refined"],
                    creditsUsed: 100,
                    toolCallsCount: 0,
                }),
            };

            const mockOutputGenerator = {
                createLegacyCompatibleSuccessResult: vi.fn().mockImplementation((
                    outputs, creditsUsed, toolCallsCount, executionTime
                ) => ({
                    success: true,
                    outputs,
                    resourceUsage: {
                        tokens: creditsUsed * 7.5,
                        apiCalls: toolCallsCount,
                        computeTime: executionTime,
                        cost: creditsUsed * 0.00001,
                    },
                })),
            };

            (strategy as any).inputHandler = mockInputHandler;
            (strategy as any).phaseExecutor = mockPhaseExecutor;
            (strategy as any).outputGenerator = mockOutputGenerator;

            await strategy.execute(mockContext);

            // Verify that createLegacyCompatibleSuccessResult was called with accumulated resources
            expect(mockOutputGenerator.createLegacyCompatibleSuccessResult).toHaveBeenCalledWith(
                expect.any(Object), // outputs
                650, // total credits used (100 + 150 + 300 + 100)
                4, // total tool calls (1 + 1 + 2 + 0)
                expect.any(Number), // execution time
                expect.any(Object), // analysis result
                expect.any(Object), // planning result
                expect.any(Object) // refinement result
            );
        });
    });

    // Learning now happens through event-driven architecture
    // Tests removed: "learn" describe block

    describe("getPerformanceMetrics", () => {
        it("should return performance metrics structure", () => {
            const metrics = strategy.getPerformanceMetrics();

            expect(metrics).toHaveProperty("totalExecutions", 0);
            expect(metrics).toHaveProperty("successCount", 0);
            expect(metrics).toHaveProperty("failureCount", 0);
            expect(metrics).toHaveProperty("averageExecutionTime", 0);
            expect(metrics).toHaveProperty("averageResourceUsage", {});
            expect(metrics).toHaveProperty("averageConfidence", 0);
            expect(metrics).toHaveProperty("evolutionScore", 0.5);
        });

        it("should return consistent metric structure", () => {
            const metrics1 = strategy.getPerformanceMetrics();
            const metrics2 = strategy.getPerformanceMetrics();

            expect(metrics1).toEqual(metrics2);
        });
    });

    describe("reasoning framework selection", () => {
        it("should select decision tree framework for decision tasks", () => {
            const decisionContext = {
                ...mockContext,
                stepType: "decide_best_option",
            };

            // Access private method for testing
            const framework = (strategy as any).selectReasoningFramework(decisionContext);

            expect(framework.type).toBe("decision_tree");
            expect(framework.steps).toHaveLength(4); // options, criteria, evaluation, selection
        });

        it("should select analytical framework for analysis tasks", () => {
            const analysisContext = {
                ...mockContext,
                stepType: "analyze_performance",
            };

            const framework = (strategy as any).selectReasoningFramework(analysisContext);

            expect(framework.type).toBe("analytical");
            expect(framework.steps).toHaveLength(3); // data_gathering, pattern_analysis, synthesis
        });

        it("should select evidence-based framework for validation tasks", () => {
            const validationContext = {
                ...mockContext,
                stepType: "validate_hypothesis",
            };

            const framework = (strategy as any).selectReasoningFramework(validationContext);

            expect(framework.type).toBe("evidence_based");
            expect(framework.steps).toHaveLength(4); // hypothesis, evidence_gathering, evaluation, verdict
        });

        it("should select logical framework as default", () => {
            const defaultContext = {
                ...mockContext,
                stepType: "process_data",
            };

            const framework = (strategy as any).selectReasoningFramework(defaultContext);

            expect(framework.type).toBe("logical");
            expect(framework.steps).toHaveLength(3); // premises, rules, conclusion
        });
    });

    describe("integration with modular components", () => {
        it("should properly coordinate between all components", async () => {
            // This test verifies that the main strategy properly orchestrates all components
            const inputResult = { ...mockContext, inputs: { enhanced: true } };
            const analysisResult = { insights: ["test"], creditsUsed: 100, toolCallsCount: 1 };
            const planningResult = { executionPlan: "test plan", creditsUsed: 150, toolCallsCount: 1 };
            const executionResult = { outputs: { result: true }, creditsUsed: 300, toolCallsCount: 2 };
            const refinementResult = { outputs: { refined: true }, refinements: ["test"], creditsUsed: 100, toolCallsCount: 0 };
            const finalResult = { success: true, outputs: { refined: true } };

            // Mock all components
            (strategy as any).inputHandler = {
                handleMissingInputs: vi.fn().resolves(inputResult),
            };

            (strategy as any).phaseExecutor = {
                analyzeTask: vi.fn().resolves(analysisResult),
                planReasoningApproach: vi.fn().resolves(planningResult),
                executeReasoningLogic: vi.fn().resolves(executionResult),
                refineAndValidate: vi.fn().resolves(refinementResult),
            };

            (strategy as any).outputGenerator = {
                createLegacyCompatibleSuccessResult: vi.fn().returns(finalResult),
            };

            const result = await strategy.execute(mockContext);

            // Verify proper data flow between components
            expect((strategy as any).phaseExecutor.analyzeTask).toHaveBeenCalledWith(inputResult);
            expect((strategy as any).phaseExecutor.planReasoningApproach).toHaveBeenCalledWith(
                inputResult,
                analysisResult.insights
            );
            expect((strategy as any).phaseExecutor.executeReasoningLogic).toHaveBeenCalledWith(
                inputResult,
                planningResult.executionPlan
            );
            expect((strategy as any).phaseExecutor.refineAndValidate).toHaveBeenCalledWith(
                inputResult,
                executionResult,
                planningResult.executionPlan
            );

            expect(result).toBe(finalResult);
        });
    });
});