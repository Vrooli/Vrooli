import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient, ScheduleRecurrenceType } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface ScheduleRelationConfig extends RelationConfig {
    withExceptions?: boolean | number;
    withRecurrences?: boolean | number;
    withUser?: { userId: string };
    withTeam?: { teamId: string };
    withRuns?: number;
    withMeetings?: number;
}

/**
 * Enhanced database fixture factory for Schedule model
 * Provides comprehensive testing capabilities for schedules with exceptions and recurrences
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for complex recurrence patterns
 * - Exception handling for schedule modifications
 * - User and team associations
 * - Timezone support
 * - Predefined test scenarios
 * - Temporal validation
 */
export class ScheduleDbFactory extends EnhancedDatabaseFactory<
    Prisma.scheduleCreateInput,
    Prisma.scheduleCreateInput,
    Prisma.scheduleInclude,
    Prisma.scheduleUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('Schedule', prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.schedule;
    }

    /**
     * Get complete test fixtures for Schedule model
     */
    protected getFixtures(): DbTestFixtures<Prisma.scheduleCreateInput, Prisma.scheduleUpdateInput> {
        const now = new Date();
        const tomorrow = new Date(now.getTime() + 24 * 60 * 60 * 1000);
        const nextWeek = new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000);
        const nextMonth = new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000);

        return {
            minimal: {
                id: generatePK().toString(),
                publicId: generatePublicId(),
                startTime: tomorrow,
                endTime: new Date(tomorrow.getTime() + 60 * 60 * 1000), // 1 hour later
                timezone: "UTC",
                user: {
                    connect: { id: "user_id" }
                },
            },
            complete: {
                id: generatePK().toString(),
                publicId: generatePublicId(),
                startTime: tomorrow,
                endTime: new Date(tomorrow.getTime() + 2 * 60 * 60 * 1000), // 2 hours later
                timezone: "America/New_York",
                user: {
                    connect: { id: "user_id" }
                },
                exceptions: {
                    create: [
                        {
                            id: generatePK().toString(),
                            originalStartTime: new Date(tomorrow.getTime() + 24 * 60 * 60 * 1000),
                            newStartTime: new Date(tomorrow.getTime() + 25 * 60 * 60 * 1000),
                            newEndTime: new Date(tomorrow.getTime() + 27 * 60 * 60 * 1000),
                        },
                    ],
                },
                recurrences: {
                    create: [
                        {
                            id: generatePK().toString(),
                            recurrenceType: ScheduleRecurrenceType.Daily,
                            interval: 1,
                            endDate: nextMonth,
                            duration: 120, // 2 hours in minutes
                        },
                    ],
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, publicId, startTime, endTime, timezone, user
                    createdAt: new Date(),
                },
                invalidTypes: {
                    id: "not-a-bigint",
                    publicId: 123, // Should be string
                    startTime: "tomorrow", // Should be Date
                    endTime: "later", // Should be Date
                    timezone: null, // Should be string
                    userId: "string-not-bigint", // Should be BigInt
                },
                invalidTimeRange: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    startTime: tomorrow,
                    endTime: now, // End time before start time
                    timezone: "UTC",
                    user: {
                        connect: { id: "user_id" }
                    },
                },
                invalidTimezone: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    startTime: tomorrow,
                    endTime: new Date(tomorrow.getTime() + 60 * 60 * 1000),
                    timezone: "Invalid/Timezone",
                    user: {
                        connect: { id: "user_id" }
                    },
                },
                conflictingOwnership: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    startTime: tomorrow,
                    endTime: new Date(tomorrow.getTime() + 60 * 60 * 1000),
                    timezone: "UTC",
                    // Both user and team ownership
                    user: {
                        connect: { id: "user_id" }
                    },
                    team: {
                        connect: { id: "team_id" }
                    },
                },
            },
            edgeCases: {
                allDayEvent: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    startTime: new Date(tomorrow.setHours(0, 0, 0, 0)),
                    endTime: new Date(tomorrow.setHours(23, 59, 59, 999)),
                    timezone: "UTC",
                    user: {
                        connect: { id: "user_id" }
                    },
                },
                multiTimezoneEvent: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    startTime: tomorrow,
                    endTime: new Date(tomorrow.getTime() + 3 * 60 * 60 * 1000),
                    timezone: "Asia/Tokyo",
                    user: {
                        connect: { id: "user_id" }
                    },
                },
                complexRecurrence: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    startTime: tomorrow,
                    endTime: new Date(tomorrow.getTime() + 90 * 60 * 1000),
                    timezone: "UTC",
                    user: {
                        connect: { id: "user_id" }
                    },
                    recurrences: {
                        create: [
                            {
                                id: generatePK().toString(),
                                recurrenceType: ScheduleRecurrenceType.Weekly,
                                interval: 2,
                                dayOfWeek: 3, // Wednesday
                                endDate: nextMonth,
                                duration: 90,
                            },
                            {
                                id: generatePK().toString(),
                                recurrenceType: ScheduleRecurrenceType.Monthly,
                                interval: 1,
                                dayOfMonth: 15,
                                endDate: new Date(now.getTime() + 365 * 24 * 60 * 60 * 1000),
                                duration: 120,
                            },
                        ],
                    },
                },
                manyExceptions: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    startTime: tomorrow,
                    endTime: new Date(tomorrow.getTime() + 60 * 60 * 1000),
                    timezone: "UTC",
                    user: {
                        connect: { id: "user_id" }
                    },
                    exceptions: {
                        create: Array.from({ length: 10 }, (_, i) => ({
                            id: generatePK().toString(),
                            originalStartTime: new Date(tomorrow.getTime() + (i + 1) * 24 * 60 * 60 * 1000),
                            newStartTime: new Date(tomorrow.getTime() + (i + 1) * 24 * 60 * 60 * 1000 + 2 * 60 * 60 * 1000),
                            newEndTime: new Date(tomorrow.getTime() + (i + 1) * 24 * 60 * 60 * 1000 + 3 * 60 * 60 * 1000),
                        })),
                    },
                },
                yearlyRecurrence: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    startTime: tomorrow,
                    endTime: new Date(tomorrow.getTime() + 4 * 60 * 60 * 1000),
                    timezone: "UTC",
                    user: {
                        connect: { id: "user_id" }
                    },
                    recurrences: {
                        create: [
                            {
                                id: generatePK().toString(),
                                recurrenceType: ScheduleRecurrenceType.Yearly,
                                interval: 1,
                                month: 12, // December
                                dayOfMonth: 25, // Christmas
                                duration: 240,
                            },
                        ],
                    },
                },
                teamSchedule: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    startTime: tomorrow,
                    endTime: new Date(tomorrow.getTime() + 2 * 60 * 60 * 1000),
                    timezone: "UTC",
                    team: {
                        connect: { id: "team_id" }
                    },
                },
            },
            updates: {
                minimal: {
                    endTime: new Date(tomorrow.getTime() + 3 * 60 * 60 * 1000),
                },
                complete: {
                    startTime: nextWeek,
                    endTime: new Date(nextWeek.getTime() + 2 * 60 * 60 * 1000),
                    timezone: "Europe/London",
                    exceptions: {
                        create: [{
                            id: generatePK().toString(),
                            originalStartTime: new Date(nextWeek.getTime() + 24 * 60 * 60 * 1000),
                            newStartTime: null, // Cancelled
                            newEndTime: null,
                        }],
                    },
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.scheduleCreateInput>): Prisma.scheduleCreateInput {
        const tomorrow = new Date(Date.now() + 24 * 60 * 60 * 1000);
        
        return {
            id: generatePK().toString(),
            publicId: generatePublicId(),
            startTime: tomorrow,
            endTime: new Date(tomorrow.getTime() + 60 * 60 * 1000),
            timezone: "UTC",
            user: {
                connect: { id: "default_user_id" }
            },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.scheduleCreateInput>): Prisma.scheduleCreateInput {
        const tomorrow = new Date(Date.now() + 24 * 60 * 60 * 1000);
        
        return {
            id: generatePK().toString(),
            publicId: generatePublicId(),
            startTime: tomorrow,
            endTime: new Date(tomorrow.getTime() + 2 * 60 * 60 * 1000),
            timezone: "America/New_York",
            user: {
                connect: { id: "default_user_id" }
            },
            exceptions: {
                create: [
                    {
                        id: generatePK().toString(),
                        originalStartTime: new Date(tomorrow.getTime() + 7 * 24 * 60 * 60 * 1000),
                        newStartTime: new Date(tomorrow.getTime() + 7 * 24 * 60 * 60 * 1000 + 2 * 60 * 60 * 1000),
                        newEndTime: new Date(tomorrow.getTime() + 7 * 24 * 60 * 60 * 1000 + 4 * 60 * 60 * 1000),
                    },
                ],
            },
            recurrences: {
                create: [
                    {
                        id: generatePK().toString(),
                        recurrenceType: ScheduleRecurrenceType.Weekly,
                        interval: 1,
                        endDate: new Date(tomorrow.getTime() + 90 * 24 * 60 * 60 * 1000),
                        duration: 120,
                    },
                ],
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
                description: "Daily standup meeting schedule",
                config: {
                    overrides: {
                        startTime: new Date(new Date().setHours(9, 0, 0, 0)),
                        endTime: new Date(new Date().setHours(9, 30, 0, 0)),
                        timezone: "America/New_York",
                    },
                    withRecurrences: true,
                },
            },
            weeklyTeamMeeting: {
                name: "weeklyTeamMeeting",
                description: "Weekly team meeting with exceptions",
                config: {
                    overrides: {
                        startTime: new Date(new Date().setHours(14, 0, 0, 0)),
                        endTime: new Date(new Date().setHours(15, 0, 0, 0)),
                        timezone: "UTC",
                    },
                    withRecurrences: true,
                    withExceptions: 2,
                },
            },
            monthlyReview: {
                name: "monthlyReview",
                description: "Monthly review meeting",
                config: {
                    overrides: {
                        startTime: new Date(new Date().setHours(10, 0, 0, 0)),
                        endTime: new Date(new Date().setHours(12, 0, 0, 0)),
                        timezone: "UTC",
                    },
                    withRecurrences: true,
                },
            },
            oneTimeEvent: {
                name: "oneTimeEvent",
                description: "One-time event without recurrence",
                config: {
                    overrides: {
                        startTime: new Date(Date.now() + 3 * 24 * 60 * 60 * 1000),
                        endTime: new Date(Date.now() + 3 * 24 * 60 * 60 * 1000 + 4 * 60 * 60 * 1000),
                        timezone: "Europe/London",
                    },
                },
            },
            complexSchedule: {
                name: "complexSchedule",
                description: "Complex schedule with multiple recurrences and exceptions",
                config: {
                    overrides: {
                        startTime: new Date(new Date().setHours(8, 0, 0, 0)),
                        endTime: new Date(new Date().setHours(17, 0, 0, 0)),
                        timezone: "Asia/Tokyo",
                    },
                    withRecurrences: 3,
                    withExceptions: 5,
                },
            },
        };
    }

    /**
     * Create specific schedule types
     */
    async createDailySchedule(userId: string): Promise<Prisma.schedule> {
        const tomorrow = new Date(Date.now() + 24 * 60 * 60 * 1000);
        tomorrow.setHours(9, 0, 0, 0);
        
        return await this.createWithRelations({
            overrides: {
                startTime: tomorrow,
                endTime: new Date(tomorrow.getTime() + 60 * 60 * 1000),
                user: { connect: { id: userId } },
            },
            withRecurrences: true,
        });
    }

    async createWeeklySchedule(userId: string, dayOfWeek: number = 1): Promise<Prisma.schedule> {
        const nextWeek = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000);
        
        return await this.createComplete({
            startTime: nextWeek,
            endTime: new Date(nextWeek.getTime() + 2 * 60 * 60 * 1000),
            user: { connect: { id: userId } },
            recurrences: {
                create: [{
                    id: generatePK().toString(),
                    recurrenceType: ScheduleRecurrenceType.Weekly,
                    interval: 1,
                    dayOfWeek,
                    duration: 120,
                }],
            },
        });
    }

    async createTeamSchedule(teamId: string): Promise<Prisma.schedule> {
        return await this.createMinimal({
            user: undefined,
            team: { connect: { id: teamId } },
        });
    }

    protected getDefaultInclude(): Prisma.scheduleInclude {
        return {
            user: true,
            team: true,
            exceptions: true,
            recurrences: true,
            meetings: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                },
            },
            runs: {
                select: {
                    id: true,
                    name: true,
                    status: true,
                },
            },
            _count: {
                select: {
                    exceptions: true,
                    recurrences: true,
                    meetings: true,
                    runs: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.scheduleCreateInput,
        config: ScheduleRelationConfig,
        tx: any
    ): Promise<Prisma.scheduleCreateInput> {
        let data = { ...baseData };

        // Handle user relationship
        if (config.withUser) {
            data.user = {
                connect: { id: config.withUser.userId }
            };
        }

        // Handle team relationship
        if (config.withTeam) {
            data.team = {
                connect: { id: config.withTeam.teamId }
            };
            // Remove user if team is set
            delete data.user;
        }

        // Handle exceptions
        if (config.withExceptions) {
            const exceptionCount = typeof config.withExceptions === 'number' ? config.withExceptions : 1;
            const startTime = data.startTime as Date;
            
            data.exceptions = {
                create: Array.from({ length: exceptionCount }, (_, i) => ({
                    id: generatePK().toString(),
                    originalStartTime: new Date(startTime.getTime() + (i + 1) * 7 * 24 * 60 * 60 * 1000),
                    newStartTime: new Date(startTime.getTime() + (i + 1) * 7 * 24 * 60 * 60 * 1000 + 2 * 60 * 60 * 1000),
                    newEndTime: new Date(startTime.getTime() + (i + 1) * 7 * 24 * 60 * 60 * 1000 + 3 * 60 * 60 * 1000),
                })),
            };
        }

        // Handle recurrences
        if (config.withRecurrences) {
            const recurrenceCount = typeof config.withRecurrences === 'number' ? config.withRecurrences : 1;
            const recurrenceTypes = [
                ScheduleRecurrenceType.Daily,
                ScheduleRecurrenceType.Weekly,
                ScheduleRecurrenceType.Monthly,
            ];
            
            data.recurrences = {
                create: Array.from({ length: recurrenceCount }, (_, i) => ({
                    id: generatePK().toString(),
                    recurrenceType: recurrenceTypes[i % recurrenceTypes.length],
                    interval: 1,
                    duration: 60,
                    endDate: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000), // 1 year
                })),
            };
        }

        return data;
    }

    protected async checkModelConstraints(record: Prisma.schedule): Promise<string[]> {
        const violations: string[] = [];
        
        // Check time validity
        if (record.startTime >= record.endTime) {
            violations.push('Start time must be before end time');
        }

        // Check timezone validity
        try {
            new Intl.DateTimeFormat('en-US', { timeZone: record.timezone });
        } catch {
            violations.push('Invalid timezone');
        }

        // Check ownership (must have either user or team, not both)
        if (record.userId && record.teamId) {
            violations.push('Schedule cannot belong to both user and team');
        }

        if (!record.userId && !record.teamId) {
            violations.push('Schedule must belong to either user or team');
        }

        // Check exceptions
        if (record.exceptions) {
            for (const exception of record.exceptions) {
                if (exception.newStartTime && exception.newEndTime && 
                    exception.newStartTime >= exception.newEndTime) {
                    violations.push('Exception new start time must be before new end time');
                }
            }
        }

        // Check recurrences
        if (record.recurrences) {
            for (const recurrence of record.recurrences) {
                if (recurrence.interval < 1) {
                    violations.push('Recurrence interval must be positive');
                }

                if (recurrence.recurrenceType === ScheduleRecurrenceType.Weekly && 
                    (recurrence.dayOfWeek === null || recurrence.dayOfWeek < 0 || recurrence.dayOfWeek > 6)) {
                    violations.push('Weekly recurrence must have valid day of week (0-6)');
                }

                if (recurrence.recurrenceType === ScheduleRecurrenceType.Monthly && 
                    (recurrence.dayOfMonth === null || recurrence.dayOfMonth < 1 || recurrence.dayOfMonth > 31)) {
                    violations.push('Monthly recurrence must have valid day of month (1-31)');
                }

                if (recurrence.recurrenceType === ScheduleRecurrenceType.Yearly && 
                    (recurrence.month === null || recurrence.month < 1 || recurrence.month > 12)) {
                    violations.push('Yearly recurrence must have valid month (1-12)');
                }
            }
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            exceptions: true,
            recurrences: true,
            meetings: true,
            runs: true,
            subscriptions: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.schedule,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[]
    ): Promise<void> {
        const shouldDelete = (relation: string) => 
            !includeOnly || includeOnly.includes(relation);

        // Delete exceptions
        if (shouldDelete('exceptions') && record.exceptions?.length) {
            await tx.schedule_exception.deleteMany({
                where: { scheduleId: record.id },
            });
        }

        // Delete recurrences
        if (shouldDelete('recurrences') && record.recurrences?.length) {
            await tx.schedule_recurrence.deleteMany({
                where: { scheduleId: record.id },
            });
        }

        // Delete meetings
        if (shouldDelete('meetings') && record.meetings?.length) {
            await tx.meeting.deleteMany({
                where: { scheduleId: record.id },
            });
        }

        // Delete runs
        if (shouldDelete('runs') && record.runs?.length) {
            await tx.run.deleteMany({
                where: { scheduleId: record.id },
            });
        }

        // Delete notification subscriptions
        if (shouldDelete('subscriptions') && record.subscriptions?.length) {
            await tx.notification_subscription.deleteMany({
                where: { scheduleId: record.id },
            });
        }
    }

    /**
     * Create a schedule with multiple occurrences
     */
    async createRecurringSchedule(
        type: ScheduleRecurrenceType,
        interval: number,
        duration: number,
        userId: string
    ): Promise<Prisma.schedule> {
        return await this.createComplete({
            user: { connect: { id: userId } },
            recurrences: {
                create: [{
                    id: generatePK().toString(),
                    recurrenceType: type,
                    interval,
                    duration,
                    endDate: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000),
                }],
            },
        });
    }

    /**
     * Create test schedules for different scenarios
     */
    async createTestSchedules(userId: string): Promise<{
        daily: Prisma.schedule;
        weekly: Prisma.schedule;
        monthly: Prisma.schedule;
        oneTime: Prisma.schedule;
        withExceptions: Prisma.schedule;
    }> {
        const [daily, weekly, monthly, oneTime, withExceptions] = await Promise.all([
            this.createDailySchedule(userId),
            this.createWeeklySchedule(userId),
            this.createRecurringSchedule(ScheduleRecurrenceType.Monthly, 1, 120, userId),
            this.createMinimal({ user: { connect: { id: userId } } }),
            this.seedScenario('weeklyTeamMeeting'),
        ]);

        return {
            daily,
            weekly,
            monthly,
            oneTime,
            withExceptions: withExceptions as unknown as Prisma.schedule,
        };
    }
}

// Export factory creator function
export const createScheduleDbFactory = (prisma: PrismaClient) => 
    ScheduleDbFactory.getInstance('Schedule', prisma);

// Export the class for type usage
export { ScheduleDbFactory as ScheduleDbFactoryClass };