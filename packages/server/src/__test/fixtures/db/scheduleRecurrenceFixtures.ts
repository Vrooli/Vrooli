import { generatePK } from "@vrooli/shared";
import { type Prisma, ScheduleRecurrenceType } from "@prisma/client";

/**
 * Database fixtures for ScheduleRecurrence model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const scheduleRecurrenceDbIds = {
    recurrence1: generatePK(),
    recurrence2: generatePK(),
    recurrence3: generatePK(),
    recurrence4: generatePK(),
    recurrence5: generatePK(),
    schedule1: generatePK(),
    schedule2: generatePK(),
    schedule3: generatePK(),
};

/**
 * Minimal schedule recurrence data for database creation
 */
export const minimalScheduleRecurrenceDb: Prisma.schedule_recurrenceCreateInput = {
    id: scheduleRecurrenceDbIds.recurrence1,
    recurrenceType: ScheduleRecurrenceType.Daily,
    interval: 1,
    schedule: { connect: { id: scheduleRecurrenceDbIds.schedule1 } },
};

/**
 * Weekly schedule recurrence
 */
export const weeklyScheduleRecurrenceDb: Prisma.schedule_recurrenceCreateInput = {
    id: scheduleRecurrenceDbIds.recurrence2,
    recurrenceType: ScheduleRecurrenceType.Weekly,
    interval: 1,
    dayOfWeek: 3, // Wednesday (1-7, where 1 is Monday)
    duration: 60, // 60 minutes
    schedule: { connect: { id: scheduleRecurrenceDbIds.schedule1 } },
};

/**
 * Monthly schedule recurrence
 */
export const monthlyScheduleRecurrenceDb: Prisma.schedule_recurrenceCreateInput = {
    id: scheduleRecurrenceDbIds.recurrence3,
    recurrenceType: ScheduleRecurrenceType.Monthly,
    interval: 1,
    dayOfMonth: 15, // 15th of each month
    duration: 90, // 90 minutes
    endDate: new Date("2025-12-31T23:59:59Z"),
    schedule: { connect: { id: scheduleRecurrenceDbIds.schedule2 } },
};

/**
 * Complete schedule recurrence with all optional fields
 */
export const completeScheduleRecurrenceDb: Prisma.schedule_recurrenceCreateInput = {
    id: scheduleRecurrenceDbIds.recurrence4,
    recurrenceType: ScheduleRecurrenceType.Yearly,
    interval: 1,
    month: 6, // June
    dayOfMonth: 15, // June 15th
    duration: 120, // 2 hours
    endDate: new Date("2030-06-15T23:59:59Z"),
    schedule: { connect: { id: scheduleRecurrenceDbIds.schedule3 } },
};

/**
 * Factory for creating schedule recurrence database fixtures with overrides
 */
export class ScheduleRecurrenceDbFactory {
    static createMinimal(
        scheduleId: string | bigint,
        overrides?: Partial<Prisma.schedule_recurrenceCreateInput>,
    ): Prisma.schedule_recurrenceCreateInput {
        return {
            id: generatePK(),
            recurrenceType: ScheduleRecurrenceType.Daily,
            interval: 1,
            schedule: { connect: { id: typeof scheduleId === "string" ? BigInt(scheduleId) : scheduleId } },
            ...overrides,
        };
    }

    static createDaily(
        scheduleId: string | bigint,
        interval = 1,
        overrides?: Partial<Prisma.schedule_recurrenceCreateInput>,
    ): Prisma.schedule_recurrenceCreateInput {
        return {
            ...this.createMinimal(scheduleId, overrides),
            recurrenceType: ScheduleRecurrenceType.Daily,
            interval,
        };
    }

    static createWeekly(
        scheduleId: string | bigint,
        dayOfWeek: number,
        interval = 1,
        overrides?: Partial<Prisma.schedule_recurrenceCreateInput>,
    ): Prisma.schedule_recurrenceCreateInput {
        return {
            ...this.createMinimal(scheduleId, overrides),
            recurrenceType: ScheduleRecurrenceType.Weekly,
            interval,
            dayOfWeek,
        };
    }

    static createMonthly(
        scheduleId: string | bigint,
        dayOfMonth: number,
        interval = 1,
        overrides?: Partial<Prisma.schedule_recurrenceCreateInput>,
    ): Prisma.schedule_recurrenceCreateInput {
        return {
            ...this.createMinimal(scheduleId, overrides),
            recurrenceType: ScheduleRecurrenceType.Monthly,
            interval,
            dayOfMonth,
        };
    }

    static createYearly(
        scheduleId: string | bigint,
        month: number,
        dayOfMonth: number,
        interval = 1,
        overrides?: Partial<Prisma.schedule_recurrenceCreateInput>,
    ): Prisma.schedule_recurrenceCreateInput {
        return {
            ...this.createMinimal(scheduleId, overrides),
            recurrenceType: ScheduleRecurrenceType.Yearly,
            interval,
            month,
            dayOfMonth,
        };
    }

