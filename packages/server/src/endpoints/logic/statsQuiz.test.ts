import { ApiKeyPermission, StatPeriodType, StatsQuizSearchInput, uuid } from "@local/shared";
import { PeriodType, quiz as QuizModelPrisma } from "@prisma/client"; // Assuming quiz model exists
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { statsQuiz_findMany } from "../generated/statsQuiz_findMany.js"; // Assuming this generated type exists
import { statsQuiz } from "./statsQuiz.js";

// Test data
const testQuizId1 = uuid();
const testQuizId2 = uuid();
const privateQuizId1 = uuid(); // Private Quiz owned by user1
const privateQuizId2 = uuid(); // Private Quiz owned by user2

// User IDs for ownership testing
const user1Id = uuid();
const user2Id = uuid();

// Sample Quiz data structure (adjust based on actual Quiz model)
const quizData1: Partial<QuizModelPrisma> & { id: string } = {
    id: testQuizId1,
    isPrivate: false,
    // ownedByUserId: null, // Removed - Adjust ownership based on actual model
    // Add other required Quiz fields (e.g., name, description, questions, createdById)
};

const quizData2: Partial<QuizModelPrisma> & { id: string } = {
    id: testQuizId2,
    isPrivate: false,
    // ownedByUserId: null, // Removed
    // Add other required Quiz fields
};

const privateQuizData1: Partial<QuizModelPrisma> & { id: string } = {
    id: privateQuizId1,
    isPrivate: true,
    // ownedByUserId: user1Id, // Removed - Link ownership via createdById or similar
    createdById: user1Id, // Assuming createdById determines ownership
    // Add other required Quiz fields
};

const privateQuizData2: Partial<QuizModelPrisma> & { id: string } = {
    id: privateQuizId2,
    isPrivate: true,
    // ownedByUserId: user2Id, // Removed
    createdById: user2Id, // Assuming createdById determines ownership
    // Add other required Quiz fields
};

// Adjust fields based on actual StatsQuiz model
const statsQuizData1 = {
    id: uuid(),
    quizId: testQuizId1,
    periodStart: new Date("2023-01-01"),
    periodEnd: new Date("2023-01-31"),
    periodType: PeriodType.Monthly,
    attempts: 100, // Example field
    // averageScore: 85.5, // Renamed to match model
    // Add required fields
    timesStarted: 0,
    timesPassed: 0,
    timesFailed: 0,
    scoreAverage: 85.5,
    completionTimeAverage: 0.0,
};

const statsQuizData2 = {
    id: uuid(),
    quizId: testQuizId2,
    periodStart: new Date("2023-02-01"),
    periodEnd: new Date("2023-02-28"),
    periodType: PeriodType.Monthly,
    attempts: 200,
    // averageScore: 90.0,
    timesStarted: 0,
    timesPassed: 0,
    timesFailed: 0,
    scoreAverage: 90.0,
    completionTimeAverage: 0.0,
};

const privateQuizStats1 = {
    id: uuid(),
    quizId: privateQuizId1,
    periodStart: new Date("2023-03-01"),
    periodEnd: new Date("2023-03-31"),
    periodType: PeriodType.Monthly,
    attempts: 50,
    // averageScore: 75.0,
    timesStarted: 0,
    timesPassed: 0,
    timesFailed: 0,
    scoreAverage: 75.0,
    completionTimeAverage: 0.0,
};

const privateQuizStats2 = {
    id: uuid(),
    quizId: privateQuizId2,
    periodStart: new Date("2023-03-01"),
    periodEnd: new Date("2023-03-31"),
    periodType: PeriodType.Monthly,
    attempts: 75,
    // averageScore: 80.0,
    timesStarted: 0,
    timesPassed: 0,
    timesFailed: 0,
    scoreAverage: 80.0,
    completionTimeAverage: 0.0,
};

