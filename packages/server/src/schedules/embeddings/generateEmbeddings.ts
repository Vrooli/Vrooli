import * as crypto from "crypto";
import https from "https";
import { logger } from "../../events";
import { initializeRedis } from "../../redisConn";

/**
 * Creates a hash of the given text. Used as a key for storing embeddings in Redis.
 */
const hashText = (text: string): string => {
    const hash = crypto.createHash("sha256");
    hash.update(text);
    return hash.digest("hex");
};

/**
 * Function to get embeddings from Hugging Face API
 * @param text The text for which to generate embeddings
 * @returns A Promise that resolves with the embeddings
 * @throws An Error if the API request fails
 */
async function getEmbeddings(text: string): Promise<any> {
    // Initialize Redis client
    const client = await initializeRedis();
    // Compute the key for this text
    const key = hashText(text);
    // Try to get the embedding from Redis
    const cachedEmbedding = await client.get(key);
    if (cachedEmbedding) {
        try {
            // If the embedding is in Redis, parse it and return it
            const embeddingParsed = JSON.parse(cachedEmbedding);
            // If embedding is not an array, log the error and delete the key from Redis
            if (!Array.isArray(embeddingParsed)) {
                logger.error("Embedding from Redis is not an array", { trace: "0454" });
                await client.del(key);
            }
            else return embeddingParsed;
        } catch (error) {
            // If parsing fails, log the error and delete the key from Redis
            logger.error("Failed to parse embedding from Redis", { trace: "0455", error });
            await client.del(key);
        }
    }
    // If the embedding is not in Redis, fetch it from the Hugging Face API
    return new Promise((resolve, reject) => {
        const apiToken = process.env.HUGGINGFACE_API_KEY;
        const data = JSON.stringify({ inputs: text });
        const options = {
            hostname: "api-inference.huggingface.co",
            port: 443,
            path: "/models/instructor-base",
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                "Authorization": `Bearer ${apiToken}`,
                "Content-Length": data.length,
            },
        };
        const apiRequest = https.request(options, apiRes => {
            let responseBody = "";
            apiRes.on("data", chunk => {
                responseBody += chunk;
            });
            apiRes.on("end", () => {
                const embeddings = JSON.parse(responseBody);
                logger.info(`Embedding request END "${text}": ${JSON.stringify(embeddings)}`, { trace: "0457" });
                // Store the embedding in Redis for future use
                client.set(key, JSON.stringify(embeddings));
                resolve(embeddings);
            });
        });
        apiRequest.on("error", error => {
            console.error(`Error: ${error}`);
            reject(error);
        });
        apiRequest.write(data);
        apiRequest.end();
    });
}

/**
 * Creates text embeddings for all searchable objects, which either:
 * - Don't have embeddings yet
 * - Have been updated since their last embedding was created
 */
export const generateEmbeddings = async () => {
    const testString = "The quick brown fox jumps over the lazy dog";
    const testEmbedding = await getEmbeddings(testString);
    logger.info(`Embedding for "${testString}": ${JSON.stringify(testEmbedding)}`, { trace: "0456" });
};
