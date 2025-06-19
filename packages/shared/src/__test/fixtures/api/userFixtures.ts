import type { BotCreateInput, ProfileUpdateInput, UserTranslationCreateInput, UserTranslationUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TestDataFactory, TypedTestDataFactory, createTypedFixtures, testValues } from "../../../validation/models/__test/validationTestUtils.js";
import { userValidation } from "../../../validation/models/user.js";

// Valid Snowflake IDs for testing
const validIds = {
    id1: "300000000000000001",
    id2: "300000000000000002",
    id3: "300000000000000003",
    id4: "300000000000000004",
};

// Theme constants
const validTheme = "light";

// User test fixtures - note that user validation handles both bot creation and profile updates
// BotCreateInput is used for create operations since you can only create bots, not regular users directly
// ProfileUpdateInput is used for update operations (profile updates)
export const userFixtures: ModelTestFixtures<BotCreateInput, ProfileUpdateInput> = {
    minimal: {
        // Create is for bots (can't create non-bot users directly)
        create: {
            id: validIds.id1,
            handle: "testbot123",
            isPrivate: false,
            name: "Test Bot",
            isBotDepictingPerson: false,
            botSettings: {
                __version: "1.0",
            },
        },
        // Update is for profile updates  
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        // Complete bot creation
        create: {
            id: validIds.id2,
            handle: "completebot456",
            isPrivate: false,
            name: "Complete Test Bot",
            isBotDepictingPerson: true,
            botSettings: {
                __version: "1.0",
                model: "gpt-4",
                maxTokens: 4096,
                persona: {
                    creativity: 0.7,
                    bias: "helpful and informative",
                },
            },
            bannerImage: "data:image/png;base64,iVBORw0KGg...",
            profileImage: "data:image/png;base64,iVBORw0KGg...",
            translationsCreate: [
                {
                    id: "400000000000000001",
                    language: "en",
                    bio: "A comprehensive test bot with all features",
                },
                {
                    id: "400000000000000002", 
                    language: "es",
                    bio: "Un bot de prueba integral con todas las características",
                },
            ],
        },
        // Complete profile update
        update: {
            id: validIds.id2,
            handle: "updateduser789",
            name: "Updated User Name",
            theme: validTheme,
            isPrivate: true,
            isPrivateMemberships: true,
            isPrivatePullRequests: false,
            isPrivateResources: true,
            isPrivateResourcesCreated: false,
            isPrivateTeamsCreated: true,
            isPrivateBookmarks: false,
            isPrivateVotes: true,
            bannerImage: "data:image/jpeg;base64,/9j/4AAQSkZJRg...",
            profileImage: "data:image/jpeg;base64,/9j/4AAQSkZJRg...",
            // Bot-specific fields for bot updates (these would be separate from profile updates in practice)
            // isBotDepictingPerson: false, // Not part of ProfileUpdateInput - this would be bot-specific
            // botSettings: { // Not part of ProfileUpdateInput - this would be bot-specific  
            //     __version: "1.0",
            //     model: "claude-3",
            //     maxTokens: 2048,
            //     persona: {
            //         creativity: 0.5,
            //         bias: "balanced and objective",
            //     },
            // },
            translationsUpdate: [
                {
                    id: "400000000000000001",
                    language: "en",
                    bio: "Updated bio text",
                },
            ],
            translationsCreate: [
                {
                    id: "400000000000000003",
                    language: "fr",
                    bio: "Nouvelle biographie",
                },
            ],
            translationsDelete: ["400000000000000002"],
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required fields: handle, isPrivate, name
                isBotDepictingPerson: true,
            },
            update: {
                // Missing id
                handle: "noIdUser",
            },
        },
        invalidTypes: {
            create: {
                handle: 123, // Should be string
                isPrivate: "yes", // Should be boolean
                name: true, // Should be string
            },
            update: {
                id: true, // Should be string
                handle: [], // Should be string
                isPrivate: "false", // Should be boolean
                theme: 123, // Should be string
            },
        },
        invalidHandle: {
            create: {
                id: validIds.id1,
                handle: "ab", // Too short (min 3 chars)
                isPrivate: false,
                name: "Test Bot",
                isBotDepictingPerson: false,
                botSettings: {},
            },
            update: {
                id: validIds.id1,
                handle: "a", // Too short
            },
        },
        tooLongHandle: {
            create: {
                id: validIds.id1,
                handle: "a".repeat(17), // Too long (max 16 chars)
                isPrivate: false,
                name: "Test Bot",
                isBotDepictingPerson: false,
                botSettings: {},
            },
            update: {
                id: validIds.id1,
                handle: "x".repeat(17), // Too long
            },
        },
        invalidTheme: {
            update: {
                id: validIds.id1,
                theme: "neon", // Not a valid theme
            },
        },
        invalidImageFormat: {
            update: {
                id: validIds.id1,
                profileImage: "not-a-valid-image-string", // Invalid image data
            },
        },
        tooLongName: {
            create: {
                id: validIds.id1,
                handle: "validhandle",
                isPrivate: false,
                name: "N".repeat(257), // Too long (max 256 chars)
                isBotDepictingPerson: false,
                botSettings: {},
            },
            update: {
                id: validIds.id1,
                name: "M".repeat(257), // Too long
            },
        },
        tooLongBio: {
            create: {
                id: validIds.id1,
                handle: "biobot",
                isPrivate: false,
                name: "Bio Bot",
                isBotDepictingPerson: false,
                botSettings: {},
                translations: {
                    create: [{
                        language: "en",
                        bio: "B".repeat(2049), // Too long (max 2048 chars)
                    }],
                },
            },
            update: {
                id: validIds.id1,
                translations: {
                    create: [{
                        language: "en",
                        bio: "B".repeat(2049), // Too long
                    }],
                },
            },
        },
    },
    edgeCases: {
        minimalHandle: {
            create: {
                id: validIds.id1,
                handle: "abc", // Minimum 3 chars
                isPrivate: false,
                name: "A",
                isBotDepictingPerson: false,
                botSettings: {},
            },
        },
        maximalHandle: {
            create: {
                id: validIds.id1,
                handle: "h".repeat(16), // Maximum 16 chars
                isPrivate: false,
                name: "Max Handle Bot",
                isBotDepictingPerson: false,
                botSettings: {},
            },
        },
        underscoreHandle: {
            create: {
                id: validIds.id1,
                handle: "user_bot_123", // Letters, numbers, and underscores
                isPrivate: false,
                name: "Underscore Bot",
                isBotDepictingPerson: false,
                botSettings: {},
            },
        },
        emptyTranslations: {
            create: {
                id: validIds.id1,
                handle: "notransbot",
                isPrivate: false,
                name: "No Translations Bot",
                isBotDepictingPerson: false,
                botSettings: {},
                translations: {
                    create: [],
                },
            },
        },
        privateBot: {
            create: {
                id: validIds.id1,
                handle: "privatebot",
                isPrivate: true,
                name: "Private Bot",
                isBotDepictingPerson: false,
                botSettings: {},
            },
        },
        publicBot: {
            create: {
                id: validIds.id1,
                handle: "publicbot",
                isPrivate: false,
                name: "Public Bot",
                isBotDepictingPerson: false,
                botSettings: {},
            },
        },
        allPrivacyFlags: {
            update: {
                id: validIds.id1,
                isPrivate: true,
                isPrivateMemberships: true,
                isPrivatePullRequests: true,
                isPrivateResources: true,
                isPrivateResourcesCreated: true,
                isPrivateTeamsCreated: true,
                isPrivateBookmarks: true,
                isPrivateVotes: true,
            },
        },
        noPrivacyFlags: {
            update: {
                id: validIds.id1,
                isPrivate: false,
                isPrivateMemberships: false,
                isPrivatePullRequests: false,
                isPrivateResources: false,
                isPrivateResourcesCreated: false,
                isPrivateTeamsCreated: false,
                isPrivateBookmarks: false,
                isPrivateVotes: false,
            },
        },
        minimalBotSettings: {
            create: {
                id: validIds.id3,
                handle: "minbot",
                isPrivate: false,
                name: "Min Settings Bot",
                isBotDepictingPerson: false,
                botSettings: {},
            },
        },
        complexBotSettings: {
            create: {
                id: validIds.id4,
                handle: "complexbot",
                isPrivate: false,
                name: "Complex Bot",
                isBotDepictingPerson: false,
                botSettings: {
                    assistantId: "asst_complex123",
                    model: "gpt-4-turbo",
                    temperature: 0.9,
                    maxTokens: 4096,
                    topP: 0.95,
                    frequencyPenalty: 0.5,
                    presencePenalty: 0.5,
                    systemPrompt: "You are a helpful assistant.",
                    customSettings: {
                        feature1: true,
                        feature2: "enabled",
                        nestedConfig: {
                            option1: 42,
                            option2: ["a", "b", "c"],
                        },
                    },
                },
            },
        },
    },
};

