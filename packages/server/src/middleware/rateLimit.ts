import { Request } from "express";
import { RedisClientType } from "redis";
import { getUser } from "../auth/request";
import { CustomError } from "../events/error";
import { logger } from "../events/logger";
import { initializeRedis } from "../redisConn";

/**
 * Applies a rate limit check to a key in redis. 
 * Throws an error if the limit is exceeded.
 * @param client The redis client
 * @param key The key to check
 * @param max The max to check against
 * @param window The window for the limit
 * @param languages Languages preferred to display error message
 */
export async function checkRateLimit(
    client: RedisClientType | null,
    key: string,
    max: number,
    window: number,
    languages: string[],
) {
    // Increment and get the current count.
    const count = client ? await client.incr(key) : 0;
    // If limit reached, throw error.
    if (count > max) {
        throw new CustomError("0017", "RateLimitExceeded", languages);
    }
    // If key is new, set expiration.
    if (client && count === 1) {
        await client.expire(key, window);
    }
}

export interface RateLimitProps {
    /**
     * Maximum number of requests allowed per window, tied to API key (if not made from a safe origin)
     */
    maxApi?: number;
    /**
     * Maximum number of requests allowed per window, tied to IP address (if API key is not supplied)
     */
    maxIp?: number;
    /**
     * Maximum number of requests allowed per window, tied to user (regardless of origin)
     */
    maxUser?: number;
    req: Request;
    window?: number;
}

/**
 * Middelware to rate limit the requests. 
 * Limits requests based on API token, account, and IP address.
 * Throws error if rate limit is exceeded.
 */
export async function rateLimit({
    maxApi,
    maxIp,
    maxUser = 250,
    req,
    window = 60 * 60 * 24,
}: RateLimitProps): Promise<void> {
    // Create key that uniquely identifies the endpoint
    let keyBase = "rate-limit:";
    // For GraphQL requests, use the operation name
    if (req.body?.operationName) {
        keyBase += `${req.body.operationName}:`;
    }
    // For REST requests, use the route path and method
    else if (req.route) {
        keyBase += `${req.route.path}:${req.method}:`;
    }
    // For other requests (typically when req is mocked by a task queue), use the path
    else {
        keyBase += `${req.path}:`;
    }
    // If maxApi not supplied, use maxUser * 1000
    maxApi = maxApi ?? (maxUser * 1000);
    // If maxIp not supplied, use maxUser
    maxIp = maxIp ?? maxUser;
    // Parse request
    const hasApiToken = req.session.apiToken === true;
    const userData = getUser(req.session);
    const hasUserData = req.session.isLoggedIn === true && userData !== null;
    // Try connecting to redis
    // TODO should use `withRedis` function to let users through if Redis fails, but that would 
    // also catch the errors we want to throw
    try {
        const client = await initializeRedis();
        // Apply rate limit to API TODO factor in cost of request, instead of just incrementing by 1
        if (hasApiToken) {
            const key = `${keyBase}:api:${req.session.apiToken}`;
            await checkRateLimit(client, key, maxApi, window, req.session.languages);
        }
        // If API token is not supplied, make sure request is from the official Vrooli app/website
        else if (req.session.fromSafeOrigin === false) {
            throw new CustomError("0271", "MustUseApiToken", req.session.languages, { keyBase });
        }
        // Apply rate limit to IP address
        const key = `${keyBase}:ip:${req.ip}`;
        await checkRateLimit(client, key, maxIp, window, req.session.languages);
        // Apply rate limit to user
        if (hasUserData) {
            const key = `${keyBase}:user:${userData.id}`;
            await checkRateLimit(client, key, maxUser, window, req.session.languages);
        }
    }
    // If Redis fails, let the user through. It's not their fault. 
    catch (error) {
        logger.error("Error occured while connecting or accessing redis server", { trace: "0168", error });
        throw error;
    }
}
