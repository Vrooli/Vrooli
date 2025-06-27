/* c8 ignore start */
/**
 * Chat API Response Fixtures
 * 
 * Comprehensive fixtures for chat management endpoints including
 * chat creation, participant management, and conversation features.
 */

import type {
    Chat,
    ChatCreateInput,
    ChatUpdateInput,
    ChatParticipant,
    ChatMessage,
    User,
    Team,
} from "../../../api/types.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";
import { generatePK } from "../../../id/index.js";
import { userResponseFactory } from "./userResponses.js";

/**
 * Chat API response factory
 * 
 * Handles chat creation, updates, participant management, and chat features.
 */
export class ChatResponseFactory extends BaseAPIResponseFactory<
    Chat,
    ChatCreateInput,
    ChatUpdateInput
> {
    protected readonly entityName = "chat";

    /**
     * Create mock chat data
     */
    createMockData(options?: MockDataOptions): Chat {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const chatId = options?.overrides?.id || generatePK().toString();
        const creator = userResponseFactory.createMockData();

        const baseChat: Chat = {
            __typename: "Chat",
            id: chatId,
            created_at: now,
            updated_at: now,
            openToAnyoneWithInvite: false,
            creator: { __typename: "User", id: creator.id } as User,
            invites: [],
            invitesCount: 0,
            participants: [],
            participantsCount: 1, // Creator is always a participant
            messages: [],
            messagesCount: 0,
            restrictedToRoles: [],
            team: null,
            translations: [],
            you: {
                canDelete: false,
                canInvite: false,
                canUpdate: false,
                canRead: true,
            },
        };

        if (scenario === "complete" || scenario === "edge-case") {
            const participants = this.createParticipants(chatId, creator, 3);
            const messages = this.createMessages(chatId, participants, 5);

            return {
                ...baseChat,
                openToAnyoneWithInvite: true,
                participants,
                participantsCount: participants.length,
                messages,
                messagesCount: messages.length,
                translations: [{
                    __typename: "ChatTranslation",
                    id: generatePK().toString(),
                    language: "en",
                    name: "Team Discussion",
                    description: "A place for team collaboration and updates",
                }],
                you: {
                    canDelete: true,
                    canInvite: true,
                    canUpdate: true,
                    canRead: true,
                },
                ...options?.overrides,
            };
        }

        return {
            ...baseChat,
            ...options?.overrides,
        };
    }

    /**
     * Create chat from input
     */
    createFromInput(input: ChatCreateInput): Chat {
        const now = new Date().toISOString();
        const chatId = generatePK().toString();
        const creator = userResponseFactory.createMockData();

        const chat: Chat = {
            __typename: "Chat",
            id: chatId,
            created_at: now,
            updated_at: now,
            openToAnyoneWithInvite: input.openToAnyoneWithInvite || false,
            creator: { __typename: "User", id: creator.id } as User,
            invites: [],
            invitesCount: 0,
            participants: [{
                __typename: "ChatParticipant",
                id: generatePK().toString(),
                created_at: now,
                updated_at: now,
                chat: { id: chatId } as Chat,
                user: creator,
            }],
            participantsCount: 1,
            messages: [],
            messagesCount: 0,
            restrictedToRoles: input.restrictedToRoles || [],
            team: input.teamConnect ? { __typename: "Team", id: input.teamConnect } as Team : null,
            translations: input.translationsCreate?.map(t => ({
                __typename: "ChatTranslation" as const,
                id: generatePK().toString(),
                language: t.language,
                name: t.name || null,
                description: t.description || null,
            })) || [],
            you: {
                canDelete: true,
                canInvite: true,
                canUpdate: true,
                canRead: true,
            },
        };

        // Add additional participants if specified
        if (input.participantsCreate) {
            input.participantsCreate.forEach(p => {
                if (p.userConnect) {
                    chat.participants.push({
                        __typename: "ChatParticipant",
                        id: generatePK().toString(),
                        created_at: now,
                        updated_at: now,
                        chat: { id: chatId } as Chat,
                        user: { __typename: "User", id: p.userConnect } as User,
                    });
                }
            });
            chat.participantsCount = chat.participants.length;
        }

        return chat;
    }

    /**
     * Update chat from input
     */
    updateFromInput(existing: Chat, input: ChatUpdateInput): Chat {
        const updates: Partial<Chat> = {
            updated_at: new Date().toISOString(),
        };

        if (input.openToAnyoneWithInvite !== undefined) {
            updates.openToAnyoneWithInvite = input.openToAnyoneWithInvite;
        }

        if (input.translationsUpdate) {
            updates.translations = existing.translations?.map(t => {
                const update = input.translationsUpdate?.find(u => u.id === t.id);
                return update ? { ...t, ...update } : t;
            });
        }

        if (input.restrictedToRolesUpdate) {
            updates.restrictedToRoles = input.restrictedToRolesUpdate;
        }

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: ChatCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.teamConnect && input.participantsCreate && input.participantsCreate.length > 0) {
            errors.participants = "Cannot specify participants for team chats";
        }

        if (input.restrictedToRoles && input.restrictedToRoles.length > 0 && !input.teamConnect) {
            errors.restrictedToRoles = "Role restrictions only apply to team chats";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: ChatUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.id) {
            errors.id = "Chat ID is required";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create chat participants
     */
    private createParticipants(chatId: string, creator: User, additionalCount = 2): ChatParticipant[] {
        const now = new Date().toISOString();
        const participants: ChatParticipant[] = [];

        // Add creator as first participant
        participants.push({
            __typename: "ChatParticipant",
            id: generatePK().toString(),
            created_at: now,
            updated_at: now,
            chat: { id: chatId } as Chat,
            user: creator,
        });

        // Add additional participants
        for (let i = 0; i < additionalCount; i++) {
            const user = userResponseFactory.createMockData();
            participants.push({
                __typename: "ChatParticipant",
                id: generatePK().toString(),
                created_at: now,
                updated_at: now,
                chat: { id: chatId } as Chat,
                user,
            });
        }

        return participants;
    }

    /**
     * Create chat messages
     */
    private createMessages(chatId: string, participants: ChatParticipant[], messageCount = 5): ChatMessage[] {
        const messages: ChatMessage[] = [];
        const baseTime = Date.now() - messageCount * 60000; // Start messages 5 minutes ago

        for (let i = 0; i < messageCount; i++) {
            const participant = participants[i % participants.length];
            const messageTime = new Date(baseTime + i * 60000).toISOString();

            messages.push({
                __typename: "ChatMessage",
                id: generatePK().toString(),
                created_at: messageTime,
                updated_at: messageTime,
                chat: { id: chatId } as Chat,
                user: participant.user,
                versionIndex: 0,
                parent: null,
                translations: [{
                    __typename: "ChatMessageTranslation",
                    id: generatePK().toString(),
                    language: "en",
                    text: `Message ${i + 1} from ${participant.user.name}`,
                }],
                reactionSummaries: [],
                reportsCount: 0,
                you: {
                    hasReported: false,
                    reaction: null,
                },
            });
        }

        return messages;
    }

    /**
     * Create private chat between two users
     */
    createPrivateChat(user1?: User, user2?: User): Chat {
        const participant1 = user1 || userResponseFactory.createMockData();
        const participant2 = user2 || userResponseFactory.createMockData();
        const chatId = generatePK().toString();
        const now = new Date().toISOString();

        return {
            __typename: "Chat",
            id: chatId,
            created_at: now,
            updated_at: now,
            openToAnyoneWithInvite: false,
            creator: { __typename: "User", id: participant1.id } as User,
            invites: [],
            invitesCount: 0,
            participants: [
                {
                    __typename: "ChatParticipant",
                    id: generatePK().toString(),
                    created_at: now,
                    updated_at: now,
                    chat: { id: chatId } as Chat,
                    user: participant1,
                },
                {
                    __typename: "ChatParticipant",
                    id: generatePK().toString(),
                    created_at: now,
                    updated_at: now,
                    chat: { id: chatId } as Chat,
                    user: participant2,
                },
            ],
            participantsCount: 2,
            messages: [],
            messagesCount: 0,
            restrictedToRoles: [],
            team: null,
            translations: [],
            you: {
                canDelete: true,
                canInvite: false,
                canUpdate: true,
                canRead: true,
            },
        };
    }

    /**
     * Create team chat
     */
    createTeamChat(teamId: string, memberCount = 5): Chat {
        const chatId = generatePK().toString();
        const now = new Date().toISOString();
        const participants = this.createParticipants(chatId, userResponseFactory.createMockData(), memberCount - 1);

        return {
            __typename: "Chat",
            id: chatId,
            created_at: now,
            updated_at: now,
            openToAnyoneWithInvite: false,
            creator: { __typename: "User", id: participants[0].user.id } as User,
            invites: [],
            invitesCount: 0,
            participants,
            participantsCount: participants.length,
            messages: [],
            messagesCount: 0,
            restrictedToRoles: ["Member", "Admin", "Owner"],
            team: { __typename: "Team", id: teamId } as Team,
            translations: [{
                __typename: "ChatTranslation",
                id: generatePK().toString(),
                language: "en",
                name: "Team General",
                description: "General discussion for all team members",
            }],
            you: {
                canDelete: false,
                canInvite: true,
                canUpdate: true,
                canRead: true,
            },
        };
    }
}

/**
 * Pre-configured chat response scenarios
 */
export const chatResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<ChatCreateInput>) => {
        const factory = new ChatResponseFactory();
        const defaultInput: ChatCreateInput = {
            openToAnyoneWithInvite: false,
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    findSuccess: (chat?: Chat) => {
        const factory = new ChatResponseFactory();
        return factory.createSuccessResponse(
            chat || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new ChatResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    privateChatSuccess: (user1?: User, user2?: User) => {
        const factory = new ChatResponseFactory();
        return factory.createSuccessResponse(
            factory.createPrivateChat(user1, user2),
        );
    },

    teamChatSuccess: (teamId?: string, memberCount?: number) => {
        const factory = new ChatResponseFactory();
        return factory.createSuccessResponse(
            factory.createTeamChat(teamId || generatePK().toString(), memberCount),
        );
    },

    updateSuccess: (existing?: Chat, updates?: Partial<ChatUpdateInput>) => {
        const factory = new ChatResponseFactory();
        const chat = existing || factory.createMockData({ scenario: "complete" });
        const input: ChatUpdateInput = {
            id: chat.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(chat, input),
        );
    },

    listSuccess: (chats?: Chat[]) => {
        const factory = new ChatResponseFactory();
        const DEFAULT_COUNT = 10;
        return factory.createPaginatedResponse(
            chats || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: chats?.length || DEFAULT_COUNT },
        );
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new ChatResponseFactory();
        return factory.createValidationErrorResponse({
            participants: "Cannot specify participants for team chats",
            restrictedToRoles: "Role restrictions only apply to team chats",
        });
    },

    notFoundError: (chatId?: string) => {
        const factory = new ChatResponseFactory();
        return factory.createNotFoundErrorResponse(
            chatId || "non-existent-chat",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new ChatResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "update",
            ["chat:update"],
        );
    },

    participantLimitError: () => {
        const factory = new ChatResponseFactory();
        const PARTICIPANT_LIMIT = 100;
        return factory.createBusinessErrorResponse("limit", {
            resource: "participants",
            limit: PARTICIPANT_LIMIT,
            current: PARTICIPANT_LIMIT,
            message: "Chat has reached the maximum number of participants",
        });
    },

    notParticipantError: () => {
        const factory = new ChatResponseFactory();
        return factory.createBusinessErrorResponse("state", {
            reason: "Not a participant",
            message: "You must be a participant to access this chat",
        });
    },

    // MSW handlers
    handlers: {
        success: () => new ChatResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate = 0.1) {
            return new ChatResponseFactory().createMSWHandlers({ errorRate });
        },
        withDelay: function createWithDelay(delay?: number) {
            const DEFAULT_DELAY_MS = 300;
            return new ChatResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const chatResponseFactory = new ChatResponseFactory();
