import { type Schedule, type ScheduleRecurrence, type ScheduleRecurrenceType, type ScheduleException, type ScheduleEdge, type ScheduleSearchResult } from "@vrooli/shared";
import { minimalUserResponse, completeUserResponse } from "./userResponses.js";

/**
 * API response fixtures for schedules
 * These represent what components receive from API calls
 */

/**
 * Mock schedule recurrence data
 */
const dailyRecurrence: ScheduleRecurrence = {
    __typename: "ScheduleRecurrence",
    id: "recurrence_daily_123456789",
    recurrenceType: "Daily" as ScheduleRecurrenceType,
    interval: 1,
    duration: 3600, // 1 hour in seconds
    dayOfWeek: null,
    dayOfMonth: null,
    month: null,
    endDate: "2024-12-31T23:59:59Z",
    schedule: null as any, // Avoid circular reference
};

const weeklyRecurrence: ScheduleRecurrence = {
    __typename: "ScheduleRecurrence",
    id: "recurrence_weekly_123456789",
    recurrenceType: "Weekly" as ScheduleRecurrenceType,
    interval: 1,
    duration: 7200, // 2 hours in seconds
    dayOfWeek: 1, // Monday
    dayOfMonth: null,
    month: null,
    endDate: null, // No end date
    schedule: null as any, // Avoid circular reference
};

const monthlyRecurrence: ScheduleRecurrence = {
    __typename: "ScheduleRecurrence",
    id: "recurrence_monthly_123456789",
    recurrenceType: "Monthly" as ScheduleRecurrenceType,
    interval: 1,
    duration: 5400, // 1.5 hours in seconds
    dayOfWeek: null,
    dayOfMonth: 15, // 15th of each month
    month: null,
    endDate: "2025-12-31T23:59:59Z",
    schedule: null as any, // Avoid circular reference
};

const yearlyRecurrence: ScheduleRecurrence = {
    __typename: "ScheduleRecurrence",
    id: "recurrence_yearly_123456789",
    recurrenceType: "Yearly" as ScheduleRecurrenceType,
    interval: 1,
    duration: 28800, // 8 hours in seconds
    dayOfWeek: null,
    dayOfMonth: 1,
    month: 1, // January 1st
    endDate: null,
    schedule: null as any, // Avoid circular reference
};

const biWeeklyRecurrence: ScheduleRecurrence = {
    __typename: "ScheduleRecurrence",
    id: "recurrence_biweekly_123456789",
    recurrenceType: "Weekly" as ScheduleRecurrenceType,
    interval: 2, // Every 2 weeks
    duration: 3600, // 1 hour in seconds
    dayOfWeek: 3, // Wednesday
    dayOfMonth: null,
    month: null,
    endDate: "2024-06-30T23:59:59Z",
    schedule: null as any, // Avoid circular reference
};

/**
 * Mock schedule exception data
 */
const holidayException: ScheduleException = {
    __typename: "ScheduleException",
    id: "exception_holiday_123456789",
    originalStartTime: "2024-07-04T14:00:00Z",
    newStartTime: null, // Cancelled for holiday
    newEndTime: null,
    schedule: null as any, // Avoid circular reference
};

const rescheduledException: ScheduleException = {
    __typename: "ScheduleException",
    id: "exception_reschedule_123456789",
    originalStartTime: "2024-03-15T14:00:00Z",
    newStartTime: "2024-03-15T16:00:00Z", // Moved 2 hours later
    newEndTime: "2024-03-15T17:00:00Z",
    schedule: null as any, // Avoid circular reference
};

const extendedException: ScheduleException = {
    __typename: "ScheduleException",
    id: "exception_extended_123456789",
    originalStartTime: "2024-04-10T10:00:00Z",
    newStartTime: "2024-04-10T10:00:00Z", // Same start time
    newEndTime: "2024-04-10T13:00:00Z", // Extended by 1 hour
    schedule: null as any, // Avoid circular reference
};

