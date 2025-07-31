import { type Prisma } from "@prisma/client";
import { generatePK } from "@vrooli/shared";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { BulkSeedOptions, BulkSeedResult, DbErrorScenarios, DbTestFixtures } from "./types.js";

/**
 * Database fixtures for BookmarkList model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const bookmarkListDbIds = {
    list1: generatePK(),
    list2: generatePK(),
    list3: generatePK(),
    list4: generatePK(),
    list5: generatePK(),
};

/**
 * Enhanced test fixtures for BookmarkList model following standard structure
 */
export const bookmarkListDbFixtures: DbTestFixtures<Prisma.bookmark_listCreateInput> = {
    minimal: {
        id: generatePK(),
        label: "My Bookmarks",
        user: { connect: { id: generatePK() } },
    },
    complete: {
        id: generatePK(),
        label: "Complete Research Collection",
        user: { connect: { id: generatePK() } },
        bookmarks: {
            create: [
                {
                    id: generatePK(),
                    resource: { connect: { id: generatePK() } },
                },
                {
                    id: generatePK(),
                    team: { connect: { id: generatePK() } },
                },
                {
                    id: generatePK(),
                    user: { connect: { id: generatePK() } },
                },
            ],
        },
    },
    invalid: {
        missingRequired: {
            // Missing required id, label, user
            id: undefined,
            label: undefined,
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            label: 123, // Should be string
            user: "invalid-user-reference", // Should be connection object
        },
        emptyLabel: {
            id: generatePK(),
            label: "", // Empty label should be invalid
            user: { connect: { id: generatePK() } },
        },
        duplicateLabel: {
            id: generatePK(),
            label: "duplicate_label", // Same label for same user
            user: { connect: { id: generatePK() } },
        },
        invalidBookmarkReference: {
            id: generatePK(),
            label: "Invalid Bookmark List",
            user: { connect: { id: generatePK() } },
            bookmarks: {
                create: [{
                    id: generatePK(),
                    resource: { connect: { id: generatePK() } },
                }],
            },
        },
    },
    edgeCases: {
        maxLengthLabel: {
            id: generatePK(),
            label: "a".repeat(128), // Maximum label length
            user: { connect: { id: generatePK() } },
        },
        manyBookmarks: {
            id: generatePK(),
            label: "Large Collection",
            user: { connect: { id: generatePK() } },
            bookmarks: {
                create: Array.from({ length: 50 }, () => ({
                    id: generatePK(),
                    resource: { connect: { id: generatePK() } },
                })),
            },
        },
        allBookmarkTypes: {
            id: generatePK(),
            label: "All Types Collection",
            user: { connect: { id: generatePK() } },
            bookmarks: {
                create: [
                    {
                        id: generatePK(),
                        resource: { connect: { id: generatePK() } },
                    },
                    {
                        id: generatePK(),
                        comment: { connect: { id: generatePK() } },
                    },
                    {
                        id: generatePK(),
                        issue: { connect: { id: generatePK() } },
                    },
                    {
                        id: generatePK(),
                        tag: { connect: { id: generatePK() } },
                    },
                    {
                        id: generatePK(),
                        team: { connect: { id: generatePK() } },
                    },
                    {
                        id: generatePK(),
                        user: { connect: { id: generatePK() } },
                    },
                ],
            },
        },
        specialCharactersLabel: {
            id: generatePK(),
            label: "📚 My Favorites! @#$%^&*()[]{}\"'`~<>?/|+=_-",
            user: { connect: { id: generatePK() } },
        },
        unicodeLabel: {
            id: generatePK(),
            label: "私のブックマーク 📖 мои закладки 🔖 mes favoris",
            user: { connect: { id: generatePK() } },
        },
    },
};

/**
 * Enhanced factory for creating bookmark list database fixtures
 */
export class BookmarkListDbFactory extends EnhancedDbFactory<Prisma.bookmark_listCreateInput> {

    /**
     * Get the test fixtures for BookmarkList model
     */
    protected getFixtures(): DbTestFixtures<Prisma.bookmark_listCreateInput> {
        return bookmarkListDbFixtures;
    }

