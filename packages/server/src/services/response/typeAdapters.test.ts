import { describe, it, expect, beforeEach } from "vitest";
import { 
    type MessageState,
    type MessageConfigObject,
    type ChatMessage,
    type Tool,
    type ToolCall,
    generatePK,
} from "@vrooli/shared";
import { MessageTypeAdapters } from "./typeAdapters.js";

describe("MessageTypeAdapters", () => {
    let adapters: MessageTypeAdapters;

    beforeEach(() => {
        adapters = new MessageTypeAdapters();
    });

    describe("messageStateToLlmMessage", () => {
        it("should convert user message correctly", () => {
            const messageState: MessageState = {
                id: generatePK(),
                parent: null,
                userId: "user123",
                content: "Hello, assistant!",
                config: {
                    modelType: "gpt-4",
                    role: "user",
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = adapters.messageStateToLlmMessage(messageState);

            expect(result).toEqual({
                role: "user",
                content: "Hello, assistant!",
            });
        });

        it("should convert assistant message correctly", () => {
            const messageState: MessageState = {
                id: generatePK(),
                parent: null,
                userId: "bot123",
                content: "Hello! How can I help you?",
                config: {
                    modelType: "gpt-4",
                    role: "assistant",
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = adapters.messageStateToLlmMessage(messageState);

            expect(result).toEqual({
                role: "assistant",
                content: "Hello! How can I help you?",
            });
        });

        it("should convert system message correctly", () => {
            const messageState: MessageState = {
                id: generatePK(),
                parent: null,
                userId: "system",
                content: "You are a helpful assistant.",
                config: {
                    modelType: "gpt-4",
                    role: "system",
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = adapters.messageStateToLlmMessage(messageState);

            expect(result).toEqual({
                role: "system",
                content: "You are a helpful assistant.",
            });
        });

        it("should handle message with tool calls", () => {
            const toolCalls: ToolCall[] = [
                {
                    id: "call123",
                    name: "search",
                    arguments: { query: "weather today" },
                },
            ];

            const messageState: MessageState = {
                id: generatePK(),
                parent: null,
                userId: "bot123",
                content: "Let me search for that information.",
                config: {
                    modelType: "gpt-4",
                    role: "assistant",
                    toolCalls,
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = adapters.messageStateToLlmMessage(messageState);

            expect(result).toEqual({
                role: "assistant",
                content: "Let me search for that information.",
                tool_calls: [
                    {
                        id: "call123",
                        type: "function",
                        function: {
                            name: "search",
                            arguments: JSON.stringify({ query: "weather today" }),
                        },
                    },
                ],
            });
        });

        it("should handle tool response message", () => {
            const messageState: MessageState = {
                id: generatePK(),
                parent: null,
                userId: "tool",
                content: JSON.stringify({ result: "It's sunny today" }),
                config: {
                    modelType: "gpt-4",
                    role: "tool",
                    toolCallId: "call123",
                    name: "search",
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = adapters.messageStateToLlmMessage(messageState);

            expect(result).toEqual({
                role: "tool",
                content: JSON.stringify({ result: "It's sunny today" }),
                tool_call_id: "call123",
                name: "search",
            });
        });

        it("should handle empty content", () => {
            const messageState: MessageState = {
                id: generatePK(),
                parent: null,
                userId: "user123",
                content: "",
                config: {
                    modelType: "gpt-4",
                    role: "user",
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = adapters.messageStateToLlmMessage(messageState);

            expect(result).toEqual({
                role: "user",
                content: "",
            });
        });

        it("should handle missing role in config", () => {
            const messageState: MessageState = {
                id: generatePK(),
                parent: null,
                userId: "user123",
                content: "Test message",
                config: {
                    modelType: "gpt-4",
                    // Missing role
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = adapters.messageStateToLlmMessage(messageState);

            // Should default to user role
            expect(result).toEqual({
                role: "user",
                content: "Test message",
            });
        });

        it("should handle multiple tool calls", () => {
            const toolCalls: ToolCall[] = [
                {
                    id: "call1",
                    name: "search",
                    arguments: { query: "weather" },
                },
                {
                    id: "call2",
                    name: "calculate",
                    arguments: { expression: "2+2" },
                },
            ];

            const messageState: MessageState = {
                id: generatePK(),
                parent: null,
                userId: "bot123",
                content: "I'll search and calculate for you.",
                config: {
                    modelType: "gpt-4",
                    role: "assistant",
                    toolCalls,
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = adapters.messageStateToLlmMessage(messageState);

            expect(result.tool_calls).toHaveLength(2);
            expect(result.tool_calls?.[0]).toMatchObject({
                id: "call1",
                type: "function",
                function: {
                    name: "search",
                    arguments: JSON.stringify({ query: "weather" }),
                },
            });
            expect(result.tool_calls?.[1]).toMatchObject({
                id: "call2",
                type: "function",
                function: {
                    name: "calculate",
                    arguments: JSON.stringify({ expression: "2+2" }),
                },
            });
        });
    });

    describe("chatMessageToLlmMessage", () => {
        it("should convert ChatMessage correctly", () => {
            const chatMessage: ChatMessage = {
                role: "user",
                content: "What's the weather?",
            };

            const result = adapters.chatMessageToLlmMessage(chatMessage);

            expect(result).toEqual({
                role: "user",
                content: "What's the weather?",
            });
        });

        it("should preserve tool_calls if present", () => {
            const chatMessage: ChatMessage = {
                role: "assistant",
                content: "Let me check the weather.",
                tool_calls: [
                    {
                        id: "tc123",
                        type: "function",
                        function: {
                            name: "get_weather",
                            arguments: "{\"location\": \"New York\"}",
                        },
                    },
                ],
            };

            const result = adapters.chatMessageToLlmMessage(chatMessage);

            expect(result).toEqual(chatMessage);
        });

        it("should handle tool role messages", () => {
            const chatMessage: ChatMessage = {
                role: "tool",
                content: "Sunny, 72°F",
                tool_call_id: "tc123",
                name: "get_weather",
            };

            const result = adapters.chatMessageToLlmMessage(chatMessage);

            expect(result).toEqual(chatMessage);
        });
    });

    describe("toolToOpenAIFormat", () => {
        it("should convert tool with all fields", () => {
            const tool: Tool = {
                name: "search_web",
                description: "Search the web for information",
                inputSchema: {
                    type: "object",
                    properties: {
                        query: {
                            type: "string",
                            description: "The search query",
                        },
                        limit: {
                            type: "number",
                            description: "Maximum results",
                            default: 10,
                        },
                    },
                    required: ["query"],
                },
                category: "search",
                permissions: ["web_access"],
            };

            const result = adapters.toolToOpenAIFormat(tool);

            expect(result).toEqual({
                type: "function",
                function: {
                    name: "search_web",
                    description: "Search the web for information",
                    parameters: {
                        type: "object",
                        properties: {
                            query: {
                                type: "string",
                                description: "The search query",
                            },
                            limit: {
                                type: "number",
                                description: "Maximum results",
                                default: 10,
                            },
                        },
                        required: ["query"],
                    },
                },
            });
        });

        it("should handle tool without description", () => {
            const tool: Tool = {
                name: "simple_tool",
                description: "",
                inputSchema: {
                    type: "object",
                    properties: {},
                },
            };

            const result = adapters.toolToOpenAIFormat(tool);

            expect(result).toEqual({
                type: "function",
                function: {
                    name: "simple_tool",
                    description: "",
                    parameters: {
                        type: "object",
                        properties: {},
                    },
                },
            });
        });

        it("should handle complex nested schemas", () => {
            const tool: Tool = {
                name: "complex_tool",
                description: "A tool with nested schema",
                inputSchema: {
                    type: "object",
                    properties: {
                        user: {
                            type: "object",
                            properties: {
                                name: { type: "string" },
                                age: { type: "number" },
                                tags: {
                                    type: "array",
                                    items: { type: "string" },
                                },
                            },
                            required: ["name"],
                        },
                        options: {
                            type: "object",
                            additionalProperties: true,
                        },
                    },
                    required: ["user"],
                },
            };

            const result = adapters.toolToOpenAIFormat(tool);

            expect(result.function.parameters).toEqual(tool.inputSchema);
        });
    });

    describe("toolCallToInternal", () => {
        it("should convert OpenAI tool call format to internal format", () => {
            const openAIToolCall = {
                id: "call_abc123",
                type: "function" as const,
                function: {
                    name: "get_weather",
                    arguments: "{\"location\": \"San Francisco\", \"unit\": \"celsius\"}",
                },
            };

            const result = adapters.toolCallToInternal(openAIToolCall);

            expect(result).toEqual({
                id: "call_abc123",
                name: "get_weather",
                arguments: {
                    location: "San Francisco",
                    unit: "celsius",
                },
            });
        });

        it("should handle empty arguments", () => {
            const openAIToolCall = {
                id: "call_xyz",
                type: "function" as const,
                function: {
                    name: "simple_tool",
                    arguments: "{}",
                },
            };

            const result = adapters.toolCallToInternal(openAIToolCall);

            expect(result).toEqual({
                id: "call_xyz",
                name: "simple_tool",
                arguments: {},
            });
        });

        it("should handle invalid JSON in arguments", () => {
            const openAIToolCall = {
                id: "call_bad",
                type: "function" as const,
                function: {
                    name: "bad_tool",
                    arguments: "not valid json",
                },
            };

            const result = adapters.toolCallToInternal(openAIToolCall);

            expect(result).toEqual({
                id: "call_bad",
                name: "bad_tool",
                arguments: {}, // Should default to empty object
            });
        });

        it("should handle array arguments", () => {
            const openAIToolCall = {
                id: "call_array",
                type: "function" as const,
                function: {
                    name: "process_items",
                    arguments: "[\"item1\", \"item2\", \"item3\"]",
                },
            };

            const result = adapters.toolCallToInternal(openAIToolCall);

            expect(result).toEqual({
                id: "call_array",
                name: "process_items",
                arguments: ["item1", "item2", "item3"],
            });
        });

        it("should handle complex nested arguments", () => {
            const complexArgs = {
                query: "test",
                filters: {
                    date: { from: "2024-01-01", to: "2024-12-31" },
                    categories: ["tech", "science"],
                },
                options: {
                    limit: 10,
                    offset: 0,
                    sortBy: "relevance",
                },
            };

            const openAIToolCall = {
                id: "call_complex",
                type: "function" as const,
                function: {
                    name: "advanced_search",
                    arguments: JSON.stringify(complexArgs),
                },
            };

            const result = adapters.toolCallToInternal(openAIToolCall);

            expect(result).toEqual({
                id: "call_complex",
                name: "advanced_search",
                arguments: complexArgs,
            });
        });
    });

    describe("llmMessageToMessageState", () => {
        it("should convert LLM message to MessageState", () => {
            const llmMessage = {
                role: "assistant" as const,
                content: "Here's the information you requested.",
            };

            const result = adapters.llmMessageToMessageState(
                llmMessage,
                "user123",
                "en",
            );

            expect(result).toMatchObject({
                id: expect.any(String),
                parent: null,
                userId: "user123",
                content: "Here's the information you requested.",
                config: {
                    role: "assistant",
                },
                language: "en",
                createdAt: expect.any(String),
            });
        });

        it("should include tool calls in config", () => {
            const llmMessage = {
                role: "assistant" as const,
                content: "Let me search for that.",
                tool_calls: [
                    {
                        id: "tc1",
                        type: "function" as const,
                        function: {
                            name: "search",
                            arguments: "{\"q\": \"test\"}",
                        },
                    },
                ],
            };

            const result = adapters.llmMessageToMessageState(
                llmMessage,
                "bot123",
                "en",
            );

            expect(result.config).toMatchObject({
                role: "assistant",
                toolCalls: [
                    {
                        id: "tc1",
                        name: "search",
                        arguments: { q: "test" },
                    },
                ],
            });
        });

        it("should handle tool response messages", () => {
            const llmMessage = {
                role: "tool" as const,
                content: "{\"result\": \"Success\"}",
                tool_call_id: "tc1",
                name: "my_tool",
            };

            const result = adapters.llmMessageToMessageState(
                llmMessage,
                "tool",
                "en",
            );

            expect(result.config).toMatchObject({
                role: "tool",
                toolCallId: "tc1",
                name: "my_tool",
            });
        });

        it("should handle system messages", () => {
            const llmMessage = {
                role: "system" as const,
                content: "You are a helpful assistant.",
            };

            const result = adapters.llmMessageToMessageState(
                llmMessage,
                "system",
                "en",
            );

            expect(result).toMatchObject({
                userId: "system",
                content: "You are a helpful assistant.",
                config: {
                    role: "system",
                },
            });
        });

        it("should use current timestamp for createdAt", () => {
            const before = new Date();
            
            const llmMessage = {
                role: "user" as const,
                content: "Test",
            };

            const result = adapters.llmMessageToMessageState(
                llmMessage,
                "user123",
                "en",
            );

            const after = new Date();
            const createdAt = new Date(result.createdAt);

            expect(createdAt.getTime()).toBeGreaterThanOrEqual(before.getTime());
            expect(createdAt.getTime()).toBeLessThanOrEqual(after.getTime());
        });
    });

    describe("batch conversions", () => {
        it("should convert multiple messages efficiently", () => {
            const messages: MessageState[] = Array(100).fill(0).map((_, i) => ({
                id: generatePK(),
                parent: null,
                userId: i % 2 === 0 ? "user123" : "bot123",
                content: `Message ${i}`,
                config: {
                    modelType: "gpt-4",
                    role: i % 2 === 0 ? "user" : "assistant",
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date(Date.now() - (100 - i) * 1000).toISOString(),
            }));

            const results = messages.map(msg => 
                adapters.messageStateToLlmMessage(msg),
            );

            expect(results).toHaveLength(100);
            expect(results[0].role).toBe("user");
            expect(results[1].role).toBe("assistant");
        });
    });

    describe("edge cases", () => {
        it("should handle null/undefined gracefully", () => {
            const messageState: MessageState = {
                id: generatePK(),
                parent: null,
                userId: "user123",
                content: "Test",
                config: null as any, // Invalid but should handle
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = adapters.messageStateToLlmMessage(messageState);

            expect(result).toEqual({
                role: "user",
                content: "Test",
            });
        });

        it("should handle very long content", () => {
            const longContent = "x".repeat(100000); // 100KB
            
            const messageState: MessageState = {
                id: generatePK(),
                parent: null,
                userId: "user123",
                content: longContent,
                config: {
                    modelType: "gpt-4",
                    role: "user",
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = adapters.messageStateToLlmMessage(messageState);

            expect(result.content).toBe(longContent);
        });

        it("should handle special characters in content", () => {
            const specialContent = "Test with \"quotes\" and 'apostrophes' and \n newlines \t tabs";
            
            const messageState: MessageState = {
                id: generatePK(),
                parent: null,
                userId: "user123",
                content: specialContent,
                config: {
                    modelType: "gpt-4",
                    role: "user",
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = adapters.messageStateToLlmMessage(messageState);

            expect(result.content).toBe(specialContent);
        });

        it("should handle Unicode content", () => {
            const unicodeContent = "Hello 👋 世界 🌍 नमस्ते 🙏";
            
            const messageState: MessageState = {
                id: generatePK(),
                parent: null,
                userId: "user123",
                content: unicodeContent,
                config: {
                    modelType: "gpt-4",
                    role: "user",
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = adapters.messageStateToLlmMessage(messageState);

            expect(result.content).toBe(unicodeContent);
        });
    });
});
