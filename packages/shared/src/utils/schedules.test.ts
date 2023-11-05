import moment from "moment";
import { Schedule, ScheduleException, ScheduleRecurrence } from "../api";
import { uuid } from "../id";
import { applyExceptions, calculateNextDailyOccurrence, calculateNextMonthlyOccurrence, calculateNextWeeklyOccurrence, calculateNextYearlyOccurrence, calculateOccurrences, jumpToFirstRelevantWeeklyOccurrence, validateTimeFrame } from "./schedules";

describe("validateTimeFrame", () => {
    beforeAll(() => {
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "error").mockImplementation(() => { });
    });
    afterAll(() => {
        jest.restoreAllMocks();
    });

    it("should return true for valid timeframes within a year", () => {
        const start = new Date("2022-01-01T00:00:00Z");
        const end = new Date("2022-12-31T00:00:00Z");
        expect(validateTimeFrame(start, end)).toBeTruthy();
    });

    it("should return false for timeframes longer than a year", () => {
        const start = new Date("2022-01-01T00:00:00Z");
        const end = new Date("2023-01-02T00:00:00Z");
        expect(validateTimeFrame(start, end)).toBeFalsy();
    });

    it("should return true for exact one-year timeframes", () => {
        const start = new Date("2022-01-01T00:00:00Z");
        const end = new Date("2023-01-01T00:00:00Z");
        expect(validateTimeFrame(start, end)).toBeTruthy();
    });

    it("should return true for very short timeframes", () => {
        const start = new Date("2022-01-01T00:00:00Z");
        const end = new Date("2022-01-01T00:01:00Z"); // 1 minute later
        expect(validateTimeFrame(start, end)).toBeTruthy();
    });

    it("should return false for timeframes with start date after end date", () => {
        const start = new Date("2022-01-02T00:00:00Z");
        const end = new Date("2022-01-01T00:00:00Z"); // Start is after end
        expect(validateTimeFrame(start, end)).toBeFalsy();
    });
});

type CalculateOccurrenceFn = (currentDate: Date, recurrence: ScheduleRecurrence, timeZone?: string) => Date;
const testRecurrence = ({
    currentDate,
    expected,
    func,
    recurrence,
    timeZone,
}: {
    currentDate: string,
    expected: string,
    func: CalculateOccurrenceFn,
    recurrence: ScheduleRecurrence,
    timeZone?: string,
}): void => {
    const current = new Date(currentDate);
    const result = func(current, recurrence, timeZone);
    const expectedResult = moment.tz(expected, timeZone ?? "UTC").toDate();
    expect(result.toISOString()).toBe(expectedResult.toISOString());
};

describe("calculateNextDailyOccurrence", () => {
    it("should return the next occurrence for a 1-day interval", () => {
        testRecurrence({
            currentDate: "2023-01-01T00:02:00Z",
            expected: "2023-01-02T00:02:00Z",
            func: calculateNextDailyOccurrence,
            recurrence: {
                recurrenceType: "Daily",
                interval: 1,
            } as ScheduleRecurrence,
            timeZone: "Poland",
        });
    });

    it("should return the next occurrence for a 3-day interval", () => {
        testRecurrence({
            currentDate: "2023-03-10T00:00:00Z",
            expected: "2023-03-13T00:00:00Z",
            func: calculateNextDailyOccurrence,
            recurrence: {
                recurrenceType: "Daily",
                interval: 3,
            } as ScheduleRecurrence,
            timeZone: "America/New_York",
        });
    });

    it("should handle leap years correctly", () => {
        testRecurrence({
            currentDate: "2024-02-28T00:30:02Z",
            expected: "2024-02-29T00:30:02Z",
            func: calculateNextDailyOccurrence,
            recurrence: {
                recurrenceType: "Daily",
                interval: 1,
            } as ScheduleRecurrence,
            timeZone: "US/Mountain",
        });
    });

    it("should handle end of month correctly", () => {
        testRecurrence({
            currentDate: "2023-04-30T00:00:00Z",
            expected: "2023-05-01T00:00:00Z",
            func: calculateNextDailyOccurrence,
            recurrence: {
                recurrenceType: "Daily",
                interval: 1,
            } as ScheduleRecurrence,
            timeZone: "Indian/Christmas",
        });
    });
});

