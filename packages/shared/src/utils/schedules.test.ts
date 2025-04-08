import { expect } from "chai";
import moment from "moment";
import sinon from "sinon";
import type { Schedule, ScheduleException, ScheduleRecurrence } from "../api/types.js";
import { HOURS_2_MS } from "../consts/numbers.js";
import { uuid } from "../id/uuid.js";
import { applyExceptions, calculateNextDailyOccurrence, calculateNextMonthlyOccurrence, calculateNextWeeklyOccurrence, calculateNextYearlyOccurrence, calculateOccurrences, jumpToFirstRelevantDailyOccurrence, jumpToFirstRelevantMonthlyOccurrence, jumpToFirstRelevantWeeklyOccurrence, jumpToFirstRelevantYearlyOccurrence, validateTimeFrame } from "./schedules.js";

describe("validateTimeFrame", () => {
    let consoleErrorStub: sinon.SinonStub;

    before(() => {
        consoleErrorStub = sinon.stub(console, "error");
    });

    beforeEach(() => {
        consoleErrorStub.resetHistory();
    });

    after(() => {
        consoleErrorStub.restore();
    });

    it("should return true for valid timeframes within a year", () => {
        const start = new Date("2022-01-01T00:00:00Z");
        const end = new Date("2022-12-31T00:00:00Z");
        expect(validateTimeFrame(start, end)).to.be.ok;
    });

    it("should return false for timeframes longer than a year", () => {
        const start = new Date("2022-01-01T00:00:00Z");
        const end = new Date("2023-01-02T00:00:00Z");
        expect(validateTimeFrame(start, end)).to.not.be.ok;
    });

    it("should return true for exact one-year timeframes", () => {
        const start = new Date("2022-01-01T00:00:00Z");
        const end = new Date("2023-01-01T00:00:00Z");
        expect(validateTimeFrame(start, end)).to.be.ok;
    });

    it("should return true for very short timeframes", () => {
        const start = new Date("2022-01-01T00:00:00Z");
        const end = new Date("2022-01-01T00:01:00Z"); // 1 minute later
        expect(validateTimeFrame(start, end)).to.be.ok;
    });

    it("should return false for timeframes with start date after end date", () => {
        const start = new Date("2022-01-02T00:00:00Z");
        const end = new Date("2022-01-01T00:00:00Z"); // Start is after end
        expect(validateTimeFrame(start, end)).to.not.be.ok;
    });
});

type CalculateOccurrenceFn = (currentDate: Date, recurrence: ScheduleRecurrence, timeZone?: string) => Promise<Date>;
async function testRecurrence({
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
}): Promise<void> {
    const current = new Date(currentDate);
    const result = await func(current, recurrence, timeZone);
    const expectedResult = moment.tz(expected, timeZone ?? "UTC").toDate();
    expect(result.toISOString()).to.equal(expectedResult.toISOString());
}

