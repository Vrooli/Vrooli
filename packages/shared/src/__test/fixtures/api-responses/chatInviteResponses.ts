/* c8 ignore start */
/**
 * Chat Invite API Response Fixtures
 * 
 * Comprehensive fixtures for chat invitation management including
 * invite creation, acceptance/decline, and invite status tracking.
 */

import type {
    ChatInvite,
    ChatInviteCreateInput,
    ChatInviteUpdateInput,
    ChatInviteStatus,
    Chat,
    User,
} from "../../../api/types.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";
import { generatePK } from "../../../id/index.js";
import { chatResponseFactory } from "./chatResponses.js";
import { userResponseFactory } from "./userResponses.js";

// Constants
const DEFAULT_COUNT = 10;
const DEFAULT_ERROR_RATE = 0.1;
const DEFAULT_DELAY_MS = 500;
const MAX_MESSAGE_LENGTH = 500;
const INVITE_EXPIRY_DAYS = 7;

// Chat invite statuses
const CHAT_INVITE_STATUSES = ["Pending", "Accepted", "Declined"] as const;

/**
 * Chat Invite API response factory
 */
export class ChatInviteResponseFactory extends BaseAPIResponseFactory<
    ChatInvite,
    ChatInviteCreateInput,
    ChatInviteUpdateInput
