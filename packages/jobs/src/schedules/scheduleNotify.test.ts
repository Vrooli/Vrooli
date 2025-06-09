import { generatePK, generatePublicId } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { scheduleNotify } from "./scheduleNotify.js";

// Direct import to avoid problematic services
const { DbProvider } = await import("../../../server/src/db/provider.ts");

// Mock services
vi.mock("@vrooli/server", async () => {
    const actual = await vi.importActual("@vrooli/server");
    return {
        ...actual,
        CacheService: {
            get: () => ({
                get: vi.fn().mockResolvedValue(null), // Not cached by default
                set: vi.fn().mockResolvedValue(undefined),
            }),
        },
        Notify: vi.fn(() => ({
            pushScheduleReminder: vi.fn().mockReturnValue({
                toUsers: vi.fn().mockResolvedValue(undefined),
            }),
        })),
        findFirstRel: vi.fn().mockImplementation((obj, fields) => {
            for (const field of fields) {
                if (obj[field] && obj[field].length > 0) {
                    return [field, obj[field]];
                }
            }
            return [null, null];
        }),
        parseJsonOrDefault: vi.fn().mockImplementation((json, defaultValue) => {
            if (!json) return defaultValue;
            try {
                return JSON.parse(json);
            } catch {
                return defaultValue;
            }
        }),
        scheduleExceptionsWhereInTimeframe: vi.fn().mockReturnValue({}),
        scheduleRecurrencesWhereInTimeframe: vi.fn().mockReturnValue({}),
        schedulesWhereInTimeframe: vi.fn().mockReturnValue({}),
        calculateOccurrences: vi.fn().mockResolvedValue([
            {
                start: new Date(Date.now() + 2 * 60 * 60 * 1000), // 2 hours from now
                end: new Date(Date.now() + 3 * 60 * 60 * 1000), // 3 hours from now
            },
        ]),
    };
});