/**
 * Minimal schedule API response
 */
export const minimalScheduleResponse: Schedule = {
    __typename: "Schedule",
    id: "schedule_123456789012345",
    publicId: "sched_minimal_123456",
    startTime: "2024-03-15T14:00:00Z",
    endTime: "2024-03-15T15:00:00Z",
    timezone: "America/New_York",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    user: minimalUserResponse,
    exceptions: [],
    recurrences: [],
    meetings: [],
    runs: [],
};

/**
 * Complete schedule API response with all fields
 */
export const completeScheduleResponse: Schedule = {
    __typename: "Schedule",
    id: "schedule_987654321098765",
    publicId: "sched_complete_987654",
    startTime: "2024-01-01T09:00:00Z",
    endTime: "2024-12-31T17:00:00Z",
    timezone: "UTC",
    createdAt: "2023-12-01T00:00:00Z",
    updatedAt: "2024-01-20T15:30:00Z",
    user: completeUserResponse,
    exceptions: [holidayException, rescheduledException],
    recurrences: [dailyRecurrence],
    meetings: [], // Would be populated with meeting references
    runs: [], // Would be populated with run references
};

// Set up circular references
completeScheduleResponse.exceptions[0].schedule = completeScheduleResponse;
completeScheduleResponse.exceptions[1].schedule = completeScheduleResponse;
completeScheduleResponse.recurrences[0].schedule = completeScheduleResponse;

/**
 * One-time schedule (no recurrence)
 */
export const oneTimeScheduleResponse: Schedule = {
    __typename: "Schedule",
    id: "schedule_onetime_123456789",
    publicId: "sched_onetime_123456",
    startTime: "2024-04-15T10:00:00Z",
    endTime: "2024-04-15T12:00:00Z",
    timezone: "America/Los_Angeles",
    createdAt: "2024-03-01T00:00:00Z",
    updatedAt: "2024-03-01T00:00:00Z",
    user: completeUserResponse,
    exceptions: [],
    recurrences: [],
    meetings: [],
    runs: [],
};

/**
 * Daily recurring schedule
 */
export const dailyScheduleResponse: Schedule = {
    __typename: "Schedule",
    id: "schedule_daily_123456789",
    publicId: "sched_daily_123456",
    startTime: "2024-01-01T08:00:00Z",
    endTime: "2024-12-31T09:00:00Z",
    timezone: "Europe/London",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-15T00:00:00Z",
    user: minimalUserResponse,
    exceptions: [extendedException],
    recurrences: [{ ...dailyRecurrence, schedule: null as any }],
    meetings: [],
    runs: [],
};

// Set up circular references
dailyScheduleResponse.exceptions[0].schedule = dailyScheduleResponse;
dailyScheduleResponse.recurrences[0].schedule = dailyScheduleResponse;

/**
 * Weekly recurring schedule
 */
export const weeklyScheduleResponse: Schedule = {
    __typename: "Schedule",
    id: "schedule_weekly_123456789",
    publicId: "sched_weekly_123456",
    startTime: "2024-01-01T14:00:00Z",
    endTime: "2024-12-31T16:00:00Z",
    timezone: "America/Chicago",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-02-01T00:00:00Z",
    user: completeUserResponse,
    exceptions: [],
    recurrences: [{ ...weeklyRecurrence, schedule: null as any }],
    meetings: [],
    runs: [],
};

// Set up circular reference
weeklyScheduleResponse.recurrences[0].schedule = weeklyScheduleResponse;

/**
 * Monthly recurring schedule
 */
export const monthlyScheduleResponse: Schedule = {
    __typename: "Schedule",
    id: "schedule_monthly_123456789",
    publicId: "sched_monthly_123456",
    startTime: "2024-01-15T10:00:00Z",
    endTime: "2025-12-15T11:30:00Z",
    timezone: "Asia/Tokyo",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    user: minimalUserResponse,
    exceptions: [rescheduledException],
    recurrences: [{ ...monthlyRecurrence, schedule: null as any }],
    meetings: [],
    runs: [],
};

