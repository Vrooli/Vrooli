import { describe, it, expect, beforeEach, vi } from "vitest";
import { RunState } from "@vrooli/shared";
import { type Logger } from "winston";
import { RoutineStateMachine } from "./RoutineStateMachine.js";
import { type IEventBus } from "../../../events/types.js";
import { type ISwarmContextManager } from "../../shared/SwarmContextManager.js";

describe("RoutineStateMachine", () => {
    let mockLogger: Logger;
    let mockEventBus: IEventBus;
    let mockSwarmContextManager: ISwarmContextManager;
    let stateMachine: RoutineStateMachine;
    const contextId = "test-context-123";

    beforeEach(() => {
        // Mock logger
        mockLogger = {
            info: vi.fn(),
            debug: vi.fn(),
            warn: vi.fn(),
            error: vi.fn(),
        } as any;

        // Mock event bus
        mockEventBus = {
            publish: vi.fn().mockResolvedValue(undefined),
            on: vi.fn(),
            off: vi.fn(),
        } as any;

        // Mock SwarmContextManager
        mockSwarmContextManager = {
            updateContext: vi.fn().mockResolvedValue(undefined),
            getContext: vi.fn(),
        } as any;

        // Create state machine
        stateMachine = new RoutineStateMachine(
            contextId,
            mockSwarmContextManager,
            mockEventBus,
            mockLogger,
        );
    });

    describe("Initial State", () => {
        it("should start in UNINITIALIZED state", () => {
            expect(stateMachine.getState()).toBe(RunState.UNINITIALIZED);
        });

        it("should return contextId as taskId", () => {
            expect(stateMachine.getTaskId()).toBe(contextId);
        });
    });

    describe("State Transitions", () => {
        it("should allow valid state transitions", async () => {
            // UNINITIALIZED -> LOADING
            await stateMachine.transitionTo(RunState.LOADING);
            expect(stateMachine.getState()).toBe(RunState.LOADING);

            // LOADING -> CONFIGURING
            await stateMachine.transitionTo(RunState.CONFIGURING);
            expect(stateMachine.getState()).toBe(RunState.CONFIGURING);

            // CONFIGURING -> READY
            await stateMachine.transitionTo(RunState.READY);
            expect(stateMachine.getState()).toBe(RunState.READY);

            // READY -> RUNNING
            await stateMachine.transitionTo(RunState.RUNNING);
            expect(stateMachine.getState()).toBe(RunState.RUNNING);

            // RUNNING -> COMPLETED
            await stateMachine.transitionTo(RunState.COMPLETED);
            expect(stateMachine.getState()).toBe(RunState.COMPLETED);
        });

        it("should reject invalid state transitions", async () => {
            // Try to go directly from UNINITIALIZED to RUNNING
            await expect(
                stateMachine.transitionTo(RunState.RUNNING)
            ).rejects.toThrow("Invalid state transition from UNINITIALIZED to RUNNING");
        });

        it("should allow execution to be paused and resumed", async () => {
            // Get to running state
            await stateMachine.transitionTo(RunState.LOADING);
            await stateMachine.transitionTo(RunState.CONFIGURING);
            await stateMachine.transitionTo(RunState.READY);
            await stateMachine.transitionTo(RunState.RUNNING);

            // Pause
            await stateMachine.transitionTo(RunState.PAUSED);
            expect(stateMachine.getState()).toBe(RunState.PAUSED);

            // Resume
            await stateMachine.transitionTo(RunState.RUNNING);
            expect(stateMachine.getState()).toBe(RunState.RUNNING);
        });

        it("should allow execution to be cancelled from paused state", async () => {
            // Get to paused state
            await stateMachine.transitionTo(RunState.LOADING);
            await stateMachine.transitionTo(RunState.CONFIGURING);
            await stateMachine.transitionTo(RunState.READY);
            await stateMachine.transitionTo(RunState.RUNNING);
            await stateMachine.transitionTo(RunState.PAUSED);

            // Cancel
            await stateMachine.transitionTo(RunState.CANCELLED);
            expect(stateMachine.getState()).toBe(RunState.CANCELLED);
        });
    });

    describe("State Persistence", () => {
        it("should persist state changes to SwarmContextManager", async () => {
            await stateMachine.transitionTo(RunState.LOADING);

            expect(mockSwarmContextManager.updateContext).toHaveBeenCalledWith(
                contextId,
                expect.objectContaining({
                    "execution.state": RunState.LOADING,
                    "execution.lastTransition": expect.any(String),
                    "execution.stateHistory": expect.any(Array),
                }),
            );
        });

        it("should handle missing SwarmContextManager gracefully", async () => {
            // Create state machine without context manager
            const stateMachineNoContext = new RoutineStateMachine(
                contextId,
                undefined,
                mockEventBus,
                mockLogger,
            );

            // Should not throw when transitioning state
            await expect(
                stateMachineNoContext.transitionTo(RunState.LOADING),
            ).resolves.not.toThrow();

            // Should log debug message
            expect(mockLogger.debug).toHaveBeenCalledWith(
                "[RoutineStateMachine] No SwarmContextManager, skipping persistence",
                expect.any(Object),
            );
        });
    });

    describe("Event Emission", () => {
        it("should emit state change events", async () => {
            await stateMachine.transitionTo(RunState.LOADING);

            expect(mockEventBus.publish).toHaveBeenCalledWith({
                type: "routine.state.changed",
                source: { tier: 2, component: "RoutineStateMachine" },
                data: {
                    contextId,
                    fromState: RunState.UNINITIALIZED,
                    toState: RunState.LOADING,
                    timestamp: expect.any(String),
                },
            });
        });

        it("should not fail if event emission fails", async () => {
            mockEventBus.publish.mockRejectedValueOnce(new Error("Event bus failed"));

            // Should not throw even if event emission fails
            await expect(
                stateMachine.transitionTo(RunState.LOADING),
            ).resolves.not.toThrow();

            // Should log error but continue
            expect(mockLogger.error).toHaveBeenCalledWith(
                "[RoutineStateMachine] Failed to emit state change event",
                expect.any(Object),
            );
        });
    });

    describe("Execution Initialization", () => {
        it("should initialize execution with proper state and context", async () => {
            const executionId = "exec-123";
            
            await stateMachine.initializeExecution(executionId);

            // Should update context with initial data
            expect(mockSwarmContextManager.updateContext).toHaveBeenCalledWith(
                contextId,
                expect.objectContaining({
                    "execution.id": executionId,
                    "execution.startTime": expect.any(String),
                    "execution.state": RunState.READY,
                }),
            );

            // Should log initialization
            expect(mockLogger.info).toHaveBeenCalledWith(
                "[RoutineStateMachine] Execution initialized",
                expect.objectContaining({
                    executionId,
                    contextId,
                }),
            );
        });
    });

    describe("Helper Methods", () => {
        it("should correctly determine if execution can proceed", async () => {
            // Non-terminal states should allow proceeding
            expect(await stateMachine.canProceed()).toBe(true);

            await stateMachine.transitionTo(RunState.LOADING);
            expect(await stateMachine.canProceed()).toBe(true);

            // Get to running state, then fail it
            await stateMachine.transitionTo(RunState.CONFIGURING);
            await stateMachine.transitionTo(RunState.READY);
            await stateMachine.transitionTo(RunState.RUNNING);
            await stateMachine.transitionTo(RunState.FAILED);
            expect(await stateMachine.canProceed()).toBe(false);
        });

        it("should correctly identify terminal states", async () => {
            // Non-terminal states
            expect(await stateMachine.isTerminal()).toBe(false);

            await stateMachine.transitionTo(RunState.LOADING);
            expect(await stateMachine.isTerminal()).toBe(false);

            // Get to running state, then fail it
            await stateMachine.transitionTo(RunState.CONFIGURING);
            await stateMachine.transitionTo(RunState.READY);
            await stateMachine.transitionTo(RunState.RUNNING);
            await stateMachine.transitionTo(RunState.FAILED);
            expect(await stateMachine.isTerminal()).toBe(true);
        });
    });

    describe("Lifecycle Methods", () => {
        it("should handle pause correctly", async () => {
            // Get to running state
            await stateMachine.transitionTo(RunState.LOADING);
            await stateMachine.transitionTo(RunState.CONFIGURING);
            await stateMachine.transitionTo(RunState.READY);
            await stateMachine.transitionTo(RunState.RUNNING);

            await stateMachine.pause();
            expect(stateMachine.getState()).toBe(RunState.PAUSED);
        });

        it("should reject pause from invalid state", async () => {
            // Cannot pause from UNINITIALIZED state
            await expect(stateMachine.pause()).rejects.toThrow("Cannot pause from state UNINITIALIZED");
        });

        it("should handle resume correctly", async () => {
            // Get to paused state
            await stateMachine.transitionTo(RunState.LOADING);
            await stateMachine.transitionTo(RunState.CONFIGURING);
            await stateMachine.transitionTo(RunState.READY);
            await stateMachine.transitionTo(RunState.RUNNING);
            await stateMachine.pause();

            await stateMachine.resume();
            expect(stateMachine.getState()).toBe(RunState.RUNNING);
        });

        it("should reject resume from invalid state", async () => {
            // Cannot resume from UNINITIALIZED state
            await expect(stateMachine.resume()).rejects.toThrow("Cannot resume from state UNINITIALIZED");
        });

        it("should handle completion correctly", async () => {
            // Get to running state
            await stateMachine.transitionTo(RunState.LOADING);
            await stateMachine.transitionTo(RunState.CONFIGURING);
            await stateMachine.transitionTo(RunState.READY);
            await stateMachine.transitionTo(RunState.RUNNING);

            await stateMachine.complete();
            expect(stateMachine.getState()).toBe(RunState.COMPLETED);

            // Should update context
            expect(mockSwarmContextManager.updateContext).toHaveBeenCalledWith(
                contextId,
                expect.objectContaining({
                    "execution.completionTime": expect.any(String),
                }),
            );
        });

        it("should reject completion from invalid state", async () => {
            // Cannot complete from UNINITIALIZED state
            await expect(stateMachine.complete()).rejects.toThrow("Cannot complete from state UNINITIALIZED");
        });

        it("should handle failure correctly", async () => {
            const error = "Test error";
            
            await stateMachine.fail(error);
            expect(stateMachine.getState()).toBe(RunState.FAILED);

            // Should update context with error
            expect(mockSwarmContextManager.updateContext).toHaveBeenCalledWith(
                contextId,
                expect.objectContaining({
                    "execution.error": error,
                    "execution.failureTime": expect.any(String),
                }),
            );
        });

        it("should handle stop correctly", async () => {
            const reason = "Test stop";
            
            await stateMachine.stop(reason);
            expect(stateMachine.getState()).toBe(RunState.CANCELLED);

            // Should update context with stop reason
            expect(mockSwarmContextManager.updateContext).toHaveBeenCalledWith(
                contextId,
                expect.objectContaining({
                    "execution.stopReason": reason,
                    "execution.stopTime": expect.any(String),
                }),
            );
        });

        it("should ignore stop when already in terminal state", async () => {
            // Get to completed state
            await stateMachine.transitionTo(RunState.LOADING);
            await stateMachine.transitionTo(RunState.CONFIGURING);
            await stateMachine.transitionTo(RunState.READY);
            await stateMachine.transitionTo(RunState.RUNNING);
            await stateMachine.complete();

            // Stop should be ignored
            await stateMachine.stop("Should be ignored");
            expect(stateMachine.getState()).toBe(RunState.COMPLETED);
        });
    });
});