// User translation fixtures
export const userTranslationFixtures: ModelTestFixtures<UserTranslationCreateInput, UserTranslationUpdateInput> = {
    minimal: {
        create: {
            id: "400000000000000001",
            language: "en",
        },
        update: {
            id: "400000000000000001",
            language: "en",
        },
    },
    complete: {
        create: {
            id: "400000000000000002", 
            language: "en",
            bio: "A comprehensive bio with detailed information about the user",
        },
        update: {
            id: "400000000000000002",
            language: "en",
            bio: "Updated bio with new information",
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing language and id
                bio: "Bio without required fields",
            },
            update: {
                // Missing id and language
                bio: "Updated bio without required fields",
            },
        },
        invalidTypes: {
            create: {
                id: "400000000000000003",
                language: 123, // Should be string
                bio: true, // Should be string
            },
            update: {
                id: 456, // Should be string
                language: "en",
                bio: [], // Should be string
            },
        },
    },
    edgeCases: {
        emptyBio: {
            create: {
                id: "400000000000000004",
                language: "en",
                bio: "",
            },
        },
        longBio: {
            create: {
                id: "400000000000000005",
                language: "en",
                bio: "B".repeat(2048), // Max length bio
            },
        },
        multipleLanguages: {
            create: {
                id: "400000000000000006",
                language: "fr",
                bio: "Biographie en français",
            },
        },
    },
};

