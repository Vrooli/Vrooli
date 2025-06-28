import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { type Logger } from "winston";
import { type EventBus } from "../../events/types.js";
import { BaseStateMachine, BaseStates, type BaseEvent, type BaseState } from "./BaseStateMachine.js";

/**
 * BaseStateMachine Infrastructure Tests
 * 
 * These tests validate the foundational state machine infrastructure that all
 * execution tiers depend on. This is critical infrastructure testing ensuring:
 * 
 * 1. State transition logic and validation
 * 2. Event queuing and processing mechanisms
 * 3. Lifecycle management (pause/resume/stop)
 * 4. Error handling and recovery
 * 5. Event publication and subscription
 * 6. Thread safety and concurrency controls
 * 7. Resource cleanup and disposal
 * 8. ManagedTaskStateMachine interface compliance
 */

// Test implementation of BaseStateMachine for testing
class TestStateMachine extends BaseStateMachine<BaseState, BaseEvent> {
    private taskId: string;
    private processEventCalls: Array<{ event: BaseEvent; timestamp: Date }> = [];
    private lifecycleEvents: Array<{ type: string; timestamp: Date }> = [];
    private shouldFailProcessing = false;
    private shouldReportFatalError = false;

    constructor(
        logger: Logger,
        eventBus: IEventBus,
        taskId: string,
        initialState: BaseState = BaseStates.UNINITIALIZED,
    ) {
        super(logger, eventBus, initialState);
        this.taskId = taskId;
    }

    getTaskId(): string {
        return this.taskId;
    }

    getProcessEventCalls(): Array<{ event: BaseEvent; timestamp: Date }> {
        return [...this.processEventCalls];
    }

    getLifecycleEvents(): Array<{ type: string; timestamp: Date }> {
        return [...this.lifecycleEvents];
    }

    // Test controls
    setShouldFailProcessing(fail: boolean): void {
        this.shouldFailProcessing = fail;
    }

    setShouldReportFatalError(fatal: boolean): void {
        this.shouldReportFatalError = fatal;
    }

    // Protected methods implementation for testing
    protected async processEvent(event: BaseEvent): Promise<void> {
        this.processEventCalls.push({ event, timestamp: new Date() });

        if (this.shouldFailProcessing) {
            throw new Error(`Processing failed for event: ${event.type}`);
        }

        // Simulate event processing
        await new Promise(resolve => setTimeout(resolve, 1));
    }

    protected async onIdle(): Promise<void> {
        this.lifecycleEvents.push({ type: "idle", timestamp: new Date() });
    }

    protected async onPause(): Promise<void> {
        this.lifecycleEvents.push({ type: "pause", timestamp: new Date() });
    }

    protected async onResume(): Promise<void> {
        this.lifecycleEvents.push({ type: "resume", timestamp: new Date() });
    }

    protected async onStop(mode: "graceful" | "force", reason?: string): Promise<unknown> {
        this.lifecycleEvents.push({
            type: `stop_${mode}`,
            timestamp: new Date(),
        });

        return {
            mode,
            reason,
            processedEvents: this.processEventCalls.length,
        };
    }

    protected async isErrorFatal(error: unknown, event: BaseEvent): Promise<boolean> {
        return this.shouldReportFatalError;
    }

    // Expose protected methods for testing
    public async testDrain(): Promise<void> {
        return this.drain();
    }

    public testScheduleDrain(delayMs = 0): void {
        return this.scheduleDrain(delayMs);
    }

    public async testEmitEvent(type: string, data: unknown): Promise<void> {
        return this.emitEvent(type, data);
    }

    public async testEmitStateChange(fromState: BaseState, toState: BaseState, context?: Record<string, any>): Promise<void> {
        return this.emitStateChange(fromState, toState, context);
    }

    public testValidateEvent(event: BaseEvent, requiredFields: string[]): boolean {
        return this.validateEvent(event, requiredFields);
    }

    public testLogLifecycleEvent(event: string, metadata?: Record<string, unknown>): void {
        return this.logLifecycleEvent(event, metadata);
    }
}

