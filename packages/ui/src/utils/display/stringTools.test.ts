/* eslint-disable @typescript-eslint/ban-ts-comment */
import { Headers, displayDate, firstString, fontSizeToPixels, generateContext, getLineAtIndex, getLineEnd, getLineStart, getLinesAtRange, getTextSelection, insertBulletList, insertCheckboxList, insertCode, insertHeader, insertLink, insertNumberList, insertQuote, insertTable, padSelection, replaceText } from "./stringTools";

describe("firstString", () => {
    it("returns the first non-empty, non-whitespace string", () => {
        expect(firstString(" ", " hello", "world")).toBe(" hello");
    });

    it("returns an empty string when all inputs are empty or whitespace", () => {
        expect(firstString(" ", "", "   ")).toBe("");
    });

    it("ignores non-string inputs", () => {
        expect(firstString(42, true, {}, [], "hello", "world")).toBe("hello");
    });

    it("evaluates functions and returns the first non-blank string returned by a function", () => {
        expect(firstString(
            () => 42,
            () => " ",
            () => "hello",
            () => "world",
        )).toBe("hello");
    });

    it("returns an empty string if functions do not return a non-blank string", () => {
        expect(firstString(() => 42, () => true, () => " ")).toBe("");
    });

    it("correctly ignores undefined and null values", () => {
        expect(firstString(undefined, null, "valid")).toBe("valid");
    });

    it("returns an empty string if no arguments are provided", () => {
        expect(firstString()).toBe("");
    });

    it("handles a mix of strings, functions, and other types", () => {
        expect(firstString(
            123,
            " ",
            () => "  ",
            () => "valid",
            "not returned",
        )).toBe("valid");
    });

    it("treats function that returns non-string as invalid", () => {
        expect(firstString(
            () => { return { toString: () => "not a string" }; },
            () => "valid",
        )).toBe("valid");
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
        expect(displayDate(timestamp)).toBe("Dec 31, 2021, 9:00:00 AM"); // Adjust based on your locale
    });

    it("displays full date and time for dates not today (ISO string)", () => {
        // ISO string for Dec 31, 2021, 9:00 AM
        const timestamp = "2021-12-31T09:00:00.000Z";
        expect(displayDate(timestamp)).toBe("Dec 31, 2021, 9:00:00 AM"); // Adjust based on your locale
    });

    it("displays only date if showDateAndTime is false (timestamp)", () => {
        const timestamp = new Date("2021-12-31T09:00:00.000Z").getTime();
        expect(displayDate(timestamp, false)).toBe("Dec 31, 2021"); // Adjust based on your locale
    });

    it("displays only date if showDateAndTime is false (ISO string)", () => {
        const timestamp = "2021-12-31T09:00:00.000Z";
        expect(displayDate(timestamp, false)).toBe("Dec 31, 2021"); // Adjust based on your locale
    });

    it("displays \"Today at <time>\" for the current day (timestamp)", () => {
        // Using the mocked current day
        const timestamp = new Date("2022-01-01T12:00:00.000Z").getTime();
        expect(displayDate(timestamp)).toBe("12:00:00 PM"); // Adjust based on your locale
    });

    it("displays \"Today at <time>\" for the current day (ISO string)", () => {
        // Using the mocked current day
        const timestamp = "2022-01-01T12:00:00.000Z";
        expect(displayDate(timestamp)).toBe("12:00:00 PM"); // Adjust based on your locale
    });

    it("omits the year for dates within the current year but not today (timestamp)", () => {
        // Timestamp for Jan 2, 2022, 8:00 AM
        const timestamp = new Date("2022-01-02T08:00:00.000Z").getTime();
        expect(displayDate(timestamp)).toBe("Jan 2, 8:00:00 AM"); // Adjust based on your locale
    });

    it("omits the year for dates within the current year but not today (ISO string)", () => {
        // ISO string for Jan 2, 2022, 8:00 AM
        const timestamp = "2022-01-02T08:00:00.000Z";
        expect(displayDate(timestamp)).toBe("Jan 2, 8:00:00 AM"); // Adjust based on your locale
    });

    it("handles edge cases around midnight (timestamp)", () => {
        // Just before midnight
        let timestamp = new Date("2021-12-31T23:59:59.000Z").getTime();
        expect(displayDate(timestamp)).toBe("Dec 31, 2021, 11:59:59 PM");

        // Just after midnight
        timestamp = new Date("2022-01-01T00:00:01.000Z").getTime();
        expect(displayDate(timestamp)).toBe("12:00:01 AM"); // Assuming the current date is 2022-01-01
    });

    it("handles edge cases around midnight (ISO string)", () => {
        // Just before midnight
        let timestamp = "2021-12-31T23:59:59.000Z";
        expect(displayDate(timestamp)).toBe("Dec 31, 2021, 11:59:59 PM");

        // Just after midnight
        timestamp = "2022-01-01T00:00:01.000Z";
        expect(displayDate(timestamp)).toBe("12:00:01 AM"); // Assuming the current date is 2022-01-01
    });

    it("handles leap year correctly (timestamp)", () => {
        // Leap day in a leap year
        const timestamp = new Date("2020-02-29T10:00:00.000Z").getTime();
        expect(displayDate(timestamp)).toBe("Feb 29, 2020, 10:00:00 AM");
    });

    it("handles leap year correctly (ISO string)", () => {
        // Leap day in a leap year
        const timestamp = "2020-02-29T10:00:00.000Z";
        expect(displayDate(timestamp)).toBe("Feb 29, 2020, 10:00:00 AM");
    });

    it("adjusts for year-end and year-beginning dates (timestamp)", () => {
        // End of the year
        let timestamp = new Date("2021-12-31T10:00:00.000Z").getTime();
        expect(displayDate(timestamp)).toBe("Dec 31, 2021, 10:00:00 AM");

        // Beginning of the year
        timestamp = new Date("2022-01-01T10:00:00.000Z").getTime();
        expect(displayDate(timestamp)).toBe("10:00:00 AM");
    });

    it("adjusts for year-end and year-beginning dates (ISO string)", () => {
        // End of the year
        let timestamp = "2021-12-31T10:00:00.000Z";
        expect(displayDate(timestamp)).toBe("Dec 31, 2021, 10:00:00 AM");

        // Beginning of the year
        timestamp = "2022-01-01T10:00:00.000Z";
        expect(displayDate(timestamp)).toBe("10:00:00 AM");
    });

    it("distinguishes same day across different years (timestamp)", () => {
        // Same day, previous year
        const timestamp = new Date("2021-01-01T10:00:00.000Z").getTime();
        expect(displayDate(timestamp)).toBe("Jan 1, 2021, 10:00:00 AM");
    });

    it("distinguishes same day across different years (ISO string)", () => {
        // Same day, previous year
        const timestamp = "2021-01-01T10:00:00.000Z";
        expect(displayDate(timestamp)).toBe("Jan 1, 2021, 10:00:00 AM");
    });

    it("handles invalid timestamps gracefully", () => {
        // Invalid timestamp
        expect(displayDate(NaN)).toBe(null);
        expect(displayDate("invalid-date")).toBe(null);
    });

    it("distinguishes close times within the same day (timestamp)", () => {
        // 5 minutes before the mock current time
        let timestamp = new Date("2022-01-01T09:55:00Z").getTime();
        expect(displayDate(timestamp)).toBe("9:55:00 AM");

        // 5 minutes after the mock current time
        timestamp = new Date("2022-01-01T10:05:00Z").getTime();
        expect(displayDate(timestamp)).toBe("10:05:00 AM");
    });

    it("distinguishes close times within the same day (ISO string)", () => {
        // 5 minutes before the mock current time
        let timestamp = "2022-01-01T09:55:00Z";
        expect(displayDate(timestamp)).toBe("9:55:00 AM");

        // 5 minutes after the mock current time
        timestamp = "2022-01-01T10:05:00Z";
        expect(displayDate(timestamp)).toBe("10:05:00 AM");
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
        expect(fontSizeToPixels(14)).toBe(14);
    });

    it("converts px to pixels accurately", () => {
        expect(fontSizeToPixels("20px")).toBe(20);
    });

    it("converts rem to pixels based on default root font-size of 16px", () => {
        expect(fontSizeToPixels("2rem")).toBe(32);
    });

    it("converts em to pixels based on element's computed font-size", () => {
        // Mocking document.getElementById to simulate an element with a specific id
        document.getElementById = jest.fn().mockReturnValue({});
        expect(fontSizeToPixels("2em", "testId")).toBe(32); // Assuming mock of getComputedStyle returns 16px as fontSize
    });

    it("logs an error and returns 0 if trying to convert em without an id", () => {
        console.error = jest.fn();

        expect(fontSizeToPixels("2em")).toBe(0);
        expect(console.error).toHaveBeenCalledWith("Must provide id to convert em to px");
    });

    it("returns 0 for unsupported units", () => {
        expect(fontSizeToPixels("2pt")).toBe(0);
    });

    it("handles invalid string input gracefully", () => {
        expect(fontSizeToPixels("abc")).toBe(0);
    });

    it("returns 0 if element with specified id does not exist for em conversion", () => {
        document.getElementById = jest.fn().mockReturnValue(null); // Element not found
        expect(fontSizeToPixels("2em", "nonExistentId")).toBe(0);
    });
});

