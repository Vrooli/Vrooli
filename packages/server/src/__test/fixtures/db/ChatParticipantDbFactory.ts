import { generatePK } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface ChatParticipantRelationConfig extends RelationConfig {
    chat: { chatId: string };
    user: { userId: string };
}

/**
 * Enhanced database fixture factory for ChatParticipant model
 * Provides comprehensive testing capabilities for chat participation systems
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for read/unread status tracking
 * - Bulk participant management
 * - Join/leave operations
 * - Predefined test scenarios
 * - Constraint validation
 */
export class ChatParticipantDbFactory extends EnhancedDatabaseFactory<
    Prisma.chat_participantsCreateInput,
    Prisma.chat_participantsCreateInput,
    Prisma.chat_participantsInclude,
    Prisma.chat_participantsUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('ChatParticipant', prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.chatParticipant;
    }

    /**
     * Get complete test fixtures for ChatParticipant model
     */
    protected getFixtures(): DbTestFixtures<Prisma.ChatParticipantCreateInput, Prisma.ChatParticipantUpdateInput> {
        return {
            minimal: {
                id: generatePK().toString(),
                hasUnread: true,
                chat: { connect: { id: "chat-123" } },
                user: { connect: { id: "user-123" } },
            },
            complete: {
                id: generatePK().toString(),
                hasUnread: false,
                chat: { connect: { id: "chat-456" } },
                user: { connect: { id: "user-456" } },
            },
            invalid: {
                missingRequired: {
                    // Missing id, chat, user
                    hasUnread: true,
                },
                invalidTypes: {
                    id: "not-a-snowflake",
                    hasUnread: "yes", // Should be boolean
                    chat: "not-an-object", // Should be object
                    user: null, // Should be object
                },
                duplicateParticipant: {
                    id: generatePK().toString(),
                    hasUnread: true,
                    chat: { connect: { id: "existing-chat" } },
                    user: { connect: { id: "existing-user" } }, // Assumes this combo exists
                },
            },
            edgeCases: {
                participantWithUnread: {
                    id: generatePK().toString(),
                    hasUnread: true,
                    chat: { connect: { id: "chat-unread" } },
                    user: { connect: { id: "user-unread" } },
                },
                participantAllRead: {
                    id: generatePK().toString(),
                    hasUnread: false,
                    chat: { connect: { id: "chat-read" } },
                    user: { connect: { id: "user-read" } },
                },
                recentlyJoined: {
                    id: generatePK().toString(),
                    hasUnread: true, // New participants typically have unread messages
                    chat: { connect: { id: "chat-new" } },
                    user: { connect: { id: "user-new" } },
                },
                longTimeParticipant: {
                    id: generatePK().toString(),
                    hasUnread: false, // Long-time participants likely caught up
                    chat: { connect: { id: "chat-veteran" } },
                    user: { connect: { id: "user-veteran" } },
                },
            },
            updates: {
                minimal: {
                    hasUnread: false,
                },
                complete: {
                    hasUnread: true,
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.ChatParticipantCreateInput>): Prisma.ChatParticipantCreateInput {
        // Note: chat and user connections must be provided via overrides or config
        return {
            id: generatePK().toString(),
            hasUnread: true,
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.ChatParticipantCreateInput>): Prisma.ChatParticipantCreateInput {
        return {
            id: generatePK().toString(),
            hasUnread: false,
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            activeParticipant: {
                name: "activeParticipant",
                description: "Active participant with no unread messages",
                config: {
                    overrides: {
                        hasUnread: false,
                    },
                    chat: { chatId: "chat-active" },
                    user: { userId: "user-active" },
                },
            },
            inactiveParticipant: {
                name: "inactiveParticipant",
                description: "Inactive participant with unread messages",
                config: {
                    overrides: {
                        hasUnread: true,
                    },
                    chat: { chatId: "chat-inactive" },
                    user: { userId: "user-inactive" },
                },
            },
            newParticipant: {
                name: "newParticipant",
                description: "Newly joined participant",
                config: {
                    overrides: {
                        hasUnread: true,
                    },
                    chat: { chatId: "chat-welcome" },
                    user: { userId: "user-newbie" },
                },
            },
            bulkParticipants: {
                name: "bulkParticipants",
                description: "Multiple participants for group chat",
                config: {
                    overrides: {
                        hasUnread: false,
                    },
                    chat: { chatId: "chat-group" },
                    user: { userId: "user-bulk-1" },
                },
            },
            adminParticipant: {
                name: "adminParticipant",
                description: "Chat administrator participant",
                config: {
                    overrides: {
                        hasUnread: false,
                    },
                    chat: { chatId: "chat-admin" },
                    user: { userId: "user-admin" },
                },
            },
        };
    }

    protected getDefaultInclude(): Prisma.ChatParticipantInclude {
        return {
            chat: {
                select: {
                    id: true,
                    publicId: true,
                    isPrivate: true,
                    translations: {
                        select: {
                            language: true,
                            name: true,
                        },
                    },
                    _count: {
                        select: {
                            participants: true,
                            messages: true,
                        },
                    },
                },
            },
            user: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                    isBot: true,
                    status: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.ChatParticipantCreateInput,
        config: ChatParticipantRelationConfig,
        tx: any
    ): Promise<Prisma.ChatParticipantCreateInput> {
        let data = { ...baseData };

        // Handle chat connection (required)
        if (config.chat) {
            data.chat = {
                connect: { id: config.chat.chatId },
            };
        } else {
            throw new Error('ChatParticipant requires a chat connection');
        }

        // Handle user connection (required)
        if (config.user) {
            data.user = {
                connect: { id: config.user.userId },
            };
        } else {
            throw new Error('ChatParticipant requires a user connection');
        }

        return data;
    }

    /**
     * Add a participant to a chat
     */
    async addParticipant(
        chatId: string,
        userId: string,
        hasUnread: boolean = true
    ): Promise<Prisma.ChatParticipant> {
        return await this.createWithRelations({
            overrides: {
                hasUnread,
            },
            chat: { chatId },
            user: { userId },
        });
    }

    /**
     * Add multiple participants to a chat
     */
    async addBulkParticipants(
        chatId: string,
        userIds: string[],
        hasUnread: boolean = true
    ): Promise<Prisma.ChatParticipant[]> {
        const participants = await Promise.all(
            userIds.map(userId => 
                this.addParticipant(chatId, userId, hasUnread)
            )
        );
        return participants;
    }

    /**
     * Remove a participant from a chat
     */
    async removeParticipant(chatId: string, userId: string): Promise<void> {
        await this.prisma.chatParticipant.deleteMany({
            where: {
                chatId,
                userId,
            },
        });
    }

    /**
     * Mark messages as read for a participant
     */
    async markAsRead(chatId: string, userId: string): Promise<Prisma.ChatParticipant> {
        const participant = await this.prisma.chatParticipant.findFirst({
            where: {
                chatId,
                userId,
            },
        });

        if (!participant) {
            throw new Error('Participant not found');
        }

        return await this.prisma.chatParticipant.update({
            where: { id: participant.id },
            data: { hasUnread: false },
            include: this.getDefaultInclude(),
        });
    }

    /**
     * Mark messages as unread for a participant
     */
    async markAsUnread(chatId: string, userId: string): Promise<Prisma.ChatParticipant> {
        const participant = await this.prisma.chatParticipant.findFirst({
            where: {
                chatId,
                userId,
            },
        });

        if (!participant) {
            throw new Error('Participant not found');
        }

        return await this.prisma.chatParticipant.update({
            where: { id: participant.id },
            data: { hasUnread: true },
            include: this.getDefaultInclude(),
        });
    }

    protected async checkModelConstraints(record: Prisma.ChatParticipant): Promise<string[]> {
        const violations: string[] = [];
        
        // Check for duplicate participant
        const duplicate = await this.prisma.chatParticipant.findFirst({
            where: {
                chatId: record.chatId,
                userId: record.userId,
                id: { not: record.id },
            },
        });
        
        if (duplicate) {
            violations.push('User is already a participant in this chat');
        }

        // Check if chat exists
        const chat = await this.prisma.chat.findUnique({
            where: { id: record.chatId },
        });
        
        if (!chat) {
            violations.push('Chat does not exist');
        }

        // Check if user exists
        const user = await this.prisma.user.findUnique({
            where: { id: record.userId },
        });
        
        if (!user) {
            violations.push('User does not exist');
        }

        // Check if user has active invite (they shouldn't be participant if invite is pending)
        const pendingInvite = await this.prisma.chatInvite.findFirst({
            where: {
                chatId: record.chatId,
                userId: record.userId,
                status: "Pending",
            },
        });
        
        if (pendingInvite) {
            violations.push('User has a pending invite to this chat');
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            // ChatParticipant has no dependent records
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.ChatParticipant,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[]
    ): Promise<void> {
        // ChatParticipant has no dependent records to delete
    }

    /**
     * Get participants with unread messages
     */
    async getParticipantsWithUnread(chatId: string): Promise<Prisma.ChatParticipant[]> {
        return await this.prisma.chatParticipant.findMany({
            where: {
                chatId,
                hasUnread: true,
            },
            include: this.getDefaultInclude(),
        });
    }

    /**
     * Get active participants (those who have read all messages)
     */
    async getActiveParticipants(chatId: string): Promise<Prisma.ChatParticipant[]> {
        return await this.prisma.chatParticipant.findMany({
            where: {
                chatId,
                hasUnread: false,
            },
            include: this.getDefaultInclude(),
        });
    }

    /**
     * Get user's chat participations
     */
    async getUserChats(userId: string, hasUnread?: boolean): Promise<Prisma.ChatParticipant[]> {
        const where: Prisma.ChatParticipantWhereInput = { userId };
        if (hasUnread !== undefined) {
            where.hasUnread = hasUnread;
        }

        return await this.prisma.chatParticipant.findMany({
            where,
            include: this.getDefaultInclude(),
            orderBy: { updatedAt: 'desc' },
        });
    }

    /**
     * Create a chat with initial participants
     */
    async createChatWithParticipants(
        chatId: string,
        participantConfigs: Array<{ userId: string; hasUnread?: boolean }>
    ): Promise<Prisma.ChatParticipant[]> {
        const participants = await Promise.all(
            participantConfigs.map(config => 
                this.addParticipant(
                    chatId,
                    config.userId,
                    config.hasUnread ?? true
                )
            )
        );
        return participants;
    }

    /**
     * Update read status for multiple participants
     */
    async bulkUpdateReadStatus(
        chatId: string,
        updates: Array<{ userId: string; hasUnread: boolean }>
    ): Promise<void> {
        for (const update of updates) {
            if (update.hasUnread) {
                await this.markAsUnread(chatId, update.userId);
            } else {
                await this.markAsRead(chatId, update.userId);
            }
        }
    }

    /**
     * Create test participation scenarios
     */
    async createTestingScenarios(chatId: string, userIds: string[]): Promise<{
        active: Prisma.ChatParticipant[];
        inactive: Prisma.ChatParticipant[];
        mixed: Prisma.ChatParticipant[];
    }> {
        if (userIds.length < 6) {
            throw new Error('Need at least 6 user IDs for test scenarios');
        }

        // All active participants (no unread)
        const active = await this.createChatWithParticipants(
            chatId,
            userIds.slice(0, 2).map(userId => ({ userId, hasUnread: false }))
        );

        // All inactive participants (with unread)
        const inactive = await this.createChatWithParticipants(
            chatId,
            userIds.slice(2, 4).map(userId => ({ userId, hasUnread: true }))
        );

        // Mixed read status
        const mixed = await this.createChatWithParticipants(
            chatId,
            [
                { userId: userIds[4], hasUnread: false },
                { userId: userIds[5], hasUnread: true },
            ]
        );

        return {
            active,
            inactive,
            mixed,
        };
    }
}

// Export factory creator function
export const createChatParticipantDbFactory = (prisma: PrismaClient) => 
    ChatParticipantDbFactory.getInstance('ChatParticipant', prisma);

// Export the class for type usage
export { ChatParticipantDbFactory as ChatParticipantDbFactoryClass };