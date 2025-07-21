import { PeriodType } from "@prisma/client";
import { StatPeriodType, type StatsUserSearchInput, type StatsUserSearchResult, generatePK } from "@vrooli/shared";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import { UserDbFactory } from "../../__test/fixtures/db/userFixtures.js";
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { type RecursivePartial } from "../../types.js";
import { statsUser_findMany } from "../generated/statsUser_findMany.js";
import { statsUser } from "./statsUser.js";

// Helper to extract stats info from results
async function extractStattedObjectInfoFromStats(result: RecursivePartial<StatsUserSearchResult>) {
    const resultsStatIds = result.edges!.map(edge => edge!.node!.id) as string[];
    const withStattedIds = await DbProvider.get().stats_user.findMany({
        where: { id: { in: resultsStatIds } },
        select: { userId: true },
    });
    const resultStattedIds = withStattedIds.map(stat => stat.userId);
    return { resultStattedIds };
}

describe("EndpointsStatsUser", () => {
    let loggerErrorStub: any;
    let loggerInfoStub: any;

    beforeAll(async () => {
        loggerErrorStub = vi.spyOn(logger, "error").mockImplementation(() => undefined);
        loggerInfoStub = vi.spyOn(logger, "info").mockImplementation(() => undefined);
    });

    beforeEach(async () => {
        // Clean up using dependency-ordered cleanup helpers
        await cleanupGroups.minimal(DbProvider.get());
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

    afterAll(async () => {
        loggerErrorStub.mockRestore();
        loggerInfoStub.mockRestore();
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns own stats, stats for own bots, and stats for public users/bots", async () => {
                // Create users with stats within transaction
                const user1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1 - Public",
                        handle: "test-user-1",
                        isPrivate: false,
                    }),
                });

                const user2 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 2 - Private",
                        handle: "test-user-2",
                        isPrivate: true,
                    }),
                });

                const publicUser = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Public User",
                        handle: "public-user",
                        isPrivate: false,
                    }),
                });

                // Create bots
                const user1Bot = await DbProvider.get().user.create({
                    data: UserDbFactory.createBot({
                        id: generatePK(),
                        name: "User1's Bot - Private",
                        handle: "user1-bot",
                        isPrivate: true,
                        invitedByUserId: user1.id,
                    }),
                });

                const publicBot = await DbProvider.get().user.create({
                    data: UserDbFactory.createBot({
                        id: generatePK(),
                        name: "Public Bot",
                        handle: "public-bot",
                        isPrivate: false,
                        invitedByUserId: user2.id,
                    }),
                });

                // Create stats for all users
                await DbProvider.get().stats_user.createMany({
                    data: [
                        {
                            id: generatePK(),
                            userId: user1.id,
                            periodStart: new Date("2023-01-01"),
                            periodEnd: new Date("2023-01-31"),
                            periodType: PeriodType.Monthly,
                            resourcesCreatedByType: { "API": 0, "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourcesCompletedByType: { "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourceCompletionTimeAverageByType: { "CODE": 0.0, "PROJECT": 0.0, "ROUTINE": 0.0, "STANDARD": 0.0 },
                            teamsCreated: 0,
                            runsStarted: 0,
                            runsCompleted: 0,
                            runCompletionTimeAverage: 0.0,
                            runContextSwitchesAverage: 0.0,
                        },
                        {
                            id: generatePK(),
                            userId: user2.id,
                            periodStart: new Date("2023-02-01"),
                            periodEnd: new Date("2023-02-28"),
                            periodType: PeriodType.Monthly,
                            resourcesCreatedByType: { "API": 0, "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourcesCompletedByType: { "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourceCompletionTimeAverageByType: { "CODE": 0.0, "PROJECT": 0.0, "ROUTINE": 0.0, "STANDARD": 0.0 },
                            teamsCreated: 0,
                            runsStarted: 0,
                            runsCompleted: 0,
                            runCompletionTimeAverage: 0.0,
                            runContextSwitchesAverage: 0.0,
                        },
                        {
                            id: generatePK(),
                            userId: publicUser.id,
                            periodStart: new Date("2023-03-01"),
                            periodEnd: new Date("2023-03-31"),
                            periodType: PeriodType.Monthly,
                            resourcesCreatedByType: { "API": 0, "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourcesCompletedByType: { "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourceCompletionTimeAverageByType: { "CODE": 0.0, "PROJECT": 0.0, "ROUTINE": 0.0, "STANDARD": 0.0 },
                            teamsCreated: 0,
                            runsStarted: 0,
                            runsCompleted: 0,
                            runCompletionTimeAverage: 0.0,
                            runContextSwitchesAverage: 0.0,
                        },
                        {
                            id: generatePK(),
                            userId: user1Bot.id,
                            periodStart: new Date("2023-04-01"),
                            periodEnd: new Date("2023-04-30"),
                            periodType: PeriodType.Monthly,
                            resourcesCreatedByType: { "API": 0, "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourcesCompletedByType: { "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourceCompletionTimeAverageByType: { "CODE": 0.0, "PROJECT": 0.0, "ROUTINE": 0.0, "STANDARD": 0.0 },
                            teamsCreated: 0,
                            runsStarted: 0,
                            runsCompleted: 0,
                            runCompletionTimeAverage: 0.0,
                            runContextSwitchesAverage: 0.0,
                        },
                        {
                            id: generatePK(),
                            userId: publicBot.id,
                            periodStart: new Date("2023-01-01"),
                            periodEnd: new Date("2023-05-31"),
                            periodType: PeriodType.Monthly,
                            resourcesCreatedByType: { "API": 0, "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourcesCompletedByType: { "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourceCompletionTimeAverageByType: { "CODE": 0.0, "PROJECT": 0.0, "ROUTINE": 0.0, "STANDARD": 0.0 },
                            teamsCreated: 0,
                            runsStarted: 0,
                            runsCompleted: 0,
                            runCompletionTimeAverage: 0.0,
                            runContextSwitchesAverage: 0.0,
                        },
                    ],
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: user1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsUserSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const expectedStattedIds = [
                    user1.id, // Own (Public)
                    //user2.id, // Private User
                    publicUser.id, // Public User
                    user1Bot.id, // Own Bot
                    publicBot.id, // Public Bot
                ];

                const result = await statsUser.findMany({ input }, { req, res }, statsUser_findMany);

                expect(result).not.toBeNull();
                const { resultStattedIds } = await extractStattedObjectInfoFromStats(result);
                expect(resultStattedIds.sort()).toEqual(expectedStattedIds.sort());
            });

            it("properly filters by periodType", async () => {
                const user1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1",
                        isPrivate: false,
                    }),
                });

                const statsData = await DbProvider.get().stats_user.create({
                    data: {
                        id: generatePK(),
                        userId: user1.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        resourcesCreatedByType: { "API": 0, "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                        resourcesCompletedByType: { "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                        resourceCompletionTimeAverageByType: { "CODE": 0.0, "PROJECT": 0.0, "ROUTINE": 0.0, "STANDARD": 0.0 },
                        teamsCreated: 0,
                        runsStarted: 0,
                        runsCompleted: 0,
                        runCompletionTimeAverage: 0.0,
                        runContextSwitchesAverage: 0.0,
                    },
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: user1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsUserSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: {
                        after: new Date("2023-01-01"),
                        before: new Date("2023-01-31"),
                    },
                };

                const result = await statsUser.findMany({ input }, { req, res }, statsUser_findMany);
                const resultStatIds = result.edges!.map(edge => edge!.node!.id);

                expect(result).not.toBeNull();
                expect(resultStatIds).toContain(statsData.id);
            });

            it("API key with public permission returns public user/bot stats", async () => {
                const user1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1 - Public",
                        handle: "test-user-1",
                        isPrivate: false,
                    }),
                });

                const publicUser = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Public User",
                        handle: "public-user",
                        isPrivate: false,
                    }),
                });

                const publicBot = await DbProvider.get().user.create({
                    data: UserDbFactory.createBot({
                        id: generatePK(),
                        name: "Public Bot",
                        handle: "public-bot",
                        isPrivate: false,
                        invitedByUserId: user1.id,
                    }),
                });

                // Create stats
                await DbProvider.get().stats_user.createMany({
                    data: [
                        {
                            id: generatePK(),
                            userId: user1.id,
                            periodStart: new Date("2023-01-01"),
                            periodEnd: new Date("2023-01-31"),
                            periodType: PeriodType.Monthly,
                            resourcesCreatedByType: { "API": 0, "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourcesCompletedByType: { "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourceCompletionTimeAverageByType: { "CODE": 0.0, "PROJECT": 0.0, "ROUTINE": 0.0, "STANDARD": 0.0 },
                            teamsCreated: 0,
                            runsStarted: 0,
                            runsCompleted: 0,
                            runCompletionTimeAverage: 0.0,
                            runContextSwitchesAverage: 0.0,
                        },
                        {
                            id: generatePK(),
                            userId: publicUser.id,
                            periodStart: new Date("2023-01-01"),
                            periodEnd: new Date("2023-01-31"),
                            periodType: PeriodType.Monthly,
                            resourcesCreatedByType: { "API": 0, "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourcesCompletedByType: { "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourceCompletionTimeAverageByType: { "CODE": 0.0, "PROJECT": 0.0, "ROUTINE": 0.0, "STANDARD": 0.0 },
                            teamsCreated: 0,
                            runsStarted: 0,
                            runsCompleted: 0,
                            runCompletionTimeAverage: 0.0,
                            runContextSwitchesAverage: 0.0,
                        },
                        {
                            id: generatePK(),
                            userId: publicBot.id,
                            periodStart: new Date("2023-01-01"),
                            periodEnd: new Date("2023-01-31"),
                            periodType: PeriodType.Monthly,
                            resourcesCreatedByType: { "API": 0, "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourcesCompletedByType: { "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourceCompletionTimeAverageByType: { "CODE": 0.0, "PROJECT": 0.0, "ROUTINE": 0.0, "STANDARD": 0.0 },
                            teamsCreated: 0,
                            runsStarted: 0,
                            runsCompleted: 0,
                            runCompletionTimeAverage: 0.0,
                            runContextSwitchesAverage: 0.0,
                        },
                    ],
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: user1.id.toString() };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: StatsUserSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const expectedStattedIds = [
                    user1.id, // Own (Public)
                    publicUser.id, // Public User
                    publicBot.id, // Public Bot
                ];

                const result = await statsUser.findMany({ input }, { req, res }, statsUser_findMany);
                const { resultStattedIds } = await extractStattedObjectInfoFromStats(result);
                expect(resultStattedIds.sort()).toEqual(expectedStattedIds.sort());
            });

            it("not logged in returns only public user/bot stats", async () => {
                const user1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1 - Public",
                        handle: "test-user-1",
                        isPrivate: false,
                    }),
                });

                const publicUser = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Public User",
                        handle: "public-user",
                        isPrivate: false,
                    }),
                });

                const publicBot = await DbProvider.get().user.create({
                    data: UserDbFactory.createBot({
                        id: generatePK(),
                        name: "Public Bot",
                        handle: "public-bot",
                        isPrivate: false,
                        invitedByUserId: user1.id,
                    }),
                });

                // Create stats
                await DbProvider.get().stats_user.createMany({
                    data: [
                        {
                            id: generatePK(),
                            userId: user1.id,
                            periodStart: new Date("2023-01-01"),
                            periodEnd: new Date("2023-01-31"),
                            periodType: PeriodType.Monthly,
                            resourcesCreatedByType: { "API": 0, "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourcesCompletedByType: { "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourceCompletionTimeAverageByType: { "CODE": 0.0, "PROJECT": 0.0, "ROUTINE": 0.0, "STANDARD": 0.0 },
                            teamsCreated: 0,
                            runsStarted: 0,
                            runsCompleted: 0,
                            runCompletionTimeAverage: 0.0,
                            runContextSwitchesAverage: 0.0,
                        },
                        {
                            id: generatePK(),
                            userId: publicUser.id,
                            periodStart: new Date("2023-01-01"),
                            periodEnd: new Date("2023-01-31"),
                            periodType: PeriodType.Monthly,
                            resourcesCreatedByType: { "API": 0, "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourcesCompletedByType: { "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourceCompletionTimeAverageByType: { "CODE": 0.0, "PROJECT": 0.0, "ROUTINE": 0.0, "STANDARD": 0.0 },
                            teamsCreated: 0,
                            runsStarted: 0,
                            runsCompleted: 0,
                            runCompletionTimeAverage: 0.0,
                            runContextSwitchesAverage: 0.0,
                        },
                        {
                            id: generatePK(),
                            userId: publicBot.id,
                            periodStart: new Date("2023-01-01"),
                            periodEnd: new Date("2023-01-31"),
                            periodType: PeriodType.Monthly,
                            resourcesCreatedByType: { "API": 0, "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourcesCompletedByType: { "CODE": 0, "PROJECT": 0, "ROUTINE": 0, "STANDARD": 0 },
                            resourceCompletionTimeAverageByType: { "CODE": 0.0, "PROJECT": 0.0, "ROUTINE": 0.0, "STANDARD": 0.0 },
                            teamsCreated: 0,
                            runsStarted: 0,
                            runsCompleted: 0,
                            runCompletionTimeAverage: 0.0,
                            runContextSwitchesAverage: 0.0,
                        },
                    ],
                });

                const { req, res } = await mockLoggedOutSession();

                const input: StatsUserSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const expectedStattedIds = [
                    user1.id, // Public
                    publicUser.id, // Public User
                    publicBot.id, // Public Bot
                ];

                const result = await statsUser.findMany({ input }, { req, res }, statsUser_findMany);

                expect(result).not.toBeNull();
                const { resultStattedIds } = await extractStattedObjectInfoFromStats(result);
                expect(resultStattedIds.sort()).toEqual(expectedStattedIds.sort());
            });
        });

        describe("invalid", () => {
            it("invalid time range format should throw error", async () => {
                const user1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1",
                        isPrivate: false,
                    }),
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: user1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsUserSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: { after: new Date("invalid"), before: new Date("invalid") },
                };

                await expect(async () => {
                    await statsUser.findMany({ input }, { req, res }, statsUser_findMany);
                }).rejects.toThrow();
            });

            it("invalid periodType should throw error", async () => {
                const user1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1",
                        isPrivate: false,
                    }),
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: user1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = { periodType: "InvalidPeriod" as any };

                await expect(async () => {
                    await statsUser.findMany({ input: input as StatsUserSearchInput }, { req, res }, statsUser_findMany);
                }).rejects.toThrow();
            });
        });
    });
});
