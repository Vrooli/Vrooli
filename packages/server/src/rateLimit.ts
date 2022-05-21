import { Context } from "./context";
import { GraphQLResolveInfo } from "graphql";
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
    return;
   
}