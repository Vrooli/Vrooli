import type { ChatCreateInput, ChatInviteCreateInput, ChatMessageUpdateInput, ChatTranslationCreateInput, ChatTranslationUpdateInput, ChatUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TypedTestDataFactory, createTypedFixtures, testValues } from "../../../validation/models/__test/validationTestUtils.js";
import { chatTranslationValidation, chatValidation } from "../../../validation/models/chat.js";
import { messageConfigFixtures } from "../config/messageConfigFixtures.js";

// Magic number constants for testing
const NAME_MAX_LENGTH = 50;
const DESCRIPTION_MAX_LENGTH = 2048;

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    chatId1: "123456789012345678",
    chatId2: "123456789012345679",
    chatId3: "123456789012345680",
    inviteId1: "223456789012345678",
    inviteId2: "223456789012345679",
    messageId1: "323456789012345678",
    messageId2: "323456789012345679",
    teamId1: "423456789012345678",
    participantId1: "523456789012345678",
    participantId2: "523456789012345679",
    userId1: "623456789012345678",
    userId2: "623456789012345679",
};

// Shared chat test fixtures that can be imported by API tests, UI tests, etc.
export const chatFixtures: ModelTestFixtures<ChatCreateInput, ChatUpdateInput> = {
    minimal: {
        create: {
            id: validIds.chatId1,
        },
        update: {
            id: validIds.chatId1,
        },
    },
    complete: {
        create: {
            id: validIds.chatId2,
            openToAnyoneWithInvite: true,
            invitesCreate: [{
                id: validIds.inviteId1,
                message: "Join our chat!",
                chatConnect: validIds.chatId2,
                userConnect: validIds.userId1,
            }],
            messagesCreate: [{
                id: validIds.messageId1,
                config: messageConfigFixtures.minimal,
                chatConnect: validIds.chatId2,
                userConnect: validIds.userId1,
                language: "en",
                text: "Hello world!",
                versionIndex: 0,
            }],
            teamConnect: validIds.teamId1,
            translationsCreate: [{
                id: "723456789012345678",
                language: "en",
                name: "Project Discussion",
                description: "Chat for discussing project updates",
            }],
        },
        update: {
            id: validIds.chatId2,
            openToAnyoneWithInvite: false,
            invitesCreate: [{
                id: validIds.inviteId2,
                message: "Welcome to the team chat",
                chatConnect: validIds.chatId2,
                userConnect: validIds.userId2,
            }],
            invitesUpdate: [{
                id: validIds.inviteId1,
                message: "Updated invite message",
            }],
            invitesDelete: [validIds.inviteId1],
            messagesCreate: [{
                id: validIds.messageId2,
                config: messageConfigFixtures.minimal,
                chatConnect: validIds.chatId2,
                userConnect: validIds.userId2,
                language: "en",
                text: "New message",
                versionIndex: 0,
            }],
            messagesUpdate: [{
                id: validIds.messageId1,
                config: messageConfigFixtures.minimal,
                text: "Updated message content",
            }],
            messagesDelete: [validIds.messageId1],
            participantsDelete: [validIds.participantId1, validIds.participantId2],
            translationsCreate: [{
                id: "823456789012345678",
                language: "es",
                name: "Discusión del Proyecto",
                description: "Chat para discutir actualizaciones del proyecto",
            }],
            translationsUpdate: [{
                id: validIds.chatId3,
                language: "en",
                name: "Updated Project Discussion",
                description: "Updated chat description",
            }],
            translationsDelete: [validIds.chatId3],
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing id
                openToAnyoneWithInvite: true,
            } as ChatCreateInput,
            update: {
                // Missing id
                openToAnyoneWithInvite: false,
            } as ChatUpdateInput,
        },
        invalidTypes: {
            create: {
                id: 123,
                openToAnyoneWithInvite: "yes",
            } as unknown as ChatCreateInput,
            update: {
                id: validIds.chatId1,
                openToAnyoneWithInvite: "no",
                participantsDelete: "all",
            } as unknown as ChatUpdateInput,
        },
        invalidRelations: {
            create: {
                id: validIds.chatId1,
                invitesCreate: [{
                    // Missing required fields for invite
                    message: "Invalid invite",
                } as Partial<ChatInviteCreateInput>],
            } as ChatCreateInput,
            update: {
                id: validIds.chatId1,
                messagesUpdate: [{
                    // Missing id for update
                    text: "Updated content",
                } as Partial<ChatMessageUpdateInput>],
            } as ChatUpdateInput,
        },
        invalidTranslations: {
            create: {
                id: validIds.chatId1,
                translationsCreate: [{
                    // Missing required fields
                    name: "Chat name",
                } as Partial<ChatTranslationCreateInput>],
            } as ChatCreateInput,
            update: {
                id: validIds.chatId1,
                translationsUpdate: [{
                    // Missing id
                    language: "en",
                    name: "Updated name",
                } as Partial<ChatTranslationUpdateInput>],
            } as ChatUpdateInput,
        },
    },
    edgeCases: {
        emptyTranslations: {
            create: {
                id: validIds.chatId1,
                translationsCreate: [],
            },
        },
        multipleTranslations: {
            create: {
                id: validIds.chatId1,
                translationsCreate: [
                    {
                        id: "923456789012345678",
                        language: "en",
                        name: "English Chat",
                        description: "English description",
                    },
                    {
                        id: "923456789012345679",
                        language: "fr",
                        name: "Chat Français",
                        description: "Description française",
                    },
                    {
                        id: "923456789012345680",
                        language: "de",
                        name: "Deutscher Chat",
                        description: "Deutsche Beschreibung",
                    },
                ],
            },
        },
        onlyParticipantDelete: {
            update: {
                id: validIds.chatId1,
                participantsDelete: [validIds.participantId1],
            },
        },
        maxLengthFields: {
            create: {
                id: validIds.chatId1,
                translationsCreate: [{
                    id: "923456789012345681",
                    language: "en",
                    name: testValues.longString(NAME_MAX_LENGTH), // Max length for name field
                    description: testValues.longString(DESCRIPTION_MAX_LENGTH), // Assuming reasonable description length
                }],
            },
        },
    },
};

