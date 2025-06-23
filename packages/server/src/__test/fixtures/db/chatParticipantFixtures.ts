import { generatePK } from "../../../../../shared/src/id/index.js";
import { type Prisma } from "@prisma/client";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { DbTestFixtures, BulkSeedOptions, BulkSeedResult, DbErrorScenarios } from "./types.js";

/**
 * Database fixtures for ChatParticipant model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const chatParticipantDbIds = {
    participant1: generatePK(),
    participant2: generatePK(),
    participant3: generatePK(),
    chat1: generatePK(),
    chat2: generatePK(),
    user1: generatePK(),
    user2: generatePK(),
    admin1: generatePK(),
};

/**
 * Enhanced test fixtures for ChatParticipant model following standard structure
 */
export const chatParticipantDbFixtures: DbTestFixtures<Prisma.chat_participantsCreateInput> = {
    minimal: {
        id: generatePK(),
        chatId: BigInt(chatParticipantDbIds.chat1),
        userId: BigInt(chatParticipantDbIds.user1),
    },
    complete: {
        id: generatePK(),
        chatId: BigInt(chatParticipantDbIds.chat1),
        userId: BigInt(chatParticipantDbIds.admin1),
        hasUnread: false,
    },
    invalid: {
        missingRequired: {
            // Missing required chat and user
            id: generatePK(),
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            hasUnread: "yes", // Should be boolean
            chatId: "invalid-chat-reference", // Should be BigInt
            userId: "invalid-user-reference", // Should be BigInt
        },
        duplicateParticipant: {
            id: generatePK(),
            chatId: BigInt(chatParticipantDbIds.chat1),
            userId: BigInt(chatParticipantDbIds.user1), // Same user in same chat
        },
    },
    edgeCases: {
        participantWithNoUnread: {
            id: generatePK(),
            chatId: BigInt(chatParticipantDbIds.chat1),
            userId: BigInt(chatParticipantDbIds.admin1),
            hasUnread: false,
        },
        participantWithUnread: {
            id: generatePK(),
            chat: { connect: { id: chatParticipantDbIds.chat1 } },
            user: { connect: { id: chatParticipantDbIds.user1 } },
            hasUnread: true,
        },
        multipleChats: {
            id: generatePK(),
            chatId: BigInt(chatParticipantDbIds.chat2),
            userId: BigInt(chatParticipantDbIds.user1),
            hasUnread: true,
        },
    },
};

/**
 * Enhanced factory for creating chat participant database fixtures
 */
export class ChatParticipantDbFactory extends EnhancedDbFactory<Prisma.chat_participantsCreateInput> {
    
    /**
     * Get the test fixtures for ChatParticipant model
     */
    protected getFixtures(): DbTestFixtures<Prisma.chat_participantsCreateInput> {
        return chatParticipantDbFixtures;
    }

