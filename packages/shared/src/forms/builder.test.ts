/* eslint-disable @typescript-eslint/ban-ts-comment */
import { expect } from "chai";
import { ResourceSubType, ResourceSubTypeRoutine, type Run } from "../api/types.js";
import { InputType } from "../consts/model.js";
import { RoutineVersionConfig, type FormInputConfigObject, type FormOutputConfigObject, type RoutineVersionConfigObject } from "../shape/index.js";
import { FormBuilder } from "./builder.js";
import { FormStructureType, type FormElement, type FormSchema } from "./types.js";

describe("FormBuilder", () => {
    describe("generateInitialValues", () => {
        // Test handling of null input
        it("should return an empty object when elements is null", () => {
            const result = FormBuilder.generateInitialValues(null);
            expect(result).to.deep.equal({});
        });

        // Test handling of undefined input
        it("should return an empty object when elements is undefined", () => {
            const result = FormBuilder.generateInitialValues(undefined);
            expect(result).to.deep.equal({});
        });

        // Test handling of empty array
        it("should return an empty object when elements is an empty array", () => {
            const result = FormBuilder.generateInitialValues([]);
            expect(result).to.deep.equal({});
        });

        // Test skipping non-input elements (no fieldName)
        it("should skip non-input elements without fieldName", () => {
            const elements = [{ type: "header", label: "Section" }] as unknown as readonly FormElement[];
            const result = FormBuilder.generateInitialValues(elements);
            expect(result).to.deep.equal({});
        });

        // Test input with type in healFormInputPropsMap and defaultValue in props
        it("should handle input with type in map and defaultValue", () => {
            const elements = [{ type: "text", fieldName: "name", props: { defaultValue: "John" } }] as unknown as readonly FormElement[];
            const result = FormBuilder.generateInitialValues(elements);
            expect(result).to.deep.equal({ name: "John" });
        });

        // Test input with type in healFormInputPropsMap but no defaultValue
        it("should handle input with type in map without defaultValue", () => {
            const elements = [{ type: "text", fieldName: "email", props: {} }] as unknown as readonly FormElement[];
            const result = FormBuilder.generateInitialValues(elements);
            expect(result).to.deep.equal({ email: "" });
        });

        // Test input with type not in healFormInputPropsMap but with defaultValue
        it("should handle input with type not in map and defaultValue", () => {
            const elements = [{ type: "custom", fieldName: "preference", props: { defaultValue: "dark" } }] as unknown as readonly FormElement[];
            const result = FormBuilder.generateInitialValues(elements);
            expect(result).to.deep.equal({ preference: "dark" });
        });

        // Test input with type not in healFormInputPropsMap and no defaultValue
        it("should handle input with type not in map without defaultValue", () => {
            const elements = [{ type: "custom", fieldName: "other", props: {} }] as unknown as readonly FormElement[];
            const result = FormBuilder.generateInitialValues(elements);
            expect(result).to.deep.equal({ other: "" });
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
            expect(result).to.deep.equal({
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
            expect(result).to.deep.equal({ "input-name": "John" });
        });

        // Test applying a prefix to multiple inputs
        it("should apply prefix to multiple inputs", () => {
            const elements = [
                { type: "text", fieldName: "name", props: { defaultValue: "John" } },
                { type: "checkbox", fieldName: "subscribe", props: { defaultValue: true } },
            ] as unknown as readonly FormElement[];
            const result = FormBuilder.generateInitialValues(elements, "form");
            expect(result).to.deep.equal({
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
            expect(result).to.deep.equal({
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
            expect(result).to.deep.equal({
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

            expect(result).to.deep.equal({
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
            expect(initialValues).to.deep.equal({
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
            expect(initialValues).to.deep.equal({
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

            expect(result).to.deep.equal({});
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

            expect(result).to.deep.equal({
                "input-name": "",
                "input-age": 0,
                "input-subscribe": [false, false],
                "input-active": false,
                "output-result": "",
            });
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

            expect(result).to.deep.equal({
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
            expect(FormBuilder.generateYupSchema(null)).to.be.null;
            // @ts-ignore Testing runtime scenario
            expect(FormBuilder.generateYupSchema(undefined)).to.be.null;
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
                expect(error.message).to.equal("Username must be at least 3 characters");
            }

            // Test validation error - required field
            try {
                await validationSchema.validate({});
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).to.equal("Username is required");
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
                expect(error.message).to.equal("You must be at least 18 years old");
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
                expect(error.message).to.include("valid email");
            }

            // Password too short
            try {
                await validationSchema.validate({
                    email: "test@example.com",
                    password: "short",
                });
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).to.equal("Password must be at least 6 characters");
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
                expect(error.message).to.equal("Must be a valid URL");
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
                expect(error.message).to.equal("You must accept the terms");
            }

            // Terms missing
            try {
                await validationSchema.validate({});
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).to.equal("Accept Terms is required");
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
                expect(error.message).to.equal("Username must be at least 3 characters");
            }

            // Username too long
            try {
                await validationSchema.validate({ username: "thisusernameistoolong" });
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).to.equal("Username must be at most 10 characters");
            }

            // Username not alphanumeric
            try {
                await validationSchema.validate({ username: "user!@#" });
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).to.equal("Username must be alphanumeric");
            }
        });

        it("should skip fields with unsupported input types", () => {
            const formSchema: FormSchema = {
                elements: [
                    {
                        id: "unsupported",
                        label: "Unsupported Field",
                        fieldName: "unsupported",
                        type: InputType.Checkbox, // Assuming Checkbox is not in InputToYupType
                        yup: {
                            required: true,
                            checks: [],
                        },
                        props: {
                            options: [], // Add required options property for CheckboxFormInputProps
                        },
                    },
                ],
                containers: [],
            };

            const validationSchema = FormBuilder.generateYupSchema(formSchema);

            if (!validationSchema) {
                expect.fail("validationSchema should not be null");
            }
            expect(validationSchema.fields).not.to.have.property("unsupported");
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
            expect(validationSchema.fields).not.to.have.property("noValidation");
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

            expect(() => FormBuilder.generateYupSchema(formSchema)).to.throw("Validation method nonExistentMethod does not exist on Yup.string()");
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
                expect(error.message).to.equal("Username must be at least 3 characters");
            }

            // Invalid: status not allowed (if provided)
            try {
                await yupSchema.validate({ "input-username": "JohnDoe", "output-status": "pending" });
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).to.equal("Status must be active or inactive");
            }

            // Invalid: missing required username
            try {
                await yupSchema.validate({ "output-status": "active" });
                expect.fail("Validation should have failed but passed");
            } catch (error) {
                expect(error.message).to.equal("Username is required");
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
                expect(error.message).to.equal("Must be a valid email");
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
                expect(error.message).to.equal("Result is required");
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
});
