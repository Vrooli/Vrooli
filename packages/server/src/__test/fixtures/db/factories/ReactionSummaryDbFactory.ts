import { generatePK } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface ReactionSummaryRelationConfig extends RelationConfig {
    objectType?: "resource" | "chatMessage" | "comment" | "issue";
    objectId?: string;
    reactionData?: Array<{ emoji: string; count: number }>;
}

/**
 * Database fixture factory for ReactionSummary model
 * Handles aggregated reaction counts for performance optimization
 */
export class ReactionSummaryDbFactory extends DatabaseFixtureFactory<
    Prisma.reaction_summary,
    Prisma.reaction_summaryCreateInput,
    Prisma.reaction_summaryInclude,
    Prisma.reaction_summaryUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("reaction_summary", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.reaction_summary;
    }

    protected getMinimalData(overrides?: Partial<Prisma.reaction_summaryCreateInput>): Prisma.reaction_summaryCreateInput {
        return {
            id: generatePK(),
            emoji: "👍",
            count: 1,
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.reaction_summaryCreateInput>): Prisma.reaction_summaryCreateInput {
        return {
            id: generatePK(),
            emoji: "❤️",
            count: 10,
            ...overrides,
        };
    }

    /**
     * Create a reaction summary for a specific object
     */
    async createForObject(
        objectType: "resource" | "chatMessage" | "comment" | "issue",
        objectId: string,
        emoji: string,
        count: number,
        overrides?: Partial<Prisma.reaction_summaryCreateInput>,
    ): Promise<Prisma.reaction_summary> {
        const data: Prisma.reaction_summaryCreateInput = {
            ...this.getMinimalData(),
            emoji,
            count,
            [objectType]: { connect: { id: objectId } },
            ...overrides,
        };

        const result = await this.prisma.reaction_summary.create({ data });
        this.trackCreatedId(result.id);
        return result;
    }

    /**
     * Create multiple summaries from reaction data
     */
    async createFromReactionData(
        objectType: "resource" | "chatMessage" | "comment" | "issue",
        objectId: string,
        reactionData: Array<{ emoji: string; count: number }>,
    ): Promise<Prisma.reaction_summary[]> {
        const summaries: Prisma.reaction_summary[] = [];

        for (const { emoji, count } of reactionData) {
            const summary = await this.createForObject(
                objectType,
                objectId,
                emoji,
                count,
            );
            summaries.push(summary);
        }

        return summaries;
    }

    /**
     * Create summaries from actual reactions
     */
    async createFromReactions(
        objectType: "resource" | "chatMessage" | "comment" | "issue",
        objectId: string,
    ): Promise<Prisma.reaction_summary[]> {
        // Get all reactions for this object
        const whereClause = { [`${objectType}Id`]: objectId };
        const reactions = await this.prisma.reaction.findMany({
            where: whereClause,
            select: { emoji: true },
        });

        // Group by emoji and count
        const emojiCounts = reactions.reduce((acc, reaction) => {
            acc[reaction.emoji] = (acc[reaction.emoji] || 0) + 1;
            return acc;
        }, {} as Record<string, number>);

        // Create summaries
        const reactionData = Object.entries(emojiCounts).map(([emoji, count]) => ({
            emoji,
            count,
        }));

        return this.createFromReactionData(objectType, objectId, reactionData);
    }

    /**
     * Update or create summary (upsert)
     */
    async upsertForObject(
        objectType: "resource" | "chatMessage" | "comment" | "issue",
        objectId: string,
        emoji: string,
        delta: number,
    ): Promise<Prisma.reaction_summary> {
        const whereClause = {
            [`${objectType}Id`]: objectId,
            emoji,
        };

        const existing = await this.prisma.reaction_summary.findFirst({
            where: whereClause,
        });

        if (existing) {
            const updated = await this.prisma.reaction_summary.update({
                where: { id: existing.id },
                data: { count: Math.max(0, existing.count + delta) },
            });
            return updated;
        } else if (delta > 0) {
            return this.createForObject(objectType, objectId, emoji, delta);
        } else {
            // Don't create summary with 0 or negative count
            throw new Error("Cannot create reaction summary with non-positive count");
        }
    }

    protected getDefaultInclude(): Prisma.reaction_summaryInclude {
        return {
            // Include polymorphic relationships
            resource: { select: { id: true, publicId: true } },
            chatMessage: { select: { id: true } },
            comment: { select: { id: true } },
            issue: { select: { id: true, publicId: true } },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.reaction_summaryCreateInput,
        config: ReactionSummaryRelationConfig,
        tx: any,
    ): Promise<Prisma.reaction_summaryCreateInput> {
        const data = { ...baseData };

        // Handle object connection
        if (config.objectType && config.objectId) {
            data[config.objectType] = { connect: { id: config.objectId } };
        }

        // Handle reaction data
        if (config.reactionData && config.reactionData.length > 0) {
            // Only use first reaction data for single summary
            const { emoji, count } = config.reactionData[0];
            data.emoji = emoji;
            data.count = count;
        }

        return data;
    }

    /**
     * Create test scenarios
     */
    async createPopularContent(
        objectType: "resource" | "chatMessage" | "comment" | "issue",
        objectId: string,
    ): Promise<Prisma.reaction_summary[]> {
        const popularReactions = [
            { emoji: "👍", count: 150 },
            { emoji: "❤️", count: 89 },
            { emoji: "🎉", count: 45 },
            { emoji: "💯", count: 32 },
            { emoji: "🔥", count: 28 },
        ];

        return this.createFromReactionData(objectType, objectId, popularReactions);
    }

    async createControversialContent(
        objectType: "resource" | "chatMessage" | "comment" | "issue",
        objectId: string,
    ): Promise<Prisma.reaction_summary[]> {
        const controversialReactions = [
            { emoji: "👍", count: 120 },
            { emoji: "👎", count: 115 },
            { emoji: "😕", count: 45 },
            { emoji: "🤔", count: 38 },
            { emoji: "😡", count: 22 },
        ];

        return this.createFromReactionData(objectType, objectId, controversialReactions);
    }

    async createDiverseReactions(
        objectType: "resource" | "chatMessage" | "comment" | "issue",
        objectId: string,
    ): Promise<Prisma.reaction_summary[]> {
        const diverseReactions = [
            { emoji: "👍", count: 25 },
            { emoji: "❤️", count: 18 },
            { emoji: "😂", count: 15 },
            { emoji: "😮", count: 12 },
            { emoji: "🤯", count: 10 },
            { emoji: "🙏", count: 8 },
            { emoji: "💡", count: 7 },
            { emoji: "🚀", count: 6 },
            { emoji: "✨", count: 5 },
            { emoji: "🌟", count: 4 },
        ];

        return this.createFromReactionData(objectType, objectId, diverseReactions);
    }

    async createMinimalEngagement(
        objectType: "resource" | "chatMessage" | "comment" | "issue",
        objectId: string,
    ): Promise<Prisma.reaction_summary[]> {
        const minimalReactions = [
            { emoji: "👍", count: 1 },
        ];

        return this.createFromReactionData(objectType, objectId, minimalReactions);
    }

    protected async checkModelConstraints(record: Prisma.reaction_summary): Promise<string[]> {
        const violations: string[] = [];

        // Check exactly one parent object
        const parentFields = ["resourceId", "chatMessageId", "commentId", "issueId"];
        const connectedObjects = parentFields.filter(field => 
            record[field as keyof Prisma.reaction_summary],
        );
        
        if (connectedObjects.length === 0) {
            violations.push("ReactionSummary must reference exactly one object");
        } else if (connectedObjects.length > 1) {
            violations.push("ReactionSummary cannot reference multiple objects");
        }

        // Check emoji is valid
        if (!record.emoji || record.emoji.length === 0) {
            violations.push("ReactionSummary must have an emoji");
        }

        // Check count is valid
        if (record.count < 0) {
            violations.push("ReactionSummary count cannot be negative");
        }

        // Check for duplicate summaries (same object and emoji)
        if (connectedObjects.length === 1) {
            const fieldName = connectedObjects[0];
            const objectId = record[fieldName as keyof Prisma.reaction_summary];
            
            const duplicate = await this.prisma.reaction_summary.findFirst({
                where: {
                    [fieldName]: objectId,
                    emoji: record.emoji,
                    id: { not: record.id },
                },
            });
            
            if (duplicate) {
                violations.push("ReactionSummary already exists for this object and emoji");
            }
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing emoji and count
                id: generatePK(),
            },
            invalidTypes: {
                id: "not-a-snowflake",
                emoji: 123, // Should be string
                count: "not-a-number", // Should be number
            },
            noParentObject: {
                id: generatePK(),
                emoji: "👍",
                count: 10,
                // No object connected
            },
            multipleObjects: {
                id: generatePK(),
                emoji: "👍",
                count: 10,
                resource: { connect: { id: generatePK() } },
                comment: { connect: { id: generatePK() } },
                // Multiple objects (invalid)
            },
            negativeCount: {
                id: generatePK(),
                emoji: "👍",
                count: -5, // Negative count
                resource: { connect: { id: generatePK() } },
            },
            emptyEmoji: {
                id: generatePK(),
                emoji: "", // Empty emoji
                count: 10,
                resource: { connect: { id: generatePK() } },
            },
            duplicateSummary: {
                id: generatePK(),
                emoji: "👍",
                count: 10,
                resource: { connect: { id: "existing_resource_id" } }, // Assumes this exists
                // This would be a duplicate if a summary already exists for 👍
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.reaction_summaryCreateInput> {
        const resourceId = generatePK();
        
        return {
            zeroCount: {
                ...this.getMinimalData(),
                resource: { connect: { id: resourceId } },
                count: 0, // Edge case: zero reactions
            },
            veryHighCount: {
                ...this.getMinimalData(),
                resource: { connect: { id: resourceId } },
                count: 999999, // Very high engagement
            },
            complexEmoji: {
                ...this.getMinimalData(),
                resource: { connect: { id: resourceId } },
                emoji: "👨‍👩‍👧‍👦", // Multi-codepoint emoji
                count: 5,
            },
            textEmoji: {
                ...this.getMinimalData(),
                resource: { connect: { id: resourceId } },
                emoji: ":)", // Text-based emoji
                count: 3,
            },
            customEmoji: {
                ...this.getMinimalData(),
                resource: { connect: { id: resourceId } },
                emoji: ":custom_reaction:", // Custom emoji format
                count: 7,
            },
            singleReaction: {
                ...this.getMinimalData(),
                resource: { connect: { id: resourceId } },
                emoji: "💎", // Rare emoji with single reaction
                count: 1,
            },
        };
    }

    /**
     * Synchronize summaries with actual reactions
     */
    async synchronizeWithReactions(
        objectType: "resource" | "chatMessage" | "comment" | "issue",
        objectId: string,
    ): Promise<{
        created: Prisma.reaction_summary[];
        updated: Prisma.reaction_summary[];
        deleted: string[];
    }> {
        const created: Prisma.reaction_summary[] = [];
        const updated: Prisma.reaction_summary[] = [];
        const deleted: string[] = [];

        // Get current summaries
        const whereClause = { [`${objectType}Id`]: objectId };
        const currentSummaries = await this.prisma.reaction_summary.findMany({
            where: whereClause,
        });

        // Get actual reaction counts
        const reactions = await this.prisma.reaction.findMany({
            where: whereClause,
            select: { emoji: true },
        });

        const actualCounts = reactions.reduce((acc, reaction) => {
            acc[reaction.emoji] = (acc[reaction.emoji] || 0) + 1;
            return acc;
        }, {} as Record<string, number>);

        // Update or create summaries
        for (const [emoji, count] of Object.entries(actualCounts)) {
            const existing = currentSummaries.find(s => s.emoji === emoji);
            
            if (existing) {
                if (existing.count !== count) {
                    const updatedSummary = await this.prisma.reaction_summary.update({
                        where: { id: existing.id },
                        data: { count },
                    });
                    updated.push(updatedSummary);
                }
            } else {
                const newSummary = await this.createForObject(objectType, objectId, emoji, count);
                created.push(newSummary);
            }
        }

        // Delete summaries for emojis that no longer have reactions
        for (const summary of currentSummaries) {
            if (!actualCounts[summary.emoji]) {
                await this.prisma.reaction_summary.delete({
                    where: { id: summary.id },
                });
                deleted.push(summary.id);
            }
        }

        return { created, updated, deleted };
    }

    protected getCascadeInclude(): any {
        return {
            // ReactionSummaries don't have child records
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.reaction_summary,
        remainingDepth: number,
        tx: any,
    ): Promise<void> {
        // ReactionSummaries don't have child records to delete
    }
}

// Export factory creator function
export const createReactionSummaryDbFactory = (prisma: PrismaClient) => new ReactionSummaryDbFactory(prisma);
