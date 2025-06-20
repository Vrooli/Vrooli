import { generatePK, generatePublicId } from "@vrooli/shared";
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
export const chatParticipantDbFixtures: DbTestFixtures<Prisma.Chat_participantsCreateInput> = {
    minimal: {
        id: generatePK(),
        publicId: generatePublicId(),
        chat: { connect: { id: chatParticipantDbIds.chat1 } },
        user: { connect: { id: chatParticipantDbIds.user1 } },
    },
    complete: {
        id: generatePK(),
        publicId: generatePublicId(),
        chat: { connect: { id: chatParticipantDbIds.chat1 } },
        user: { connect: { id: chatParticipantDbIds.admin1 } },
        permissions: JSON.stringify({
            canDelete: true,
            canInvite: true,
            canKick: true,
            canUpdate: true,
            canViewHistory: true,
            canMention: true,
        }),
    },
    invalid: {
        missingRequired: {
            // Missing required chat and user
            publicId: generatePublicId(),
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            publicId: 123, // Should be string
            chat: "invalid-chat-reference", // Should be connect object
            user: "invalid-user-reference", // Should be connect object
            permissions: { invalid: "object" }, // Should be JSON string
        },
        invalidPermissions: {
            id: generatePK(),
            publicId: generatePublicId(),
            chat: { connect: { id: chatParticipantDbIds.chat1 } },
            user: { connect: { id: chatParticipantDbIds.user1 } },
            permissions: "invalid-json", // Invalid JSON
        },
    },
    edgeCases: {
        adminParticipant: {
            id: generatePK(),
            publicId: generatePublicId(),
            chat: { connect: { id: chatParticipantDbIds.chat1 } },
            user: { connect: { id: chatParticipantDbIds.admin1 } },
            permissions: JSON.stringify({
                canDelete: true,
                canInvite: true,
                canKick: true,
                canUpdate: true,
            }),
        },
        memberParticipant: {
            id: generatePK(),
            publicId: generatePublicId(),
            chat: { connect: { id: chatParticipantDbIds.chat1 } },
            user: { connect: { id: chatParticipantDbIds.user1 } },
            permissions: JSON.stringify({
                canDelete: false,
                canInvite: false,
                canKick: false,
                canUpdate: false,
            }),
        },
        customPermissions: {
            id: generatePK(),
            publicId: generatePublicId(),
            chat: { connect: { id: chatParticipantDbIds.chat1 } },
            user: { connect: { id: chatParticipantDbIds.user2 } },
            permissions: JSON.stringify({
                canDelete: false,
                canInvite: true,
                canKick: false,
                canUpdate: true,
                canMention: true,
                canViewHistory: true,
            }),
        },
        noPermissions: {
            id: generatePK(),
            publicId: generatePublicId(),
            chat: { connect: { id: chatParticipantDbIds.chat1 } },
            user: { connect: { id: chatParticipantDbIds.user1 } },
            permissions: null,
        },
        emptyPermissions: {
            id: generatePK(),
            publicId: generatePublicId(),
            chat: { connect: { id: chatParticipantDbIds.chat1 } },
            user: { connect: { id: chatParticipantDbIds.user1 } },
            permissions: JSON.stringify({}),
        },
        multipleChats: {
            id: generatePK(),
            publicId: generatePublicId(),
            chat: { connect: { id: chatParticipantDbIds.chat2 } },
            user: { connect: { id: chatParticipantDbIds.user1 } },
            permissions: JSON.stringify({
                canDelete: false,
                canInvite: false,
                canKick: false,
                canUpdate: false,
            }),
        },
    },
};

/**
 * Enhanced factory for creating chat participant database fixtures
 */
export class ChatParticipantDbFactory extends EnhancedDbFactory<Prisma.Chat_participantsCreateInput> {
    
