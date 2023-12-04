import { PrismaType, logger, withPrisma } from "@local/server";

const processTableInBatches = async (prisma: PrismaType, tableName: string): Promise<void> => {
    const batchSize = 100;
    let skip = 0;
    let continueProcessing = true;

    while (continueProcessing) {
        const items = await prisma[tableName].findMany({
            skip,
            take: batchSize,
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

        if (items.length === 0) {
            continueProcessing = false;
            break;
        }

        for (const item of items) {
            const actualCount = item._count.bookmarkedBy;
            if (item.bookmarks !== actualCount) {
                logger.warn(`Had to update ${tableName} ${item.id} bookmarks from ${item.bookmarks} to ${actualCount}. This should not happen.`, { trace: "0166" });
                await prisma[tableName].update({
                    where: { id: item.id },
                    data: { bookmarks: actualCount },
                });
            }
        }

        skip += batchSize;
    }
};

export const countBookmarks = async (): Promise<void> => {
    const tableNames = [
        "api",
        "bookmark_list",
        "comment",
        "issue",
        "note",
        "organization",
        "post",
        "project",
        "question",
        "question_answer",
        "quiz",
        "routine",
        "smart_contract",
        "standard",
        "tag",
        "user",
    ];

    await withPrisma({
        process: async (prisma) => {
            for (const tableName of tableNames) {
                await processTableInBatches(prisma, tableName);
            }
        },
        trace: "0167",
        traceObject: { tableNames },
    });
};
