/**
 * Production-grade chat fixture factory
 * 
 * This factory provides type-safe chat fixtures using real functions from @vrooli/shared.
 * It eliminates `any` types and integrates with actual validation and shape transformation logic.
 */

import type { 
    Chat, 
    ChatCreateInput, 
    ChatUpdateInput,
    ChatTranslation,
    ChatParticipant,
    ChatMessage,
    Team,
    User,
    ChatInvite
} from "@vrooli/shared";
import { 
    chatValidation,
    shapeChat,
    ChatType
} from "@vrooli/shared";
import type { 
    FixtureFactory, 
    ValidationResult, 
    MSWHandlers
} from "../types.js";
import { rest } from "msw";

/**
 * UI-specific form data for chat creation
 */
export interface ChatFormData {
    name: string;
    description?: string;
    chatType?: ChatType;
    isPrivate?: boolean;
    participantIds?: string[];
    teamId?: string;
    welcomeMessage?: string;
}

/**
 * UI state for chat components
 */
export interface ChatUIState {
    isLoading: boolean;
    chat: Chat | null;
    error: string | null;
    messages: ChatMessage[];
    participants: ChatParticipant[];
    pendingInvites: ChatInvite[];
    isTyping: string[]; // User IDs of those typing
    unreadCount: number;
    canSendMessage: boolean;
    canInviteMembers: boolean;
}

export type ChatScenario = 
    | "minimal" 
    | "complete" 
    | "invalid" 
    | "directMessage"
    | "groupChat"
    | "teamChat"
    | "privateChat"
    | "largeChat";

/**
 * Type-safe chat fixture factory that uses real @vrooli/shared functions
 */
export class ChatFixtureFactory implements FixtureFactory<
    ChatFormData,
    ChatCreateInput,
    ChatUpdateInput,
    Chat
