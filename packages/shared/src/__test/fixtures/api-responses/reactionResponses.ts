/* c8 ignore start */
/**
 * Reaction API Response Fixtures
 * 
 * Comprehensive fixtures for reaction/emoji endpoints including
 * user engagement, sentiment tracking, and emotional feedback systems.
 */

import type {
    ReactInput,
    Reaction,
    ReactionFor,
    ReactionSearchResult,
    ReactionSummary,
    Success,
} from "../../../api/types.js";
import { generatePK } from "../../../id/index.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";
import { userResponseFactory } from "./userResponses.js";
import {
    DEFAULT_COUNT,
    DEFAULT_DELAY_MS,
    DEFAULT_ERROR_RATE,
    ONE_HOUR_MS,
    ONE_THOUSAND,
    SEVEN_DAYS_MS,
} from "../constants.js";

// Reaction specific constants
const MAX_REACTIONS_PER_USER = 1; // One reaction per user per object
const RATE_LIMIT_PER_HOUR = ONE_THOUSAND;

// Common emoji reactions used in tests
const COMMON_REACTIONS = {
    positive: ["👍", "❤️", "🎉", "🚀", "😊", "👏", "🔥", "💯", "⭐", "✨"],
    negative: ["👎", "😕", "😡", "🤮", "💔", "😞"],
    neutral: ["🤔", "😐", "🤷", "👀", "📌", "💭", "🔔", "❓"],
};

// Reaction target types
const REACTION_FOR_TYPES: ReactionFor[] = ["ChatMessage", "Comment", "Issue", "Post", "Project", "Question", "Resource", "Routine", "Standard"];

/**
 * Reaction API response factory
 */
export class ReactionResponseFactory extends BaseAPIResponseFactory<
    Reaction,
    ReactInput,
    never // Reactions don't have update input, they're created or deleted
