import { type ScheduleRecurrenceCreateInput, type ScheduleRecurrenceUpdateInput, type ScheduleRecurrenceType } from "@vrooli/shared";

/**
 * Form data fixtures for schedule recurrence-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * Schedule recurrence creation form data
 */
export const minimalScheduleRecurrenceCreateFormInput = {
    recurrenceType: "Daily" as ScheduleRecurrenceType,
    interval: 1,
    duration: 60, // 1 hour in minutes
    scheduleId: "123456789012345678", // Schedule ID to connect to
};

export const completeScheduleRecurrenceCreateFormInput = {
    recurrenceType: "Weekly" as ScheduleRecurrenceType,
    interval: 2, // Every 2 weeks
    duration: 480, // 8 hours in minutes
    dayOfWeek: 3, // Wednesday (1-7, where 1 is Monday)
    dayOfMonth: null,
    month: null,
    endDate: new Date("2025-12-31T23:59:59Z"),
    scheduleId: "123456789012345678", // Schedule ID to connect to
};

/**
 * Schedule recurrence update form data
 */
export const minimalScheduleRecurrenceUpdateFormInput = {
    id: "234567890123456789",
    interval: 3, // Changed from original
};

export const completeScheduleRecurrenceUpdateFormInput = {
    id: "234567890123456789",
    recurrenceType: "Monthly" as ScheduleRecurrenceType,
    interval: 1, // Every month
    duration: 120, // 2 hours in minutes
    dayOfWeek: null,
    dayOfMonth: 15, // 15th of each month
    month: null,
    endDate: new Date("2026-06-30T23:59:59Z"),
};

/**
 * Different recurrence patterns for various use cases
 */
export const scheduleRecurrenceFormVariants = {
    daily: {
        recurrenceType: "Daily" as ScheduleRecurrenceType,
        interval: 1,
        duration: 60, // 1 hour
        dayOfWeek: null,
        dayOfMonth: null,
        month: null,
        endDate: new Date("2025-12-31T23:59:59Z"),
        scheduleId: "345678901234567890",
    },
    weeklyMondayFriday: {
        recurrenceType: "Weekly" as ScheduleRecurrenceType,
        interval: 1,
        duration: 480, // 8 hours
        dayOfWeek: 1, // Monday
        dayOfMonth: null,
        month: null,
        endDate: new Date("2025-06-30T23:59:59Z"),
        scheduleId: "345678901234567890",
    },
    biweekly: {
        recurrenceType: "Weekly" as ScheduleRecurrenceType,
        interval: 2, // Every 2 weeks
        duration: 90, // 1.5 hours
        dayOfWeek: 5, // Friday
        dayOfMonth: null,
        month: null,
        endDate: new Date("2025-12-31T23:59:59Z"),
        scheduleId: "345678901234567890",
    },
    monthlyFirstDay: {
        recurrenceType: "Monthly" as ScheduleRecurrenceType,
        interval: 1,
        duration: 240, // 4 hours
        dayOfWeek: null,
        dayOfMonth: 1, // First day of month
        month: null,
        endDate: new Date("2025-12-31T23:59:59Z"),
        scheduleId: "345678901234567890",
    },
    monthlyMidMonth: {
        recurrenceType: "Monthly" as ScheduleRecurrenceType,
        interval: 1,
        duration: 180, // 3 hours
        dayOfWeek: null,
        dayOfMonth: 15, // 15th of each month
        month: null,
        endDate: new Date("2025-12-31T23:59:59Z"),
        scheduleId: "345678901234567890",
    },
    quarterly: {
        recurrenceType: "Monthly" as ScheduleRecurrenceType,
        interval: 3, // Every 3 months
        duration: 480, // 8 hours
        dayOfWeek: null,
        dayOfMonth: 1, // First day of quarter
        month: null,
        endDate: new Date("2026-12-31T23:59:59Z"),
        scheduleId: "345678901234567890",
    },
    yearly: {
        recurrenceType: "Yearly" as ScheduleRecurrenceType,
        interval: 1,
        duration: 60, // 1 hour
        dayOfWeek: null,
        dayOfMonth: 1,
        month: 1, // January 1st
        endDate: null, // Indefinite
        scheduleId: "345678901234567890",
    },
    workingDays: {
        recurrenceType: "Weekly" as ScheduleRecurrenceType,
        interval: 1,
        duration: 480, // 8 hours
        dayOfWeek: 1, // Monday (Note: Multiple days would need separate recurrences)
        dayOfMonth: null,
        month: null,
        endDate: new Date("2025-12-31T23:59:59Z"),
        scheduleId: "345678901234567890",
    },
    maintenanceWindow: {
        recurrenceType: "Monthly" as ScheduleRecurrenceType,
        interval: 1,
        duration: 240, // 4 hours
        dayOfWeek: null,
        dayOfMonth: 1, // First of each month
        month: null,
        endDate: null, // Indefinite
        scheduleId: "345678901234567890",
    },
};

