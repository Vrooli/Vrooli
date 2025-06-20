import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { DbTestFixtures, BulkSeedOptions, BulkSeedResult, DbErrorScenarios } from "./types.js";

/**
 * Database fixtures for Schedule model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const scheduleDbIds = {
    schedule1: generatePK(),
    schedule2: generatePK(),
    schedule3: generatePK(),
    recurrence1: generatePK(),
    recurrence2: generatePK(),
    exception1: generatePK(),
    exception2: generatePK(),
};

/**
 * Enhanced test fixtures for Schedule model following standard structure
 */
export const scheduleDbFixtures: DbTestFixtures<Prisma.ScheduleCreateInput> = {
    minimal: {
        id: generatePK(),
        publicId: generatePublicId(),
        startTime: new Date(),
        endTime: new Date(Date.now() + 60 * 60 * 1000), // 1 hour later
        timezone: "UTC",
    },
    complete: {
        id: generatePK(),
        publicId: generatePublicId(),
        startTime: new Date(),
        endTime: new Date(Date.now() + 2 * 60 * 60 * 1000), // 2 hours later
        timezone: "America/New_York",
        meeting: { connect: { id: "meeting_placeholder_id" } },
        recurrences: {
            create: {
                id: generatePK(),
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 1, // Monday
                dayOfMonth: null,
                month: null,
                endDate: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000), // 1 year from now
            },
        },
        exceptions: {
            create: [
                {
                    id: generatePK(),
                    originalStartTime: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000),
                    newStartTime: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000 + 2 * 60 * 60 * 1000), // 2 hours later
                    newEndTime: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000 + 4 * 60 * 60 * 1000), // 4 hours later
                },
            ],
        },
    },
    invalid: {
        missingRequired: {
            // Missing required startTime, endTime, timezone
            publicId: generatePublicId(),
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            publicId: 123, // Should be string
            startTime: "not-a-date", // Should be Date
            endTime: "not-a-date", // Should be Date
            timezone: 123, // Should be string
        },
        invalidTimeRange: {
            id: generatePK(),
            publicId: generatePublicId(),
            startTime: new Date(Date.now() + 60 * 60 * 1000), // Start after end
            endTime: new Date(), // End before start
            timezone: "UTC",
        },
        invalidTimezone: {
            id: generatePK(),
            publicId: generatePublicId(),
            startTime: new Date(),
            endTime: new Date(Date.now() + 60 * 60 * 1000),
            timezone: "Invalid/Timezone",
        },
    },
    edgeCases: {
        longDurationSchedule: {
            id: generatePK(),
            publicId: generatePublicId(),
            startTime: new Date(),
            endTime: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000), // 30 days duration
            timezone: "UTC",
            runProject: { connect: { id: "project_placeholder_id" } },
        },
        dailyRecurringSchedule: {
            id: generatePK(),
            publicId: generatePublicId(),
            startTime: new Date(),
            endTime: new Date(Date.now() + 60 * 60 * 1000),
            timezone: "UTC",
            recurrences: {
                create: {
                    id: generatePK(),
                    recurrenceType: "Daily",
                    interval: 1,
                    dayOfWeek: null,
                    dayOfMonth: null,
                    month: null,
                    endDate: new Date(Date.now() + 90 * 24 * 60 * 60 * 1000), // 3 months
                },
            },
        },
        monthlyRecurringSchedule: {
            id: generatePK(),
            publicId: generatePublicId(),
            startTime: new Date(),
            endTime: new Date(Date.now() + 2 * 60 * 60 * 1000),
            timezone: "Europe/London",
            recurrences: {
                create: {
                    id: generatePK(),
                    recurrenceType: "Monthly",
                    interval: 1,
                    dayOfWeek: null,
                    dayOfMonth: 15, // 15th of each month
                    month: null,
                    endDate: null, // No end date
                },
            },
        },
        scheduleWithMultipleExceptions: {
            id: generatePK(),
            publicId: generatePublicId(),
            startTime: new Date(),
            endTime: new Date(Date.now() + 60 * 60 * 1000),
            timezone: "Asia/Tokyo",
            focusMode: { connect: { id: "focus_mode_placeholder_id" } },
            exceptions: {
                create: [
                    {
                        id: generatePK(),
                        originalStartTime: new Date(Date.now() + 24 * 60 * 60 * 1000),
                        newStartTime: null, // Cancelled
                        newEndTime: null,
                    },
                    {
                        id: generatePK(),
                        originalStartTime: new Date(Date.now() + 48 * 60 * 60 * 1000),
                        newStartTime: new Date(Date.now() + 48 * 60 * 60 * 1000 + 30 * 60 * 1000), // 30 min later
                        newEndTime: new Date(Date.now() + 48 * 60 * 60 * 1000 + 90 * 60 * 1000), // 90 min later
                    },
                ],
            },
        },
        crossTimezoneSchedule: {
            id: generatePK(),
            publicId: generatePublicId(),
            startTime: new Date(),
            endTime: new Date(Date.now() + 60 * 60 * 1000),
            timezone: "Pacific/Auckland",
            runRoutine: { connect: { id: "routine_placeholder_id" } },
        },
    },
};

