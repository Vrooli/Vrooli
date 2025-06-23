// AI_CHECK: TEST_COVERAGE=3 | LAST: 2025-06-18
/* eslint-disable @typescript-eslint/ban-ts-comment */
import { describe, it, expect, vi, beforeAll, beforeEach, afterAll } from "vitest";
import { ResourceSubType } from "../../api/types.js";
import { McpToolName } from "../../consts/mcp.js";
import { BotStyle } from "../../run/enums.js";
import { validatePublicId } from "../../id/publicId.js";
import { type SubroutineIOMapping } from "../../run/types.js";
import { type CodeVersionInputDefinition } from "./code.js";
import { CallDataActionConfig, CallDataCodeConfig, CallDataGenerateConfig, CallDataWebConfig, GraphBpmnConfig, RoutineVersionConfig } from "./routine.js";

const LATEST_VERSION = "1.0.0";

describe("CallDataActionConfig", () => {
    let consoleErrorSpy: any;
    let config: CallDataActionConfig;
    let ioMapping: SubroutineIOMapping;
    let userLanguages: string[];

    beforeAll(async () => {
        consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    });

    beforeEach(() => {
        consoleErrorSpy.mockClear();
        config = new CallDataActionConfig({
            __version: LATEST_VERSION,
            schema: {
                toolName: McpToolName.SendMessage,
                inputTemplate: "",
                outputMapping: {},
            },
        });
        ioMapping = {
            inputs: {
                // Strings
                routineInputA: { value: "valueA" },
                routineInputB: { value: "valueB" },
                // Numbers
                routineInputC: { value: 42 },
                // Booleans
                routineInputD: { value: true },
                routineInputE: { value: false },
                // Arrays
                routineInputF: { value: ["item1", "item2"] },
                routineInputG: { value: [1, 2, 3] },
                // Objects
                routineInputH: { value: { key: "value" } },
                routineInputI: { value: { nested: { key: "value" } } },
                // Dates
                routineInputJ: { value: new Date("2024-01-01") },
                routineInputK: { value: new Date("2024-01-01T12:00:00Z") },
                // Null
                routineInputL: { value: null },
            },
            outputs: {},
        } as unknown as SubroutineIOMapping;
        userLanguages = ["en", "es"];
    });

    afterAll(() => {
        consoleErrorSpy.mockRestore();
    });

    describe("replacePlaceholders", () => {
        describe("Routine Inputs", () => {
            describe("valid test cases", () => {
                it("should replace string input placeholders", () => {
                    const str = "{{input.routineInputA}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).toBe(ioMapping.inputs.routineInputA.value);
                });

                it("should replace numeric input placeholders", () => {
                    const str = "{{input.routineInputC}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).toBe(ioMapping.inputs.routineInputC.value);
                });

                it("should replace boolean input placeholders", () => {
                    const str = "{{input.routineInputD}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).toBe(ioMapping.inputs.routineInputD.value);
                });

                it("should replace array input placeholders", () => {
                    const str = "{{input.routineInputF}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).toEqual(ioMapping.inputs.routineInputF.value);
                });

                it("should replace object input placeholders", () => {
                    const str = "{{input.routineInputH}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).toEqual(ioMapping.inputs.routineInputH.value);
                });

                it("should replace date input placeholders", () => {
                    const str = "{{input.routineInputJ}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).toBe(ioMapping.inputs.routineInputJ.value);
                });

                it("should replace null input placeholders", () => {
                    const str = "{{input.routineInputL}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).toBe(ioMapping.inputs.routineInputL.value);
                });

                it("should handle multiple placeholders in a single string", () => {
                    const str = "A: {{input.routineInputA}}, B: {{input.routineInputC}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).toBe(`A: ${ioMapping.inputs.routineInputA.value}, B: ${ioMapping.inputs.routineInputC.value}`);
                });

                it("should leave static strings unchanged", () => {
                    const str = "static string";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).toBe(str);
                });
            });

            describe("invalid test cases", () => {
                it("should throw an error for missing input", () => {
                    const str = "{{input.missingInput}}";
                    expect(() => config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} })).toThrow(
                        "Input \"missingInput\" not found",
                    );
                });
            });
        });

        describe("Special Functions", () => {
            describe("valid test cases", () => {
                it("should replace userLanguage placeholder with the first language", () => {
                    const str = "{{userLanguage}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).toBe("en");
                });

                it("should replace userLanguages placeholder with the full language array", () => {
                    const str = "{{userLanguages}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).toEqual(["en", "es"]);
                });

                it("should replace now() placeholder with a valid ISO date string", () => {
                    const str = "{{now()}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    const asDate = new Date(result as string);
                    expect(asDate.toISOString()).toBe(result);
                });

                it("should replace nanoid() placeholder with a valid id", () => {
                    const str = "{{nanoid()}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(validatePublicId(result)).toBe(true);
                });

                it("should replace nanoid(seed) placeholder with consistent id for the same seed", () => {
                    const str = "{{nanoid(123)}}|{{nanoid(123)}}|{{nanoid(456)}}|{{nanoid(456)}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    const ids = (result as string).split("|");
                    expect(ids[0]).toBe(ids[1]);
                    expect(ids[2]).toBe(ids[3]);
                });

                it("should replace random() placeholder with a random number", () => {
                    const str = "{{random()}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    // Should be a number from 0 to 1
                    expect(result).to.be.a("number");
                    expect(result).to.be.within(0, 1);
                });

                it("should replace random(min, max) placeholder with a number within the specified range", () => {
                    const str = "{{random(1, 10)}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).to.be.a("number");
                    expect(result).to.be.within(1, 10);
                });
            });

            describe("invalid test cases", () => {
                it("should throw an error for invalid arguments in random(min, max)", () => {
                    const str = "{{random(abc, 10)}}";
                    expect(() => config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} })).toThrow();
                });
            });
        });

        describe("Error Handling", () => {
            it("should throw an error for unknown placeholders", () => {
                const str = "{{unknown}}";
                expect(() => config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} })).toThrow();
            });

            it("should not transform malformed placeholders", () => {
                const str = "{{input.routineInputA";
                const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                expect(result).toBe(str);
            });
        });
    });

    describe("buildTaskInput", () => {
        describe("Basic Placeholder Replacement", () => {
            it("should replace simple string input placeholders", () => {
                config.schema.inputTemplate = "{\"field\": \"{{input.routineInputA}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).toEqual({ field: ioMapping.inputs.routineInputA.value });
            });

            it("should replace numeric input placeholders", () => {
                config.schema.inputTemplate = "{\"number\": \"{{input.routineInputC}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).toEqual({ number: ioMapping.inputs.routineInputC.value });
            });

            it("should replace boolean input placeholders", () => {
                config.schema.inputTemplate = "{\"bool\": \"{{input.routineInputD}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).toEqual({ bool: ioMapping.inputs.routineInputD.value });
            });

            it("should replace array input placeholders", () => {
                config.schema.inputTemplate = "{\"array\": \"{{input.routineInputF}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).toEqual({ array: ioMapping.inputs.routineInputF.value });
            });

            it("should replace object input placeholders", () => {
                config.schema.inputTemplate = "{\"object\": \"{{input.routineInputH}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).toEqual({ object: ioMapping.inputs.routineInputH.value });
            });
        });

        describe("Special Function Placeholders", () => {
            it("should replace {{nanoid()}} with a valid id", () => {
                config.schema.inputTemplate = "{\"id\": \"{{nanoid()}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).toHaveProperty("id");
                expect(validatePublicId(result.id as string)).toBe(true);
            });

            it("should generate consistent ids for the same seed across the template", () => {
                config.schema.inputTemplate = "{\"id1\": \"{{nanoid(123)}}\", \"child\": {\"id2\": \"{{nanoid(123)}}\", \"id3\": \"{{nanoid(456)}}\"}}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                // @ts-ignore Testing runtime scenario
                expect(result.id1).toBe(result.child.id2);
                // @ts-ignore Testing runtime scenario
                expect(result.id1).to.not.equal(result.child.id3);
                expect(validatePublicId(result.id1 as string)).toBe(true);
                // @ts-ignore Testing runtime scenario
                expect(validatePublicId(result.child.id3 as string)).toBe(true);
            });

            it("should replace {{userLanguage}} with the first language", () => {
                config.schema.inputTemplate = "{\"lang\": \"{{userLanguage}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).toEqual({ lang: userLanguages[0] });
            });

            it("should replace {{userLanguages}} with the full language array", () => {
                config.schema.inputTemplate = "{\"langs\": \"{{userLanguages}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).toEqual({ langs: userLanguages });
            });

            it("should replace {{now()}} with a valid ISO date string", () => {
                config.schema.inputTemplate = "{\"date\": \"{{now()}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(new Date(result.date as string).toISOString()).toBe(result.date);
            });

            it("should replace {{random()}} with a number between 0 and 1", () => {
                config.schema.inputTemplate = "{\"num\": \"{{random()}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result.num).to.be.a("number").within(0, 1);
            });

            it("should replace {{random(min, max)}} with an integer in the specified range", () => {
                config.schema.inputTemplate = "{\"num\": \"{{random(1, 10)}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result.num).to.be.a("number").within(1, 10);
                expect(Number.isInteger(result.num)).toBe(true);
            });
        });

        describe("Nested Structures", () => {
            it("should replace placeholders in nested objects", () => {
                config.schema.inputTemplate = "{\"nested\": {\"field\": \"{{input.routineInputB}}\"}}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).toEqual({ nested: { field: ioMapping.inputs.routineInputB.value } });
            });

            it("should replace placeholders in arrays", () => {
                config.schema.inputTemplate = "{\"items\": [\"{{input.routineInputA}}\", \"{{input.routineInputB}}\"]}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).toEqual({ items: [ioMapping.inputs.routineInputA.value, ioMapping.inputs.routineInputB.value] });
            });

            it("should replace placeholders in arrays of objects", () => {
                config.schema.inputTemplate = "{\"users\": [{\"name\": \"{{input.routineInputA}}\"}, {\"name\": \"{{input.routineInputB}}\"}]}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).toEqual({ users: [{ name: ioMapping.inputs.routineInputA.value }, { name: ioMapping.inputs.routineInputB.value }] });
            });
        });

        describe("Multiple Placeholders", () => {
            it("should replace multiple placeholders in a string", () => {
                config.schema.inputTemplate = "{\"message\": \"Hello, {{input.routineInputA}}! Score: {{input.routineInputC}}.\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).toEqual({ message: `Hello, ${ioMapping.inputs.routineInputA.value}! Score: ${ioMapping.inputs.routineInputC.value}.` });
            });

            it("should replace multiple placeholders across object fields", () => {
                config.schema.inputTemplate = "{\"name\": \"{{input.routineInputA}}\", \"score\": \"{{input.routineInputC}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).toEqual({ name: ioMapping.inputs.routineInputA.value, score: ioMapping.inputs.routineInputC.value });
            });
        });

        describe("Entire String as Placeholder", () => {
            it("should return raw string value when template is a single placeholder", () => {
                config.schema.inputTemplate = "\"{{input.routineInputA}}\"";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).toBe(ioMapping.inputs.routineInputA.value);
            });

            it("should return raw number value when template is a single placeholder", () => {
                config.schema.inputTemplate = "\"{{input.routineInputC}}\"";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).toBe(ioMapping.inputs.routineInputC.value);
            });

            it("should return raw array value when template is a single placeholder", () => {
                config.schema.inputTemplate = "\"{{input.routineInputF}}\"";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).toEqual(ioMapping.inputs.routineInputF.value);
            });
        });

        describe("Primitive Templates", () => {
            it("should return a primitive string without placeholders", () => {
                config.schema.inputTemplate = "\"hello\"";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).toBe("hello");
            });

            it("should return a primitive number", () => {
                config.schema.inputTemplate = "42";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).toBe(42);
            });

            it("should return a primitive boolean", () => {
                config.schema.inputTemplate = "true";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).toBe(true);
            });
        });

        describe("Complex Cases", () => {
            it("should handle complex nested structures with various placeholders", () => {
                config.schema.inputTemplate = `{
                    "id": "{{nanoid()}}",
                    "user": {
                        "name": "{{input.routineInputA}}",
                        "languages": "{{userLanguages}}",
                        "registrationDate": "{{now()}}"
                    },
                    "scores": [
                        {"level": 1, "score": "{{random(0, 100)}}"},
                        {"level": 2, "score": "{{random(0, 100)}}"}
                    ]
                }`;
                const result = config.buildTaskInput(ioMapping, userLanguages);

                // Check id is a valid string
                expect(result).toHaveProperty("id");
                expect(typeof result.id).toBe("string");
                expect(validatePublicId(result.id)).toBe(true);

                // Check user object
                expect(result).toHaveProperty("user");
                expect(typeof result.user).toBe("object");
                expect(result.user).toHaveProperty("name", ioMapping.inputs.routineInputA.value);
                expect(result.user).toHaveProperty("languages");
                expect(result.user.languages).toEqual(userLanguages);
                expect(result.user).toHaveProperty("registrationDate");
                expect(typeof result.user.registrationDate).toBe("string");

                // Validate registrationDate is a valid ISO date
                // @ts-ignore Testing runtime scenario
                const registrationDate = result.user.registrationDate;
                expect(new Date(registrationDate).toISOString()).toBe(registrationDate);

                // Check scores array
                expect(result).toHaveProperty("scores");
                expect(Array.isArray(result.scores)).toBe(true);
                expect(result.scores).toHaveLength(2);

                // Check scores[0]
                // @ts-ignore Testing runtime scenario
                expect(result.scores[0]).toHaveProperty("level", 1);
                // @ts-ignore Testing runtime scenario
                expect(result.scores[0]).toHaveProperty("score");
                expect(typeof result.scores[0].score).toBe("number");
                expect(Number.isInteger(result.scores[0].score)).toBe(true);
                expect(result.scores[0].score).toBeGreaterThanOrEqual(0);
                expect(result.scores[0].score).toBeLessThanOrEqual(100);

                // Check scores[1]
                // @ts-ignore Testing runtime scenario
                expect(result.scores[1]).toHaveProperty("level", 2);
                // @ts-ignore Testing runtime scenario
                expect(result.scores[1]).toHaveProperty("score");
                expect(typeof result.scores[1].score).toBe("number");
                expect(Number.isInteger(result.scores[1].score)).toBe(true);
                expect(result.scores[1].score).toBeGreaterThanOrEqual(0);
                expect(result.scores[1].score).toBeLessThanOrEqual(100);
            });
        });

        describe("Error Handling", () => {
            it("should throw an error for missing input", () => {
                config.schema.inputTemplate = "{\"field\": \"{{input.missingInput}}\"}";
                ioMapping.inputs = {};
                expect(() => config.buildTaskInput(ioMapping, userLanguages)).toThrow("Input \"missingInput\" not found");
            });

            it("should throw an error for invalid JSON in inputTemplate", () => {
                config.schema.inputTemplate = "{ invalid json }";
                expect(() => config.buildTaskInput(ioMapping, userLanguages)).toThrow("Failed to parse inputTemplate");
            });

            it("should preserve malformed placeholders without throwing", () => {
                config.schema.inputTemplate = "{\"field\": \"{{input.routineInputA\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).toEqual({ field: "{{input.routineInputA" });
            });

            it("should throw an error for invalid arguments in random()", () => {
                config.schema.inputTemplate = "{\"num\": \"{{random(abc, 10)}}\"}";
                expect(() => config.buildTaskInput(ioMapping, userLanguages)).toThrow("Invalid arguments for random()");
            });
        });
    });

    describe("parseActionResult", () => {
        it("should map a simple property from result to output", () => {
            config.schema.outputMapping = { outputPayload: "." };
            ioMapping.outputs = { outputPayload: { value: null } } as unknown as SubroutineIOMapping["outputs"];
            const result = { payload: "some value" };
            config.parseActionResult(ioMapping, result);
            expect(ioMapping.outputs.outputPayload.value).toBe(result.payload);
        });

        it("should map a nested property from result to output", () => {
            config.schema.outputMapping = { outputId: "data.id" };
            ioMapping.outputs = { outputId: { value: null } } as unknown as SubroutineIOMapping["outputs"];
            const result = { payload: { data: { id: 123 } } };
            config.parseActionResult(ioMapping, result);
            expect(ioMapping.outputs.outputId.value).toBe(result.payload.data.id);
        });

        it("should not update output if the path is missing in result", () => {
            config.schema.outputMapping = { "outputMissing": "missing" };
            ioMapping.outputs = { outputMissing: { value: null } } as unknown as SubroutineIOMapping["outputs"];
            const result = { payload: { data: { id: 123 } } };
            config.parseActionResult(ioMapping, result);
            expect(ioMapping.outputs.outputMissing.value).toBeNull();
        });

        it("should not throw if the output name does not exist in ioMapping.outputs", () => {
            config.schema.outputMapping = { "nonexistentOutput": "data.id" };
            ioMapping.outputs = {} as unknown as SubroutineIOMapping["outputs"];
            const result = { payload: { data: { id: 123 } } };
            expect(() => config.parseActionResult(ioMapping, result)).to.not.throw();
        });

        it("should handle multiple mappings correctly", () => {
            config.schema.outputMapping = {
                "outputId": "data.id",
                "outputName": "data.name",
            };
            ioMapping.outputs = {
                outputId: { value: null },
                outputName: { value: null },
            } as unknown as SubroutineIOMapping["outputs"];
            const result = { payload: { data: { id: 123, name: "test" } } };
            config.parseActionResult(ioMapping, result);
            expect(ioMapping.outputs.outputId.value).toBe(result.payload.data.id);
            expect(ioMapping.outputs.outputName.value).toBe(result.payload.data.name);
        });

        it("should handle different data types correctly", () => {
            config.schema.outputMapping = {
                "outputBool": "bool",
                "outputNum": "num",
                "outputObj": "obj",
            };
            ioMapping.outputs = {
                outputBool: { value: null },
                outputNum: { value: null },
                outputObj: { value: null },
            } as unknown as SubroutineIOMapping["outputs"];
            const result = { payload: { bool: true, num: 42, obj: { key: "value" } } };
            config.parseActionResult(ioMapping, result);
            expect(ioMapping.outputs.outputBool.value).toBe(result.payload.bool);
            expect(ioMapping.outputs.outputNum.value).toBe(result.payload.num);
            expect(ioMapping.outputs.outputObj.value).toEqual(result.payload.obj);
        });

        it("should map an array element correctly", () => {
            config.schema.outputMapping = { "firstItem": "1" };
            ioMapping.outputs = { firstItem: { value: null } } as unknown as SubroutineIOMapping["outputs"];
            const result = { payload: ["a", "b", "c"] };
            config.parseActionResult(ioMapping, result);
            expect(ioMapping.outputs.firstItem.value).toBe(result.payload[1]);
        });

        it("should do nothing if result has no payload", () => {
            config.schema.outputMapping = { "outputData": "data" };
            ioMapping.outputs = { outputData: { value: null } } as unknown as SubroutineIOMapping["outputs"];
            const result = { other: "data" };
            // @ts-ignore Testing runtime scenario
            config.parseActionResult(ioMapping, result);
            expect(ioMapping.outputs.outputData.value).toBeNull();
        });

        it("should handle null payload correctly", () => {
            config.schema.outputMapping = { "outputSomething": "." };
            ioMapping.outputs = { outputSomething: { value: "not-null" } } as unknown as SubroutineIOMapping["outputs"];
            const result = { payload: null };
            config.parseActionResult(ioMapping, result);
            expect(ioMapping.outputs.outputSomething.value).toBeNull(); // Overwrites the previous value with the requested value (null)
        });

        it("should handle deeply nested properties", () => {
            config.schema.outputMapping = { "outputDeep": "a.b.c.d" };
            ioMapping.outputs = { outputDeep: { value: null } } as unknown as SubroutineIOMapping["outputs"];
            const result = { payload: { a: { b: { c: { d: "deep value" } } } } };
            config.parseActionResult(ioMapping, result);
            expect(ioMapping.outputs.outputDeep.value).toBe(result.payload.a.b.c.d);
        });

        it("should not update output if nested property does not exist", () => {
            config.schema.outputMapping = { "outputMissing": "data.missing" };
            ioMapping.outputs = { outputMissing: { value: null } } as unknown as SubroutineIOMapping["outputs"];
            const result = { payload: { data: { id: 123 } } };
            config.parseActionResult(ioMapping, result);
            expect(ioMapping.outputs.outputMissing.value).toBeNull();
        });

        it("should not modify unmapped outputs", () => {
            config.schema.outputMapping = { "outputId": "data.id" };
            ioMapping.outputs = {
                outputId: { value: null },
                otherOutput: { value: "original" },
            } as unknown as SubroutineIOMapping["outputs"];
            const result = { payload: { data: { id: 123 } } };
            config.parseActionResult(ioMapping, result);
            expect(ioMapping.outputs.outputId.value).toBe(result.payload.data.id);
            expect(ioMapping.outputs.otherOutput.value).toBe("original");
        });
    });
});

