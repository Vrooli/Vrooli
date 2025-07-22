import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { 
    type ResponseContext, 
    type BotParticipant, 
    type ChatConfigObject,
    type TeamConfigObject,
    type SessionUser,
    type Tool,
    type ToolCall,
    type ModelType,
    AgentSpec,
    ResourceSubType,
    generatePK,
    ModelStrategy,
} from "@vrooli/shared";
import { ResponseService, type PromptContext, type ResponseGenerationParams } from "./responseService.js";
import { FallbackRouter, type LlmRouter } from "./router.js";
import { CompositeToolRunner, type ToolRunner } from "./toolRunner.js";
import { MessageHistoryBuilder } from "./messageHistoryBuilder.js";
import { EventPublisher } from "../events/publisher.js";
import { logger } from "../../events/logger.js";
import { SwarmStateAccessor } from "../execution/shared/SwarmStateAccessor.js";
import type { OkErr } from "../conversation/types.js";
import * as fs from "fs/promises";
import { NetworkMonitor } from "./NetworkMonitor.js";
import { AIServiceRegistry } from "./registry.js";
import { ModelSelectionStrategyFactory } from "./ModelSelectionStrategy.js";

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

vi.mock("../execution/shared/SwarmStateAccessor.js", () => ({
    SwarmStateAccessor: {
        get: vi.fn(() => ({
            getState: vi.fn().mockResolvedValue({
                id: "swarm123",
                name: "Test Swarm",
                config: {},
                agentPool: [],
                subtasks: [],
                environment: {},
            }),
        })),
    },
}));

vi.mock("./messageHistoryBuilder.js", () => ({
    MessageHistoryBuilder: {
        get: vi.fn(() => ({
            buildMessages: vi.fn().mockResolvedValue([
                { role: "system", content: "You are a helpful assistant." },
                { role: "user", content: "Hello" },
            ]),
        })),
    },
}));

vi.mock("fs/promises", () => ({
    readFile: vi.fn(),
}));

vi.mock("./NetworkMonitor.js", () => ({
    NetworkMonitor: {
        getInstance: vi.fn(() => ({
            getState: vi.fn(),
        })),
    },
}));

vi.mock("./registry.js", () => ({
    AIServiceRegistry: {
        get: vi.fn(() => ({
            getBestService: vi.fn(),
        })),
    },
}));

vi.mock("./ModelSelectionStrategy.js", () => ({
    ModelSelectionStrategyFactory: {
        getStrategy: vi.fn(() => ({
            selectModel: vi.fn(),
        })),
    },
}));

