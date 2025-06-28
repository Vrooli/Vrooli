import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import Redis from "ioredis";
import { SwarmContextManager } from "./SwarmContextManager.js";
import type { UnifiedSwarmContext, ResourcePool, ContextUpdateEvent } from "./UnifiedSwarmContext.js";
import { logger } from "../../../logger.js";

describe("SwarmContextManager", () => {
    let contextManager: SwarmContextManager;
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

        contextManager = new SwarmContextManager(redis, mockLogger);
        await contextManager.initialize();
    });

    afterEach(async () => {
        await redis.flushdb(); // Clean test database
        await contextManager.shutdown();
        await redis.quit();
    });

    describe("Context Lifecycle Management", () => {
        it("should create a new unified swarm context with default values", async () => {
            const swarmId = "test-swarm-001";
            const initialConfig = {
                goal: "Test swarm coordination",
                resourcePool: {
                    credits: BigInt(10000),
                    timeoutMs: 300000,
                    memoryMB: 1024,
                    concurrentExecutions: 5,
                },
            };

            const context = await contextManager.createContext(swarmId, initialConfig);

            expect(context.swarmId).toBe(swarmId);
            expect(context.version).toBe(1);
            expect(context.configuration.emergentCapabilitiesEnabled).toBe(true);
            expect(context.resources.total.credits).toBe(BigInt(10000));
            expect(context.resources.available.credits).toBe(BigInt(10000));
            expect(context.resources.allocated).toHaveLength(0);
        });

        it("should retrieve existing context by swarmId", async () => {
            const swarmId = "test-swarm-002";
            const initialConfig = { goal: "Retrieve test" };

            const created = await contextManager.createContext(swarmId, initialConfig);
            const retrieved = await contextManager.getContext(swarmId);

            expect(retrieved.swarmId).toBe(created.swarmId);
            expect(retrieved.version).toBe(created.version);
            expect(retrieved.configuration.goal).toBe("Retrieve test");
        });

        it("should update context with version increment", async () => {
            const swarmId = "test-swarm-003";
            const context = await contextManager.createContext(swarmId, { goal: "Original goal" });

            const updates = {
                configuration: {
                    ...context.configuration,
                    goal: "Updated goal",
                },
            };

            await contextManager.updateContext(swarmId, updates);
            const updated = await contextManager.getContext(swarmId);

            expect(updated.version).toBe(2);
            expect(updated.configuration.goal).toBe("Updated goal");
            expect(updated.lastUpdated.getTime()).toBeGreaterThan(context.lastUpdated.getTime());
        });

        it("should delete context and cleanup resources", async () => {
            const swarmId = "test-swarm-004";
            await contextManager.createContext(swarmId, { goal: "To be deleted" });

            await contextManager.deleteContext(swarmId);

            await expect(contextManager.getContext(swarmId)).rejects.toThrow(
                `Swarm context not found: ${swarmId}`,
            );
        });
    });

    describe("Resource Management", () => {
        let context: UnifiedSwarmContext;
        const swarmId = "resource-test-swarm";

        beforeEach(async () => {
            context = await contextManager.createContext(swarmId, {
                goal: "Resource management test",
                resourcePool: {
                    credits: BigInt(5000),
                    timeoutMs: 60000,
                    memoryMB: 512,
                    concurrentExecutions: 3,
                },
            });
        });

        it("should allocate resources hierarchically", async () => {
            const request = {
                credits: BigInt(1000),
                timeoutMs: 10000,
                memoryMB: 128,
                concurrentExecutions: 1,
                priority: "medium" as const,
                purpose: "test-routine-execution",
            };

            const allocation = await contextManager.allocateResources(swarmId, request);

            expect(allocation.allocated.credits).toBe(BigInt(1000));
            expect(allocation.allocated.timeoutMs).toBe(10000);
            expect(allocation.allocated.memoryMB).toBe(128);
            expect(allocation.allocated.concurrentExecutions).toBe(1);

            // Verify context updated
            const updated = await contextManager.getContext(swarmId);
            expect(updated.resources.available.credits).toBe(BigInt(4000));
            expect(updated.resources.allocated).toHaveLength(1);
        });

        it("should prevent overallocation of resources", async () => {
            const request = {
                credits: BigInt(10000), // More than available (5000)
                timeoutMs: 10000,
                memoryMB: 128,
                concurrentExecutions: 1,
                priority: "high" as const,
                purpose: "excessive-request",
            };

            await expect(contextManager.allocateResources(swarmId, request)).rejects.toThrow(
                "Insufficient credits",
            );
        });

        it("should release resources and return to available pool", async () => {
            // Allocate resources first
            const allocation = await contextManager.allocateResources(swarmId, {
                credits: BigInt(2000),
                timeoutMs: 15000,
                memoryMB: 256,
                concurrentExecutions: 2,
                priority: "low" as const,
                purpose: "temporary-allocation",
            });

            // Release partial resources
            await contextManager.releaseResources(swarmId, {
                ...allocation,
                allocated: {
                    credits: BigInt(500), // Return 500 credits
                    timeoutMs: 5000,
                    memoryMB: 64,
                    concurrentExecutions: 1,
                },
            });

            const updated = await contextManager.getContext(swarmId);
            expect(updated.resources.available.credits).toBe(BigInt(3500)); // 5000 - 2000 + 500
        });

        it("should track multiple concurrent allocations", async () => {
            const allocation1 = await contextManager.allocateResources(swarmId, {
                credits: BigInt(1000),
                timeoutMs: 10000,
                memoryMB: 128,
                concurrentExecutions: 1,
                priority: "medium" as const,
                purpose: "allocation-1",
            });

            const allocation2 = await contextManager.allocateResources(swarmId, {
                credits: BigInt(1500),
                timeoutMs: 15000,
                memoryMB: 192,
                concurrentExecutions: 1,
                priority: "high" as const,
                purpose: "allocation-2",
            });

            const updated = await contextManager.getContext(swarmId);
            expect(updated.resources.allocated).toHaveLength(2);
            expect(updated.resources.available.credits).toBe(BigInt(2500)); // 5000 - 1000 - 1500
        });
    });

    describe("Live Update Subscriptions", () => {
        it("should notify subscribers of context updates", async () => {
            const swarmId = "subscription-test-swarm";
            const context = await contextManager.createContext(swarmId, { goal: "Subscription test" });

            let receivedUpdate: ContextUpdateEvent | null = null;
            const subscription = await contextManager.subscribe(swarmId, (update) => {
                receivedUpdate = update;
            });

            // Update context to trigger notification
            const updates = {
                configuration: {
                    ...context.configuration,
                    goal: "Updated via subscription",
                },
            };

            await contextManager.updateContext(swarmId, updates);

            // Wait for async notification
            await new Promise(resolve => setTimeout(resolve, 50));

            expect(receivedUpdate).toBeTruthy();
            expect(receivedUpdate!.swarmId).toBe(swarmId);
            expect(receivedUpdate!.changeType).toBe("update");
            expect(receivedUpdate!.changedPaths).toContain("configuration.goal");

            // Cleanup
            subscription.unsubscribe();
        });

        it("should handle multiple subscribers for same swarm", async () => {
            const swarmId = "multi-subscriber-test";
            await contextManager.createContext(swarmId, { goal: "Multi-subscriber test" });

            const updates: ContextUpdateEvent[] = [];
            
            const sub1 = await contextManager.subscribe(swarmId, (update) => updates.push(update));
            const sub2 = await contextManager.subscribe(swarmId, (update) => updates.push(update));

            await contextManager.updateContext(swarmId, {
                blackboard: new Map([["test-key", "test-value"]]),
            });

            await new Promise(resolve => setTimeout(resolve, 50));

            expect(updates).toHaveLength(2); // Both subscribers notified
            expect(updates[0].swarmId).toBe(swarmId);
            expect(updates[1].swarmId).toBe(swarmId);

            sub1.unsubscribe();
            sub2.unsubscribe();
        });
    });

    describe("Context Validation", () => {
        it("should validate context integrity", async () => {
            const swarmId = "validation-test-swarm";
            const context = await contextManager.createContext(swarmId, {
                goal: "Validation test",
                resourcePool: {
                    credits: BigInt(1000),
                    timeoutMs: 30000,
                    memoryMB: 256,
                    concurrentExecutions: 2,
                },
            });

            const result = await contextManager.validateContext(context);

            expect(result.isValid).toBe(true);
            expect(result.errors).toHaveLength(0);
            expect(result.warnings).toHaveLength(0);
        });

        it("should detect invalid resource allocations", async () => {
            const invalidContext: UnifiedSwarmContext = {
                swarmId: "invalid-test",
                version: 1,
                lastUpdated: new Date(),
                resources: {
                    total: {
                        credits: BigInt(1000),
                        timeoutMs: 30000,
                        memoryMB: 256,
                        concurrentExecutions: 2,
                    },
                    available: {
                        credits: BigInt(-500), // Invalid: negative available
                        timeoutMs: 30000,
                        memoryMB: 256,
                        concurrentExecutions: 2,
                    },
                    allocated: [],
                },
                policy: {
                    security: {
                        requireApproval: false,
                        allowedTools: [],
                        maxRiskLevel: "medium",
                        dataClassification: "internal",
                    },
                    resource: {
                        allocation: {
                            strategy: "proportional",
                            reservePercentage: 10,
                            maxAllocationPerRequest: 50,
                        },
                        limits: {
                            maxConcurrentRuns: 5,
                            maxQueueSize: 20,
                            timeoutMs: 300000,
                        },
                    },
                    organizational: {
                        teamStructure: "flat",
                        decisionMaking: "consensus",
                        communicationPattern: "broadcast",
                    },
                },
                configuration: {
                    goal: "Invalid context test",
                    timeouts: {
                        stepTimeoutMs: 30000,
                        routineTimeoutMs: 300000,
                        swarmTimeoutMs: 3600000,
                    },
                    retries: {
                        maxAttempts: 3,
                        backoffMs: 1000,
                        backoffMultiplier: 2,
                    },
                    features: {
                        emergentCapabilitiesEnabled: true,
                        learningEnabled: true,
                        optimizationEnabled: true,
                    },
                },
                blackboard: new Map(),
                executionState: {
                    currentStatus: "running",
                    startedAt: new Date(),
                    teams: [],
                    agents: [],
                    activeRuns: [],
                    metrics: {
                        totalExecutions: 0,
                        successfulExecutions: 0,
                        failedExecutions: 0,
                        averageExecutionTimeMs: 0,
                        resourceUtilization: 0,
                    },
                },
            };

            const result = await contextManager.validateContext(invalidContext);

            expect(result.isValid).toBe(false);
            expect(result.errors.length).toBeGreaterThan(0);
            expect(result.errors.some(error => error.includes("negative available credits"))).toBe(true);
        });
    });

    describe("Health Monitoring", () => {
        it("should provide health status", () => {
            const health = contextManager.getHealth();

            expect(health.status).toBe("healthy");
            expect(health.activeContexts).toBe(0);
            expect(health.activeSubscriptions).toBe(0);
            expect(typeof health.memoryUsageMB).toBe("number");
        });

        it("should track metrics for context operations", async () => {
            const swarmId = "metrics-test-swarm";
            
            // Perform operations to generate metrics
            await contextManager.createContext(swarmId, { goal: "Metrics test" });
            await contextManager.getContext(swarmId);
            await contextManager.updateContext(swarmId, { 
                blackboard: new Map([["metric", "test"]]), 
            });

            const metrics = contextManager.getMetrics();

            expect(metrics.contextOperations.creates).toBe(1);
            expect(metrics.contextOperations.reads).toBeGreaterThanOrEqual(1);
            expect(metrics.contextOperations.updates).toBe(1);
        });
    });

    describe("Distributed Synchronization", () => {
        it("should acquire and release distributed locks", async () => {
            const swarmId = "lock-test-swarm";
            const resource = "test-resource";

            const lock = await contextManager.acquireLock(swarmId, resource);

            expect(lock.acquired).toBe(true);
            expect(lock.lockId).toBeTruthy();

            await lock.release();
            
            // Verify lock is released by acquiring it again
            const lock2 = await contextManager.acquireLock(swarmId, resource);
            expect(lock2.acquired).toBe(true);
            await lock2.release();
        });

        it("should create and manage barriers for coordination", async () => {
            const swarmId = "barrier-test-swarm";
            const barrierName = "test-barrier";
            const participantCount = 3;

            const barrier = await contextManager.createBarrier(swarmId, barrierName, participantCount);

            expect(barrier.name).toBe(barrierName);
            expect(barrier.expectedCount).toBe(participantCount);
            expect(barrier.currentCount).toBe(0);

            // Simulate participants arriving at barrier
            await barrier.arrive();
            expect(barrier.currentCount).toBe(1);

            await barrier.arrive();
            expect(barrier.currentCount).toBe(2);

            // Last participant should trigger barrier release
            await barrier.arrive();
            expect(barrier.isReleased).toBe(true);
        });
    });
});
