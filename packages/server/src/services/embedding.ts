import { DEFAULT_LANGUAGE, MINUTES_15_S, VisibilityType, WEEKS_1_S } from "@vrooli/shared";
import OpenAI, { APIError } from "openai";
import stopword from "stopword";
import { hashString } from "../auth/codes.js";
import { logger } from "../events/logger.js";
import { CacheService } from "../redisConn.js";
import { ServiceStatus, type ServiceHealth } from "./health.js";

// Define constant for the embedding dimension
const OPENAI_EMBEDDING_DIMENSION = 1536;

/**
 * All object types that can be embedded.
 */
export type EmbeddableType =
    | "Chat"
    | "Issue"
    | "Meeting"
    | "Reminder"
    | "ResourceVersion"
    | "Run"
    | "Tag"
    | "Team"
    | "User";

// Define EmbedSortOption as a standard enum export
export enum EmbedSortOption {
    EmbedDateCreatedAsc = "EmbedDateCreatedAsc",
    EmbedDateCreatedDesc = "EmbedDateCreatedDesc",
    EmbedDateUpdatedAsc = "EmbedDateUpdatedAsc",
    EmbedDateUpdatedDesc = "EmbedDateUpdatedDesc",
    EmbedTopAsc = "EmbedTopAsc",
    EmbedTopDesc = "EmbedTopDesc",
}

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

interface FetchEmbeddingsResponse {
    embeddings: (number[] | null)[];
    model: string;
}

// Define constants for string processing
const MAX_UNICODE_CHAR_CODE = 0xFFFF;

/**
 * Service for handling text embeddings, including fetching vectors from API, caching vectors,
 * and managing the cache for search results based on embeddings.
 */
export class EmbeddingService {
    private static instance: EmbeddingService;
    private openaiClient?: OpenAI;

    // --- Constants for Redis Keys ---
    private static readonly EMBEDDING_VECTOR_CACHE_PREFIX = "embeddings";
    private static readonly SEARCH_RESULT_CACHE_PREFIX = "search-embeddings-results";
    /** How long embeddings search results can be cached */
    private static readonly SEARCH_RESULT_CACHE_EXPIRY_SECONDS = MINUTES_15_S;

    // Private constructor for singleton pattern
    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() {
        // Initialize OpenAI client in the constructor
        const apiKey = process.env.OPENAI_API_KEY;
        if (!apiKey) {
            logger.warn("OPENAI_API_KEY not found in environment variables. Embedding service will not be able to fetch new embeddings.", { trace: "embedSvc-constructor-no-key" });
            // Handle the absence of API key appropriately
            this.openaiClient = undefined; // Set to undefined for type safety
        } else {
            this.openaiClient = new OpenAI({ apiKey });
        }
    }

    /**
     * Get the singleton instance of EmbeddingService.
     */
    public static get(): EmbeddingService {
        if (!EmbeddingService.instance) {
            EmbeddingService.instance = new EmbeddingService();
        }
        return EmbeddingService.instance;
    }

    /**
     * Processes a single string field according to embedding rules.
     * Internal helper for getEmbeddableString.
     */
    private static processString(str: string, language: string, minimalProcessing: boolean): string {
        // Trim whitespace, remove extra spaces and newlines
        str = str.replace(/\s+/g, " ").trim();
        // Lowercase the string
        str = str.toLowerCase();
        // Limit the Unicode range depending on the language
        // This will remove any character not in the basic multilingual plane
        str = Array.from(str).filter(ch => ch.charCodeAt(0) <= MAX_UNICODE_CHAR_CODE).join(""); // Use constant
        if (!minimalProcessing) {
            // Split the string into words
            let words = str.split(" ");
            // Remove stop words
            try {
                // Ensure language exists in stopword object
                const swRef = stopword as any; // Use type assertion if needed for dynamic access
                if (swRef[language]) {
                    words = swRef.removeStopwords(words, swRef[language]);
                } else {
                    logger.warn("Stopwords not available for language, skipping removal.", { language, trace: "embedSvc-processString-sw" });
                }
            } catch (e) {
                // Log error but don't throw, just proceed without stopword removal
                logger.error("Error removing stopwords", { error: e, language, trace: "embedSvc-processString-sw-err" });
            }
            // Remove punctuation around words, but not inside words
            words = words.map(word => word.replace(/[\p{P}\p{S}](?!\p{L})|(?<!\p{L})[\p{P}\p{S}]/gu, ""));
            // Remove duplicate words
            words = Array.from(new Set(words));
            // Join the words back with a space
            str = words.join(" ");
        }
        return str;
    }

