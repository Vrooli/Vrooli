import { getUser } from "../auth";
import { CustomError } from "../events/error";
import { logger } from "../events/logger";
import { initializeRedis } from "../redisConn";
export async function checkRateLimit(client, key, max, window, languages) {
    const count = await client.incr(key);
    if (count > max) {
        throw new CustomError("0017", "RateLimitExceeded", languages);
    }
    if (count === 1) {
        await client.expire(key, window);
    }
}
export async function rateLimit({ info, maxApi, maxIp, maxUser = 250, req, window = 60 * 60 * 24, }) {
    const keyBase = `rate-limit:${info.path.key}`;
    maxApi = maxApi ?? (maxUser * 1000);
    maxIp = maxIp ?? maxUser;
    const hasApiToken = req.apiToken === true;
    const userData = getUser(req);
    const hasUserData = req.isLoggedIn === true && userData !== null;
    try {
        const client = await initializeRedis();
        if (hasApiToken) {
            const key = `${keyBase}:api:${req.apiToken}`;
            await checkRateLimit(client, key, maxApi, window, req.languages);
        }
        else if (req.fromSafeOrigin === false) {
            throw new CustomError("0271", "MustUseApiToken", req.languages);
        }
        const key = `${keyBase}:ip:${req.ip}`;
        await checkRateLimit(client, key, maxIp, window, req.languages);
        if (hasUserData) {
            const key = `${keyBase}:user:${userData.id}`;
            await checkRateLimit(client, key, maxUser, window, req.languages);
        }
    }
    catch (error) {
        logger.error("Error occured while connecting or accessing redis server", { trace: "0168", error });
    }
}
//# sourceMappingURL=rateLimit.js.map