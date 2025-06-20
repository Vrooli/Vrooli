import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { 
    ScheduleDbFactory, 
    ScheduleExceptionDbFactory, 
    ScheduleRecurrenceDbFactory,
    RRuleHelpers
} from "./index.js";

describe("Schedule System Fixture Factories", () => {
    describe("ScheduleDbFactory", () => {
        it("should create minimal schedule", () => {
            const schedule = ScheduleDbFactory.createMinimal("meeting_123", "Meeting");
            
            expect(schedule).toBeDefined();
            expect(schedule.id).toBeDefined();
            expect(schedule.publicId).toBeDefined();
            expect(schedule.startTime).toBeInstanceOf(Date);
            expect(schedule.endTime).toBeInstanceOf(Date);
            expect(schedule.timezone).toBe("UTC");
            expect(schedule.meeting?.connect?.id).toBe("meeting_123");
        });

        it("should create schedule with RRULE", () => {
            const rrule = RRuleHelpers.weekly(["MO", "WE", "FR"], 1);
            const schedule = ScheduleDbFactory.createWithRRule(
                "project_456",
                "RunProject",
                rrule
            );
            
            expect(schedule.recurrences?.create).toBeDefined();
            const recurrence = schedule.recurrences.create as any;
            expect(recurrence.recurrenceType).toBe("Weekly");
            expect(recurrence.interval).toBe(1);
            expect(recurrence.dayOfWeek).toBe(1); // Monday
        });

        it("should parse complex RRULE patterns", () => {
            const testCases = [
                {
                    rrule: "FREQ=DAILY;INTERVAL=2",
                    expected: { recurrenceType: "Daily", interval: 2 }
                },
                {
                    rrule: "FREQ=WEEKLY;INTERVAL=1;BYDAY=TU,TH",
                    expected: { recurrenceType: "Weekly", interval: 1, dayOfWeek: 2 }
                },
                {
                    rrule: "FREQ=MONTHLY;INTERVAL=1;BYMONTHDAY=15",
                    expected: { recurrenceType: "Monthly", interval: 1, dayOfMonth: 15 }
                },
                {
                    rrule: "FREQ=YEARLY;INTERVAL=1;BYMONTH=6;BYMONTHDAY=15",
                    expected: { recurrenceType: "Yearly", interval: 1, month: 6, dayOfMonth: 15 }
                },
            ];

            testCases.forEach(({ rrule, expected }) => {
                const parsed = RRuleHelpers.parseToRecurrence(rrule);
                Object.entries(expected).forEach(([key, value]) => {
                    expect(parsed[key as keyof typeof parsed]).toBe(value);
                });
            });
        });

        it("should generate occurrences correctly", () => {
            const schedule = {
                startTime: new Date("2025-07-01T10:00:00Z"),
                endTime: new Date("2025-07-01T11:00:00Z"),
                timezone: "UTC",
                recurrences: [{
                    recurrenceType: "Weekly",
                    interval: 1,
                    dayOfWeek: 3, // Wednesday
                    dayOfMonth: null,
                    month: null,
                    endDate: null,
                }],
            };

            const occurrences = ScheduleDbFactory.generateOccurrences(
                schedule,
                {
                    start: new Date("2025-07-01T00:00:00Z"),
                    end: new Date("2025-07-31T23:59:59Z"),
                }
            );

            // July 2025 has 5 Wednesdays: 2, 9, 16, 23, 30
            expect(occurrences).toHaveLength(5);
            expect(occurrences[0].start.getDate()).toBe(2);
            expect(occurrences[4].start.getDate()).toBe(30);
        });

        it("should validate schedule data", () => {
            const factory = new ScheduleDbFactory();
            
            // Valid schedule
            const validSchedule = factory.createMinimal();
            const validResult = factory.validateFixture(validSchedule);
            expect(validResult.isValid).toBe(true);
            expect(validResult.errors).toHaveLength(0);

            // Invalid schedule - missing required fields
            const invalidSchedule = factory.createInvalid("missingRequired");
            const invalidResult = factory.validateFixture(invalidSchedule);
            expect(invalidResult.isValid).toBe(false);
            expect(invalidResult.errors.length).toBeGreaterThan(0);
        });

        it("should create schedule with multiple recurrences", () => {
            const schedule = ScheduleDbFactory.createWithRecurrences(
                "routine_789",
                "RunRoutine",
                [
                    { recurrenceType: "Daily", interval: 1 },
                    "FREQ=WEEKLY;BYDAY=MO,FR",
                    { recurrenceType: "Monthly", interval: 1, dayOfMonth: 1 },
                ]
            );

            expect(schedule.recurrences?.create).toHaveLength(3);
            const recurrences = schedule.recurrences.create as any[];
            expect(recurrences[0].recurrenceType).toBe("Daily");
            expect(recurrences[1].recurrenceType).toBe("Weekly");
            expect(recurrences[2].recurrenceType).toBe("Monthly");
        });
    });

    describe("ScheduleExceptionDbFactory", () => {
        it("should create minimal exception", () => {
            const exception = ScheduleExceptionDbFactory.createMinimal("schedule_123");
            
            expect(exception).toBeDefined();
            expect(exception.id).toBeDefined();
            expect(exception.originalStartTime).toBeInstanceOf(Date);
            expect(exception.schedule?.connect?.id).toBe("schedule_123");
        });

        it("should create cancellation exception", () => {
            const exception = ScheduleExceptionDbFactory.createCancellation(
                "schedule_456",
                new Date("2025-07-04T10:00:00Z")
            );
            
            expect(exception.newStartTime).toBeNull();
            expect(exception.newEndTime).toBeNull();
            expect(exception.originalStartTime).toEqual(new Date("2025-07-04T10:00:00Z"));
        });

        it("should create rescheduled exception", () => {
            const originalTime = new Date("2025-07-10T10:00:00Z");
            const newStartTime = new Date("2025-07-10T14:00:00Z");
            const newEndTime = new Date("2025-07-10T15:00:00Z");
            
            const exception = ScheduleExceptionDbFactory.createRescheduled(
                "schedule_789",
                originalTime,
                newStartTime,
                newEndTime
            );
            
            expect(exception.originalStartTime).toEqual(originalTime);
            expect(exception.newStartTime).toEqual(newStartTime);
            expect(exception.newEndTime).toEqual(newEndTime);
        });

        it("should create batch of exceptions", () => {
            const exceptions = ScheduleExceptionDbFactory.createBatch("schedule_101", [
                {
                    type: "cancel",
                    originalDate: new Date("2025-07-04T10:00:00Z"),
                },
                {
                    type: "reschedule",
                    originalDate: new Date("2025-07-10T10:00:00Z"),
                    newStartTime: new Date("2025-07-10T14:00:00Z"),
                    newEndTime: new Date("2025-07-10T15:00:00Z"),
                },
                {
                    type: "extend",
                    originalDate: new Date("2025-07-15T10:00:00Z"),
                    extensionHours: 2,
                },
            ]);

            expect(exceptions).toHaveLength(3);
            expect(exceptions[0].newStartTime).toBeNull(); // Cancelled
            expect(exceptions[1].newStartTime).toBeDefined(); // Rescheduled
            expect(exceptions[2].newEndTime?.getTime()).toBeGreaterThan(
                exceptions[2].originalStartTime.getTime()
            ); // Extended
        });

        it("should validate exception data", () => {
            const factory = new ScheduleExceptionDbFactory();
            
            // Valid exception
            const validException = factory.createMinimal();
            const validResult = factory.validateFixture(validException);
            expect(validResult.isValid).toBe(true);

            // Invalid exception - end before start
            const invalidException = factory.createInvalid("invalidTimeRange");
            const invalidResult = factory.validateFixture(invalidException);
            expect(invalidResult.isValid).toBe(false);
            expect(invalidResult.errors).toContain("New start time must be before new end time");
        });
    });

    describe("ScheduleRecurrenceDbFactory", () => {
        it("should create minimal recurrence", () => {
            const recurrence = ScheduleRecurrenceDbFactory.createMinimal("schedule_123");
            
            expect(recurrence).toBeDefined();
            expect(recurrence.id).toBeDefined();
            expect(recurrence.recurrenceType).toBe("Daily");
            expect(recurrence.interval).toBe(1);
            expect(recurrence.schedule?.connect?.id).toBe("schedule_123");
        });

        it("should create weekly recurrence", () => {
            const recurrence = ScheduleRecurrenceDbFactory.createWeekly(
                "schedule_456",
                3, // Wednesday
                1
            );
            
            expect(recurrence.recurrenceType).toBe("Weekly");
            expect(recurrence.dayOfWeek).toBe(3);
            expect(recurrence.interval).toBe(1);
        });

        it("should create monthly recurrence", () => {
            const recurrence = ScheduleRecurrenceDbFactory.createMonthly(
                "schedule_789",
                15, // 15th of month
                1
            );
            
            expect(recurrence.recurrenceType).toBe("Monthly");
            expect(recurrence.dayOfMonth).toBe(15);
            expect(recurrence.interval).toBe(1);
        });

        it("should create yearly recurrence", () => {
            const recurrence = ScheduleRecurrenceDbFactory.createYearly(
                "schedule_101",
                6, // June
                15, // 15th
                1
            );
            
            expect(recurrence.recurrenceType).toBe("Yearly");
            expect(recurrence.month).toBe(6);
            expect(recurrence.dayOfMonth).toBe(15);
            expect(recurrence.interval).toBe(1);
        });

        it("should create from RRULE", () => {
            const testCases = [
                {
                    rrule: "FREQ=DAILY;INTERVAL=2",
                    expected: { recurrenceType: "Daily", interval: 2 }
                },
                {
                    rrule: "FREQ=WEEKLY;INTERVAL=1;BYDAY=FR",
                    expected: { recurrenceType: "Weekly", interval: 1, dayOfWeek: 5 }
                },
                {
                    rrule: "FREQ=MONTHLY;INTERVAL=3;BYMONTHDAY=1",
                    expected: { recurrenceType: "Monthly", interval: 3, dayOfMonth: 1 }
                },
            ];

            testCases.forEach(({ rrule, expected }) => {
                const recurrence = ScheduleRecurrenceDbFactory.createFromRRule(
                    "schedule_test",
                    rrule
                );
                
                Object.entries(expected).forEach(([key, value]) => {
                    expect(recurrence[key as keyof typeof recurrence]).toBe(value);
                });
            });
        });

        it("should validate recurrence data", () => {
            const factory = new ScheduleRecurrenceDbFactory();
            
            // Valid recurrence
            const validRecurrence = factory.createMinimal();
            const validResult = factory.validateFixture(validRecurrence);
            expect(validResult.isValid).toBe(true);

            // Invalid - weekly without day
            const invalidRecurrence = factory.createMinimal({
                recurrenceType: "Weekly",
                dayOfWeek: null,
            });
            const invalidResult = factory.validateFixture(invalidRecurrence);
            expect(invalidResult.isValid).toBe(false);
            expect(invalidResult.errors).toContain("Weekly recurrence requires day of week");
        });

        it("should create multi-day weekly recurrence", () => {
            const recurrences = ScheduleRecurrenceDbFactory.createMultiDayWeekly(
                "schedule_202",
                [1, 3, 5], // Monday, Wednesday, Friday
                { duration: 60 }
            );

            expect(recurrences).toHaveLength(3);
            expect(recurrences[0].dayOfWeek).toBe(1);
            expect(recurrences[1].dayOfWeek).toBe(3);
            expect(recurrences[2].dayOfWeek).toBe(5);
            recurrences.forEach(r => {
                expect(r.duration).toBe(60);
            });
        });
    });

    describe("RRuleHelpers", () => {
        it("should create daily RRULE", () => {
            expect(RRuleHelpers.daily()).toBe("FREQ=DAILY;INTERVAL=1");
            expect(RRuleHelpers.daily(3)).toBe("FREQ=DAILY;INTERVAL=3");
        });

        it("should create weekly RRULE with days", () => {
            expect(RRuleHelpers.weekly(["MO", "WE", "FR"])).toBe("FREQ=WEEKLY;INTERVAL=1;BYDAY=MO,WE,FR");
            expect(RRuleHelpers.weekly(["TU", "TH"], 2)).toBe("FREQ=WEEKLY;INTERVAL=2;BYDAY=TU,TH");
        });

        it("should create monthly RRULE", () => {
            expect(RRuleHelpers.monthlyByDate(15)).toBe("FREQ=MONTHLY;INTERVAL=1;BYMONTHDAY=15");
            expect(RRuleHelpers.monthlyByPosition(2, "TU")).toBe("FREQ=MONTHLY;INTERVAL=1;BYDAY=2TU");
        });

        it("should create yearly RRULE", () => {
            expect(RRuleHelpers.yearly(6, 15)).toBe("FREQ=YEARLY;INTERVAL=1;BYMONTH=6;BYMONTHDAY=15");
        });

        it("should add count limit", () => {
            const base = "FREQ=DAILY;INTERVAL=1";
            expect(RRuleHelpers.withCount(base, 10)).toBe("FREQ=DAILY;INTERVAL=1;COUNT=10");
        });

        it("should add until date", () => {
            const base = "FREQ=WEEKLY;INTERVAL=1";
            const until = new Date("2025-12-31T23:59:59Z");
            const result = RRuleHelpers.withUntil(base, until);
            expect(result).toContain("UNTIL=20251231T235959Z");
        });
    });
});