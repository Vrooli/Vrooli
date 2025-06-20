import { generatePK, InviteStatus } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface ChatInviteRelationConfig extends RelationConfig {
    chat: { chatId: string };
    user: { userId: string };
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
    Prisma.chat_invite,
    Prisma.chat_inviteCreateInput,
    Prisma.chat_inviteInclude,
    Prisma.chat_inviteUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('chat_invite', prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.chat_invite;
    }

    /**
     * Get complete test fixtures for ChatInvite model
     */
    protected getFixtures(): DbTestFixtures<Prisma.ChatInviteCreateInput, Prisma.ChatInviteUpdateInput> {
        return {
            minimal: {
                id: generatePK().toString(),
                status: InviteStatus.Pending,
                chat: { connect: { id: "chat-123" } },
                user: { connect: { id: "user-123" } },
            },
            complete: {
                id: generatePK().toString(),
                status: InviteStatus.Pending,
                message: "You're invited to join our team discussion about the new project!",
                chat: { connect: { id: "chat-456" } },
                user: { connect: { id: "user-456" } },
            },
            invalid: {
                missingRequired: {
                    // Missing id, chat, user
                    status: InviteStatus.Pending,
                    message: "Invalid invite",
                },
                invalidTypes: {
                    id: "not-a-snowflake",
                    status: "pending", // Should be InviteStatus enum
                    message: 123, // Should be string
                    chat: "not-an-object", // Should be object
                    user: null, // Should be object
                },
                duplicateInvite: {
                    id: generatePK().toString(),
                    status: InviteStatus.Pending,
                    chat: { connect: { id: "existing-chat" } },
                    user: { connect: { id: "existing-user" } }, // Assumes this combo exists
                },
                invalidStatus: {
                    id: generatePK().toString(),
                    // @ts-expect-error - Intentionally invalid status
                    status: "InvalidStatus",
                    chat: { connect: { id: "chat-123" } },
                    user: { connect: { id: "user-123" } },
                },
            },
            edgeCases: {
                maxLengthMessage: {
                    id: generatePK().toString(),
                    status: InviteStatus.Pending,
                    message: 'a'.repeat(4096), // Max length message
                    chat: { connect: { id: "chat-789" } },
                    user: { connect: { id: "user-789" } },
                },
                unicodeMessage: {
                    id: generatePK().toString(),
                    status: InviteStatus.Pending,
                    message: "You're invited! 🎉 Join us for a great discussion 💬",
                    chat: { connect: { id: "chat-unicode" } },
                    user: { connect: { id: "user-unicode" } },
                },
                acceptedInvite: {
                    id: generatePK().toString(),
                    status: InviteStatus.Accepted,
                    message: "Thanks for joining!",
                    chat: { connect: { id: "chat-accepted" } },
                    user: { connect: { id: "user-accepted" } },
                },
                declinedInvite: {
                    id: generatePK().toString(),
                    status: InviteStatus.Declined,
                    message: "Sorry, I can't join right now",
                    chat: { connect: { id: "chat-declined" } },
                    user: { connect: { id: "user-declined" } },
                },
                noMessageInvite: {
                    id: generatePK().toString(),
                    status: InviteStatus.Pending,
                    message: null,
                    chat: { connect: { id: "chat-no-msg" } },
                    user: { connect: { id: "user-no-msg" } },
                },
            },
            updates: {
                minimal: {
                    status: InviteStatus.Accepted,
                },
                complete: {
                    status: InviteStatus.Declined,
                    message: "Updated message - Unfortunately I cannot join at this time",
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.ChatInviteCreateInput>): Prisma.ChatInviteCreateInput {
        // Note: chat and user connections must be provided via overrides or config
        return {
            id: generatePK().toString(),
            status: InviteStatus.Pending,
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.ChatInviteCreateInput>): Prisma.ChatInviteCreateInput {
        return {
            id: generatePK().toString(),
            status: InviteStatus.Pending,
            message: "You're invited to join our team discussion!",
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
                        status: InviteStatus.Pending,
                        message: "You've been invited to join our chat",
                    },
                    chat: { chatId: "chat-pending" },
                    user: { userId: "user-pending" },
                },
            },
            acceptedInvite: {
                name: "acceptedInvite",
                description: "Accepted invitation",
                config: {
                    overrides: {
                        status: InviteStatus.Accepted,
                        message: "Welcome! Glad you could join us",
                    },
                    chat: { chatId: "chat-accepted" },
                    user: { userId: "user-accepted" },
                },
            },
            declinedInvite: {
                name: "declinedInvite",
                description: "Declined invitation",
                config: {
                    overrides: {
                        status: InviteStatus.Declined,
                        message: "Sorry, I'm not available",
                    },
                    chat: { chatId: "chat-declined" },
                    user: { userId: "user-declined" },
                },
            },
            bulkTeamInvites: {
                name: "bulkTeamInvites",
                description: "Multiple invitations for team chat",
                config: {
                    overrides: {
                        status: InviteStatus.Pending,
                        message: "Join our team chat for project updates",
                    },
                    chat: { chatId: "chat-team" },
                    user: { userId: "user-bulk-1" },
                },
            },
            personalInvite: {
                name: "personalInvite",
                description: "Personal invitation with custom message",
                config: {
                    overrides: {
                        status: InviteStatus.Pending,
                        message: "Hey! I'd love to chat with you about the new features we're building. Are you available?",
                    },
                    chat: { chatId: "chat-personal" },
                    user: { userId: "user-personal" },
                },
            },
        };
    }

    protected getDefaultInclude(): Prisma.ChatInviteInclude {
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
        baseData: Prisma.ChatInviteCreateInput,
        config: ChatInviteRelationConfig,
        tx: any
    ): Promise<Prisma.ChatInviteCreateInput> {
        let data = { ...baseData };

        // Handle chat connection (required)
        if (config.chat) {
            data.chat = {
                connect: { id: config.chat.chatId },
            };
        } else {
            throw new Error('ChatInvite requires a chat connection');
        }

        // Handle user connection (required)
        if (config.user) {
            data.user = {
                connect: { id: config.user.userId },
            };
        } else {
            throw new Error('ChatInvite requires a user connection');
        }

        return data;
    }

    /**
     * Create a pending invite
     */
    async createPendingInvite(chatId: string, userId: string, message?: string): Promise<Prisma.ChatInvite> {
        return await this.createWithRelations({
            overrides: {
                status: InviteStatus.Pending,
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
        chatId: string,
        userIds: string[],
        message?: string
    ): Promise<Prisma.ChatInvite[]> {
        const invites = await Promise.all(
            userIds.map(userId => 
                this.createPendingInvite(chatId, userId, message)
            )
        );
        return invites;
    }

    /**
     * Accept an invite
     */
    async acceptInvite(inviteId: string): Promise<Prisma.ChatInvite> {
        const updated = await this.prisma.chatInvite.update({
            where: { id: inviteId },
            data: { status: InviteStatus.Accepted },
            include: this.getDefaultInclude(),
        });
        return updated;
    }

    /**
     * Decline an invite
     */
    async declineInvite(inviteId: string): Promise<Prisma.ChatInvite> {
        const updated = await this.prisma.chatInvite.update({
            where: { id: inviteId },
            data: { status: InviteStatus.Declined },
            include: this.getDefaultInclude(),
        });
        return updated;
    }

    protected async checkModelConstraints(record: Prisma.ChatInvite): Promise<string[]> {
        const violations: string[] = [];
        
        // Check for duplicate invite
        const duplicate = await this.prisma.chatInvite.findFirst({
            where: {
                chatId: record.chatId,
                userId: record.userId,
                id: { not: record.id },
            },
        });
        
        if (duplicate) {
            violations.push('User already has an invite to this chat');
        }

        // Check if user is already a participant
        const existingParticipant = await this.prisma.chatParticipant.findFirst({
            where: {
                chatId: record.chatId,
                userId: record.userId,
            },
        });
        
        if (existingParticipant) {
            violations.push('User is already a participant in this chat');
        }

        // Check message length
        if (record.message && record.message.length > 4096) {
            violations.push('Message exceeds maximum length of 4096 characters');
        }

        // Check valid status
        if (!Object.values(InviteStatus).includes(record.status as InviteStatus)) {
            violations.push('Invalid invite status');
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            // ChatInvite has no dependent records
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.ChatInvite,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[]
    ): Promise<void> {
        // ChatInvite has no dependent records to delete
    }

    /**
     * Get invites by status
     */
    async getInvitesByStatus(status: InviteStatus, limit?: number): Promise<Prisma.ChatInvite[]> {
        return await this.prisma.chatInvite.findMany({
            where: { status },
            include: this.getDefaultInclude(),
            take: limit,
            orderBy: { createdAt: 'desc' },
        });
    }

    /**
     * Get user's pending invites
     */
    async getUserPendingInvites(userId: string): Promise<Prisma.ChatInvite[]> {
        return await this.prisma.chatInvite.findMany({
            where: {
                userId,
                status: InviteStatus.Pending,
            },
            include: this.getDefaultInclude(),
            orderBy: { createdAt: 'desc' },
        });
    }

    /**
     * Create test invites with different statuses
     */
    async createTestInviteSet(chatId: string, userIds: string[]): Promise<{
        pending: Prisma.ChatInvite[];
        accepted: Prisma.ChatInvite[];
        declined: Prisma.ChatInvite[];
    }> {
        if (userIds.length < 3) {
            throw new Error('Need at least 3 user IDs for test set');
        }

        const pending = await this.createPendingInvite(
            chatId,
            userIds[0],
            "Pending invitation"
        );

        const acceptedInvite = await this.createPendingInvite(
            chatId,
            userIds[1],
            "This will be accepted"
        );
        const accepted = await this.acceptInvite(acceptedInvite.id);

        const declinedInvite = await this.createPendingInvite(
            chatId,
            userIds[2],
            "This will be declined"
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
    ChatInviteDbFactory.getInstance('ChatInvite', prisma);

// Export the class for type usage
export { ChatInviteDbFactory as ChatInviteDbFactoryClass };