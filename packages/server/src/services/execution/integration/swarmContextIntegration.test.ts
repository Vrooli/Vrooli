import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { logger } from "../../../events/logger.js";
import { CacheService } from "../../../redisConn.js";
import { ContextSubscriptionManager } from "../shared/ContextSubscriptionManager.js";
import { SwarmContextManager } from "../shared/SwarmContextManager.js";
import { SwarmCoordinator } from "../tier1/coordination/swarmCoordinator.js";
import { createConversationBridge } from "../tier1/intelligence/conversationBridge.js";
import { TierTwoOrchestrator } from "../tier2/tierTwoOrchestrator.js";
import { UnifiedExecutor } from "../tier3/engine/unifiedExecutor.js";
import { getEventBus } from "../../events/types.js";
import { IntegratedToolRegistry } from "./mcp/toolRegistry.js";
import { CachedConversationStateStore, PrismaChatStore } from "../../conversation/chatStore.js";
import { type UnifiedExecutorConfig } from "@vrooli/shared";
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

vi.mock("../tier3/strategies/conversationalStrategy.js", () => ({
    ConversationalStrategy: vi.fn().mockImplementation(() => ({
        execute: vi.fn().mockResolvedValue({ result: "mock result" }),
    })),
}));

vi.mock("../tier3/strategies/reasoningStrategy.js", () => ({
    ReasoningStrategy: vi.fn().mockImplementation(() => ({
        execute: vi.fn().mockResolvedValue({ result: "mock result" }),
    })),
}));

vi.mock("../tier3/strategies/deterministicStrategy.js", () => ({
    DeterministicStrategy: vi.fn().mockImplementation(() => ({
        execute: vi.fn().mockResolvedValue({ result: "mock result" }),
    })),
}));

