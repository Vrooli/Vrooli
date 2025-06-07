import { StatPeriodType, type StatsUserSearchInput, type StatsUserSearchResult, uuid } from "@vrooli/shared";
import { PeriodType } from "@prisma/client";
import { describe, it, expect, beforeAll, afterAll, beforeEach, vi } from "vitest";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { type RecursivePartial } from "../../types.ts";
import { statsUser_findMany } from "../generated/statsUser_findMany.js"; // Assuming this generated type exists
import { statsUser } from "./statsUser.js";

// User IDs
const user1Id = uuid();
const user2Id = uuid();
const publicUserId = uuid(); // A public user (not a bot)
const user1BotId = uuid(); // A bot owned by user1
const publicBotId = uuid(); // A public bot not owned by user1

// Sample User data structure
const userData1 = {
    id: user1Id,
    isBot: false,
    isPrivate: false,
    name: "Test User 1 - Public",
};

const userData2 = {
    id: user2Id,
    isBot: false,
    isPrivate: true,
    name: "Test User 2 - Private",
};

const publicUserData = {
    id: publicUserId,
    isBot: false,
    isPrivate: false,
    name: "Public User",
};

const user1BotData = {
    id: user1BotId,
    isBot: true,
    isPrivate: true,
    name: "User1's Bot - Private",
    invitedByUserId: user1Id,
};

const publicBotData = {
    id: publicBotId,
    isPrivate: false,
    isBot: true,
    name: "Public Bot",
    invitedByUserId: user2Id,
};

// Adjust fields based on actual StatsUser model
const statsUserData1 = {
    id: uuid(),
    userId: user1Id,
    periodStart: new Date("2023-01-01"),
    periodEnd: new Date("2023-01-31"),
    periodType: PeriodType.Monthly,
    apisCreated: 0,
    codesCreated: 0,
    codesCompleted: 0,
    codeCompletionTimeAverage: 0.0,
    projectsCreated: 0,
    projectsCompleted: 0,
    projectCompletionTimeAverage: 0.0,
    routinesCreated: 0,
    routinesCompleted: 0,
    routineCompletionTimeAverage: 0.0,
    standardsCreated: 0,
    teamsCreated: 0,
    runProjectsStarted: 0,
    runProjectsCompleted: 0,
    runProjectCompletionTimeAverage: 0.0,
    runProjectContextSwitchesAverage: 0.0,
    runRoutinesStarted: 0,
    runRoutinesCompleted: 0,
    runRoutineCompletionTimeAverage: 0.0,
    runRoutineContextSwitchesAverage: 0.0,
    standardsCompleted: 0,
    standardCompletionTimeAverage: 0.0,
};

const statsUserData2 = {
    id: uuid(),
    userId: user2Id,
    periodStart: new Date("2023-02-01"),
    periodEnd: new Date("2023-02-28"),
    periodType: PeriodType.Monthly,
    apisCreated: 0,
    codesCreated: 0,
    codesCompleted: 0,
    codeCompletionTimeAverage: 0.0,
    projectsCreated: 0,
    projectsCompleted: 0,
    projectCompletionTimeAverage: 0.0,
    routinesCreated: 0,
    routinesCompleted: 0,
    routineCompletionTimeAverage: 0.0,
    standardsCreated: 0,
    teamsCreated: 0,
    runProjectsStarted: 0,
    runProjectsCompleted: 0,
    runProjectCompletionTimeAverage: 0.0,
    runProjectContextSwitchesAverage: 0.0,
    runRoutinesStarted: 0,
    runRoutinesCompleted: 0,
    runRoutineCompletionTimeAverage: 0.0,
    runRoutineContextSwitchesAverage: 0.0,
    standardsCompleted: 0,
    standardCompletionTimeAverage: 0.0,
};

