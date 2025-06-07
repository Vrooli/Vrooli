import { AwardCategory, type AwardSearchInput } from "@vrooli/shared";
import { describe, it, expect, beforeAll, afterAll, beforeEach, vi } from "vitest";
import { assertFindManyResultIds } from "../../__test/helpers.js";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { award_findMany } from "../generated/award_findMany.js";
import { award } from "./award.js";

// Import database fixtures for seeding
import { AwardDbFactory, seedAwards } from "../../__test/fixtures/awardFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/userFixtures.js";

describe("EndpointsAward", () => {
    let testUsers: any[];
    let userAward1: any;
    let userAward2: any;

    beforeAll(() => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Reset Redis and database tables
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Seed test users using database fixtures
        testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

        // Seed awards using database fixtures
        const awards1 = await seedAwards(DbProvider.get(), {
            userId: testUsers[0].id,
            categories: [
                { name: AwardCategory.RoutineCreate, progress: 75 },
            ],
        });
        userAward1 = awards1[0];

        const awards2 = await seedAwards(DbProvider.get(), {
            userId: testUsers[1].id,
            categories: [
                { name: AwardCategory.ProjectCreate, progress: 25 },
            ],
        });
        userAward2 = awards2[0];

        // Update created/updated times for time filtering tests
        await DbProvider.get().award.update({
            where: { id: userAward1.id },
            data: {
                createdAt: new Date("2023-03-01"),
                updatedAt: new Date("2023-03-01"),
            },
        });
        await DbProvider.get().award.update({
            where: { id: userAward2.id },
            data: {
                createdAt: new Date("2023-03-15"),
                updatedAt: new Date("2023-03-15"),
            },
        });
    });

    afterAll(async () => {
        // Clean up
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns awards without filters", async () => {
                const { req, res } = await mockAuthenticatedSession({ 
                    ...loggedInUserNoPremiumData(), 
                    id: testUsers[0].id 
                });

                // When logged in as user1, should see user1's awards
                const expectedIds = [
                    userAward1.id,   // User1's award
                ];

                const input: AwardSearchInput = { take: 10 };
                const result = await award.findMany({ input }, { req, res }, award_findMany);
                assertFindManyResultIds(expect, result, expectedIds);
            });

            it("filters by updated time frame", async () => {
                const { req, res } = await mockAuthenticatedSession({ 
                    ...loggedInUserNoPremiumData(), 
                    id: testUsers[0].id 
                });

                // For the given time range, should only see awards updated in Feb-Mar that user1 has access to
                const expectedIds = [
                    userAward1.id,   // Updated in March and belongs to user1
                ];

                const input: AwardSearchInput = {
                    updatedTimeFrame: {
                        after: new Date("2023-02-01"),
                        before: new Date("2023-04-01"),
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
                    id: testUsers[0].id 
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
                    id: testUsers[0].id 
                });

                const input: AwardSearchInput = {
                    updatedTimeFrame: {
                        // Invalid date objects that will cause errors
                        after: new Date("invalid-date"),
                        before: new Date("invalid-date"),
                    },
                };

                await expect(async () => {
                    await award.findMany({ input }, { req, res }, award_findMany);
                }).rejects.toThrow();
            });
        });
    });
}); 
