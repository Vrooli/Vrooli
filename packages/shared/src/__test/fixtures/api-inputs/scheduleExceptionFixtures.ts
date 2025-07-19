import type { ScheduleExceptionCreateInput, ScheduleExceptionUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TestDataFactory, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { scheduleExceptionValidation } from "../../../validation/models/scheduleException.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
    id6: "123456789012345683",
};

// Shared schedule exception test fixtures
export const scheduleExceptionFixtures: ModelTestFixtures<ScheduleExceptionCreateInput, ScheduleExceptionUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            originalStartTime: new Date("2025-07-04T09:00:00Z").toISOString(), // Required by API type
            newStartTime: new Date("2025-07-05T10:00:00Z").toISOString(), // Need startTime for endTime validation
            newEndTime: new Date("2025-07-05T17:00:00Z").toISOString(), // Required field
            scheduleConnect: validIds.id2,
        },
        update: {
            id: validIds.id1,
            newStartTime: new Date("2025-07-05T10:00:00Z").toISOString(), // Need startTime for endTime validation
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            originalStartTime: new Date("2025-07-04T09:00:00Z").toISOString(), // Original scheduled time
            newStartTime: new Date("2025-07-05T10:00:00Z").toISOString(), // Moved to next day
            newEndTime: new Date("2025-07-05T16:00:00Z").toISOString(), // Shorter duration
            scheduleConnect: validIds.id2,
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 123,
            unknownField3: true,
        } as ScheduleExceptionCreateInput,
        update: {
            id: validIds.id1,
            originalStartTime: new Date("2025-12-25T09:00:00Z").toISOString(),
            newStartTime: new Date("2025-12-26T11:00:00Z").toISOString(), // Changed time
            newEndTime: new Date("2025-12-26T15:00:00Z").toISOString(), // Changed duration
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 456,
        } as ScheduleExceptionUpdateInput,
    },
    invalid: {
        missingRequired: {
            create: {
                id: validIds.id1,
                // Missing required originalStartTime, newEndTime and scheduleConnect
                newStartTime: new Date("2025-07-05T10:00:00Z").toISOString(),
            } as ScheduleExceptionCreateInput,
            update: {
                // Missing required id
                newStartTime: new Date("2025-07-05T10:00:00Z").toISOString(),
                newEndTime: new Date("2025-07-05T17:00:00Z").toISOString(),
            } as ScheduleExceptionUpdateInput,
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                originalStartTime: "not-a-date", // Should be Date
                newStartTime: "invalid-date", // Should be Date
                newEndTime: 12345, // Should be Date
                scheduleConnect: validIds.id2,
            } as unknown as ScheduleExceptionCreateInput,
            update: {
                id: validIds.id1,
                originalStartTime: "2025-07-04", // Should be Date object
                newStartTime: 123, // Should be Date
                newEndTime: true, // Should be Date
            } as unknown as ScheduleExceptionUpdateInput,
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                newStartTime: new Date("2025-07-05T10:00:00Z").toISOString(),
                newEndTime: new Date("2025-07-05T17:00:00Z").toISOString(),
                scheduleConnect: validIds.id2,
            } as ScheduleExceptionCreateInput,
            update: {
                id: "invalid-id",
            } as ScheduleExceptionUpdateInput,
        },
        invalidTimeOrder: {
            create: {
                id: validIds.id1,
                newStartTime: new Date("2025-07-05T17:00:00Z").toISOString(),
                newEndTime: new Date("2025-07-05T10:00:00Z").toISOString(), // Before start time
                scheduleConnect: validIds.id2,
            } as ScheduleExceptionCreateInput,
            update: {
                id: validIds.id1,
                newStartTime: new Date("2025-07-05T12:00:00Z").toISOString(),
                newEndTime: new Date("2025-07-05T12:00:00Z").toISOString(), // Same as start time
            } as ScheduleExceptionUpdateInput,
        },
        missingScheduleConnect: {
            create: {
                id: validIds.id1,
                newStartTime: new Date("2025-07-05T10:00:00Z").toISOString(),
                newEndTime: new Date("2025-07-05T17:00:00Z").toISOString(),
                // Missing required scheduleConnect
            } as ScheduleExceptionCreateInput,
        },
        endTimeWithoutStartTime: {
            create: {
                id: validIds.id1,
                // No newStartTime
                newEndTime: new Date("2025-07-05T17:00:00Z").toISOString(),
                scheduleConnect: validIds.id2,
            } as ScheduleExceptionCreateInput,
        },
    },
    edgeCases: {
        cancelledMeeting: {
            create: {
                id: validIds.id1,
                originalStartTime: new Date("2025-07-04T09:00:00Z").toISOString(),
                // For cancelled meetings, we still need valid times due to validation constraints
                newStartTime: new Date("2025-07-04T09:00:00Z").toISOString(), // Same as original
                newEndTime: new Date("2025-07-04T09:01:00Z").toISOString(), // Just 1 second meeting
                scheduleConnect: validIds.id2,
            },
            update: {
                id: validIds.id1,
                // Updating a cancellation
                newStartTime: new Date("2025-07-04T09:00:00Z").toISOString(),
                newEndTime: new Date("2025-07-04T09:01:00Z").toISOString(),
            },
        },
        movedEarlier: {
            create: {
                id: validIds.id1,
                originalStartTime: new Date("2025-07-04T14:00:00Z").toISOString(),
                newStartTime: new Date("2025-07-04T09:00:00Z").toISOString(), // Moved earlier same day
                newEndTime: new Date("2025-07-04T16:00:00Z").toISOString(),
                scheduleConnect: validIds.id2,
            },
        },
        movedLater: {
            create: {
                id: validIds.id1,
                originalStartTime: new Date("2025-07-04T09:00:00Z").toISOString(),
                newStartTime: new Date("2025-07-04T14:00:00Z").toISOString(), // Moved later same day
                newEndTime: new Date("2025-07-04T21:00:00Z").toISOString(),
                scheduleConnect: validIds.id2,
            },
        },
        extendedDuration: {
            create: {
                id: validIds.id1,
                originalStartTime: new Date("2025-07-04T09:00:00Z").toISOString(),
                newStartTime: new Date("2025-07-04T09:00:00Z").toISOString(), // Same start
                newEndTime: new Date("2025-07-04T18:00:00Z").toISOString(), // Extended end time
                scheduleConnect: validIds.id2,
            },
        },
        shortenedDuration: {
            create: {
                id: validIds.id1,
                originalStartTime: new Date("2025-07-04T09:00:00Z").toISOString(),
                newStartTime: new Date("2025-07-04T09:00:00Z").toISOString(), // Same start
                newEndTime: new Date("2025-07-04T12:00:00Z").toISOString(), // Shortened end time
                scheduleConnect: validIds.id2,
            },
        },
        differentDay: {
            create: {
                id: validIds.id1,
                originalStartTime: new Date("2025-07-04T09:00:00Z").toISOString(),
                newStartTime: new Date("2025-07-10T14:00:00Z").toISOString(), // Different day entirely
                newEndTime: new Date("2025-07-10T16:00:00Z").toISOString(),
                scheduleConnect: validIds.id2,
            },
        },
        differentMonth: {
            create: {
                id: validIds.id1,
                originalStartTime: new Date("2025-07-04T09:00:00Z").toISOString(),
                newStartTime: new Date("2025-08-15T10:00:00Z").toISOString(), // Different month
                newEndTime: new Date("2025-08-15T17:00:00Z").toISOString(),
                scheduleConnect: validIds.id2,
            },
        },
        differentYear: {
            create: {
                id: validIds.id1,
                originalStartTime: new Date("2025-07-04T09:00:00Z").toISOString(),
                newStartTime: new Date("2026-07-04T09:00:00Z").toISOString(), // Next year
                newEndTime: new Date("2026-07-04T17:00:00Z").toISOString(),
                scheduleConnect: validIds.id2,
            },
        },
        minimalTimeRange: {
            create: {
                id: validIds.id1,
                originalStartTime: new Date("2025-07-04T08:00:00Z").toISOString(),
                newStartTime: new Date("2025-07-04T09:00:00Z").toISOString(),
                newEndTime: new Date("2025-07-04T09:00:01Z").toISOString(), // Just 1 second duration
                scheduleConnect: validIds.id2,
            },
        },
        largeTimeRange: {
            create: {
                id: validIds.id1,
                originalStartTime: new Date("2025-07-03T09:00:00Z").toISOString(),
                newStartTime: new Date("2025-07-04T00:00:00Z").toISOString(),
                newEndTime: new Date("2025-07-04T23:59:59Z").toISOString(), // All day
                scheduleConnect: validIds.id2,
            },
        },
        multiDayEvent: {
            create: {
                id: validIds.id1,
                originalStartTime: new Date("2025-07-04T09:00:00Z").toISOString(),
                newStartTime: new Date("2025-07-04T09:00:00Z").toISOString(),
                newEndTime: new Date("2025-07-06T17:00:00Z").toISOString(), // Multi-day event
                scheduleConnect: validIds.id2,
            },
        },
        differentScheduleId: {
            create: {
                id: validIds.id1,
                originalStartTime: new Date("2025-07-04T08:00:00Z").toISOString(),
                newStartTime: new Date("2025-07-04T09:00:00Z").toISOString(),
                newEndTime: new Date("2025-07-04T17:00:00Z").toISOString(),
                scheduleConnect: validIds.id6, // Different schedule
            },
        },
        onlyRequiredInUpdate: {
            update: {
                id: validIds.id1,
                newStartTime: new Date("2025-07-05T10:00:00Z").toISOString(), // Need startTime for endTime validation
                // Only id is required in update, but we need startTime for validation
            },
        },
        updateOnlyOriginalTime: {
            update: {
                id: validIds.id1,
                originalStartTime: new Date("2025-09-01T10:00:00Z").toISOString(),
                newStartTime: new Date("2025-07-05T10:00:00Z").toISOString(), // Need startTime for endTime validation
                // Not updating new times
            },
        },
        updateOnlyNewTimes: {
            update: {
                id: validIds.id1,
                newStartTime: new Date("2025-09-01T11:00:00Z").toISOString(),
                newEndTime: new Date("2025-09-01T16:00:00Z").toISOString(),
                // Not updating original time
            },
        },
    },
};

// Custom factory that always generates valid IDs and required fields
const customizers = {
    create: (base: ScheduleExceptionCreateInput): ScheduleExceptionCreateInput => ({
        ...base,
        id: base.id || validIds.id1,
        newStartTime: base.newStartTime || new Date("2025-01-01T10:00:00Z").toISOString(),
        newEndTime: base.newEndTime || new Date("2025-01-01T17:00:00Z").toISOString(),
        scheduleConnect: base.scheduleConnect || validIds.id2,
        originalStartTime: base.originalStartTime || new Date("2025-01-01T09:00:00Z").toISOString(),
    }),
    update: (base: ScheduleExceptionUpdateInput): ScheduleExceptionUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export factories for creating test data programmatically
export const scheduleExceptionTestDataFactory = new TypedTestDataFactory(scheduleExceptionFixtures, scheduleExceptionValidation, customizers);
export const typedScheduleExceptionFixtures = createTypedFixtures(scheduleExceptionFixtures, scheduleExceptionValidation);

// Legacy export for backward compatibility
export const legacyScheduleExceptionTestDataFactory = new TestDataFactory(scheduleExceptionFixtures, customizers);
