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
    let llmServiceStub: any;

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

        // Create mock LLM service
        llmServiceStub = {
            executeRequest: vi.fn(),
        };
        
        // Mock the LLMIntegrationService constructor
        vi.spyOn(LLMIntegrationService.prototype, "constructor" as any).mockImplementation(() => llmServiceStub);
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
            expect(strategy.canHandle("chat_response", {})).toEqual(true);
            expect(strategy.canHandle("creative_writing", {})).toEqual(true);
            expect(strategy.canHandle("brainstorm_ideas", {})).toEqual(true);
            expect(strategy.canHandle("calculate_sum", {})).toEqual(false);
        });

        it("should handle explicit strategy request", () => {
            expect(strategy.canHandle("any_type", { strategy: "conversational" })).toEqual(true);
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
            expect(result.result.result).toContain("encountered an issue");
            expect(result.metadata.confidence).toBeLessThan(0.5);
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
            expect(strategy.canHandle("analyze_problem", {})).toEqual(true);
            expect(strategy.canHandle("plan_solution", {})).toEqual(true);
            expect(strategy.canHandle("evaluate_options", {})).toEqual(true);
            expect(strategy.canHandle("chat_response", {})).toEqual(false);
        });

        it("should execute reasoning tasks with four phases", async () => {
            llmServiceStub.executeRequest.mockResolvedValueOnce({
                content: "The problem involves processing data with value 42",
                reasoning: "Understanding phase complete",
                confidence: 0.9,
                tokensUsed: 100,
            }).mockResolvedValueOnce({
                content: "We can use arithmetic operations to process this value",
                reasoning: "Planning phase complete",
                confidence: 0.85,
                tokensUsed: 120,
            }).mockResolvedValueOnce({
                content: "Result: 84 (doubled the value)",
                reasoning: "Execution phase complete",
                confidence: 0.95,
                tokensUsed: 80,
            }).mockResolvedValueOnce({
                content: "The process successfully doubled the input value",
                reasoning: "Validation phase complete",
                confidence: 0.9,
                tokensUsed: 90,
            });

            const result = await strategy.execute(mockContext);

            expect(result.success).toBe(true);
            expect(result.metadata.strategyType).toBe(StrategyType.REASONING);
            expect(result.metadata.phases).toBe(4);
            expect(llmServiceStub.executeRequest).toHaveBeenCalledTimes(4);
        });

        it("should handle partial phase failures", async () => {
            llmServiceStub.executeRequest
                .mockResolvedValueOnce({
                    content: "Understanding the problem",
                    confidence: 0.8,
                    tokensUsed: 100,
                })
                .mockRejectedValueOnce(new Error("Planning phase failed"))
                .mockResolvedValueOnce({
                    content: "Attempting direct execution",
                    confidence: 0.6,
                    tokensUsed: 150,
                })
                .mockResolvedValueOnce({
                    content: "Partial success achieved",
                    confidence: 0.65,
                    tokensUsed: 100,
                });

            const result = await strategy.execute(mockContext);

            expect(result.success).toBe(true);
            expect(result.metadata.confidence).toBeLessThan(0.7);
            expect(result.feedback.issues).toContain("Planning phase encountered issues");
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
            expect(strategy.canHandle("calculate_sum", {})).toEqual(true);
            expect(strategy.canHandle("api_call", {})).toEqual(true);
            expect(strategy.canHandle("database_query", {})).toEqual(true);
            expect(strategy.canHandle("chat_response", {})).toEqual(false);
        });

        it("should execute deterministic tasks without LLM", async () => {
            const deterministicContext = {
                ...mockContext,
                stepType: "calculate_sum",
                inputs: {
                    numbers: [1, 2, 3, 4, 5],
                },
            };

            const result = await strategy.execute(deterministicContext);

            expect(result.success).toBe(true);
            expect(result.result).toHaveProperty("sum", 15);
            expect(result.metadata.strategyType).toBe(StrategyType.DETERMINISTIC);
            expect(result.metadata.tokensUsed).toBe(0);
            expect(llmServiceStub.executeRequest).not.toHaveBeenCalled();
        });

        it("should handle API call simulation", async () => {
            const apiContext = {
                ...mockContext,
                stepType: "api_call",
                inputs: {
                    endpoint: "/users/123",
                    method: "GET",
                },
            };

            const result = await strategy.execute(apiContext);

            expect(result.success).toBe(true);
            expect(result.result).toHaveProperty("status", 200);
            expect(result.result).toHaveProperty("data");
            expect(result.metadata.apiCallsUsed).toBe(1);
        });

        it("should estimate minimal resources", () => {
            const estimate = strategy.estimateResources(mockContext);

            expect(estimate.tokens).toBe(0);
            expect(estimate.apiCalls).toBe(0);
            expect(estimate.computeTime).toBe(1000);
            expect(estimate.cost).toBe(0);
        });
    });

    describe("Strategy Selection", () => {
        it("should correctly identify strategy requirements", () => {
            const conversationalCtx = { ...mockContext, stepType: "chat_response" };
            const reasoningCtx = { ...mockContext, stepType: "analyze_problem" };
            const deterministicCtx = { ...mockContext, stepType: "calculate_sum" };

            const conversational = new ConversationalStrategy(logger);
            const reasoning = new ReasoningStrategy(logger);
            const deterministic = new DeterministicStrategy(logger);

            expect(conversational.canHandle(conversationalCtx.stepType, conversationalCtx.config)).toBe(true);
            expect(reasoning.canHandle(reasoningCtx.stepType, reasoningCtx.config)).toBe(true);
            expect(deterministic.canHandle(deterministicCtx.stepType, deterministicCtx.config)).toBe(true);

            // Cross-check they don't handle each other's types
            expect(conversational.canHandle(deterministicCtx.stepType, deterministicCtx.config)).toBe(false);
            expect(reasoning.canHandle(conversationalCtx.stepType, conversationalCtx.config)).toBe(false);
            expect(deterministic.canHandle(reasoningCtx.stepType, reasoningCtx.config)).toBe(false);
        });
    });
});