// Custom factory that always generates valid IDs and required fields with proper typing
const customizers = {
    create: (base: ChatCreateInput): ChatCreateInput => ({
        ...base,
        id: base.id || validIds.chatId1,
    }),
    update: (base: ChatUpdateInput): ChatUpdateInput => ({
        ...base,
        id: base.id || validIds.chatId1,
    }),
};

// Test data factory for generating dynamic test data
export const chatTestDataFactory = new TypedTestDataFactory(chatFixtures, chatValidation, customizers);
export const typedChatFixtures = createTypedFixtures(chatFixtures, chatValidation);

// Export translation fixtures separately for focused testing
export const chatTranslationFixtures: ModelTestFixtures<ChatTranslationCreateInput, ChatTranslationUpdateInput> = {
    minimal: {
        create: {
            id: validIds.chatId3,
            language: "en",
        },
        update: {
            id: validIds.chatId3,
            language: "en",
        },
    },
    complete: {
        create: {
            id: validIds.chatId3,
            language: "en",
            name: "Chat Name",
            description: "Chat Description",
        },
        update: {
            id: validIds.chatId3,
            language: "fr",
            name: "Nom du Chat",
            description: "Description du Chat",
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing id and language
                name: "Chat Name",
            } as ChatTranslationCreateInput,
            update: {
                // Missing id
                language: "en",
                description: "Updated description",
            } as ChatTranslationUpdateInput,
        },
        invalidTypes: {
            create: {
                id: validIds.chatId3,
                language: 123,
                name: true,
            } as unknown as ChatTranslationCreateInput,
            update: {
                id: validIds.chatId3,
                language: "en",
                description: [],
            } as unknown as ChatTranslationUpdateInput,
        },
    },
    edgeCases: {
        // Add edge cases for translation fixtures
        minimalFields: {
            create: {
                id: validIds.chatId3,
                language: "en",
            },
        },
        maxLengthFields: {
            create: {
                id: validIds.chatId3,
                language: "en",
                name: testValues.longString(NAME_MAX_LENGTH),
                description: testValues.longString(DESCRIPTION_MAX_LENGTH),
            },
        },
    },
};

// Export factory for chat translation test data
export const chatTranslationTestDataFactory = new TypedTestDataFactory(chatTranslationFixtures, chatTranslationValidation, {
    create: (base: ChatTranslationCreateInput): ChatTranslationCreateInput => ({
        id: validIds.chatId1,
        language: "en",
        ...base,
    }),
    update: (base: ChatTranslationUpdateInput): ChatTranslationUpdateInput => ({
        id: validIds.chatId1,
        ...base,
    }),
});
