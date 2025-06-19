import { type ScheduleCreateInput, type ScheduleUpdateInput, type ScheduleRecurrenceType } from "@vrooli/shared";

/**
 * Form data fixtures for schedule-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * Schedule creation form data
 */
export const minimalScheduleCreateFormInput = {
    timezone: "America/New_York",
    startTime: new Date("2025-01-01T09:00:00Z"),
    endTime: null,
    recurrences: [],
    exceptions: [],
    meetingId: null,
    runId: null,
};

export const completeScheduleCreateFormInput = {
    timezone: "America/New_York",
    startTime: new Date("2025-01-01T09:00:00Z"),
    endTime: new Date("2025-12-31T17:00:00Z"),
    meetingId: "123456789012345678", // Meeting ID if scheduling for meeting
    runId: null,
    recurrences: [
        {
            recurrenceType: "Daily" as ScheduleRecurrenceType,
            interval: 1,
            duration: 480, // 8 hours in minutes
            dayOfWeek: null,
            dayOfMonth: null,
            month: null,
            endDate: new Date("2025-12-31T23:59:59Z"),
        },
    ],
    exceptions: [
        {
            originalStartTime: new Date("2025-07-04T09:00:00Z"),
            newStartTime: new Date("2025-07-04T09:00:00Z"),
            newEndTime: new Date("2025-07-04T17:00:00Z"),
        },
        {
            originalStartTime: new Date("2025-12-25T09:00:00Z"),
            newStartTime: new Date("2025-12-26T10:00:00Z"), // Moved to next day
            newEndTime: new Date("2025-12-26T16:00:00Z"),
        },
    ],
};

/**
 * Schedule update form data
 */
export const minimalScheduleUpdateFormInput = {
    timezone: "Europe/London",
    startTime: new Date("2025-02-01T10:00:00Z"),
    endTime: new Date("2025-11-30T16:00:00Z"),
};

export const completeScheduleUpdateFormInput = {
    timezone: "Europe/London",
    startTime: new Date("2025-02-01T10:00:00Z"),
    endTime: new Date("2025-11-30T16:00:00Z"),
    recurrencesToCreate: [
        {
            recurrenceType: "Weekly" as ScheduleRecurrenceType,
            interval: 2,
            duration: 480, // 8 hours in minutes
            dayOfWeek: 1, // Monday
            dayOfMonth: null,
            month: null,
            endDate: new Date("2025-06-30T23:59:59Z"),
        },
    ],
    recurrencesToUpdate: [
        {
            id: "123456789012345679",
            interval: 2, // Changed from daily to every other day
            duration: 600, // Updated duration
        },
    ],
    recurrencesToDelete: ["123456789012345680"],
    exceptionsToCreate: [
        {
            originalStartTime: new Date("2025-05-01T10:00:00Z"),
            newStartTime: new Date("2025-05-01T10:00:00Z"),
            newEndTime: new Date("2025-05-01T16:00:00Z"),
        },
    ],
    exceptionsToUpdate: [
        {
            id: "123456789012345681",
            newStartTime: new Date("2025-07-05T11:00:00Z"), // Changed time
            newEndTime: new Date("2025-07-05T18:00:00Z"),
        },
    ],
    exceptionsToDelete: ["123456789012345682"],
};

/**
 * Different schedule types for various use cases
 */