> {
    protected readonly entityName = "reaction";

    /**
     * Create mock reaction data
     */
    createMockData(options?: MockDataOptions): Reaction {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const reactionId = options?.overrides?.id || generatePK().toString();

        const baseReaction: Reaction = {
            __typename: "Reaction",
            id: reactionId,
            createdAt: now,
            updatedAt: now,
            emoji: this.getRandomEmoji("positive"),
            by: userResponseFactory.createMockData(),
            to: this.createMockReactionTarget("Comment"),
        };

        if (scenario === "complete" || scenario === "edge-case") {
            const targetType = scenario === "edge-case" ? "Issue" : "Post";

            return {
                ...baseReaction,
                emoji: scenario === "edge-case" ? this.getRandomEmoji("negative") : this.getRandomEmoji("positive"),
                by: userResponseFactory.createMockData({ scenario: "complete" }),
                to: this.createMockReactionTarget(targetType as ReactionFor),
                ...options?.overrides,
            };
        }

        return {
            ...baseReaction,
            ...options?.overrides,
        };
    }

    /**
     * Create reaction from input
     */
    createFromInput(input: ReactInput): Reaction {
        const now = new Date().toISOString();
        const reactionId = generatePK().toString();

        return {
            __typename: "Reaction",
            id: reactionId,
            createdAt: now,
            updatedAt: now,
            emoji: input.emoji || this.getRandomEmoji("positive"),
            by: userResponseFactory.createMockData(), // Current user reacting
            to: this.createMockReactionTarget(input.reactionFor, input.forConnect),
        };
    }

    /**
     * Validate create input (react input)
     */
    async validateCreateInput(input: ReactInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.forConnect) {
            errors.forConnect = "Target object ID is required";
        }

        if (!input.reactionFor) {
            errors.reactionFor = "Reaction type must be specified";
        } else if (!REACTION_FOR_TYPES.includes(input.reactionFor)) {
            errors.reactionFor = `Invalid reaction type. Must be one of: ${REACTION_FOR_TYPES.join(", ")}`;
        }

        if (input.emoji) {
            if (input.emoji.length > 10) {
                errors.emoji = "Emoji must be 10 characters or less";
            }
            // Could add emoji validation here (check if it's a valid emoji)
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create multiple reactions for the same object
     */
    createMultipleReactions(targetType: ReactionFor, targetId: string, count = 5): Reaction[] {
        const target = this.createMockReactionTarget(targetType, targetId);

        return Array.from({ length: count }, (_, index) => {
            const categories: Array<keyof typeof COMMON_REACTIONS> = ["positive", "negative", "neutral"];
            const category = categories[index % categories.length];

            return this.createMockData({
                overrides: {
                    id: `reaction_${targetId}_${index}`,
                    emoji: this.getRandomEmoji(category),
                    by: userResponseFactory.createMockData({ overrides: { id: `user_${index}` } }),
                    to: target,
                },
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
                // Sort by count (most popular first)
                return b.count - a.count;
            });
    }

    /**
     * Create reaction search result
     */
    createReactionSearchResult(reactions: Reaction[], pagination?: {
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
     * Create trending reactions (popular emojis)
     */
    createTrendingReactions(): Reaction[] {
        const trendingEmojis = ["👍", "❤️", "🎉", "🔥", "💯"];

        return trendingEmojis.flatMap((emoji, emojiIndex) =>
            Array.from({ length: 20 - (emojiIndex * 3) }, (_, userIndex) =>
                this.createMockData({
                    overrides: {
                        id: `trending_${emoji}_${userIndex}`,
                        emoji,
                        by: userResponseFactory.createMockData({ overrides: { id: `user_${emoji}_${userIndex}` } }),
                        to: this.createMockReactionTarget("Post", `trending_post_${emojiIndex}`),
                    },
                }),
            ),
        );
    }

    /**
     * Create reactions by user
     */
    createReactionsByUser(userId: string, count = 15): Reaction[] {
        const user = userResponseFactory.createMockData({ overrides: { id: userId } });

        return Array.from({ length: count }, (_, index) => {
            const targetType = REACTION_FOR_TYPES[index % REACTION_FOR_TYPES.length];
            const category: keyof typeof COMMON_REACTIONS = index % 3 === 0 ? "positive" : index % 3 === 1 ? "neutral" : "negative";

            return this.createMockData({
                overrides: {
                    id: `user_reaction_${userId}_${index}`,
                    emoji: this.getRandomEmoji(category),
                    by: user,
                    to: this.createMockReactionTarget(targetType, `target_${index}`),
                    createdAt: new Date(Date.now().toISOString() - (index * ONE_HOUR_MS)).toISOString(), // Spread over time
                },
            });
        });
    }

    /**
     * Create emoji distribution analysis
     */
    createEmojiDistribution(): { emoji: string; count: number; percentage: number }[] {
        const distribution = [
            { emoji: "👍", count: 450 },
            { emoji: "❤️", count: 320 },
            { emoji: "🎉", count: 280 },
            { emoji: "😊", count: 150 },
            { emoji: "🚀", count: 120 },
            { emoji: "🔥", count: 100 },
            { emoji: "👎", count: 80 },
            { emoji: "😕", count: 50 },
            { emoji: "🤔", count: 30 },
        ];

        const total = distribution.reduce((sum, item) => sum + item.count, 0);

        return distribution.map(item => ({
            ...item,
            percentage: Math.round((item.count / total) * 100),
        }));
    }

    /**
     * Create success response for reaction operations
     */
    createReactSuccessResponse(success = true): typeof this.createSuccessResponse {
        return this.createSuccessResponse({
            __typename: "Success",
            success,
        } as Success);
    }

    /**
     * Create duplicate reaction error response
     */
    createDuplicateReactionErrorResponse(objectId: string, objectType: string) {
        return this.createBusinessErrorResponse("duplicate", {
            resource: "reaction",
            objectId,
            objectType,
            existingReaction: true,
            message: `You have already reacted to this ${objectType}`,
        });
    }

    /**
     * Create rate limit error response
     */
    createRateLimitErrorResponse(limit = RATE_LIMIT_PER_HOUR, window = "1 hour") {
        return this.createBusinessErrorResponse("rate_limit", {
            resource: "reaction",
            limit,
            window,
            retryAfter: 3600, // 1 hour in seconds
            message: `Too many reaction requests. Limit: ${limit} per ${window}`,
        });
    }

    /**
     * Create invalid emoji error response
     */
    createInvalidEmojiErrorResponse(emoji: string) {
        return this.createValidationErrorResponse({
            emoji: `Invalid emoji: ${emoji}. Only standard emojis are allowed.`,
        });
    }

    /**
     * Get random emoji from category
     */
    private getRandomEmoji(category: keyof typeof COMMON_REACTIONS = "positive"): string {
        const emojis = COMMON_REACTIONS[category];
        return emojis[Math.floor(Math.random() * emojis.length)];
    }

    /**
     * Create mock reaction target
     */
    private createMockReactionTarget(type: ReactionFor, targetId?: string): any {
        const id = targetId || `${type.toLowerCase()}_${generatePK().toString()}`;
        const now = new Date().toISOString();

        // Return a minimal object that represents the reaction target
        // In real implementation, this would come from the appropriate factory
        return {
            __typename: type,
            id,
            createdAt: now,
            updatedAt: now,
            // Add minimal required fields based on type
            ...(type === "ChatMessage" && { content: "Sample message" }),
            ...(type === "Comment" && { text: "Sample comment" }),
            ...(type === "Issue" && { title: "Sample issue", status: "Open" }),
            ...(type === "Post" && { title: "Sample post" }),
            ...(type === "Project" && { name: "Sample project" }),
            ...(type === "Question" && { title: "Sample question" }),
            ...(type === "Resource" && { name: "Sample resource" }),
            ...(type === "Routine" && { name: "Sample routine" }),
            ...(type === "Standard" && { name: "Sample standard" }),
        };
    }
}

/**
 * Pre-configured reaction response scenarios
 */
export const reactionResponseScenarios = {
    // Success scenarios
    reactSuccess: (success = true) => {
        const factory = new ReactionResponseFactory();
        return factory.createReactSuccessResponse(success);
    },

    createSuccess: (input?: Partial<ReactInput>) => {
        const factory = new ReactionResponseFactory();
        const defaultInput: ReactInput = {
            forConnect: generatePK().toString(),
            reactionFor: "Comment",
            emoji: "👍",
            ...input,
        };
        return factory.createReactSuccessResponse(true);
    },

    searchSuccess: (reactions?: Reaction[]) => {
        const factory = new ReactionResponseFactory();
        const searchResult = factory.createReactionSearchResult(
            reactions || factory.createMultipleReactions("Comment", generatePK().toString(), 10),
        );
        return factory.createSuccessResponse(searchResult);
    },

    listSuccess: (reactions?: Reaction[]) => {
        const factory = new ReactionResponseFactory();
        return factory.createPaginatedResponse(
            reactions || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: reactions?.length || DEFAULT_COUNT },
        );
    },

    reactionSummarySuccess: (targetType: ReactionFor = "Comment", count = 20) => {
        const factory = new ReactionResponseFactory();
        const reactions = factory.createMultipleReactions(targetType, generatePK().toString(), count);
        const summary = factory.createReactionSummary(reactions);
        return factory.createSuccessResponse(summary);
    },

    trendingReactionsSuccess: () => {
        const factory = new ReactionResponseFactory();
        const reactions = factory.createTrendingReactions();
        return factory.createPaginatedResponse(
            reactions,
            { page: 1, totalCount: reactions.length },
        );
    },

    userReactionsSuccess: (userId?: string) => {
        const factory = new ReactionResponseFactory();
        const reactions = factory.createReactionsByUser(userId || generatePK().toString());
        return factory.createPaginatedResponse(
            reactions,
            { page: 1, totalCount: reactions.length },
        );
    },

    multipleReactionsSuccess: (targetType: ReactionFor, targetId?: string, count = 15) => {
        const factory = new ReactionResponseFactory();
        const reactions = factory.createMultipleReactions(targetType, targetId || generatePK().toString(), count);
        return factory.createPaginatedResponse(
            reactions,
            { page: 1, totalCount: reactions.length },
        );
    },

    emojiDistributionSuccess: () => {
        const factory = new ReactionResponseFactory();
        const distribution = factory.createEmojiDistribution();
        return factory.createSuccessResponse(distribution);
    },

    reactionAnalyticsSuccess: (objectId?: string) => {
        const factory = new ReactionResponseFactory();
        const reactions = factory.createMultipleReactions("Post", objectId || generatePK().toString(), 50);
        const summary = factory.createReactionSummary(reactions);
        const distribution = factory.createEmojiDistribution();

        return factory.createSuccessResponse({
            objectId: objectId || generatePK().toString(),
            totalReactions: reactions.length,
            uniqueUsers: reactions.length, // Assuming one reaction per user
            summary,
            distribution,
            timeRange: {
                start: new Date(Date.now().toISOString() - SEVEN_DAYS_MS).toISOString(), // 7 days ago
                end: new Date().toISOString(),
            },
        });
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new ReactionResponseFactory();
        return factory.createValidationErrorResponse({
            forConnect: "Target object ID is required",
            reactionFor: "Reaction type must be specified",
        });
    },

    duplicateReactionError: (objectId?: string, objectType = "comment") => {
        const factory = new ReactionResponseFactory();
        return factory.createDuplicateReactionErrorResponse(
            objectId || generatePK().toString(),
            objectType,
        );
    },

    rateLimitError: (limit?: number, window?: string) => {
        const factory = new ReactionResponseFactory();
        return factory.createRateLimitErrorResponse(limit, window);
    },

    permissionError: (operation?: string) => {
        const factory = new ReactionResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
            ["reaction:write"],
        );
    },

    invalidEmojiError: (emoji = "invalid_emoji") => {
        const factory = new ReactionResponseFactory();
        return factory.createInvalidEmojiErrorResponse(emoji);
    },

    invalidTargetError: () => {
        const factory = new ReactionResponseFactory();
        return factory.createValidationErrorResponse({
            forConnect: "Target object does not exist or is not reactable",
        });
    },

    invalidReactionTypeError: () => {
        const factory = new ReactionResponseFactory();
        return factory.createValidationErrorResponse({
            reactionFor: `Invalid reaction type. Must be one of: ${REACTION_FOR_TYPES.join(", ")}`,
        });
    },

    // Special scenarios
    removeReactionSuccess: () => {
        const factory = new ReactionResponseFactory();
        return factory.createReactSuccessResponse(true);
    },

    optimisticUpdateSuccess: () => {
        const factory = new ReactionResponseFactory();
        // Simulate very fast response for optimistic UI updates
        return factory.createReactSuccessResponse(true);
    },

    // MSW handlers
    handlers: {
        success: () => new ReactionResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new ReactionResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new ReactionResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
        optimistic: function createOptimistic() {
            return new ReactionResponseFactory().createMSWHandlers({ delay: 50 }); // Very fast for optimistic updates
        },
    },
};

// Export factory instance for direct use
export const reactionResponseFactory = new ReactionResponseFactory();
