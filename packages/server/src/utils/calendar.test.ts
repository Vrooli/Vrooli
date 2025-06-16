import { describe, it, expect, beforeEach } from "vitest";
import { ScheduleRecurrenceType, HOURS_1_MS, MINUTES_1_MS } from "@vrooli/shared";
import { parseICalFile, convertICalEventsToSchedules, createICalFromSchedules, createICalEvent } from "./calendar.js";
import type { RequestFile } from "../types.js";
import type { CalendarResponse, VEvent, DateWithTimeZone } from "node-ical";

// Mock setup for generatePK
let mockIdCounter = 0;

describe("calendar.ts", () => {
    beforeEach(() => {
        mockIdCounter = 0;
    });

    describe("parseICalFile", () => {
        it("should throw error for invalid file", async () => {
            const invalidFile = null as unknown as RequestFile;
            await expect(parseICalFile(invalidFile)).rejects.toThrow("Invalid or empty iCalendar file provided.");
        });

        it("should throw error for file without buffer", async () => {
            const fileWithoutBuffer = {} as RequestFile;
            await expect(parseICalFile(fileWithoutBuffer)).rejects.toThrow("Invalid or empty iCalendar file provided.");
        });

        it("should parse valid iCalendar file", async () => {
            const validIcalContent = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VEVENT
UID:test-event@example.com
DTSTART:20240101T120000Z
DTEND:20240101T130000Z
SUMMARY:Test Event
DESCRIPTION:A test event
END:VEVENT
END:VCALENDAR`;

            const file: RequestFile = {
                buffer: Buffer.from(validIcalContent, "utf-8"),
                originalname: "test.ics",
                mimetype: "text/calendar",
                size: validIcalContent.length,
            };

            const result = await parseICalFile(file);
            expect(result).toBeDefined();
            expect(typeof result).toBe("object");
        });

        it("should handle malformed iCalendar content gracefully", async () => {
            const malformedContent = "This is not a valid iCalendar file";
            const file: RequestFile = {
                buffer: Buffer.from(malformedContent, "utf-8"),
                originalname: "invalid.ics",
                mimetype: "text/calendar",
                size: malformedContent.length,
            };

            // The node-ical library may return an empty object for malformed content
            const result = await parseICalFile(file);
            expect(result).toBeDefined();
            expect(typeof result).toBe("object");
        });
    });

    describe("convertICalEventsToSchedules", () => {
        it("should convert simple event to schedule", () => {
            const mockEvents: CalendarResponse = {
                "test-event": {
                    type: "VEVENT",
                    summary: "Test Meeting",
                    description: "A test meeting",
                    start: new Date("2024-01-01T12:00:00Z"),
                    end: new Date("2024-01-01T13:00:00Z"),
                } as VEvent,
            };

            const result = convertICalEventsToSchedules(mockEvents, "user-123");

            expect(result).toHaveLength(1);
            expect(result[0]).toMatchObject({
                title: "Test Meeting",
                description: "A test meeting",
                user: { id: "user-123" },
                timezone: "UTC",
            });
            expect(result[0].id).toBeDefined();
            expect(result[0].startTime).toEqual(new Date("2024-01-01T12:00:00Z"));
            expect(result[0].endTime).toEqual(new Date("2024-01-01T13:00:00Z"));
        });

        it("should handle event without end time", () => {
            const mockEvents: CalendarResponse = {
                "test-event": {
                    type: "VEVENT",
                    summary: "Test Event",
                    start: new Date("2024-01-01T12:00:00Z"),
                } as VEvent,
            };

            const result = convertICalEventsToSchedules(mockEvents, "user-123");

            expect(result).toHaveLength(1);
            expect(result[0].endTime).toEqual(new Date(new Date("2024-01-01T12:00:00Z").getTime() + HOURS_1_MS));
        });

        it("should handle event with timezone", () => {
            const startDate = new Date("2024-01-01T12:00:00Z");
            Object.assign(startDate, { tz: "America/New_York" });

            const mockEvents: CalendarResponse = {
                "test-event": {
                    type: "VEVENT",
                    summary: "Timezone Event",
                    start: startDate as DateWithTimeZone,
                    end: new Date("2024-01-01T13:00:00Z"),
                } as VEvent,
            };

            const result = convertICalEventsToSchedules(mockEvents, "user-123");

            expect(result).toHaveLength(1);
            expect(result[0].timezone).toBe("America/New_York");
        });

        it("should handle event with string dates", () => {
            const mockEvents: CalendarResponse = {
                "test-event": {
                    type: "VEVENT",
                    summary: "String Date Event",
                    start: "2024-01-01T12:00:00Z",
                    end: "2024-01-01T13:00:00Z",
                } as VEvent,
            };

            const result = convertICalEventsToSchedules(mockEvents, "user-123");

            expect(result).toHaveLength(1);
            expect(result[0].startTime).toEqual(new Date("2024-01-01T12:00:00Z"));
        });

        it("should handle event with object date format", () => {
            const mockEvents: CalendarResponse = {
                "test-event": {
                    type: "VEVENT",
                    summary: "Object Date Event",
                    start: { date: "2024-01-01T12:00:00Z", tz: "Europe/London" },
                    end: { date: "2024-01-01T13:00:00Z" },
                } as VEvent,
            };

            const result = convertICalEventsToSchedules(mockEvents, "user-123");

            expect(result).toHaveLength(1);
            expect(result[0].startTime).toEqual(new Date("2024-01-01T12:00:00Z"));
            expect(result[0].timezone).toBe("Europe/London");
        });

        it("should skip events without start time", () => {
            const mockEvents: CalendarResponse = {
                "invalid-event": {
                    type: "VEVENT",
                    summary: "Invalid Event",
                    start: null,
                    end: null,
                } as VEvent,
            };

            const result = convertICalEventsToSchedules(mockEvents, "user-123");

            expect(result).toHaveLength(0);
        });

        it("should skip non-VEVENT entries", () => {
            const mockEvents: CalendarResponse = {
                "not-an-event": {
                    type: "VTIMEZONE",
                    tzid: "America/New_York",
                } as any,
            };

            const result = convertICalEventsToSchedules(mockEvents, "user-123");

            expect(result).toHaveLength(0);
        });

        it("should handle daily recurrence", () => {
            const mockEvents: CalendarResponse = {
                "recurring-event": {
                    type: "VEVENT",
                    summary: "Daily Meeting",
                    start: new Date("2024-01-01T12:00:00Z"),
                    end: new Date("2024-01-01T13:00:00Z"),
                    rrule: {
                        freq: "DAILY",
                        interval: 1,
                        until: new Date("2024-01-31T12:00:00Z"),
                    },
                } as VEvent,
            };

            const result = convertICalEventsToSchedules(mockEvents, "user-123");

            expect(result).toHaveLength(1);
            expect(result[0].recurrences).toHaveLength(1);
            expect(result[0].recurrences![0]).toMatchObject({
                recurrenceType: ScheduleRecurrenceType.Daily,
                interval: 1,
                duration: 60,
                endDate: new Date("2024-01-31T12:00:00Z"),
            });
            expect(result[0].recurrences![0].id).toBeDefined();
        });

        it("should handle weekly recurrence with specific day", () => {
            const mockEvents: CalendarResponse = {
                "weekly-event": {
                    type: "VEVENT",
                    summary: "Weekly Meeting",
                    start: new Date("2024-01-01T12:00:00Z"), // Monday
                    end: new Date("2024-01-01T13:00:00Z"),
                    rrule: {
                        freq: "WEEKLY",
                        interval: 2,
                        byweekday: "MO",
                    },
                } as VEvent,
            };

            const result = convertICalEventsToSchedules(mockEvents, "user-123");

            expect(result).toHaveLength(1);
            expect(result[0].recurrences![0]).toMatchObject({
                recurrenceType: ScheduleRecurrenceType.Weekly,
                interval: 2,
                dayOfWeek: 1, // Monday
            });
        });

        it("should handle monthly recurrence with specific day of month", () => {
            const mockEvents: CalendarResponse = {
                "monthly-event": {
                    type: "VEVENT",
                    summary: "Monthly Meeting",
                    start: new Date("2024-01-15T12:00:00Z"),
                    end: new Date("2024-01-15T13:00:00Z"),
                    rrule: {
                        freq: "MONTHLY",
                        bymonthday: 15,
                    },
                } as VEvent,
            };

            const result = convertICalEventsToSchedules(mockEvents, "user-123");

            expect(result).toHaveLength(1);
            expect(result[0].recurrences![0]).toMatchObject({
                recurrenceType: ScheduleRecurrenceType.Monthly,
                dayOfMonth: 15,
            });
        });

        it("should handle yearly recurrence with specific month", () => {
            const mockEvents: CalendarResponse = {
                "yearly-event": {
                    type: "VEVENT",
                    summary: "Yearly Meeting",
                    start: new Date("2024-06-01T12:00:00Z"),
                    end: new Date("2024-06-01T13:00:00Z"),
                    rrule: {
                        freq: "YEARLY",
                        bymonth: 6,
                    },
                } as VEvent,
            };

            const result = convertICalEventsToSchedules(mockEvents, "user-123");

            expect(result).toHaveLength(1);
            expect(result[0].recurrences![0]).toMatchObject({
                recurrenceType: ScheduleRecurrenceType.Yearly,
                month: 6,
            });
        });

        it("should handle unsupported recurrence frequency", () => {
            const mockEvents: CalendarResponse = {
                "unsupported-event": {
                    type: "VEVENT",
                    summary: "Unsupported Recurrence",
                    start: new Date("2024-01-01T12:00:00Z"),
                    end: new Date("2024-01-01T13:00:00Z"),
                    rrule: {
                        freq: "MINUTELY", // Unsupported
                    },
                } as VEvent,
            };

            const result = convertICalEventsToSchedules(mockEvents, "user-123");

            expect(result).toHaveLength(1);
            expect(result[0].recurrences).toBeUndefined();
        });

        it("should use default title and description when missing", () => {
            const mockEvents: CalendarResponse = {
                "minimal-event": {
                    type: "VEVENT",
                    start: new Date("2024-01-01T12:00:00Z"),
                } as VEvent,
            };

            const result = convertICalEventsToSchedules(mockEvents, "user-123");

            expect(result).toHaveLength(1);
            expect(result[0].title).toBe("Imported Event");
            expect(result[0].description).toBe("");
        });
    });

    describe("createICalFromSchedules", () => {
        it("should create basic iCalendar from schedule", () => {
            const schedules = [{
                schedule: {
                    id: "schedule-1",
                    startTime: new Date("2024-01-01T12:00:00Z"),
                    endTime: new Date("2024-01-01T13:00:00Z"),
                    timezone: "UTC",
                    recurrences: [],
                } as any,
                meeting: {
                    translations: [
                        { name: "Team Meeting", description: "Weekly team sync", language: "en" },
                    ],
                },
            }];

            const result = createICalFromSchedules(schedules);

            expect(result).toContain("BEGIN:VCALENDAR");
            expect(result).toContain("VERSION:2.0");
            expect(result).toContain("PRODID:-//Vrooli//Vrooli Calendar Service v1.0//EN");
            expect(result).toContain("BEGIN:VEVENT");
            expect(result).toContain("UID:schedule-1@vrooli.com");
            expect(result).toContain("SUMMARY:Team Meeting");
            expect(result).toContain("DESCRIPTION:Weekly team sync");
            expect(result).toContain("DTSTART;TZID=UTC:20240101T120000Z");
            expect(result).toContain("DTEND;TZID=UTC:20240101T130000Z");
            expect(result).toContain("END:VEVENT");
            expect(result).toContain("END:VCALENDAR");
        });

        it("should handle schedule with run instead of meeting", () => {
            const schedules = [{
                schedule: {
                    id: "schedule-2",
                    startTime: new Date("2024-01-01T12:00:00Z"),
                    endTime: new Date("2024-01-01T13:00:00Z"),
                    timezone: "America/New_York",
                    recurrences: [],
                } as any,
                run: {
                    name: "Data Processing",
                    resourceVersion: {
                        translations: [
                            { name: "Process Data", description: "Daily data processing", language: "en" },
                        ],
                    },
                },
            }];

            const result = createICalFromSchedules(schedules);

            expect(result).toContain("SUMMARY:Process Data");
            expect(result).toContain("DESCRIPTION:Daily data processing");
            expect(result).toContain("DTSTART;TZID=America/New_York:20240101T120000Z");
        });

        it("should handle schedule with recurrence", () => {
            const schedules = [{
                schedule: {
                    id: "schedule-3",
                    startTime: new Date("2024-01-01T12:00:00Z"),
                    endTime: new Date("2024-01-01T13:00:00Z"),
                    timezone: "UTC",
                    recurrences: [{
                        recurrenceType: ScheduleRecurrenceType.Weekly,
                        interval: 1,
                        dayOfWeek: 1, // Monday
                        duration: 60,
                        endDate: new Date("2024-12-31T12:00:00Z"),
                    }],
                } as any,
                meeting: {
                    translations: [{ name: "Weekly Standup", language: "en" }],
                },
            }];

            const result = createICalFromSchedules(schedules);

            expect(result).toContain("RRULE:FREQ=WEEKLY;BYDAY=MO;UNTIL=20241231T120000Z");
        });

        it("should escape special characters in text", () => {
            const schedules = [{
                schedule: {
                    id: "schedule-4",
                    startTime: new Date("2024-01-01T12:00:00Z"),
                    endTime: new Date("2024-01-01T13:00:00Z"),
                    timezone: "UTC",
                    recurrences: [],
                } as any,
                meeting: {
                    translations: [{
                        name: "Meeting with, special; characters\\and\nnewlines",
                        description: "Description with\r\nline breaks",
                        language: "en",
                    }],
                },
            }];

            const result = createICalFromSchedules(schedules);

            expect(result).toContain("SUMMARY:Meeting with\\, special\\; characters\\\\and\\nnewlines");
            expect(result).toContain("DESCRIPTION:Description with\\nline breaks");
        });

        it("should use default title when no meeting or run provided", () => {
            const schedules = [{
                schedule: {
                    id: "schedule-5",
                    startTime: new Date("2024-01-01T12:00:00Z"),
                    endTime: new Date("2024-01-01T13:00:00Z"),
                    timezone: "UTC",
                    recurrences: [],
                } as any,
            }];

            const result = createICalFromSchedules(schedules);

            expect(result).toContain("SUMMARY:Scheduled Event");
        });
    });

    describe("createICalEvent", () => {
        it("should create basic VCalendar event", () => {
            const schedule = {
                id: "event-1",
                name: "Test Event",
                startTime: new Date("2024-01-01T12:00:00Z"),
                endTime: new Date("2024-01-01T13:00:00Z"),
                timezone: "UTC",
            };

            const result = createICalEvent(schedule);

            expect(result.type).toBe("VCALENDAR");
            expect(result.prodid).toBe("-//Vrooli//Vrooli Calendar Service v1.0//EN");
            expect(result.version).toBe("2.0");
        });

        it("should handle event with recurrence", () => {
            const schedule = {
                id: "event-2",
                name: "Recurring Event",
                startTime: new Date("2024-01-01T12:00:00Z"),
                endTime: new Date("2024-01-01T13:00:00Z"),
                timezone: "UTC",
            };

            const recurrence = {
                recurrenceType: "weekly",
                interval: 2,
                dayOfWeek: "MO",
                endDate: new Date("2024-12-31T12:00:00Z"),
            };

            const result = createICalEvent(schedule, recurrence);

            expect(result.type).toBe("VCALENDAR");
            expect(result.prodid).toBe("-//Vrooli//Vrooli Calendar Service v1.0//EN");
        });
    });

    describe("Date formatting and escaping", () => {
        it("should format dates correctly for iCalendar", () => {
            const schedules = [{
                schedule: {
                    id: "date-test",
                    startTime: new Date("2024-01-01T12:30:45.123Z"),
                    endTime: new Date("2024-01-01T13:30:45.123Z"),
                    timezone: "UTC",
                    recurrences: [],
                } as any,
                meeting: {
                    translations: [{ name: "Date Test", language: "en" }],
                },
            }];

            const result = createICalFromSchedules(schedules);

            expect(result).toContain("DTSTART;TZID=UTC:20240101T123045Z");
            expect(result).toContain("DTEND;TZID=UTC:20240101T133045Z");
        });
    });

    describe("RRULE generation", () => {
        it("should generate monthly recurrence with day of month", () => {
            const schedules = [{
                schedule: {
                    id: "monthly-test",
                    startTime: new Date("2024-01-15T12:00:00Z"),
                    endTime: new Date("2024-01-15T13:00:00Z"),
                    timezone: "UTC",
                    recurrences: [{
                        recurrenceType: ScheduleRecurrenceType.Monthly,
                        interval: 1,
                        dayOfMonth: 15,
                        duration: 60,
                    }],
                } as any,
                meeting: {
                    translations: [{ name: "Monthly Event", language: "en" }],
                },
            }];

            const result = createICalFromSchedules(schedules);

            expect(result).toContain("RRULE:FREQ=MONTHLY;BYMONTHDAY=15");
        });

        it("should generate yearly recurrence with month", () => {
            const schedules = [{
                schedule: {
                    id: "yearly-test",
                    startTime: new Date("2024-06-01T12:00:00Z"),
                    endTime: new Date("2024-06-01T13:00:00Z"),
                    timezone: "UTC",
                    recurrences: [{
                        recurrenceType: ScheduleRecurrenceType.Yearly,
                        interval: 1,
                        month: 6,
                        duration: 60,
                    }],
                } as any,
                meeting: {
                    translations: [{ name: "Yearly Event", language: "en" }],
                },
            }];

            const result = createICalFromSchedules(schedules);

            expect(result).toContain("RRULE:FREQ=YEARLY;BYMONTH=6");
        });

        it("should handle recurrence with custom interval", () => {
            const schedules = [{
                schedule: {
                    id: "interval-test",
                    startTime: new Date("2024-01-01T12:00:00Z"),
                    endTime: new Date("2024-01-01T13:00:00Z"),
                    timezone: "UTC",
                    recurrences: [{
                        recurrenceType: ScheduleRecurrenceType.Daily,
                        interval: 3,
                        duration: 60,
                    }],
                } as any,
                meeting: {
                    translations: [{ name: "Every 3 Days", language: "en" }],
                },
            }];

            const result = createICalFromSchedules(schedules);

            expect(result).toContain("RRULE:FREQ=DAILY;INTERVAL=3");
        });
    });

    describe("Edge cases and error handling", () => {
        it("should handle empty schedules array", () => {
            const result = createICalFromSchedules([]);

            expect(result).toContain("BEGIN:VCALENDAR");
            expect(result).toContain("END:VCALENDAR");
            expect(result).not.toContain("BEGIN:VEVENT");
        });

        it("should handle missing translation fallback", () => {
            const schedules = [{
                schedule: {
                    id: "fallback-test",
                    startTime: new Date("2024-01-01T12:00:00Z"),
                    endTime: new Date("2024-01-01T13:00:00Z"),
                    timezone: "UTC",
                    recurrences: [],
                } as any,
                meeting: {
                    translations: [
                        { name: "German Event", description: "German description", language: "de" },
                    ],
                },
            }];

            const result = createICalFromSchedules(schedules);

            expect(result).toContain("SUMMARY:German Event");
            expect(result).toContain("DESCRIPTION:German description");
        });

        it("should handle run without resourceVersion", () => {
            const schedules = [{
                schedule: {
                    id: "simple-run",
                    startTime: new Date("2024-01-01T12:00:00Z"),
                    endTime: new Date("2024-01-01T13:00:00Z"),
                    timezone: "UTC",
                    recurrences: [],
                } as any,
                run: {
                    name: "Simple Run",
                },
            }];

            const result = createICalFromSchedules(schedules);

            expect(result).toContain("SUMMARY:Simple Run");
        });
    });
});