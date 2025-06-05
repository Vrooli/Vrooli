import { ModelTestFixtures, TestDataFactory, testValues } from "../validationTestUtils.js";

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
            invites: {
                create: [{
                    id: validIds.inviteId1,
                    message: "Join our chat!",
                    user: { connect: { id: validIds.userId1 } },
                }],
            },
            messages: {
                create: [{
                    id: validIds.messageId1,
                    content: "Hello world!",
                    user: { connect: { id: validIds.userId1 } },
                }],
            },
            team: {
                connect: { id: validIds.teamId1 },
            },
            translations: {
                create: [{
                    language: "en",
                    name: "Project Discussion",
                    description: "Chat for discussing project updates",
                }],
            },
        },
        update: {
            id: validIds.chatId2,
            openToAnyoneWithInvite: false,
            invites: {
                create: [{
                    id: validIds.inviteId2,
                    message: "Welcome to the team chat",
                    user: { connect: { id: validIds.userId2 } },
                }],
                update: [{
                    id: validIds.inviteId1,
                    message: "Updated invite message",
                }],
                delete: [validIds.inviteId1],
            },
            messages: {
                create: [{
                    id: validIds.messageId2,
                    content: "New message",
                    user: { connect: { id: validIds.userId2 } },
                }],
                update: [{
                    id: validIds.messageId1,
                    content: "Updated message content",
                }],
                delete: [validIds.messageId1],
            },
            participants: {
                delete: [validIds.participantId1, validIds.participantId2],
            },
            translations: {
                create: [{
                    language: "es",
                    name: "Discusión del Proyecto",
                    description: "Chat para discutir actualizaciones del proyecto",
                }],
                update: [{
                    id: validIds.chatId3,
                    language: "en",
                    name: "Updated Project Discussion",
                    description: "Updated chat description",
                }],
                delete: [validIds.chatId3],
            },
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
                translations: {
                    create: [],
                },
            },
        },
        multipleTranslations: {
            create: {
                id: validIds.chatId1,
                translations: {
                    create: [
                        {
                            language: "en",
                            name: "English Chat",
                            description: "English description",
                        },
                        {
                            language: "fr",
                            name: "Chat Français",
                            description: "Description française",
                        },
                        {
                            language: "de",
                            name: "Deutscher Chat",
                            description: "Deutsche Beschreibung",
                        },
                    ],
                },
            },
        },
        onlyParticipantDelete: {
            update: {
                id: validIds.chatId1,
                participants: {
                    delete: [validIds.participantId1],
                },
            },
        },
        maxLengthFields: {
            create: {
                id: validIds.chatId1,
                translations: {
                    create: [{
                        language: "en",
                        name: testValues.longString(255), // Assuming max length
                        description: testValues.longString(2048), // Assuming reasonable description length
                    }],
                },
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