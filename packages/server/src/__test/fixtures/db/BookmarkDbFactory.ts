// AI_CHECK: TYPE_SAFETY=server-factory-bigint-migration | LAST: 2025-06-29 - Migrated to BigInt IDs, snake_case tables, correct field names
import { type bookmark, type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";

/**
 * Enhanced database fixture factory for Bookmark model
 * Uses snake_case table name: 'bookmark'
 * 
 * Breaking Changes:
 * - IDs are now BigInt only
 * - Uses snake_case Prisma types (bookmarkCreateInput, bookmarkInclude)
 * - Uses correct field names from schema: listId, resourceId, commentId, issueId, tagId, teamId, userId
 * - Uses direct field assignment for relationships (foreign key IDs)
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for bookmarking various resource types
 * - Bookmark list associations
 * - User relationships
 * - Predefined test scenarios
 */
export class BookmarkDbFactory extends EnhancedDatabaseFactory<
    bookmark,
    Prisma.bookmarkCreateInput,
    Prisma.bookmarkInclude,
    Prisma.bookmarkUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("bookmark", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.bookmark;
    }

    protected generateMinimalData(overrides?: Partial<Prisma.bookmarkCreateInput>): Prisma.bookmarkCreateInput {
        return {
            id: this.generateId(),
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.bookmarkCreateInput>): Prisma.bookmarkCreateInput {
        return {
            id: this.generateId(),
            ...overrides,
        };
    }

    /**
     * Create bookmark for a user
     */
    async createForUser(userId: bigint, overrides?: Partial<Prisma.bookmarkCreateInput>) {
        const data = this.generateMinimalData({
            ...overrides,
            user: { connect: { id: userId } },
        });
        return await this.createMinimal(data);
    }

    /**
     * Create bookmark with bookmark list
     */
    async createWithList(listId: bigint, overrides?: Partial<Prisma.bookmarkCreateInput>) {
        const data = this.generateMinimalData({
            ...overrides,
            list: { connect: { id: listId } },
        });
        return await this.createMinimal(data);
    }

    /**
     * Create bookmark for resource
     */
    async createForResource(resourceId: bigint, overrides?: Partial<Prisma.bookmarkCreateInput>) {
        return await this.createMinimal({
            resource: { connect: { id: resourceId } },
            ...overrides,
        });
    }

    /**
     * Create bookmark for comment
     */
    async createForComment(commentId: bigint, overrides?: Partial<Prisma.bookmarkCreateInput>) {
        return await this.createMinimal({
            comment: { connect: { id: commentId } },
            ...overrides,
        });
    }

    /**
     * Create bookmark for issue
     */
    async createForIssue(issueId: bigint, overrides?: Partial<Prisma.bookmarkCreateInput>) {
        return await this.createMinimal({
            issue: { connect: { id: issueId } },
            ...overrides,
        });
    }

    /**
     * Create bookmark for tag
     */
    async createForTag(tagId: bigint, overrides?: Partial<Prisma.bookmarkCreateInput>) {
        return await this.createMinimal({
            tag: { connect: { id: tagId } },
            ...overrides,
        });
    }

    /**
     * Create bookmark for team
     */
    async createForTeam(teamId: bigint, overrides?: Partial<Prisma.bookmarkCreateInput>) {
        return await this.createMinimal({
            team: { connect: { id: teamId } },
            ...overrides,
        });
    }

    /**
     * Create bookmark with multiple associations
     */
    async createComplex(params: {
        userId?: bigint;
        listId?: bigint;
        resourceId?: bigint;
        commentId?: bigint;
        issueId?: bigint;
        tagId?: bigint;
        teamId?: bigint;
    }, overrides?: Partial<Prisma.bookmarkCreateInput>) {
        return await this.createMinimal({
            ...params,
            ...overrides,
        });
    }
}
