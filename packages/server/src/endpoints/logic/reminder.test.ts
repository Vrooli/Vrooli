import { type FindByIdInput, type ReminderCreateInput, type ReminderSearchInput, type ReminderUpdateInput, generatePK } from "@vrooli/shared";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { assertFindManyResultIds } from "../../__test/helpers.js";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { reminder_createOne } from "../generated/reminder_createOne.js";
import { reminder_findMany } from "../generated/reminder_findMany.js";
import { reminder_findOne } from "../generated/reminder_findOne.js";
import { reminder_updateOne } from "../generated/reminder_updateOne.js";
import { reminder } from "./reminder.js";
// Import database fixtures for seeding
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
// Import validation fixtures for API input testing
import { reminderTestDataFactory } from "@vrooli/shared";
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";
import { CacheService } from "../../redisConn.js";

describe("EndpointsReminder", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    afterEach(async () => {
        // Validate cleanup to detect any missed records
        const orphans = await validateCleanup(DbProvider.get(), {
            tables: ["user", "user_auth", "email", "session"],
            logOrphans: true,
        });
        if (orphans.length > 0) {
            console.warn("Test cleanup incomplete:", orphans);
        }
    });

    beforeEach(async () => {
        // Clean up using dependency-ordered cleanup helpers
        await cleanupGroups.minimal(DbProvider.get());
        // Clear Redis cache to reset rate limiting
        await CacheService.get().flushAll();
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    // Helper function to create test data
    async function createTestData() {
        // Seed test users using database fixtures
        const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

        // Create reminder lists
        const reminderListUser1 = await DbProvider.get().reminder_list.create({
            data: {
                id: generatePK(),
                userId: testUsers[0].id,
            },
        });
        const reminderListUser2 = await DbProvider.get().reminder_list.create({
            data: {
                id: generatePK(),
                userId: testUsers[1].id,
            },
        });

        // Create reminders
        const userReminder1 = await DbProvider.get().reminder.create({
            data: {
                id: generatePK(),
                reminderListId: reminderListUser1.id,
                name: "Test Reminder 1",
                description: "This is a test reminder for user 1",
                dueDate: new Date("2023-03-15"),
                index: 0,
                createdAt: new Date("2023-03-01"),
                updatedAt: new Date("2023-03-12"),
            },
        });

        const userReminder2 = await DbProvider.get().reminder.create({
            data: {
                id: generatePK(),
                reminderListId: reminderListUser2.id,
                name: "Test Reminder 2",
                description: "This is a test reminder for user 2",
                dueDate: new Date("2023-03-20"),
                index: 0,
                createdAt: new Date("2023-03-05"),
                updatedAt: new Date("2023-03-05"),
            },
        });


        return {
            testUsers,
            reminderListUser1,
            reminderListUser2,
            userReminder1,
            userReminder2,
        };
    }

    describe("findOne", () => {
        describe("valid", () => {
            it("returns reminder by id when user owns the reminder", async () => {
                const { testUsers, userReminder1 } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input: FindByIdInput = {
                    id: userReminder1.id,
                };

                const result = await reminder.findOne({ input }, { req, res }, reminder_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toEqual(userReminder1.id);
                expect(result.name).toBe("Test Reminder 1");
            });

            it("API key with public permissions", async () => {
                const { testUsers, userReminder1 } = await createTestData();
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input: FindByIdInput = {
                    id: userReminder1.id,
                };

                await expect(async () => {
                    await reminder.findOne({ input }, { req, res }, reminder_findOne);
                }).rejects.toThrow(); // Reminders are private
            });
        });

        describe("invalid", () => {
            it("fails when reminder id doesn't exist", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input: FindByIdInput = {
                    id: "non-existent-id",
                };

                await expect(async () => {
                    await reminder.findOne({ input }, { req, res }, reminder_findOne);
                }).rejects.toThrow();
            });

            it("fails when user tries to access another user's reminder", async () => {
                const { testUsers, userReminder2 } = await createTestData();
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: FindByIdInput = {
                    id: userReminder2.id, // Reminder belongs to user2
                };

                await expect(async () => {
                    await reminder.findOne({ input }, { req, res }, reminder_findOne);
                }).rejects.toThrow();
            });

            it("not logged in user can't access reminders", async () => {
                const { userReminder1 } = await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: FindByIdInput = {
                    id: userReminder1.id,
                };

                await expect(async () => {
                    await reminder.findOne({ input }, { req, res }, reminder_findOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns reminders without filters for authenticated user", async () => {
                const { testUsers, userReminder1 } = await createTestData();
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                // When logged in as user1, should only see user1's reminders
                const expectedReminderIds = [
                    userReminder1.id,   // User1's reminder
                    // userReminder2.id, // User2's reminder (should not be included)
                ];

                const input: ReminderSearchInput = {
                    take: 10,
                };
                const result = await reminder.findMany({ input }, { req, res }, reminder_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);

                assertFindManyResultIds(expect, result, expectedReminderIds);
            });

            it("filters by updated time frame", async () => {
                const { testUsers, userReminder1 } = await createTestData();
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                // For the given time range, should only see reminders updated in Feb-Mar that user1 has access to
                const expectedReminderIds = [
                    userReminder1.id,   // Updated in March and belongs to user1
                    // userReminder2.id,   // Belongs to user2 (no access)
                ];

                const input: ReminderSearchInput = {
                    updatedTimeFrame: {
                        after: new Date("2023-02-01"),
                        before: new Date("2023-04-01"),
                    },
                };
                const result = await reminder.findMany({ input }, { req, res }, reminder_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);

                assertFindManyResultIds(expect, result, expectedReminderIds);
            });

            it("filters by due date", async () => {
                const { testUsers, userReminder1 } = await createTestData();
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: ReminderSearchInput = {
                    updatedTimeFrame: {
                        after: new Date("2023-03-10"),
                        before: new Date("2023-03-20"),
                    },
                };
                const expectedIds = [
                    userReminder1.id,
                ];
                const result = await reminder.findMany({ input }, { req, res }, reminder_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);

                assertFindManyResultIds(expect, result, expectedIds);
            });

            it("API key - public permissions", async () => {
                const { testUsers } = await createTestData();
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: ReminderSearchInput = {
                    take: 10,
                };
                await expect(async () => {
                    await reminder.findMany({ input }, { req, res }, reminder_findMany);
                }).rejects.toThrow(); // reminders are private
            });
        });

        describe("invalid", () => {
            it("not logged in", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: ReminderSearchInput = {
                    take: 10,
                };
                await expect(async () => {
                    await reminder.findMany({ input }, { req, res }, reminder_findMany);
                }).rejects.toThrow(); // reminders are private
            });

            it("invalid time range format", async () => {
                const { testUsers } = await createTestData();
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = {
                    updatedTimeFrame: {
                        // Invalid date objects that will cause errors
                        after: new Date("invalid-date"),
                        before: new Date("invalid-date"),
                    },
                };

                await expect(async () => {
                    await reminder.findMany({ input }, { req, res }, reminder_findMany);
                }).rejects.toThrow();
            });
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates a reminder for authenticated user", async () => {
                const { testUsers, reminderListUser1 } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                // Use validation fixtures for API input
                const input: ReminderCreateInput = reminderTestDataFactory.createMinimal({
                    name: "New Test Reminder",
                    description: "This is a newly created reminder",
                    dueDate: new Date("2023-04-01"),
                    reminderListConnect: reminderListUser1.id,
                });

                const creationResult = await reminder.createOne({ input }, { req, res }, reminder_createOne);

                expect(creationResult).not.toBeNull();
                expect(creationResult.id).toBeDefined();
                expect(creationResult.name).toBe("New Test Reminder");
                expect(creationResult.description).toBe("This is a newly created reminder");
            });

            it("API key with write permissions can create reminder", async () => {
                const { testUsers } = await createTestData();
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                // Use validation fixtures for API input
                const input: ReminderCreateInput = reminderTestDataFactory.createComplete({
                    name: "API Created Reminder",
                    description: "This reminder was created via API key",
                    dueDate: new Date("2023-04-15"),
                    reminderListCreate: {},
                });

                const result = await reminder.createOne({ input }, { req, res }, reminder_createOne);

                expect(result).not.toBeNull();
                expect(result.id).toBeDefined();
                expect(result.name).toBe("API Created Reminder");
            });
        });

        describe("invalid", () => {
            it("not logged in", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                // Use validation fixtures for API input
                const input: ReminderCreateInput = reminderTestDataFactory.createMinimal({
                    name: "Unauthorized Reminder",
                    description: "This reminder should not be created",
                    dueDate: new Date("2023-04-01"),
                    reminderListConnect: "non-existent-list-id",
                });

                await expect(async () => {
                    await reminder.createOne({ input }, { req, res }, reminder_createOne);
                }).rejects.toThrow();
            });

            it("cannot create reminder with invalid date", async () => {
                const { testUsers, reminderListUser1 } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                // Use validation fixtures for API input
                const input: ReminderCreateInput = reminderTestDataFactory.createMinimal({
                    name: "Invalid Date Reminder",
                    description: "This reminder has an invalid date",
                    dueDate: new Date("invalid-date"),
                    reminderListConnect: reminderListUser1.id,
                });

                await expect(async () => {
                    await reminder.createOne({ input }, { req, res }, reminder_createOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("updates a reminder for authenticated user", async () => {
                const { testUsers, userReminder1 } = await createTestData();
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: ReminderUpdateInput = {
                    id: userReminder1.id,
                    name: "Updated Reminder Title",
                    description: "This reminder has been updated",
                    isComplete: true,
                };

                const result = await reminder.updateOne({ input }, { req, res }, reminder_updateOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(userReminder1.id);
                expect(result.name).toBe(input.name);
                expect(result.description).toBe(input.description);
            });

            it("API key with write permissions can update reminder", async () => {
                const { testUsers, userReminder1 } = await createTestData();
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: ReminderUpdateInput = {
                    id: userReminder1.id,
                    name: "API Updated Reminder",
                    isComplete: true,
                };

                const result = await reminder.updateOne({ input }, { req, res }, reminder_updateOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(userReminder1.id);
                expect(result.name).toBe(input.name);
            });
        });

        describe("invalid", () => {
            it("cannot update another user's reminder", async () => {
                const { testUsers, userReminder2 } = await createTestData();
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: ReminderUpdateInput = {
                    id: userReminder2.id, // Belongs to user2
                    name: "Unauthorized Update",
                    isComplete: true,
                };

                await expect(async () => {
                    await reminder.updateOne({ input }, { req, res }, reminder_updateOne);
                }).rejects.toThrow();
            });

            it("cannot connect to another user's reminder list", async () => {
                const { testUsers, userReminder1, reminderListUser2 } = await createTestData();
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: ReminderUpdateInput = {
                    id: userReminder1.id,
                    reminderListConnect: reminderListUser2.id,
                };

                await expect(async () => {
                    await reminder.updateOne({ input }, { req, res }, reminder_updateOne);
                }).rejects.toThrow();
            });

            it("not logged in user can't update reminder", async () => {
                const { userReminder1 } = await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: ReminderUpdateInput = {
                    id: userReminder1.id,
                    name: "Unauthorized Update",
                    isComplete: true,
                };

                await expect(async () => {
                    await reminder.updateOne({ input }, { req, res }, reminder_updateOne);
                }).rejects.toThrow();
            });

            it("cannot update non-existent reminder", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input: ReminderUpdateInput = {
                    id: "non-existent-id",
                    name: "Update Non-existent Reminder",
                    isComplete: true,
                };

                await expect(async () => {
                    await reminder.updateOne({ input }, { req, res }, reminder_updateOne);
                }).rejects.toThrow();
            });
        });
    });
}); 