describe("calculateNextDailyOccurrence", () => {
    it("should return the next occurrence for a 1-day interval", async () => {
        await testRecurrence({
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

    it("should return the next occurrence for a 3-day interval", async () => {
        await testRecurrence({
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

    it("should handle leap years correctly", async () => {
        await testRecurrence({
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

    it("should handle end of month correctly", async () => {
        await testRecurrence({
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

    it("next occurrence without specified dayOfWeek", async () => {
        await testRecurrence({
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

    it("next occurrence with specified dayOfWeek that is in the future of the current week", async () => {
        await testRecurrence({
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

    it("next occurrence with specified dayOfWeek that has already passed in the current week", async () => {
        await testRecurrence({
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

    it("next occurrence with dayOfWeek same as current day", async () => {
        await testRecurrence({
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

    it("next occurrence with interval greater than 1 and no dayOfWeek", async () => {
        await testRecurrence({
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

    it("next occurrence with interval greater than 1 and dayOfWeek", async () => {
        await testRecurrence({
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
    it("should calculate the next occurrence in the current month, when the day hasn't happened yet", async () => {
        await testRecurrence({
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

    it("should calculate the next occurrence in the next month, when the day has already passed", async () => {
        await testRecurrence({
            currentDate: "2023-01-15T00:08:09Z",
            expected: "2023-02-10T00:08:09Z",
            func: calculateNextMonthlyOccurrence,
            recurrence: {
                recurrenceType: "Monthly",
                interval: 1,
                dayOfMonth: 10,
            } as ScheduleRecurrence,
            timeZone: "America/Mexico_City",
        });
    });

    it("should calculate the next occurrence for the same day when dayOfMonth is not set", async () => {
        await testRecurrence({
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

    it("should handle the case where the dayOfMonth exceeds the number of days in the month", async () => {
        await testRecurrence({
            currentDate: "2023-02-15T08:32:00Z",
            expected: "2023-02-28T08:32:00Z",
            func: calculateNextMonthlyOccurrence,
            recurrence: {
                recurrenceType: "Monthly",
                interval: 1,
                dayOfMonth: 31,
            } as ScheduleRecurrence,
            timeZone: "America/New_York",
        });
    });

    it("should find the event later this month, even with the interval of 2", async () => {
        await testRecurrence({
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

    it("should find the event 2 months from now, since the dayOfMonth has passed", async () => {
        await testRecurrence({
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

    it("should calculate the next occurrence 4 months later when dayOfMonth is not set", async () => {
        await testRecurrence({
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

    it("should calculate the next occurrence 4 months later with dayOfMonth set", async () => {
        await testRecurrence({
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
    it("should calculate the next occurrence for a specific month and day of the month", async () => {
        await testRecurrence({
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

    it("should work with february 29th on leap years", async () => {
        await testRecurrence({
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

    it("should handle the case where the dayOfMonth exceeds the number of days in the month", async () => {
        await testRecurrence({
            currentDate: "2023-02-15T08:32:00Z",
            expected: "2023-02-28T08:32:00Z",
            func: calculateNextYearlyOccurrence,
            recurrence: {
                recurrenceType: "Yearly",
                interval: 1,
                month: 1, // February
                dayOfMonth: 31,
            } as ScheduleRecurrence,
        });
    });

    it("should handle fall DST transition properly", async () => {
        await testRecurrence({
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

    it("should calculate the next occurrence for the same day and month when month and dayOfMonth are not set", async () => {
        await testRecurrence({
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

    it("should handle the case where the specified date is already passed in the current year", async () => {
        await testRecurrence({
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

type JumpFn = (scheduleStart: Date, recurrence: ScheduleRecurrence, timeframeStart: Date, timeZone?: string) => Promise<Date>;
async function testJump({
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
}): Promise<void> {
    const start = new Date(scheduleStart);
    const timeframe = new Date(timeframeStart);
    const result = await func(start, recurrence, timeframe, timeZone);
    const expectedResult = new Date(expected);
    expect(result.toISOString()).to.equal(expectedResult.toISOString());
}

describe("jumpToFirstRelevantDailyOccurrence", () => {
    let consoleErrorStub: sinon.SinonStub;

    before(() => {
        consoleErrorStub = sinon.stub(console, "error");
    });

    beforeEach(() => {
        consoleErrorStub.resetHistory();
    });

    after(() => {
        consoleErrorStub.restore();
    });

    it("should jump to the first relevant daily occurrence", async () => {
        await testJump({
            scheduleStart: "2000-01-01T00:00:00Z",
            timeframeStart: "2023-01-03T00:00:00Z",
            recurrence: {
                recurrenceType: "Daily",
                interval: 1,
            } as ScheduleRecurrence,
            expected: "2023-01-03T00:00:00Z",
            timeZone: "UTC",
            func: jumpToFirstRelevantDailyOccurrence,
        });
    });

    it("should respect timezone differences", async () => {
        await testJump({
            scheduleStart: "2000-01-01T13:00:00Z",
            timeframeStart: "2023-01-03T12:00:00Z", // Adjust for time zone difference
            recurrence: {
                recurrenceType: "Daily",
                interval: 1,
            } as ScheduleRecurrence,
            expected: "2023-01-03T13:00:00Z",
            timeZone: "America/New_York",
            func: jumpToFirstRelevantDailyOccurrence,
        });
    });

    it("should handle the schedule start being the same as the time frame start", async () => {
        await testJump({
            scheduleStart: "2023-01-01T01:02:03Z",
            timeframeStart: "2023-01-01T00:00:00Z",
            recurrence: {
                recurrenceType: "Daily",
                interval: 1,
            } as ScheduleRecurrence,
            expected: "2023-01-01T01:02:03Z",
            timeZone: "UTC",
            func: jumpToFirstRelevantDailyOccurrence,
        });
    });

    it("should use the schedule start when the time frame is before the schedule", async () => {
        await testJump({
            scheduleStart: "2023-01-10T00:00:00Z",
            timeframeStart: "2023-01-03T00:00:00Z",
            recurrence: {
                recurrenceType: "Daily",
                interval: 1,
            } as ScheduleRecurrence,
            expected: "2023-01-10T00:00:00Z",
            timeZone: "UTC",
            func: jumpToFirstRelevantDailyOccurrence,
        });
        await testJump({
            scheduleStart: "2023-01-10T00:00:00Z",
            timeframeStart: "2023-01-03T00:00:00Z",
            recurrence: {
                recurrenceType: "Daily",
                interval: 5,
            } as ScheduleRecurrence,
            expected: "2023-01-10T00:00:00Z", // Still the same as the previous test, as the interval begins on the schedule start
            timeZone: "UTC",
            func: jumpToFirstRelevantDailyOccurrence,
        });
    });

    it("should work with different intervals", async () => {
        await testJump({
            scheduleStart: "2000-01-01T00:00:00Z",
            timeframeStart: "2023-01-02T00:00:00Z",
            recurrence: {
                recurrenceType: "Daily",
                interval: 2, // Every other day
            } as ScheduleRecurrence,
            expected: "2023-01-02T00:00:00Z", // Should be the first day in the timeframe on the correct interval
            timeZone: "UTC",
            func: jumpToFirstRelevantDailyOccurrence,
        });
        await testJump({
            scheduleStart: "2000-01-01T00:00:00Z",
            timeframeStart: "2023-01-01T00:00:00Z", // Day before the previous timeframe start
            recurrence: {
                recurrenceType: "Daily",
                interval: 2, // Every other day
            } as ScheduleRecurrence,
            expected: "2023-01-02T00:00:00Z", // Should be the same as the previous test because of the interval
            timeZone: "UTC",
            func: jumpToFirstRelevantDailyOccurrence,
        });
    });

    it("should handle daylight savings changes", async () => {
        await testJump({
            scheduleStart: "2023-03-01T00:00:00Z",
            timeframeStart: "2023-03-14T00:00:00Z", // After daylight savings time starts
            recurrence: {
                recurrenceType: "Daily",
                interval: 1,
            } as ScheduleRecurrence,
            expected: "2023-03-14T00:00:00Z",
            timeZone: "America/New_York",
            func: jumpToFirstRelevantDailyOccurrence,
        });
    });

    it("should handle leap years correctly", async () => {
        await testJump({
            scheduleStart: "2020-02-24T00:00:00Z",
            timeframeStart: "2024-02-29T00:00:00Z", // Leap day
            recurrence: {
                recurrenceType: "Daily",
                interval: 1,
            } as ScheduleRecurrence,
            expected: "2024-02-29T00:00:00Z",
            timeZone: "UTC",
            func: jumpToFirstRelevantDailyOccurrence,
        });
    });
});

describe("jumpToFirstRelevantWeeklyOccurrence", () => {
    let consoleErrorStub: sinon.SinonStub;

    before(() => {
        consoleErrorStub = sinon.stub(console, "error");
    });

    beforeEach(() => {
        consoleErrorStub.resetHistory();
    });

    after(() => {
        consoleErrorStub.restore();
    });

    it("should jump to the first relevant weekly occurrence", async () => {
        await testJump({
            scheduleStart: "0000-01-01T00:00:00Z",
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

    it("should respect timezone differences", async () => {
        await testJump({
            scheduleStart: "2000-01-01T13:00:00Z",
            timeframeStart: "2023-01-03T13:00:00Z", // Tuesday
            recurrence: {
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 7, // Sunday, 5 days after the timeframeStart
            } as ScheduleRecurrence,
            expected: "2023-01-08T13:00:00Z", // First Sunday after the timeframeStart
            timeZone: "America/New_York",
            func: jumpToFirstRelevantWeeklyOccurrence,
        });
    });

    it("should handle the schedule start being the same as the time frame start", async () => {
        await testJump({
            scheduleStart: "2023-01-01T00:04:20Z",
            timeframeStart: "2023-01-01T00:04:20Z",
            recurrence: {
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 1, // Monday
            } as ScheduleRecurrence,
            expected: "2023-01-02T00:04:20Z",
            timeZone: "UTC",
            func: jumpToFirstRelevantWeeklyOccurrence,
        });
    });

    it("should return the schedule start when the time frame is before the schedule", async () => {
        await testJump({
            scheduleStart: "2023-01-10T00:00:00Z",
            timeframeStart: "2023-01-03T00:00:00Z",
            recurrence: {
                recurrenceType: "Weekly",
                interval: 1,
            } as ScheduleRecurrence,
            expected: "2023-01-10T00:00:00Z",
            timeZone: "UTC",
            func: jumpToFirstRelevantWeeklyOccurrence,
        });
    });

    it("should work with different intervals", async () => {
        await testJump({
            scheduleStart: "2000-01-01T00:00:00Z", // Sunday
            timeframeStart: "2023-01-05T00:00:00Z", // Thursday
            recurrence: {
                recurrenceType: "Weekly",
                interval: 2,
                dayOfWeek: 3, // Wednesday
            } as ScheduleRecurrence,
            expected: "2023-01-18T00:00:00Z", // The second Wednesday in the timeframe
            timeZone: "UTC",
            func: jumpToFirstRelevantWeeklyOccurrence,
        });
        await testJump({
            scheduleStart: "2000-01-01T00:00:00Z", // Same as above
            timeframeStart: "2023-01-12T00:00:00Z", // Later Thursday
            recurrence: {
                recurrenceType: "Weekly",
                interval: 2,
                dayOfWeek: 3, // Wednesday
            } as ScheduleRecurrence,
            expected: "2023-01-18T00:00:00Z", // The first Wednesday in the timeframe, but should still be correct
            timeZone: "UTC",
            func: jumpToFirstRelevantWeeklyOccurrence,
        });
    });

    it("should handle daylight savings", async () => {
        await testJump({
            scheduleStart: "2023-03-01T00:00:00Z",
            timeframeStart: "2023-03-14T00:00:00Z", // After daylight savings time starts
            recurrence: {
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 2, // Tuesday
            } as ScheduleRecurrence,
            expected: "2023-03-14T00:00:00Z", // Should remain the same regardless of daylight savings
            timeZone: "America/New_York",
            func: jumpToFirstRelevantWeeklyOccurrence,
        });
    });

    it("should handle leap years", async () => {
        await testJump({
            scheduleStart: "2020-02-24T00:00:00Z", // A leap year Monday
            timeframeStart: "2024-02-29T00:00:00Z", // Leap day on Friday
            recurrence: {
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 1, // Looking for the next Monday
            } as ScheduleRecurrence,
            expected: "2024-03-04T00:00:00Z", // The Monday after leap day in 2024
            timeZone: "UTC",
            func: jumpToFirstRelevantWeeklyOccurrence,
        });
    });
});

describe("jumpToFirstRelevantMonthlyOccurrence", () => {
    let consoleErrorStub: sinon.SinonStub;

    before(() => {
        consoleErrorStub = sinon.stub(console, "error");
    });

    beforeEach(() => {
        consoleErrorStub.resetHistory();
    });

    after(() => {
        consoleErrorStub.restore();
    });

    it("should jump to the first relevant monthly occurrence", async () => {
        await testJump({
            scheduleStart: "2000-01-01T10:02:03Z",
            timeframeStart: "2023-01-15T00:00:00Z",
            recurrence: {
                recurrenceType: "Monthly",
                interval: 1,
                dayOfMonth: 15,
            } as ScheduleRecurrence,
            expected: "2023-01-15T10:02:03Z",
            timeZone: "UTC",
            func: jumpToFirstRelevantMonthlyOccurrence,
        });
    });

    it("should handle monthly intervals correctly", async () => {
        await testJump({
            scheduleStart: "2020-01-15T00:00:00Z",
            timeframeStart: "2023-04-10T00:00:00Z",
            recurrence: {
                recurrenceType: "Monthly",
                interval: 3,
                dayOfMonth: 15,
            } as ScheduleRecurrence,
            expected: "2023-04-15T00:00:00Z", // The next 15th after the timeframeal
            timeZone: "UTC",
            func: jumpToFirstRelevantMonthlyOccurrence,
        });
        await testJump({
            scheduleStart: "2020-01-15T00:00:00Z",
            timeframeStart: "2023-03-10T00:00:00Z", // The same as the previous test, but one month earlier
            recurrence: {
                recurrenceType: "Monthly",
                interval: 3, // Every third
                dayOfMonth: 15,
            } as ScheduleRecurrence,
            expected: "2023-04-15T00:00:00Z", // The same result as the previous test, since we're on an interval
            timeZone: "UTC",
            func: jumpToFirstRelevantMonthlyOccurrence,
        });
    });

    it("should handle end of month dates correctly", async () => {
        await testJump({
            scheduleStart: "2020-01-31T00:00:00Z",
            timeframeStart: "2023-02-15T00:00:00Z",
            recurrence: {
                recurrenceType: "Monthly",
                interval: 1,
                dayOfMonth: 31,
            } as ScheduleRecurrence,
            expected: "2023-02-28T00:00:00Z", // February only has 28 days
            timeZone: "UTC",
            func: jumpToFirstRelevantMonthlyOccurrence,
        });
    });

    it("should use the schedule start when the timeframe is before the schedule for monthly recurrence", async () => {
        await testJump({
            scheduleStart: "2023-01-15T00:00:00Z",
            timeframeStart: "2022-01-10T00:00:00Z",
            recurrence: {
                recurrenceType: "Monthly",
                interval: 1,
                dayOfMonth: 17,
            } as ScheduleRecurrence,
            expected: "2023-01-17T00:00:00Z",
            timeZone: "UTC",
            func: jumpToFirstRelevantMonthlyOccurrence,
        });
    });
});

describe("jumpToFirstRelevantYearlyOccurrence", () => {
    let consoleErrorStub: sinon.SinonStub;

    before(() => {
        consoleErrorStub = sinon.stub(console, "error");
    });

    beforeEach(() => {
        consoleErrorStub.resetHistory();
    });

    after(() => {
        consoleErrorStub.restore();
    });

    it("should jump to the first relevant yearly occurrence", async () => {
        await testJump({
            scheduleStart: "2010-04-15T00:00:00Z",
            timeframeStart: "2023-01-01T00:00:00Z",
            recurrence: {
                recurrenceType: "Yearly",
                interval: 1,
                month: 4,
                dayOfMonth: 11,
            } as ScheduleRecurrence,
            expected: "2023-05-11T00:00:00Z",
            timeZone: "UTC",
            func: jumpToFirstRelevantYearlyOccurrence,
        });
    });

    it("should handle leap years correctly", async () => {
        await testJump({
            scheduleStart: "2016-02-29T00:00:00Z",
            timeframeStart: "2023-01-01T00:00:00Z",
            recurrence: {
                recurrenceType: "Yearly",
                interval: 1,
                month: 1,
                dayOfMonth: 29,
            } as ScheduleRecurrence,
            expected: "2023-02-28T00:00:00Z", // The first year doesn't have a leap day
            timeZone: "UTC",
            func: jumpToFirstRelevantYearlyOccurrence,
        });
        await testJump({
            scheduleStart: "2016-02-29T00:00:00Z",
            timeframeStart: "2024-01-01T00:00:00Z",
            recurrence: {
                recurrenceType: "Yearly",
                interval: 1,
                month: 1,
                dayOfMonth: 29,
            } as ScheduleRecurrence,
            expected: "2024-02-29T00:00:00Z", // The first year does have a leap day
            timeZone: "UTC",
            func: jumpToFirstRelevantYearlyOccurrence,
        });
    });

    it("should respect multi-year intervals", async () => {
        await testJump({
            scheduleStart: "0010-01-01T23:00:06Z",
            timeframeStart: "2023-01-01T00:00:00Z",
            recurrence: {
                recurrenceType: "Yearly",
                interval: 5, // Every five years
                month: 1,
                dayOfMonth: 1,
            } as ScheduleRecurrence,
            expected: "2025-02-01T23:00:06Z", // The next occurrence in the 5-year cycle
            timeZone: "UTC",
            func: jumpToFirstRelevantYearlyOccurrence,
        });
    });

    it("should handle end of month dates correctly", async () => {
        await testJump({
            scheduleStart: "1492-01-31T00:00:00Z",
            timeframeStart: "2023-02-15T00:00:00Z",
            recurrence: {
                recurrenceType: "Yearly",
                interval: 1,
                month: 1,
                dayOfMonth: 31,
            } as ScheduleRecurrence,
            expected: "2023-02-28T00:00:00Z", // February only has 28 days
            timeZone: "UTC",
            func: jumpToFirstRelevantYearlyOccurrence,
        });
    });

    it("should use the schedule start when the timeframe is before the schedule for yearly recurrence", async () => {
        await testJump({
            scheduleStart: "2023-04-15T01:04:20Z",
            timeframeStart: "1874-01-01T00:00:00Z",
            recurrence: {
                recurrenceType: "Yearly",
                interval: 1,
                month: 3,
                dayOfMonth: 17,
            } as ScheduleRecurrence,
            expected: "2023-04-17T01:04:20Z",
            timeZone: "UTC",
            func: jumpToFirstRelevantYearlyOccurrence,
        });
    });
});

function testApplyException({
    currentStartTime,
    schedule,
    duration,
    timeZone,
    expected,
}: {
    currentStartTime: string,
    schedule: Schedule,
    duration: number,
    timeZone: string,
    expected: { start: string, end: string } | null | undefined,
}): void {
    const current = new Date(currentStartTime);
    const result = applyExceptions(current, schedule, duration, timeZone);

    if (expected === null) {
        expect(result).to.be.null;
    } else if (expected === undefined) {
        expect(result).to.be.undefined;
    } else {
        expect(result).not.to.be.null;
        const expectedStart = moment.tz(expected.start, timeZone ?? "UTC").toDate();
        expect(result!.start.toISOString()).to.equal(expectedStart.toISOString());
        const expectedEnd = moment.tz(expected.end, timeZone ?? "UTC").toDate();
        expect(result!.end.toISOString()).to.equal(expectedEnd.toISOString());
    }
}

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
            duration: 60,
            timeZone: "Canada/Pacific",
            expected: undefined, // No occurrence should be returned
        });
    });

    it("should return undefined for a currentStartTime after the schedule's endTime", () => {
        testApplyException({
            currentStartTime: "2025-07-26T00:00:00Z", // A time after the schedule's end
            schedule,
            duration: 10000,
            timeZone: "Europe/Dublin",
            expected: undefined, // No occurrence should be returned
        });
    });

    it("should return undefined for a currentStartTime at exactly the schedule's endTime", () => {
        testApplyException({
            currentStartTime: "2025-07-25T23:59:59Z", // A time after the schedule's end
            schedule,
            duration: 10,
            timeZone: "Japan",
            expected: undefined, // No occurrence should be returned
        });
    });

    it("should return undefined for a schedule without exceptions", () => {
        testApplyException({
            currentStartTime: "2024-02-01T00:01:00Z",
            schedule,
            duration: 40,
            timeZone: "Iran",
            expected: undefined,
        });
    });

    it("should return undefined for a schedule without any applicable exceptions", () => {
        schedule.exceptions.push({
            originalStartTime: new Date("2024-01-02T10:00:00Z").toISOString(), // One day too late
            newStartTime: null,
            newEndTime: null,
        } as ScheduleException);
        testApplyException({
            currentStartTime: "2024-01-01T00:00:00Z",
            schedule,
            duration: 90000,
            timeZone: "Israel",
            expected: undefined,
        });
    });

    it("should return undefined for a schedule where an exception seems applicable, but is the wrong year", () => {
        schedule.exceptions.push({
            originalStartTime: new Date("2023-01-01T00:01:00Z").toISOString(), // One year too early
            newStartTime: null,
            newEndTime: null,
        } as ScheduleException);
        testApplyException({
            currentStartTime: "2024-01-01T00:00:00Z",
            schedule,
            duration: 420,
            timeZone: "Libya",
            expected: undefined,
        });
    });

    it("should return null for an exception without a newStartTime or a newEndTime", () => {
        schedule.exceptions.push({
            originalStartTime: new Date("2024-01-01T00:00:31Z").toISOString(), // Correctly points to the current start time, within one minute
            newStartTime: null,
            newEndTime: null,
        } as ScheduleException);
        testApplyException({
            currentStartTime: "2024-01-01T00:00:00Z",
            schedule,
            duration: 69,
            timeZone: "Libya",
            expected: null,
        });
    });

    it("should return the end time using duration if no explicit end time was set", () => {
        const exceptionStartTime = new Date("2024-01-01T00:00:11Z").toISOString();
        schedule.exceptions.push({
            originalStartTime: exceptionStartTime,
            newStartTime: exceptionStartTime,
            newEndTime: null,
        } as ScheduleException);
        testApplyException({
            currentStartTime: "2024-01-01T00:00:00Z",
            schedule,
            duration: HOURS_2_MS,
            timeZone: "Europe/Zurich",
            expected: {
                start: "2024-01-01T00:00:11Z",
                end: new Date(new Date(exceptionStartTime).getTime() + HOURS_2_MS).toISOString(),
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
            duration: 10000,
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
            duration: 420,
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
            duration: 69,
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
            duration: 20,
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
    recurrences: Partial<ScheduleRecurrence>[];
    exceptions?: Partial<ScheduleException>[];
    timeframeStart: string;
    timeframeEnd: string;
    expectedOccurrences: {
        start: string;
        end: string;
    }[];
    timeZone: string;
};

async function testCalculateOccurrences({
    startTime,
    endTime,
    recurrences,
    exceptions = [],
    timeframeStart,
    timeframeEnd,
    expectedOccurrences,
    timeZone,
}: OccurrenceTestParams): Promise<void> {
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
        recurrences: recurrences as ScheduleRecurrence[],
        exceptions: exceptions as ScheduleException[],
    };

    const start = new Date(timeframeStart);
    const end = new Date(timeframeEnd);

    // Call the calculateOccurrences function
    const occurrences = await calculateOccurrences(schedule, start, end);

    // Check if the number of occurrences is as expected
    expect(occurrences).to.have.lengthOf(expectedOccurrences.length);

    // Compare each occurrence against the expected value
    occurrences.forEach((occurrence, index) => {
        const expected = expectedOccurrences[index];
        const expectedStart = moment.tz(expected?.start, timeZone ?? "UTC").toDate();
        expect(occurrence.start.toISOString()).to.equal(expectedStart.toISOString());
        const expectedEnd = moment.tz(expected?.end, timeZone ?? "UTC").toDate();
        expect(occurrence.end.toISOString()).to.equal(expectedEnd.toISOString());
    });
}

describe("calculateOccurrences", () => {
    let consoleErrorStub: sinon.SinonStub;

    before(() => {
        consoleErrorStub = sinon.stub(console, "error");
    });

    beforeEach(() => {
        consoleErrorStub.resetHistory();
    });

    after(() => {
        consoleErrorStub.restore();
    });

    it("calculates daily occurrence", async () => {
        await testCalculateOccurrences({
            // The schedule is very long
            startTime: "2023-04-20T12:12:12Z",
            endTime: "9999-01-01T00:00:00Z",
            recurrences: [{
                recurrenceType: "Daily",
                interval: 15,
                duration: 120, // 2 hours in minutes
            }] as ScheduleRecurrence[],
            timeframeStart: "2023-01-01T00:00:00Z",
            timeframeEnd: "2023-04-30T23:59:37Z",
            expectedOccurrences: [
                {
                    start: "2023-04-20T12:12:12Z", // startTime, since startTime is after timeframeStart
                    end: "2023-04-20T14:12:12Z", // 2 hours later
                },
            ],
            timeZone: "Europe/London",
        });
    });

    it("calculates weekly occurrences", async () => {
        await testCalculateOccurrences({
            // The schedule is very long
            startTime: "0720-01-01T00:00:00Z",
            endTime: "2040-01-01T00:00:00Z",
            recurrences: [{
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 1, // Monday

            }] as ScheduleRecurrence[],
            // The time frame we're looking at is very short
            timeframeStart: "2023-01-01T00:00:00Z",
            timeframeEnd: "2023-01-08T00:00:00Z",
            expectedOccurrences: [
                {
                    start: "2023-01-02T00:00:00Z", // First Monday in the timeframe
                    end: "2023-01-02T01:00:00Z",
                },
            ],
            timeZone: "Europe/London",
        });
    });

    it("calculates multiple weekly occurrences", async () => {
        await testCalculateOccurrences({
            startTime: "2023-01-01T00:00:10Z",
            endTime: "2023-12-31T00:00:00Z",
            recurrences: [{
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 3, // Wednesday
                duration: 120, // 2 hours in minutes
            }] as ScheduleRecurrence[],
            timeframeStart: "2023-02-28T00:00:00Z",
            timeframeEnd: "2023-04-01T00:00:00Z",
            expectedOccurrences: [
                { start: "2023-03-01T00:00:10Z", end: "2023-03-01T02:00:10Z" },
                { start: "2023-03-08T00:00:10Z", end: "2023-03-08T02:00:10Z" },
                { start: "2023-03-15T00:00:10Z", end: "2023-03-15T02:00:10Z" },
                { start: "2023-03-22T00:00:10Z", end: "2023-03-22T02:00:10Z" },
                { start: "2023-03-29T00:00:10Z", end: "2023-03-29T02:00:10Z" },
            ],
            timeZone: "Brazil/West",
        });
    });


    it("calculates monthly occurrences", async () => {
        await testCalculateOccurrences({
            startTime: "2023-01-01T00:00:00Z",
            endTime: "2023-12-31T00:00:00Z",
            recurrences: [{
                recurrenceType: "Monthly",
                interval: 1,
                dayOfMonth: 1,
                duration: 120, // 2 hours in minutes
            }] as ScheduleRecurrence[],
            timeframeStart: "2023-01-01T00:00:00Z",
            timeframeEnd: "2023-04-01T00:00:00Z",
            expectedOccurrences: [
                { start: "2023-01-01T00:00:00Z", end: "2023-01-01T02:00:00Z" },
                { start: "2023-02-01T00:00:00Z", end: "2023-02-01T02:00:00Z" },
                { start: "2023-03-01T00:00:00Z", end: "2023-03-01T02:00:00Z" },
                { start: "2023-04-01T00:00:00Z", end: "2023-04-01T02:00:00Z" },
            ],
            timeZone: "US/Alaska",
        });
    });

    it("calculates yearly occurrences", async () => {
        await testCalculateOccurrences({
            startTime: "2023-01-01T00:14:00Z",
            endTime: "2023-12-31T00:00:00Z",
            recurrences: [{
                recurrenceType: "Yearly",
                interval: 1,
                month: 0,
                dayOfMonth: 2,
                duration: 181, // 3 hours and 1 minute in minutes
            }] as ScheduleRecurrence[],
            timeframeStart: "2023-01-01T00:00:00Z",
            timeframeEnd: "2023-04-01T00:00:00Z",
            expectedOccurrences: [
                { start: "2023-01-02T00:14:00Z", end: "2023-01-02T03:15:00Z" },
            ],
            timeZone: "UTC",
        });
    });

    it("calculates an occurrence which starts exactly at the timeframe start", async () => {
        await testCalculateOccurrences({
            startTime: "2023-01-01T00:00:00Z",
            endTime: "2023-06-01T00:00:00Z", // Schedule ends on June 1st, 2023
            recurrences: [{
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 5, // Friday
                endDate: "2023-02-15T00:00:00Z", // Recurrence ends earlier on Feb 15th, 2023
                // No duration, so it should default to 1 hour
            }] as ScheduleRecurrence[],
            timeframeStart: "2023-01-01T00:00:00Z",
            timeframeEnd: "2023-12-31T00:00:00Z",
            expectedOccurrences: [
                // Occurrences every Friday from Jan 1st until recurrence endDate
                { start: "2023-01-06T00:00:00Z", end: "2023-01-06T01:00:00Z" },
                { start: "2023-01-13T00:00:00Z", end: "2023-01-13T01:00:00Z" },
                { start: "2023-01-20T00:00:00Z", end: "2023-01-20T01:00:00Z" },
                { start: "2023-01-27T00:00:00Z", end: "2023-01-27T01:00:00Z" },
                { start: "2023-02-03T00:00:00Z", end: "2023-02-03T01:00:00Z" },
                { start: "2023-02-10T00:00:00Z", end: "2023-02-10T01:00:00Z" },
            ],
            timeZone: "Egypt",
        });
    });

    it("calculates an occurrence which starts exactly at the timeframe end", async () => {
        await testCalculateOccurrences({
            startTime: "2023-01-01T13:00:00Z",
            endTime: "2023-06-02T13:00:00Z",
            recurrences: [{
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 5, // Friday
                duration: 5, // 5 minutes
            }] as ScheduleRecurrence[],
            timeframeStart: "2023-05-27T00:00:00Z",
            timeframeEnd: "2023-06-02T13:00:00Z", // End of the timeframe is on June 2nd
            expectedOccurrences: [
                // Occurrence starts right at the end of the timeframe
                { start: "2023-06-02T13:00:00Z", end: "2023-06-02T13:05:00Z" },
            ],
            timeZone: "US/Michigan",
        });

    });

    it("Returns no occurrences for a schedule with no applicable recurrences", async () => {
        // Same as previous test, but the recurrence ends just before the occurrence would start
        await testCalculateOccurrences({
            startTime: "1923-01-01T13:00:00Z",
            endTime: "2023-06-02T13:00:00Z",
            recurrences: [{
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 5, // Friday
                duration: 5, // 5 minutes
                endDate: new Date("2023-06-02T12:55:00Z").toISOString(), // Just before the occurrence would start
            }] as ScheduleRecurrence[],
            timeframeStart: "2023-05-27T00:00:00Z",
            timeframeEnd: "2023-06-02T13:00:00Z", // End of the timeframe is on June 2nd
            expectedOccurrences: [],
            timeZone: "US/Michigan",
        });
    });

    it("Returns no occurrences for a schedule with no recurrences", async () => {
        await testCalculateOccurrences({
            startTime: "2023-01-01T13:00:00Z",
            endTime: "2023-06-02T13:00:00Z",
            recurrences: [],
            timeframeStart: "2023-05-27T00:00:00Z",
            timeframeEnd: "2023-06-02T13:00:00Z", // End of the timeframe is on June 2nd
            expectedOccurrences: [],
            timeZone: "US/Michigan",
        });
    });

    it("Returns no occurrences for a schedule with an applicable recurrence, but an exception which cancels it out", async () => {
        await testCalculateOccurrences({
            startTime: "1920-01-01T00:00:12Z",
            endTime: "2040-12-21T00:00:00Z",
            recurrences: [{
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 1, // Monday

            }] as ScheduleRecurrence[],
            timeframeStart: "2023-01-01T00:00:00Z",
            timeframeEnd: "2023-01-08T00:00:00Z",
            exceptions: [{
                originalStartTime: new Date("2023-01-02T00:00:12Z").toISOString(),
                newStartTime: null,
                newEndTime: null,
            }] as ScheduleException[],
            expectedOccurrences: [],
            timeZone: "Europe/London",
        });
    });

    it("Exceptions reschedule correctly using newEndTime", async () => {
        await testCalculateOccurrences({
            startTime: "1920-01-01T00:00:12Z",
            endTime: "2040-12-21T00:00:00Z",
            recurrences: [{
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 1, // Monday
                duration: 5, // 5 minutes
            }] as ScheduleRecurrence[],
            timeframeStart: "2023-01-01T00:00:00Z",
            timeframeEnd: "2023-01-08T00:00:00Z",
            exceptions: [{
                originalStartTime: new Date("2023-01-02T00:00:12Z").toISOString(),
                newStartTime: null,
                newEndTime: new Date("2023-01-02T02:01:01Z").toISOString(), // Should use this instead of the recurrence duration
            }] as ScheduleException[],
            expectedOccurrences: [
                { start: "2023-01-02T00:00:12Z", end: "2023-01-02T02:01:01Z" },
            ],
            timeZone: "Europe/London",
        });
    });

    it("Exceptions reschedule correctly using recurrence duration", async () => {
        await testCalculateOccurrences({
            startTime: "1920-01-01T00:00:12Z",
            endTime: "2040-12-21T00:00:00Z",
            recurrences: [{
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 1, // Monday
                duration: 17, // 17 minutes
            }] as ScheduleRecurrence[],
            timeframeStart: "2023-01-01T00:00:00Z",
            timeframeEnd: "2023-01-08T00:00:00Z",
            exceptions: [{
                originalStartTime: new Date("2023-01-02T00:00:12Z").toISOString(),
                newStartTime: new Date("2023-01-02T02:01:01Z").toISOString(),
                newEndTime: null, // Should use the recurrence duration instead of this
            }] as ScheduleException[],
            expectedOccurrences: [
                { start: "2023-01-02T02:01:01Z", end: "2023-01-02T02:18:01Z" },
            ],
            timeZone: "Europe/London",
        });
    });

    it("No occurrences are returned if the time frame is longer than a year", async () => {
        await testCalculateOccurrences({
            startTime: "0001-01-01T00:00:00Z",
            endTime: "9999-01-01T00:00:00Z",
            recurrences: [{
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 1,
            }] as ScheduleRecurrence[],
            timeframeStart: "2023-01-01T00:00:00Z",
            timeframeEnd: "2024-01-01T00:00:01Z",
            expectedOccurrences: [],
            timeZone: "UTC",
        });
    });

    it("No occurrences are returned if the time frame start is after the time frame end", async () => {
        await testCalculateOccurrences({
            startTime: "0001-01-01T00:00:00Z",
            endTime: "9999-01-01T00:00:00Z",
            recurrences: [{
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 1,
            }] as ScheduleRecurrence[],
            timeframeStart: "2023-01-01T00:00:00Z",
            timeframeEnd: "2022-01-01T00:00:01Z",
            expectedOccurrences: [],
            timeZone: "UTC",
        });
    });

    it("Ensures that we handle leap years correctly", async () => {
        await testCalculateOccurrences({
            startTime: "2020-02-29T10:00:10Z",
            endTime: "2029-02-29T00:00:00Z",
            recurrences: [{
                recurrenceType: "Yearly",
                interval: 1,
                month: 1, // February
                dayOfMonth: 31,
                duration: 1440, // 24 hours in minutes
            }] as ScheduleRecurrence[],
            timeframeStart: "2024-02-15T00:00:00Z",
            timeframeEnd: "2024-03-01T00:00:00Z",
            expectedOccurrences: [
                { start: "2024-02-29T10:00:10Z", end: "2024-03-01T10:00:10Z" },
            ],
            timeZone: "UTC",
        });
    });
});
