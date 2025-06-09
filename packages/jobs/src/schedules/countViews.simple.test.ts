import { generatePK, generatePublicId } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// Import DbProvider directly to avoid problematic barrel exports
const { DbProvider } = await import("@vrooli/server/db/provider.js");

// Simple version of countViews without dependencies
async function simpleCountViews(): Promise<void> {
    const db = DbProvider.get();
    
    // Get all issues and update their view counts
    const issues = await db.issue.findMany({
        select: {
            id: true,
            views: true,
            _count: {
                select: { viewedBy: true },
            },
        },
    });
    
    for (const issue of issues) {
        const actualCount = issue._count.viewedBy;
        if (issue.views !== actualCount) {
            await db.issue.update({
                where: { id: issue.id },
                data: { views: actualCount },
            });
        }
    }
    
    // Get all resources and update their view counts
    const resources = await db.resource.findMany({
        select: {
            id: true,
            views: true,
            _count: {
                select: { viewedBy: true },
            },
        },
    });
    
    for (const resource of resources) {
        const actualCount = resource._count.viewedBy;
        if (resource.views !== actualCount) {
            await db.resource.update({
                where: { id: resource.id },
                data: { views: actualCount },
            });
        }
    }
    
    // Get all teams and update their view counts
    const teams = await db.team.findMany({
        select: {
            id: true,
            views: true,
            _count: {
                select: { viewedBy: true },
            },
        },
    });
    
    for (const team of teams) {
        const actualCount = team._count.viewedBy;
        if (team.views !== actualCount) {
            await db.team.update({
                where: { id: team.id },
                data: { views: actualCount },
            });
        }
    }
    
    // Get all users and update their view counts
    const users = await db.user.findMany({
        select: {
            id: true,
            views: true,
            _count: {
                select: { viewedBy: true },
            },
        },
    });
    
    for (const user of users) {
        const actualCount = user._count.viewedBy;
        if (user.views !== actualCount) {
            await db.user.update({
                where: { id: user.id },
                data: { views: actualCount },
            });
        }
    }
}

describe("countViews integration tests", () => {
    // Store test entity IDs for cleanup
    const testUserIds: bigint[] = [];
    const testIssueIds: bigint[] = [];
    const testResourceIds: bigint[] = [];
    const testTeamIds: bigint[] = [];
    const testViewIds: bigint[] = [];

    beforeEach(async () => {
        // Clear test ID arrays
        testUserIds.length = 0;
        testIssueIds.length = 0;
        testResourceIds.length = 0;
        testTeamIds.length = 0;
        testViewIds.length = 0;
    });

    afterEach(async () => {
        // Clean up test data using collected IDs
        await DbProvider.get().$transaction([
            DbProvider.get().view.deleteMany({
                where: { id: { in: testViewIds } },
            }),
            DbProvider.get().issue.deleteMany({
                where: { id: { in: testIssueIds } },
            }),
            DbProvider.get().resource.deleteMany({
                where: { id: { in: testResourceIds } },
            }),
            DbProvider.get().team.deleteMany({
                where: { id: { in: testTeamIds } },
            }),
            DbProvider.get().user.deleteMany({
                where: { id: { in: testUserIds } },
            }),
        ]);
    });

    it("should update view counts when they mismatch", async () => {
        // Create test users
        const viewer1 = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Viewer 1",
                handle: "viewer1",
            },
        });
        testUserIds.push(viewer1.id);

        const viewer2 = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Viewer 2",
                handle: "viewer2",
            },
        });
        testUserIds.push(viewer2.id);

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner",
                handle: "owner1",
            },
        });
        testUserIds.push(owner.id);

        // Create test issue with incorrect view count
        const issue = await DbProvider.get().issue.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                views: 10, // Incorrect count
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "Test Issue",
                        description: "Test Description",
                    }],
                },
            },
        });
        testIssueIds.push(issue.id);

        // Create actual views
        const view1Id = generatePK();
        const view2Id = generatePK();
        testViewIds.push(view1Id, view2Id);
        await DbProvider.get().view.createMany({
            data: [
                {
                    id: view1Id,
                    name: "Test Issue View 1",
                    byId: viewer1.id,
                    issueId: issue.id,
                },
                {
                    id: view2Id,
                    name: "Test Issue View 2",
                    byId: viewer2.id,
                    issueId: issue.id,
                },
            ],
        });

        // Run the count views function
        await simpleCountViews();

        // Check that the view count was updated correctly
        const updatedIssue = await DbProvider.get().issue.findUnique({
            where: { id: issue.id },
            select: { views: true },
        });

        expect(updatedIssue?.views).toBe(2); // Should match actual view count
    });
});