// AI_CHECK: TEST_QUALITY=3 | LAST: 2025-08-04
import { describe, it, expect, vi, beforeEach, afterEach, type Mock } from "vitest";
import { Command } from "commander";
import { HistoryCommands } from "./history.js";
import { HistoryManager } from "../history/HistoryManager.js";
import { HistoryUI } from "../history/HistoryUI.js";
import type { ApiClient } from "../utils/client.js";
import type { ConfigManager } from "../utils/config.js";
import { output } from "../utils/output.js";
import chalk from "chalk";

// Mock dependencies
vi.mock("../history/HistoryManager.js");
vi.mock("../history/HistoryUI.js");
vi.mock("../utils/output.js", () => ({
    output: {
        error: vi.fn(),
        info: vi.fn(),
        success: vi.fn(),
        warning: vi.fn(),
        debug: vi.fn(),
        json: vi.fn(),
        newline: vi.fn(),
    },
}));
vi.mock("chalk", () => ({
    default: {
        green: vi.fn((text: string) => `[green]${text}[/green]`),
        red: vi.fn((text: string) => `[red]${text}[/red]`),
        yellow: vi.fn((text: string) => `[yellow]${text}[/yellow]`),
        blue: vi.fn((text: string) => `[blue]${text}[/blue]`),
        gray: vi.fn((text: string) => `[gray]${text}[/gray]`),
        cyan: vi.fn((text: string) => `[cyan]${text}[/cyan]`),
        bold: vi.fn((text: string) => `[bold]${text}[/bold]`),
        dim: vi.fn((text: string) => `[dim]${text}[/dim]`),
    },
}));
vi.mock("fs", () => ({
    promises: {
        writeFile: vi.fn(),
        mkdir: vi.fn(),
    },
}));

