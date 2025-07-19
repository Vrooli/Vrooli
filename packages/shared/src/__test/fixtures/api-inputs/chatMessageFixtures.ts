import type { ChatMessageCreateInput, ChatMessageUpdateInput } from "../../../api/types.js";
import { type MessageConfigObject } from "../../../shape/configs/message.js";
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
            language: "en",
            text: "Hello world",
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
            language: "en",
            text: "Hello, how are you?",
        },
        update: {
            id: validIds.id2,
            config: messageConfigFixtures.variants.assistantWithTools,
            text: "Updated message text",
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
                language: "en",
                text: "", // Empty text should fail
            } as ChatMessageCreateInput,
        },
        textTooLong: {
            create: {
                id: validIds.id1,
                config: messageConfigFixtures.minimal,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                versionIndex: 0,
                language: "en",
                text: "x".repeat(TEXT_TOO_LONG_LENGTH), // Over max length - should fail validation
            } as ChatMessageCreateInput,
        },
        missingConfigVersion: {
            create: {
                id: validIds.id1,
                config: messageConfigFixtures.invalid.missingVersion as MessageConfigObject,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                versionIndex: 0,
                language: "en",
                text: "Test message with missing version config",
            },
        },
        invalidConfigVersion: {
            create: {
                id: validIds.id2,
                config: messageConfigFixtures.invalid.invalidVersion as MessageConfigObject,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                versionIndex: 0,
                language: "en",
                text: "Test message with invalid version config",
            },
        },
        malformedConfigStructure: {
            create: {
                id: validIds.id3,
                config: messageConfigFixtures.invalid.malformedStructure as MessageConfigObject,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                versionIndex: 0,
                language: "en",
                text: "Test message with malformed config",
            },
        },
    },
    edgeCases: {
        maxVersionIndex: {
            create: {
                id: validIds.id1,
                versionIndex: 999999,
                config: messageConfigFixtures.minimal,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                language: "en",
                text: "Test message with max version index",
            },
        },
        zeroVersionIndex: {
            create: {
                id: validIds.id1,
                versionIndex: 0,
                config: messageConfigFixtures.minimal,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                language: "en",
                text: "Test message with zero version index",
            },
        },
        allRoles: [
            {
                id: validIds.id1,
                config: messageConfigFixtures.variants.userMessage,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                language: "en",
                text: "User message test",
            },
            {
                id: validIds.id2,
                config: messageConfigFixtures.variants.assistantWithTools,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                language: "en",
                text: "Assistant message test",
            },
            {
                id: validIds.id3,
                config: messageConfigFixtures.variants.systemMessage,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                language: "en",
                text: "System message test",
            },
            {
                id: validIds.id1,
                config: messageConfigFixtures.variants.toolErrorMessage,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                language: "en",
                text: "Tool error message test",
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
                userConnect: validIds.userId1,
                language: "en",
                text: "Message with successful tool call",
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
                userConnect: validIds.userId1,
                language: "en",
                text: "Message with failed tool call",
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
                userConnect: validIds.userId1,
                language: "en",
                text: "Message with tool call without result",
            },
        },
        maxLengthText: {
            create: {
                id: validIds.id1,
                config: messageConfigFixtures.minimal,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                language: "en",
                text: "x".repeat(TEXT_MAX_LENGTH), // Exactly at max length
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
                language: "en",
                text: "Message with complex config",
            },
        },
        messageWithMultipleRuns: {
            create: {
                id: validIds.id1,
                config: messageConfigFixtures.variants.messageWithMultipleRuns,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                versionIndex: 0,
                language: "en",
                text: "Message with multiple runs",
            },
        },
        broadcastMessage: {
            create: {
                id: validIds.id2,
                config: messageConfigFixtures.variants.broadcastMessage,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                versionIndex: 0,
                language: "en",
                text: "Broadcast message",
            },
        },
        eventDrivenMessage: {
            create: {
                id: validIds.id3,
                config: messageConfigFixtures.variants.eventDrivenMessage,
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
                versionIndex: 0,
                language: "en",
                text: "Event driven message",
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
