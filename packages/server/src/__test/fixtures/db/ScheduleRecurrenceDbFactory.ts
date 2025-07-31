/* eslint-disable no-magic-numbers */
import { type Prisma } from "@prisma/client";
import { generatePK } from "@vrooli/shared";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { DbErrorScenarios, DbTestFixtures } from "./types.js";

/**
 * Enhanced test fixtures for ScheduleRecurrence model following standard structure
 */
export const scheduleRecurrenceDbFixtures: DbTestFixtures<
    Prisma.schedule_recurrenceCreateInput,
    Prisma.schedule_recurrenceUpdateInput
> = {
    minimal: {
        id: generatePK(),
        recurrenceType: "Daily",
        interval: 1,
        schedule: { connect: { id: "schedule_placeholder_id" } },
    },
    complete: {
        id: generatePK(),
        recurrenceType: "Monthly",
        interval: 1,
        dayOfWeek: null,
        dayOfMonth: 15,
        month: null,
        endDate: new Date("2026-12-31T23:59:59Z"),
        duration: 90, // 90 minutes
        schedule: { connect: { id: "schedule_placeholder_id" } },
    },
    invalid: {
        missingRequired: {
            // Missing required recurrenceType and schedule
            id: generatePK(),
            interval: 1,
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            recurrenceType: "InvalidType", // Not in enum
            interval: "not-a-number", // Should be number
            dayOfWeek: 8, // Out of range (1-7)
            dayOfMonth: "fifteen", // Should be number
            month: 13, // Out of range (1-12)
            endDate: "not-a-date", // Should be Date
            duration: -60, // Negative duration
            schedule: "not-an-object", // Should be object
        },
        invalidRanges: {
            id: generatePK(),
            recurrenceType: "Weekly",
            interval: 0, // Should be at least 1
            dayOfWeek: 8, // Should be 1-7
            schedule: { connect: { id: "schedule_placeholder_id" } },
        },
    },
    edgeCases: {
        dailyForever: {
            id: generatePK(),
            recurrenceType: "Daily",
            interval: 1,
            endDate: null, // No end date
            schedule: { connect: { id: "schedule_placeholder_id" } },
        },
        weeklyMWF: {
            id: generatePK(),
            recurrenceType: "Weekly",
            interval: 1,
            dayOfWeek: 1, // Monday (would need multiple recurrences for MWF)
            duration: 60,
            schedule: { connect: { id: "schedule_placeholder_id" } },
        },
        biWeekly: {
            id: generatePK(),
            recurrenceType: "Weekly",
            interval: 2,
            dayOfWeek: 3, // Wednesday
            duration: 120,
            schedule: { connect: { id: "schedule_placeholder_id" } },
        },
        monthlyFirstMonday: {
            id: generatePK(),
            recurrenceType: "Monthly",
            interval: 1,
            dayOfWeek: 1, // Monday (first of month logic would be handled elsewhere)
            dayOfMonth: null,
            duration: 180,
            schedule: { connect: { id: "schedule_placeholder_id" } },
        },
        monthlyLastDay: {
            id: generatePK(),
            recurrenceType: "Monthly",
            interval: 1,
            dayOfMonth: 31, // Will adjust to last day of month
            duration: 60,
            schedule: { connect: { id: "schedule_placeholder_id" } },
        },
        quarterly: {
            id: generatePK(),
            recurrenceType: "Monthly",
            interval: 3,
            dayOfMonth: 15,
            duration: 240, // 4 hours
            schedule: { connect: { id: "schedule_placeholder_id" } },
        },
        yearlyBirthday: {
            id: generatePK(),
            recurrenceType: "Yearly",
            interval: 1,
            month: 6, // June
            dayOfMonth: 15,
            duration: 1440, // All day (24 hours)
            schedule: { connect: { id: "schedule_placeholder_id" } },
        },
        leapYearEvent: {
            id: generatePK(),
            recurrenceType: "Yearly",
            interval: 1,
            month: 2, // February
            dayOfMonth: 29, // Only occurs on leap years
            duration: 60,
            schedule: { connect: { id: "schedule_placeholder_id" } },
        },
        everyThreeHours: {
            id: generatePK(),
            recurrenceType: "Daily",
            interval: 1, // Daily recurrence, but duration implies multiple per day
            duration: 180, // 3 hours
            endDate: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000), // One week
            schedule: { connect: { id: "schedule_placeholder_id" } },
        },
        longRunningEvent: {
            id: generatePK(),
            recurrenceType: "Yearly",
            interval: 1,
            month: 12,
            dayOfMonth: 1,
            duration: 43200, // 30 days in minutes
            endDate: new Date("2050-12-31T23:59:59Z"),
            schedule: { connect: { id: "schedule_placeholder_id" } },
        },
    },
};

