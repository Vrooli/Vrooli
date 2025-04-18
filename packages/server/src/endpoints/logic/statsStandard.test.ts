import { ApiKeyPermission, StatPeriodType, StatsStandardSearchInput, uuid } from "@local/shared";
import { PeriodType, standard as StandardModelPrisma } from "@prisma/client"; // Assuming standard model exists
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { statsStandard_findMany } from "../generated/statsStandard_findMany.js"; // Assuming this generated type exists
import { statsStandard } from "./statsStandard.js";

// Test data
const testStandardId1 = uuid();
const testStandardId2 = uuid();
const privateStandardId1 = uuid(); // Private Standard owned by user1
const privateStandardId2 = uuid(); // Private Standard owned by user2

// User IDs for ownership testing
const user1Id = uuid();
const user2Id = uuid();

// Sample Standard data structure (adjust based on actual Standard model)
const standardData1: Partial<StandardModelPrisma> & { id: string } = {
    id: testStandardId1,
    isPrivate: false,
    ownedByUserId: null, // Placeholder: Assuming null means public
    // Add other required Standard fields (e.g., name, description, versions)
};

const standardData2: Partial<StandardModelPrisma> & { id: string } = {
    id: testStandardId2,
    isPrivate: false,
    ownedByUserId: null,
    // Add other required Standard fields
};

const privateStandardData1: Partial<StandardModelPrisma> & { id: string } = {
    id: privateStandardId1,
    isPrivate: true,
    ownedByUserId: user1Id, // Placeholder: Owned by user1
    // Add other required Standard fields
};

const privateStandardData2: Partial<StandardModelPrisma> & { id: string } = {
    id: privateStandardId2,
    isPrivate: true,
    ownedByUserId: user2Id, // Placeholder: Owned by user2
    // Add other required Standard fields
};

// Adjust fields based on actual StatsStandard model
const statsStandardData1 = {
    id: uuid(),
    standardId: testStandardId1,
    periodStart: new Date("2023-01-01"),
    periodEnd: new Date("2023-01-31"),
    periodType: PeriodType.Monthly,
    linksToInputs: 0, // Added required field
    linksToOutputs: 0, // Added required field
};

const statsStandardData2 = {
    id: uuid(),
    standardId: testStandardId2,
    periodStart: new Date("2023-02-01"),
    periodEnd: new Date("2023-02-28"),
    periodType: PeriodType.Monthly,
    linksToInputs: 0, // Added required field
    linksToOutputs: 0, // Added required field
};

const privateStandardStats1 = {
    id: uuid(),
    standardId: privateStandardId1,
    periodStart: new Date("2023-03-01"),
    periodEnd: new Date("2023-03-31"),
    periodType: PeriodType.Monthly,
    linksToInputs: 0, // Added required field
    linksToOutputs: 0, // Added required field
};

const privateStandardStats2 = {
    id: uuid(),
    standardId: privateStandardId2,
    periodStart: new Date("2023-03-01"),
    periodEnd: new Date("2023-03-31"),
    periodType: PeriodType.Monthly,
    linksToInputs: 0, // Added required field
    linksToOutputs: 0, // Added required field
};

