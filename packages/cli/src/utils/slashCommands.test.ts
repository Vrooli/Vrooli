// AI_CHECK: TEST_COVERAGE=42 | LAST: 2025-07-13
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { SlashCommandParser } from "./slashCommands.js";
import { type ConfigManager } from "./config.js";
import { type ContextManager } from "./contextManager.js";
import { ConversationExporter } from "./conversationExporter.js";
import chalk from "chalk";
import type { ChatMessage, User } from "@vrooli/shared";

// Mock dependencies
const mockExportToFile = vi.fn().mockResolvedValue(undefined);
vi.mock("./conversationExporter.js", () => ({
    ConversationExporter: Object.assign(
        vi.fn().mockImplementation(() => ({
            exportToFile: mockExportToFile,
        })),
        {
            isValidFormat: vi.fn().mockReturnValue(true),
            getAvailableFormats: vi.fn().mockReturnValue(["json", "markdown", "txt"]),
        },
    ),
}));

vi.mock("chalk", () => ({
    default: {
        bold: vi.fn((text: string) => text),
        cyan: vi.fn((text: string) => text),
        gray: vi.fn((text: string) => text),
        yellow: vi.fn((text: string) => text),
        green: vi.fn((text: string) => text),
        red: vi.fn((text: string) => text),
    },
}));