describe("CallDataCodeConfig", () => {
    let consoleErrorSpy: any;
    let config: CallDataCodeConfig;

    beforeAll(async () => {
        consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    });

    beforeEach(() => {
        consoleErrorSpy.mockClear();
        config = new CallDataCodeConfig({
            __version: LATEST_VERSION,
            schema: { inputTemplate: [], outputMappings: [] }, // Default, overridden in tests
        });
    });

    afterAll(() => {
        consoleErrorSpy.mockRestore();
    });

    describe("buildSandboxInput", () => {
        const ioMapping = {
            inputs: {
                // Strings
                routineInputA: { value: "valueA" },
                routineInputB: { value: "valueB" },
                // Numbers
                routineInputC: { value: 42 },
                // Booleans
                routineInputD: { value: true },
                routineInputE: { value: false },
                // Arrays
                routineInputF: { value: ["item1", "item2"] },
                routineInputG: { value: [1, 2, 3] },
                // Objects
                routineInputH: { value: { key: "value" } },
                routineInputI: { value: { nested: { key: "value" } } },
                // Dates
                routineInputJ: { value: new Date("2024-01-01") },
                routineInputK: { value: new Date("2024-01-01T12:00:00Z") },
                // Null
                routineInputL: { value: null },
            },
        } as unknown as SubroutineIOMapping;

        describe("Spread Input (shouldSpread: true)", () => {
            const inputConfig: CodeVersionInputDefinition = {
                inputSchema: { type: "array", items: { type: "string" } },
                shouldSpread: true,
            };

            describe("valid test cases", () => {
                it("should replace placeholders in an array template", () => {
                    config.schema.inputTemplate = ["{{routineInputA}}", "{{routineInputB}}"];
                    const result = config.buildSandboxInput(ioMapping, inputConfig);
                    expect(result.input).toEqual(["valueA", "valueB"]);
                    expect(result.shouldSpreadInput).toBe(true);
                });

                it("should handle nested placeholders in an array", () => {
                    config.schema.inputTemplate = ["{{routineInputA}}", ["{{routineInputB}}", "{{routineInputC}}"]];
                    const result = config.buildSandboxInput(ioMapping, inputConfig);
                    expect(result.input).toEqual(["valueA", ["valueB", 42]]);
                });

                it("should handle a single placeholder in an array", () => {
                    config.schema.inputTemplate = ["{{routineInputA}}"];
                    const result = config.buildSandboxInput(ioMapping, inputConfig);
                    expect(result.input).toEqual(["valueA"]);
                });

                it("should leave a string untransformed if it doesn't contain placeholders", () => {
                    config.schema.inputTemplate = ["static string"];
                    const result = config.buildSandboxInput(ioMapping, inputConfig);
                    expect(result.input).toEqual(["static string"]);
                });

                it("should be able to handle a mix of placeholders and static strings", () => {
                    config.schema.inputTemplate = ["{{routineInputA}}", "static string", "placeholder: {{routineInputB}}"];
                    const result = config.buildSandboxInput(ioMapping, inputConfig);
                    expect(result.input).toEqual(["valueA", "static string", "placeholder: valueB"]);
                });

                it("should handle plaeholders of all types", () => {
                    config.schema.inputTemplate = ["{{routineInputA}}", "{{routineInputB}}", "{{routineInputC}}", "{{routineInputD}}", "{{routineInputE}}", "{{routineInputF}}", "{{routineInputG}}", "{{routineInputH}}", "{{routineInputI}}", "{{routineInputJ}}", "{{routineInputK}}", "{{routineInputL}}"];
                    const result = config.buildSandboxInput(ioMapping, inputConfig);
                    expect(result.input).toEqual(["valueA", "valueB", 42, true, false, ["item1", "item2"], [1, 2, 3], { key: "value" }, { nested: { key: "value" } }, new Date("2024-01-01"), new Date("2024-01-01T12:00:00Z"), null]);
                });

                it("should correctly stringify all types when using string interpolation", () => {
                    config.schema.inputTemplate = ["a: {{routineInputA}}", "b: {{routineInputB}}", "c: {{routineInputC}}", "d: {{routineInputD}}", "e: {{routineInputE}}", "f: {{routineInputF}}", "g: {{routineInputG}}", "h: {{routineInputH}}", "i: {{routineInputI}}", "j: {{routineInputJ}}", "k: {{routineInputK}}", "l: {{routineInputL}}"];
                    const result = config.buildSandboxInput(ioMapping, inputConfig);
                    expect(result.input).toEqual(["a: valueA", "b: valueB", "c: 42", "d: true", "e: false", "f: [\"item1\",\"item2\"]", "g: [1,2,3]", "h: {\"key\":\"value\"}", "i: {\"nested\":{\"key\":\"value\"}}", "j: 2024-01-01T00:00:00.000Z", "k: 2024-01-01T12:00:00.000Z", "l: null"]);
                });
            });

            describe("invalid test cases", () => {
                it("should throw an error if a placeholder input is missing", () => {
                    config.schema.inputTemplate = ["{{missingInput}}"];
                    expect(() => config.buildSandboxInput(ioMapping, inputConfig)).toThrow(
                        "Input \"missingInput\" not found in ioMapping or missing value",
                    );
                });

                it("should throw an error if the template is not an array", () => {
                    config.schema.inputTemplate = { key: "{{routineInputA}}" };
                    expect(() => config.buildSandboxInput(ioMapping, inputConfig)).toThrow();
                });
            });
        });

        describe("Direct Input (shouldSpread: false)", () => {
            const inputConfig: CodeVersionInputDefinition = {
                inputSchema: { type: "object" },
                shouldSpread: false,
            };

            describe("valid test cases", () => {
                it("should replace placeholders in an object template", () => {
                    config.schema.inputTemplate = { key: "{{routineInputA}}", nested: { value: "placeholder 1: {{routineInputB}}, placeholder 2: {{routineInputC}}" } };
                    const result = config.buildSandboxInput(ioMapping, inputConfig);
                    expect(result.input).toEqual({ key: "valueA", nested: { value: "placeholder 1: valueB, placeholder 2: 42" } });
                    expect(result.shouldSpreadInput).toBe(false);
                });
            });

            describe("invalid test cases", () => {
                it("should throw an error if a placeholder input is missing", () => {
                    config.schema.inputTemplate = { key: "{{missingInput}}" };
                    expect(() => config.buildSandboxInput(ioMapping, inputConfig)).toThrow(
                        "Input \"missingInput\" not found in ioMapping or missing value",
                    );
                });
            });
        });
    });

    describe("parseSandboxOutput", () => {
        const ioMapping = {
            inputs: {},
            outputs: {
                routineOutputX: { value: undefined },
                routineOutputY: { value: undefined },
                routineOutputZ: { value: undefined },
                routineOutputError: { value: undefined },
            },
        } as unknown as SubroutineIOMapping;

        // Reset ioMapping outputs before each test to ensure a clean state
        beforeEach(() => {
            Object.values(ioMapping.outputs).forEach((output) => {
                output.value = undefined;
            });
        });

        describe("valid test cases", () => {
            describe("single output", () => {
                it("string", () => {
                    config.schema.outputMappings = [
                        // We use "." to indicate the entire output object
                        { schemaIndex: 0, mapping: { routineOutputZ: "." } },
                    ];
                    const outputConfig = [{ type: "string" }] as const;
                    const runOutput = { output: "hello" };
                    config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputZ.value).toBe("hello");
                });

                it("number", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputZ: "." } },
                    ];
                    const outputConfig = [{ type: "number" }] as const;
                    const runOutput = { output: 42 };
                    config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputZ.value).toBe(42);
                });

                it("boolean", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputZ: "." } },
                    ];
                    const outputConfig = [{ type: "boolean" }] as const;
                    const runOutput = { output: true };
                    config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputZ.value).toBe(true);
                });

                it("full array", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputZ: "." } },
                    ];
                    const outputConfig = [{ type: "array", items: { type: "string" } }] as const;
                    const runOutput = { output: ["item1", "item2"] };
                    config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputZ.value).toEqual(["item1", "item2"]);
                });

                it("specific array index", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputZ: "[1]" } },
                    ];
                    const outputConfig = [{
                        type: "array",
                        items: {
                            type: "string",
                        },
                        minItems: 1,
                        maxItems: 2,
                    }] as const;
                    const runOutput = { output: ["item1", "item2"] };
                    config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputZ.value).toBe("item2");
                });

                it("object", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputZ: "." } },
                    ];
                    const outputConfig = [{ type: "object", properties: { key: { type: "string" } } }] as const;
                    const runOutput = { output: { key: "value" } };
                    config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputZ.value).toEqual({ key: "value" });
                });

                it("null", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputZ: "." } },
                    ];
                    const outputConfig = [{ type: "null" }] as const;
                    const runOutput = { output: null };
                    config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputZ.value).toBeNull();
                });
            });

            describe("multiple outputs", () => {
                it("string and number", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputX: "result", routineOutputY: "message" } },
                    ];
                    const outputConfig = [
                        {
                            type: "object",
                            properties: {
                                result: { type: "string" },
                                message: { type: "number" },
                            },
                        },
                    ] as const;
                    const runOutput = { output: { result: "hello", message: 42 } };
                    config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputX.value).toBe("hello");
                    expect(ioMapping.outputs.routineOutputY.value).toBe(42);
                });

                it("null and object", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputX: "result", routineOutputY: "message" } },
                    ];
                    const outputConfig = [
                        {
                            type: "object",
                            properties: {
                                result: { type: "null" },
                                message: { type: "object", properties: { key: { type: "string" } } },
                            },
                        },
                    ] as const;
                    const runOutput = { output: { result: null, message: { key: "value" } } };
                    config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputX.value).toBeNull();
                    expect(ioMapping.outputs.routineOutputY.value).toEqual({ key: "value" });
                });

                it("accessing both full output and individual outputs", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputX: ".", routineOutputY: "message" } },
                    ];
                    const outputConfig = [
                        {
                            type: "object",
                            properties: { result: { type: "string" }, message: { type: "string" } },
                        },
                    ] as const;
                    const runOutput = { output: { result: "hello", message: "world" } };
                    config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputX.value).toEqual({ result: "hello", message: "world" });
                    expect(ioMapping.outputs.routineOutputY.value).toBe("world");
                });
            });

            describe("multiple output mappings", () => {
                it("string and null", () => {
                    config.schema.outputMappings = [
                        // When output matches schemaIndex 0 (string), map to routineOutputX
                        { schemaIndex: 0, mapping: { routineOutputX: "." } },
                        // When output matches schemaIndex 1 (boolean), map to routineOutputY
                        { schemaIndex: 1, mapping: { routineOutputY: "." } },
                    ];
                    const outputConfig = [
                        { type: "string" },
                        { type: "boolean" },
                    ] as const;
                    const runOutputA = { output: "hello" };
                    const runOutputB = { output: true };
                    config.parseSandboxOutput(runOutputA, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputX.value).toBe("hello");
                    expect(ioMapping.outputs.routineOutputY.value).toBeNull();
                    config.parseSandboxOutput(runOutputB, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputX.value).toBeNull();
                    expect(ioMapping.outputs.routineOutputY.value).toBe(true);
                });

                it("string and null, but outputMappings out of order", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 1, mapping: { routineOutputY: "." } },
                        { schemaIndex: 0, mapping: { routineOutputX: "." } },
                    ];
                    const outputConfig = [
                        { type: "string" },
                        { type: "boolean" },
                    ] as const;
                    const runOutputA = { output: "hello" };
                    const runOutputB = { output: true };
                    // Should give the same result as the previous test
                    config.parseSandboxOutput(runOutputA, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputX.value).toBe("hello");
                    expect(ioMapping.outputs.routineOutputY.value).toBeNull();
                    config.parseSandboxOutput(runOutputB, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputX.value).toBeNull();
                    expect(ioMapping.outputs.routineOutputY.value).toBe(true);
                });

                it("different object types", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputX: "." } },
                        { schemaIndex: 1, mapping: { routineOutputY: "." } },
                    ];
                    const outputConfig = [
                        { type: "object", properties: { key: { type: "string" } } },
                        { type: "object", properties: { key: { type: "number" } } },
                    ] as const;
                    const runOutputA = { output: { key: "value" } };
                    const runOutputB = { output: { key: 42 } };
                    config.parseSandboxOutput(runOutputA, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputX.value).toEqual({ key: "value" });
                    expect(ioMapping.outputs.routineOutputY.value).toBeNull();
                    config.parseSandboxOutput(runOutputB, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputX.value).toBeNull();
                    expect(ioMapping.outputs.routineOutputY.value).toEqual({ key: 42 });
                });
            });
        });

        describe("invalid test cases", () => {
            describe("single output", () => {
                it("trying to access an object property on a string output", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputZ: "value" } },
                    ];
                    const outputConfig = [{ type: "string" }] as const;
                    const runOutput = { output: "hello" };
                    config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputZ.value).toBeUndefined();
                });

                it("trying to access an array index on a string output", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputZ: "[0]" } },
                    ];
                    const outputConfig = [{ type: "string" }] as const;
                    const runOutput = { output: "hello" };
                    config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputZ.value).toBeUndefined();
                });

                it("output doesn't match any schema", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputX: "." } },
                    ];
                    const outputConfig = [{ type: "string" }] as const;
                    const runOutput = { output: 42 };
                    expect(() => config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig])).toThrow();
                });
            });

            describe("multiple outputs", () => {
                it("more output mappings than output configs", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputX: "." } },
                        { schemaIndex: 1, mapping: { routineOutputY: "." } },
                    ];
                    const outputConfig = [{ type: "string" }] as const;
                    const runOutput = { output: "hello" };
                    expect(() => config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig])).toThrow();
                });

                it("more output configs than output mappings", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputX: "." } },
                    ];
                    const outputConfig = [{ type: "string" }, { type: "number" }] as const;
                    const runOutput = { output: "hello" };
                    expect(() => config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig])).toThrow();
                });
            });
        });

        it("should throw an error if the sandboxed code returns an error", () => {
            const runOutput = { error: "Runtime error", output: undefined };
            expect(() =>
                config.parseSandboxOutput(runOutput, ioMapping, []),
            ).toThrow();
        });
    });
});