describe("ResponseService", () => {
    let responseService: ResponseService;
    let mockToolRunner: ToolRunner;
    let mockLlmRouter: LlmRouter;
    let mockContext: ResponseContext;

    beforeEach(() => {
        vi.clearAllMocks();
        
        // Create mock tool runner
        mockToolRunner = {
            run: vi.fn().mockResolvedValue({
                ok: true,
                data: { output: "Tool executed successfully", creditsUsed: "10" },
            }),
        } as unknown as ToolRunner;

        // Create mock LLM router
        mockLlmRouter = {
            complete: vi.fn().mockResolvedValue({
                ok: true,
                data: {
                    messageContent: "Hello! How can I help you?",
                    toolCalls: [],
                    creditsUsed: "50",
                    tokensUsed: { prompt: 100, completion: 20, total: 120 },
                },
            }),
        } as unknown as LlmRouter;

        // Create test context
        mockContext = {
            swarmId: "swarm123",
            conversationId: "conv123",
            bot: {
                id: "bot123",
                name: "TestBot",
                role: "assistant",
                model: "gpt-4" as ModelType,
                config: {
                    temperature: 0.7,
                    maxTokens: 4000,
                    systemPrompt: "You are a helpful assistant",
                } as ChatConfigObject,
            } as BotParticipant,
            userData: {
                id: "user123",
                role: "User",
            } as SessionUser,
            messages: [],
            availableTools: [
                {
                    name: "search",
                    description: "Search for information",
                    inputSchema: {
                        type: "object",
                        properties: { query: { type: "string" } },
                        required: ["query"],
                    },
                } as Tool,
            ],
            strategy: { type: "standard" },
            resourceLimits: {
                maxCredits: "1000",
                maxTokens: 4000,
                timeoutMs: 60000,
            },
        };

        responseService = new ResponseService(
            { enableDetailedLogging: true },
            mockToolRunner,
            mockLlmRouter,
        );
    });

    afterEach(() => {
        vi.useRealTimers();
    });

    describe("generateResponse", () => {
        it("should generate a simple text response successfully", async () => {
            const params: ResponseGenerationParams = {
                context: mockContext,
            };

            const result = await responseService.generateResponse(params);

            expect(result.success).toBe(true);
            expect(result.botId).toBe("bot123");
            expect(result.message).toBe("Hello! How can I help you?");
            expect(result.toolCalls).toEqual([]);
            expect(result.resourcesUsed.creditsUsed).toBe("50");
            expect(result.error).toBeUndefined();

            // Verify typing events were emitted
            const eventPublisher = EventPublisher.get();
            expect(eventPublisher.publish).toHaveBeenCalledWith({
                type: "ChatTypingStart",
                payload: { chatId: "conv123", userId: "bot123" },
            });
            expect(eventPublisher.publish).toHaveBeenCalledWith({
                type: "ChatTypingStop",
                payload: { chatId: "conv123", userId: "bot123" },
            });
        });

        it("should handle tool calls in response", async () => {
            // Mock LLM response with tool call
            (mockLlmRouter.complete as any).mockResolvedValueOnce({
                ok: true,
                data: {
                    messageContent: "Let me search for that information.",
                    toolCalls: [
                        {
                            id: "call123",
                            name: "search",
                            arguments: { query: "test query" },
                        },
                    ],
                    creditsUsed: "50",
                    tokensUsed: { prompt: 100, completion: 30, total: 130 },
                },
            });

            // Mock second LLM call after tool execution
            (mockLlmRouter.complete as any).mockResolvedValueOnce({
                ok: true,
                data: {
                    messageContent: "Based on my search, here's what I found...",
                    toolCalls: [],
                    creditsUsed: "60",
                    tokensUsed: { prompt: 150, completion: 40, total: 190 },
                },
            });

            const params: ResponseGenerationParams = {
                context: mockContext,
            };

            const result = await responseService.generateResponse(params);

            expect(result.success).toBe(true);
            expect(result.message).toBe("Based on my search, here's what I found...");
            expect(result.toolCalls).toHaveLength(1);
            expect(result.toolCalls[0]).toMatchObject({
                id: "call123",
                name: "search",
                arguments: { query: "test query" },
                result: "Tool executed successfully",
            });
            expect(result.resourcesUsed.creditsUsed).toBe("120"); // 50 + 10 + 60
            expect(result.resourcesUsed.toolCalls).toBe(1);

            // Verify tool was executed
            expect(mockToolRunner.run).toHaveBeenCalledWith(
                "search",
                { query: "test query" },
                expect.any(Object),
            );
        });

        it("should handle LLM errors gracefully", async () => {
            (mockLlmRouter.complete as any).mockResolvedValueOnce({
                ok: false,
                error: {
                    code: "LLM_ERROR",
                    message: "Service unavailable",
                    creditsUsed: "0",
                },
            });

            const params: ResponseGenerationParams = {
                context: mockContext,
            };

            const result = await responseService.generateResponse(params);

            expect(result.success).toBe(false);
            expect(result.error).toMatchObject({
                code: "LLM_ERROR",
                message: "Service unavailable",
            });
            expect(result.resourcesUsed.creditsUsed).toBe("0");
        });

        it("should handle tool execution errors", async () => {
            // Mock LLM response with tool call
            (mockLlmRouter.complete as any).mockResolvedValueOnce({
                ok: true,
                data: {
                    messageContent: "Let me help with that.",
                    toolCalls: [
                        {
                            id: "call456",
                            name: "search",
                            arguments: { query: "test" },
                        },
                    ],
                    creditsUsed: "50",
                    tokensUsed: { prompt: 100, completion: 20, total: 120 },
                },
            });

            // Mock tool error
            (mockToolRunner.run as any).mockResolvedValueOnce({
                ok: false,
                error: {
                    code: "TOOL_ERROR",
                    message: "Search service down",
                    creditsUsed: "5",
                },
            });

            // Mock recovery response
            (mockLlmRouter.complete as any).mockResolvedValueOnce({
                ok: true,
                data: {
                    messageContent: "I apologize, but I couldn't complete the search.",
                    toolCalls: [],
                    creditsUsed: "40",
                    tokensUsed: { prompt: 120, completion: 25, total: 145 },
                },
            });

            const params: ResponseGenerationParams = {
                context: mockContext,
            };

            const result = await responseService.generateResponse(params);

            expect(result.success).toBe(true);
            expect(result.message).toBe("I apologize, but I couldn't complete the search.");
            expect(result.toolCalls[0].error).toBe("Search service down");
            expect(result.resourcesUsed.creditsUsed).toBe("95"); // 50 + 5 + 40
        });

        it("should respect maximum tool iterations", async () => {
            // Mock LLM to always return tool calls
            (mockLlmRouter.complete as any).mockResolvedValue({
                ok: true,
                data: {
                    messageContent: "Processing...",
                    toolCalls: [
                        {
                            id: generatePK(),
                            name: "search",
                            arguments: { query: "test" },
                        },
                    ],
                    creditsUsed: "50",
                    tokensUsed: { prompt: 100, completion: 20, total: 120 },
                },
            });

            const params: ResponseGenerationParams = {
                context: mockContext,
            };

            const result = await responseService.generateResponse(params);

            // Should stop after MAX_TOOL_ITERATIONS (10)
            expect(mockToolRunner.run).toHaveBeenCalledTimes(10);
            expect(result.success).toBe(true);
            expect(result.toolCalls).toHaveLength(10);
        });

        it("should handle invalid context", async () => {
            const params: ResponseGenerationParams = {
                context: {
                    ...mockContext,
                    swarmId: "", // Invalid
                },
            };

            const result = await responseService.generateResponse(params);

            expect(result.success).toBe(false);
            expect(result.error?.code).toBe("INVALID_CONTEXT");
        });

        it("should handle abort signal", async () => {
            const abortController = new AbortController();
            
            // Abort immediately
            abortController.abort();

            const params: ResponseGenerationParams = {
                context: mockContext,
                abortSignal: abortController.signal,
            };

            const result = await responseService.generateResponse(params);

            expect(result.success).toBe(false);
            expect(result.error?.code).toBe("OPERATION_CANCELLED");
        });

        it("should track resources accurately across multiple operations", async () => {
            // First LLM call with tool
            (mockLlmRouter.complete as any).mockResolvedValueOnce({
                ok: true,
                data: {
                    messageContent: "Let me search multiple things.",
                    toolCalls: [
                        {
                            id: "call1",
                            name: "search",
                            arguments: { query: "query1" },
                        },
                        {
                            id: "call2",
                            name: "search",
                            arguments: { query: "query2" },
                        },
                    ],
                    creditsUsed: "100",
                    tokensUsed: { prompt: 200, completion: 50, total: 250 },
                },
            });

            // Tool executions
            (mockToolRunner.run as any)
                .mockResolvedValueOnce({
                    ok: true,
                    data: { output: "Result 1", creditsUsed: "15" },
                })
                .mockResolvedValueOnce({
                    ok: true,
                    data: { output: "Result 2", creditsUsed: "20" },
                });

            // Final LLM call
            (mockLlmRouter.complete as any).mockResolvedValueOnce({
                ok: true,
                data: {
                    messageContent: "Here are the results from both searches...",
                    toolCalls: [],
                    creditsUsed: "80",
                    tokensUsed: { prompt: 300, completion: 60, total: 360 },
                },
            });

            const params: ResponseGenerationParams = {
                context: mockContext,
            };

            const result = await responseService.generateResponse(params);

            expect(result.success).toBe(true);
            expect(result.resourcesUsed.creditsUsed).toBe("215"); // 100 + 15 + 20 + 80
            expect(result.resourcesUsed.toolCalls).toBe(2);
            expect(result.toolCalls).toHaveLength(2);
        });

        it("should apply configuration overrides", async () => {
            const params: ResponseGenerationParams = {
                context: mockContext,
                overrides: {
                    defaultMaxTokens: 2000,
                    defaultTemperature: 0.5,
                },
            };

            await responseService.generateResponse(params);

            // Verify overrides were applied
            expect(mockLlmRouter.complete).toHaveBeenCalledWith(
                expect.objectContaining({
                    maxTokens: 2000,
                    temperature: 0.5,
                }),
            );
        });
    });

    describe("prompt generation", () => {
        it("should generate prompts with template variables", async () => {
            // Mock template file
            (fs.readFile as any).mockResolvedValueOnce(`
                Goal: {{GOAL}}
                Role: {{BOT.role}}
                Team: {{TEAM_CONFIG.name}}
                {{ROLE_SPECIFIC_INSTRUCTIONS}}
            `);

            const promptContext: PromptContext = {
                goal: "Help the user",
                bot: mockContext.bot,
                convoConfig: mockContext.bot.config as ChatConfigObject,
                teamConfig: {
                    name: "Support Team",
                    description: "Customer support",
                } as any,
                userId: "user123",
                swarmId: "swarm123",
            };

            // Access the private method through the public interface
            const params: ResponseGenerationParams = {
                context: mockContext,
            };

            await responseService.generateResponse(params);

            // Verify template was processed correctly
            const historyBuilder = MessageHistoryBuilder.get();
            expect(historyBuilder.buildMessages).toHaveBeenCalled();
        });

        it("should handle recruitment rules for leadership roles", async () => {
            const leaderBot: BotParticipant = {
                ...mockContext.bot,
                role: "leader",
            };

            const params: ResponseGenerationParams = {
                context: {
                    ...mockContext,
                    bot: leaderBot,
                },
            };

            await responseService.generateResponse(params);

            // Should include recruitment instructions for leader role
            const historyBuilder = MessageHistoryBuilder.get();
            expect(historyBuilder.buildMessages).toHaveBeenCalled();
        });

        it("should handle missing template files gracefully", async () => {
            // Mock template file not found
            (fs.readFile as any).mockRejectedValueOnce(new Error("ENOENT"));

            const params: ResponseGenerationParams = {
                context: mockContext,
            };

            const result = await responseService.generateResponse(params);

            // Should fall back to default template
            expect(result.success).toBe(true);
            expect(logger.warn).toHaveBeenCalledWith(
                expect.stringContaining("Template file not found"),
                expect.any(Object),
            );
        });
    });

    describe("buildSystemMessage (comprehensive prompt generation)", () => {
        const createPromptContext = (overrides: Partial<PromptContext> = {}): PromptContext => ({
            goal: "Help users with their questions",
            bot: {
                id: "bot123",
                name: "TestBot",
                role: "assistant",
                config: {
                    model: "gpt-4o",
                    temperature: 0.7,
                } as ChatConfigObject,
            } as BotParticipant,
            convoConfig: {
                teamId: "team123",
                swarmLeader: "leader_bot",
                subtasks: [
                    { id: "task1", description: "Help user", status: "in_progress" },
                ],
                blackboard: [],
                resources: [],
                stats: { totalToolCalls: 5, totalCredits: 100 },
            } as ChatConfigObject,
            team: {
                id: "team123",
                name: "Support Team",
                config: {
                    deploymentType: "saas",
                    goal: "Provide excellent support",
                    businessPrompt: "Focus on customer satisfaction",
                } as TeamConfigObject,
            },
            userId: "user123",
            swarmId: "swarm123",
            ...overrides,
        });

        beforeEach(() => {
            // Clear template cache before each test
            ResponseService.clearTemplateCache();
        });

        describe("direct prompt content", () => {
            it("should use direct prompt content when provided", async () => {
                const directContent = "You are a specialized assistant. Goal: {{GOAL}}";
                const context = createPromptContext();

                const result = await ResponseService.buildSystemMessage(context, {
                    directPromptContent: directContent,
                });

                expect(result).toContain("You are a specialized assistant");
                expect(result).toContain("Goal: Help users with their questions");
            });

            it("should process template variables in direct content", async () => {
                const directContent = `
                    Role: {{BOT.role}}
                    Member Count: {{MEMBER_COUNT_LABEL}}
                    Current Time: {{ISO_EPOCH_SECONDS}}
                    {{ROLE_SPECIFIC_INSTRUCTIONS}}
                `;
                const context = createPromptContext({
                    bot: { ...createPromptContext().bot, role: "coordinator" },
                });

                const result = await ResponseService.buildSystemMessage(context, {
                    directPromptContent: directContent,
                });

                expect(result).toContain("Role: coordinator");
                expect(result).toContain("Member Count: team-based swarm");
                expect(result).toMatch(/Current Time: \d+/);
                expect(result).toContain("## Recruitment rule:"); // coordinator is a leadership role
            });
        });

        describe("agent-specific prompts", () => {
            it("should use agent direct prompt when configured", async () => {
                const agentPrompt = "I am a specialized bot for {{GOAL}}";
                const context = createPromptContext({
                    bot: {
                        ...createPromptContext().bot,
                        config: {
                            ...createPromptContext().bot.config,
                            agentSpec: {
                                prompt: {
                                    source: "direct",
                                    content: agentPrompt,
                                    mode: "replace",
                                },
                            },
                        } as ChatConfigObject,
                    },
                });

                const result = await ResponseService.buildSystemMessage(context);

                expect(result).toContain("I am a specialized bot for Help users with their questions");
                expect(result).not.toContain("You are a helpful assistant"); // Should not use default
            });

            it("should supplement base template when mode is supplement", async () => {
                const baseTemplate = "You are a helpful assistant.";
                const agentPrompt = "Additional instructions: Focus on {{GOAL}}";
                
                (fs.readFile as any).mockResolvedValueOnce(baseTemplate);

                const context = createPromptContext({
                    bot: {
                        ...createPromptContext().bot,
                        config: {
                            ...createPromptContext().bot.config,
                            agentSpec: {
                                prompt: {
                                    source: "direct",
                                    content: agentPrompt,
                                    mode: "supplement",
                                },
                            },
                        } as ChatConfigObject,
                    },
                });

                const result = await ResponseService.buildSystemMessage(context, {
                    templateIdentifier: "custom.txt",
                });

                expect(result).toContain("You are a helpful assistant");
                expect(result).toContain("## Agent-Specific Instructions");
                expect(result).toContain("Additional instructions: Focus on Help users with their questions");
            });

            it("should handle agent prompt variables mapping", async () => {
                const agentPrompt = "My team is {{TEAM_NAME}} and my role is {{MY_ROLE}}";
                const context = createPromptContext({
                    bot: {
                        ...createPromptContext().bot,
                        config: {
                            ...createPromptContext().bot.config,
                            agentSpec: {
                                prompt: {
                                    source: "direct",
                                    content: agentPrompt,
                                    mode: "replace",
                                    variables: {
                                        TEAM_NAME: "context.team.name",
                                        MY_ROLE: "context.bot.role",
                                    },
                                },
                            },
                        } as ChatConfigObject,
                    },
                });

                const result = await ResponseService.buildSystemMessage(context);

                expect(result).toContain("My team is Support Team and my role is assistant");
            });
        });

        describe("template loading and caching", () => {
            it("should load template from file system", async () => {
                const templateContent = "Template from file: {{GOAL}}";
                (fs.readFile as any).mockResolvedValueOnce(templateContent);

                const context = createPromptContext();
                const result = await ResponseService.buildSystemMessage(context, {
                    templateIdentifier: "custom.txt",
                });

                expect(fs.readFile).toHaveBeenCalledWith(
                    expect.stringContaining("custom.txt"),
                    "utf-8",
                );
                expect(result).toContain("Template from file: Help users with their questions");
            });

            it("should cache loaded templates", async () => {
                const templateContent = "Cached template: {{GOAL}}";
                (fs.readFile as any).mockResolvedValue(templateContent);

                const context = createPromptContext();
                
                // First call
                await ResponseService.buildSystemMessage(context, {
                    templateIdentifier: "cached.txt",
                });
                
                // Second call
                await ResponseService.buildSystemMessage(context, {
                    templateIdentifier: "cached.txt",
                });

                // Should only read file once due to caching
                expect(fs.readFile).toHaveBeenCalledTimes(1);
            });

            it("should use default fallback when template loading fails", async () => {
                (fs.readFile as any).mockRejectedValue(new Error("File not found"));

                const context = createPromptContext();
                const result = await ResponseService.buildSystemMessage(context, {
                    templateIdentifier: "missing.txt",
                });

                expect(result).toContain("Critical: Prompt template file not found");
                expect(logger.error).toHaveBeenCalledWith(
                    expect.stringContaining("Failed to load template"),
                    expect.any(Object),
                );
            });
        });

        describe("variable resolution and processing", () => {
            it("should resolve standard template variables", async () => {
                const template = `
                    Goal: {{GOAL}}
                    Date: {{DISPLAY_DATE}}
                    Epoch: {{ISO_EPOCH_SECONDS}}
                    Member Count: {{MEMBER_COUNT_LABEL}}
                    Role Instructions: {{ROLE_SPECIFIC_INSTRUCTIONS}}
                `;
                (fs.readFile as any).mockResolvedValueOnce(template);

                const context = createPromptContext({
                    bot: { ...createPromptContext().bot, role: "leader" },
                });

                const result = await ResponseService.buildSystemMessage(context, {
                    templateIdentifier: "variables.txt",
                });

                expect(result).toContain("Goal: Help users with their questions");
                expect(result).toMatch(/Date: \d/); // Should contain formatted date
                expect(result).toMatch(/Epoch: \d+/); // Should contain epoch seconds
                expect(result).toContain("Member Count: team-based swarm");
                expect(result).toContain("## Recruitment rule:"); // leader role gets recruitment rules
            });

            it("should format swarm state information", async () => {
                const template = "Swarm State: {{SWARM_STATE}}";
                (fs.readFile as any).mockResolvedValueOnce(template);

                const context = createPromptContext({
                    convoConfig: {
                        teamId: "team123",
                        swarmLeader: "leader_bot",
                        subtasks: [
                            { id: "task1", description: "Active task", status: "in_progress" },
                            { id: "task2", description: "Todo task", status: "todo" },
                            { id: "task3", description: "Done task", status: "done" },
                        ],
                        blackboard: [{ id: "note1", content: "Important note" }],
                        stats: { totalToolCalls: 10, totalCredits: 250 },
                    } as ChatConfigObject,
                });

                const result = await ResponseService.buildSystemMessage(context, {
                    templateIdentifier: "swarm.txt",
                });

                expect(result).toContain("SWARM STATE DETAILS:");
                expect(result).toContain("Swarm Leader:\nleader_bot");
                expect(result).toContain("Subtasks (active: 2, completed: 1):");
                expect(result).toContain("Active task");
                expect(result).toContain("Stats:");
                expect(result).toContain("totalToolCalls");
            });

            it("should truncate very long swarm state sections", async () => {
                const template = "State: {{SWARM_STATE}}";
                (fs.readFile as any).mockResolvedValueOnce(template);

                const veryLongSubtasks = Array(100).fill(0).map((_, i) => ({
                    id: `task${i}`,
                    description: `Task ${i} with a very long description that repeats many times`.repeat(10),
                    status: "todo",
                }));

                const context = createPromptContext({
                    convoConfig: {
                        ...createPromptContext().convoConfig,
                        subtasks: veryLongSubtasks,
                    },
                });

                const result = await ResponseService.buildSystemMessage(context, {
                    templateIdentifier: "long.txt",
                    config: { maxStringPreviewLength: 500 },
                });

                // Should contain truncation indicator
                expect(result).toContain("...");
                // Should not be excessively long
                expect(result.length).toBeLessThan(10000);
            });

            it("should handle tool schemas formatting", async () => {
                const template = "Tools: {{TOOL_SCHEMAS}}";
                (fs.readFile as any).mockResolvedValueOnce(template);

                const mockToolRegistry = {
                    getBuiltInDefinitions: vi.fn().mockReturnValue([
                        { name: "search", description: "Search tool" },
                    ]),
                    getSwarmToolDefinitions: vi.fn().mockReturnValue([
                        { name: "update_state", description: "Update swarm state" },
                    ]),
                };

                const context = createPromptContext({
                    toolRegistry: mockToolRegistry as any,
                });

                const result = await ResponseService.buildSystemMessage(context, {
                    templateIdentifier: "tools.txt",
                });

                expect(result).toContain("search");
                expect(result).toContain("update_state");
                expect(mockToolRegistry.getBuiltInDefinitions).toHaveBeenCalled();
                expect(mockToolRegistry.getSwarmToolDefinitions).toHaveBeenCalled();
            });
        });

        describe("SwarmStateAccessor integration", () => {
            it("should use SwarmStateAccessor for swarm variable resolution", async () => {
                const template = "Swarm Variable: {{swarm.customProperty}}";
                (fs.readFile as any).mockResolvedValueOnce(template);

                const mockAccessor = {
                    buildTriggerContext: vi.fn().mockReturnValue({ swarmId: "swarm123" }),
                    accessData: vi.fn().mockResolvedValue("custom_value"),
                };

                const context = createPromptContext({
                    swarmState: {
                        id: "swarm123",
                        name: "Test Swarm",
                        customProperty: "custom_value",
                    } as any,
                    bot: {
                        ...createPromptContext().bot,
                        config: {
                            ...createPromptContext().bot.config,
                            agentSpec: {
                                prompt: {
                                    source: "direct",
                                    content: template,
                                    mode: "replace",
                                    variables: {
                                        swarmVar: "swarm.customProperty",
                                    },
                                },
                            },
                        } as ChatConfigObject,
                    },
                });

                // Mock SwarmStateAccessor
                vi.doMock("../execution/shared/SwarmStateAccessor.js", () => ({
                    SwarmStateAccessor: vi.fn().mockImplementation(() => mockAccessor),
                }));

                const result = await ResponseService.buildSystemMessage(context, {
                    templateIdentifier: "swarm-var.txt",
                });

                expect(result).toContain("custom_value");
            });

            it("should fall back to direct access when SwarmStateAccessor fails", async () => {
                const template = "Direct Access: {{context.goal}}";
                (fs.readFile as any).mockResolvedValueOnce(template);

                const mockAccessor = {
                    buildTriggerContext: vi.fn().mockReturnValue({ swarmId: "swarm123" }),
                    accessData: vi.fn().mockRejectedValue(new Error("Accessor failed")),
                };

                const context = createPromptContext({
                    bot: {
                        ...createPromptContext().bot,
                        config: {
                            ...createPromptContext().bot.config,
                            agentSpec: {
                                prompt: {
                                    source: "direct",
                                    content: template,
                                    mode: "replace",
                                    variables: {
                                        goalVar: "context.goal",
                                    },
                                },
                            },
                        } as ChatConfigObject,
                    },
                });

                const result = await ResponseService.buildSystemMessage(context, {
                    templateIdentifier: "direct.txt",
                });

                expect(result).toContain("Help users with their questions");
                expect(logger.warn).toHaveBeenCalledWith(
                    expect.stringContaining("SwarmStateAccessor failed"),
                    expect.any(Object),
                );
            });
        });

        describe("security and validation", () => {
            it("should validate prompt safety", async () => {
                const suspiciousTemplate = "Execute: {{USER_INPUT}} <script>alert('xss')</script>";
                
                const context = createPromptContext();
                const result = await ResponseService.buildSystemMessage(context, {
                    directPromptContent: suspiciousTemplate,
                });

                // Should still process but log the validation
                expect(result).toBeDefined();
                expect(logger.debug).toHaveBeenCalledWith(
                    expect.stringContaining("Validating prompt safety"),
                    expect.any(Object),
                );
            });

            it("should handle sensitive data access logging", async () => {
                const template = "Secret: {{config.secrets.apiKey}}";
                
                const context = createPromptContext({
                    convoConfig: {
                        ...createPromptContext().convoConfig,
                        secrets: {
                            "config.secrets.*": {
                                type: "api_key",
                                accessLog: true,
                                requireConfirmation: false,
                            },
                        },
                    },
                    bot: {
                        ...createPromptContext().bot,
                        config: {
                            ...createPromptContext().bot.config,
                            agentSpec: {
                                prompt: {
                                    source: "direct",
                                    content: template,
                                    mode: "replace",
                                    variables: {
                                        secret: "config.secrets.apiKey",
                                    },
                                },
                            },
                        } as ChatConfigObject,
                    },
                });

                await ResponseService.buildSystemMessage(context);

                expect(logger.info).toHaveBeenCalledWith(
                    expect.stringContaining("Accessing sensitive data"),
                    expect.objectContaining({
                        path: "config.secrets.apiKey",
                        agentId: "bot123",
                        userId: "user123",
                    }),
                );
            });

            it("should handle unknown variable scopes gracefully", async () => {
                const template = "Unknown: {{unknown.scope.variable}}";
                
                const context = createPromptContext({
                    bot: {
                        ...createPromptContext().bot,
                        config: {
                            ...createPromptContext().bot.config,
                            agentSpec: {
                                prompt: {
                                    source: "direct",
                                    content: template,
                                    mode: "replace",
                                    variables: {
                                        unknownVar: "unknown.scope.variable",
                                    },
                                },
                            },
                        } as ChatConfigObject,
                    },
                });

                await expect(async () => {
                    await ResponseService.buildSystemMessage(context);
                }).rejects.toThrow("Unknown variable scope: unknown");

                expect(logger.warn).toHaveBeenCalledWith(
                    expect.stringContaining("Failed to resolve variable"),
                    expect.any(Object),
                );
            });
        });

        describe("role-specific instructions", () => {
            it("should include recruitment rules for leadership roles", async () => {
                const leadershipRoles = ["leader", "coordinator", "delegator"];
                
                for (const role of leadershipRoles) {
                    const context = createPromptContext({
                        bot: { ...createPromptContext().bot, role },
                    });

                    const result = await ResponseService.buildSystemMessage(context, {
                        directPromptContent: "Role: {{BOT.role}} {{ROLE_SPECIFIC_INSTRUCTIONS}}",
                    });

                    expect(result).toContain(`Role: ${role}`);
                    expect(result).toContain("## Recruitment rule:");
                    expect(result).toContain("update_swarm_shared_state");
                }
            });

            it("should not include recruitment rules for non-leadership roles", async () => {
                const nonLeadershipRoles = ["worker", "specialist", "analyst"];
                
                for (const role of nonLeadershipRoles) {
                    const context = createPromptContext({
                        bot: { ...createPromptContext().bot, role },
                    });

                    const result = await ResponseService.buildSystemMessage(context, {
                        directPromptContent: "Role: {{BOT.role}} {{ROLE_SPECIFIC_INSTRUCTIONS}}",
                    });

                    expect(result).toContain(`Role: ${role}`);
                    expect(result).not.toContain("## Recruitment rule:");
                    expect(result).toContain("Perform tasks according to your role");
                }
            });
        });

        describe("cache management", () => {
            it("should clear template cache when requested", () => {
                // Populate cache
                ResponseService.buildSystemMessage(createPromptContext(), {
                    directPromptContent: "Test content",
                });

                const initialStats = ResponseService.getCacheStats();
                expect(initialStats.enabled).toBe(true);

                ResponseService.clearTemplateCache();

                // Cache should be cleared but still enabled
                const clearedStats = ResponseService.getCacheStats();
                expect(clearedStats.size).toBe(0);
                expect(clearedStats.enabled).toBe(true);
            });

            it("should provide cache statistics", () => {
                ResponseService.clearTemplateCache();
                
                const stats = ResponseService.getCacheStats();
                expect(stats).toHaveProperty("size");
                expect(stats).toHaveProperty("enabled");
                expect(typeof stats.size).toBe("number");
                expect(typeof stats.enabled).toBe("boolean");
            });
        });

        describe("error handling and edge cases", () => {
            it("should handle null/undefined context gracefully", async () => {
                await expect(async () => {
                    await ResponseService.buildSystemMessage(null as any);
                }).rejects.toThrow();
            });

            it("should handle malformed agent spec", async () => {
                const context = createPromptContext({
                    bot: {
                        ...createPromptContext().bot,
                        config: {
                            ...createPromptContext().bot.config,
                            agentSpec: {
                                prompt: {
                                    source: "direct",
                                    // Missing required content field
                                    mode: "replace",
                                },
                            },
                        } as any,
                    },
                });

                await expect(async () => {
                    await ResponseService.buildSystemMessage(context);
                }).rejects.toThrow("Direct prompt source requires content field");
            });

            it("should handle empty template gracefully", async () => {
                const result = await ResponseService.buildSystemMessage(createPromptContext(), {
                    directPromptContent: "",
                });

                expect(result).toBe(""); // Should return empty string, not crash
            });

            it("should handle circular variable references", async () => {
                const template = "Circular: {{VAR1}}";
                
                const context = createPromptContext({
                    bot: {
                        ...createPromptContext().bot,
                        config: {
                            ...createPromptContext().bot.config,
                            agentSpec: {
                                prompt: {
                                    source: "direct",
                                    content: template,
                                    mode: "replace",
                                    variables: {
                                        VAR1: "context.VAR2",
                                        VAR2: "context.VAR1", // Circular reference
                                    },
                                },
                            },
                        } as ChatConfigObject,
                    },
                });

                // Should not hang or crash
                const result = await ResponseService.buildSystemMessage(context);
                expect(result).toBeDefined();
            });
        });
    });

    describe("swarm state formatting", () => {
        it("should format swarm state correctly", async () => {
            const swarmState = {
                id: "swarm123",
                name: "Test Swarm",
                config: { goal: "Test goal" },
                agentPool: [
                    { id: "agent1", role: "worker", status: "active" },
                    { id: "agent2", role: "coordinator", status: "idle" },
                ],
                subtasks: [
                    { id: "task1", description: "Do something", status: "in_progress" },
                    { id: "task2", description: "Do another thing", status: "todo" },
                ],
                environment: {
                    variables: { key: "value" },
                    secrets: { secret: "hidden" },
                },
            };

            (SwarmStateAccessor.get as any).mockReturnValueOnce({
                getState: vi.fn().mockResolvedValueOnce(swarmState),
            });

            const params: ResponseGenerationParams = {
                context: mockContext,
            };

            await responseService.generateResponse(params);

            // Verify swarm state was accessed
            const accessor = SwarmStateAccessor.get();
            expect(accessor.getState).toHaveBeenCalledWith("swarm123");
        });

        it("should truncate long swarm state sections", async () => {
            const veryLongArray = Array(1000).fill({
                id: "task",
                description: "A very long task description that repeats",
                status: "todo",
            });

            const swarmState = {
                id: "swarm123",
                name: "Test Swarm",
                config: {},
                agentPool: [],
                subtasks: veryLongArray,
                environment: {},
            };

            (SwarmStateAccessor.get as any).mockReturnValueOnce({
                getState: vi.fn().mockResolvedValueOnce(swarmState),
            });

            const params: ResponseGenerationParams = {
                context: mockContext,
            };

            const result = await responseService.generateResponse(params);

            expect(result.success).toBe(true);
            // Should not fail despite very long state
        });
    });

    describe("error handling edge cases", () => {
        it("should handle null bot config gracefully", async () => {
            const params: ResponseGenerationParams = {
                context: {
                    ...mockContext,
                    bot: {
                        ...mockContext.bot,
                        config: null as any,
                    },
                },
            };

            const result = await responseService.generateResponse(params);

            expect(result.success).toBe(false);
            expect(result.error?.code).toBe("INVALID_CONTEXT");
        });

        it("should handle missing user data", async () => {
            const params: ResponseGenerationParams = {
                context: {
                    ...mockContext,
                    userData: null as any,
                },
            };

            const result = await responseService.generateResponse(params);

            expect(result.success).toBe(false);
            expect(result.error?.code).toBe("INVALID_CONTEXT");
        });

        it("should handle LLM timeout", async () => {
            // Mock delayed LLM response
            (mockLlmRouter.complete as any).mockImplementationOnce(
                () => new Promise((resolve) => {
                    setTimeout(() => {
                        resolve({
                            ok: false,
                            error: {
                                code: "TIMEOUT",
                                message: "Request timed out",
                                creditsUsed: "0",
                            },
                        });
                    }, 100);
                }),
            );

            const params: ResponseGenerationParams = {
                context: {
                    ...mockContext,
                    resourceLimits: {
                        ...mockContext.resourceLimits,
                        timeoutMs: 50, // Very short timeout
                    },
                },
            };

            const result = await responseService.generateResponse(params);

            expect(result.success).toBe(false);
            expect(result.error?.code).toBe("TIMEOUT");
        });
    });

    describe("Enhanced Edge Cases - Complex Tool Execution Chains", () => {
        it("should handle cascading tool failures with partial recovery", async () => {
            // First LLM call requests multiple dependent tools
            (mockLlmRouter.complete as any).mockResolvedValueOnce({
                ok: true,
                data: {
                    messageContent: "I'll need to fetch data and then process it.",
                    toolCalls: [
                        {
                            id: "fetch_call",
                            name: "search",
                            arguments: { query: "initial data" },
                        },
                        {
                            id: "process_call", 
                            name: "process_data",
                            arguments: { depends_on: "fetch_call" },
                        },
                    ],
                    creditsUsed: "50",
                    tokensUsed: { prompt: 100, completion: 30, total: 130 },
                },
            });

            // First tool succeeds, second fails
            (mockToolRunner.run as any)
                .mockResolvedValueOnce({
                    ok: true,
                    data: { output: "fetched data", creditsUsed: "10" },
                })
                .mockResolvedValueOnce({
                    ok: false,
                    error: {
                        code: "TOOL_ERROR",
                        message: "Processing failed",
                        creditsUsed: "5",
                    },
                });

            // Recovery LLM call
            (mockLlmRouter.complete as any).mockResolvedValueOnce({
                ok: true,
                data: {
                    messageContent: "I was able to fetch the data but processing failed. Here's what I found from the initial search.",
                    toolCalls: [],
                    creditsUsed: "40",
                    tokensUsed: { prompt: 150, completion: 35, total: 185 },
                },
            });

            const params: ResponseGenerationParams = {
                context: mockContext,
            };

            const result = await responseService.generateResponse(params);

            expect(result.success).toBe(true);
            expect(result.toolCalls).toHaveLength(2);
            expect(result.toolCalls[0].result).toBe("fetched data");
            expect(result.toolCalls[1].error).toBe("Processing failed");
            expect(result.resourcesUsed.creditsUsed).toBe("105"); // 50 + 10 + 5 + 40
            expect(result.message).toContain("processing failed");
        });

        it("should prevent infinite tool call loops with circuit breaker", async () => {
            let callCount = 0;
            
            // Mock LLM to create a potential infinite loop
            (mockLlmRouter.complete as any).mockImplementation(() => {
                callCount++;
                return Promise.resolve({
                    ok: true,
                    data: {
                        messageContent: `Call ${callCount}`,
                        toolCalls: [
                            {
                                id: `call_${callCount}`,
                                name: "search",
                                arguments: { query: `recursive_query_${callCount}` },
                            },
                        ],
                        creditsUsed: "10",
                        tokensUsed: { prompt: 50, completion: 20, total: 70 },
                    },
                });
            });

            const params: ResponseGenerationParams = {
                context: mockContext,
            };

            const result = await responseService.generateResponse(params);

            // Should stop at MAX_TOOL_ITERATIONS (10)
            expect(result.success).toBe(true);
            expect(result.toolCalls).toHaveLength(10);
            expect(callCount).toBe(10); // Should not exceed the limit
        });

        it("should handle tool execution with mixed success/failure patterns", async () => {
            // Mock LLM response with batch of tools
            (mockLlmRouter.complete as any).mockResolvedValueOnce({
                ok: true,
                data: {
                    messageContent: "Processing multiple tasks simultaneously.",
                    toolCalls: [
                        { id: "task1", name: "search", arguments: { query: "query1" } },
                        { id: "task2", name: "search", arguments: { query: "query2" } },
                        { id: "task3", name: "search", arguments: { query: "query3" } },
                        { id: "task4", name: "search", arguments: { query: "query4" } },
                    ],
                    creditsUsed: "80",
                    tokensUsed: { prompt: 200, completion: 60, total: 260 },
                },
            });

            // Mix of successes and failures
            (mockToolRunner.run as any)
                .mockResolvedValueOnce({ ok: true, data: { output: "success1", creditsUsed: "5" } })
                .mockResolvedValueOnce({ ok: false, error: { code: "ERROR", message: "fail1", creditsUsed: "3" } })
                .mockResolvedValueOnce({ ok: true, data: { output: "success2", creditsUsed: "7" } })
                .mockResolvedValueOnce({ ok: false, error: { code: "ERROR", message: "fail2", creditsUsed: "2" } });

            // Final response
            (mockLlmRouter.complete as any).mockResolvedValueOnce({
                ok: true,
                data: {
                    messageContent: "Completed 2 out of 4 tasks successfully.",
                    toolCalls: [],
                    creditsUsed: "30",
                    tokensUsed: { prompt: 250, completion: 40, total: 290 },
                },
            });

            const params: ResponseGenerationParams = {
                context: mockContext,
            };

            const result = await responseService.generateResponse(params);

            expect(result.success).toBe(true);
            expect(result.toolCalls).toHaveLength(4);
            
            // Check specific success/failure pattern
            expect(result.toolCalls[0].result).toBe("success1");
            expect(result.toolCalls[1].error).toBe("fail1");
            expect(result.toolCalls[2].result).toBe("success2");
            expect(result.toolCalls[3].error).toBe("fail2");
            
            expect(result.resourcesUsed.creditsUsed).toBe("127"); // 80 + 5 + 3 + 7 + 2 + 30
        });
    });

    describe("Enhanced Edge Cases - Resource Management Under Stress", () => {
        it("should handle credit exhaustion during tool execution", async () => {
            // Start with limited credits
            const limitedContext = {
                ...mockContext,
                resourceLimits: {
                    ...mockContext.resourceLimits,
                    maxCredits: "60", // Very limited
                },
            };

            // LLM call uses most credits
            (mockLlmRouter.complete as any).mockResolvedValueOnce({
                ok: true,
                data: {
                    messageContent: "Starting expensive operations.",
                    toolCalls: [
                        { id: "expensive1", name: "search", arguments: { query: "complex query" } },
                        { id: "expensive2", name: "search", arguments: { query: "another query" } },
                    ],
                    creditsUsed: "45", // Uses most of the budget
                    tokensUsed: { prompt: 100, completion: 30, total: 130 },
                },
            });

            // First tool succeeds but uses remaining credits
            (mockToolRunner.run as any).mockResolvedValueOnce({
                ok: true,
                data: { output: "result1", creditsUsed: "15" }, // Total now at 60/60
            });

            // Second tool should fail due to credit exhaustion
            (mockToolRunner.run as any).mockResolvedValueOnce({
                ok: false,
                error: {
                    code: "INSUFFICIENT_CREDITS",
                    message: "Credit limit exceeded",
                    creditsUsed: "0",
                },
            });

            const params: ResponseGenerationParams = {
                context: limitedContext,
            };

            const result = await responseService.generateResponse(params);

            expect(result.success).toBe(true);
            expect(result.toolCalls).toHaveLength(2);
            expect(result.toolCalls[0].result).toBe("result1");
            expect(result.toolCalls[1].error).toBe("Credit limit exceeded");
            expect(result.resourcesUsed.creditsUsed).toBe("60");
        });

        it("should handle token limit boundary conditions", async () => {
            const tokenLimitedContext = {
                ...mockContext,
                resourceLimits: {
                    ...mockContext.resourceLimits,
                    maxTokens: 200, // Very limited tokens
                },
            };

            // Mock LLM to approach token limit
            (mockLlmRouter.complete as any).mockResolvedValueOnce({
                ok: true,
                data: {
                    messageContent: "This is a response that uses many tokens to test the boundary conditions.",
                    toolCalls: [],
                    creditsUsed: "50",
                    tokensUsed: { prompt: 100, completion: 95, total: 195 }, // Near limit
                },
            });

            const params: ResponseGenerationParams = {
                context: tokenLimitedContext,
            };

            const result = await responseService.generateResponse(params);

            expect(result.success).toBe(true);
            expect(result.resourcesUsed.tokensUsed?.total).toBe(195);
            expect(result.resourcesUsed.tokensUsed?.total).toBeLessThanOrEqual(200);
        });

        it("should handle concurrent resource allocation conflicts", async () => {
            // Simulate concurrent requests by running multiple generateResponse calls
            const concurrentPromises = Array.from({ length: 5 }, (_, i) => {
                const contextCopy = {
                    ...mockContext,
                    bot: { ...mockContext.bot, id: `bot${i}` },
                };

                // Each request uses some resources
                (mockLlmRouter.complete as any).mockResolvedValue({
                    ok: true,
                    data: {
                        messageContent: `Response from bot ${i}`,
                        toolCalls: [],
                        creditsUsed: "20",
                        tokensUsed: { prompt: 50, completion: 25, total: 75 },
                    },
                });

                return responseService.generateResponse({ context: contextCopy });
            });

            const results = await Promise.all(concurrentPromises);

            // All should succeed
            results.forEach((result, i) => {
                expect(result.success).toBe(true);
                expect(result.botId).toBe(`bot${i}`);
                expect(result.resourcesUsed.creditsUsed).toBe("20");
            });
        });
    });

    describe("Enhanced Edge Cases - Template Security & Validation", () => {
        it("should sanitize potentially dangerous template variables", async () => {
            const suspiciousTemplate = `
                Goal: {{GOAL}}
                Injected: {{USER_INPUT}}
                Script: <script>alert('xss')</script>
                SQL: '; DROP TABLE users; --
            `;

            const params: ResponseGenerationParams = {
                context: {
                    ...mockContext,
                    bot: {
                        ...mockContext.bot,
                        config: {
                            ...mockContext.bot.config,
                            agentSpec: {
                                prompt: {
                                    source: "direct",
                                    content: suspiciousTemplate,
                                    mode: "replace",
                                },
                            },
                        } as any,
                    },
                },
            };

            const result = await responseService.generateResponse(params);

            // Should complete successfully but log security validation
            expect(result.success).toBe(true);
            expect(logger.debug).toHaveBeenCalledWith(
                expect.stringContaining("Validating prompt safety"),
                expect.any(Object),
            );
        });

        it("should handle variable injection attempts", async () => {
            const injectionTemplate = `
                Normal: {{GOAL}}
                Injection: {{__proto__.constructor.prototype}}
                Path Traversal: {{../../../config.secrets}}
                Function Call: {{console.log('injected')}}
            `;

            const params: ResponseGenerationParams = {
                context: {
                    ...mockContext,
                    bot: {
                        ...mockContext.bot,
                        config: {
                            ...mockContext.bot.config,
                            agentSpec: {
                                prompt: {
                                    source: "direct",
                                    content: injectionTemplate,
                                    mode: "replace",
                                },
                            },
                        } as any,
                    },
                },
            };

            const result = await responseService.generateResponse(params);

            // Should handle injection attempts gracefully
            expect(result.success).toBe(true);
            // Variables like __proto__ should not be resolved to dangerous values
            expect(result.message).not.toContain("constructor.prototype");
        });

        it("should validate template access permissions", async () => {
            const restrictedTemplate = `
                Public: {{GOAL}}
                Restricted: {{config.secrets.apiKey}}
                Internal: {{context.userData.sensitiveData}}
            `;

            const params: ResponseGenerationParams = {
                context: {
                    ...mockContext,
                    bot: {
                        ...mockContext.bot,
                        config: {
                            ...mockContext.bot.config,
                            agentSpec: {
                                prompt: {
                                    source: "direct",
                                    content: restrictedTemplate,
                                    mode: "replace",
                                    variables: {
                                        secretKey: "config.secrets.apiKey",
                                        sensitiveData: "context.userData.sensitiveData",
                                    },
                                },
                            },
                        } as any,
                    },
                },
            };

            const result = await responseService.generateResponse(params);

            // Should log access to sensitive data
            expect(result.success).toBe(true);
            expect(logger.info).toHaveBeenCalledWith(
                expect.stringContaining("Accessing sensitive data"),
                expect.objectContaining({
                    agentId: "bot123",
                    userId: "user123",
                }),
            );
        });
    });

    describe("Enhanced Edge Cases - SwarmState Integration Stress", () => {
        it("should handle very large swarm states efficiently", async () => {
            // Create a large swarm state with many subtasks and resources
            const largeSubtasks = Array.from({ length: 1000 }, (_, i) => ({
                id: `task_${i}`,
                description: `Large task ${i} with extensive description that contains many details and requirements that need to be processed efficiently by the system`,
                status: i % 3 === 0 ? "completed" : i % 3 === 1 ? "in_progress" : "todo",
                dependencies: i > 0 ? [`task_${i - 1}`] : [],
                metadata: {
                    priority: i % 5,
                    estimatedDuration: i * 1000,
                    resources: Array.from({ length: 10 }, (_, j) => `resource_${i}_${j}`),
                },
            }));

            const largeSwarmState = {
                id: "large_swarm",
                name: "Large Test Swarm",
                config: { goal: "Process large dataset" },
                agentPool: Array.from({ length: 100 }, (_, i) => ({
                    id: `agent_${i}`,
                    role: `role_${i % 10}`,
                    status: "active",
                })),
                subtasks: largeSubtasks,
                environment: {
                    variables: Object.fromEntries(
                        Array.from({ length: 500 }, (_, i) => [`var_${i}`, `value_${i}`]),
                    ),
                },
            };

            (SwarmStateAccessor.get as any).mockReturnValueOnce({
                getState: vi.fn().mockResolvedValueOnce(largeSwarmState),
            });

            const params: ResponseGenerationParams = {
                context: mockContext,
            };

            const startTime = Date.now();
            const result = await responseService.generateResponse(params);
            const duration = Date.now() - startTime;

            expect(result.success).toBe(true);
            expect(duration).toBeLessThan(5000); // Should complete within 5 seconds
            
            // Verify swarm state was accessed but response is reasonable size
            const accessor = SwarmStateAccessor.get();
            expect(accessor.getState).toHaveBeenCalledWith("swarm123");
        });

        it("should handle corrupted swarm state gracefully", async () => {
            // Mock corrupted state that causes SwarmStateAccessor to fail
            (SwarmStateAccessor.get as any).mockReturnValueOnce({
                getState: vi.fn().mockRejectedValueOnce(new Error("Swarm state corrupted")),
            });

            const params: ResponseGenerationParams = {
                context: mockContext,
            };

            const result = await responseService.generateResponse(params);

            // Should handle corruption gracefully and continue with default state
            expect(result.success).toBe(true);
            expect(logger.warn).toHaveBeenCalledWith(
                expect.stringContaining("Failed to load swarm state"),
                expect.any(Object),
            );
        });

        it("should handle swarm state updates during execution", async () => {
            let stateVersion = 1;
            
            // Mock state that changes during execution
            (SwarmStateAccessor.get as any).mockReturnValue({
                getState: vi.fn().mockImplementation(() => {
                    const state = {
                        id: "dynamic_swarm",
                        name: "Dynamic Swarm",
                        version: stateVersion++,
                        config: { goal: `Updated goal v${stateVersion}` },
                        agentPool: [],
                        subtasks: [],
                        environment: {},
                    };
                    return Promise.resolve(state);
                }),
            });

            // Mock LLM to require multiple state accesses
            (mockLlmRouter.complete as any).mockResolvedValueOnce({
                ok: true,
                data: {
                    messageContent: "Checking swarm state multiple times.",
                    toolCalls: [
                        { id: "check1", name: "search", arguments: { query: "state check 1" } },
                    ],
                    creditsUsed: "30",
                    tokensUsed: { prompt: 100, completion: 25, total: 125 },
                },
            });

            (mockLlmRouter.complete as any).mockResolvedValueOnce({
                ok: true,
                data: {
                    messageContent: "State has been updated during execution.",
                    toolCalls: [],
                    creditsUsed: "20",
                    tokensUsed: { prompt: 120, completion: 20, total: 140 },
                },
            });

            const params: ResponseGenerationParams = {
                context: mockContext,
            };

            const result = await responseService.generateResponse(params);

            expect(result.success).toBe(true);
            // Should handle dynamic state updates during execution
            const accessor = SwarmStateAccessor.get();
            expect(accessor.getState).toHaveBeenCalledTimes(2); // Once for each LLM call
        });
    });

    describe("Model Selection - Network-aware Strategy-based Selection", () => {
        let mockNetworkMonitor: any;
        let mockRegistry: any;
        let mockStrategy: any;

        beforeEach(() => {
            mockNetworkMonitor = {
                getState: vi.fn(),
            };
            vi.mocked(NetworkMonitor.getInstance).mockReturnValue(mockNetworkMonitor);

            mockRegistry = {
                getBestService: vi.fn(),
            };
            vi.mocked(AIServiceRegistry.get).mockReturnValue(mockRegistry);

            mockStrategy = {
                selectModel: vi.fn(),
            };
            vi.mocked(ModelSelectionStrategyFactory.getStrategy).mockReturnValue(mockStrategy);

            // Mock other dependencies
            const mockLlmRouter = {
                stream: vi.fn(),
            };
            const mockToolRunner = {
                executeTools: vi.fn(),
            };

            responseService = new ResponseService({
                llmRouter: mockLlmRouter as any,
                toolRunner: mockToolRunner as any,
            });
        });

        afterEach(() => {
            vi.clearAllMocks();
        });

        describe("selectModel", () => {
            it("should use chat modelConfig when available", async () => {
                const context: ResponseContext = {
                    chatConfig: {
                        modelConfig: {
                            strategy: ModelStrategy.QUALITY_FIRST,
                            preferredModel: "gpt-4o",
                            offlineOnly: false,
                        },
                    },
                    bot: {
                        config: {
                            modelConfig: {
                                strategy: ModelStrategy.COST_OPTIMIZED,
                                preferredModel: "gpt-4o-mini",
                                offlineOnly: false,
                            },
                        },
                    },
                    teamConfig: {
                        modelConfig: {
                            strategy: ModelStrategy.LOCAL_FIRST,
                            preferredModel: "llama3.1:8b",
                            offlineOnly: false,
                        },
                    },
                } as any;

                mockNetworkMonitor.getState.mockResolvedValue({
                    isOnline: true,
                    cloudServicesReachable: true,
                    localServicesReachable: true,
                });

                mockStrategy.selectModel.mockResolvedValue("gpt-4o");

                const result = await responseService["selectModel"](context);

                expect(ModelSelectionStrategyFactory.getStrategy).toHaveBeenCalledWith(ModelStrategy.QUALITY_FIRST);
                expect(mockStrategy.selectModel).toHaveBeenCalledWith({
                    modelConfig: {
                        strategy: ModelStrategy.QUALITY_FIRST,
                        preferredModel: "gpt-4o",
                        offlineOnly: false,
                    },
                    networkState: expect.any(Object),
                    registry: mockRegistry,
                    userCredits: undefined,
                });
                expect(result).toBe("gpt-4o");
            });

            it("should use bot modelConfig when chat config not available", async () => {
                const context: ResponseContext = {
                    bot: {
                        config: {
                            modelConfig: {
                                strategy: ModelStrategy.COST_OPTIMIZED,
                                preferredModel: "gpt-4o-mini",
                                offlineOnly: false,
                            },
                        },
                    },
                    teamConfig: {
                        modelConfig: {
                            strategy: ModelStrategy.LOCAL_FIRST,
                            preferredModel: "llama3.1:8b",
                            offlineOnly: false,
                        },
                    },
                } as any;

                mockNetworkMonitor.getState.mockResolvedValue({
                    isOnline: true,
                    cloudServicesReachable: true,
                    localServicesReachable: true,
                });

                mockStrategy.selectModel.mockResolvedValue("gpt-4o-mini");

                const result = await responseService["selectModel"](context);

                expect(ModelSelectionStrategyFactory.getStrategy).toHaveBeenCalledWith(ModelStrategy.COST_OPTIMIZED);
                expect(result).toBe("gpt-4o-mini");
            });

            it("should use team modelConfig when chat and bot configs not available", async () => {
                const context: ResponseContext = {
                    bot: {
                        config: {},
                    },
                    teamConfig: {
                        modelConfig: {
                            strategy: ModelStrategy.LOCAL_FIRST,
                            preferredModel: "llama3.1:8b",
                            offlineOnly: false,
                        },
                    },
                } as any;

                mockNetworkMonitor.getState.mockResolvedValue({
                    isOnline: true,
                    cloudServicesReachable: true,
                    localServicesReachable: true,
                });

                mockStrategy.selectModel.mockResolvedValue("llama3.1:8b");

                const result = await responseService["selectModel"](context);

                expect(ModelSelectionStrategyFactory.getStrategy).toHaveBeenCalledWith(ModelStrategy.LOCAL_FIRST);
                expect(result).toBe("llama3.1:8b");
            });

            it("should use default config when no configs available", async () => {
                const context: ResponseContext = {
                    bot: {
                        config: {},
                    },
                } as any;

                mockNetworkMonitor.getState.mockResolvedValue({
                    isOnline: true,
                    cloudServicesReachable: true,
                    localServicesReachable: true,
                });

                mockStrategy.selectModel.mockResolvedValue("gpt-4");

                const result = await responseService["selectModel"](context);

                expect(ModelSelectionStrategyFactory.getStrategy).toHaveBeenCalledWith(ModelStrategy.FALLBACK);
                expect(mockStrategy.selectModel).toHaveBeenCalledWith({
                    modelConfig: {
                        strategy: ModelStrategy.FALLBACK,
                        preferredModel: "gpt-4",
                        offlineOnly: false,
                    },
                    networkState: expect.any(Object),
                    registry: mockRegistry,
                    userCredits: undefined,
                });
                expect(result).toBe("gpt-4");
            });

            it("should pass user credits when available", async () => {
                const context: ResponseContext = {
                    userData: {
                        credits: 1000,
                    },
                    bot: {
                        config: {},
                    },
                } as any;

                mockNetworkMonitor.getState.mockResolvedValue({
                    isOnline: true,
                    cloudServicesReachable: true,
                    localServicesReachable: true,
                });

                mockStrategy.selectModel.mockResolvedValue("gpt-4");

                await responseService["selectModel"](context);

                expect(mockStrategy.selectModel).toHaveBeenCalledWith({
                    modelConfig: expect.any(Object),
                    networkState: expect.any(Object),
                    registry: mockRegistry,
                    userCredits: 1000,
                });
            });

            it("should fallback to default model on strategy error", async () => {
                const context: ResponseContext = {
                    bot: {
                        config: {},
                    },
                } as any;

                mockNetworkMonitor.getState.mockResolvedValue({
                    isOnline: true,
                    cloudServicesReachable: true,
                    localServicesReachable: true,
                });

                mockStrategy.selectModel.mockRejectedValue(new Error("Strategy failed"));

                const result = await responseService["selectModel"](context);

                expect(result).toBe("gpt-4");
            });
        });

        describe("getEffectiveModelConfig", () => {
            it("should return chat config when available", () => {
                const context: ResponseContext = {
                    chatConfig: {
                        modelConfig: {
                            strategy: ModelStrategy.QUALITY_FIRST,
                            preferredModel: "gpt-4o",
                            offlineOnly: false,
                        },
                    },
                    bot: {
                        config: {
                            modelConfig: {
                                strategy: ModelStrategy.COST_OPTIMIZED,
                                preferredModel: "gpt-4o-mini",
                                offlineOnly: false,
                            },
                        },
                    },
                } as any;

                const result = responseService["getEffectiveModelConfig"](context);
                expect(result).toEqual({
                    strategy: ModelStrategy.QUALITY_FIRST,
                    preferredModel: "gpt-4o",
                    offlineOnly: false,
                });
            });

            it("should return bot config when chat config not available", () => {
                const context: ResponseContext = {
                    bot: {
                        config: {
                            modelConfig: {
                                strategy: ModelStrategy.COST_OPTIMIZED,
                                preferredModel: "gpt-4o-mini",
                                offlineOnly: false,
                            },
                        },
                    },
                    teamConfig: {
                        modelConfig: {
                            strategy: ModelStrategy.LOCAL_FIRST,
                            preferredModel: "llama3.1:8b",
                            offlineOnly: false,
                        },
                    },
                } as any;

                const result = responseService["getEffectiveModelConfig"](context);
                expect(result).toEqual({
                    strategy: ModelStrategy.COST_OPTIMIZED,
                    preferredModel: "gpt-4o-mini",
                    offlineOnly: false,
                });
            });

            it("should return team config when chat and bot configs not available", () => {
                const context: ResponseContext = {
                    bot: {
                        config: {},
                    },
                    teamConfig: {
                        modelConfig: {
                            strategy: ModelStrategy.LOCAL_FIRST,
                            preferredModel: "llama3.1:8b",
                            offlineOnly: false,
                        },
                    },
                } as any;

                const result = responseService["getEffectiveModelConfig"](context);
                expect(result).toEqual({
                    strategy: ModelStrategy.LOCAL_FIRST,
                    preferredModel: "llama3.1:8b",
                    offlineOnly: false,
                });
            });

            it("should return default config when no configs available", () => {
                const context: ResponseContext = {
                    bot: {
                        config: {},
                    },
                } as any;

                const result = responseService["getEffectiveModelConfig"](context);
                expect(result).toEqual({
                    strategy: ModelStrategy.FALLBACK,
                    preferredModel: "gpt-4",
                    offlineOnly: false,
                });
            });
        });

        describe("getModelConfigSource", () => {
            it("should return correct source for each config type", () => {
                // Chat config
                let context: ResponseContext = {
                    chatConfig: { modelConfig: {} },
                    bot: { config: {} },
                } as any;
                expect(responseService["getModelConfigSource"](context)).toBe("chat");

                // Bot config
                context = {
                    bot: { config: { modelConfig: {} } },
                    teamConfig: { modelConfig: {} },
                } as any;
                expect(responseService["getModelConfigSource"](context)).toBe("bot");

                // Team config
                context = {
                    bot: { config: {} },
                    teamConfig: { modelConfig: {} },
                } as any;
                expect(responseService["getModelConfigSource"](context)).toBe("team");

                // Default
                context = {
                    bot: { config: {} },
                } as any;
                expect(responseService["getModelConfigSource"](context)).toBe("default");
            });
        });
    });
});
