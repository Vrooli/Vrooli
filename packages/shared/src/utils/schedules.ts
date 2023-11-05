import { Schedule, ScheduleRecurrence } from "@local/shared";
import moment from "moment-timezone"; // Native Date objects don't handle time zones well

const ONE_YEAR_IN_MS = 1000 * 60 * 60 * 24 * 365;
const ONE_HOUR_IN_MS = 1000 * 60 * 60;
const ONE_MINUTE_IN_MS = 1000 * 60;


/**
 * Ensure that the given time frame does not exceed a year.
 *
 * @param timeframeStart - The start of the time frame as a Date, in UTC.
 * @param timeframeEnd - The end of the time frame as a Date, in UTC.
 * @returns boolean - Returns true if the timeframe is valid (not longer than a year) and false otherwise.
 */
export const validateTimeFrame = (timeframeStart: Date, timeframeEnd: Date): boolean => {
    if (timeframeStart.getTime() > timeframeEnd.getTime()) {
        console.error("Start date is after end date", { trace: "0232", timeframeStart, timeframeEnd });
        return false;
    }
    const timeframeDuration = timeframeEnd.getTime() - timeframeStart.getTime();
    if (timeframeDuration > ONE_YEAR_IN_MS) {
        console.error("Time frame too large", { trace: "0432", timeframeStart, timeframeEnd });
        return false;
    }
    return true;
};

/**
 * Calculate the next occurrence for a daily recurrence.
 * 
 * @param currentStartTime - The current start time of the occurrence.
 * @param recurrence - The daily recurrence details.
 * @param timeZone - The time zone identifier (e.g., 'America/New_York').
 * @returns The next start time for the daily recurrence.
 */
export const calculateNextDailyOccurrence = (currentStartTime: Date, recurrence: ScheduleRecurrence, timeZone = "UTC"): Date => {
    // Convert the current start time to the desired time zone.
    const currentMoment = moment.tz(currentStartTime, timeZone ?? "UTC");

    // Add the interval to the current date.
    currentMoment.add(recurrence.interval, "days");

    // Ensure time remains consistent across DST boundary
    if (currentMoment.utcOffset() !== moment.tz(currentStartTime, timeZone ?? "UTC").utcOffset()) {
        currentMoment.add(currentMoment.utcOffset() - moment.tz(currentStartTime, timeZone ?? "UTC").utcOffset(), "minutes");
    }

    // Convert the result back to UTC and return.
    return currentMoment.utc().toDate();
};

/**
 * Calculate the next occurrence for a weekly recurrence.
 * 
 * @param currentStartTime - The current start time of the occurrence.
 * @param recurrence - The weekly recurrence details.
 * @param timeZone - The time zone identifier (e.g., 'America/New_York').
 * @returns The start time of the next weekly occurrence.
 */
export const calculateNextWeeklyOccurrence = (currentStartTime: Date, recurrence: ScheduleRecurrence, timeZone = "UTC"): Date => {
    // Convert the current start time to the desired time zone.
    const currentMoment = moment.tz(currentStartTime, timeZone ?? "UTC");

    // If dayOfWeek is set, then the recurrence is only on that day of the week.
    if (recurrence.dayOfWeek !== undefined && recurrence.dayOfWeek !== null) {
        const dayOfWeek = recurrence.dayOfWeek;
        // Calculate the difference in days to the next occurrence.
        const diffDays = (dayOfWeek - currentMoment.day() + 7) % 7 || recurrence.interval * 7;
        currentMoment.add(diffDays, "days");
    } else {
        // If dayOfWeek is not set, then the recurrence happens every week on the same day as currentStartTime.
        currentMoment.add(recurrence.interval, "weeks");
    }

    // Ensure time remains consistent across DST boundary
    if (currentMoment.utcOffset() !== moment.tz(currentStartTime, timeZone ?? "UTC").utcOffset()) {
        currentMoment.add(currentMoment.utcOffset() - moment.tz(currentStartTime, timeZone ?? "UTC").utcOffset(), "minutes");
    }

    // Convert the result back to UTC and return.
    return currentMoment.utc().toDate();
};

/**
 * Calculate the next occurrence for a monthly recurrence.
 * 
 * @param currentStartTime - The current start time of the occurrence.
 * @param recurrence - The monthly recurrence details.
 * @param timeZone - The time zone identifier (e.g., 'America/New_York').
 * @returns The start time of the next monthly occurrence.
 */