    /**
     * Get ChatParticipant-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: chatParticipantDbIds.participant1, // Duplicate ID
                    chatId: BigInt(chatParticipantDbIds.chat1),
                    userId: BigInt(chatParticipantDbIds.user1),
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    chatId: "non-existent-chat-id",
                    userId: BigInt(chatParticipantDbIds.user1),
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    chatId: BigInt(chatParticipantDbIds.chat1),
                    userId: BigInt(chatParticipantDbIds.user1),
                    // We might create a constraint violation in the future
                },
            },
            validation: {
                requiredFieldMissing: chatParticipantDbFixtures.invalid.missingRequired,
                invalidDataType: chatParticipantDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    chatId: BigInt(chatParticipantDbIds.chat1),
                    userId: BigInt(chatParticipantDbIds.user1),
                    // No out of range scenarios for current schema
                },
            },
            businessLogic: {
                duplicateParticipant: chatParticipantDbFixtures.invalid.duplicateParticipant,
            },
        };
    }

    /**
     * ChatParticipant-specific validation
     */
    protected validateSpecific(data: Prisma.chat_participantsCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to ChatParticipant
        if (!data.chatId) errors.push("ChatParticipant chatId is required");
        if (!data.userId) errors.push("ChatParticipant userId is required");

        // Check hasUnread field
        if (data.hasUnread !== undefined && typeof data.hasUnread !== "boolean") {
            errors.push("hasUnread must be a boolean");
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(
        chatId: string,
        userId: string,
        overrides?: Partial<Prisma.chat_participantsCreateInput>,
    ): Prisma.chat_participantsCreateInput {
        const factory = new ChatParticipantDbFactory();
        return factory.createMinimal({
            chatId: BigInt(chatId),
            userId: BigInt(userId),
            ...overrides,
        });
    }

    static createWithUnread(
        chatId: string,
        userId: string,
        hasUnread = true,
        overrides?: Partial<Prisma.chat_participantsCreateInput>,
    ): Prisma.chat_participantsCreateInput {
        return this.createMinimal(chatId, userId, {
            hasUnread,
            ...overrides,
        });
    }
}

/**
 * Enhanced helper to seed multiple test chat participants with comprehensive options
 */
export async function seedChatParticipants(
    prisma: any,
    options: {
        chatId: string;
        participants: Array<{
            userId: string;
            hasUnread?: boolean;
        }>;
    },
): Promise<BulkSeedResult<any>> {
    const createdParticipants = [];
    let withUnreadCount = 0;
    let withoutUnreadCount = 0;

    for (const participant of options.participants) {
        const participantData = ChatParticipantDbFactory.createWithUnread(
            options.chatId,
            participant.userId,
            participant.hasUnread ?? true,
        );

        if (participant.hasUnread !== false) {
            withUnreadCount++;
        } else {
            withoutUnreadCount++;
        }

        const chatParticipant = await prisma.chat_participants.create({
            data: participantData,
            include: {
                chat: true,
                user: true,
            },
        });
        createdParticipants.push(chatParticipant);
    }

    return {
        records: createdParticipants,
        summary: {
            total: createdParticipants.length,
            withAuth: 0, // ChatParticipants don't have auth
            bots: 0, // Not tracked here
            teams: 0, // ChatParticipants don't have teams
            withUnread: withUnreadCount,
            withoutUnread: withoutUnreadCount,
        },
    };
}

/**
 * Helper to add participants to an existing chat
 */
export async function addParticipantsToChat(
    prisma: any,
    chatId: string,
    userIds: string[],
    hasUnread = true,
): Promise<BulkSeedResult<any>> {
    const participants = userIds.map(userId => ({
        userId,
        hasUnread,
    }));

    return seedChatParticipants(prisma, {
        chatId,
        participants,
    });
}

/**
 * Helper to get all participants for a chat
 */
export async function getChatParticipants(
    prisma: any,
    chatId: string,
) {
    return prisma.chat_participants.findMany({
        where: { chatId },
        include: {
            user: true,
            chat: true,
        },
    });
}

/**
 * Helper to mark messages as read
 */
export async function markAsRead(
    prisma: any,
    chatId: string,
    userId: string,
): Promise<any> {
    return prisma.chat_participants.update({
        where: {
            chatId_userId: {
                chatId: BigInt(chatId),
                userId: BigInt(userId),
            },
        },
        data: {
            hasUnread: false,
        },
    });
}

/**
 * Helper to simulate real-time participant states
 */
export async function seedParticipantStates(
    prisma: any,
    chatId: string,
    states: Array<{
        userId: string;
        hasUnread?: boolean;
    }>,
): Promise<any[]> {
    const updates = [];

    for (const state of states) {
        const updateData: any = {};

        if (state.hasUnread !== undefined) {
            updateData.hasUnread = state.hasUnread;
        }

        const updated = await prisma.chat_participants.update({
            where: {
                chatId_userId: {
                    chatId: BigInt(chatId),
                    userId: BigInt(state.userId),
                },
            },
            data: updateData,
        });

        updates.push(updated);
    }

    return updates;
}
