import { DbProvider, FindManyArgs, batch, logger } from "@local/server";
import { ModelType, camelCase, uppercaseFirstLetter } from "@local/shared";
import pkg from "@prisma/client";

const { PrismaClient } = pkg;

async function processTableViewsInBatches(
    tableName: keyof InstanceType<typeof PrismaClient>,
): Promise<void> {
    try {
        await batch<FindManyArgs>({
            // Convert the table name (e.g. "resource") into the proper model name (e.g. "Resource")
            objectType: uppercaseFirstLetter(camelCase(tableName as string)) as ModelType,
            processBatch: async (batch) => {
                for (const item of batch) {
                    // We assume that the viewable model has a relation count alias `viewedBy`
                    // (i.e. an array of related view records) and a persisted `views` count.
                    const actualCount = item._count.viewedBy;
                    if (item.views !== actualCount) {
                        logger.warning(
                            `Updating ${tableName as string} ${item.id} views from ${item.views} to ${actualCount}.`,
                            { trace: "views_001" },
                        );
                        await (DbProvider.get()[tableName] as { update: any }).update({
                            where: { id: item.id },
                            data: { views: actualCount },
                        });
                    }
                }
            },
            select: {
                id: true,
                views: true,
                _count: {
                    select: {
                        viewedBy: true,
                    },
                },
            },
        });
    } catch (error) {
        logger.error("processTableViewsInBatches caught error", { error, trace: "views_002" });
    }
}

export async function countViews(): Promise<void> {
    const tableNames = [
        "issue",
        "resource",
        "team",
        "user",
    ] as const;

    for (const tableName of tableNames) {
        await processTableViewsInBatches(tableName);
    }
}
