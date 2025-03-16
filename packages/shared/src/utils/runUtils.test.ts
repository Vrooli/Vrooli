/* eslint-disable @typescript-eslint/ban-ts-comment */
/* eslint-disable @typescript-eslint/no-explicit-any */
import { Node, NodeLink, NodeType, Project, ProjectVersion, ProjectVersionDirectory, ProjectVersionYou, Routine, RoutineType, RoutineVersion, RoutineVersionInput, RoutineVersionOutput, RoutineVersionYou, RunProject, RunProjectStep, RunRoutine, RunRoutineInput, RunRoutineStep, RunRoutineStepStatus, RunStatus } from "../api/types";
import { InputType } from "../consts";
import { FormSchema, FormStructureType } from "../forms";
import { uuid, uuidValidate } from "../id/uuid";
import { DecisionStep, DirectoryStep, EndStep, ExistingInput, ExistingOutput, MultiRoutineStep, RootStep, RoutineListStep, RunIOManager, RunIOUpdateParams, RunRequestLimits, RunStatusChangeReason, RunStep, RunStepBuilder, RunStepNavigator, RunStepType, RunnableProjectVersion, RunnableRoutineVersion, ShouldStopParams, SingleRoutineStep, StartStep, defaultConfigFormInputMap, defaultConfigFormOutputMap, generateRoutineInitialValues, getRunPercentComplete, getStepComplexity, parseSchemaInput, parseSchemaOutput, routineVersionHasSubroutines, saveRunProgress, shouldStopRun } from "./runUtils";

describe("getRunPercentComplete", () => {
    it("should return 0 when completedComplexity is null", () => {
        expect(getRunPercentComplete(null, 100)).toBe(0);
    });

    it("should return 0 when completedComplexity is undefined", () => {
        expect(getRunPercentComplete(undefined, 100)).toBe(0);
    });

    it("should return 0 when totalComplexity is null", () => {
        expect(getRunPercentComplete(50, null)).toBe(0);
    });

    it("should return 0 when totalComplexity is undefined", () => {
        expect(getRunPercentComplete(50, undefined)).toBe(0);
    });

    it("should return 0 when totalComplexity is 0", () => {
        expect(getRunPercentComplete(50, 0)).toBe(0);
    });

    it("should return 50 when half of the complexity is completed", () => {
        expect(getRunPercentComplete(50, 100)).toBe(50);
    });

    it("should return 100 when all complexity is completed", () => {
        expect(getRunPercentComplete(100, 100)).toBe(100);
    });

    it("should return 100 when completed complexity exceeds total complexity", () => {
        expect(getRunPercentComplete(150, 100)).toBe(100);
    });

    it("should round down to the nearest integer", () => {
        expect(getRunPercentComplete(66, 100)).toBe(66);
    });

    it("should round up to the nearest integer", () => {
        expect(getRunPercentComplete(67, 100)).toBe(67);
    });

    it("should handle very small fractions", () => {
        expect(getRunPercentComplete(1, 1000)).toBe(0);
    });

    it("should handle very large numbers", () => {
        expect(getRunPercentComplete(1000000, 2000000)).toBe(50);
    });

    it("should return 100 for equal non-zero values", () => {
        expect(getRunPercentComplete(5, 5)).toBe(100);
    });

    it("should handle decimal inputs", () => {
        expect(getRunPercentComplete(2.5, 5)).toBe(50);
    });
});

describe("RunIOManager", () => {
    const logger = console;

    describe("parseRunIO", () => {
        it("should handle both inputs and outputs", () => {
            const inputs = [
                { input: { id: "1", name: "input1" }, data: "\"input value\"" },
            ] as any[]; // Using 'any' to simulate ExistingInput type

            const outputs = [
                { output: { id: "2", name: "output2" }, data: "\"output value\"" },
            ] as any[]; // Using 'any' to simulate ExistingOutput type

            expect(new RunIOManager(logger).parseRunIO(inputs, "input")).toEqual({
                "input-input1": "input value",
            });

            expect(new RunIOManager(logger).parseRunIO(outputs, "output")).toEqual({
                "output-output2": "output value",
            });
        });

        it("should handle various data types", () => {
            const mixedIO = [
                { input: { id: "1", name: "string" }, data: "\"text\"" },
                { input: { id: "2", name: "number" }, data: "42" },
                { input: { id: "3", name: "boolean" }, data: "true" },
                { input: { id: "4", name: "object" }, data: "{\"key\": \"value\"}" },
                { input: { id: "5", name: "array" }, data: "[1,2,3]" },
            ] as any[];

            expect(new RunIOManager(logger).parseRunIO(mixedIO, "input")).toEqual({
                "input-string": "text",
                "input-number": 42,
                "input-boolean": true,
                "input-object": { key: "value" },
                "input-array": [1, 2, 3],
            });
        });

        it("should return an empty object for non-array input", () => {
            // @ts-ignore: Testing runtime scenario
            expect(new RunIOManager(logger).parseRunIO({}, "input")).toEqual({});
            // @ts-ignore: Testing runtime scenario
            expect(new RunIOManager(logger).parseRunIO(null, "output")).toEqual({});
        });

        it("should use value as-is if JSON parse fails", () => {
            const invalidData = [
                { input: { id: "1", name: "invalid" }, data: "invalid json" },
            ] as any[];

            expect(new RunIOManager(logger).parseRunIO(invalidData, "input")).toEqual({
                "input-invalid": "invalid json",
            });
        });
    });

    describe("parseRunInputs", () => {
        it("should return an empty object for null input", () => {
            // @ts-ignore: Testing runtime scenario
            expect(new RunIOManager(logger).parseRunInputs(null)).toEqual({});
        });

        it("should return an empty object for invalid input type", () => {
            const notInputs = { __typename: "SomethingElse" };
            // @ts-ignore: Testing runtime scenario
            expect(new RunIOManager(logger).parseRunInputs(notInputs)).toEqual({});
        });

        it("should parse inputs correctly", () => {
            const inputs = [
                { input: { id: "1", name: "input1" }, data: "\"string value\"" },
                { input: { id: "2", name: "input2" }, data: "42" },
                { input: { id: "3", name: "input3" }, data: "true" },
                { input: { id: "4", name: "input4" }, data: "{\"key\": \"value\"}" },
                { input: { id: "5", name: "input5" }, data: "[1,2,3]" },
            ] as RunRoutineInput[];

            expect(new RunIOManager(logger).parseRunInputs(inputs)).toEqual({
                "input-input1": "string value",
                "input-input2": 42,
                "input-input3": true,
                "input-input4": { key: "value" },
                "input-input5": [1, 2, 3],
            });
        });

        it("should use id as key when name is not available", () => {
            const inputs = [
                { input: { id: "1" }, data: "\"value\"" },
            ] as RunRoutineInput[];

            expect(new RunIOManager(logger).parseRunInputs(inputs)).toEqual({
                "input-1": "value",
            });
        });

        it("should handle parsing errors gracefully", () => {
            const inputs = [
                { input: { id: "1", name: "input1" }, data: "invalid json" },
            ] as RunRoutineInput[];

            expect(new RunIOManager(logger).parseRunInputs(inputs)).toEqual({
                "input-input1": "invalid json",
            });
        });
    });

    describe("parseRunOutputs", () => {
        it("should return an empty object for null input", () => {
            // @ts-ignore: Testing runtime scenario
            expect(new RunIOManager(logger).parseRunOutputs(null)).toEqual({});
        });

        it("should return an empty object for invalid input type", () => {
            const notOutputs = { __typename: "SomethingElse" };
            // @ts-ignore: Testing runtime scenario
            expect(new RunIOManager(logger).parseRunOutputs(notOutputs)).toEqual({});
        });

        it("should parse outputs correctly", () => {
            const outputs = [
                { output: { id: "1", name: "output1" }, data: "\"string value\"" },
                { output: { id: "2", name: "output2" }, data: "42" },
                { output: { id: "3", name: "output3" }, data: "true" },
                { output: { id: "4", name: "output4" }, data: "{\"key\": \"value\"}" },
                { output: { id: "5", name: "output5" }, data: "[1,2,3]" },
            ] as any[]; // Using 'any' to simulate RunRoutineOutput type

            expect(new RunIOManager(logger).parseRunOutputs(outputs)).toEqual({
                "output-output1": "string value",
                "output-output2": 42,
                "output-output3": true,
                "output-output4": { key: "value" },
                "output-output5": [1, 2, 3],
            });
        });

        it("should use id as key when name is not available", () => {
            const outputs = [
                { output: { id: "1" }, data: "\"value\"" },
            ] as any[]; // Using 'any' to simulate RunRoutineOutput type

            expect(new RunIOManager(logger).parseRunOutputs(outputs)).toEqual({
                "output-1": "value",
            });
        });

        it("should handle parsing errors gracefully", () => {
            const outputs = [
                { output: { id: "1", name: "output1" }, data: "invalid json" },
            ] as any[]; // Using 'any' to simulate RunRoutineOutput type

            expect(new RunIOManager(logger).parseRunOutputs(outputs)).toEqual({
                "output-output1": "invalid json",
            });
        });
    });

    describe("runInputsUpdate", () => {
        it("should create new inputs when they do not exist", () => {
            const params: Omit<RunIOUpdateParams<"input">, "ioType"> = {
                existingIO: [],
                formData: {
                    "output-routineInputName1": "valueBoop",
                    "input-routineInputName1": "value1",
                    "input-routineInputName2": 42,
                },
                routineIO: [{
                    id: "routineInput1",
                    name: "routineInputName1",
                }, {
                    id: "routineInput2",
                    name: "routineInputName2",
                }],
                runRoutineId: "run1",
            };

            const result = new RunIOManager(logger).runInputsUpdate(params);

            expect(result.inputsCreate).toEqual([
                {
                    id: expect.any(String),
                    data: "\"value1\"",
                    inputConnect: "routineInput1",
                    runRoutineConnect: "run1",
                },
                {
                    id: expect.any(String),
                    data: "42",
                    inputConnect: "routineInput2",
                    runRoutineConnect: "run1",
                },
            ]);
            result.inputsCreate?.forEach(input => expect(uuidValidate(input.id)).toBe(true));
            expect(result.inputsUpdate).toBeUndefined();
            expect(result.inputsDelete).toBeUndefined();
        });

        it("should update existing inputs when data has changed", () => {
            const params: Omit<RunIOUpdateParams<"input">, "ioType"> = {
                existingIO: [{
                    id: "runInput1",
                    data: "\"old-value1\"",
                    input: { name: "routineInputName1" } as RoutineVersionInput,
                }, {
                    id: "runInput2",
                    data: "999",
                    input: { id: "routineInput2" },
                }],
                formData: { "input-routineInputName1": "new-value1", "input-routineInputName2": 42 },
                routineIO: [{
                    id: "routineInput1",
                    name: "routineInputName1",
                }, {
                    id: "routineInput2",
                    name: "routineInputName2",
                }],
                runRoutineId: "run1",
            };

            const result = new RunIOManager(logger).runInputsUpdate(params);

            expect(result.inputsUpdate).toEqual([
                {
                    id: "runInput1",
                    data: "\"new-value1\"",
                },
                {
                    id: "runInput2",
                    data: "42",
                },
            ]);
            expect(result.inputsCreate).toBeUndefined();
            expect(result.inputsDelete).toBeUndefined();
        });

        it("should handle creating and updating inputs simultaneously", () => {
            const params: Omit<RunIOUpdateParams<"input">, "ioType"> = {
                existingIO: [{
                    id: "runInput2",
                    data: "999",
                    input: { id: "routineInput2", name: null },
                }, {
                    id: "runInput6",
                    data: "\"hello world\"",
                    input: { id: "routineInput6" },
                }],
                formData: { "input-routineInputName1": "value1", "input-routineInputName2": 42 },
                routineIO: [{
                    id: "routineInput1",
                    name: "routineInputName1",
                }, {
                    id: "routineInput2",
                    name: "routineInputName2",
                }, {
                    id: "routineInput3",
                    name: "routineInputName3",
                }],
                runRoutineId: "run1",
            };

            const result = new RunIOManager(logger).runInputsUpdate(params);

            expect(result.inputsCreate).toEqual([
                {
                    id: expect.any(String),
                    data: "\"value1\"",
                    inputConnect: "routineInput1",
                    runRoutineConnect: "run1",
                },
            ]);
            result.inputsCreate?.forEach(input => expect(uuidValidate(input.id)).toBe(true));
            expect(result.inputsUpdate).toEqual([
                {
                    id: "runInput2",
                    data: "42",
                },
            ]);
            expect(result.inputsDelete).toBeUndefined();
        });
    });

    describe("runOutputsUpdate", () => {
        it("should create new outputs when they do not exist", () => {
            const params: Omit<RunIOUpdateParams<"output">, "ioType"> = {
                existingIO: [],
                formData: { "output-routineOutputName1": "value1", "output-routineOutputName2": 42 },
                routineIO: [{
                    id: "routineOutput1",
                    name: "routineOutputName1",
                }, {
                    id: "routineOutput2",
                    name: "routineOutputName2",
                }],
                runRoutineId: "run1",
            };

            const result = new RunIOManager(logger).runOutputsUpdate(params);

            expect(result.outputsCreate).toEqual([
                {
                    id: expect.any(String),
                    data: "\"value1\"",
                    outputConnect: "routineOutput1",
                    runRoutineConnect: "run1",
                },
                {
                    id: expect.any(String),
                    data: "42",
                    outputConnect: "routineOutput2",
                    runRoutineConnect: "run1",
                },
            ]);
            result.outputsCreate?.forEach(output => expect(uuidValidate(output.id)).toBe(true));
            expect(result.outputsUpdate).toBeUndefined();
            expect(result.outputsDelete).toBeUndefined();
        });

        it("should update existing outputs when data has changed", () => {
            const params: Omit<RunIOUpdateParams<"output">, "ioType"> = {
                existingIO: [{
                    id: "runOutput1",
                    data: "\"old-value1\"",
                    output: { name: "routineOutputName1" } as RoutineVersionOutput,
                }, {
                    id: "runOutput2",
                    data: "999",
                    output: { id: "routineOutput2" },
                }],
                formData: { "output-routineOutputName1": "new-value1", "output-routineOutputName2": 42 },
                routineIO: [{
                    id: "routineOutput1",
                    name: "routineOutputName1",
                }, {
                    id: "routineOutput2",
                    name: "routineOutputName2",
                }],
                runRoutineId: "run1",
            };

            const result = new RunIOManager(logger).runOutputsUpdate(params);

            expect(result.outputsUpdate).toEqual([
                {
                    id: "runOutput1",
                    data: "\"new-value1\"",
                },
                {
                    id: "runOutput2",
                    data: "42",
                },
            ]);
            expect(result.outputsCreate).toBeUndefined();
            expect(result.outputsDelete).toBeUndefined();
        });

        it("should handle creating and updating outputs simultaneously", () => {
            const params: Omit<RunIOUpdateParams<"output">, "ioType"> = {
                existingIO: [{
                    id: "runOutput2",
                    data: "999",
                    output: { id: "routineOutput2", name: null },
                }, {
                    id: "runOutput6",
                    data: "\"hello world\"",
                    output: { id: "routineOutput6" },
                }],
                formData: { "output-routineOutputName1": "value1", "output-routineOutputName2": 42 },
                routineIO: [{
                    id: "routineOutput1",
                    name: "routineOutputName1",
                }, {
                    id: "routineOutput2",
                    name: "routineOutputName2",
                }, {
                    id: "routineOutput3",
                    name: "routineOutputName3",
                }],
                runRoutineId: "run1",
            };

            const result = new RunIOManager(logger).runOutputsUpdate(params);

            expect(result.outputsCreate).toEqual([
                {
                    id: expect.any(String),
                    data: "\"value1\"",
                    outputConnect: "routineOutput1",
                    runRoutineConnect: "run1",
                },
            ]);
            result.outputsCreate?.forEach(output => expect(uuidValidate(output.id)).toBe(true));
            expect(result.outputsUpdate).toEqual([
                {
                    id: "runOutput2",
                    data: "42",
                },
            ]);
            expect(result.outputsDelete).toBeUndefined();
        });
    });

    describe("getIOKey", () => {
        it("should return a key based on name for input", () => {
            const result = new RunIOManager(logger).getIOKey({ name: "test input" }, "input");
            expect(result).toBe("input-test_input");
        });

        it("should return a key based on name for output", () => {
            const result = new RunIOManager(logger).getIOKey({ name: "test output" }, "output");
            expect(result).toBe("output-test_output");
        });

        it("should use id if name is not provided for input", () => {
            const result = new RunIOManager(logger).getIOKey({ id: "test-id" }, "input");
            expect(result).toBe("input-test-id");
        });

        it("should use id if name is not provided for output", () => {
            const result = new RunIOManager(logger).getIOKey({ id: "test-id" }, "output");
            expect(result).toBe("output-test-id");
        });

        it("should sanitize special characters in name", () => {
            const result = new RunIOManager(logger).getIOKey({ name: "test@input#123\n\t" }, "input");
            expect(result).toBe("input-test_input_123__");
        });

        it("should sanitize special characters in id", () => {
            const result = new RunIOManager(logger).getIOKey({ id: "test@id#123" }, "input");
            expect(result).toBe("input-test_id_123");
        });

        it("should return null and if both name and id are missing", () => {
            const result = new RunIOManager(logger).getIOKey({}, "input");
            expect(result).toBeNull();
        });

        it("should return null if both name and id are null", () => {
            const result = new RunIOManager(logger).getIOKey({ name: null, id: null }, "output");
            expect(result).toBeNull();
        });

        it("shouldn't use empty string name", () => {
            const result = new RunIOManager(logger).getIOKey({ name: "", id: "123" }, "input");
            expect(result).toBe("input-123");
        });

        it("should return null for no name and empty string id", () => {
            const result = new RunIOManager(logger).getIOKey({ id: "" }, "output");
            expect(result).toBeNull();
        });

        it("should prefer name over id when both are provided", () => {
            const result = new RunIOManager(logger).getIOKey({ name: "test-name", id: "test-id" }, "input");
            expect(result).toBe("input-test-name");
        });

        it("should handle numeric name", () => {
            const result = new RunIOManager(logger).getIOKey({ name: "123" }, "input");
            expect(result).toBe("input-123");
        });

        it("should handle numeric id", () => {
            const result = new RunIOManager(logger).getIOKey({ id: "456" }, "output");
            expect(result).toBe("output-456");
        });
    });
});

