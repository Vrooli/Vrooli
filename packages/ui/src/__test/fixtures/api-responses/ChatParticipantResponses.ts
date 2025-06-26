/**
 * ChatParticipant API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for ChatParticipant endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 * 
 * ChatParticipants represent the relationship between users and chat conversations,
 * including roles, permissions, and participation metadata.
 */

import { http, type RestHandler } from "msw";
import type { 
    ChatParticipant, 
    ChatParticipantUpdateInput,
    ChatParticipantSearchInput,
    Chat,
    User,
} from "@vrooli/shared";
import { 
    chatParticipantValidation 
,
    ChatParticipantSortBy} from "@vrooli/shared";

/**
 * Standard API response wrapper
 */
export interface APIResponse<T> {
    data: T;
    meta: {
        timestamp: string;
        requestId: string;
        version: string;
        links?: {
            self?: string;
            related?: Record<string, string>;
        };
    };
}

/**
 * API error response structure
 */
export interface APIErrorResponse {
    error: {
        code: string;
        message: string;
        details?: Record<string, any>;
        timestamp: string;
        requestId: string;
        path: string;
    };
}

/**
 * Paginated response structure
 */
export interface PaginatedAPIResponse<T> extends APIResponse<T[]> {
    pagination: {
        page: number;
        pageSize: number;
        totalCount: number;
        totalPages: number;
        hasNextPage: boolean;
        hasPreviousPage: boolean;
    };
}

/**
 * ChatParticipant API response factory
 */
export class ChatParticipantResponseFactory {
    private readonly baseUrl: string;
    
    constructor(baseUrl: string = process.env.VITE_SERVER_URL || "http://localhost:5329") {
        this.baseUrl = baseUrl;
    }
    
