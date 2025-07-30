import { type Prisma } from "@prisma/client";
import { BookmarkFor, generatePK, generatePublicId } from "@vrooli/shared";
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
            project1: generatePK(),
            routine1: generatePK(),
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
        publicId: generatePublicId(),
        by: { connect: { id: getBookmarkDbIds().user1 } },
    },
    complete: {
        id: generatePK(),
        publicId: generatePublicId(),
        by: { connect: { id: getBookmarkDbIds().user1 } },
        list: { connect: { id: getBookmarkDbIds().bookmarkList1 } },
        project: { connect: { id: getBookmarkDbIds().project1 } },
    },
    invalid: {
        missingRequired: {
            // Missing required by (user)
            publicId: generatePublicId(),
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            publicId: 123, // Should be string
            by: "invalid-user-reference", // Should be connect object
            list: "invalid-list-reference", // Should be connect object
        },
        noBookmarkableObject: {
            id: generatePK(),
            publicId: generatePublicId(),
            by: { connect: { id: getBookmarkDbIds().user1 } },
            // No bookmarkable object connected
        },
        multipleObjects: {
            id: generatePK(),
            publicId: generatePublicId(),
            by: { connect: { id: getBookmarkDbIds().user1 } },
            project: { connect: { id: getBookmarkDbIds().project1 } },
            routine: { connect: { id: getBookmarkDbIds().routine1 } },
            // Multiple objects connected (business logic violation)
        },
    },
    edgeCases: {
        projectBookmark: {
            id: generatePK(),
            publicId: generatePublicId(),
            by: { connect: { id: getBookmarkDbIds().user1 } },
            project: { connect: { id: getBookmarkDbIds().project1 } },
        },
        routineBookmark: {
            id: generatePK(),
            publicId: generatePublicId(),
            by: { connect: { id: getBookmarkDbIds().user1 } },
            routine: { connect: { id: getBookmarkDbIds().routine1 } },
        },
        userBookmark: {
            id: generatePK(),
            publicId: generatePublicId(),
            by: { connect: { id: getBookmarkDbIds().user1 } },
            user: { connect: { id: getBookmarkDbIds().user2 } },
        },
        inListBookmark: {
            id: generatePK(),
            publicId: generatePublicId(),
            by: { connect: { id: getBookmarkDbIds().user1 } },
            list: { connect: { id: getBookmarkDbIds().bookmarkList1 } },
            project: { connect: { id: getBookmarkDbIds().project1 } },
        },
    },
};

/**
 * Enhanced test fixtures for BookmarkList model following standard structure
 */
