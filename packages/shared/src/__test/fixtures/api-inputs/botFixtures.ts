import type { BotCreateInput, BotUpdateInput, UserTranslationCreateInput, UserTranslationUpdateInput } from "../../../api/types.js";
import { ModelStrategy } from "../../../shape/configs/base.js";
import { type ModelTestFixtures, TypedTestDataFactory, createTypedFixtures, testValues } from "../../../validation/models/__test/validationTestUtils.js";
import { botTranslationValidation, botValidation } from "../../../validation/models/bot.js";
import { setupFileMock } from "../../mocks/fileMock.js";

// Ensure File mock is available before using it
setupFileMock();

// Helper function to create File objects for testing
function createMockFile(content: string, filename: string, mimeType = "image/png"): File {
    return new File([content], filename, { type: mimeType });
}

// Magic number constants for testing
const NAME_TOO_LONG_LENGTH = 257;
const HANDLE_MAX_LENGTH = 16;
const ARRAY_VALUE_1 = 5;
const ARRAY_VALUE_2 = 6;
const ARRAY_VALUE_3 = 7;
const BIO_MAX_LENGTH = 2048;
const BIO_TOO_LONG_LENGTH = 2049;

// Valid Snowflake IDs for testing
const validIds = {
    id1: "100000000000000001",
    id2: "100000000000000002",
    id3: "100000000000000003",
    id4: "100000000000000004",
};