export const scheduleFormVariants = {
    meeting: {
        timezone: "America/New_York",
        startTime: new Date("2025-01-15T14:00:00Z"),
        endTime: new Date("2025-01-15T15:00:00Z"),
        meetingId: "234567890123456789",
        runId: null,
        recurrences: [
            {
                recurrenceType: "Weekly" as ScheduleRecurrenceType,
                interval: 1,
                duration: 60, // 1 hour
                dayOfWeek: 2, // Tuesday
                dayOfMonth: null,
                month: null,
                endDate: new Date("2025-06-30T23:59:59Z"),
            },
        ],
        exceptions: [],
    },
    runExecution: {
        timezone: "UTC",
        startTime: new Date("2025-01-01T00:00:00Z"),
        endTime: null, // Open-ended execution
        meetingId: null,
        runId: "345678901234567890",
        recurrences: [
            {
                recurrenceType: "Daily" as ScheduleRecurrenceType,
                interval: 1,
                duration: 1440, // 24 hours (for long-running processes)
                dayOfWeek: null,
                dayOfMonth: null,
                month: null,
                endDate: null, // Indefinite
            },
        ],
        exceptions: [],
    },
    monthlyReport: {
        timezone: "America/New_York",
        startTime: new Date("2025-01-01T09:00:00Z"),
        endTime: new Date("2025-01-01T17:00:00Z"),
        meetingId: null,
        runId: "456789012345678901",
        recurrences: [
            {
                recurrenceType: "Monthly" as ScheduleRecurrenceType,
                interval: 1,
                duration: 480, // 8 hours
                dayOfWeek: null,
                dayOfMonth: 1, // First day of month
                month: null,
                endDate: new Date("2025-12-31T23:59:59Z"),
            },
        ],
        exceptions: [
            {
                originalStartTime: new Date("2025-07-01T09:00:00Z"),
                newStartTime: new Date("2025-07-02T09:00:00Z"), // Holiday moved to next day
                newEndTime: new Date("2025-07-02T17:00:00Z"),
            },
        ],
    },
    yearlyMaintenance: {
        timezone: "UTC",
        startTime: new Date("2025-01-01T02:00:00Z"), // Off-hours
        endTime: new Date("2025-01-01T06:00:00Z"),
        meetingId: null,
        runId: "567890123456789012",
        recurrences: [
            {
                recurrenceType: "Yearly" as ScheduleRecurrenceType,
                interval: 1,
                duration: 240, // 4 hours
                dayOfWeek: null,
                dayOfMonth: 1,
                month: 1, // January 1st
                endDate: null, // Indefinite
            },
        ],
        exceptions: [],
    },
};

/**
 * Recurrence pattern form data
 */
export const recurrencePatterns = {
    daily: {
        recurrenceType: "Daily" as ScheduleRecurrenceType,
        interval: 1,
        duration: 60, // 1 hour
        dayOfWeek: null,
        dayOfMonth: null,
        month: null,
        endDate: new Date("2025-12-31T23:59:59Z"),
    },
    weeklyMondayWednesdayFriday: {
        recurrenceType: "Weekly" as ScheduleRecurrenceType,
        interval: 1,
        duration: 120, // 2 hours
        dayOfWeek: [1, 3, 5], // Monday, Wednesday, Friday (Note: This might need special handling)
        dayOfMonth: null,
        month: null,
        endDate: new Date("2025-06-30T23:59:59Z"),
    },
    biweekly: {
        recurrenceType: "Weekly" as ScheduleRecurrenceType,
        interval: 2,
        duration: 90, // 1.5 hours
        dayOfWeek: 3, // Wednesday
        dayOfMonth: null,
        month: null,
        endDate: new Date("2025-12-31T23:59:59Z"),
    },
    monthlyFirstMonday: {
        recurrenceType: "Monthly" as ScheduleRecurrenceType,
        interval: 1,
        duration: 180, // 3 hours
        dayOfWeek: 1, // Monday
        dayOfMonth: null, // First occurrence
        month: null,
        endDate: new Date("2025-12-31T23:59:59Z"),
    },
    quarterly: {
        recurrenceType: "Monthly" as ScheduleRecurrenceType,
        interval: 3, // Every 3 months
        duration: 480, // 8 hours
        dayOfWeek: null,
        dayOfMonth: 15, // 15th of the month
        month: null,
        endDate: new Date("2026-12-31T23:59:59Z"),
    },
};

/**
 * Form validation test cases
 */
