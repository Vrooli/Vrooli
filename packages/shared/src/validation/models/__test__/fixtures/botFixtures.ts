import { ModelTestFixtures, TestDataFactory, testValues } from "../validationTestUtils.js";

// Valid Snowflake IDs for testing
const validIds = {
    id1: "100000000000000001",
    id2: "100000000000000002",
    id3: "100000000000000003",
    id4: "100000000000000004",
};

// Bot test fixtures
export const botFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
            botSettings: {},
            isBotDepictingPerson: false,
            name: "Test Bot",
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id2,
            bannerImage: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==",
            botSettings: {
                model: "gpt-4",
                temperature: 0.7,
                maxTokens: 2048,
                systemPrompt: "You are a helpful assistant.",
                assistantId: "asst_123456789",
                customConfig: {
                    feature1: true,
                    feature2: "enabled",
                    nestedSettings: {
                        option1: 42,
                        option2: ["a", "b", "c"],
                    },
                },
            },
            handle: "completebot",
            isBotDepictingPerson: true,
            isPrivate: false,
            name: "Complete Bot",
            profileImage: "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEAAAAA...",
            translations: {
                create: [
                    {
                        id: "200000000000000001",
                        language: "en",
                        bio: "A sophisticated AI assistant with advanced capabilities",
                    },
                    {
                        id: "200000000000000002",
                        language: "es",
                        bio: "Un asistente de IA sofisticado con capacidades avanzadas",
                    },
                ],
            },
        },
        update: {
            id: validIds.id2,
            botSettings: {
                model: "claude-3",
                temperature: 0.5,
                maxTokens: 4096,
            },
            handle: "updatedbot",
            isBotDepictingPerson: false,
            isPrivate: true,
            name: "Updated Bot Name",
            translations: {
                update: [
                    {
                        id: "200000000000000001",
                        bio: "Updated bot description",
                    },
                ],
                create: [
                    {
                        id: "200000000000000003",
                        language: "fr",
                        bio: "Description du bot mise à jour",
                    },
                ],
                delete: ["200000000000000002"],
            },
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing: id, botSettings, isBotDepictingPerson, name
                handle: "invalidbot",
                isPrivate: false,
            },
            update: {
                // Missing id
                handle: "noIdBot",
                name: "Bot without ID",
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                botSettings: "not-an-object", // Should be object
                isBotDepictingPerson: "yes", // Should be boolean
                name: true, // Should be string
            },
            update: {
                id: true, // Should be string
                botSettings: [], // Should be object
                isBotDepictingPerson: 1, // Should be boolean
                isPrivate: "false", // Should be boolean
            },
        },
        invalidHandle: {
            create: {
                id: validIds.id1,
                botSettings: {},
                handle: "a", // Too short (min 3 chars)
                isBotDepictingPerson: false,
                name: "Bot with Short Handle",
            },
            update: {
                id: validIds.id1,
                handle: "this_is_way_too_long", // Too long (max 16 chars)
            },
        },
        invalidName: {
            create: {
                id: validIds.id1,
                botSettings: {},
                isBotDepictingPerson: false,
                name: "", // Empty name
            },
            update: {
                id: validIds.id1,
                name: "N".repeat(257), // Too long (max 256 chars)
            },
        },
        invalidImage: {
            create: {
                id: validIds.id1,
                botSettings: {},
                isBotDepictingPerson: false,
                name: "Bot with Invalid Image",
                profileImage: "P".repeat(257), // Too long (max 256 chars)
            },
            update: {
                id: validIds.id1,
                bannerImage: "B".repeat(257), // Too long (max 256 chars)
            },
        },
    },
    edgeCases: {
        minimalHandle: {
            create: {
                id: validIds.id1,
                botSettings: {},
                handle: "bot", // Minimum 3 chars
                isBotDepictingPerson: false,
                name: "B",
            },
        },
        maximalHandle: {
            create: {
                id: validIds.id1,
                botSettings: {},
                handle: "b".repeat(16), // Maximum 16 chars
                isBotDepictingPerson: false,
                name: "Bot with Max Handle",
            },
        },
        emptyBotSettings: {
            create: {
                id: validIds.id1,
                botSettings: {}, // Empty object is valid
                isBotDepictingPerson: false,
                name: "Bot with Empty Settings",
            },
        },
        complexBotSettings: {
            create: {
                id: validIds.id1,
                botSettings: {
                    // Very nested structure
                    level1: {
                        level2: {
                            level3: {
                                level4: {
                                    value: "deep",
                                },
                            },
                        },
                    },
                    arrays: [1, 2, 3, [4, 5, [6, 7]]],
                    mixed: {
                        string: "value",
                        number: 42,
                        boolean: true,
                        null: null,
                        array: [1, "two", false],
                    },
                },
                isBotDepictingPerson: false,
                name: "Bot with Complex Settings",
            },
        },
        botDepictingPerson: {
            create: {
                id: validIds.id1,
                botSettings: {
                    personality: "friendly",
                    background: "I am a virtual representation of a real person",
                },
                isBotDepictingPerson: true,
                name: "Virtual Assistant",
            },
        },
        publicBot: {
            create: {
                id: validIds.id1,
                botSettings: {},
                isBotDepictingPerson: false,
                isPrivate: false,
                name: "Public Bot",
            },
        },
        privateBot: {
            create: {
                id: validIds.id1,
                botSettings: {},
                isBotDepictingPerson: false,
                isPrivate: true,
                name: "Private Bot",
            },
        },
        withTranslations: {
            create: {
                id: validIds.id1,
                botSettings: {},
                isBotDepictingPerson: false,
                name: "Multilingual Bot",
                translations: {
                    create: [
                        {
                            id: "200000000000000010",
                            language: "en",
                            bio: "English bio",
                        },
                        {
                            id: "200000000000000011",
                            language: "de",
                            bio: "Deutsche Biografie",
                        },
                        {
                            id: "200000000000000012",
                            language: "ja",
                            bio: "日本語の経歴",
                        },
                    ],
                },
            },
        },
        longBio: {
            create: {
                id: validIds.id1,
                botSettings: {},
                isBotDepictingPerson: false,
                name: "Bot with Long Bio",
                translations: {
                    create: [{
                        id: "200000000000000020",
                        language: "en",
                        bio: "B".repeat(2048), // Maximum allowed
                    }],
                },
            },
        },
    },
};

