import { Moment } from "moment-timezone";
import { type Schedule, type ScheduleRecurrence } from "../api/types";
import { HOURS_1_MS, MINUTES_1_MS, YEARS_1_MS } from "../consts/numbers";

const DAYS_IN_WEEK = 7;
const OCCURRENCE_LOOP_LIMIT = 5000;

// Native Date objects don't handle time zones well, so we use moment-timezone instead.
// This must be loaded asynchronously so the UI can code-split it.
let moment;
async function ensureMomentTimezone() {
    if (!moment) {
        if (typeof window !== "undefined") {
            // Client-side: Use dynamic import
            moment = (await import("moment-timezone")).default;
        } else {
            // Server-side: Use regular require
            moment = require("moment-timezone");
        }
    }
}

/**
 * Ensure that the given time frame does not exceed a year.
 *
 * @param timeframeStart - The start of the time frame as a Date, in UTC.
 * @param timeframeEnd - The end of the time frame as a Date, in UTC.
 * @returns boolean - Returns true if the timeframe is valid (not longer than a year) and false otherwise.
 */
export function validateTimeFrame(timeframeStart: Date, timeframeEnd: Date): boolean {
    if (timeframeStart.getTime() > timeframeEnd.getTime()) {
        console.error("Start date is after end date", { trace: "0232", timeframeStart, timeframeEnd });
        return false;
    }
    const timeframeDuration = timeframeEnd.getTime() - timeframeStart.getTime();
    if (timeframeDuration > YEARS_1_MS) {
        console.error("Time frame too large", { trace: "0432", timeframeStart, timeframeEnd });
        return false;
    }
    return true;
}

/**
 * Calculate the next occurrence for a daily recurrence.
 * 
 * @param currentStartTime - The current start time of the occurrence.
 * @param recurrence - The daily recurrence details.
 * @param timeZone - The time zone identifier (e.g., 'America/New_York').
 * @returns The next start time for the daily recurrence.
 */
export async function calculateNextDailyOccurrence(currentStartTime: Date, recurrence: ScheduleRecurrence, timeZone = "UTC"): Promise<Date> {
    await ensureMomentTimezone();

    // Keep the current start time in UTC.
    const currentMoment = moment.utc(currentStartTime);

    // Add the interval to the current date.
    currentMoment.add(recurrence.interval, "days");

    // Convert the result to the desired time zone and return.
    return currentMoment.tz(timeZone).toDate();
}

/**
 * Calculate the next occurrence for a weekly recurrence.
 * 
 * @param currentStartTime - The current start time of the occurrence.
 * @param recurrence - The weekly recurrence details.
 * @param timeZone - The time zone identifier (e.g., 'America/New_York').
 * @returns The start time of the next weekly occurrence.
 */
export async function calculateNextWeeklyOccurrence(currentStartTime: Date, recurrence: ScheduleRecurrence, timeZone = "UTC"): Promise<Date> {
    await ensureMomentTimezone();

    // Keep the current start time in UTC.
    const currentMoment = moment.utc(currentStartTime);

    // If dayOfWeek is set, then the recurrence is only on that day of the week.
    if (recurrence.dayOfWeek !== undefined && recurrence.dayOfWeek !== null) {
        const dayOfWeek = recurrence.dayOfWeek;
        // Calculate the difference in days to the next occurrence.
        const diffDays = (dayOfWeek - currentMoment.day() + DAYS_IN_WEEK) % DAYS_IN_WEEK || recurrence.interval * DAYS_IN_WEEK;
        currentMoment.add(diffDays, "days");
    } else {
        // If dayOfWeek is not set, then the recurrence happens every week on the same day as currentStartTime.
        currentMoment.add(recurrence.interval, "weeks");
    }

    // Convert the result to the desired time zone and return.
    return currentMoment.tz(timeZone).toDate();
}

/**
 * Calculate the next occurrence for a monthly recurrence.
 * 
 * @param currentStartTime - The current start time of the occurrence.
 * @param recurrence - The monthly recurrence details.
 * @param timeZone - The time zone identifier (e.g., 'America/New_York').
 * @returns The start time of the next monthly occurrence.
 */
