// AI_CHECK: TEST_COVERAGE=7 | LAST: 2025-06-18
import { describe, it, expect } from "vitest";
import { createTierRequest } from "../communication.js";
import { isExecutionEvent } from "../events.js";

describe("Execution Types Functions", () => {
    describe("createTierRequest", () => {
        it("should create a valid tier request with all parameters", () => {
            const context = {
                executionId: "exec-123",
                swarmId: "swarm-456", 
                userId: "user-789",
                timestamp: new Date(),
                correlationId: "corr-abc",
            };
            
            const allocation = {
                compute: { cpuCores: 2 },
                memory: { maxHeapMb: 512 },
                timeoutMs: 30000,
            };
            
            const input = { message: "test input" };
            const options = { priority: "high" };
            
            const request = createTierRequest(context, input, allocation, options);
            
            expect(request).toEqual({
                context,
                input,
                allocation,
                options,
            });
        });

        it("should create a valid tier request without options", () => {
            const context = {
                executionId: "exec-123",
                swarmId: "swarm-456",
                userId: "user-789", 
                timestamp: new Date(),
                correlationId: "corr-abc",
            };
            
            const allocation = {
                compute: { cpuCores: 1 },
                memory: { maxHeapMb: 256 },
                timeoutMs: 15000,
            };
            
            const input = "simple string input";
            
            const request = createTierRequest(context, input, allocation);
            
            expect(request).toEqual({
                context,
                input,
                allocation,
                options: undefined,
            });
        });

        it("should handle different input types", () => {
            const context = {
                executionId: "exec-123",
                swarmId: "swarm-456",
                userId: "user-789",
                timestamp: new Date(),
                correlationId: "corr-abc",
            };
            
            const allocation = {
                compute: { cpuCores: 1 },
                memory: { maxHeapMb: 256 },
                timeoutMs: 15000,
            };
            
            // Test with number input
            const numRequest = createTierRequest(context, 42, allocation);
            expect(numRequest.input).toBe(42);
            
            // Test with array input
            const arrayRequest = createTierRequest(context, [1, 2, 3], allocation);
            expect(arrayRequest.input).toEqual([1, 2, 3]);
            
            // Test with null input
            const nullRequest = createTierRequest(context, null, allocation);
            expect(nullRequest.input).toBeNull();
        });
    });

    describe("isExecutionEvent", () => {
        it("should return true for valid execution events", () => {
            const validEvent = {
                id: "event-123",
                type: "test.event",
                timestamp: new Date(),
                source: {
                    tier: "tier1.swarm",
                    component: "coordination",
                    instanceId: "inst-456",
                },
                correlationId: "corr-789",
                data: { message: "test" },
            };
            
            expect(isExecutionEvent(validEvent)).toBe(true);
        });

        it("should return true for minimal valid execution events", () => {
            const minimalEvent = {
                id: "event-123",
                type: "simple.event",
                timestamp: new Date(),
            };
            
            expect(isExecutionEvent(minimalEvent)).toBe(true);
        });

        it("should return false for objects without type field", () => {
            const invalidEvent = {
                id: "event-123",
                timestamp: new Date(),
            };
            
            expect(isExecutionEvent(invalidEvent)).toBe(false);
        });

        it("should return false for objects with non-string type", () => {
            const invalidEvent = {
                id: "event-123",
                type: 123,
                timestamp: new Date(),
            };
            
            expect(isExecutionEvent(invalidEvent)).toBe(false);
        });

        it("should return false for null and undefined", () => {
            expect(isExecutionEvent(null)).toBe(false);
            expect(isExecutionEvent(undefined)).toBe(false);
        });

        it("should return false for primitive types", () => {
            expect(isExecutionEvent("string")).toBe(false);
            expect(isExecutionEvent(123)).toBe(false);
            expect(isExecutionEvent(true)).toBe(false);
        });

        it("should return false for arrays", () => {
            expect(isExecutionEvent([])).toBe(false);
            expect(isExecutionEvent([{ type: "event" }])).toBe(false);
        });

        it("should return true for events with optional fields", () => {
            const eventWithOptionals = {
                id: "event-123",
                type: "optional.event",
                timestamp: new Date(),
                source: {
                    tier: "tier2",
                    component: "orchestrator", 
                    instanceId: "inst-789",
                },
                correlationId: "corr-123",
                causationId: "cause-456",
                data: { key: "value" },
                metadata: { extra: "info" },
            };
            
            expect(isExecutionEvent(eventWithOptionals)).toBe(true);
        });
    });
});
