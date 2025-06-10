/* eslint-disable @typescript-eslint/ban-ts-comment */
import { describe, it, expect, vi, beforeAll, beforeEach, afterAll } from "vitest";
import { McpToolName } from "../../consts/mcp.js";
import { validatePublicId } from "../../id/publicId.js";
import { type SubroutineIOMapping } from "../../run/types.js";
import { type CodeVersionInputDefinition } from "./code.js";
import { CallDataActionConfig, CallDataCodeConfig } from "./routine.js";

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
                    expect(result).to.equal(ioMapping.inputs.routineInputA.value);
                });

                it("should replace numeric input placeholders", () => {
                    const str = "{{input.routineInputC}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).to.equal(ioMapping.inputs.routineInputC.value);
                });

                it("should replace boolean input placeholders", () => {
                    const str = "{{input.routineInputD}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).to.equal(ioMapping.inputs.routineInputD.value);
                });

                it("should replace array input placeholders", () => {
                    const str = "{{input.routineInputF}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).to.deep.equal(ioMapping.inputs.routineInputF.value);
                });

                it("should replace object input placeholders", () => {
                    const str = "{{input.routineInputH}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).to.deep.equal(ioMapping.inputs.routineInputH.value);
                });

                it("should replace date input placeholders", () => {
                    const str = "{{input.routineInputJ}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).to.equal(ioMapping.inputs.routineInputJ.value);
                });

                it("should replace null input placeholders", () => {
                    const str = "{{input.routineInputL}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).to.equal(ioMapping.inputs.routineInputL.value);
                });

                it("should handle multiple placeholders in a single string", () => {
                    const str = "A: {{input.routineInputA}}, B: {{input.routineInputC}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).to.equal(`A: ${ioMapping.inputs.routineInputA.value}, B: ${ioMapping.inputs.routineInputC.value}`);
                });

                it("should leave static strings unchanged", () => {
                    const str = "static string";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).to.equal(str);
                });
            });

            describe("invalid test cases", () => {
                it("should throw an error for missing input", () => {
                    const str = "{{input.missingInput}}";
                    expect(() => config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} })).to.throw(
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
                    expect(result).to.equal("en");
                });

                it("should replace userLanguages placeholder with the full language array", () => {
                    const str = "{{userLanguages}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(result).to.deep.equal(["en", "es"]);
                });

                it("should replace now() placeholder with a valid ISO date string", () => {
                    const str = "{{now()}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    const asDate = new Date(result as string);
                    expect(asDate.toISOString()).to.equal(result);
                });

                it("should replace nanoid() placeholder with a valid id", () => {
                    const str = "{{nanoid()}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    expect(validatePublicId(result)).to.be.true;
                });

                it("should replace nanoid(seed) placeholder with consistent id for the same seed", () => {
                    const str = "{{nanoid(123)}}|{{nanoid(123)}}|{{nanoid(456)}}|{{nanoid(456)}}";
                    const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                    const ids = (result as string).split("|");
                    expect(ids[0]).to.equal(ids[1]);
                    expect(ids[2]).to.equal(ids[3]);
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
                    expect(() => config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} })).to.throw();
                });
            });
        });

        describe("Error Handling", () => {
            it("should throw an error for unknown placeholders", () => {
                const str = "{{unknown}}";
                expect(() => config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} })).to.throw();
            });

            it("should not transform malformed placeholders", () => {
                const str = "{{input.routineInputA";
                const result = config["replacePlaceholders"](str, { inputs: ioMapping.inputs, userLanguages, seededIds: {} });
                expect(result).to.equal(str);
            });
        });
    });

    describe("buildTaskInput", () => {
        describe("Basic Placeholder Replacement", () => {
            it("should replace simple string input placeholders", () => {
                config.schema.inputTemplate = "{\"field\": \"{{input.routineInputA}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).to.deep.equal({ field: ioMapping.inputs.routineInputA.value });
            });

            it("should replace numeric input placeholders", () => {
                config.schema.inputTemplate = "{\"number\": \"{{input.routineInputC}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).to.deep.equal({ number: ioMapping.inputs.routineInputC.value });
            });

            it("should replace boolean input placeholders", () => {
                config.schema.inputTemplate = "{\"bool\": \"{{input.routineInputD}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).to.deep.equal({ bool: ioMapping.inputs.routineInputD.value });
            });

            it("should replace array input placeholders", () => {
                config.schema.inputTemplate = "{\"array\": \"{{input.routineInputF}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).to.deep.equal({ array: ioMapping.inputs.routineInputF.value });
            });

            it("should replace object input placeholders", () => {
                config.schema.inputTemplate = "{\"object\": \"{{input.routineInputH}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).to.deep.equal({ object: ioMapping.inputs.routineInputH.value });
            });
        });

        describe("Special Function Placeholders", () => {
            it("should replace {{nanoid()}} with a valid id", () => {
                config.schema.inputTemplate = "{\"id\": \"{{nanoid()}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).to.have.property("id");
                expect(validatePublicId(result.id as string)).to.be.true;
            });

            it("should generate consistent ids for the same seed across the template", () => {
                config.schema.inputTemplate = "{\"id1\": \"{{nanoid(123)}}\", \"child\": {\"id2\": \"{{nanoid(123)}}\", \"id3\": \"{{nanoid(456)}}\"}}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                // @ts-ignore Testing runtime scenario
                expect(result.id1).to.equal(result.child.id2);
                // @ts-ignore Testing runtime scenario
                expect(result.id1).to.not.equal(result.child.id3);
                expect(validatePublicId(result.id1 as string)).to.be.true;
                // @ts-ignore Testing runtime scenario
                expect(validatePublicId(result.child.id3 as string)).to.be.true;
            });

            it("should replace {{userLanguage}} with the first language", () => {
                config.schema.inputTemplate = "{\"lang\": \"{{userLanguage}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).to.deep.equal({ lang: userLanguages[0] });
            });

            it("should replace {{userLanguages}} with the full language array", () => {
                config.schema.inputTemplate = "{\"langs\": \"{{userLanguages}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).to.deep.equal({ langs: userLanguages });
            });

            it("should replace {{now()}} with a valid ISO date string", () => {
                config.schema.inputTemplate = "{\"date\": \"{{now()}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(new Date(result.date as string).toISOString()).to.equal(result.date);
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
                expect(Number.isInteger(result.num)).to.be.true;
            });
        });

        describe("Nested Structures", () => {
            it("should replace placeholders in nested objects", () => {
                config.schema.inputTemplate = "{\"nested\": {\"field\": \"{{input.routineInputB}}\"}}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).to.deep.equal({ nested: { field: ioMapping.inputs.routineInputB.value } });
            });

            it("should replace placeholders in arrays", () => {
                config.schema.inputTemplate = "{\"items\": [\"{{input.routineInputA}}\", \"{{input.routineInputB}}\"]}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).to.deep.equal({ items: [ioMapping.inputs.routineInputA.value, ioMapping.inputs.routineInputB.value] });
            });

            it("should replace placeholders in arrays of objects", () => {
                config.schema.inputTemplate = "{\"users\": [{\"name\": \"{{input.routineInputA}}\"}, {\"name\": \"{{input.routineInputB}}\"}]}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).to.deep.equal({ users: [{ name: ioMapping.inputs.routineInputA.value }, { name: ioMapping.inputs.routineInputB.value }] });
            });
        });

        describe("Multiple Placeholders", () => {
            it("should replace multiple placeholders in a string", () => {
                config.schema.inputTemplate = "{\"message\": \"Hello, {{input.routineInputA}}! Score: {{input.routineInputC}}.\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).to.deep.equal({ message: `Hello, ${ioMapping.inputs.routineInputA.value}! Score: ${ioMapping.inputs.routineInputC.value}.` });
            });

            it("should replace multiple placeholders across object fields", () => {
                config.schema.inputTemplate = "{\"name\": \"{{input.routineInputA}}\", \"score\": \"{{input.routineInputC}}\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).to.deep.equal({ name: ioMapping.inputs.routineInputA.value, score: ioMapping.inputs.routineInputC.value });
            });
        });

        describe("Entire String as Placeholder", () => {
            it("should return raw string value when template is a single placeholder", () => {
                config.schema.inputTemplate = "\"{{input.routineInputA}}\"";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).to.equal(ioMapping.inputs.routineInputA.value);
            });

            it("should return raw number value when template is a single placeholder", () => {
                config.schema.inputTemplate = "\"{{input.routineInputC}}\"";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).to.equal(ioMapping.inputs.routineInputC.value);
            });

            it("should return raw array value when template is a single placeholder", () => {
                config.schema.inputTemplate = "\"{{input.routineInputF}}\"";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).to.deep.equal(ioMapping.inputs.routineInputF.value);
            });
        });

        describe("Primitive Templates", () => {
            it("should return a primitive string without placeholders", () => {
                config.schema.inputTemplate = "\"hello\"";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).to.equal("hello");
            });

            it("should return a primitive number", () => {
                config.schema.inputTemplate = "42";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).to.equal(42);
            });

            it("should return a primitive boolean", () => {
                config.schema.inputTemplate = "true";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).to.equal(true);
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
                expect(result).to.have.property("id").that.is.a("string");
                expect(validatePublicId(result.id)).to.be.true;

                // Check user object
                expect(result).to.have.property("user").that.is.an("object");
                expect(result.user).to.have.property("name", ioMapping.inputs.routineInputA.value);
                expect(result.user).to.have.property("languages").that.deep.equals(userLanguages);
                expect(result.user).to.have.property("registrationDate").that.is.a("string");

                // Validate registrationDate is a valid ISO date
                // @ts-ignore Testing runtime scenario
                const registrationDate = result.user.registrationDate;
                expect(new Date(registrationDate).toISOString()).to.equal(registrationDate);

                // Check scores array
                expect(result).to.have.property("scores").that.is.an("array").with.lengthOf(2);

                // Check scores[0]
                // @ts-ignore Testing runtime scenario
                expect(result.scores[0]).to.have.property("level", 1);
                // @ts-ignore Testing runtime scenario
                expect(result.scores[0]).to.have.property("score")
                    .that.is.a("number")
                    .and.satisfies(Number.isInteger)
                    .and.within(0, 100);

                // Check scores[1]
                // @ts-ignore Testing runtime scenario
                expect(result.scores[1]).to.have.property("level", 2);
                // @ts-ignore Testing runtime scenario
                expect(result.scores[1]).to.have.property("score")
                    .that.is.a("number")
                    .and.satisfies(Number.isInteger)
                    .and.within(0, 100);
            });
        });

        describe("Error Handling", () => {
            it("should throw an error for missing input", () => {
                config.schema.inputTemplate = "{\"field\": \"{{input.missingInput}}\"}";
                ioMapping.inputs = {};
                expect(() => config.buildTaskInput(ioMapping, userLanguages)).to.throw("Input \"missingInput\" not found");
            });

            it("should throw an error for invalid JSON in inputTemplate", () => {
                config.schema.inputTemplate = "{ invalid json }";
                expect(() => config.buildTaskInput(ioMapping, userLanguages)).to.throw("Failed to parse inputTemplate");
            });

            it("should preserve malformed placeholders without throwing", () => {
                config.schema.inputTemplate = "{\"field\": \"{{input.routineInputA\"}";
                const result = config.buildTaskInput(ioMapping, userLanguages);
                expect(result).to.deep.equal({ field: "{{input.routineInputA" });
            });

            it("should throw an error for invalid arguments in random()", () => {
                config.schema.inputTemplate = "{\"num\": \"{{random(abc, 10)}}\"}";
                expect(() => config.buildTaskInput(ioMapping, userLanguages)).to.throw("Invalid arguments for random()");
            });
        });
    });

    describe("parseActionResult", () => {
        it("should map a simple property from result to output", () => {
            config.schema.outputMapping = { outputPayload: "." };
            ioMapping.outputs = { outputPayload: { value: null } } as unknown as SubroutineIOMapping["outputs"];
            const result = { payload: "some value" };
            config.parseActionResult(ioMapping, result);
            expect(ioMapping.outputs.outputPayload.value).to.equal(result.payload);
        });

        it("should map a nested property from result to output", () => {
            config.schema.outputMapping = { outputId: "data.id" };
            ioMapping.outputs = { outputId: { value: null } } as unknown as SubroutineIOMapping["outputs"];
            const result = { payload: { data: { id: 123 } } };
            config.parseActionResult(ioMapping, result);
            expect(ioMapping.outputs.outputId.value).to.equal(result.payload.data.id);
        });

        it("should not update output if the path is missing in result", () => {
            config.schema.outputMapping = { "outputMissing": "missing" };
            ioMapping.outputs = { outputMissing: { value: null } } as unknown as SubroutineIOMapping["outputs"];
            const result = { payload: { data: { id: 123 } } };
            config.parseActionResult(ioMapping, result);
            expect(ioMapping.outputs.outputMissing.value).to.be.null;
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
            expect(ioMapping.outputs.outputId.value).to.equal(result.payload.data.id);
            expect(ioMapping.outputs.outputName.value).to.equal(result.payload.data.name);
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
            expect(ioMapping.outputs.outputBool.value).to.equal(result.payload.bool);
            expect(ioMapping.outputs.outputNum.value).to.equal(result.payload.num);
            expect(ioMapping.outputs.outputObj.value).to.deep.equal(result.payload.obj);
        });

        it("should map an array element correctly", () => {
            config.schema.outputMapping = { "firstItem": "1" };
            ioMapping.outputs = { firstItem: { value: null } } as unknown as SubroutineIOMapping["outputs"];
            const result = { payload: ["a", "b", "c"] };
            config.parseActionResult(ioMapping, result);
            expect(ioMapping.outputs.firstItem.value).to.equal(result.payload[1]);
        });

        it("should do nothing if result has no payload", () => {
            config.schema.outputMapping = { "outputData": "data" };
            ioMapping.outputs = { outputData: { value: null } } as unknown as SubroutineIOMapping["outputs"];
            const result = { other: "data" };
            // @ts-ignore Testing runtime scenario
            config.parseActionResult(ioMapping, result);
            expect(ioMapping.outputs.outputData.value).to.be.null;
        });

        it("should handle null payload correctly", () => {
            config.schema.outputMapping = { "outputSomething": "." };
            ioMapping.outputs = { outputSomething: { value: "not-null" } } as unknown as SubroutineIOMapping["outputs"];
            const result = { payload: null };
            config.parseActionResult(ioMapping, result);
            expect(ioMapping.outputs.outputSomething.value).to.be.null; // Overwrites the previous value with the requested value (null)
        });

        it("should handle deeply nested properties", () => {
            config.schema.outputMapping = { "outputDeep": "a.b.c.d" };
            ioMapping.outputs = { outputDeep: { value: null } } as unknown as SubroutineIOMapping["outputs"];
            const result = { payload: { a: { b: { c: { d: "deep value" } } } } };
            config.parseActionResult(ioMapping, result);
            expect(ioMapping.outputs.outputDeep.value).to.equal(result.payload.a.b.c.d);
        });

        it("should not update output if nested property does not exist", () => {
            config.schema.outputMapping = { "outputMissing": "data.missing" };
            ioMapping.outputs = { outputMissing: { value: null } } as unknown as SubroutineIOMapping["outputs"];
            const result = { payload: { data: { id: 123 } } };
            config.parseActionResult(ioMapping, result);
            expect(ioMapping.outputs.outputMissing.value).to.be.null;
        });

        it("should not modify unmapped outputs", () => {
            config.schema.outputMapping = { "outputId": "data.id" };
            ioMapping.outputs = {
                outputId: { value: null },
                otherOutput: { value: "original" },
            } as unknown as SubroutineIOMapping["outputs"];
            const result = { payload: { data: { id: 123 } } };
            config.parseActionResult(ioMapping, result);
            expect(ioMapping.outputs.outputId.value).to.equal(result.payload.data.id);
            expect(ioMapping.outputs.otherOutput.value).to.equal("original");
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
                    expect(result.input).to.deep.equal(["valueA", "valueB"]);
                    expect(result.shouldSpreadInput).to.be.true;
                });

                it("should handle nested placeholders in an array", () => {
                    config.schema.inputTemplate = ["{{routineInputA}}", ["{{routineInputB}}", "{{routineInputC}}"]];
                    const result = config.buildSandboxInput(ioMapping, inputConfig);
                    expect(result.input).to.deep.equal(["valueA", ["valueB", 42]]);
                });

                it("should handle a single placeholder in an array", () => {
                    config.schema.inputTemplate = ["{{routineInputA}}"];
                    const result = config.buildSandboxInput(ioMapping, inputConfig);
                    expect(result.input).to.deep.equal(["valueA"]);
                });

                it("should leave a string untransformed if it doesn't contain placeholders", () => {
                    config.schema.inputTemplate = ["static string"];
                    const result = config.buildSandboxInput(ioMapping, inputConfig);
                    expect(result.input).to.deep.equal(["static string"]);
                });

                it("should be able to handle a mix of placeholders and static strings", () => {
                    config.schema.inputTemplate = ["{{routineInputA}}", "static string", "placeholder: {{routineInputB}}"];
                    const result = config.buildSandboxInput(ioMapping, inputConfig);
                    expect(result.input).to.deep.equal(["valueA", "static string", "placeholder: valueB"]);
                });

                it("should handle plaeholders of all types", () => {
                    config.schema.inputTemplate = ["{{routineInputA}}", "{{routineInputB}}", "{{routineInputC}}", "{{routineInputD}}", "{{routineInputE}}", "{{routineInputF}}", "{{routineInputG}}", "{{routineInputH}}", "{{routineInputI}}", "{{routineInputJ}}", "{{routineInputK}}", "{{routineInputL}}"];
                    const result = config.buildSandboxInput(ioMapping, inputConfig);
                    expect(result.input).to.deep.equal(["valueA", "valueB", 42, true, false, ["item1", "item2"], [1, 2, 3], { key: "value" }, { nested: { key: "value" } }, new Date("2024-01-01"), new Date("2024-01-01T12:00:00Z"), null]);
                });

                it("should correctly stringify all types when using string interpolation", () => {
                    config.schema.inputTemplate = ["a: {{routineInputA}}", "b: {{routineInputB}}", "c: {{routineInputC}}", "d: {{routineInputD}}", "e: {{routineInputE}}", "f: {{routineInputF}}", "g: {{routineInputG}}", "h: {{routineInputH}}", "i: {{routineInputI}}", "j: {{routineInputJ}}", "k: {{routineInputK}}", "l: {{routineInputL}}"];
                    const result = config.buildSandboxInput(ioMapping, inputConfig);
                    expect(result.input).to.deep.equal(["a: valueA", "b: valueB", "c: 42", "d: true", "e: false", "f: [\"item1\",\"item2\"]", "g: [1,2,3]", "h: {\"key\":\"value\"}", "i: {\"nested\":{\"key\":\"value\"}}", "j: 2024-01-01T00:00:00.000Z", "k: 2024-01-01T12:00:00.000Z", "l: null"]);
                });
            });

            describe("invalid test cases", () => {
                it("should throw an error if a placeholder input is missing", () => {
                    config.schema.inputTemplate = ["{{missingInput}}"];
                    expect(() => config.buildSandboxInput(ioMapping, inputConfig)).to.throw(
                        "Input \"missingInput\" not found in ioMapping or missing value",
                    );
                });

                it("should throw an error if the template is not an array", () => {
                    config.schema.inputTemplate = { key: "{{routineInputA}}" };
                    expect(() => config.buildSandboxInput(ioMapping, inputConfig)).to.throw();
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
                    expect(result.input).to.deep.equal({ key: "valueA", nested: { value: "placeholder 1: valueB, placeholder 2: 42" } });
                    expect(result.shouldSpreadInput).to.be.false;
                });
            });

            describe("invalid test cases", () => {
                it("should throw an error if a placeholder input is missing", () => {
                    config.schema.inputTemplate = { key: "{{missingInput}}" };
                    expect(() => config.buildSandboxInput(ioMapping, inputConfig)).to.throw(
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
                    expect(ioMapping.outputs.routineOutputZ.value).to.equal("hello");
                });

                it("number", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputZ: "." } },
                    ];
                    const outputConfig = [{ type: "number" }] as const;
                    const runOutput = { output: 42 };
                    config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputZ.value).to.equal(42);
                });

                it("boolean", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputZ: "." } },
                    ];
                    const outputConfig = [{ type: "boolean" }] as const;
                    const runOutput = { output: true };
                    config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputZ.value).to.equal(true);
                });

                it("full array", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputZ: "." } },
                    ];
                    const outputConfig = [{ type: "array", items: { type: "string" } }] as const;
                    const runOutput = { output: ["item1", "item2"] };
                    config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputZ.value).to.deep.equal(["item1", "item2"]);
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
                    expect(ioMapping.outputs.routineOutputZ.value).to.equal("item2");
                });

                it("object", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputZ: "." } },
                    ];
                    const outputConfig = [{ type: "object", properties: { key: { type: "string" } } }] as const;
                    const runOutput = { output: { key: "value" } };
                    config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputZ.value).to.deep.equal({ key: "value" });
                });

                it("null", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputZ: "." } },
                    ];
                    const outputConfig = [{ type: "null" }] as const;
                    const runOutput = { output: null };
                    config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputZ.value).to.be.null;
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
                    expect(ioMapping.outputs.routineOutputX.value).to.equal("hello");
                    expect(ioMapping.outputs.routineOutputY.value).to.equal(42);
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
                    expect(ioMapping.outputs.routineOutputX.value).to.be.null;
                    expect(ioMapping.outputs.routineOutputY.value).to.deep.equal({ key: "value" });
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
                    expect(ioMapping.outputs.routineOutputX.value).to.deep.equal({ result: "hello", message: "world" });
                    expect(ioMapping.outputs.routineOutputY.value).to.equal("world");
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
                    expect(ioMapping.outputs.routineOutputX.value).to.equal("hello");
                    expect(ioMapping.outputs.routineOutputY.value).to.be.null;
                    config.parseSandboxOutput(runOutputB, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputX.value).to.be.null;
                    expect(ioMapping.outputs.routineOutputY.value).to.equal(true);
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
                    expect(ioMapping.outputs.routineOutputX.value).to.equal("hello");
                    expect(ioMapping.outputs.routineOutputY.value).to.be.null;
                    config.parseSandboxOutput(runOutputB, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputX.value).to.be.null;
                    expect(ioMapping.outputs.routineOutputY.value).to.equal(true);
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
                    expect(ioMapping.outputs.routineOutputX.value).to.deep.equal({ key: "value" });
                    expect(ioMapping.outputs.routineOutputY.value).to.be.null;
                    config.parseSandboxOutput(runOutputB, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputX.value).to.be.null;
                    expect(ioMapping.outputs.routineOutputY.value).to.deep.equal({ key: 42 });
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
                    expect(ioMapping.outputs.routineOutputZ.value).to.be.undefined;
                });

                it("trying to access an array index on a string output", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputZ: "[0]" } },
                    ];
                    const outputConfig = [{ type: "string" }] as const;
                    const runOutput = { output: "hello" };
                    config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig]);
                    expect(ioMapping.outputs.routineOutputZ.value).to.be.undefined;
                });

                it("output doesn't match any schema", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputX: "." } },
                    ];
                    const outputConfig = [{ type: "string" }] as const;
                    const runOutput = { output: 42 };
                    expect(() => config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig])).to.throw();
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
                    expect(() => config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig])).to.throw();
                });

                it("more output configs than output mappings", () => {
                    config.schema.outputMappings = [
                        { schemaIndex: 0, mapping: { routineOutputX: "." } },
                    ];
                    const outputConfig = [{ type: "string" }, { type: "number" }] as const;
                    const runOutput = { output: "hello" };
                    expect(() => config.parseSandboxOutput(runOutput, ioMapping, [...outputConfig])).to.throw();
                });
            });
        });

        it("should throw an error if the sandboxed code returns an error", () => {
            const runOutput = { error: "Runtime error", output: undefined };
            expect(() =>
                config.parseSandboxOutput(runOutput, ioMapping, []),
            ).to.throw();
        });
    });
});
