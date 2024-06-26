/**
 * Redis connection, so we don't have to keep creating new connections
 */
import { createClient, RedisClientType } from "redis";
import { ErrorTrace } from "./events/error";
import { logger } from "./events/logger";

const split = (process.env.REDIS_CONN || "redis:6379").split(":");
export const HOST = split[0];
export const PORT = Number(split[1]);

let redisClient: RedisClientType;

export const createRedisClient = async () => {
    const url = `redis://${HOST}:${PORT}`;
    logger.info("Creating Redis client.", { trace: "0184", url });
    const redisClient = createClient({ url });
    redisClient.on("error", (error) => {
        logger.error("Error occured while connecting or accessing redis server", { trace: "0002", error });
    });
    redisClient.on("end", () => {
        logger.info("Redis client closed.", { trace: "0208", url });
    });
    await redisClient.connect();
    return redisClient;
};

export const initializeRedis = async (): Promise<RedisClientType> => {
    // If the client was never created or was disconnected
    if (!redisClient || !redisClient.isReady) {
        redisClient = await createRedisClient() as RedisClientType;
    }
    return redisClient;
};

interface WithRedisProps {
    process: (redisClient: RedisClientType) => Promise<void>,
    trace: string,
    traceObject?: ErrorTrace,
}

/**
 * Handles the Redis connection/disconnection and error logging
 */
export const withRedis = async ({
    process,
    trace,
    traceObject,
}: WithRedisProps): Promise<boolean> => {
    let success = false;
    try {
        const redis = await initializeRedis();
        await process(redis);
        success = true;
    } catch (error) {
        logger.error("Caught error in withRedis", { trace, error, ...traceObject });
    }
    return success;
};
