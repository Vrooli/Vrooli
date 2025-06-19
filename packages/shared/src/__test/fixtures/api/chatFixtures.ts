import { type ModelTestFixtures, TestDataFactory, testValues } from "../../../validation/models/__test/validationTestUtils.js";

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
export const chatFixtures: ModelTestFixtures = {
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
                config: {
                    __version: "1.0.0",
                    resources: [],
                },
                chatConnect: validIds.chatId2,
                userConnect: validIds.userId1,
                translationsCreate: [{
                    id: "1023456789012345678",
                    language: "en",
                    text: "Hello world!",
                }],
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
                config: {
                    __version: "1.0.0",
                    resources: [],
                },
                chatConnect: validIds.chatId2,
                userConnect: validIds.userId2,
                translationsCreate: [{
                    id: "1123456789012345678",
                    language: "en",
                    text: "New message",
                }],
            }],
            messagesUpdate: [{
                id: validIds.messageId1,
                config: {
                    __version: "1.0.0",
                    resources: [],
                },
                translationsCreate: [{
                    id: "1223456789012345678",
                    language: "en",
                    text: "Updated message content",
                }],
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
            },
            update: {
                // Missing id
                openToAnyoneWithInvite: false,
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                openToAnyoneWithInvite: "yes", // Should be boolean
            },
            update: {
                id: validIds.chatId1,
                openToAnyoneWithInvite: "no", // Should be boolean
                participants: {
                    delete: "all", // Should be array
                },
            },
        },
        invalidRelations: {
            create: {
                id: validIds.chatId1,
                invites: {
                    create: [{
                        // Missing required fields for invite
                        message: "Invalid invite",
                    }],
                },
            },
            update: {
                id: validIds.chatId1,
                messages: {
                    update: [{
                        // Missing id for update
                        content: "Updated content",
                    }],
                },
            },
        },
        invalidTranslations: {
            create: {
                id: validIds.chatId1,
                translations: {
                    create: [{
                        // Missing language
                        name: "Chat name",
                    }],
                },
            },
            update: {
                id: validIds.chatId1,
                translations: {
                    update: [{
                        // Missing id
                        language: "en",
                        name: "Updated name",
                    }],
                },
            },
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
                    name: testValues.longString(50), // Max length for name field
                    description: testValues.longString(2048), // Assuming reasonable description length
                }],
            },
        },
    },
};

// Test data factory for generating dynamic test data
export const chatTestDataFactory = new TestDataFactory(chatFixtures);

// Export translation fixtures separately for focused testing
export const chatTranslationFixtures = {
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
            },
            update: {
                // Missing id
                language: "en",
                description: "Updated description",
            },
        },
        invalidTypes: {
            create: {
                id: validIds.chatId3,
                language: 123, // Should be string
                name: true, // Should be string
            },
            update: {
                id: validIds.chatId3,
                language: "en",
                description: [], // Should be string
            },
        },
    },
};
