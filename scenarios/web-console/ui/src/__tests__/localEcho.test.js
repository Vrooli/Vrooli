import { describe, it, expect, beforeEach } from "vitest";
import { LocalEchoController } from "../lib/localEcho";
function fakeClock(startMs = 0) {
    let now = startMs;
    const clock = () => now;
    clock.advance = (ms) => { now += ms; };
    clock.set = (ms) => { now = ms; };
    return clock;
}
describe("LocalEchoController", () => {
    let echo;
    beforeEach(() => {
        echo = new LocalEchoController();
    });
    describe("handleInput", () => {
        it("echoes a single printable character", () => {
            expect(echo.handleInput("a")).toBe("a");
            expect(echo.pendingCount).toBe(1);
        });
        it("echoes space", () => {
            expect(echo.handleInput(" ")).toBe(" ");
            expect(echo.pendingCount).toBe(1);
        });
        it("echoes tilde (highest printable ASCII)", () => {
            expect(echo.handleInput("~")).toBe("~");
        });
        it("returns null for control characters", () => {
            expect(echo.handleInput("\x03")).toBeNull(); // Ctrl-C
            expect(echo.handleInput("\t")).toBeNull(); // Tab
            expect(echo.handleInput("\r")).toBeNull(); // Enter
            expect(echo.handleInput("\n")).toBeNull(); // Newline
            expect(echo.pendingCount).toBe(0);
        });
        it("returns null for escape sequences", () => {
            expect(echo.handleInput("\x1b")).toBeNull();
            expect(echo.pendingCount).toBe(0);
        });
        it("returns null for DEL (0x7f)", () => {
            expect(echo.handleInput("\x7f")).toBeNull();
            expect(echo.pendingCount).toBe(0);
        });
        it("returns null for multi-char paste", () => {
            expect(echo.handleInput("hello")).toBeNull();
            expect(echo.pendingCount).toBe(0);
        });
        it("returns null when disabled", () => {
            echo.enabled = false;
            expect(echo.handleInput("a")).toBeNull();
            expect(echo.pendingCount).toBe(0);
        });
        it("returns null for multi-byte unicode (surrogate pairs)", () => {
            // Emoji is length > 1 due to surrogate pairs
            expect(echo.handleInput("😀")).toBeNull();
            expect(echo.pendingCount).toBe(0);
        });
    });
    describe("processOutput", () => {
        it("passes through data when no predictions pending", () => {
            expect(echo.processOutput("hello")).toBe("hello");
        });
        it("suppresses matching server echo", () => {
            echo.handleInput("a");
            expect(echo.processOutput("a")).toBe("");
            expect(echo.pendingCount).toBe(0);
        });
        it("suppresses matching prefix and passes through extra output", () => {
            echo.handleInput("l");
            echo.handleInput("s");
            // Server echoes "ls" then sends "\r\nfile1\r\n"
            expect(echo.processOutput("ls\r\nfile1\r\n")).toBe("\r\nfile1\r\n");
            expect(echo.pendingCount).toBe(0);
        });
        it("drops predictions on mismatch and returns full server data", () => {
            echo.handleInput("a");
            echo.handleInput("b");
            // Server sends something completely different (e.g., tab completion).
            const result = echo.processOutput("xyz");
            // Predictions are dropped silently; server bytes pass through
            // unchanged. No backspace erasure is written — xterm will repaint
            // from the server-authoritative stream.
            expect(result).toBe("xyz");
            expect(echo.pendingCount).toBe(0);
        });
        it("handles partial match then mismatch", () => {
            echo.handleInput("a");
            echo.handleInput("b");
            echo.handleInput("c");
            // Server echoes "a" then diverges with "XY".
            const result = echo.processOutput("aXY");
            // "a" matches and is consumed; on mismatch "b" and "c" are dropped
            // and the unmatched server tail passes through verbatim.
            expect(result).toBe("XY");
            expect(echo.pendingCount).toBe(0);
        });
        it("handles server echoing only some predicted chars", () => {
            echo.handleInput("a");
            echo.handleInput("b");
            echo.handleInput("c");
            // Server only echoes "a"
            expect(echo.processOutput("a")).toBe("");
            expect(echo.pendingCount).toBe(2); // "b" and "c" still pending
        });
    });
    describe("reset", () => {
        it("clears all pending predictions", () => {
            echo.handleInput("a");
            echo.handleInput("b");
            echo.reset();
            expect(echo.pendingCount).toBe(0);
        });
    });
    describe("enabled", () => {
        it("clears predictions when disabled", () => {
            echo.handleInput("a");
            echo.handleInput("b");
            echo.enabled = false;
            expect(echo.pendingCount).toBe(0);
        });
        it("defaults to enabled", () => {
            expect(echo.enabled).toBe(true);
        });
        it("can be re-enabled", () => {
            echo.enabled = false;
            echo.enabled = true;
            expect(echo.handleInput("a")).toBe("a");
        });
    });
    describe("prediction aging", () => {
        it("auto-resets stale predictions in handleInput", () => {
            const clock = fakeClock(1000);
            const ec = new LocalEchoController(clock);
            ec.handleInput("a");
            ec.handleInput("b");
            expect(ec.pendingCount).toBe(2);
            // Advance past MAX_PREDICTION_AGE_MS (2000ms)
            clock.advance(2001);
            ec.handleInput("c");
            // Stale predictions cleared, only "c" remains
            expect(ec.pendingCount).toBe(1);
        });
        it("discards stale predictions in processOutput", () => {
            const clock = fakeClock(1000);
            const ec = new LocalEchoController(clock);
            ec.handleInput("a");
            ec.handleInput("b");
            clock.advance(2001);
            // processOutput should discard stale predictions and pass through data unchanged
            expect(ec.processOutput("xyz")).toBe("xyz");
            expect(ec.pendingCount).toBe(0);
        });
        it("does not reset predictions within age limit", () => {
            const clock = fakeClock(1000);
            const ec = new LocalEchoController(clock);
            ec.handleInput("a");
            ec.handleInput("b");
            clock.advance(1999);
            // Within age limit — normal reconciliation
            expect(ec.processOutput("ab")).toBe("");
            expect(ec.pendingCount).toBe(0);
        });
        it("does not reset in handleInput within age limit", () => {
            const clock = fakeClock(1000);
            const ec = new LocalEchoController(clock);
            ec.handleInput("a");
            clock.advance(1999);
            ec.handleInput("b");
            expect(ec.pendingCount).toBe(2);
        });
    });
    describe("prediction cap", () => {
        it("stops echoing and resets when cap is reached", () => {
            const ec = new LocalEchoController();
            // Fill to the cap (32)
            for (let i = 0; i < 32; i++) {
                expect(ec.handleInput("a")).toBe("a");
            }
            expect(ec.pendingCount).toBe(32);
            // 33rd input should hit cap, reset, and return null
            expect(ec.handleInput("b")).toBeNull();
            expect(ec.pendingCount).toBe(0);
        });
        it("can echo again after cap reset", () => {
            const ec = new LocalEchoController();
            for (let i = 0; i < 32; i++) {
                ec.handleInput("a");
            }
            // Hit cap
            ec.handleInput("b");
            expect(ec.pendingCount).toBe(0);
            // Should be able to echo again
            expect(ec.handleInput("c")).toBe("c");
            expect(ec.pendingCount).toBe(1);
        });
    });
    describe("ANSI escape handling", () => {
        it("passes through data unchanged when server output starts with ESC", () => {
            echo.handleInput("a");
            echo.handleInput("b");
            const data = "\x1b[32mgreen\x1b[0m";
            expect(echo.processOutput(data)).toBe(data);
            expect(echo.pendingCount).toBe(0);
        });
        it("clears predictions when ESC detected (no erase sequences)", () => {
            echo.handleInput("a");
            echo.handleInput("b");
            echo.handleInput("c");
            const data = "\x1b[1m bold prompt";
            const result = echo.processOutput(data);
            // Should NOT contain backspace erase sequences
            expect(result).not.toContain("\b");
            expect(result).toBe(data);
            expect(echo.pendingCount).toBe(0);
        });
        it("still reconciles when ANSI appears mid-string", () => {
            echo.handleInput("l");
            echo.handleInput("s");
            // Server echoes "ls" then sends colored output
            const result = echo.processOutput("ls\r\n\x1b[32mfile\x1b[0m");
            expect(result).toBe("\r\n\x1b[32mfile\x1b[0m");
            expect(echo.pendingCount).toBe(0);
        });
        it("passes through ESC data unchanged with no pending predictions", () => {
            const data = "\x1b[31mred\x1b[0m";
            expect(echo.processOutput(data)).toBe(data);
        });
    });
    describe("clock injection", () => {
        it("accepts a custom clock for testability", () => {
            const clock = fakeClock(5000);
            const ec = new LocalEchoController(clock);
            ec.handleInput("a");
            expect(ec.pendingCount).toBe(1);
            // Without advancing clock, predictions stay fresh
            expect(ec.processOutput("a")).toBe("");
        });
    });
});
