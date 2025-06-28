import {
    type ExecutionContext,
    type StrategyFactoryConfig,
    StrategyType as StrategyTypeEnum,
    generatePK,
} from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { type Logger } from "winston";
import { type EventBus } from "../../../events/eventBus.js";
import { SimpleStrategyProvider } from "./simpleStrategyProvider.js";

describe("SimpleStrategyProvider", () => {
    let provider: SimpleStrategyProvider;
    let logger: Logger;
    let eventBus: EventBus;
    let config: StrategyFactoryConfig;

    beforeEach(() => {
        logger = {
            info: vi.fn(),
            error: vi.fn(),
            warn: vi.fn(),
            debug: vi.fn(),
        } as unknown as Logger;

        eventBus = {
            publish: vi.fn(),
            subscribe: vi.fn(),
            unsubscribe: vi.fn(),
            on: vi.fn(),
            off: vi.fn(),
        } as unknown as EventBus;

        config = {
            defaultStrategy: StrategyTypeEnum.CONVERSATIONAL,
            fallbackChain: [StrategyTypeEnum.CONVERSATIONAL, StrategyTypeEnum.REASONING, StrategyTypeEnum.DETERMINISTIC],
            adaptationEnabled: true,
            learningRate: 0.1,
        };

        provider = new SimpleStrategyProvider(config, logger, eventBus);

        vi.clearAllMocks();
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("Strategy Selection", () => {
        it("should select strategy from manifest config", async () => {
            const context: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                runId: generatePK(),
                routineId: generatePK(),
                stepType: "conversation",
                inputs: { message: "Hello" },
                config: {
                    strategy: "reasoning", // Explicit strategy in config
                },
                resources: {},
                constraints: {},
            };

            const strategy = await provider.getStrategy(context);

            expect(strategy.type).toBe(StrategyTypeEnum.REASONING);
            expect(eventBus.publish).toHaveBeenCalledWith("strategy.selection", expect.objectContaining({
                stepId: context.stepId,
                declaredStrategy: StrategyTypeEnum.REASONING,
                selectedStrategy: StrategyTypeEnum.REASONING,
                selectionReason: "manifest_declared",
            }));
        });

        it("should select strategy based on step type hints", async () => {
            const deterministicContext: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                runId: generatePK(),
                routineId: generatePK(),
                stepType: "deterministic_calculation",
                inputs: { operation: "add", values: [1, 2, 3] },
                config: {},
                resources: {},
                constraints: {},
            };

            const strategy = await provider.getStrategy(deterministicContext);

            expect(strategy.type).toBe(StrategyTypeEnum.DETERMINISTIC);
        });

        it("should fall back to default strategy when no hints", async () => {
            const context: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                runId: generatePK(),
                routineId: generatePK(),
                stepType: "generic_step",
                inputs: { data: "something" },
                config: {},
                resources: {},
                constraints: {},
            };

            const strategy = await provider.getStrategy(context);

            expect(strategy.type).toBe(StrategyTypeEnum.CONVERSATIONAL); // Default from config
        });

        it("should handle reasoning step type hints", async () => {
            const reasoningContext: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                runId: generatePK(),
                routineId: generatePK(),
                stepType: "analysis_reasoning",
                inputs: { problem: "complex analysis" },
                config: {},
                resources: {},
                constraints: {},
            };

            const strategy = await provider.getStrategy(reasoningContext);

            expect(strategy.type).toBe(StrategyTypeEnum.REASONING);
        });
    });

    describe("Safety Validation", () => {
        it("should override conversational strategy for sensitive data", async () => {
            const sensitiveContext: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                runId: generatePK(),
                routineId: generatePK(),
                stepType: "conversation",
                inputs: { message: "Process this PII data" },
                config: {
                    strategy: "conversational",
                    sensitiveData: true, // Explicit sensitive data flag
                },
                resources: {},
                constraints: {},
            };

            const strategy = await provider.getStrategy(sensitiveContext);

            expect(strategy.type).toBe(StrategyTypeEnum.DETERMINISTIC); // Safety override
            expect(eventBus.publish).toHaveBeenCalledWith("strategy.safety_override", expect.objectContaining({
                stepId: sensitiveContext.stepId,
                originalStrategy: StrategyTypeEnum.CONVERSATIONAL,
                overrideStrategy: StrategyTypeEnum.DETERMINISTIC,
                reason: "sensitive_data_protection",
            }));
        });

        it("should allow conversational strategy when no sensitive data", async () => {
            const normalContext: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                runId: generatePK(),
                routineId: generatePK(),
                stepType: "conversation",
                inputs: { message: "Hello there" },
                config: {
                    strategy: "conversational",
                    sensitiveData: false,
                },
                resources: {},
                constraints: {},
            };

            const strategy = await provider.getStrategy(normalContext);

            expect(strategy.type).toBe(StrategyTypeEnum.CONVERSATIONAL);
            expect(eventBus.publish).not.toHaveBeenCalledWith("strategy.safety_override", expect.anything());
        });
    });

    describe("Event Emission", () => {
        it("should emit strategy selection events for agent analysis", async () => {
            const context: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                runId: generatePK(),
                routineId: generatePK(),
                stepType: "test_step",
                inputs: { data: "test" },
                config: {},
                resources: {},
                constraints: {},
                userId: "test-user",
                swarmId: "test-swarm",
            };

            const usageHints = {
                historicalSuccessRate: 0.8,
                executionFrequency: 10,
                averageComplexity: 0.5,
                userPreference: StrategyTypeEnum.REASONING,
            };

            await provider.getStrategy(context, usageHints);

            expect(eventBus.publish).toHaveBeenCalledWith("strategy.selection", expect.objectContaining({
                stepId: context.stepId,
                routineId: context.routineId,
                declaredStrategy: StrategyTypeEnum.CONVERSATIONAL, // Default
                selectedStrategy: StrategyTypeEnum.CONVERSATIONAL,
                usageHints,
                context: expect.objectContaining({
                    stepType: context.stepType,
                    tier: "tier3",
                    userId: context.userId,
                    swarmId: context.swarmId,
                }),
                selectionReason: "manifest_declared",
                timestamp: expect.any(Date),
            }));
        });

        it("should include usage hints in event data", async () => {
            const context: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                runId: generatePK(),
                routineId: generatePK(),
                stepType: "test",
                inputs: {},
                config: {},
                resources: {},
                constraints: {},
            };

            const usageHints = {
                historicalSuccessRate: 0.9,
                executionFrequency: 5,
                domainRestrictions: ["security", "compliance"],
            };

            await provider.getStrategy(context, usageHints);

            const publishCall = vi.mocked(eventBus.publish).mock.calls[0];
            expect(publishCall[1]).toMatchObject({
                usageHints,
            });
        });
    });

    describe("Strategy Creation", () => {
        it("should create different strategy instances for each type", async () => {
            const context1: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                runId: generatePK(),
                routineId: generatePK(),
                stepType: "conversation",
                inputs: {},
                config: { strategy: "conversational" },
                resources: {},
                constraints: {},
            };

            const context2: ExecutionContext = {
                ...context1,
                stepId: generatePK(),
                config: { strategy: "reasoning" },
            };

            const context3: ExecutionContext = {
                ...context1,
                stepId: generatePK(),
                config: { strategy: "deterministic" },
            };

            const strategy1 = await provider.getStrategy(context1);
            const strategy2 = await provider.getStrategy(context2);
            const strategy3 = await provider.getStrategy(context3);

            expect(strategy1.type).toBe(StrategyTypeEnum.CONVERSATIONAL);
            expect(strategy2.type).toBe(StrategyTypeEnum.REASONING);
            expect(strategy3.type).toBe(StrategyTypeEnum.DETERMINISTIC);

            // Strategies should be different instances
            expect(strategy1).not.toBe(strategy2);
            expect(strategy2).not.toBe(strategy3);
        });

        it("should reuse strategy instances for same type", async () => {
            const context1: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                runId: generatePK(),
                routineId: generatePK(),
                stepType: "test",
                inputs: {},
                config: { strategy: "conversational" },
                resources: {},
                constraints: {},
            };

            const context2: ExecutionContext = {
                ...context1,
                stepId: generatePK(),
            };

            const strategy1 = await provider.getStrategy(context1);
            const strategy2 = await provider.getStrategy(context2);

            expect(strategy1).toBe(strategy2); // Same instance reused
        });

        it("should handle unknown strategy types gracefully", async () => {
            const context: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                runId: generatePK(),
                routineId: generatePK(),
                stepType: "test",
                inputs: {},
                config: { strategy: "unknown_strategy" },
                resources: {},
                constraints: {},
            };

            const strategy = await provider.getStrategy(context);

            expect(strategy.type).toBe(StrategyTypeEnum.CONVERSATIONAL); // Falls back to default
        });
    });

    // StrategyProviderCompat tests removed - compatibility wrapper deprecated

    describe("Data-Driven Configuration", () => {
        it("should respect configured default strategy", async () => {
            const customConfig = {
                ...config,
                defaultStrategy: StrategyTypeEnum.REASONING,
            };

            const customProvider = new SimpleStrategyProvider(customConfig, logger, eventBus);

            const context: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                runId: generatePK(),
                routineId: generatePK(),
                stepType: "generic",
                inputs: {},
                config: {}, // No explicit strategy
                resources: {},
                constraints: {},
            };

            const strategy = await customProvider.getStrategy(context);

            expect(strategy.type).toBe(StrategyTypeEnum.REASONING); // Custom default
        });

        it("should validate strategy types against enum", async () => {
            const context: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                runId: generatePK(),
                routineId: generatePK(),
                stepType: "test",
                inputs: {},
                config: { strategy: "invalid_strategy" },
                resources: {},
                constraints: {},
            };

            const strategy = await provider.getStrategy(context);

            // Should fall back to default when invalid strategy provided
            expect(strategy.type).toBe(StrategyTypeEnum.CONVERSATIONAL);
        });
    });
});
