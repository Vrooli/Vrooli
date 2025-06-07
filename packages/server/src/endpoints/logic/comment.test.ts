import { type CommentCreateInput, CommentFor, type CommentSearchInput, type CommentUpdateInput, type FindByIdInput } from "@vrooli/shared";
import { describe, it, expect, beforeAll, afterAll, beforeEach, vi } from "vitest";
import { defaultPublicUserData, loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions, seedMockAdminUser } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { comment_createOne } from "../generated/comment_createOne.js";
import { comment_findMany } from "../generated/comment_findMany.js";
import { comment_findOne } from "../generated/comment_findOne.js";
import { comment_updateOne } from "../generated/comment_updateOne.js";
import { comment } from "./comment.js";

// Import database fixtures for seeding
import { CommentDbFactory, seedCommentThread } from "../../__test/fixtures/commentFixtures.js";
import { UserDbFactory, seedTestUsers } from "../../__test/fixtures/userFixtures.js";

// Import validation fixtures for API input testing
import { commentTestDataFactory } from "@vrooli/shared/src/validation/models/__test__/fixtures/commentFixtures.js";

describe("EndpointsComment", () => {
    let testUsers: any[];
    let adminUser: any;
    let issue: any;
    let comments: any[];

    beforeAll(() => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Clear databases
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Seed test users using database fixtures
        testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });
        
        // Ensure admin user exists for update tests
        adminUser = await seedMockAdminUser();

        // Create a public issue to comment on using database fixtures
        issue = await DbProvider.get().issue.create({
            data: {
                id: UserDbFactory.createMinimal().id, // Generate unique ID
                publicId: UserDbFactory.createMinimal().publicId,
                createdBy: { connect: { id: testUsers[0].id } },
            },
        });

        // Seed comment thread using database fixtures
        comments = await seedCommentThread(DbProvider.get(), {
            createdById: testUsers[0].id,
            objectId: issue.id,
            objectType: "Issue",
            commentCount: 2,
            withReplies: true,
        });
    });

    afterAll(async () => {
        // Clean up
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("findOne", () => {
        it("returns comment by id for any authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({ 
                ...loggedInUserNoPremiumData(), 
                id: testUsers[1].id 
            });
            const input: FindByIdInput = { id: comments[0].id };
            const result = await comment.findOne({ input }, { req, res }, comment_findOne);
            expect(result).not.toBeNull();
            expect(result.id).toEqual(comments[0].id);
        });

        it("returns comment by id when not authenticated", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: FindByIdInput = { id: comments[0].id };
            const result = await comment.findOne({ input }, { req, res }, comment_findOne);
            expect(result).not.toBeNull();
            expect(result.id).toEqual(comments[0].id);
        });

        it("returns comment by id with API key public read", async () => {
            const permissions = mockReadPublicPermissions();
            const apiToken = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(apiToken, permissions, { 
                ...loggedInUserNoPremiumData(), 
                id: testUsers[0].id 
            });
            const input: FindByIdInput = { id: comments[0].id };
            const result = await comment.findOne({ input }, { req, res }, comment_findOne);
            expect(result).not.toBeNull();
            expect(result.id).toEqual(comments[0].id);
        });
    });

    describe("findMany", () => {
        it("returns comments for issue", async () => {
            const { req, res } = await mockAuthenticatedSession({ 
                ...loggedInUserNoPremiumData(), 
                id: testUsers[0].id 
            });
            const input: CommentSearchInput = { 
                forObjectType: CommentFor.Issue,
                forIds: [issue.id],
                take: 20,
            };
            const result = await comment.findMany({ input }, { req, res }, comment_findMany);
            expect(result).not.toBeNull();
            expect(result.edges).toBeInstanceOf(Array);
            expect(result.edges.length).toBeGreaterThan(0);
            
            // Check that we get comments for the issue
            const issueComments = result.edges.filter((edge: any) => 
                edge.node.issueId === issue.id
            );
            expect(issueComments.length).toBeGreaterThan(0);
        });

        it("returns comments for not authenticated user", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: CommentSearchInput = { 
                forObjectType: CommentFor.Issue,
                forIds: [issue.id],
                take: 10,
            };
            const result = await comment.findMany({ input }, { req, res }, comment_findMany);
            expect(result).not.toBeNull();
            expect(result.edges).toBeInstanceOf(Array);
        });

        it("returns comments with API key public read", async () => {
            const permissions = mockReadPublicPermissions();
            const apiToken = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(apiToken, permissions, { 
                ...loggedInUserNoPremiumData(), 
                id: testUsers[0].id 
            });
            const input: CommentSearchInput = { take: 10 };
            const result = await comment.findMany({ input }, { req, res }, comment_findMany);
            expect(result).not.toBeNull();
            expect(result.edges).toBeInstanceOf(Array);
        });
    });

    describe("createOne", () => {
        it("creates a comment for authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({ 
                ...loggedInUserNoPremiumData(), 
                id: testUsers[0].id 
            });
            
            // Use validation fixtures for API input
            const input: CommentCreateInput = commentTestDataFactory.createMinimal({
                forConnect: issue.id,
                translationsCreate: [{
                    language: "en",
                    text: "This is a new comment",
                }],
            });
            
            const result = await comment.createOne({ input }, { req, res }, comment_createOne);
            expect(result).not.toBeNull();
            expect(result.translations).toHaveLength(1);
            expect(result.translations[0].text).toBe("This is a new comment");
        });

        it("creates a reply comment for authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({ 
                ...loggedInUserNoPremiumData(), 
                id: testUsers[1].id 
            });
            
            // Use validation fixtures for creating a reply
            const input: CommentCreateInput = commentTestDataFactory.createComplete({
                forConnect: issue.id,
                parentConnect: comments[0].id,
                translationsCreate: [{
                    language: "en",
                    text: "This is a reply to the first comment",
                }],
            });
            
            const result = await comment.createOne({ input }, { req, res }, comment_createOne);
            expect(result).not.toBeNull();
            expect(result.parentId).toEqual(comments[0].id);
        });

        it("throws error for not logged in user", async () => {
            const { req, res } = await mockLoggedOutSession();
            
            const input: CommentCreateInput = commentTestDataFactory.createMinimal({
                forConnect: issue.id,
            });
            
            await expect(async () => {
                await comment.createOne({ input }, { req, res }, comment_createOne);
            }).rejects.toThrow();
        });
    });

    describe("updateOne", () => {
        it("updates comment for owner", async () => {
            const { req, res } = await mockAuthenticatedSession({ 
                ...loggedInUserNoPremiumData(), 
                id: testUsers[0].id 
            });
            
            // Use validation fixtures for update
            const input: CommentUpdateInput = {
                id: comments[0].id,
                translationsUpdate: [{
                    id: comments[0].translations[0].id,
                    text: "Updated comment text",
                }],
            };
            
            const result = await comment.updateOne({ input }, { req, res }, comment_updateOne);
            expect(result).not.toBeNull();
            expect(result.translations[0].text).toBe("Updated comment text");
        });

        it("admin can update any comment", async () => {
            const { req, res } = await mockAuthenticatedSession({ 
                ...loggedInUserNoPremiumData(), 
                id: adminUser.id 
            });
            
            const input: CommentUpdateInput = {
                id: comments[0].id,
                translationsUpdate: [{
                    id: comments[0].translations[0].id,
                    text: "Admin updated this comment",
                }],
            };
            
            const result = await comment.updateOne({ input }, { req, res }, comment_updateOne);
            expect(result).not.toBeNull();
            expect(result.translations[0].text).toBe("Admin updated this comment");
        });

        it("throws error for not logged in user", async () => {
            const { req, res } = await mockLoggedOutSession();
            
            const input: CommentUpdateInput = {
                id: comments[0].id,
                translationsUpdate: [{
                    id: comments[0].translations[0].id,
                    text: "Should not update",
                }],
            };
            
            await expect(async () => {
                await comment.updateOne({ input }, { req, res }, comment_updateOne);
            }).rejects.toThrow();
        });
    });
});