    /**
     * Get BookmarkList-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: bookmarkListDbIds.list1, // Duplicate ID
                    label: "Duplicate ID List",
                    user: { connect: { id: generatePK() } },
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    label: "Foreign Key Violation List",
                    user: { connect: { id: generatePK() } },
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    label: "", // Empty label violates check constraint
                    user: { connect: { id: generatePK() } },
                },
            },
            validation: {
                requiredFieldMissing: bookmarkListDbFixtures.invalid.missingRequired,
                invalidDataType: bookmarkListDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    label: "a".repeat(500), // Label too long
                    user: { connect: { id: generatePK() } },
                },
            },
            businessLogic: {
                duplicateLabelSameUser: {
                    id: generatePK(),
                    label: "same_label", // Same label for same user (business rule violation)
                    user: { connect: { id: generatePK() } },
                },
                bookmarkWithoutObject: {
                    id: generatePK(),
                    label: "Invalid Bookmark List",
                    user: { connect: { id: generatePK() } },
                    bookmarks: {
                        create: [{
                            id: generatePK(),
                            // No object connected - violates business logic
                        }],
                    },
                },
                bookmarkMultipleObjects: {
                    id: generatePK(),
                    label: "Multiple Objects Bookmark List",
                    user: { connect: { id: generatePK() } },
                    bookmarks: {
                        create: [{
                            id: generatePK(),
                            resource: { connect: { id: generatePK() } },
                            comment: { connect: { id: generatePK() } }, // Multiple objects connected
                        }],
                    },
                },
            },
        };
    }

    /**
     * Generate fresh identifiers for BookmarkList (no publicId or handle)
     */
    protected generateFreshIdentifiers(): Record<string, any> {
        return {
            id: generatePK(),
        };
    }