// Schema-specific fixtures
export const emailLogInFixtures = {
    minimal: {
        email: "user@example.com",
    },
    complete: {
        email: "user@example.com",
        password: "SecurePass123!",
        verificationCode: "123456",
    },
    withPasswordOnly: {
        password: "SecurePass123!",
    },
    withVerificationOnly: {
        verificationCode: "123456",
    },
    invalid: {
        invalidEmail: {
            email: "not-an-email",
            password: "SecurePass123!",
        },
        tooLongPassword: {
            email: "user@example.com",
            password: "P".repeat(257), // Too long (max 256)
        },
        tooLongVerificationCode: {
            email: "user@example.com",
            verificationCode: "V".repeat(129), // Too long (max 128)
        },
    },
};

export const userDeleteOneFixtures = {
    valid: {
        password: "CurrentPassword123!",
        deletePublicData: true,
    },
    keepPublicData: {
        password: "CurrentPassword123!",
        deletePublicData: false,
    },
    invalid: {
        missingPassword: {
            deletePublicData: true,
        },
        missingDeleteFlag: {
            password: "CurrentPassword123!",
        },
        tooLongPassword: {
            password: "P".repeat(257), // Too long (max 256),
            deletePublicData: true,
        },
    },
};

export const emailRequestPasswordChangeFixtures = {
    valid: {
        email: "user@example.com",
    },
    invalid: {
        missingEmail: {},
        invalidEmail: {
            email: "not-an-email",
        },
    },
};

export const emailResetPasswordFormFixtures = {
    valid: {
        newPassword: "NewSecure123!",
        confirmNewPassword: "NewSecure123!",
    },
    invalid: {
        missingNewPassword: {
            confirmNewPassword: "NewSecure123!",
        },
        missingConfirmPassword: {
            newPassword: "NewSecure123!",
        },
        passwordMismatch: {
            newPassword: "NewSecure123!",
            confirmNewPassword: "DifferentPass123!",
        },
        weakPassword: {
            newPassword: "weak",
            confirmNewPassword: "weak",
        },
    },
};

