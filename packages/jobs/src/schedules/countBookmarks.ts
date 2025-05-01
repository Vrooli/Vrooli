import { DbProvider, FindManyArgs, batch, logger } from "@local/server";
import { ModelType, camelCase, uppercaseFirstLetter } from "@local/shared";
import pkg from "@prisma/client";

const { PrismaClient } = pkg;

async function processTableInBatches(tableName: keyof InstanceType<typeof PrismaClient>): Promise<void> {
    try {
        await batch<FindManyArgs>({
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
            select: {
                id: true,
                bookmarks: true,
                _count: {
                    select: {
                        bookmarkedBy: true,
                    },
                },
            },
        });
    } catch (error) {
        logger.error("processTableInBatches caught error", { error, trace: "0166" });
    }
}

export async function countBookmarks(): Promise<void> {
    const tableNames = [
        "bookmark_list",
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