export async function calculateNextMonthlyOccurrence(currentStartTime: Date, recurrence: ScheduleRecurrence, timeZone = "UTC"): Promise<Date> {
    await ensureMomentTimezone();

    // Convert the current start time to the desired time zone.
    const currentMoment = moment.utc(currentStartTime);

    const nextOccurrence = currentMoment.clone();
    if (recurrence.dayOfMonth) {
        // Check if the current day of the month is already past the desired dayOfMonth.
        if (currentMoment.date() >= recurrence.dayOfMonth) {
            // Advance by the interval
            nextOccurrence.add(recurrence.interval, "months");
        }
        // Set to the desired dayOfMonth or the last day of the next month if it doesn't exist
        nextOccurrence.date(Math.min(recurrence.dayOfMonth, nextOccurrence.daysInMonth()));
    } else {
        // If there is no dayOfMonth specified, just add the interval months
        nextOccurrence.add(recurrence.interval, "months");
    }

    // Convert the result to the desired time zone and return.
    return nextOccurrence.tz(timeZone).toDate();
}

/**
 * Calculate the next occurrence for a yearly recurrence.
 * 
 * @param currentStartTime - The current start time of the occurrence.
 * @param recurrence - The yearly recurrence details.
 * @param timeZone - The time zone identifier (e.g., 'America/New_York').
 * @returns The start time of the next yearly occurrence.
 */
export async function calculateNextYearlyOccurrence(currentStartTime: Date, recurrence: ScheduleRecurrence, timeZone = "UTC"): Promise<Date> {
    await ensureMomentTimezone();

    // Keep the current start time in UTC.
    const currentMoment = moment.utc(currentStartTime);

    // Capture the time components to apply them later
    const timeComponents = {
        hours: currentMoment.hours(),
        minutes: currentMoment.minutes(),
        seconds: currentMoment.seconds(),
    };

    let nextOccurrence: Moment;
    // If month and dayOfMonth are set, calculate the occurrence for the current year.
    if (recurrence.month !== null && recurrence.month !== undefined && recurrence.dayOfMonth !== null && recurrence.dayOfMonth !== undefined) {
        // Create a moment for the potential next occurrence this year, keeping the time
        nextOccurrence = currentMoment.clone().month(recurrence.month).startOf("month").add(timeComponents);

        // Set to the desired dayOfMonth or the last day of the month if it doesn't exist
        const daysInMonth = nextOccurrence.daysInMonth();
        nextOccurrence.date(Math.min(recurrence.dayOfMonth, daysInMonth));

        // If the date has already passed, then set the next occurrence to next year's month
        if (nextOccurrence.isSameOrBefore(currentMoment, "day")) {
            // Add years first, then set month and day to avoid rolling over
            nextOccurrence.add(recurrence.interval, "years").month(recurrence.month).startOf("month").add(timeComponents);
            const daysInNextOccurrenceMonth = nextOccurrence.daysInMonth();
            nextOccurrence.date(Math.min(recurrence.dayOfMonth, daysInNextOccurrenceMonth));
        }
    } else {
        // If no specific month and dayOfMonth, add the interval to the year while preserving the original date and time.
        nextOccurrence = currentMoment.clone().add(recurrence.interval, "years");
    }

    // Convert the result to the desired time zone and return.
    return nextOccurrence.tz(timeZone).toDate();
}

/**
 * Jumps to the first occurrence of the event that is on or after the specified timeframe start,
 * taking into account the daily recurrence pattern.
 *
 * @param scheduleStart - The original start date of the schedule.
 * @param recurrence - The daily recurrence details.
 * @param timeframeStart - The start of the timeframe of interest.
 * @param timeZone - The time zone identifier (e.g., 'Europe/London').
 * @returns The Date object for the first occurrence on or after the timeframe start.
 */
export async function jumpToFirstRelevantDailyOccurrence(
    scheduleStart: Date,
    recurrence: ScheduleRecurrence,
    timeframeStart: Date,
    timeZone = "UTC",
): Promise<Date> {
    await ensureMomentTimezone();

    const relevantStart = moment.utc(scheduleStart);
    let timeframeStartMoment = moment.utc(timeframeStart);

    // Check if the timeframe start is before the schedule start
    if (timeframeStartMoment.isBefore(relevantStart)) {
        console.error("Timeframe start is before the schedule start.");
        timeframeStartMoment = relevantStart.clone();
    }

    // Calculate days between schedule start and timeframe start
    const daysBetween = timeframeStartMoment.diff(relevantStart, "days");

    // Calculate how many intervals have occurred between the timeframe start and the schedule start
    const intervalsPassed = daysBetween === 0 ? 0 : Math.floor(daysBetween / recurrence.interval);

    // Jump ahead by the number of intervals that have passed
    relevantStart.add(intervalsPassed * recurrence.interval, "days");

    // Now find the first occurrence on or after the timeframeStart
    while (relevantStart.isBefore(timeframeStartMoment)) {
        relevantStart.add(recurrence.interval, "days");
    }

    // Convert the result to the desired time zone and return.
    return relevantStart.tz(timeZone).toDate();
}

