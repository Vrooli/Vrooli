import type { BotCreateInput, ProfileUpdateInput, UserTranslationCreateInput, UserTranslationUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TypedTestDataFactory, createTypedFixtures, testValues } from "../../../validation/models/__test/validationTestUtils.js";
import { userTranslationValidation, userValidation } from "../../../validation/models/user.js";
import { setupFileMock } from "../../mocks/fileMock.js";

// Ensure File mock is available before using it
setupFileMock();

// Helper function to create File objects for testing
function createMockFile(content: string, filename: string, mimeType = "image/png"): File {
    return new File([content], filename, { type: mimeType });
}

// Magic number constants for testing
const HANDLE_TOO_LONG_LENGTH = 17;
const NAME_TOO_LONG_LENGTH = 257;
const BIO_TOO_LONG_LENGTH = 2049;
const HANDLE_MAX_LENGTH = 16;
const BIO_MAX_LENGTH = 2048;
const PASSWORD_TOO_LONG_LENGTH = 257;
const VERIFICATION_CODE_TOO_LONG_LENGTH = 129;
const TIMEZONE_TOO_LONG_LENGTH = 129;

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
            },
            bannerImage: createMockFile("fake-user-banner-data", "banner.png", "image/png"),
            profileImage: createMockFile("fake-user-profile-data", "profile.png", "image/png"),
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
            bannerImage: createMockFile("fake-updated-banner-data", "banner.jpg", "image/jpeg"),
            profileImage: createMockFile("fake-updated-profile-data", "profile.jpg", "image/jpeg"),
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
            } as BotCreateInput,
            update: {
                // Missing id
                handle: "noIdUser",
            } as ProfileUpdateInput,
        },
        invalidTypes: {
            create: {
                handle: 123,
                isPrivate: "yes",
                name: true,
            } as unknown as BotCreateInput,
            update: {
                id: true,
                handle: [],
                isPrivate: "false",
                theme: 123,
            } as unknown as ProfileUpdateInput,
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
                handle: "a".repeat(HANDLE_TOO_LONG_LENGTH), // Too long (max 16 chars)
                isPrivate: false,
                name: "Test Bot",
                isBotDepictingPerson: false,
                botSettings: {},
            },
            update: {
                id: validIds.id1,
                handle: "x".repeat(HANDLE_TOO_LONG_LENGTH), // Too long
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
                name: "N".repeat(NAME_TOO_LONG_LENGTH), // Too long (max 256 chars)
                isBotDepictingPerson: false,
                botSettings: {},
            },
            update: {
                id: validIds.id1,
                name: "M".repeat(NAME_TOO_LONG_LENGTH), // Too long
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
                        bio: "B".repeat(BIO_TOO_LONG_LENGTH), // Too long (max 2048 chars)
                    }],
                },
            } as unknown as BotCreateInput,
            update: {
                id: validIds.id1,
                translations: {
                    create: [{
                        language: "en",
                        bio: "B".repeat(BIO_TOO_LONG_LENGTH), // Too long
                    }],
                },
            } as unknown as ProfileUpdateInput,
        },
    },
    edgeCases: {
        minimalHandle: {
            create: {
                id: validIds.id1,
                handle: "abc", // Minimum 3 chars
                isPrivate: false,
                name: "Abc", // Name also requires minimum 3 chars
                isBotDepictingPerson: false,
                botSettings: {},
            },
        },
        maximalHandle: {
            create: {
                id: validIds.id1,
                handle: "h".repeat(HANDLE_MAX_LENGTH), // Maximum 16 chars
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
            } as unknown as BotCreateInput,
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
            } as UserTranslationCreateInput,
            update: {
                // Missing id and language
                bio: "Updated bio without required fields",
            } as UserTranslationUpdateInput,
        },
        invalidTypes: {
            create: {
                id: "400000000000000003",
                language: 123,
                bio: true,
            } as unknown as UserTranslationCreateInput,
            update: {
                id: 456,
                language: "en",
                bio: [],
            } as unknown as UserTranslationUpdateInput,
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
                bio: "B".repeat(BIO_MAX_LENGTH), // Max length bio
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
        create: {
            email: "user@example.com",
        },
    },
    complete: {
        create: {
            email: "user@example.com",
            password: "SecurePass123!",
            verificationCode: "123456",
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing all fields - at least one required
            },
        },
        invalidTypes: {
            create: {
                email: 123, // Should be string
                password: "SecurePass123!",
            },
        },
        invalidEmail: {
            create: {
                email: "not-an-email",
                password: "SecurePass123!",
            },
        },
        tooLongPassword: {
            create: {
                email: "user@example.com",
                password: "P".repeat(PASSWORD_TOO_LONG_LENGTH), // Too long (max 256)
            },
        },
        tooLongVerificationCode: {
            create: {
                email: "user@example.com",
                verificationCode: "V".repeat(VERIFICATION_CODE_TOO_LONG_LENGTH), // Too long (max 128)
            },
        },
    },
};