const statsPublicUserData = {
    id: uuid(),
    userId: publicUserId,
    periodStart: new Date("2023-03-01"),
    periodEnd: new Date("2023-03-31"),
    periodType: PeriodType.Monthly,
    apisCreated: 0,
    codesCreated: 0,
    codesCompleted: 0,
    codeCompletionTimeAverage: 0.0,
    projectsCreated: 0,
    projectsCompleted: 0,
    projectCompletionTimeAverage: 0.0,
    routinesCreated: 0,
    routinesCompleted: 0,
    routineCompletionTimeAverage: 0.0,
    standardsCreated: 0,
    teamsCreated: 0,
    runProjectsStarted: 0,
    runProjectsCompleted: 0,
    runProjectCompletionTimeAverage: 0.0,
    runProjectContextSwitchesAverage: 0.0,
    runRoutinesStarted: 0,
    runRoutinesCompleted: 0,
    runRoutineCompletionTimeAverage: 0.0,
    runRoutineContextSwitchesAverage: 0.0,
    standardsCompleted: 0,
    standardCompletionTimeAverage: 0.0,
};

const statsUser1BotData = {
    id: uuid(),
    userId: user1BotId,
    periodStart: new Date("2023-04-01"),
    periodEnd: new Date("2023-04-30"),
    periodType: PeriodType.Monthly,
    apisCreated: 0,
    codesCreated: 0,
    codesCompleted: 0,
    codeCompletionTimeAverage: 0.0,
    projectsCreated: 0,
    projectsCompleted: 0,
    projectCompletionTimeAverage: 0.0,
    routinesCreated: 0,
    routinesCompleted: 0,
    routineCompletionTimeAverage: 0.0,
    standardsCreated: 0,
    teamsCreated: 0,
    runProjectsStarted: 0,
    runProjectsCompleted: 0,
    runProjectCompletionTimeAverage: 0.0,
    runProjectContextSwitchesAverage: 0.0,
    runRoutinesStarted: 0,
    runRoutinesCompleted: 0,
    runRoutineCompletionTimeAverage: 0.0,
    runRoutineContextSwitchesAverage: 0.0,
    standardsCompleted: 0,
    standardCompletionTimeAverage: 0.0,
};

const statsPublicBotData = {
    id: uuid(),
    userId: publicBotId,
    periodStart: new Date("2023-01-01"),
    periodEnd: new Date("2023-05-31"),
    periodType: PeriodType.Monthly,
    apisCreated: 0,
    codesCreated: 0,
    codesCompleted: 0,
    codeCompletionTimeAverage: 0.0,
    projectsCreated: 0,
    projectsCompleted: 0,
    projectCompletionTimeAverage: 0.0,
    routinesCreated: 0,
    routinesCompleted: 0,
    routineCompletionTimeAverage: 0.0,
    standardsCreated: 0,
    teamsCreated: 0,
    runProjectsStarted: 0,
    runProjectsCompleted: 0,
    runProjectCompletionTimeAverage: 0.0,
    runProjectContextSwitchesAverage: 0.0,
    runRoutinesStarted: 0,
    runRoutinesCompleted: 0,
    runRoutineCompletionTimeAverage: 0.0,
    runRoutineContextSwitchesAverage: 0.0,
    standardsCompleted: 0,
    standardCompletionTimeAverage: 0.0,
};

const allStattedObjectIds = [userData1, userData2, publicUserData, user1BotData, publicBotData];
async function extractStattedObjectInfoFromStats(result: RecursivePartial<StatsUserSearchResult>) {
    const resultsStatIds = result.edges!.map(edge => edge!.node!.id) as string[];
    const withStattedIds = await DbProvider.get().stats_user.findMany({
        where: { id: { in: resultsStatIds } },
        select: { userId: true },
    });
    const resultStattedIds = withStattedIds.map(stat => stat[Object.keys(stat)[0]]);
    const resultStattedNames = allStattedObjectIds.filter(user => resultStattedIds.includes(user.id)).map(user => user.name);

    return { resultStattedIds, resultStattedNames };
}

