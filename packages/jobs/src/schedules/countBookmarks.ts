import type { Prisma } from "@prisma/client";
import { DbProvider, batch, logger } from "@vrooli/server";

// Define the specific table names that this job processes
const PROCESSED_BOOKMARK_TABLE_NAMES = [
    "comment",
    "issue",
    "resource",
    "tag",
    "team",
    "user",
] as const;

// Create a union type from these table names
type ProcessableBookmarkTableName = typeof PROCESSED_BOOKMARK_TABLE_NAMES[number];

// Map table names to their corresponding ModelType.
// This assumes ModelType from "@vrooli/shared" is a union of capitalized model names
// like "Comment", "Issue", "User", etc.
const TABLE_NAME_TO_MODEL_TYPE_MAP = {
    "comment": "Comment",
    "issue": "Issue",
    "resource": "Resource",
    "tag": "Tag",
    "team": "Team",
    "user": "User",
} as const;

// Declare select shape for bookmarks (with relation count)
const bookmarkSelect = {
    id: true,
    bookmarks: true,
    _count: { select: { bookmarkedBy: true } },
} as const;

// Union payload type for dynamic table names
interface BookmarkPayloadItem {
    id: bigint;
    bookmarks: number | null;
    _count: { bookmarkedBy: number };
}

// Union FindManyArgs type for bookmark tables
type BookmarkFindManyArgs =
    Prisma.commentFindManyArgs |
    Prisma.issueFindManyArgs |
    Prisma.resourceFindManyArgs |
    Prisma.tagFindManyArgs |
    Prisma.teamFindManyArgs |
    Prisma.userFindManyArgs;

async function processTableInBatches(tableName: ProcessableBookmarkTableName): Promise<void> {
    try {
        await batch<BookmarkFindManyArgs, BookmarkPayloadItem>({
            objectType: TABLE_NAME_TO_MODEL_TYPE_MAP[tableName],
            processBatch: async (batchItems) => {
                const db = DbProvider.get();
                for (const item of batchItems) {
                    const actualCount = item._count.bookmarkedBy;
                    try {
                        if (item.bookmarks !== actualCount) {
                            logger.warning(`Updating ${tableName} ${item.id} bookmarks from ${item.bookmarks} to ${actualCount}.`, { trace: "0165" });
                            const update = {
                                where: { id: item.id },
                                data: { bookmarks: actualCount },
                            } as const;
                            switch (tableName) {
                                case "comment":
                                    await db.comment.update(update);
                                    break;
                                case "issue":
                                    await db.issue.update(update);
                                    break;
                                case "resource":
                                    await db.resource.update(update);
                                    break;
                                case "tag":
                                    await db.tag.update(update);
                                    break;
                                case "team":
                                    await db.team.update(update);
                                    break;
                                case "user":
                                    await db.user.update(update);
                                    break;
                                default: {
                                    const _exhaustiveCheck: never = tableName;
                                    logger.error(`Unhandled table name: ${_exhaustiveCheck} in countBookmarks`, { trace: "0167" });
                                    throw new Error(`Unhandled table name: ${_exhaustiveCheck}`);
                                }
                            }
                        }
                    } catch (updateError) {
                        logger.error(`Failed to update ${tableName} with ID ${item.id}.`, {
                            error: updateError,
                            itemId: item.id,
                            tableName,
                            trace: "0168",
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
    const tableNames = PROCESSED_BOOKMARK_TABLE_NAMES;

    for (const tableName of tableNames) {
        await processTableInBatches(tableName);
    }
}
