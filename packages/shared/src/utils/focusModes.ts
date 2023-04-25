import { ActiveFocusMode, FocusMode, FocusModeStopCondition, Schedule } from "@local/shared";

/**
 * Calculate occurrences of a scheduled event within a given time frame.
 *
 * @param schedule - The schedule object.
 * @param timeframeStart - The start of the time frame as a Date, in UTC.
 * @param timeframeEnd - The end of the time frame as a Date, in UTC.
 * @returns An array of objects representing the occurrences of the scheduled event.
 */
export const calculateOccurrences = (
    schedule: Schedule,
    timeframeStart: Date,
    timeframeEnd: Date
): Array<{ start: Date; end: Date }> => {
    // Make sure that the time frame is no longer than a year, so that we don't overload the server
    const timeframeDuration = timeframeEnd.getTime() - timeframeStart.getTime();
    if (timeframeDuration > 1000 * 60 * 60 * 24 * 365) {
        console.error('calculateOccurrences time frame to large', { trace: '0432', timeframeStart, timeframeEnd });
        return [];
    }
    const occurrences: Array<{ start: Date; end: Date }> = [];
    // Convert schedule start and end times to Date objects with the correct timezone
    const startTime = new Date(schedule.startTime.toLocaleString("en-US", { timeZone: schedule.timezone }));
    const endTime = new Date(schedule.endTime.toLocaleString("en-US", { timeZone: schedule.timezone }));
    const duration = endTime.getTime() - startTime.getTime();
    // Calcuate occurrences for each recurrence
    for (const recurrence of schedule.recurrences) {
        // Create field to store the start time as we iterate though the recurrence's period
        let currentStartTime = startTime;

        // While the current start time is before the end of the time frame
        const maxOccurrences = 5000;
        let occurrenceCount = 0;
        while (currentStartTime <= timeframeEnd && occurrences.length < maxOccurrences) {
            const currentEndTime = new Date(currentStartTime.getTime() + duration);
            const prevStartTime = currentStartTime; // Store the previous start time to compare later
            occurrenceCount++;
            // Check if this occurrence is an exception (i.e. cancelled or rescheduled)
            let isException = false;
            for (const exception of schedule.exceptions) {
                // In case times aren't exactly the same, check if exception's originalStartTime is within 1 minute of the current occurrence's start time
                if (Math.abs(exception.originalStartTime.getTime() - currentStartTime.getTime()) < 60000) {
                    isException = true;
                    // If the exception has a new start or end time, then it is a reschedule 
                    // and should be added to the occurrences
                    if (exception.newStartTime || exception.newEndTime) {
                        occurrences.push({
                            start: exception.newStartTime ? new Date(exception.newStartTime.toLocaleString("en-US", { timeZone: schedule.timezone })) : currentStartTime,
                            end: exception.newEndTime ? new Date(exception.newEndTime.toLocaleString("en-US", { timeZone: schedule.timezone })) : currentEndTime,
                        });
                    }
                    break;
                }
            }
            // If the occurrence is not an exception, add it to the occurrences
            if (!isException) {
                occurrences.push({
                    start: currentStartTime,
                    end: currentEndTime,
                });
            }
            // Move to the next occurrence
            switch (recurrence.recurrenceType) {
                case 'Daily':
                    // Daily recurrences are incremented by the interval in days
                    currentStartTime = new Date(currentStartTime.getTime() + recurrence.interval * 24 * 60 * 60 * 1000);
                    break;
                case 'Weekly':
                    // If dayOfWeek is set, then the recurrence is only on that day of the week.
                    if (recurrence.dayOfWeek) {
                        const dayOfWeek = recurrence.dayOfWeek;
                        const currentDayOfWeek = currentStartTime.getDay();
                        const daysUntilNextOccurrence = (dayOfWeek - currentDayOfWeek + 7) % 7 || recurrence.interval * 7;
                        currentStartTime = new Date(currentStartTime.getTime() + daysUntilNextOccurrence * 24 * 60 * 60 * 1000);
                    }
                    // Otherwise, weekly recurrences are incremented by the interval in weeks. 
                    else {
                        currentStartTime = new Date(currentStartTime.getTime() + recurrence.interval * 7 * 24 * 60 * 60 * 1000);
                    }
                    break;
                case 'Monthly':
                    // If dayOfMonth is set, then the recurrence is only on that day of the month.
                    if (recurrence.dayOfMonth) {
                        const dayOfMonth = recurrence.dayOfMonth;
                        const currentMonth = currentStartTime.getMonth();
                        const nextOccurrence = new Date(currentStartTime);
                        nextOccurrence.setMonth(currentMonth + recurrence.interval);
                        nextOccurrence.setDate(dayOfMonth);
                        // If the next occurrence is in the past, increment by another interval
                        if (nextOccurrence <= currentStartTime) {
                            nextOccurrence.setMonth(currentMonth + 2 * recurrence.interval);
                        }
                        currentStartTime = nextOccurrence;
                    }
                    // Otherwise, monthly recurrences are incremented by the interval in months.
                    else {
                        const newMonth = currentStartTime.getMonth() + recurrence.interval;
                        currentStartTime = new Date(currentStartTime.setMonth(newMonth));
                    }
                    break;
                case 'Yearly':
                    // If month and dayOfMonth are set, then the recurrence is only on that day of the month in that month.
                    if (recurrence.month && recurrence.dayOfMonth) {
                        const nextOccurrence = new Date(currentStartTime);
                        nextOccurrence.setFullYear(currentStartTime.getFullYear() + recurrence.interval);
                        nextOccurrence.setMonth(recurrence.month);
                        nextOccurrence.setDate(recurrence.dayOfMonth);

                        // If the next occurrence is in the past, increment by another interval
                        if (nextOccurrence <= currentStartTime) {
                            nextOccurrence.setFullYear(currentStartTime.getFullYear() + 2 * recurrence.interval);
                        }
                        currentStartTime = nextOccurrence;
                    }
                    // Otherwise, yearly recurrences are incremented by the interval in years.
                    else {
                        const newYear = currentStartTime.getFullYear() + recurrence.interval;
                        currentStartTime = new Date(currentStartTime.setFullYear(newYear));
                    }
                    break;
            }
            // Break if the recurrence end date is before the current occurrence start time, 
            // or if the current start time hasn't changed, 
            if ((recurrence.endDate && new Date(recurrence.endDate).getTime() < currentStartTime.getTime()) ||
                currentStartTime.getTime() === prevStartTime.getTime()) {
                break;
            }
        }
    }
    // Return occurrences
    return occurrences;
}

