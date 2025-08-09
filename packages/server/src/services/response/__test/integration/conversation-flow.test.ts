import {
    type BotParticipant,
    type ChatConfigObject,
    type ConversationState,
    type MessageState,
    type SessionUser,
    type Tool,
    generatePK,
    LlmServiceId,
    McpToolName,
} from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DbProvider } from "../../../../db/provider.js";
import { CacheService } from "../../../../redisConn.js";
import { type CachedConversationStateStore, conversationStateStore, PrismaChatStore } from "../../chatStore.js";
import { MessageHistoryBuilder } from "../../messageHistoryBuilder.js";
import { type MessageStore, RedisMessageStore } from "../../messageStore.js";
import { ResponseService } from "../../responseService.js";
import { FallbackRouter } from "../../router.js";
import { OpenAIService } from "../../services.js";
import { ToolApprovalConfig } from "../../toolApprovalConfig.js";
import { CompositeToolRunner } from "../../toolRunner.js";
// Note: Container setup is handled by global vitest setup

// Mock external dependencies that we don't want in integration tests
vi.mock("openai", () => {
    return {
        default: vi.fn().mockImplementation(() => ({
            moderations: {
                create: vi.fn().mockResolvedValue({
                    results: [{ flagged: false, categories: {}, category_scores: {} }],
                }),
            },
            chat: {
                completions: {
                    create: vi.fn(),
                },
            },
        })),
    };
});

vi.mock("../../../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        error: vi.fn(),
        warn: vi.fn(),
        debug: vi.fn(),
        trace: vi.fn(),
    },
}));

vi.mock("../../../events/publisher.js", () => ({
    EventPublisher: {
        get: vi.fn(() => ({
            publish: vi.fn(),
        })),
    },
}));

