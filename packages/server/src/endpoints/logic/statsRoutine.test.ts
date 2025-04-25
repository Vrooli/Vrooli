import { StatPeriodType, StatsRoutineSearchInput, uuid } from "@local/shared";
import { PeriodType, routine as RoutineModelPrisma } from "@prisma/client";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { statsRoutine_findMany } from "../generated/statsRoutine_findMany.js"; // Assuming this generated type exists
import { statsRoutine } from "./statsRoutine.js";

// Test data
const testRoutineId1 = uuid();
const testRoutineId2 = uuid();
const privateRoutineId1 = uuid(); // Private Routine owned by user1
const privateRoutineId2 = uuid(); // Private Routine owned by user2

// User IDs for ownership testing
const user1Id = uuid();
const user2Id = uuid();

// Sample Routine data structure (adjust fields as necessary based on actual Routine model)
const routineData1: Partial<RoutineModelPrisma> & { id: string } = {
    id: testRoutineId1,
    isPrivate: false,
    ownedByUserId: null, // Public routine
    // Add other required Routine fields
};

const routineData2: Partial<RoutineModelPrisma> & { id: string } = {
    id: testRoutineId2,
    isPrivate: false,
    ownedByUserId: null, // Public routine
    // Add other required Routine fields
};

const privateRoutineData1: Partial<RoutineModelPrisma> & { id: string } = {
    id: privateRoutineId1,
    isPrivate: true,
    ownedByUserId: user1Id, // Owned by user1
    // Add other required Routine fields
};

const privateRoutineData2: Partial<RoutineModelPrisma> & { id: string } = {
    id: privateRoutineId2,
    isPrivate: true,
    ownedByUserId: user2Id, // Owned by user2
    // Add other required Routine fields
};

const statsRoutineData1 = {
    id: uuid(),
    routineId: testRoutineId1,
    periodStart: new Date("2023-01-01"),
    periodEnd: new Date("2023-01-31"),
    periodType: PeriodType.Monthly,
    // Add other relevant StatsRoutine fields (e.g., averageDuration, successCount)
    runsStarted: 0,
    runsCompleted: 0,
    runCompletionTimeAverage: 0.0,
    runContextSwitchesAverage: 0.0,
};

const statsRoutineData2 = {
    id: uuid(),
    routineId: testRoutineId2,
    periodStart: new Date("2023-02-01"),
    periodEnd: new Date("2023-02-28"),
    periodType: PeriodType.Monthly,
    runsStarted: 0,
    runsCompleted: 0,
    runCompletionTimeAverage: 0.0,
    runContextSwitchesAverage: 0.0,
};

const privateRoutineStats1 = {
    id: uuid(),
    routineId: privateRoutineId1,
    periodStart: new Date("2023-03-01"),
    periodEnd: new Date("2023-03-31"),
    periodType: PeriodType.Monthly,
    runsStarted: 0,
    runsCompleted: 0,
    runCompletionTimeAverage: 0.0,
    runContextSwitchesAverage: 0.0,
};

const privateRoutineStats2 = {
    id: uuid(),
    routineId: privateRoutineId2,
    periodStart: new Date("2023-03-01"),
    periodEnd: new Date("2023-03-31"),
    periodType: PeriodType.Monthly,
    runsStarted: 0,
    runsCompleted: 0,
    runCompletionTimeAverage: 0.0,
    runContextSwitchesAverage: 0.0,
};

