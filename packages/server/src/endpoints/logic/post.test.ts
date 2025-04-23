// Tests for the Post endpoint (findOne, findMany, createOne, updateOne)
import { FindByIdInput, PostCreateInput, PostSearchInput, PostUpdateInput, SEEDED_IDS, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { post_createOne } from "../generated/post_createOne.js";
import { post_findMany } from "../generated/post_findMany.js";
import { post_findOne } from "../generated/post_findOne.js";
import { post_updateOne } from "../generated/post_updateOne.js";
import { post } from "./post.js";

// Test users
const user1Id = uuid();
const user2Id = uuid();

// Test posts
let postUser1Public: any;
let postUser1Private: any;
let postUser2Public: any;
let postUser2Private: any;

describe("EndpointsPost", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        // Stub logger to suppress output
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        // Clear Redis and truncate tables
        await (await initializeRedis())?.flushAll();
        await DbProvider.get().session.deleteMany();
        await DbProvider.get().user_auth.deleteMany();
        await DbProvider.get().user.deleteMany({});
        await DbProvider.get().post_translation.deleteMany();
        await DbProvider.get().post.deleteMany({});

        // Create two users
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

        // Seed posts with varying privacy and ownership
        postUser1Public = await DbProvider.get().post.create({ data: { id: uuid(), userId: user1Id, isPrivate: false } });
        postUser1Private = await DbProvider.get().post.create({ data: { id: uuid(), userId: user1Id, isPrivate: true } });
        postUser2Public = await DbProvider.get().post.create({ data: { id: uuid(), userId: user2Id, isPrivate: false } });
        postUser2Private = await DbProvider.get().post.create({ data: { id: uuid(), userId: user2Id, isPrivate: true } });
    });

    after(async () => {
        // Clean up after tests
        await (await initializeRedis())?.flushAll();
        await DbProvider.get().post_translation.deleteMany();
        await DbProvider.get().post.deleteMany();
        await DbProvider.get().user.deleteMany({});
        // Restore logger stubs
        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns own public post", async () => {
                const user = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(user);
                const input: FindByIdInput = { id: postUser1Public.id };
                const result = await post.findOne({ input }, { req, res }, post_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(postUser1Public.id);
            });

            it("returns own private post", async () => {
                const user = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(user);
                const input: FindByIdInput = { id: postUser1Private.id };
                const result = await post.findOne({ input }, { req, res }, post_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(postUser1Private.id);
            });

            it("returns another user's public post", async () => {
                const user = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(user);
                const input: FindByIdInput = { id: postUser2Public.id };
                const result = await post.findOne({ input }, { req, res }, post_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(postUser2Public.id);
            });

            it("API key with public read can find public post", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);
                const input: FindByIdInput = { id: postUser2Public.id };
                const result = await post.findOne({ input }, { req, res }, post_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(postUser2Public.id);
            });

            it("logged out user can find public post", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: postUser1Public.id };
                const result = await post.findOne({ input }, { req, res }, post_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(postUser1Public.id);
            });
        });

        describe("invalid", () => {
            it("does not return another user's private post", async () => {
                const user = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(user);
                const input: FindByIdInput = { id: postUser2Private.id };
                try {
                    await post.findOne({ input }, { req, res }, post_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) {
                    // Error expected
                }
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns own and public posts for authenticated user", async () => {
                const user = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(user);
                const input: PostSearchInput = { take: 10 };
                const result = await post.findMany({ input }, { req, res }, post_findMany);
                expect(result.edges.map(e => e!.node!.id).sort()).to.deep.equal(
                    [postUser1Public.id, postUser1Private.id, postUser2Public.id].sort(),
                );
            });

            it("API key with public read returns only public posts", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);
                const input: PostSearchInput = { take: 10 };
                const result = await post.findMany({ input }, { req, res }, post_findMany);
                expect(result.edges.map(e => e!.node!.id).sort()).to.deep.equal(
                    [postUser1Public.id, postUser2Public.id].sort(),
                );
            });
        });

        describe("invalid", () => {
            it("logged out user returns only public posts", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: PostSearchInput = { take: 10 };
                const result = await post.findMany({ input }, { req, res }, post_findMany);
                expect(result.edges.map(e => e!.node!.id).sort()).to.deep.equal(
                    [postUser1Public.id, postUser2Public.id].sort(),
                );
            });
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates a post for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const newPostId = uuid();
                const input: PostCreateInput = { id: newPostId, isPrivate: false };
                const result = await post.createOne({ input }, { req, res }, post_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newPostId);
                expect(result.you.canDelete).to.be.false; // canDelete false for non-admin
            });

            it("API key with write permissions can create post", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);
                const newPostId = uuid();
                const input: PostCreateInput = { id: newPostId, isPrivate: true };
                const result = await post.createOne({ input }, { req, res }, post_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newPostId);
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create post", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: PostCreateInput = { id: uuid(), isPrivate: false };
                try {
                    await post.createOne({ input }, { req, res }, post_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch {
                    // Error expected
                }
            });

            it("API key without write permissions cannot create post", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);
                const input: PostCreateInput = { id: uuid(), isPrivate: false };
                try {
                    await post.createOne({ input }, { req, res }, post_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch {
                    // Error expected
                }
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("allows admin to update a post", async () => {
                // Ensure admin user exists for update tests
                await DbProvider.get().user.upsert({
                    where: { id: SEEDED_IDS.User.Admin },
                    update: {},
                    create: {
                        id: SEEDED_IDS.User.Admin,
                        name: "Admin",
                        handle: "admin",
                        status: "Unlocked",
                        isBot: false,
                        isBotDepictingPerson: false,
                        isPrivate: false,
                        auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] },
                    },
                });
                const admin = { ...loggedInUserNoPremiumData, id: SEEDED_IDS.User.Admin };
                const { req, res } = await mockAuthenticatedSession(admin);
                const input: PostUpdateInput = { id: postUser1Public.id, isPrivate: true };
                const result = await post.updateOne({ input }, { req, res }, post_updateOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(postUser1Public.id);
            });
        });

        describe("invalid", () => {
            it("denies update for non-admin user", async () => {
                const user = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(user);
                const input: PostUpdateInput = { id: postUser1Public.id, isPrivate: true };
                try {
                    await post.updateOne({ input }, { req, res }, post_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch {
                    // Error expected
                }
            });

            it("denies update for not logged in user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: PostUpdateInput = { id: postUser1Public.id, isPrivate: true };
                try {
                    await post.updateOne({ input }, { req, res }, post_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch {
                    // Error expected
                }
            });

            it("throws when updating non-existent post as admin", async () => {
                await DbProvider.get().user.upsert({
                    where: { id: SEEDED_IDS.User.Admin },
                    update: {},
                    create: {
                        id: SEEDED_IDS.User.Admin,
                        name: "Admin",
                        handle: "admin",
                        status: "Unlocked",
                        isBot: false,
                        isBotDepictingPerson: false,
                        isPrivate: false,
                        auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] },
                    },
                });
                const admin = { ...loggedInUserNoPremiumData, id: SEEDED_IDS.User.Admin };
                const { req, res } = await mockAuthenticatedSession(admin);
                const input: PostUpdateInput = { id: uuid(), isPrivate: true };
                try {
                    await post.updateOne({ input }, { req, res }, post_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch {
                    // Error expected
                }
            });
        });
    });
});
