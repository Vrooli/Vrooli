import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { TierTwoRunStateMachine } from "./runStateMachine.js";
import { type Logger } from "winston";
import { EventBus } from "../../cross-cutting/events/eventBus.js";
import { type IRunStateStore } from "../state/runStateStore.js";
import { type TierCommunicationInterface, type TierExecutionRequest, type ExecutionResult } from "@vrooli/shared";
import { InMemoryRunStateStore } from "../state/inMemoryRunStateStore.js";
import { BaseStates } from "../../shared/BaseStateMachine.js";
import { generatePK, RunState, RunEventType } from "@vrooli/shared";

/**
 * RunStateMachine Tests - Process Intelligence Layer
 * 
 * These tests validate that Tier 2 provides minimal orchestration infrastructure
 * while enabling complex behaviors to emerge from:
 * 
 * 1. **Navigator Pattern**: Platform-agnostic routine execution
 * 2. **Event-Driven Coordination**: Orchestration agents respond to patterns
 * 3. **Minimal State Management**: Only operational states, not strategies
 * 4. **Resource-Aware Execution**: Credit/time limits without optimization logic
 * 5. **Cross-Tier Communication**: Clean interfaces between tiers
 * 
 * The tests avoid prescriptive orchestration logic and instead verify the
 * infrastructure enables emergent optimization through agent analysis.
 */

