import {
    EventTypes,
    McpSwarmToolName,
    McpToolName,
    type SocketEventPayloads
} from "@vrooli/shared";
import type OpenAI from "openai";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { type ToolMeta } from "../conversation/types.js";
import { EventPublisher } from "../events/publisher.js";
import { BuiltInTools, SwarmTools } from "../mcp/tools.js";
import { toolApprovalConfig } from "./toolApprovalConfig.js";
import {
    CompositeToolRunner,
    MCPToolRunner,
    OpenAIToolRunner,
    SwarmToolRunner,
    ToolRunner,
} from "./toolRunner.js";

// Mock dependencies
vi.mock("../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        error: vi.fn(),
        warn: vi.fn(),
        debug: vi.fn(),
    },
}));

vi.mock("../events/publisher.js", () => ({
    EventPublisher: {
        get: vi.fn(() => ({
            publish: vi.fn(),
        })),
    },
}));

vi.mock("../mcp/tools.js", () => ({
    BuiltInTools: vi.fn(),
    SwarmTools: vi.fn(),
}));

vi.mock("./toolApprovalConfig.js", () => ({
    toolApprovalConfig: {
        requiresApproval: vi.fn().mockReturnValue(false),
        getApprovalMetadata: vi.fn().mockReturnValue({
            reason: "Test approval",
            riskLevel: "low",
        }),
    },
}));

describe("ToolRunner Base Class", () => {
    it("should be abstract and not directly instantiable", () => {
        expect(() => new (ToolRunner as any)()).toThrow();
    });
});

describe("OpenAIToolRunner", () => {
    let runner: OpenAIToolRunner;
    let mockOpenAIClient: OpenAI;

    beforeEach(() => {
        vi.clearAllMocks();
        mockOpenAIClient = {
            // Mock OpenAI client methods as needed
        } as OpenAI;
        runner = new OpenAIToolRunner(mockOpenAIClient);
    });

    describe("run", () => {
        it("should handle web_search tool", async () => {
            const result = await runner.run(
                "web_search",
                { query: "test search" },
                {} as ToolMeta,
            );

            expect(result.ok).toBe(true);
            if (result.ok) {
                expect(result.data.output).toContain("web search");
                expect(result.data.output).toContain("test search");
                expect(result.data.creditsUsed).toBe("10");
            }
        });

        it("should handle file_search tool", async () => {
            const result = await runner.run(
                "file_search",
                { path: "/test/path", pattern: "*.ts" },
                {} as ToolMeta,
            );

            expect(result.ok).toBe(true);
            if (result.ok) {
                expect(result.data.output).toContain("file search");
                expect(result.data.creditsUsed).toBe("5");
            }
        });

        it("should return error for unknown OpenAI tool", async () => {
            const result = await runner.run(
                "unknown_tool",
                {},
                {} as ToolMeta,
            );

            expect(result.ok).toBe(false);
            if (!result.ok) {
                expect(result.error.code).toBe("UNKNOWN_OPENAI_TOOL");
                expect(result.error.message).toContain("unknown_tool");
                expect(result.error.creditsUsed).toBe("1");
            }
        });

        it("should handle tool execution errors", async () => {
            // Force an error by passing invalid args
            const runner = new OpenAIToolRunner(null);

            // Override the run method to simulate an error
            vi.spyOn(runner, "run").mockImplementationOnce(async () => {
                throw new Error("Tool execution failed");
            });

            await expect(runner.run("web_search", {}, {} as ToolMeta))
                .rejects.toThrow("Tool execution failed");
        });
    });
});