/**
 * Jumps to the first occurrence of the event that is on or after the specified timeframe start.
 *
 * @param scheduleStart - The original start date of the schedule.
 * @param recurrence - The daily recurrence details.
 * @param timeframeStart - The start of the timeframe of interest.
 * @param timeZone - The time zone identifier (e.g., 'Europe/London').
 * @returns The Date object for the first occurrence on or after the timeframe start.
 */
export async function jumpToFirstRelevantWeeklyOccurrence(
    scheduleStart: Date,
    recurrence: ScheduleRecurrence,
    timeframeStart: Date,
    timeZone = "UTC",
): Promise<Date> {
    await ensureMomentTimezone();

    const relevantStart = moment.utc(scheduleStart);
    let timeframeStartMoment = moment.utc(timeframeStart);
    const targetDayOfWeek = (recurrence.dayOfWeek !== null && recurrence.dayOfWeek !== undefined) ? recurrence.dayOfWeek : timeframeStartMoment.day();

    // Check if the timeframe start is before the schedule start
    if (timeframeStartMoment.isBefore(relevantStart)) {
        console.error("Timeframe start is before the schedule start.");
        timeframeStartMoment = relevantStart.clone();
    }

    // Calculate weeks between schedule start and timeframe start
    const weeksBetween = timeframeStartMoment.diff(relevantStart, "weeks");

    // Calculate how many intervals have occurred between the timeframe start and the schedule start
    const intervalsPassed = weeksBetween === 0 ? 0 : Math.floor(weeksBetween / recurrence.interval);

    // Jump ahead by the number of intervals that have passed
    relevantStart.add(intervalsPassed * recurrence.interval * DAYS_IN_WEEK, "days");

    // Adjust to the next relevant dayOfWeek
    let daysToAdd = ((DAYS_IN_WEEK - relevantStart.day() + targetDayOfWeek) % DAYS_IN_WEEK);
    if (daysToAdd === 0 && relevantStart.isBefore(timeframeStartMoment)) {
        daysToAdd = DAYS_IN_WEEK;
    }
    relevantStart.add(daysToAdd, "days");

    // Now find the first occurrence on or after the timeframeStart
    while (relevantStart.isBefore(timeframeStartMoment)) {
        relevantStart.add(recurrence.interval * DAYS_IN_WEEK, "days");
    }

    // Convert the result to the desired time zone and return.
    return relevantStart.tz(timeZone).toDate();
}

/**
 * Jumps to the first occurrence of the event that is on or after the specified timeframe start,
 * taking into account the monthly recurrence pattern.
 *
 * @param scheduleStart - The original start date of the schedule.
 * @param recurrence - The monthly recurrence details.
 * @param timeframeStart - The start of the timeframe of interest.
 * @param timeZone - The time zone identifier (e.g., 'Europe/London').
 * @returns The Date object for the first occurrence on or after the timeframe start.
 */
export async function jumpToFirstRelevantMonthlyOccurrence(
    scheduleStart: Date,
    recurrence: ScheduleRecurrence,
    timeframeStart: Date,
    timeZone = "UTC",
): Promise<Date> {
    await ensureMomentTimezone();

    const relevantStart = moment.utc(scheduleStart);
    let timeframeStartMoment = moment.utc(timeframeStart);

    // Check if the timeframe start is before the schedule start
    if (timeframeStartMoment.isBefore(relevantStart)) {
        console.error("Timeframe start is before the schedule start.");
        timeframeStartMoment = relevantStart.clone();
    }

    const nextOccurrence = relevantStart.clone();

    if (recurrence.dayOfMonth) {
        // Set to the first occurrence of the day of the month after the timeframe start
        nextOccurrence.date(recurrence.dayOfMonth);
        if (nextOccurrence.isBefore(timeframeStartMoment) || nextOccurrence.date() !== recurrence.dayOfMonth) {
            nextOccurrence.add(recurrence.interval, "months");
        }
    } else {
        // If dayOfMonth isn't specified, simply set to the same day on the next interval that's on or after timeframe start
        while (nextOccurrence.isBefore(timeframeStartMoment)) {
            nextOccurrence.add(recurrence.interval, "months");
        }
    }

    // Ensure the next occurrence falls on or after the timeframe start
    while (nextOccurrence.isBefore(timeframeStartMoment)) {
        nextOccurrence.add(recurrence.interval, "months");
    }

    // Convert the result to the desired time zone and return.
    return nextOccurrence.tz(timeZone).toDate();
}

