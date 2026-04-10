/**
 * Telemetry Uploader Tests
 *
 * DOC: docs/internal/SEAMS.md#telemetry-uploader-tests
 *
 * Tests for the telemetry uploader module.
 * Uses mock filesystem and HTTP client for testing.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import type { IFileSystem, IHttpClient, IPathUtils } from "../types";
import { createTelemetryUploader, type TelemetryUploaderConfig } from "../uploader";

// ===== Mock Factories =====

function createMockFileSystem(): IFileSystem & {
    _files: Map<string, string>;
    _stats: Map<string, { size: number; mtimeMs: number }>;
} {
    const files = new Map<string, string>();
    const stats = new Map<string, { size: number; mtimeMs: number }>();

    return {
        _files: files,
        _stats: stats,

        appendFile: vi.fn(async (path: string, content: string) => {
            const existing = files.get(path) ?? "";
            files.set(path, existing + content);
        }),

        readFile: vi.fn(async (path: string, _encoding: "utf-8") => {
            const content = files.get(path);
            if (content === undefined) {
                const error = new Error(`ENOENT: no such file or directory`);
                (error as NodeJS.ErrnoException).code = "ENOENT";
                throw error;
            }
            return content;
        }),

        writeFile: vi.fn(async (path: string, content: string) => {
            files.set(path, content);
        }),

        stat: vi.fn(async (path: string) => {
            const customStats = stats.get(path);
            if (customStats) return customStats;

            const content = files.get(path);
            if (content === undefined) {
                const error = new Error(`ENOENT: no such file or directory`);
                (error as NodeJS.ErrnoException).code = "ENOENT";
                throw error;
            }
            return {
                size: content.length,
                mtimeMs: 1704067200000, // 2024-01-01
            };
        }),

        mkdir: vi.fn(async () => {
            // No-op
        }),
    };
}

function createMockHttpClient(): IHttpClient & {
    _lastRequest: { url: string; body: string; headers: Record<string, string> | undefined } | null;
    _response: { ok: boolean; status: number; text: string };
} {
    const mock = {
        _lastRequest: null as { url: string; body: string; headers: Record<string, string> | undefined } | null,
        _response: { ok: true, status: 200, text: "" },

        post: vi.fn(async (url: string, body: string, headers?: Record<string, string>) => {
            mock._lastRequest = { url, body, headers };
            return {
                ok: mock._response.ok,
                status: mock._response.status,
                text: async () => mock._response.text,
            };
        }),
    };
    return mock;
}

function createMockPathUtils(): IPathUtils {
    return {
        join: (...segments) => segments.filter(Boolean).join("/"),
        dirname: (path) => {
            const parts = path.split("/");
            parts.pop();
            return parts.join("/") || "/";
        },
    };
}

function createTestConfig(overrides?: Partial<TelemetryUploaderConfig>): TelemetryUploaderConfig {
    return {
        scenarioName: "test-scenario",
        deploymentMode: "bundled",
        statePath: "/mock/state/telemetry-upload.json",
        ...overrides,
    };
}

// ===== Tests =====

describe("createTelemetryUploader", () => {
    let fs: ReturnType<typeof createMockFileSystem>;
    let http: ReturnType<typeof createMockHttpClient>;
    let pathUtils: IPathUtils;
    let config: TelemetryUploaderConfig;

    beforeEach(() => {
        fs = createMockFileSystem();
        http = createMockHttpClient();
        pathUtils = createMockPathUtils();
        config = createTestConfig();
    });

    describe("upload", () => {
        it("uploads telemetry events to the given URL", async () => {
            fs._files.set("/telemetry.jsonl", '{"event":"test1"}\n{"event":"test2"}\n');
            const uploader = createTelemetryUploader(fs, http, pathUtils, config);

            await uploader.upload("/telemetry.jsonl", "https://api.example.com/telemetry", "startup");

            expect(http.post).toHaveBeenCalled();
            expect(http._lastRequest?.url).toBe("https://api.example.com/telemetry");
            expect(http._lastRequest?.headers).toEqual({ "Content-Type": "application/json" });

            const body = JSON.parse(http._lastRequest?.body ?? "{}");
            expect(body.scenario_name).toBe("test-scenario");
            expect(body.deployment_mode).toBe("bundled");
            expect(body.source).toBe("desktop-runtime");
            expect(body.events).toHaveLength(2);
        });

        it("throws when file is empty", async () => {
            fs._files.set("/telemetry.jsonl", "");
            fs._stats.set("/telemetry.jsonl", { size: 0, mtimeMs: Date.now() });
            const uploader = createTelemetryUploader(fs, http, pathUtils, config);

            await expect(uploader.upload("/telemetry.jsonl", "https://api.example.com/telemetry", "startup"))
                .rejects.toThrow("Telemetry file is empty");
        });

        it("throws when no events found", async () => {
            fs._files.set("/telemetry.jsonl", "\n\n\n");
            const uploader = createTelemetryUploader(fs, http, pathUtils, config);

            await expect(uploader.upload("/telemetry.jsonl", "https://api.example.com/telemetry", "startup"))
                .rejects.toThrow("No telemetry events found to upload");
        });

        it("throws on HTTP error", async () => {
            fs._files.set("/telemetry.jsonl", '{"event":"test"}\n');
            http._response = { ok: false, status: 500, text: "Internal Server Error" };
            const uploader = createTelemetryUploader(fs, http, pathUtils, config);

            await expect(uploader.upload("/telemetry.jsonl", "https://api.example.com/telemetry", "startup"))
                .rejects.toThrow("Internal Server Error");
        });

        it("saves upload state after successful upload", async () => {
            fs._files.set("/telemetry.jsonl", '{"event":"test"}\n');
            const uploader = createTelemetryUploader(fs, http, pathUtils, config);

            await uploader.upload("/telemetry.jsonl", "https://api.example.com/telemetry", "startup");

            expect(fs.writeFile).toHaveBeenCalled();
            const stateContent = fs._files.get(config.statePath);
            expect(stateContent).toBeDefined();
            const state = JSON.parse(stateContent ?? "{}");
            expect(state.lastSignature).toContain("/telemetry.jsonl");
            expect(state.reason).toBe("startup");
        });

        it("calls onUploadSuccess callback", async () => {
            fs._files.set("/telemetry.jsonl", '{"event":"test1"}\n{"event":"test2"}\n');
            const onUploadSuccess = vi.fn();
            const uploaderConfig = createTestConfig({ onUploadSuccess });
            const uploader = createTelemetryUploader(fs, http, pathUtils, uploaderConfig);

            await uploader.upload("/telemetry.jsonl", "https://api.example.com/telemetry", "startup");

            expect(onUploadSuccess).toHaveBeenCalledWith(
                "https://api.example.com/telemetry",
                2,
                "startup"
            );
        });
    });

    describe("duplicate detection", () => {
        it("skips upload when signature matches", async () => {
            fs._files.set("/telemetry.jsonl", '{"event":"test"}\n');
            const uploader = createTelemetryUploader(fs, http, pathUtils, config);

            // First upload
            await uploader.upload("/telemetry.jsonl", "https://api.example.com/telemetry", "startup");
            expect(http.post).toHaveBeenCalledTimes(1);

            // Second upload with same signature should be skipped
            await uploader.upload("/telemetry.jsonl", "https://api.example.com/telemetry", "shutdown");
            expect(http.post).toHaveBeenCalledTimes(1);
        });

        it("uploads when signature changes", async () => {
            fs._files.set("/telemetry.jsonl", '{"event":"test"}\n');
            const uploader = createTelemetryUploader(fs, http, pathUtils, config);

            // First upload
            await uploader.upload("/telemetry.jsonl", "https://api.example.com/telemetry", "startup");
            expect(http.post).toHaveBeenCalledTimes(1);

            // Change file content (changes signature via size)
            fs._files.set("/telemetry.jsonl", '{"event":"test"}\n{"event":"test2"}\n');

            // Second upload should proceed
            await uploader.upload("/telemetry.jsonl", "https://api.example.com/telemetry", "shutdown");
            expect(http.post).toHaveBeenCalledTimes(2);
        });

        it("uploads when URL changes", async () => {
            fs._files.set("/telemetry.jsonl", '{"event":"test"}\n');
            const uploader = createTelemetryUploader(fs, http, pathUtils, config);

            await uploader.upload("/telemetry.jsonl", "https://api.example.com/telemetry", "startup");
            expect(http.post).toHaveBeenCalledTimes(1);

            // Different URL should trigger new upload
            await uploader.upload("/telemetry.jsonl", "https://api.other.com/telemetry", "startup");
            expect(http.post).toHaveBeenCalledTimes(2);
        });

        it("force flag bypasses signature check", async () => {
            fs._files.set("/telemetry.jsonl", '{"event":"test"}\n');
            const uploader = createTelemetryUploader(fs, http, pathUtils, config);

            await uploader.upload("/telemetry.jsonl", "https://api.example.com/telemetry", "startup");
            expect(http.post).toHaveBeenCalledTimes(1);

            // Force upload with same signature
            await uploader.upload("/telemetry.jsonl", "https://api.example.com/telemetry", "manual", true);
            expect(http.post).toHaveBeenCalledTimes(2);
        });

        it("loads state from disk on first upload", async () => {
            // Pre-populate files
            const telemetryContent = '{"event":"test"}\n';
            fs._files.set("/telemetry.jsonl", telemetryContent);

            // Pre-populate state file with matching signature
            // Signature format: filePath:mtimeMs:size:uploadURL
            fs._files.set(config.statePath, JSON.stringify({
                lastSignature: `/telemetry.jsonl:1704067200000:${telemetryContent.length}:https://api.example.com/telemetry`,
            }));

            const uploader = createTelemetryUploader(fs, http, pathUtils, config);

            // Should skip because signature matches stored state
            await uploader.upload("/telemetry.jsonl", "https://api.example.com/telemetry", "startup");
            expect(http.post).not.toHaveBeenCalled();
        });
    });
});
