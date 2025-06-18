// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-18
/* eslint-disable @typescript-eslint/ban-ts-comment */
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import { displayDate, firstString, fontSizeToPixels, generateContext } from "./stringTools.js";

describe("firstString", () => {
    it("returns the first non-empty, non-whitespace string", () => {
        expect(firstString(" ", " hello", "world")).toEqual(" hello");
    });

    it("returns an empty string when all inputs are empty or whitespace", () => {
        expect(firstString(" ", "", "   ")).toEqual("");
    });

    it("ignores non-string inputs", () => {
        expect(firstString(42, true, {}, [], "hello", "world")).toEqual("hello");
    });

    it("evaluates functions and returns the first non-blank string returned by a function", () => {
        expect(firstString(
            () => 42,
            () => " ",
            () => "hello",
            () => "world",
        )).toEqual("hello");
    });

    it("returns an empty string if functions do not return a non-blank string", () => {
        expect(firstString(() => 42, () => true, () => " ")).toEqual("");
    });

    it("correctly ignores undefined and null values", () => {
        expect(firstString(undefined, null, "valid")).toEqual("valid");
    });

    it("returns an empty string if no arguments are provided", () => {
        expect(firstString()).toEqual("");
    });

    it("handles a mix of strings, functions, and other types", () => {
        expect(firstString(
            123,
            " ",
            () => "  ",
            () => "valid",
            "not returned",
        )).toEqual("valid");
    });

    it("treats function that returns non-string as invalid", () => {
        expect(firstString(
            () => { return { toString: () => "not a string" }; },
            () => "valid",
        )).toEqual("valid");
    });
});

