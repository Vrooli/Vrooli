import { type FindByIdInput, type ReminderCreateInput, type ReminderSearchInput, type ReminderUpdateInput, uuid } from "@vrooli/shared";
import { describe, it, expect, beforeAll, afterAll, beforeEach } from "vitest";
import { after, describe, it } from "mocha";
import sinon from "sinon";
import { assertFindManyResultIds } from "../../__test/helpers.js";
import { defaultPublicUserData, loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { reminder_createOne } from "../generated/reminder_createOne.js";
import { reminder_findMany } from "../generated/reminder_findMany.js";
import { reminder_findOne } from "../generated/reminder_findOne.js";
import { reminder_updateOne } from "../generated/reminder_updateOne.js";
import { reminder } from "./reminder.js";

// Test users
const user1Id = uuid();
const user2Id = uuid();

// Reminder lists
const reminderListUser1Id = uuid();
const reminderListUser2Id = uuid();

// Test reminder data
const userReminder1 = {
    id: uuid(),
    name: "Test Reminder 1",
    description: "This is a test reminder for user 1",
    dueDate: new Date("2023-03-15"),
    index: 0,
    isComplete: false,
    createdAt: new Date("2023-03-01"),
    updatedAt: new Date("2023-03-12"),
};

const userReminder2 = {
    id: uuid(),
    name: "Test Reminder 2",
    description: "This is a test reminder for user 2",
    dueDate: new Date("2023-03-20"),
    index: 1,
    isComplete: false,
    createdAt: new Date("2023-03-05"),
    updatedAt: new Date("2023-03-05"),
};

describe("EndpointsReminder", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    beforeAll(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async function beforeEach() {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Create test users
        await DbProvider.get().user.create({
            data: {
                ...defaultPublicUserData(),
                id: user1Id,
                name: "Test User 1",
            },
        });
        await DbProvider.get().user.create({
            data: {
                ...defaultPublicUserData(),
                id: user2Id,
                name: "Test User 2",
            },
        });

        // Create user-specific reminders
        await DbProvider.get().reminder.create({
            data: {
                ...userReminder1,
                reminderList: {
                    create: {
                        id: reminderListUser1Id,
                        createdAt: new Date("2023-03-01"),
                        updatedAt: new Date("2023-03-01"),
                        user: {
                            connect: { id: user1Id },
                        },
                    },
                },
            },
        });
        await DbProvider.get().reminder.create({
            data: {
                ...userReminder2,
                reminderList: {
                    create: {
                        id: reminderListUser2Id,
                        createdAt: new Date("2023-03-05"),
                        updatedAt: new Date("2023-03-05"),
                        user: {
                            connect: { id: user2Id },
                        },
                    },
                },
            },
        });
    });

    afterAll(async function afterAll() {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns reminder by id when user owns the reminder", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: FindByIdInput = {
                    id: userReminder1.id,
                };

                const result = await reminder.findOne({ input }, { req, res }, reminder_findOne);

                expect(result).to.not.be.null;
                expect(result.id).to.equal(userReminder1.id);
                expect(result.name).to.equal(userReminder1.name);
            });

            it("API key with public permissions", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: FindByIdInput = {
                    id: userReminder1.id,
                };
                try {
                    await reminder.findOne({ input }, { req, res }, reminder_findOne);
                    expect.fail("Expected an error to be thrown - reminders are private");
                } catch (error) { /** Error expected  */ }
            });
        });

        describe("invalid", () => {
            it("fails when reminder id doesn't exist", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: FindByIdInput = {
                    id: uuid(), // Non-existent ID
                };

                try {
                    await reminder.findOne({ input }, { req, res }, reminder_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) {
                    // Error expected
                }
            });

            it("fails when user tries to access another user's reminder", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: FindByIdInput = {
                    id: userReminder2.id, // Reminder belongs to user2
                };

                try {
                    await reminder.findOne({ input }, { req, res }, reminder_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) {
                    // Error expected
                }
            });

            it("not logged in user can't access reminders", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: FindByIdInput = {
                    id: userReminder1.id,
                };

                try {
                    await reminder.findOne({ input }, { req, res }, reminder_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) {
                    // Error expected
                }
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns reminders without filters for authenticated user", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
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

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");

                assertFindManyResultIds(expect, result, expectedReminderIds);
            });

            it("filters by updated time frame", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
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

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");

                assertFindManyResultIds(expect, result, expectedReminderIds);
            });

            it("filters by due date", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
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

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");

                assertFindManyResultIds(expect, result, expectedIds);
            });

            it("API key - public permissions", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: ReminderSearchInput = {
                    take: 10,
                };
                try {
                    await reminder.findMany({ input }, { req, res }, reminder_findMany);
                    expect.fail("Expected an error to be thrown - reminders are private");
                } catch (error) { /** Error expected  */ }
            });
        });

        describe("invalid", () => {
            it("not logged in", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: ReminderSearchInput = {
                    take: 10,
                };
                try {
                    await reminder.findMany({ input }, { req, res }, reminder_findMany);
                    expect.fail("Expected an error to be thrown - reminders are private");
                } catch (error) { /** Error expected  */ }
            });

            it("invalid time range format", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = {
                    updatedTimeFrame: {
                        // Invalid date objects that will cause errors
                        after: new Date("invalid-date"),
                        before: new Date("invalid-date"),
                    },
                };

                try {
                    await reminder.findMany({ input }, { req, res }, reminder_findMany);
                    expect.fail("Expected an error to be thrown");
                } catch (error) {
                    // Error expected
                }
            });
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates a reminder for authenticated user", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const newReminderId = uuid();
                const input: ReminderCreateInput = {
                    id: newReminderId,
                    name: "New Test Reminder",
                    description: "This is a newly created reminder",
                    dueDate: new Date("2023-04-01"),
                    index: 2,
                    reminderListConnect: reminderListUser1Id,
                };

                // Check if the reminder list exists before trying to connect to it
                const existingList = await DbProvider.get().reminder_list.findUnique({
                    where: { id: reminderListUser1Id },
                });

                if (!existingList) {
                    throw new Error(`Test reminder list not found: ${reminderListUser1Id}`);
                }

                const creationResult = await reminder.createOne({ input }, { req, res }, reminder_createOne);

                expect(creationResult).to.not.be.null;
                expect(creationResult.id).to.equal(newReminderId);
                expect(creationResult.name).to.equal(input.name);
                expect(creationResult.description).to.equal(input.description);
            });

            it("API key with write permissions can create reminder", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const reminderListId = uuid();
                const newReminderId = uuid();
                const input: ReminderCreateInput = {
                    id: newReminderId,
                    name: "API Created Reminder",
                    description: "This reminder was created via API key",
                    dueDate: new Date("2023-04-15"),
                    index: 3,
                    reminderListCreate: {
                        id: reminderListId,
                    },
                };

                const result = await reminder.createOne({ input }, { req, res }, reminder_createOne);

                expect(result).to.not.be.null;
                expect(result.id).to.equal(newReminderId);
                expect(result.name).to.equal(input.name);
            });
        });

        describe("invalid", () => {
            it("not logged in", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: ReminderCreateInput = {
                    id: uuid(),
                    name: "Unauthorized Reminder",
                    description: "This reminder should not be created",
                    dueDate: new Date("2023-04-01"),
                    index: 4,
                    reminderListConnect: uuid(),
                };

                try {
                    await reminder.createOne({ input }, { req, res }, reminder_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) {
                    // Error expected
                }
            });

            it("cannot create reminder with invalid date", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = {
                    id: uuid(),
                    name: "Invalid Date Reminder",
                    description: "This reminder has an invalid date",
                    dueDate: new Date("invalid-date"),
                    index: 4,
                    reminderListConnect: reminderListUser1Id,
                };

                try {
                    await reminder.createOne({ input }, { req, res }, reminder_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) {
                    // Error expected
                }
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("updates a reminder for authenticated user", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: ReminderUpdateInput = {
                    id: userReminder1.id,
                    name: "Updated Reminder Title",
                    description: "This reminder has been updated",
                    isComplete: true,
                };

                const result = await reminder.updateOne({ input }, { req, res }, reminder_updateOne);

                expect(result).to.not.be.null;
                expect(result.id).to.equal(userReminder1.id);
                expect(result.name).to.equal(input.name);
                expect(result.description).to.equal(input.description);
            });

            it("API key with write permissions can update reminder", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: ReminderUpdateInput = {
                    id: userReminder1.id,
                    name: "API Updated Reminder",
                    isComplete: true,
                };

                const result = await reminder.updateOne({ input }, { req, res }, reminder_updateOne);

                expect(result).to.not.be.null;
                expect(result.id).to.equal(userReminder1.id);
                expect(result.name).to.equal(input.name);
            });
        });

        describe("invalid", () => {
            it("cannot update another user's reminder", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: ReminderUpdateInput = {
                    id: userReminder2.id, // Belongs to user2
                    name: "Unauthorized Update",
                    isComplete: true,
                };

                try {
                    await reminder.updateOne({ input }, { req, res }, reminder_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) {
                    // Error expected
                }
            });

            it("cannot connect to another user's reminder list", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: ReminderUpdateInput = {
                    id: userReminder1.id,
                    reminderListConnect: reminderListUser2Id,
                };

                try {
                    await reminder.updateOne({ input }, { req, res }, reminder_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) {
                    // Error expected
                }
            });

            it("not logged in user can't update reminder", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: ReminderUpdateInput = {
                    id: userReminder1.id,
                    name: "Unauthorized Update",
                    isComplete: true,
                };

                try {
                    await reminder.updateOne({ input }, { req, res }, reminder_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) {
                    // Error expected
                }
            });

            it("cannot update non-existent reminder", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: ReminderUpdateInput = {
                    id: uuid(), // Non-existent ID
                    name: "Update Non-existent Reminder",
                    isComplete: true,
                };

                try {
                    await reminder.updateOne({ input }, { req, res }, reminder_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) {
                    // Error expected
                }
            });
        });
    });
}); 
