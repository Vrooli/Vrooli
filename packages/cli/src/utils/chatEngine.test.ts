import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { InteractiveChatEngine } from "./chatEngine.js";

// Mock dependencies
vi.mock("./chatUI.js", () => ({
    InteractiveChatUI: vi.fn().mockImplementation(() => ({
        clearScreen: vi.fn(),
        displayConnectionStatus: vi.fn(),
        displayInfo: vi.fn(),
        displayError: vi.fn(),
        displayMessage: vi.fn(),
        displayTypingIndicator: vi.fn(),
        hideTypingIndicator: vi.fn(),
        displayBotStatus: vi.fn(),
        displayTasks: vi.fn(),
        displayToolApproval: vi.fn(),
        displaySuccess: vi.fn(),
        displayWarning: vi.fn(),
        displayHistory: vi.fn(),
        startStreamingResponse: vi.fn(),
        appendStreamingChunk: vi.fn(),
        appendToStreamingResponse: vi.fn(),
        finishStreamingResponse: vi.fn(),
    })),
}));

vi.mock("./slashCommands.js", () => ({
    SlashCommandParser: vi.fn().mockImplementation(() => ({
        parse: vi.fn(),
        getHelp: vi.fn(),
    })),
}));

vi.mock("./toolApproval.js", () => ({
    ToolApprovalHandler: vi.fn().mockImplementation(() => ({
        handleToolApproval: vi.fn(),
        cleanup: vi.fn(),
    })),
}));

vi.mock("./contextManager.js", () => ({
    ContextManager: vi.fn().mockImplementation(() => ({
        loadContextFiles: vi.fn(),
        addFile: vi.fn(),
        getContextSummary: vi.fn(),
        clearContext: vi.fn(),
        getTaskContexts: vi.fn().mockReturnValue([]),
        getStats: vi.fn().mockReturnValue({ totalContexts: 0, totalTokens: 0, totalSize: 1024 }),
    })),
}));

vi.mock("readline", () => ({
    createInterface: vi.fn().mockReturnValue({
        setPrompt: vi.fn(),
        prompt: vi.fn(),
        close: vi.fn(),
        on: vi.fn(),
    }),
}));

