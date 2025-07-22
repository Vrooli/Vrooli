import type { StopOptions } from "@vrooli/shared";
import { EventTypes, RunState, type SessionUser } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, test, vi, type MockedFunction } from "vitest";
import type { ServiceEvent } from "../../events/types.js";
import { BaseStateMachine, type StateMachineCoordinationConfig } from "./BaseStateMachine.js";
import { ErrorHandler } from "./ErrorHandler.js";

// Mock dependencies
vi.mock("../../../events/logger.js", async () => {
    const { createMockLogger } = await import("../../../__test/mocks/logger.js");
    return {
        logger: createMockLogger(),
    };
});

vi.mock("../../events/eventBus.js", () => ({
    getEventBus: vi.fn(() => ({
        subscribe: vi.fn().mockResolvedValue("subscription-id"),
        unsubscribe: vi.fn().mockResolvedValue(undefined),
        publish: vi.fn().mockResolvedValue({ success: true }),
        start: vi.fn().mockResolvedValue(undefined),
        stop: vi.fn().mockResolvedValue(undefined),
    })),
}));

vi.mock("../../events/publisher.js", () => ({
    EventPublisher: {
        emit: vi.fn().mockResolvedValue({ proceed: true, reason: null }),
    },
}));

vi.mock("@vrooli/shared", async () => {
    const actual = await vi.importActual("@vrooli/shared");
    return {
        ...actual,
        generatePK: vi.fn(() => "test-pk-123"),
    };
});

vi.mock("./ErrorHandler.js", () => ({
    ErrorHandler: vi.fn().mockImplementation(() => ({
        wrap: vi.fn().mockImplementation(async (operation) => {
            try {
                const result = await operation();
                return { success: true, data: result };
            } catch (error) {
                return { success: false, error };
            }
        }),
    })),
}));

// Test implementation of BaseStateMachine
class TestStateMachine extends BaseStateMachine<ServiceEvent> {
    public lastProcessedEvent: ServiceEvent | null = null;
    public idleCallCount = 0;
    public pauseCallCount = 0;
    public resumeCallCount = 0;
    public stopCallCount = 0;
    public lastStopMode: "graceful" | "force" | null = null;
    public lastStopReason: string | undefined;
    public shouldHandleEventResult = true;
    public fatalErrorResult = false;
    public eventPatterns: Array<{ pattern: string }> = [
        { pattern: "test/event" },
        { pattern: "test/other" },
    ];

    constructor(
        initialState = RunState.UNINITIALIZED,
        componentName = "TestStateMachine",
        coordinationConfig: StateMachineCoordinationConfig = {},
    ) {
        super(initialState, componentName, coordinationConfig);
    }

    public getTaskId(): string {
        return "test-task-123";
    }

    protected async processEvent(event: ServiceEvent): Promise<void> {
        this.lastProcessedEvent = event;
        // Simulate some processing time
        await new Promise(resolve => setTimeout(resolve, 1));
    }

    protected async onIdle(): Promise<void> {
        this.idleCallCount++;
    }

    protected async onPause(): Promise<void> {
        this.pauseCallCount++;
    }

    protected async onResume(): Promise<void> {
        this.resumeCallCount++;
    }

    protected async onStop(mode: "graceful" | "force", reason?: string): Promise<unknown> {
        this.stopCallCount++;
        this.lastStopMode = mode;
        this.lastStopReason = reason;
        return { stopped: true, mode, reason };
    }

    protected async isErrorFatal(_error: unknown, _event: ServiceEvent): Promise<boolean> {
        return this.fatalErrorResult;
    }

    protected getEventPatterns(): Array<{ pattern: string }> {
        return this.eventPatterns;
    }

    protected shouldHandleEvent(_event: ServiceEvent): boolean {
        return this.shouldHandleEventResult;
    }

    // Expose protected methods for testing
    public testApplyStateTransition(toState: RunState, reason: string): Promise<void> {
        return this.applyStateTransition(toState, reason);
    }

    public testIsValidTransition(from: RunState, to: RunState): boolean {
        return this.isValidTransition(from, to);
    }

    public testScheduleDrain(delayMs = 0): void {
        return this.scheduleDrain(delayMs);
    }

    public testClearPendingDrainTimeout(): void {
        return this.clearPendingDrainTimeout();
    }

    public getEventQueueLength(): number {
        return (this as any).eventQueue.length;
    }

    public getEventQueue(): ServiceEvent[] {
        return [...(this as any).eventQueue];
    }

