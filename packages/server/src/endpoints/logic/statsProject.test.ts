import { StatPeriodType, type StatsProjectSearchInput } from "@vrooli/shared";
import { PeriodType, type project as ProjectModelPrisma } from "@prisma/client"; // Correct import
import { describe, it, expect, beforeAll, afterAll, beforeEach, vi } from "vitest";
import { defaultPublicUserData, loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { statsProject_findMany } from "../generated/statsProject_findMany.js"; // Assuming this generated type exists
import { statsProject } from "./statsProject.js";

// Test data
const testProjectId1 = "2001";
const testProjectId2 = "2002";
const privateProjectId1 = "2003"; // Private Project owned by user1
const privateProjectId2 = "2004"; // Private Project owned by user2

// User IDs for ownership testing
const user1Id = "1001";
const user2Id = "1002";

// Sample Project data structure (adjust fields as necessary based on actual Project model)
// Projects might have team ownership or different privacy fields
const projectData1: Partial<ProjectModelPrisma> & { id: string } = {
    id: testProjectId1,
    isPrivate: false,
    ownedByUserId: null, // Placeholder: Assuming direct user ownership is possible and null means public
    // ownedByTeamId: null,
    // Add other required Project fields (e.g., name, versions)
};

const projectData2: Partial<ProjectModelPrisma> & { id: string } = {
    id: testProjectId2,
    isPrivate: false,
    ownedByUserId: null,
    // Add other required Project fields
};

const privateProjectData1: Partial<ProjectModelPrisma> & { id: string } = {
    id: privateProjectId1,
    isPrivate: true,
    ownedByUserId: user1Id, // Placeholder: Assuming owned by user1
    // Add other required Project fields
};

const privateProjectData2: Partial<ProjectModelPrisma> & { id: string } = {
    id: privateProjectId2,
    isPrivate: true,
    ownedByUserId: user2Id, // Placeholder: Assuming owned by user2
    // Add other required Project fields
};

const statsProjectData1 = {
    id: `stats-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`,
    projectId: testProjectId1,
    periodStart: new Date("2023-01-01"),
    periodEnd: new Date("2023-01-31"),
    periodType: PeriodType.Monthly,
    directories: 0,
    apis: 0,
    codes: 0,
    notes: 0,
    routines: 0,
    standards: 0,
    runCompletionTimeAverage: 0.0,
    projects: 0,
    runsStarted: 0,
    runsCompleted: 0,
    runContextSwitchesAverage: 0.0,
    teams: 0,
};

const statsProjectData2 = {
    id: `stats-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`,
    projectId: testProjectId2,
    periodStart: new Date("2023-02-01"),
    periodEnd: new Date("2023-02-28"),
    periodType: PeriodType.Monthly,
    directories: 0,
    apis: 0,
    codes: 0,
    notes: 0,
    routines: 0,
    standards: 0,
    runCompletionTimeAverage: 0.0,
    projects: 0,
    runsStarted: 0,
    runsCompleted: 0,
    runContextSwitchesAverage: 0.0,
    teams: 0,
};

const privateProjectStats1 = {
    id: `stats-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`,
    projectId: privateProjectId1,
    periodStart: new Date("2023-03-01"),
    periodEnd: new Date("2023-03-31"),
    periodType: PeriodType.Monthly,
    directories: 0,
    apis: 0,
    codes: 0,
    notes: 0,
    routines: 0,
    standards: 0,
    runCompletionTimeAverage: 0.0,
    projects: 0,
    runsStarted: 0,
    runsCompleted: 0,
    runContextSwitchesAverage: 0.0,
    teams: 0,
};

const privateProjectStats2 = {
    id: `stats-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`,
    projectId: privateProjectId2,
    periodStart: new Date("2023-03-01"),
    periodEnd: new Date("2023-03-31"),
    periodType: PeriodType.Monthly,
    directories: 0,
    apis: 0,
    codes: 0,
    notes: 0,
    routines: 0,
    standards: 0,
    runCompletionTimeAverage: 0.0,
    projects: 0,
    runsStarted: 0,
    runsCompleted: 0,
    runContextSwitchesAverage: 0.0,
    teams: 0,
};

describe("EndpointsStatsProject", () => {
    let loggerErrorStub: any;
    let loggerInfoStub: any;

    beforeAll(() => {
        loggerErrorStub = vi.spyOn(logger, "error").mockImplementation(() => undefined);
        loggerInfoStub = vi.spyOn(logger, "info").mockImplementation(() => undefined);
    });

    beforeEach(async () => {
        await CacheService.get().flushAll();
        await DbProvider.deleteAll();

        // Create test users individually
        await DbProvider.get().user.create({
            data: {
                ...defaultPublicUserData(),
                id: user1Id,
                name: "Test User 1",
            },
        });
        await DbProvider.get().user.create({
            data: {
                ...defaultPublicUserData(),
                id: user2Id,
                name: "Test User 2",
            },
        });

        // Create test projects (ensure all required fields are present)
        // Placeholder: Assuming projects need a name, permissions and potentially versions
        await DbProvider.get().project.createMany({
            data: [
                { ...projectData1, permissions: JSON.stringify({}) /* versions: { create: [...] } */ },
                { ...projectData2, permissions: JSON.stringify({}) /* versions: { create: [...] } */ },
                { ...privateProjectData1, permissions: JSON.stringify({}) /* versions: { create: [...] } */ },
                { ...privateProjectData2, permissions: JSON.stringify({}) /* versions: { create: [...] } */ },
            ].map(p => ({ // Adjust ownership fields based on actual model
                ...p,
                ownedByUserId: p.ownedByUserId ?? undefined,
                // ownedByTeamId: p.ownedByTeamId ?? undefined,
            })),
        });

        // Create fresh test stats data
        await DbProvider.get().stats_project.createMany({
            data: [
                statsProjectData1,
                statsProjectData2,
                privateProjectStats1,
                privateProjectStats2,
            ],
        });
    });

    afterAll(async () => {
        await CacheService.get().flushAll();
        await DbProvider.deleteAll();

        loggerErrorStub.mockRestore();
        loggerInfoStub.mockRestore();
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns stats for public and owned projects when logged in", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id }; // User 1 owns privateProject1
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsProjectSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsProject.findMany({ input }, { req, res }, statsProject_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                // User 1 should see public projects and their own private project
                expect(resultIds).toContain(statsProjectData1.id);
                expect(resultIds).toContain(statsProjectData2.id);
                expect(resultIds).toContain(privateProjectStats1.id);
                // User 1 should NOT see user 2's private project stats
                expect(resultIds).not.toContain(privateProjectStats2.id);
            });

            it("filters by periodType", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsProjectSearchInput = { periodType: StatPeriodType.Monthly };
                const result = await statsProject.findMany({ input }, { req, res }, statsProject_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                const resultIds = result.edges!.map(edge => edge?.node?.id);
                expect(resultIds).toContain(statsProjectData1.id);
                expect(resultIds).toContain(statsProjectData2.id);
                expect(resultIds).toContain(privateProjectStats1.id);
            });

            it("filters by time range", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsProjectSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: {
                        after: new Date("2023-01-01"),
                        before: new Date("2023-01-31"),
                    },
                };
                const result = await statsProject.findMany({ input }, { req, res }, statsProject_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                const resultIds = result.edges!.map(edge => edge?.node?.id);
                expect(resultIds).toContain(statsProjectData1.id); // Should include Jan stats
                expect(resultIds).not.toContain(statsProjectData2.id); // Should exclude Feb stats
                expect(resultIds).not.toContain(privateProjectStats1.id); // Should exclude Mar stats
            });

            it("API key - public permissions returns only public projects", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: StatsProjectSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsProject.findMany({ input }, { req, res }, statsProject_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                expect(resultIds).toContain(statsProjectData1.id);
                expect(resultIds).toContain(statsProjectData2.id);
                expect(resultIds).not.toContain(privateProjectStats1.id);
                expect(resultIds).not.toContain(privateProjectStats2.id);
            });

            it("not logged in returns only public projects", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: StatsProjectSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                // Assuming readManyHelper allows public access for projects when not logged in
                const result = await statsProject.findMany({ input }, { req, res }, statsProject_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                expect(resultIds).toContain(statsProjectData1.id);
                expect(resultIds).toContain(statsProjectData2.id);
                expect(resultIds).not.toContain(privateProjectStats1.id);
                expect(resultIds).not.toContain(privateProjectStats2.id);
            });
        });

        describe("invalid", () => {
            it("invalid time range format should throw error", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsProjectSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: { after: new Date("invalid"), before: new Date("invalid") },
                };

                try {
                    await statsProject.findMany({ input }, { req, res }, statsProject_findMany);
                    expect.fail("Expected an error");
                } catch (error) {
                    expect(error).toBeInstanceOf(Error);
                }
            });

            it("invalid periodType should throw error", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = { periodType: "InvalidPeriod" as any };

                try {
                    await statsProject.findMany({ input: input as StatsProjectSearchInput }, { req, res }, statsProject_findMany);
                    expect.fail("Expected an error");
                } catch (error) {
                    expect(error).toBeInstanceOf(Error);
                }
            });

            it("cannot see stats of private project you don't own when searching by name", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id }; // User 1
                const { req, res } = await mockAuthenticatedSession(testUser);

                // Search for User 2's private project
                const input: StatsProjectSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    searchString: "Private Project 2",
                };
                const result = await statsProject.findMany({ input }, { req, res }, statsProject_findMany);

                expect(result).not.toBeNull();
                expect(result.edges!.length).toEqual(0);
                expect(result.edges!.every(edge => edge?.node?.id !== privateProjectStats2.id)).to.be.true;
            });
        });
    });
}); 
