/**
 * Verify Setup Test
 * 
 * Simple test to verify execution tests have proper access to test infrastructure
 */

import { describe, it, expect } from "vitest";
import { DbProvider } from "../../db/provider.js";
import { getEventBus } from "../../services/events/eventBus.js";
import { generatePK } from "@vrooli/shared";

describe("Execution Test Infrastructure Verification", () => {
    it("should have access to database provider", async () => {
        // The main setup.ts should have initialized DbProvider
        const db = DbProvider.get();
        expect(db).toBeDefined();
        expect(db.user).toBeDefined();
        expect(db.routine).toBeDefined();
        expect(db.bot).toBeDefined();
        expect(db.team).toBeDefined();
    });

    it("should be able to create a test user", async () => {
        const db = DbProvider.get();
        const testUser = await db.user.create({
            data: {
                id: generatePK().toString(),
                name: "Test User for Execution Tests",
                isPrivate: false,
                handle: `test-exec-${Date.now()}`,
            },
        });

        expect(testUser).toBeDefined();
        expect(testUser.id).toBeDefined();
        expect(testUser.name).toBe("Test User for Execution Tests");

        // Clean up
        await db.user.delete({ where: { id: testUser.id } });
    });

    it("should have access to event bus", async () => {
        const eventBus = getEventBus();
        expect(eventBus).toBeDefined();

        // Test that we can subscribe
        const subscriptionId = await eventBus.subscribe(
            "test/execution/verify",
            () => {},
            { mode: "standard" },
        );
        expect(subscriptionId).toBeDefined();

        // Clean up
        await eventBus.unsubscribe(subscriptionId);
    });

    it("should have initialized shared ID generator", async () => {
        // Import shared to test if ID generator works
        const id1 = generatePK();
        const id2 = generatePK();
        
        expect(id1).toBeDefined();
        expect(id2).toBeDefined();
        expect(id1).not.toBe(id2); // Should generate unique IDs
    });
});
