import { ApiKeyPermission, StatPeriodType, StatsCodeSearchInput, uuid } from "@local/shared";
import { code as CodeModelPrisma, PeriodType } from "@prisma/client"; // Use lowercase 'code' and alias
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { statsCode_findMany } from "../generated/statsCode_findMany.js"; // Assuming this generated type exists
import { statsCode } from "./statsCode.js";

// Test data
const testCodeId1 = uuid();
const testCodeId2 = uuid();
const privateCodeId1 = uuid(); // Private Code owned by user1
const privateCodeId2 = uuid(); // Private Code owned by user2

// User IDs for ownership testing
const user1Id = uuid();
const user2Id = uuid();

// Sample Code data structure (adjust based on actual Code model)
const codeData1: Partial<CodeModelPrisma> & { id: string } = {
    id: testCodeId1,
    isPrivate: false,
    ownedByUserId: null, // Placeholder: Assuming null means public
    // Add other required Code fields (e.g., name, language, versions)
};

const codeData2: Partial<CodeModelPrisma> & { id: string } = {
    id: testCodeId2,
    isPrivate: false,
    ownedByUserId: null,
    // Add other required Code fields
};

const privateCodeData1: Partial<CodeModelPrisma> & { id: string } = {
    id: privateCodeId1,
    isPrivate: true,
    ownedByUserId: user1Id, // Placeholder: Owned by user1
    // Add other required Code fields
};

const privateCodeData2: Partial<CodeModelPrisma> & { id: string } = {
    id: privateCodeId2,
    isPrivate: true,
    ownedByUserId: user2Id, // Placeholder: Owned by user2
    // Add other required Code fields
};

const statsCodeData1 = {
    id: uuid(),
    codeId: testCodeId1,
    periodStart: new Date("2023-01-01"),
    periodEnd: new Date("2023-01-31"),
    periodType: PeriodType.Monthly,
    // Add other relevant StatsCode fields (e.g., linesAdded, linesRemoved, commits)
    calls: 100,
    routineVersions: 5, // Assuming similar fields to statsApi
};

const statsCodeData2 = {
    id: uuid(),
    codeId: testCodeId2,
    periodStart: new Date("2023-02-01"),
    periodEnd: new Date("2023-02-28"),
    periodType: PeriodType.Monthly,
    calls: 200,
    routineVersions: 10,
};

const privateCodeStats1 = {
    id: uuid(),
    codeId: privateCodeId1,
    periodStart: new Date("2023-03-01"),
    periodEnd: new Date("2023-03-31"),
    periodType: PeriodType.Monthly,
    calls: 50,
    routineVersions: 3,
};

const privateCodeStats2 = {
    id: uuid(),
    codeId: privateCodeId2,
    periodStart: new Date("2023-03-01"),
    periodEnd: new Date("2023-03-31"),
    periodType: PeriodType.Monthly,
    calls: 75,
    routineVersions: 4,
};

describe("EndpointsStatsCode", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async function beforeEach() {
        // Clear databases
        await (await initializeRedis())?.flushAll();
        await DbProvider.get().stats_code.deleteMany({});
        await DbProvider.get().code.deleteMany({});
        await DbProvider.get().user.deleteMany({});

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

        // Create test code snippets (ensure all required fields are present)
        // Placeholder: Assuming code needs a name and potentially versions/language
        await DbProvider.get().code.createMany({
            data: [
                { ...codeData1 /* language: "typescript", versions: { create: [...] } */ },
                { ...codeData2 /* language: "python", versions: { create: [...] } */ },
                { ...privateCodeData1 /* language: "typescript", versions: { create: [...] } */ },
                { ...privateCodeData2 /* language: "python", versions: { create: [...] } */ },
            ].map(c => ({ // Adjust ownership fields and add required permissions
                ...c,
                ownedByUserId: c.ownedByUserId ?? undefined,
                permissions: c.permissions ?? JSON.stringify({}), // Add default permissions
            })),
        });

        // Create fresh test stats data
        await DbProvider.get().stats_code.createMany({
            data: [
                statsCodeData1,
                statsCodeData2,
                privateCodeStats1,
                privateCodeStats2,
            ],
        });
    });

    after(async function after() {
        // Clear databases
        await (await initializeRedis())?.flushAll();
        await DbProvider.get().stats_code.deleteMany({});
        await DbProvider.get().code.deleteMany({});
        await DbProvider.get().user.deleteMany({});

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns stats for public and owned code when logged in", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id }; // User 1 owns privateCode1
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsCodeSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsCode.findMany({ input }, { req, res }, statsCode_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                expect(resultIds).to.include(statsCodeData1.id);
                expect(resultIds).to.include(statsCodeData2.id);
                expect(resultIds).to.include(privateCodeStats1.id);
                expect(resultIds).to.not.include(privateCodeStats2.id);
            });

            it("filters by periodType", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsCodeSearchInput = { periodType: StatPeriodType.Monthly };
                const result = await statsCode.findMany({ input }, { req, res }, statsCode_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);
                expect(resultIds).to.include(statsCodeData1.id);
                expect(resultIds).to.include(statsCodeData2.id);
                expect(resultIds).to.include(privateCodeStats1.id);
            });

            it("filters by time range", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsCodeSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: {
                        after: new Date("2023-01-01"),
                        before: new Date("2023-01-31"),
                    },
                };
                const result = await statsCode.findMany({ input }, { req, res }, statsCode_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);
                expect(resultIds).to.include(statsCodeData1.id);
                expect(resultIds).to.not.include(statsCodeData2.id);
                expect(resultIds).to.not.include(privateCodeStats1.id);
            });

            it("API key - public permissions returns only public code stats", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = { [ApiKeyPermission.ReadPublic]: true } as Record<ApiKeyPermission, boolean>;
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: StatsCodeSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsCode.findMany({ input }, { req, res }, statsCode_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                expect(resultIds).to.include(statsCodeData1.id);
                expect(resultIds).to.include(statsCodeData2.id);
                expect(resultIds).to.not.include(privateCodeStats1.id);
                expect(resultIds).to.not.include(privateCodeStats2.id);
            });

            it("not logged in returns only public code stats", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: StatsCodeSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                // Assuming readManyHelper allows public access for code when not logged in
                const result = await statsCode.findMany({ input }, { req, res }, statsCode_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                expect(resultIds).to.include(statsCodeData1.id);
                expect(resultIds).to.include(statsCodeData2.id);
                expect(resultIds).to.not.include(privateCodeStats1.id);
                expect(resultIds).to.not.include(privateCodeStats2.id);
            });
        });

        describe("invalid", () => {
            it("invalid time range format should throw error", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsCodeSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: { after: new Date("invalid"), before: new Date("invalid") },
                };

                try {
                    await statsCode.findMany({ input }, { req, res }, statsCode_findMany);
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
                    await statsCode.findMany({ input: input as StatsCodeSearchInput }, { req, res }, statsCode_findMany);
                    expect.fail("Expected an error");
                } catch (error) {
                    expect(error).to.be.instanceOf(Error);
                }
            });

            it("cannot see stats of private code you don't own when searching by name", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id }; // User 1
                const { req, res } = await mockAuthenticatedSession(testUser);

                // Search for User 2's private code
                const input: StatsCodeSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    searchString: "Private Code 2", // Assuming name field exists
                };
                const result = await statsCode.findMany({ input }, { req, res }, statsCode_findMany);

                expect(result).to.not.be.null;
                expect(result.edges!.length).to.equal(0);
                expect(result.edges!.every(edge => edge?.node?.id !== privateCodeStats2.id)).to.be.true;
            });
        });
    });
}); 
