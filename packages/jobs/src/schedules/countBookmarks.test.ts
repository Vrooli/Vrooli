import { generatePK, generatePublicId } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { countBookmarks } from "./countBookmarks.js";

// Direct import to avoid problematic services
const { DbProvider } = await import("../../../server/src/db/provider.ts");

describe("countBookmarks integration tests", () => {
    // Store test entity IDs for cleanup
    const testUserIds: bigint[] = [];
    const testCommentIds: bigint[] = [];
    const testIssueIds: bigint[] = [];
    const testResourceIds: bigint[] = [];
    const testTagIds: bigint[] = [];
    const testTeamIds: bigint[] = [];
    const testBookmarkIds: bigint[] = [];

    beforeEach(async () => {
        // Clear test ID arrays
        testUserIds.length = 0;
        testCommentIds.length = 0;
        testIssueIds.length = 0;
        testResourceIds.length = 0;
        testTagIds.length = 0;
        testTeamIds.length = 0;
        testBookmarkIds.length = 0;
    });

    afterEach(async () => {
        // Clean up test data using collected IDs
        await DbProvider.get().$transaction([
            DbProvider.get().bookmark.deleteMany({
                where: { id: { in: testBookmarkIds } },
            }),
            DbProvider.get().comment.deleteMany({
                where: { id: { in: testCommentIds } },
            }),
            DbProvider.get().issue.deleteMany({
                where: { id: { in: testIssueIds } },
            }),
            DbProvider.get().resource.deleteMany({
                where: { id: { in: testResourceIds } },
            }),
            DbProvider.get().tag.deleteMany({
                where: { id: { in: testTagIds } },
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
        // Create test users
        const bookmarker1 = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Bookmarker 1",
                handle: "bookmarker1",
            },
        });
        testUserIds.push(bookmarker1.id);

        const bookmarker2 = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Bookmarker 2",
                handle: "bookmarker2",
            },
        });
        testUserIds.push(bookmarker2.id);

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner",
                handle: "owner1",
            },
        });
        testUserIds.push(owner.id);

        // Create test issue with incorrect bookmark count
        const issue = await DbProvider.get().issue.create({
            data: {
                id: generatePK(),
                createdById: owner.id,
                name: "Test Issue",
                description: "Test Description",
                bookmarks: 10, // Incorrect count
            },
        });
        testIssueIds.push(issue.id);

        // Create actual bookmarks
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

        // Run the count bookmarks function
        await countBookmarks();

        // Check that the bookmark count was updated correctly
        const updatedIssue = await DbProvider.get().issue.findUnique({
            where: { id: issue.id },
            select: { bookmarks: true },
        });

        expect(updatedIssue?.bookmarks).toBe(2); // Should match actual bookmark count
    });

    it("should handle null bookmark counts", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner 2",
                handle: "owner2",
            },
        });
        testUserIds.push(owner.id);

        const bookmarker = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Bookmarker 3",
                handle: "bookmarker3",
            },
        });
        testUserIds.push(bookmarker.id);

        // Create test tag with null bookmark count
        const tag = await DbProvider.get().tag.create({
            data: {
                id: generatePK(),
                createdById: owner.id,
                tag: "test-tag",
                bookmarks: null, // Null count
            },
        });
        testTagIds.push(tag.id);

        // Create actual bookmark
        const bookmarkId = generatePK();
        testBookmarkIds.push(bookmarkId);
        await DbProvider.get().bookmark.create({
            data: {
                id: bookmarkId,
                tagId: tag.id,
            },
        });

        // Run the count bookmarks function
        await countBookmarks();

        // Check that the bookmark count was updated from null to actual count
        const updatedTag = await DbProvider.get().tag.findUnique({
            where: { id: tag.id },
            select: { bookmarks: true },
        });

        expect(updatedTag?.bookmarks).toBe(1);
    });

    it("should handle multiple entity types", async () => {
        const bookmarker = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Bookmarker 4",
                handle: "bookmarker4",
            },
        });
        testUserIds.push(bookmarker.id);

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner 3",
                handle: "owner3",
                bookmarks: 5, // Incorrect count
            },
        });
        testUserIds.push(owner.id);

        const team = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                handle: "testteam1",
                bookmarks: 3, // Incorrect count
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

        const resource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                resourceType: "RoutineVersion",
                bookmarks: null, // Null count
            },
        });
        testResourceIds.push(resource.id);

        // Create bookmarks for user
        const userBookmarkId = generatePK();
        testBookmarkIds.push(userBookmarkId);
        await DbProvider.get().bookmark.create({
            data: {
                id: userBookmarkId,
                userId: owner.id,
            },
        });

        // Create bookmarks for team
        const teamBookmark1Id = generatePK();
        const teamBookmark2Id = generatePK();
        testBookmarkIds.push(teamBookmark1Id, teamBookmark2Id);
        await DbProvider.get().bookmark.createMany({
            data: [
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

        // Create bookmark for resource
        const resourceBookmarkId = generatePK();
        testBookmarkIds.push(resourceBookmarkId);
        await DbProvider.get().bookmark.create({
            data: {
                id: resourceBookmarkId,
                resourceId: resource.id,
            },
        });

        // Run the count bookmarks function
        await countBookmarks();

        // Check that all counts were updated
        const updatedUser = await DbProvider.get().user.findUnique({
            where: { id: owner.id },
            select: { bookmarks: true },
        });

        const updatedTeam = await DbProvider.get().team.findUnique({
            where: { id: team.id },
            select: { bookmarks: true },
        });

        const updatedResource = await DbProvider.get().resource.findUnique({
            where: { id: resource.id },
            select: { bookmarks: true },
        });

        expect(updatedUser?.bookmarks).toBe(1); // 1 bookmark
        expect(updatedTeam?.bookmarks).toBe(2); // 2 bookmarks
        expect(updatedResource?.bookmarks).toBe(1); // 1 bookmark (was null)
    });

    it("should handle entities with no bookmarks", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner 4",
                handle: "owner4",
            },
        });
        testUserIds.push(owner.id);

        // Create entities with non-zero bookmark counts but no actual bookmarks
        const issue = await DbProvider.get().issue.create({
            data: {
                id: generatePK(),
                createdById: owner.id,
                name: "Issue with no bookmarks",
                description: "Should be reset to 0",
                bookmarks: 15, // Should be corrected to 0
            },
        });
        testIssueIds.push(issue.id);

        const comment = await DbProvider.get().comment.create({
            data: {
                id: generatePK(),
                createdById: owner.id,
                issueId: issue.id,
                body: "Test comment",
                bookmarks: 7, // Should be corrected to 0
            },
        });
        testCommentIds.push(comment.id);

        // Run the count bookmarks function
        await countBookmarks();

        // Check that the counts were corrected to 0
        const correctedIssue = await DbProvider.get().issue.findUnique({
            where: { id: issue.id },
            select: { bookmarks: true },
        });

        const correctedComment = await DbProvider.get().comment.findUnique({
            where: { id: comment.id },
            select: { bookmarks: true },
        });

        expect(correctedIssue?.bookmarks).toBe(0);
        expect(correctedComment?.bookmarks).toBe(0);
    });

    it("should process all supported entity types", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner 5",
                handle: "owner5",
            },
        });
        testUserIds.push(owner.id);

        // Create one entity of each type with incorrect counts
        const issue = await DbProvider.get().issue.create({
            data: {
                id: generatePK(),
                createdById: owner.id,
                name: "Test",
                description: "Test",
                bookmarks: 5,
            },
        });
        testIssueIds.push(issue.id);

        const comment = await DbProvider.get().comment.create({
            data: {
                id: generatePK(),
                createdById: owner.id,
                issueId: issue.id,
                body: "Test",
                bookmarks: 5,
            },
        });
        testCommentIds.push(comment.id);

        const resource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                resourceType: "RoutineVersion",
                bookmarks: 5,
            },
        });
        testResourceIds.push(resource.id);

        const tag = await DbProvider.get().tag.create({
            data: {
                id: generatePK(),
                createdById: owner.id,
                tag: "testtag",
                bookmarks: 5,
            },
        });
        testTagIds.push(tag.id);

        const team = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                handle: "testteam2",
                bookmarks: 5,
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

        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Test User",
                handle: "testuser6",
                bookmarks: 5,
            },
        });
        testUserIds.push(user.id);

        // Run the count bookmarks function
        await countBookmarks();

        // Check that all counts were corrected to 0
        const results = await DbProvider.get().$transaction([
            DbProvider.get().comment.findUnique({ where: { id: comment.id }, select: { bookmarks: true } }),
            DbProvider.get().issue.findUnique({ where: { id: issue.id }, select: { bookmarks: true } }),
            DbProvider.get().resource.findUnique({ where: { id: resource.id }, select: { bookmarks: true } }),
            DbProvider.get().tag.findUnique({ where: { id: tag.id }, select: { bookmarks: true } }),
            DbProvider.get().team.findUnique({ where: { id: team.id }, select: { bookmarks: true } }),
            DbProvider.get().user.findUnique({ where: { id: user.id }, select: { bookmarks: true } }),
        ]);

        results.forEach(result => {
            expect(result?.bookmarks).toBe(0);
        });
    });
});
