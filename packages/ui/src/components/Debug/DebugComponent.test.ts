import { expect } from "chai";
import { afterEach, describe, it } from "mocha";
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
        expect(trace).to.exist;
        expect(trace.count).to.equal(1);
        expect(trace.data).to.deep.equal(testData);

        // Add more data to the same trace
        store.addData(traceId, { value: 43, message: "updated" });

        // Verify the count increased and data was updated
        expect(store.traces[traceId].count).to.equal(2);
        expect(store.traces[traceId].data).to.deep.equal({ value: 43, message: "updated" });
    });

    it("should handle multiple traces", () => {
        const store = useDebugStore.getState();

        // Add data to multiple traces
        store.addData("trace1", { value: 1 });
        store.addData("trace2", { value: 2 });
        store.addData("trace3", { value: 3 });

        // Verify all traces exist
        expect(Object.keys(store.traces).length).to.be.at.least(3);
        expect(store.traces["trace1"]).to.exist;
        expect(store.traces["trace2"]).to.exist;
        expect(store.traces["trace3"]).to.exist;

        // Verify data for each trace
        expect(store.traces["trace1"].data).to.deep.equal({ value: 1 });
        expect(store.traces["trace2"].data).to.deep.equal({ value: 2 });
        expect(store.traces["trace3"].data).to.deep.equal({ value: 3 });
    });
}); 