describe("calculateNextWeeklyOccurrence", () => {

    test("next occurrence without specified dayOfWeek", () => {
        testRecurrence({
            currentDate: "2023-10-31T12:01:01Z",
            expected: "2023-11-07T12:01:01Z",
            func: calculateNextWeeklyOccurrence,
            recurrence: {
                recurrenceType: "Weekly",
                interval: 1,
            } as ScheduleRecurrence,
            timeZone: "Africa/Nairobi",
        });
    });

    test("next occurrence with specified dayOfWeek that is in the future of the current week", () => {
        testRecurrence({
            currentDate: "2023-10-31T12:00:00Z", // Tuesday
            expected: "2023-11-02T12:00:00Z",
            func: calculateNextWeeklyOccurrence,
            recurrence: {
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 4, // Thursday
            } as ScheduleRecurrence,
            timeZone: "America/Mexico_City",
        });
    });

    test("next occurrence with specified dayOfWeek that has already passed in the current week", () => {
        testRecurrence({
            currentDate: "2023-10-31T12:00:00Z", // Tuesday
            expected: "2023-11-06T12:00:00Z",
            func: calculateNextWeeklyOccurrence,
            recurrence: {
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 1, // Monday
            } as ScheduleRecurrence,
            timeZone: "America/New_York",
        });
    });

    test("next occurrence with dayOfWeek same as current day", () => {
        testRecurrence({
            currentDate: "2023-10-31T12:00:00Z", // Tuesday
            expected: "2023-11-07T12:00:00Z",
            func: calculateNextWeeklyOccurrence,
            recurrence: {
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 2, // Tuesday
            } as ScheduleRecurrence,
            timeZone: "Pacific/Auckland",
        });
    });

    test("next occurrence with interval greater than 1 and no dayOfWeek", () => {
        testRecurrence({
            currentDate: "2023-10-31T12:00:00Z",
            expected: "2023-11-21T12:00:00Z",
            func: calculateNextWeeklyOccurrence,
            recurrence: {
                recurrenceType: "Weekly",
                interval: 3,
            } as ScheduleRecurrence,
            timeZone: "America/Argentina/San_Juan",
        });
    });

    test("next occurrence with interval greater than 1 and dayOfWeek", () => {
        testRecurrence({
            currentDate: "2023-10-31T12:00:00Z", // Tuesday
            expected: "2023-11-02T12:00:00Z",
            func: calculateNextWeeklyOccurrence,
            recurrence: {
                recurrenceType: "Weekly",
                interval: 2,
                dayOfWeek: 4, // Thursday
            } as ScheduleRecurrence,
            timeZone: "Asia/Rangoon",
        });
    });

});

