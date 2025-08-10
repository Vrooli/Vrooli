import type {
    Chat,
    ChatMessage,
    ChatMessageSearchResult,
    ChatSearchResult,
    User,
    UserSearchResult,
} from "@vrooli/shared";
import chalk from "chalk";
import { Command } from "commander";
import { afterEach, beforeEach, describe, expect, it, vi, type MockedFunction } from "vitest";
import { type ApiClient } from "../utils/client.js";
import { type ConfigManager } from "../utils/config.js";
import { ChatCommands } from "./chat.js";
import { output } from "../utils/output.js";

// Mock dependencies
vi.mock("../utils/chatEngine.js", () => ({
    InteractiveChatEngine: vi.fn().mockImplementation(() => ({
        startSession: vi.fn().mockResolvedValue(undefined),
        stop: vi.fn(),
    })),
}));

vi.mock("ora", () => ({
    default: vi.fn(() => ({
        start: vi.fn().mockReturnThis(),
        succeed: vi.fn().mockReturnThis(),
        fail: vi.fn().mockReturnThis(),
        stop: vi.fn().mockReturnThis(),
    })),
}));

vi.mock("chalk", () => ({
    default: {
        bold: vi.fn((text: string) => text),
        dim: vi.fn((text: string) => text),
        green: vi.fn((text: string) => text),
        red: vi.fn((text: string) => text),
        yellow: vi.fn((text: string) => text),
        cyan: vi.fn((text: string) => text),
        gray: vi.fn((text: string) => text),
        blue: vi.fn((text: string) => text),
        underline: vi.fn((text: string) => text),
    },
}));

// Mock the output utility
vi.mock("../utils/output.js", () => ({
    output: {
        info: vi.fn(),
        success: vi.fn(),
        error: vi.fn(),
        warn: vi.fn(),
        debug: vi.fn(),
        json: vi.fn(),
        table: vi.fn(),
        raw: vi.fn(),
        newline: vi.fn(),
        section: vi.fn(),
        listItem: vi.fn(),
        keyValue: vi.fn(),
    },
}));

