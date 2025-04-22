// Tests for the Label endpoint (findOne, findMany, createOne, updateOne)
import { ApiKeyPermission, FindByIdInput, LabelCreateInput, LabelSearchInput, LabelUpdateInput, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { label_createOne } from "../generated/label_createOne.js";
import { label_findMany } from "../generated/label_findMany.js";
import { label_findOne } from "../generated/label_findOne.js";
import { label_updateOne } from "../generated/label_updateOne.js";
import { label } from "./label.js";

// Generate two test users
const user1Id = uuid();
const user2Id = uuid();

// Generate IDs for seeded labels
const labelUser1Id = uuid();
const labelUser2Id = uuid();

// Hold references to seeded label records
let labelUser1: any;
let labelUser2: any;

describe("EndpointsLabel", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        // Stub logger to suppress console output during tests
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async function beforeEach() {
        // Clear Redis cache and truncate relevant tables
        await (await initializeRedis())?.flushAll();
        await DbProvider.get().session.deleteMany({});
        await DbProvider.get().user_auth.deleteMany({});
        await DbProvider.get().user.deleteMany({});
        await DbProvider.get().label_translation.deleteMany({});
        await DbProvider.get().label.deleteMany({});

        // Create two users for ownership tests
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

        // Seed two labels owned by each user
        labelUser1 = await DbProvider.get().label.create({
            data: {
                id: labelUser1Id,
                label: "Label One",
                color: "#FF0000",
                ownedByUser: { connect: { id: user1Id } },
            },
        });
        labelUser2 = await DbProvider.get().label.create({
            data: {
                id: labelUser2Id,
                label: "Label Two",
                color: "#00FF00",
                ownedByUser: { connect: { id: user2Id } },
            },
        });
    });

    after(async () => {
        // Clean up database and restore logger stubs
        await (await initializeRedis())?.flushAll();
        await DbProvider.get().label_translation.deleteMany({});
        await DbProvider.get().label.deleteMany({});
        await DbProvider.get().user.deleteMany({});
        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns label by id for any authenticated user", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: FindByIdInput = { id: labelUser1Id };
                const result = await label.findOne({ input }, { req, res }, label_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(labelUser1Id);
                expect(result.label).to.equal("Label One");
            });

            it("returns label by id when not authenticated", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: labelUser1Id };
                const result = await label.findOne({ input }, { req, res }, label_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(labelUser1Id);
            });

            it("returns label by id with API key public read", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = { [ApiKeyPermission.ReadPublic]: true } as Record<ApiKeyPermission, boolean>;
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: FindByIdInput = { id: labelUser2Id };
                const result = await label.findOne({ input }, { req, res }, label_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(labelUser2Id);
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns all labels without filters for any authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const input: LabelSearchInput = { take: 10 };
                const result = await label.findMany({ input }, { req, res }, label_findMany);
                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const ids = result.edges!.map(e => e!.node!.id).sort();
                expect(ids).to.deep.equal([labelUser1Id, labelUser2Id].sort());
            });

            it("returns all labels without filters for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: LabelSearchInput = { take: 10 };
                const result = await label.findMany({ input }, { req, res }, label_findMany);
                expect(result).to.not.be.null;
                const ids = result.edges!.map(e => e!.node!.id).sort();
                expect(ids).to.deep.equal([labelUser1Id, labelUser2Id].sort());
            });

            it("returns all labels without filters for API key public read", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = { [ApiKeyPermission.ReadPublic]: true } as Record<ApiKeyPermission, boolean>;
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: LabelSearchInput = { take: 10 };
                const result = await label.findMany({ input }, { req, res }, label_findMany);
                expect(result).to.not.be.null;
                const ids = result.edges!.map(e => e!.node!.id).sort();
                expect(ids).to.deep.equal([labelUser1Id, labelUser2Id].sort());
            });
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates a label for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const newLabelId = uuid();
                const input: LabelCreateInput = { id: newLabelId, label: "New Label", color: "#ABCDEF" };
                const result = await label.createOne({ input }, { req, res }, label_createOne);
                expect(result).to.not.be.null;
                expect(result.label).to.equal(input.label);
            });

            it("API key with write permissions can create label", async () => {
                const permissions = { [ApiKeyPermission.WritePrivate]: true } as Record<ApiKeyPermission, boolean>;
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData, id: user1Id });
                const newLabelId = uuid();
                const input: LabelCreateInput = { id: newLabelId, label: "API Label", color: "#123456" };
                const result = await label.createOne({ input }, { req, res }, label_createOne);
                expect(result).to.not.be.null;
                expect(result.label).to.equal(input.label);
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create label", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: LabelCreateInput = { id: uuid(), label: "Bad Label" };
                try {
                    await label.createOne({ input }, { req, res }, label_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* expected error */ }
            });

            it("API key without write permissions cannot create label", async () => {
                const permissions = { [ApiKeyPermission.ReadPublic]: true } as Record<ApiKeyPermission, boolean>;
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData, id: user1Id });
                const input: LabelCreateInput = { id: uuid(), label: "NoPerm Label" };
                try {
                    await label.createOne({ input }, { req, res }, label_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* expected error */ }
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("updates own label", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const input: LabelUpdateInput = { id: labelUser1Id, label: "Updated Label" };
                const result = await label.updateOne({ input }, { req, res }, label_updateOne);
                expect(result).to.not.be.null;
                expect(result.label).to.equal(input.label);
            });

            it("API key with write permissions can update own label", async () => {
                const permissions = { [ApiKeyPermission.WritePrivate]: true } as Record<ApiKeyPermission, boolean>;
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData, id: user1Id });
                const input: LabelUpdateInput = { id: labelUser1Id, label: "API Updated" };
                const result = await label.updateOne({ input }, { req, res }, label_updateOne);
                expect(result).to.not.be.null;
                expect(result.label).to.equal(input.label);
            });
        });

        describe("invalid", () => {
            it("cannot update another user's label", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const input: LabelUpdateInput = { id: labelUser2Id, label: "Hack Update" };
                try {
                    await label.updateOne({ input }, { req, res }, label_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* expected error */ }
            });

            it("not logged in user cannot update label", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: LabelUpdateInput = { id: labelUser1Id, label: "NoAuth Update" };
                try {
                    await label.updateOne({ input }, { req, res }, label_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* expected error */ }
            });
        });
    });
}); 