describe("generateRoutineInitialValues", () => {
    const inputSchema = {
        containers: [],
        elements: [{
            type: FormStructureType.Header,
            id: "header1",
            tag: "h1",
        }, {
            type: InputType.Text,
            fieldName: "inputField1", // `fieldName` should match routine input's name
            id: "inputId1", // `id` should match routine input's id
            label: "Input #1",
            props: {}, // No default value for this one
        }, {
            type: InputType.IntegerInput,
            fieldName: "inputField2",
            id: "inputId2",
            label: "Input #2",
            props: {
                defaultValue: -23.0,
            },
        }, {
            type: InputType.Checkbox,
            fieldName: "inputField3",
            id: "inputId3",
            label: "Input #3",
            props: {
                defaultValue: [false, true],
                options: [{
                    label: "Option 1",
                    value: "option1",
                }, {
                    label: "Option 2",
                    value: "option2",
                }],
            },
        },
        // Not included in run data, and doesn't have default value, so "" should be used in result
        {
            type: InputType.IntegerInput,
            fieldName: "inputField4",
            id: "inputId4",
            label: "Input #4",
        }],
    } as FormSchema;

    const outputSchema = {
        containers: [],
        elements: [{
            type: FormStructureType.Header,
            id: "header9",
            tag: "h1",
        }, {
            type: InputType.Text,
            fieldName: "outputField1",
            id: "outputId1",
            label: "Output #1",
            props: {
                defaultValue: "default output",
            },
        }, {
            type: InputType.Text,
            fieldName: "outputField2",
            id: "outputId2",
            label: "Output #2",
            props: {
                defaultValue: "\n\t🐇🐇🐇\n\t",
            },
        },
        // Not included in run data, so default value should be used in result
        {
            type: InputType.IntegerInput,
            fieldName: "outputField3",
            id: "outputId3",
            label: "Output #3",
            props: {
                defaultValue: 23,
                max: 912,
                min: 2,
            },
        }],
    } as FormSchema;

    const runInputs = [
        { input: { id: "inputId1", name: "inputField1" }, data: "\"string value\"" },
        { input: { id: "inputId2", name: "inputField2" }, data: "-1234.20" },
        { input: { id: "inputId3", name: "inputField3" }, data: "\"hello world\"" },
        { input: { id: "inputId9999", name: "inputField9999" }, data: "{\"key\": \"value\"}" }, // Doesn't match any inputs, so should be ignored
    ] as ExistingInput[];

    const runOutputs = [
        { output: { id: "outputId999", name: "outputField9999" }, data: "\"boopies\"" }, // Doesn't match any outputs, so should be ignored
        { output: { id: "outputId123", name: "inputField1" }, data: "999999" }, // Name matches input instead of output, so should be ignored
        { output: { id: "outputId1", name: "outputField1" }, data: "\"fdksjafl;sa\\n\\nfjdkls;fj;ldsa\"" },
        { output: { id: "outputId2", name: "outputField2" }, data: "{\"key2\": \"value2\"}" },
    ] as ExistingOutput[];

    it("should work when no run data is provided", () => {
        const result = generateRoutineInitialValues({
            configFormInput: inputSchema,
            configFormOutput: outputSchema,
            logger: console,
            runInputs: null,
            runOutputs: undefined,
        });

        expect(result).toEqual({
            "input-inputField1": "",
            "input-inputField2": -23.0,
            "input-inputField3": [false, true],
            "input-inputField4": 0, // This is the default value for integer inputs if not provided
            "output-outputField1": "default output",
            "output-outputField2": "\n\t🐇🐇🐇\n\t",
            "output-outputField3": 23,
        });
    });

    it("should work when run data is provided", () => {
        const result = generateRoutineInitialValues({
            configFormInput: inputSchema,
            configFormOutput: outputSchema,
            logger: console,
            runInputs,
            runOutputs,
        });

        expect(result).toEqual({
            "input-inputField1": "string value", // Overridden by run data
            "input-inputField2": -1234.20, // Overridden by run data
            "input-inputField3": "hello world", // Overridden by run data
            "input-inputField4": 0, // Default value for integer inputs if not provided
            "output-outputField1": "fdksjafl;sa\n\nfjdkls;fj;ldsa", // Overridden by run data
            "output-outputField2": { key2: "value2" }, // Overridden by run data
            "output-outputField3": 23,
        });
    });

    // describe("real world example", () => {
    //     it("example 1", () => {
    //         const configFormInput = "{\"containers\":[{\"totalItems\":1},{\"totalItems\":1}],\"elements\":[{\"type\":\"Text\",\"props\":{\"autoComplete\":\"off\",\"defaultValue\":\"\",\"isMarkdown\":true,\"maxChars\":1000,\"maxRows\":2,\"minRows\":4},\"fieldName\":\"input-xme5mnsjdz\",\"id\":\"4c85518b-ec56-48ee-b231-caadabe08234\",\"label\":\"What do you want to know?\",\"yup\":{\"checks\":[]},\"isRequired\":true},{\"type\":\"LinkUrl\",\"props\":{\"acceptedHosts\":[],\"defaultValue\":\"http://google.com\"},\"fieldName\":\"field-gi7s9c0b3m\",\"id\":\"26af2256-c093-4cae-ad62-3c6171ee2798\",\"label\":\"Input #2\",\"yup\":{\"checks\":[]}}]}";
    //         const configFormOutput = "{\"containers\":[],\"elements\":[{\"fieldName\":\"response\",\"id\":\"response\",\"label\":\"Response\",\"props\":{\"placeholder\":\"Model response will be displayed here\"},\"type\":\"Text\"}]}";
    //         const runInputs = [{
    //             __typename: "RunRoutineInput",
    //             data: "\"requiredValue\"",
    //             id: "5963a94c-e4c9-4f41-8cd0-6f5131bd88de",
    //             input: {
    //                 __typename: "RoutineVersionInput",
    //                 id: "4c85518b-ec56-48ee-b231-caadabe08234",
    //                 index: 0,
    //                 isRequired: true,
    //                 name: "input-xme5mnsjdz",
    //                 standardVersion: null,
    //                 translations: [],
    //             },
    //         }, {
    //             __typename: "RunRoutineInput",
    //             data: "\"http://google.com\"",
    //             id: "f48dcc31-d070-4f83-acfb-2a2d0286a16c",
    //             input: {
    //                 __typename: "RoutineVersionInput",
    //                 id: "4c85518b-ec56-48ee-b231-caadabe08234",
    //                 index: 0,
    //                 isRequired: false,
    //                 name: "field-gi7s9c0b3m",
    //                 standardVersion: null,
    //                 translations: [],
    //             },
    //         }];
    //         const runOutputs = [];
    //     });
    // });
});

describe("routineVersionHasSubroutines", () => {
    it("should return false for null input", () => {
        expect(routineVersionHasSubroutines(null as any)).toBe(false);
    });

    it("should return false for undefined input", () => {
        expect(routineVersionHasSubroutines(undefined as any)).toBe(false);
    });

    it("should return false for empty object", () => {
        expect(routineVersionHasSubroutines({})).toBe(false);
    });

    it("should return false for non-MultiStep routine type", () => {
        const routineVersion: Partial<RoutineVersion> = { routineType: RoutineType.Generate };
        expect(routineVersionHasSubroutines(routineVersion)).toBe(false);
    });

    it("should return true for MultiStep routine with nodes", () => {
        const routineVersion: Partial<RoutineVersion> = {
            routineType: RoutineType.MultiStep,
            nodes: [{ __typename: "Node" } as any],
        };
        expect(routineVersionHasSubroutines(routineVersion)).toBe(true);
    });

    it("should return true for MultiStep routine with nodeLinks", () => {
        const routineVersion: Partial<RoutineVersion> = {
            routineType: RoutineType.MultiStep,
            nodeLinks: [{ __typename: "NodeLink" } as any],
        };
        expect(routineVersionHasSubroutines(routineVersion)).toBe(true);
    });

    it("should return true for MultiStep routine with nodesCount > 0", () => {
        const routineVersion: Partial<RoutineVersion> = {
            routineType: RoutineType.MultiStep,
            nodesCount: 1,
        };
        expect(routineVersionHasSubroutines(routineVersion)).toBe(true);
    });

    it("should return false for MultiStep routine with empty nodes, nodeLinks, and nodesCount = 0", () => {
        const routineVersion: Partial<RoutineVersion> = {
            routineType: RoutineType.MultiStep,
            nodes: [],
            nodeLinks: [],
            nodesCount: 0,
        };
        expect(routineVersionHasSubroutines(routineVersion)).toBe(false);
    });

    it("should return false for MultiStep routine with undefined nodes, nodeLinks, and nodesCount", () => {
        const routineVersion: Partial<RoutineVersion> = {
            routineType: RoutineType.MultiStep,
        };
        expect(routineVersionHasSubroutines(routineVersion)).toBe(false);
    });

    it("should return true if any of nodes, nodeLinks, or nodesCount indicate subroutines", () => {
        const routineVersion: Partial<RoutineVersion> = {
            routineType: RoutineType.MultiStep,
            nodes: [{ __typename: "Node" } as any],
            nodeLinks: [],
            nodesCount: 1,
        };
        expect(routineVersionHasSubroutines(routineVersion)).toBe(true);
    });
});

