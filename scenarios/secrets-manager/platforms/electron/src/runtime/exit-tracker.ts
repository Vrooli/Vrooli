/**
 * Runtime Exit Tracker
 *
 * DOC: docs/internal/SEAMS.md#runtime-exit-tracker
 *
 * Tracks runtime process exit status for crash detection.
 * Allows smoke tests to detect runtime crashes after server ready check.
 */

import type { RuntimeExitInfo } from "./types";
import { createInitialExitInfo } from "./types";

/**
 * Create an exit tracker instance.
 */
export function createExitTracker(): {
    info: RuntimeExitInfo;
    reset(): void;
    recordExit(code: number | null, signal: NodeJS.Signals | null): void;
    appendStderr(chunk: string): void;
    hasExitedUnexpectedly(): boolean;
} {
    let info = createInitialExitInfo();

    return {
        get info() {
            return info;
        },

        reset() {
            info = createInitialExitInfo();
        },

        recordExit(code: number | null, signal: NodeJS.Signals | null) {
            info.exited = true;
            info.code = code;
            info.signal = signal;
            info.exitedAt = new Date();
        },

        appendStderr(chunk: string) {
            info.stderr += chunk;
        },

        /**
         * Check if runtime has exited unexpectedly (non-zero exit or signal).
         */
        hasExitedUnexpectedly(): boolean {
            if (!info.exited) return false;
            // Exit code 0 is normal shutdown, null with signal means killed
            return info.code !== 0;
        },
    };
}

/**
 * Error patterns for structured error reporting.
 */
export interface ErrorPattern {
    pattern: RegExp;
    kind: "config" | "network" | "runtime" | "validation";
    message: string;
}

/**
 * Default error patterns for runtime stderr analysis.
 */
export const DEFAULT_ERROR_PATTERNS: ErrorPattern[] = [
    {
        pattern: /no go\.mod|staleness check/i,
        kind: "config",
        message: "Go binary staleness check failed - VROOLI_API_SKIP_STALE_CHECK should be set",
    },
    {
        pattern: /ECONNREFUSED|connection refused/i,
        kind: "network",
        message: "Connection refused - target service may not be running",
    },
    {
        pattern: /ENOENT|no such file/i,
        kind: "config",
        message: "Required file or binary not found in bundle",
    },
    {
        pattern: /permission denied|EACCES/i,
        kind: "runtime",
        message: "Permission denied accessing file or port",
    },
    {
        pattern: /address already in use|EADDRINUSE/i,
        kind: "network",
        message: "Port already in use - another instance may be running",
    },
];

/**
 * Check stderr for matching error patterns.
 * @returns The first matching error pattern, or null if none match.
 */
export function matchErrorPattern(
    stderr: string,
    patterns: ErrorPattern[] = DEFAULT_ERROR_PATTERNS
): ErrorPattern | null {
    for (const pattern of patterns) {
        if (pattern.pattern.test(stderr)) {
            return pattern;
        }
    }
    return null;
}
