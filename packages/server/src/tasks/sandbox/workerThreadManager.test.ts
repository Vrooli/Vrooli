/* eslint-disable @typescript-eslint/ban-ts-comment */
import { jest } from "@jest/globals";
import { Worker } from "worker_threads";
import { WorkerThreadManager } from "./workerThreadManager";

// List of handlers we expect to be added to the worker
const workerOnHandlers = ["message", "error", "exit"];

function checkNumWorkersCreated(workersCreated: number): Worker[] {
    // Check that the correct number of workers were created
    // @ts-ignore: Testing runtime scenario
    const workers = Worker.getInstances();
    expect(workers).toHaveLength(workersCreated);

    // Check that each worker has the correct handlers added
    workerOnHandlers.forEach(handler => {
        workers.forEach(worker => {
            expect(worker.on).toHaveBeenCalledWith(handler, expect.any(Function));
        });
    });

    return workers;
}

function checkJobsSentToWorker(jobsSent: number, worker: Worker): void {
    // @ts-ignore: Testing runtime scenario
    expect(worker.postMessage).toHaveBeenCalledTimes(jobsSent);
}

describe("WorkerThreadManager", () => {
    let manager: WorkerThreadManager;

    beforeEach(() => {
        jest.clearAllMocks();
        manager = new WorkerThreadManager();
    });

    afterEach(() => {
        jest.useRealTimers();
        // @ts-ignore: Testing runtime scenario
        Worker.resetMock();
    });

    test("constructor initializes with default values", () => {
        expect(manager).toHaveProperty("memoryLimit");
        expect(manager).toHaveProperty("idleTimeout");
        expect(manager["worker"]).toBeNull();
        expect(manager["timeoutHandle"]).toBeNull();
    });

    test("runUserCode starts a new worker if none exists", async () => {
        const input = { code: "function test() { return \"Hello\"; }", input: {} };
        await manager.runUserCode(input);

        const [worker] = checkNumWorkersCreated(1);
        checkJobsSentToWorker(1, worker);
    });

    test("runUserCode reuses existing worker", async () => {
        const input = { code: "function test() { return \"Hello\"; }", input: {} };
        await manager.runUserCode(input);
        await manager.runUserCode(input);

        const [worker] = checkNumWorkersCreated(1);
        checkJobsSentToWorker(2, worker);
    });

    test("worker is terminated after idle timeout", async () => {
        jest.useFakeTimers();
        const input = { code: "function test() { return \"Hello\"; }", input: {} };
        await manager.runUserCode(input);

        const [worker] = checkNumWorkersCreated(1);
        expect(manager["worker"]).not.toBeNull();

        jest.advanceTimersByTime(manager["idleTimeout"] + 1000);

        expect(worker.terminate).toHaveBeenCalled();
        expect(manager["worker"]).toBeNull();
    });

    describe("handle sucessful code execution", () => {
        test("without an input", async () => {
            const input = { code: "function test() { return \"Hello\"; }" };
            const result = await manager.runUserCode(input);

            expect(result).not.toHaveProperty("error");
            expect(result).toHaveProperty("output");
            expect(result.output).toEqual("Hello");
        });

        test("with a string input", async () => {
            const input = { code: "function test(boop) { return boop; }", input: "Hi" };
            const result = await manager.runUserCode(input);

            expect(result).not.toHaveProperty("error");
            expect(result).toHaveProperty("output");
            expect(result.output).toEqual("Hi");
        });

        test("with a number input", async () => {
            const input = { code: "function test(input) { return Math.abs(input * 2); }", input: -4.20 };
            const result = await manager.runUserCode(input);

            expect(result).not.toHaveProperty("error");
            expect(result).toHaveProperty("output");
            expect(result.output).toEqual(8.40);
        });

        test("with an object input", async () => {
            const input = { code: "function test(obj) { return obj; }", input: { foo: "bar" } };
            const result = await manager.runUserCode(input);

            expect(result).not.toHaveProperty("error");
            expect(result).toHaveProperty("output");
            expect(result.output).toEqual({ foo: "bar" });
        });

        // Date input, circular input, array input, null input, etc.
    });

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
