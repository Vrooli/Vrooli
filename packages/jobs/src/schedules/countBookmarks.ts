import { DbProvider, batch, logger } from "@local/server";
import type { ModelType } from "@local/shared";
import { camelCase, uppercaseFirstLetter } from "@local/shared";
import type { Prisma } from "@prisma/client";
import pkg from "@prisma/client";

const { PrismaClient } = pkg;

// Declare select shape for bookmarks (with relation count)
const bookmarkSelect = {
    id: true,
    bookmarks: true,
    _count: { select: { bookmarkedBy: true } },
} as const;

// Union payload type for dynamic table names
type BookmarkPayload =
    Prisma.commentGetPayload<{ select: typeof bookmarkSelect }> |
    Prisma.issueGetPayload<{ select: typeof bookmarkSelect }> |
    Prisma.resourceGetPayload<{ select: typeof bookmarkSelect }> |
    Prisma.tagGetPayload<{ select: typeof bookmarkSelect }> |
    Prisma.teamGetPayload<{ select: typeof bookmarkSelect }> |
    Prisma.userGetPayload<{ select: typeof bookmarkSelect }>;

// Union FindManyArgs type for bookmark tables
type BookmarkFindManyArgs =
    Prisma.commentFindManyArgs |
    Prisma.issueFindManyArgs |
    Prisma.resourceFindManyArgs |
    Prisma.tagFindManyArgs |
    Prisma.teamFindManyArgs |
    Prisma.userFindManyArgs;

async function processTableInBatches(tableName: keyof InstanceType<typeof PrismaClient>): Promise<void> {
    try {
        await batch<BookmarkFindManyArgs, BookmarkPayload>({
            objectType: uppercaseFirstLetter(camelCase(tableName as string)) as ModelType,
            processBatch: async (batch) => {
                for (const item of batch) {
                    const actualCount = item._count.bookmarkedBy;
                    if (item.bookmarks !== actualCount) {
                        logger.warning(`Updating ${tableName as string} ${item.id} bookmarks from ${item.bookmarks} to ${actualCount}.`, { trace: "0165" });
                        await (DbProvider.get()[tableName] as { update: any }).update({
                            where: { id: item.id },
                            data: { bookmarks: actualCount },
                        });
                    }
                }
            },
            select: bookmarkSelect,
        });
    } catch (error) {
        logger.error("processTableInBatches caught error", { error, trace: "0166" });
    }
}

export async function countBookmarks(): Promise<void> {
    const tableNames = [
        // "bookmark_list",
        "comment",
        "issue",
        "resource",
        "tag",
        "team",
        "user",
    ] as const;

    for (const tableName of tableNames) {
        await processTableInBatches(tableName);
    }
}