describe("RoutineVersionConfig", () => {
    const mockLogger = {
        log: vi.fn(),
        info: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
    };

    describe("constructor", () => {
        it("should create config with provided values", () => {
            const config = new RoutineVersionConfig({
                config: {
                    __version: "1.0",
                    resources: [],
                    callDataAction: {
                        __version: "1.0",
                        schema: {
                            toolName: McpToolName.SendMessage,
                            inputTemplate: "",
                            outputMapping: {},
                        },
                    },
                    formInput: {
                        __version: "1.0",
                        schema: {
                            fields: [],
                            steps: [],
                        },
                    },
                },
                resourceSubType: ResourceSubType.RoutineVersion,
            });

            expect(config.__version).toBe("1.0");
            expect(config.callDataAction).toBeDefined();
            expect(config.formInput).toBeDefined();
            expect(config.resourceSubType).toBe(ResourceSubType.RoutineVersion);
        });

        it("should handle minimal config", () => {
            const config = new RoutineVersionConfig({
                config: {
                    __version: "1.0",
                    resources: [],
                },
                resourceSubType: ResourceSubType.RoutineVersion,
            });

            expect(config.__version).toBe("1.0");
            expect(config.callDataAction).toBeUndefined();
            expect(config.callDataApi).toBeUndefined();
            expect(config.callDataCode).toBeUndefined();
            expect(config.formInput).toBeUndefined();
            expect(config.formOutput).toBeUndefined();
        });
    });

    describe("parse", () => {
        it("should parse valid config", () => {
            const version = {
                config: {
                    __version: "1.0",
                    resources: [],
                    callDataGenerate: {
                        __version: "1.0",
                        schema: {
                            botStyle: BotStyle.Casual,
                            maxTokens: 100,
                        },
                    },
                },
                resourceSubType: ResourceSubType.RoutineVersion,
            };

            const config = RoutineVersionConfig.parse(version, mockLogger);

            expect(config.callDataGenerate).toBeDefined();
            expect(config.callDataGenerate?.schema.botStyle).toBe(BotStyle.Casual);
            expect(config.callDataGenerate?.schema.maxTokens).toBe(100);
        });

        it("should handle missing config with fallbacks enabled", () => {
            const version = {
                config: null,
                resourceSubType: ResourceSubType.RoutineVersion,
            };

            const config = RoutineVersionConfig.parse(version, mockLogger, { useFallbacks: true });

            expect(config.__version).toBeDefined();
            expect(config.callDataAction).toBeDefined();
            expect(config.callDataApi).toBeDefined();
            expect(config.callDataCode).toBeDefined();
            expect(config.callDataGenerate).toBeDefined();
            expect(config.callDataSmartContract).toBeDefined();
            expect(config.callDataWeb).toBeDefined();
            expect(config.formInput).toBeDefined();
            expect(config.formOutput).toBeDefined();
        });

        it("should handle missing config with fallbacks disabled", () => {
            const version = {
                config: null,
                resourceSubType: ResourceSubType.RoutineVersion,
            };

            const config = RoutineVersionConfig.parse(version, mockLogger, { useFallbacks: false });

            expect(config.__version).toBeDefined();
            expect(config.resources).toEqual([]);
            expect(config.callDataAction).toBeUndefined();
            expect(config.callDataApi).toBeUndefined();
            expect(config.callDataCode).toBeUndefined();
            expect(config.formInput).toBeUndefined();
            expect(config.formOutput).toBeUndefined();
        });
    });

    describe("export", () => {
        it("should export all config properties", () => {
            const config = new RoutineVersionConfig({
                config: {
                    __version: "1.0",
                    resources: [],
                    callDataAction: {
                        __version: "1.0",
                        schema: {
                            toolName: McpToolName.SendMessage,
                            inputTemplate: "{}",
                            outputMapping: {},
                        },
                    },
                    callDataWeb: {
                        __version: "1.0",
                        schema: {
                            queryTemplate: "search for {{input.query}}",
                            outputMapping: { results: "results[*]" },
                        },
                    },
                    graph: {
                        __version: "1.0",
                        __type: "BPMN-2.0",
                        graph: "{\"nodes\": []}",
                    },
                },
                resourceSubType: ResourceSubType.RoutineVersion,
            });

            const exported = config.export();

            expect(exported.__version).toBe("1.0");
            expect(exported.callDataAction).toBeDefined();
            expect(exported.callDataWeb).toBeDefined();
            expect(exported.graph).toBeDefined();
            expect(exported.callDataApi).toBeUndefined();
            expect(exported.callDataCode).toBeUndefined();
        });
    });
});