    /**
     * Shapes a string or object to be used as an embedding input.
     * This includes trimming, lowercasing, limiting Unicode, optionally removing stopwords/punctuation/duplicates,
     * limiting length, and finally JSON stringifying the result.
     * @param input The string or object to be shaped. If object, values should be strings or arrays of strings.
     * @param language The language code to use for shaping the string.
     * @param minimalProcessing If set to true, only trim, lowercasing, and character limit are performed.
     * @returns A JSON string representation of the processed input.
     */
    public static getEmbeddableString(
        input: string | Record<string, any>,
        language: string | undefined,
        minimalProcessing = false,
    ): string {
        let processedInput: Record<string, any> | string;
        const effectiveLanguage = language || DEFAULT_LANGUAGE;

        if (typeof input === "string") {
            processedInput = EmbeddingService.processString(input, effectiveLanguage, minimalProcessing);
        } else if (typeof input === "object" && input !== null) {
            processedInput = {};
            for (const key in input) {
                const value = input[key];
                if (typeof value === "string") {
                    processedInput[key] = EmbeddingService.processString(value, effectiveLanguage, minimalProcessing);
                } else if (Array.isArray(value)) {
                    // Process each string element in the array
                    processedInput[key] = value.map(el => typeof el === "string" ? EmbeddingService.processString(el, effectiveLanguage, minimalProcessing) : el);
                }
                // Ignore other types like numbers, booleans, nested objects etc.
            }
        } else {
            // Log error for invalid input type, return empty JSON string
            logger.error("Invalid input type for getEmbeddableString", { trace: "embedSvc-getEmbedStr-invalid", inputType: typeof input });
            return JSON.stringify({});
        }

        // Always JSON stringify the final processed input
        return JSON.stringify(processedInput);
    }

    /**
     * Creates a Redis key for caching search results based on embeddings.
     */
    public static createSearchResultCacheKey({
        objectType,
        searchString,
        sortOption,
        userId,
        visibility,
    }: CreateSearchEmbeddingsKeyProps): string {
        const baseKey = `${EmbeddingService.SEARCH_RESULT_CACHE_PREFIX}:${objectType}:${searchString}:${sortOption}`;
        // Public results do not change based on the user
        if (visibility === VisibilityType.Public || !userId) {
            return `${baseKey}:Public`;
        }
        // All other results are user-specific
        return `${baseKey}:${visibility}:${userId}`;
    }

    /**
     * Checks the search result cache for a specific range and returns them if they exist.
     * @returns The cached results for the specified range, or null if not fully cached.
     */
    public static async checkSearchResultCacheRange({
        cacheKey,
        offset,
        take,
    }: CheckSearchEmbeddingsCacheProps): Promise<{ id: string }[] | null> {
        let result: { id: string }[] | null = null;
        try {
            const cachedResults = await CacheService.get().get<string>(cacheKey);
            if (!cachedResults) return null;

            const parsedResults = JSON.parse(cachedResults);
            // If the desired range is within the cached results, return them
            if (Array.isArray(parsedResults) && parsedResults.length >= offset + take) {
                result = parsedResults.slice(offset, offset + take);
            }
        } catch (e) {
            logger.error("Failed to parse or retrieve search result cache", { trace: "embedSvc-cache-check", error: e, cacheKey });
            // Treat error as cache miss
        }
        return result;
    }

