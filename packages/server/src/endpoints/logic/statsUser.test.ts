import { ApiKeyPermission, StatPeriodType, StatsUserSearchInput, uuid } from "@local/shared";
import { PeriodType, user as UserModelPrisma } from "@prisma/client";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { statsUser_findMany } from "../generated/statsUser_findMany.js"; // Assuming this generated type exists
import { statsUser } from "./statsUser.js";

// User IDs
const user1Id = uuid();
const user2Id = uuid();
const publicUserId = uuid(); // A potentially public user

// Sample User data structure
const userData1: Partial<UserModelPrisma> & { id: string } = {
    id: user1Id,
    isPrivate: false, // Assuming default is not private
    // Add other required User fields (name, handle, etc.)
};

const userData2: Partial<UserModelPrisma> & { id: string } = {
    id: user2Id,
    isPrivate: false,
};

const publicUserData: Partial<UserModelPrisma> & { id: string } = {
    id: publicUserId,
    isPrivate: false, // Explicitly public for testing
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
    quizzesPassed: 0,
    quizCompletionTimeAverage: 0.0,
    quizScoreAverage: 0.0,
    routinesCreated: 0,
    routinesCompleted: 0,
    routineCompletionTimeAverage: 0.0,
    runContextSwitchesAverage: 0.0,
    standardsCreated: 0,
    standardsImplemented: 0,
    standardsViews: 0,
    teamsCreated: 0,
    teamsJoined: 0,
    teamsLeft: 0,
    quizzesFailed: 0,
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
    quizzesPassed: 0,
    quizCompletionTimeAverage: 0.0,
    quizScoreAverage: 0.0,
    routinesCreated: 0,
    routinesCompleted: 0,
    routineCompletionTimeAverage: 0.0,
    runContextSwitchesAverage: 0.0,
    standardsCreated: 0,
    standardsImplemented: 0,
    standardsViews: 0,
    teamsCreated: 0,
    teamsJoined: 0,
    teamsLeft: 0,
    quizzesFailed: 0,
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
    quizzesPassed: 0,
    quizCompletionTimeAverage: 0.0,
    quizScoreAverage: 0.0,
    routinesCreated: 0,
    routinesCompleted: 0,
    routineCompletionTimeAverage: 0.0,
    runContextSwitchesAverage: 0.0,
    standardsCreated: 0,
    standardsImplemented: 0,
    standardsViews: 0,
    teamsCreated: 0,
    teamsJoined: 0,
    teamsLeft: 0,
    quizzesFailed: 0,
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

describe("EndpointsStatsUser", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async function beforeEach() {
        this.timeout(15_000);

        // Clean previous test data
        await DbProvider.get().stats_user.deleteMany({
            where: { userId: { in: [user1Id, user2Id, publicUserId] } }
        });
        await DbProvider.get().user.deleteMany({
            where: { id: { in: [user1Id, user2Id, publicUserId] } }
        });

        // Create test users
        await DbProvider.get().user.create({
            data: { ...userData1, name: "Test User 1", handle: "test-user-1", status: "Unlocked", isBot: false, isBotDepictingPerson: false, auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] } }
        });
        await DbProvider.get().user.create({
            data: { ...userData2, name: "Test User 2", handle: "test-user-2", status: "Unlocked", isBot: false, isBotDepictingPerson: false, auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] } }
        });
        await DbProvider.get().user.create({
            data: { ...publicUserData, name: "Public User", handle: "public-user", status: "Unlocked", isBot: false, isBotDepictingPerson: false, auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] } }
        });

        // Create fresh test stats data
        await DbProvider.get().stats_user.createMany({
            data: [
                statsUserData1,
                statsUserData2,
                statsPublicUserData
            ]
        });
    });

    after(async function after() {
        this.timeout(15_000);

        // Clean up test data
        await DbProvider.get().stats_user.deleteMany({
            where: { userId: { in: [user1Id, user2Id, publicUserId] } }
        });
        await DbProvider.get().user.deleteMany({
            where: { id: { in: [user1Id, user2Id, publicUserId] } }
        });

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns only the logged-in user's stats", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsUserSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly
                };
                // Assuming readManyHelper for StatsUser filters by req.user.id implicitly
                const result = await statsUser.findMany({ input }, { req, res }, statsUser_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                expect(result.edges!.length).to.equal(1); // Should only get own stats
                const resultNode = result.edges![0]?.node;
                expect(resultNode?.id).to.equal(statsUserData1.id);
                // Ensure other users' stats are not returned
                expect(result.edges!.some(e => e?.node?.id === statsUserData2.id)).to.be.false;
                expect(result.edges!.some(e => e?.node?.id === statsPublicUserData.id)).to.be.false;
            });

            it("returns own stats when filtering by periodType", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsUserSearchInput = { periodType: StatPeriodType.Monthly };
                const result = await statsUser.findMany({ input }, { req, res }, statsUser_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                expect(result.edges!.length).to.equal(1);
                expect(result.edges![0]?.node?.id).to.equal(statsUserData1.id);
            });

            it("returns own stats when filtering by time range (matching)", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsUserSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: {
                        after: new Date("2023-01-01"),
                        before: new Date("2023-01-31")
                    }
                };
                const result = await statsUser.findMany({ input }, { req, res }, statsUser_findMany);

                expect(result).to.not.be.null;
                expect(result.edges!.length).to.equal(1);
                expect(result.edges![0]?.node?.id).to.equal(statsUserData1.id);
            });

            it("returns empty list when filtering by time range (not matching)", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id }; // User 1 stats are in Jan
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsUserSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: {
                        after: new Date("2023-02-01"), // Feb range
                        before: new Date("2023-02-28")
                    }
                };
                const result = await statsUser.findMany({ input }, { req, res }, statsUser_findMany);

                expect(result).to.not.be.null;
                expect(result.edges!.length).to.equal(0); // No stats for user 1 in Feb
            });

            // Depending on readManyHelper logic, API keys/logged out might see public user stats
            // Test A: Assuming NO stats are returned for API keys/logged out
            it("API key - returns no user stats", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id }; // Provide user context just in case
                const permissions = { [ApiKeyPermission.ReadPublic]: true } as Record<ApiKeyPermission, boolean>;
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: StatsUserSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly
                };
                // Assuming readManyHelper requires an authenticated user session for StatsUser
                const result = await statsUser.findMany({ input }, { req, res }, statsUser_findMany);

                expect(result).to.not.be.null;
                expect(result.edges!.length).to.equal(0);
            });

            it("not logged in - returns no user stats", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: StatsUserSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly
                };
                // Assuming readManyHelper requires an authenticated user session for StatsUser
                const result = await statsUser.findMany({ input }, { req, res }, statsUser_findMany);

                expect(result).to.not.be.null;
                expect(result.edges!.length).to.equal(0);
            });

            /* // Test B: Alternative if public user stats ARE visible
            it("API key - public permissions returns only public user stats", async () => {
                // ... setup api session ...
                const input: StatsUserSearchInput = { take: 10, periodType: StatPeriodType.Monthly };
                const result = await statsUser.findMany({ input }, { req, res }, statsUser_findMany);
                expect(result.edges!.length).to.equal(1);
                expect(result.edges![0]?.node?.id).to.equal(statsPublicUserData.id);
            });

            it("not logged in returns only public user stats", async () => {
                // ... setup logged out session ...
                const input: StatsUserSearchInput = { take: 10, periodType: StatPeriodType.Monthly };
                const result = await statsUser.findMany({ input }, { req, res }, statsUser_findMany);
                expect(result.edges!.length).to.equal(1);
                expect(result.edges![0]?.node?.id).to.equal(statsPublicUserData.id);
            });
            */
        });

        describe("invalid", () => {
            it("invalid time range format should throw error", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsUserSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: { after: new Date("invalid"), before: new Date("invalid") }
                };

                try {
                    await statsUser.findMany({ input }, { req, res }, statsUser_findMany);
                    expect.fail("Expected an error");
                } catch (error) {
                    expect(error).to.be.instanceOf(Error);
                }
            });

            it("invalid periodType should throw error", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = { periodType: "InvalidPeriod" as any };

                try {
                    await statsUser.findMany({ input: input as StatsUserSearchInput }, { req, res }, statsUser_findMany);
                    expect.fail("Expected an error");
                } catch (error) {
                    expect(error).to.be.instanceOf(Error);
                }
            });

            // Searching by name/handle might be possible, but shouldn't return other users' stats
            it("searching by another user's handle returns no results", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsUserSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    searchString: "test-user-2" // Handle of user2
                };
                const result = await statsUser.findMany({ input }, { req, res }, statsUser_findMany);

                expect(result).to.not.be.null;
                // Even though user2 exists, their stats shouldn't be returned to user1
                expect(result.edges!.length).to.equal(0);
            });
        });
    });
}); 