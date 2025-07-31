// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: replaced any with PrismaClient type
import { type chat, type Prisma, type PrismaClient } from "@prisma/client";
import { chatConfigFixtures } from "@vrooli/shared/test-fixtures/config";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type {
    DbTestFixtures,
    RelationConfig,
    TestScenario,
} from "./types.js";

interface ChatRelationConfig extends RelationConfig {
    withCreator?: { userId: bigint };
    withTeam?: { teamId: bigint };
    participants?: Array<{ userId: bigint; hasUnread?: boolean }>;
    messages?: number;
    invites?: Array<{ userId: bigint; status?: "Pending" | "Accepted" | "Declined" }>;
    translations?: Array<{ language: string; name?: string; description?: string }>;
}

/**
 * Enhanced database fixture factory for Chat model
 * Provides comprehensive testing capabilities for chat systems
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for private and public chats
 * - Team and user ownership
 * - Participant and invite management
 * - Message threading support
 * - Translation support
 * - Predefined test scenarios
 */
export class ChatDbFactory extends EnhancedDatabaseFactory<
    chat,
    Prisma.chatCreateInput,
    Prisma.chatInclude,
    Prisma.chatUpdateInput
> {
    protected scenarios: Record<string, TestScenario> = {};
    constructor(prisma: PrismaClient) {
        super("chat", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.chat;
    }

    /**
     * Get complete test fixtures for Chat model
     */
    protected getFixtures(): DbTestFixtures<Prisma.chatCreateInput, Prisma.chatUpdateInput> {
        return {
            minimal: {
                id: this.generateId(),
                publicId: this.generatePublicId(),
                isPrivate: true,
                openToAnyoneWithInvite: false,
                config: chatConfigFixtures.minimal as unknown as Prisma.InputJsonValue,
            },
            complete: {
                id: this.generateId(),
                publicId: this.generatePublicId(),
                isPrivate: false,
                openToAnyoneWithInvite: true,
                config: chatConfigFixtures.complete as unknown as Prisma.InputJsonValue,
                translations: {
                    create: [
                        {
                            id: this.generateId(),
                            language: "en",
                            name: "General Discussion",
                            description: "A chat room for general discussions",
                        },
                        {
                            id: this.generateId(),
                            language: "es",
                            name: "Discusión General",
                            description: "Una sala de chat para discusiones generales",
                        },
                    ],
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, publicId
                    isPrivate: true,
                    openToAnyoneWithInvite: false,
                },
                invalidTypes: {
                    id: this.generateId(), // Use valid snowflake ID for type test
                    publicId: 123, // Should be string
                    isPrivate: "yes", // Should be boolean
                    openToAnyoneWithInvite: 1, // Should be boolean
                    config: "not-an-object", // Should be object
                },
                conflictingSettings: {
                    id: this.generateId(),
                    publicId: this.generatePublicId(),
                    isPrivate: true,
                    openToAnyoneWithInvite: true, // Conflict with isPrivate
                    config: {},
                },
                invalidConfig: {
                    id: this.generateId(),
                    publicId: this.generatePublicId(),
                    isPrivate: false,
                    openToAnyoneWithInvite: false,
                    config: chatConfigFixtures.invalid.malformedStructure,
                },
            },
            edgeCases: {
                maxParticipants: {
                    id: this.generateId(),
                    publicId: this.generatePublicId(),
                    isPrivate: false,
                    openToAnyoneWithInvite: true,
                    config: {
                        ...chatConfigFixtures.complete,
                        limits: {
                            ...chatConfigFixtures.complete.limits,
                            maxParticipants: 1000,
                        },
                    } as unknown as Prisma.InputJsonValue,
                },
                unicodeName: {
                    id: this.generateId(),
                    publicId: this.generatePublicId(),
                    isPrivate: false,
                    openToAnyoneWithInvite: false,
                    config: chatConfigFixtures.minimal as unknown as Prisma.InputJsonValue,
                    translations: {
                        create: [{
                            id: this.generateId(),
                            language: "en",
                            name: "チャット 💬 🌍",
                            description: "Unicode test chat with emojis",
                        }],
                    },
                },
                teamOnlyChat: {
                    id: this.generateId(),
                    publicId: this.generatePublicId(),
                    isPrivate: true,
                    openToAnyoneWithInvite: false,
                    config: { ...chatConfigFixtures.complete } as unknown as Prisma.InputJsonValue,
                    team: { connect: { id: this.generateId() } },
                },
                publicOpenChat: {
                    id: this.generateId(),
                    publicId: this.generatePublicId(),
                    isPrivate: false,
                    openToAnyoneWithInvite: true,
                    config: chatConfigFixtures.minimal as unknown as Prisma.InputJsonValue,
                },
                highLimitsChat: {
                    id: this.generateId(),
                    publicId: this.generatePublicId(),
                    isPrivate: false,
                    openToAnyoneWithInvite: false,
                    config: chatConfigFixtures.variants.highLimitSwarm,
                },
            },
            updates: {
                minimal: {
                    isPrivate: false,
                },
                complete: {
                    isPrivate: true,
                    openToAnyoneWithInvite: false,
                    config: chatConfigFixtures.complete as unknown as Prisma.InputJsonValue,
                    translations: {
                        update: [{
                            where: { id: this.generateId() },
                            data: {
                                name: "Updated Chat Name",
                                description: "Updated chat description",
                            },
                        }],
                    },
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.chatCreateInput>): Prisma.chatCreateInput {
        return {
            id: this.generateId(),
            publicId: this.generatePublicId(),
            isPrivate: true,
            openToAnyoneWithInvite: false,
            config: chatConfigFixtures.minimal as unknown as Prisma.InputJsonValue,
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.chatCreateInput>): Prisma.chatCreateInput {
        return {
            id: this.generateId(),
            publicId: this.generatePublicId(),
            isPrivate: false,
            openToAnyoneWithInvite: true,
            config: chatConfigFixtures.complete as unknown as Prisma.InputJsonValue,
            translations: {
                create: [
                    {
                        id: this.generateId(),
                        language: "en",
                        name: "General Discussion",
                        description: "A chat room for general discussions",
                    },
                    {
                        id: this.generateId(),
                        language: "es",
                        name: "Discusión General",
                        description: "Una sala de chat para discusiones generales",
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
            privateChat: {
                name: "privateChat",
                description: "Private one-on-one chat",
                config: {
                    overrides: {
                        isPrivate: true,
                        openToAnyoneWithInvite: false,
                    },
                    participants: [
                        { userId: this.generateId() },
                        { userId: this.generateId() },
                    ],
                },
            },
            teamChat: {
                name: "teamChat",
                description: "Team-owned chat room",
                config: {
                    overrides: {
                        isPrivate: true,
                        openToAnyoneWithInvite: false,
                    },
                    withTeam: { teamId: this.generateId() },
                    participants: [
                        { userId: this.generateId() },
                        { userId: this.generateId() },
                        { userId: this.generateId() },
                    ],
                },
            },
            publicCommunity: {
                name: "publicCommunity",
                description: "Public community chat open to all",
                config: {
                    overrides: {
                        isPrivate: false,
                        openToAnyoneWithInvite: true,
                        config: chatConfigFixtures.variants.publicSwarm,
                    },
                    translations: [{
                        language: "en",
                        name: "Public Community",
                        description: "Open discussion for everyone",
                    }],
                },
            },
            supportChat: {
                name: "supportChat",
                description: "Support chat with bot assistance",
                config: {
                    overrides: {
                        isPrivate: true,
                        openToAnyoneWithInvite: false,
                        config: {
                            ...chatConfigFixtures.complete,
                            swarmLeader: "support-bot",
                        } as unknown as Prisma.InputJsonValue,
                    },
                    participants: [
                        { userId: this.generateId() },
                        { userId: this.generateId() },
                    ],
                },
            },
            largeGroupChat: {
                name: "largeGroupChat",
                description: "Large group chat with many participants",
                config: {
                    overrides: {
                        isPrivate: false,
                        openToAnyoneWithInvite: true,
                    },
                    participants: Array.from({ length: 50 }, (_, i) => ({
                        userId: this.generateId(),
                        hasUnread: i % 2 === 0,
                    })),
                },
            },
        };
    }

    protected getDefaultInclude(): Prisma.chatInclude {
        return {
            creator: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                },
            },
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
            participants: {
                select: {
                    id: true,
                    hasUnread: true,
                    user: {
                        select: {
                            id: true,
                            publicId: true,
                            name: true,
                            handle: true,
                        },
                    },
                },
            },
            invites: {
                select: {
                    id: true,
                    status: true,
                    message: true,
                    user: {
                        select: {
                            id: true,
                            publicId: true,
                            name: true,
                            handle: true,
                        },
                    },
                },
            },
            translations: true,
            _count: {
                select: {
                    messages: true,
                    participants: true,
                    invites: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.chatCreateInput,
        config: ChatRelationConfig,
        tx: PrismaClient,
    ): Promise<Prisma.chatCreateInput> {
        const data = { ...baseData };

        // Handle creator
        if (config.withCreator) {
            data.creator = {
                connect: { id: config.withCreator.userId },
            };
        }

        // Handle team
        if (config.withTeam) {
            data.team = {
                connect: { id: config.withTeam.teamId },
            };
        }

        // Handle participants
        if (config.participants && Array.isArray(config.participants)) {
            data.participants = {
                create: config.participants.map(participant => ({
                    id: this.generateId(),
                    userId: participant.userId,
                    hasUnread: participant.hasUnread ?? true,
                })),
            };
        }

        // Handle invites
        if (config.invites && Array.isArray(config.invites)) {
            data.invites = {
                create: config.invites.map(invite => ({
                    id: this.generateId(),
                    userId: invite.userId,
                    status: invite.status || "Pending",
                    message: "You've been invited to join this chat",
                })),
            };
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

    /**
     * Create a private chat between users
     */
    async createPrivateChat(userIds: bigint[]): Promise<chat> {
        return await this.createWithRelations({
            overrides: {
                isPrivate: true,
                openToAnyoneWithInvite: false,
            },
            participants: userIds.map(userId => ({ userId })),
        });
    }

    /**
     * Create a team chat
     */
    async createTeamChat(teamId: bigint, memberIds: bigint[]): Promise<chat> {
        return await this.createWithRelations({
            overrides: {
                isPrivate: true,
                openToAnyoneWithInvite: false,
            },
            withTeam: { teamId },
            participants: memberIds.map(userId => ({ userId })),
        });
    }

    /**
     * Create a public community chat
     */
    async createPublicChat(name: string, description?: string): Promise<chat> {
        return await this.createWithRelations({
            overrides: {
                isPrivate: false,
                openToAnyoneWithInvite: true,
            },
            translations: [{
                language: "en",
                name,
                description,
            }],
        });
    }

    protected async checkModelConstraints(record: chat): Promise<string[]> {
        const violations: string[] = [];

        // Check conflicting privacy settings
        if (record.isPrivate && record.openToAnyoneWithInvite) {
            violations.push("Private chats should not be open to anyone with invite");
        }

        // Check participant count
        const participantCount = await this.prisma.chat_participants.count({
            where: { chatId: record.id },
        });

        if (participantCount === 0) {
            violations.push("Chat should have at least one participant");
        }

        // Check team-owned chats have team members
        if (record.teamId) {
            const teamMembers = await this.prisma.member.findMany({
                where: { teamId: record.teamId },
                select: { userId: true },
            });

            const participants = await this.prisma.chat_participants.findMany({
                where: { chatId: record.id },
                select: { userId: true },
            });

            const participantUserIds = new Set(participants.map(p => p.userId));
            const hasTeamMember = teamMembers.some(m => participantUserIds.has(m.userId));

            if (!hasTeamMember) {
                violations.push("Team chat should have at least one team member as participant");
            }
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            messages: true,
            participants: true,
            invites: true,
            translations: true,
            subscriptions: true,
        };
    }

    protected async deleteRelatedRecords(
        record: chat,
        remainingDepth: number,
        tx: PrismaClient,
        includeOnly?: string[],
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        const shouldDelete = (relation: string) =>
            !includeOnly || includeOnly.includes(relation);

        // Delete in order of dependencies

        // Delete messages
        if (shouldDelete("messages")) {
            await tx.chat_message.deleteMany({
                where: { chatId: record.id },
            });
        }

        // Delete invites
        if (shouldDelete("invites")) {
            await tx.chat_invite.deleteMany({
                where: { chatId: record.id },
            });
        }

        // Delete participants
        if (shouldDelete("participants")) {
            await tx.chat_participants.deleteMany({
                where: { chatId: record.id },
            });
        }

        // Delete subscriptions
        if (shouldDelete("subscriptions")) {
            await tx.notification_subscription.deleteMany({
                where: { chatId: record.id },
            });
        }

        // Delete translations
        if (shouldDelete("translations")) {
            await tx.chat_translation.deleteMany({
                where: { chatId: record.id },
            });
        }
    }

    /**
     * Add participants to an existing chat
     */
    async addParticipants(chatId: string | bigint, userIds: Array<string | bigint>): Promise<void> {
        await this.prisma.chat_participants.createMany({
            data: userIds.map(userId => ({
                id: this.generateId(),
                chatId: typeof chatId === "string" ? BigInt(chatId) : chatId,
                userId: typeof userId === "string" ? BigInt(userId) : userId,
                hasUnread: true,
            })),
            skipDuplicates: true,
        });
    }

    /**
     * Create chat invites
     */
    async createInvites(chatId: string | bigint, userIds: Array<string | bigint>, message?: string): Promise<void> {
        await this.prisma.chat_invite.createMany({
            data: userIds.map(userId => ({
                id: this.generateId(),
                chatId: typeof chatId === "string" ? BigInt(chatId) : chatId,
                userId: typeof userId === "string" ? BigInt(userId) : userId,
                status: "Pending" as const,
                message: message || "You've been invited to join this chat",
            })),
            skipDuplicates: true,
        });
    }

    /**
     * Create different chat scenarios for testing
     */
    async createTestingScenarios(): Promise<{
        privateChat: chat;
        teamChat: chat;
        publicChat: chat;
        supportChat: chat;
    }> {
        const [privateChat, teamChat, publicChat, supportChat] = await Promise.all([
            this.createWithRelations(this.scenarios.privateChat.config),
            this.createWithRelations(this.scenarios.teamChat.config),
            this.createWithRelations(this.scenarios.publicCommunity.config),
            this.createWithRelations(this.scenarios.supportChat.config),
        ]);

        return {
            privateChat,
            teamChat,
            publicChat,
            supportChat,
        };
    }
}

// Export factory creator function
export const createChatDbFactory = (prisma: PrismaClient) =>
    new ChatDbFactory(prisma);

// Export the class for type usage
export { ChatDbFactory as ChatDbFactoryClass };
