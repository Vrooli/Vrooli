import { expect } from "chai";
import { describe } from "mocha";
import sinon from "sinon";
import { ResourceVersion } from "../../api/types.js";
import { CodeLanguage } from "../../consts/index.js";
import { CodeVersionConfig, CodeVersionConfigObject } from "./code.js";
import { LATEST_CONFIG_VERSION } from "./utils.js";

type PartialCodeVersion = Pick<ResourceVersion, "codeLanguage" | "content" | "data">;

// Sample CodeVersion objects with filled-out content fields
const sampleCodeVersionSpread: PartialCodeVersion = {
    codeLanguage: CodeLanguage.Javascript,
    content: "function sum(a, b, c) { return a + b + c; }",
    data: null,  // Will be set in tests
};

const sampleCodeVersionDirect: PartialCodeVersion = {
    codeLanguage: CodeLanguage.Javascript,
    content: "function greet({ name, age }) { return `Hello, ${name}, age ${age}`; }",
    data: null,  // Will be set in tests
};

describe("CodeVersionConfig", () => {
    let consoleErrorStub: sinon.SinonStub;

    before(async () => {
        consoleErrorStub = sinon.stub(console, "error");
    });

    beforeEach(() => {
        consoleErrorStub.resetHistory();
    });

    after(() => {
        consoleErrorStub.restore();
    });

    describe("deserialization", () => {
        describe("inputs", () => {
            describe("spreads", () => {
                it("number array", () => {
                    const validData: CodeVersionConfigObject = {
                        __version: LATEST_CONFIG_VERSION,
                        inputConfig: {
                            inputSchema: {
                                type: "array",
                                items: { type: "number" },
                                minItems: 3,
                                maxItems: 3,
                            },
                            shouldSpread: true,
                        },
                        outputConfig: [],
                        testCases: [],
                    };
                    const validDataString = JSON.stringify(validData);
                    const codeVersion = { ...sampleCodeVersionSpread, data: validDataString } as CodeVersion;
                    const config = CodeVersionConfig.deserialize(codeVersion, console);
                    expect(config.__version).to.equal(LATEST_CONFIG_VERSION);
                    expect(config.inputConfig).to.deep.equal(validData.inputConfig);
                });

                it("mixed type array", () => {
                    const validData: CodeVersionConfigObject = {
                        __version: LATEST_CONFIG_VERSION,
                        inputConfig: {
                            inputSchema: {
                                type: "array",
                                items: [
                                    { type: "number" },
                                    { type: "string" },
                                    { type: "null" },
                                ],
                                minItems: 3,
                                maxItems: 3,
                            },
                            shouldSpread: true,
                        },
                        outputConfig: [],
                        testCases: [],
                    };
                    const validDataString = JSON.stringify(validData);
                    const codeVersion = { ...sampleCodeVersionSpread, data: validDataString } as CodeVersion;
                    const config = CodeVersionConfig.deserialize(codeVersion, console);
                    expect(config.__version).to.equal(LATEST_CONFIG_VERSION);
                    expect(config.inputConfig).to.deep.equal(validData.inputConfig);
                });
            });

            describe("directs", () => {
                it("simple object", () => {
                    const validDirectData = {
                        __version: LATEST_CONFIG_VERSION,
                        inputConfig: {
                            inputSchema: {
                                type: "object",
                                properties: {
                                    name: { type: "string" },
                                    age: { type: "number" },
                                },
                                required: ["name"],
                            },
                            shouldSpread: false,
                        },
                        outputConfig: [],
                        testCases: [],
                    };
                    const validDirectDataString = JSON.stringify(validDirectData);
                    const codeVersion = { ...sampleCodeVersionDirect, data: validDirectDataString } as CodeVersion;
                    const config = CodeVersionConfig.deserialize(codeVersion, console);
                    expect(config.__version).to.equal(LATEST_CONFIG_VERSION);
                    expect(config.inputConfig).to.deep.equal(validDirectData.inputConfig);
                });

                it("nested object", () => {
                    const validDirectData = {
                        __version: LATEST_CONFIG_VERSION,
                        inputConfig: {
                            inputSchema: {
                                type: "object",
                                properties: {
                                    name: { type: "string" },
                                    children: {
                                        type: "array",
                                        items: {
                                            type: "object",
                                            properties: {
                                                name: { type: "string" },
                                                isAdult: { type: "boolean" },
                                            },
                                            required: ["name", "isAdult"],
                                        },
                                    },
                                },
                                required: ["name"],
                            },
                            shouldSpread: false,
                        },
                    };
                    const validDirectDataString = JSON.stringify(validDirectData);
                    const codeVersion = { ...sampleCodeVersionDirect, data: validDirectDataString } as CodeVersion;
                    const config = CodeVersionConfig.deserialize(codeVersion, console);
                    expect(config.__version).to.equal(LATEST_CONFIG_VERSION);
                    expect(config.inputConfig).to.deep.equal(validDirectData.inputConfig);
                });
            });

            it("null input", () => {
                const codeVersion = { ...sampleCodeVersionDirect, data: null } as CodeVersion;
                const config = CodeVersionConfig.deserialize(codeVersion, console);
                const defaultConfig = CodeVersionConfig.default(codeVersion);
                expect(config.__version).to.equal(defaultConfig.__version);
                expect(config.inputConfig).to.deep.equal(defaultConfig.inputConfig);
            });
        });

        describe("outputs", () => {
            describe("single schema", () => {
                it("string", () => {
                    const validData = {
                        __version: LATEST_CONFIG_VERSION,
                        inputConfig: CodeVersionConfig.defaultInputConfig(),
                        outputConfig: { type: "string" },
                        testCases: [],
                    } as CodeVersionConfigObject;
                    const validDataString = JSON.stringify(validData);
                    const codeVersion = { ...sampleCodeVersionSpread, data: validDataString } as CodeVersion;
                    const config = CodeVersionConfig.deserialize(codeVersion, console);
                    expect(config.__version).to.equal(LATEST_CONFIG_VERSION);
                    expect(config.inputConfig).to.deep.equal(validData.inputConfig);
                    expect(config.outputConfig).to.deep.equal({ type: "string" });
                });


                it("object with properties", () => {
                    const validData: CodeVersionConfigObject = {
                        __version: LATEST_CONFIG_VERSION,
                        inputConfig: CodeVersionConfig.defaultInputConfig(),
                        outputConfig: {
                            type: "object",
                            properties: {
                                result: { type: "number" },
                                message: { type: "string" },
                            },
                            required: ["result"],
                        },
                        testCases: [],
                    } as CodeVersionConfigObject;
                    const validDataString = JSON.stringify(validData);
                    const codeVersion = { ...sampleCodeVersionSpread, data: validDataString } as CodeVersion;
                    const config = CodeVersionConfig.deserialize(codeVersion, console);
                    expect(config.__version).to.equal(LATEST_CONFIG_VERSION);
                    expect(config.inputConfig).to.deep.equal(validData.inputConfig);
                    expect(config.outputConfig).to.deep.equal(validData.outputConfig);
                });
            });

            describe("multiple schemas", () => {
                it("string and object", () => {
                    const validData: CodeVersionConfigObject = {
                        __version: LATEST_CONFIG_VERSION,
                        inputConfig: CodeVersionConfig.defaultInputConfig(),
                        outputConfig: [
                            { type: "string" },
                            {
                                type: "object",
                                properties: { error: { type: "string" } },
                                required: ["error"],
                            },
                        ],
                        testCases: [],
                    } as CodeVersionConfigObject;
                    const validDataString = JSON.stringify(validData);
                    const codeVersion = { ...sampleCodeVersionSpread, data: validDataString } as CodeVersion;
                    const config = CodeVersionConfig.deserialize(codeVersion, console);
                    expect(config.__version).to.equal(LATEST_CONFIG_VERSION);
                    expect(config.inputConfig).to.deep.equal(validData.inputConfig);
                    expect(config.outputConfig).to.deep.equal(validData.outputConfig);
                });
            });

            it("no outputConfig provided", () => {
                const validData: Partial<CodeVersionConfigObject> = {
                    __version: LATEST_CONFIG_VERSION,
                    inputConfig: CodeVersionConfig.defaultInputConfig(),
                    // outputConfig is omitted
                };
                const validDataString = JSON.stringify(validData);
                const codeVersion = { ...sampleCodeVersionSpread, data: validDataString } as CodeVersion;
                const config = CodeVersionConfig.deserialize(codeVersion, console);
                expect(config.__version).to.equal(LATEST_CONFIG_VERSION);
                expect(config.inputConfig).to.deep.equal(validData.inputConfig);
                expect(config.outputConfig).to.deep.equal(CodeVersionConfig.defaultOutputConfig());
            });
        });

        it("correctly falls back to defaults on invalid data", () => {
            const codeVersion = { ...sampleCodeVersionSpread, data: "invalid json" } as CodeVersion;
            const config = CodeVersionConfig.deserialize(codeVersion, console);
            const defaultConfig = CodeVersionConfig.default(codeVersion);
            expect(config.__version).to.equal(defaultConfig.__version);
            expect(config.inputConfig).to.deep.equal(defaultConfig.inputConfig);
        });
    });

    describe("serialization", () => {
        it("spread input and array output", () => {
            const validData: CodeVersionConfigObject = {
                __version: LATEST_CONFIG_VERSION,
                inputConfig: { inputSchema: { type: "array", items: { type: "number" } }, shouldSpread: true },
                outputConfig: [{ type: "string" }],
                testCases: [],
            };
            const validDataString = JSON.stringify(validData);
            const codeVersion = { ...sampleCodeVersionSpread, data: validDataString } as CodeVersion;
            const config = CodeVersionConfig.deserialize(codeVersion, console);
            const serialized = config.serialize("json");
            const expected = JSON.stringify(config.export());
            expect(serialized).to.equal(expected);
            expect(expected).to.equal(validDataString);
        });

        it("direct input and single schema output", () => {
            const validData: CodeVersionConfigObject = {
                __version: LATEST_CONFIG_VERSION,
                inputConfig: { inputSchema: { type: "object" }, shouldSpread: false },
                outputConfig: [{ type: "string" }],
                testCases: [],
            };
            const validDataString = JSON.stringify(validData);
            const codeVersion = { ...sampleCodeVersionDirect, data: validDataString } as CodeVersion;
            const config = CodeVersionConfig.deserialize(codeVersion, console);
            const serialized = config.serialize("json");
            const expected = JSON.stringify(config.export());
            expect(serialized).to.equal(expected);
            expect(expected).to.equal(validDataString);
        });
    });

    describe("runTestCases", () => {
        it("runs a passing test case correctly with spread input", async () => {
            const configData: CodeVersionConfigObject = {
                __version: LATEST_CONFIG_VERSION,
                inputConfig: {
                    inputSchema: { type: "array", items: { type: "number" } },
                    shouldSpread: true,
                },
                outputConfig: [{ type: "number" }],
                testCases: [
                    {
                        description: "Simple addition",
                        input: [1, 2, 3],
                        expectedOutput: 6,
                    },
                ],
            };
            const codeVersion: PartialCodeVersion = {
                codeLanguage: CodeLanguage.Javascript,
                content: "function sum(a, b, c) { return a + b + c; }",
                data: JSON.stringify(configData),
            };
            const config = CodeVersionConfig.deserialize(codeVersion as CodeVersion, console);

            async function runSandbox() {
                return { output: 6 };
            }
            const results = await config.runTestCases(runSandbox);
            expect(results).to.deep.equal([
                {
                    description: "Simple addition",
                    passed: true,
                },
            ]);
        });

        it("runs a failing test case correctly with spread input", async () => {
            const configData: CodeVersionConfigObject = {
                __version: LATEST_CONFIG_VERSION,
                inputConfig: {
                    inputSchema: { type: "array", items: { type: "number" } },
                    shouldSpread: true,
                },
                outputConfig: [{ type: "number" }],
                testCases: [
                    {
                        description: "Simple addition",
                        input: [1, 2, 3],
                        expectedOutput: 7,  // Expecting 7, but runSandbox returns 6
                    },
                ],
            };
            const codeVersion: PartialCodeVersion = {
                codeLanguage: CodeLanguage.Javascript,
                content: "function sum(a, b, c) { return a + b + c; }",
                data: JSON.stringify(configData),
            };
            const config = CodeVersionConfig.deserialize(codeVersion as CodeVersion, console);

            async function runSandbox() {
                return { output: 6 };
            }
            const results = await config.runTestCases(runSandbox);
            expect(results).to.deep.equal([
                {
                    description: "Simple addition",
                    passed: false,
                    actualOutput: 6,
                },
            ]);
        });

        it("handles errors in runSandbox correctly", async () => {
            const configData: CodeVersionConfigObject = {
                __version: LATEST_CONFIG_VERSION,
                inputConfig: {
                    inputSchema: { type: "array", items: { type: "number" } },
                    shouldSpread: true,
                },
                outputConfig: [{ type: "number" }],
                testCases: [
                    {
                        description: "Error case",
                        input: [1, 2, 3],
                        expectedOutput: 6,
                    },
                ],
            };
            const codeVersion: PartialCodeVersion = {
                codeLanguage: CodeLanguage.Javascript,
                content: "function sum(a, b, c) { throw new Error('Test error'); }",
                data: JSON.stringify(configData),
            };
            const config = CodeVersionConfig.deserialize(codeVersion as CodeVersion, console);

            async function runSandbox() {
                return { error: "Test error" };
            }
            const results = await config.runTestCases(runSandbox);
            expect(results).to.deep.equal([
                {
                    description: "Error case",
                    passed: false,
                    error: "Test error",
                },
            ]);
        });

        it("runs multiple test cases correctly", async () => {
            const configData: CodeVersionConfigObject = {
                __version: LATEST_CONFIG_VERSION,
                inputConfig: {
                    inputSchema: { type: "array", items: { type: "number" } },
                    shouldSpread: true,
                },
                outputConfig: [{ type: "number" }],
                testCases: [
                    {
                        description: "First test",
                        input: [1, 2, 3],
                        expectedOutput: 6,
                    },
                    {
                        description: "Second test",
                        input: [4, 5, 6],
                        expectedOutput: 15,
                    },
                ],
            };
            const codeVersion: PartialCodeVersion = {
                codeLanguage: CodeLanguage.Javascript,
                content: "function sum(a, b, c) { return a + b + c; }",
                data: JSON.stringify(configData),
            };
            const config = CodeVersionConfig.deserialize(codeVersion as CodeVersion, console);

            async function runSandbox(params: { input?: unknown }) {
                const input = params.input as number[];
                const sum = input.reduce((a, b) => a + b, 0);
                return { output: sum };
            }
            const results = await config.runTestCases(runSandbox);
            expect(results).to.deep.equal([
                {
                    description: "First test",
                    passed: true,
                },
                {
                    description: "Second test",
                    passed: true,
                },
            ]);
        });

        it("runs test cases with direct input correctly", async () => {
            const configData: CodeVersionConfigObject = {
                __version: LATEST_CONFIG_VERSION,
                inputConfig: {
                    inputSchema: {
                        type: "object",
                        properties: { a: { type: "number" }, b: { type: "number" } },
                    },
                    shouldSpread: false,
                },
                outputConfig: [{ type: "number" }],
                testCases: [
                    {
                        description: "Object input",
                        input: { a: 1, b: 2 },
                        expectedOutput: 3,
                    },
                ],
            };
            const codeVersion: PartialCodeVersion = {
                codeLanguage: CodeLanguage.Javascript,
                content: "function add(obj) { return obj.a + obj.b; }",
                data: JSON.stringify(configData),
            };
            const config = CodeVersionConfig.deserialize(codeVersion as CodeVersion, console);

            async function runSandbox(params: { input?: unknown }) {
                const input = params.input as { a: number; b: number };
                return { output: input.a + input.b };
            }
            const results = await config.runTestCases(runSandbox);
            expect(results).to.deep.equal([
                {
                    description: "Object input",
                    passed: true,
                },
            ]);
        });

        it("runs test cases with complex object output correctly", async () => {
            const configData: CodeVersionConfigObject = {
                __version: LATEST_CONFIG_VERSION,
                inputConfig: {
                    inputSchema: { type: "object", properties: { name: { type: "string" } } },
                    shouldSpread: false,
                },
                outputConfig: [{ type: "object", properties: { greeting: { type: "string" } } }],
                testCases: [
                    {
                        description: "Greeting test",
                        input: { name: "Alice" },
                        expectedOutput: { greeting: "Hello, Alice" },
                    },
                ],
            };
            const codeVersion: PartialCodeVersion = {
                codeLanguage: CodeLanguage.Javascript,
                content: "function greet(obj) { return { greeting: `Hello, ${obj.name}` }; }",
                data: JSON.stringify(configData),
            };
            const config = CodeVersionConfig.deserialize(codeVersion as CodeVersion, console);

            async function runSandbox(params: { input?: unknown }) {
                const input = params.input as { name: string };
                return { output: { greeting: `Hello, ${input.name}` } };
            }
            const results = await config.runTestCases(runSandbox);
            expect(results).to.deep.equal([
                {
                    description: "Greeting test",
                    passed: true,
                },
            ]);
        });

        it("detects failing test case with object output", async () => {
            const configData: CodeVersionConfigObject = {
                __version: LATEST_CONFIG_VERSION,
                inputConfig: {
                    inputSchema: { type: "object", properties: { name: { type: "string" } } },
                    shouldSpread: false,
                },
                outputConfig: [{ type: "object", properties: { greeting: { type: "string" } } }],
                testCases: [
                    {
                        description: "Greeting test",
                        input: { name: "Alice" },
                        expectedOutput: { greeting: "Hi, Alice" },  // Expecting "Hi", but getting "Hello"
                    },
                ],
            };
            const codeVersion: PartialCodeVersion = {
                codeLanguage: CodeLanguage.Javascript,
                content: "function greet(obj) { return { greeting: `Hello, ${obj.name}` }; }",
                data: JSON.stringify(configData),
            };
            const config = CodeVersionConfig.deserialize(codeVersion as CodeVersion, console);

            async function runSandbox(params: { input?: unknown }) {
                const input = params.input as { name: string };
                return { output: { greeting: `Hello, ${input.name}` } };
            }
            const results = await config.runTestCases(runSandbox);
            expect(results).to.deep.equal([
                {
                    description: "Greeting test",
                    passed: false,
                    actualOutput: { greeting: "Hello, Alice" },
                },
            ]);
        });
    });
});
