import { DEFAULT_LANGUAGE, MINUTES_15_S, VisibilityType, WEEKS_1_S } from "@local/shared";
import https from "https";
import stopword from "stopword";
import { hashString } from "../auth/codes.js";
import { logger } from "../events/logger.js";
import { initializeRedis, withRedis } from "../redisConn.js";
import { ServiceStatus, type ServiceHealth } from "./health.js";

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

// The model used for embedding (instructor-base) requires an instruction
// to create embeddings. This acts as a way to fine-tune the model for
// a specific task. Most of our objects may end up using the same
// instruction, but it's possible that some may need a different one.
// See https://github.com/Vrooli/text-embedder-tests for more details.
const INSTRUCTION_COMMON = "Embed this text";
const Instructions: { [key in EmbeddableType]: string } = {
    "Chat": INSTRUCTION_COMMON,
    "Issue": INSTRUCTION_COMMON,
    "Meeting": INSTRUCTION_COMMON,
    "Reminder": INSTRUCTION_COMMON,
    "ResourceVersion": INSTRUCTION_COMMON,
    "Run": INSTRUCTION_COMMON,
    "Tag": INSTRUCTION_COMMON,
    "Team": INSTRUCTION_COMMON,
    "User": INSTRUCTION_COMMON,
};

interface FetchEmbeddingsResponse {
    embeddings: number[][];
    model: string;
}

// Define timeout constant
const HEALTH_CHECK_TIMEOUT_MS = 15_000;

// Define constants for string processing
const MAX_EMBEDDABLE_STRING_LENGTH = 128;
const MAX_UNICODE_CHAR_CODE = 0xFFFF;

/**
 * Service for handling text embeddings, including fetching vectors from API, caching vectors,
 * and managing the cache for search results based on embeddings.
 */
export class EmbeddingService {
    private static instance: EmbeddingService;

    // --- Constants for Redis Keys ---
    private static readonly EMBEDDING_VECTOR_CACHE_PREFIX = "embeddings";
    private static readonly SEARCH_RESULT_CACHE_PREFIX = "search-embeddings-results";
    /** How long embeddings search results can be cached */
    private static readonly SEARCH_RESULT_CACHE_EXPIRY_SECONDS = MINUTES_15_S;