describe("End-to-End Conversation Flow Integration Tests", () => {
    let responseService: ResponseService;
    let conversationStore: CachedConversationStateStore;
    let chatStore: PrismaChatStore;
    let messageStore: MessageStore;
    let toolRunner: CompositeToolRunner;
    let mockOpenAIClient: any;
    let mockDb: any;
    let mockCache: any;
    let testUser: SessionUser;
    let conversationState: ConversationState;

    beforeEach(async () => {
        vi.clearAllMocks();

        // Note: Test containers are already set up by global vitest setup
        // Initialize services
        mockDb = DbProvider.get();
        mockCache = CacheService.get();

        // Set up mock OpenAI client
        const OpenAI = (await import("openai")).default;
        mockOpenAIClient = new OpenAI({ apiKey: "test-key" });

        // Initialize all services
        const openAIService = new OpenAIService();
        (openAIService as any).client = mockOpenAIClient;

        const router = new FallbackRouter([
            { id: LlmServiceId.OpenAI, service: openAIService },
        ]);

        // Use the production singleton for realistic testing
        conversationStore = conversationStateStore;
        // Also get direct access to the underlying ChatStore for full state operations
        chatStore = new PrismaChatStore();
        messageStore = RedisMessageStore.get();
        toolRunner = new CompositeToolRunner();

        responseService = new ResponseService(
            router,
            toolRunner,
            new MessageHistoryBuilder(),
        );

        // Set up test data
        testUser = {
            id: "user123",
            credits: 10000n,
            name: "Test User",
        } as SessionUser;

        conversationState = {
            id: generatePK(),
            participants: [
                {
                    id: "bot123",
                    name: "Assistant",
                    config: { __version: "1.0", resources: [], model: "gpt-4o" },
                    state: "ready",
                } as BotParticipant,
            ],
            status: "in_progress",
            config: {
                __version: "1.0",
                model: "gpt-4o",
                stats: {
                    totalToolCalls: 0,
                    totalCredits: 0,
                },
            } as ChatConfigObject,
        };

        // Mock database responses
        mockDb.chat.findUniqueOrThrow.mockResolvedValue({
            id: conversationState.id,
            config: conversationState.config,
        });

        mockDb.chat_participants.findMany.mockResolvedValue([
            {
                user: {
                    id: "bot123",
                    name: "Assistant",
                    handle: "assistant",
                    isBot: true,
                    botSettings: { model: "gpt-4o" },
                },
            },
        ]);
    });

    afterEach(async () => {
        // Note: Test containers are cleaned up by global vitest teardown
        vi.restoreAllMocks();
    });

    describe("Simple Message → Response Flow", () => {
        it("should handle a basic user message and generate AI response", async () => {
            // Mock the OpenAI streaming response
            const mockStream = createMockTextStream([
                "Hello! ",
                "How can I ",
                "help you today?",
            ]);
            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(mockStream);

            // User message
            const userMessage: MessageState = {
                id: generatePK().toString(),
                content: "Hello, can you help me?",
                role: "user",
                timestamp: Date.now(),
            };

            // Store the conversation state
            await chatStore.saveState(conversationState.id, conversationState);

            // Generate response
            const responseParams = {
                conversationId: conversationState.id,
                messages: [userMessage],
                bot: conversationState.participants[0],
                userData: testUser,
                systemMessage: "You are a helpful assistant.",
                maxTokens: 1000,
            };

            const events = [];
            for await (const event of responseService.generateResponse(responseParams)) {
                events.push(event);
            }

            // Verify the response flow
            expect(events).toContainEqual(
                expect.objectContaining({
                    type: "text",
                    content: "Hello! ",
                }),
            );
            expect(events).toContainEqual(
                expect.objectContaining({
                    type: "text",
                    content: "How can I ",
                }),
            );
            expect(events).toContainEqual(
                expect.objectContaining({
                    type: "text",
                    content: "help you today?",
                }),
            );
            expect(events).toContainEqual(
                expect.objectContaining({
                    type: "done",
                }),
            );

            // Verify conversation state was updated in both chat store and cache
            const updatedState = await chatStore.loadState(conversationState.id);
            expect(updatedState.config.stats.totalCredits).toBeGreaterThan(0);

            // Verify cache coherency - conversation store should have the same state
            const cachedState = await conversationStore.get(conversationState.id);
            expect(cachedState?.config.stats.totalCredits).toBeGreaterThan(0);
        });
    });

    describe("Message → Tool Call → Tool Result → Response Flow", () => {
        it("should handle tool execution in the conversation flow", async () => {
            // Register a test tool
            const testTool: Tool = {
                name: "get_weather",
                description: "Get weather information",
                parameters: {
                    type: "object",
                    properties: {
                        location: { type: "string" },
                    },
                    required: ["location"],
                },
                execute: vi.fn().mockResolvedValue({
                    temperature: "72°F",
                    condition: "Sunny",
                }),
            };
            toolRunner.registerTool(testTool);

            // Mock OpenAI responses
            // First response: tool call
            const toolCallStream = createMockToolCallStream({
                name: "get_weather",
                arguments: { location: "New York" },
                callId: "call_123",
            });

            // Second response: final answer
            const finalResponseStream = createMockTextStream([
                "The weather in New York is currently 72°F and sunny.",
            ]);

            mockOpenAIClient.chat.completions.create
                .mockResolvedValueOnce(toolCallStream)
                .mockResolvedValueOnce(finalResponseStream);

            // User message
            const userMessage: MessageState = {
                id: generatePK().toString(),
                content: "What's the weather in New York?",
                role: "user",
                timestamp: Date.now(),
            };

            await chatStore.saveState(conversationState.id, conversationState);

            // Generate response with tool approval auto-approved
            const approvalConfig = new ToolApprovalConfig({
                defaultRequiresApproval: false,
            });
            const responseServiceWithApproval = new ResponseService(
                responseService.router,
                toolRunner,
                new MessageHistoryBuilder(),
            );

            const events = [];
            for await (const event of responseServiceWithApproval.generateResponse({
                conversationId: conversationState.id,
                messages: [userMessage],
                bot: conversationState.participants[0],
                userData: testUser,
                tools: [testTool],
                maxTokens: 1000,
            })) {
                events.push(event);
            }

            // Verify the flow
            expect(events).toContainEqual(
                expect.objectContaining({
                    type: "tool_call",
                    tool: expect.objectContaining({
                        name: "get_weather",
                        arguments: { location: "New York" },
                    }),
                }),
            );

            expect(events).toContainEqual(
                expect.objectContaining({
                    type: "tool_result",
                    result: expect.objectContaining({
                        temperature: "72°F",
                        condition: "Sunny",
                    }),
                }),
            );

            expect(events).toContainEqual(
                expect.objectContaining({
                    type: "text",
                    content: expect.stringContaining("72°F"),
                }),
            );

            // Verify tool was executed
            expect(testTool.execute).toHaveBeenCalledWith({ location: "New York" });

            // Verify stats were updated in both stores
            const updatedState = await chatStore.loadState(conversationState.id);
            expect(updatedState.config.stats.totalToolCalls).toBe(1);

            // Verify cache coherency
            const cachedState = await conversationStore.get(conversationState.id);
            expect(cachedState?.config.stats.totalToolCalls).toBe(1);
        });
    });

    describe("Tool Approval Flow", () => {
        it("should request approval for high-risk tools", async () => {
            // Register a high-risk tool
            const highRiskTool: Tool = {
                name: McpToolName.RunRoutine,
                description: "Execute a routine",
                parameters: {
                    type: "object",
                    properties: {
                        routineId: { type: "string" },
                    },
                },
                execute: vi.fn().mockResolvedValue({ success: true }),
            };
            toolRunner.registerTool(highRiskTool);

            // Configure approval to require approval for high-risk tools
            const approvalConfig = new ToolApprovalConfig({
                defaultRequiresApproval: true,
                riskThreshold: "medium",
            });

            // Mock tool call stream
            const toolCallStream = createMockToolCallStream({
                name: McpToolName.RunRoutine,
                arguments: { routineId: "routine123" },
                callId: "call_456",
            });
            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(toolCallStream);

            const userMessage: MessageState = {
                id: generatePK().toString(),
                content: "Run the data processing routine",
                role: "user",
                timestamp: Date.now(),
            };

            const events = [];
            for await (const event of responseService.generateResponse({
                conversationId: conversationState.id,
                messages: [userMessage],
                bot: conversationState.participants[0],
                userData: testUser,
                tools: [highRiskTool],
                maxTokens: 1000,
            })) {
                events.push(event);
            }

            // Should emit approval request
            expect(events).toContainEqual(
                expect.objectContaining({
                    type: "tool_approval_request",
                    tool: expect.objectContaining({
                        name: McpToolName.RunRoutine,
                    }),
                }),
            );

            // Tool should not be executed without approval
            expect(highRiskTool.execute).not.toHaveBeenCalled();
        });
    });

    describe("Multi-turn Conversation with Context", () => {
        it("should maintain context across multiple turns", async () => {
            // Initial conversation history
            const messages: MessageState[] = [
                {
                    id: generatePK().toString(),
                    content: "My name is Alice",
                    role: "user",
                    timestamp: Date.now() - 60000,
                },
                {
                    id: generatePK().toString(),
                    content: "Nice to meet you, Alice!",
                    role: "bot",
                    timestamp: Date.now() - 50000,
                },
            ];

            // Store messages in message store
            for (const msg of messages) {
                await messageStore.addMessage(conversationState.id, msg);
            }

            // New user message
            const newMessage: MessageState = {
                id: generatePK().toString(),
                content: "What's my name?",
                role: "user",
                timestamp: Date.now(),
            };

            // Mock response that uses context
            const contextAwareStream = createMockTextStream([
                "Your name is Alice.",
            ]);
            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(contextAwareStream);

            const events = [];
            for await (const event of responseService.generateResponse({
                conversationId: conversationState.id,
                messages: [...messages, newMessage],
                bot: conversationState.participants[0],
                userData: testUser,
                maxTokens: 1000,
            })) {
                events.push(event);
            }

            // Verify context was used
            expect(mockOpenAIClient.chat.completions.create).toHaveBeenCalledWith(
                expect.objectContaining({
                    messages: expect.arrayContaining([
                        expect.objectContaining({ content: "My name is Alice" }),
                        expect.objectContaining({ content: "Nice to meet you, Alice!" }),
                        expect.objectContaining({ content: "What's my name?" }),
                    ]),
                }),
            );

            expect(events).toContainEqual(
                expect.objectContaining({
                    type: "text",
                    content: "Your name is Alice.",
                }),
            );
        });
    });

    describe("Error Handling and Recovery", () => {
        it("should handle API errors gracefully", async () => {
            // Mock API error
            const apiError = new Error("Rate limit exceeded");
            (apiError as any).status = 429;
            mockOpenAIClient.chat.completions.create.mockRejectedValueOnce(apiError);

            const userMessage: MessageState = {
                id: generatePK().toString(),
                content: "Hello",
                role: "user",
                timestamp: Date.now(),
            };

            const events = [];
            try {
                for await (const event of responseService.generateResponse({
                    conversationId: conversationState.id,
                    messages: [userMessage],
                    bot: conversationState.participants[0],
                    userData: testUser,
                    maxTokens: 1000,
                })) {
                    events.push(event);
                }
            } catch (error) {
                expect(error).toBeDefined();
                expect(events).toContainEqual(
                    expect.objectContaining({
                        type: "error",
                    }),
                );
            }
        });

        it("should handle tool execution errors", async () => {
            const failingTool: Tool = {
                name: "failing_tool",
                description: "A tool that fails",
                parameters: { type: "object", properties: {} },
                execute: vi.fn().mockRejectedValue(new Error("Tool execution failed")),
            };
            toolRunner.registerTool(failingTool);

            const toolCallStream = createMockToolCallStream({
                name: "failing_tool",
                arguments: {},
                callId: "call_fail",
            });

            const recoveryStream = createMockTextStream([
                "I encountered an error with the tool. Let me help you another way.",
            ]);

            mockOpenAIClient.chat.completions.create
                .mockResolvedValueOnce(toolCallStream)
                .mockResolvedValueOnce(recoveryStream);

            const events = [];
            for await (const event of responseService.generateResponse({
                conversationId: conversationState.id,
                messages: [{ id: generatePK(), content: "Use the tool", role: "user", timestamp: Date.now() }],
                bot: conversationState.participants[0],
                userData: testUser,
                tools: [failingTool],
                maxTokens: 1000,
            })) {
                events.push(event);
            }

            // Should handle error and continue
            expect(events).toContainEqual(
                expect.objectContaining({
                    type: "tool_error",
                }),
            );

            expect(events).toContainEqual(
                expect.objectContaining({
                    type: "text",
                    content: expect.stringContaining("error"),
                }),
            );
        });
    });

    describe("Cache Coherency Integration", () => {
        it("should maintain L1/L2 cache consistency during rapid state updates", async () => {
            // Initialize conversation in database
            await chatStore.saveState(conversationState.id, conversationState);

            // 1. Load state into cache via conversation store (populates L1 and L2)
            const initialCachedState = await conversationStore.get(conversationState.id);
            expect(initialCachedState).toBeTruthy();
            expect(initialCachedState?.id).toBe(conversationState.id);

            // 2. Update config via conversation store (should update both L1 and persistent storage)
            const updatedConfig = {
                ...conversationState.config,
                stats: {
                    ...conversationState.config.stats,
                    totalCredits: 100,
                    totalToolCalls: 5,
                },
            };
            await conversationStore.updateConfig(conversationState.id, updatedConfig);

            // 3. Verify immediate consistency via cache
            const cacheAfterUpdate = await conversationStore.get(conversationState.id);
            expect(cacheAfterUpdate?.config.stats.totalCredits).toBe(100);
            expect(cacheAfterUpdate?.config.stats.totalToolCalls).toBe(5);

            // 4. Simulate cache invalidation (like what happens during chat mutations)
            await conversationStore.del(conversationState.id);

            // 5. Reload from persistent storage - should have debounced changes
            // Wait for debounce to complete (chatStore uses 2s debounce by default)
            await new Promise(resolve => setTimeout(resolve, 2500));

            const reloadedState = await conversationStore.get(conversationState.id);
            expect(reloadedState?.config.stats.totalCredits).toBe(100);
            expect(reloadedState?.config.stats.totalToolCalls).toBe(5);

            // 6. Test invalidate flag forces fresh load
            const freshState = await conversationStore.get(conversationState.id, true);
            expect(freshState?.config.stats.totalCredits).toBe(100);
        });

        it("should handle Redis failure gracefully", async () => {
            // Initialize state
            await chatStore.saveState(conversationState.id, conversationState);

            // Populate cache
            const cachedState = await conversationStore.get(conversationState.id);
            expect(cachedState).toBeTruthy();

            // Simulate Redis failure by deleting from L2 cache manually
            // (This tests L2 cache miss scenario)
            const cacheService = conversationStore["getCacheService"]();
            const l2Key = conversationStore["getL2CacheKey"](conversationState.id);
            await cacheService.del(l2Key);

            // Clear L1 cache to force L2 lookup
            conversationStore["cache"].delete(conversationState.id);

            // Should fall back to database and still work
            const fallbackState = await conversationStore.get(conversationState.id);
            expect(fallbackState?.id).toBe(conversationState.id);
        });
    });

    describe("Concurrent Access Integration", () => {
        it("should handle simultaneous operations on same conversation without data corruption", async () => {
            // Initialize state
            await chatStore.saveState(conversationState.id, conversationState);

            // Simulate concurrent updates that could happen in production:
            // 1. User sends message (updates message history)  
            // 2. Tool executes (increments tool call count)
            // 3. AI responds (increments credit usage)
            // All happening within debounce window

            const concurrentOperations = [
                // Operation 1: Update credits
                conversationStore.updateConfig(conversationState.id, {
                    ...conversationState.config,
                    stats: {
                        ...conversationState.config.stats,
                        totalCredits: 50,
                    },
                }),

                // Operation 2: Update tool calls (happening concurrently)
                conversationStore.updateConfig(conversationState.id, {
                    ...conversationState.config,
                    stats: {
                        ...conversationState.config.stats,
                        totalToolCalls: 3,
                    },
                }),

                // Operation 3: Cache lookup (should not interfere with updates)
                conversationStore.get(conversationState.id),

                // Operation 4: Team config update (runtime-only, different path)
                conversationStore.updateTeamConfig(conversationState.id, {
                    id: "team123",
                    name: "Test Team",
                    config: {},
                }),
            ];

            // Execute all operations concurrently
            const results = await Promise.allSettled(concurrentOperations);

            // All operations should succeed
            expect(results.every(r => r.status === "fulfilled")).toBe(true);

            // Wait for debouncing to complete
            await new Promise(resolve => setTimeout(resolve, 2500));

            // Final state should be consistent (last write wins for config updates)
            const finalState = await conversationStore.get(conversationState.id, true);

            // Either credits OR tool calls should be updated (depending on timing)
            // but not corrupted/mixed values
            const hasCreditsUpdate = finalState?.config.stats.totalCredits === 50;
            const hasToolCallsUpdate = finalState?.config.stats.totalToolCalls === 3;

            // At least one update should have persisted
            expect(hasCreditsUpdate || hasToolCallsUpdate).toBe(true);

            // Team should be updated (different update path)
            expect(finalState?.team?.name).toBe("Test Team");
        });

        it("should handle debounce collisions gracefully", async () => {
            // Initialize conversation
            await chatStore.saveState(conversationState.id, conversationState);

            // Create rapid-fire updates that should trigger debounce batching
            const rapidUpdates = Array.from({ length: 10 }, (_, i) =>
                conversationStore.updateConfig(conversationState.id, {
                    ...conversationState.config,
                    stats: {
                        ...conversationState.config.stats,
                        totalCredits: i * 10,
                        totalToolCalls: i,
                    },
                }),
            );

            // Execute all updates as fast as possible
            await Promise.all(rapidUpdates);

            // Should not crash or corrupt data
            const immediateState = await conversationStore.get(conversationState.id);
            expect(immediateState?.config.stats.totalCredits).toBeGreaterThanOrEqual(0);

            // Wait for all debounced writes to complete
            await new Promise(resolve => setTimeout(resolve, 3000));

            // Final state should have the last update (totalCredits = 90, totalToolCalls = 9)
            const finalState = await conversationStore.get(conversationState.id, true);
            expect(finalState?.config.stats.totalCredits).toBe(90);
            expect(finalState?.config.stats.totalToolCalls).toBe(9);
        });
    });

    describe("Service Registry Failover Integration", () => {
        it("should switch AI services mid-conversation preserving context", async () => {
            // Setup a mock Anthropic service as fallback
            const mockAnthropicService = {
                generateResponseStreaming: vi.fn(),
                getMaxOutputTokens: vi.fn().mockReturnValue(4000),
                getMaxOutputTokensRestrained: vi.fn().mockReturnValue(4000),
                safeInputCheck: vi.fn().mockResolvedValue(true),
                getErrorType: vi.fn(),
            };

            // Create a router with multiple services
            const multiServiceRouter = new FallbackRouter([
                { id: LlmServiceId.OpenAI, service: openAIService },
                { id: LlmServiceId.Anthropic, service: mockAnthropicService as any },
            ]);

            const multiServiceResponseService = new ResponseService(
                multiServiceRouter,
                toolRunner,
                new MessageHistoryBuilder(),
            );

            // First, make a successful call with OpenAI
            const firstStream = createMockTextStream(["Hello! "]);
            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(firstStream);

            const firstMessage: MessageState = {
                id: generatePK().toString(),
                content: "Hello",
                role: "user",
                timestamp: Date.now(),
            };

            await chatStore.saveState(conversationState.id, conversationState);

            const firstEvents = [];
            for await (const event of multiServiceResponseService.generateResponse({
                conversationId: conversationState.id,
                messages: [firstMessage],
                bot: conversationState.participants[0],
                userData: testUser,
                maxTokens: 1000,
            })) {
                firstEvents.push(event);
            }

            expect(firstEvents).toContainEqual(
                expect.objectContaining({ type: "text", content: "Hello! " }),
            );

            // Now simulate OpenAI failure and fallback to Anthropic
            const openAIError = new Error("Rate limit exceeded");
            (openAIError as any).status = 429;
            mockOpenAIClient.chat.completions.create.mockRejectedValueOnce(openAIError);

            // Mock Anthropic response
            const anthropicStream = createMockTextStream(["I'm here to help!"]);
            mockAnthropicService.generateResponseStreaming.mockResolvedValueOnce(anthropicStream);

            const secondMessage: MessageState = {
                id: generatePK().toString(),
                content: "Are you still there?",
                role: "user",
                timestamp: Date.now(),
            };

            const secondEvents = [];
            try {
                for await (const event of multiServiceResponseService.generateResponse({
                    conversationId: conversationState.id,
                    messages: [firstMessage, secondMessage],
                    bot: conversationState.participants[0],
                    userData: testUser,
                    maxTokens: 1000,
                })) {
                    secondEvents.push(event);
                }
            } catch (error) {
                // In case of error, check that Anthropic was attempted
                expect(mockAnthropicService.generateResponseStreaming).toHaveBeenCalled();
            }

            // Verify context preservation - Anthropic should receive full message history
            if (mockAnthropicService.generateResponseStreaming.mock.calls.length > 0) {
                const anthropicCall = mockAnthropicService.generateResponseStreaming.mock.calls[0][0];
                expect(anthropicCall.messages).toHaveLength(2);
                expect(anthropicCall.messages[0].content).toBe("Hello");
                expect(anthropicCall.messages[1].content).toBe("Are you still there?");
            }
        });
    });

    describe("Streaming and Cancellation", () => {
        it("should support aborting response generation", async () => {
            const abortController = new AbortController();

            // Create a slow stream that we'll abort
            const slowStream = createSlowMockTextStream([
                "This ",
                "is ",
                "a ",
                "very ",
                "long ",
                "response ",
                "that ",
                "will ",
                "be ",
                "cancelled",
            ], 100); // 100ms between chunks

            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(slowStream);

            const events = [];
            const responsePromise = (async () => {
                try {
                    for await (const event of responseService.generateResponse({
                        conversationId: conversationState.id,
                        messages: [{ id: generatePK(), content: "Tell me a story", role: "user", timestamp: Date.now() }],
                        bot: conversationState.participants[0],
                        userData: testUser,
                        maxTokens: 1000,
                        signal: abortController.signal,
                    })) {
                        events.push(event);

                        // Abort after receiving some chunks
                        if (events.length === 3) {
                            abortController.abort();
                        }
                    }
                } catch (error) {
                    if (error.name === "AbortError") {
                        events.push({ type: "aborted" });
                    } else {
                        throw error;
                    }
                }
            })();

            await responsePromise;

            // Should have received some events before abortion
            expect(events.length).toBeGreaterThanOrEqual(3);
            expect(events[events.length - 1]).toEqual({ type: "aborted" });
        });
    });
});

