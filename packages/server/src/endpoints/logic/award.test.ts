import { AwardCategory, generatePK } from "@local/shared";
import { expect } from "chai";
import { after, describe, it } from "mocha";
import sinon from "sinon";
import { assertFindManyResultIds } from "../../__test/helpers.js";
import { defaultPublicUserData, loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { award_findMany } from "../generated/award_findMany.js";
import { award } from "./award.js";

describe("EndpointsAward", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;
    let user1Id: bigint;
    let user2Id: bigint;
    let userAward1: any;
    let userAward2: any;

    before(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async function beforeEach() {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Initialize IDs after snowflake generator setup
        user1Id = generatePK();
        user2Id = generatePK();

        await DbProvider.get().user.create({
            data: {
                ...defaultPublicUserData(),
                id: user1Id,
                name: "Test User 1",
            },
        });
        await DbProvider.get().user.create({
            data: {
                ...defaultPublicUserData(),
                id: user2Id,
                name: "Test User 2",
            },
        });

        // Define award data using initialized IDs
        userAward1 = {
            id: generatePK(),
            category: AwardCategory.RoutineCreate,
            progress: 75,
            createdAt: new Date("2023-03-01"),
            updatedAt: new Date("2023-03-01"),
            tierCompletedAt: null,
            userId: user1Id,
        };

        userAward2 = {
            id: generatePK(),
            category: AwardCategory.ProjectCreate,
            progress: 25,
            createdAt: new Date("2023-03-15"),
            updatedAt: new Date("2023-03-15"),
            tierCompletedAt: null,
            userId: user2Id,
        };

        // Create user-specific awards
        await DbProvider.get().award.create({
            data: userAward1,
        });
        await DbProvider.get().award.create({
            data: userAward2,
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
            it("returns awards without filters", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                // When logged in as user1, should see user1's awards
                const expectedIds = [
                    userAward1.id,   // User1's award
                ];

                const input = { take: 10 };
                const result = await award.findMany({ input }, { req, res }, award_findMany);
                assertFindManyResultIds(expect, result, expectedIds);
            });

            it("filters by updated time frame", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                // For the given time range, should only see awards updated in Feb-Mar that user1 has access to
                const expectedIds = [
                    userAward1.id,   // Updated in March and belongs to user1
                ];

                const input = {
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
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input = {
                    take: 10,
                };
                try {
                    await award.findMany({ input }, { req, res }, award_findMany);
                    expect.fail("Expected an error to be thrown - awards are private");
                } catch (error) { /** Error expected  */ }
            });

            it("not logged in", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input = {
                    take: 10,
                };
                try {
                    await award.findMany({ input }, { req, res }, award_findMany);
                    expect.fail("Expected an error to be thrown - awards are private");
                } catch (error) { /** Error expected  */ }
            });
        });

        describe("invalid", () => {
            it("invalid time range format", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = {
                    updatedTimeFrame: {
                        // Invalid date objects that will cause errors
                        after: new Date("invalid-date"),
                        before: new Date("invalid-date"),
                    },
                };

                try {
                    await award.findMany({ input }, { req, res }, award_findMany);
                    expect.fail("Expected an error to be thrown");
                } catch (error) {
                    // Error expected
                }
            });
        });
    });
}); 
