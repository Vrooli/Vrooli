import { getEmbeddings } from "../embeddings";
import { getDateLimit, QueryEmbeddingsHelperProps, QueryEmbeddingsProps, SearchTimePeriod } from "./base";

export const findTopTagsWithEmbedding = async ({
    dateLimit,
    embedding,
    limit = 5,
    offset = 0,
    prisma,
    thresholdBookmarks = 0,
    thresholdDistance = 1,
}: QueryEmbeddingsHelperProps) => {
    const test = await prisma.$queryRaw`
        SELECT 
            t."id", 
            ((1 / (1 + (tt."embedding" <-> ${embedding}::vector))) * 2) + t."bookmarks" as points
        FROM "tag" t
        INNER JOIN "tag_translation" tt ON t."id" = tt."tagId"
        WHERE t."bookmarks" >= ${thresholdBookmarks} 
            AND tt."embedding" <-> ${embedding}::vector < ${thresholdDistance}
            AND EXTRACT(EPOCH FROM t."created_at") >= ${dateLimit}
        ORDER BY points DESC
        LIMIT ${limit}
        OFFSET ${offset}
    `;
    return test;
};


export const findTopTagsWithoutEmbedding = async ({
    dateLimit,
    limit = 5,
    offset = 0,
    prisma,
    thresholdBookmarks = 0,
}: QueryEmbeddingsHelperProps) => {
    const test = await prisma.$queryRaw`
        SELECT 
            t."id", 
            t."bookmarks" as points
        FROM "tag" t
        WHERE t."bookmarks" >= ${thresholdBookmarks} 
            AND EXTRACT(EPOCH FROM t."created_at") >= ${dateLimit}
        ORDER BY points DESC
        LIMIT ${limit}
        OFFSET ${offset}
    `;
    return test;
};


export const findNewTagsWithEmbedding = async ({
    dateLimit,
    embedding,
    limit = 5,
    offset = 0,
    prisma,
    thresholdBookmarks = 0,
    thresholdDistance = 1,
}: QueryEmbeddingsHelperProps) => {
    const test = await prisma.$queryRaw`
        SELECT 
            t."id", 
            ((1 / (1 + (tt."embedding" <-> ${embedding}::vector))) * 2) + t."bookmarks" as points
        FROM "tag" t
        INNER JOIN "tag_translation" tt ON t."id" = tt."tagId"
        WHERE t."bookmarks" >= ${thresholdBookmarks} 
            AND tt."embedding" <-> ${embedding}::vector < ${thresholdDistance}
            AND EXTRACT(EPOCH FROM t."created_at") >= ${dateLimit}
        ORDER BY t."created_at" DESC
        LIMIT ${limit}
        OFFSET ${offset}
    `;
    return test;
};


export const findNewTagsWithoutEmbedding = async ({
    dateLimit,
    limit = 5,
    offset = 0,
    prisma,
    thresholdBookmarks = 0,
}: QueryEmbeddingsHelperProps) => {
    const test = await prisma.$queryRaw`
        SELECT 
            t."id", 
            t."bookmarks" as points
        FROM "tag" t
        WHERE t."bookmarks" >= ${thresholdBookmarks} 
            AND EXTRACT(EPOCH FROM t."created_at") >= ${dateLimit}
        ORDER BY t."created_at" DESC
        LIMIT ${limit}
        OFFSET ${offset}
    `;
    return test;
};


export const findTags = async ({
    prisma,
    searchString,
    sortOption,
    thresholdBookmarks = 0,
    timePeriod = SearchTimePeriod.AllTime,
    limit = 5,
    offset = 0,
}: QueryEmbeddingsProps) => {
    // If searchString is provided, get the embedding for it
    let embedding = "";
    const searchStringTrimmed = searchString.trim();
    if (searchStringTrimmed.length > 0) {
        const embeddings = await getEmbeddings("Tag", [searchString]);
        embedding = JSON.stringify(embeddings[0]);
    }
    // Common props for all helper functions
    const dateLimit = getDateLimit(timePeriod);
    const props = { prisma, dateLimit, embedding, thresholdBookmarks, limit, offset };

    switch (sortOption) {
        case "Top":
            return searchStringTrimmed.length > 0 ?
                findTopTagsWithEmbedding(props) :
                findTopTagsWithoutEmbedding(props);
        case "New":
            return searchStringTrimmed.length > 0 ?
                findNewTagsWithEmbedding(props) :
                findNewTagsWithoutEmbedding(props);
        default:
            throw new Error(`Invalid sort option: ${sortOption}`);
    }
};
