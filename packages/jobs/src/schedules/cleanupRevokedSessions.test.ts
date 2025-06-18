import { DAYS_90_MS, generatePK, generatePublicId } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanupRevokedSessions } from "./cleanupRevokedSessions.js";

// Direct import to avoid problematic services
const { DbProvider } = await import("@vrooli/server");

describe("cleanupRevokedSessions integration tests", () => {
    // Store test entity IDs for cleanup
    const testUserIds: bigint[] = [];
    const testSessionIds: bigint[] = [];

    beforeEach(async () => {
        // Clear test ID arrays
        testUserIds.length = 0;
        testSessionIds.length = 0;
    });

    afterEach(async () => {
        // Clean up test data using collected IDs
        const db = DbProvider.get();
        await db.session.deleteMany({
            where: { id: { in: testSessionIds } },
        });
        await db.user.deleteMany({
            where: { id: { in: testUserIds } },
        });
    });

    it("should delete sessions revoked more than 90 days ago", async () => {
        // Create a test user
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Test User",
                handle: "testuser_cleanup",
            },
        });
        testUserIds.push(user.id);

        // Create auth record for the user
        const userAuth = await DbProvider.get().user_auth.create({
            data: {
                id: generatePK(),
                user_id: user.id,
                provider: "local",
            },
        });

        // Create sessions with different revoked dates
        const now = new Date();
        const oldDate = new Date(now.getTime() - DAYS_90_MS - 1000); // 90+ days ago
        const recentDate = new Date(now.getTime() - DAYS_90_MS + 1000); // Just under 90 days

        // Create old session (should be deleted)
        const oldSessionId = generatePK();
        testSessionIds.push(oldSessionId);
        await DbProvider.get().session.create({
            data: {
                id: oldSessionId,
                user_id: user.id,
                auth_id: userAuth.id,
                revokedAt: oldDate,
                expires_at: new Date(now.getTime() + 1000 * 60 * 60 * 24), // 24 hours from now
            },
        });

        // Create recent session (should NOT be deleted)
        const recentSessionId = generatePK();
        testSessionIds.push(recentSessionId);
        await DbProvider.get().session.create({
            data: {
                id: recentSessionId,
                user_id: user.id,
                auth_id: userAuth.id,
                revokedAt: recentDate,
                expires_at: new Date(now.getTime() + 1000 * 60 * 60 * 24), // 24 hours from now
            },
        });

        // Create active session (no revokedAt, should NOT be deleted)
        const activeSessionId = generatePK();
        testSessionIds.push(activeSessionId);
        await DbProvider.get().session.create({
            data: {
                id: activeSessionId,
                user_id: user.id,
                auth_id: userAuth.id,
                revokedAt: null,
                expires_at: new Date(now.getTime() + 1000 * 60 * 60 * 24), // 24 hours from now
            },
        });

        // Run the cleanup function
        await cleanupRevokedSessions();

        // Check that only the old session was deleted
        const remainingSessions = await DbProvider.get().session.findMany({
            where: { user_id: user.id },
            select: { id: true },
        });

        expect(remainingSessions).toHaveLength(2);
        const remainingIds = remainingSessions.map(s => s.id).sort((a, b) => a < b ? -1 : 1);
        const expectedIds = [activeSessionId, recentSessionId].sort((a, b) => a < b ? -1 : 1);
        expect(remainingIds).toEqual(expectedIds);
    });

    it("should handle empty results gracefully", async () => {
        // Run cleanup when there are no sessions to delete
        await cleanupRevokedSessions();

        // Should complete without errors
        expect(true).toBe(true);
    });

    it("should handle batch processing for multiple sessions", async () => {
        // Create a test user
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Batch Test User",
                handle: "testuser_batch",
            },
        });
        testUserIds.push(user.id);

        // Create auth record for the user
        const userAuth = await DbProvider.get().user_auth.create({
            data: {
                id: generatePK(),
                user_id: user.id,
                provider: "local",
            },
        });

        const oldDate = new Date(Date.now() - DAYS_90_MS - 1000);

        // Create multiple old sessions
        const sessionPromises = [];
        for (let i = 0; i < 5; i++) {
            const sessionId = generatePK();
            testSessionIds.push(sessionId);
            sessionPromises.push(
                DbProvider.get().session.create({
                    data: {
                        id: sessionId,
                        user_id: user.id,
                        auth_id: userAuth.id,
                        revokedAt: oldDate,
                        expires_at: new Date(Date.now() + 1000 * 60 * 60 * 24),
                    },
                }),
            );
        }
        await Promise.all(sessionPromises);

        // Run the cleanup function
        await cleanupRevokedSessions();

        // Check that all old sessions were deleted
        const remainingSessions = await DbProvider.get().session.findMany({
            where: { user_id: user.id },
        });

        expect(remainingSessions).toHaveLength(0);
    });
});
