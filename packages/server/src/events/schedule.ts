import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { logger } from "./logger";

/**
 * Calculate occurrences of a scheduled event within a given time frame.
 *
 * @param {Prisma.scheduleGetPayload<SelectWrap<Prisma.scheduleSelect>>} schedule - The schedule object.
 * @param {Date} timeframeStart - The start of the time frame as a Date, in UTC.
 * @param {Date} timeframeEnd - The end of the time frame as a Date, in UTC.
 * @return {Array<{ start: Date; end: Date }>} - An array of objects representing the occurrences of the scheduled event.
 */
export const calculateOccurrences = (
    schedule: Prisma.scheduleGetPayload<SelectWrap<Prisma.scheduleSelect>>,
    timeframeStart: Date,
    timeframeEnd: Date
): Array<{ start: Date; end: Date }> => {
    // Make sure that the time frame is no longer than a year, so that we don't overload the server
    const timeframeDuration = timeframeEnd.getTime() - timeframeStart.getTime();
    if (timeframeDuration > 1000 * 60 * 60 * 24 * 365) {
        logger.error('calculateOccurrences time frame to large', { trace: '0432', timeframeStart, timeframeEnd });
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

        // While the current start time is before the recurrence end date, add occurrences
        while (currentStartTime <= endTime) {
            const currentEndTime = new Date(currentStartTime.getTime() + duration);
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
                        const daysUntilNextOccurrence = dayOfWeek - currentDayOfWeek;
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
                        const currentDayOfMonth = currentStartTime.getDate();
                        const daysUntilNextOccurrence = dayOfMonth - currentDayOfMonth;
                        currentStartTime = new Date(currentStartTime.getTime() + daysUntilNextOccurrence * 24 * 60 * 60 * 1000);
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
                        const month = recurrence.month;
                        const dayOfMonth = recurrence.dayOfMonth;
                        const currentMonth = currentStartTime.getMonth();
                        const currentDayOfMonth = currentStartTime.getDate();
                        const monthsUntilNextOccurrence = month - currentMonth;
                        const daysUntilNextOccurrence = dayOfMonth - currentDayOfMonth;
                        currentStartTime = new Date(currentStartTime.getTime() + monthsUntilNextOccurrence * 30 * 24 * 60 * 60 * 1000 + daysUntilNextOccurrence * 24 * 60 * 60 * 1000);
                    }
                    // Otherwise, yearly recurrences are incremented by the interval in years.
                    else {
                        const newYear = currentStartTime.getFullYear() + recurrence.interval;
                        currentStartTime = new Date(currentStartTime.setFullYear(newYear));
                    }
                    break;
            }
            // Break if the recurrence end date is before the current occurrence start time
            if (recurrence.endDate && new Date(recurrence.endDate).getTime() < currentStartTime.getTime()) {
                break;
            }
        }
    }
    // Return occurrences
    return occurrences;
}