export const userDeleteOneFixtures = {
    minimal: {
        create: {
            password: "CurrentPassword123!",
            deletePublicData: true,
        },
    },
    complete: {
        create: {
            password: "CurrentPassword123!",
            deletePublicData: false,
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required fields
            },
        },
        invalidTypes: {
            create: {
                password: 123, // Should be string
                deletePublicData: true,
            },
        },
        missingPassword: {
            create: {
                deletePublicData: true,
            },
        },
        missingDeleteFlag: {
            create: {
                password: "CurrentPassword123!",
            },
        },
        tooLongPassword: {
            create: {
                password: "P".repeat(PASSWORD_TOO_LONG_LENGTH), // Too long (max 256),
                deletePublicData: true,
            },
        },
    },
};

export const emailRequestPasswordChangeFixtures = {
    minimal: {
        create: {
            email: "user@example.com",
        },
    },
    complete: {
        create: {
            email: "user@example.com",
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required email field
            },
        },
        invalidTypes: {
            create: {
                email: 123, // Should be string
            },
        },
        invalidEmail: {
            create: {
                email: "not-an-email",
            },
        },
    },
};

export const emailResetPasswordFormFixtures = {
    minimal: {
        create: {
            newPassword: "NewSecure123!",
            confirmNewPassword: "NewSecure123!",
        },
    },
    complete: {
        create: {
            newPassword: "NewSecure123!",
            confirmNewPassword: "NewSecure123!",
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required newPassword field
            },
        },
        invalidTypes: {
            create: {
                newPassword: 123, // Should be string
            },
        },
        weakPassword: {
            create: {
                newPassword: "weak",
                confirmNewPassword: "weak",
            },
        },
        passwordMismatch: {
            create: {
                newPassword: "NewSecure123!",
                confirmNewPassword: "DifferentPass123!",
            },
        },
    },
};

export const emailResetPasswordFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
            code: "RESET123456",
            newPassword: "NewSecure123!",
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            publicId: "usrabc123xy", // Valid 10-12 char alphanumeric publicId (lowercase only)
            code: "RESET123456",
            newPassword: "NewSecure123!",
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required fields
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                code: "RESET123456",
                newPassword: "NewSecure123!",
            },
        },
        missingIdentifier: {
            create: {
                code: "RESET123456",
                newPassword: "NewSecure123!",
            },
        },
        missingCode: {
            create: {
                id: validIds.id1,
                newPassword: "NewSecure123!",
            },
        },
        missingPassword: {
            create: {
                id: validIds.id1,
                code: "RESET123456",
            },
        },
        tooLongCode: {
            create: {
                id: validIds.id1,
                code: "C".repeat(VERIFICATION_CODE_TOO_LONG_LENGTH),
                newPassword: "NewSecure123!",
            },
        },
    },
};

export const validateSessionFixtures = {
    minimal: {
        create: {
            timeZone: "America/New_York",
        },
    },
    complete: {
        create: {
            timeZone: "America/New_York",
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required timeZone field
            },
        },
        invalidTypes: {
            create: {
                timeZone: 123, // Should be string
            },
        },
        emptyTimeZone: {
            create: {
                timeZone: "",
            },
        },
        tooLongTimeZone: {
            create: {
                timeZone: "T".repeat(TIMEZONE_TOO_LONG_LENGTH),
            },
        },
    },
};

export const switchCurrentAccountFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id1,
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required id field
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
            },
        },
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
        missingRequired: {
            update: {
                // Missing required 'id' field
            } as ProfileUpdateInput,
        },
        invalidTypes: {
            update: {
                id: 123, // Should be string
                currentPassword: "CurrentPass123!",
            } as unknown as ProfileUpdateInput,
        },
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
export const userTranslationTestDataFactory = new TypedTestDataFactory(userTranslationFixtures, userTranslationValidation, {
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
export const typedUserFixtures = createTypedFixtures(userFixtures, userValidation);