describe("RunStepNavigator", () => {
    const logger = console;

    describe("stepFromLocation", () => {
        beforeAll(() => {
            // eslint-disable-next-line @typescript-eslint/no-empty-function
            jest.spyOn(console, "warn").mockImplementation(() => { });
        });
        afterAll(() => {
            jest.restoreAllMocks();
        });

        it("should navigate through a Directory step correctly", () => {
            const rootStep: DirectoryStep = {
                __type: RunStepType.Directory,
                description: "boop",
                directoryId: "directory123",
                hasBeenQueried: true,
                isOrdered: false,
                isRoot: true,
                location: [1],
                name: "hello",
                projectVersionId: "projectVersion123",
                steps: [
                    {
                        __type: RunStepType.SingleRoutine,
                        description: "a",
                        location: [1, 1],
                        name: "Step 1",
                        routineVersion: { id: "routineVersion123" } as RoutineVersion,
                    },
                    {
                        __type: RunStepType.SingleRoutine,
                        description: "b",
                        location: [1, 3], // Shouldn't matter that the location is incorrect, as it should rely on the structure of the steps rather than the defined locations
                        name: "Step 2",
                        routineVersion: { id: "routineVersion234" } as RoutineVersion,
                    },
                ],
            };
            const result1 = new RunStepNavigator(logger).stepFromLocation([1], rootStep);
            expect(result1).toEqual(rootStep);
            const result2 = new RunStepNavigator(logger).stepFromLocation([1, 2], rootStep);
            expect(result2).toEqual(rootStep.steps[1]);
            const result3 = new RunStepNavigator(logger).stepFromLocation([1, 3], rootStep);
            expect(result3).toBeNull();
            const result4 = new RunStepNavigator(logger).stepFromLocation([2], rootStep);
            expect(result4).toBeNull();
            const result5 = new RunStepNavigator(logger).stepFromLocation([2, 1], rootStep);
            expect(result5).toBeNull();
        });

        it("should navigate through a multi-step routine correctly", () => {
            const rootStep: MultiRoutineStep = {
                __type: RunStepType.MultiRoutine,
                description: "root",
                location: [1],
                name: "root",
                nodes: [
                    {
                        __type: RunStepType.Start,
                        description: "start",
                        location: [1, 1],
                        name: "start",
                        nextLocation: [],
                        nodeId: "startNode",
                    },
                    {
                        __type: RunStepType.RoutineList,
                        description: "routine list",
                        isOrdered: false,
                        location: [1, 2],
                        name: "routine list",
                        nextLocation: [9, 1, 1],
                        nodeId: "routineListNode1",
                        parentRoutineVersionId: "routine123",
                        steps: [
                            {
                                __type: RunStepType.SingleRoutine,
                                description: "subroutine",
                                location: [1, 2, 1],
                                name: "subroutine",
                                routineVersion: { id: "routineVersion123" } as RoutineVersion,
                            },
                        ],
                    },
                    {
                        __type: RunStepType.RoutineList,
                        description: "routine list",
                        isOrdered: true,
                        location: [1, 3],
                        name: "routine list",
                        nextLocation: [],
                        nodeId: "routineListNode2",
                        parentRoutineVersionId: "routine123",
                        steps: [
                            {
                                __type: RunStepType.SingleRoutine,
                                description: "subroutine",
                                location: [1, 3, 1],
                                name: "subroutine",
                                routineVersion: { id: "routineVersion234" } as RoutineVersion,
                            },
                            {
                                __type: RunStepType.MultiRoutine,
                                description: "multi routine",
                                location: [1, 3, 2],
                                name: "multi routine",
                                nodes: [
                                    {
                                        __type: RunStepType.RoutineList,
                                        description: "routine list",
                                        isOrdered: true,
                                        location: [1, 3, 2, 1],
                                        name: "routine list",
                                        nextLocation: [1],
                                        nodeId: "routineListNode3",
                                        parentRoutineVersionId: "routine234",
                                        steps: [
                                            {
                                                __type: RunStepType.SingleRoutine,
                                                description: "subroutine",
                                                location: [1, 3, 2, 1, 1],
                                                name: "subroutine",
                                                routineVersion: { id: "routineVersion345" } as RoutineVersion,
                                            },
                                        ],
                                    },
                                ],
                                nodeLinks: [],
                                routineVersionId: "multiRoutine123",
                                startNodeIndexes: [],
                            },
                        ],
                    },
                    {
                        __type: RunStepType.End,
                        description: "end",
                        location: [1, 3],
                        name: "end",
                        nextLocation: null,
                        nodeId: "endNode",
                        wasSuccessful: true,
                    },
                ],
                nodeLinks: [],
                routineVersionId: "rootRoutine",
                startNodeIndexes: [0],
            };
            const result1 = new RunStepNavigator(logger).stepFromLocation([1], rootStep);
            expect(result1).toEqual(rootStep);
            const result2 = new RunStepNavigator(logger).stepFromLocation([1, 1], rootStep);
            expect(result2).toEqual(rootStep.nodes[0]);
            const result3 = new RunStepNavigator(logger).stepFromLocation([1, 2], rootStep);
            expect(result3).toEqual(rootStep.nodes[1]);
            const result4 = new RunStepNavigator(logger).stepFromLocation([1, 3, 2], rootStep);
            expect(result4).toEqual((rootStep.nodes[2] as RoutineListStep).steps[1]);
            const result5 = new RunStepNavigator(logger).stepFromLocation([1, 3, 2, 1], rootStep);
            expect(result5).toEqual(((rootStep.nodes[2] as RoutineListStep).steps[1] as MultiRoutineStep).nodes[0]);
            const result6 = new RunStepNavigator(logger).stepFromLocation([1, 3, 2, 1, 1], rootStep);
            expect(result6).toEqual((((rootStep.nodes[2] as RoutineListStep).steps[1] as MultiRoutineStep).nodes[0] as RoutineListStep).steps[0]);
        });

        // Should return null if the location is invalid
        it("should return null for an invalid location", () => {
            const rootStep: DirectoryStep = {
                __type: RunStepType.Directory,
                description: "root",
                directoryId: "directory123",
                hasBeenQueried: false,
                isOrdered: false,
                isRoot: true,
                location: [1],
                name: "root",
                projectVersionId: "projectVersion123",
                steps: [],
            };
            const result1 = new RunStepNavigator(logger).stepFromLocation([2, 1], rootStep);
            expect(result1).toBeNull();
            const result2 = new RunStepNavigator(logger).stepFromLocation([], rootStep);
            expect(result2).toBeNull();
        });
    });

    describe("getNextLocation and getPreviousLocation", () => {
        beforeAll(() => {
            // eslint-disable-next-line @typescript-eslint/no-empty-function
            jest.spyOn(console, "warn").mockImplementation(() => { });
        });
        afterAll(() => {
            jest.restoreAllMocks();
        });

        it("returns null if the root step is invalid", () => {
            expect(new RunStepNavigator(logger).getNextLocation([1], null)).toBeNull();
        });

        it("works with directories", () => {
            const rootStep: DirectoryStep = {
                __type: RunStepType.Directory,
                description: "root",
                directoryId: "directory123",
                hasBeenQueried: false,
                isOrdered: false,
                isRoot: true,
                location: [1],
                name: "root",
                projectVersionId: "projectVersion123",
                steps: [
                    {
                        __type: RunStepType.SingleRoutine,
                        description: "a",
                        location: [1, 1],
                        name: "Step 1",
                        routineVersion: { id: "routineVersion123" } as RoutineVersion,
                    },
                    {
                        __type: RunStepType.SingleRoutine,
                        description: "b",
                        location: [1, 2],
                        name: "Step 2",
                        routineVersion: { id: "routineVersion234" } as RoutineVersion,
                    },
                ],
            };
            expect(new RunStepNavigator(logger).getNextLocation([1], rootStep)).toEqual([1, 1]);
            expect(new RunStepNavigator(logger).getPreviousLocation([1, 1], rootStep)).toEqual([1]);

            expect(new RunStepNavigator(logger).getNextLocation([1, 1], rootStep)).toEqual([1, 2]);
            expect(new RunStepNavigator(logger).getPreviousLocation([1, 2], rootStep)).toEqual([1, 1]);

            expect(new RunStepNavigator(logger).getNextLocation([1, 2], rootStep)).toBeNull();

            expect(new RunStepNavigator(logger).getNextLocation([2], rootStep)).toBeNull();

            expect(new RunStepNavigator(logger).getNextLocation([], rootStep)).toEqual([1]);
            expect(new RunStepNavigator(logger).getPreviousLocation([1], rootStep)).toBeNull();
        });

        it("works with multi-step routines", () => {
            const rootStep: MultiRoutineStep = {
                __type: RunStepType.MultiRoutine,
                description: "root",
                location: [1],
                name: "root",
                nodes: [
                    {
                        __type: RunStepType.Start,
                        description: "start",
                        location: [1, 1],
                        name: "start",
                        nextLocation: [1, 4],
                        nodeId: "startNode",
                    },
                    {
                        __type: RunStepType.RoutineList,
                        description: "routine list",
                        isOrdered: false,
                        location: [1, 2],
                        name: "routine list",
                        nextLocation: [1, 5],
                        nodeId: "routineListNode1",
                        parentRoutineVersionId: "routine123",
                        steps: [
                            {
                                __type: RunStepType.SingleRoutine,
                                description: "subroutine",
                                location: [1, 2, 1],
                                name: "subroutine",
                                routineVersion: { id: "routineVersion123" } as RoutineVersion,
                            },
                        ],
                    },
                    {
                        __type: RunStepType.End,
                        description: "end",
                        location: [1, 4],
                        name: "end",
                        nextLocation: null,
                        nodeId: "endNode",
                        wasSuccessful: false,
                    },
                    {
                        __type: RunStepType.RoutineList,
                        description: "routine list",
                        isOrdered: false,
                        location: [1, 3],
                        name: "routine list",
                        nextLocation: [1, 5],
                        nodeId: "routineListNode2",
                        parentRoutineVersionId: "routine123",
                        steps: [
                            {
                                __type: RunStepType.SingleRoutine,
                                description: "subroutine",
                                location: [1, 3, 1],
                                name: "subroutine",
                                routineVersion: { id: "routineVersion234" } as RoutineVersion,
                            },
                            {
                                __type: RunStepType.MultiRoutine,
                                description: "multi routine",
                                location: [1, 3, 2],
                                name: "multi routine",
                                nodes: [
                                    {
                                        __type: RunStepType.RoutineList,
                                        description: "routine list",
                                        isOrdered: true,
                                        location: [1, 3, 2, 1],
                                        name: "routine list",
                                        nextLocation: null,
                                        nodeId: "routineListNode3",
                                        parentRoutineVersionId: "routine234",
                                        steps: [
                                            {
                                                __type: RunStepType.SingleRoutine,
                                                description: "subroutine",
                                                location: [1, 3, 2, 1, 1],
                                                name: "subroutine",
                                                routineVersion: { id: "routineVersion345" } as RoutineVersion,
                                            },
                                        ],
                                    },
                                ],
                                nodeLinks: [],
                                routineVersionId: "multiRoutine123",
                                startNodeIndexes: [],
                            },
                        ],
                    },
                    {
                        __type: RunStepType.RoutineList,
                        description: "routine list",
                        isOrdered: false,
                        location: [1, 5],
                        name: "routine list",
                        nextLocation: [1, 2],
                        nodeId: "routineListNode1",
                        steps: [],
                        parentRoutineVersionId: "routine123",
                    },
                ],
                nodeLinks: [],
                routineVersionId: "rootRoutine",
                startNodeIndexes: [0],
            };
            // Root step to first node
            expect(new RunStepNavigator(logger).getNextLocation([1], rootStep)).toEqual([1, 1]);
            expect(new RunStepNavigator(logger).getPreviousLocation([1, 1], rootStep)).toEqual([1]);

            // First node to its `nextLocation` field
            expect(new RunStepNavigator(logger).getPreviousLocation([1, 4], rootStep)).toEqual([1, 1]);

            // Second node prefers its child over `nextLocation`
            expect(new RunStepNavigator(logger).getNextLocation([1, 2], rootStep)).toEqual([1, 2, 1]);
            expect(new RunStepNavigator(logger).getPreviousLocation([1, 2, 1], rootStep)).toEqual([1, 2]);

            // Second node's child has no more siblings, to goes to parent's `nextLocation`
            expect(new RunStepNavigator(logger).getNextLocation([1, 2, 1], rootStep)).toEqual([1, 5]);
            expect(new RunStepNavigator(logger).getPreviousLocation([1, 5], rootStep)).toEqual([1, 2]); // Doesn't go down children

            // Go to first child
            expect(new RunStepNavigator(logger).getNextLocation([1, 3], rootStep)).toEqual([1, 3, 1]);
            expect(new RunStepNavigator(logger).getPreviousLocation([1, 3, 1], rootStep)).toEqual([1, 3]);

            // Go to second child
            expect(new RunStepNavigator(logger).getNextLocation([1, 3, 1], rootStep)).toEqual([1, 3, 2]);
            expect(new RunStepNavigator(logger).getPreviousLocation([1, 3, 2], rootStep)).toEqual([1, 3, 1]);

            // Go to first child
            expect(new RunStepNavigator(logger).getNextLocation([1, 3, 2], rootStep)).toEqual([1, 3, 2, 1]);
            expect(new RunStepNavigator(logger).getPreviousLocation([1, 3, 2, 1], rootStep)).toEqual([1, 3, 2]);

            // Go to first child
            expect(new RunStepNavigator(logger).getNextLocation([1, 3, 2, 1], rootStep)).toEqual([1, 3, 2, 1, 1]);
            expect(new RunStepNavigator(logger).getPreviousLocation([1, 3, 2, 1, 1], rootStep)).toEqual([1, 3, 2, 1]);

            // Backtrack multiple times to [1, 3]'s `nextLocation`
            expect(new RunStepNavigator(logger).getNextLocation([1, 3, 2, 1, 1], rootStep)).toEqual([1, 5]);
            expect(new RunStepNavigator(logger).getNextLocation([1, 1], rootStep)).toEqual([1, 4]);
            // There are actually 2 locations that can lead to [1, 5], so either is valid
            expect([[1, 2], [1, 3]]).toContainEqual(new RunStepNavigator(logger).getPreviousLocation([1, 5], rootStep));
        });

        it("another multi-step routine test", () => {
            const rootStep: MultiRoutineStep = {
                __type: RunStepType.MultiRoutine,
                description: "root",
                location: [1],
                name: "root",
                nodes: [
                    {
                        __type: RunStepType.Start,
                        description: "start",
                        location: [1, 1],
                        name: "start",
                        nextLocation: [1, 2],
                        nodeId: "startNode",
                    },
                    {
                        __type: RunStepType.Decision,
                        description: "decision",
                        location: [1, 2],
                        name: "Decision Step",
                        options: [
                            {
                                link: {} as NodeLink,
                                step: {
                                    __type: RunStepType.RoutineList,
                                    description: "routine list 1",
                                    isOrdered: false,
                                    location: [1, 3, 2, 1],
                                    name: "routine list a",
                                    nodeId: "routineListNode3",
                                    nextLocation: [1, 4],
                                    parentRoutineVersionId: "routine123",
                                    steps: [
                                        {
                                            __type: RunStepType.SingleRoutine,
                                            description: "subroutine 1",
                                            location: [1, 3, 2, 1, 1],
                                            name: "subroutine a",
                                            routineVersion: { id: "routineVersion345" } as RoutineVersion,
                                        },
                                    ],
                                } as RoutineListStep,
                            },
                            {
                                link: {} as NodeLink,
                                step: {
                                    __type: RunStepType.RoutineList,
                                    description: "routine list 2",
                                    isOrdered: true,
                                    location: [1, 4, 2, 1, 2, 2, 2],
                                    name: "routine list b",
                                    nextLocation: [1, 4, 2, 1, 2, 2, 3],
                                    nodeId: "routineListNode3",
                                    parentRoutineVersionId: "routine123",
                                    steps: [
                                        {
                                            __type: RunStepType.SingleRoutine,
                                            description: "subroutine 2",
                                            location: [1, 3, 2, 1, 1],
                                            name: "subroutine b",
                                            routineVersion: { id: "routineVersion345" } as RoutineVersion,
                                        },
                                    ],
                                } as RoutineListStep,
                            },
                        ],
                    },
                    {
                        __type: RunStepType.RoutineList,
                        description: "routine list",
                        isOrdered: false,
                        location: [1, 3],
                        name: "routine list",
                        nextLocation: [1, 4],
                        nodeId: "routineListNode2",
                        parentRoutineVersionId: "routine123",
                        steps: [
                            {
                                __type: RunStepType.SingleRoutine,
                                description: "subroutine",
                                location: [1, 3, 1],
                                name: "subroutine",
                                routineVersion: { id: "routineVersion234" } as RoutineVersion,
                            },
                            {
                                __type: RunStepType.MultiRoutine,
                                description: "multi routine",
                                location: [1, 3, 2],
                                name: "multi routine",
                                nodes: [
                                    {
                                        __type: RunStepType.RoutineList,
                                        description: "routine list",
                                        isOrdered: true,
                                        location: [1, 3, 2, 1],
                                        name: "routine list",
                                        nextLocation: null as unknown as number[], // Done on purpose,
                                        nodeId: "routineListNode3",
                                        parentRoutineVersionId: "routine234",
                                        steps: [
                                            {
                                                __type: RunStepType.SingleRoutine,
                                                description: "subroutine",
                                                location: [1, 3, 2, 1, 1],
                                                name: "subroutine",
                                                routineVersion: { id: "routineVersion345" } as RoutineVersion,
                                            },
                                        ],
                                    },
                                ],
                                nodeLinks: [],
                                routineVersionId: "multiRoutine123",
                                startNodeIndexes: [],
                            },
                        ],
                    },
                    {
                        __type: RunStepType.End,
                        description: "end",
                        location: [1, 4],
                        name: "end",
                        nextLocation: null,
                        nodeId: "endNode",
                        wasSuccessful: true,
                    },
                ],
                nodeLinks: [],
                routineVersionId: "rootRoutine",
                startNodeIndexes: [0],
            };
            // To `nextLocation`
            expect(new RunStepNavigator(logger).getNextLocation([1, 1], rootStep)).toEqual([1, 2]);
            expect(new RunStepNavigator(logger).getPreviousLocation([1, 2], rootStep)).toEqual([1, 1]);

            // Decisions should return null
            expect(new RunStepNavigator(logger).getNextLocation([1, 2], rootStep)).toBeNull();
        });
    });

    describe("siblingsAtLocation", () => {
        // Mock console.error to prevent actual console output during tests
        const originalConsoleError = console.error;
        beforeEach(() => {
            console.error = jest.fn();
        });
        afterEach(() => {
            console.error = originalConsoleError;
        });

        it("should return 0 for an empty location array", () => {
            const rootStep: DirectoryStep = {
                __type: RunStepType.Directory,
                description: "a",
                directoryId: null,
                hasBeenQueried: true,
                isOrdered: false,
                isRoot: true,
                location: [1, 2, 3],
                name: "a",
                projectVersionId: "projectVersion123",
                steps: [],
            };
            expect(new RunStepNavigator(logger).siblingsAtLocation([], rootStep)).toBe(0);
        });

        it("should return 1 for a location array with only one element", () => {
            const rootStep: DirectoryStep = {
                __type: RunStepType.Directory,
                description: "a",
                directoryId: null,
                hasBeenQueried: true,
                isOrdered: false,
                isRoot: true,
                location: [1, 2, 3],
                name: "a",
                projectVersionId: "projectVersion123",
                steps: [],
            };
            expect(new RunStepNavigator(logger).siblingsAtLocation([1], rootStep)).toBe(1);
        });

        it("should return the correct number of siblings for a DirectoryStep", () => {
            const rootStep: DirectoryStep = {
                __type: RunStepType.Directory,
                steps: [
                    { __type: RunStepType.SingleRoutine } as SingleRoutineStep,
                    { __type: RunStepType.SingleRoutine } as SingleRoutineStep,
                    { __type: RunStepType.SingleRoutine } as SingleRoutineStep,
                ],
            } as DirectoryStep;

            expect(new RunStepNavigator(logger).siblingsAtLocation([1, 2], rootStep)).toBe(3);
        });

        it("should return the correct number of siblings for a MultiRoutineStep", () => {
            const rootStep: MultiRoutineStep = {
                __type: RunStepType.MultiRoutine,
                nodes: [
                    { __type: RunStepType.Start },
                    { __type: RunStepType.RoutineList },
                    { __type: RunStepType.End },
                ],
            } as MultiRoutineStep;

            expect(new RunStepNavigator(logger).siblingsAtLocation([1, 2], rootStep)).toBe(3);
        });

        it("should return the correct number of siblings for a RoutineListStep", () => {
            const rootStep: RoutineListStep = {
                __type: RunStepType.RoutineList,
                steps: [
                    { __type: RunStepType.SingleRoutine } as SingleRoutineStep,
                    { __type: RunStepType.SingleRoutine } as SingleRoutineStep,
                ],
            } as RoutineListStep;

            expect(new RunStepNavigator(logger).siblingsAtLocation([1, 2], rootStep)).toBe(2);
        });

        it("should return 1 for an unknown step type", () => {
            const rootStep: RootStep = {
                __type: "UnknownType" as RunStepType,
            } as unknown as RootStep;

            expect(new RunStepNavigator(logger).siblingsAtLocation([1, 2], rootStep)).toBe(1);
        });

        it("should handle deeply nested locations", () => {
            const rootStep: DirectoryStep = {
                __type: RunStepType.Directory,
                location: [1],
                steps: [
                    {
                        __type: RunStepType.MultiRoutine,
                        location: [1, 1],
                        nodes: [
                            {
                                __type: RunStepType.Start,
                                location: [1, 1, 1],
                            },
                            {
                                __type: RunStepType.RoutineList,
                                location: [1, 1, 2],
                                steps: [
                                    { __type: RunStepType.SingleRoutine } as SingleRoutineStep,
                                    { __type: RunStepType.SingleRoutine } as SingleRoutineStep,
                                    { __type: RunStepType.SingleRoutine } as SingleRoutineStep,
                                    { __type: RunStepType.SingleRoutine } as SingleRoutineStep,
                                ],
                            },
                            {
                                __type: RunStepType.End,
                                location: [1, 1, 3],
                            },
                        ],
                    } as MultiRoutineStep,
                ],
            } as DirectoryStep;

            expect(new RunStepNavigator(logger).siblingsAtLocation([1, 1, 1], rootStep)).toBe(3);
            expect(new RunStepNavigator(logger).siblingsAtLocation([1, 1, 2], rootStep)).toBe(3);
            expect(new RunStepNavigator(logger).siblingsAtLocation([1, 1, 3], rootStep)).toBe(3);
            expect(new RunStepNavigator(logger).siblingsAtLocation([1, 1, 2, 1], rootStep)).toBe(4);
            expect(new RunStepNavigator(logger).siblingsAtLocation([1, 1, 2, 2], rootStep)).toBe(4);
            expect(new RunStepNavigator(logger).siblingsAtLocation([1, 1, 2, 3], rootStep)).toBe(4);
            expect(new RunStepNavigator(logger).siblingsAtLocation([1, 1, 2, 4], rootStep)).toBe(4);
        });

        it("should return 0 and log an error when parent is not found", () => {
            const rootStep: DirectoryStep = {
                __type: RunStepType.Directory,
                description: "root",
                directoryId: "directory123",
                hasBeenQueried: false,
                isOrdered: false,
                isRoot: true,
                location: [1],
                name: "root",
                projectVersionId: "projectVersion123",
                steps: [],
            };

            const result = new RunStepNavigator(logger).siblingsAtLocation([1, 2, 3], rootStep);
            expect(result).toBe(0);
            expect(console.error).toHaveBeenCalled();
        });
    });

    describe("findStep", () => {
        // Helper function to create a basic DirectoryStep
        function createDirectoryStep(name: string, location: number[]): DirectoryStep {
            return {
                __type: RunStepType.Directory,
                name,
                description: `Description of ${name}`,
                location,
                directoryId: null,
                hasBeenQueried: true,
                isOrdered: false,
                isRoot: location.length === 1,
                projectVersionId: "project123",
                steps: [],
            };
        }

        // Helper function to create a basic SingleRoutineStep
        function createSingleRoutineStep(name: string, location: number[]): SingleRoutineStep {
            return {
                __type: RunStepType.SingleRoutine,
                name,
                description: `Description of ${name}`,
                location,
                routineVersion: { id: "routine123" } as any,
            };
        }

        it("should find a step in a simple DirectoryStep structure", () => {
            const rootStep: DirectoryStep = {
                ...createDirectoryStep("Root", [1]),
                steps: [
                    createSingleRoutineStep("Child1", [1, 1]),
                    createSingleRoutineStep("Child2", [1, 2]),
                ],
            };

            const result = new RunStepNavigator(logger).findStep(rootStep, step => step.name === "Child2");
            expect(result).toEqual(rootStep.steps[1]);
        });

        it("should find a step in a nested DirectoryStep structure", () => {
            const rootStep: DirectoryStep = {
                ...createDirectoryStep("Root", [1]),
                steps: [
                    {
                        ...createDirectoryStep("Nested", [1, 1]),
                        steps: [
                            createSingleRoutineStep("Target", [1, 1, 1]),
                        ],
                    },
                    createSingleRoutineStep("Sibling", [1, 2]),
                ],
            };

            const result = new RunStepNavigator(logger).findStep(rootStep, step => step.name === "Target");
            expect(result).toEqual((rootStep.steps[0] as DirectoryStep).steps[0]);
        });

        it("should find a step in a MultiRoutineStep structure", () => {
            const rootStep: MultiRoutineStep = {
                __type: RunStepType.MultiRoutine,
                name: "Root",
                description: "Root MultiRoutine",
                location: [1],
                routineVersionId: "multiRoutine123",
                nodes: [
                    {
                        __type: RunStepType.Start,
                        name: "Start",
                        description: "Start node",
                        location: [1, 1],
                        nextLocation: [1, 2],
                        nodeId: "start123",
                    },
                    {
                        __type: RunStepType.RoutineList,
                        name: "RoutineList",
                        description: "Routine List",
                        isOrdered: false,
                        location: [1, 2],
                        nextLocation: [1, 3],
                        nodeId: "routineList123",
                        parentRoutineVersionId: "routineVersion123",
                        steps: [
                            createSingleRoutineStep("Target", [1, 2, 1]),
                        ],
                    },
                    {
                        __type: RunStepType.End,
                        name: "End",
                        description: "End node",
                        location: [1, 3],
                        nextLocation: null,
                        nodeId: "end123",
                        wasSuccessful: true,
                    },
                ],
                nodeLinks: [],
                startNodeIndexes: [0],
            };

            const result = new RunStepNavigator(logger).findStep(rootStep, step => step.name === "Target");
            expect(result).toEqual((rootStep.nodes[1] as RoutineListStep).steps[0]);
        });

        it("should find a step in a DecisionStep structure", () => {
            const rootStep: DecisionStep = {
                __type: RunStepType.Decision,
                name: "Decision",
                description: "Decision step",
                location: [1],
                options: [
                    {
                        link: { id: "link1" } as any,
                        step: {
                            __type: RunStepType.RoutineList,
                            name: "RoutineList",
                            description: "Routine List",
                            isOrdered: false,
                            location: [1, 2],
                            nextLocation: [1, 3],
                            nodeId: "routineList123",
                            parentRoutineVersionId: "routineVersion123",
                            steps: [
                                createSingleRoutineStep("Option1", [1, 1]),
                            ],
                        },
                    },
                    {
                        link: { id: "link1" } as any,
                        step: {
                            __type: RunStepType.RoutineList,
                            name: "RoutineList",
                            description: "Routine List",
                            isOrdered: false,
                            location: [1, 2],
                            nextLocation: [1, 3],
                            nodeId: "routineList123",
                            parentRoutineVersionId: "routineVersion123",
                            steps: [
                                createSingleRoutineStep("Target", [1, 2]),
                            ],
                        },
                    },
                ],
            };

            const result = new RunStepNavigator(logger).findStep(rootStep, step => step.name === "Target");
            expect(result).toEqual((rootStep.options[1].step as RoutineListStep).steps[0]);
        });

        it("should return null if no step matches the predicate", () => {
            const rootStep: DirectoryStep = {
                ...createDirectoryStep("Root", [1]),
                steps: [
                    createSingleRoutineStep("Child1", [1, 1]),
                    createSingleRoutineStep("Child2", [1, 2]),
                ],
            };

            const result = new RunStepNavigator(logger).findStep(rootStep, step => step.name === "NonExistent");
            expect(result).toBeNull();
        });

        it("should handle empty structures", () => {
            const emptyDirectory: DirectoryStep = createDirectoryStep("Empty", [1]);
            const emptyMultiRoutine: MultiRoutineStep = {
                __type: RunStepType.MultiRoutine,
                name: "Empty",
                description: "Empty MultiRoutine",
                location: [1],
                routineVersionId: "emptyMultiRoutine123",
                nodes: [],
                nodeLinks: [],
                startNodeIndexes: [],
            };

            expect(new RunStepNavigator(logger).findStep(emptyDirectory, () => true)).toEqual(emptyDirectory);
            expect(new RunStepNavigator(logger).findStep(emptyMultiRoutine, () => true)).toEqual(emptyMultiRoutine);
        });

        it("should find a step based on a complex predicate", () => {
            const rootStep: DirectoryStep = {
                ...createDirectoryStep("Root", [1]),
                steps: [
                    createSingleRoutineStep("Child1", [1, 1]),
                    {
                        ...createDirectoryStep("Nested", [1, 2]),
                        steps: [
                            createSingleRoutineStep("Target", [1, 2, 1]),
                        ],
                    },
                ],
            };

            const result = new RunStepNavigator(logger).findStep(rootStep, step =>
                step.__type === RunStepType.SingleRoutine &&
                step.name === "Target" &&
                step.location.length === 3,
            );
            expect(result).toEqual((rootStep.steps[1] as DirectoryStep).steps[0]);
        });

        it("should return the first matching step when multiple steps match", () => {
            const rootStep: DirectoryStep = {
                ...createDirectoryStep("Root", [1]),
                steps: [
                    createSingleRoutineStep("Target", [1, 1]),
                    createSingleRoutineStep("Target", [1, 2]),
                ],
            };

            const result = new RunStepNavigator(logger).findStep(rootStep, step => step.name === "Target");
            expect(result).toEqual(rootStep.steps[0]);
        });
    });

    describe("locationArraysMatch", () => {
        it("should return true for two empty arrays", () => {
            expect(new RunStepNavigator(logger).locationArraysMatch([], [])).toBe(true);
        });

        it("should return true for identical single-element arrays", () => {
            expect(new RunStepNavigator(logger).locationArraysMatch([1], [1])).toBe(true);
        });

        it("should return false for different single-element arrays", () => {
            expect(new RunStepNavigator(logger).locationArraysMatch([1], [2])).toBe(false);
        });

        it("should return true for identical multi-element arrays", () => {
            expect(new RunStepNavigator(logger).locationArraysMatch([1, 2, 3], [1, 2, 3])).toBe(true);
        });

        it("should return false for arrays with different lengths", () => {
            expect(new RunStepNavigator(logger).locationArraysMatch([1, 2], [1, 2, 3])).toBe(false);
        });

        it("should return false for arrays with same length but different elements", () => {
            expect(new RunStepNavigator(logger).locationArraysMatch([1, 2, 3], [1, 2, 4])).toBe(false);
        });

        it("should return false for arrays with elements in different order", () => {
            expect(new RunStepNavigator(logger).locationArraysMatch([1, 2, 3], [3, 2, 1])).toBe(false);
        });

        it("should handle large arrays", () => {
            const largeArray = Array.from({ length: 1000 }, (_, i) => i);
            expect(new RunStepNavigator(logger).locationArraysMatch(largeArray, largeArray)).toBe(true);
        });

        it("should return true for arrays with negative numbers", () => {
            expect(new RunStepNavigator(logger).locationArraysMatch([-1, -2, -3], [-1, -2, -3])).toBe(true);
        });

        it("should return false for arrays with mixed positive and negative numbers", () => {
            expect(new RunStepNavigator(logger).locationArraysMatch([1, -2, 3], [1, 2, 3])).toBe(false);
        });

        it("should handle arrays with repeated elements", () => {
            expect(new RunStepNavigator(logger).locationArraysMatch([1, 1, 2, 2], [1, 1, 2, 2])).toBe(true);
        });

        it("should return false for arrays with different repeated elements", () => {
            expect(new RunStepNavigator(logger).locationArraysMatch([1, 1, 2, 2], [1, 2, 2, 2])).toBe(false);
        });

        it("should handle arrays with zero", () => {
            expect(new RunStepNavigator(logger).locationArraysMatch([0, 1, 2], [0, 1, 2])).toBe(true);
        });

        it("should return false when comparing with undefined", () => {
            expect(new RunStepNavigator(logger).locationArraysMatch([1, 2, 3], undefined as any)).toBe(false);
        });

        it("should return false when comparing with null", () => {
            expect(new RunStepNavigator(logger).locationArraysMatch([1, 2, 3], null as any)).toBe(false);
        });
    });

    describe("stepNeedsQuerying", () => {
        beforeAll(() => {
            // eslint-disable-next-line @typescript-eslint/no-empty-function
            jest.spyOn(console, "warn").mockImplementation(() => { });
        });
        afterAll(() => {
            jest.restoreAllMocks();
        });

        it("returns false if the step is null", () => {
            expect(new RunStepNavigator(logger).stepNeedsQuerying(null)).toBe(false);
        });

        it("returns false if the step is undefined", () => {
            expect(new RunStepNavigator(logger).stepNeedsQuerying(undefined)).toBe(false);
        });

        it("returns false for an EndStep", () => {
            const endStep: EndStep = {
                __type: RunStepType.End,
                description: "hi",
                name: "End",
                nextLocation: null,
                nodeId: "node123",
                location: [1, 2],
                wasSuccessful: true,
            };
            expect(new RunStepNavigator(logger).stepNeedsQuerying(endStep)).toBe(false);
        });

        it("returns false for a StartStep", () => {
            const startStep: StartStep = {
                __type: RunStepType.Start,
                description: "hey",
                name: "Start",
                nextLocation: [2, 2, 2],
                nodeId: "node123",
                location: [1, 1],
            };
            expect(new RunStepNavigator(logger).stepNeedsQuerying(startStep)).toBe(false);
        });

        describe("SingleRoutineStep", () => {
            it("returns true if the subroutine has subroutines", () => {
                const singleRoutineStep: SingleRoutineStep = {
                    __type: RunStepType.SingleRoutine,
                    description: "boop",
                    name: "Subroutine Step",
                    location: [1],
                    routineVersion: {
                        id: "routine123",
                        routineType: RoutineType.MultiStep,
                        nodes: [{ id: "node123" }] as Node[],
                        nodeLinks: [{ id: "link123" }] as NodeLink[],
                    } as RoutineVersion,
                };
                expect(new RunStepNavigator(logger).stepNeedsQuerying(singleRoutineStep)).toBe(true);
            });

            it("returns false if the subroutine does not have subroutines", () => {
                const singleRoutineStep: SingleRoutineStep = {
                    __type: RunStepType.SingleRoutine,
                    description: "boopies",
                    name: "Subroutine Step",
                    location: [1],
                    routineVersion: {
                        id: "routine123",
                        routineType: RoutineType.Informational,
                    } as RoutineVersion,
                };
                expect(new RunStepNavigator(logger).stepNeedsQuerying(singleRoutineStep)).toBe(false);
            });
        });

        describe("DirectoryStep", () => {
            it("returns true if the directory has not been queried yet", () => {
                const directoryStep: DirectoryStep = {
                    __type: RunStepType.Directory,
                    description: "boop",
                    directoryId: "directory123",
                    hasBeenQueried: false, // Marked as not queried
                    isOrdered: false,
                    isRoot: false,
                    name: "Directory",
                    location: [1],
                    projectVersionId: "projectVersion123",
                    steps: [{ name: "Child Step", __type: RunStepType.SingleRoutine }] as DirectoryStep["steps"],
                };
                expect(new RunStepNavigator(logger).stepNeedsQuerying(directoryStep)).toBe(true);
            });

            it("returns false if the directory has already been queried", () => {
                const directoryStep: DirectoryStep = {
                    __type: RunStepType.Directory,
                    description: "boop",
                    directoryId: "directory123",
                    hasBeenQueried: true, // Marked as already queried
                    isOrdered: false,
                    isRoot: false,
                    name: "Directory",
                    location: [1],
                    projectVersionId: "projectVersion123",
                    steps: [{ name: "Child Step", __type: RunStepType.SingleRoutine }] as DirectoryStep["steps"],
                };
                expect(new RunStepNavigator(logger).stepNeedsQuerying(directoryStep)).toBe(false);
            });
        });
    });
});

