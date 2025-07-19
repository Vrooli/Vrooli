import { ScheduleRecurrenceType, type ScheduleRecurrenceCreateInput, type ScheduleRecurrenceUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { scheduleRecurrenceValidation } from "../../../validation/models/scheduleRecurrence.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
    id6: "123456789012345683",
};

// Shared schedule recurrence test fixtures
export const scheduleRecurrenceFixtures: ModelTestFixtures<ScheduleRecurrenceCreateInput, ScheduleRecurrenceUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            recurrenceType: ScheduleRecurrenceType.Daily,
            interval: 1, // Every day
            duration: 60, // 60 minutes
            scheduleConnect: validIds.id2,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            recurrenceType: ScheduleRecurrenceType.Weekly,
            interval: 2, // Every 2 weeks
            dayOfWeek: 3, // Wednesday (1-7, where 1 is Monday)
            dayOfMonth: null,
            duration: 60, // 60 minutes
            endDate: new Date("2025-12-31T23:59:59Z").toISOString(),
            scheduleConnect: validIds.id2,
        } as ScheduleRecurrenceCreateInput,
        update: {
            id: validIds.id1,
            recurrenceType: ScheduleRecurrenceType.Monthly,
            interval: 1, // Every month
            dayOfWeek: null,
            dayOfMonth: 15, // 15th of each month
            duration: 90, // 90 minutes
            endDate: new Date("2026-06-30T23:59:59Z").toISOString(),
        } as ScheduleRecurrenceUpdateInput,
    },
    invalid: {
        missingRequired: {
            create: {
                id: validIds.id1,
                // Missing required recurrenceType, interval, duration, and scheduleConnect
                dayOfWeek: 3,
                endDate: new Date("2025-12-31T23:59:59Z").toISOString(),
            } as ScheduleRecurrenceCreateInput,
            update: {
                // Missing required id
                recurrenceType: ScheduleRecurrenceType.Daily,
                interval: 1,
            } as ScheduleRecurrenceUpdateInput,
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                recurrenceType: "InvalidType", // Invalid enum value
                interval: "one", // Should be number
                dayOfWeek: "Monday", // Should be number
                dayOfMonth: "first", // Should be number
                duration: -5, // Should be positive
                endDate: "not-a-date", // Should be Date
                scheduleConnect: validIds.id2,
            } as unknown as ScheduleRecurrenceCreateInput,
            update: {
                id: validIds.id1,
                recurrenceType: 123, // Should be string enum
                interval: 0, // Should be >= 1
                dayOfWeek: 8, // Should be 1-7
                dayOfMonth: 32, // Should be 1-31
                duration: -1, // Should be positive
                endDate: "2025-12-31", // Should be Date object
            } as unknown as ScheduleRecurrenceUpdateInput,
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                recurrenceType: ScheduleRecurrenceType.Daily,
                interval: 1,
                scheduleConnect: validIds.id2,
            } as ScheduleRecurrenceCreateInput,
            update: {
                id: "invalid-id",
            } as ScheduleRecurrenceUpdateInput,
        },
        invalidRecurrenceType: {
            create: {
                id: validIds.id1,
                recurrenceType: "Hourly", // Not a valid enum value
                interval: 1,
                scheduleConnect: validIds.id2,
            } as unknown as ScheduleRecurrenceCreateInput,
        },
        invalidInterval: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Daily,
                interval: 0, // Must be >= 1
                scheduleConnect: validIds.id2,
            } as ScheduleRecurrenceCreateInput,
            update: {
                id: validIds.id1,
                interval: -5, // Must be >= 1
            } as ScheduleRecurrenceUpdateInput,
        },
        invalidDayOfWeek: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Weekly,
                interval: 1,
                dayOfWeek: 0, // Must be 1-7
                scheduleConnect: validIds.id2,
            } as ScheduleRecurrenceCreateInput,
            update: {
                id: validIds.id1,
                dayOfWeek: 8, // Must be 1-7
            } as ScheduleRecurrenceUpdateInput,
        },
        invalidDayOfMonth: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Monthly,
                interval: 1,
                dayOfMonth: 0, // Must be 1-31
                scheduleConnect: validIds.id2,
            } as ScheduleRecurrenceCreateInput,
            update: {
                id: validIds.id1,
                dayOfMonth: 32, // Must be 1-31
            } as ScheduleRecurrenceUpdateInput,
        },
        invalidDuration: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Daily,
                interval: 1,
                duration: 0, // Must be >= 1
                scheduleConnect: validIds.id2,
            } as ScheduleRecurrenceCreateInput,
        },
        missingScheduleConnect: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Daily,
                interval: 1,
                duration: 60,
                // Missing required scheduleConnect
            } as ScheduleRecurrenceCreateInput,
        },
    },
    edgeCases: {
        dailyRecurrence: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Daily,
                interval: 1,
                duration: 60,
                scheduleConnect: validIds.id2,
            },
        },
        weeklyMonday: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Weekly,
                interval: 1,
                dayOfWeek: 1, // Monday
                duration: 60,
                scheduleConnect: validIds.id2,
            },
        },
        weeklyFriday: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Weekly,
                interval: 1,
                dayOfWeek: 5, // Friday
                duration: 60,
                scheduleConnect: validIds.id2,
            },
        },
        weeklySunday: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Weekly,
                interval: 1,
                dayOfWeek: 7, // Sunday
                duration: 60,
                scheduleConnect: validIds.id2,
            },
        },
        biweekly: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Weekly,
                interval: 2,
                dayOfWeek: 3, // Wednesday
                duration: 60,
                scheduleConnect: validIds.id2,
            },
        },
        monthlyFirstDay: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Monthly,
                interval: 1,
                dayOfMonth: 1, // First day of month
                duration: 60,
                scheduleConnect: validIds.id2,
            },
        },
        monthlyMidMonth: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Monthly,
                interval: 1,
                dayOfMonth: 15, // Middle of month
                duration: 60,
                scheduleConnect: validIds.id2,
            },
        },
        monthlyLastDay: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Monthly,
                interval: 1,
                dayOfMonth: 31, // Last possible day
                duration: 60,
                scheduleConnect: validIds.id2,
            },
        },
        quarterlyRecurrence: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Monthly,
                interval: 3, // Every 3 months
                dayOfMonth: 1,
                duration: 60,
                scheduleConnect: validIds.id2,
            },
        },
        yearlyRecurrence: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Yearly,
                interval: 1,
                duration: 60,
                scheduleConnect: validIds.id2,
            },
        },
        biYearlyRecurrence: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Yearly,
                interval: 2, // Every 2 years
                duration: 60,
                scheduleConnect: validIds.id2,
            },
        },
        withEndDate: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Daily,
                interval: 1,
                duration: 60,
                endDate: new Date("2025-12-31T23:59:59Z").toISOString(),
                scheduleConnect: validIds.id2,
            },
        },
        withoutEndDate: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Weekly,
                interval: 1,
                dayOfWeek: 2, // Tuesday
                duration: 60,
                // No end date - infinite recurrence
                scheduleConnect: validIds.id2,
            },
        },
        shortDuration: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Daily,
                interval: 1,
                duration: 15, // 15 minutes
                scheduleConnect: validIds.id2,
            },
        },
        longDuration: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Weekly,
                interval: 1,
                dayOfWeek: 4, // Thursday
                duration: 480, // 8 hours
                scheduleConnect: validIds.id2,
            },
        },
        differentScheduleId: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Daily,
                interval: 1,
                duration: 60,
                scheduleConnect: validIds.id6, // Different schedule
            },
        },
        maxInterval: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Daily,
                interval: 365, // Every 365 days
                duration: 60,
                scheduleConnect: validIds.id2,
            },
        },
        complexWeeklyPattern: {
            create: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Weekly,
                interval: 3, // Every 3 weeks
                dayOfWeek: 6, // Saturday
                duration: 120, // 2 hours
                endDate: new Date("2026-12-31T23:59:59Z").toISOString(),
                scheduleConnect: validIds.id2,
            },
        },
        onlyRequiredInUpdate: {
            update: {
                id: validIds.id1,
                // Only id is required in update
            },
        },
        updateRecurrenceType: {
            update: {
                id: validIds.id1,
                recurrenceType: ScheduleRecurrenceType.Monthly, // Changing from whatever it was
                dayOfMonth: 10,
            },
        },
        updateInterval: {
            update: {
                id: validIds.id1,
                interval: 7, // Changing interval
            },
        },
        updateEndDate: {
            update: {
                id: validIds.id1,
                endDate: new Date("2027-01-01T00:00:00Z").toISOString(),
            },
        },
    },
};

// Custom factory that always generates valid IDs and required fields
const customizers = {
    create: (base: Partial<ScheduleRecurrenceCreateInput>): ScheduleRecurrenceCreateInput => ({
        id: validIds.id1,
        recurrenceType: ScheduleRecurrenceType.Daily,
        interval: 1,
        duration: 60,
        scheduleConnect: validIds.id2,
        ...base,
    }),
    update: (base: Partial<ScheduleRecurrenceUpdateInput>): ScheduleRecurrenceUpdateInput => ({
        id: validIds.id1,
        ...base,
    }),
};

// Export a factory for creating test data programmatically
export const scheduleRecurrenceTestDataFactory = new TypedTestDataFactory(scheduleRecurrenceFixtures, scheduleRecurrenceValidation, customizers);
export const typedScheduleRecurrenceFixtures = createTypedFixtures(scheduleRecurrenceFixtures, scheduleRecurrenceValidation);
