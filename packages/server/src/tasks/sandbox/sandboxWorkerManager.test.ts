/* eslint-disable @typescript-eslint/ban-ts-comment */
import { CodeLanguage } from "@local/shared";
import * as chai from "chai";
import { expect } from "chai";
import chaiAsPromised from "chai-as-promised";
import sinon from "sinon";
import { MB } from "./consts.js";
import { SandboxChildProcessManager } from "./sandboxWorkerManager.js";
import { RunUserCodeInput, RunUserCodeOutput } from "./types.js";

chai.use(chaiAsPromised);

type WorkerManager = SandboxChildProcessManager;

// Custom timeouts to speed up tests
const IDLE_TIMEOUT_MS = 100;
const JOB_TIMEOUT_MS = 300;
// Names of private properties in SandboxWorkerThreadManager.
// We define them here to make sure we access them with the correct names in the tests.
const Property = {
    memoryLimitBytes: "memoryLimitBytes",
    idleTimeoutMs: "idleTimeoutMs",
    jobTimeoutMs: "jobTimeoutMs",
    idleTimeoutHandle: "idleTimeoutHandle",
    jobTimeoutHandle: "jobTimeoutHandle",
    jobQueue: "jobQueue",
    status: "status",
    _isWorkerActive: "_isWorkerActive",
    _getWorkerId: "_getWorkerId",
} as const;

// Global variables for setup
let sandbox: sinon.SinonSandbox;

function assertResultIsError(result: RunUserCodeOutput, ...acceptedErrorMessages: string[]) {
    expect(result).to.have.property("__type", "error");
    expect(result).to.have.property("error");
    expect(acceptedErrorMessages.some(message => result.error?.includes(message))).to.be.true;
    expect(result).not.to.have.property("output");
}

function assertResultIsOutput(result: RunUserCodeOutput, output: unknown, deepEqual = true) {
    expect(result).to.have.property("__type", "output");
    expect(result).to.have.property("output");
    if (deepEqual) {
        expect(result.output).to.deep.equal(output);
    } else {
        expect(result.output).to.equal(output);
    }
    expect(result).not.to.have.property("error");
}

