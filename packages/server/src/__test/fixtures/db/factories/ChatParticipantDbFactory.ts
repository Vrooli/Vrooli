import { generatePK } from "@vrooli/shared";
import { nanoid } from "nanoid";
import { type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface ChatParticipantCreateInput {
    id: bigint;
    chatId: bigint;
    userId: bigint;
    hasUnread?: boolean;
}

interface ChatParticipantRelationConfig extends RelationConfig {
    chatId?: string;
    userId?: string;
    withUnread?: boolean;
    participantRole?: 'member' | 'admin' | 'moderator';
}

/**
 * Database fixture factory for ChatParticipant model
 * Handles user/team/bot participants with roles and unread status
 */
export class ChatParticipantDbFactory extends DatabaseFixtureFactory<
    any,
    ChatParticipantCreateInput,
    any,
    any
> {
    constructor(prisma: PrismaClient) {
        super('chat_participants', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.chat_participants;
    }

    protected getMinimalData(overrides?: Partial<ChatParticipantCreateInput>): ChatParticipantCreateInput {
        // Generate placeholder IDs - these should be overridden with actual chat/user IDs
        const chatId = BigInt(generatePK());
        const userId = BigInt(generatePK());
        
        return {
            id: generatePK(),
            chatId,
            userId,
            hasUnread: false,
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<ChatParticipantCreateInput>): ChatParticipantCreateInput {
        // Generate placeholder IDs - these should be overridden with actual chat/user IDs
        const chatId = BigInt(generatePK());
        const userId = BigInt(generatePK());
        
        return {
            id: generatePK(),
            chatId,
            userId,
            hasUnread: false,
            ...overrides,
        };
    }

    /**
     * Create participant with specific chat and user
     */
    async createParticipant(
        chatId: string, 
        userId: string, 
        overrides?: Partial<ChatParticipantCreateInput>
    ): Promise<any> {
        const data: ChatParticipantCreateInput = {
            ...this.getMinimalData(),
            chatId: BigInt(chatId),
            userId: BigInt(userId),
            ...overrides,
        };
        
        const result = await this.prisma.chat_participants.create({ data });
        this.trackCreatedId(result.id.toString());
        return result;
    }

    /**
     * Create participant with unread messages
     */
    async createWithUnread(
        chatId: string, 
        userId: string, 
        hasUnread: boolean = true
    ): Promise<any> {
        return this.createParticipant(chatId, userId, { hasUnread });
    }

    /**
     * Create admin participant
     */
    async createAdmin(
        chatId: string, 
        userId: string
    ): Promise<any> {
        return this.createParticipant(chatId, userId, {
            hasUnread: false,
            // Note: Admin role would be stored in chat config or separate table
            // For now, we just create a regular participant
        });
    }

    /**
     * Create bot participant
     */
    async createBotParticipant(
        chatId: string, 
        botId: string
    ): Promise<any> {
        return this.createParticipant(chatId, botId, {
            hasUnread: false,
        });
    }

    protected getDefaultInclude(): any {
        return {
            chat: {
                select: {
                    id: true,
                    publicId: true,
                    isPrivate: true,
                    translations: {
                        where: { language: 'en' },
                        select: {
                            name: true,
                            description: true,
                        },
                        take: 1,
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
                    profileImage: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: ChatParticipantCreateInput,
        config: ChatParticipantRelationConfig,
        tx: any
    ): Promise<ChatParticipantCreateInput> {
        let data = { ...baseData };

        // Apply chat and user connections
        if (config.chatId) {
            data.chatId = BigInt(config.chatId);
        }
        
        if (config.userId) {
            data.userId = BigInt(config.userId);
        }

        // Apply unread status
        if (config.withUnread !== undefined) {
            data.hasUnread = config.withUnread;
        }

        return data;
    }

    /**
     * Bulk create participants for a chat
     */
    async addParticipantsToChat(
        chatId: string, 
        userIds: string[], 
        hasUnread: boolean = true
    ): Promise<any[]> {
        const participants: any[] = [];
        
        for (const userId of userIds) {
            const participant = await this.createParticipant(chatId, userId, { hasUnread });
            participants.push(participant);
        }
        
        return participants;
    }

    /**
     * Mark participant as having read messages
     */
    async markAsRead(chatId: string, userId: string): Promise<any> {
        const result = await this.prisma.chat_participants.updateMany({
            where: {
                chatId: BigInt(chatId),
                userId: BigInt(userId),
            },
            data: {
                hasUnread: false,
            },
        });
        
        return result;
    }

    /**
     * Mark participant as having unread messages
     */
    async markAsUnread(chatId: string, userId: string): Promise<any> {
        const result = await this.prisma.chat_participants.updateMany({
            where: {
                chatId: BigInt(chatId),
                userId: BigInt(userId),
            },
            data: {
                hasUnread: true,
            },
        });
        
        return result;
    }

    protected async checkModelConstraints(record: any): Promise<string[]> {
        const violations: string[] = [];
        
        // Check unique constraint (chatId, userId combination)
        const duplicate = await this.prisma.chat_participants.findFirst({
            where: { 
                chatId: record.chatId,
                userId: record.userId,
                id: { not: record.id },
            },
        });
        if (duplicate) {
            violations.push('User can only participate once per chat');
        }

        // Check chat exists
        const chat = await this.prisma.chat.findUnique({
            where: { id: record.chatId },
        });
        if (!chat) {
            violations.push('Referenced chat does not exist');
        }

        // Check user exists
        const user = await this.prisma.user.findUnique({
            where: { id: record.userId },
        });
        if (!user) {
            violations.push('Referenced user does not exist');
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing id, chatId, userId
                hasUnread: false,
            },
            invalidTypes: {
                id: "not-a-snowflake",
                chatId: "invalid-chat-reference", // Should be BigInt
                userId: "invalid-user-reference", // Should be BigInt
                hasUnread: "yes", // Should be boolean
            },
            duplicateParticipant: {
                id: generatePK(),
                chatId: BigInt("123456789012345678"), // Same chat/user combo
                userId: BigInt("987654321098765432"),
                hasUnread: false,
            },
            nonExistentChat: {
                id: generatePK(),
                chatId: BigInt("999999999999999999"), // Non-existent chat
                userId: BigInt(generatePK()),
                hasUnread: false,
            },
            nonExistentUser: {
                id: generatePK(),
                chatId: BigInt(generatePK()),
                userId: BigInt("999999999999999999"), // Non-existent user
                hasUnread: false,
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, ChatParticipantCreateInput> {
        const baseChatId = BigInt(generatePK());
        const baseUserId = BigInt(generatePK());
        
        return {
            participantWithUnread: {
                ...this.getMinimalData(),
                chatId: baseChatId,
                userId: baseUserId,
                hasUnread: true,
            },
            participantWithoutUnread: {
                ...this.getMinimalData(),
                chatId: baseChatId,
                userId: BigInt(generatePK()),
                hasUnread: false,
            },
            botParticipant: {
                ...this.getMinimalData(),
                chatId: baseChatId,
                userId: BigInt(generatePK()), // Would be a bot user ID
                hasUnread: false,
            },
            multiChatParticipant: {
                ...this.getMinimalData(),
                chatId: BigInt(generatePK()), // Different chat
                userId: baseUserId, // Same user
                hasUnread: true,
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            // ChatParticipant doesn't have child relationships to cascade
        };
    }

    protected async deleteRelatedRecords(
        record: any,
        remainingDepth: number,
        tx: any
    ): Promise<void> {
        // ChatParticipant doesn't have dependent records to delete
        // It's a junction table record
    }

    /**
     * Test scenario helpers
     */
    async createActiveParticipants(chatId: string, userIds: string[]): Promise<any[]> {
        return this.addParticipantsToChat(chatId, userIds, false);
    }

    async createMixedReadStates(
        chatId: string, 
        participants: Array<{ userId: string; hasUnread: boolean }>
    ): Promise<any[]> {
        const results: any[] = [];
        
        for (const participant of participants) {
            const result = await this.createParticipant(
                chatId, 
                participant.userId, 
                { hasUnread: participant.hasUnread }
            );
            results.push(result);
        }
        
        return results;
    }

    async createChatWithParticipants(
        chatId: string,
        config: {
            activeUsers: string[];
            unreadUsers: string[];
            bots: string[];
        }
    ): Promise<{
        active: any[];
        unread: any[];
        bots: any[];
    }> {
        const [active, unread, bots] = await Promise.all([
            this.addParticipantsToChat(chatId, config.activeUsers, false),
            this.addParticipantsToChat(chatId, config.unreadUsers, true),
            this.addParticipantsToChat(chatId, config.bots, false),
        ]);

        return { active, unread, bots };
    }
}

// Export factory creator function
export const createChatParticipantDbFactory = (prisma: PrismaClient) => new ChatParticipantDbFactory(prisma);