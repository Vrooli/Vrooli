import { generatePK } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ConfigManager } from "../utils/config.js";
import { HistoryManager } from "./HistoryManager.js";
import type { HistoryEntry, HistorySearchQuery, HistoryStats } from "./types.js";

// Mock storage
const mockStorage = {
    add: vi.fn(),
    get: vi.fn(),
    search: vi.fn(),
    getStats: vi.fn(),
    clear: vi.fn(),
    delete: vi.fn(),
    export: vi.fn(),
};

vi.mock("./storage/SqliteStorage.js", () => ({
    SqliteStorage: vi.fn(() => mockStorage),
}));

vi.mock("@vrooli/shared", async () => {
    const actual = await vi.importActual("@vrooli/shared");
    return {
        ...actual,
        generatePK: vi.fn(() => ({ toString: () => "mock-id-123" })),
    };
});

describe("HistoryManager", () => {
    let historyManager: HistoryManager;
    let mockConfig: ConfigManager;

    beforeEach(() => {
        vi.clearAllMocks();

        mockConfig = {
            getActiveProfileName: vi.fn().mockReturnValue("default"),
            getActiveProfile: vi.fn().mockReturnValue({ userId: "user123" }),
        } as unknown as ConfigManager;

        historyManager = new HistoryManager(mockConfig);
    });

    afterEach(() => {
        vi.clearAllMocks();
    });


    describe("startCommand", () => {
        it("should start a new command entry", async () => {
            const command = "routine";
            const args = ["list", "--format", "json"];
            const options = { format: "json", limit: 10 };

            await historyManager.startCommand(command, args, options);

            expect(vi.mocked(generatePK)).toHaveBeenCalled();
            expect(mockConfig.getActiveProfileName).toHaveBeenCalled();
            expect(mockConfig.getActiveProfile).toHaveBeenCalled();
        });

        it("should skip history when disabled", async () => {
            const options = { noHistory: true };

            await historyManager.startCommand("routine", ["list"], options);

            // Should not generate PK or call config methods when history is disabled
            expect(vi.mocked(generatePK)).not.toHaveBeenCalled();
        });

        it("should sanitize sensitive options", async () => {
            const command = "auth";
            const args = ["login"];
            const options = {
                email: "user@example.com",
                password: "secret123",
                token: "bearer-token",
                apiKey: "api-key-value"
            };

            await historyManager.startCommand(command, args, options);

            expect(vi.mocked(generatePK)).toHaveBeenCalled();
        });
    });

    describe("endCommand", () => {
        beforeEach(async () => {
            await historyManager.startCommand("routine", ["list"], {});
        });

        it("should complete a command entry successfully", async () => {
            await historyManager.endCommand(0);

            expect(mockStorage.add).toHaveBeenCalledWith(
                expect.objectContaining({
                    id: "mock-id-123",
                    command: "routine",
                    args: ["list"],
                    exitCode: 0,
                    success: true,
                    profile: "default",
                    userId: "user123",
                })
            );
        });

        it("should handle command failure", async () => {
            const error = new Error("Command failed");

            await historyManager.endCommand(1, error);

            expect(mockStorage.add).toHaveBeenCalledWith(
                expect.objectContaining({
                    exitCode: 1,
                    success: false,
                    error: "Command failed",
                })
            );
        });

        it("should calculate command duration", async () => {
            // Reset mocks and create fresh history manager
            vi.clearAllMocks();

            // Mock both Date constructor and Date.now
            const mockStartTime = 1000;
            const mockEndTime = 6000;
            let currentMockTime = mockStartTime;

            const OriginalDate = Date;
            global.Date = vi.fn((...args) => {
                if (args.length === 0) {
                    return new OriginalDate(currentMockTime);
                }
                return new OriginalDate(...args);
            }) as any;
            global.Date.now = vi.fn(() => currentMockTime);
            Object.setPrototypeOf(global.Date, OriginalDate);

            const freshHistoryManager = new HistoryManager(mockConfig);

            // Start command
            await freshHistoryManager.startCommand("routine", ["list"], {});

            // Simulate 5 seconds passing
            currentMockTime = mockEndTime;

            await freshHistoryManager.endCommand(0);

            expect(mockStorage.add).toHaveBeenCalledWith(
                expect.objectContaining({
                    duration: 5000,
                })
            );

            // Restore Date
            global.Date = OriginalDate;
        });

        it("should not add entry when no current entry exists", async () => {
            const freshManager = new HistoryManager(mockConfig);

            await freshManager.endCommand(0);

            expect(mockStorage.add).not.toHaveBeenCalled();
        });
    });

    describe("getHistory", () => {
        it("should retrieve history entries", async () => {
            const mockEntries: HistoryEntry[] = [
                {
                    id: "1",
                    command: "routine",
                    args: ["list"],
                    options: {},
                    timestamp: new Date(),
                    duration: 1000,
                    exitCode: 0,
                    success: true,
                    profile: "default",
                    userId: "user123",
                },
            ];

            vi.mocked(mockStorage.search).mockResolvedValue(mockEntries);

            const result = await historyManager.search();

            expect(mockStorage.search).toHaveBeenCalledWith({});
            expect(result).toEqual(mockEntries);
        });

        it("should pass options to storage", async () => {
            const options = { limit: 10, command: "routine" };

            vi.mocked(mockStorage.search).mockResolvedValue([]);

            await historyManager.search(options);

            expect(mockStorage.search).toHaveBeenCalledWith(options);
        });
    });

    describe("searchHistory", () => {
        it("should search history entries", async () => {
            const query: HistorySearchQuery = {
                query: "list routines",
                limit: 5,
            };

            const mockResults: HistoryEntry[] = [];
            vi.mocked(mockStorage.search).mockResolvedValue(mockResults);

            const result = await historyManager.search(query);

            expect(mockStorage.search).toHaveBeenCalledWith(query);
            expect(result).toEqual(mockResults);
        });
    });

    describe("getEntry", () => {
        it("should get specific history entry", async () => {
            const entryId = "123";
            const mockEntry: HistoryEntry = {
                id: entryId,
                command: "routine",
                args: ["list"],
                options: {},
                timestamp: new Date(),
                duration: 1000,
                exitCode: 0,
                success: true,
                profile: "default",
                userId: "user123",
            };

            vi.mocked(mockStorage.get).mockResolvedValue(mockEntry);

            const result = await historyManager.get(entryId);

            expect(mockStorage.get).toHaveBeenCalledWith(entryId);
            expect(result).toEqual(mockEntry);
        });
    });

    describe("deleteEntry", () => {
        it("should delete history entry", async () => {
            const entryId = "123";

            await historyManager.delete(entryId);

            expect(mockStorage.delete).toHaveBeenCalledWith(entryId);
        });
    });

    describe("getStats", () => {
        it("should get history statistics", async () => {
            const mockStats: HistoryStats = {
                totalCommands: 100,
                successfulCommands: 95,
                failedCommands: 5,
                uniqueCommands: 25,
                avgDuration: 2500,
                lastCommand: new Date("2024-01-02"),
                topCommands: [
                    { command: "routine list", count: 30 },
                    { command: "auth status", count: 25 },
                ],
                commandsByProfile: {
                    "default": 80,
                    "production": 20,
                },
                recentActivity: [
                    { date: "2024-01-01", count: 10 },
                    { date: "2024-01-02", count: 15 },
                ],
            };

            vi.mocked(mockStorage.getStats).mockResolvedValue(mockStats);

            const result = await historyManager.getStats();

            expect(mockStorage.getStats).toHaveBeenCalled();
            expect(result).toEqual(mockStats);
        });
    });

    describe("clearHistory", () => {
        it("should clear all history", async () => {
            await historyManager.clear();

            expect(mockStorage.clear).toHaveBeenCalled();
        });

    });


    describe("sensitive data handling", () => {
        it("should identify and sanitize sensitive keys", async () => {
            const sensitiveOptions = {
                password: "secret",
                token: "bearer-token",
                secret: "api-secret",
                key: "api-key",
                auth: "auth-header",
                normalOption: "safe-value",
            };

            await historyManager.startCommand("test", [], sensitiveOptions);
            await historyManager.endCommand(0);

            expect(mockStorage.add).toHaveBeenCalledWith(
                expect.objectContaining({
                    options: expect.objectContaining({
                        password: "***",
                        token: "***",
                        secret: "***",
                        key: "***",
                        auth: "***",
                        normalOption: "safe-value",
                    }),
                })
            );
        });

        it("should handle nested sensitive data", async () => {
            const nestedOptions = {
                config: {
                    password: "secret",
                    username: "user",
                },
                credentials: {
                    token: "bearer-token",
                },
            };

            await historyManager.startCommand("test", [], nestedOptions);
            await historyManager.endCommand(0);

            expect(mockStorage.add).toHaveBeenCalledWith(
                expect.objectContaining({
                    options: expect.objectContaining({
                        config: {
                            password: "***",
                            username: "user",
                        },
                        credentials: {
                            token: "***",
                        },
                    }),
                })
            );
        });
    });

    describe("history disabled detection", () => {
        const disablingOptions = [
            { noHistory: true },
            { "no-history": true },
            { disableHistory: true },
            { "disable-history": true },
        ];

        disablingOptions.forEach((options, index) => {
            it(`should detect history disabled with option set ${index + 1}`, async () => {
                await historyManager.startCommand("test", [], options);

                expect(vi.mocked(generatePK)).not.toHaveBeenCalled();
            });
        });
    });
});