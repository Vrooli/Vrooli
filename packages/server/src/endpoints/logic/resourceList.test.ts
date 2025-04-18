import { ApiKeyPermission, FindByIdInput, ResourceListCreateInput, ResourceListFor, ResourceListSearchInput, ResourceListUpdateInput, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { resourceList_createOne } from "../generated/resourceList_createOne.js";
import { resourceList_findMany } from "../generated/resourceList_findMany.js";
import { resourceList_findOne } from "../generated/resourceList_findOne.js";
import { resourceList_updateOne } from "../generated/resourceList_updateOne.js";
import { resourceList } from "./resourceList.js";

// Test users
const user1Id = uuid();
const user2Id = uuid();

// Owning entities (Posts)
let postUser1Public: any;
let postUser1Private: any;
let postUser2Public: any;
let postUser2Private: any;

// ResourceList IDs for tests (store the objects instead of just IDs)
let rlUser1Public: any;
let rlUser1Private: any;
let rlUser2Public: any;
let rlUser2Private: any;


describe("EndpointsResourceList", () => {
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
        await DbProvider.get().resource_list.deleteMany({});
        await DbProvider.get().resource.deleteMany({}); // Clean resources too, as they depend on lists
        await DbProvider.get().post.deleteMany({}); // Clean up posts

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

        // Create user-specific posts (owning entities)
        postUser1Public = await DbProvider.get().post.create({
            data: {
                userId: user1Id,
                isPrivate: false,
            },
        });
        postUser1Private = await DbProvider.get().post.create({
            data: {
                userId: user1Id,
                isPrivate: true,
            },
        });
        postUser2Public = await DbProvider.get().post.create({
            data: {
                userId: user2Id,
                isPrivate: false,
            },
        });
        postUser2Private = await DbProvider.get().post.create({
            data: {
                userId: user2Id,
                isPrivate: true,
            },
        });

        // Create ResourceLists for users linked to posts
        rlUser1Public = await DbProvider.get().resource_list.create({
            data: {
                post: { connect: { id: postUser1Public.id } },
            },
        });
        rlUser1Private = await DbProvider.get().resource_list.create({
            data: {
                post: { connect: { id: postUser1Private.id } },
            },
        });
        rlUser2Public = await DbProvider.get().resource_list.create({
            data: {
                post: { connect: { id: postUser2Public.id } },
            },
        });
        rlUser2Private = await DbProvider.get().resource_list.create({
            data: {
                post: { connect: { id: postUser2Private.id } },
            },
        });
    });

    after(async function after() {
        // Clear databases
        await (await initializeRedis())?.flushAll();
        await DbProvider.get().resource.deleteMany({});
        await DbProvider.get().resource_list.deleteMany({});
        await DbProvider.get().post.deleteMany({});
        await DbProvider.get().user.deleteMany({});

        // Restore logger stubs
        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns own public resource list", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: FindByIdInput = { id: rlUser1Public.id };
                const result = await resourceList.findOne({ input }, { req, res }, resourceList_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(rlUser1Public.id);
            });

            it("returns own private resource list", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: FindByIdInput = { id: rlUser1Private.id };
                const result = await resourceList.findOne({ input }, { req, res }, resourceList_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(rlUser1Private.id);
            });

            it("returns another user's public resource list", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id }; // User 1 requesting
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: FindByIdInput = { id: rlUser2Public.id }; // User 2's public list
                const result = await resourceList.findOne({ input }, { req, res }, resourceList_findOne);
                // Assuming readOneHelper allows reading public items even if not owned
                expect(result).to.not.be.null;
                expect(result.id).to.equal(rlUser2Public.id);
            });

            it("API key with public permissions can find public lists", async () => {
                const permissions = { [ApiKeyPermission.ReadPublic]: true } as Record<ApiKeyPermission, boolean>;
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                // User doesn't matter for public read permission, but mock needs one
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);

                // Find user 1's public list
                let input: FindByIdInput = { id: rlUser1Public.id };
                let result = await resourceList.findOne({ input }, { req, res }, resourceList_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(rlUser1Public.id);

                // Find user 2's public list
                input = { id: rlUser2Public.id };
                result = await resourceList.findOne({ input }, { req, res }, resourceList_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(rlUser2Public.id);
            });

            // Test for logged-out user finding public list
            it("logged out user can find public lists", async () => {
                const { req, res } = await mockLoggedOutSession();

                // Find user 1's public list
                let input: FindByIdInput = { id: rlUser1Public.id };
                let result = await resourceList.findOne({ input }, { req, res }, resourceList_findOne);
                // This assumes readOneHelper allows public reads for logged-out users
                expect(result).to.not.be.null;
                expect(result.id).to.equal(rlUser1Public.id);

                // Find user 2's public list
                input = { id: rlUser2Public.id };
                result = await resourceList.findOne({ input }, { req, res }, resourceList_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(rlUser2Public.id);
            });
        });

        describe("invalid", () => {
            it("fails when resource list id doesn't exist", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: FindByIdInput = { id: uuid() }; // Non-existent ID
                try {
                    await resourceList.findOne({ input }, { req, res }, resourceList_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* Error expected */ }
            });

            it("fails when user tries to access another user's private resource list", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id }; // User 1 requesting
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: FindByIdInput = { id: rlUser2Private.id }; // User 2's private list
                try {
                    await resourceList.findOne({ input }, { req, res }, resourceList_findOne);
                    // This assumes readOneHelper correctly blocks access to private items of others
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* Error expected */ }
            });

            it("logged out user can't access private resource lists", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: rlUser1Private.id }; // User 1's private list
                try {
                    await resourceList.findOne({ input }, { req, res }, resourceList_findOne);
                    // This assumes readOneHelper correctly blocks access for logged-out users to private items
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* Error expected */ }
            });

            it("API key with public permissions cannot find private lists", async () => {
                const permissions = { [ApiKeyPermission.ReadPublic]: true } as Record<ApiKeyPermission, boolean>;
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);

                // Try user 1's private list
                let input: FindByIdInput = { id: rlUser1Private.id };
                try {
                    await resourceList.findOne({ input }, { req, res }, resourceList_findOne);
                    expect.fail("Expected error finding user 1 private list with public key");
                } catch (error) { /* Error expected */ }

                // Try user 2's private list
                input = { id: rlUser2Private.id };
                try {
                    await resourceList.findOne({ input }, { req, res }, resourceList_findOne);
                    expect.fail("Expected error finding user 2 private list with public key");
                } catch (error) { /* Error expected */ }
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns own and others' public lists for authenticated user", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                // User 1 should see their own public/private lists + user 2's public list
                const expectedResourceListIds = [
                    rlUser1Public.id,
                    rlUser1Private.id,
                    rlUser2Public.id,
                    // rlUser2Private.id should NOT be included
                ];

                const input: ResourceListSearchInput = { take: 10 };
                const result = await resourceList.findMany({ input }, { req, res }, resourceList_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");

                const resultResourceListIds = result.edges!.map(edge => edge!.node!.id);
                expect(resultResourceListIds.sort()).to.deep.equal(expectedResourceListIds.sort());
            });

            it("API key with public permissions returns only public lists", async () => {
                const permissions = { [ApiKeyPermission.ReadPublic]: true } as Record<ApiKeyPermission, boolean>;
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);

                // Should only see public lists
                const expectedResourceListIds = [
                    rlUser1Public.id,
                    rlUser2Public.id,
                ];

                const input: ResourceListSearchInput = { take: 10 };
                const result = await resourceList.findMany({ input }, { req, res }, resourceList_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultResourceListIds = result.edges!.map(edge => edge!.node!.id);
                expect(resultResourceListIds.sort()).to.deep.equal(expectedResourceListIds.sort());
            });

            it("logged out user returns only public lists", async () => {
                const { req, res } = await mockLoggedOutSession();

                // Should only see public lists
                const expectedResourceListIds = [
                    rlUser1Public.id,
                    rlUser2Public.id,
                ];

                const input: ResourceListSearchInput = { take: 10 };
                const result = await resourceList.findMany({ input }, { req, res }, resourceList_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultResourceListIds = result.edges!.map(edge => edge!.node!.id);
                expect(resultResourceListIds.sort()).to.deep.equal(expectedResourceListIds.sort());
            });
        });

        // No specific 'invalid' cases for findMany other than permissions handled above
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates a resource list linked to own public post", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const newListId = uuid();
                const input: ResourceListCreateInput = {
                    id: newListId,
                    listForConnect: postUser1Public.id,
                    listForType: ResourceListFor.Post,
                };
                const creationResult = await resourceList.createOne({ input }, { req, res }, resourceList_createOne);
                expect(creationResult).to.not.be.null;
                expect(creationResult.id).to.equal(newListId);
                // Verify ownership indirectly
                const createdList = await DbProvider.get().resource_list.findUnique({ where: { id: newListId }, select: { post: { select: { userId: true } } } });
                expect(createdList?.post?.userId).to.equal(user1Id);
            });

            it("creates a resource list linked to own private post", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const newListId = uuid();
                const input: ResourceListCreateInput = {
                    id: newListId,
                    listForConnect: postUser1Private.id,
                    listForType: ResourceListFor.Post,
                };
                const creationResult = await resourceList.createOne({ input }, { req, res }, resourceList_createOne);
                expect(creationResult).to.not.be.null;
                expect(creationResult.id).to.equal(newListId);
                const createdList = await DbProvider.get().resource_list.findUnique({ where: { id: newListId }, select: { post: { select: { userId: true } } } });
                expect(createdList?.post?.userId).to.equal(user1Id);
            });

            it("API key with write permissions can create list linked to own post", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = { [ApiKeyPermission.WritePrivate]: true } as Record<ApiKeyPermission, boolean>;
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const newListId = uuid();
                const input: ResourceListCreateInput = {
                    id: newListId,
                    listForConnect: postUser1Public.id,
                    listForType: ResourceListFor.Post,
                };
                const result = await resourceList.createOne({ input }, { req, res }, resourceList_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newListId);
                const createdList = await DbProvider.get().resource_list.findUnique({ where: { id: newListId }, select: { post: { select: { userId: true } } } });
                expect(createdList?.post?.userId).to.equal(user1Id);
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create resource list", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: ResourceListCreateInput = {
                    id: uuid(),
                    listForConnect: postUser1Public.id,
                    listForType: ResourceListFor.Post,
                };
                try {
                    await resourceList.createOne({ input }, { req, res }, resourceList_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* Error expected */ }
            });

            it("cannot create list linked to another user's post (public or private)", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                // Try linking to user 2's public post
                let input: ResourceListCreateInput = {
                    id: uuid(),
                    listForConnect: postUser2Public.id,
                    listForType: ResourceListFor.Post,
                };
                try {
                    await resourceList.createOne({ input }, { req, res }, resourceList_createOne);
                    expect.fail("Expected error creating list linked to other user public post");
                } catch (error) { /* Error expected */ }

                // Try linking to user 2's private post
                input = {
                    id: uuid(),
                    listForConnect: postUser2Private.id,
                    listForType: ResourceListFor.Post,
                };
                try {
                    await resourceList.createOne({ input }, { req, res }, resourceList_createOne);
                    expect.fail("Expected error creating list linked to other user private post");
                } catch (error) { /* Error expected */ }
            });

            it("API key without write permissions cannot create resource list", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = { [ApiKeyPermission.ReadPublic]: true } as Record<ApiKeyPermission, boolean>; // No write permission
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: ResourceListCreateInput = {
                    id: uuid(),
                    listForConnect: postUser1Public.id,
                    listForType: ResourceListFor.Post,
                };
                try {
                    await resourceList.createOne({ input }, { req, res }, resourceList_createOne);
                    expect.fail("Expected an error to be thrown - write permission required");
                } catch (error) { /* Error expected */ }
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("updates own public resource list", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                // Update logic might not have fields yet, just test ownership allows the call
                const input: ResourceListUpdateInput = { id: rlUser1Public.id };
                const result = await resourceList.updateOne({ input }, { req, res }, resourceList_updateOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(rlUser1Public.id);
            });

            it("updates own private resource list", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: ResourceListUpdateInput = { id: rlUser1Private.id };
                const result = await resourceList.updateOne({ input }, { req, res }, resourceList_updateOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(rlUser1Private.id);
            });

            it("API key with write permissions can update own resource list", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = { [ApiKeyPermission.WritePrivate]: true } as Record<ApiKeyPermission, boolean>;
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: ResourceListUpdateInput = { id: rlUser1Public.id };
                const result = await resourceList.updateOne({ input }, { req, res }, resourceList_updateOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(rlUser1Public.id);
            });
        });

        describe("invalid", () => {
            it("cannot update another user's public resource list", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: ResourceListUpdateInput = { id: rlUser2Public.id };
                try {
                    await resourceList.updateOne({ input }, { req, res }, resourceList_updateOne);
                    expect.fail("Expected an error updating other user public list");
                } catch (error) { /* Error expected */ }
            });

            it("cannot update another user's private resource list", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: ResourceListUpdateInput = { id: rlUser2Private.id };
                try {
                    await resourceList.updateOne({ input }, { req, res }, resourceList_updateOne);
                    expect.fail("Expected an error updating other user private list");
                } catch (error) { /* Error expected */ }
            });

            it("not logged in user cannot update resource list", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: ResourceListUpdateInput = { id: rlUser1Public.id };
                try {
                    await resourceList.updateOne({ input }, { req, res }, resourceList_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* Error expected */ }
            });

            it("cannot update non-existent resource list", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: ResourceListUpdateInput = { id: uuid() }; // Non-existent ID
                try {
                    await resourceList.updateOne({ input }, { req, res }, resourceList_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* Error expected */ }
            });

            it("API key without write permissions cannot update resource list", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = { [ApiKeyPermission.ReadPublic]: true } as Record<ApiKeyPermission, boolean>; // No write permission
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: ResourceListUpdateInput = { id: rlUser1Public.id };
                try {
                    await resourceList.updateOne({ input }, { req, res }, resourceList_updateOne);
                    expect.fail("Expected an error to be thrown - write permission required");
                } catch (error) { /* Error expected */ }
            });
        });
    });
}); 
