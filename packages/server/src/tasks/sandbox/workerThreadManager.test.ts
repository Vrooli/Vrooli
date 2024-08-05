/* eslint-disable @typescript-eslint/ban-ts-comment */
import { jest } from "@jest/globals";
import { Worker } from "worker_threads";
import { WorkerThreadManager } from "./workerThreadManager";

// List of handlers we expect to be added to the worker
const workerOnHandlers = ["message", "error", "exit"];

describe("WorkerThreadManager", () => {
    let manager: WorkerThreadManager;

    beforeEach(() => {
        jest.clearAllMocks();
        manager = new WorkerThreadManager();
    });

    afterEach(() => {
        jest.useRealTimers();
    });

    test("constructor initializes with default values", () => {
        expect(manager).toHaveProperty("memoryLimit");
        expect(manager).toHaveProperty("idleTimeout");
        expect(manager["worker"]).toBeNull();
        expect(manager["timeoutHandle"]).toBeNull();
    });

    test("runUserCode starts a new worker if none exists", async () => {
        const input = { code: "function test() { return \"Hello\"; }", input: {} };
        console.log("yeet before", Worker);
        await manager.runUserCode(input);
        console.log("yeet after", Worker, Worker.on);

        // @ts-ignore: Testing runtime scenario
        const workers = Worker.getInstances();
        expect(workers).toHaveLength(1);
        const worker = workers[0];
        expect(worker.on).toHaveBeenCalledTimes(workerOnHandlers.length);
        // @ts-ignore: Testing runtime scenario
        expect(worker.postMessage).toHaveBeenCalledWith(input);
    });

    // test("runUserCode reuses existing worker", async () => {
    //     const input = { code: "function test() { return \"Hello\"; }", input: {} };
    //     await manager.runUserCode(input);
    //     await manager.runUserCode(input);

    //     expect(Worker).toHaveBeenCalledTimes(1);
    //     // @ts-ignore: Testing runtime scenario
    //     expect(Worker.postMessage).toHaveBeenCalledTimes(2);
    // });

    // test("worker is terminated after idle timeout", async () => {
    //     jest.useFakeTimers();
    //     const input = { code: "function test() { return \"Hello\"; }", input: {} };
    //     await manager.runUserCode(input);

    //     jest.advanceTimersByTime(manager["idleTimeout"] + 1000);

    //     // @ts-ignore: Testing runtime scenario
    //     expect(Worker.terminate).toHaveBeenCalled();
    //     expect(manager["worker"]).toBeNull();
    // });

    // test("handles successful code execution", async () => {
    //     // @ts-ignore: Testing runtime scenario
    //     Worker.on.mockImplementation((event, callback) => {
    //         if (event === "message") {
    //             setTimeout(() => callback({ result: "Success" }), 10);
    //         }
    //     });

    //     const input = { code: "function test() { return \"Success\"; }", input: {} };
    //     const result = await manager.runUserCode(input);

    //     expect(result).toEqual({ result: "Success" });
    // });

    // test("handles code execution errors", async () => {
    //     Worker.on.mockImplementation((event, callback) => {
    //         if (event === "error") {
    //             setTimeout(() => callback(new Error("Execution error")), 10);
    //         }
    //     });

    //     const input = { code: "function test() { throw new Error(\"Execution error\"); }", input: {} };
    //     await expect(manager.runUserCode(input)).rejects.toThrow("Execution error");
    // });

    // test("handles worker exit", async () => {
    //     Worker.on.mockImplementation((event, callback) => {
    //         if (event === "exit") {
    //             setTimeout(() => callback(1), 10);
    //         }
    //     });

    //     const input = { code: "function test() { process.exit(1); }", input: {} };
    //     await expect(manager.runUserCode(input)).rejects.toThrow("Child process exited with code 1");
    // });

    // test("handles maximum code length", async () => {
    //     Worker.on.mockImplementation((event, callback) => {
    //         if (event === "message") {
    //             setTimeout(() => callback({ error: "Code is too long" }), 10);
    //         }
    //     });

    //     const longCode = "a".repeat(8193);
    //     const input = { code: `function test() { ${longCode} }`, input: {} };

    //     const result = await manager.runUserCode(input);

    //     expect(result).toEqual({ error: "Code is too long" });
    // });

    // test("handles missing function name", async () => {
    //     Worker.on.mockImplementation((event, callback) => {
    //         if (event === "message") {
    //             setTimeout(() => callback({ error: "Function name not found" }), 10);
    //         }
    //     });

    //     const input = { code: "const x = 5;", input: {} };

    //     const result = await manager.runUserCode(input);

    //     expect(result).toEqual({ error: "Function name not found" });
    // });

    // test("handles various function declaration styles", async () => {
    //     const testCases = [
    //         { code: "function test() { return \"Hello\"; }", expected: "Hello" },
    //         { code: "const test = function() { return \"Hello\"; }", expected: "Hello" },
    //         { code: "const test = () => \"Hello\"", expected: "Hello" },
    //         { code: "async function test() { return \"Hello\"; }", expected: "Hello" },
    //         { code: "const test = async () => \"Hello\"", expected: "Hello" },
    //     ];

    //     for (const { code, expected } of testCases) {
    //         Worker.on.mockImplementation((event, callback) => {
    //             if (event === "message") {
    //                 setTimeout(() => callback({ result: expected }), 10);
    //             }
    //         });

    //         const input = { code, input: {} };
    //         const result = await manager.runUserCode(input);

    //         expect(result).toEqual({ result: expected });
    //     }
    // });
});