/**
 * Enhanced factory for creating schedule database fixtures
 */
export class ScheduleDbFactory extends EnhancedDbFactory<Prisma.ScheduleCreateInput> {
    
    /**
     * Get the test fixtures for Schedule model
     */
    protected getFixtures(): DbTestFixtures<Prisma.ScheduleCreateInput> {
        return scheduleDbFixtures;
    }

    /**
     * Get Schedule-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: scheduleDbIds.schedule1, // Duplicate ID
                    publicId: generatePublicId(),
                    startTime: new Date(),
                    endTime: new Date(Date.now() + 60 * 60 * 1000),
                    timezone: "UTC",
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    startTime: new Date(),
                    endTime: new Date(Date.now() + 60 * 60 * 1000),
                    timezone: "UTC",
                    meeting: { connect: { id: "non-existent-meeting-id" } },
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    publicId: "", // Empty publicId violates constraint
                    startTime: new Date(),
                    endTime: new Date(Date.now() + 60 * 60 * 1000),
                    timezone: "UTC",
                },
            },
            validation: {
                requiredFieldMissing: scheduleDbFixtures.invalid.missingRequired,
                invalidDataType: scheduleDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    startTime: new Date("1900-01-01"), // Too far in past
                    endTime: new Date("3000-01-01"), // Too far in future
                    timezone: "UTC",
                },
            },
            businessLogic: {
                endBeforeStart: scheduleDbFixtures.invalid.invalidTimeRange,
                invalidTimezone: scheduleDbFixtures.invalid.invalidTimezone,
                recurringWithoutRecurrence: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    startTime: new Date(),
                    endTime: new Date(Date.now() + 60 * 60 * 1000),
                    timezone: "UTC",
                    // Implies recurring but no recurrence data
                    exceptions: {
                        create: [{
                            id: generatePK(),
                            originalStartTime: new Date(Date.now() + 24 * 60 * 60 * 1000),
                        }],
                    },
                },
            },
        };
    }

    /**
     * Add object association to a schedule fixture
     */
    protected addObjectAssociation(data: Prisma.ScheduleCreateInput, forId: string, forType: string): Prisma.ScheduleCreateInput {
        const connections: Record<string, any> = {
            FocusMode: { focusMode: { connect: { id: forId } } },
            Meeting: { meeting: { connect: { id: forId } } },
            RunProject: { runProject: { connect: { id: forId } } },
            RunRoutine: { runRoutine: { connect: { id: forId } } },
        };

        return {
            ...data,
            ...(connections[forType] || {}),
        };
    }

    /**
     * Add recurrence to a schedule fixture
     */
    protected addRecurrence(data: Prisma.ScheduleCreateInput, recurrence: Partial<Prisma.ScheduleRecurrenceCreateWithoutScheduleInput>): Prisma.ScheduleCreateInput {
        return {
            ...data,
            recurrences: {
                create: {
                    id: generatePK(),
                    recurrenceType: "Daily",
                    interval: 1,
                    dayOfWeek: null,
                    dayOfMonth: null,
                    month: null,
                    endDate: null,
                    ...recurrence,
                },
            },
        };
    }

    /**
     * Add exceptions to a schedule fixture
     */
    protected addExceptions(data: Prisma.ScheduleCreateInput, exceptions: Array<{ date: Date; newStartTime?: Date; newEndTime?: Date }>): Prisma.ScheduleCreateInput {
        return {
            ...data,
            exceptions: {
                create: exceptions.map(ex => ({
                    id: generatePK(),
                    originalStartTime: ex.date,
                    newStartTime: ex.newStartTime,
                    newEndTime: ex.newEndTime,
                })),
            },
        };
    }