export const emailResetPasswordFixtures = {
    validWithId: {
        id: validIds.id1,
        code: "RESET123456",
        newPassword: "NewSecure123!",
    },
    validWithPublicId: {
        publicId: "usrabc123xy", // Valid 10-12 char alphanumeric publicId (lowercase only)
        code: "RESET123456",
        newPassword: "NewSecure123!",
    },
    validWithBoth: {
        id: validIds.id1,
        publicId: "usrabc123xy", // Valid 10-12 char alphanumeric publicId (lowercase only)
        code: "RESET123456",
        newPassword: "NewSecure123!",
    },
    invalid: {
        missingIdentifier: {
            code: "RESET123456",
            newPassword: "NewSecure123!",
        },
        missingCode: {
            id: validIds.id1,
            newPassword: "NewSecure123!",
        },
        missingPassword: {
            id: validIds.id1,
            code: "RESET123456",
        },
        tooLongCode: {
            id: validIds.id1,
            code: "C".repeat(129),
            newPassword: "NewSecure123!",
        },
    },
};

export const validateSessionFixtures = {
    valid: {
        timeZone: "America/New_York",
    },
    otherTimeZones: {
        utc: { timeZone: "UTC" },
        tokyo: { timeZone: "Asia/Tokyo" },
        london: { timeZone: "Europe/London" },
    },
    invalid: {
        missingTimeZone: {},
        emptyTimeZone: { timeZone: "" },
        tooLongTimeZone: { timeZone: "T".repeat(129) },
    },
};

export const switchCurrentAccountFixtures = {
    valid: {
        id: validIds.id1,
    },
    invalid: {
        missingId: {},
        invalidIdType: { id: 123 },
    },
};

export const profileEmailUpdateFixtures = {
    minimal: {
        update: {
            currentPassword: "CurrentPass123!",
        },
    },
    withNewPassword: {
        update: {
            currentPassword: "CurrentPass123!",
            newPassword: "NewPass456!",
        },
    },
    withEmails: {
        update: {
            currentPassword: "CurrentPass123!",
            emails: {
                create: [
                    {
                        id: "500000000000000001",
                        emailAddress: "new@example.com",
                    },
                ],
                delete: ["500000000000000002"],
            },
        },
    },
    complete: {
        update: {
            currentPassword: "CurrentPass123!",
            newPassword: "NewSecurePass789!",
            emails: {
                create: [
                    {
                        id: "500000000000000003",
                        emailAddress: "additional@example.com",
                    },
                ],
                delete: ["500000000000000004", "500000000000000005"],
            },
        },
    },
    invalid: {
        missingPassword: {
            update: {
                newPassword: "NewPass123!",
            },
        },
        weakNewPassword: {
            update: {
                currentPassword: "CurrentPass123!",
                newPassword: "weak",
            },
        },
    },
};

// Custom factory that always generates valid defaults with proper typing
const customizers = {
    create: (base: BotCreateInput): BotCreateInput => ({
        ...base,
        id: base.id || testValues.snowflakeId(),
        handle: base.handle || testValues.handle("bot"),
        isPrivate: base.isPrivate !== undefined ? base.isPrivate : false,
        name: base.name || testValues.shortString("Bot"),
        isBotDepictingPerson: base.isBotDepictingPerson !== undefined ? base.isBotDepictingPerson : false,
        botSettings: base.botSettings !== undefined ? base.botSettings : { __version: "1.0" },
    }),
    update: (base: ProfileUpdateInput): ProfileUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export enhanced type-safe factories
export const userTestDataFactory = new TypedTestDataFactory(userFixtures, userValidation, customizers);
export const userTranslationTestDataFactory = new TestDataFactory(userTranslationFixtures);

// Export type-safe fixtures with validation capabilities
export const typedUserFixtures = createTypedFixtures(userFixtures, userValidation);
