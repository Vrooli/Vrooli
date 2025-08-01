import { AwardCategory, type Award, type AwardSearchInput } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { seedAwards } from "../../__test/fixtures/db/awardFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
import { assertFindManyResultIds } from "../../__test/helpers.js";
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { award_findMany } from "../generated/award_findMany.js";
import { award } from "./award.js";

// AI_CHECK: TYPE_SAFETY=phase1-test-4 | LAST: 2025-07-04 - Replaced any[] with proper User[] and Award types

describe("EndpointsAward", () => {
    let testUsers: any[]; // Database users have bigint IDs, not string IDs like the shared User type
    let userAward1: Award;
    let userAward2: Award;

    beforeAll(() => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    afterEach(async () => {
        // Validate cleanup to detect any missed records
        const orphans = await validateCleanup(DbProvider.get(), {
            tables: ["user", "user_auth", "email", "session"],
            logOrphans: true,
        });
        if (orphans.length > 0) {
            console.warn("Test cleanup incomplete:", orphans);
        }
    });

    beforeEach(async () => {
        // Clean up using dependency-ordered cleanup helpers
        await cleanupGroups.minimal(DbProvider.get());

        // Create test users
        const seedResult = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });
        testUsers = seedResult.records;

        // Seed awards using database fixtures
        const awards1 = await seedAwards(DbProvider.get(), {
            userId: testUsers[0].id.toString(),
            categories: [
                { name: AwardCategory.RoutineCreate, progress: 75 },
            ],
        });
        userAward1 = awards1[0];

        const awards2 = await seedAwards(DbProvider.get(), {
            userId: testUsers[1].id.toString(),
            categories: [
                { name: AwardCategory.ProjectCreate, progress: 25 },
            ],
        });
        userAward2 = awards2[0];

        // Update created/updated times for time filtering tests
        await DbProvider.get().award.update({
            where: { id: BigInt(userAward1.id) },
            data: {
                createdAt: new Date("2023-03-01"),
                updatedAt: new Date("2023-03-01"),
            },
        });
        await DbProvider.get().award.update({
            where: { id: BigInt(userAward2.id) },
            data: {
                createdAt: new Date("2023-03-15"),
                updatedAt: new Date("2023-03-15"),
            },
        });
    });

    afterAll(async () => {
        // Clean up
        await CacheService.get().flushAll();
        await DbProvider.deleteAll();

        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns awards without filters", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id.toString(),
                });

                // When logged in as user1, should see user1's awards
                const expectedIds = [
                    BigInt(userAward1.id),   // User1's award
                ];

                const input: AwardSearchInput = { take: 10 };
                const result = await award.findMany({ input }, { req, res }, award_findMany);
                assertFindManyResultIds(expect, result, expectedIds);
            });

            it("filters by updated time frame", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id.toString(),
                });

                // For the given time range, should only see awards updated in Feb-Mar that user1 has access to
                const expectedIds = [
                    BigInt(userAward1.id),   // Updated in March and belongs to user1
                ];

                const input: AwardSearchInput = {
                    updatedTimeFrame: {
                        after: new Date("2023-02-01").toISOString(),
                        before: new Date("2023-04-01").toISOString(),
                    },
                };
                const result = await award.findMany({ input }, { req, res }, award_findMany);

                // Use the helper function for assertion
                assertFindManyResultIds(expect, result, expectedIds);
            });

            it("API key - public permissions", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input: AwardSearchInput = {
                    take: 10,
                };

                await expect(async () => {
                    await award.findMany({ input }, { req, res }, award_findMany);
                }).rejects.toThrow(); // Awards are private
            });

            it("not logged in", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: AwardSearchInput = {
                    take: 10,
                };

                await expect(async () => {
                    await award.findMany({ input }, { req, res }, award_findMany);
                }).rejects.toThrow(); // Awards are private
            });
        });

        describe("invalid", () => {
            it("invalid time range format", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input: AwardSearchInput = {
                    updatedTimeFrame: {
                        // Invalid date objects - NaN dates should not cause errors,
                        // they just won't match any records
                        after: new Date("invalid-date").toISOString(),
                        before: new Date("invalid-date").toISOString(),
                    },
                };

                // This shouldn't throw an error, it should just return empty results
                const result = await award.findMany({ input }, { req, res }, award_findMany);
                expect(result).toBeDefined();
                expect(result.edges).toEqual([]);
            });
        });
    });
}); 
