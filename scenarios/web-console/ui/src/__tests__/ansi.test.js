import { describe, it, expect } from "vitest";
import { ANSI } from "../lib/ansi";
// [REQ:P0-002b] WebSocket I/O Streaming - terminal status messages
describe("ANSI escape codes", () => {
    it("gray starts with ESC[", () => {
        expect(ANSI.gray).toBe("\x1b[90m");
    });
    it("red starts with ESC[", () => {
        expect(ANSI.red).toBe("\x1b[31m");
    });
    it("reset is the standard reset sequence", () => {
        expect(ANSI.reset).toBe("\x1b[0m");
    });
    it("all values are non-empty strings", () => {
        for (const value of Object.values(ANSI)) {
            expect(typeof value).toBe("string");
            expect(value.length).toBeGreaterThan(0);
        }
    });
});