export const bookmarkListDbFixtures: DbTestFixtures<Prisma.BookmarkListCreateInput> = {
    minimal: {
        id: generatePK(),
        publicId: generatePublicId(),
        isPrivate: false,
        createdBy: { connect: { id: getBookmarkDbIds().user1 } },
    },
    complete: {
        id: generatePK(),
        publicId: generatePublicId(),
        isPrivate: false,
        createdBy: { connect: { id: getBookmarkDbIds().user1 } },
        translations: {
            create: [
                {
                    id: generatePK(),
                    language: "en",
                    name: "My Bookmarks",
                    description: "A collection of my favorite items",
                },
                {
                    id: generatePK(),
                    language: "es",
                    name: "Mis Marcadores",
                    description: "Una colección de mis elementos favoritos",
                },
            ],
        },
        bookmarks: {
            create: [
                {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    by: { connect: { id: getBookmarkDbIds().user1 } },
                    project: { connect: { id: getBookmarkDbIds().project1 } },
                },
            ],
        },
    },
    invalid: {
        missingRequired: {
            // Missing required createdBy
            publicId: generatePublicId(),
            isPrivate: false,
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            publicId: 123, // Should be string
            isPrivate: "yes", // Should be boolean
            createdBy: "invalid-user-reference", // Should be connect object
        },
    },
    edgeCases: {
        privateList: {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: true,
            createdBy: { connect: { id: getBookmarkDbIds().user1 } },
        },
        emptyList: {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            createdBy: { connect: { id: getBookmarkDbIds().user1 } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Empty List",
                }],
            },
        },
        multiLanguageList: {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            createdBy: { connect: { id: getBookmarkDbIds().user1 } },
            translations: {
                create: [
                    { id: generatePK(), language: "en", name: "English List" },
                    { id: generatePK(), language: "es", name: "Lista Española" },
                    { id: generatePK(), language: "fr", name: "Liste Française" },
                    { id: generatePK(), language: "de", name: "Deutsche Liste" },
                ],
            },
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
                    publicId: generatePublicId(),
                    by: { connect: { id: getBookmarkDbIds().user1 } },
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    by: { connect: { id: "non-existent-user-id" } },
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    publicId: "", // Empty publicId
                    by: { connect: { id: getBookmarkDbIds().user1 } },
                },
            },
            validation: {
                requiredFieldMissing: bookmarkDbFixtures.invalid.missingRequired,
                invalidDataType: bookmarkDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    publicId: "a".repeat(500), // PublicId too long
                    by: { connect: { id: getBookmarkDbIds().user1 } },
                },
            },
            businessLogic: {
                multipleObjects: bookmarkDbFixtures.invalid.multipleObjects,
                noBookmarkableObject: bookmarkDbFixtures.invalid.noBookmarkableObject,
                selfBookmark: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    by: { connect: { id: getBookmarkDbIds().user1 } },
                    user: { connect: { id: getBookmarkDbIds().user1 } }, // User bookmarking themselves
                },
            },
        };
    }

    /**
     * Bookmark-specific validation
     */
    protected validateSpecific(data: Prisma.bookmarkCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to Bookmark
        if (!data.by) errors.push("Bookmark by (user) is required");
        if (!data.publicId) errors.push("Bookmark publicId is required");

        // Check business logic - must have exactly one bookmarkable object
        const bookmarkableFields = ["api", "code", "comment", "issue", "note", "post", "project", "prompt", "question", "quiz", "routine", "runProject", "runRoutine", "smartContract", "standard", "team", "user"];
        const connectedObjects = bookmarkableFields.filter(field => data[field as keyof Prisma.bookmarkCreateInput]);

        if (connectedObjects.length === 0) {
            errors.push("Bookmark must reference exactly one bookmarkable object");
        } else if (connectedObjects.length > 1) {
            errors.push("Bookmark cannot reference multiple objects");
        }

        // Check for self-bookmark
        if (data.by && data.user &&
            typeof data.by === "object" && "connect" in data.by &&
            typeof data.user === "object" && "connect" in data.user &&
            data.by.connect.id === data.user.connect.id) {
            warnings.push("User is bookmarking themselves");
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(
        byId: string,
        overrides?: Partial<Prisma.bookmarkCreateInput>,
    ): Prisma.bookmarkCreateInput {
        const factory = new BookmarkDbFactory();
        return factory.createMinimal({
            by: { connect: { id: byId } },
            ...overrides,
        });
    }

    static createForObject(
        byId: string,
        objectId: string,
        objectType: BookmarkFor | string,
        overrides?: Partial<Prisma.bookmarkCreateInput>,
    ): Prisma.bookmarkCreateInput {
        const baseBookmark = this.createMinimal(byId, overrides);

        // Add the appropriate connection based on object type
        const connections: Record<string, any> = {
            // BookmarkFor enum values
            [BookmarkFor.Comment]: { comment: { connect: { id: objectId } } },
            [BookmarkFor.Issue]: { issue: { connect: { id: objectId } } },
            [BookmarkFor.Resource]: { resource: { connect: { id: objectId } } },
            [BookmarkFor.Tag]: { tag: { connect: { id: objectId } } },
            [BookmarkFor.Team]: { team: { connect: { id: objectId } } },
            [BookmarkFor.User]: { user: { connect: { id: objectId } } },
            // Additional supported types not in BookmarkFor enum
            Api: { api: { connect: { id: objectId } } },
            Code: { code: { connect: { id: objectId } } },
            Note: { note: { connect: { id: objectId } } },
            Post: { post: { connect: { id: objectId } } },
            Project: { project: { connect: { id: objectId } } },
            Prompt: { prompt: { connect: { id: objectId } } },
            Question: { question: { connect: { id: objectId } } },
            Quiz: { quiz: { connect: { id: objectId } } },
            Routine: { routine: { connect: { id: objectId } } },
            RunProject: { runProject: { connect: { id: objectId } } },
            RunRoutine: { runRoutine: { connect: { id: objectId } } },
            SmartContract: { smartContract: { connect: { id: objectId } } },
            Standard: { standard: { connect: { id: objectId } } },
        };

        if (!connections[objectType]) {
            throw new Error(`Invalid bookmark object type: ${objectType}`);
        }

        return {
            ...baseBookmark,
            ...connections[objectType],
        };
    }

    static createInList(
        byId: string,
        listId: string,
        objectId: string,
        objectType: string,
        overrides?: Partial<Prisma.bookmarkCreateInput>,
    ): Prisma.bookmarkCreateInput {
        return {
            ...this.createForObject(byId, objectId, objectType, overrides),
            list: { connect: { id: listId } },
        };
    }
}

/**
 * Enhanced factory for creating bookmark list database fixtures
 */
export class BookmarkListDbFactory extends EnhancedDbFactory<Prisma.BookmarkListCreateInput> {

    /**
     * Get the test fixtures for BookmarkList model
     */
    protected getFixtures(): DbTestFixtures<Prisma.BookmarkListCreateInput> {
        return bookmarkListDbFixtures;
    }

    /**
     * Get BookmarkList-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: getBookmarkDbIds().bookmarkList1, // Duplicate ID
                    publicId: generatePublicId(),
                    isPrivate: false,
                    createdBy: { connect: { id: getBookmarkDbIds().user1 } },
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    createdBy: { connect: { id: "non-existent-user-id" } },
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    publicId: "", // Empty publicId
                    isPrivate: false,
                    createdBy: { connect: { id: getBookmarkDbIds().user1 } },
                },
            },
            validation: {
                requiredFieldMissing: bookmarkListDbFixtures.invalid.missingRequired,
                invalidDataType: bookmarkListDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    publicId: "a".repeat(500), // PublicId too long
                    isPrivate: false,
                    createdBy: { connect: { id: getBookmarkDbIds().user1 } },
                },
            },
            businessLogic: {},
        };
    }

    /**
     * BookmarkList-specific validation
     */
    protected validateSpecific(data: Prisma.BookmarkListCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to BookmarkList
        if (!data.createdBy) errors.push("BookmarkList createdBy is required");
        if (!data.publicId) errors.push("BookmarkList publicId is required");
        if (data.isPrivate === undefined) errors.push("BookmarkList isPrivate is required");

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(
        createdById: string,
        overrides?: Partial<Prisma.BookmarkListCreateInput>,
    ): Prisma.BookmarkListCreateInput {
        const factory = new BookmarkListDbFactory();
        return factory.createMinimal({
            createdBy: { connect: { id: createdById } },
            isPrivate: false,
            ...overrides,
        });
    }

    static createWithTranslations(
        createdById: string,
        translations: Array<{ language: string; name: string; description?: string }>,
        overrides?: Partial<Prisma.BookmarkListCreateInput>,
    ): Prisma.BookmarkListCreateInput {
        return {
            ...this.createMinimal(createdById, overrides),
            translations: {
                create: translations.map(t => ({
                    id: generatePK(),
                    language: t.language,
                    name: t.name,
                    description: t.description,
                })),
            },
        };
    }

    static createWithBookmarks(
        createdById: string,
        bookmarks: Array<{ objectId: string; objectType: BookmarkFor | string }>,
        overrides?: Partial<Prisma.BookmarkListCreateInput>,
    ): Prisma.BookmarkListCreateInput {
        return {
            ...this.createWithTranslations(
                createdById,
                [{ language: "en", name: "Test Bookmark List" }],
                overrides,
            ),
            bookmarks: {
                create: bookmarks.map(b =>
                    BookmarkDbFactory.createForObject(createdById, b.objectId, b.objectType),
                ),
            },
        };
    }

    /**
     * Create a private bookmark list
     */
    static createPrivateList(
        createdById: string,
        overrides?: Partial<Prisma.BookmarkListCreateInput>,
    ): Prisma.BookmarkListCreateInput {
        return this.createMinimal(createdById, {
            isPrivate: true,
            ...overrides,
        });
    }

    /**
     * Create a shared bookmark list with collaborators
     */
    static createSharedList(
        createdById: string,
        name: string,
        collaboratorIds: string[],
        overrides?: Partial<Prisma.BookmarkListCreateInput>,
    ): Prisma.BookmarkListCreateInput {
        return this.createWithTranslations(
            createdById,
            [{ language: "en", name }],
            {
                isPrivate: false,
                ...overrides,
            },
        );
    }
}

/**
 * Enhanced helper to seed multiple test bookmarks with comprehensive options
 */
export async function seedBookmarks(
    prisma: any,
    options: {
        userId: string;
        objects: Array<{ id: string; type: string }>;
        withList?: boolean;
        listName?: string;
    },
): Promise<BulkSeedResult<any>> {
    const bookmarks = [];
    let listCount = 0;

    if (options.withList) {
        // Create a bookmark list with all bookmarks
        const list = await prisma.bookmarkList.create({
            data: BookmarkListDbFactory.createWithBookmarks(
                options.userId,
                options.objects.map(o => ({ objectId: o.id, objectType: o.type })),
                {
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            name: options.listName || "My Bookmarks",
                        }],
                    },
                },
            ),
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
                data: BookmarkDbFactory.createForObject(
                    options.userId,
                    obj.id,
                    obj.type,
                ),
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
