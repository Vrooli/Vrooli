import { generatePK } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { DbTestFixtures, DbErrorScenarios } from "./types.js";

/**
 * Enhanced test fixtures for ScheduleException model following standard structure
 */
export const scheduleExceptionDbFixtures: DbTestFixtures<Prisma.schedule_exceptionCreateInput> = {
    minimal: {
        id: generatePK(),
        originalStartTime: new Date("2025-07-04T09:00:00Z"),
        schedule: { connect: { id: "schedule_placeholder_id" } },
    },
    complete: {
        id: generatePK(),
        originalStartTime: new Date("2025-12-25T09:00:00Z"),
        newStartTime: new Date("2025-12-26T11:00:00Z"),
        newEndTime: new Date("2025-12-26T15:00:00Z"),
        schedule: { connect: { id: "schedule_placeholder_id" } },
    },
    invalid: {
        missingRequired: {
            // Missing required originalStartTime and schedule
            id: generatePK(),
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            originalStartTime: "not-a-date", // Should be Date
            newStartTime: 123, // Should be Date or null
            newEndTime: "invalid", // Should be Date or null
            schedule: "not-an-object", // Should be object
        },
        invalidTimeRange: {
            id: generatePK(),
            originalStartTime: new Date(),
            newStartTime: new Date(Date.now() + 2 * 60 * 60 * 1000), // 2 hours later
            newEndTime: new Date(Date.now() + 60 * 60 * 1000), // 1 hour later (before start)
            schedule: { connect: { id: "schedule_placeholder_id" } },
        },
    },
    edgeCases: {
        cancelledOccurrence: {
            id: generatePK(),
            originalStartTime: new Date("2025-08-15T14:00:00Z"),
            newStartTime: null,
            newEndTime: null,
            schedule: { connect: { id: "schedule_placeholder_id" } },
        },
        holidayException: {
            id: generatePK(),
            originalStartTime: new Date("2025-12-25T09:00:00Z"), // Christmas
            newStartTime: null,
            newEndTime: null,
            schedule: { connect: { id: "schedule_placeholder_id" } },
        },
        extendedDuration: {
            id: generatePK(),
            originalStartTime: new Date("2025-06-15T10:00:00Z"),
            newStartTime: new Date("2025-06-15T10:00:00Z"), // Same start
            newEndTime: new Date("2025-06-15T18:00:00Z"), // Extended to 8 hours
            schedule: { connect: { id: "schedule_placeholder_id" } },
        },
        rescheduledToNextWeek: {
            id: generatePK(),
            originalStartTime: new Date("2025-05-01T14:00:00Z"),
            newStartTime: new Date("2025-05-08T14:00:00Z"), // Next week same time
            newEndTime: new Date("2025-05-08T15:00:00Z"),
            schedule: { connect: { id: "schedule_placeholder_id" } },
        },
        movedEarlier: {
            id: generatePK(),
            originalStartTime: new Date("2025-07-10T15:00:00Z"),
            newStartTime: new Date("2025-07-10T09:00:00Z"), // 6 hours earlier
            newEndTime: new Date("2025-07-10T11:00:00Z"),
            schedule: { connect: { id: "schedule_placeholder_id" } },
        },
        shortenedDuration: {
            id: generatePK(),
            originalStartTime: new Date("2025-09-20T13:00:00Z"),
            newStartTime: new Date("2025-09-20T13:00:00Z"), // Same start
            newEndTime: new Date("2025-09-20T14:00:00Z"), // Shortened to 1 hour
            schedule: { connect: { id: "schedule_placeholder_id" } },
        },
        multipleDayEvent: {
            id: generatePK(),
            originalStartTime: new Date("2025-10-01T09:00:00Z"),
            newStartTime: new Date("2025-10-15T09:00:00Z"), // 2 weeks later
            newEndTime: new Date("2025-10-17T17:00:00Z"), // 3-day event
            schedule: { connect: { id: "schedule_placeholder_id" } },
        },
    },
};

/**
 * Enhanced factory for creating schedule exception database fixtures
 */
export class ScheduleExceptionDbFactory extends EnhancedDbFactory<Prisma.schedule_exceptionCreateInput> {
    
    /**
     * Get the test fixtures for ScheduleException model
     */
    protected getFixtures(): DbTestFixtures<Prisma.schedule_exceptionCreateInput> {
        return scheduleExceptionDbFixtures;
    }

