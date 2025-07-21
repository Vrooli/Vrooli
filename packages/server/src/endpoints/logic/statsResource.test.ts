import { StatPeriodType, type StatsResourceSearchInput, generatePK } from "@vrooli/shared";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { statsResource_findMany } from "../generated/statsResource_findMany.js";
import { statsResource } from "./statsResource.js";
// Import database fixtures for seeding
import { ResourceDbFactory, seedTestResources } from "../../__test/fixtures/db/resourceFixtures.js";
import { seedTestStatsResource } from "../../__test/fixtures/db/statsResourceFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";

describe("EndpointsStatsResource", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warning").mockImplementation(() => logger);
    });

    afterEach(async () => {
        // Validate cleanup to detect any missed records
        const orphans = await validateCleanup(DbProvider.get(), {
            tables: ["run", "run_step", "run_io", "resource", "resource_version", "user"],
            logOrphans: true,
        });
        if (orphans.length > 0) {
            console.warn("Test cleanup incomplete:", orphans);
        }
    });

    beforeEach(async () => {
        // Clean up using dependency-ordered cleanup helpers
        await cleanupGroups.execution(DbProvider.get());
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    // Helper function to create test data
    async function createTestData() {
        // Create test users
        const testUsers = await seedTestUsers(DbProvider.get(), 3, { withAuth: true });

        // Create test resources with different privacy settings
        const publicResources = await seedTestResources(DbProvider.get(), {
            createdById: testUsers[0].id,
            count: 2,
            isPrivate: false,
            isInternal: false,
            withVersions: true,
        });

        const privateResources = await Promise.all([
            // Private resource owned by user 1
            DbProvider.get().resource.create({
                data: ResourceDbFactory.createWithVersion({
                    id: generatePK(),
                    createdById: testUsers[0].id,
                    isPrivate: true,
                    isInternal: false,
                }),
                include: { versions: true },
            }),
            // Private resource owned by user 2
            DbProvider.get().resource.create({
                data: ResourceDbFactory.createWithVersion({
                    id: generatePK(),
                    createdById: testUsers[1].id,
                    isPrivate: true,
                    isInternal: false,
                }),
                include: { versions: true },
            }),
        ]);

        // Create internal resource (should be filtered out)
        const internalResource = await DbProvider.get().resource.create({
            data: ResourceDbFactory.createWithVersion({
                id: generatePK(),
                createdById: testUsers[0].id,
                isPrivate: false,
                isInternal: true, // Internal resources should be filtered out
            }),
            include: { versions: true },
        });

        // Create stats for all resources
        const statsData = [];

        // Public resource stats
        for (const resource of publicResources) {
            const stats = await seedTestStatsResource(DbProvider.get(), {
                resourceId: resource.id,
                periodStart: new Date("2023-01-01"),
                periodEnd: new Date("2023-01-31"),
                count: 1,
            });
            statsData.push(...stats);
        }

        // Private resource stats
        for (const resource of privateResources) {
            const stats = await seedTestStatsResource(DbProvider.get(), {
                resourceId: resource.id,
                periodStart: new Date("2023-03-01"),
                periodEnd: new Date("2023-03-31"),
                count: 1,
            });
            statsData.push(...stats);
        }

        // Internal resource stats (should be filtered out)
        const internalStats = await seedTestStatsResource(DbProvider.get(), {
            resourceId: internalResource.id,
            periodStart: new Date("2023-01-01"),
            periodEnd: new Date("2023-01-31"),
            count: 1,
        });

        return {
            testUsers,
            publicResources,
            privateResources,
            internalResource,
            statsData,
            internalStats,
        };
    }

    describe("findMany", () => {
        describe("valid", () => {
            it("returns stats for public and owned resources when logged in", async () => {
                const { testUsers, publicResources, privateResources } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // User 1 owns first private resource
                });

                const input: StatsResourceSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsResource.findMany({ input }, { req, res }, statsResource_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);

                const resultResourceIds = result.edges!.map(edge =>
                    edge?.node?.resource?.id,
                ).filter(Boolean);

                // User 1 should see public resources and their own private resource
                expect(resultResourceIds).toContain(publicResources[0].id);
                expect(resultResourceIds).toContain(publicResources[1].id);
                expect(resultResourceIds).toContain(privateResources[0].id); // User 1's private resource
                // User 1 should NOT see user 2's private resource stats
                expect(resultResourceIds).not.toContain(privateResources[1].id);
            });

            it("filters by periodType", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input: StatsResourceSearchInput = {
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsResource.findMany({ input }, { req, res }, statsResource_findMany);

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

                const input: StatsResourceSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: {
                        after: new Date("2023-01-01"),
                        before: new Date("2023-01-31"),
                    },
                };
                const result = await statsResource.findMany({ input }, { req, res }, statsResource_findMany);

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

            it("API key with public permissions returns only public resources", async () => {
                const { testUsers, publicResources, privateResources } = await createTestData();
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input: StatsResourceSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsResource.findMany({ input }, { req, res }, statsResource_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);

                const resultResourceIds = result.edges!.map(edge =>
                    edge?.node?.resource?.id,
                ).filter(Boolean);

                // Should only see public resources
                expect(resultResourceIds).toContain(publicResources[0].id);
                expect(resultResourceIds).toContain(publicResources[1].id);
                expect(resultResourceIds).not.toContain(privateResources[0].id);
                expect(resultResourceIds).not.toContain(privateResources[1].id);
            });

            it("filters out internal resources (isInternal: true)", async () => {
                const { testUsers, publicResources, internalResource } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input: StatsResourceSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsResource.findMany({ input }, { req, res }, statsResource_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);

                const resultResourceIds = result.edges!.map(edge =>
                    edge?.node?.resource?.id,
                ).filter(Boolean);

                // Should include public resources
                expect(resultResourceIds).toContain(publicResources[0].id);
                expect(resultResourceIds).toContain(publicResources[1].id);
                // Should NOT include internal resource
                expect(resultResourceIds).not.toContain(internalResource.id);
            });

            it("not logged in returns only public resources", async () => {
                const { publicResources, privateResources } = await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: StatsResourceSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsResource.findMany({ input }, { req, res }, statsResource_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);

                const resultResourceIds = result.edges!.map(edge =>
                    edge?.node?.resource?.id,
                ).filter(Boolean);

                // Should only see public resources
                expect(resultResourceIds).toContain(publicResources[0].id);
                expect(resultResourceIds).toContain(publicResources[1].id);
                expect(resultResourceIds).not.toContain(privateResources[0].id);
                expect(resultResourceIds).not.toContain(privateResources[1].id);
            });
        });

        describe("invalid", () => {
            it("invalid time range format should throw error", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input: StatsResourceSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: {
                        after: new Date("invalid-date"),
                        before: new Date("invalid-date"),
                    },
                };

                await expect(async () => {
                    await statsResource.findMany({ input }, { req, res }, statsResource_findMany);
                }).rejects.toThrow();
            });

            it("invalid periodType should throw error", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input = {
                    periodType: "InvalidPeriodType" as any,
                };

                await expect(async () => {
                    await statsResource.findMany({
                        input: input as StatsResourceSearchInput,
                    }, { req, res }, statsResource_findMany);
                }).rejects.toThrow();
            });

            it("cannot see stats of private resource you don't own", async () => {
                const { testUsers, privateResources } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // User 1
                });

                // Try to access User 2's private resource stats
                const input: StatsResourceSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    resourceId: privateResources[1].id, // User 2's private resource
                };
                const result = await statsResource.findMany({ input }, { req, res }, statsResource_findMany);

                expect(result).not.toBeNull();
                expect(result.edges!.length).toBe(0);
            });
        });
    });
});
