/* eslint-disable @typescript-eslint/ban-ts-comment */
import { CodeLanguage } from "@local/shared";
import * as chai from "chai";
import { expect } from "chai";
import chaiAsPromised from "chai-as-promised";
import esmock from "esmock";
import sinon from "sinon";
import * as worker_threads from "worker_threads";
import { Worker as originalWorker } from "worker_threads";
import { RunUserCodeOutput } from "./types.js";
import { WorkerThreadManager } from "./workerThreadManager.js";

chai.use(chaiAsPromised);

// Define an interface for spied worker methods
interface SpiedWorker extends worker_threads.Worker {
    on: worker_threads.Worker["on"] & sinon.SinonSpy;
    postMessage: worker_threads.Worker["postMessage"] & sinon.SinonSpy;
    terminate: worker_threads.Worker["terminate"] & sinon.SinonSpy;
    removeListener: worker_threads.Worker["removeListener"] & sinon.SinonSpy;
}

// Names of private properties in WorkerThreadManager.
// We define them here to make sure we access them with the correct names in the tests.
const Property = {
    memoryLimitBytes: "memoryLimitBytes",
    idleTimeoutMs: "idleTimeoutMs",
    jobTimeoutMs: "jobTimeoutMs",
    worker: "worker",
    idleTimeoutHandle: "idleTimeoutHandle",
    jobTimeoutHandle: "jobTimeoutHandle",
    jobQueue: "jobQueue",
    workerStarting: "workerStarting",
    isProcessing: "isProcessing",
} as const;

// Global variables for setup
let sandbox: sinon.SinonSandbox;
let workerInstances: SpiedWorker[] = [];

// List of handlers we expect to be added to the worker
const workerOnHandlers = ["message", "error"];

// Helper functions to check worker creation and job posting
function checkNumWorkersCreated(workersCreated: number) {
    expect(workerInstances).to.have.lengthOf(workersCreated);
    workerOnHandlers.forEach((handler) => {
        workerInstances.forEach((worker) => {
            expect(worker.on.calledWith(handler, sinon.match.func)).to.be.true;
        });
    });
    return workerInstances;
}

function checkJobsSentToWorker(jobsSent: number, worker: SpiedWorker) {
    expect(worker.postMessage.callCount).to.equal(jobsSent);
}

