/**
 * Redis connection, so we don't have to keep creating new connections
 */
import { createClient, RedisClientType } from "redis";
import { logger } from "./events/logger";

const split = (process.env.REDIS_CONN || "redis:6379").split(":");
export const HOST = split[0];
export const PORT = Number(split[1]);

let redisClient: RedisClientType;

const createRedisClient = async () => {
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
    const _redisClient = redisClient ?? await createRedisClient();
    if (!redisClient) redisClient = _redisClient;

    return _redisClient;
};