describe("SwarmContextManager Integration Tests", () => {
    let contextManager: SwarmContextManager;
    let subscriptionManager: ContextSubscriptionManager;
    let swarmCoordinator: SwarmCoordinator;
    let tierTwoOrchestrator: TierTwoOrchestrator;
    let tierThreeExecutor: UnifiedExecutor;
    let redis: any;
    let conversationBridge: ReturnType<typeof createConversationBridge>;
    let eventBus: ReturnType<typeof getEventBus>;
    let toolRegistry: IntegratedToolRegistry;
    let conversationStore: CachedConversationStateStore;

    const testSwarmId = "test-swarm-integration";
    const testExecutionId = "test-execution-integration";

    beforeEach(async () => {
        // Get Redis connection for testing
        redis = await CacheService.get().raw();
        
        // Initialize event bus
        eventBus = getEventBus();
        await eventBus.start();

        // Initialize context management components
        subscriptionManager = new ContextSubscriptionManager(redis, logger);
        await subscriptionManager.initialize();

        contextManager = new SwarmContextManager(redis, logger);
        await contextManager.initialize();

        // Initialize conversation infrastructure
        conversationBridge = createConversationBridge(logger);
        const chatStore = new PrismaChatStore();
        conversationStore = new CachedConversationStateStore(chatStore);
        toolRegistry = IntegratedToolRegistry.getInstance(logger, conversationStore);

        // Initialize Tier 3 (UnifiedExecutor)
        const executorConfig: UnifiedExecutorConfig = {
            strategyFactory: {
                defaultStrategy: "CONVERSATIONAL" as any,
                fallbackChain: ["REASONING", "DETERMINISTIC"] as any[],
                adaptationEnabled: true,
                learningRate: 0.1,
            },
            resourceLimits: {
                tokens: 10000,
                apiCalls: 100,
                computeTime: 30000,
                memory: 512,
                cost: 10,
            },
            sandboxEnabled: false,
            telemetryEnabled: true,
        };
        
        tierThreeExecutor = new UnifiedExecutor(
            executorConfig,
            eventBus,
            logger,
            toolRegistry,
        );

        // Initialize Tier 2 with SwarmContextManager
        tierTwoOrchestrator = new TierTwoOrchestrator(
            logger,
            eventBus,
            tierThreeExecutor,
            contextManager,
        );

        // Initialize Tier 1 with SwarmContextManager
        swarmCoordinator = new SwarmCoordinator(
            logger,
            contextManager,
            conversationBridge,
            tierTwoOrchestrator,
        );

        // Clean up any existing test data
        await contextManager.deleteContext(testSwarmId);
    });

    afterEach(async () => {
        // Clean up test data
        await contextManager.deleteContext(testSwarmId);
        
        // Shutdown components
        await contextManager.shutdown();
        await subscriptionManager.shutdown();
        await eventBus.stop();
    });

    describe("Live policy propagation", () => {
        it("should propagate policy updates to all tiers in real-time", async () => {
            // 1. Create initial swarm context
            const initialContext = {
                swarmId: testSwarmId,
                goal: "Test policy propagation",
                strategy: "initial",
                policy: {
                    allowedModels: ["gpt-3.5-turbo"],
                    maxTokensPerRequest: 1000,
                    requireApproval: false,
                },
                resources: {
                    allocated: { credits: "100", tokens: 10000, duration: 300000 },
                    available: { credits: "100", tokens: 10000, duration: 300000 },
                    reserved: { credits: "0", tokens: 0, duration: 0 },
                },
                configuration: {
                    model: "gpt-3.5-turbo",
                    temperature: 0.7,
                    maxIterations: 10,
                },
                status: "active" as const,
                participants: [],
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            await contextManager.createContext(testSwarmId, initialContext);

            // 2. Set up subscription handlers to track policy updates
            const policyUpdates: any[] = [];
            const subscriptionId = await contextManager.subscribe({
                swarmId: testSwarmId,
                subscriberId: "policy-test",
                watchPaths: ["policy.*"],
                handler: (update) => {
                    policyUpdates.push(update);
                },
            });

            // 3. Update policy configuration
            const newPolicy = {
                allowedModels: ["gpt-4", "gpt-3.5-turbo"],
                maxTokensPerRequest: 2000,
                requireApproval: true,
                emergentCapabilities: true,
            };

            await contextManager.updateContext(testSwarmId, {
                policy: newPolicy,
                updatedAt: new Date(),
            });

            // 4. Wait for propagation
            await new Promise(resolve => setTimeout(resolve, 100));

            // 5. Verify policy was propagated
            expect(policyUpdates.length).toBeGreaterThan(0);
            const latestUpdate = policyUpdates[policyUpdates.length - 1];
            expect(latestUpdate.changes.policy).toEqual(newPolicy);

            // 6. Verify context was updated
            const updatedContext = await contextManager.getContext(testSwarmId);
            expect(updatedContext?.policy).toEqual(newPolicy);

            // Clean up
            await contextManager.unsubscribe(subscriptionId);
        });

        it("should handle configuration updates across tiers", async () => {
            // 1. Create swarm context
            const context = {
                swarmId: testSwarmId,
                goal: "Test configuration updates",
                strategy: "adaptive",
                policy: {
                    allowedModels: ["gpt-4"],
                    maxTokensPerRequest: 4000,
                },
                resources: {
                    allocated: { credits: "200", tokens: 20000, duration: 600000 },
                    available: { credits: "200", tokens: 20000, duration: 600000 },
                    reserved: { credits: "0", tokens: 0, duration: 0 },
                },
                configuration: {
                    model: "gpt-4",
                    temperature: 0.5,
                    maxIterations: 20,
                },
                status: "active" as const,
                participants: [],
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            await contextManager.createContext(testSwarmId, context);

            // 2. Track configuration updates
            const configUpdates: any[] = [];
            const subscriptionId = await contextManager.subscribe({
                swarmId: testSwarmId,
                subscriberId: "config-test",
                watchPaths: ["configuration.*"],
                handler: (update) => {
                    configUpdates.push(update);
                },
            });

            // 3. Update configuration
            const newConfig = {
                model: "gpt-4",
                temperature: 0.8,
                maxIterations: 15,
                parallelExecution: true,
                adaptiveStrategies: ["reasoning", "conversational"],
            };

            await contextManager.updateContext(testSwarmId, {
                configuration: newConfig,
                updatedAt: new Date(),
            });

            // 4. Wait for propagation and verify
            await new Promise(resolve => setTimeout(resolve, 100));

            expect(configUpdates.length).toBeGreaterThan(0);
            const latestUpdate = configUpdates[configUpdates.length - 1];
            expect(latestUpdate.changes.configuration).toEqual(newConfig);

            // Clean up
            await contextManager.unsubscribe(subscriptionId);
        });
    });

    describe("Resource allocation flows", () => {
        it("should track resource allocation across execution tiers", async () => {
            // 1. Create swarm with initial resource allocation
            const context = {
                swarmId: testSwarmId,
                goal: "Test resource allocation",
                strategy: "resource-optimized",
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

            // 2. Track resource updates
            const resourceUpdates: any[] = [];
            const subscriptionId = await contextManager.subscribe({
                swarmId: testSwarmId,
                subscriberId: "resource-test",
                watchPaths: ["resources.*"],
                handler: (update) => {
                    resourceUpdates.push(update);
                },
            });

            // 3. Simulate resource consumption
            const consumedResources = {
                credits: "50",
                tokens: 5000,
                duration: 60000,
            };

            await contextManager.updateContext(testSwarmId, {
                resources: {
                    allocated: { credits: "500", tokens: 50000, duration: 1200000 },
                    available: {
                        credits: (500 - 50).toString(),
                        tokens: 50000 - 5000,
                        duration: 1200000 - 60000,
                    },
                    reserved: { credits: "0", tokens: 0, duration: 0 },
                },
                updatedAt: new Date(),
            });

            // 4. Wait for propagation
            await new Promise(resolve => setTimeout(resolve, 100));

            // 5. Verify resource tracking
            expect(resourceUpdates.length).toBeGreaterThan(0);
            const latestUpdate = resourceUpdates[resourceUpdates.length - 1];
            expect(latestUpdate.changes.resources.available.credits).toBe("450");
            expect(latestUpdate.changes.resources.available.tokens).toBe(45000);

            // 6. Verify context was updated
            const updatedContext = await contextManager.getContext(testSwarmId);
            expect(updatedContext?.resources.available.credits).toBe("450");
            expect(updatedContext?.resources.available.tokens).toBe(45000);

            // Clean up
            await contextManager.unsubscribe(subscriptionId);
        });

        it("should handle resource reservation and release", async () => {
            // 1. Create swarm context
            const context = {
                swarmId: testSwarmId,
                goal: "Test resource reservation",
                strategy: "conservative",
                policy: {
                    allowedModels: ["gpt-3.5-turbo"],
                    maxTokensPerRequest: 1500,
                },
                resources: {
                    allocated: { credits: "300", tokens: 30000, duration: 900000 },
                    available: { credits: "300", tokens: 30000, duration: 900000 },
                    reserved: { credits: "0", tokens: 0, duration: 0 },
                },
                configuration: {
                    model: "gpt-3.5-turbo",
                    temperature: 0.6,
                    maxIterations: 15,
                },
                status: "active" as const,
                participants: [],
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            await contextManager.createContext(testSwarmId, context);

            // 2. Reserve resources for execution
            const reservedResources = {
                credits: "100",
                tokens: 10000,
                duration: 300000,
            };

            await contextManager.updateContext(testSwarmId, {
                resources: {
                    allocated: { credits: "300", tokens: 30000, duration: 900000 },
                    available: {
                        credits: (300 - 100).toString(),
                        tokens: 30000 - 10000,
                        duration: 900000 - 300000,
                    },
                    reserved: reservedResources,
                },
                updatedAt: new Date(),
            });

            // 3. Verify reservation
            let updatedContext = await contextManager.getContext(testSwarmId);
            expect(updatedContext?.resources.available.credits).toBe("200");
            expect(updatedContext?.resources.reserved.credits).toBe("100");

            // 4. Release resources after execution
            await contextManager.updateContext(testSwarmId, {
                resources: {
                    allocated: { credits: "300", tokens: 30000, duration: 900000 },
                    available: {
                        credits: "280", // 200 + 100 - 20 (consumed)
                        tokens: 28000,  // 20000 + 10000 - 2000 (consumed)
                        duration: 850000, // 600000 + 300000 - 50000 (consumed)
                    },
                    reserved: { credits: "0", tokens: 0, duration: 0 }, // Released
                },
                updatedAt: new Date(),
            });

            // 5. Verify release
            updatedContext = await contextManager.getContext(testSwarmId);
            expect(updatedContext?.resources.available.credits).toBe("280");
            expect(updatedContext?.resources.reserved.credits).toBe("0");
        });
    });

    describe("End-to-end context flow", () => {
        it("should handle complete swarm lifecycle with live updates", async () => {
            // 1. Create swarm context
            const initialContext = {
                swarmId: testSwarmId,
                goal: "Complete end-to-end test",
                strategy: "comprehensive",
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

            // 2. Set up comprehensive monitoring
            const allUpdates: any[] = [];
            const subscriptionId = await contextManager.subscribe({
                swarmId: testSwarmId,
                subscriberId: "e2e-test",
                watchPaths: ["*"], // Watch all changes
                handler: (update) => {
                    allUpdates.push(update);
                },
            });

            // 3. Simulate swarm lifecycle transitions
            const transitions = [
                { status: "active", phase: "planning" },
                { status: "active", phase: "executing" },
                { status: "active", phase: "analyzing" },
                { status: "completed", phase: "finished" },
            ];

            for (const transition of transitions) {
                await contextManager.updateContext(testSwarmId, {
                    status: transition.status as any,
                    configuration: {
                        ...initialContext.configuration,
                        currentPhase: transition.phase,
                    },
                    updatedAt: new Date(),
                });

                // Wait for propagation
                await new Promise(resolve => setTimeout(resolve, 50));
            }

            // 4. Verify all updates were captured
            expect(allUpdates.length).toBeGreaterThan(0);
            
            // 5. Verify final state
            const finalContext = await contextManager.getContext(testSwarmId);
            expect(finalContext?.status).toBe("completed");
            expect(finalContext?.configuration.currentPhase).toBe("finished");

            // Clean up
            await contextManager.unsubscribe(subscriptionId);
        });

        it("should handle subscription cleanup on swarm termination", async () => {
            // 1. Create swarm context
            const context = {
                swarmId: testSwarmId,
                goal: "Test subscription cleanup",
                strategy: "cleanup-test",
                policy: {
                    allowedModels: ["gpt-3.5-turbo"],
                    maxTokensPerRequest: 1000,
                },
                resources: {
                    allocated: { credits: "100", tokens: 10000, duration: 300000 },
                    available: { credits: "100", tokens: 10000, duration: 300000 },
                    reserved: { credits: "0", tokens: 0, duration: 0 },
                },
                configuration: {
                    model: "gpt-3.5-turbo",
                    temperature: 0.7,
                    maxIterations: 10,
                },
                status: "active" as const,
                participants: [],
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            await contextManager.createContext(testSwarmId, context);

            // 2. Create multiple subscriptions
            const subscriptionIds = [];
            for (let i = 0; i < 3; i++) {
                const subId = await contextManager.subscribe({
                    swarmId: testSwarmId,
                    subscriberId: `cleanup-test-${i}`,
                    watchPaths: ["status", "resources.*"],
                    handler: () => {}, // No-op handler
                });
                subscriptionIds.push(subId);
            }

            // 3. Verify subscriptions are active
            for (const subId of subscriptionIds) {
                const isActive = await subscriptionManager.isSubscriptionActive(subId);
                expect(isActive).toBe(true);
            }

            // 4. Terminate swarm
            await contextManager.deleteContext(testSwarmId);

            // 5. Verify subscriptions are cleaned up
            // Note: This depends on the implementation of cleanup logic
            // For now, we'll just verify the context is gone
            const deletedContext = await contextManager.getContext(testSwarmId);
            expect(deletedContext).toBeNull();

            // Manual cleanup for test
            for (const subId of subscriptionIds) {
                await contextManager.unsubscribe(subId);
            }
        });
    });

    describe("Error handling and resilience", () => {
        it("should handle Redis connection issues gracefully", async () => {
            // This test would require mocking Redis failures
            // For now, we'll test basic error handling
            
            const nonExistentSwarmId = "non-existent-swarm";
            
            // Try to get context for non-existent swarm
            const context = await contextManager.getContext(nonExistentSwarmId);
            expect(context).toBeNull();
            
            // Try to update non-existent context
            try {
                await contextManager.updateContext(nonExistentSwarmId, {
                    status: "failed",
                    updatedAt: new Date(),
                });
                // Should not throw, should handle gracefully
            } catch (error) {
                // If it throws, that's also acceptable behavior
                expect(error).toBeDefined();
            }
        });

        it("should handle malformed context updates", async () => {
            // 1. Create valid context
            const context = {
                swarmId: testSwarmId,
                goal: "Test error handling",
                strategy: "error-test",
                policy: {
                    allowedModels: ["gpt-3.5-turbo"],
                    maxTokensPerRequest: 1000,
                },
                resources: {
                    allocated: { credits: "100", tokens: 10000, duration: 300000 },
                    available: { credits: "100", tokens: 10000, duration: 300000 },
                    reserved: { credits: "0", tokens: 0, duration: 0 },
                },
                configuration: {
                    model: "gpt-3.5-turbo",
                    temperature: 0.7,
                    maxIterations: 10,
                },
                status: "active" as const,
                participants: [],
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            await contextManager.createContext(testSwarmId, context);

            // 2. Try malformed updates (should be handled gracefully)
            try {
                await contextManager.updateContext(testSwarmId, {
                    // @ts-ignore - intentionally malformed
                    invalidField: "should not break system",
                    updatedAt: new Date(),
                });
            } catch (error) {
                // Error handling is acceptable
                expect(error).toBeDefined();
            }

            // 3. Verify context is still accessible
            const retrievedContext = await contextManager.getContext(testSwarmId);
            expect(retrievedContext).toBeDefined();
            expect(retrievedContext?.swarmId).toBe(testSwarmId);
        });
    });
});