describe("EndpointsStatsStandard", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async function beforeEach() {
        // Clear databases
        await (await initializeRedis())?.flushAll();
        await DbProvider.get().stats_standard.deleteMany({});
        await DbProvider.get().standard.deleteMany({});
        await DbProvider.get().user.deleteMany({});

        // Create test users
        await DbProvider.get().user.create({
            data: { id: user1Id, name: "Test User 1", handle: "test-user-1", status: "Unlocked", isBot: false, isBotDepictingPerson: false, isPrivate: false, auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] } },
        });
        await DbProvider.get().user.create({
            data: { id: user2Id, name: "Test User 2", handle: "test-user-2", status: "Unlocked", isBot: false, isBotDepictingPerson: false, isPrivate: false, auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] } },
        });

        // Create test standards (ensure all required fields are present)
        // Placeholder: Assuming standards need a name and permissions
        await DbProvider.get().standard.createMany({
            data: [
                { ...standardData1, permissions: JSON.stringify({}) },
                { ...standardData2, permissions: JSON.stringify({}) },
                { ...privateStandardData1, permissions: JSON.stringify({}) },
                { ...privateStandardData2, permissions: JSON.stringify({}) },
            ].map(s => ({ // Adjust ownership fields based on actual model
                ...s,
                ownedByUserId: s.ownedByUserId ?? undefined,
            })),
        });

        // Create fresh test stats data
        await DbProvider.get().stats_standard.createMany({
            data: [
                statsStandardData1,
                statsStandardData2,
                privateStandardStats1,
                privateStandardStats2,
            ],
        });
    });

    after(async function after() {
        // Clear databases
        await (await initializeRedis())?.flushAll();
        await DbProvider.get().stats_standard.deleteMany({});
        await DbProvider.get().standard.deleteMany({});
        await DbProvider.get().user.deleteMany({});

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns stats for public and owned standards when logged in", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id }; // User 1 owns privateStandard1
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsStandardSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsStandard.findMany({ input }, { req, res }, statsStandard_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                expect(resultIds).to.include(statsStandardData1.id);
                expect(resultIds).to.include(statsStandardData2.id);
                expect(resultIds).to.include(privateStandardStats1.id);
                expect(resultIds).to.not.include(privateStandardStats2.id);
            });

            it("filters by periodType", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsStandardSearchInput = { periodType: StatPeriodType.Monthly };
                const result = await statsStandard.findMany({ input }, { req, res }, statsStandard_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);
                expect(resultIds).to.include(statsStandardData1.id);
                expect(resultIds).to.include(statsStandardData2.id);
                expect(resultIds).to.include(privateStandardStats1.id);
            });

            it("filters by time range", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsStandardSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: {
                        after: new Date("2023-01-01"),
                        before: new Date("2023-01-31"),
                    },
                };
                const result = await statsStandard.findMany({ input }, { req, res }, statsStandard_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);
                expect(resultIds).to.include(statsStandardData1.id);
                expect(resultIds).to.not.include(statsStandardData2.id);
                expect(resultIds).to.not.include(privateStandardStats1.id);
            });

            it("API key - public permissions returns only public standard stats", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = { [ApiKeyPermission.ReadPublic]: true } as Record<ApiKeyPermission, boolean>;
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: StatsStandardSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsStandard.findMany({ input }, { req, res }, statsStandard_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                expect(resultIds).to.include(statsStandardData1.id);
                expect(resultIds).to.include(statsStandardData2.id);
                expect(resultIds).to.not.include(privateStandardStats1.id);
                expect(resultIds).to.not.include(privateStandardStats2.id);
            });

            it("not logged in returns only public standard stats", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: StatsStandardSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                // Assuming readManyHelper allows public access for standards
                const result = await statsStandard.findMany({ input }, { req, res }, statsStandard_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                expect(resultIds).to.include(statsStandardData1.id);
                expect(resultIds).to.include(statsStandardData2.id);
                expect(resultIds).to.not.include(privateStandardStats1.id);
                expect(resultIds).to.not.include(privateStandardStats2.id);
            });
        });

        describe("invalid", () => {
            it("invalid time range format should throw error", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsStandardSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: { after: new Date("invalid"), before: new Date("invalid") },
                };

                try {
                    await statsStandard.findMany({ input }, { req, res }, statsStandard_findMany);
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
                    await statsStandard.findMany({ input: input as StatsStandardSearchInput }, { req, res }, statsStandard_findMany);
                    expect.fail("Expected an error");
                } catch (error) {
                    expect(error).to.be.instanceOf(Error);
                }
            });

            it("cannot see stats of private standard you don't own when searching by name", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id }; // User 1
                const { req, res } = await mockAuthenticatedSession(testUser);

                // Search for User 2's private standard
                const input: StatsStandardSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    searchString: "Private Standard 2", // Assuming name field exists
                };
                const result = await statsStandard.findMany({ input }, { req, res }, statsStandard_findMany);

                expect(result).to.not.be.null;
                expect(result.edges!.length).to.equal(0);
                expect(result.edges!.every(edge => edge?.node?.id !== privateStandardStats2.id)).to.be.true;
            });
        });
    });
}); 
