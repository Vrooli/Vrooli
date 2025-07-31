import { type Prisma } from "@prisma/client";
import { generatePK } from "@vrooli/shared";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { BulkSeedResult, DbErrorScenarios, DbTestFixtures } from "./types.js";

/**
 * Database fixtures for Bookmark model - used for seeding test data
 * These follow Prisma's shape for database operations
 * 
 * Bookmarks support polymorphic relationships through different object types:
 * Comment, Issue, Resource, Tag, Team, User
 */

// Cached IDs for consistent testing - lazy initialization pattern
let _bookmarkDbIds: Record<string, bigint> | null = null;
export function getBookmarkDbIds() {
    if (!_bookmarkDbIds) {
        _bookmarkDbIds = {
            bookmark1: generatePK(),
            bookmark2: generatePK(),
            bookmark3: generatePK(),
            bookmarkList1: generatePK(),
            bookmarkList2: generatePK(),
            user1: generatePK(),
            user2: generatePK(),
            resource1: generatePK(),
            resource2: generatePK(),
            team1: generatePK(),
            comment1: generatePK(),
            issue1: generatePK(),
            tag1: generatePK(),
        };
    }
    return _bookmarkDbIds;
}

/**
 * Enhanced test fixtures for Bookmark model following standard structure
 */
export const bookmarkDbFixtures: DbTestFixtures<Prisma.bookmarkCreateInput> = {
    minimal: {
        id: generatePK(),
        resource: { connect: { id: getBookmarkDbIds().resource1 } },
    },
    complete: {
        id: generatePK(),
        list: { connect: { id: getBookmarkDbIds().bookmarkList1 } },
        resource: { connect: { id: getBookmarkDbIds().resource1 } },
    },
    invalid: {
        missingRequired: {
            // Missing required id and object connection
            id: undefined,
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            resource: "invalid-resource-reference", // Should be connect object
            list: "invalid-list-reference", // Should be connect object
        },
        noBookmarkableObject: {
            id: generatePK(),
            // No bookmarkable object connected
        },
        multipleObjects: {
            id: generatePK(),
            resource: { connect: { id: getBookmarkDbIds().resource1 } },
            team: { connect: { id: getBookmarkDbIds().team1 } },
            // Multiple objects connected (business logic violation)
        },
    },
    edgeCases: {
        resourceBookmark: {
            id: generatePK(),
            resource: { connect: { id: getBookmarkDbIds().resource1 } },
        },
        teamBookmark: {
            id: generatePK(),
            team: { connect: { id: getBookmarkDbIds().team1 } },
        },
        userBookmark: {
            id: generatePK(),
            user: { connect: { id: getBookmarkDbIds().user2 } },
        },
        commentBookmark: {
            id: generatePK(),
            comment: { connect: { id: getBookmarkDbIds().comment1 } },
        },
        issueBookmark: {
            id: generatePK(),
            issue: { connect: { id: getBookmarkDbIds().issue1 } },
        },
        tagBookmark: {
            id: generatePK(),
            tag: { connect: { id: getBookmarkDbIds().tag1 } },
        },
        inListBookmark: {
            id: generatePK(),
            list: { connect: { id: getBookmarkDbIds().bookmarkList1 } },
            resource: { connect: { id: getBookmarkDbIds().resource1 } },
        },
    },
};

/**
 * Enhanced factory for creating bookmark database fixtures
 */
export class BookmarkDbFactory extends EnhancedDbFactory<Prisma.bookmarkCreateInput> {

    /**
     * Get the test fixtures for Bookmark model
     */
    protected getFixtures(): DbTestFixtures<Prisma.bookmarkCreateInput> {
        return bookmarkDbFixtures;
    }

    /**
     * Get Bookmark-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: getBookmarkDbIds().bookmark1, // Duplicate ID
                    resource: { connect: { id: getBookmarkDbIds().resource1 } },
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    resource: { connect: { id: "non-existent-resource-id" } },
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    // No object connected violates check constraint
                },
            },
            validation: {
                requiredFieldMissing: bookmarkDbFixtures.invalid.missingRequired,
                invalidDataType: bookmarkDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    resource: { connect: { id: getBookmarkDbIds().resource1 } },
                    // No specific out-of-range scenario for bookmark
                },
            },
            businessLogic: {
                multipleObjects: bookmarkDbFixtures.invalid.multipleObjects,
                noBookmarkableObject: bookmarkDbFixtures.invalid.noBookmarkableObject,
            },
        };
    }

    /**
     * Generate fresh identifiers for Bookmark (no publicId)
     */
    protected generateFreshIdentifiers(): Record<string, any> {
        return {
            id: generatePK(),
        };
    }