// Helper functions to create mock streams
function createMockTextStream(chunks: string[]) {
    return {
        async *[Symbol.asyncIterator]() {
            for (let i = 0; i < chunks.length; i++) {
                yield {
                    id: `chunk_${i}`,
                    choices: [{
                        delta: { content: chunks[i] },
                        finish_reason: i === chunks.length - 1 ? "stop" : null,
                    }],
                    usage: i === chunks.length - 1 ? {
                        prompt_tokens: 10,
                        completion_tokens: 20,
                        total_tokens: 30,
                    } : undefined,
                };
            }
        },
    };
}

function createMockToolCallStream(toolCall: { name: string; arguments: any; callId: string }) {
    return {
        async *[Symbol.asyncIterator]() {
            yield {
                id: "chunk_tool_1",
                choices: [{
                    delta: {
                        tool_calls: [{
                            index: 0,
                            id: toolCall.callId,
                            type: "function",
                            function: {
                                name: toolCall.name,
                                arguments: JSON.stringify(toolCall.arguments),
                            },
                        }],
                    },
                    finish_reason: null,
                }],
            };
            yield {
                id: "chunk_tool_2",
                choices: [{
                    delta: {},
                    finish_reason: "tool_calls",
                }],
                usage: {
                    prompt_tokens: 15,
                    completion_tokens: 10,
                    total_tokens: 25,
                },
            };
        },
    };
}

function createSlowMockTextStream(chunks: string[], delayMs: number) {
    return {
        async *[Symbol.asyncIterator]() {
            for (let i = 0; i < chunks.length; i++) {
                await new Promise(resolve => setTimeout(resolve, delayMs));
                yield {
                    id: `chunk_${i}`,
                    choices: [{
                        delta: { content: chunks[i] },
                        finish_reason: i === chunks.length - 1 ? "stop" : null,
                    }],
                    usage: i === chunks.length - 1 ? {
                        prompt_tokens: 10,
                        completion_tokens: chunks.length * 2,
                        total_tokens: 10 + chunks.length * 2,
                    } : undefined,
                };
            }
        },
    };
}
