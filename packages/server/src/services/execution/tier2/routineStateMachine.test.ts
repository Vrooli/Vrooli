import { describe, expect, test, beforeEach, afterEach, vi, type MockedFunction, type Mock } from "vitest";
import {
    RunState,
    EventTypes,
    generatePK,
    type ExecutionResourceUsage,
    type RunExecutionContext,
    type RunAllocation,
    type ServiceEvent,
    type SessionUser,
} from "@vrooli/shared";
import { RoutineStateMachine } from "./routineStateMachine.js";
import type { ISwarmContextManager } from "../shared/SwarmContextManager.js";
import type { IRunContextManager } from "./runContextManager.js";
import { DbProvider } from "../../../db/provider.js";
import { EventPublisher } from "../../events/publisher.js";
import { logger } from "../../../events/logger.js";

// Mock dependencies
vi.mock("../../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        debug: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
    },
}));

vi.mock("../../events/publisher.js", () => ({
    EventPublisher: {
        emit: vi.fn().mockResolvedValue({ proceed: true, reason: null }),
    },
}));

// Create mock instance that will be returned by DbProvider.get()
const mockDbInstance = {
    run: {
        findUnique: vi.fn(),
    },
};

vi.mock("../../../db/provider.js", () => ({
    DbProvider: {
        init: vi.fn().mockResolvedValue(undefined),
        get: vi.fn(() => mockDbInstance),
    },
}));

vi.mock("@vrooli/shared", async () => {
    const actual = await vi.importActual("@vrooli/shared");
    return {
        ...actual,
        generatePK: vi.fn(() => "test-pk-123"),
    };
});

// Test data factories
function createMockSwarmContextManager(): ISwarmContextManager {
    return {
        updateContext: vi.fn().mockResolvedValue(undefined),
        getSwarmState: vi.fn().mockResolvedValue({
            swarmId: "test-swarm-123",
            version: 1,
            chatConfig: { __version: "1.0.0" },
            execution: { status: RunState.RUNNING, agents: [], activeRuns: [] },
            resources: { allocated: [], consumed: { credits: 0, tokens: 0, time: 0 }, remaining: { credits: 10000, tokens: 10000, time: 3600 } },
            metadata: { createdAt: new Date(), lastUpdated: new Date(), updatedBy: "system", subscribers: new Set() },
        }),
        isSwarmActive: vi.fn().mockResolvedValue(true),
        removeContext: vi.fn().mockResolvedValue(undefined),
        saveSwarmState: vi.fn().mockResolvedValue(undefined),
    } as unknown as ISwarmContextManager;
}

function createMockRunContextManager(): IRunContextManager {
    return {
        getRunContext: vi.fn().mockResolvedValue({
            runId: "test-run-123",
            routineId: "test-routine-123",
            swarmId: "test-swarm-123",
            navigator: null,
            currentLocation: { id: "start", routineId: "test-routine-123", nodeId: "start" },
            visitedLocations: [],
            variables: {},
            outputs: {},
            completedSteps: [],
            parallelBranches: [],
            availableResources: [],
            sharedKnowledge: {},
            resourceLimits: { maxCredits: "1000", maxDurationMs: 300000, maxMemoryMB: 512, maxSteps: 50 },
            resourceUsage: { creditsUsed: "0", durationMs: 0, memoryUsedMB: 0, stepsExecuted: 0, startTime: new Date() },
            progress: { currentStepId: null, completedSteps: [], totalSteps: 0, percentComplete: 0 },
            retryCount: 0,
        }),
        updateRunContext: vi.fn().mockResolvedValue(undefined),
        allocateFromSwarm: vi.fn().mockResolvedValue({
            allocationId: "alloc-123",
            runId: "test-run-123",
            swarmId: "test-swarm-123",
            allocated: { credits: 1000, durationMs: 300000, memoryMB: 512 },
            remaining: { credits: 9000, durationMs: 3300000, memoryMB: 7680 },
            expiresAt: new Date(Date.now() + 3600000),
        }),
        releaseToSwarm: vi.fn().mockResolvedValue(undefined),
        checkResourceAvailability: vi.fn().mockResolvedValue(true),
        getResourceUsage: vi.fn().mockResolvedValue({ creditsUsed: "100", durationMs: 60000, memoryUsedMB: 256, stepsExecuted: 5, toolCalls: 3 }),
        validateContext: vi.fn().mockResolvedValue(true),
        clearContext: vi.fn().mockResolvedValue(undefined),
        emitRunStarted: vi.fn().mockResolvedValue(undefined),
        emitRunCompleted: vi.fn().mockResolvedValue(undefined),
        emitRunFailed: vi.fn().mockResolvedValue(undefined),
        allocateStep: vi.fn().mockResolvedValue({
            stepId: "step-123",
            allocated: { credits: 100, durationMs: 60000, memoryMB: 128 },
        }),
        releaseStep: vi.fn().mockResolvedValue(undefined),
    } as unknown as IRunContextManager;
}

