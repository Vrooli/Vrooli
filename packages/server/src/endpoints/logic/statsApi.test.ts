import { StatPeriodType, uuid } from "@local/shared";
import { PeriodType } from "@prisma/client";
import { expect } from "chai";
import { after, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { statsApi_findMany } from "../generated/statsApi_findMany.js";
import { statsApi } from "./statsApi.js";

// Test data
const testApiId1 = uuid();
const testApiId2 = uuid();
const privateApiId1 = uuid(); // Private API owned by user1
const privateApiId2 = uuid(); // Private API owned by user2 

// User IDs for ownership testing
const user1Id = uuid();
const user2Id = uuid();

const statsApiData1 = {
    id: uuid(),
    apiId: testApiId1,
    periodStart: new Date("2023-01-01"),
    periodEnd: new Date("2023-01-31"),
    periodType: PeriodType.Monthly,
    calls: 100,
    routineVersions: 5,
};

const statsApiData2 = {
    id: uuid(),
    apiId: testApiId2,
    periodStart: new Date("2023-02-01"),
    periodEnd: new Date("2023-02-28"),
    periodType: PeriodType.Monthly,
    calls: 200,
    routineVersions: 10,
};

// Stats for private APIs
const privateApiStats1 = {
    id: uuid(),
    apiId: privateApiId1,
    periodStart: new Date("2023-03-01"),
    periodEnd: new Date("2023-03-31"),
    periodType: PeriodType.Monthly,
    calls: 50,
    routineVersions: 3,
};

const privateApiStats2 = {
    id: uuid(),
    apiId: privateApiId2,
    periodStart: new Date("2023-03-01"),
    periodEnd: new Date("2023-03-31"),
    periodType: PeriodType.Monthly,
    calls: 75,
    routineVersions: 4,
};

describe("EndpointsStatsApi", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async function beforeEach() {
        // Clear databases
        await (await initializeRedis())?.flushAll();
        await DbProvider.get().user.deleteMany({});
        await DbProvider.get().stats_api.deleteMany({});
        await DbProvider.get().api.deleteMany({});

        // Create test users
        await DbProvider.get().user.create({
            data: {
                id: user1Id,
                name: "Test User 1",
                handle: "test-user-1",
                status: "Unlocked",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
                auths: {
                    create: [{
                        provider: "Password",
                        hashed_password: "dummy-hash",
                    }],
                },
            },
        });
        await DbProvider.get().user.create({
            data: {
                id: user2Id,
                name: "Test User 2",
                handle: "test-user-2",
                status: "Unlocked",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
                auths: {
                    create: [{
                        provider: "Password",
                        hashed_password: "dummy-hash",
                    }],
                },
            },
        });

        // Create test APIs
        await DbProvider.get().api.deleteMany({
            where: {
                id: { in: [testApiId1, testApiId2, privateApiId1, privateApiId2] },
            },
        });

        await DbProvider.get().api.create({
            data: {
                id: testApiId1,
                isPrivate: false,
                permissions: JSON.stringify({}),
                versions: {
                    create: [
                        {
                            id: uuid(),
                            callLink: "https://test-api-1.com",
                            translations: {
                                create: [
                                    {
                                        id: uuid(),
                                        language: "en",
                                        name: "Test API 1",
                                    },
                                ],
                            },
                        },
                    ],
                },
            },
        });
        await DbProvider.get().api.create({
            data: {
                id: testApiId2,
                isPrivate: false,
                permissions: JSON.stringify({}),
                versions: {
                    create: [
                        {
                            id: uuid(),
                            callLink: "https://test-api-2.com",
                            translations: {
                                create: [
                                    {
                                        id: uuid(),
                                        language: "en",
                                        name: "Test API 2",
                                    },
                                ],
                            },
                        },
                    ],
                },
            },
        });
        await DbProvider.get().api.create({
            data: {
                id: privateApiId1,
                isPrivate: true,
                permissions: JSON.stringify({}),
                ownedByUser: { connect: { id: user1Id } },
                versions: {
                    create: [
                        {
                            id: uuid(),
                            callLink: "https://private-api-1.com",
                            translations: {
                                create: [
                                    {
                                        id: uuid(),
                                        language: "en",
                                        name: "Private API 1",
                                    },
                                ],
                            },
                        },
                    ],
                },
            },
        });
        await DbProvider.get().api.create({
            data: {
                id: privateApiId2,
                isPrivate: true,
                permissions: JSON.stringify({}),
                ownedByUser: { connect: { id: user2Id } },
                versions: {
                    create: [
                        {
                            id: uuid(),
                            callLink: "https://private-api-2.com",
                            translations: {
                                create: [
                                    {
                                        id: uuid(),
                                        language: "en",
                                        name: "Private API 2",
                                    },
                                ],
                            },
                        },
                    ],
                },
            },
        });

        // Delete any existing test stats data
        await DbProvider.get().stats_api.deleteMany({
            where: {
                id: { in: [statsApiData1.id, statsApiData2.id, privateApiStats1.id, privateApiStats2.id] },
            },
        });

        // Create fresh test stats data directly with DbProvider
        await DbProvider.get().stats_api.createMany({
            data: [
                statsApiData1,
                statsApiData2,
                privateApiStats1,
                privateApiStats2,
            ],
        });
    });

    after(async function after() {
        // Clear databases
        await (await initializeRedis())?.flushAll();
        await DbProvider.get().user.deleteMany({});
        await DbProvider.get().stats_api.deleteMany({});
        await DbProvider.get().api.deleteMany({});

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns stats without filters", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                // Need to include the required periodType
                const input = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsApi.findMany({ input }, { req, res }, statsApi_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                expect(result.edges!.length).to.be.at.least(2);
                expect(result.edges!.some(edge => edge?.node?.id === statsApiData1.id)).to.be.true;
                expect(result.edges!.some(edge => edge?.node?.id === statsApiData2.id)).to.be.true;
            });

            it("filters by periodType", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = { periodType: StatPeriodType.Monthly };
                const result = await statsApi.findMany({ input }, { req, res }, statsApi_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                expect(result.edges!.length).to.be.at.least(2);
                expect(result.edges!.some(edge => edge?.node?.id === statsApiData1.id)).to.be.true;
                expect(result.edges!.some(edge => edge?.node?.id === statsApiData2.id)).to.be.true;
            });

            it("filters by time range", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: {
                        after: new Date("2023-01-01"),
                        before: new Date("2023-01-31"),
                    },
                };
                const result = await statsApi.findMany({ input }, { req, res }, statsApi_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                expect(result.edges!.some(edge => edge?.node?.id === statsApiData1.id)).to.be.true;
                expect(result.edges!.every(edge => edge?.node?.id !== statsApiData2.id)).to.be.true;
            });

            it("API key - public permissions", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsApi.findMany({ input }, { req, res }, statsApi_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                expect(result.edges!.length).to.be.at.least(2);
            });

            it("not logged in", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsApi.findMany({ input }, { req, res }, statsApi_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                expect(result.edges!.length).to.be.at.least(2);
            });

            it("returns stats of private API you own", async () => {
                // Create a session for user1 who owns privateApiId1
                const testUser = {
                    ...loggedInUserNoPremiumData,
                    id: user1Id,
                };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = {
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsApi.findMany({ input }, { req, res }, statsApi_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");

                // Should see public stats and their own private API stats
                expect(result.edges!.some(edge => edge?.node?.id === statsApiData1.id)).to.be.true;
                expect(result.edges!.some(edge => edge?.node?.id === statsApiData2.id)).to.be.true;
                expect(result.edges!.some(edge => edge?.node?.id === privateApiStats1.id)).to.be.true;

                // Should NOT see other user's private API stats
                expect(result.edges!.every(edge => edge?.node?.id !== privateApiStats2.id)).to.be.true;
            });
        });

        describe("invalid", () => {
            it("invalid time range format", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: {
                        // Invalid date objects that will cause errors
                        after: new Date("invalid-date"),
                        before: new Date("invalid-date"),
                    },
                };

                try {
                    await statsApi.findMany({ input }, { req, res }, statsApi_findMany);
                    expect.fail("Expected an error to be thrown");
                } catch (error) {
                    // Error expected
                }
            });

            it("invalid periodType", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = {
                    periodType: "InvalidPeriod" as StatPeriodType,
                };

                try {
                    await statsApi.findMany({ input }, { req, res }, statsApi_findMany);
                    expect.fail("Expected an error to be thrown");
                } catch (error) {
                    // Error expected
                }
            });

            it("cannot see stats of private API you don't own when searching by API name", async () => {
                // Create a session for user1 who does NOT own privateApiId2
                const testUser = {
                    ...loggedInUserNoPremiumData,
                    id: user1Id,
                };
                const { req, res } = await mockAuthenticatedSession(testUser);

                // Try to specifically query the other user's private API stats by name
                const input = {
                    periodType: StatPeriodType.Monthly,
                    searchString: "Private API 2", // This should match the name of user2's private API
                };

                const result = await statsApi.findMany({ input }, { req, res }, statsApi_findMany);

                // The stats for the private API should not be included
                expect(result.edges!.every(edge => edge?.node?.id !== privateApiStats2.id)).to.be.true;

                // No results matching the query should be returned
                expect(result.edges!.length).to.equal(0);
            });
        });
    });
}); 
