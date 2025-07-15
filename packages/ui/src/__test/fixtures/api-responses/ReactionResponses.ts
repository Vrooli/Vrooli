/**
 * Reaction API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for reaction endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

// AI_CHECK: TYPE_SAFETY=fixed-msw-v2-migration-and-arithmetic-type-error | LAST: 2025-07-02
import { http, type HttpHandler } from "msw";
import type { 
    Reaction, 
    ReactInput,
    ReactionSearchResult,
    ReactionFor,
    ReactionSummary,
    Success,
    User,
    ChatMessage,
    Comment,
    Issue,
    Resource,
} from "@vrooli/shared";
import { 
    ReactionFor as ReactionForEnum,
    getReactionScore, 
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
        details?: Record<string, unknown>;
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
 * Common emoji reactions used in tests
 */
export const COMMON_REACTIONS = {
    positive: ["👍", "❤️", "🎉", "🚀", "😊", "👏", "🔥", "💯"],
    negative: ["👎", "😕", "😡", "🤮"],
    neutral: ["🤔", "😐", "🤷", "👀", "📌", "💭", "🔔", "⭐"],
};

/**
 * Reaction API response factory
 */
export class ReactionResponseFactory {
    private readonly baseUrl: string;
    
    constructor(baseUrl: string = process.env.VITE_SERVER_URL || "http://localhost:5329") {
        this.baseUrl = baseUrl;
    }
    
    /**
     * Generate unique request ID
     */
    private generateRequestId(): string {
        const BASE_36 = 36;
        const START_INDEX = 2;
        const END_INDEX = 11;
        const randomPart = Math.random().toString(BASE_36).substring(START_INDEX, END_INDEX);
        return `req_${Date.now()}_${randomPart}`;
    }
    
    /**
     * Generate unique resource ID
     */
    private generateId(): string {
        const BASE_36 = 36;
        const START_INDEX = 2;
        const END_INDEX = 11;
        const randomPart = Math.random().toString(BASE_36).substring(START_INDEX, END_INDEX);
        return `${Date.now()}_${randomPart}`;
    }
    
    /**
     * Get random emoji from category
     */
    private getRandomEmoji(category: "positive" | "negative" | "neutral" = "positive"): string {
        const emojis = COMMON_REACTIONS[category];
        return emojis[Math.floor(Math.random() * emojis.length)];
    }
    
