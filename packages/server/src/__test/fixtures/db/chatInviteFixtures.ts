/* eslint-disable no-magic-numbers */
import { type Prisma } from "@prisma/client";
import { generatePK } from "@vrooli/shared";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { BulkSeedResult, DbErrorScenarios, DbTestFixtures } from "./types.js";

/**
 * Database fixtures for ChatInvite model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing - using factory function to avoid module-level generatePK() calls
export function getChatInviteDbIds() {
    return {
        invite1: generatePK(),
        invite2: generatePK(),
        invite3: generatePK(),
        chat1: generatePK(),
        chat2: generatePK(),
        user1: generatePK(),
        user2: generatePK(),
        user3: generatePK(),
    };
}

/**
 * Enhanced test fixtures for ChatInvite model following standard structure
 */
export function getChatInviteDbFixtures(): DbTestFixtures<Prisma.chat_inviteCreateInput> {
    const ids = getChatInviteDbIds();
    return {
        minimal: {
            id: generatePK(),
            message: "You're invited to join this chat!",
            chat: { connect: { id: BigInt(ids.chat1) } },
            user: { connect: { id: BigInt(ids.user1) } },
            status: "Pending",
        },
        complete: {
            id: generatePK(),
            message: "Welcome to our discussion! We'd love to have you join us in this conversation.",
            chat: { connect: { id: BigInt(ids.chat1) } },
            user: { connect: { id: BigInt(ids.user1) } },
            status: "Accepted",
        },
        invalid: {
            missingRequired: {
                // Missing required chat and user
                id: generatePK(),
                status: "Pending",
            },
            invalidTypes: {
                id: BigInt("123456789"), // Invalid snowflake value but properly typed
                message: 123, // Should be string
                chatId: BigInt("987654321"), // Invalid chat reference but properly typed
                userId: BigInt("192837465"), // Invalid user reference but properly typed
                status: "InvalidStatus", // Not a valid status
            },
            emptyMessage: {
                id: generatePK(),
                message: "", // Empty message
                chatId: BigInt(ids.chat1),
                userId: BigInt(ids.user1),
                status: "Pending",
            },
        },
        edgeCases: {
            pendingInvite: {
                id: generatePK(),
                message: "Pending invitation",
                chat: { connect: { id: BigInt(ids.chat1) } },
                user: { connect: { id: BigInt(ids.user1) } },
                status: "Pending",
            },
            acceptedInvite: {
                id: generatePK(),
                message: "Accepted invitation",
                chat: { connect: { id: BigInt(ids.chat1) } },
                user: { connect: { id: BigInt(ids.user2) } },
                status: "Accepted",
            },
            declinedInvite: {
                id: generatePK(),
                message: "Declined invitation",
                chat: { connect: { id: BigInt(ids.chat1) } },
                user: { connect: { id: BigInt(ids.user3) } },
                status: "Declined",
            },
            longMessage: {
                id: generatePK(),
                message: "This is a very long invitation message that goes into great detail about the chat, its purpose, the participants, the expected discussion topics, and all the reasons why the invitee should consider joining this particular conversation. ".repeat(5),
                chat: { connect: { id: BigInt(ids.chat1) } },
                user: { connect: { id: BigInt(ids.user1) } },
                status: "Pending",
            },
            shortMessage: {
                id: generatePK(),
                message: "Hi!",
                chat: { connect: { id: BigInt(ids.chat1) } },
                user: { connect: { id: BigInt(ids.user1) } },
                status: "Pending",
            },
            specialCharactersMessage: {
                id: generatePK(),
                message: "Welcome! 🚀 Join us for a great discussion 💬 with special chars: @#$%^&*()_+{}|:<>?[]\\/.,;'\"",
                chat: { connect: { id: BigInt(ids.chat1) } },
                user: { connect: { id: BigInt(ids.user1) } },
                status: "Pending",
            },
            multipleChatsInvite: {
                id: generatePK(),
                message: "Invitation to second chat",
                chat: { connect: { id: BigInt(ids.chat2) } },
                user: { connect: { id: BigInt(ids.user1) } },
                status: "Pending",
            },
        },
    };
}

/**
 * Enhanced factory for creating chat invite database fixtures
 */
