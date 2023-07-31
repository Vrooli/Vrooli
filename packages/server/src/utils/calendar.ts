import fs from "fs";
import ical, { CalendarResponse } from "node-ical";

/**
 * Parses an iCalendar file.
 * 
 * @param file - The iCalendar file to parse. This should be provided as an Apollo Server `Upload`.
 * @returns A promise that resolves to the parsed events in the iCalendar file.
 */
export const parseICalFile = async (file: Express.Multer.File): Promise<CalendarResponse> => {
    // Directly parse the file using its path
    const events = await ical.async.parseFile(file.path);

    // After parsing, you might want to remove the uploaded file
    fs.unlinkSync(file.path);

    return events;
};

// /**
//  * Creates an iCalendar event from a schedule.
//  * 
//  * @param schedule - The schedule to create the event from.
//  * @param recurrence - The recurrence related to the schedule.
//  * @returns A promise that resolves to the iCalendar event.
//  */
// export const createICalEvent = (schedule: {
//     id: string;
//     startTime: Date;
//     endTime: Date;
//     timezone: string;
// }, recurrence: {
//     recurrenceType: string;
//     interval: number;
//     dayOfWeek: number;
//     dayOfMonth: number;
//     month: number;
//     endDate: Date;
// }): string => {
//     // Create an iCal event from the schedule and recurrence
//     const event: ical.CalendarComponent = {
//         type: "VEVENT",
//         uid: schedule.id,
//         start: new Date(schedule.startTime),
//         end: new Date(schedule.endTime),
//         dtstamp: new Date(),
//         tz: schedule.timezone,
//         rrule: {
//             freq: recurrence.recurrenceType,
//             interval: recurrence.interval,
//             byday: recurrence.dayOfWeek,
//             bymonthday: recurrence.dayOfMonth,
//             bymonth: recurrence.month,
//             until: new Date(recurrence.endDate),
//         },
//     };

//     const calendar: ical.CalendarResponse = {
//         type: "VCALENDAR",
//         tz: schedule.timezone,
//         prodid: "-//Your Company//Your Product//EN",
//         version: "2.0",
//         components: [event],
//     } as const;

//     return ical.stringify(calendar);
// };
