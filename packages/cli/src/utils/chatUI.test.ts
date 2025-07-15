// AI_CHECK: TEST_COVERAGE=10 | LAST: 2025-01-13
import { describe, it, expect, vi, beforeEach, afterEach, type Mock } from "vitest";
import { InteractiveChatUI, type ChatDisplayOptions, type BotStatus } from "./chatUI.js";
import { type ChatMessage, type User } from "@vrooli/shared";

// Mock chalk module
vi.mock("chalk", () => {
    // Helper to create chainable methods
    const createChainableMock = (prefix: string) => {
        const mock: any = vi.fn((text: string) => `[${prefix}]${text}[/${prefix}]`);
        mock.bold = vi.fn((text: string) => `[${prefix}.bold]${text}[/${prefix}.bold]`);
        return mock;
    };

    return {
        default: {
            cyan: createChainableMock("cyan"),
            green: createChainableMock("green"),
            yellow: createChainableMock("yellow"),
            red: createChainableMock("red"),
            gray: createChainableMock("gray"),
            dim: createChainableMock("dim"),
            blue: createChainableMock("blue"),
            white: vi.fn((text: string) => `[white]${text}[/white]`),
            bgGray: {
                white: vi.fn((text: string) => `[bgGray.white]${text}[/bgGray.white]`),
            },
            bold: {
                cyan: vi.fn((text: string) => `[bold.cyan]${text}[/bold.cyan]`),
                green: vi.fn((text: string) => `[bold.green]${text}[/bold.green]`),
                red: vi.fn((text: string) => `[bold.red]${text}[/bold.red]`),
                blue: vi.fn((text: string) => `[bold.blue]${text}[/bold.blue]`),
            },
        },
    };
});

