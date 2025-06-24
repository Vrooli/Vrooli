// AI_CHECK: TEST_COVERAGE=2,TEST_QUALITY=1 | LAST: 2025-06-24
import { generatePK, generatePublicId } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { verifyCountField } from "./verifyCountField.js";

// Direct import to use real services (integration testing pattern)
const { DbProvider } = await import("@vrooli/server");

describe("verifyCountField integration tests", () => {
    // Store test entity IDs for cleanup
    const testUserIds: bigint[] = [];
    const testTeamIds: bigint[] = [];
    const testIssueIds: bigint[] = [];
    const testBookmarkIds: bigint[] = [];

    beforeEach(async () => {
        // Clear test ID arrays
        testUserIds.length = 0;
        testTeamIds.length = 0;
        testIssueIds.length = 0;
        testBookmarkIds.length = 0;
    });

    afterEach(async () => {
        // Clean up test data using collected IDs
        await DbProvider.get().$transaction([
            DbProvider.get().bookmark.deleteMany({
                where: { id: { in: testBookmarkIds } },
            }),
            DbProvider.get().issue.deleteMany({
                where: { id: { in: testIssueIds } },
            }),
            DbProvider.get().team.deleteMany({
                where: { id: { in: testTeamIds } },
            }),
            DbProvider.get().user.deleteMany({
                where: { id: { in: testUserIds } },
            }),
        ]);
    });

    it("should update bookmark counts when they mismatch", async () => {
        // Create test user (owner of the issue)
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Issue Owner",
                handle: `issue_owner_${Date.now()}`,
            },
        });
        testUserIds.push(owner.id);

        // Create test issue with incorrect bookmark count
        const issue = await DbProvider.get().issue.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                bookmarks: 10, // Incorrect count - should be 2
                translations: {
                    create: {
                        id: generatePK(),
                        language: "en",
                        name: "Test Issue",
                        description: "Test Description",
                    },
                },
            },
        });
        testIssueIds.push(issue.id);

        // Create actual bookmarks for the issue
        const bookmark1Id = generatePK();
        const bookmark2Id = generatePK();
        testBookmarkIds.push(bookmark1Id, bookmark2Id);
        await DbProvider.get().bookmark.createMany({
            data: [
                {
                    id: bookmark1Id,
                    issueId: issue.id,
                },
                {
                    id: bookmark2Id,
                    issueId: issue.id,
                },
            ],
        });

        // Run verifyCountField
        const config = {
            tableNames: ["issue"] as const,
            countField: "bookmarks",
            relationName: "bookmarkedBy",
            select: { id: true, bookmarks: true, _count: { select: { bookmarkedBy: true } } },
            traceId: "test-trace",
        };

        await verifyCountField(config);

        // Check that the bookmark count was updated correctly
        const updatedIssue = await DbProvider.get().issue.findUnique({
            where: { id: issue.id },
            select: { bookmarks: true },
        });

        expect(updatedIssue?.bookmarks).toBe(2); // Should match actual bookmark count
    });

    it("should handle zero bookmark counts", async () => {
        // Create user with non-zero bookmark count but no actual bookmarks
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "User No Bookmarks",
                handle: `user_no_bookmarks_${Date.now()}`,
                bookmarks: 5, // Should be corrected to 0
            },
        });
        testUserIds.push(user.id);

        // Run verifyCountField
        const config = {
            tableNames: ["user"] as const,
            countField: "bookmarks",
            relationName: "bookmarkedBy",
            select: { id: true, bookmarks: true, _count: { select: { bookmarkedBy: true } } },
            traceId: "test-trace",
        };

        await verifyCountField(config);

        // Check that the bookmark count was corrected to 0
        const updatedUser = await DbProvider.get().user.findUnique({
            where: { id: user.id },
            select: { bookmarks: true },
        });

        expect(updatedUser?.bookmarks).toBe(0);
    });

    it("should handle multiple entity types", async () => {
        const uniqueId = Date.now();
        
        // Create user with incorrect bookmark count
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "User Multi",
                handle: `user_multi_${uniqueId}`,
                bookmarks: 10, // Should be 1
            },
        });
        testUserIds.push(user.id);

        // Create team with incorrect bookmark count
        const team = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: user.id,
                handle: `team_multi_${uniqueId}`,
                bookmarks: 10, // Should be 2
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "Test Team",
                    }],
                },
            },
        });
        testTeamIds.push(team.id);

        // Create bookmarks
        const userBookmarkId = generatePK();
        const teamBookmark1Id = generatePK();
        const teamBookmark2Id = generatePK();
        testBookmarkIds.push(userBookmarkId, teamBookmark1Id, teamBookmark2Id);
        
        await DbProvider.get().bookmark.createMany({
            data: [
                {
                    id: userBookmarkId,
                    userId: user.id,
                },
                {
                    id: teamBookmark1Id,
                    teamId: team.id,
                },
                {
                    id: teamBookmark2Id,
                    teamId: team.id,
                },
            ],
        });

        // Run verifyCountField for both users and teams
        const config = {
            tableNames: ["user", "team"] as const,
            countField: "bookmarks",
            relationName: "bookmarkedBy",
            select: { id: true, bookmarks: true, _count: { select: { bookmarkedBy: true } } },
            traceId: "test-trace",
        };

        await verifyCountField(config);

        // Check that all counts were updated
        const updatedUser = await DbProvider.get().user.findUnique({
            where: { id: user.id },
            select: { bookmarks: true },
        });

        const updatedTeam = await DbProvider.get().team.findUnique({
            where: { id: team.id },
            select: { bookmarks: true },
        });

        expect(updatedUser?.bookmarks).toBe(1); // 1 bookmark
        expect(updatedTeam?.bookmarks).toBe(2); // 2 bookmarks
    });

    it("should not update when cached and actual counts match", async () => {
        // Create user with correct bookmark count
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "User Correct Count",
                handle: `user_correct_${Date.now()}`,
                bookmarks: 2, // Correct count
            },
        });
        testUserIds.push(user.id);

        // Create matching number of actual bookmarks
        const bookmark1Id = generatePK();
        const bookmark2Id = generatePK();
        testBookmarkIds.push(bookmark1Id, bookmark2Id);
        await DbProvider.get().bookmark.createMany({
            data: [
                {
                    id: bookmark1Id,
                    userId: user.id,
                },
                {
                    id: bookmark2Id,
                    userId: user.id,
                },
            ],
        });

        // Run verifyCountField
        const config = {
            tableNames: ["user"] as const,
            countField: "bookmarks",
            relationName: "bookmarkedBy",
            select: { id: true, bookmarks: true, _count: { select: { bookmarkedBy: true } } },
            traceId: "test-trace",
        };

        await verifyCountField(config);

        // Check that the bookmark count remains unchanged
        const updatedUser = await DbProvider.get().user.findUnique({
            where: { id: user.id },
            select: { bookmarks: true },
        });

        expect(updatedUser?.bookmarks).toBe(2); // Should remain 2
    });

    it("should handle empty table names array", async () => {
        const config = {
            tableNames: [] as const,
            countField: "bookmarks",
            relationName: "bookmarkedBy",
            select: { id: true },
            traceId: "test-trace",
        };

        // This should complete without errors
        await expect(verifyCountField(config)).resolves.not.toThrow();
    });

});