    // Private constructor for singleton pattern
    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }

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
        // Limit the length of the field
        if (str.length > MAX_EMBEDDABLE_STRING_LENGTH) { // Use constant
            str = str.substring(0, MAX_EMBEDDABLE_STRING_LENGTH); // Use constant
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
        await withRedis({
            process: async (redisClient) => {
                if (!redisClient) return;
                const cachedResults = await redisClient.get(cacheKey);
                if (!cachedResults) return;
                try {
                    const parsedResults = JSON.parse(cachedResults);
                    // If the desired range is within the cached results, return them
                    if (Array.isArray(parsedResults) && parsedResults.length >= offset + take) {
                        result = parsedResults.slice(offset, offset + take);
                    }
                } catch (e) {
                    logger.error("Failed to parse search result cache", { trace: "embedSvc-cache-check-parse", error: e, cacheKey });
                    // Treat parse failure as cache miss
                }
                return;
            },
            trace: "checkSearchResultCacheRange",
        });
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
        await withRedis({
            process: async (redisClient) => {
                if (!redisClient || !results || results.length === 0) return;
                try {
                    // Get the existing cached results, if any
                    const cachedResults = await redisClient.get(cacheKey);
                    let parsedResults: { id: string }[] = [];
                    if (cachedResults) {
                        try {
                            parsedResults = JSON.parse(cachedResults);
                            if (!Array.isArray(parsedResults)) parsedResults = []; // Ensure it's an array
                        } catch (e) {
                            logger.error("Failed to parse existing search cache before setting", { trace: "embedSvc-cache-set-parse", error: e, cacheKey });
                            // If parsing fails, overwrite with new results if offset is 0, otherwise abort
                            if (offset !== 0) return;
                        }
                    }

                    // Ensure the array is long enough to splice into, padding with null (or a placeholder) if necessary
                    // This handles cases where offset > parsedResults.length
                    if (offset > parsedResults.length) {
                        // Pad with a placeholder or handle differently if strict continuity is required
                        // For simplicity, let's prevent caching if there's a gap.
                        // Alternatively, could pad with nulls: parsedResults.length = offset;
                        logger.warn("Attempted to set search cache with offset beyond current length", { trace: "embedSvc-cache-set-gap", cacheKey, offset, currentLength: parsedResults.length });
                        return; // Prevent setting cache if offset creates a gap
                    }

                    // Place the results in the correct position in the cached results
                    parsedResults.splice(offset, take, ...results);

                    // Set the new cached results with expiry
                    await redisClient.set(cacheKey, JSON.stringify(parsedResults), { EX: this.SEARCH_RESULT_CACHE_EXPIRY_SECONDS });

                } catch (error) {
                    logger.error("Failed to set search result cache range", { trace: "embedSvc-cache-set", error, cacheKey });
                }
            },
            trace: "setSearchResultCacheRange",
        });
    }

    // --- Embedding Vector Fetching and Caching --- (Previously EmbeddingService methods)

    /**
     * Internal function to fetch embeddings from the API endpoint.
     * Renamed from fetchEmbeddingsFromApi.
     */
    private async fetchEmbeddings(objectType: EmbeddableType | `${EmbeddableType}`, sentences: string[]): Promise<FetchEmbeddingsResponse> {
        try {
            // Ensure the objectType exists in Instructions before proceeding
            const instruction = Instructions[objectType as EmbeddableType];
            if (!instruction) {
                throw new Error(`Unsupported object type for embedding: ${objectType}`);
            }

            return new Promise((resolve, reject) => {
                const data = JSON.stringify({ instruction, sentences });
                const options = {
                    hostname: "embedtext.com", // TODO: Move to config/env var?
                    port: 443,
                    path: "/",
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json",
                        "Content-Length": Buffer.byteLength(data, "utf8").toString(), // Use toString()
                    },
                    // Add timeout here
                    signal: AbortSignal.timeout(HEALTH_CHECK_TIMEOUT_MS),
                };
                const apiRequest = https.request(options, apiRes => {
                    let responseBody = "";
                    apiRes.on("data", chunk => {
                        responseBody += chunk;
                    });
                    apiRes.on("end", () => {
                        try { // Add try-catch for JSON parsing
                            const result = JSON.parse(responseBody);
                            // Ensure result is of the expected type
                            if (typeof result !== "object" || !Object.prototype.hasOwnProperty.call(result, "embeddings") || !Array.isArray(result.embeddings)) {
                                const error = "Invalid response from embedding API";
                                logger.error(error, { trace: "embedSvc-fetch-0021", result, data, options });
                                reject(new Error(error));
                            } else {
                                resolve(result as FetchEmbeddingsResponse);
                            }
                        } catch (parseError) {
                            const error = "Failed to parse JSON response from embedding API";
                            logger.error(error, { trace: "embedSvc-fetch-0022", responseBody, parseError });
                            reject(new Error(error));
                        }
                    });
                });
                apiRequest.on("error", error => {
                    logger.error("Embedding API request error", { trace: "embedSvc-fetch-0023", error });
                    reject(error);
                });
                apiRequest.write(data);
                apiRequest.end();
            });
        } catch (error) {
            logger.error("Error preparing fetch embeddings request", { trace: "embedSvc-fetch-0084", error });
            // Return a default structure on error to prevent downstream issues
            return { embeddings: [], model: "" };
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

        await withRedis({
            process: async (redisClient) => {
                // 1. Try fetching all from cache
                let cachedEmbeddings: (number[] | null)[] = new Array(sentences.length).fill(null);
                if (redisClient) {
                    try {
                        const cachedData = await redisClient.mGet(cacheKeys);
                        cachedEmbeddings = cachedData.map((data: string | null) => data ? JSON.parse(data) : null);
                    } catch (redisError) {
                        logger.error("Redis mGet error in getEmbeddings", { trace: "embedSvc-get-0100", redisError });
                        // Continue without cache if Redis fails
                    }
                }

                // 2. Identify sentences needing fetching
                const uncachedSentences: { sentence: string, index: number }[] = [];
                sentences.forEach((sentence, index) => {
                    if (cachedEmbeddings[index]) {
                        finalEmbeddings[index] = cachedEmbeddings[index]; // Use cached version
                    } else {
                        uncachedSentences.push({ sentence, index }); // Mark for fetching
                    }
                });

                // 3. Fetch uncached sentences from API if any
                if (uncachedSentences.length > 0) {
                    try {
                        // Use the renamed private method
                        const batchedResults = await this.fetchEmbeddings(objectType, uncachedSentences.map(item => item.sentence));

                        // 4. Process results and update cache
                        batchedResults.embeddings.forEach((embedding, idx) => {
                            const originalIndex = uncachedSentences[idx].index;
                            finalEmbeddings[originalIndex] = embedding; // Update final results

                            // Cache the result if Redis is available
                            if (redisClient && embedding && embedding.length > 0) { // Only cache valid embeddings
                                try {
                                    const cacheKey = cacheKeys[originalIndex];
                                    // Store fetched embeddings in cache with expiry
                                    redisClient.set(cacheKey, JSON.stringify(embedding), { EX: WEEKS_1_S });
                                } catch (redisError) {
                                    logger.error("Redis SET error in getEmbeddings", { trace: "embedSvc-get-0101", redisError, key: cacheKeys[originalIndex] });
                                    // Log error but continue
                                }
                            }
                        });
                    } catch (fetchError) {
                        logger.error("Failed to fetch embeddings batch", { trace: "embedSvc-get-0102", fetchError });
                        // Failed sentences will remain null in finalEmbeddings
                    }
                }
            },
            trace: "getEmbeddings",
        });

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

        const redisClient = await initializeRedis(); // Get direct client for DEL

        // --- Test 1: Vector Fetching and Caching --- (API + embeddings: cache)
        try {
            if (!redisClient) throw new Error("Redis client not available for health check");

            const testObjectType = "User"; // Use a common type
            const testSentence = `health-check-${Date.now()}-${Math.random()}`;
            const testHash = hashString(testSentence);
            const embeddingCacheKey = `${EmbeddingService.EMBEDDING_VECTOR_CACHE_PREFIX}:${testObjectType}:${testHash}`;

            // 1a. Ensure cache key is initially clear
            await redisClient.del(embeddingCacheKey);

            // 1b. Call getEmbeddings - this should fetch and cache
            const embeddingsResult = await this.getEmbeddings(testObjectType, [testSentence]);

            // 1c. Verify an embedding was returned
            if (!embeddingsResult || embeddingsResult.length !== 1 || !embeddingsResult[0] || embeddingsResult[0].length === 0) {
                throw new Error("getEmbeddings did not return a valid vector.");
            }
            vectorDetails["fetchSuccess"] = true;
            vectorDetails["vectorLength"] = embeddingsResult[0].length;

            // 1d. Verify the embedding was cached
            const cachedValue = await redisClient.get(embeddingCacheKey);
            if (!cachedValue) {
                throw new Error("getEmbeddings did not cache the vector in Redis.");
            }
            vectorDetails["cacheSuccess"] = true;
            vectorFetchCacheHealthy = true;

        } catch (error) {
            const errorMessage = (error instanceof Error) ? error.message : "Unknown error during vector fetch/cache check";
            logger.error("Embedding vector fetch/cache health check failed", { trace: "embedSvc-HC-vector", error: errorMessage });
            vectorDetails["error"] = errorMessage;
            vectorFetchCacheHealthy = false;
        }

        // --- Test 2: Search Result Caching --- (search: cache)
        try {
            if (!redisClient) throw new Error("Redis client not available for health check");

            const testSearchString = `health-check-search-${Date.now()}`;
            // Access enum directly
            const testSort = EmbedSortOption.EmbedTopDesc;
            // Access enum directly
            const testVisibility = VisibilityType.Public;
            const testUserId = `health-check-user-${Date.now()}`;
            const testObjectType = "Tag";
            const offset = 0;
            const take = 1;
            const dummyResults = [{ id: `test-id-${Date.now()}` }];

            const searchCacheKey = EmbeddingService.createSearchResultCacheKey({
                objectType: testObjectType,
                searchString: testSearchString,
                sortOption: testSort,
                userId: testUserId,
                visibility: testVisibility,
            });

            // 2a. Clear any existing key
            await redisClient.del(searchCacheKey);

            // 2b. Set a range in the cache
            await EmbeddingService.setSearchResultCacheRange({ cacheKey: searchCacheKey, offset, take, results: dummyResults });
            searchCacheDetails["setSuccess"] = true;

            // 2c. Check the same range
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
