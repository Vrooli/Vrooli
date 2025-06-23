import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface ReactionRelationConfig extends RelationConfig {
    userId?: string;
    targetType?: "resource" | "chatMessage" | "comment" | "issue";
    targetId?: string;
}

/**
 * Enhanced database fixture factory for Reaction model
 * Provides comprehensive testing capabilities for polymorphic reactions
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for polymorphic relationships (resource, chatMessage, comment, issue)
 * - Emoji validation and testing
 * - User association management
 * - Predefined test scenarios
 * - Reaction summary synchronization testing
 */
export class ReactionDbFactory extends EnhancedDatabaseFactory<
    Prisma.reactionCreateInput,
    Prisma.reactionCreateInput,
    Prisma.reactionInclude,
    Prisma.reactionUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("reaction", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.reaction;
    }

    /**
     * Get complete test fixtures for Reaction model
     */
    protected getFixtures(): DbTestFixtures<Prisma.reactionCreateInput, Prisma.reactionUpdateInput> {
        return {
            minimal: {
                id: generatePK().toString(),
                emoji: "👍",
                byId: generatePK().toString(),
                // Must have at least one target
                resourceId: generatePK().toString(),
            },
            complete: {
                id: generatePK().toString(),
                emoji: "❤️",
                byId: generatePK().toString(),
                resourceId: generatePK().toString(),
                createdAt: new Date(),
                updatedAt: new Date(),
            },
            invalid: {
                missingRequired: {
                    // Missing id, emoji, byId, and any target
                },
                invalidTypes: {
                    id: "not-a-snowflake",
                    emoji: 123, // Should be string
                    byId: null, // Should be string
                    resourceId: true, // Should be string
                },
                noTarget: {
                    id: generatePK().toString(),
                    emoji: "👍",
                    byId: generatePK().toString(),
                    // No target specified (resourceId, chatMessageId, commentId, or issueId)
                },
                multipleTargets: {
                    id: generatePK().toString(),
                    emoji: "👍",
                    byId: generatePK().toString(),
                    resourceId: generatePK().toString(),
                    chatMessageId: generatePK().toString(), // Multiple targets not allowed
                },
                invalidEmoji: {
                    id: generatePK().toString(),
                    emoji: "not_an_emoji", // Invalid emoji format
                    byId: generatePK().toString(),
                    resourceId: generatePK().toString(),
                },
            },
            edgeCases: {
                unicodeEmoji: {
                    id: generatePK().toString(),
                    emoji: "🏳️‍🌈", // Complex unicode emoji
                    byId: generatePK().toString(),
                    resourceId: generatePK().toString(),
                },
                textEmoji: {
                    id: generatePK().toString(),
                    emoji: ":heart:", // Text representation
                    byId: generatePK().toString(),
                    resourceId: generatePK().toString(),
                },
                multipleEmojis: {
                    id: generatePK().toString(),
                    emoji: "👍❤️🎉", // Multiple emojis
                    byId: generatePK().toString(),
                    resourceId: generatePK().toString(),
                },
                reactionToComment: {
                    id: generatePK().toString(),
                    emoji: "💬",
                    byId: generatePK().toString(),
                    commentId: generatePK().toString(),
                },
                reactionToIssue: {
                    id: generatePK().toString(),
                    emoji: "🐛",
                    byId: generatePK().toString(),
                    issueId: generatePK().toString(),
                },
                reactionToChatMessage: {
                    id: generatePK().toString(),
                    emoji: "💬",
                    byId: generatePK().toString(),
                    chatMessageId: generatePK().toString(),
                },
            },
            updates: {
                minimal: {
                    emoji: "👎", // Change reaction
                },
                complete: {
                    emoji: "🎉",
                    updatedAt: new Date(),
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.reactionCreateInput>): Prisma.reactionCreateInput {
        return {
            id: generatePK().toString(),
            emoji: "👍",
            byId: generatePK().toString(),
            resourceId: generatePK().toString(), // Default to resource
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.reactionCreateInput>): Prisma.reactionCreateInput {
        return {
            id: generatePK().toString(),
            emoji: "❤️",
            byId: generatePK().toString(),
            resourceId: generatePK().toString(),
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
            likeReaction: {
                name: "likeReaction",
                description: "Simple like reaction to a resource",
                config: {
                    overrides: {
                        emoji: "👍",
                    },
                    targetType: "resource",
                },
            },
            loveReaction: {
                name: "loveReaction",
                description: "Love reaction to content",
                config: {
                    overrides: {
                        emoji: "❤️",
                    },
                    targetType: "resource",
                },
            },
            issueReaction: {
                name: "issueReaction",
                description: "Reaction to an issue",
                config: {
                    overrides: {
                        emoji: "🐛",
                    },
                    targetType: "issue",
                },
            },
            commentReaction: {
                name: "commentReaction",
                description: "Reaction to a comment",
                config: {
                    overrides: {
                        emoji: "💬",
                    },
                    targetType: "comment",
                },
            },
            chatReaction: {
                name: "chatReaction",
                description: "Reaction to a chat message",
                config: {
                    overrides: {
                        emoji: "👋",
                    },
                    targetType: "chatMessage",
                },
            },
            multipleReactions: {
                name: "multipleReactions",
                description: "Multiple reactions from same user to different targets",
                config: {
                    overrides: {
                        emoji: "🎉",
                    },
                },
            },
        };
    }

    protected getDefaultInclude(): Prisma.reactionInclude {
        return {
            by: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                },
            },
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
        baseData: Prisma.reactionCreateInput,
        config: ReactionRelationConfig,
        tx: any,
    ): Promise<Prisma.reactionCreateInput> {
        const data = { ...baseData };

        // Handle user association
        if (config.userId) {
            data.byId = config.userId;
        }

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
     * Create a reaction to a specific target type
     */
    async createReactionTo(
        targetType: "resource" | "chatMessage" | "comment" | "issue",
        targetId: string,
        userId: string,
        emoji = "👍",
    ): Promise<Prisma.reaction> {
        return await this.createWithRelations({
            overrides: { emoji },
            userId,
            targetType,
            targetId,
        });
    }

    /**
     * Create multiple reactions to the same target
     */
    async createMultipleReactionsToTarget(
        targetType: "resource" | "chatMessage" | "comment" | "issue",
        targetId: string,
        reactions: Array<{ userId: string; emoji: string }>,
    ): Promise<Prisma.reaction[]> {
        const createdReactions: Prisma.reaction[] = [];

        for (const { userId, emoji } of reactions) {
            const reaction = await this.createReactionTo(targetType, targetId, userId, emoji);
            createdReactions.push(reaction);
        }

        return createdReactions;
    }

    /**
     * Create common reaction patterns
     */
    async createLikeReaction(targetId: string, userId: string): Promise<Prisma.reaction> {
        return await this.createReactionTo("resource", targetId, userId, "👍");
    }

    async createLoveReaction(targetId: string, userId: string): Promise<Prisma.reaction> {
        return await this.createReactionTo("resource", targetId, userId, "❤️");
    }

    async createCelebrationReaction(targetId: string, userId: string): Promise<Prisma.reaction> {
        return await this.createReactionTo("resource", targetId, userId, "🎉");
    }

    protected async checkModelConstraints(record: Prisma.reaction): Promise<string[]> {
        const violations: string[] = [];
        
        // Check that only one target is specified
        const targetCount = [
            record.resourceId,
            record.chatMessageId,
            record.commentId,
            record.issueId,
        ].filter(Boolean).length;

        if (targetCount === 0) {
            violations.push("Reaction must have exactly one target");
        } else if (targetCount > 1) {
            violations.push("Reaction cannot have multiple targets");
        }

        // Check emoji format (basic validation)
        if (!record.emoji || record.emoji.length === 0) {
            violations.push("Emoji cannot be empty");
        }

        if (record.emoji && record.emoji.length > 32) {
            violations.push("Emoji exceeds maximum length of 32 characters");
        }

        // Check user exists
        const user = await this.prisma.user.findUnique({
            where: { id: record.byId },
        });
        if (!user) {
            violations.push("User does not exist");
        }

        // Check uniqueness (user can only have one reaction per target)
        const existingReaction = await this.prisma.reaction.findFirst({
            where: {
                byId: record.byId,
                resourceId: record.resourceId,
                chatMessageId: record.chatMessageId,
                commentId: record.commentId,
                issueId: record.issueId,
                id: { not: record.id },
            },
        });
        if (existingReaction) {
            violations.push("User already has a reaction to this target");
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            // Reactions don't have dependent records to cascade delete
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.reaction,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[],
    ): Promise<void> {
        // Reactions don't have dependent records
        // The reaction summary should be updated when reactions are deleted
        // but that's handled by triggers or application logic
    }

    /**
     * Get reaction statistics for a target
     */
    async getReactionStats(
        targetType: "resource" | "chatMessage" | "comment" | "issue",
        targetId: string,
    ): Promise<Record<string, number>> {
        const where: any = {};
        where[`${targetType}Id`] = targetId;

        const reactions = await this.prisma.reaction.findMany({
            where,
            select: { emoji: true },
        });

        const stats: Record<string, number> = {};
        for (const reaction of reactions) {
            stats[reaction.emoji] = (stats[reaction.emoji] || 0) + 1;
        }

        return stats;
    }

    /**
     * Create a set of diverse reactions for testing
     */
    async createDiverseReactions(
        targetType: "resource" | "chatMessage" | "comment" | "issue",
        targetId: string,
        userIds: string[],
    ): Promise<Prisma.reaction[]> {
        const emojis = ["👍", "❤️", "🎉", "🔥", "👏", "😊", "🚀", "💯"];
        const reactions: Array<{ userId: string; emoji: string }> = [];

        // Distribute emojis across users
        userIds.forEach((userId, index) => {
            reactions.push({
                userId,
                emoji: emojis[index % emojis.length],
            });
        });

        return await this.createMultipleReactionsToTarget(targetType, targetId, reactions);
    }
}

// Export factory creator function
export const createReactionDbFactory = (prisma: PrismaClient) => 
    ReactionDbFactory.getInstance("reaction", prisma);

// Export the class for type usage
export { ReactionDbFactory as ReactionDbFactoryClass };
