// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: replaced any with PrismaClient type
import { type Prisma, type PrismaClient, type chat_invite } from "@prisma/client";
import { ChatInviteStatus } from "@vrooli/shared";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type {
    DbTestFixtures,
    RelationConfig,
    TestScenario,
} from "./types.js";

interface ChatInviteRelationConfig extends RelationConfig {
    chat: { chatId: bigint };
    user: { userId: bigint };
}

/**
 * Enhanced database fixture factory for ChatInvite model
 * Provides comprehensive testing capabilities for chat invitation systems
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for different invite statuses
 * - Message customization
 * - Bulk invite operations
 * - Predefined test scenarios
 * - Constraint validation
 */
export class ChatInviteDbFactory extends EnhancedDatabaseFactory<
    chat_invite,
    Prisma.chat_inviteCreateInput,
    Prisma.chat_inviteInclude,
    Prisma.chat_inviteUpdateInput
> {
    protected scenarios: Record<string, TestScenario> = {};

    constructor(modelName: string, prisma: PrismaClient) {
        super(modelName, prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.chat_invite;
    }

    /**
     * Get complete test fixtures for ChatInvite model
     */
    protected getFixtures(): DbTestFixtures<Prisma.chat_inviteCreateInput, Prisma.chat_inviteUpdateInput> {
        return {
            minimal: {
                id: this.generateId(),
                status: ChatInviteStatus.Pending,
                chat: { connect: { id: this.generateId() } },
                user: { connect: { id: this.generateId() } },
            },
            complete: {
                id: this.generateId(),
                status: ChatInviteStatus.Pending,
                message: "You're invited to join our team discussion about the new project!",
                chat: { connect: { id: this.generateId() } },
                user: { connect: { id: this.generateId() } },
            },
            invalid: {
                missingRequired: {
                    // Missing id, chat, user
                    status: ChatInviteStatus.Pending,
                    message: "Invalid invite",
                },
                invalidTypes: {
                    id: this.generateId(),
                    status: "pending", // Should be ChatInviteStatus enum
                    message: 123, // Should be string
                    chat: "not-an-object", // Should be object
                    user: null, // Should be object
                },
                duplicateInvite: {
                    id: this.generateId(),
                    status: ChatInviteStatus.Pending,
                    chat: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } }, // Assumes this combo exists
                },
                invalidStatus: {
                    id: this.generateId(),
                    status: "InvalidStatus" as any,
                    chat: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
            },
            edgeCases: {
                maxLengthMessage: {
                    id: this.generateId(),
                    status: ChatInviteStatus.Pending,
                    message: "a".repeat(4096), // Max length message
                    chat: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
                unicodeMessage: {
                    id: this.generateId(),
                    status: ChatInviteStatus.Pending,
                    message: "You're invited! 🎉 Join us for a great discussion 💬",
                    chat: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
                acceptedInvite: {
                    id: this.generateId(),
                    status: ChatInviteStatus.Accepted,
                    message: "Thanks for joining!",
                    chat: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
                declinedInvite: {
                    id: this.generateId(),
                    status: ChatInviteStatus.Declined,
                    message: "Sorry, I can't join right now",
                    chat: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
                noMessageInvite: {
                    id: this.generateId(),
                    status: ChatInviteStatus.Pending,
                    message: null,
                    chat: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
            },
            updates: {
                minimal: {
                    status: ChatInviteStatus.Accepted,
                },
                complete: {
                    status: ChatInviteStatus.Declined,
                    message: "Updated message - Unfortunately I cannot join at this time",
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.chat_inviteCreateInput>): Prisma.chat_inviteCreateInput {
        // Note: chat and user connections must be provided via overrides or config
        return {
            id: this.generateId(),
            status: ChatInviteStatus.Pending,
            chat: { connect: { id: this.generateId() } },
            user: { connect: { id: this.generateId() } },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.chat_inviteCreateInput>): Prisma.chat_inviteCreateInput {
        return {
            id: this.generateId(),
            status: ChatInviteStatus.Pending,
            message: "You're invited to join our team discussion!",
            chat: { connect: { id: this.generateId() } },
            user: { connect: { id: this.generateId() } },
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            pendingInvite: {
                name: "pendingInvite",
                description: "Standard pending invitation",
                config: {
                    overrides: {
                        status: ChatInviteStatus.Pending,
                        message: "You've been invited to join our chat",
                    },
                    chat: { chatId: this.generateId() },
                    user: { userId: this.generateId() },
                },
            },
            acceptedInvite: {
                name: "acceptedInvite",
                description: "Accepted invitation",
                config: {
                    overrides: {
                        status: ChatInviteStatus.Accepted,
                        message: "Welcome! Glad you could join us",
                    },
                    chat: { chatId: this.generateId() },
                    user: { userId: this.generateId() },
                },
            },
            declinedInvite: {
                name: "declinedInvite",
                description: "Declined invitation",
                config: {
                    overrides: {
                        status: ChatInviteStatus.Declined,
                        message: "Sorry, I'm not available",
                    },
                    chat: { chatId: this.generateId() },
                    user: { userId: this.generateId() },
                },
            },
            bulkTeamInvites: {
                name: "bulkTeamInvites",
                description: "Multiple invitations for team chat",
                config: {
                    overrides: {
                        status: ChatInviteStatus.Pending,
                        message: "Join our team chat for project updates",
                    },
                    chat: { chatId: this.generateId() },
                    user: { userId: this.generateId() },
                },
            },
            personalInvite: {
                name: "personalInvite",
                description: "Personal invitation with custom message",
                config: {
                    overrides: {
                        status: ChatInviteStatus.Pending,
                        message: "Hey! I'd love to chat with you about the new features we're building. Are you available?",
                    },
                    chat: { chatId: this.generateId() },
                    user: { userId: this.generateId() },
                },
            },
        };
    }

    protected getDefaultInclude(): Prisma.chat_inviteInclude {
        return {
            chat: {
                select: {
                    id: true,
                    publicId: true,
                    isPrivate: true,
                    openToAnyoneWithInvite: true,
                    translations: {
                        select: {
                            language: true,
                            name: true,
                            description: true,
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
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.chat_inviteCreateInput,
        config: ChatInviteRelationConfig,
        _tx: PrismaClient,
    ): Promise<Prisma.chat_inviteCreateInput> {
        const data = { ...baseData };

        // Handle chat connection (required)
        if (config.chat) {
            data.chat = {
                connect: { id: config.chat.chatId },
            };
        } else {
            throw new Error("ChatInvite requires a chat connection");
        }

        // Handle user connection (required)
        if (config.user) {
            data.user = {
                connect: { id: config.user.userId },
            };
        } else {
            throw new Error("ChatInvite requires a user connection");
        }

        return data;
    }

    /**
     * Create a pending invite
     */
    async createPendingInvite(chatId: bigint, userId: bigint, message?: string): Promise<chat_invite> {
        return await this.createWithRelations({
            overrides: {
                status: ChatInviteStatus.Pending,
                message: message || "You've been invited to join this chat",
            },
            chat: { chatId },
            user: { userId },
        });
    }

    /**
     * Create bulk invites for multiple users
     */
    async createBulkInvites(
        chatId: bigint,
        userIds: bigint[],
        message?: string,
    ): Promise<chat_invite[]> {
        const invites = await Promise.all(
            userIds.map(userId =>
                this.createPendingInvite(chatId, userId, message),
            ),
        );
        return invites;
    }

    /**
     * Accept an invite
     */
    async acceptInvite(inviteId: bigint): Promise<chat_invite> {
        const updated = await this.prisma.chat_invite.update({
            where: { id: inviteId },
            data: { status: ChatInviteStatus.Accepted },
            include: this.getDefaultInclude(),
        });
        return updated;
    }

    /**
     * Decline an invite
     */
    async declineInvite(inviteId: bigint): Promise<chat_invite> {
        const updated = await this.prisma.chat_invite.update({
            where: { id: inviteId },
            data: { status: ChatInviteStatus.Declined },
            include: this.getDefaultInclude(),
        });
        return updated;
    }

    protected async checkModelConstraints(record: chat_invite): Promise<string[]> {
        const violations: string[] = [];

        // Check for duplicate invite
        const duplicate = await this.prisma.chat_invite.findFirst({
            where: {
                chatId: record.chatId,
                userId: record.userId,
                id: { not: record.id },
            },
        });

        if (duplicate) {
            violations.push("User already has an invite to this chat");
        }

        // Check if user is already a participant
        const existingParticipant = await this.prisma.chat_participants.findFirst({
            where: {
                chatId: record.chatId,
                userId: record.userId,
            },
        });

        if (existingParticipant) {
            violations.push("User is already a participant in this chat");
        }

        // Check message length
        if (record.message && record.message.length > 4096) {
            violations.push("Message exceeds maximum length of 4096 characters");
        }

        // Check valid status
        if (!Object.values(ChatInviteStatus).includes(record.status as ChatInviteStatus)) {
            violations.push("Invalid invite status");
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            // ChatInvite has no dependent records
        };
    }

    protected async deleteRelatedRecords(
        record: chat_invite,
        remainingDepth: number,
        tx: PrismaClient,
        includeOnly?: string[],
    ): Promise<void> {
        // ChatInvite has no dependent records to delete
    }

    /**
     * Get invites by status
     */
    async getInvitesByStatus(status: ChatInviteStatus, limit?: number): Promise<chat_invite[]> {
        return await this.prisma.chat_invite.findMany({
            where: { status },
            include: this.getDefaultInclude(),
            take: limit,
            orderBy: { createdAt: "desc" },
        });
    }

    /**
     * Get user's pending invites
     */
    async getUserPendingInvites(userId: bigint): Promise<chat_invite[]> {
        return await this.prisma.chat_invite.findMany({
            where: {
                userId,
                status: ChatInviteStatus.Pending,
            },
            include: this.getDefaultInclude(),
            orderBy: { createdAt: "desc" },
        });
    }

    /**
     * Create test invites with different statuses
     */
    async createTestInviteSet(chatId: bigint, userIds: bigint[]): Promise<{
        pending: chat_invite[];
        accepted: chat_invite[];
        declined: chat_invite[];
    }> {
        if (userIds.length < 3) {
            throw new Error("Need at least 3 user IDs for test set");
        }

        const pending = await this.createPendingInvite(
            chatId,
            userIds[0],
            "Pending invitation",
        );

        const acceptedInvite = await this.createPendingInvite(
            chatId,
            userIds[1],
            "This will be accepted",
        );
        const accepted = await this.acceptInvite(acceptedInvite.id);

        const declinedInvite = await this.createPendingInvite(
            chatId,
            userIds[2],
            "This will be declined",
        );
        const declined = await this.declineInvite(declinedInvite.id);

        return {
            pending: [pending],
            accepted: [accepted],
            declined: [declined],
        };
    }
}

// Export factory creator function
export const createChatInviteDbFactory = (prisma: PrismaClient) =>
    ChatInviteDbFactory.getInstance("chat_invite", prisma);

// Export the class for type usage
export { ChatInviteDbFactory as ChatInviteDbFactoryClass };
