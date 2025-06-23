// AI_CHECK: TEST_COVERAGE=7 | LAST: 2025-06-18
import { describe, it, expect } from "vitest";
import { ExecutionStates } from "../swarm.js";

describe("Execution Types Constants", () => {
    describe("ExecutionStates", () => {
        it("should export all required execution states", () => {
            expect(ExecutionStates.UNINITIALIZED).toBe("UNINITIALIZED");
            expect(ExecutionStates.STARTING).toBe("STARTING");
            expect(ExecutionStates.RUNNING).toBe("RUNNING");
            expect(ExecutionStates.IDLE).toBe("IDLE");
            expect(ExecutionStates.PAUSED).toBe("PAUSED");
            expect(ExecutionStates.STOPPED).toBe("STOPPED");
            expect(ExecutionStates.FAILED).toBe("FAILED");
            expect(ExecutionStates.TERMINATED).toBe("TERMINATED");
        });

        it("should have consistent string values", () => {
            // Verify that each state key matches its value
            Object.entries(ExecutionStates).forEach(([key, value]) => {
                expect(key).toBe(value);
            });
        });

        it("should have exactly 8 states", () => {
            expect(Object.keys(ExecutionStates)).toHaveLength(8);
        });

        it("should be a well-defined constant object", () => {
            // Verify that ExecutionStates is properly defined
            expect(typeof ExecutionStates).toBe("object");
            expect(ExecutionStates).not.toBe(null);
            
            // Verify each state is a string
            Object.values(ExecutionStates).forEach(state => {
                expect(typeof state).toBe("string");
                expect(state.length).toBeGreaterThan(0);
            });
        });
    });
});
