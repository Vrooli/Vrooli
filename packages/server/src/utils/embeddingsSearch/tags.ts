import { PrismaType } from "../../types";

export enum SearchTimePeriod {
    Day = "Day",
    Week = "Week",
    Month = "Month",
    Year = "Year",
    AllTime = "AllTime"
}

export function getDateLimit(period?: SearchTimePeriod): number {
    let dateLimit: Date = new Date();
    switch (period) {
        case SearchTimePeriod.Day:
            dateLimit.setDate(dateLimit.getDate() - 1);
            break;
        case SearchTimePeriod.Week:
            dateLimit.setDate(dateLimit.getDate() - 7);
            break;
        case SearchTimePeriod.Month:
            dateLimit.setMonth(dateLimit.getMonth() - 1);
            break;
        case SearchTimePeriod.Year:
            dateLimit.setFullYear(dateLimit.getFullYear() - 1);
            break;
        case SearchTimePeriod.AllTime:
        default:
            dateLimit = new Date(0);
            break;
    }
    return Math.floor(dateLimit.getTime() / 1000); // Convert to seconds
}


interface QueryEmbeddingsBaseProps {
    prisma: PrismaType;
    embedding: number[];
    distanceThreshold?: number;
    limit?: number;
    offset?: number;
    timePeriod?: SearchTimePeriod;
}

interface QueryTagsBookmarksDescProps extends QueryEmbeddingsBaseProps {
    bookmarksThreshold?: number;
}

export const queryTagsBookmarks = async ({
    prisma,
    embedding,
    bookmarksThreshold = 0,
    distanceThreshold = 1,
    limit = 5,
    offset = 0,
    timePeriod = SearchTimePeriod.Year,
}: QueryTagsBookmarksDescProps) => {
    const test1Vector = JSON.stringify(embedding);
    const dateLimit = getDateLimit(timePeriod);
    const test = await prisma.$queryRaw`
        SELECT 
            t."id", 
            ((1 / (1 + (tt."embedding" <-> ${test1Vector}::vector))) * 2) + t."bookmarks" as points
        FROM "tag" t
        INNER JOIN "tag_translation" tt ON t."id" = tt."tagId"
        WHERE t."bookmarks" >= ${bookmarksThreshold} 
            AND tt."embedding" <-> ${test1Vector}::vector < ${distanceThreshold}
            AND EXTRACT(EPOCH FROM t."created_at") >= ${dateLimit}
        ORDER BY points DESC
        LIMIT ${limit}
        OFFSET ${offset}
    `;

    return test;
};