describe("replaceText", () => {
    it("replaces text in the middle of the string", () => {
        const originalText = "Hello, world!";
        const newText = "beautiful";
        const start = 7;
        const end = 12;
        const result = replaceText(originalText, newText, start, end);
        expect(result).toBe("Hello, beautiful!");
    });

    it("replaces text at the start of the string", () => {
        const originalText = "Hello, world!";
        const newText = "Hi";
        const start = 0;
        const end = 5;
        const result = replaceText(originalText, newText, start, end);
        expect(result).toBe("Hi, world!");
    });

    it("replaces text at the end of the string", () => {
        const originalText = "Hello, world!";
        const newText = "planet!";
        const start = 7;
        const end = 13;
        const result = replaceText(originalText, newText, start, end);
        expect(result).toBe("Hello, planet!");
    });

    it("inserts text when start and end are the same", () => {
        const originalText = "Hello, world!";
        const newText = " beautiful";
        const start = 6;
        const end = 6;
        const result = replaceText(originalText, newText, start, end);
        expect(result).toBe("Hello, beautiful world!");
    });

    it("returns the same text when start and end are out of bounds", () => {
        const originalText = "Hello, world!";
        const newText = "ignored";
        const start = 100;
        const end = 105;
        const result = replaceText(originalText, newText, start, end);
        expect(result).toBe(originalText);
    });

    it("handles negative start and end indices by ignoring the new text", () => {
        const originalText = "Hello, world!";
        const newText = "ignored";
        const start = -1;
        const end = -5;
        const result = replaceText(originalText, newText, start, end);
        expect(result).toBe(originalText);
    });

    it("correctly handles start greater than end by not modifying the original text", () => {
        const originalText = "Hello, world!";
        const newText = "ignored";
        const start = 14;
        const end = 5;
        const result = replaceText(originalText, newText, start, end);
        // Depending on the function's intended behavior, this could also be an error case.
        expect(result).toBe("Hello, world!");
    });

    it("allows replacing the entire text by setting start and end to encompass the whole string", () => {
        const originalText = "Hello, world!";
        const newText = "Goodbye, planet!";
        const start = 0;
        const end = originalText.length;
        const result = replaceText(originalText, newText, start, end);
        expect(result).toBe(newText);
    });
});

describe("getTextSelection", () => {
    // Mocking the textArea object
    function mockTextArea() {
        return {
            tagName: "TEXTAREA",
            value: "Hello, world!",
            selectionStart: 0,
            selectionEnd: 0,
            // You can add more properties and methods if needed
        } as const;
    }

    beforeAll(() => {
        // Mock console.error to avoid cluttering test output
        console.error = jest.fn();
    });

    beforeEach(() => {
        // Reset mockTextArea properties if needed
        const textArea = mockTextArea();
        document.getElementById = jest.fn().mockImplementation((id) => {
            if (id === "testTextarea") {
                return textArea;
            }
            return null; // Simulate behavior for non-existent elements
        });
    });

    it("returns selection details for a valid textarea", () => {
        const textArea = mockTextArea();
        textArea.selectionStart = 0;
        textArea.selectionEnd = 5;
        // @ts-ignore Mocking the getElementById function
        document.getElementById.mockReturnValueOnce(textArea);

        const selection = getTextSelection("testTextarea");
        expect(selection).toEqual({
            start: 0,
            end: 5,
            selected: "Hello",
            inputElement: textArea,
        });
    });

    it("returns zeros and null for a non-existent element", () => {
        // @ts-ignore Mocking the getElementById function
        document.getElementById.mockReturnValueOnce(null); // Element not found
        const selection = getTextSelection("nonExistentId");
        expect(selection).toEqual({
            start: 0,
            end: 0,
            selected: "",
            inputElement: null,
        });
        expect(console.error).toHaveBeenCalledWith("Element not found or is not a textarea: nonExistentId");
    });

    it("returns full text selection when all text is selected", () => {
        const textArea = mockTextArea();
        textArea.selectionStart = 0;
        textArea.selectionEnd = textArea.value.length;
        // @ts-ignore Mocking the getElementById function
        document.getElementById.mockReturnValueOnce(textArea);

        const selection = getTextSelection("testTextarea");
        expect(selection).toEqual({
            start: 0,
            end: textArea.value.length,
            selected: "Hello, world!",
            inputElement: textArea,
        });
    });

    it("returns empty string when there is no selection", () => {
        const textArea = mockTextArea();
        // @ts-ignore Mocking the getElementById function
        document.getElementById.mockReturnValueOnce(textArea);

        const selection = getTextSelection("testTextarea");
        expect(selection.selected).toBe("");
    });

    it("logs an error and returns defaults for non-textarea elements", () => {
        // Mock a non-textarea element
        document.getElementById = jest.fn().mockReturnValue({ tagName: "DIV" });
        const selection = getTextSelection("testDiv");
        expect(selection).toEqual({ start: 0, end: 0, selected: "", inputElement: null });
        expect(console.error).toHaveBeenCalledWith("Element not found or is not a textarea: testDiv");
    });
});

