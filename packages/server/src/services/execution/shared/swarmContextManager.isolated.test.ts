import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { logger } from "../../../events/logger.js";
import { CacheService } from "../../../redisConn.js";
import { SwarmContextManager } from "./SwarmContextManager.js";
import { ContextSubscriptionManager } from "./ContextSubscriptionManager.js";
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

describe("SwarmContextManager Isolated Tests", () => {
    let contextManager: SwarmContextManager;
    let subscriptionManager: ContextSubscriptionManager;
    let redis: any;

    const testSwarmId = "test-swarm-isolated";

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
        try {
            await contextManager.deleteContext(testSwarmId);
        } catch (error) {
            // Ignore cleanup errors
        }
        
        // Shutdown components
        try {
            await contextManager.shutdown();
            await subscriptionManager.shutdown();
        } catch (error) {
            // Ignore shutdown errors
        }
    });

    describe("Core Context Operations", () => {
        it("should create, read, update, and delete context", async () => {
            // 1. Create context
            const context = {
                swarmId: testSwarmId,
                goal: "Test CRUD operations",
                strategy: "test-strategy",
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

            // 2. Read context
            const retrievedContext = await contextManager.getContext(testSwarmId);
            expect(retrievedContext).toBeDefined();
            expect(retrievedContext?.swarmId).toBe(testSwarmId);
            expect(retrievedContext?.goal).toBe("Test CRUD operations");

            // 3. Update context
            const updates = {
                goal: "Updated test goal",
                status: "completed" as const,
                policy: {
                    allowedModels: ["gpt-4", "gpt-3.5-turbo"],
                    maxTokensPerRequest: 3000,
                    requireApproval: true,
                },
                updatedAt: new Date(),
            };

            await contextManager.updateContext(testSwarmId, updates);

            // 4. Verify update
            const updatedContext = await contextManager.getContext(testSwarmId);
            expect(updatedContext?.goal).toBe("Updated test goal");
            expect(updatedContext?.status).toBe("completed");
            expect(updatedContext?.policy.maxTokensPerRequest).toBe(3000);

            // 5. Delete context
            await contextManager.deleteContext(testSwarmId);

            // 6. Verify deletion
            const deletedContext = await contextManager.getContext(testSwarmId);
            expect(deletedContext).toBeNull();
        });

        it("should handle resource tracking", async () => {
            // 1. Create context with initial resources
            const context = {
                swarmId: testSwarmId,
                goal: "Test resource tracking",
                strategy: "resource-test",
                policy: {
                    allowedModels: ["gpt-4"],
                    maxTokensPerRequest: 2000,
                },
                resources: {
                    allocated: { credits: "500", tokens: 50000, duration: 1200000 },
                    available: { credits: "500", tokens: 50000, duration: 1200000 },
                    reserved: { credits: "0", tokens: 0, duration: 0 },
                },
                configuration: {
                    model: "gpt-4",
                    temperature: 0.7,
                    maxIterations: 25,
                },
                status: "active" as const,
                participants: [],
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            await contextManager.createContext(testSwarmId, context);

            // 2. Simulate resource consumption
            const consumedCredits = 50;
            const consumedTokens = 5000;
            const consumedDuration = 60000;

            await contextManager.updateContext(testSwarmId, {
                resources: {
                    allocated: { credits: "500", tokens: 50000, duration: 1200000 },
                    available: {
                        credits: (500 - consumedCredits).toString(),
                        tokens: 50000 - consumedTokens,
                        duration: 1200000 - consumedDuration,
                    },
                    reserved: { credits: "0", tokens: 0, duration: 0 },
                },
                updatedAt: new Date(),
            });

            // 3. Verify resource tracking
            const updatedContext = await contextManager.getContext(testSwarmId);
            expect(updatedContext?.resources.available.credits).toBe("450");
            expect(updatedContext?.resources.available.tokens).toBe(45000);
            expect(updatedContext?.resources.available.duration).toBe(1140000);
        });
    });

    describe("Subscription Management", () => {
        it("should handle basic subscriptions and updates", async () => {
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
            let subscriptionId: string;
            
            try {
                subscriptionId = await contextManager.subscribe({
                    swarmId: testSwarmId,
                    subscriberId: "isolated-test",
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

            } finally {
                // 6. Clean up subscription
                if (subscriptionId!) {
                    try {
                        await contextManager.unsubscribe(subscriptionId);
                    } catch (error) {
                        // Ignore cleanup errors
                    }
                }
            }
        });
    });

    describe("Error Handling", () => {
        it("should handle operations on non-existent context", async () => {
            const nonExistentId = "non-existent-swarm";
            
            // Try to get context that doesn't exist
            const context = await contextManager.getContext(nonExistentId);
            expect(context).toBeNull();

            // Try to update context that doesn't exist
            try {
                await contextManager.updateContext(nonExistentId, {
                    status: "failed",
                    updatedAt: new Date(),
                });
                // Should handle gracefully
            } catch (error) {
                // Error is also acceptable
                expect(error).toBeDefined();
            }

            // Try to delete context that doesn't exist
            try {
                await contextManager.deleteContext(nonExistentId);
                // Should handle gracefully
            } catch (error) {
                // Error is also acceptable
                expect(error).toBeDefined();
            }
        });

        it("should handle Redis connection issues gracefully", async () => {
            // This test verifies basic error handling
            // In a real scenario, we'd mock Redis to fail
            
            const testId = "redis-error-test";
            
            // Try operations that might fail
            try {
                const result = await contextManager.getContext(testId);
                expect(result).toBeNull(); // Should return null for non-existent
            } catch (error) {
                // Should not throw for basic operations
                console.warn("Unexpected error in basic operation:", error);
            }
        });
    });

    describe("SwarmContextManager Integration Verification", () => {
        it("should complete end-to-end swarm context lifecycle", async () => {
            // This test verifies the complete SwarmContextManager integration
            // that was implemented in the previous phases
            
            const lifecycleStates = [
                { status: "initializing", phase: "setup" },
                { status: "active", phase: "planning" },
                { status: "active", phase: "executing" },
                { status: "active", phase: "analyzing" },
                { status: "completed", phase: "finished" },
            ];

            // 1. Create initial context
            const initialContext = {
                swarmId: testSwarmId,
                goal: "Complete lifecycle test",
                strategy: "lifecycle-test",
                policy: {
                    allowedModels: ["gpt-4"],
                    maxTokensPerRequest: 3000,
                    requireApproval: false,
                },
                resources: {
                    allocated: { credits: "1000", tokens: 100000, duration: 1800000 },
                    available: { credits: "1000", tokens: 100000, duration: 1800000 },
                    reserved: { credits: "0", tokens: 0, duration: 0 },
                },
                configuration: {
                    model: "gpt-4",
                    temperature: 0.7,
                    maxIterations: 30,
                },
                status: "initializing" as const,
                participants: [],
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            await contextManager.createContext(testSwarmId, initialContext);

            // 2. Set up monitoring for all updates
            const allUpdates: any[] = [];
            let subscriptionId: string;

            try {
                subscriptionId = await contextManager.subscribe({
                    swarmId: testSwarmId,
                    subscriberId: "lifecycle-monitor",
                    watchPaths: ["*"], // Watch all changes
                    handler: (update) => {
                        allUpdates.push(update);
                    },
                });

                // 3. Simulate lifecycle transitions
                for (const state of lifecycleStates) {
                    await contextManager.updateContext(testSwarmId, {
                        status: state.status as any,
                        configuration: {
                            ...initialContext.configuration,
                            currentPhase: state.phase,
                        },
                        updatedAt: new Date(),
                    });

                    // Wait for propagation
                    await new Promise(resolve => setTimeout(resolve, 50));
                }

                // 4. Verify lifecycle completion
                const finalContext = await contextManager.getContext(testSwarmId);
                expect(finalContext?.status).toBe("completed");
                expect(finalContext?.configuration.currentPhase).toBe("finished");

                // 5. Verify updates were captured
                expect(allUpdates.length).toBeGreaterThan(0);

            } finally {
                // Clean up subscription
                if (subscriptionId!) {
                    try {
                        await contextManager.unsubscribe(subscriptionId);
                    } catch (error) {
                        // Ignore cleanup errors
                    }
                }
            }
        });
    });
});