describe("displayDate", () => {
    const originalDate = global.Date;
    const originalNavigator = global.navigator;

    beforeAll(() => {
        // Mock the current date
        const mockCurrentDate = new Date("2022-01-01T10:00:00Z");
        const RealDate = Date;

        // Mock implementation of Date
        const mockDate = class extends RealDate {
            constructor(date?: string | number) {
                super(date ? date : mockCurrentDate.toISOString());
            }

            static now() {
                return mockCurrentDate.getTime();
            }
        };

        // Override the global Date with our mock
        global.Date = mockDate as any; // Casting to `any` to bypass TypeScript's strict typing

        // Mock navigator.language if necessary
        Object.defineProperty(global.navigator, "language", { value: "en-US", configurable: true });
    });

    afterAll(() => {
        // Restore the original Date and navigator objects
        global.Date = originalDate;
        global.navigator = originalNavigator;
    });

    it("displays full date and time for dates not today (timestamp)", () => {
        // Timestamp for Dec 31, 2021, 9:00 AM
        const timestamp = new Date("2021-12-31T09:00:00.000Z").getTime();
        expect(displayDate(timestamp)).toEqual("Dec 31, 2021, 9:00:00 AM"); // Adjust based on your locale
    });

    it("displays full date and time for dates not today (ISO string)", () => {
        // ISO string for Dec 31, 2021, 9:00 AM
        const timestamp = "2021-12-31T09:00:00.000Z";
        expect(displayDate(timestamp)).toEqual("Dec 31, 2021, 9:00:00 AM"); // Adjust based on your locale
    });

    it("displays only date if showDateAndTime is false (timestamp)", () => {
        const timestamp = new Date("2021-12-31T09:00:00.000Z").getTime();
        expect(displayDate(timestamp, false)).toEqual("Dec 31, 2021"); // Adjust based on your locale
    });

    it("displays only date if showDateAndTime is false (ISO string)", () => {
        const timestamp = "2021-12-31T09:00:00.000Z";
        expect(displayDate(timestamp, false)).toEqual("Dec 31, 2021"); // Adjust based on your locale
    });

    it("displays \"Today at <time>\" for the current day (timestamp)", () => {
        // Using the mocked current day
        const timestamp = new Date("2022-01-01T12:00:00.000Z").getTime();
        expect(displayDate(timestamp)).toEqual("12:00:00 PM"); // Adjust based on your locale
    });

    it("displays \"Today at <time>\" for the current day (ISO string)", () => {
        // Using the mocked current day
        const timestamp = "2022-01-01T12:00:00.000Z";
        expect(displayDate(timestamp)).toEqual("12:00:00 PM"); // Adjust based on your locale
    });

    it("omits the year for dates within the current year but not today (timestamp)", () => {
        // Timestamp for Jan 2, 2022, 8:00 AM
        const timestamp = new Date("2022-01-02T08:00:00.000Z").getTime();
        expect(displayDate(timestamp)).toEqual("Jan 2, 8:00:00 AM"); // Adjust based on your locale
    });

    it("omits the year for dates within the current year but not today (ISO string)", () => {
        // ISO string for Jan 2, 2022, 8:00 AM
        const timestamp = "2022-01-02T08:00:00.000Z";
        expect(displayDate(timestamp)).toEqual("Jan 2, 8:00:00 AM"); // Adjust based on your locale
    });

    it("handles edge cases around midnight (timestamp)", () => {
        // Just before midnight
        let timestamp = new Date("2021-12-31T23:59:59.000Z").getTime();
        expect(displayDate(timestamp)).toEqual("Dec 31, 2021, 11:59:59 PM");

        // Just after midnight
        timestamp = new Date("2022-01-01T00:00:01.000Z").getTime();
        expect(displayDate(timestamp)).toEqual("12:00:01 AM"); // Assuming the current date is 2022-01-01
    });

    it("handles edge cases around midnight (ISO string)", () => {
        // Just before midnight
        let timestamp = "2021-12-31T23:59:59.000Z";
        expect(displayDate(timestamp)).toEqual("Dec 31, 2021, 11:59:59 PM");

        // Just after midnight
        timestamp = "2022-01-01T00:00:01.000Z";
        expect(displayDate(timestamp)).toEqual("12:00:01 AM"); // Assuming the current date is 2022-01-01
    });

    it("handles leap year correctly (timestamp)", () => {
        // Leap day in a leap year
        const timestamp = new Date("2020-02-29T10:00:00.000Z").getTime();
        expect(displayDate(timestamp)).toEqual("Feb 29, 2020, 10:00:00 AM");
    });

    it("handles leap year correctly (ISO string)", () => {
        // Leap day in a leap year
        const timestamp = "2020-02-29T10:00:00.000Z";
        expect(displayDate(timestamp)).toEqual("Feb 29, 2020, 10:00:00 AM");
    });

    it("adjusts for year-end and year-beginning dates (timestamp)", () => {
        // End of the year
        let timestamp = new Date("2021-12-31T10:00:00.000Z").getTime();
        expect(displayDate(timestamp)).toEqual("Dec 31, 2021, 10:00:00 AM");

        // Beginning of the year
        timestamp = new Date("2022-01-01T10:00:00.000Z").getTime();
        expect(displayDate(timestamp)).toEqual("10:00:00 AM");
    });

    it("adjusts for year-end and year-beginning dates (ISO string)", () => {
        // End of the year
        let timestamp = "2021-12-31T10:00:00.000Z";
        expect(displayDate(timestamp)).toEqual("Dec 31, 2021, 10:00:00 AM");

        // Beginning of the year
        timestamp = "2022-01-01T10:00:00.000Z";
        expect(displayDate(timestamp)).toEqual("10:00:00 AM");
    });

    it("distinguishes same day across different years (timestamp)", () => {
        // Same day, previous year
        const timestamp = new Date("2021-01-01T10:00:00.000Z").getTime();
        expect(displayDate(timestamp)).toEqual("Jan 1, 2021, 10:00:00 AM");
    });

    it("distinguishes same day across different years (ISO string)", () => {
        // Same day, previous year
        const timestamp = "2021-01-01T10:00:00.000Z";
        expect(displayDate(timestamp)).toEqual("Jan 1, 2021, 10:00:00 AM");
    });

    it("handles invalid timestamps gracefully", () => {
        // Invalid timestamp
        expect(displayDate(NaN)).toEqual(null);
        expect(displayDate("invalid-date")).toEqual(null);
    });

    it("distinguishes close times within the same day (timestamp)", () => {
        // 5 minutes before the mock current time
        let timestamp = new Date("2022-01-01T09:55:00Z").getTime();
        expect(displayDate(timestamp)).toEqual("9:55:00 AM");

        // 5 minutes after the mock current time
        timestamp = new Date("2022-01-01T10:05:00Z").getTime();
        expect(displayDate(timestamp)).toEqual("10:05:00 AM");
    });

    it("distinguishes close times within the same day (ISO string)", () => {
        // 5 minutes before the mock current time
        let timestamp = "2022-01-01T09:55:00Z";
        expect(displayDate(timestamp)).toEqual("9:55:00 AM");

        // 5 minutes after the mock current time
        timestamp = "2022-01-01T10:05:00Z";
        expect(displayDate(timestamp)).toEqual("10:05:00 AM");
    });
});

