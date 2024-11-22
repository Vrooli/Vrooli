import { FindManyArgs, batch, logger, prismaInstance } from "@local/server";
import { GqlModelType, camelCase, uppercaseFirstLetter } from "@local/shared";

async function processTableInBatches(tableName: string): Promise<void> {
    try {
        await batch<FindManyArgs>({
            objectType: uppercaseFirstLetter(camelCase(tableName)) as GqlModelType,
            processBatch: async (batch) => {
                for (const item of batch) {
                    const actualCount = item._count.bookmarkedBy;
                    if (item.bookmarks !== actualCount) {
                        logger.warning(`Updating ${tableName} ${item.id} bookmarks from ${item.bookmarks} to ${actualCount}.`, { trace: "0165" });
                        await prismaInstance[tableName].update({
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
};

export async function countBookmarks(): Promise<void> {
    const tableNames = [
        "api",
        "bookmark_list",
        "comment",
        "code",
        "issue",
        "note",
        "post",
        "project",
        "question",
        "question_answer",
        "quiz",
        "routine",
        "standard",
        "tag",
        "team",
        "user",
    ];

    for (const tableName of tableNames) {
        await processTableInBatches(tableName);
    }
};
