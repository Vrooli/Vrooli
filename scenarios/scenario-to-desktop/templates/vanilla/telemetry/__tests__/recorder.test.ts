/**
 * Telemetry Recorder Tests
 *
 * DOC: docs/internal/SEAMS.md#telemetry-recorder-tests
 *
 * Tests for the telemetry recorder module.
 * Uses mock filesystem to test without disk access.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import type { IFileSystem, TelemetryConfig, TelemetryEvent } from "../types";
import { createTelemetryRecorder, readTelemetryEvents } from "../recorder";

// ===== Mock Factories =====

function createMockFileSystem(): IFileSystem & {
    _files: Map<string, string>;
    _appendCalls: Array<{ path: string; content: string }>;
} {
    const files = new Map<string, string>();
    const appendCalls: Array<{ path: string; content: string }> = [];

    return {
        _files: files,
        _appendCalls: appendCalls,

        appendFile: vi.fn(async (path: string, content: string) => {
            appendCalls.push({ path, content });
            const existing = files.get(path) ?? "";
            files.set(path, existing + content);
        }),

        readFile: vi.fn(async (path: string, _encoding: "utf-8") => {
            const content = files.get(path);
            if (content === undefined) {
                const error = new Error(`ENOENT: no such file or directory, open '${path}'`);
                (error as NodeJS.ErrnoException).code = "ENOENT";
                throw error;
            }
            return content;
        }),

        writeFile: vi.fn(async (path: string, content: string) => {
            files.set(path, content);
        }),

        stat: vi.fn(async (path: string) => {
            const content = files.get(path);
            if (content === undefined) {
                const error = new Error(`ENOENT: no such file or directory, stat '${path}'`);
                (error as NodeJS.ErrnoException).code = "ENOENT";
                throw error;
            }
            return {
                size: content.length,
                mtimeMs: Date.now(),
            };
        }),

        mkdir: vi.fn(async (_path: string, _options?: { recursive?: boolean }) => {
            // No-op for tests
        }),
    };
}

function createTestConfig(overrides?: Partial<TelemetryConfig>): TelemetryConfig {
    return {
        sessionId: "test-session-123",
        sessionKind: "app",
        deploymentMode: "bundled",
        serverType: "node",
        filePath: "/mock/telemetry.jsonl",
        ...overrides,
    };
}

// ===== Tests =====

describe("createTelemetryRecorder", () => {
    let fs: ReturnType<typeof createMockFileSystem>;
    let config: TelemetryConfig;

    beforeEach(() => {
        fs = createMockFileSystem();
        config = createTestConfig();
    });

    describe("initialize", () => {
        it("creates or touches the telemetry file", async () => {
            const recorder = createTelemetryRecorder(fs, config);

            await recorder.initialize();

            expect(fs.appendFile).toHaveBeenCalledWith(config.filePath, "");
        });

        it("sets filePath after successful initialization", async () => {
            const recorder = createTelemetryRecorder(fs, config);

            expect(recorder.getFilePath()).toBeNull();
            await recorder.initialize();
            expect(recorder.getFilePath()).toBe(config.filePath);
        });

        it("throws on initialization failure", async () => {
            fs.appendFile = vi.fn().mockRejectedValue(new Error("Disk full"));
            const recorder = createTelemetryRecorder(fs, config);

            await expect(recorder.initialize()).rejects.toThrow("Disk full");
        });

        it("is idempotent - second call does nothing", async () => {
            const recorder = createTelemetryRecorder(fs, config);

            await recorder.initialize();
            await recorder.initialize();

            expect(fs.appendFile).toHaveBeenCalledTimes(1);
        });
    });

    describe("record", () => {
        it("writes event to file as JSONL", async () => {
            const recorder = createTelemetryRecorder(fs, config);
            await recorder.initialize();

            await recorder.record("test_event", { foo: "bar" });

            expect(fs._appendCalls.length).toBe(2); // initialize + record
            const recordCall = fs._appendCalls[1];
            expect(recordCall).toBeDefined();
            expect(recordCall?.path).toBe(config.filePath);

            const parsed = JSON.parse(recordCall?.content.replace("\n", "") ?? "{}") as TelemetryEvent;
            expect(parsed.event).toBe("test_event");
            expect(parsed.details).toEqual({ foo: "bar" });
            expect(parsed.level).toBe("info");
            expect(parsed.session_id).toBe(config.sessionId);
        });

        it("includes correct metadata in event", async () => {
            const recorder = createTelemetryRecorder(fs, config);
            await recorder.initialize();

            await recorder.record("test_event");

            const recordCall = fs._appendCalls[1];
            const parsed = JSON.parse(recordCall?.content.replace("\n", "") ?? "{}") as TelemetryEvent;
            expect(parsed.session_kind).toBe(config.sessionKind);
            expect(parsed.deploymentMode).toBe(config.deploymentMode);
            expect(parsed.serverType).toBe(config.serverType);
            expect(parsed.timestamp).toMatch(/^\d{4}-\d{2}-\d{2}T/);
        });

        it("respects level parameter", async () => {
            const recorder = createTelemetryRecorder(fs, config);
            await recorder.initialize();

            await recorder.record("error_event", {}, "error");

            const recordCall = fs._appendCalls[1];
            const parsed = JSON.parse(recordCall?.content.replace("\n", "") ?? "{}") as TelemetryEvent;
            expect(parsed.level).toBe("error");
        });

        it("does nothing if not initialized", async () => {
            const recorder = createTelemetryRecorder(fs, config);

            await recorder.record("test_event");

            expect(fs.appendFile).not.toHaveBeenCalled();
        });

        it("handles write errors gracefully", async () => {
            const recorder = createTelemetryRecorder(fs, config);
            await recorder.initialize();

            // Make subsequent writes fail
            fs.appendFile = vi.fn().mockRejectedValue(new Error("Write error"));

            // Should not throw
            await expect(recorder.record("test_event")).resolves.toBeUndefined();
        });
    });

    describe("recordSessionOutcome", () => {
        it("records success when readyAt is set", async () => {
            const recorder = createTelemetryRecorder(fs, config);
            await recorder.initialize();
            recorder.setSessionStarted("2024-01-01T00:00:00Z");
            recorder.setSessionReady("2024-01-01T00:00:05Z");

            await recorder.recordSessionOutcome();

            const recordCall = fs._appendCalls[1];
            const parsed = JSON.parse(recordCall?.content.replace("\n", "") ?? "{}") as TelemetryEvent;
            expect(parsed.event).toBe("app_session_succeeded");
            expect(parsed.level).toBe("info");
            expect(parsed.details.started_at).toBe("2024-01-01T00:00:00Z");
            expect(parsed.details.ready_at).toBe("2024-01-01T00:00:05Z");
        });

        it("records failure when readyAt is not set", async () => {
            const recorder = createTelemetryRecorder(fs, config);
            await recorder.initialize();
            recorder.setSessionStarted("2024-01-01T00:00:00Z");

            await recorder.recordSessionOutcome();

            const recordCall = fs._appendCalls[1];
            const parsed = JSON.parse(recordCall?.content.replace("\n", "") ?? "{}") as TelemetryEvent;
            expect(parsed.event).toBe("app_session_failed");
            expect(parsed.level).toBe("error");
            expect(parsed.details.reason).toBe("app_exit_before_ready");
        });

        it("uses failureMessage if set", async () => {
            const recorder = createTelemetryRecorder(fs, config);
            await recorder.initialize();
            recorder.setSessionFailure("Server crashed");

            await recorder.recordSessionOutcome();

            const recordCall = fs._appendCalls[1];
            const parsed = JSON.parse(recordCall?.content.replace("\n", "") ?? "{}") as TelemetryEvent;
            expect(parsed.details.reason).toBe("Server crashed");
        });

        it("uses provided reason parameter", async () => {
            const recorder = createTelemetryRecorder(fs, config);
            await recorder.initialize();

            await recorder.recordSessionOutcome("User cancelled");

            const recordCall = fs._appendCalls[1];
            const parsed = JSON.parse(recordCall?.content.replace("\n", "") ?? "{}") as TelemetryEvent;
            expect(parsed.details.reason).toBe("User cancelled");
        });

        it("can only be called once", async () => {
            const recorder = createTelemetryRecorder(fs, config);
            await recorder.initialize();

            await recorder.recordSessionOutcome("First");
            await recorder.recordSessionOutcome("Second");

            // Should only have initialize + one outcome record
            expect(fs._appendCalls.length).toBe(2);
        });

        it("does nothing if not initialized", async () => {
            const recorder = createTelemetryRecorder(fs, config);

            await recorder.recordSessionOutcome();

            expect(fs.appendFile).not.toHaveBeenCalled();
        });
    });
});

describe("readTelemetryEvents", () => {
    let fs: ReturnType<typeof createMockFileSystem>;

    beforeEach(() => {
        fs = createMockFileSystem();
    });

    it("parses JSONL file into events array", async () => {
        fs._files.set("/telemetry.jsonl",
            '{"event":"a"}\n{"event":"b"}\n{"event":"c"}\n'
        );

        const events = await readTelemetryEvents(fs, "/telemetry.jsonl");

        expect(events).toHaveLength(3);
        expect(events[0]).toEqual({ event: "a" });
        expect(events[1]).toEqual({ event: "b" });
        expect(events[2]).toEqual({ event: "c" });
    });

    it("handles empty lines", async () => {
        fs._files.set("/telemetry.jsonl",
            '{"event":"a"}\n\n{"event":"b"}\n  \n{"event":"c"}\n'
        );

        const events = await readTelemetryEvents(fs, "/telemetry.jsonl");

        expect(events).toHaveLength(3);
    });

    it("handles CRLF line endings", async () => {
        fs._files.set("/telemetry.jsonl",
            '{"event":"a"}\r\n{"event":"b"}\r\n'
        );

        const events = await readTelemetryEvents(fs, "/telemetry.jsonl");

        expect(events).toHaveLength(2);
    });

    it("respects limit parameter", async () => {
        const lines = Array.from({ length: 100 }, (_, i) => `{"event":"${i}"}`).join("\n");
        fs._files.set("/telemetry.jsonl", lines);

        const events = await readTelemetryEvents(fs, "/telemetry.jsonl", 10);

        expect(events).toHaveLength(10);
        expect(events[0]).toEqual({ event: "0" });
        expect(events[9]).toEqual({ event: "9" });
    });

    it("throws on invalid JSON", async () => {
        fs._files.set("/telemetry.jsonl",
            '{"event":"a"}\nnot json\n{"event":"c"}\n'
        );

        await expect(readTelemetryEvents(fs, "/telemetry.jsonl"))
            .rejects.toThrow("Telemetry line 2 is invalid JSON");
    });

    it("throws on missing file", async () => {
        await expect(readTelemetryEvents(fs, "/nonexistent.jsonl"))
            .rejects.toThrow("ENOENT");
    });
});
