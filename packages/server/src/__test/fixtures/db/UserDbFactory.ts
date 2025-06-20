import { AccountStatus, generatePK, generatePublicId, nanoid, LATEST_CONFIG_VERSION } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";
import { botConfigFixtures } from "@vrooli/shared/__test/fixtures/config";

interface UserRelationConfig extends RelationConfig {
    withAuth?: boolean;
    withEmails?: boolean | number;
    teams?: Array<{ teamId: string; role: string }>;
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
    Prisma.User,
    Prisma.UserCreateInput,
    Prisma.UserInclude,
    Prisma.UserUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('User', prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.user;
    }

    /**
     * Get complete test fixtures for User model
     */
    protected getFixtures(): DbTestFixtures<Prisma.UserCreateInput, Prisma.UserUpdateInput> {
        return {
            minimal: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Test User",
                handle: `user_${nanoid(8)}`,
                status: AccountStatus.Unlocked,
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
            },
            complete: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Complete Test User",
                handle: `complete_${nanoid(8)}`,
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
                    // Missing id, publicId, name, handle
                    status: AccountStatus.Unlocked,
                    isBot: false,
                },
                invalidTypes: {
                    id: "not-a-snowflake",
                    publicId: 123, // Should be string
                    name: null, // Should be string
                    handle: true, // Should be string
                    isBot: "yes", // Should be boolean
                    status: "unlocked", // Should be AccountStatus enum
                },
                duplicateHandle: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Duplicate Handle User",
                    handle: "existing_handle", // Assumes this exists
                    status: AccountStatus.Unlocked,
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                },
                invalidBotConfig: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Invalid Bot",
                    handle: `invalid_bot_${nanoid(8)}`,
                    status: AccountStatus.Unlocked,
                    isBot: true,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    // Missing botSettings for a bot
                },
            },
            edgeCases: {
                maxLengthHandle: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Max Length Handle User",
                    handle: 'a'.repeat(50), // Max length handle
                    status: AccountStatus.Unlocked,
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                },
                unicodeNameUser: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "测试用户 🎉", // Unicode characters
                    handle: `unicode_${nanoid(8)}`,
                    status: AccountStatus.Unlocked,
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                },
                lockedAccount: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Locked User",
                    handle: `locked_${nanoid(8)}`,
                    status: AccountStatus.HardLocked,
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                },
                deletedAccount: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "[Deleted User]",
                    handle: `deleted_${nanoid(8)}`,
                    status: AccountStatus.Deleted,
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: true,
                },
                botWithComplexSettings: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Complex Bot",
                    handle: `complex_bot_${nanoid(8)}`,
                    status: AccountStatus.Unlocked,
                    isBot: true,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    botSettings: {
                        ...botConfigFixtures.complete,
                        maxTokens: 4096,
                        persona: {
                            ...botConfigFixtures.complete.persona,
                            creativity: 0.9,
                            verbosity: 0.1,
                        },
                    },
                },
                multiLanguageTranslations: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Multilingual User",
                    handle: `multilang_${nanoid(8)}`,
                    status: AccountStatus.Unlocked,
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    translations: {
                        create: Array.from({ length: 10 }, (_, i) => ({
                            id: generatePK(),
                            language: ['en', 'es', 'fr', 'de', 'it', 'pt', 'ru', 'ja', 'zh', 'ar'][i],
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
                            where: { id: "translation_id" },
                            data: { bio: "Updated bio" },
                        }],
                    },
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.UserCreateInput>): Prisma.UserCreateInput {
        const uniqueHandle = `user_${nanoid(8)}`;
        
        return {
            id: generatePK(),
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

    protected generateCompleteData(overrides?: Partial<Prisma.UserCreateInput>): Prisma.UserCreateInput {
        const uniqueHandle = `complete_${nanoid(8)}`;
        
        return {
            id: generatePK(),
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
                        handle: `admin_${nanoid(8)}`,
                    },
                    withAuth: true,
                    withEmails: true,
                    teams: [{
                        teamId: "admin-team",
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
                        botSettings: botConfigFixtures.complete,
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
                        { teamId: "enterprise-team-1", role: "Owner" },
                        { teamId: "enterprise-team-2", role: "Admin" },
                        { teamId: "enterprise-team-3", role: "Member" },
                    ],
                },
            },
        };
    }

    /**
     * Create a bot user with settings
     */
    async createBot(overrides?: Partial<Prisma.UserCreateInput>): Promise<Prisma.User> {
        const botHandle = `bot_${nanoid(8)}`;
        
        const data: Prisma.UserCreateInput = {
            ...this.generateMinimalData(),
            handle: botHandle,
            name: "Test Bot",
            isBot: true,
            botSettings: botConfigFixtures.complete,
            ...overrides,
        };
        
        return await this.createMinimal(data);
    }

    /**
     * Create specific user personas
     */
    async createAdminUser(): Promise<Prisma.User> {
        return await this.seedScenario('admin');
    }

    async createSuspendedUser(): Promise<Prisma.User> {
        return await this.createMinimal({
            status: AccountStatus.SoftLocked,
            handle: `suspended_${nanoid(8)}`,
        });
    }

    async createPrivateUser(): Promise<Prisma.User> {
        return await this.createMinimal({
            isPrivate: true,
            handle: `private_${nanoid(8)}`,
        });
    }

    async createBotDepictingPerson(): Promise<Prisma.User> {
        return await this.createBot({
            isBotDepictingPerson: true,
            handle: `bot_person_${nanoid(8)}`,
            name: "AI Assistant (depicts real person)",
        });
    }

    protected getDefaultInclude(): Prisma.UserInclude {
        return {
            emails: true,
            auths: true,
            translations: true,
            teams: {
                select: {
                    id: true,
                    role: true,
                    team: {
                        select: {
                            id: true,
                            publicId: true,
                            name: true,
                        },
                    },
                },
            },
            _count: {
                select: {
                    emails: true,
                    auths: true,
                    teams: true,
                    projects: true,
                    comments: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.UserCreateInput,
        config: UserRelationConfig,
        tx: any
    ): Promise<Prisma.UserCreateInput> {
        let data = { ...baseData };

        // Handle authentication
        if (config.withAuth) {
            data.auths = {
                create: [{
                    id: generatePK(),
                    provider: "Password",
                    hashed_password: "$2b$10$dummy.hashed.password.for.testing",
                }],
            };
        }

        // Handle emails
        if (config.withEmails) {
            const emailCount = typeof config.withEmails === 'number' ? config.withEmails : 1;
            data.emails = {
                create: Array.from({ length: emailCount }, (_, i) => ({
                    id: generatePK(),
                    emailAddress: `test_${nanoid(6)}_${i}@example.com`,
                    verifiedAt: i === 0 ? new Date() : null, // First email is verified
                })),
            };
        }

        // Handle team memberships
        if (config.teams && Array.isArray(config.teams)) {
            data.teams = {
                create: config.teams.map(team => ({
                    id: generatePK(),
                    teamId: team.teamId,
                    role: team.role || 'Member',
                })),
            };
        }

        // Handle bot settings
        if (config.withBotSettings && !data.isBot) {
            data.isBot = true;
            data.botSettings = botConfigFixtures.complete;
        }

        // Handle translations
        if (config.translations && Array.isArray(config.translations)) {
            data.translations = {
                create: config.translations.map(trans => ({
                    id: generatePK(),
                    ...trans,
                })),
            };
        }

        return data;
    }

    protected async checkModelConstraints(record: Prisma.User): Promise<string[]> {
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
                violations.push('Handle must be unique');
            }
        }

        // Check handle format
        if (record.handle && !/^[a-zA-Z0-9_]+$/.test(record.handle)) {
            violations.push('Handle contains invalid characters');
        }

        // Check bot settings
        if (record.isBot && !record.botSettings) {
            violations.push('Bot users must have botSettings');
        }

        // Check email verification
        const emails = await this.prisma.email.findMany({
            where: { userId: record.id },
        });
        
        if (emails.length > 0 && !emails.some(e => e.verifiedAt !== null)) {
            violations.push('User should have at least one verified email');
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            emails: true,
            auths: true,
            teams: true,
            translations: true,
            comments: true,
            projects: true,
            bookmarks: true,
            chats: true,
            sessions: true,
            views: true,
            reactions: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.User,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[]
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        const shouldDelete = (relation: string) => 
            !includeOnly || includeOnly.includes(relation);

        // Delete in order of dependencies
        
        // Delete views
        if (shouldDelete('views') && record.views?.length) {
            await tx.view.deleteMany({
                where: { byId: record.id },
            });
        }

        // Delete reactions
        if (shouldDelete('reactions') && record.reactions?.length) {
            await tx.reaction.deleteMany({
                where: { byId: record.id },
            });
        }

        // Delete comments
        if (shouldDelete('comments') && record.comments?.length) {
            await tx.comment.deleteMany({
                where: { userId: record.id },
            });
        }

        // Delete bookmarks
        if (shouldDelete('bookmarks') && record.bookmarks?.length) {
            await tx.bookmark.deleteMany({
                where: { byId: record.id },
            });
        }

        // Delete chat participations
        if (shouldDelete('chats') && record.chats?.length) {
            await tx.chatParticipant.deleteMany({
                where: { userId: record.id },
            });
        }

        // Delete team memberships
        if (shouldDelete('teams') && record.teams?.length) {
            await tx.member.deleteMany({
                where: { userId: record.id },
            });
        }

        // Delete sessions
        if (shouldDelete('sessions') && record.sessions?.length) {
            await tx.session.deleteMany({
                where: { userId: record.id },
            });
        }

        // Delete translations
        if (shouldDelete('translations') && record.translations?.length) {
            await tx.userTranslation.deleteMany({
                where: { userId: record.id },
            });
        }

        // Delete emails
        if (shouldDelete('emails') && record.emails?.length) {
            await tx.email.deleteMany({
                where: { userId: record.id },
            });
        }

        // Delete auth records
        if (shouldDelete('auths') && record.auths?.length) {
            await tx.auth.deleteMany({
                where: { userId: record.id },
            });
        }
    }

    /**
     * Create a user with session for testing authenticated requests
     */
    async createWithSession(overrides?: Partial<Prisma.UserCreateInput>): Promise<{
        user: Prisma.User;
        sessionToken: string;
    }> {
        const user = await this.createWithRelations({
            overrides,
            withAuth: true,
            withEmails: true,
        });

        const sessionToken = `test_session_${nanoid(32)}`;
        await this.prisma.session.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                token: sessionToken,
                userId: user.id,
                deviceId: `test_device_${nanoid(8)}`,
                expiresAt: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000), // 30 days
            },
        });

        return { user, sessionToken };
    }

    /**
     * Create users with different permission levels
     */
    async createPermissionTestUsers(): Promise<{
        admin: Prisma.User;
        member: Prisma.User;
        guest: Prisma.User;
        suspended: Prisma.User;
    }> {
        const [admin, member, guest, suspended] = await Promise.all([
            this.seedScenario('admin'),
            this.seedScenario('regularMember'),
            this.createMinimal({ name: "Guest User" }),
            this.seedScenario('suspendedUser'),
        ]);

        return {
            admin: admin as unknown as Prisma.User,
            member: member as unknown as Prisma.User,
            guest,
            suspended: suspended as unknown as Prisma.User,
        };
    }
}

// Export factory creator function
export const createUserDbFactory = (prisma: PrismaClient) => 
    UserDbFactory.getInstance('User', prisma);

// Export the class for type usage
export { UserDbFactory as UserDbFactoryClass };