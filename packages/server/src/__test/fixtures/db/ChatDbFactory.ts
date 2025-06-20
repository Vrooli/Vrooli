import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";
import { chatConfigFixtures } from "@vrooli/shared/__test/fixtures/config";

interface ChatRelationConfig extends RelationConfig {
    withCreator?: { userId: string };
    withTeam?: { teamId: string };
    participants?: Array<{ userId: string; hasUnread?: boolean }>;
    messages?: number;
    invites?: Array<{ userId: string; status?: "Pending" | "Accepted" | "Declined" }>;
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
    Prisma.chat,
    Prisma.chatCreateInput,
    Prisma.chatInclude,
    Prisma.chatUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('chat', prisma);
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
                id: generatePK().toString(),
                publicId: generatePublicId(),
                isPrivate: true,
                openToAnyoneWithInvite: false,
                config: chatConfigFixtures.minimal,
            },
            complete: {
                id: generatePK().toString(),
                publicId: generatePublicId(),
                isPrivate: false,
                openToAnyoneWithInvite: true,
                config: chatConfigFixtures.complete,
                translations: {
                    create: [
                        {
                            id: generatePK().toString(),
                            language: "en",
                            name: "General Discussion",
                            description: "A chat room for general discussions",
                        },
                        {
                            id: generatePK().toString(),
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
                    id: "not-a-snowflake",
                    publicId: 123, // Should be string
                    isPrivate: "yes", // Should be boolean
                    openToAnyoneWithInvite: 1, // Should be boolean
                    config: "not-an-object", // Should be object
                },
                conflictingSettings: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isPrivate: true,
                    openToAnyoneWithInvite: true, // Conflict with isPrivate
                    config: {},
                },
                invalidConfig: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    openToAnyoneWithInvite: false,
                    config: chatConfigFixtures.invalid.malformedStructure,
                },
            },
            edgeCases: {
                maxParticipants: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    openToAnyoneWithInvite: true,
                    config: {
                        ...chatConfigFixtures.complete,
                        limits: {
                            ...chatConfigFixtures.complete.limits,
                            maxParticipants: 1000,
                        },
                    },
                },
                unicodeName: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    openToAnyoneWithInvite: false,
                    config: chatConfigFixtures.minimal,
                    translations: {
                        create: [{
                            id: generatePK().toString(),
                            language: "en",
                            name: "チャット 💬 🌍",
                            description: "Unicode test chat with emojis",
                        }],
                    },
                },
                teamOnlyChat: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isPrivate: true,
                    openToAnyoneWithInvite: false,
                    config: {
                        ...chatConfigFixtures.complete,
                        teamId: "team-123",
                    },
                },
                publicOpenChat: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    openToAnyoneWithInvite: true,
                    config: chatConfigFixtures.minimal,
                },
                highLimitsChat: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
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
                    config: chatConfigFixtures.complete,
                    translations: {
                        update: [{
                            where: { id: "translation_id" },
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
            id: generatePK().toString(),
            publicId: generatePublicId(),
            isPrivate: true,
            openToAnyoneWithInvite: false,
            config: chatConfigFixtures.minimal,
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.chatCreateInput>): Prisma.chatCreateInput {
        return {
            id: generatePK().toString(),
            publicId: generatePublicId(),
            isPrivate: false,
            openToAnyoneWithInvite: true,
            config: chatConfigFixtures.complete,
            translations: {
                create: [
                    {
                        id: generatePK().toString(),
                        language: "en",
                        name: "General Discussion",
                        description: "A chat room for general discussions",
                    },
                    {
                        id: generatePK().toString(),
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
                        { userId: "user-1" },
                        { userId: "user-2" },
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
                    withTeam: { teamId: "team-123" },
                    participants: [
                        { userId: "team-member-1" },
                        { userId: "team-member-2" },
                        { userId: "team-member-3" },
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
                        },
                    },
                    participants: [
                        { userId: "user-needing-help" },
                        { userId: "support-bot" },
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
                        userId: `participant-${i}`,
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
                    name: true,
                    handle: true,
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
        tx: any
    ): Promise<Prisma.chatCreateInput> {
        let data = { ...baseData };

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
                    id: generatePK().toString(),
                    userId: participant.userId,
                    hasUnread: participant.hasUnread ?? true,
                })),
            };
        }

        // Handle invites
        if (config.invites && Array.isArray(config.invites)) {
            data.invites = {
                create: config.invites.map(invite => ({
                    id: generatePK().toString(),
                    userId: invite.userId,
                    status: invite.status || "Pending",
                    message: `You've been invited to join this chat`,
                })),
            };
        }

        // Handle translations
        if (config.translations && Array.isArray(config.translations)) {
            data.translations = {
                create: config.translations.map(trans => ({
                    id: generatePK().toString(),
                    ...trans,
                })),
            };
        }

        return data;
    }

    /**
     * Create a private chat between users
     */
    async createPrivateChat(userIds: string[]): Promise<Prisma.chat> {
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
    async createTeamChat(teamId: string, memberIds: string[]): Promise<Prisma.chat> {
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
    async createPublicChat(name: string, description?: string): Promise<Prisma.chat> {
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

    protected async checkModelConstraints(record: Prisma.chat): Promise<string[]> {
        const violations: string[] = [];
        
        // Check conflicting privacy settings
        if (record.isPrivate && record.openToAnyoneWithInvite) {
            violations.push('Private chats should not be open to anyone with invite');
        }

        // Check participant count
        const participantCount = await this.prisma.chat_participants.count({
            where: { chatId: record.id },
        });
        
        if (participantCount === 0) {
            violations.push('Chat should have at least one participant');
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
                violations.push('Team chat should have at least one team member as participant');
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
        record: Prisma.Chat,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[]
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        const shouldDelete = (relation: string) => 
            !includeOnly || includeOnly.includes(relation);

        // Delete in order of dependencies
        
        // Delete messages
        if (shouldDelete('messages') && record.messages?.length) {
            await tx.chat_message.deleteMany({
                where: { chatId: record.id },
            });
        }

        // Delete invites
        if (shouldDelete('invites') && record.invites?.length) {
            await tx.chat_invite.deleteMany({
                where: { chatId: record.id },
            });
        }

        // Delete participants
        if (shouldDelete('participants') && record.participants?.length) {
            await tx.chat_participants.deleteMany({
                where: { chatId: record.id },
            });
        }

        // Delete subscriptions
        if (shouldDelete('subscriptions') && record.subscriptions?.length) {
            await tx.notification_subscription.deleteMany({
                where: { chatId: record.id },
            });
        }

        // Delete translations
        if (shouldDelete('translations') && record.translations?.length) {
            await tx.chat_translation.deleteMany({
                where: { chatId: record.id },
            });
        }
    }

    /**
     * Add participants to an existing chat
     */
    async addParticipants(chatId: string, userIds: string[]): Promise<void> {
        await this.prisma.chat_participants.createMany({
            data: userIds.map(userId => ({
                id: generatePK().toString(),
                chatId,
                userId,
                hasUnread: true,
            })),
            skipDuplicates: true,
        });
    }

    /**
     * Create chat invites
     */
    async createInvites(chatId: string, userIds: string[], message?: string): Promise<void> {
        await this.prisma.chat_invite.createMany({
            data: userIds.map(userId => ({
                id: generatePK().toString(),
                chatId,
                userId,
                status: "Pending",
                message: message || "You've been invited to join this chat",
            })),
            skipDuplicates: true,
        });
    }

    /**
     * Create different chat scenarios for testing
     */
    async createTestingScenarios(): Promise<{
        privateChat: Prisma.chat;
        teamChat: Prisma.chat;
        publicChat: Prisma.chat;
        supportChat: Prisma.chat;
    }> {
        const [privateChat, teamChat, publicChat, supportChat] = await Promise.all([
            this.seedScenario('privateChat'),
            this.seedScenario('teamChat'),
            this.seedScenario('publicCommunity'),
            this.seedScenario('supportChat'),
        ]);

        return {
            privateChat: privateChat as unknown as Prisma.chat,
            teamChat: teamChat as unknown as Prisma.chat,
            publicChat: publicChat as unknown as Prisma.chat,
            supportChat: supportChat as unknown as Prisma.chat,
        };
    }
}

// Export factory creator function
export const createChatDbFactory = (prisma: PrismaClient) => 
    ChatDbFactory.getInstance('chat', prisma);

// Export the class for type usage
export { ChatDbFactory as ChatDbFactoryClass };