import { Schedule, ScheduleRecurrence } from "@local/shared";
import moment from "moment-timezone"; // Native Date objects don't handle time zones well

/** Milliseconds in one day */
export const ONE_DAY_IN_MS = 1000 * 60 * 60 * 24;
/** Milliseconds in one week */
export const ONE_WEEK_IN_MS = 1000 * 60 * 60 * 24 * 7;
/** Milliseconds in a year */
export const ONE_YEAR_IN_MS = 1000 * 60 * 60 * 24 * 365;

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
 * @param interval - The interval of the daily recurrence.
 * @param timeZone - The time zone identifier (e.g., 'America/New_York').
 * @returns The next start time for the daily recurrence.
 */
export const calculateNextDailyOccurrence = (currentStartTime: Date, recurrence: ScheduleRecurrence, timeZone?: string): Date => {
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
export const calculateNextWeeklyOccurrence = (currentStartTime: Date, recurrence: ScheduleRecurrence, timeZone?: string): Date => {
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
export const calculateNextMonthlyOccurrence = (currentStartTime: Date, recurrence: ScheduleRecurrence, timeZone?: string): Date => {
    // Convert the current start time to the desired time zone.
    const currentMoment = moment.tz(currentStartTime, timeZone ?? "UTC");
    console.log("currentMoment", currentMoment);

    const nextOccurrence = currentMoment.clone();
    console.log("nextOccurrence", nextOccurrence);
    if (recurrence.dayOfMonth) {
        // Check if the current day of the month is already past the desired dayOfMonth.
        if (currentMoment.date() >= recurrence.dayOfMonth) {
            console.log("calculatemonth hereee 1", currentMoment);
            // Advance by the interval and set to the desired dayOfMonth.
            nextOccurrence.add(recurrence.interval, "months").date(recurrence.dayOfMonth);
        } else {
            console.log("calculatemonth hereee 2", currentMoment);
            // Simply set to the desired dayOfMonth.
            nextOccurrence.date(recurrence.dayOfMonth);
        }
    } else {
        nextOccurrence.add(recurrence.interval, "months");
    }

    console.log("before DST", nextOccurrence, nextOccurrence.utcOffset());
    // Ensure time remains consistent across DST boundary
    if (nextOccurrence.utcOffset() !== currentMoment.utcOffset()) {
        nextOccurrence.add(nextOccurrence.utcOffset() - currentMoment.utcOffset(), "minutes");
    }

    console.log("after DST", nextOccurrence);
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
    const currentMoment = moment.tz(currentStartTime, timeZone);

    let nextOccurrence;
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

    // Ensure time remains consistent across DST boundary
    if (nextOccurrence.utcOffset() !== currentMoment.utcOffset()) {
        nextOccurrence.add(nextOccurrence.utcOffset() - currentMoment.utcOffset(), "minutes");
    }

    // Convert the result back to UTC and return.
    return nextOccurrence.utc().toDate();
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
};
