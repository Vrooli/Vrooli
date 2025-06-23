import { generatePK } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface ReactionRelationConfig extends RelationConfig {
    objectType?: "resource" | "chatMessage" | "comment" | "issue";
    objectId?: string;
    emoji?: string;
}

/**
 * Database fixture factory for Reaction model
 * Handles likes, dislikes, and custom emoji reactions on various objects
 */
export class ReactionDbFactory extends DatabaseFixtureFactory<
    Prisma.reaction,
    Prisma.reactionCreateInput,
    Prisma.reactionInclude,
    Prisma.reactionUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("reaction", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.reaction;
    }

    protected getMinimalData(overrides?: Partial<Prisma.reactionCreateInput>): Prisma.reactionCreateInput {
        return {
            id: generatePK(),
            emoji: "👍",
            by: { connect: { id: generatePK() } }, // Will be overridden by relationship config
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.reactionCreateInput>): Prisma.reactionCreateInput {
        return {
            id: generatePK(),
            emoji: "❤️",
            by: { connect: { id: generatePK() } }, // Will be overridden by relationship config
            createdAt: new Date(),
            updatedAt: new Date(),
            ...overrides,
        };
    }

    /**
     * Create a reaction for a specific object
     */
    async createForObject(
        byId: string,
        objectType: "resource" | "chatMessage" | "comment" | "issue",
        objectId: string,
        emoji = "👍",
        overrides?: Partial<Prisma.reactionCreateInput>,
    ): Promise<Prisma.reaction> {
        const data: Prisma.reactionCreateInput = {
            ...this.getMinimalData(),
            emoji,
            by: { connect: { id: byId } },
            [objectType]: { connect: { id: objectId } },
            ...overrides,
        };

        const result = await this.prisma.reaction.create({ data });
        this.trackCreatedId(result.id);
        return result;
    }

    /**
     * Create multiple reactions with different emojis
     */
    async createMultiple(
        byIds: string[],
        objectType: "resource" | "chatMessage" | "comment" | "issue",
        objectId: string,
        emojis?: string[],
    ): Promise<Prisma.reaction[]> {
        const defaultEmojis = ["👍", "❤️", "😄", "🎉", "🤔"];
        const emojiList = emojis || defaultEmojis;
        const reactions: Prisma.reaction[] = [];

        for (let i = 0; i < byIds.length; i++) {
            const reaction = await this.createForObject(
                byIds[i],
                objectType,
                objectId,
                emojiList[i % emojiList.length],
            );
            reactions.push(reaction);
        }

        return reactions;
    }

    /**
     * Create common reaction patterns
     */
    async createReactionPattern(
        objectType: "resource" | "chatMessage" | "comment" | "issue",
        objectId: string,
        pattern: "liked" | "loved" | "mixed" | "controversial" | "viral",
        userIds: string[],
    ): Promise<Prisma.reaction[]> {
        const patterns = {
            liked: { emojis: ["👍", "👍", "👍"], minUsers: 3 },
            loved: { emojis: ["❤️", "❤️", "😍", "💕"], minUsers: 4 },
            mixed: { emojis: ["👍", "❤️", "😊", "🤔", "👏"], minUsers: 5 },
            controversial: { emojis: ["👍", "👎", "👍", "👎", "😕", "🤨"], minUsers: 6 },
            viral: { emojis: ["🔥", "💯", "🚀", "⭐", "✨", "🎉", "🙌", "💪"], minUsers: 8 },
        };

        const { emojis, minUsers } = patterns[pattern];
        const usersToUse = userIds.slice(0, Math.max(minUsers, emojis.length));

        return this.createMultiple(usersToUse, objectType, objectId, emojis);
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
            // Include polymorphic relationships
            resource: { select: { id: true, publicId: true } },
            chatMessage: { select: { id: true } },
            comment: { select: { id: true } },
            issue: { select: { id: true, publicId: true } },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.reactionCreateInput,
        config: ReactionRelationConfig,
        tx: any,
    ): Promise<Prisma.reactionCreateInput> {
        const data = { ...baseData };

        // Ensure user is set
        if (!data.by && config.byId) {
            data.by = { connect: { id: config.byId } };
        }

        // Handle object connection
        if (config.objectType && config.objectId) {
            data[config.objectType] = { connect: { id: config.objectId } };
        }

        // Handle emoji
        if (config.emoji) {
            data.emoji = config.emoji;
        }

        return data;
    }

    /**
     * Create test scenarios
     */
    async createLikeDislikeBattle(
        objectType: "resource" | "chatMessage" | "comment" | "issue",
        objectId: string,
        likerIds: string[],
        dislikerIds: string[],
    ): Promise<{
        likes: Prisma.reaction[];
        dislikes: Prisma.reaction[];
    }> {
        const likes = await this.createMultiple(likerIds, objectType, objectId, ["👍"]);
        const dislikes = await this.createMultiple(dislikerIds, objectType, objectId, ["👎"]);
        
        return { likes, dislikes };
    }

    async createEmojiSpectrum(
        objectType: "resource" | "chatMessage" | "comment" | "issue",
        objectId: string,
        userId: string,
    ): Promise<Prisma.reaction[]> {
        // Create reactions with a wide variety of emojis
        const diverseEmojis = [
            "😀", "😃", "😄", "😁", "😆", "😅", "🤣", "😂", "🙂", "🙃",
            "😉", "😊", "😇", "🥰", "😍", "🤩", "😘", "😗", "☺️", "😚",
            "😙", "😋", "😛", "😜", "🤪", "😝", "🤑", "🤗", "🤭", "🤫",
            "🤔", "🤐", "🤨", "😐", "😑", "😶", "😏", "😒", "🙄", "😬",
        ];
        
        const userIds = Array(diverseEmojis.length).fill(userId);
        return this.createMultiple(userIds, objectType, objectId, diverseEmojis);
    }

    async createTimeBasedReactions(
        objectType: "resource" | "chatMessage" | "comment" | "issue",
        objectId: string,
        userIds: string[],
        hoursBack = 24,
    ): Promise<Prisma.reaction[]> {
        const reactions: Prisma.reaction[] = [];
        const now = new Date();
        
        for (let i = 0; i < userIds.length; i++) {
            const hoursAgo = (hoursBack / userIds.length) * i;
            const createdAt = new Date(now.getTime() - (hoursAgo * 60 * 60 * 1000));
            
            const reaction = await this.createForObject(
                userIds[i],
                objectType,
                objectId,
                ["👍", "❤️", "🎉", "💯"][i % 4],
                {
                    createdAt,
                    updatedAt: createdAt,
                },
            );
            reactions.push(reaction);
        }
        
        return reactions;
    }

    protected async checkModelConstraints(record: Prisma.reaction): Promise<string[]> {
        const violations: string[] = [];
        
        // Check user exists
        if (record.byId) {
            const user = await this.prisma.user.findUnique({
                where: { id: record.byId },
            });
            if (!user) {
                violations.push("Reaction user must exist");
            }
        }

        // Check exactly one reacted object
        const reactableFields = ["resourceId", "chatMessageId", "commentId", "issueId"];
        const connectedObjects = reactableFields.filter(field => 
            record[field as keyof Prisma.reaction],
        );
        
        if (connectedObjects.length === 0) {
            violations.push("Reaction must reference exactly one object");
        } else if (connectedObjects.length > 1) {
            violations.push("Reaction cannot reference multiple objects");
        }

        // Check emoji is valid
        if (!record.emoji || record.emoji.length === 0) {
            violations.push("Reaction must have an emoji");
        }

        // Check for duplicate reactions (same user, object, emoji)
        if (record.byId && connectedObjects.length === 1) {
            const fieldName = connectedObjects[0];
            const objectId = record[fieldName as keyof Prisma.reaction];
            
            const duplicate = await this.prisma.reaction.findFirst({
                where: {
                    byId: record.byId,
                    [fieldName]: objectId,
                    emoji: record.emoji,
                    id: { not: record.id },
                },
            });
            
            if (duplicate) {
                violations.push("User has already reacted with this emoji to this object");
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
                // Missing emoji and by
                id: generatePK(),
            },
            invalidTypes: {
                id: "not-a-snowflake",
                emoji: 123, // Should be string
                by: "invalid-user-reference", // Should be connect object
            },
            noReactedObject: {
                id: generatePK(),
                emoji: "👍",
                by: { connect: { id: generatePK() } },
                // No object connected
            },
            multipleObjects: {
                id: generatePK(),
                emoji: "👍",
                by: { connect: { id: generatePK() } },
                resource: { connect: { id: generatePK() } },
                comment: { connect: { id: generatePK() } },
                // Multiple objects (invalid)
            },
            emptyEmoji: {
                id: generatePK(),
                emoji: "", // Empty emoji
                by: { connect: { id: generatePK() } },
                resource: { connect: { id: generatePK() } },
            },
            duplicateReaction: {
                id: generatePK(),
                emoji: "👍",
                by: { connect: { id: "existing_user_id" } }, // Assumes this exists
                resource: { connect: { id: "existing_resource_id" } }, // Assumes this exists
                // This would be a duplicate if the user already reacted with 👍
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.reactionCreateInput> {
        const userId = generatePK();
        const resourceId = generatePK();
        
        return {
            singleCharEmoji: {
                ...this.getMinimalData(),
                by: { connect: { id: userId } },
                resource: { connect: { id: resourceId } },
                emoji: "👍",
            },
            multiCharEmoji: {
                ...this.getMinimalData(),
                by: { connect: { id: userId } },
                resource: { connect: { id: resourceId } },
                emoji: "👨‍👩‍👧‍👦", // Family emoji (multiple codepoints)
            },
            textEmoji: {
                ...this.getMinimalData(),
                by: { connect: { id: userId } },
                resource: { connect: { id: resourceId } },
                emoji: ":)", // Text-based emoji
            },
            customEmoji: {
                ...this.getMinimalData(),
                by: { connect: { id: userId } },
                resource: { connect: { id: resourceId } },
                emoji: ":custom_emoji:", // Custom emoji format
            },
            flagEmoji: {
                ...this.getMinimalData(),
                by: { connect: { id: userId } },
                resource: { connect: { id: resourceId } },
                emoji: "🇺🇸", // Flag emoji (two codepoints)
            },
            skinToneEmoji: {
                ...this.getMinimalData(),
                by: { connect: { id: userId } },
                resource: { connect: { id: resourceId } },
                emoji: "👍🏽", // Thumbs up with skin tone
            },
            rareEmoji: {
                ...this.getMinimalData(),
                by: { connect: { id: userId } },
                resource: { connect: { id: resourceId } },
                emoji: "🧿", // Nazar amulet (less common)
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            // Reactions don't have child records
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.reaction,
        remainingDepth: number,
        tx: any,
    ): Promise<void> {
        // Reactions don't have child records to delete
        // ReactionSummary should be updated separately when reactions change
    }
}

// Export factory creator function
export const createReactionDbFactory = (prisma: PrismaClient) => new ReactionDbFactory(prisma);