    /**
     * Sets/updates a specific range within the search result cache.
     */
    public static async setSearchResultCacheRange({
        cacheKey,
        offset,
        take,
        results,
    }: SetSearchEmbeddingsCacheProps): Promise<void> {
        if (!results || results.length === 0) return;

        // Critical check: Ensure the number of results matches the 'take' parameter.
        // If they don't match, proceeding would lead to cache inconsistency where the
        // logical segment size (defined by 'take') doesn't match the actual data stored.
        if (results.length !== take) {
            logger.error("setSearchResultCacheRange: results.length does not match take. This is a programming error and will lead to cache inconsistency.", {
                trace: "embedSvc-cache-set-length-error",
                cacheKey,
                offset,
                take,
                resultsLength: results.length,
            });
            // Prevent cache corruption by not proceeding with the inconsistent data.
            return;
        }

        try {
            const currentCache = CacheService.get();

            // --- Race Condition Warning ---
            // The following block (read-modify-write) is not atomic.
            // If multiple concurrent calls to setSearchResultCacheRange operate on the same cacheKey,
            // a race condition can occur:
            // 1. Call A reads currentCache.
            // 2. Call B reads currentCache (same value as A).
            // 3. Call A modifies its parsedResults and writes back.
            // 4. Call B modifies its parsedResults (based on the old value) and writes back,
            //    potentially overwriting A's changes or leading to other inconsistencies.
            //
            // To fix this robustly, CacheService would need to support atomic list operations
            // (e.g., via Redis Lua scripts or WATCH/MULTI/EXEC), or a distributed locking
            // mechanism should be implemented around this block for operations on the same cacheKey.
            // Without such measures, there's a risk of data loss/inconsistency under high concurrency
            // for the same search result cache key.
            // --- End Race Condition Warning ---

            const cachedResults = await currentCache.get<string>(cacheKey);
            let parsedResults: { id: string }[] = [];

            if (cachedResults) {
                try {
                    parsedResults = JSON.parse(cachedResults);
                    if (!Array.isArray(parsedResults)) {
                        // This case implies valid JSON that isn't an array, which is unexpected.
                        logger.warn("Cached search results were not an array, re-initializing.", { trace: "embedSvc-cache-set-not-array", cacheKey });
                        parsedResults = []; // Treat as empty.
                    }
                } catch (e) {
                    logger.error("Failed to parse existing search cache. Clearing corrupted entry.", { trace: "embedSvc-cache-set-parse-corrupt", error: e, cacheKey });
                    try {
                        // Attempt to delete the corrupted key to prevent further use
                        await currentCache.del(cacheKey);
                    } catch (delError) {
                        logger.error("Failed to delete corrupted search cache entry.", { trace: "embedSvc-cache-set-delete-corrupt-fail", error: delError, cacheKey });
                    }
                    parsedResults = []; // Proceed as if cache was empty or successfully cleared
                }
            }

            if (offset > parsedResults.length) {
                logger.warn("Attempted to set search cache with offset beyond current length", { trace: "embedSvc-cache-set-gap", cacheKey, offset, currentLength: parsedResults.length });
                return; // Prevent setting cache if offset creates a gap
            }

            parsedResults.splice(offset, take, ...results);

            await currentCache.set(cacheKey, JSON.stringify(parsedResults), this.SEARCH_RESULT_CACHE_EXPIRY_SECONDS);
        } catch (error) {
            logger.error("Failed to set search result cache range", { trace: "embedSvc-cache-set", error, cacheKey });
        }
    }

    // --- Embedding Vector Fetching and Caching --- (Previously EmbeddingService methods)

