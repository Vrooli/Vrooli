import { StatPeriodType, type StatsRoutineSearchInput, generatePK } from "@vrooli/shared";
import { PeriodType, type routine as RoutineModelPrisma } from "@prisma/client";
import { describe, it, expect, beforeAll, afterAll, vi } from "vitest";
import { defaultPublicUserData, loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { statsRoutine_findMany } from "../generated/statsRoutine_findMany.js"; // Assuming this generated type exists
import { statsRoutine } from "./statsResource.js";


describe("EndpointsStatsRoutine", () => {
    let loggerErrorStub: any;
    let loggerInfoStub: any;

    beforeAll(async () => {
        loggerErrorStub = vi.spyOn(logger, "error").mockImplementation(() => undefined);
        loggerInfoStub = vi.spyOn(logger, "info").mockImplementation(() => undefined);
    });

    beforeEach(async () => {
        // Clean up tables used in tests
        const prisma = DbProvider.get();
        await prisma.routine.deleteMany();
        await prisma.stats_routine.deleteMany();
        await prisma.user.deleteMany();
    });

    afterAll(async () => {
        loggerErrorStub.mockRestore();
        loggerInfoStub.mockRestore();
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns stats for public and owned routines when logged in", async () => {
                // Create test users
                const user1 = await DbProvider.get().user.create({
                    data: {
                        ...defaultPublicUserData(),
                        id: generatePK(),
                        name: "Test User 1",
                    },
                });
                const user2 = await DbProvider.get().user.create({
                    data: {
                        ...defaultPublicUserData(),
                        id: generatePK(),
                        name: "Test User 2",
                    },
                });

                // Create test routines
                const routine1 = await DbProvider.get().routine.create({
                    data: {
                        id: generatePK(),
                        isPrivate: false,
                        ownedByUserId: null,
                        permissions: JSON.stringify({}),
                    },
                });
                const routine2 = await DbProvider.get().routine.create({
                    data: {
                        id: generatePK(),
                        isPrivate: false,
                        ownedByUserId: null,
                        permissions: JSON.stringify({}),
                    },
                });
                const privateRoutine1 = await DbProvider.get().routine.create({
                    data: {
                        id: generatePK(),
                        isPrivate: true,
                        ownedByUserId: user1.id,
                        permissions: JSON.stringify({}),
                    },
                });
                const privateRoutine2 = await DbProvider.get().routine.create({
                    data: {
                        id: generatePK(),
                        isPrivate: true,
                        ownedByUserId: user2.id,
                        permissions: JSON.stringify({}),
                    },
                });

                // Create stats
                const statsRoutineData1 = await DbProvider.get().stats_routine.create({
                    data: {
                        id: generatePK(),
                        routineId: routine1.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        runsStarted: 0,
                        runsCompleted: 0,
                        runCompletionTimeAverage: 0.0,
                        runContextSwitchesAverage: 0.0,
                    },
                });
                const statsRoutineData2 = await DbProvider.get().stats_routine.create({
                    data: {
                        id: generatePK(),
                        routineId: routine2.id,
                        periodStart: new Date("2023-02-01"),
                        periodEnd: new Date("2023-02-28"),
                        periodType: PeriodType.Monthly,
                        runsStarted: 0,
                        runsCompleted: 0,
                        runCompletionTimeAverage: 0.0,
                        runContextSwitchesAverage: 0.0,
                    },
                });
                const privateRoutineStats1 = await DbProvider.get().stats_routine.create({
                    data: {
                        id: generatePK(),
                        routineId: privateRoutine1.id,
                        periodStart: new Date("2023-03-01"),
                        periodEnd: new Date("2023-03-31"),
                        periodType: PeriodType.Monthly,
                        runsStarted: 0,
                        runsCompleted: 0,
                        runCompletionTimeAverage: 0.0,
                        runContextSwitchesAverage: 0.0,
                    },
                });
                const privateRoutineStats2 = await DbProvider.get().stats_routine.create({
                    data: {
                        id: generatePK(),
                        routineId: privateRoutine2.id,
                        periodStart: new Date("2023-03-01"),
                        periodEnd: new Date("2023-03-31"),
                        periodType: PeriodType.Monthly,
                        runsStarted: 0,
                        runsCompleted: 0,
                        runCompletionTimeAverage: 0.0,
                        runContextSwitchesAverage: 0.0,
                    },
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: user1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsRoutineSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsRoutine.findMany({ input }, { req, res }, statsRoutine_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                // User 1 should see public routines and their own private routine
                expect(resultIds).toContain(statsRoutineData1.id.toString());
                expect(resultIds).toContain(statsRoutineData2.id.toString());
                expect(resultIds).toContain(privateRoutineStats1.id.toString());
                // User 1 should NOT see user 2's private routine stats
                expect(resultIds).not.toContain(privateRoutineStats2.id.toString());
            });

            it("filters by periodType", async () => {
                const user1 = await DbProvider.get().user.create({
                    data: {
                        ...defaultPublicUserData(),
                        id: generatePK(),
                        name: "Test User 1",
                    },
                });

                const routine = await DbProvider.get().routine.create({
                    data: {
                        id: generatePK(),
                        isPrivate: false,
                        ownedByUserId: user1.id,
                        permissions: JSON.stringify({}),
                    },
                });

                const monthlyStats = await DbProvider.get().stats_routine.create({
                    data: {
                        id: generatePK(),
                        routineId: routine.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        runsStarted: 0,
                        runsCompleted: 0,
                        runCompletionTimeAverage: 0.0,
                        runContextSwitchesAverage: 0.0,
                    },
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: user1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsRoutineSearchInput = { periodType: StatPeriodType.Monthly };
                const result = await statsRoutine.findMany({ input }, { req, res }, statsRoutine_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                const resultIds = result.edges!.map(edge => edge?.node?.id);
                expect(resultIds).toContain(monthlyStats.id.toString());
            });

            it("filters by time range", async () => {
                const user1 = await DbProvider.get().user.create({
                    data: {
                        ...defaultPublicUserData(),
                        id: generatePK(),
                        name: "Test User 1",
                    },
                });

                const routine = await DbProvider.get().routine.create({
                    data: {
                        id: generatePK(),
                        isPrivate: false,
                        ownedByUserId: user1.id,
                        permissions: JSON.stringify({}),
                    },
                });

                const janStats = await DbProvider.get().stats_routine.create({
                    data: {
                        id: generatePK(),
                        routineId: routine.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        runsStarted: 0,
                        runsCompleted: 0,
                        runCompletionTimeAverage: 0.0,
                        runContextSwitchesAverage: 0.0,
                    },
                });

                const febStats = await DbProvider.get().stats_routine.create({
                    data: {
                        id: generatePK(),
                        routineId: routine.id,
                        periodStart: new Date("2023-02-01"),
                        periodEnd: new Date("2023-02-28"),
                        periodType: PeriodType.Monthly,
                        runsStarted: 0,
                        runsCompleted: 0,
                        runCompletionTimeAverage: 0.0,
                        runContextSwitchesAverage: 0.0,
                    },
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: user1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsRoutineSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: {
                        after: new Date("2023-01-01"),
                        before: new Date("2023-01-31"),
                    },
                };
                const result = await statsRoutine.findMany({ input }, { req, res }, statsRoutine_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                const resultIds = result.edges!.map(edge => edge?.node?.id);
                expect(resultIds).toContain(janStats.id.toString()); // Should include Jan stats
                expect(resultIds).not.toContain(febStats.id.toString()); // Should exclude Feb stats
            });

            it("API key - public permissions (likely returns only public routines)", async () => {
                const user1 = await DbProvider.get().user.create({
                    data: {
                        ...defaultPublicUserData(),
                        id: generatePK(),
                        name: "Test User 1",
                    },
                });

                const publicRoutine1 = await DbProvider.get().routine.create({
                    data: {
                        id: generatePK(),
                        isPrivate: false,
                        ownedByUserId: null,
                        permissions: JSON.stringify({}),
                    },
                });

                const publicRoutine2 = await DbProvider.get().routine.create({
                    data: {
                        id: generatePK(),
                        isPrivate: false,
                        ownedByUserId: null,
                        permissions: JSON.stringify({}),
                    },
                });

                const privateRoutine = await DbProvider.get().routine.create({
                    data: {
                        id: generatePK(),
                        isPrivate: true,
                        ownedByUserId: user1.id,
                        permissions: JSON.stringify({}),
                    },
                });

                const publicStats1 = await DbProvider.get().stats_routine.create({
                    data: {
                        id: generatePK(),
                        routineId: publicRoutine1.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        runsStarted: 0,
                        runsCompleted: 0,
                        runCompletionTimeAverage: 0.0,
                        runContextSwitchesAverage: 0.0,
                    },
                });

                const publicStats2 = await DbProvider.get().stats_routine.create({
                    data: {
                        id: generatePK(),
                        routineId: publicRoutine2.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        runsStarted: 0,
                        runsCompleted: 0,
                        runCompletionTimeAverage: 0.0,
                        runContextSwitchesAverage: 0.0,
                    },
                });

                const privateStats = await DbProvider.get().stats_routine.create({
                    data: {
                        id: generatePK(),
                        routineId: privateRoutine.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        runsStarted: 0,
                        runsCompleted: 0,
                        runCompletionTimeAverage: 0.0,
                        runContextSwitchesAverage: 0.0,
                    },
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: user1.id.toString() };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: StatsRoutineSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsRoutine.findMany({ input }, { req, res }, statsRoutine_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                // Expect only public routine stats
                expect(resultIds).toContain(publicStats1.id.toString());
                expect(resultIds).toContain(publicStats2.id.toString());
                expect(resultIds).not.toContain(privateStats.id.toString());
            });

            it("not logged in (likely returns empty or only public, depending on readManyHelper)", async () => {
                const publicRoutine1 = await DbProvider.get().routine.create({
                    data: {
                        id: generatePK(),
                        isPrivate: false,
                        ownedByUserId: null,
                        permissions: JSON.stringify({}),
                    },
                });

                const publicRoutine2 = await DbProvider.get().routine.create({
                    data: {
                        id: generatePK(),
                        isPrivate: false,
                        ownedByUserId: null,
                        permissions: JSON.stringify({}),
                    },
                });

                const user1 = await DbProvider.get().user.create({
                    data: {
                        ...defaultPublicUserData(),
                        id: generatePK(),
                        name: "Test User 1",
                    },
                });

                const privateRoutine = await DbProvider.get().routine.create({
                    data: {
                        id: generatePK(),
                        isPrivate: true,
                        ownedByUserId: user1.id,
                        permissions: JSON.stringify({}),
                    },
                });

                const publicStats1 = await DbProvider.get().stats_routine.create({
                    data: {
                        id: generatePK(),
                        routineId: publicRoutine1.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        runsStarted: 0,
                        runsCompleted: 0,
                        runCompletionTimeAverage: 0.0,
                        runContextSwitchesAverage: 0.0,
                    },
                });

                const publicStats2 = await DbProvider.get().stats_routine.create({
                    data: {
                        id: generatePK(),
                        routineId: publicRoutine2.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        runsStarted: 0,
                        runsCompleted: 0,
                        runCompletionTimeAverage: 0.0,
                        runContextSwitchesAverage: 0.0,
                    },
                });

                const privateStats = await DbProvider.get().stats_routine.create({
                    data: {
                        id: generatePK(),
                        routineId: privateRoutine.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        runsStarted: 0,
                        runsCompleted: 0,
                        runCompletionTimeAverage: 0.0,
                        runContextSwitchesAverage: 0.0,
                    },
                });

                const { req, res } = await mockLoggedOutSession();

                const input: StatsRoutineSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };

                const result = await statsRoutine.findMany({ input }, { req, res }, statsRoutine_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                // Expect only public routine stats
                expect(resultIds).toContain(publicStats1.id.toString());
                expect(resultIds).toContain(publicStats2.id.toString());
                expect(resultIds).not.toContain(privateStats.id.toString());
            });

        });

        describe("invalid", () => {
            it("invalid time range format should throw error", async () => {
                const user1 = await DbProvider.get().user.create({
                    data: {
                        ...defaultPublicUserData(),
                        id: generatePK(),
                        name: "Test User 1",
                    },
                });
                
                const testUser = { ...loggedInUserNoPremiumData(), id: user1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsRoutineSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: {
                        after: new Date("invalid-date"), // Invalid date
                        before: new Date("invalid-date"),
                    },
                };

                try {
                    await statsRoutine.findMany({ input }, { req, res }, statsRoutine_findMany);
                    expect.fail("Expected an error to be thrown due to invalid date");
                } catch (error) {
                    expect(error).toBeInstanceOf(Error);
                }
            });

            it("invalid periodType should throw error", async () => {
                const user1 = await DbProvider.get().user.create({
                    data: {
                        ...defaultPublicUserData(),
                        id: generatePK(),
                        name: "Test User 1",
                    },
                });
                
                const testUser = { ...loggedInUserNoPremiumData(), id: user1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = {
                    periodType: "InvalidPeriodType" as any,
                };

                try {
                    await statsRoutine.findMany({ input: input as StatsRoutineSearchInput }, { req, res }, statsRoutine_findMany);
                    expect.fail("Expected an error to be thrown due to invalid periodType");
                } catch (error) {
                    expect(error).toBeInstanceOf(Error);
                }
            });

            it("cannot see stats of private routine you don't own when searching by name", async () => {
                const user1 = await DbProvider.get().user.create({
                    data: {
                        ...defaultPublicUserData(),
                        id: generatePK(),
                        name: "Test User 1",
                    },
                });

                const user2 = await DbProvider.get().user.create({
                    data: {
                        ...defaultPublicUserData(),
                        id: generatePK(),
                        name: "Test User 2",
                    },
                });

                const privateRoutine2 = await DbProvider.get().routine.create({
                    data: {
                        id: generatePK(),
                        isPrivate: true,
                        ownedByUserId: user2.id,
                        permissions: JSON.stringify({}),
                    },
                });

                await DbProvider.get().stats_routine.create({
                    data: {
                        id: generatePK(),
                        routineId: privateRoutine2.id,
                        periodStart: new Date("2023-03-01"),
                        periodEnd: new Date("2023-03-31"),
                        periodType: PeriodType.Monthly,
                        runsStarted: 0,
                        runsCompleted: 0,
                        runCompletionTimeAverage: 0.0,
                        runContextSwitchesAverage: 0.0,
                    },
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: user1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsRoutineSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    searchString: "Private Routine 2",
                };

                const result = await statsRoutine.findMany({ input }, { req, res }, statsRoutine_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                expect(result.edges!.length).toEqual(0);
            });
        });
    });
}); 