describe("CallDataGenerateConfig", () => {
    describe("constructor", () => {
        it("should create config with provided values", () => {
            const config = new CallDataGenerateConfig({
                __version: "1.0",
                schema: {
                    botStyle: BotStyle.Funny,
                    maxTokens: 500,
                    model: "gpt-4",
                    prompt: "Generate a joke",
                    respondingBot: { id: "bot123" },
                },
            });

            expect(config.__version).toBe("1.0");
            expect(config.schema.botStyle).toBe(BotStyle.Funny);
            expect(config.schema.maxTokens).toBe(500);
            expect(config.schema.model).toBe("gpt-4");
            expect(config.schema.prompt).toBe("Generate a joke");
            expect(config.schema.respondingBot).toEqual({ id: "bot123" });
        });

        it("should handle minimal config", () => {
            const config = new CallDataGenerateConfig({
                __version: "1.0",
                schema: {},
            });

            expect(config.__version).toBe("1.0");
            expect(config.schema.botStyle).toBeUndefined();
            expect(config.schema.maxTokens).toBeUndefined();
            expect(config.schema.model).toBeUndefined();
            expect(config.schema.prompt).toBeUndefined();
            expect(config.schema.respondingBot).toBeUndefined();
        });
    });

    describe("export", () => {
        it("should export config correctly", () => {
            const config = new CallDataGenerateConfig({
                __version: "1.0",
                schema: {
                    botStyle: BotStyle.Professional,
                    maxTokens: 1000,
                },
            });

            const exported = config.export();

            expect(exported.__version).toBe("1.0");
            expect(exported.schema.botStyle).toBe(BotStyle.Professional);
            expect(exported.schema.maxTokens).toBe(1000);
        });
    });
});

