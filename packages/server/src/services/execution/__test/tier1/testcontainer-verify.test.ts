import { expect, describe, it } from "vitest";
import IORedis from "ioredis";
import { PrismaClient } from "@prisma/client";

describe("Testcontainer Verification", () => {
    it("should connect to Redis via testcontainer", async () => {
        // Create Redis connection using URL from environment
        const redis = new IORedis(process.env.REDIS_URL!);
        
        // Try to set and get a value
        const key = "test:verification";
        const value = "testcontainer-works";
        
        await redis.set(key, value);
        const retrieved = await redis.get(key);
        
        expect(retrieved).toBe(value);
        
        await redis.quit();
    });

    it("should connect to PostgreSQL via testcontainer", async () => {
        // Create Prisma client using URL from environment
        const prisma = new PrismaClient({
            datasources: {
                db: {
                    url: process.env.DB_URL,
                },
            },
        });
        
        // Try to run a simple query
        const result = await prisma.$queryRaw`SELECT 1 as test`;
        
        expect(result).to.not.be.undefined;
        expect(Array.isArray(result)).toBe(true);
        expect(result[0]).toHaveProperty("test", 1);
        
        await prisma.$disconnect();
    });

    it("should have proper environment variables set", () => {
        expect(process.env.REDIS_URL).to.not.be.undefined;
        expect(process.env.REDIS_URL).to.match(/redis:\/\//);
        
        expect(process.env.DB_URL).to.not.be.undefined;
        expect(process.env.DB_URL).to.match(/postgresql:\/\//);
    });
});