export const calculateNextMonthlyOccurrence = (currentStartTime: Date, recurrence: ScheduleRecurrence, timeZone = "UTC"): Date => {
    // Convert the current start time to the desired time zone.
    const currentMoment = moment.tz(currentStartTime, timeZone ?? "UTC");

    const nextOccurrence = currentMoment.clone();
    if (recurrence.dayOfMonth) {
        // Check if the current day of the month is already past the desired dayOfMonth.
        if (currentMoment.date() >= recurrence.dayOfMonth) {
            // Advance by the interval and set to the desired dayOfMonth.
            nextOccurrence.add(recurrence.interval, "months").date(recurrence.dayOfMonth);
        } else {
            // Simply set to the desired dayOfMonth.
            nextOccurrence.date(recurrence.dayOfMonth);
        }
    } else {
        nextOccurrence.add(recurrence.interval, "months");
    }

    // Ensure time remains consistent across DST boundary
    if (nextOccurrence.utcOffset() !== currentMoment.utcOffset()) {
        nextOccurrence.add(nextOccurrence.utcOffset() - currentMoment.utcOffset(), "minutes");
    }

    // Convert the result back to UTC.
    return nextOccurrence.toDate();
};

/**
 * Calculate the next occurrence for a yearly recurrence.
 * 
 * @param currentStartTime - The current start time of the occurrence.
 * @param recurrence - The yearly recurrence details.
 * @param timeZone - The time zone identifier (e.g., 'America/New_York').
 * @returns The start time of the next yearly occurrence.
 */
export const calculateNextYearlyOccurrence = (currentStartTime: Date, recurrence: ScheduleRecurrence, timeZone = "UTC"): Date => {
    const currentMoment = moment.utc(currentStartTime);

    let nextOccurrence: moment.Moment;
    // If month and dayOfMonth are set, calculate the occurrence for the current year.
    if (recurrence.month !== undefined && recurrence.month !== null && recurrence.dayOfMonth !== undefined && recurrence.dayOfMonth !== null) {
        // Temporarily set the next occurrence to this year's target month and day
        nextOccurrence = currentMoment.clone().month(recurrence.month).date(recurrence.dayOfMonth);

        // If this date has already passed, set the next occurrence for next year
        if (nextOccurrence.isSameOrBefore(currentMoment, "day")) {
            nextOccurrence.add(recurrence.interval, "years");
        }
    } else {
        // If no specific month and dayOfMonth, add the interval to the year.
        nextOccurrence = currentMoment.clone().add(recurrence.interval, "years");
    }

    // Convert the result to the desired time zone and return.
    return nextOccurrence.tz(timeZone).toDate();
};

/**
 * Jumps to the first occurrence of the event that is on or after the specified timeframe start.
 *
 * @param scheduleStart - The original start date of the schedule.
 * @param recurrence - The daily recurrence details.
 * @param timeframeStart - The start of the timeframe of interest.
 * @param timeZone - The time zone identifier (e.g., 'Europe/London').
 * @returns The Date object for the first occurrence on or after the timeframe start.
 */
export const jumpToFirstRelevantWeeklyOccurrence = (
    scheduleStart: Date,
    recurrence: ScheduleRecurrence,
    timeframeStart: Date,
    timeZone = "UTC",
): Date => {
    const relevantStart = moment.utc(scheduleStart);
    const timeframeStartMoment = moment.utc(timeframeStart);
    const targetDayOfWeek = (recurrence.dayOfWeek !== null && recurrence.dayOfWeek !== undefined) ? recurrence.dayOfWeek : timeframeStartMoment.day();

    // Calculate weeks between schedule start and timeframe start
    const weeksBetween = timeframeStartMoment.diff(relevantStart, "weeks");

    // Calculate how many intervals have occurred between the timeframe start and the schedule start
    const intervalsPassed = Math.floor(weeksBetween / recurrence.interval);

    // Jump ahead by the number of intervals that have passed
    relevantStart.add(intervalsPassed * recurrence.interval * 7, "days");

    // Adjust to the next relevant dayOfWeek
    let daysToAdd = ((7 - relevantStart.day() + targetDayOfWeek) % 7);
    if (daysToAdd === 0 && relevantStart.isBefore(timeframeStartMoment)) {
        daysToAdd = 7;
    }
    relevantStart.add(daysToAdd, "days");

    // Now find the first occurrence on or after the timeframeStart
    while (relevantStart.isBefore(timeframeStartMoment)) {
        relevantStart.add(recurrence.interval * 7, "days");
    }

    // Convert the result to the desired time zone and return.
    return relevantStart.tz(timeZone).toDate();
};

