/**
 * Basic Framework Test
 * 
 * Minimal test to verify execution framework is functional
 */

import { describe, it, expect, beforeAll } from "vitest";
import { initializeExecutionAssertions } from "./assertions/index.js";
import { MockController } from "./factories/scenario/MockController.js";
import { ResourceManager } from "./factories/scenario/ResourceManager.js";
import { ErrorHandler } from "./factories/scenario/ErrorHandler.js";

describe("Basic Execution Framework", () => {
    beforeAll(async () => {
        // Initialize custom assertions
        initializeExecutionAssertions();
    });

    describe("Core Components", () => {
        it("should instantiate MockController without errors", () => {
            const controller = new MockController();
            expect(controller).toBeDefined();
            expect(typeof controller.configure).toBe("function");
            expect(typeof controller.override).toBe("function");
            expect(typeof controller.clear).toBe("function");
        });

        it("should instantiate ResourceManager without errors", () => {
            const manager = new ResourceManager({
                maxMemoryMB: 256,
                maxDurationMs: 30000,
                maxEventSubscriptions: 50,
                maxConcurrentOperations: 10,
            });
            expect(manager).toBeDefined();
            expect(typeof manager.addResource).toBe("function");
            expect(typeof manager.destroy).toBe("function");
        });

        it("should instantiate ErrorHandler without errors", () => {
            const handler = new ErrorHandler();
            expect(handler).toBeDefined();
            expect(typeof handler.withErrorHandling).toBe("function");
        });
    });

    describe("Mock System", () => {
        it("should configure and retrieve mocks", async () => {
            const controller = new MockController();
            
            await controller.configure({
                ai: {
                    "test-routine": {
                        responses: [
                            { response: { result: "test-response" } },
                        ],
                    },
                },
            });

            const response = await controller.getMockResponse("ai", "test-routine");
            expect(response).toBeDefined();
            expect(response.response.result).toBe("test-response");
        });
    });

    describe("Custom Assertions", () => {
        it("should support blackboard assertions", () => {
            const blackboard = new Map();
            blackboard.set("key1", "value1");
            blackboard.set("key2", { nested: true });

            expect(blackboard).toHaveKey("key1");
            expect(blackboard).toHaveValue("key1", "value1");
            expect(blackboard).toHaveKeys(["key1", "key2"]);
        });

        it("should support event assertions", () => {
            const events = [
                { topic: "test/event1", data: { id: 1 }, timestamp: new Date(), source: "test" },
                { topic: "test/event2", data: { id: 2 }, timestamp: new Date(), source: "test" },
                { topic: "test/event1", data: { id: 3 }, timestamp: new Date(), source: "test" },
            ];

            expect(events).toContainEvents(["test/event1", "test/event2"]);
            expect(events).toHaveEventCount("test/event1", 2);
            expect(events).toHaveEventCount("test/event2", 1);
        });
    });

    describe("Error Handling", () => {
        it("should handle errors with retry logic", async () => {
            const handler = new ErrorHandler({
                maxRetries: 2,
                retryDelay: 100,
            });

            let attemptCount = 0;
            const operation = async () => {
                attemptCount++;
                if (attemptCount < 3) {
                    throw new Error("Temporary error");
                }
                return "success";
            };

            const result = await handler.withErrorHandling(operation, {
                scenarioName: "test",
                operation: "retry-test",
                startTime: new Date(),
            });

            expect(result).toBe("success");
            expect(attemptCount).toBe(3);
        });

        it("should suppress errors when configured", async () => {
            const handler = new ErrorHandler({
                maxRetries: 0,
                suppressErrors: true,
            });

            const result = await handler.withErrorHandling(
                () => Promise.reject(new Error("Test error")),
                {
                    scenarioName: "test",
                    operation: "suppress-test",
                    startTime: new Date(),
                },
            );

            expect(result).toBeNull();
        });
    });

    describe("Resource Management", () => {
        it("should track and clean up resources", async () => {
            const manager = new ResourceManager({
                maxMemoryMB: 256,
                maxDurationMs: 30000,
                maxEventSubscriptions: 10,
                maxConcurrentOperations: 5,
            });

            let cleanedUp = false;
            await manager.addResource({
                id: "test-resource",
                type: "subscription",
                cleanup: async () => {
                    cleanedUp = true;
                },
            });

            const usage = manager.getCurrentUsage();
            expect(usage.eventSubscriptions).toBe(1);

            await manager.destroy();
            expect(cleanedUp).toBe(true);
        });

        it("should enforce resource limits", async () => {
            const manager = new ResourceManager({
                maxMemoryMB: 256,
                maxDurationMs: 30000,
                maxEventSubscriptions: 2,
                maxConcurrentOperations: 5,
            });

            await manager.addResource({
                id: "resource1",
                type: "subscription",
                cleanup: async () => {},
            });

            await manager.addResource({
                id: "resource2",
                type: "subscription",
                cleanup: async () => {},
            });

            // Should throw when exceeding limit
            await expect(
                manager.addResource({
                    id: "resource3",
                    type: "subscription",
                    cleanup: async () => {},
                }),
            ).rejects.toThrow("Event subscription limit exceeded: 2");

            await manager.destroy();
        });
    });
});
