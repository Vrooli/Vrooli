// AI_CHECK: TEST_COVERAGE=23 | LAST: 2025-07-13
import inquirer from "inquirer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { type ApiClient } from "./client.js";
import { type ConfigManager } from "./config.js";
import { ToolApprovalHandler } from "./toolApproval.js";

// Mock dependencies
vi.mock("inquirer", () => ({
    default: {
        prompt: vi.fn(),
    },
}));

vi.mock("chalk", () => {
    const createChalkMock = (text: string) => text;
    const chalkMock = Object.assign(createChalkMock, {
        bold: Object.assign(createChalkMock, {
            yellow: createChalkMock,
            blue: createChalkMock,
            red: createChalkMock,
            green: createChalkMock,
            cyan: createChalkMock,
        }),
        cyan: createChalkMock,
        yellow: createChalkMock,
        green: createChalkMock,
        red: createChalkMock,
        gray: createChalkMock,
        magenta: createChalkMock,
        dim: createChalkMock,
        blue: createChalkMock,
    });

    return {
        default: chalkMock,
    };
});

describe("ToolApprovalHandler", () => {
    let handler: ToolApprovalHandler;
    let mockClient: ApiClient;
    let mockConfig: ConfigManager;
    let mockStatusCallback: any;
    let consoleLogSpy: any;

    beforeEach(() => {
        mockClient = {
            post: vi.fn(),
        } as unknown as ApiClient;

        mockConfig = {
            isJsonOutput: vi.fn().mockReturnValue(false),
        } as unknown as ConfigManager;

        mockStatusCallback = vi.fn();

        handler = new ToolApprovalHandler(mockClient, mockConfig, mockStatusCallback);

        consoleLogSpy = vi.spyOn(console, "log").mockImplementation(() => undefined);
    });

    afterEach(() => {
        consoleLogSpy.mockRestore();
        vi.clearAllMocks();
    });

    describe("handleApprovalRequest", () => {
        const validApprovalData = {
            pendingId: "pending123",
            toolCallId: "tool123",
            toolName: "search",
            toolArguments: { query: "test" },
            callerBotId: "bot123",
            callerBotName: "TestBot",
            approvalTimeoutAt: Date.now() + 30000,
            estimatedCost: "$0.01",
        };

        it("should handle valid approval request and prompt user", async () => {
            (inquirer.prompt as any).mockResolvedValue({ action: "approve" });

            await handler.handleApprovalRequest(validApprovalData, "chat123");

            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("🔧 Tool Execution Request"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Tool: search"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Bot: TestBot"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Estimated Cost: $0.01 credits"));

            expect(inquirer.prompt).toHaveBeenCalledWith([
                expect.objectContaining({
                    type: "list",
                    name: "action",
                    message: expect.stringContaining("Do you want to allow this tool execution?"),
                    choices: expect.arrayContaining([
                        expect.objectContaining({ value: "approve" }),
                        expect.objectContaining({ value: "deny" }),
                    ]),
                }),
            ]);

            expect(mockClient.post).toHaveBeenCalledWith("/task/respondToToolApproval", {
                conversationId: "chat123",
                pendingId: "pending123",
                approved: true,
            });
        });

        it("should handle rejection from user", async () => {
            (inquirer.prompt as any).mockResolvedValue({ action: "deny" });

            await handler.handleApprovalRequest(validApprovalData, "chat123");

            expect(mockClient.post).toHaveBeenCalledWith("/task/respondToToolApproval", {
                conversationId: "chat123",
                pendingId: "pending123",
                approved: false,
            });

            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("✗ Tool execution denied:"));
        });

        it("should handle approval with timeout warning", async () => {
            const dataWithShortTimeout = {
                ...validApprovalData,
                approvalTimeoutAt: Date.now() + 5000, // 5 seconds
            };

            (inquirer.prompt as any).mockResolvedValue({ action: "approve" });

            await handler.handleApprovalRequest(dataWithShortTimeout, "chat123");

            expect(consoleLogSpy).toHaveBeenCalledWith(
                expect.stringContaining("Timeout: 5 seconds"),
            );
        });

        it("should reject invalid approval data", async () => {
            const invalidData = {
                pendingId: "pending123",
                // Missing required fields
            };

            await expect(handler.handleApprovalRequest(invalidData, "chat123"))
                .rejects.toThrow("Invalid tool approval data");
        });

        it("should display tool arguments properly", async () => {
            const dataWithComplexArgs = {
                ...validApprovalData,
                toolArguments: {
                    query: "test search",
                    limit: 10,
                    filters: { category: "docs" },
                },
            };

            (inquirer.prompt as any).mockResolvedValue({ action: "approve" });

            await handler.handleApprovalRequest(dataWithComplexArgs, "chat123");

            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Arguments:"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("query: \"test search\""));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("limit: 10"));
        });

        it("should handle missing optional fields", async () => {
            const minimalData = {
                pendingId: "pending123",
                toolCallId: "tool123",
                toolName: "search",
                callerBotId: "bot123",
            };

            (inquirer.prompt as any).mockResolvedValue({ action: "approve" });

            await handler.handleApprovalRequest(minimalData, "chat123");

            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Bot: bot123"));
            // Should not display cost when not provided
            expect(consoleLogSpy).not.toHaveBeenCalledWith(expect.stringContaining("Estimated Cost:"));
        });

        it("should queue multiple approval requests", async () => {
            vi.useFakeTimers();

            // First approval - block on prompt
            let resolveFirst: any;
            const firstPromise = new Promise(resolve => { resolveFirst = resolve; });
            (inquirer.prompt as any).mockImplementationOnce(() => firstPromise);

            // Start first approval
            const firstApproval = handler.handleApprovalRequest(validApprovalData, "chat123");

            // Try to add second approval while first is pending
            const secondData = {
                ...validApprovalData,
                pendingId: "pending456",
                toolCallId: "tool456",
            };

            (inquirer.prompt as any).mockResolvedValueOnce({ action: "approve" });
            const secondApproval = handler.handleApprovalRequest(secondData, "chat123");

            // Resolve first approval
            resolveFirst({ action: "approve" });
            await firstApproval;

            // Advance timers to process queued approval
            await vi.advanceTimersByTimeAsync(100);

            // Second should complete after timer fires
            await secondApproval;

            expect(mockClient.post).toHaveBeenCalledTimes(2);

            vi.useRealTimers();
        });
    });

    describe("handleExecutionStart", () => {
        it("should track execution start and notify callback", () => {
            const data = {
                toolCallId: "tool123",
                toolName: "search",
                callerBotId: "bot123",
            };

            handler.handleExecutionStart(data);

            expect(mockStatusCallback).toHaveBeenCalledWith(
                expect.objectContaining({
                    toolCallId: "tool123",
                    toolName: "search",
                    callerBotId: "bot123",
                    status: "executing",
                    startTime: expect.any(Date),
                }),
            );

            // handleExecutionStart doesn't log, it just tracks execution
        });

        it("should reject invalid execution start data", () => {
            const invalidData = { toolCallId: 123 }; // Wrong type

            expect(() => handler.handleExecutionStart(invalidData))
                .toThrow("Invalid tool execution start data");
        });
    });

    describe("handleExecutionComplete", () => {
        it("should handle successful execution completion", () => {
            // First start the execution
            handler.handleExecutionStart({
                toolCallId: "tool123",
                toolName: "search",
                callerBotId: "bot123",
            });

            // Then complete it
            handler.handleExecutionComplete({
                toolCallId: "tool123",
                result: { data: "search results" },
            });

            expect(mockStatusCallback).toHaveBeenLastCalledWith(
                expect.objectContaining({
                    toolCallId: "tool123",
                    status: "completed",
                    endTime: expect.any(Date),
                    result: { data: "search results" },
                }),
            );

            // handleExecutionComplete doesn't log, it just updates status
        });

        it("should handle execution failure", () => {
            // First start the execution
            handler.handleExecutionStart({
                toolCallId: "tool123",
                toolName: "search",
                callerBotId: "bot123",
            });

            // Then fail it
            handler.handleExecutionComplete({
                toolCallId: "tool123",
                error: "Network error",
            });

            expect(mockStatusCallback).toHaveBeenLastCalledWith(
                expect.objectContaining({
                    toolCallId: "tool123",
                    status: "failed",
                    error: "Network error",
                }),
            );

            // handleExecutionComplete doesn't log, it just updates status
        });

        it("should handle completion for unknown tool", () => {
            handler.handleExecutionComplete({
                toolCallId: "unknown123",
                result: { data: "result" },
            });

            expect(consoleLogSpy).toHaveBeenCalledWith(
                expect.stringContaining("Received completion for unknown tool: unknown123"),
            );
        });

        it("should reject invalid completion data", () => {
            const invalidData = { notAToolCallId: "test" };

            expect(() => handler.handleExecutionComplete(invalidData))
                .toThrow("Invalid tool execution complete data");
        });
    });

    describe("getExecutingTools", () => {
        it("should return list of executing tools", () => {
            // Start multiple tools
            handler.handleExecutionStart({
                toolCallId: "tool1",
                toolName: "search",
                callerBotId: "bot123",
            });

            handler.handleExecutionStart({
                toolCallId: "tool2",
                toolName: "calculate",
                callerBotId: "bot123",
            });

            const executing = handler.getExecutingTools();

            expect(executing).toHaveLength(2);
            expect(executing).toEqual(
                expect.arrayContaining([
                    expect.objectContaining({ toolCallId: "tool1", toolName: "search" }),
                    expect.objectContaining({ toolCallId: "tool2", toolName: "calculate" }),
                ]),
            );
        });

        it("should not include completed tools", () => {
            // Start and complete a tool
            handler.handleExecutionStart({
                toolCallId: "tool1",
                toolName: "search",
                callerBotId: "bot123",
            });

            handler.handleExecutionComplete({
                toolCallId: "tool1",
                result: { data: "result" },
            });

            const executing = handler.getExecutingTools();

            expect(executing).toHaveLength(0);
        });
    });

    describe("displayExecutingTools", () => {
        it("should display executing tools", () => {
            handler.handleExecutionStart({
                toolCallId: "tool1",
                toolName: "search",
                callerBotId: "bot123",
            });

            handler.handleExecutionStart({
                toolCallId: "tool2",
                toolName: "calculate",
                callerBotId: "bot123",
            });

            handler.displayExecutingTools();

            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("🔧 Currently Executing Tools:"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("search"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("calculate"));
        });

        it("should show message when no tools executing", () => {
            handler.displayExecutingTools();

            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("No tools currently executing"));
        });
    });

    describe("edge cases", () => {
        it("should handle prompt cancellation", async () => {
            const error = new Error("User cancelled");
            (error as any).isTtyError = true;
            (inquirer.prompt as any).mockRejectedValue(error);

            await handler.handleApprovalRequest({
                pendingId: "pending123",
                toolCallId: "tool123",
                toolName: "search",
                callerBotId: "bot123",
            }, "chat123");

            expect(consoleLogSpy).toHaveBeenCalledWith("Error handling tool approval: User cancelled");
        });

        it("should handle API errors gracefully", async () => {
            (inquirer.prompt as any).mockResolvedValue({ action: "approve" });
            (mockClient.post as any).mockRejectedValue(new Error("API error"));

            await handler.handleApprovalRequest({
                pendingId: "pending123",
                toolCallId: "tool123",
                toolName: "search",
                callerBotId: "bot123",
            }, "chat123");

            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Failed to send approval response"));
        });

        it("should format timeout properly for different durations", async () => {
            consoleLogSpy.mockClear();
            (inquirer.prompt as any).mockResolvedValue({ action: "approve" });

            await handler.handleApprovalRequest({
                pendingId: "pending123",
                toolCallId: "tool123",
                toolName: "search",
                callerBotId: "bot123",
                approvalTimeoutAt: Date.now() + 30000, // 30 seconds from now
            }, "chat123");

            expect(consoleLogSpy).toHaveBeenCalledWith(
                expect.stringContaining("Timeout: 30 seconds"),
            );
        });
    });

    describe("getToolStatus", () => {
        it("should return status of pending and executing tools", () => {
            // Add a pending approval directly to simulate a request that hasn't been processed yet
            const pendingRequest = {
                pendingId: "pending1",
                toolCallId: "tool1",
                toolName: "search",
                toolArguments: { query: "test" },
                callerBotId: "bot123",
                conversationId: "test-conversation",
            };

            // Access private field to add directly without triggering the approval flow
            (handler as any).pendingApprovals.set(pendingRequest.pendingId, pendingRequest);

            // Add an executing tool
            handler.handleExecutionStart({
                toolCallId: "tool2",
                toolName: "calculate",
                callerBotId: "bot123",
            });

            const status = handler.getToolStatus();

            expect(status.pending).toHaveLength(1);
            expect(status.executing).toHaveLength(1);
            expect(status.pending[0].toolName).toBe("search");
            expect(status.executing[0].toolName).toBe("calculate");
        });

        it("should return empty arrays when no tools are pending or executing", () => {
            const status = handler.getToolStatus();

            expect(status.pending).toHaveLength(0);
            expect(status.executing).toHaveLength(0);
        });
    });

    describe("cleanup", () => {
        it("should clear all pending and executing tools", () => {
            // Add some data directly to simulate state
            const pendingRequest = {
                pendingId: "pending1",
                toolCallId: "tool1",
                toolName: "search",
                toolArguments: { query: "test" },
                callerBotId: "bot123",
                conversationId: "test-conversation",
            };

            (handler as any).pendingApprovals.set(pendingRequest.pendingId, pendingRequest);

            handler.handleExecutionStart({
                toolCallId: "tool2",
                toolName: "calculate",
                callerBotId: "bot123",
            });

            // Verify data exists
            expect(handler.getToolStatus().pending).toHaveLength(1);
            expect(handler.getToolStatus().executing).toHaveLength(1);

            // Cleanup
            handler.cleanup();

            // Verify data is cleared
            const status = handler.getToolStatus();
            expect(status.pending).toHaveLength(0);
            expect(status.executing).toHaveLength(0);
        });
    });
});
