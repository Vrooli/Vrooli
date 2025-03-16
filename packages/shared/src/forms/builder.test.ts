/* eslint-disable @typescript-eslint/ban-ts-comment */
import { expect } from "chai";
import { RoutineType, RunRoutine } from "../api/types.js";
import { InputType } from "../consts/model.js";
import { RoutineVersionConfig } from "../run/index.js";
import { FormBuilder } from "./builder.js";
import { FormElement, FormSchema, FormStructureType } from "./types.js";

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
            const sampleInputSchema = {
                elements: [
                    { type: InputType.Text, fieldName: "name", props: { defaultValue: "John" } },
                    { type: InputType.IntegerInput, fieldName: "age", props: { defaultValue: 30 } },
                    { type: InputType.Checkbox, fieldName: "subscribe", props: { defaultValue: [true, false] } },
                    { type: InputType.Switch, fieldName: "active", props: { defaultValue: true } },
                ],
            };
            const sampleOutputSchema = {
                elements: [
                    { type: InputType.Text, fieldName: "result", props: { defaultValue: "success" } },
                ],
            };
            const config = {
                formInput: { schema: sampleInputSchema },
                formOutput: { schema: sampleOutputSchema },
            } as unknown as RoutineVersionConfig;

            const result = FormBuilder.generateInitialValuesFromRoutineConfig(config, RoutineType.Informational);

            expect(result).to.deep.equal({
                "input-name": "John",
                "input-age": 30,
                "input-subscribe": [true, false],
                "input-active": true,
                "output-result": "success",
            });
        });

        it("should work when formInput is not provided", () => {
            const sampleOutputSchema = {
                elements: [
                    { type: InputType.Text, fieldName: "result", props: { defaultValue: "success" } },
                ],
            };
            const config = {
                formOutput: { schema: sampleOutputSchema },
            } as unknown as RoutineVersionConfig;

            const result = FormBuilder.generateInitialValuesFromRoutineConfig(config, RoutineType.Api);

            expect(result).to.deep.equal({
                "output-result": "success",
            });
        });

        it("should work when formOutput is not provided", () => {
            const sampleInputSchema = {
                elements: [
                    { type: InputType.Text, fieldName: "name", props: { defaultValue: "John" } },
                ],
            };
            const config = {
                formInput: { schema: sampleInputSchema },
            } as unknown as RoutineVersionConfig;

            const result = FormBuilder.generateInitialValuesFromRoutineConfig(config, RoutineType.SmartContract);

            expect(result).to.deep.equal({
                "input-name": "John",
            });
        });

        it("should handle schemas with no input elements", () => {
            const sampleInputSchema = {
                elements: [
                    { type: FormStructureType.Header, label: "Input Section" },
                ],
            };
            const sampleOutputSchema = {
                elements: [
                    { type: FormStructureType.Header, label: "Output Section" },
                ],
            };
            const config = {
                formInput: { schema: sampleInputSchema },
                formOutput: { schema: sampleOutputSchema },
            } as unknown as RoutineVersionConfig;

            const result = FormBuilder.generateInitialValuesFromRoutineConfig(config, RoutineType.Action);

            expect(result).to.deep.equal({});
        });

        it("should use correct default for inputs and outputs without default values", () => {
            const sampleInputSchema = {
                elements: [
                    { type: InputType.Text, fieldName: "name", props: {} },
                    { type: InputType.IntegerInput, fieldName: "age", props: {} },
                    { type: InputType.Checkbox, fieldName: "subscribe" },
                    { type: InputType.Switch, fieldName: "active" },
                ],
            };
            const sampleOutputSchema = {
                elements: [
                    { type: InputType.Text, fieldName: "result" },
                ],
            };
            const config = {
                formInput: { schema: sampleInputSchema },
                formOutput: { schema: sampleOutputSchema },
            } as unknown as RoutineVersionConfig;

            const result = FormBuilder.generateInitialValuesFromRoutineConfig(config, RoutineType.Action);

            expect(result).to.deep.equal({
                "input-name": "",
                "input-age": 0,
                "input-subscribe": [],
                "input-active": false,
                "output-result": "",
            });
        });

        it("should override values if run data is provided", () => {
            const sampleInputSchema = {
                elements: [
                    { type: InputType.Text, fieldName: "name", props: { defaultValue: "John" } },
                    { type: InputType.IntegerInput, fieldName: "age" },
                    { type: InputType.Checkbox, fieldName: "subscribe", props: { defaultValue: [true, false] } },
                    { type: InputType.Switch, fieldName: "active" },
                ],
            };
            const sampleOutputSchema = {
                elements: [
                    { type: InputType.JSON, fieldName: "result", props: { defaultValue: { result: "success", info: [] } } },
                    { type: InputType.IntegerInput, fieldName: "error" },
                ],
            };
            const config = {
                formInput: { schema: sampleInputSchema },
                formOutput: { schema: sampleOutputSchema },
            } as unknown as RoutineVersionConfig;
            const run = {
                io: [
                    {
                        data: JSON.stringify("Jane"),
                        routineVersionInput: {
                            name: "name",
                        },
                    },
                    {
                        data: JSON.stringify({ result: "success!", info: [1, 2, 3] }),
                        routineVersionOutput: {
                            name: "result",
                        },
                    },
                    {
                        data: JSON.stringify(101),
                        routineVersionOutput: {
                            name: "error",
                        },
                    },
                    // Add another io with no matching routineVersionInput or routineVersionOutput
                    {
                        data: JSON.stringify({ other: "other" }),
                        routineVersionInput: {
                            name: "noMatch",
                        },
                    },
                ],
            } as unknown as Pick<RunRoutine, "io">;
            const result = FormBuilder.generateInitialValuesFromRoutineConfig(config, RoutineType.Action, run);

            expect(result).to.deep.equal({
                "input-name": "Jane",
                "input-age": 0,
                "input-subscribe": [true, false],
                "input-active": false,
                "output-result": { result: "success!", info: [1, 2, 3] },
                "output-error": 101,
                // Shouldn't include the additional io data
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
            await expect(validationSchema.validate({ username: "abc" })).to.be.fulfilled;
            await expect(validationSchema.validate({ username: "ab" })).to.be.rejectedWith("Username must be at least 3 characters");
            await expect(validationSchema.validate({})).to.be.rejectedWith("Username is required");
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
            await expect(validationSchema.validate({ age: 20 })).to.be.fulfilled;
            await expect(validationSchema.validate({ age: 16 })).to.be.rejectedWith("You must be at least 18 years old");
            await expect(validationSchema.validate({})).to.be.fulfilled; // Age is optional
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

            await expect(
                validationSchema.validate({
                    email: "test@example.com",
                    password: "secret",
                    rememberMe: true,
                }),
            ).to.be.fulfilled;

            await expect(
                validationSchema.validate({
                    email: "invalid-email",
                    password: "secret",
                    rememberMe: false,
                }),
            ).to.be.rejectedWith(/ust be a valid email/);

            await expect(
                validationSchema.validate({
                    email: "test@example.com",
                    password: "short",
                }),
            ).to.be.rejectedWith("Password must be at least 6 characters");

            await expect(
                validationSchema.validate({
                    email: "test@example.com",
                    password: "secret",
                }),
            ).to.be.fulfilled; // rememberMe is optional
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
            await expect(validationSchema.validate({ website: "https://example.com" })).to.be.fulfilled;
            await expect(validationSchema.validate({ website: "invalid-url" })).to.be.rejectedWith("Must be a valid URL");
            await expect(validationSchema.validate({})).to.be.fulfilled; // website is optional
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
                                    value: [true],
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
            await expect(validationSchema.validate({ terms: true })).to.be.fulfilled;
            await expect(validationSchema.validate({ terms: false })).to.be.rejectedWith("You must accept the terms");
            await expect(validationSchema.validate({})).to.be.rejectedWith("Accept Terms is required");
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
                                    value: /^[a-zA-Z0-9]+$/,
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
            await expect(validationSchema.validate({ username: "user123" })).to.be.fulfilled;
            await expect(validationSchema.validate({ username: "us" })).to.be.rejectedWith("Username must be at least 3 characters");
            await expect(validationSchema.validate({ username: "thisusernameistoolong" })).to.be.rejectedWith("Username must be at most 10 characters");
            await expect(validationSchema.validate({ username: "user!@#" })).to.be.rejectedWith("Username must be alphanumeric");
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
                        props: {},
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
                                    value: ["active", "inactive"],
                                    error: "Status must be active or inactive",
                                },
                            ],
                        },
                        props: {},
                    },
                ],
                containers: [],
            };

            const config = {
                formInput: { schema: sampleInputSchema },
                formOutput: { schema: sampleOutputSchema },
            } as unknown as RoutineVersionConfig;

            const yupSchema = FormBuilder.generateYupSchemaFromRoutineConfig(config, RoutineType.Api);

            // Valid case: both fields valid
            await expect(
                yupSchema.validate({ "input-username": "JohnDoe", "output-status": "active" }),
            ).to.be.fulfilled;

            // Invalid: username too short
            await expect(
                yupSchema.validate({ "input-username": "Jo", "output-status": "active" }),
            ).to.be.rejectedWith("Username must be at least 3 characters");

            // Invalid: status not allowed (if provided)
            await expect(
                yupSchema.validate({ "input-username": "JohnDoe", "output-status": "pending" }),
            ).to.be.rejectedWith("Status must be active or inactive");

            // Invalid: missing required username
            await expect(
                yupSchema.validate({ "output-status": "active" }),
            ).to.be.rejectedWith("Username is required");
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

            const configInputOnly = {
                formInput: { schema: sampleInputSchema },
            } as unknown as RoutineVersionConfig;

            const yupSchemaInputOnly = FormBuilder.generateYupSchemaFromRoutineConfig(configInputOnly, RoutineType.Api);

            await expect(
                yupSchemaInputOnly.validate({ "input-email": "test@example.com" }),
            ).to.be.fulfilled;
            await expect(
                yupSchemaInputOnly.validate({ "input-email": "not-an-email" }),
            ).to.be.rejectedWith("Must be a valid email");

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

            const configOutputOnly = {
                formOutput: { schema: sampleOutputSchema },
            } as unknown as RoutineVersionConfig;

            const yupSchemaOutputOnly = FormBuilder.generateYupSchemaFromRoutineConfig(configOutputOnly, RoutineType.Api);

            await expect(
                yupSchemaOutputOnly.validate({ "output-result": "success" }),
            ).to.be.fulfilled;
            await expect(
                yupSchemaOutputOnly.validate({}),
            ).to.be.rejectedWith("Result is required");
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

            const config = {
                formInput: { schema: sampleInputSchema },
            } as unknown as RoutineVersionConfig;

            const yupSchema = FormBuilder.generateYupSchemaFromRoutineConfig(config, RoutineType.Api);
            // Expect that no validation exists for the field since yup is not provided.
            expect(yupSchema.fields).to.not.have.property("input-noValidation");
        });
    });
});
