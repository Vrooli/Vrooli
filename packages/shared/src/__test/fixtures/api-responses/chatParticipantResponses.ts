/* c8 ignore start */
/**
 * Chat Participant API Response Fixtures
 * 
 * Comprehensive fixtures for chat participant management including
 * user roles, permissions, participation tracking, and member management.
 */

import type {
    ChatParticipant,
    ChatParticipantUpdateInput,
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
const MAX_PARTICIPANTS_PER_CHAT = 100;
const MILLISECONDS_PER_HOUR = 60 * 60 * 1000;
const MILLISECONDS_PER_DAY = 24 * MILLISECONDS_PER_HOUR;
const DAYS_IN_WEEK = 7;
const HOURS_IN_2 = 2;
const HOURS_IN_6 = 6;
const DAYS_IN_3 = 3;

// Participant roles (if any - some systems have admin/member roles)
const PARTICIPANT_ROLES = ["member", "admin", "moderator"] as const;

/**
 * Chat Participant API response factory
 */
export class ChatParticipantResponseFactory extends BaseAPIResponseFactory<
    ChatParticipant,
    never, // No create input - participants are added differently
    ChatParticipantUpdateInput
> {
    protected readonly entityName = "chat_participant";

    /**
     * Create mock chat participant data
     */
    createMockData(options?: MockDataOptions): ChatParticipant {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const participantId = options?.overrides?.id || generatePK().toString();

        const baseParticipant: ChatParticipant = {
            __typename: "ChatParticipant",
            id: participantId,
            created_at: now,
            updated_at: now,
            chat: chatResponseFactory.createMockData(),
            user: userResponseFactory.createMockData(),
            you: {
                canDelete: false, // Usually can't delete yourself
                canUpdate: false, // Usually can't update your own participation
            },
        };

        if (scenario === "complete" || scenario === "edge-case") {
            return {
                ...baseParticipant,
                chat: chatResponseFactory.createMockData({ scenario: "complete" }),
                user: scenario === "edge-case" 
                    ? userResponseFactory.createMockData({ 
                        overrides: { 
                            isBot: true, 
                            name: "System Bot",
                            handle: "system_bot",
                        },
                    })
                    : userResponseFactory.createMockData({ scenario: "complete" }),
                created_at: scenario === "edge-case"
                    ? new Date(Date.now() - (DAYS_IN_WEEK * MILLISECONDS_PER_DAY)).toISOString() // 1 week ago
                    : new Date(Date.now() - (HOURS_IN_2 * MILLISECONDS_PER_HOUR)).toISOString(), // 2 hours ago
                you: {
                    canDelete: scenario === "complete", // Admin can remove participants
                    canUpdate: scenario === "complete", // Admin can update participant settings
                },
                ...options?.overrides,
            };
        }

        return {
            ...baseParticipant,
            ...options?.overrides,
        };
    }

    /**
     * Update chat participant from input
     */
    updateFromInput(existing: ChatParticipant, input: ChatParticipantUpdateInput): ChatParticipant {
        const updates: Partial<ChatParticipant> = {
            updated_at: new Date().toISOString(),
        };

        // ChatParticipant updates are typically limited to metadata
        // Most properties are read-only after creation

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: ChatParticipantUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.id) {
            errors.id = "Participant ID is required";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create participants for a chat
     */
    createParticipantsForChat(chatId: string, count = 5): ChatParticipant[] {
        const chat = chatResponseFactory.createMockData({ overrides: { id: chatId } });
        
        return Array.from({ length: count }, (_, index) => {
            const joinTime = new Date(Date.now() - (index * HOURS_IN_6 * MILLISECONDS_PER_HOUR));
            
            return this.createMockData({
                overrides: {
                    id: `participant_${chatId}_${index}`,
                    chat,
                    user: userResponseFactory.createMockData({ 
                        overrides: { 
                            id: `user_${index}`,
                            name: `Participant ${index + 1}`,
                            handle: `participant_${index + 1}`,
                        },
                    }),
                    created_at: joinTime.toISOString(),
                    updated_at: joinTime.toISOString(),
                    you: {
                        canDelete: index === 0, // First participant (creator) can remove others
                        canUpdate: index === 0, // First participant (creator) can update settings
                    },
                },
            });
        });
    }

    /**
     * Create participants for a user (all chats they're in)
     */
    createParticipantsForUser(userId: string, count = 3): ChatParticipant[] {
        const user = userResponseFactory.createMockData({ overrides: { id: userId } });
        
        return Array.from({ length: count }, (_, index) => {
            const joinTime = new Date(Date.now() - (index * DAYS_IN_3 * MILLISECONDS_PER_DAY));
            
            return this.createMockData({
                overrides: {
                    id: `participant_user_${userId}_${index}`,
                    user,
                    chat: chatResponseFactory.createMockData({ 
                        overrides: { 
                            id: `chat_${index}`,
                        },
                    }),
                    created_at: joinTime.toISOString(),
                    updated_at: joinTime.toISOString(),
                    you: {
                        canDelete: false, // Can't delete own participation
                        canUpdate: false, // Can't update own participation
                    },
                },
            });
        });
    }

    /**
     * Create participants with different scenarios
     */
    createParticipantScenarios(): ChatParticipant[] {
        const baseTime = Date.now();
        const sharedChat = chatResponseFactory.createMockData({ 
            overrides: { 
                id: "shared_chat_123",
            },
        });
        
        return [
            // Chat creator/admin
            this.createMockData({
                overrides: {
                    id: "creator_participant",
                    chat: sharedChat,
                    user: userResponseFactory.createMockData({ 
                        overrides: { 
                            name: "Chat Creator",
                            handle: "chat_creator",
                        },
                    }),
                    created_at: new Date(baseTime - (DAYS_IN_WEEK * MILLISECONDS_PER_DAY)).toISOString(), // Created chat 1 week ago
                    you: {
                        canDelete: true, // Can remove other participants
                        canUpdate: true, // Can update participant settings
                    },
                },
            }),

            // Regular active member
            this.createMockData({
                overrides: {
                    id: "active_participant",
                    chat: sharedChat,
                    user: userResponseFactory.createMockData({ 
                        overrides: { 
                            name: "Active Member",
                            handle: "active_member",
                        },
                    }),
                    created_at: new Date(baseTime - (DAYS_IN_3 * MILLISECONDS_PER_DAY)).toISOString(), // Joined 3 days ago
                    you: {
                        canDelete: false,
                        canUpdate: false,
                    },
                },
            }),

            // Recently joined member
            this.createMockData({
                overrides: {
                    id: "new_participant",
                    chat: sharedChat,
                    user: userResponseFactory.createMockData({ 
                        overrides: { 
                            name: "New Member",
                            handle: "new_member",
                        },
                    }),
                    created_at: new Date(baseTime - (HOURS_IN_2 * MILLISECONDS_PER_HOUR)).toISOString(), // Joined 2 hours ago
                    you: {
                        canDelete: false,
                        canUpdate: false,
                    },
                },
            }),

            // Bot participant
            this.createMockData({
                overrides: {
                    id: "bot_participant",
                    chat: sharedChat,
                    user: userResponseFactory.createMockData({ 
                        overrides: { 
                            isBot: true,
                            name: "AI Assistant",
                            handle: "ai_assistant",
                        },
                    }),
                    created_at: new Date(baseTime - (DAYS_IN_WEEK * MILLISECONDS_PER_DAY)).toISOString(), // Added when chat was created
                    you: {
                        canDelete: true, // Bots can be removed
                        canUpdate: true, // Bot settings can be updated
                    },
                },
            }),

            // Participant in different chat (for user's perspective)
            this.createMockData({
                overrides: {
                    id: "other_chat_participant",
                    chat: chatResponseFactory.createMockData({ 
                        overrides: { 
                            id: "other_chat_456",
                        },
                    }),
                    user: userResponseFactory.createMockData({ 
                        overrides: { 
                            name: "Current User",
                            handle: "current_user",
                        },
                    }),
                    created_at: new Date(baseTime - (DAYS_IN_3 * MILLISECONDS_PER_DAY)).toISOString(),
                    you: {
                        canDelete: false, // Can't delete own participation
                        canUpdate: false, // Can't update own participation
                    },
                },
            }),
        ];
    }

    /**
     * Create user already in chat error response
     */
    createUserAlreadyInChatErrorResponse(userId: string, chatId: string) {
        return this.createBusinessErrorResponse("already_member", {
            resource: "chat_participant",
            userId,
            chatId,
            message: "User is already a participant in this chat",
        });
    }

    /**
     * Create chat full error response
     */
    createChatFullErrorResponse(chatId: string, currentCount = MAX_PARTICIPANTS_PER_CHAT) {
        return this.createBusinessErrorResponse("chat_full", {
            resource: "chat_participant",
            chatId,
            limit: MAX_PARTICIPANTS_PER_CHAT,
            current: currentCount,
            message: `Chat has reached the maximum number of participants (${MAX_PARTICIPANTS_PER_CHAT})`,
        });
    }

    /**
     * Create cannot remove self error response
     */
    createCannotRemoveSelfErrorResponse() {
        return this.createBusinessErrorResponse("cannot_remove_self", {
            resource: "chat_participant",
            message: "You cannot remove yourself from the chat. Use leave chat instead.",
        });
    }

    /**
     * Create cannot remove creator error response
     */
    createCannotRemoveCreatorErrorResponse(creatorId: string) {
        return this.createBusinessErrorResponse("cannot_remove_creator", {
            resource: "chat_participant",
            creatorId,
            message: "Cannot remove the chat creator. Transfer ownership first.",
        });
    }

    /**
     * Create user not in chat error response
     */
    createUserNotInChatErrorResponse(userId: string, chatId: string) {
        return this.createBusinessErrorResponse("not_member", {
            resource: "chat_participant",
            userId,
            chatId,
            message: "User is not a participant in this chat",
        });
    }
}

/**
 * Pre-configured chat participant response scenarios
 */
export const chatParticipantResponseScenarios = {
    // Success scenarios
    findSuccess: (chatParticipant?: ChatParticipant) => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createSuccessResponse(
            chatParticipant || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: ChatParticipant, updates?: Partial<ChatParticipantUpdateInput>) => {
        const factory = new ChatParticipantResponseFactory();
        const participant = existing || factory.createMockData({ scenario: "complete" });
        const input: ChatParticipantUpdateInput = {
            id: participant.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(participant, input),
        );
    },

    listSuccess: (chatParticipants?: ChatParticipant[]) => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createPaginatedResponse(
            chatParticipants || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: chatParticipants?.length || DEFAULT_COUNT },
        );
    },

    chatParticipantsSuccess: (chatId?: string, count?: number) => {
        const factory = new ChatParticipantResponseFactory();
        const participants = factory.createParticipantsForChat(chatId || generatePK().toString(), count);
        return factory.createPaginatedResponse(
            participants,
            { page: 1, totalCount: participants.length },
        );
    },

    userParticipationsSuccess: (userId?: string, count?: number) => {
        const factory = new ChatParticipantResponseFactory();
        const participations = factory.createParticipantsForUser(userId || generatePK().toString(), count);
        return factory.createPaginatedResponse(
            participations,
            { page: 1, totalCount: participations.length },
        );
    },

    scenariosSuccess: () => {
        const factory = new ChatParticipantResponseFactory();
        const scenarios = factory.createParticipantScenarios();
        return factory.createPaginatedResponse(
            scenarios,
            { page: 1, totalCount: scenarios.length },
        );
    },

    creatorParticipantSuccess: () => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    user: userResponseFactory.createMockData({ 
                        overrides: { 
                            name: "Chat Creator",
                            handle: "chat_creator",
                        },
                    }),
                    you: {
                        canDelete: true,
                        canUpdate: true,
                    },
                },
            }),
        );
    },

    botParticipantSuccess: () => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    user: userResponseFactory.createMockData({ 
                        overrides: { 
                            isBot: true,
                            name: "AI Assistant",
                            handle: "ai_assistant",
                        },
                    }),
                },
            }),
        );
    },

    // Error scenarios
    notFoundError: (participantId?: string) => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createNotFoundErrorResponse(
            participantId || "non-existent-participant",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "update",
            ["chat:manage_participants"],
        );
    },

    userAlreadyInChatError: (userId?: string, chatId?: string) => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createUserAlreadyInChatErrorResponse(
            userId || generatePK().toString(),
            chatId || generatePK().toString(),
        );
    },

    chatFullError: (chatId?: string, currentCount?: number) => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createChatFullErrorResponse(chatId || generatePK().toString(), currentCount);
    },

    cannotRemoveSelfError: () => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createCannotRemoveSelfErrorResponse();
    },

    cannotRemoveCreatorError: (creatorId?: string) => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createCannotRemoveCreatorErrorResponse(creatorId || generatePK().toString());
    },

    userNotInChatError: (userId?: string, chatId?: string) => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createUserNotInChatErrorResponse(
            userId || generatePK().toString(),
            chatId || generatePK().toString(),
        );
    },

    updateValidationError: () => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createValidationErrorResponse({
            id: "Participant ID is required",
        });
    },

    // MSW handlers
    handlers: {
        success: () => new ChatParticipantResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new ChatParticipantResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new ChatParticipantResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const chatParticipantResponseFactory = new ChatParticipantResponseFactory();
