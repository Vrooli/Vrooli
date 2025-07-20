import { describe, expect, it, vi, beforeEach } from "vitest";
import type { Tool } from "@vrooli/shared";
import { ToolConverter } from "./toolConverter.js";
import { logger } from "../../../events/logger.js";

// Mock the logger
vi.mock("../../../events/logger.js", () => ({
    logger: {
        warn: vi.fn(),
        error: vi.fn(),
        info: vi.fn(),
        debug: vi.fn(),
    },
}));

describe("ToolConverter", () => {
    beforeEach(() => {
        // Clear all mocks before each test
        vi.clearAllMocks();
    });
    describe("toOpenAIFormat", () => {
        it("should convert valid tools correctly", () => {
            const tools: Tool[] = [
                {
                    name: "search",
                    description: "Search for information",
                    inputSchema: {
                        type: "object",
                        properties: {
                            query: { type: "string" },
                        },
                        required: ["query"],
                    },
                },
                {
                    name: "calculate",
                    description: "Perform calculations",
                    inputSchema: {
                        type: "object",
                        properties: {
                            expression: { type: "string" },
                        },
                    },
                    category: "math",
                    permissions: ["calculate"],
                },
            ];

            const result = ToolConverter.toOpenAIFormat(tools);

            expect(result).toHaveLength(2);
            expect(result[0]).toEqual({
                type: "function",
                function: {
                    name: "search",
                    description: "Search for information",
                    parameters: {
                        type: "object",
                        properties: {
                            query: { type: "string" },
                        },
                        required: ["query"],
                    },
                    strict: true,
                },
            });
            expect(result[1]).toEqual({
                type: "function",
                function: {
                    name: "calculate",
                    description: "Perform calculations",
                    parameters: {
                        type: "object",
                        properties: {
                            expression: { type: "string" },
                        },
                    },
                    strict: true,
                },
            });
        });

        it("should handle tools with missing description", () => {
            const tools: Tool[] = [
                {
                    name: "test_tool",
                    description: "", // Empty description
                    inputSchema: { type: "object" },
                },
            ];

            const result = ToolConverter.toOpenAIFormat(tools);

            expect(result[0]).toEqual({
                type: "function",
                function: {
                    name: "test_tool",
                    description: undefined,
                    parameters: { type: "object" },
                    strict: true,
                },
            });
        });

        it("should warn and use empty object for missing inputSchema", () => {
            const tools: Tool[] = [
                {
                    name: "bad_tool",
                    description: "Tool without schema",
                    inputSchema: undefined as any, // Missing inputSchema
                },
            ];

            const result = ToolConverter.toOpenAIFormat(tools);

            expect(logger.warn).toHaveBeenCalledWith(
                "Tool \"bad_tool\" has invalid or missing inputSchema, using empty object",
            );
            expect(result[0]).toEqual({
                type: "function",
                function: {
                    name: "bad_tool",
                    description: "Tool without schema",
                    parameters: {},
                    strict: true,
                },
            });
        });

        it("should throw error for tool without name", () => {
            const tools: Tool[] = [
                {
                    name: "",
                    description: "Nameless tool",
                    inputSchema: { type: "object" },
                },
            ];

            expect(() => ToolConverter.toOpenAIFormat(tools)).toThrow(
                "Tool at index 0 is missing required field: name",
            );
        });

        it("should handle empty array", () => {
            const tools: Tool[] = [];
            const result = ToolConverter.toOpenAIFormat(tools);
            expect(result).toEqual([]);
        });

        it("should preserve all valid fields and ignore extra fields", () => {
            const tools: Tool[] = [
                {
                    name: "complex_tool",
                    description: "A complex tool",
                    inputSchema: {
                        type: "object",
                        properties: {
                            nested: {
                                type: "object",
                                properties: {
                                    field: { type: "string" },
                                },
                            },
                        },
                    },
                    outputSchema: { type: "string" }, // Extra field - should be ignored
                    category: "advanced", // Extra field - should be ignored
                    permissions: ["admin"], // Extra field - should be ignored
                },
            ];

            const result = ToolConverter.toOpenAIFormat(tools);

            expect(result[0]).toEqual({
                type: "function",
                function: {
                    name: "complex_tool",
                    description: "A complex tool",
                    parameters: {
                        type: "object",
                        properties: {
                            nested: {
                                type: "object",
                                properties: {
                                    field: { type: "string" },
                                },
                            },
                        },
                    },
                    strict: true,
                },
            });
        });
    });

    describe("isValidTool", () => {
        it("should return true for valid tools", () => {
            const tool: Tool = {
                name: "valid",
                description: "Valid tool",
                inputSchema: { type: "object" },
            };
            expect(ToolConverter.isValidTool(tool)).toBe(true);
        });

        it("should return false for tools without name", () => {
            const tool = {
                name: "",
                description: "No name",
                inputSchema: { type: "object" },
            } as Tool;
            expect(ToolConverter.isValidTool(tool)).toBe(false);
        });

        it("should return false for tools without inputSchema", () => {
            const tool = {
                name: "no_schema",
                description: "No schema",
            } as Tool;
            expect(ToolConverter.isValidTool(tool)).toBe(false);
        });

        it("should return false for tools with non-object inputSchema", () => {
            const tool = {
                name: "bad_schema",
                description: "Bad schema",
                inputSchema: "not an object",
            } as unknown as Tool;
            expect(ToolConverter.isValidTool(tool)).toBe(false);
        });
    });

    describe("toOpenAIFormatSafe", () => {
        it("should filter out invalid tools and convert valid ones", () => {
            const tools: Tool[] = [
                {
                    name: "valid1",
                    description: "Valid tool 1",
                    inputSchema: { type: "object" },
                },
                {
                    name: "", // Invalid - no name
                    description: "Invalid tool",
                    inputSchema: { type: "object" },
                },
                {
                    name: "valid2",
                    description: "Valid tool 2",
                    inputSchema: { type: "object" },
                },
                {
                    name: "invalid2",
                    description: "Invalid tool 2",
                    inputSchema: undefined as any, // Invalid - no schema
                },
            ];

            const result = ToolConverter.toOpenAIFormatSafe(tools);

            expect(logger.warn).toHaveBeenCalledWith(
                "Skipping invalid tools: unnamed, invalid2",
            );
            expect(result).toHaveLength(2);
            expect(result[0].function.name).toBe("valid1");
            expect(result[1].function.name).toBe("valid2");
        });

        it("should handle all invalid tools", () => {
            const tools: Tool[] = [
                {
                    name: "",
                    description: "Invalid 1",
                    inputSchema: { type: "object" },
                },
                {
                    name: "invalid2",
                    description: "Invalid 2",
                    inputSchema: null as any,
                },
            ];

            const result = ToolConverter.toOpenAIFormatSafe(tools);

            expect(result).toEqual([]);
        });

        it("should handle all valid tools", () => {
            const tools: Tool[] = [
                {
                    name: "tool1",
                    description: "Tool 1",
                    inputSchema: { type: "object" },
                },
                {
                    name: "tool2",
                    description: "Tool 2",
                    inputSchema: { type: "string" },
                },
            ];

            const result = ToolConverter.toOpenAIFormatSafe(tools);

            expect(logger.warn).not.toHaveBeenCalled();
            expect(result).toHaveLength(2);
        });
    });

    describe("placeholder methods", () => {
        it("toAnthropicFormat should log warning and return placeholder format", () => {
            const tools: Tool[] = [
                {
                    name: "test",
                    description: "Test tool",
                    inputSchema: { type: "object" },
                },
            ];

            const result = ToolConverter.toAnthropicFormat(tools);

            expect(logger.warn).toHaveBeenCalledWith(
                "Anthropic tool conversion not yet implemented",
            );
            expect(result).toEqual([
                {
                    name: "test",
                    description: "Test tool",
                    input_schema: { type: "object" },
                },
            ]);
        });

        it("toMistralFormat should log warning and return placeholder format", () => {
            const tools: Tool[] = [
                {
                    name: "test",
                    description: "Test tool",
                    inputSchema: { type: "object" },
                },
            ];

            const result = ToolConverter.toMistralFormat(tools);

            expect(logger.warn).toHaveBeenCalledWith(
                "Mistral tool conversion not yet implemented",
            );
            expect(result).toEqual([
                {
                    name: "test",
                    description: "Test tool",
                    parameters: { type: "object" },
                },
            ]);
        });
    });
});
