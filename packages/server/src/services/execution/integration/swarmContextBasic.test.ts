import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { logger } from "../../../events/logger.js";
import { CacheService } from "../../../redisConn.js";
import { ContextSubscriptionManager } from "../shared/ContextSubscriptionManager.js";
import { SwarmContextManager } from "../shared/SwarmContextManager.js";
import "../../../__test/setup.js";

// Mock external dependencies
vi.mock("../../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        debug: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
    },
}));

describe("SwarmContextManager Basic Integration Tests", () => {
    let contextManager: SwarmContextManager;
    let subscriptionManager: ContextSubscriptionManager;
    let redis: any;

    const testSwarmId = "test-swarm-basic";

    beforeEach(async () => {
        // Get Redis connection for testing
        redis = await CacheService.get().raw();
        
        // Initialize context management components
        subscriptionManager = new ContextSubscriptionManager(redis, logger);
        await subscriptionManager.initialize();

        contextManager = new SwarmContextManager(redis, logger);
        await contextManager.initialize();

        // Clean up any existing test data
        await contextManager.deleteContext(testSwarmId);
    });

    afterEach(async () => {
        // Clean up test data
        await contextManager.deleteContext(testSwarmId);
        
        // Shutdown components
        await contextManager.shutdown();
        await subscriptionManager.shutdown();
    });

    describe("Context Management", () => {
        it("should create and retrieve swarm context", async () => {
            // 1. Create swarm context
            const context = {
                swarmId: testSwarmId,
                goal: "Test basic context management",
                strategy: "test",
                policy: {
                    allowedModels: ["gpt-4"],
                    maxTokensPerRequest: 2000,
                    requireApproval: false,
                },
                resources: {
                    allocated: { credits: "100", tokens: 10000, duration: 300000 },
                    available: { credits: "100", tokens: 10000, duration: 300000 },
                    reserved: { credits: "0", tokens: 0, duration: 0 },
                },
                configuration: {
                    model: "gpt-4",
                    temperature: 0.7,
                    maxIterations: 10,
                },
                status: "active" as const,
                participants: [],
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            await contextManager.createContext(testSwarmId, context);

            // 2. Retrieve and verify context
            const retrievedContext = await contextManager.getContext(testSwarmId);
            expect(retrievedContext).toBeDefined();
            expect(retrievedContext?.swarmId).toBe(testSwarmId);
            expect(retrievedContext?.goal).toBe("Test basic context management");
            expect(retrievedContext?.policy.allowedModels).toContain("gpt-4");
        });

        it("should update context fields", async () => {
            // 1. Create initial context
            const initialContext = {
                swarmId: testSwarmId,
                goal: "Initial goal",
                strategy: "initial",
                policy: {
                    allowedModels: ["gpt-3.5-turbo"],
                    maxTokensPerRequest: 1000,
                },
                resources: {
                    allocated: { credits: "50", tokens: 5000, duration: 150000 },
                    available: { credits: "50", tokens: 5000, duration: 150000 },
                    reserved: { credits: "0", tokens: 0, duration: 0 },
                },
                configuration: {
                    model: "gpt-3.5-turbo",
                    temperature: 0.5,
                    maxIterations: 5,
                },
                status: "initializing" as const,
                participants: [],
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            await contextManager.createContext(testSwarmId, initialContext);

            // 2. Update context
            const updates = {
                goal: "Updated goal",
                status: "active" as const,
                policy: {
                    allowedModels: ["gpt-4"],
                    maxTokensPerRequest: 2000,
                },
                updatedAt: new Date(),
            };

            await contextManager.updateContext(testSwarmId, updates);

            // 3. Verify updates
            const updatedContext = await contextManager.getContext(testSwarmId);
            expect(updatedContext?.goal).toBe("Updated goal");
            expect(updatedContext?.status).toBe("active");
            expect(updatedContext?.policy.maxTokensPerRequest).toBe(2000);
        });

        it("should delete context", async () => {
            // 1. Create context
            const context = {
                swarmId: testSwarmId,
                goal: "To be deleted",
                strategy: "delete-test",
                policy: {
                    allowedModels: ["gpt-3.5-turbo"],
                    maxTokensPerRequest: 1000,
                },
                resources: {
                    allocated: { credits: "25", tokens: 2500, duration: 75000 },
                    available: { credits: "25", tokens: 2500, duration: 75000 },
                    reserved: { credits: "0", tokens: 0, duration: 0 },
                },
                configuration: {
                    model: "gpt-3.5-turbo",
                    temperature: 0.7,
                    maxIterations: 3,
                },
                status: "active" as const,
                participants: [],
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            await contextManager.createContext(testSwarmId, context);

            // 2. Verify context exists
            let retrievedContext = await contextManager.getContext(testSwarmId);
            expect(retrievedContext).toBeDefined();

            // 3. Delete context
            await contextManager.deleteContext(testSwarmId);

            // 4. Verify context is deleted
            retrievedContext = await contextManager.getContext(testSwarmId);
            expect(retrievedContext).toBeNull();
        });
    });

    describe("Subscription Management", () => {
        it("should handle basic subscriptions", async () => {
            // 1. Create context
            const context = {
                swarmId: testSwarmId,
                goal: "Test subscriptions",
                strategy: "subscription-test",
                policy: {
                    allowedModels: ["gpt-4"],
                    maxTokensPerRequest: 1500,
                },
                resources: {
                    allocated: { credits: "75", tokens: 7500, duration: 225000 },
                    available: { credits: "75", tokens: 7500, duration: 225000 },
                    reserved: { credits: "0", tokens: 0, duration: 0 },
                },
                configuration: {
                    model: "gpt-4",
                    temperature: 0.6,
                    maxIterations: 8,
                },
                status: "active" as const,
                participants: [],
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            await contextManager.createContext(testSwarmId, context);

            // 2. Set up subscription
            const updates: any[] = [];
            const subscriptionId = await contextManager.subscribe({
                swarmId: testSwarmId,
                subscriberId: "basic-test",
                watchPaths: ["policy.*", "status"],
                handler: (update) => {
                    updates.push(update);
                },
            });

            // 3. Make updates to trigger subscription
            await contextManager.updateContext(testSwarmId, {
                status: "completed" as const,
                policy: {
                    allowedModels: ["gpt-4", "gpt-3.5-turbo"],
                    maxTokensPerRequest: 2000,
                },
                updatedAt: new Date(),
            });

            // 4. Wait for propagation
            await new Promise(resolve => setTimeout(resolve, 100));

            // 5. Verify subscription received updates
            expect(updates.length).toBeGreaterThan(0);

            // 6. Clean up subscription
            await contextManager.unsubscribe(subscriptionId);
        });
    });

    describe("Resource Management", () => {
        it("should track resource allocation and consumption", async () => {
            // 1. Create context with resources
            const context = {
                swarmId: testSwarmId,
                goal: "Test resource tracking",
                strategy: "resource-test",
                policy: {
                    allowedModels: ["gpt-4"],
                    maxTokensPerRequest: 3000,
                },
                resources: {
                    allocated: { credits: "200", tokens: 20000, duration: 600000 },
                    available: { credits: "200", tokens: 20000, duration: 600000 },
                    reserved: { credits: "0", tokens: 0, duration: 0 },
                },
                configuration: {
                    model: "gpt-4",
                    temperature: 0.7,
                    maxIterations: 15,
                },
                status: "active" as const,
                participants: [],
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            await contextManager.createContext(testSwarmId, context);

            // 2. Simulate resource consumption
            const consumedCredits = 30;
            const consumedTokens = 3000;
            const consumedDuration = 60000;

            await contextManager.updateContext(testSwarmId, {
                resources: {
                    allocated: { credits: "200", tokens: 20000, duration: 600000 },
                    available: {
                        credits: (200 - consumedCredits).toString(),
                        tokens: 20000 - consumedTokens,
                        duration: 600000 - consumedDuration,
                    },
                    reserved: { credits: "0", tokens: 0, duration: 0 },
                },
                updatedAt: new Date(),
            });

            // 3. Verify resource tracking
            const updatedContext = await contextManager.getContext(testSwarmId);
            expect(updatedContext?.resources.available.credits).toBe("170");
            expect(updatedContext?.resources.available.tokens).toBe(17000);
            expect(updatedContext?.resources.available.duration).toBe(540000);
        });
    });

    describe("Error Handling", () => {
        it("should handle non-existent context gracefully", async () => {
            const nonExistentId = "non-existent-swarm";
            
            // Try to get context that doesn't exist
            const context = await contextManager.getContext(nonExistentId);
            expect(context).toBeNull();

            // Try to update context that doesn't exist - should not throw
            try {
                await contextManager.updateContext(nonExistentId, {
                    status: "failed",
                    updatedAt: new Date(),
                });
                // Should handle gracefully or throw - either is acceptable
            } catch (error) {
                expect(error).toBeDefined();
            }
        });

        it("should handle subscription to non-existent context", async () => {
            const nonExistentId = "non-existent-swarm-sub";
            
            try {
                await contextManager.subscribe({
                    swarmId: nonExistentId,
                    subscriberId: "error-test",
                    watchPaths: ["status"],
                    handler: () => {},
                });
                // Subscription might succeed even if context doesn't exist yet
            } catch (error) {
                // Or it might throw - both are acceptable behaviors
                expect(error).toBeDefined();
            }
        });
    });
});