> {
    readonly objectType = "chat";

    /**
     * Generate a unique ID for testing
     */
    private generateId(): string {
        return `test_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }

    /**
     * Create form data for different test scenarios
     */
    createFormData(scenario: ChatScenario = "minimal"): ChatFormData {
        switch (scenario) {
            case "minimal":
                return {
                    name: "Test Chat",
                    chatType: ChatType.Direct,
                    isPrivate: false,
                    participantIds: [this.generateId()]
                };

            case "complete":
                return {
                    name: "Complete Test Chat",
                    description: "A comprehensive chat with all fields filled",
                    chatType: ChatType.Group,
                    isPrivate: false,
                    participantIds: [this.generateId(), this.generateId(), this.generateId()],
                    welcomeMessage: "Welcome to our test chat! Feel free to ask questions."
                };

            case "invalid":
                return {
                    name: "", // Empty name
                    // @ts-expect-error - Testing invalid chat type
                    chatType: "InvalidType",
                    participantIds: [] // No participants
                };

            case "directMessage":
                return {
                    name: "Direct Message",
                    chatType: ChatType.Direct,
                    isPrivate: true,
                    participantIds: [this.generateId()]
                };

            case "groupChat":
                return {
                    name: "Group Chat",
                    description: "A group chat for collaboration",
                    chatType: ChatType.Group,
                    isPrivate: false,
                    participantIds: [this.generateId(), this.generateId(), this.generateId(), this.generateId()],
                    welcomeMessage: "Welcome to the group!"
                };

            case "teamChat":
                return {
                    name: "Team Discussion",
                    description: "Official team communication channel",
                    chatType: ChatType.Group,
                    isPrivate: false,
                    teamId: this.generateId(),
                    participantIds: []
                };

            case "privateChat":
                return {
                    name: "Private Discussion",
                    description: "Invite-only private chat",
                    chatType: ChatType.Group,
                    isPrivate: true,
                    participantIds: [this.generateId(), this.generateId()]
                };

            case "largeChat":
                return {
                    name: "Community Chat",
                    description: "Large community discussion",
                    chatType: ChatType.Group,
                    isPrivate: false,
                    participantIds: Array.from({ length: 20 }, () => this.generateId())
                };

            default:
                throw new Error(`Unknown chat scenario: ${scenario}`);
        }
    }

    /**
     * Transform form data to API create input using real shape function
     */
    transformToAPIInput(formData: ChatFormData): ChatCreateInput {
        // Create the chat shape that matches the expected API structure
        const chatShape = {
            __typename: "Chat" as const,
            id: this.generateId(),
            name: formData.name,
            chatType: formData.chatType || ChatType.Direct,
            isPrivate: formData.isPrivate || false,
            team: formData.teamId ? { id: formData.teamId } : null,
            translations: formData.description ? [{
                __typename: "ChatTranslation" as const,
                id: this.generateId(),
                language: "en",
                description: formData.description
            }] : null,
            participantsConnect: formData.participantIds || [],
            welcomeMessage: formData.welcomeMessage
        };

        // Use real shape function from @vrooli/shared
        return shapeChat.create(chatShape);
    }

    /**
     * Create API update input
     */
    createUpdateInput(id: string, updates: Partial<ChatFormData>): ChatUpdateInput {
        const updateInput: ChatUpdateInput = { id };

        if (updates.name) updateInput.name = updates.name;
        if (updates.isPrivate !== undefined) updateInput.isPrivate = updates.isPrivate;

        // Handle translations for description
        if (updates.description) {
            updateInput.translationsUpdate = [{
                id: this.generateId(),
                language: "en",
                description: updates.description
            }];
        }

        // Handle participant updates
        if (updates.participantIds) {
            updateInput.participantsConnect = updates.participantIds;
        }

        if (updates.welcomeMessage !== undefined) {
            updateInput.welcomeMessage = updates.welcomeMessage;
        }

        return updateInput;
    }

    /**
     * Create mock chat response with realistic data
     */
    createMockResponse(overrides?: Partial<Chat>): Chat {
        const now = new Date().toISOString();
        const chatId = this.generateId();
        const creatorId = this.generateId();
        
        const defaultChat: Chat = {
            __typename: "Chat",
            id: chatId,
            name: "Test Chat",
            chatType: ChatType.Direct,
            createdAt: now,
            updatedAt: now,
            isPrivate: false,
            creator: {
                __typename: "User",
                id: creatorId,
                handle: "chatcreator",
                name: "Chat Creator",
                email: "creator@example.com",
                emailVerified: true,
                createdAt: now,
                updatedAt: now,
                isBot: false,
                isPrivate: false,
                profileImage: null,
                bannerImage: null,
                premium: false,
                premiumExpiration: null,
                translations: [],
                translationsCount: 0,
                you: {
                    __typename: "UserYou",
                    isBlocked: false,
                    isBlockedByYou: false,
                    canDelete: false,
                    canReport: false,
                    canUpdate: false,
                    isBookmarked: false,
                    isReacted: false,
                    reactionSummary: {
                        __typename: "ReactionSummary",
                        emotion: null,
                        count: 0
                    }
                }
            },
            team: null,
            teamId: null,
            translations: [{
                __typename: "ChatTranslation",
                id: this.generateId(),
                language: "en",
                description: "A test chat for development"
            }],
            translationsCount: 1,
            participants: [{
                __typename: "ChatParticipant",
                id: this.generateId(),
                created_at: now,
                chatId,
                userId: creatorId,
                user: {
                    __typename: "User",
                    id: creatorId,
                    handle: "chatcreator",
                    name: "Chat Creator",
                    email: "creator@example.com",
                    emailVerified: true,
                    createdAt: now,
                    updatedAt: now,
                    isBot: false,
                    isPrivate: false,
                    profileImage: null,
                    bannerImage: null,
                    premium: false,
                    premiumExpiration: null,
                    translations: [],
                    translationsCount: 0,
                    you: {
                        __typename: "UserYou",
                        isBlocked: false,
                        isBlockedByYou: false,
                        canDelete: false,
                        canReport: false,
                        canUpdate: false,
                        isBookmarked: false,
                        isReacted: false,
                        reactionSummary: {
                            __typename: "ReactionSummary",
                            emotion: null,
                            count: 0
                        }
                    }
                }
            }],
            participantsCount: 1,
            messages: [],
            messagesCount: 0,
            invites: [],
            invitesCount: 0,
            restrictedToRoles: [],
            labels: [],
            labelsCount: 0,
            openToAnyoneWithInvite: true,
            you: {
                __typename: "ChatYou",
                canDelete: true,
                canInvite: true,
                canUpdate: true,
                canRead: true,
                hasUnread: false,
                isBookmarked: false,
                isMuted: false,
                isPinned: false
            }
        };

        return {
            ...defaultChat,
            ...overrides
        };
    }

    /**
     * Validate form data using real validation from @vrooli/shared
     */
    async validateFormData(formData: ChatFormData): Promise<ValidationResult> {
        try {
            // Transform to API input format for validation
            const apiInput = this.transformToAPIInput(formData);
            
            // Use real validation schema from @vrooli/shared
            await chatValidation.create.validate(apiInput, { abortEarly: false });
            
            // Additional validation for chat-specific rules
            if (formData.chatType === ChatType.Direct && formData.participantIds && formData.participantIds.length > 1) {
                return {
                    isValid: false,
                    errors: ["Direct messages can only have one other participant"],
                    fieldErrors: {
                        participantIds: ["Direct messages can only have one other participant"]
                    }
                };
            }
            
            return { isValid: true };
        } catch (error: any) {
            return {
                isValid: false,
                errors: error.errors || [error.message],
                fieldErrors: error.inner?.reduce((acc: any, err: any) => {
                    if (err.path) {
                        acc[err.path] = acc[err.path] || [];
                        acc[err.path].push(err.message);
                    }
                    return acc;
                }, {})
            };
        }
    }

    /**
     * Create MSW handlers for different scenarios
     */
    createMSWHandlers(): MSWHandlers {
        const baseUrl = process.env.VITE_SERVER_URL || 'http://localhost:3000';

        return {
            success: [
                // Create chat
                rest.post(`${baseUrl}/api/chat`, async (req, res, ctx) => {
                    const body = await req.json();
                    
                    // Validate the request body
                    const validation = await this.validateFormData(body);
                    if (!validation.isValid) {
                        return res(
                            ctx.status(400),
                            ctx.json({ 
                                errors: validation.errors,
                                fieldErrors: validation.fieldErrors 
                            })
                        );
                    }

                    // Return successful response
                    const mockChat = this.createMockResponse({
                        name: body.name,
                        chatType: body.chatType,
                        isPrivate: body.isPrivate
                    });

                    return res(
                        ctx.status(201),
                        ctx.json(mockChat)
                    );
                }),

                // Update chat
                rest.put(`${baseUrl}/api/chat/:id`, async (req, res, ctx) => {
                    const { id } = req.params;
                    const body = await req.json();

                    const mockChat = this.createMockResponse({ 
                        id: id as string,
                        ...body,
                        updatedAt: new Date().toISOString()
                    });

                    return res(
                        ctx.status(200),
                        ctx.json(mockChat)
                    );
                }),

                // Get chat
                rest.get(`${baseUrl}/api/chat/:id`, (req, res, ctx) => {
                    const { id } = req.params;
                    const mockChat = this.createMockResponse({ id: id as string });
                    
                    return res(
                        ctx.status(200),
                        ctx.json(mockChat)
                    );
                }),

                // Get user's chats
                rest.get(`${baseUrl}/api/chats`, (req, res, ctx) => {
                    const chats = [
                        this.createMockResponse(),
                        this.createMockResponse({ 
                            name: "Team Chat",
                            chatType: ChatType.Group 
                        })
                    ];
                    
                    return res(
                        ctx.status(200),
                        ctx.json({ data: chats })
                    );
                }),

                // Delete chat
                rest.delete(`${baseUrl}/api/chat/:id`, (req, res, ctx) => {
                    return res(
                        ctx.status(204)
                    );
                })
            ],

            error: [
                rest.post(`${baseUrl}/api/chat`, (req, res, ctx) => {
                    return res(
                        ctx.status(400),
                        ctx.json({ 
                            message: 'Invalid chat configuration',
                            code: 'INVALID_CHAT' 
                        })
                    );
                }),

                rest.put(`${baseUrl}/api/chat/:id`, (req, res, ctx) => {
                    return res(
                        ctx.status(403),
                        ctx.json({ 
                            message: 'You do not have permission to update this chat',
                            code: 'PERMISSION_DENIED' 
                        })
                    );
                }),

                rest.get(`${baseUrl}/api/chat/:id`, (req, res, ctx) => {
                    return res(
                        ctx.status(404),
                        ctx.json({ 
                            message: 'Chat not found',
                            code: 'CHAT_NOT_FOUND' 
                        })
                    );
                })
            ],

            loading: [
                rest.post(`${baseUrl}/api/chat`, (req, res, ctx) => {
                    return res(
                        ctx.delay(2000), // 2 second delay
                        ctx.status(201),
                        ctx.json(this.createMockResponse())
                    );
                })
            ],

            networkError: [
                rest.post(`${baseUrl}/api/chat`, (req, res, ctx) => {
                    return res.networkError('Network connection failed');
                })
            ]
        };
    }

    /**
     * Create UI state fixtures for different scenarios
     */
    createUIState(
        state: "loading" | "error" | "success" | "empty" | "withMessages" | "typing" = "empty", 
        data?: any
    ): ChatUIState {
        switch (state) {
            case "loading":
                return {
                    isLoading: true,
                    chat: null,
                    error: null,
                    messages: [],
                    participants: [],
                    pendingInvites: [],
                    isTyping: [],
                    unreadCount: 0,
                    canSendMessage: false,
                    canInviteMembers: false
                };

            case "error":
                return {
                    isLoading: false,
                    chat: null,
                    error: data?.message || "Failed to load chat",
                    messages: [],
                    participants: [],
                    pendingInvites: [],
                    isTyping: [],
                    unreadCount: 0,
                    canSendMessage: false,
                    canInviteMembers: false
                };

            case "success":
            case "withMessages":
                const chat = data?.chat || this.createMockResponse();
                return {
                    isLoading: false,
                    chat,
                    error: null,
                    messages: data?.messages || [],
                    participants: chat.participants || [],
                    pendingInvites: [],
                    isTyping: [],
                    unreadCount: 0,
                    canSendMessage: true,
                    canInviteMembers: chat.chatType === ChatType.Group
                };

            case "typing":
                const typingChat = data?.chat || this.createMockResponse();
                return {
                    isLoading: false,
                    chat: typingChat,
                    error: null,
                    messages: data?.messages || [],
                    participants: typingChat.participants || [],
                    pendingInvites: [],
                    isTyping: data?.typingUsers || [this.generateId()],
                    unreadCount: 0,
                    canSendMessage: true,
                    canInviteMembers: typingChat.chatType === ChatType.Group
                };

            case "empty":
            default:
                return {
                    isLoading: false,
                    chat: null,
                    error: null,
                    messages: [],
                    participants: [],
                    pendingInvites: [],
                    isTyping: [],
                    unreadCount: 0,
                    canSendMessage: false,
                    canInviteMembers: false
                };
        }
    }

    /**
     * Create a chat with multiple participants
     */
    createWithParticipants(participantCount: number = 3): Chat {
        const participants: ChatParticipant[] = Array.from({ length: participantCount }, (_, i) => ({
            __typename: "ChatParticipant",
            id: this.generateId(),
            created_at: new Date().toISOString(),
            chatId: this.generateId(),
            userId: this.generateId(),
            user: {
                __typename: "User",
                id: this.generateId(),
                handle: `user${i + 1}`,
                name: `User ${i + 1}`,
                email: `user${i + 1}@example.com`,
                emailVerified: true,
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
                isBot: false,
                isPrivate: false,
                profileImage: null,
                bannerImage: null,
                premium: false,
                premiumExpiration: null,
                translations: [],
                translationsCount: 0,
                you: {
                    __typename: "UserYou",
                    isBlocked: false,
                    isBlockedByYou: false,
                    canDelete: false,
                    canReport: false,
                    canUpdate: false,
                    isBookmarked: false,
                    isReacted: false,
                    reactionSummary: {
                        __typename: "ReactionSummary",
                        emotion: null,
                        count: 0
                    }
                }
            }
        }));

        return this.createMockResponse({
            chatType: ChatType.Group,
            participants,
            participantsCount: participantCount
        });
    }

    /**
     * Create a team chat
     */
    createTeamChat(teamId: string, teamName: string = "Test Team"): Chat {
        return this.createMockResponse({
            name: `${teamName} Chat`,
            chatType: ChatType.Group,
            team: {
                __typename: "Team",
                id: teamId,
                handle: `team_${teamId}`,
                name: teamName,
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
                isPrivate: false,
                profileImage: null,
                bannerImage: null,
                permissions: JSON.stringify({}),
                tags: [],
                tagsCount: 0,
                translations: [],
                translationsCount: 0,
                members: [],
                membersCount: 0,
                projects: [],
                projectsCount: 0,
                isOpenToNewMembers: true,
                routines: [],
                routinesCount: 0,
                standards: [],
                standardsCount: 0,
                resourceLists: [],
                resourceListsCount: 0,
                you: {
                    __typename: "TeamYou",
                    canAddMembers: false,
                    canDelete: false,
                    canRemoveMembers: false,
                    canUpdate: false,
                    canReport: false,
                    isBookmarked: false,
                    isReacted: false,
                    reaction: null,
                    yourMemberRole: null
                }
            },
            teamId
        });
    }

    /**
     * Create test cases for various scenarios
     */
    createTestCases() {
        return [
            {
                name: "Valid chat creation",
                formData: this.createFormData("minimal"),
                shouldSucceed: true
            },
            {
                name: "Complete chat profile",
                formData: this.createFormData("complete"),
                shouldSucceed: true
            },
            {
                name: "Empty name",
                formData: { ...this.createFormData("minimal"), name: "" },
                shouldSucceed: false,
                expectedError: "name is a required field"
            },
            {
                name: "No participants",
                formData: { ...this.createFormData("minimal"), participantIds: [] },
                shouldSucceed: false,
                expectedError: "At least one participant required"
            },
            {
                name: "Direct message",
                formData: this.createFormData("directMessage"),
                shouldSucceed: true
            },
            {
                name: "Group chat",
                formData: this.createFormData("groupChat"),
                shouldSucceed: true
            }
        ];
    }
}

/**
 * Default factory instance for easy importing
 */
export const chatFixtures = new ChatFixtureFactory();

/**
 * Specific test scenarios for common use cases
 */
export const chatTestScenarios = {
    // Basic scenarios
    minimalChat: () => chatFixtures.createFormData("minimal"),
    completeChat: () => chatFixtures.createFormData("complete"),
    invalidChat: () => chatFixtures.createFormData("invalid"),
    
    // Chat type scenarios
    directMessage: () => chatFixtures.createFormData("directMessage"),
    groupChat: () => chatFixtures.createFormData("groupChat"),
    teamChat: () => chatFixtures.createFormData("teamChat"),
    privateChat: () => chatFixtures.createFormData("privateChat"),
    largeChat: () => chatFixtures.createFormData("largeChat"),
    
    // Mock responses
    basicChatResponse: () => chatFixtures.createMockResponse(),
    groupChatResponse: () => chatFixtures.createWithParticipants(5),
    teamChatResponse: () => chatFixtures.createTeamChat("team123"),
    
    // UI state scenarios
    loadingState: () => chatFixtures.createUIState("loading"),
    errorState: (message?: string) => chatFixtures.createUIState("error", { message }),
    successState: (chat?: Chat) => chatFixtures.createUIState("success", { chat }),
    withMessagesState: (chat?: Chat, messages?: ChatMessage[]) => chatFixtures.createUIState("withMessages", { chat, messages }),
    typingState: (chat?: Chat, typingUsers?: string[]) => chatFixtures.createUIState("typing", { chat, typingUsers }),
    emptyState: () => chatFixtures.createUIState("empty"),
    
    // Test data sets
    allTestCases: () => chatFixtures.createTestCases(),
    
    // MSW handlers
    successHandlers: () => chatFixtures.createMSWHandlers().success,
    errorHandlers: () => chatFixtures.createMSWHandlers().error,
    loadingHandlers: () => chatFixtures.createMSWHandlers().loading,
    networkErrorHandlers: () => chatFixtures.createMSWHandlers().networkError
};