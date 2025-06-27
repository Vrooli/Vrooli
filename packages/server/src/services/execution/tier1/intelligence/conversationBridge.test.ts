/**
 * Tests for ConversationBridge integration with existing conversation infrastructure
 */

import { type SessionUser, generatePK } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { type Logger } from "winston";
import { mockLogger } from "../../../../__test/logger.mock.js";
import { ConversationBridge, type ConversationBridgeConfig } from "./conversationBridge.js";

// Mock the conversation service
const mockCompletionService = {
    getConversationState: vi.fn(),
    getReasoningEngine: vi.fn(),
    getToolRegistry: vi.fn(),
    generateSystemMessageForBot: vi.fn(),
    updateConversationConfig: vi.fn(),
};

const mockReasoningEngine = {
    runLoop: vi.fn(),
    toolRunner: {
        run: vi.fn(),
    },
};

const mockToolRegistry = {
    getToolsForConversation: vi.fn(),
    getToolByName: vi.fn(),
};

// Mock the import to return our mocked services
vi.mock("../../../../../conversation/responseEngine.js", () => ({
    completionService: mockCompletionService,
}));

describe("ConversationBridge", () => {
    let bridge: ConversationBridge;
    let logger: Logger;
    let config: ConversationBridgeConfig;
    let mockUser: SessionUser;

    beforeEach(() => {
        logger = mockLogger as unknown as Logger;
        bridge = new ConversationBridge(logger);

        mockUser = {
            id: "user_123",
            name: "Test User",
            email: "test@example.com",
            languages: ["en"],
            isAdmin: false,
        } as SessionUser;

        config = {
            conversationId: "conv_123",
            sessionUser: mockUser,
        };

        // Reset mocks
        vi.clearAllMocks();

        // Setup default mock returns
        mockCompletionService.getReasoningEngine.mockReturnValue(mockReasoningEngine);
        mockCompletionService.getToolRegistry.mockReturnValue(mockToolRegistry);
        mockToolRegistry.getToolsForConversation.mockResolvedValue([]);
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("generateResponse", () => {
        it("should generate a response using the conversation infrastructure", async () => {
            // Setup mocks
            const mockConversationState = {
                config: {
                    swarmLeader: "bot_123",
                    swarmTask: "Help the user",
                    limits: {
                        toolCallsPerResponse: 5,
                        creditsPerResponse: 500,
                    },
                    model: "gpt-4",
                },
                bots: [{
                    config: { name: "Test Bot" },
                }],
                teamConfig: undefined,
            };

            const mockFinalMessage = {
                id: generatePK().toString(),
                text: "Hello! I can help you with that.",
                createdAt: new Date(),
                config: { role: "assistant" },
                language: "en",
                user: { id: "bot_123" },
                parent: null,
            };

            const mockResponseStats = {
                toolCallsExecuted: 0,
                creditsUsed: BigInt(25),
            };

            mockCompletionService.getConversationState.mockResolvedValue(mockConversationState);
            mockCompletionService.generateSystemMessageForBot.mockResolvedValue("You are a helpful AI assistant.");
            mockReasoningEngine.runLoop.mockResolvedValue({
                finalMessage: mockFinalMessage,
                responseStats: mockResponseStats,
            });

            // Execute
            const result = await bridge.generateResponse(config, "Hello, can you help me?");

            // Verify
            expect(result).toHaveLength(1);
            expect(result[0]).toEqual(mockFinalMessage);
            expect(mockCompletionService.getConversationState).toHaveBeenCalledWith("conv_123");
            expect(mockReasoningEngine.runLoop).toHaveBeenCalledWith(
                { text: "Hello, can you help me?" },
                "You are a helpful AI assistant.",
                [],
                expect.objectContaining({
                    id: "bot_123",
                    name: "Tier1 Coordinator",
                    meta: expect.objectContaining({
                        role: "coordinator",
                    }),
                }),
                "user_123",
                mockConversationState.config,
                5,
                BigInt(500),
                mockUser,
                "conv_123",
                "gpt-4",
            );
        });

        it("should use custom system message when provided", async () => {
            const mockConversationState = {
                config: {
                    swarmLeader: "bot_123",
                    swarmTask: "Help the user",
                    limits: {},
                },
                bots: [{ config: {} }],
            };

            const mockFinalMessage = {
                id: generatePK().toString(),
                text: "Custom response",
                createdAt: new Date(),
                config: { role: "assistant" },
                language: "en",
                user: { id: "bot_123" },
                parent: null,
            };

            mockCompletionService.getConversationState.mockResolvedValue(mockConversationState);
            mockReasoningEngine.runLoop.mockResolvedValue({
                finalMessage: mockFinalMessage,
                responseStats: { toolCallsExecuted: 0, creditsUsed: BigInt(10) },
            });

            const customSystemMessage = "You are a specialized assistant for testing.";

            await bridge.generateResponse(config, "Test prompt", customSystemMessage);

            expect(mockReasoningEngine.runLoop).toHaveBeenCalledWith(
                { text: "Test prompt" },
                customSystemMessage,
                [],
                expect.any(Object),
                "user_123",
                mockConversationState.config,
                10, // default toolCallsPerResponse
                BigInt(1000), // default creditsPerResponse
                mockUser,
                "conv_123",
                "gpt-4", // default model
            );

            // Should not call generateSystemMessageForBot when custom system message provided
            expect(mockCompletionService.generateSystemMessageForBot).not.toHaveBeenCalled();
        });

        it("should throw error when conversation not found", async () => {
            mockCompletionService.getConversationState.mockResolvedValue(null);

            await expect(bridge.generateResponse(config, "Hello")).rejects.toThrow(
                "Conversation not found: conv_123",
            );
        });
    });

    describe("executeTool", () => {
        it("should execute a tool successfully", async () => {
            const mockToolResult = {
                ok: true,
                data: {
                    output: "Tool executed successfully",
                    creditsUsed: 10,
                },
            };

            mockReasoningEngine.toolRunner.run.mockResolvedValue(mockToolResult);

            const result = await bridge.executeTool(config, "testTool", { param: "value" });

            expect(result).toBe("Tool executed successfully");
            expect(mockReasoningEngine.toolRunner.run).toHaveBeenCalledWith(
                "testTool",
                { param: "value" },
                {
                    conversationId: "conv_123",
                    callerBotId: "tier1-coordinator",
                    sessionUser: mockUser,
                },
            );
        });

        it("should throw error when tool execution fails", async () => {
            const mockToolResult = {
                ok: false,
                error: {
                    message: "Tool failed to execute",
                    code: "TOOL_ERROR",
                },
            };

            mockReasoningEngine.toolRunner.run.mockResolvedValue(mockToolResult);

            await expect(bridge.executeTool(config, "failingTool", {})).rejects.toThrow(
                "Tool execution failed: Tool failed to execute",
            );
        });
    });

    describe("handleToolApproval", () => {
        it("should approve a pending tool call", async () => {
            const mockConversationState = {
                config: {
                    pendingToolCalls: [
                        {
                            id: "call_123",
                            toolName: "testTool",
                            status: "pending",
                            args: { test: true },
                        },
                    ],
                },
            };

            mockCompletionService.getConversationState.mockResolvedValue(mockConversationState);

            await bridge.handleToolApproval(config, "call_123", true);

            expect(mockCompletionService.updateConversationConfig).toHaveBeenCalledWith(
                "conv_123",
                expect.objectContaining({
                    pendingToolCalls: [
                        expect.objectContaining({
                            id: "call_123",
                            status: "approved",
                        }),
                    ],
                }),
            );
        });

        it("should reject a pending tool call with reason", async () => {
            const mockConversationState = {
                config: {
                    pendingToolCalls: [
                        {
                            id: "call_456",
                            toolName: "dangerousTool",
                            status: "pending",
                            args: { risk: "high" },
                        },
                    ],
                },
            };

            mockCompletionService.getConversationState.mockResolvedValue(mockConversationState);

            await bridge.handleToolApproval(config, "call_456", false, "Security risk detected");

            expect(mockCompletionService.updateConversationConfig).toHaveBeenCalledWith(
                "conv_123",
                expect.objectContaining({
                    pendingToolCalls: [
                        expect.objectContaining({
                            id: "call_456",
                            status: "rejected",
                            rejectionReason: "Security risk detected",
                        }),
                    ],
                }),
            );
        });

        it("should handle non-existent tool call gracefully", async () => {
            const mockConversationState = {
                config: {
                    pendingToolCalls: [],
                },
            };

            mockCompletionService.getConversationState.mockResolvedValue(mockConversationState);

            // Should not throw error
            await bridge.handleToolApproval(config, "nonexistent_call", true);

            // Should not update config since tool call not found
            expect(mockCompletionService.updateConversationConfig).not.toHaveBeenCalled();
        });
    });

    describe("createConversationBridge factory", () => {
        it("should create a ConversationBridge instance", async () => {
            const { createConversationBridge } = await import("./conversationBridge.js");
            const testLogger = mockLogger as unknown as Logger;
            const createdBridge = createConversationBridge(testLogger);

            expect(createdBridge).toBeInstanceOf(ConversationBridge);
        });
    });
});
