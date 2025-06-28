import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import Redis from "ioredis";
import { ContextSubscriptionManager } from "./ContextSubscriptionManager.js";
import type { ContextUpdateEvent, SubscriptionFilter, NotificationBatch } from "./ContextSubscriptionManager.js";
import { logger } from "../../../logger.js";

describe("ContextSubscriptionManager", () => {
    let subscriptionManager: ContextSubscriptionManager;
    let redis: Redis;
    let mockLogger: typeof logger;

    beforeEach(async () => {
        // Create test Redis instance (will use testcontainers in real implementation)
        redis = new Redis({
            host: process.env.REDIS_HOST || "localhost",
            port: parseInt(process.env.REDIS_PORT || "6379"),
            db: 15, // Use separate test database
        });

        mockLogger = {
            ...logger,
            info: vi.fn(),
            debug: vi.fn(),
            warn: vi.fn(),
            error: vi.fn(),
        };

        subscriptionManager = new ContextSubscriptionManager(redis, mockLogger);
        await subscriptionManager.initialize();
    });

    afterEach(async () => {
        await redis.flushdb(); // Clean test database
        await subscriptionManager.shutdown();
        await redis.quit();
    });

    describe("Subscription Management", () => {
        it("should create and manage subscriptions", async () => {
            const swarmId = "test-swarm-001";
            let receivedEvent: ContextUpdateEvent | null = null;

            const subscription = await subscriptionManager.subscribe(
                swarmId,
                (event) => {
                    receivedEvent = event;
                },
            );

            expect(subscription.id).toBeTruthy();
            expect(subscription.swarmId).toBe(swarmId);
            expect(subscription.isActive).toBe(true);

            // Simulate context update event
            const updateEvent: ContextUpdateEvent = {
                swarmId,
                changeType: "update",
                changedPaths: ["configuration.goal"],
                timestamp: new Date(),
                newVersion: 2,
                previousVersion: 1,
                changes: {
                    "configuration.goal": {
                        oldValue: "old goal",
                        newValue: "new goal",
                    },
                },
            };

            await subscriptionManager.publishUpdate(updateEvent);

            // Wait for async notification
            await new Promise(resolve => setTimeout(resolve, 50));

            expect(receivedEvent).toBeTruthy();
            expect(receivedEvent!.swarmId).toBe(swarmId);
            expect(receivedEvent!.changeType).toBe("update");

            // Cleanup
            subscription.unsubscribe();
        });

        it("should support multiple subscribers for same swarm", async () => {
            const swarmId = "multi-subscriber-test";
            const updates: ContextUpdateEvent[] = [];

            const sub1 = await subscriptionManager.subscribe(swarmId, (event) => 
                updates.push({ ...event, source: "sub1" as any }),
            );
            const sub2 = await subscriptionManager.subscribe(swarmId, (event) => 
                updates.push({ ...event, source: "sub2" as any }),
            );

            const updateEvent: ContextUpdateEvent = {
                swarmId,
                changeType: "resource_allocation",
                changedPaths: ["resources.allocated"],
                timestamp: new Date(),
                newVersion: 2,
                previousVersion: 1,
                changes: {
                    "resources.allocated": {
                        oldValue: [],
                        newValue: [{ allocationId: "test-allocation" }],
                    },
                },
            };

            await subscriptionManager.publishUpdate(updateEvent);
            await new Promise(resolve => setTimeout(resolve, 50));

            expect(updates).toHaveLength(2);
            expect(updates[0].swarmId).toBe(swarmId);
            expect(updates[1].swarmId).toBe(swarmId);

            sub1.unsubscribe();
            sub2.unsubscribe();
        });

        it("should handle subscription cleanup on unsubscribe", async () => {
            const swarmId = "cleanup-test";
            let notificationCount = 0;

            const subscription = await subscriptionManager.subscribe(swarmId, () => {
                notificationCount++;
            });

            // First update should be received
            await subscriptionManager.publishUpdate({
                swarmId,
                changeType: "update",
                changedPaths: ["test"],
                timestamp: new Date(),
                newVersion: 2,
                previousVersion: 1,
                changes: {},
            });
            await new Promise(resolve => setTimeout(resolve, 50));

            expect(notificationCount).toBe(1);

            // Unsubscribe
            subscription.unsubscribe();

            // Second update should not be received
            await subscriptionManager.publishUpdate({
                swarmId,
                changeType: "update",
                changedPaths: ["test2"],
                timestamp: new Date(),
                newVersion: 3,
                previousVersion: 2,
                changes: {},
            });
            await new Promise(resolve => setTimeout(resolve, 50));

            expect(notificationCount).toBe(1); // Should still be 1
        });
    });

    describe("Filtered Subscriptions", () => {
        it("should filter notifications by path patterns", async () => {
            const swarmId = "filter-test-swarm";
            const configUpdates: ContextUpdateEvent[] = [];
            const resourceUpdates: ContextUpdateEvent[] = [];

            const configFilter: SubscriptionFilter = {
                pathPatterns: ["configuration.*"],
                changeTypes: ["update"],
                emergentOnly: false,
            };

            const resourceFilter: SubscriptionFilter = {
                pathPatterns: ["resources.*"],
                changeTypes: ["resource_allocation", "resource_deallocation"],
                emergentOnly: false,
            };

            const configSub = await subscriptionManager.subscribe(
                swarmId,
                (event) => configUpdates.push(event),
                configFilter,
            );

            const resourceSub = await subscriptionManager.subscribe(
                swarmId,
                (event) => resourceUpdates.push(event),
                resourceFilter,
            );

            // Send configuration update
            await subscriptionManager.publishUpdate({
                swarmId,
                changeType: "update",
                changedPaths: ["configuration.goal"],
                timestamp: new Date(),
                newVersion: 2,
                previousVersion: 1,
                changes: {},
            });

            // Send resource update
            await subscriptionManager.publishUpdate({
                swarmId,
                changeType: "resource_allocation",
                changedPaths: ["resources.allocated"],
                timestamp: new Date(),
                newVersion: 3,
                previousVersion: 2,
                changes: {},
            });

            await new Promise(resolve => setTimeout(resolve, 50));

            // Config subscriber should only receive config updates
            expect(configUpdates).toHaveLength(1);
            expect(configUpdates[0].changedPaths).toContain("configuration.goal");

            // Resource subscriber should only receive resource updates
            expect(resourceUpdates).toHaveLength(1);
            expect(resourceUpdates[0].changedPaths).toContain("resources.allocated");

            configSub.unsubscribe();
            resourceSub.unsubscribe();
        });

        it("should filter notifications by change types", async () => {
            const swarmId = "change-type-filter-test";
            const updateEvents: ContextUpdateEvent[] = [];
            const allocationEvents: ContextUpdateEvent[] = [];

            const updateFilter: SubscriptionFilter = {
                pathPatterns: ["*"],
                changeTypes: ["update"],
                emergentOnly: false,
            };

            const allocationFilter: SubscriptionFilter = {
                pathPatterns: ["*"],
                changeTypes: ["resource_allocation"],
                emergentOnly: false,
            };

            const updateSub = await subscriptionManager.subscribe(
                swarmId,
                (event) => updateEvents.push(event),
                updateFilter,
            );

            const allocationSub = await subscriptionManager.subscribe(
                swarmId,
                (event) => allocationEvents.push(event),
                allocationFilter,
            );

            // Send various change types
            await subscriptionManager.publishUpdate({
                swarmId,
                changeType: "update",
                changedPaths: ["blackboard"],
                timestamp: new Date(),
                newVersion: 2,
                previousVersion: 1,
                changes: {},
            });

            await subscriptionManager.publishUpdate({
                swarmId,
                changeType: "resource_allocation",
                changedPaths: ["resources.allocated"],
                timestamp: new Date(),
                newVersion: 3,
                previousVersion: 2,
                changes: {},
            });

            await subscriptionManager.publishUpdate({
                swarmId,
                changeType: "created",
                changedPaths: ["swarmId"],
                timestamp: new Date(),
                newVersion: 1,
                previousVersion: 0,
                changes: {},
            });

            await new Promise(resolve => setTimeout(resolve, 50));

            // Update subscriber should only receive update events
            expect(updateEvents).toHaveLength(1);
            expect(updateEvents[0].changeType).toBe("update");

            // Allocation subscriber should only receive allocation events
            expect(allocationEvents).toHaveLength(1);
            expect(allocationEvents[0].changeType).toBe("resource_allocation");

            updateSub.unsubscribe();
            allocationSub.unsubscribe();
        });

        it("should filter emergent capability events", async () => {
            const swarmId = "emergent-filter-test";
            const emergentEvents: ContextUpdateEvent[] = [];
            const allEvents: ContextUpdateEvent[] = [];

            const emergentFilter: SubscriptionFilter = {
                pathPatterns: ["*"],
                changeTypes: ["update", "resource_allocation"],
                emergentOnly: true,
            };

            const allFilter: SubscriptionFilter = {
                pathPatterns: ["*"],
                changeTypes: ["update", "resource_allocation"],
                emergentOnly: false,
            };

            const emergentSub = await subscriptionManager.subscribe(
                swarmId,
                (event) => emergentEvents.push(event),
                emergentFilter,
            );

            const allSub = await subscriptionManager.subscribe(
                swarmId,
                (event) => allEvents.push(event),
                allFilter,
            );

            // Send emergent capability update
            await subscriptionManager.publishUpdate({
                swarmId,
                changeType: "update",
                changedPaths: ["configuration.features.emergentCapabilitiesEnabled"],
                timestamp: new Date(),
                newVersion: 2,
                previousVersion: 1,
                changes: {},
                emergentCapability: true,
            });

            // Send regular update
            await subscriptionManager.publishUpdate({
                swarmId,
                changeType: "update",
                changedPaths: ["configuration.goal"],
                timestamp: new Date(),
                newVersion: 3,
                previousVersion: 2,
                changes: {},
            });

            await new Promise(resolve => setTimeout(resolve, 50));

            // Emergent subscriber should only receive emergent events
            expect(emergentEvents).toHaveLength(1);
            expect(emergentEvents[0].emergentCapability).toBe(true);

            // All events subscriber should receive both
            expect(allEvents).toHaveLength(2);

            emergentSub.unsubscribe();
            allSub.unsubscribe();
        });
    });

    describe("Notification Batching", () => {
        it("should batch high-frequency notifications", async () => {
            const swarmId = "batching-test-swarm";
            const batches: NotificationBatch[] = [];

            const subscription = await subscriptionManager.subscribeToBatches(
                swarmId,
                (batch) => batches.push(batch),
                { batchSize: 3, batchTimeoutMs: 100 },
            );

            // Send multiple rapid updates
            for (let i = 0; i < 5; i++) {
                await subscriptionManager.publishUpdate({
                    swarmId,
                    changeType: "update",
                    changedPaths: [`test.${i}`],
                    timestamp: new Date(),
                    newVersion: i + 2,
                    previousVersion: i + 1,
                    changes: {},
                });
            }

            // Wait for batching
            await new Promise(resolve => setTimeout(resolve, 150));

            expect(batches.length).toBeGreaterThan(0);
            
            // Should receive batched events
            const totalEvents = batches.reduce((sum, batch) => sum + batch.events.length, 0);
            expect(totalEvents).toBe(5);

            subscription.unsubscribe();
        });

        it("should respect batch size limits", async () => {
            const swarmId = "batch-size-test";
            const batches: NotificationBatch[] = [];

            const subscription = await subscriptionManager.subscribeToBatches(
                swarmId,
                (batch) => batches.push(batch),
                { batchSize: 2, batchTimeoutMs: 1000 }, // Large timeout, small batch size
            );

            // Send exactly 4 updates (should create 2 batches of 2)
            for (let i = 0; i < 4; i++) {
                await subscriptionManager.publishUpdate({
                    swarmId,
                    changeType: "update",
                    changedPaths: [`batch.${i}`],
                    timestamp: new Date(),
                    newVersion: i + 2,
                    previousVersion: i + 1,
                    changes: {},
                });
            }

            // Wait for batching
            await new Promise(resolve => setTimeout(resolve, 50));

            expect(batches).toHaveLength(2);
            expect(batches[0].events).toHaveLength(2);
            expect(batches[1].events).toHaveLength(2);

            subscription.unsubscribe();
        });

        it("should flush batches on timeout", async () => {
            const swarmId = "batch-timeout-test";
            const batches: NotificationBatch[] = [];

            const subscription = await subscriptionManager.subscribeToBatches(
                swarmId,
                (batch) => batches.push(batch),
                { batchSize: 10, batchTimeoutMs: 50 }, // Large batch size, small timeout
            );

            // Send fewer updates than batch size
            await subscriptionManager.publishUpdate({
                swarmId,
                changeType: "update",
                changedPaths: ["timeout.test"],
                timestamp: new Date(),
                newVersion: 2,
                previousVersion: 1,
                changes: {},
            });

            // Wait for timeout
            await new Promise(resolve => setTimeout(resolve, 75));

            expect(batches).toHaveLength(1);
            expect(batches[0].events).toHaveLength(1);

            subscription.unsubscribe();
        });
    });

    describe("Rate Limiting", () => {
        it("should enforce rate limits on subscriptions", async () => {
            const swarmId = "rate-limit-test";
            let notificationCount = 0;
            let rateLimitHit = false;

            const subscription = await subscriptionManager.subscribe(
                swarmId,
                (event) => notificationCount++,
                {
                    pathPatterns: ["*"],
                    changeTypes: ["update"],
                    emergentOnly: false,
                },
                {
                    maxNotificationsPerSecond: 2,
                    burstAllowance: 1,
                },
            );

            // Send burst of updates (should hit rate limit)
            for (let i = 0; i < 5; i++) {
                try {
                    await subscriptionManager.publishUpdate({
                        swarmId,
                        changeType: "update",
                        changedPaths: [`burst.${i}`],
                        timestamp: new Date(),
                        newVersion: i + 2,
                        previousVersion: i + 1,
                        changes: {},
                    });
                } catch (error) {
                    if (error.message.includes("rate limit")) {
                        rateLimitHit = true;
                    }
                }
            }

            await new Promise(resolve => setTimeout(resolve, 50));

            // Should not receive all notifications due to rate limiting
            expect(notificationCount).toBeLessThan(5);
            
            subscription.unsubscribe();
        });

        it("should allow burst within burst allowance", async () => {
            const swarmId = "burst-allowance-test";
            let notificationCount = 0;

            const subscription = await subscriptionManager.subscribe(
                swarmId,
                (event) => notificationCount++,
                {
                    pathPatterns: ["*"],
                    changeTypes: ["update"],
                    emergentOnly: false,
                },
                {
                    maxNotificationsPerSecond: 1,
                    burstAllowance: 3,
                },
            );

            // Send burst within allowance
            for (let i = 0; i < 3; i++) {
                await subscriptionManager.publishUpdate({
                    swarmId,
                    changeType: "update",
                    changedPaths: [`allowance.${i}`],
                    timestamp: new Date(),
                    newVersion: i + 2,
                    previousVersion: i + 1,
                    changes: {},
                });
            }

            await new Promise(resolve => setTimeout(resolve, 50));

            // Should receive all notifications within burst allowance
            expect(notificationCount).toBe(3);

            subscription.unsubscribe();
        });
    });

    describe("Health Monitoring", () => {
        it("should track subscription health metrics", async () => {
            const swarmId = "health-test-swarm";
            
            const subscription = await subscriptionManager.subscribe(swarmId, () => {});

            // Generate some activity
            await subscriptionManager.publishUpdate({
                swarmId,
                changeType: "update",
                changedPaths: ["health.test"],
                timestamp: new Date(),
                newVersion: 2,
                previousVersion: 1,
                changes: {},
            });

            await new Promise(resolve => setTimeout(resolve, 50));

            const health = subscriptionManager.getHealth();

            expect(health.status).toBe("healthy");
            expect(health.activeSubscriptions).toBeGreaterThan(0);
            expect(health.totalNotificationsSent).toBeGreaterThan(0);
            expect(typeof health.averageNotificationLatencyMs).toBe("number");

            subscription.unsubscribe();
        });

        it("should detect unhealthy subscriptions and clean them up", async () => {
            const swarmId = "unhealthy-test";
            
            // Create subscription that will fail
            const subscription = await subscriptionManager.subscribe(swarmId, () => {
                throw new Error("Simulated subscription failure");
            });

            // Send update that will cause failure
            await subscriptionManager.publishUpdate({
                swarmId,
                changeType: "update",
                changedPaths: ["failure.test"],
                timestamp: new Date(),
                newVersion: 2,
                previousVersion: 1,
                changes: {},
            });

            await new Promise(resolve => setTimeout(resolve, 50));

            const health = subscriptionManager.getHealth();
            expect(health.failedNotifications).toBeGreaterThan(0);

            // Cleanup
            subscription.unsubscribe();
        });

        it("should provide performance metrics", () => {
            const metrics = subscriptionManager.getMetrics();

            expect(metrics).toHaveProperty("subscriptionOperations");
            expect(metrics).toHaveProperty("notificationLatency");
            expect(metrics).toHaveProperty("batchingMetrics");
            expect(metrics).toHaveProperty("rateLimitingMetrics");

            expect(typeof metrics.subscriptionOperations.creates).toBe("number");
            expect(typeof metrics.notificationLatency.average).toBe("number");
        });
    });

    describe("Cross-Instance Coordination", () => {
        it("should handle Redis pub/sub for cross-instance notifications", async () => {
            const swarmId = "cross-instance-test";
            let receivedEvent: ContextUpdateEvent | null = null;

            const subscription = await subscriptionManager.subscribe(swarmId, (event) => {
                receivedEvent = event;
            });

            // Simulate external Redis pub/sub message
            const externalMessage = JSON.stringify({
                swarmId,
                changeType: "update",
                changedPaths: ["external.update"],
                timestamp: new Date().toISOString(),
                newVersion: 2,
                previousVersion: 1,
                changes: {},
            });

            // Simulate receiving message from another instance
            await redis.publish(`swarm-context-updates:${swarmId}`, externalMessage);

            await new Promise(resolve => setTimeout(resolve, 50));

            expect(receivedEvent).toBeTruthy();
            expect(receivedEvent!.swarmId).toBe(swarmId);
            expect(receivedEvent!.changedPaths).toContain("external.update");

            subscription.unsubscribe();
        });

        it("should handle Redis connection failures gracefully", async () => {
            const swarmId = "connection-failure-test";
            let notificationReceived = false;

            const subscription = await subscriptionManager.subscribe(swarmId, () => {
                notificationReceived = true;
            });

            // Simulate Redis connection failure by disconnecting
            await redis.disconnect();

            // Try to publish update (should handle gracefully)
            await expect(subscriptionManager.publishUpdate({
                swarmId,
                changeType: "update",
                changedPaths: ["connection.test"],
                timestamp: new Date(),
                newVersion: 2,
                previousVersion: 1,
                changes: {},
            })).not.toThrow();

            subscription.unsubscribe();
        });
    });
});
