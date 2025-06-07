import { describe, it, expect, beforeEach, afterEach } from "@jest/globals";
import winston from "winston";
import { ConversationalStrategyAdapter } from "../conversationalAdapter.js";
import { StrategyType } from "@vrooli/shared";

describe("ConversationalStrategyAdapter", () => {
    let adapter: ConversationalStrategyAdapter;
    let logger: winston.Logger;
    // Sandbox type removed for Vitest compatibility

    beforeEach(() => {
        
        logger = winston.createLogger({
            level: "error",
            transports: [new winston.transports.Console()],
        });
        
        adapter = new ConversationalStrategyAdapter(logger);
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("Interface Compliance", () => {
        it("should have correct type and name", () => {
            expect(adapter.type).toBe(StrategyType.CONVERSATIONAL);
            expect(adapter.name).toBe("ConversationalStrategy");
            expect(adapter.version).toBe("1.0.0-adapter");
        });

        it("should implement all required methods", () => {
            expect(typeof adapter.canHandle).toBe("function");
            expect(typeof adapter.execute).toBe("function");
            expect(typeof adapter.estimateResources).toBe("function");
            expect(typeof adapter.learn).toBe("function");
            expect(typeof adapter.getPerformanceMetrics).toBe("function");
        });
    });

    describe("canHandle", () => {
        it("should handle conversational step types", () => {
            expect(adapter.canHandle("chat_response")).toBe(true);
            expect(adapter.canHandle("discuss_topic")).toBe(true);
            expect(adapter.canHandle("creative_writing")).toBe(true);
        });

        it("should handle explicit strategy request", () => {
            expect(adapter.canHandle("any_type", { executionStrategy: "conversational" })).toBe(true);
        });

        it("should reject non-conversational types", () => {
            expect(adapter.canHandle("calculate_sum")).toBe(false);
            expect(adapter.canHandle("api_call")).toBe(false);
        });
    });

    describe("estimateResources", () => {
        it("should provide reasonable resource estimates", () => {
            const context = {
                stepId: "step-123",
                stepType: "chat_response",
                inputs: { message: "Hello", context: "test" },
                config: {},
                resources: { credits: 1000 },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1 },
                constraints: { maxTokens: 2000 },
                metadata: { userId: "user-123" },
            };

            const estimate = adapter.estimateResources(context);

            expect(estimate.tokens).toBeGreaterThan(1000);
            expect(estimate.apiCalls).toBe(1);
            expect(estimate.computeTime).toBe(15000);
            expect(estimate.cost).toBeGreaterThan(0);
        });
    });

    describe("execute", () => {
        it("should execute with mock reasoning engine", async () => {
            const context = {
                stepId: "step-123",
                stepType: "chat_response",
                inputs: { message: "Hello world" },
                config: { name: "Test Chat", description: "Testing adapter" },
                resources: { credits: 1000, tools: [] },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1 },
                constraints: { maxTime: 30000 },
                metadata: { userId: "user-123" },
            };

            const result = await adapter.execute(context);

            expect(result.success).toBe(true);
            expect(result.metadata.strategyType).toBe(StrategyType.CONVERSATIONAL);
            expect(result.metadata.executionTime).toBeGreaterThan(0);
            expect(result.feedback.outcome).toBe("success");
        });

        it("should handle execution errors gracefully", async () => {
            // Stub the legacy strategy to throw an error
            const errorStub = sandbox.stub(adapter as any, "adaptContext").throws(new Error("Test error"));

            const context = {
                stepId: "step-123",
                stepType: "chat_response",
                inputs: { message: "Hello" },
                config: {},
                resources: { credits: 1000 },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1 },
                constraints: {},
                metadata: { userId: "user-123" },
            };

            const result = await adapter.execute(context);

            expect(result.success).toBe(false);
            expect(result.error).toBe("Test error");
            expect(result.feedback.outcome).toBe("failure");
            
            errorStub.restore();
        });
    });

    describe("Context Adaptation", () => {
        it("should adapt inputs correctly", () => {
            const context = {
                stepId: "step-123",
                stepType: "chat_response",
                inputs: { message: "Hello", data: { value: 42 } },
                config: {},
                resources: { credits: 1000 },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1 },
                constraints: {},
                metadata: { userId: "user-123" },
            };

            const legacyContext = (adapter as any).adaptContext(context);

            expect(legacyContext.context.ioMapping.inputs.message.value).toBe("Hello");
            expect(legacyContext.context.ioMapping.inputs.data.value).toEqual({ value: 42 });
        });

        it("should set reasonable limits", () => {
            const context = {
                stepId: "step-123",
                stepType: "chat_response",
                inputs: {},
                config: {},
                resources: { credits: 5000 },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1 },
                constraints: { maxTime: 60000 },
                metadata: { userId: "user-123" },
            };

            const legacyContext = (adapter as any).adaptContext(context);

            expect(legacyContext.context.limits.maxCredits).toBe(BigInt(5000));
            expect(legacyContext.context.limits.maxTimeMs).toBe(60000);
            expect(legacyContext.context.limits.maxToolCalls).toBe(10);
        });
    });

    describe("Result Adaptation", () => {
        it("should adapt successful legacy results", () => {
            const legacyResult = {
                success: true,
                ioMapping: {
                    outputs: {
                        response: { value: "Hello back!" },
                        confidence: { value: 0.9 },
                    },
                },
                creditsUsed: BigInt(150),
                timeElapsed: 5000,
                toolCallsCount: 2,
                messages: [
                    { text: "Hello", role: "user" },
                    { text: "Hello back!", role: "assistant" },
                ],
                metadata: {
                    metrics: {
                        conversationTurns: 2,
                        averageResponseTime: 8000,
                    },
                },
            };

            const result = (adapter as any).adaptResult(legacyResult, 5000);

            expect(result.success).toBe(true);
            expect(result.result.response).toBe("Hello back!");
            expect(result.result.confidence).toBe(0.9);
            expect(result.result.conversationHistory).toEqual(legacyResult.messages);
            expect(result.metadata.confidence).toBe(0.8);
            expect(result.feedback.outcome).toBe("success");
        });

        it("should adapt failed legacy results", () => {
            const legacyResult = {
                success: false,
                error: { message: "Conversation failed" },
                creditsUsed: BigInt(50),
                timeElapsed: 2000,
                toolCallsCount: 0,
            };

            const result = (adapter as any).adaptResult(legacyResult, 2000);

            expect(result.success).toBe(false);
            expect(result.error).toBe("Conversation failed");
            expect(result.metadata.confidence).toBe(0);
            expect(result.feedback.outcome).toBe("failure");
            expect(result.feedback.issues).toContain("Conversation failed");
        });
    });

    describe("Performance Metrics", () => {
        it("should return placeholder metrics", () => {
            const metrics = adapter.getPerformanceMetrics();

            expect(metrics.totalExecutions).toBe(0);
            expect(metrics.successCount).toBe(0);
            expect(metrics.failureCount).toBe(0);
            expect(metrics.averageConfidence).toBe(0.7);
            expect(metrics.evolutionScore).toBe(0);
        });
    });

    describe("Learning", () => {
        it("should log feedback without errors", () => {
            const logSpy = vi.spyOn(logger, "info");
            
            const feedback = {
                outcome: "success" as const,
                performanceScore: 0.8,
                userSatisfaction: 0.9,
            };

            adapter.learn(feedback);

            expect(logSpy.calledWith("[ConversationalStrategyAdapter] Learning from feedback")).toBe(true);
        });
    });
});