    /**
     * Get ScheduleException-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: this.generateId(),
                    originalStartTime: new Date(),
                    schedule: { connect: { id: "schedule_placeholder_id" } },
                },
                foreignKeyViolation: {
                    id: this.generateId(),
                    originalStartTime: new Date(),
                    schedule: { connect: { id: "non-existent-schedule-id" } },
                },
                checkConstraintViolation: {
                    id: this.generateId(),
                    originalStartTime: new Date(),
                    newStartTime: new Date(Date.now() + 2 * 60 * 60 * 1000),
                    newEndTime: new Date(Date.now() + 60 * 60 * 1000), // End before start
                    schedule: { connect: { id: "schedule_placeholder_id" } },
                },
            },
            validation: {
                requiredFieldMissing: scheduleExceptionDbFixtures.invalid.missingRequired,
                invalidDataType: scheduleExceptionDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: this.generateId(),
                    originalStartTime: new Date("1900-01-01"), // Too far in past
                    schedule: { connect: { id: "schedule_placeholder_id" } },
                },
            },
            businessLogic: {
                endBeforeStart: scheduleExceptionDbFixtures.invalid.invalidTimeRange,
                exceptionWithoutRecurrence: {
                    id: this.generateId(),
                    originalStartTime: new Date(),
                    schedule: { connect: { id: "one-time-schedule-id" } }, // Non-recurring schedule
                },
                duplicateException: {
                    id: this.generateId(),
                    originalStartTime: new Date("2025-06-01T10:00:00Z"), // Same as another exception
                    schedule: { connect: { id: "schedule_placeholder_id" } },
                },
            },
        };
    }

    /**
     * Generate fresh identifiers specific to ScheduleException
     */
    protected generateFreshIdentifiers(): Record<string, any> {
        return {
            id: this.generateId(),
        };
    }

    /**
     * ScheduleException-specific validation
     */
    protected validateSpecific(data: Prisma.schedule_exceptionCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields
        if (!data.originalStartTime) errors.push("Original start time is required");
        if (!data.schedule) errors.push("Schedule association is required");

        // Check business logic
        if (data.newStartTime && data.newEndTime && data.newStartTime >= data.newEndTime) {
            errors.push("New start time must be before new end time");
        }

        if (!data.newStartTime && data.newEndTime) {
            warnings.push("New end time specified without new start time");
        }

        if (data.newStartTime && !data.newEndTime) {
            warnings.push("New start time specified without new end time - consider adding duration");
        }

        // Check for cancellation
        if (!data.newStartTime && !data.newEndTime) {
            warnings.push("No new times specified - this represents a cancelled occurrence");
        }

        return { errors, warnings };
    }

    /**
     * Add schedule association to exception
     */
    protected addScheduleAssociation(data: Prisma.schedule_exceptionCreateInput, scheduleId: string): Prisma.schedule_exceptionCreateInput {
        return {
            ...data,
            schedule: { connect: { id: scheduleId } },
        };
    }

    // Static factory methods for backward compatibility and convenience

    static createMinimal(
        scheduleId: string,
        overrides?: Partial<Prisma.schedule_exceptionCreateInput>,
    ): Prisma.schedule_exceptionCreateInput {
        const factory = new ScheduleExceptionDbFactory();
        const data = factory.createMinimal(overrides);
        return factory.addScheduleAssociation(data, scheduleId);
    }

    static createComplete(
        scheduleId: string,
        overrides?: Partial<Prisma.schedule_exceptionCreateInput>,
    ): Prisma.schedule_exceptionCreateInput {
        const factory = new ScheduleExceptionDbFactory();
        const data = factory.createComplete(overrides);
        return factory.addScheduleAssociation(data, scheduleId);
    }

    static createCancellation(
        scheduleId: string,
        originalTime: Date,
        overrides?: Partial<Prisma.schedule_exceptionCreateInput>,
    ): Prisma.schedule_exceptionCreateInput {
        const factory = new ScheduleExceptionDbFactory();
        return factory.createMinimal({
            originalStartTime: originalTime,
            newStartTime: null,
            newEndTime: null,
            schedule: { connect: { id: scheduleId } },
            ...overrides,
        });
    }

    static createRescheduled(
        scheduleId: string,
        originalTime: Date,
        newStartTime: Date,
        newEndTime: Date,
        overrides?: Partial<Prisma.schedule_exceptionCreateInput>,
    ): Prisma.schedule_exceptionCreateInput {
        const factory = new ScheduleExceptionDbFactory();
        return factory.createMinimal({
            originalStartTime: originalTime,
            newStartTime,
            newEndTime,
            schedule: { connect: { id: scheduleId } },
            ...overrides,
        });
    }

