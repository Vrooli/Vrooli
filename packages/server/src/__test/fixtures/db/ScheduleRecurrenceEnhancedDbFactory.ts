/* eslint-disable no-magic-numbers */
// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: replaced any with PrismaClient type
import { type Prisma, type PrismaClient, ScheduleRecurrenceType, type schedule_recurrence } from "@prisma/client";
import { } from "@vrooli/shared";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type {
    DbTestFixtures,
    RelationConfig,
    TestScenario,
} from "./types.js";

interface ScheduleRecurrenceRelationConfig extends RelationConfig {
    withSchedule?: { scheduleId: bigint };
}

/**
 * Enhanced database fixture factory for ScheduleRecurrence model
 * Provides comprehensive testing capabilities for schedule recurrence patterns
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for all recurrence types
 * - Complex pattern generation
 * - RRULE parsing support
 * - Schedule association
 * - Predefined test scenarios
 * - Business rule validation
 */
export class ScheduleRecurrenceEnhancedDbFactory extends EnhancedDatabaseFactory<
    schedule_recurrence,
    Prisma.schedule_recurrenceCreateInput,
    Prisma.schedule_recurrenceInclude,
    Prisma.schedule_recurrenceUpdateInput
> {
    // Store commonly used schedule IDs for fixtures
    private fixtureScheduleId: bigint;
    private defaultScheduleId: bigint;
    protected scenarios: Record<string, TestScenario> = {};

    constructor(prisma: PrismaClient) {
        super("ScheduleRecurrence", prisma);
        // Generate valid schedule IDs that can be reused in fixtures
        this.fixtureScheduleId = this.generateId();
        this.defaultScheduleId = this.generateId();
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.schedule_recurrence;
    }

    /**
     * Get complete test fixtures for ScheduleRecurrence model
     */
    protected getFixtures(): DbTestFixtures<Prisma.schedule_recurrenceCreateInput, Prisma.schedule_recurrenceUpdateInput> {
        const oneYear = new Date(Date.now() + 365 * 24 * 60 * 60 * 1000);
        const twoYears = new Date(Date.now() + 2 * 365 * 24 * 60 * 60 * 1000);

        return {
            minimal: {
                id: this.generateId(),
                recurrenceType: ScheduleRecurrenceType.Daily,
                interval: 1,
                schedule: {
                    connect: { id: this.fixtureScheduleId },
                },
            },
            complete: {
                id: this.generateId(),
                recurrenceType: ScheduleRecurrenceType.Monthly,
                interval: 1,
                dayOfMonth: 15,
                endDate: oneYear,
                duration: 90, // 90 minutes
                schedule: {
                    connect: { id: this.fixtureScheduleId },
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, recurrenceType, interval, schedule
                    duration: 60,
                },
                invalidTypes: {
                    id: BigInt("12345"), // Invalid ID value but properly typed as BigInt
                    recurrenceType: "InvalidType", // Should be enum value
                    interval: "1", // Should be number
                    dayOfWeek: "Monday", // Should be number
                    dayOfMonth: "15th", // Should be number
                    month: "December", // Should be number
                    endDate: "next year", // Should be Date
                    duration: "one hour", // Should be number
                    scheduleId: BigInt("987654321"), // Invalid schedule ID but properly typed
                },
                invalidRanges: {
                    id: this.generateId(),
                    recurrenceType: ScheduleRecurrenceType.Weekly,
                    interval: 0, // Should be at least 1
                    dayOfWeek: 8, // Should be 0-6
                    schedule: {
                        connect: { id: this.fixtureScheduleId },
                    },
                },
                invalidMonthlyConfig: {
                    id: this.generateId(),
                    recurrenceType: ScheduleRecurrenceType.Monthly,
                    interval: 1,
                    dayOfMonth: 32, // Invalid day of month
                    schedule: {
                        connect: { id: this.fixtureScheduleId },
                    },
                },
                invalidYearlyConfig: {
                    id: this.generateId(),
                    recurrenceType: ScheduleRecurrenceType.Yearly,
                    interval: 1,
                    month: 13, // Invalid month
                    dayOfMonth: 15,
                    schedule: {
                        connect: { id: this.fixtureScheduleId },
                    },
                },
                negativeInterval: {
                    id: this.generateId(),
                    recurrenceType: ScheduleRecurrenceType.Daily,
                    interval: -1, // Should be positive
                    schedule: {
                        connect: { id: this.fixtureScheduleId },
                    },
                },
                negativeDuration: {
                    id: this.generateId(),
                    recurrenceType: ScheduleRecurrenceType.Daily,
                    interval: 1,
                    duration: -60, // Should be positive
                    schedule: {
                        connect: { id: this.fixtureScheduleId },
                    },
                },
            },
            edgeCases: {
                dailyForever: {
                    id: this.generateId(),
                    recurrenceType: ScheduleRecurrenceType.Daily,
                    interval: 1,
                    endDate: null, // No end date
                    schedule: {
                        connect: { id: this.fixtureScheduleId },
                    },
                },
                weeklyMondayToFriday: {
                    id: this.generateId(),
                    recurrenceType: ScheduleRecurrenceType.Weekly,
                    interval: 1,
                    dayOfWeek: 1, // Monday (need multiple for M-F)
                    duration: 60,
                    schedule: {
                        connect: { id: this.fixtureScheduleId },
                    },
                },
                biWeeklyWednesday: {
                    id: this.generateId(),
                    recurrenceType: ScheduleRecurrenceType.Weekly,
                    interval: 2,
                    dayOfWeek: 3, // Wednesday
                    duration: 120,
                    endDate: oneYear,
                    schedule: {
                        connect: { id: this.fixtureScheduleId },
                    },
                },
                monthlyFirstMonday: {
                    id: this.generateId(),
                    recurrenceType: ScheduleRecurrenceType.Monthly,
                    interval: 1,
                    dayOfWeek: 1, // Monday (first of month logic handled elsewhere)
                    duration: 180,
                    schedule: {
                        connect: { id: this.fixtureScheduleId },
                    },
                },
                monthlyLastDay: {
                    id: this.generateId(),
                    recurrenceType: ScheduleRecurrenceType.Monthly,
                    interval: 1,
                    dayOfMonth: 31, // Will adjust to last day of month
                    duration: 60,
                    schedule: {
                        connect: { id: this.fixtureScheduleId },
                    },
                },
                quarterly: {
                    id: this.generateId(),
                    recurrenceType: ScheduleRecurrenceType.Monthly,
                    interval: 3, // Every 3 months
                    dayOfMonth: 15,
                    duration: 240, // 4 hours
                    endDate: twoYears,
                    schedule: {
                        connect: { id: this.fixtureScheduleId },
                    },
                },
                yearlyBirthday: {
                    id: this.generateId(),
                    recurrenceType: ScheduleRecurrenceType.Yearly,
                    interval: 1,
                    month: 6, // June
                    dayOfMonth: 15,
                    duration: 1440, // All day (24 hours)
                    schedule: {
                        connect: { id: this.fixtureScheduleId },
                    },
                },
                leapYearEvent: {
                    id: this.generateId(),
                    recurrenceType: ScheduleRecurrenceType.Yearly,
                    interval: 1,
                    month: 2, // February
                    dayOfMonth: 29, // Only occurs on leap years
                    duration: 60,
                    schedule: {
                        connect: { id: this.fixtureScheduleId },
                    },
                },
                highFrequencyDaily: {
                    id: this.generateId(),
                    recurrenceType: ScheduleRecurrenceType.Daily,
                    interval: 1,
                    duration: 30, // 30 minutes
                    endDate: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000), // One week
                    schedule: {
                        connect: { id: this.fixtureScheduleId },
                    },
                },
                longTermYearly: {
                    id: this.generateId(),
                    recurrenceType: ScheduleRecurrenceType.Yearly,
                    interval: 1,
                    month: 12,
                    dayOfMonth: 1,
                    duration: 480, // 8 hours
                    endDate: new Date("2050-12-31T23:59:59Z"),
                    schedule: {
                        connect: { id: this.fixtureScheduleId },
                    },
                },
                extremeInterval: {
                    id: this.generateId(),
                    recurrenceType: ScheduleRecurrenceType.Monthly,
                    interval: 12, // Every year via monthly
                    dayOfMonth: 1,
                    duration: 60,
                    schedule: {
                        connect: { id: this.fixtureScheduleId },
                    },
                },
            },
            updates: {
                minimal: {
                    interval: 2,
                },
                complete: {
                    recurrenceType: ScheduleRecurrenceType.Weekly,
                    interval: 1,
                    dayOfWeek: 5, // Friday
                    duration: 120,
                    endDate: twoYears,
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.schedule_recurrenceCreateInput>): Prisma.schedule_recurrenceCreateInput {
        return {
            id: this.generateId(),
            recurrenceType: ScheduleRecurrenceType.Daily,
            interval: 1,
            schedule: {
                connect: { id: this.defaultScheduleId },
            },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.schedule_recurrenceCreateInput>): Prisma.schedule_recurrenceCreateInput {
        return {
            id: this.generateId(),
            recurrenceType: ScheduleRecurrenceType.Weekly,
            interval: 1,
            dayOfWeek: 1, // Monday
            duration: 60,
            endDate: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000),
            schedule: {
                connect: { id: this.defaultScheduleId },
            },
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            dailyStandup: {
                name: "dailyStandup",
                description: "Daily standup meeting pattern",
                config: {
                    overrides: {
                        recurrenceType: ScheduleRecurrenceType.Daily,
                        interval: 1,
                        duration: 15,
                        endDate: new Date(Date.now() + 90 * 24 * 60 * 60 * 1000), // 3 months
                    },
                },
            },
            weeklyTeamMeeting: {
                name: "weeklyTeamMeeting",
                description: "Weekly team meeting on Wednesday",
                config: {
                    overrides: {
                        recurrenceType: ScheduleRecurrenceType.Weekly,
                        interval: 1,
                        dayOfWeek: 3, // Wednesday
                        duration: 60,
                    },
                },
            },
            biWeeklySprintPlanning: {
                name: "biWeeklySprintPlanning",
                description: "Sprint planning every two weeks",
                config: {
                    overrides: {
                        recurrenceType: ScheduleRecurrenceType.Weekly,
                        interval: 2,
                        dayOfWeek: 1, // Monday
                        duration: 120,
                    },
                },
            },
            monthlyAllHands: {
                name: "monthlyAllHands",
                description: "Monthly all-hands meeting",
                config: {
                    overrides: {
                        recurrenceType: ScheduleRecurrenceType.Monthly,
                        interval: 1,
                        dayOfMonth: 1,
                        duration: 90,
                    },
                },
            },
            quarterlyReview: {
                name: "quarterlyReview",
                description: "Quarterly business review",
                config: {
                    overrides: {
                        recurrenceType: ScheduleRecurrenceType.Monthly,
                        interval: 3,
                        dayOfMonth: 15,
                        duration: 180,
                    },
                },
            },
            annualReview: {
                name: "annualReview",
                description: "Annual performance review",
                config: {
                    overrides: {
                        recurrenceType: ScheduleRecurrenceType.Yearly,
                        interval: 1,
                        month: 12,
                        dayOfMonth: 15,
                        duration: 60,
                    },
                },
            },
        };
    }

    /**
     * Create specific recurrence patterns
     */
    async createDaily(scheduleId: bigint | string, interval = 1, duration?: number): Promise<schedule_recurrence> {
        return await this.createMinimal({
            recurrenceType: ScheduleRecurrenceType.Daily,
            interval,
            duration,
            schedule: { connect: { id: typeof scheduleId === "string" ? BigInt(scheduleId) : scheduleId } },
        });
    }

    async createWeekly(
        scheduleId: bigint | string,
        dayOfWeek: number,
        interval = 1,
        duration?: number,
    ): Promise<schedule_recurrence> {
        return await this.createMinimal({
            recurrenceType: ScheduleRecurrenceType.Weekly,
            interval,
            dayOfWeek,
            duration,
            schedule: { connect: { id: typeof scheduleId === "string" ? BigInt(scheduleId) : scheduleId } },
        });
    }

    async createMonthly(
        scheduleId: bigint | string,
        dayOfMonth: number,
        interval = 1,
        duration?: number,
    ): Promise<schedule_recurrence> {
        return await this.createMinimal({
            recurrenceType: ScheduleRecurrenceType.Monthly,
            interval,
            dayOfMonth,
            duration,
            schedule: { connect: { id: typeof scheduleId === "string" ? BigInt(scheduleId) : scheduleId } },
        });
    }

    async createYearly(
        scheduleId: bigint | string,
        month: number,
        dayOfMonth: number,
        interval = 1,
        duration?: number,
    ): Promise<schedule_recurrence> {
        return await this.createMinimal({
            recurrenceType: ScheduleRecurrenceType.Yearly,
            interval,
            month,
            dayOfMonth,
            duration,
            schedule: { connect: { id: typeof scheduleId === "string" ? BigInt(scheduleId) : scheduleId } },
        });
    }

    protected getDefaultInclude(): Prisma.schedule_recurrenceInclude {
        return {
            schedule: {
                select: {
                    id: true,
                    publicId: true,
                    startTime: true,
                    endTime: true,
                    timezone: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.schedule_recurrenceCreateInput,
        config: ScheduleRecurrenceRelationConfig,
        _tx: PrismaClient,
    ): Promise<Prisma.schedule_recurrenceCreateInput> {
        const data = { ...baseData };

        // Handle schedule relationship
        if (config.withSchedule) {
            data.schedule = {
                connect: { id: config.withSchedule.scheduleId },
            };
        }

        return data;
    }

    protected async checkModelConstraints(record: schedule_recurrence): Promise<string[]> {
        const violations: string[] = [];

        // Check interval
        if (record.interval < 1) {
            violations.push("Interval must be at least 1");
        }

        // Check duration
        if (record.duration !== null && record.duration !== undefined && record.duration < 0) {
            violations.push("Duration cannot be negative");
        }

        // Type-specific validation
        switch (record.recurrenceType) {
            case ScheduleRecurrenceType.Weekly:
                if (record.dayOfWeek === null || record.dayOfWeek === undefined) {
                    violations.push("Weekly recurrence requires day of week");
                } else if (record.dayOfWeek < 0 || record.dayOfWeek > 6) {
                    violations.push("Day of week must be between 0 (Sunday) and 6 (Saturday)");
                }
                break;

            case ScheduleRecurrenceType.Monthly:
                if (!record.dayOfMonth && !record.dayOfWeek) {
                    violations.push("Monthly recurrence requires either day of month or day of week");
                }
                if (record.dayOfMonth && (record.dayOfMonth < 1 || record.dayOfMonth > 31)) {
                    violations.push("Day of month must be between 1 and 31");
                }
                break;

            case ScheduleRecurrenceType.Yearly:
                if (!record.month) {
                    violations.push("Yearly recurrence requires month");
                } else if (record.month < 1 || record.month > 12) {
                    violations.push("Month must be between 1 and 12");
                }
                if (!record.dayOfMonth) {
                    violations.push("Yearly recurrence requires day of month");
                } else if (record.dayOfMonth < 1 || record.dayOfMonth > 31) {
                    violations.push("Day of month must be between 1 and 31");
                }
                break;

            case ScheduleRecurrenceType.Daily:
                // Daily recurrence doesn't need additional validation
                break;
        }

        // Check end date
        if (record.endDate && record.endDate < new Date()) {
            violations.push("End date cannot be in the past");
        }

        // Check schedule association
        if (!record.scheduleId) {
            violations.push("Schedule recurrence must belong to a schedule");
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {};
    }

    protected async deleteRelatedRecords(
        _record: schedule_recurrence,
        _remainingDepth: number,
        _tx: PrismaClient,
        _includeOnly?: string[],
    ): Promise<void> {
        // ScheduleRecurrence has no dependent records to delete
    }

    /**
     * Create multiple recurrences for a schedule (e.g., MWF meetings)
     */
    async createMultiDayWeekly(
        scheduleId: bigint | string,
        daysOfWeek: number[],
        interval = 1,
        duration?: number,
    ): Promise<schedule_recurrence[]> {
        const scheduleIdBigint = typeof scheduleId === "string" ? BigInt(scheduleId) : scheduleId;
        const records: schedule_recurrence[] = [];
        for (const day of daysOfWeek) {
            const record = await this.createMinimal({
                recurrenceType: ScheduleRecurrenceType.Weekly,
                interval,
                dayOfWeek: day,
                duration,
                schedule: { connect: { id: scheduleIdBigint } },
            });
            records.push(record);
        }
        return records;
    }

    /**
     * Create recurrence pattern from RRULE string
     */
    async createFromRRule(
        scheduleId: bigint | string,
        rrule: string,
        overrides?: Partial<Prisma.schedule_recurrenceCreateInput>,
    ): Promise<schedule_recurrence> {
        const parts = rrule.split(";").reduce((acc, part) => {
            const [key, value] = part.split("=");
            acc[key] = value;
            return acc;
        }, {} as Record<string, string>);

        const baseData: Partial<Prisma.schedule_recurrenceCreateInput> = {
            interval: parseInt(parts.INTERVAL || "1"),
        };

        // Map frequency
        switch (parts.FREQ) {
            case "DAILY":
                baseData.recurrenceType = ScheduleRecurrenceType.Daily;
                break;
            case "WEEKLY":
                baseData.recurrenceType = ScheduleRecurrenceType.Weekly;
                if (parts.BYDAY) {
                    const dayMap: Record<string, number> = {
                        SU: 0, MO: 1, TU: 2, WE: 3, TH: 4, FR: 5, SA: 6,
                    };
                    const days = parts.BYDAY.split(",");
                    baseData.dayOfWeek = dayMap[days[0]] || 0;
                }
                break;
            case "MONTHLY":
                baseData.recurrenceType = ScheduleRecurrenceType.Monthly;
                if (parts.BYMONTHDAY) {
                    baseData.dayOfMonth = parseInt(parts.BYMONTHDAY);
                }
                break;
            case "YEARLY":
                baseData.recurrenceType = ScheduleRecurrenceType.Yearly;
                if (parts.BYMONTH) {
                    baseData.month = parseInt(parts.BYMONTH);
                }
                if (parts.BYMONTHDAY) {
                    baseData.dayOfMonth = parseInt(parts.BYMONTHDAY);
                }
                break;
            default:
                baseData.recurrenceType = ScheduleRecurrenceType.Daily;
        }

        // Handle end conditions
        if (parts.UNTIL) {
            const year = parts.UNTIL.substring(0, 4);
            const month = parts.UNTIL.substring(4, 6);
            const day = parts.UNTIL.substring(6, 8);
            baseData.endDate = new Date(`${year}-${month}-${day}`);
        }

        return await this.createMinimal({
            ...baseData,
            schedule: { connect: { id: typeof scheduleId === "string" ? BigInt(scheduleId) : scheduleId } },
            ...overrides,
        });
    }

    /**
     * Create test recurrences for various scenarios
     */
    async createTestRecurrences(scheduleId: bigint | string): Promise<{
        daily: schedule_recurrence;
        weekly: schedule_recurrence;
        monthly: schedule_recurrence;
        yearly: schedule_recurrence;
        biweekly: schedule_recurrence;
        quarterly: schedule_recurrence;
    }> {
        const scheduleIdBigint = typeof scheduleId === "string" ? BigInt(scheduleId) : scheduleId;
        const [daily, weekly, monthly, yearly, biweekly, quarterly] = await Promise.all([
            this.createDaily(scheduleIdBigint, 1, 30),
            this.createWeekly(scheduleIdBigint, 1, 1, 60), // Monday
            this.createMonthly(scheduleIdBigint, 15, 1, 90),
            this.createYearly(scheduleIdBigint, 12, 25, 1, 120), // Christmas
            this.createWeekly(scheduleIdBigint, 5, 2, 90), // Every other Friday
            this.createMonthly(scheduleIdBigint, 15, 3, 180), // Quarterly (every 3 months)
        ]);

        return {
            daily,
            weekly,
            monthly,
            yearly,
            biweekly,
            quarterly,
        };
    }

    /**
     * Create business meeting patterns
     */
    async createBusinessMeetingPatterns(scheduleId: bigint | string): Promise<{
        standup: schedule_recurrence;
        teamMeeting: schedule_recurrence;
        sprintPlanning: schedule_recurrence;
        allHands: schedule_recurrence;
        review: schedule_recurrence;
    }> {
        const scheduleIdBigint = typeof scheduleId === "string" ? BigInt(scheduleId) : scheduleId;
        const [standup, teamMeeting, sprintPlanning, allHands, review] = await Promise.all([
            this.createDaily(scheduleIdBigint, 1, 15), // Daily standup for 15 minutes
            this.createWeekly(scheduleIdBigint, 3, 1, 60), // Weekly team meeting on Wednesday
            this.createWeekly(scheduleIdBigint, 1, 2, 120), // Bi-weekly sprint planning on Monday
            this.createMonthly(scheduleIdBigint, 1, 1, 90), // Monthly all-hands on the 1st
            this.createYearly(scheduleIdBigint, 12, 15, 60), // Annual review
        ]);

        return {
            standup,
            teamMeeting,
            sprintPlanning,
            allHands,
            review,
        };
    }

    /**
     * Seed a specific test scenario
     */
    async seedScenario(scenarioName: string): Promise<schedule_recurrence> {
        const scenario = this.scenarios[scenarioName];
        if (!scenario) {
            throw new Error(`Scenario ${scenarioName} not found`);
        }
        return await this.createMinimal(scenario.config.overrides);
    }
}

// Export factory creator function
export const createScheduleRecurrenceEnhancedDbFactory = (prisma: PrismaClient) =>
    new ScheduleRecurrenceEnhancedDbFactory(prisma);

// Export the class for type usage
export { ScheduleRecurrenceEnhancedDbFactory as ScheduleRecurrenceEnhancedDbFactoryClass };