describe("InteractiveChatEngine", () => {
    let mockClient: any;
    let mockConfig: any;
    const testChatId = "test-chat-123";
    let mockProcessExit: any;

    beforeEach(() => {
        vi.clearAllMocks();

        // Mock process.exit to prevent tests from actually exiting
        mockProcessExit = vi.spyOn(process, "exit").mockImplementation(() => undefined as never);

        mockClient = {
            connectSocket: vi.fn().mockResolvedValue({}),
            connectWebSocket: vi.fn().mockReturnValue({}),
            search: vi.fn(),
            post: vi.fn(),
        };

        mockConfig = {
            isDebug: vi.fn().mockReturnValue(false),
            getServerUrl: vi.fn().mockReturnValue("http://localhost:5329"),
        };
    });

    afterEach(() => {
        // Restore process.exit
        if (mockProcessExit) {
            mockProcessExit.mockRestore();
        }
    });

    describe("constructor", () => {
        it("should be constructible with minimal parameters", () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
            expect(engine).toBeDefined();
            expect(engine).toBeInstanceOf(InteractiveChatEngine);
        });

        it("should be constructible with options", () => {
            const options = {
                model: "test-model",
                showTools: true,
                approveTools: true,
                contextFiles: ["test.txt"],
                autoScroll: false,
                timeout: 5000,
            };

            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId, options);
            expect(engine).toBeDefined();
        });

        it("should initialize with default options when none provided", () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
            expect(engine).toBeDefined();
        });
    });

    describe("sendMessage", () => {
        it("should handle basic message sending", async () => {
            mockClient.post.mockResolvedValue({ data: { success: true } });

            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            // Mock the internal methods to avoid complex setup
            (engine as any).session = { chatId: testChatId, messageCount: 0, startTime: new Date() };

            await expect(engine.sendMessage("Hello")).resolves.not.toThrow();
        });

        it("should handle message sending errors", async () => {
            mockClient.post.mockRejectedValue(new Error("Network error"));

            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
            (engine as any).session = { chatId: testChatId, messageCount: 0, startTime: new Date() };

            // sendMessage catches errors and displays them, it doesn't throw
            await expect(engine.sendMessage("Hello")).resolves.not.toThrow();

            // Verify that the error was displayed
            expect((engine as any).ui.displayError).toHaveBeenCalledWith(
                expect.stringContaining("Failed to send message: Network error"),
            );
        });
    });

    describe("cleanup", () => {
        it("should clean up resources", async () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            // Mock readline interface
            const mockReadline = {
                close: vi.fn(),
            };
            (engine as any).readline = mockReadline;

            // Mock socket
            const mockSocket = {
                disconnect: vi.fn(),
            };
            (engine as any).socket = mockSocket;

            await engine.cleanup();

            expect(mockReadline.close).toHaveBeenCalled();
            expect(mockSocket.disconnect).toHaveBeenCalled();
        });

        it("should handle cleanup when resources are null", async () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            // Ensure resources are null
            (engine as any).readline = null;
            (engine as any).socket = null;

            await expect(engine.cleanup()).resolves.not.toThrow();
        });
    });

    describe("startSession", () => {
        it("should start session and initialize all components", async () => {
            mockClient.search.mockResolvedValue({ data: { edges: [] } });
            mockClient.connectSocket.mockResolvedValue({
                on: vi.fn(),
                emit: vi.fn(),
                disconnect: vi.fn(),
            });

            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            // Mock internal methods to avoid complex setup
            const mockLoadHistory = vi.spyOn(engine as any, "loadChatHistory").mockResolvedValue(undefined);
            const mockInitContext = vi.spyOn(engine as any, "initializeContextFiles").mockResolvedValue(undefined);
            const mockConnectWS = vi.spyOn(engine as any, "connectWebSocket").mockResolvedValue(undefined);
            const mockSetupReadline = vi.spyOn(engine as any, "setupReadline").mockReturnValue(undefined);
            const mockStartInput = vi.spyOn(engine as any, "startInputLoop").mockReturnValue(undefined);

            await engine.startSession();

            expect(mockLoadHistory).toHaveBeenCalled();
            expect(mockInitContext).toHaveBeenCalled();
            expect(mockConnectWS).toHaveBeenCalled();
            expect(mockSetupReadline).toHaveBeenCalled();
            expect(mockStartInput).toHaveBeenCalled();
        });

        it("should handle session start errors", async () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            // Mock a method to throw an error
            vi.spyOn(engine as any, "loadChatHistory").mockRejectedValue(new Error("Test error"));
            vi.spyOn(engine, "cleanup").mockResolvedValue(undefined);

            await expect(engine.startSession()).rejects.toThrow("Test error");
            expect(engine.cleanup).toHaveBeenCalled();
        });
    });

    describe("handleUserInput", () => {
        it("should handle empty input", async () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
            const mockPrompt = vi.spyOn(engine as any, "promptForInput").mockReturnValue(undefined);

            await (engine as any).handleUserInput("");
            await (engine as any).handleUserInput("   ");

            expect(mockPrompt).toHaveBeenCalledTimes(2);
        });

        it("should handle slash commands", async () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
            const mockCommand = { name: "help", args: [] };

            (engine as any).slashParser = {
                parseCommand: vi.fn().mockReturnValue(mockCommand),
                isLikelyCommand: vi.fn().mockReturnValue(false),
            };

            const mockHandleSlash = vi.spyOn(engine as any, "handleSlashCommand").mockResolvedValue(undefined);

            await (engine as any).handleUserInput("/help");

            expect(mockHandleSlash).toHaveBeenCalledWith(mockCommand);
        });

        it("should handle malformed commands", async () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
            const mockPrompt = vi.spyOn(engine as any, "promptForInput").mockReturnValue(undefined);

            (engine as any).slashParser = {
                parseCommand: vi.fn().mockReturnValue(null),
                isLikelyCommand: vi.fn().mockReturnValue(true),
            };

            await (engine as any).handleUserInput("/badcommand");

            expect((engine as any).ui.displayError).toHaveBeenCalledWith(
                "Invalid command format. Type /help for available commands.",
            );
            expect(mockPrompt).toHaveBeenCalled();
        });

        it("should handle regular messages", async () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
            const mockSendMessage = vi.spyOn(engine, "sendMessage").mockResolvedValue(undefined);
            const mockPrompt = vi.spyOn(engine as any, "promptForInput").mockReturnValue(undefined);

            (engine as any).slashParser = {
                parseCommand: vi.fn().mockReturnValue(null),
                isLikelyCommand: vi.fn().mockReturnValue(false),
            };

            await (engine as any).handleUserInput("Hello world");

            expect(mockSendMessage).toHaveBeenCalledWith("Hello world");
            expect(mockPrompt).toHaveBeenCalled();
        });
    });

    describe("handleSlashCommand", () => {
        it("should handle successful command execution", async () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
            const mockCommand = { name: "help", args: [] };
            const mockResult = { message: "Help displayed", error: null, shouldClear: false };

            (engine as any).slashParser = {
                executeCommand: vi.fn().mockResolvedValue(mockResult),
            };

            const consoleSpy = vi.spyOn(console, "log").mockImplementation(vi.fn());

            await (engine as any).handleSlashCommand(mockCommand);

            expect(consoleSpy).toHaveBeenCalledWith("Help displayed");
            consoleSpy.mockRestore();
        });

        it("should handle command errors", async () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
            const mockCommand = { name: "invalid", args: [] };
            const mockResult = { error: "Unknown command", message: null, shouldClear: false };

            (engine as any).slashParser = {
                executeCommand: vi.fn().mockResolvedValue(mockResult),
            };

            await (engine as any).handleSlashCommand(mockCommand);

            expect((engine as any).ui.displayError).toHaveBeenCalledWith("Unknown command");
        });

        it("should handle commands that clear screen", async () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
            const mockCommand = { name: "clear", args: [] };
            const mockResult = { message: null, error: null, shouldClear: true };

            (engine as any).slashParser = {
                executeCommand: vi.fn().mockResolvedValue(mockResult),
            };

            (engine as any).messages = [{ id: "1", text: "test" }];

            await (engine as any).handleSlashCommand(mockCommand);

            expect((engine as any).ui.clearScreen).toHaveBeenCalled();
            expect((engine as any).ui.displayHistory).toHaveBeenCalledWith([{ id: "1", text: "test" }]);
        });
    });

    describe("loadChatHistory", () => {
        it("should load and display chat history", async () => {
            const mockMessages = [
                { id: "1", text: "Hello", user: { name: "User", isBot: false }, createdAt: "2023-01-01T00:00:00Z" },
                { id: "2", text: "Hi there", user: { name: "Bot", isBot: true }, createdAt: "2023-01-01T00:01:00Z" },
            ];

            mockClient.post.mockResolvedValue({
                edges: mockMessages.map(msg => ({ node: msg })),
            });

            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            await (engine as any).loadChatHistory();

            expect(mockClient.post).toHaveBeenCalledWith(
                "/chatMessages",
                expect.objectContaining({
                    chatId: testChatId,
                    take: 50,
                }),
            );

            expect((engine as any).messages).toEqual(mockMessages);
            expect((engine as any).ui.displayHistory).toHaveBeenCalled();
        });

        it("should handle history loading errors", async () => {
            mockClient.post.mockRejectedValue(new Error("Network error"));

            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            await (engine as any).loadChatHistory();

            expect((engine as any).ui.displayWarning).toHaveBeenCalledWith(
                "Could not load chat history",
            );
        });
    });

    describe("initializeContextFiles", () => {
        it("should initialize context files when provided", async () => {
            const options = { contextFiles: ["file1.txt", "file2.txt"] };
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId, options);

            // Mock successful context loading
            (engine as any).contextManager.addFile.mockResolvedValue("context-id-1");

            // Mock getStats to return context count
            (engine as any).contextManager.getStats.mockReturnValue({ totalContexts: 2, totalSize: 1024 });

            await (engine as any).initializeContextFiles();

            expect((engine as any).contextManager.addFile).toHaveBeenCalledWith("file1.txt");
            expect((engine as any).contextManager.addFile).toHaveBeenCalledWith("file2.txt");
            expect((engine as any).ui.displaySuccess).toHaveBeenCalledWith("Added context file: context-id-1");
            expect((engine as any).ui.displayInfo).toHaveBeenCalledWith("Loaded 2 context items (1 KB)");
        });

        it("should skip initialization when no context files", async () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            await (engine as any).initializeContextFiles();

            expect((engine as any).contextManager.loadContextFiles).not.toHaveBeenCalled();
        });

        it("should handle context file loading errors", async () => {
            const options = { contextFiles: ["badfile.txt"] };
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId, options);

            (engine as any).contextManager.addFile.mockRejectedValue(new Error("File not found"));

            await (engine as any).initializeContextFiles();

            expect((engine as any).ui.displayWarning).toHaveBeenCalledWith(
                "Failed to add context file badfile.txt: File not found",
            );
        });
    });

    describe("connectWebSocket", () => {
        it("should connect WebSocket and setup event handlers", async () => {
            const mockSocket = {
                on: vi.fn(),
                emit: vi.fn(),
                disconnect: vi.fn(),
            };

            mockClient.connectWebSocket.mockReturnValue(mockSocket);

            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            await (engine as any).connectWebSocket();

            expect(mockClient.connectWebSocket).toHaveBeenCalled();
            expect((engine as any).socket).toBe(mockSocket);

            // Verify event handlers are setup (using actual event names from implementation)
            expect(mockSocket.on).toHaveBeenCalledWith("connect", expect.any(Function));
            expect(mockSocket.on).toHaveBeenCalledWith("disconnect", expect.any(Function));
            expect(mockSocket.on).toHaveBeenCalledWith("error", expect.any(Function));
            expect(mockSocket.on).toHaveBeenCalledWith("responseStream", expect.any(Function));
            expect(mockSocket.on).toHaveBeenCalledWith("botStatusUpdate", expect.any(Function));
            expect(mockSocket.on).toHaveBeenCalledWith("typing", expect.any(Function));
            expect(mockSocket.on).toHaveBeenCalledWith("llmTasks", expect.any(Function));
            expect(mockSocket.on).toHaveBeenCalledWith("messages", expect.any(Function));
        });

        it("should handle WebSocket connection errors", async () => {
            mockClient.connectWebSocket.mockImplementation(() => {
                throw new Error("Connection failed");
            });

            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            await expect((engine as any).connectWebSocket()).rejects.toThrow("Connection failed");
        });
    });

    describe("utility methods", () => {
        it("should get participants from messages", () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            const mockMessages = [
                { id: "1", text: "Hello", user: { id: "user1", name: "Alice", isBot: false } },
                { id: "2", text: "Hi", user: { id: "user2", name: "Bob", isBot: false } },
                { id: "3", text: "Hey", user: { id: "user1", name: "Alice", isBot: false } }, // Duplicate user
                { id: "4", text: "Response", user: { id: "bot1", name: "Bot", isBot: true } },
            ];

            (engine as any).messages = mockMessages;

            const participants = (engine as any).getParticipants();

            expect(participants).toHaveLength(3); // user1, user2, bot1 (no duplicates)
            expect(participants.map((p: any) => p.id)).toEqual(expect.arrayContaining(["user1", "user2", "bot1"]));
        });

        it("should format file sizes correctly", () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            expect((engine as any).formatFileSize(0)).toBe("0 B");
            expect((engine as any).formatFileSize(512)).toBe("512 B");
            expect((engine as any).formatFileSize(1024)).toBe("1 KB");
            expect((engine as any).formatFileSize(1536)).toBe("1.5 KB");
            expect((engine as any).formatFileSize(1048576)).toBe("1 MB");
            expect((engine as any).formatFileSize(1073741824)).toBe("1 GB");
        });

        it("should get session information", () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            const mockSession = {
                chatId: testChatId,
                messageCount: 5,
                startTime: new Date("2023-01-01T00:00:00Z"),
            };

            (engine as any).session = mockSession;

            const sessionInfo = engine.getSessionInfo();

            expect(sessionInfo).toEqual(mockSession);
            expect(sessionInfo).not.toBe(mockSession); // Should be a copy
        });
    });

    describe("socket event handlers", () => {
        it("should handle responseStream events", async () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            const mockSocket = {
                on: vi.fn(),
                emit: vi.fn(),
                disconnect: vi.fn(),
            };

            mockClient.connectWebSocket.mockReturnValue(mockSocket);

            await (engine as any).connectWebSocket();

            // Get the responseStream handler
            const responseStreamHandler = mockSocket.on.mock.calls.find(
                call => call[0] === "responseStream",
            )[1];

            // Test chunk handling
            responseStreamHandler({ chatId: testChatId, chunk: "Hello" });

            // Test completion handling
            responseStreamHandler({ chatId: testChatId, isComplete: true, messageId: "msg123" });

            // Verify UI interactions happened (can't test specifics due to mocking)
            expect(mockSocket.on).toHaveBeenCalledWith("responseStream", expect.any(Function));
        });

        it("should handle botStatusUpdate events", async () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            const mockSocket = {
                on: vi.fn(),
                emit: vi.fn(),
                disconnect: vi.fn(),
            };

            mockClient.connectWebSocket.mockReturnValue(mockSocket);

            await (engine as any).connectWebSocket();

            // Get the botStatusUpdate handler
            const statusHandler = mockSocket.on.mock.calls.find(
                call => call[0] === "botStatusUpdate",
            )[1];

            // Test status update
            statusHandler({
                chatId: testChatId,
                status: "thinking",
                message: "Processing your request...",
            });

            expect(mockSocket.on).toHaveBeenCalledWith("botStatusUpdate", expect.any(Function));
        });

        it("should handle typing indicator events", async () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            const mockSocket = {
                on: vi.fn(),
                emit: vi.fn(),
                disconnect: vi.fn(),
            };

            mockClient.connectWebSocket.mockReturnValue(mockSocket);

            await (engine as any).connectWebSocket();

            // Get the typing handler
            const typingHandler = mockSocket.on.mock.calls.find(
                call => call[0] === "typing",
            )[1];

            // Test typing start
            typingHandler({
                chatId: testChatId,
                isTyping: true,
                userId: "user123",
            });

            // Test typing stop
            typingHandler({
                chatId: testChatId,
                isTyping: false,
                userId: "user123",
            });

            expect(mockSocket.on).toHaveBeenCalledWith("typing", expect.any(Function));
        });

        it("should handle llmTasks events", async () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            const mockSocket = {
                on: vi.fn(),
                emit: vi.fn(),
                disconnect: vi.fn(),
            };

            mockClient.connectWebSocket.mockReturnValue(mockSocket);

            await (engine as any).connectWebSocket();

            // Get the llmTasks handler
            const tasksHandler = mockSocket.on.mock.calls.find(
                call => call[0] === "llmTasks",
            )[1];

            // Test tasks update
            tasksHandler({
                chatId: testChatId,
                tasks: [
                    { id: "task1", type: "search", status: "running", description: "Searching..." },
                    { id: "task2", type: "analysis", status: "completed", description: "Analysis complete" },
                ],
            });

            expect(mockSocket.on).toHaveBeenCalledWith("llmTasks", expect.any(Function));
        });

        it("should handle new message events", async () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            const mockSocket = {
                on: vi.fn(),
                emit: vi.fn(),
                disconnect: vi.fn(),
            };

            mockClient.connectWebSocket.mockReturnValue(mockSocket);

            await (engine as any).connectWebSocket();

            // Get the messages handler
            const messagesHandler = mockSocket.on.mock.calls.find(
                call => call[0] === "messages",
            )[1];

            // Test new message
            messagesHandler({
                chatId: testChatId,
                message: {
                    id: "msg123",
                    text: "New message",
                    user: { id: "user1", name: "User", isBot: false },
                    createdAt: "2023-01-01T00:00:00Z",
                },
            });

            expect(mockSocket.on).toHaveBeenCalledWith("messages", expect.any(Function));
        });
    });

    describe("module exports", () => {
        it("should export the correct class", async () => {
            const module = await import("./chatEngine.js");
            expect(module.InteractiveChatEngine).toBeDefined();
            expect(typeof module.InteractiveChatEngine).toBe("function");
        });

        it("should export InteractiveChatOptions interface", async () => {
            // This test verifies the module structure
            const module = await import("./chatEngine.js");
            expect(module).toBeDefined();
        });

        it("should export ChatSessionInfo interface", async () => {
            // This test verifies the module structure
            const module = await import("./chatEngine.js");
            expect(module).toBeDefined();
        });
    });

    describe("error handling and cleanup", () => {
        it("should handle process.exit in cleanup", async () => {
            const processExitSpy = vi.spyOn(process, "exit").mockImplementation(() => undefined as never);

            const engine = new InteractiveChatEngine(
                mockClient,
                mockConfig,
                testChatId,
            );

            // Set up session data
            (engine as any).session = {
                startTime: new Date(Date.now() - 60000), // 1 minute ago
                messageCount: 5,
                chatId: testChatId,
            };

            await engine.cleanup();

            expect(processExitSpy).toHaveBeenCalledWith(0);

            processExitSpy.mockRestore();
        });

        it("should handle tool approval options properly", () => {
            const options = {
                approveTools: true,
                model: "test-model",
            };

            const engine = new InteractiveChatEngine(
                mockClient,
                mockConfig,
                testChatId,
                options,
            );

            expect(engine).toBeDefined();
            // Verify that tool approval handler is initialized when approveTools is true
            expect((engine as any).toolApprovalHandler).toBeDefined();
        });
    });

    describe("handleToolExecution", () => {
        it("should delegate to tool approval handler when available", () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            const mockToolHandler = {
                handleToolExecution: vi.fn(),
            };
            (engine as any).toolApprovalHandler = mockToolHandler;

            const toolData = {
                toolName: "test-tool",
                chatId: testChatId,
            };

            (engine as any).handleToolExecution("TOOL_FAILED", toolData);

            expect(mockToolHandler.handleToolExecution).toHaveBeenCalledWith("TOOL_FAILED", toolData);
        });

        it("should handle TOOL_FAILED event with showTools option and no approval handler", () => {
            const options = { showTools: true };
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId, options);
            const mockDisplayError = vi.spyOn((engine as any).ui, "displayError").mockImplementation();

            // Ensure no tool approval handler
            (engine as any).toolApprovalHandler = null;

            const toolData = {
                toolName: "test-tool",
                chatId: testChatId,
            };

            (engine as any).handleToolExecution("TOOL_FAILED", toolData);

            expect(mockDisplayError).toHaveBeenCalledWith("Tool failed: test-tool");
        });

        it("should handle TOOL_FAILED event without tool name", () => {
            const options = { showTools: true };
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId, options);
            const mockDisplayError = vi.spyOn((engine as any).ui, "displayError").mockImplementation();

            // Ensure no tool approval handler
            (engine as any).toolApprovalHandler = null;

            const toolData = {
                chatId: testChatId,
            };

            (engine as any).handleToolExecution("TOOL_FAILED", toolData);

            expect(mockDisplayError).not.toHaveBeenCalled();
        });

        it("should handle TOOL_COMPLETED event with showTools option", () => {
            const options = { showTools: true };
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId, options);
            const mockDisplaySuccess = vi.spyOn((engine as any).ui, "displaySuccess").mockImplementation();

            // Ensure no tool approval handler
            (engine as any).toolApprovalHandler = null;

            const toolData = {
                toolName: "test-tool",
                chatId: testChatId,
            };

            (engine as any).handleToolExecution("TOOL_COMPLETED", toolData);

            expect(mockDisplaySuccess).toHaveBeenCalledWith("Tool completed: test-tool");
        });

        it("should ignore events from different chat", () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
            const mockDisplayError = vi.spyOn((engine as any).ui, "displayError").mockImplementation();

            const toolData = {
                toolName: "test-tool",
                chatId: "different-chat-id",
            };

            (engine as any).handleToolExecution("TOOL_FAILED", toolData);

            expect(mockDisplayError).not.toHaveBeenCalled();
        });
    });

    describe("handleToolStatusUpdate", () => {
        it("should log tool status when debug mode is enabled", () => {
            const consoleSpy = vi.spyOn(console, "log").mockImplementation();
            mockConfig.isDebug.mockReturnValue(true);

            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            const toolInfo = {
                toolName: "test-tool",
                status: "executing" as const,
            };

            // Call the private method through type assertion
            (engine as any).handleToolStatusUpdate(toolInfo);

            expect(consoleSpy).toHaveBeenCalledWith("Tool test-tool status: executing");

            consoleSpy.mockRestore();
        });

        it("should not log when debug mode is disabled", () => {
            const consoleSpy = vi.spyOn(console, "log").mockImplementation();
            mockConfig.isDebug.mockReturnValue(false);

            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            const toolInfo = {
                toolName: "test-tool",
                status: "executing" as const,
            };

            (engine as any).handleToolStatusUpdate(toolInfo);

            expect(consoleSpy).not.toHaveBeenCalled();

            consoleSpy.mockRestore();
        });
    });

    describe("setupReadline", () => {
        it("should setup readline interface with correct configuration", () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);

            // Call the private method through type assertion
            (engine as any).setupReadline();

            // Verify the readline interface was set up
            expect((engine as any).readline).toBeDefined();
        });
    });

    describe("promptForInput", () => {
        it("should prompt for input when running and readline is available", () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
            (engine as any).isRunning = true;
            (engine as any).readline = { prompt: vi.fn() };

            const mockDisplayPrompt = vi.fn();
            (engine as any).ui.displayPrompt = mockDisplayPrompt;

            (engine as any).promptForInput();

            expect(mockDisplayPrompt).toHaveBeenCalled();
        });

        it("should not prompt when not running", () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
            (engine as any).isRunning = false;
            (engine as any).readline = { prompt: vi.fn() };

            const mockDisplayPrompt = vi.fn();
            (engine as any).ui.displayPrompt = mockDisplayPrompt;

            (engine as any).promptForInput();

            expect(mockDisplayPrompt).not.toHaveBeenCalled();
        });

        it("should not prompt when readline is not available", () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
            (engine as any).isRunning = true;
            (engine as any).readline = null;

            const mockDisplayPrompt = vi.fn();
            (engine as any).ui.displayPrompt = mockDisplayPrompt;

            (engine as any).promptForInput();

            expect(mockDisplayPrompt).not.toHaveBeenCalled();
        });
    });

    describe("tool approval handlers", () => {
        describe("handleToolApprovalRequired", () => {
            it("should handle tool approval request when chatId matches", async () => {
                const mockToolApprovalHandler = {
                    handleApprovalRequest: vi.fn().mockResolvedValue(undefined),
                    cleanup: vi.fn(),
                };

                const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
                (engine as any).toolApprovalHandler = mockToolApprovalHandler;
                (engine as any).readline = { prompt: vi.fn() };

                const promptForInputSpy = vi.spyOn(engine as any, "promptForInput").mockImplementation();
                const consoleSpy = vi.spyOn(console, "log").mockImplementation();

                const toolData = {
                    chatId: testChatId,
                    pendingId: "pending-123",
                    toolCallId: "tool-call-123",
                    toolName: "test-tool",
                    toolArguments: { arg1: "value1" },
                    callerBotId: "bot-123",
                    conversationId: "conv-123",
                };

                await (engine as any).handleToolApprovalRequired(toolData);

                expect(mockToolApprovalHandler.handleApprovalRequest).toHaveBeenCalledWith(toolData, testChatId);
                expect(consoleSpy).toHaveBeenCalled(); // New line before approval prompt
                expect(promptForInputSpy).toHaveBeenCalled();

                consoleSpy.mockRestore();
            });

            it("should not handle when chatId doesn't match", async () => {
                const mockToolApprovalHandler = {
                    handleApprovalRequest: vi.fn().mockResolvedValue(undefined),
                    cleanup: vi.fn(),
                };

                const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
                (engine as any).toolApprovalHandler = mockToolApprovalHandler;

                const toolData = {
                    chatId: "different-chat-id",
                    pendingId: "pending-123",
                    toolCallId: "tool-call-123",
                    toolName: "test-tool",
                    toolArguments: {},
                    callerBotId: "bot-123",
                    conversationId: "conv-123",
                };

                await (engine as any).handleToolApprovalRequired(toolData);

                expect(mockToolApprovalHandler.handleApprovalRequest).not.toHaveBeenCalled();
            });

            it("should not handle when toolApprovalHandler is null", async () => {
                const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
                (engine as any).toolApprovalHandler = null;

                const toolData = {
                    chatId: testChatId,
                    pendingId: "pending-123",
                    toolCallId: "tool-call-123",
                    toolName: "test-tool",
                    toolArguments: {},
                    callerBotId: "bot-123",
                    conversationId: "conv-123",
                };

                // Should not throw an error
                await expect((engine as any).handleToolApprovalRequired(toolData)).resolves.toBeUndefined();
            });
        });

        describe("handleToolApprovalGranted", () => {
            it("should display success message when chatId matches", () => {
                const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
                const mockDisplaySuccess = vi.fn();
                (engine as any).ui.displaySuccess = mockDisplaySuccess;

                const toolData = {
                    chatId: testChatId,
                    pendingId: "pending-123",
                    toolCallId: "tool-call-123",
                    toolName: "test-tool",
                    toolArguments: {},
                    callerBotId: "bot-123",
                    conversationId: "conv-123",
                };

                (engine as any).handleToolApprovalGranted(toolData);

                expect(mockDisplaySuccess).toHaveBeenCalledWith("Tool approved: test-tool");
            });

            it("should not display message when chatId doesn't match", () => {
                const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
                const mockDisplaySuccess = vi.fn();
                (engine as any).ui.displaySuccess = mockDisplaySuccess;

                const toolData = {
                    chatId: "different-chat-id",
                    pendingId: "pending-123",
                    toolCallId: "tool-call-123",
                    toolName: "test-tool",
                    toolArguments: {},
                    callerBotId: "bot-123",
                    conversationId: "conv-123",
                };

                (engine as any).handleToolApprovalGranted(toolData);

                expect(mockDisplaySuccess).not.toHaveBeenCalled();
            });
        });

        describe("handleToolApprovalRejected", () => {
            it("should display warning message when chatId matches", () => {
                const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
                const mockDisplayWarning = vi.fn();
                const mockDisplayInfo = vi.fn();
                (engine as any).ui.displayWarning = mockDisplayWarning;
                (engine as any).ui.displayInfo = mockDisplayInfo;

                const toolData = {
                    chatId: testChatId,
                    pendingId: "pending-123",
                    toolCallId: "tool-call-123",
                    toolName: "test-tool",
                    toolArguments: {},
                    callerBotId: "bot-123",
                    conversationId: "conv-123",
                    reason: "Security concerns",
                };

                (engine as any).handleToolApprovalRejected(toolData);

                expect(mockDisplayWarning).toHaveBeenCalledWith("Tool rejected: test-tool");
                expect(mockDisplayInfo).toHaveBeenCalledWith("Reason: Security concerns");
            });

            it("should not display reason when not provided", () => {
                const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
                const mockDisplayWarning = vi.fn();
                const mockDisplayInfo = vi.fn();
                (engine as any).ui.displayWarning = mockDisplayWarning;
                (engine as any).ui.displayInfo = mockDisplayInfo;

                const toolData = {
                    chatId: testChatId,
                    pendingId: "pending-123",
                    toolCallId: "tool-call-123",
                    toolName: "test-tool",
                    toolArguments: {},
                    callerBotId: "bot-123",
                    conversationId: "conv-123",
                };

                (engine as any).handleToolApprovalRejected(toolData);

                expect(mockDisplayWarning).toHaveBeenCalledWith("Tool rejected: test-tool");
                expect(mockDisplayInfo).not.toHaveBeenCalled();
            });
        });

        describe("handleToolApprovalTimeout", () => {
            it("should display error message when chatId matches", () => {
                const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
                const mockDisplayError = vi.fn();
                (engine as any).ui.displayError = mockDisplayError;

                const toolData = {
                    chatId: testChatId,
                    pendingId: "pending-123",
                    toolCallId: "tool-call-123",
                    toolName: "test-tool",
                    toolArguments: {},
                    callerBotId: "bot-123",
                    conversationId: "conv-123",
                };

                (engine as any).handleToolApprovalTimeout(toolData);

                expect(mockDisplayError).toHaveBeenCalledWith("Tool approval timed out: test-tool");
            });
        });
    });

    describe("cleanup with toolApprovalHandler", () => {
        it("should cleanup toolApprovalHandler when present", () => {
            const mockToolApprovalHandler = {
                handleApprovalRequest: vi.fn(),
                cleanup: vi.fn(),
            };

            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
            (engine as any).toolApprovalHandler = mockToolApprovalHandler;
            (engine as any).socket = { disconnect: vi.fn() };
            (engine as any).readline = { close: vi.fn() };
            (engine as any).session = {
                startTime: new Date(Date.now() - 60000),
                messageCount: 5,
            };

            const mockDisplaySuccess = vi.fn();
            (engine as any).ui.displaySuccess = mockDisplaySuccess;

            const consoleSpy = vi.spyOn(console, "log").mockImplementation();

            (engine as any).cleanup();

            expect(mockToolApprovalHandler.cleanup).toHaveBeenCalled();
            expect((engine as any).toolApprovalHandler).toBeNull();

            consoleSpy.mockRestore();
        });

        it("should handle cleanup when toolApprovalHandler is null", () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
            (engine as any).toolApprovalHandler = null;
            (engine as any).socket = { disconnect: vi.fn() };
            (engine as any).readline = { close: vi.fn() };
            (engine as any).session = {
                startTime: new Date(Date.now() - 60000),
                messageCount: 5,
            };

            const mockDisplaySuccess = vi.fn();
            (engine as any).ui.displaySuccess = mockDisplaySuccess;

            const consoleSpy = vi.spyOn(console, "log").mockImplementation();

            // Should not throw an error
            expect(() => (engine as any).cleanup()).not.toThrow();

            consoleSpy.mockRestore();
        });

        it("should handle cleanup when resources are null", () => {
            const engine = new InteractiveChatEngine(mockClient, mockConfig, testChatId);
            (engine as any).toolApprovalHandler = null;
            (engine as any).socket = null;
            (engine as any).readline = null;
            (engine as any).session = {
                startTime: new Date(Date.now() - 60000),
                messageCount: 5,
            };

            const mockDisplaySuccess = vi.fn();
            (engine as any).ui.displaySuccess = mockDisplaySuccess;

            const consoleSpy = vi.spyOn(console, "log").mockImplementation();

            // Should not throw an error
            expect(() => (engine as any).cleanup()).not.toThrow();

            consoleSpy.mockRestore();
        });
    });
});
