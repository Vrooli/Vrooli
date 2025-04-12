/**
 * Redis connection, so we don't have to keep creating new connections
 */
import { createClient, RedisClientType } from "redis";
import { ErrorTrace } from "./events/error.js";
import { logger } from "./events/logger.js";

const MAX_RETRIES = 5;

let redisClient: RedisClientType | null = null;
let retryCount = 0;

/**
 * Get the Redis URL from the environment variable.
 * This is a function because it allows us to mock the Redis URL for testing.
 * 
 * @returns The Redis URL
 */
export function getRedisUrl() {
    return process.env.REDIS_URL || "redis://redis:6379";
}

export async function createRedisClient() {
    if (retryCount >= MAX_RETRIES) {
        logger.error("Max retries reached, disabling Redis client", { trace: "0579" });
        return null;
    }

    logger.info("Creating Redis client.", { trace: "0184" });
    const REDIS_URL = getRedisUrl();
    const redisClient = createClient({ url: REDIS_URL });
    redisClient.on("error", (error) => {
        retryCount++;
        logger.error("Error occured while connecting or accessing redis server", { trace: "0002", error, retryCount });
        if (retryCount >= MAX_RETRIES) {
            logger.error("Max retries reached, disabling Redis client", { trace: "0580" });
            redisClient.disconnect();
        }
    });
    redisClient.on("end", () => {
        logger.info("Redis client closed.", { trace: "0208" });
        retryCount = 0;
    });
    redisClient.on("connect", () => {
        if (redisClient.isReady) {
            retryCount = 0;
        }
    });
    await redisClient.connect();
    // IF the client is not ready (meaning it failed to connect), return null
    if (!redisClient.isReady) {
        logger.error("Redis client is not ready", { trace: "0581" });
        return null;
    }
    // Otherwise, return the client
    return redisClient;
}

export async function initializeRedis(): Promise<RedisClientType | null> {
    // If the client was never created or was disconnected
    if (!redisClient || !redisClient.isReady) {
        redisClient = await createRedisClient() as RedisClientType | null;
    }
    return redisClient;
}

/**
 * Safely closes the Redis client connection
 * This should be called during test teardown to prevent hanging connections
 */
export async function closeRedis(): Promise<void> {
    // If client doesn't exist, nothing to do
    if (!redisClient) {
        logger.info("No Redis client to close", { trace: "0585" });
        return;
    }

    try {
        logger.info("Closing Redis client connection", { trace: "0582" });

        // Check if the client is actually connected before trying operations
        let isConnected = false;
        try {
            // Try a simple operation to check connection status
            if (redisClient.isReady) {
                await redisClient.ping().catch(() => { });
                isConnected = true;
            }
        } catch (e) {
            // The connection is already closed
            logger.info("Redis client is already disconnected", { trace: "0586" });
            redisClient = null;
            retryCount = 0;
            return;
        }

        // Only perform cleanup if the connection is still active
        if (isConnected) {
            // Remove all listeners before disconnecting
            redisClient.removeAllListeners();

            // Try to unsubscribe from all channels if this is a subscriber
            try {
                if (typeof redisClient.unsubscribe === 'function') {
                    await redisClient.unsubscribe();
                }
            } catch (e) {
                // Ignore unsubscribe errors
                logger.error("Error unsubscribing from Redis channels", { trace: "0584", error: e });
            }

            // Disconnect with force option
            await redisClient.disconnect();
        }

        // Ensure the client is really closed by setting it to null
        redisClient = null;

        // Reset retry count
        retryCount = 0;

        logger.info("Redis client successfully closed", { trace: "0587" });
    } catch (error) {
        logger.error("Error closing Redis client", { trace: "0583", error });

        // Even if there's an error, set the client to null to prevent further usage
        redisClient = null;
        retryCount = 0;
    }
}

interface WithRedisProps {
    process: (redisClient: RedisClientType | null) => Promise<void>,
    trace: string,
    traceObject?: ErrorTrace,
}

/**
 * Handles the Redis connection/disconnection and error logging
 */
export async function withRedis({
    process,
    trace,
    traceObject,
}: WithRedisProps): Promise<boolean> {
    let success = false;
    try {
        const redis = await initializeRedis();
        await process(redis);
        success = true;
    } catch (error) {
        logger.error("Caught error in withRedis", { trace, error, ...traceObject });
    }
    return success;
}