> {
    protected readonly entityName = "chat_invite";

    /**
     * Create mock chat invite data
     */
    createMockData(options?: MockDataOptions): ChatInvite {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const inviteId = options?.overrides?.id || generatePK().toString();

        const baseChatInvite: ChatInvite = {
            __typename: "ChatInvite",
            id: inviteId,
            created_at: now,
            updated_at: now,
            message: "You've been invited to join this chat!",
            status: "Pending",
            chat: chatResponseFactory.createMockData(),
            user: userResponseFactory.createMockData(),
            you: {
                canDelete: true,
                canUpdate: true,
            },
        };

        if (scenario === "complete" || scenario === "edge-case") {
            return {
                ...baseChatInvite,
                message: scenario === "edge-case" 
                    ? null 
                    : "Hey! I'd love to discuss the project with you. Can you join our team chat?",
                status: scenario === "edge-case" ? "Declined" : "Accepted",
                chat: chatResponseFactory.createMockData({ scenario: "complete" }),
                user: userResponseFactory.createMockData({ scenario: "complete" }),
                updated_at: scenario === "edge-case" 
                    ? new Date(Date.now() - (24 * 60 * 60 * 1000)).toISOString() // Updated 1 day ago
                    : new Date(Date.now() - (60 * 60 * 1000)).toISOString(), // Updated 1 hour ago
                you: {
                    canDelete: scenario !== "edge-case",
                    canUpdate: scenario !== "edge-case",
                },
                ...options?.overrides,
            };
        }

        return {
            ...baseChatInvite,
            ...options?.overrides,
        };
    }

    /**
     * Create chat invite from input
     */
    createFromInput(input: ChatInviteCreateInput): ChatInvite {
        const now = new Date().toISOString();
        const inviteId = generatePK().toString();

        return {
            __typename: "ChatInvite",
            id: inviteId,
            created_at: now,
            updated_at: now,
            message: input.message || null,
            status: "Pending",
            chat: chatResponseFactory.createMockData({ overrides: { id: input.chatConnect } }),
            user: userResponseFactory.createMockData({ overrides: { id: input.userConnect } }),
            you: {
                canDelete: true,
                canUpdate: true,
            },
        };
    }

    /**
     * Update chat invite from input
     */
    updateFromInput(existing: ChatInvite, input: ChatInviteUpdateInput): ChatInvite {
        const updates: Partial<ChatInvite> = {
            updated_at: new Date().toISOString(),
        };

        if (input.message !== undefined) updates.message = input.message;
        if (input.status !== undefined) updates.status = input.status;

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: ChatInviteCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.chatConnect) {
            errors.chatConnect = "Chat ID is required";
        }

        if (!input.userConnect) {
            errors.userConnect = "User ID is required";
        }

        if (input.message && input.message.length > MAX_MESSAGE_LENGTH) {
            errors.message = `Message must be ${MAX_MESSAGE_LENGTH} characters or less`;
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: ChatInviteUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.message !== undefined && input.message && input.message.length > MAX_MESSAGE_LENGTH) {
            errors.message = `Message must be ${MAX_MESSAGE_LENGTH} characters or less`;
        }

        if (input.status !== undefined && !CHAT_INVITE_STATUSES.includes(input.status as any)) {
            errors.status = `Status must be one of: ${CHAT_INVITE_STATUSES.join(", ")}`;
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create chat invites for different statuses
     */
    createChatInvitesForAllStatuses(): ChatInvite[] {
        return CHAT_INVITE_STATUSES.map((status, index) =>
            this.createMockData({
                overrides: {
                    id: `invite_${status.toLowerCase()}_${index}`,
                    status: status as ChatInviteStatus,
                    created_at: new Date(Date.now() - (index * 24 * 60 * 60 * 1000)).toISOString(),
                    updated_at: status !== "Pending" 
                        ? new Date(Date.now() - (index * 12 * 60 * 60 * 1000)).toISOString()
                        : new Date(Date.now() - (index * 24 * 60 * 60 * 1000)).toISOString(),
                },
            }),
        );
    }

    /**
     * Create chat invites for a specific chat
     */
    createChatInvitesForChat(chatId: string, count = 5): ChatInvite[] {
        const chat = chatResponseFactory.createMockData({ overrides: { id: chatId } });
        
        return Array.from({ length: count }, (_, index) =>
            this.createMockData({
                overrides: {
                    id: `invite_${chatId}_${index}`,
                    chat,
                    user: userResponseFactory.createMockData({ 
                        overrides: { 
                            id: `user_invite_${index}`,
                            name: `Invited User ${index + 1}`,
                            handle: `invited_user_${index + 1}`,
                        },
                    }),
                    status: index === 0 ? "Pending" : (index % 2 === 0 ? "Accepted" : "Declined"),
                    created_at: new Date(Date.now() - (index * 2 * 60 * 60 * 1000)).toISOString(),
                },
            }),
        );
    }

    /**
     * Create chat invites for a specific user
     */
    createChatInvitesForUser(userId: string, count = 3): ChatInvite[] {
        const user = userResponseFactory.createMockData({ overrides: { id: userId } });
        
        return Array.from({ length: count }, (_, index) =>
            this.createMockData({
                overrides: {
                    id: `invite_user_${userId}_${index}`,
                    user,
                    chat: chatResponseFactory.createMockData({ 
                        overrides: { 
                            id: `chat_for_user_${index}`,
                        },
                    }),
                    status: index === 0 ? "Pending" : (index === 1 ? "Accepted" : "Declined"),
                    message: index === 0 ? "You've been invited to join our project discussion!" : null,
                    created_at: new Date(Date.now() - (index * 4 * 60 * 60 * 1000)).toISOString(),
                },
            }),
        );
    }

    /**
     * Create different chat invite scenarios
     */
    createChatInviteScenarios(): ChatInvite[] {
        const baseTime = Date.now();
        
        return [
            // Fresh pending invite
            this.createMockData({
                overrides: {
                    id: "fresh_pending_invite",
                    status: "Pending",
                    message: "Join our team discussion about the new features!",
                    created_at: new Date(baseTime - (2 * 60 * 60 * 1000)).toISOString(), // 2 hours ago
                },
            }),

            // Recently accepted invite
            this.createMockData({
                overrides: {
                    id: "recent_accepted_invite",
                    status: "Accepted",
                    message: "Welcome to our development chat!",
                    created_at: new Date(baseTime - (24 * 60 * 60 * 1000)).toISOString(), // 1 day ago
                    updated_at: new Date(baseTime - (2 * 60 * 60 * 1000)).toISOString(), // 2 hours ago
                },
            }),

            // Declined invite
            this.createMockData({
                overrides: {
                    id: "declined_invite",
                    status: "Declined",
                    message: "Chat about project roadmap and planning",
                    created_at: new Date(baseTime - (3 * 24 * 60 * 60 * 1000)).toISOString(), // 3 days ago
                    updated_at: new Date(baseTime - (2 * 24 * 60 * 60 * 1000)).toISOString(), // 2 days ago
                },
            }),

            // Invite without custom message
            this.createMockData({
                overrides: {
                    id: "no_message_invite",
                    status: "Pending",
                    message: null,
                    created_at: new Date(baseTime - (6 * 60 * 60 * 1000)).toISOString(), // 6 hours ago
                },
            }),

            // Old pending invite (potentially expired)
            this.createMockData({
                overrides: {
                    id: "old_pending_invite",
                    status: "Pending",
                    message: "Join our weekly sync chat",
                    created_at: new Date(baseTime - (INVITE_EXPIRY_DAYS * 24 * 60 * 60 * 1000)).toISOString(),
                },
            }),
        ];
    }

    /**
     * Create invite already processed error response
     */
    createInviteAlreadyProcessedErrorResponse(status: ChatInviteStatus) {
        return this.createBusinessErrorResponse("already_processed", {
            resource: "chat_invite",
            currentStatus: status,
            allowedActions: status === "Pending" ? ["accept", "decline"] : [],
            message: `This chat invite has already been ${status.toLowerCase()}`,
        });
    }

    /**
     * Create invite expired error response
     */
    createInviteExpiredErrorResponse(inviteId: string, expiredAt: string) {
        return this.createBusinessErrorResponse("expired", {
            resource: "chat_invite",
            inviteId,
            expiredAt,
            expiryDays: INVITE_EXPIRY_DAYS,
            message: `Chat invite expired after ${INVITE_EXPIRY_DAYS} days`,
        });
    }

    /**
     * Create user already in chat error response
     */
    createUserAlreadyInChatErrorResponse(userId: string, chatId: string) {
        return this.createBusinessErrorResponse("already_member", {
            resource: "chat_invite",
            userId,
            chatId,
            message: "User is already a participant in this chat",
        });
    }

    /**
     * Create self-invite error response
     */
    createSelfInviteErrorResponse() {
        return this.createValidationErrorResponse({
            userConnect: "You cannot invite yourself to a chat",
        });
    }
}

/**
 * Pre-configured chat invite response scenarios
 */
export const chatInviteResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<ChatInviteCreateInput>) => {
        const factory = new ChatInviteResponseFactory();
        const defaultInput: ChatInviteCreateInput = {
            chatConnect: generatePK().toString(),
            userConnect: generatePK().toString(),
            message: "You've been invited to join this chat!",
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    findSuccess: (chatInvite?: ChatInvite) => {
        const factory = new ChatInviteResponseFactory();
        return factory.createSuccessResponse(
            chatInvite || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new ChatInviteResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: ChatInvite, updates?: Partial<ChatInviteUpdateInput>) => {
        const factory = new ChatInviteResponseFactory();
        const chatInvite = existing || factory.createMockData({ scenario: "complete" });
        const input: ChatInviteUpdateInput = {
            id: chatInvite.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(chatInvite, input),
        );
    },

    acceptSuccess: (inviteId?: string) => {
        const factory = new ChatInviteResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    id: inviteId,
                    status: "Accepted",
                    updated_at: new Date().toISOString(),
                },
            }),
        );
    },

    declineSuccess: (inviteId?: string) => {
        const factory = new ChatInviteResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    id: inviteId,
                    status: "Declined",
                    updated_at: new Date().toISOString(),
                },
            }),
        );
    },

    listSuccess: (chatInvites?: ChatInvite[]) => {
        const factory = new ChatInviteResponseFactory();
        return factory.createPaginatedResponse(
            chatInvites || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: chatInvites?.length || DEFAULT_COUNT },
        );
    },

    allStatusesSuccess: () => {
        const factory = new ChatInviteResponseFactory();
        const invites = factory.createChatInvitesForAllStatuses();
        return factory.createPaginatedResponse(
            invites,
            { page: 1, totalCount: invites.length },
        );
    },

    chatInvitesSuccess: (chatId?: string) => {
        const factory = new ChatInviteResponseFactory();
        const invites = factory.createChatInvitesForChat(chatId || generatePK().toString());
        return factory.createPaginatedResponse(
            invites,
            { page: 1, totalCount: invites.length },
        );
    },

    userInvitesSuccess: (userId?: string) => {
        const factory = new ChatInviteResponseFactory();
        const invites = factory.createChatInvitesForUser(userId || generatePK().toString());
        return factory.createPaginatedResponse(
            invites,
            { page: 1, totalCount: invites.length },
        );
    },

    pendingInvitesSuccess: () => {
        const factory = new ChatInviteResponseFactory();
        const invites = factory.createChatInviteScenarios().filter(invite => invite.status === "Pending");
        return factory.createPaginatedResponse(
            invites,
            { page: 1, totalCount: invites.length },
        );
    },

    scenariosSuccess: () => {
        const factory = new ChatInviteResponseFactory();
        const scenarios = factory.createChatInviteScenarios();
        return factory.createPaginatedResponse(
            scenarios,
            { page: 1, totalCount: scenarios.length },
        );
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new ChatInviteResponseFactory();
        return factory.createValidationErrorResponse({
            chatConnect: "Chat ID is required",
            userConnect: "User ID is required",
        });
    },

    notFoundError: (inviteId?: string) => {
        const factory = new ChatInviteResponseFactory();
        return factory.createNotFoundErrorResponse(
            inviteId || "non-existent-invite",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new ChatInviteResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
            ["chat:invite"],
        );
    },

    alreadyProcessedError: (status: ChatInviteStatus = "Accepted") => {
        const factory = new ChatInviteResponseFactory();
        return factory.createInviteAlreadyProcessedErrorResponse(status);
    },

    expiredInviteError: (inviteId?: string) => {
        const factory = new ChatInviteResponseFactory();
        return factory.createInviteExpiredErrorResponse(
            inviteId || generatePK().toString(),
            new Date(Date.now() - (INVITE_EXPIRY_DAYS * 24 * 60 * 60 * 1000)).toISOString(),
        );
    },

    userAlreadyInChatError: (userId?: string, chatId?: string) => {
        const factory = new ChatInviteResponseFactory();
        return factory.createUserAlreadyInChatErrorResponse(
            userId || generatePK().toString(),
            chatId || generatePK().toString(),
        );
    },

    selfInviteError: () => {
        const factory = new ChatInviteResponseFactory();
        return factory.createSelfInviteErrorResponse();
    },

    messageTooLongError: () => {
        const factory = new ChatInviteResponseFactory();
        return factory.createValidationErrorResponse({
            message: `Message must be ${MAX_MESSAGE_LENGTH} characters or less`,
        });
    },

    // MSW handlers
    handlers: {
        success: () => new ChatInviteResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new ChatInviteResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new ChatInviteResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const chatInviteResponseFactory = new ChatInviteResponseFactory();
