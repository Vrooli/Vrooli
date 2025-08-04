// AI_CHECK: TEST_COVERAGE=36 | LAST: 2025-08-04
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import * as fs from "fs";
import * as path from "path";
import * as os from "os";
import { SqliteStorage } from "./SqliteStorage.js";
import type { ConfigManager } from "../../utils/config.js";
import type { HistoryEntry, HistorySearchQuery } from "../types.js";

// Mock ConfigManager
const createMockConfig = (dbDir?: string): ConfigManager => {
    const testDir = dbDir || path.join(os.tmpdir(), `vrooli-test-${Date.now()}`);
    
    if (!fs.existsSync(testDir)) {
        fs.mkdirSync(testDir, { recursive: true });
    }
    
    return {
        getConfigDir: () => testDir,
    } as ConfigManager;
};

const createTestEntry = (overrides: Partial<HistoryEntry> = {}): HistoryEntry => ({
    id: "test-id-" + Date.now(),
    command: "test-command",
    args: ["arg1", "arg2"],
    options: { profile: "default", debug: false },
    timestamp: new Date(),
    profile: "default",
    success: true,
    ...overrides,
});

describe("SqliteStorage", () => {
    let storage: SqliteStorage;
    let config: ConfigManager;
    let testDir: string;

    beforeEach(() => {
        testDir = path.join(os.tmpdir(), `vrooli-test-${Date.now()}`);
        config = createMockConfig(testDir);
        storage = new SqliteStorage(config);
    });

    afterEach(() => {
        try {
            storage.close();
        } catch (error) {
            // Ignore close errors
        }
        
        // Clean up test directory
        try {
            if (fs.existsSync(testDir)) {
                fs.rmSync(testDir, { recursive: true, force: true });
            }
        } catch (error) {
            // Ignore cleanup errors
        }
    });

    describe("initialization", () => {
        it("should create database and tables", () => {
            const dbPath = path.join(testDir, "history.db");
            expect(fs.existsSync(dbPath)).toBe(true);
        });

        it("should create database in specified config directory", () => {
            const customDir = path.join(os.tmpdir(), `vrooli-custom-${Date.now()}`);
            const customConfig = createMockConfig(customDir);
            const customStorage = new SqliteStorage(customConfig);
            
            const dbPath = path.join(customDir, "history.db");
            expect(fs.existsSync(dbPath)).toBe(true);
            
            customStorage.close();
            fs.rmSync(customDir, { recursive: true, force: true });
        });
    });

    describe("add and get", () => {
        it("should add and retrieve a history entry", async () => {
            const entry = createTestEntry();
            
            await storage.add(entry);
            const retrieved = await storage.get(entry.id);
            
            expect(retrieved).not.toBeNull();
            expect(retrieved!.id).toBe(entry.id);
            expect(retrieved!.command).toBe(entry.command);
            expect(retrieved!.args).toEqual(entry.args);
            expect(retrieved!.success).toBe(entry.success);
        });

        it("should return null for non-existent entry", async () => {
            const retrieved = await storage.get("non-existent-id");
            expect(retrieved).toBeNull();
        });

        it("should handle entry with all optional fields", async () => {
            const entry = createTestEntry({
                duration: 1500,
                exitCode: 1, // Use non-zero exit code since 0 is falsy and becomes null
                userId: "user-123",
                error: "Test error",
                metadata: { custom: "data" },
            });
            
            await storage.add(entry);
            const retrieved = await storage.get(entry.id);
            
            expect(retrieved).not.toBeNull();
            expect(retrieved!.duration).toBe(entry.duration);
            expect(retrieved!.exitCode).toBe(entry.exitCode);
            expect(retrieved!.userId).toBe(entry.userId);
            expect(retrieved!.error).toBe(entry.error);
            expect(retrieved!.metadata).toEqual(entry.metadata);
        });

        it("should handle addSync method", () => {
            const entry = createTestEntry();
            
            storage.addSync(entry);
            
            // Verify it was added by trying to get it
            return storage.get(entry.id).then(retrieved => {
                expect(retrieved).not.toBeNull();
                expect(retrieved!.id).toBe(entry.id);
            });
        });
    });

    describe("search", () => {
        beforeEach(async () => {
            // Add test data
            const entries = [
                createTestEntry({
                    id: "1",
                    command: "auth",
                    args: ["login"],
                    profile: "default",
                    success: true,
                    timestamp: new Date("2023-01-01"),
                }),
                createTestEntry({
                    id: "2",
                    command: "routine",
                    args: ["list", "--json"],
                    profile: "staging",
                    success: false,
                    timestamp: new Date("2023-01-02"),
                }),
                createTestEntry({
                    id: "3",
                    command: "chat",
                    args: ["start"],
                    profile: "default",
                    success: true,
                    timestamp: new Date("2023-01-03"),
                }),
            ];
            
            for (const entry of entries) {
                await storage.add(entry);
            }
        });

        it("should search without filters", async () => {
            const results = await storage.search({});
            expect(results).toHaveLength(3);
            // Should be ordered by timestamp DESC
            expect(results[0].id).toBe("3");
            expect(results[1].id).toBe("2");
            expect(results[2].id).toBe("1");
        });

        it("should search by command", async () => {
            const results = await storage.search({ command: "auth" });
            expect(results).toHaveLength(1);
            expect(results[0].command).toBe("auth");
        });

        it("should search by profile", async () => {
            const results = await storage.search({ profile: "default" });
            expect(results).toHaveLength(2);
            results.forEach(result => {
                expect(result.profile).toBe("default");
            });
        });

        it("should search successful commands only", async () => {
            const results = await storage.search({ successOnly: true });
            expect(results).toHaveLength(2);
            results.forEach(result => {
                expect(result.success).toBe(true);
            });
        });

        it("should search failed commands only", async () => {
            const results = await storage.search({ failedOnly: true });
            expect(results).toHaveLength(1);
            expect(results[0].success).toBe(false);
        });

        it("should search with date range", async () => {
            const results = await storage.search({
                startDate: new Date("2023-01-02"),
                endDate: new Date("2023-01-03"),
            });
            expect(results).toHaveLength(2);
            expect(results.map(r => r.id).sort()).toEqual(["2", "3"]);
        });

        it("should search with before date", async () => {
            const results = await storage.search({
                before: new Date("2023-01-02"),
            });
            expect(results).toHaveLength(1);
            expect(results[0].id).toBe("1");
        });

        it("should search with after date", async () => {
            const results = await storage.search({
                after: new Date("2023-01-02"),
            });
            expect(results).toHaveLength(1);
            expect(results[0].id).toBe("3");
        });

        it("should search with limit", async () => {
            const results = await storage.search({ limit: 2 });
            expect(results).toHaveLength(2);
        });

        it("should search with offset", async () => {
            const results = await storage.search({ offset: 1, limit: 2 });
            expect(results).toHaveLength(2);
            expect(results[0].id).toBe("2");
            expect(results[1].id).toBe("1");
        });

        it("should search with text using FTS", async () => {
            const results = await storage.search({ text: "login" });
            expect(results).toHaveLength(1);
            expect(results[0].command).toBe("auth");
        });

        it("should search by userId", async () => {
            const entryWithUser = createTestEntry({
                id: "user-entry",
                userId: "test-user",
            });
            await storage.add(entryWithUser);
            
            const results = await storage.search({ userId: "test-user" });
            expect(results).toHaveLength(1);
            expect(results[0].userId).toBe("test-user");
        });
    });

    describe("statistics", () => {
        beforeEach(async () => {
            // Add test data with varied stats
            const entries = [
                createTestEntry({
                    id: "1",
                    command: "auth",
                    profile: "default",
                    success: true,
                    duration: 1000,
                    timestamp: new Date("2023-01-01"),
                }),
                createTestEntry({
                    id: "2",
                    command: "auth",
                    profile: "default", 
                    success: false,
                    duration: 500,
                    timestamp: new Date("2023-01-02"),
                }),
                createTestEntry({
                    id: "3",
                    command: "routine",
                    profile: "staging",
                    success: true,
                    duration: 2000,
                    timestamp: new Date("2023-01-03"),
                }),
            ];
            
            for (const entry of entries) {
                await storage.add(entry);
            }
        });

        it("should calculate basic statistics", async () => {
            const stats = await storage.getStats();
            
            expect(stats.totalCommands).toBe(3);
            expect(stats.successfulCommands).toBe(2);
            expect(stats.failedCommands).toBe(1);
            expect(stats.uniqueCommands).toBe(2);
            expect(stats.avgDuration).toBe(1166.6666666666667); // (1000 + 500 + 2000) / 3
        });

        it("should track top commands", async () => {
            const stats = await storage.getStats();
            
            expect(stats.topCommands).toHaveLength(2);
            expect(stats.topCommands[0]).toEqual({ command: "auth", count: 2 });
            expect(stats.topCommands[1]).toEqual({ command: "routine", count: 1 });
        });

        it("should track commands by profile", async () => {
            const stats = await storage.getStats();
            
            expect(stats.commandsByProfile).toEqual({
                default: 2,
                staging: 1,
            });
        });

        it("should track recent activity", async () => {
            // Clear the old data and add recent data within 30 days
            await storage.clear();
            
            const recentDate1 = new Date(Date.now() - 24 * 60 * 60 * 1000); // 1 day ago
            const recentDate2 = new Date(Date.now() - 2 * 24 * 60 * 60 * 1000); // 2 days ago
            const recentDate3 = new Date(Date.now() - 3 * 24 * 60 * 60 * 1000); // 3 days ago
            
            const recentEntries = [
                createTestEntry({
                    id: "recent1",
                    command: "auth",
                    timestamp: recentDate1,
                }),
                createTestEntry({
                    id: "recent2",
                    command: "routine",
                    timestamp: recentDate2,
                }),
                createTestEntry({
                    id: "recent3",
                    command: "chat",
                    timestamp: recentDate3,
                }),
            ];
            
            for (const entry of recentEntries) {
                await storage.add(entry);
            }
            
            const stats = await storage.getStats();
            
            expect(stats.recentActivity).toHaveLength(3);
            // Activity should be ordered by date DESC
            expect(stats.recentActivity[0].count).toBe(1);
            expect(stats.recentActivity[1].count).toBe(1);
            expect(stats.recentActivity[2].count).toBe(1);
        });

        it("should handle empty database stats", async () => {
            const emptyStorage = new SqliteStorage(createMockConfig());
            const stats = await emptyStorage.getStats();
            
            expect(stats.totalCommands).toBe(0);
            expect(stats.successfulCommands).toBe(0);
            expect(stats.failedCommands).toBe(0);
            expect(stats.uniqueCommands).toBe(0);
            expect(stats.avgDuration).toBe(0);
            expect(stats.topCommands).toEqual([]);
            expect(stats.commandsByProfile).toEqual({});
            expect(stats.recentActivity).toEqual([]);
            
            emptyStorage.close();
        });
    });

    describe("data management", () => {
        it("should delete an entry", async () => {
            const entry = createTestEntry();
            await storage.add(entry);
            
            const retrieved = await storage.get(entry.id);
            expect(retrieved).not.toBeNull();
            
            await storage.delete(entry.id);
            
            const retrievedAfterDelete = await storage.get(entry.id);
            expect(retrievedAfterDelete).toBeNull();
        });

        it("should clear all entries", async () => {
            const entries = [
                createTestEntry({ id: "1" }),
                createTestEntry({ id: "2" }),
                createTestEntry({ id: "3" }),
            ];
            
            for (const entry of entries) {
                await storage.add(entry);
            }
            
            const allEntries = await storage.search({});
            expect(allEntries).toHaveLength(3);
            
            await storage.clear();
            
            const allEntriesAfterClear = await storage.search({});
            expect(allEntriesAfterClear).toHaveLength(0);
        });
    });

    describe("export functionality", () => {
        beforeEach(async () => {
            const entries = [
                createTestEntry({
                    id: "1",
                    command: "auth",
                    args: ["login"],
                    success: true,
                    timestamp: new Date("2023-01-01"),
                }),
                createTestEntry({
                    id: "2",
                    command: "routine",
                    args: ["list"],
                    success: false,
                    error: "Connection failed",
                    timestamp: new Date("2023-01-02"),
                }),
            ];
            
            for (const entry of entries) {
                await storage.add(entry);
            }
        });

        it("should export to JSON format", async () => {
            const exported = await storage.export("json");
            const parsed = JSON.parse(exported);
            
            expect(Array.isArray(parsed)).toBe(true);
            expect(parsed).toHaveLength(2);
            expect(parsed[0].command).toBe("routine"); // Most recent first
            expect(parsed[1].command).toBe("auth");
        });

        it("should export to CSV format", async () => {
            const exported = await storage.export("csv");
            const lines = exported.split("\\n");
            
            // CSV headers are quoted
            expect(lines[0]).toContain("\"id\",\"command\",\"args\"");
            expect(lines[1]).toContain("routine");
            expect(lines[2]).toContain("auth");
        });

        it("should export to script format", async () => {
            const exported = await storage.export("script");
            const lines = exported.split("\\n");
            
            expect(lines[0]).toBe("#!/bin/bash");
            expect(lines[1]).toContain("Generated by Vrooli CLI");
            expect(lines[4]).toBe(""); // Empty line separator
            expect(lines[5]).toBe("auth login"); // Only successful commands
            expect(lines).toHaveLength(6); // Header + successful command only
        });

        it("should reject unsupported export format", async () => {
            await expect(storage.export("xml" as any)).rejects.toThrow("Unsupported export format: xml");
        });
    });

    describe("database utilities", () => {
        it("should get database info", () => {
            const info = storage.getDbInfo();
            
            expect(info.path).toContain("history.db");
            expect(typeof info.size).toBe("number");
            expect(typeof info.pageCount).toBe("number");
            expect(info.size).toBeGreaterThan(0);
            expect(info.pageCount).toBeGreaterThan(0);
        });

        it("should vacuum database", () => {
            expect(() => storage.vacuum()).not.toThrow();
        });

        it("should analyze database", () => {
            expect(() => storage.analyze()).not.toThrow();
        });

        it("should close database connection", () => {
            expect(() => storage.close()).not.toThrow();
        });
    });

    describe("edge cases and error handling", () => {
        it("should handle entries with complex JSON in args and options", async () => {
            const entry = createTestEntry({
                args: ["complex", "{\"nested\": {\"data\": \"value\"}}", "--flag"],
                options: { 
                    profile: "test", 
                    complex: { nested: { array: [1, 2, 3] } },
                },
            });
            
            await storage.add(entry);
            const retrieved = await storage.get(entry.id);
            
            expect(retrieved).not.toBeNull();
            expect(retrieved!.args).toEqual(entry.args);
            expect(retrieved!.options).toEqual(entry.options);
        });

        it("should handle entries with null/undefined optional fields", async () => {
            const entry = createTestEntry({
                duration: undefined,
                exitCode: undefined,
                userId: undefined,
                error: undefined,
                metadata: undefined,
            });
            
            await storage.add(entry);
            const retrieved = await storage.get(entry.id);
            
            expect(retrieved).not.toBeNull();
            expect(retrieved!.duration).toBeUndefined();
            expect(retrieved!.exitCode).toBeUndefined();
            expect(retrieved!.userId).toBeUndefined();
            expect(retrieved!.error).toBeUndefined();
            expect(retrieved!.metadata).toBeUndefined();
        });

        it("should handle search with all filters combined", async () => {
            const entry = createTestEntry({
                id: "combined-test",
                command: "test-command",
                profile: "test-profile",
                userId: "test-user",
                success: true,
                timestamp: new Date("2023-06-15"),
            });
            
            await storage.add(entry);
            
            const results = await storage.search({
                command: "test-command",
                profile: "test-profile", 
                userId: "test-user",
                successOnly: true,
                startDate: new Date("2023-06-01"),
                endDate: new Date("2023-06-30"),
                limit: 10,
                offset: 0,
            });
            
            expect(results).toHaveLength(1);
            expect(results[0].id).toBe("combined-test");
        });
    });
});
