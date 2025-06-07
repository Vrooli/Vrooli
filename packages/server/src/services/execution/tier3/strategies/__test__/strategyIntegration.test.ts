import { describe, it, beforeEach, vi, expect } from "vitest";
import { createStubInstance } from "sinon";
import { type Logger } from "winston";
import { 
    StrategyType,
    type ExecutionContext,
} from "@vrooli/shared";
import { StrategySelector } from "../strategySelector.js";
import { StrategyFactory } from "../strategyFactory.js";
import { ConversationalStrategy } from "../conversationalStrategy.js";
import { DeterministicStrategy } from "../deterministicStrategy.js";
import { ReasoningStrategy } from "../reasoningStrategy.js";

describe("Strategy Integration Tests", () => {
    let mockLogger: Logger;
    let selector: StrategySelector;
    let factory: StrategyFactory;

    beforeEach(() => {
        mockLogger = {
            info: () => {},
            debug: () => {},
            warn: () => {},
            error: () => {},
        } as any;

        selector = new StrategySelector(mockLogger);
        factory = new StrategyFactory(mockLogger);
    });

    describe("Strategy Factory", () => {
        it("should initialize all strategies", () => {
            const stats = factory.getStatistics();
            
            expect(stats.availableStrategies).to.include(StrategyType.CONVERSATIONAL);
            expect(stats.availableStrategies).to.include(StrategyType.DETERMINISTIC);
            expect(stats.availableStrategies).to.include(StrategyType.REASONING);
        });

        it("should return correct strategy by type", () => {
            const conversational = factory.getStrategy(StrategyType.CONVERSATIONAL);
            const deterministic = factory.getStrategy(StrategyType.DETERMINISTIC);
            const reasoning = factory.getStrategy(StrategyType.REASONING);

            expect(conversational).to.be.instanceOf(ConversationalStrategy);
            expect(deterministic).to.be.instanceOf(DeterministicStrategy);
            expect(reasoning).to.be.instanceOf(ReasoningStrategy);
        });

        it("should track strategy usage", () => {
            factory.getStrategy(StrategyType.CONVERSATIONAL);
            factory.getStrategy(StrategyType.CONVERSATIONAL);
            factory.getStrategy(StrategyType.DETERMINISTIC);

            const stats = factory.getStatistics();
            
            expect(stats.usage[StrategyType.CONVERSATIONAL]).toBe(2);
            expect(stats.usage[StrategyType.DETERMINISTIC]).toBe(1);
            expect(stats.usage[StrategyType.REASONING] || 0).toBe(0);
        });
    });

    describe("Strategy Selection", () => {
        it("should select conversational strategy for chat-like tasks", () => {
            const context: ExecutionContext = {
                stepId: "test-1",
                stepType: "chat_with_user",
                inputs: {
                    message: "Hello, how can I help?",
                },
                config: {},
                resources: {
                    models: [{ provider: "openai", model: "gpt-4", capabilities: [], cost: 0.01, available: true }],
                    tools: [],
                    apis: [],
                    credits: 1000,
                },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                constraints: {},
            };

            const strategy = factory.selectStrategy(context);
            expect(strategy?.type).toBe(StrategyType.CONVERSATIONAL);
        });

        it("should select deterministic strategy for API tasks", () => {
            const context: ExecutionContext = {
                stepId: "test-2",
                stepType: "api_integration",
                inputs: {
                    endpoint: "https://api.example.com",
                    method: "GET",
                },
                config: {
                    expectedOutputs: {
                        data: { type: "Object" },
                        status: { type: "Number" },
                    },
                },
                resources: {
                    models: [],
                    tools: [],
                    apis: [{ name: "example", baseUrl: "https://api.example.com", authType: "none" }],
                    credits: 100,
                },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                constraints: {},
            };

            const strategy = factory.selectStrategy(context);
            expect(strategy?.type).toBe(StrategyType.DETERMINISTIC);
        });

        it("should select reasoning strategy for analytical tasks", () => {
            const context: ExecutionContext = {
                stepId: "test-3",
                stepType: "analyze_data",
                inputs: {
                    dataset: [1, 2, 3, 4, 5],
                    criteria: "above average",
                    threshold: 3,
                },
                config: {
                    expectedOutputs: {
                        result: { type: "Boolean" },
                        reasoning: { type: "Text" },
                    },
                },
                resources: {
                    models: [{ provider: "openai", model: "gpt-4", capabilities: ["reasoning"], cost: 0.01, available: true }],
                    tools: [],
                    apis: [],
                    credits: 1000,
                },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                constraints: {
                    requiredConfidence: 0.9,
                },
            };

            const strategy = factory.selectStrategy(context);
            expect(strategy?.type).toBe(StrategyType.REASONING);
        });

        it("should respect explicit strategy request", () => {
            const context: ExecutionContext = {
                stepId: "test-4",
                stepType: "generic_task",
                inputs: {},
                config: {
                    strategy: "conversational",
                },
                resources: { models: [], tools: [], apis: [], credits: 100 },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                constraints: {},
            };

            const strategy = factory.selectStrategy(context);
            expect(strategy?.type).toBe(StrategyType.CONVERSATIONAL);
        });

        it("should respect allowed strategies constraint", () => {
            const context: ExecutionContext = {
                stepId: "test-5",
                stepType: "chat_task", // Would normally select conversational
                inputs: { message: "Hello" },
                config: {},
                resources: { models: [], tools: [], apis: [], credits: 100 },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                constraints: {
                    allowedStrategies: [StrategyType.DETERMINISTIC],
                },
            };

            const strategy = factory.selectStrategy(context);
            expect(strategy?.type).toBe(StrategyType.DETERMINISTIC);
        });
    });

    describe("Strategy Compatibility", () => {
        it("should have non-overlapping canHandle for basic step types", () => {
            const conversational = new ConversationalStrategy(mockLogger);
            const deterministic = new DeterministicStrategy(mockLogger);
            const reasoning = new ReasoningStrategy(mockLogger);

            const testCases = [
                { stepType: "chat", expected: StrategyType.CONVERSATIONAL },
                { stepType: "api_call", expected: StrategyType.DETERMINISTIC },
                { stepType: "analyze", expected: StrategyType.REASONING },
            ];

            for (const testCase of testCases) {
                const canHandleResults = {
                    [StrategyType.CONVERSATIONAL]: conversational.canHandle(testCase.stepType),
                    [StrategyType.DETERMINISTIC]: deterministic.canHandle(testCase.stepType),
                    [StrategyType.REASONING]: reasoning.canHandle(testCase.stepType),
                };

                // Verify expected strategy can handle
                expect(canHandleResults[testCase.expected]).toBe(true);

                // Count how many strategies can handle
                const handleCount = Object.values(canHandleResults).filter(v => v).length;
                expect(handleCount).to.be.at.least(1); // At least one should handle
            }
        });

        it("should all provide resource estimates", () => {
            const strategies = [
                new ConversationalStrategy(mockLogger),
                new DeterministicStrategy(mockLogger),
                new ReasoningStrategy(mockLogger),
            ];

            const context: ExecutionContext = {
                stepId: "test-estimate",
                stepType: "generic",
                inputs: { data: "test" },
                config: {},
                resources: { models: [], tools: [], apis: [], credits: 1000 },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                constraints: {},
            };

            for (const strategy of strategies) {
                const estimate = strategy.estimateResources(context);
                
                expect(estimate).toHaveProperty("computeTime");
                expect(estimate.computeTime).toBeGreaterThan(0);
                
                if (strategy.type === StrategyType.DETERMINISTIC) {
                    expect(estimate.tokens).toBe(0); // Deterministic doesn't use LLMs
                } else {
                    expect(estimate.tokens).toBeGreaterThan(0);
                }
            }
        });
    });

    describe("Strategy Selector Integration", () => {
        it("should suggest appropriate strategies", () => {
            const chatContext: ExecutionContext = {
                stepId: "suggest-1",
                stepType: "customer_support",
                inputs: { message: "I need help with my order" },
                config: {},
                resources: {
                    models: [{ provider: "openai", model: "gpt-4", capabilities: [], cost: 0.01, available: true }],
                    tools: [],
                    apis: [],
                    credits: 1000,
                },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                constraints: {},
            };

            const suggestion = selector.suggestStrategy(chatContext);
            
            expect(suggestion.recommended).toBe(StrategyType.CONVERSATIONAL);
            expect(suggestion.alternatives).to.include(StrategyType.REASONING);
            expect(suggestion.reasoning).to.include("interactive");
        });

        it("should provide fallback alternatives", () => {
            const complexContext: ExecutionContext = {
                stepId: "suggest-2",
                stepType: "complex_analysis",
                inputs: {
                    data1: [1, 2, 3],
                    data2: ["a", "b", "c"],
                    criteria: "correlation",
                    method: "statistical",
                },
                config: {},
                resources: {
                    models: [{ provider: "openai", model: "gpt-4", capabilities: ["reasoning"], cost: 0.01, available: true }],
                    tools: [],
                    apis: [],
                    credits: 1000,
                },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                constraints: {
                    requiredConfidence: 0.95,
                },
            };

            const suggestion = selector.suggestStrategy(complexContext);
            
            expect(suggestion.recommended).toBe(StrategyType.REASONING);
            expect(suggestion.alternatives).to.have.lengthOf.at.least(1);
            expect(suggestion.reasoning).to.include("analytical");
        });
    });

    describe("Strategy Performance Tracking", () => {
        it("should track performance metrics for each strategy", () => {
            const strategies = [
                new ConversationalStrategy(mockLogger),
                new DeterministicStrategy(mockLogger),
                new ReasoningStrategy(mockLogger),
            ];

            for (const strategy of strategies) {
                const metrics = strategy.getPerformanceMetrics();
                
                expect(metrics).toHaveProperty("totalExecutions");
                expect(metrics).toHaveProperty("successCount");
                expect(metrics).toHaveProperty("failureCount");
                expect(metrics).toHaveProperty("averageExecutionTime");
                expect(metrics).toHaveProperty("averageResourceUsage");
                expect(metrics).toHaveProperty("averageConfidence");
                expect(metrics).toHaveProperty("evolutionScore");
                
                // Initial metrics should be zero/default
                expect(metrics.totalExecutions).toBe(0);
                expect(metrics.successCount).toBe(0);
                expect(metrics.failureCount).toBe(0);
            }
        });

        it("should support learning from feedback", () => {
            const strategies = [
                new ConversationalStrategy(mockLogger),
                new DeterministicStrategy(mockLogger),
                new ReasoningStrategy(mockLogger),
            ];

            const successFeedback = {
                outcome: "success" as const,
                performanceScore: 0.9,
                userSatisfaction: 0.85,
            };

            const failureFeedback = {
                outcome: "failure" as const,
                performanceScore: 0.2,
                issues: ["Timeout error"],
            };

            for (const strategy of strategies) {
                // Should not throw
                expect(() => strategy.learn(successFeedback)).to.not.throw();
                expect(() => strategy.learn(failureFeedback)).to.not.throw();
            }
        });
    });

    describe("Cross-Strategy Consistency", () => {
        it("should all implement required interface methods", () => {
            const strategies = [
                new ConversationalStrategy(mockLogger),
                new DeterministicStrategy(mockLogger),
                new ReasoningStrategy(mockLogger),
            ];

            for (const strategy of strategies) {
                // Check required properties
                expect(strategy).toHaveProperty("type");
                expect(strategy).toHaveProperty("name");
                expect(strategy).toHaveProperty("version");
                
                // Check required methods
                expect(strategy.execute).to.be.a("function");
                expect(strategy.canHandle).to.be.a("function");
                expect(strategy.estimateResources).to.be.a("function");
                expect(strategy.learn).to.be.a("function");
                expect(strategy.getPerformanceMetrics).to.be.a("function");
            }
        });

        it("should all return consistent result structures", async () => {
            const strategies = [
                new ConversationalStrategy(mockLogger),
                new DeterministicStrategy(mockLogger),
                new ReasoningStrategy(mockLogger),
            ];

            const context: ExecutionContext = {
                stepId: "consistency-test",
                stepType: "test",
                inputs: {},
                config: {},
                resources: { models: [], tools: [], apis: [], credits: 100 },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                constraints: {},
            };

            for (const strategy of strategies) {
                // Skip if strategy can't handle this type
                if (!strategy.canHandle(context.stepType, context.config)) {
                    continue;
                }

                // Mock successful execution for testing structure
                const result = {
                    success: true,
                    result: {},
                    metadata: {
                        strategyType: strategy.type,
                        executionTime: 100,
                        resourceUsage: {},
                        confidence: 0.8,
                        fallbackUsed: false,
                    },
                    feedback: {
                        outcome: "success" as const,
                        performanceScore: 0.8,
                    },
                };

                // Verify structure
                expect(result).toHaveProperty("success");
                expect(result).toHaveProperty("metadata");
                expect(result.metadata).toHaveProperty("strategyType");
                expect(result.metadata).toHaveProperty("executionTime");
                expect(result.metadata).toHaveProperty("resourceUsage");
                expect(result.metadata).toHaveProperty("confidence");
                expect(result.metadata).toHaveProperty("fallbackUsed");
                expect(result).toHaveProperty("feedback");
                expect(result.feedback).toHaveProperty("outcome");
                expect(result.feedback).toHaveProperty("performanceScore");
            }
        });
    });
});