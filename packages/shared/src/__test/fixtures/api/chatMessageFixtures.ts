import type { ChatMessageCreateInput, ChatMessageUpdateInput } from "../../../api/types.js";
import { type MessageConfigObject, type ToolFunctionCall } from "../../../shape/configs/message.js";
import { type ModelTestFixtures, type TestDataFactory, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { chatMessageValidation } from "../../../validation/models/chatMessage.js";
import { messageConfigFixtures, createSuccessfulToolCall, createFailedToolCall } from "../config/messageConfigFixtures.js";

// Magic number constants for testing
const TEXT_MAX_LENGTH = 32768;
const TEXT_TOO_LONG_LENGTH = 32769;

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


// Shared chatMessage test fixtures
export const chatMessageFixtures: ModelTestFixtures<ChatMessageCreateInput, ChatMessageUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            config: messageConfigFixtures.minimal,
            chatConnect: validIds.chatId1,
            userConnect: validIds.userId1,
            versionIndex: 0,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id2,
            versionIndex: 1,
            config: messageConfigFixtures.complete,
            chatConnect: validIds.chatId2,
            userConnect: validIds.userId1,
            translationsCreate: [{
                id: validIds.id3,
                language: "en",
                text: "Hello, how are you?",
            }],
        },
        update: {
            id: validIds.id2,
            config: messageConfigFixtures.variants.assistantWithTools,
            translationsUpdate: [{
                id: validIds.id3,
                language: "en",
                text: "Updated message text",
            }],
        },
    },
    invalidConfigs: {
        missingVersion: {
            create: {
                id: validIds.id1,
                config: messageConfigFixtures.invalid.missingVersion as MessageConfigObject,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                versionIndex: 0,
            },
        },
        invalidVersion: {
            create: {
                id: validIds.id2,
                config: messageConfigFixtures.invalid.invalidVersion as MessageConfigObject,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                versionIndex: 0,
            },
        },
        malformedStructure: {
            create: {
                id: validIds.id3,
                config: messageConfigFixtures.invalid.malformedStructure as MessageConfigObject,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                versionIndex: 0,
            },
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing id, config, and chatConnect
                versionIndex: 0,
            } as ChatMessageCreateInput,
            update: {
                // Missing id
                config: messageConfigFixtures.minimal,
            } as ChatMessageUpdateInput,
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                versionIndex: "not-a-number", // Should be number
                config: "not-an-object", // Should be object
                chatConnect: 456, // Should be string
            } as unknown as ChatMessageCreateInput,
            update: {
                id: validIds.id1,
                config: false, // Should be object
            } as unknown as ChatMessageUpdateInput,
        },
        invalidConfig: {
            create: {
                id: validIds.id1,
                config: {
                    // Missing required __version and resources
                    role: "user",
                },
                chatConnect: validIds.chatId1,
            } as ChatMessageCreateInput,
        },
        invalidRole: {
            create: {
                id: validIds.id1,
                config: {
                    __version: "1.0.0",
                    resources: [],
                    role: "invalid_role", // Should be one of user, assistant, system, tool
                } as unknown as MessageConfigObject,
                chatConnect: validIds.chatId1,
            } as ChatMessageCreateInput,
        },
        invalidVersionIndex: {
            create: {
                id: validIds.id1,
                versionIndex: -1, // Should be >= 0
                config: messageConfigFixtures.minimal,
                chatConnect: validIds.chatId1,
            } as ChatMessageCreateInput,
        },
        invalidToolCall: {
            create: {
                id: validIds.id1,
                config: messageConfigFixtures.invalid.invalidTypes as MessageConfigObject,
                chatConnect: validIds.chatId1,
            } as ChatMessageCreateInput,
        },
        invalidToolResult: {
            create: {
                id: validIds.id1,
                config: {
                    ...messageConfigFixtures.minimal,
                    toolCalls: [
                        createFailedToolCall("test_function", { param: "value" }, "CODE", "msg"),
                    ],
                },
                chatConnect: validIds.chatId1,
            } as ChatMessageCreateInput,
        },
        invalidTranslationText: {
            create: {
                id: validIds.id1,
                config: messageConfigFixtures.minimal,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                versionIndex: 0,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        text: "", // Empty text should fail
                    },
                ],
            } as ChatMessageCreateInput,
        },
        textTooLong: {
            create: {
                id: validIds.id1,
                config: messageConfigFixtures.minimal,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                versionIndex: 0,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        text: "x".repeat(TEXT_TOO_LONG_LENGTH), // Over max length - should fail validation
                    },
                ],
            } as ChatMessageCreateInput,
        },
    },
    edgeCases: {
        maxVersionIndex: {
            create: {
                id: validIds.id1,
                versionIndex: 999999,
                config: messageConfigFixtures.minimal,
                chatConnect: validIds.chatId1,
            },
        },
        zeroVersionIndex: {
            create: {
                id: validIds.id1,
                versionIndex: 0,
                config: messageConfigFixtures.minimal,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
            },
        },
        allRoles: [
            {
                id: validIds.id1,
                config: messageConfigFixtures.variants.userMessage,
                chatConnect: validIds.chatId1,
            },
            {
                id: validIds.id2,
                config: messageConfigFixtures.variants.assistantWithTools,
                chatConnect: validIds.chatId1,
            },
            {
                id: validIds.id3,
                config: messageConfigFixtures.variants.systemMessage,
                chatConnect: validIds.chatId1,
            },
            {
                id: validIds.id1,
                config: messageConfigFixtures.variants.toolErrorMessage,
                chatConnect: validIds.chatId1,
            },
        ],
        toolCallWithSuccessResult: {
            create: {
                id: validIds.id1,
                config: {
                    ...messageConfigFixtures.minimal,
                    toolCalls: [
                        createSuccessfulToolCall("test_function", { param: "value" }, JSON.stringify({ data: "complex output" })),
                    ],
                },
                chatConnect: validIds.chatId1,
            },
        },
        toolCallWithErrorResult: {
            create: {
                id: validIds.id1,
                config: {
                    ...messageConfigFixtures.minimal,
                    toolCalls: [
                        createFailedToolCall("test_function", { param: "value" }, "TOOL_ERROR", "Tool execution failed"),
                    ],
                },
                chatConnect: validIds.chatId1,
            },
        },
        toolCallWithoutResult: {
            create: {
                id: validIds.id1,
                config: {
                    ...messageConfigFixtures.minimal,
                    toolCalls: [
                        {
                            id: validIds.toolCallId1,
                            function: {
                                name: "test_function",
                                arguments: JSON.stringify({ param: "value" }),
                            },
                            // No result field - should be optional
                        },
                    ],
                },
                chatConnect: validIds.chatId1,
            },
        },
        maxLengthText: {
            create: {
                id: validIds.id1,
                config: messageConfigFixtures.minimal,
                chatConnect: validIds.chatId1,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        text: "x".repeat(TEXT_MAX_LENGTH), // Exactly at max length
                    },
                ],
            },
        },
        complexConfig: {
            create: {
                id: validIds.id1,
                config: messageConfigFixtures.complete,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
            },
        },
        messageWithMultipleRuns: {
            create: {
                id: validIds.id1,
                config: messageConfigFixtures.variants.messageWithMultipleRuns,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                versionIndex: 0,
            },
        },
        broadcastMessage: {
            create: {
                id: validIds.id2,
                config: messageConfigFixtures.variants.broadcastMessage,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                versionIndex: 0,
            },
        },
        eventDrivenMessage: {
            create: {
                id: validIds.id3,
                config: messageConfigFixtures.variants.eventDrivenMessage,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                versionIndex: 0,
            },
        },
    },
};

// Custom factory that always generates valid IDs
const customizers: {
    create: (base: ChatMessageCreateInput) => ChatMessageCreateInput;
    update: (base: ChatMessageUpdateInput) => ChatMessageUpdateInput;
} = {
    create: (base: ChatMessageCreateInput): ChatMessageCreateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
    update: (base: ChatMessageUpdateInput): ChatMessageUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const chatMessageTestDataFactory = new TypedTestDataFactory(chatMessageFixtures, chatMessageValidation, customizers);

// Export typed fixtures with optional validation
export const typedChatMessageFixtures = createTypedFixtures(chatMessageFixtures, chatMessageValidation);

// Legacy export for backward compatibility - the factory already extends TestDataFactory
// so this is just an alias to maintain existing code that uses TestDataFactory
export const chatMessageTestDataFactoryLegacy = chatMessageTestDataFactory as TestDataFactory<ChatMessageCreateInput, ChatMessageUpdateInput>;