    /**
     * Bookmark-specific validation
     */
    protected validateSpecific(data: Prisma.bookmarkCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check that exactly one bookmarkable object is connected
        const bookmarkableFields = ["comment", "issue", "resource", "tag", "team", "user"];
        const connectedObjects = bookmarkableFields.filter(field => data[field as keyof Prisma.bookmarkCreateInput]);

        if (connectedObjects.length === 0) {
            errors.push("Bookmark must reference exactly one bookmarkable object");
        } else if (connectedObjects.length > 1) {
            errors.push("Bookmark cannot reference multiple objects");
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(
        objectId: bigint,
        objectType: "Resource" | "Comment" | "Issue" | "Tag" | "Team" | "User",
        overrides?: Partial<Prisma.bookmarkCreateInput>,
    ): Prisma.bookmarkCreateInput {
        const factory = new BookmarkDbFactory();
        const connections: Record<string, any> = {
            Resource: { resource: { connect: { id: objectId } } },
            Comment: { comment: { connect: { id: objectId } } },
            Issue: { issue: { connect: { id: objectId } } },
            Tag: { tag: { connect: { id: objectId } } },
            Team: { team: { connect: { id: objectId } } },
            User: { user: { connect: { id: objectId } } },
        };

        if (!connections[objectType]) {
            throw new Error(`Invalid bookmark object type: ${objectType}`);
        }

        return factory.createMinimal({
            ...connections[objectType],
            ...overrides,
        });
    }

    static createForObject(
        objectId: bigint,
        objectType: "Resource" | "Comment" | "Issue" | "Tag" | "Team" | "User",
        overrides?: Partial<Prisma.bookmarkCreateInput>,
    ): Prisma.bookmarkCreateInput {
        return this.createMinimal(objectId, objectType, overrides);
    }

    static createInList(
        listId: bigint,
        objectId: bigint,
        objectType: "Resource" | "Comment" | "Issue" | "Tag" | "Team" | "User",
        overrides?: Partial<Prisma.bookmarkCreateInput>,
    ): Prisma.bookmarkCreateInput {
        return {
            ...this.createForObject(objectId, objectType, overrides),
            list: { connect: { id: listId } },
        };
    }
}

/**
 * Enhanced helper to seed multiple test bookmarks with comprehensive options
 */
export async function seedBookmarks(
    prisma: any,
    options: {
        objects: Array<{ id: bigint; type: "Resource" | "Comment" | "Issue" | "Tag" | "Team" | "User" }>;
        withList?: boolean;
        listId?: bigint;
        listLabel?: string;
        listUserId?: bigint;
    },
): Promise<BulkSeedResult<any>> {
    const bookmarks = [];
    let listCount = 0;

    if (options.withList && options.listUserId) {
        // Create a bookmark list with all bookmarks
        const list = await prisma.bookmark_list.create({
            data: {
                id: options.listId || generatePK(),
                label: options.listLabel || "My Bookmarks",
                user: { connect: { id: options.listUserId } },
                bookmarks: {
                    create: options.objects.map(o =>
                        BookmarkDbFactory.createForObject(o.id, o.type),
                    ),
                },
            },
            include: { bookmarks: true },
        });
        listCount = 1;
        return {
            records: [list],
            summary: {
                total: 1,
                withAuth: 0,
                bots: 0,
                teams: 0,
                lists: listCount,
                bookmarks: list.bookmarks.length,
            },
        };
    } else {
        // Create individual bookmarks
        for (const obj of options.objects) {
            const bookmark = await prisma.bookmark.create({
                data: BookmarkDbFactory.createForObject(obj.id, obj.type),
            });
            bookmarks.push(bookmark);
        }
        return {
            records: bookmarks,
            summary: {
                total: bookmarks.length,
                withAuth: 0,
                bots: 0,
                teams: 0,
                lists: 0,
                bookmarks: bookmarks.length,
            },
        };
    }
}