describe("getStepComplexity", () => {
    beforeAll(() => {
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "warn").mockImplementation(() => { });
    });
    afterAll(() => {
        jest.restoreAllMocks();
    });

    it("should return 0 for End step", () => {
        const endStep: RunStep = {
            __type: RunStepType.End,
            description: "",
            location: [1, 2, 3],
            name: "",
            nextLocation: null,
            nodeId: "1",
            wasSuccessful: true,
        };
        expect(getStepComplexity(endStep, console)).toBe(0);
    });

    it("should return 0 for Start step", () => {
        const startStep: RunStep = {
            __type: RunStepType.Start,
            description: "",
            name: "",
            nextLocation: [1, 2, 3, 4, 5],
            nodeId: "1",
            location: [1],
        };
        expect(getStepComplexity(startStep, console)).toBe(0);
    });

    it("should return 1 for Decision step", () => {
        const decisionStep: RunStep = {
            __type: RunStepType.Decision,
            description: "",
            name: "",
            location: [1],
            options: [],
        };
        expect(getStepComplexity(decisionStep, console)).toBe(1);
    });

    it("should return the complexity of SingleRoutine step", () => {
        const singleRoutineStep: SingleRoutineStep = {
            __type: RunStepType.SingleRoutine,
            description: "",
            name: "",
            location: [1],
            routineVersion: { complexity: 3, id: "1", routineType: RoutineType.MultiStep } as RoutineVersion,
        };
        expect(getStepComplexity(singleRoutineStep, console)).toBe(3); // Complexity of the routine version
    });

    it("should calculate complexity for MultiRoutine step", () => {
        const multiRoutineStep: MultiRoutineStep = {
            __type: RunStepType.MultiRoutine,
            description: "",
            name: "",
            location: [1],
            nodes: [
                {
                    __type: RunStepType.Start,
                    description: "",
                    name: "",
                    nextLocation: [],
                    nodeId: "1",
                    location: [1, 1],
                },
                {
                    __type: RunStepType.Decision,
                    description: "",
                    name: "",
                    location: [1, 2],
                    options: [],
                },
                {
                    __type: RunStepType.RoutineList,
                    description: "",
                    isOrdered: true,
                    location: [3, 3, 3, 3, 3],
                    name: "",
                    nextLocation: [2, 2, 2],
                    nodeId: "1",
                    parentRoutineVersionId: "133",
                    steps: [
                        {
                            __type: RunStepType.SingleRoutine,
                            description: "",
                            name: "",
                            location: [1, 1],
                            routineVersion: { complexity: 1, id: "1", routineType: RoutineType.Informational } as RoutineVersion,
                        },
                        {
                            __type: RunStepType.SingleRoutine,
                            description: "",
                            name: "",
                            location: [1, 2],
                            routineVersion: { complexity: 9, id: "2", routineType: RoutineType.MultiStep } as RoutineVersion,
                        },
                    ],
                },
                {
                    __type: RunStepType.End,
                    description: "",
                    name: "",
                    nextLocation: null,
                    nodeId: "3",
                    location: [1, 3],
                    wasSuccessful: false,
                },
            ],
            nodeLinks: [],
            routineVersionId: "1",
            startNodeIndexes: [0],
        };
        expect(getStepComplexity(multiRoutineStep, console)).toBe(11);
    });

    it("should calculate complexity for RoutineList step", () => {
        const routineListStep: RoutineListStep = {
            __type: RunStepType.RoutineList,
            description: "",
            isOrdered: false,
            name: "",
            nextLocation: null,
            nodeId: "1",
            location: [1],
            parentRoutineVersionId: "420",
            steps: [
                {
                    __type: RunStepType.SingleRoutine,
                    description: "",
                    name: "",
                    location: [1, 1],
                    routineVersion: { complexity: 2, id: "1", routineType: RoutineType.MultiStep } as RoutineVersion,
                },
                {
                    __type: RunStepType.MultiRoutine,
                    description: "root",
                    location: [1],
                    name: "root",
                    nodes: [
                        {
                            __type: RunStepType.Start,
                            description: "start",
                            location: [1, 1],
                            name: "start",
                            nextLocation: [1, 2],
                            nodeId: "startNode",
                        },
                        {
                            __type: RunStepType.Decision,
                            description: "decision",
                            location: [1, 2],
                            name: "Decision Step",
                            options: [
                                {
                                    link: {} as NodeLink,
                                    step: {
                                        __type: RunStepType.RoutineList,
                                        description: "routine list 1",
                                        isOrdered: false,
                                        location: [1, 3, 2, 1],
                                        name: "routine list a",
                                        nextLocation: [],
                                        nodeId: "routineListNode3",
                                        parentRoutineVersionId: "23fdhsaf",
                                        steps: [
                                            {
                                                __type: RunStepType.SingleRoutine,
                                                description: "subroutine 1",
                                                location: [1, 3, 2, 1, 1],
                                                name: "subroutine a",
                                                routineVersion: { id: "routineVersion345", complexity: 100000 } as RoutineVersion,
                                            },
                                        ],
                                    } as RoutineListStep,
                                },
                                {
                                    link: {} as NodeLink,
                                    step: {
                                        __type: RunStepType.RoutineList,
                                        description: "routine list 2",
                                        isOrdered: true,
                                        location: [1, 4, 2, 1, 2, 2, 2],
                                        name: "routine list b",
                                        nextLocation: [1, 4, 2, 1, 2, 2, 3],
                                        nodeId: "routineListNode3",
                                        parentRoutineVersionId: "23fdhsaf",
                                        steps: [
                                            {
                                                __type: RunStepType.SingleRoutine,
                                                description: "subroutine 2",
                                                location: [1, 3, 2, 1, 1],
                                                name: "subroutine b",
                                                routineVersion: { id: "routineVersion345", complexity: 9999999 } as RoutineVersion,
                                            },
                                        ],
                                    } as RoutineListStep,
                                },
                            ],
                        },
                        {
                            __type: RunStepType.RoutineList,
                            description: "routine list",
                            isOrdered: true,
                            location: [1, 3],
                            name: "routine list",
                            nextLocation: [1, 4],
                            nodeId: "routineListNode2",
                            parentRoutineVersionId: "routine123",
                            steps: [
                                {
                                    __type: RunStepType.SingleRoutine,
                                    description: "subroutine",
                                    location: [1, 3, 1],
                                    name: "subroutine",
                                    routineVersion: { id: "routineVersion234", complexity: 2 } as RoutineVersion,
                                },
                                {
                                    __type: RunStepType.MultiRoutine,
                                    description: "multi routine",
                                    location: [1, 3, 2],
                                    name: "multi routine",
                                    nodes: [
                                        {
                                            __type: RunStepType.RoutineList,
                                            description: "routine list",
                                            isOrdered: true,
                                            location: [1, 3, 2, 1],
                                            name: "routine list",
                                            nextLocation: [1, 1, 1, 1],
                                            nodeId: "routineListNode3",
                                            parentRoutineVersionId: "chicken",
                                            steps: [
                                                {
                                                    __type: RunStepType.SingleRoutine,
                                                    description: "subroutine",
                                                    location: [1, 3, 2, 1, 1],
                                                    name: "subroutine",
                                                    routineVersion: { id: "routineVersion345", complexity: 10 } as RoutineVersion,
                                                },
                                            ],
                                        },
                                    ],
                                    nodeLinks: [],
                                    routineVersionId: "multiRoutine123",
                                    startNodeIndexes: [],
                                },
                            ],
                        },
                        {
                            __type: RunStepType.End,
                            description: "end",
                            location: [1, 4],
                            name: "end",
                            nextLocation: null,
                            nodeId: "endNode",
                            wasSuccessful: true,
                        },
                    ],
                    nodeLinks: [],
                    routineVersionId: "rootRoutine",
                    startNodeIndexes: [0],
                },
            ],
        };
        expect(getStepComplexity(routineListStep, console)).toBe(15);
    });

    it("should calculate complexity for Directory step", () => {
        const directoryStep: DirectoryStep = {
            __type: RunStepType.Directory,
            description: "",
            name: "",
            location: [1],
            directoryId: "1",
            hasBeenQueried: true,
            isOrdered: false,
            isRoot: false,
            projectVersionId: "1",
            steps: [
                {
                    __type: RunStepType.SingleRoutine,
                    description: "",
                    name: "",
                    location: [1, 1],
                    routineVersion: { complexity: 2, id: "1", routineType: RoutineType.MultiStep } as RoutineVersion,
                },
                {
                    __type: RunStepType.Directory,
                    description: "b",
                    directoryId: null,
                    hasBeenQueried: false,
                    isOrdered: false,
                    isRoot: true,
                    location: [1, 1],
                    name: "b",
                    projectVersionId: "projectVersion123", // Matches mockStepData, so it should be replaced by mockStepData
                    steps: [{
                        __type: RunStepType.SingleRoutine,
                        description: "",
                        name: "",
                        location: [1, 1],
                        routineVersion: { complexity: 12, id: "1", routineType: RoutineType.MultiStep } as RoutineVersion,
                    }],
                },
            ],
        };
        expect(getStepComplexity(directoryStep, console)).toBe(14);
    });
});

/**
 * Helper function to validate that routine steps are in a logical order
 */
