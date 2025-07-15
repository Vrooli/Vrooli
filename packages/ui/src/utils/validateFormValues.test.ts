// AI_CHECK: TEST_COVERAGE=1 | TEST_QUALITY=1 | LAST: 2025-06-19
// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-01 - Fixed 9 'as any' type assertions with proper vi.mocked usage
import { describe, it, expect, vi, beforeEach } from "vitest";
import { validateFormValues } from "./validateFormValues.js";
import type { YupModel } from "@vrooli/shared";

// Mock the imported function
vi.mock("@vrooli/shared", async () => {
    const actual = await vi.importActual("@vrooli/shared");
    return {
        ...actual,
        validateAndGetYupErrors: vi.fn(),
    };
});

// Import the mocked function
import { validateAndGetYupErrors } from "@vrooli/shared";

describe("validateFormValues", () => {
    const mockValidationMap: YupModel<["create", "update"]> = {
        create: vi.fn(),
        update: vi.fn(),
    };

    const mockTransformFunction = vi.fn();
    const mockCreateSchema = { validate: vi.fn() };
    const mockUpdateSchema = { validate: vi.fn() };

    beforeEach(() => {
        vi.clearAllMocks();
        mockValidationMap.create.mockReturnValue(mockCreateSchema);
        mockValidationMap.update.mockReturnValue(mockUpdateSchema);
    });

    describe("create mode", () => {
        it("validates with create schema when isCreate is true", async () => {
            const values = { name: "test", email: "test@example.com" };
            const existing = {};
            const transformedValues = { name: "test", email: "test@example.com" };
            const expectedErrors = {};

            mockTransformFunction.mockReturnValue(transformedValues);
            vi.mocked(validateAndGetYupErrors).mockResolvedValue(expectedErrors);

            const result = await validateFormValues(
                values,
                existing,
                true,
                mockTransformFunction,
                mockValidationMap,
            );

            expect(mockTransformFunction).toHaveBeenCalledWith(values, existing, true);
            expect(mockValidationMap.create).toHaveBeenCalledWith({ env: process.env.NODE_ENV });
            expect(validateAndGetYupErrors).toHaveBeenCalledWith(mockCreateSchema, transformedValues);
            expect(result).toBe(expectedErrors);
        });

        it("passes through validation errors in create mode", async () => {
            const values = { name: "", email: "invalid" };
            const existing = {};
            const transformedValues = { name: "", email: "invalid" };
            const expectedErrors = { name: "Name is required", email: "Invalid email" };

            mockTransformFunction.mockReturnValue(transformedValues);
            vi.mocked(validateAndGetYupErrors).mockResolvedValue(expectedErrors);

            const result = await validateFormValues(
                values,
                existing,
                true,
                mockTransformFunction,
                mockValidationMap,
            );

            expect(result).toEqual(expectedErrors);
        });
    });

    describe("update mode", () => {
        it("validates with update schema when isCreate is false", async () => {
            const values = { id: "123", name: "updated name" };
            const existing = { id: "123", name: "original name" };
            const transformedValues = { id: "123", name: "updated name" };
            const expectedErrors = {};

            mockTransformFunction.mockReturnValue(transformedValues);
            vi.mocked(validateAndGetYupErrors).mockResolvedValue(expectedErrors);

            const result = await validateFormValues(
                values,
                existing,
                false,
                mockTransformFunction,
                mockValidationMap,
            );

            expect(mockTransformFunction).toHaveBeenCalledWith(values, existing, false);
            expect(mockValidationMap.update).toHaveBeenCalledWith({ env: process.env.NODE_ENV });
            expect(validateAndGetYupErrors).toHaveBeenCalledWith(mockUpdateSchema, transformedValues);
            expect(result).toBe(expectedErrors);
        });

        it("returns 'No changes' error when transform returns undefined", async () => {
            const values = { id: "123", name: "same name" };
            const existing = { id: "123", name: "same name" };

            mockTransformFunction.mockReturnValue(undefined);

            const result = await validateFormValues(
                values,
                existing,
                false,
                mockTransformFunction,
                mockValidationMap,
            );

            expect(result).toEqual({ _form: "No changes" });
            expect(validateAndGetYupErrors).not.toHaveBeenCalled();
        });

        it("returns 'No changes' error when transform returns empty object", async () => {
            const values = { id: "123", name: "same name" };
            const existing = { id: "123", name: "same name" };

            mockTransformFunction.mockReturnValue({});

            const result = await validateFormValues(
                values,
                existing,
                false,
                mockTransformFunction,
                mockValidationMap,
            );

            expect(result).toEqual({ _form: "No changes" });
            expect(validateAndGetYupErrors).not.toHaveBeenCalled();
        });

        it("returns 'No changes' error when transform returns only ID", async () => {
            const values = { id: "123", name: "same name" };
            const existing = { id: "123", name: "same name" };

            mockTransformFunction.mockReturnValue({ id: "123" });

            const result = await validateFormValues(
                values,
                existing,
                false,
                mockTransformFunction,
                mockValidationMap,
            );

            expect(result).toEqual({ _form: "No changes" });
            expect(validateAndGetYupErrors).not.toHaveBeenCalled();
        });

        it("validates when transform returns more than just ID", async () => {
            const values = { id: "123", name: "updated name" };
            const existing = { id: "123", name: "original name" };
            const transformedValues = { id: "123", name: "updated name" };
            const expectedErrors = {};

            mockTransformFunction.mockReturnValue(transformedValues);
            vi.mocked(validateAndGetYupErrors).mockResolvedValue(expectedErrors);

            const result = await validateFormValues(
                values,
                existing,
                false,
                mockTransformFunction,
                mockValidationMap,
            );

            expect(validateAndGetYupErrors).toHaveBeenCalledWith(mockUpdateSchema, transformedValues);
            expect(result).toBe(expectedErrors);
        });

        it("passes through validation errors in update mode", async () => {
            const values = { id: "123", name: "" };
            const existing = { id: "123", name: "original name" };
            const transformedValues = { id: "123", name: "" };
            const expectedErrors = { name: "Name is required" };

            mockTransformFunction.mockReturnValue(transformedValues);
            vi.mocked(validateAndGetYupErrors).mockResolvedValue(expectedErrors);

            const result = await validateFormValues(
                values,
                existing,
                false,
                mockTransformFunction,
                mockValidationMap,
            );

            expect(result).toEqual(expectedErrors);
        });
    });

    describe("environment handling", () => {
        it("passes correct environment to validation schema in create mode", async () => {
            const originalEnv = process.env.NODE_ENV;
            process.env.NODE_ENV = "production";

            const values = { name: "test" };
            const existing = {};
            mockTransformFunction.mockReturnValue({ name: "test" });
            vi.mocked(validateAndGetYupErrors).mockResolvedValue({});

            await validateFormValues(
                values,
                existing,
                true,
                mockTransformFunction,
                mockValidationMap,
            );

            expect(mockValidationMap.create).toHaveBeenCalledWith({ env: "production" });

            process.env.NODE_ENV = originalEnv;
        });

        it("passes correct environment to validation schema in update mode", async () => {
            const originalEnv = process.env.NODE_ENV;
            process.env.NODE_ENV = "development";

            const values = { id: "123", name: "test" };
            const existing = { id: "123", name: "original" };
            mockTransformFunction.mockReturnValue({ id: "123", name: "test" });
            vi.mocked(validateAndGetYupErrors).mockResolvedValue({});

            await validateFormValues(
                values,
                existing,
                false,
                mockTransformFunction,
                mockValidationMap,
            );

            expect(mockValidationMap.update).toHaveBeenCalledWith({ env: "development" });

            process.env.NODE_ENV = originalEnv;
        });
    });

    describe("edge cases", () => {
        it("handles transform function that returns null", async () => {
            const values = { id: "123" };
            const existing = { id: "123" };

            mockTransformFunction.mockReturnValue(null);

            const result = await validateFormValues(
                values,
                existing,
                false,
                mockTransformFunction,
                mockValidationMap,
            );

            expect(result).toEqual({ _form: "No changes" });
        });

        it("handles async validation errors", async () => {
            const values = { name: "test" };
            const existing = {};
            const transformedValues = { name: "test" };
            const validationError = new Error("Validation failed");

            mockTransformFunction.mockReturnValue(transformedValues);
            vi.mocked(validateAndGetYupErrors).mockRejectedValue(validationError);

            await expect(
                validateFormValues(
                    values,
                    existing,
                    true,
                    mockTransformFunction,
                    mockValidationMap,
                ),
            ).rejects.toThrow("Validation failed");
        });

        it("handles complex transformed values object", async () => {
            const values = { id: "123", nested: { prop: "value" }, array: [1, 2, 3] };
            const existing = { id: "123", nested: { prop: "old" }, array: [1, 2] };
            const transformedValues = { 
                id: "123", 
                nested: { prop: "value" }, 
                array: [1, 2, 3],
                additional: "property",
            };
            const expectedErrors = {};

            mockTransformFunction.mockReturnValue(transformedValues);
            vi.mocked(validateAndGetYupErrors).mockResolvedValue(expectedErrors);

            const result = await validateFormValues(
                values,
                existing,
                false,
                mockTransformFunction,
                mockValidationMap,
            );

            expect(Object.keys(transformedValues).length).toBeGreaterThan(1);
            expect(validateAndGetYupErrors).toHaveBeenCalledWith(mockUpdateSchema, transformedValues);
            expect(result).toBe(expectedErrors);
        });
    });
});
