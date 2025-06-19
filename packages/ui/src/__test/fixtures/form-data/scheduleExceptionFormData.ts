import { type ScheduleExceptionCreateInput, type ScheduleExceptionUpdateInput } from "@vrooli/shared";

/**
 * Form data fixtures for schedule exception-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * Schedule exception creation form data
 */
export const minimalScheduleExceptionCreateFormInput = {
    originalStartTime: new Date("2025-07-04T09:00:00Z"),
    newStartTime: new Date("2025-07-05T10:00:00Z"),
    newEndTime: new Date("2025-07-05T17:00:00Z"),
    scheduleId: "123456789012345678",
};

export const completeScheduleExceptionCreateFormInput = {
    originalStartTime: new Date("2025-07-04T09:00:00Z"), // Original scheduled time
    newStartTime: new Date("2025-07-05T10:00:00Z"), // Moved to next day
    newEndTime: new Date("2025-07-05T16:00:00Z"), // Shorter duration
    scheduleId: "123456789012345678", // Parent schedule ID
    reason: "Holiday rescheduling", // Form-only field for user context
    notifyParticipants: true, // Form-only field for notification preference
};

/**
 * Schedule exception update form data
 */
export const minimalScheduleExceptionUpdateFormInput = {
    newStartTime: new Date("2025-07-05T11:00:00Z"),
    newEndTime: new Date("2025-07-05T18:00:00Z"),
};

export const completeScheduleExceptionUpdateFormInput = {
    originalStartTime: new Date("2025-12-25T09:00:00Z"), // Updated original time
    newStartTime: new Date("2025-12-26T11:00:00Z"), // Changed time
    newEndTime: new Date("2025-12-26T15:00:00Z"), // Changed duration
    reason: "Christmas holiday adjustment", // Form-only field
    notifyParticipants: false, // Form-only field
};

/**
 * Different schedule exception scenarios for various use cases
 */
export const scheduleExceptionFormVariants = {
    cancellation: {
        originalStartTime: new Date("2025-07-04T09:00:00Z"),
        newStartTime: new Date("2025-07-04T09:00:00Z"), // Same as original (minimal change)
        newEndTime: new Date("2025-07-04T09:01:00Z"), // Very short duration to indicate cancellation
        scheduleId: "123456789012345678",
        reason: "Meeting cancelled",
        notifyParticipants: true,
    },
    rescheduleToNextDay: {
        originalStartTime: new Date("2025-07-04T14:00:00Z"),
        newStartTime: new Date("2025-07-05T14:00:00Z"), // Same time, next day
        newEndTime: new Date("2025-07-05T16:00:00Z"),
        scheduleId: "234567890123456789",
        reason: "Conflict with urgent meeting",
        notifyParticipants: true,
    },
    timeChange: {
        originalStartTime: new Date("2025-07-04T09:00:00Z"),
        newStartTime: new Date("2025-07-04T14:00:00Z"), // Later same day
        newEndTime: new Date("2025-07-04T16:00:00Z"),
        scheduleId: "345678901234567890",
        reason: "Participant availability change",
        notifyParticipants: true,
    },
    durationExtension: {
        originalStartTime: new Date("2025-07-04T09:00:00Z"),
        newStartTime: new Date("2025-07-04T09:00:00Z"), // Same start time
        newEndTime: new Date("2025-07-04T18:00:00Z"), // Extended end time
        scheduleId: "456789012345678901",
        reason: "More time needed for comprehensive discussion",
        notifyParticipants: true,
    },
    durationShortening: {
        originalStartTime: new Date("2025-07-04T09:00:00Z"),
        newStartTime: new Date("2025-07-04T09:00:00Z"), // Same start time
        newEndTime: new Date("2025-07-04T12:00:00Z"), // Shortened end time
        scheduleId: "567890123456789012",
        reason: "Agenda items reduced",
        notifyParticipants: true,
    },
    holidayMove: {
        originalStartTime: new Date("2025-12-25T09:00:00Z"),
        newStartTime: new Date("2025-12-26T10:00:00Z"), // Day after holiday
        newEndTime: new Date("2025-12-26T16:00:00Z"),
        scheduleId: "678901234567890123",
        reason: "Christmas holiday",
        notifyParticipants: true,
    },
    emergencyReschedule: {
        originalStartTime: new Date("2025-07-04T09:00:00Z"),
        newStartTime: new Date("2025-07-04T16:00:00Z"), // Much later same day
        newEndTime: new Date("2025-07-04T17:00:00Z"),
        scheduleId: "789012345678901234",
        reason: "Emergency situation",
        notifyParticipants: true,
    },
};