function expectValidStepSequence(
    steps: MultiRoutineStep["nodes"],
    nodeLinks: NodeLink[],
) {
    if (steps.length === 0) return;
    // Length of location array
    let locationLength = 0;
    const usedLocations = new Set<string>();

    // For each step
    for (let i = 0; i < steps.length; i++) {
        const step = steps[i];

        const locationStr = step.location.join(",");
        // Location should be unique
        expect(usedLocations.has(locationStr)).toBe(
            false,
            // @ts-ignore: expect-message
            `Expected location ${locationStr} to be unique`,
        );
        usedLocations.add(locationStr);

        // Should only be one StartStep, and should be first
        if (i === 0) {
            expect(step.__type).toBe(
                RunStepType.Start,
                // @ts-ignore: expect-message
                `Expected StartStep to be first in the sequence. Got ${step.__type}`,
            );
            locationLength = step.location.length;
            // Last number in location should be 1
            expect(step.location[locationLength - 1]).toBe(
                1,
                // @ts-ignore: expect-message
                `Expected StartStep to have a location ending in 1. Got ${step.location[locationLength - 1]}`,
            );
        } else {
            expect(step.__type).not.toBe(
                RunStepType.Start,
                // @ts-ignore: expect-message
                `Expected StartStep in a different position than index 0. Got ${step.__type} at index ${i}`,
            );
            // Last number in location should not be 1
            expect(step.location[locationLength - 1]).not.toBe(
                1,
                // @ts-ignore: expect-message
                `Expected StartStep to have a location ending in 1. Got ${step.location[locationLength - 1]}`,
            );
        }

        // If it's an EndStep or DecisionStep, `nextLocation` should be null or undefined
        if ([RunStepType.End, RunStepType.Decision].includes(step.__type)) {
            const nextLocation = (step as { nextLocation?: string | null }).nextLocation;
            expect(nextLocation === null || nextLocation === undefined).toBe(
                true,
                // @ts-ignore: expect-message
                `Expected EndStep or DecisionStep to have a null/undefined nextLocation. Got ${nextLocation}`,
            );
            // Rest of the checks don't apply to EndStep or DecisionStep
            continue;
        }
        // If it's a RoutineList or StartStep, `nextLocation` should be defined and valid
        else if ([RunStepType.RoutineList, RunStepType.Start].includes(step.__type)) {
            const currNodeId = (step as StartStep | RoutineListStep).nodeId;
            const currLocation = step.location;
            const nextLocation = (step as StartStep | RoutineListStep).nextLocation;
            const outgoingLinks = nodeLinks.filter(link => link.from?.id === currNodeId);
            const nextStep = steps.find(s => s.location.join(",") === nextLocation?.join(","));

            expect(Array.isArray(nextLocation)).toBe(
                true,
                // @ts-ignore: expect-message
                `Expected StartStep or RoutineListStep to have a defined nextLocation. Got ${typeof nextLocation} ${nextLocation}`,
            );
            expect(JSON.stringify(nextLocation)).not.toBe(
                JSON.stringify(currLocation),
                // @ts-ignore: expect-message
                `Expected nextLocation to not point to itself. Got ${nextLocation}`,
            );
            expect(Boolean(nextStep)).toBe(
                true,
                // @ts-ignore: expect-message
                `Expected nextLocation to point to a valid step. Got ${nextLocation}`,
            );
            expect(outgoingLinks.length).toBeGreaterThanOrEqual(
                1,
                // @ts-ignore: expect-message
                `Expected at least one outgoing link from step ${currNodeId}. Got ${outgoingLinks.length}`,
            );

            // If there are multiple paths to take, we should point to a valid DecisionStep
            if (outgoingLinks.length > 1) {
                expect(nextStep?.__type).toBe(
                    RunStepType.Decision,
                    // @ts-ignore: expect-message
                    `Expected nextLocation to point to a DecisionStep. Got ${nextStep?.__type}`,
                );
                outgoingLinks.forEach(link => {
                    expect((nextStep as DecisionStep).options.some(option => option.link.to?.id === link.to?.id)).toBe(
                        true,
                        // @ts-ignore: expect-message
                        `Expected outgoing link ${link.to?.id} to be an option in the DecisionStep`,
                    );
                });
                (nextStep as DecisionStep).options.forEach(option => {
                    expect(outgoingLinks.some(link => link.to?.id === option.link.to?.id)).toBe(
                        true,
                        // @ts-ignore: expect-message
                        `Expected DecisionStep option ${option.link.to?.id} to have a corresponding outgoing link`,
                    );
                });
            }
            // If there is only one path to take, we should point to anythign but a DecisionStep
            else {
                expect(nextStep?.__type).not.toBe(
                    RunStepType.Decision,
                    // @ts-ignore: expect-message
                    `Expected nextLocation to not point to a DecisionStep. Got ${nextStep?.__type}`,
                );
                expect(outgoingLinks[0].to?.id).toBe(
                    (nextStep as EndStep | RoutineListStep | StartStep).nodeId,
                    // @ts-ignore: expect-message
                    `Expected outgoing link to match nextLocation. Got ${outgoingLinks[0].to?.id}, expected ${nextLocation}`,
                );
            }
        }
        // Otherwise, it's an invalid step type
        else {
            expect(false).toBe(
                true,
                // @ts-ignore: expect-message
                `Expected step to be a StartStep, RoutineListStep, DecisionStep, or EndStep. Got ${step.__type}`,
            );
        }
    }
}