    /**
     * Schedule-specific validation
     */
    protected validateSpecific(data: Prisma.ScheduleCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to Schedule
        if (!data.startTime) errors.push("Schedule start time is required");
        if (!data.endTime) errors.push("Schedule end time is required");
        if (!data.timezone) errors.push("Schedule timezone is required");

        // Check business logic
        if (data.startTime && data.endTime && data.startTime >= data.endTime) {
            errors.push("Schedule start time must be before end time");
        }

        if (data.timezone && !data.timezone.includes("/")) {
            warnings.push("Timezone should be in IANA format (e.g., America/New_York)");
        }

        // Check recurrence logic
        if (data.exceptions?.create && !data.recurrences?.create) {
            warnings.push("Schedule has exceptions but no recurrence pattern");
        }

        // Check object associations
        const associations = [data.focusMode, data.meeting, data.runProject, data.runRoutine].filter(Boolean);
        if (associations.length > 1) {
            errors.push("Schedule can only be associated with one object type");
        }

        if (associations.length === 0) {
            warnings.push("Schedule should be associated with an object (meeting, focus mode, etc.)");
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(
        forId: string,
        forType: string,
        overrides?: Partial<Prisma.ScheduleCreateInput>
    ): Prisma.ScheduleCreateInput {
        const factory = new ScheduleDbFactory();
        const data = factory.createMinimal(overrides);
        return factory.addObjectAssociation(data, forId, forType);
    }

    static createRecurring(
        forId: string,
        forType: string,
        recurrence: Partial<Prisma.ScheduleRecurrenceCreateWithoutScheduleInput>,
        overrides?: Partial<Prisma.ScheduleCreateInput>
    ): Prisma.ScheduleCreateInput {
        const factory = new ScheduleDbFactory();
        let data = factory.createMinimal(overrides);
        data = factory.addObjectAssociation(data, forId, forType);
        return factory.addRecurrence(data, recurrence);
    }

    static createWithExceptions(
        forId: string,
        forType: string,
        exceptions: Array<{ date: Date; newStartTime?: Date; newEndTime?: Date }>,
        overrides?: Partial<Prisma.ScheduleCreateInput>
    ): Prisma.ScheduleCreateInput {
        const factory = new ScheduleDbFactory();
        let data = factory.createMinimal(overrides);
        data = factory.addObjectAssociation(data, forId, forType);
        return factory.addExceptions(data, exceptions);
    }

    static createWithLabels(
        forId: string,
        forType: string,
        labelIds: string[],
        overrides?: Partial<Prisma.ScheduleCreateInput>
    ): Prisma.ScheduleCreateInput {
        const factory = new ScheduleDbFactory();
        let data = factory.createMinimal({
            labels: {
                connect: labelIds.map(id => ({ id })),
            },
            ...overrides,
        });
        return factory.addObjectAssociation(data, forId, forType);
    }
}

/**
 * Enhanced helper to seed multiple test schedules with comprehensive options
 */
export async function seedSchedules(
    prisma: any,
    options: {
        forObjects: Array<{ id: string; type: string }>;
        withRecurrence?: boolean;
        withExceptions?: boolean;
        withLabels?: string[];
    }
): Promise<BulkSeedResult<any>> {
    const factory = new ScheduleDbFactory();
    const schedules = [];
    let recurringCount = 0;
    let exceptionCount = 0;
    let labelCount = 0;

    for (const obj of options.forObjects) {
        let scheduleData: Prisma.ScheduleCreateInput;

        if (options.withRecurrence) {
            scheduleData = ScheduleDbFactory.createRecurring(
                obj.id,
                obj.type,
                {
                    recurrenceType: "Weekly",
                    interval: 1,
                    dayOfWeek: 1, // Monday
                },
                {
                    ...(options.withLabels && {
                        labels: {
                            connect: options.withLabels.map(id => ({ id })),
                        },
                    }),
                }
            );
            recurringCount++;
        } else if (options.withExceptions) {
            const tomorrow = new Date(Date.now() + 24 * 60 * 60 * 1000);
            const nextWeek = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000);
            
            scheduleData = ScheduleDbFactory.createWithExceptions(
                obj.id,
                obj.type,
                [
                    { date: tomorrow, newStartTime: new Date(tomorrow.getTime() + 2 * 60 * 60 * 1000) },
                    { date: nextWeek }, // Cancelled
                ],
                {
                    ...(options.withLabels && {
                        labels: {
                            connect: options.withLabels.map(id => ({ id })),
                        },
                    }),
                }
            );
            exceptionCount += 2; // Two exceptions created
        } else {
            scheduleData = options.withLabels
                ? ScheduleDbFactory.createWithLabels(obj.id, obj.type, options.withLabels)
                : ScheduleDbFactory.createMinimal(obj.id, obj.type);
        }

        if (options.withLabels) {
            labelCount += options.withLabels.length;
        }

        const schedule = await prisma.schedule.create({
            data: scheduleData,
            include: { 
                recurrences: true, 
                exceptions: true,
                labels: true,
                meeting: true,
                focusMode: true,
                runProject: true,
                runRoutine: true,
            },
        });
        schedules.push(schedule);
    }

    return {
        records: schedules,
        summary: {
            total: schedules.length,
            recurring: recurringCount,
            exceptions: exceptionCount,
            labels: labelCount,
        },
    };
}