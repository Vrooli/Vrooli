import { Request } from "express";
import { GraphQLResolveInfo } from "graphql";
import { initializeRedis } from "../redisConn";
import { CustomError } from "../events/error";
import { CODE } from "@shared/consts";
import { genErrorCode, logger, LogLevel } from "../events/logger";
import { getUser } from "../models";

/**
 * Applies a rate limit check to a key in redis. 
 * Throws an error if the limit is exceeded.
 * @param client The redis client
 * @param key The key to check
 * @param max The max to check against
 * @param window The window for the limit
 */
export async function checkRateLimit(client: any, key: string, max: number, window: number) {
    // If 
    // Increment and get the current count.
    const count = await client.incr(key);
    // If limit reached, throw error.
    if (count > max) {
        throw new CustomError(CODE.RateLimitExceeded, `Rate limit exceeded. Please try again in ${window} seconds.`, { code: genErrorCode('0017') });
    }
    // If key is new, set expiration.
    if (count === 1) {
        await client.expire(key, window);
    }
}

export interface RateLimitProps {
    info: GraphQLResolveInfo;
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
    info,
    maxApi,
    maxIp,
    maxUser = 250,
    req,
    window = 60 * 60 * 24,
}: RateLimitProps): Promise<void> {
    const keyBase = `rate-limit:${info.path.key}`;
    // If maxApi not supplied, use maxUser * 1000
    maxApi = maxApi ?? (maxUser * 1000);
    // If maxIp not supplied, use maxUser
    maxIp = maxIp ?? maxUser;
    // Parse request
    const hasApiToken = req.apiToken === true;
    const userData = getUser(req);
    const hasUserData = req.isLoggedIn === true && userData !== null;
    // Try connecting to redis
    try {
        const client = await initializeRedis();
        // Apply rate limit to API TODO factor in cost of request, instead of just incrementing by 1
        if (hasApiToken) {
            const key = `${keyBase}:api:${req.apiToken}`;
            await checkRateLimit(client, key, maxApi, window);
        }
        // If API token is not supplied, make sure request is from the official Vrooli app/website
        else if (req.fromSafeOrigin === false) {
            throw new CustomError(CODE.Unauthorized, 'Call is missing an API key.', { code: genErrorCode('0271') });
        }
        // Apply rate limit to IP address
        const key = `${keyBase}:ip:${req.ip}`;
        await checkRateLimit(client, key, maxIp, window);
        // Apply rate limit to user
        if (hasUserData) {
            const key = `${keyBase}:user:${userData.id}`;
            await checkRateLimit(client, key, maxUser, window);
        }
    }
    // If Redis fails, let the user through. It's not their fault. 
    catch (error) {
        logger.log(LogLevel.error, 'Error occured while connecting or accessing redis server', { code: genErrorCode('0168'), error });
    }
}