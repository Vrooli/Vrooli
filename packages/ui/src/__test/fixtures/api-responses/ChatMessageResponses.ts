/**
 * ChatMessage API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for chat message endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { http, HttpResponse, type RequestHandler } from "msw";
import type { 
    ChatMessage, 
    ChatMessageCreateInput, 
    ChatMessageUpdateInput,
    ChatMessageSearchInput,
    ChatMessageSearchTreeInput,
    ChatMessageSearchResult,
    ChatMessageSearchTreeResult,
    Chat,
    User,
    ChatMessageParent,
    ReactionSummary,
    ChatMessageYou,
} from "@vrooli/shared";
import { 
    chatMessageValidation,
    type MessageConfigObject,
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
 * ChatMessage API response factory
 */
export class ChatMessageResponseFactory {
    private readonly baseUrl: string;
    
    constructor(baseUrl: string = process.env.VITE_SERVER_URL || "http://localhost:5329") {
        this.baseUrl = baseUrl;
    }
    
    /**
     * Generate unique request ID
     */
    private generateRequestId(): string {
        return `req_${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
    }
    
    /**
     * Generate unique resource ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
    }
    
    /**
     * Create successful chat message response
     */
    createSuccessResponse(chatMessage: ChatMessage): APIResponse<ChatMessage> {
        return {
            data: chatMessage,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/chatMessage/${chatMessage.id}`,
                    related: {
                        chat: `${this.baseUrl}/api/chat/${chatMessage.chat.id}`,
                        user: `${this.baseUrl}/api/user/${chatMessage.user.id}`,
                        ...(chatMessage.parent && {
                            parent: `${this.baseUrl}/api/chatMessage/${chatMessage.parent.id}`,
                        }),
                    },
                },
            },
        };
    }
    
    /**
     * Create chat message list response
     */
    createChatMessageListResponse(chatMessages: ChatMessage[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<ChatMessage> {
        const paginationData = pagination || {
            page: 1,
            pageSize: chatMessages.length,
            totalCount: chatMessages.length,
        };
        
        return {
            data: chatMessages,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/chatMessage?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
     * Create chat message search response
     */
    createSearchResponse(chatMessages: ChatMessage[], hasNextPage = false): APIResponse<ChatMessageSearchResult> {
        return {
            data: {
                __typename: "ChatMessageSearchResult",
                edges: chatMessages.map((message, index) => ({
                    __typename: "ChatMessageEdge" as const,
                    cursor: `cursor_${index}`,
                    node: message,
                })),
                pageInfo: {
                    __typename: "PageInfo",
                    hasNextPage,
                    endCursor: chatMessages.length > 0 ? `cursor_${chatMessages.length - 1}` : null,
                },
            },
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/chatMessage/search`,
                },
            },
        };
    }
    
    /**
     * Create chat message tree response
     */
    createTreeResponse(
        chatMessages: ChatMessage[], 
        hasMoreUp = false, 
        hasMoreDown = false,
    ): APIResponse<ChatMessageSearchTreeResult> {
        return {
            data: {
                __typename: "ChatMessageSearchTreeResult",
                messages: chatMessages,
                hasMoreUp,
                hasMoreDown,
            },
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/chatMessageTree`,
                },
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
                path: "/api/chatMessage",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(messageId: string): APIErrorResponse {
        return {
            error: {
                code: "CHAT_MESSAGE_NOT_FOUND",
                message: `Chat message with ID '${messageId}' was not found`,
                details: {
                    messageId,
                    searchCriteria: { id: messageId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/chatMessage/${messageId}`,
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
                message: `You do not have permission to ${operation} this chat message`,
                details: {
                    operation,
                    requiredPermissions: ["chat:write"],
                    userPermissions: ["chat:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/chatMessage",
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
                path: "/api/chatMessage",
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
                path: "/api/chatMessage",
            },
        };
    }
    
    /**
     * Create rate limit error response
     */
    createRateLimitErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: "RATE_LIMIT_EXCEEDED",
                message: "Too many messages sent. Please wait before sending another message.",
                details: {
                    retryAfter: 60000,
                    limit: 100,
                    window: "1 hour",
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/chatMessage",
            },
        };
    }
    
    /**
     * Create mock user data
     */
    private createMockUser(overrides?: Partial<User>): User {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        return {
            __typename: "User",
            id,
            handle: "testuser",
            name: "Test User",
            createdAt: now,
            updatedAt: now,
            isBot: false,
            isPrivate: false,
            profileImage: null,
            bannerImage: null,
            premium: null,
            wallets: [],
            translations: [],
            you: {
                __typename: "UserYou",
                canDelete: false,
                canReport: false,
                canUpdate: false,
                isBookmarked: false,
                isViewed: false,
            },
            ...overrides,
        } as unknown as User;
    }
    
    /**
     * Create mock chat data
     */
    private createMockChat(overrides?: Partial<Chat>): Chat {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        return {
            __typename: "Chat",
            id,
            createdAt: now,
            updatedAt: now,
            openToAnyoneWithInvite: false,
            invites: [],
            invitesCount: 0,
            messages: [],
            participants: [],
            participantsCount: 1,
            translations: [],
            translationsCount: 0,
            you: {
                __typename: "ChatYou",
                canDelete: true,
                canInvite: true,
                canRead: true,
                canUpdate: true,
            },
            ...overrides,
        } as unknown as Chat;
    }
    
    /**
     * Create mock message config
     */
    createMockConfig(overrides?: Partial<MessageConfigObject>): MessageConfigObject {
        return {
            __version: "1.0.0",
            resources: [],
            role: "user",
            ...overrides,
        };
    }
    
    /**
     * Create mock chat message data
     */
    createMockChatMessage(overrides?: Partial<ChatMessage>): ChatMessage {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultMessage: ChatMessage = {
            __typename: "ChatMessage",
            id,
            createdAt: now,
            updatedAt: now,
            config: this.createMockConfig(),
            language: "en",
            text: "Hello, this is a test message!",
            sequence: 1,
            versionIndex: 0,
            score: 0,
            reportsCount: 0,
            chat: this.createMockChat(),
            user: this.createMockUser(),
            parent: null,
            reactionSummaries: [],
            reports: [],
            you: {
                __typename: "ChatMessageYou",
                canDelete: true,
                canReact: true,
                canReply: true,
                canReport: false,
                canUpdate: true,
                reaction: null,
            },
        };
        
        return {
            ...defaultMessage,
            ...overrides,
        };
    }
    
    /**
     * Create chat message from API input
     */
    createChatMessageFromInput(input: ChatMessageCreateInput): ChatMessage {
        const message = this.createMockChatMessage();
        
        // Update message based on input
        message.id = input.id;
        message.text = input.text;
        message.language = input.language;
        message.config = input.config;
        message.versionIndex = input.versionIndex;
        message.chat.id = input.chatConnect;
        message.user.id = input.userConnect;
        
        if (input.parentConnect) {
            message.parent = {
                __typename: "ChatMessageParent",
                id: input.parentConnect,
                createdAt: new Date().toISOString(),
            };
        }
        
        return message;
    }
    
    /**
     * Create multiple chat messages with different scenarios
     */
    createChatMessagesForScenarios(): ChatMessage[] {
        return [
            // User message
            this.createMockChatMessage({
                text: "Hello, how can I help you today?",
                config: this.createMockConfig({ role: "user" }),
                sequence: 1,
            }),
            
            // Bot/Assistant response
            this.createMockChatMessage({
                text: "I'm here to assist you with any questions you might have.",
                config: this.createMockConfig({ 
                    role: "assistant",
                    respondingBots: ["@all"],
                }),
                sequence: 2,
                user: this.createMockUser({ 
                    isBot: true, 
                    handle: "assistant",
                    name: "AI Assistant", 
                }),
            }),
            
            // System message
            this.createMockChatMessage({
                text: "Chat has been created successfully.",
                config: this.createMockConfig({ role: "system" }),
                sequence: 3,
                user: this.createMockUser({ 
                    isBot: true, 
                    handle: "system",
                    name: "System", 
                }),
            }),
            
            // Message with parent (threaded)
            this.createMockChatMessage({
                text: "This is a reply to the first message.",
                config: this.createMockConfig({ role: "user" }),
                sequence: 4,
                parent: {
                    __typename: "ChatMessageParent",
                    id: "parent_message_id",
                    createdAt: new Date().toISOString(),
                },
            }),
            
            // Message with tool calls
            this.createMockChatMessage({
                text: "Let me search for that information.",
                config: this.createMockConfig({ 
                    role: "assistant",
                    toolCalls: [{
                        id: "tool_call_1",
                        function: {
                            name: "search_web",
                            arguments: JSON.stringify({ query: "user question" }),
                        },
                        result: {
                            success: true,
                            output: "Search results found",
                        },
                    }],
                }),
                sequence: 5,
            }),
        ];
    }
    
    /**
     * Create messages for thread scenario
     */
    createThreadMessages(): ChatMessage[] {
        const parentMessage = this.createMockChatMessage({
            text: "What's the weather like today?",
            sequence: 1,
        });
        
        const reply1 = this.createMockChatMessage({
            text: "Let me check the weather for you.",
            sequence: 2,
            parent: {
                __typename: "ChatMessageParent",
                id: parentMessage.id,
                createdAt: parentMessage.createdAt,
            },
        });
        
        const reply2 = this.createMockChatMessage({
            text: "It's sunny with a temperature of 75°F.",
            sequence: 3,
            parent: {
                __typename: "ChatMessageParent",
                id: parentMessage.id,
                createdAt: parentMessage.createdAt,
            },
        });
        
        return [parentMessage, reply1, reply2];
    }
    
    /**
     * Validate chat message create input
     */
    async validateCreateInput(input: ChatMessageCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await chatMessageValidation.create({}).validate(input);
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
 * MSW handlers factory for chat message endpoints
 */
export class ChatMessageMSWHandlers {
    private responseFactory: ChatMessageResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new ChatMessageResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all chat message endpoints
     */
    createSuccessHandlers(): RequestHandler[] {
        return [
            // Create chat message
            http.post(`${this.responseFactory["baseUrl"]}/api/chatMessage`, async ({ request }) => {
                const body = await request.json() as ChatMessageCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return HttpResponse.json(
                        this.responseFactory.createValidationErrorResponse(validation.errors || {}),
                        { status: 400 }
                    );
                }
                
                // Create message
                const message = this.responseFactory.createChatMessageFromInput(body);
                const response = this.responseFactory.createSuccessResponse(message);
                
                return HttpResponse.json(response, { status: 201 });
            }),
            
            // Get message by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/chatMessage/:id`, ({ params }) => {
                const { id } = params;
                
                const message = this.responseFactory.createMockChatMessage({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(message);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Update message
            http.put(`${this.responseFactory["baseUrl"]}/api/chatMessage/:id`, async ({ params, request }) => {
                const { id } = params;
                const body = await request.json() as ChatMessageUpdateInput;
                
                const message = this.responseFactory.createMockChatMessage({ 
                    id: id as string,
                    text: body.text || "Updated message text",
                    config: body.config || this.responseFactory.createMockChatMessage().config,
                    updatedAt: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(message);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Delete message
            http.delete(`${this.responseFactory["baseUrl"]}/api/chatMessage/:id`, () => {
                return new HttpResponse(null, { status: 204 });
            }),
            
            // Search messages
            http.get(`${this.responseFactory["baseUrl"]}/api/chatMessage`, ({ request }) => {
                const url = new URL(request.url);
                const chatId = url.searchParams.get("chatId");
                const take = parseInt(url.searchParams.get("take") || "10");
                const searchString = url.searchParams.get("searchString");
                
                let messages = this.responseFactory.createChatMessagesForScenarios();
                
                // Filter by chat ID if specified
                if (chatId) {
                    messages = messages.map(m => ({ ...m, chat: { ...m.chat, id: chatId } }));
                }
                
                // Filter by search string if specified
                if (searchString) {
                    messages = messages.filter(m => 
                        m.text.toLowerCase().includes(searchString.toLowerCase()),
                    );
                }
                
                // Limit results
                messages = messages.slice(0, take);
                
                const response = this.responseFactory.createSearchResponse(messages, messages.length === take);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Get message tree
            http.get(`${this.responseFactory["baseUrl"]}/api/chatMessageTree`, ({ request, params }) => {
                const url = new URL(request.url);
                const chatId = url.searchParams.get("chatId");
                const startId = url.searchParams.get("startId");
                const take = parseInt(url.searchParams.get("take") || "10");
                
                let messages = this.responseFactory.createThreadMessages();
                
                // Filter by chat ID if specified
                if (chatId) {
                    messages = messages.map(m => ({ ...m, chat: { ...m.chat, id: chatId } }));
                }
                
                // Limit results
                messages = messages.slice(0, take);
                
                const response = this.responseFactory.createTreeResponse(
                    messages, 
                    false, // hasMoreUp
                    messages.length === take, // hasMoreDown
                );
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Regenerate response
            http.post(`${this.responseFactory["baseUrl"]}/api/regenerateResponse`, async ({ request, params }) => {
                const body = await request.json();
                
                // Create a new assistant response
                const message = this.responseFactory.createMockChatMessage({
                    text: "This is a regenerated response.",
                    config: this.responseFactory.createMockConfig({ role: "assistant" }),
                    user: this.responseFactory.createMockChatMessage().user, // Reset user to default
                });
                
                const response = this.responseFactory.createSuccessResponse(message);
                
                return HttpResponse.json(response, { status: 201 });
            }),
        ];
    }
    
    /**
     * Create error handlers for testing error scenarios
     */
    createErrorHandlers(): RequestHandler[] {
        return [
            // Validation error
            http.post(`${this.responseFactory["baseUrl"]}/api/chatMessage`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createValidationErrorResponse({
                        text: "Message text is required",
                        chatConnect: "Chat ID must be specified",
                        userConnect: "User ID must be specified",
                    }),
                    { status: 400 }
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/chatMessage/:id`, ({ request, params }) => {
                const { id } = params;
                return HttpResponse.json(
                    this.responseFactory.createNotFoundErrorResponse(id as string), { status: 404 });
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/chatMessage`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createPermissionErrorResponse("create"), { status: 403 });
            }),
            
            // Rate limit error
            http.post(`${this.responseFactory["baseUrl"]}/api/chatMessage`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createRateLimitErrorResponse(), { status: 429 });
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/chatMessage`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createServerErrorResponse(), { status: 500 });
            }),
        ];
    }
    
    /**
     * Create loading simulation handlers
     */
    createLoadingHandlers(delay = 2000): RequestHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/chatMessage`, async ({ request, params }) => {
                const body = await request.json() as ChatMessageCreateInput;
                const message = this.responseFactory.createChatMessageFromInput(body);
                const response = this.responseFactory.createSuccessResponse(message);
                
                await new Promise(resolve => setTimeout(resolve, delay));
                return HttpResponse.json(response, { status: 201 });
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/chatMessageTree`, async ({ request, params }) => {
                const messages = this.responseFactory.createThreadMessages();
                const response = this.responseFactory.createTreeResponse(messages);
                
                await new Promise(resolve => setTimeout(resolve, delay));
                return HttpResponse.json(response, { status: 200 });
            }),
        ];
    }
    
    /**
     * Create network error handlers
     */
    createNetworkErrorHandlers(): RequestHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/chatMessage`, ({ request, params }) => {
                return HttpResponse.error();
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/chatMessage/:id`, ({ request, params }) => {
                return HttpResponse.error();
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/chatMessageTree`, ({ request, params }) => {
                return HttpResponse.error();
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
    }): RequestHandler {
        const { endpoint, method, status, response, delay } = config;
        const fullEndpoint = `${this.responseFactory["baseUrl"]}${endpoint}`;
        
        const httpMethod = method.toLowerCase() as keyof typeof http;
        return http[httpMethod](fullEndpoint, async ({ request, params }) => {
            if (delay) {
                await new Promise(resolve => setTimeout(resolve, delay));
            }
            
            return HttpResponse.json(response, { status });
        });
    }
}

/**
 * Pre-configured response scenarios
 */
export const chatMessageResponseScenarios = {
    // Success scenarios
    createSuccess: (message?: ChatMessage) => {
        const factory = new ChatMessageResponseFactory();
        return factory.createSuccessResponse(
            message || factory.createMockChatMessage(),
        );
    },
    
    searchSuccess: (messages?: ChatMessage[]) => {
        const factory = new ChatMessageResponseFactory();
        return factory.createSearchResponse(
            messages || factory.createChatMessagesForScenarios(),
        );
    },
    
    treeSuccess: (messages?: ChatMessage[]) => {
        const factory = new ChatMessageResponseFactory();
        return factory.createTreeResponse(
            messages || factory.createThreadMessages(),
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new ChatMessageResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                text: "Message text is required",
                chatConnect: "Chat ID is required",
                userConnect: "User ID is required",
            },
        );
    },
    
    notFoundError: (messageId?: string) => {
        const factory = new ChatMessageResponseFactory();
        return factory.createNotFoundErrorResponse(
            messageId || "non-existent-message-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new ChatMessageResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },
    
    rateLimitError: () => {
        const factory = new ChatMessageResponseFactory();
        return factory.createRateLimitErrorResponse();
    },
    
    serverError: () => {
        const factory = new ChatMessageResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new ChatMessageMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new ChatMessageMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new ChatMessageMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new ChatMessageMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const chatMessageResponseFactory = new ChatMessageResponseFactory();
export const chatMessageMSWHandlers = new ChatMessageMSWHandlers();