// Set up circular references
monthlyScheduleResponse.exceptions[0].schedule = monthlyScheduleResponse;
monthlyScheduleResponse.recurrences[0].schedule = monthlyScheduleResponse;

/**
 * Yearly recurring schedule
 */
export const yearlyScheduleResponse: Schedule = {
    __typename: "Schedule",
    id: "schedule_yearly_123456789",
    publicId: "sched_yearly_123456",
    startTime: "2024-01-01T00:00:00Z",
    endTime: "2030-01-01T08:00:00Z",
    timezone: "Australia/Sydney",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    user: completeUserResponse,
    exceptions: [],
    recurrences: [{ ...yearlyRecurrence, schedule: null as any }],
    meetings: [],
    runs: [],
};

// Set up circular reference
yearlyScheduleResponse.recurrences[0].schedule = yearlyScheduleResponse;

/**
 * Complex schedule with multiple recurrences and exceptions
 */
export const complexScheduleResponse: Schedule = {
    __typename: "Schedule",
    id: "schedule_complex_123456789",
    publicId: "sched_complex_123456",
    startTime: "2024-01-01T09:00:00Z",
    endTime: "2024-12-31T17:00:00Z",
    timezone: "America/New_York",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-02-15T10:30:00Z",
    user: completeUserResponse,
    exceptions: [holidayException, rescheduledException, extendedException],
    recurrences: [
        { ...weeklyRecurrence, schedule: null as any },
        { ...biWeeklyRecurrence, schedule: null as any },
    ],
    meetings: [],
    runs: [],
};

// Set up circular references
complexScheduleResponse.exceptions.forEach(exception => {
    exception.schedule = complexScheduleResponse;
});
complexScheduleResponse.recurrences.forEach(recurrence => {
    recurrence.schedule = complexScheduleResponse;
});

/**
 * Bi-weekly schedule
 */
export const biWeeklyScheduleResponse: Schedule = {
    __typename: "Schedule",
    id: "schedule_biweekly_123456789",
    publicId: "sched_biweekly_123456",
    startTime: "2024-01-03T15:00:00Z", // Starting on a Wednesday
    endTime: "2024-06-30T16:00:00Z",
    timezone: "America/Denver",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    user: minimalUserResponse,
    exceptions: [],
    recurrences: [{ ...biWeeklyRecurrence, schedule: null as any }],
    meetings: [],
    runs: [],
};

// Set up circular reference
biWeeklyScheduleResponse.recurrences[0].schedule = biWeeklyScheduleResponse;

/**
 * Long-running schedule (multi-year)
 */
export const longRunningScheduleResponse: Schedule = {
    __typename: "Schedule",
    id: "schedule_longrun_123456789",
    publicId: "sched_longrun_123456",
    startTime: "2024-01-01T12:00:00Z",
    endTime: "2029-12-31T12:00:00Z", // 5 years
    timezone: "UTC",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    user: completeUserResponse,
    exceptions: [],
    recurrences: [{ ...monthlyRecurrence, endDate: null, schedule: null as any }], // No end date
    meetings: [],
    runs: [],
};

// Set up circular reference
longRunningScheduleResponse.recurrences[0].schedule = longRunningScheduleResponse;

/**
 * Past schedule (completed)
 */
export const pastScheduleResponse: Schedule = {
    __typename: "Schedule",
    id: "schedule_past_123456789",
    publicId: "sched_past_123456",
    startTime: "2023-01-01T10:00:00Z",
    endTime: "2023-12-31T11:00:00Z",
    timezone: "Europe/Paris",
    createdAt: "2022-12-01T00:00:00Z",
    updatedAt: "2023-12-31T11:00:00Z",
    user: minimalUserResponse,
    exceptions: [
        {
            ...holidayException,
            id: "exception_past_holiday",
            originalStartTime: "2023-07-04T10:00:00Z",
            schedule: null as any,
        },
    ],
    recurrences: [
        {
            ...weeklyRecurrence,
            id: "recurrence_past_weekly",
            endDate: "2023-12-31T23:59:59Z",
            schedule: null as any,
        },
    ],
    meetings: [],
    runs: [],
};

