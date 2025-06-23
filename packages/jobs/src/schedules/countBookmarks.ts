import type { Prisma } from "@prisma/client";
import { verifyCountField } from "../utils/verifyCountField.js";

// Define the specific table names that this job processes
const PROCESSED_BOOKMARK_TABLE_NAMES = [
    "comment",
    "issue",
    "resource",
    "tag",
    "team",
    "user",
] as const;

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

export async function countBookmarks(): Promise<void> {
    await verifyCountField<BookmarkFindManyArgs, BookmarkPayloadItem, typeof bookmarkSelect>({
        tableNames: PROCESSED_BOOKMARK_TABLE_NAMES,
        countField: "bookmarks",
        relationName: "bookmarkedBy",
        select: bookmarkSelect,
        traceId: "0165",
    });
}