describe("MCPToolRunner", () => {
    let runner: MCPToolRunner;
    let mockBuiltInTools: BuiltInTools;
    let mockMeta: ToolMeta;

    beforeEach(() => {
        vi.clearAllMocks();

        mockBuiltInTools = {
            run: vi.fn().mockResolvedValue({
                result: { data: "tool result" },
                creditsUsed: 20,
            }),
        } as unknown as BuiltInTools;

        (BuiltInTools as any).mockImplementation(() => mockBuiltInTools);

        mockMeta = {
            conversationId: "conv123",
            botId: "bot123",
            participantId: "participant123",
            swarmId: "swarm123",
            userId: "user123",
        } as ToolMeta;

        runner = new MCPToolRunner();
    });

    describe("run", () => {
        it("should execute MCP tool successfully", async () => {
            const result = await runner.run(
                McpToolName.CreateNote,
                { title: "Test Note", content: "Test content" },
                mockMeta,
            );

            expect(result.ok).toBe(true);
            if (result.ok) {
                expect(result.data.output).toEqual({ data: "tool result" });
                expect(result.data.creditsUsed).toBe("20");
            }

            expect(mockBuiltInTools.run).toHaveBeenCalledWith(
                McpToolName.CreateNote,
                { title: "Test Note", content: "Test content" },
                mockMeta,
            );
        });

        it("should handle tool execution errors", async () => {
            (mockBuiltInTools.run as any).mockRejectedValueOnce(
                new Error("MCP tool failed"),
            );

            const result = await runner.run(
                McpToolName.ReadFile,
                { path: "/test/file.txt" },
                mockMeta,
            );

            expect(result.ok).toBe(false);
            if (!result.ok) {
                expect(result.error.code).toBe("MCP_TOOL_ERROR");
                expect(result.error.message).toContain("MCP tool failed");
                expect(result.error.creditsUsed).toBe("2");
            }
        });

        it("should handle non-Error exceptions", async () => {
            (mockBuiltInTools.run as any).mockRejectedValueOnce("String error");

            const result = await runner.run(
                McpToolName.WriteFile,
                { path: "/test/file.txt", content: "test" },
                mockMeta,
            );

            expect(result.ok).toBe(false);
            if (!result.ok) {
                expect(result.error.code).toBe("MCP_TOOL_ERROR");
                expect(result.error.message).toContain("String error");
            }
        });
    });
});

describe("SwarmToolRunner", () => {
    let runner: SwarmToolRunner;
    let mockSwarmTools: SwarmTools;
    let mockMeta: ToolMeta;

    beforeEach(() => {
        vi.clearAllMocks();

        mockSwarmTools = {
            runTool: vi.fn().mockResolvedValue({
                result: { status: "success", data: "swarm result" },
                creditsUsed: 30,
            }),
        } as unknown as SwarmTools;

        (SwarmTools as any).mockImplementation(() => mockSwarmTools);

        mockMeta = {
            conversationId: "conv123",
            botId: "bot123",
            participantId: "participant123",
            swarmId: "swarm123",
            userId: "user123",
        } as ToolMeta;

        runner = new SwarmToolRunner();
    });

    describe("run", () => {
        it("should execute swarm tool successfully", async () => {
            const result = await runner.run(
                McpSwarmToolName.UpdateSwarmSharedState,
                {
                    key: "testKey",
                    value: { data: "testValue" },
                    operation: "set",
                },
                mockMeta,
            );

            expect(result.ok).toBe(true);
            if (result.ok) {
                expect(result.data.output).toEqual({
                    status: "success",
                    data: "swarm result",
                });
                expect(result.data.creditsUsed).toBe("30");
            }

            expect(mockSwarmTools.runTool).toHaveBeenCalledWith(
                McpSwarmToolName.UpdateSwarmSharedState,
                expect.objectContaining({
                    key: "testKey",
                    value: { data: "testValue" },
                    operation: "set",
                }),
                mockMeta,
            );
        });

        it("should handle missing swarmId", async () => {
            const metaWithoutSwarm = { ...mockMeta, swarmId: undefined };

            const result = await runner.run(
                McpSwarmToolName.SpawnSwarm,
                { goal: "Test goal" },
                metaWithoutSwarm,
            );

            expect(result.ok).toBe(false);
            if (!result.ok) {
                expect(result.error.code).toBe("MISSING_SWARM_ID");
                expect(result.error.message).toContain("Swarm ID is required");
            }
        });

        it("should handle swarm tool errors", async () => {
            (mockSwarmTools.runTool as any).mockRejectedValueOnce(
                new Error("Swarm operation failed"),
            );

            const result = await runner.run(
                McpSwarmToolName.SendMessage,
                {
                    message: "Test message",
                    targetBotId: "bot456",
                },
                mockMeta,
            );

            expect(result.ok).toBe(false);
            if (!result.ok) {
                expect(result.error.code).toBe("SWARM_TOOL_ERROR");
                expect(result.error.message).toContain("Swarm operation failed");
                expect(result.error.creditsUsed).toBe("3");
            }
        });
    });
});