describe("getLineStart", () => {
    it("returns 0 for single-line text regardless of start index", () => {
        const text = "This is a single line of text.";
        expect(getLineStart(text, 10)).toBe(0);
        expect(getLineStart(text, text.length)).toBe(0); // End of line
    });

    it("returns the correct index at the start of the text", () => {
        const text = "First line\nSecond line\nThird line";
        expect(getLineStart(text, 0)).toBe(0); // At the very beginning
    });

    it("returns the correct index for multi-line text when cursor is on the first line", () => {
        const text = "First line\nSecond line\nThird line";
        expect(getLineStart(text, 5)).toBe(0); // In the middle of the first line
    });

    it("returns the correct index for multi-line text when cursor is on the middle line", () => {
        const text = "First line\nSecond line\nThird line";
        const startSecondLine = text.indexOf("\n") + 1; // Start of second line
        expect(getLineStart(text, startSecondLine + 5)).toBe(startSecondLine);
    });

    it("returns the correct index for multi-line text when cursor is on the last line", () => {
        const text = "First line\nSecond line\nThird line";
        const startLastLine = text.lastIndexOf("\n") + 1; // Start of last line
        expect(getLineStart(text, startLastLine + 5)).toBe(startLastLine);
    });

    it("returns the correct index when cursor is at the end of the text", () => {
        const text = "First line\nSecond line\nThird line";
        expect(getLineStart(text, text.length)).toBe(text.lastIndexOf("\n") + 1);
    });

    it("handles the start index being out of bounds", () => {
        const text = "Out of bounds test";
        expect(getLineStart(text, -1)).toBe(0); // Less than 0
        expect(getLineStart(text, text.length + 1)).toBe(0); // Greater than text length
    });

    it("handles empty string correctly", () => {
        const text = "";
        expect(getLineStart(text, 0)).toBe(0);
    });

    it("returns the start index when the cursor is immediately after a newline character", () => {
        const text = "First line\nSecond line\nThird line";
        const startSecondLine = text.indexOf("\n") + 1; // Start of second line
        expect(getLineStart(text, startSecondLine)).toBe(startSecondLine);
    });

    it("returns 0 when the text contains no newline characters", () => {
        const text = "No newline characters here";
        expect(getLineStart(text, 10)).toBe(0);
    });
});

describe("getLineEnd", () => {
    it("returns the length of text for single-line text regardless of start index", () => {
        const text = "This is a single line of text.";
        expect(getLineEnd(text, 10)).toBe(text.length);
        expect(getLineEnd(text, text.length)).toBe(text.length); // Cursor at the end of the line
    });

    it("returns the length of text when cursor is at the end of the text", () => {
        const text = "End of text\nSecond line\nThird line";
        expect(getLineEnd(text, text.length)).toBe(text.length);
    });

    it("returns the correct index for multi-line text when cursor is on the first line", () => {
        const text = "First line\nSecond line\nThird line";
        const endFirstLine = text.indexOf("\n"); // End of first line
        expect(getLineEnd(text, 5)).toBe(endFirstLine); // Cursor in the middle of the first line
    });

    it("returns the correct index for multi-line text when cursor is on the middle line", () => {
        const text = "First line\nSecond line\nThird line";
        const startSecondLine = text.indexOf("\n") + 1; // Start of second line
        const endSecondLine = text.indexOf("\n", startSecondLine); // End of second line
        expect(getLineEnd(text, startSecondLine + 5)).toBe(endSecondLine);
    });

    it("returns text length for multi-line text when cursor is on the last line", () => {
        const text = "First line\nSecond line\nThird line";
        const startLastLine = text.lastIndexOf("\n") + 1; // Start of last line
        expect(getLineEnd(text, startLastLine + 5)).toBe(text.length);
    });

    it("handles the start index being out of bounds", () => {
        const text = "Out of bounds test";
        expect(getLineEnd(text, -1)).toBe(text.length); // Less than 0
        expect(getLineEnd(text, text.length + 1)).toBe(text.length); // Greater than text length
    });

    it("handles empty string correctly", () => {
        const text = "";
        expect(getLineEnd(text, 0)).toBe(0);
    });

    it("returns the correct index when the cursor is at the start of a line", () => {
        const text = "First line\nSecond line\nThird line";
        const startSecondLine = text.indexOf("\n") + 1; // Start of second line
        const endSecondLine = text.indexOf("\n", startSecondLine); // End of second line
        expect(getLineEnd(text, startSecondLine)).toBe(endSecondLine);
    });

    it("returns text length if there are no newline characters after the cursor", () => {
        const text = "Line with no newline characters";
        expect(getLineEnd(text, 5)).toBe(text.length);
    });

    it("returns the correct index for lines followed by a newline character", () => {
        const text = "Line ending with newline\n";
        expect(getLineEnd(text, 0)).toBe(text.indexOf("\n"));
    });
});

