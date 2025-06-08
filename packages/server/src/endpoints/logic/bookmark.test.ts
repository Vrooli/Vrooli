import { type BookmarkCreateInput, BookmarkFor, type BookmarkSearchInput, bookmarkTestDataFactory, type BookmarkUpdateInput, type FindByIdInput, generatePK } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { BookmarkListDbFactory, seedBookmarks } from "../../__test/fixtures/db/bookmarkFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
import { loggedInUserNoPremiumData, mockAuthenticatedSession, mockLoggedOutSession, seedMockAdminUser } from "../../__test/session.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { bookmark_createOne } from "../generated/bookmark_createOne.js";
import { bookmark_findMany } from "../generated/bookmark_findMany.js";
import { bookmark_findOne } from "../generated/bookmark_findOne.js";
import { bookmark_updateOne } from "../generated/bookmark_updateOne.js";
import { bookmark } from "./bookmark.js";

describe("EndpointsBookmark", () => {
    let testUsers: any[];
    let adminUser: any;
    let tags: any[];
    let bookmarkData: any;

    beforeAll(() => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        await CacheService.get().flushAll();
        await DbProvider.deleteAll();

        // Seed test users using database fixtures
        testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });
        adminUser = await seedMockAdminUser();

        // Create tags to be bookmarked
        tags = await Promise.all([
            DbProvider.get().tag.create({ data: { id: generatePK(), tag: "tag-1" } }),
            DbProvider.get().tag.create({ data: { id: generatePK(), tag: "tag-2" } }),
        ]);

        // Seed bookmarks with lists using database fixtures
        bookmarkData = await seedBookmarks(DbProvider.get(), {
            userId: testUsers[0].id,
            objects: [
                { id: tags[0].id, type: "Tag" },
                { id: tags[1].id, type: "Tag" },
            ],
            withList: true,
            listName: "My Tag Collection",
        });
    });

    afterAll(async () => {
        // Clean up
        await CacheService.get().flushAll();
        await DbProvider.deleteAll();

        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("findOne", () => {
        it("returns bookmark by id for owner", async () => {
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id
            });
            const input: FindByIdInput = { id: bookmarkData.bookmarks[0].id };
            const result = await bookmark.findOne({ input }, { req, res }, bookmark_findOne);
            expect(result).not.toBeNull();
            expect(result.id).toEqual(bookmarkData.bookmarks[0].id);
        });

        it("does not return bookmark for non-owner", async () => {
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[1].id
            });
            const input: FindByIdInput = { id: bookmarkData.bookmarks[0].id };

            await expect(async () => {
                await bookmark.findOne({ input }, { req, res }, bookmark_findOne);
            }).rejects.toThrow();
        });

        it("throws error when not authenticated", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: FindByIdInput = { id: bookmarkData.bookmarks[0].id };

            await expect(async () => {
                await bookmark.findOne({ input }, { req, res }, bookmark_findOne);
            }).rejects.toThrow();
        });
    });

    describe("findMany", () => {
        it("returns bookmarks for authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id
            });
            const input: BookmarkSearchInput = {
                forObjectType: BookmarkFor.Tag,
                take: 10,
            };
            const result = await bookmark.findMany({ input }, { req, res }, bookmark_findMany);
            expect(result).not.toBeNull();
            expect(result.edges).toBeInstanceOf(Array);
            expect(result.edges.length).toBe(2); // Should return user's 2 bookmarks
        });

        it("returns empty for user with no bookmarks", async () => {
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[1].id
            });
            const input: BookmarkSearchInput = { take: 10 };
            const result = await bookmark.findMany({ input }, { req, res }, bookmark_findMany);
            expect(result).not.toBeNull();
            expect(result.edges).toHaveLength(0);
        });

        it("throws error when not authenticated", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: BookmarkSearchInput = { take: 10 };

            await expect(async () => {
                await bookmark.findMany({ input }, { req, res }, bookmark_findMany);
            }).rejects.toThrow();
        });
    });

    describe("createOne", () => {
        it("creates a bookmark for authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[1].id
            });

            // Use validation fixtures for API input
            const input: BookmarkCreateInput = bookmarkTestDataFactory.createMinimal({
                forConnect: tags[0].id,
            });

            const result = await bookmark.createOne({ input }, { req, res }, bookmark_createOne);
            expect(result).not.toBeNull();
            expect(result.tagId).toEqual(tags[0].id);
            expect(result.byId).toEqual(testUsers[1].id);
        });

        it("creates a bookmark with list", async () => {
            // First create a list for the user
            const list = await DbProvider.get().bookmark_list.create({
                data: BookmarkListDbFactory.createMinimal(testUsers[1].id, {
                    label: "New List",
                }),
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[1].id
            });

            // Use validation fixtures with list connection
            const input: BookmarkCreateInput = bookmarkTestDataFactory.createComplete({
                forConnect: tags[1].id,
                listConnect: list.id,
            });

            const result = await bookmark.createOne({ input }, { req, res }, bookmark_createOne);
            expect(result).not.toBeNull();
            expect(result.listId).toEqual(list.id);
            expect(result.tagId).toEqual(tags[1].id);
        });

        it("throws error for not logged in user", async () => {
            const { req, res } = await mockLoggedOutSession();

            const input: BookmarkCreateInput = bookmarkTestDataFactory.createMinimal({
                forConnect: tags[0].id,
            });

            await expect(async () => {
                await bookmark.createOne({ input }, { req, res }, bookmark_createOne);
            }).rejects.toThrow();
        });
    });

    describe("updateOne", () => {
        it("updates bookmark list for owner", async () => {
            // Create a new list for the update
            const newList = await DbProvider.get().bookmark_list.create({
                data: BookmarkListDbFactory.createMinimal(testUsers[0].id, {
                    label: "Updated List",
                }),
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id
            });

            const input: BookmarkUpdateInput = {
                id: bookmarkData.bookmarks[0].id,
                listConnect: newList.id,
            };

            const result = await bookmark.updateOne({ input }, { req, res }, bookmark_updateOne);
            expect(result).not.toBeNull();
            expect(result.listId).toEqual(newList.id);
        });

        it("disconnects bookmark from list", async () => {
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id
            });

            const input: BookmarkUpdateInput = {
                id: bookmarkData.bookmarks[0].id,
                listDisconnect: true,
            };

            const result = await bookmark.updateOne({ input }, { req, res }, bookmark_updateOne);
            expect(result).not.toBeNull();
            expect(result.listId).toBeNull();
        });

        it("throws error for non-owner", async () => {
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[1].id
            });

            const input: BookmarkUpdateInput = {
                id: bookmarkData.bookmarks[0].id,
                listDisconnect: true,
            };

            await expect(async () => {
                await bookmark.updateOne({ input }, { req, res }, bookmark_updateOne);
            }).rejects.toThrow();
        });

        it("throws error for not logged in user", async () => {
            const { req, res } = await mockLoggedOutSession();

            const input: BookmarkUpdateInput = {
                id: bookmarkData.bookmarks[0].id,
                listDisconnect: true,
            };

            await expect(async () => {
                await bookmark.updateOne({ input }, { req, res }, bookmark_updateOne);
            }).rejects.toThrow();
        });
    });
});