describe("/sandbox/managers", () => {
    let manager: WorkerManager;

    beforeEach(async () => {
        sandbox = sinon.createSandbox();
    });

    afterEach(async () => {
        sandbox.restore();
        await manager.terminate();
    });

    describe("SandboxChildProcessManager", () => {
        beforeEach(async () => {
            manager = new SandboxChildProcessManager({ idleTimeoutMs: IDLE_TIMEOUT_MS, jobTimeoutMs: JOB_TIMEOUT_MS });
        });

        afterEach(async () => {
            await manager.terminate();
        });

        it("constructor initializes with default values", () => {
            expect(manager).to.have.property(Property.memoryLimitBytes);
            expect(manager[Property.idleTimeoutMs]).to.equal(IDLE_TIMEOUT_MS);
            expect(manager[Property.jobTimeoutMs]).to.equal(JOB_TIMEOUT_MS);
            expect(manager[Property.idleTimeoutHandle]).to.be.null;
            expect(manager[Property.jobTimeoutHandle]).to.be.null;
            expect(manager[Property.jobQueue]).to.deep.equal([]);
            expect(manager[Property.status]).to.equal("Inactive");

            const isWorkerActive = manager[Property._isWorkerActive]();
            expect(isWorkerActive).to.be.false;
        });

        it("runUserCode starts a new worker if none exists", async () => {
            const input = {
                code: String.raw`function test() { return "Hello"; }`,
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            await manager.runUserCode(input);

            let isWorkerActive = manager[Property._isWorkerActive]();
            expect(isWorkerActive).to.be.true;

            await manager.terminate();
            isWorkerActive = manager[Property._isWorkerActive]();
            expect(isWorkerActive).to.be.false;
        });

        it("runUserCode reuses existing worker", async () => {
            const input1 = {
                code: String.raw`function test() { return "Hello"; }`,
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const input2 = {
                code: String.raw`function test() { return "Hello"; }`,
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            await manager.runUserCode(input1);
            const workerId1 = manager[Property._getWorkerId]();
            await manager.runUserCode(input2);
            const workerId2 = manager[Property._getWorkerId]();

            expect(workerId1).to.equal(workerId2);
        });

        it("runUserCode rejects unsupported languages", async () => {
            const input = {
                code: String.raw`function test() { return "Hello"; }`,
                codeLanguage: CodeLanguage.Python,
                input: {},
            };
            const result = await manager.runUserCode(input);

            assertResultIsError(result, `Unsupported code language: ${CodeLanguage.Python}`);
        });

        it("worker is terminated after idle timeout", async () => {
            const input = {
                code: String.raw`function test() { return "Hello"; }`,
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            await manager.runUserCode(input);
            const workerId1 = manager[Property._getWorkerId]();
            expect(workerId1).not.to.be.null;

            // Advance the clock by the idle timeout + a little extra to trigger the idle timeout
            await new Promise(resolve => setTimeout(resolve, manager[Property.idleTimeoutMs] + 50));

            const workerId2 = manager[Property._getWorkerId]();
            expect(workerId2).to.be.null;
        });

        describe("handle successful code execution", () => {
            it("without an input", async () => {
                const input = {
                    code: String.raw`function test() { return "Hello"; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, "Hello");
            });

            it("with a deconstructed input", async () => {
                const input = {
                    code: String.raw`function test({ input }) { 
    return input; 
}`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: { input: "Hello" },
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, "Hello");
            });

            it("without an output", async () => {
                const input = {
                    code: String.raw`function test() { return; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, undefined);
            });

            describe("primitive input types", () => {
                it("string", async () => {
                    const input = {
                        code: String.raw`function test(boop) { return boop; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: "hello, world! 🌍\"Chicken coop\" \nBeep boop",
                    };
                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, input.input);
                });

                it("negative number", async () => {
                    const input = {
                        code: String.raw`function test(input) { return Math.abs(input * 2); }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: -4.20,
                    };
                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, Math.abs(input.input * 2));
                });
                it("positive number", async () => {
                    const input = {
                        code: String.raw`function test(input) { return input * 100; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: 4.20,
                    };
                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, input.input * 100);
                });
                it("NaN", async () => {
                    const input = {
                        code: String.raw`function test(input) { return input; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: NaN,
                    };
                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, NaN);
                });
                it("Infinity", async () => {
                    const input = {
                        code: String.raw`function test(input) { return input; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: Infinity,
                    };
                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, Infinity);
                });
                it("boolean true", async () => {
                    const input = {
                        code: String.raw`function test(input) { return input; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: true,
                    };
                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, true);
                });
                it("boolean false", async () => {
                    const input = {
                        code: String.raw`function test(input) { return input; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: false,
                    };
                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, false);
                });
                it("null", async () => {
                    const input = {
                        code: String.raw`function test(input) { return input; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: null,
                    };
                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, null);
                });
                it("undefined", async () => {
                    const input = {
                        code: String.raw`function test(input) { return input; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: undefined,
                    };
                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, undefined);
                });
                it("BigInt", async () => {
                    const input = {
                        code: String.raw`function test(input) { return input; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: BigInt(9007199254740991),
                    };
                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, input.input);
                });
                it("Date", async () => {
                    const input = {
                        code: String.raw`function test(input) { return input; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: new Date("2021-01-01T00:00:00Z"),
                    };
                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, input.input);
                });
                it("regexp", async () => {
                    const input = {
                        code: String.raw`function test(input) { return input; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: /foo/gi,
                    };
                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, input.input);
                });
                it("Error", async () => {
                    const input = {
                        code: String.raw`function test(input) { return input; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: new Error("test error"),
                    };
                    const result = await manager.runUserCode(input);

                    // The result is still an output shape, but the attached output is an error
                    assertResultIsOutput(result, input.input);
                });
            });

            it("lots of calls in a row", async () => {
                const input = {
                    code: String.raw`function test() { return 'hello'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                for (let i = 0; i < 100; i++) {
                    // Don't await each call, as we want the queue to fill up
                    manager.runUserCode(input);
                }
                // Await the last call to finish
                await manager.runUserCode(input);

                // Check that the queue is empty
                expect(manager[Property.jobQueue]).to.deep.equal([]);
                expect(manager[Property.status]).to.equal("Idle");
            });

            it("URL - from input", async () => {
                const href = "https://example.com/";
                const input = {
                    code: String.raw`function test(input) { return input; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: new URL(href),
                };
                const result = await manager.runUserCode(input);

                expect((result.output as URL).href).to.equal(href);
            });

            it("URL - created in code", async () => {
                const href = "https://example.com/";
                const input = {
                    code: String.raw`function test() { return new URL("${href}"); }`,
                    codeLanguage: CodeLanguage.Javascript,
                };
                const result = await manager.runUserCode(input);

                expect((result.output as URL).href).to.equal(href);
            });

            // it("Buffer - from input", async () => {
            //     const input = {
            //         code: String.raw`function test(input) { return input; }`,
            //         codeLanguage: CodeLanguage.Javascript,
            //         input: Buffer.from("hello, world!"),
            //     };
            //     const result = await manager.runUserCode(input);

            //     assertResultIsOutput(result, input.input);
            // });

            // it("Buffer - created in code", async () => {
            //     const input = {
            //         code: String.raw`function test() { return Buffer.from('hello, world!'); }`,
            //         codeLanguage: CodeLanguage.Javascript,
            //     };
            //     const result = await manager.runUserCode(input);

            //     expect((result.output as Buffer).toString()).to.equal("hello, world!");
            // });

            it("Uint8Array", async () => {
                const input = {
                    code: String.raw`function test(input) { return input; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: new Uint8Array([0, 1, 2, 3, 4, 5]),
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, input.input);
            });

            it("Symbol", async () => {
                const input = {
                    code: String.raw`function test(input) { return input; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: Symbol("test"),
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, undefined); // Symbols are not serializable
            });

            it("function", async () => {
                const input = {
                    code: String.raw`function test(input) { return input; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    // eslint-disable-next-line @typescript-eslint/no-empty-function
                    input: () => { },
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, undefined); // Functions are not serializable
            });

            describe("Map and Set input types", () => {
                it("Map with primitive values", async () => {
                    const input = {
                        code: String.raw`function test(input) { return input; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: new Map<unknown, unknown>([["foo", "bar"], [42, "baz"]]),
                    };
                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, input.input);
                });
                it("Map with object values", async () => {
                    const input = {
                        code: String.raw`function test(input) { return input; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: new Map<unknown, unknown>([["foo", { bar: "baz" }]]),
                    };
                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, input.input);
                });
                it("Set with primitive values", async () => {
                    const input = {
                        code: String.raw`function test(input) { return input; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: new Set<unknown>(["foo", 42]),
                    };
                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, input.input);
                });
                it("Set with object values", async () => {
                    const input = {
                        code: String.raw`function test(input) { return input; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: new Set<unknown>([{ foo: "bar" }]),
                    };
                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, input.input);
                });
            });

            describe("object and array input types", () => {
                it("empty object", async () => {
                    const input = {
                        code: String.raw`function test(input) { return input; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: {},
                    };
                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, input.input);
                });
                it("empty array", async () => {
                    const input = {
                        code: String.raw`function test(input) { return input; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: [],
                    };
                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, input.input);
                });
                it("nested object", async () => {
                    const input = {
                        code: String.raw`function test(input) { return input; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: { foo: { bar: "baz" } },
                    };
                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, input.input);
                });
                it("nested array", async () => {
                    const input = {
                        code: String.raw`function test(input) { return input; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: [1, [2, 3], 4],
                    };
                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, input.input);
                });
                it("object with circular reference", async () => {
                    const circularObj: object = { foo: "bar" };
                    (circularObj as { self: object }).self = circularObj;

                    const input = {
                        code: String.raw`function test(input) { return input; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: circularObj,
                    };

                    const result = await manager.runUserCode(input);

                    assertResultIsOutput(result, input.input);

                    // Check if the circular reference is preserved
                    expect((result.output as { self: object }).self).to.equal((result.output as object));
                });
                it("array with circular reference", async () => {
                    const circularArray: unknown[] = ["foo", "bar"];
                    circularArray.push(circularArray);

                    const input = {
                        code: String.raw`function test(input) { return input; }`,
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
                        code: String.raw`function test(input) { return input; }`,
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
                        code: String.raw`function test(a, b, c) { return a + b + c; }`,
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
                        code: String.raw`function test(a, b, c) { return a.foo + b.bar + c.baz; }`,
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
                        code: String.raw`function createsURL() { return new URL('https://example.com/home?one=true'); }`,
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
                        code: String.raw`function createsBigInt() { return BigInt(9007199254740991); }`,
                        codeLanguage: CodeLanguage.Javascript,
                    };
                    const result = await manager.runUserCode(input);

                    expect(result).not.to.have.property("error");
                    expect(result).to.have.property("output");
                    expect(result.output).to.deep.equal(BigInt(9007199254740991));
                });
                // it("Buffer", async () => {
                //     const input = { code: String.raw`function createsBuffer() { return Buffer.from('hello, world!'); }` };
                //     const result = await manager.runUserCode(input);

                //     expect(result).not.to.have.property("error");
                //     expect(result).to.have.property("output");
                //     // @ts-ignore: Testing runtime scenario
                //     expect(result.output.toString()).to.deep.equal("hello, world!");
                // });
                it("Uint8Array", async () => {
                    const input = {
                        code: String.raw`function createsUint8Array() { return new Uint8Array([0, 1, 2, 3, 4, 5]); }`,
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
                code: String.raw`function test() { throw new Error('Test error'); }`,
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const result = await manager.runUserCode(input);

            assertResultIsError(result, "Test error");
        });

        it("'process' is not available", async () => {
            const input = {
                code: String.raw`function test() { process.exit(1); }`,
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };
            const result = await manager.runUserCode(input);

            assertResultIsError(result, "process is not defined");
        });

        describe("handles different function syntax", () => {
            it("function declaration - no input", async () => {
                const input = {
                    code: String.raw`function test() { return 'Hello'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, "Hello");
            });

            it("function declaration - with non-spread input", async () => {
                const input = {
                    code: String.raw`function test(input) { return input; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: { foo: "bar" },
                    shouldSpreadInput: false,
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, input.input);
            });

            it("function declaration - with spread input", async () => {
                const input = {
                    // Add the first 2 items passed in
                    code: String.raw`function test(input1, input2) { return input1 + input2; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: [1, 2],
                    shouldSpreadInput: true,
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, input.input[0] + input.input[1]);
            });

            it("async function declaration - no input", async () => {
                const input = {
                    code: String.raw`async function test() { return 'Hello'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, "Hello");
            });

            it("async function declaration - with non-spread input", async () => {
                const input = {
                    code: String.raw`async function test(input) { return input; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: { foo: "bar" },
                    shouldSpreadInput: false,
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, input.input);
            });

            it("async function declaration - with spread input", async () => {
                const input = {
                    code: String.raw`async function test(input) { return input; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: [1, 2],
                    shouldSpreadInput: true,
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, input.input[0]);
            });

            // NOT supported: function expressions, arrow functions
        });

        describe("handles comments", () => {
            it("handles block comments", async () => {
                const input = {
                    code: String.raw`/**
 * @returns {string}
 */
function test() {
    return "Hello";
}`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, "Hello");
            });

            it("handles slash comments", async () => {
                const input = {
                    code: String.raw`// function test() { return 'Hello'; }
function realTest() { return 'Hello2'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, "Hello2");
            });

            it("handles inline comments", async () => {
                const input = {
                    code: String.raw`function test() {
                           let result = "Hello";//comment
                           return result;
                       }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, "Hello");
            });

            it("handles comment syntax inside another comment", async () => {
                const input = {
                    code: String.raw`function test() {
                           let result = "Hello";//comment with // slashes
                           return result;
                       }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, "Hello");
            });
        });

        describe("handles quotes", () => {
            it("handles backticks", async () => {
                // NOTE: Since the code has backticks, we have to put it in a variable first 
                // and then use String.raw with the variable.
                const code = `function test123() {
    const value = "Jeff";
    const result = \`Hello \${value}\`;
    return result;
}`;
                const input = {
                    code: String.raw`${code}`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, "Hello Jeff");
            });

            it("handles single quotes", async () => {
                const input = {
                    code: String.raw`function test() { return 'Hello'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, "Hello");
            });
        });

        describe("handles escape characters", () => {
            it("\\t", async () => {
                const input = {
                    code: String.raw`function test() {
    const str = "Hello\tworld what is\tup?";
    const result = str.split("\t");
    return result;
}`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, ["Hello", "world what is", "up?"]);
            });

            it("\\n", async () => {
                const input = {
                    code: String.raw`function test() {
    const str = "Hello\nworld\twhat is\nup?";
    const result = str.split("\n");
    return result;
}`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, ["Hello", "world\twhat is", "up?"]);
            });
        });

        describe("handles non-ascii characters", () => {
            it("Chinese characters", async () => {
                const input = {
                    code: String.raw`function test() {
    const str = "你好，世界！";
    return str;
}`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, "你好，世界！");
            });

            it("Japanese characters", async () => {
                const input = {
                    code: String.raw`function test() {
    const str = "こんにちは、世界！";
    return str;
}`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, "こんにちは、世界！");
            });

            it("Emoji characters", async () => {
                const input = {
                    code: String.raw`function test() {
    const str = "👋🌍";
    return str;
}`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, "👋🌍");
            });

            it("Accented characters", async () => {
                const input = {
                    code: String.raw`function test() {
    const str = "café";
    return str;
}`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, "café");
            });

            it("Removing accents", async () => {
                const input = {
                    code: String.raw`function test() {
    const str = "café";
    const result = str.normalize('NFD').replace(/[\u0300-\u036f]/g, '');
    return result;
}`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, "cafe");
            });
        });

        it("handles maximum code length", async () => {
            const longCode = "a".repeat(8193);
            const input = {
                code: String.raw`function test() { ${longCode} }`,
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };

            const result = await manager.runUserCode(input);

            assertResultIsError(result, "Code is too long");
        });

        it("handles missing function name", async () => {
            const input = {
                code: String.raw`const x = 5;`,
                codeLanguage: CodeLanguage.Javascript,
                input: {},
            };

            const result = await manager.runUserCode(input);

            assertResultIsError(result, "Function name not found");
        });

        describe("handles malicious code", () => {
            it("infinite loop", async () => {
                const input = {
                    code: String.raw`function test() { while (true) {} }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsError(result, "Execution timed out", "Worker exited with code 1", `Job timed out after ${JOB_TIMEOUT_MS} ms`);
            });

            it("infinite recursion", async () => {
                const input = {
                    code: String.raw`function test() { return test(); }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsError(result, "Maximum call stack size exceeded");
            });

            it("timeout from loop", async () => {
                const input = {
                    code: String.raw`function test() { while (true) {} }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsError(result, "Execution timed out", "Worker exited with code 1", `Job timed out after ${JOB_TIMEOUT_MS} ms`);
            });

            it("attempting to use setTimout", async () => {
                const input = {
                    code: String.raw`function test() { setTimeout(() => {}, 100000); }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsError(result, "setTimeout is not defined");
            });

            it("cannot access fs", async () => {
                const input1 = {
                    code: String.raw`function test() { require('fs'); }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result1 = await manager.runUserCode(input1);

                assertResultIsError(result1, "require is not defined");

                const input2 = {
                    code: String.raw`function test() { import('fs'); }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result2 = await manager.runUserCode(input2);

                assertResultIsError(result2, "Not supported");
            });

            it("cannot require modules", async () => {
                const input = {
                    code: String.raw`function test() { require('http'); }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsError(result, "require is not defined");
            });

            it("cannot shutdown main process", async () => {
                const input = {
                    code: String.raw`function test() { process.exit(); }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const result = await manager.runUserCode(input);

                assertResultIsError(result, "process is not defined");
            });

            it("cannot make network requests", async () => {
                const input = {
                    code: String.raw`function test() { return fetch('https://example.com'); }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                const result = await manager.runUserCode(input);

                assertResultIsError(result, "fetch is not defined");
            });
        });

        describe("Worker Thread Isolation", () => {
            it("cannot modify global object", async () => {
                const input1 = {
                    code: String.raw`function test() { global.newProperty = 'test'; return 'done'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const input2 = {
                    code: String.raw`function test() { return global.newProperty || 'did not modify global object'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                await manager.runUserCode(input1);
                const result = await manager.runUserCode(input2);

                assertResultIsOutput(result, "did not modify global object");
            });

            it("cannot modify URL constructor", async () => {
                const input1 = {
                    code: String.raw`function test() { URL.prototype.newMethod = () => 'test'; return 'done'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const input2 = {
                    code: String.raw`function test() { return new URL('http://example.com').newMethod?.(); }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                await manager.runUserCode(input1);
                const result = await manager.runUserCode(input2);

                assertResultIsOutput(result, undefined);
            });

            it("cannot add new global functions", async () => {
                const input1 = {
                    code: String.raw`function test() { global.newFunction = () => 'test'; return 'done'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const input2 = {
                    code: String.raw`function test() { return global.newFunction?.(); }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                await manager.runUserCode(input1);
                const result = await manager.runUserCode(input2);

                assertResultIsOutput(result, undefined);
            });

            it("cannot persist variables between runs", async () => {
                const input1 = {
                    code: String.raw`let persistentVar = 'test'; function test() { return persistentVar; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const input2 = {
                    code: String.raw`function test() { return typeof persistentVar; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                await manager.runUserCode(input1);
                const result = await manager.runUserCode(input2);

                assertResultIsOutput(result, "undefined");
            });

            it("cannot modify Object prototype - add method", async () => {
                const input1 = {
                    code: String.raw`function test() { Object.prototype.newMethod = () => 'test'; return 'done'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const input2 = {
                    code: String.raw`function test() { return {}.newMethod?.(); }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                await manager.runUserCode(input1);
                const result = await manager.runUserCode(input2);

                assertResultIsOutput(result, undefined);
            });

            it("cannot modify Object prototype - have it return a string", async () => {
                const input1 = {
                    code: String.raw`function test() { class MyObject extends Object { toString() { return 'test'; } } Object.prototype = MyObject.prototype; return 'done'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const input2 = {
                    code: String.raw`function test() { return {}.toString(); }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                await manager.runUserCode(input1);
                const result = await manager.runUserCode(input2);

                assertResultIsOutput(result, "[object Object]");
            });

            it("cannot access constructor to escape sandbox", async () => {
                const input = {
                    code: String.raw`function test() { return this.constructor.constructor('return process')().exit; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                const result = await manager.runUserCode(input);

                assertResultIsError(result, "process is not defined");
            });

            it("maintains proper typeof for built-ins", async () => {
                const input = {
                    code: String.raw`function test() { return { url: typeof URL, obj: typeof Object, func: typeof Function }; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                const result = await manager.runUserCode(input);

                assertResultIsOutput(result, { url: "function", obj: "function", func: "function" });
            });

            it("doesn't break if 'global' is not defined", async () => {
                const input1 = {
                    code: String.raw`function test() { delete global; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const input2 = {
                    code: String.raw`function test() { return 'Hello'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                await manager.runUserCode(input1);
                const result = await manager.runUserCode(input2);

                assertResultIsOutput(result, "Hello");
            });

            it("cannot call code from previous run", async () => {
                const input1 = {
                    code: String.raw`function first() { return 'first'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const input2 = {
                    code: String.raw`function second() { return first(); }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                await manager.runUserCode(input1);
                const result = await manager.runUserCode(input2);

                assertResultIsError(result, "first is not defined");
            });

            it("works fine after previous test throws error", async () => {
                const input1 = {
                    code: String.raw`function test() { throw new Error('Test error'); }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const input2 = {
                    code: String.raw`function test() { return 'Hello'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                await manager.runUserCode(input1);
                const result = await manager.runUserCode(input2);

                assertResultIsOutput(result, "Hello");
            });

            it("works fine after previous test has a memory overload", async () => {
                const input1 = {
                    code: String.raw`function test() { const arr = []; while (true) { arr.push(new Array(1_000_000)); } }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const input2 = {
                    code: String.raw`function test() { return 'Hello'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                await manager.runUserCode(input1);
                const result = await manager.runUserCode(input2);

                assertResultIsOutput(result, "Hello");
            });

            it("works fine after previous test times out", async () => {
                const input1 = {
                    code: String.raw`function test() { while (true) {} }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const input2 = {
                    code: String.raw`function test() { return 'Hello'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                await manager.runUserCode(input1);
                const result = await manager.runUserCode(input2);

                assertResultIsOutput(result, "Hello");
            });

            it("cannot modify Array constructor", async () => {
                const input1 = {
                    code: String.raw`function test() { Array.prototype.myMethod = () => 'test'; return 'done'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const input2 = {
                    code: String.raw`function test() { return [].myMethod?.(); }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                await manager.runUserCode(input1);
                const result = await manager.runUserCode(input2);

                assertResultIsOutput(result, undefined);
            });

            it("cannot modify Date constructor", async () => {
                const input1 = {
                    code: String.raw`function test() { Date.prototype.myMethod = () => 'test'; return 'done'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const input2 = {
                    code: String.raw`function test() { return new Date().myMethod?.(); }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                await manager.runUserCode(input1);
                const result = await manager.runUserCode(input2);

                assertResultIsOutput(result, undefined);
            });

            it("cannot access sensitive environment variables", async () => {
                const input = {
                    code: String.raw`function test() { return process.env; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                const result = await manager.runUserCode(input);

                assertResultIsError(result, "process is not defined");
            });

            it("cannot add sensitive environment variables", async () => {
                const input1 = {
                    code: String.raw`function test() { process.env.NEW_SECRET = 'injected'; return 'done'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const input2 = {
                    code: String.raw`function test() { return process.env.NEW_SECRET; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                await manager.runUserCode(input1);
                const result = await manager.runUserCode(input2);

                assertResultIsError(result, "process is not defined");
            });
        });

        describe("Memory leaks", () => {
            it("from memory overloads", async function runMemoryLeakTest() {
                // Increase the timeout to 10 seconds, since these tests are time-consuming
                this.timeout(10_000);

                // Record initial memory usage before any tests
                const initialMemoryUsage = process.memoryUsage().heapUsed;
                const memoryThreshold = initialMemoryUsage * 1.2; // Allow 20% growth for test overhead
                const iterations = 50;

                // Run the iterations
                for (let i = 0; i < iterations; i++) {
                    const input = {
                        code: String.raw`function test() { const arr = []; while (true) { arr.push(new Array(1_000_000)); } }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: {},
                    };

                    // Execute the code and expect it to fail due to memory limits
                    const result = await manager.runUserCode(input);

                    // Verify that the worker thread correctly handles memory overload
                    expect(result).to.have.property("__type", "error");
                    expect(result).not.to.have.property("output");
                }

                const finalMemoryUsage = process.memoryUsage().heapUsed;

                const initialMemoryUsageMB = Math.floor(initialMemoryUsage / MB);
                const finalMemoryUsageMB = Math.floor(finalMemoryUsage / MB);
                const memoryIncreasePercentage = Math.floor(
                    ((finalMemoryUsageMB - initialMemoryUsageMB) / initialMemoryUsageMB) * 100,
                );

                // Assert that memory usage hasn’t grown beyond the threshold
                expect(finalMemoryUsage).to.be.lessThan(
                    memoryThreshold,
                    `Memory usage grew from ${initialMemoryUsageMB} MB to ${finalMemoryUsageMB} MB (${memoryIncreasePercentage}%), exceeding threshold of ${memoryThreshold} MB. Possible memory leak detected.`,
                );
            });

            it("from normal usage", async () => {
                // Record initial memory usage before any tests
                const initialMemoryUsage = process.memoryUsage().heapUsed;
                const memoryThreshold = initialMemoryUsage * 1.2; // Allow 20% growth for test overhead
                const iterations = 200;

                // Run the iterations
                for (let i = 0; i < iterations; i++) {
                    const input = {
                        code: String.raw`function test(a, b, c) { return a + b + c; }`,
                        codeLanguage: CodeLanguage.Javascript,
                        input: [1, 2, 3],
                        shouldSpreadInput: true,
                    };

                    const result = await manager.runUserCode(input);

                    expect(result).to.have.property("__type", "output");
                    expect(result).to.have.property("output").to.equal(1 + 2 + 3);
                }

                // Make sure the worker is terminated to accurately measure memory usage
                await manager.terminate();

                const finalMemoryUsage = process.memoryUsage().heapUsed;

                const initialMemoryUsageMB = Math.floor(initialMemoryUsage / MB);
                const finalMemoryUsageMB = Math.floor(finalMemoryUsage / MB);
                const memoryIncreasePercentage = Math.floor(
                    ((finalMemoryUsageMB - initialMemoryUsageMB) / initialMemoryUsageMB) * 100,
                );

                // Assert that memory usage hasn’t grown beyond the threshold
                expect(finalMemoryUsage).to.be.lessThan(
                    memoryThreshold,
                    `Memory usage grew from ${initialMemoryUsageMB} MB to ${finalMemoryUsageMB} MB (${memoryIncreasePercentage}%), exceeding threshold of ${memoryThreshold} MB. Possible memory leak detected.`,
                );
            });

            it("from a mixture of passing and failing jobs", async function runPassFailLeakTest() {
                // Increase the timeout to 10 seconds, since these tests are time-consuming
                this.timeout(10_000);

                // Record initial memory usage before any tests
                const initialMemoryUsage = process.memoryUsage().heapUsed;
                const memoryThreshold = initialMemoryUsage * 1.2; // Allow 20% growth for test overhead
                const iterations = 200;

                const passingInput = {
                    code: String.raw`function test() { return 'Hello'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const failingInput = {
                    code: String.raw`function test() { throw new Error('Test error'); }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };
                const memoryOverloadInput = {
                    code: String.raw`function test() { const arr = []; while (true) { arr.push(new Array(1_000_000)); } }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                // Put iterations in a randomized list
                const inputs: RunUserCodeInput[] = [];
                for (let i = 0; i < iterations; i++) {
                    const num = Math.random();
                    if (num < 0.5) {
                        inputs.push(passingInput);
                    } else if (num < 0.75) {
                        inputs.push(failingInput);
                    } else {
                        inputs.push(memoryOverloadInput);
                    }
                }

                // Run the iterations
                for (let i = 0; i < iterations; i++) {
                    await manager.runUserCode(inputs[i]);
                }

                // Make sure the worker is terminated to accurately measure memory usage
                await manager.terminate();

                const finalMemoryUsage = process.memoryUsage().heapUsed;

                const initialMemoryUsageMB = Math.floor(initialMemoryUsage / MB);
                const finalMemoryUsageMB = Math.floor(finalMemoryUsage / MB);
                const memoryIncreasePercentage = Math.floor(
                    ((finalMemoryUsageMB - initialMemoryUsageMB) / initialMemoryUsageMB) * 100,
                );

                // Assert that memory usage hasn’t grown beyond the threshold
                expect(finalMemoryUsage).to.be.lessThan(
                    memoryThreshold,
                    `Memory usage grew from ${initialMemoryUsageMB} MB to ${finalMemoryUsageMB} MB (${memoryIncreasePercentage}%), exceeding threshold of ${memoryThreshold} MB. Possible memory leak detected.`,
                );
            });

            it("terminating worker thread after each job", async function runTerminateLeakTest() {
                // Increase the timeout to 10 seconds, since these tests are time-consuming
                this.timeout(10_000);

                // Record initial memory usage before any tests
                const initialMemoryUsage = process.memoryUsage().heapUsed;
                const memoryThreshold = initialMemoryUsage * 1.2; // Allow 20% growth for test overhead
                const iterations = 100;

                const input = {
                    code: String.raw`function test() { return 'Hello'; }`,
                    codeLanguage: CodeLanguage.Javascript,
                    input: {},
                };

                // Run the iterations
                for (let i = 0; i < iterations; i++) {
                    await manager.runUserCode(input);
                    // Terminate the worker after each job so that it's forced to create a new one
                    await manager.terminate();
                }

                // Make sure the worker is terminated to accurately measure memory usage
                await manager.terminate();

                const finalMemoryUsage = process.memoryUsage().heapUsed;

                const initialMemoryUsageMB = Math.floor(initialMemoryUsage / MB);
                const finalMemoryUsageMB = Math.floor(finalMemoryUsage / MB);
                const memoryIncreasePercentage = Math.floor(
                    ((finalMemoryUsageMB - initialMemoryUsageMB) / initialMemoryUsageMB) * 100,
                );

                // Assert that memory usage hasn’t grown beyond the threshold
                expect(finalMemoryUsage).to.be.lessThan(
                    memoryThreshold,
                    `Memory usage grew from ${initialMemoryUsageMB} MB to ${finalMemoryUsageMB} MB (${memoryIncreasePercentage}%), exceeding threshold of ${memoryThreshold} MB. Possible memory leak detected.`,
                );
            });
        });

        describe("performance", () => {
            it("runs 1000 jobs in under 10 seconds", async function runPerformanceTest() {
                this.timeout(10_000);
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
});
