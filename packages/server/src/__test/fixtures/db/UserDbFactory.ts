/* eslint-disable no-magic-numbers */
// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: replaced any with PrismaClient type
import { type Prisma, type PrismaClient, type user } from "@prisma/client";
import { AccountStatus, generatePublicId, nanoid } from "@vrooli/shared";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type {
    DbTestFixtures,
    RelationConfig,
    TestScenario,
} from "./types.js";

interface UserRelationConfig extends RelationConfig {
    withAuth?: boolean;
    withEmails?: boolean | number;
    teams?: Array<{ teamId: bigint | (() => bigint); role: string }>;
    withBotSettings?: boolean;
    translations?: Array<{ language: string; bio: string }>;
}

/**
 * Enhanced database fixture factory for User model
 * Provides comprehensive testing capabilities for both regular users and bots
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for regular users and bot users
 * - Authentication and email management
 * - Team membership handling
 * - Translation support
 * - Predefined test scenarios
 * - Comprehensive validation
 */
export class UserDbFactory extends EnhancedDatabaseFactory<
    user,
    Prisma.userCreateInput,
    Prisma.userInclude,
    Prisma.userUpdateInput
> {
    protected scenarios: Record<string, TestScenario> = {};
    constructor(prisma: PrismaClient) {
        super("User", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.user;
    }

    /**
     * Get complete test fixtures for User model
     */
    protected getFixtures(): DbTestFixtures<Prisma.userCreateInput, Prisma.userUpdateInput> {
        return {
            minimal: {
                id: this.generateId(),
                publicId: generatePublicId(),
                name: "Test User",
                handle: `user_${nanoid()}`,
                status: AccountStatus.Unlocked,
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
            },
            complete: {
                id: this.generateId(),
                publicId: generatePublicId(),
                name: "Complete Test User",
                handle: `complete_${nanoid()}`,
                status: AccountStatus.Unlocked,
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
                theme: "light",
                bannerImage: "https://example.com/banner.jpg",
                profileImage: "https://example.com/profile.jpg",
                translations: {
                    create: [
                        {
                            id: this.generateId(),
                            language: "en",
                            bio: "I'm a test user with a complete profile",
                        },
                        {
                            id: this.generateId(),
                            language: "es",
                            bio: "Soy un usuario de prueba con un perfil completo",
                        },
                    ],
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, publicId, name, handle
                    status: AccountStatus.Unlocked,
                    isBot: false,
                },
                invalidTypes: {
                    id: BigInt(-1), // Invalid bigint ID
                    publicId: 123, // Should be string
                    name: null, // Should be string
                    handle: true, // Should be string
                    isBot: "yes", // Should be boolean
                    status: "unlocked", // Should be AccountStatus enum
                },
                duplicateHandle: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    name: "Duplicate Handle User",
                    handle: "existing_handle", // Assumes this exists
                    status: AccountStatus.Unlocked,
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                },
                invalidBotConfig: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    name: "Invalid Bot",
                    handle: `invalid_bot_${nanoid()}`,
                    status: AccountStatus.Unlocked,
                    isBot: true,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    // Missing botSettings for a bot
                },
            },
            edgeCases: {
                maxLengthHandle: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    name: "Max Length Handle User",
                    handle: "a".repeat(50), // Max length handle
                    status: AccountStatus.Unlocked,
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                },
                unicodeNameUser: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    name: "测试用户 🎉", // Unicode characters
                    handle: `unicode_${nanoid()}`,
                    status: AccountStatus.Unlocked,
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                },
                lockedAccount: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    name: "Locked User",
                    handle: `locked_${nanoid()}`,
                    status: AccountStatus.HardLocked,
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                },
                deletedAccount: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    name: "[Deleted User]",
                    handle: `deleted_${nanoid()}`,
                    status: AccountStatus.Deleted,
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: true,
                },
                botWithComplexSettings: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    name: "Complex Bot",
                    handle: `complex_bot_${nanoid()}`,
                    status: AccountStatus.Unlocked,
                    isBot: true,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    botSettings: JSON.stringify({
                        model: "gpt-4",
                        temperature: 0.7,
                        maxTokens: 4096,
                    }),
                },
                multiLanguageTranslations: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    name: "Multilingual User",
                    handle: `multilang_${nanoid()}`,
                    status: AccountStatus.Unlocked,
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    translations: {
                        create: Array.from({ length: 10 }, (_, i) => ({
                            id: this.generateId(),
                            language: ["en", "es", "fr", "de", "it", "pt", "ru", "ja", "zh", "ar"][i],
                            bio: `Bio in language ${i}`,
                        })),
                    },
                },
            },
            updates: {
                minimal: {
                    name: "Updated Name",
                },
                complete: {
                    name: "Completely Updated User",
                    theme: "dark",
                    bannerImage: "https://example.com/new-banner.jpg",
                    profileImage: "https://example.com/new-profile.jpg",
                    translations: {
                        update: [{
                            where: { id: this.generateId() }, // Use proper bigint ID
                            data: { bio: "Updated bio" },
                        }],
                    },
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.userCreateInput>): Prisma.userCreateInput {
        const uniqueHandle = `user_${nanoid()}`;

        return {
            id: this.generateId(),
            publicId: generatePublicId(),
            name: "Test User",
            handle: uniqueHandle,
            status: AccountStatus.Unlocked,
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: false,
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.userCreateInput>): Prisma.userCreateInput {
        const uniqueHandle = `complete_${nanoid()}`;

        return {
            id: this.generateId(),
            publicId: generatePublicId(),
            name: "Complete Test User",
            handle: uniqueHandle,
            status: AccountStatus.Unlocked,
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: false,
            theme: "light",
            bannerImage: "https://example.com/banner.jpg",
            profileImage: "https://example.com/profile.jpg",
            translations: {
                create: [
                    {
                        id: this.generateId(),
                        language: "en",
                        bio: "I'm a test user with a complete profile",
                    },
                    {
                        id: this.generateId(),
                        language: "es",
                        bio: "Soy un usuario de prueba con un perfil completo",
                    },
                ],
            },
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            admin: {
                name: "admin",
                description: "Admin user with full permissions and team ownership",
                config: {
                    overrides: {
                        name: "Admin User",
                        handle: `admin_${nanoid()}`,
                    },
                    withAuth: true,
                    withEmails: true,
                    teams: [{
                        teamId: () => this.generateId(), // Generate at runtime
                        role: "Owner",
                    }],
                },
            },
            regularMember: {
                name: "regularMember",
                description: "Regular member with basic permissions",
                config: {
                    overrides: {
                        name: "Regular Member",
                    },
                    withAuth: true,
                    withEmails: 1,
                },
            },
            bot: {
                name: "bot",
                description: "Bot user with AI capabilities",
                config: {
                    overrides: {
                        name: "AI Assistant",
                        isBot: true,
                        botSettings: JSON.stringify({
                            model: "gpt-4",
                            temperature: 0.7,
                        }),
                    },
                },
            },
            suspendedUser: {
                name: "suspendedUser",
                description: "User with suspended account",
                config: {
                    overrides: {
                        name: "Suspended User",
                        status: AccountStatus.SoftLocked,
                    },
                    withAuth: true,
                },
            },
            enterpriseUser: {
                name: "enterpriseUser",
                description: "User with enterprise features and multiple teams",
                config: {
                    overrides: {
                        name: "Enterprise User",
                    },
                    withAuth: true,
                    withEmails: 3,
                    teams: [
                        { teamId: () => this.generateId(), role: "Owner" },
                        { teamId: () => this.generateId(), role: "Admin" },
                        { teamId: () => this.generateId(), role: "Member" },
                    ],
                },
            },
        };
    }

    /**
     * Create a bot user with settings
     */
    async createBot(overrides?: Partial<Prisma.userCreateInput>): Promise<user> {
        const botHandle = `bot_${nanoid()}`;

        const data: Prisma.userCreateInput = {
            ...this.generateMinimalData(),
            handle: botHandle,
            name: "Test Bot",
            isBot: true,
            botSettings: JSON.stringify({
                model: "gpt-4",
                temperature: 0.7,
            }),
            ...overrides,
        };

        return await this.createMinimal(data);
    }

    /**
     * Create specific user personas
     */
    async createAdminUser(): Promise<user> {
        const scenario = this.scenarios["admin"];
        if (!scenario) {
            throw new Error("Admin scenario not found");
        }
        return await this.createWithRelations(scenario.config as UserRelationConfig);
    }

    async createSuspendedUser(): Promise<user> {
        return await this.createMinimal({
            status: AccountStatus.SoftLocked,
            handle: `suspended_${nanoid()}`,
        });
    }

    async createPrivateUser(): Promise<user> {
        return await this.createMinimal({
            isPrivate: true,
            handle: `private_${nanoid()}`,
        });
    }

    async createBotDepictingPerson(): Promise<user> {
        return await this.createBot({
            isBotDepictingPerson: true,
            handle: `bot_person_${nanoid()}`,
            name: "AI Assistant (depicts real person)",
        });
    }

    protected getDefaultInclude(): Prisma.userInclude {
        return {
            emails: true,
            auths: true,
            translations: true,
            memberships: {
                select: {
                    id: true,
                    isAdmin: true,
                    permissions: true,
                    team: {
                        select: {
                            id: true,
                            publicId: true,
                            handle: true,
                            translations: {
                                select: {
                                    language: true,
                                    name: true,
                                },
                            },
                        },
                    },
                },
            },
            _count: {
                select: {
                    emails: true,
                    auths: true,
                    memberships: true,
                    comments: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.userCreateInput,
        config: UserRelationConfig,
        tx: PrismaClient,
    ): Promise<Prisma.userCreateInput> {
        const data = { ...baseData };

        // Handle authentication
        if (config.withAuth) {
            data.auths = {
                create: [{
                    id: this.generateId(),
                    provider: "Password",
                    hashed_password: "$2b$10$dummy.hashed.password.for.testing",
                }],
            };
        }

        // Handle emails
        if (config.withEmails) {
            const emailCount = typeof config.withEmails === "number" ? config.withEmails : 1;
            data.emails = {
                create: Array.from({ length: emailCount }, (_, i) => ({
                    id: this.generateId(),
                    emailAddress: `test_${nanoid()}_${i}@example.com`,
                    verifiedAt: i === 0 ? new Date() : null, // First email is verified
                })),
            };
        }

        // Handle team memberships
        if (config.teams && Array.isArray(config.teams)) {
            data.memberships = {
                create: config.teams.map(team => {
                    const teamId = typeof team.teamId === "function" ? team.teamId() : team.teamId;
                    return {
                        id: this.generateId(),
                        publicId: generatePublicId(),
                        team: {
                            connect: { id: teamId },
                        },
                        isAdmin: team.role === "Owner" || team.role === "Admin",
                        permissions: team.role === "Owner" ? JSON.stringify({
                            canInvite: true,
                            canEdit: true,
                            canDelete: true,
                            canManageRoles: true,
                        }) : team.role === "Admin" ? JSON.stringify({
                            canInvite: true,
                            canEdit: true,
                        }) : "{}",
                    };
                }),
            };
        }

        // Handle bot settings
        if (config.withBotSettings && !data.isBot) {
            data.isBot = true;
            data.botSettings = JSON.stringify({
                model: "gpt-4",
                temperature: 0.7,
            });
        }

        // Handle translations
        if (config.translations && Array.isArray(config.translations)) {
            data.translations = {
                create: config.translations.map(trans => ({
                    id: this.generateId(),
                    ...trans,
                })),
            };
        }

        return data;
    }

    protected async checkModelConstraints(record: user): Promise<string[]> {
        const violations: string[] = [];

        // Check handle uniqueness
        if (record.handle) {
            const duplicate = await this.prisma.user.findFirst({
                where: {
                    handle: record.handle,
                    id: { not: record.id },
                },
            });
            if (duplicate) {
                violations.push("Handle must be unique");
            }
        }

        // Check handle format
        if (record.handle && !/^[a-zA-Z0-9_]+$/.test(record.handle)) {
            violations.push("Handle contains invalid characters");
        }

        // Check bot settings
        if (record.isBot && !record.botSettings) {
            violations.push("Bot users must have botSettings");
        }

        // Check email verification
        const emails = await this.prisma.email.findMany({
            where: { userId: record.id },
        });

        if (emails.length > 0 && !emails.some(e => e.verifiedAt !== null)) {
            violations.push("User should have at least one verified email");
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            emails: true,
            auths: true,
            memberships: true,
            translations: true,
            comments: true,
            bookmarkedBy: true,
            chats: true,
            sessions: true,
            viewed: true,
            reacted: true,
        };
    }

    protected async deleteRelatedRecords(
        record: user,
        remainingDepth: number,
        tx: PrismaClient,
        includeOnly?: string[],
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        const shouldDelete = (relation: string) =>
            !includeOnly || includeOnly.includes(relation);

        // Delete in order of dependencies

        // Delete views
        if (shouldDelete("viewed")) {
            await tx.view.deleteMany({
                where: { byId: record.id },
            });
        }

        // Delete reactions
        if (shouldDelete("reacted")) {
            await tx.reaction.deleteMany({
                where: { byId: record.id },
            });
        }

        // Delete comments
        if (shouldDelete("comments")) {
            await tx.comment.deleteMany({
                where: { userId: record.id },
            });
        }

        // Delete bookmarks
        if (shouldDelete("bookmarkedBy")) {
            await tx.bookmark.deleteMany({
                where: { byId: record.id },
            });
        }

        // Delete chat participations
        if (shouldDelete("chats")) {
            await tx.chat_participants.deleteMany({
                where: { userId: record.id },
            });
        }

        // Delete team memberships
        if (shouldDelete("memberships")) {
            await tx.member.deleteMany({
                where: { userId: record.id },
            });
        }

        // Delete sessions
        if (shouldDelete("sessions")) {
            await tx.session.deleteMany({
                where: { userId: record.id },
            });
        }

        // Delete translations
        if (shouldDelete("translations")) {
            await tx.user_translation.deleteMany({
                where: { userId: record.id },
            });
        }

        // Delete emails
        if (shouldDelete("emails")) {
            await tx.email.deleteMany({
                where: { userId: record.id },
            });
        }

        // Delete auth records
        if (shouldDelete("auths")) {
            await tx.user_auth.deleteMany({
                where: { userId: record.id },
            });
        }
    }

    /**
     * Create a user with session for testing authenticated requests
     */
    async createWithSession(overrides?: Partial<Prisma.userCreateInput>): Promise<{
        user: user;
        sessionId: bigint;
    }> {
        const user = await this.createWithRelations({
            overrides,
            withAuth: true,
            withEmails: true,
        });

        // Get the user's auth record
        const auth = await this.prisma.user_auth.findFirst({
            where: { user_id: user.id },
        });

        if (!auth) {
            throw new Error("User auth record not found");
        }

        const session = await this.prisma.session.create({
            data: {
                id: this.generateId(),
                user_id: user.id,
                auth_id: auth.id,
                device_info: JSON.stringify({ device: "test_device", browser: "test_browser" }),
                ip_address: "127.0.0.1",
                expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000), // 30 days
            },
        });

        return { user, sessionId: session.id };
    }

    /**
     * Create users with different permission levels
     */
    async createPermissionTestUsers(): Promise<{
        admin: user;
        member: user;
        guest: user;
        suspended: user;
    }> {
        const adminScenario = this.scenarios["admin"];
        const memberScenario = this.scenarios["regularMember"];
        const suspendedScenario = this.scenarios["suspendedUser"];

        const [admin, member, guest, suspended] = await Promise.all([
            this.createWithRelations(adminScenario.config as UserRelationConfig),
            this.createWithRelations(memberScenario.config as UserRelationConfig),
            this.createMinimal({ name: "Guest User" }),
            this.createWithRelations(suspendedScenario.config as UserRelationConfig),
        ]);

        return {
            admin,
            member,
            guest,
            suspended,
        };
    }
}

// Export factory creator function
export const createUserDbFactory = (prisma: PrismaClient) =>
    new UserDbFactory(prisma);

// Export the class for type usage
export { UserDbFactory as UserDbFactoryClass };