// Bot test fixtures
export const botFixtures: ModelTestFixtures<BotCreateInput, BotUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            botSettings: {
                __version: "1.0",
            },
            isBotDepictingPerson: false,
            isPrivate: false,
            name: "Test Bot",
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id2,
            bannerImage: createMockFile("fake-banner-image-data", "banner.png", "image/png"),
            botSettings: {
                __version: "1.0",
                modelConfig: {
                    strategy: ModelStrategy.FALLBACK,
                    preferredModel: "gpt-4",
                    offlineOnly: false,
                },
                maxTokens: 2048,
                agentSpec: {
                    goal: "Provide helpful assistance to users",
                    role: "assistant",
                    subscriptions: ["user.message", "system.event"],
                },
            },
            handle: "completebot",
            isBotDepictingPerson: true,
            isPrivate: false,
            name: "Complete Bot",
            profileImage: createMockFile("fake-profile-image-data", "profile.jpg", "image/jpeg"),
            translationsCreate: [
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
        update: {
            id: validIds.id2,
            botSettings: {
                __version: "1.0",
                modelConfig: {
                    strategy: ModelStrategy.QUALITY_FIRST,
                    preferredModel: "claude-3",
                    offlineOnly: false,
                },
                maxTokens: 4096,
            },
            handle: "updatedbot",
            isBotDepictingPerson: false,
            isPrivate: true,
            name: "Updated Bot Name",
            translationsUpdate: [
                {
                    id: "200000000000000001",
                    language: "en",
                    bio: "Updated bot description",
                },
            ],
            translationsCreate: [
                {
                    id: "200000000000000003",
                    language: "fr",
                    bio: "Description du bot mise à jour",
                },
            ],
            translationsDelete: ["200000000000000002"],
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing: id, botSettings, isBotDepictingPerson, name
                handle: "invalidbot",
                isPrivate: false,
            } as BotCreateInput,
            update: {
                // Missing id
                handle: "noIdBot",
                name: "Bot without ID",
            } as BotUpdateInput,
        },
        invalidTypes: {
            create: {
                id: 123,
                botSettings: "not-an-object",
                isBotDepictingPerson: "yes",
                name: true,
            } as unknown as BotCreateInput,
            update: {
                id: true,
                botSettings: [],
                isBotDepictingPerson: 1,
                isPrivate: "false",
            } as unknown as BotUpdateInput,
        },
        invalidHandle: {
            create: {
                id: validIds.id1,
                botSettings: {
                    __version: "1.0",
                },
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
                botSettings: {
                    __version: "1.0",
                },
                isBotDepictingPerson: false,
                name: "", // Empty name
            },
            update: {
                id: validIds.id1,
                name: "N".repeat(NAME_TOO_LONG_LENGTH), // Too long (max 256 chars)
            },
        },
        invalidImage: {
            create: {
                id: validIds.id1,
                botSettings: {
                    __version: "1.0",
                },
                isBotDepictingPerson: false,
                name: "Bot with Invalid Image",
                profileImage: "P".repeat(NAME_TOO_LONG_LENGTH), // Too long (max 256 chars)
            },
            update: {
                id: validIds.id1,
                bannerImage: "B".repeat(NAME_TOO_LONG_LENGTH), // Too long (max 256 chars)
            },
        },
    },
    edgeCases: {
        minimalHandle: {
            create: {
                id: validIds.id1,
                botSettings: {
                    __version: "1.0",
                },
                handle: "bot", // Minimum 3 chars
                isBotDepictingPerson: false,
                name: "Bot", // Name also requires minimum 3 chars
            },
        },
        maximalHandle: {
            create: {
                id: validIds.id1,
                botSettings: {
                    __version: "1.0",
                },
                handle: "b".repeat(HANDLE_MAX_LENGTH), // Maximum 16 chars
                isBotDepictingPerson: false,
                name: "Bot with Max Handle",
            },
        },
        emptyBotSettings: {
            create: {
                id: validIds.id1,
                botSettings: {
                    __version: "1.0",
                }, // Empty object is valid
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
                    arrays: [1, 2, 3, [4, ARRAY_VALUE_1, [ARRAY_VALUE_2, ARRAY_VALUE_3]]],
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
                    __version: "1.0",
                    agentSpec: {
                        goal: "Represent a real person in virtual interactions",
                        role: "representative",
                    },
                },
                isBotDepictingPerson: true,
                name: "Virtual Assistant",
            },
        },
        publicBot: {
            create: {
                id: validIds.id1,
                botSettings: {
                    __version: "1.0",
                },
                isBotDepictingPerson: false,
                isPrivate: false,
                name: "Public Bot",
            },
        },
        privateBot: {
            create: {
                id: validIds.id1,
                botSettings: {
                    __version: "1.0",
                },
                isBotDepictingPerson: false,
                isPrivate: true,
                name: "Private Bot",
            },
        },
        withTranslations: {
            create: {
                id: validIds.id1,
                botSettings: {
                    __version: "1.0",
                },
                isBotDepictingPerson: false,
                name: "Multilingual Bot",
                translationsCreate: [
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
        longBio: {
            create: {
                id: validIds.id1,
                botSettings: {
                    __version: "1.0",
                },
                isBotDepictingPerson: false,
                name: "Bot with Long Bio",
                translationsCreate: [{
                    id: "200000000000000020",
                    language: "en",
                    bio: "B".repeat(BIO_MAX_LENGTH), // Maximum allowed
                }],
            },
        },
    },
};

// Bot translation fixtures
export const botTranslationFixtures: ModelTestFixtures<UserTranslationCreateInput, UserTranslationUpdateInput> = {
    minimal: {
        create: {
            id: "200000000000000001",
            language: "en",
        },
        update: {
            id: "200000000000000001",
            language: "en",
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
            language: "en",
            bio: "Updated bio with new information about features and improvements",
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing id and language
                bio: "Bio without required fields",
            } as UserTranslationCreateInput,
            update: {
                // Missing id
                bio: "Updated bio without id",
            } as UserTranslationUpdateInput,
        },
        invalidTypes: {
            create: {
                id: 123,
                language: true,
                bio: [],
            } as unknown as UserTranslationCreateInput,
            update: {
                id: false,
                bio: 456,
            } as unknown as UserTranslationUpdateInput,
        },
        tooLongBio: {
            create: {
                id: "200000000000000003",
                language: "en",
                bio: "B".repeat(BIO_TOO_LONG_LENGTH), // Too long (max 2048)
            },
            update: {
                id: "200000000000000003",
                bio: "B".repeat(BIO_TOO_LONG_LENGTH), // Too long
            },
        },
    },
    edgeCases: {
        emptyBio: {
            create: {
                id: "200000000000000004",
                language: "en",
                bio: "",
            },
            update: {
                id: "200000000000000004",
                language: "en",
                bio: "",
            },
        },
        maxLengthBio: {
            create: {
                id: "200000000000000005",
                language: "en",
                bio: "B".repeat(BIO_MAX_LENGTH),
            },
            update: {
                id: "200000000000000005",
                language: "en",
                bio: "B".repeat(BIO_MAX_LENGTH),
            },
        },
    },
};

// Custom factory that always generates valid defaults
const customizers = {
    create: (base: BotCreateInput): BotCreateInput => ({
        ...base,
        id: base.id || testValues.snowflakeId(),
        botSettings: base.botSettings !== undefined ? base.botSettings : { __version: "1.0" },
        isBotDepictingPerson: base.isBotDepictingPerson !== undefined ? base.isBotDepictingPerson : false,
        isPrivate: base.isPrivate !== undefined ? base.isPrivate : false,
        name: base.name || testValues.shortString("Bot"),
    }),
    update: (base: BotUpdateInput): BotUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export factories for creating test data programmatically
export const botTestDataFactory = new TypedTestDataFactory(botFixtures, botValidation, customizers);
export const botTranslationTestDataFactory = new TypedTestDataFactory(botTranslationFixtures, botTranslationValidation, {
    create: (base: UserTranslationCreateInput): UserTranslationCreateInput => ({
        id: testValues.snowflakeId(),
        language: "en",
        ...base,
    }),
    update: (base: UserTranslationUpdateInput): UserTranslationUpdateInput => ({
        id: testValues.snowflakeId(),
        ...base,
    }),
});

// Export type-safe fixtures with validation capabilities
export const typedBotFixtures = createTypedFixtures(botFixtures, botValidation);