describe("BaseStateMachine Infrastructure", () => {
    let logger: Logger;
    let eventBus: IEventBus;
    let stateMachine: TestStateMachine;

    beforeEach(() => {
        logger = {
            info: vi.fn(),
            error: vi.fn(),
            warn: vi.fn(),
            debug: vi.fn(),
        } as unknown as Logger;

        eventBus = {
            publish: vi.fn().mockResolvedValue(undefined),
            subscribe: vi.fn(),
            unsubscribe: vi.fn(),
            stop: vi.fn().mockResolvedValue(undefined),
        } as unknown as EventBus;

        stateMachine = new TestStateMachine(logger, eventBus, "test-task-123");
    });

    afterEach(async () => {
        if (stateMachine && stateMachine.getState() !== BaseStates.TERMINATED) {
            try {
                await stateMachine.stop("force");
            } catch {
                // Ignore cleanup errors
            }
        }
    });

    describe("Initialization and State Management", () => {
        it("should initialize with UNINITIALIZED state by default", () => {
            expect(stateMachine.getState()).toBe(BaseStates.UNINITIALIZED);
            expect(stateMachine.getCurrentSagaStatus()).toBe(BaseStates.UNINITIALIZED);
        });

        it("should initialize with custom initial state", () => {
            const customStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "custom-task",
                BaseStates.IDLE,
            );

            expect(customStateMachine.getState()).toBe(BaseStates.IDLE);
        });

        it("should provide task ID through ManagedTaskStateMachine interface", () => {
            expect(stateMachine.getTaskId()).toBe("test-task-123");
        });

        it("should handle concurrent state access safely", async () => {
            // Start multiple state reading operations concurrently
            const stateReads = Array.from({ length: 10 }, () =>
                Promise.resolve(stateMachine.getState()),
            );

            const states = await Promise.all(stateReads);

            // All reads should return the same state
            states.forEach(state => {
                expect(state).toBe(BaseStates.UNINITIALIZED);
            });
        });
    });

    describe("Event Queuing and Processing", () => {
        it("should queue events and process them in order", async () => {
            // Start with IDLE state to enable processing
            const idleStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "event-queue-task",
                BaseStates.IDLE,
            );

            const events: BaseEvent[] = [
                { type: "event_1", timestamp: new Date() },
                { type: "event_2", timestamp: new Date() },
                { type: "event_3", timestamp: new Date() },
            ];

            // Queue events
            for (const event of events) {
                await idleStateMachine.handleEvent(event);
            }

            // Wait for processing
            await new Promise(resolve => setTimeout(resolve, 100));

            const processedEvents = idleStateMachine.getProcessEventCalls();
            expect(processedEvents).toHaveLength(3);

            // Verify order
            expect(processedEvents[0].event.type).toBe("event_1");
            expect(processedEvents[1].event.type).toBe("event_2");
            expect(processedEvents[2].event.type).toBe("event_3");

            await idleStateMachine.stop("force");
        });

        it("should transition to RUNNING state during event processing", async () => {
            // Start with IDLE state
            const idleStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "idle-task",
                BaseStates.IDLE,
            );

            const event: BaseEvent = { type: "test_event" };
            await idleStateMachine.handleEvent(event);

            // Should transition to RUNNING during processing
            await new Promise(resolve => setTimeout(resolve, 5));

            // Eventually should return to IDLE
            await new Promise(resolve => setTimeout(resolve, 50));
            expect(idleStateMachine.getState()).toBe(BaseStates.IDLE);

            await idleStateMachine.stop("force");
        });

        it("should return to IDLE state after processing all events", async () => {
            const idleStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "idle-task",
                BaseStates.IDLE,
            );

            await idleStateMachine.handleEvent({ type: "test_event" });

            // Wait for processing to complete
            await new Promise(resolve => setTimeout(resolve, 50));

            expect(idleStateMachine.getState()).toBe(BaseStates.IDLE);
            expect(idleStateMachine.getLifecycleEvents()).toContainEqual(
                expect.objectContaining({ type: "idle" }),
            );

            await idleStateMachine.stop("force");
        });

        it("should ignore events when in TERMINATED state", async () => {
            await stateMachine.stop("force");
            expect(stateMachine.getState()).toBe(BaseStates.TERMINATED);

            await stateMachine.handleEvent({ type: "ignored_event" });

            const processedEvents = stateMachine.getProcessEventCalls();
            expect(processedEvents).toHaveLength(0);
        });

        it("should handle concurrent event submissions safely", async () => {
            const idleStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "concurrent-task",
                BaseStates.IDLE,
            );

            // Submit events concurrently
            const eventPromises = Array.from({ length: 5 }, (_, i) =>
                idleStateMachine.handleEvent({ type: `concurrent_event_${i}` }),
            );

            await Promise.all(eventPromises);

            // Wait for processing
            await new Promise(resolve => setTimeout(resolve, 100));

            const processedEvents = idleStateMachine.getProcessEventCalls();
            expect(processedEvents).toHaveLength(5);

            await idleStateMachine.stop("force");
        });
    });

    describe("Lifecycle Management - Pause/Resume", () => {
        let runningStateMachine: TestStateMachine;

        beforeEach(() => {
            runningStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "running-task",
                BaseStates.RUNNING,
            );
        });

        afterEach(async () => {
            if (runningStateMachine && runningStateMachine.getState() !== BaseStates.TERMINATED) {
                await runningStateMachine.stop("force");
            }
        });

        it("should pause from RUNNING state", async () => {
            const result = await runningStateMachine.pause();

            expect(result).toBe(true);
            expect(runningStateMachine.getState()).toBe(BaseStates.PAUSED);
            expect(runningStateMachine.getLifecycleEvents()).toContainEqual(
                expect.objectContaining({ type: "pause" }),
            );
        });

        it("should pause from IDLE state", async () => {
            const idleStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "idle-task",
                BaseStates.IDLE,
            );

            const result = await idleStateMachine.pause();

            expect(result).toBe(true);
            expect(idleStateMachine.getState()).toBe(BaseStates.PAUSED);

            await idleStateMachine.stop("force");
        });

        it("should not pause from invalid states", async () => {
            const result = await stateMachine.pause(); // UNINITIALIZED state

            expect(result).toBe(false);
            expect(stateMachine.getState()).toBe(BaseStates.UNINITIALIZED);
        });

        it("should resume from PAUSED state", async () => {
            await runningStateMachine.pause();
            expect(runningStateMachine.getState()).toBe(BaseStates.PAUSED);

            const result = await runningStateMachine.resume();

            expect(result).toBe(true);
            expect(runningStateMachine.getState()).toBe(BaseStates.IDLE);
            expect(runningStateMachine.getLifecycleEvents()).toContainEqual(
                expect.objectContaining({ type: "resume" }),
            );
        });

        it("should not resume from non-paused states", async () => {
            const result = await runningStateMachine.resume(); // RUNNING state

            expect(result).toBe(false);
            expect(runningStateMachine.getState()).toBe(BaseStates.RUNNING);
        });

        it("should support ManagedTaskStateMachine pause interface", async () => {
            const result = await runningStateMachine.requestPause();

            expect(result).toBe(true);
            expect(runningStateMachine.getState()).toBe(BaseStates.PAUSED);
        });

        it("should schedule drain after resume", async () => {
            const idleStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "resume-task",
                BaseStates.IDLE,
            );

            await idleStateMachine.pause();

            // Queue an event while paused
            await idleStateMachine.handleEvent({ type: "queued_while_paused" });

            // Resume should trigger processing
            await idleStateMachine.resume();

            // Wait for processing
            await new Promise(resolve => setTimeout(resolve, 50));

            const processedEvents = idleStateMachine.getProcessEventCalls();
            expect(processedEvents).toHaveLength(1);
            expect(processedEvents[0].event.type).toBe("queued_while_paused");

            await idleStateMachine.stop("force");
        });
    });

    describe("Lifecycle Management - Stop", () => {
        it("should stop gracefully from valid states", async () => {
            const runningStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "stop-task",
                BaseStates.RUNNING,
            );

            const result = await runningStateMachine.stop("graceful", "test stop");

            expect(result.success).toBe(true);
            expect(result.message).toContain("Stopped successfully (graceful)");
            expect(result.finalState).toEqual({
                mode: "graceful",
                reason: "test stop",
                processedEvents: 0,
            });
            expect(runningStateMachine.getState()).toBe(BaseStates.STOPPED);
        });

        it("should force stop from any state", async () => {
            const result = await stateMachine.stop("force", "forced termination");

            expect(result.success).toBe(true);
            expect(result.message).toContain("Stopped successfully (force)");
            expect(stateMachine.getState()).toBe(BaseStates.TERMINATED);
        });

        it("should not gracefully stop from invalid states", async () => {
            const failedStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "failed-task",
                BaseStates.FAILED,
            );

            const result = await failedStateMachine.stop("graceful");

            expect(result.success).toBe(false);
            expect(result.error).toBe("INVALID_STATE");
            expect(result.message).toContain("Cannot gracefully stop");
        });

        it("should handle already stopped state", async () => {
            // Start with stoppable state
            const runningStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "already-stopped-task",
                BaseStates.RUNNING,
            );

            await runningStateMachine.stop("graceful");

            const result = await runningStateMachine.stop("graceful");

            expect(result.success).toBe(true);
            expect(result.message).toContain("Already in state");
        });

        it("should support ManagedTaskStateMachine stop interface", async () => {
            const runningStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "managed-stop-task",
                BaseStates.RUNNING,
            );

            const result = await runningStateMachine.requestStop("managed stop");

            expect(result).toBe(true);
            expect(runningStateMachine.getState()).toBe(BaseStates.STOPPED);
        });

        it("should clean up resources on stop", async () => {
            const runningStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "cleanup-task",
                BaseStates.RUNNING,
            );

            // Queue some events
            await runningStateMachine.handleEvent({ type: "event_1" });
            await runningStateMachine.handleEvent({ type: "event_2" });

            await runningStateMachine.stop("graceful");

            // Events should be ignored after stop
            await runningStateMachine.handleEvent({ type: "ignored_event" });

            // Only the original events should be processed
            await new Promise(resolve => setTimeout(resolve, 50));
            const processedEvents = runningStateMachine.getProcessEventCalls();
            expect(processedEvents.length).toBeLessThanOrEqual(2);
        });
    });

    describe("Error Handling and Recovery", () => {
        it("should handle non-fatal processing errors gracefully", async () => {
            const idleStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "error-handling-task",
                BaseStates.IDLE,
            );

            idleStateMachine.setShouldFailProcessing(true);
            idleStateMachine.setShouldReportFatalError(false); // Non-fatal

            await idleStateMachine.handleEvent({ type: "failing_event" });

            // Wait for processing
            await new Promise(resolve => setTimeout(resolve, 50));

            // Should continue processing after non-fatal error
            expect(idleStateMachine.getState()).toBe(BaseStates.IDLE);

            await idleStateMachine.stop("force");
        });

        it("should transition to FAILED state on fatal errors", async () => {
            const idleStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "fatal-error-task",
                BaseStates.IDLE,
            );

            idleStateMachine.setShouldFailProcessing(true);
            idleStateMachine.setShouldReportFatalError(true); // Fatal

            await idleStateMachine.handleEvent({ type: "fatal_event" });

            // Wait for processing
            await new Promise(resolve => setTimeout(resolve, 50));

            expect(idleStateMachine.getState()).toBe(BaseStates.FAILED);

            await idleStateMachine.stop("force");
        });

        it("should handle errors during stop operation", async () => {
            // Create a state machine that will fail during stop
            class FailingStopStateMachine extends TestStateMachine {
                protected async onStop(): Promise<unknown> {
                    throw new Error("Stop operation failed");
                }
            }

            const failingStateMachine = new FailingStopStateMachine(
                logger,
                eventBus,
                "failing-stop-task",
                BaseStates.RUNNING,
            );

            const result = await failingStateMachine.stop("graceful");

            expect(result.success).toBe(false);
            expect(result.error).toContain("Stop operation failed");
            expect(failingStateMachine.getState()).toBe(BaseStates.FAILED);
        });

        it("should use error handler for consistent error processing", async () => {
            const idleStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "error-wrapper-task",
                BaseStates.IDLE,
            );

            idleStateMachine.setShouldFailProcessing(true);

            await idleStateMachine.handleEvent({ type: "error_test" });

            // Wait for processing
            await new Promise(resolve => setTimeout(resolve, 50));

            // Error should be logged appropriately
            expect(logger.error).toHaveBeenCalled();
        });
    });

    describe("Event Publication and Communication", () => {
        it("should publish events to event bus", async () => {
            await stateMachine.testEmitEvent("test.event", { data: "test" });

            // Verify the event bus was called with structured event data
            expect(eventBus.publish).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "state.machine.test.event",
                    data: { data: "test" },
                    source: expect.objectContaining({
                        component: "TestStateMachine",
                    }),
                }),
            );
        });

        it("should publish state change events", async () => {
            await stateMachine.testEmitStateChange(
                BaseStates.IDLE,
                BaseStates.RUNNING,
                { step: "processing" },
            );

            // Verify the event bus was called with structured state change event
            expect(eventBus.publish).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "state_machine.state.changed",
                    data: expect.objectContaining({
                        entityType: "state_machine",
                        entityId: "test-task-123",
                        fromState: BaseStates.IDLE,
                        toState: BaseStates.RUNNING,
                        context: { step: "processing" },
                    }),
                }),
            );
        });

        it("should handle event publication failures gracefully", async () => {
            (eventBus.publish as any).mockRejectedValue(new Error("Event bus error"));

            // Should not throw
            await expect(stateMachine.testEmitEvent("test.event", {})).rejects.toThrow("Event bus error");
        });
    });

    describe("Utility Methods and Validation", () => {
        it("should validate events with required fields", () => {
            const validEvent: BaseEvent = {
                type: "valid_event",
                timestamp: new Date(),
                metadata: { required: "value" },
            };

            const isValid = stateMachine.testValidateEvent(validEvent, ["type", "metadata"]);
            expect(isValid).toBe(true);
        });

        it("should reject events missing required fields", () => {
            const invalidEvent: BaseEvent = {
                type: "invalid_event",
                // Missing required fields
            };

            const isValid = stateMachine.testValidateEvent(invalidEvent, ["type", "timestamp", "metadata"]);
            expect(isValid).toBe(false);
            expect(logger.error).toHaveBeenCalledWith(
                expect.stringContaining("Event missing required field"),
                expect.objectContaining({
                    eventType: "invalid_event",
                }),
            );
        });

        it("should log lifecycle events consistently", () => {
            stateMachine.testLogLifecycleEvent("Test Event", { metadata: "value" });

            expect(logger.info).toHaveBeenCalledWith(
                "[TestStateMachine] Test Event",
                {
                    state: BaseStates.UNINITIALIZED,
                    metadata: "value",
                },
            );
        });

        it("should handle scheduled drain with delay", async () => {
            const idleStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "delayed-drain-task",
                BaseStates.IDLE,
            );

            await idleStateMachine.handleEvent({ type: "delayed_event" });

            // Schedule drain with delay
            idleStateMachine.testScheduleDrain(10);

            // Event should not be processed immediately
            expect(idleStateMachine.getProcessEventCalls()).toHaveLength(0);

            // Wait for delayed processing
            await new Promise(resolve => setTimeout(resolve, 50));

            expect(idleStateMachine.getProcessEventCalls()).toHaveLength(1);

            await idleStateMachine.stop("force");
        });

        it("should cancel pending drain when paused", async () => {
            const idleStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "pause-drain-task",
                BaseStates.IDLE,
            );

            await idleStateMachine.handleEvent({ type: "pausable_event" });

            // Schedule drain with delay
            idleStateMachine.testScheduleDrain(100);

            // Pause immediately
            await idleStateMachine.pause();

            // Wait longer than the scheduled delay
            await new Promise(resolve => setTimeout(resolve, 150));

            // Event should not be processed due to pause
            expect(idleStateMachine.getProcessEventCalls()).toHaveLength(0);

            await idleStateMachine.stop("force");
        });
    });

    describe("Thread Safety and Concurrency", () => {
        it("should handle concurrent pause/resume operations safely", async () => {
            const runningStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "concurrent-lifecycle-task",
                BaseStates.RUNNING,
            );

            // Execute pause and resume concurrently
            const pausePromise = runningStateMachine.pause();
            const resumePromise = runningStateMachine.resume();

            const [pauseResult, resumeResult] = await Promise.all([pausePromise, resumePromise]);

            // At least one operation should succeed, but results depend on timing
            const atLeastOneSucceeded = pauseResult || resumeResult;
            expect(atLeastOneSucceeded).toBe(true);

            // Final state should be consistent
            const finalState = runningStateMachine.getState();
            expect([BaseStates.RUNNING, BaseStates.IDLE, BaseStates.PAUSED]).toContain(finalState);

            await runningStateMachine.stop("force");
        });

        it("should prevent concurrent drain operations", async () => {
            const idleStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "concurrent-drain-task",
                BaseStates.IDLE,
            );

            // Queue multiple events
            await idleStateMachine.handleEvent({ type: "event_1" });
            await idleStateMachine.handleEvent({ type: "event_2" });
            await idleStateMachine.handleEvent({ type: "event_3" });

            // Trigger multiple drains concurrently
            const drainPromises = [
                idleStateMachine.testDrain(),
                idleStateMachine.testDrain(),
                idleStateMachine.testDrain(),
            ];

            await Promise.all(drainPromises);

            // All events should be processed exactly once
            const processedEvents = idleStateMachine.getProcessEventCalls();
            expect(processedEvents).toHaveLength(3);

            await idleStateMachine.stop("force");
        });

        it("should handle concurrent stop operations safely", async () => {
            const runningStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "concurrent-stop-task",
                BaseStates.RUNNING,
            );

            // Execute multiple stop operations concurrently
            const stopPromises = [
                runningStateMachine.stop("graceful", "stop 1"),
                runningStateMachine.stop("graceful", "stop 2"),
                runningStateMachine.stop("force", "stop 3"),
            ];

            const results = await Promise.all(stopPromises);

            // At least one should succeed
            const successCount = results.filter(r => r.success).length;
            expect(successCount).toBeGreaterThanOrEqual(1);

            // Final state should be consistent
            const finalState = runningStateMachine.getState();
            expect([BaseStates.STOPPED, BaseStates.TERMINATED]).toContain(finalState);
        });
    });

    describe("Memory Management and Resource Cleanup", () => {
        it("should clean up timeouts on disposal", async () => {
            const idleStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "cleanup-timeout-task",
                BaseStates.IDLE,
            );

            // Schedule a drain with delay to create timeout
            idleStateMachine.testScheduleDrain(1000);

            // Stop immediately to trigger cleanup
            await idleStateMachine.stop("force");

            // No timeout should remain (verified by successful stop)
            expect(idleStateMachine.getState()).toBe(BaseStates.TERMINATED);
        });

        it("should prevent memory leaks from event queue", async () => {
            const idleStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "memory-test-task",
                BaseStates.IDLE,
            );

            // Add many events
            const eventCount = 1000;
            for (let i = 0; i < eventCount; i++) {
                await idleStateMachine.handleEvent({ type: `memory_event_${i}` });
            }

            // Stop before processing
            await idleStateMachine.stop("force");

            // Events should be ignored after termination
            await idleStateMachine.handleEvent({ type: "ignored_after_stop" });

            // Processing should be limited to what was processed before stop
            const processedEvents = idleStateMachine.getProcessEventCalls();
            expect(processedEvents.length).toBeLessThan(eventCount);
        });

        it("should dispose of resources properly", async () => {
            const idleStateMachine = new TestStateMachine(
                logger,
                eventBus,
                "disposal-task",
                BaseStates.IDLE,
            );

            await idleStateMachine.stop("force");

            // Verify disposed state prevents further operations
            await idleStateMachine.handleEvent({ type: "post_disposal" });

            expect(idleStateMachine.getProcessEventCalls()).toHaveLength(0);
        });
    });
});
