import { type ModelTestFixtures, TestDataFactory } from "../../../validation/models/__test/validationTestUtils.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    chatId1: "123456789012345681",
    chatId2: "123456789012345682",
    userId1: "123456789012345683",
    userId2: "123456789012345684",
    toolCallId1: "123456789012345685",
    toolCallId2: "123456789012345686",
};

// Helper to create valid message config
function createValidConfig(overrides = {}) {
    return {
        __version: "1.0.0",
        resources: [],
        ...overrides,
    };
}

// Helper for tool function calls
function createToolCall(overrides = {}) {
    return {
        id: validIds.toolCallId1,
        function: {
            name: "test_function",
            arguments: JSON.stringify({ param: "value" }),
        },
        ...overrides,
    };
}

// Helper for successful tool result
function createSuccessResult(output = "success output") {
    return {
        success: true,
        output,
    };
}

// Helper for error tool result
function createErrorResult(code = "ERROR_CODE", message = "Error message") {
    return {
        success: false,
        error: { code, message },
    };
}

// Shared chatMessage test fixtures
export const chatMessageFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
            config: createValidConfig(),
            chatConnect: validIds.chatId1,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id2,
            versionIndex: 1,
            config: createValidConfig({
                contextHints: ["hint1", "hint2"],
                eventTopic: "chat.message",
                respondingBots: ["@all", "bot1"],
                role: "user",
                turnId: 123,
                toolCalls: [
                    createToolCall({
                        result: createSuccessResult("Tool executed successfully"),
                    }),
                ],
            }),
            chatConnect: validIds.chatId2,
            userConnect: validIds.userId1,
            translationsCreate: [
                {
                    id: validIds.id3,
                    language: "en",
                    text: "Hello, how are you?",
                },
            ],
        },
        update: {
            id: validIds.id2,
            config: createValidConfig({
                role: "assistant",
                turnId: 124,
            }),
            translationsUpdate: [
                {
                    id: validIds.id3,
                    text: "Updated message text",
                },
            ],
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing id, config, and chatConnect
                versionIndex: 0,
            },
            update: {
                // Missing id
                config: createValidConfig(),
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                versionIndex: "not-a-number", // Should be number
                config: "not-an-object", // Should be object
                chatConnect: 456, // Should be string
            },
            update: {
                id: validIds.id1,
                config: false, // Should be object
            },
        },
        invalidConfig: {
            create: {
                id: validIds.id1,
                config: {
                    // Missing required __version and resources
                    role: "user",
                },
                chatConnect: validIds.chatId1,
            },
        },
        invalidRole: {
            create: {
                id: validIds.id1,
                config: createValidConfig({
                    role: "invalid_role", // Should be one of user, assistant, system, tool
                }),
                chatConnect: validIds.chatId1,
            },
        },
        invalidVersionIndex: {
            create: {
                id: validIds.id1,
                versionIndex: -1, // Should be >= 0
                config: createValidConfig(),
                chatConnect: validIds.chatId1,
            },
        },
        invalidToolCall: {
            create: {
                id: validIds.id1,
                config: createValidConfig({
                    toolCalls: [
                        {
                            // Missing required id and function
                            result: createSuccessResult(),
                        },
                    ],
                }),
                chatConnect: validIds.chatId1,
            },
        },
        invalidToolResult: {
            create: {
                id: validIds.id1,
                config: createValidConfig({
                    toolCalls: [
                        createToolCall({
                            result: {
                                success: true,
                                // Missing output when success is true
                                error: { code: "CODE", message: "msg" }, // Should not have error when success is true
                            },
                        }),
                    ],
                }),
                chatConnect: validIds.chatId1,
            },
        },
        invalidTranslationText: {
            create: {
                id: validIds.id1,
                config: createValidConfig(),
                chatConnect: validIds.chatId1,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        text: "", // Empty text should fail
                    },
                ],
            },
        },
    },
    edgeCases: {
        maxVersionIndex: {
            create: {
                id: validIds.id1,
                versionIndex: 999999,
                config: createValidConfig(),
                chatConnect: validIds.chatId1,
            },
        },
        zeroVersionIndex: {
            create: {
                id: validIds.id1,
                versionIndex: 0,
                config: createValidConfig(),
                chatConnect: validIds.chatId1,
            },
        },
        allRoles: [
            {
                id: validIds.id1,
                config: createValidConfig({ role: "user" }),
                chatConnect: validIds.chatId1,
            },
            {
                id: validIds.id2,
                config: createValidConfig({ role: "assistant" }),
                chatConnect: validIds.chatId1,
            },
            {
                id: validIds.id3,
                config: createValidConfig({ role: "system" }),
                chatConnect: validIds.chatId1,
            },
            {
                id: validIds.id1,
                config: createValidConfig({ role: "tool" }),
                chatConnect: validIds.chatId1,
            },
        ],
        toolCallWithSuccessResult: {
            create: {
                id: validIds.id1,
                config: createValidConfig({
                    toolCalls: [
                        createToolCall({
                            result: createSuccessResult({ data: "complex output" }),
                        }),
                    ],
                }),
                chatConnect: validIds.chatId1,
            },
        },
        toolCallWithErrorResult: {
            create: {
                id: validIds.id1,
                config: createValidConfig({
                    toolCalls: [
                        createToolCall({
                            result: createErrorResult("TOOL_ERROR", "Tool execution failed"),
                        }),
                    ],
                }),
                chatConnect: validIds.chatId1,
            },
        },
        toolCallWithoutResult: {
            create: {
                id: validIds.id1,
                config: createValidConfig({
                    toolCalls: [
                        createToolCall({
                            // No result field - should be optional
                        }),
                    ],
                }),
                chatConnect: validIds.chatId1,
            },
        },
        maxLengthText: {
            create: {
                id: validIds.id1,
                config: createValidConfig(),
                chatConnect: validIds.chatId1,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        text: "x".repeat(32768), // Exactly at max length
                    },
                ],
            },
        },
        textTooLong: {
            create: {
                id: validIds.id1,
                config: createValidConfig(),
                chatConnect: validIds.chatId1,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        text: "x".repeat(32769), // Over max length
                    },
                ],
            },
        },
        complexConfig: {
            create: {
                id: validIds.id1,
                config: createValidConfig({
                    resources: [
                        { id: "resource1", type: "document" },
                        { id: "resource2", type: "image" },
                    ],
                    contextHints: ["context1", "context2", "context3"],
                    eventTopic: "complex.event.topic",
                    respondingBots: ["@all", "bot1", "bot2"],
                    role: "assistant",
                    turnId: null, // Explicitly null
                    toolCalls: [
                        createToolCall({
                            id: validIds.toolCallId1,
                            function: {
                                name: "complex_function",
                                arguments: JSON.stringify({
                                    param1: "value1",
                                    param2: 123,
                                    param3: { nested: "object" },
                                }),
                            },
                            result: createSuccessResult({
                                complexOutput: {
                                    data: [1, 2, 3],
                                    status: "completed",
                                },
                            }),
                        }),
                        createToolCall({
                            id: validIds.toolCallId2,
                            function: {
                                name: "another_function",
                                arguments: "{}",
                            },
                        }),
                    ],
                }),
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
            },
        },
    },
};

// Custom factory that always generates valid IDs
const customizers = {
    create: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
    update: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const chatMessageTestDataFactory = new TestDataFactory(chatMessageFixtures, customizers);