describe("scheduleNotify integration tests", () => {
    // Store test entity IDs for cleanup
    const testUserIds: bigint[] = [];
    const testRunIds: bigint[] = [];
    const testRoutineIds: bigint[] = [];
    const testScheduleIds: bigint[] = [];
    const testNotificationSubscriptionIds: bigint[] = [];
    const testMeetingIds: bigint[] = [];

    beforeEach(async () => {
        // Clear test ID arrays
        testUserIds.length = 0;
        testRunIds.length = 0;
        testRoutineIds.length = 0;
        testScheduleIds.length = 0;
        testNotificationSubscriptionIds.length = 0;
        testMeetingIds.length = 0;

        // Reset mocks
        vi.clearAllMocks();
    });

    afterEach(async () => {
        // Clean up test data
        const db = DbProvider.get();
        
        // Clean up in reverse dependency order
        if (testNotificationSubscriptionIds.length > 0) {
            await db.notification_subscription.deleteMany({ where: { id: { in: testNotificationSubscriptionIds } } });
        }
        if (testMeetingIds.length > 0) {
            await db.meeting.deleteMany({ where: { id: { in: testMeetingIds } } });
        }
        if (testRunIds.length > 0) {
            await db.run.deleteMany({ where: { id: { in: testRunIds } } });
        }
        if (testScheduleIds.length > 0) {
            await db.schedule.deleteMany({ where: { id: { in: testScheduleIds } } });
        }
        if (testRoutineIds.length > 0) {
            await db.routine.deleteMany({ where: { id: { in: testRoutineIds } } });
        }
        if (testUserIds.length > 0) {
            await db.user.deleteMany({ where: { id: { in: testUserIds } } });
        }
    });

    it("should process scheduled runs with notification subscriptions", async () => {
        const now = new Date();
        
        // Create user
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Scheduled User",
                handle: "scheduleduser",
            },
        });
        testUserIds.push(user.id);

        // Create routine
        const routine = await DbProvider.get().routine.create({
            data: {
                id: generatePK(),
                createdById: user.id,
                ownedByUserId: user.id,
                isPrivate: false,
                isInternal: false,
            },
        });
        testRoutineIds.push(routine.id);

        // Create schedule
        const schedule = await DbProvider.get().schedule.create({
            data: {
                id: generatePK(),
                startTime: new Date(now.getTime() + 2 * 60 * 60 * 1000), // 2 hours from now
                endTime: new Date(now.getTime() + 3 * 60 * 60 * 1000), // 3 hours from now
                timezone: "UTC",
            },
        });
        testScheduleIds.push(schedule.id);

        // Create run with schedule
        const run = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                routineId: routine.id,
                scheduleId: schedule.id,
                isPrivate: false,
                status: "Scheduled",
                name: "Test Scheduled Run",
                createdById: user.id,
            },
        });
        testRunIds.push(run.id);

        // Create notification subscription
        const subscription = await DbProvider.get().notification_subscription.create({
            data: {
                id: generatePK(),
                scheduleId: schedule.id,
                subscriberId: user.id,
                silent: false,
                context: JSON.stringify({
                    reminders: [
                        { minutesBefore: 30 },
                        { minutesBefore: 5 },
                    ],
                }),
            },
        });
        testNotificationSubscriptionIds.push(subscription.id);

        await scheduleNotify();

        // Check that notifications were sent
        const { Notify } = await import("@vrooli/server");
        const mockNotify = Notify as any;
        expect(mockNotify).toHaveBeenCalledWith(["en"]);
        
        const mockPushScheduleReminder = mockNotify.mock.results[0].value.pushScheduleReminder;
        expect(mockPushScheduleReminder).toHaveBeenCalledWith(
            run.id.toString(),
            "Run",
            expect.any(Date)
        );

        const mockToUsers = mockPushScheduleReminder.mock.results[0].value.toUsers;
        expect(mockToUsers).toHaveBeenCalledWith([
            {
                userId: user.id.toString(),
                delays: expect.any(Array),
            },
        ]);

        // Check that cache was set
        const { CacheService } = await import("@vrooli/server");
        const mockSet = CacheService.get().set as any;
        expect(mockSet).toHaveBeenCalled();
    });

    it("should process scheduled meetings with notification subscriptions", async () => {
        const now = new Date();
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Meeting User",
                handle: "meetinguser",
            },
        });
        testUserIds.push(user.id);

        const schedule = await DbProvider.get().schedule.create({
            data: {
                id: generatePK(),
                startTime: new Date(now.getTime() + 1 * 60 * 60 * 1000), // 1 hour from now
                endTime: new Date(now.getTime() + 2 * 60 * 60 * 1000), // 2 hours from now
                timezone: "UTC",
            },
        });
        testScheduleIds.push(schedule.id);

        // Create meeting with schedule
        const meeting = await DbProvider.get().meeting.create({
            data: {
                id: generatePK(),
                scheduleId: schedule.id,
                createdById: user.id,
                openToAnyoneWithInvite: false,
                allowEntranceAfterStart: false,
                linkShareRole: "Spectator",
                showOnTeamProfile: false,
            },
        });
        testMeetingIds.push(meeting.id);

        const subscription = await DbProvider.get().notification_subscription.create({
            data: {
                id: generatePK(),
                scheduleId: schedule.id,
                subscriberId: user.id,
                silent: false,
                context: JSON.stringify({
                    reminders: [{ minutesBefore: 15 }],
                }),
            },
        });
        testNotificationSubscriptionIds.push(subscription.id);

        await scheduleNotify();

        // Check that notifications were sent for meeting
        const { Notify } = await import("@vrooli/server");
        const mockNotify = Notify as any;
        const mockPushScheduleReminder = mockNotify.mock.results[0].value.pushScheduleReminder;
        expect(mockPushScheduleReminder).toHaveBeenCalledWith(
            meeting.id.toString(),
            "Meeting",
            expect.any(Date)
        );
    });

    it("should handle subscriptions with silent notifications", async () => {
        const now = new Date();
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Silent User",
                handle: "silentuser",
            },
        });
        testUserIds.push(user.id);

        const routine = await DbProvider.get().routine.create({
            data: {
                id: generatePK(),
                createdById: user.id,
                ownedByUserId: user.id,
                isPrivate: false,
                isInternal: false,
            },
        });
        testRoutineIds.push(routine.id);

        const schedule = await DbProvider.get().schedule.create({
            data: {
                id: generatePK(),
                startTime: new Date(now.getTime() + 1 * 60 * 60 * 1000),
                endTime: new Date(now.getTime() + 2 * 60 * 60 * 1000),
                timezone: "UTC",
            },
        });
        testScheduleIds.push(schedule.id);

        const run = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                routineId: routine.id,
                scheduleId: schedule.id,
                isPrivate: false,
                status: "Scheduled",
                name: "Silent Run",
                createdById: user.id,
            },
        });
        testRunIds.push(run.id);

        // Create silent subscription
        const subscription = await DbProvider.get().notification_subscription.create({
            data: {
                id: generatePK(),
                scheduleId: schedule.id,
                subscriberId: user.id,
                silent: true, // Silent notification
                context: JSON.stringify({
                    reminders: [{ minutesBefore: 10 }],
                }),
            },
        });
        testNotificationSubscriptionIds.push(subscription.id);

        await scheduleNotify();

        // Should still process but with silent flag
        const { Notify } = await import("@vrooli/server");
        expect(Notify).toHaveBeenCalled();
    });

    it("should handle subscriptions with invalid context gracefully", async () => {
        const now = new Date();
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Invalid Context User",
                handle: "invalidcontextuser",
            },
        });
        testUserIds.push(user.id);

        const routine = await DbProvider.get().routine.create({
            data: {
                id: generatePK(),
                createdById: user.id,
                ownedByUserId: user.id,
                isPrivate: false,
                isInternal: false,
            },
        });
        testRoutineIds.push(routine.id);

        const schedule = await DbProvider.get().schedule.create({
            data: {
                id: generatePK(),
                startTime: new Date(now.getTime() + 1 * 60 * 60 * 1000),
                endTime: new Date(now.getTime() + 2 * 60 * 60 * 1000),
                timezone: "UTC",
            },
        });
        testScheduleIds.push(schedule.id);

        const run = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                routineId: routine.id,
                scheduleId: schedule.id,
                isPrivate: false,
                status: "Scheduled",
                name: "Invalid Context Run",
                createdById: user.id,
            },
        });
        testRunIds.push(run.id);

        // Create subscription with invalid JSON context
        const subscription = await DbProvider.get().notification_subscription.create({
            data: {
                id: generatePK(),
                scheduleId: schedule.id,
                subscriberId: user.id,
                silent: false,
                context: "invalid json{", // Invalid JSON
            },
        });
        testNotificationSubscriptionIds.push(subscription.id);

        // Should not throw
        await expect(scheduleNotify()).resolves.not.toThrow();

        // Should use default context due to invalid JSON
        const { parseJsonOrDefault } = await import("@vrooli/server");
        expect(parseJsonOrDefault).toHaveBeenCalledWith("invalid json{", { reminders: [] });
    });

    it("should skip processing when no runs or meetings are associated with schedule", async () => {
        const now = new Date();
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Empty Schedule User",
                handle: "emptyscheduleuser",
            },
        });
        testUserIds.push(user.id);

        // Create schedule without any runs or meetings
        const schedule = await DbProvider.get().schedule.create({
            data: {
                id: generatePK(),
                startTime: new Date(now.getTime() + 1 * 60 * 60 * 1000),
                endTime: new Date(now.getTime() + 2 * 60 * 60 * 1000),
                timezone: "UTC",
            },
        });
        testScheduleIds.push(schedule.id);

        const subscription = await DbProvider.get().notification_subscription.create({
            data: {
                id: generatePK(),
                scheduleId: schedule.id,
                subscriberId: user.id,
                silent: false,
                context: JSON.stringify({
                    reminders: [{ minutesBefore: 10 }],
                }),
            },
        });
        testNotificationSubscriptionIds.push(subscription.id);

        await scheduleNotify();

        // Should not send notifications because no runs/meetings
        const { Notify } = await import("@vrooli/server");
        expect(Notify).not.toHaveBeenCalled();
    });

    it("should handle multiple subscribers with different reminder preferences", async () => {
        const now = new Date();
        
        // Create multiple users
        const users = await Promise.all([
            DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "User 1",
                    handle: "user1",
                },
            }),
            DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "User 2",
                    handle: "user2",
                },
            }),
        ]);
        users.forEach(u => testUserIds.push(u.id));

        const routine = await DbProvider.get().routine.create({
            data: {
                id: generatePK(),
                createdById: users[0].id,
                ownedByUserId: users[0].id,
                isPrivate: false,
                isInternal: false,
            },
        });
        testRoutineIds.push(routine.id);

        const schedule = await DbProvider.get().schedule.create({
            data: {
                id: generatePK(),
                startTime: new Date(now.getTime() + 1 * 60 * 60 * 1000),
                endTime: new Date(now.getTime() + 2 * 60 * 60 * 1000),
                timezone: "UTC",
            },
        });
        testScheduleIds.push(schedule.id);

        const run = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                routineId: routine.id,
                scheduleId: schedule.id,
                isPrivate: false,
                status: "Scheduled",
                name: "Multi User Run",
                createdById: users[0].id,
            },
        });
        testRunIds.push(run.id);

        // Create subscriptions with different reminder preferences
        const subscription1 = await DbProvider.get().notification_subscription.create({
            data: {
                id: generatePK(),
                scheduleId: schedule.id,
                subscriberId: users[0].id,
                silent: false,
                context: JSON.stringify({
                    reminders: [
                        { minutesBefore: 30 },
                        { minutesBefore: 5 },
                    ],
                }),
            },
        });
        testNotificationSubscriptionIds.push(subscription1.id);

        const subscription2 = await DbProvider.get().notification_subscription.create({
            data: {
                id: generatePK(),
                scheduleId: schedule.id,
                subscriberId: users[1].id,
                silent: false,
                context: JSON.stringify({
                    reminders: [{ minutesBefore: 15 }],
                }),
            },
        });
        testNotificationSubscriptionIds.push(subscription2.id);

        await scheduleNotify();

        // Check that notifications were sent to both users
        const { Notify } = await import("@vrooli/server");
        const mockNotify = Notify as any;
        const mockToUsers = mockNotify.mock.results[0].value.pushScheduleReminder.mock.results[0].value.toUsers;
        
        expect(mockToUsers).toHaveBeenCalledWith(
            expect.arrayContaining([
                expect.objectContaining({
                    userId: users[0].id.toString(),
                    delays: expect.any(Array),
                }),
                expect.objectContaining({
                    userId: users[1].id.toString(),
                    delays: expect.any(Array),
                }),
            ])
        );
    });

    it("should check cache to avoid duplicate notifications", async () => {
        const now = new Date();
        
        // Mock cache to return existing entry (already notified)
        const { CacheService } = await import("@vrooli/server");
        const mockGet = CacheService.get().get as any;
        mockGet.mockResolvedValueOnce("true"); // Already cached
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Cached User",
                handle: "cacheduser",
            },
        });
        testUserIds.push(user.id);

        const routine = await DbProvider.get().routine.create({
            data: {
                id: generatePK(),
                createdById: user.id,
                ownedByUserId: user.id,
                isPrivate: false,
                isInternal: false,
            },
        });
        testRoutineIds.push(routine.id);

        const schedule = await DbProvider.get().schedule.create({
            data: {
                id: generatePK(),
                startTime: new Date(now.getTime() + 1 * 60 * 60 * 1000),
                endTime: new Date(now.getTime() + 2 * 60 * 60 * 1000),
                timezone: "UTC",
            },
        });
        testScheduleIds.push(schedule.id);

        const run = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                routineId: routine.id,
                scheduleId: schedule.id,
                isPrivate: false,
                status: "Scheduled",
                name: "Cached Run",
                createdById: user.id,
            },
        });
        testRunIds.push(run.id);

        const subscription = await DbProvider.get().notification_subscription.create({
            data: {
                id: generatePK(),
                scheduleId: schedule.id,
                subscriberId: user.id,
                silent: false,
                context: JSON.stringify({
                    reminders: [{ minutesBefore: 10 }],
                }),
            },
        });
        testNotificationSubscriptionIds.push(subscription.id);

        await scheduleNotify();

        // Should not send notification because user was already notified (cached)
        const { Notify } = await import("@vrooli/server");
        const mockNotify = Notify as any;
        if (mockNotify.mock.calls.length > 0) {
            const mockToUsers = mockNotify.mock.results[0].value.pushScheduleReminder.mock.results[0].value.toUsers;
            expect(mockToUsers).toHaveBeenCalledWith([]); // Empty array due to cache filter
        }
    });

    it("should handle schedules without notification subscriptions", async () => {
        const now = new Date();
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "No Subscription User",
                handle: "nosubscriptionuser",
            },
        });
        testUserIds.push(user.id);

        const routine = await DbProvider.get().routine.create({
            data: {
                id: generatePK(),
                createdById: user.id,
                ownedByUserId: user.id,
                isPrivate: false,
                isInternal: false,
            },
        });
        testRoutineIds.push(routine.id);

        const schedule = await DbProvider.get().schedule.create({
            data: {
                id: generatePK(),
                startTime: new Date(now.getTime() + 1 * 60 * 60 * 1000),
                endTime: new Date(now.getTime() + 2 * 60 * 60 * 1000),
                timezone: "UTC",
            },
        });
        testScheduleIds.push(schedule.id);

        const run = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                routineId: routine.id,
                scheduleId: schedule.id,
                isPrivate: false,
                status: "Scheduled",
                name: "No Subscription Run",
                createdById: user.id,
            },
        });
        testRunIds.push(run.id);

        // No notification subscriptions created

        // Should not throw
        await expect(scheduleNotify()).resolves.not.toThrow();

        // Should not send any notifications
        const { Notify } = await import("@vrooli/server");
        expect(Notify).not.toHaveBeenCalled();
    });

    it("should handle reminders with invalid minutesBefore values", async () => {
        const now = new Date();
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Invalid Reminder User",
                handle: "invalidreminderuser",
            },
        });
        testUserIds.push(user.id);

        const routine = await DbProvider.get().routine.create({
            data: {
                id: generatePK(),
                createdById: user.id,
                ownedByUserId: user.id,
                isPrivate: false,
                isInternal: false,
            },
        });
        testRoutineIds.push(routine.id);

        const schedule = await DbProvider.get().schedule.create({
            data: {
                id: generatePK(),
                startTime: new Date(now.getTime() + 1 * 60 * 60 * 1000),
                endTime: new Date(now.getTime() + 2 * 60 * 60 * 1000),
                timezone: "UTC",
            },
        });
        testScheduleIds.push(schedule.id);

        const run = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                routineId: routine.id,
                scheduleId: schedule.id,
                isPrivate: false,
                status: "Scheduled",
                name: "Invalid Reminder Run",
                createdById: user.id,
            },
        });
        testRunIds.push(run.id);

        // Create subscription with invalid reminder values
        const subscription = await DbProvider.get().notification_subscription.create({
            data: {
                id: generatePK(),
                scheduleId: schedule.id,
                subscriberId: user.id,
                silent: false,
                context: JSON.stringify({
                    reminders: [
                        { minutesBefore: "not_a_number" }, // Invalid
                        { minutesBefore: 10 }, // Valid
                        { minutesBefore: null }, // Invalid
                    ],
                }),
            },
        });
        testNotificationSubscriptionIds.push(subscription.id);

        // Should not throw and should process only valid reminders
        await expect(scheduleNotify()).resolves.not.toThrow();
    });
});