/**
 * Finds which focus modes are active for a given time frame, based on the focus 
 * modes' schedules. This can be overridden by manually setting the focus mode
 * @param focusModes The user's focus modes.
 * @param startDate The start of the time frame.
 * @param endDate The end of the time frame.
 * @returns An array of focus modes that are active for the given time frame.
 */
export const getFocusModesFromOccurrences = (
    focusModes: FocusMode[],
    startDate: Date,
    endDate: Date
): FocusMode[] => {
    // Get the schedules associated with each focus mode
    const schedules = focusModes.map((focusMode) => focusMode.schedule);
    // Get the occurrences for each schedule
    const occurrences = schedules.map((schedule) => schedule ? calculateOccurrences(schedule as Schedule, startDate, endDate) : []);
    // Get the focus modes that have occurrences in the time frame
    const activeFocusModes = focusModes.filter((focusMode, index) => occurrences[index].length > 0);
    return activeFocusModes;
}

/**
 * Finds the actual active focus mode
 * @param currentlyActive The focus mode that is currently active, which 
 * may be expired
 * @param focusModes The user's focus modes
 * @returns The focus mode that should be active
 */
export const getActiveFocusMode = (
    currentlyActive: ActiveFocusMode | null | undefined,
    focusModes: FocusMode[]
): ActiveFocusMode | null => {
    // If there is an active focus mode
    if (currentlyActive) {
        // If the focus mode must be manually switched, then return it
        if (currentlyActive.stopCondition === FocusModeStopCondition.Never) {
            return currentlyActive;
        }
        // If the focus mode is set to stop at a certain time, then check if that time has passed
        else if (currentlyActive.stopCondition === FocusModeStopCondition.AfterStopTime && currentlyActive.stopTime) {
            if (new Date(currentlyActive.stopTime) > new Date()) {
                return currentlyActive;
            }
        }
    }
    // Get the focus modes that are active according to their schedules
    const activeFocusModes = getFocusModesFromOccurrences(focusModes, new Date(), new Date(new Date().getTime() + 1000 * 60)); // Add 1 minute buffer just in case
    // If there are no active focus modes and currently active is set to NextBegins, then return 
    // currently active
    if (activeFocusModes.length === 0 && currentlyActive && currentlyActive.stopCondition === FocusModeStopCondition.NextBegins) {
        return currentlyActive;
    }
    // If there is at least one active focus mode, then return the first one. 
    // Otherwise, return null
    if (activeFocusModes.length > 0) {
        return {
            __typename: 'ActiveFocusMode',
            mode: activeFocusModes[0],
            stopCondition: FocusModeStopCondition.Automatic,
        }
    }
    // If there is at least one focus mode in the list, then return the first one.
    // Otherwise, return null
    return focusModes.length > 0 ? {
        __typename: 'ActiveFocusMode',
        mode: focusModes[0],
        stopCondition: FocusModeStopCondition.Automatic,
    } : null;
}