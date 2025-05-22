import { type RequestFile } from "@local/server";
import { type CalendarResponse, type DateWithTimeZone } from "node-ical";

/**
 * Parses an iCalendar file.
 * 
 * @param file The iCalendar file to parse
 * @returns A promise that resolves to the parsed events in the iCalendar file.
 */
export async function parseICalFile(file: RequestFile): Promise<CalendarResponse> {
    // return new Promise((resolve, reject) => {
    //     const writeStream = fs.createWriteStream(filePath);

    //     file.pipe(writeStream);

    //     writeStream.on("finish", async () => {
    //         try {
    //             const events = await ical.async.parseFile(filePath);
    //             fs.unlinkSync(filePath); // Clean up temporary file
    //             resolve(events);
    //         } catch (error) {
    //             reject(error);
    //         }
    //     });

    //     writeStream.on("error", (error) => reject(error));
    // });
    //TODO
    return {} as any;
}

/**
 * Creates an iCalendar event from a schedule.
 * 
 * @param schedule - The schedule to create the event from.
 * @param recurrence - The recurrence related to the schedule.
 * @returns A string representing the iCalendar event.
 */
export function createICalEvent(
    schedule: {
        id: string;
        startTime: Date;
        endTime: Date;
        timezone: string;
    },
    recurrence?: {
        recurrenceType: string;
        interval?: number;
        dayOfWeek?: number;
        dayOfMonth?: number;
        month?: number;
        endDate?: Date;
    },
): CalendarResponse {
    function convertToDateWithTimeZone(date: Date, tz: string): DateWithTimeZone {
        return { ...date, tz };
    }

    const event = {
        type: "VEVENT",
        uid: schedule.id,
        start: convertToDateWithTimeZone(schedule.startTime, schedule.timezone),
        end: convertToDateWithTimeZone(schedule.endTime, schedule.timezone),
        dtstamp: convertToDateWithTimeZone(new Date(), schedule.timezone),
        ...(recurrence && {
            rrule: {
                freq: recurrence.recurrenceType,
                interval: recurrence.interval || 1,
                byday: recurrence.dayOfWeek,
                bymonthday: recurrence.dayOfMonth,
                bymonth: recurrence.month,
                until: recurrence.endDate,
            },
        }),
    } as const;

    const calendar: CalendarResponse = {
        type: "VCALENDAR",
        prodid: "-//Your Company//Your Product//EN",
        version: "2.0",
        components: [event],
    } as any;// TODO

    return calendar;
}