describe("RunStepBuilder", () => {
    const logger = console;
    const languages = ["en"];

    describe("runnableObjectToStep", () => {
        it("should convert a single-step routine", () => {
            const routineVersion: RunnableRoutineVersion = {
                __typename: "RoutineVersion",
                id: uuid(),
                created_at: new Date().toISOString(),
                complexity: 2,
                configCallData: "{}",
                configFormInput: "{}",
                configFormOutput: "{}",
                inputs: [],
                nodeLinks: [],
                nodes: [],
                outputs: [],
                root: {
                    __typename: "Routine",
                    id: uuid(),
                } as Routine,
                routineType: RoutineType.Informational,
                translations: [{
                    __typename: "RoutineVersionTranslation",
                    id: uuid(),
                    language: "en",
                    name: "english name",
                    description: "english description",
                }, {
                    __typename: "RoutineVersionTranslation",
                    id: uuid(),
                    language: "es",
                    name: "spanish name",
                    description: "spanish description",
                }],
                versionLabel: "1.0.0",
                you: {} as RoutineVersionYou,
            };
            const result = new RunStepBuilder(languages, logger).runnableObjectToStep(routineVersion, [1]);
            expect(result).toEqual({
                __type: RunStepType.SingleRoutine,
                description: "english description",
                name: "english name",
                location: [1],
                routineVersion,
            });
        });

        it("should convert a multi-step routine", () => {
            const startId = uuid();
            const routineListId = uuid();
            const endId = uuid();

            const routineVersion: RunnableRoutineVersion = {
                __typename: "RoutineVersion",
                id: uuid(),
                created_at: new Date().toISOString(),
                complexity: 2,
                configCallData: "{}",
                configFormInput: "{}",
                configFormOutput: "{}",
                inputs: [],
                nodeLinks: [{
                    __typename: "NodeLink",
                    id: uuid(),
                    from: { id: startId },
                    to: { id: routineListId },
                } as NodeLink, {
                    __typename: "NodeLink",
                    id: uuid(),
                    from: { id: routineListId },
                    to: { id: endId },
                } as NodeLink],
                nodes: [{
                    __typename: "Node",
                    created_at: new Date().toISOString(),
                    updated_at: new Date().toISOString(),
                    id: startId,
                    columnIndex: 0,
                    rowIndex: 0,
                    nodeType: NodeType.Start,
                } as Node, {
                    __typename: "Node",
                    created_at: new Date().toISOString(),
                    updated_at: new Date().toISOString(),
                    id: routineListId,
                    columnIndex: 0,
                    rowIndex: 0,
                    nodeType: NodeType.RoutineList,
                    routineList: {
                        __typename: "NodeRoutineList",
                        id: uuid(),
                        isOptional: true,
                        isOrdered: false,
                        items: [{
                            __typename: "NodeRoutineListItem",
                            id: uuid(),
                            index: 0,
                            routineVersion: {
                                __typename: "RoutineVersion",
                                id: uuid(),
                                complexity: 3,
                            },
                        }, {
                            __typename: "NodeRoutineListItem",
                            id: uuid(),
                            index: 1,
                            routineVersion: {
                                __typename: "RoutineVersion",
                                id: uuid(),
                                complexity: 4,
                                translations: [{
                                    __typename: "RoutineVersionTranslation",
                                    id: uuid(),
                                    language: "en",
                                    description: "thee description",
                                    name: "thee name",
                                }],
                            },
                        }],
                    },
                } as Node, {
                    __typename: "Node",
                    created_at: new Date().toISOString(),
                    updated_at: new Date().toISOString(),
                    id: endId,
                    columnIndex: 0,
                    rowIndex: 0,
                    nodeType: NodeType.End,
                    end: {
                        __typename: "NodeEnd",
                        id: uuid(),
                        wasSuccessful: true,
                    },
                } as Node],
                outputs: [],
                root: {
                    __typename: "Routine",
                    id: uuid(),
                } as Routine,
                routineType: RoutineType.MultiStep,
                translations: [{
                    __typename: "RoutineVersionTranslation",
                    id: uuid(),
                    language: "en",
                    name: "english name",
                    description: "english description",
                }, {
                    __typename: "RoutineVersionTranslation",
                    id: uuid(),
                    language: "es",
                    name: "spanish name",
                    description: "spanish description",
                }],
                versionLabel: "1.0.0",
                you: {} as RoutineVersionYou,
            };
            const result = new RunStepBuilder(languages, logger).runnableObjectToStep(routineVersion, [1, 2]);
            expect(result).toEqual({
                __type: RunStepType.MultiRoutine,
                description: "english description",
                name: "english name",
                location: [1, 2],
                routineVersionId: routineVersion.id,
                nodeLinks: routineVersion.nodeLinks,
                nodes: [{
                    __type: RunStepType.Start,
                    description: null,
                    location: [1, 2, 1],
                    name: "Untitled",
                    nextLocation: [1, 2, 2],
                    nodeId: routineVersion.nodes[0].id,
                }, {
                    __type: RunStepType.RoutineList,
                    description: null,
                    isOrdered: false,
                    location: [1, 2, 2],
                    name: "Untitled",
                    nextLocation: [1, 2, 3],
                    nodeId: routineVersion.nodes[1].id,
                    parentRoutineVersionId: routineVersion.id,
                    steps: [
                        {
                            __type: RunStepType.SingleRoutine,
                            description: null,
                            location: [1, 2, 2, 1],
                            name: "Untitled",
                            routineVersion: (routineVersion as any).nodes[1].routineList.items[0].routineVersion,
                        },
                        {
                            __type: RunStepType.SingleRoutine,
                            description: "thee description",
                            location: [1, 2, 2, 2],
                            name: "thee name",
                            routineVersion: (routineVersion as any).nodes[1].routineList.items[1].routineVersion,
                        },
                    ],
                }, {
                    __type: RunStepType.End,
                    description: null,
                    location: [1, 2, 3],
                    name: "Untitled",
                    nextLocation: null,
                    nodeId: routineVersion.nodes[2].id,
                    wasSuccessful: true,
                }],
                startNodeIndexes: [0],
            });
        });

        it("should convert a project", () => {
            const projectVersion: RunnableProjectVersion = {
                __typename: "ProjectVersion",
                id: uuid(),
                created_at: new Date().toISOString(),
                directories: [{
                    __typename: "ProjectVersionDirectory",
                    id: uuid(),
                    created_at: new Date().toISOString(),
                    updated_at: new Date().toISOString(),
                    isRoot: false,
                    translations: [{
                        __typename: "ProjectVersionDirectoryTranslation",
                        id: uuid(),
                        language: "en",
                        description: "directory description",
                        name: "directory name",
                    }],
                } as ProjectVersionDirectory],
                root: {
                    __typename: "Project",
                    id: uuid(),
                } as Project,
                translations: [{
                    __typename: "ProjectVersionTranslation",
                    id: uuid(),
                    language: "en",
                    description: "english description",
                    name: "english name",
                }],
                versionLabel: "1.2.3",
                you: {} as ProjectVersionYou,
            };
            const result = new RunStepBuilder(languages, logger).runnableObjectToStep(projectVersion, [1, 2]);
            expect(result).toEqual({
                __type: RunStepType.Directory,
                description: "english description",
                name: "english name",
                location: [1, 2],
                directoryId: null,
                hasBeenQueried: true,
                isOrdered: false,
                isRoot: true,
                projectVersionId: projectVersion.id,
                steps: [{
                    __type: RunStepType.Directory,
                    description: "directory description",
                    location: [1, 2, 1],
                    name: "directory name",
                    directoryId: (projectVersion as any).directories[0].id,
                    hasBeenQueried: true,
                    isOrdered: false,
                    isRoot: false,
                    projectVersionId: projectVersion.id,
                    steps: [],
                }],
            });
        });
    });

    describe("multiRoutineToStep", () => {
        it("Single swim lane (one start node)", () => {
            const startId = uuid();
            const routineListId = uuid();
            const endId = uuid();

            const routineVersion: RunnableRoutineVersion = {
                __typename: "RoutineVersion",
                id: uuid(),
                created_at: new Date().toISOString(),
                complexity: 2,
                configCallData: "{}",
                configFormInput: "{}",
                configFormOutput: "{}",
                inputs: [],
                nodeLinks: [{
                    __typename: "NodeLink",
                    id: uuid(),
                    from: { id: startId },
                    to: { id: routineListId },
                } as NodeLink, {
                    __typename: "NodeLink",
                    id: uuid(),
                    from: { id: routineListId },
                    to: { id: endId },
                } as NodeLink],
                nodes: [{
                    __typename: "Node",
                    created_at: new Date().toISOString(),
                    updated_at: new Date().toISOString(),
                    id: startId,
                    columnIndex: 0,
                    rowIndex: 0,
                    nodeType: NodeType.Start,
                } as Node, {
                    __typename: "Node",
                    created_at: new Date().toISOString(),
                    updated_at: new Date().toISOString(),
                    id: routineListId,
                    columnIndex: 0,
                    rowIndex: 0,
                    nodeType: NodeType.RoutineList,
                    routineList: {
                        __typename: "NodeRoutineList",
                        id: uuid(),
                        isOptional: true,
                        isOrdered: false,
                        items: [{
                            __typename: "NodeRoutineListItem",
                            id: uuid(),
                            index: 0,
                            routineVersion: {
                                __typename: "RoutineVersion",
                                id: uuid(),
                                complexity: 3,
                            },
                        }, {
                            __typename: "NodeRoutineListItem",
                            id: uuid(),
                            index: 1,
                            routineVersion: {
                                __typename: "RoutineVersion",
                                id: uuid(),
                                complexity: 4,
                                translations: [{
                                    __typename: "RoutineVersionTranslation",
                                    id: uuid(),
                                    language: "en",
                                    description: "thee description",
                                    name: "thee name",
                                }],
                            },
                        }],
                    },
                } as Node, {
                    __typename: "Node",
                    created_at: new Date().toISOString(),
                    updated_at: new Date().toISOString(),
                    id: endId,
                    columnIndex: 0,
                    rowIndex: 0,
                    nodeType: NodeType.End,
                    end: {
                        __typename: "NodeEnd",
                        id: uuid(),
                        wasSuccessful: true,
                    },
                } as Node],
                outputs: [],
                root: {
                    __typename: "Routine",
                    id: uuid(),
                } as Routine,
                routineType: RoutineType.Informational,
                translations: [{
                    __typename: "RoutineVersionTranslation",
                    id: uuid(),
                    language: "en",
                    name: "english name",
                    description: "english description",
                }, {
                    __typename: "RoutineVersionTranslation",
                    id: uuid(),
                    language: "es",
                    name: "spanish name",
                    description: "spanish description",
                }],
                versionLabel: "1.0.0",
                you: {} as RoutineVersionYou,
            };
            const result = new RunStepBuilder(languages, logger).multiRoutineToStep(routineVersion, [1, 2]);
            expect(result).toEqual({
                __type: RunStepType.MultiRoutine,
                description: "english description",
                name: "english name",
                location: [1, 2],
                routineVersionId: routineVersion.id,
                nodeLinks: routineVersion.nodeLinks,
                nodes: [{
                    __type: RunStepType.Start,
                    description: null,
                    location: [1, 2, 1],
                    name: "Untitled",
                    nextLocation: [1, 2, 2],
                    nodeId: routineVersion.nodes[0].id,
                }, {
                    __type: RunStepType.RoutineList,
                    description: null,
                    isOrdered: false,
                    location: [1, 2, 2],
                    name: "Untitled",
                    nextLocation: [1, 2, 3],
                    nodeId: routineVersion.nodes[1].id,
                    parentRoutineVersionId: routineVersion.id,
                    steps: [
                        {
                            __type: RunStepType.SingleRoutine,
                            description: null,
                            location: [1, 2, 2, 1],
                            name: "Untitled",
                            routineVersion: (routineVersion as any).nodes[1].routineList.items[0].routineVersion,
                        },
                        {
                            __type: RunStepType.SingleRoutine,
                            description: "thee description",
                            location: [1, 2, 2, 2],
                            name: "thee name",
                            routineVersion: (routineVersion as any).nodes[1].routineList.items[1].routineVersion,
                        },
                    ],
                }, {
                    __type: RunStepType.End,
                    description: null,
                    location: [1, 2, 3],
                    name: "Untitled",
                    nextLocation: null,
                    nodeId: routineVersion.nodes[2].id,
                    wasSuccessful: true,
                }],
                startNodeIndexes: [0],
            });
        });
        it("Multiple swim lanes (two start nodes)", () => {
            // Generate unique IDs
            const startId1 = uuid();
            const routineListId1 = uuid();
            const endId1 = uuid();

            const startId2 = uuid();
            const endId2 = uuid();

            // Construct a routineVersion representing two separate lanes:
            // Lane 1: Start (startId1) -> RoutineList (routineListId1) -> End (endId1)
            // Lane 2: Start (startId2) -> End (endId2)
            const routineVersion: RunnableRoutineVersion = {
                __typename: "RoutineVersion",
                id: uuid(),
                created_at: new Date().toISOString(),
                complexity: 2,
                configCallData: "{}",
                configFormInput: "{}",
                configFormOutput: "{}",
                inputs: [],
                nodeLinks: [
                    {
                        __typename: "NodeLink",
                        id: uuid(),
                        from: { id: startId1 },
                        to: { id: routineListId1 },
                    } as NodeLink,
                    {
                        __typename: "NodeLink",
                        id: uuid(),
                        from: { id: routineListId1 },
                        to: { id: endId1 },
                    } as NodeLink,
                    {
                        __typename: "NodeLink",
                        id: uuid(),
                        from: { id: startId2 },
                        to: { id: endId2 },
                    } as NodeLink,
                ],
                nodes: [
                    {
                        __typename: "Node",
                        created_at: new Date().toISOString(),
                        updated_at: new Date().toISOString(),
                        id: startId1,
                        columnIndex: 0,
                        rowIndex: 0,
                        nodeType: NodeType.Start,
                    } as Node,
                    {
                        __typename: "Node",
                        created_at: new Date().toISOString(),
                        updated_at: new Date().toISOString(),
                        id: routineListId1,
                        columnIndex: 1,
                        rowIndex: 0,
                        nodeType: NodeType.RoutineList,
                        routineList: {
                            __typename: "NodeRoutineList",
                            id: uuid(),
                            isOptional: true,
                            isOrdered: false,
                            items: [
                                {
                                    __typename: "NodeRoutineListItem",
                                    id: uuid(),
                                    index: 0,
                                    routineVersion: {
                                        __typename: "RoutineVersion",
                                        id: uuid(),
                                        complexity: 3,
                                    },
                                },
                            ],
                        },
                    } as Node,
                    {
                        __typename: "Node",
                        created_at: new Date().toISOString(),
                        updated_at: new Date().toISOString(),
                        id: endId1,
                        columnIndex: 2,
                        rowIndex: 0,
                        nodeType: NodeType.End,
                        end: {
                            __typename: "NodeEnd",
                            id: uuid(),
                            wasSuccessful: true,
                        },
                    } as Node,

                    // Second lane's nodes
                    {
                        __typename: "Node",
                        created_at: new Date().toISOString(),
                        updated_at: new Date().toISOString(),
                        id: startId2,
                        columnIndex: 0,
                        rowIndex: 1,
                        nodeType: NodeType.Start,
                    } as Node,
                    {
                        __typename: "Node",
                        created_at: new Date().toISOString(),
                        updated_at: new Date().toISOString(),
                        id: endId2,
                        columnIndex: 1,
                        rowIndex: 1,
                        nodeType: NodeType.End,
                        end: {
                            __typename: "NodeEnd",
                            id: uuid(),
                            wasSuccessful: false,
                        },
                    } as Node,
                ],
                outputs: [],
                root: {
                    __typename: "Routine",
                    id: uuid(),
                } as Routine,
                routineType: RoutineType.MultiStep, // Important: multi-step routine
                translations: [{
                    __typename: "RoutineVersionTranslation",
                    id: uuid(),
                    language: "en",
                    name: "multi-lane routine",
                    description: "A routine with multiple start nodes",
                }],
                versionLabel: "1.0.0",
                you: {} as RoutineVersionYou,
            };

            const result = new RunStepBuilder(languages, logger).multiRoutineToStep(routineVersion, [1, 2]);

            // We expect two start nodes. Each will yield a separate subtree in the final nodes array.
            // The first start node encountered will have startNodeIndexes[0] = 0 (the first node in result.nodes)
            // The second start node encountered will be appended after the first lane's nodes have been processed.
            // Since lane 1 has 3 steps: Start, RoutineList, End => indexes: 0,1,2
            // Lane 2 will start at index 3.

            expect(result).toBeTruthy();
            expect(result!.__type).toBe(RunStepType.MultiRoutine);
            expect(result!.startNodeIndexes).toEqual([0, 3]); // Lane 1 starts at nodes[0], Lane 2 at nodes[3]

            expect(result!.nodes.length).toBe(5);

            // Lane 1 steps:
            // 0: Start
            expect(result!.nodes[0]).toMatchObject({
                __type: RunStepType.Start,
                nodeId: startId1,
            });
            // 1: RoutineList
            expect(result!.nodes[1]).toMatchObject({
                __type: RunStepType.RoutineList,
                nodeId: routineListId1,
            });
            // 2: End
            expect(result!.nodes[2]).toMatchObject({
                __type: RunStepType.End,
                nodeId: endId1,
                wasSuccessful: true,
            });

            // Lane 2 steps:
            // 3: Start
            expect(result!.nodes[3]).toMatchObject({
                __type: RunStepType.Start,
                nodeId: startId2,
            });
            // 4: End
            expect(result!.nodes[4]).toMatchObject({
                __type: RunStepType.End,
                nodeId: endId2,
                wasSuccessful: false,
            });
        });
    });

    describe("addSubroutinesToStep", () => {
        it("should do nothing when no subroutines are provided", () => {
            const rootStep = {
                __type: RunStepType.MultiRoutine,
                description: "english description",
                name: "english name",
                location: [1, 2],
                routineVersionId: uuid(),
                nodeLinks: [],
                nodes: [],
                startNodeIndexes: [],
            } as MultiRoutineStep;
            const result = new RunStepBuilder(languages, logger).addSubroutinesToStep([], rootStep);
            expect(result).toEqual(rootStep);
        });

        it("should add subroutines where the routine ID matches a SingleRoutineStep's routine ID", () => {
            const rootStep: MultiRoutineStep = {
                __type: RunStepType.MultiRoutine,
                description: "root",
                location: [1],
                name: "root",
                nodes: [{
                    __type: RunStepType.Start,
                    description: "start",
                    location: [1, 1],
                    name: "start",
                    nextLocation: [8, 33, 3],
                    nodeId: "startNode",
                }, {
                    __type: RunStepType.RoutineList,
                    description: "routine list level 1",
                    isOrdered: false,
                    location: [1, 2],
                    name: "routine list level 1",
                    nextLocation: [],
                    nodeId: "routineListNode1",
                    parentRoutineVersionId: "routine123",
                    steps: [{
                        __type: RunStepType.MultiRoutine,
                        description: "multi routine level 2",
                        location: [1, 2, 1],
                        name: "multi routine level 2",
                        nodes: [{
                            __type: RunStepType.RoutineList,
                            description: "routine list level 3",
                            isOrdered: false,
                            name: "routine list level 3",
                            nextLocation: [],
                            location: [1, 2, 1, 1],
                            nodeId: "routineListNode3",
                            parentRoutineVersionId: "routine123",
                            steps: [{
                                __type: RunStepType.SingleRoutine,
                                description: "subroutine",
                                location: [1, 2, 1, 1, 1],
                                name: "subroutine",
                                routineVersion: { id: "deepRoutineVersion" } as RunnableRoutineVersion,
                            }],
                        }],
                        nodeLinks: [],
                        routineVersionId: "level2Routine",
                        startNodeIndexes: [],
                    }],
                }, {
                    __type: RunStepType.End,
                    description: "end",
                    location: [1, 3],
                    name: "end",
                    nextLocation: null,
                    nodeId: "endNode",
                    wasSuccessful: false,
                }],
                nodeLinks: [],
                routineVersionId: "rootRoutine",
                startNodeIndexes: [0],
            };
            const subroutines: RunnableRoutineVersion[] = [{
                __typename: "RoutineVersion",
                id: "deepRoutineVersion",
                created_at: new Date().toISOString(),
                complexity: 2,
                configCallData: "{}",
                configFormInput: "{}",
                configFormOutput: "{}",
                inputs: [],
                nodeLinks: [],
                nodes: [],
                outputs: [],
                root: {
                    __typename: "Routine",
                    id: uuid(),
                } as Routine,
                routineType: RoutineType.MultiStep,
                translations: [{
                    __typename: "RoutineVersionTranslation",
                    id: uuid(),
                    language: "en",
                    name: "subroutine name",
                    description: "subroutine description",
                }],
                versionLabel: "1.0.0",
                you: {} as RoutineVersionYou,
            }];
            const result = new RunStepBuilder(languages, logger).addSubroutinesToStep(subroutines, rootStep);
            expect(result).toEqual({
                ...rootStep,
                nodes: [{
                    ...rootStep.nodes[0],
                }, {
                    ...rootStep.nodes[1],
                    steps: [{
                        ...(rootStep.nodes[1] as RoutineListStep).steps[0],
                        nodes: [{
                            ...((rootStep.nodes[1] as RoutineListStep).steps[0] as MultiRoutineStep).nodes[0],
                            steps: [{
                                __type: RunStepType.MultiRoutine,
                                description: "subroutine description",
                                location: [1, 2, 1, 1, 1],
                                name: "subroutine name",
                                nodeLinks: [],
                                nodes: [],
                                routineVersionId: "deepRoutineVersion",
                                startNodeIndexes: [],
                            }],
                        }],
                    }],
                }, {
                    ...rootStep.nodes[2],
                }],
            });
        });
    });

    describe("singleRoutineToStep", () => {
        it("should convert a RoutineVersion to a SingleRoutineStep", () => {
            const routineVersion: RunnableRoutineVersion = {
                __typename: "RoutineVersion",
                id: uuid(),
                created_at: new Date().toISOString(),
                complexity: 2,
                configCallData: "{}",
                configFormInput: "{}",
                configFormOutput: "{}",
                inputs: [],
                nodeLinks: [],
                nodes: [],
                outputs: [],
                root: {
                    __typename: "Routine",
                    id: uuid(),
                } as Routine,
                routineType: RoutineType.Informational,
                translations: [{
                    __typename: "RoutineVersionTranslation",
                    id: uuid(),
                    language: "en",
                    name: "english name",
                    description: "english description",
                }, {
                    __typename: "RoutineVersionTranslation",
                    id: uuid(),
                    language: "es",
                    name: "spanish name",
                    description: "spanish description",
                }],
                versionLabel: "1.0.0",
                you: {} as RoutineVersionYou,
            };
            const englishResult = new RunStepBuilder(["en"], logger).singleRoutineToStep(routineVersion, [1]);
            expect(englishResult).toEqual({
                __type: RunStepType.SingleRoutine,
                description: "english description",
                name: "english name",
                location: [1],
                routineVersion,
            });
            const spanishResult = new RunStepBuilder(["es"], logger).singleRoutineToStep(routineVersion, [1]);
            expect(spanishResult).toEqual({
                __type: RunStepType.SingleRoutine,
                description: "spanish description",
                name: "spanish name",
                location: [1],
                routineVersion,
            });
            const secondLanguageResult = new RunStepBuilder(["fr", "es"], logger).singleRoutineToStep(routineVersion, [1]);
            expect(secondLanguageResult).toEqual({
                __type: RunStepType.SingleRoutine,
                description: "spanish description",
                name: "spanish name",
                location: [1],
                routineVersion,
            });
        });
    });

    describe("projectToStep", () => {
        it("should convert a ProjectVersion to a DirectoryStep", () => {
            const projectVersion: RunnableProjectVersion = {
                __typename: "ProjectVersion",
                id: uuid(),
                created_at: new Date().toISOString(),
                directories: [{
                    __typename: "ProjectVersionDirectory",
                    id: uuid(),
                    created_at: new Date().toISOString(),
                    updated_at: new Date().toISOString(),
                    isRoot: false,
                    translations: [{
                        __typename: "ProjectVersionDirectoryTranslation",
                        id: uuid(),
                        language: "en",
                        description: "directory description",
                        name: "directory name",
                    }],
                } as ProjectVersionDirectory],
                root: {
                    __typename: "Project",
                    id: uuid(),
                } as Project,
                translations: [{
                    __typename: "ProjectVersionTranslation",
                    id: uuid(),
                    language: "en",
                    description: "english description",
                    name: "english name",
                }],
                versionLabel: "1.2.3",
                you: {} as ProjectVersionYou,
            };
            const result = new RunStepBuilder(languages, logger).projectToStep(projectVersion, [1, 2]);
            expect(result).toEqual({
                __type: RunStepType.Directory,
                description: "english description",
                name: "english name",
                location: [1, 2],
                directoryId: null,
                hasBeenQueried: true,
                isOrdered: false,
                isRoot: true,
                projectVersionId: projectVersion.id,
                steps: [{
                    __type: RunStepType.Directory,
                    description: "directory description",
                    location: [1, 2, 1],
                    name: "directory name",
                    directoryId: (projectVersion as any).directories[0].id,
                    hasBeenQueried: true,
                    isOrdered: false,
                    isRoot: false,
                    projectVersionId: projectVersion.id,
                    steps: [],
                }],
            });
        });
    });

    describe("directoryToStep", () => {
        it("should convert a ProjectVersionDirectory to a DirectoryStep", () => {
            const projectVersionDirectory: ProjectVersionDirectory = {
                __typename: "ProjectVersionDirectory",
                id: uuid(),
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                isRoot: false,
                projectVersion: {
                    __typename: "ProjectVersion",
                    id: uuid(),
                } as ProjectVersion,
                translations: [{
                    __typename: "ProjectVersionDirectoryTranslation",
                    id: uuid(),
                    language: "en",
                    description: "directory description",
                    name: "directory name",
                }],
            } as ProjectVersionDirectory;
            const result1 = new RunStepBuilder(languages, logger).directoryToStep(projectVersionDirectory, [9, 3]);
            expect(result1).toEqual({
                __type: RunStepType.Directory,
                description: "directory description",
                location: [9, 3],
                name: "directory name",
                directoryId: projectVersionDirectory.id,
                hasBeenQueried: true,
                isOrdered: false,
                isRoot: false,
                projectVersionId: (projectVersionDirectory as any).projectVersion.id,
                steps: [],
            });
            delete projectVersionDirectory.projectVersion;
            const result2 = new RunStepBuilder(languages, logger).directoryToStep(projectVersionDirectory, [9, 3], "anotherProjectVersionId");
            expect(result2).toEqual({
                __type: RunStepType.Directory,
                description: "directory description",
                location: [9, 3],
                name: "directory name",
                directoryId: projectVersionDirectory.id,
                hasBeenQueried: true,
                isOrdered: false,
                isRoot: false,
                projectVersionId: "anotherProjectVersionId",
                steps: [],
            });
        });
    });

    describe("parseChildOrder", () => {
        beforeAll(() => {
            // eslint-disable-next-line @typescript-eslint/no-empty-function
            jest.spyOn(console, "warn").mockImplementation(() => { });
        });
        afterAll(() => {
            jest.restoreAllMocks();
        });

        // It should work with both uuids and alphanumeric strings
        const id1 = uuid();
        const id2 = "123";
        const id3 = uuid();
        const id4 = "9ahqqh9AGZ";
        const id5 = uuid();
        const id6 = "88201376AB1IUO96883667654";

        describe("without sides", () => {
            it("comma-separated string", () => {
                const input = `${id1},${id2},${id3},${id4}`;
                const expected = [id1, id2, id3, id4];
                expect(new RunStepBuilder(languages, logger).parseChildOrder(input)).toEqual(expected);
            });
            it("space and comma-separated string", () => {
                const input = `${id1},${id2} ${id3}, ${id4},,${id5}`;
                const expected = [id1, id2, id3, id4, id5];
                expect(new RunStepBuilder(languages, logger).parseChildOrder(input)).toEqual(expected);
            });
            it("single UUID", () => {
                const input = uuid();
                const expected = [input];
                expect(new RunStepBuilder(languages, logger).parseChildOrder(input)).toEqual(expected);
            });
            it("single alphanumeric string", () => {
                const input = "123abCD";
                const expected = [input];
                expect(new RunStepBuilder(languages, logger).parseChildOrder(input)).toEqual(expected);
            });
        });

        describe("with left and right sides", () => {
            it("comma delimiter", () => {
                const input = `l(${id1},${id2},${id3}),r(${id4},${id5},${id6})`;
                const expected = [id1, id2, id3, id4, id5, id6];
                expect(new RunStepBuilder(languages, logger).parseChildOrder(input)).toEqual(expected);
            });
            it("comma and space delimiter", () => {
                const input = `l(${id1},${id2},${id3}), r(${id4},${id5},${id6})`;
                const expected = [id1, id2, id3, id4, id5, id6];
                expect(new RunStepBuilder(languages, logger).parseChildOrder(input)).toEqual(expected);
            });
            it("spaces delimiter", () => {
                const input = `l(${id1},${id2},${id3})   r(${id4},${id5},${id6})`;
                const expected = [id1, id2, id3, id4, id5, id6];
                expect(new RunStepBuilder(languages, logger).parseChildOrder(input)).toEqual(expected);
            });
            it("no delimiter", () => {
                const input = `l(${id1},${id2},${id3})r(${id4},${id5},${id6})`;
                const expected = [id1, id2, id3, id4, id5, id6];
                expect(new RunStepBuilder(languages, logger).parseChildOrder(input)).toEqual(expected);
            });
        });

        it("should return an empty array for an empty string", () => {
            const input = "";
            const expected: string[] = [];
            expect(new RunStepBuilder(languages, logger).parseChildOrder(input)).toEqual(expected);
        });

        it("should parse root format with empty left order correctly", () => {
            const input = `l(),r(${id4},${id2},${id6})`;
            const expected = [id4, id2, id6];
            expect(new RunStepBuilder(languages, logger).parseChildOrder(input)).toEqual(expected);
        });

        it("should parse root format with empty right order correctly", () => {
            const input = `l(${id1},${id2},${id6}),r()`;
            const expected = [id1, id2, id6];
            expect(new RunStepBuilder(languages, logger).parseChildOrder(input)).toEqual(expected);
        });

        it("should return an empty array for root format with both orders empty", () => {
            const input = "l(),r()";
            const expected: string[] = [];
            expect(new RunStepBuilder(languages, logger).parseChildOrder(input)).toEqual(expected);
        });

        describe("should handle malformed input string", () => {
            it("no closing parenthesis", () => {
                const input = "l(123,456,789";
                const expected = [];
                expect(new RunStepBuilder(languages, logger).parseChildOrder(input)).toEqual(expected);
            });
            it("invalid character", () => {
                const input = "23423_3242";
                const expected = [];
                expect(new RunStepBuilder(languages, logger).parseChildOrder(input)).toEqual(expected);
            });
            it("characters after closing parenthesis", () => {
                const input = "l(123)2342";
                const expected = [];
                expect(new RunStepBuilder(languages, logger).parseChildOrder(input)).toEqual(expected);
            });
            it("nested parentheses", () => {
                const input = "l(333,(222,111),555),r(888,123,321)";
                const expected = [];
                expect(new RunStepBuilder(languages, logger).parseChildOrder(input)).toEqual(expected);
            });
        });
    });

    describe("sortStepsAndAddDecisions", () => {
        it("doesn't sort if start step missing", () => {
            const steps = [
                {
                    __type: RunStepType.End,
                    description: "",
                    location: [1, 3],
                    name: "",
                    nextLocation: null,
                    nodeId: "1",
                    wasSuccessful: true,
                } as EndStep,
            ];
            const nodeLinks = [] as NodeLink[];
            const result = new RunStepBuilder(languages, logger).sortStepsAndAddDecisions(steps, nodeLinks);
            expect(result.sorted).toEqual(steps); // Result should be unchanged
            expect(result.startNodeIndexes).toEqual([]); // No start node
        });

        it("should sort steps in correct order for a linear path", () => {
            const commonProps = { description: "", name: "", location: [69] };
            const steps = [
                {
                    __type: RunStepType.Start,
                    nextLocation: [],
                    nodeId: "1",
                    ...commonProps,
                } as StartStep,
                {
                    __type: RunStepType.RoutineList,
                    isOrdered: true,
                    parentRoutineVersionId: "routineVersion123",
                    nextLocation: [],
                    nodeId: "2",
                    steps: [],
                    ...commonProps,
                } as RoutineListStep,
                {
                    __type: RunStepType.End,
                    nodeId: "3",
                    ...commonProps,
                } as EndStep,
            ];
            const nodeLinks = [
                { from: { id: "1" }, to: { id: "2" } },
                { from: { id: "2" }, to: { id: "3" } },
            ] as NodeLink[];
            const result = new RunStepBuilder(languages, logger).sortStepsAndAddDecisions(steps, nodeLinks);
            expect(result.sorted.map(step => (step as { nodeId: string | null }).nodeId)).toEqual(["1", "2", "3"]);
            expect(result.startNodeIndexes).toEqual([0]);
        });

        it("should add a decision step when there are multiple outgoing links", () => {
            const commonProps = { description: "", name: "", location: [1, 3, 2, 1, 69] };
            const steps = [
                {
                    __type: RunStepType.Start,
                    nextLocation: [],
                    nodeId: "1",
                    ...commonProps,
                } as StartStep,
                {
                    __type: RunStepType.RoutineList,
                    isOrdered: true,
                    parentRoutineVersionId: "routineVersion123",
                    nextLocation: [],
                    nodeId: "2",
                    steps: [],
                    ...commonProps,
                } as RoutineListStep,
                {
                    __type: RunStepType.RoutineList,
                    isOrdered: true,
                    parentRoutineVersionId: "routineVersion123",
                    nextLocation: [],
                    nodeId: "3",
                    steps: [],
                    ...commonProps,
                } as RoutineListStep,
                {
                    __type: RunStepType.End,
                    nodeId: "4",
                    ...commonProps,
                } as EndStep,
            ];
            const nodeLinks = [
                // Start to both RoutineLists
                { from: { id: "1" }, to: { id: "2" } },
                { from: { id: "1" }, to: { id: "3" } },
                // RoutineLists to End
                { from: { id: "2" }, to: { id: "4" } },
                { from: { id: "3" }, to: { id: "4" } },
            ] as NodeLink[];
            const result = new RunStepBuilder(languages, logger).sortStepsAndAddDecisions(steps, nodeLinks);
            expectValidStepSequence(result.sorted, nodeLinks);
            expect(result.sorted.length).toBe(steps.length + 1); // One decision step
            expect(result.startNodeIndexes).toEqual([0]);
        });

        it("should handle cycles in the graph", () => {
            const commonProps = { description: "", name: "", location: [1, 3, 2, 420] };
            const steps = [
                {
                    __type: RunStepType.Start,
                    nextLocation: [],
                    nodeId: "1",
                    ...commonProps,
                } as StartStep,
                {
                    __type: RunStepType.RoutineList,
                    isOrdered: true,
                    parentRoutineVersionId: "routineVersion123",
                    nextLocation: [],
                    nodeId: "2",
                    steps: [],
                    ...commonProps,
                } as RoutineListStep,
                {
                    __type: RunStepType.RoutineList,
                    isOrdered: true,
                    parentRoutineVersionId: "routineVersion123",
                    nextLocation: [],
                    nodeId: "3",
                    steps: [],
                    ...commonProps,
                } as RoutineListStep,
                {
                    __type: RunStepType.RoutineList,
                    isOrdered: true,
                    parentRoutineVersionId: "routineVersion123",
                    nextLocation: [],
                    nodeId: "4",
                    steps: [],
                    ...commonProps,
                } as RoutineListStep,
                {
                    __type: RunStepType.End,
                    nodeId: "5",
                    ...commonProps,
                } as EndStep,
            ];
            const nodeLinks = [
                // Start to first RoutineList
                { from: { id: "1" }, to: { id: "2" } },
                // First RoutineList to next two RoutineLists
                { from: { id: "2" }, to: { id: "3" } },
                { from: { id: "2" }, to: { id: "4" } },
                // Second RoutineList to End
                { from: { id: "3" }, to: { id: "5" } },
                // Third RoutineList to first RoutineList
                { from: { id: "4" }, to: { id: "2" } },
            ] as NodeLink[];
            const result = new RunStepBuilder(languages, logger).sortStepsAndAddDecisions(steps, nodeLinks);
            expectValidStepSequence(result.sorted, nodeLinks);
            expect(result.sorted.length).toBe(steps.length + 1); // One decision step
            expect(result.startNodeIndexes).toEqual([0]);
        });

        it("should handle complex graphs with multiple decision points", () => {
            const commonProps = { description: "", name: "", location: [1, 3] };
            const steps = [
                {
                    __type: RunStepType.Start,
                    nextLocation: [],
                    nodeId: "1",
                    ...commonProps,
                } as StartStep,
                {
                    __type: RunStepType.RoutineList,
                    isOrdered: true,
                    parentRoutineVersionId: "routineVersion123",
                    nextLocation: [],
                    nodeId: "2",
                    steps: [],
                    ...commonProps,
                } as RoutineListStep,
                {
                    __type: RunStepType.RoutineList,
                    isOrdered: true,
                    parentRoutineVersionId: "routineVersion123",
                    nextLocation: [],
                    nodeId: "3",
                    steps: [],
                    ...commonProps,
                } as RoutineListStep,
                {
                    __type: RunStepType.RoutineList,
                    isOrdered: true,
                    parentRoutineVersionId: "routineVersion123",
                    nextLocation: [],
                    nodeId: "4",
                    steps: [],
                    ...commonProps,
                } as RoutineListStep,
                {
                    __type: RunStepType.End,
                    nodeId: "5",
                    ...commonProps,
                } as EndStep,
                {
                    __type: RunStepType.RoutineList,
                    isOrdered: true,
                    parentRoutineVersionId: "routineVersion123",
                    nextLocation: [],
                    nodeId: "6",
                    steps: [],
                    ...commonProps,
                } as RoutineListStep,
                {
                    __type: RunStepType.End,
                    nodeId: "7",
                    ...commonProps,
                } as EndStep,
            ];
            const nodeLinks = [
                // Start to first three RoutineLists
                { from: { id: "1" }, to: { id: "2" } },
                { from: { id: "1" }, to: { id: "3" } },
                { from: { id: "1" }, to: { id: "4" } },
                // First RoutineList points to second two
                { from: { id: "2" }, to: { id: "3" } },
                { from: { id: "2" }, to: { id: "4" } },
                // Second RoutineList points to first RoutineList and first End
                { from: { id: "3" }, to: { id: "2" } },
                { from: { id: "3" }, to: { id: "5" } },
                // Third RoutineList points to first and fourth RoutineLists
                { from: { id: "4" }, to: { id: "2" } },
                { from: { id: "4" }, to: { id: "6" } },
                // Fourth RoutineList points to second End
                { from: { id: "6" }, to: { id: "7" } },
            ] as NodeLink[];
            const result = new RunStepBuilder(languages, logger).sortStepsAndAddDecisions(steps, nodeLinks);
            expectValidStepSequence(result.sorted, nodeLinks);
            expect(result.sorted.length).toBe(steps.length + 4); // Four decision steps
            expect(result.startNodeIndexes).toEqual([0]);
        });

        it("should ignore unlinked steps", () => {
            const commonProps = { description: "", name: "", location: [1, 3] };
            const steps = [
                {
                    __type: RunStepType.Start,
                    nextLocation: [],
                    nodeId: "1",
                    ...commonProps,
                } as StartStep,
                {
                    __type: RunStepType.RoutineList,
                    isOrdered: true,
                    parentRoutineVersionId: "routineVersion123",
                    nextLocation: [],
                    nodeId: "2",
                    steps: [],
                    ...commonProps,
                } as RoutineListStep,
                {
                    __type: RunStepType.End,
                    nodeId: "3",
                    ...commonProps,
                } as EndStep,
                {
                    __type: RunStepType.RoutineList,
                    isOrdered: true,
                    parentRoutineVersionId: "routineVersion123",
                    nextLocation: [],
                    nodeId: "4",
                    steps: [],
                    ...commonProps,
                } as RoutineListStep,
            ];
            const nodeLinks = [
                // Start to first RoutineList
                { from: { id: "1" }, to: { id: "2" } },
                // First RoutineList to End
                { from: { id: "2" }, to: { id: "3" } },
                // Second RoutineList has no links
            ] as NodeLink[];
            const result = new RunStepBuilder(languages, logger).sortStepsAndAddDecisions(steps, nodeLinks);
            expect(result.sorted.map(step => (step as { nodeId: string | null }).nodeId)).toEqual(["1", "2", "3"]);
            expect(result.startNodeIndexes).toEqual([0]);
        });

        it("should update the location of steps within a RoutineListStep", () => {
            const baseLocation = [69, 420];
            const commonProps = { description: "", name: "" };
            const steps = [
                {
                    __type: RunStepType.Start,
                    location: [...baseLocation, 44444],
                    nextLocation: [],
                    nodeId: "1",
                    ...commonProps,
                } as StartStep,
                {
                    __type: RunStepType.RoutineList,
                    isOrdered: true,
                    location: [...baseLocation, 23],
                    parentRoutineVersionId: "routineVersion123",
                    nextLocation: [],
                    nodeId: "2",
                    steps: [{
                        __type: RunStepType.SingleRoutine,
                        description: "",
                        name: "",
                        location: [5, 3],
                        routineVersion: { complexity: 2, id: "1", routineType: RoutineType.MultiStep } as RoutineVersion,
                    }, {
                        __type: RunStepType.SingleRoutine,
                        description: "",
                        name: "",
                        location: [5, 4],
                        routineVersion: { complexity: 3, id: "2", routineType: RoutineType.MultiStep } as RoutineVersion,
                    }],
                    ...commonProps,
                } as RoutineListStep,
                {
                    __type: RunStepType.End,
                    location: [...baseLocation, 999999],
                    nodeId: "3",
                    ...commonProps,
                } as EndStep,
            ];
            const nodeLinks = [
                { from: { id: "1" }, to: { id: "2" } },
                { from: { id: "2" }, to: { id: "3" } },
            ] as NodeLink[];
            const result = new RunStepBuilder(languages, logger).sortStepsAndAddDecisions(steps, nodeLinks);
            const routineListStep = result.sorted.find(step => step.__type === RunStepType.RoutineList) as RoutineListStep;
            expect(routineListStep.steps[0].location).toEqual([...routineListStep.location, 1]);
            expect(routineListStep.steps[1].location).toEqual([...routineListStep.location, 2]);
            expect(result.startNodeIndexes).toEqual([0]);
        });
    });

    describe("insertStep", () => {
        beforeAll(() => {
            // eslint-disable-next-line @typescript-eslint/no-empty-function
            jest.spyOn(console, "warn").mockImplementation(() => { });
        });
        afterAll(() => {
            jest.restoreAllMocks();
        });

        it("adds project data to a project", () => {
            const mockStepData: DirectoryStep = {
                __type: RunStepType.Directory,
                description: "a",
                directoryId: null,
                hasBeenQueried: true,
                isOrdered: false,
                isRoot: true,
                location: [1, 2, 3],
                name: "a",
                projectVersionId: "projectVersion123",
                steps: [],
            };
            const mockRootStep: DirectoryStep = {
                __type: RunStepType.Directory,
                description: "root",
                directoryId: null,
                hasBeenQueried: true,
                isOrdered: false,
                isRoot: true,
                location: [1],
                name: "root",
                projectVersionId: "projectVersion234",
                steps: [{
                    __type: RunStepType.Directory,
                    description: "b",
                    directoryId: null,
                    hasBeenQueried: false,
                    isOrdered: false,
                    isRoot: true,
                    location: [1, 1],
                    name: "b",
                    projectVersionId: "projectVersion123", // Matches mockStepData, so it should be replaced by mockStepData
                    steps: [],
                }],
            };
            const result = new RunStepBuilder(languages, logger).insertStep(mockStepData, mockRootStep);
            expect((result as DirectoryStep).steps[0]).toEqual({ ...mockStepData, location: [1, 1] });
        });

        it("adds multi-step routine data to a multi-step routine", () => {
            const mockStepData: MultiRoutineStep = {
                __type: RunStepType.MultiRoutine,
                description: "newBoi",
                location: [3, 3, 2],
                name: "hello",
                nodes: [],
                nodeLinks: [],
                routineVersionId: "routineVersion999",
                startNodeIndexes: [],
            };
            const mockRootStep: MultiRoutineStep = {
                __type: RunStepType.MultiRoutine,
                description: "root",
                name: "root",
                location: [1],
                nodes: [{
                    __type: RunStepType.Start,
                    description: "",
                    location: [1, 1],
                    name: "",
                    nextLocation: [1, 2],
                    nodeId: "node123",
                }, {
                    __type: RunStepType.RoutineList,
                    description: "boop",
                    isOrdered: false,
                    location: [1, 2],
                    name: "beep",
                    nextLocation: [1, 3],
                    nodeId: "node234",
                    parentRoutineVersionId: "routine123",
                    steps: [{
                        __type: RunStepType.SingleRoutine,
                        description: "",
                        name: "",
                        location: [1, 2, 1],
                        routineVersion: { id: "routineVersion999" } as RoutineVersion, // Matches mockStepData, so it should be replaced by mockStepData
                    }],
                }, {
                    __type: RunStepType.End,
                    description: "",
                    name: "",
                    nextLocation: null,
                    location: [1, 3],
                    nodeId: "node345",
                    wasSuccessful: true,
                }],
                nodeLinks: [],
                routineVersionId: "routineVersion123",
                startNodeIndexes: [0],
            };
            const result = new RunStepBuilder(languages, logger).insertStep(mockStepData, mockRootStep);
            expect(((result as MultiRoutineStep).nodes[1] as RoutineListStep).steps[0]).toEqual({ ...mockStepData, location: [1, 2, 1] });
        });

        it("does not add data to a root that's a subroutine", () => {
            const mockStepData: MultiRoutineStep = {
                __type: RunStepType.MultiRoutine,
                description: "newBoi",
                location: [999],
                name: "hello",
                nodes: [],
                nodeLinks: [],
                routineVersionId: "routineVersion999",
                startNodeIndexes: [],
            };
            const mockRootStep: SingleRoutineStep = {
                __type: RunStepType.SingleRoutine,
                description: "root",
                location: [1],
                name: "root",
                routineVersion: { id: "routineVersion529" } as RoutineVersion,
            };
            const result = new RunStepBuilder(languages, logger).insertStep(mockStepData, mockRootStep);
            expect(result).toEqual({ ...mockRootStep, location: [1] });
        });

        it("adds project data to a deeply-nested project", () => {
            const mockStepData: DirectoryStep = {
                __type: RunStepType.Directory,
                description: "new data",
                directoryId: null,
                hasBeenQueried: false,
                isOrdered: false,
                isRoot: true,
                location: [],
                name: "new data",
                projectVersionId: "projectVersionNested",
                steps: [],
            };

            const mockRootStep: DirectoryStep = {
                __type: RunStepType.Directory,
                description: "root",
                directoryId: null,
                hasBeenQueried: true,
                isOrdered: false,
                isRoot: true,
                location: [1],
                name: "root",
                projectVersionId: "rootProject",
                steps: [{
                    __type: RunStepType.Directory,
                    description: "nested level 1",
                    directoryId: null,
                    hasBeenQueried: true,
                    isOrdered: false,
                    isRoot: false,
                    location: [1, 1],
                    name: "nested level 1",
                    projectVersionId: "level1Project",
                    steps: [{
                        __type: RunStepType.Directory,
                        description: "nested level 2",
                        directoryId: null,
                        hasBeenQueried: true,
                        isOrdered: false,
                        isRoot: false,
                        location: [1, 1, 1],
                        name: "nested level 2",
                        projectVersionId: "level2Project",
                        steps: [{
                            __type: RunStepType.Directory,
                            description: "nested level 3",
                            directoryId: null,
                            hasBeenQueried: true,
                            isOrdered: false,
                            isRoot: false,
                            location: [1, 1, 1, 1],
                            name: "nested level 3",
                            projectVersionId: "projectVersionNested", // Matches mockStepData
                            steps: [],
                        }],
                    }],
                }],
            };

            const result = new RunStepBuilder(languages, logger).insertStep(mockStepData, mockRootStep);

            // Traversing down to the deeply-nested step
            expect((((result as DirectoryStep).steps[0] as DirectoryStep).steps[0] as DirectoryStep).steps[0]).toEqual({ ...mockStepData, location: [1, 1, 1, 1] });
        });

        it("adds routine data to a deeply-nested routine", () => {
            const mockStepData: MultiRoutineStep = {
                __type: RunStepType.MultiRoutine,
                description: "deep routine",
                location: [3, 3, 3, 3, 3],
                name: "deep routine",
                nodes: [],
                nodeLinks: [],
                routineVersionId: "deepRoutineVersion",
                startNodeIndexes: [],
            };

            const mockRootStep: MultiRoutineStep = {
                __type: RunStepType.MultiRoutine,
                description: "root",
                location: [1],
                name: "root",
                nodes: [{
                    __type: RunStepType.Start,
                    description: "start",
                    location: [1, 1],
                    name: "start",
                    nextLocation: [8, 33, 3],
                    nodeId: "startNode",
                }, {
                    __type: RunStepType.RoutineList,
                    description: "routine list level 1",
                    isOrdered: false,
                    location: [1, 2],
                    name: "routine list level 1",
                    nextLocation: [],
                    nodeId: "routineListNode1",
                    parentRoutineVersionId: "routine123",
                    steps: [{
                        __type: RunStepType.MultiRoutine,
                        description: "multi routine level 2",
                        location: [1, 2, 1],
                        name: "multi routine level 2",
                        nodes: [{
                            __type: RunStepType.RoutineList,
                            description: "routine list level 3",
                            isOrdered: false,
                            name: "routine list level 3",
                            nextLocation: [],
                            location: [1, 2, 1, 1],
                            nodeId: "routineListNode3",
                            parentRoutineVersionId: "routine123",
                            steps: [{
                                __type: RunStepType.SingleRoutine,
                                description: "subroutine",
                                location: [1, 2, 1, 1, 1],
                                name: "subroutine",
                                routineVersion: { id: "deepRoutineVersion" } as any, // Mock routineVersion
                            }],
                        }],
                        nodeLinks: [],
                        routineVersionId: "level2Routine",
                        startNodeIndexes: [],
                    }],
                }, {
                    __type: RunStepType.End,
                    description: "end",
                    location: [1, 3],
                    name: "end",
                    nextLocation: null,
                    nodeId: "endNode",
                    wasSuccessful: false,
                }],
                nodeLinks: [],
                routineVersionId: "rootRoutine",
                startNodeIndexes: [0],
            };

            const result = new RunStepBuilder(languages, logger).insertStep(mockStepData, mockRootStep);

            // Traversing down to the deeply-nested step
            expect(((((result as MultiRoutineStep).nodes[1] as RoutineListStep).steps[0] as MultiRoutineStep).nodes[0] as RoutineListStep).steps[0]).toEqual({ ...mockStepData, location: [1, 2, 1, 1, 1] });
        });

        it("does not change the root if the data doesn't match", () => {
            const mockStepData: MultiRoutineStep = {
                __type: RunStepType.MultiRoutine,
                description: "non-matching data",
                location: [4, 3, 1, 3],
                name: "non-matching data",
                nodes: [],
                nodeLinks: [],
                routineVersionId: "nonMatchingRoutineVersion",
                startNodeIndexes: [],
            };

            const mockRootStep: MultiRoutineStep = {
                __type: RunStepType.MultiRoutine,
                description: "root",
                location: [1],
                name: "root",
                nodes: [{
                    __type: RunStepType.Start,
                    description: "start",
                    location: [1, 1],
                    name: "start",
                    nextLocation: [1, 2],
                    nodeId: "startNode",
                }, {
                    __type: RunStepType.RoutineList,
                    description: "routine list",
                    isOrdered: false,
                    location: [1, 2],
                    name: "routine list",
                    nextLocation: [],
                    nodeId: "routineListNode",
                    parentRoutineVersionId: "routine123",
                    steps: [{
                        __type: RunStepType.SingleRoutine,
                        description: "subroutine",
                        location: [1, 2, 1],
                        name: "subroutine",
                        routineVersion: { id: "routine333" } as RoutineVersion,
                    }],
                }],
                nodeLinks: [],
                routineVersionId: "rootRoutine",
                startNodeIndexes: [0],
            };

            const result = new RunStepBuilder(languages, logger).insertStep(mockStepData, mockRootStep);
            expect(result).toEqual(mockRootStep);
        });
    });
});