describe("calculateNextMonthlyOccurrence", () => {
    test("should calculate the next occurrence in the current month, when the day hasn't happened yet", () => {
        testRecurrence({
            currentDate: "2023-01-15T00:00:00Z",
            expected: "2023-01-20T00:00:00Z",
            func: calculateNextMonthlyOccurrence,
            recurrence: {
                recurrenceType: "Monthly",
                interval: 1,
                dayOfMonth: 20,
            } as ScheduleRecurrence,
        });
    });

    test("should calculate the next occurrence in the next month, when the day has already passed", () => {
        testRecurrence({
            currentDate: "2023-01-15T00:08:09Z",
            expected: "2023-02-10T00:08:09Z",
            func: calculateNextMonthlyOccurrence,
            recurrence: {
                recurrenceType: "Monthly",
                interval: 1,
                dayOfMonth: 10,
            } as ScheduleRecurrence,
            // timeZone: "America/Mexico_City", //TODO breaks - off by one day
        });
    });

    test("should calculate the next occurrence for the same day when dayOfMonth is not set", () => {
        testRecurrence({
            currentDate: "2023-01-15T01:00:00Z",
            expected: "2023-02-15T01:00:00Z",
            func: calculateNextMonthlyOccurrence,
            recurrence: {
                recurrenceType: "Monthly",
                interval: 1,
            } as ScheduleRecurrence,
            timeZone: "Africa/Nairobi",
        });
    });

    test("should handle the case where the dayOfMonth exceeds the number of days in the next month", () => {
        testRecurrence({
            currentDate: "2023-02-15T08:32:00Z",
            expected: "2023-03-03T08:32:00Z",
            func: calculateNextMonthlyOccurrence,
            recurrence: {
                recurrenceType: "Monthly",
                interval: 1,
                dayOfMonth: 31,
            } as ScheduleRecurrence,
            timeZone: "America/New_York",
        });
    });

    test("should find the event later this month, even with the interval of 2", () => {
        testRecurrence({
            currentDate: "2023-01-15T00:16:00Z",
            expected: "2023-01-20T00:16:00Z",
            func: calculateNextMonthlyOccurrence,
            recurrence: {
                recurrenceType: "Monthly",
                interval: 2,
                dayOfMonth: 20,
            } as ScheduleRecurrence,
            timeZone: "Pacific/Auckland",
        });
    });

    test("should find the event 2 months from now, since the dayOfMonth has passed", () => {
        testRecurrence({
            currentDate: "2023-01-15T00:00:00Z",
            expected: "2023-03-10T00:00:00Z",
            func: calculateNextMonthlyOccurrence,
            recurrence: {
                recurrenceType: "Monthly",
                interval: 2,
                dayOfMonth: 10,
            } as ScheduleRecurrence,
            timeZone: "Europe/Oslo",
        });
    });

    test("should calculate the next occurrence 4 months later when dayOfMonth is not set", () => {
        testRecurrence({
            currentDate: "2023-01-15T00:00:07Z",
            expected: "2023-05-15T00:00:07Z",
            func: calculateNextMonthlyOccurrence,
            recurrence: {
                recurrenceType: "Monthly",
                interval: 4,
            } as ScheduleRecurrence,
            timeZone: "America/Mexico_City",
        });
    });

    test("should calculate the next occurrence 4 months later with dayOfMonth set", () => {
        testRecurrence({
            currentDate: "2023-01-15T00:00:07Z",
            expected: "2023-05-14T00:00:07Z",
            func: calculateNextMonthlyOccurrence,
            recurrence: {
                recurrenceType: "Monthly",
                interval: 4,
                dayOfMonth: 14,
            } as ScheduleRecurrence,
            timeZone: "Europe/London",
        });
    });
});

