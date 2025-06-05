import type { Prisma } from "@prisma/client";
import pkg from "@prisma/client";
import { DbProvider, batch, logger } from "@vrooli/server";
import { camelCase, uppercaseFirstLetter, type ModelType } from "@vrooli/shared";

const { PrismaClient } = pkg;

// Select shape for view count updates
const viewsSelect = {
    id: true,
    views: true,
    _count: {
        select: { viewedBy: true },
    },
} as const;
// Union of FindManyArgs types for viewable tables
type ViewsFindManyArgs =
    Prisma.issueFindManyArgs |
    Prisma.resourceFindManyArgs |
    Prisma.teamFindManyArgs |
    Prisma.userFindManyArgs;
// Union of payload types for viewable tables
type ViewsPayload =
    Prisma.issueGetPayload<{ select: typeof viewsSelect }> |
    Prisma.resourceGetPayload<{ select: typeof viewsSelect }> |
    Prisma.teamGetPayload<{ select: typeof viewsSelect }> |
    Prisma.userGetPayload<{ select: typeof viewsSelect }>;

async function processTableViewsInBatches(
    tableName: keyof InstanceType<typeof PrismaClient>,
): Promise<void> {
    try {
        await batch<ViewsFindManyArgs, ViewsPayload>({
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
            select: viewsSelect,
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
