import { describe, it, expect, vi, beforeEach, afterEach, type Mock } from "vitest";
import chalk from "chalk";
import inquirer from "inquirer";
import { HistoryUI } from "./HistoryUI.js";
import { HistoryManager } from "./HistoryManager.js";
import { ConfigManager } from "../utils/config.js";
import type { HistoryEntry, HistorySearchQuery } from "./types.js";

// Mock dependencies
vi.mock("chalk", () => ({
    default: {
        bold: vi.fn((text: string) => `[bold]${text}[/bold]`),
        red: vi.fn((text: string) => `[red]${text}[/red]`),
        green: vi.fn((text: string) => `[green]${text}[/green]`),
        yellow: vi.fn((text: string) => `[yellow]${text}[/yellow]`),
        cyan: vi.fn((text: string) => `[cyan]${text}[/cyan]`),
        gray: vi.fn((text: string) => `[gray]${text}[/gray]`),
    },
}));

vi.mock("inquirer", () => ({
    default: {
        prompt: vi.fn(),
        Separator: vi.fn(() => ({ type: "separator" })),
    },
}));

vi.mock("./HistoryManager.js");
vi.mock("../utils/config.js");

describe("HistoryUI", () => {
    let historyUI: HistoryUI;
    let mockHistoryManager: jest.Mocked<HistoryManager>;
    let mockConfigManager: jest.Mocked<ConfigManager>;
    let consoleLogSpy: Mock;
    let consoleClearSpy: Mock;
    let consoleErrorSpy: Mock;

    const mockHistoryEntries: HistoryEntry[] = [
        {
            id: "1",
            command: "test",
            args: ["arg1"],
            options: {},
            profile: "default",
            timestamp: new Date("2025-01-01T10:00:00Z"),
            success: true,
            duration: 1000,
        },
        {
            id: "2", 
            command: "failed",
            args: ["arg2"],
            options: {},
            profile: "dev",
            timestamp: new Date("2025-01-01T11:00:00Z"),
            success: false,
            duration: 2000,
            error: "Test error message",
        },
    ];

    beforeEach(() => {
        // Setup console spies
        consoleLogSpy = vi.spyOn(console, "log").mockImplementation(() => {});
        consoleClearSpy = vi.spyOn(console, "clear").mockImplementation(() => {});
        consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});

        // Mock HistoryManager
        mockHistoryManager = {
            search: vi.fn().mockResolvedValue(mockHistoryEntries),
            getStats: vi.fn(),
            exportToJson: vi.fn(),
            exportToCsv: vi.fn(),
            exportToScript: vi.fn(),
            deleteEntry: vi.fn(),
        } as any;

        // Mock ConfigManager
        mockConfigManager = {
            getShellConfig: vi.fn().mockReturnValue({ profile: "default" }),
        } as any;

        historyUI = new HistoryUI(mockHistoryManager, mockConfigManager);

        // Reset inquirer mock
        (inquirer.prompt as Mock).mockReset();
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("constructor", () => {
        it("should initialize with default state", () => {
            expect(historyUI).toBeInstanceOf(HistoryUI);
        });
    });

    describe("browse", () => {
        it("should start interactive browser and display welcome message", async () => {
            // Mock the main menu to exit immediately
            (inquirer.prompt as Mock).mockResolvedValueOnce({ action: "exit" });

            await historyUI.browse();

            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("📚 Interactive History Browser"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Navigate through your command history"));
            expect(mockHistoryManager.search).toHaveBeenCalled();
        });

        it("should handle search errors gracefully", async () => {
            mockHistoryManager.search.mockRejectedValueOnce(new Error("Search failed"));

            await expect(historyUI.browse()).rejects.toThrow("Search failed");
            expect(consoleErrorSpy).toHaveBeenCalledWith(expect.stringContaining("Failed to load history entries"));
        });
    });

    describe("refreshEntries", () => {
        it("should load entries with default query", async () => {
            // Access private method for testing
            const refreshEntries = (historyUI as any).refreshEntries.bind(historyUI);
            await refreshEntries();

            expect(mockHistoryManager.search).toHaveBeenCalledWith({
                limit: 10,
                offset: 0,
            });
        });

        it("should apply search filters to query", async () => {
            // Set state filters
            const state = (historyUI as any).state;
            state.searchQuery = "test search";
            state.filterCommand = "test";
            state.filterProfile = "dev";
            state.showSuccessOnly = true;

            const refreshEntries = (historyUI as any).refreshEntries.bind(historyUI);
            await refreshEntries();

            expect(mockHistoryManager.search).toHaveBeenCalledWith({
                limit: 10,
                offset: 0,
                text: "test search",
                command: "test",
                profile: "dev",
                successOnly: true,
            });
        });

        it("should apply failed only filter", async () => {
            const state = (historyUI as any).state;
            state.showFailedOnly = true;

            const refreshEntries = (historyUI as any).refreshEntries.bind(historyUI);
            await refreshEntries();

            expect(mockHistoryManager.search).toHaveBeenCalledWith(
                expect.objectContaining({
                    failedOnly: true,
                })
            );
        });

        it("should calculate total entries for pagination", async () => {
            const refreshEntries = (historyUI as any).refreshEntries.bind(historyUI);
            await refreshEntries();

            // Should call search twice - once for entries, once for total count
            expect(mockHistoryManager.search).toHaveBeenCalledTimes(2);
        });
    });

    describe("displayHeader", () => {
        it("should display basic header with no filters", () => {
            const displayHeader = (historyUI as any).displayHeader.bind(historyUI);
            displayHeader();

            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("📚 History Browser"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Total entries: 0"));
        });

        it("should display active filters", () => {
            const state = (historyUI as any).state;
            state.searchQuery = "test";
            state.filterCommand = "command";
            state.showSuccessOnly = true;
            state.totalEntries = 5;

            const displayHeader = (historyUI as any).displayHeader.bind(historyUI);
            displayHeader();

            expect(consoleLogSpy).toHaveBeenCalledWith(
                expect.stringContaining("Active filters: Search: \"test\", Command: command, Success only")
            );
        });
    });

    describe("displayEntries", () => {
        it("should display message when no entries found", () => {
            const state = (historyUI as any).state;
            state.entries = [];

            const displayEntries = (historyUI as any).displayEntries.bind(historyUI);
            displayEntries();

            expect(consoleLogSpy).toHaveBeenCalledWith(
                expect.stringContaining("No history entries found matching your criteria")
            );
        });

        it("should display entries with proper formatting", () => {
            const state = (historyUI as any).state;
            state.entries = mockHistoryEntries;

            const displayEntries = (historyUI as any).displayEntries.bind(historyUI);
            displayEntries();

            // Should display each entry
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("test arg1"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("failed arg2"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Error: Test error message"));
        });

        it("should handle entries without duration", () => {
            const entryWithoutDuration = {
                ...mockHistoryEntries[0],
                duration: undefined,
            };
            const state = (historyUI as any).state;
            state.entries = [entryWithoutDuration];

            const displayEntries = (historyUI as any).displayEntries.bind(historyUI);
            displayEntries();

            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("N/A"));
        });
    });

    describe("displayPagination", () => {
        it("should not display pagination for single page", () => {
            const state = (historyUI as any).state;
            state.totalEntries = 5;
            state.pageSize = 10;

            const displayPagination = (historyUI as any).displayPagination.bind(historyUI);
            displayPagination();

            // Should not display page info for single page
            expect(consoleLogSpy).not.toHaveBeenCalledWith(expect.stringContaining("Page"));
        });

        it("should display pagination for multiple pages", () => {
            const state = (historyUI as any).state;
            state.totalEntries = 25;
            state.pageSize = 10;
            state.currentPage = 1;

            const displayPagination = (historyUI as any).displayPagination.bind(historyUI);
            displayPagination();

            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Page 2 of 3"));
        });
    });

    describe("promptForAction", () => {
        it("should return selected action", async () => {
            (inquirer.prompt as Mock).mockResolvedValueOnce({ action: "search" });

            const promptForAction = (historyUI as any).promptForAction.bind(historyUI);
            const result = await promptForAction();

            expect(result).toBe("search");
            expect(inquirer.prompt).toHaveBeenCalled();
        });

        it("should filter out unavailable actions", async () => {
            const state = (historyUI as any).state;
            state.currentPage = 0; // First page
            state.totalEntries = 5;
            state.pageSize = 10; // Single page
            state.showSuccessOnly = true;

            (inquirer.prompt as Mock).mockResolvedValueOnce({ action: "exit" });

            const promptForAction = (historyUI as any).promptForAction.bind(historyUI);
            await promptForAction();

            const promptCall = (inquirer.prompt as Mock).mock.calls[0][0][0];
            const choiceValues = promptCall.choices
                .filter((choice: any) => choice && typeof choice === "object" && "value" in choice)
                .map((choice: any) => choice.value);

            // Should not have prev_page (on first page) or next_page (single page) or filter_success (already active)
            expect(choiceValues).not.toContain("prev_page");
            expect(choiceValues).not.toContain("next_page");
            expect(choiceValues).not.toContain("filter_success");
        });
    });

    describe("handleAction", () => {
        it("should handle search action", async () => {
            (inquirer.prompt as Mock).mockResolvedValueOnce({ query: "test" });

            const handleAction = (historyUI as any).handleAction.bind(historyUI);
            await handleAction("search");

            expect(inquirer.prompt).toHaveBeenCalledWith(
                expect.arrayContaining([
                    expect.objectContaining({
                        name: "query",
                        message: expect.stringContaining("Enter search query"),
                    })
                ])
            );
        });

        it("should handle filter_success action", async () => {
            const state = (historyUI as any).state;
            
            const handleAction = (historyUI as any).handleAction.bind(historyUI);
            await handleAction("filter_success");

            expect(state.showSuccessOnly).toBe(true);
            expect(state.showFailedOnly).toBe(false);
            expect(state.currentPage).toBe(0);
            expect(mockHistoryManager.search).toHaveBeenCalled();
        });

        it("should handle filter_failed action", async () => {
            const state = (historyUI as any).state;
            
            const handleAction = (historyUI as any).handleAction.bind(historyUI);
            await handleAction("filter_failed");

            expect(state.showFailedOnly).toBe(true);
            expect(state.showSuccessOnly).toBe(false);
            expect(state.currentPage).toBe(0);
            expect(mockHistoryManager.search).toHaveBeenCalled();
        });

        it("should handle clear_filters action", async () => {
            const state = (historyUI as any).state;
            state.searchQuery = "test";
            state.filterCommand = "command";
            state.showSuccessOnly = true;
            
            const handleAction = (historyUI as any).handleAction.bind(historyUI);
            await handleAction("clear_filters");

            expect(state.searchQuery).toBeUndefined();
            expect(state.filterCommand).toBeUndefined();
            expect(state.showSuccessOnly).toBe(false);
            expect(mockHistoryManager.search).toHaveBeenCalled();
        });

        it("should handle pagination actions", async () => {
            const state = (historyUI as any).state;
            state.currentPage = 1;
            
            const handleAction = (historyUI as any).handleAction.bind(historyUI);
            
            // Test prev_page
            await handleAction("prev_page");
            expect(state.currentPage).toBe(0);
            
            // Test next_page
            await handleAction("next_page");
            expect(state.currentPage).toBe(1);
        });
    });

    describe("hasActiveFilters", () => {
        it("should return false when no filters are active", () => {
            const hasActiveFilters = (historyUI as any).hasActiveFilters.bind(historyUI);
            expect(hasActiveFilters()).toBe(false);
        });

        it("should return true when filters are active", () => {
            const state = (historyUI as any).state;
            state.searchQuery = "test";

            const hasActiveFilters = (historyUI as any).hasActiveFilters.bind(historyUI);
            expect(hasActiveFilters()).toBe(true);
        });

        it("should detect various filter types", () => {
            const state = (historyUI as any).state;
            const hasActiveFilters = (historyUI as any).hasActiveFilters.bind(historyUI);

            // Test each filter type
            state.filterCommand = "test";
            expect(hasActiveFilters()).toBe(true);

            state.filterCommand = undefined;
            state.filterProfile = "dev";
            expect(hasActiveFilters()).toBe(true);

            state.filterProfile = undefined;
            state.showSuccessOnly = true;
            expect(hasActiveFilters()).toBe(true);

            state.showSuccessOnly = false;
            state.showFailedOnly = true;
            expect(hasActiveFilters()).toBe(true);
        });
    });

    describe("clearFilters", () => {
        it("should reset all filters and pagination", () => {
            const state = (historyUI as any).state;
            state.searchQuery = "test";
            state.filterCommand = "command";
            state.filterProfile = "profile";
            state.showSuccessOnly = true;
            state.showFailedOnly = true;
            state.currentPage = 2;

            const clearFilters = (historyUI as any).clearFilters.bind(historyUI);
            clearFilters();

            expect(state.searchQuery).toBeUndefined();
            expect(state.filterCommand).toBeUndefined();
            expect(state.filterProfile).toBeUndefined();
            expect(state.showSuccessOnly).toBe(false);
            expect(state.showFailedOnly).toBe(false);
            expect(state.currentPage).toBe(0);
        });
    });

    describe("selectEntry", () => {
        it("should return null when no entries available", async () => {
            const state = (historyUI as any).state;
            state.entries = [];

            const selectEntry = (historyUI as any).selectEntry.bind(historyUI);
            const result = await selectEntry("Select an entry");

            expect(result).toBeNull();
            expect(consoleLogSpy).toHaveBeenCalledWith(
                expect.stringContaining("No entries available")
            );
        });

        it("should allow entry selection", async () => {
            const state = (historyUI as any).state;
            state.entries = mockHistoryEntries;

            (inquirer.prompt as Mock).mockResolvedValueOnce({ selectedEntry: mockHistoryEntries[0] });

            const selectEntry = (historyUI as any).selectEntry.bind(historyUI);
            const result = await selectEntry("Select an entry");

            expect(result).toBe(mockHistoryEntries[0]);
            expect(inquirer.prompt).toHaveBeenCalledWith(
                expect.arrayContaining([
                    expect.objectContaining({
                        type: "list",
                        name: "selectedEntry",
                        message: "Select an entry",
                    })
                ])
            );
        });
    });

    describe("waitForKeypress", () => {
        it("should wait for user input", async () => {
            (inquirer.prompt as Mock).mockResolvedValueOnce({ continue: true });

            const waitForKeypress = (historyUI as any).waitForKeypress.bind(historyUI);
            await waitForKeypress();

            expect(inquirer.prompt).toHaveBeenCalledWith(
                expect.arrayContaining([
                    expect.objectContaining({
                        type: "input",
                        message: expect.stringContaining("Press Enter to continue"),
                    })
                ])
            );
        });
    });

    describe("error handling", () => {
        it("should handle history manager errors gracefully", async () => {
            mockHistoryManager.search.mockRejectedValueOnce(new Error("Database error"));

            const refreshEntries = (historyUI as any).refreshEntries.bind(historyUI);
            
            await expect(refreshEntries()).rejects.toThrow("Database error");
            expect(consoleErrorSpy).toHaveBeenCalledWith(
                expect.stringContaining("Failed to load history entries")
            );
        });

        it("should handle inquirer prompt errors", async () => {
            (inquirer.prompt as Mock).mockRejectedValueOnce(new Error("Prompt error"));

            const promptForAction = (historyUI as any).promptForAction.bind(historyUI);
            
            await expect(promptForAction()).rejects.toThrow("Prompt error");
        });
    });

    describe("integration scenarios", () => {
        it("should handle complete user workflow", async () => {
            // Mock a sequence of user interactions
            const promptMock = inquirer.prompt as Mock;
            
            // First interaction: search
            promptMock.mockResolvedValueOnce({ action: "search" });
            promptMock.mockResolvedValueOnce({ searchTerm: "test" });
            
            // Second interaction: exit
            promptMock.mockResolvedValueOnce({ action: "exit" });

            await historyUI.browse();

            expect(mockHistoryManager.search).toHaveBeenCalled();
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("👋 Goodbye!"));
        });

        it("should handle pagination workflow", async () => {
            const state = (historyUI as any).state;
            state.totalEntries = 25;
            state.pageSize = 10;

            const promptMock = inquirer.prompt as Mock;
            
            // Navigate to next page, then exit
            promptMock.mockResolvedValueOnce({ action: "next_page" });
            promptMock.mockResolvedValueOnce({ action: "exit" });

            await historyUI.browse();

            expect(state.currentPage).toBe(1);
            expect(mockHistoryManager.search).toHaveBeenCalledWith(
                expect.objectContaining({
                    offset: 10, // Second page
                })
            );
        });
    });
});