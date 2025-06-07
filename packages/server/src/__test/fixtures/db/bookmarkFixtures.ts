import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for Bookmark model - used for seeding test data
 */

export class BookmarkDbFactory {
    static createMinimal(
        byId: string,
        overrides?: Partial<Prisma.BookmarkCreateInput>
    ): Prisma.BookmarkCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            by: { connect: { id: byId } },
            ...overrides,
        };
    }

    static createForObject(
        byId: string,
        objectId: string,
        objectType: string,
        overrides?: Partial<Prisma.BookmarkCreateInput>
    ): Prisma.BookmarkCreateInput {
        const baseBookmark = this.createMinimal(byId, overrides);
        
        // Add the appropriate connection based on object type
        const connections: Record<string, any> = {
            Api: { api: { connect: { id: objectId } } },
            Code: { code: { connect: { id: objectId } } },
            Comment: { comment: { connect: { id: objectId } } },
            Issue: { issue: { connect: { id: objectId } } },
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
            Team: { team: { connect: { id: objectId } } },
            User: { user: { connect: { id: objectId } } },
        };

        return {
            ...baseBookmark,
            ...(connections[objectType] || {}),
        };
    }

    static createInList(
        byId: string,
        listId: string,
        objectId: string,
        objectType: string,
        overrides?: Partial<Prisma.BookmarkCreateInput>
    ): Prisma.BookmarkCreateInput {
        return {
            ...this.createForObject(byId, objectId, objectType, overrides),
            list: { connect: { id: listId } },
        };
    }
}

/**
 * Database fixtures for BookmarkList model
 */
export class BookmarkListDbFactory {
    static createMinimal(
        createdById: string,
        overrides?: Partial<Prisma.BookmarkListCreateInput>
    ): Prisma.BookmarkListCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            createdBy: { connect: { id: createdById } },
            ...overrides,
        };
    }

    static createWithTranslations(
        createdById: string,
        translations: Array<{ language: string; name: string; description?: string }>,
        overrides?: Partial<Prisma.BookmarkListCreateInput>
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
        bookmarks: Array<{ objectId: string; objectType: string }>,
        overrides?: Partial<Prisma.BookmarkListCreateInput>
    ): Prisma.BookmarkListCreateInput {
        return {
            ...this.createWithTranslations(
                createdById,
                [{ language: "en", name: "Test Bookmark List" }],
                overrides
            ),
            bookmarks: {
                create: bookmarks.map(b => 
                    BookmarkDbFactory.createForObject(createdById, b.objectId, b.objectType)
                ),
            },
        };
    }
}

/**
 * Helper to seed bookmarks for testing
 */
export async function seedBookmarks(
    prisma: any,
    options: {
        userId: string;
        objects: Array<{ id: string; type: string }>;
        withList?: boolean;
        listName?: string;
    }
) {
    const bookmarks = [];

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
                }
            ),
            include: { bookmarks: true },
        });
        return { list, bookmarks: list.bookmarks };
    } else {
        // Create individual bookmarks
        for (const obj of options.objects) {
            const bookmark = await prisma.bookmark.create({
                data: BookmarkDbFactory.createForObject(
                    options.userId,
                    obj.id,
                    obj.type
                ),
            });
            bookmarks.push(bookmark);
        }
        return { bookmarks };
    }
}