import { nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient, type comment } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface CommentRelationConfig extends RelationConfig {
    withOwnerUser?: boolean | string;
    withOwnerTeam?: boolean | string;
    withResourceVersion?: boolean | string;
    withIssue?: boolean | string;
    withParent?: boolean | string;
    withPullRequest?: boolean | string;
    withTranslations?: boolean | Array<{ language: string; text: string }>;
    withReplies?: boolean | number;
}

/**
 * Enhanced database fixture factory for Comment model
 * Provides comprehensive testing capabilities for commenting system
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for various commentable types
 * - Threaded comment support
 * - Multi-language translations
 * - Ownership by user or team
 * - Reaction and bookmark tracking
 * - Predefined test scenarios
 * - Comprehensive validation
 */
export class CommentDbFactory extends EnhancedDatabaseFactory<
    comment,
    Prisma.commentCreateInput,
    Prisma.commentInclude,
    Prisma.commentUpdateInput
> {
    protected scenarios: Record<string, TestScenario> = {};
    constructor(prisma: PrismaClient) {
        super("comment", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.comment;
    }

    /**
     * Get complete test fixtures for Comment model
     */
    protected getFixtures(): DbTestFixtures<Prisma.commentCreateInput, Prisma.commentUpdateInput> {
        return {
            minimal: {
                id: this.generateId(),
                ownedByUser: { connect: { id: this.generateId() } },
                translations: {
                    create: [{
                        id: this.generateId(),
                        language: "en",
                        text: "This is a test comment",
                    }],
                },
                // Must have at least one target
                resourceVersion: { connect: { id: this.generateId() } },
            },
            complete: {
                id: this.generateId(),
                ownedByUser: { connect: { id: this.generateId() } },
                resourceVersion: { connect: { id: this.generateId() } },
                score: 5,
                bookmarks: 2,
                translations: {
                    create: [
                        {
                            id: this.generateId(),
                            language: "en",
                            text: "This is a detailed comment with multiple paragraphs.\n\nIt includes formatting and links.",
                        },
                        {
                            id: this.generateId(),
                            language: "es",
                            text: "Este es un comentario detallado con múltiples párrafos.\n\nIncluye formato y enlaces.",
                        },
                    ],
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id and owner
                    resourceVersion: { connect: { id: this.generateId() } },
                },
                invalidTypes: {
                    id: "not-a-bigint",
                    ownedByUserId: 123, // Should be string
                    score: "five", // Should be number
                    bookmarks: true, // Should be number
                },
                noOwner: {
                    id: this.generateId(),
                    // Missing both ownedByUserId and ownedByTeamId
                    resourceVersion: { connect: { id: this.generateId() } },
                },
                noTarget: {
                    id: this.generateId(),
                    ownedByUser: { connect: { id: this.generateId() } },
                    // Missing any target (resourceVersion, issue, pullRequest)
                },
                multipleOwners: {
                    id: this.generateId(),
                    ownedByUser: { connect: { id: this.generateId() } },
                    ownedByTeam: { connect: { id: this.generateId() } }, // Should not have both
                    resourceVersion: { connect: { id: this.generateId() } },
                },
                invalidParentRelation: {
                    id: this.generateId(),
                    ownedByUser: { connect: { id: this.generateId() } },
                    parent: { connect: { id: this.generateId() } }, // Cannot be parent of itself
                    resourceVersion: { connect: { id: this.generateId() } },
                },
            },
            edgeCases: {
                deeplyNestedComment: {
                    id: this.generateId(),
                    ownedByUser: { connect: { id: this.generateId() } },
                    parent: { connect: { id: this.generateId() } }, // Deep nesting
                    resourceVersion: { connect: { id: this.generateId() } },
                    translations: {
                        create: [{
                            id: this.generateId(),
                            language: "en",
                            text: "Reply to a reply to a reply...",
                        }],
                    },
                },
                maxLengthComment: {
                    id: this.generateId(),
                    ownedByUser: { connect: { id: this.generateId() } },
                    resourceVersion: { connect: { id: this.generateId() } },
                    translations: {
                        create: [{
                            id: this.generateId(),
                            language: "en",
                            text: "a".repeat(32768), // Max length text
                        }],
                    },
                },
                unicodeComment: {
                    id: this.generateId(),
                    ownedByUser: { connect: { id: this.generateId() } },
                    resourceVersion: { connect: { id: this.generateId() } },
                    translations: {
                        create: [{
                            id: this.generateId(),
                            language: "en",
                            text: "Unicode test: 🎉 测试 테스트 тест اختبار",
                        }],
                    },
                },
                highlyRatedComment: {
                    id: this.generateId(),
                    ownedByUser: { connect: { id: this.generateId() } },
                    resourceVersion: { connect: { id: this.generateId() } },
                    score: 1000,
                    bookmarks: 500,
                    translations: {
                        create: [{
                            id: this.generateId(),
                            language: "en",
                            text: "This comment was extremely helpful!",
                        }],
                    },
                },
                negativeScoreComment: {
                    id: this.generateId(),
                    ownedByUser: { connect: { id: this.generateId() } },
                    resourceVersion: { connect: { id: this.generateId() } },
                    score: -50,
                    translations: {
                        create: [{
                            id: this.generateId(),
                            language: "en",
                            text: "This comment was not well received",
                        }],
                    },
                },
                teamOwnedComment: {
                    id: this.generateId(),
                    ownedByTeam: { connect: { id: this.generateId() } }, // Team owned instead of user
                    issue: { connect: { id: this.generateId() } },
                    translations: {
                        create: [{
                            id: this.generateId(),
                            language: "en",
                            text: "Official team response",
                        }],
                    },
                },
            },
            updates: {
                minimal: {
                    translations: {
                        update: [{
                            where: { 
                                id: this.generateId(),
                            },
                            data: { text: "Updated comment text" },
                        }],
                    },
                },
                complete: {
                    score: 10,
                    bookmarks: 5,
                    translations: {
                        create: [{
                            id: this.generateId(),
                            language: "fr",
                            text: "Texte de commentaire mis à jour",
                        }],
                        update: [{
                            where: { 
                                id: this.generateId(),
                            },
                            data: { text: "Completely updated comment with new information" },
                        }],
                    },
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.commentCreateInput>): Prisma.commentCreateInput {
        return {
            id: this.generateId(),
            ownedByUser: { connect: { id: this.generateId() } },
            resourceVersion: { connect: { id: this.generateId() } },
            translations: {
                create: [{
                    id: this.generateId(),
                    language: "en",
                    text: "Test comment",
                }],
            },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.commentCreateInput>): Prisma.commentCreateInput {
        return {
            id: this.generateId(),
            ownedByUser: { connect: { id: this.generateId() } },
            resourceVersion: { connect: { id: this.generateId() } },
            score: 0,
            bookmarks: 0,
            translations: {
                create: [
                    {
                        id: this.generateId(),
                        language: "en",
                        text: "This is a comprehensive test comment with detailed information.",
                    },
                    {
                        id: this.generateId(),
                        language: "es",
                        text: "Este es un comentario de prueba completo con información detallada.",
                    },
                ],
            },
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            simpleUserComment: {
                name: "simpleUserComment",
                description: "Simple user comment on a resource",
                config: {
                    overrides: {
                        translations: {
                            create: [{
                                id: this.generateId(),
                                language: "en",
                                text: "Great resource, very helpful!",
                            }],
                        },
                    },
                },
            },
            issueDiscussion: {
                name: "issueDiscussion",
                description: "Comment in an issue discussion",
                config: {
                    overrides: {
                        issue: { connect: { id: this.generateId() } },
                        resourceVersion: undefined,
                        translations: {
                            create: [{
                                id: this.generateId(),
                                language: "en",
                                text: "I can reproduce this issue on version 2.1.0",
                            }],
                        },
                    },
                },
            },
            threadedConversation: {
                name: "threadedConversation",
                description: "Comment thread with replies",
                config: {
                    overrides: {
                        translations: {
                            create: [{
                                id: this.generateId(),
                                language: "en",
                                text: "Starting a discussion thread",
                            }],
                        },
                    },
                    withReplies: 3,
                },
            },
            pullRequestReview: {
                name: "pullRequestReview",
                description: "Code review comment on pull request",
                config: {
                    overrides: {
                        pullRequest: { connect: { id: this.generateId() } },
                        resourceVersion: undefined,
                        translations: {
                            create: [{
                                id: this.generateId(),
                                language: "en",
                                text: "This code looks good, but consider adding error handling",
                            }],
                        },
                    },
                },
            },
            teamResponse: {
                name: "teamResponse",
                description: "Official team response comment",
                config: {
                    overrides: {
                        ownedByTeam: { connect: { id: this.generateId() } },
                        ownedByUser: undefined,
                        translations: {
                            create: [{
                                id: this.generateId(),
                                language: "en",
                                text: "Thank you for your feedback. We'll address this in the next release.",
                            }],
                        },
                    },
                },
            },
        };
    }

    /**
     * Create comment for specific contexts
     */
    async createResourceComment(userId: string, resourceVersionId: string, text: string): Promise<comment> {
        return await this.createMinimal({
            ownedByUser: { connect: { id: BigInt(userId) } },
            resourceVersion: { connect: { id: BigInt(resourceVersionId) } },
            translations: {
                create: [{
                    id: this.generateId(),
                    language: "en",
                    text,
                }],
            },
        });
    }

    async createIssueComment(userId: string, issueId: string, text: string): Promise<comment> {
        return await this.createMinimal({
            ownedByUser: { connect: { id: BigInt(userId) } },
            issue: { connect: { id: BigInt(issueId) } },
            resourceVersion: undefined,
            translations: {
                create: [{
                    id: this.generateId(),
                    language: "en",
                    text,
                }],
            },
        });
    }

    async createPullRequestComment(userId: string, pullRequestId: string, text: string): Promise<comment> {
        return await this.createMinimal({
            ownedByUser: { connect: { id: BigInt(userId) } },
            pullRequest: { connect: { id: BigInt(pullRequestId) } },
            resourceVersion: undefined,
            translations: {
                create: [{
                    id: this.generateId(),
                    language: "en",
                    text,
                }],
            },
        });
    }

    /**
     * Create a comment reply
     */
    async createReply(parentId: string, userId: string, text: string): Promise<comment> {
        // Get parent to inherit target
        const parent = await this.prisma.comment.findUnique({
            where: { id: BigInt(parentId) },
        });
        
        if (!parent) {
            throw new Error(`Parent comment ${parentId} not found`);
        }

        return await this.createMinimal({
            ownedByUser: { connect: { id: BigInt(userId) } },
            parent: { connect: { id: BigInt(parentId) } },
            ...(parent.resourceVersionId && { resourceVersion: { connect: { id: parent.resourceVersionId } } }),
            ...(parent.issueId && { issue: { connect: { id: parent.issueId } } }),
            ...(parent.pullRequestId && { pullRequest: { connect: { id: parent.pullRequestId } } }),
            translations: {
                create: [{
                    id: this.generateId(),
                    language: "en",
                    text,
                }],
            },
        });
    }

    /**
     * Create a comment thread with replies
     */
    async createCommentThread(rootComment: Partial<Prisma.commentCreateInput>, replyCount = 3): Promise<comment[]> {
        const comments: comment[] = [];
        
        // Create root comment
        const root = await this.createMinimal(rootComment);
        comments.push(root);
        
        // Create replies
        for (let i = 0; i < replyCount; i++) {
            const reply = await this.createReply(
                this.bigIntToString(root.id),
                this.bigIntToString(this.generateId()),
                `Reply ${i + 1} to the comment`,
            );
            comments.push(reply);
        }
        
        return comments;
    }

    protected getDefaultInclude(): Prisma.commentInclude {
        return {
            ownedByUser: true,
            ownedByTeam: true,
            resourceVersion: true,
            issue: true,
            pullRequest: true,
            parent: true,
            translations: true,
            parents: {
                take: 5,
                include: {
                    translations: true,
                    ownedByUser: true,
                },
            },
            _count: {
                select: {
                    parents: true,
                    bookmarkedBy: true,
                    reactions: true,
                    reports: true,
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

        // Handle owner relationships
        if (config.withOwnerUser) {
            const userId = typeof config.withOwnerUser === "string" ? BigInt(config.withOwnerUser) : this.generateId();
            data.ownedByUser = { connect: { id: userId } };
            data.ownedByTeam = undefined;
        }
        
        if (config.withOwnerTeam) {
            const teamId = typeof config.withOwnerTeam === "string" ? BigInt(config.withOwnerTeam) : this.generateId();
            data.ownedByTeam = { connect: { id: teamId } };
            data.ownedByUser = undefined;
        }

        // Handle target relationships
        if (config.withResourceVersion) {
            const resourceVersionId = typeof config.withResourceVersion === "string" ? BigInt(config.withResourceVersion) : this.generateId();
            data.resourceVersion = { connect: { id: resourceVersionId } };
        }
        
        if (config.withIssue) {
            const issueId = typeof config.withIssue === "string" ? BigInt(config.withIssue) : this.generateId();
            data.issue = { connect: { id: issueId } };
            data.resourceVersion = undefined;
        }
        
        if (config.withPullRequest) {
            const pullRequestId = typeof config.withPullRequest === "string" ? BigInt(config.withPullRequest) : this.generateId();
            data.pullRequest = { connect: { id: pullRequestId } };
            data.resourceVersion = undefined;
        }

        // Handle parent relationship
        if (config.withParent) {
            const parentId = typeof config.withParent === "string" ? BigInt(config.withParent) : this.generateId();
            data.parent = { connect: { id: parentId } };
        }

        // Handle translations
        if (config.withTranslations && Array.isArray(config.withTranslations)) {
            data.translations = {
                create: config.withTranslations.map(trans => ({
                    id: this.generateId(),
                    ...trans,
                })),
            };
        }

        // Handle replies (will be created after main comment)
        if (config.withReplies && typeof config.withReplies === "number") {
            // Replies will be handled in createWithRelations
        }

        return data;
    }

    protected async checkModelConstraints(record: comment): Promise<string[]> {
        const violations: string[] = [];
        
        // Check that comment has an owner
        if (!record.ownedByUserId && !record.ownedByTeamId) {
            violations.push("Comment must have an owner (user or team)");
        }
        
        // Check that comment doesn't have multiple owners
        if (record.ownedByUserId && record.ownedByTeamId) {
            violations.push("Comment cannot be owned by both user and team");
        }

        // Check that comment has a target
        const hasTarget = record.resourceVersionId || record.issueId || record.pullRequestId;
        if (!hasTarget) {
            violations.push("Comment must target a resource version, issue, or pull request");
        }

        // Check that comment doesn't have multiple targets
        const targetCount = [record.resourceVersionId, record.issueId, record.pullRequestId].filter(Boolean).length;
        if (targetCount > 1) {
            violations.push("Comment can only have one target");
        }

        // Check parent relationship validity
        if (record.parentId) {
            if (record.parentId === record.id) {
                violations.push("Comment cannot be its own parent");
            }
            
            // Check parent exists and has same target
            const parent = await this.prisma.comment.findUnique({
                where: { id: record.parentId },
            });
            
            if (!parent) {
                violations.push("Parent comment does not exist");
            } else {
                // Check same target
                if (record.resourceVersionId && parent.resourceVersionId !== record.resourceVersionId) {
                    violations.push("Reply must target same resource as parent");
                }
                if (record.issueId && parent.issueId !== record.issueId) {
                    violations.push("Reply must target same issue as parent");
                }
                if (record.pullRequestId && parent.pullRequestId !== record.pullRequestId) {
                    violations.push("Reply must target same pull request as parent");
                }
            }
        }

        // Check translations exist
        const translations = await this.prisma.comment_translation.findMany({
            where: { commentId: record.id },
        });
        
        if (translations.length === 0) {
            violations.push("Comment must have at least one translation");
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            translations: true,
            parents: true,
            bookmarkedBy: true,
            reactions: true,
            reactionSummaries: true,
            reports: true,
            subscriptions: true,
        };
    }

    protected async deleteRelatedRecords(
        record: comment,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[],
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        const shouldDelete = (relation: string) => 
            !includeOnly || includeOnly.includes(relation);

        // Delete child comments (replies)
        if (shouldDelete("parents")) {
            const children = await tx.comment.findMany({
                where: { parentId: record.id },
            });
            for (const child of children) {
                await this.delete(child.id);
            }
        }

        // Delete translations
        if (shouldDelete("translations")) {
            await tx.comment_translation.deleteMany({
                where: { commentId: record.id },
            });
        }

        // Delete bookmarks
        if (shouldDelete("bookmarkedBy")) {
            await tx.bookmark.deleteMany({
                where: { commentId: record.id },
            });
        }

        // Delete reactions
        if (shouldDelete("reactions")) {
            await tx.reaction.deleteMany({
                where: { commentId: record.id },
            });
        }

        // Delete reaction summaries
        if (shouldDelete("reactionSummaries")) {
            await tx.reaction_summary.deleteMany({
                where: { commentId: record.id },
            });
        }

        // Delete reports
        if (shouldDelete("reports")) {
            await tx.report.deleteMany({
                where: { commentId: record.id },
            });
        }

        // Delete notification subscriptions
        if (shouldDelete("subscriptions")) {
            await tx.notification_subscription.deleteMany({
                where: { commentId: record.id },
            });
        }
    }

    /**
     * Get comment thread (parent and all replies)
     */
    async getCommentThread(commentId: string): Promise<comment[]> {
        const rootComment = await this.prisma.comment.findUnique({
            where: { id: BigInt(commentId) },
            include: this.getDefaultInclude(),
        });
        
        if (!rootComment) {
            return [];
        }

        // Find the root of the thread
        let currentComment = rootComment;
        while (currentComment.parentId) {
            const parent = await this.prisma.comment.findUnique({
                where: { id: currentComment.parentId },
                include: this.getDefaultInclude(),
            });
            if (!parent) break;
            currentComment = parent;
        }

        // Get all comments in thread
        const thread = await this.getThreadComments(this.bigIntToString(currentComment.id));
        return thread;
    }

    /**
     * Recursively get all comments in a thread
     */
    private async getThreadComments(rootId: string): Promise<comment[]> {
        const comments: comment[] = [];
        
        const root = await this.prisma.comment.findUnique({
            where: { id: BigInt(rootId) },
            include: this.getDefaultInclude(),
        });
        
        if (!root) return comments;
        
        comments.push(root);
        
        // Get all replies
        const replies = await this.prisma.comment.findMany({
            where: { parentId: BigInt(rootId) },
            include: this.getDefaultInclude(),
            orderBy: { createdAt: "asc" },
        });
        
        for (const reply of replies) {
            const subThread = await this.getThreadComments(this.bigIntToString(reply.id));
            comments.push(...subThread);
        }
        
        return comments;
    }
}

// Export factory creator function
export const createCommentDbFactory = (prisma: PrismaClient) => 
    new CommentDbFactory(prisma);

// Export the class for type usage
export { CommentDbFactory as CommentDbFactoryClass };