    /**
     * Internal function to fetch embeddings from the OpenAI API endpoint.
     * Renamed from fetchEmbeddingsFromApi.
     */
    private async fetchEmbeddings(objectType: EmbeddableType | `${EmbeddableType}`, sentences: string[]): Promise<FetchEmbeddingsResponse> {
        const model = "text-embedding-3-small"; // Use the target model
        if (sentences.length === 0) {
            return { embeddings: [], model };
        }
        // Check if the client was initialized correctly
        if (!this.openaiClient?.embeddings) {
            logger.error("OpenAI client not initialized correctly. Cannot fetch embeddings.", { trace: "embedSvc-fetch-no-client" });
            return { embeddings: new Array(sentences.length).fill(null), model }; // Return array of nulls matching sentence length
        }

        try {
            const response = await this.openaiClient.embeddings.create({
                model,
                input: sentences,
                // Optionally specify dimensions if needed later, default is 1536 for small
                // dimensions: 1536
            });

            // Ensure the response structure is as expected
            if (!response || !response.data || !Array.isArray(response.data)) {
                // Log the actual structure if possible, being mindful of size
                logger.error("Invalid response structure from OpenAI embedding API", { trace: "embedSvc-fetch-openai-struct-err", responseType: typeof response, hasData: !!response?.data, isArray: Array.isArray(response?.data) });
                throw new Error("Invalid response structure from OpenAI embedding API");
            }

            // Initialize result array with nulls, with the same length as sentences
            const embeddingsMap: (number[] | null)[] = new Array(sentences.length).fill(null);
            let receivedEmbeddingsCount = 0;

            for (const item of response.data) {
                // Ensure item and item.embedding are valid before accessing item.index
                if (item && typeof item.index === "number" && Array.isArray(item.embedding)) {
                    if (item.index >= 0 && item.index < sentences.length) {
                        embeddingsMap[item.index] = item.embedding;
                        receivedEmbeddingsCount++;
                    } else {
                        logger.warn("OpenAI API returned embedding with out-of-bounds index", {
                            trace: "embedSvc-fetch-openai-idx-err",
                            index: item.index,
                            maxIndex: sentences.length - 1,
                            objectType,
                        });
                    }
                } else {
                    logger.warn("OpenAI API returned malformed embedding item", {
                        trace: "embedSvc-fetch-openai-item-err",
                        itemReceived: item !== null && typeof item === "object" ? Object.keys(item) : typeof item, // log keys or type of item
                        objectType,
                    });
                }
            }

            // Log if the number of successfully mapped embeddings does not match the input sentence count
            if (receivedEmbeddingsCount !== sentences.length) {
                const missingIndices = sentences.map((_, i) => embeddingsMap[i] === null ? i : -1).filter(i => i !== -1);
                logger.warn("Mismatch between input sentences and valid embeddings received/mapped from OpenAI", {
                    trace: "embedSvc-fetch-mismatch",
                    inputCount: sentences.length,
                    receivedAndMappedCount: receivedEmbeddingsCount,
                    missingIndices, // Log indices that are still null
                    objectType,
                });
                // Proceed with the partially populated map; nulls indicate failures for specific sentences.
            }

            return {
                embeddings: embeddingsMap,
                model: response.model || model, // Use the model returned in the response if available
            };

        } catch (error: unknown) {
            let errorMessage = "Unknown error fetching embeddings";
            const errorDetails: Record<string, any> = { // Store structured error info
                sentenceCount: sentences.length,
                objectType,
            };

            if (error instanceof APIError) {
                // OpenAI specific error
                errorMessage = error.message;
                errorDetails.statusCode = error.status;
                errorDetails.errorType = error.type;
                errorDetails.errorCode = error.code;
                // errorDetails.headers = error.headers; // Optionally log headers if needed and not too verbose
            } else if (error instanceof Error) {
                // Standard JavaScript error
                errorMessage = error.message;
                errorDetails.errorName = error.name;
                // errorDetails.stack = error.stack; // Optionally log stack if needed
            } else if (typeof error === "string") {
                // Simple string error
                errorMessage = error;
            } else if (error && typeof error === "object") {
                // Other object types, attempt to stringify
                try {
                    errorMessage = `Non-standard error object: ${JSON.stringify(error)}`;
                } catch (stringifyError) {
                    errorMessage = "Non-standard, non-serializable error object encountered.";
                }
            }
            // The main log message is now more concise
            logger.error(`Error fetching embeddings from OpenAI API: ${errorMessage}`, {
                trace: "embedSvc-fetch-openai-err",
                details: errorDetails, // Contains all the structured info
            });
            // Return a default structure on error
            return { embeddings: new Array(sentences.length).fill(null), model };
        }
    }

