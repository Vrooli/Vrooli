/**
 * ChatInvite API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for chat invite endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { http, type RestHandler } from "msw";
import type { 
    ChatInvite, 
    ChatInviteCreateInput, 
    ChatInviteUpdateInput,
    ChatInviteStatus,
    Chat,
    User, 
} from "@vrooli/shared";
import { 
    chatInviteValidation,
    ChatInviteStatus as ChatInviteStatusEnum, 
} from "@vrooli/shared";

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
 * ChatInvite API response factory
 */
export class ChatInviteResponseFactory {
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
     * Create successful chat invite response
     */
    createSuccessResponse(chatInvite: ChatInvite): APIResponse<ChatInvite> {
        return {
            data: chatInvite,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/chatInvite/${chatInvite.id}`,
                    related: {
                        chat: `${this.baseUrl}/api/chat/${chatInvite.chat.id}`,
                        user: `${this.baseUrl}/api/user/${chatInvite.user.id}`,
                        accept: `${this.baseUrl}/api/chatInvite/${chatInvite.id}/accept`,
                        decline: `${this.baseUrl}/api/chatInvite/${chatInvite.id}/decline`,
                    },
                },
            },
        };
    }
    
    /**
     * Create chat invite list response
     */
    createChatInviteListResponse(chatInvites: ChatInvite[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<ChatInvite> {
        const paginationData = pagination || {
            page: 1,
            pageSize: chatInvites.length,
            totalCount: chatInvites.length,
        };
        
        return {
            data: chatInvites,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/chatInvite?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/chatInvite",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(chatInviteId: string): APIErrorResponse {
        return {
            error: {
                code: "CHAT_INVITE_NOT_FOUND",
                message: `Chat invite with ID '${chatInviteId}' was not found`,
                details: {
                    chatInviteId,
                    searchCriteria: { id: chatInviteId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/chatInvite/${chatInviteId}`,
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
                message: `You do not have permission to ${operation} this chat invite`,
                details: {
                    operation,
                    requiredPermissions: ["chat:invite"],
                    userPermissions: ["chat:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/chatInvite",
            },
        };
    }
    
    /**
     * Create already processed error response
     */
    createAlreadyProcessedErrorResponse(status: ChatInviteStatus): APIErrorResponse {
        return {
            error: {
                code: "INVITE_ALREADY_PROCESSED",
                message: `This chat invite has already been ${status.toLowerCase()}`,
                details: {
                    currentStatus: status,
                    allowedActions: status === ChatInviteStatusEnum.Pending ? ["accept", "decline"] : [],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/chatInvite",
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
                path: "/api/chatInvite",
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
                path: "/api/chatInvite",
            },
        };
    }
    
    /**
     * Create mock chat data
     */
    private createMockChat(overrides?: Partial<Chat>): Chat {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultChat: Chat = {
            __typename: "Chat",
            id: `chat_${id}`,
            createdAt: now,
            updatedAt: now,
            invites: [],
            invitesCount: 1,
            messages: [],
            openToAnyoneWithInvite: false,
            participants: [],
            participantsCount: 2,
            publicId: `pub_${id}`,
            team: null,
            translations: [],
            translationsCount: 0,
            you: {
                __typename: "ChatYou",
                canDelete: false,
                canInvite: true,
                canRead: true,
                canUpdate: false,
            },
        };
        
        return {
            ...defaultChat,
            ...overrides,
        };
    }
    
    /**
     * Create mock user data
     */
    private createMockUser(overrides?: Partial<User>): User {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultUser: User = {
            __typename: "User",
            id: `user_${id}`,
            handle: `testuser_${id.slice(-4)}`,
            name: "Test User",
            createdAt: now,
            updatedAt: now,
            isBot: false,
            isPrivate: false,
            profileImage: null,
            bannerImage: null,
            premium: false,
            premiumExpiration: null,
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
        };
        
        return {
            ...defaultUser,
            ...overrides,
        };
    }
    
    /**
     * Create mock chat invite data
     */
    createMockChatInvite(overrides?: Partial<ChatInvite>): ChatInvite {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultChatInvite: ChatInvite = {
            __typename: "ChatInvite",
            id: `chatinvite_${id}`,
            createdAt: now,
            updatedAt: now,
            message: "You've been invited to join this chat conversation!",
            status: ChatInviteStatusEnum.Pending,
            chat: this.createMockChat(),
            user: this.createMockUser(),
            you: {
                __typename: "ChatInviteYou",
                canDelete: true,
                canUpdate: true,
            },
        };
        
        return {
            ...defaultChatInvite,
            ...overrides,
        };
    }
    
    /**
     * Create chat invite from API input
     */
    createChatInviteFromInput(input: ChatInviteCreateInput): ChatInvite {
        const chatInvite = this.createMockChatInvite();
        
        // Update chat invite based on input
        chatInvite.id = input.id;
        chatInvite.message = input.message || null;
        chatInvite.chat.id = input.chatConnect;
        chatInvite.user.id = input.userConnect;
        
        return chatInvite;
    }
    
    /**
     * Create multiple chat invites for different statuses
     */
    createChatInvitesForAllStatuses(): ChatInvite[] {
        return Object.values(ChatInviteStatusEnum).map(status => 
            this.createMockChatInvite({
                status,
                updatedAt: status !== ChatInviteStatusEnum.Pending ? new Date().toISOString() : new Date(Date.now() - 86400000).toISOString(),
            }),
        );
    }
    
    /**
     * Create chat invites for different scenarios
     */
    createChatInviteScenarios(): {
        pending: ChatInvite;
        accepted: ChatInvite;
        declined: ChatInvite;
        expired: ChatInvite;
        withCustomMessage: ChatInvite;
        teamChat: ChatInvite;
    } {
        const baseTime = Date.now();
        
        return {
            pending: this.createMockChatInvite({
                status: ChatInviteStatusEnum.Pending,
                createdAt: new Date(baseTime - 3600000).toISOString(), // 1 hour ago
            }),
            accepted: this.createMockChatInvite({
                status: ChatInviteStatusEnum.Accepted,
                createdAt: new Date(baseTime - 86400000).toISOString(), // 1 day ago
                updatedAt: new Date(baseTime - 3600000).toISOString(), // 1 hour ago
            }),
            declined: this.createMockChatInvite({
                status: ChatInviteStatusEnum.Declined,
                createdAt: new Date(baseTime - 172800000).toISOString(), // 2 days ago
                updatedAt: new Date(baseTime - 86400000).toISOString(), // 1 day ago
            }),
            expired: this.createMockChatInvite({
                status: ChatInviteStatusEnum.Pending,
                createdAt: new Date(baseTime - 604800000).toISOString(), // 7 days ago (expired)
            }),
            withCustomMessage: this.createMockChatInvite({
                message: "Hey! I'd love to discuss the project with you. Can you join our team chat?",
                status: ChatInviteStatusEnum.Pending,
            }),
            teamChat: this.createMockChatInvite({
                chat: this.createMockChat({
                    team: {
                        __typename: "Team",
                        id: `team_${this.generateId()}`,
                        handle: "awesome-team",
                        name: "Awesome Team",
                        createdAt: new Date().toISOString(),
                        updatedAt: new Date().toISOString(),
                        isPrivate: false,
                        members: [],
                        membersCount: 5,
                        translations: [],
                        translationsCount: 0,
                        you: {
                            __typename: "TeamYou",
                            canDelete: false,
                            canReport: false,
                            canUpdate: false,
                            isBookmarked: false,
                            isReacted: false,
                            reaction: null,
                        },
                    },
                }),
                status: ChatInviteStatusEnum.Pending,
            }),
        };
    }
    
    /**
     * Validate chat invite create input
     */
    async validateCreateInput(input: ChatInviteCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await chatInviteValidation.create.validate(input);
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
    
    /**
     * Validate chat invite update input
     */
    async validateUpdateInput(input: ChatInviteUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await chatInviteValidation.update.validate(input);
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
 * MSW handlers factory for chat invite endpoints
 */
export class ChatInviteMSWHandlers {
    private responseFactory: ChatInviteResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new ChatInviteResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all chat invite endpoints
     */
    createSuccessHandlers(): RestHandler[] {
        return [
            // Create chat invite
            http.post(`${this.responseFactory["baseUrl"]}/api/chatInvite`, async (req, res, ctx) => {
                const body = await req.json() as ChatInviteCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                // Create chat invite
                const chatInvite = this.responseFactory.createChatInviteFromInput(body);
                const response = this.responseFactory.createSuccessResponse(chatInvite);
                
                return res(
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            // Get chat invite by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/chatInvite/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const chatInvite = this.responseFactory.createMockChatInvite({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(chatInvite);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Update chat invite
            http.put(`${this.responseFactory["baseUrl"]}/api/chatInvite/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as ChatInviteUpdateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateUpdateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                const chatInvite = this.responseFactory.createMockChatInvite({ 
                    id: id as string,
                    message: body.message || null,
                    updatedAt: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(chatInvite);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Accept chat invite
            http.put(`${this.responseFactory["baseUrl"]}/api/chatInvite/:id/accept`, (req, res, ctx) => {
                const { id } = req.params;
                
                const chatInvite = this.responseFactory.createMockChatInvite({ 
                    id: id as string,
                    status: ChatInviteStatusEnum.Accepted,
                    updatedAt: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(chatInvite);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Decline chat invite
            http.put(`${this.responseFactory["baseUrl"]}/api/chatInvite/:id/decline`, (req, res, ctx) => {
                const { id } = req.params;
                
                const chatInvite = this.responseFactory.createMockChatInvite({ 
                    id: id as string,
                    status: ChatInviteStatusEnum.Declined,
                    updatedAt: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(chatInvite);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Delete chat invite
            http.delete(`${this.responseFactory["baseUrl"]}/api/chatInvite/:id`, (req, res, ctx) => {
                return res(ctx.status(204));
            }),
            
            // List chat invites
            http.get(`${this.responseFactory["baseUrl"]}/api/chatInvite`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const status = url.searchParams.get("status") as ChatInviteStatus;
                const chatId = url.searchParams.get("chatId");
                const userId = url.searchParams.get("userId");
                
                let chatInvites = this.responseFactory.createChatInvitesForAllStatuses();
                
                // Filter by status if specified
                if (status) {
                    chatInvites = chatInvites.filter(invite => invite.status === status);
                }
                
                // Filter by chat ID if specified
                if (chatId) {
                    chatInvites = chatInvites.filter(invite => invite.chat.id === chatId);
                }
                
                // Filter by user ID if specified
                if (userId) {
                    chatInvites = chatInvites.filter(invite => invite.user.id === userId);
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedChatInvites = chatInvites.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createChatInviteListResponse(
                    paginatedChatInvites,
                    {
                        page,
                        pageSize: limit,
                        totalCount: chatInvites.length,
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
     * Create error handlers for testing error scenarios
     */
    createErrorHandlers(): RestHandler[] {
        return [
            // Validation error
            http.post(`${this.responseFactory["baseUrl"]}/api/chatInvite`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        chatConnect: "Chat ID is required",
                        userConnect: "User ID is required",
                    })),
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/chatInvite/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string)),
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/chatInvite`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse("create")),
                );
            }),
            
            // Already processed error
            http.put(`${this.responseFactory["baseUrl"]}/api/chatInvite/:id/accept`, (req, res, ctx) => {
                return res(
                    ctx.status(409),
                    ctx.json(this.responseFactory.createAlreadyProcessedErrorResponse(ChatInviteStatusEnum.Accepted)),
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/chatInvite`, (req, res, ctx) => {
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
            http.post(`${this.responseFactory["baseUrl"]}/api/chatInvite`, async (req, res, ctx) => {
                const body = await req.json() as ChatInviteCreateInput;
                const chatInvite = this.responseFactory.createChatInviteFromInput(body);
                const response = this.responseFactory.createSuccessResponse(chatInvite);
                
                return res(
                    ctx.delay(delay),
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            http.put(`${this.responseFactory["baseUrl"]}/api/chatInvite/:id/accept`, (req, res, ctx) => {
                const { id } = req.params;
                const chatInvite = this.responseFactory.createMockChatInvite({ 
                    id: id as string,
                    status: ChatInviteStatusEnum.Accepted,
                    updatedAt: new Date().toISOString(),
                });
                const response = this.responseFactory.createSuccessResponse(chatInvite);
                
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
            http.post(`${this.responseFactory["baseUrl"]}/api/chatInvite`, (req, res, ctx) => {
                return res.networkError("Network connection failed");
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/chatInvite/:id`, (req, res, ctx) => {
                return res.networkError("Connection timeout");
            }),
            
            http.put(`${this.responseFactory["baseUrl"]}/api/chatInvite/:id/accept`, (req, res, ctx) => {
                return res.networkError("Network error during accept operation");
            }),
            
            http.put(`${this.responseFactory["baseUrl"]}/api/chatInvite/:id/decline`, (req, res, ctx) => {
                return res.networkError("Network error during decline operation");
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
export const chatInviteResponseScenarios = {
    // Success scenarios
    createSuccess: (chatInvite?: ChatInvite) => {
        const factory = new ChatInviteResponseFactory();
        return factory.createSuccessResponse(
            chatInvite || factory.createMockChatInvite(),
        );
    },
    
    acceptSuccess: (chatInviteId?: string) => {
        const factory = new ChatInviteResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockChatInvite({
                id: chatInviteId || `chatinvite_${factory["generateId"]()}`,
                status: ChatInviteStatusEnum.Accepted,
                updatedAt: new Date().toISOString(),
            }),
        );
    },
    
    declineSuccess: (chatInviteId?: string) => {
        const factory = new ChatInviteResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockChatInvite({
                id: chatInviteId || `chatinvite_${factory["generateId"]()}`,
                status: ChatInviteStatusEnum.Declined,
                updatedAt: new Date().toISOString(),
            }),
        );
    },
    
    listSuccess: (chatInvites?: ChatInvite[]) => {
        const factory = new ChatInviteResponseFactory();
        return factory.createChatInviteListResponse(
            chatInvites || factory.createChatInvitesForAllStatuses(),
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new ChatInviteResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                chatConnect: "Chat ID is required",
                userConnect: "User ID is required",
            },
        );
    },
    
    notFoundError: (chatInviteId?: string) => {
        const factory = new ChatInviteResponseFactory();
        return factory.createNotFoundErrorResponse(
            chatInviteId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new ChatInviteResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },
    
    alreadyProcessedError: (status?: ChatInviteStatus) => {
        const factory = new ChatInviteResponseFactory();
        return factory.createAlreadyProcessedErrorResponse(
            status || ChatInviteStatusEnum.Accepted,
        );
    },
    
    serverError: () => {
        const factory = new ChatInviteResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new ChatInviteMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new ChatInviteMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new ChatInviteMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new ChatInviteMSWHandlers().createNetworkErrorHandlers(),
    
    // Scenario collections
    allScenarios: () => {
        const factory = new ChatInviteResponseFactory();
        return factory.createChatInviteScenarios();
    },
};

// Export factory instances for easy use
export const chatInviteResponseFactory = new ChatInviteResponseFactory();
export const chatInviteMSWHandlers = new ChatInviteMSWHandlers();