export const invalidScheduleFormInputs = {
    missingTimezone: {
        // Missing required timezone
        startTime: new Date("2025-01-01T09:00:00Z"),
        endTime: new Date("2025-01-01T17:00:00Z"),
        recurrences: [],
        exceptions: [],
    },
    missingStartTime: {
        timezone: "America/New_York",
        // Missing startTime when endTime is provided
        endTime: new Date("2025-01-01T17:00:00Z"),
        recurrences: [],
        exceptions: [],
    },
    invalidTimeOrder: {
        timezone: "America/New_York",
        startTime: new Date("2025-01-01T17:00:00Z"),
        endTime: new Date("2025-01-01T09:00:00Z"), // Before start time
        recurrences: [],
        exceptions: [],
    },
    invalidTimezone: {
        timezone: "", // Too short
        startTime: new Date("2025-01-01T09:00:00Z"),
        recurrences: [],
        exceptions: [],
    },
    invalidRecurrence: {
        timezone: "America/New_York",
        startTime: new Date("2025-01-01T09:00:00Z"),
        recurrences: [
            {
                // Missing required recurrenceType
                interval: 1,
                duration: 60,
                dayOfWeek: null,
                dayOfMonth: null,
                month: null,
                endDate: new Date("2025-12-31T23:59:59Z"),
            },
        ],
        exceptions: [],
    },
    invalidException: {
        timezone: "America/New_York",
        startTime: new Date("2025-01-01T09:00:00Z"),
        recurrences: [],
        exceptions: [
            {
                // Missing required originalStartTime
                newStartTime: new Date("2025-07-05T10:00:00Z"),
                newEndTime: new Date("2025-07-05T17:00:00Z"),
            },
        ],
    },
    conflictingConnections: {
        timezone: "America/New_York",
        startTime: new Date("2025-01-01T09:00:00Z"),
        meetingId: "123456789012345678",
        runId: "234567890123456789", // Can't have both
        recurrences: [],
        exceptions: [],
    },
};

/**
 * Form validation states
 */
export const scheduleFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            timezone: "", // Required but empty
            startTime: new Date("2025-01-01T17:00:00Z"),
            endTime: new Date("2025-01-01T09:00:00Z"), // Invalid order
        },
        errors: {
            timezone: "Timezone is required",
            endTime: "End time must be after start time",
        },
        touched: {
            timezone: true,
            startTime: true,
            endTime: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalScheduleCreateFormInput,
        errors: {},
        touched: {
            timezone: true,
            startTime: true,
        },
        isValid: true,
        isSubmitting: false,
    },
    submitting: {
        values: completeScheduleCreateFormInput,
        errors: {},
        touched: {},
        isValid: true,
        isSubmitting: true,
    },
};

/**
 * Helper function to create schedule form initial values
 */
export const createScheduleFormInitialValues = (scheduleData?: Partial<any>) => ({
    timezone: scheduleData?.timezone || "America/New_York",
    startTime: scheduleData?.startTime || null,
    endTime: scheduleData?.endTime || null,
    meetingId: scheduleData?.meeting?.id || null,
    runId: scheduleData?.run?.id || null,
    recurrences: scheduleData?.recurrences || [],
    exceptions: scheduleData?.exceptions || [],
    ...scheduleData,
});

/**
 * Helper function to transform form data to API format
 */
