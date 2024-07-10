import { MINUTES_15_S, VisibilityType } from "@local/shared";
import { withRedis } from "../../redisConn";
import { EmbeddableType } from "./types";

type CreateSearchEmbeddingsKeyProps = {
    /** The type of object being searched */
    objectType: EmbeddableType;
    /** The search string (not the search string's embedding, as that would increase the cache size by a lot) */
    searchString: string;
    /** The sort option */
    sortOption: EmbedSortOption | `${EmbedSortOption}`;
    userId: string | null | undefined;
    visibility: VisibilityType;
};

type CheckSearchEmbeddingsCacheProps = {
    cacheKey: string;
    /** The offset for the search */
    offset: number;
    /** The number of results to take */
    take: number;
};

type SetSearchEmbeddingsCacheProps = CheckSearchEmbeddingsCacheProps & {
    /** The results to cache */
    results: { id: string }[];
};

export enum EmbedSortOption {
    EmbedDateCreatedAsc = "EmbedDateCreatedAsc",
    EmbedDateCreatedDesc = "EmbedDateCreatedDesc",
    EmbedDateUpdatedAsc = "EmbedDateUpdatedAsc",
    EmbedDateUpdatedDesc = "EmbedDateUpdatedDesc",
    EmbedTopAsc = "EmbedTopAsc",
    EmbedTopDesc = "EmbedTopDesc",
}

/**
 * Manages the search embeddings cache, to speed up search results.
 */
export class SearchEmbeddingsCache {
    /** How long embeddings search results can be cached */
    static CACHE_EXPIRY_SECONDS = MINUTES_15_S;

    /**
     * Creates redis key for search embeddings results.
     */
    static createKey = ({
        objectType,
        searchString,
        sortOption,
        userId,
        visibility,
    }: CreateSearchEmbeddingsKeyProps): string => {
        const baseKey = `search:${objectType}:${searchString}:${sortOption}`;
        // Public results do not change based on the user
        if (visibility === VisibilityType.Public || !userId) {
            return `${baseKey}:Public`;
        }
        // All other results are user-specific
        return `${baseKey}:${visibility}:${userId}`;
    };

    /**
     * Checks for cached results and returns them if they exist.
     * @returns The cached results, or null if they don't exist
     */
    static check = async ({
        cacheKey,
        offset,
        take,
    }: CheckSearchEmbeddingsCacheProps): Promise<{ id: string }[] | null> => {
        let result: { id: string }[] | null = null;
        await withRedis({
            process: async (redisClient) => {
                if (!redisClient) return;
                const cachedResults = await redisClient.get(cacheKey);
                if (!cachedResults) return;
                const parsedResults = JSON.parse(cachedResults);
                // If the desired range is within the cached results, return them
                if (parsedResults.length >= offset + take) {
                    result = parsedResults.slice(offset, offset + take);
                }
                return;
            },
            trace: "0411",
        });
        return result;
    };

    /**
     * Sets the cache for search results.
     */
    static set = async ({
        cacheKey,
        offset,
        take,
        results,
    }: SetSearchEmbeddingsCacheProps & { results: { id: string }[] }): Promise<void> => {
        await withRedis({
            process: async (redisClient) => {
                if (!redisClient) return;
                // Get the existing cached results, if any
                const cachedResults = await redisClient.get(cacheKey);
                const parsedResults = cachedResults ? JSON.parse(cachedResults) : [];
                // If the offset is greater than the length of the cached results, we can't cache the new results
                if (offset > parsedResults.length) return;
                // Place the results in the correct position in the cached results
                if (offset <= parsedResults.length) {
                    parsedResults.splice(offset, Math.min(take, parsedResults.length - offset), ...results);
                } else if (offset === parsedResults.length) {
                    parsedResults.push(...results);
                }
                // Set the new cached results
                await redisClient.set(cacheKey, JSON.stringify(parsedResults));
                // If the cache didn't exist before, set an expiry
                if (!cachedResults) await redisClient.expire(cacheKey, this.CACHE_EXPIRY_SECONDS);
            },
            trace: "0410",
        });
    };
}