    /**
     * Get the test fixtures for ChatParticipant model
     */
    protected getFixtures(): DbTestFixtures<Prisma.Chat_participantsCreateInput> {
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
                    publicId: generatePublicId(),
                    chat: { connect: { id: chatParticipantDbIds.chat1 } },
                    user: { connect: { id: chatParticipantDbIds.user1 } },
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    chat: { connect: { id: "non-existent-chat-id" } },
                    user: { connect: { id: chatParticipantDbIds.user1 } },
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    publicId: "", // Empty publicId
                    chat: { connect: { id: chatParticipantDbIds.chat1 } },
                    user: { connect: { id: chatParticipantDbIds.user1 } },
                },
            },
            validation: {
                requiredFieldMissing: chatParticipantDbFixtures.invalid.missingRequired,
                invalidDataType: chatParticipantDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    publicId: "a".repeat(500), // PublicId too long
                    chat: { connect: { id: chatParticipantDbIds.chat1 } },
                    user: { connect: { id: chatParticipantDbIds.user1 } },
                    permissions: "a".repeat(10000), // Very long permissions string
                },
            },
            businessLogic: {
                duplicateParticipant: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    chat: { connect: { id: chatParticipantDbIds.chat1 } },
                    user: { connect: { id: chatParticipantDbIds.user1 } }, // Same user in same chat
                },
                invalidPermissions: chatParticipantDbFixtures.invalid.invalidPermissions,
            },
        };
    }

    /**
     * ChatParticipant-specific validation
     */
    protected validateSpecific(data: Prisma.Chat_participantsCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to ChatParticipant
        if (!data.chat) errors.push("ChatParticipant chat is required");
        if (!data.user) errors.push("ChatParticipant user is required");
        if (!data.publicId) errors.push("ChatParticipant publicId is required");

        // Check permissions
        if (data.permissions && typeof data.permissions === 'string') {
            try {
                const perms = JSON.parse(data.permissions);
                if (typeof perms !== 'object') {
                    errors.push("Permissions must be a JSON object");
                }
            } catch (e) {
                errors.push("Permissions must be valid JSON");
            }
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(
        chatId: string,
        userId: string,
        overrides?: Partial<Prisma.Chat_participantsCreateInput>
    ): Prisma.Chat_participantsCreateInput {
        const factory = new ChatParticipantDbFactory();
        return factory.createMinimal({
            chat: { connect: { id: chatId } },
            user: { connect: { id: userId } },
            ...overrides,
        });
    }

    static createAdmin(
        chatId: string,
        userId: string,
        overrides?: Partial<Prisma.Chat_participantsCreateInput>
    ): Prisma.Chat_participantsCreateInput {
        return this.createMinimal(chatId, userId, {
            permissions: JSON.stringify({
                canDelete: true,
                canInvite: true,
                canKick: true,
                canUpdate: true,
            }),
            ...overrides,
        });
    }

    static createMember(
        chatId: string,
        userId: string,
        overrides?: Partial<Prisma.Chat_participantsCreateInput>
    ): Prisma.Chat_participantsCreateInput {
        return this.createMinimal(chatId, userId, {
            permissions: JSON.stringify({
                canDelete: false,
                canInvite: false,
                canKick: false,
                canUpdate: false,
            }),
            ...overrides,
        });
    }

    static createWithCustomPermissions(
        chatId: string,
        userId: string,
        permissions: Record<string, boolean>,
        overrides?: Partial<Prisma.Chat_participantsCreateInput>
    ): Prisma.Chat_participantsCreateInput {
        return this.createMinimal(chatId, userId, {
            permissions: JSON.stringify(permissions),
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
            role?: "admin" | "member";
            permissions?: Record<string, boolean>;
        }>;
    }
): Promise<BulkSeedResult<any>> {
    const createdParticipants = [];
    let adminCount = 0;
    let memberCount = 0;
    let customCount = 0;

    for (const participant of options.participants) {
        let participantData: Prisma.Chat_participantsCreateInput;

        if (participant.role === "admin") {
            participantData = ChatParticipantDbFactory.createAdmin(options.chatId, participant.userId);
            adminCount++;
        } else if (participant.permissions) {
            participantData = ChatParticipantDbFactory.createWithCustomPermissions(
                options.chatId,
                participant.userId,
                participant.permissions
            );
            customCount++;
        } else {
            participantData = ChatParticipantDbFactory.createMember(options.chatId, participant.userId);
            memberCount++;
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
            admins: adminCount,
            members: memberCount,
            customPermissions: customCount,
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
    role: "admin" | "member" = "member"
): Promise<BulkSeedResult<any>> {
    const participants = userIds.map(userId => ({
        userId,
        role,
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
    chatId: string
) {
    return prisma.chat_participants.findMany({
        where: { chatId },
        include: {
            user: true,
            chat: true,
        },
    });
}