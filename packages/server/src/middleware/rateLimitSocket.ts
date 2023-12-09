import { Socket } from "socket.io";
import { getUser } from "../auth/request";
import { initializeRedis } from "../redisConn";
import { checkRateLimit } from "./rateLimit";

// Rate limit props
interface SocketRateLimitProps {
    maxIp?: number;
    maxUser?: number;
    window?: number;
    socket: Socket;
}

/**
 * Rate limit a socket handler, similar to how you would rate limit an API endpoint. 
 * Returns instead of throws an error if rate limit is exceeded.
 */
export async function rateLimitSocket({
    maxIp,
    maxUser = 250,
    window = 60 * 60 * 24,
    socket,
}: SocketRateLimitProps): Promise<string | undefined> {
    const keyBase = "rate-limit:";
    // Retrieve user data from the socket
    const userData = getUser(socket.session);
    const hasUserData = socket.session.isLoggedIn === true && userData !== null;
    // If maxIp not supplied, use maxUser
    maxIp = maxIp ?? maxUser;
    // Parse socket
    const keyBaseWithId = `${keyBase}:${socket.id}:`;
    // Try connecting to redis
    // TODO should use `withRedis` function to let users through if Redis fails, but that would 
    // also catch the errors we want to throw
    try {
        const client = await initializeRedis();
        // Apply rate limit to IP address
        const key = `${keyBase}:ip:${socket.req.ip}`;
        await checkRateLimit(client, key, maxIp, window, socket.session.languages);
        // Apply rate limit to user
        if (hasUserData) {
            const key = `${keyBaseWithId}:user:${userData.id}`;
            await checkRateLimit(client, key, maxUser, window, socket.session.languages);
        }
    } catch (error) {
        console.error("Error occurred while connecting or accessing redis server", { trace: "0492", error });
        return (error as any)?.message ?? "Rate limit exceeded";
    }
}
