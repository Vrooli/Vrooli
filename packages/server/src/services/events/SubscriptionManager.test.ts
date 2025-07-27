import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { SubscriptionManager } from "./SubscriptionManager.js";
import { generatePK } from "@vrooli/shared";
import type {
    EventHandler,
    EventSubscriptionId,
    ServiceEvent,
    SubscriptionOptions,
} from "./types.js";

// Mock dependencies
vi.mock("../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        debug: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
    },
}));

describe("SubscriptionManager", () => {
    let subscriptionManager: SubscriptionManager;
    let mockHandler: jest.MockedFunction<EventHandler>;
    let sampleEvent: ServiceEvent;

    beforeEach(() => {
        subscriptionManager = new SubscriptionManager();
        mockHandler = vi.fn();

        sampleEvent = {
            id: generatePK().toString(),
            type: "test/event",
            data: { message: "test" },
            timestamp: new Date(),
            source: "test-service",
        };
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("subscription management", () => {
        it("should add a subscription successfully", () => {
            const patterns = ["test/*", "user/login"];
            const options: SubscriptionOptions = {
                priority: "high",
            };

            const id = subscriptionManager.addSubscription(patterns, mockHandler, options);

            expect(id).toMatch(/^sub_\d+_[a-z0-9]{7}$/);

            const metrics = subscriptionManager.getMetrics();
            expect(metrics.totalSubscriptions).toBe(1);
            expect(metrics.activeSubscriptions).toBe(1);
        });

        it("should add subscription with batch processing", () => {
            const patterns = ["batch/*"];
            const options: SubscriptionOptions = {
                batchSize: 10,
            };

            const id = subscriptionManager.addSubscription(patterns, mockHandler, options);

            expect(id).toBeDefined();
            expect(subscriptionManager.getMetrics().activeSubscriptions).toBe(1);
        });

        it("should remove subscription successfully", async () => {
            const id = subscriptionManager.addSubscription(["test/*"], mockHandler);

            await subscriptionManager.removeSubscription(id);

            const metrics = subscriptionManager.getMetrics();
            expect(metrics.activeSubscriptions).toBe(0);
            expect(metrics.totalSubscriptions).toBe(1); // Total stays the same
        });

        it("should handle removing non-existent subscription", async () => {
            await subscriptionManager.removeSubscription("non-existent-id" as EventSubscriptionId);
            // Should not throw error
            expect(subscriptionManager.getMetrics().activeSubscriptions).toBe(0);
        });

        it("should index subscriptions by pattern", () => {
            const id1 = subscriptionManager.addSubscription(["user/*"], mockHandler);
            const id2 = subscriptionManager.addSubscription(["user/login"], mockHandler);
            const id3 = subscriptionManager.addSubscription(["chat/*"], mockHandler);

            const userSubscriptions = subscriptionManager.findSubscriptionsByPattern("user/*");
            const loginSubscriptions = subscriptionManager.findSubscriptionsByPattern("user/login");
            const chatSubscriptions = subscriptionManager.findSubscriptionsByPattern("chat/*");

            expect(userSubscriptions).toHaveLength(1);
            expect(loginSubscriptions).toHaveLength(1);
            expect(chatSubscriptions).toHaveLength(1);
        });

        it("should track subscriptions by handler", () => {
            const handler1 = vi.fn();
            const handler2 = vi.fn();

            const id1 = subscriptionManager.addSubscription(["test/*"], handler1);
            const id2 = subscriptionManager.addSubscription(["user/*"], handler2);
            const id3 = subscriptionManager.addSubscription(["chat/*"], handler1);

            const handler1Subscriptions = subscriptionManager.getSubscriptionsByHandler(handler1);
            const handler2Subscriptions = subscriptionManager.getSubscriptionsByHandler(handler2);

            expect(handler1Subscriptions).toHaveLength(2);
            expect(handler2Subscriptions).toHaveLength(1);
            expect(handler1Subscriptions).toContain(id1);
            expect(handler1Subscriptions).toContain(id3);
            expect(handler2Subscriptions).toContain(id2);
        });
    });

    describe("event delivery", () => {
        let subscriptionId: EventSubscriptionId;

        beforeEach(() => {
            subscriptionId = subscriptionManager.addSubscription(["test/*"], mockHandler);
        });

        it("should deliver event immediately when no batching", async () => {
            const subscription = subscriptionManager["subscriptions"].get(subscriptionId)!;

            await subscriptionManager.deliverEvent(subscription, sampleEvent);

            expect(mockHandler).toHaveBeenCalledWith(sampleEvent);
            expect(mockHandler).toHaveBeenCalledTimes(1);
        });

        it("should apply event filter when configured", async () => {
            const filterId = subscriptionManager.addSubscription(
                ["test/*"],
                mockHandler,
                {
                    filter: (event) => event.data.message === "allowed",
                },
            );

            const filteredSubscription = subscriptionManager["subscriptions"].get(filterId)!;

            // Event should be filtered out
            await subscriptionManager.deliverEvent(filteredSubscription, sampleEvent);
            expect(mockHandler).not.toHaveBeenCalled();

            // Event should pass filter
            const allowedEvent = {
                ...sampleEvent,
                data: { message: "allowed" },
            };
            await subscriptionManager.deliverEvent(filteredSubscription, allowedEvent);
            expect(mockHandler).toHaveBeenCalledWith(allowedEvent);
        });

        it("should handle delivery to inactive subscriptions", async () => {
            const subscription = subscriptionManager["subscriptions"].get(subscriptionId)!;
            subscription.isActive = false;

            await subscriptionManager.deliverEvent(subscription, sampleEvent);

            expect(mockHandler).not.toHaveBeenCalled();
        });

        it("should track delivery statistics", async () => {
            const subscription = subscriptionManager["subscriptions"].get(subscriptionId)!;

            await subscriptionManager.deliverEvent(subscription, sampleEvent);

            const stats = subscriptionManager.getSubscriptionStats(subscriptionId);
            expect(stats).toBeDefined();
            expect(stats!.eventsProcessed).toBe(1);
            expect(stats!.averageProcessingTime).toBeGreaterThan(0);
            expect(stats!.lastEventTime).toBeGreaterThan(0);
            expect(stats!.errorCount).toBe(0);

            const metrics = subscriptionManager.getMetrics();
            expect(metrics.eventsDelivered).toBe(1);
            expect(metrics.deliveryErrors).toBe(0);
        });
    });

    describe("error handling and backoff", () => {
        let subscriptionId: EventSubscriptionId;

        beforeEach(() => {
            mockHandler.mockRejectedValue(new Error("Handler error"));
            subscriptionId = subscriptionManager.addSubscription(["test/*"], mockHandler);
        });

        it("should handle delivery errors with exponential backoff", async () => {
            const subscription = subscriptionManager["subscriptions"].get(subscriptionId)!;

            // First error
            await subscriptionManager.deliverEvent(subscription, sampleEvent);

            expect(subscription.stats.errorCount).toBe(1);
            expect(subscription.errorBackoff).toBe(100); // Initial backoff

            const metrics = subscriptionManager.getMetrics();
            expect(metrics.deliveryErrors).toBe(1);

            // Second error should increase backoff
            await subscriptionManager.deliverEvent(subscription, sampleEvent);

            expect(subscription.stats.errorCount).toBe(2);
            expect(subscription.errorBackoff).toBe(200); // Doubled
        });

        it("should disable subscription after too many errors", async () => {
            const subscription = subscriptionManager["subscriptions"].get(subscriptionId)!;

            // Force error count to be near the limit
            subscription.stats.errorCount = 99;

            await subscriptionManager.deliverEvent(subscription, sampleEvent);

            expect(subscription.isActive).toBe(false);
            expect(subscriptionManager.getMetrics().activeSubscriptions).toBe(0);
        });

        it("should reset backoff on successful delivery", async () => {
            const subscription = subscriptionManager["subscriptions"].get(subscriptionId)!;

            // Cause initial error and backoff
            await subscriptionManager.deliverEvent(subscription, sampleEvent);
            expect(subscription.errorBackoff).toBeGreaterThan(0);

            // Reset handler to succeed
            mockHandler.mockResolvedValue(undefined);

            // Successful delivery should reset backoff
            await subscriptionManager.deliverEvent(subscription, sampleEvent);
            expect(subscription.errorBackoff).toBe(0);
        });

        it("should respect maximum backoff limit", async () => {
            const subscription = subscriptionManager["subscriptions"].get(subscriptionId)!;

            // Force large backoff
            subscription.errorBackoff = 6000; // Above max

            await subscriptionManager.deliverEvent(subscription, sampleEvent);

            expect(subscription.errorBackoff).toBe(5000); // Capped at MAX_BACKOFF_MS
        });
    });

    describe("retry mechanism", () => {
        let subscriptionId: EventSubscriptionId;

        beforeEach(() => {
            subscriptionId = subscriptionManager.addSubscription(
                ["test/*"],
                mockHandler,
                { maxRetries: 3 },
            );
        });

        it("should retry failed handler calls", async () => {
            const subscription = subscriptionManager["subscriptions"].get(subscriptionId)!;

            // Fail first two attempts, succeed on third
            mockHandler
                .mockRejectedValueOnce(new Error("First failure"))
                .mockRejectedValueOnce(new Error("Second failure"))
                .mockResolvedValueOnce(undefined);

            await subscriptionManager.deliverEvent(subscription, sampleEvent);

            expect(mockHandler).toHaveBeenCalledTimes(3);
            expect(subscription.stats.errorCount).toBe(0); // Success should not increment error count
        });

        it("should fail after all retries exhausted", async () => {
            const subscription = subscriptionManager["subscriptions"].get(subscriptionId)!;

            mockHandler.mockRejectedValue(new Error("Persistent error"));

            await subscriptionManager.deliverEvent(subscription, sampleEvent);

            expect(mockHandler).toHaveBeenCalledTimes(4); // 1 + 3 retries
            expect(subscription.stats.errorCount).toBe(1);
        });

        it("should apply exponential backoff between retries", async () => {
            const subscription = subscriptionManager["subscriptions"].get(subscriptionId)!;

            const delayMock = vi.spyOn(subscriptionManager as any, "delay")
                .mockResolvedValue(undefined);

            mockHandler.mockRejectedValue(new Error("Always fails"));

            await subscriptionManager.deliverEvent(subscription, sampleEvent);

            // Should have delays: 100ms, 200ms, 400ms (exponential)
            expect(delayMock).toHaveBeenCalledTimes(3);
            expect(delayMock).toHaveBeenNthCalledWith(1, 100);
            expect(delayMock).toHaveBeenNthCalledWith(2, 200);
            expect(delayMock).toHaveBeenNthCalledWith(3, 400);

            delayMock.mockRestore();
        });

        it("should respect maximum retry backoff", async () => {
            const subscription = subscriptionManager["subscriptions"].get(subscriptionId)!;

            const delayMock = vi.spyOn(subscriptionManager as any, "delay")
                .mockResolvedValue(undefined);

            // Add more retries to test max backoff
            subscription.options.maxRetries = 10;
            mockHandler.mockRejectedValue(new Error("Always fails"));

            await subscriptionManager.deliverEvent(subscription, sampleEvent);

            // Later delays should be capped at MAX_RETRY_BACKOFF_MS (1000ms)
            const calls = delayMock.mock.calls;
            const laterCalls = calls.slice(-3); // Last 3 calls
            laterCalls.forEach(([delay]) => {
                expect(delay).toBeLessThanOrEqual(1000);
            });

            delayMock.mockRestore();
        });
    });

    describe("batch processing", () => {
        let batchHandler: jest.MockedFunction<EventHandler>;
        let subscriptionId: EventSubscriptionId;

        beforeEach(() => {
            batchHandler = vi.fn().mockResolvedValue(undefined);
            subscriptionId = subscriptionManager.addSubscription(
                ["batch/*"],
                batchHandler,
                { batchSize: 3 },
            );
        });

        it("should queue events for batch processing", async () => {
            const subscription = subscriptionManager["subscriptions"].get(subscriptionId)!;

            // Add events to queue
            await subscriptionManager.deliverEvent(subscription, sampleEvent);
            await subscriptionManager.deliverEvent(subscription, sampleEvent);

            // Handler should not be called yet (batch not full)
            expect(batchHandler).not.toHaveBeenCalled();
        });

        it("should process batch when size limit reached", async () => {
            const subscription = subscriptionManager["subscriptions"].get(subscriptionId)!;

            // Fill the batch
            await subscriptionManager.deliverEvent(subscription, sampleEvent);
            await subscriptionManager.deliverEvent(subscription, sampleEvent);
            await subscriptionManager.deliverEvent(subscription, sampleEvent);

            // Allow batch processing to complete
            await new Promise(resolve => setImmediate(resolve));

            expect(batchHandler).toHaveBeenCalledTimes(3); // Each event processed individually in batch
        });

        it("should process batch on timeout", async () => {
            const subscription = subscriptionManager["subscriptions"].get(subscriptionId)!;

            // Add only one event (batch not full)
            await subscriptionManager.deliverEvent(subscription, sampleEvent);

            // Wait for timeout to trigger flush
            await new Promise(resolve => setTimeout(resolve, 100));

            expect(batchHandler).toHaveBeenCalled();
        });

        it("should handle batch processing errors", async () => {
            const subscription = subscriptionManager["subscriptions"].get(subscriptionId)!;

            batchHandler.mockRejectedValue(new Error("Batch processing error"));

            // Fill the batch to trigger processing
            await subscriptionManager.deliverEvent(subscription, sampleEvent);
            await subscriptionManager.deliverEvent(subscription, sampleEvent);
            await subscriptionManager.deliverEvent(subscription, sampleEvent);

            // Allow batch processing to complete
            await new Promise(resolve => setImmediate(resolve));

            expect(subscription.stats.errorCount).toBeGreaterThan(0);
        });

        it("should process events in priority order", async () => {
            const subscription = subscriptionManager["subscriptions"].get(subscriptionId)!;

            const lowPriorityEvent = {
                ...sampleEvent,
                id: "low-priority",
                metadata: { priority: "low" as const },
            };

            const highPriorityEvent = {
                ...sampleEvent,
                id: "high-priority",
                metadata: { priority: "high" as const },
            };

            const criticalPriorityEvent = {
                ...sampleEvent,
                id: "critical-priority",
                metadata: { priority: "critical" as const },
            };

            // Add events in mixed order
            await subscriptionManager.deliverEvent(subscription, lowPriorityEvent);
            await subscriptionManager.deliverEvent(subscription, criticalPriorityEvent);
            await subscriptionManager.deliverEvent(subscription, highPriorityEvent);

            // Allow batch processing to complete
            await new Promise(resolve => setImmediate(resolve));

            // Events should be processed in priority order: critical, high, low
            expect(batchHandler).toHaveBeenNthCalledWith(1, criticalPriorityEvent);
            expect(batchHandler).toHaveBeenNthCalledWith(2, highPriorityEvent);
            expect(batchHandler).toHaveBeenNthCalledWith(3, lowPriorityEvent);
        });

        it("should shutdown batch processors on subscription removal", async () => {
            const subscription = subscriptionManager["subscriptions"].get(subscriptionId)!;

            // Add event to batch queue
            await subscriptionManager.deliverEvent(subscription, sampleEvent);

            // Remove subscription
            await subscriptionManager.removeSubscription(subscriptionId);

            // Any pending events should be processed during shutdown
            expect(batchHandler).toHaveBeenCalled();
        });
    });

    describe("cleanup and maintenance", () => {
        it("should clean up inactive subscriptions", async () => {
            const id1 = subscriptionManager.addSubscription(["test/*"], mockHandler);
            const id2 = subscriptionManager.addSubscription(["user/*"], mockHandler);

            // Mark first subscription as inactive and old
            const subscription1 = subscriptionManager["subscriptions"].get(id1)!;
            subscription1.isActive = false;
            subscription1.stats.lastEventTime = Date.now() - 7200000; // 2 hours ago

            // Second subscription is active
            const subscription2 = subscriptionManager["subscriptions"].get(id2)!;
            subscription2.stats.lastEventTime = Date.now();

            const cleaned = await subscriptionManager.cleanup();

            expect(cleaned).toBe(1);
            expect(subscriptionManager["subscriptions"].has(id1)).toBe(false);
            expect(subscriptionManager["subscriptions"].has(id2)).toBe(true);
        });

        it("should not clean up recently inactive subscriptions", async () => {
            const id = subscriptionManager.addSubscription(["test/*"], mockHandler);

            // Mark as inactive but recent
            const subscription = subscriptionManager["subscriptions"].get(id)!;
            subscription.isActive = false;
            subscription.stats.lastEventTime = Date.now() - 1800000; // 30 minutes ago

            const cleaned = await subscriptionManager.cleanup();

            expect(cleaned).toBe(0);
            expect(subscriptionManager["subscriptions"].has(id)).toBe(true);
        });

        it("should provide comprehensive metrics", () => {
            subscriptionManager.addSubscription(["test/*"], mockHandler);
            subscriptionManager.addSubscription(["user/*"], mockHandler);

            const metrics = subscriptionManager.getMetrics();

            expect(metrics).toEqual({
                totalSubscriptions: 2,
                activeSubscriptions: 2,
                eventsDelivered: 0,
                deliveryErrors: 0,
            });
        });

        it("should update metrics on event delivery", async () => {
            const id = subscriptionManager.addSubscription(["test/*"], mockHandler);
            const subscription = subscriptionManager["subscriptions"].get(id)!;

            await subscriptionManager.deliverEvent(subscription, sampleEvent);

            const metrics = subscriptionManager.getMetrics();
            expect(metrics.eventsDelivered).toBe(1);
        });

        it("should update metrics on errors", async () => {
            mockHandler.mockRejectedValue(new Error("Test error"));

            const id = subscriptionManager.addSubscription(["test/*"], mockHandler);
            const subscription = subscriptionManager["subscriptions"].get(id)!;

            await subscriptionManager.deliverEvent(subscription, sampleEvent);

            const metrics = subscriptionManager.getMetrics();
            expect(metrics.deliveryErrors).toBe(1);
        });
    });

    describe("utility methods", () => {
        it("should generate unique subscription IDs", () => {
            const id1 = subscriptionManager.addSubscription(["test/*"], mockHandler);
            const id2 = subscriptionManager.addSubscription(["test/*"], mockHandler);

            expect(id1).not.toBe(id2);
            expect(id1).toMatch(/^sub_\d+_[a-z0-9]{7}$/);
            expect(id2).toMatch(/^sub_\d+_[a-z0-9]{7}$/);
        });

        it("should calculate event priority correctly", () => {
            const calculatePriority = subscriptionManager["calculateEventPriority"].bind(subscriptionManager);

            expect(calculatePriority({ ...sampleEvent, metadata: { priority: "critical" } })).toBe(1000);
            expect(calculatePriority({ ...sampleEvent, metadata: { priority: "high" } })).toBe(100);
            expect(calculatePriority({ ...sampleEvent, metadata: { priority: "medium" } })).toBe(10);
            expect(calculatePriority({ ...sampleEvent, metadata: { priority: "low" } })).toBe(1);
            expect(calculatePriority(sampleEvent)).toBe(10); // Default medium
        });

        it("should return empty array for unknown patterns", () => {
            const subscriptions = subscriptionManager.findSubscriptionsByPattern("unknown/*");
            expect(subscriptions).toEqual([]);
        });

        it("should return empty array for handler with no subscriptions", () => {
            const newHandler = vi.fn();
            const subscriptions = subscriptionManager.getSubscriptionsByHandler(newHandler);
            expect(subscriptions).toEqual([]);
        });

        it("should handle subscription stats for non-existent subscription", () => {
            const stats = subscriptionManager.getSubscriptionStats("non-existent" as EventSubscriptionId);
            expect(stats).toBeUndefined();
        });
    });

    describe("index management", () => {
        it("should maintain pattern index integrity", async () => {
            const id1 = subscriptionManager.addSubscription(["user/*", "chat/*"], mockHandler);
            const id2 = subscriptionManager.addSubscription(["user/*"], mockHandler);

            // Verify patterns are indexed
            expect(subscriptionManager.findSubscriptionsByPattern("user/*")).toHaveLength(2);
            expect(subscriptionManager.findSubscriptionsByPattern("chat/*")).toHaveLength(1);

            // Remove one subscription
            await subscriptionManager.removeSubscription(id1);

            // Verify index is updated
            expect(subscriptionManager.findSubscriptionsByPattern("user/*")).toHaveLength(1);
            expect(subscriptionManager.findSubscriptionsByPattern("chat/*")).toHaveLength(0);
        });

        it("should maintain handler index integrity", async () => {
            const handler1 = vi.fn();
            const handler2 = vi.fn();

            const id1 = subscriptionManager.addSubscription(["test/*"], handler1);
            const id2 = subscriptionManager.addSubscription(["user/*"], handler1);
            const id3 = subscriptionManager.addSubscription(["chat/*"], handler2);

            expect(subscriptionManager.getSubscriptionsByHandler(handler1)).toHaveLength(2);
            expect(subscriptionManager.getSubscriptionsByHandler(handler2)).toHaveLength(1);

            // Remove subscription
            await subscriptionManager.removeSubscription(id1);

            expect(subscriptionManager.getSubscriptionsByHandler(handler1)).toHaveLength(1);
            expect(subscriptionManager.getSubscriptionsByHandler(handler2)).toHaveLength(1);
        });

        it("should clean up empty pattern entries", async () => {
            const id = subscriptionManager.addSubscription(["unique/pattern"], mockHandler);

            // Verify pattern exists
            expect(subscriptionManager["patternIndex"].has("unique/pattern")).toBe(true);

            // Remove subscription
            await subscriptionManager.removeSubscription(id);

            // Verify pattern is cleaned up
            expect(subscriptionManager["patternIndex"].has("unique/pattern")).toBe(false);
        });
    });
});
