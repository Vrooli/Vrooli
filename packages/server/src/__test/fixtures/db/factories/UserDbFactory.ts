import { AccountStatus } from "@vrooli/shared";
import { generatePK, generatePublicId, nanoid } from "../idHelpers.js";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface UserRelationConfig extends RelationConfig {
    withAuth?: boolean;
    withEmails?: boolean | number;
    teams?: Array<{ teamId: string; role: string }>;
    withBotSettings?: boolean;
    translations?: Array<{ language: string; bio: string }>;
}

/**
 * Database fixture factory for User model
 * Handles both regular users and bots with comprehensive relationship support
 */
export class UserDbFactory extends DatabaseFixtureFactory<
    Prisma.user,
    Prisma.userCreateInput,
    Prisma.userInclude,
    Prisma.userUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("user", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.user;
    }

    protected getMinimalData(overrides?: Partial<Prisma.userCreateInput>): Prisma.userCreateInput {
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

    protected getCompleteData(overrides?: Partial<Prisma.userCreateInput>): Prisma.userCreateInput {
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
     * Create a bot user with settings
     */
    async createBot(overrides?: Partial<Prisma.userCreateInput>): Promise<Prisma.user> {
        const botHandle = `bot_${nanoid(8)}`;
        
        const data: Prisma.userCreateInput = {
            ...this.getMinimalData(),
            handle: botHandle,
            name: "Test Bot",
            isBot: true,
            botSettings: {
                model: "gpt-4",
                creativity: 0.7,
                verbosity: 0.5,
                translations: [{
                    language: "en",
                    persona: "I am a helpful bot",
                }],
            },
            ...overrides,
        };
        
        const result = await this.prisma.user.create({ data });
        this.trackCreatedId(result.id);
        return result;
    }

    protected getDefaultInclude(): Prisma.userInclude {
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
        baseData: Prisma.userCreateInput,
        config: UserRelationConfig,
        tx: any,
    ): Promise<Prisma.userCreateInput> {
        const data = { ...baseData };

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
            const emailCount = typeof config.withEmails === "number" ? config.withEmails : 1;
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
            // Note: This creates Member records through the User model
            data.teams = {
                create: config.teams.map(team => ({
                    id: generatePK(),
                    teamId: team.teamId,
                    role: team.role || "Member",
                })),
            };
        }

        // Handle bot settings
        if (config.withBotSettings && !data.isBot) {
            data.isBot = true;
            data.botSettings = {
                model: "gpt-4",
                creativity: 0.7,
                verbosity: 0.5,
                translations: [{
                    language: "en",
                    persona: "I am a helpful bot assistant",
                }],
            };
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

    /**
     * Create test scenarios
     */
    async createAdminUser(): Promise<Prisma.user> {
        return this.createWithRelations({
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
        });
    }

    async createSuspendedUser(): Promise<Prisma.user> {
        return this.createMinimal({
            status: AccountStatus.SoftLocked,
            handle: `suspended_${nanoid(8)}`,
        });
    }

    async createPrivateUser(): Promise<Prisma.user> {
        return this.createMinimal({
            isPrivate: true,
            handle: `private_${nanoid(8)}`,
        });
    }

    async createBotDepictingPerson(): Promise<Prisma.user> {
        return this.createBot({
            isBotDepictingPerson: true,
            handle: `bot_person_${nanoid(8)}`,
            name: "AI Assistant (depicts real person)",
        });
    }

    protected async checkModelConstraints(record: Prisma.user): Promise<string[]> {
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

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
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
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.userCreateInput> {
        return {
            maxLengthHandle: {
                ...this.getMinimalData(),
                handle: "a".repeat(50), // Max length handle
            },
            unicodeNameUser: {
                ...this.getMinimalData(),
                name: "测试用户 🎉", // Unicode characters
                handle: `unicode_${nanoid(8)}`,
            },
            multiLanguageUser: {
                ...this.getCompleteData(),
                handle: `multilang_${nanoid(8)}`,
                translations: {
                    create: Array.from({ length: 10 }, (_, i) => ({
                        id: generatePK(),
                        language: ["en", "es", "fr", "de", "it", "pt", "ru", "ja", "zh", "ar"][i],
                        bio: `Bio in language ${i}`,
                    })),
                },
            },
            botWithComplexSettings: {
                ...this.getMinimalData(),
                handle: `complex_bot_${nanoid(8)}`,
                isBot: true,
                botSettings: {
                    model: "gpt-4",
                    creativity: 0.9,
                    verbosity: 0.1,
                    maxTokens: 4096,
                    translations: [{
                        language: "en",
                        persona: "I am a highly creative but concise bot",
                    }],
                },
            },
        };
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
            // Add other relationships as needed
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.user,
        remainingDepth: number,
        tx: any,
    ): Promise<void> {
        // Delete in order of dependencies
        
        // Delete comments
        if (record.comments?.length) {
            await tx.comment.deleteMany({
                where: { userId: record.id },
            });
        }

        // Delete bookmarks
        if (record.bookmarks?.length) {
            await tx.bookmark.deleteMany({
                where: { userId: record.id },
            });
        }

        // Delete team memberships
        if (record.teams?.length) {
            await tx.member.deleteMany({
                where: { userId: record.id },
            });
        }

        // Delete translations
        if (record.translations?.length) {
            await tx.userTranslation.deleteMany({
                where: { userId: record.id },
            });
        }

        // Delete emails
        if (record.emails?.length) {
            await tx.email.deleteMany({
                where: { userId: record.id },
            });
        }

        // Delete auth records
        if (record.auths?.length) {
            await tx.auth.deleteMany({
                where: { userId: record.id },
            });
        }
    }
}

// Export factory creator function
export const createUserDbFactory = (prisma: PrismaClient) => new UserDbFactory(prisma);