    /**
     * Generate unique request ID
     */
    private generateRequestId(): string {
        return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Generate unique resource ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create successful ChatParticipant response
     */
    createSuccessResponse(participant: ChatParticipant): APIResponse<ChatParticipant> {
        return {
            data: participant,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/chatParticipant/${participant.id}`,
                    related: {
                        chat: `${this.baseUrl}/api/chat/${participant.chat.id}`,
                        user: `${this.baseUrl}/api/user/${participant.user.id}`,
                        participants: `${this.baseUrl}/api/chatParticipants?chatId=${participant.chat.id}`,
                    },
                },
            },
        };
    }
    
    /**
     * Create ChatParticipant list response
     */
    createChatParticipantListResponse(participants: ChatParticipant[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<ChatParticipant> {
        const paginationData = pagination || {
            page: 1,
            pageSize: participants.length,
            totalCount: participants.length,
        };
        
        return {
            data: participants,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/chatParticipants?page=${paginationData.page}&limit=${paginationData.pageSize}`,
                },
            },
            pagination: {
                ...paginationData,
                totalPages: Math.ceil(paginationData.totalCount / paginationData.pageSize),
                hasNextPage: paginationData.page * paginationData.pageSize < paginationData.totalCount,
                hasPreviousPage: paginationData.page > 1,
            },
        };
    }
    
    /**
     * Create validation error response
     */
    createValidationErrorResponse(fieldErrors: Record<string, string>): APIErrorResponse {
        return {
            error: {
                code: "VALIDATION_ERROR",
                message: "The request contains invalid data",
                details: {
                    fieldErrors,
                    invalidFields: Object.keys(fieldErrors),
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/chatParticipant",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(participantId: string): APIErrorResponse {
        return {
            error: {
                code: "CHAT_PARTICIPANT_NOT_FOUND",
                message: `Chat participant with ID '${participantId}' was not found`,
                details: {
                    participantId,
                    searchCriteria: { id: participantId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/chatParticipant/${participantId}`,
            },
        };
    }
    
    /**
     * Create permission error response
     */
    createPermissionErrorResponse(operation: string): APIErrorResponse {
        return {
            error: {
                code: "PERMISSION_DENIED",
                message: `You do not have permission to ${operation} this chat participant`,
                details: {
                    operation,
                    requiredPermissions: ["chat:participate"],
                    userPermissions: ["chat:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/chatParticipant",
            },
        };
    }
    
    /**
     * Create chat not found error response
     */
    createChatNotFoundErrorResponse(chatId: string): APIErrorResponse {
        return {
            error: {
                code: "CHAT_NOT_FOUND",
                message: `Chat with ID '${chatId}' was not found`,
                details: {
                    chatId,
                    searchCriteria: { chatId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/chatParticipants",
            },
        };
    }
    
    /**
     * Create user already participant error response
     */
    createAlreadyParticipantErrorResponse(userId: string, chatId: string): APIErrorResponse {
        return {
            error: {
                code: "USER_ALREADY_PARTICIPANT",
                message: "User is already a participant in this chat",
                details: {
                    userId,
                    chatId,
                    conflictType: "duplicate_participation",
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/chatParticipant",
            },
        };
    }
    
    /**
     * Create network error response
     */
    createNetworkErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: "NETWORK_ERROR",
                message: "Network request failed",
                details: {
                    reason: "Connection timeout",
                    retryable: true,
                    retryAfter: 5000,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/chatParticipant",
            },
        };
    }
    
    /**
     * Create server error response
     */
    createServerErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: "INTERNAL_SERVER_ERROR",
                message: "An unexpected server error occurred",
                details: {
                    errorId: `ERR_${Date.now()}`,
                    reportable: true,
                    retryable: true,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/chatParticipant",
            },
        };
    }
    
    /**
     * Create mock user data
     */
    createMockUser(overrides?: Partial<User>): User {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultUser: User = {
            __typename: "User",
            id,
            createdAt: now,
            updatedAt: now,
            handle: `user_${id.slice(-6)}`,
            name: `Test User ${id.slice(-3)}`,
            isBot: false,
            isPrivate: false,
            premium: false,
            premiumExpiration: null,
            profileImage: null,
            bannerImage: null,
            roles: [],
            wallets: [],
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
                    count: 0,
                },
            },
        } as User;
        
        return {
            ...defaultUser,
            ...overrides,
        };
    }
    
    /**
     * Create mock chat data
     */
    createMockChat(overrides?: Partial<Chat>): Chat {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultChat: Chat = {
            __typename: "Chat",
            id,
            createdAt: now,
            updatedAt: now,
            publicId: `chat_${id.slice(-8)}`,
            openToAnyoneWithInvite: false,
            invites: [],
            invitesCount: 0,
            messages: [],
            participants: [],
            participantsCount: 0,
            team: null,
            translations: [],
            translationsCount: 0,
            you: {
                __typename: "ChatYou",
                canDelete: false,
                canUpdate: false,
                canInvite: false,
                canManageParticipants: false,
            },
        } as Chat;
        
        return {
            ...defaultChat,
            ...overrides,
        };
    }
    
    /**
     * Create mock ChatParticipant data
     */
    createMockChatParticipant(overrides?: Partial<ChatParticipant>): ChatParticipant {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultParticipant: ChatParticipant = {
            __typename: "ChatParticipant",
            id,
            createdAt: now,
            updatedAt: now,
            user: this.createMockUser(),
            chat: this.createMockChat(),
        };
        
        return {
            ...defaultParticipant,
            ...overrides,
        };
    }
    
    /**
     * Create ChatParticipant from update input
     */
    createChatParticipantFromUpdateInput(input: ChatParticipantUpdateInput, existing?: Partial<ChatParticipant>): ChatParticipant {
        const base = existing || this.createMockChatParticipant();
        
        return {
            ...base,
            id: input.id,
            updatedAt: new Date().toISOString(),
        } as ChatParticipant;
    }
    
    /**
     * Create multiple participants for a chat
     */
    createChatParticipants(chatId: string, userCount = 5): ChatParticipant[] {
        const chat = this.createMockChat({ id: chatId });
        const participants: ChatParticipant[] = [];
        
        for (let i = 0; i < userCount; i++) {
            const user = this.createMockUser({
                id: `user_${chatId}_${i}`,
                handle: `participant_${i}`,
                name: `Participant ${i + 1}`,
                isBot: i === 0, // First user is a bot
                premium: i % 3 === 0, // Every third user is premium
            });
            
            participants.push(this.createMockChatParticipant({
                id: `participant_${chatId}_${i}`,
                user,
                chat,
                createdAt: new Date(Date.now() - (userCount - i) * 86400000).toISOString(), // Stagger join times
            }));
        }
        
        return participants;
    }
    
    /**
     * Create participants for different chat scenarios
     */
    createParticipantsForScenarios(): { [scenario: string]: ChatParticipant[] } {
        return {
            // Active public chat with diverse participants
            publicChat: this.createChatParticipants("public_chat_123", 8),
            
            // Small private team chat
            teamChat: this.createChatParticipants("team_chat_456", 3),
            
            // Large community chat
            communityChat: this.createChatParticipants("community_chat_789", 15),
            
            // One-on-one chat
            directMessage: this.createChatParticipants("dm_chat_012", 2),
            
            // Support chat with bot
            supportChat: this.createChatParticipants("support_chat_345", 4),
        };
    }
    
    /**
     * Validate ChatParticipant update input
     */
    async validateUpdateInput(input: ChatParticipantUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await chatParticipantValidation.update.validate(input);
            return { valid: true };
        } catch (error: any) {
            const fieldErrors: Record<string, string> = {};
            
            if (error.inner) {
                error.inner.forEach((err: any) => {
                    if (err.path) {
                        fieldErrors[err.path] = err.message;
                    }
                });
            } else if (error.message) {
                fieldErrors.general = error.message;
            }
            
            return {
                valid: false,
                errors: fieldErrors,
            };
        }
    }
}

/**
 * MSW handlers factory for ChatParticipant endpoints
 */
export class ChatParticipantMSWHandlers {
    private responseFactory: ChatParticipantResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new ChatParticipantResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all ChatParticipant endpoints
     */
    createSuccessHandlers(): RestHandler[] {
        return [
            // Get ChatParticipant by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/chatParticipant/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const participant = this.responseFactory.createMockChatParticipant({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(participant);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Update ChatParticipant
            http.put(`${this.responseFactory["baseUrl"]}/api/chatParticipant/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as ChatParticipantUpdateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateUpdateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                // Update participant
                const participant = this.responseFactory.createChatParticipantFromUpdateInput(body);
                const response = this.responseFactory.createSuccessResponse(participant);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // List ChatParticipants
            http.get(`${this.responseFactory["baseUrl"]}/api/chatParticipants`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const chatId = url.searchParams.get("chatId");
                const userId = url.searchParams.get("userId");
                const sortBy = url.searchParams.get("sortBy") as ChatParticipantSortBy;
                
                let participants: ChatParticipant[] = [];
                
                if (chatId) {
                    // Get participants for specific chat
                    participants = this.responseFactory.createChatParticipants(chatId, 8);
                } else if (userId) {
                    // Get chats where user is participant
                    const scenarios = this.responseFactory.createParticipantsForScenarios();
                    participants = Object.values(scenarios).flat().filter(p => p.user.id === userId);
                } else {
                    // Get all participants (for admin/moderation)
                    const scenarios = this.responseFactory.createParticipantsForScenarios();
                    participants = Object.values(scenarios).flat();
                }
                
                // Apply sorting
                if (sortBy) {
                    participants = this.applySorting(participants, sortBy);
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedParticipants = participants.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createChatParticipantListResponse(
                    paginatedParticipants,
                    {
                        page,
                        pageSize: limit,
                        totalCount: participants.length,
                    },
                );
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
        ];
    }
    
    /**
     * Apply sorting to participants list
     */
    private applySorting(participants: ChatParticipant[], sortBy: ChatParticipantSortBy): ChatParticipant[] {
        const sorted = [...participants];
        
        switch (sortBy) {
            case ChatParticipantSortBy.DateCreatedAsc:
                return sorted.sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime());
            case ChatParticipantSortBy.DateCreatedDesc:
                return sorted.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
            case ChatParticipantSortBy.DateUpdatedAsc:
                return sorted.sort((a, b) => new Date(a.updatedAt).getTime() - new Date(b.updatedAt).getTime());
            case ChatParticipantSortBy.DateUpdatedDesc:
                return sorted.sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime());
            case ChatParticipantSortBy.UserNameAsc:
                return sorted.sort((a, b) => a.user.name.localeCompare(b.user.name));
            case ChatParticipantSortBy.UserNameDesc:
                return sorted.sort((a, b) => b.user.name.localeCompare(a.user.name));
            default:
                return sorted;
        }
    }
    
    /**
     * Create error handlers for testing error scenarios
     */
    createErrorHandlers(): RestHandler[] {
        return [
            // Validation error
            http.put(`${this.responseFactory["baseUrl"]}/api/chatParticipant/:id`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        id: "Participant ID is required and must be a valid format",
                    })),
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/chatParticipant/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string)),
                );
            }),
            
            // Permission error
            http.put(`${this.responseFactory["baseUrl"]}/api/chatParticipant/:id`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse("update")),
                );
            }),
            
            // Chat not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/chatParticipants`, (req, res, ctx) => {
                const url = new URL(req.url);
                const chatId = url.searchParams.get("chatId");
                
                if (chatId === "non-existent-chat") {
                    return res(
                        ctx.status(404),
                        ctx.json(this.responseFactory.createChatNotFoundErrorResponse(chatId)),
                    );
                }
                
                return res(
                    ctx.status(500),
                    ctx.json(this.responseFactory.createServerErrorResponse()),
                );
            }),
            
            // Already participant error
            http.post(`${this.responseFactory["baseUrl"]}/api/chatParticipant`, (req, res, ctx) => {
                return res(
                    ctx.status(409),
                    ctx.json(this.responseFactory.createAlreadyParticipantErrorResponse("user_123", "chat_456")),
                );
            }),
            
            // Server error
            http.get(`${this.responseFactory["baseUrl"]}/api/chatParticipants`, (req, res, ctx) => {
                return res(
                    ctx.status(500),
                    ctx.json(this.responseFactory.createServerErrorResponse()),
                );
            }),
        ];
    }
    
    /**
     * Create loading simulation handlers
     */
    createLoadingHandlers(delay = 2000): RestHandler[] {
        return [
            http.get(`${this.responseFactory["baseUrl"]}/api/chatParticipants`, (req, res, ctx) => {
                const url = new URL(req.url);
                const chatId = url.searchParams.get("chatId") || "default_chat";
                
                const participants = this.responseFactory.createChatParticipants(chatId, 5);
                const response = this.responseFactory.createChatParticipantListResponse(participants);
                
                return res(
                    ctx.delay(delay),
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            http.put(`${this.responseFactory["baseUrl"]}/api/chatParticipant/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as ChatParticipantUpdateInput;
                
                const participant = this.responseFactory.createChatParticipantFromUpdateInput(body);
                const response = this.responseFactory.createSuccessResponse(participant);
                
                return res(
                    ctx.delay(delay),
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
        ];
    }
    
    /**
     * Create network error handlers
     */
    createNetworkErrorHandlers(): RestHandler[] {
        return [
            http.get(`${this.responseFactory["baseUrl"]}/api/chatParticipant/:id`, (req, res, ctx) => {
                return res.networkError("Network connection failed");
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/chatParticipants`, (req, res, ctx) => {
                return res.networkError("Connection timeout");
            }),
            
            http.put(`${this.responseFactory["baseUrl"]}/api/chatParticipant/:id`, (req, res, ctx) => {
                return res.networkError("Network error during update");
            }),
        ];
    }
    
    /**
     * Create custom handler with specific configuration
     */
    createCustomHandler(config: {
        endpoint: string;
        method: "GET" | "POST" | "PUT" | "DELETE";
        status: number;
        response: any;
        delay?: number;
    }): RestHandler {
        const { endpoint, method, status, response, delay } = config;
        const fullEndpoint = `${this.responseFactory["baseUrl"]}${endpoint}`;
        
        return rest[method.toLowerCase() as keyof typeof rest](fullEndpoint, (req, res, ctx) => {
            const responseCtx = [ctx.status(status), ctx.json(response)];
            
            if (delay) {
                responseCtx.unshift(ctx.delay(delay));
            }
            
            return res(...responseCtx);
        });
    }
}

/**
 * Pre-configured response scenarios
 */
export const chatParticipantResponseScenarios = {
    // Success scenarios
    getSuccess: (participant?: ChatParticipant) => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createSuccessResponse(
            participant || factory.createMockChatParticipant(),
        );
    },
    
    updateSuccess: (participant?: ChatParticipant) => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createSuccessResponse(
            participant || factory.createMockChatParticipant({ updatedAt: new Date().toISOString() }),
        );
    },
    
    listSuccess: (participants?: ChatParticipant[], chatId?: string) => {
        const factory = new ChatParticipantResponseFactory();
        const data = participants || factory.createChatParticipants(chatId || "default_chat", 5);
        return factory.createChatParticipantListResponse(data);
    },
    
    // Specific chat scenarios
    publicChatParticipants: () => {
        const factory = new ChatParticipantResponseFactory();
        const scenarios = factory.createParticipantsForScenarios();
        return factory.createChatParticipantListResponse(scenarios.publicChat);
    },
    
    teamChatParticipants: () => {
        const factory = new ChatParticipantResponseFactory();
        const scenarios = factory.createParticipantsForScenarios();
        return factory.createChatParticipantListResponse(scenarios.teamChat);
    },
    
    directMessageParticipants: () => {
        const factory = new ChatParticipantResponseFactory();
        const scenarios = factory.createParticipantsForScenarios();
        return factory.createChatParticipantListResponse(scenarios.directMessage);
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                id: "Participant ID is required and must be a valid format",
            },
        );
    },
    
    notFoundError: (participantId?: string) => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createNotFoundErrorResponse(
            participantId || "non-existent-participant",
        );
    },
    
    chatNotFoundError: (chatId?: string) => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createChatNotFoundErrorResponse(
            chatId || "non-existent-chat",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "view",
        );
    },
    
    alreadyParticipantError: (userId?: string, chatId?: string) => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createAlreadyParticipantErrorResponse(
            userId || "user_123",
            chatId || "chat_456",
        );
    },
    
    serverError: () => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    networkError: () => {
        const factory = new ChatParticipantResponseFactory();
        return factory.createNetworkErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new ChatParticipantMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new ChatParticipantMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new ChatParticipantMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new ChatParticipantMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const chatParticipantResponseFactory = new ChatParticipantResponseFactory();
export const chatParticipantMSWHandlers = new ChatParticipantMSWHandlers();
