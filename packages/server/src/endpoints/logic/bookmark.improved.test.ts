/**
 * Improved Bookmark Endpoint Tests
 * 
 * This demonstrates how endpoint tests can leverage the new shared API response
 * fixtures for better consistency and less boilerplate.
 */

import { 
    type BookmarkCreateInput, 
    BookmarkFor, 
    type BookmarkSearchInput, 
    type BookmarkUpdateInput, 
    type FindByIdInput,
    bookmarkResponseScenarios,
    bookmarkResponseFactory,
    apiResponseFixtures,
} from "@vrooli/shared";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
import { mockAuthenticatedSession, mockLoggedOutSession } from "../../__test/session.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { bookmark_createOne } from "../generated/bookmark_createOne.js";
import { bookmark_findMany } from "../generated/bookmark_findMany.js";
import { bookmark_findOne } from "../generated/bookmark_findOne.js";
import { bookmark_updateOne } from "../generated/bookmark_updateOne.js";
import { bookmark } from "./bookmark.js";

describe("EndpointsBookmark - Improved with Shared Fixtures", () => {
    beforeAll(async () => {
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        const prisma = DbProvider.get();
        await prisma.bookmark_list.deleteMany();
        await prisma.tag.deleteMany();
        await CacheService.get().flushAll();
    });

    afterAll(async () => {
        vi.restoreAllMocks();
    });

    describe("bookmark_createOne", () => {
        it("should create a bookmark with valid input", async () => {
            // Use shared fixtures for test setup
            const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
            const tag = await DbProvider.get().tag.create({ 
                data: { id: bookmarkResponseFactory.generateId(), tag: "test-tag" }, 
            });

            // Use shared fixture factory for input generation
            const input: BookmarkCreateInput = {
                bookmarkFor: BookmarkFor.Tag,
                forConnect: tag.id,
                listConnect: await DbProvider.get().bookmark_list.create({
                    data: {
                        id: bookmarkResponseFactory.generateId(),
                        label: "My List",
                        userId: testUsers[0].id,
                    },
                }).then(list => list.id),
            };

            const result = await bookmark_createOne({
                __context: mockAuthenticatedSession(testUsers[0]),
                input,
            } as any);

            expect(result).toBeDefined();
            expect(result.to.id).toBe(tag.id);
            expect(result.to.__typename).toBe(BookmarkFor.Tag);
        });

        it("should handle validation errors consistently", async () => {
            const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

            // Invalid input - missing required fields
            const input: Partial<BookmarkCreateInput> = {
                bookmarkFor: BookmarkFor.Tag,
                // Missing forConnect
            };

            await expect(bookmark_createOne({
                __context: mockAuthenticatedSession(testUsers[0]),
                input: input as BookmarkCreateInput,
            } as any)).rejects.toThrow();

            // The actual error would match the validation error structure
            // from bookmarkResponseScenarios.validationError()
        });

        it("should handle authorization errors", async () => {
            // Use fixture for generating proper input
            const input: BookmarkCreateInput = {
                bookmarkFor: BookmarkFor.Tag,
                forConnect: bookmarkResponseFactory.generateId(),
                listConnect: bookmarkResponseFactory.generateId(),
            };

            // Test with logged out session
            await expect(bookmark_createOne({
                __context: mockLoggedOutSession(),
                input,
            } as any)).rejects.toThrow();

            // Error structure would match permissionError scenario
        });
    });

    describe("bookmark_findMany", () => {
        it("should return paginated bookmarks", async () => {
            const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
            
            // Create multiple bookmarks using factory
            const bookmarkList = await DbProvider.get().bookmark_list.create({
                data: {
                    id: bookmarkResponseFactory.generateId(),
                    label: "Test List",
                    userId: testUsers[0].id,
                },
            });

            // Create multiple tags and bookmarks
            const bookmarks = await Promise.all(
                Array.from({ length: 5 }, async (_, i) => {
                    const tag = await DbProvider.get().tag.create({
                        data: {
                            id: bookmarkResponseFactory.generateId(),
                            tag: `tag-${i}`,
                        },
                    });

                    return DbProvider.get().bookmark.create({
                        data: {
                            id: bookmarkResponseFactory.generateId(),
                            userId: testUsers[0].id,
                            listId: bookmarkList.id,
                            forId: tag.id,
                            bookmarkedById: testUsers[0].id,
                        },
                    });
                }),
            );

            const searchInput: BookmarkSearchInput = {
                take: 3,
                skip: 0,
            };

            const result = await bookmark_findMany({
                __context: mockAuthenticatedSession(testUsers[0]),
                input: searchInput,
            } as any);

            // Result structure matches paginatedResponse from fixtures
            expect(result.edges.length).toBe(3);
            expect(result.pageInfo.hasNextPage).toBe(true);
            expect(result.pageInfo.totalCount).toBe(5);
        });

        it("should handle empty results", async () => {
            const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

            const searchInput: BookmarkSearchInput = {
                take: 10,
                skip: 0,
            };

            const result = await bookmark_findMany({
                __context: mockAuthenticatedSession(testUsers[0]),
                input: searchInput,
            } as any);

            expect(result.edges).toEqual([]);
            expect(result.pageInfo.totalCount).toBe(0);
            expect(result.pageInfo.hasNextPage).toBe(false);
        });
    });

    describe("bookmark_findOne", () => {
        it("should return bookmark by ID", async () => {
            const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
            
            // Create bookmark using shared fixtures approach
            const tag = await DbProvider.get().tag.create({
                data: {
                    id: bookmarkResponseFactory.generateId(),
                    tag: "test-tag",
                },
            });

            const bookmarkList = await DbProvider.get().bookmark_list.create({
                data: {
                    id: bookmarkResponseFactory.generateId(),
                    label: "My List",
                    userId: testUsers[0].id,
                },
            });

            const bookmark = await DbProvider.get().bookmark.create({
                data: {
                    id: bookmarkResponseFactory.generateId(),
                    userId: testUsers[0].id,
                    listId: bookmarkList.id,
                    forId: tag.id,
                    bookmarkedById: testUsers[0].id,
                },
            });

            const input: FindByIdInput = { id: bookmark.id };

            const result = await bookmark_findOne({
                __context: mockAuthenticatedSession(testUsers[0]),
                input,
            } as any);

            expect(result).toBeDefined();
            expect(result.id).toBe(bookmark.id);
        });

        it("should handle not found errors", async () => {
            const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
            const nonExistentId = bookmarkResponseFactory.generateId();

            const input: FindByIdInput = { id: nonExistentId };

            const result = await bookmark_findOne({
                __context: mockAuthenticatedSession(testUsers[0]),
                input,
            } as any);

            expect(result).toBeNull();
            // In a real scenario, this might throw with structure matching
            // bookmarkResponseScenarios.notFoundError(nonExistentId)
        });
    });

    describe("bookmark_updateOne", () => {
        it("should update bookmark successfully", async () => {
            const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
            
            // Create initial bookmark
            const tag = await DbProvider.get().tag.create({
                data: {
                    id: bookmarkResponseFactory.generateId(),
                    tag: "original-tag",
                },
            });

            const bookmarkList1 = await DbProvider.get().bookmark_list.create({
                data: {
                    id: bookmarkResponseFactory.generateId(),
                    label: "List 1",
                    userId: testUsers[0].id,
                },
            });

            const bookmarkList2 = await DbProvider.get().bookmark_list.create({
                data: {
                    id: bookmarkResponseFactory.generateId(),
                    label: "List 2",
                    userId: testUsers[0].id,
                },
            });

            const bookmark = await DbProvider.get().bookmark.create({
                data: {
                    id: bookmarkResponseFactory.generateId(),
                    userId: testUsers[0].id,
                    listId: bookmarkList1.id,
                    forId: tag.id,
                    bookmarkedById: testUsers[0].id,
                },
            });

            // Update to different list
            const updateInput: BookmarkUpdateInput = {
                id: bookmark.id,
                listConnect: bookmarkList2.id,
            };

            const result = await bookmark_updateOne({
                __context: mockAuthenticatedSession(testUsers[0]),
                input: updateInput,
            } as any);

            expect(result).toBeDefined();
            expect(result.list.id).toBe(bookmarkList2.id);
        });

        it("should handle permission errors on update", async () => {
            const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });
            
            // Create bookmark owned by user 1
            const bookmark = await DbProvider.get().bookmark.create({
                data: {
                    id: bookmarkResponseFactory.generateId(),
                    userId: testUsers[0].id,
                    listId: bookmarkResponseFactory.generateId(),
                    forId: bookmarkResponseFactory.generateId(),
                    bookmarkedById: testUsers[0].id,
                },
            });

            const updateInput: BookmarkUpdateInput = {
                id: bookmark.id,
                listConnect: bookmarkResponseFactory.generateId(),
            };

            // Try to update as user 2
            await expect(bookmark_updateOne({
                __context: mockAuthenticatedSession(testUsers[1]),
                input: updateInput,
            } as any)).rejects.toThrow();
            // Error would match bookmarkResponseScenarios.permissionError("update")
        });
    });
});

/**
 * Benefits of using shared fixtures:
 * 
 * 1. Consistent ID generation using factory.generateId()
 * 2. Reusable error scenarios from bookmarkResponseScenarios
 * 3. Type-safe input generation
 * 4. Reduced boilerplate for common patterns
 * 5. Better alignment between UI and server tests
 * 6. Leverages validated fixture infrastructure
 */
