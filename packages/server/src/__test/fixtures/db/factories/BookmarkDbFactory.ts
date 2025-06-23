import { generatePK, generatePublicId, nanoid } from "../idHelpers.js";
import { type bookmark, type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface BookmarkRelationConfig extends RelationConfig {
    listId?: string | bigint;
    objectType?: string;
    objectId?: string | bigint;
}

/**
 * Database fixture factory for Bookmark model
 * Handles polymorphic bookmarking of any object with optional list organization
 */
export class BookmarkDbFactory extends DatabaseFixtureFactory<
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

    protected getMinimalData(overrides?: Partial<Prisma.bookmarkCreateInput>): Prisma.bookmarkCreateInput {
        return {
            id: generatePK(),
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.bookmarkCreateInput>): Prisma.bookmarkCreateInput {
        return {
            id: generatePK(),
            list: { connect: { id: generatePK() } }, // Will be overridden by relationship config
            ...overrides,
        };
    }

    /**
     * Create a bookmark for a specific object type
     */
    async createForObject(
        objectType: string,
        objectId: string | bigint,
        overrides?: Partial<Prisma.bookmarkCreateInput>,
    ): Promise<bookmark> {
        const typeMapping: Record<string, string> = {
            "comment": "comment",
            "issue": "issue",
            "resource": "resource",
            "tag": "tag",
            "team": "team",
            "user": "user",
        };

        const fieldName = typeMapping[objectType];
        if (!fieldName) {
            throw new Error(`Invalid bookmark object type: ${objectType}`);
        }

        const data: Prisma.bookmarkCreateInput = {
            ...this.getMinimalData(),
            [fieldName]: { connect: { id: BigInt(objectId) } },
            ...overrides,
        };

        const result = await this.prisma.bookmark.create({ data });
        this.trackCreatedId(result.id.toString());
        return result;
    }

    /**
     * Create a bookmark in a specific list
     */
    async createInList(
        listId: string | bigint,
        objectType: string,
        objectId: string | bigint,
        overrides?: Partial<Prisma.bookmarkCreateInput>,
    ): Promise<bookmark> {
        const bookmark = await this.createForObject(objectType, objectId, {
            list: { connect: { id: BigInt(listId) } },
            ...overrides,
        });
        return bookmark;
    }

    /**
     * Create multiple bookmarks for a user
     */
    async createBulk(
        objects: Array<{ type: string; id: string | bigint }>,
        listId?: string | bigint,
    ): Promise<bookmark[]> {
        const bookmarks: bookmark[] = [];
        
        for (const obj of objects) {
            const bookmark = listId
                ? await this.createInList(listId, obj.type, obj.id)
                : await this.createForObject(obj.type, obj.id);
            bookmarks.push(bookmark);
        }
        
        return bookmarks;
    }

    protected getDefaultInclude(): Prisma.bookmarkInclude {
        return {
            list: {
                select: {
                    id: true,
                    publicId: true,
                    translations: {
                        where: { language: "en" },
                        take: 1,
                    },
                },
            },
            // Include polymorphic relationships
            comment: { select: { id: true } },
            issue: { select: { id: true, publicId: true } },
            resource: { select: { id: true, publicId: true } },
            tag: { select: { id: true, tag: true } },
            team: { select: { id: true, publicId: true, handle: true } },
            user: { select: { id: true, publicId: true, name: true, handle: true } },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.bookmarkCreateInput,
        config: BookmarkRelationConfig,
        tx: any,
    ): Promise<Prisma.bookmarkCreateInput> {
        const data = { ...baseData };


        // Handle list connection
        if (config.listId) {
            data.list = { connect: { id: BigInt(config.listId) } };
        }

        // Handle object connection
        if (config.objectType && config.objectId) {
            const typeMapping: Record<string, string> = {
                "comment": "comment",
                "issue": "issue",
                "resource": "resource",
                "tag": "tag",
                "team": "team",
                "user": "user",
            };

            const fieldName = typeMapping[config.objectType];
            if (fieldName) {
                data[fieldName] = { connect: { id: BigInt(config.objectId) } };
            }
        }

        return data;
    }

    /**
     * Create test scenarios
     */
    async createUserBookmark(userId: string | bigint): Promise<bookmark> {
        return this.createForObject("user", userId);
    }

    async createOrganizedCollection(
        listId: string | bigint,
        theme: string,
    ): Promise<bookmark[]> {
        // Create themed bookmarks
        const themedObjects = [
            { type: "resource", id: generatePK() },
            { type: "team", id: generatePK() },
            { type: "user", id: generatePK() },
        ];

        return this.createBulk(themedObjects, listId);
    }

    async createCrossTypeBookmarks(): Promise<bookmark[]> {
        // Create bookmarks across different object types
        const diverseObjects = [
            { type: "user", id: generatePK() },
            { type: "team", id: generatePK() },
            { type: "comment", id: generatePK() },
            { type: "issue", id: generatePK() },
            { type: "resource", id: generatePK() },
            { type: "tag", id: generatePK() },
        ];

        return this.createBulk(diverseObjects);
    }

    protected async checkModelConstraints(record: bookmark): Promise<string[]> {
        const violations: string[] = [];

        // Check exactly one bookmarked object
        const bookmarkableFields = [
            "commentId", "issueId", "resourceId",
            "tagId", "teamId", "userId",
        ];
        
        const connectedObjects = bookmarkableFields.filter(field => 
            record[field as keyof bookmark],
        );
        
        if (connectedObjects.length === 0) {
            violations.push("Bookmark must reference exactly one object");
        } else if (connectedObjects.length > 1) {
            violations.push("Bookmark cannot reference multiple objects");
        }

        // Check list exists if specified
        if (record.listId) {
            const list = await this.prisma.bookmarkList.findUnique({
                where: { id: record.listId },
            });
            if (!list) {
                violations.push("Bookmark list must exist");
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
                // Missing id
            },
            invalidTypes: {
                id: "not-a-snowflake",
            },
            noBookmarkableObject: {
                id: generatePK(),
                // No object connected
            },
            multipleObjects: {
                id: generatePK(),
                resource: { connect: { id: generatePK() } },
                team: { connect: { id: generatePK() } },
                user: { connect: { id: generatePK() } },
                // Multiple objects (invalid)
            },
            invalidList: {
                id: generatePK(),
                list: { connect: { id: BigInt("999999999999999") } }, // Non-existent list
                resource: { connect: { id: generatePK() } },
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.bookmarkCreateInput> {
        const userId = generatePK();
        const resourceId = generatePK();
        
        return {
            userBookmark: {
                ...this.getMinimalData(),
                user: { connect: { id: userId } },
            },
            orphanedBookmark: {
                ...this.getMinimalData(),
                resource: { connect: { id: resourceId } },
                // No list - floating bookmark
            },
            tagBookmark: {
                ...this.getMinimalData(),
                tag: { connect: { id: generatePK() } }, // Less common bookmark type
            },
            commentBookmark: {
                ...this.getMinimalData(),
                comment: { connect: { id: generatePK() } }, // Bookmarking a comment
            },
            issueBookmark: {
                ...this.getMinimalData(),
                issue: { connect: { id: generatePK() } }, // Bookmarking an issue
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            // No direct children to cascade
        };
    }

    protected async deleteRelatedRecords(
        record: bookmark,
        remainingDepth: number,
        tx: any,
    ): Promise<void> {
        // Bookmarks don't have child records to delete
    }
}

// Export factory creator function
export const createBookmarkDbFactory = (prisma: PrismaClient) => new BookmarkDbFactory(prisma);