describe("getLineAtIndex", () => {
    it("returns an empty line with 0 start and end indices for null input", () => {
        expect(getLineAtIndex(null, 5)).toEqual(["", 0, 0]);
    });

    it("returns an empty line with 0 start and end indices for undefined input", () => {
        expect(getLineAtIndex(undefined, 5)).toEqual(["", 0, 0]);
    });

    it("returns an empty line with 0 start and end indices for negative index", () => {
        const text = "This is a test.";
        expect(getLineAtIndex(text, -1)).toEqual(["", 0, 0]);
    });

    it("returns an empty line with 0 start and end indices for index out of text length", () => {
        const text = "Short text";
        expect(getLineAtIndex(text, 20)).toEqual(["", 0, 0]);
    });

    it("returns the whole text with correct start and end indices for single-line input", () => {
        const text = "Single line text";
        expect(getLineAtIndex(text, 5)).toEqual([text, 0, text.length]);
    });

    it("returns the first line with correct start and end indices for multi-line input", () => {
        const text = "First line\nSecond line\nThird line";
        expect(getLineAtIndex(text, 5)).toEqual(["First line", 0, "First line".length]);
    });

    it("returns the middle line with correct start and end indices for multi-line input", () => {
        const text = "First line\nSecond line\nThird line";
        const startOfSecondLine = "First line\n".length;
        const secondLine = "Second line";
        expect(getLineAtIndex(text, startOfSecondLine + 1)).toEqual([secondLine, startOfSecondLine, startOfSecondLine + secondLine.length]);
    });

    it("returns the last line with correct start and end indices for multi-line input", () => {
        const text = "First line\nSecond line\nThird line";
        const startOfLastLine = "First line\nSecond line\n".length;
        const lastLine = "Third line";
        expect(getLineAtIndex(text, startOfLastLine + 1)).toEqual([lastLine, startOfLastLine, text.length]);
    });

    it("handles index at the end of the text correctly", () => {
        const text = "Text ending without newline";
        expect(getLineAtIndex(text, text.length - 1)).toEqual([text, 0, text.length]);
    });

    it("returns the correct line and indices when index is on a newline character", () => {
        const text = "Line one\nLine two\nLine three";
        const startOfLineTwo = "Line one\n".length;
        const lineTwo = "Line two";
        expect(getLineAtIndex(text, startOfLineTwo)).toEqual([lineTwo, startOfLineTwo, startOfLineTwo + lineTwo.length]);
    });

    it("returns the correct line for empty lines", () => {
        const text = "First line\n\nThird line";
        const startOfEmptyLine = "First line\n".length;
        // The end of the empty line is the same as its start because the line is empty.
        expect(getLineAtIndex(text, startOfEmptyLine)).toEqual(["", startOfEmptyLine, startOfEmptyLine]);
    });
});

describe("getLinesAtRange", () => {
    const multiLineText = `First line
Second line
Third line
Fourth line`;

    it("returns a single line when start and end are within the same line", () => {
        const start = 0; // Start of the first line
        const end = 10; // End of the first line
        expect(getLinesAtRange(multiLineText, start, end)).toEqual([["First line"], start, multiLineText.indexOf("\n")]);
    });

    it("returns the correct start and end indices for a single line", () => {
        const start = multiLineText.indexOf("Second"); // Start of the second line
        const end = start + "Second line".length;
        const lineStart = start;
        const lineEnd = multiLineText.indexOf("\n", start);
        expect(getLinesAtRange(multiLineText, start, end)).toEqual([["Second line"], lineStart, lineEnd]);
    });

    it("returns multiple lines when the range spans across lines", () => {
        const start = 0; // Beginning of the first line
        const end = multiLineText.indexOf("Fourth line") + "Fourth line".length; // End of the last line
        expect(getLinesAtRange(multiLineText, start, end)).toEqual([
            ["First line", "Second line", "Third line", "Fourth line"],
            0,
            multiLineText.length,
        ]);
    });

    it("handles ranges that start and end with newline characters", () => {
        const start = multiLineText.indexOf("\n") + 1; // Just after the first newline
        const end = multiLineText.lastIndexOf("\n"); // Just before the last newline
        expect(getLinesAtRange(multiLineText, start, end)).toEqual([
            ["Second line", "Third line"],
            start,
            end,
        ]);
    });

    it("returns an empty array and 0 for start and end indices for an empty string", () => {
        expect(getLinesAtRange("", 0, 0)).toEqual([[], 0, 0]);
    });

    it("returns an empty array and 0 for start and end indices if start and end are out of bounds", () => {
        expect(getLinesAtRange(multiLineText, -1, multiLineText.length + 1)).toEqual([[], 0, 0]);
    });

    it("returns the last line correctly when the range starts in the last line", () => {
        const start = multiLineText.lastIndexOf("Fourth line"); // Start of the last line
        const end = multiLineText.length; // End of the text
        expect(getLinesAtRange(multiLineText, start, end)).toEqual([["Fourth line"], start, end]);
    });

    it("adjusts the end index to include the entire last selected line", () => {
        const start = multiLineText.indexOf("Second"); // Start of the second line
        const end = multiLineText.indexOf("Third line") - 1; // End right before the third line starts
        expect(getLinesAtRange(multiLineText, start, end)).toEqual([
            ["Second line"],
            start,
            multiLineText.indexOf("\n", start),
        ]);
    });

    it("returns the correct lines when the range includes only newlines", () => {
        const start = multiLineText.indexOf("\n") + 1; // \n before "Second line"
        const end = multiLineText.indexOf("\n", start) + 1; // \n before "Third line"
        expect(getLinesAtRange(multiLineText, start, end)).toEqual([
            // Since we have the newlines for the second and third lines, we should get both lines
            [
                "Second line",
                "Third line",
            ],
            start,
            multiLineText.indexOf("\n", end),
        ]);
    });
});

describe("insertHeader function", () => {
    it("should insert a header at the beginning of the text if there is none", () => {
        const initialText = "This is a test";
        const { end, start, text } = insertHeader(Headers.h2, initialText, 0, 0);
        expect(text).toEqual("## This is a test");
        // Cursor should be right after the inserted header (plus the space always inserted after a header)
        expect(start).toEqual(3);
        expect(end).toEqual(3);
    });

    it("should replace an existing header with a new one", () => {
        const initialText = "## This is a test";
        const { end, start, text } = insertHeader(Headers.h1, initialText, 3, 5);
        expect(text).toEqual("# This is a test");
        // Since new header is one character shorter, selection should move back by 1
        expect(start).toEqual(2);
        expect(end).toEqual(4);
    });

    it("should remove an existing header if the same type is selected", () => {
        const initialText = "### This is a test";
        const { end, start, text } = insertHeader(Headers.h3, initialText, 5, 7);
        expect(text).toEqual("This is a test");
        // Since removed header is 4 characters long (including the space), selection should move back by 4
        expect(start).toEqual(1);
        expect(end).toEqual(3);
    });
});

