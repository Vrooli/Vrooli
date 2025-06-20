import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface BookmarkRelationConfig extends RelationConfig {
    withList?: boolean | string;
    withResource?: boolean | string;
    withComment?: boolean | string;
    withIssue?: boolean | string;
    withTag?: boolean | string;
    withTeam?: boolean | string;
    withUser?: boolean | string;
}

/**
 * Enhanced database fixture factory for Bookmark model
 * Provides comprehensive testing capabilities for bookmarking system
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for various bookmarkable types
 * - List management
 * - Polymorphic relationships
 * - Predefined test scenarios
 * - Comprehensive validation
 */
export class BookmarkDbFactory extends EnhancedDatabaseFactory<
    Prisma.bookmark,
    Prisma.bookmarkCreateInput,
    Prisma.bookmarkInclude,
    Prisma.bookmarkUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('bookmark', prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.bookmark;
    }

    /**
     * Get complete test fixtures for Bookmark model
     */
    protected getFixtures(): DbTestFixtures<Prisma.bookmarkCreateInput, Prisma.bookmarkUpdateInput> {
        return {
            minimal: {
                id: BigInt(generatePK()),
                // At least one relation must be present for a valid bookmark
                user: {
                    connect: { id: BigInt(generatePK()) }
                },
            },
            complete: {
                id: BigInt(generatePK()),
                list: {
                    connect: { id: BigInt(generatePK()) }
                },
                resource: {
                    connect: { id: BigInt(generatePK()) }
                },
                user: {
                    connect: { id: BigInt(generatePK()) }
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id
                    user: {
                        connect: { id: BigInt(generatePK()) }
                    },
                },
                invalidTypes: {
                    id: "not-a-bigint",
                    list: 123, // Should be object
                    resource: true, // Should be object
                    user: null, // Should be object
                },
                noRelation: {
                    id: BigInt(generatePK()),
                    // No relation specified - invalid as bookmark must reference something
                },
                multipleTargets: {
                    id: BigInt(generatePK()),
                    // Multiple bookmark targets should not be allowed
                    resource: { connect: { id: BigInt(generatePK()) } },
                    comment: { connect: { id: BigInt(generatePK()) } },
                    issue: { connect: { id: BigInt(generatePK()) } },
                    tag: { connect: { id: BigInt(generatePK()) } },
                    team: { connect: { id: BigInt(generatePK()) } },
                    user: { connect: { id: BigInt(generatePK()) } },
                },
            },
            edgeCases: {
                userBookmarkingOwnResource: {
                    id: BigInt(generatePK()),
                    resource: { connect: { id: BigInt(generatePK()) } },
                    user: { connect: { id: BigInt(generatePK()) } }, // Same user owns the resource
                },
                listWithoutUser: {
                    id: BigInt(generatePK()),
                    list: { connect: { id: BigInt(generatePK()) } },
                    resource: { connect: { id: BigInt(generatePK()) } },
                    // No user specified when list is present
                },
                orphanedBookmark: {
                    id: BigInt(generatePK()),
                    user: { connect: { id: BigInt(generatePK()) } },
                    // Referenced entities might be deleted
                },
                duplicateBookmark: {
                    id: BigInt(generatePK()),
                    // Same user bookmarking same resource
                    resource: { connect: { id: BigInt("123456789") } },
                    user: { connect: { id: BigInt("987654321") } },
                },
                maxBookmarksInList: {
                    id: BigInt(generatePK()),
                    list: { connect: { id: BigInt("555555555") } }, // List at capacity
                    resource: { connect: { id: BigInt(generatePK()) } },
                    user: { connect: { id: BigInt(generatePK()) } },
                },
            },
            updates: {
                minimal: {
                    list: { connect: { id: BigInt(generatePK()) } },
                },
                complete: {
                    list: { connect: { id: BigInt(generatePK()) } },
                    // Note: bookmark target (resource, comment, etc.) typically cannot be changed
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.bookmarkCreateInput>): Prisma.bookmarkCreateInput {
        // A minimal bookmark requires at least one target and a user
        return {
            id: BigInt(generatePK()),
            user: {
                connect: { id: BigInt(generatePK()) }
            },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.bookmarkCreateInput>): Prisma.bookmarkCreateInput {
        return {
            id: BigInt(generatePK()),
            list: { connect: { id: BigInt(generatePK()) } },
            resource: { connect: { id: BigInt(generatePK()) } },
            user: { connect: { id: BigInt(generatePK()) } },
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            userResourceBookmark: {
                name: "userResourceBookmark",
                description: "User bookmarking a resource",
                config: {
                    overrides: {
                        resource: { connect: { id: BigInt(generatePK()) } },
                        user: { connect: { id: BigInt(generatePK()) } },
                    },
                },
            },
            organizedBookmark: {
                name: "organizedBookmark",
                description: "Bookmark organized in a list",
                config: {
                    overrides: {
                        list: { connect: { id: BigInt(generatePK()) } },
                        resource: { connect: { id: BigInt(generatePK()) } },
                        user: { connect: { id: BigInt(generatePK()) } },
                    },
                },
            },
            commentBookmark: {
                name: "commentBookmark",
                description: "User bookmarking a comment",
                config: {
                    overrides: {
                        comment: { connect: { id: BigInt(generatePK()) } },
                        user: { connect: { id: BigInt(generatePK()) } },
                    },
                },
            },
            issueBookmark: {
                name: "issueBookmark",
                description: "User bookmarking an issue",
                config: {
                    overrides: {
                        issue: { connect: { id: BigInt(generatePK()) } },
                        user: { connect: { id: BigInt(generatePK()) } },
                    },
                },
            },
            tagBookmark: {
                name: "tagBookmark",
                description: "User bookmarking a tag",
                config: {
                    overrides: {
                        tag: { connect: { id: BigInt(generatePK()) } },
                        user: { connect: { id: BigInt(generatePK()) } },
                    },
                },
            },
            teamBookmark: {
                name: "teamBookmark",
                description: "User bookmarking a team",
                config: {
                    overrides: {
                        team: { connect: { id: BigInt(generatePK()) } },
                        user: { connect: { id: BigInt(generatePK()) } },
                    },
                },
            },
        };
    }

    /**
     * Create bookmark for specific target types
     */
    async createResourceBookmark(userId: string, resourceId: string, listId?: string): Promise<Prisma.bookmark> {
        return await this.createMinimal({
            user: { connect: { id: BigInt(userId) } },
            resource: { connect: { id: BigInt(resourceId) } },
            ...(listId && { list: { connect: { id: BigInt(listId) } } }),
        });
    }

    async createCommentBookmark(userId: string, commentId: string, listId?: string): Promise<Prisma.bookmark> {
        return await this.createMinimal({
            user: { connect: { id: BigInt(userId) } },
            comment: { connect: { id: BigInt(commentId) } },
            ...(listId && { list: { connect: { id: BigInt(listId) } } }),
        });
    }

    async createIssueBookmark(userId: string, issueId: string, listId?: string): Promise<Prisma.bookmark> {
        return await this.createMinimal({
            user: { connect: { id: BigInt(userId) } },
            issue: { connect: { id: BigInt(issueId) } },
            ...(listId && { list: { connect: { id: BigInt(listId) } } }),
        });
    }

    async createTagBookmark(userId: string, tagId: string, listId?: string): Promise<Prisma.bookmark> {
        return await this.createMinimal({
            user: { connect: { id: BigInt(userId) } },
            tag: { connect: { id: BigInt(tagId) } },
            ...(listId && { list: { connect: { id: BigInt(listId) } } }),
        });
    }

    async createTeamBookmark(userId: string, teamId: string, listId?: string): Promise<Prisma.bookmark> {
        return await this.createMinimal({
            user: { connect: { id: BigInt(userId) } },
            team: { connect: { id: BigInt(teamId) } },
            ...(listId && { list: { connect: { id: BigInt(listId) } } }),
        });
    }

    /**
     * Create multiple bookmarks for testing collections
     */
    async createBookmarkCollection(userId: string, listId: string, count: number = 5): Promise<Prisma.bookmark[]> {
        const bookmarks: Prisma.bookmark[] = [];
        
        for (let i = 0; i < count; i++) {
            const bookmark = await this.createMinimal({
                user: { connect: { id: BigInt(userId) } },
                list: { connect: { id: BigInt(listId) } },
                resource: { connect: { id: BigInt(generatePK()) } },
            });
            bookmarks.push(bookmark);
        }
        
        return bookmarks;
    }

    protected getDefaultInclude(): Prisma.bookmarkInclude {
        return {
            list: true,
            resource: true,
            comment: true,
            issue: true,
            tag: true,
            team: true,
            user: true,
        };
    }

    protected async applyRelationships(
        baseData: Prisma.bookmarkCreateInput,
        config: BookmarkRelationConfig,
        tx: any
    ): Promise<Prisma.bookmarkCreateInput> {
        let data = { ...baseData };

        // Handle list relationship
        if (config.withList) {
            const listId = typeof config.withList === 'string' ? config.withList : generatePK().toString();
            data.list = { connect: { id: BigInt(listId) } };
        }

        // Handle resource relationship
        if (config.withResource) {
            const resourceId = typeof config.withResource === 'string' ? config.withResource : generatePK().toString();
            data.resource = { connect: { id: BigInt(resourceId) } };
        }

        // Handle comment relationship
        if (config.withComment) {
            const commentId = typeof config.withComment === 'string' ? config.withComment : generatePK().toString();
            data.comment = { connect: { id: BigInt(commentId) } };
        }

        // Handle issue relationship
        if (config.withIssue) {
            const issueId = typeof config.withIssue === 'string' ? config.withIssue : generatePK().toString();
            data.issue = { connect: { id: BigInt(issueId) } };
        }

        // Handle tag relationship
        if (config.withTag) {
            const tagId = typeof config.withTag === 'string' ? config.withTag : generatePK().toString();
            data.tag = { connect: { id: BigInt(tagId) } };
        }

        // Handle team relationship
        if (config.withTeam) {
            const teamId = typeof config.withTeam === 'string' ? config.withTeam : generatePK().toString();
            data.team = { connect: { id: BigInt(teamId) } };
        }

        // Handle user relationship
        if (config.withUser) {
            const userId = typeof config.withUser === 'string' ? config.withUser : generatePK().toString();
            data.user = { connect: { id: BigInt(userId) } };
        }

        return data;
    }

    protected async checkModelConstraints(record: Prisma.bookmark): Promise<string[]> {
        const violations: string[] = [];
        
        // Check that only one target is specified
        const targets = [
            record.resourceId,
            record.commentId,
            record.issueId,
            record.tagId,
            record.teamId,
        ].filter(Boolean);
        
        if (targets.length > 1) {
            violations.push('Bookmark can only have one target');
        }
        
        if (targets.length === 0 && !record.userId) {
            violations.push('Bookmark must have at least one target or user');
        }

        // Check that bookmark belongs to a user or a list
        if (!record.userId && !record.listId) {
            violations.push('Bookmark must belong to a user or a list');
        }

        // Check for duplicate bookmarks
        if (record.userId && targets.length === 1) {
            const existingBookmark = await this.prisma.bookmark.findFirst({
                where: {
                    userId: record.userId,
                    resourceId: record.resourceId,
                    commentId: record.commentId,
                    issueId: record.issueId,
                    tagId: record.tagId,
                    teamId: record.teamId,
                    id: { not: record.id },
                },
            });
            
            if (existingBookmark) {
                violations.push('User already has this bookmark');
            }
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            list: true,
            resource: true,
            comment: true,
            issue: true,
            tag: true,
            team: true,
            user: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.bookmark,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[]
    ): Promise<void> {
        // Bookmarks don't have child records to cascade delete
        // The bookmark itself will be deleted by the parent deleteRelatedRecords
    }

    /**
     * Check if a user has bookmarked a specific item
     */
    async hasUserBookmarked(userId: string, targetId: string, targetType: 'resource' | 'comment' | 'issue' | 'tag' | 'team' | 'user'): Promise<boolean> {
        const whereClause: any = { userId: BigInt(userId) };
        
        switch (targetType) {
            case 'resource':
                whereClause.resourceId = BigInt(targetId);
                break;
            case 'comment':
                whereClause.commentId = BigInt(targetId);
                break;
            case 'issue':
                whereClause.issueId = BigInt(targetId);
                break;
            case 'tag':
                whereClause.tagId = BigInt(targetId);
                break;
            case 'team':
                whereClause.teamId = BigInt(targetId);
                break;
            case 'user':
                // Note: This assumes there's a bookmarked user field
                break;
        }
        
        const bookmark = await this.prisma.bookmark.findFirst({
            where: whereClause,
        });
        
        return bookmark !== null;
    }

    /**
     * Get all bookmarks for a user
     */
    async getUserBookmarks(userId: string, includeList: boolean = false): Promise<Prisma.bookmark[]> {
        return await this.prisma.bookmark.findMany({
            where: { userId: BigInt(userId) },
            include: {
                list: includeList,
                resource: true,
                comment: true,
                issue: true,
                tag: true,
                team: true,
                user: true,
            },
            orderBy: { createdAt: 'desc' },
        });
    }

    /**
     * Get all bookmarks in a list
     */
    async getListBookmarks(listId: string): Promise<Prisma.bookmark[]> {
        return await this.prisma.bookmark.findMany({
            where: { listId: BigInt(listId) },
            include: this.getDefaultInclude(),
            orderBy: { createdAt: 'desc' },
        });
    }
}

// Export factory creator function
export const createBookmarkDbFactory = (prisma: PrismaClient) => 
    BookmarkDbFactory.getInstance('bookmark', prisma);

// Export the class for type usage
export { BookmarkDbFactory as BookmarkDbFactoryClass };