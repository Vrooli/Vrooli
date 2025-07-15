import { AccountStatus, generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { DbTestFixtures, BulkSeedOptions, BulkSeedResult, DbErrorScenarios } from "./types.js";

/**
 * Database fixtures for User model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const userDbIds = {
    user1: generatePK(),
    user2: generatePK(),
    user3: generatePK(),
    bot1: generatePK(),
    bot2: generatePK(),
    auth1: generatePK(),
    auth2: generatePK(),
    auth3: generatePK(),
    email1: generatePK(),
    email2: generatePK(),
    email3: generatePK(),
};

/**
 * Minimal user data for database creation
 */
export const minimalUserDb: Prisma.userCreateInput = {
    id: userDbIds.user1,
    publicId: generatePublicId(),
    name: "Test User",
    handle: `testuser_${nanoid()}`,
    status: AccountStatus.Unlocked,
    isBot: false,
    isBotDepictingPerson: false,
    isPrivate: false,
};

/**
 * User with authentication
 */
export const userWithAuthDb: Prisma.userCreateInput = {
    id: userDbIds.user2,
    publicId: generatePublicId(),
    name: "Authenticated User",
    handle: `authuser_${nanoid()}`,
    status: AccountStatus.Unlocked,
    isBot: false,
    isBotDepictingPerson: false,
    isPrivate: false,
    auths: {
        create: [{
            id: userDbIds.auth1,
            provider: "Password",
            hashed_password: "$2b$10$dummy.hashed.password.for.testing",
        }],
    },
    emails: {
        create: [{
            id: userDbIds.email1,
            emailAddress: "authuser@example.com",
            verifiedAt: new Date(),
        }],
    },
};

/**
 * Complete user with all features
 */
export const completeUserDb: Prisma.userCreateInput = {
    id: userDbIds.user3,
    publicId: generatePublicId(),
    name: "Complete User",
    handle: `complete_${nanoid()}`,
    status: AccountStatus.Unlocked,
    isBot: false,
    isBotDepictingPerson: false,
    isPrivate: false,
    theme: "light",
    bannerImage: "https://example.com/banner.jpg",
    profileImage: "https://example.com/profile.jpg",
    auths: {
        create: [{
            id: userDbIds.auth2,
            provider: "Password",
            hashed_password: "$2b$10$dummy.hashed.password.for.testing",
        }],
    },
    emails: {
        create: [
            {
                id: userDbIds.email2,
                emailAddress: "complete@example.com",
                verifiedAt: new Date(),
            },
            {
                id: userDbIds.email3,
                emailAddress: "complete.secondary@example.com",
                verifiedAt: null,
            },
        ],
    },
    translations: {
        create: [
            {
                id: generatePK(),
                language: "en",
                bio: "I'm a test user with a complete profile",
            },
            {
                id: generatePK(),
                language: "es",
                bio: "Soy un usuario de prueba con un perfil completo",
            },
        ],
    },
};

/**
 * Bot user
 */
export const botUserDb: Prisma.userCreateInput = {
    id: userDbIds.bot1,
    publicId: generatePublicId(),
    name: "Test Bot",
    handle: `testbot_${nanoid()}`,
    status: AccountStatus.Unlocked,
    isBot: true,
    isBotDepictingPerson: false,
    isPrivate: false,
    botSettings: {
        assistantId: "asst_test123",
        model: "gpt-4",
        temperature: 0.7,
    },
};

/**
 * Enhanced test fixtures for User model following standard structure
 */
export const userDbFixtures: DbTestFixtures<Prisma.userCreateInput> = {
    minimal: {
        id: generatePK(),
        publicId: generatePublicId(),
        name: "Test User",
        handle: `testuser_${nanoid()}`,
        status: AccountStatus.Unlocked,
        isBot: false,
        isBotDepictingPerson: false,
        isPrivate: false,
    },
    complete: {
        id: generatePK(),
        publicId: generatePublicId(),
        name: "Complete User",
        handle: `complete_${nanoid()}`,
        status: AccountStatus.Unlocked,
        isBot: false,
        isBotDepictingPerson: false,
        isPrivate: false,
        theme: "light",
        bannerImage: "https://example.com/banner.jpg",
        profileImage: "https://example.com/profile.jpg",
        auths: {
            create: [{
                id: generatePK(),
                provider: "Password",
                hashed_password: "$2b$10$dummy.hashed.password.for.testing",
            }],
        },
        emails: {
            create: [
                {
                    id: generatePK(),
                    emailAddress: "complete@example.com",
                    verifiedAt: new Date(),
                },
                {
                    id: generatePK(),
                    emailAddress: "complete.secondary@example.com",
                    verifiedAt: null,
                },
            ],
        },
        translations: {
            create: [
                {
                    id: generatePK(),
                    language: "en",
                    bio: "I'm a test user with a complete profile",
                },
                {
                    id: generatePK(),
                    language: "es",
                    bio: "Soy un usuario de prueba con un perfil completo",
                },
            ],
        },
    },
    invalid: {
        missingRequired: {
            // Missing required id, name, handle
            status: AccountStatus.Unlocked,
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            name: 123, // Should be string
            handle: true, // Should be string
            status: "InvalidStatus", // Not a valid AccountStatus
            isBot: "yes", // Should be boolean
        },
        duplicateHandle: {
            id: generatePK(),
            publicId: generatePublicId(),
            name: "Duplicate User",
            handle: "duplicate_handle", // Same handle as another user
            status: AccountStatus.Unlocked,
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: false,
        },
        invalidEmail: {
            id: generatePK(),
            publicId: generatePublicId(),
            name: "Invalid Email User",
            handle: `invalid_${nanoid()}`,
            status: AccountStatus.Unlocked,
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: false,
            emails: {
                create: [{
                    id: generatePK(),
                    emailAddress: "not-an-email", // Invalid email format
                    verifiedAt: new Date(),
                }],
            },
        },
    },
    edgeCases: {
        maxLengthHandle: {
            id: generatePK(),
            publicId: generatePublicId(),
            name: "Max Length User",
            handle: "a".repeat(64), // Maximum handle length
            status: AccountStatus.Unlocked,
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: false,
        },
        botWithComplexSettings: {
            id: generatePK(),
            publicId: generatePublicId(),
            name: "Complex Bot",
            handle: `complexbot_${nanoid()}`,
            status: AccountStatus.Unlocked,
            isBot: true,
            isBotDepictingPerson: false,
            isPrivate: false,
            botSettings: {
                assistantId: "asst_complex123",
                model: "gpt-4-turbo",
                temperature: 0.95,
                maxTokens: 4096,
                topP: 0.9,
                frequencyPenalty: 0.1,
                presencePenalty: 0.2,
                systemPrompt: "You are a complex testing bot with advanced settings",
            },
        },
        multiLanguageTranslations: {
            id: generatePK(),
            publicId: generatePublicId(),
            name: "Multilingual User",
            handle: `multilingual_${nanoid()}`,
            status: AccountStatus.Unlocked,
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: false,
            translations: {
                create: [
                    { id: generatePK(), language: "en", bio: "English bio" },
                    { id: generatePK(), language: "es", bio: "Spanish bio" },
                    { id: generatePK(), language: "fr", bio: "French bio" },
                    { id: generatePK(), language: "de", bio: "German bio" },
                    { id: generatePK(), language: "ja", bio: "Japanese bio" },
                ],
            },
        },
        lockedAccount: {
            id: generatePK(),
            publicId: generatePublicId(),
            name: "Locked User",
            handle: `locked_${nanoid()}`,
            status: AccountStatus.HardLocked,
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: true,
        },
        deletedAccount: {
            id: generatePK(),
            publicId: generatePublicId(),
            name: "Deleted User",
            handle: `deleted_${nanoid()}`,
            status: AccountStatus.Deleted,
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: true,
        },
    },
};

/**
 * Enhanced factory for creating user database fixtures
 */
export class UserDbFactory extends EnhancedDbFactory<Prisma.userCreateInput> {
    
    /**
     * Get the test fixtures for User model
     */
    protected getFixtures(): DbTestFixtures<Prisma.userCreateInput> {
        return userDbFixtures;
    }

    /**
     * Get User-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: userDbIds.user1, // Duplicate ID
                    publicId: generatePublicId(),
                    name: "Duplicate ID User",
                    handle: `duplicate_${nanoid()}`,
                    status: AccountStatus.Unlocked,
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Foreign Key User",
                    handle: `fk_${nanoid()}`,
                    status: AccountStatus.Unlocked,
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    // Note: Foreign key violations would be tested through separate member creation
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Check Constraint User",
                    handle: "", // Empty handle violates check constraint
                    status: AccountStatus.Unlocked,
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                },
            },
            validation: {
                requiredFieldMissing: userDbFixtures.invalid.missingRequired,
                invalidDataType: userDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "a".repeat(500), // Name too long
                    handle: `toolong_${nanoid()}`, // Handle too long
                    status: AccountStatus.Unlocked,
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                },
            },
            businessLogic: {
                botWithoutSettings: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Bot Without Settings",
                    handle: `botnone_${nanoid()}`,
                    status: AccountStatus.Unlocked,
                    isBot: true, // Bot without botSettings
                    isBotDepictingPerson: false,
                    isPrivate: false,
                },
                depictingPersonBot: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Depicting Person Bot",
                    handle: `depictbot_${nanoid()}`,
                    status: AccountStatus.Unlocked,
                    isBot: false, // Not a bot but depicting person
                    isBotDepictingPerson: true,  
                    isPrivate: false,
                },
            },
        };
    }

    /**
     * Add authentication to a user fixture
     */
    protected addAuthentication(data: Prisma.userCreateInput): Prisma.userCreateInput {
        const email = `user_${nanoid()}@example.com`;
        return {
            ...data,
            auths: {
                create: [{
                    id: generatePK(),
                    provider: "Password",
                    hashed_password: "$2b$10$dummy.hashed.password.for.testing",
                }],
            },
            emails: {
                create: [{
                    id: generatePK(),
                    emailAddress: email,
                    verifiedAt: new Date(),
                }],
            },
        };
    }

    /**
     * Add team memberships to a user fixture
     * Note: User-team relationships are handled through the member model, not directly on user
     * This method now just returns the data unchanged since memberships must be created separately
     */
    protected addTeamMemberships(data: Prisma.userCreateInput, teams: Array<{ teamId: string; role: string }>): Prisma.userCreateInput {
        // Team memberships cannot be created directly on user - they need to be created
        // as separate member records after the user is created
        console.warn("Team memberships must be created separately through the member model");
        return data;
    }

    /**
     * Add roles to a user fixture
     * Note: User roles are not directly supported in the current schema
     */
    protected addRoles(data: Prisma.userCreateInput, roles: Array<{ id: string; name: string }>): Prisma.userCreateInput {
        // Roles are not directly supported in the user model
        // They would be handled through team memberships or other mechanisms
        return data;
    }

    /**
     * User-specific validation
     */
    protected validateSpecific(data: Prisma.userCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to User
        if (!data.name) errors.push("User name is required");
        if (!data.handle) errors.push("User handle is required");
        if (data.status === undefined) errors.push("User status is required");

        // Check business logic
        if (data.isBot && !data.botSettings) {
            warnings.push("Bot user should have botSettings");
        }

        if (!data.isBot && data.botSettings) {
            warnings.push("Non-bot user should not have botSettings");
        }

        if (data.isBotDepictingPerson && data.isBot) {
            warnings.push("Bot depicting person should not be marked as bot");
        }

        // Check handle format
        if (data.handle && (data.handle.length < 3 || data.handle.length > 64)) {
            errors.push("Handle must be between 3 and 64 characters");
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(overrides?: Partial<Prisma.userCreateInput>): Prisma.userCreateInput {
        const factory = new UserDbFactory();
        return factory.createMinimal(overrides);
    }

    static createWithAuth(overrides?: Partial<Prisma.userCreateInput>): Prisma.userCreateInput {
        try {
            const factory = new UserDbFactory();
            const result = factory.createWithRelationships({ withAuth: true, overrides });
            
            if (!result || !result.data) {
                throw new Error("createWithRelationships did not return expected result structure");
            }
            
            return result.data;
        } catch (error) {
            console.error("Error in UserDbFactory.createWithAuth:", error);
            throw error;
        }
    }

    static createComplete(overrides?: Partial<Prisma.userCreateInput>): Prisma.userCreateInput {
        const factory = new UserDbFactory();
        return factory.createComplete(overrides);
    }

    static createBot(overrides?: Partial<Prisma.userCreateInput>): Prisma.userCreateInput {
        const factory = new UserDbFactory();
        return factory.createEdgeCase("botWithComplexSettings");
    }

    static createWithRoles(
        roles: Array<{ id: string; name: string }>,
        overrides?: Partial<Prisma.userCreateInput>,
    ): Prisma.userCreateInput {
        try {
            const factory = new UserDbFactory();
            const result = factory.createWithRelationships({ withAuth: true, withRoles: roles, overrides });
            
            if (!result || !result.data) {
                throw new Error("createWithRelationships did not return expected result structure");
            }
            
            return result.data;
        } catch (error) {
            console.error("Error in UserDbFactory.createWithRoles:", error);
            throw error;
        }
    }

    static createWithTeams(
        teams: Array<{ teamId: string; role: string }>,
        overrides?: Partial<Prisma.userCreateInput>,
    ): Prisma.userCreateInput {
        try {
            const factory = new UserDbFactory();
            const result = factory.createWithRelationships({ withAuth: true, withTeams: teams, overrides });
            
            if (!result || !result.data) {
                throw new Error("createWithRelationships did not return expected result structure");
            }
            
            return result.data;
        } catch (error) {
            console.error("Error in UserDbFactory.createWithTeams:", error);
            throw error;
        }
    }
}

/**
 * Helper to create a user that matches the shape expected by mockAuthenticatedSession
 */
export function createSessionUser(overrides?: Partial<Prisma.userCreateInput>) {
    const userData = UserDbFactory.createWithAuth(overrides);
    return {
        ...userData,
        lastLoginAttempt: new Date(),
        logInAttempts: 0,
        updatedAt: new Date(),
        languages: ["en"],
        sessions: [],
    };
}

/**
 * Enhanced helper to seed multiple test users with comprehensive options
 */
export async function seedTestUsers(
    prisma: any,
    count = 3,
    options?: BulkSeedOptions,
): Promise<BulkSeedResult<any>> {
    const factory = new UserDbFactory();
    const users = [];
    let authCount = 0;
    let botCount = 0;
    let teamCount = 0;

    for (let i = 0; i < count; i++) {
        const overrides = options?.overrides?.[i] || { name: `Test User ${i + 1}` };
        
        let userData: Prisma.userCreateInput;
        
        if (options?.withAuth) {
            userData = factory.createWithRelationships({ 
                withAuth: true, 
                overrides, 
            }).data;
            authCount++;
        } else {
            userData = factory.createMinimal(overrides);
        }

        // Add team membership if requested
        // Note: Team memberships need to be created separately after user creation
        if (options?.teamId) {
            // This would need to be handled separately through member model creation
            teamCount++;
        }

        users.push(await prisma.user.create({ data: userData }));
    }

    // Add bots if requested
    if (options?.withBots) {
        const bot = await prisma.user.create({
            data: factory.createEdgeCase("botWithComplexSettings"),
        });
        users.push(bot);
        botCount++;
    }

    return {
        records: users,
        summary: {
            total: users.length,
            withAuth: authCount,
            bots: botCount,
            teams: teamCount,
        },
    };
}