    static createWithEndDate(
        scheduleId: string | bigint,
        endDate: Date,
        overrides?: Partial<Prisma.schedule_recurrenceCreateInput>,
    ): Prisma.schedule_recurrenceCreateInput {
        return {
            ...this.createMinimal(scheduleId, overrides),
            endDate,
        };
    }

    static createWithDuration(
        scheduleId: string | bigint,
        duration: number,
        overrides?: Partial<Prisma.schedule_recurrenceCreateInput>,
    ): Prisma.schedule_recurrenceCreateInput {
        return {
            ...this.createMinimal(scheduleId, overrides),
            duration,
        };
    }
}

/**
 * Helper to seed schedule recurrences for testing
 */
export async function seedScheduleRecurrences(
    prisma: any,
    options: {
        scheduleId: string | bigint;
        recurrences: Array<{
            type: "Daily" | "Weekly" | "Monthly" | "Yearly";
            interval?: number;
            dayOfWeek?: number;
            dayOfMonth?: number;
            month?: number;
            duration?: number;
            endDate?: Date;
        }>;
    },
) {
    const recurrences = [];

    for (const recurrenceConfig of options.recurrences) {
        let recurrenceData: Prisma.schedule_recurrenceCreateInput;

        switch (recurrenceConfig.type) {
            case "Daily":
                recurrenceData = ScheduleRecurrenceDbFactory.createDaily(
                    options.scheduleId,
                    recurrenceConfig.interval,
                    {
                        duration: recurrenceConfig.duration,
                        endDate: recurrenceConfig.endDate,
                    },
                );
                break;
            case "Weekly":
                recurrenceData = ScheduleRecurrenceDbFactory.createWeekly(
                    options.scheduleId,
                    recurrenceConfig.dayOfWeek || 1,
                    recurrenceConfig.interval,
                    {
                        duration: recurrenceConfig.duration,
                        endDate: recurrenceConfig.endDate,
                    },
                );
                break;
            case "Monthly":
                recurrenceData = ScheduleRecurrenceDbFactory.createMonthly(
                    options.scheduleId,
                    recurrenceConfig.dayOfMonth || 1,
                    recurrenceConfig.interval,
                    {
                        duration: recurrenceConfig.duration,
                        endDate: recurrenceConfig.endDate,
                    },
                );
                break;
            case "Yearly":
                recurrenceData = ScheduleRecurrenceDbFactory.createYearly(
                    options.scheduleId,
                    recurrenceConfig.month || 1,
                    recurrenceConfig.dayOfMonth || 1,
                    recurrenceConfig.interval,
                    {
                        duration: recurrenceConfig.duration,
                        endDate: recurrenceConfig.endDate,
                    },
                );
                break;
        }

        const recurrence = await prisma.schedule_recurrence.create({
            data: recurrenceData,
            include: {
                schedule: true,
            },
        });
        recurrences.push(recurrence);
    }

    return recurrences;
}

/**
 * Helper to create common recurrence patterns
 */
export const scheduleRecurrencePatterns = {
    /**
     * Daily at 9 AM for 1 hour, ends in 6 months
     */
    dailyMorningMeeting: (scheduleId: string | bigint) =>
        ScheduleRecurrenceDbFactory.createDaily(scheduleId, 1, {
            duration: 60,
            endDate: new Date(Date.now() + 6 * 30 * 24 * 60 * 60 * 1000), // ~6 months
        }),

    /**
     * Weekly on Fridays for 2 hours (team retrospective)
     */
    weeklyRetrospective: (scheduleId: string | bigint) =>
        ScheduleRecurrenceDbFactory.createWeekly(scheduleId, 5, 1, {
            duration: 120,
        }),

    /**
     * Bi-weekly on Wednesdays for 1 hour (sprint planning)
     */
    biWeeklyPlanning: (scheduleId: string | bigint) =>
        ScheduleRecurrenceDbFactory.createWeekly(scheduleId, 3, 2, {
            duration: 60,
        }),

    /**
     * Monthly on the 1st for 30 minutes (monthly review)
     */
    monthlyReview: (scheduleId: string | bigint) =>
        ScheduleRecurrenceDbFactory.createMonthly(scheduleId, 1, 1, {
            duration: 30,
        }),

    /**
     * Quarterly on the 15th for 3 hours (quarterly planning)
     */
    quarterlyPlanning: (scheduleId: string | bigint) =>
        ScheduleRecurrenceDbFactory.createMonthly(scheduleId, 15, 3, {
            duration: 180,
        }),

    /**
     * Annual on June 15th for all day (company retreat)
     */
    annualRetreat: (scheduleId: string | bigint) =>
        ScheduleRecurrenceDbFactory.createYearly(scheduleId, 6, 15, 1, {
            duration: 480, // 8 hours
            endDate: new Date("2030-12-31T23:59:59Z"),
        }),
};
