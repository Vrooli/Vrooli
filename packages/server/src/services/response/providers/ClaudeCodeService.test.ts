// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-08-05
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { LlmServiceId, LATEST_CONFIG_VERSION, type MessageState, ClaudeCodeModel } from "@vrooli/shared";
import { ClaudeCodeService, TOOL_PROFILES, isClaudeCodeConfig } from "./ClaudeCodeService.js";
import { AIServiceErrorType } from "../registry.js";
import { TokenEstimatorType } from "../tokenTypes.js";
import { EventEmitter } from "events";
import type { spawn } from "child_process";

// Mock the logger
vi.mock("../../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
        debug: vi.fn(),
    },
}));

// Mock child_process
vi.mock("child_process", () => ({
    spawn: vi.fn(),
    exec: vi.fn(),
    execSync: vi.fn(),
    execFile: vi.fn(),
    fork: vi.fn(),
}));

// Mock message validation functions
vi.mock("../messageValidation.js", () => ({
    hasHarmfulContent: vi.fn().mockReturnValue({ isHarmful: false }),
    hasPromptInjection: vi.fn().mockReturnValue({ isInjection: false }),
}));

// Mock context generation
vi.mock("../contextGeneration.js", () => ({
    generateContextFromMessages: vi.fn().mockReturnValue([
        { role: "user", content: "test message" },
    ]),
}));

// Create a mock ChildProcess
class MockChildProcess extends EventEmitter {
    stdout = new EventEmitter();
    stderr = new EventEmitter();
    stdin = { end: vi.fn() };
    killed = false;
    pid = 12345;
    
    kill(_signal?: string) {
        this.killed = true;
        this.emit("exit", 0);
        return true;
    }
    
    on(event: string, listener: (...args: unknown[]) => void) {
        return super.on(event, listener);
    }
}

