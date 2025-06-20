import type { ScheduleCreateInput, ScheduleUpdateInput } from "../../../api/types.js";
import { ScheduleRecurrenceType } from "../../../api/types.js";
import { type ModelTestFixtures, TypedTestDataFactory, createTypedFixtures, testValues } from "../../../validation/models/__test/validationTestUtils.js";
import { scheduleValidation } from "../../../validation/models/schedule.js";

// Magic number constants for testing
const TIMEZONE_TOO_LONG_LENGTH = 65;
const TIMEZONE_MAX_LENGTH = 64;

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
    id6: "123456789012345683",
    id7: "123456789012345684",
    id8: "123456789012345685",
    id9: "123456789012345686",
    id10: "123456789012345687",
};

// Shared schedule test fixtures
export const scheduleFixtures: ModelTestFixtures<ScheduleCreateInput, ScheduleUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            timezone: "America/New_York",
            startTime: new Date("2025-01-01T00:00:00Z"), // Adding startTime to avoid endTime validation issues
        },
        update: {
            id: validIds.id1,
            startTime: new Date("2025-01-01T00:00:00Z"), // Adding startTime to avoid endTime validation issues
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            startTime: new Date("2025-01-01T09:00:00Z"),
            endTime: new Date("2025-12-31T17:00:00Z"),
            timezone: "America/New_York",
            meetingConnect: validIds.id2,
            exceptionsCreate: [
                {
                    id: validIds.id3,
                    originalStartTime: new Date("2025-07-04T09:00:00Z"),
                    newStartTime: new Date("2025-07-04T09:00:00Z"), // Need valid time for validation
                    newEndTime: new Date("2025-07-04T17:00:00Z"), // Required field
                    scheduleConnect: validIds.id1,
                },
                {
                    id: validIds.id4,
                    originalStartTime: new Date("2025-12-25T09:00:00Z"),
                    newStartTime: new Date("2025-12-26T10:00:00Z"), // Moved to next day
                    newEndTime: new Date("2025-12-26T16:00:00Z"),
                    scheduleConnect: validIds.id1,
                },
            ],
            recurrencesCreate: [
                {
                    id: validIds.id5,
                    recurrenceType: ScheduleRecurrenceType.Daily,
                    interval: 1,
                    duration: 60, // 1 hour in minutes
                    dayOfWeek: null,
                    dayOfMonth: null,
                    month: null,
                    endDate: new Date("2025-12-31T23:59:59Z"),
                    scheduleConnect: validIds.id1,
                },
            ],
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 123,
            unknownField3: true,
        } as unknown as ScheduleCreateInput,
        update: {
            id: validIds.id1,
            startTime: new Date("2025-02-01T10:00:00Z"),
            endTime: new Date("2025-11-30T16:00:00Z"),
            timezone: "Europe/London",
            exceptionsCreate: [{
                id: validIds.id6,
                originalStartTime: new Date("2025-05-01T10:00:00Z"),
                newStartTime: new Date("2025-05-01T10:00:00Z"), // Need valid time for validation
                newEndTime: new Date("2025-05-01T16:00:00Z"), // Required field
                scheduleConnect: validIds.id1,
            }],
            exceptionsUpdate: [{
                id: validIds.id3,
                newStartTime: new Date("2025-07-05T11:00:00Z"), // Changed time
            }],
            exceptionsDelete: [validIds.id4],
            recurrencesCreate: [{
                id: validIds.id7,
                recurrenceType: ScheduleRecurrenceType.Weekly,
                interval: 2,
                duration: 120, // 2 hours in minutes
                dayOfWeek: 1, // Monday
                dayOfMonth: null,
                month: null,
                endDate: new Date("2025-06-30T23:59:59Z"),
                scheduleConnect: validIds.id1,
            }],
            recurrencesUpdate: [{
                id: validIds.id5,
                interval: 2, // Changed from daily to every other day
            }],
            recurrencesDelete: [validIds.id8],
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 456,
        } as unknown as ScheduleUpdateInput,
    },
    invalid: {
        missingRequired: {
            create: {
                id: validIds.id1,
                // Missing required timezone
                startTime: new Date("2025-01-01T09:00:00Z"),
            } as ScheduleCreateInput,
            update: {
                // Missing required id
                timezone: "America/New_York",
            } as ScheduleUpdateInput,
        },
        invalidTypes: {
            create: {
                // @ts-expect-error - Testing invalid types
                id: 123, // Should be string
                // @ts-expect-error - Testing invalid types
                timezone: 123, // Should be string
                // @ts-expect-error - Testing invalid types
                startTime: "not-a-date", // Should be Date
                // @ts-expect-error - Testing invalid types
                endTime: "invalid-date", // Should be Date
            } as unknown as ScheduleCreateInput,
            update: {
                id: validIds.id1,
                // @ts-expect-error - Testing invalid types
                timezone: true, // Should be string
                // @ts-expect-error - Testing invalid types
                startTime: "2025-01-01", // Should be Date object
                // @ts-expect-error - Testing invalid types
                endTime: 12345, // Should be Date
            } as unknown as ScheduleUpdateInput,
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                timezone: "America/New_York",
            },
            update: {
                id: "invalid-id",
            },
        },
        invalidTimezone: {
            create: {
                id: validIds.id1,
                timezone: "", // Too short
            },
            update: {
                id: validIds.id1,
                timezone: "A".repeat(TIMEZONE_TOO_LONG_LENGTH), // Too long (max 64)
            },
        },
        invalidTimeOrder: {
            create: {
                id: validIds.id1,
                timezone: "America/New_York",
                startTime: new Date("2025-12-31T17:00:00Z"),
                endTime: new Date("2025-01-01T09:00:00Z"), // Before start time
            },
            update: {
                id: validIds.id1,
                startTime: new Date("2025-06-01T17:00:00Z"),
                endTime: new Date("2025-06-01T17:00:00Z"), // Same as start time
            },
        },
        invalidException: {
            create: {
                id: validIds.id1,
                timezone: "America/New_York",
                startTime: new Date("2025-01-01T00:00:00Z"), // Need startTime
                exceptionsCreate: [{
                    id: validIds.id3,
                    // Missing required originalStartTime
                    newStartTime: new Date("2025-07-05T10:00:00Z"),
                    // @ts-expect-error - Missing required originalStartTime for testing
                }],
            } as unknown as ScheduleCreateInput,
        },
        invalidRecurrence: {
            create: {
                id: validIds.id1,
                timezone: "America/New_York",
                startTime: new Date("2025-01-01T00:00:00Z"), // Need startTime
                recurrencesCreate: [{
                    id: validIds.id5,
                    // Missing required recurrenceType and duration
                    interval: 1,
                    // @ts-expect-error - Missing required recurrenceType and duration for testing
                }],
            } as unknown as ScheduleCreateInput,
        },
    },
    edgeCases: {
        emptySchedule: {
            create: {
                id: validIds.id1,
                timezone: "UTC",
                startTime: new Date("2025-01-01T00:00:00Z"), // Adding startTime to avoid endTime validation issues
                // No exceptions or recurrences
            },
            update: {
                id: validIds.id1,
                startTime: new Date("2025-01-01T00:00:00Z"), // Adding startTime to avoid endTime validation issues
                // No changes
            },
        },
        onlyStartTime: {
            create: {
                id: validIds.id1,
                timezone: "America/New_York",
                startTime: new Date("2025-01-01T00:00:00Z"),
                // No end time
            },
        },
        onlyEndTime: {
            create: {
                id: validIds.id1,
                timezone: "America/New_York",
                endTime: new Date("2025-12-31T23:59:59Z"),
                // No start time - should fail validation
            } as ScheduleCreateInput,
        },
        maxLengthTimezone: {
            create: {
                id: validIds.id1,
                timezone: "A".repeat(TIMEZONE_MAX_LENGTH), // Maximum length
                startTime: new Date("2025-01-01T00:00:00Z"),
            },
        },
        differentTimezones: {
            create: {
                id: validIds.id1,
                timezone: "Pacific/Auckland",
                startTime: new Date("2025-01-01T00:00:00Z"),
                endTime: new Date("2025-12-31T23:59:59Z"),
            },
            update: {
                id: validIds.id1,
                timezone: "Asia/Tokyo",
                startTime: new Date("2025-01-01T00:00:00Z"), // Need startTime
            },
        },
        multipleConnections: {
            create: {
                id: validIds.id1,
                timezone: "America/New_York",
                startTime: new Date("2025-01-01T00:00:00Z"), // Adding startTime
                meetingConnect: validIds.id2,
                runConnect: validIds.id3, // Can't have both
            } as ScheduleCreateInput,
        },
        withRun: {
            create: {
                id: validIds.id1,
                timezone: "America/New_York",
                startTime: new Date("2025-01-01T00:00:00Z"), // Adding startTime
                runConnect: validIds.id2,
            },
        },
        manyExceptions: {
            create: {
                id: validIds.id1,
                timezone: "America/New_York",
                startTime: new Date("2025-01-01T00:00:00Z"), // Adding startTime
                exceptionsCreate: Array.from({ length: 10 }, (_, i) => ({
                    id: `10000000000000000${i}`,
                    originalStartTime: new Date(`2025-${String(i + 1).padStart(2, "0")}-15T09:00:00Z`),
                    newStartTime: new Date(`2025-${String(i + 1).padStart(2, "0")}-16T10:00:00Z`),
                    newEndTime: new Date(`2025-${String(i + 1).padStart(2, "0")}-16T17:00:00Z`),
                    scheduleConnect: validIds.id1,
                })),
            },
        },
        manyRecurrences: {
            create: {
                id: validIds.id1,
                timezone: "America/New_York",
                startTime: new Date("2025-01-01T00:00:00Z"), // Adding startTime
                recurrencesCreate: [
                    {
                        id: validIds.id5,
                        recurrenceType: ScheduleRecurrenceType.Daily,
                        interval: 1,
                        duration: 60, // 1 hour in minutes
                        dayOfWeek: null,
                        dayOfMonth: null,
                        month: null,
                        endDate: new Date("2025-03-31T23:59:59Z"),
                        scheduleConnect: validIds.id1,
                    },
                    {
                        id: validIds.id6,
                        recurrenceType: ScheduleRecurrenceType.Weekly,
                        interval: 1,
                        duration: 90, // 1.5 hours in minutes
                        dayOfWeek: 3, // Wednesday
                        dayOfMonth: null,
                        month: null,
                        endDate: new Date("2025-06-30T23:59:59Z"),
                        scheduleConnect: validIds.id1,
                    },
                    {
                        id: validIds.id7,
                        recurrenceType: ScheduleRecurrenceType.Monthly,
                        interval: 1,
                        duration: 180, // 3 hours in minutes
                        dayOfWeek: null,
                        dayOfMonth: 15,
                        month: null,
                        endDate: new Date("2025-12-31T23:59:59Z"),
                        scheduleConnect: validIds.id1,
                    },
                ],
            },
        },
        complexUpdate: {
            update: {
                id: validIds.id1,
                timezone: "Europe/Paris",
                startTime: new Date("2025-01-01T00:00:00Z"), // Need startTime
                exceptionsCreate: [
                    {
                        id: validIds.id8,
                        originalStartTime: new Date("2025-08-15T09:00:00Z"),
                        newStartTime: new Date("2025-08-15T09:00:00Z"), // Need valid time for validation
                        newEndTime: new Date("2025-08-15T17:00:00Z"), // Required field
                        scheduleConnect: validIds.id1,
                    },
                ],
                exceptionsUpdate: [
                    {
                        id: validIds.id3,
                        newStartTime: new Date("2025-07-04T14:00:00Z"),
                    },
                    {
                        id: validIds.id4,
                        newStartTime: new Date("2025-12-26T10:00:00Z"), // Need startTime for endTime validation
                        newEndTime: new Date("2025-12-26T15:00:00Z"),
                    },
                ],
                exceptionsDelete: [validIds.id5, validIds.id6],
                recurrencesCreate: [
                    {
                        id: validIds.id9,
                        recurrenceType: ScheduleRecurrenceType.Yearly,
                        interval: 1,
                        duration: 240, // 4 hours in minutes
                        dayOfWeek: null,
                        dayOfMonth: 1,
                        month: 1, // January
                        endDate: null, // No end date
                        scheduleConnect: validIds.id1,
                    },
                ],
                recurrencesUpdate: [
                    {
                        id: validIds.id7,
                        interval: 3,
                    },
                ],
                recurrencesDelete: [validIds.id10],
            },
        },
        whitespaceTimezone: {
            create: {
                id: validIds.id1,
                timezone: "   America/New_York   ", // Should be trimmed
                startTime: new Date("2025-01-01T00:00:00Z"), // Adding startTime
            },
        },
        minimalValidTimeRange: {
            create: {
                id: validIds.id1,
                timezone: "UTC",
                startTime: new Date("2025-01-01T00:00:00Z"),
                endTime: new Date("2025-01-01T00:00:01Z"), // Just 1 second after
            },
        },
        largeValidTimeRange: {
            create: {
                id: validIds.id1,
                timezone: "UTC",
                startTime: new Date("2025-01-01T00:00:00Z"),
                endTime: new Date("2030-12-31T23:59:59Z"), // 5+ years
            },
        },
    },
};

// Custom factory that always generates valid IDs with proper typing
const customizers = {
    create: (base: ScheduleCreateInput): ScheduleCreateInput => ({
        ...base,
        id: base.id || testValues.snowflakeId(),
        timezone: base.timezone || "UTC",
    }),
    update: (base: ScheduleUpdateInput): ScheduleUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export enhanced type-safe factories
export const scheduleTestDataFactory = new TypedTestDataFactory(scheduleFixtures, scheduleValidation, customizers);

// Export type-safe fixtures with validation capabilities
export const typedScheduleFixtures = createTypedFixtures(scheduleFixtures, scheduleValidation);