function createMockAllocation(): RunAllocation {
    return {
        allocationId: "alloc-123",
        runId: "test-run-123",
        swarmId: "test-swarm-123",
        allocated: { credits: 1000, durationMs: 300000, memoryMB: 512 },
        remaining: { credits: 9000, durationMs: 3300000, memoryMB: 7680 },
        expiresAt: new Date(Date.now() + 3600000),
    };
}

interface MockRunData {
    id: bigint;
    status: string;  // Database uses string status
    contextSwitches: number;
    completedComplexity: number;
    isPrivate: boolean;
    name: string | null;
    timeElapsed: number | null;
    data?: unknown;
    startedAt?: Date;
    completedAt?: Date | null;
}

function createMockRunData(overrides: Partial<MockRunData> = {}): MockRunData {
    return {
        id: BigInt(123),
        status: "Paused",
        contextSwitches: 2,
        completedComplexity: 50,
        isPrivate: false,
        name: "Test Run",
        timeElapsed: 30000,
        data: null,
        startedAt: new Date(),
        completedAt: null,
        ...overrides,
    };
}

describe("RoutineStateMachine", () => {
    let stateMachine: RoutineStateMachine;
    let mockSwarmContextManager: ISwarmContextManager;
    let mockRunContextManager: IRunContextManager;
    const contextId = "test-context-123";
    const userId = "test-user-123";
    const parentSwarmId = "test-swarm-123";

    beforeEach(() => {
        vi.clearAllMocks();
        vi.useFakeTimers();
        mockSwarmContextManager = createMockSwarmContextManager();
        mockRunContextManager = createMockRunContextManager();
    });

    afterEach(async () => {
        // Stop state machine if it exists
        if (stateMachine && !stateMachine.isDisposed()) {
            try {
                await stateMachine.stop({ mode: "force", reason: "Test cleanup" });
            } catch (error) {
                // Ignore cleanup errors in tests
            }
        }
        
        // Clear all mocks to prevent state leakage between tests
        vi.clearAllMocks();
        
        // Clear all timers to prevent timeout issues
        vi.clearAllTimers();
        
        // Restore real timers
        vi.useRealTimers();
    });

    describe("Constructor and Initialization", () => {
        test("should initialize with UNINITIALIZED state", () => {
            stateMachine = new RoutineStateMachine(
                contextId,
                mockSwarmContextManager,
                mockRunContextManager,
                userId,
                parentSwarmId,
            );

            expect(stateMachine.getState()).toBe(RunState.UNINITIALIZED);
            expect(stateMachine.getTaskId()).toBe(contextId);
            expect(stateMachine.getAssociatedUserId()).toBe(userId);
        });

        test("should work without optional parameters", () => {
            stateMachine = new RoutineStateMachine(contextId, undefined, undefined);
            
            expect(stateMachine.getState()).toBe(RunState.UNINITIALIZED);
            expect(stateMachine.getTaskId()).toBe(contextId);
            expect(stateMachine.getAssociatedUserId()).toBeUndefined();
        });

        test("should initialize resource tracking", () => {
            stateMachine = new RoutineStateMachine(
                contextId,
                mockSwarmContextManager,
                mockRunContextManager,
                userId,
                parentSwarmId,
            );

            const usage = stateMachine.getResourceUsage();
            expect(usage).toEqual({
                creditsUsed: "0",
                durationMs: 0,
                memoryUsedMB: 0,
                stepsExecuted: 0,
                toolCalls: 0,
            });
        });
    });

    describe("State Transitions", () => {
        beforeEach(() => {
            stateMachine = new RoutineStateMachine(
                contextId,
                mockSwarmContextManager,
                mockRunContextManager,
                userId,
                parentSwarmId,
            );
        });

        test("should allow valid state transitions", async () => {
            // Test the happy path
            await stateMachine.transitionTo(RunState.LOADING);
            expect(stateMachine.getState()).toBe(RunState.LOADING);

            await stateMachine.transitionTo(RunState.CONFIGURING);
            expect(stateMachine.getState()).toBe(RunState.CONFIGURING);

            await stateMachine.transitionTo(RunState.READY);
            expect(stateMachine.getState()).toBe(RunState.READY);

            await stateMachine.transitionTo(RunState.RUNNING);
            expect(stateMachine.getState()).toBe(RunState.RUNNING);

            await stateMachine.transitionTo(RunState.COMPLETED);
            expect(stateMachine.getState()).toBe(RunState.COMPLETED);
        });

        test("should reject invalid state transitions", async () => {
            // Cannot go from UNINITIALIZED to RUNNING directly
            await expect(stateMachine.transitionTo(RunState.RUNNING))
                .rejects.toThrow("Invalid state transition from UNINITIALIZED to RUNNING");

            // Cannot go from LOADING to RUNNING directly
            await stateMachine.transitionTo(RunState.LOADING);
            await expect(stateMachine.transitionTo(RunState.RUNNING))
                .rejects.toThrow("Invalid state transition from LOADING to RUNNING");
        });

        test("should handle transitions from RUNNING state", async () => {
            // Setup to RUNNING state
            await stateMachine.transitionTo(RunState.LOADING);
            await stateMachine.transitionTo(RunState.CONFIGURING);
            await stateMachine.transitionTo(RunState.READY);
            await stateMachine.transitionTo(RunState.RUNNING);

            // Can pause
            await stateMachine.transitionTo(RunState.PAUSED);
            expect(stateMachine.getState()).toBe(RunState.PAUSED);

            // Can resume from paused
            await stateMachine.transitionTo(RunState.RUNNING);
            
            // Can suspend
            await stateMachine.transitionTo(RunState.SUSPENDED);
            expect(stateMachine.getState()).toBe(RunState.SUSPENDED);

            // Can resume from suspended
            await stateMachine.transitionTo(RunState.RUNNING);

            // Can fail
            await stateMachine.transitionTo(RunState.FAILED);
            expect(stateMachine.getState()).toBe(RunState.FAILED);
        });

        test("should not allow transitions from terminal states", async () => {
            await stateMachine.transitionTo(RunState.LOADING);
            await stateMachine.transitionTo(RunState.FAILED);

            // Cannot transition from FAILED
            await expect(stateMachine.transitionTo(RunState.RUNNING))
                .rejects.toThrow("Invalid state transition from FAILED to RUNNING");
        });

        test("should persist state when transitioning", async () => {
            // The state machine needs to have a run context for persistence to work
            stateMachine = new RoutineStateMachine(
                contextId,
                mockSwarmContextManager,
                mockRunContextManager,
                userId,
                parentSwarmId,
            );
            
            await stateMachine.transitionTo(RunState.LOADING);

            // Should call updateRunContext as part of state persistence
            expect(mockRunContextManager.updateRunContext).toHaveBeenCalled();
        });

        test("should continue execution if state persistence fails", async () => {
            (mockRunContextManager.getRunContext as Mock).mockRejectedValueOnce(new Error("DB error"));

            // Should not throw
            await stateMachine.transitionTo(RunState.LOADING);
            expect(stateMachine.getState()).toBe(RunState.LOADING);
            expect(logger.warn).toHaveBeenCalledWith(
                "[RoutineStateMachine] Failed to persist routine state, continuing execution",
                expect.any(Object),
            );
        });
    });

    describe("Execution Initialization", () => {
        beforeEach(() => {
            stateMachine = new RoutineStateMachine(
                contextId,
                mockSwarmContextManager,
                mockRunContextManager,
                userId,
                parentSwarmId,
            );
        });

        test("should initialize new execution with resource allocation", async () => {
            const executionId = "exec-123";
            const resourceVersionId = "resource-123";
            const swarmId = "swarm-123";

            await stateMachine.initializeExecution(executionId, resourceVersionId, swarmId);

            // Should transition through states
            expect(stateMachine.getState()).toBe(RunState.READY);

            // Should allocate resources
            expect(mockRunContextManager.allocateFromSwarm).toHaveBeenCalledWith(
                swarmId,
                expect.objectContaining({
                    runId: executionId,
                    routineId: resourceVersionId,
                    priority: "medium",
                }),
            );

            // Should update run context
            expect(mockRunContextManager.updateRunContext).toHaveBeenCalledWith(
                contextId,
                expect.objectContaining({
                    runId: executionId,
                    routineId: resourceVersionId,
                    swarmId,
                }),
            );

            // Should notify swarm
            expect(mockSwarmContextManager.updateContext).toHaveBeenCalled();
        });

        test("should handle resource allocation failure", async () => {
            (mockRunContextManager.allocateFromSwarm as Mock).mockRejectedValueOnce(new Error("No resources"));

            await expect(stateMachine.initializeExecution("exec-123", "resource-123", "swarm-123"))
                .rejects.toThrow("No resources");

            expect(stateMachine.getState()).toBe(RunState.FAILED);
        });

        test("should resume existing run", async () => {
            const runId = "123";
            const mockRun = createMockRunData({ status: "Paused" });
            (mockDbInstance.run.findUnique as Mock).mockResolvedValueOnce(mockRun);

            // Ensure RunContextManager returns valid context for resumption
            (mockRunContextManager.getRunContext as Mock).mockResolvedValueOnce({
                runId,
                routineId: "test-routine-123",
                variables: {},
                resourceUsage: {
                    creditsUsed: "0",
                    durationMs: 0,
                    memoryUsedMB: 0,
                    stepsExecuted: 0,
                    startTime: new Date(),
                },
            });

            await stateMachine.initializeExecution("exec-123", "resource-123", "swarm-123", runId);

            expect(mockDbInstance.run.findUnique).toHaveBeenCalledWith({
                where: { id: BigInt(runId) },
                select: expect.any(Object),
            });

            expect(stateMachine.getState()).toBe(RunState.PAUSED);
        });

        test("should reject resumption of non-resumable runs", async () => {
            const runId = "123";
            const mockRun = createMockRunData({ status: "Completed" });
            const mockDbProvider = DbProvider.get() as ReturnType<typeof vi.fn>;
            
            vi.clearAllMocks();
            (mockDbProvider.run.findUnique as Mock).mockResolvedValueOnce(mockRun);

            await expect(stateMachine.initializeExecution("exec-123", "resource-123", "swarm-123", runId))
                .rejects.toThrow(`Cannot resume run ${runId} - status: Completed`);
        });

        test("should work without RunContextManager", async () => {
            stateMachine = new RoutineStateMachine(contextId, mockSwarmContextManager);

            await stateMachine.initializeExecution("exec-123", "resource-123");
            expect(stateMachine.getState()).toBe(RunState.READY);
        });
    });

    describe("Lifecycle Methods", () => {
        beforeEach(async () => {
            stateMachine = new RoutineStateMachine(
                contextId,
                mockSwarmContextManager,
                mockRunContextManager,
                userId,
                parentSwarmId,
            );
            // Initialize to READY state
            await stateMachine.initializeExecution("exec-123", "resource-123", "swarm-123");
        });

        describe("start", () => {
            test("should start execution from READY state", async () => {
                await stateMachine.start();
                expect(stateMachine.getState()).toBe(RunState.RUNNING);
                expect(logger.info).toHaveBeenCalledWith("[RoutineStateMachine] Starting execution", expect.any(Object));
            });

            test("should reject start from non-READY state", async () => {
                await stateMachine.start();
                await expect(stateMachine.start()).rejects.toThrow("Cannot start from state RUNNING");
            });

            test("should track execution start time", async () => {
                await stateMachine.start();
                const usage = stateMachine.getResourceUsage();
                expect(usage.durationMs).toBe(0); // Just started
            });
        });

        describe("pause", () => {
            test("should pause running execution", async () => {
                await stateMachine.start();
                const result = await stateMachine.pause();
                
                expect(result).toBe(true);
                expect(stateMachine.getState()).toBe(RunState.PAUSED);
                expect(logger.info).toHaveBeenCalledWith("[RoutineStateMachine] Pausing execution", expect.any(Object));
            });

            test("should return false when not running", async () => {
                const result = await stateMachine.pause();
                expect(result).toBe(false);
                expect(logger.warn).toHaveBeenCalledWith("[RoutineStateMachine] Cannot pause from state", expect.any(Object));
            });

            test("should persist pause state", async () => {
                await stateMachine.start();
                await stateMachine.pause();

                expect(mockRunContextManager.updateRunContext).toHaveBeenCalledWith(
                    contextId,
                    expect.objectContaining({
                        pauseData: expect.objectContaining({
                            pausedAt: expect.any(String),
                            resourceUsageSnapshot: expect.any(Object),
                        }),
                    }),
                );
            });
        });

        describe("resume", () => {
            test("should resume from paused state", async () => {
                await stateMachine.start();
                await stateMachine.pause();
                const result = await stateMachine.resume();

                expect(result).toBe(true);
                expect(stateMachine.getState()).toBe(RunState.RUNNING);
            });

            test("should resume from suspended state", async () => {
                await stateMachine.start();
                await stateMachine.suspend();
                const result = await stateMachine.resume();

                expect(result).toBe(true);
                expect(stateMachine.getState()).toBe(RunState.RUNNING);
            });

            test("should return false when not paused/suspended", async () => {
                const result = await stateMachine.resume();
                expect(result).toBe(false);
            });

            test("should restore pause data on resume", async () => {
                const pauseData = {
                    pausedAt: new Date().toISOString(),
                    resourceUsageSnapshot: { creditsUsed: "50" },
                };
                (mockRunContextManager.getRunContext as Mock).mockResolvedValueOnce({
                    ...await mockRunContextManager.getRunContext(contextId),
                    pauseData,
                });

                await stateMachine.start();
                await stateMachine.pause();
                await stateMachine.resume();

                expect(mockRunContextManager.updateRunContext).toHaveBeenCalledWith(
                    contextId,
                    expect.objectContaining({
                        pauseData: null,
                    }),
                );
            });
        });

        describe("suspend", () => {
            test("should suspend running execution", async () => {
                await stateMachine.start();
                await stateMachine.suspend();

                expect(stateMachine.getState()).toBe(RunState.SUSPENDED);
                expect(logger.info).toHaveBeenCalledWith("[RoutineStateMachine] Suspending execution", expect.any(Object));
            });

            test("should reject suspend from non-running state", async () => {
                await expect(stateMachine.suspend()).rejects.toThrow("Cannot suspend from state READY");
            });
        });

        describe("fail", () => {
            test("should fail execution with error", async () => {
                await stateMachine.start();
                await stateMachine.fail("Test error");

                expect(stateMachine.getState()).toBe(RunState.FAILED);
                expect(logger.error).toHaveBeenCalledWith("[RoutineStateMachine] Execution failed", expect.any(Object));
            });

            test("should deallocate resources on failure", async () => {
                const allocation = createMockAllocation();
                await stateMachine.start();
                // Set allocation
                stateMachine["currentAllocation"] = allocation;
                
                await stateMachine.fail("Test error");

                expect(mockRunContextManager.releaseToSwarm).toHaveBeenCalledWith(
                    allocation.swarmId,
                    allocation.allocationId,
                    expect.any(Object),
                );
            });

            test("should not fail from terminal state", async () => {
                await stateMachine.start();
                await stateMachine.complete();
                await stateMachine.fail("Test error");

                // Should still be completed
                expect(stateMachine.getState()).toBe(RunState.COMPLETED);
            });

            test("should update swarm on failure", async () => {
                await stateMachine.start();
                await stateMachine.fail("Test error");

                expect(mockSwarmContextManager.updateContext).toHaveBeenCalledWith(
                    parentSwarmId,
                    expect.objectContaining({
                        execution: expect.objectContaining({
                            activeRuns: expect.any(Array),
                        }),
                    }),
                );
            });
        });

        describe("complete", () => {
            test("should complete execution with result", async () => {
                await stateMachine.start();
                const result = { output: "test result" };
                await stateMachine.complete(result);

                expect(stateMachine.getState()).toBe(RunState.COMPLETED);
            });

            test("should only complete from RUNNING state", async () => {
                await expect(stateMachine.complete()).rejects.toThrow("Cannot complete from state READY");
            });

            test("should deallocate resources on completion", async () => {
                const allocation = createMockAllocation();
                await stateMachine.start();
                stateMachine["currentAllocation"] = allocation;

                await stateMachine.complete();

                expect(mockRunContextManager.releaseToSwarm).toHaveBeenCalledWith(
                    allocation.swarmId,
                    allocation.allocationId,
                    expect.any(Object),
                );
            });

            test("should persist completion data", async () => {
                await stateMachine.start();
                const result = { output: "test" };
                await stateMachine.complete(result);

                expect(mockRunContextManager.updateRunContext).toHaveBeenCalledWith(
                    contextId,
                    expect.objectContaining({
                        completionData: expect.objectContaining({
                            completedAt: expect.any(String),
                            result,
                            resourceUsage: expect.any(Object),
                        }),
                    }),
                );
            });
        });

        describe("cancel", () => {
            test("should cancel execution", async () => {
                await stateMachine.start();
                await stateMachine.cancel();

                expect(stateMachine.getState()).toBe(RunState.CANCELLED);
                expect(logger.info).toHaveBeenCalledWith("[RoutineStateMachine] Cancelling execution", expect.any(Object));
            });

            test("should not cancel from terminal state", async () => {
                await stateMachine.start();
                await stateMachine.complete();
                await stateMachine.cancel();

                expect(stateMachine.getState()).toBe(RunState.COMPLETED);
            });
        });

        describe("stop", () => {
            test("should stop execution gracefully", async () => {
                await stateMachine.start();
                const result = await stateMachine.stop({ mode: "graceful", reason: "User requested" });

                expect(result).toEqual({
                    success: true,
                    state: RunState.CANCELLED,
                    cleanup: {
                        resourcesDeallocated: false,
                        contextCleared: false,
                        swarmNotified: true,
                    },
                });
            });

            test("should force stop execution", async () => {
                await stateMachine.start();
                const result = await stateMachine.stop({ mode: "force", reason: "Emergency" });

                expect(result.success).toBe(true);
                expect(result.state).toBe(RunState.CANCELLED);
            });

            test("should handle cleanup errors gracefully", async () => {
                (mockSwarmContextManager.updateContext as Mock).mockRejectedValueOnce(new Error("Update failed"));

                await stateMachine.start();
                const result = await stateMachine.stop({ mode: "graceful" });

                expect(result.success).toBe(true);
                expect(result.cleanup.swarmNotified).toBe(false);
            });
        });
    });

    describe("Resource Management", () => {
        beforeEach(async () => {
            stateMachine = new RoutineStateMachine(
                contextId,
                mockSwarmContextManager,
                mockRunContextManager,
                userId,
                parentSwarmId,
            );
            await stateMachine.initializeExecution("exec-123", "resource-123", "swarm-123");
        });

        test("should track credits usage", () => {
            stateMachine.addCreditsUsed("100");
            stateMachine.addCreditsUsed("50");

            const usage = stateMachine.getResourceUsage();
            expect(usage.creditsUsed).toBe("150");
        });

        test("should increment step count", () => {
            stateMachine.incrementStepCount();
            stateMachine.incrementStepCount();

            const usage = stateMachine.getResourceUsage();
            expect(usage.stepsExecuted).toBe(2);
        });

        test("should track execution duration", async () => {
            await stateMachine.start();
            
            // Mock time passing
            vi.advanceTimersByTime(5000);
            
            // Force update resource usage
            stateMachine["updateResourceUsage"]();
            
            const usage = stateMachine.getResourceUsage();
            expect(usage.durationMs).toBe(5000);
        });

        test("should get current allocation", () => {
            const allocation = createMockAllocation();
            stateMachine["currentAllocation"] = allocation;

            expect(stateMachine.getCurrentAllocation()).toEqual(allocation);
        });
    });

    describe("Event Handling", () => {
        beforeEach(async () => {
            stateMachine = new RoutineStateMachine(
                contextId,
                mockSwarmContextManager,
                mockRunContextManager,
                userId,
                parentSwarmId,
            );
            await stateMachine.initializeExecution("exec-123", "resource-123", "swarm-123");
        });

        test("should handle emergency stop event", async () => {
            await stateMachine.start();
            
            const event: ServiceEvent = {
                id: "event-123",
                type: "safety/emergency_stop",
                timestamp: new Date(),
                data: { executionId: contextId, reason: "Safety violation" },
                source: "safety-monitor",
            };

            await stateMachine["processEvent"](event);

            expect(stateMachine.getState()).toBe(RunState.FAILED);
            expect(logger.error).toHaveBeenCalledWith(
                "[RoutineStateMachine] Emergency stop triggered",
                expect.any(Object),
            );
        });

        test("should handle step completed event", async () => {
            await stateMachine.start();

            const event: ServiceEvent = {
                id: "event-123",
                type: "step/completed",
                timestamp: new Date(),
                data: { executionId: contextId, stepId: "step-123" },
                source: "step-executor",
            };

            await stateMachine["processEvent"](event);

            const usage = stateMachine.getResourceUsage();
            expect(usage.stepsExecuted).toBe(1);
        });

        test("should filter events by execution ID", async () => {
            const event: ServiceEvent = {
                id: "event-123",
                type: "safety/emergency_stop",
                timestamp: new Date(),
                data: { executionId: "different-execution" },
                source: "safety-monitor",
            };

            const shouldHandle = stateMachine["shouldHandleEvent"](event);
            expect(shouldHandle).toBe(false);
        });

        test("should get event patterns", () => {
            const patterns = stateMachine["getEventPatterns"]();
            
            expect(patterns).toContainEqual({ pattern: `run/${contextId}/*` });
            expect(patterns).toContainEqual({ pattern: `step/${contextId}/*` });
            expect(patterns).toContainEqual({ pattern: `chat/${contextId}/safety/*` });
            expect(patterns).toContainEqual({ pattern: `swarm/${contextId}/safety/*` });
        });
    });

    describe("Query Methods", () => {
        beforeEach(async () => {
            stateMachine = new RoutineStateMachine(
                contextId,
                mockSwarmContextManager,
                mockRunContextManager,
                userId,
                parentSwarmId,
            );
        });

        test("canProceed should return true for non-terminal states", async () => {
            await stateMachine.initializeExecution("exec-123", "resource-123");
            expect(await stateMachine.canProceed()).toBe(true);

            await stateMachine.start();
            expect(await stateMachine.canProceed()).toBe(true);
        });

        test("canProceed should return false for terminal states", async () => {
            await stateMachine.initializeExecution("exec-123", "resource-123");
            await stateMachine.start();
            await stateMachine.complete();

            expect(await stateMachine.canProceed()).toBe(false);
        });

        test("isTerminal should identify terminal states", async () => {
            expect(await stateMachine.isTerminal()).toBe(false);

            await stateMachine.transitionTo(RunState.LOADING);
            expect(await stateMachine.isTerminal()).toBe(false);

            await stateMachine.transitionTo(RunState.FAILED);
            expect(await stateMachine.isTerminal()).toBe(true);
        });
    });

    describe("Error Handling", () => {
        beforeEach(async () => {
            stateMachine = new RoutineStateMachine(
                contextId,
                mockSwarmContextManager,
                mockRunContextManager,
                userId,
                parentSwarmId,
            );
        });

        test("should handle missing run when resuming", async () => {
            (mockDbInstance.run.findUnique as Mock).mockResolvedValueOnce(null);

            await expect(stateMachine.initializeExecution("exec-123", "resource-123", "swarm-123", "123"))
                .rejects.toThrow("Run 123 not found in database");
        });

        test("should determine if error is fatal", async () => {
            await stateMachine.initializeExecution("exec-123", "resource-123");
            await stateMachine.start();

            const event: ServiceEvent = {
                id: "event-123",
                type: "execution/error",
                timestamp: new Date(),
                data: { error: "Out of memory" },
                source: "executor",
            };

            const isFatal = await stateMachine["isErrorFatal"](new Error("Out of memory"), event);
            expect(isFatal).toBe(true);
        });

        test("should handle idle state", async () => {
            await stateMachine.initializeExecution("exec-123", "resource-123");
            
            // Call onIdle when in READY state
            await stateMachine["onIdle"]();

            // onIdle attempts to start, which will throw because we're already READY
            // No specific log to check since it's handled internally
            expect(stateMachine.getState()).toBe(RunState.READY);
        });
    });

    describe("Integration Scenarios", () => {
        beforeEach(async () => {
            stateMachine = new RoutineStateMachine(
                contextId,
                mockSwarmContextManager,
                mockRunContextManager,
                userId,
                parentSwarmId,
            );
        });

        test("should handle full execution lifecycle", async () => {
            // Initialize
            await stateMachine.initializeExecution("exec-123", "resource-123", "swarm-123");
            expect(stateMachine.getState()).toBe(RunState.READY);

            // Start
            await stateMachine.start();
            expect(stateMachine.getState()).toBe(RunState.RUNNING);

            // Execute some steps
            stateMachine.incrementStepCount();
            stateMachine.addCreditsUsed("100");

            // Pause
            await stateMachine.pause();
            expect(stateMachine.getState()).toBe(RunState.PAUSED);

            // Resume
            await stateMachine.resume();
            expect(stateMachine.getState()).toBe(RunState.RUNNING);

            // Complete
            await stateMachine.complete({ success: true });
            expect(stateMachine.getState()).toBe(RunState.COMPLETED);

            // Verify final resource usage
            const usage = stateMachine.getResourceUsage();
            expect(usage.creditsUsed).toBe("100");
            expect(usage.stepsExecuted).toBe(1);
        });

        test("should handle execution failure and cleanup", async () => {
            const allocation = createMockAllocation();
            
            // Initialize with allocation
            await stateMachine.initializeExecution("exec-123", "resource-123", "swarm-123");
            stateMachine["currentAllocation"] = allocation;

            // Start and fail
            await stateMachine.start();
            await stateMachine.fail("Critical error");

            // Verify cleanup
            expect(mockRunContextManager.releaseToSwarm).toHaveBeenCalled();
            expect(mockSwarmContextManager.updateContext).toHaveBeenCalled();
            expect(stateMachine.getState()).toBe(RunState.FAILED);
        });

        test("should handle concurrent events", async () => {
            await stateMachine.initializeExecution("exec-123", "resource-123", "swarm-123");
            await stateMachine.start();

            // Simulate concurrent events
            const events = [
                {
                    id: "event-1",
                    type: "step/completed",
                    timestamp: new Date(),
                    data: { executionId: contextId, stepId: "step-1" },
                    source: "executor",
                },
                {
                    id: "event-2",
                    type: "step/completed",
                    timestamp: new Date(),
                    data: { executionId: contextId, stepId: "step-2" },
                    source: "executor",
                },
            ];

            // Process events concurrently
            await Promise.all(events.map(event => stateMachine["processEvent"](event as ServiceEvent)));

            const usage = stateMachine.getResourceUsage();
            expect(usage.stepsExecuted).toBe(2);
        });

        test("should resume execution with existing context", async () => {
            const existingContext: Partial<RunExecutionContext> = {
                variables: { foo: "bar" },
                completedSteps: ["step-1", "step-2"],
                resourceUsage: {
                    creditsUsed: "50",
                    durationMs: 30000,
                    memoryUsedMB: 256,
                    stepsExecuted: 2,
                    startTime: new Date(Date.now() - 30000),
                },
            };

            (mockRunContextManager.getRunContext as Mock).mockResolvedValueOnce({
                ...await mockRunContextManager.getRunContext(contextId),
                ...existingContext,
            });

            const mockRun = createMockRunData({ status: "Paused" });
            (mockDbInstance.run.findUnique as Mock).mockResolvedValueOnce(mockRun);

            // Ensure RunContextManager returns valid context for resumption
            (mockRunContextManager.getRunContext as Mock).mockResolvedValueOnce({
                runId: "123",
                routineId: "test-routine-123",
                variables: {},
                resourceUsage: {
                    creditsUsed: "50",
                    durationMs: 30000,
                    memoryUsedMB: 256,
                    stepsExecuted: 2,
                    startTime: new Date(Date.now() - 30000),
                },
            });

            await stateMachine.initializeExecution("exec-123", "resource-123", "swarm-123", "123");

            // Verify context was loaded
            const usage = stateMachine.getResourceUsage();
            expect(usage.creditsUsed).toBe("50");
            expect(usage.stepsExecuted).toBe(2);
        });
    });
});
