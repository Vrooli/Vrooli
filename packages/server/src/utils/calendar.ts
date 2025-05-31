import { HOURS_1_MS, MINUTES_1_MS, ScheduleRecurrenceType, generatePK, type Schedule, type ScheduleRecurrence } from "@local/shared";
import ical, { type CalendarResponse, type DateWithTimeZone, type VCalendar, type VEvent } from "node-ical";
import { type RequestFile } from "../types.js";

const DEFAULT_EVENT_DURATION_MINUTES = 60;
const SUNDAY_DAY_NUMBER = 7;

/**
 * Parses an iCalendar file.
 * 
 * @param file The iCalendar file to parse
 * @returns A promise that resolves to the parsed events in the iCalendar file.
 */
export async function parseICalFile(file: RequestFile): Promise<CalendarResponse> {
    if (!file || !file.buffer) {
        throw new Error("Invalid or empty iCalendar file provided.");
    }
    try {
        const icsString = file.buffer.toString("utf-8");
        const events = await ical.async.parseICS(icsString);
        return events;
    } catch (error) {
        console.error("Error parsing iCal file:", error);
        throw new Error("Failed to parse iCalendar file. Ensure the file is valid.");
    }
}

/**
 * Converts parsed iCal events to schedule creation inputs.
 * 
 * @param events - The parsed iCal events from parseICalFile
 * @param userId - The ID of the user importing the calendar
 * @returns An array of schedule creation inputs
 */
export function convertICalEventsToSchedules(events: CalendarResponse, userId: string): Array<{
    id: string;
    startTime: Date | null;
    endTime: Date | null;
    timezone: string;
    title: string;
    description: string;
    user: { id: string };
    recurrences?: Array<{
        id: string;
        recurrenceType: ScheduleRecurrenceType;
        interval: number;
        dayOfWeek?: number;
        dayOfMonth?: number;
        month?: number;
        endDate?: Date;
        duration: number;
    }>;
}> {
    const schedules: ReturnType<typeof convertICalEventsToSchedules> = [];

    for (const [_eventId, event] of Object.entries(events)) {
        if (event.type !== "VEVENT") continue;

        const vevent = event as VEvent;

        // Extract basic event information
        const title = vevent.summary || "Imported Event";
        const description = vevent.description || "";

        // Handle dates - they could be Date objects or DateWithTimeZone objects
        const startTime = extractDate(vevent.start);
        const endTime = extractDate(vevent.end);

        // Default timezone or extract from event
        const timezone = extractTimezone(vevent.start) || "UTC";

        if (!startTime) continue; // Skip events without start time

        const schedule: {
            id: string;
            startTime: Date;
            endTime: Date;
            timezone: string;
            title: string;
            description: string;
            user: { id: string };
            recurrences?: Array<{
                id: string;
                recurrenceType: ScheduleRecurrenceType;
                interval: number;
                dayOfWeek?: number;
                dayOfMonth?: number;
                month?: number;
                endDate?: Date;
                duration: number;
            }>;
        } = {
            id: generatePK().toString(),
            startTime,
            endTime: endTime || new Date(startTime.getTime() + HOURS_1_MS), // Default 1 hour duration
            timezone,
            title,
            description,
            user: { id: userId },
        };

        // Handle recurrence rules
        if (vevent.rrule) {
            const recurrence = parseRRule(vevent.rrule, startTime, endTime);
            if (recurrence) {
                schedule.recurrences = [recurrence];
            }
        }

        schedules.push(schedule);
    }

    return schedules;
}

/**
 * Extracts a Date from various iCal date formats.
 */
function extractDate(dateValue: any): Date | null {
    if (!dateValue) return null;

    if (dateValue instanceof Date) {
        return dateValue;
    }

    if (typeof dateValue === "object" && dateValue.date) {
        return new Date(dateValue.date);
    }

    if (typeof dateValue === "string") {
        return new Date(dateValue);
    }

    return null;
}

/**
 * Extracts timezone from iCal date object.
 */
function extractTimezone(dateValue: any): string | null {
    if (typeof dateValue === "object" && dateValue.tz) {
        return dateValue.tz;
    }
    return null;
}

/**
 * Parses an RRULE object into a recurrence pattern.
 */