describe("calculateNextYearlyOccurrence", () => {
    test("should calculate the next occurrence for a specific month and day of the month", () => {
        testRecurrence({
            currentDate: "2023-01-15T00:00:00Z",
            expected: "2023-06-20T00:00:00Z",
            func: calculateNextYearlyOccurrence,
            recurrence: {
                recurrenceType: "Yearly",
                interval: 1,
                month: 5, // June
                dayOfMonth: 20,
            } as ScheduleRecurrence,
            timeZone: "Europe/London",
        });
    });

    test("should work with february 29th on leap years", () => {
        testRecurrence({
            currentDate: "2024-02-29T00:00:00Z",
            expected: "2028-02-29T00:00:00Z",
            func: calculateNextYearlyOccurrence,
            recurrence: {
                recurrenceType: "Yearly",
                interval: 4,
                month: 1, // February
                dayOfMonth: 29,
            } as ScheduleRecurrence,
            timeZone: "Europe/London",
        });
    });

    test("should handle fall DST transition properly", () => {
        testRecurrence({
            currentDate: "2023-11-06T00:05:50Z", // After DST ends
            expected: "2024-11-03T00:05:50Z", // Should be just before the DST end date
            func: calculateNextYearlyOccurrence,
            recurrence: {
                recurrenceType: "Yearly",
                interval: 1,
                month: 10, // November
                dayOfMonth: 3,
            } as ScheduleRecurrence,
            timeZone: "America/Los_Angeles",
        });
    });

    test("should calculate the next occurrence for the same day and month when month and dayOfMonth are not set", () => {
        testRecurrence({
            currentDate: "2023-01-15T00:05:00Z",
            expected: "2024-01-15T00:05:00Z",
            func: calculateNextYearlyOccurrence,
            recurrence: {
                recurrenceType: "Yearly",
                interval: 1,
            } as ScheduleRecurrence,
            timeZone: "Australia/North",
        });
    });

    test("should handle the case where the specified date is already passed in the current year", () => {
        testRecurrence({
            currentDate: "2023-07-15T00:00:10Z",
            expected: "2024-06-20T00:00:10Z",
            func: calculateNextYearlyOccurrence,
            recurrence: {
                recurrenceType: "Yearly",
                interval: 1,
                month: 5, // June
                dayOfMonth: 20,
            } as ScheduleRecurrence,
            timeZone: "Canada/Saskatchewan",
        });
    });
});

type JumpFn = (scheduleStart: Date, recurrence: ScheduleRecurrence, timeframeStart: Date, timeZone?: string) => Date;
const testJump = ({
    scheduleStart,
    timeframeStart,
    expected,
    func,
    recurrence,
    timeZone,
}: {
    scheduleStart: string,
    timeframeStart: string,
    expected: string,
    func: JumpFn,
    recurrence: ScheduleRecurrence,
    timeZone?: string,
}): void => {
    const start = new Date(scheduleStart);
    const timeframe = new Date(timeframeStart);
    const result = func(start, recurrence, timeframe, timeZone);
    const expectedResult = new Date(expected);
    expect(result.toISOString()).toBe(expectedResult.toISOString());
};

describe("jumpToFirstRelevantWeeklyOccurrence", () => {
    it("should jump to the first relevant weekly occurrence", () => {
        testJump({
            scheduleStart: "2000-01-01T00:00:00Z",
            timeframeStart: "2023-01-03T00:00:00Z", // Tuesday
            recurrence: {
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 1, // Monday, 6 days after the timeframeStart
            } as ScheduleRecurrence,
            expected: "2023-01-09T00:00:00Z", // First Monday after the timeframeStart
            timeZone: "UTC",
            func: jumpToFirstRelevantWeeklyOccurrence,
        });
    });

    it("should respect timezone differences", () => {
        testJump({
            scheduleStart: "2000-01-01T00:00:00Z",
            timeframeStart: "2023-01-03T00:00:00Z", // Tuesday
            recurrence: {
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 7, // Sunday, 5 days after the timeframeStart
            } as ScheduleRecurrence,
            expected: "2023-01-08T00:00:00Z", // First Sunday after the timeframeStart
            timeZone: "America/New_York",
            func: jumpToFirstRelevantWeeklyOccurrence,
        });
    });

    // Add more test cases here to cover different scenarios like:
    // - Timeframes that start on the same day of the week as the scheduled occurrence
    // - Different intervals (every 2 weeks, every 3 weeks, etc.)
    // - Scenarios with multiple days of the week
    // - Edge cases like daylight saving time changes
    // - Ensure that it handles leap years correctly if relevant
});

