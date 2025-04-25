import { FindByIdInput, ResourceCreateInput, ResourceSearchInput, ResourceUpdateInput, ResourceUsedFor, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { resource_createOne } from "../generated/resource_createOne.js";
import { resource_findMany } from "../generated/resource_findMany.js";
import { resource_findOne } from "../generated/resource_findOne.js";
import { resource_updateOne } from "../generated/resource_updateOne.js";
import { resource } from "./resource.js";

// Test users
const user1Id = uuid();
const user2Id = uuid();

// Owning entities (Posts)
let postUser1Public: any;
let postUser1Private: any;
let postUser2Public: any;
let postUser2Private: any;

// Resource lists
let rlUser1Public: any;
let rlUser1Private: any;
let rlUser2Public: any;
let rlUser2Private: any;

// Test resource data
let resourceUser1Public: any;
let resourceUser1Private: any;
let resourceUser2Public: any;
let resourceUser2Private: any;

const userResource1PublicData = {
    id: uuid(),
    link: "http://example.com/resource1-public",
    index: 0,
    usedFor: ResourceUsedFor.Learning,
};

const userResource1PrivateData = {
    id: uuid(),
    link: "http://example.com/resource1-private",
    index: 1,
    usedFor: ResourceUsedFor.Notes,
};

const userResource2PublicData = {
    id: uuid(),
    link: "http://example.com/resource2-public",
    index: 2,
    usedFor: ResourceUsedFor.Social,
};

const userResource2PrivateData = {
    id: uuid(),
    link: "http://example.com/resource2-private",
    index: 3,
    usedFor: ResourceUsedFor.ExternalService,
};

describe("EndpointsResource", () => {
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

        // Create user-specific resource lists linked to posts
        rlUser1Public = await DbProvider.get().resource_list.create({
            data: { post: { connect: { id: postUser1Public.id } } },
        });
        rlUser1Private = await DbProvider.get().resource_list.create({
            data: { post: { connect: { id: postUser1Private.id } } },
        });
        rlUser2Public = await DbProvider.get().resource_list.create({
            data: { post: { connect: { id: postUser2Public.id } } },
        });
        rlUser2Private = await DbProvider.get().resource_list.create({
            data: { post: { connect: { id: postUser2Private.id } } },
        });

        // Create user-specific resources
        resourceUser1Public = await DbProvider.get().resource.create({
            data: {
                ...userResource1PublicData,
                list: { connect: { id: rlUser1Public.id } },
            },
        });
        resourceUser1Private = await DbProvider.get().resource.create({
            data: {
                ...userResource1PrivateData,
                list: { connect: { id: rlUser1Private.id } },
            },
        });
        resourceUser2Public = await DbProvider.get().resource.create({
            data: {
                ...userResource2PublicData,
                list: { connect: { id: rlUser2Public.id } },
            },
        });
        resourceUser2Private = await DbProvider.get().resource.create({
            data: {
                ...userResource2PrivateData,
                list: { connect: { id: rlUser2Private.id } },
            },
        });
    });

    after(async function after() {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns own public resource", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: FindByIdInput = { id: resourceUser1Public.id };
                const result = await resource.findOne({ input }, { req, res }, resource_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(resourceUser1Public.id);
            });

            it("returns own private resource", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: FindByIdInput = { id: resourceUser1Private.id };
                const result = await resource.findOne({ input }, { req, res }, resource_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(resourceUser1Private.id);
            });

            it("returns another user's public resource", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id }; // User 1 requesting
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: FindByIdInput = { id: resourceUser2Public.id }; // User 2's public resource
                const result = await resource.findOne({ input }, { req, res }, resource_findOne);
                // Assuming helper allows reading resources in public lists of others
                expect(result).to.not.be.null;
                expect(result.id).to.equal(resourceUser2Public.id);
            });

            it("API key with public permissions finds public resources", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);

                // Find user 1's public resource
                let input: FindByIdInput = { id: resourceUser1Public.id };
                let result = await resource.findOne({ input }, { req, res }, resource_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(resourceUser1Public.id);

                // Find user 2's public resource
                input = { id: resourceUser2Public.id };
                result = await resource.findOne({ input }, { req, res }, resource_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(resourceUser2Public.id);
            });

            it("logged out user finds public resources", async () => {
                const { req, res } = await mockLoggedOutSession();

                // Find user 1's public resource
                let input: FindByIdInput = { id: resourceUser1Public.id };
                let result = await resource.findOne({ input }, { req, res }, resource_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(resourceUser1Public.id);

                // Find user 2's public resource
                input = { id: resourceUser2Public.id };
                result = await resource.findOne({ input }, { req, res }, resource_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(resourceUser2Public.id);
            });
        });

        describe("invalid", () => {
            it("fails when resource id doesn't exist", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: FindByIdInput = { id: uuid() };
                try {
                    await resource.findOne({ input }, { req, res }, resource_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* Error expected */ }
            });

            it("fails when user tries to access another user's private resource", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: FindByIdInput = { id: resourceUser2Private.id }; // User 2's private resource
                try {
                    await resource.findOne({ input }, { req, res }, resource_findOne);
                    expect.fail("Expected an error accessing other user private resource");
                } catch (error) { /* Error expected */ }
            });

            it("logged out user can't access private resources", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: resourceUser1Private.id }; // User 1's private resource
                try {
                    await resource.findOne({ input }, { req, res }, resource_findOne);
                    expect.fail("Expected error for logged out user accessing private resource");
                } catch (error) { /* Error expected */ }
            });

            it("API key with public permissions cannot find private resources", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);

                // Try user 1's private resource
                let input: FindByIdInput = { id: resourceUser1Private.id };
                try {
                    await resource.findOne({ input }, { req, res }, resource_findOne);
                    expect.fail("Expected error finding user 1 private resource with public key");
                } catch (error) { /* Error expected */ }

                // Try user 2's private resource
                input = { id: resourceUser2Private.id };
                try {
                    await resource.findOne({ input }, { req, res }, resource_findOne);
                    expect.fail("Expected error finding user 2 private resource with public key");
                } catch (error) { /* Error expected */ }
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns own and others' public resources for authenticated user", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                // User 1 should see their public/private resources + user 2's public resource
                const expectedResourceIds = [
                    resourceUser1Public.id,
                    resourceUser1Private.id,
                    resourceUser2Public.id,
                    // resourceUser2Private.id should NOT be included
                ];

                const input: ResourceSearchInput = { take: 10 };
                const result = await resource.findMany({ input }, { req, res }, resource_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultResourceIds = result.edges!.map(edge => edge!.node!.id);
                expect(resultResourceIds.sort()).to.deep.equal(expectedResourceIds.sort());
            });

            it("filters by updated time frame (respecting visibility)", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                // Manually update a resource to test timeframe
                const updatedResource = await DbProvider.get().resource.update({
                    where: { id: resourceUser1Public.id },
                    data: { updated_at: new Date("2024-01-01") },
                });

                // User 1 searching within timeframe - should see updated public resource
                const expectedResourceIds = [updatedResource.id];

                const input: ResourceSearchInput = {
                    take: 10,
                    updatedTimeFrame: {
                        after: new Date("2023-12-31"),
                        before: new Date("2024-01-02"),
                    },
                };
                const result = await resource.findMany({ input }, { req, res }, resource_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultResourceIds = result.edges!.map(edge => edge!.node!.id);
                expect(resultResourceIds.sort()).to.deep.equal(expectedResourceIds.sort());
            });

            it("API key with public permissions returns only public resources", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);

                // Should only see public resources
                const expectedResourceIds = [
                    resourceUser1Public.id,
                    resourceUser2Public.id,
                ];

                const input: ResourceSearchInput = { take: 10 };
                const result = await resource.findMany({ input }, { req, res }, resource_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultResourceIds = result.edges!.map(edge => edge!.node!.id);
                expect(resultResourceIds.sort()).to.deep.equal(expectedResourceIds.sort());
            });

            it("logged out user returns only public resources", async () => {
                const { req, res } = await mockLoggedOutSession();

                // Should only see public resources
                const expectedResourceIds = [
                    resourceUser1Public.id,
                    resourceUser2Public.id,
                ];

                const input: ResourceSearchInput = { take: 10 };
                const result = await resource.findMany({ input }, { req, res }, resource_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultResourceIds = result.edges!.map(edge => edge!.node!.id);
                expect(resultResourceIds.sort()).to.deep.equal(expectedResourceIds.sort());
            });
        });

        // No specific 'invalid' cases for findMany other than permissions handled above
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates a resource in own public list", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const newResourceId = uuid();
                const input: ResourceCreateInput = {
                    id: newResourceId,
                    link: "http://new.example.com/public",
                    usedFor: ResourceUsedFor.Learning,
                    index: 10,
                    listConnect: rlUser1Public.id, // Link to user 1's public list
                };
                const creationResult = await resource.createOne({ input }, { req, res }, resource_createOne);
                expect(creationResult).to.not.be.null;
                expect(creationResult.id).to.equal(newResourceId);
            });

            it("creates a resource in own private list", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const newResourceId = uuid();
                const input: ResourceCreateInput = {
                    id: newResourceId,
                    link: "http://new.example.com/private",
                    usedFor: ResourceUsedFor.Notes,
                    index: 11,
                    listConnect: rlUser1Private.id, // Link to user 1's private list
                };
                const creationResult = await resource.createOne({ input }, { req, res }, resource_createOne);
                expect(creationResult).to.not.be.null;
                expect(creationResult.id).to.equal(newResourceId);
            });

            it("API key with write permissions can create resource in own list", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const newResourceId = uuid();
                const input: ResourceCreateInput = {
                    id: newResourceId,
                    link: "http://api.example.com",
                    usedFor: ResourceUsedFor.Social,
                    index: 12,
                    listConnect: rlUser1Public.id,
                };
                const result = await resource.createOne({ input }, { req, res }, resource_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newResourceId);
            });
        });

        describe("invalid", () => {
            it("not logged in", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: ResourceCreateInput = {
                    id: uuid(),
                    link: "http://unauthorized.com",
                    usedFor: ResourceUsedFor.Related,
                    index: 13,
                    listConnect: rlUser1Public.id,
                };
                try {
                    await resource.createOne({ input }, { req, res }, resource_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* Error expected */ }
            });

            it("cannot create resource connected to another user's list (public or private)", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                // Try linking to user 2's public list
                let input: ResourceCreateInput = {
                    id: uuid(),
                    link: "http://cross.user.com/public",
                    usedFor: ResourceUsedFor.Tutorial,
                    index: 14,
                    listConnect: rlUser2Public.id,
                };
                try {
                    await resource.createOne({ input }, { req, res }, resource_createOne);
                    expect.fail("Expected error creating resource in other user public list");
                } catch (error) { /* Error expected */ }

                // Try linking to user 2's private list
                input = {
                    id: uuid(),
                    link: "http://cross.user.com/private",
                    usedFor: ResourceUsedFor.Notes,
                    index: 15,
                    listConnect: rlUser2Private.id,
                };
                try {
                    await resource.createOne({ input }, { req, res }, resource_createOne);
                    expect.fail("Expected error creating resource in other user private list");
                } catch (error) { /* Error expected */ }
            });

            it("API key without write permission cannot create resource", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = mockReadPublicPermissions(); // No write permission
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: ResourceCreateInput = {
                    id: uuid(),
                    link: "http://api-no-write.com",
                    usedFor: ResourceUsedFor.Notes,
                    index: 16,
                    listConnect: rlUser1Public.id,
                };
                try {
                    await resource.createOne({ input }, { req, res }, resource_createOne);
                    expect.fail("Expected an error to be thrown due to missing write permission");
                } catch (error) { /* Error expected */ }
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("updates own public resource", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: ResourceUpdateInput = {
                    id: resourceUser1Public.id,
                    link: "http://updated.example.com/public",
                    index: 20,
                };
                const result = await resource.updateOne({ input }, { req, res }, resource_updateOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(resourceUser1Public.id);
                expect(result.link).to.equal(input.link);
                expect(result.index).to.equal(input.index);
            });

            it("updates own private resource", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: ResourceUpdateInput = {
                    id: resourceUser1Private.id,
                    link: "http://updated.example.com/private",
                    index: 21,
                };
                const result = await resource.updateOne({ input }, { req, res }, resource_updateOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(resourceUser1Private.id);
                expect(result.link).to.equal(input.link);
                expect(result.index).to.equal(input.index);
            });

            it("API key with write permissions can update own resource", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: ResourceUpdateInput = {
                    id: resourceUser1Public.id,
                    link: "http://api-updated.example.com",
                };
                const result = await resource.updateOne({ input }, { req, res }, resource_updateOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(resourceUser1Public.id);
                expect(result.link).to.equal(input.link);
            });
        });

        describe("invalid", () => {
            it("cannot update another user's public resource", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: ResourceUpdateInput = {
                    id: resourceUser2Public.id, // User 2's public resource
                    link: "http://unauthorized-update.com/public",
                };
                try {
                    await resource.updateOne({ input }, { req, res }, resource_updateOne);
                    expect.fail("Expected error updating other user public resource");
                } catch (error) { /* Error expected */ }
            });

            it("cannot update another user's private resource", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: ResourceUpdateInput = {
                    id: resourceUser2Private.id, // User 2's private resource
                    link: "http://unauthorized-update.com/private",
                };
                try {
                    await resource.updateOne({ input }, { req, res }, resource_updateOne);
                    expect.fail("Expected error updating other user private resource");
                } catch (error) { /* Error expected */ }
            });

            it("not logged in user can't update resource", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: ResourceUpdateInput = {
                    id: resourceUser1Public.id,
                    link: "http://unauthorized-update.com",
                };
                try {
                    await resource.updateOne({ input }, { req, res }, resource_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* Error expected */ }
            });

            it("cannot update non-existent resource", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: ResourceUpdateInput = {
                    id: uuid(), // Non-existent ID
                    link: "http://update-non-existent.com",
                };
                try {
                    await resource.updateOne({ input }, { req, res }, resource_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* Error expected */ }
            });

            it("API key without write permission cannot update resource", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = mockReadPublicPermissions(); // No write permission
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: ResourceUpdateInput = {
                    id: resourceUser1Public.id,
                    link: "http://api-no-write-update.com",
                };
                try {
                    await resource.updateOne({ input }, { req, res }, resource_updateOne);
                    expect.fail("Expected an error to be thrown due to missing write permission");
                } catch (error) { /* Error expected */ }
            });
        });
    });
}); 
