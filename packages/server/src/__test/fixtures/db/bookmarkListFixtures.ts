import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

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
 * Minimal bookmark list data for database creation
 */
export const minimalBookmarkListDb: Prisma.bookmark_listCreateInput = {
    id: bookmarkListDbIds.list1,
    label: "My Bookmarks",
    user: { connect: { id: generatePK() } }, // Will be overridden in factory
};

/**
 * Bookmark list with standard label
 */
export const standardBookmarkListDb: Prisma.bookmark_listCreateInput = {
    id: bookmarkListDbIds.list2,
    label: "Favorite Resources",
    user: { connect: { id: generatePK() } },
};

/**
 * Complete bookmark list with bookmarks
 */
export const completeBookmarkListDb: Prisma.bookmark_listCreateInput = {
    id: bookmarkListDbIds.list3,
    label: "Research Collection",
    user: { connect: { id: generatePK() } },
    bookmarks: {
        create: [
            {
                id: generatePK(),
                publicId: generatePublicId(),
                by: { connect: { id: generatePK() } }, // Same as list owner
            },
        ],
    },
};

/**
 * Factory for creating bookmark list database fixtures with overrides
 */
export class BookmarkListDbFactory {
    /**
     * Create minimal bookmark list
     */
    static createMinimal(
        userId: string,
        overrides?: Partial<Prisma.bookmark_listCreateInput>
    ): Prisma.bookmark_listCreateInput {
        return {
            id: generatePK(),
            label: overrides?.label || "My Bookmarks",
            user: { connect: { id: userId } },
            ...overrides,
        };
    }

    /**
     * Create bookmark list with custom label
     */
    static createWithLabel(
        userId: string,
        label: string,
        overrides?: Partial<Prisma.bookmark_listCreateInput>
    ): Prisma.bookmark_listCreateInput {
        return {
            ...this.createMinimal(userId, overrides),
            label,
        };
    }

    /**
     * Create bookmark list with bookmarks
     */
    static createWithBookmarks(
        userId: string,
        bookmarks: Array<{
            objectId: string;
            objectType: "Api" | "Code" | "Comment" | "Issue" | "Note" | "Post" | 
                       "Project" | "Prompt" | "Question" | "Quiz" | "Routine" | 
                       "RunProject" | "RunRoutine" | "SmartContract" | "Standard" | 
                       "Team" | "User";
        }>,
        overrides?: Partial<Prisma.bookmark_listCreateInput>
    ): Prisma.bookmark_listCreateInput {
        const bookmarkConnections: Record<string, (id: string) => any> = {
            Api: (id) => ({ api: { connect: { id } } }),
            Code: (id) => ({ code: { connect: { id } } }),
            Comment: (id) => ({ comment: { connect: { id } } }),
            Issue: (id) => ({ issue: { connect: { id } } }),
            Note: (id) => ({ note: { connect: { id } } }),
            Post: (id) => ({ post: { connect: { id } } }),
            Project: (id) => ({ project: { connect: { id } } }),
            Prompt: (id) => ({ prompt: { connect: { id } } }),
            Question: (id) => ({ question: { connect: { id } } }),
            Quiz: (id) => ({ quiz: { connect: { id } } }),
            Routine: (id) => ({ routine: { connect: { id } } }),
            RunProject: (id) => ({ runProject: { connect: { id } } }),
            RunRoutine: (id) => ({ runRoutine: { connect: { id } } }),
            SmartContract: (id) => ({ smartContract: { connect: { id } } }),
            Standard: (id) => ({ standard: { connect: { id } } }),
            Team: (id) => ({ team: { connect: { id } } }),
            User: (id) => ({ user: { connect: { id } } }),
        };

        return {
            ...this.createMinimal(userId, overrides),
            bookmarks: {
                create: bookmarks.map(b => ({
                    id: generatePK(),
                    publicId: generatePublicId(),
                    by: { connect: { id: userId } },
                    ...bookmarkConnections[b.objectType](b.objectId),
                })),
            },
        };
    }

    /**
     * Create multiple bookmark lists for a user
     */
    static createMultiple(
        userId: string,
        count: number,
        baseLabel: string = "List"
    ): Array<Prisma.bookmark_listCreateInput> {
        return Array.from({ length: count }, (_, i) => 
            this.createMinimal(userId, {
                label: `${baseLabel} ${i + 1}`,
            })
        );
    }