// Set up circular references
pastScheduleResponse.exceptions[0].schedule = pastScheduleResponse;
pastScheduleResponse.recurrences[0].schedule = pastScheduleResponse;

/**
 * Schedule variant states for testing
 */
export const scheduleResponseVariants = {
    minimal: minimalScheduleResponse,
    complete: completeScheduleResponse,
    oneTime: oneTimeScheduleResponse,
    daily: dailyScheduleResponse,
    weekly: weeklyScheduleResponse,
    monthly: monthlyScheduleResponse,
    yearly: yearlyScheduleResponse,
    complex: complexScheduleResponse,
    biWeekly: biWeeklyScheduleResponse,
    longRunning: longRunningScheduleResponse,
    past: pastScheduleResponse,
    noRecurrence: {
        ...minimalScheduleResponse,
        id: "schedule_norec_123456789",
        publicId: "sched_norec_123456",
        startTime: "2024-05-20T09:00:00Z",
        endTime: "2024-05-20T17:00:00Z",
        timezone: "America/Phoenix",
    },
    withExceptionsOnly: {
        ...minimalScheduleResponse,
        id: "schedule_exceptions_123456",
        publicId: "sched_exceptions_123456",
        startTime: "2024-01-01T08:00:00Z",
        endTime: "2024-12-31T08:00:00Z",
        exceptions: [rescheduledException, extendedException],
        recurrences: [],
    },
    differentTimezone: {
        ...weeklyScheduleResponse,
        id: "schedule_timezone_123456789",
        publicId: "sched_timezone_123456",
        timezone: "Pacific/Auckland",
        startTime: "2024-01-01T01:00:00Z", // Adjusted for timezone
        endTime: "2024-12-31T03:00:00Z",
    },
    cancelled: {
        ...oneTimeScheduleResponse,
        id: "schedule_cancelled_123456",
        publicId: "sched_cancelled_123456",
        exceptions: [
            {
                ...holidayException,
                id: "exception_cancelled_123456",
                originalStartTime: "2024-04-15T10:00:00Z",
                newStartTime: null, // Cancelled
                newEndTime: null,
                schedule: null as any,
            },
        ],
    },
} as const;

// Set up circular references for variants
scheduleResponseVariants.withExceptionsOnly.exceptions.forEach(exception => {
    exception.schedule = scheduleResponseVariants.withExceptionsOnly;
});
scheduleResponseVariants.cancelled.exceptions[0].schedule = scheduleResponseVariants.cancelled;

/**
 * Schedule search response
 */
export const scheduleSearchResponse: ScheduleSearchResult = {
    __typename: "ScheduleSearchResult",
    edges: [
        {
            __typename: "ScheduleEdge",
            cursor: "cursor_1",
            node: scheduleResponseVariants.complete,
        },
        {
            __typename: "ScheduleEdge",
            cursor: "cursor_2",
            node: scheduleResponseVariants.weekly,
        },
        {
            __typename: "ScheduleEdge",
            cursor: "cursor_3",
            node: scheduleResponseVariants.monthly,
        },
        {
            __typename: "ScheduleEdge",
            cursor: "cursor_4",
            node: scheduleResponseVariants.oneTime,
        },
    ],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: true,
        hasPreviousPage: false,
        startCursor: "cursor_1",
        endCursor: "cursor_4",
    },
};

/**
 * Filtered search responses for different schedule types
 */