describe("HistoryCommands", () => {
    let mockProgram: Command;
    let mockClient: ApiClient;
    let mockConfig: ConfigManager;
    let mockHistoryManager: any;
    let mockHistoryUI: any;
    let historyCommands: HistoryCommands;
    let processExitSpy: any;

    beforeEach(() => {
        // Reset mocks
        vi.clearAllMocks();
        
        // Mock process.exit to prevent test failures
        processExitSpy = vi.spyOn(process, "exit").mockImplementation(() => undefined as never);

        // Mock Commander program
        mockProgram = new Command();
        mockProgram.exitOverride();
        // Configure the mock program to handle parsing without throwing
        mockProgram.configureOutput({
            writeOut: () => {},
            writeErr: () => {},
        });

        // Mock ApiClient
        mockClient = {} as ApiClient;

        // Mock ConfigManager
        mockConfig = {
            getDataDir: vi.fn().mockReturnValue("/home/user/.vrooli"),
            isJsonOutput: vi.fn().mockReturnValue(false),
        } as any;

        // Mock HistoryManager with proper methods
        mockHistoryManager = {
            search: vi.fn().mockResolvedValue([
                {
                    id: "1",
                    command: "auth login",
                    args: [],
                    timestamp: new Date("2024-01-01T10:00:00Z"),
                    duration: 5000,
                    exitCode: 0,
                    profile: "default",
                    success: true,
                },
                {
                    id: "2",
                    command: "routine list",
                    args: [],
                    timestamp: new Date("2024-01-01T11:00:00Z"),
                    duration: 2000,
                    exitCode: 0,
                    profile: "default",
                    success: true,
                },
            ]),
            get: vi.fn().mockResolvedValue({
                id: "1",
                command: "auth login",
                args: [],
                timestamp: new Date("2024-01-01T10:00:00Z"),
                duration: 5000,
                exitCode: 0,
                profile: "default",
                success: true,
            }),
            getStats: vi.fn().mockResolvedValue({
                totalCommands: 100,
                successfulCommands: 90,
                failedCommands: 10,
                uniqueCommands: 15,
                avgDuration: 3000,
                lastCommand: new Date("2024-01-01T12:00:00Z"),
                topCommands: [
                    { command: "chat start", count: 25 },
                    { command: "routine list", count: 15 },
                ],
                commandsByProfile: {
                    default: 80,
                    staging: 20,
                },
                recentActivity: [
                    { date: "2024-01-01", count: 10 },
                    { date: "2024-01-02", count: 5 },
                ],
            }),
            delete: vi.fn().mockResolvedValue(true),
            clear: vi.fn().mockResolvedValue(undefined),
            getSuggestions: vi.fn().mockResolvedValue([
                {
                    id: "1",
                    command: "auth login",
                    args: [],
                    timestamp: new Date("2024-01-01T10:00:00Z"),
                    duration: 5000,
                    exitCode: 0,
                    profile: "default",
                    success: true,
                },
            ]),
            getFrequentCommands: vi.fn().mockResolvedValue([
                { command: "chat start", count: 25, lastUsed: new Date("2024-01-01T10:00:00Z") },
                { command: "routine list", count: 15, lastUsed: new Date("2024-01-01T11:00:00Z") },
            ]),
        };

        vi.mocked(HistoryManager).mockImplementation(() => mockHistoryManager);

        // Mock HistoryUI
        mockHistoryUI = {
            browse: vi.fn().mockResolvedValue(undefined),
        };

        vi.mocked(HistoryUI).mockImplementation(() => mockHistoryUI);

        // Create instance - this will register commands on mockProgram
        historyCommands = new HistoryCommands(mockProgram, mockClient, mockConfig);
    });

    afterEach(() => {
        processExitSpy.mockRestore();
        vi.clearAllMocks();
    });

    describe("constructor", () => {
        it("should initialize HistoryManager and register commands", () => {
            expect(HistoryManager).toHaveBeenCalledWith(mockConfig);
            
            const historyCommand = mockProgram.commands.find(cmd => cmd.name() === "history");
            expect(historyCommand).toBeDefined();
            expect(historyCommand?.description()).toContain("View and manage command history");
        });
    });

    describe("list command", () => {
        it("should list recent commands with default options", async () => {
            await mockProgram.parseAsync(["node", "test", "history", "list"]);

            expect(mockHistoryManager.search).toHaveBeenCalledWith({
                limit: 50,
                command: undefined,
                successOnly: undefined,
                failedOnly: undefined,
                profile: undefined,
            });
        });

        it("should handle list command with options", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "list",
                "-n", "10",
                "-c", "auth",
                "--success",
                "-p", "staging",
            ]);

            expect(mockHistoryManager.search).toHaveBeenCalledWith({
                limit: 10,
                command: "auth",
                successOnly: true,
                failedOnly: undefined,
                profile: "staging",
            });
        });

        it("should handle failed filter option", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "list",
                "--failed",
            ]);

            expect(mockHistoryManager.search).toHaveBeenCalledWith({
                limit: 50,
                command: undefined,
                successOnly: undefined,
                failedOnly: true,
                profile: undefined,
            });
        });

        it("should handle JSON format output", async () => {
            mockConfig.isJsonOutput = vi.fn().mockReturnValue(true);

            await mockProgram.parseAsync([
                "node", "test", "history", "list",
                "-f", "json",
            ]);

            expect(output.json).toHaveBeenCalledWith(expect.any(Array));
        });

        it("should handle script format output", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "list",
                "-f", "script",
            ]);

            // Script format outputs successful commands via output.info
            expect(output.info).toHaveBeenCalled();
            const calls = (output.info as Mock).mock.calls;
            expect(calls.some((call: any[]) => 
                call[0]?.includes("auth login") || call[0]?.includes("routine list"),
            )).toBeTruthy();
        });

        it("should handle empty history", async () => {
            mockHistoryManager.search.mockResolvedValue([]);

            await mockProgram.parseAsync(["node", "test", "history", "list"]);

            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("No history entries found"));
        });
    });

    describe("search command", () => {
        it("should search history with query", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "search", "login",
            ]);

            expect(mockHistoryManager.search).toHaveBeenCalledWith({
                text: "login",
                limit: 20,
                command: undefined,
                successOnly: undefined,
            });
        });

        it("should handle search with options", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "search", "test",
                "-n", "5",
                "-c", "auth",
                "--success",
            ]);

            expect(mockHistoryManager.search).toHaveBeenCalledWith({
                text: "test",
                limit: 5,
                command: "auth",
                successOnly: true,
            });
        });

        it("should handle no search results", async () => {
            mockHistoryManager.search.mockResolvedValue([]);

            await mockProgram.parseAsync([
                "node", "test", "history", "search", "nonexistent",
            ]);

            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("No history entries found"));
        });
    });

    describe("replay command", () => {
        it("should replay a command by ID", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "replay", "1",
            ]);

            expect(mockHistoryManager.get).toHaveBeenCalledWith("1");
            // Should show what would be executed since replay is not implemented
            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("Command replay not implemented"));
        });

        it("should handle dry-run option", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "replay", "1",
                "--dry-run",
            ]);

            expect(mockHistoryManager.get).toHaveBeenCalledWith("1");
            expect(output.info).toHaveBeenCalledWith(
                expect.stringContaining("Would execute:"),
            );
        });

        it("should handle replay of non-existent entry", async () => {
            mockHistoryManager.get.mockResolvedValue(null);

            await mockProgram.parseAsync([
                "node", "test", "history", "replay", "999",
            ]);

            expect(output.error).toHaveBeenCalledWith(
                expect.stringContaining("Command not found in history"),
            );
            expect(processExitSpy).toHaveBeenCalledWith(1);
        });
    });

    describe("stats command", () => {
        it("should show history statistics", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "stats",
            ]);

            expect(mockHistoryManager.getStats).toHaveBeenCalled();
            expect(output.info).toHaveBeenCalledWith(
                expect.stringContaining("Command History Statistics"),
            );
        });

        it("should handle JSON output for stats", async () => {
            mockConfig.isJsonOutput = vi.fn().mockReturnValue(true);

            await mockProgram.parseAsync([
                "node", "test", "history", "stats",
            ]);

            expect(output.json).toHaveBeenCalledWith(expect.any(Object));
        });
    });

    describe("browse command", () => {
        it("should launch interactive history browser", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "browse",
            ]);

            expect(mockHistoryUI.browse).toHaveBeenCalled();
        });
    });

    describe("alias command", () => {
        it("should create an alias from history entry", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "alias", "mylogin", "1",
            ]);

            expect(mockHistoryManager.get).toHaveBeenCalledWith("1");
            expect(output.success).toHaveBeenCalledWith(
                expect.stringContaining("Would create alias 'mylogin'"),
            );
        });

        it("should handle alias for non-existent entry", async () => {
            mockHistoryManager.get.mockResolvedValue(null);

            await mockProgram.parseAsync([
                "node", "test", "history", "alias", "myalias", "999",
            ]);

            expect(output.error).toHaveBeenCalledWith(
                expect.stringContaining("Command not found in history"),
            );
            expect(processExitSpy).toHaveBeenCalledWith(1);
        });
    });

    describe("export command", () => {
        it("should export history to JSON", async () => {
            const fs = await import("fs");
            const writeFileSpy = vi.spyOn(fs.promises, "writeFile").mockResolvedValue(undefined);

            await mockProgram.parseAsync([
                "node", "test", "history", "export",
                "-o", "/tmp/history.json",
            ]);

            expect(mockHistoryManager.search).toHaveBeenCalledWith({});
            expect(writeFileSpy).toHaveBeenCalledWith(
                "/tmp/history.json",
                expect.any(String),
            );
        });

        it("should export history to CSV", async () => {
            const fs = await import("fs");
            const writeFileSpy = vi.spyOn(fs.promises, "writeFile").mockResolvedValue(undefined);

            await mockProgram.parseAsync([
                "node", "test", "history", "export",
                "-o", "/tmp/history.csv",
                "--format", "csv",
            ]);

            expect(writeFileSpy).toHaveBeenCalledWith(
                "/tmp/history.csv",
                expect.stringContaining("\"id\",\"command\",\"args\""),
            );
        });

        it("should export history as script", async () => {
            const fs = await import("fs");
            const writeFileSpy = vi.spyOn(fs.promises, "writeFile").mockResolvedValue(undefined);

            await mockProgram.parseAsync([
                "node", "test", "history", "export",
                "-o", "/tmp/history.sh",
                "--format", "script",
            ]);

            expect(writeFileSpy).toHaveBeenCalledWith(
                "/tmp/history.sh",
                expect.stringContaining("#!/bin/bash"),
            );
        });

        it("should filter export by date range", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "export",
                "--since", "2024-01-01",
                "--until", "2024-01-31",
            ]);

            expect(mockHistoryManager.search).toHaveBeenCalledWith({
                startDate: expect.any(Date),
                endDate: expect.any(Date),
            });
        });

        it("should filter export by success status", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "export",
                "--success",
            ]);

            expect(mockHistoryManager.search).toHaveBeenCalledWith({
                successOnly: true,
            });
        });

        it("should output to console when no file specified", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "export",
            ]);

            expect(output.info).toHaveBeenCalled();
        });
    });

    describe("clear command", () => {
        it("should clear all history with force flag", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "clear",
                "--force",
            ]);

            expect(mockHistoryManager.clear).toHaveBeenCalled();
            expect(output.success).toHaveBeenCalledWith("History cleared");
        });

        it("should show warning without force flag", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "clear",
            ]);

            expect(mockHistoryManager.clear).not.toHaveBeenCalled();
            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("requires --force flag"));
        });
    });

    describe("delete command", () => {
        it("should delete entry with force flag", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "delete", "1",
                "--force",
            ]);

            expect(mockHistoryManager.delete).toHaveBeenCalledWith("1");
            expect(output.success).toHaveBeenCalledWith("History entry deleted");
        });

        it("should show warning without force flag", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "delete", "1",
            ]);

            expect(mockHistoryManager.delete).not.toHaveBeenCalled();
            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("requires --force flag"));
        });
    });

    describe("info command", () => {
        it("should show database info", async () => {
            // Add getDbInfo to storage mock
            (mockHistoryManager as any).storage = {
                getDbInfo: vi.fn().mockReturnValue({
                    path: "/home/user/.vrooli/history.db",
                    size: 1024000,
                    pageCount: 100,
                }),
            };

            await mockProgram.parseAsync([
                "node", "test", "history", "info",
            ]);

            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("History Database Information"));
        });
    });

    describe("suggest command", () => {
        it("should show command suggestions", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "suggest", "au",
            ]);

            expect(mockHistoryManager.getSuggestions).toHaveBeenCalledWith("au", 5);
            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("Suggestions for \"au\""));
        });

        it("should handle custom count", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "suggest", "ro",
                "-n", "3",
            ]);

            expect(mockHistoryManager.getSuggestions).toHaveBeenCalledWith("ro", 3);
        });
    });

    describe("frequent command", () => {
        it("should show frequent commands", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "frequent",
            ]);

            expect(mockHistoryManager.getFrequentCommands).toHaveBeenCalledWith(10);
            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("Frequently Used Commands"));
        });

        it("should handle custom count", async () => {
            await mockProgram.parseAsync([
                "node", "test", "history", "frequent",
                "-n", "5",
            ]);

            expect(mockHistoryManager.getFrequentCommands).toHaveBeenCalledWith(5);
        });
    });

    describe("error handling", () => {
        it("should handle errors gracefully", async () => {
            mockHistoryManager.search.mockRejectedValue(new Error("Database error"));

            await mockProgram.parseAsync(["node", "test", "history", "list"]);

            expect(output.error).toHaveBeenCalledWith(
                expect.stringContaining("Failed to list history"),
            );
            expect(processExitSpy).toHaveBeenCalledWith(1);
        });
    });
});