/**
 * Form validation test cases
 */
export const invalidScheduleExceptionFormInputs = {
    missingOriginalStartTime: {
        // Missing required originalStartTime
        newStartTime: new Date("2025-07-05T10:00:00Z"),
        newEndTime: new Date("2025-07-05T17:00:00Z"),
        scheduleId: "123456789012345678",
    },
    missingNewEndTime: {
        originalStartTime: new Date("2025-07-04T09:00:00Z"),
        newStartTime: new Date("2025-07-05T10:00:00Z"),
        // Missing required newEndTime
        scheduleId: "123456789012345678",
    },
    missingScheduleId: {
        originalStartTime: new Date("2025-07-04T09:00:00Z"),
        newStartTime: new Date("2025-07-05T10:00:00Z"),
        newEndTime: new Date("2025-07-05T17:00:00Z"),
        // Missing required scheduleId
    },
    invalidTimeOrder: {
        originalStartTime: new Date("2025-07-04T09:00:00Z"),
        newStartTime: new Date("2025-07-05T17:00:00Z"),
        newEndTime: new Date("2025-07-05T10:00:00Z"), // Before start time
        scheduleId: "123456789012345678",
    },
    endTimeWithoutStartTime: {
        originalStartTime: new Date("2025-07-04T09:00:00Z"),
        // No newStartTime but has newEndTime
        newEndTime: new Date("2025-07-05T17:00:00Z"),
        scheduleId: "123456789012345678",
    },
    invalidDateTypes: {
        // @ts-expect-error - Testing invalid input
        originalStartTime: "not-a-date",
        // @ts-expect-error - Testing invalid input
        newStartTime: "invalid-date",
        // @ts-expect-error - Testing invalid input
        newEndTime: 12345,
        scheduleId: "123456789012345678",
    },
    invalidScheduleId: {
        originalStartTime: new Date("2025-07-04T09:00:00Z"),
        newStartTime: new Date("2025-07-05T10:00:00Z"),
        newEndTime: new Date("2025-07-05T17:00:00Z"),
        scheduleId: "invalid-id", // Invalid ID format
    },
    emptyScheduleId: {
        originalStartTime: new Date("2025-07-04T09:00:00Z"),
        newStartTime: new Date("2025-07-05T10:00:00Z"),
        newEndTime: new Date("2025-07-05T17:00:00Z"),
        scheduleId: "", // Empty string
    },
    sameStartAndEndTime: {
        originalStartTime: new Date("2025-07-04T09:00:00Z"),
        newStartTime: new Date("2025-07-05T12:00:00Z"),
        newEndTime: new Date("2025-07-05T12:00:00Z"), // Same as start time
        scheduleId: "123456789012345678",
    },
};

/**
 * Form validation states
 */
export const scheduleExceptionFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            originalStartTime: new Date("2025-07-04T09:00:00Z"),
            newStartTime: new Date("2025-07-05T17:00:00Z"),
            newEndTime: new Date("2025-07-05T10:00:00Z"), // Invalid order
            scheduleId: "", // Required but empty
        },
        errors: {
            newEndTime: "End time must be after start time",
            scheduleId: "Schedule is required",
        },
        touched: {
            originalStartTime: true,
            newStartTime: true,
            newEndTime: true,
            scheduleId: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalScheduleExceptionCreateFormInput,
        errors: {},
        touched: {
            originalStartTime: true,
            newStartTime: true,
            newEndTime: true,
            scheduleId: true,
        },
        isValid: true,
        isSubmitting: false,
    },
    submitting: {
        values: completeScheduleExceptionCreateFormInput,
        errors: {},
        touched: {},
        isValid: true,
        isSubmitting: true,
    },
};

/**
 * Helper function to create schedule exception form initial values
 */
export const createScheduleExceptionFormInitialValues = (exceptionData?: Partial<any>) => ({
    originalStartTime: exceptionData?.originalStartTime || new Date(),
    newStartTime: exceptionData?.newStartTime || null,
    newEndTime: exceptionData?.newEndTime || null,
    scheduleId: exceptionData?.schedule?.id || "",
    reason: exceptionData?.reason || "",
    notifyParticipants: exceptionData?.notifyParticipants !== false, // Default to true
    ...exceptionData,
});

/**
 * Helper function to transform form data to API format
 */