/**
 * Form validation test cases
 */
export const invalidScheduleRecurrenceFormInputs = {
    missingRecurrenceType: {
        // Missing required recurrenceType
        interval: 1,
        duration: 60,
        scheduleId: "123456789012345678",
    },
    missingInterval: {
        recurrenceType: "Daily" as ScheduleRecurrenceType,
        // Missing required interval
        duration: 60,
        scheduleId: "123456789012345678",
    },
    missingDuration: {
        recurrenceType: "Daily" as ScheduleRecurrenceType,
        interval: 1,
        // Missing required duration
        scheduleId: "123456789012345678",
    },
    missingScheduleId: {
        recurrenceType: "Daily" as ScheduleRecurrenceType,
        interval: 1,
        duration: 60,
        // Missing scheduleId for creation
    },
    invalidInterval: {
        recurrenceType: "Daily" as ScheduleRecurrenceType,
        interval: 0, // Must be >= 1
        duration: 60,
        scheduleId: "123456789012345678",
    },
    invalidDuration: {
        recurrenceType: "Daily" as ScheduleRecurrenceType,
        interval: 1,
        duration: 0, // Must be >= 1
        scheduleId: "123456789012345678",
    },
    invalidDayOfWeek: {
        recurrenceType: "Weekly" as ScheduleRecurrenceType,
        interval: 1,
        duration: 60,
        dayOfWeek: 8, // Must be 1-7
        scheduleId: "123456789012345678",
    },
    invalidDayOfMonth: {
        recurrenceType: "Monthly" as ScheduleRecurrenceType,
        interval: 1,
        duration: 60,
        dayOfMonth: 32, // Must be 1-31
        scheduleId: "123456789012345678",
    },
    invalidRecurrenceType: {
        recurrenceType: "Hourly" as any, // Not a valid enum value
        interval: 1,
        duration: 60,
        scheduleId: "123456789012345678",
    },
    invalidEndDate: {
        recurrenceType: "Daily" as ScheduleRecurrenceType,
        interval: 1,
        duration: 60,
        endDate: "invalid-date", // Should be Date object
        scheduleId: "123456789012345678",
    },
    negativeInterval: {
        recurrenceType: "Daily" as ScheduleRecurrenceType,
        interval: -1, // Must be positive
        duration: 60,
        scheduleId: "123456789012345678",
    },
    negativeDuration: {
        recurrenceType: "Daily" as ScheduleRecurrenceType,
        interval: 1,
        duration: -30, // Must be positive
        scheduleId: "123456789012345678",
    },
    missingIdForUpdate: {
        // Missing required id for update
        recurrenceType: "Daily" as ScheduleRecurrenceType,
        interval: 2,
    },
};

/**
 * Form validation states
 */
export const scheduleRecurrenceFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            recurrenceType: "", // Required but empty
            interval: 0, // Invalid value
            duration: -10, // Invalid negative value
        },
        errors: {
            recurrenceType: "Recurrence type is required",
            interval: "Interval must be at least 1",
            duration: "Duration must be positive",
        },
        touched: {
            recurrenceType: true,
            interval: true,
            duration: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalScheduleRecurrenceCreateFormInput,
        errors: {},
        touched: {
            recurrenceType: true,
            interval: true,
            duration: true,
        },
        isValid: true,
        isSubmitting: false,
    },
    submitting: {
        values: completeScheduleRecurrenceCreateFormInput,
        errors: {},
        touched: {},
        isValid: true,
        isSubmitting: true,
    },
};

/**
 * Helper function to create schedule recurrence form initial values
 */