describe("EndpointsStatsQuiz", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async function beforeEach() {
        this.timeout(15_000);

        // Clean previous test data
        await DbProvider.get().stats_quiz.deleteMany({
            where: { quizId: { in: [testQuizId1, testQuizId2, privateQuizId1, privateQuizId2] } }
        });
        await DbProvider.get().quiz.deleteMany({
            where: { id: { in: [testQuizId1, testQuizId2, privateQuizId1, privateQuizId2] } }
        });
        await DbProvider.get().user.deleteMany({
            where: { id: { in: [user1Id, user2Id] } }
        });

        // Create test users
        await DbProvider.get().user.create({
            data: { id: user1Id, name: "Test User 1", handle: "test-user-1", status: "Unlocked", isBot: false, isBotDepictingPerson: false, isPrivate: false, auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] } }
        });
        await DbProvider.get().user.create({
            data: { id: user2Id, name: "Test User 2", handle: "test-user-2", status: "Unlocked", isBot: false, isBotDepictingPerson: false, isPrivate: false, auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] } }
        });

        // Create test quizzes
        await DbProvider.get().quiz.createMany({
            data: [
                { ...quizData1 },
                { ...quizData2 },
                { ...privateQuizData1 },
                { ...privateQuizData2 }
            ]
        });

        // Create fresh test stats data
        await DbProvider.get().stats_quiz.createMany({
            data: [
                statsQuizData1,
                statsQuizData2,
                privateQuizStats1,
                privateQuizStats2
            ]
        });
    });

    after(async function after() {
        this.timeout(15_000);

        // Clean up test data
        await DbProvider.get().stats_quiz.deleteMany({
            where: { quizId: { in: [testQuizId1, testQuizId2, privateQuizId1, privateQuizId2] } }
        });
        await DbProvider.get().quiz.deleteMany({
            where: { id: { in: [testQuizId1, testQuizId2, privateQuizId1, privateQuizId2] } }
        });
        await DbProvider.get().user.deleteMany({
            where: { id: { in: [user1Id, user2Id] } }
        });

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns stats for public and owned quizzes when logged in", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id }; // User 1 owns privateQuiz1
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsQuizSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly
                };
                const result = await statsQuiz.findMany({ input }, { req, res }, statsQuiz_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                expect(resultIds).to.include(statsQuizData1.id);
                expect(resultIds).to.include(statsQuizData2.id);
                expect(resultIds).to.include(privateQuizStats1.id);
                expect(resultIds).to.not.include(privateQuizStats2.id);
            });

            it("filters by periodType", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsQuizSearchInput = { periodType: StatPeriodType.Monthly };
                const result = await statsQuiz.findMany({ input }, { req, res }, statsQuiz_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);
                expect(resultIds).to.include(statsQuizData1.id);
                expect(resultIds).to.include(statsQuizData2.id);
                expect(resultIds).to.include(privateQuizStats1.id);
            });

            it("filters by time range", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsQuizSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: {
                        after: new Date("2023-01-01"),
                        before: new Date("2023-01-31")
                    }
                };
                const result = await statsQuiz.findMany({ input }, { req, res }, statsQuiz_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);
                expect(resultIds).to.include(statsQuizData1.id);
                expect(resultIds).to.not.include(statsQuizData2.id);
                expect(resultIds).to.not.include(privateQuizStats1.id);
            });

            it("API key - public permissions returns only public quiz stats", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = { [ApiKeyPermission.ReadPublic]: true } as Record<ApiKeyPermission, boolean>;
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: StatsQuizSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly
                };
                const result = await statsQuiz.findMany({ input }, { req, res }, statsQuiz_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                expect(resultIds).to.include(statsQuizData1.id);
                expect(resultIds).to.include(statsQuizData2.id);
                expect(resultIds).to.not.include(privateQuizStats1.id);
                expect(resultIds).to.not.include(privateQuizStats2.id);
            });

            it("not logged in returns only public quiz stats", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: StatsQuizSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly
                };
                // Assuming readManyHelper allows public access for quizzes
                const result = await statsQuiz.findMany({ input }, { req, res }, statsQuiz_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                expect(resultIds).to.include(statsQuizData1.id);
                expect(resultIds).to.include(statsQuizData2.id);
                expect(resultIds).to.not.include(privateQuizStats1.id);
                expect(resultIds).to.not.include(privateQuizStats2.id);
            });
        });

        describe("invalid", () => {
            it("invalid time range format should throw error", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsQuizSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: { after: new Date("invalid"), before: new Date("invalid") }
                };

                try {
                    await statsQuiz.findMany({ input }, { req, res }, statsQuiz_findMany);
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
                    await statsQuiz.findMany({ input: input as StatsQuizSearchInput }, { req, res }, statsQuiz_findMany);
                    expect.fail("Expected an error");
                } catch (error) {
                    expect(error).to.be.instanceOf(Error);
                }
            });

            it("cannot see stats of private quiz you don't own when searching by name", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id }; // User 1
                const { req, res } = await mockAuthenticatedSession(testUser);

                // Search for User 2's private quiz
                const input: StatsQuizSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    searchString: "Private Quiz 2" // Assuming name field exists
                };
                const result = await statsQuiz.findMany({ input }, { req, res }, statsQuiz_findMany);

                expect(result).to.not.be.null;
                expect(result.edges!.length).to.equal(0);
                expect(result.edges!.every(edge => edge?.node?.id !== privateQuizStats2.id)).to.be.true;
            });
        });
    });
}); 