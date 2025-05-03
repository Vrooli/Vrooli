// Tests for the Bookmark endpoint (findOne, findMany, createOne, updateOne)
import { BookmarkCreateInput, BookmarkFor, BookmarkSearchInput, BookmarkUpdateInput, DUMMY_ID, FindByIdInput, generatePK, generatePublicId, validatePK } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions, seedMockAdminUser } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { bookmark_createOne } from "../generated/bookmark_createOne.js";
import { bookmark_findMany } from "../generated/bookmark_findMany.js";
import { bookmark_findOne } from "../generated/bookmark_findOne.js";
import { bookmark_updateOne } from "../generated/bookmark_updateOne.js";
import { bookmark } from "./bookmark.js";

describe("EndpointsBookmark", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;
    // Test users
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    let adminId: bigint;
    let user1Id: bigint;
    let user2Id: bigint;
    // Bookmark lists for tests
    let listUser1: any;
    let listUser2: any;
    // Bookmark IDs for tests
    let bookmarkUser1Id: bigint;
    let bookmarkUser2Id: bigint;
    // Tags for tests
    let tagUser1: any;
    let tagUser2: any;

    before(() => {
        // Stub logger methods to suppress output
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Create users for testing
        user1Id = generatePK();
        await DbProvider.get().user.create({
            data: {
                id: user1Id,
                publicId: generatePublicId(),
                name: "Test User 1",
                handle: "test-user-1",
                status: "Unlocked",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
                auths: { create: [{ id: generatePK(), provider: "Password", hashed_password: "dummy-hash" }] },
            },
        });
        user2Id = generatePK();
        await DbProvider.get().user.create({
            data: {
                id: user2Id,
                publicId: generatePublicId(),
                name: "Test User 2",
                handle: "test-user-2",
                status: "Unlocked",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
                auths: { create: [{ id: generatePK(), provider: "Password", hashed_password: "dummy-hash" }] },
            },
        });
        const admin = await seedMockAdminUser();
        adminId = admin.id;

        // Create bookmark lists for each user
        listUser1 = await DbProvider.get().bookmark_list.create({
            data: {
                id: generatePK(),
                label: "List1",
                user: { connect: { id: user1Id } },
            },
        });
        listUser2 = await DbProvider.get().bookmark_list.create({
            data: {
                id: generatePK(),
                label: "List2",
                user: { connect: { id: user2Id } },
            },
        });

        // Create tags to be bookmarked
        tagUser1 = await DbProvider.get().tag.create({ data: { id: generatePK(), tag: "tag-1" } });
        tagUser2 = await DbProvider.get().tag.create({ data: { id: generatePK(), tag: "tag-2" } });

        // Seed bookmarks for each user
        bookmarkUser1Id = generatePK();
        await DbProvider.get().bookmark.create({
            data: {
                id: bookmarkUser1Id,
                list: { connect: { id: listUser1.id } },
                user: { connect: { id: user1Id } },
                tag: { connect: { id: tagUser1.id } },
            },
        });
        bookmarkUser2Id = generatePK();
        await DbProvider.get().bookmark.create({
            data: {
                id: bookmarkUser2Id,
                list: { connect: { id: listUser2.id } },
                user: { connect: { id: user2Id } },
                tag: { connect: { id: tagUser2.id } },
            },
        });
    });

    after(async () => {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Restore logger stubs
        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns own bookmark", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: FindByIdInput = { id: bookmarkUser1Id.toString() };
                const result = await bookmark.findOne({ input }, { req, res }, bookmark_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(bookmarkUser1Id);
            });

            it("API key with read permissions can find own bookmark", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: user1Id });
                const input: FindByIdInput = { id: bookmarkUser1Id.toString() };
                const result = await bookmark.findOne({ input }, { req, res }, bookmark_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(bookmarkUser1Id);
            });
        });

        describe("invalid", () => {
            it("cannot find another user's bookmark", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: FindByIdInput = { id: bookmarkUser2Id.toString() };
                try {
                    await bookmark.findOne({ input }, { req, res }, bookmark_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* Error expected */ }
            });

            it("not logged in user cannot find bookmark", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: bookmarkUser1Id.toString() };
                try {
                    await bookmark.findOne({ input }, { req, res }, bookmark_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* Error expected */ }
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns own bookmarks for authenticated user", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: BookmarkSearchInput = { take: 10 };
                const result = await bookmark.findMany({ input }, { req, res }, bookmark_findMany);
                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const ids = result.edges!.map(edge => edge!.node!.id);
                expect(ids).to.deep.equal([bookmarkUser1Id]);
            });
        });

        describe("invalid", () => {
            it("logged out user cannot find any bookmarks", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: BookmarkSearchInput = { take: 10 };
                try {
                    await bookmark.findMany({ input }, { req, res }, bookmark_findMany);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* Error expected */ }
            });
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates a bookmark for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: BookmarkCreateInput = {
                    id: DUMMY_ID,
                    bookmarkFor: BookmarkFor.Tag,
                    forConnect: tagUser1.id,
                    listConnect: listUser1.id,
                };
                const result = await bookmark.createOne({ input }, { req, res }, bookmark_createOne);
                expect(result).to.not.be.null;
                expect(validatePK(result.id)).to.be.true;
            });

            it("API key with write permissions can create bookmark", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: user1Id });
                const input: BookmarkCreateInput = {
                    id: DUMMY_ID,
                    bookmarkFor: BookmarkFor.Tag,
                    forConnect: tagUser1.id,
                    listConnect: listUser1.id,
                };
                const result = await bookmark.createOne({ input }, { req, res }, bookmark_createOne);
                expect(result).to.not.be.null;
                expect(validatePK(result.id)).to.be.true;
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create bookmark", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: BookmarkCreateInput = {
                    id: DUMMY_ID,
                    bookmarkFor: BookmarkFor.Tag,
                    forConnect: tagUser1.id,
                    listConnect: listUser1.id,
                };
                try {
                    await bookmark.createOne({ input }, { req, res }, bookmark_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* Error expected */ }
            });

            it("cannot create bookmark under another user's list", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: BookmarkCreateInput = {
                    id: DUMMY_ID,
                    bookmarkFor: BookmarkFor.Tag,
                    forConnect: tagUser2.id,
                    listConnect: listUser2.id,
                };
                try {
                    await bookmark.createOne({ input }, { req, res }, bookmark_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* Error expected */ }
            });

            it("API key without write permissions cannot create bookmark", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: user1Id });
                const input: BookmarkCreateInput = {
                    id: DUMMY_ID,
                    bookmarkFor: BookmarkFor.Tag,
                    forConnect: tagUser1.id,
                    listConnect: listUser1.id,
                };
                try {
                    await bookmark.createOne({ input }, { req, res }, bookmark_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* Error expected */ }
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("updates own bookmark", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: BookmarkUpdateInput = { id: bookmarkUser1Id.toString(), listConnect: listUser1.id.toString() };
                const result = await bookmark.updateOne({ input }, { req, res }, bookmark_updateOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(bookmarkUser1Id);
            });

            it("API key with write permissions can update own bookmark", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: user1Id });
                const input: BookmarkUpdateInput = { id: bookmarkUser1Id.toString(), listConnect: listUser1.id.toString() };
                const result = await bookmark.updateOne({ input }, { req, res }, bookmark_updateOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(bookmarkUser1Id);
            });
        });

        describe("invalid", () => {
            it("cannot update another user's bookmark", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: BookmarkUpdateInput = { id: bookmarkUser2Id.toString(), listConnect: listUser2.id.toString() };
                try {
                    await bookmark.updateOne({ input }, { req, res }, bookmark_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* Error expected */ }
            });

            it("not logged in user cannot update bookmark", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: BookmarkUpdateInput = { id: bookmarkUser1Id.toString(), listConnect: listUser1.id.toString() };
                try {
                    await bookmark.updateOne({ input }, { req, res }, bookmark_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* Error expected */ }
            });
        });
    });
});  