export const createScheduleRecurrenceFormInitialValues = (recurrenceData?: Partial<any>) => ({
    recurrenceType: recurrenceData?.recurrenceType || "Daily" as ScheduleRecurrenceType,
    interval: recurrenceData?.interval || 1,
    duration: recurrenceData?.duration || 60,
    dayOfWeek: recurrenceData?.dayOfWeek || null,
    dayOfMonth: recurrenceData?.dayOfMonth || null,
    month: recurrenceData?.month || null,
    endDate: recurrenceData?.endDate || null,
    scheduleId: recurrenceData?.schedule?.id || null,
    ...recurrenceData,
});

/**
 * Helper function to transform form data to API format
 */
export const transformScheduleRecurrenceFormToApiInput = (formData: any, isUpdate: boolean = false): ScheduleRecurrenceCreateInput | ScheduleRecurrenceUpdateInput => {
    const baseInput = {
        recurrenceType: formData.recurrenceType,
        interval: formData.interval,
        duration: formData.duration,
        dayOfWeek: formData.dayOfWeek || undefined,
        dayOfMonth: formData.dayOfMonth || undefined,
        month: formData.month || undefined,
        endDate: formData.endDate || undefined,
    };

    if (isUpdate) {
        return {
            ...baseInput,
            id: formData.id,
        } as ScheduleRecurrenceUpdateInput;
    } else {
        return {
            ...baseInput,
            id: formData.id || `temp_${Date.now()}_${Math.random()}`, // Temporary ID for creation
            scheduleConnect: formData.scheduleId,
        } as ScheduleRecurrenceCreateInput;
    }
};

/**
 * Helper function to validate recurrence interval
 */
export const validateRecurrenceInterval = (interval: number): string | null => {
    if (!interval || interval < 1) {
        return "Interval must be at least 1";
    }
    
    if (interval > 999) {
        return "Interval cannot exceed 999";
    }
    
    return null;
};

/**
 * Helper function to validate duration
 */
export const validateDuration = (duration: number): string | null => {
    if (!duration || duration < 1) {
        return "Duration must be at least 1 minute";
    }
    
    if (duration > 10080) { // 7 days in minutes
        return "Duration cannot exceed 7 days";
    }
    
    return null;
};

/**
 * Helper function to validate day of week
 */
export const validateDayOfWeek = (dayOfWeek: number | null, recurrenceType: ScheduleRecurrenceType): string | null => {
    if (recurrenceType === "Weekly" && dayOfWeek !== null) {
        if (dayOfWeek < 1 || dayOfWeek > 7) {
            return "Day of week must be between 1 (Monday) and 7 (Sunday)";
        }
    }
    
    return null;
};

/**
 * Helper function to validate day of month
 */
export const validateDayOfMonth = (dayOfMonth: number | null, recurrenceType: ScheduleRecurrenceType): string | null => {
    if (recurrenceType === "Monthly" && dayOfMonth !== null) {
        if (dayOfMonth < 1 || dayOfMonth > 31) {
            return "Day of month must be between 1 and 31";
        }
    }
    
    return null;
};

/**
 * Mock duration presets (in minutes) for schedule recurrence
 */
export const mockRecurrenceDurationPresets = [
    { label: "15 minutes", value: 15 },
    { label: "30 minutes", value: 30 },
    { label: "1 hour", value: 60 },
    { label: "2 hours", value: 120 },
    { label: "4 hours", value: 240 },
    { label: "8 hours", value: 480 },
    { label: "12 hours", value: 720 },
    { label: "Full day", value: 1440 },
];

/**
 * Mock recurrence type options
 */
export const mockRecurrenceTypeOptions = [
    { label: "Daily", value: "Daily" },
    { label: "Weekly", value: "Weekly" },
    { label: "Monthly", value: "Monthly" },
    { label: "Yearly", value: "Yearly" },
];

/**
 * Mock day of week options (1-7, Monday to Sunday)
 */
export const mockDayOfWeekOptions = [
    { label: "Monday", value: 1 },
    { label: "Tuesday", value: 2 },
    { label: "Wednesday", value: 3 },
    { label: "Thursday", value: 4 },
    { label: "Friday", value: 5 },
    { label: "Saturday", value: 6 },
    { label: "Sunday", value: 7 },
];