describe("ChatCommands", () => {
    let program: Command;
    let mockClient: ApiClient;
    let mockConfig: ConfigManager;
    let chatCommands: ChatCommands;
    let processExitSpy: any;

    beforeEach(() => {
        // Clear all mocks before each test
        vi.clearAllMocks();
        
        // Mock process.exit
        processExitSpy = vi.spyOn(process, "exit").mockImplementation(() => undefined as never);

        // Create mocks
        program = new Command();
        program.exitOverride(); // Prevent commander from exiting process

        mockClient = {
            post: vi.fn(),
            get: vi.fn(),
            getSocket: vi.fn(),
            requestWithEndpoint: vi.fn(),
            isAuthenticated: vi.fn(),
            connectWebSocket: vi.fn(),
        } as unknown as ApiClient;

        mockConfig = {
            isJsonOutput: vi.fn().mockReturnValue(false),
            getServerUrl: vi.fn().mockReturnValue("http://localhost:5329"),
            getAuthToken: vi.fn().mockReturnValue("test-token"),
        } as unknown as ConfigManager;

        // Create instance
        chatCommands = new ChatCommands(program, mockClient, mockConfig);
    });

    afterEach(() => {
        processExitSpy.mockRestore();
        vi.clearAllMocks();
    });

    describe("list-bots command", () => {
        const mockBots: UserSearchResult = {
            edges: [
                {
                    cursor: "cursor1",
                    node: {
                        id: "bot1",
                        name: "TestBot",
                        handle: "testbot",
                        isBot: true,
                        isPrivate: false,
                        botSettings: {
                            model: "gpt-4",
                            creativity: 0.7,
                            verbosity: 0.5,
                            translations: [],
                            startingMessage: "Hello!",
                            __typename: "BotSettings",
                        },
                        __typename: "User",
                    } as User,
                },
                {
                    cursor: "cursor2",
                    node: {
                        id: "bot2",
                        name: "HelperBot",
                        handle: "helperbot",
                        isBot: true,
                        isPrivate: false,
                        botSettings: {
                            model: "claude-3",
                            creativity: 0.5,
                            verbosity: 0.3,
                            translations: [],
                            startingMessage: "How can I help?",
                            __typename: "BotSettings",
                        },
                        __typename: "User",
                    } as User,
                },
            ],
            pageInfo: {
                hasNextPage: false,
                hasPreviousPage: false,
                __typename: "PageInfo",
            },
            __typename: "UserSearchResult",
        };

        it("should list available bots in table format", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockBots);

            await program.parseAsync(["node", "test", "chat", "list-bots"]);

            expect(mockClient.requestWithEndpoint).toHaveBeenCalledWith(
                expect.anything(),
                expect.objectContaining({
                    searchString: undefined,
                    isBot: true,
                    take: 20,
                }),
            );

            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("Available Bots"));
            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("TestBot"));
            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("HelperBot"));
        });

        it("should list bots in JSON format when specified", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockBots);
            (mockConfig.isJsonOutput as any).mockReturnValue(true);

            await program.parseAsync(["node", "test", "chat", "list-bots", "--format", "json"]);

            expect(output.json).toHaveBeenCalledWith(mockBots);
        });

        it("should search bots by name", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockBots);

            await program.parseAsync(["node", "test", "chat", "list-bots", "--search", "test"]);

            expect(mockClient.requestWithEndpoint).toHaveBeenCalledWith(
                expect.anything(),
                expect.objectContaining({
                    searchString: "test",
                    isBot: true,
                    take: 20,
                }),
            );
        });

        it("should handle API errors gracefully", async () => {
            (mockClient.requestWithEndpoint as any).mockRejectedValue(new Error("Network error"));

            await program.parseAsync(["node", "test", "chat", "list-bots"]);

            expect(output.error).toHaveBeenCalledWith(expect.stringContaining("Failed to list bots"));
            expect(processExitSpy).toHaveBeenCalledWith(1);
        });

        it("should display 'No bots found' when no bots are available", async () => {
            const emptyBotsResponse = {
                edges: [],
                pageInfo: {
                    hasNextPage: false,
                    hasPreviousPage: false,
                    __typename: "PageInfo",
                },
                __typename: "UserSearchResult",
            };
            (mockClient.requestWithEndpoint as any).mockResolvedValue(emptyBotsResponse);

            await program.parseAsync(["node", "test", "chat", "list-bots"]);

            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("No bots found"));
        });
    });

    describe("create chat command", () => {
        const mockChat: Chat = {
            id: "chat123",
            name: "New Chat",
            messages: [],
            participants: [{
                id: "part1",
                user: {
                    id: "bot1",
                    name: "TestBot",
                    isBot: true,
                    __typename: "User",
                } as User,
                __typename: "ChatParticipant",
            }],
            labels: [],
            translations: [],
            __typename: "Chat",
        };

        it("should create a new chat with a bot", async () => {
            (mockClient.isAuthenticated as any).mockReturnValue(true);
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockChat);

            await program.parseAsync(["node", "test", "chat", "create", "bot1"]);

            expect(mockClient.requestWithEndpoint).toHaveBeenCalledWith(
                expect.anything(),
                expect.objectContaining({
                    id: expect.any(String),
                    openToAnyoneWithInvite: false,
                    invitesCreate: expect.arrayContaining([
                        expect.objectContaining({
                            userConnect: "bot1",
                        }),
                    ]),
                }),
            );

            expect(output.success).toHaveBeenCalledWith("Chat created");
            expect(output.info).toHaveBeenCalledWith("  Chat ID: chat123");
        });

        it("should create a chat with custom name", async () => {
            (mockClient.isAuthenticated as any).mockReturnValue(true);
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockChat);

            await program.parseAsync(["node", "test", "chat", "create", "bot1", "--name", "My Chat"]);

            expect(mockClient.requestWithEndpoint).toHaveBeenCalledWith(
                expect.anything(),
                expect.objectContaining({
                    translationsCreate: expect.arrayContaining([
                        expect.objectContaining({
                            language: "en",
                            name: "My Chat",
                        }),
                    ]),
                }),
            );
        });

    });

    describe("list chats command", () => {
        const mockChats: ChatSearchResult = {
            edges: [
                {
                    cursor: "cursor1",
                    node: {
                        id: "chat1",
                        createdAt: "2025-01-12T10:00:00Z",
                        updatedAt: "2025-01-12T10:00:00Z",
                        messages: [],
                        participants: [],
                        participantsCount: 2,
                        labels: [],
                        translations: [
                            {
                                language: "en",
                                name: "Chat 1",
                                __typename: "ChatTranslation",
                            },
                        ],
                        __typename: "Chat",
                    } as Chat,
                },
                {
                    cursor: "cursor2",
                    node: {
                        id: "chat2",
                        createdAt: "2025-01-12T10:00:00Z",
                        updatedAt: "2025-01-12T10:00:00Z",
                        messages: [],
                        participants: [],
                        participantsCount: 1,
                        labels: [],
                        translations: [
                            {
                                language: "en",
                                name: "Chat 2",
                                __typename: "ChatTranslation",
                            },
                        ],
                        __typename: "Chat",
                    } as Chat,
                },
            ],
            pageInfo: {
                hasNextPage: false,
                hasPreviousPage: false,
                __typename: "PageInfo",
            },
            __typename: "ChatSearchResult",
        };

        it("should list user chats in table format", async () => {
            (mockClient.isAuthenticated as any).mockReturnValue(true);
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockChats);

            await program.parseAsync(["node", "test", "chat", "list"]);

            expect(mockClient.requestWithEndpoint).toHaveBeenCalledWith(
                expect.anything(),
                expect.objectContaining({
                    searchString: undefined,
                    take: 20,
                    visibility: "Own",
                }),
            );

            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("Your Chats"));
            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("Chat 1"));
            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("Chat 2"));
        });

        it("should list chats in JSON format", async () => {
            (mockClient.isAuthenticated as any).mockReturnValue(true);
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockChats);
            (mockConfig.isJsonOutput as any).mockReturnValue(true);

            await program.parseAsync(["node", "test", "chat", "list", "--format", "json"]);

            expect(output.json).toHaveBeenCalledWith(mockChats);
        });

        it("should handle empty chat list", async () => {
            (mockClient.isAuthenticated as any).mockReturnValue(true);
            (mockClient.requestWithEndpoint as any).mockResolvedValue({
                edges: [],
                pageInfo: {
                    hasNextPage: false,
                    hasPreviousPage: false,
                    __typename: "PageInfo",
                },
                __typename: "ChatSearchResult",
            });

            await program.parseAsync(["node", "test", "chat", "list"]);

            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("No chats found"));
        });
    });

    describe("show chat command", () => {
        const mockMessages: ChatMessageSearchResult = {
            edges: [
                {
                    cursor: "cursor1",
                    node: {
                        id: "msg1",
                        createdAt: new Date("2025-01-12T10:00:00Z"),
                        updatedAt: new Date("2025-01-12T10:00:00Z"),
                        messageFor: "chat1",
                        user: {
                            id: "user1",
                            name: "User",
                            isBot: false,
                            __typename: "User",
                        } as User,
                        text: "Hello bot!",
                        translations: [{
                            id: "trans1",
                            language: "en",
                            text: "Hello bot!",
                            __typename: "ChatMessageTranslation",
                        }],
                        __typename: "ChatMessage",
                    } as ChatMessage,
                },
                {
                    cursor: "cursor2",
                    node: {
                        id: "msg2",
                        createdAt: new Date("2025-01-12T10:01:00Z"),
                        updatedAt: new Date("2025-01-12T10:01:00Z"),
                        messageFor: "chat1",
                        user: {
                            id: "bot1",
                            name: "Bot",
                            isBot: true,
                            __typename: "User",
                        } as User,
                        text: "Hello! How can I help?",
                        translations: [{
                            id: "trans2",
                            language: "en",
                            text: "Hello! How can I help?",
                            __typename: "ChatMessageTranslation",
                        }],
                        __typename: "ChatMessage",
                    } as ChatMessage,
                },
            ],
            pageInfo: {
                hasNextPage: false,
                hasPreviousPage: false,
                __typename: "PageInfo",
            },
            __typename: "ChatMessageSearchResult",
        };

        it("should display chat history in conversation format", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockMessages);

            await program.parseAsync(["node", "test", "chat", "show", "chat1"]);

            expect(mockClient.requestWithEndpoint).toHaveBeenCalledWith(
                expect.anything(),
                expect.objectContaining({
                    chatId: "chat1",
                    take: 50,
                }),
            );

            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("Chat History"));
            expect(output.info).toHaveBeenCalledWith("─".repeat(300));
            expect(output.info).toHaveBeenCalledWith("Hello bot!");
            expect(output.info).toHaveBeenCalledWith("Hello! How can I help?");
        });

        it("should display chat history in JSON format", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockMessages);
            (mockConfig.isJsonOutput as any).mockReturnValue(true);

            await program.parseAsync(["node", "test", "chat", "show", "chat1", "--format", "json"]);

            expect(output.json).toHaveBeenCalledWith(mockMessages);
        });

        it("should handle empty chat", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue({
                edges: [],
                pageInfo: {
                    hasNextPage: false,
                    hasPreviousPage: false,
                    __typename: "PageInfo",
                },
                __typename: "ChatMessageSearchResult",
            });

            await program.parseAsync(["node", "test", "chat", "show", "chat1"]);

            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("No messages found"));
        });
    });

    describe("send message command", () => {
        it("should send a message to a chat", async () => {
            (mockClient.isAuthenticated as any).mockReturnValue(true);
            (mockClient.requestWithEndpoint as any).mockResolvedValue({
                id: "msg123",
                text: "Test message",
                translations: [{
                    language: "en",
                    text: "Test message",
                    __typename: "ChatMessageTranslation",
                }],
                __typename: "ChatMessage",
            });

            await program.parseAsync(["node", "test", "chat", "send", "chat1", "Test message"]);

            expect(mockClient.requestWithEndpoint).toHaveBeenCalledWith(
                expect.anything(),
                expect.objectContaining({
                    message: expect.objectContaining({
                        chatConnect: "chat1",
                        text: "Test message",
                        language: "en",
                    }),
                    model: "claude-3-5-sonnet-20241022",
                    task: expect.objectContaining({
                        goal: "Test message",
                    }),
                }),
            );

            expect(output.success).toHaveBeenCalledWith("Message sent");
        });

    });

    describe("interactive chat command", () => {
        it("should start an interactive chat session with existing chat", async () => {
            (mockClient.isAuthenticated as any).mockReturnValue(true);

            // Mock the InteractiveChatEngine
            const { InteractiveChatEngine } = await import("../utils/chatEngine.js");
            const mockStartSession = vi.fn().mockResolvedValue(undefined);
            (InteractiveChatEngine as any).mockImplementation(() => ({
                startSession: mockStartSession,
                stop: vi.fn(),
            }));

            await program.parseAsync(["node", "test", "chat", "interactive", "chat1"]);

            expect(InteractiveChatEngine).toHaveBeenCalledWith(
                mockClient,
                mockConfig,
                "chat1",
                expect.objectContaining({
                    model: "claude-3-5-sonnet-20241022",
                    showTools: false,
                    approveTools: false,
                    timeout: 300,
                }),
            );

            expect(mockStartSession).toHaveBeenCalled();
        });

        it("should create new chat in interactive mode when bot ID provided", async () => {
            (mockClient.isAuthenticated as any).mockReturnValue(true);

            // Mock the InteractiveChatEngine
            const { InteractiveChatEngine } = await import("../utils/chatEngine.js");
            const mockStartSession = vi.fn().mockResolvedValue(undefined);
            (InteractiveChatEngine as any).mockImplementation(() => ({
                startSession: mockStartSession,
                stop: vi.fn(),
            }));

            // Mock the chat creation
            (mockClient.requestWithEndpoint as any).mockResolvedValue({
                id: "new-chat-id",
                name: "New Chat",
                publicId: "public-id",
                translations: [],
            });

            await program.parseAsync(["node", "test", "chat", "interactive", "--bot", "bot1"]);

            expect(InteractiveChatEngine).toHaveBeenCalledWith(
                mockClient,
                mockConfig,
                "new-chat-id",
                expect.objectContaining({
                    model: "claude-3-5-sonnet-20241022",
                    showTools: false,
                    approveTools: false,
                    timeout: 300,
                }),
            );

            expect(mockStartSession).toHaveBeenCalled();
        });

        it("should handle interactive chat errors in JSON mode", async () => {
            (mockClient.isAuthenticated as any).mockReturnValue(true);
            (mockConfig.isJsonOutput as any).mockReturnValue(true);

            // Mock the InteractiveChatEngine to throw an error
            const { InteractiveChatEngine } = await import("../utils/chatEngine.js");
            const mockStartSession = vi.fn().mockRejectedValue(new Error("Connection failed"));
            (InteractiveChatEngine as any).mockImplementation(() => ({
                startSession: mockStartSession,
                stop: vi.fn(),
            }));

            await program.parseAsync(["node", "test", "chat", "interactive", "chat1"]);

            expect(output.json).toHaveBeenCalledWith({
                success: false,
                error: "Connection failed",
            });
            expect(processExitSpy).toHaveBeenCalledWith(1);
        });

        it("should handle interactive chat errors in regular mode", async () => {
            (mockClient.isAuthenticated as any).mockReturnValue(true);
            (mockConfig.isJsonOutput as any).mockReturnValue(false);

            // Mock the InteractiveChatEngine to throw an error
            const { InteractiveChatEngine } = await import("../utils/chatEngine.js");
            const mockStartSession = vi.fn().mockRejectedValue(new Error("Network timeout"));
            (InteractiveChatEngine as any).mockImplementation(() => ({
                startSession: mockStartSession,
                stop: vi.fn(),
            }));

            await program.parseAsync(["node", "test", "chat", "interactive", "chat1"]);

            expect(output.error).toHaveBeenCalledWith(
                "Interactive chat failed: Network timeout",
            );
            expect(processExitSpy).toHaveBeenCalledWith(1);
        });

        it("should handle chat creation error when creating new chat for interactive mode", async () => {
            (mockClient.isAuthenticated as any).mockReturnValue(true);
            (mockConfig.isJsonOutput as any).mockReturnValue(false);

            // Mock chat creation to fail
            (mockClient.requestWithEndpoint as any).mockRejectedValue(new Error("Failed to create chat"));

            await program.parseAsync(["node", "test", "chat", "interactive", "--bot", "bot1"]);

            expect(output.error).toHaveBeenCalledWith(
                "Failed to create chat: Failed to create chat",
            );
            expect(processExitSpy).toHaveBeenCalledWith(1);
        });
    });
});