describe("padSelection function", () => {
    it("should add italics to an unstyled text", () => {
        const initialText = "This is a test";
        const { text, start, end } = padSelection("*", "*", initialText, 0, initialText.length);
        expect(text).toEqual("*This is a test*");
        expect(start).toEqual(1);
        expect(end).toEqual(15);
    });

    it("should remove italics from styled text", () => {
        const initialText = "*This is a test*";
        const { text, start, end } = padSelection("*", "*", initialText, 1, 15);
        expect(text).toEqual("This is a test");
        expect(start).toEqual(0);
        expect(end).toEqual(14);
    });

    it("should add italics inside bold text", () => {
        const initialText = "**This is a test**";
        const { text, start, end } = padSelection("*", "*", initialText, 2, 16);
        expect(text).toEqual("***This is a test***");
        expect(start).toEqual(3);
        expect(end).toEqual(17);
    });

    it("should remove italics but not bold when italics are within bold", () => {
        const initialText = "***This is a test***";
        const { text, start, end } = padSelection("*", "*", initialText, 3, 17);
        expect(text).toEqual("**This is a test**");
        expect(start).toEqual(2);
        expect(end).toEqual(16);
    });

    it("should handle no selection correctly by adding italics", () => {
        const initialText = "This is a test";
        const { text, start, end } = padSelection("*", "*", initialText, 5, 5);
        expect(text).toEqual("This **is a test");
        expect(start).toEqual(6);
        expect(end).toEqual(6);
    });

    it("should add strikethrough to unstyled text", () => {
        const initialText = "This is a test";
        const { text, start, end } = padSelection("~~", "~~", initialText, 0, initialText.length);
        expect(text).toEqual("~~This is a test~~");
        expect(start).toEqual(2);
        expect(end).toEqual(16);
    });

    it("should remove strikethrough from styled text", () => {
        const initialText = "~~This is a test~~";
        const { text, start, end } = padSelection("~~", "~~", initialText, 2, 16);
        expect(text).toEqual("This is a test");
        expect(start).toEqual(0);
        expect(end).toEqual(14);
    });

    it("should add underline to unstyled text", () => {
        const initialText = "This is a test";
        const { text, start, end } = padSelection("<u>", "</u>", initialText, 0, initialText.length);
        expect(text).toEqual("<u>This is a test</u>");
        expect(start).toEqual(3);
        expect(end).toEqual(17);
    });

    it("should remove underline from styled text", () => {
        const initialText = "<u>This is a test</u>";
        const { text, start, end } = padSelection("<u>", "</u>", initialText, 3, 17);
        expect(text).toEqual("This is a test");
        expect(start).toEqual(0);
        expect(end).toEqual(14);
    });

    it("should add custom markdown for spoiler to unstyled text", () => {
        const initialText = "This is a test";
        const { text, start, end } = padSelection("||", "||", initialText, 0, initialText.length);
        expect(text).toEqual("||This is a test||");
        expect(start).toEqual(2);
        expect(end).toEqual(16);
    });

    it("should remove custom markdown for spoiler from styled text", () => {
        const initialText = "||This is a test||";
        const { text, start, end } = padSelection("||", "||", initialText, 2, 16);
        expect(text).toEqual("This is a test");
        expect(start).toEqual(0);
        expect(end).toEqual(14);
    });

    // Test toggling behavior for strikethrough, underline, and spoiler when text is already styled
    it("should toggle strikethrough when already applied", () => {
        const initialText = "Normal ~~Strikethrough~~ Normal";
        const { text } = padSelection("~~", "~~", initialText, 9, 22);
        expect(text).toEqual("Normal Strikethrough Normal");
    });

    it("should toggle underline when already applied", () => {
        const initialText = "Normal <u>Underline</u> Normal";
        const { text } = padSelection("<u>", "</u>", initialText, 10, 19);
        expect(text).toEqual("Normal Underline Normal");
    });

    it("should toggle spoiler when already applied", () => {
        const initialText = "Normal ||Spoiler|| Normal";
        const { text } = padSelection("||", "||", initialText, 9, 16);
        expect(text).toEqual("Normal Spoiler Normal");
    });
});

describe("insertLink function", () => {
    // Test inserting a link without any selection
    it("should insert a placeholder link at the cursor position when no text is selected", () => {
        const text = "Some initial text.";
        const { text: updatedText, start, end } = insertLink(text, text.length, text.length);
        const expectedText = "Some initial text.[label](url)";
        expect(updatedText).toEqual(expectedText);
        // Expect selection to be the "url" placeholder, so it can be edited right away
        expect(start).toEqual(expectedText.indexOf("url"));
        expect(end).toEqual(expectedText.indexOf("url") + 3);
    });

    // Test wrapping selected text with link markdown as a label
    it("should wrap the selected text with markdown link syntax treating it as a label", () => {
        const text = "Click here for more information.";
        const { text: updatedText, start, end } = insertLink(text, 0, 10);
        const expectedText = "[Click here](url) for more information.";
        expect(updatedText).toEqual(expectedText);
        // Expect selection to be the "url" placeholder, so it can be edited right away
        expect(start).toEqual(expectedText.indexOf("url"));
        expect(end).toEqual(expectedText.indexOf("url") + 3);
    });

    // Test recognizing a valid URL and using it as the link target
    it("should recognize a valid URL in the selected text and use it as the link target", () => {
        const text = "Visit https://example.com for more information.";
        const { text: updatedText, start, end } = insertLink(text, 6, 25);
        const expectedText = "Visit [label](https://example.com) for more information.";
        expect(updatedText).toEqual(expectedText);
        // Expect selection to be the "label" placeholder, so it can be edited right away
        expect(start).toEqual(expectedText.indexOf("label"));
        expect(end).toEqual(expectedText.indexOf("label") + 5);
    });

    // Test recognizing a localhost URL in development mode as a valid URL
    it("should recognize a localhost URL in the selected text as a valid URL", () => {
        const text = "Development site is at http://localhost:3000.";
        const { text: updatedText, start, end } = insertLink(text, 22, 44); // The link plus the space before the link, to see if it trims correctly
        const expectedText = "Development site is at[label](http://localhost:3000).";
        expect(updatedText).toEqual(expectedText);
        // Expect selection to be the "label" placeholder, so it can be edited right away
        expect(start).toEqual(expectedText.indexOf("label"));
        expect(end).toEqual(expectedText.indexOf("label") + 5);
    });

    // Test recognizing a user handle as a valid URL
    it("should recognize a user handle as a valid URL", () => {
        const text = "Contact @user123 for more information.";
        const { text: updatedText, start, end } = insertLink(text, 8, 16);
        const expectedText = "Contact [label](@user123) for more information.";
        expect(updatedText).toEqual(expectedText);
        // Expect selection to be the "label" placeholder, so it can be edited right away
        expect(start).toEqual(expectedText.indexOf("label"));
        expect(end).toEqual(expectedText.indexOf("label") + 5);
    });

    it("should recognize an invalid user handle", () => {
        const text = "Contact us@er123 for more information.";
        const { text: updatedText, start, end } = insertLink(text, 8, 16);
        const expectedText = "Contact [us@er123](url) for more information.";
        expect(updatedText).toEqual(expectedText);
        // Expect selection to be the "url" placeholder, so it can be edited right away
        expect(start).toEqual(expectedText.indexOf("url"));
        expect(end).toEqual(expectedText.indexOf("url") + 3);
    });
});

