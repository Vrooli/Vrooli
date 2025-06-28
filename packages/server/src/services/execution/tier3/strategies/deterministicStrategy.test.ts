import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { type Logger } from "winston";
import { type EventBus } from "../../../events/eventBus.js";
import { DeterministicStrategy } from "./deterministicStrategy.js";

// Mock dependencies
const mockLogger: Logger = {
    debug: vi.fn(),
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
} as any;

const mockEventBus: EventBus = {
    emit: vi.fn(),
    on: vi.fn(),
    off: vi.fn(),
} as any;

// Mock fetch globally
global.fetch = vi.fn();

describe("DeterministicStrategy", () => {
    let strategy: DeterministicStrategy;

    beforeEach(() => {
        strategy = new DeterministicStrategy(mockLogger, mockEventBus);
        vi.clearAllMocks();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("executeApiCall", () => {
        it("should execute GET request successfully", async () => {
            const mockResponse = {
                ok: true,
                status: 200,
                statusText: "OK",
                headers: new Headers({ "content-type": "application/json" }),
                json: vi.fn().mockResolvedValue({ data: "test" }),
            };
            (global.fetch as any).mockResolvedValue(mockResponse);

            const result = await (strategy as any).executeApiCall({
                url: "https://api.example.com/data",
                method: "GET",
            }, 5000);

            expect(global.fetch).toHaveBeenCalledWith(
                "https://api.example.com/data",
                expect.objectContaining({
                    method: "GET",
                    signal: expect.any(AbortSignal),
                }),
            );

            expect(result).toMatchObject({
                status: 200,
                statusText: "OK",
                data: { data: "test" },
                ok: true,
                timestamp: expect.any(String),
            });
        });

        it("should execute POST request with JSON body", async () => {
            const mockResponse = {
                ok: true,
                status: 201,
                statusText: "Created",
                headers: new Headers({ "content-type": "application/json" }),
                json: vi.fn().mockResolvedValue({ id: 123 }),
            };
            (global.fetch as any).mockResolvedValue(mockResponse);

            const result = await (strategy as any).executeApiCall({
                url: "https://api.example.com/users",
                method: "POST",
                headers: { "content-type": "application/json" },
                body: { name: "John Doe", email: "john@example.com" },
            }, 5000);

            expect(global.fetch).toHaveBeenCalledWith(
                "https://api.example.com/users",
                expect.objectContaining({
                    method: "POST",
                    body: JSON.stringify({ name: "John Doe", email: "john@example.com" }),
                    headers: { "content-type": "application/json" },
                }),
            );

            expect(result).toMatchObject({
                status: 201,
                data: { id: 123 },
            });
        });

        it("should handle timeout errors", async () => {
            (global.fetch as any).mockRejectedValue(new DOMException("Aborted", "AbortError"));

            await expect(
                (strategy as any).executeApiCall({
                    url: "https://api.example.com/slow",
                }, 100),
            ).rejects.toThrow("API call timed out after 100ms");
        });

        it("should handle different response types", async () => {
            const mockText = "Plain text response";
            const mockResponse = {
                ok: true,
                status: 200,
                statusText: "OK",
                headers: new Headers({ "content-type": "text/plain" }),
                text: vi.fn().mockResolvedValue(mockText),
            };
            (global.fetch as any).mockResolvedValue(mockResponse);

            const result = await (strategy as any).executeApiCall({
                url: "https://api.example.com/text",
                responseType: "text",
            }, 5000);

            expect(result).toMatchObject({
                data: mockText,
            });
        });

        it("should throw error when URL is missing", async () => {
            await expect(
                (strategy as any).executeApiCall({}, 5000),
            ).rejects.toThrow("API call requires a valid URL parameter");
        });
    });

    describe("executeDataTransform", () => {
        it("should return input when no transform rules provided", async () => {
            const input = { foo: "bar" };
            const result = await (strategy as any).executeDataTransform({ input });
            expect(result).toEqual(input);
        });

        it("should apply simple field mapping", async () => {
            const input = { oldField: "value", other: "data" };
            const transformRules = {
                newField: "oldField",
                copied: "other",
            };

            const result = await (strategy as any).executeDataTransform({
                input,
                transformRules,
            });

            expect(result).toEqual({
                newField: "value",
                copied: "data",
            });
        });

        it("should apply array transformation rules in sequence", async () => {
            const input = [
                { name: "John", age: 30 },
                { name: "Jane", age: 25 },
            ];

            const transformRules = [
                {
                    type: "filter",
                    field: "age",
                    operation: "greaterThan",
                    value: 26,
                },
                {
                    type: "map",
                    field: "isAdult",
                    value: true,
                },
            ];

            const result = await (strategy as any).executeDataTransform({
                input,
                transformRules,
            });

            expect(result).toEqual([
                { name: "John", age: 30, isAdult: true },
            ]);
        });

        it("should handle rename transformations", async () => {
            const input = { oldName: "value", keep: "this" };
            const transformRules = [
                {
                    type: "rename",
                    from: "oldName",
                    to: "newName",
                },
            ];

            const result = await (strategy as any).executeDataTransform({
                input,
                transformRules,
            });

            expect(result).toEqual({
                newName: "value",
                keep: "this",
            });
        });

        it("should handle replace transformations", async () => {
            const input = {
                text: "Hello World",
                nested: {
                    message: "Hello Universe",
                },
            };

            const transformRules = [
                {
                    type: "replace",
                    pattern: "Hello",
                    replacement: "Hi",
                },
            ];

            const result = await (strategy as any).executeDataTransform({
                input,
                transformRules,
            });

            expect(result).toEqual({
                text: "Hi World",
                nested: {
                    message: "Hi Universe",
                },
            });
        });

        it("should handle extract transformations", async () => {
            const input = {
                user: {
                    profile: {
                        name: "John",
                        email: "john@example.com",
                    },
                },
            };

            const transformRules = [
                {
                    type: "extract",
                    field: "user.profile.name",
                },
            ];

            const result = await (strategy as any).executeDataTransform({
                input,
                transformRules,
            });

            expect(result).toBe("John");
        });

        it("should handle merge transformations", async () => {
            const input = { existing: "data" };
            const transformRules = [
                {
                    type: "merge",
                    value: { new: "field", another: "value" },
                },
            ];

            const result = await (strategy as any).executeDataTransform({
                input,
                transformRules,
            });

            expect(result).toEqual({
                existing: "data",
                new: "field",
                another: "value",
            });
        });

        it("should handle complex object transformations", async () => {
            const input = {
                user: {
                    firstName: "John",
                    lastName: "Doe",
                    age: 30,
                },
                metadata: {
                    created: "2024-01-01",
                },
            };

            const transformRules = {
                name: {
                    source: "user",
                    transform: {
                        type: "extract",
                        field: "firstName",
                    },
                },
                fullName: {
                    source: "user",
                    transform: {
                        type: "map",
                        field: "fullName",
                        value: "John Doe", // In real use, this would be computed
                    },
                },
                isAdult: {
                    defaultValue: false,
                },
                createdAt: "metadata.created",
            };

            const result = await (strategy as any).executeDataTransform({
                input,
                transformRules,
            });

            expect(result).toEqual({
                name: "John",
                fullName: {
                    firstName: "John",
                    lastName: "Doe",
                    age: 30,
                    fullName: "John Doe",
                },
                isAdult: false,
                createdAt: "2024-01-01",
            });
        });

        it("should handle output format transformations", async () => {
            const input = { data: "value" };

            // Test array format
            let result = await (strategy as any).executeDataTransform({
                input,
                outputFormat: "array",
            });
            expect(result).toEqual([{ data: "value" }]);

            // Test string format
            result = await (strategy as any).executeDataTransform({
                input,
                outputFormat: "string",
            });
            expect(result).toBe("{\"data\":\"value\"}");

            // Test object format with primitive
            result = await (strategy as any).executeDataTransform({
                input: "primitive",
                outputFormat: "object",
            });
            expect(result).toEqual({ value: "primitive" });
        });

        it("should handle nested path operations", async () => {
            const input = {
                deeply: {
                    nested: {
                        value: "found",
                    },
                },
            };

            // Test getting nested value
            expect((strategy as any).getNestedValue(input, "deeply.nested.value")).toBe("found");
            expect((strategy as any).getNestedValue(input, "deeply.missing.value")).toBeUndefined();

            // Test setting nested value
            const obj = { ...input };
            (strategy as any).setNestedValue(obj, "deeply.nested.newValue", "set");
            expect(obj.deeply.nested).toHaveProperty("newValue", "set");

            // Test deleting nested value
            (strategy as any).deleteNestedValue(obj, "deeply.nested.value");
            expect(obj.deeply.nested).not.toHaveProperty("value");
        });

        it("should handle conditional evaluations correctly", async () => {
            const data = {
                number: 10,
                text: "hello world",
                flag: true,
                missing: null,
            };

            const strategy = new DeterministicStrategy(mockLogger, mockEventBus);

            // Test various conditions
            expect((strategy as any).evaluateCondition(data, "number", "equals", 10)).toBe(true);
            expect((strategy as any).evaluateCondition(data, "number", "greaterThan", 5)).toBe(true);
            expect((strategy as any).evaluateCondition(data, "number", "lessThan", 20)).toBe(true);
            expect((strategy as any).evaluateCondition(data, "text", "contains", "hello")).toBe(true);
            expect((strategy as any).evaluateCondition(data, "text", "startsWith", "hello")).toBe(true);
            expect((strategy as any).evaluateCondition(data, "text", "endsWith", "world")).toBe(true);
            expect((strategy as any).evaluateCondition(data, "flag", "exists", null)).toBe(true);
            expect((strategy as any).evaluateCondition(data, "missing", "exists", null)).toBe(false);
        });
    });
});
