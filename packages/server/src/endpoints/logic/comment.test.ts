import { CommentCreateInput, CommentFor, CommentSearchInput, CommentUpdateInput, FindByIdInput, SEEDED_IDS, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { comment_createOne } from "../generated/comment_createOne.js";
import { comment_findMany } from "../generated/comment_findMany.js";
import { comment_findOne } from "../generated/comment_findOne.js";
import { comment_updateOne } from "../generated/comment_updateOne.js";
import { comment } from "./comment.js";

// Test users
const user1Id = uuid();
const user2Id = uuid();

let post: any;
let comment1: { id: string; translations: Array<{ id: string; language: string; text: string; }> };
let comment2: { id: string; translations: Array<{ id: string; language: string; text: string; }> };

describe("EndpointsComment", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        // Clear databases
        await (await initializeRedis())?.flushAll();
        await DbProvider.get().session.deleteMany();
        await DbProvider.get().user_auth.deleteMany();
        await DbProvider.get().user.deleteMany({});
        await DbProvider.get().post.deleteMany();
        await DbProvider.get().comment.deleteMany();

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

        // Create a public post to comment on
        post = await DbProvider.get().post.create({
            data: {
                userId: user1Id,
                isPrivate: false,
            },
        });

        // Seed top-level comment
        comment1 = {
            id: uuid(),
            translations: [{ id: uuid(), language: "en", text: "First comment" }],
        };
        await DbProvider.get().comment.create({
            data: {
                id: comment1.id,
                ownedByUser: { connect: { id: user1Id } },
                post: { connect: { id: post.id } },
                translations: {
                    create: comment1.translations.map((t) => ({
                        id: t.id,
                        language: t.language,
                        text: t.text,
                    })),
                },
            },
        });

        // Seed reply comment (child thread)
        comment2 = {
            id: uuid(),
            translations: [{ id: uuid(), language: "en", text: "Reply to first comment" }],
        };
        await DbProvider.get().comment.create({
            data: {
                id: comment2.id,
                ownedByUser: { connect: { id: user2Id } },
                post: { connect: { id: post.id } },
                parent: { connect: { id: comment1.id } },
                translations: {
                    create: comment2.translations.map((t) => ({
                        id: t.id,
                        language: t.language,
                        text: t.text,
                    })),
                },
            },
        });
    });

    after(async () => {
        // Clear databases
        await (await initializeRedis())?.flushAll();
        await DbProvider.get().comment.deleteMany();
        await DbProvider.get().post.deleteMany();
        await DbProvider.get().user.deleteMany({});
        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns comment by id for any authenticated user", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: FindByIdInput = { id: comment1.id };
                const result = await comment.findOne({ input }, { req, res }, comment_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(comment1.id);
                expect(result.translations?.[0]?.text).to.equal(comment1.translations[0].text);
            });

            it("returns comment by id when not authenticated", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: comment1.id };
                const result = await comment.findOne({ input }, { req, res }, comment_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(comment1.id);
            });

            it("returns comment by id with API key public read", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: FindByIdInput = { id: comment1.id };
                const result = await comment.findOne({ input }, { req, res }, comment_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(comment1.id);
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns comments with nested replies for any authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const input: CommentSearchInput = { take: 10 };
                const result = await comment.findMany({ input }, { req, res }, comment_findMany);
                expect(result).to.not.be.null;
                expect(result).to.have.property("threads").that.is.an("array");
                // Top-level threads should include the first comment
                const topThread = result.threads![0];
                expect(topThread.comment.id).to.equal(comment1.id);
                // There should be one nested reply
                expect(topThread.childThreads).to.be.an("array").with.length(1);
                expect(topThread.childThreads![0].comment.id).to.equal(comment2.id);
                expect(result.totalThreads).to.equal(1);
            });

            it("returns comments for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: CommentSearchInput = { take: 10 };
                const result = await comment.findMany({ input }, { req, res }, comment_findMany);
                expect(result).to.not.be.null;
                expect(result.threads).to.have.lengthOf(1);
            });

            it("returns comments for API key public read", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: CommentSearchInput = { take: 10 };
                const result = await comment.findMany({ input }, { req, res }, comment_findMany);
                expect(result).to.not.be.null;
                expect(result.threads).to.have.lengthOf(1);
            });
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates a comment for authenticated user", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user2Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const newCommentId = uuid();
                const translationId = uuid();
                const input: CommentCreateInput = {
                    id: newCommentId,
                    createdFor: CommentFor.Post,
                    forConnect: post.id,
                    translationsCreate: [{ id: translationId, language: "en", text: "New comment text" }],
                };
                const result = await comment.createOne({ input }, { req, res }, comment_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newCommentId);
                expect(result.translations?.[0]?.text).to.equal(input.translationsCreate![0].text);
            });

            it("API key with write permissions can create comment", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user2Id };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const newCommentId = uuid();
                const translationId = uuid();
                const input: CommentCreateInput = {
                    id: newCommentId,
                    createdFor: CommentFor.Post,
                    forConnect: post.id,
                    translationsCreate: [{ id: translationId, language: "en", text: "API created comment" }],
                };
                const result = await comment.createOne({ input }, { req, res }, comment_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newCommentId);
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create comment", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: CommentCreateInput = {
                    id: uuid(),
                    createdFor: CommentFor.Post,
                    forConnect: post.id,
                    translationsCreate: [{ id: uuid(), language: "en", text: "Unauthorized" }],
                };
                try {
                    await comment.createOne({ input }, { req, res }, comment_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) {
                    // Error expected
                }
            });

            it("API key without write permissions cannot create comment", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user2Id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: CommentCreateInput = {
                    id: uuid(),
                    createdFor: CommentFor.Post,
                    forConnect: post.id,
                    translationsCreate: [{ id: uuid(), language: "en", text: "Unauthorized" }],
                };
                try {
                    await comment.createOne({ input }, { req, res }, comment_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) {
                    // Error expected
                }
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("allows admin to update a comment", async () => {
                const adminUser = { ...loggedInUserNoPremiumData, id: SEEDED_IDS.User.Admin };
                const { req, res } = await mockAuthenticatedSession(adminUser);
                const input: CommentUpdateInput = {
                    id: comment1.id,
                    translationsUpdate: [{ id: comment1.translations[0].id, language: "en", text: "Updated comment" }],
                };
                const result = await comment.updateOne({ input }, { req, res }, comment_updateOne);
                expect(result).to.not.be.null;
                expect(result.translations?.[0]?.text).to.equal("Updated comment");
            });
        });

        describe("invalid", () => {
            it("denies update for non-admin user", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: CommentUpdateInput = {
                    id: comment1.id,
                    translationsUpdate: [{ id: comment1.translations[0].id, language: "en", text: "Should fail" }],
                };
                try {
                    await comment.updateOne({ input }, { req, res }, comment_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) {
                    // Error expected
                }
            });

            it("denies update for not logged in user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: CommentUpdateInput = {
                    id: comment1.id,
                    translationsUpdate: [{ id: comment1.translations[0].id, language: "en", text: "Should fail" }],
                };
                try {
                    await comment.updateOne({ input }, { req, res }, comment_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) {
                    // Error expected
                }
            });

            it("throws when updating non-existent comment as admin", async () => {
                const adminUser = { ...loggedInUserNoPremiumData, id: SEEDED_IDS.User.Admin };
                const { req, res } = await mockAuthenticatedSession(adminUser);
                const input: CommentUpdateInput = {
                    id: uuid(),
                    translationsUpdate: [{ id: uuid(), language: "en", text: "No such comment" }],
                };
                try {
                    await comment.updateOne({ input }, { req, res }, comment_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) {
                    // Error expected
                }
            });
        });
    });
});
