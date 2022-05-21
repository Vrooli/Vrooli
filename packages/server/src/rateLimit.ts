import { Context } from "./context";
import { GraphQLResolveInfo } from "graphql";
import { initializeRedis } from "./redisConn";
import { CustomError } from "./error";
import { CODE } from "@local/shared";
import { genErrorCode, logger, LogLevel } from "./logger";

export interface RateLimitProps {
    byAccount?: boolean;
    context: Context;
    info: GraphQLResolveInfo;
    max?: number;
    window?: number;
}

/**
 * Middelware to rate limit the requests. 
 * Uses userId for authenticated users and ip for unauthenticated users.
 * Tracks request count using redis.
 * Throws error if rate limit is exceeded.
 * @param context 
 * @param max The maximum number of requests per user/ip, per unit of time. Defaults to 100.
 * @param window The unit of time for the limit. Defaults to 1 day.
 * @param byAccount Whether to use userId or ip. Defaults to false (i.e. ip).
 */
export async function rateLimit({
    byAccount = false,
    context, 
    info,
    max = 1000,
    window = 60 * 60 * 24,
}: RateLimitProps): Promise<void> {
    if (byAccount && !context.req.userId) throw new CustomError(CODE.Unauthorized, "If calling rateLimit with 'byAccount' set to true, you must be logged in", { code: genErrorCode('0015') });
    // Unique key for this request. Combination of GraphQL endpoint and userId/ip.
    const key = `rate-limit:${info.path.key}:${byAccount ? context.req.userId : context.req.ip}`;
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
}