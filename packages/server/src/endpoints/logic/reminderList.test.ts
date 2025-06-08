import { type ReminderListCreateInput, type ReminderListUpdateInput } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { reminderList_createOne } from "../generated/reminderList_createOne.js";
import { reminderList_updateOne } from "../generated/reminderList_updateOne.js";
import { reminderList } from "./reminderList.js";

// Import database fixtures for seeding
import { ReminderListDbFactory } from "../../__test/fixtures/db/reminderFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";

// Import validation fixtures for API input testing
import { reminderListTestDataFactory } from "@vrooli/shared/validation/models";

describe("EndpointsReminderList", () => {
    let testUsers: any[];
    let reminderListUser1: any;
    let reminderListUser2: any;

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

        // Create reminder lists using database fixtures
        reminderListUser1 = await DbProvider.get().reminderList.create({
            data: ReminderListDbFactory.createMinimal(testUsers[0].id),
        });
        reminderListUser2 = await DbProvider.get().reminderList.create({
            data: ReminderListDbFactory.createMinimal(testUsers[1].id),
        });
    });

    afterAll(async () => {
        // Clean up
        await CacheService.get().flushAll();
        await DbProvider.deleteAll();

        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates a reminder list for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });

                // Use validation fixtures for API input
                const input: ReminderListCreateInput = reminderListTestDataFactory.createMinimal({});

                const creationResult = await reminderList.createOne({ input }, { req, res }, reminderList_createOne);

                expect(creationResult).not.toBeNull();
                expect(creationResult.id).toBeDefined();
                expect(creationResult.created_by).toEqual(testUsers[0].id);
            });

            it("API key with write permissions can create reminder list", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });
                // Use validation fixtures for API input
                const input: ReminderListCreateInput = reminderListTestDataFactory.createComplete({});

                const result = await reminderList.createOne({ input }, { req, res }, reminderList_createOne);

                expect(result).not.toBeNull();
                expect(result.id).toBeDefined();
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create reminder list", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: ReminderListCreateInput = reminderListTestDataFactory.createMinimal({});

                await expect(async () => {
                    await reminderList.createOne({ input }, { req, res }, reminderList_createOne);
                }).rejects.toThrow();
            });

            it("API key without write permissions cannot create reminder list", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });

                const input: ReminderListCreateInput = reminderListTestDataFactory.createMinimal({});

                await expect(async () => {
                    await reminderList.createOne({ input }, { req, res }, reminderList_createOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("updates a reminder list for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });

                const input: ReminderListUpdateInput = { id: reminderListUser1.id };

                const result = await reminderList.updateOne({ input }, { req, res }, reminderList_updateOne);

                expect(result).not.toBeNull();
                expect(result.id).toEqual(reminderListUser1.id);
            });

            it("API key with write permissions can update reminder list", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: ReminderListUpdateInput = { id: reminderListUser1.id };

                const result = await reminderList.updateOne({ input }, { req, res }, reminderList_updateOne);

                expect(result).not.toBeNull();
                expect(result.id).toEqual(reminderListUser1.id);
            });
        });

        describe("invalid", () => {
            it("cannot update another user's reminder list", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: ReminderListUpdateInput = { id: reminderListUser2.id };

                await expect(async () => {
                    await reminderList.updateOne({ input }, { req, res }, reminderList_updateOne);
                }).rejects.toThrow();
            });

            it("not logged in user cannot update reminder list", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: ReminderListUpdateInput = { id: reminderListUser1.id };

                await expect(async () => {
                    await reminderList.updateOne({ input }, { req, res }, reminderList_updateOne);
                }).rejects.toThrow();
            });

            it("cannot update non-existent reminder list", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });

                const input: ReminderListUpdateInput = { id: "non-existent-id" };

                await expect(async () => {
                    await reminderList.updateOne({ input }, { req, res }, reminderList_updateOne);
                }).rejects.toThrow();
            });
        });
    });
});
