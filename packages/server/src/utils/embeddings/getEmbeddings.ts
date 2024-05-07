import https from "https";
import { hashString } from "../../auth/codes";
import { logger } from "../../events/logger";
import { withRedis } from "../../redisConn";
import { EmbeddableType } from "./types";

// The model used for embedding (instructor-base) requires an instruction 
// to create embeddings. This acts as a way to fine-tune the model for 
// a specific task. Most of our objects may end up using the same 
// instruction, but it's possible that some may need a different one. 
// See https://github.com/Vrooli/text-embedder-tests for more details.
const INSTRUCTION_COMMON = "Embed this text";
const Instructions: { [key in EmbeddableType]: string } = {
    "Api": INSTRUCTION_COMMON,
    "Chat": INSTRUCTION_COMMON,
    "Issue": INSTRUCTION_COMMON,
    "Meeting": INSTRUCTION_COMMON,
    "Note": INSTRUCTION_COMMON,
    "Organization": INSTRUCTION_COMMON,
    "Post": INSTRUCTION_COMMON,
    "Project": INSTRUCTION_COMMON,
    "Question": INSTRUCTION_COMMON,
    "Quiz": INSTRUCTION_COMMON,
    "Reminder": INSTRUCTION_COMMON,
    "Routine": INSTRUCTION_COMMON,
    "RunProject": INSTRUCTION_COMMON,
    "RunRoutine": INSTRUCTION_COMMON,
    "SmartContract": INSTRUCTION_COMMON,
    "Standard": INSTRUCTION_COMMON,
    "Tag": INSTRUCTION_COMMON,
    "User": INSTRUCTION_COMMON,
};

/**
 * Function to get embeddings from a https://github.com/Vrooli/text-embedder-endpoint API
 * @param objectType The type of object to generate embeddings for
 * @param sentences The sentences to generate embeddings for. Each sentence 
 * can be plaintext (if only embedding one field) or stringified JSON (for multiple fields)
 * @returns A Promise that resolves with the embeddings, in the same order as the sentences
 * @throws An Error if the API request fails
 */
export const fetchEmbeddingsFromApi = async (objectType: EmbeddableType | `${EmbeddableType}`, sentences: string[]): Promise<{
    embeddings: number[][],
    model: string
}> => {
    try {
        const instruction = Instructions[objectType];
        return new Promise((resolve, reject) => {
            const data = JSON.stringify({ instruction, sentences });
            const options = {
                hostname: "embedtext.com",
                port: 443,
                path: "/",
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    "Content-Length": Buffer.byteLength(data, "utf8"),
                },
            };
            const apiRequest = https.request(options, apiRes => {
                let responseBody = "";
                apiRes.on("data", chunk => {
                    responseBody += chunk;
                });
                apiRes.on("end", () => {
                    const result = JSON.parse(responseBody);
                    // Ensure result is of the expected type
                    if (typeof result !== "object" || !Object.prototype.hasOwnProperty.call(result, "embeddings") || !Array.isArray(result.embeddings)) {
                        const error = "Invalid response from embedding API";
                        logger.error(error, { trace: "0021", result, data, options });
                        reject(new Error(error));
                    }
                    else resolve(result);
                });
            });
            apiRequest.on("error", error => {
                console.error(`Error: ${error}`);
                reject(error);
            });
            apiRequest.write(data);
            apiRequest.end();
        });
    } catch (error) {
        logger.error("Error fetching embeddings", { trace: "0084", error });
        return { embeddings: [], model: "" };
    }
};

/**
 * Finds embeddings for a list of strings, preferring cached results over fetching new ones
 */
export const getEmbeddings = async (objectType: EmbeddableType | `${EmbeddableType}`, sentences: string[]) => {
    let cachedEmbeddings: (number[] | null)[] = [];
    const cacheKeys = sentences.map(sentence => `embeddings:${objectType}:${hashString(sentence)}`);

    await withRedis({
        process: async (redisCient) => {
            // Find cached embeddings
            const cachedData = await redisCient.mGet(cacheKeys);
            cachedEmbeddings = cachedData.map((data: string | null) => data ? JSON.parse(data) : null);

            // If some are not cached, fetch them from the API
            const uncachedSentences: { sentence: string, index: number }[] = sentences.map((sentence, index) => ({ sentence, index })).filter((_, index) => !cachedEmbeddings[index]);
            if (uncachedSentences.length > 0) {
                const batchedResults = await fetchEmbeddingsFromApi(objectType, uncachedSentences.map(item => item.sentence));
                batchedResults.embeddings.forEach((embedding, idx) => {
                    const { sentence, index } = uncachedSentences[idx];
                    const sentenceHash = hashString(sentence);
                    const cacheKey = `embeddings:${objectType}:${sentenceHash}`;

                    // Store fetched embeddings in cache
                    redisCient.set(cacheKey, JSON.stringify(embedding));
                    // Set expiry of 7 days
                    redisCient.expire(cacheKey, 60 * 60 * 24 * 7);

                    // Update cached embeddings
                    cachedEmbeddings[index] = embedding;
                });
            }
        },
        trace: "getEmbeddings",
    });

    return cachedEmbeddings;
};