    /**
     * BookmarkList-specific validation
     */
    protected validateSpecific(data: Prisma.bookmark_listCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to BookmarkList
        if (!data.label) errors.push("BookmarkList label is required");
        if (!data.user) errors.push("BookmarkList user is required");

        // Check label format and length
        if (data.label) {
            if (typeof data.label !== "string") {
                errors.push("Label must be a string");
            } else {
                if (data.label.length === 0) {
                    errors.push("Label cannot be empty");
                }
                if (data.label.length > 128) {
                    errors.push("Label must be 128 characters or less");
                }
                if (data.label.trim() !== data.label) {
                    warnings.push("Label has leading or trailing whitespace");
                }
            }
        }

        // Check bookmarks if present
        if (data.bookmarks && "create" in data.bookmarks && Array.isArray(data.bookmarks.create)) {
            const bookmarks = data.bookmarks.create;
            if (bookmarks.length > 100) {
                warnings.push("Large number of bookmarks may affect performance");
            }

            bookmarks.forEach((bookmark, index) => {
                // Check that exactly one object type is connected
                const objectConnections = [
                    "resource", "comment", "issue", "tag", "team", "user",
                ].filter(key => bookmark[key as keyof typeof bookmark]);

                if (objectConnections.length === 0) {
                    errors.push(`Bookmark ${index} must connect to at least one object`);
                } else if (objectConnections.length > 1) {
                    errors.push(`Bookmark ${index} cannot connect to multiple objects`);
                }
            });
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(
        userId: bigint,
        overrides?: Partial<Prisma.bookmark_listCreateInput>,
    ): Prisma.bookmark_listCreateInput {
        const factory = new BookmarkListDbFactory();
        return factory.createMinimal({
            label: overrides?.label || "My Bookmarks",
            user: { connect: { id: userId } },
            ...overrides,
        });
    }

    /**
     * Create bookmark list with custom label
     */
    static createWithLabel(
        userId: bigint,
        label: string,
        overrides?: Partial<Prisma.bookmark_listCreateInput>,
    ): Prisma.bookmark_listCreateInput {
        return this.createMinimal(userId, { label, ...overrides });
    }

    /**
     * Create bookmark list with bookmarks
     */
    static createWithBookmarks(
        userId: bigint,
        bookmarks: Array<{
            objectId: bigint;
            objectType: "Resource" | "Comment" | "Issue" | "Tag" | "Team" | "User";
        }>,
        overrides?: Partial<Prisma.bookmark_listCreateInput>,
    ): Prisma.bookmark_listCreateInput {
        const bookmarkConnections: Record<string, (id: bigint) => any> = {
            Resource: (id) => ({ resource: { connect: { id } } }),
            Comment: (id) => ({ comment: { connect: { id } } }),
            Issue: (id) => ({ issue: { connect: { id } } }),
            Tag: (id) => ({ tag: { connect: { id } } }),
            Team: (id) => ({ team: { connect: { id } } }),
            User: (id) => ({ user: { connect: { id } } }),
        };

        return {
            ...this.createMinimal(userId, overrides),
            bookmarks: {
                create: bookmarks.map(b => ({
                    id: generatePK(),
                    ...bookmarkConnections[b.objectType](b.objectId),
                })),
            },
        };
    }

    /**
     * Create multiple bookmark lists for a user
     */
    static createMultiple(
        userId: bigint,
        count: number,
        baseLabel = "List",
    ): Array<Prisma.bookmark_listCreateInput> {
        const factory = new BookmarkListDbFactory();
        return factory.createBatch(
            Array.from({ length: count }, (_, i) => ({
                type: "minimal",
                overrides: {
                    label: `${baseLabel} ${i + 1}`,
                    user: { connect: { id: userId } },
                },
            })),
        );
    }

    /**
     * Create bookmark list with existing bookmarks
     */
    static createWithExistingBookmarks(
        userId: bigint,
        bookmarkIds: bigint[],
        overrides?: Partial<Prisma.bookmark_listCreateInput>,
    ): Prisma.bookmark_listCreateInput {
        return this.createMinimal(userId, {
            bookmarks: {
                connect: bookmarkIds.map(id => ({ id })),
            },
            ...overrides,
        });
    }

    static createComplete(overrides?: Partial<Prisma.bookmark_listCreateInput>): Prisma.bookmark_listCreateInput {
        const factory = new BookmarkListDbFactory();
        return factory.createComplete(overrides);
    }

    static createEdgeCase(scenario: string): Prisma.bookmark_listCreateInput {
        const factory = new BookmarkListDbFactory();
        return factory.createEdgeCase(scenario);
    }

    static createInvalid(scenario: string): any {
        const factory = new BookmarkListDbFactory();
        return factory.createInvalid(scenario);
    }
}

/**
 * Scenario-based bookmark list fixtures
 */
export const bookmarkListScenarios = {
    /**
     * Empty bookmark list
     */
    empty: (userId: bigint): Prisma.bookmark_listCreateInput => ({
        id: generatePK(),
        label: "Empty List",
        user: { connect: { id: userId } },
    }),

    /**
     * Research collection with multiple types
     */
    researchCollection: (userId: bigint, objectIds: {
        resourceId?: bigint;
        teamId?: bigint;
        tagId?: bigint;
    }): Prisma.bookmark_listCreateInput => {
        const bookmarks = [];
        if (objectIds.resourceId) {
            bookmarks.push({ objectId: objectIds.resourceId, objectType: "Resource" as const });
        }
        if (objectIds.teamId) {
            bookmarks.push({ objectId: objectIds.teamId, objectType: "Team" as const });
        }
        if (objectIds.tagId) {
            bookmarks.push({ objectId: objectIds.tagId, objectType: "Tag" as const });
        }

        return BookmarkListDbFactory.createWithBookmarks(
            userId,
            bookmarks,
            { label: "Research Collection" },
        );
    },

    /**
     * Tutorial resources list
     */
    tutorials: (userId: bigint, resourceIds: bigint[]): Prisma.bookmark_listCreateInput => {
        return BookmarkListDbFactory.createWithBookmarks(
            userId,
            resourceIds.map(id => ({ objectId: id, objectType: "Resource" as const })),
            { label: "Tutorial Resources" },
        );
    },

    /**
     * Team resources list
     */
    teamResources: (userId: bigint, teamId: bigint, resourceIds: {
        resourceIds?: bigint[];
        commentIds?: bigint[];
    }): Prisma.bookmark_listCreateInput => {
        const bookmarks = [];

        if (resourceIds.resourceIds) {
            bookmarks.push(...resourceIds.resourceIds.map(id => ({
                objectId: id,
                objectType: "Resource" as const,
            })));
        }

        if (resourceIds.commentIds) {
            bookmarks.push(...resourceIds.commentIds.map(id => ({
                objectId: id,
                objectType: "Comment" as const,
            })));
        }

        return BookmarkListDbFactory.createWithBookmarks(
            userId,
            bookmarks,
            { label: `Team ${teamId} Resources` },
        );
    },
};

/**
 * Enhanced helper to seed multiple bookmark lists with comprehensive options
 */
export async function seedBookmarkLists(
    prisma: any,
    userId: bigint,
    count = 3,
    options?: BulkSeedOptions & {
        withBookmarks?: boolean;
        bookmarkObjects?: Array<{ id: bigint; type: string }>;
        labels?: string[];
    },
): Promise<BulkSeedResult<any>> {
    const factory = new BookmarkListDbFactory();
    const lists = [];
    const actualCount = options?.count || count;
    const labels = options?.labels || Array.from({ length: actualCount }, (_, i) => `List ${i + 1}`);

    for (let i = 0; i < actualCount; i++) {
        const baseOverrides = options?.overrides?.[i] || { label: labels[i] };
        const overrides = {
            ...baseOverrides,
            user: { connect: { id: userId } },
        };

        let listData: Prisma.bookmark_listCreateInput;

        if (options?.withBookmarks && options.bookmarkObjects) {
            // Create list with bookmarks
            listData = BookmarkListDbFactory.createWithBookmarks(
                userId,
                options.bookmarkObjects as any,
                overrides,
            );
        } else {
            // Create empty list
            listData = factory.createMinimal(overrides);
        }

        const list = await prisma.bookmark_list.create({
            data: listData,
            include: {
                bookmarks: {
                    include: {
                        api: true,
                        code: true,
                        comment: true,
                        issue: true,
                        note: true,
                        post: true,
                        project: true,
                        prompt: true,
                        question: true,
                        quiz: true,
                        routine: true,
                        runProject: true,
                        runRoutine: true,
                        smartContract: true,
                        standard: true,
                        team: true,
                        user: true,
                    },
                },
                user: true,
            },
        });

        lists.push(list);
    }

    return {
        records: lists,
        summary: {
            total: lists.length,
            withAuth: 0, // Not applicable for bookmark lists
            bots: 0, // Not applicable for bookmark lists
            teams: 0, // Not applicable for bookmark lists
        },
    };
}

/**
 * Helper to clean up bookmark lists and their associated bookmarks
 */
export async function cleanupBookmarkLists(
    prisma: any,
    listIds: bigint[],
) {
    if (listIds.length === 0) return;

    try {
        // Delete bookmarks first (if not cascade deleted)
        await prisma.bookmark.deleteMany({
            where: { listId: { in: listIds } },
        });

        // Then delete lists
        const result = await prisma.bookmark_list.deleteMany({
            where: { id: { in: listIds } },
        });

        return result;
    } catch (error) {
        console.error("Error cleaning up bookmark lists:", error);
        throw error;
    }
}

// Legacy exports for backward compatibility
export const minimalBookmarkListDb: Prisma.bookmark_listCreateInput = {
    id: bookmarkListDbIds.list1,
    label: "My Bookmarks",
    user: { connect: { id: generatePK() } },
};

export const standardBookmarkListDb: Prisma.bookmark_listCreateInput = {
    id: bookmarkListDbIds.list2,
    label: "Favorite Resources",
    user: { connect: { id: generatePK() } },
};

export const completeBookmarkListDb: Prisma.bookmark_listCreateInput = {
    id: bookmarkListDbIds.list3,
    label: "Research Collection",
    user: { connect: { id: generatePK() } },
    bookmarks: {
        create: [
            {
                id: generatePK(),
                resource: { connect: { id: generatePK() } },
            },
        ],
    },
};