describe("saveRunProgress", () => {
    // Common setup
    const mockLogger = {
        error: jest.fn(),
        warn: jest.fn(),
        info: jest.fn(),
        debug: jest.fn(),
    };

    const mockHandleRunProjectUpdate = jest.fn();
    const mockHandleRunRoutineUpdate = jest.fn();

    beforeEach(() => {
        jest.clearAllMocks();
    });

    it("should update existing step data for a RunProject", async () => {
        const mockRun = {
            id: "run-1",
            __typename: "RunProject",
            contextSwitches: 2, // Should be overwritten by sum of all steps
            steps: [{
                id: "step-1", timeElapsed: 100, contextSwitches: 5,
            }, {
                id: "step-2", timeElapsed: 23, contextSwitches: 1,
            }],
            timeElapsed: 123, // Should be overwritten by sum of all steps
        } as RunProject;

        const mockCurrentStep = { name: "Test Step", location: [0] } as RunStep;
        const mockCurrentStepRunData = { id: "step-1" } as RunProjectStep;
        const mockRunnableObject = { __typename: "ProjectVersion" } as ProjectVersion;

        await saveRunProgress({
            contextSwitches: 7, // Not cumulative, and for step only. Overall context switches are sum of all steps
            currentStep: mockCurrentStep,
            currentStepOrder: 1,
            currentStepRunData: mockCurrentStepRunData,
            formData: {},
            handleRunProjectUpdate: mockHandleRunProjectUpdate,
            handleRunRoutineUpdate: mockHandleRunRoutineUpdate,
            isStepCompleted: false,
            isRunCompleted: false,
            logger: mockLogger,
            run: mockRun,
            runnableObject: mockRunnableObject,
            timeElapsed: 200, // Not cumulative, and for step only. Overall time elapsed is sum of all steps
        });

        expect(mockHandleRunProjectUpdate).toHaveBeenCalledWith({
            id: "run-1",
            stepsUpdate: [{
                id: "step-1",
                timeElapsed: 200,
                contextSwitches: 7,
                status: RunRoutineStepStatus.InProgress, // Reflects `isStepCompleted` false
            }], // No update for step-2, but its timeElapsed and contextSwitches should be added to the RunProject
            timeElapsed: 223, // Sum of all steps' timeElapsed
            contextSwitches: 8, // Sum of all steps' contextSwitches
            status: RunStatus.InProgress, // Reflects `isRunCompleted` false
        });
    });

    it("should create new step data for a RunProject with completed step", async () => {
        const mockRun = {
            id: "run-2",
            __typename: "RunProject",
            contextSwitches: 0,
            steps: [{
                id: "step-1", timeElapsed: 420, contextSwitches: 5,
            }, {
                id: "step-2", timeElapsed: 69, contextSwitches: 1,
            }],
            timeElapsed: 0,
        } as unknown as RunProject;

        const mockCurrentStep = { name: "New Step", location: [1], nodeId: "nodeBoop" } as RunStep;
        const mockRunnableObject = { __typename: "ProjectVersion" } as ProjectVersion;

        await saveRunProgress({
            contextSwitches: 3,
            currentStep: mockCurrentStep,
            currentStepOrder: 1,
            currentStepRunData: null,
            formData: {},
            handleRunProjectUpdate: mockHandleRunProjectUpdate,
            handleRunRoutineUpdate: mockHandleRunRoutineUpdate,
            isStepCompleted: true,
            isRunCompleted: false,
            logger: mockLogger,
            run: mockRun,
            runnableObject: mockRunnableObject,
            timeElapsed: 150,
        });

        expect(mockHandleRunProjectUpdate).toHaveBeenCalledWith({
            id: "run-2",
            stepsCreate: [{
                id: expect.any(String),
                name: "New Step",
                nodeConnect: "nodeBoop",
                order: 1,
                step: [1],
                runProjectConnect: "run-2",
                timeElapsed: 150,
                contextSwitches: 3,
                status: RunRoutineStepStatus.Completed, // Reflects `isStepCompleted` true
                subroutineConnect: undefined,
            }],
            timeElapsed: 639, // 420 + 69 + 150
            contextSwitches: 9, // 5 + 1 + 3
            status: RunStatus.InProgress, // Reflects `isRunCompleted` false
        });
    });

    it("should update existing step data for a RunRoutine with completed run", async () => {
        const mockRun = {
            id: "run-3",
            __typename: "RunRoutine",
            contextSwitches: 5, // Should be overwritten by sum of all steps
            steps: [{
                id: "step-1", timeElapsed: 200, contextSwitches: 5,
            }, {
                id: "step-2", timeElapsed: 100, contextSwitches: 2,
            }],
            timeElapsed: 300, // Should be overwritten by sum of all steps
            inputs: [],
            outputs: [],
        } as unknown as RunRoutine;

        const mockCurrentStep = { name: "Final Step", location: [2] } as RunStep;
        const mockCurrentStepRunData = { id: "step-2" } as RunRoutineStep;
        const mockRunnableObject = {
            __typename: "RoutineVersion",
            inputs: [],
            outputs: [],
        } as unknown as RoutineVersion;

        await saveRunProgress({
            contextSwitches: 3, // Not cumulative, and for step only. Overall context switches are sum of all steps
            currentStep: mockCurrentStep,
            currentStepOrder: 2,
            currentStepRunData: mockCurrentStepRunData,
            formData: {},
            handleRunProjectUpdate: mockHandleRunProjectUpdate,
            handleRunRoutineUpdate: mockHandleRunRoutineUpdate,
            isStepCompleted: true,
            isRunCompleted: true,
            logger: mockLogger,
            run: mockRun,
            runnableObject: mockRunnableObject,
            timeElapsed: 150, // Not cumulative, and for step only. Overall time elapsed is sum of all steps
        });

        expect(mockHandleRunRoutineUpdate).toHaveBeenCalledWith({
            id: "run-3",
            stepsUpdate: [{
                id: "step-2",
                timeElapsed: 150,
                contextSwitches: 3,
                status: RunRoutineStepStatus.Completed, // Reflects `isStepCompleted` true
            }], // No update for step-1, but its timeElapsed and contextSwitches should be added to the RunRoutine
            timeElapsed: 350, // Sum of all steps' timeElapsed (200 + 150)
            contextSwitches: 8, // Sum of all steps' contextSwitches (5 + 3)
            status: RunStatus.Completed, // Reflects `isRunCompleted` true
        });
    });

    it("should create new step data for a RunRoutine and handle inputs/outputs", async () => {
        const mockRun = {
            id: "run-4",
            __typename: "RunRoutine",
            steps: [{
                id: "step-1", timeElapsed: 50, contextSwitches: 2,
            }],
            inputs: [{
                id: "existing-input-1",
                data: "\"oldInputValue\"",
                input: { id: "input-1", name: "existingInput" },
            }],
            outputs: [{
                id: "existing-output-1",
                data: "\"oldOutputValue\"",
                output: { id: "output-1", name: "existingOutput" },
            }],
            contextSwitches: 2, // Should be overwritten by sum of all steps
            timeElapsed: 50, // Should be overwritten by sum of all steps
        } as unknown as RunRoutine;

        const mockCurrentStep = { name: "New Step", location: [1], routineVersion: { id: "subroutineId" } } as RunStep;
        const mockRunnableObject = {
            __typename: "RoutineVersion",
            inputs: [
                { id: "input-1", name: "existingInput" },
                { id: "input-2", name: "newInput" },
            ],
            outputs: [
                { id: "output-1", name: "existingOutput" },
                { id: "output-2", name: "newOutput" },
            ],
        } as RoutineVersion;

        const formData = {
            "input-existingInput": "updatedInputValue",
            "input-newInput": "newInputValue",
            "output-existingOutput": "updatedOutputValue",
            "output-newOutput": "newOutputValue",
        };


        await saveRunProgress({
            contextSwitches: 1, // Not cumulative, and for step only. Overall context switches are sum of all steps
            currentStep: mockCurrentStep,
            currentStepOrder: 2,
            currentStepRunData: null,
            formData,
            handleRunProjectUpdate: mockHandleRunProjectUpdate,
            handleRunRoutineUpdate: mockHandleRunRoutineUpdate,
            isStepCompleted: false,
            isRunCompleted: false,
            logger: mockLogger,
            run: mockRun,
            runnableObject: mockRunnableObject,
            timeElapsed: 30, // Not cumulative, and for step only. Overall time elapsed is sum of all steps
        });

        expect(mockHandleRunRoutineUpdate).toHaveBeenCalledWith({
            id: "run-4",
            stepsCreate: [{
                id: expect.any(String),
                name: "New Step",
                nodeConnect: undefined,
                order: 2,
                step: [1],
                runRoutineConnect: "run-4",
                timeElapsed: 30,
                contextSwitches: 1,
                status: RunRoutineStepStatus.InProgress, // Reflects `isStepCompleted` false
                subroutineConnect: "subroutineId",
            }],
            timeElapsed: 80, // Sum of all steps' timeElapsed (50 + 30)
            contextSwitches: 3, // Sum of all steps' contextSwitches (2 + 1)
            status: RunStatus.InProgress, // Reflects `isRunCompleted` false
            inputsCreate: [{
                id: expect.any(String),
                data: "\"newInputValue\"",
                inputConnect: "input-2",
                runRoutineConnect: "run-4",
            }],
            inputsUpdate: [{
                id: "existing-input-1",
                data: "\"updatedInputValue\"",
            }],
            outputsCreate: [{
                id: expect.any(String),
                data: "\"newOutputValue\"",
                outputConnect: "output-2",
                runRoutineConnect: "run-4",
            }],
            outputsUpdate: [{
                id: "existing-output-1",
                data: "\"updatedOutputValue\"",
            }],
        });
    });

    it("should not include stepCreate or stepUpdate if currentStep and currentStepRunData are not provided", () => {
        const mockRun = {
            id: "run-5",
            __typename: "RunProject",
            contextSwitches: 0,
            steps: [{
                id: "step-1", timeElapsed: 420, contextSwitches: 5,
            }, {
                id: "step-2", timeElapsed: 69, contextSwitches: 1,
            }],
            timeElapsed: 0,
        } as unknown as RunProject;

        const mockRunnableObject = { __typename: "ProjectVersion" } as ProjectVersion;

        saveRunProgress({
            contextSwitches: 6, // 5 + 1
            currentStep: null,
            currentStepOrder: 99999, // No step data provided, so this should be ignored
            currentStepRunData: null,
            formData: {},
            handleRunProjectUpdate: mockHandleRunProjectUpdate,
            handleRunRoutineUpdate: mockHandleRunRoutineUpdate,
            isStepCompleted: true,
            isRunCompleted: false,
            logger: mockLogger,
            run: mockRun,
            runnableObject: mockRunnableObject,
            timeElapsed: 489, // 420 + 69
        });

        expect(mockHandleRunProjectUpdate).toHaveBeenCalledWith({
            id: "run-5",
            timeElapsed: 489,
            contextSwitches: 6,
            status: RunStatus.InProgress,
        });
    });
});

