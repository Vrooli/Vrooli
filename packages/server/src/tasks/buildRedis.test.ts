import { describe, it, expect, afterEach } from "vitest";
import IORedis from "ioredis";
import { buildRedis, clearRedisCache, closeRedisConnections } from "./queueFactory.js";
import "../__test/setup.js";

describe("buildRedis", () => {
    const redisUrl = process.env.REDIS_URL || "redis://localhost:6379";

    afterEach(async () => {
        // Clean up any remaining Redis connections
        try {
            await closeRedisConnections();
        } catch (error) {
            // Ignore errors during cleanup
        }
        // Clear the cache to ensure clean state
        clearRedisCache();
    });

    it("should create a new Redis connection", async () => {
        const client = await buildRedis(redisUrl);
        expect(client).toBeDefined();
        expect(client.status).toBe("ready");
        await client.quit();
    });

    it("should reuse existing connection for same URL", async () => {
        const client1 = await buildRedis(redisUrl);
        const client2 = await buildRedis(redisUrl);
        
        expect(client1).toBe(client2);
        expect(client1.status).toBe("ready");
        expect(client2.status).toBe("ready");
        
        // Don't quit in this test - let afterEach handle it
        // This avoids the double-quit issue
    });

    it("should handle concurrent connection requests", async () => {
        // Clear any existing connections
        const clients = (buildRedis as any).redisClients || {};
        Object.keys(clients).forEach(key => delete clients[key]);

        // Make multiple concurrent requests
        const promises = Array.from({ length: 5 }, () => buildRedis(redisUrl));
        const connections = await Promise.all(promises);

        // All should be the same instance
        const firstConnection = connections[0];
        connections.forEach(conn => {
            expect(conn).toBe(firstConnection);
        });

        // Don't quit here - let afterEach handle it
        // This avoids the double-quit issue
    });

    it("should handle connection errors", async () => {
        const invalidUrl = "redis://invalid-host:6379";
        await expect(buildRedis(invalidUrl)).rejects.toThrow();
        
        // Immediately clear the Redis cache to prevent pollution
        clearRedisCache();
    });
});
