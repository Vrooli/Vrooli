import { generatePK } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { TestEventGuards } from "../../__test/helpers/testEvents.js";
import { logger } from "../../events/logger.js";
import { EVENT_BUS_CONSTANTS } from "./constants.js";
import { EventBus, getEventBus } from "./eventBus.js";
import { EventBusMonitor } from "./EventBusMonitor.js";
import { EventBusRateLimiter } from "./rateLimiter.js";
import { getEventBehavior } from "./registry.js";
import type { EventPublishResult, ServiceEvent } from "./types.js";
import { EventMode } from "./types.js";

// Mock dependencies
vi.mock("../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        debug: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
    },
}));

vi.mock("./EventBusMonitor.js", () => ({
    EventBusMonitor: vi.fn(),
}));

vi.mock("./rateLimiter.js", () => ({
    EventBusRateLimiter: vi.fn(),
}));

vi.mock("./registry.js", () => ({
    getEventBehavior: vi.fn(() => ({
        mode: 0, // EventMode.PASSIVE
        interceptable: false,
    })),
}));

const mockSocketService = {
    emitSocketEvent: vi.fn(),
};

vi.mock("../../sockets/io.js", () => ({
    SocketService: {
        get: vi.fn(() => mockSocketService),
    },
}));

vi.mock("./constants.js", () => ({
    EVENT_BUS_CONSTANTS: {
        MAX_EVENT_LISTENERS: 100,
        DEFAULT_BARRIER_TIMEOUT_MS: 30000,
        RETRY_BASE_DELAY_MS: 100,
        RETRY_EXPONENTIAL_FACTOR: 2,
    },
}));