describe("SlashCommandParser", () => {
    let parser: SlashCommandParser;
    let mockConfig: ConfigManager;
    let mockContextManager: ContextManager;
    let mockMessages: ChatMessage[];
    let mockParticipants: User[];

    beforeEach(() => {
        // Setup mocks
        mockConfig = {
            getActiveProfileName: vi.fn().mockReturnValue("default"),
            getServerUrl: vi.fn().mockReturnValue("http://localhost:5329"),
            getAuthToken: vi.fn().mockReturnValue("test-token"),
            isJsonOutput: vi.fn().mockReturnValue(false),
            isDebug: vi.fn().mockReturnValue(false),
        } as unknown as ConfigManager;

        mockContextManager = {
            addFile: vi.fn(),
            addUrl: vi.fn(),
            addRoutine: vi.fn(),
            removeContext: vi.fn(),
            clearAll: vi.fn(),
            displayContextSummary: vi.fn(),
            getAllContexts: vi.fn().mockReturnValue(new Map([
                ["file1", { path: "/test.txt", size: 1024, type: "text" }],
                ["url1", { url: "https://example.com", size: 1024, type: "webpage" }],
            ])),
            getStats: vi.fn().mockReturnValue({
                totalContexts: 2,
                maxContexts: 100,
                totalSize: 2048,
                byType: { file: 1, url: 1 },
            }),
        } as unknown as ContextManager;

        mockMessages = [
            {
                id: "msg1",
                text: "Hello",
                createdAt: new Date().toISOString(),
                user: { id: "user1", name: "User", isBot: false } as User,
            } as ChatMessage,
            {
                id: "msg2", 
                text: "Hi there!",
                createdAt: new Date().toISOString(),
                user: { id: "bot1", name: "Bot", isBot: true } as User,
            } as ChatMessage,
        ];

        mockParticipants = [
            { id: "user1", name: "User", isBot: false } as User,
            { id: "bot1", name: "Bot", isBot: true } as User,
        ];

        parser = new SlashCommandParser(
            mockConfig,
            "chat123",
            mockContextManager,
            () => mockMessages,
            () => mockParticipants,
        );
    });

    afterEach(() => {
        vi.clearAllMocks();
        mockExportToFile.mockClear();
    });

    describe("parseCommand", () => {
        it("should parse a simple command", () => {
            const result = parser.parseCommand("/help");

            expect(result).toEqual({
                name: "help",
                args: [],
                rawInput: "/help",
            });
        });

        it("should parse command with arguments", () => {
            const result = parser.parseCommand("/context file /path/to/file.txt");

            expect(result).toEqual({
                name: "context",
                args: ["file", "/path/to/file.txt"],
                rawInput: "/context file /path/to/file.txt",
            });
        });

        it("should handle multiple spaces", () => {
            const result = parser.parseCommand("/history   10   ");

            expect(result).toEqual({
                name: "history",
                args: ["10"],
                rawInput: "/history   10",
            });
        });

        it("should convert command name to lowercase", () => {
            const result = parser.parseCommand("/HELP");

            expect(result?.name).toBe("help");
        });

        it("should return null for non-command input", () => {
            expect(parser.parseCommand("hello")).toBeNull();
            expect(parser.parseCommand("")).toBeNull();
            expect(parser.parseCommand("  ")).toBeNull();
        });

        it("should return null for just a slash", () => {
            expect(parser.parseCommand("/")).toBeNull();
            expect(parser.parseCommand("/ ")).toBeNull();
        });
    });

    describe("executeCommand", () => {
        describe("help command", () => {
            it("should handle /help", async () => {
                const command = { name: "help", args: [], rawInput: "/help" };
                const result = await parser.executeCommand(command);

                expect(result.message).toContain("Available Slash Commands");
                expect(result.message).toContain("/help");
                expect(result.message).toContain("/exit");
                expect(result.message).toContain("/context");
                expect(result.error).toBeUndefined();
            });

            it("should handle /? alias", async () => {
                const command = { name: "?", args: [], rawInput: "/?" };
                const result = await parser.executeCommand(command);

                expect(result.message).toContain("Available Slash Commands");
            });
        });

        describe("exit command", () => {
            it("should handle /exit", async () => {
                const command = { name: "exit", args: [], rawInput: "/exit" };
                const result = await parser.executeCommand(command);

                expect(result.shouldExit).toBe(true);
                expect(result.message).toContain("Exiting interactive chat mode");
            });

            it("should handle /quit alias", async () => {
                const command = { name: "quit", args: [], rawInput: "/quit" };
                const result = await parser.executeCommand(command);

                expect(result.shouldExit).toBe(true);
            });

            it("should handle /q alias", async () => {
                const command = { name: "q", args: [], rawInput: "/q" };
                const result = await parser.executeCommand(command);

                expect(result.shouldExit).toBe(true);
            });
        });

        describe("clear command", () => {
            it("should handle /clear", async () => {
                const command = { name: "clear", args: [], rawInput: "/clear" };
                const result = await parser.executeCommand(command);

                expect(result.shouldClear).toBe(true);
                expect(result.message).toContain("Chat display cleared");
            });

            it("should handle /cls alias", async () => {
                const command = { name: "cls", args: [], rawInput: "/cls" };
                const result = await parser.executeCommand(command);

                expect(result.shouldClear).toBe(true);
            });
        });

        describe("history command", () => {
            it("should handle /history without args", async () => {
                const command = { name: "history", args: [], rawInput: "/history" };
                const result = await parser.executeCommand(command);

                expect(result.message).toContain("Showing last");
                expect(result.message).toContain("messages");
            });

            it("should handle /history with count", async () => {
                const command = { name: "history", args: ["5"], rawInput: "/history 5" };
                const result = await parser.executeCommand(command);

                expect(result.message).toContain("Showing last 5 messages");
            });

            it("should handle invalid count", async () => {
                const command = { name: "history", args: ["abc"], rawInput: "/history abc" };
                const result = await parser.executeCommand(command);

                expect(result.error).toContain("Invalid number");
            });

            it("should handle negative count", async () => {
                const command = { name: "history", args: ["-5"], rawInput: "/history -5" };
                const result = await parser.executeCommand(command);

                expect(result.error).toContain("Invalid number");
            });
        });

        describe("status command", () => {
            it("should show status information", async () => {
                const command = { name: "status", args: [], rawInput: "/status" };
                const result = await parser.executeCommand(command);

                expect(result.message).toContain("Chat Status");
                expect(result.message).toContain("chat123");
                expect(result.message).toContain("default");
                expect(result.message).toContain("http://localhost:5329");
                expect(result.message).toContain("Connected");
            });

            it("should show not authenticated when no token", async () => {
                (mockConfig.getAuthToken as any).mockReturnValue(null);

                const command = { name: "status", args: [], rawInput: "/status" };
                const result = await parser.executeCommand(command);

                expect(result.message).toContain("Not authenticated");
            });
        });

        describe("context command", () => {
            it("should require arguments", async () => {
                const command = { name: "context", args: [], rawInput: "/context" };
                const result = await parser.executeCommand(command);

                expect(result.error).toContain("Context command requires arguments");
            });

            it("should handle context file", async () => {
                (mockContextManager.addFile as any).mockResolvedValue("file1");

                const command = { name: "context", args: ["file", "/test.txt"], rawInput: "/context file /test.txt" };
                const result = await parser.executeCommand(command);

                expect(mockContextManager.addFile).toHaveBeenCalledWith("/test.txt", undefined);
                expect(result.message).toContain("Added file context: file1");
            });

            it("should handle context file with alias", async () => {
                (mockContextManager.addFile as any).mockResolvedValue("myfile");

                const command = { name: "context", args: ["file", "/test.txt", "myfile"], rawInput: "/context file /test.txt myfile" };
                const result = await parser.executeCommand(command);

                expect(mockContextManager.addFile).toHaveBeenCalledWith("/test.txt", "myfile");
            });

            it("should handle context url", async () => {
                (mockContextManager.addUrl as any).mockResolvedValue("webpage");

                const command = { name: "context", args: ["url", "https://example.com"], rawInput: "/context url https://example.com" };
                const result = await parser.executeCommand(command);

                expect(mockContextManager.addUrl).toHaveBeenCalledWith("https://example.com", undefined);
                expect(result.message).toContain("Added URL context: webpage");
            });

            it("should handle context list", async () => {
                const command = { name: "context", args: ["list"], rawInput: "/context list" };
                const result = await parser.executeCommand(command);

                expect(mockContextManager.getAllContexts).toHaveBeenCalled();
                expect(result.message).toContain("Context Items");
                expect(result.message).toContain("Total: 2/100");
            });

            it("should handle context remove", async () => {
                (mockContextManager.removeContext as any).mockReturnValue(true);

                const command = { name: "context", args: ["remove", "file1"], rawInput: "/context remove file1" };
                const result = await parser.executeCommand(command);

                expect(mockContextManager.removeContext).toHaveBeenCalledWith("file1");
                expect(result.message).toContain("Removed context: file1");
            });

            it("should handle context remove failure", async () => {
                (mockContextManager.removeContext as any).mockReturnValue(false);

                const command = { name: "context", args: ["remove", "nonexistent"], rawInput: "/context remove nonexistent" };
                const result = await parser.executeCommand(command);

                expect(result.error).toContain("Context not found: nonexistent");
            });

            it("should handle context clear", async () => {
                const command = { name: "context", args: ["clear"], rawInput: "/context clear" };
                const result = await parser.executeCommand(command);

                expect(mockContextManager.clearAll).toHaveBeenCalled();
                expect(result.message).toContain("✓ Cleared all context (2 items removed)");
            });

            it("should handle unknown context subcommand", async () => {
                const command = { name: "context", args: ["unknown"], rawInput: "/context unknown" };
                const result = await parser.executeCommand(command);

                expect(result.error).toContain("Unknown context subcommand");
            });

            it("should handle /ctx alias", async () => {
                const command = { name: "ctx", args: ["list"], rawInput: "/ctx list" };
                const result = await parser.executeCommand(command);

                expect(mockContextManager.getAllContexts).toHaveBeenCalled();
            });

            it("should handle context errors", async () => {
                (mockContextManager.addFile as any).mockRejectedValue(new Error("File not found"));

                const command = { name: "context", args: ["file", "/missing.txt"], rawInput: "/context file /missing.txt" };
                const result = await parser.executeCommand(command);

                expect(result.error).toContain("Failed to add file: File not found");
            });
        });

        describe("save command", () => {
            it("should handle save with filename", async () => {
                const command = { name: "save", args: ["chat.md"], rawInput: "/save chat.md" };
                const result = await parser.executeCommand(command);

                expect(ConversationExporter).toHaveBeenCalled();
                expect(mockExportToFile).toHaveBeenCalledWith(
                    expect.objectContaining({
                        chatId: "chat123",
                        messages: expect.arrayContaining([
                            expect.objectContaining({ id: "msg1" }),
                            expect.objectContaining({ id: "msg2" }),
                        ]),
                        participants: expect.arrayContaining([
                            expect.objectContaining({ id: "user1" }),
                            expect.objectContaining({ id: "bot1" }),
                        ]),
                    }),
                    "chat.md",
                    expect.any(Object),
                );
                expect(result.message).toContain("Conversation exported");
            });

            it("should handle save with format", async () => {
                const command = { name: "save", args: ["chat.txt", "json"], rawInput: "/save chat.txt json" };
                const result = await parser.executeCommand(command);

                // Check if there's an error first
                expect(result.error).toBeUndefined();
                expect(result.message).toBeDefined();
                
                expect(ConversationExporter).toHaveBeenCalled();
                expect(mockExportToFile).toHaveBeenCalledWith(
                    expect.any(Object),
                    "chat.txt",
                    expect.objectContaining({ format: "json" }),
                );
            });

            it("should require filename", async () => {
                const command = { name: "save", args: [], rawInput: "/save" };
                const result = await parser.executeCommand(command);

                expect(result.error).toContain("Save command requires a filename");
            });

            it("should handle save errors", async () => {
                const mockExporter = {
                    export: vi.fn().mockRejectedValue(new Error("Permission denied")),
                };
                (ConversationExporter as any).mockImplementation(() => mockExporter);

                const command = { name: "save", args: ["chat.md"], rawInput: "/save chat.md" };
                const result = await parser.executeCommand(command);

                expect(result.error).toContain("Failed to export conversation");
            });
        });

        describe("bots command", () => {
            it("should show future feature message", async () => {
                const command = { name: "bots", args: [], rawInput: "/bots" };
                const result = await parser.executeCommand(command);

                expect(result.message).toContain("not yet implemented");
            });
        });

        describe("settings command", () => {
            it("should show current settings", async () => {
                const command = { name: "settings", args: [], rawInput: "/settings" };
                const result = await parser.executeCommand(command);

                expect(result.message).toContain("Current Settings:");
                expect(result.message).toContain("Output Format:");
                expect(result.message).toContain("Debug Mode:");
                expect(result.message).toContain("Profile:");
                expect(result.message).toContain("Setting modification is not yet implemented");
            });
        });

        describe("unknown command", () => {
            it("should handle unknown commands", async () => {
                const command = { name: "unknown", args: [], rawInput: "/unknown" };
                const result = await parser.executeCommand(command);

                expect(result.error).toContain("Unknown command: unknown");
                expect(result.error).toContain("Type /help");
            });
        });
    });

    describe("edge cases", () => {
        it("should handle parser without context manager", async () => {
            const parserNoContext = new SlashCommandParser(mockConfig, "chat123");
            
            const command = { name: "context", args: ["list"], rawInput: "/context list" };
            const result = await parserNoContext.executeCommand(command);

            expect(result.error).toContain("Context management is not available");
        });

        it("should handle parser without message getter", async () => {
            const parserNoMessages = new SlashCommandParser(
                mockConfig,
                "chat123",
                mockContextManager,
            );

            const command = { name: "save", args: ["chat.md"], rawInput: "/save chat.md" };
            const result = await parserNoMessages.executeCommand(command);

            expect(result.error).toContain("Conversation export is not available in this session");
            expect(ConversationExporter).not.toHaveBeenCalled();
        });
    });

    describe("utility methods", () => {
        it("should get available commands for autocomplete", () => {
            const parser = new SlashCommandParser(
                mockConfig,
                "chat123",
                mockContextManager,
                mockMessages,
                mockParticipants,
            );

            const commands = parser.getAvailableCommands();
            
            expect(Array.isArray(commands)).toBe(true);
            expect(commands).toContain("help");
            expect(commands).toContain("exit");
            expect(commands).toContain("clear");
            expect(commands).toContain("history");
            expect(commands).toContain("status");
            expect(commands).toContain("context");
            expect(commands).toContain("save");
            expect(commands).toContain("bots");
            expect(commands).toContain("settings");
        });
    });
});
