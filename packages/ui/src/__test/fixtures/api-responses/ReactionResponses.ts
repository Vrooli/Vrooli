/**
 * Reaction API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for reaction endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { rest, type RestHandler } from "msw";
import type { 
    Reaction, 
    ReactInput,
    ReactionSearchInput,
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
        return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Generate unique resource ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
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
                hasPreviousPage: paginationData.page > 1,
                startCursor: reactions[0]?.id || null,
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
        const now = new Date().toISOString();
        
        return {
            __typename: "User",
            id: userId,
            handle: `user_${userId.slice(-6)}`,
            name: `Test User ${userId.slice(-4)}`,
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
                    emoji: null,
                    count: 0,
                },
            },
        };
    }
    
    /**
     * Create mock reaction target based on type
     */
    private createMockReactionTarget(type: ReactionFor): ChatMessage | Comment | Issue | Resource {
        const id = `${type.toLowerCase()}_${this.generateId()}`;
        const now = new Date().toISOString();
        
        switch (type) {
            case ReactionForEnum.ChatMessage:
                return {
                    __typename: "ChatMessage",
                    id,
                    created_at: now,
                    updated_at: now,
                    chat: {
                        __typename: "Chat",
                        id: `chat_${this.generateId()}`,
                        created_at: now,
                        updated_at: now,
                        inviteCode: null,
                        createdBy: this.createMockUser(),
                        messages: [],
                        messagesCount: 1,
                        participants: [],
                        participantsCount: 1,
                        team: null,
                        labels: [],
                        labelsCount: 0,
                        you: {
                            __typename: "ChatYou",
                            canDelete: false,
                            canInvite: false,
                            canUpdate: false,
                        },
                    },
                    you: {
                        __typename: "ChatMessageYou",
                        canDelete: false,
                        canReply: true,
                        canUpdate: false,
                        canReport: true,
                        isBookmarked: false,
                        isReacted: false,
                        reaction: null,
                    },
                } as ChatMessage;
                
            case ReactionForEnum.Comment:
                return {
                    __typename: "Comment",
                    id,
                    created_at: now,
                    updated_at: now,
                    createdBy: this.createMockUser(),
                    isDeleted: false,
                    reports: [],
                    reportsCount: 0,
                    you: {
                        __typename: "CommentYou",
                        canDelete: false,
                        canUpdate: false,
                        canReport: true,
                        isBookmarked: false,
                        isReacted: false,
                        reaction: null,
                    },
                } as Comment;
                
            case ReactionForEnum.Issue:
                return {
                    __typename: "Issue",
                    id,
                    created_at: now,
                    updated_at: now,
                    closedAt: null,
                    closedBy: null,
                    createdBy: this.createMockUser(),
                    isPrivate: false,
                    issueType: "Bug",
                    status: "Open",
                    labels: [],
                    labelsCount: 0,
                    comments: [],
                    commentsCount: 0,
                    reportsCount: 0,
                    references: [],
                    referencedByCount: 0,
                    translations: [],
                    translationsCount: 0,
                    bookmarkedBy: [],
                    bookmarkedByCount: 0,
                    you: {
                        __typename: "IssueYou",
                        canComment: true,
                        canDelete: false,
                        canUpdate: false,
                        canReport: true,
                        canBookmark: true,
                        canRead: true,
                        canReact: true,
                        isBookmarked: false,
                        isReacted: false,
                        reaction: null,
                    },
                } as Issue;
                
            case ReactionForEnum.Resource:
                return {
                    __typename: "Resource",
                    id,
                    createdAt: now,
                    updatedAt: now,
                    isInternal: false,
                    isPrivate: false,
                    usedBy: [],
                    usedByCount: 0,
                    versions: [],
                    versionsCount: 0,
                    you: {
                        __typename: "ResourceYou",
                        canDelete: false,
                        canUpdate: false,
                        canReport: true,
                        isBookmarked: false,
                        isReacted: false,
                        reaction: null,
                    },
                } as Resource;
                
            default:
                throw new Error(`Unknown reaction type: ${type}`);
        }
    }
    
    /**
     * Create mock reaction data
     */
    createMockReaction(overrides?: Partial<Reaction>): Reaction {
        const now = new Date().toISOString();
        const id = this.generateId();
        const emoji = this.getRandomEmoji("positive");
        
        const defaultReaction: Reaction = {
            __typename: "Reaction",
            id,
            createdAt: now,
            updatedAt: now,
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
    createMultipleReactions(targetType: ReactionFor, targetId: string, count = 5): Reaction[] {
        const target = this.createMockReactionTarget(targetType);
        target.id = targetId;
        
        return Array.from({ length: count }, (_, index) => {
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
                return b.count - a.count;
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
        
        if (input.emoji && input.emoji.length > 10) {
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
        
        const userMap = this.userReactions.get(userId)!;
        if (emoji === null) {
            userMap.delete(objectId);
        } else {
            userMap.set(objectId, emoji);
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
    createSuccessHandlers(): RestHandler[] {
        return [
            // React to object (add/update/remove reaction)
            rest.post(`${this.responseFactory["baseUrl"]}/api/reaction`, async (req, res, ctx) => {
                const body = await req.json() as ReactInput;
                const userId = "user_test"; // In real tests, extract from auth header
                
                // Validate input
                const validation = await this.responseFactory.validateReactInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                // Track the reaction
                this.trackUserReaction(userId, body.forConnect, body.emoji || null);
                
                // Return success
                const response = this.responseFactory.createSuccessResponse(true);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Search reactions
            rest.get(`${this.responseFactory["baseUrl"]}/api/reaction`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const chatMessageId = url.searchParams.get("chatMessageId");
                const commentId = url.searchParams.get("commentId");
                const issueId = url.searchParams.get("issueId");
                const resourceId = url.searchParams.get("resourceId");
                
                let reactions: Reaction[] = [];
                
                // Generate reactions based on query parameters
                if (chatMessageId) {
                    reactions = this.responseFactory.createMultipleReactions(
                        ReactionForEnum.ChatMessage, 
                        chatMessageId, 
                        15,
                    );
                } else if (commentId) {
                    reactions = this.responseFactory.createMultipleReactions(
                        ReactionForEnum.Comment, 
                        commentId, 
                        10,
                    );
                } else if (issueId) {
                    reactions = this.responseFactory.createMultipleReactions(
                        ReactionForEnum.Issue, 
                        issueId, 
                        8,
                    );
                } else if (resourceId) {
                    reactions = this.responseFactory.createMultipleReactions(
                        ReactionForEnum.Resource, 
                        resourceId, 
                        5,
                    );
                } else {
                    // Return mixed reactions
                    reactions = [
                        ...this.responseFactory.createMultipleReactions(ReactionForEnum.Comment, "comment_1", 3),
                        ...this.responseFactory.createMultipleReactions(ReactionForEnum.Issue, "issue_1", 2),
                        ...this.responseFactory.createMultipleReactions(ReactionForEnum.Resource, "resource_1", 2),
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
            rest.post(`${this.responseFactory["baseUrl"]}/api/reaction`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        forConnect: "Target object ID is required",
                        reactionFor: "Reaction type must be specified",
                    })),
                );
            }),
            
            // Rate limit error
            rest.post(`${this.responseFactory["baseUrl"]}/api/reaction`, (req, res, ctx) => {
                return res(
                    ctx.status(429),
                    ctx.json(this.responseFactory.createRateLimitErrorResponse()),
                );
            }),
            
            // Permission error
            rest.post(`${this.responseFactory["baseUrl"]}/api/reaction`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse("create")),
                );
            }),
            
            // Server error
            rest.post(`${this.responseFactory["baseUrl"]}/api/reaction`, (req, res, ctx) => {
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
            rest.post(`${this.responseFactory["baseUrl"]}/api/reaction`, async (req, res, ctx) => {
                const body = await req.json() as ReactInput;
                const validation = await this.responseFactory.validateReactInput(body);
                
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                const response = this.responseFactory.createSuccessResponse(true);
                
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
            rest.post(`${this.responseFactory["baseUrl"]}/api/reaction`, (req, res, ctx) => {
                return res.networkError("Network connection failed");
            }),
            
            rest.get(`${this.responseFactory["baseUrl"]}/api/reaction`, (req, res, ctx) => {
                return res.networkError("Connection timeout");
            }),
        ];
    }
    
    /**
     * Create optimistic update handlers (simulating immediate UI updates)
     */
    createOptimisticHandlers(): RestHandler[] {
        return [
            rest.post(`${this.responseFactory["baseUrl"]}/api/reaction`, async (req, res, ctx) => {
                const body = await req.json() as ReactInput;
                
                // Simulate very fast response for optimistic UI updates
                const response = this.responseFactory.createSuccessResponse(true);
                
                return res(
                    ctx.delay(50), // Very small delay
                    ctx.status(200),
                    ctx.json(response),
                );
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
export const reactionResponseScenarios = {
    // Success scenarios
    reactSuccess: (success = true) => {
        const factory = new ReactionResponseFactory();
        return factory.createSuccessResponse(success);
    },
    
    searchSuccess: (reactions?: Reaction[]) => {
        const factory = new ReactionResponseFactory();
        return factory.createReactionSearchResponse(
            reactions || factory.createMultipleReactions(ReactionForEnum.Comment, "comment_123", 10),
        );
    },
    
    reactionSummary: (targetType: ReactionFor = ReactionForEnum.Comment, count = 20) => {
        const factory = new ReactionResponseFactory();
        const reactions = factory.createMultipleReactions(targetType, "target_123", count);
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
        ];
        
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