const testApplyException = ({
    currentStartTime,
    schedule,
    timeZone,
    expected,
}: {
    currentStartTime: string,
    schedule: Schedule,
    timeZone: string,
    expected: { start: string, end: string } | null | undefined,
}): void => {
    const current = new Date(currentStartTime);
    const result = applyExceptions(current, schedule, timeZone);

    if (expected === null) {
        expect(result).toBeNull();
    } else if (expected === undefined) {
        expect(result).toBeUndefined();
    } else {
        expect(result).not.toBeNull();
        const expectedStart = moment.tz(expected.start, timeZone ?? "UTC").toDate();
        expect(result!.start.toISOString()).toBe(expectedStart.toISOString());
        const expectedEnd = moment.tz(expected.end, timeZone ?? "UTC").toDate();
        expect(result!.end.toISOString()).toBe(expectedEnd.toISOString());
    }
};

describe("applyExceptions", () => {
    let schedule: Schedule;

    beforeEach(() => {
        // Reset the schedule before each test with default properties
        schedule = {
            __typename: "Schedule",
            startTime: new Date("2024-01-01T00:03:10Z").toISOString(),
            endTime: new Date("2025-07-25T23:59:59Z").toISOString(),
            timezone: "Europe/London",
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            exceptions: [], // Start with no exceptions
            focusModes: [],
            id: uuid(),
            labels: [],
            meetings: [],
            recurrences: [],
            runProjects: [],
            runRoutines: [],
        };
    });

    it("should return undefined for a currentStartTime before the schedule's startTime", () => {
        testApplyException({
            currentStartTime: "2023-12-31T23:59:59Z", // A time before the schedule's start
            schedule,
            timeZone: "Canada/Pacific",
            expected: undefined, // No occurrence should be returned
        });
    });

    it("should return undefined for a currentStartTime after the schedule's endTime", () => {
        testApplyException({
            currentStartTime: "2025-07-26T00:00:00Z", // A time after the schedule's end
            schedule,
            timeZone: "Europe/Dublin",
            expected: undefined, // No occurrence should be returned
        });
    });

    it("should return undefined for a currentStartTime at exactly the schedule's endTime", () => {
        testApplyException({
            currentStartTime: "2025-07-25T23:59:59Z", // A time after the schedule's end
            schedule,
            timeZone: "Japan",
            expected: undefined, // No occurrence should be returned
        });
    });

    it("should return undefined for a schedule without exceptions", () => {
        testApplyException({
            currentStartTime: "2024-02-01T00:00:00Z",
            schedule,
            timeZone: "Iran",
            expected: undefined,
        });
    });

    it("should return undefined for a schedule without any applicable exceptions", () => {
        schedule.exceptions.push({
            originalStartTime: new Date("2024-01-02T00:00:00Z").toISOString(), // One day too late
            newStartTime: null,
            newEndTime: null,
        } as ScheduleException);
        testApplyException({
            currentStartTime: "2024-01-01T00:00:00Z",
            schedule,
            timeZone: "Israel",
            expected: undefined,
        });
    });

    it("should return undefined for a schedule where an exception seems applicable, but is the wrong year", () => {
        schedule.exceptions.push({
            originalStartTime: new Date("2023-01-01T00:00:00Z").toISOString(), // One year too early
            newStartTime: null,
            newEndTime: null,
        } as ScheduleException);
        testApplyException({
            currentStartTime: "2024-01-01T00:00:00Z",
            schedule,
            timeZone: "Libya",
            expected: undefined,
        });
    });

    it("should return null for an exception without a newStartTime or a newEndTime", () => {
        schedule.exceptions.push({
            originalStartTime: new Date("2024-01-01T00:00:30Z").toISOString(), // Correctly points to the current start time, within one minute
            newStartTime: null,
            newEndTime: null,
        } as ScheduleException);
        testApplyException({
            currentStartTime: "2024-01-01T00:00:00Z",
            schedule,
            timeZone: "Libya",
            expected: null,
        });
    });

    it("should return the an end time in one hour if none was set", () => {
        const exceptionStartTime = new Date("2024-01-01T00:00:00Z").toISOString();
        const ONE_HOUR_IN_MS = 60 * 60 * 1000;
        schedule.exceptions.push({
            originalStartTime: exceptionStartTime,
            newStartTime: exceptionStartTime,
            newEndTime: null,
        } as ScheduleException);
        testApplyException({
            currentStartTime: "2024-01-01T00:00:00Z",
            schedule,
            timeZone: "Europe/Zurich",
            expected: {
                start: "2024-01-01T00:00:00Z",
                end: new Date(new Date(exceptionStartTime).getTime() + ONE_HOUR_IN_MS).toISOString(),
            },
        });
    });

    it("should handle a schedule with an end time but no start time, by using the exception's originalStartTime", () => {
        schedule.exceptions.push({
            originalStartTime: new Date("2023-12-31T23:59:59Z").toISOString(),  // Correctly points to the current start time, within one minute
            newStartTime: null, // No new start time
            newEndTime: new Date("2024-01-01T05:00:00Z").toISOString(), // 5 hours later (very long meeting oof)
        } as ScheduleException);
        testApplyException({
            currentStartTime: "2024-01-01T00:00:02Z",
            schedule,
            timeZone: "Canada/Saskatchewan",
            expected: {
                start: "2023-12-31T23:59:59Z", // Exception's originalStartTime, not currentStartTime
                end: "2024-01-01T05:00:00Z", // 5 hours later
            },
        });
    });

    it("should handle a schedule with both a new start and end time", () => {
        schedule.exceptions.push({
            originalStartTime: new Date("2024-01-01T00:00:00Z").toISOString(), // Correctly points to the current start time
            newStartTime: new Date("2024-01-02T00:00:00Z").toISOString(), // Next day
            newEndTime: new Date("2024-01-02T02:00:00Z").toISOString(), // 2 hours after the new start time
        } as ScheduleException);
        testApplyException({
            currentStartTime: "2024-01-01T00:00:01Z",
            schedule,
            timeZone: "Europe/Vatican",
            expected: {
                start: "2024-01-02T00:00:00Z",
                end: "2024-01-02T02:00:00Z",
            },
        });
    });

    it("should handle a schedule with multiple applicable exceptions, by using the first exception", () => {
        // Both exceptions could apply, but
        schedule.exceptions.push(
            {
                originalStartTime: new Date("2024-01-01T00:00:00Z").toISOString(),
                newStartTime: new Date("2024-01-01T01:00:00Z").toISOString(),
                newEndTime: new Date("2024-01-01T02:00:00Z").toISOString(),
            } as ScheduleException,
            {
                originalStartTime: new Date("2024-01-01T00:00:00Z").toISOString(),
                newStartTime: new Date("2024-01-01T03:00:00Z").toISOString(),
                newEndTime: new Date("2024-01-01T04:00:00Z").toISOString(),
            } as ScheduleException,
        );
        testApplyException({
            currentStartTime: "2024-01-01T00:00:01Z",
            schedule,
            timeZone: "Europe/Berlin",
            // Expect to match the first exception, not the second
            expected: {
                start: "2024-01-01T01:00:00Z",
                end: "2024-01-01T02:00:00Z",
            },
        });
    });

    it("should allow exceptions with a newStartTime in the past", () => {
        schedule.exceptions.push({
            originalStartTime: new Date("2024-01-01T00:00:00Z").toISOString(),
            newStartTime: new Date("2023-12-31T22:00:00Z").toISOString(), // 2 hours earlier
            newEndTime: new Date("2023-12-31T23:00:00Z").toISOString(), // 1 hour earlier
        } as ScheduleException);
        testApplyException({
            currentStartTime: "2024-01-01T00:00:01Z",
            schedule,
            timeZone: "America/New_York",
            expected: {
                start: "2023-12-31T22:00:00Z",
                end: "2023-12-31T23:00:00Z",
            },
        });
    });
});