describe("Schema Parsing Functions", () => {
    describe("parseSchemaInput", () => {
        test("parses valid JSON string correctly", () => {
            const input = JSON.stringify({ containers: [{ id: 1 }], elements: [{ id: 1 }] });
            const expected = { containers: [{ id: 1 }], elements: [{ id: 1 }] };
            expect(parseSchemaInput(input, RoutineType.Informational, console)).toEqual(expected);
        });

        test("falls back to default input schema on invalid JSON string", () => {
            const input = "{ containers: [}";
            expect(parseSchemaInput(input, RoutineType.Api, console)).toEqual(defaultConfigFormInputMap[RoutineType.Api]());
        });

        test("handles non-string, non-object inputs by returning default input schema", () => {
            const input = 12345;
            expect(parseSchemaInput(input, RoutineType.Generate, console)).toEqual(defaultConfigFormInputMap[RoutineType.Generate]());
        });

        test("handles valid object input without parsing", () => {
            const input = { containers: [{ id: 2 }], elements: [{ id: 2 }] };
            expect(parseSchemaInput(input, RoutineType.SmartContract, console)).toEqual(input);
        });

        test("adds missing containers and elements as empty arrays", () => {
            const input = "{}";
            const expected = { containers: [], elements: [] };
            expect(parseSchemaInput(input, RoutineType.Code, console)).toEqual(expected);
        });

        test("replaces non-array containers and elements with empty arrays", () => {
            const input = JSON.stringify({ containers: "not-an-array", elements: "not-an-array" });
            const expected = { containers: [], elements: [] };
            expect(parseSchemaInput(input, RoutineType.Data, console)).toEqual(expected);
        });

        test("handles malformed object inputs", () => {
            const input = { someRandomKey: 123 };
            const expected = { containers: [], elements: [], someRandomKey: 123 };
            expect(parseSchemaInput(input, RoutineType.Informational, console)).toEqual(expected);
        });
    });

    describe("parseSchemaOutput", () => {
        test("parses valid JSON string correctly", () => {
            const input = JSON.stringify({ containers: [{ id: 1 }], elements: [{ id: 1 }] });
            const expected = { containers: [{ id: 1 }], elements: [{ id: 1 }] };
            expect(parseSchemaOutput(input, RoutineType.Generate, console)).toEqual(expected);
        });

        test("falls back to default output schema on invalid JSON string", () => {
            const input = "{ containers: [}";
            expect(parseSchemaOutput(input, RoutineType.Generate, console)).toEqual(defaultConfigFormOutputMap[RoutineType.Generate]());
        });

        test("handles non-string, non-object inputs by returning default output schema", () => {
            const input = 12345;
            expect(parseSchemaOutput(input, RoutineType.Action, console)).toEqual(defaultConfigFormOutputMap[RoutineType.Action]());
        });

        test("handles valid object input without parsing", () => {
            const input = { containers: [{ id: 2 }], elements: [{ id: 2 }] };
            expect(parseSchemaOutput(input, RoutineType.Data, console)).toEqual(input);
        });

        test("adds missing containers and elements as empty arrays", () => {
            const input = "{}";
            const expected = { containers: [], elements: [] };
            expect(parseSchemaOutput(input, RoutineType.Informational, console)).toEqual(expected);
        });

        test("replaces non-array containers and elements with empty arrays", () => {
            const input = JSON.stringify({ containers: "not-an-array", elements: "not-an-array" });
            const expected = { containers: [], elements: [] };
            expect(parseSchemaOutput(input, RoutineType.Code, console)).toEqual(expected);
        });

        test("handles malformed object inputs", () => {
            const input = { someRandomKey: 123 };
            const expected = { containers: [], elements: [], someRandomKey: 123 };
            expect(parseSchemaOutput(input, RoutineType.SmartContract, console)).toEqual(expected);
        });
    });
});

describe("shouldStopRun", () => {
    const defaultLimits: RunRequestLimits = {
        onMaxCredits: "Stop",
        onMaxTime: "Stop",
        onMaxSteps: "Stop",
    };

    const defaultParams: ShouldStopParams = {
        totalStepCost: BigInt(50),
        maxCredits: BigInt(100),
        previousTimeElapsed: 500,
        currentTimeElapsed: 100,
        maxTime: 1000,
        stepsRun: 5,
        maxSteps: 10,
        limits: defaultLimits,
    };

    it("should return InProgress when no limits are exceeded", () => {
        const result = shouldStopRun(defaultParams);
        expect(result).toEqual({ statusChangeReason: undefined, runStatus: RunStatus.InProgress });
    });

    it("should return Failed when maxCredits is exceeded and onMaxCredits is Stop", () => {
        const params = { ...defaultParams, totalStepCost: BigInt(150) };
        const result = shouldStopRun(params);
        expect(result).toEqual({ statusChangeReason: RunStatusChangeReason.MaxCredits, runStatus: RunStatus.Failed });
    });

    it("should return Cancelled when maxCredits is exceeded and onMaxCredits is Cancel", () => {
        const params = {
            ...defaultParams,
            totalStepCost: BigInt(150),
            limits: { ...defaultLimits, onMaxCredits: "Pause" as const },
        };
        const result = shouldStopRun(params);
        expect(result).toEqual({ statusChangeReason: RunStatusChangeReason.MaxCredits, runStatus: RunStatus.Cancelled });
    });

    it("should return Failed when maxTime is exceeded and onMaxTime is Stop", () => {
        const params = { ...defaultParams, currentTimeElapsed: 600 };
        const result = shouldStopRun(params);
        expect(result).toEqual({ statusChangeReason: RunStatusChangeReason.MaxTime, runStatus: RunStatus.Failed });
    });

    it("should return Cancelled when maxTime is exceeded and onMaxTime is Cancel", () => {
        const params = {
            ...defaultParams,
            currentTimeElapsed: 600,
            limits: { ...defaultLimits, onMaxTime: "Pause" as const },
        };
        const result = shouldStopRun(params);
        expect(result).toEqual({ statusChangeReason: RunStatusChangeReason.MaxTime, runStatus: RunStatus.Cancelled });
    });

    it("should return Failed when maxSteps is exceeded and onMaxSteps is Stop", () => {
        const params = { ...defaultParams, stepsRun: 11 };
        const result = shouldStopRun(params);
        expect(result).toEqual({ statusChangeReason: RunStatusChangeReason.MaxSteps, runStatus: RunStatus.Failed });
    });

    it("should return Cancelled when maxSteps is exceeded and onMaxSteps is Cancel", () => {
        const params = {
            ...defaultParams,
            stepsRun: 11,
            limits: { ...defaultLimits, onMaxSteps: "Pause" as const },
        };
        const result = shouldStopRun(params);
        expect(result).toEqual({ statusChangeReason: RunStatusChangeReason.MaxSteps, runStatus: RunStatus.Cancelled });
    });

    it("should prioritize maxCredits over maxTime and maxSteps", () => {
        const params = {
            ...defaultParams,
            totalStepCost: BigInt(150),
            currentTimeElapsed: 600,
            stepsRun: 11,
        };
        const result = shouldStopRun(params);
        expect(result).toEqual({ statusChangeReason: RunStatusChangeReason.MaxCredits, runStatus: RunStatus.Failed });
    });

    it("should prioritize maxTime over maxSteps", () => {
        const params = {
            ...defaultParams,
            currentTimeElapsed: 600,
            stepsRun: 11,
        };
        const result = shouldStopRun(params);
        expect(result).toEqual({ statusChangeReason: RunStatusChangeReason.MaxTime, runStatus: RunStatus.Failed });
    });

    it("should handle edge cases with BigInt values", () => {
        const params = {
            ...defaultParams,
            totalStepCost: BigInt(Number.MAX_SAFE_INTEGER) + BigInt(1),
            maxCredits: BigInt(Number.MAX_SAFE_INTEGER),
        };
        const result = shouldStopRun(params);
        expect(result).toEqual({ statusChangeReason: RunStatusChangeReason.MaxCredits, runStatus: RunStatus.Failed });
    });

    it("should handle edge cases with time values", () => {
        const params = {
            ...defaultParams,
            previousTimeElapsed: Number.MAX_SAFE_INTEGER,
            currentTimeElapsed: 1,
            maxTime: Number.MAX_SAFE_INTEGER,
        };
        const result = shouldStopRun(params);
        expect(result).toEqual({ statusChangeReason: RunStatusChangeReason.MaxTime, runStatus: RunStatus.Failed });
    });
});
