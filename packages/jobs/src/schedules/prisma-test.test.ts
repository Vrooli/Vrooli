import { describe, expect, it } from "vitest";
import { PrismaClient } from "@prisma/client";

describe("prisma test", () => {
    it("should create prisma client", async () => {
        console.log("Creating PrismaClient...");
        const prisma = new PrismaClient();
        console.log("PrismaClient created");
        
        // Try a simple query
        console.log("Running query...");
        const userCount = await prisma.user.count();
        console.log(`User count: ${userCount}`);
        
        expect(userCount).toBeGreaterThanOrEqual(0);
        
        // Disconnect
        console.log("Disconnecting...");
        await prisma.$disconnect();
        console.log("Disconnected");
    });
});