    public isDisposed(): boolean {
        return (this as any).disposed;
    }
}

describe("BaseStateMachine", () => {
    let stateMachine: TestStateMachine;
    let mockErrorHandler: { wrap: MockedFunction<any> };

    beforeEach(() => {
        vi.clearAllMocks();
        stateMachine = new TestStateMachine();
        mockErrorHandler = (stateMachine as any).errorHandler;
    });

    afterEach(async () => {
        try {
            if (stateMachine && !stateMachine.isDisposed()) {
                await stateMachine.stop();
            }
        } catch (error) {
            // Ignore cleanup errors in tests
        }

        // Clear all mocks to prevent state leakage between tests
        vi.clearAllMocks();

        // Clear all timers to prevent timeout issues
        vi.clearAllTimers();
    });

    describe("Constructor and Initialization", () => {
        test("should initialize with default values", () => {
            expect(stateMachine.getState()).toBe(RunState.UNINITIALIZED);
            expect(stateMachine.getTaskId()).toBe("test-task-123");
            expect(stateMachine.getEventQueueLength()).toBe(0);
            expect(stateMachine.isDisposed()).toBe(false);
        });

        test("should initialize with custom state and config", () => {
            const config: StateMachineCoordinationConfig = {
                chatId: "chat-123",
                swarmId: "swarm-456",
                contextId: "context-789",
            };
            const customMachine = new TestStateMachine(RunState.READY, "CustomMachine", config);

            expect(customMachine.getState()).toBe(RunState.READY);
            expect(customMachine.getCoordinationStatus().chatId).toBe("chat-123");
        });

        test("should create ErrorHandler with correct config", () => {
            expect(ErrorHandler).toHaveBeenCalledWith({
                component: "TestStateMachine",
                chatId: undefined,
            });
        });
    });

    describe("State Transitions", () => {
        test("should validate correct state transitions", () => {
            // Valid transitions from UNINITIALIZED
            expect(stateMachine.testIsValidTransition(RunState.UNINITIALIZED, RunState.LOADING)).toBe(true);
            expect(stateMachine.testIsValidTransition(RunState.UNINITIALIZED, RunState.FAILED)).toBe(true);
            expect(stateMachine.testIsValidTransition(RunState.UNINITIALIZED, RunState.CANCELLED)).toBe(true);

            // Valid transitions from READY
            expect(stateMachine.testIsValidTransition(RunState.READY, RunState.RUNNING)).toBe(true);
            expect(stateMachine.testIsValidTransition(RunState.READY, RunState.PAUSED)).toBe(true);
            expect(stateMachine.testIsValidTransition(RunState.READY, RunState.COMPLETED)).toBe(true);

            // Invalid transitions
            expect(stateMachine.testIsValidTransition(RunState.UNINITIALIZED, RunState.RUNNING)).toBe(false);
            expect(stateMachine.testIsValidTransition(RunState.COMPLETED, RunState.RUNNING)).toBe(false);
            expect(stateMachine.testIsValidTransition(RunState.FAILED, RunState.READY)).toBe(false);
        });

        test("should apply valid state transitions", async () => {
            await stateMachine.testApplyStateTransition(RunState.LOADING, "test_transition");

            expect(stateMachine.getState()).toBe(RunState.LOADING);
        });

        test("should reject invalid state transitions", async () => {
            const initialState = stateMachine.getState();

            await stateMachine.testApplyStateTransition(RunState.RUNNING, "invalid_transition");

            // State should remain unchanged
            expect(stateMachine.getState()).toBe(initialState);
        });

        test("should emit state change events on valid transitions", async () => {
            const { EventPublisher } = await import("../../events/publisher.js");

            await stateMachine.testApplyStateTransition(RunState.LOADING, "test_emit");

            expect(EventPublisher.emit).toHaveBeenCalledWith(
                EventTypes.SYSTEM.STATE_CHANGED,
                expect.objectContaining({
                    componentName: "TestStateMachine",
                    previousState: RunState.UNINITIALIZED,
                    newState: RunState.LOADING,
                    context: { reason: "test_emit" },
                }),
            );
        });
    });

    describe("Event Queue Management", () => {
        function createTestEvent(type = "test/event", id = "event-1"): ServiceEvent {
            return {
                id,
                type,
                timestamp: new Date(),
                tier: "tier2",
                source: "test",
                chatId: "test-chat",
            };
        }

        test("should queue events when not in terminated state", async () => {
            const event = createTestEvent();

            await stateMachine.handleEvent(event);

            expect(stateMachine.getEventQueueLength()).toBe(1);
            expect(stateMachine.getEventQueue()[0]).toBe(event);
        });

        test("should ignore events when disposed", async () => {
            // Move to a state where stop can work
            await stateMachine.testApplyStateTransition(RunState.READY, "setup");
            await stateMachine.stop();
            const event = createTestEvent();

            await stateMachine.handleEvent(event);

            expect(stateMachine.getEventQueueLength()).toBe(0);
        });

        test("should ignore events when cancelled", async () => {
            await stateMachine.testApplyStateTransition(RunState.CANCELLED, "test_cancel");
            const event = createTestEvent();

            await stateMachine.handleEvent(event);

            expect(stateMachine.getEventQueueLength()).toBe(0);
        });

        test("should filter events based on shouldHandleEvent", async () => {
            stateMachine.shouldHandleEventResult = false;
            const event = createTestEvent();

            await stateMachine.handleEvent(event);

            expect(stateMachine.getEventQueueLength()).toBe(0);
        });

        test("should enforce queue capacity limits", async () => {
            // Fill queue to capacity (1000 events)
            const events: ServiceEvent[] = [];
            for (let i = 0; i < 1000; i++) {
                events.push(createTestEvent("test/event", `event-${i}`));
            }

            // Add events up to capacity
            for (const event of events) {
                await stateMachine.handleEvent(event);
            }
            expect(stateMachine.getEventQueueLength()).toBe(1000);

            // Add one more event - should trigger eviction
            const overflowEvent = createTestEvent("test/overflow", "overflow-event");
            await stateMachine.handleEvent(overflowEvent);

            // Should still be at capacity, but oldest 10% (100 events) should be evicted
            expect(stateMachine.getEventQueueLength()).toBe(901); // 1000 - 100 + 1

            // Verify the overflow event was added
            const queue = stateMachine.getEventQueue();
            expect(queue[queue.length - 1]).toBe(overflowEvent);

            // Verify oldest events were removed (first 100 should be gone)
            expect(queue.find(e => e.id === "event-0")).toBeUndefined();
            expect(queue.find(e => e.id === "event-99")).toBeUndefined();
            expect(queue.find(e => e.id === "event-100")).toBeDefined();
        });

        test("should schedule drain when in READY state", async () => {
            await stateMachine.testApplyStateTransition(RunState.READY, "test_ready");
            const drainSpy = vi.spyOn(stateMachine as any, "scheduleDrain");

            const event = createTestEvent();
            await stateMachine.handleEvent(event);

            expect(drainSpy).toHaveBeenCalled();
        });

        test("should not schedule drain when not in READY state", async () => {
            // Machine starts in UNINITIALIZED state
            const drainSpy = vi.spyOn(stateMachine as any, "scheduleDrain");

            const event = createTestEvent();
            await stateMachine.handleEvent(event);

            expect(drainSpy).not.toHaveBeenCalled();
        });
    });

    describe("Lifecycle Control - Pause and Resume", () => {
        test("should pause from RUNNING state", async () => {
            // Valid path: UNINITIALIZED -> LOADING -> CONFIGURING -> READY -> RUNNING
            await stateMachine.testApplyStateTransition(RunState.LOADING, "setup1");
            await stateMachine.testApplyStateTransition(RunState.CONFIGURING, "setup2");
            await stateMachine.testApplyStateTransition(RunState.READY, "setup3");
            await stateMachine.testApplyStateTransition(RunState.RUNNING, "setup4");

            const result = await stateMachine.pause();

            expect(result).toBe(true);
            expect(stateMachine.getState()).toBe(RunState.PAUSED);
            expect(stateMachine.pauseCallCount).toBe(1);
        });

        test("should pause from READY state", async () => {
            // Valid path: UNINITIALIZED -> LOADING -> CONFIGURING -> READY
            await stateMachine.testApplyStateTransition(RunState.LOADING, "setup1");
            await stateMachine.testApplyStateTransition(RunState.CONFIGURING, "setup2");
            await stateMachine.testApplyStateTransition(RunState.READY, "setup3");

            const result = await stateMachine.pause();

            expect(result).toBe(true);
            expect(stateMachine.getState()).toBe(RunState.PAUSED);
        });

        test("should not pause from invalid states", async () => {
            // Try to pause from UNINITIALIZED
            const result = await stateMachine.pause();

            expect(result).toBe(false);
            expect(stateMachine.getState()).toBe(RunState.UNINITIALIZED);
            expect(stateMachine.pauseCallCount).toBe(0);
        });

        test("should resume from PAUSED state", async () => {
            // Valid path to PAUSED: UNINITIALIZED -> LOADING -> CONFIGURING -> READY -> PAUSED
            await stateMachine.testApplyStateTransition(RunState.LOADING, "setup1");
            await stateMachine.testApplyStateTransition(RunState.CONFIGURING, "setup2");
            await stateMachine.testApplyStateTransition(RunState.READY, "setup3");
            await stateMachine.testApplyStateTransition(RunState.PAUSED, "setup4");

            const result = await stateMachine.resume();

            expect(result).toBe(true);
            expect(stateMachine.getState()).toBe(RunState.READY);
            expect(stateMachine.resumeCallCount).toBe(1);
        });

        test("should resume from SUSPENDED state", async () => {
            // SUSPENDED can only be reached by manual state setting (not through normal transitions)
            // This simulates an external system putting the machine in SUSPENDED state
            (stateMachine as any).state = RunState.SUSPENDED;

            const result = await stateMachine.resume();

            expect(result).toBe(true);
            expect(stateMachine.getState()).toBe(RunState.READY);
        });

        test("should not resume from invalid states", async () => {
            // Try to resume from RUNNING (valid path first)
            await stateMachine.testApplyStateTransition(RunState.LOADING, "setup1");
            await stateMachine.testApplyStateTransition(RunState.CONFIGURING, "setup2");
            await stateMachine.testApplyStateTransition(RunState.READY, "setup3");
            await stateMachine.testApplyStateTransition(RunState.RUNNING, "setup4");

            const result = await stateMachine.resume();

            expect(result).toBe(false);
            expect(stateMachine.getState()).toBe(RunState.RUNNING);
            expect(stateMachine.resumeCallCount).toBe(0);
        });

        test("should schedule drain after resume", async () => {
            // Valid path to PAUSED
            await stateMachine.testApplyStateTransition(RunState.LOADING, "setup1");
            await stateMachine.testApplyStateTransition(RunState.CONFIGURING, "setup2");
            await stateMachine.testApplyStateTransition(RunState.READY, "setup3");
            await stateMachine.testApplyStateTransition(RunState.PAUSED, "setup4");
            const drainSpy = vi.spyOn(stateMachine as any, "scheduleDrain");

            await stateMachine.resume();

            expect(drainSpy).toHaveBeenCalled();
        });
    });

    describe("Lifecycle Control - Stop Operations", () => {
        test("should stop with no parameters (graceful default)", async () => {
            // Valid path to RUNNING
            await stateMachine.testApplyStateTransition(RunState.LOADING, "setup1");
            await stateMachine.testApplyStateTransition(RunState.CONFIGURING, "setup2");
            await stateMachine.testApplyStateTransition(RunState.READY, "setup3");
            await stateMachine.testApplyStateTransition(RunState.RUNNING, "setup4");

            const result = await stateMachine.stop();

            expect(result.success).toBe(true);
            expect(result.message).toContain("Stopped successfully (graceful)");
            expect(stateMachine.getState()).toBe(RunState.COMPLETED);
            expect(stateMachine.stopCallCount).toBe(1);
            expect(stateMachine.lastStopMode).toBe("graceful");
            expect(stateMachine.isDisposed()).toBe(true);
        });

        test("should stop with reason string", async () => {
            // Valid path to RUNNING
            await stateMachine.testApplyStateTransition(RunState.LOADING, "setup1");
            await stateMachine.testApplyStateTransition(RunState.CONFIGURING, "setup2");
            await stateMachine.testApplyStateTransition(RunState.READY, "setup3");
            await stateMachine.testApplyStateTransition(RunState.RUNNING, "setup4");

            const result = await stateMachine.stop({ reason: "user requested" });

            expect(result.success).toBe(true);
            expect(stateMachine.lastStopReason).toBe("user requested");
            expect(stateMachine.lastStopMode).toBe("graceful");
        });

        test("should stop with mode and reason", async () => {
            // Valid path to RUNNING
            await stateMachine.testApplyStateTransition(RunState.LOADING, "setup1");
            await stateMachine.testApplyStateTransition(RunState.CONFIGURING, "setup2");
            await stateMachine.testApplyStateTransition(RunState.READY, "setup3");
            await stateMachine.testApplyStateTransition(RunState.RUNNING, "setup4");

            const result = await stateMachine.stop({ mode: "force", reason: "emergency stop" });

            expect(result.success).toBe(true);
            expect(stateMachine.getState()).toBe(RunState.CANCELLED);
            expect(stateMachine.lastStopMode).toBe("force");
            expect(stateMachine.lastStopReason).toBe("emergency stop");
        });

        test("should stop with options object", async () => {
            // Valid path to RUNNING
            await stateMachine.testApplyStateTransition(RunState.LOADING, "setup1");
            await stateMachine.testApplyStateTransition(RunState.CONFIGURING, "setup2");
            await stateMachine.testApplyStateTransition(RunState.READY, "setup3");
            await stateMachine.testApplyStateTransition(RunState.RUNNING, "setup4");
            const user: SessionUser = { id: "user-123" } as SessionUser;
            const options: StopOptions = {
                mode: "force",
                reason: "system shutdown",
                requestingUser: user,
            };

            const result = await stateMachine.stop(options);

            expect(result.success).toBe(true);
            expect(stateMachine.lastStopMode).toBe("force");
            expect(stateMachine.lastStopReason).toBe("system shutdown");
        });

        test("should return success if already completed", async () => {
            await stateMachine.testApplyStateTransition(RunState.COMPLETED, "setup");

            const result = await stateMachine.stop();

            expect(result.success).toBe(true);
            expect(result.message).toContain("Already in state COMPLETED");
        });

        test("should return success if already cancelled", async () => {
            await stateMachine.testApplyStateTransition(RunState.CANCELLED, "setup");

            const result = await stateMachine.stop();

            expect(result.success).toBe(true);
            expect(result.message).toContain("Already in state CANCELLED");
        });

        test("should reject graceful stop from invalid states", async () => {
            // Try to gracefully stop from UNINITIALIZED
            const result = await stateMachine.stop({ mode: "graceful" });

            expect(result.success).toBe(false);
            expect(result.message).toContain("Cannot gracefully stop from state UNINITIALIZED");
            expect(result.error).toBe("INVALID_STATE");
        });

        test("should force stop from any state", async () => {
            // Force stop should work from any state, including UNINITIALIZED
            const result = await stateMachine.stop({ mode: "force", reason: "emergency" });

            expect(result.success).toBe(true);
            expect(stateMachine.getState()).toBe(RunState.CANCELLED);
        });

        test("should handle errors during stop gracefully", async () => {
            // Valid path to RUNNING
            await stateMachine.testApplyStateTransition(RunState.LOADING, "setup1");
            await stateMachine.testApplyStateTransition(RunState.CONFIGURING, "setup2");
            await stateMachine.testApplyStateTransition(RunState.READY, "setup3");
            await stateMachine.testApplyStateTransition(RunState.RUNNING, "setup4");

            // Mock error handler to return failure
            mockErrorHandler.wrap.mockResolvedValueOnce({
                success: false,
                error: new Error("Stop failed"),
            });

            const result = await stateMachine.stop();

            expect(result.success).toBe(false);
            expect(result.message).toBe("Error during stop");
            expect(result.error).toBe("Stop failed");
            expect(stateMachine.getState()).toBe(RunState.FAILED);
        });
    });

    describe("ManagedTaskStateMachine Interface", () => {
        test("should implement requestPause", async () => {
            // Valid path to RUNNING
            await stateMachine.testApplyStateTransition(RunState.LOADING, "setup1");
            await stateMachine.testApplyStateTransition(RunState.CONFIGURING, "setup2");
            await stateMachine.testApplyStateTransition(RunState.READY, "setup3");
            await stateMachine.testApplyStateTransition(RunState.RUNNING, "setup4");

            const result = await stateMachine.requestPause();

            expect(result).toBe(true);
            expect(stateMachine.getState()).toBe(RunState.PAUSED);
        });

        test("should implement requestStop", async () => {
            // Valid path to RUNNING
            await stateMachine.testApplyStateTransition(RunState.LOADING, "setup1");
            await stateMachine.testApplyStateTransition(RunState.CONFIGURING, "setup2");
            await stateMachine.testApplyStateTransition(RunState.READY, "setup3");
            await stateMachine.testApplyStateTransition(RunState.RUNNING, "setup4");

            const result = await stateMachine.requestStop("test reason");

            expect(result).toBe(true);
            expect(stateMachine.getState()).toBe(RunState.COMPLETED);
            expect(stateMachine.lastStopReason).toBe("test reason");
        });
    });

    describe("Error Handling Integration", () => {
        test("should wrap operations with ErrorHandler", async () => {
            const event = createTestEvent();
            await stateMachine.testApplyStateTransition(RunState.READY, "setup");
            await stateMachine.handleEvent(event);

            // Trigger drain to process the event
            await stateMachine.testScheduleDrain();

            // Wait for async processing
            await new Promise(resolve => setTimeout(resolve, 10));

            expect(mockErrorHandler.wrap).toHaveBeenCalled();
        });

        test("should handle non-fatal errors gracefully", async () => {
            stateMachine.fatalErrorResult = false;
            // Valid path to READY
            await stateMachine.testApplyStateTransition(RunState.LOADING, "setup1");
            await stateMachine.testApplyStateTransition(RunState.CONFIGURING, "setup2");
            await stateMachine.testApplyStateTransition(RunState.READY, "setup3");

            // Mock error handler to return failure
            mockErrorHandler.wrap.mockResolvedValueOnce({
                success: false,
                error: new Error("Non-fatal error"),
            });

            const event = createTestEvent();
            await stateMachine.handleEvent(event);

            // Trigger drain to process the event
            await (stateMachine as any).drain();
            await new Promise(resolve => setTimeout(resolve, 20));

            // Should remain in READY state for non-fatal errors
            expect(stateMachine.getState()).toBe(RunState.READY);
        });

        test("should transition to FAILED on fatal errors", async () => {
            stateMachine.fatalErrorResult = true;
            // Valid path to READY
            await stateMachine.testApplyStateTransition(RunState.LOADING, "setup1");
            await stateMachine.testApplyStateTransition(RunState.CONFIGURING, "setup2");
            await stateMachine.testApplyStateTransition(RunState.READY, "setup3");

            // Mock error handler to return failure
            mockErrorHandler.wrap.mockResolvedValueOnce({
                success: false,
                error: new Error("Fatal error"),
            });

            const event = createTestEvent();
            await stateMachine.handleEvent(event);

            // Trigger drain to process the event
            await (stateMachine as any).drain();
            await new Promise(resolve => setTimeout(resolve, 20));

            expect(stateMachine.getState()).toBe(RunState.FAILED);
        });
    });

    describe("Event Subscription and Coordination", () => {
        let mockEventBus: any;

        beforeEach(async () => {
            const { getEventBus } = await import("../../events/eventBus.js");
            mockEventBus = getEventBus();
        });

        test("should setup event subscriptions for patterns", async () => {
            await (stateMachine as any).setupEventSubscriptions();

            expect(mockEventBus.subscribe).toHaveBeenCalledTimes(2);
            expect(mockEventBus.subscribe).toHaveBeenCalledWith(
                "test/event",
                expect.any(Function),
            );
            expect(mockEventBus.subscribe).toHaveBeenCalledWith(
                "test/other",
                expect.any(Function),
            );
        });

        test("should cleanup event subscriptions", async () => {
            await (stateMachine as any).setupEventSubscriptions();
            await (stateMachine as any).cleanupEventSubscriptions();

            expect(mockEventBus.unsubscribe).toHaveBeenCalledTimes(2);
            expect(mockEventBus.unsubscribe).toHaveBeenCalledWith("subscription-id");
        });

        test("should handle subscription cleanup errors gracefully", async () => {
            await (stateMachine as any).setupEventSubscriptions();

            // Mock unsubscribe to throw error
            mockEventBus.unsubscribe.mockRejectedValueOnce(new Error("Unsubscribe failed"));

            // Should not throw
            await expect((stateMachine as any).cleanupEventSubscriptions()).resolves.not.toThrow();
        });

        test("should route subscribed events to handleEvent", async () => {
            await (stateMachine as any).setupEventSubscriptions();

            // Get the subscription handler that was registered
            const subscriptionHandler = mockEventBus.subscribe.mock.calls[0][1];
            const handleEventSpy = vi.spyOn(stateMachine, "handleEvent");

            const testEvent = createTestEvent();
            await subscriptionHandler(testEvent);

            expect(handleEventSpy).toHaveBeenCalledWith(testEvent);
        });
    });

    describe("Event Drain Processing", () => {
        const createTestEvent = (type = "test/event", id = "event-1"): ServiceEvent => ({
            id,
            type,
            timestamp: new Date(),
            tier: "tier2",
            source: "test",
            chatId: "test-chat",
        });

        test("should not drain when not in drainable state", async () => {
            const event = createTestEvent();
            await stateMachine.handleEvent(event);

            // Machine starts in UNINITIALIZED, which is not drainable
            await (stateMachine as any).drain();

            // Event should still be in queue
            expect(stateMachine.getEventQueueLength()).toBe(1);
            expect(stateMachine.lastProcessedEvent).toBeNull();
        });

        test("should process events in FIFO order", async () => {
            // Valid path to READY
            await stateMachine.testApplyStateTransition(RunState.LOADING, "setup1");
            await stateMachine.testApplyStateTransition(RunState.CONFIGURING, "setup2");
            await stateMachine.testApplyStateTransition(RunState.READY, "setup3");

            const event1 = createTestEvent("test/event", "event-1");
            const event2 = createTestEvent("test/event", "event-2");
            const event3 = createTestEvent("test/event", "event-3");

            await stateMachine.handleEvent(event1);
            await stateMachine.handleEvent(event2);
            await stateMachine.handleEvent(event3);

            expect(stateMachine.getEventQueueLength()).toBe(3);

            // Process events one by one
            await (stateMachine as any).drain();
            await new Promise(resolve => setTimeout(resolve, 20));

            // All events should be processed and queue should be empty
            expect(stateMachine.getEventQueueLength()).toBe(0);
            expect(stateMachine.lastProcessedEvent).toBe(event3); // Last processed
            expect(stateMachine.getState()).toBe(RunState.READY); // Should return to READY
            expect(stateMachine.idleCallCount).toBe(1); // onIdle should be called
        });

        test("should transition to RUNNING during drain", async () => {
            // Valid path to READY
            await stateMachine.testApplyStateTransition(RunState.LOADING, "setup1");
            await stateMachine.testApplyStateTransition(RunState.CONFIGURING, "setup2");
            await stateMachine.testApplyStateTransition(RunState.READY, "setup3");
            const event = createTestEvent();
            await stateMachine.handleEvent(event);

            const drainPromise = (stateMachine as any).drain();

            // Check state immediately after drain starts
            await new Promise(resolve => setTimeout(resolve, 1));
            expect(stateMachine.getState()).toBe(RunState.RUNNING);

            await drainPromise;
        });

        test("should schedule additional drain if events arrive during processing", async () => {
            // Valid path to READY
            await stateMachine.testApplyStateTransition(RunState.LOADING, "setup1");
            await stateMachine.testApplyStateTransition(RunState.CONFIGURING, "setup2");
            await stateMachine.testApplyStateTransition(RunState.READY, "setup3");

            const event1 = createTestEvent("test/event", "event-1");
            await stateMachine.handleEvent(event1);

            const scheduleDrainSpy = vi.spyOn(stateMachine as any, "scheduleDrain");

            // Start draining
            const drainPromise = (stateMachine as any).drain();

            // Add another event while draining
            const event2 = createTestEvent("test/event", "event-2");
            await stateMachine.handleEvent(event2);

            await drainPromise;
            await new Promise(resolve => setTimeout(resolve, 10));

            // scheduleDrain should be called again for the new event
            expect(scheduleDrainSpy).toHaveBeenCalled();
        });

        test("should not drain when disposed", async () => {
            const event = createTestEvent();
            await stateMachine.handleEvent(event);
            await stateMachine.stop();

            await (stateMachine as any).drain();

            // Event should still be in queue (not processed)
            expect(stateMachine.lastProcessedEvent).toBeNull();
        });

        test("should clear pending drain timeout", () => {
            stateMachine.testScheduleDrain(100); // Schedule with delay

            // Should set a timeout
            expect((stateMachine as any).pendingDrainTimeout).not.toBeNull();

            stateMachine.testClearPendingDrainTimeout();

            // Should clear the timeout
            expect((stateMachine as any).pendingDrainTimeout).toBeNull();
        });

        test("should schedule immediate drain with no delay", () => {
            const setImmediateSpy = vi.spyOn(global, "setImmediate");

            stateMachine.testScheduleDrain(0);

            expect(setImmediateSpy).toHaveBeenCalled();
        });

        test("should schedule delayed drain with timeout", () => {
            const setTimeoutSpy = vi.spyOn(global, "setTimeout");

            stateMachine.testScheduleDrain(100);

            expect(setTimeoutSpy).toHaveBeenCalledWith(expect.any(Function), 100);
        });

        test("should not schedule drain when paused", async () => {
            // Valid path to PAUSED
            await stateMachine.testApplyStateTransition(RunState.LOADING, "setup1");
            await stateMachine.testApplyStateTransition(RunState.CONFIGURING, "setup2");
            await stateMachine.testApplyStateTransition(RunState.READY, "setup3");
            await stateMachine.testApplyStateTransition(RunState.PAUSED, "setup4");

            const setTimeoutSpy = vi.spyOn(global, "setTimeout");
            const setImmediateSpy = vi.spyOn(global, "setImmediate");

            stateMachine.testScheduleDrain();

            expect(setTimeoutSpy).not.toHaveBeenCalled();
            expect(setImmediateSpy).not.toHaveBeenCalled();
        });
    });

    describe("Distributed Lock Coordination", () => {
        test("should acquire and release distributed processing lock", async () => {
            await stateMachine.testApplyStateTransition(RunState.READY, "setup");
            const event = createTestEvent();
            await stateMachine.handleEvent(event);

            const statusBefore = stateMachine.getCoordinationStatus();
            expect(statusBefore.currentLock).toBeNull();

            // Trigger drain which should acquire lock
            await (stateMachine as any).drain();

            // Lock should be released after drain
            const statusAfter = stateMachine.getCoordinationStatus();
            expect(statusAfter.currentLock).toBeNull();
        });

        test("should handle lock acquisition failure gracefully", async () => {
            // Mock the private method to simulate lock failure
            const originalAcquire = (stateMachine as any).acquireDistributedProcessingLock;
            (stateMachine as any).acquireDistributedProcessingLock = vi.fn().mockResolvedValue(false);

            await stateMachine.testApplyStateTransition(RunState.READY, "setup");
            const event = createTestEvent();
            await stateMachine.handleEvent(event);

            await (stateMachine as any).drain();

            // Event should still be in queue since lock wasn't acquired
            expect(stateMachine.getEventQueueLength()).toBe(1);

            // Restore original method
            (stateMachine as any).acquireDistributedProcessingLock = originalAcquire;
        });
    });

    describe("Event Publishing", () => {
        test("should publish events with correct metadata", async () => {
            const { EventPublisher } = await import("../../events/publisher.js");

            await (stateMachine as any).publishEvent("test/custom", { data: "test" }, { custom: "metadata" });

            expect(EventPublisher.emit).toHaveBeenCalledWith(
                "test/custom",
                { data: "test" },
                { custom: "metadata" },
            );
        });

        test("should handle blocked event publication gracefully", async () => {
            const { EventPublisher } = await import("../../events/publisher.js");
            EventPublisher.emit.mockResolvedValueOnce({ proceed: false, reason: "Rate limited" });

            // Should not throw
            await expect((stateMachine as any).publishEvent("test/blocked", {})).resolves.not.toThrow();
        });

        test("should handle event publication errors gracefully", async () => {
            const { EventPublisher } = await import("../../events/publisher.js");
            EventPublisher.emit.mockRejectedValueOnce(new Error("Publication failed"));

            // Should not throw
            await expect((stateMachine as any).publishEvent("test/error", {})).resolves.not.toThrow();
        });
    });

    describe("Utility Methods", () => {
        test("should validate events with required fields", () => {
            const validEvent = createTestEvent();
            const result = (stateMachine as any).validateEvent(validEvent, ["id", "type"]);
            expect(result).toBe(true);
        });

        test("should reject events missing required fields", () => {
            const invalidEvent = { type: "test/event" }; // Missing id
            const result = (stateMachine as any).validateEvent(invalidEvent, ["id", "type"]);
            expect(result).toBe(false);
        });

        test("should log lifecycle events", async () => {
            const { logger } = await import("../../../events/logger.js");

            (stateMachine as any).logLifecycleEvent("test_event", { custom: "data" });

            expect(logger.info).toHaveBeenCalledWith(
                "[TestStateMachine] test_event",
                expect.objectContaining({
                    state: RunState.UNINITIALIZED,
                    custom: "data",
                }),
            );
        });
    });

    describe("Coordination Status", () => {
        test("should return coordination status", () => {
            const config: StateMachineCoordinationConfig = {
                chatId: "chat-123",
                swarmId: "swarm-456",
            };
            const machine = new TestStateMachine(RunState.UNINITIALIZED, "TestMachine", config);

            const status = machine.getCoordinationStatus();

            expect(status.chatId).toBe("chat-123");
            expect(status.currentLock).toBeNull();
        });
    });

    // Helper function for tests
    function createTestEvent(type = "test/event", id = "event-1"): ServiceEvent {
        return {
            id,
            type,
            timestamp: new Date(),
            tier: "tier2",
            source: "test",
            chatId: "test-chat",
        };
    }
});
