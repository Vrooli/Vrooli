import { StatPeriodType, type StatsProjectSearchInput, generatePK } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { statsProject_findMany } from "../generated/statsProject_findMany.js";
import { statsProject } from "./statsProject.js";
// Import database fixtures for seeding
import { ProjectDbFactory, seedTestProjects } from "../../__test/fixtures/db/projectFixtures.js";
import { seedTestStatsProject } from "../../__test/fixtures/db/statsProjectFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";

describe("EndpointsStatsProject", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warning").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Clean up tables used in tests
        const prisma = DbProvider.get();
        await prisma.stats_project.deleteMany();
        await prisma.projectVersion.deleteMany();
        await prisma.project.deleteMany();
        await prisma.user.deleteMany();
        // Clear Redis cache
        await CacheService.get().flushAll();
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    // Helper function to create test data
    const createTestData = async () => {
        // Create test users
        const testUsers = await seedTestUsers(DbProvider.get(), 3, { withAuth: true });
        
        // Create test projects with different privacy settings
        const publicProjects = await seedTestProjects(DbProvider.get(), {
            createdById: testUsers[0].id,
            count: 2,
            isPrivate: false,
            withVersions: true,
        });
        
        const privateProjects = await Promise.all([
            // Private project owned by user 1
            DbProvider.get().project.create({
                data: ProjectDbFactory.createWithVersion({
                    id: generatePK(),
                    createdById: testUsers[0].id,
                    isPrivate: true,
                }),
                include: { versions: true },
            }),
            // Private project owned by user 2
            DbProvider.get().project.create({
                data: ProjectDbFactory.createWithVersion({
                    id: generatePK(),
                    createdById: testUsers[1].id,
                    isPrivate: true,
                }),
                include: { versions: true },
            }),
        ]);
        
        // Create stats for all projects
        const statsData = [];
        
        // Public project stats
        for (const project of publicProjects) {
            const stats = await seedTestStatsProject(DbProvider.get(), {
                projectId: project.id,
                periodStart: new Date("2023-01-01"),
                periodEnd: new Date("2023-01-31"),
                count: 1,
            });
            statsData.push(...stats);
        }
        
        // Private project stats
        for (const project of privateProjects) {
            const stats = await seedTestStatsProject(DbProvider.get(), {
                projectId: project.id,
                periodStart: new Date("2023-03-01"),
                periodEnd: new Date("2023-03-31"),
                count: 1,
            });
            statsData.push(...stats);
        }
        
        return { 
            testUsers, 
            publicProjects, 
            privateProjects, 
            statsData 
        };
    };

    describe("findMany", () => {
        describe("valid", () => {
            it("returns stats for public and owned projects when logged in", async () => {
                const { testUsers, publicProjects, privateProjects, statsData } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // User 1 owns first private project
                });

                const input: StatsProjectSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsProject.findMany({ input }, { req, res }, statsProject_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                
                const resultProjectIds = result.edges!.map(edge => 
                    edge?.node?.project?.id
                ).filter(Boolean);

                // User 1 should see public projects and their own private project
                expect(resultProjectIds).toContain(publicProjects[0].id);
                expect(resultProjectIds).toContain(publicProjects[1].id);
                expect(resultProjectIds).toContain(privateProjects[0].id); // User 1's private project
                // User 1 should NOT see user 2's private project stats
                expect(resultProjectIds).not.toContain(privateProjects[1].id);
            });

            it("filters by periodType", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input: StatsProjectSearchInput = { 
                    periodType: StatPeriodType.Monthly 
                };
                const result = await statsProject.findMany({ input }, { req, res }, statsProject_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                
                // All returned stats should be monthly
                result.edges!.forEach(edge => {
                    expect(edge?.node?.periodType).toBe("Monthly");
                });
            });

            it("filters by time range", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

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
                
                // All returned stats should be within the specified time range
                result.edges!.forEach(edge => {
                    const periodStart = new Date(edge?.node?.periodStart);
                    expect(periodStart.getTime()).toBeGreaterThanOrEqual(new Date("2023-01-01").getTime());
                    expect(periodStart.getTime()).toBeLessThanOrEqual(new Date("2023-01-31").getTime());
                });
            });

            it("API key with public permissions returns only public projects", async () => {
                const { testUsers, publicProjects, privateProjects } = await createTestData();
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input: StatsProjectSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsProject.findMany({ input }, { req, res }, statsProject_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                
                const resultProjectIds = result.edges!.map(edge => 
                    edge?.node?.project?.id
                ).filter(Boolean);

                // Should only see public projects
                expect(resultProjectIds).toContain(publicProjects[0].id);
                expect(resultProjectIds).toContain(publicProjects[1].id);
                expect(resultProjectIds).not.toContain(privateProjects[0].id);
                expect(resultProjectIds).not.toContain(privateProjects[1].id);
            });

            it("not logged in returns only public projects", async () => {
                const { publicProjects, privateProjects } = await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: StatsProjectSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsProject.findMany({ input }, { req, res }, statsProject_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                
                const resultProjectIds = result.edges!.map(edge => 
                    edge?.node?.project?.id
                ).filter(Boolean);

                // Should only see public projects
                expect(resultProjectIds).toContain(publicProjects[0].id);
                expect(resultProjectIds).toContain(publicProjects[1].id);
                expect(resultProjectIds).not.toContain(privateProjects[0].id);
                expect(resultProjectIds).not.toContain(privateProjects[1].id);
            });
        });

        describe("invalid", () => {
            it("invalid time range format should throw error", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input: StatsProjectSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: { 
                        after: new Date("invalid"), 
                        before: new Date("invalid") 
                    },
                };

                await expect(async () => {
                    await statsProject.findMany({ input }, { req, res }, statsProject_findMany);
                }).rejects.toThrow();
            });

            it("invalid periodType should throw error", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input = { 
                    periodType: "InvalidPeriod" as any 
                };

                await expect(async () => {
                    await statsProject.findMany({ 
                        input: input as StatsProjectSearchInput 
                    }, { req, res }, statsProject_findMany);
                }).rejects.toThrow();
            });

            it("cannot see stats of private project you don't own", async () => {
                const { testUsers, privateProjects } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // User 1
                });

                // Try to access User 2's private project stats
                const input: StatsProjectSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    projectId: privateProjects[1].id, // User 2's private project
                };
                const result = await statsProject.findMany({ input }, { req, res }, statsProject_findMany);

                expect(result).not.toBeNull();
                expect(result.edges!.length).toBe(0);
            });
        });
    });
});