import { describe, it, beforeEach, vi, expect } from "vitest";
import { stub, createStubInstance } from "sinon";
import { type Logger } from "winston";
import { 
    StrategyType,
    type ExecutionContext,
} from "@vrooli/shared";
import { StrategySelector } from "../strategySelector.js";

describe("StrategySelector", () => {
    let mockLogger: Logger;
    let selector: StrategySelector;

    beforeEach(() => {
        mockLogger = {
            info: stub(),
            debug: stub(),
            warn: stub(),
            error: stub(),
        } as any;

        selector = new StrategySelector(mockLogger);
    });

    describe("execute", () => {
        it("should execute successfully with primary strategy", async () => {
            const context: ExecutionContext = {
                stepId: "test-success",
                stepType: "api_call",
                inputs: { endpoint: "/test" },
                config: {
                    expectedOutputs: { result: { type: "String" } },
                },
                resources: {
                    models: [],
                    tools: [],
                    apis: [],
                    credits: 100,
                },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                constraints: {},
            };

            const result = await selector.execute(context);

            // Should succeed with deterministic strategy
            expect(result.success).toBe(true);
            expect(result.metadata.strategyType).toBe(StrategyType.DETERMINISTIC);
            expect(result.metadata.fallbackUsed).toBe(false);
        });

        it("should handle no suitable strategy", async () => {
            const context: ExecutionContext = {
                stepId: "test-no-strategy",
                stepType: "unknown_type",
                inputs: {},
                config: {},
                resources: {
                    models: [],
                    tools: [],
                    apis: [],
                    credits: 0, // No credits
                },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                constraints: {
                    allowedStrategies: [], // No strategies allowed
                },
            };

            const result = await selector.execute(context);

            expect(result.success).toBe(false);
            expect(result.error).toBe("No suitable strategy found");
            expect(result.metadata.fallbackUsed).toBe(true);
        });

        it("should respect strategy constraints", async () => {
            const context: ExecutionContext = {
                stepId: "test-constraints",
                stepType: "chat", // Would normally use conversational
                inputs: { message: "Hello" },
                config: {},
                resources: {
                    models: [],
                    tools: [],
                    apis: [],
                    credits: 100,
                },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                constraints: {
                    allowedStrategies: [StrategyType.DETERMINISTIC], // Force deterministic
                },
            };

            const result = await selector.execute(context);

            // Should use deterministic despite chat type
            expect(result.metadata.strategyType).toBe(StrategyType.DETERMINISTIC);
        });
    });

    describe("suggestStrategy", () => {
        it("should suggest conversational for chat tasks", () => {
            const context: ExecutionContext = {
                stepId: "suggest-chat",
                stepType: "customer_chat",
                inputs: { message: "Help me with my order" },
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

            const suggestion = selector.suggestStrategy(context);

            expect(suggestion.recommended).toBe(StrategyType.CONVERSATIONAL);
            expect(suggestion.reasoning).to.include("interactive");
            expect(suggestion.alternatives).to.have.lengthOf.at.least(1);
        });

        it("should suggest deterministic for API tasks", () => {
            const context: ExecutionContext = {
                stepId: "suggest-api",
                stepType: "api_integration",
                inputs: { 
                    endpoint: "https://api.example.com/data",
                    method: "GET",
                },
                config: {
                    expectedOutputs: {
                        data: { type: "Array" },
                        status: { type: "Number" },
                    },
                },
                resources: {
                    models: [],
                    tools: [],
                    apis: [{ name: "example", baseUrl: "https://api.example.com", authType: "apiKey" }],
                    credits: 100,
                },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                constraints: {},
            };

            const suggestion = selector.suggestStrategy(context);

            expect(suggestion.recommended).toBe(StrategyType.DETERMINISTIC);
            expect(suggestion.reasoning).to.include("predictable");
            expect(suggestion.reasoning).to.include("output expectations");
        });

        it("should suggest reasoning for analytical tasks", () => {
            const context: ExecutionContext = {
                stepId: "suggest-analysis",
                stepType: "analyze_performance",
                inputs: {
                    metrics: [85, 92, 78, 88, 95],
                    baseline: 80,
                    criteria: "improvement trend",
                    confidence: "high",
                },
                config: {
                    expectedOutputs: {
                        analysis: { type: "Text" },
                        recommendation: { type: "Text" },
                        confidence: { type: "Number" },
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

            const suggestion = selector.suggestStrategy(context);

            expect(suggestion.recommended).toBe(StrategyType.REASONING);
            expect(suggestion.reasoning).to.include("analytical");
            expect(suggestion.reasoning).to.include("High confidence requirement");
        });

        it("should handle no suitable strategy", () => {
            const context: ExecutionContext = {
                stepId: "suggest-none",
                stepType: "impossible_task",
                inputs: {},
                config: {},
                resources: {
                    models: [],
                    tools: [],
                    apis: [],
                    credits: 0,
                },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                constraints: {
                    allowedStrategies: [], // No strategies allowed
                },
            };

            const suggestion = selector.suggestStrategy(context);

            expect(suggestion.recommended).to.be.null;
            expect(suggestion.alternatives).toHaveLength(0);
            expect(suggestion.reasoning).to.include("No suitable strategy");
        });
    });

    describe("getStatistics", () => {
        it("should return comprehensive statistics", () => {
            const stats = selector.getStatistics();

            expect(stats).toHaveProperty("factory");
            expect(stats).toHaveProperty("fallbackOrder");
            
            expect(stats.factory).toHaveProperty("availableStrategies");
            expect(stats.factory).toHaveProperty("usage");
            expect(stats.factory).toHaveProperty("performance");
            
            expect(stats.fallbackOrder).to.be.an("array");
            expect(stats.fallbackOrder).to.include(StrategyType.DETERMINISTIC);
            expect(stats.fallbackOrder).to.include(StrategyType.CONVERSATIONAL);
            expect(stats.fallbackOrder).to.include(StrategyType.REASONING);
        });
    });

    describe("reset", () => {
        it("should reset selector state", async () => {
            // Execute some strategies to create state
            const context: ExecutionContext = {
                stepId: "reset-test",
                stepType: "api_call",
                inputs: {},
                config: {},
                resources: { models: [], tools: [], apis: [], credits: 100 },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                constraints: {},
            };

            await selector.execute(context);
            
            // Get stats before reset
            const statsBefore = selector.getStatistics();
            expect(statsBefore.factory.usage[StrategyType.DETERMINISTIC]).toBeGreaterThan(0);

            // Reset
            selector.reset();

            // Get stats after reset
            const statsAfter = selector.getStatistics();
            expect(statsAfter.factory.usage[StrategyType.DETERMINISTIC] || 0).toBe(0);
        });
    });

    describe("Reasoning generation", () => {
        it("should generate appropriate reasoning for each strategy type", () => {
            const testCases = [
                {
                    context: {
                        stepId: "reason-1",
                        stepType: "chat",
                        inputs: { message: "Hello" },
                        config: {},
                        resources: {
                            models: [{ provider: "openai", model: "gpt-4", capabilities: [], cost: 0.01, available: true }],
                            tools: [],
                            apis: [],
                            credits: 1000,
                        },
                        history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                        constraints: {},
                    },
                    expectedStrategy: StrategyType.CONVERSATIONAL,
                    expectedReasoningKeywords: ["interactive", "Natural language"],
                },
                {
                    context: {
                        stepId: "reason-2",
                        stepType: "transform_data",
                        inputs: { data: [1, 2, 3] },
                        config: {
                            expectedOutputs: { result: { type: "Array" } },
                        },
                        resources: {
                            models: [],
                            tools: [],
                            apis: [],
                            credits: 100,
                        },
                        history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                        constraints: {},
                    },
                    expectedStrategy: StrategyType.DETERMINISTIC,
                    expectedReasoningKeywords: ["predictable", "No LLM"],
                },
                {
                    context: {
                        stepId: "reason-3",
                        stepType: "analyze_complex",
                        inputs: {
                            data1: [1, 2, 3],
                            data2: ["a", "b", "c"],
                            data3: { x: 10, y: 20 },
                            data4: true,
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
                            requiredConfidence: 0.9,
                        },
                    },
                    expectedStrategy: StrategyType.REASONING,
                    expectedReasoningKeywords: ["analytical", "Multiple inputs", "High confidence"],
                },
            ];

            for (const testCase of testCases) {
                const suggestion = selector.suggestStrategy(testCase.context as ExecutionContext);
                
                expect(suggestion.recommended).toBe(testCase.expectedStrategy);
                
                for (const keyword of testCase.expectedReasoningKeywords) {
                    expect(suggestion.reasoning).to.include(keyword);
                }
            }
        });
    });
});