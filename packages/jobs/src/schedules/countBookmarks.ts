import { FindManyArgs, batch, logger } from "@local/server";
import { GqlModelType, uppercaseFirstLetter } from "@local/shared";
import pkg from "lodash";

const { camelCase } = pkg;

const processTableInBatches = async (tableName: string): Promise<void> => {
    await batch<FindManyArgs>({
        objectType: uppercaseFirstLetter(camelCase(tableName)) as GqlModelType,
        processBatch: async (batch, prisma) => {
            for (const item of batch) {
                const actualCount = item._count.bookmarkedBy;
                if (item.bookmarks !== actualCount) {
                    logger.warning(`Updating ${tableName} ${item.id} bookmarks from ${item.bookmarks} to ${actualCount}.`, { trace: "0165" });
                    await prisma[tableName].update({
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
        trace: "0166",
    });
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

    for (const tableName of tableNames) {
        await processTableInBatches(tableName);
    }
};
