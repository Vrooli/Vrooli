import { generatePK } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for Reaction model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const reactionDbIds = {
    reaction1: generatePK(),
    reaction2: generatePK(),
    reaction3: generatePK(),
    reaction4: generatePK(),
    reaction5: generatePK(),
    reaction6: generatePK(),
};

/**
 * Minimal reaction data for database creation
 */
export const minimalReactionDb: Prisma.reactionCreateInput = {
    id: reactionDbIds.reaction1,
    emoji: "👍",
    by: { connect: { id: generatePK() } },
};

/**
 * Complete reaction with all fields
 */
export const completeReactionDb: Prisma.reactionCreateInput = {
    id: reactionDbIds.reaction2,
    emoji: "❤️",
    by: { connect: { id: generatePK() } },
    createdAt: new Date(),
    updatedAt: new Date(),
};

/**
 * Factory for creating reaction database fixtures with overrides
 */
export class ReactionDbFactory {
    /**
     * Create a minimal reaction
     */
    static createMinimal(
        byId: string,
        overrides?: Partial<Prisma.reactionCreateInput>,
    ): Prisma.reactionCreateInput {
        return {
            id: generatePK(),
            emoji: "👍",
            by: { connect: { id: byId } },
            ...overrides,
        };
    }

    /**
     * Create a reaction for a specific object type
     */
    static createForObject(
        byId: string,
        objectId: string,
        objectType: "resource" | "chatMessage" | "comment" | "issue",
        emoji = "👍",
        overrides?: Partial<Prisma.reactionCreateInput>,
    ): Prisma.reactionCreateInput {
        const baseReaction = this.createMinimal(byId, overrides);
        
        // Add the appropriate connection based on object type
        const connections: Record<string, any> = {
            resource: { resource: { connect: { id: objectId } } },
            chatMessage: { chatMessage: { connect: { id: objectId } } },
            comment: { comment: { connect: { id: objectId } } },
            issue: { issue: { connect: { id: objectId } } },
        };

        return {
            ...baseReaction,
            emoji,
            ...(connections[objectType] || {}),
        };
    }

    /**
     * Create multiple reactions with different emojis for the same object
     */
    static createMultipleForObject(
        byIds: string[],
        objectId: string,
        objectType: "resource" | "chatMessage" | "comment" | "issue",
        emojis: string[],
    ): Prisma.reactionCreateInput[] {
        return byIds.map((byId, index) => 
            this.createForObject(
                byId,
                objectId,
                objectType,
                emojis[index % emojis.length],
            ),
        );
    }

    /**
     * Create common reaction patterns
     */
    static createCommonReactions(
        objectId: string,
        objectType: "resource" | "chatMessage" | "comment" | "issue",
        pattern: "liked" | "loved" | "mixed" | "controversial",
    ): Prisma.reactionCreateInput[] {
        const patterns = {
            liked: { emojis: ["👍", "👍", "👍"], count: 3 },
            loved: { emojis: ["❤️", "❤️", "😍"], count: 3 },
            mixed: { emojis: ["👍", "❤️", "😊", "🤔"], count: 4 },
            controversial: { emojis: ["👍", "👎", "👍", "👎", "😕"], count: 5 },
        };

        const { emojis, count } = patterns[pattern];
        const byIds = Array.from({ length: count }, () => generatePK());
        
        return this.createMultipleForObject(byIds, objectId, objectType, emojis);
    }
}

/**
 * Helper to create reaction summary data based on reactions
 */
export class ReactionSummaryDbFactory {
    static createFromReactions(
        reactions: Prisma.reactionCreateInput[],
        objectId: string,
        objectType: "resource" | "chatMessage" | "comment" | "issue",
    ): Prisma.reaction_summaryCreateInput[] {
        // Group reactions by emoji
        const emojiCounts = reactions.reduce((acc, reaction) => {
            const emoji = reaction.emoji as string;
            acc[emoji] = (acc[emoji] || 0) + 1;
            return acc;
        }, {} as Record<string, number>);

        // Create summary for each emoji
        return Object.entries(emojiCounts).map(([emoji, count]) => {
            const baseSummary: Prisma.reaction_summaryCreateInput = {
                id: generatePK(),
                emoji,
                count,
            };

            // Add the appropriate connection
            const connections: Record<string, any> = {
                resource: { resource: { connect: { id: objectId } } },
                chatMessage: { chatMessage: { connect: { id: objectId } } },
                comment: { comment: { connect: { id: objectId } } },
                issue: { issue: { connect: { id: objectId } } },
            };

            return {
                ...baseSummary,
                ...(connections[objectType] || {}),
            };
        });
    }
}

/**
 * Helper to seed reactions for testing
 */
export async function seedReactions(
    prisma: any,
    options: {
        byUserIds: string[];
        objectId: string;
        objectType: "resource" | "chatMessage" | "comment" | "issue";
        pattern?: "liked" | "loved" | "mixed" | "controversial";
        customEmojis?: string[];
    },
) {
    const reactions = [];
    
    if (options.pattern) {
        // Use predefined pattern
        const patternReactions = ReactionDbFactory.createCommonReactions(
            options.objectId,
            options.objectType,
            options.pattern,
        );
        
        // Update with actual user IDs
        for (let i = 0; i < patternReactions.length && i < options.byUserIds.length; i++) {
            patternReactions[i].by = { connect: { id: options.byUserIds[i] } };
            reactions.push(patternReactions[i]);
        }
    } else if (options.customEmojis) {
        // Use custom emojis
        const customReactions = ReactionDbFactory.createMultipleForObject(
            options.byUserIds,
            options.objectId,
            options.objectType,
            options.customEmojis,
        );
        reactions.push(...customReactions);
    } else {
        // Default single thumbs up from first user
        reactions.push(
            ReactionDbFactory.createForObject(
                options.byUserIds[0],
                options.objectId,
                options.objectType,
            ),
        );
    }

    // Create reactions
    const createdReactions = [];
    for (const reactionData of reactions) {
        const created = await prisma.reaction.create({ data: reactionData });
        createdReactions.push(created);
    }

    // Create reaction summaries
    const summaries = ReactionSummaryDbFactory.createFromReactions(
        reactions,
        options.objectId,
        options.objectType,
    );

    for (const summaryData of summaries) {
        await prisma.reaction_summary.create({ data: summaryData });
    }

    return { reactions: createdReactions, summaries };
}

/**
 * Helper to clean up reactions and summaries
 */
export async function cleanupReactions(
    prisma: any,
    objectId: string,
    objectType: "resource" | "chatMessage" | "comment" | "issue",
) {
    const whereClause = {
        [`${objectType}Id`]: objectId,
    };

    // Delete reactions and summaries (summaries should cascade delete)
    await prisma.reaction.deleteMany({ where: whereClause });
    await prisma.reaction_summary.deleteMany({ where: whereClause });
}
