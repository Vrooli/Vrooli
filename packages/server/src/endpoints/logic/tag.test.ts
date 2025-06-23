import { type FindByIdInput, type TagCreateInput, type TagSearchInput, type TagUpdateInput } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { assertFindManyResultIds } from "../../__test/helpers.js";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions, seedMockAdminUser } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { tag_createOne } from "../generated/tag_createOne.js";
import { tag_findMany } from "../generated/tag_findMany.js";
import { tag_findOne } from "../generated/tag_findOne.js";
import { tag_updateOne } from "../generated/tag_updateOne.js";
import { tag } from "./tag.js";
// Import database fixtures for seeding
import { seedTags } from "../../__test/fixtures/db/tagFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
// Import validation fixtures for API input testing
import { tagTestDataFactory } from "@vrooli/shared";

describe("EndpointsTag", () => {
    let testUsers: any[];
    let adminUser: any;
    let tags: any[];

    beforeAll(() => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Clean up tables used in tests
        try {
            const prisma = DbProvider.get();
            if (prisma) {
                testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });
            }
        } catch (error) {
            // If database is not initialized, skip cleanup
        }
        
        adminUser = await seedMockAdminUser();

        // Seed tags using database fixtures
        tags = await seedTags(DbProvider.get(), [
            "programming",
            "javascript",
            "testing",
            "tutorial",
        ], {
            withTranslations: true,
            popular: true,
        });

        // Assign ownership to some tags
        await DbProvider.get().tag.update({
            where: { id: tags[0].id },
            data: { createdBy: { connect: { id: testUsers[0].id } } },
        });
        await DbProvider.get().tag.update({
            where: { id: tags[1].id },
            data: { createdBy: { connect: { id: testUsers[1].id } } },
        });
    });

    afterAll(async () => {
        await CacheService.get().flushAll();
        await DbProvider.deleteAll();

        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns tag by id for any authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });
                const input: FindByIdInput = { id: tags[0].id };
                const result = await tag.findOne({ input }, { req, res }, tag_findOne);
                expect(result).not.toBeNull();
                expect(result.id).toEqual(tags[0].id);
                expect(result.tag).toBe("programming");
            });

            it("returns tag by id when not authenticated", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: tags[1].id };
                const result = await tag.findOne({ input }, { req, res }, tag_findOne);
                expect(result).not.toBeNull();
                expect(result.id).toEqual(tags[1].id);
                expect(result.tag).toBe("javascript");
            });

            it("returns tag by id with API key public read", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });
                const input: FindByIdInput = { id: tags[2].id };
                const result = await tag.findOne({ input }, { req, res }, tag_findOne);
                expect(result).not.toBeNull();
                expect(result.id).toEqual(tags[2].id);
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns tags without filters for any authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });
                const input: TagSearchInput = { take: 10 };
                const expectedIds = tags.map(t => t.id);
                const result = await tag.findMany({ input }, { req, res }, tag_findMany);
                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
                assertFindManyResultIds(expect, result, expectedIds);
            });

            it("returns tags with search term", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });
                const input: TagSearchInput = {
                    searchString: "script",
                    take: 10,
                };
                const result = await tag.findMany({ input }, { req, res }, tag_findMany);
                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
                expect(result.edges.length).toBeGreaterThan(0);
                expect(result.edges.some((edge: any) => edge.node.tag === "javascript")).toBe(true);
            });

            it("returns tags without filters for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: TagSearchInput = { take: 10 };
                const result = await tag.findMany({ input }, { req, res }, tag_findMany);
                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
            });

            it("returns tags without filters for API key public read", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });
                const input: TagSearchInput = { take: 10 };
                const result = await tag.findMany({ input }, { req, res }, tag_findMany);
                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
            });
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates a tag for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                // Use validation fixtures for API input
                const input: TagCreateInput = tagTestDataFactory.createMinimal({
                    tag: "newtag",
                    translationsCreate: [{
                        language: "en",
                        description: "A new tag for testing",
                    }],
                });

                const result = await tag.createOne({ input }, { req, res }, tag_createOne);
                expect(result).not.toBeNull();
                expect(result.tag).toBe("newtag");
                expect(result.translations).toHaveLength(1);
                expect(result.translations[0].description).toBe("A new tag for testing");
            });

            it("API key with write permissions can create tag", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                // Use complete fixture for comprehensive test
                const input: TagCreateInput = tagTestDataFactory.createComplete({
                    tag: "apitag",
                });

                const result = await tag.createOne({ input }, { req, res }, tag_createOne);
                expect(result).not.toBeNull();
                expect(result.tag).toBe("apitag");
            });
        });

        describe("invalid", () => {
            it("throws error for duplicate tag", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                // Try to create a tag that already exists
                const input: TagCreateInput = tagTestDataFactory.createMinimal({
                    tag: "programming", // Already exists
                });

                await expect(async () => {
                    await tag.createOne({ input }, { req, res }, tag_createOne);
                }).rejects.toThrow();
            });

            it("throws error for not logged in user", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: TagCreateInput = tagTestDataFactory.createMinimal({
                    tag: "unauthorized",
                });

                await expect(async () => {
                    await tag.createOne({ input }, { req, res }, tag_createOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("updates tag translations for owner", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input: TagUpdateInput = {
                    id: tags[0].id,
                    translationsUpdate: [{
                        id: tags[0].translations[0].id,
                        description: "Updated description for programming",
                    }],
                };

                const result = await tag.updateOne({ input }, { req, res }, tag_updateOne);
                expect(result).not.toBeNull();
                expect(result.translations[0].description).toBe("Updated description for programming");
            });

            it("admin can update any tag", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: adminUser.id,
                });

                const input: TagUpdateInput = {
                    id: tags[2].id,
                    translationsCreate: [{
                        language: "fr",
                        description: "Tests en français",
                    }],
                };

                const result = await tag.updateOne({ input }, { req, res }, tag_updateOne);
                expect(result).not.toBeNull();
                expect(result.translations.length).toBeGreaterThan(1);
            });
        });

        describe("invalid", () => {
            it("throws error for non-owner", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[1].id,
                });

                const input: TagUpdateInput = {
                    id: tags[0].id, // Owned by testUsers[0]
                    translationsUpdate: [{
                        id: tags[0].translations[0].id,
                        description: "Should not update",
                    }],
                };

                await expect(async () => {
                    await tag.updateOne({ input }, { req, res }, tag_updateOne);
                }).rejects.toThrow();
            });

            it("throws error for not logged in user", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: TagUpdateInput = {
                    id: tags[0].id,
                    translationsUpdate: [{
                        id: tags[0].translations[0].id,
                        description: "Should not update",
                    }],
                };

                await expect(async () => {
                    await tag.updateOne({ input }, { req, res }, tag_updateOne);
                }).rejects.toThrow();
            });
        });
    });
});
