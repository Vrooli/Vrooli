import { expect } from "chai";
import { InputType } from "../consts/model.js";
import { FormSchema } from "./types.js";
import { generateYupSchema } from "./yupGenerator.js";

describe("generateYupSchema", () => {
    it("should return null when formSchema is null or undefined", () => {
        expect(generateYupSchema(null as any)).to.be.null;
        expect(generateYupSchema(undefined as any)).to.be.null;
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

        const validationSchema = generateYupSchema(formSchema);

        await expect(validationSchema!.validate({ username: "abc" })).resolves.toBeDefined();
        await expect(validationSchema!.validate({ username: "ab" })).rejects.toThrow("Username must be at least 3 characters");
        await expect(validationSchema!.validate({})).rejects.toThrow("Username is required");
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

        const validationSchema = generateYupSchema(formSchema);

        await expect(validationSchema!.validate({ age: 20 })).resolves.toBeDefined();
        await expect(validationSchema!.validate({ age: 16 })).rejects.toThrow("You must be at least 18 years old");
        await expect(validationSchema!.validate({})).resolves.toBeDefined(); // Age is optional
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

        const validationSchema = generateYupSchema(formSchema);

        await expect(
            validationSchema!.validate({
                email: "test@example.com",
                password: "secret",
                rememberMe: true,
            }),
        ).resolves.toBeDefined();

        await expect(
            validationSchema!.validate({
                email: "invalid-email",
                password: "secret",
                rememberMe: false,
            }),
        ).rejects.toThrow(expect.objectContaining({ message: expect.stringContaining("ust be a valid email") }));

        await expect(
            validationSchema!.validate({
                email: "test@example.com",
                password: "short",
            }),
        ).rejects.toThrow("Password must be at least 6 characters");

        await expect(
            validationSchema!.validate({
                email: "test@example.com",
                password: "secret",
            }),
        ).resolves.toBeDefined(); // rememberMe is optional
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

        const validationSchema = generateYupSchema(formSchema);

        await expect(validationSchema!.validate({ website: "https://example.com" })).resolves.toBeDefined();
        await expect(validationSchema!.validate({ website: "invalid-url" })).rejects.toThrow("Must be a valid URL");
        await expect(validationSchema!.validate({})).resolves.toBeDefined(); // website is optional
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

        const validationSchema = generateYupSchema(formSchema);

        await expect(validationSchema!.validate({ terms: true })).resolves.toBeDefined();
        await expect(validationSchema!.validate({ terms: false })).rejects.toThrow("You must accept the terms");
        await expect(validationSchema!.validate({})).rejects.toThrow("Accept Terms is required");
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

        const validationSchema = generateYupSchema(formSchema);

        await expect(validationSchema!.validate({ username: "user123" })).resolves.toBeDefined();
        await expect(validationSchema!.validate({ username: "us" })).rejects.toThrow("Username must be at least 3 characters");
        await expect(validationSchema!.validate({ username: "thisusernameistoolong" })).rejects.toThrow("Username must be at most 10 characters");
        await expect(validationSchema!.validate({ username: "user!@#" })).rejects.toThrow("Username must be alphanumeric");
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

        const validationSchema = generateYupSchema(formSchema);

        expect(validationSchema!.fields).not.to.have.property("unsupported");
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

        const validationSchema = generateYupSchema(formSchema);

        expect(validationSchema!.fields).not.to.have.property("noValidation");
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

        expect(() => generateYupSchema(formSchema)).toThrow("Validation method nonExistentMethod does not exist on Yup.string()");
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
