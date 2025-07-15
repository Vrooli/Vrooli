// AI_CHECK: TASK_ID=TYPE_SAFETY COUNT=1 | LAST: 2025-07-04
import type { Prisma } from "@prisma/client";
import { verifyCountField } from "../utils/verifyCountField.js";

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

export async function countViews(): Promise<void> {
    const tableNames = [
        "issue",
        "resource",
        "team",
        "user",
    ] as const;

    await verifyCountField<ViewsFindManyArgs, ViewsPayload, typeof viewsSelect>({
        tableNames,
        countField: "views",
        relationName: "viewedBy",
        select: viewsSelect,
        traceId: "views_001",
    });
}
