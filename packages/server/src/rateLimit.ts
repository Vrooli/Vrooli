import { Request } from "express";
import { GraphQLResolveInfo } from "graphql";
import { initializeRedis } from "./redisConn";
import { CustomError } from "./error";
import { CODE } from "@local/shared";
import { genErrorCode, logger, LogLevel } from "./logger";

export interface RateLimitProps {
    /**
     * True if rate limit should be applied to API key or user, 
     * rather than IP address.
     */
    byAccountOrKey?: boolean;
    info: GraphQLResolveInfo;
    /**
     * Maximum number of requests allowed per window. Different than the 
     * API key rate limit, which happens over a longer period of time.
     */
    max?: number;
    req: Request;
    window?: number;
}

/**
 * Middelware to rate limit the requests. 
 * Uses userId for authenticated users and ip for unauthenticated users.
 * Tracks request count using redis.
 * Throws error if rate limit is exceeded.
 */
export async function rateLimit({
    byAccountOrKey = false,
    info,
    max = 1000,
    req,
    window = 60 * 60 * 24,
}: RateLimitProps): Promise<void> {
    if (byAccountOrKey && !req.userId) throw new CustomError(CODE.Unauthorized, "Calling rateLimit with 'byAccountOrKey' set to true, but with an invalid or expired JWT", { code: genErrorCode('0015') });
    // Unique key for this request. Combination of GraphQL endpoint and userId/ip.
    const key = `rate-limit:${info.path.key}:${byAccountOrKey ? (req.apiToken ?? req.userId) : req.ip}`;
    try {
        const client = await initializeRedis();
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
    // If Redis fails, let the user through. It's not their fault. 
    catch (error) {
        logger.log(LogLevel.error, 'Error occured while connecting or accessing redis server', { code: genErrorCode('0168'), error });
    }
    // TODO also calculate cost of API request and add to redis. Will have to find a way to map request to a cost function
}