function parseRRule(rrule: any, startTime: Date, endTime: Date | null): {
    id: string;
    recurrenceType: ScheduleRecurrenceType;
    interval: number;
    dayOfWeek?: number;
    dayOfMonth?: number;
    month?: number;
    endDate?: Date;
    duration: number;
} | null {
    if (!rrule || !rrule.freq) return null;

    const duration = endTime ? Math.floor((endTime.getTime() - startTime.getTime()) / MINUTES_1_MS) : DEFAULT_EVENT_DURATION_MINUTES; // Duration in minutes

    let recurrenceType: ScheduleRecurrenceType;
    switch (rrule.freq?.toUpperCase()) {
        case "DAILY":
            recurrenceType = ScheduleRecurrenceType.Daily;
            break;
        case "WEEKLY":
            recurrenceType = ScheduleRecurrenceType.Weekly;
            break;
        case "MONTHLY":
            recurrenceType = ScheduleRecurrenceType.Monthly;
            break;
        case "YEARLY":
            recurrenceType = ScheduleRecurrenceType.Yearly;
            break;
        default:
            return null;
    }

    const recurrence = {
        id: generatePK().toString(),
        recurrenceType,
        interval: rrule.interval || 1,
        duration,
    } as ReturnType<typeof parseRRule>;

    if (!recurrence) return null;

    // Handle day of week for weekly recurrence
    if (recurrenceType === ScheduleRecurrenceType.Weekly && rrule.byweekday) {
        const weekdays = Array.isArray(rrule.byweekday) ? rrule.byweekday : [rrule.byweekday];
        if (weekdays.length > 0) {
            // Convert to our format (1 = Monday, 7 = Sunday)
            const dayMap: Record<string, number> = {
                "MO": 1, "TU": 2, "WE": 3, "TH": 4, "FR": 5, "SA": 6, "SU": SUNDAY_DAY_NUMBER,
            };
            const firstDay = weekdays[0].toString().substring(0, 2);
            recurrence.dayOfWeek = dayMap[firstDay] || startTime.getDay() || SUNDAY_DAY_NUMBER;
        }
    }

    // Handle day of month for monthly recurrence
    if (recurrenceType === ScheduleRecurrenceType.Monthly && rrule.bymonthday && recurrence) {
        recurrence.dayOfMonth = Array.isArray(rrule.bymonthday) ? rrule.bymonthday[0] : rrule.bymonthday;
    }

    // Handle month for yearly recurrence
    if (recurrenceType === ScheduleRecurrenceType.Yearly && rrule.bymonth && recurrence) {
        recurrence.month = Array.isArray(rrule.bymonth) ? rrule.bymonth[0] : rrule.bymonth;
    }

    // Handle end date
    if (rrule.until && recurrence) {
        recurrence.endDate = new Date(rrule.until);
    }

    return recurrence;
}

/**
 * Creates an iCalendar string from schedules.
 * 
 * @param schedules - Array of schedules with their related meetings/runs
 * @returns A string containing the iCalendar data
 */
export function createICalFromSchedules(schedules: Array<{
    schedule: Schedule;
    meeting?: { translations: Array<{ name: string; description?: string; language: string }> };
    run?: { name: string; resourceVersion?: { translations: Array<{ name: string; description?: string; language: string }> } };
}>): string {
    const lines: string[] = [
        "BEGIN:VCALENDAR",
        "VERSION:2.0",
        "PRODID:-//Vrooli//Vrooli Calendar Service v1.0//EN",
        "CALSCALE:GREGORIAN",
    ];

    schedules.forEach(({ schedule, meeting, run }) => {
        // Get event title and description
        let title = "Scheduled Event";
        let description = "";

        if (meeting) {
            const translation = meeting.translations.find(t => t.language === "en") || meeting.translations[0];
            if (translation) {
                title = translation.name;
                description = translation.description || "";
            }
        } else if (run) {
            title = run.name || "Run";
            if (run.resourceVersion) {
                const translation = run.resourceVersion.translations.find(t => t.language === "en") || run.resourceVersion.translations[0];
                if (translation) {
                    title = translation.name;
                    description = translation.description || "";
                }
            }
        }

        lines.push("BEGIN:VEVENT");
        lines.push(`UID:${schedule.id}@vrooli.com`);
        lines.push(`DTSTAMP:${formatICalDate(new Date())}`);
        lines.push(`DTSTART;TZID=${schedule.timezone}:${formatICalDate(new Date(schedule.startTime))}`);
        lines.push(`DTEND;TZID=${schedule.timezone}:${formatICalDate(new Date(schedule.endTime))}`);
        lines.push(`SUMMARY:${escapeICalText(title)}`);

        if (description) {
            lines.push(`DESCRIPTION:${escapeICalText(description)}`);
        }

        // Add recurrence rules
        schedule.recurrences?.forEach(recurrence => {
            const rrule = createRRuleString(recurrence);
            if (rrule) {
                lines.push(`RRULE:${rrule}`);
            }
        });

        lines.push("END:VEVENT");
    });

    lines.push("END:VCALENDAR");
    return lines.join("\r\n");
}