describe("TierTwoRunStateMachine - Adaptive Process Intelligence", () => {
    let logger: Logger;
    let eventBus: EventBus;
    let stateStore: IRunStateStore;
    let tier3Executor: TierCommunicationInterface;
    let runStateMachine: TierTwoRunStateMachine;

    beforeEach(() => {
        logger = {
            info: vi.fn(),
            error: vi.fn(),
            warn: vi.fn(),
            debug: vi.fn(),
        } as unknown as Logger;
        eventBus = new EventBus(logger);
        stateStore = new InMemoryRunStateStore();
        
        // Mock Tier 3 executor - represents execution intelligence
        tier3Executor = {
            execute: vi.fn(),
            getMetrics: vi.fn().mockResolvedValue({
                activeExecutions: 0,
                totalExecutions: 0,
                averageExecutionTime: 0,
            }),
        };

        runStateMachine = new TierTwoRunStateMachine(
            logger,
            eventBus,
            stateStore,
            tier3Executor,
        );
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("Minimal Orchestration Infrastructure", () => {
        it("should provide basic run lifecycle management", async () => {
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                type: "routine",
                userId: generatePK(),
                payload: {
                    routineId: generatePK(),
                    inputs: { query: "test input" },
                },
            };

            // Mock successful execution
            vi.mocked(tier3Executor.execute).mockResolvedValue({
                executionId: request.executionId,
                status: "completed" as ExecutionStatus,
                data: { result: "processed" },
            });

            const result = await runStateMachine.execute(request);

            expect(result.status).toBe("completed");
            expect(result.data).toEqual({ result: "processed" });
            
            // Verify delegation to Tier 3
            expect(tier3Executor.execute).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "step",
                    payload: expect.objectContaining({
                        runId: expect.any(String),
                    }),
                }),
            );
        });

        it("should manage run state transitions without prescriptive logic", async () => {
            const runId = generatePK();
            const run = {
                id: runId,
                state: RunState.Created,
                userId: generatePK(),
                routineId: generatePK(),
                progress: { currentLocation: null },
                config: {},
            };

            // Store initial run
            await stateStore.saveRun(run);

            // Transition states through events
            await runStateMachine["transitionRunState"](runId, RunState.Running);
            
            const updatedRun = await stateStore.getRun(runId);
            expect(updatedRun?.state).toBe(RunState.Running);

            // Complete the run
            await runStateMachine["transitionRunState"](runId, RunState.Completed);
            
            const completedRun = await stateStore.getRun(runId);
            expect(completedRun?.state).toBe(RunState.Completed);
        });
    });

    describe("Navigator Pattern for Universal Execution", () => {
        it("should execute routines without knowing their format", async () => {
            // Test different routine formats through same interface
            const routineFormats = [
                { format: "native", routineId: "native-123" },
                { format: "bpmn", routineId: "bpmn-456" },
                { format: "langchain", routineId: "langchain-789" },
            ];

            for (const { format, routineId } of routineFormats) {
                const request: TierExecutionRequest = {
                    executionId: generatePK(),
                    type: "routine",
                    userId: generatePK(),
                    payload: {
                        routineId,
                        inputs: { format },
                        metadata: { sourceFormat: format },
                    },
                };

                vi.mocked(tier3Executor.execute).mockResolvedValue({
                    executionId: request.executionId,
                    status: "completed" as ExecutionStatus,
                    data: { processedFormat: format },
                });

                const result = await runStateMachine.execute(request);

                // Same interface regardless of format
                expect(result.status).toBe("completed");
                expect(tier3Executor.execute).toHaveBeenCalledWith(
                    expect.objectContaining({
                        type: "step",
                        payload: expect.objectContaining({
                            routineId,
                        }),
                    }),
                );
            }
        });

        it("should handle multi-step routines through delegation", async () => {
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                type: "routine",
                userId: generatePK(),
                payload: {
                    routineId: generatePK(),
                    inputs: { steps: 3 },
                },
            };

            // Simulate multi-step execution
            const executionResults = [
                { status: "in_progress" as ExecutionStatus, data: { step: 1 } },
                { status: "in_progress" as ExecutionStatus, data: { step: 2 } },
                { status: "completed" as ExecutionStatus, data: { step: 3, final: true } },
            ];

            let callCount = 0;
            vi.mocked(tier3Executor.execute).mockImplementation(async () => {
                return {
                    executionId: request.executionId,
                    ...executionResults[callCount++],
                };
            });

            const result = await runStateMachine.execute(request);

            // Should complete after multiple steps
            expect(result.status).toBe("completed");
            expect(result.data).toEqual({ step: 3, final: true });
            expect(tier3Executor.execute).toHaveBeenCalledTimes(3);
        });
    });

    describe("Event-Driven Orchestration Patterns", () => {
        it("should emit events for orchestration agents to analyze", async () => {
            const capturedEvents: any[] = [];
            await eventBus.subscribe("run.*", async (event) => {
                capturedEvents.push(event);
            });

            const request: TierExecutionRequest = {
                executionId: generatePK(),
                type: "routine",
                userId: generatePK(),
                payload: {
                    routineId: generatePK(),
                    inputs: { action: "process_data" },
                },
            };

            vi.mocked(tier3Executor.execute).mockResolvedValue({
                executionId: request.executionId,
                status: "completed" as ExecutionStatus,
                data: { processed: true },
            });

            await runStateMachine.execute(request);

            // Wait for events to propagate
            await new Promise(resolve => setTimeout(resolve, 100));

            // Verify events emitted for pattern analysis
            expect(capturedEvents).toContainEqual(
                expect.objectContaining({
                    type: "run.created",
                    data: expect.objectContaining({
                        runId: expect.any(String),
                        routineId: request.payload.routineId,
                    }),
                }),
            );

            expect(capturedEvents).toContainEqual(
                expect.objectContaining({
                    type: "run.completed",
                    data: expect.objectContaining({
                        runId: expect.any(String),
                    }),
                }),
            );
        });

        it("should enable orchestration agents to optimize through events", async () => {
            // Simulate orchestration agent listening for patterns
            const orchestrationInsights: any[] = [];
            await eventBus.subscribe("run.step.*", async (event) => {
                // Agent analyzes step patterns
                if (event.data.duration > 1000) {
                    orchestrationInsights.push({
                        pattern: "slow_step",
                        stepId: event.data.stepId,
                        suggestion: "consider_caching",
                    });
                }
            });

            const request: TierExecutionRequest = {
                executionId: generatePK(),
                type: "routine",
                userId: generatePK(),
                payload: {
                    routineId: generatePK(),
                    inputs: {},
                },
            };

            // Simulate slow execution
            vi.mocked(tier3Executor.execute).mockImplementation(async () => {
                await new Promise(resolve => setTimeout(resolve, 50));
                
                // Emit step event
                await eventBus.publish("run.step.completed", {
                    stepId: generatePK(),
                    duration: 1500, // Slow step
                    runId: request.executionId,
                });

                return {
                    executionId: request.executionId,
                    status: "completed" as ExecutionStatus,
                    data: {},
                };
            });

            await runStateMachine.execute(request);
            await new Promise(resolve => setTimeout(resolve, 100));

            // Orchestration agent should have detected pattern
            expect(orchestrationInsights).toHaveLength(1);
            expect(orchestrationInsights[0].pattern).toBe("slow_step");
        });
    });

    describe("Resource-Aware Execution", () => {
        it("should enforce resource limits without optimization logic", async () => {
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                type: "routine",
                userId: generatePK(),
                payload: {
                    routineId: generatePK(),
                    inputs: {},
                    config: {
                        maxCredits: 100,
                        maxDuration: 5000, // 5 seconds
                    },
                },
            };

            // Simulate resource exhaustion
            vi.mocked(tier3Executor.execute).mockResolvedValue({
                executionId: request.executionId,
                status: "failed" as ExecutionStatus,
                error: "Credit limit exceeded",
                metadata: {
                    creditsUsed: 101,
                },
            });

            const result = await runStateMachine.execute(request);

            expect(result.status).toBe("failed");
            expect(result.error).toBe("Credit limit exceeded");
            
            // No optimization attempted - just enforcement
            expect(tier3Executor.execute).toHaveBeenCalledOnce();
        });

        it("should emit resource usage for monitoring agents", async () => {
            const resourceEvents: any[] = [];
            await eventBus.subscribe("run.resource.*", async (event) => {
                resourceEvents.push(event);
            });

            const request: TierExecutionRequest = {
                executionId: generatePK(),
                type: "routine",
                userId: generatePK(),
                payload: {
                    routineId: generatePK(),
                    inputs: {},
                },
            };

            vi.mocked(tier3Executor.execute).mockResolvedValue({
                executionId: request.executionId,
                status: "completed" as ExecutionStatus,
                data: {},
                metadata: {
                    creditsUsed: 50,
                    executionTime: 2500,
                },
            });

            await runStateMachine.execute(request);
            await new Promise(resolve => setTimeout(resolve, 100));

            // Resource monitoring agents can analyze usage
            expect(resourceEvents).toContainEqual(
                expect.objectContaining({
                    type: "run.resource.consumed",
                    data: expect.objectContaining({
                        creditsUsed: 50,
                        executionTime: 2500,
                    }),
                }),
            );
        });
    });

    describe("Cross-Tier Communication", () => {
        it("should maintain clean interfaces between tiers", async () => {
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                type: "routine",
                userId: generatePK(),
                payload: {
                    routineId: generatePK(),
                    inputs: { data: "test" },
                },
            };

            vi.mocked(tier3Executor.execute).mockResolvedValue({
                executionId: request.executionId,
                status: "completed" as ExecutionStatus,
                data: { transformed: "TEST" },
            });

            const result = await runStateMachine.execute(request);

            // Verify clean request transformation
            expect(tier3Executor.execute).toHaveBeenCalledWith({
                executionId: expect.any(String),
                type: "step",
                userId: request.userId,
                payload: {
                    runId: expect.any(String),
                    stepId: expect.any(String),
                    routineId: request.payload.routineId,
                    inputs: request.payload.inputs,
                    context: {},
                },
            });

            // Clean response pass-through
            expect(result).toEqual({
                executionId: request.executionId,
                status: "completed",
                data: { transformed: "TEST" },
            });
        });

        it("should handle tier communication failures gracefully", async () => {
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                type: "routine",
                userId: generatePK(),
                payload: {
                    routineId: generatePK(),
                    inputs: {},
                },
            };

            // Simulate Tier 3 failure
            vi.mocked(tier3Executor.execute).mockRejectedValue(
                new Error("Tier 3 temporarily unavailable"),
            );

            const result = await runStateMachine.execute(request);

            expect(result.status).toBe("failed");
            expect(result.error).toContain("Tier 3 temporarily unavailable");
            
            // Run should be marked as failed
            const runs = await stateStore.getUserRuns(request.userId);
            expect(runs[0]?.state).toBe(RunState.Failed);
        });
    });

    describe("Parallel Branch Coordination", () => {
        it("should enable parallel execution without coordination logic", async () => {
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                type: "routine",
                userId: generatePK(),
                payload: {
                    routineId: generatePK(),
                    inputs: {},
                    metadata: {
                        branches: ["branch-1", "branch-2", "branch-3"],
                    },
                },
            };

            // Track execution order
            const executionOrder: string[] = [];
            vi.mocked(tier3Executor.execute).mockImplementation(async (req) => {
                const branchId = req.payload.branchId || "main";
                executionOrder.push(branchId);
                
                // Simulate varying execution times
                await new Promise(resolve => 
                    setTimeout(resolve, Math.random() * 100),
                );
                
                return {
                    executionId: req.executionId,
                    status: "completed" as ExecutionStatus,
                    data: { branch: branchId },
                };
            });

            const result = await runStateMachine.execute(request);

            // Infrastructure enables parallel execution
            // Coordination logic comes from agents analyzing patterns
            expect(result.status).toBe("completed");
            expect(executionOrder.length).toBeGreaterThan(0);
        });
    });

    describe("Checkpointing and Recovery", () => {
        it("should save checkpoints without recovery strategies", async () => {
            const runId = generatePK();
            const checkpoint = {
                location: { x: 1, y: 2 },
                progress: { completed: 5, total: 10 },
                timestamp: new Date().toISOString(),
            };

            await runStateMachine["saveCheckpoint"](runId, checkpoint);

            const savedCheckpoint = await stateStore.getCheckpoint(runId);
            expect(savedCheckpoint).toEqual(checkpoint);
            
            // No recovery logic - agents determine recovery strategies
        });

        it("should emit checkpoint events for recovery agents", async () => {
            const checkpointEvents: any[] = [];
            await eventBus.subscribe("run.checkpoint.*", async (event) => {
                checkpointEvents.push(event);
            });

            const request: TierExecutionRequest = {
                executionId: generatePK(),
                type: "routine",
                userId: generatePK(),
                payload: {
                    routineId: generatePK(),
                    inputs: {},
                },
            };

            // Simulate execution with checkpoint
            vi.mocked(tier3Executor.execute).mockImplementation(async () => {
                await eventBus.publish("run.checkpoint.saved", {
                    runId: request.executionId,
                    checkpoint: {
                        step: "data_processing",
                        progress: 0.5,
                    },
                });
                
                return {
                    executionId: request.executionId,
                    status: "completed" as ExecutionStatus,
                    data: {},
                };
            });

            await runStateMachine.execute(request);
            await new Promise(resolve => setTimeout(resolve, 100));

            // Recovery agents can use checkpoint data
            expect(checkpointEvents).toHaveLength(1);
            expect(checkpointEvents[0].data.checkpoint.step).toBe("data_processing");
        });
    });
});
