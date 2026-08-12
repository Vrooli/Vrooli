import { describe, it, expect } from "vitest";
import { getShellName, formatSessionTime, truncateId, parseDurationMs, formatCountdown } from "../lib/format";
// [REQ:P0-008b] Session Status and Controls - formatting utilities
describe("format utilities", () => {
    describe("getShellName", () => {
        it("extracts name from absolute path", () => {
            expect(getShellName("/bin/bash")).toBe("bash");
        });
        it("extracts name from nested path", () => {
            expect(getShellName("/usr/local/bin/zsh")).toBe("zsh");
        });
        it("returns the string itself when no slashes", () => {
            expect(getShellName("bash")).toBe("bash");
        });
        it("handles trailing slash gracefully with fallback", () => {
            expect(getShellName("/bin/")).toBe("shell");
        });
        it("handles empty string with fallback", () => {
            expect(getShellName("")).toBe("shell");
        });
    });
    describe("formatSessionTime", () => {
        it("returns a locale time string for valid ISO input", () => {
            const result = formatSessionTime("2026-01-15T14:30:00Z");
            // Exact format depends on locale, but should be non-empty
            expect(result.length).toBeGreaterThan(0);
        });
        it("returns dash for invalid date string", () => {
            expect(formatSessionTime("not-a-date")).toBe("—");
        });
        it("returns dash for empty string", () => {
            expect(formatSessionTime("")).toBe("—");
        });
    });
    describe("truncateId", () => {
        it("truncates to 8 characters by default", () => {
            expect(truncateId("abcdefgh-1234-5678")).toBe("abcdefgh...");
        });
        it("supports custom length", () => {
            expect(truncateId("abcdefgh-1234", 4)).toBe("abcd...");
        });
        it("returns dash for empty string", () => {
            expect(truncateId("")).toBe("—");
        });
    });
    describe("parseDurationMs", () => {
        it("parses hours", () => {
            expect(parseDurationMs("1h")).toBe(3600000);
            expect(parseDurationMs("8h")).toBe(8 * 3600000);
        });
        it("parses minutes", () => {
            expect(parseDurationMs("30m")).toBe(30 * 60000);
        });
        it("parses seconds", () => {
            expect(parseDurationMs("15s")).toBe(15000);
        });
        it("returns 0 for undefined", () => {
            expect(parseDurationMs(undefined)).toBe(0);
        });
        it("returns 0 for empty string", () => {
            expect(parseDurationMs("")).toBe(0);
        });
        it("returns 0 for unrecognized format", () => {
            expect(parseDurationMs("5d")).toBe(0);
            expect(parseDurationMs("abc")).toBe(0);
        });
    });
    describe("formatCountdown", () => {
        it("formats hours and minutes", () => {
            expect(formatCountdown(7500)).toBe("2h 5m");
        });
        it("formats minutes and seconds", () => {
            expect(formatCountdown(125)).toBe("2m 5s");
        });
        it("formats seconds only", () => {
            expect(formatCountdown(45)).toBe("45s");
        });
        it("returns expired for zero", () => {
            expect(formatCountdown(0)).toBe("expired");
        });
        it("returns expired for negative", () => {
            expect(formatCountdown(-10)).toBe("expired");
        });
    });
});
