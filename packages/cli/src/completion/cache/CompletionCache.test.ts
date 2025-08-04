import { promises as fs } from "fs";
import * as path from "path";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ConfigManager } from "../../utils/config.js";
import type { CompletionResult } from "../types.js";
import { CompletionCache } from "./CompletionCache.js";

vi.mock("fs", () => ({
    promises: {
        readFile: vi.fn(),
        writeFile: vi.fn(),
        access: vi.fn(),
        unlink: vi.fn(),
        mkdir: vi.fn(),
    },
}));

vi.mock("path", () => ({
    join: vi.fn((...args) => args.join("/")),
    dirname: vi.fn((path) => path.substring(0, path.lastIndexOf("/"))),
}));

describe("CompletionCache", () => {
    let cache: CompletionCache;
    let mockConfig: ConfigManager;
    let mockResults: CompletionResult[];
    let originalDateNow: typeof Date.now;

    beforeEach(() => {
        mockConfig = {
            getConfigDir: vi.fn().mockReturnValue("/config/dir"),
        } as unknown as ConfigManager;

        mockResults = [
            { value: "test1", description: "Test completion 1", type: "command" },
            { value: "test2", description: "Test completion 2", type: "command" },
        ];

        vi.mocked(fs.readFile).mockRejectedValue(new Error("File not found"));
        vi.mocked(fs.writeFile).mockResolvedValue(undefined);
        vi.mocked(fs.access).mockRejectedValue(new Error("File not found"));

        originalDateNow = Date.now;
        Date.now = vi.fn().mockReturnValue(1000000); // Fixed timestamp

        cache = new CompletionCache(mockConfig);
    });

    afterEach(() => {
        vi.clearAllMocks();
        Date.now = originalDateNow;
    });

    describe("constructor", () => {
        it("should initialize with config dir path", () => {
            expect(mockConfig.getConfigDir).toHaveBeenCalled();
            expect(path.join).toHaveBeenCalledWith("/config/dir", "completion-cache.json");
        });

        it("should load existing cache from disk", () => {
            expect(fs.readFile).toHaveBeenCalledWith("/config/dir/completion-cache.json", "utf-8");
        });
    });

    describe("get", () => {
        it("should return null for non-existent key", async () => {
            const result = await cache.get("non-existent");
            expect(result).toBeNull();
        });

        it("should return cached results for valid key", async () => {
            await cache.set("test-key", mockResults, 300);

            const result = await cache.get("test-key");
            expect(result).toEqual(mockResults);
        });

        it("should return null for expired entries", async () => {
            await cache.set("test-key", mockResults, 1); // 1 second TTL

            // Advance time beyond TTL
            Date.now = vi.fn().mockReturnValue(1000000 + 2000); // +2 seconds

            const result = await cache.get("test-key");
            expect(result).toBeNull();
        });
    });

    describe("set", () => {
        it("should store results with default TTL", async () => {
            await cache.set("test-key", mockResults);

            const result = await cache.get("test-key");
            expect(result).toEqual(mockResults);
        });

        it("should store results with custom TTL", async () => {
            await cache.set("test-key", mockResults, 600); // 10 minutes

            const result = await cache.get("test-key");
            expect(result).toEqual(mockResults);
        });

        it("should save to disk after setting", async () => {
            vi.mocked(fs.mkdir).mockResolvedValue(undefined);

            await cache.set("test-key", mockResults);

            // Allow time for async save
            await new Promise(resolve => setTimeout(resolve, 50));

            expect(fs.mkdir).toHaveBeenCalledWith("/config/dir", { recursive: true });
            expect(fs.writeFile).toHaveBeenCalledWith(
                "/config/dir/completion-cache.json",
                expect.any(String),
            );
        });
    });

    describe("clear", () => {
        it("should clear all cached entries", async () => {
            await cache.set("key1", mockResults);
            await cache.set("key2", mockResults);

            await cache.clear();

            expect(await cache.get("key1")).toBeNull();
            expect(await cache.get("key2")).toBeNull();
        });

        it("should delete cache file from disk", async () => {
            vi.mocked(fs.unlink).mockResolvedValue(undefined);

            await cache.set("key1", mockResults);
            await cache.clear();

            expect(fs.unlink).toHaveBeenCalledWith("/config/dir/completion-cache.json");
        });
    });

    describe("TTL handling", () => {
        it("should convert TTL from seconds to milliseconds", async () => {
            await cache.set("test-key", mockResults, 5); // 5 seconds

            // Advance time by 4 seconds (should still be valid)
            Date.now = vi.fn().mockReturnValue(1000000 + 4000);

            expect(await cache.get("test-key")).toEqual(mockResults);

            // Advance time by 6 seconds (should be expired)
            Date.now = vi.fn().mockReturnValue(1000000 + 6000);

            expect(await cache.get("test-key")).toBeNull();
        });
    });

    describe("error handling", () => {
        it("should handle file write errors gracefully", async () => {
            vi.mocked(fs.writeFile).mockRejectedValue(new Error("Permission denied"));

            await expect(cache.set("test-key", mockResults)).resolves.not.toThrow();
        });

        it("should handle file read errors during load", () => {
            vi.mocked(fs.readFile).mockRejectedValue(new Error("Permission denied"));

            expect(() => new CompletionCache(mockConfig)).not.toThrow();
        });
    });
});
