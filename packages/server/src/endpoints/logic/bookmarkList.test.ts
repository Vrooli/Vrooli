import { type BookmarkListCreateInput, type BookmarkListSearchInput, type BookmarkListUpdateInput, type FindByIdInput } from "@vrooli/shared";
import { bookmarkListTestDataFactory } from "@vrooli/shared/validation/models";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { BookmarkListDbFactory } from "../../__test/fixtures/db/bookmarkFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
import { assertFindManyResultIds } from "../../__test/helpers.js";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPrivatePermissions, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { bookmarkList_createOne } from "../generated/bookmarkList_createOne.js";
import { bookmarkList_findMany } from "../generated/bookmarkList_findMany.js";
import { bookmarkList_findOne } from "../generated/bookmarkList_findOne.js";
import { bookmarkList_updateOne } from "../generated/bookmarkList_updateOne.js";
import { bookmarkList } from "./bookmarkList.js";

describe("EndpointsBookmarkList", () => {
    let testUsers: any[];
    let listUser1: any;
    let listUser2: any;

    beforeAll(() => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Reset Redis and database tables
        await CacheService.get().flushAll();
        await DbProvider.deleteAll();

        // Seed test users using database fixtures
        testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

        // Seed bookmark lists using database fixtures
        listUser1 = await DbProvider.get().bookmarkList.create({
            data: BookmarkListDbFactory.createWithTranslations(
                testUsers[0].id,
                [{ language: "en", name: "List One" }]
            ),
        });
        listUser2 = await DbProvider.get().bookmarkList.create({
            data: BookmarkListDbFactory.createWithTranslations(
                testUsers[1].id,
                [{ language: "en", name: "List Two" }]
            ),
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
        describe("valid", () => {
            it("returns own bookmark list", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });
                const input: FindByIdInput = { id: listUser1.id };
                const result = await bookmarkList.findOne({ input }, { req, res }, bookmarkList_findOne);
                expect(result).not.toBeNull();
                expect(result.id).toEqual(listUser1.id);
            });

            it("API key cannot find another user's list", async () => {
                const permissions = mockReadPrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });
                const input: FindByIdInput = { id: listUser2.id }; // Try to access listUser2

                await expect(async () => {
                    await bookmarkList.findOne({ input }, { req, res }, bookmarkList_findOne);
                }).rejects.toThrow("Unauthorized");
            });

            it("logged out user cannot find any list", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: listUser2.id };

                await expect(async () => {
                    await bookmarkList.findOne({ input }, { req, res }, bookmarkList_findOne);
                }).rejects.toThrow("Unauthorized");
            });
        });

        describe("invalid", () => {
            it("does not return another user's list", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });
                const input: FindByIdInput = { id: listUser2.id };

                await expect(async () => {
                    await bookmarkList.findOne({ input }, { req, res }, bookmarkList_findOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("findMany", () => {
        it("returns only own lists for authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id
            });
            const input: BookmarkListSearchInput = { take: 10 };
            const expectedIds = [
                listUser1.id,
            ];
            const result = await bookmarkList.findMany({ input }, { req, res }, bookmarkList_findMany);
            expect(result).not.toBeNull();
            expect(result.edges).toBeDefined();
            assertFindManyResultIds(expect, result, expectedIds);
        });

        it("API key with public read returns no lists (as they are private)", async () => {
            const permissions = mockReadPublicPermissions();
            const apiToken = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(apiToken, permissions, {
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id
            });
            const input: BookmarkListSearchInput = { take: 10 };

            await expect(async () => {
                await bookmarkList.findMany({ input }, { req, res }, bookmarkList_findMany);
            }).rejects.toThrow("0782");
        });

        it("logged out user returns no lists (as they are private)", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: BookmarkListSearchInput = { take: 10 };

            await expect(async () => {
                await bookmarkList.findMany({ input }, { req, res }, bookmarkList_findMany);
            }).rejects.toThrow("0782");
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates a bookmark list for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });

                // Use validation fixtures for API input
                const input: BookmarkListCreateInput = bookmarkListTestDataFactory.createMinimal({
                    translationsCreate: [{
                        language: "en",
                        name: "New List",
                        description: "Test bookmark list",
                    }],
                });

                const result = await bookmarkList.createOne({ input }, { req, res }, bookmarkList_createOne);
                expect(result).not.toBeNull();
                expect(result.id).toBeDefined();
                expect(result.translations).toHaveLength(1);
                expect(result.translations[0].name).toBe("New List");
            });

            it("API key with write permissions can create list", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });

                // Use validation fixtures for API input
                const input: BookmarkListCreateInput = bookmarkListTestDataFactory.createComplete({
                    translationsCreate: [{
                        language: "en",
                        name: "API List",
                    }],
                });

                const result = await bookmarkList.createOne({ input }, { req, res }, bookmarkList_createOne);
                expect(result).not.toBeNull();
                expect(result.id).toBeDefined();
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create list", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: BookmarkListCreateInput = bookmarkListTestDataFactory.createMinimal({
                    translationsCreate: [{
                        language: "en",
                        name: "NoAuth",
                    }],
                });

                await expect(async () => {
                    await bookmarkList.createOne({ input }, { req, res }, bookmarkList_createOne);
                }).rejects.toThrow();
            });

            it("API key without write permissions cannot create list", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });

                const input: BookmarkListCreateInput = bookmarkListTestDataFactory.createMinimal({
                    translationsCreate: [{
                        language: "en",
                        name: "BadAPI",
                    }],
                });

                await expect(async () => {
                    await bookmarkList.createOne({ input }, { req, res }, bookmarkList_createOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("updates own bookmark list", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });

                const input: BookmarkListUpdateInput = {
                    id: listUser1.id,
                    translationsUpdate: [{
                        id: listUser1.translations[0].id,
                        name: "Updated Label",
                    }],
                };

                const result = await bookmarkList.updateOne({ input }, { req, res }, bookmarkList_updateOne);
                expect(result).not.toBeNull();
                expect(result.id).toEqual(listUser1.id);
                expect(result.translations[0].name).toBe("Updated Label");
            });

            it("API key with write permissions can update list", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });

                const input: BookmarkListUpdateInput = {
                    id: listUser1.id,
                    translationsUpdate: [{
                        id: listUser1.translations[0].id,
                        name: "API Updated",
                    }],
                };

                const result = await bookmarkList.updateOne({ input }, { req, res }, bookmarkList_updateOne);
                expect(result.translations[0].name).toBe("API Updated");
            });
        });

        describe("invalid", () => {
            it("cannot update another user's list", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });

                const input: BookmarkListUpdateInput = {
                    id: listUser2.id,
                    translationsUpdate: [{
                        id: listUser2.translations[0].id,
                        name: "Hacked",
                    }],
                };

                await expect(async () => {
                    await bookmarkList.updateOne({ input }, { req, res }, bookmarkList_updateOne);
                }).rejects.toThrow();
            });

            it("not logged in user cannot update", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: BookmarkListUpdateInput = {
                    id: listUser1.id,
                    translationsUpdate: [{
                        id: listUser1.translations[0].id,
                        name: "NoAuthUpdate",
                    }],
                };

                await expect(async () => {
                    await bookmarkList.updateOne({ input }, { req, res }, bookmarkList_updateOne);
                }).rejects.toThrow();
            });
        });
    });
}); 