describe("insertBulletList", () => {
    it("should add bullets to unbulleted lines", () => {
        const text = "Line 1\nLine 2\nLine 3";
        const { text: updatedText, start, end } = insertBulletList(text, 0, text.length - 1); // One character before the end
        const expectedText = "* Line 1\n* Line 2\n* Line 3";
        expect(updatedText).toEqual(expectedText);
        // Check if the selection has been correctly adjusted
        expect(start).toEqual(2);
        expect(end).toEqual(expectedText.length - 1); // Still one character before the end
    });

    it("should remove bullets from bulleted lines", () => {
        const text = "* Line 1\n* Line 2\n* Line 3";
        const { text: updatedText, start, end } = insertBulletList(text, 5, text.length); // Hasn't selected the full first line, but should still remove the bullet
        const expectedText = "Line 1\nLine 2\nLine 3";
        expect(updatedText).toEqual(expectedText);
        // Check if the selection has been correctly adjusted back
        expect(start).toEqual(3); // In the same relative position as before
        expect(end).toEqual(expectedText.length);
    });

    it("should add a bullet to a single line without affecting the other lines", () => {
        const text = "Line 1\nLine 2\nLine 3";
        const { text: updatedText, start, end } = insertBulletList(text, 7, 13); // Select the entire "Line 2"
        const expectedText = "Line 1\n* Line 2\nLine 3";
        expect(updatedText).toEqual(expectedText);
        // Check if the selection has been correctly adjusted
        expect(start).toEqual(9); // 7 + "* ".length
        expect(end).toEqual(15); // 13 + "* ".length
    });

    it("should handle selection that starts and ends mid-line", () => {
        const text = "Line 1 is here\nLine 2 is there\nLine 3 is everywhere";
        const { text: updatedText, start, end } = insertBulletList(text, 14, 30); // Select very end of "Line 1" and all of "Line 2"
        const expectedText = "* Line 1 is here\n* Line 2 is there\nLine 3 is everywhere";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(16); // 14 + "* ".length
        expect(end).toEqual(34); // 30 + "* ".length * 2, since two bullets were added
    });

    it("should still add bullets if some selected lines are not bulleted", () => {
        const text = "Line 1\n* Line 2\n* Line 3";
        const { text: updatedText } = insertBulletList(text, 0, text.length);
        const expectedText = "* Line 1\n* * Line 2\n* * Line 3";
        expect(updatedText).toEqual(expectedText);
    });

    it("should handle empty lines correctly", () => {
        const text = "";
        const { text: updatedText, start, end } = insertBulletList(text, 0, text.length);
        const expectedText = "* ";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(2); // After the bullet
        expect(end).toEqual(2); // At the end of the text
    });
});

describe("insertNumberList", () => {
    it("should add numbers to unnumbered lines", () => {
        const text = "Line 1\nLine 2\nLine 3";
        const { text: updatedText, start, end } = insertNumberList(text, 0, text.length);
        const expectedText = "1. Line 1\n2. Line 2\n3. Line 3";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(3); // After "1. "
        expect(end).toEqual(expectedText.length);
    });

    it("should remove numbers from numbered lines", () => {
        const text = "1. Line 1\n2. Line 2\n3. Line 3";
        const { text: updatedText, start, end } = insertNumberList(text, 1, text.length - 2);
        const expectedText = "Line 1\nLine 2\nLine 3";
        expect(updatedText).toEqual(expectedText);
        // Expect selection to adjust back considering the removal of numbers
        expect(start).toEqual(0);
        expect(end).toEqual(expectedText.length - 2);
    });

    it("should handle the transition from single to double digits gracefully", () => {
        let text = "";
        for (let i = 1; i <= 10; i++) {
            text += `Line ${i}\n`;
        }
        // Remove last newline character
        text = text.slice(0, -1);
        const { text: updatedText, start, end } = insertNumberList(text, 0, text.length);
        let expectedText = "";
        for (let i = 1; i <= 10; i++) {
            expectedText += `${i}. Line ${i}\n`;
        }
        // Remove last newline character
        expectedText = expectedText.slice(0, -1);
        expect(updatedText).toEqual(expectedText);
        // Adjusting for the length of the numbers
        expect(start).toEqual(3); // After "1. "
        expect(end).toEqual(expectedText.length);
    });

    it("should add numbers only to selected lines", () => {
        const text = "Line 1\nLine 2\nLine 3\nLine 4";
        const { text: updatedText, start, end } = insertNumberList(text, 7, 20); // Select "Line 2" to "Line 3"
        const expectedText = "Line 1\n1. Line 2\n2. Line 3\nLine 4";
        expect(updatedText).toEqual(expectedText);
        // Check if the selection has been correctly adjusted
        expect(start).toEqual(10); // After "1. "
        expect(end).toEqual(expectedText.length - 7); // At the end of "Line 3"
    });

    it("should remove numbers if all selected lines are numbered", () => {
        const text = "1. Line 1\n2. Line 2\n3. Line 3";
        const { text: updatedText, start, end } = insertNumberList(text, 0, text.length);
        const expectedText = "Line 1\nLine 2\nLine 3";
        expect(updatedText).toEqual(expectedText);
        // Expect selection to adjust back considering the removal of numbers
        expect(start).toEqual(0);
        expect(end).toEqual(expectedText.length);
    });

    // This test ensures that when we mix numbered and unnumbered lines in a selection, all get numbered again
    it("should correctly number mixed selection of numbered and unnumbered lines", () => {
        const text = "Line 1\n2. Line 2\nLine 3";
        const { text: updatedText } = insertNumberList(text, 0, text.length);
        const expectedText = "1. Line 1\n2. 2. Line 2\n3. Line 3"; // Note how we keep the existing number for the second line
        expect(updatedText).toEqual(expectedText);
    });

    it("should handle empty lines correctly", () => {
        const text = "";
        const { text: updatedText, start, end } = insertNumberList(text, 0, text.length);
        const expectedText = "1. ";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(3); // After the bullet
        expect(end).toEqual(3); // At the end of the text
    });
});

