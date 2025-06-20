import { type ChatInviteCreateInput, type ChatInviteUpdateInput, generatePK } from "@vrooli/shared";
import { nanoid } from "nanoid";
import { type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface ChatInviteRelationConfig extends RelationConfig {
    chatId?: string;
    userId?: string;
    status?: "Pending" | "Accepted" | "Declined";
    withCustomMessage?: boolean;
    customMessage?: string;
}

/**
 * Database fixture factory for ChatInvite model
 * Handles invitation workflows with expiry, status tracking, and custom messages
 */
export class ChatInviteDbFactory extends DatabaseFixtureFactory<
    any,
    ChatInviteCreateInput,
    any,
    ChatInviteUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('chat_invite', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.chat_invite;
    }

    protected getMinimalData(overrides?: Partial<ChatInviteCreateInput>): ChatInviteCreateInput {
        // Generate placeholder IDs - these should be overridden with actual chat/user IDs
        const chatId = BigInt(generatePK());
        const userId = BigInt(generatePK());
        
        return {
            id: generatePK(),
            message: "You're invited to join this chat!",
            chatId,
            userId,
            status: "Pending",
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<ChatInviteCreateInput>): ChatInviteCreateInput {
        // Generate placeholder IDs - these should be overridden with actual chat/user IDs
        const chatId = BigInt(generatePK());
        const userId = BigInt(generatePK());
        
        return {
            id: generatePK(),
            message: "Welcome to our discussion! We'd love to have you join us in this conversation where we share ideas, collaborate on projects, and build meaningful connections.",
            chatId,
            userId,
            status: "Pending",
            ...overrides,
        };
    }

    /**
     * Create invite with specific chat and user
     */
    async createInvite(
        chatId: string, 
        userId: string, 
        message?: string,
        overrides?: Partial<ChatInviteCreateInput>
    ): Promise<any> {
        const data: ChatInviteCreateInput = {
            ...this.getMinimalData(),
            chatId: BigInt(chatId),
            userId: BigInt(userId),
            message: message || "You're invited to join this chat!",
            ...overrides,
        };
        
        const result = await this.prisma.chat_invite.create({ data });
        this.trackCreatedId(result.id.toString());
        return result;
    }

    /**
     * Create pending invite
     */
    async createPending(
        chatId: string, 
        userId: string, 
        message?: string
    ): Promise<any> {
        return this.createInvite(chatId, userId, message, { status: "Pending" });
    }

    /**
     * Create accepted invite
     */
    async createAccepted(
        chatId: string, 
        userId: string, 
        message?: string
    ): Promise<any> {
        return this.createInvite(chatId, userId, message, { status: "Accepted" });
    }

    /**
     * Create declined invite
     */
    async createDeclined(
        chatId: string, 
        userId: string, 
        message?: string
    ): Promise<any> {
        return this.createInvite(chatId, userId, message, { status: "Declined" });
    }

    /**
     * Create invite with custom message
     */
    async createWithCustomMessage(
        chatId: string, 
        userId: string, 
        customMessage: string
    ): Promise<any> {
        return this.createInvite(chatId, userId, customMessage);
    }

    /**
     * Create public chat invite
     */
    async createPublicInvite(
        chatId: string, 
        creatorId: string, 
        chatName: string
    ): Promise<any> {
        return this.createInvite(chatId, creatorId, `Join the ${chatName} discussion! Open to everyone.`);
    }

    /**
     * Create team invite
     */
    async createTeamInvite(
        chatId: string, 
        inviterId: string, 
        teamName: string
    ): Promise<any> {
        return this.createInvite(chatId, inviterId, `You're invited to join the ${teamName} team chat. This is a private discussion for team members.`);
    }

    protected getDefaultInclude(): any {
        return {
            chat: {
                select: {
                    id: true,
                    publicId: true,
                    isPrivate: true,
                    openToAnyoneWithInvite: true,
                    translations: {
                        where: { language: 'en' },
                        select: {
                            name: true,
                            description: true,
                        },
                        take: 1,
                    },
                    team: {
                        select: {
                            id: true,
                            publicId: true,
                            name: true,
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
                    profileImage: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: ChatInviteCreateInput,
        config: ChatInviteRelationConfig,
        tx: any
    ): Promise<ChatInviteCreateInput> {
        let data = { ...baseData };

        // Apply chat and user connections
        if (config.chatId) {
            data.chatId = BigInt(config.chatId);
        }
        
        if (config.userId) {
            data.userId = BigInt(config.userId);
        }

        // Apply status
        if (config.status) {
            data.status = config.status;
        }

        // Apply custom message
        if (config.withCustomMessage && config.customMessage) {
            data.message = config.customMessage;
        }

        return data;
    }

    /**
     * Accept an invite and add user as participant
     */
    async acceptInvite(inviteId: string): Promise<{ 
        invite: any; 
        participant?: any;
        success: boolean;
        message: string;
    }> {
        return await this.prisma.$transaction(async (tx) => {
            // Get the invite
            const invite = await tx.chat_invite.findUnique({
                where: { id: BigInt(inviteId) },
                include: { chat: true, user: true },
            });

            if (!invite) {
                return { 
                    invite: null as any, 
                    success: false, 
                    message: "Invite not found" 
                };
            }

            if (invite.status !== "Pending") {
                return { 
                    invite, 
                    success: false, 
                    message: "Invite already responded to" 
                };
            }

            // Update invite status
            const updatedInvite = await tx.chat_invite.update({
                where: { id: BigInt(inviteId) },
                data: { status: "Accepted" },
            });

            // Check if user is already a participant
            const existingParticipant = await tx.chat_participants.findUnique({
                where: {
                    chatId_userId: {
                        chatId: invite.chatId,
                        userId: invite.userId,
                    },
                },
            });

            let participant = existingParticipant;

            // Add user as participant if not already present
            if (!existingParticipant) {
                participant = await tx.chat_participants.create({
                    data: {
                        id: generatePK(),
                        chatId: invite.chatId,
                        userId: invite.userId,
                        hasUnread: false,
                    },
                });
            }

            return { 
                invite: updatedInvite, 
                participant,
                success: true, 
                message: "Successfully joined chat" 
            };
        });
    }

    /**
     * Decline an invite
     */
    async declineInvite(inviteId: string): Promise<{ 
        invite: any; 
        success: boolean;
        message: string;
    }> {
        const invite = await this.prisma.chat_invite.findUnique({
            where: { id: BigInt(inviteId) },
        });

        if (!invite) {
            return { 
                invite: null as any, 
                success: false, 
                message: "Invite not found" 
            };
        }

        if (invite.status !== "Pending") {
            return { 
                invite, 
                success: false, 
                message: "Invite already responded to" 
            };
        }

        const updatedInvite = await this.prisma.chat_invite.update({
            where: { id: BigInt(inviteId) },
            data: { status: "Declined" },
        });

        return { 
            invite: updatedInvite, 
            success: true, 
            message: "Invite declined" 
        };
    }

    /**
     * Bulk create invites
     */
    async createBulkInvites(
        chatId: string, 
        creatorId: string, 
        messages: string[]
    ): Promise<any[]> {
        const invites: any[] = [];
        
        for (const message of messages) {
            const invite = await this.createInvite(chatId, creatorId, message);
            invites.push(invite);
        }
        
        return invites;
    }

    /**
     * Create multiple invites for different users
     */
    async createInvitesForUsers(
        chatId: string, 
        userIds: string[], 
        baseMessage: string = "You're invited to join this chat!"
    ): Promise<Prisma.chat_invite[]> {
        const invites: any[] = [];
        
        for (let i = 0; i < userIds.length; i++) {
            const userId = userIds[i];
            const customMessage = `${baseMessage} Invite ${i + 1}`;
            const invite = await this.createInvite(chatId, userId, customMessage);
            invites.push(invite);
        }
        
        return invites;
    }

    protected async checkModelConstraints(record: any): Promise<string[]> {
        const violations: string[] = [];
        
        // Check message length
        if (record.message && record.message.length > 4096) {
            violations.push('Message exceeds maximum length of 4096 characters');
        }

        if (!record.message || record.message.trim().length === 0) {
            violations.push('Invite message cannot be empty');
        }

        // Check valid status
        if (!['Pending', 'Accepted', 'Declined'].includes(record.status)) {
            violations.push('Invalid status value');
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

        // Business logic: Check for duplicate pending invites
        const duplicateInvite = await this.prisma.chat_invite.findFirst({
            where: { 
                chatId: record.chatId,
                userId: record.userId,
                status: 'Pending',
                id: { not: record.id },
            },
        });
        if (duplicateInvite) {
            violations.push('User already has a pending invite for this chat');
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing id, chatId, userId, message, status
                id: generatePK(),
            },
            invalidTypes: {
                id: "not-a-snowflake",
                message: 123, // Should be string
                chatId: "invalid-chat-reference", // Should be BigInt
                userId: "invalid-user-reference", // Should be BigInt
                status: "InvalidStatus", // Not a valid status
            },
            emptyMessage: {
                id: generatePK(),
                message: "", // Empty message
                chatId: BigInt(generatePK()),
                userId: BigInt(generatePK()),
                status: "Pending",
            },
            messageTooLong: {
                id: generatePK(),
                message: "a".repeat(5000), // Too long
                chatId: BigInt(generatePK()),
                userId: BigInt(generatePK()),
                status: "Pending",
            },
            invalidStatus: {
                id: generatePK(),
                message: "Valid message",
                chatId: BigInt(generatePK()),
                userId: BigInt(generatePK()),
                status: "InvalidStatus" as any,
            },
            nonExistentChat: {
                id: generatePK(),
                message: "Valid message",
                chatId: BigInt("999999999999999999"), // Non-existent chat
                userId: BigInt(generatePK()),
                status: "Pending",
            },
            nonExistentUser: {
                id: generatePK(),
                message: "Valid message",
                chatId: BigInt(generatePK()),
                userId: BigInt("999999999999999999"), // Non-existent user
                status: "Pending",
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, ChatInviteCreateInput> {
        const baseChatId = BigInt(generatePK());
        const baseUserId = BigInt(generatePK());
        
        return {
            pendingInvite: {
                ...this.getMinimalData(),
                chatId: baseChatId,
                userId: baseUserId,
                message: "Pending invitation",
                status: "Pending",
            },
            acceptedInvite: {
                ...this.getMinimalData(),
                chatId: baseChatId,
                userId: BigInt(generatePK()),
                message: "Accepted invitation",
                status: "Accepted",
            },
            declinedInvite: {
                ...this.getMinimalData(),
                chatId: baseChatId,
                userId: BigInt(generatePK()),
                message: "Declined invitation",
                status: "Declined",
            },
            shortMessage: {
                ...this.getMinimalData(),
                chatId: baseChatId,
                userId: BigInt(generatePK()),
                message: "Hi!",
                status: "Pending",
            },
            longMessage: {
                ...this.getMinimalData(),
                chatId: baseChatId,
                userId: BigInt(generatePK()),
                message: "This is a very long invitation message that goes into great detail about the chat, its purpose, the participants, the expected discussion topics, and all the reasons why the invitee should consider joining this particular conversation. ".repeat(5),
                status: "Pending",
            },
            specialCharactersMessage: {
                ...this.getMinimalData(),
                chatId: baseChatId,
                userId: BigInt(generatePK()),
                message: "Welcome! 🚀 Join us for a great discussion 💬 with special chars: @#$%^&*()_+{}|:<>?[]\\/.,;'\"",
                status: "Pending",
            },
            multipleChatsInvite: {
                ...this.getMinimalData(),
                chatId: BigInt(generatePK()), // Different chat
                userId: baseUserId, // Same user
                message: "Invitation to second chat",
                status: "Pending",
            },
            teamInviteMessage: {
                ...this.getMinimalData(),
                chatId: baseChatId,
                userId: BigInt(generatePK()),
                message: "You're invited to join our team's private discussion. This chat is for strategic planning and confidential team matters.",
                status: "Pending",
            },
            publicChatInvite: {
                ...this.getMinimalData(),
                chatId: baseChatId,
                userId: BigInt(generatePK()),
                message: "Join our public discussion! Everyone is welcome to participate and share ideas.",
                status: "Pending",
            },
            urgentInvite: {
                ...this.getMinimalData(),
                chatId: baseChatId,
                userId: BigInt(generatePK()),
                message: "🚨 URGENT: You're needed in the emergency response chat. Please join immediately.",
                status: "Pending",
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            // ChatInvite doesn't have child relationships to cascade
        };
    }

    protected async deleteRelatedRecords(
        record: any,
        remainingDepth: number,
        tx: any
    ): Promise<void> {
        // ChatInvite doesn't have dependent records to delete
        // It's an independent entity that references chat and user
    }

    /**
     * Test scenario helpers
     */
    async createInviteWorkflow(
        chatId: string,
        creatorId: string,
        inviteeIds: string[]
    ): Promise<{
        pending: any[];
        accepted: any[];
        declined: any[];
    }> {
        const pending: any[] = [];
        const accepted: any[] = [];
        const declined: any[] = [];

        for (let i = 0; i < inviteeIds.length; i++) {
            const inviteeId = inviteeIds[i];
            const status = i % 3 === 0 ? "Pending" : i % 3 === 1 ? "Accepted" : "Declined";
            
            const invite = await this.createInvite(chatId, creatorId, 
                `Invite for user ${i + 1}`, { status });
            
            if (status === "Pending") pending.push(invite);
            else if (status === "Accepted") accepted.push(invite);
            else declined.push(invite);
        }

        return { pending, accepted, declined };
    }

    async simulateInviteUsage(
        chatId: string,
        creatorId: string,
        scenarios: Array<{
            message: string;
            shouldAccept: boolean;
            shouldDecline: boolean;
        }>
    ): Promise<Array<{
        invite: any;
        participant?: any;
        finalStatus: string;
    }>> {
        const results = [];

        for (const scenario of scenarios) {
            const invite = await this.createInvite(chatId, creatorId, scenario.message);
            
            let finalStatus = "Pending";
            let participant = undefined;

            if (scenario.shouldAccept) {
                const result = await this.acceptInvite(invite.id);
                finalStatus = "Accepted";
                participant = result.participant;
            } else if (scenario.shouldDecline) {
                await this.declineInvite(invite.id);
                finalStatus = "Declined";
            }

            results.push({
                invite: {
                    ...invite,
                    status: finalStatus,
                },
                participant,
                finalStatus,
            });
        }

        return results;
    }
}

// Export factory creator function
export const createChatInviteDbFactory = (prisma: PrismaClient) => new ChatInviteDbFactory(prisma);