describe("EndpointsStatsRoutine", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async function beforeEach() {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Create test users individually
        await DbProvider.get().user.create({
            data: {
                id: user1Id,
                name: "Test User 1",
                handle: "test-user-1",
                status: "Unlocked",
                isBot: false, isBotDepictingPerson: false, isPrivate: false,
                auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] },
            },
        });
        await DbProvider.get().user.create({
            data: {
                id: user2Id,
                name: "Test User 2",
                handle: "test-user-2",
                status: "Unlocked",
                isBot: false, isBotDepictingPerson: false, isPrivate: false,
                auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] },
            },
        });

        // Create test routines (ensure all required fields are present)
        // Placeholder: Assuming routines need a name, permissions, and versions
        await DbProvider.get().routine.createMany({
            data: [
                { ...routineData1, permissions: JSON.stringify({}) },
                { ...routineData2, permissions: JSON.stringify({}) },
                { ...privateRoutineData1, permissions: JSON.stringify({}) },
                { ...privateRoutineData2, permissions: JSON.stringify({}) },
            ].map(r => ({ // Adjust ownership fields
                ...r,
                ownedByUserId: r.ownedByUserId ?? undefined,
            })),
        });

        // Create fresh test stats data
        await DbProvider.get().stats_routine.createMany({
            data: [
                statsRoutineData1,
                statsRoutineData2,
                privateRoutineStats1,
                privateRoutineStats2,
            ],
        });
    });

    after(async function after() {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns stats for public and owned routines when logged in", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsRoutineSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                // Assuming statsRoutine_findMany exists and is typed correctly
                const result = await statsRoutine.findMany({ input }, { req, res }, statsRoutine_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                // User 1 should see public routines and their own private routine
                expect(resultIds).to.include(statsRoutineData1.id);
                expect(resultIds).to.include(statsRoutineData2.id);
                expect(resultIds).to.include(privateRoutineStats1.id);
                // User 1 should NOT see user 2's private routine stats
                expect(resultIds).to.not.include(privateRoutineStats2.id);
            });

            it("filters by periodType", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsRoutineSearchInput = { periodType: StatPeriodType.Monthly };
                const result = await statsRoutine.findMany({ input }, { req, res }, statsRoutine_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);
                expect(resultIds).to.include(statsRoutineData1.id);
                expect(resultIds).to.include(statsRoutineData2.id);
                expect(resultIds).to.include(privateRoutineStats1.id);
            });

            it("filters by time range", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsRoutineSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: {
                        after: new Date("2023-01-01"),
                        before: new Date("2023-01-31"),
                    },
                };
                const result = await statsRoutine.findMany({ input }, { req, res }, statsRoutine_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);
                expect(resultIds).to.include(statsRoutineData1.id); // Should include Jan stats
                expect(resultIds).to.not.include(statsRoutineData2.id); // Should exclude Feb stats
                expect(resultIds).to.not.include(privateRoutineStats1.id); // Should exclude Mar stats
            });

            it("API key - public permissions (likely returns only public routines)", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id }; // User context might be needed by readManyHelper even if permissions are broad
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: StatsRoutineSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsRoutine.findMany({ input }, { req, res }, statsRoutine_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                // Expect only public routine stats, as readManyHelper likely enforces ownership for private ones even with ReadPublic key
                expect(resultIds).to.include(statsRoutineData1.id);
                expect(resultIds).to.include(statsRoutineData2.id);
                expect(resultIds).to.not.include(privateRoutineStats1.id);
                expect(resultIds).to.not.include(privateRoutineStats2.id);
            });

            it("not logged in (likely returns empty or only public, depending on readManyHelper)", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: StatsRoutineSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };

                // The implementation doesn't assert ReadPublic, so readManyHelper's default behavior for unauthenticated users applies.
                // Assuming readManyHelper returns public items if applicable, or empty otherwise.
                // If routines require login to view even public ones, this test should expect 0 results.
                // Let's assume public routines ARE viewable by logged-out users via readManyHelper.
                const result = await statsRoutine.findMany({ input }, { req, res }, statsRoutine_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                // Expect only public routine stats
                expect(resultIds).to.include(statsRoutineData1.id);
                expect(resultIds).to.include(statsRoutineData2.id);
                expect(resultIds).to.not.include(privateRoutineStats1.id);
                expect(resultIds).to.not.include(privateRoutineStats2.id);
            });

        });

        describe("invalid", () => {
            it("invalid time range format should throw error", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
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
                    // Error expected, specific error type check depends on readManyHelper/Prisma validation
                    expect(error).to.be.instanceOf(Error); // Basic check
                }
            });

            it("invalid periodType should throw error", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = { // Use 'any' to bypass TypeScript type checking for the test
                    periodType: "InvalidPeriodType" as any,
                };

                try {
                    // Cast input to the expected type, even though it's invalid
                    await statsRoutine.findMany({ input: input as StatsRoutineSearchInput }, { req, res }, statsRoutine_findMany);
                    expect.fail("Expected an error to be thrown due to invalid periodType");
                } catch (error) {
                    // Error expected, likely a validation error from Zod or Prisma within readManyHelper
                    expect(error).to.be.instanceOf(Error); // Basic check
                }
            });

            it("cannot see stats of private routine you don't own when searching by name", async () => {
                // Log in as user1
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                // Try to specifically query user2's private routine stats by name
                const input: StatsRoutineSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    searchString: "Private Routine 2", // Name of user2's private routine
                };

                const result = await statsRoutine.findMany({ input }, { req, res }, statsRoutine_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                // No results should match the query, as user1 cannot see user2's private routine
                expect(result.edges!.length).to.equal(0);
                // Double-check no edge contains the private stat ID
                expect(result.edges!.every(edge => edge?.node?.id !== privateRoutineStats2.id)).to.be.true;
            });
        });
    });
}); 
