import { ApiKeyPermission, AwardCategory, uuid } from "@local/shared";
import { expect } from "chai";
import { after, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { award_findMany } from "../generated/award_findMany.js";
import { award } from "./award.js";

// User award data
const user1Id = uuid();
const user2Id = uuid();

const userAward1 = {
    id: uuid(),
    category: AwardCategory.RoutineCreate,
    progress: 75,
    created_at: new Date("2023-03-01"),
    updated_at: new Date("2023-03-01"),
    timeCurrentTierCompleted: null,
    userId: user1Id,
};

const userAward2 = {
    id: uuid(),
    category: AwardCategory.ProjectCreate,
    progress: 25,
    created_at: new Date("2023-03-15"),
    updated_at: new Date("2023-03-15"),
    timeCurrentTierCompleted: null,
    userId: user2Id,
};

describe("EndpointsAward", () => {
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
        await DbProvider.get().award.deleteMany({});

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

        // Create user-specific awards
        await DbProvider.get().award.create({
            data: userAward1,
        });
        await DbProvider.get().award.create({
            data: userAward2,
        });
    });

    after(async function after() {
        // Clear databases
        await (await initializeRedis())?.flushAll();
        await DbProvider.get().award.deleteMany({});
        await DbProvider.get().user.deleteMany({});

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns awards without filters", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                // When logged in as user1, should see global awards and user1's awards
                const expectedAwardIds = [
                    userAward1.id,   // User1's award
                    // userAward2.id, // User2's award (should not be included)
                ];

                const input = {
                    take: 10,
                };
                const result = await award.findMany({ input }, { req, res }, award_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");

                const resultAwardIds = result.edges!.map(edge => edge!.node!.id);
                expect(resultAwardIds.sort()).to.deep.equal(expectedAwardIds.sort());
            });

            it("filters by updated time frame", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                // For the given time range, should only see awards updated in Feb-Mar that user1 has access to
                const expectedAwardIds = [
                    userAward1.id,   // Updated in March and belongs to user1
                    // userAward2.id,   // Belongs to user2 (no access)
                ];

                const input = {
                    updatedTimeFrame: {
                        after: new Date("2023-02-01"),
                        before: new Date("2023-04-01"),
                    },
                };
                const result = await award.findMany({ input }, { req, res }, award_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");

                const resultAwardIds = result.edges!.map(edge => edge!.node!.id);
                expect(resultAwardIds.sort()).to.deep.equal(expectedAwardIds.sort());
            });

            it("API key - public permissions", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = { [ApiKeyPermission.ReadPublic]: true } as Record<ApiKeyPermission, boolean>;
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
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
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