describe("EndpointsStatsUser", () => {
    let loggerErrorStub: any;
    let loggerInfoStub: any;

    beforeAll(() => {
        loggerErrorStub = vi.spyOn(logger, "error").mockImplementation(() => undefined);
        loggerInfoStub = vi.spyOn(logger, "info").mockImplementation(() => undefined);
    });

    beforeEach(async () => {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Create test users
        await DbProvider.get().user.create({
            data: { ...userData1, handle: "test-user-1", status: "Unlocked", isBot: false, isBotDepictingPerson: false, auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] } },
        });
        await DbProvider.get().user.create({
            data: { ...userData2, handle: "test-user-2", status: "Unlocked", isBot: false, isBotDepictingPerson: false, auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] } },
        });
        await DbProvider.get().user.create({
            data: { ...publicUserData, handle: "public-user", status: "Unlocked", isBot: false, isBotDepictingPerson: false, auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] } },
        });
        // Create the bot owned by user1
        await DbProvider.get().user.create({
            data: {
                ...user1BotData,
                handle: "user1-bot",
                status: "Unlocked",
                isBotDepictingPerson: false,
                invitedByUserId: user1Id,
            },
        });
        // Create the public bot not owned by user1
        await DbProvider.get().user.create({
            data: {
                ...publicBotData,
                handle: "public-bot",
                status: "Unlocked",
                isBotDepictingPerson: false,
                invitedByUserId: user2Id,
            },
        });

        // Create fresh test stats data
        await DbProvider.get().stats_user.createMany({
            data: [
                statsUserData1,
                statsUserData2,
                statsPublicUserData,
                statsUser1BotData,
                statsPublicBotData,
            ],
        });
    });

    afterAll(async () => {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        loggerErrorStub.mockRestore();
        loggerInfoStub.mockRestore();
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns own stats, stats for own bots, and stats for public users/bots", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsUserSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const expectedStattedIds = [
                    user1Id, // Own (Public)
                    //user2Id, // Private User
                    publicUserId, // Public User
                    user1BotId, // Own Bot
                    publicBotId, // Public Bot
                    //privateBotId, // Private Bot
                ];

                const result = await statsUser.findMany({ input }, { req, res }, statsUser_findMany);

                expect(result).not.toBeNull();
                const { resultStattedIds, resultStattedNames } = await extractStattedObjectInfoFromStats(result);
                expect(resultStattedIds.sort()).toEqual(expectedStattedIds.sort());
            });

            it("properly filters by periodType", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsUserSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: {
                        after: new Date("2023-01-01"),
                        before: new Date("2023-01-31"),
                    },
                };
                const expectedStatIds = [
                    statsUserData1.id,
                    // statsUserData2.id,
                    // statsPublicUserData.id,
                    // statsUser1BotData.id,
                    // statsPublicBotData.id
                ];

                const result = await statsUser.findMany({ input }, { req, res }, statsUser_findMany);
                const resultStatIds = result.edges!.map(edge => edge!.node!.id);

                expect(result).not.toBeNull();
                expect(expectedStatIds.sort()).toEqual(resultStatIds.sort());
            });

            it("API key with public permission returns public user/bot stats", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: StatsUserSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const expectedStattedIds = [
                    user1Id, // Own (Public)
                    // user2Id, // Private User
                    publicUserId, // Public User
                    // user1BotId, // Own Bot
                    publicBotId, // Public Bot
                    // privateBotId, // Private Bot
                ];

                const result = await statsUser.findMany({ input }, { req, res }, statsUser_findMany);
                const { resultStattedIds, resultStattedNames } = await extractStattedObjectInfoFromStats(result);
                expect(resultStattedIds.sort()).toEqual(expectedStattedIds.sort());
            });

            it("not logged in returns only public user/bot stats", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: StatsUserSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const expectedStattedIds = [
                    user1Id, // Own (Public)
                    // user2Id, // Private User
                    publicUserId, // Public User
                    // user1BotId, // Own Bot
                    publicBotId, // Public Bot
                    // privateBotId, // Private Bot
                ];

                const result = await statsUser.findMany({ input }, { req, res }, statsUser_findMany);

                expect(result).not.toBeNull();
                const { resultStattedIds, resultStattedNames } = await extractStattedObjectInfoFromStats(result);
                expect(resultStattedIds.sort()).toEqual(expectedStattedIds.sort());
            });
        });

        describe("invalid", () => {
            it("invalid time range format should throw error", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsUserSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: { after: new Date("invalid"), before: new Date("invalid") },
                };

                try {
                    await statsUser.findMany({ input }, { req, res }, statsUser_findMany);
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
                    await statsUser.findMany({ input: input as StatsUserSearchInput }, { req, res }, statsUser_findMany);
                    expect.fail("Expected an error");
                } catch (error) {
                    expect(error).toBeInstanceOf(Error);
                }
            });
        });
    });
}); 