describe("insertCheckboxList", () => {
    it("should add checkboxes to uncheckboxed lines", () => {
        const text = "Line 1\nLine 2\nLine 3";
        const { text: updatedText, start, end } = insertCheckboxList(text, 0, text.length);
        const expectedText = "- [ ] Line 1\n- [ ] Line 2\n- [ ] Line 3";
        expect(updatedText).toEqual(expectedText);
        // Check if the selection has been correctly adjusted
        expect(start).toEqual(6); // After "- [ ] "
        expect(end).toEqual(expectedText.length);
    });

    it("should remove checkboxes from checkboxed lines", () => {
        const text = "- [ ] Line 1\n- [ ] Line 2\n- [ ] Line 3";
        const { text: updatedText, start, end } = insertCheckboxList(text, 0, text.length);
        const expectedText = "Line 1\nLine 2\nLine 3";
        expect(updatedText).toEqual(expectedText);
        // Check if the selection has been correctly adjusted back
        expect(start).toEqual(0);
        expect(end).toEqual(expectedText.length);
    });

    it("should handle partial line selection correctly and mixed checkboxed and uncheckboxed lines", () => {
        const text = "Line 1\n- [ ] Line 2\nLine 3";
        const { text: updatedText, start, end } = insertCheckboxList(text, 7, 21); // Select Line 2 and part of Line 3
        const expectedText = "Line 1\n- [ ] - [ ] Line 2\n- [ ] Line 3";
        expect(updatedText).toEqual(expectedText);
        // Check if the selection has been correctly adjusted
        expect(start).toEqual(7 + "- [ ] ".length);
        expect(end).toEqual(expectedText.length - 5); // Same relative position in "Line 3"
    });

    it("should handle empty lines correctly", () => {
        const text = "";
        const { text: updatedText, start, end } = insertCheckboxList(text, 0, text.length);
        const expectedText = "- [ ] ";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(6); // After "- [ ] "
        expect(end).toEqual(6); // At the end of the text
    });

    it("should remove checkboxes if all selected lines are checkboxed", () => {
        const text = "- [ ] Line 1\n- [ ] Line 2\n- [ ] Line 3";
        const { text: updatedText, start, end } = insertCheckboxList(text, 0, text.length);
        const expectedText = "Line 1\nLine 2\nLine 3";
        expect(updatedText).toEqual(expectedText);
        // Expect selection to adjust back considering the removal of checkboxes
        expect(start).toEqual(0);
        expect(end).toEqual(expectedText.length);
    });

    it("should toggle checkboxes for a single line selection", () => {
        const text = "Line 1\n- [ ] Line 2\nLine 3";
        const { text: updatedText, start, end } = insertCheckboxList(text, 8, 18); // Select only "Line 2"
        const expectedText = "Line 1\nLine 2\nLine 3"; // Checkbox should be removed
        expect(updatedText).toEqual(expectedText);
        // Check if the selection has been correctly adjusted
        expect(start).toEqual(7); // Beginning of Line 2
        expect(end).toEqual(18 - "- [ ] ".length); // Same relative position in "Line 2"
    });

    // This test ensures the function works correctly even with non-standard checkbox states
    it("should handle checkboxes with 'x' correctly", () => {
        const text = "- [x] Line 1\n- [ ] Line 2";
        const { text: updatedText, start, end } = insertCheckboxList(text, 0, text.length);
        const expectedText = "Line 1\nLine 2";
        expect(updatedText).toEqual(expectedText);
        // Check if the selection has been correctly adjusted
        expect(start).toEqual(0);
        expect(end).toEqual(expectedText.length);
    });
});

describe("insertCode", () => {
    it("should wrap a single line in inline code markers if not already wrapped", () => {
        const text = "This is some text.";
        const { text: updatedText, start, end } = insertCode(text, 0, text.length);
        const expectedText = "`This is some text.`";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(1); // After "`"
        expect(end).toEqual(expectedText.length - 1); // Before the closing "`"
    });

    it("should remove inline code markers if the single line is already wrapped - code markers included in selection", () => {
        const text = "`This is some text.`";
        const { text: updatedText, start, end } = insertCode(text, 0, text.length);
        const expectedText = "This is some text.";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(0);
        expect(end).toEqual(expectedText.length);
    });

    it("should remove inline code markers if the single line is already wrapped - code markers around selection", () => {
        const text = "`This is some text.`";
        const { text: updatedText, start, end } = insertCode(text, 1, text.length - 1);
        const expectedText = "This is some text.";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(0);
        expect(end).toEqual(expectedText.length);
    });

    it("should wrap multiple lines in code block markers if not already wrapped", () => {
        const text = "Line 1\nLine 2\nLine 3";
        const { text: updatedText, start, end } = insertCode(text, 0, text.length);
        const expectedText = "```\nLine 1\nLine 2\nLine 3\n```";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(4); // After "```\n"
        expect(end).toEqual(expectedText.length - 4); // Before the closing "\n```"
    });

    it("should remove code block markers if multiple lines are already wrapped - code markers included in selection", () => {
        const text = "```\nLine 1\nLine 2\nLine 3\n```";
        const { text: updatedText, start, end } = insertCode(text, 0, text.length);
        const expectedText = "Line 1\nLine 2\nLine 3";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(0);
        expect(end).toEqual(expectedText.length);
    });

    it("should remove code block markers if multiple lines are already wrapped - code markers not included in selection", () => {
        const text = "```\nLine 1\nLine 2\nLine 3\n```";
        const { text: updatedText, start, end } = insertCode(text, 4, text.length - 4);
        const expectedText = "Line 1\nLine 2\nLine 3";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(0);
        expect(end).toEqual(expectedText.length);
    });

    it("should correctly handle single line selection surrounded by other text", () => {
        const text = "Some text `inline code` some more text.";
        const start = text.indexOf("`");
        const end = text.lastIndexOf("`") + 1;
        const { text: updatedText, start: updatedStart, end: updatedEnd } = insertCode(text, start, end);
        const expectedText = "Some text inline code some more text.";
        expect(updatedText).toEqual(expectedText);
        expect(updatedStart).toEqual(start);
        expect(updatedEnd).toEqual(end - 2); // 2 code markers removed
    });

    it("should add inline code markers for single line selection surrounded by other text if not present", () => {
        const text = "Some text inline code some more text.";
        const start = text.indexOf("inline");
        const end = text.indexOf("code") + "code".length;
        const { text: updatedText, start: updatedStart, end: updatedEnd } = insertCode(text, start, end);
        const expectedText = "Some text `inline code` some more text.";
        expect(updatedText).toEqual(expectedText);
        expect(updatedStart).toEqual(11);
        expect(updatedEnd).toEqual(22);
    });

    it("should handle an empty selection by inserting inline code markers", () => {
        const text = "Some text.";
        const { text: updatedText, start, end } = insertCode(text, text.length, text.length);
        const expectedText = "Some text.``";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(expectedText.length - 1);
        expect(end).toEqual(expectedText.length - 1);
    });

    it("should correctly handle multiline selection with mixed content (code and text)", () => {
        const text = "Some text\n```\nCode block\n```";
        const { text: updatedText, start, end } = insertCode(text, 0, text.length);
        const expectedText = "```\nSome text\n```\nCode block\n```\n```";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(4); // After the first "```\n"
        expect(end).toEqual(expectedText.length - 4); // Before the last "\n```"
    });
});