    /**
     * Create bookmark list with existing bookmarks
     */
    static createWithExistingBookmarks(
        userId: string,
        bookmarkIds: string[],
        overrides?: Partial<Prisma.bookmark_listCreateInput>
    ): Prisma.bookmark_listCreateInput {
        return {
            ...this.createMinimal(userId, overrides),
            bookmarks: {
                connect: bookmarkIds.map(id => ({ id })),
            },
        };
    }
}

/**
 * Scenario-based bookmark list fixtures
 */
export const bookmarkListScenarios = {
    /**
     * Empty bookmark list
     */
    empty: (userId: string): Prisma.bookmark_listCreateInput => ({
        id: generatePK(),
        label: "Empty List",
        user: { connect: { id: userId } },
    }),

    /**
     * Research collection with multiple types
     */
    researchCollection: (userId: string, objectIds: {
        projectId?: string;
        routineId?: string;
        standardId?: string;
    }): Prisma.bookmark_listCreateInput => {
        const bookmarks = [];
        if (objectIds.projectId) {
            bookmarks.push({ objectId: objectIds.projectId, objectType: "Project" as const });
        }
        if (objectIds.routineId) {
            bookmarks.push({ objectId: objectIds.routineId, objectType: "Routine" as const });
        }
        if (objectIds.standardId) {
            bookmarks.push({ objectId: objectIds.standardId, objectType: "Standard" as const });
        }
        
        return BookmarkListDbFactory.createWithBookmarks(
            userId,
            bookmarks,
            { label: "Research Collection" }
        );
    },

    /**
     * Tutorial resources list
     */
    tutorials: (userId: string, routineIds: string[]): Prisma.bookmark_listCreateInput => {
        return BookmarkListDbFactory.createWithBookmarks(
            userId,
            routineIds.map(id => ({ objectId: id, objectType: "Routine" as const })),
            { label: "Tutorial Routines" }
        );
    },

    /**
     * Team resources list
     */
    teamResources: (userId: string, teamId: string, resourceIds: {
        projectIds?: string[];
        routineIds?: string[];
    }): Prisma.bookmark_listCreateInput => {
        const bookmarks = [];
        
        if (resourceIds.projectIds) {
            bookmarks.push(...resourceIds.projectIds.map(id => ({
                objectId: id,
                objectType: "Project" as const,
            })));
        }
        
        if (resourceIds.routineIds) {
            bookmarks.push(...resourceIds.routineIds.map(id => ({
                objectId: id,
                objectType: "Routine" as const,
            })));
        }
        
        return BookmarkListDbFactory.createWithBookmarks(
            userId,
            bookmarks,
            { label: `Team ${teamId} Resources` }
        );
    },
};

/**
 * Helper to seed bookmark lists for testing
 */
export async function seedBookmarkLists(
    prisma: any,
    options: {
        userId: string;
        count?: number;
        withBookmarks?: boolean;
        bookmarkObjects?: Array<{ id: string; type: string }>;
        labels?: string[];
    }
) {
    const lists = [];
    const count = options.count || 1;
    const labels = options.labels || Array.from({ length: count }, (_, i) => `List ${i + 1}`);

    for (let i = 0; i < count; i++) {
        let listData: Prisma.bookmark_listCreateInput;
        
        if (options.withBookmarks && options.bookmarkObjects) {
            // Create list with bookmarks
            listData = BookmarkListDbFactory.createWithBookmarks(
                options.userId,
                options.bookmarkObjects as any,
                { label: labels[i] }
            );
        } else {
            // Create empty list
            listData = BookmarkListDbFactory.createMinimal(
                options.userId,
                { label: labels[i] }
            );
        }

        const list = await prisma.bookmark_list.create({
            data: listData,
            include: {
                bookmarks: true,
                user: true,
            },
        });
        
        lists.push(list);
    }

    return lists;
}

/**
 * Helper to clean up bookmark lists
 */
export async function cleanupBookmarkLists(
    prisma: any,
    listIds: string[]
) {
    // Delete bookmarks first (if not cascade deleted)
    await prisma.bookmark.deleteMany({
        where: { listId: { in: listIds } },
    });
    
    // Then delete lists
    await prisma.bookmark_list.deleteMany({
        where: { id: { in: listIds } },
    });
}