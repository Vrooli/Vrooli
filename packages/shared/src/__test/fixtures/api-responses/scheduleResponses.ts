/* c8 ignore start */
/**
 * Schedule API Response Fixtures
 * 
 * Comprehensive fixtures for schedule management endpoints including
 * recurring schedules, time zones, and calendar integration.
 */

import type {
    Schedule,
    ScheduleCreateInput,
    ScheduleUpdateInput,
} from "../../../api/types.js";
import { generatePK } from "../../../id/index.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";

// Constants
const DEFAULT_COUNT = 10;
const DEFAULT_ERROR_RATE = 0.1;
const DEFAULT_DELAY_MS = 500;
const HOURS_24 = 24 * 60 * 60 * 1000;
const DAYS_7 = 7 * HOURS_24;
const DAYS_30 = 30 * HOURS_24;

// Recurrence types
type RecurrenceType = "Daily" | "Weekly" | "Monthly" | "Yearly";

/**
 * Schedule API response factory
 */
export class ScheduleResponseFactory extends BaseAPIResponseFactory<
    Schedule,
    ScheduleCreateInput,
    ScheduleUpdateInput
> {
    protected readonly entityName = "schedule";

    /**
     * Create mock schedule data
     */
    createMockData(options?: MockDataOptions): Schedule {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const scheduleId = options?.overrides?.id || generatePK().toString();
        const nextWeek = new Date(Date.now().toISOString() + DAYS_7).toISOString();

        const baseSchedule: Schedule = {
            __typename: "Schedule",
            id: scheduleId,
            createdAt: now,
            updatedAt: now,
            startTime: nextWeek,
            endTime: null,
            timezone: "UTC",
            recurrences: [],
            recurrencesCount: 0,
            exceptions: [],
            exceptionsCount: 0,
            labels: [],
            labelsCount: 0,
            focusModes: [],
            focusModesCount: 0,
            meetings: [],
            meetingsCount: 0,
            runProjects: [],
            runProjectsCount: 0,
            runRoutines: [],
            runRoutinesCount: 0,
            reminders: [],
            remindersCount: 0,
            you: {
                canDelete: true,
                canUpdate: true,
            },
        };

        if (scenario === "complete" || scenario === "edge-case") {
            const startTime = new Date(Date.now().toISOString() + HOURS_24).toISOString();
            const endTime = new Date(Date.now().toISOString() + HOURS_24 + (2 * 60 * 60 * 1000)).toISOString(); // 2 hours later

            return {
                ...baseSchedule,
                startTime,
                endTime,
                timezone: "America/New_York",
                recurrences: [{
                    __typename: "ScheduleRecurrence",
                    id: generatePK().toString(),
                    createdAt: now,
                    updatedAt: now,
                    recurrenceType: "Weekly",
                    interval: 1,
                    dayOfWeek: 1, // Monday
                    dayOfMonth: null,
                    month: null,
                    endDate: new Date(Date.now().toISOString() + (365 * HOURS_24)).toISOString(), // 1 year
                }],
                recurrencesCount: 1,
                exceptions: [{
                    __typename: "ScheduleException",
                    id: generatePK().toString(),
                    createdAt: now,
                    updatedAt: now,
                    originalStartTime: startTime,
                    newStartTime: new Date(new Date(startTime).toISOString().getTime() + HOURS_24).toISOString(),
                    newEndTime: new Date(new Date(endTime).toISOString().getTime() + HOURS_24).toISOString(),
                }],
                exceptionsCount: 1,
                labels: [{
                    __typename: "Label",
                    id: generatePK().toString(),
                    createdAt: now,
                    updatedAt: now,
                    label: "Important",
                    color: "#ff4444",
                }],
                labelsCount: 1,
                reminders: [{
                    __typename: "Reminder",
                    id: generatePK().toString(),
                    createdAt: now,
                    updatedAt: now,
                    durationMinutes: 15,
                }],
                remindersCount: 1,
                you: {
                    canDelete: true,
                    canUpdate: true,
                },
                ...options?.overrides,
            };
        }

        return {
            ...baseSchedule,
            ...options?.overrides,
        };
    }

    /**
     * Create schedule from input
     */
    createFromInput(input: ScheduleCreateInput): Schedule {
        const now = new Date().toISOString();
        const scheduleId = generatePK().toString();

        return {
            __typename: "Schedule",
            id: scheduleId,
            createdAt: now,
            updatedAt: now,
            startTime: input.startTime || new Date(Date.now().toISOString() + HOURS_24).toISOString(),
            endTime: input.endTime || null,
            timezone: input.timezone || "UTC",
            recurrences: input.recurrencesCreate?.map(r => ({
                __typename: "ScheduleRecurrence" as const,
                id: generatePK().toString(),
                createdAt: now,
                updatedAt: now,
                recurrenceType: r.recurrenceType as RecurrenceType,
                interval: r.interval || 1,
                dayOfWeek: r.dayOfWeek || null,
                dayOfMonth: r.dayOfMonth || null,
                month: r.month || null,
                endDate: r.endDate || null,
            })) || [],
            recurrencesCount: input.recurrencesCreate?.length || 0,
            exceptions: input.exceptionsCreate?.map(e => ({
                __typename: "ScheduleException" as const,
                id: generatePK().toString(),
                createdAt: now,
                updatedAt: now,
                originalStartTime: e.originalStartTime,
                newStartTime: e.newStartTime || null,
                newEndTime: e.newEndTime || null,
            })) || [],
            exceptionsCount: input.exceptionsCreate?.length || 0,
            labels: [],
            labelsCount: 0,
            focusModes: [],
            focusModesCount: 0,
            meetings: [],
            meetingsCount: 0,
            runProjects: [],
            runProjectsCount: 0,
            runRoutines: [],
            runRoutinesCount: 0,
            reminders: input.remindersCreate?.map(r => ({
                __typename: "Reminder" as const,
                id: generatePK().toString(),
                createdAt: now,
                updatedAt: now,
                durationMinutes: r.durationMinutes || 15,
            })) || [],
            remindersCount: input.remindersCreate?.length || 0,
            you: {
                canDelete: true,
                canUpdate: true,
            },
        };
    }

    /**
     * Update schedule from input
     */
    updateFromInput(existing: Schedule, input: ScheduleUpdateInput): Schedule {
        const updates: Partial<Schedule> = {
            updatedAt: new Date().toISOString(),
        };

        if (input.startTime !== undefined) updates.startTime = input.startTime;
        if (input.endTime !== undefined) updates.endTime = input.endTime;
        if (input.timezone !== undefined) updates.timezone = input.timezone;

        // Handle recurrence updates
        if (input.recurrencesUpdate) {
            updates.recurrences = existing.recurrences?.map(recurrence => {
                const update = input.recurrencesUpdate?.find(u => u.id === recurrence.id);
                return update ? { ...recurrence, ...update } : recurrence;
            });
        }

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: ScheduleCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.startTime) {
            errors.startTime = "Start time is required";
        } else {
            const startTime = new Date(input.startTime).toISOString();
            if (isNaN(startTime.getTime())) {
                errors.startTime = "Invalid start time format";
            } else if (startTime.getTime() < Date.now()) {
                errors.startTime = "Start time must be in the future";
            }
        }

        if (input.endTime) {
            const endTime = new Date(input.endTime).toISOString();
            if (isNaN(endTime.getTime())) {
                errors.endTime = "Invalid end time format";
            } else if (input.startTime && endTime.getTime() <= new Date(input.startTime).toISOString().getTime()) {
                errors.endTime = "End time must be after start time";
            }
        }

        if (!input.timezone) {
            errors.timezone = "Timezone is required";
        }

        // Validate recurrences
        if (input.recurrencesCreate) {
            input.recurrencesCreate.forEach((recurrence, index) => {
                if (!recurrence.recurrenceType) {
                    errors[`recurrences.${index}.recurrenceType`] = "Recurrence type is required";
                }
                if (!recurrence.interval || recurrence.interval < 1) {
                    errors[`recurrences.${index}.interval`] = "Interval must be at least 1";
                }
            });
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: ScheduleUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.startTime !== undefined) {
            const startTime = new Date(input.startTime).toISOString();
            if (isNaN(startTime.getTime())) {
                errors.startTime = "Invalid start time format";
            }
        }

        if (input.endTime !== undefined && input.endTime !== null) {
            const endTime = new Date(input.endTime).toISOString();
            if (isNaN(endTime.getTime())) {
                errors.endTime = "Invalid end time format";
            }
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create daily recurring schedule
     */
    createDailySchedule(): Schedule {
        const tomorrow = new Date(Date.now().toISOString() + HOURS_24);
        return this.createMockData({
            scenario: "complete",
            overrides: {
                startTime: tomorrow.toISOString(),
                endTime: new Date(tomorrow.getTime().toISOString() + (60 * 60 * 1000)).toISOString(), // 1 hour later
                recurrences: [{
                    __typename: "ScheduleRecurrence",
                    id: generatePK().toString(),
                    createdAt: new Date().toISOString(),
                    updatedAt: new Date().toISOString(),
                    recurrenceType: "Daily",
                    interval: 1,
                    dayOfWeek: null,
                    dayOfMonth: null,
                    month: null,
                    endDate: new Date(Date.now().toISOString() + (30 * HOURS_24)).toISOString(), // 30 days
                }],
                recurrencesCount: 1,
            },
        });
    }

    /**
     * Create weekly recurring schedule
     */
    createWeeklySchedule(): Schedule {
        const nextMonday = new Date().toISOString();
        nextMonday.setDate(nextMonday.getDate() + ((1 + 7 - nextMonday.getDay()) % 7));

        return this.createMockData({
            scenario: "complete",
            overrides: {
                startTime: nextMonday.toISOString(),
                endTime: new Date(nextMonday.getTime().toISOString() + (2 * 60 * 60 * 1000)).toISOString(), // 2 hours later
                recurrences: [{
                    __typename: "ScheduleRecurrence",
                    id: generatePK().toString(),
                    createdAt: new Date().toISOString(),
                    updatedAt: new Date().toISOString(),
                    recurrenceType: "Weekly",
                    interval: 1,
                    dayOfWeek: 1, // Monday
                    dayOfMonth: null,
                    month: null,
                    endDate: new Date(Date.now().toISOString() + (365 * HOURS_24)).toISOString(), // 1 year
                }],
                recurrencesCount: 1,
            },
        });
    }

    /**
     * Create monthly recurring schedule
     */
    createMonthlySchedule(): Schedule {
        const nextMonth = new Date().toISOString();
        nextMonth.setMonth(nextMonth.getMonth() + 1);
        nextMonth.setDate(1); // First day of next month

        return this.createMockData({
            scenario: "complete",
            overrides: {
                startTime: nextMonth.toISOString(),
                endTime: new Date(nextMonth.getTime().toISOString() + (3 * 60 * 60 * 1000)).toISOString(), // 3 hours later
                recurrences: [{
                    __typename: "ScheduleRecurrence",
                    id: generatePK().toString(),
                    createdAt: new Date().toISOString(),
                    updatedAt: new Date().toISOString(),
                    recurrenceType: "Monthly",
                    interval: 1,
                    dayOfWeek: null,
                    dayOfMonth: 1,
                    month: null,
                    endDate: new Date(Date.now().toISOString() + (365 * HOURS_24)).toISOString(), // 1 year
                }],
                recurrencesCount: 1,
            },
        });
    }

    /**
     * Create schedule with exceptions
     */
    createScheduleWithExceptions(): Schedule {
        const baseSchedule = this.createWeeklySchedule();
        const originalStart = new Date(baseSchedule.startTime).toISOString();
        const nextWeek = new Date(originalStart.getTime().toISOString() + DAYS_7);

        return {
            ...baseSchedule,
            exceptions: [{
                __typename: "ScheduleException",
                id: generatePK().toString(),
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
                originalStartTime: nextWeek.toISOString(),
                newStartTime: new Date(nextWeek.getTime().toISOString() + (2 * 60 * 60 * 1000)).toISOString(), // 2 hours later
                newEndTime: new Date(nextWeek.getTime().toISOString() + (4 * 60 * 60 * 1000)).toISOString(), // 4 hours later
            }],
            exceptionsCount: 1,
        };
    }

    /**
     * Create schedule with multiple time zones
     */
    createMultiTimezoneSchedules(): Schedule[] {
        const timezones = [
            "America/New_York",
            "Europe/London",
            "Asia/Tokyo",
            "Australia/Sydney",
            "America/Los_Angeles",
        ];

        return timezones.map(timezone =>
            this.createMockData({
                overrides: {
                    timezone,
                    startTime: new Date(Date.now().toISOString() + HOURS_24).toISOString(),
                },
            }),
        );
    }

    /**
     * Create schedule conflict error response
     */
    createScheduleConflictErrorResponse(conflictingScheduleId: string, conflictTime: string) {
        return this.createBusinessErrorResponse("conflict", {
            resource: "schedule",
            conflictingScheduleId,
            conflictTime,
            message: "Schedule conflicts with existing appointment",
        });
    }

    /**
     * Create timezone error response
     */
    createTimezoneErrorResponse(invalidTimezone: string) {
        return this.createValidationErrorResponse({
            timezone: `Invalid timezone: ${invalidTimezone}`,
        });
    }

    /**
     * Create recurrence limit error response
     */
    createRecurrenceLimitErrorResponse(limit = 50) {
        return this.createBusinessErrorResponse("limit", {
            resource: "recurrences",
            limit,
            current: limit + 1,
            message: "Schedule has too many recurrence rules",
        });
    }
}

/**
 * Pre-configured schedule response scenarios
 */
export const scheduleResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<ScheduleCreateInput>) => {
        const factory = new ScheduleResponseFactory();
        const defaultInput: ScheduleCreateInput = {
            startTime: new Date(Date.now().toISOString() + HOURS_24).toISOString(),
            timezone: "UTC",
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    findSuccess: (schedule?: Schedule) => {
        const factory = new ScheduleResponseFactory();
        return factory.createSuccessResponse(
            schedule || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new ScheduleResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: Schedule, updates?: Partial<ScheduleUpdateInput>) => {
        const factory = new ScheduleResponseFactory();
        const schedule = existing || factory.createMockData({ scenario: "complete" });
        const input: ScheduleUpdateInput = {
            id: schedule.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(schedule, input),
        );
    },

    dailyScheduleSuccess: () => {
        const factory = new ScheduleResponseFactory();
        return factory.createSuccessResponse(
            factory.createDailySchedule(),
        );
    },

    weeklyScheduleSuccess: () => {
        const factory = new ScheduleResponseFactory();
        return factory.createSuccessResponse(
            factory.createWeeklySchedule(),
        );
    },

    monthlyScheduleSuccess: () => {
        const factory = new ScheduleResponseFactory();
        return factory.createSuccessResponse(
            factory.createMonthlySchedule(),
        );
    },

    scheduleWithExceptionsSuccess: () => {
        const factory = new ScheduleResponseFactory();
        return factory.createSuccessResponse(
            factory.createScheduleWithExceptions(),
        );
    },

    listSuccess: (schedules?: Schedule[]) => {
        const factory = new ScheduleResponseFactory();
        return factory.createPaginatedResponse(
            schedules || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: schedules?.length || DEFAULT_COUNT },
        );
    },

    multiTimezoneSuccess: () => {
        const factory = new ScheduleResponseFactory();
        const schedules = factory.createMultiTimezoneSchedules();
        return factory.createPaginatedResponse(
            schedules,
            { page: 1, totalCount: schedules.length },
        );
    },

    upcomingSchedulesSuccess: () => {
        const factory = new ScheduleResponseFactory();
        const schedules = [
            factory.createDailySchedule(),
            factory.createWeeklySchedule(),
            factory.createMonthlySchedule(),
        ];
        return factory.createPaginatedResponse(
            schedules,
            { page: 1, totalCount: schedules.length },
        );
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new ScheduleResponseFactory();
        return factory.createValidationErrorResponse({
            startTime: "Start time is required",
            timezone: "Timezone is required",
            "recurrences.0.recurrenceType": "Recurrence type is required",
            "recurrences.0.interval": "Interval must be at least 1",
        });
    },

    notFoundError: (scheduleId?: string) => {
        const factory = new ScheduleResponseFactory();
        return factory.createNotFoundErrorResponse(
            scheduleId || "non-existent-schedule",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new ScheduleResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "update",
            ["schedule:update"],
        );
    },

    scheduleConflictError: (conflictingScheduleId?: string) => {
        const factory = new ScheduleResponseFactory();
        return factory.createScheduleConflictErrorResponse(
            conflictingScheduleId || generatePK().toString(),
            new Date(Date.now().toISOString() + HOURS_24).toISOString(),
        );
    },

    pastTimeError: () => {
        const factory = new ScheduleResponseFactory();
        return factory.createValidationErrorResponse({
            startTime: "Start time must be in the future",
        });
    },

    invalidTimezoneError: (timezone = "Invalid/Timezone") => {
        const factory = new ScheduleResponseFactory();
        return factory.createTimezoneErrorResponse(timezone);
    },

    recurrenceLimitError: () => {
        const factory = new ScheduleResponseFactory();
        return factory.createRecurrenceLimitErrorResponse();
    },

    endBeforeStartError: () => {
        const factory = new ScheduleResponseFactory();
        return factory.createValidationErrorResponse({
            endTime: "End time must be after start time",
        });
    },

    // MSW handlers
    handlers: {
        success: () => new ScheduleResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new ScheduleResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new ScheduleResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const scheduleResponseFactory = new ScheduleResponseFactory();
