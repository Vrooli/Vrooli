import { type ModelTestFixtures, TestDataFactory } from "../../../validation/models/__test/validationTestUtils.js";

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
export const scheduleRecurrenceFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
            recurrenceType: "Daily",
            interval: 1, // Every day
            scheduleConnect: validIds.id2,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            recurrenceType: "Weekly",
            interval: 2, // Every 2 weeks
            dayOfWeek: 3, // Wednesday (1-7, where 1 is Monday)
            dayOfMonth: null,
            duration: 60, // 60 minutes
            endDate: new Date("2025-12-31T23:59:59Z"),
            scheduleConnect: validIds.id2,
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 123,
            unknownField3: true,
        },
        update: {
            id: validIds.id1,
            recurrenceType: "Monthly",
            interval: 1, // Every month
            dayOfWeek: null,
            dayOfMonth: 15, // 15th of each month
            duration: 90, // 90 minutes
            endDate: new Date("2026-06-30T23:59:59Z"),
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 456,
        },
    },
    invalid: {
        missingRequired: {
            create: {
                id: validIds.id1,
                // Missing required recurrenceType, interval, and scheduleConnect
                dayOfWeek: 3,
                endDate: new Date("2025-12-31T23:59:59Z"),
            },
            update: {
                // Missing required id
                recurrenceType: "Daily",
                interval: 1,
            },
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
            },
            update: {
                id: validIds.id1,
                recurrenceType: 123, // Should be string enum
                interval: 0, // Should be >= 1
                dayOfWeek: 8, // Should be 1-7
                dayOfMonth: 32, // Should be 1-31
                duration: -1, // Should be positive
                endDate: "2025-12-31", // Should be Date object
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                recurrenceType: "Daily",
                interval: 1,
                scheduleConnect: validIds.id2,
            },
            update: {
                id: "invalid-id",
            },
        },
        invalidRecurrenceType: {
            create: {
                id: validIds.id1,
                recurrenceType: "Hourly", // Not a valid enum value
                interval: 1,
                scheduleConnect: validIds.id2,
            },
        },
        invalidInterval: {
            create: {
                id: validIds.id1,
                recurrenceType: "Daily",
                interval: 0, // Must be >= 1
                scheduleConnect: validIds.id2,
            },
            update: {
                id: validIds.id1,
                interval: -5, // Must be >= 1
            },
        },
        invalidDayOfWeek: {
            create: {
                id: validIds.id1,
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 0, // Must be 1-7
                scheduleConnect: validIds.id2,
            },
            update: {
                id: validIds.id1,
                dayOfWeek: 8, // Must be 1-7
            },
        },
        invalidDayOfMonth: {
            create: {
                id: validIds.id1,
                recurrenceType: "Monthly",
                interval: 1,
                dayOfMonth: 0, // Must be 1-31
                scheduleConnect: validIds.id2,
            },
            update: {
                id: validIds.id1,
                dayOfMonth: 32, // Must be 1-31
            },
        },
        invalidDuration: {
            create: {
                id: validIds.id1,
                recurrenceType: "Daily",
                interval: 1,
                duration: 0, // Must be >= 1
                scheduleConnect: validIds.id2,
            },
        },
        missingScheduleConnect: {
            create: {
                id: validIds.id1,
                recurrenceType: "Daily",
                interval: 1,
                // Missing required scheduleConnect
            },
        },
    },
    edgeCases: {
        dailyRecurrence: {
            create: {
                id: validIds.id1,
                recurrenceType: "Daily",
                interval: 1,
                scheduleConnect: validIds.id2,
            },
        },
        weeklyMonday: {
            create: {
                id: validIds.id1,
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 1, // Monday
                scheduleConnect: validIds.id2,
            },
        },
        weeklyFriday: {
            create: {
                id: validIds.id1,
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 5, // Friday
                scheduleConnect: validIds.id2,
            },
        },
        weeklySunday: {
            create: {
                id: validIds.id1,
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 7, // Sunday
                scheduleConnect: validIds.id2,
            },
        },
        biweekly: {
            create: {
                id: validIds.id1,
                recurrenceType: "Weekly",
                interval: 2,
                dayOfWeek: 3, // Wednesday
                scheduleConnect: validIds.id2,
            },
        },
        monthlyFirstDay: {
            create: {
                id: validIds.id1,
                recurrenceType: "Monthly",
                interval: 1,
                dayOfMonth: 1, // First day of month
                scheduleConnect: validIds.id2,
            },
        },
        monthlyMidMonth: {
            create: {
                id: validIds.id1,
                recurrenceType: "Monthly",
                interval: 1,
                dayOfMonth: 15, // Middle of month
                scheduleConnect: validIds.id2,
            },
        },
        monthlyLastDay: {
            create: {
                id: validIds.id1,
                recurrenceType: "Monthly",
                interval: 1,
                dayOfMonth: 31, // Last possible day
                scheduleConnect: validIds.id2,
            },
        },
        quarterlyRecurrence: {
            create: {
                id: validIds.id1,
                recurrenceType: "Monthly",
                interval: 3, // Every 3 months
                dayOfMonth: 1,
                scheduleConnect: validIds.id2,
            },
        },
        yearlyRecurrence: {
            create: {
                id: validIds.id1,
                recurrenceType: "Yearly",
                interval: 1,
                scheduleConnect: validIds.id2,
            },
        },
        biYearlyRecurrence: {
            create: {
                id: validIds.id1,
                recurrenceType: "Yearly",
                interval: 2, // Every 2 years
                scheduleConnect: validIds.id2,
            },
        },
        withEndDate: {
            create: {
                id: validIds.id1,
                recurrenceType: "Daily",
                interval: 1,
                endDate: new Date("2025-12-31T23:59:59Z"),
                scheduleConnect: validIds.id2,
            },
        },
        withoutEndDate: {
            create: {
                id: validIds.id1,
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 2, // Tuesday
                // No end date - infinite recurrence
                scheduleConnect: validIds.id2,
            },
        },
        shortDuration: {
            create: {
                id: validIds.id1,
                recurrenceType: "Daily",
                interval: 1,
                duration: 15, // 15 minutes
                scheduleConnect: validIds.id2,
            },
        },
        longDuration: {
            create: {
                id: validIds.id1,
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 4, // Thursday
                duration: 480, // 8 hours
                scheduleConnect: validIds.id2,
            },
        },
        differentScheduleId: {
            create: {
                id: validIds.id1,
                recurrenceType: "Daily",
                interval: 1,
                scheduleConnect: validIds.id6, // Different schedule
            },
        },
        maxInterval: {
            create: {
                id: validIds.id1,
                recurrenceType: "Daily",
                interval: 365, // Every 365 days
                scheduleConnect: validIds.id2,
            },
        },
        complexWeeklyPattern: {
            create: {
                id: validIds.id1,
                recurrenceType: "Weekly",
                interval: 3, // Every 3 weeks
                dayOfWeek: 6, // Saturday
                duration: 120, // 2 hours
                endDate: new Date("2026-12-31T23:59:59Z"),
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
                recurrenceType: "Monthly", // Changing from whatever it was
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
                endDate: new Date("2027-01-01T00:00:00Z"),
            },
        },
    },
};

// Custom factory that always generates valid IDs and required fields
const customizers = {
    create: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
        recurrenceType: base.recurrenceType || "Daily",
        interval: base.interval || 1,
        scheduleConnect: base.scheduleConnect || validIds.id2,
    }),
    update: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const scheduleRecurrenceTestDataFactory = new TestDataFactory(scheduleRecurrenceFixtures, customizers);