// Bot translation fixtures
export const botTranslationFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            id: "200000000000000001",
            language: "en",
        },
        update: {
            id: "200000000000000001",
        },
    },
    complete: {
        create: {
            id: "200000000000000002",
            language: "en",
            bio: "A comprehensive bio with detailed information about the bot's capabilities and purpose",
        },
        update: {
            id: "200000000000000002",
            bio: "Updated bio with new information about features and improvements",
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing id and language
                bio: "Bio without required fields",
            },
            update: {
                // Missing id
                bio: "Updated bio without id",
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                language: true, // Should be string
                bio: [], // Should be string
            },
            update: {
                id: false, // Should be string
                bio: 456, // Should be string
            },
        },
        tooLongBio: {
            create: {
                id: "200000000000000003",
                language: "en",
                bio: "B".repeat(2049), // Too long (max 2048)
            },
            update: {
                id: "200000000000000003",
                bio: "B".repeat(2049), // Too long
            },
        },
    },
};

// Custom factory that always generates valid defaults
const customizers = {
    create: (base: any) => ({
        ...base,
        id: base.id || testValues.snowflakeId(),
        botSettings: base.botSettings !== undefined ? base.botSettings : {},
        isBotDepictingPerson: base.isBotDepictingPerson !== undefined ? base.isBotDepictingPerson : false,
        name: base.name || testValues.shortString("Bot"),
    }),
    update: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export factories for creating test data programmatically
export const botTestDataFactory = new TestDataFactory(botFixtures, customizers);
export const botTranslationTestDataFactory = new TestDataFactory(botTranslationFixtures);