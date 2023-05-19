import { PrismaType } from "../../types";

interface QueryEmbeddingsBaseProps {
    prisma: PrismaType;
    embedding: number[];
    distanceThreshold?: number;
    limit?: number;
    offset?: number;
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
}: QueryTagsBookmarksDescProps) => {
    const test1Vector = JSON.stringify(embedding);
    const test = await prisma.$queryRaw`
        SELECT 
            t."id", 
            ((1 / (1 + (tt."embedding" <-> ${test1Vector}::vector))) * 2) + t."bookmarks" as points
        FROM "tag" t
        INNER JOIN "tag_translation" tt ON t."id" = tt."tagId"
        WHERE t."bookmarks" >= ${bookmarksThreshold} AND tt."embedding" <-> ${test1Vector}::vector < ${distanceThreshold}
        ORDER BY points DESC
        LIMIT ${limit}
        OFFSET ${offset}
    `;

    return test;
};