describe("fontSizeToPixels", () => {
    const originalGetComputedStyle = window.getComputedStyle;

    beforeAll(() => {
        // Define a mock for getComputedStyle that returns a consistent fontSize
        window.getComputedStyle = () => ({
            fontSize: "16px",
        } as CSSStyleDeclaration);
    });

    afterAll(() => {
        // Restore the original getComputedStyle function
        window.getComputedStyle = originalGetComputedStyle;
    });

    it("returns the same number for numeric input", () => {
        expect(fontSizeToPixels(14)).toEqual(14);
    });

    it("converts px to pixels accurately", () => {
        expect(fontSizeToPixels("20px")).toEqual(20);
    });

    it("converts rem to pixels based on default root font-size of 16px", () => {
        expect(fontSizeToPixels("2rem")).toEqual(32);
    });

    it("converts em to pixels based on element's computed font-size", () => {
        // Mocking document.getElementById to simulate an element with a specific id
        document.getElementById = vi.fn().mockReturnValue({});
        expect(fontSizeToPixels("2em", "testId")).toEqual(32); // Assuming mock of getComputedStyle returns 16px as fontSize
    });

    it("logs an error and returns 0 if trying to convert em without an id", () => {
        console.error = vi.fn();

        expect(fontSizeToPixels("2em")).toEqual(0);
        expect(console.error).toHaveBeenCalledWith("Must provide id to convert em to px");
    });

    it("returns 0 for unsupported units", () => {
        expect(fontSizeToPixels("2pt")).toEqual(0);
    });

    it("handles invalid string input gracefully", () => {
        expect(fontSizeToPixels("abc")).toEqual(0);
    });

    it("returns 0 if element with specified id does not exist for em conversion", () => {
        document.getElementById = vi.fn().mockReturnValue(null); // Element not found
        expect(fontSizeToPixels("2em", "nonExistentId")).toEqual(0);
    });
});

describe("generateContext function", () => {
    const MAX_CONTENT_LENGTH = 1500;

    it("returns the trimmed selected text", () => {
        const selected = " Selected text ";
        const fullText = "Full text is irrelevant in this case";
        const context = generateContext(selected, fullText);
        expect(context).toEqual("Selected text");
    });

    it("returns the last 1500 characters of selected text when it exceeds the maximum length", () => {
        const longSelected = "z" + "a".repeat(MAX_CONTENT_LENGTH + 50);
        const context = generateContext(longSelected, "");
        expect(context).toEqual(`…${longSelected.slice(-MAX_CONTENT_LENGTH - 1)}`);
    });

    it("returns the full text trimmed if no selected text", () => {
        const selected = "";
        const fullText = " Full text ";
        const context = generateContext(selected, fullText);
        expect(context).toEqual("Full text");
    });

    it("returns the last 1500 characters of the full text if it exceeds the maximum length and no selected text", () => {
        const fullText = "z" + "b".repeat(MAX_CONTENT_LENGTH + 50);
        const context = generateContext("", fullText);
        expect(context).toEqual(`…${fullText.slice(-MAX_CONTENT_LENGTH - 1)}`);
    });

    it("returns empty string when both selected text and full text are empty", () => {
        const context = generateContext("", "");
        expect(context).toEqual("");
    });
});
