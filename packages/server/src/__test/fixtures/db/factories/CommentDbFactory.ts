import { generatePK, generatePublicId, nanoid } from "../idHelpers.js";

// TODO: Update this when CommentFor is available
const CommentFor = {
    Issue: "Issue",
    PullRequest: "PullRequest", 
    ResourceVersion: "ResourceVersion",
} as const;

type CommentFor = typeof CommentFor[keyof typeof CommentFor];
import { type comment, type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface CommentRelationConfig extends RelationConfig {
    withReplies?: boolean | number;
    parentId?: string;
    withReactions?: boolean | number;
    reactionEmojis?: string[];
    withBookmarks?: boolean;
    translations?: Array<{ language: string; text: string }>;
    ownedByTeam?: boolean;
    teamId?: string;
}

/**
 * Database fixture factory for Comment model
 * Handles polymorphic comments on any object with threaded replies and reactions
 */
export class CommentDbFactory extends DatabaseFixtureFactory<
    comment,
    Prisma.commentCreateInput,
    Prisma.commentInclude,
    Prisma.commentUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("comment", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.comment;
    }

    protected getMinimalData(overrides?: Partial<Prisma.commentCreateInput>): Prisma.commentCreateInput {
        return {
            id: generatePK(),
            score: 0,
            bookmarks: 0,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "Test comment",
                }],
            },
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.commentCreateInput>): Prisma.commentCreateInput {
        return {
            id: generatePK(),
            score: 5,
            bookmarks: 3,
            ownedByUser: { connect: { id: generatePK() } }, // Will be overridden by relationship config
            translations: {
                create: [
                    {
                        id: generatePK(),
                        language: "en",
                        text: "This is a comprehensive test comment with detailed content that provides meaningful feedback and insights.",
                    },
                    {
                        id: generatePK(),
                        language: "es",
                        text: "Este es un comentario de prueba completo con contenido detallado que proporciona retroalimentación e ideas significativas.",
                    },
                ],
            },
            ...overrides,
        };
    }

    /**
     * Create a comment for a specific object type
     */
    async createForObject(
        objectType: CommentFor,
        objectId: string | bigint,
        ownerId: string | bigint,
        overrides?: Partial<Prisma.commentCreateInput>,
    ): Promise<comment> {
        const typeMapping: Record<CommentFor, string> = {
            [CommentFor.Issue]: "issue",
            [CommentFor.PullRequest]: "pullRequest",
            [CommentFor.ResourceVersion]: "resourceVersion",
        };
        
        const fieldName = typeMapping[objectType];
        if (!fieldName) {
            throw new Error(`Invalid comment object type: ${objectType}`);
        }
        
        const data: Prisma.commentCreateInput = {
            ...this.getMinimalData(),
            ownedByUser: { connect: { id: BigInt(ownerId) } },
            [fieldName]: { connect: { id: BigInt(objectId) } },
            ...overrides,
        };
        
        const result = await this.prisma.comment.create({ data });
        this.trackCreatedId(result.id.toString());
        return result;
    }

    /**
     * Create a threaded reply
     */
    async createReply(
        parentId: string | bigint,
        ownerId: string | bigint,
        overrides?: Partial<Prisma.commentCreateInput>,
    ): Promise<comment> {
        const data: Prisma.commentCreateInput = {
            ...this.getMinimalData(),
            parent: { connect: { id: BigInt(parentId) } },
            ownedByUser: { connect: { id: BigInt(ownerId) } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "Reply comment",
                }],
            },
            ...overrides,
        };
        
        const result = await this.prisma.comment.create({ data });
        this.trackCreatedId(result.id.toString());
        return result;
    }

    /**
     * Create a team-owned comment
     */
    async createTeamComment(
        teamId: string | bigint,
        objectType: CommentFor,
        objectId: string | bigint,
        overrides?: Partial<Prisma.commentCreateInput>,
    ): Promise<comment> {
        const typeMapping: Record<CommentFor, string> = {
            [CommentFor.Issue]: "issue",
            [CommentFor.PullRequest]: "pullRequest",
            [CommentFor.ResourceVersion]: "resourceVersion",
        };
        
        const fieldName = typeMapping[objectType];
        const data: Prisma.commentCreateInput = {
            ...this.getMinimalData(),
            ownedByTeam: { connect: { id: BigInt(teamId) } },
            [fieldName]: { connect: { id: BigInt(objectId) } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "Official team response",
                }],
            },
            ...overrides,
        };
        
        const result = await this.prisma.comment.create({ data });
        this.trackCreatedId(result.id.toString());
        return result;
    }

    protected getDefaultInclude(): Prisma.commentInclude {
        return {
            translations: true,
            ownedByUser: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                },
            },
            ownedByTeam: {
                select: {
                    id: true,
                    publicId: true,
                    handle: true,
                },
            },
            parent: {
                select: {
                    id: true,
                    translations: {
                        where: { language: "en" },
                        take: 1,
                    },
                },
            },
            _count: {
                select: {
                    parents: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.commentCreateInput,
        config: CommentRelationConfig,
        tx: any,
    ): Promise<Prisma.commentCreateInput> {
        const data = { ...baseData };

        // Handle ownership
        if (config.ownedByTeam && config.teamId) {
            data.ownedByTeam = { connect: { id: BigInt(config.teamId) } };
            delete data.ownedByUser;
        } else if (config.ownerId) {
            data.ownedByUser = { connect: { id: BigInt(config.ownerId) } };
        }

        // Handle parent comment
        if (config.parentId) {
            data.parent = { connect: { id: BigInt(config.parentId) } };
        }

        // Handle translations
        if (config.translations && Array.isArray(config.translations)) {
            data.translations = {
                create: config.translations.map(trans => ({
                    id: generatePK(),
                    ...trans,
                })),
            };
        }

        // Handle reactions
        if (config.withReactions && config.reactionUserIds && config.reactionUserIds.length > 0) {
            const reactionCount = typeof config.withReactions === "number" 
                ? Math.min(config.withReactions, config.reactionUserIds.length)
                : config.reactionUserIds.length;
            
            const emojis = config.reactionEmojis || ["👍", "❤️", "😄", "🎉", "🤔"];
            
            data.reactions = {
                create: config.reactionUserIds.slice(0, reactionCount).map((userId, i) => ({
                    id: generatePK(),
                    emoji: emojis[i % emojis.length],
                    by: { connect: { id: BigInt(userId) } },
                })),
            };
        }

        // Handle bookmarks
        if (config.withBookmarks && config.bookmarkUserIds && config.bookmarkUserIds.length > 0) {
            data.bookmarkedBy = {
                create: config.bookmarkUserIds.map(userId => ({
                    id: generatePK(),
                    user: { connect: { id: BigInt(userId) } },
                })),
            };
        }

        return data;
    }

    /**
     * Create test scenarios
     */
    async createDiscussionThread(
        rootObjectType: CommentFor,
        rootObjectId: string | bigint,
        participants: (string | bigint)[],
        depth = 3,
    ): Promise<comment[]> {
        const comments: comment[] = [];
        
        // Create root comment
        const rootComment = await this.createForObject(
            rootObjectType,
            rootObjectId,
            participants[0],
            {
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        text: "Starting a discussion about this topic...",
                    }],
                },
            },
        );
        comments.push(rootComment);

        // Create nested replies
        let parentComment = rootComment;
        for (let i = 1; i < Math.min(depth, participants.length); i++) {
            const reply = await this.createReply(
                parentComment.id,
                participants[i],
                {
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            text: `Reply ${i}: Adding to the discussion...`,
                        }],
                    },
                },
            );
            comments.push(reply);
            parentComment = reply;
        }

        return comments;
    }

    async createControversialComment(
        objectType: CommentFor,
        objectId: string,
        authorId: string,
        reactionUserIds: string[],
    ): Promise<comment> {
        return this.createWithRelations({
            overrides: {
                score: -50, // Negative score
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        text: "This is a controversial opinion that sparked debate.",
                    }],
                },
            },
            ownerId: authorId,
            withReactions: true,
            reactionUserIds,
            reactionEmojis: ["👎", "😡", "👎", "🤔", "👍"], // Mixed reactions
        });
    }

    async createPopularComment(
        objectType: CommentFor,
        objectId: string,
        authorId: string,
        reactionUserIds: string[],
    ): Promise<comment> {
        return this.createWithRelations({
            overrides: {
                score: 150,
                bookmarks: 25,
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        text: "This insightful comment really resonated with the community!",
                    }],
                },
            },
            ownerId: authorId,
            withReactions: true,
            reactionUserIds,
            reactionEmojis: ["👍", "❤️", "🎉", "💯", "🔥"],
            withBookmarks: true,
            bookmarkUserIds: reactionUserIds.slice(0, 5),
        });
    }

    protected async checkModelConstraints(record: any): Promise<string[]> {
        const violations: string[] = [];
        
        // Check has at least one translation
        const translations = await this.prisma.comment_translation.findMany({
            where: { commentId: record.id },
        });
        
        if (translations.length === 0) {
            violations.push("Comment must have at least one translation");
        }

        // Check ownership
        if (!record.ownedByUserId && !record.ownedByTeamId) {
            violations.push("Comment must be owned by either a user or team");
        }

        if (record.ownedByUserId && record.ownedByTeamId) {
            violations.push("Comment cannot be owned by both user and team");
        }

        // Check parent object or parent comment
        const parentObjects = [
            record.issueId,
            record.pullRequestId,
            record.resourceVersionId,
        ].filter(Boolean);

        if (parentObjects.length === 0 && !record.parentId) {
            violations.push("Comment must be associated with a parent object or be a reply");
        }

        if (parentObjects.length > 1) {
            violations.push("Comment cannot be associated with multiple parent objects");
        }

        // Check self-reference
        if (record.parentId && record.parentId === record.id) {
            violations.push("Comment cannot be its own parent");
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing translations
                score: 0,
                bookmarks: 0,
            },
            invalidTypes: {
                id: "not-a-snowflake",
                score: "not-a-number",
                bookmarks: "not-a-number",
                translations: "not-an-object",
            },
            noOwner: {
                id: generatePK(),
                score: 0,
                bookmarks: 0,
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        text: "Comment with no owner",
                    }],
                },
            },
            bothOwners: {
                id: generatePK(),
                score: 0,
                bookmarks: 0,
                ownedByUser: { connect: { id: generatePK() } },
                ownedByTeam: { connect: { id: generatePK() } },
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        text: "Comment with both owners",
                    }],
                },
            },
            selfReference: {
                id: generatePK(),
                score: 0,
                bookmarks: 0,
                parent: { connect: { id: generatePK() } }, // Will be same as id
                ownedByUser: { connect: { id: generatePK() } },
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        text: "Self-referencing comment",
                    }],
                },
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.commentCreateInput> {
        const userId = generatePK();
        const issueId = generatePK();
        
        return {
            maxLengthText: {
                ...this.getMinimalData(),
                ownedByUser: { connect: { id: userId } },
                issue: { connect: { id: issueId } },
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        text: "A".repeat(32768), // Maximum VARCHAR length
                    }],
                },
            },
            negativeScore: {
                ...this.getMinimalData(),
                score: -999999,
                ownedByUser: { connect: { id: userId } },
                issue: { connect: { id: issueId } },
            },
            highEngagement: {
                ...this.getMinimalData(),
                score: 999999,
                bookmarks: 50000,
                ownedByUser: { connect: { id: userId } },
                issue: { connect: { id: issueId } },
            },
            deeplyNested: {
                ...this.getMinimalData(),
                parent: { connect: { id: generatePK() } }, // Assumes deep nesting
                ownedByUser: { connect: { id: userId } },
            },
            multiLanguage: {
                ...this.getMinimalData(),
                ownedByUser: { connect: { id: userId } },
                issue: { connect: { id: issueId } },
                translations: {
                    create: [
                        { id: generatePK(), language: "en", text: "English comment" },
                        { id: generatePK(), language: "es", text: "Comentario en español" },
                        { id: generatePK(), language: "fr", text: "Commentaire en français" },
                        { id: generatePK(), language: "de", text: "Kommentar auf Deutsch" },
                        { id: generatePK(), language: "ja", text: "日本語のコメント" },
                        { id: generatePK(), language: "zh", text: "中文评论" },
                        { id: generatePK(), language: "ar", text: "تعليق بالعربية" },
                        { id: generatePK(), language: "ru", text: "Комментарий на русском" },
                    ],
                },
            },
            unicodeContent: {
                ...this.getMinimalData(),
                ownedByUser: { connect: { id: userId } },
                issue: { connect: { id: issueId } },
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        text: "Unicode test: 🚀💬🎉 Special chars: ñáéíóú ™®© Math: ∑∏∫ Symbols: ♠♣♥♦",
                    }],
                },
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            translations: true,
            parents: true,
            reactions: true,
            bookmarkedBy: true,
        };
    }

    protected async deleteRelatedRecords(
        record: any,
        remainingDepth: number,
        tx: any,
    ): Promise<void> {
        // Delete replies recursively
        if (record.parents?.length) {
            for (const reply of record.parents) {
                await this.cascadeDelete(reply.id, remainingDepth, tx);
            }
        }

        // Delete reactions
        if (record.reactions?.length) {
            await tx.reaction.deleteMany({
                where: { commentId: record.id },
            });
        }

        // Delete bookmarks
        if (record.bookmarkedBy?.length) {
            await tx.bookmark.deleteMany({
                where: { commentId: record.id },
            });
        }

        // Delete translations
        if (record.translations?.length) {
            await tx.comment_translation.deleteMany({
                where: { commentId: record.id },
            });
        }
    }
}

// Export factory creator function
export const createCommentDbFactory = (prisma: PrismaClient) => new CommentDbFactory(prisma);
