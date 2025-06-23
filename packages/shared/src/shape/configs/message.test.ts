import { describe, it, expect, vi } from "vitest";
import { MessageConfig, type MessageConfigObject, type ToolFunctionCall, type ChatMessageRunConfig } from "./message.js";
import { messageConfigFixtures } from "../../__test/fixtures/config/messageConfigFixtures.js";
import { runComprehensiveConfigTests } from "./__test/configTestUtils.js";

describe("MessageConfig", () => {
    // Standardized config tests using fixtures
    runComprehensiveConfigTests(
        MessageConfig,
        messageConfigFixtures,
        "message",
    );

    // Message-specific business logic tests
    describe("message-specific functionality", () => {
        const mockLogger = {
            trace: vi.fn(),
            debug: vi.fn(),
            info: vi.fn(),
            warn: vi.fn(),
            error: vi.fn(),
        };

        describe("parse with fallbacks", () => {
            it("should parse a MessageConfig with fallbacks enabled (default)", () => {
                const config: MessageConfigObject = {
                    __version: "1.0",
                    resources: [],
                    eventTopic: "test-event",
                };

                const messageConfig = MessageConfig.parse({ config }, mockLogger);

                expect(messageConfig.contextHints).toEqual([]);
                expect(messageConfig.eventTopic).toBe("test-event");
                expect(messageConfig.respondingBots).toEqual([]);
                expect(messageConfig.role).toBe("user");
                expect(messageConfig.turnId).toBeNull();
                expect(messageConfig.toolCalls).toEqual([]);
                expect(messageConfig.runs).toEqual([]);
            });

            it("should parse a MessageConfig without fallbacks", () => {
                const config: MessageConfigObject = {
                    __version: "1.0",
                    resources: [],
                    eventTopic: "test-event",
                };

                const messageConfig = MessageConfig.parse({ config }, mockLogger, { useFallbacks: false });

                expect(messageConfig.contextHints).toBeUndefined();
                expect(messageConfig.eventTopic).toBe("test-event");
                expect(messageConfig.respondingBots).toBeUndefined();
                expect(messageConfig.role).toBeUndefined();
                expect(messageConfig.turnId).toBeUndefined();
                expect(messageConfig.toolCalls).toBeUndefined();
                expect(messageConfig.runs).toBeUndefined();
            });

            it("should parse a MessageConfig with all fields populated", () => {
                const toolCall: ToolFunctionCall = {
                    id: "call1",
                    function: {
                        name: "calculateSum",
                        arguments: JSON.stringify({ a: 1, b: 2 }),
                    },
                    result: {
                        success: false,
                        error: { code: "CALC_ERROR", message: "Invalid input" },
                    },
                };

                const runConfig: ChatMessageRunConfig = {
                    runId: "run1",
                    resourceVersionId: "rv1",
                    taskId: "task1",
                    createdAt: "2024-01-01T00:00:00Z",
                };

                const config: MessageConfigObject = {
                    __version: "1.0",
                    resources: [],
                    contextHints: ["existing hint"],
                    eventTopic: "payment-succeeded",
                    respondingBots: ["bot1", "bot2"],
                    role: "system",
                    turnId: 10,
                    toolCalls: [toolCall],
                    runs: [runConfig],
                };

                const messageConfig = MessageConfig.parse({ config }, mockLogger);

                expect(messageConfig.contextHints).toEqual(["existing hint"]);
                expect(messageConfig.eventTopic).toBe("payment-succeeded");
                expect(messageConfig.respondingBots).toEqual(["bot1", "bot2"]);
                expect(messageConfig.role).toBe("system");
                expect(messageConfig.turnId).toBe(10);
                expect(messageConfig.toolCalls).toEqual([toolCall]);
                expect(messageConfig.runs).toEqual([runConfig]);
            });
        });

        describe("mutator methods", () => {
            it("should set context hints", () => {
                const messageConfig = MessageConfig.default();
                messageConfig.setContextHints(["new hint 1", "new hint 2"]);
                expect(messageConfig.contextHints).toEqual(["new hint 1", "new hint 2"]);
            });

            it("should set event topic", () => {
                const messageConfig = MessageConfig.default();
                messageConfig.setEventTopic("order-created");
                expect(messageConfig.eventTopic).toBe("order-created");
            });

            it("should unset event topic", () => {
                const messageConfig = MessageConfig.default();
                messageConfig.setEventTopic("test");
                messageConfig.setEventTopic(undefined);
                expect(messageConfig.eventTopic).toBeUndefined();
            });

            it("should set responding bots", () => {
                const messageConfig = MessageConfig.default();
                messageConfig.setRespondingBots(["bot1", "@all", "bot2"]);
                expect(messageConfig.respondingBots).toEqual(["bot1", "@all", "bot2"]);
            });

            it("should set role", () => {
                const messageConfig = MessageConfig.default();
                
                messageConfig.setRole("assistant");
                expect(messageConfig.role).toBe("assistant");
                
                messageConfig.setRole("system");
                expect(messageConfig.role).toBe("system");
                
                messageConfig.setRole("tool");
                expect(messageConfig.role).toBe("tool");
                
                messageConfig.setRole("user");
                expect(messageConfig.role).toBe("user");
            });

            it("should set turn ID", () => {
                const messageConfig = MessageConfig.default();
                
                messageConfig.setTurnId(42);
                expect(messageConfig.turnId).toBe(42);
                
                messageConfig.setTurnId(null);
                expect(messageConfig.turnId).toBeNull();
            });

            it("should set tool calls", () => {
                const messageConfig = MessageConfig.default();
                const toolCalls: ToolFunctionCall[] = [
                    {
                        id: "call1",
                        function: { name: "tool1", arguments: "{}" },
                        result: { success: true, output: "done" },
                    },
                    {
                        id: "call2",
                        function: { name: "tool2", arguments: JSON.stringify({ key: "value" }) },
                    },
                ];
                
                messageConfig.setToolCalls(toolCalls);
                expect(messageConfig.toolCalls).toEqual(toolCalls);
            });

            it("should set runs", () => {
                const messageConfig = MessageConfig.default();
                const runs: ChatMessageRunConfig[] = [
                    {
                        runId: "run1",
                        resourceVersionId: "rv1",
                        resourceVersionName: "My Routine v1",
                        taskId: "task1",
                        runStatus: "running",
                        createdAt: "2024-01-01T00:00:00Z",
                    },
                    {
                        runId: "run2",
                        resourceVersionId: "rv2",
                        taskId: "task2",
                        createdAt: "2024-01-01T00:10:00Z",
                        completedAt: "2024-01-01T00:15:00Z",
                    },
                ];
                
                messageConfig.setRuns(runs);
                expect(messageConfig.runs).toEqual(runs);
            });
        });

        describe("integration tests", () => {
            it("should handle full lifecycle: create, modify, export, parse", () => {
                // Create initial config
                const initial = MessageConfig.default();
                
                // Modify it
                initial.setEventTopic("user-action");
                initial.setRole("assistant");
                initial.setContextHints(["Remember user preference"]);
                initial.setRespondingBots(["bot1"]);
                initial.setTurnId(1);
                initial.setToolCalls([
                    {
                        id: "calc1",
                        function: { name: "calculate", arguments: JSON.stringify({ expr: "2+2" }) },
                        result: { success: true, output: 4 },
                    },
                ]);
                initial.setRuns([
                    {
                        runId: "run123",
                        resourceVersionId: "routine456",
                        taskId: "task789",
                        createdAt: new Date().toISOString(),
                    },
                ]);

                // Export it
                const exported = initial.export();

                // Parse it back
                const parsed = MessageConfig.parse({ config: exported }, mockLogger);

                // Verify everything matches
                expect(parsed.eventTopic).toBe("user-action");
                expect(parsed.role).toBe("assistant");
                expect(parsed.contextHints).toEqual(["Remember user preference"]);
                expect(parsed.respondingBots).toEqual(["bot1"]);
                expect(parsed.turnId).toBe(1);
                expect(parsed.toolCalls).toHaveLength(1);
                expect(parsed.toolCalls?.[0]?.function.name).toBe("calculate");
                expect(parsed.runs).toHaveLength(1);
                expect(parsed.runs?.[0]?.runId).toBe("run123");
            });
        });
    });
});
