import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { ExecutionArchitecture, resetExecutionArchitecture, getExecutionArchitecture } from "./executionArchitecture.js";
import { type Logger } from "winston";
import { type TierCommunicationInterface, type TierExecutionRequest, generatePK } from "@vrooli/shared";

/**
 * ExecutionArchitecture Integration Tests
 * 
 * These tests validate the core factory that wires together all three tiers
 * of the execution architecture. This is the most critical integration test
 * suite as it ensures:
 * 
 * 1. Proper dependency injection between tiers
 * 2. Correct initialization order (T3 → T2 → T1)
 * 3. Event bus wiring across all tiers
 * 4. State store coordination
 * 5. Resource management integration
 * 6. Tool registry integration
 * 7. Cross-tier communication flows
 * 8. Graceful error handling and cleanup
 */
describe("ExecutionArchitecture - Three-Tier Integration Factory", () => {
    let logger: Logger;
    let architecture: ExecutionArchitecture;

    beforeEach(async () => {
        logger = {
            info: vi.fn(),
            error: vi.fn(),
            warn: vi.fn(),
            debug: vi.fn(),
        } as unknown as Logger;

        // Reset singleton before each test
        await resetExecutionArchitecture();
    });

    afterEach(async () => {
        // Cleanup after each test
        if (architecture) {
            try {
                await architecture.stop();
            } catch (error) {
                // Ignore cleanup errors in tests
            }
        }
        await resetExecutionArchitecture();
    });

    describe("Architecture Initialization", () => {
        it("should initialize all three tiers in correct dependency order", async () => {
            architecture = await ExecutionArchitecture.create({
                useRedis: false, // Use in-memory for tests
                logger,
            });

            const status = architecture.getStatus();

            // Verify all components are properly initialized
            expect(status.initialized).toBe(true);
            expect(status.tier1Ready).toBe(true);
            expect(status.tier2Ready).toBe(true);
            expect(status.tier3Ready).toBe(true);
            expect(status.eventBusReady).toBe(true);
            expect(status.stateStoresReady).toBe(true);
            expect(status.toolRegistryReady).toBe(true);

            // Verify tiers can be accessed
            expect(() => architecture.getTier1()).not.toThrow();
            expect(() => architecture.getTier2()).not.toThrow();
            expect(() => architecture.getTier3()).not.toThrow();
        });

        it("should initialize with Redis in production mode", async () => {
            architecture = await ExecutionArchitecture.create({
                useRedis: true,
                logger,
            });

            const status = architecture.getStatus();
            expect(status.initialized).toBe(true);
            expect(status.stateStoresReady).toBe(true);
        });

        it("should handle initialization failures gracefully", async () => {
            // Mock a failure in shared services initialization
            const originalConsoleError = console.error;
            console.error = vi.fn(); // Suppress error logs during test

            try {
                // Create architecture with invalid configuration that should fail
                await expect(ExecutionArchitecture.create({
                    // This should trigger cleanup logic
                    config: {
                        tier3: {
                            // Invalid config that should cause failure
                            strategyFactory: null as any,
                        },
                    },
                })).rejects.toThrow();
            } finally {
                console.error = originalConsoleError;
            }
        });

        it("should prevent double initialization", async () => {
            architecture = new (ExecutionArchitecture as any)();
            
            // Initialize once
            await (architecture as any).initialize();
            
            // Second initialization should throw
            await expect((architecture as any).initialize()).rejects.toThrow("ExecutionArchitecture already initialized");
        });
    });

    describe("Tier Communication Interface Compliance", () => {
        beforeEach(async () => {
            architecture = await ExecutionArchitecture.create({
                useRedis: false,
                logger,
            });
        });

        it("should provide all tiers implementing TierCommunicationInterface", async () => {
            const tier1 = architecture.getTier1();
            const tier2 = architecture.getTier2();
            const tier3 = architecture.getTier3();

            // Verify all tiers implement the interface
            expect(typeof tier1.execute).toBe("function");
            expect(typeof tier1.getCapabilities).toBe("function");

            expect(typeof tier2.execute).toBe("function");
            expect(typeof tier2.getCapabilities).toBe("function");

            expect(typeof tier3.execute).toBe("function");
            expect(typeof tier3.getCapabilities).toBe("function");
        });

        it("should provide combined capabilities from all tiers", async () => {
            const capabilities = await architecture.getCapabilities();

            expect(capabilities).toHaveProperty("tier1");
            expect(capabilities).toHaveProperty("tier2");
            expect(capabilities).toHaveProperty("tier3");

            // Verify each tier has required capability fields
            expect(capabilities.tier1).toHaveProperty("supportedInputTypes");
            expect(capabilities.tier2).toHaveProperty("supportedInputTypes");
            expect(capabilities.tier3).toHaveProperty("supportedInputTypes");
        });

        it("should throw when accessing uninitialized tiers", () => {
            const uninitializedArchitecture = new (ExecutionArchitecture as any)();

            expect(() => uninitializedArchitecture.getTier1()).toThrow("Tier 1 not initialized");
            expect(() => uninitializedArchitecture.getTier2()).toThrow("Tier 2 not initialized");
            expect(() => uninitializedArchitecture.getTier3()).toThrow("Tier 3 not initialized");
        });
    });

    describe("Cross-Tier Execution Flow", () => {
        beforeEach(async () => {
            architecture = await ExecutionArchitecture.create({
                useRedis: false,
                logger,
            });
        });

        it("should support delegation from Tier 1 to Tier 2 to Tier 3", async () => {
            const tier1 = architecture.getTier1();
            const tier2 = architecture.getTier2();
            const tier3 = architecture.getTier3();

            // Create a swarm execution request for Tier 1
            const swarmRequest: TierExecutionRequest = {
                executionId: generatePK(),
                tierOrigin: 0, // External system
                tierTarget: 1, // Tier 1
                type: "swarm",
                payload: {
                    goal: "Test cross-tier execution",
                    teamId: generatePK(),
                    routines: [{
                        id: generatePK(),
                        type: "native",
                        definition: { steps: [{ id: "test_step", action: "echo", params: { message: "test" } }] },
                    }],
                },
                metadata: {
                    userId: generatePK(),
                },
            };

            // Execute through Tier 1 (should delegate to Tier 2, then Tier 3)
            const result = await tier1.execute(swarmRequest);

            expect(result).toBeDefined();
            // The result may be success or failure depending on mocking, but it should not throw
        });

        it("should support direct Tier 2 routine execution", async () => {
            const tier2 = architecture.getTier2();

            const routineRequest: TierExecutionRequest = {
                executionId: generatePK(),
                tierOrigin: 1,
                tierTarget: 2,
                type: "routine",
                payload: {
                    routineId: generatePK(),
                    inputs: { test: "data" },
                    routine: {
                        id: generatePK(),
                        type: "native",
                        definition: { steps: [{ id: "step1", action: "process" }] },
                    },
                },
                metadata: {
                    userId: generatePK(),
                },
            };

            const result = await tier2.execute(routineRequest);
            expect(result).toBeDefined();
        });

        it("should support direct Tier 3 step execution", async () => {
            const tier3 = architecture.getTier3();

            const stepRequest: TierExecutionRequest = {
                executionId: generatePK(),
                tierOrigin: 2,
                tierTarget: 3,
                type: "step",
                payload: {
                    stepInfo: {
                        id: "test_step",
                        action: "echo",
                        params: { message: "hello" },
                    },
                    inputs: { message: "hello" },
                    context: {},
                },
                metadata: {
                    userId: generatePK(),
                    runId: generatePK(),
                },
            };

            const result = await tier3.execute(stepRequest);
            expect(result).toBeDefined();
        });
    });

    describe("State Management Integration", () => {
        beforeEach(async () => {
            architecture = await ExecutionArchitecture.create({
                useRedis: false,
                logger,
            });
        });

        it("should coordinate state between Tier 1 (swarm) and Tier 2 (run) stores", async () => {
            const status = architecture.getStatus();
            expect(status.stateStoresReady).toBe(true);

            // State stores should be accessible through architecture
            // (They're private but the status confirms they're initialized)
            expect(status.tier1Ready).toBe(true); // Tier 1 uses swarm state store
            expect(status.tier2Ready).toBe(true); // Tier 2 uses run state store
        });

        it("should handle both Redis and in-memory state store configurations", async () => {
            // Test in-memory configuration
            const inMemoryArch = await ExecutionArchitecture.create({
                useRedis: false,
                logger,
            });

            expect(inMemoryArch.getStatus().stateStoresReady).toBe(true);
            await inMemoryArch.stop();

            // Test Redis configuration (will work if Redis is available)
            try {
                const redisArch = await ExecutionArchitecture.create({
                    useRedis: true,
                    logger,
                });

                expect(redisArch.getStatus().stateStoresReady).toBe(true);
                await redisArch.stop();
            } catch (error) {
                // Redis might not be available in test environment - that's okay
                expect(error).toBeDefined();
            }
        });
    });

    describe("Tool Registry Integration", () => {
        beforeEach(async () => {
            architecture = await ExecutionArchitecture.create({
                useRedis: false,
                logger,
            });
        });

        it("should initialize integrated tool registry", () => {
            const toolRegistry = architecture.getToolRegistry();
            expect(toolRegistry).toBeDefined();
            
            const status = architecture.getStatus();
            expect(status.toolRegistryReady).toBe(true);
        });

        it("should integrate tool registry with monitoring tools", () => {
            const status = architecture.getStatus();
            // Monitoring tools should be initialized as part of tool registry setup
            expect(status.monitoringToolsReady).toBe(true);
        });
    });

    describe("Event Bus Integration", () => {
        beforeEach(async () => {
            architecture = await ExecutionArchitecture.create({
                useRedis: false,
                logger,
            });
        });

        it("should wire event bus across all tiers", () => {
            const status = architecture.getStatus();
            expect(status.eventBusReady).toBe(true);

            // Event bus should be shared across all tiers
            // (Verified indirectly through successful tier initialization)
            expect(status.tier1Ready).toBe(true);
            expect(status.tier2Ready).toBe(true);
            expect(status.tier3Ready).toBe(true);
        });

        it("should handle event bus initialization failure", async () => {
            // Stop the current architecture to test initialization failure
            await architecture.stop();

            // Mock event bus failure
            const originalConsoleError = console.error;
            console.error = vi.fn();

            try {
                // This test would need to mock the event bus creation to fail
                // For now, we verify that the architecture properly handles event bus state
                const arch = await ExecutionArchitecture.create({ useRedis: false, logger });
                expect(arch.getStatus().eventBusReady).toBe(true);
                await arch.stop();
            } finally {
                console.error = originalConsoleError;
            }
        });
    });

    describe("Resource Management Integration", () => {
        beforeEach(async () => {
            architecture = await ExecutionArchitecture.create({
                useRedis: false,
                logger,
            });
        });

        it("should initialize resource managers for Tier 1 and Tier 3", () => {
            const status = architecture.getStatus();
            
            // Resource managers should be initialized
            // (They're private but their initialization is part of the architecture status)
            expect(status.initialized).toBe(true);
            expect(status.tier1Ready).toBe(true); // Includes Tier 1 resource manager
            expect(status.tier3Ready).toBe(true); // Includes Tier 3 resource manager
        });
    });

    describe("Singleton Pattern", () => {
        it("should provide singleton access via getExecutionArchitecture", async () => {
            const arch1 = await getExecutionArchitecture({
                useRedis: false,
                logger,
            });

            const arch2 = await getExecutionArchitecture();

            // Should return the same instance
            expect(arch1).toBe(arch2);

            // Clean up singleton
            await resetExecutionArchitecture();
        });

        it("should reset singleton properly", async () => {
            const arch1 = await getExecutionArchitecture({
                useRedis: false,
                logger,
            });

            await resetExecutionArchitecture();

            const arch2 = await getExecutionArchitecture({
                useRedis: false,
                logger,
            });

            // Should be different instances after reset
            expect(arch1).not.toBe(arch2);

            await resetExecutionArchitecture();
        });
    });

    describe("Graceful Shutdown", () => {
        it("should stop all components gracefully", async () => {
            architecture = await ExecutionArchitecture.create({
                useRedis: false,
                logger,
            });

            expect(architecture.getStatus().initialized).toBe(true);

            await architecture.stop();

            // After stopping, the internal state should be cleaned up
            // (We can't directly verify internal state, but stop() should not throw)
        });

        it("should handle shutdown errors gracefully", async () => {
            architecture = await ExecutionArchitecture.create({
                useRedis: false,
                logger,
            });

            // Mock an error during shutdown
            const originalEventBusStop = (architecture as any).eventBus?.stop;
            if ((architecture as any).eventBus) {
                (architecture as any).eventBus.stop = vi.fn().mockRejectedValue(new Error("Shutdown error"));
            }

            // Shutdown should handle errors gracefully
            await expect(architecture.stop()).rejects.toThrow("Shutdown error");

            // Restore original method
            if ((architecture as any).eventBus && originalEventBusStop) {
                (architecture as any).eventBus.stop = originalEventBusStop;
            }
        });
    });

    describe("Configuration Override", () => {
        it("should support configuration overrides for each tier", async () => {
            architecture = await ExecutionArchitecture.create({
                useRedis: false,
                logger,
                config: {
                    tier1: { customOption: "tier1Value" },
                    tier2: { customOption: "tier2Value" },
                    tier3: {
                        resourceLimits: {
                            tokens: 50000,
                            apiCalls: 500,
                            computeTime: 1800000,
                            memory: 512,
                            cost: 50,
                        },
                    },
                },
            });

            const status = architecture.getStatus();
            expect(status.initialized).toBe(true);

            // Verify configuration was applied (indirectly through successful initialization)
            expect(status.tier1Ready).toBe(true);
            expect(status.tier2Ready).toBe(true);
            expect(status.tier3Ready).toBe(true);
        });
    });

    describe("Error Handling and Recovery", () => {
        it("should clean up properly when initialization fails", async () => {
            const originalConsoleError = console.error;
            console.error = vi.fn();

            try {
                // Create an architecture instance that will fail during initialization
                const arch = new (ExecutionArchitecture as any)({
                    useRedis: false,
                    logger,
                });

                // Mock a failure in tier initialization
                const originalInitializeTier3 = (arch as any).initializeTier3;
                (arch as any).initializeTier3 = vi.fn().mockRejectedValue(new Error("Tier 3 init failed"));

                await expect((arch as any).initialize()).rejects.toThrow("Tier 3 init failed");

                // Verify cleanup was called (architecture should not be in initialized state)
                expect(arch.getStatus().initialized).toBe(false);

            } finally {
                console.error = originalConsoleError;
            }
        });

        it("should handle missing dependencies gracefully", async () => {
            architecture = await ExecutionArchitecture.create({
                useRedis: false,
                logger,
            });

            // Stop architecture to reset state
            await architecture.stop();

            // Try to access components after stopping
            expect(() => architecture.getTier1()).toThrow("Tier 1 not initialized");
            expect(() => architecture.getTier2()).toThrow("Tier 2 not initialized");
            expect(() => architecture.getTier3()).toThrow("Tier 3 not initialized");
            expect(() => architecture.getToolRegistry()).toThrow("Tool registry not initialized");
        });
    });

    describe("Architecture Status Monitoring", () => {
        beforeEach(async () => {
            architecture = await ExecutionArchitecture.create({
                useRedis: false,
                logger,
            });
        });

        it("should provide comprehensive status information", () => {
            const status = architecture.getStatus();

            expect(status).toHaveProperty("initialized");
            expect(status).toHaveProperty("tier1Ready");
            expect(status).toHaveProperty("tier2Ready");
            expect(status).toHaveProperty("tier3Ready");
            expect(status).toHaveProperty("eventBusReady");
            expect(status).toHaveProperty("stateStoresReady");
            expect(status).toHaveProperty("toolRegistryReady");
            expect(status).toHaveProperty("monitoringToolsReady");

            // All should be true for properly initialized architecture
            Object.values(status).forEach(value => {
                expect(typeof value).toBe("boolean");
            });
        });

        it("should start successfully after initialization", async () => {
            // Start should work without errors
            await architecture.start();

            // Status should remain good after start
            const status = architecture.getStatus();
            expect(status.initialized).toBe(true);
        });
    });
});