/**
 * Formats a Date object for iCalendar format (YYYYMMDDTHHMMSSZ).
 */
function formatICalDate(date: Date): string {
    return date.toISOString().replace(/[-:]/g, "").replace(/\.\d{3}/, "");
}

/**
 * Escapes text for iCalendar format.
 */
function escapeICalText(text: string): string {
    return text
        .replace(/\\/g, "\\\\")
        .replace(/,/g, "\\,")
        .replace(/;/g, "\\;")
        .replace(/\n/g, "\\n")
        .replace(/\r/g, "");
}

/**
 * Creates an RRULE string from a schedule recurrence.
 */
function createRRuleString(recurrence: ScheduleRecurrence): string | null {
    const parts: string[] = [];

    parts.push(`FREQ=${recurrence.recurrenceType.toUpperCase()}`);

    if (recurrence.interval && recurrence.interval > 1) {
        parts.push(`INTERVAL=${recurrence.interval}`);
    }

    if (recurrence.dayOfWeek && recurrence.recurrenceType === ScheduleRecurrenceType.Weekly) {
        const dayMap = ["", "MO", "TU", "WE", "TH", "FR", "SA", "SU"];
        parts.push(`BYDAY=${dayMap[recurrence.dayOfWeek]}`);
    }

    if (recurrence.dayOfMonth && recurrence.recurrenceType === ScheduleRecurrenceType.Monthly) {
        parts.push(`BYMONTHDAY=${recurrence.dayOfMonth}`);
    }

    if (recurrence.month && recurrence.recurrenceType === ScheduleRecurrenceType.Yearly) {
        parts.push(`BYMONTH=${recurrence.month}`);
    }

    if (recurrence.endDate) {
        parts.push(`UNTIL=${formatICalDate(new Date(recurrence.endDate))}`);
    }

    return parts.join(";");
}

/**
 * Creates an iCalendar event from a schedule (legacy function for backwards compatibility).
 * 
 * @param schedule - The schedule to create the event from.
 * @param recurrence - The recurrence related to the schedule.
 * @returns A VCalendar object representing the iCalendar event.
 */
export function createICalEvent(
    schedule: {
        id: string;
        name: string;
        startTime: Date;
        endTime: Date;
        timezone: string;
    },
    recurrence?: {
        recurrenceType: string;
        interval?: number;
        dayOfWeek?: string;
        dayOfMonth?: number;
        month?: number;
        endDate?: Date;
    },
): VCalendar {
    function convertToDateWithTimeZone(date: Date, tz: string): DateWithTimeZone {
        return Object.assign(date, { tz }) as DateWithTimeZone;
    }

    const event: Partial<VEvent> = {
        type: "VEVENT",
        uid: schedule.id,
        summary: schedule.name,
        start: convertToDateWithTimeZone(schedule.startTime, schedule.timezone),
        end: convertToDateWithTimeZone(schedule.endTime, schedule.timezone),
        dtstamp: convertToDateWithTimeZone(new Date(), schedule.timezone),
        status: "CONFIRMED",
        sequence: "0",
    };

    if (recurrence) {
        const freq = recurrence.recurrenceType.toUpperCase() as any;
        const rruleParams: any = { freq, interval: recurrence.interval || 1 };

        if (recurrence.dayOfWeek) {
            rruleParams.byweekday = Array.isArray(recurrence.dayOfWeek) ? recurrence.dayOfWeek : [recurrence.dayOfWeek];
        }
        if (recurrence.dayOfMonth) {
            rruleParams.bymonthday = recurrence.dayOfMonth;
        }
        if (recurrence.month) {
            rruleParams.bymonth = recurrence.month;
        }
        if (recurrence.endDate) {
            rruleParams.until = convertToDateWithTimeZone(recurrence.endDate, schedule.timezone);
        }
        (event as any).rrule = rruleParams;
    }

    const calendar: Partial<VCalendar> = {
        type: "VCALENDAR",
        prodid: "-//Vrooli//Vrooli Calendar Service v1.0//EN",
        version: "2.0",
    };

    return calendar as VCalendar;
}