export const transformScheduleFormToApiInput = (formData: any, isUpdate: boolean = false): ScheduleCreateInput | ScheduleUpdateInput => {
    const baseInput = {
        timezone: formData.timezone,
        startTime: formData.startTime,
        endTime: formData.endTime || undefined,
    };

    if (isUpdate) {
        return {
            ...baseInput,
            id: formData.id,
            exceptionsCreate: formData.exceptionsToCreate?.map((exception: any) => ({
                id: `temp_${Date.now()}_${Math.random()}`, // Temporary ID for creation
                originalStartTime: exception.originalStartTime,
                newStartTime: exception.newStartTime,
                newEndTime: exception.newEndTime,
                scheduleConnect: formData.id,
            })),
            exceptionsUpdate: formData.exceptionsToUpdate?.map((exception: any) => ({
                id: exception.id,
                newStartTime: exception.newStartTime,
                newEndTime: exception.newEndTime,
            })),
            exceptionsDelete: formData.exceptionsToDelete || [],
            recurrencesCreate: formData.recurrencesToCreate?.map((recurrence: any) => ({
                id: `temp_${Date.now()}_${Math.random()}`, // Temporary ID for creation
                recurrenceType: recurrence.recurrenceType,
                interval: recurrence.interval,
                duration: recurrence.duration,
                dayOfWeek: recurrence.dayOfWeek,
                dayOfMonth: recurrence.dayOfMonth,
                month: recurrence.month,
                endDate: recurrence.endDate,
                scheduleConnect: formData.id,
            })),
            recurrencesUpdate: formData.recurrencesToUpdate?.map((recurrence: any) => ({
                id: recurrence.id,
                recurrenceType: recurrence.recurrenceType,
                interval: recurrence.interval,
                duration: recurrence.duration,
                dayOfWeek: recurrence.dayOfWeek,
                dayOfMonth: recurrence.dayOfMonth,
                month: recurrence.month,
                endDate: recurrence.endDate,
            })),
            recurrencesDelete: formData.recurrencesToDelete || [],
        } as ScheduleUpdateInput;
    } else {
        return {
            ...baseInput,
            id: formData.id || `temp_${Date.now()}_${Math.random()}`, // Temporary ID for creation
            meetingConnect: formData.meetingId || undefined,
            runConnect: formData.runId || undefined,
            exceptionsCreate: formData.exceptions?.map((exception: any) => ({
                id: `temp_${Date.now()}_${Math.random()}`, // Temporary ID for creation
                originalStartTime: exception.originalStartTime,
                newStartTime: exception.newStartTime,
                newEndTime: exception.newEndTime,
                scheduleConnect: formData.id || `temp_${Date.now()}_${Math.random()}`,
            })),
            recurrencesCreate: formData.recurrences?.map((recurrence: any) => ({
                id: `temp_${Date.now()}_${Math.random()}`, // Temporary ID for creation
                recurrenceType: recurrence.recurrenceType,
                interval: recurrence.interval,
                duration: recurrence.duration,
                dayOfWeek: recurrence.dayOfWeek,
                dayOfMonth: recurrence.dayOfMonth,
                month: recurrence.month,
                endDate: recurrence.endDate,
                scheduleConnect: formData.id || `temp_${Date.now()}_${Math.random()}`,
            })),
        } as ScheduleCreateInput;
    }
};

/**
 * Helper function to validate schedule time range
 */
export const validateScheduleTimeRange = (startTime: Date | null, endTime: Date | null): string | null => {
    if (!startTime && endTime) {
        return "Start time is required when end time is specified";
    }
    
    if (startTime && endTime && startTime >= endTime) {
        return "End time must be after start time";
    }
    
    return null;
};

/**
 * Helper function to validate timezone
 */
export const validateTimezone = (timezone: string): string | null => {
    if (!timezone || timezone.trim().length === 0) {
        return "Timezone is required";
    }
    
    if (timezone.length > 64) {
        return "Timezone cannot exceed 64 characters";
    }
    
    return null;
};

/**
 * Mock timezone suggestions
 */
export const mockTimezoneSuggestions = [
    { value: "America/New_York", label: "Eastern Time (EST/EDT)" },
    { value: "America/Chicago", label: "Central Time (CST/CDT)" },
    { value: "America/Denver", label: "Mountain Time (MST/MDT)" },
    { value: "America/Los_Angeles", label: "Pacific Time (PST/PDT)" },
    { value: "UTC", label: "Coordinated Universal Time (UTC)" },
    { value: "Europe/London", label: "Greenwich Mean Time (GMT/BST)" },
    { value: "Europe/Paris", label: "Central European Time (CET/CEST)" },
    { value: "Asia/Tokyo", label: "Japan Standard Time (JST)" },
    { value: "Asia/Shanghai", label: "China Standard Time (CST)" },
    { value: "Australia/Sydney", label: "Australian Eastern Time (AEST/AEDT)" },
];

/**
 * Mock duration presets (in minutes)
 */
export const mockDurationPresets = [
    { label: "15 minutes", value: 15 },
    { label: "30 minutes", value: 30 },
    { label: "1 hour", value: 60 },
    { label: "2 hours", value: 120 },
    { label: "4 hours", value: 240 },
    { label: "8 hours", value: 480 },
    { label: "Full day", value: 1440 },
];