describe("CallDataWebConfig", () => {
    describe("constructor", () => {
        it("should create config with provided values", () => {
            const config = new CallDataWebConfig({
                __version: "1.0",
                schema: {
                    queryTemplate: "latest news about {{input.topic}}",
                    searchEngine: "google",
                    maxResults: 10,
                    timeRange: "past_week",
                    region: "US",
                    outputMapping: {
                        titles: "results[*].title",
                        links: "results[*].link",
                    },
                },
            });

            expect(config.__version).toBe("1.0");
            expect(config.schema.queryTemplate).toBe("latest news about {{input.topic}}");
            expect(config.schema.searchEngine).toBe("google");
            expect(config.schema.maxResults).toBe(10);
            expect(config.schema.timeRange).toBe("past_week");
            expect(config.schema.region).toBe("US");
            expect(config.schema.outputMapping).toEqual({
                titles: "results[*].title",
                links: "results[*].link",
            });
        });

        it("should handle minimal config", () => {
            const config = new CallDataWebConfig({
                __version: "1.0",
                schema: {
                    queryTemplate: "{{input.query}}",
                    outputMapping: {},
                },
            });

            expect(config.__version).toBe("1.0");
            expect(config.schema.queryTemplate).toBe("{{input.query}}");
            expect(config.schema.outputMapping).toEqual({});
            expect(config.schema.searchEngine).toBeUndefined();
            expect(config.schema.maxResults).toBeUndefined();
        });
    });

    describe("buildSearchQuery", () => {
        it("should replace placeholders in query template", () => {
            const config = new CallDataWebConfig({
                __version: "1.0",
                schema: {
                    queryTemplate: "AI news {{input.year}}",
                    outputMapping: {},
                },
            });

            const ioMapping = {
                inputs: {
                    year: { value: "2024" },
                },
                outputs: {},
            } as unknown as SubroutineIOMapping;

            const query = config.buildSearchQuery(ioMapping);
            expect(query).toBe("AI news 2024");
        });

        it("should handle multiple placeholders", () => {
            const config = new CallDataWebConfig({
                __version: "1.0",
                schema: {
                    queryTemplate: "{{input.topic}} in {{input.location}} {{input.year}}",
                    outputMapping: {},
                },
            });

            const ioMapping = {
                inputs: {
                    topic: { value: "technology trends" },
                    location: { value: "Silicon Valley" },
                    year: { value: "2024" },
                },
                outputs: {},
            } as unknown as SubroutineIOMapping;

            const query = config.buildSearchQuery(ioMapping);
            expect(query).toBe("technology trends in Silicon Valley 2024");
        });
    });

    describe("parseSearchResult", () => {
        it("should map search results to outputs", () => {
            const config = new CallDataWebConfig({
                __version: "1.0",
                schema: {
                    queryTemplate: "test",
                    outputMapping: {
                        titles: "results[*].title",
                        firstLink: "results[0].link",
                    },
                },
            });

            const searchResult = {
                results: [
                    { title: "Result 1", link: "http://example1.com" },
                    { title: "Result 2", link: "http://example2.com" },
                ],
            };

            const ioMapping = {
                inputs: {},
                outputs: {
                    titles: { value: null },
                    firstLink: { value: null },
                },
            } as unknown as SubroutineIOMapping;

            config.parseSearchResult(ioMapping, searchResult);

            // The parseSearchResult should set the values correctly
            expect(ioMapping.outputs.titles.value).toEqual(["Result 1", "Result 2"]);
            expect(ioMapping.outputs.firstLink.value).toBe("http://example1.com");
        });

        it("should handle missing paths gracefully", () => {
            const config = new CallDataWebConfig({
                __version: "1.0",
                schema: {
                    queryTemplate: "test",
                    outputMapping: {
                        missing: "results[10].title",
                    },
                },
            });

            const searchResult = {
                results: [{ title: "Result 1", link: "http://example1.com" }],
            };

            const ioMapping = {
                inputs: {},
                outputs: {
                    missing: { value: null },
                },
            } as unknown as SubroutineIOMapping;

            expect(() => config.parseSearchResult(ioMapping, searchResult)).not.toThrow();
            // When the path doesn't exist, it should be set to null (not undefined)
            expect(ioMapping.outputs.missing.value).toBeNull();
        });
    });

    describe("export", () => {
        it("should export config correctly", () => {
            const config = new CallDataWebConfig({
                __version: "1.0",
                schema: {
                    queryTemplate: "search query",
                    searchEngine: "bing",
                    outputMapping: { result: "data" },
                },
            });

            const exported = config.export();

            expect(exported.__version).toBe("1.0");
            expect(exported.schema.queryTemplate).toBe("search query");
            expect(exported.schema.searchEngine).toBe("bing");
            expect(exported.schema.outputMapping).toEqual({ result: "data" });
        });
    });
});