/**
 * Enhanced factory for creating schedule recurrence database fixtures
 */
export class ScheduleRecurrenceDbFactory extends EnhancedDbFactory<
    Prisma.schedule_recurrenceCreateInput,
    Prisma.schedule_recurrenceUpdateInput
> {

    /**
     * Get the test fixtures for ScheduleRecurrence model
     */
    protected getFixtures(): DbTestFixtures<
        Prisma.schedule_recurrenceCreateInput,
        Prisma.schedule_recurrenceUpdateInput
    > {
        return scheduleRecurrenceDbFixtures;
    }

    /**
     * Get ScheduleRecurrence-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: this.generateId(),
                    recurrenceType: "Daily",
                    interval: 1,
                    schedule: { connect: { id: "schedule_placeholder_id" } },
                },
                foreignKeyViolation: {
                    id: this.generateId(),
                    recurrenceType: "Daily",
                    interval: 1,
                    schedule: { connect: { id: "non-existent-schedule-id" } },
                },
                checkConstraintViolation: {
                    id: this.generateId(),
                    recurrenceType: "Weekly",
                    interval: 1,
                    dayOfWeek: 0, // Invalid day (should be 1-7)
                    schedule: { connect: { id: "schedule_placeholder_id" } },
                },
            },
            validation: {
                requiredFieldMissing: scheduleRecurrenceDbFixtures.invalid.missingRequired,
                invalidDataType: scheduleRecurrenceDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: this.generateId(),
                    recurrenceType: "Monthly",
                    interval: 1,
                    dayOfMonth: 32, // Invalid day of month
                    schedule: { connect: { id: "schedule_placeholder_id" } },
                },
            },
            businessLogic: {
                invalidInterval: {
                    id: this.generateId(),
                    recurrenceType: "Daily",
                    interval: 0, // Should be at least 1
                    schedule: { connect: { id: "schedule_placeholder_id" } },
                },
                weeklyWithoutDayOfWeek: {
                    id: this.generateId(),
                    recurrenceType: "Weekly",
                    interval: 1,
                    dayOfWeek: null, // Weekly needs day of week
                    schedule: { connect: { id: "schedule_placeholder_id" } },
                },
                monthlyConflict: {
                    id: this.generateId(),
                    recurrenceType: "Monthly",
                    interval: 1,
                    dayOfWeek: 1, // Both day of week and day of month specified
                    dayOfMonth: 15,
                    schedule: { connect: { id: "schedule_placeholder_id" } },
                },
                yearlyWithoutMonthDay: {
                    id: this.generateId(),
                    recurrenceType: "Yearly",
                    interval: 1,
                    month: null, // Yearly needs month
                    dayOfMonth: null, // And day
                    schedule: { connect: { id: "schedule_placeholder_id" } },
                },
            },
        };
    }

    /**
     * Generate fresh identifiers specific to ScheduleRecurrence
     */
    protected generateFreshIdentifiers(): Record<string, any> {
        return {
            id: this.generateId(),
        };
    }

    /**
     * ScheduleRecurrence-specific validation
     */
    protected validateSpecific(data: Prisma.schedule_recurrenceCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields
        if (!data.recurrenceType) errors.push("Recurrence type is required");
        if (!data.schedule) errors.push("Schedule association is required");

        // Check interval
        if (data.interval !== undefined && data.interval < 1) {
            errors.push("Interval must be at least 1");
        }

        // Type-specific validation
        switch (data.recurrenceType) {
            case "Weekly":
                if (!data.dayOfWeek) {
                    errors.push("Weekly recurrence requires day of week");
                } else if (data.dayOfWeek < 1 || data.dayOfWeek > 7) {
                    errors.push("Day of week must be between 1 (Monday) and 7 (Sunday)");
                }
                if (data.dayOfMonth) {
                    warnings.push("Day of month is ignored for weekly recurrence");
                }
                break;

            case "Monthly":
                if (!data.dayOfMonth && !data.dayOfWeek) {
                    errors.push("Monthly recurrence requires either day of month or day of week");
                }
                if (data.dayOfMonth && data.dayOfWeek) {
                    warnings.push("Both day of month and day of week specified - this may cause conflicts");
                }
                if (data.dayOfMonth && (data.dayOfMonth < 1 || data.dayOfMonth > 31)) {
                    errors.push("Day of month must be between 1 and 31");
                }
                break;

            case "Yearly":
                if (!data.month) {
                    errors.push("Yearly recurrence requires month");
                } else if (data.month < 1 || data.month > 12) {
                    errors.push("Month must be between 1 and 12");
                }
                if (!data.dayOfMonth) {
                    errors.push("Yearly recurrence requires day of month");
                }
                break;

            case "Daily":
                if (data.dayOfWeek) {
                    warnings.push("Day of week is ignored for daily recurrence");
                }
                if (data.dayOfMonth) {
                    warnings.push("Day of month is ignored for daily recurrence");
                }
                if (data.month) {
                    warnings.push("Month is ignored for daily recurrence");
                }
                break;
        }

        // Check duration
        if (data.duration !== undefined && data.duration < 0) {
            errors.push("Duration cannot be negative");
        }

        // Check end date
        if (data.endDate && data.endDate < new Date()) {
            warnings.push("End date is in the past");
        }

        return { errors, warnings };
    }

    /**
     * Add schedule association to recurrence
     */
    protected addScheduleAssociation(data: Prisma.schedule_recurrenceCreateInput, scheduleId: string): Prisma.schedule_recurrenceCreateInput {
        return {
            ...data,
            schedule: { connect: { id: scheduleId } },
        };
    }

    // Static factory methods for convenience

    static createMinimal(
        scheduleId: string,
        overrides?: Partial<Prisma.schedule_recurrenceCreateInput>,
    ): Prisma.schedule_recurrenceCreateInput {
        const factory = new ScheduleRecurrenceDbFactory();
        const data = factory.createMinimal(overrides);
        return factory.addScheduleAssociation(data, scheduleId);
    }

    static createDaily(
        scheduleId: string,
        interval = 1,
        overrides?: Partial<Prisma.schedule_recurrenceCreateInput>,
    ): Prisma.schedule_recurrenceCreateInput {
        const factory = new ScheduleRecurrenceDbFactory();
        return factory.createMinimal({
            recurrenceType: "Daily",
            interval,
            schedule: { connect: { id: scheduleId } },
            ...overrides,
        });
    }

    static createWeekly(
        scheduleId: string,
        dayOfWeek: number,
        interval = 1,
        overrides?: Partial<Prisma.schedule_recurrenceCreateInput>,
    ): Prisma.schedule_recurrenceCreateInput {
        const factory = new ScheduleRecurrenceDbFactory();
        return factory.createMinimal({
            recurrenceType: "Weekly",
            interval,
            dayOfWeek,
            schedule: { connect: { id: scheduleId } },
            ...overrides,
        });
    }

    static createMonthly(
        scheduleId: string,
        dayOfMonth: number,
        interval = 1,
        overrides?: Partial<Prisma.schedule_recurrenceCreateInput>,
    ): Prisma.schedule_recurrenceCreateInput {
        const factory = new ScheduleRecurrenceDbFactory();
        return factory.createMinimal({
            recurrenceType: "Monthly",
            interval,
            dayOfMonth,
            schedule: { connect: { id: scheduleId } },
            ...overrides,
        });
    }

    static createYearly(
        scheduleId: string,
        month: number,
        dayOfMonth: number,
        interval = 1,
        overrides?: Partial<Prisma.schedule_recurrenceCreateInput>,
    ): Prisma.schedule_recurrenceCreateInput {
        const factory = new ScheduleRecurrenceDbFactory();
        return factory.createMinimal({
            recurrenceType: "Yearly",
            interval,
            month,
            dayOfMonth,
            schedule: { connect: { id: scheduleId } },
            ...overrides,
        });
    }

    /**
     * Create from RRULE string
     */
    static createFromRRule(
        scheduleId: string,
        rrule: string,
        overrides?: Partial<Prisma.schedule_recurrenceCreateInput>,
    ): Prisma.schedule_recurrenceCreateInput {
        const parts = rrule.split(";").reduce((acc, part) => {
            const [key, value] = part.split("=");
            acc[key] = value;
            return acc;
        }, {} as Record<string, string>);

        const factory = new ScheduleRecurrenceDbFactory();
        const baseData: Partial<Prisma.schedule_recurrenceCreateInput> = {
            interval: parseInt(parts.INTERVAL || "1"),
        };

        // Map frequency
        switch (parts.FREQ) {
            case "DAILY":
                baseData.recurrenceType = "Daily";
                break;
            case "WEEKLY":
                baseData.recurrenceType = "Weekly";
                if (parts.BYDAY) {
                    const dayMap: Record<string, number> = { MO: 1, TU: 2, WE: 3, TH: 4, FR: 5, SA: 6, SU: 7 };
                    const days = parts.BYDAY.split(",");
                    baseData.dayOfWeek = dayMap[days[0]] || 1;
                }
                break;
            case "MONTHLY":
                baseData.recurrenceType = "Monthly";
                if (parts.BYMONTHDAY) {
                    baseData.dayOfMonth = parseInt(parts.BYMONTHDAY);
                } else if (parts.BYDAY) {
                    // Handle positional day (e.g., 2TU for 2nd Tuesday)
                    const match = parts.BYDAY.match(/^([+-]?\d+)?([A-Z]{2})$/);
                    if (match) {
                        const dayMap: Record<string, number> = { MO: 1, TU: 2, WE: 3, TH: 4, FR: 5, SA: 6, SU: 7 };
                        baseData.dayOfWeek = dayMap[match[2]] || 1;
                        // Position would need to be handled separately in business logic
                    }
                }
                break;
            case "YEARLY":
                baseData.recurrenceType = "Yearly";
                if (parts.BYMONTH) {
                    baseData.month = parseInt(parts.BYMONTH);
                }
                if (parts.BYMONTHDAY) {
                    baseData.dayOfMonth = parseInt(parts.BYMONTHDAY);
                }
                break;
            default:
                baseData.recurrenceType = "Daily"; // Default
        }

        // Handle end conditions
        if (parts.UNTIL) {
            const year = parts.UNTIL.substring(0, 4);
            const month = parts.UNTIL.substring(4, 6);
            const day = parts.UNTIL.substring(6, 8);
            baseData.endDate = new Date(`${year}-${month}-${day}`);
        }

        return factory.createMinimal({
            ...baseData,
            schedule: { connect: { id: scheduleId } },
            ...overrides,
        });
    }

    /**
     * Create multiple day weekly recurrence (e.g., MWF)
     */
    static createMultiDayWeekly(
        scheduleId: string,
        daysOfWeek: number[],
        overrides?: Partial<Prisma.schedule_recurrenceCreateInput>,
    ): Prisma.schedule_recurrenceCreateInput[] {
        return daysOfWeek.map(day =>
            ScheduleRecurrenceDbFactory.createWeekly(scheduleId, day, 1, overrides),
        );
    }
}

