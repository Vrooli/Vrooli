/**
 * Exit Tracker Tests
 *
 * DOC: docs/internal/SEAMS.md#exit-tracker-tests
 *
 * Tests for the runtime exit tracking module.
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
    createExitTracker,
    matchErrorPattern,
    DEFAULT_ERROR_PATTERNS,
    type ErrorPattern,
} from "../exit-tracker";

describe("createExitTracker", () => {
    describe("initial state", () => {
        it("starts with exited: false", () => {
            const tracker = createExitTracker();
            expect(tracker.info.exited).toBe(false);
        });

        it("starts with null code and signal", () => {
            const tracker = createExitTracker();
            expect(tracker.info.code).toBeNull();
            expect(tracker.info.signal).toBeNull();
        });

        it("starts with empty stderr", () => {
            const tracker = createExitTracker();
            expect(tracker.info.stderr).toBe("");
        });

        it("starts with null exitedAt", () => {
            const tracker = createExitTracker();
            expect(tracker.info.exitedAt).toBeNull();
        });
    });

    describe("reset", () => {
        it("resets to initial state", () => {
            const tracker = createExitTracker();
            tracker.recordExit(1, null);
            tracker.appendStderr("Error message");

            tracker.reset();

            expect(tracker.info.exited).toBe(false);
            expect(tracker.info.code).toBeNull();
            expect(tracker.info.signal).toBeNull();
            expect(tracker.info.stderr).toBe("");
            expect(tracker.info.exitedAt).toBeNull();
        });
    });

    describe("recordExit", () => {
        it("records exit code", () => {
            const tracker = createExitTracker();
            tracker.recordExit(1, null);

            expect(tracker.info.exited).toBe(true);
            expect(tracker.info.code).toBe(1);
            expect(tracker.info.signal).toBeNull();
            expect(tracker.info.exitedAt).toBeInstanceOf(Date);
        });

        it("records signal", () => {
            const tracker = createExitTracker();
            tracker.recordExit(null, "SIGTERM");

            expect(tracker.info.exited).toBe(true);
            expect(tracker.info.code).toBeNull();
            expect(tracker.info.signal).toBe("SIGTERM");
        });

        it("records both code and signal", () => {
            const tracker = createExitTracker();
            tracker.recordExit(137, "SIGKILL");

            expect(tracker.info.code).toBe(137);
            expect(tracker.info.signal).toBe("SIGKILL");
        });
    });

    describe("appendStderr", () => {
        it("accumulates stderr chunks", () => {
            const tracker = createExitTracker();
            tracker.appendStderr("Error: ");
            tracker.appendStderr("Something went wrong\n");
            tracker.appendStderr("Stack trace...");

            expect(tracker.info.stderr).toBe("Error: Something went wrong\nStack trace...");
        });
    });

    describe("hasExitedUnexpectedly", () => {
        it("returns false when not exited", () => {
            const tracker = createExitTracker();
            expect(tracker.hasExitedUnexpectedly()).toBe(false);
        });

        it("returns false for exit code 0", () => {
            const tracker = createExitTracker();
            tracker.recordExit(0, null);
            expect(tracker.hasExitedUnexpectedly()).toBe(false);
        });

        it("returns true for non-zero exit code", () => {
            const tracker = createExitTracker();
            tracker.recordExit(1, null);
            expect(tracker.hasExitedUnexpectedly()).toBe(true);
        });

        it("returns true for exit code 137 (SIGKILL)", () => {
            const tracker = createExitTracker();
            tracker.recordExit(137, "SIGKILL");
            expect(tracker.hasExitedUnexpectedly()).toBe(true);
        });

        it("returns true for null code with signal (killed)", () => {
            const tracker = createExitTracker();
            tracker.recordExit(null, "SIGTERM");
            expect(tracker.hasExitedUnexpectedly()).toBe(true);
        });
    });
});

describe("matchErrorPattern", () => {
    describe("with default patterns", () => {
        it("matches ECONNREFUSED", () => {
            const result = matchErrorPattern("Error: connect ECONNREFUSED 127.0.0.1:8080");
            expect(result).not.toBeNull();
            expect(result?.kind).toBe("network");
        });

        it("matches connection refused", () => {
            const result = matchErrorPattern("Connection refused to server");
            expect(result).not.toBeNull();
            expect(result?.kind).toBe("network");
        });

        it("matches ENOENT", () => {
            const result = matchErrorPattern("Error: ENOENT: no such file or directory");
            expect(result).not.toBeNull();
            expect(result?.kind).toBe("config");
        });

        it("matches permission denied", () => {
            const result = matchErrorPattern("Error: permission denied opening port 80");
            expect(result).not.toBeNull();
            expect(result?.kind).toBe("runtime");
        });

        it("matches EADDRINUSE", () => {
            const result = matchErrorPattern("Error: listen EADDRINUSE: address already in use :::8080");
            expect(result).not.toBeNull();
            expect(result?.kind).toBe("network");
        });

        it("matches go.mod staleness check", () => {
            const result = matchErrorPattern("no go.mod file found in directory");
            expect(result).not.toBeNull();
            expect(result?.kind).toBe("config");
        });

        it("returns null for unrecognized error", () => {
            const result = matchErrorPattern("Some random error message");
            expect(result).toBeNull();
        });

        it("returns first matching pattern", () => {
            // "no such file" matches both ENOENT and no go.mod patterns
            const result = matchErrorPattern("no such file or directory");
            expect(result).not.toBeNull();
            // First pattern to match wins
            expect(result?.kind).toBe("config");
        });
    });

    describe("with custom patterns", () => {
        const customPatterns: ErrorPattern[] = [
            { pattern: /custom error/i, kind: "validation", message: "Custom validation error" },
        ];

        it("matches custom patterns", () => {
            const result = matchErrorPattern("Custom Error occurred", customPatterns);
            expect(result).not.toBeNull();
            expect(result?.kind).toBe("validation");
            expect(result?.message).toBe("Custom validation error");
        });

        it("returns null when no custom patterns match", () => {
            const result = matchErrorPattern("ECONNREFUSED", customPatterns);
            expect(result).toBeNull();
        });
    });
});

describe("DEFAULT_ERROR_PATTERNS", () => {
    it("has patterns for common errors", () => {
        expect(DEFAULT_ERROR_PATTERNS.length).toBeGreaterThan(0);

        // Check that we have network, config, and runtime errors covered
        const kinds = DEFAULT_ERROR_PATTERNS.map((p) => p.kind);
        expect(kinds).toContain("network");
        expect(kinds).toContain("config");
        expect(kinds).toContain("runtime");
    });
});
