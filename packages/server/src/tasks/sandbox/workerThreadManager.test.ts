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

        test("without an output", async () => {
            const input = { code: "function test() { return; }", input: {} };
            const result = await manager.runUserCode(input);

            expect(result).not.toHaveProperty("error");
            expect(result).toHaveProperty("output");
            expect(result.output).toBeUndefined();
        });

        test("async function", async () => {
            const input = { code: "async function test() { return \"Hello\"; }", input: {} };
            const result = await manager.runUserCode(input);

            expect(result).not.toHaveProperty("error");
            expect(result).toHaveProperty("output");
            expect(result.output).toEqual("Hello");
        });

        describe("primitive input types", () => {
            test("string", async () => {
                const input = { code: "function test(boop) { return boop; }", input: "hello, world! 🌍\"Chicken coop\" \nBeep boop" };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toEqual(input.input);
            });
            test("negative number", async () => {
                const input = { code: "function test(input) { return Math.abs(input * 2); }", input: -4.20 };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toEqual(Math.abs(input.input * 2));
            });
            test("positive number", async () => {
                const input = { code: "function test(input) { return input * 100; }", input: 4.20 };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toEqual(input.input * 100);
            });
            test("NaN", async () => {
                const input = { code: "function test(input) { return input; }", input: NaN };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toBeNaN();
            });
            test("Infinity", async () => {
                const input = { code: "function test(input) { return input; }", input: Infinity };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toBe(Infinity);
            });
            test("boolean true", async () => {
                const input = { code: "function test(input) { return input; }", input: true };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toBe(true);
            });
            test("boolean false", async () => {
                const input = { code: "function test(input) { return input; }", input: false };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toBe(false);
            });
            test("null", async () => {
                const input = { code: "function test(input) { return input; }", input: null };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toBeNull();
            });
            test("undefined", async () => {
                const input = { code: "function test(input) { return input; }", input: undefined };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toBeUndefined();
            });
            test("BigInt", async () => {
                const input = { code: "function test(input) { return input; }", input: BigInt(9007199254740991) };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toBe(input.input);
            });
            test("Date", async () => {
                const input = { code: "function test(input) { return input; }", input: new Date("2021-01-01T00:00:00Z") };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toEqual(input.input);
            });
            test("regexp", async () => {
                const input = { code: "function test(input) { return input; }", input: /foo/gi };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toEqual(input.input);
            });
            test("Error", async () => {
                const input = { code: "function test(input) { return input; }", input: new Error("test error") };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output.message).toBe(input.input.message);
            });
            test("URL", async () => {
                const input = { code: "function test(input) { return input; }", input: new URL("https://example.com") };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                // @ts-ignore: Testing runtime scenario
                expect(result.output.href).toBe(input.input.href);
            });
            test("Buffer", async () => {
                const input = { code: "function test(input) { return input; }", input: Buffer.from("hello, world!") };
                console.log("yeet before", input.input.toString());
                const result = await manager.runUserCode(input);
                console.log("yeet after", result.output?.toString());

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                // @ts-ignore: Testing runtime scenario
                expect(result.output.toString()).toBe(input.input.toString());
            });
            test("Uint8Array", async () => {
                const input = { code: "function test(input) { return input; }", input: new Uint8Array([0, 1, 2, 3, 4, 5]) };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toEqual(input.input);
            });
            test("Symbol", async () => {
                const input = { code: "function test(input) { return input; }", input: Symbol("test") };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toBe(undefined); // Symbols are not serializable
            });
            test("function", async () => {
                const input = { code: "function test(input) { return input; }", input: () => { } };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toBe(undefined); // Functions are not serializable
            });
        });

        describe("Map and Set input types", () => {
            test("Map with primitive values", async () => {
                const input = { code: "function test(input) { return input; }", input: new Map<any, any>([["foo", "bar"], [42, "baz"]]) };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toEqual(input.input);
            });
            test("Map with object values", async () => {
                const input = { code: "function test(input) { return input; }", input: new Map<any, any>([["foo", { bar: "baz" }]]) };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toEqual(input.input);
            });
            test("Set with primitive values", async () => {
                const input = { code: "function test(input) { return input; }", input: new Set<any>(["foo", 42]) };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toEqual(input.input);
            });
            test("Set with object values", async () => {
                const input = { code: "function test(input) { return input; }", input: new Set<any>([{ foo: "bar" }]) };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toEqual(input.input);
            });
        });

        describe("object and array input types", () => {
            test("empty object", async () => {
                const input = { code: "function test(input) { return input; }", input: {} };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toEqual(input.input);
            });
            test("empty array", async () => {
                const input = { code: "function test(input) { return input; }", input: [] };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toEqual(input.input);
            });
            test("nested object", async () => {
                const input = { code: "function test(input) { return input; }", input: { foo: { bar: "baz" } } };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toEqual(input.input);
            });
            test("nested array", async () => {
                const input = { code: "function test(input) { return input; }", input: [1, [2, 3], 4] };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toEqual(input.input);
            });
            test("object with circular reference", async () => {
                const circularObj: any = { foo: "bar" };
                circularObj.self = circularObj;

                const input = {
                    code: "function test(input) { return input; }",
                    input: circularObj,
                };

                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");

                // Check if the circular reference is preserved
                expect(result.output).toHaveProperty("foo", "bar");
                expect(result.output.self).toBe(result.output);
            });
            test("array with circular reference", async () => {
                const circularArray: any[] = ["foo", "bar"];
                circularArray.push(circularArray);

                const input = {
                    code: "function test(input) { return input; }",
                    input: circularArray,
                };

                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");

                // Check if the circular reference is preserved
                expect(result.output).toHaveLength(3);
                expect(result.output[0]).toBe("foo");
                expect(result.output[1]).toBe("bar");
                expect(result.output[2]).toBe(result.output);
            });
            test("object with Date", async () => {
                const input = { code: "function test(input) { return input; }", input: { date: new Date("2021-01-01T00:00:00Z") } };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output.date).toEqual(input.input.date);
            });
        });

        describe("spread inputs", () => {
            test("spread primitives", async () => {
                const input = { code: "function test(a, b, c) { return a + b + c; }", input: [1, 2, 3], shouldSpreadInput: true };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toBe(6);
            });
            test("spread objects", async () => {
                const input = { code: "function test(a, b, c) { return a.foo + b.bar + c.baz; }", input: [{ foo: 1 }, { bar: 2 }, { baz: 3 }], shouldSpreadInput: true };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toBe(6);
            });
        });

        describe("supported non-v8 types", () => {
            test("URL", async () => {
                const input = { code: "function createsURL() { return new URL('https://example.com?one=true'); }" };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                // @ts-ignore: Testing runtime scenario
                expect(result.output.href).toEqual("https://example.com?one=true");
                // @ts-ignore: Testing runtime scenario
                expect(result.output.protocol).toEqual("https:");
            });
            test("BigInt", async () => {
                const input = { code: "function createsBigInt() { return BigInt(9007199254740991); }" };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toEqual(BigInt(9007199254740991));
            });
            test("Buffer", async () => {
                const input = { code: "function createsBuffer() { return Buffer.from('hello, world!'); }" };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                // @ts-ignore: Testing runtime scenario
                expect(result.output.toString()).toEqual("hello, world!");
            });
            test("Uint8Array", async () => {
                const input = { code: "function createsUint8Array() { return new Uint8Array([0, 1, 2, 3, 4, 5]); }" };
                const result = await manager.runUserCode(input);

                expect(result).not.toHaveProperty("error");
                expect(result).toHaveProperty("output");
                expect(result.output).toEqual(new Uint8Array([0, 1, 2, 3, 4, 5]));
            });
        });
    });

    test("handles code execution errors", async () => {
        const input = { code: "function test() { throw new Error('Test error'); }", input: {} };
        const result = await manager.runUserCode(input);

        expect(result).toHaveProperty("error");
        expect(result.error).toEqual("Test error");
        expect(result).not.toHaveProperty("output");
    });

    test("handles worker exit", async () => {
        const input = { code: "function test() { process.exit(1); }", input: {} };
        const result = await manager.runUserCode(input);

        expect(result).toHaveProperty("error");
        expect(result.error).toEqual("Worker exited with code 1");
        expect(result).not.toHaveProperty("output");
    });

    describe("handles different function syntax", () => {
        test("function declaration", async () => {
            const input = { code: "function test() { return 'Hello'; }", input: {} };
            const result = await manager.runUserCode(input);

            expect(result).not.toHaveProperty("error");
            expect(result).toHaveProperty("output");
            expect(result.output).toEqual("Hello");
        });

        test("function expression", async () => {
            const input = { code: "const test = function() { return 'Hello'; };", input: {} };
            const result = await manager.runUserCode(input);

            expect(result).not.toHaveProperty("error");
            expect(result).toHaveProperty("output");
            expect(result.output).toEqual("Hello");
        });

        test("arrow function", async () => {
            const input = { code: "const test = () => 'Hello';", input: {} };
            const result = await manager.runUserCode(input);

            expect(result).not.toHaveProperty("error");
            expect(result).toHaveProperty("output");
            expect(result.output).toEqual("Hello");
        });

        test("async function declaration", async () => {
            const input = { code: "async function test() { return 'Hello'; }", input: {} };
            const result = await manager.runUserCode(input);

            expect(result).not.toHaveProperty("error");
            expect(result).toHaveProperty("output");
            expect(result.output).toEqual("Hello");
        });

        test("async arrow function", async () => {
            const input = { code: "const test = async () => 'Hello';", input: {} };
            const result = await manager.runUserCode(input);

            expect(result).not.toHaveProperty("error");
            expect(result).toHaveProperty("output");
            expect(result.output).toEqual("Hello");
        });
    });

    test("handles maximum code length", async () => {
        const longCode = "a".repeat(8193);
        const input = { code: `function test() { ${longCode} }`, input: {} };

        const result = await manager.runUserCode(input);

        expect(result).toEqual({ __type: "error", error: expect.any(String) });
    });

    test("handles missing function name", async () => {
        const input = { code: "const x = 5;", input: {} };

        const result = await manager.runUserCode(input);

        expect(result).toEqual({ __type: "error", error: expect.any(String) });
    });

    describe("handles malicious code", () => {
        test("infinite loop", async () => {
            const input = { code: "function test() { while (true) {} }", input: {} };
            const result = await manager.runUserCode(input);

            expect(result).toEqual({ __type: "error", error: expect.any(String) });
        });

        test("memory overload", async () => {
            const input = { code: "function test() { const arr = []; while (true) { arr.push(new Array(1_000_000)); } }", input: {} };
            const result = await manager.runUserCode(input);

            expect(result).toEqual({ __type: "error", error: expect.any(String) });
        });

        test("infinite recursion", async () => {
            const input = { code: "function test() { return test(); }", input: {} };
            const result = await manager.runUserCode(input);

            expect(result).toEqual({ __type: "error", error: expect.any(String) });
        });

        test("timeout from loop", async () => {
            const input = { code: "function test() { while (true) {} }", input: {} };
            const result = await manager.runUserCode(input);

            expect(result).toEqual({ __type: "error", error: expect.any(String) });
        });

        test("timeout from setTimout", async () => {
            const input = { code: "function test() { setTimeout(() => {}, 100000); }", input: {} };
            const result = await manager.runUserCode(input);

            expect(result).toEqual({ __type: "error", error: expect.any(String) });
        });

        test("cannot access fs", async () => {
            const input = { code: "function test() { require('fs'); }", input: {} };
            const result = await manager.runUserCode(input);

            expect(result).toEqual({ __type: "error", error: expect.any(String) });
        });

        test("cannot require modules", async () => {
            const input = { code: "function test() { require('http'); }", input: {} };
            const result = await manager.runUserCode(input);

            expect(result).toEqual({ __type: "error", error: expect.any(String) });
        });

        test("cannot shutdown main process", async () => {
            const input = { code: "function test() { process.exit(); }", input: {} };
            const result = await manager.runUserCode(input);

            expect(result).toEqual({ __type: "error", error: expect.any(String) });
        });
    });
});