/**
 * Common recurrence patterns for testing
 */
export const scheduleRecurrencePatterns = {
    /**
     * Daily standup meeting
     */
    dailyStandup: (scheduleId: string) =>
        ScheduleRecurrenceDbFactory.createDaily(scheduleId, 1, {
            duration: 15,
            endDate: new Date(Date.now() + 90 * 24 * 60 * 60 * 1000), // 3 months
        }),

    /**
     * Weekly team meeting
     */
    weeklyTeamMeeting: (scheduleId: string) =>
        ScheduleRecurrenceDbFactory.createWeekly(scheduleId, 3, 1, { // Wednesday
            duration: 60,
        }),

    /**
     * Bi-weekly sprint planning
     */
    biWeeklySprintPlanning: (scheduleId: string) =>
        ScheduleRecurrenceDbFactory.createWeekly(scheduleId, 1, 2, { // Every other Monday
            duration: 120,
        }),

    /**
     * Monthly all-hands meeting
     */
    monthlyAllHands: (scheduleId: string) =>
        ScheduleRecurrenceDbFactory.createMonthly(scheduleId, 1, 1, { // First of month
            duration: 90,
        }),

    /**
     * Quarterly business review
     */
    quarterlyReview: (scheduleId: string) =>
        ScheduleRecurrenceDbFactory.createMonthly(scheduleId, 15, 3, { // Every 3 months on 15th
            duration: 180,
        }),

    /**
     * Annual performance review
     */
    annualReview: (scheduleId: string) =>
        ScheduleRecurrenceDbFactory.createYearly(scheduleId, 12, 15, 1, { // December 15th
            duration: 60,
        }),

    /**
     * Complex RRULE patterns
     */
    fromRRule: {
        // Every weekday (Monday-Friday)
        weekdays: (scheduleId: string) =>
            ScheduleRecurrenceDbFactory.createFromRRule(
                scheduleId,
                "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR;INTERVAL=1",
            ),

        // Last Friday of every month
        lastFridayMonthly: (scheduleId: string) =>
            ScheduleRecurrenceDbFactory.createFromRRule(
                scheduleId,
                "FREQ=MONTHLY;BYDAY=-1FR;INTERVAL=1",
            ),

        // Every 3 days for 10 occurrences
        every3DaysLimited: (scheduleId: string) =>
            ScheduleRecurrenceDbFactory.createFromRRule(
                scheduleId,
                "FREQ=DAILY;INTERVAL=3;COUNT=10",
            ),

        // Yearly on the 100th day
        yearlyDay100: (scheduleId: string) =>
            ScheduleRecurrenceDbFactory.createFromRRule(
                scheduleId,
                "FREQ=YEARLY;BYYEARDAY=100",
            ),
    },
};
