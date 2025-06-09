import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { useDebugStore } from "../../stores/debugStore.js";

const originalNodeEnv = process.env.NODE_ENV;

describe("DebugComponent", () => {
    afterEach(() => {
        // Restore original NODE_ENV after each test
        process.env.NODE_ENV = originalNodeEnv;
    });

    it("should add data to the store correctly", () => {
        const store = useDebugStore.getState();
        const traceId = "test-trace";
        const testData = { value: 42, message: "test" };

        // Add data to the store
        store.addData(traceId, testData);

        // Verify data was added correctly
        const trace = store.traces[traceId];
        expect(trace).toBeDefined();
        expect(trace.count).toBe(1);
        expect(trace.data).toEqual(testData);

        // Add more data to the same trace
        store.addData(traceId, { value: 43, message: "updated" });

        // Verify the count increased and data was updated
        expect(store.traces[traceId].count).toBe(2);
        expect(store.traces[traceId].data).toEqual({ value: 43, message: "updated" });
    });

    it("should handle multiple traces", () => {
        const store = useDebugStore.getState();

        // Add data to multiple traces
        store.addData("trace1", { value: 1 });
        store.addData("trace2", { value: 2 });
        store.addData("trace3", { value: 3 });

        // Verify all traces exist
        expect(Object.keys(store.traces).length).toBeGreaterThanOrEqual(3);
        expect(store.traces["trace1"]).toBeDefined();
        expect(store.traces["trace2"]).toBeDefined();
        expect(store.traces["trace3"]).toBeDefined();

        // Verify data for each trace
        expect(store.traces["trace1"].data).toEqual({ value: 1 });
        expect(store.traces["trace2"].data).toEqual({ value: 2 });
        expect(store.traces["trace3"].data).toEqual({ value: 3 });
    });
}); 