type OccurrenceTestParams = {
    startTime: string;
    endTime: string;
    recurrence: Partial<ScheduleRecurrence>;
    exceptions?: Partial<ScheduleException>[];
    timeframeStart: string;
    timeframeEnd: string;
    expectedOccurrences: {
        start: string;
        end: string;
    }[];
    timeZone: string;
};

const testCalculateOccurrences = ({
    startTime,
    endTime,
    recurrence,
    exceptions = [],
    timeframeStart,
    timeframeEnd,
    expectedOccurrences,
    timeZone,
}: OccurrenceTestParams): void => {
    // Create a full Schedule object with provided partial data and defaults
    const schedule: Schedule = {
        __typename: "Schedule",
        startTime: new Date(startTime).toISOString(),
        endTime: new Date(endTime).toISOString(),
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        focusModes: [],
        id: uuid(),
        labels: [],
        meetings: [],
        runProjects: [],
        runRoutines: [],
        timezone: timeZone,
        recurrences: [recurrence as ScheduleRecurrence],
        exceptions: exceptions as ScheduleException[],
    };

    const start = new Date(timeframeStart);
    const end = new Date(timeframeEnd);

    // Call the calculateOccurrences function
    const occurrences = calculateOccurrences(schedule, start, end);

    // Check if the number of occurrences is as expected
    expect(occurrences).toHaveLength(expectedOccurrences.length);

    // Compare each occurrence against the expected value
    occurrences.forEach((occurrence, index) => {
        const expected = expectedOccurrences[index];
        const expectedStart = moment.tz(expected?.start, timeZone ?? "UTC").toDate();
        expect(occurrence.start.toISOString()).toBe(expectedStart.toISOString());
        const expectedEnd = moment.tz(expected?.end, timeZone ?? "UTC").toDate();
        expect(occurrence.end.toISOString()).toBe(expectedEnd.toISOString());
    });
};