export const transformScheduleExceptionFormToApiInput = (
    formData: any,
    isUpdate: boolean = false,
): ScheduleExceptionCreateInput | ScheduleExceptionUpdateInput => {
    if (isUpdate) {
        return {
            id: formData.id,
            originalStartTime: formData.originalStartTime,
            newStartTime: formData.newStartTime,
            newEndTime: formData.newEndTime,
        } as ScheduleExceptionUpdateInput;
    } else {
        return {
            id: formData.id || `temp_${Date.now()}_${Math.random()}`, // Temporary ID for creation
            originalStartTime: formData.originalStartTime,
            newStartTime: formData.newStartTime,
            newEndTime: formData.newEndTime,
            scheduleConnect: formData.scheduleId,
        } as ScheduleExceptionCreateInput;
    }
};

/**
 * Helper function to validate schedule exception time range
 */
export const validateScheduleExceptionTimeRange = (
    newStartTime: Date | null,
    newEndTime: Date | null,
): string | null => {
    if (!newStartTime && newEndTime) {
        return "Start time is required when end time is specified";
    }

    if (newStartTime && newEndTime && newStartTime >= newEndTime) {
        return "End time must be after start time";
    }

    return null;
};

/**
 * Helper function to validate required fields
 */
export const validateScheduleExceptionRequiredFields = (formData: any): Record<string, string> => {
    const errors: Record<string, string> = {};

    if (!formData.originalStartTime) {
        errors.originalStartTime = "Original start time is required";
    }

    if (!formData.newEndTime) {
        errors.newEndTime = "New end time is required";
    }

    if (!formData.scheduleId || formData.scheduleId.trim().length === 0) {
        errors.scheduleId = "Schedule is required";
    }

    return errors;
};

/**
 * Helper function to calculate duration difference
 */
export const calculateDurationChange = (
    originalStart: Date,
    originalEnd: Date,
    newStart: Date,
    newEnd: Date,
): number => {
    const originalDuration = originalEnd.getTime() - originalStart.getTime();
    const newDuration = newEnd.getTime() - newStart.getTime();
    return newDuration - originalDuration; // Positive = extended, negative = shortened
};

/**
 * Helper function to format duration change for display
 */
export const formatDurationChange = (durationChangeMs: number): string => {
    const absChange = Math.abs(durationChangeMs);
    const hours = Math.floor(absChange / (1000 * 60 * 60));
    const minutes = Math.floor((absChange % (1000 * 60 * 60)) / (1000 * 60));

    let result = "";
    if (hours > 0) result += `${hours}h `;
    if (minutes > 0) result += `${minutes}m`;
    if (!result) result = "0m";

    return durationChangeMs >= 0 ? `+${result}` : `-${result}`;
};

/**
 * Mock reason suggestions for UI
 */
export const mockExceptionReasonSuggestions = [
    "Holiday rescheduling",
    "Participant unavailable",
    "Conflict with urgent meeting",
    "Technical difficulties",
    "Weather conditions",
    "Travel delays",
    "Emergency situation",
    "Equipment maintenance",
    "Venue unavailable",
    "Speaker cancellation",
    "Budget constraints",
    "Agenda changes",
];

/**
 * Quick action templates for common exception types
 */
export const scheduleExceptionQuickActions = {
    moveToNextDay: (originalDate: Date) => ({
        originalStartTime: originalDate,
        newStartTime: new Date(originalDate.getTime() + 24 * 60 * 60 * 1000), // +1 day
        newEndTime: new Date(originalDate.getTime() + 24 * 60 * 60 * 1000 + 2 * 60 * 60 * 1000), // +1 day +2 hours
        reason: "Rescheduled to next day",
    }),
    moveToNextWeek: (originalDate: Date) => ({
        originalStartTime: originalDate,
        newStartTime: new Date(originalDate.getTime() + 7 * 24 * 60 * 60 * 1000), // +1 week
        newEndTime: new Date(originalDate.getTime() + 7 * 24 * 60 * 60 * 1000 + 2 * 60 * 60 * 1000), // +1 week +2 hours
        reason: "Rescheduled to next week",
    }),
    extendByOneHour: (originalStart: Date, originalEnd: Date) => ({
        originalStartTime: originalStart,
        newStartTime: originalStart,
        newEndTime: new Date(originalEnd.getTime() + 60 * 60 * 1000), // +1 hour
        reason: "Extended duration",
    }),
    shortenByOneHour: (originalStart: Date, originalEnd: Date) => ({
        originalStartTime: originalStart,
        newStartTime: originalStart,
        newEndTime: new Date(originalEnd.getTime() - 60 * 60 * 1000), // -1 hour
        reason: "Shortened duration",
    }),
};