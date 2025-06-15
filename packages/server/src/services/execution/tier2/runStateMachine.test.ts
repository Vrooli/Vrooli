import { describe, it, expect, vi, beforeEach } from "vitest";
import { TierTwoRunStateMachine } from "./runStateMachine.js";
import { EventBus } from "../cross-cutting/events/eventBus.js";
import { mockLogger } from "../../../__test/logger.mock.js";
import { mockRunOrchestration, mockRoutineNavigation } from "../../../__test/fixtures/execution/runFixtures.js";
import { mockRoutines } from "../../../__test/fixtures/execution/routineFixtures.js";
import { type RunContext, type TierThreeExecutor, RunStatus } from "@vrooli/shared";

describe("TierTwoRunStateMachine", () => {
    let runStateMachine: TierTwoRunStateMachine;
    let eventBus: EventBus;
    let mockTierThree: TierThreeExecutor;
    let mockNavigator: any;

    beforeEach(() => {
        vi.clearAllMocks();
        eventBus = new EventBus(mockLogger);
        
        // Mock Tier 3 executor
        mockTierThree = {
            executeStep: vi.fn().mockResolvedValue({
                success: true,
                outputs: { result: "test output" },
                resourceUsage: { tokens: 100, credits: 10, duration: 1000 },
            }),
        } as unknown as TierThreeExecutor;

        // Mock navigator
        mockNavigator = {
            getNextStep: vi.fn().mockReturnValue({ stepId: "step-1", type: "action" }),
            recordStepResult: vi.fn(),
            isComplete: vi.fn().mockReturnValue(false),
            getOutputs: vi.fn().mockReturnValue({ finalResult: "completed" }),
        };
    });

    describe("initialization", () => {
        it("should initialize with correct run ID and context", () => {
            const mockRun = mockRunOrchestration.basicRun;
            const context: RunContext = {
                runId: mockRun.id,
                routineId: mockRun.routineId,
                inputs: mockRun.inputs,
                userId: "test-user",
                metadata: {},
            };

            runStateMachine = new TierTwoRunStateMachine(
                mockRun.id,
                context,
                mockNavigator,
                mockLogger,
                eventBus,
                mockTierThree,
            );

            expect(runStateMachine.getRunId()).toBe(mockRun.id);
        });

        it("should start in UNINITIALIZED state", () => {
            const context: RunContext = {
                runId: "test-run",
                routineId: "test-routine",
                inputs: {},
                userId: "test-user",
                metadata: {},
            };

            runStateMachine = new TierTwoRunStateMachine(
                "test-run",
                context,
                mockNavigator,
                mockLogger,
                eventBus,
                mockTierThree,
            );

            // Using BaseStates from actual implementation
            expect(runStateMachine.getCurrentState()).toBe("UNINITIALIZED");
        });
    });

    describe("state transitions", () => {
        let runContext: RunContext;

        beforeEach(() => {
            runContext = {
                runId: "test-run",
                routineId: "test-routine",
                inputs: { test: "input" },
                userId: "test-user",
                metadata: {},
            };
            runStateMachine = new TierTwoRunStateMachine(
                "test-run",
                runContext,
                mockNavigator,
                mockLogger,
                eventBus,
                mockTierThree,
            );
        });

        it("should transition from UNINITIALIZED to RUNNING on start", async () => {
            const result = await runStateMachine.start();

            expect(result.runId).toBe("test-run");
            expect(runStateMachine.getCurrentState()).toBe("RUNNING");
        });

        it("should handle START_EXECUTION event", async () => {
            await runStateMachine.start();

            const event = {
                type: "START_EXECUTION",
                data: { immediate: true },
            };

            const result = await runStateMachine.processEvent(event);
            expect(result).toBeDefined();
            expect(result.message).toContain("execution");
        });

        it("should handle pause request", async () => {
            await runStateMachine.start();
            const paused = await runStateMachine.requestPause();

            expect(paused).toBe(true);
            expect(runStateMachine.getCurrentState()).toBe("PAUSED");
        });

        it("should handle stop request", async () => {
            await runStateMachine.start();
            const stopped = await runStateMachine.requestStop("Test stop");

            expect(stopped).toBe(true);
            expect(runStateMachine.getCurrentState()).toBe("STOPPED");
        });
    });

    describe("routine navigation", () => {
        it("should navigate through routine steps", async () => {
            const mockRun = mockRunOrchestration.parallelRun;
            const runContext: RunContext = {
                runId: mockRun.id,
                routineId: mockRun.routineId,
                inputs: mockRun.inputs,
                userId: "test-user",
                metadata: {},
            };

            // Setup navigator to return steps sequentially
            let stepCount = 0;
            mockNavigator.getNextStep.mockImplementation(() => {
                stepCount++;
                if (stepCount > 3) {
                    mockNavigator.isComplete.mockReturnValue(true);
                    return null;
                }
                return { stepId: `step-${stepCount}`, type: "action" };
            });

            runStateMachine = new TierTwoRunStateMachine(
                mockRun.id,
                runContext,
                mockNavigator,
                mockLogger,
                eventBus,
                mockTierThree,
            );

            await runStateMachine.start();

            // Process execution
            await runStateMachine.processEvent({
                type: "START_EXECUTION",
                data: {},
            });

            // Verify steps were executed
            expect(mockTierThree.executeStep).toHaveBeenCalledTimes(3);
            expect(mockNavigator.recordStepResult).toHaveBeenCalledTimes(3);
        });

        it("should handle parallel branch execution", async () => {
            const mockParallel = mockRoutineNavigation.parallelBranches;
            const runContext: RunContext = {
                runId: "parallel-run",
                routineId: "parallel-routine",
                inputs: {},
                userId: "test-user",
                metadata: {},
            };

            // Mock navigator for parallel branches
            mockNavigator.getNextStep.mockReturnValueOnce({ 
                stepId: "branch-1-step-1", 
                type: "action",
                branchId: "branch-1",
            }).mockReturnValueOnce({ 
                stepId: "branch-2-step-1", 
                type: "action",
                branchId: "branch-2",
            }).mockReturnValueOnce(null);

            runStateMachine = new TierTwoRunStateMachine(
                "parallel-run",
                runContext,
                mockNavigator,
                mockLogger,
                eventBus,
                mockTierThree,
            );

            await runStateMachine.start();
            await runStateMachine.processEvent({
                type: "START_EXECUTION",
                data: {},
            });

            // Verify both branches were executed
            expect(mockTierThree.executeStep).toHaveBeenCalledTimes(2);
        });
    });

    describe("error handling", () => {
        beforeEach(() => {
            const runContext: RunContext = {
                runId: "error-test-run",
                routineId: "test-routine",
                inputs: {},
                userId: "test-user",
                metadata: {},
            };
            runStateMachine = new TierTwoRunStateMachine(
                "error-test-run",
                runContext,
                mockNavigator,
                mockLogger,
                eventBus,
                mockTierThree,
            );
        });

        it("should handle execution errors", async () => {
            // Mock Tier 3 to return an error
            mockTierThree.executeStep = vi.fn().mockResolvedValueOnce({
                success: false,
                error: "Step execution failed",
                resourceUsage: { tokens: 0, credits: 0, duration: 0 },
            });

            await runStateMachine.start();
            
            const result = await runStateMachine.processEvent({
                type: "START_EXECUTION",
                data: {},
            });

            expect(runStateMachine.getCurrentState()).toBe("FAILED");
        });

        it("should emit error events", async () => {
            const emitSpy = vi.spyOn(eventBus, "emit");
            
            // Mock Tier 3 to throw an error
            mockTierThree.executeStep = vi.fn().mockRejectedValueOnce(
                new Error("Critical execution error"),
            );

            await runStateMachine.start();
            await runStateMachine.processEvent({
                type: "START_EXECUTION",
                data: {},
            });

            // Check that error event was emitted
            expect(emitSpy).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "run.failed",
                    source: "tier2.run",
                    data: expect.objectContaining({
                        runId: "error-test-run",
                        error: expect.any(String),
                    }),
                }),
            );
        });
    });

    describe("cross-tier communication", () => {
        it("should coordinate with Tier 3 for step execution", async () => {
            const runContext: RunContext = {
                runId: "tier-comm-run",
                routineId: "test-routine",
                inputs: { testData: "value" },
                userId: "test-user",
                metadata: {},
            };

            runStateMachine = new TierTwoRunStateMachine(
                "tier-comm-run",
                runContext,
                mockNavigator,
                mockLogger,
                eventBus,
                mockTierThree,
            );

            await runStateMachine.start();
            await runStateMachine.processEvent({
                type: "START_EXECUTION",
                data: {},
            });

            // Verify Tier 3 was called with correct parameters
            expect(mockTierThree.executeStep).toHaveBeenCalledWith(
                expect.objectContaining({
                    stepId: "step-1",
                    context: expect.objectContaining({
                        runId: "tier-comm-run",
                        inputs: { testData: "value" },
                    }),
                }),
            );
        });

        it("should emit events for Tier 1 coordination", async () => {
            const emitSpy = vi.spyOn(eventBus, "emit");
            const runContext: RunContext = {
                runId: "event-test-run",
                routineId: "test-routine",
                inputs: {},
                userId: "test-user",
                metadata: {},
            };

            // Complete after first step
            mockNavigator.isComplete.mockReturnValue(true);
            mockNavigator.getOutputs.mockReturnValue({ result: "done" });

            runStateMachine = new TierTwoRunStateMachine(
                "event-test-run",
                runContext,
                mockNavigator,
                mockLogger,
                eventBus,
                mockTierThree,
            );

            await runStateMachine.start();
            await runStateMachine.processEvent({
                type: "START_EXECUTION",
                data: {},
            });

            // Check run.completed event was emitted
            expect(emitSpy).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "run.completed",
                    source: "tier2.run",
                    data: expect.objectContaining({
                        runId: "event-test-run",
                        outputs: { result: "done" },
                    }),
                }),
            );
        });
    });

    describe("resource tracking", () => {
        it("should track resource usage across steps", async () => {
            const runContext: RunContext = {
                runId: "resource-test-run",
                routineId: "test-routine",
                inputs: {},
                userId: "test-user",
                metadata: {},
            };

            // Mock different resource usage for each step
            mockTierThree.executeStep = vi.fn()
                .mockResolvedValueOnce({
                    success: true,
                    outputs: { step1: "done" },
                    resourceUsage: { tokens: 100, credits: 10, duration: 1000 },
                })
                .mockResolvedValueOnce({
                    success: true,
                    outputs: { step2: "done" },
                    resourceUsage: { tokens: 200, credits: 20, duration: 2000 },
                });

            // Two steps then complete
            mockNavigator.getNextStep
                .mockReturnValueOnce({ stepId: "step-1", type: "action" })
                .mockReturnValueOnce({ stepId: "step-2", type: "action" })
                .mockReturnValueOnce(null);
            mockNavigator.isComplete.mockReturnValue(true);

            runStateMachine = new TierTwoRunStateMachine(
                "resource-test-run",
                runContext,
                mockNavigator,
                mockLogger,
                eventBus,
                mockTierThree,
            );

            await runStateMachine.start();
            await runStateMachine.processEvent({
                type: "START_EXECUTION",
                data: {},
            });

            // Verify resource tracking
            expect(mockNavigator.recordStepResult).toHaveBeenCalledWith(
                "step-1",
                expect.objectContaining({
                    resourceUsage: { tokens: 100, credits: 10, duration: 1000 },
                }),
            );
            expect(mockNavigator.recordStepResult).toHaveBeenCalledWith(
                "step-2",
                expect.objectContaining({
                    resourceUsage: { tokens: 200, credits: 20, duration: 2000 },
                }),
            );
        });
    });
});