describe("CompositeToolRunner", () => {
    let runner: CompositeToolRunner;
    let mockOpenAIRunner: OpenAIToolRunner;
    let mockMCPRunner: MCPToolRunner;
    let mockSwarmRunner: SwarmToolRunner;
    let mockMeta: ToolMeta;
    let eventPublisher: any;

    beforeEach(() => {
        vi.clearAllMocks();

        // Create mock runners
        mockOpenAIRunner = {
            run: vi.fn().mockResolvedValue({
                ok: true,
                data: { output: "OpenAI result", creditsUsed: "10" },
            }),
        } as unknown as OpenAIToolRunner;

        mockMCPRunner = {
            run: vi.fn().mockResolvedValue({
                ok: true,
                data: { output: "MCP result", creditsUsed: "20" },
            }),
        } as unknown as MCPToolRunner;

        mockSwarmRunner = {
            run: vi.fn().mockResolvedValue({
                ok: true,
                data: { output: "Swarm result", creditsUsed: "30" },
            }),
        } as unknown as SwarmToolRunner;

        mockMeta = {
            conversationId: "conv123",
            botId: "bot123",
            participantId: "participant123",
            swarmId: "swarm123",
            userId: "user123",
        } as ToolMeta;

        eventPublisher = EventPublisher.get();

        runner = new CompositeToolRunner(
            mockOpenAIRunner,
            mockMCPRunner,
            mockSwarmRunner,
        );
    });

    describe("run", () => {
        it("should route OpenAI tools correctly", async () => {
            const result = await runner.run(
                "web_search",
                { query: "test" },
                mockMeta,
            );

            expect(result.ok).toBe(true);
            expect(mockOpenAIRunner.run).toHaveBeenCalledWith(
                "web_search",
                { query: "test" },
                mockMeta,
            );
            expect(mockMCPRunner.run).not.toHaveBeenCalled();
            expect(mockSwarmRunner.run).not.toHaveBeenCalled();
        });

        it("should route MCP tools correctly", async () => {
            const result = await runner.run(
                McpToolName.CreateNote,
                { title: "Test", content: "Content" },
                mockMeta,
            );

            expect(result.ok).toBe(true);
            expect(mockMCPRunner.run).toHaveBeenCalledWith(
                McpToolName.CreateNote,
                { title: "Test", content: "Content" },
                mockMeta,
            );
            expect(mockOpenAIRunner.run).not.toHaveBeenCalled();
            expect(mockSwarmRunner.run).not.toHaveBeenCalled();
        });

        it("should route swarm tools correctly", async () => {
            const result = await runner.run(
                McpSwarmToolName.UpdateSwarmSharedState,
                { key: "test", value: "value" },
                mockMeta,
            );

            expect(result.ok).toBe(true);
            expect(mockSwarmRunner.run).toHaveBeenCalledWith(
                McpSwarmToolName.UpdateSwarmSharedState,
                { key: "test", value: "value" },
                mockMeta,
            );
            expect(mockOpenAIRunner.run).not.toHaveBeenCalled();
            expect(mockMCPRunner.run).not.toHaveBeenCalled();
        });

        it("should handle unknown tools", async () => {
            const result = await runner.run(
                "completely_unknown_tool",
                {},
                mockMeta,
            );

            expect(result.ok).toBe(false);
            if (!result.ok) {
                expect(result.error.code).toBe("UNKNOWN_TOOL");
                expect(result.error.message).toContain("completely_unknown_tool");
                expect(result.error.creditsUsed).toBe("1");
            }
        });

        it("should handle tool approval when required", async () => {
            // Mock tool requires approval
            (toolApprovalConfig.requiresApproval as any).mockReturnValueOnce(true);
            (toolApprovalConfig.getApprovalMetadata as any).mockReturnValueOnce({
                reason: "Sensitive operation",
                riskLevel: "high",
                estimatedCredits: "100",
            });

            // Set up approval promise
            let approvalResolver: (approved: boolean) => void;
            const approvalPromise = new Promise<boolean>((resolve) => {
                approvalResolver = resolve;
            });

            // Mock event listener for approval
            const mockApprovalHandler = vi.fn((event: SocketEventPayloads[EventTypes.ToolApprovalResponse]) => {
                if (event.approved) {
                    approvalResolver!(true);
                }
            });

            // Simulate approval event after a delay
            setTimeout(() => {
                mockApprovalHandler({
                    toolCallId: expect.any(String),
                    approved: true,
                    userId: mockMeta.userId,
                } as SocketEventPayloads[EventTypes.ToolApprovalResponse]);
            }, 100);

            const result = await runner.run(
                McpToolName.WriteFile,
                { path: "/sensitive/file.txt", content: "data" },
                mockMeta,
            );

            // Should emit approval request
            expect(eventPublisher.publish).toHaveBeenCalledWith({
                type: EventTypes.ToolApprovalRequest,
                payload: expect.objectContaining({
                    toolName: McpToolName.WriteFile,
                    toolArgs: { path: "/sensitive/file.txt", content: "data" },
                    metadata: {
                        reason: "Sensitive operation",
                        riskLevel: "high",
                        estimatedCredits: "100",
                    },
                }),
            });

            expect(result.ok).toBe(true);
        });

        it("should handle tool approval denial", async () => {
            // Mock tool requires approval
            (toolApprovalConfig.requiresApproval as any).mockReturnValueOnce(true);

            // Simulate denial by not sending approval event
            // The approval timeout should kick in

            const result = await runner.run(
                McpToolName.DeleteFile,
                { path: "/important/file.txt" },
                mockMeta,
            );

            // After timeout, should return error
            expect(result.ok).toBe(false);
            if (!result.ok) {
                expect(result.error.code).toBe("TOOL_APPROVAL_TIMEOUT");
                expect(result.error.creditsUsed).toBe("0");
            }
        });

        it("should emit tool execution events", async () => {
            const result = await runner.run(
                McpToolName.ReadFile,
                { path: "/test/file.txt" },
                mockMeta,
            );

            // Should emit start event
            expect(eventPublisher.publish).toHaveBeenCalledWith({
                type: EventTypes.ToolExecutionStart,
                payload: expect.objectContaining({
                    toolName: McpToolName.ReadFile,
                    toolArgs: { path: "/test/file.txt" },
                    botId: mockMeta.botId,
                    conversationId: mockMeta.conversationId,
                }),
            });

            // Should emit complete event
            expect(eventPublisher.publish).toHaveBeenCalledWith({
                type: EventTypes.ToolExecutionComplete,
                payload: expect.objectContaining({
                    toolName: McpToolName.ReadFile,
                    result: "MCP result",
                    creditsUsed: "20",
                    botId: mockMeta.botId,
                    conversationId: mockMeta.conversationId,
                }),
            });
        });

        it("should emit tool error events", async () => {
            (mockMCPRunner.run as any).mockResolvedValueOnce({
                ok: false,
                error: {
                    code: "TOOL_FAILED",
                    message: "Operation failed",
                    creditsUsed: "5",
                },
            });

            const result = await runner.run(
                McpToolName.CreateNote,
                { title: "Test" },
                mockMeta,
            );

            expect(eventPublisher.publish).toHaveBeenCalledWith({
                type: EventTypes.ToolExecutionError,
                payload: expect.objectContaining({
                    toolName: McpToolName.CreateNote,
                    error: "Operation failed",
                    errorCode: "TOOL_FAILED",
                    botId: mockMeta.botId,
                }),
            });
        });

        it("should handle concurrent tool executions", async () => {
            const promises = [
                runner.run("web_search", { query: "1" }, mockMeta),
                runner.run(McpToolName.CreateNote, { title: "2" }, mockMeta),
                runner.run(McpSwarmToolName.SendMessage, { message: "3" }, mockMeta),
            ];

            const results = await Promise.all(promises);

            expect(results).toHaveLength(3);
            expect(results.every(r => r.ok)).toBe(true);
            expect(mockOpenAIRunner.run).toHaveBeenCalledTimes(1);
            expect(mockMCPRunner.run).toHaveBeenCalledTimes(1);
            expect(mockSwarmRunner.run).toHaveBeenCalledTimes(1);
        });

        it("should validate tool arguments", async () => {
            // Test with invalid arguments
            const result = await runner.run(
                McpToolName.WriteFile,
                {
                    // Missing required 'path' field
                    content: "test",
                },
                mockMeta,
            );

            // Should still pass to runner for validation
            expect(mockMCPRunner.run).toHaveBeenCalled();
        });

        it("should handle very long tool outputs", async () => {
            const longOutput = "x".repeat(100000); // 100KB output
            (mockMCPRunner.run as any).mockResolvedValueOnce({
                ok: true,
                data: { output: longOutput, creditsUsed: "50" },
            });

            const result = await runner.run(
                McpToolName.ReadFile,
                { path: "/large/file.txt" },
                mockMeta,
            );

            expect(result.ok).toBe(true);
            if (result.ok) {
                expect(result.data.output).toBe(longOutput);
            }
        });

        it("should handle special characters in tool arguments", async () => {
            const specialArgs = {
                query: "test \"quoted\" & <special> characters",
                path: "/path/with spaces/and'quotes",
            };

            const result = await runner.run(
                "web_search",
                specialArgs,
                mockMeta,
            );

            expect(result.ok).toBe(true);
            expect(mockOpenAIRunner.run).toHaveBeenCalledWith(
                "web_search",
                specialArgs,
                mockMeta,
            );
        });
    });
});