describe("ClaudeCodeService", () => {
    let service: ClaudeCodeService;
    let mockChildProcess: MockChildProcess;
    let mockAbortController: AbortController;
    let mockSpawn: ReturnType<typeof vi.mocked<typeof spawn>>;

    beforeEach(async () => {
        vi.clearAllMocks();
        mockAbortController = new AbortController();
        mockChildProcess = new MockChildProcess();
        
        // Import the mocked spawn function
        const { spawn } = await import("child_process");
        mockSpawn = vi.mocked(spawn);
        mockSpawn.mockReturnValue(mockChildProcess);
        
        service = new ClaudeCodeService({
            cliCommand: "claude",
            workingDirectory: "/test/dir",
            defaultModel: ClaudeCodeModel.Sonnet,
        });
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("constructor", () => {
        it("should initialize with default values", () => {
            const defaultService = new ClaudeCodeService();
            expect(defaultService.defaultModel).toBe(ClaudeCodeModel.Sonnet);
            expect(defaultService.__id).toBe(LlmServiceId.ClaudeCode);
            expect(defaultService.featureFlags.supportsStatefulConversations).toBe(true);
        });

        it("should initialize with custom options", () => {
            const customService = new ClaudeCodeService({
                cliCommand: "/custom/path/claude",
                workingDirectory: "/custom/dir",
                defaultModel: ClaudeCodeModel.Opus,
                allowedTools: ["Bash", "Edit"],
            });
            expect(customService.defaultModel).toBe(ClaudeCodeModel.Opus);
        });
    });

    describe("supportsModel", () => {
        it("should return true for supported Claude models", async () => {
            expect(await service.supportsModel("sonnet")).toBe(true);
            expect(await service.supportsModel("opus")).toBe(true);
            expect(await service.supportsModel("haiku")).toBe(true);
            expect(await service.supportsModel("claude-3.5-sonnet")).toBe(true);
        });

        it("should return false for unsupported models", async () => {
            expect(await service.supportsModel("gpt-4")).toBe(false);
            expect(await service.supportsModel("random-model")).toBe(false);
        });
    });

    describe("generateContext", () => {
        it("should delegate to generateContextFromMessages", () => {
            const messages = [
                { 
                    id: "msg-1", 
                    text: "Hello", 
                    config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                    language: "en", 
                    createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                    parent: null,
                    user: { id: "user123" },
                },
            ] as MessageState[];

            const context = service.generateContext(messages, "System message");

            expect(context).toEqual([{ role: "user", content: "test message" }]);
        });
    });

    describe("generateResponseStreaming", () => {
        it("should generate streaming response successfully", async () => {
            const options = {
                model: "sonnet",
                input: [{ 
                    id: "msg-1", 
                    text: "Hello", 
                    config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                    language: "en", 
                    createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                    parent: null,
                    user: { id: "user123" },
                }] as MessageState[],
                systemMessage: "You are helpful",
                maxTokens: 100,
                signal: mockAbortController.signal,
                tools: [],
            };

            // Start the generator
            const generator = service.generateResponseStreaming(options);
            const generatorPromise = (async () => {
                const events = [];
                for await (const event of generator) {
                    events.push(event);
                }
                return events;
            })();

            // Simulate Claude CLI output
            setTimeout(() => {
                // System initialization event
                mockChildProcess.stdout.emit("data", JSON.stringify({
                    type: "system",
                    subtype: "init",
                    session_id: "session-123",
                    model: "sonnet",
                    tools: ["Bash", "Edit"],
                }) + "\n");

                // Assistant response
                mockChildProcess.stdout.emit("data", JSON.stringify({
                    type: "assistant",
                    message: {
                        id: "msg-123",
                        content: [{ type: "text", text: "Hello there!" }],
                        role: "assistant",
                        model: "sonnet",
                    },
                    session_id: "session-123",
                }) + "\n");

                // Result event
                mockChildProcess.stdout.emit("data", JSON.stringify({
                    type: "result",
                    subtype: "success",
                    is_error: false,
                    duration_ms: 1000,
                    num_turns: 1,
                    result: "success",
                    session_id: "session-123",
                    usage: {
                        input_tokens: 10,
                        output_tokens: 5,
                    },
                }) + "\n");

                // End process
                mockChildProcess.emit("exit", 0);
            }, 10);

            const events = await generatorPromise;

            expect(events).toHaveLength(2);
            expect(events[0]).toEqual({ type: "text", content: "Hello there!" });
            expect(events[1]).toEqual({ type: "done", cost: 0 });
        });

        it("should handle tool calls", async () => {
            const options = {
                model: "sonnet",
                input: [{ 
                    id: "msg-1", 
                    text: "List files", 
                    config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                    language: "en", 
                    createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                    parent: null,
                    user: { id: "user123" },
                }] as MessageState[],
                tools: [],
            };

            const generator = service.generateResponseStreaming(options);
            const generatorPromise = (async () => {
                const events = [];
                for await (const event of generator) {
                    events.push(event);
                }
                return events;
            })();

            setTimeout(() => {
                // Tool use event
                mockChildProcess.stdout.emit("data", JSON.stringify({
                    type: "tool_use",
                    tool_name: "Bash",
                    tool_input: { command: "ls -la" },
                    tool_use_id: "tool-123",
                    session_id: "session-123",
                }) + "\n");

                // Result event
                mockChildProcess.stdout.emit("data", JSON.stringify({
                    type: "result",
                    subtype: "success",
                    is_error: false,
                    session_id: "session-123",
                }) + "\n");

                mockChildProcess.emit("exit", 0);
            }, 10);

            const events = await generatorPromise;

            expect(events).toHaveLength(2);
            expect(events[0]).toEqual({
                type: "function_call",
                name: "Bash",
                arguments: { command: "ls -la" },
                callId: "tool-123",
            });
            expect(events[1]).toEqual({ type: "done", cost: 0 });
        });

        it("should throw error for unsupported model", async () => {
            const options = {
                model: "unsupported-model",
                input: [{ 
                    id: "msg-1", 
                    text: "Hello", 
                    config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                    language: "en", 
                    createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                    parent: null,
                    user: { id: "user123" },
                }] as MessageState[],
                tools: [],
            };

            await expect(async () => {
                for await (const _ of service.generateResponseStreaming(options)) {
                    // Should not reach here
                }
            }).rejects.toThrow("Model unsupported-model not supported by Claude Code");
        });

        it("should handle spawn errors", async () => {
            mockSpawn.mockImplementationOnce(() => {
                throw new Error("Command not found");
            });

            const options = {
                model: "sonnet",
                input: [{ 
                    id: "msg-1", 
                    text: "Hello", 
                    config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                    language: "en", 
                    createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                    parent: null,
                    user: { id: "user123" },
                }] as MessageState[],
                tools: [],
            };

            await expect(async () => {
                for await (const _ of service.generateResponseStreaming(options)) {
                    // Should not reach here
                }
            }).rejects.toThrow("Failed to spawn Claude CLI: Error: Command not found");
        });

        it("should handle process with no stdout/stderr", async () => {
            const brokenProcess = new MockChildProcess();
            brokenProcess.stdout = null as any;
            brokenProcess.stderr = null as any;
            mockSpawn.mockReturnValueOnce(brokenProcess);

            const options = {
                model: "sonnet",
                input: [{ 
                    id: "msg-1", 
                    text: "Hello", 
                    config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                    language: "en", 
                    createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                    parent: null,
                    user: { id: "user123" },
                }] as MessageState[],
                tools: [],
            };

            await expect(async () => {
                for await (const _ of service.generateResponseStreaming(options)) {
                    // Should not reach here
                }
            }).rejects.toThrow("Failed to get process streams");
        });

        it("should handle abort signal", async () => {
            const options = {
                model: "sonnet",
                input: [{ id: "msg-1", text: "Hello", config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, language: "en", createdAt: new Date("2024-01-01T00:00:00Z") }] as unknown as MessageState[],
                signal: mockAbortController.signal,
                tools: [],
            };

            const generator = service.generateResponseStreaming(options);
            
            // Abort the request
            setTimeout(() => {
                mockAbortController.abort();
            }, 10);

            setTimeout(() => {
                mockChildProcess.emit("exit", 0);
            }, 20);

            const events = [];
            for await (const event of generator) {
                events.push(event);
            }

            expect(mockChildProcess.kill).toHaveBeenCalled();
        });

        it("should handle malformed JSON lines", async () => {
            const options = {
                model: "sonnet",
                input: [{ 
                    id: "msg-1", 
                    text: "Hello", 
                    config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                    language: "en", 
                    createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                    parent: null,
                    user: { id: "user123" },
                }] as MessageState[],
                tools: [],
            };

            const generator = service.generateResponseStreaming(options);
            const generatorPromise = (async () => {
                const events = [];
                for await (const event of generator) {
                    events.push(event);
                }
                return events;
            })();

            setTimeout(() => {
                // Send malformed JSON
                mockChildProcess.stdout.emit("data", "invalid json\n");
                
                // Send valid result
                mockChildProcess.stdout.emit("data", JSON.stringify({
                    type: "result",
                    subtype: "success",
                    is_error: false,
                    session_id: "session-123",
                }) + "\n");

                mockChildProcess.emit("exit", 0);
            }, 10);

            const events = await generatorPromise;
            expect(events).toHaveLength(1);
            expect(events[0]).toEqual({ type: "done", cost: 0 });
        });
    });

    describe("getContextSize", () => {
        it("should return default context size", () => {
            const size = service.getContextSize();
            expect(size).toBe(200_000);
        });

        it("should return context size for specific model", () => {
            const size = service.getContextSize("sonnet");
            expect(size).toBe(200_000);
        });
    });

    describe("getModelInfo", () => {
        it("should return model info for all Claude models", () => {
            const modelInfo = service.getModelInfo();

            expect(modelInfo).toHaveProperty(ClaudeCodeModel.Sonnet);
            expect(modelInfo).toHaveProperty(ClaudeCodeModel.Opus);
            expect(modelInfo).toHaveProperty(ClaudeCodeModel.Haiku);

            expect(modelInfo[ClaudeCodeModel.Sonnet]).toMatchObject({
                enabled: true,
                name: "Claude Code Sonnet",
                inputCost: 0,
                outputCost: 0,
                contextWindow: 200_000,
                maxOutputTokens: 8_192,
                supportsReasoning: true,
            });

            expect(modelInfo[ClaudeCodeModel.Opus]).toMatchObject({
                enabled: true,
                name: "Claude Code Opus",
                inputCost: 0,
                outputCost: 0,
                maxOutputTokens: 4_096,
                supportsReasoning: false,
            });
        });
    });

    describe("getMaxOutputTokens", () => {
        it("should return default max output tokens", () => {
            const maxTokens = service.getMaxOutputTokens();
            expect(maxTokens).toBe(8_192);
        });

        it("should return max output tokens for specific model", () => {
            const maxTokens = service.getMaxOutputTokens(ClaudeCodeModel.Opus);
            expect(maxTokens).toBe(4_096);
        });
    });

    describe("getMaxOutputTokensRestrained", () => {
        it("should return max output tokens without credit constraints", () => {
            const params = {
                maxCredits: BigInt(1),
                model: "sonnet",
                inputTokens: 10000,
            };

            const maxTokens = service.getMaxOutputTokensRestrained(params);
            expect(maxTokens).toBe(8_192); // No credit constraints for Claude Code
        });
    });

    describe("getResponseCost", () => {
        it("should always return zero cost", () => {
            const params = {
                model: "sonnet",
                usage: { input: 1000, output: 500 },
            };

            const cost = service.getResponseCost(params);
            expect(cost).toBe(0); // Monthly subscription model
        });
    });

    describe("getEstimationInfo", () => {
        it("should return estimation info", () => {
            const info = service.getEstimationInfo();
            expect(info).toEqual({
                estimationModel: TokenEstimatorType.Default,
                encoding: "cl100k_base",
            });
        });
    });

    describe("getModel", () => {
        it("should map common model names", () => {
            expect(service.getModel("sonnet")).toBe(ClaudeCodeModel.Sonnet);
            expect(service.getModel("opus")).toBe(ClaudeCodeModel.Opus);
            expect(service.getModel("haiku")).toBe(ClaudeCodeModel.Haiku);
            expect(service.getModel("claude-3.5-sonnet")).toBe(ClaudeCodeModel.Claude_3_5_Sonnet);
        });

        it("should return provided model if not in mapping", () => {
            expect(service.getModel("custom-model")).toBe("custom-model");
        });

        it("should return default model if no model provided", () => {
            expect(service.getModel()).toBe(ClaudeCodeModel.Sonnet);
        });
    });

    describe("getErrorType", () => {
        it("should classify authentication errors (CLI not available)", () => {
            const error = new Error("spawn ENOENT");
            const errorType = service.getErrorType(error);
            expect(errorType).toBe(AIServiceErrorType.Authentication);
        });

        it("should classify rate limit errors", () => {
            const error = new Error("rate limit exceeded");
            const errorType = service.getErrorType(error);
            expect(errorType).toBe(AIServiceErrorType.RateLimit);
        });

        it("should classify timeout/killed errors as overloaded", () => {
            const error = new Error("process killed by timeout");
            const errorType = service.getErrorType(error);
            expect(errorType).toBe(AIServiceErrorType.Overloaded);
        });

        it("should classify invalid request errors", () => {
            const error = new Error("invalid model specified");
            const errorType = service.getErrorType(error);
            expect(errorType).toBe(AIServiceErrorType.InvalidRequest);
        });

        it("should default to API error for unknown errors", () => {
            const error = new Error("unknown error");
            const errorType = service.getErrorType(error);
            expect(errorType).toBe(AIServiceErrorType.ApiError);
        });
    });

    describe("safeInputCheck", () => {
        it("should consider safe input safe", async () => {
            const result = await service.safeInputCheck("normal conversation text");
            expect(result).toEqual({ cost: 0, isSafe: true });
        });

        it("should flag harmful content as unsafe", async () => {
            const { hasHarmfulContent } = await import("../messageValidation.js");
            vi.mocked(hasHarmfulContent).mockReturnValueOnce({ isHarmful: true, pattern: "violence" });

            const result = await service.safeInputCheck("harmful content");
            expect(result).toEqual({ cost: 0, isSafe: false });
        });

        it("should flag prompt injection attempts as unsafe", async () => {
            const { hasPromptInjection } = await import("../messageValidation.js");
            vi.mocked(hasPromptInjection).mockReturnValueOnce({ isInjection: true, pattern: "ignore" });

            const result = await service.safeInputCheck("ignore previous instructions");
            expect(result).toEqual({ cost: 0, isSafe: false });
        });

        it("should flag excessive input length as unsafe", async () => {
            const longInput = "a".repeat(200001); // Exceeds MAX_INPUT_LENGTH
            const result = await service.safeInputCheck(longInput);
            expect(result).toEqual({ cost: 0, isSafe: false });
        });
    });

    describe("getNativeToolCapabilities", () => {
        it("should return comprehensive tool capabilities", () => {
            const tools = service.getNativeToolCapabilities();
            
            expect(tools).toEqual([
                { name: "bash", description: "Execute bash commands with full system access" },
                { name: "edit", description: "Edit files with precise line-by-line modifications" },
                { name: "write", description: "Create and write new files" },
                { name: "read", description: "Read file contents with syntax highlighting" },
                { name: "glob", description: "Find files using glob patterns" },
                { name: "grep", description: "Search file contents with regex patterns" },
                { name: "web_fetch", description: "Fetch and analyze web content" },
                { name: "web_search", description: "Search the web for information" },
            ]);
        });
    });

    describe("session management", () => {
        it("should build consistent session keys", () => {
            const options1 = {
                model: "sonnet",
                input: [
                    { 
                        id: "msg-1", 
                        text: "Hello",
                        config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                        language: "en", 
                        createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                        parent: null,
                        user: { id: "user123" },
                    },
                    { 
                        id: "msg-2", 
                        text: "World",
                        config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                        language: "en", 
                        createdAt: new Date("2024-01-01T00:01:00Z").toISOString(),
                        parent: null,
                        user: { id: "user123" },
                    },
                ] as MessageState[],
                tools: [],
            };

            const options2 = {
                model: "sonnet", 
                input: [
                    { 
                        id: "msg-1", 
                        text: "Hello",
                        config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                        language: "en", 
                        createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                        parent: null,
                        user: { id: "user123" },
                    },
                    { 
                        id: "msg-2", 
                        text: "World",
                        config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                        language: "en", 
                        createdAt: new Date("2024-01-01T00:01:00Z").toISOString(),
                        parent: null,
                        user: { id: "user123" },
                    },
                ] as MessageState[],
                tools: [],
            };

            // Use reflection to access private method for testing
            const buildSessionKey = (service as any).buildSessionKey.bind(service);
            
            expect(buildSessionKey(options1)).toBe(buildSessionKey(options2));
        });

        it("should generate different session keys for different models/messages", () => {
            const options1 = {
                model: "sonnet",
                input: [{ 
                    id: "msg-1", 
                    text: "Hello",
                    config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                    language: "en", 
                    createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                    parent: null,
                    user: { id: "user123" },
                }] as MessageState[],
                tools: [],
            };

            const options2 = {
                model: "opus",
                input: [{ 
                    id: "msg-1", 
                    text: "Hello",
                    config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                    language: "en", 
                    createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                    parent: null,
                    user: { id: "user123" },
                }] as MessageState[],
                tools: [],
            };

            const buildSessionKey = (service as any).buildSessionKey.bind(service);
            
            expect(buildSessionKey(options1)).not.toBe(buildSessionKey(options2));
        });
    });

    describe("command building", () => {
        it("should build correct command arguments", () => {
            const options = {
                model: "sonnet",
                input: [{ 
                    id: "msg-1", 
                    text: "Hello", 
                    config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                    language: "en", 
                    createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                    parent: null,
                    user: { id: "user123" },
                }] as MessageState[],
                systemMessage: "You are helpful",
                maxTokens: 100,
                tools: [],
            };

            const buildCommandArgs = (service as any).buildCommandArgs.bind(service);
            const args = buildCommandArgs(options);

            expect(args).toContain("-p");
            expect(args).toContain("--output-format");
            expect(args).toContain("stream-json");
            expect(args).toContain("--verbose");
            expect(args).toContain("--model");
            expect(args).toContain("sonnet");
            expect(args).toContain("--allowedTools");
        });

        it("should include max-turns when maxTokens specified", () => {
            const options = {
                model: "sonnet",
                input: [{ 
                    id: "msg-1", 
                    text: "Hello", 
                    config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                    language: "en", 
                    createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                    parent: null,
                    user: { id: "user123" },
                }] as MessageState[],
                maxTokens: 100,
                tools: [],
            };

            const buildCommandArgs = (service as any).buildCommandArgs.bind(service);
            const args = buildCommandArgs(options);

            expect(args).toContain("--max-turns");
            expect(args).toContain("1");
        });
    });

    describe("service config functionality", () => {
        it("should use readonly tool profile", () => {
            const options = {
                model: "sonnet",
                input: [{ 
                    id: "msg-1", 
                    text: "Hello", 
                    config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                    language: "en", 
                    createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                    parent: null,
                    user: { id: "user123" },
                }] as MessageState[],
                tools: [],
                serviceConfig: {
                    toolProfile: "readonly",
                },
            };

            const buildCommandArgs = (service as any).buildCommandArgs.bind(service);
            const args = buildCommandArgs(options);

            // Should only have readonly tools
            const allowedToolsArgs = args.filter((_, i) => i > 0 && args[i - 1] === "--allowedTools");
            expect(allowedToolsArgs).toEqual(["Read", "Glob", "Grep", "LS"]);
        });

        it("should use explicit allowed tools list", () => {
            const options = {
                model: "sonnet",
                input: [{ 
                    id: "msg-1", 
                    text: "Hello", 
                    config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                    language: "en", 
                    createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                    parent: null,
                    user: { id: "user123" },
                }] as MessageState[],
                tools: [],
                serviceConfig: {
                    allowedTools: ["Read", "WebFetch"],
                },
            };

            const buildCommandArgs = (service as any).buildCommandArgs.bind(service);
            const args = buildCommandArgs(options);

            // Should only have specified tools
            const allowedToolsArgs = args.filter((_, i) => i > 0 && args[i - 1] === "--allowedTools");
            expect(allowedToolsArgs).toEqual(["Read", "WebFetch"]);
        });

        it("should prioritize explicit tools over profile", () => {
            const options = {
                model: "sonnet",
                input: [{ 
                    id: "msg-1", 
                    text: "Hello", 
                    config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                    language: "en", 
                    createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                    parent: null,
                    user: { id: "user123" },
                }] as MessageState[],
                tools: [],
                serviceConfig: {
                    toolProfile: "full",
                    allowedTools: ["Read"], // This should override the full profile
                },
            };

            const buildCommandArgs = (service as any).buildCommandArgs.bind(service);
            const args = buildCommandArgs(options);

            // Should only have explicitly allowed tools
            const allowedToolsArgs = args.filter((_, i) => i > 0 && args[i - 1] === "--allowedTools");
            expect(allowedToolsArgs).toEqual(["Read"]);
        });

        it("should use custom working directory", async () => {
            const customWorkingDir = "/custom/working/dir";

            const options = {
                model: "sonnet",
                input: [{ 
                    id: "msg-1", 
                    text: "Hello", 
                    config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                    language: "en", 
                    createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                    parent: null,
                    user: { id: "user123" },
                }] as MessageState[],
                tools: [],
                serviceConfig: {
                    workingDirectory: customWorkingDir,
                },
            };

            // Start generator but don't wait for it
            const generator = service.generateResponseStreaming(options);
            
            // Give it time to spawn
            await new Promise(resolve => setTimeout(resolve, 10));
            
            // Check that spawn was called with custom working directory
            expect(mockSpawn).toHaveBeenCalledWith(
                "claude",
                expect.any(Array),
                expect.objectContaining({
                    cwd: customWorkingDir,
                }),
            );

            // Clean up by aborting
            mockChildProcess.emit("exit", 0);
            
            // Consume generator to avoid unhandled promise
            for await (const _ of generator) {
                break;
            }
        });

        it("should handle invalid service config gracefully", () => {
            const options = {
                model: "sonnet",
                input: [{ 
                    id: "msg-1", 
                    text: "Hello", 
                    config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                    language: "en", 
                    createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                    parent: null,
                    user: { id: "user123" },
                }] as MessageState[],
                tools: [],
                serviceConfig: {
                    invalidOption: "should be ignored",
                    allowedTools: "not an array", // Invalid type
                } as any,
            };

            const buildCommandArgs = (service as any).buildCommandArgs.bind(service);
            const args = buildCommandArgs(options);

            // Should fall back to default tools
            const allowedToolsArgs = args.filter((_, i) => i > 0 && args[i - 1] === "--allowedTools");
            expect(allowedToolsArgs.length).toBeGreaterThan(0);
            expect(allowedToolsArgs).toContain("Bash(*)");
            expect(allowedToolsArgs).toContain("Edit");
        });

        it("should maintain backwards compatibility without serviceConfig", () => {
            const options = {
                model: "sonnet",
                input: [{ 
                    id: "msg-1", 
                    text: "Hello", 
                    config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                    language: "en", 
                    createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                    parent: null,
                    user: { id: "user123" },
                }] as MessageState[],
                tools: [],
                // No serviceConfig
            };

            const buildCommandArgs = (service as any).buildCommandArgs.bind(service);
            const args = buildCommandArgs(options);

            // Should use default tools from constructor
            const allowedToolsArgs = args.filter((_, i) => i > 0 && args[i - 1] === "--allowedTools");
            expect(allowedToolsArgs).toContain("Bash(*)");
            expect(allowedToolsArgs).toContain("Edit");
            expect(allowedToolsArgs).toContain("Write");
        });

        it("should use analysis profile", () => {
            const options = {
                model: "sonnet",
                input: [{ 
                    id: "msg-1", 
                    text: "Analyze this code", 
                    config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                    language: "en", 
                    createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                    parent: null,
                    user: { id: "user123" },
                }] as MessageState[],
                tools: [],
                serviceConfig: {
                    toolProfile: "analysis",
                },
            };

            const buildCommandArgs = (service as any).buildCommandArgs.bind(service);
            const args = buildCommandArgs(options);

            // Should have analysis tools
            const allowedToolsArgs = args.filter((_, i) => i > 0 && args[i - 1] === "--allowedTools");
            expect(allowedToolsArgs).toEqual(["Read", "Glob", "Grep", "WebFetch", "WebSearch", "LS"]);
        });

        it("should use safe profile", () => {
            const options = {
                model: "sonnet",
                input: [{ 
                    id: "msg-1", 
                    text: "Edit this file", 
                    config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, 
                    language: "en", 
                    createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
                    parent: null,
                    user: { id: "user123" },
                }] as MessageState[],
                tools: [],
                serviceConfig: {
                    toolProfile: "safe",
                },
            };

            const buildCommandArgs = (service as any).buildCommandArgs.bind(service);
            const args = buildCommandArgs(options);

            // Should have safe tools (no Bash)
            const allowedToolsArgs = args.filter((_, i) => i > 0 && args[i - 1] === "--allowedTools");
            expect(allowedToolsArgs).toEqual(["Read", "Edit", "Write", "Glob", "Grep", "LS", "MultiEdit"]);
            expect(allowedToolsArgs).not.toContain("Bash(*)");
        });
    });

    describe("isClaudeCodeConfig type guard", () => {
        it("should return true for valid config", () => {
            const validConfigs = [
                {},
                { allowedTools: ["Read", "Write"] },
                { toolProfile: "readonly" },
                { workingDirectory: "/tmp" },
                { sessionTimeout: 5000 },
                { 
                    allowedTools: ["Read"], 
                    toolProfile: "safe",
                    workingDirectory: "/app",
                    sessionTimeout: 10000,
                },
            ];

            for (const config of validConfigs) {
                expect(isClaudeCodeConfig(config)).toBe(true);
            }
        });

        it("should return false for invalid config", () => {
            const invalidConfigs = [
                null,
                undefined,
                "string",
                123,
                [],
                { allowedTools: "not an array" },
                { toolProfile: 123 },
                { workingDirectory: true },
                { sessionTimeout: "not a number" },
            ];

            for (const config of invalidConfigs) {
                expect(isClaudeCodeConfig(config)).toBe(false);
            }
        });
    });

    describe("TOOL_PROFILES", () => {
        it("should have all expected profiles", () => {
            expect(Object.keys(TOOL_PROFILES)).toEqual(["readonly", "analysis", "safe", "full"]);
        });

        it("should not have Bash in readonly profile", () => {
            expect(TOOL_PROFILES.readonly).not.toContain("Bash(*)");
        });

        it("should not have Bash in safe profile", () => {
            expect(TOOL_PROFILES.safe).not.toContain("Bash(*)");
        });

        it("should have Bash in full profile", () => {
            expect(TOOL_PROFILES.full).toContain("Bash(*)");
        });

        it("should have web tools in analysis profile", () => {
            expect(TOOL_PROFILES.analysis).toContain("WebFetch");
            expect(TOOL_PROFILES.analysis).toContain("WebSearch");
        });
    });
});