export class ChatInviteDbFactory extends EnhancedDbFactory<Prisma.chat_inviteCreateInput> {

    /**
     * Get the test fixtures for ChatInvite model
     */
    protected getFixtures(): DbTestFixtures<Prisma.chat_inviteCreateInput> {
        return getChatInviteDbFixtures();
    }

    /**
     * Get ChatInvite-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        const ids = getChatInviteDbIds();
        const fixtures = getChatInviteDbFixtures();
        return {
            constraints: {
                uniqueViolation: {
                    id: ids.invite1, // Duplicate ID
                    message: "Duplicate invite",
                    chatId: BigInt(ids.chat1),
                    userId: BigInt(ids.user1),
                    status: "Pending",
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    message: "Foreign key violation",
                    chat: { connect: { id: BigInt("999999999") } }, // Non-existent chat ID but properly typed
                    user: { connect: { id: BigInt(ids.user1) } },
                    status: "Pending",
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    message: "Check constraint violation",
                    chat: { connect: { id: BigInt(ids.chat1) } },
                    user: { connect: { id: BigInt(ids.user1) } },
                    status: "Pending",
                },
            },
            validation: {
                requiredFieldMissing: fixtures.invalid.missingRequired,
                invalidDataType: fixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    message: "a".repeat(10000), // Message too long (max 4096)
                    chat: { connect: { id: BigInt(ids.chat1) } },
                    user: { connect: { id: BigInt(ids.user1) } },
                    status: "Pending",
                },
            },
            businessLogic: {
                duplicateInvite: {
                    id: generatePK(),
                    message: "Duplicate chat invite",
                    chat: { connect: { id: BigInt(ids.chat1) } },
                    user: { connect: { id: BigInt(ids.user1) } }, // Same user/chat combo
                    status: "Pending",
                },
                selfInvite: {
                    id: generatePK(),
                    message: "Self invitation",
                    chat: { connect: { id: BigInt(ids.chat1) } },
                    user: { connect: { id: BigInt(ids.user1) } }, // User inviting themselves
                    status: "Pending",
                },
            },
        };
    }

    /**
     * ChatInvite-specific validation
     */
    protected validateSpecific(data: Prisma.chat_inviteCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to ChatInvite
        if (!data.chat) errors.push("ChatInvite chat relationship is required");
        if (!data.user) errors.push("ChatInvite user relationship is required");

        // Check message content
        if (data.message && typeof data.message === "string") {
            if (data.message.length > 4096) {
                errors.push("Message exceeds maximum length of 4096 characters");
            }
            if (data.message.length > 1000) {
                warnings.push("Message is very long (>1000 chars)");
            }
        }

        // Check status
        if (data.status && !["Pending", "Accepted", "Declined"].includes(data.status)) {
            errors.push("Invalid status value");
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(
        chatId: string,
        userId: string,
        overrides?: Partial<Prisma.chat_inviteCreateInput>,
    ): Prisma.chat_inviteCreateInput {
        const factory = new ChatInviteDbFactory();
        return factory.createMinimal({
            chat: { connect: { id: BigInt(chatId) } },
            user: { connect: { id: BigInt(userId) } },
            message: "You're invited to join this chat!",
            status: "Pending",
            ...overrides,
        });
    }

    static createWithCustomMessage(
        chatId: string,
        userId: string,
        message: string,
        overrides?: Partial<Prisma.chat_inviteCreateInput>,
    ): Prisma.chat_inviteCreateInput {
        return this.createMinimal(chatId, userId, {
            message,
            ...overrides,
        });
    }

    static createAccepted(
        chatId: string,
        userId: string,
        overrides?: Partial<Prisma.chat_inviteCreateInput>,
    ): Prisma.chat_inviteCreateInput {
        return this.createMinimal(chatId, userId, {
            status: "Accepted",
            ...overrides,
        });
    }

    static createDeclined(
        chatId: string,
        userId: string,
        overrides?: Partial<Prisma.chat_inviteCreateInput>,
    ): Prisma.chat_inviteCreateInput {
        return this.createMinimal(chatId, userId, {
            status: "Declined",
            ...overrides,
        });
    }

    /**
     * Create invite with custom message
     */
    static createWithMessage(
        chatId: string,
        userId: string,
        message: string,
        overrides?: Partial<Prisma.chat_inviteCreateInput>,
    ): Prisma.chat_inviteCreateInput {
        return this.createMinimal(chatId, userId, {
            message,
            ...overrides,
        });
    }
}

/**
 * Enhanced helper to seed multiple test chat invites with comprehensive options
 */
export async function seedChatInvites(
    prisma: any,
    options: {
        chatId: string;
        userIds: string[];
        status?: "Pending" | "Accepted" | "Declined";
        withCustomMessages?: boolean;
    },
): Promise<BulkSeedResult<any>> {
    const invites = [];
    let pendingCount = 0;
    let acceptedCount = 0;
    let declinedCount = 0;

    for (let i = 0; i < options.userIds.length; i++) {
        const userId = options.userIds[i];
        let inviteData: Prisma.chat_inviteCreateInput;

        if (options.status === "Accepted") {
            inviteData = ChatInviteDbFactory.createAccepted(options.chatId, userId);
            acceptedCount++;
        } else if (options.status === "Declined") {
            inviteData = ChatInviteDbFactory.createDeclined(options.chatId, userId);
            declinedCount++;
        } else if (options.withCustomMessages) {
            inviteData = ChatInviteDbFactory.createWithCustomMessage(
                options.chatId,
                userId,
                `Custom invite message for user ${i + 1}`,
            );
            pendingCount++;
        } else {
            inviteData = ChatInviteDbFactory.createMinimal(options.chatId, userId);
            pendingCount++;
        }

        const invite = await prisma.chat_invite.create({ data: inviteData });
        invites.push(invite);
    }

    return {
        records: invites,
        summary: {
            total: invites.length,
            withAuth: 0, // ChatInvites don't have auth
            bots: 0, // ChatInvites don't have bots
            teams: 0, // ChatInvites don't have teams
            pending: pendingCount,
            accepted: acceptedCount,
            declined: declinedCount,
        },
    };
}

/**
 * Helper to create bulk invites with different configurations
 */
export async function seedBulkInvites(
    prisma: any,
    options: {
        chatId: string;
        creatorId: string;
        messages: string[];
    },
): Promise<BulkSeedResult<any>> {
    const invites = [];

    for (const message of options.messages) {
        const inviteData = ChatInviteDbFactory.createWithMessage(
            options.chatId,
            options.creatorId,
            message,
        );

        const invite = await prisma.chat_invite.create({ data: inviteData });
        invites.push(invite);
    }

    return {
        records: invites,
        summary: {
            total: invites.length,
            withAuth: 0,
            bots: 0,
            teams: 0,
        },
    };
}

/**
 * Helper to simulate invite usage
 */
export async function useInvite(
    prisma: any,
    inviteId: string,
    userId: string,
): Promise<{ success: boolean; message?: string; chatId?: string }> {
    const invite = await prisma.chatInvite.findUnique({
        where: { id: BigInt(inviteId) },
        include: { chat: true },
    });

    if (!invite) {
        return { success: false, message: "Invite not found" };
    }

    if (invite.status !== "Pending") {
        return { success: false, message: "Invite already used or declined" };
    }

    // Simple invite logic without expiry/usage limits

    // Update invite status
    await prisma.chat_invite.update({
        where: { id: BigInt(inviteId) },
        data: { status: "Accepted" },
    });

    // Add user to chat participants
    await prisma.chat_participants.create({
        data: {
            id: generatePK(),
            chatId: invite.chatId,
            userId: invite.userId,
        },
    });

    return {
        success: true,
        chatId: invite.chatId.toString(),
        message: "Successfully joined chat",
    };
}

/**
 * Helper to decline an invite
 */
export async function declineInvite(
    prisma: any,
    inviteId: string,
): Promise<{ success: boolean; message?: string }> {
    const invite = await prisma.chat_invite.findUnique({
        where: { id: BigInt(inviteId) },
    });

    if (!invite) {
        return { success: false, message: "Invite not found" };
    }

    if (invite.status !== "Pending") {
        return { success: false, message: "Invite already responded to" };
    }

    // Update invite status
    await prisma.chat_invite.update({
        where: { id: BigInt(inviteId) },
        data: { status: "Declined" },
    });

    return {
        success: true,
        message: "Invite declined",
    };
}
