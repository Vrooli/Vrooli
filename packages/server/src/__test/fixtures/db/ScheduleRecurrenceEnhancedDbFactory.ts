import { generatePK } from "@vrooli/shared";
import { type Prisma, type PrismaClient, ScheduleRecurrenceType } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface ScheduleRecurrenceRelationConfig extends RelationConfig {
    withSchedule?: { scheduleId: string };
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
    Prisma.schedule_recurrenceCreateInput,
    Prisma.schedule_recurrenceCreateInput,
    Prisma.schedule_recurrenceInclude,
    Prisma.schedule_recurrenceUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('ScheduleRecurrence', prisma);
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
                id: generatePK().toString(),
                recurrenceType: ScheduleRecurrenceType.Daily,
                interval: 1,
                schedule: {
                    connect: { id: "schedule_id" }
                },
            },
            complete: {
                id: generatePK().toString(),
                recurrenceType: ScheduleRecurrenceType.Monthly,
                interval: 1,
                dayOfMonth: 15,
                endDate: oneYear,
                duration: 90, // 90 minutes
                schedule: {
                    connect: { id: "schedule_id" }
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, recurrenceType, interval, schedule
                    duration: 60,
                },
                invalidTypes: {
                    id: "not-a-bigint",
                    recurrenceType: "InvalidType", // Should be enum value
                    interval: "1", // Should be number
                    dayOfWeek: "Monday", // Should be number
                    dayOfMonth: "15th", // Should be number
                    month: "December", // Should be number
                    endDate: "next year", // Should be Date
                    duration: "one hour", // Should be number
                    scheduleId: "string-not-bigint", // Should be BigInt
                },
                invalidRanges: {
                    id: generatePK().toString(),
                    recurrenceType: ScheduleRecurrenceType.Weekly,
                    interval: 0, // Should be at least 1
                    dayOfWeek: 8, // Should be 0-6
                    schedule: {
                        connect: { id: "schedule_id" }
                    },
                },
                invalidMonthlyConfig: {
                    id: generatePK().toString(),
                    recurrenceType: ScheduleRecurrenceType.Monthly,
                    interval: 1,
                    dayOfMonth: 32, // Invalid day of month
                    schedule: {
                        connect: { id: "schedule_id" }
                    },
                },
                invalidYearlyConfig: {
                    id: generatePK().toString(),
                    recurrenceType: ScheduleRecurrenceType.Yearly,
                    interval: 1,
                    month: 13, // Invalid month
                    dayOfMonth: 15,
                    schedule: {
                        connect: { id: "schedule_id" }
                    },
                },
                negativeInterval: {
                    id: generatePK().toString(),
                    recurrenceType: ScheduleRecurrenceType.Daily,
                    interval: -1, // Should be positive
                    schedule: {
                        connect: { id: "schedule_id" }
                    },
                },
                negativeDuration: {
                    id: generatePK().toString(),
                    recurrenceType: ScheduleRecurrenceType.Daily,
                    interval: 1,
                    duration: -60, // Should be positive
                    schedule: {
                        connect: { id: "schedule_id" }
                    },
                },
            },
            edgeCases: {
                dailyForever: {
                    id: generatePK().toString(),
                    recurrenceType: ScheduleRecurrenceType.Daily,
                    interval: 1,
                    endDate: null, // No end date
                    schedule: {
                        connect: { id: "schedule_id" }
                    },
                },
                weeklyMondayToFriday: {
                    id: generatePK().toString(),
                    recurrenceType: ScheduleRecurrenceType.Weekly,
                    interval: 1,
                    dayOfWeek: 1, // Monday (need multiple for M-F)
                    duration: 60,
                    schedule: {
                        connect: { id: "schedule_id" }
                    },
                },
                biWeeklyWednesday: {
                    id: generatePK().toString(),
                    recurrenceType: ScheduleRecurrenceType.Weekly,
                    interval: 2,
                    dayOfWeek: 3, // Wednesday
                    duration: 120,
                    endDate: oneYear,
                    schedule: {
                        connect: { id: "schedule_id" }
                    },
                },
                monthlyFirstMonday: {
                    id: generatePK().toString(),
                    recurrenceType: ScheduleRecurrenceType.Monthly,
                    interval: 1,
                    dayOfWeek: 1, // Monday (first of month logic handled elsewhere)
                    duration: 180,
                    schedule: {
                        connect: { id: "schedule_id" }
                    },
                },
                monthlyLastDay: {
                    id: generatePK().toString(),
                    recurrenceType: ScheduleRecurrenceType.Monthly,
                    interval: 1,
                    dayOfMonth: 31, // Will adjust to last day of month
                    duration: 60,
                    schedule: {
                        connect: { id: "schedule_id" }
                    },
                },
                quarterly: {
                    id: generatePK().toString(),
                    recurrenceType: ScheduleRecurrenceType.Monthly,
                    interval: 3, // Every 3 months
                    dayOfMonth: 15,
                    duration: 240, // 4 hours
                    endDate: twoYears,
                    schedule: {
                        connect: { id: "schedule_id" }
                    },
                },
                yearlyBirthday: {
                    id: generatePK().toString(),
                    recurrenceType: ScheduleRecurrenceType.Yearly,
                    interval: 1,
                    month: 6, // June
                    dayOfMonth: 15,
                    duration: 1440, // All day (24 hours)
                    schedule: {
                        connect: { id: "schedule_id" }
                    },
                },
                leapYearEvent: {
                    id: generatePK().toString(),
                    recurrenceType: ScheduleRecurrenceType.Yearly,
                    interval: 1,
                    month: 2, // February
                    dayOfMonth: 29, // Only occurs on leap years
                    duration: 60,
                    schedule: {
                        connect: { id: "schedule_id" }
                    },
                },
                highFrequencyDaily: {
                    id: generatePK().toString(),
                    recurrenceType: ScheduleRecurrenceType.Daily,
                    interval: 1,
                    duration: 30, // 30 minutes
                    endDate: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000), // One week
                    schedule: {
                        connect: { id: "schedule_id" }
                    },
                },
                longTermYearly: {
                    id: generatePK().toString(),
                    recurrenceType: ScheduleRecurrenceType.Yearly,
                    interval: 1,
                    month: 12,
                    dayOfMonth: 1,
                    duration: 480, // 8 hours
                    endDate: new Date("2050-12-31T23:59:59Z"),
                    schedule: {
                        connect: { id: "schedule_id" }
                    },
                },
                extremeInterval: {
                    id: generatePK().toString(),
                    recurrenceType: ScheduleRecurrenceType.Monthly,
                    interval: 12, // Every year via monthly
                    dayOfMonth: 1,
                    duration: 60,
                    schedule: {
                        connect: { id: "schedule_id" }
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
            id: generatePK().toString(),
            recurrenceType: ScheduleRecurrenceType.Daily,
            interval: 1,
            schedule: {
                connect: { id: "default_schedule_id" }
            },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.schedule_recurrenceCreateInput>): Prisma.schedule_recurrenceCreateInput {
        return {
            id: generatePK().toString(),
            recurrenceType: ScheduleRecurrenceType.Weekly,
            interval: 1,
            dayOfWeek: 1, // Monday
            duration: 60,
            endDate: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000),
            schedule: {
                connect: { id: "default_schedule_id" }
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
    async createDaily(scheduleId: string, interval: number = 1, duration?: number): Promise<Prisma.schedule_recurrence> {
        return await this.createMinimal({
            recurrenceType: ScheduleRecurrenceType.Daily,
            interval,
            duration,
            schedule: { connect: { id: scheduleId } },
        });
    }

    async createWeekly(
        scheduleId: string,
        dayOfWeek: number,
        interval: number = 1,
        duration?: number
    ): Promise<Prisma.schedule_recurrence> {
        return await this.createMinimal({
            recurrenceType: ScheduleRecurrenceType.Weekly,
            interval,
            dayOfWeek,
            duration,
            schedule: { connect: { id: scheduleId } },
        });
    }

    async createMonthly(
        scheduleId: string,
        dayOfMonth: number,
        interval: number = 1,
        duration?: number
    ): Promise<Prisma.schedule_recurrence> {
        return await this.createMinimal({
            recurrenceType: ScheduleRecurrenceType.Monthly,
            interval,
            dayOfMonth,
            duration,
            schedule: { connect: { id: scheduleId } },
        });
    }

    async createYearly(
        scheduleId: string,
        month: number,
        dayOfMonth: number,
        interval: number = 1,
        duration?: number
    ): Promise<Prisma.schedule_recurrence> {
        return await this.createMinimal({
            recurrenceType: ScheduleRecurrenceType.Yearly,
            interval,
            month,
            dayOfMonth,
            duration,
            schedule: { connect: { id: scheduleId } },
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
        tx: any
    ): Promise<Prisma.schedule_recurrenceCreateInput> {
        let data = { ...baseData };

        // Handle schedule relationship
        if (config.withSchedule) {
            data.schedule = {
                connect: { id: config.withSchedule.scheduleId }
            };
        }

        return data;
    }

    protected async checkModelConstraints(record: Prisma.schedule_recurrence): Promise<string[]> {
        const violations: string[] = [];
        
        // Check interval
        if (record.interval < 1) {
            violations.push('Interval must be at least 1');
        }

        // Check duration
        if (record.duration !== null && record.duration !== undefined && record.duration < 0) {
            violations.push('Duration cannot be negative');
        }

        // Type-specific validation
        switch (record.recurrenceType) {
            case ScheduleRecurrenceType.Weekly:
                if (record.dayOfWeek === null || record.dayOfWeek === undefined) {
                    violations.push('Weekly recurrence requires day of week');
                } else if (record.dayOfWeek < 0 || record.dayOfWeek > 6) {
                    violations.push('Day of week must be between 0 (Sunday) and 6 (Saturday)');
                }
                break;

            case ScheduleRecurrenceType.Monthly:
                if (!record.dayOfMonth && !record.dayOfWeek) {
                    violations.push('Monthly recurrence requires either day of month or day of week');
                }
                if (record.dayOfMonth && (record.dayOfMonth < 1 || record.dayOfMonth > 31)) {
                    violations.push('Day of month must be between 1 and 31');
                }
                break;

            case ScheduleRecurrenceType.Yearly:
                if (!record.month) {
                    violations.push('Yearly recurrence requires month');
                } else if (record.month < 1 || record.month > 12) {
                    violations.push('Month must be between 1 and 12');
                }
                if (!record.dayOfMonth) {
                    violations.push('Yearly recurrence requires day of month');
                } else if (record.dayOfMonth < 1 || record.dayOfMonth > 31) {
                    violations.push('Day of month must be between 1 and 31');
                }
                break;

            case ScheduleRecurrenceType.Daily:
                // Daily recurrence doesn't need additional validation
                break;
        }

        // Check end date
        if (record.endDate && record.endDate < new Date()) {
            violations.push('End date cannot be in the past');
        }

        // Check schedule association
        if (!record.scheduleId) {
            violations.push('Schedule recurrence must belong to a schedule');
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {};
    }

    protected async deleteRelatedRecords(
        record: Prisma.schedule_recurrence,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[]
    ): Promise<void> {
        // ScheduleRecurrence has no dependent records to delete
    }

    /**
     * Create multiple recurrences for a schedule (e.g., MWF meetings)
     */
    async createMultiDayWeekly(
        scheduleId: string,
        daysOfWeek: number[],
        interval: number = 1,
        duration?: number
    ): Promise<Prisma.schedule_recurrence[]> {
        return await this.createMany(
            daysOfWeek.map(day => ({
                recurrenceType: ScheduleRecurrenceType.Weekly,
                interval,
                dayOfWeek: day,
                duration,
                schedule: { connect: { id: scheduleId } },
            }))
        );
    }

    /**
     * Create recurrence pattern from RRULE string
     */
    async createFromRRule(
        scheduleId: string,
        rrule: string,
        overrides?: Partial<Prisma.schedule_recurrenceCreateInput>
    ): Promise<Prisma.schedule_recurrence> {
        const parts = rrule.split(';').reduce((acc, part) => {
            const [key, value] = part.split('=');
            acc[key] = value;
            return acc;
        }, {} as Record<string, string>);
        
        let baseData: Partial<Prisma.schedule_recurrenceCreateInput> = {
            interval: parseInt(parts.INTERVAL || '1'),
        };
        
        // Map frequency
        switch (parts.FREQ) {
            case 'DAILY':
                baseData.recurrenceType = ScheduleRecurrenceType.Daily;
                break;
            case 'WEEKLY':
                baseData.recurrenceType = ScheduleRecurrenceType.Weekly;
                if (parts.BYDAY) {
                    const dayMap: Record<string, number> = { 
                        SU: 0, MO: 1, TU: 2, WE: 3, TH: 4, FR: 5, SA: 6 
                    };
                    const days = parts.BYDAY.split(',');
                    baseData.dayOfWeek = dayMap[days[0]] || 0;
                }
                break;
            case 'MONTHLY':
                baseData.recurrenceType = ScheduleRecurrenceType.Monthly;
                if (parts.BYMONTHDAY) {
                    baseData.dayOfMonth = parseInt(parts.BYMONTHDAY);
                }
                break;
            case 'YEARLY':
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
            schedule: { connect: { id: scheduleId } },
            ...overrides,
        });
    }

    /**
     * Create test recurrences for various scenarios
     */
    async createTestRecurrences(scheduleId: string): Promise<{
        daily: Prisma.schedule_recurrence;
        weekly: Prisma.schedule_recurrence;
        monthly: Prisma.schedule_recurrence;
        yearly: Prisma.schedule_recurrence;
        biweekly: Prisma.schedule_recurrence;
        quarterly: Prisma.schedule_recurrence;
    }> {
        const [daily, weekly, monthly, yearly, biweekly, quarterly] = await Promise.all([
            this.createDaily(scheduleId, 1, 30),
            this.createWeekly(scheduleId, 1, 1, 60), // Monday
            this.createMonthly(scheduleId, 15, 1, 90),
            this.createYearly(scheduleId, 12, 25, 1, 120), // Christmas
            this.createWeekly(scheduleId, 5, 2, 90), // Every other Friday
            this.seedScenario('quarterlyReview'),
        ]);

        return {
            daily,
            weekly,
            monthly,
            yearly,
            biweekly,
            quarterly: quarterly as unknown as Prisma.schedule_recurrence,
        };
    }

    /**
     * Create business meeting patterns
     */
    async createBusinessMeetingPatterns(scheduleId: string): Promise<{
        standup: Prisma.schedule_recurrence;
        teamMeeting: Prisma.schedule_recurrence;
        sprintPlanning: Prisma.schedule_recurrence;
        allHands: Prisma.schedule_recurrence;
        review: Prisma.schedule_recurrence;
    }> {
        const [standup, teamMeeting, sprintPlanning, allHands, review] = await Promise.all([
            this.seedScenario('dailyStandup'),
            this.seedScenario('weeklyTeamMeeting'),
            this.seedScenario('biWeeklySprintPlanning'),
            this.seedScenario('monthlyAllHands'),
            this.seedScenario('annualReview'),
        ]);

        return {
            standup: standup as unknown as Prisma.schedule_recurrence,
            teamMeeting: teamMeeting as unknown as Prisma.schedule_recurrence,
            sprintPlanning: sprintPlanning as unknown as Prisma.schedule_recurrence,
            allHands: allHands as unknown as Prisma.schedule_recurrence,
            review: review as unknown as Prisma.schedule_recurrence,
        };
    }
}

// Export factory creator function
export const createScheduleRecurrenceEnhancedDbFactory = (prisma: PrismaClient) => 
    ScheduleRecurrenceEnhancedDbFactory.getInstance('ScheduleRecurrence', prisma);

// Export the class for type usage
export { ScheduleRecurrenceEnhancedDbFactory as ScheduleRecurrenceEnhancedDbFactoryClass };