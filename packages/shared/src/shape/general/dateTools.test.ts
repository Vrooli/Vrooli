import { expect } from "chai";
import { describe, it } from "mocha";
import { toDatetimeLocal, fromDatetimeLocal } from "./dateTools.js";

describe("dateTools", () => {
    describe("toDatetimeLocal", () => {
        it("should format a date correctly", () => {
            const date = new Date(2023, 11, 25, 14, 30); // Dec 25, 2023, 2:30 PM
            const result = toDatetimeLocal(date);
            expect(result).to.equal("2023-12-25T14:30");
        });

        it("should pad single digits with zeros", () => {
            const date = new Date(2023, 0, 5, 8, 5); // Jan 5, 2023, 8:05 AM
            const result = toDatetimeLocal(date);
            expect(result).to.equal("2023-01-05T08:05");
        });

        it("should handle midnight correctly", () => {
            const date = new Date(2023, 6, 15, 0, 0); // Jul 15, 2023, 12:00 AM
            const result = toDatetimeLocal(date);
            expect(result).to.equal("2023-07-15T00:00");
        });

        it("should handle end of day correctly", () => {
            const date = new Date(2023, 11, 31, 23, 59); // Dec 31, 2023, 11:59 PM
            const result = toDatetimeLocal(date);
            expect(result).to.equal("2023-12-31T23:59");
        });

        it("should handle leap year date", () => {
            const date = new Date(2024, 1, 29, 12, 30); // Feb 29, 2024, 12:30 PM
            const result = toDatetimeLocal(date);
            expect(result).to.equal("2024-02-29T12:30");
        });

        it("should handle year before 1000", () => {
            const date = new Date(999, 0, 1, 12, 0); // Jan 1, 999, 12:00 PM
            const result = toDatetimeLocal(date);
            expect(result).to.equal("0999-01-01T12:00");
        });

        it("should handle year 1901", () => {
            // Note: JavaScript Date constructor with year < 100 adds 1900 to it
            // So new Date(1, 0, 1) actually creates year 1901, not year 1
            const date = new Date(1, 0, 1, 0, 0); // Jan 1, 1901, 12:00 AM
            const result = toDatetimeLocal(date);
            expect(result).to.equal("1901-01-01T00:00");
        });

        it("should handle actual year 1", () => {
            // To create actual year 1, we need to use setFullYear
            const date = new Date();
            date.setFullYear(1, 0, 1);
            date.setHours(0, 0, 0, 0);
            const result = toDatetimeLocal(date);
            expect(result).to.equal("0001-01-01T00:00");
        });

        it("should create a new Date object internally", () => {
            const originalDate = new Date(2023, 5, 15, 10, 30);
            const result = toDatetimeLocal(originalDate);
            expect(result).to.equal("2023-06-15T10:30");
            // Verify original date is unchanged
            expect(originalDate.getFullYear()).to.equal(2023);
        });
    });

    describe("fromDatetimeLocal", () => {
        it("should parse a datetime-local string correctly", () => {
            const datetimeStr = "2023-12-25T14:30";
            const result = fromDatetimeLocal(datetimeStr);
            expect(result.getFullYear()).to.equal(2023);
            expect(result.getMonth()).to.equal(11); // December (0-indexed)
            expect(result.getDate()).to.equal(25);
            expect(result.getHours()).to.equal(14);
            expect(result.getMinutes()).to.equal(30);
        });

        it("should parse single digit values correctly", () => {
            const datetimeStr = "2023-01-05T08:05";
            const result = fromDatetimeLocal(datetimeStr);
            expect(result.getFullYear()).to.equal(2023);
            expect(result.getMonth()).to.equal(0); // January
            expect(result.getDate()).to.equal(5);
            expect(result.getHours()).to.equal(8);
            expect(result.getMinutes()).to.equal(5);
        });

        it("should parse midnight correctly", () => {
            const datetimeStr = "2023-07-15T00:00";
            const result = fromDatetimeLocal(datetimeStr);
            expect(result.getFullYear()).to.equal(2023);
            expect(result.getMonth()).to.equal(6); // July
            expect(result.getDate()).to.equal(15);
            expect(result.getHours()).to.equal(0);
            expect(result.getMinutes()).to.equal(0);
        });

        it("should parse end of day correctly", () => {
            const datetimeStr = "2023-12-31T23:59";
            const result = fromDatetimeLocal(datetimeStr);
            expect(result.getFullYear()).to.equal(2023);
            expect(result.getMonth()).to.equal(11); // December
            expect(result.getDate()).to.equal(31);
            expect(result.getHours()).to.equal(23);
            expect(result.getMinutes()).to.equal(59);
        });

        it("should throw error for missing T separator", () => {
            expect(() => fromDatetimeLocal("2023-12-25 14:30")).to.throw("Invalid datetime-local string");
        });

        it("should throw error for missing time part", () => {
            expect(() => fromDatetimeLocal("2023-12-25T")).to.throw("Invalid datetime-local string");
        });

        it("should throw error for missing date part", () => {
            expect(() => fromDatetimeLocal("T14:30")).to.throw("Invalid datetime-local string");
        });

        it("should throw error for invalid date format - wrong year length", () => {
            expect(() => fromDatetimeLocal("23-12-25T14:30")).to.throw("Invalid date format");
        });

        it("should throw error for invalid date format - wrong month length", () => {
            expect(() => fromDatetimeLocal("2023-1-25T14:30")).to.throw("Invalid date format");
        });

        it("should throw error for invalid date format - wrong day length", () => {
            expect(() => fromDatetimeLocal("2023-12-5T14:30")).to.throw("Invalid date format");
        });

        it("should throw error for invalid time format - wrong hour length", () => {
            expect(() => fromDatetimeLocal("2023-12-25T1:30")).to.throw("Invalid time format");
        });

        it("should throw error for invalid time format - wrong minute length", () => {
            expect(() => fromDatetimeLocal("2023-12-25T14:5")).to.throw("Invalid time format");
        });

        it("should throw error for missing date separators", () => {
            expect(() => fromDatetimeLocal("20231225T14:30")).to.throw("Invalid date format");
        });

        it("should throw error for missing time separator", () => {
            expect(() => fromDatetimeLocal("2023-12-25T1430")).to.throw("Invalid time format");
        });

        it("should throw error for empty string", () => {
            expect(() => fromDatetimeLocal("")).to.throw("Invalid datetime-local string");
        });

        it("should round-trip conversion correctly", () => {
            const originalDate = new Date(2023, 5, 15, 10, 30);
            const datetimeStr = toDatetimeLocal(originalDate);
            const parsedDate = fromDatetimeLocal(datetimeStr);
            
            expect(parsedDate.getFullYear()).to.equal(originalDate.getFullYear());
            expect(parsedDate.getMonth()).to.equal(originalDate.getMonth());
            expect(parsedDate.getDate()).to.equal(originalDate.getDate());
            expect(parsedDate.getHours()).to.equal(originalDate.getHours());
            expect(parsedDate.getMinutes()).to.equal(originalDate.getMinutes());
        });
    });
});