describe("InteractiveChatUI", () => {
    let ui: InteractiveChatUI;
    let consoleLogSpy: Mock;
    let consoleErrorSpy: Mock;
    let consoleClearSpy: Mock;
    let processStdoutWriteSpy: Mock;
    let processStdoutClearLineSpy: Mock;
    let processStdoutCursorToSpy: Mock;

    beforeEach(() => {
        // Mock console methods
        consoleLogSpy = vi.spyOn(console, "log").mockImplementation(vi.fn());
        consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(vi.fn());
        consoleClearSpy = vi.spyOn(console, "clear").mockImplementation(vi.fn());
        
        // Mock process.stdout methods
        processStdoutWriteSpy = vi.spyOn(process.stdout, "write").mockImplementation(() => true);
        // Mock clearLine and cursorTo if they don't exist
        if (!process.stdout.clearLine) {
            (process.stdout as any).clearLine = vi.fn(() => true);
        }
        if (!process.stdout.cursorTo) {
            (process.stdout as any).cursorTo = vi.fn(() => true);
        }
        processStdoutClearLineSpy = vi.spyOn(process.stdout, "clearLine").mockImplementation(() => true);
        processStdoutCursorToSpy = vi.spyOn(process.stdout, "cursorTo").mockImplementation(() => true);

        const options: ChatDisplayOptions = {
            showTimestamps: true,
            showUserIds: false,
            maxMessageLength: 2000,
            enableColors: true,
        };
        ui = new InteractiveChatUI(options);
    });

    afterEach(() => {
        vi.clearAllMocks();
        consoleLogSpy.mockRestore();
        consoleErrorSpy.mockRestore();
        consoleClearSpy.mockRestore();
        processStdoutWriteSpy.mockRestore();
        processStdoutClearLineSpy?.mockRestore();
        processStdoutCursorToSpy?.mockRestore();
    });

    describe("constructor", () => {
        it("should initialize with default options", () => {
            const defaultUI = new InteractiveChatUI();
            expect(defaultUI).toBeDefined();
            expect(defaultUI).toBeInstanceOf(InteractiveChatUI);
        });

        it("should initialize with custom options", () => {
            const customOptions: ChatDisplayOptions = {
                showTimestamps: false,
                showUserIds: true,
                maxMessageLength: 1000,
                enableColors: false,
            };
            const customUI = new InteractiveChatUI(customOptions);
            expect(customUI).toBeDefined();
        });
    });

    describe("clearScreen", () => {
        it("should clear the console and display header", () => {
            ui.clearScreen();

            expect(consoleClearSpy).toHaveBeenCalledTimes(1);
            expect(consoleLogSpy).toHaveBeenCalled();
            // Header should contain "Vrooli Interactive Chat"
            const calls = consoleLogSpy.mock.calls.flat().join("\n");
            expect(calls).toContain("Vrooli Interactive Chat");
        });
    });

    describe("displayMessage", () => {
        it("should display a user message", () => {
            const message: ChatMessage = {
                id: "msg-1",
                text: "Hello bot",
                user: { 
                    id: "user-123", 
                    name: "TestUser",
                    isBot: false, 
                } as User,
                createdAt: new Date().toISOString(),
            } as ChatMessage;

            ui.displayMessage(message);

            expect(consoleLogSpy).toHaveBeenCalled();
            const output = consoleLogSpy.mock.calls.flat().join("\n");
            expect(output).toContain("TestUser");
            expect(output).toContain("Hello bot");
            expect(output).toContain("👤"); // User icon
        });

        it("should display a bot message", () => {
            const message: ChatMessage = {
                id: "msg-2",
                text: "Hello user",
                user: { 
                    id: "bot-123", 
                    name: "TestBot",
                    isBot: true, 
                } as User,
                createdAt: new Date().toISOString(),
            } as ChatMessage;

            ui.displayMessage(message);

            expect(consoleLogSpy).toHaveBeenCalled();
            const output = consoleLogSpy.mock.calls.flat().join("\n");
            expect(output).toContain("TestBot");
            expect(output).toContain("Hello user");
            expect(output).toContain("🤖"); // Bot icon
        });
    });

    describe("streaming responses", () => {
        it("should start streaming response", () => {
            const botUser: User = {
                id: "bot-123",
                name: "TestBot",
                isBot: true,
            } as User;

            ui.startStreamingResponse(botUser);

            expect(consoleLogSpy).toHaveBeenCalled();
            const output = consoleLogSpy.mock.calls[0][0];
            expect(output).toContain("🤖");
            expect(output).toContain("TestBot");
        });

        it("should append to streaming response", () => {
            ui.appendToStreamingResponse("Hello ");
            
            expect(processStdoutClearLineSpy).toHaveBeenCalledWith(0);
            expect(processStdoutCursorToSpy).toHaveBeenCalledWith(0);
            expect(processStdoutWriteSpy).toHaveBeenCalled();
            
            ui.appendToStreamingResponse("world!");
            
            expect(processStdoutWriteSpy).toHaveBeenLastCalledWith(expect.stringContaining("Hello world!"));
        });

        it("should finish streaming response", () => {
            ui.finishStreamingResponse();

            expect(consoleLogSpy).toHaveBeenCalledTimes(2); // Two new lines
        });
    });

    describe("displayBotStatus", () => {
        it("should display thinking status", () => {
            const status: BotStatus = {
                status: "thinking",
            };

            ui.displayBotStatus(status);

            expect(consoleLogSpy).toHaveBeenCalled();
            const output = consoleLogSpy.mock.calls[0][0];
            expect(output).toContain("💭");
            expect(output).toContain("thinking");
        });

        it("should display tool calling status", () => {
            const status: BotStatus = {
                status: "tool_calling",
                message: "Searching database",
            };

            ui.displayBotStatus(status);

            expect(consoleLogSpy).toHaveBeenCalled();
            const output = consoleLogSpy.mock.calls[0][0];
            expect(output).toContain("🔧");
            expect(output).toContain("Searching database");
        });

        it("should display error status", () => {
            const status: BotStatus = {
                status: "error",
                message: "Something went wrong",
            };

            ui.displayBotStatus(status);

            expect(consoleLogSpy).toHaveBeenCalled();
            const output = consoleLogSpy.mock.calls[0][0];
            expect(output).toContain("❌");
            expect(output).toContain("Something went wrong");
        });

        it("should not display idle status", () => {
            const status: BotStatus = {
                status: "idle",
            };

            ui.displayBotStatus(status);

            expect(consoleLogSpy).not.toHaveBeenCalled();
        });
    });

    describe("displayTypingIndicator", () => {
        it("should display single user typing", () => {
            ui.displayTypingIndicator(["Alice"]);

            expect(consoleLogSpy).toHaveBeenCalled();
            const output = consoleLogSpy.mock.calls[0][0];
            expect(output).toContain("Alice");
            expect(output).toContain("is typing");
        });

        it("should display multiple users typing", () => {
            ui.displayTypingIndicator(["Alice", "Bob"]);

            expect(consoleLogSpy).toHaveBeenCalled();
            const output = consoleLogSpy.mock.calls[0][0];
            expect(output).toContain("Alice, Bob");
            expect(output).toContain("are typing");
        });

        it("should not display anything for empty array", () => {
            ui.displayTypingIndicator([]);

            expect(consoleLogSpy).not.toHaveBeenCalled();
        });
    });

    describe("displayConnectionStatus", () => {
        it("should display connected status", () => {
            ui.displayConnectionStatus(true, "chat-123");

            expect(consoleLogSpy).toHaveBeenCalled();
            const output = consoleLogSpy.mock.calls[0][0];
            expect(output).toContain("Connected");
            expect(output).toContain("chat-123");
        });

        it("should display disconnected status", () => {
            ui.displayConnectionStatus(false, "chat-123");

            expect(consoleLogSpy).toHaveBeenCalled();
            const output = consoleLogSpy.mock.calls[0][0];
            expect(output).toContain("Disconnected");
            expect(output).toContain("chat-123");
        });
    });

    describe("message helpers", () => {
        it("should display error message", () => {
            ui.displayError("Network error");

            expect(consoleLogSpy).toHaveBeenCalled();
            const output = consoleLogSpy.mock.calls[0][0];
            expect(output).toContain("❌");
            expect(output).toContain("Network error");
        });

        it("should display info message", () => {
            ui.displayInfo("Loading data...");

            expect(consoleLogSpy).toHaveBeenCalled();
            const output = consoleLogSpy.mock.calls[0][0];
            expect(output).toContain("ℹ️");
            expect(output).toContain("Loading data...");
        });

        it("should display success message", () => {
            ui.displaySuccess("Operation complete");

            expect(consoleLogSpy).toHaveBeenCalled();
            const output = consoleLogSpy.mock.calls[0][0];
            expect(output).toContain("✅");
            expect(output).toContain("Operation complete");
        });

        it("should display warning message", () => {
            ui.displayWarning("Please be careful");

            expect(consoleLogSpy).toHaveBeenCalled();
            const output = consoleLogSpy.mock.calls[0][0];
            expect(output).toContain("⚠️");
            expect(output).toContain("Please be careful");
        });
    });

    describe("displayTools", () => {
        it("should display tool execution info", () => {
            ui.displayToolExecution("TestTool", { param1: "value1" });

            expect(consoleLogSpy).toHaveBeenCalled();
            const output = consoleLogSpy.mock.calls[0][0];
            expect(output).toContain("🔧");
            expect(output).toContain("TestTool");
        });

        it.skip("should display tool result", () => {
            // Method doesn't exist yet - needs implementation
            // ui.displayToolResult("TestTool", { success: true });
        });

        it.skip("should display tool error", () => {
            // Method doesn't exist yet - needs implementation
            // ui.displayToolError("TestTool", "Failed to execute");
        });
    });

    describe("displaySlashCommand", () => {
        it.skip("should display slash command result", () => {
            // Method doesn't exist yet - needs implementation
            // ui.displaySlashCommandResult("/help", "Available commands...");
        });
    });
});