    /**
     * Create successful reaction response
     */
    createSuccessResponse(success = true): APIResponse<Success> {
        return {
            data: {
                __typename: "Success",
                success,
            },
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
            },
        };
    }
    
    /**
     * Create reaction search result response
     */
    createReactionSearchResponse(reactions: Reaction[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): ReactionSearchResult {
        const paginationData = pagination || {
            page: 1,
            pageSize: reactions.length,
            totalCount: reactions.length,
        };
        
        return {
            __typename: "ReactionSearchResult",
            edges: reactions.map(reaction => ({
                __typename: "ReactionEdge",
                cursor: reaction.id,
                node: reaction,
            })),
            pageInfo: {
                __typename: "PageInfo",
                hasNextPage: paginationData.page * paginationData.pageSize < paginationData.totalCount,
                endCursor: reactions[reactions.length - 1]?.id || null,
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
                path: "/api/reaction",
            },
        };
    }
    
    /**
     * Create duplicate reaction error response
     */
    createDuplicateReactionErrorResponse(objectId: string, objectType: string): APIErrorResponse {
        return {
            error: {
                code: "DUPLICATE_REACTION",
                message: `You have already reacted to this ${objectType}`,
                details: {
                    objectId,
                    objectType,
                    existingReaction: true,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/reaction",
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
                message: "Too many reaction requests. Please slow down.",
                details: {
                    limit: 1000,
                    window: "1 hour",
                    retryAfter: 3600,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/reaction",
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
                message: `You do not have permission to ${operation} reactions`,
                details: {
                    operation,
                    requiredPermissions: ["reaction:write"],
                    userPermissions: ["reaction:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/reaction",
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
                path: "/api/reaction",
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
                path: "/api/reaction",
            },
        };
    }
    
    /**
     * Create mock user
     */
    private createMockUser(id?: string): User {
        const userId = id || `user_${this.generateId()}`;
        
        const HANDLE_SUFFIX_LENGTH = 6;
        const NAME_SUFFIX_LENGTH = 4;
        return {
            __typename: "User",
            id: userId,
            handle: `user_${userId.slice(Math.max(0, userId.length - HANDLE_SUFFIX_LENGTH))}`,
            name: `Test User ${userId.slice(Math.max(0, userId.length - NAME_SUFFIX_LENGTH))}`,
        } as User;
    }
    
    /**
     * Create mock reaction target based on type
     */
    private createMockReactionTarget(type: ReactionFor): ChatMessage | Comment | Issue | Resource {
        const id = `${type.toLowerCase()}_${this.generateId()}`;
        
        switch (type) {
            case ReactionForEnum.ChatMessage:
                return {
                    __typename: "ChatMessage",
                    id,
                } as ChatMessage;
                
            case ReactionForEnum.Comment:
                return {
                    __typename: "Comment",
                    id,
                } as Comment;
                
            case ReactionForEnum.Issue:
                return {
                    __typename: "Issue",
                    id,
                } as Issue;
                
            case ReactionForEnum.Resource:
                return {
                    __typename: "Resource",
                    id,
                } as Resource;
                
            default:
                throw new Error(`Unknown reaction type: ${type}`);
        }
    }
    
    /**
     * Create mock reaction data
     */
    createMockReaction(overrides?: Partial<Reaction>): Reaction {
        const id = this.generateId();
        const emoji = this.getRandomEmoji("positive");
        
        const defaultReaction: Reaction = {
            __typename: "Reaction",
            id,
            emoji,
            by: this.createMockUser(),
            to: this.createMockReactionTarget(ReactionForEnum.Comment),
        };
        
        return {
            ...defaultReaction,
            ...overrides,
        };
    }
    
    /**
     * Create multiple reactions for different users on the same object
     */
    createMultipleReactions(targetType: ReactionFor, targetId: string, count?: number): Reaction[] {
        const DEFAULT_REACTION_COUNT = 10;
        const actualCount = count ?? DEFAULT_REACTION_COUNT;
        const target = this.createMockReactionTarget(targetType);
        target.id = targetId;
        
        return Array.from({ length: actualCount }, (_, index) => {
            const categories: Array<"positive" | "negative" | "neutral"> = ["positive", "negative", "neutral"];
            const category = categories[index % categories.length];
            
            return this.createMockReaction({
                emoji: this.getRandomEmoji(category),
                by: this.createMockUser(`user_${index}`),
                to: target,
            });
        });
    }
    
    /**
     * Create reaction summary for an object
     */
    createReactionSummary(reactions: Reaction[]): ReactionSummary[] {
        const emojiCounts = reactions.reduce((acc, reaction) => {
            acc[reaction.emoji] = (acc[reaction.emoji] || 0) + 1;
            return acc;
        }, {} as Record<string, number>);
        
        return Object.entries(emojiCounts)
            .map(([emoji, count]) => ({
                __typename: "ReactionSummary" as const,
                emoji,
                count,
            }))
            .sort((a, b) => {
                // Sort by score first (positive, neutral, negative)
                const scoreA = getReactionScore(a.emoji);
                const scoreB = getReactionScore(b.emoji);
                if (scoreA !== scoreB) return scoreB - scoreA;
                
                // Then by count
                return Number(b.count) - Number(a.count);
            });
    }
    
    /**
     * Validate react input
     */
    async validateReactInput(input: ReactInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};
        
        if (!input.forConnect) {
            errors.forConnect = "Target object ID is required";
        }
        
        if (!input.reactionFor) {
            errors.reactionFor = "Reaction type must be specified";
        } else if (!Object.values(ReactionForEnum).includes(input.reactionFor)) {
            errors.reactionFor = `Invalid reaction type. Must be one of: ${Object.values(ReactionForEnum).join(", ")}`;
        }
        
        const MAX_EMOJI_LENGTH = 10;
        if (input.emoji && input.emoji.length > MAX_EMOJI_LENGTH) {
            errors.emoji = "Emoji must be 10 characters or less";
        }
        
        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }
}

/**
 * MSW handlers factory for reaction endpoints
 */
export class ReactionMSWHandlers {
    private responseFactory: ReactionResponseFactory;
    private userReactions: Map<string, Map<string, string>>; // userId -> objectId -> emoji
    
    constructor(baseUrl?: string) {
        this.responseFactory = new ReactionResponseFactory(baseUrl);
        this.userReactions = new Map();
    }
    
    /**
     * Track user reaction
     */
    private trackUserReaction(userId: string, objectId: string, emoji: string | null): void {
        if (!this.userReactions.has(userId)) {
            this.userReactions.set(userId, new Map());
        }
        
        const userMap = this.userReactions.get(userId);
        if (userMap) {
            if (emoji === null) {
                userMap.delete(objectId);
            } else {
                userMap.set(objectId, emoji);
            }
        }
    }
    
    /**
     * Get user's reaction for an object
     */
    private getUserReaction(userId: string, objectId: string): string | null {
        return this.userReactions.get(userId)?.get(objectId) || null;
    }
    
    /**
     * Create success handlers for all reaction endpoints
     */
    createSuccessHandlers(): HttpHandler[] {
        return [
            // React to object (add/update/remove reaction)
            http.post(`${this.responseFactory["baseUrl"]}/api/reaction`, async ({ request }) => {
                const body = await request.json() as ReactInput;
                const userId = "user_test"; // In real tests, extract from auth header
                
                // Validate input
                const validation = await this.responseFactory.validateReactInput(body);
                if (!validation.valid) {
                    return Response.json(
                        this.responseFactory.createValidationErrorResponse(validation.errors || {}),
                        { status: 400 },
                    );
                }
                
                // Track the reaction
                this.trackUserReaction(userId, body.forConnect, body.emoji || null);
                
                // Return success
                const response = this.responseFactory.createSuccessResponse(true);
                
                return Response.json(response, { status: 200 });
            }),
            
            // Search reactions
            http.get(`${this.responseFactory["baseUrl"]}/api/reaction`, ({ request }) => {
                const url = new URL(request.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const chatMessageId = url.searchParams.get("chatMessageId");
                const commentId = url.searchParams.get("commentId");
                const issueId = url.searchParams.get("issueId");
                const resourceId = url.searchParams.get("resourceId");
                
                let reactions: Reaction[] = [];
                
                const DEFAULT_CHAT_MESSAGE_REACTIONS = 15;
                const DEFAULT_COMMENT_REACTIONS = 10;
                const DEFAULT_ISSUE_REACTIONS = 8;
                const DEFAULT_RESOURCE_REACTIONS = 5;
                const DEFAULT_MIXED_COMMENT_REACTIONS = 3;
                const DEFAULT_MIXED_ISSUE_REACTIONS = 2;
                const DEFAULT_MIXED_RESOURCE_REACTIONS = 2;
                
                // Generate reactions based on query parameters
                if (chatMessageId) {
                    reactions = this.responseFactory.createMultipleReactions(
                        ReactionForEnum.ChatMessage, 
                        chatMessageId, 
                        DEFAULT_CHAT_MESSAGE_REACTIONS,
                    );
                } else if (commentId) {
                    reactions = this.responseFactory.createMultipleReactions(
                        ReactionForEnum.Comment, 
                        commentId, 
                        DEFAULT_COMMENT_REACTIONS,
                    );
                } else if (issueId) {
                    reactions = this.responseFactory.createMultipleReactions(
                        ReactionForEnum.Issue, 
                        issueId, 
                        DEFAULT_ISSUE_REACTIONS,
                    );
                } else if (resourceId) {
                    reactions = this.responseFactory.createMultipleReactions(
                        ReactionForEnum.Resource, 
                        resourceId, 
                        DEFAULT_RESOURCE_REACTIONS,
                    );
                } else {
                    // Return mixed reactions
                    reactions = [
                        ...this.responseFactory.createMultipleReactions(ReactionForEnum.Comment, "comment_1", DEFAULT_MIXED_COMMENT_REACTIONS),
                        ...this.responseFactory.createMultipleReactions(ReactionForEnum.Issue, "issue_1", DEFAULT_MIXED_ISSUE_REACTIONS),
                        ...this.responseFactory.createMultipleReactions(ReactionForEnum.Resource, "resource_1", DEFAULT_MIXED_RESOURCE_REACTIONS),
                    ];
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedReactions = reactions.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createReactionSearchResponse(
                    paginatedReactions,
                    {
                        page,
                        pageSize: limit,
                        totalCount: reactions.length,
                    },
                );
                
                return Response.json(response, { status: 200 });
            }),
        ];
    }
    
    /**
     * Create error handlers for testing error scenarios
     */
    createErrorHandlers(): HttpHandler[] {
        return [
            // Validation error
            http.post(`${this.responseFactory["baseUrl"]}/api/reaction`, () => {
                return Response.json(
                    this.responseFactory.createValidationErrorResponse({
                        forConnect: "Target object ID is required",
                        reactionFor: "Reaction type must be specified",
                    }),
                    { status: 400 },
                );
            }),
            
            // Rate limit error
            http.post(`${this.responseFactory["baseUrl"]}/api/reaction`, () => {
                return Response.json(
                    this.responseFactory.createRateLimitErrorResponse(),
                    { status: 429 },
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/reaction`, () => {
                return Response.json(
                    this.responseFactory.createPermissionErrorResponse("create"),
                    { status: 403 },
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/reaction`, () => {
                return Response.json(
                    this.responseFactory.createServerErrorResponse(),
                    { status: 500 },
                );
            }),
        ];
    }
    
    /**
     * Create loading simulation handlers
     */
    createLoadingHandlers(delay?: number): HttpHandler[] {
        const DEFAULT_DELAY = 2000;
        const actualDelay = delay ?? DEFAULT_DELAY;
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/reaction`, async ({ request }) => {
                const body = await request.json() as ReactInput;
                const validation = await this.responseFactory.validateReactInput(body);
                
                if (!validation.valid) {
                    return Response.json(
                        this.responseFactory.createValidationErrorResponse(validation.errors || {}),
                        { status: 400 },
                    );
                }
                
                const response = this.responseFactory.createSuccessResponse(true);
                
                // Simulate delay
                await new Promise(resolve => setTimeout(resolve, actualDelay));
                
                return Response.json(response, { status: 200 });
            }),
        ];
    }
    
    /**
     * Create network error handlers
     */
    createNetworkErrorHandlers(): HttpHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/reaction`, () => {
                return Response.error();
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/reaction`, () => {
                return Response.error();
            }),
        ];
    }
    
    /**
     * Create optimistic update handlers (simulating immediate UI updates)
     */
    createOptimisticHandlers(): HttpHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/reaction`, async ({ request }) => {
                const _body = await request.json() as ReactInput;
                
                // Simulate very fast response for optimistic UI updates
                const response = this.responseFactory.createSuccessResponse(true);
                
                // Simulate very small delay
                const OPTIMISTIC_DELAY = 50;
                await new Promise(resolve => setTimeout(resolve, OPTIMISTIC_DELAY));
                
                return Response.json(response, { status: 200 });
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
        response: unknown;
        delay?: number;
    }): HttpHandler {
        const { endpoint, method, status, response, delay } = config;
        const fullEndpoint = `${this.responseFactory["baseUrl"]}${endpoint}`;
        
        return http[method.toLowerCase() as keyof typeof http](fullEndpoint, async () => {
            if (delay) {
                await new Promise(resolve => setTimeout(resolve, delay));
            }
            
            return Response.json(response, { status });
        });
    }
}

/**
 * Pre-configured response scenarios
 */
export const reactionResponseScenarios = {
    // Success scenarios
    reactSuccess: (success = true) => {
        const factory = new ReactionResponseFactory();
        return factory.createSuccessResponse(success);
    },
    
    searchSuccess: (reactions?: Reaction[]) => {
        const factory = new ReactionResponseFactory();
        const DEFAULT_SEARCH_REACTIONS = 10;
        return factory.createReactionSearchResponse(
            reactions || factory.createMultipleReactions(ReactionForEnum.Comment, "comment_123", DEFAULT_SEARCH_REACTIONS),
        );
    },
    
    reactionSummary: (targetType: ReactionFor = ReactionForEnum.Comment, count?: number) => {
        const DEFAULT_REACTION_COUNT = 20;
        const actualCount = count ?? DEFAULT_REACTION_COUNT;
        const factory = new ReactionResponseFactory();
        const reactions = factory.createMultipleReactions(targetType, "target_123", actualCount);
        return factory.createReactionSummary(reactions);
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new ReactionResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                forConnect: "Target object is required",
                reactionFor: "Reaction type must be specified",
            },
        );
    },
    
    duplicateReactionError: (objectId?: string, objectType?: string) => {
        const factory = new ReactionResponseFactory();
        return factory.createDuplicateReactionErrorResponse(
            objectId || "comment_123",
            objectType || "comment",
        );
    },
    
    rateLimitError: () => {
        const factory = new ReactionResponseFactory();
        return factory.createRateLimitErrorResponse();
    },
    
    permissionError: (operation?: string) => {
        const factory = new ReactionResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },
    
    serverError: () => {
        const factory = new ReactionResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // Test data generators
    generateReactions: (type: ReactionFor, count: number) => {
        const factory = new ReactionResponseFactory();
        return factory.createMultipleReactions(type, `${type.toLowerCase()}_test`, count);
    },
    
    generateEmojiDistribution: () => {
        const factory = new ReactionResponseFactory();
        const reactions: Reaction[] = [];
        
        // Create realistic distribution
        const distribution = [
            { emoji: "👍", count: 45 },
            { emoji: "❤️", count: 32 },
            { emoji: "🎉", count: 28 },
            { emoji: "😊", count: 15 },
            { emoji: "🚀", count: 12 },
            { emoji: "👎", count: 8 },
            { emoji: "😕", count: 5 },
            { emoji: "🤔", count: 3 },
        ] as const;
        
        distribution.forEach(({ emoji, count }) => {
            for (let i = 0; i < count; i++) {
                reactions.push(factory.createMockReaction({
                    emoji,
                    by: factory["createMockUser"](`user_${emoji}_${i}`),
                }));
            }
        });
        
        return reactions;
    },
    
    // MSW handlers
    successHandlers: () => new ReactionMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new ReactionMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new ReactionMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new ReactionMSWHandlers().createNetworkErrorHandlers(),
    optimisticHandlers: () => new ReactionMSWHandlers().createOptimisticHandlers(),
};

// Export factory instances for easy use
export const reactionResponseFactory = new ReactionResponseFactory();
export const reactionMSWHandlers = new ReactionMSWHandlers();
