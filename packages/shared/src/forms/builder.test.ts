// AI_CHECK: TEST_COVERAGE=4 | LAST: 2025-06-18
/* eslint-disable @typescript-eslint/ban-ts-comment */
import { describe, it, expect } from "vitest";
import { ResourceSubType, ResourceSubTypeRoutine, type Run } from "../api/types.js";
import { InputType } from "../consts/model.js";
import { RoutineVersionConfig, type FormInputConfigObject, type FormOutputConfigObject, type RoutineVersionConfigObject } from "../shape/index.js";
import { FormBuilder, createFormInput, getFormikFieldName, healFormInputPropsMap } from "./builder.js";
import { FormStructureType, type FormElement, type FormSchema } from "./types.js";

describe("FormBuilder", () => {
    describe("generateInitialValues", () => {
        // Test handling of null input - should throw or return empty object based on API design
        it("should return an empty object when elements is null", () => {
            // If the function accepts null, it should handle it gracefully
            // This tests defensive programming against invalid input
            const result = FormBuilder.generateInitialValues(null);
            expect(result).toEqual({});
        });

        // Test handling of undefined input - should throw or return empty object based on API design
        it("should return an empty object when elements is undefined", () => {
            // If the function accepts undefined, it should handle it gracefully
            // This tests defensive programming against invalid input
            const result = FormBuilder.generateInitialValues(undefined);
            expect(result).toEqual({});
        });

        // Test handling of empty array
        it("should return an empty object when elements is an empty array", () => {
            const result = FormBuilder.generateInitialValues([]);
            expect(result).toEqual({});
        });

        // Test skipping non-input elements (no fieldName)
        it("should skip non-input elements without fieldName", () => {
            const elements = [{ type: "header", label: "Section" }] as unknown as readonly FormElement[];
            const result = FormBuilder.generateInitialValues(elements);
            expect(result).toEqual({});
        });

        // Test input with type in healFormInputPropsMap and defaultValue in props
        it("should handle input with type in map and defaultValue", () => {
            const elements = [{ type: "text", fieldName: "name", props: { defaultValue: "John" } }] as unknown as readonly FormElement[];
            const result = FormBuilder.generateInitialValues(elements);
            expect(result).toEqual({ name: "John" });
        });

        // Test input with type in healFormInputPropsMap but no defaultValue
        it("should handle input with type in map without defaultValue", () => {
            const elements = [{ type: "text", fieldName: "email", props: {} }] as unknown as readonly FormElement[];
            const result = FormBuilder.generateInitialValues(elements);
            expect(result).toEqual({ email: "" });
        });

        // Test input with type not in healFormInputPropsMap but with defaultValue
        it("should handle input with type not in map and defaultValue", () => {
            const elements = [{ type: "custom", fieldName: "preference", props: { defaultValue: "dark" } }] as unknown as readonly FormElement[];
            const result = FormBuilder.generateInitialValues(elements);
            expect(result).toEqual({ preference: "dark" });
        });

        // Test input with type not in healFormInputPropsMap and no defaultValue
        it("should handle input with type not in map without defaultValue", () => {
            const elements = [{ type: "custom", fieldName: "other", props: {} }] as unknown as readonly FormElement[];
            const result = FormBuilder.generateInitialValues(elements);
            expect(result).toEqual({ other: "" });
        });

        // Test handling multiple inputs with mixed types and conditions
        it("should handle multiple inputs with various types", () => {
            const elements = [
                { type: "text", fieldName: "name", props: { defaultValue: "John" } },
                { type: "checkbox", fieldName: "subscribe", props: { defaultValue: true } },
                { type: "custom", fieldName: "preference", props: { defaultValue: "dark" } },
                { type: "text", fieldName: "email", props: {} },
                { type: "custom", fieldName: "other", props: {} },
            ] as unknown as readonly FormElement[];
            const result = FormBuilder.generateInitialValues(elements);
            expect(result).toEqual({
                name: "John",
                subscribe: true,
                preference: "dark",
                email: "",
                other: "",
            });
        });

        // Test applying a prefix to a single input
        it("should apply prefix to field names", () => {
            const elements = [{ type: "text", fieldName: "name", props: { defaultValue: "John" } }] as unknown as readonly FormElement[];
            const result = FormBuilder.generateInitialValues(elements, "input");
            expect(result).toEqual({ "input-name": "John" });
        });

        // Test applying a prefix to multiple inputs
        it("should apply prefix to multiple inputs", () => {
            const elements = [
                { type: "text", fieldName: "name", props: { defaultValue: "John" } },
                { type: "checkbox", fieldName: "subscribe", props: { defaultValue: true } },
            ] as unknown as readonly FormElement[];
            const result = FormBuilder.generateInitialValues(elements, "form");
            expect(result).toEqual({
                "form-name": "John",
                "form-subscribe": true,
            });
        });

        // Test mixed array with input and non-input elements
        it("should process inputs and skip non-inputs in a mixed array", () => {
            const elements = [
                { type: "header", label: "Section" },
                { type: "text", fieldName: "name", props: { defaultValue: "John" } },
                { type: "divider" },
                { type: "checkbox", fieldName: "subscribe", props: { defaultValue: true } },
            ] as unknown as readonly FormElement[];
            const result = FormBuilder.generateInitialValues(elements);
            expect(result).toEqual({
                name: "John",
                subscribe: true,
            });
        });

        // Test handling different data types for defaultValue
        it("should handle various data types for defaultValue", () => {
            const elements = [
                { type: "number", fieldName: "age", props: { defaultValue: 18 } },
                { type: "checkbox", fieldName: "agree", props: { defaultValue: false } },
            ] as unknown as readonly FormElement[];
            const result = FormBuilder.generateInitialValues(elements);
            expect(result).toEqual({
                age: 18,
                agree: false,
            });
        });
    });

    describe("generateInitialValuesFromRoutineConfig", () => {
        it("should generate initial values from provided formInput and formOutput schemas", () => {
            const sampleInputSchema: FormSchema = {
                elements: [
                    { id: "name", label: "Name", type: InputType.Text, fieldName: "name", props: { defaultValue: "John" } },
                    { id: "age", label: "Age", type: InputType.IntegerInput, fieldName: "age", props: { defaultValue: 30 } },
                    { id: "subscribe", label: "Subscribe", type: InputType.Checkbox, fieldName: "subscribe", props: { defaultValue: [true, false], options: [{ value: "true", label: "Yes" }, { value: "false", label: "No" }] } },
                    { id: "active", label: "Active", type: InputType.Switch, fieldName: "active", props: { defaultValue: true } },
                ],
                containers: [],
            };
            const sampleOutputSchema: FormSchema = {
                elements: [
                    { id: "result", label: "Result", type: InputType.Text, fieldName: "result", props: { defaultValue: "success" } },
                ],
                containers: [],
            };
            const config = new RoutineVersionConfig({
                config: {
                    __version: "1.0",
                    formInput: { __version: "1.0", schema: sampleInputSchema },
                    formOutput: { __version: "1.0", schema: sampleOutputSchema },
                },
                resourceSubType: ResourceSubType.RoutineInformational,
            });

            const result = FormBuilder.generateInitialValuesFromRoutineConfig(config, ResourceSubTypeRoutine.RoutineInformational);

            expect(result).toEqual({
                "input-name": "John",
                "input-age": 30,
                "input-subscribe": [true, false],
                "input-active": true,
                "output-result": "success",
            });
        });

        it("should work when formInput is not provided", () => {
            const configWithoutInput = new RoutineVersionConfig({
                config: {
                    __version: "1.0",
                    formOutput: {
                        __version: "1.0",
                        schema: {
                            containers: [],
                            elements: [
                                { fieldName: "outputField1", type: InputType.Text, props: { defaultValue: "out1" }, id: "outputField1", label: "Output Field 1" },
                            ],
                        },
                    } as FormOutputConfigObject,
                } as RoutineVersionConfigObject,
                resourceSubType: ResourceSubType.RoutineCode,
            });
            const initialValues = FormBuilder.generateInitialValuesFromRoutineConfig(configWithoutInput, ResourceSubTypeRoutine.RoutineCode);
            // Expect only output fields because formInput is missing and its default is empty
            expect(initialValues).toEqual({
                "output-outputField1": "out1",
            });
        });

        it("should work when formOutput is not provided", () => {
            const configWithoutOutput = new RoutineVersionConfig({
                config: {
                    __version: "1.0",
                    formInput: {
                        __version: "1.0",
                        schema: {
                            containers: [],
                            elements: [
                                { fieldName: "inputField1", type: InputType.Text, props: { defaultValue: "in1" }, id: "inputField1", label: "Input Field 1" },
                            ],
                        },
                    } as FormInputConfigObject,
                } as RoutineVersionConfigObject,
                resourceSubType: ResourceSubType.RoutineCode,
            });
            const initialValues = FormBuilder.generateInitialValuesFromRoutineConfig(configWithoutOutput, ResourceSubTypeRoutine.RoutineCode);
            // Expect only input fields because formOutput is missing and its default is empty
            expect(initialValues).toEqual({
                "input-inputField1": "in1",
            });
        });

        it("should handle schemas with no input elements", () => {
            const sampleInputSchema: FormSchema = {
                elements: [
                    { id: "inputSection", type: FormStructureType.Header, label: "Input Section", tag: "h3" },
                ],
                containers: [],
            };
            const sampleOutputSchema: FormSchema = {
                elements: [
                    { id: "outputSection", type: FormStructureType.Header, label: "Output Section", tag: "h3" },
                ],
                containers: [],
            };
            const config = new RoutineVersionConfig({
                config: {
                    __version: "1.0",
                    formInput: { __version: "1.0", schema: sampleInputSchema },
                    formOutput: { __version: "1.0", schema: sampleOutputSchema },
                },
                resourceSubType: ResourceSubType.RoutineInternalAction,
            });

            const result = FormBuilder.generateInitialValuesFromRoutineConfig(config, ResourceSubTypeRoutine.RoutineInternalAction);

            expect(result).toEqual({});
        });

        it("should use correct default for inputs and outputs without default values", () => {
            const sampleInputSchema: FormSchema = {
                elements: [
                    { id: "name", label: "Name", type: InputType.Text, fieldName: "name", props: {} },
                    { id: "age", label: "Age", type: InputType.IntegerInput, fieldName: "age", props: {} },
                    { id: "subscribe", label: "Subscribe", type: InputType.Checkbox, fieldName: "subscribe", props: { options: [{ value: "true", label: "Yes" }, { value: "false", label: "No" }] } },
                    { id: "active", label: "Active", type: InputType.Switch, fieldName: "active", props: {} },
                ],
                containers: [],
            };
            const sampleOutputSchema: FormSchema = {
                elements: [
                    { id: "result", label: "Result", type: InputType.Text, fieldName: "result", props: {} },
                ],
                containers: [],
            };
            const config = new RoutineVersionConfig({
                config: {
                    __version: "1.0",
                    formInput: { __version: "1.0", schema: sampleInputSchema },
                    formOutput: { __version: "1.0", schema: sampleOutputSchema },
                },
                resourceSubType: ResourceSubType.RoutineInternalAction,
            });

            const result = FormBuilder.generateInitialValuesFromRoutineConfig(config, ResourceSubTypeRoutine.RoutineInternalAction);

            expect(result).toEqual({
                "input-name": "",
                "input-age": 0,
                "input-subscribe": [false, false],
                "input-active": false,
                "output-result": "",
            });
        });

        it("should handle JSON parsing errors in run.io data", () => {
            const sampleInputSchema: FormSchema = {
                elements: [
                    { id: "name", label: "Name", type: InputType.Text, fieldName: "name", props: { defaultValue: "John" } },
                ],
                containers: [],
            };
            const sampleOutputSchema: FormSchema = {
                elements: [
                    { id: "result", label: "Result", type: InputType.Text, fieldName: "result", props: { defaultValue: "success" } },
                ],
                containers: [],
            };
            const config = new RoutineVersionConfig({
                config: {
                    __version: "1.0",
                    formInput: { __version: "1.0", schema: sampleInputSchema },
                    formOutput: { __version: "1.0", schema: sampleOutputSchema },
                },
                resourceSubType: ResourceSubType.RoutineInternalAction,
            });
            
            // Mock console.error to verify error handling behavior
            const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
            
            const run = {
                io: [
                    {
                        data: "invalid json {{{",
                        nodeInputName: "name",
                    },
                    {
                        data: "invalid json [[[",
                        nodeName: "result",
                    },
                ],
            } as unknown as Pick<Run, "io">;
            
            const result = FormBuilder.generateInitialValuesFromRoutineConfig(config, ResourceSubTypeRoutine.RoutineInternalAction, run);

            // When JSON parsing fails, it should use the raw string as fallback
            expect(result).toEqual({
                "input-name": "invalid json {{{",
                "output-result": "invalid json [[[",
            });
            
            // Verify console.error was called for both parsing errors
            expect(consoleSpy).toHaveBeenCalledTimes(2);
            expect(consoleSpy).toHaveBeenCalledWith("Error parsing input run.io data for name:", expect.any(Error));
            expect(consoleSpy).toHaveBeenCalledWith("Error parsing output run.io data for result:", expect.any(Error));
            
            consoleSpy.mockRestore();
        });

        it("should override values if run data is provided", () => {
            const sampleInputSchema: FormSchema = {
                elements: [
                    { id: "name", label: "Name", type: InputType.Text, fieldName: "name", props: { defaultValue: "John" } },
                    { id: "age", label: "Age", type: InputType.IntegerInput, fieldName: "age", props: {} },
                    { id: "subscribe", label: "Subscribe", type: InputType.Checkbox, fieldName: "subscribe", props: { defaultValue: [true, false], options: [{ value: "true", label: "Yes" }, { value: "false", label: "No" }] } },
                    { id: "active", label: "Active", type: InputType.Switch, fieldName: "active", props: {} },
                ],
                containers: [],
            };
            const sampleOutputSchema: FormSchema = {
                elements: [
                    { id: "result", label: "Result", type: InputType.JSON, fieldName: "result", props: { defaultValue: JSON.stringify({ result: "success", info: [] }) } },
                    { id: "error", label: "Error", type: InputType.IntegerInput, fieldName: "error", props: {} },
                ],
                containers: [],
            };
            const config = new RoutineVersionConfig({
                config: {
                    __version: "1.0",
                    formInput: { __version: "1.0", schema: sampleInputSchema },
                    formOutput: { __version: "1.0", schema: sampleOutputSchema },
                },
                resourceSubType: ResourceSubType.RoutineInternalAction,
            });
            const run = {
                io: [
                    {
                        data: JSON.stringify("Jane"),
                        nodeInputName: "name",
                    },
                    {
                        data: JSON.stringify({ result: "success!", info: [1, 2, 3] }),
                        nodeName: "result",
                    },
                    {
                        data: JSON.stringify(101),
                        nodeName: "error",
                    },
                    {
                        data: JSON.stringify({ other: "other" }),
                        nodeInputName: "noMatch",
                    },
                ],
            } as unknown as Pick<Run, "io">;
            const result = FormBuilder.generateInitialValuesFromRoutineConfig(config, ResourceSubTypeRoutine.RoutineInternalAction, run);

            expect(result).toEqual({
                "input-name": "Jane",
                "input-age": 0,
                "input-subscribe": [true, false],
                "input-active": false,
                "output-result": { result: "success!", info: [1, 2, 3] },
                "output-error": 101,
            });
        });
    });

    describe("generateYupSchema", () => {
        it("should return null when formSchema is null or undefined", () => {
            // @ts-ignore Testing runtime scenario
            expect(FormBuilder.generateYupSchema(null)).toBeNull();
            // @ts-ignore Testing runtime scenario
            expect(FormBuilder.generateYupSchema(undefined)).toBeNull();
        });

        it("should generate a Yup schema for required string input", async () => {
            const formSchema: FormSchema = {
                elements: [
                    {
                        id: "username",
                        label: "Username",
                        fieldName: "username",
                        type: InputType.Text,
                        yup: {
                            required: true,
                            checks: [
                                {
                                    key: "min",
                                    value: 3,
                                    error: "Username must be at least 3 characters",
                                },
                            ],
                        },
                        props: {},
                    },
                ],
                containers: [],
            };

            const validationSchema = FormBuilder.generateYupSchema(formSchema);

            if (!validationSchema) {
                expect.fail("validationSchema should not be null");
            }

            // Test valid case
            await validationSchema.validate({ username: "abc" });

            // Test validation error - too short
            try {
                await validationSchema.validate({ username: "ab" });
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).toBe("Username must be at least 3 characters");
            }

            // Test validation error - required field
            try {
                await validationSchema.validate({});
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).toBe("Username is required");
            }
        });

        it("should generate a Yup schema for optional number input", async () => {
            const formSchema: FormSchema = {
                elements: [
                    {
                        id: "age",
                        label: "Age",
                        fieldName: "age",
                        type: InputType.IntegerInput,
                        yup: {
                            required: false,
                            checks: [
                                {
                                    key: "min",
                                    value: 18,
                                    error: "You must be at least 18 years old",
                                },
                            ],
                        },
                        props: {},
                    },
                ],
                containers: [],
            };

            const validationSchema = FormBuilder.generateYupSchema(formSchema);

            if (!validationSchema) {
                expect.fail("validationSchema should not be null");
            }

            // Valid age
            await validationSchema.validate({ age: 20 });

            // Invalid age
            try {
                await validationSchema.validate({ age: 16 });
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).toBe("You must be at least 18 years old");
            }

            // Age is optional
            await validationSchema.validate({});
        });

        it("should handle multiple fields with different types", async () => {
            const formSchema: FormSchema = {
                elements: [
                    {
                        id: "email",
                        label: "Email",
                        fieldName: "email",
                        type: InputType.Text,
                        yup: {
                            required: true,
                            checks: [
                                {
                                    key: "email",
                                    error: "Must be a valid email",
                                },
                            ],
                        },
                        props: {},
                    },
                    {
                        id: "password",
                        label: "Password",
                        fieldName: "password",
                        type: InputType.Text,
                        yup: {
                            required: true,
                            checks: [
                                {
                                    key: "min",
                                    value: 6,
                                    error: "Password must be at least 6 characters",
                                },
                            ],
                        },
                        props: {},
                    },
                    {
                        id: "rememberMe",
                        label: "Remember Me",
                        fieldName: "rememberMe",
                        type: InputType.Switch,
                        yup: {
                            required: false,
                            checks: [],
                        },
                        props: {},
                    },
                ],
                containers: [],
            };

            const validationSchema = FormBuilder.generateYupSchema(formSchema);

            if (!validationSchema) {
                expect.fail("validationSchema should not be null");
            }

            // Valid case with all fields
            await validationSchema.validate({
                email: "test@example.com",
                password: "secret",
                rememberMe: true,
            });

            // Invalid email format
            try {
                await validationSchema.validate({
                    email: "invalid-email",
                    password: "secret",
                    rememberMe: false,
                });
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).toContain("valid email");
            }

            // Password too short
            try {
                await validationSchema.validate({
                    email: "test@example.com",
                    password: "short",
                });
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).toBe("Password must be at least 6 characters");
            }

            // RememberMe is optional
            await validationSchema.validate({
                email: "test@example.com",
                password: "secret",
            });
        });

        it("should handle custom validation methods", async () => {
            const formSchema: FormSchema = {
                elements: [
                    {
                        id: "website",
                        label: "Website",
                        fieldName: "website",
                        type: InputType.Text,
                        yup: {
                            required: false,
                            checks: [
                                {
                                    key: "url",
                                    error: "Must be a valid URL",
                                },
                            ],
                        },
                        props: {},
                    },
                ],
                containers: [],
            };

            const validationSchema = FormBuilder.generateYupSchema(formSchema);

            if (!validationSchema) {
                expect.fail("validationSchema should not be null");
            }

            // Valid URL
            await validationSchema.validate({ website: "https://example.com" });

            // Invalid URL
            try {
                await validationSchema.validate({ website: "invalid-url" });
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).toBe("Must be a valid URL");
            }

            // Website is optional
            await validationSchema.validate({});
        });

        it("should handle boolean fields", async () => {
            const formSchema: FormSchema = {
                elements: [
                    {
                        id: "terms",
                        label: "Accept Terms",
                        fieldName: "terms",
                        type: InputType.Switch,
                        yup: {
                            required: true,
                            checks: [
                                {
                                    key: "oneOf",
                                    value: [true] as any, // Type cast to avoid compile error
                                    error: "You must accept the terms",
                                },
                            ],
                        },
                        props: {},
                    },
                ],
                containers: [],
            };

            const validationSchema = FormBuilder.generateYupSchema(formSchema);

            if (!validationSchema) {
                expect.fail("validationSchema should not be null");
            }

            // Terms accepted
            await validationSchema.validate({ terms: true });

            // Terms declined
            try {
                await validationSchema.validate({ terms: false });
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).toBe("You must accept the terms");
            }

            // Terms missing
            try {
                await validationSchema.validate({});
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).toBe("Accept Terms is required");
            }
        });

        it("should handle fields with multiple checks", async () => {
            const formSchema: FormSchema = {
                elements: [
                    {
                        id: "username",
                        label: "Username",
                        fieldName: "username",
                        type: InputType.Text,
                        yup: {
                            required: true,
                            checks: [
                                {
                                    key: "min",
                                    value: 3,
                                    error: "Username must be at least 3 characters",
                                },
                                {
                                    key: "max",
                                    value: 10,
                                    error: "Username must be at most 10 characters",
                                },
                                {
                                    key: "matches",
                                    value: /^[a-zA-Z0-9]+$/ as any, // Type cast to avoid compile error
                                    error: "Username must be alphanumeric",
                                },
                            ],
                        },
                        props: {},
                    },
                ],
                containers: [],
            };

            const validationSchema = FormBuilder.generateYupSchema(formSchema);

            if (!validationSchema) {
                expect.fail("validationSchema should not be null");
            }

            // Valid username
            await validationSchema.validate({ username: "user123" });

            // Username too short
            try {
                await validationSchema.validate({ username: "us" });
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).toBe("Username must be at least 3 characters");
            }

            // Username too long
            try {
                await validationSchema.validate({ username: "thisusernameistoolong" });
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).toBe("Username must be at most 10 characters");
            }

            // Username not alphanumeric
            try {
                await validationSchema.validate({ username: "user!@#" });
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).toBe("Username must be alphanumeric");
            }
        });

        it("should skip fields with unsupported input types", () => {
            const formSchema: FormSchema = {
                elements: [
                    {
                        id: "unsupported",
                        label: "Unsupported Field",
                        fieldName: "unsupported",
                        type: InputType.Dropzone, // Dropzone is not in InputToYupType
                        yup: {
                            required: true,
                            checks: [],
                        },
                        props: {
                            defaultValue: [], // Add required defaultValue property for DropzoneFormInputProps
                        },
                    },
                ],
                containers: [],
            };

            const validationSchema = FormBuilder.generateYupSchema(formSchema);

            if (!validationSchema) {
                expect.fail("validationSchema should not be null");
            }
            expect(validationSchema.fields).not.toHaveProperty("unsupported");
        });

        it("should handle fields without yup property", () => {
            const formSchema: FormSchema = {
                elements: [
                    {
                        id: "noValidation",
                        label: "No Validation",
                        fieldName: "noValidation",
                        type: InputType.Text,
                        props: {},
                    },
                ],
                containers: [],
            };

            const validationSchema = FormBuilder.generateYupSchema(formSchema);

            if (!validationSchema) {
                expect.fail("validationSchema should not be null");
            }
            expect(validationSchema.fields).not.toHaveProperty("noValidation");
        });


        it("should handle validation methods with different parameter combinations", async () => {
            const formSchema: FormSchema = {
                elements: [
                    {
                        id: "field1",
                        label: "Field 1",
                        fieldName: "field1",
                        type: InputType.Text,
                        yup: {
                            required: false,
                            checks: [
                                {
                                    key: "trim", // Method with no parameters
                                },
                                {
                                    key: "lowercase", // Another method with no parameters
                                },
                                {
                                    key: "default", // Method that takes only a value
                                    value: "default value",
                                },
                                {
                                    key: "min", // Method that takes value and error
                                    value: 3,
                                    error: "Must be at least 3 characters",
                                },
                            ],
                        },
                        props: {},
                    },
                ],
                containers: [],
            };

            const validationSchema = FormBuilder.generateYupSchema(formSchema);

            if (!validationSchema) {
                expect.fail("validationSchema should not be null");
            }

            // Test that the schema was created successfully
            const result = await validationSchema.validate({ field1: "  TEST  " });
            // Note: The actual behavior depends on how yup handles these methods
            expect(result).to.be.an("object");
        });

        it("should throw an error for invalid check methods", () => {
            const formSchema: FormSchema = {
                elements: [
                    {
                        id: "invalidCheck",
                        label: "Invalid Check",
                        fieldName: "invalidCheck",
                        type: InputType.Text,
                        yup: {
                            required: true,
                            checks: [
                                {
                                    key: "nonExistentMethod",
                                    value: true,
                                    error: "This method does not exist",
                                },
                            ],
                        },
                        props: {},
                    },
                ],
                containers: [],
            };

            // Modify generateYupSchema to throw an error or log a warning when an invalid method is used

            expect(() => FormBuilder.generateYupSchema(formSchema)).toThrow("Validation method nonExistentMethod does not exist on Yup.string()");
        });

        // it("should handle nested objects if required", () => {
        //     // Assuming your form supports nested objects
        //     const formSchema: FormSchema = {
        //         elements: [
        //             {
        //                 id: "address",
        //                 label: "Address",
        //                 fieldName: "address",
        //                 type: InputType.Object,
        //                 yup: {
        //                     required: true,
        //                     checks: [],
        //                 },
        //                 props: {},
        //             },
        //         ],
        //         containers: [],
        //     };

        //     // Implement nested object handling in generateYupSchema if necessary
        // });
    });

    describe("generateYupSchemaFromRoutineConfig", () => {
        it("should generate a combined Yup schema for formInput and formOutput", async () => {
            const sampleInputSchema: FormSchema = {
                elements: [
                    {
                        id: "username",
                        label: "Username",
                        fieldName: "username",
                        type: InputType.Text,
                        yup: {
                            required: true,
                            checks: [
                                {
                                    key: "min",
                                    value: 3,
                                    error: "Username must be at least 3 characters",
                                },
                            ],
                        },
                        props: {},
                    },
                ],
                containers: [],
            };

            const sampleOutputSchema: FormSchema = {
                elements: [
                    {
                        id: "status",
                        label: "Status",
                        fieldName: "status",
                        type: InputType.Text,
                        yup: {
                            required: false,
                            checks: [
                                {
                                    key: "oneOf",
                                    value: ["active", "inactive"] as any, // Type cast to avoid compile error
                                    error: "Status must be active or inactive",
                                },
                            ],
                        },
                        props: {},
                    },
                ],
                containers: [],
            };

            const config = new RoutineVersionConfig({
                config: {
                    __version: "1.0",
                    formInput: { __version: "1.0", schema: sampleInputSchema as unknown as FormInputConfigObject["schema"] },
                    formOutput: { __version: "1.0", schema: sampleOutputSchema as unknown as FormOutputConfigObject["schema"] },
                },
                resourceSubType: ResourceSubType.RoutineApi,
            });

            const yupSchema = FormBuilder.generateYupSchemaFromRoutineConfig(config, ResourceSubTypeRoutine.RoutineApi);

            // Valid case: both fields valid
            await yupSchema.validate({ "input-username": "JohnDoe", "output-status": "active" });

            // Invalid: username too short
            try {
                await yupSchema.validate({ "input-username": "Jo", "output-status": "active" });
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).toBe("Username must be at least 3 characters");
            }

            // Invalid: status not allowed (if provided)
            try {
                await yupSchema.validate({ "input-username": "JohnDoe", "output-status": "pending" });
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).toBe("Status must be active or inactive");
            }

            // Invalid: missing required username
            try {
                await yupSchema.validate({ "output-status": "active" });
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).toBe("Username is required");
            }
        });

        it("should handle cases when only one of formInput or formOutput is provided", async () => {
            // Test with only formInput provided
            const sampleInputSchema: FormSchema = {
                elements: [
                    {
                        id: "email",
                        label: "Email",
                        fieldName: "email",
                        type: InputType.Text,
                        yup: {
                            required: true,
                            checks: [
                                {
                                    key: "email",
                                    error: "Must be a valid email",
                                },
                            ],
                        },
                        props: {},
                    },
                ],
                containers: [],
            };

            const configInputOnly = new RoutineVersionConfig({
                config: {
                    __version: "1.0",
                    formInput: { __version: "1.0", schema: sampleInputSchema as unknown as FormInputConfigObject["schema"] },
                },
                resourceSubType: ResourceSubType.RoutineApi,
            });

            const yupSchemaInputOnly = FormBuilder.generateYupSchemaFromRoutineConfig(configInputOnly, ResourceSubTypeRoutine.RoutineApi);

            // Valid email
            await yupSchemaInputOnly.validate({ "input-email": "test@example.com" });

            // Invalid email
            try {
                await yupSchemaInputOnly.validate({ "input-email": "not-an-email" });
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).toBe("Must be a valid email");
            }

            // Test with only formOutput provided
            const sampleOutputSchema: FormSchema = {
                elements: [
                    {
                        id: "result",
                        label: "Result",
                        fieldName: "result",
                        type: InputType.Text,
                        yup: {
                            required: true,
                            checks: [],
                        },
                        props: {},
                    },
                ],
                containers: [],
            };

            const configOutputOnly = new RoutineVersionConfig({
                config: {
                    __version: "1.0",
                    formOutput: { __version: "1.0", schema: sampleOutputSchema as unknown as FormOutputConfigObject["schema"] },
                },
                resourceSubType: ResourceSubType.RoutineApi,
            });

            const yupSchemaOutputOnly = FormBuilder.generateYupSchemaFromRoutineConfig(configOutputOnly, ResourceSubTypeRoutine.RoutineApi);

            // Valid result
            await yupSchemaOutputOnly.validate({ "output-result": "success" });

            // Missing required result
            try {
                await yupSchemaOutputOnly.validate({});
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).toBe("Result is required");
            }
        });

        it("should skip fields without a yup property", () => {
            const sampleInputSchema: FormSchema = {
                elements: [
                    {
                        id: "noValidation",
                        label: "No Validation",
                        fieldName: "noValidation",
                        type: InputType.Text,
                        // No yup validation provided
                        props: { defaultValue: "default" },
                    },
                ],
                containers: [],
            };

            const config = new RoutineVersionConfig({
                config: {
                    __version: "1.0",
                    formInput: { __version: "1.0", schema: sampleInputSchema as unknown as FormInputConfigObject["schema"] },
                },
                resourceSubType: ResourceSubType.RoutineApi,
            });

            const yupSchema = FormBuilder.generateYupSchemaFromRoutineConfig(config, ResourceSubTypeRoutine.RoutineApi);
            // Expect that no validation exists for the field since yup is not provided.
            expect(yupSchema.fields).to.not.have.property("input-noValidation");
        });
    });

    describe("createFormInput", () => {
        it("should create a form input with valid type and heal props", () => {
            const input = createFormInput({
                fieldName: "testField",
                id: "test-id",
                label: "Test Field",
                type: InputType.Text,
                props: { defaultValue: "hello", maxChars: 500 },
                yup: { required: true, checks: [] }
            });

            expect(input).not.toBeNull();
            expect(input!.fieldName).toBe("testField");
            expect(input!.id).toBe("test-id");
            expect(input!.label).toBe("Test Field");
            expect(input!.type).toBe(InputType.Text);
            expect(input!.props.defaultValue).toBe("hello");
            expect(input!.props.maxChars).toBe(500);
            expect(input!.yup.required).toBe(true);
        });

        it("should return null for invalid input type", () => {
            const input = createFormInput({
                fieldName: "testField",
                type: "InvalidType" as any,
                props: {},
                yup: {}
            });

            expect(input).toBeNull();
        });

        it("should parse stringified props", () => {
            const propsString = JSON.stringify({ defaultValue: "parsed", maxChars: 200 });
            const input = createFormInput({
                fieldName: "testField",
                type: InputType.Text,
                props: propsString,
                yup: { required: false, checks: [] }
            });

            expect(input).not.toBeNull();
            expect(input!.props.defaultValue).toBe("parsed");
            expect(input!.props.maxChars).toBe(200);
        });

        it("should parse stringified yup", () => {
            const yupString = JSON.stringify({ required: true, checks: [{ key: "min", value: 5 }] });
            const input = createFormInput({
                fieldName: "testField",
                type: InputType.Text,
                props: {},
                yup: yupString
            });

            expect(input).not.toBeNull();
            expect(input!.yup.required).toBe(true);
            expect(input!.yup.checks).to.have.length(1);
            expect(input!.yup.checks[0].key).toBe("min");
        });

        it("should reject invalid JSON in props and return null with error", () => {
            const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
            
            const input = createFormInput({
                fieldName: "testField",
                type: InputType.Text,
                props: "invalid json {",
                yup: {}
            });

            // Invalid JSON should result in null return value
            expect(input).toBeNull();
            
            // Error should be logged for debugging
            expect(consoleSpy).toHaveBeenCalledWith("Error parsing props/yup", expect.any(Error));
            
            consoleSpy.mockRestore();
        });

        it("should reject invalid JSON in yup and return null with error", () => {
            const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
            
            const input = createFormInput({
                fieldName: "testField",
                type: InputType.Text,
                props: {},
                yup: "invalid json ["
            });

            // Invalid JSON should result in null return value
            expect(input).toBeNull();
            
            // Error should be logged for debugging
            expect(consoleSpy).toHaveBeenCalledWith("Error parsing props/yup", expect.any(Error));
            
            consoleSpy.mockRestore();
        });

        it("should handle null/undefined props and yup", () => {
            const input = createFormInput({
                fieldName: "testField",
                type: InputType.Text,
                props: null,
                yup: undefined
            });

            expect(input).not.toBeNull();
            expect(input!.props).to.be.an("object");
            expect(input!.yup.checks).to.be.an("array");
        });

        it("should generate nanoid when id is not provided", () => {
            const input = createFormInput({
                fieldName: "testField",
                type: InputType.Text,
                props: {},
                yup: {}
            });

            expect(input).not.toBeNull();
            expect(input!.id).to.be.a("string");
            expect(input!.id.length).toBeGreaterThan(0);
        });

        it("should use empty strings for missing fieldName and label", () => {
            const input = createFormInput({
                type: InputType.Text,
                props: {},
                yup: {}
            });

            expect(input).not.toBeNull();
            expect(input!.fieldName).toBe("");
            expect(input!.label).toBe("");
        });

        it("should handle non-object parsed props by using empty object", () => {
            const input = createFormInput({
                fieldName: "testField",
                type: InputType.Text,
                props: '"string instead of object"',
                yup: {}
            });

            expect(input).not.toBeNull();
            expect(input!.props).to.be.an("object");
        });

        it("should handle non-object parsed yup by using empty object", () => {
            const input = createFormInput({
                fieldName: "testField", 
                type: InputType.Text,
                props: {},
                yup: '"string instead of object"'
            });

            expect(input).not.toBeNull();
            expect(input!.yup).to.be.an("object");
            // When parsing fails or returns non-object, it should default to { checks: [] }
            if (input!.yup.checks) {
                expect(input!.yup.checks).to.be.an("array");
            }
        });

        it("should pass through additional properties via rest parameter", () => {
            const input = createFormInput({
                fieldName: "testField",
                type: InputType.Text,
                props: {},
                yup: {},
                customProperty: "customValue"
            } as any);

            expect(input).not.toBeNull();
            expect((input as any).customProperty).toBe("customValue");
        });
    });

    describe("getFormikFieldName", () => {
        it("should return fieldName when no prefix is provided", () => {
            const result = getFormikFieldName("username");
            expect(result).toBe("username");
        });

        it("should return prefixed fieldName when prefix is provided", () => {
            const result = getFormikFieldName("username", "input");
            expect(result).toBe("input-username");
        });

        it("should handle empty strings", () => {
            const result1 = getFormikFieldName("", "prefix");
            expect(result1).toBe("prefix-");

            // Empty prefix is falsy, so it returns just the fieldName
            const result2 = getFormikFieldName("field", "");
            expect(result2).toBe("field");
        });

        it("should handle special characters in fieldName and prefix", () => {
            const result = getFormikFieldName("field_name", "form.input");
            expect(result).toBe("form.input-field_name");
        });
    });

    describe("healFormInputPropsMap", () => {

        describe("healCheckboxProps", () => {
            it("should provide defaults for checkbox props", () => {
                const healed = healFormInputPropsMap[InputType.Checkbox]({});
                expect(healed.color).toBe("secondary");
                expect(healed.defaultValue).toEqual([false]);
                expect(healed.options).toEqual([{ label: "Option 1", value: "option-1" }]);
                expect(healed.maxSelection).toBe(0);
                expect(healed.minSelection).toBe(0);
                expect(healed.row).toBe(false);
            });

            it("should preserve existing checkbox props", () => {
                const props = {
                    color: "primary",
                    defaultValue: [true, false],
                    options: [
                        { label: "Option A", value: "a" },
                        { label: "Option B", value: "b" },
                    ],
                    maxSelection: 2,
                    minSelection: 1,
                    row: true,
                };
                const healed = healFormInputPropsMap[InputType.Checkbox](props);
                expect(healed).toEqual(props);
            });
        });

        describe("healDropzoneProps", () => {
            it("should provide defaults for dropzone props", () => {
                const healed = healFormInputPropsMap[InputType.Dropzone]({});
                expect(healed.defaultValue).toEqual([]);
            });

            it("should preserve existing dropzone props", () => {
                const props = {
                    defaultValue: ["file1.txt", "file2.txt"],
                    accept: ".txt,.pdf",
                };
                const healed = healFormInputPropsMap[InputType.Dropzone](props);
                expect(healed).toEqual(props);
            });
        });

        describe("healJsonProps", () => {
            it("should provide defaults for JSON props", () => {
                const healed = healFormInputPropsMap[InputType.JSON]({});
                expect(healed.defaultValue).toBe("");
            });

            it("should preserve existing JSON props", () => {
                const props = {
                    defaultValue: '{"key": "value"}',
                };
                const healed = healFormInputPropsMap[InputType.JSON](props);
                expect(healed).toEqual(props);
            });
        });

        describe("healRadioProps", () => {
            it("should provide default value from first option", () => {
                const props = {
                    options: [
                        { label: "Yes", value: "yes" },
                        { label: "No", value: "no" },
                    ],
                };
                const healed = healFormInputPropsMap[InputType.Radio](props);
                expect(healed.defaultValue).toBe("yes");
                expect(healed.options).toEqual(props.options);
            });

            it("should handle empty options", () => {
                const healed = healFormInputPropsMap[InputType.Radio]({});
                expect(healed.defaultValue).toBe("");
                expect(healed.options).toEqual([]);
            });
        });

        describe("healSliderProps", () => {
            it("should provide defaults for slider props", () => {
                const healed = healFormInputPropsMap[InputType.Slider]({});
                expect(healed.min).toBe(0);
                expect(healed.max).toBe(100);
                expect(healed.step).toBe(5); // (100-0)/20 = 5
                expect(healed.defaultValue).toBe(60); // nearest((0+100)/2, 0, 100, step) with step calculation
            });

            it("should calculate defaultValue based on min/max", () => {
                const props = {
                    min: 10,
                    max: 50,
                };
                const healed = healFormInputPropsMap[InputType.Slider](props);
                expect(healed.defaultValue).toBe(30); // nearest((10+50)/2, 10, 50, (50-10)/20)
            });

            it("should preserve existing slider props", () => {
                const props = {
                    min: 5,
                    max: 95,
                    step: 5,
                    defaultValue: 50,
                };
                const healed = healFormInputPropsMap[InputType.Slider](props);
                expect(healed).toEqual(props);
            });
        });

        describe("healSwitchProps", () => {
            it("should provide defaults for switch props", () => {
                const healed = healFormInputPropsMap[InputType.Switch]({});
                expect(healed.defaultValue).toBe(false);
                expect(healed.color).toBe("secondary");
            });

            it("should preserve existing switch props", () => {
                const props = {
                    defaultValue: true,
                    color: "primary",
                };
                const healed = healFormInputPropsMap[InputType.Switch](props);
                expect(healed).toEqual({
                    ...props,
                    label: "",
                    size: "medium",
                });
            });
        });

        describe("healTextProps", () => {
            it("should provide defaults for text props", () => {
                const healed = healFormInputPropsMap[InputType.Text]({});
                expect(healed.autoComplete).toBe("off");
                expect(healed.defaultValue).toBe("");
                expect(healed.isMarkdown).toBe(true);
                expect(healed.maxChars).toBe(1000);
                expect(healed.maxRows).toBe(2);
                expect(healed.minRows).toBe(4);
            });

            it("should preserve existing text props", () => {
                const props = {
                    autoComplete: "email",
                    defaultValue: "test@example.com",
                    isMarkdown: true,
                    maxChars: 500,
                    maxRows: 5,
                    minRows: 2,
                };
                const healed = healFormInputPropsMap[InputType.Text](props);
                expect(healed).toEqual(props);
            });
        });
    });
});