    /**
     * Finds embeddings vectors for a list of strings, preferring cached results over fetching new ones.
     * Caches newly fetched vectors.
     */
    public async getEmbeddings(objectType: EmbeddableType | `${EmbeddableType}`, sentences: string[]): Promise<(number[] | null)[]> {
        // Initialize with nulls, same length as sentences
        const finalEmbeddings: (number[] | null)[] = new Array(sentences.length).fill(null);
        // Use the constant for the prefix
        const cacheKeys = sentences.map(sentence => `${EmbeddingService.EMBEDDING_VECTOR_CACHE_PREFIX}:${objectType}:${hashString(sentence)}`);

        // 1. Try fetching all from cache
        let cachedEmbeddingsPromises: Promise<number[] | null>[] = [];
        try {
            cachedEmbeddingsPromises = cacheKeys.map(key => CacheService.get().get<number[]>(key));
        } catch (cacheError) {
            logger.error("CacheService get error in getEmbeddings setup", { trace: "embedSvc-get-0099", cacheError });
            // If creating promises fails, proceed as if all are uncached.
        }

        const cachedResults = await Promise.allSettled(cachedEmbeddingsPromises);

        // 2. Identify sentences needing fetching
        const uncachedSentences: { sentence: string, index: number }[] = [];
        cachedResults.forEach((result, index) => {
            if (result.status === "fulfilled" && result.value) {
                finalEmbeddings[index] = result.value; // Use cached version
            } else {
                if (result.status === "rejected") {
                    logger.warn("Failed to retrieve a cached embedding, will attempt to refetch.", { trace: "embedSvc-get-cache-fail", error: result.reason, key: cacheKeys[index] });
                }
                uncachedSentences.push({ sentence: sentences[index], index }); // Mark for fetching
            }
        });


        // 3. Fetch uncached sentences from API if any
        if (uncachedSentences.length > 0) {
            try {
                // Use the renamed private method
                const batchedResults = await this.fetchEmbeddings(objectType, uncachedSentences.map(item => item.sentence));

                // 4. Process results and update cache
                const cachePromises: Promise<void>[] = [];
                batchedResults.embeddings.forEach((embedding, idx) => {
                    const originalIndex = uncachedSentences[idx].index;
                    finalEmbeddings[originalIndex] = embedding; // Update final results

                    if (embedding && embedding.length > 0) { // Only cache valid embeddings
                        try {
                            const cacheKey = cacheKeys[originalIndex];
                            cachePromises.push(CacheService.get().set(cacheKey, embedding, WEEKS_1_S));
                        } catch (cacheError) {
                            logger.error("CacheService SET error in getEmbeddings", { trace: "embedSvc-get-0101", cacheError, key: cacheKeys[originalIndex] });
                            // Log error but continue
                        }
                    }
                });
                await Promise.allSettled(cachePromises);
            } catch (fetchError) {
                logger.error("Failed to fetch embeddings batch", { trace: "embedSvc-get-0102", fetchError });
                // Failed sentences will remain null in finalEmbeddings
            }
        }
        return finalEmbeddings;
    }

    // --- Health Check --- (Updated Logic)

