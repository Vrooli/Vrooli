import { describe, it, expect, vi, beforeAll, beforeEach, afterAll } from "vitest";
import { type ResourceVersion } from "../../api/types.js";
import { CodeLanguage } from "../../consts/index.js";
import { CodeVersionConfig, type CodeVersionConfigObject, type JsonSchema } from "./code.js";
import { LATEST_CONFIG_VERSION } from "./utils.js";

// Type for constructing the argument to CodeVersionConfig.parse
type VersionInputForParse = Pick<ResourceVersion, "codeLanguage" | "config">;

const DEFAULT_TEST_CONTENT = "function main() { return 'test'; }";

describe("CodeVersionConfig", () => {
    let consoleErrorSpy: any;

    beforeAll(async () => {
        consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    });

    beforeEach(() => {
        consoleErrorSpy.mockClear();
    });

    afterAll(() => {
        consoleErrorSpy.mockRestore();
    });

    describe("deserialization", () => {
        it("correctly falls back to defaults on invalid data (e.g. non-JSON string)", () => {
            const versionWithInvalidConfig: VersionInputForParse = {
                codeLanguage: CodeLanguage.Javascript,
                config: "not a valid json string" as any,
            };
            const config = CodeVersionConfig.parse(versionWithInvalidConfig, console);
            const defaultConfig = CodeVersionConfig.default({
                codeLanguage: CodeLanguage.Javascript,
                initialContent: "",
            });
            expect(config.export()).toEqual(defaultConfig.export());
            expect(consoleErrorSpy).toHaveBeenCalledWith("Failed to parse CodeVersionConfig string. Initializing with default content.", expect.any(Object));
        });

        describe("inputs", () => {
            describe("spreads", () => {
                it("number array", () => {
                    const configObject: CodeVersionConfigObject = {
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
                        content: "function sum(a, b, c) { return a + b + c; }",
                    };
                    const configString = JSON.stringify(configObject);
                    const versionInput: VersionInputForParse = {
                        codeLanguage: CodeLanguage.Javascript,
                        config: configString as any,
                    };
                    const parsedConfig = CodeVersionConfig.parse(versionInput, console);
                    expect(parsedConfig.__version).toBe(LATEST_CONFIG_VERSION);
                    expect(parsedConfig.inputConfig).toEqual(configObject.inputConfig);
                    expect(parsedConfig.content).toBe(configObject.content);
                    expect(consoleErrorSpy).not.toHaveBeenCalled();
                });

                it("mixed type array", () => {
                    const configObject: CodeVersionConfigObject = {
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
                        content: "function process(a,b,c) { console.log(a,b,c); }",
                    };
                    const configString = JSON.stringify(configObject);
                    const versionInput: VersionInputForParse = {
                        codeLanguage: CodeLanguage.Javascript,
                        config: configString as any,
                    };
                    const parsedConfig = CodeVersionConfig.parse(versionInput, console);
                    expect(parsedConfig.__version).toBe(LATEST_CONFIG_VERSION);
                    expect(parsedConfig.inputConfig).toEqual(configObject.inputConfig);
                    expect(parsedConfig.content).toBe(configObject.content);
                    expect(consoleErrorSpy).not.toHaveBeenCalled();
                });
            });

            describe("directs", () => {
                it("simple object", () => {
                    const configObject: CodeVersionConfigObject = {
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
                        content: "function greet(obj) { return obj.name; }",
                    };
                    const configString = JSON.stringify(configObject);
                    const versionInput: VersionInputForParse = {
                        codeLanguage: CodeLanguage.Javascript,
                        config: configString as any,
                    };
                    const parsedConfig = CodeVersionConfig.parse(versionInput, console);
                    expect(parsedConfig.__version).toBe(LATEST_CONFIG_VERSION);
                    expect(parsedConfig.inputConfig).toEqual(configObject.inputConfig);
                    expect(parsedConfig.content).toBe(configObject.content);
                    expect(consoleErrorSpy).not.toHaveBeenCalled();
                });

                it("nested object", () => {
                    const configObject: CodeVersionConfigObject = {
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
                        outputConfig: [],
                        testCases: [],
                        content: DEFAULT_TEST_CONTENT,
                    };
                    const configString = JSON.stringify(configObject);
                    const versionInput: VersionInputForParse = {
                        codeLanguage: CodeLanguage.Javascript,
                        config: configString as any,
                    };
                    const parsedConfig = CodeVersionConfig.parse(versionInput, console);
                    expect(parsedConfig.__version).toBe(LATEST_CONFIG_VERSION);
                    expect(parsedConfig.inputConfig).toEqual(configObject.inputConfig);
                    expect(parsedConfig.content).toBe(configObject.content);
                    expect(consoleErrorSpy).not.toHaveBeenCalled();
                });
            });

            it("null config string in version input", () => {
                const versionWithNullConfig: VersionInputForParse = {
                    codeLanguage: CodeLanguage.Javascript,
                    config: null,
                };
                const parsedConfig = CodeVersionConfig.parse(versionWithNullConfig, console);
                const defaultConfig = CodeVersionConfig.default({
                    codeLanguage: CodeLanguage.Javascript,
                    initialContent: "",
                });
                expect(parsedConfig.export()).toEqual(defaultConfig.export());
                expect(consoleErrorSpy).toHaveBeenCalledWith(expect.stringMatching(/Content was not available in parsed config/));
            });
        });

        describe("outputs", () => {
            describe("single schema", () => {
                it("string", () => {
                    const configObject: CodeVersionConfigObject = {
                        __version: LATEST_CONFIG_VERSION,
                        inputConfig: CodeVersionConfig.defaultInputConfig(),
                        outputConfig: { type: "string" } as JsonSchema,
                        testCases: [],
                        content: DEFAULT_TEST_CONTENT,
                    };
                    const configString = JSON.stringify(configObject);
                    const versionInput: VersionInputForParse = {
                        codeLanguage: CodeLanguage.Javascript,
                        config: configString as any,
                    };
                    const parsedConfig = CodeVersionConfig.parse(versionInput, console);
                    expect(parsedConfig.__version).toBe(LATEST_CONFIG_VERSION);
                    expect(parsedConfig.inputConfig).toEqual(configObject.inputConfig);
                    expect(parsedConfig.outputConfig).toEqual(configObject.outputConfig);
                    expect(parsedConfig.content).toBe(configObject.content);
                    expect(consoleErrorSpy).not.toHaveBeenCalled();
                });

                it("object with properties", () => {
                    const configObject: CodeVersionConfigObject = {
                        __version: LATEST_CONFIG_VERSION,
                        inputConfig: CodeVersionConfig.defaultInputConfig(),
                        outputConfig: {
                            type: "object",
                            properties: {
                                result: { type: "number" },
                                message: { type: "string" },
                            },
                            required: ["result"],
                        } as JsonSchema,
                        testCases: [],
                        content: DEFAULT_TEST_CONTENT,
                    };
                    const configString = JSON.stringify(configObject);
                    const versionInput: VersionInputForParse = {
                        codeLanguage: CodeLanguage.Javascript,
                        config: configString as any,
                    };
                    const parsedConfig = CodeVersionConfig.parse(versionInput, console);
                    expect(parsedConfig.__version).toBe(LATEST_CONFIG_VERSION);
                    expect(parsedConfig.inputConfig).toEqual(configObject.inputConfig);
                    expect(parsedConfig.outputConfig).toEqual(configObject.outputConfig);
                    expect(parsedConfig.content).toBe(configObject.content);
                    expect(consoleErrorSpy).not.toHaveBeenCalled();
                });
            });

            describe("multiple schemas", () => {
                it("string and object", () => {
                    const configObject: CodeVersionConfigObject = {
                        __version: LATEST_CONFIG_VERSION,
                        inputConfig: CodeVersionConfig.defaultInputConfig(),
                        outputConfig: [
                            { type: "string" },
                            {
                                type: "object",
                                properties: { error: { type: "string" } },
                                required: ["error"],
                            },
                        ] as JsonSchema[],
                        testCases: [],
                        content: DEFAULT_TEST_CONTENT,
                    };
                    const configString = JSON.stringify(configObject);
                    const versionInput: VersionInputForParse = {
                        codeLanguage: CodeLanguage.Javascript,
                        config: configString as any,
                    };
                    const parsedConfig = CodeVersionConfig.parse(versionInput, console);
                    expect(parsedConfig.__version).toBe(LATEST_CONFIG_VERSION);
                    expect(parsedConfig.inputConfig).toEqual(configObject.inputConfig);
                    expect(parsedConfig.outputConfig).toEqual(configObject.outputConfig);
                    expect(parsedConfig.content).toBe(configObject.content);
                    expect(consoleErrorSpy).not.toHaveBeenCalled();
                });
            });

            it("no outputConfig provided", () => {
                const configObject: CodeVersionConfigObject = {
                    __version: LATEST_CONFIG_VERSION,
                    inputConfig: CodeVersionConfig.defaultInputConfig(),
                    content: DEFAULT_TEST_CONTENT,
                    testCases: CodeVersionConfig.defaultTestCases(),
                };
                const configString = JSON.stringify(configObject);
                const versionInput: VersionInputForParse = {
                    codeLanguage: CodeLanguage.Javascript,
                    config: configString as any,
                };
                const parsedConfig = CodeVersionConfig.parse(versionInput, console);
                expect(parsedConfig.__version).toBe(LATEST_CONFIG_VERSION);
                expect(parsedConfig.inputConfig).toEqual(configObject.inputConfig);
                expect(parsedConfig.outputConfig).toEqual(CodeVersionConfig.defaultOutputConfig());
                expect(parsedConfig.content).toBe(configObject.content);
                expect(consoleErrorSpy).not.toHaveBeenCalled();
            });
        });
    });

    describe("serialization", () => {
        it("spread input and array output", () => {
            const configObject: CodeVersionConfigObject = {
                __version: LATEST_CONFIG_VERSION,
                inputConfig: { inputSchema: { type: "array", items: { type: "number" } }, shouldSpread: true },
                outputConfig: [{ type: "string" }] as JsonSchema[],
                testCases: [],
                content: "function spreadTest() { return 'ok'; }",
                resources: [],
                contractDetails: undefined,
            };
            const configString = JSON.stringify(configObject);
            const versionInput: VersionInputForParse = {
                codeLanguage: CodeLanguage.Javascript,
                config: configString as any,
            };
            const parsedConfig = CodeVersionConfig.parse(versionInput, console);
            expect(parsedConfig.export()).toEqual(configObject);
            expect(consoleErrorSpy).not.toHaveBeenCalled();
        });

        it("direct input and single schema output", () => {
            const configObject: CodeVersionConfigObject = {
                __version: LATEST_CONFIG_VERSION,
                inputConfig: { inputSchema: { type: "object" }, shouldSpread: false },
                outputConfig: { type: "string" } as JsonSchema,
                testCases: [],
                content: "function directTest() { return 'ok'; }",
                resources: [],
                contractDetails: undefined,
            };
            const configString = JSON.stringify(configObject);
            const versionInput: VersionInputForParse = {
                codeLanguage: CodeLanguage.Javascript,
                config: configString as any,
            };
            const parsedConfig = CodeVersionConfig.parse(versionInput, console);
            expect(parsedConfig.export()).toEqual(configObject);
            expect(consoleErrorSpy).not.toHaveBeenCalled();
        });
    });

    describe("runTestCases", () => {
        it("runs a passing test case correctly with spread input", async () => {
            const testContent = "function sum(a, b, c) { return a + b + c; }";
            const configObject: CodeVersionConfigObject = {
                __version: LATEST_CONFIG_VERSION,
                inputConfig: {
                    inputSchema: { type: "array", items: { type: "number" } },
                    shouldSpread: true,
                },
                outputConfig: [{ type: "number" }] as JsonSchema[],
                testCases: [
                    {
                        description: "Simple addition",
                        input: [1, 2, 3],
                        expectedOutput: 6,
                    },
                ],
                content: testContent,
            };
            const configString = JSON.stringify(configObject);
            const versionInput: VersionInputForParse = {
                codeLanguage: CodeLanguage.Javascript,
                config: configString as any,
            };
            const parsedConfig = CodeVersionConfig.parse(versionInput, console);

            async function runSandbox() {
                return { output: 6 };
            }
            const results = await parsedConfig.runTestCases(runSandbox);
            expect(results).toEqual([
                {
                    description: "Simple addition",
                    passed: true,
                },
            ]);
            expect(consoleErrorSpy).not.toHaveBeenCalled();
        });

        it("runs a failing test case correctly with spread input", async () => {
            const testContent = "function sum(a, b, c) { return a + b + c; }";
            const configObject: CodeVersionConfigObject = {
                __version: LATEST_CONFIG_VERSION,
                inputConfig: {
                    inputSchema: { type: "array", items: { type: "number" } },
                    shouldSpread: true,
                },
                outputConfig: [{ type: "number" }] as JsonSchema[],
                testCases: [
                    {
                        description: "Simple addition",
                        input: [1, 2, 3],
                        expectedOutput: 7,
                    },
                ],
                content: testContent,
            };
            const configString = JSON.stringify(configObject);
            const versionInput: VersionInputForParse = {
                codeLanguage: CodeLanguage.Javascript,
                config: configString as any,
            };
            const parsedConfig = CodeVersionConfig.parse(versionInput, console);

            async function runSandbox() {
                return { output: 6 };
            }
            const results = await parsedConfig.runTestCases(runSandbox);
            expect(results).toEqual([
                {
                    description: "Simple addition",
                    passed: false,
                    actualOutput: 6,
                },
            ]);
            expect(consoleErrorSpy).not.toHaveBeenCalled();
        });

        it("handles errors in runSandbox correctly", async () => {
            const testContent = "function sum(a, b, c) { throw new Error('Test error'); }";
            const configObject: CodeVersionConfigObject = {
                __version: LATEST_CONFIG_VERSION,
                inputConfig: {
                    inputSchema: { type: "array", items: { type: "number" } },
                    shouldSpread: true,
                },
                outputConfig: [{ type: "number" }] as JsonSchema[],
                testCases: [
                    {
                        description: "Error case",
                        input: [1, 2, 3],
                        expectedOutput: 6,
                    },
                ],
                content: testContent,
            };
            const configString = JSON.stringify(configObject);
            const versionInput: VersionInputForParse = {
                codeLanguage: CodeLanguage.Javascript,
                config: configString as any,
            };
            const parsedConfig = CodeVersionConfig.parse(versionInput, console);

            async function runSandbox() {
                return { error: "Test error" };
            }
            const results = await parsedConfig.runTestCases(runSandbox);
            expect(results).toEqual([
                {
                    description: "Error case",
                    passed: false,
                    error: "Test error",
                },
            ]);
            expect(consoleErrorSpy).not.toHaveBeenCalled();
        });

        it("runs multiple test cases correctly", async () => {
            const testContent = "function sum(a, b, c) { return a + b + c; }";
            const configObject: CodeVersionConfigObject = {
                __version: LATEST_CONFIG_VERSION,
                inputConfig: {
                    inputSchema: { type: "array", items: { type: "number" } },
                    shouldSpread: true,
                },
                outputConfig: [{ type: "number" }] as JsonSchema[],
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
                content: testContent,
            };
            const configString = JSON.stringify(configObject);
            const versionInput: VersionInputForParse = {
                codeLanguage: CodeLanguage.Javascript,
                config: configString as any,
            };
            const parsedConfig = CodeVersionConfig.parse(versionInput, console);

            async function runSandbox(params: { input?: unknown }) {
                const input = params.input as number[];
                const sum = input.reduce((a, b) => a + b, 0);
                return { output: sum };
            }
            const results = await parsedConfig.runTestCases(runSandbox);
            expect(results).toEqual([
                {
                    description: "First test",
                    passed: true,
                },
                {
                    description: "Second test",
                    passed: true,
                },
            ]);
            expect(consoleErrorSpy).not.toHaveBeenCalled();
        });

        it("runs test cases with direct input correctly", async () => {
            const testContent = "function add(obj) { return obj.a + obj.b; }";
            const configObject: CodeVersionConfigObject = {
                __version: LATEST_CONFIG_VERSION,
                inputConfig: {
                    inputSchema: {
                        type: "object",
                        properties: { a: { type: "number" }, b: { type: "number" } },
                    },
                    shouldSpread: false,
                },
                outputConfig: [{ type: "number" }] as JsonSchema[],
                testCases: [
                    {
                        description: "Object input",
                        input: { a: 1, b: 2 },
                        expectedOutput: 3,
                    },
                ],
                content: testContent,
            };
            const configString = JSON.stringify(configObject);
            const versionInput: VersionInputForParse = {
                codeLanguage: CodeLanguage.Javascript,
                config: configString as any,
            };
            const parsedConfig = CodeVersionConfig.parse(versionInput, console);

            async function runSandbox(params: { input?: unknown }) {
                const input = params.input as { a: number; b: number };
                return { output: input.a + input.b };
            }
            const results = await parsedConfig.runTestCases(runSandbox);
            expect(results).toEqual([
                {
                    description: "Object input",
                    passed: true,
                },
            ]);
            expect(consoleErrorSpy).not.toHaveBeenCalled();
        });

        it("runs test cases with complex object output correctly", async () => {
            const testContent = "function greet(obj) { return { greeting: `Hello, ${obj.name}` }; }";
            const configObject: CodeVersionConfigObject = {
                __version: LATEST_CONFIG_VERSION,
                inputConfig: {
                    inputSchema: { type: "object", properties: { name: { type: "string" } } },
                    shouldSpread: false,
                },
                outputConfig: [{ type: "object", properties: { greeting: { type: "string" } } }] as JsonSchema[],
                testCases: [
                    {
                        description: "Greeting test",
                        input: { name: "Alice" },
                        expectedOutput: { greeting: "Hello, Alice" },
                    },
                ],
                content: testContent,
            };
            const configString = JSON.stringify(configObject);
            const versionInput: VersionInputForParse = {
                codeLanguage: CodeLanguage.Javascript,
                config: configString as any,
            };
            const parsedConfig = CodeVersionConfig.parse(versionInput, console);

            async function runSandbox(params: { input?: unknown }) {
                const input = params.input as { name: string };
                return { output: { greeting: `Hello, ${input.name}` } };
            }
            const results = await parsedConfig.runTestCases(runSandbox);
            expect(results).toEqual([
                {
                    description: "Greeting test",
                    passed: true,
                },
            ]);
            expect(consoleErrorSpy).not.toHaveBeenCalled();
        });

        it("detects failing test case with object output", async () => {
            const testContent = "function greet(obj) { return { greeting: `Hello, ${obj.name}` }; }";
            const configObject: CodeVersionConfigObject = {
                __version: LATEST_CONFIG_VERSION,
                inputConfig: {
                    inputSchema: { type: "object", properties: { name: { type: "string" } } },
                    shouldSpread: false,
                },
                outputConfig: [{ type: "object", properties: { greeting: { type: "string" } } }] as JsonSchema[],
                testCases: [
                    {
                        description: "Greeting test",
                        input: { name: "Alice" },
                        expectedOutput: { greeting: "Hi, Alice" },
                    },
                ],
                content: testContent,
            };
            const configString = JSON.stringify(configObject);
            const versionInput: VersionInputForParse = {
                codeLanguage: CodeLanguage.Javascript,
                config: configString as any,
            };
            const parsedConfig = CodeVersionConfig.parse(versionInput, console);

            async function runSandbox(params: { input?: unknown }) {
                const input = params.input as { name: string };
                return { output: { greeting: `Hello, ${input.name}` } };
            }
            const results = await parsedConfig.runTestCases(runSandbox);
            expect(results).toEqual([
                {
                    description: "Greeting test",
                    passed: false,
                    actualOutput: { greeting: "Hello, Alice" },
                },
            ]);
            expect(consoleErrorSpy).not.toHaveBeenCalled();
        });
    });
});
