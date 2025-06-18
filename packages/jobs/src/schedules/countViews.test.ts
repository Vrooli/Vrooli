import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { generatePK, generatePublicId } from "@vrooli/shared";
import { DbProvider } from "@vrooli/server";
import { countViews } from "./countViews.js";

// Remove the batch mock - use the real implementation from testcontainers

vi.mock("@vrooli/server", async () => {
    const actual = await vi.importActual("@vrooli/server");
    return {
        ...actual,
        logger: {
            info: vi.fn(),
            warning: vi.fn(),
            error: vi.fn(),
        }
    };
});

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
        const uniqueId = Date.now();
        const viewer1 = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Viewer 1",
                handle: `viewer1_${uniqueId}`,
            },
        });
        testUserIds.push(viewer1.id);

        const viewer2 = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Viewer 2",
                handle: `viewer2_${uniqueId}`,
            },
        });
        testUserIds.push(viewer2.id);

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner",
                handle: `owner1_${uniqueId}`,
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
        await countViews();

        // Check that the view count was updated correctly
        const updatedIssue = await DbProvider.get().issue.findUnique({
            where: { id: issue.id },
            select: { views: true },
        });

        expect(updatedIssue?.views).toBe(2); // Should match actual view count
    });

    it("should not update when counts already match", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner 2",
                handle: "owner2",
            },
        });
        testUserIds.push(owner.id);

        // Create test resource with correct view count
        const resource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                resourceType: "RoutineVersion",
                views: 0, // Correct count (no views)
            },
        });
        testResourceIds.push(resource.id);

        // Get the original updatedAt timestamp
        const originalResource = await DbProvider.get().resource.findUnique({
            where: { id: resource.id },
            select: { updatedAt: true, views: true },
        });

        // Run the count views function
        await countViews();

        // Check that the resource was not updated
        const unchangedResource = await DbProvider.get().resource.findUnique({
            where: { id: resource.id },
            select: { updatedAt: true, views: true },
        });

        expect(unchangedResource?.updatedAt).toEqual(originalResource?.updatedAt);
        expect(unchangedResource?.views).toBe(0);
    });

    it("should handle multiple entity types", async () => {
        const viewer = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Viewer 3",
                handle: "viewer3",
            },
        });
        testUserIds.push(viewer.id);

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner 3",
                handle: "owner3",
                views: 5, // Incorrect count
            },
        });
        testUserIds.push(owner.id);

        const team = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                handle: "testteam1",
                views: 3, // Incorrect count
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

        // Create views for user
        const userViewId = generatePK();
        testViewIds.push(userViewId);
        await DbProvider.get().view.create({
            data: {
                id: userViewId,
                name: "Test User View",
                byId: viewer.id,
                userId: owner.id,
            },
        });

        // Create views for team
        const teamView1Id = generatePK();
        const teamView2Id = generatePK();
        testViewIds.push(teamView1Id, teamView2Id);
        await DbProvider.get().view.createMany({
            data: [
                {
                    id: teamView1Id,
                    name: "Test Team View 1",
                    byId: viewer.id,
                    teamId: team.id,
                },
                {
                    id: teamView2Id,
                    name: "Test Team View 2",
                    byId: owner.id,
                    teamId: team.id,
                },
            ],
        });

        // Run the count views function
        await countViews();

        // Check that both counts were updated
        const updatedUser = await DbProvider.get().user.findUnique({
            where: { id: owner.id },
            select: { views: true },
        });

        const updatedTeam = await DbProvider.get().team.findUnique({
            where: { id: team.id },
            select: { views: true },
        });

        expect(updatedUser?.views).toBe(1); // 1 view
        expect(updatedTeam?.views).toBe(2); // 2 views
    });

    it("should handle entities with no views", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner 4",
                handle: "owner4",
            },
        });
        testUserIds.push(owner.id);

        // Create entities with non-zero view counts but no actual views
        const issue = await DbProvider.get().issue.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                views: 15, // Should be corrected to 0
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "Issue with no views",
                        description: "Should be reset to 0",
                    }],
                },
            },
        });
        testIssueIds.push(issue.id);

        // Run the count views function
        await countViews();

        // Check that the count was corrected to 0
        const correctedIssue = await DbProvider.get().issue.findUnique({
            where: { id: issue.id },
            select: { views: true },
        });

        expect(correctedIssue?.views).toBe(0);
    });

    it("should process multiple entities in batches", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner 5",
                handle: "owner5",
            },
        });
        testUserIds.push(owner.id);

        // Create multiple resources
        const resourcePromises = [];
        for (let i = 0; i < 5; i++) {
            resourcePromises.push(
                DbProvider.get().resource.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        createdById: owner.id,
                        resourceType: "RoutineVersion",
                        views: i * 10, // Incorrect counts
                    },
                }),
            );
        }
        const resources = await Promise.all(resourcePromises);
        // Sort resources by id to ensure consistent ordering
        resources.sort((a, b) => {
            if (a.id < b.id) return -1;
            if (a.id > b.id) return 1;
            return 0;
        });
        resources.forEach(r => testResourceIds.push(r.id));

        // Create some views
        const view1Id = generatePK();
        const view2Id = generatePK();
        const view3Id = generatePK();
        testViewIds.push(view1Id, view2Id, view3Id);
        await DbProvider.get().view.createMany({
            data: [
                {
                    id: view1Id,
                    name: "Test Resource View 1",
                    byId: owner.id,
                    resourceId: resources[0].id,
                },
                {
                    id: view2Id,
                    name: "Test Resource View 2",
                    byId: owner.id,
                    resourceId: resources[0].id,
                },
                {
                    id: view3Id,
                    name: "Test Resource View 3",
                    byId: owner.id,
                    resourceId: resources[2].id,
                },
            ],
        });

        // Check the counts before running countViews
        const beforeResources = await DbProvider.get().resource.findMany({
            where: { id: { in: resources.map(r => r.id) } },
            select: { 
                id: true, 
                views: true,
                _count: {
                    select: { viewedBy: true }
                }
            },
            orderBy: { id: "asc" },
        });
        
        console.log("=== Before countViews ===");
        beforeResources.forEach((r, i) => {
            console.log(`Resource[${i}] id=${r.id}: views=${r.views}, actual viewedBy count=${r._count.viewedBy}`);
        });

        // Run the count views function
        await countViews();

        // Check that all counts were updated correctly
        const updatedResources = await DbProvider.get().resource.findMany({
            where: { id: { in: resources.map(r => r.id) } },
            select: { 
                id: true, 
                views: true,
                _count: {
                    select: { viewedBy: true }
                }
            },
            orderBy: { id: "asc" },
        });
        
        console.log("\n=== After countViews ===");
        updatedResources.forEach((r, i) => {
            console.log(`Resource[${i}] id=${r.id}: views=${r.views}, actual viewedBy count=${r._count.viewedBy}`);
        });

        expect(updatedResources[0].views).toBe(2); // Has 2 views
        expect(updatedResources[1].views).toBe(0); // No views
        expect(updatedResources[2].views).toBe(1); // Has 1 view
        expect(updatedResources[3].views).toBe(0); // No views
        expect(updatedResources[4].views).toBe(0); // No views
    });
});