// Mock mqtt-match with fallback behavior
vi.mock("mqtt-match", () => ({
    default: (pattern: string, topic: string) => {
        // Simple fallback pattern matching logic
        let regexPattern = pattern;

        // Handle # multi-level wildcard
        if (pattern.endsWith("/#")) {
            regexPattern = pattern.replace("/#", "(/.*)");
        } else if (pattern === "#") {
            regexPattern = ".*";
        } else {
            regexPattern = regexPattern.replace(/#/g, ".*");
        }

        // Handle + single level wildcard
        regexPattern = regexPattern.replace(/\+/g, "[^/]+");

        // Handle * within level wildcard
        regexPattern = regexPattern.replace(/\*/g, "[^/]*");

        const regex = new RegExp(`^${regexPattern}$`);
        return regex.test(topic);
    },
}));

describe("EventBus", () => {
    let eventBus: EventBus;
    let mockRateLimiter: any;
    let mockMonitor: any;

    beforeEach(() => {
        vi.clearAllMocks();

        // Reset socket service mock
        mockSocketService.emitSocketEvent.mockClear();

        // Reset registry mock to default behavior
        vi.mocked(getEventBehavior).mockReturnValue({
            mode: EventMode.PASSIVE,
            interceptable: false,
        });

        // Create fresh mocks for each test
        mockRateLimiter = {
            checkEventRateLimit: vi.fn().mockResolvedValue({
                allowed: true,
                remainingQuota: 100,
                resetTime: Date.now() + 60000,
            }),
            createRateLimitedEvent: vi.fn().mockReturnValue({
                id: "rate-limited-event",
                type: "system/rate_limited",
                timestamp: new Date(),
                data: {},
            }),
            getRateLimitStatus: vi.fn().mockResolvedValue({
                global: { allowed: 100, total: 100 },
            }),
        };

        mockMonitor = {
            start: vi.fn(),
            stop: vi.fn(),
            recordEvent: vi.fn(),
            getPerformanceReport: vi.fn().mockReturnValue({
                averageLatency: 10,
                eventThroughput: 100,
                errorRate: 0.01,
            }),
            getSystemHealth: vi.fn().mockReturnValue({
                status: "healthy",
                cpuUsage: 0.5,
                memoryUsage: 0.3,
            }),
            getOptimizationSuggestions: vi.fn().mockReturnValue([]),
        };

        // Mock the constructor calls
        vi.mocked(EventBusRateLimiter).mockImplementation(() => mockRateLimiter);
        vi.mocked(EventBusMonitor).mockImplementation(() => mockMonitor);

        eventBus = new EventBus(mockRateLimiter);
    });

    afterEach(async () => {
        if (eventBus) {
            await eventBus.stop();
        }
    });

    describe("constructor", () => {
        it("should initialize with default rate limiter when none provided", () => {
            const bus = new EventBus();
            expect(bus).toBeInstanceOf(EventBus);
        });

        it("should use provided rate limiter", () => {
            const customRateLimiter = new EventBusRateLimiter();
            const bus = new EventBus(customRateLimiter);
            expect(bus).toBeInstanceOf(EventBus);
        });

        it("should set up EventEmitter with correct max listeners", () => {
            expect(eventBus).toBeInstanceOf(EventBus);
        });
    });

    describe("lifecycle management", () => {
        it("should start successfully", async () => {
            await eventBus.start();
            expect(mockMonitor.start).toHaveBeenCalled();
            expect(logger.info).toHaveBeenCalledWith("[EventBus] Started enhanced event bus with performance monitoring");
        });

        it("should handle double start gracefully", async () => {
            await eventBus.start();
            await eventBus.start();
            expect(logger.warn).toHaveBeenCalledWith("[EventBus] Already started");
        });

        it("should stop successfully and clean up resources", async () => {
            await eventBus.start();

            // Add a subscription to test cleanup
            const handler = vi.fn();
            await eventBus.subscribe("test/event", handler);

            await eventBus.stop();

            expect(mockMonitor.stop).toHaveBeenCalled();
            expect(logger.info).toHaveBeenCalledWith("[EventBus] Stopped event bus");
        });

        it("should handle stop when not started", async () => {
            await eventBus.stop();
            // Should not throw error
        });

        it("should clean up pending barriers on stop", async () => {
            await eventBus.start();

            // Mock a barrier sync event that defers (keeps barrier pending)
            const getEventBehaviorMock = getEventBehavior as any;
            getEventBehaviorMock.mockReturnValue({
                mode: EventMode.APPROVAL,
                barrierConfig: {
                    quorum: 1,
                    timeoutMs: 30000,
                    timeoutAction: "defer", // Use defer to keep barrier pending
                },
            });

            // Start a barrier event but don't resolve it
            const publishPromise = eventBus.publish({
                type: "test/approval",
                data: { test: true },
            });

            // Give time for the barrier to be created
            await new Promise(resolve => setTimeout(resolve, 50));

            // Verify barrier is pending
            const metricsBeforeStop = eventBus.getMetrics();
            expect(metricsBeforeStop.pendingBarriers).toBe(1);

            // Stop the bus
            await eventBus.stop();

            // The publish should be rejected with "Event bus stopped"
            await expect(publishPromise).rejects.toThrow("Event bus stopped");
        });
    });

    describe("event publishing", () => {
        beforeEach(async () => {
            await eventBus.start();
        });

        it("should publish simple event successfully", async () => {
            const event = {
                type: "test/simple",
                data: { message: "hello" },
            };

            const result = await eventBus.publish(event);

            expect(result.success).toBe(true);
            expect(result.eventId).toBeDefined();
            expect(result.duration).toBeGreaterThanOrEqual(0);
            expect(mockRateLimiter.checkEventRateLimit).toHaveBeenCalled();
        });

        it("should fail when event bus not started", async () => {
            await eventBus.stop();

            const event = {
                type: "test/simple",
                data: { message: "hello" },
            };

            const result = await eventBus.publish(event);

            expect(result.success).toBe(false);
            expect(result.error?.message).toBe("Event bus not started");
        });

        it("should handle rate limiting", async () => {
            mockRateLimiter.checkEventRateLimit.mockResolvedValueOnce({
                allowed: false,
                limitType: "global",
                retryAfterMs: 1000,
            });

            const event = {
                type: "test/rate_limited",
                data: { message: "hello" },
            };

            const result = await eventBus.publish(event);

            expect(result.success).toBe(false);
            expect(result.error?.message).toBe("Rate limit exceeded: global");
            expect(mockRateLimiter.createRateLimitedEvent).toHaveBeenCalled();
        });

        it("should record metrics for successful events", async () => {
            const metrics = eventBus.getMetrics();
            const initialPublished = metrics.eventsPublished;

            await eventBus.publish({
                type: "test/metrics",
                data: { test: true },
            });

            const updatedMetrics = eventBus.getMetrics();
            expect(updatedMetrics.eventsPublished).toBe(initialPublished + 1);
            expect(updatedMetrics.eventsDelivered).toBe(metrics.eventsDelivered + 1);
        });

        it("should record metrics for failed events", async () => {
            const metrics = eventBus.getMetrics();
            const initialFailed = metrics.eventsFailed;

            // Mock an error during event processing
            const getEventBehaviorMock = getEventBehavior as any;
            getEventBehaviorMock.mockImplementationOnce(() => {
                throw new Error("Registry error");
            });

            const result = await eventBus.publish({
                type: "test/error",
                data: { test: true },
            });

            expect(result.success).toBe(false);
            const updatedMetrics = eventBus.getMetrics();
            expect(updatedMetrics.eventsFailed).toBe(initialFailed + 1);
        });
    });

    describe("barrier synchronization", () => {
        beforeEach(async () => {
            await eventBus.start();
        });

        it("should handle approval mode events with barrier", async () => {
            const getEventBehaviorMock = getEventBehavior as any;
            getEventBehaviorMock.mockReturnValue({
                mode: EventMode.APPROVAL,
                barrierConfig: {
                    quorum: 1,
                    timeoutMs: 100,
                    timeoutAction: "continue",
                },
            });

            const result = await eventBus.publish({
                type: "test/approval",
                data: { needsApproval: true },
            });

            expect(result.success).toBe(true);
            expect(result.wasBlocking).toBe(true);
            expect(result.progression).toBe("continue");
        });

        it("should handle consensus mode events", async () => {
            const getEventBehaviorMock = getEventBehavior as any;
            getEventBehaviorMock.mockReturnValue({
                mode: EventMode.CONSENSUS,
                barrierConfig: {
                    quorum: 2,
                    timeoutMs: 100,
                    timeoutAction: "block",
                },
            });

            const result = await eventBus.publish({
                type: "test/consensus",
                data: { requiresConsensus: true },
            });

            expect(result.success).toBe(true);
            expect(result.wasBlocking).toBe(true);
            expect(result.progression).toBe("block");
        });

        it("should handle barrier timeout with continue action", async () => {
            const getEventBehaviorMock = getEventBehavior as any;
            getEventBehaviorMock.mockReturnValue({
                mode: EventMode.APPROVAL,
                barrierConfig: {
                    quorum: 1,
                    timeoutMs: 100,
                    timeoutAction: "continue",
                },
            });

            const result = await eventBus.publish({
                type: "test/timeout_continue",
                data: { test: true },
            });

            expect(result.success).toBe(true);
            expect(result.wasBlocking).toBe(true);
            expect(result.progression).toBe("continue");
        });

        it("should handle barrier timeout with defer action", async () => {
            const getEventBehaviorMock = getEventBehavior as any;
            getEventBehaviorMock.mockReturnValue({
                mode: EventMode.APPROVAL,
                barrierConfig: {
                    quorum: 1,
                    timeoutMs: 50,
                    timeoutAction: "defer",
                },
            });

            // For defer action, the barrier stays pending and doesn't resolve
            const publishPromise = eventBus.publish({
                type: "test/timeout_defer",
                data: { test: true },
            });

            // Wait a bit then check metrics
            await new Promise(resolve => setTimeout(resolve, 100));
            const metrics = eventBus.getMetrics();

            // Should still have pending barrier due to defer action
            expect(metrics.pendingBarriers).toBe(1);
        });
    });

    describe("subscription management", () => {
        beforeEach(async () => {
            await eventBus.start();
        });

        it("should add subscription successfully", async () => {
            const handler = vi.fn();
            const subscriptionId = await eventBus.subscribe("test/event", handler);

            expect(subscriptionId).toBeDefined();
            expect(typeof subscriptionId).toBe("string");

            const metrics = eventBus.getMetrics();
            expect(metrics.activeSubscriptions).toBe(1);
        });

        it("should remove subscription successfully", async () => {
            const handler = vi.fn();
            const subscriptionId = await eventBus.subscribe("test/event", handler);

            await eventBus.unsubscribe(subscriptionId);

            const metrics = eventBus.getMetrics();
            expect(metrics.activeSubscriptions).toBe(0);
        });

        it("should handle multiple patterns in subscription", async () => {
            const handler = vi.fn();
            const patterns = ["test/event1", "test/event2", "test/event3"];

            const subscriptionId = await eventBus.subscribe(patterns, handler);
            expect(subscriptionId).toBeDefined();
        });

        it("should call subscription handlers for matching events", async () => {
            const handler = vi.fn();
            await eventBus.subscribe("test/simple", handler);

            await eventBus.publish({
                type: "test/simple",
                data: { message: "hello" },
            });

            // Allow async event delivery
            await new Promise(resolve => setTimeout(resolve, 10));

            expect(handler).toHaveBeenCalled();
        });

        it("should support wildcard patterns", async () => {
            const handler = vi.fn();
            await eventBus.subscribe("test/+", handler);

            await eventBus.publish({
                type: "test/simple",
                data: { message: "hello" },
            });

            await new Promise(resolve => setTimeout(resolve, 10));
            expect(handler).toHaveBeenCalled();
        });

        it("should support multi-level wildcard patterns", async () => {
            const handler = vi.fn();
            await eventBus.subscribe("test/#", handler);

            await eventBus.publish({
                type: "test/deep/nested/event",
                data: { message: "hello" },
            });

            await new Promise(resolve => setTimeout(resolve, 10));
            expect(handler).toHaveBeenCalled();
        });

        it("should handle subscription filters", async () => {
            const handler = vi.fn();
            const filter = (event: ServiceEvent) => {
                if (TestEventGuards.isTestFiltered(event)) {
                    return event.data.important === true;
                }
                return false;
            };

            await eventBus.subscribe("test/filtered", handler, { filter });

            // This should be filtered out
            await eventBus.publish({
                type: "test/filtered",
                data: { important: false },
            });

            // This should pass through
            await eventBus.publish({
                type: "test/filtered",
                data: { important: true },
            });

            await new Promise(resolve => setTimeout(resolve, 10));
            expect(handler).toHaveBeenCalledTimes(1);
        });

        it("should handle subscription retries", async () => {
            const handler = vi.fn()
                .mockRejectedValueOnce(new Error("First attempt failed"))
                .mockResolvedValueOnce(undefined);

            await eventBus.subscribe("test/retry", handler, { maxRetries: 1 });

            await eventBus.publish({
                type: "test/retry",
                data: { test: true },
            });

            await new Promise(resolve => setTimeout(resolve, 200));
            expect(handler).toHaveBeenCalledTimes(2);
        });
    });

    describe("pattern matching", () => {
        beforeEach(async () => {
            await eventBus.start();
        });

        it("should match exact patterns", async () => {
            const handler = vi.fn();
            await eventBus.subscribe("test/exact", handler);

            await eventBus.publish({
                type: "test/exact",
                data: {},
            });

            await new Promise(resolve => setTimeout(resolve, 10));
            expect(handler).toHaveBeenCalled();
        });

        it("should not match non-matching patterns", async () => {
            const handler = vi.fn();
            await eventBus.subscribe("test/specific", handler);

            await eventBus.publish({
                type: "test/different",
                data: {},
            });

            await new Promise(resolve => setTimeout(resolve, 10));
            expect(handler).not.toHaveBeenCalled();
        });

        it("should get subscriber count for pattern", () => {
            const handler1 = vi.fn();
            const handler2 = vi.fn();

            eventBus.subscribe("test/count", handler1);
            eventBus.subscribe("test/count", handler2);
            eventBus.subscribe("other/pattern", handler1);

            const count = eventBus.getSubscriberCount("test/count");
            expect(count).toBe(2);
        });
    });

    describe("socket integration", () => {
        beforeEach(async () => {
            await eventBus.start();
        });

        it("should emit events to socket clients", async () => {
            const chatId = generatePK().toString();

            await eventBus.publish({
                type: "chat/message",
                data: { chatId, message: "hello" },
            });

            expect(mockSocketService.emitSocketEvent).toHaveBeenCalledWith(
                "chat/message",
                chatId,
                expect.objectContaining({
                    type: "chat/message",
                    data: { chatId, message: "hello" },
                }),
            );
        });

        it("should handle socket emission errors gracefully", async () => {
            // Make the socket service throw an error
            mockSocketService.emitSocketEvent.mockImplementation(() => {
                throw new Error("Socket error");
            });

            // Should not throw despite socket error
            const chatId = generatePK().toString();
            const result = await eventBus.publish({
                type: "chat/message",
                data: { chatId, message: "hello" },
            });

            expect(result.success).toBe(true);
            expect(logger.error).toHaveBeenCalledWith(
                "[EventBus] Failed to emit event to socket clients",
                expect.any(Object),
            );

            // Reset the mock
            mockSocketService.emitSocketEvent.mockReset();
        });

        it("should extract room IDs correctly", async () => {
            const chatId = generatePK().toString();
            const runId = generatePK().toString();
            const userId = generatePK().toString();

            // Test chat events
            await eventBus.publish({
                type: "chat/message",
                data: { chatId },
            });

            // Test run events  
            await eventBus.publish({
                type: "run/started",
                data: { runId },
            });

            // Test user events
            await eventBus.publish({
                type: "user/login",
                data: { userId },
            });

            expect(mockSocketService.emitSocketEvent).toHaveBeenCalledTimes(3);
            expect(mockSocketService.emitSocketEvent).toHaveBeenNthCalledWith(
                1, "chat/message", chatId, expect.any(Object),
            );
            expect(mockSocketService.emitSocketEvent).toHaveBeenNthCalledWith(
                2, "run/started", runId, expect.any(Object),
            );
            expect(mockSocketService.emitSocketEvent).toHaveBeenNthCalledWith(
                3, "user/login", userId, expect.any(Object),
            );
        });
    });

    describe("metrics and monitoring", () => {
        beforeEach(async () => {
            await eventBus.start();
        });

        it("should provide comprehensive metrics", () => {
            const metrics = eventBus.getMetrics();

            expect(metrics).toMatchObject({
                eventsPublished: expect.any(Number),
                eventsDelivered: expect.any(Number),
                eventsFailed: expect.any(Number),
                eventsRateLimited: expect.any(Number),
                barrierSyncsCompleted: expect.any(Number),
                barrierSyncsTimedOut: expect.any(Number),
                activeSubscriptions: expect.any(Number),
                lastEventTime: expect.any(Number),
                pendingBarriers: expect.any(Number),
                rateLimitingEnabled: true,
                performance: expect.any(Object),
                health: expect.any(Object),
                optimizations: expect.any(Array),
            });
        });

        it("should provide rate limit status", async () => {
            const userId = generatePK().toString();
            const status = await eventBus.getRateLimitStatus(userId, "test/event");

            expect(status).toMatchObject({
                global: {
                    allowed: expect.any(Number),
                    total: expect.any(Number),
                },
            });
            expect(mockRateLimiter.getRateLimitStatus).toHaveBeenCalledWith(userId, "test/event");
        });

        it("should track metrics correctly across operations", async () => {
            const initialMetrics = eventBus.getMetrics();

            // Add subscription
            await eventBus.subscribe("test/metrics", vi.fn());

            // Publish event
            await eventBus.publish({
                type: "test/metrics",
                data: { test: true },
            });

            const finalMetrics = eventBus.getMetrics();

            expect(finalMetrics.activeSubscriptions).toBe(initialMetrics.activeSubscriptions + 1);
            expect(finalMetrics.eventsPublished).toBe(initialMetrics.eventsPublished + 1);
            expect(finalMetrics.eventsDelivered).toBe(initialMetrics.eventsDelivered + 1);
        });
    });

    describe("error handling", () => {
        beforeEach(async () => {
            await eventBus.start();
        });

        it("should handle subscription handler errors gracefully", async () => {
            const faultyHandler = vi.fn().mockRejectedValue(new Error("Handler error"));
            await eventBus.subscribe("test/error", faultyHandler);

            // Should not throw despite handler error
            const result = await eventBus.publish({
                type: "test/error",
                data: { test: true },
            });

            expect(result.success).toBe(true);

            // Allow async error handling
            await new Promise(resolve => setTimeout(resolve, 10));

            expect(logger.error).toHaveBeenCalledWith(
                "[EventBus] Subscription handler error",
                expect.any(Object),
            );
        });

        it("should handle pattern matching errors", async () => {
            // Mock mqtt-match import failure
            vi.doMock("mqtt-match", () => {
                throw new Error("Import failed");
            });

            const handler = vi.fn();
            await eventBus.subscribe("test/pattern", handler);

            // Should fall back to simple pattern matching
            await eventBus.publish({
                type: "test/pattern",
                data: { test: true },
            });

            await new Promise(resolve => setTimeout(resolve, 10));
            expect(handler).toHaveBeenCalled();
        });

        it("should handle unknown barrier responses", async () => {
            // Should not throw error
            const unknownEventId = generatePK().toString();
            const botId = generatePK().toString();
            await eventBus.respondToBarrier(unknownEventId, botId, {
                progression: "continue",
                reason: "Test",
            });

            expect(logger.warn).toHaveBeenCalledWith(
                "[EventBus] Received response for unknown or completed barrier",
                expect.any(Object),
            );
        });
    });

    describe("singleton instance", () => {
        it("should return same instance from getEventBus", () => {
            const instance1 = getEventBus();
            const instance2 = getEventBus();

            expect(instance1).toBe(instance2);
        });

        it("should create new instance if none exists", () => {
            const instance = getEventBus();
            expect(instance).toBeInstanceOf(EventBus);
        });
    });

    describe("concurrent access and race conditions", () => {
        beforeEach(async () => {
            await eventBus.start();
        });

        it("should handle concurrent barrier responses without race conditions", async () => {
            const getEventBehaviorMock = getEventBehavior as any;
            getEventBehaviorMock.mockReturnValue({
                mode: EventMode.CONSENSUS,
                barrierConfig: {
                    quorum: 3,
                    timeoutMs: 100,
                    timeoutAction: "block",
                },
            });

            const result = await eventBus.publish({
                type: "test/concurrent_barrier",
                data: { test: true },
            });

            expect(result.success).toBe(true);
            expect(result.wasBlocking).toBe(true);
            expect(result.progression).toBe("block");
        });

        it("should handle concurrent subscription modifications during event processing", async () => {
            const handlers = Array(10).fill(null).map(() => vi.fn());
            const subscriptionPromises = handlers.map(handler =>
                eventBus.subscribe("test/concurrent_sub", handler),
            );

            const subscriptionIds = await Promise.all(subscriptionPromises);

            // Publish event while simultaneously modifying subscriptions
            const publishPromise = eventBus.publish({
                type: "test/concurrent_sub",
                data: { test: true },
            });

            // Unsubscribe half the handlers concurrently
            const unsubPromises = subscriptionIds.slice(0, 5).map(id =>
                eventBus.unsubscribe(id),
            );

            await Promise.all([publishPromise, ...unsubPromises]);

            // Allow event processing to complete
            await new Promise(resolve => setTimeout(resolve, 20));

            // Only remaining handlers should have been called
            const calledHandlers = handlers.filter(handler => handler.mock.calls.length > 0);
            expect(calledHandlers.length).toBe(5);
        });

        it("should handle rapid sequential event publishing without corruption", async () => {
            const handler = vi.fn();
            await eventBus.subscribe("test/rapid", handler);

            const eventCount = 100;
            const publishPromises = Array(eventCount).fill(null).map((_, i) =>
                eventBus.publish({
                    type: "test/rapid",
                    data: { index: i },
                }),
            );

            const results = await Promise.all(publishPromises);

            // All events should succeed
            expect(results.every(r => r.success)).toBe(true);

            // Allow all handlers to complete
            await new Promise(resolve => setTimeout(resolve, 100));

            // Handler should be called for each event
            expect(handler).toHaveBeenCalledTimes(eventCount);
        });
    });

    describe("memory management and resource leaks", () => {
        beforeEach(async () => {
            await eventBus.start();
        });

        it("should clean up pending barriers on timeout without memory leaks", async () => {
            const getEventBehaviorMock = getEventBehavior as any;
            getEventBehaviorMock.mockReturnValue({
                mode: EventMode.APPROVAL,
                barrierConfig: {
                    quorum: 1,
                    timeoutMs: 50,
                    timeoutAction: "block",
                },
            });

            const initialMetrics = eventBus.getMetrics();

            // Create multiple barrier events that will timeout
            const promises = Array(10).fill(null).map((_, i) =>
                eventBus.publish({
                    type: `test/timeout_${i}`,
                    data: { index: i },
                }),
            );

            await Promise.all(promises);

            // Verify barriers were cleaned up
            const finalMetrics = eventBus.getMetrics();
            expect(finalMetrics.pendingBarriers).toBe(0);
            expect(finalMetrics.barrierSyncsTimedOut).toBe(10);
        });

        it("should handle subscription cleanup during high event volume", async () => {
            const handlers = Array(50).fill(null).map(() => vi.fn());
            const subscriptionIds = await Promise.all(
                handlers.map(handler => eventBus.subscribe("test/cleanup", handler)),
            );

            // Start publishing events rapidly
            const publishInterval = setInterval(() => {
                eventBus.publish({
                    type: "test/cleanup",
                    data: { timestamp: Date.now() },
                });
            }, 1);

            // Remove subscriptions while events are flowing
            await new Promise(resolve => setTimeout(resolve, 20));
            await Promise.all(subscriptionIds.map(id => eventBus.unsubscribe(id)));

            clearInterval(publishInterval);

            // Verify cleanup
            const metrics = eventBus.getMetrics();
            expect(metrics.activeSubscriptions).toBe(0);
        });

        it("should prevent EventEmitter memory leaks with many subscriptions", async () => {
            const maxListeners = EVENT_BUS_CONSTANTS.MAX_EVENT_LISTENERS;
            const handlers = Array(maxListeners + 10).fill(null).map(() => vi.fn());

            // This should not cause memory leak warnings
            const consoleWarnSpy = vi.spyOn(console, "warn").mockImplementation(() => { });

            const subscriptionIds = await Promise.all(
                handlers.map(handler => eventBus.subscribe("test/memory", handler)),
            );

            // Should not have triggered EventEmitter memory leak warnings
            expect(consoleWarnSpy).not.toHaveBeenCalledWith(
                expect.stringContaining("MaxListenersExceededWarning"),
            );

            // Clean up
            await Promise.all(subscriptionIds.map(id => eventBus.unsubscribe(id)));
            consoleWarnSpy.mockRestore();
        });
    });

    describe("error recovery and resilience", () => {
        beforeEach(async () => {
            await eventBus.start();
        });

        it("should recover from rate limiter failures", async () => {
            mockRateLimiter.checkEventRateLimit.mockRejectedValueOnce(new Error("Rate limiter error"));

            const result = await eventBus.publish({
                type: "test/rate_limiter_error",
                data: { test: true },
            });

            expect(result.success).toBe(false);
            expect(result.error?.message).toBe("Rate limiter error");
        });

        it("should continue operating when monitor service fails", async () => {
            mockMonitor.recordEvent.mockImplementation(() => {
                throw new Error("Monitor error");
            });

            const result = await eventBus.publish({
                type: "test/monitor_error",
                data: { test: true },
            });

            // Should still succeed despite monitor failure
            expect(result.success).toBe(true);
        });

        it("should handle registry failures gracefully", async () => {
            const getEventBehaviorMock = getEventBehavior as any;
            getEventBehaviorMock.mockImplementation(() => {
                throw new Error("Registry failure");
            });

            const result = await eventBus.publish({
                type: "test/registry_error",
                data: { test: true },
            });

            expect(result.success).toBe(false);
            expect(result.error?.message).toBe("Registry failure");
        });

        it("should handle subscription handler failures without affecting other handlers", async () => {
            const goodHandler = vi.fn();
            const badHandler = vi.fn().mockRejectedValue(new Error("Handler error"));
            const anotherGoodHandler = vi.fn();

            await eventBus.subscribe("test/mixed_handlers", goodHandler);
            await eventBus.subscribe("test/mixed_handlers", badHandler);
            await eventBus.subscribe("test/mixed_handlers", anotherGoodHandler);

            const result = await eventBus.publish({
                type: "test/mixed_handlers",
                data: { test: true },
            });

            expect(result.success).toBe(true);

            // Allow async processing
            await new Promise(resolve => setTimeout(resolve, 20));

            // Good handlers should still be called
            expect(goodHandler).toHaveBeenCalled();
            expect(anotherGoodHandler).toHaveBeenCalled();
            expect(badHandler).toHaveBeenCalled();
        });
    });

    describe("performance and load testing", () => {
        beforeEach(async () => {
            await eventBus.start();
        });

        it("should handle high-frequency event publishing", async () => {
            const eventCount = 1000;
            const startTime = performance.now();

            const promises = Array(eventCount).fill(null).map((_, i) =>
                eventBus.publish({
                    type: "test/high_frequency",
                    data: { index: i },
                }),
            );

            const results = await Promise.all(promises);
            const duration = performance.now() - startTime;

            // All events should succeed
            expect(results.every(r => r.success)).toBe(true);

            // Should complete within reasonable time (< 5 seconds)
            expect(duration).toBeLessThan(5000);

            // Should maintain reasonable throughput (> 200 events/second)
            const throughput = eventCount / (duration / 1000);
            expect(throughput).toBeGreaterThan(200);
        });

        it("should handle large number of concurrent subscriptions", async () => {
            const subscriptionCount = 500;
            const handler = vi.fn();

            const startTime = performance.now();
            const subscriptionPromises = Array(subscriptionCount).fill(null).map((_, i) =>
                eventBus.subscribe(`test/load_${i}`, handler),
            );

            const subscriptionIds = await Promise.all(subscriptionPromises);
            const subscriptionDuration = performance.now() - startTime;

            // Should complete subscriptions quickly (< 1 second)
            expect(subscriptionDuration).toBeLessThan(1000);

            // Verify all subscriptions were created
            expect(subscriptionIds.length).toBe(subscriptionCount);

            const metrics = eventBus.getMetrics();
            expect(metrics.activeSubscriptions).toBe(subscriptionCount);

            // Clean up
            await Promise.all(subscriptionIds.map(id => eventBus.unsubscribe(id)));
        });

        it("should handle barrier synchronization under load", async () => {
            const getEventBehaviorMock = getEventBehavior as any;
            getEventBehaviorMock.mockReturnValue({
                mode: EventMode.APPROVAL,
                barrierConfig: {
                    quorum: 1,
                    timeoutMs: 100,
                    timeoutAction: "continue",
                },
            });

            const eventCount = 100;
            const startTime = performance.now();

            const promises = Array(eventCount).fill(null).map((_, i) =>
                eventBus.publish({
                    type: "test/barrier_load",
                    data: { index: i },
                }),
            );

            const results = await Promise.all(promises);
            const duration = performance.now() - startTime;

            // All events should succeed
            expect(results.every(r => r.success)).toBe(true);

            // Should complete within reasonable time (< 10 seconds)
            expect(duration).toBeLessThan(10000);

            const metrics = eventBus.getMetrics();
            expect(metrics.barrierSyncsTimedOut).toBe(eventCount);
        });
    });

    describe("event ordering and consistency", () => {
        beforeEach(async () => {
            await eventBus.start();
        });

        it("should maintain event ordering for sequential publications", async () => {
            const receivedEvents: ServiceEvent[] = [];
            const handler = vi.fn(async (event: ServiceEvent) => {
                receivedEvents.push(event);
            });

            await eventBus.subscribe("test/ordering", handler);

            // Publish events sequentially
            for (let i = 0; i < 10; i++) {
                await eventBus.publish({
                    type: "test/ordering",
                    data: { index: i },
                });
            }

            // Allow processing to complete
            await new Promise(resolve => setTimeout(resolve, 50));

            // Events should be received in order
            for (let i = 0; i < 10; i++) {
                if (TestEventGuards.isTestOrdering(receivedEvents[i]) && "index" in receivedEvents[i].data) {
                    expect(receivedEvents[i].data.index).toBe(i);
                }
            }
        });

        it("should handle barrier event ordering correctly", async () => {
            const getEventBehaviorMock = getEventBehavior as any;
            getEventBehaviorMock.mockReturnValue({
                mode: EventMode.APPROVAL,
                barrierConfig: {
                    quorum: 1,
                    timeoutMs: 50,
                    timeoutAction: "continue",
                },
            });

            const results: EventPublishResult[] = [];
            const publishPromises = Array(5).fill(null).map((_, i) =>
                eventBus.publish({
                    type: "test/barrier_ordering",
                    data: { index: i },
                }).then(result => {
                    results.push(result);
                    return result;
                }),
            );

            await Promise.all(publishPromises);

            // All should succeed
            expect(results.every(r => r.success)).toBe(true);
            expect(results.every(r => r.wasBlocking)).toBe(true);
        });

        it("should maintain subscription handler execution order", async () => {
            const executionOrder: number[] = [];
            const handlers = Array(5).fill(null).map((_, i) =>
                vi.fn(async () => {
                    // Add small delay to test ordering
                    await new Promise(resolve => setTimeout(resolve, 10));
                    executionOrder.push(i);
                }),
            );

            // Subscribe handlers in order
            for (let i = 0; i < handlers.length; i++) {
                await eventBus.subscribe("test/handler_order", handlers[i]);
            }

            await eventBus.publish({
                type: "test/handler_order",
                data: { test: true },
            });

            // Allow all handlers to complete
            await new Promise(resolve => setTimeout(resolve, 100));

            // Handlers should execute in subscription order
            expect(executionOrder).toEqual([0, 1, 2, 3, 4]);
        });
    });

    describe("advanced barrier scenarios", () => {
        beforeEach(async () => {
            await eventBus.start();
        });

        it("should handle blockOnFirst configuration correctly", async () => {
            const getEventBehaviorMock = getEventBehavior as any;
            getEventBehaviorMock.mockReturnValue({
                mode: EventMode.CONSENSUS,
                barrierConfig: {
                    quorum: 3,
                    timeoutMs: 1000,
                    timeoutAction: "continue",
                    blockOnFirst: true,
                },
            });

            const publishPromise = eventBus.publish({
                type: "test/block_on_first",
                data: { test: true },
            });

            // Wait for barrier to be created
            await new Promise(resolve => setTimeout(resolve, 10));

            // Get the actual event ID
            let eventId: string;
            const originalEmitToSubscribers = (eventBus as any).emitToSubscribers;
            (eventBus as any).emitToSubscribers = vi.fn(async (event: ServiceEvent) => {
                eventId = event.id;
                await originalEmitToSubscribers.call(eventBus, event);
            });

            // Wait for emission
            await new Promise(resolve => setTimeout(resolve, 10));

            // First response approves
            const bot1Id = generatePK().toString();
            const bot2Id = generatePK().toString();
            await eventBus.respondToBarrier(eventId!, bot1Id, {
                progression: "continue",
                reason: "First approval",
            });

            // Second response blocks - should immediately complete
            await eventBus.respondToBarrier(eventId!, bot2Id, {
                progression: "block",
                reason: "Blocked!",
            });

            const result = await publishPromise;
            expect(result.success).toBe(true);
            expect(result.progression).toBe("block");
            expect(result.responses).toHaveLength(2);
        });

        it("should handle continueThreshold configuration", async () => {
            const getEventBehaviorMock = getEventBehavior as any;
            getEventBehaviorMock.mockReturnValue({
                mode: EventMode.CONSENSUS,
                barrierConfig: {
                    quorum: 5,
                    timeoutMs: 1000,
                    timeoutAction: "block",
                    continueThreshold: 3,
                },
            });

            const publishPromise = eventBus.publish({
                type: "test/continue_threshold",
                data: { test: true },
            });

            // Wait for barrier setup
            await new Promise(resolve => setTimeout(resolve, 10));

            // Get event ID
            let eventId: string;
            const originalEmitToSubscribers = (eventBus as any).emitToSubscribers;
            (eventBus as any).emitToSubscribers = vi.fn(async (event: ServiceEvent) => {
                eventId = event.id;
                await originalEmitToSubscribers.call(eventBus, event);
            });

            await new Promise(resolve => setTimeout(resolve, 10));

            // Send exactly 3 continue responses and 2 block responses
            const responses = [
                { botId: generatePK().toString(), progression: "continue" as const },
                { botId: generatePK().toString(), progression: "continue" as const },
                { botId: generatePK().toString(), progression: "continue" as const },
                { botId: generatePK().toString(), progression: "block" as const },
                { botId: generatePK().toString(), progression: "block" as const },
            ];

            for (const { botId, progression } of responses) {
                await eventBus.respondToBarrier(eventId!, botId, {
                    progression,
                    reason: `${botId} response`,
                });
            }

            const result = await publishPromise;
            expect(result.success).toBe(true);
            expect(result.progression).toBe("continue");
        });
    });

    describe("edge cases and boundary conditions", () => {
        beforeEach(async () => {
            await eventBus.start();
        });

        it("should handle empty event data gracefully", async () => {
            const result = await eventBus.publish({
                type: "test/empty",
                data: {},
            });

            expect(result.success).toBe(true);
        });

        it("should handle very long event type names", async () => {
            const longEventType = "test/" + "very".repeat(100) + "/long/event/type";

            const result = await eventBus.publish({
                type: longEventType,
                data: { test: true },
            });

            expect(result.success).toBe(true);
        });

        it("should handle subscription to non-existent event patterns", async () => {
            const handler = vi.fn();
            const subscriptionId = await eventBus.subscribe("non/existent/pattern", handler);

            expect(subscriptionId).toBeDefined();

            // Publishing to different pattern should not trigger
            await eventBus.publish({
                type: "different/pattern",
                data: { test: true },
            });

            await new Promise(resolve => setTimeout(resolve, 10));
            expect(handler).not.toHaveBeenCalled();
        });

        it("should handle duplicate barrier responses from same bot", async () => {
            const getEventBehaviorMock = getEventBehavior as any;
            getEventBehaviorMock.mockReturnValue({
                mode: EventMode.APPROVAL,
                barrierConfig: {
                    quorum: 1,
                    timeoutMs: 1000,
                    timeoutAction: "block",
                },
            });

            const publishPromise = eventBus.publish({
                type: "test/duplicate_response",
                data: { test: true },
            });

            await new Promise(resolve => setTimeout(resolve, 10));

            // Get event ID
            let eventId: string;
            const originalEmitToSubscribers = (eventBus as any).emitToSubscribers;
            (eventBus as any).emitToSubscribers = vi.fn(async (event: ServiceEvent) => {
                eventId = event.id;
                await originalEmitToSubscribers.call(eventBus, event);
            });

            await new Promise(resolve => setTimeout(resolve, 10));

            // Send duplicate responses from same bot
            const duplicateBotId = generatePK().toString();
            await eventBus.respondToBarrier(eventId!, duplicateBotId, {
                progression: "continue",
                reason: "First response",
            });

            await eventBus.respondToBarrier(eventId!, duplicateBotId, {
                progression: "block",
                reason: "Second response",
            });

            const result = await publishPromise;
            expect(result.success).toBe(true);
            expect(result.responses).toHaveLength(2); // Both responses should be recorded
        });
    });
});
