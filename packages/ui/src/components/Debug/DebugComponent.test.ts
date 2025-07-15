// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-19
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { useDebugStore } from "../../stores/debugStore.js";

const originalNodeEnv = process.env.NODE_ENV;

describe("useDebugStore", () => {
    beforeEach(() => {
        // Reset the debug store before each test
        useDebugStore.setState({ traces: {} });
    });

    afterEach(() => {
        // Restore original NODE_ENV after each test
        process.env.NODE_ENV = originalNodeEnv;
    });

    it("adds data to the store and increments count", () => {
        const traceId = "test-trace";
        const testData = { value: 42, message: "test" };

        // Add data to the store
        useDebugStore.getState().addData(traceId, testData);

        // Get fresh state after update
        const stateAfterAdd = useDebugStore.getState();
        const trace = stateAfterAdd.traces[traceId];
        
        expect(trace).toBeDefined();
        expect(trace.count).toBe(1);
        expect(trace.data).toEqual(testData);

        // Add more data to the same trace
        useDebugStore.getState().addData(traceId, { value: 43, message: "updated" });

        // Get fresh state again
        const stateAfterUpdate = useDebugStore.getState();
        expect(stateAfterUpdate.traces[traceId].count).toBe(2);
        expect(stateAfterUpdate.traces[traceId].data).toEqual({ value: 43, message: "updated" });
    });

    it("handles multiple independent traces", () => {
        // Add data to multiple traces
        useDebugStore.getState().addData("trace1", { value: 1 });
        useDebugStore.getState().addData("trace2", { value: 2 });
        useDebugStore.getState().addData("trace3", { value: 3 });

        // Get fresh state
        const stateAfterAdds = useDebugStore.getState();

        // Verify all traces exist
        expect(Object.keys(stateAfterAdds.traces).length).toBeGreaterThanOrEqual(3);
        expect(stateAfterAdds.traces["trace1"]).toBeDefined();
        expect(stateAfterAdds.traces["trace2"]).toBeDefined();
        expect(stateAfterAdds.traces["trace3"]).toBeDefined();

        // Verify data for each trace
        expect(stateAfterAdds.traces["trace1"].data).toEqual({ value: 1 });
        expect(stateAfterAdds.traces["trace2"].data).toEqual({ value: 2 });
        expect(stateAfterAdds.traces["trace3"].data).toEqual({ value: 3 });
    });
}); 