describe("GraphBpmnConfig", () => {
    describe("constructor", () => {
        it("should create BPMN config with provided values", () => {
            const config = new GraphBpmnConfig({
                __version: "1.0",
                __type: "BPMN-2.0",
                schema: {
                    __format: "xml",
                    data: "<bpmn:definitions></bpmn:definitions>",
                    activityMap: {
                        activityA: {
                            subroutineId: "sub123",
                            inputMap: { input1: "subInput1" },
                            outputMap: { output1: "subOutput1" },
                        },
                    },
                    rootContext: {
                        inputMap: { mainInput: "routineInput" },
                        outputMap: { mainOutput: "routineOutput" },
                    },
                },
            });

            expect(config.__version).toBe("1.0");
            expect(config.__type).toBe("BPMN-2.0");
            expect(config.schema.__format).toBe("xml");
            expect(config.schema.data).toBe("<bpmn:definitions></bpmn:definitions>");
            expect(config.schema.activityMap.activityA.subroutineId).toBe("sub123");
        });
    });

    describe("create static method", () => {
        it("should create appropriate graph config based on type", () => {
            const configData = {
                __version: "1.0",
                __type: "BPMN-2.0" as const,
                schema: {
                    __format: "xml" as const,
                    data: "<bpmn:process id=\"Process_1\"></bpmn:process>",
                    activityMap: {},
                    rootContext: {
                        inputMap: {},
                        outputMap: {},
                    },
                },
            };

            const config = GraphBpmnConfig.create(configData);

            expect(config).toBeInstanceOf(GraphBpmnConfig);
            expect(config.__type).toBe("BPMN-2.0");
            expect(config.schema.data).toBe("<bpmn:process id=\"Process_1\"></bpmn:process>");
        });
    });

    describe("export", () => {
        it("should export BPMN config correctly", () => {
            const config = new GraphBpmnConfig({
                __version: "1.0",
                __type: "BPMN-2.0",
                schema: {
                    __format: "xml",
                    data: "<bpmn:definitions id=\"definitions\"></bpmn:definitions>",
                    activityMap: {
                        taskA: {
                            subroutineId: "routine123",
                            inputMap: { a: "x" },
                            outputMap: { y: "b" },
                        },
                    },
                    rootContext: {
                        inputMap: { start: "init" },
                        outputMap: { end: "result" },
                    },
                },
            });

            const exported = config.export();

            expect(exported.__version).toBe("1.0");
            expect(exported.__type).toBe("BPMN-2.0");
            expect(exported.schema.__format).toBe("xml");
            expect(exported.schema.data).toBe("<bpmn:definitions id=\"definitions\"></bpmn:definitions>");
            expect(exported.schema.activityMap.taskA.subroutineId).toBe("routine123");
        });
    });

    describe("getIONamesToSubroutineInputNames", () => {
        it("should return input mappings for existing node", () => {
            const config = new GraphBpmnConfig({
                __version: "1.0",
                __type: "BPMN-2.0",
                schema: {
                    __format: "xml",
                    data: "",
                    activityMap: {
                        nodeA: {
                            subroutineId: "sub1",
                            inputMap: { input1: "subInput1", input2: "subInput2" },
                            outputMap: {},
                        },
                    },
                    rootContext: { inputMap: {}, outputMap: {} },
                },
            });

            const result = config.getIONamesToSubroutineInputNames("nodeA");
            expect(result).toEqual({ input1: "subInput1", input2: "subInput2" });
        });

        it("should return empty object for non-existent node", () => {
            const config = new GraphBpmnConfig({
                __version: "1.0",
                __type: "BPMN-2.0",
                schema: {
                    __format: "xml",
                    data: "",
                    activityMap: {},
                    rootContext: { inputMap: {}, outputMap: {} },
                },
            });

            const result = config.getIONamesToSubroutineInputNames("nonExistent");
            expect(result).toEqual({});
        });
    });

    describe("getIONamesToSubroutineOutputNames", () => {
        it("should return output mappings for existing node", () => {
            const config = new GraphBpmnConfig({
                __version: "1.0",
                __type: "BPMN-2.0",
                schema: {
                    __format: "xml",
                    data: "",
                    activityMap: {
                        nodeB: {
                            subroutineId: "sub2",
                            inputMap: {},
                            outputMap: { output1: "subOutput1", output2: "subOutput2" },
                        },
                    },
                    rootContext: { inputMap: {}, outputMap: {} },
                },
            });

            const result = config.getIONamesToSubroutineOutputNames("nodeB");
            expect(result).toEqual({ output1: "subOutput1", output2: "subOutput2" });
        });
    });

    describe("getRootIONamesToRoutineInputNames", () => {
        it("should return prefixed root input mappings", () => {
            const config = new GraphBpmnConfig({
                __version: "1.0",
                __type: "BPMN-2.0",
                schema: {
                    __format: "xml",
                    data: "",
                    activityMap: {},
                    rootContext: {
                        inputMap: { start: "routineStart", data: "routineData" },
                        outputMap: {},
                    },
                },
            });

            const result = config.getRootIONamesToRoutineInputNames();
            expect(result).toEqual({
                "root.start": "routineStart",
                "root.data": "routineData",
            });
        });
    });

    describe("getRootIONamesToRoutineOutputNames", () => {
        it("should return prefixed root output mappings", () => {
            const config = new GraphBpmnConfig({
                __version: "1.0",
                __type: "BPMN-2.0",
                schema: {
                    __format: "xml",
                    data: "",
                    activityMap: {},
                    rootContext: {
                        inputMap: {},
                        outputMap: { result: "routineResult", status: "routineStatus" },
                    },
                },
            });

            const result = config.getRootIONamesToRoutineOutputNames();
            expect(result).toEqual({
                "root.result": "routineResult",
                "root.status": "routineStatus",
            });
        });
    });
});