export const recurringSchedulesSearchResponse: ScheduleSearchResult = {
    __typename: "ScheduleSearchResult",
    edges: [
        {
            __typename: "ScheduleEdge",
            cursor: "cursor_recurring_1",
            node: scheduleResponseVariants.daily,
        },
        {
            __typename: "ScheduleEdge",
            cursor: "cursor_recurring_2",
            node: scheduleResponseVariants.weekly,
        },
        {
            __typename: "ScheduleEdge",
            cursor: "cursor_recurring_3",
            node: scheduleResponseVariants.monthly,
        },
        {
            __typename: "ScheduleEdge",
            cursor: "cursor_recurring_4",
            node: scheduleResponseVariants.yearly,
        },
    ],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: false,
        hasPreviousPage: false,
        startCursor: "cursor_recurring_1",
        endCursor: "cursor_recurring_4",
    },
};

export const oneTimeSchedulesSearchResponse: ScheduleSearchResult = {
    __typename: "ScheduleSearchResult",
    edges: [
        {
            __typename: "ScheduleEdge",
            cursor: "cursor_onetime_1",
            node: scheduleResponseVariants.oneTime,
        },
        {
            __typename: "ScheduleEdge",
            cursor: "cursor_onetime_2",
            node: scheduleResponseVariants.noRecurrence,
        },
    ],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: false,
        hasPreviousPage: false,
        startCursor: "cursor_onetime_1",
        endCursor: "cursor_onetime_2",
    },
};

export const upcomingSchedulesSearchResponse: ScheduleSearchResult = {
    __typename: "ScheduleSearchResult",
    edges: [
        {
            __typename: "ScheduleEdge",
            cursor: "cursor_upcoming_1",
            node: scheduleResponseVariants.oneTime,
        },
        {
            __typename: "ScheduleEdge",
            cursor: "cursor_upcoming_2",
            node: scheduleResponseVariants.weekly,
        },
        {
            __typename: "ScheduleEdge",
            cursor: "cursor_upcoming_3",
            node: scheduleResponseVariants.monthly,
        },
    ],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: true,
        hasPreviousPage: false,
        startCursor: "cursor_upcoming_1",
        endCursor: "cursor_upcoming_3",
    },
};

export const pastSchedulesSearchResponse: ScheduleSearchResult = {
    __typename: "ScheduleSearchResult",
    edges: [
        {
            __typename: "ScheduleEdge",
            cursor: "cursor_past_1",
            node: scheduleResponseVariants.past,
        },
    ],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: false,
        hasPreviousPage: false,
        startCursor: "cursor_past_1",
        endCursor: "cursor_past_1",
    },
};

/**
 * Loading and error states for UI testing
 */
export const scheduleUIStates = {
    loading: null,
    error: {
        code: "SCHEDULE_NOT_FOUND",
        message: "The requested schedule could not be found",
    },
    createError: {
        code: "SCHEDULE_CREATE_FAILED",
        message: "Failed to create schedule. Please try again.",
    },
    updateError: {
        code: "SCHEDULE_UPDATE_FAILED",
        message: "Failed to update schedule. Please try again.",
    },
    deleteError: {
        code: "SCHEDULE_DELETE_FAILED",
        message: "Failed to delete schedule. Please try again.",
    },
    permissionError: {
        code: "SCHEDULE_PERMISSION_DENIED",
        message: "You don't have permission to modify this schedule",
    },
    conflictError: {
        code: "SCHEDULE_CONFLICT",
        message: "This schedule conflicts with another existing schedule",
    },
    invalidRecurrenceError: {
        code: "SCHEDULE_INVALID_RECURRENCE",
        message: "The recurrence pattern is invalid",
    },
    timezoneError: {
        code: "SCHEDULE_INVALID_TIMEZONE",
        message: "The specified timezone is not valid",
    },
    timeRangeError: {
        code: "SCHEDULE_INVALID_TIME_RANGE",
        message: "End time must be after start time",
    },
    empty: {
        edges: [],
        pageInfo: {
            __typename: "PageInfo",
            hasNextPage: false,
            hasPreviousPage: false,
            startCursor: null,
            endCursor: null,
        },
    },
};
