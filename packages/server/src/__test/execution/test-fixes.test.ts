/**
 * Test Fixes Verification
 * 
 * Simple test to verify that our fixes work and the framework can be instantiated
 */

import { describe, it, expect } from "vitest";
import "./assertions/index.js"; // Auto-initializes custom assertions
import { RoutineFactory } from "./factories/routine/RoutineFactory.js";
import { AgentFactory } from "./factories/agent/AgentFactory.js";
import { SwarmFactory } from "./factories/swarm/SwarmFactory.js";
import { ScenarioFactory } from "./factories/scenario/ScenarioFactory.js";
import { MockController } from "./factories/scenario/MockController.js";
import { ResourceManager } from "./factories/scenario/ResourceManager.js";
import { ErrorHandler } from "./factories/scenario/ErrorHandler.js";
import { RoutineSchemaRegistry } from "./schemas/routines/index.js";
import { AgentSchemaRegistry } from "./schemas/agents/index.js";
import { SwarmSchemaRegistry } from "./schemas/swarms/index.js";

describe("Execution Test Framework Fixes", () => {

    describe("Schema System", () => {
        it("should instantiate schema registries without errors", async () => {
            // Test that registries can be created and initialized
            expect(() => new RoutineSchemaRegistry()).not.toThrow();
            expect(() => new AgentSchemaRegistry()).not.toThrow();
            expect(() => new SwarmSchemaRegistry()).not.toThrow();
            
            // Test that list methods work
            expect(RoutineSchemaRegistry.list()).toBeInstanceOf(Array);
            expect(AgentSchemaRegistry.list()).toBeInstanceOf(Array);
            expect(SwarmSchemaRegistry.list()).toBeInstanceOf(Array);
        });

        it("should handle schema validation errors gracefully", async () => {
            // Test that validation catches invalid schemas
            await expect(
                RoutineSchemaRegistry.validate({ invalid: "schema" }),
            ).rejects.toThrow();
            
            await expect(
                AgentSchemaRegistry.validate({ invalid: "schema" }),
            ).rejects.toThrow();
            
            await expect(
                SwarmSchemaRegistry.validate({ invalid: "schema" }),
            ).rejects.toThrow();
        });
    });

    describe("Factory System", () => {
        it("should instantiate factories without errors", () => {
            expect(() => new RoutineFactory()).not.toThrow();
            expect(() => new AgentFactory()).not.toThrow();
            expect(() => new SwarmFactory()).not.toThrow();
            expect(() => new ScenarioFactory()).not.toThrow();
        });

        it("should handle factory configuration properly", () => {
            const routineFactory = new RoutineFactory();
            const agentFactory = new AgentFactory();
            const swarmFactory = new SwarmFactory();
            
            // Test that factories have expected methods
            expect(typeof routineFactory.createFromSchema).toBe("function");
            expect(typeof routineFactory.createBatch).toBe("function");
            
            expect(typeof agentFactory.createFromSchema).toBe("function");
            expect(typeof agentFactory.createBatch).toBe("function");
            
            expect(typeof swarmFactory.createFromSchema).toBe("function");
            expect(typeof swarmFactory.createBatch).toBe("function");
        });
    });

    describe("Mock System", () => {
        it("should instantiate mock controllers without static state issues", () => {
            const mockController1 = new MockController();
            const mockController2 = new MockController();
            
            // Test that mock controllers are independent
            expect(mockController1).not.toBe(mockController2);
            
            // Test that they have expected methods
            expect(typeof mockController1.configure).toBe("function");
            expect(typeof mockController1.override).toBe("function");
            expect(typeof mockController1.clear).toBe("function");
        });

        it("should handle mock configurations properly", async () => {
            const mockController = new MockController();
            
            const testConfig = {
                ai: {
                    "test-routine": {
                        responses: [{ response: { result: "test" } }],
                    },
                },
            };
            
            // Test that configuration works
            await expect(mockController.configure(testConfig)).resolves.not.toThrow();
            
            // Test that responses can be retrieved
            const response = await mockController.getMockResponse("ai", "test-routine");
            expect(response).toBeDefined();
            expect(response.response.result).toBe("test");
        });
    });

    describe("Resource Management", () => {
        it("should instantiate resource manager properly", () => {
            const resourceManager = new ResourceManager({
                maxMemoryMB: 256,
                maxDurationMs: 30000,
                maxEventSubscriptions: 50,
                maxConcurrentOperations: 10,
            });
            
            expect(resourceManager).toBeDefined();
            expect(typeof resourceManager.addResource).toBe("function");
            expect(typeof resourceManager.checkLimits).toBe("function");
            expect(typeof resourceManager.destroy).toBe("function");
        });

        it("should track resource usage correctly", async () => {
            const resourceManager = new ResourceManager({
                maxMemoryMB: 256,
                maxDurationMs: 30000,
                maxEventSubscriptions: 50,
                maxConcurrentOperations: 10,
            });
            
            const usage = resourceManager.getCurrentUsage();
            expect(usage).toBeDefined();
            expect(usage.startTime).toBeInstanceOf(Date);
            expect(usage.eventSubscriptions).toBe(0);
            expect(usage.concurrentOperations).toBe(0);
            
            await resourceManager.destroy();
        });
    });

    describe("Error Handling", () => {
        it("should instantiate error handler properly", () => {
            const errorHandler = new ErrorHandler();
            
            expect(errorHandler).toBeDefined();
            expect(typeof errorHandler.withErrorHandling).toBe("function");
            expect(typeof errorHandler.withTimeout).toBe("function");
            expect(typeof errorHandler.createErrorBoundary).toBe("function");
        });

        it("should handle errors with proper context", async () => {
            const errorHandler = new ErrorHandler({
                maxRetries: 1,
                retryDelay: 100,
                suppressErrors: true,
            });
            
            const result = await errorHandler.withErrorHandling(
                () => Promise.reject(new Error("Test error")),
                {
                    scenarioName: "test-scenario",
                    operation: "test-operation",
                    startTime: new Date(),
                },
            );
            
            // Should return null when suppressErrors is true
            expect(result).toBeNull();
            
            // Should have recorded the error
            const history = errorHandler.getErrorHistory();
            expect(history).toHaveLength(2); // 1 initial attempt + 1 retry
        });
    });

    describe("Type System", () => {
        it("should have proper TypeScript types", () => {
            // This test ensures our types compile correctly
            const routineFactory = new RoutineFactory();
            const agentFactory = new AgentFactory();
            const swarmFactory = new SwarmFactory();
            
            // Test that TypeScript infers the correct return types
            expect(routineFactory.createFromSchema).toBeDefined();
            expect(agentFactory.createFromSchema).toBeDefined();
            expect(swarmFactory.createFromSchema).toBeDefined();
        });
    });

    describe("Custom Assertions", () => {
        it("should have custom blackboard assertions", () => {
            const blackboard = new Map();
            blackboard.set("test-key", "test-value");
            
            // Test that custom assertions are available
            expect(blackboard).toHaveKey("test-key");
            expect(blackboard).toHaveValue("test-key", "test-value");
        });

        it("should have custom event assertions", () => {
            const events = [
                { topic: "event1", data: {}, timestamp: new Date(), source: "test" },
                { topic: "event2", data: {}, timestamp: new Date(), source: "test" },
            ];
            
            // Test that custom assertions are available
            expect(events).toContainEvents(["event1", "event2"]);
            expect(events).toHaveEventCount("event1", 1);
        });
    });
});
