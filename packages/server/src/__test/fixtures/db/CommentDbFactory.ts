import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
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
    Prisma.commentCreateInput,
    Prisma.commentCreateInput,
    Prisma.commentInclude,
    Prisma.commentUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('comment', prisma);
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
                id: generatePK().toString(),
                ownedByUserId: generatePK().toString(),
                translations: {
                    create: [{
                        id: generatePK().toString(),
                        language: "en",
                        text: "This is a test comment",
                    }],
                },
                // Must have at least one target
                resourceVersionId: generatePK().toString(),
            },
            complete: {
                id: generatePK().toString(),
                ownedByUserId: generatePK().toString(),
                resourceVersionId: generatePK().toString(),
                score: 5,
                bookmarks: 2,
                translations: {
                    create: [
                        {
                            id: generatePK().toString(),
                            language: "en",
                            text: "This is a detailed comment with multiple paragraphs.\n\nIt includes formatting and links.",
                        },
                        {
                            id: generatePK().toString(),
                            language: "es",
                            text: "Este es un comentario detallado con múltiples párrafos.\n\nIncluye formato y enlaces.",
                        },
                    ],
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id and owner
                    resourceVersionId: generatePK().toString(),
                },
                invalidTypes: {
                    id: "not-a-bigint",
                    ownedByUserId: 123, // Should be string
                    score: "five", // Should be number
                    bookmarks: true, // Should be number
                },
                noOwner: {
                    id: generatePK().toString(),
                    // Missing both ownedByUserId and ownedByTeamId
                    resourceVersionId: generatePK().toString(),
                },
                noTarget: {
                    id: generatePK().toString(),
                    ownedByUserId: generatePK().toString(),
                    // Missing any target (resourceVersion, issue, pullRequest)
                },
                multipleOwners: {
                    id: generatePK().toString(),
                    ownedByUserId: generatePK().toString(),
                    ownedByTeamId: generatePK().toString(), // Should not have both
                    resourceVersionId: generatePK().toString(),
                },
                invalidParentRelation: {
                    id: generatePK().toString(),
                    ownedByUserId: generatePK().toString(),
                    parentId: "self_id", // Cannot be parent of itself
                    resourceVersionId: generatePK().toString(),
                },
            },
            edgeCases: {
                deeplyNestedComment: {
                    id: generatePK().toString(),
                    ownedByUserId: generatePK().toString(),
                    parentId: generatePK().toString(), // Deep nesting
                    resourceVersionId: generatePK().toString(),
                    translations: {
                        create: [{
                            id: generatePK().toString(),
                            language: "en",
                            text: "Reply to a reply to a reply...",
                        }],
                    },
                },
                maxLengthComment: {
                    id: generatePK().toString(),
                    ownedByUserId: generatePK().toString(),
                    resourceVersionId: generatePK().toString(),
                    translations: {
                        create: [{
                            id: generatePK().toString(),
                            language: "en",
                            text: 'a'.repeat(32768), // Max length text
                        }],
                    },
                },
                unicodeComment: {
                    id: generatePK().toString(),
                    ownedByUserId: generatePK().toString(),
                    resourceVersionId: generatePK().toString(),
                    translations: {
                        create: [{
                            id: generatePK().toString(),
                            language: "en",
                            text: "Unicode test: 🎉 测试 테스트 тест اختبار",
                        }],
                    },
                },
                highlyRatedComment: {
                    id: generatePK().toString(),
                    ownedByUserId: generatePK().toString(),
                    resourceVersionId: generatePK().toString(),
                    score: 1000,
                    bookmarks: 500,
                    translations: {
                        create: [{
                            id: generatePK().toString(),
                            language: "en",
                            text: "This comment was extremely helpful!",
                        }],
                    },
                },
                negativeScoreComment: {
                    id: generatePK().toString(),
                    ownedByUserId: generatePK().toString(),
                    resourceVersionId: generatePK().toString(),
                    score: -50,
                    translations: {
                        create: [{
                            id: generatePK().toString(),
                            language: "en",
                            text: "This comment was not well received",
                        }],
                    },
                },
                teamOwnedComment: {
                    id: generatePK().toString(),
                    ownedByTeamId: generatePK().toString(), // Team owned instead of user
                    issueId: generatePK().toString(),
                    translations: {
                        create: [{
                            id: generatePK().toString(),
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
                                commentId_language: {
                                    commentId: generatePK().toString(),
                                    language: "en",
                                },
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
                            id: generatePK().toString(),
                            language: "fr",
                            text: "Texte de commentaire mis à jour",
                        }],
                        update: [{
                            where: { 
                                commentId_language: {
                                    commentId: generatePK().toString(),
                                    language: "en",
                                },
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
            id: generatePK().toString(),
            ownedByUserId: generatePK().toString(),
            resourceVersionId: generatePK().toString(),
            translations: {
                create: [{
                    id: generatePK().toString(),
                    language: "en",
                    text: "Test comment",
                }],
            },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.commentCreateInput>): Prisma.commentCreateInput {
        return {
            id: generatePK().toString(),
            ownedByUserId: generatePK().toString(),
            resourceVersionId: generatePK().toString(),
            score: 0,
            bookmarks: 0,
            translations: {
                create: [
                    {
                        id: generatePK().toString(),
                        language: "en",
                        text: "This is a comprehensive test comment with detailed information.",
                    },
                    {
                        id: generatePK().toString(),
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
                                id: generatePK().toString(),
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
                        issueId: generatePK().toString(),
                        resourceVersionId: undefined,
                        translations: {
                            create: [{
                                id: generatePK().toString(),
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
                                id: generatePK().toString(),
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
                        pullRequestId: generatePK().toString(),
                        resourceVersionId: undefined,
                        translations: {
                            create: [{
                                id: generatePK().toString(),
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
                        ownedByTeamId: generatePK().toString(),
                        ownedByUserId: undefined,
                        translations: {
                            create: [{
                                id: generatePK().toString(),
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
    async createResourceComment(userId: string, resourceVersionId: string, text: string): Promise<Prisma.comment> {
        return await this.createMinimal({
            ownedByUserId: userId,
            resourceVersionId,
            translations: {
                create: [{
                    id: generatePK().toString(),
                    language: "en",
                    text,
                }],
            },
        });
    }

    async createIssueComment(userId: string, issueId: string, text: string): Promise<Prisma.comment> {
        return await this.createMinimal({
            ownedByUserId: userId,
            issueId,
            resourceVersionId: undefined,
            translations: {
                create: [{
                    id: generatePK().toString(),
                    language: "en",
                    text,
                }],
            },
        });
    }

    async createPullRequestComment(userId: string, pullRequestId: string, text: string): Promise<Prisma.comment> {
        return await this.createMinimal({
            ownedByUserId: userId,
            pullRequestId,
            resourceVersionId: undefined,
            translations: {
                create: [{
                    id: generatePK().toString(),
                    language: "en",
                    text,
                }],
            },
        });
    }

    /**
     * Create a comment reply
     */
    async createReply(parentId: string, userId: string, text: string): Promise<Prisma.comment> {
        // Get parent to inherit target
        const parent = await this.prisma.comment.findUnique({
            where: { id: parentId },
        });
        
        if (!parent) {
            throw new Error(`Parent comment ${parentId} not found`);
        }

        return await this.createMinimal({
            ownedByUserId: userId,
            parentId,
            resourceVersionId: parent.resourceVersionId,
            issueId: parent.issueId,
            pullRequestId: parent.pullRequestId,
            translations: {
                create: [{
                    id: generatePK().toString(),
                    language: "en",
                    text,
                }],
            },
        });
    }

    /**
     * Create a comment thread with replies
     */
    async createCommentThread(rootComment: Partial<Prisma.commentCreateInput>, replyCount: number = 3): Promise<Prisma.comment[]> {
        const comments: Prisma.comment[] = [];
        
        // Create root comment
        const root = await this.createMinimal(rootComment);
        comments.push(root);
        
        // Create replies
        for (let i = 0; i < replyCount; i++) {
            const reply = await this.createReply(
                root.id,
                generatePK().toString(),
                `Reply ${i + 1} to the comment`
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
        tx: any
    ): Promise<Prisma.commentCreateInput> {
        let data = { ...baseData };

        // Handle owner relationships
        if (config.withOwnerUser) {
            const userId = typeof config.withOwnerUser === 'string' ? config.withOwnerUser : generatePK().toString();
            data.ownedByUserId = userId;
            data.ownedByTeamId = undefined;
        }
        
        if (config.withOwnerTeam) {
            const teamId = typeof config.withOwnerTeam === 'string' ? config.withOwnerTeam : generatePK().toString();
            data.ownedByTeamId = teamId;
            data.ownedByUserId = undefined;
        }

        // Handle target relationships
        if (config.withResourceVersion) {
            const resourceVersionId = typeof config.withResourceVersion === 'string' ? config.withResourceVersion : generatePK().toString();
            data.resourceVersionId = resourceVersionId;
        }
        
        if (config.withIssue) {
            const issueId = typeof config.withIssue === 'string' ? config.withIssue : generatePK().toString();
            data.issueId = issueId;
            data.resourceVersionId = undefined;
        }
        
        if (config.withPullRequest) {
            const pullRequestId = typeof config.withPullRequest === 'string' ? config.withPullRequest : generatePK().toString();
            data.pullRequestId = pullRequestId;
            data.resourceVersionId = undefined;
        }

        // Handle parent relationship
        if (config.withParent) {
            const parentId = typeof config.withParent === 'string' ? config.withParent : generatePK().toString();
            data.parentId = parentId;
        }

        // Handle translations
        if (config.withTranslations && Array.isArray(config.withTranslations)) {
            data.translations = {
                create: config.withTranslations.map(trans => ({
                    id: generatePK().toString(),
                    ...trans,
                })),
            };
        }

        // Handle replies (will be created after main comment)
        if (config.withReplies && typeof config.withReplies === 'number') {
            // Replies will be handled in createWithRelations
        }

        return data;
    }

    protected async checkModelConstraints(record: Prisma.comment): Promise<string[]> {
        const violations: string[] = [];
        
        // Check that comment has an owner
        if (!record.ownedByUserId && !record.ownedByTeamId) {
            violations.push('Comment must have an owner (user or team)');
        }
        
        // Check that comment doesn't have multiple owners
        if (record.ownedByUserId && record.ownedByTeamId) {
            violations.push('Comment cannot be owned by both user and team');
        }

        // Check that comment has a target
        const hasTarget = record.resourceVersionId || record.issueId || record.pullRequestId;
        if (!hasTarget) {
            violations.push('Comment must target a resource version, issue, or pull request');
        }

        // Check that comment doesn't have multiple targets
        const targetCount = [record.resourceVersionId, record.issueId, record.pullRequestId].filter(Boolean).length;
        if (targetCount > 1) {
            violations.push('Comment can only have one target');
        }

        // Check parent relationship validity
        if (record.parentId) {
            if (record.parentId === record.id) {
                violations.push('Comment cannot be its own parent');
            }
            
            // Check parent exists and has same target
            const parent = await this.prisma.comment.findUnique({
                where: { id: record.parentId },
            });
            
            if (!parent) {
                violations.push('Parent comment does not exist');
            } else {
                // Check same target
                if (record.resourceVersionId && parent.resourceVersionId !== record.resourceVersionId) {
                    violations.push('Reply must target same resource as parent');
                }
                if (record.issueId && parent.issueId !== record.issueId) {
                    violations.push('Reply must target same issue as parent');
                }
                if (record.pullRequestId && parent.pullRequestId !== record.pullRequestId) {
                    violations.push('Reply must target same pull request as parent');
                }
            }
        }

        // Check translations exist
        const translations = await this.prisma.comment_translation.findMany({
            where: { commentId: record.id },
        });
        
        if (translations.length === 0) {
            violations.push('Comment must have at least one translation');
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
        record: Prisma.comment,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[]
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        const shouldDelete = (relation: string) => 
            !includeOnly || includeOnly.includes(relation);

        // Delete child comments (replies)
        if (shouldDelete('parents') && record.parents?.length) {
            for (const child of record.parents) {
                await this.cascadeDelete(child.id, remainingDepth, tx, includeOnly);
            }
        }

        // Delete translations
        if (shouldDelete('translations') && record.translations?.length) {
            await tx.comment_translation.deleteMany({
                where: { commentId: record.id },
            });
        }

        // Delete bookmarks
        if (shouldDelete('bookmarkedBy') && record.bookmarkedBy?.length) {
            await tx.bookmark.deleteMany({
                where: { commentId: record.id },
            });
        }

        // Delete reactions
        if (shouldDelete('reactions') && record.reactions?.length) {
            await tx.reaction.deleteMany({
                where: { commentId: record.id },
            });
        }

        // Delete reaction summaries
        if (shouldDelete('reactionSummaries') && record.reactionSummaries?.length) {
            await tx.reaction_summary.deleteMany({
                where: { commentId: record.id },
            });
        }

        // Delete reports
        if (shouldDelete('reports') && record.reports?.length) {
            await tx.report.deleteMany({
                where: { commentId: record.id },
            });
        }

        // Delete notification subscriptions
        if (shouldDelete('subscriptions') && record.subscriptions?.length) {
            await tx.notification_subscription.deleteMany({
                where: { commentId: record.id },
            });
        }
    }

    /**
     * Get comment thread (parent and all replies)
     */
    async getCommentThread(commentId: string): Promise<Prisma.comment[]> {
        const rootComment = await this.prisma.comment.findUnique({
            where: { id: commentId },
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
        const thread = await this.getThreadComments(currentComment.id);
        return thread;
    }

    /**
     * Recursively get all comments in a thread
     */
    private async getThreadComments(rootId: string): Promise<Prisma.comment[]> {
        const comments: Prisma.comment[] = [];
        
        const root = await this.prisma.comment.findUnique({
            where: { id: rootId },
            include: this.getDefaultInclude(),
        });
        
        if (!root) return comments;
        
        comments.push(root);
        
        // Get all replies
        const replies = await this.prisma.comment.findMany({
            where: { parentId: rootId },
            include: this.getDefaultInclude(),
            orderBy: { createdAt: 'asc' },
        });
        
        for (const reply of replies) {
            const subThread = await this.getThreadComments(reply.id);
            comments.push(...subThread);
        }
        
        return comments;
    }
}

// Export factory creator function
export const createCommentDbFactory = (prisma: PrismaClient) => 
    CommentDbFactory.getInstance('comment', prisma);

// Export the class for type usage
export { CommentDbFactory as CommentDbFactoryClass };