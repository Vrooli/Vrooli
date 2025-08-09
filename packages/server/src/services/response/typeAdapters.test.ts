import {
    type MessageConfigObject,
    type MessageState,
    generatePK,
} from "@vrooli/shared";
import { describe, expect, it } from "vitest";
import { MessageTypeAdapters } from "./typeAdapters.js";

describe("MessageTypeAdapters", () => {

    describe("messageStateToLlmMessage", () => {
        it("should convert user message correctly", () => {
            const messageState: MessageState = {
                id: generatePK().toString(),
                parent: null,
                user: { id: "user123" },
                text: "Hello, assistant!",
                config: {
                    __version: "1.0",
                    resources: [],
                    role: "user",
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = MessageTypeAdapters.messageStateToLLMMessage(messageState);

            expect(result).toEqual({
                role: "user",
                content: "Hello, assistant!",
                name: "user123",
            });
        });

        it("should convert assistant message correctly", () => {
            const messageState: MessageState = {
                id: generatePK().toString(),
                parent: null,
                user: { id: "bot123" },
                text: "Hello! How can I help you?",
                config: {
                    __version: "1.0",
                    resources: [],
                    role: "assistant",
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = MessageTypeAdapters.messageStateToLLMMessage(messageState);

            expect(result).toEqual({
                role: "assistant",
                content: "Hello! How can I help you?",
                name: "bot123",
            });
        });

        it("should convert system message correctly", () => {
            const messageState: MessageState = {
                id: generatePK().toString(),
                parent: null,
                user: { id: "system" },
                text: "You are a helpful assistant.",
                config: {
                    __version: "1.0",
                    resources: [],
                    role: "system",
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = MessageTypeAdapters.messageStateToLLMMessage(messageState);

            expect(result).toEqual({
                role: "system",
                content: "You are a helpful assistant.",
                name: "system",
            });
        });

        it("should handle message with tool calls", () => {
            const toolCalls = [
                {
                    id: "call123",
                    function: {
                        name: "search",
                        arguments: JSON.stringify({ query: "weather today" }),
                    },
                },
            ];

            const messageState: MessageState = {
                id: generatePK().toString(),
                parent: null,
                user: { id: "bot123" },
                text: "Let me search for that information.",
                config: {
                    __version: "1.0",
                    resources: [],
                    role: "assistant",
                    toolCalls,
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = MessageTypeAdapters.messageStateToLLMMessage(messageState);

            expect(result).toEqual({
                role: "assistant",
                content: "Let me search for that information.",
                name: "bot123",
            });
        });

        it("should handle tool response message", () => {
            const messageState: MessageState = {
                id: generatePK().toString(),
                parent: null,
                user: { id: "tool" },
                text: JSON.stringify({ result: "It's sunny today" }),
                config: {
                    __version: "1.0",
                    resources: [],
                    role: "tool",
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = MessageTypeAdapters.messageStateToLLMMessage(messageState);

            expect(result).toEqual({
                role: "tool",
                content: JSON.stringify({ result: "It's sunny today" }),
                name: "tool",
            });
        });

        it("should handle empty content", () => {
            const messageState: MessageState = {
                id: generatePK().toString(),
                parent: null,
                user: { id: "user123" },
                text: "",
                config: {
                    __version: "1.0",
                    resources: [],
                    role: "user",
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = MessageTypeAdapters.messageStateToLLMMessage(messageState);

            expect(result).toEqual({
                role: "user",
                content: "",
                name: "user123",
            });
        });

        it("should handle missing role in config", () => {
            const messageState: MessageState = {
                id: generatePK().toString(),
                parent: null,
                user: { id: "user123" },
                text: "Test message",
                config: {
                    __version: "1.0",
                    resources: [],
                    // Missing role
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = MessageTypeAdapters.messageStateToLLMMessage(messageState);

            // Should default to user role
            expect(result).toEqual({
                role: "user",
                content: "Test message",
                name: "user123",
            });
        });

        it("should handle multiple tool calls", () => {
            const toolCalls = [
                {
                    id: "call1",
                    function: {
                        name: "search",
                        arguments: JSON.stringify({ query: "weather" }),
                    },
                },
                {
                    id: "call2",
                    function: {
                        name: "calculate",
                        arguments: JSON.stringify({ expression: "2+2" }),
                    },
                },
            ];

            const messageState: MessageState = {
                id: generatePK().toString(),
                parent: null,
                user: { id: "bot123" },
                text: "I'll search and calculate for you.",
                config: {
                    __version: "1.0",
                    resources: [],
                    role: "assistant",
                    toolCalls,
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = MessageTypeAdapters.messageStateToLLMMessage(messageState);

            expect(result).toEqual({
                role: "assistant",
                content: "I'll search and calculate for you.",
                name: "bot123",
            });
        });
    });





    describe("batch conversions", () => {
        it("should convert multiple messages efficiently", () => {
            const messages: MessageState[] = Array(100).fill(0).map((_, i) => ({
                id: generatePK().toString(),
                parent: null,
                user: { id: i % 2 === 0 ? "user123" : "bot123" },
                text: `Message ${i}`,
                config: {
                    __version: "1.0",
                    resources: [],
                    role: i % 2 === 0 ? "user" : "assistant",
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date(Date.now() - (100 - i) * 1000).toISOString(),
            }));

            const results = messages.map(msg =>
                MessageTypeAdapters.messageStateToLLMMessage(msg),
            );

            expect(results).toHaveLength(100);
            expect(results[0].role).toBe("user");
            expect(results[1].role).toBe("assistant");
        });
    });

    describe("edge cases", () => {
        it("should handle null/undefined gracefully", () => {
            const messageState: MessageState = {
                id: generatePK().toString(),
                parent: null,
                user: { id: "user123" },
                text: "Test",
                config: null as any, // Invalid but should handle
                language: "en",
                createdAt: new Date().toISOString(),
            };

            expect(() => MessageTypeAdapters.messageStateToLLMMessage(messageState)).toThrow();
        });

        it("should handle very long content", () => {
            const longContent = "x".repeat(100000); // 100KB

            const messageState: MessageState = {
                id: generatePK().toString(),
                parent: null,
                user: { id: "user123" },
                text: longContent,
                config: {
                    __version: "1.0",
                    resources: [],
                    role: "user",
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = MessageTypeAdapters.messageStateToLLMMessage(messageState);

            expect(result.content).toBe(longContent);
        });

        it("should handle special characters in content", () => {
            const specialContent = "Test with \"quotes\" and 'apostrophes' and \n newlines \t tabs";

            const messageState: MessageState = {
                id: generatePK().toString(),
                parent: null,
                user: { id: "user123" },
                text: specialContent,
                config: {
                    __version: "1.0",
                    resources: [],
                    role: "user",
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = MessageTypeAdapters.messageStateToLLMMessage(messageState);

            expect(result.content).toBe(specialContent);
        });

        it("should handle Unicode content", () => {
            const unicodeContent = "Hello 👋 世界 🌍 नमस्ते 🙏";

            const messageState: MessageState = {
                id: generatePK().toString(),
                parent: null,
                user: { id: "user123" },
                text: unicodeContent,
                config: {
                    __version: "1.0",
                    resources: [],
                    role: "user",
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = MessageTypeAdapters.messageStateToLLMMessage(messageState);

            expect(result.content).toBe(unicodeContent);
        });
    });
});
