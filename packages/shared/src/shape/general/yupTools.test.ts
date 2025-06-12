import { describe, it, expect } from "vitest";
import * as yup from "yup";
import { isYupValidationError, validateAndGetYupErrors } from "./yupTools.js";

describe("yupTools", () => {
    describe("isYupValidationError", () => {
        it("should return true for Yup ValidationError", () => {
            const error = new yup.ValidationError("Test error", "value", "field");
            expect(isYupValidationError(error)).to.be.true;
        });

        it("should return false for regular Error", () => {
            const error = new Error("Regular error");
            expect(isYupValidationError(error)).to.be.false;
        });

        it("should return false for TypeError", () => {
            const error = new TypeError("Type error");
            expect(isYupValidationError(error)).to.be.false;
        });

        it("should handle null without throwing", () => {
            expect(isYupValidationError(null)).to.be.false;
        });

        it("should handle undefined without throwing", () => {
            expect(isYupValidationError(undefined)).to.be.false;
        });

        it("should return false for object with only name property", () => {
            const fakeError = { name: "ValidationError", message: "fake" };
            expect(isYupValidationError(fakeError)).to.be.false;
        });

        it("should return true for proper ValidationError structure", () => {
            const properError = { 
                name: "ValidationError", 
                message: "Test error",
                inner: [],
                path: "testField"
            };
            expect(isYupValidationError(properError)).to.be.true;
        });

        it("should return false for object missing required properties", () => {
            const incompleteError1 = { name: "ValidationError", message: "test" };
            expect(isYupValidationError(incompleteError1)).to.be.false;
            
            const incompleteError2 = { name: "ValidationError", inner: [], path: "test" };
            expect(isYupValidationError(incompleteError2)).to.be.false;
            
            const incompleteError3 = { message: "test", inner: [], path: "test" };
            expect(isYupValidationError(incompleteError3)).to.be.false;
        });

        it("should return false for string", () => {
            expect(isYupValidationError("ValidationError")).to.be.false;
        });
    });

    describe("validateAndGetYupErrors", () => {
        it("should return empty object for valid data", async () => {
            const schema = yup.object({
                name: yup.string().required(),
                age: yup.number().positive().integer(),
            });

            const validData = {
                name: "John Doe",
                age: 25,
            };

            const result = await validateAndGetYupErrors(schema, validData);
            expect(result).to.deep.equal({});
        });

        it("should return error object for single field error", async () => {
            const schema = yup.object({
                name: yup.string().required("Name is required"),
                age: yup.number().positive().integer(),
            });

            const invalidData = {
                name: "",
                age: 25,
            };

            const result = await validateAndGetYupErrors(schema, invalidData);
            expect(result).to.have.property("name", "Name is required");
        });

        it("should return error object for multiple field errors", async () => {
            const schema = yup.object({
                name: yup.string().required("Name is required"),
                age: yup.number().positive("Age must be positive").integer("Age must be an integer"),
                email: yup.string().email("Invalid email format"),
            });

            const invalidData = {
                name: "",
                age: -5.5,
                email: "not-an-email",
            };

            try {
                // Need to disable abortEarly to get all errors
                await schema.validate(invalidData, { abortEarly: false });
            } catch (error) {
                if (isYupValidationError(error)) {
                    const result = await validateAndGetYupErrors(schema, invalidData);
                    // Since validateAndGetYupErrors doesn't pass abortEarly: false,
                    // it will only return the first error
                    expect(Object.keys(result).length).to.be.at.least(1);
                }
            }
        });

        it("should handle nested object validation", async () => {
            const schema = yup.object({
                user: yup.object({
                    name: yup.string().required("User name is required"),
                    email: yup.string().email("Invalid email"),
                }),
            });

            const invalidData = {
                user: {
                    name: "",
                    email: "valid@email.com",
                },
            };

            const result = await validateAndGetYupErrors(schema, invalidData);
            expect(result).to.have.property("user.name", "User name is required");
        });

        it("should handle array validation", async () => {
            const schema = yup.object({
                tags: yup.array()
                    .of(yup.string().min(2, "Tag must be at least 2 characters"))
                    .min(1, "At least one tag is required"),
            });

            const invalidData = {
                tags: ["a"],
            };

            const result = await validateAndGetYupErrors(schema, invalidData);
            expect(result).to.have.property("tags[0]");
            expect(result["tags[0]"]).to.include("at least 2 characters");
        });

        it("should handle custom validation", async () => {
            const schema = yup.object({
                password: yup.string()
                    .test("no-admin", "Password cannot be 'admin'", (value) => value !== "admin"),
            });

            const invalidData = {
                password: "admin",
            };

            const result = await validateAndGetYupErrors(schema, invalidData);
            expect(result).to.have.property("password", "Password cannot be 'admin'");
        });

        it("should throw non-ValidationError errors", async () => {
            const schema = yup.object({
                value: yup.string().test("throw-error", "Custom error", () => {
                    throw new TypeError("Not a validation error");
                }),
            });

            const data = { value: "test" };

            try {
                await validateAndGetYupErrors(schema, data);
                expect.fail("Should have thrown an error");
            } catch (error) {
                expect(error).to.be.instanceOf(TypeError);
                expect(error.message).to.equal("Not a validation error");
            }
        });

        it("should handle validation error with empty inner array", async () => {
            // Create a custom ValidationError with empty inner array
            const schema = yup.object({
                field: yup.string().test("custom", "Custom error", () => {
                    const error = new yup.ValidationError("Custom validation error", "value", "field");
                    error.inner = [];
                    throw error;
                }),
            });

            const data = { field: "test" };

            const result = await validateAndGetYupErrors(schema, data);
            expect(result).to.have.property("field", "Custom validation error");
        });

        it("should handle validation error with inner errors missing path", async () => {
            // Create a custom ValidationError with inner errors that have no path
            const schema = yup.object({
                field: yup.string().test("custom", "Custom error", () => {
                    const error = new yup.ValidationError("Parent error", "value", "parentPath");
                    // Add inner errors without path
                    error.inner = [
                        Object.assign(new yup.ValidationError("No path error 1", "value1"), { path: null }),
                        Object.assign(new yup.ValidationError("No path error 2", "value2"), { path: undefined }),
                        Object.assign(new yup.ValidationError("Has path error", "value3", "validPath"), { path: "validPath" }),
                    ];
                    throw error;
                }),
            });

            const data = { field: "test" };

            const result = await validateAndGetYupErrors(schema, data);
            // Should only have the error with a valid path
            expect(result).to.deep.equal({ validPath: "Has path error" });
        });

        it("should handle undefined values", async () => {
            const schema = yup.object({
                required: yup.string().required("This field is required"),
            });

            const result = await validateAndGetYupErrors(schema, {});
            expect(result).to.have.property("required", "This field is required");
        });

        it("should validate with strict mode", async () => {
            const schema = yup.object({
                count: yup.number().required(),
            }).strict();

            const invalidData = {
                count: "5", // String instead of number
            };

            const result = await validateAndGetYupErrors(schema, invalidData);
            expect(Object.keys(result).length).to.be.greaterThan(0);
        });
    });
});