/**
 * Jumps to the first occurrence of the event that is on or after the specified timeframe start,
 * taking into account the yearly recurrence pattern.
 *
 * @param scheduleStart - The original start date of the schedule.
 * @param recurrence - The yearly recurrence details.
 * @param timeframeStart - The start of the timeframe of interest.
 * @param timeZone - The time zone identifier (e.g., 'Europe/London').
 * @returns The Date object for the first occurrence on or after the timeframe start.
 */
export async function jumpToFirstRelevantYearlyOccurrence(
    scheduleStart: Date,
    recurrence: ScheduleRecurrence,
    timeframeStart: Date,
    timeZone = "UTC",
): Promise<Date> {
    await ensureMomentTimezone();

    // Use UTC for calculations
    const scheduleStartMoment = moment.utc(scheduleStart);
    let timeframeStartMoment = moment.utc(timeframeStart);

    // Capture the time components from the schedule start to apply later
    const timeComponents = {
        hours: scheduleStartMoment.hours(),
        minutes: scheduleStartMoment.minutes(),
        seconds: scheduleStartMoment.seconds(),
    };

    // If the timeframe start is before the schedule start, we use the schedule start as the reference
    if (timeframeStartMoment.isBefore(scheduleStartMoment)) {
        console.error("Timeframe start is before the schedule start.");
        timeframeStartMoment = scheduleStartMoment.clone();
    }

    // Calculate the years difference to jump to the correct year directly
    const yearsDifference = timeframeStartMoment.year() - scheduleStartMoment.year();

    // Calculate the next occurrence year by adding the necessary multiples of the interval
    const yearsToAdd = yearsDifference + (yearsDifference % recurrence.interval !== 0 ? recurrence.interval - (yearsDifference % recurrence.interval) : 0);

    // Create a moment for the potential next occurrence, applying the calculated year and original time
    const nextOccurrence = scheduleStartMoment.clone().add(yearsToAdd, "years").set(timeComponents);

    // If month and dayOfMonth are set, adjust the date
    if (recurrence.month !== undefined && recurrence.month !== null && recurrence.dayOfMonth !== undefined && recurrence.dayOfMonth !== null) {
        nextOccurrence.month(recurrence.month); // month is 0-indexed in moment

        // Set the day or adjust to the last day of the month if the provided day is too large
        const daysInMonth = nextOccurrence.daysInMonth();
        nextOccurrence.date(Math.min(recurrence.dayOfMonth, daysInMonth));

        // If the calculated occurrence is still before the timeframe start, add the interval once more
        if (nextOccurrence.isBefore(timeframeStartMoment)) {
            nextOccurrence.add(recurrence.interval, "years");
            // Re-calculate the last day of the month after adding years to handle leap years
            const daysInMonthAfterAdd = nextOccurrence.daysInMonth();
            nextOccurrence.date(Math.min(recurrence.dayOfMonth, daysInMonthAfterAdd));
        }
    } else {
        // If month and dayOfMonth are not set, just ensure the next occurrence year is after the timeframe start year
        while (nextOccurrence.isBefore(timeframeStartMoment)) {
            nextOccurrence.add(recurrence.interval, "years");
        }
    }

    // Convert the result to the desired time zone without changing the actual time and return
    return nextOccurrence.tz(timeZone, true).toDate();
}


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
export function applyExceptions(
    currentStartTime: Date,
    schedule: Schedule,
    duration: number,
    timeZone: string,
): { start: Date; end: Date } | null | undefined {
    // Iterate through the list of exceptions in the schedule
    for (const exception of schedule.exceptions) {
        const exceptionOriginalStartTime = new Date(exception.originalStartTime.toLocaleString("en-US", { timeZone }));

        // Check for exceptions within a 1-minute window of the current start time
        if (Math.abs(exceptionOriginalStartTime.getTime() - currentStartTime.getTime()) < MINUTES_1_MS) {
            // The occurrence has an exception
            if (exception.newStartTime || exception.newEndTime) {
                // If there's a new start time, use it, otherwise set it original start time
                const newStart = exception.newStartTime ? new Date(exception.newStartTime) : exceptionOriginalStartTime;
                // If there's a new end time, use it, otherwise set it to 1 hour after the new start time
                const newEnd = exception.newEndTime ? new Date(exception.newEndTime) : new Date(newStart.getTime() + duration);

                return { start: newStart, end: newEnd }; // Return the rescheduled occurrence
            } else {
                return null; // The occurrence is cancelled, return null
            }
        }
    }
    return undefined; // No exceptions apply, return undefined
}