describe("insertQuote function", () => {
    it("should add a quote marker to unquoted text", () => {
        const text = "This is a test.";
        const { text: updatedText, start, end } = insertQuote(text, 0, text.length);
        const expectedText = "> This is a test.";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(2); // Adjusted for "> "
        expect(end).toEqual(expectedText.length);
    });

    it("should remove a quote marker from quoted text", () => {
        const text = "> This is a test.";
        const { text: updatedText, start, end } = insertQuote(text, 2, text.length);
        const expectedText = "This is a test.";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(0); // Adjusted for removal of "> "
        expect(end).toEqual(expectedText.length);
    });

    it("should handle nested quotes by removing one level of quoting", () => {
        const text = ">> This is a test.";
        const { text: updatedText, start, end } = insertQuote(text, 3, text.length);
        const expectedText = "> This is a test.";
        expect(updatedText).toEqual(expectedText);
        // Since we're removing one level of quoting, adjust start by removing one ">" marker
        expect(start).toEqual(2);
        expect(end).toEqual(expectedText.length);
    });

    it("should add a quote marker to each line in a multi-line selection", () => {
        const text = "Line 1\nLine 2\nLine 3";
        const { text: updatedText, start, end } = insertQuote(text, 0, text.length);
        const expectedText = "> Line 1\n> Line 2\n> Line 3";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(2); // Adjusted for first "> "
        expect(end).toEqual(expectedText.length);
    });

    it("should remove a quote marker from each line in a multi-line selection", () => {
        const text = "> Line 1\n> Line 2\n> Line 3";
        const { text: updatedText, start, end } = insertQuote(text, 0, text.length);
        const expectedText = "Line 1\nLine 2\nLine 3";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(0); // Adjusted for removal of "> "
        expect(end).toEqual(expectedText.length);
    });

    it("should handle mixed quoted and unquoted lines by adding quotes", () => {
        const text = "Line 1\n> Line 2\nLine 3";
        const { text: updatedText, start, end } = insertQuote(text, 0, text.length);
        const expectedText = "> Line 1\n>> Line 2\n> Line 3";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(2); // Adjusted for first "> "
        expect(end).toEqual(expectedText.length);
    });

    it("should correctly handle nested quotes within a multi-line selection", () => {
        const text = ">> Line 1\n> Line 2\nLine 3";
        const { text: updatedText, start, end } = insertQuote(text, 0, text.length);
        const expectedText = ">>> Line 1\n>> Line 2\n> Line 3";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(1);
        expect(end).toEqual(expectedText.length);
    });

    it("should remove one level of quoting from mixed nested quotes within a multi-line selection", () => {
        const text = ">> Line 1\n> > Line 2\n> Line 3";
        const { text: updatedText, start, end } = insertQuote(text, 0, text.length);
        const expectedText = "> Line 1\n> Line 2\nLine 3";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(0);
        expect(end).toEqual(expectedText.length);
    });

    it("should handle an empty selection by adding a quote marker", () => {
        const text = "";
        const { text: updatedText, start, end } = insertQuote(text, text.length, text.length);
        const expectedText = "> ";
        expect(updatedText).toEqual(expectedText);
        expect(start).toEqual(expectedText.length); // At the end of the new quote
        expect(end).toEqual(expectedText.length);
    });
});

describe("insertTable function", () => {
    it("should insert a table with the specified number of rows and columns - selection at end of line", () => {
        const text = "Some initial text.";
        const { text: updatedText, start, end } = insertTable(2, 3, text, text.length, text.length);
        // Always adds to the next line when there's non-empty text
        const expectedText = `Some initial text.
| Header | Header | Header |
| ------- | ------- | ------- |
|   |   |   |
|   |   |   |
`;
        expect(updatedText).toEqual(expectedText);
        // Expect selection to be the first header cell
        expect(start).toEqual(updatedText.indexOf("Header"));
        expect(end).toEqual(updatedText.indexOf("Header") + "Header".length);
    });

    it("should insert a table with the specified number of rows and columns - selection in middle of line", () => {
        const text = "Some initial text.";
        const { text: updatedText, start, end } = insertTable(2, 3, text, 2, 5);
        // Should not replace selected text, and instead add the table to the next line
        const expectedText = `Some initial text.
| Header | Header | Header |
| ------- | ------- | ------- |
|   |   |   |
|   |   |   |
`;
        expect(updatedText).toEqual(expectedText);
        // Expect selection to be the first header cell
        expect(start).toEqual(updatedText.indexOf("Header"));
        expect(end).toEqual(updatedText.indexOf("Header") + "Header".length);
    });

    it("should insert a table with the specified number of rows and columns - selection at start of line", () => {
        const text = "Some initial text.";
        const { text: updatedText, start, end } = insertTable(2, 3, text, 0, 0);
        // Should add the table to the next line
        const expectedText = `Some initial text.
| Header | Header | Header |
| ------- | ------- | ------- |
|   |   |   |
|   |   |   |
`;
        expect(updatedText).toEqual(expectedText);
        // Expect selection to be the first header cell
        expect(start).toEqual(updatedText.indexOf("Header"));
        expect(end).toEqual(updatedText.indexOf("Header") + "Header".length);
    });

    it("should handle zero rows correctly by doing nothing", () => {
        const text = "Before table";
        const { text: updatedText, start, end } = insertTable(0, 2, text, text.length, text.length);
        const expectedText = text;
        expect(updatedText).toEqual(expectedText);
        // Expect selection to be unchanged
        expect(start).toEqual(text.length);
        expect(end).toEqual(start);
    });

    it("should handle zero columns correctly by doing nothing", () => {
        const text = "Text before table.";
        const { text: updatedText, start, end } = insertTable(2, 0, text, text.length, text.length);
        const expectedText = text;
        expect(updatedText).toEqual(expectedText);
        // Expect selection to be unchanged
        expect(start).toEqual(text.length);
        expect(end).toEqual(start);
    });

    it("should correctly insert a table when there is no initial text", () => {
        const { text: updatedText, start, end } = insertTable(1, 1, "", 0, 0);
        const expectedText = `| Header |
| ------- |
|   |`;
        expect(updatedText).toEqual(expectedText);
        // Expect selection to be in the first header cell
        expect(start).toEqual(updatedText.indexOf("Header"));
        expect(end).toEqual(updatedText.indexOf("Header") + "Header".length);
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
