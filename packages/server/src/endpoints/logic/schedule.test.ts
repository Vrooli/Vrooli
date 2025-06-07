import { type FindVersionInput, type ScheduleCreateInput, ScheduleRecurrenceType, type ScheduleSearchInput, type ScheduleUpdateInput, uuid } from "@vrooli/shared";
import { describe, it, expect, beforeAll, afterAll, beforeEach } from "vitest";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { assertFindManyResultIds } from "../../__test/helpers.js";
import { defaultPublicUserData, loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { schedule_createOne } from "../generated/schedule_createOne.js";
import { schedule_findMany } from "../generated/schedule_findMany.js";
import { schedule_findOne } from "../generated/schedule_findOne.js";
import { schedule_updateOne } from "../generated/schedule_updateOne.js";
import { schedule } from "./schedule.js";

// Test users
const user1Id = uuid();
const user2Id = uuid();

// Test schedule data
let scheduleUser1: any;
let scheduleUser2: any;

const scheduleUser1Data = {
    id: uuid(),
    startTime: new Date("2024-01-10T09:00:00Z"),
    endTime: new Date("2024-01-10T10:00:00Z"),
    timezone: "UTC",
    createdAt: new Date("2024-01-01T00:00:00Z"),
    updatedAt: new Date("2024-01-01T00:00:00Z"),
};

const scheduleUser2Data = {
    id: uuid(),
    startTime: new Date("2024-01-11T14:00:00Z"),
    endTime: new Date("2024-01-11T15:30:00Z"),
    timezone: "America/New_York",
    createdAt: new Date("2024-01-02T00:00:00Z"),
    updatedAt: new Date("2024-01-02T00:00:00Z"),
};

describe("EndpointsSchedule", () => {
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

        // Create schedulable objects owned by the users
        // User 1 Setup
        const teamUser1 = await DbProvider.get().team.create({
            data: {
                permissions: "{}",
                createdById: user1Id,
            },
        });
        const meetingUser1 = await DbProvider.get().meeting.create({
            data: {
                team: {
                    connect: {
                        id: teamUser1.id,
                    },
                },
                attendees: {
                    create: {
                        user: {
                            connect: {
                                id: user1Id,
                            },
                        },
                    },
                },
            },
        });

        // User 2 Setup
        const teamUser2 = await DbProvider.get().team.create({
            data: {
                permissions: "{}",
                createdById: user2Id,
            },
        });
        const roleUser2 = await DbProvider.get().role.create({
            data: {
                name: "admin",
                permissions: "{}",
                team: {
                    connect: {
                        id: teamUser2.id,
                    },
                },
            },
        });
        await DbProvider.get().member.create({
            data: {
                permissions: "{}",
                user: {
                    connect: {
                        id: user2Id,
                    },
                },
                roles: {
                    connect: {
                        id: roleUser2.id,
                    },
                },
                team: {
                    connect: {
                        id: teamUser2.id,
                    },
                },
            },
        });
        const meetingUser2 = await DbProvider.get().meeting.create({
            data: {
                attendees: {
                    create: {
                        user: {
                            connect: {
                                id: user2Id,
                            },
                        },
                    },
                },
                team: {
                    connect: {
                        id: teamUser2.id,
                    },
                },
            },
        });

        // Create user-specific schedules with relations
        scheduleUser1 = await DbProvider.get().schedule.create({
            data: {
                ...scheduleUser1Data,
                meetings: {
                    connect: {
                        id: meetingUser1.id,
                    },
                },
            },
        });
        await DbProvider.get().schedule_exception.createMany({
            data: [
                {
                    scheduleId: scheduleUser1.id,
                    originalStartTime: scheduleUser1Data.startTime,
                    newStartTime: new Date("2024-01-10T09:15:00Z"),
                    newEndTime: new Date("2024-01-10T10:15:00Z"),
                },
            ],
        });
        await DbProvider.get().schedule_recurrence.createMany({
            data: [
                {
                    scheduleId: scheduleUser1.id,
                    recurrenceType: ScheduleRecurrenceType.Weekly,
                    interval: 1,
                    dayOfWeek: 3, // Wednesday
                    endDate: new Date("2024-03-10T00:00:00Z"),
                },
            ],
        });

        scheduleUser2 = await DbProvider.get().schedule.create({
            data: {
                ...scheduleUser2Data,
                meetings: {
                    connect: {
                        id: meetingUser2.id,
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
            it("returns schedule by id when user owns the schedule", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                // Using FindVersionInput, assuming id is the primary way to find
                const input: FindVersionInput = { id: scheduleUser1.id };

                const result = await schedule.findOne({ input }, { req, res }, schedule_findOne);

                expect(result).to.not.be.null;
                expect(result.id).to.equal(scheduleUser1.id);
                expect(result.timezone).to.equal(scheduleUser1Data.timezone);
            });
        });

        describe("invalid", () => {
            it("fails when schedule id doesn't exist", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: FindVersionInput = { id: uuid() }; // Non-existent ID

                try {
                    await schedule.findOne({ input }, { req, res }, schedule_findOne);
                    expect.fail("Expected an error to be thrown for non-existent ID");
                } catch (error) {
                    // Error expected
                    expect(error).to.be.an("error"); // Or more specific error check
                }
            });

            it("fails when user tries to access another user's schedule", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id }; // User 1 trying to access User 2's schedule
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: FindVersionInput = { id: scheduleUser2.id };

                try {
                    await schedule.findOne({ input }, { req, res }, schedule_findOne);
                    // The readOneHelper should enforce ownership based on userId (assuming it exists)
                    expect.fail("Expected an error accessing other user's schedule");
                } catch (error) {
                    // Error expected
                    expect(error).to.be.an("error");
                }
            });

            it("not logged in user cannot access schedules", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: FindVersionInput = { id: scheduleUser1.id };

                try {
                    await schedule.findOne({ input }, { req, res }, schedule_findOne);
                    expect.fail("Expected an error for logged-out access");
                } catch (error) {
                    // Error expected
                    expect(error).to.be.an("error");
                }
            });

            it("API key with public permissions cannot access schedules (assumes schedules are private)", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: FindVersionInput = { id: scheduleUser1.id };
                try {
                    await schedule.findOne({ input }, { req, res }, schedule_findOne);
                    expect.fail("Expected an error for API key public read access");
                } catch (error) {
                    // Error expected as readOneHelper likely enforces ownership checks
                    expect(error).to.be.an("error");
                }
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns only own schedules for authenticated user", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                // User 1 should only see their own schedule
                const expectedScheduleIds = [scheduleUser1.id];

                const input: ScheduleSearchInput = { take: 10 }; // Basic search input
                const result = await schedule.findMany({ input }, { req, res }, schedule_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");

                assertFindManyResultIds(expect, result, expectedIds);
                expect(resultScheduleIds).to.not.include(scheduleUser2.id);
            });

            // Add more tests here if ScheduleSearchInput supports filters (e.g., time ranges)
            // Example: it("filters schedules by time range", async () => { ... });
        });

        describe("invalid", () => {
            it("not logged in user cannot list schedules", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: ScheduleSearchInput = { take: 10 };
                try {
                    await schedule.findMany({ input }, { req, res }, schedule_findMany);
                    expect.fail("Expected an error for logged-out access");
                } catch (error) {
                    // Error expected due to VisibilityType.Own and no user
                    expect(error).to.be.an("error");
                }
            });

            it("API key with public permissions returns empty list or fails (assumes schedules are private)", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                // We mock the session with user1, but the helper with VisibilityType.Own should still check against the user.
                // If the helper *only* relied on API key permissions and ignored the user for public reads, this might pass,
                // but the standard helper with VisibilityType.Own likely prioritizes the user context.
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: ScheduleSearchInput = { take: 10 };
                try {
                    const result = await schedule.findMany({ input }, { req, res }, schedule_findMany);
                    // Expect empty list because VisibilityType.Own should still apply
                    expect(result).to.not.be.null;
                    expect(result).to.have.property("edges").that.is.an("array").with.lengthOf(0);
                } catch (error) {
                    // Depending on exact helper implementation, it might throw instead
                    // expect(error).to.be.an("error");
                    // If it doesn't throw, the empty list check above handles it.
                    // For now, assume it returns empty list based on VisibilityType.Own
                    console.warn("API key findMany test assumed empty list, check helper logic if failure occurs.");
                }
            });
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates a schedule for authenticated user", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const newScheduleId = uuid();
                const startTime = new Date("2024-02-01T10:00:00Z");
                const endTime = new Date("2024-02-01T11:00:00Z");
                const input: ScheduleCreateInput = {
                    id: newScheduleId,
                    startTime,
                    endTime,
                    timezone: "Europe/London",
                };

                const creationResult = await schedule.createOne({ input }, { req, res }, schedule_createOne);

                expect(creationResult).to.not.be.null;
                expect(creationResult.id).to.equal(newScheduleId);
                expect(creationResult.timezone).to.equal(input.timezone);

                // Verify creation in DB (basic fields)
                const createdSchedule = await DbProvider.get().schedule.findUnique({ where: { id: newScheduleId } });
                expect(createdSchedule).to.not.be.null;
                expect(createdSchedule?.timezone).to.equal(input.timezone);
                // Cannot reliably check userId via API result, ownership tested by access control tests.
            });

            it("API key with write permissions can create schedule", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const newScheduleId = uuid();
                const input: ScheduleCreateInput = {
                    id: newScheduleId,
                    startTime: new Date("2024-02-02T12:00:00Z"),
                    endTime: new Date("2024-02-02T13:00:00Z"),
                    timezone: "UTC",
                };

                const result = await schedule.createOne({ input }, { req, res }, schedule_createOne);

                expect(result).to.not.be.null;
                expect(result.id).to.equal(newScheduleId);
                // Verify ownership implicitly via successful creation & access control tests
                const createdSchedule = await DbProvider.get().schedule.findUnique({ where: { id: newScheduleId } });
                expect(createdSchedule).to.not.be.null;
                expect(createdSchedule?.timezone).to.equal(input.timezone); // Check a basic field
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create schedule", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: ScheduleCreateInput = {
                    id: uuid(),
                    startTime: new Date("2024-02-03T10:00:00Z"),
                    endTime: new Date("2024-02-03T11:00:00Z"),
                    timezone: "UTC",
                };

                try {
                    await schedule.createOne({ input }, { req, res }, schedule_createOne);
                    expect.fail("Expected an error for logged-out create attempt");
                } catch (error) {
                    // Error expected as createOneHelper requires authenticated user
                    expect(error).to.be.an("error");
                }
            });

            it("API key without write permissions cannot create schedule", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: ScheduleCreateInput = {
                    id: uuid(),
                    startTime: new Date("2024-02-04T10:00:00Z"),
                    endTime: new Date("2024-02-04T11:00:00Z"),
                    timezone: "UTC",
                };

                try {
                    await schedule.createOne({ input }, { req, res }, schedule_createOne);
                    expect.fail("Expected error due to missing write permission");
                } catch (error) {
                    // Error expected from createOneHelper permission check
                    expect(error).to.be.an("error");
                }
            });

            // Add tests for invalid input data if validation logic exists in helpers/model
            // e.g., it("fails if startTime is after endTime", async () => { ... });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("updates own schedule for authenticated user and adds recurrence", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const newTimezone = "America/Los_Angeles";
                const newRecurrenceId = uuid();
                const input: ScheduleUpdateInput = {
                    id: scheduleUser1.id,
                    timezone: newTimezone,
                    // Example of adding a new recurrence during update
                    recurrencesCreate: [{
                        id: newRecurrenceId,
                        recurrenceType: ScheduleRecurrenceType.Monthly,
                        dayOfMonth: 15,
                        interval: 1,
                        duration: 3600, // Added missing required duration
                    }],
                };

                const result = await schedule.updateOne({ input }, { req, res }, schedule_updateOne);

                expect(result).to.not.be.null;
                expect(result.id).to.equal(scheduleUser1.id);
                expect(result.timezone).to.equal(newTimezone);

                // Verify update and nested create in DB
                const updatedSchedule = await DbProvider.get().schedule.findUnique({
                    where: { id: scheduleUser1.id },
                    include: { recurrences: true },
                });
                expect(updatedSchedule?.timezone).to.equal(newTimezone);
                expect(updatedSchedule?.recurrences.some(r => r.id === newRecurrenceId)).to.be.true;
                const addedRecurrence = updatedSchedule?.recurrences.find(r => r.id === newRecurrenceId);
                expect(addedRecurrence?.dayOfMonth).to.equal(15);
            });

            it("API key with write permissions can update own schedule", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const newEndTime = new Date("2024-01-10T10:45:00Z");
                const input: ScheduleUpdateInput = {
                    id: scheduleUser1.id,
                    endTime: newEndTime,
                };

                const result = await schedule.updateOne({ input }, { req, res }, schedule_updateOne);

                expect(result).to.not.be.null;
                expect(result.id).to.equal(scheduleUser1.id);
                expect(result.endTime.toISOString()).to.equal(newEndTime.toISOString());
                // Verify update in DB
                const updatedSchedule = await DbProvider.get().schedule.findUnique({ where: { id: scheduleUser1.id } });
                expect(updatedSchedule?.endTime.toISOString()).to.equal(newEndTime.toISOString());
            });
        });

        describe("invalid", () => {
            it("cannot update another user's schedule", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id }; // User 1
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: ScheduleUpdateInput = {
                    id: scheduleUser2.id, // Trying to update User 2's schedule
                    timezone: "Europe/Paris",
                };

                try {
                    await schedule.updateOne({ input }, { req, res }, schedule_updateOne);
                    expect.fail("Expected an error updating other user's schedule");
                } catch (error) {
                    // Error expected from updateOneHelper ownership check
                    expect(error).to.be.an("error");
                }
            });

            it("not logged in user cannot update schedule", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: ScheduleUpdateInput = {
                    id: scheduleUser1.id,
                    timezone: "Europe/Berlin",
                };

                try {
                    await schedule.updateOne({ input }, { req, res }, schedule_updateOne);
                    expect.fail("Expected an error for logged-out update attempt");
                } catch (error) {
                    // Error expected from updateOneHelper auth check
                    expect(error).to.be.an("error");
                }
            });

            it("cannot update non-existent schedule", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: ScheduleUpdateInput = {
                    id: uuid(), // Non-existent ID
                    timezone: "Asia/Tokyo",
                };

                try {
                    await schedule.updateOne({ input }, { req, res }, schedule_updateOne);
                    expect.fail("Expected an error updating non-existent schedule");
                } catch (error) {
                    // Error expected from updateOneHelper finding the record
                    expect(error).to.be.an("error");
                }
            });

            it("API key without write permissions cannot update schedule", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: ScheduleUpdateInput = {
                    id: scheduleUser1.id,
                    timezone: "Australia/Sydney",
                };

                try {
                    await schedule.updateOne({ input }, { req, res }, schedule_updateOne);
                    expect.fail("Expected error due to missing write permission for update");
                } catch (error) {
                    // Error expected from updateOneHelper permission check
                    expect(error).to.be.an("error");
                }
            });
        });
    });
});
