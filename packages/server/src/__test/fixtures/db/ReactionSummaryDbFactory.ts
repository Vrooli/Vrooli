/* eslint-disable no-magic-numbers */
import { type Prisma, type PrismaClient, type reaction_summary } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type {
    DbTestFixtures,
    RelationConfig,
    TestScenario,
} from "./types.js";

interface ReactionSummaryRelationConfig extends RelationConfig {
    targetType?: "resource" | "chatMessage" | "comment" | "issue";
    targetId?: bigint;
    reactionCounts?: Array<{ emoji: string; count: number }>;
}

/**
 * Enhanced database fixture factory for ReactionSummary model
 * Provides comprehensive testing capabilities for reaction aggregations
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for polymorphic relationships (resource, chatMessage, comment, issue)
 * - Emoji count management
 * - Synchronization testing with reactions
 * - Predefined test scenarios
 * - Edge case handling for counts
 */
export class ReactionSummaryDbFactory extends EnhancedDatabaseFactory<
    reaction_summary,
    Prisma.reaction_summaryCreateInput,
    Prisma.reaction_summaryInclude,
    Prisma.reaction_summaryUpdateInput
> {
    protected scenarios: Record<string, TestScenario> = {};
    constructor(prisma: PrismaClient) {
        super("reaction_summary", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.reaction_summary;
    }

    /**
     * Get complete test fixtures for ReactionSummary model
     */
    protected getFixtures(): DbTestFixtures<Prisma.reaction_summaryCreateInput, Prisma.reaction_summaryUpdateInput> {
        return {
            minimal: {
                id: this.generateId(),
                emoji: "👍",
                count: 1,
                // Must have at least one target
                resourceId: this.generateId(),
            },
            complete: {
                id: this.generateId(),
                emoji: "❤️",
                count: 42,
                resourceId: this.generateId(),
                createdAt: new Date(),
                updatedAt: new Date(),
            },
            invalid: {
                missingRequired: {
                    // Missing id, emoji, count, and any target
                },
                invalidTypes: {
                    id: 123 as any, // Should be bigint/string, not number
                    emoji: 123, // Should be string
                    count: "not-a-number", // Should be number
                    resourceId: true, // Should be string
                },
                noTarget: {
                    id: this.generateId(),
                    emoji: "👍",
                    count: 1,
                    // No target specified
                },
                multipleTargets: {
                    id: this.generateId(),
                    emoji: "👍",
                    count: 1,
                    resourceId: this.generateId(),
                    chatMessageId: this.generateId(), // Multiple targets not allowed
                },
                negativeCount: {
                    id: this.generateId(),
                    emoji: "👍",
                    count: -1, // Count should not be negative
                    resourceId: this.generateId(),
                },
                invalidEmoji: {
                    id: this.generateId(),
                    emoji: "not_an_emoji", // Invalid emoji format
                    count: 1,
                    resourceId: this.generateId(),
                },
            },
            edgeCases: {
                zeroCount: {
                    id: this.generateId(),
                    emoji: "👍",
                    count: 0, // Zero reactions
                    resourceId: this.generateId(),
                },
                largeCount: {
                    id: this.generateId(),
                    emoji: "❤️",
                    count: 999999, // Very large count
                    resourceId: this.generateId(),
                },
                unicodeEmoji: {
                    id: this.generateId(),
                    emoji: "🏳️‍🌈", // Complex unicode emoji
                    count: 10,
                    resourceId: this.generateId(),
                },
                summaryForComment: {
                    id: this.generateId(),
                    emoji: "💬",
                    count: 5,
                    commentId: this.generateId(),
                },
                summaryForIssue: {
                    id: this.generateId(),
                    emoji: "🐛",
                    count: 3,
                    issueId: this.generateId(),
                },
                summaryForChatMessage: {
                    id: this.generateId(),
                    emoji: "👋",
                    count: 7,
                    chatMessageId: this.generateId(),
                },
                multipleEmojisForSameTarget: {
                    id: this.generateId(),
                    emoji: "🎉",
                    count: 15,
                    resourceId: this.generateId(),
                },
            },
            updates: {
                minimal: {
                    count: 5, // Update count
                },
                complete: {
                    count: 100,
                    updatedAt: new Date(),
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.reaction_summaryCreateInput>): Prisma.reaction_summaryCreateInput {
        return {
            id: this.generateId(),
            emoji: "👍",
            count: 1,
            resourceId: this.generateId(), // Default to resource
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.reaction_summaryCreateInput>): Prisma.reaction_summaryCreateInput {
        return {
            id: this.generateId(),
            emoji: "❤️",
            count: 10,
            resourceId: this.generateId(),
            createdAt: new Date(),
            updatedAt: new Date(),
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            popularResource: {
                name: "popularResource",
                description: "Resource with many reactions",
                config: {
                    reactionCounts: [
                        { emoji: "👍", count: 100 },
                        { emoji: "❤️", count: 50 },
                        { emoji: "🎉", count: 25 },
                    ],
                    targetType: "resource",
                },
            },
            controversialIssue: {
                name: "controversialIssue",
                description: "Issue with mixed reactions",
                config: {
                    reactionCounts: [
                        { emoji: "👍", count: 30 },
                        { emoji: "👎", count: 25 },
                        { emoji: "😕", count: 15 },
                    ],
                    targetType: "issue",
                },
            },
            viralComment: {
                name: "viralComment",
                description: "Comment with viral reaction spread",
                config: {
                    reactionCounts: [
                        { emoji: "😂", count: 500 },
                        { emoji: "🔥", count: 200 },
                        { emoji: "💯", count: 150 },
                    ],
                    targetType: "comment",
                },
            },
            newContent: {
                name: "newContent",
                description: "Newly created content with few reactions",
                config: {
                    reactionCounts: [
                        { emoji: "👍", count: 1 },
                        { emoji: "❤️", count: 2 },
                    ],
                    targetType: "resource",
                },
            },
            diverseReactions: {
                name: "diverseReactions",
                description: "Content with diverse emoji reactions",
                config: {
                    reactionCounts: [
                        { emoji: "👍", count: 10 },
                        { emoji: "❤️", count: 8 },
                        { emoji: "🎉", count: 6 },
                        { emoji: "🚀", count: 4 },
                        { emoji: "👏", count: 3 },
                        { emoji: "😊", count: 2 },
                        { emoji: "💡", count: 1 },
                    ],
                    targetType: "resource",
                },
            },
        };
    }

    protected getDefaultInclude(): Prisma.reaction_summaryInclude {
        return {
            resource: {
                select: {
                    id: true,
                    index: true,
                },
            },
            chatMessage: {
                select: {
                    id: true,
                    publicId: true,
                },
            },
            comment: {
                select: {
                    id: true,
                    publicId: true,
                },
            },
            issue: {
                select: {
                    id: true,
                    publicId: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.reaction_summaryCreateInput,
        config: ReactionSummaryRelationConfig,
        tx: any,
    ): Promise<Prisma.reaction_summaryCreateInput> {
        const data = { ...baseData };

        // Handle target type
        if (config.targetType && config.targetId) {
            // Clear any existing targets
            delete data.resourceId;
            delete data.chatMessageId;
            delete data.commentId;
            delete data.issueId;

            // Set the appropriate target
            switch (config.targetType) {
                case "resource":
                    data.resourceId = config.targetId;
                    break;
                case "chatMessage":
                    data.chatMessageId = config.targetId;
                    break;
                case "comment":
                    data.commentId = config.targetId;
                    break;
                case "issue":
                    data.issueId = config.targetId;
                    break;
            }
        }

        return data;
    }

    /**
     * Create a reaction summary for a specific target
     */
    async createSummaryFor(
        targetType: "resource" | "chatMessage" | "comment" | "issue",
        targetId: bigint,
        emoji: string,
        count: number,
    ): Promise<Prisma.reaction_summary> {
        return await this.createWithRelations({
            overrides: { emoji, count },
            targetType,
            targetId,
        });
    }

    /**
     * Create multiple reaction summaries for a target
     */
    async createMultipleSummariesForTarget(
        targetType: "resource" | "chatMessage" | "comment" | "issue",
        targetId: bigint,
        reactionCounts: Array<{ emoji: string; count: number }>,
    ): Promise<Prisma.reaction_summary[]> {
        const summaries: Prisma.reaction_summary[] = [];

        for (const { emoji, count } of reactionCounts) {
            const summary = await this.createSummaryFor(targetType, targetId, emoji, count);
            summaries.push(summary);
        }

        return summaries;
    }

    /**
     * Create a complete reaction summary set for testing
     */
    async createCompleteSummarySet(
        targetType: "resource" | "chatMessage" | "comment" | "issue",
        targetId: string,
    ): Promise<Prisma.reaction_summary[]> {
        const defaultCounts = [
            { emoji: "👍", count: 25 },
            { emoji: "❤️", count: 15 },
            { emoji: "🎉", count: 10 },
            { emoji: "🔥", count: 5 },
            { emoji: "👏", count: 3 },
        ];

        return await this.createMultipleSummariesForTarget(targetType, targetId, defaultCounts);
    }

    /**
     * Update reaction count (increment/decrement)
     */
    async updateReactionCount(
        summaryId: string,
        delta: number,
    ): Promise<Prisma.reaction_summary> {
        const summary = await this.prisma.reaction_summary.findUnique({
            where: { id: summaryId },
        });

        if (!summary) {
            throw new Error("Reaction summary not found");
        }

        return await this.prisma.reaction_summary.update({
            where: { id: summaryId },
            data: { count: Math.max(0, summary.count + delta) },
            include: this.getDefaultInclude(),
        });
    }

    protected async checkModelConstraints(record: Prisma.reaction_summary): Promise<string[]> {
        const violations: string[] = [];

        // Check that only one target is specified
        const targetCount = [
            record.resourceId,
            record.chatMessageId,
            record.commentId,
            record.issueId,
        ].filter(Boolean).length;

        if (targetCount === 0) {
            violations.push("Reaction summary must have exactly one target");
        } else if (targetCount > 1) {
            violations.push("Reaction summary cannot have multiple targets");
        }

        // Check emoji format
        if (!record.emoji || record.emoji.length === 0) {
            violations.push("Emoji cannot be empty");
        }

        if (record.emoji && record.emoji.length > 32) {
            violations.push("Emoji exceeds maximum length of 32 characters");
        }

        // Check count is non-negative
        if (record.count < 0) {
            violations.push("Count cannot be negative");
        }

        // Check uniqueness (one summary per emoji per target)
        const existingSummary = await this.prisma.reaction_summary.findFirst({
            where: {
                emoji: record.emoji,
                resourceId: record.resourceId,
                chatMessageId: record.chatMessageId,
                commentId: record.commentId,
                issueId: record.issueId,
                id: { not: record.id },
            },
        });
        if (existingSummary) {
            violations.push("Reaction summary already exists for this emoji and target");
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            // Reaction summaries don't have dependent records to cascade delete
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.reaction_summary,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[],
    ): Promise<void> {
        // Reaction summaries don't have dependent records
    }

    /**
     * Synchronize summary with actual reaction counts
     */
    async synchronizeWithReactions(
        targetType: "resource" | "chatMessage" | "comment" | "issue",
        targetId: string,
        emoji: string,
    ): Promise<Prisma.reaction_summary | null> {
        const where: any = {};
        where[`${targetType}Id`] = targetId;
        where.emoji = emoji;

        // Count actual reactions
        const actualCount = await this.prisma.reaction.count({ where });

        // Find existing summary
        const existingSummary = await this.prisma.reaction_summary.findFirst({ where });

        if (actualCount === 0 && existingSummary) {
            // Delete summary if no reactions exist
            await this.prisma.reaction_summary.delete({
                where: { id: existingSummary.id },
            });
            return null;
        } else if (actualCount > 0) {
            if (existingSummary) {
                // Update existing summary
                return await this.prisma.reaction_summary.update({
                    where: { id: existingSummary.id },
                    data: { count: actualCount },
                    include: this.getDefaultInclude(),
                });
            } else {
                // Create new summary
                const data: any = {
                    id: this.generateId(),
                    emoji,
                    count: actualCount,
                };
                data[`${targetType}Id`] = targetId;

                return await this.prisma.reaction_summary.create({
                    data,
                    include: this.getDefaultInclude(),
                });
            }
        }

        return null;
    }

    /**
     * Get total reaction count for a target
     */
    async getTotalReactionCount(
        targetType: "resource" | "chatMessage" | "comment" | "issue",
        targetId: string,
    ): Promise<number> {
        const where: any = {};
        where[`${targetType}Id`] = targetId;

        const summaries = await this.prisma.reaction_summary.findMany({
            where,
            select: { count: true },
        });

        return summaries.reduce((total, summary) => total + summary.count, 0);
    }
}

// Export factory creator function
export const createReactionSummaryDbFactory = (prisma: PrismaClient) =>
    new ReactionSummaryDbFactory(prisma);

// Export the class for type usage
export { ReactionSummaryDbFactory as ReactionSummaryDbFactoryClass };