/**
 * Processes exceptions for a given occurrence of a schedule.
 * An exception can either cancel an occurrence or reschedule it to a new time.
 * 
 * - If the exception cancels the occurrence (no newStartTime and newEndTime provided),
 *   the function returns `null`.
 * - If the exception reschedules the occurrence, the function returns an object
 *   with the rescheduled start and end Date objects.
 * - If the current occurrence does not match any exceptions, the function returns `undefined`.
 * 
 * The function compares the original start time of exceptions with the current occurrence
 * start time. A tolerance of 1 minute is allowed for the comparison to accommodate slight
 * variations in times.
 *
 * @param currentStartTime - The scheduled start time of the current occurrence.
 * @param schedule - The schedule object containing the exceptions to apply.
 * @param timeZone - The IANA time zone string representing the time zone of the schedule.
 * @returns {object|null|undefined}
 *    - An object with `start` and `end` Date properties if the occurrence is rescheduled,
 *    - `null` if the occurrence is cancelled,
 *    - `undefined` if no exceptions apply to the current occurrence.
 */
export const applyExceptions = ( //TODO change to moment instead of Date
    currentStartTime: Date,
    schedule: Schedule,
    timeZone: string,
): { start: Date; end: Date } | null | undefined => {
    // Iterate through the list of exceptions in the schedule
    for (const exception of schedule.exceptions) {
        const exceptionOriginalStartTime = new Date(exception.originalStartTime.toLocaleString("en-US", { timeZone }));

        // Check for exceptions within a 1-minute window of the current start time
        if (Math.abs(exceptionOriginalStartTime.getTime() - currentStartTime.getTime()) < ONE_MINUTE_IN_MS) {
            // The occurrence has an exception
            if (exception.newStartTime || exception.newEndTime) {
                // If there's a new start time, use it, otherwise set it original start time
                const newStart = exception.newStartTime ? new Date(exception.newStartTime) : exceptionOriginalStartTime;
                // If there's a new end time, use it, otherwise set it to 1 hour after the new start time
                const newEnd = exception.newEndTime ? new Date(exception.newEndTime) : new Date(newStart.getTime() + ONE_HOUR_IN_MS);

                return { start: newStart, end: newEnd }; // Return the rescheduled occurrence
            } else {
                return null; // The occurrence is cancelled, return null
            }
        }
    }
    return undefined; // No exceptions apply, return undefined
};

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
    timeframeEnd: Date,
): Array<{ start: Date; end: Date }> => {
    const occurrences: Array<{ start: Date; end: Date }> = [];
    // Make sure that the time frame is no longer than a year, so that we don't overload the server
    if (!validateTimeFrame(timeframeStart, timeframeEnd)) {
        console.error("calculateOccurrences time frame to large", { trace: "0432", timeframeStart, timeframeEnd });
        return occurrences;
    }
    // Convert schedule start and end times to Date objects with the correct timezone
    const startTime = new Date(schedule.startTime.toLocaleString("en-US", { timeZone: schedule.timezone }));
    const endTime = new Date(schedule.endTime.toLocaleString("en-US", { timeZone: schedule.timezone }));
    const duration = endTime.getTime() - startTime.getTime();
    // Calcuate occurrences for each recurrence
    for (const recurrence of schedule.recurrences) {
        let currentStartTime = startTime;

        // While the current start time is before the end of the time frame (and also while we haven't maxed out the loop)
        while (currentStartTime <= timeframeEnd && occurrences.length < 5000) {
            const currentEndTime = new Date(currentStartTime.getTime() + duration);
            const prevStartTime = currentStartTime; // Store the previous start time to compare later

            // Apply exceptions (if any) to the current occurrence
            const exceptionResult = applyExceptions(currentStartTime, schedule, schedule.timezone);
            // If null, the occurrence was canceled
            if (exceptionResult === null) {
                continue;
            }
            // If a start and end time are returned, the occurrence was rescheduled
            else if (exceptionResult) {
                occurrences.push(exceptionResult);
            }
            // Otherwise, there was no exception. Add the unmodified occurrence to the list.
            else {
                occurrences.push({ start: currentStartTime, end: currentEndTime });
            }

            // Move to the next occurrence
            switch (recurrence.recurrenceType) {
                case "Daily":
                    currentStartTime = calculateNextDailyOccurrence(currentStartTime, recurrence);
                    break;
                case "Weekly":
                    currentStartTime = calculateNextWeeklyOccurrence(currentStartTime, recurrence);
                    break;
                case "Monthly":
                    currentStartTime = calculateNextMonthlyOccurrence(currentStartTime, recurrence);
                    break;
                case "Yearly":
                    currentStartTime = calculateNextYearlyOccurrence(currentStartTime, recurrence);
                    break;
            }
            // Break the loop early if the recurrence end date is before the current occurrence start time, 
            // or if the current start time hasn't changed, 
            if ((recurrence.endDate && new Date(recurrence.endDate).getTime() < currentStartTime.getTime()) ||
                currentStartTime.getTime() === prevStartTime.getTime()) {
                break;
            }
        }
    }
    // Return occurrences
    return occurrences;
};
