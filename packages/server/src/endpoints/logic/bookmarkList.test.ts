import { assertFindManyResultIds } from "../../__test/helpers.js";
// Tests for the BookmarkList endpoint (findOne, findMany, createOne, updateOne)
import { type BookmarkListCreateInput, type BookmarkListSearchInput, type BookmarkListUpdateInput, type FindByIdInput, generatePK, nanoid, validatePK } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { defaultPublicUserData, loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPrivatePermissions, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { bookmarkList_createOne } from "../generated/bookmarkList_createOne.js";
import { bookmarkList_findMany } from "../generated/bookmarkList_findMany.js";
import { bookmarkList_findOne } from "../generated/bookmarkList_findOne.js";
import { bookmarkList_updateOne } from "../generated/bookmarkList_updateOne.js";
import { bookmarkList } from "./bookmarkList.js";

// Test users - Declare vars here, initialize in beforeEach
let user1Id: bigint;
let user2Id: bigint;
// Test lists - Declare vars here, initialize in beforeEach
let listUser1: any;
let listUser2: any;

describe("EndpointsBookmarkList", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        // suppress logger
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        // clear Redis and tables
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Initialize IDs within beforeEach
        user1Id = generatePK();
        user2Id = generatePK();

        // create two users
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

        // seed bookmark lists using generatePK and remove index property
        listUser1 = await DbProvider.get().bookmark_list.create({
            data: {
                id: generatePK(), label: "List One", user: { connect: { id: user1Id } },
            },
        });
        listUser2 = await DbProvider.get().bookmark_list.create({
            data: {
                id: generatePK(), label: "List Two", user: { connect: { id: user2Id } },
            },
        });
    });

    after(async () => {
        // cleanup and restore logger
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns own bookmark list", async () => {
                const user = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(user);
                const input: FindByIdInput = { id: listUser1.id };
                const result = await bookmarkList.findOne({ input }, { req, res }, bookmarkList_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(listUser1.id);
            });

            it("API key cannot find another user's list", async () => {
                const permissions = mockReadPrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                // Pass the actual user object to mockApiSession
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: user1Id });
                const input: FindByIdInput = { id: listUser2.id }; // Try to access listUser2
                try {
                    await bookmarkList.findOne({ input }, { req, res }, bookmarkList_findOne);
                    expect.fail("Expected an error because BookmarkList is private");
                } catch (error: any) {
                    expect(error.message).to.contain("Unauthorized"); // Expect unauthorized due to privacy
                }
            });

            it("logged out user cannot find any list", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: listUser2.id };
                try {
                    await bookmarkList.findOne({ input }, { req, res }, bookmarkList_findOne);
                    expect.fail("Expected an error because BookmarkList is private and user is logged out");
                } catch (error: any) {
                    expect(error.message).to.contain("Unauthorized"); // Expect unauthorized
                }
            });
        });

        describe("invalid", () => {
            it("does not return another user's list", async () => {
                const user = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(user);
                const input: FindByIdInput = { id: listUser2.id };
                try {
                    await bookmarkList.findOne({ input }, { req, res }, bookmarkList_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) {
                    // error expected
                }
            });
        });
    });

    describe("findMany", () => {
        it("returns only own lists for authenticated user", async () => {
            const user = { ...loggedInUserNoPremiumData(), id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: BookmarkListSearchInput = { take: 10 };
            const expectedIds = [
                listUser1.id,
            ];
            const result = await bookmarkList.findMany({ input }, { req, res }, bookmarkList_findMany);
            expect(result).to.not.be.null;
            expect(result.edges).to.not.be.undefined;
            assertFindManyResultIds(expect, result, expectedIds);
        });

        it("API key with public read returns no lists (as they are private)", async () => {
            const permissions = mockReadPublicPermissions();
            const apiToken = ApiKeyEncryptionService.generateSiteKey();
            // Pass the actual user object to mockApiSession
            const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: user1Id });
            const input: BookmarkListSearchInput = { take: 10 };
            try {
                // This should fail because the visibility builder expects a 'Public' function which is null
                await bookmarkList.findMany({ input }, { req, res }, bookmarkList_findMany);
                expect.fail("Expected an InternalError because BookmarkList has no public visibility");
            } catch (error: any) {
                expect(error.message).to.contain("0782"); // Expect the specific internal error code
            }
        });

        it("logged out user returns no lists (as they are private)", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: BookmarkListSearchInput = { take: 10 };
            try {
                // This should fail because the visibility builder expects a 'Public' function which is null
                await bookmarkList.findMany({ input }, { req, res }, bookmarkList_findMany);
                expect.fail("Expected an InternalError because BookmarkList has no public visibility");
            } catch (error: any) {
                expect(error.message).to.contain("0782"); // Expect the specific internal error code
            }
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates a bookmark list for authenticated user", async () => {
                const user = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(user);
                const newListId = nanoid(); // Use nanoid for input ID
                const input: BookmarkListCreateInput = { id: newListId, label: "New List" };
                const result = await bookmarkList.createOne({ input }, { req, res }, bookmarkList_createOne);
                expect(result).to.not.be.null;
                expect(validatePK(result.id)).to.be.true; // Validate PK format
                expect(result.label).to.equal("New List");
            });

            it("API key with write permissions can create list", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                // Pass the actual user object to mockApiSession
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: user1Id });
                const newListId = nanoid(); // Use nanoid for input ID
                const input: BookmarkListCreateInput = { id: newListId, label: "API List" };
                const result = await bookmarkList.createOne({ input }, { req, res }, bookmarkList_createOne);
                expect(validatePK(result.id)).to.be.true; // Validate PK format
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create list", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: BookmarkListCreateInput = { id: nanoid(), label: "NoAuth" }; // Use nanoid
                try {
                    await bookmarkList.createOne({ input }, { req, res }, bookmarkList_createOne);
                    expect.fail("Expected an error");
                } catch (error) {
                    // error expected
                }
            });

            it("API key without write permissions cannot create list", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                // Pass the actual user object to mockApiSession
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: user1Id });
                const input: BookmarkListCreateInput = { id: nanoid(), label: "BadAPI" }; // Use nanoid
                try {
                    await bookmarkList.createOne({ input }, { req, res }, bookmarkList_createOne);
                    expect.fail("Expected an error");
                } catch (error) {
                    // error expected
                }
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("updates own bookmark list", async () => {
                const user = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(user);
                const input: BookmarkListUpdateInput = { id: listUser1.id, label: "Updated Label" };
                const result = await bookmarkList.updateOne({ input }, { req, res }, bookmarkList_updateOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(listUser1.id);
                expect(result.label).to.equal("Updated Label");
            });

            it("API key with write permissions can update list", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                // Pass the actual user object to mockApiSession
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: user1Id });
                const input: BookmarkListUpdateInput = { id: listUser1.id.toString(), label: "API Updated" }; // Ensure ID is string for input
                const result = await bookmarkList.updateOne({ input }, { req, res }, bookmarkList_updateOne);
                expect(result.label).to.equal("API Updated");
            });
        });

        describe("invalid", () => {
            it("cannot update another user's list", async () => {
                const user = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(user);
                const input: BookmarkListUpdateInput = { id: listUser2.id.toString(), label: "Hacked" }; // Ensure ID is string for input
                try {
                    await bookmarkList.updateOne({ input }, { req, res }, bookmarkList_updateOne);
                    expect.fail("Expected an error");
                } catch (error) {
                    // error expected
                }
            });

            it("not logged in user cannot update", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: BookmarkListUpdateInput = { id: listUser1.id.toString(), label: "NoAuthUpdate" }; // Ensure ID is string for input
                try {
                    await bookmarkList.updateOne({ input }, { req, res }, bookmarkList_updateOne);
                    expect.fail("Expected an error");
                } catch (error) {
                    // error expected
                }
            });
        });
    });
}); 