describe("WorkerThreadManager", () => {
    let manager: WorkerThreadManager;
    let clock: sinon.SinonFakeTimers;
    let WorkerSpyClass: typeof originalWorker;

    beforeEach(async () => {
        sandbox = sinon.createSandbox();
        clock = sandbox.useFakeTimers();
        workerInstances = [];

        // Define a subclass of Worker with spied methods
        WorkerSpyClass = class extends originalWorker {
            constructor(filename, options) {
                super(filename, options);
                // Replace methods with spies that wrap the originals
                this.on = sandbox.spy(this.on);
                this.postMessage = sandbox.spy(this.postMessage);
                this.terminate = sandbox.spy(this.terminate);
                this.removeListener = sandbox.spy(this.removeListener);
                workerInstances.push(this as unknown as SpiedWorker);
            }
        };

        // Use esmock to import WorkerThreadManager with mocked worker_threads
        const mockedModules = await esmock("./workerThreadManager.js", {
            "worker_threads": {
                Worker: WorkerSpyClass,
            },
        });
        manager = new mockedModules.WorkerThreadManager();
    });

    afterEach(() => {
        sandbox.restore();
        workerInstances = [];
        manager.terminate();
    });

    it("constructor initializes with default values", () => {
        expect(manager).to.have.property(Property.memoryLimitBytes);
        expect(manager).to.have.property(Property.idleTimeoutMs);
        expect(manager).to.have.property(Property.jobTimeoutMs);
        expect(manager[Property.worker]).to.be.null;
        expect(manager[Property.idleTimeoutHandle]).to.be.null;
        expect(manager[Property.jobTimeoutHandle]).to.be.null;
        expect(manager[Property.jobQueue]).to.deep.equal([]);
        expect(manager[Property.workerStarting]).to.be.false;
        expect(manager[Property.isProcessing]).to.be.false;
    });

    it("runUserCode starts a new worker if none exists", async () => {
        const input = {
            code: "function test() { return \"Hello\"; }",
            codeLanguage: CodeLanguage.Javascript,
            input: {},
        };
        await manager.runUserCode(input);

        const [worker] = checkNumWorkersCreated(1);
        checkJobsSentToWorker(1, worker);
    });

    it("runUserCode reuses existing worker", async () => {
        const input1 = {
            code: "function test() { return \"Hello\"; }",
            codeLanguage: CodeLanguage.Javascript,
            input: {},
        };
        const input2 = {
            code: "function test() { return \"Hello\"; }",
            codeLanguage: CodeLanguage.Javascript,
            input: {},
        };
        await manager.runUserCode(input1);
        await manager.runUserCode(input2);

        const [worker] = checkNumWorkersCreated(1);
        checkJobsSentToWorker(2, worker);
    });

    it("runUserCode rejects unsupported languages", async () => {
        const input = {
            code: "function test() { return \"Hello\"; }",
            codeLanguage: CodeLanguage.Python,
            input: {},
        };
        await expect(manager.runUserCode(input)).to.be.rejectedWith(`Unsupported code language: ${CodeLanguage.Python}`);
    });

    it("worker is terminated after idle timeout", async () => {
        const input = {
            code: "function test() { return \"Hello\"; }",
            codeLanguage: CodeLanguage.Javascript,
            input: {},
        };
        await manager.runUserCode(input);

        const [worker] = checkNumWorkersCreated(1);
        expect(manager[Property.worker]).not.to.be.null;

        // Advance the clock by the idle timeout + a little extra to trigger the idle timeout
        clock.tick(manager[Property.idleTimeoutMs] + 100);
        await manager.waitForTermination();

        expect(worker.terminate.called).to.be.true;
        expect(manager[Property.worker]).to.be.null;
    });

    describe("handle successful code execution", () => {
        it("without an input", async () => {
            const input = {
                code: "function test() { return \"Hello\"; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const result = await manager.runUserCode(input);

            expect(result).not.to.have.property("error");
            expect(result).to.have.property("output");
            expect(result.output).to.equal("Hello");
        });

        it("without an output", async () => {
            const input = {
                code: "function test() { return; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const result = await manager.runUserCode(input);

            expect(result).not.to.have.property("error");
            expect(result).to.have.property("output");
            expect(result.output).to.be.undefined;
        });

        describe("primitive input types", () => {
            it("string", async () => {
                const input = {
                    code: "function test(boop) { return boop; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: "hello, world! 🌍\"Chicken coop\" \nBeep boop",
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.equal(input.input);
            });

            it("negative number", async () => {
                const input = {
                    code: "function test(input) { return Math.abs(input * 2); }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: -4.20,
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.equal(Math.abs(input.input * 2));
            });
            it("positive number", async () => {
                const input = {
                    code: "function test(input) { return input * 100; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: 4.20,
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.deep.equal(input.input * 100);
            });
            it("NaN", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: NaN,
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.be.NaN;
            });
            it("Infinity", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: Infinity,
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.equal(Infinity);
            });
            it("boolean true", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: true,
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.equal(true);
            });
            it("boolean false", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: false,
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.equal(false);
            });
            it("null", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: null,
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.be.null;
            });
            it("undefined", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: undefined,
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.be.undefined;
            });
            it("BigInt", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: BigInt(9007199254740991),
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.equal(input.input);
            });
            it("Date", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: new Date("2021-01-01T00:00:00Z"),
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.deep.equal(input.input);
            });
            it("regexp", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: /foo/gi,
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.deep.equal(input.input);
            });
            it("Error", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: new Error("test error"),
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                // @ts-ignore: Testing runtime scenario
                expect(result.output.message).to.equal(input.input.message);
            });
            it("URL", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: new URL("https://example.com"),
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                // @ts-ignore: Testing runtime scenario
                expect(result.output.href).to.equal(input.input.href);
            });
            // it("Buffer", async () => {
            //     const input = { code: "function test(input) { return input; }", input: Buffer.from("hello, world!") };
            //     const result = await manager.runUserCode(input);

            //     expect(result).not.to.have.property("error");
            //     expect(result).to.have.property("output");
            //     // @ts-ignore: Testing runtime scenario
            //     expect(result.output.toString()).to.equal(input.input.toString());
            // });
            it("Uint8Array", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: new Uint8Array([0, 1, 2, 3, 4, 5]),
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.deep.equal(input.input);
            });
            it("Symbol", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: Symbol("test"),
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.equal(undefined); // Symbols are not serializable
            });
            it("function", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    // eslint-disable-next-line @typescript-eslint/no-empty-function
                    input: () => { },
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.equal(undefined); // Functions are not serializable
            });
        });

        describe("Map and Set input types", () => {
            it("Map with primitive values", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: new Map<any, any>([["foo", "bar"], [42, "baz"]]),
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                // @ts-ignore: Testing runtime scenario
                expect(result.output.size).to.equal(input.input.size);
                // @ts-ignore: Testing runtime scenario
                expect(result.output.get("foo")).to.equal("bar");
                // @ts-ignore: Testing runtime scenario
                expect(result.output.get(42)).to.equal("baz");
            });
            it("Map with object values", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: new Map<any, any>([["foo", { bar: "baz" }]]),
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                // @ts-ignore: Testing runtime scenario
                expect(result.output.size).to.equal(input.input.size);
                // @ts-ignore: Testing runtime scenario
                expect(result.output.get("foo")).to.deep.equal({ bar: "baz" });
            });
            it("Set with primitive values", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: new Set<any>(["foo", 42]),
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                // @ts-ignore: Testing runtime scenario
                expect(result.output.size).to.equal(input.input.size);
                // @ts-ignore: Testing runtime scenario
                expect(result.output.has("foo")).to.equal(true);
                // @ts-ignore: Testing runtime scenario
                expect(result.output.has(42)).to.equal(true);
            });
            it("Set with object values", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: new Set<any>([{ foo: "bar" }]),
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                // @ts-ignore: Testing runtime scenario
                expect(result.output.size).to.equal(input.input.size);
                // `has` won't work with objects, so we need to check if the object is in the set
                // @ts-ignore: Testing runtime scenario
                expect([...result.output][0]).to.deep.equal({ foo: "bar" });
            });
        });

        describe("object and array input types", () => {
            it("empty object", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.deep.equal(input.input);
            });
            it("empty array", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: [],
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.deep.equal(input.input);
            });
            it("nested object", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: { foo: { bar: "baz" } },
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.deep.equal(input.input);
            });
            it("nested array", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: [1, [2, 3], 4],
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.deep.equal(input.input);
            });
            it("object with circular reference", async () => {
                const circularObj: any = { foo: "bar" };
                circularObj.self = circularObj;

                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: circularObj,
                };

                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");

                // Check if the circular reference is preserved
                expect(result.output).to.have.property("foo", "bar");
                // @ts-ignore Testing runtime scenario
                expect(result.output.self).to.equal(result.output);
            });
            it("array with circular reference", async () => {
                const circularArray: unknown[] = ["foo", "bar"];
                circularArray.push(circularArray);

                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: circularArray,
                };

                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");

                // Check if the circular reference is preserved
                expect(result.output).to.have.lengthOf(3);
                // @ts-ignore Testing runtime scenario
                expect(result.output[0]).to.equal("foo");
                // @ts-ignore Testing runtime scenario
                expect(result.output[1]).to.equal("bar");
                // @ts-ignore Testing runtime scenario
                expect(result.output[2]).to.equal(result.output);
            });
            it("object with Date", async () => {
                const input = {
                    code: "function test(input) { return input; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: { date: new Date("2021-01-01T00:00:00Z") },
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                // @ts-ignore Testing runtime scenario
                expect(result.output.date).to.deep.equal(input.input.date);
            });
        });

        describe("spread inputs", () => {
            it("spread primitives", async () => {
                const input = {
                    code: "function test(a, b, c) { return a + b + c; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: [1, 2, 3],
                    shouldSpreadInput: true,
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.equal(6);
            });
            it("spread objects", async () => {
                const input = {
                    code: "function test(a, b, c) { return a.foo + b.bar + c.baz; }",
                    codeLanguage: CodeLanguage.Javascript,
                    input: [{ foo: 1 }, { bar: 2 }, { baz: 3 }],
                    shouldSpreadInput: true,
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.equal(6);
            });
        });

        describe("supported non-v8 types", () => {
            it("URL", async () => {
                const input = {
                    code: "function createsURL() { return new URL('https://example.com/home?one=true'); }",
                    codeLanguage: CodeLanguage.Javascript,
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                // @ts-ignore: Testing runtime scenario
                expect(result.output.href).to.deep.equal("https://example.com/home?one=true");
                // @ts-ignore: Testing runtime scenario
                expect(result.output.protocol).to.deep.equal("https:");
            });
            it("BigInt", async () => {
                const input = {
                    code: "function createsBigInt() { return BigInt(9007199254740991); }",
                    codeLanguage: CodeLanguage.Javascript,
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.deep.equal(BigInt(9007199254740991));
            });
            // it("Buffer", async () => {
            //     const input = { code: "function createsBuffer() { return Buffer.from('hello, world!'); }" };
            //     const result = await manager.runUserCode(input);

            //     expect(result).not.to.have.property("error");
            //     expect(result).to.have.property("output");
            //     // @ts-ignore: Testing runtime scenario
            //     expect(result.output.toString()).to.deep.equal("hello, world!");
            // });
            it("Uint8Array", async () => {
                const input = {
                    code: "function createsUint8Array() { return new Uint8Array([0, 1, 2, 3, 4, 5]); }",
                    codeLanguage: CodeLanguage.Javascript,
                };
                const result = await manager.runUserCode(input);

                expect(result).not.to.have.property("error");
                expect(result).to.have.property("output");
                expect(result.output).to.deep.equal(new Uint8Array([0, 1, 2, 3, 4, 5]));
            });
        });
    });

    it("handles code execution errors", async () => {
        const input = {
            code: "function test() { throw new Error('Test error'); }",
            codeLanguage: CodeLanguage.Javascript,
            input: {},
        };
        const result = await manager.runUserCode(input);

        expect(result).to.have.property("error");
        expect(result.error).to.equal("Test error");
        expect(result).not.to.have.property("output");
    });

    it("'process' is not available", async () => {
        const input = {
            code: "function test() { process.exit(1); }",
            codeLanguage: CodeLanguage.Javascript,
            input: {},
        };
        const result = await manager.runUserCode(input);

        expect(result).to.have.property("__type", "error");
        expect(result).to.have.property("error").to.equal("process is not defined");
    });

    describe("handles different function syntax", () => {
        it("function declaration - no input", async () => {
            const input = {
                code: "function test() { return 'Hello'; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const result = await manager.runUserCode(input);

            expect(result).not.to.have.property("error");
            expect(result).to.have.property("output");
            expect(result.output).to.equal("Hello");
        });

        it("function declaration - with non-spread input", async () => {
            const input = {
                code: "function test(input) { return input; }",
                codeLanguage: CodeLanguage.Javascript,
                input: { foo: "bar" },
                shouldSpreadInput: false,
            };
            const result = await manager.runUserCode(input);

            expect(result).not.to.have.property("error");
            expect(result).to.have.property("output");
            expect(result.output).to.deep.equal(input.input);
        });

        it("function declaration - with spread input", async () => {
            const input = {
                // Add the first 2 items passed in
                code: "function test(input1, input2) { return input1 + input2; }",
                codeLanguage: CodeLanguage.Javascript,
                input: [1, 2],
                shouldSpreadInput: true,
            };
            const result = await manager.runUserCode(input);

            expect(result).not.to.have.property("error");
            expect(result).to.have.property("output");
            // Result should be the sum of the first 2 items passed in
            expect(result.output).to.deep.equal(input.input[0] + input.input[1]);
        });

        it("async function declaration - no input", async () => {
            const input = {
                code: "async function test() { return 'Hello'; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const result = await manager.runUserCode(input);

            expect(result).not.to.have.property("error");
            expect(result).to.have.property("output");
            expect(result.output).to.equal("Hello");
        });

        it("async function declaration - with non-spread input", async () => {
            const input = {
                code: "async function test(input) { return input; }",
                codeLanguage: CodeLanguage.Javascript,
                input: { foo: "bar" },
                shouldSpreadInput: false,
            };
            const result = await manager.runUserCode(input);

            expect(result).not.to.have.property("error");
            expect(result).to.have.property("output");
            expect(result.output).to.deep.equal(input.input);
        });

        it("async function declaration - with spread input", async () => {
            const input = {
                code: "async function test(input) { return input; }",
                codeLanguage: CodeLanguage.Javascript,
                input: [1, 2],
                shouldSpreadInput: true,
            };
            const result = await manager.runUserCode(input);

            expect(result).not.to.have.property("error");
            expect(result).to.have.property("output");
            // Since we're spreading on a single input, we expect to return the first item
            expect(result.output).to.deep.equal(input.input[0]);
        });

        // NOT supported: function expressions, arrow functions
    });

    it("handles maximum code length", async () => {
        const longCode = "a".repeat(8193);
        const input = {
            code: `function test() { ${longCode} }`,
            codeLanguage: CodeLanguage.Javascript,
            input: {},
        };

        const result = await manager.runUserCode(input);

        expect(result).to.have.property("__type", "error");
        expect(result).to.have.property("error").to.equal("Code is too long");
    });

    it("handles missing function name", async () => {
        const input = {
            code: "const x = 5;",
            codeLanguage: CodeLanguage.Javascript,
            input: {},
        };

        const result = await manager.runUserCode(input);

        expect(result).to.have.property("__type", "error");
        expect(result).to.have.property("error").to.equal("Function name not found");
    });

    describe("handles malicious code", () => {
        it("infinite loop", async () => {
            const input = {
                code: "function test() { while (true) {} }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const result = await manager.runUserCode(input);

            expect(result).to.have.property("__type", "error");
            expect(result).to.have.property("error").to.equal("Execution timed out");
        });

        it("infinite recursion", async () => {
            const input = {
                code: "function test() { return test(); }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const result = await manager.runUserCode(input);

            expect(result).to.have.property("__type", "error");
            expect(result).to.have.property("error").to.equal("Maximum call stack size exceeded");
        });

        it("timeout from loop", async () => {
            const input = {
                code: "function test() { while (true) {} }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const result = await manager.runUserCode(input);

            expect(result).to.have.property("__type", "error");
            expect(result).to.have.property("error").to.equal("Execution timed out");
        });

        it("attempting to use setTimout", async () => {
            const input = {
                code: "function test() { setTimeout(() => {}, 100000); }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const result = await manager.runUserCode(input);

            expect(result).to.have.property("__type", "error");
            expect(result).to.have.property("error").to.equal("setTimeout is not defined");
        });

        it("cannot access fs", async () => {
            const input1 = {
                code: "function test() { require('fs'); }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const result1 = await manager.runUserCode(input1);

            expect(result1).to.have.property("__type", "error");
            expect(result1).to.have.property("error").to.equal("require is not defined");

            const input2 = {
                code: "function test() { import('fs'); }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const result2 = await manager.runUserCode(input2);

            expect(result2).to.have.property("__type", "error");
            expect(result2).to.have.property("error").to.equal("Not supported");
        });

        it("cannot require modules", async () => {
            const input = {
                code: "function test() { require('http'); }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const result = await manager.runUserCode(input);

            expect(result).to.have.property("__type", "error");
            expect(result).to.have.property("error").to.equal("require is not defined");
        });

        it("cannot shutdown main process", async () => {
            const input = {
                code: "function test() { process.exit(); }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const result = await manager.runUserCode(input);

            expect(result).to.have.property("__type", "error");
            expect(result).to.have.property("error").to.equal("process is not defined");
        });

        it("cannot make network requests", async () => {
            const input = {
                code: "function test() { return fetch('https://example.com'); }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };

            const result = await manager.runUserCode(input);

            expect(result).to.have.property("__type", "error");
            expect(result).to.have.property("error").to.equal("fetch is not defined");
        });
    });

    describe("Worker Thread Isolation", () => {
        it("cannot modify global object", async () => {
            const input1 = {
                code: "function test() { global.newProperty = 'test'; return 'done'; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const input2 = {
                code: "function test() { return global.newProperty || 'did not modify global object'; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };

            await manager.runUserCode(input1);
            const result = await manager.runUserCode(input2);

            expect(result).to.deep.equal({ __type: "output", output: "did not modify global object" });
        });

        it("cannot modify URL constructor", async () => {
            const input1 = {
                code: "function test() { URL.prototype.newMethod = () => 'test'; return 'done'; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const input2 = {
                code: "function test() { return new URL('http://example.com').newMethod?.(); }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };

            await manager.runUserCode(input1);
            const result = await manager.runUserCode(input2);

            expect(result).to.deep.equal({ __type: "output", output: undefined });
        });

        it("cannot add new global functions", async () => {
            const input1 = {
                code: "function test() { global.newFunction = () => 'test'; return 'done'; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const input2 = {
                code: "function test() { return global.newFunction?.(); }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };

            await manager.runUserCode(input1);
            const result = await manager.runUserCode(input2);

            expect(result).to.deep.equal({ __type: "output", output: undefined });
        });

        it("cannot persist variables between runs", async () => {
            const input1 = {
                code: "let persistentVar = 'test'; function test() { return persistentVar; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const input2 = {
                code: "function test() { return typeof persistentVar; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };

            await manager.runUserCode(input1);
            const result = await manager.runUserCode(input2);

            expect(result).to.deep.equal({ __type: "output", output: "undefined" });
        });

        it("cannot modify Object prototype - add method", async () => {
            const input1 = {
                code: "function test() { Object.prototype.newMethod = () => 'test'; return 'done'; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const input2 = {
                code: "function test() { return {}.newMethod?.(); }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };

            await manager.runUserCode(input1);
            const result = await manager.runUserCode(input2);

            expect(result).to.deep.equal({ __type: "output", output: undefined });
        });

        it("cannot modify Object prototype - have it return a string", async () => {
            const input1 = {
                code: "function test() { class MyObject extends Object { toString() { return 'test'; } } Object.prototype = MyObject.prototype; return 'done'; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const input2 = {
                code: "function test() { return {}.toString(); }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };

            await manager.runUserCode(input1);
            const result = await manager.runUserCode(input2);

            expect(result).to.deep.equal({ __type: "output", output: "[object Object]" });
        });

        it("cannot access constructor to escape sandbox", async () => {
            const input = {
                code: "function test() { return this.constructor.constructor('return process')().exit; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };

            const result = await manager.runUserCode(input);

            expect(result).to.have.property("__type", "error");
            expect(result).not.to.have.property("output");
        });

        it("maintains proper typeof for built-ins", async () => {
            const input = {
                code: "function test() { return { url: typeof URL, obj: typeof Object, func: typeof Function }; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };

            const result = await manager.runUserCode(input);

            expect(result).to.deep.equal({
                __type: "output",
                output: { url: "function", obj: "function", func: "function" },
            });
        });

        it("doesn't break if 'global' is not defined", async () => {
            const input1 = {
                code: "function test() { delete global; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const input2 = {
                code: "function test() { return 'Hello'; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };

            await manager.runUserCode(input1);
            const result = await manager.runUserCode(input2);

            expect(result).not.to.have.property("error");
            expect(result).to.have.property("output");
            expect(result.output).to.deep.equal("Hello");
        });

        it("cannot call code from previous run", async () => {
            const input1 = {
                code: "function first() { return 'first'; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const input2 = {
                code: "function second() { return first(); }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };

            await manager.runUserCode(input1);
            const result = await manager.runUserCode(input2);
            expect(result).to.have.property("__type", "error");
            expect(result).not.to.have.property("output");
        });

        it("works fine after previous test throws error", async () => {
            const input1 = {
                code: "function test() { throw new Error('Test error'); }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const input2 = {
                code: "function test() { return 'Hello'; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };

            await manager.runUserCode(input1);
            const result = await manager.runUserCode(input2);
            expect(result).not.to.have.property("error");
            expect(result).to.have.property("output");
            expect(result.output).to.deep.equal("Hello");
        });

        it("works fine after previous test has a memory overload", async () => {
            const input1 = {
                code: "function test() { const arr = []; while (true) { arr.push(new Array(1_000_000)); } }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const input2 = {
                code: "function test() { return 'Hello'; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };

            await manager.runUserCode(input1);
            const result = await manager.runUserCode(input2);

            expect(result).not.to.have.property("error");
            expect(result).to.have.property("output");
            expect(result.output).to.deep.equal("Hello");
        });

        it("works fine after previous test times out", async () => {
            const input1 = {
                code: "function test() { while (true) {} }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const input2 = {
                code: "function test() { return 'Hello'; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };

            try {
                console.log("yeet boopies 1");
                await manager.runUserCode(input1);
            } catch (error) {
                console.log("yeet boopies 2", error);
            }
            const result = await manager.runUserCode(input2);
            console.log("yeet boopies 3", result);

            expect(result).not.to.have.property("error");
            expect(result).to.have.property("output");
            expect(result.output).to.deep.equal("Hello");
        });

        it("cannot modify Array constructor", async () => {
            const input1 = {
                code: "function test() { Array.prototype.myMethod = () => 'test'; return 'done'; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const input2 = {
                code: "function test() { return [].myMethod?.(); }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };

            await manager.runUserCode(input1);
            const result = await manager.runUserCode(input2);
            expect(result).to.deep.equal({ __type: "output", output: undefined });
        });

        it("cannot modify Date constructor", async () => {
            const input1 = {
                code: "function test() { Date.prototype.myMethod = () => 'test'; return 'done'; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const input2 = {
                code: "function test() { return new Date().myMethod?.(); }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };

            await manager.runUserCode(input1);
            const result = await manager.runUserCode(input2);
            expect(result).to.deep.equal({ __type: "output", output: undefined });
        });

        it("cannot access sensitive environment variables", async () => {
            const input = {
                code: "function test() { return process.env; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };

            const result = await manager.runUserCode(input);
            expect(result).to.have.property("__type", "error");
            expect(result).not.to.have.property("output");
        });

        it("cannot add sensitive environment variables", async () => {
            const input1 = {
                code: "function test() { process.env.NEW_SECRET = 'injected'; return 'done'; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const input2 = {
                code: "function test() { return process.env.NEW_SECRET; }",
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };

            await manager.runUserCode(input1);
            const result = await manager.runUserCode(input2);
            expect(result).to.have.property("__type", "error");
            expect(result).not.to.have.property("output");
        });
    });

    // describe("WorkerThreadManager memory leaks", () => {
    //     it("from memory overloads", async () => {
    //         // Record initial memory usage before any tests
    //         const initialMemoryUsage = process.memoryUsage().heapUsed;
    //         const memoryThreshold = initialMemoryUsage * 1.2; // Allow 20% growth for test overhead
    //         const iterations = 20;
    //         const MB = 1024 * 1024; // For converting bytes to MB

    //         // Run the iterations
    //         for (let i = 0; i < iterations; i++) {
    //             const input = {
    //                 code: "function test() { const arr = []; while (true) { arr.push(new Array(1_000_000)); } }",
    //                 codeLanguage: CodeLanguage.Javascript,
    //                 input: {},
    //             };

    //             // Execute the code and expect it to fail due to memory limits
    //             const result = await manager.runUserCode(input);
    //             console.log("yeet result", i, result);

    //             // Verify that the worker thread correctly handles memory overload
    //             expect(result).to.have.property("__type", "error");
    //             expect(result).not.to.have.property("output");
    //         }

    //         const finalMemoryUsage = process.memoryUsage().heapUsed;

    //         const initialMemoryUsageMB = Math.floor(initialMemoryUsage / MB);
    //         const finalMemoryUsageMB = Math.floor(finalMemoryUsage / MB);
    //         const memoryIncreasePercentage = Math.floor(
    //             ((finalMemoryUsageMB - initialMemoryUsageMB) / initialMemoryUsageMB) * 100,
    //         );

    //         // Assert that memory usage hasn’t grown beyond the threshold
    //         expect(finalMemoryUsage).to.be.lessThan(
    //             memoryThreshold,
    //             `Memory usage grew from ${initialMemoryUsageMB} MB to ${finalMemoryUsageMB} MB (${memoryIncreasePercentage}%), exceeding threshold of ${memoryThreshold} MB. Possible memory leak detected.`,
    //         );
    //     });

    //     it("from normal usage", async () => {
    //         // Record initial memory usage before any tests
    //         const initialMemoryUsage = process.memoryUsage().heapUsed;
    //         const memoryThreshold = initialMemoryUsage * 1.2; // Allow 20% growth for test overhead
    //         const iterations = 20;
    //         const MB = 1024 * 1024; // For converting bytes to MB

    //         // Run the iterations
    //         for (let i = 0; i < iterations; i++) {
    //             const input = {
    //                 code: "function test(a, b, c) { return a + b + c; }",
    //                 codeLanguage: CodeLanguage.Javascript,
    //                 input: [1, 2, 3],
    //                 shouldSpreadInput: true,
    //             };

    //             const result = await manager.runUserCode(input);

    //             expect(result).to.have.property("__type", "output");
    //             expect(result).to.have.property("output").to.equal(1 + 2 + 3);
    //         }

    //         // Check that there is only one worker, and it received all the jobs
    //         expect(manager[Property.worker]).not.to.be.null;
    //         checkJobsSentToWorker(iterations, manager[Property.worker] as SpiedWorker);

    //         // Wait for the idle timeout to ensure the worker thread terminates
    //         clock.tick(manager[Property.idleTimeoutMs] + 1000);
    //         await manager.waitForTermination();
    //         expect(manager[Property.worker]).to.be.null;

    //         const finalMemoryUsage = process.memoryUsage().heapUsed;

    //         const initialMemoryUsageMB = Math.floor(initialMemoryUsage / MB);
    //         const finalMemoryUsageMB = Math.floor(finalMemoryUsage / MB);
    //         const memoryIncreasePercentage = Math.floor(
    //             ((finalMemoryUsageMB - initialMemoryUsageMB) / initialMemoryUsageMB) * 100,
    //         );

    //         // Assert that memory usage hasn’t grown beyond the threshold
    //         expect(finalMemoryUsage).to.be.lessThan(
    //             memoryThreshold,
    //             `Memory usage grew from ${initialMemoryUsageMB} MB to ${finalMemoryUsageMB} MB (${memoryIncreasePercentage}%), exceeding threshold of ${memoryThreshold} MB. Possible memory leak detected.`,
    //         );
    //     });

    //     //TODO memory tests still randomly throw core dumps sometimes
    //     //TODO from new worker threads (continually wait for termination, send new job, etc.)
    // });

    //TODO tests hang after this for some reason
    describe("performance", () => {
        it("runs 1000 jobs in under 10 seconds", async function runPerformanceTest() {
            this.timeout(10_000);
            const manager = new WorkerThreadManager();
            const code = "function test(count) { return `Hello, world! ${count}`; }";

            const promises: Promise<RunUserCodeOutput>[] = [];
            for (let i = 0; i < 1000; i++) {
                promises.push(manager.runUserCode({
                    code,
                    codeLanguage: CodeLanguage.Javascript,
                    input: i,
                    shouldSpreadInput: false,
                }));
            }
            // NOTE: The test will automatically fail if it takes longer than 10 seconds
            const results = await Promise.all(promises);

            // Check that the results are correct
            results.forEach((result, index) => {
                expect(result).to.have.property("__type", "output");
                expect(result.output).to.equal(`Hello, world! ${index}`);
            });
        });
    });
});
