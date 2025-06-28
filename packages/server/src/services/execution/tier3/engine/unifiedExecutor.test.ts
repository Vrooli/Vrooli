import {
    type ExecutionStrategy,
    type StepInput,
} from "@vrooli/shared";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { mockStrategies } from "../../../../__test/fixtures/execution/strategyFixtures.js";
import { mockTools } from "../../../../__test/fixtures/execution/toolFixtures.js";
import { mockLogger } from "../../../../__test/logger.mock.js";
import { EventBus } from "../../../events/eventBus.js";
import { UnifiedExecutor } from "./unifiedExecutor.js";

describe("UnifiedExecutor", () => {
    let unifiedExecutor: UnifiedExecutor;
    let eventBus: EventBus;
    let mockToolRegistry: any;
    let mockStrategyFactory: any;

    beforeEach(() => {
        vi.clearAllMocks();
        eventBus = new EventBus(mockLogger);

        // Mock tool registry
        mockToolRegistry = {
            getTool: vi.fn().mockImplementation((name: string) => {
                const tool = mockTools.availableTools.find(t => t.name === name);
                if (tool) {
                    return {
                        execute: vi.fn().mockResolvedValue({
                            success: true,
                            result: { output: `Result from ${name}` },
                        }),
                    };
                }
                return null;
            }),
            listTools: vi.fn().mockReturnValue(mockTools.availableTools),
        };

        // Mock strategy factory
        mockStrategyFactory = {
            getStrategy: vi.fn().mockImplementation((type: string) => {
                const strategyMap: Record<string, ExecutionStrategy> = {
                    conversational: mockStrategies.conversationalStrategy,
                    reasoning: mockStrategies.reasoningStrategy,
                    deterministic: mockStrategies.deterministicStrategy,
                };
                return strategyMap[type] || mockStrategies.conversationalStrategy;
            }),
        };
    });

    describe("initialization", () => {
        it("should initialize with event bus", () => {
            unifiedExecutor = new UnifiedExecutor(
                mockLogger,
                eventBus,
                mockToolRegistry,
                mockStrategyFactory,
            );

            expect(unifiedExecutor).toBeDefined();
        });

        it("should implement TierCommunicationInterface", () => {
            unifiedExecutor = new UnifiedExecutor(
                mockLogger,
                eventBus,
                mockToolRegistry,
                mockStrategyFactory,
            );

            // Verify interface methods exist
            expect(unifiedExecutor.handleTierRequest).toBeDefined();
            expect(unifiedExecutor.emitTierEvent).toBeDefined();
        });
    });

    describe("step execution with strategies", () => {
        beforeEach(() => {
            unifiedExecutor = new UnifiedExecutor(
                mockLogger,
                eventBus,
                mockToolRegistry,
                mockStrategyFactory,
            );
        });

        it("should execute step with conversational strategy", async () => {
            const stepInput: StepInput = {
                stepId: "conv-step-1",
                context: {
                    runId: "test-run",
                    routineId: "test-routine",
                    inputs: { query: "What is the weather?" },
                    userId: "test-user",
                    metadata: { strategy: "conversational" },
                },
                config: {
                    strategy: "conversational",
                    model: "gpt-4",
                    temperature: 0.7,
                },
            };

            const result = await unifiedExecutor.executeStep(stepInput);

            expect(result.success).toBe(true);
            expect(result.outputs).toBeDefined();
            expect(mockStrategyFactory.getStrategy).toHaveBeenCalledWith("conversational");
        });

        it("should execute step with reasoning strategy", async () => {
            const stepInput: StepInput = {
                stepId: "reason-step-1",
                context: {
                    runId: "test-run",
                    routineId: "test-routine",
                    inputs: { problem: "Calculate optimal route" },
                    userId: "test-user",
                    metadata: { strategy: "reasoning" },
                },
                config: {
                    strategy: "reasoning",
                    model: "gpt-4",
                    reasoningFramework: "chain-of-thought",
                },
            };

            const result = await unifiedExecutor.executeStep(stepInput);

            expect(result.success).toBe(true);
            expect(mockStrategyFactory.getStrategy).toHaveBeenCalledWith("reasoning");
        });

        it("should execute step with deterministic strategy", async () => {
            const stepInput: StepInput = {
                stepId: "det-step-1",
                context: {
                    runId: "test-run",
                    routineId: "test-routine",
                    inputs: { action: "database_query", params: { table: "users" } },
                    userId: "test-user",
                    metadata: { strategy: "deterministic" },
                },
                config: {
                    strategy: "deterministic",
                    toolName: "database_query",
                },
            };

            const result = await unifiedExecutor.executeStep(stepInput);

            expect(result.success).toBe(true);
            expect(mockStrategyFactory.getStrategy).toHaveBeenCalledWith("deterministic");
        });
    });

    describe("tool execution", () => {
        beforeEach(() => {
            unifiedExecutor = new UnifiedExecutor(
                mockLogger,
                eventBus,
                mockToolRegistry,
                mockStrategyFactory,
            );
        });

        it("should execute tools through strategies", async () => {
            const stepInput: StepInput = {
                stepId: "tool-step-1",
                context: {
                    runId: "test-run",
                    routineId: "test-routine",
                    inputs: { toolName: "calculator", operation: "add", values: [1, 2] },
                    userId: "test-user",
                    metadata: {},
                },
                config: {
                    strategy: "deterministic",
                    toolName: "calculator",
                },
            };

            await unifiedExecutor.executeStep(stepInput);

            expect(mockToolRegistry.getTool).toHaveBeenCalledWith("calculator");
        });

        it("should handle missing tools gracefully", async () => {
            mockToolRegistry.getTool.mockReturnValue(null);

            const stepInput: StepInput = {
                stepId: "missing-tool-step",
                context: {
                    runId: "test-run",
                    routineId: "test-routine",
                    inputs: { toolName: "nonexistent_tool" },
                    userId: "test-user",
                    metadata: {},
                },
                config: {
                    strategy: "deterministic",
                    toolName: "nonexistent_tool",
                },
            };

            const result = await unifiedExecutor.executeStep(stepInput);

            expect(result.success).toBe(false);
            expect(result.error).toContain("not found");
        });
    });

    describe("resource tracking", () => {
        beforeEach(() => {
            unifiedExecutor = new UnifiedExecutor(
                mockLogger,
                eventBus,
                mockToolRegistry,
                mockStrategyFactory,
            );
        });

        it("should track resource usage for each execution", async () => {
            // Mock strategy to return specific resource usage
            const mockStrategy: ExecutionStrategy = {
                name: "test-strategy",
                execute: vi.fn().mockResolvedValue({
                    success: true,
                    output: { result: "test output" },
                    resourceUsage: {
                        tokensUsed: 150,
                        creditsUsed: 15,
                        executionTime: 1500,
                    },
                }),
            };
            mockStrategyFactory.getStrategy.mockReturnValue(mockStrategy);

            const stepInput: StepInput = {
                stepId: "resource-step-1",
                context: {
                    runId: "test-run",
                    routineId: "test-routine",
                    inputs: {},
                    userId: "test-user",
                    metadata: {},
                },
                config: { strategy: "test" },
            };

            const result = await unifiedExecutor.executeStep(stepInput);

            expect(result.resourceUsage).toEqual({
                tokens: 150,
                credits: 15,
                duration: 1500,
            });
        });
    });

    describe("event emission", () => {
        beforeEach(() => {
            unifiedExecutor = new UnifiedExecutor(
                mockLogger,
                eventBus,
                mockToolRegistry,
                mockStrategyFactory,
            );
        });

        it("should emit step.started event", async () => {
            const emitSpy = vi.spyOn(eventBus, "emit");

            const stepInput: StepInput = {
                stepId: "event-step-1",
                context: {
                    runId: "test-run",
                    routineId: "test-routine",
                    inputs: {},
                    userId: "test-user",
                    metadata: {},
                },
                config: { strategy: "conversational" },
            };

            await unifiedExecutor.executeStep(stepInput);

            expect(emitSpy).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "step.started",
                    source: "tier3.executor",
                    data: expect.objectContaining({
                        stepId: "event-step-1",
                        runId: "test-run",
                    }),
                }),
            );
        });

        it("should emit step.completed event on success", async () => {
            const emitSpy = vi.spyOn(eventBus, "emit");

            const stepInput: StepInput = {
                stepId: "success-step-1",
                context: {
                    runId: "test-run",
                    routineId: "test-routine",
                    inputs: {},
                    userId: "test-user",
                    metadata: {},
                },
                config: { strategy: "conversational" },
            };

            await unifiedExecutor.executeStep(stepInput);

            expect(emitSpy).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "step.completed",
                    source: "tier3.executor",
                    data: expect.objectContaining({
                        stepId: "success-step-1",
                        success: true,
                    }),
                }),
            );
        });

        it("should emit step.failed event on error", async () => {
            const emitSpy = vi.spyOn(eventBus, "emit");

            // Mock strategy to fail
            const failingStrategy: ExecutionStrategy = {
                name: "failing-strategy",
                execute: vi.fn().mockRejectedValue(new Error("Strategy failed")),
            };
            mockStrategyFactory.getStrategy.mockReturnValue(failingStrategy);

            const stepInput: StepInput = {
                stepId: "fail-step-1",
                context: {
                    runId: "test-run",
                    routineId: "test-routine",
                    inputs: {},
                    userId: "test-user",
                    metadata: {},
                },
                config: { strategy: "failing" },
            };

            await unifiedExecutor.executeStep(stepInput);

            expect(emitSpy).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "step.failed",
                    source: "tier3.executor",
                    data: expect.objectContaining({
                        stepId: "fail-step-1",
                        error: expect.any(String),
                    }),
                }),
            );
        });
    });

    describe("strategy evolution", () => {
        beforeEach(() => {
            unifiedExecutor = new UnifiedExecutor(
                mockLogger,
                eventBus,
                mockToolRegistry,
                mockStrategyFactory,
            );
        });

        it("should support strategy adaptation based on context", async () => {
            // First execution with conversational
            const firstStep: StepInput = {
                stepId: "evolve-step-1",
                context: {
                    runId: "test-run",
                    routineId: "test-routine",
                    inputs: { query: "Calculate 2+2" },
                    userId: "test-user",
                    metadata: {
                        strategy: "conversational",
                        previousExecutions: 0,
                    },
                },
                config: { strategy: "conversational" },
            };

            await unifiedExecutor.executeStep(firstStep);
            expect(mockStrategyFactory.getStrategy).toHaveBeenCalledWith("conversational");

            // Second execution with deterministic (evolved)
            const secondStep: StepInput = {
                stepId: "evolve-step-2",
                context: {
                    runId: "test-run",
                    routineId: "test-routine",
                    inputs: { query: "Calculate 2+2" },
                    userId: "test-user",
                    metadata: {
                        strategy: "deterministic",
                        previousExecutions: 10,
                        optimized: true,
                    },
                },
                config: { strategy: "deterministic" },
            };

            await unifiedExecutor.executeStep(secondStep);
            expect(mockStrategyFactory.getStrategy).toHaveBeenCalledWith("deterministic");
        });
    });

    describe("cross-tier communication", () => {
        beforeEach(() => {
            unifiedExecutor = new UnifiedExecutor(
                mockLogger,
                eventBus,
                mockToolRegistry,
                mockStrategyFactory,
            );
        });

        it("should handle tier requests", async () => {
            const request = {
                type: "EXECUTE_STEP",
                data: {
                    stepId: "tier-req-step",
                    context: {
                        runId: "test-run",
                        routineId: "test-routine",
                        inputs: {},
                        userId: "test-user",
                        metadata: {},
                    },
                    config: { strategy: "conversational" },
                },
            };

            const response = await unifiedExecutor.handleTierRequest(request);

            expect(response.success).toBe(true);
            expect(response.data).toBeDefined();
        });

        it("should emit tier events", () => {
            const emitSpy = vi.spyOn(eventBus, "emit");

            unifiedExecutor.emitTierEvent({
                type: "custom.event",
                data: { test: "data" },
            });

            expect(emitSpy).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "custom.event",
                    source: "tier3.executor",
                    data: { test: "data" },
                }),
            );
        });
    });
});
