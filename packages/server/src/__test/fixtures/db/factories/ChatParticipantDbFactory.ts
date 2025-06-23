import { generatePK, generatePublicId, nanoid } from "../idHelpers.js";
import { type chat_participants, type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface ChatParticipantRelationConfig extends RelationConfig {
    chatId?: string;
    userId?: string;
    withUnread?: boolean;
    participantRole?: "member" | "admin" | "moderator";
}

/**
 * Database fixture factory for ChatParticipant model
 * Handles user/team/bot participants with roles and unread status
 */
export class ChatParticipantDbFactory extends DatabaseFixtureFactory<
    chat_participants,
    Prisma.chat_participantsCreateInput,
    Prisma.chat_participantsInclude,
    Prisma.chat_participantsUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("chat_participants", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.chat_participants;
    }

    protected getMinimalData(overrides?: Partial<Prisma.chat_participantsCreateInput>): Prisma.chat_participantsCreateInput {
        return {
            id: generatePK(),
            chat: { connect: { id: generatePK() } },
            user: { connect: { id: generatePK() } },
            hasUnread: false,
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.chat_participantsCreateInput>): Prisma.chat_participantsCreateInput {
        return {
            id: generatePK(),
            chat: { connect: { id: generatePK() } },
            user: { connect: { id: generatePK() } },
            hasUnread: false,
            ...overrides,
        };
    }

    /**
     * Create participant with specific chat and user
     */
    async createParticipant(
        chatId: string | bigint, 
        userId: string | bigint, 
        overrides?: Partial<Prisma.chat_participantsCreateInput>,
    ): Promise<chat_participants> {
        const data: Prisma.chat_participantsCreateInput = {
            ...this.getMinimalData(),
            chat: { connect: { id: BigInt(chatId) } },
            user: { connect: { id: BigInt(userId) } },
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
        chatId: string | bigint, 
        userId: string | bigint, 
        hasUnread = true,
    ): Promise<chat_participants> {
        return this.createParticipant(chatId, userId, { hasUnread });
    }

    /**
     * Create admin participant
     */
    async createAdmin(
        chatId: string | bigint, 
        userId: string | bigint,
    ): Promise<chat_participants> {
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
        chatId: string | bigint, 
        botId: string | bigint,
    ): Promise<chat_participants> {
        return this.createParticipant(chatId, botId, {
            hasUnread: false,
        });
    }

    protected getDefaultInclude(): Prisma.chat_participantsInclude {
        return {
            chat: {
                select: {
                    id: true,
                    publicId: true,
                    isPrivate: true,
                    translations: {
                        where: { language: "en" },
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
        baseData: Prisma.chat_participantsCreateInput,
        config: ChatParticipantRelationConfig,
        tx: any,
    ): Promise<Prisma.chat_participantsCreateInput> {
        const data = { ...baseData };

        // Apply chat and user connections
        if (config.chatId) {
            data.chat = { connect: { id: BigInt(config.chatId) } };
        }
        
        if (config.userId) {
            data.user = { connect: { id: BigInt(config.userId) } };
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
        chatId: string | bigint, 
        userIds: (string | bigint)[], 
        hasUnread = true,
    ): Promise<chat_participants[]> {
        const participants: chat_participants[] = [];
        
        for (const userId of userIds) {
            const participant = await this.createParticipant(chatId, userId, { hasUnread });
            participants.push(participant);
        }
        
        return participants;
    }

    /**
     * Mark participant as having read messages
     */
    async markAsRead(chatId: string | bigint, userId: string | bigint): Promise<Prisma.BatchPayload> {
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
    async markAsUnread(chatId: string | bigint, userId: string | bigint): Promise<Prisma.BatchPayload> {
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

    protected async checkModelConstraints(record: chat_participants): Promise<string[]> {
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
            violations.push("User can only participate once per chat");
        }

        // Check chat exists
        const chat = await this.prisma.chat.findUnique({
            where: { id: record.chatId },
        });
        if (!chat) {
            violations.push("Referenced chat does not exist");
        }

        // Check user exists
        const user = await this.prisma.user.findUnique({
            where: { id: record.userId },
        });
        if (!user) {
            violations.push("Referenced user does not exist");
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
    getEdgeCaseScenarios(): Record<string, Prisma.chat_participantsCreateInput> {
        const baseChatId = BigInt(generatePK());
        const baseUserId = BigInt(generatePK());
        
        return {
            participantWithUnread: {
                ...this.getMinimalData(),
                chat: { connect: { id: baseChatId } },
                user: { connect: { id: baseUserId } },
                hasUnread: true,
            },
            participantWithoutUnread: {
                ...this.getMinimalData(),
                chat: { connect: { id: baseChatId } },
                user: { connect: { id: BigInt(generatePK()) } },
                hasUnread: false,
            },
            botParticipant: {
                ...this.getMinimalData(),
                chat: { connect: { id: baseChatId } },
                user: { connect: { id: BigInt(generatePK()) } }, // Would be a bot user ID
                hasUnread: false,
            },
            multiChatParticipant: {
                ...this.getMinimalData(),
                chat: { connect: { id: BigInt(generatePK()) } }, // Different chat
                user: { connect: { id: baseUserId } }, // Same user
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
        record: chat_participants,
        remainingDepth: number,
        tx: any,
    ): Promise<void> {
        // ChatParticipant doesn't have dependent records to delete
        // It's a junction table record
    }

    /**
     * Test scenario helpers
     */
    async createActiveParticipants(chatId: string | bigint, userIds: (string | bigint)[]): Promise<chat_participants[]> {
        return this.addParticipantsToChat(chatId, userIds, false);
    }

    async createMixedReadStates(
        chatId: string | bigint, 
        participants: Array<{ userId: string | bigint; hasUnread: boolean }>,
    ): Promise<chat_participants[]> {
        const results: chat_participants[] = [];
        
        for (const participant of participants) {
            const result = await this.createParticipant(
                chatId, 
                participant.userId, 
                { hasUnread: participant.hasUnread },
            );
            results.push(result);
        }
        
        return results;
    }

    async createChatWithParticipants(
        chatId: string | bigint,
        config: {
            activeUsers: (string | bigint)[];
            unreadUsers: (string | bigint)[];
            bots: (string | bigint)[];
        },
    ): Promise<{
        active: chat_participants[];
        unread: chat_participants[];
        bots: chat_participants[];
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