    static createExtended(
        scheduleId: string,
        originalTime: Date,
        extensionHours = 2,
        overrides?: Partial<Prisma.schedule_exceptionCreateInput>,
    ): Prisma.schedule_exceptionCreateInput {
        const newEndTime = new Date(originalTime.getTime() + (extensionHours * 60 * 60 * 1000));
        const factory = new ScheduleExceptionDbFactory();
        return factory.createMinimal({
            originalStartTime: originalTime,
            newStartTime: originalTime, // Same start time
            newEndTime,
            schedule: { connect: { id: scheduleId } },
            ...overrides,
        });
    }

    static createMovedToNextDay(
        scheduleId: string,
        originalTime: Date,
        durationHours = 8,
        overrides?: Partial<Prisma.schedule_exceptionCreateInput>,
    ): Prisma.schedule_exceptionCreateInput {
        const nextDay = new Date(originalTime);
        nextDay.setDate(nextDay.getDate() + 1);
        const newEndTime = new Date(nextDay.getTime() + (durationHours * 60 * 60 * 1000));
        
        const factory = new ScheduleExceptionDbFactory();
        return factory.createMinimal({
            originalStartTime: originalTime,
            newStartTime: nextDay,
            newEndTime,
            schedule: { connect: { id: scheduleId } },
            ...overrides,
        });
    }

    /**
     * Create exception for holiday
     */
    static createHolidayException(
        scheduleId: string,
        holidayDate: Date,
        holidayName?: string,
        overrides?: Partial<Prisma.schedule_exceptionCreateInput>,
    ): Prisma.schedule_exceptionCreateInput {
        return ScheduleExceptionDbFactory.createCancellation(scheduleId, holidayDate, {
            // Could add metadata in future if schema supports it
            ...overrides,
        });
    }

    /**
     * Create batch of exceptions for testing
     */
    static createBatch(
        scheduleId: string,
        exceptions: Array<{
            type: "cancel" | "reschedule" | "extend" | "shorten";
            originalDate: Date;
            newStartTime?: Date;
            newEndTime?: Date;
            extensionHours?: number;
        }>,
    ): Prisma.schedule_exceptionCreateInput[] {
        return exceptions.map(exception => {
            switch (exception.type) {
                case "cancel":
                    return ScheduleExceptionDbFactory.createCancellation(
                        scheduleId,
                        exception.originalDate,
                    );
                case "reschedule":
                    return ScheduleExceptionDbFactory.createRescheduled(
                        scheduleId,
                        exception.originalDate,
                        exception.newStartTime!,
                        exception.newEndTime!,
                    );
                case "extend":
                    return ScheduleExceptionDbFactory.createExtended(
                        scheduleId,
                        exception.originalDate,
                        exception.extensionHours,
                    );
                case "shorten":
                    const duration = 60 * 60 * 1000; // 1 hour default
                    return ScheduleExceptionDbFactory.createRescheduled(
                        scheduleId,
                        exception.originalDate,
                        exception.originalDate,
                        new Date(exception.originalDate.getTime() + duration),
                    );
            }
        });
    }
}

/**
 * Common exception patterns for testing
 */
export const scheduleExceptionPatterns = {
    /**
     * US Federal holidays for a year
     */
    usFederalHolidays: (scheduleId: string, year: number) => [
        ScheduleExceptionDbFactory.createHolidayException(scheduleId, new Date(`${year}-01-01`), "New Year's Day"),
        ScheduleExceptionDbFactory.createHolidayException(scheduleId, new Date(`${year}-07-04`), "Independence Day"),
        ScheduleExceptionDbFactory.createHolidayException(scheduleId, new Date(`${year}-11-11`), "Veterans Day"),
        ScheduleExceptionDbFactory.createHolidayException(scheduleId, new Date(`${year}-12-25`), "Christmas Day"),
    ],

    /**
     * Conference week - all meetings moved
     */
    conferenceWeek: (scheduleId: string, weekStart: Date) => {
        const exceptions = [];
        for (let i = 0; i < 5; i++) { // Monday to Friday
            const date = new Date(weekStart);
            date.setDate(date.getDate() + i);
            exceptions.push(ScheduleExceptionDbFactory.createCancellation(scheduleId, date));
        }
        return exceptions;
    },

    /**
     * Summer schedule - meetings shortened
     */
    summerSchedule: (scheduleId: string, startDate: Date, weeks = 12) => {
        const exceptions = [];
        for (let w = 0; w < weeks; w++) {
            const date = new Date(startDate);
            date.setDate(date.getDate() + (w * 7));
            exceptions.push(ScheduleExceptionDbFactory.createExtended(scheduleId, date, -0.5)); // 30 min shorter
        }
        return exceptions;
    },
};
