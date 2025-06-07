import { describe, it, beforeEach, afterEach, vi, expect } from "vitest";
import winston from "winston";
import { ConversationalStrategy } from "../../tier3/strategies/conversationalStrategy.js";
import { ReasoningStrategy } from "../../tier3/strategies/reasoningStrategy.js";
import { DeterministicStrategy } from "../../tier3/strategies/deterministicStrategy.js";
import { LLMIntegrationService } from "../../integration/llmIntegrationService.js";
import {
    type ExecutionContext,
    StrategyType,
} from "@vrooli/shared";

describe("Execution Strategies", () => {
    let logger: winston.Logger;
    // Sandbox type removed for Vitest compatibility
    let llmServiceStub: LLMIntegrationService;

    const mockContext: ExecutionContext = {
        stepId: "step-123",
        stepType: "process_data",
        runId: "run-123",
        swarmId: "swarm-123",
        inputs: {
            data: { value: 42 },
            instruction: "Process this data",
        },
        outputs: {},
        config: {
            model: "gpt-4o-mini",
            temperature: 0.7,
        },
        constraints: {
            maxTokens: 1000,
            maxTime: 30000,
            requiredConfidence: 0.8,
        },
        resources: {
            credits: 1000,
            tokens: 10000,
            tools: [
                {
                    name: "calculator",
                    description: "Performs calculations",
                    parameters: {},
                },
            ],
        },
        history: {
            recentSteps: [],
            totalSteps: 0,
            successes: 0,
            failures: 0,
        },
        userId: "user-123",
        sessionId: "session-123",
    };

    beforeEach(() => {
        
        logger = winston.createLogger({
            level: "error",
            transports: [new winston.transports.Console()],
        });

        // Stub LLM service
        llmServiceStub = vi.mocked(new LLMIntegrationService() as any);
        sandbox.stub(LLMIntegrationService.prototype, "constructor").returns(llmServiceStub as any);
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("ConversationalStrategy", () => {
        let strategy: ConversationalStrategy;

        beforeEach(() => {
            strategy = new ConversationalStrategy(logger);
        });

        it("should identify itself correctly", () => {
            expect(strategy.type).toBe(StrategyType.CONVERSATIONAL);
            expect(strategy.name).toBe("ConversationalStrategy");
        });

        it("should handle conversational step types", () => {
            expect(strategy.canHandle("chat_response", {})).to.equal(true);
            expect(strategy.canHandle("creative_writing", {})).to.equal(true);
            expect(strategy.canHandle("brainstorm_ideas", {})).to.equal(true);
            expect(strategy.canHandle("calculate_sum", {})).to.equal(false);
        });

        it("should handle explicit strategy request", () => {
            expect(strategy.canHandle("any_type", { strategy: "conversational" })).to.equal(true);
        });

        it("should execute conversational tasks successfully", async () => {
            llmServiceStub.executeRequest.mockResolvedValue({
                content: "Based on the data provided, the value 42 appears to be significant.",
                reasoning: "The user asked to process data with value 42",
                confidence: 0.85,
                tokensUsed: 150,
                model: "gpt-4o-mini",
                finishReason: "stop",
                resourceUsage: {
                    tokens: 150,
                    apiCalls: 1,
                    computeTime: 1200,
                    cost: 0.003,
                },
            });

            const result = await strategy.execute(mockContext);

            expect(result.success).toBe(true);
            expect(result.result).toHaveProperty("result");
            expect(result.metadata.strategyType).toBe(StrategyType.CONVERSATIONAL);
            expect(result.metadata.confidence).toBeGreaterThan(0);
            expect(llmServiceStub.executeRequest).toHaveBeenCalled();
        });

        it("should handle LLM failures gracefully", async () => {
            llmServiceStub.executeRequest.mockRejectedValue(new Error("LLM service unavailable"));

            const result = await strategy.execute(mockContext);

            expect(result.success).toBe(true); // Falls back to default response
            expect(result.result.result).to.include("encountered an issue");
            expect(result.metadata.confidence).to.be.lessThan(0.5);
        });

        it("should estimate resources appropriately", () => {
            const estimate = strategy.estimateResources(mockContext);

            expect(estimate.tokens).toBeGreaterThan(1000);
            expect(estimate.apiCalls).toBe(1);
            expect(estimate.computeTime).toBe(10000);
            expect(estimate.cost).toBe(0.02);
        });
    });

    describe("ReasoningStrategy", () => {
        let strategy: ReasoningStrategy;

        beforeEach(() => {
            strategy = new ReasoningStrategy(logger);
        });

        it("should identify itself correctly", () => {
            expect(strategy.type).toBe(StrategyType.REASONING);
            expect(strategy.name).toBe("ReasoningStrategy");
        });

        it("should handle reasoning step types", () => {
            expect(strategy.canHandle("analyze_data", {})).to.equal(true);
            expect(strategy.canHandle("evaluate_options", {})).to.equal(true);
            expect(strategy.canHandle("make_decision", {})).to.equal(true);
            expect(strategy.canHandle("validate_result", {})).to.equal(true);
            expect(strategy.canHandle("chat_response", {})).to.equal(false);
        });

        it("should execute analytical tasks with reasoning framework", async () => {
            // Mock LLM responses for reasoning steps
            llmServiceStub.executeRequest
                .onFirstCall().resolves({
                    content: "Data points identified: [42, 'significant', 'processing required']",
                    confidence: 0.9,
                    tokensUsed: 100,
                    model: "gpt-4o-mini",
                    finishReason: "stop",
                    resourceUsage: {
                        tokens: 100,
                        apiCalls: 1,
                        computeTime: 800,
                        cost: 0.002,
                    },
                })
                .onSecondCall().resolves({
                    content: "Pattern detected: Sequential processing with validation",
                    confidence: 0.85,
                    tokensUsed: 120,
                    model: "gpt-4o-mini",
                    finishReason: "stop",
                    resourceUsage: {
                        tokens: 120,
                        apiCalls: 1,
                        computeTime: 900,
                        cost: 0.0024,
                    },
                })
                .onThirdCall().resolves({
                    content: "Insights: Data requires transformation before validation",
                    confidence: 0.88,
                    tokensUsed: 130,
                    model: "gpt-4o-mini",
                    finishReason: "stop",
                    resourceUsage: {
                        tokens: 130,
                        apiCalls: 1,
                        computeTime: 1000,
                        cost: 0.0026,
                    },
                });

            const analyticalContext = {
                ...mockContext,
                stepType: "analyze_data",
            };

            const result = await strategy.execute(analyticalContext);

            expect(result.success).toBe(true);
            expect(result.result).to.exist;
            expect(result.metadata.strategyType).toBe(StrategyType.REASONING);
            expect(result.metadata.confidence).toBeGreaterThan(0.7);
            expect(llmServiceStub.executeRequest).toHaveBeenCalled();Thrice; // Multiple reasoning steps
        });

        it("should select appropriate reasoning framework", () => {
            const decisionContext = { ...mockContext, stepType: "decide_action" };
            const framework = (strategy as any).selectReasoningFramework(decisionContext);
            expect(framework.type).toBe("decision_tree");

            const analyzeContext = { ...mockContext, stepType: "analyze_results" };
            const analyticalFramework = (strategy as any).selectReasoningFramework(analyzeContext);
            expect(analyticalFramework.type).toBe("analytical");

            const validateContext = { ...mockContext, stepType: "validate_output" };
            const evidenceFramework = (strategy as any).selectReasoningFramework(validateContext);
            expect(evidenceFramework.type).toBe("evidence_based");
        });

        it("should handle reasoning step failures with fallback", async () => {
            llmServiceStub.executeRequest.mockRejectedValue(new Error("LLM error"););

            const result = await strategy.execute(mockContext);

            expect(result.success).toBe(true);
            expect(result.result).to.exist;
            expect(result.metadata.confidence).toBeGreaterThan(0); // Uses simulated reasoning
        });
    });

    describe("DeterministicStrategy", () => {
        let strategy: DeterministicStrategy;

        beforeEach(() => {
            strategy = new DeterministicStrategy(logger);
        });

        it("should identify itself correctly", () => {
            expect(strategy.type).toBe(StrategyType.DETERMINISTIC);
            expect(strategy.name).toBe("DeterministicStrategy");
        });

        it("should handle deterministic step types", () => {
            expect(strategy.canHandle("calculate", {})).to.equal(true);
            expect(strategy.canHandle("api_call", {})).to.equal(true);
            expect(strategy.canHandle("data_transform", {})).to.equal(true);
            expect(strategy.canHandle("validate_schema", {})).to.equal(true);
            expect(strategy.canHandle("creative_writing", {})).to.equal(false);
        });

        it("should execute deterministic tasks without LLM", async () => {
            const calcContext = {
                ...mockContext,
                stepType: "calculate_sum",
                inputs: {
                    operation: "sum",
                    values: [10, 20, 30],
                },
            };

            const result = await strategy.execute(calcContext);

            expect(result.success).toBe(true);
            expect(result.result).toEqual({ result: 60 });
            expect(result.metadata.strategyType).toBe(StrategyType.DETERMINISTIC);
            expect(result.metadata.confidence).toBe(1); // Deterministic = 100% confidence
            expect(llmServiceStub.executeRequest).to.not.have.been.called; // No LLM needed
        });

        it("should handle API call simulation", async () => {
            const apiContext = {
                ...mockContext,
                stepType: "api_call",
                inputs: {
                    endpoint: "/api/data",
                    method: "GET",
                },
            };

            const result = await strategy.execute(apiContext);

            expect(result.success).toBe(true);
            expect(result.result).toHaveProperty("status", 200);
            expect(result.result).toHaveProperty("data");
        });

        it("should validate data against schema", async () => {
            const validationContext = {
                ...mockContext,
                stepType: "validate_schema",
                inputs: {
                    data: { name: "test", value: 42 },
                    schema: {
                        type: "object",
                        properties: {
                            name: { type: "string" },
                            value: { type: "number" },
                        },
                        required: ["name", "value"],
                    },
                },
            };

            const result = await strategy.execute(validationContext);

            expect(result.success).toBe(true);
            expect(result.result).toEqual({ valid: true });
        });

        it("should cache repeated calculations", async () => {
            const calcContext = {
                ...mockContext,
                stepType: "calculate_sum",
                inputs: {
                    operation: "sum",
                    values: [1, 2, 3],
                },
            };

            // Execute twice
            const result1 = await strategy.execute(calcContext);
            const result2 = await strategy.execute(calcContext);

            expect(result1.result).toEqual(result2.result);
            expect(result2.metadata.resourceUsage?.computeTime).to.be.lessThan(
                result1.metadata.resourceUsage?.computeTime || 0
            ); // Second call should be faster due to cache
        });

        it("should estimate minimal resources", () => {
            const estimate = strategy.estimateResources(mockContext);

            expect(estimate.tokens).toBe(0); // No LLM tokens needed
            expect(estimate.apiCalls).toBe(0);
            expect(estimate.computeTime).to.be.lessThan(1000); // Fast execution
            expect(estimate.cost).toBe(0); // No LLM cost
        });

        it("should handle execution errors gracefully", async () => {
            const errorContext = {
                ...mockContext,
                stepType: "calculate_sum",
                inputs: {
                    operation: "divide",
                    values: [10, 0], // Division by zero
                },
            };

            const result = await strategy.execute(errorContext);

            expect(result.success).toBe(false);
            expect(result.error).to.include("Division by zero");
        });
    });

    describe("Strategy Evolution", () => {
        it("should track performance metrics", () => {
            const strategy = new ConversationalStrategy(logger);
            
            const metrics = strategy.getPerformanceMetrics();
            
            expect(metrics).toHaveProperty("totalExecutions");
            expect(metrics).toHaveProperty("successCount");
            expect(metrics).toHaveProperty("failureCount");
            expect(metrics).toHaveProperty("averageExecutionTime");
            expect(metrics).toHaveProperty("evolutionScore");
        });

        it("should accept learning feedback", () => {
            const strategy = new ReasoningStrategy(logger);
            
            const feedback = {
                outcome: "success" as const,
                performanceScore: 0.9,
                userSatisfaction: 0.85,
                improvements: ["Consider more evidence sources"],
            };

            // Should not throw
            expect(() => strategy.learn(feedback)).to.not.throw();
        });
    });
});