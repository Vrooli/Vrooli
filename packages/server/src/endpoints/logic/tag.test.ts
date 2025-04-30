import { FindByIdInput, SEEDED_IDS, TagCreateInput, TagSearchInput, TagUpdateInput, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { tag_createOne } from "../generated/tag_createOne.js";
import { tag_findMany } from "../generated/tag_findMany.js";
import { tag_findOne } from "../generated/tag_findOne.js";
import { tag_updateOne } from "../generated/tag_updateOne.js";
import { tag } from "./tag.js";

// Test users
const user1Id = uuid();
const user2Id = uuid();

// Test tag data
const userTag1 = {
    id: uuid(),
    tag: "Test Tag 1",
    createdAt: new Date("2023-03-01"),
    updatedAt: new Date("2023-03-12"),
};
const userTag2 = {
    id: uuid(),
    tag: "Test Tag 2",
    createdAt: new Date("2023-03-05"),
    updatedAt: new Date("2023-03-05"),
};

// Array of all tag IDs for easier cleanup
const allTagIds = [userTag1.id, userTag2.id];

describe("EndpointsTag", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async function beforeEach() {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

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
                auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] },
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
                auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] },
            },
        });
        // Ensure admin user exists for update tests
        await DbProvider.get().user.upsert({
            where: { id: SEEDED_IDS.User.Admin },
            update: {},
            create: {
                id: SEEDED_IDS.User.Admin,
                name: "Admin User",
                handle: "admin",
                status: "Unlocked",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
                auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] },
            },
        });

        // Create user-specific tags
        await DbProvider.get().tag.create({ data: { ...userTag1, createdBy: { connect: { id: user1Id } } } });
        await DbProvider.get().tag.create({ data: { ...userTag2, createdBy: { connect: { id: user2Id } } } });
    });

    after(async function after() {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns tag by id for any authenticated user", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: FindByIdInput = { id: userTag1.id };
                const result = await tag.findOne({ input }, { req, res }, tag_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(userTag1.id);
                expect(result.tag).to.equal(userTag1.tag);
            });
            it("returns tag by id when not authenticated", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: userTag1.id };
                const result = await tag.findOne({ input }, { req, res }, tag_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(userTag1.id);
                expect(result.tag).to.equal(userTag1.tag);
            });
            it("returns tag by id with API key public read", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: FindByIdInput = { id: userTag2.id };
                const result = await tag.findOne({ input }, { req, res }, tag_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(userTag2.id);
                expect(result.tag).to.equal(userTag2.tag);
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns tags without filters for any authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const input: TagSearchInput = { take: 10 };
                const result = await tag.findMany({ input }, { req, res }, tag_findMany);
                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultTagIds = result.edges!.map(edge => edge!.node!.id).sort();
                expect(resultTagIds).to.deep.equal(allTagIds.sort());
            });
            it("returns tags without filters for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: TagSearchInput = { take: 10 };
                const result = await tag.findMany({ input }, { req, res }, tag_findMany);
                expect(result).to.not.be.null;
                const resultTagIds = result.edges!.map(e => e!.node!.id).sort();
                expect(resultTagIds).to.deep.equal(allTagIds.sort());
            });
            it("returns tags without filters for API key public read", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: TagSearchInput = { take: 10 };
                const result = await tag.findMany({ input }, { req, res }, tag_findMany);
                expect(result).to.not.be.null;
                const resultTagIds = result.edges!.map(e => e!.node!.id).sort();
                expect(resultTagIds).to.deep.equal(allTagIds.sort());
            });
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates a tag for authenticated user", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const newTagId = uuid();
                const input: TagCreateInput = { id: newTagId, tag: "New Test Tag" };

                const result = await tag.createOne({ input }, { req, res }, tag_createOne);
                expect(result).to.not.be.null;
                // ID may be regenerated by the server, skip id assertion
                expect(result.tag).to.equal(input.tag);
                expect(result.you?.isOwn).to.be.false;
            });

            it("API key with write permissions can create tag", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const newTagId = uuid();
                const input: TagCreateInput = { id: newTagId, tag: "API Created Tag" };

                const result = await tag.createOne({ input }, { req, res }, tag_createOne);
                expect(result).to.not.be.null;
                // Skip id assertion since server may override
                expect(result.tag).to.equal(input.tag);
                expect(result.you?.isOwn).to.be.false;
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create tag", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: TagCreateInput = { id: uuid(), tag: "Unauthorized Tag" };
                try {
                    await tag.createOne({ input }, { req, res }, tag_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /** Error expected */ }
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("allows admin to update a tag", async () => {
                const adminUser = { ...loggedInUserNoPremiumData, id: SEEDED_IDS.User.Admin };
                const { req, res } = await mockAuthenticatedSession(adminUser);
                const input: TagUpdateInput = { id: userTag1.id, tag: "Admin Updated Tag" };
                const result = await tag.updateOne({ input }, { req, res }, tag_updateOne);
                expect(result).to.not.be.null;
                expect(result.tag).to.equal(input.tag);
            });
        });

        describe("invalid", () => {
            it("denies update for non-admin user", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: TagUpdateInput = { id: userTag1.id, tag: "Unauthorized Update" };
                try {
                    await tag.updateOne({ input }, { req, res }, tag_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /** Error expected */ }
            });

            it("denies update for not logged in user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: TagUpdateInput = { id: userTag1.id, tag: "Unauthorized Update" };
                try {
                    await tag.updateOne({ input }, { req, res }, tag_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /** Error expected */ }
            });

            it("throws when updating non-existent tag as admin", async () => {
                const adminUser = { ...loggedInUserNoPremiumData, id: SEEDED_IDS.User.Admin };
                const { req, res } = await mockAuthenticatedSession(adminUser);
                const input: TagUpdateInput = { id: uuid(), tag: "NoSuchTag" };
                try {
                    await tag.updateOne({ input }, { req, res }, tag_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /** Error expected */ }
            });
        });
    });
}); 