async function calculateNextOccurrence(currentStartTime: Date, recurrence: ScheduleRecurrence, timeZone?: string): Promise<Date> {
    switch (recurrence.recurrenceType) {
        case "Daily":
            return await calculateNextDailyOccurrence(currentStartTime, recurrence, timeZone);
        case "Weekly":
            return await calculateNextWeeklyOccurrence(currentStartTime, recurrence, timeZone);
        case "Monthly":
            return await calculateNextMonthlyOccurrence(currentStartTime, recurrence, timeZone);
        case "Yearly":
            return await calculateNextYearlyOccurrence(currentStartTime, recurrence, timeZone);
        default:
            throw new Error("Invalid recurrence type");
    }
}

/**
 * Calculate occurrences of a scheduled event within a given time frame. 
 * NOTE: When creating schedules, make sure that dates are stored as UTC, 
 * not the datetime the user picked in their local time zone with a "Z" 
 * slapped on the end.
 *
 * @param schedule - The schedule object.
 * @param timeframeStart - The start of the time frame as a Date, in UTC.
 * @param timeframeEnd - The end of the time frame as a Date, in UTC.
 * @returns An array of objects representing the occurrences of the scheduled event.
 */
export async function calculateOccurrences(
    schedule: Schedule,
    timeframeStart: Date,
    timeframeEnd: Date,
): Promise<Array<{ start: Date; end: Date }>> {
    const occurrences: Array<{ start: Date; end: Date }> = [];
    // Make sure that the time frame is no longer than a year, so that we don't overload the server
    if (!validateTimeFrame(timeframeStart, timeframeEnd)) {
        console.error("calculateOccurrences time frame to large", { trace: "0432", timeframeStart, timeframeEnd });
        return occurrences;
    }
    const startTime = new Date(schedule.startTime.toLocaleString("en-US", { timeZone: schedule.timezone }));
    // Calcuate occurrences for each recurrence
    for (const recurrence of schedule.recurrences) {
        const duration = recurrence.duration ?? HOURS_1_MS; // Default to 1 hour if no duration is provided
        // Use jump functions to calculate the first relevant start time
        let currentStartTime = startTime;
        switch (recurrence.recurrenceType) {
            case "Daily":
                currentStartTime = await jumpToFirstRelevantDailyOccurrence(startTime, recurrence, timeframeStart, schedule.timezone);
                break;
            case "Weekly":
                currentStartTime = await jumpToFirstRelevantWeeklyOccurrence(startTime, recurrence, timeframeStart, schedule.timezone);
                break;
            case "Monthly":
                currentStartTime = await jumpToFirstRelevantMonthlyOccurrence(startTime, recurrence, timeframeStart, schedule.timezone);
                break;
            case "Yearly":
                currentStartTime = await jumpToFirstRelevantYearlyOccurrence(startTime, recurrence, timeframeStart, schedule.timezone);
                break;
        }

        // While the current start time is before the end of the time frame (and also while we haven't maxed out the loop)
        while (currentStartTime <= timeframeEnd && occurrences.length < OCCURRENCE_LOOP_LIMIT) {
            const currentEndTime = new Date(currentStartTime.getTime() + duration);

            // Check if the currentStartTime is after the recurrence endDate, if so break the loop before adding to occurrences
            if (recurrence.endDate && currentStartTime.getTime() >= new Date(recurrence.endDate).getTime()) {
                break;
            }

            const prevStartTime = currentStartTime; // Store the previous start time to compare later

            // Apply exceptions (if any) to the current occurrence
            const exceptionResult = applyExceptions(currentStartTime, schedule, duration, schedule.timezone);
            // If null, the occurrence was canceled
            if (exceptionResult === null) {
                // Move to the next occurrence immediately since this one is cancelled, but do not break, continue evaluating
                currentStartTime = await calculateNextOccurrence(currentStartTime, recurrence, schedule.timezone);
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
            currentStartTime = await calculateNextOccurrence(currentStartTime, recurrence, schedule.timezone);

            // Break the loop early if:
            // - The schedule end time is before the current occurrence start time
            // - The current start time hasn't changed (to prevent an infinite loop)
            if ((schedule.endTime && new Date(schedule.endTime).getTime() < currentStartTime.getTime()) ||
                prevStartTime.getTime() === currentStartTime.getTime()) {
                break;
            }
        }
    }
    // Return occurrences
    return occurrences;
}