    /**
     * Performs a health check on the embedding service components:
     * 1. API Connectivity & Vector Caching: Fetches embedding for a unique string, ensuring it's cached.
     * 2. Search Result Caching: Tests setting and checking a range in the search result cache.
     * @returns ServiceHealth object indicating the status.
     */
    public async checkHealth(): Promise<ServiceHealth> {
        const checkTimestamp = Date.now();
        let vectorFetchCacheHealthy = false;
        let searchResultCacheHealthy = false;
        const vectorDetails: Record<string, unknown> = {};
        const searchCacheDetails: Record<string, unknown> = {};

        // Ensure OpenAI client is available for the health check part
        if (!this.openaiClient?.embeddings) {
            vectorDetails["error"] = "OpenAI client not initialized, skipping vector fetch check.";
            vectorFetchCacheHealthy = false; // Mark as unhealthy if client is missing
        } else {
            // Only run vector fetch/cache test if client is available
            try {
                const testObjectType = "User"; // Use a common type
                const testSentence = `health-check-${Date.now()}-${Math.random()}`;
                const testHash = hashString(testSentence);
                const embeddingCacheKey = `${EmbeddingService.EMBEDDING_VECTOR_CACHE_PREFIX}:${testObjectType}:${testHash}`;

                // 1a. Ensure cache key is initially clear
                await CacheService.get().del(embeddingCacheKey);
                vectorDetails["cacheCleared"] = true;

                // 1b. Call getEmbeddings - this should fetch via OpenAI and cache
                const embeddingsResult = await this.getEmbeddings(testObjectType, [testSentence]);
                vectorDetails["getEmbeddingsCalled"] = true;


                // 1c. Verify an embedding was returned
                if (!embeddingsResult || embeddingsResult.length !== 1 || !embeddingsResult[0] || embeddingsResult[0].length === 0) {
                    throw new Error(`getEmbeddings did not return a valid vector. Result: ${JSON.stringify(embeddingsResult)}`);
                }
                vectorDetails["fetchSuccess"] = true;
                // Check the dimension, should be 1536 for text-embedding-3-small
                vectorDetails["vectorLength"] = embeddingsResult[0].length;
                if (embeddingsResult[0].length !== OPENAI_EMBEDDING_DIMENSION) {
                    logger.warn("Health Check: Unexpected vector dimension from OpenAI", { expected: OPENAI_EMBEDDING_DIMENSION, got: embeddingsResult[0].length });
                }


                // 1d. Verify the embedding was cached
                const cachedValue = await CacheService.get().get<number[]>(embeddingCacheKey);
                if (!cachedValue) {
                    // If getEmbeddings is supposed to ensure the cache is written and its promise
                    // awaits that completion, then a subsequent get should find the value immediately.
                    // Failure to do so indicates a problem with caching consistency or the promise chain.
                    throw new Error("getEmbeddings did not result in an immediately retrievable cached vector in Redis.");
                }
                vectorDetails["cacheSuccess"] = true;
                vectorFetchCacheHealthy = true;


            } catch (error) {
                const errorMessage = (error instanceof Error) ? error.message : "Unknown error during vector fetch/cache check";
                logger.error("Embedding vector fetch/cache health check failed", { trace: "embedSvc-HC-vector", error: errorMessage });
                vectorDetails["error"] = errorMessage;
                vectorFetchCacheHealthy = false;
            }
        }


        // --- Test 2: Search Result Caching ---
        try {
            // currentCache is already defined above and can be used here directly

            const testSearchString = `health-check-search-${Date.now()}`;
            const testSort = EmbedSortOption.EmbedTopDesc;
            const testVisibility = VisibilityType.Public;
            const testUserId = `health-check-user-${Date.now()}`; // Corrected to be unique for the test
            const testObjectType = "Tag"; // Corrected to use a defined EmbeddableType
            const offset = 0;
            const take = 1;
            const dummyResults = [{ id: `test-id-${Date.now()}` }];

            const searchCacheKey = EmbeddingService.createSearchResultCacheKey({
                objectType: testObjectType,
                searchString: testSearchString,
                sortOption: testSort,
                userId: testUserId, // Ensure this is handled correctly if visibility is not Public
                visibility: testVisibility,
            });

            // 2a. Clear any existing key
            await CacheService.get().del(searchCacheKey);

            // 2b. Set a range in the cache
            // Note: setSearchResultCacheRange is static, so it uses CacheService.get() internally.
            // This is fine, but ensure it's consistent with the refactor.
            await EmbeddingService.setSearchResultCacheRange({ cacheKey: searchCacheKey, offset, take, results: dummyResults });
            searchCacheDetails["setSuccess"] = true;

            // 2c. Check the same range
            // Note: checkSearchResultCacheRange is static.
            const checkedResults = await EmbeddingService.checkSearchResultCacheRange({ cacheKey: searchCacheKey, offset, take });

            // 2d. Verify results match
            if (!checkedResults || checkedResults.length !== 1 || checkedResults[0].id !== dummyResults[0].id) {
                throw new Error(`Search result cache check failed. Expected ${JSON.stringify(dummyResults)}, got ${JSON.stringify(checkedResults)}`);
            }
            searchCacheDetails["checkSuccess"] = true;
            searchResultCacheHealthy = true;

        } catch (error) {
            const errorMessage = (error instanceof Error) ? error.message : "Unknown error during search result cache check";
            logger.error("Embedding search result cache health check failed", { trace: "embedSvc-HC-search", error: errorMessage });
            searchCacheDetails["error"] = errorMessage;
            searchResultCacheHealthy = false;
        }

        // --- Determine Overall Status --- 
        let overallStatus: ServiceStatus;
        if (vectorFetchCacheHealthy && searchResultCacheHealthy) {
            overallStatus = ServiceStatus.Operational;
        } else if (vectorFetchCacheHealthy || searchResultCacheHealthy) {
            overallStatus = ServiceStatus.Degraded; // If one part works, it's degraded
        } else {
            overallStatus = ServiceStatus.Down;
        }

        return {
            healthy: overallStatus === ServiceStatus.Operational,
            status: overallStatus,
            lastChecked: checkTimestamp,
            details: {
                vectorFetchCache: { healthy: vectorFetchCacheHealthy, ...vectorDetails },
                searchResultCache: { healthy: searchResultCacheHealthy, ...searchCacheDetails },
            },
        };
    }
}
