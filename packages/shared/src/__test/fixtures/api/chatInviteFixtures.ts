import type { ChatInviteCreateInput, ChatInviteUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TestDataFactory, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { chatInviteValidation } from "../../../validation/models/chatInvite.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    chatId1: "123456789012345681",
    chatId2: "123456789012345682",
    userId1: "123456789012345683",
    userId2: "123456789012345684",
};

// Shared chatInvite test fixtures
export const chatInviteFixtures: ModelTestFixtures<ChatInviteCreateInput, ChatInviteUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            chatConnect: validIds.chatId1,
            userConnect: validIds.userId1,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id2,
            message: "You've been invited to join this chat!",
            chatConnect: validIds.chatId2,
            userConnect: validIds.userId2,
        },
        update: {
            id: validIds.id2,
            message: "Updated invitation message",
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing id, chatConnect, and userConnect
                message: "Incomplete invite",
            },
            update: {
                // Missing id
                message: "Updated message",
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                message: false, // Should be string
                chatConnect: 456, // Should be string
                userConnect: 789, // Should be string
            },
            update: {
                id: validIds.id3,
                message: 123, // Should be string
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
            },
        },
        invalidChatConnect: {
            create: {
                id: validIds.id1,
                chatConnect: "not-a-valid-snowflake",
                userConnect: validIds.userId1,
            },
        },
        invalidUserConnect: {
            create: {
                id: validIds.id1,
                chatConnect: validIds.chatId1,
                userConnect: "not-a-valid-snowflake",
            },
        },
        missingChatConnect: {
            create: {
                id: validIds.id1,
                userConnect: validIds.userId1,
                // Missing required chatConnect
            },
        },
        missingUserConnect: {
            create: {
                id: validIds.id1,
                chatConnect: validIds.chatId1,
                // Missing required userConnect
            },
        },
    },
    edgeCases: {
        maxLengthMessage: {
            create: {
                id: validIds.id1,
                message: "x".repeat(1000), // Test long message
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
            },
        },
        emptyMessage: {
            create: {
                id: validIds.id1,
                message: "", // Empty string should be removed
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
            },
        },
        whitespaceMessage: {
            create: {
                id: validIds.id1,
                message: "  Invitation message with whitespace  ",
                chatConnect: validIds.chatId1,
                userConnect: validIds.userId1,
            },
        },
        updateWithoutMessage: {
            update: {
                id: validIds.id1,
                // No message field - should be allowed in update
            },
        },
        updateWithEmptyMessage: {
            update: {
                id: validIds.id1,
                message: "", // Empty string should be removed
            },
        },
    },
};

// Custom factory that always generates valid IDs
const customizers = {
    create: (base: ChatInviteCreateInput): ChatInviteCreateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
    update: (base: ChatInviteUpdateInput): ChatInviteUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export factories for creating test data programmatically
export const chatInviteTestDataFactory = new TypedTestDataFactory(chatInviteFixtures, chatInviteValidation, customizers);
export const typedChatInviteFixtures = createTypedFixtures(chatInviteFixtures, chatInviteValidation);