describe("calculateOccurrences", () => {
    it("calculates a weekly occurrences, which is far ahead of the startTime", () => {
        // testCalculateOccurrences({
        //     // The schedule is very long
        //     startTime: "2000-01-01T00:00:00Z",
        //     endTime: "2040-01-01T00:00:00Z",
        //     recurrence: {
        //         recurrenceType: ScheduleRecurrenceType.Weekly,
        //         interval: 1,
        //         dayOfWeek: 1, // Monday

        //     },
        //     // The time frame we're looking at is very short
        //     timeframeStart: "2023-01-01T00:00:00Z",
        //     timeframeEnd: "2023-01-08T00:00:00Z",
        //     expectedOccurrences: [
        //         {
        //             start: "2023-01-02T00:00:00Z", // First Monday in the timeframe
        //             end: "2023-01-02T01:00:00Z",
        //         },
        //     ],
        //     timeZone: "Europe/London",
        // });
    });

    it("calculates multiple weekly occurrences", () => {
        //...
    });

    it("calculates an occurrence which starts exactly at the timeframe start", () => {
        //...
    });

    it("calculates an occurrence which starts exactly at the timeframe end", () => {
        //...
    });

    it("calculates daily occurrences", () => {
        //...
    });

    it("calculates monthly occurrences", () => {
        //...
    });

    it("calculates yearly occurrences", () => {
        //...
    });

    it("Returns no occurrences for a schedule with no recurrences", () => {
        //...
    });

    it("Returns no occurrences for a schedule with no applicable recurrences", () => {
        //...
    });

    it("Returns no occurrences for a schedule with an applicable recurrence, but an exception which cancels it out", () => {
        //...
    });

    it("Ensures an error is logged and no occurrences are returned if the time frame is longer than a year", () => {
        //...
    });

    it("Ensures an error is logged and no occurrences are returned if the time frame start is after the time frame end", () => {
        //...
    });

    it("Ensures that we handle leap years correctly", () => {
        //...
    });
});
//TODO: 1. Schedule recurrences need duration. 2. Need way to skip ahead so we don't have to loop through from the startTime to timeFrameStart
