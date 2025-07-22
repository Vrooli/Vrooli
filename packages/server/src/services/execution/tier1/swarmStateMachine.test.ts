import { describe, expect, test, beforeEach, afterEach, vi, type MockedFunction, type Mock } from "vitest";
import { 
    RunState, 
    EventTypes, 
    generatePK,
    toSwarmId,
    ChatConfig,
    type SessionUser, 
    type SwarmState,
    type SwarmId,
    type ServiceEvent,
    type SwarmExecutionTask,
    type ConversationContext,
    type ConversationTrigger,
    type SocketEventPayloads,
    type StateMachineState,
    type BotParticipant,
    type ChatMessage,
    type RunEventData,
} from "@vrooli/shared";
import { SwarmStateMachine } from "./swarmStateMachine.js";
import type { ISwarmContextManager } from "../shared/SwarmContextManager.js";
import type { ConversationEngine, ConversationResult } from "../../conversation/conversationEngine.js";
import type { CachedConversationStateStore } from "../../response/chatStore.js";
import { EventInterceptor } from "../../events/EventInterceptor.js";
import { EventPublisher } from "../../events/publisher.js";

// Mock dependencies
vi.mock("../../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        debug: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
    },
}));

vi.mock("../../events/publisher.js", () => ({
    EventPublisher: {
        emit: vi.fn().mockResolvedValue({ proceed: true, reason: null }),
    },
}));

vi.mock("../../events/EventInterceptor.js", () => ({
    EventInterceptor: vi.fn().mockImplementation(() => ({
        checkInterception: vi.fn().mockResolvedValue({ 
            intercepted: false, 
            progression: "continue",
            responses: [],
        }),
    })),
}));

vi.mock("../../events/LockService.js", () => ({
    InMemoryLockService: vi.fn().mockImplementation(() => ({
        acquireLock: vi.fn().mockResolvedValue(true),
        releaseLock: vi.fn().mockResolvedValue(true),
    })),
}));

vi.mock("@vrooli/shared", async () => {
    const actual = await vi.importActual("@vrooli/shared");
    return {
        ...actual,
        generatePK: vi.fn(() => "test-pk-123"),
        toSwarmId: vi.fn((id: string) => id as SwarmId),
    };
});

// Test data factories
function createMockUser(partial: Partial<SessionUser> = {}): SessionUser {
    return {
        id: "test-user-123",
        name: "Test User",
        languages: ["en"],
        ...partial,
    } as SessionUser;
}

function createMockSwarmExecutionTask(partial: Partial<SwarmExecutionTask> = {}): SwarmExecutionTask {
    const defaultTask: SwarmExecutionTask = {
        taskId: "test-task-123",
        taskType: "Swarm",
        input: {
            goal: "Test goal",
            userData: createMockUser(),
            swarmId: "test-swarm-123",
        },
        ...partial,
    };
    return defaultTask;
}

function createMockSwarmState(swarmId: SwarmId, partial: Partial<SwarmState> = {}): SwarmState {
    const now = new Date();
    return {
        swarmId,
        version: 1,
        chatConfig: {
            __version: "1.0.0",
            goal: "Test goal",
            subtasks: [],
            blackboard: [],
            resources: [],
            records: [],
            stats: {
                totalToolCalls: 0,
                totalCredits: "0",
                startedAt: Date.now(),
                lastProcessingCycleEndedAt: null,
            },
            limits: {
                maxCredits: "10000",
                maxDurationMs: 3600000,
            },
            scheduling: {
                defaultDelayMs: 0,
                requiresApprovalTools: "none",
                approvalTimeoutMs: 300000,
                autoRejectOnTimeout: true,
            },
            pendingToolCalls: [],
            ...partial.chatConfig,
        },
        execution: {
            status: RunState.UNINITIALIZED,
            agents: [],
            activeRuns: [],
            startedAt: now,
            lastActivityAt: now,
            ...partial.execution,
        },
        resources: {
            allocated: [],
            consumed: { credits: 0, tokens: 0, time: 0 },
            remaining: { credits: 10000, tokens: 10000, time: 3600 },
            ...partial.resources,
        },
        metadata: {
            createdAt: now,
            lastUpdated: now,
            updatedBy: "system",
            subscribers: new Set<string>(),
            ...partial.metadata,
        },
    };
}

function createMockConversationResult(partial: Partial<ConversationResult> = {}): ConversationResult {
    return {
        success: true,
        messages: [],
        sharedState: {},
        error: null,
        ...partial,
    };
}

function createServiceEvent(type: string, data: any, partial: Partial<ServiceEvent> = {}): ServiceEvent {
    return {
        id: generatePK(),
        type,
        timestamp: new Date(),
        tier: "tier1",
        source: "test",
        chatId: "test-swarm-123",
        data,
        ...partial,
    };
}

describe("SwarmStateMachine", () => {
    let stateMachine: SwarmStateMachine;
    let mockContextManager: ISwarmContextManager;
    let mockConversationEngine: ConversationEngine;
    let mockChatStore: CachedConversationStateStore;
    let mockEventInterceptor: EventInterceptor;

    beforeEach(() => {
        vi.clearAllMocks();

        // Setup mock context manager
        mockContextManager = {
            createContext: vi.fn().mockResolvedValue(undefined),
            getContext: vi.fn().mockResolvedValue(createMockSwarmState("test-swarm-123" as SwarmId)),
            updateContext: vi.fn().mockResolvedValue(createMockSwarmState("test-swarm-123" as SwarmId)),
            deleteContext: vi.fn().mockResolvedValue(undefined),
            allocateResources: vi.fn(),
            releaseResources: vi.fn(),
            getResourceStatus: vi.fn(),
            validate: vi.fn().mockResolvedValue({ valid: true, errors: [] }),
            query: vi.fn(),
            healthCheck: vi.fn().mockResolvedValue({ healthy: true }),
            getMetrics: vi.fn(),
            start: vi.fn().mockResolvedValue(undefined),
            stop: vi.fn().mockResolvedValue(undefined),
        } as unknown as ISwarmContextManager;

        // Setup mock conversation engine
        mockConversationEngine = {
            orchestrateConversation: vi.fn().mockResolvedValue(createMockConversationResult()),
        } as unknown as ConversationEngine;

        // Setup mock chat store
        mockChatStore = {
            get: vi.fn().mockResolvedValue({
                config: ChatConfig.default().export(),
            }),
        } as unknown as CachedConversationStateStore;

        stateMachine = new SwarmStateMachine(
            mockContextManager,
            mockConversationEngine,
            mockChatStore,
        );

        // Get the mocked event interceptor instance
        mockEventInterceptor = (stateMachine as any).eventInterceptor;
    });

    afterEach(async () => {
        // Clean up state machine
        if (stateMachine) {
            try {
                await stateMachine.stop({ mode: "force", reason: "test cleanup" });
            } catch {
                // Ignore cleanup errors
            }
        }
    });

    describe("Constructor and Initialization", () => {
        test("should initialize with correct dependencies", () => {
            expect(stateMachine).toBeInstanceOf(SwarmStateMachine);
            expect(stateMachine.getState()).toBe(RunState.UNINITIALIZED);
            expect(stateMachine.getTaskId()).toBe("undefined_swarm_task_id"); // Before start
            expect(stateMachine.getAssociatedUserId()).toBeUndefined();
        });

        test("should create EventInterceptor with correct dependencies", () => {
            expect(EventInterceptor).toHaveBeenCalledWith(
                expect.any(Object), // InMemoryLockService
                mockContextManager,
            );
        });
    });

    describe("start() method", () => {
        test("should start successfully with valid request", async () => {
            const request = createMockSwarmExecutionTask();
            
            const result = await stateMachine.start(request);
            
            expect(result.success).toBe(true);
            expect(result.error).toBeUndefined();
            expect(stateMachine.getState()).toBe(RunState.READY);
            expect(stateMachine.getTaskId()).toBe("test-swarm-123");
            expect(stateMachine.getAssociatedUserId()).toBe("test-user-123");
            
            // Verify context creation
            expect(mockContextManager.createContext).toHaveBeenCalledWith(
                "test-swarm-123",
                expect.objectContaining({
                    swarmId: "test-swarm-123",
                    version: 1,
                    chatConfig: expect.objectContaining({
                        goal: "Test goal",
                        swarmLeader: "test-user-123",
                    }),
                }),
            );
            
            // Verify conversation engine orchestration
            expect(mockConversationEngine.orchestrateConversation).toHaveBeenCalledWith({
                context: expect.any(Object),
                trigger: { type: "start" },
                strategy: "conversational",
            });
            
            // Verify state change event
            expect(EventPublisher.emit).toHaveBeenCalledWith(
                EventTypes.SWARM.STATE_CHANGED,
                expect.objectContaining({
                    chatId: "test-swarm-123",
                    oldState: "UNINITIALIZIALIZED",
                    newState: RunState.READY,
                }),
            );
        });

        test("should generate swarmId if not provided", async () => {
            const request = createMockSwarmExecutionTask({
                input: {
                    goal: "Test goal",
                    userData: createMockUser(),
                    // No swarmId provided
                },
            });
            
            const result = await stateMachine.start(request);
            
            expect(result.success).toBe(true);
            expect(generatePK).toHaveBeenCalled();
            expect(stateMachine.getTaskId()).toBe("test-pk-123");
        });

        test("should load chat config from chatStore if chatId provided", async () => {
            const chatConfig = {
                ...ChatConfig.default().export(),
                goal: "Chat-specific goal",
                activeBotId: "bot-123",
            };
            vi.mocked(mockChatStore.get).mockResolvedValueOnce({
                config: chatConfig,
            });
            
            const request = createMockSwarmExecutionTask({
                input: {
                    goal: "Test goal",
                    userData: createMockUser(),
                    chatId: "existing-chat-123",
                },
            });
            
            await stateMachine.start(request);
            
            expect(mockChatStore.get).toHaveBeenCalledWith("existing-chat-123");
            expect(mockContextManager.createContext).toHaveBeenCalledWith(
                expect.any(String),
                expect.objectContaining({
                    chatConfig: expect.objectContaining({
                        goal: "Test goal", // Request goal overrides chat config goal
                        activeBotId: "bot-123",
                    }),
                }),
            );
        });

        test("should fail if already started", async () => {
            const request = createMockSwarmExecutionTask();
            
            // First start should succeed
            await stateMachine.start(request);
            
            // Second start should fail
            const result = await stateMachine.start(request);
            
            expect(result.success).toBe(false);
            expect(result.error).toContain("Already started");
        });

        test("should handle context creation failure", async () => {
            const error = new Error("Context creation failed");
            vi.mocked(mockContextManager.createContext).mockRejectedValueOnce(error);
            
            const request = createMockSwarmExecutionTask();
            const result = await stateMachine.start(request);
            
            expect(result.success).toBe(false);
            expect(result.error).toMatchObject({
                message: "Context creation failed",
                name: "Error",
            });
            expect(stateMachine.getState()).toBe(RunState.FAILED);
            
            // Verify failure event
            expect(EventPublisher.emit).toHaveBeenCalledWith(
                EventTypes.SWARM.STATE_CHANGED,
                expect.objectContaining({
                    newState: RunState.FAILED,
                    message: expect.stringContaining("Context creation failed"),
                }),
            );
        });

        test("should handle conversation engine failure", async () => {
            vi.mocked(mockConversationEngine.orchestrateConversation).mockResolvedValueOnce(
                createMockConversationResult({ success: false, error: "Orchestration failed" }),
            );
            
            const request = createMockSwarmExecutionTask();
            const result = await stateMachine.start(request);
            
            expect(result.success).toBe(false); // Start fails when conversation engine fails
            expect(stateMachine.getState()).toBe(RunState.FAILED);
        });

        test("should handle missing context after creation", async () => {
            vi.mocked(mockContextManager.getContext).mockResolvedValueOnce(null);
            
            const request = createMockSwarmExecutionTask();
            const result = await stateMachine.start(request);
            
            expect(result.success).toBe(false);
            expect(result.error?.message).toBe("Failed to get swarm context");
        });

        test("should handle event publisher blocking", async () => {
            vi.mocked(EventPublisher.emit).mockResolvedValueOnce({ 
                proceed: false, 
                reason: "Rate limited", 
            });
            
            const request = createMockSwarmExecutionTask();
            const result = await stateMachine.start(request);
            
            // Should continue despite blocking
            expect(result.success).toBe(true);
            expect(stateMachine.getState()).toBe(RunState.READY);
        });
    });

    describe("Event Handling", () => {
        beforeEach(async () => {
            // Start the state machine for event handling tests
            await stateMachine.start(createMockSwarmExecutionTask());
        });

        describe("handleEvent validation", () => {
            test("should validate SWARM.STARTED events", async () => {
                const validEvent = createServiceEvent(EventTypes.SWARM.STARTED, {
                    chatId: "test-swarm-123",
                });
                
                await stateMachine.handleEvent(validEvent);
                // Should be queued (no error)
                expect((stateMachine as any).eventQueue.length).toBe(1);
                
                // Invalid event (missing chatId)
                const invalidEvent = createServiceEvent(EventTypes.SWARM.STARTED, {});
                await stateMachine.handleEvent(invalidEvent);
                // Should not be queued
                expect((stateMachine as any).eventQueue.length).toBe(1);
            });
        });

        describe("processEvent routing", () => {
            test("should route CHAT.MESSAGE_ADDED to handleExternalMessage", async () => {
                const message: ChatMessage = {
                    id: "msg-123",
                    text: "Test message",
                    userId: "user-123",
                    chatId: "test-swarm-123",
                    role: "user",
                    timestamp: new Date().toISOString(),
                };
                
                const event = createServiceEvent(EventTypes.CHAT.MESSAGE_ADDED, {
                    chatId: "test-swarm-123",
                    messages: [message],
                });
                
                await (stateMachine as any).processEvent(event);
                
                expect(mockConversationEngine.orchestrateConversation).toHaveBeenCalledWith({
                    context: expect.any(Object),
                    trigger: {
                        type: "user_message",
                        message,
                    },
                    strategy: "conversational",
                });
            });

            test("should route TOOL.APPROVAL_GRANTED to handleApprovedTool", async () => {
                const event = createServiceEvent(EventTypes.TOOL.APPROVAL_GRANTED, {
                    toolName: "test_tool",
                    toolCallId: "call-123",
                    pendingId: "pending-123",
                    callerBotId: "bot-123",
                    chatId: "test-swarm-123",
                });
                
                // Add original tool call context
                event.execution = {
                    originalToolCall: {
                        name: "test_tool",
                        arguments: { param: "value" },
                    },
                };
                
                await (stateMachine as any).processEvent(event);
                
                expect(mockEventInterceptor.checkInterception).toHaveBeenCalledWith(
                    event,
                    expect.any(Object),
                );
                expect(mockConversationEngine.orchestrateConversation).toHaveBeenCalled();
            });

            test("should route TOOL.APPROVAL_REJECTED to handleRejectedTool", async () => {
                const event = createServiceEvent(EventTypes.TOOL.APPROVAL_REJECTED, {
                    toolName: "test_tool",
                    toolCallId: "call-123",
                    pendingId: "pending-123",
                    reason: "Not allowed",
                    chatId: "test-swarm-123",
                });
                
                await (stateMachine as any).processEvent(event);
                
                expect(mockConversationEngine.orchestrateConversation).toHaveBeenCalledWith({
                    context: expect.any(Object),
                    trigger: {
                        type: "continue",
                        lastEvent: event,
                    },
                    strategy: "reasoning", // Uses reasoning for fallback
                });
            });

            test("should route CHAT.CANCELLATION_REQUESTED to handleCancellationRequest", async () => {
                const stopSpy = vi.spyOn(stateMachine, "stop");
                
                const event = createServiceEvent(EventTypes.CHAT.CANCELLATION_REQUESTED, {
                    chatId: "test-swarm-123",
                });
                
                await (stateMachine as any).processEvent(event);
                
                expect(stopSpy).toHaveBeenCalledWith({
                    mode: "graceful",
                    reason: "User requested cancellation",
                });
            });

            test("should route RUN.COMPLETED to handleInternalStatusUpdate", async () => {
                const runData: RunEventData = {
                    runId: "run-123",
                    status: RunState.COMPLETED,
                    executionTime: 1000,
                };
                
                const event = createServiceEvent(EventTypes.RUN.COMPLETED, runData);
                
                await (stateMachine as any).processEvent(event);
                
                expect(mockConversationEngine.orchestrateConversation).toHaveBeenCalledWith({
                    context: expect.any(Object),
                    trigger: {
                        type: "continue",
                        lastEvent: event,
                    },
                    strategy: "conversational",
                });
            });

            test("should route RUN.FAILED to handleInternalStatusUpdate with reasoning", async () => {
                const runData: RunEventData = {
                    runId: "run-123",
                    status: RunState.FAILED,
                    error: "Execution failed",
                };
                
                const event = createServiceEvent(EventTypes.RUN.FAILED, runData);
                
                // Set state to RUNNING to test state transition
                (stateMachine as any).state = RunState.RUNNING;
                
                await (stateMachine as any).processEvent(event);
                
                expect(mockConversationEngine.orchestrateConversation).toHaveBeenCalledWith({
                    context: expect.any(Object),
                    trigger: {
                        type: "continue",
                        lastEvent: event,
                    },
                    strategy: "reasoning", // Uses reasoning for failures
                });
                
                expect(stateMachine.getState()).toBe(RunState.READY);
            });

            test("should route swarm/* events to handleSwarmEvent", async () => {
                const event = createServiceEvent(EventTypes.SWARM.STATE_CHANGED, {
                    chatId: "child-swarm-123",
                    oldState: "RUNNING" as StateMachineState,
                    newState: "COMPLETED" as StateMachineState,
                    message: "Child swarm completed",
                });
                
                await (stateMachine as any).processEvent(event);
                // Should log but not error
            });

            test("should route safety/security events to handleSafetyEvent", async () => {
                const stopSpy = vi.spyOn(stateMachine, "stop");
                
                const event = createServiceEvent(EventTypes.SECURITY.EMERGENCY_STOP, {
                    reason: "Security threat detected",
                });
                
                await (stateMachine as any).processEvent(event);
                
                expect(stopSpy).toHaveBeenCalledWith({
                    mode: "force",
                    reason: "Emergency stop requested",
                });
            });
        });

        describe("shouldHandleEvent filtering", () => {
            test("should filter events by swarmId", () => {
                const ourEvent = createServiceEvent("test/event", { swarmId: "test-swarm-123" });
                const otherEvent = createServiceEvent("test/event", { swarmId: "other-swarm-456" });
                
                expect((stateMachine as any).shouldHandleEvent(ourEvent)).toBe(true);
                expect((stateMachine as any).shouldHandleEvent(otherEvent)).toBe(false);
            });

            test("should filter user events by userId", () => {
                const ourUserEvent = createServiceEvent("user/test-user-123/credits", {});
                const otherUserEvent = createServiceEvent("user/other-user-456/credits", {});
                
                expect((stateMachine as any).shouldHandleEvent(ourUserEvent)).toBe(true);
                expect((stateMachine as any).shouldHandleEvent(otherUserEvent)).toBe(false);
            });

            test("should handle events without filtering data", () => {
                const genericEvent = createServiceEvent("system/heartbeat", {});
                
                expect((stateMachine as any).shouldHandleEvent(genericEvent)).toBe(true);
            });
        });

        describe("getEventPatterns", () => {
            test("should return comprehensive event patterns", () => {
                const patterns = (stateMachine as any).getEventPatterns();
                
                expect(patterns).toContainEqual({ pattern: "chat/*" });
                expect(patterns).toContainEqual({ pattern: "tool/*" });
                expect(patterns).toContainEqual({ pattern: "swarm/*" });
                expect(patterns).toContainEqual({ pattern: "run/*" });
                expect(patterns).toContainEqual({ pattern: "user/*" });
                expect(patterns).toContainEqual({ pattern: "safety/*" });
                expect(patterns).toContainEqual({ pattern: "security/*" });
            });
        });
    });

    describe("Tool Approval Flow", () => {
        beforeEach(async () => {
            await stateMachine.start(createMockSwarmExecutionTask());
            // Clear mock calls from start method
            vi.clearAllMocks();
        });

        test("should handle approved tool with successful execution", async () => {
            const event = createServiceEvent(EventTypes.TOOL.APPROVAL_GRANTED, {
                toolName: "resource_manage",
                toolCallId: "call-123",
                pendingId: "pending-123",
                callerBotId: "bot-123",
                chatId: "test-swarm-123",
            });
            
            event.execution = {
                originalToolCall: {
                    name: "resource_manage",
                    arguments: { action: "find", type: "routine" },
                },
            };
            
            await (stateMachine as any).handleApprovedTool(event);
            
            // Verify interception check
            expect(mockEventInterceptor.checkInterception).toHaveBeenCalledWith(
                event,
                expect.any(Object),
            );
            
            // Verify conversation engine routing
            expect(mockConversationEngine.orchestrateConversation).toHaveBeenCalledWith({
                context: expect.any(Object),
                trigger: {
                    type: "continue",
                    lastEvent: event,
                },
                strategy: "conversational",
            });
        });

        test("should handle approved tool blocked by interceptor", async () => {
            vi.mocked(mockEventInterceptor.checkInterception).mockResolvedValueOnce({
                intercepted: true,
                progression: "blocked",
                responses: ["Security violation detected"],
            });
            
            const event = createServiceEvent(EventTypes.TOOL.APPROVAL_GRANTED, {
                toolName: "dangerous_tool",
                toolCallId: "call-123",
                pendingId: "pending-123",
                callerBotId: "bot-123",
                chatId: "test-swarm-123",
            });
            
            event.execution = {
                originalToolCall: {
                    name: "dangerous_tool",
                    arguments: { action: "harm" },
                },
            };
            
            await (stateMachine as any).handleApprovedTool(event);
            
            // Should emit TOOL.FAILED event
            expect(EventPublisher.emit).toHaveBeenCalledWith(
                EventTypes.TOOL.FAILED,
                expect.objectContaining({
                    toolCallId: "call-123",
                    toolName: "dangerous_tool",
                    error: expect.stringContaining("blocked by security system"),
                }),
            );
            
            // Should not route to conversation engine
            expect(mockConversationEngine.orchestrateConversation).not.toHaveBeenCalled();
        });

        test("should handle approved tool missing original call context", async () => {
            const event = createServiceEvent(EventTypes.TOOL.APPROVAL_GRANTED, {
                toolName: "test_tool",
                toolCallId: "call-123",
                pendingId: "pending-123",
                callerBotId: "bot-123",
                chatId: "test-swarm-123",
            });
            
            // No execution.originalToolCall
            
            await (stateMachine as any).handleApprovedTool(event);
            
            // Should log error but not throw
            expect(mockConversationEngine.orchestrateConversation).not.toHaveBeenCalled();
        });

        test("should handle approved tool with missing required data", async () => {
            const event = createServiceEvent(EventTypes.TOOL.APPROVAL_GRANTED, {
                // Missing toolName
                toolCallId: "call-123",
                pendingId: "pending-123",
                chatId: "test-swarm-123",
            });
            
            await (stateMachine as any).handleApprovedTool(event);
            
            // Should log error but not throw
            expect(mockEventInterceptor.checkInterception).not.toHaveBeenCalled();
        });

        test("should handle rejected tool with fallback strategy", async () => {
            const event = createServiceEvent(EventTypes.TOOL.APPROVAL_REJECTED, {
                toolName: "expensive_tool",
                toolCallId: "call-123",
                pendingId: "pending-123",
                reason: "Insufficient credits",
                chatId: "test-swarm-123",
            });
            
            await (stateMachine as any).handleRejectedTool(event);
            
            expect(mockConversationEngine.orchestrateConversation).toHaveBeenCalledWith({
                context: expect.any(Object),
                trigger: {
                    type: "continue",
                    lastEvent: event,
                },
                strategy: "reasoning", // Uses reasoning for alternatives
            });
        });

        test("should handle rejected tool with missing data", async () => {
            const event = createServiceEvent(EventTypes.TOOL.APPROVAL_REJECTED, {
                // Missing toolName
                toolCallId: "call-123",
                reason: "Invalid tool",
                chatId: "test-swarm-123",
            });
            
            await (stateMachine as any).handleRejectedTool(event);
            
            // Should log error but not throw
            expect(mockConversationEngine.orchestrateConversation).not.toHaveBeenCalled();
        });
    });

    describe("State Management", () => {
        test("should support pause and resume", async () => {
            await stateMachine.start(createMockSwarmExecutionTask());
            
            // Transition to RUNNING
            (stateMachine as any).state = RunState.RUNNING;
            
            // Pause
            const pauseResult = await stateMachine.pause();
            expect(pauseResult).toBe(true);
            expect(stateMachine.getState()).toBe(RunState.PAUSED);
            
            // Resume
            const resumeResult = await stateMachine.resume();
            expect(resumeResult).toBe(true);
            expect(stateMachine.getState()).toBe(RunState.READY);
        });

        test("should stop gracefully with statistics", async () => {
            await stateMachine.start(createMockSwarmExecutionTask());
            
            // Update context with some statistics
            const mockContext = createMockSwarmState("test-swarm-123" as SwarmId, {
                chatConfig: {
                    subtasks: [
                        { id: "task-1", description: "Task 1", status: "done" },
                        { id: "task-2", description: "Task 2", status: "pending" },
                    ],
                    stats: {
                        totalToolCalls: 5,
                        totalCredits: "100",
                        startedAt: Date.now(),
                        lastProcessingCycleEndedAt: Date.now(),
                    },
                },
            });
            vi.mocked(mockContextManager.getContext).mockResolvedValueOnce(mockContext);
            
            const result = await stateMachine.stop({ mode: "graceful", reason: "Test complete" });
            
            expect(result.success).toBe(true);
            expect(result.finalState).toMatchObject({
                mode: "graceful",
                reason: "Test complete",
                totalSubTasks: 2,
                completedSubTasks: 1,
                totalCreditsUsed: "100",
                totalToolCalls: 5,
            });
            
            expect(EventPublisher.emit).toHaveBeenCalledWith(
                EventTypes.SWARM.STATE_CHANGED,
                expect.objectContaining({
                    oldState: expect.any(String),
                    newState: "COMPLETED",
                    message: "Test complete",
                }),
            );
        });

        test("should stop forcefully", async () => {
            await stateMachine.start(createMockSwarmExecutionTask());
            
            const result = await stateMachine.stop({ mode: "force", reason: "Emergency" });
            
            expect(result.success).toBe(true);
            expect(stateMachine.getState()).toBe(RunState.CANCELLED);
            
            expect(EventPublisher.emit).toHaveBeenCalledWith(
                EventTypes.SWARM.STATE_CHANGED,
                expect.objectContaining({
                    newState: "TERMINATED",
                }),
            );
        });

        test("should handle stop with requesting user", async () => {
            await stateMachine.start(createMockSwarmExecutionTask());
            
            const requestingUser = createMockUser({ id: "admin-user" });
            const result = await stateMachine.stop({ mode: "graceful", reason: "Admin stopped", requestingUser });
            
            expect(result.success).toBe(true);
        });

        test("should handle stop with options object", async () => {
            await stateMachine.start(createMockSwarmExecutionTask());
            
            const result = await stateMachine.stop({
                mode: "force",
                reason: "System shutdown",
                requestingUser: createMockUser({ id: "system" }),
            });
            
            expect(result.success).toBe(true);
            expect(result.finalState?.mode).toBe("force");
        });

        test("should handle stop when context not found", async () => {
            await stateMachine.start(createMockSwarmExecutionTask());
            
            vi.mocked(mockContextManager.getContext).mockResolvedValueOnce(null);
            
            const result = await stateMachine.stop();
            
            expect(result.success).toBe(true);
            expect(result.finalState).toMatchObject({
                totalSubTasks: 0,
                completedSubTasks: 0,
                totalCreditsUsed: "0",
                totalToolCalls: 0,
            });
        });
    });

    describe("Error Handling", () => {
        test("should determine fatal errors correctly", async () => {
            await stateMachine.start(createMockSwarmExecutionTask());
            
            const event = createServiceEvent("test/event", {});
            
            // Network errors are non-fatal
            const networkError = new Error("ECONNREFUSED: Connection refused");
            expect(await (stateMachine as any).isErrorFatal(networkError, event)).toBe(false);
            
            // Configuration errors are fatal
            const configError = new Error("No leader bot configured");
            expect(await (stateMachine as any).isErrorFatal(configError, event)).toBe(true);
            
            // Unknown errors default to non-fatal
            const unknownError = new Error("Something went wrong");
            expect(await (stateMachine as any).isErrorFatal(unknownError, event)).toBe(false);
        });

        test("should handle errors in handleExternalMessage", async () => {
            await stateMachine.start(createMockSwarmExecutionTask());
            
            // Make conversation engine return failure (not throw)
            vi.mocked(mockConversationEngine.orchestrateConversation).mockResolvedValueOnce(
                createMockConversationResult({ success: false, error: "Orchestration failed" }),
            );
            
            const event = createServiceEvent(EventTypes.CHAT.MESSAGE_ADDED, {
                chatId: "test-swarm-123",
                messages: [{
                    id: "msg-123",
                    text: "Test",
                    userId: "user-123",
                    chatId: "test-swarm-123",
                    role: "user",
                    timestamp: new Date().toISOString(),
                }],
            });
            
            // Set to RUNNING to test state transition
            (stateMachine as any).state = RunState.RUNNING;
            
            await (stateMachine as any).handleExternalMessage(event);
            
            // Should transition to READY for non-fatal error
            expect(stateMachine.getState()).toBe(RunState.READY);
        });

        test("should handle fatal errors in handleExternalMessage", async () => {
            await stateMachine.start(createMockSwarmExecutionTask());
            
            // Make conversation engine throw fatal error
            vi.mocked(mockConversationEngine.orchestrateConversation).mockRejectedValueOnce(
                new Error("Invalid configuration"),
            );
            
            const event = createServiceEvent(EventTypes.CHAT.MESSAGE_ADDED, {
                chatId: "test-swarm-123",
                messages: [{
                    id: "msg-123",
                    text: "Test",
                    userId: "user-123",
                    chatId: "test-swarm-123",
                    role: "user",
                    timestamp: new Date().toISOString(),
                }],
            });
            
            await (stateMachine as any).handleExternalMessage(event);
            
            // Should transition to FAILED for fatal error
            expect(stateMachine.getState()).toBe(RunState.FAILED);
        });

        test("should handle missing swarmId in handlers", async () => {
            // Don't start the state machine, so swarmId is null
            const newStateMachine = new SwarmStateMachine(
                mockContextManager,
                mockConversationEngine,
                mockChatStore,
            );
            
            const event = createServiceEvent(EventTypes.CHAT.MESSAGE_ADDED, {
                chatId: "test-swarm-123",
                messages: [{ id: "msg-123" }],
            });
            
            // Should log error but not throw
            await (newStateMachine as any).handleExternalMessage(event);
            expect(mockConversationEngine.orchestrateConversation).not.toHaveBeenCalled();
        });
    });

    describe("ConversationContext Transformation", () => {
        test("should transform SwarmState to ConversationContext correctly", async () => {
            await stateMachine.start(createMockSwarmExecutionTask());
            
            const mockAgents: BotParticipant[] = [
                { id: "bot-1", name: "Bot 1", roles: ["assistant"] } as BotParticipant,
            ];
            
            const swarmState = createMockSwarmState("test-swarm-123" as SwarmId, {
                execution: {
                    agents: mockAgents,
                },
                chatConfig: {
                    blackboard: [
                        { id: "key1", value: "value1" },
                        { id: "key2", value: 42 },
                    ],
                },
            });
            
            const context = await (stateMachine as any).transformToConversationContext(swarmState);
            
            expect(context).toMatchObject({
                swarmId: "test-swarm-123",
                userData: expect.objectContaining({ id: "test-user-123" }),
                participants: mockAgents,
                sharedState: {
                    key1: "value1",
                    key2: 42,
                },
            });
        });

        test("should handle empty blackboard in transformation", async () => {
            await stateMachine.start(createMockSwarmExecutionTask());
            
            const swarmState = createMockSwarmState("test-swarm-123" as SwarmId);
            const context = await (stateMachine as any).transformToConversationContext(swarmState);
            
            expect(context.sharedState).toEqual({});
        });
    });

    describe("Event Pattern Subscriptions", () => {
        test("should subscribe to all required event patterns", () => {
            const patterns = (stateMachine as any).getEventPatterns();
            
            // Verify comprehensive coverage
            const patternStrings = patterns.map(p => p.pattern);
            
            // Chat events
            expect(patternStrings).toContain("chat/*");
            expect(patternStrings).toContain("chat/message/*");
            expect(patternStrings).toContain("chat/cancellation/*");
            
            // Tool events
            expect(patternStrings).toContain("tool/*");
            expect(patternStrings).toContain("tool/approval/*");
            
            // Swarm events
            expect(patternStrings).toContain("swarm/*");
            expect(patternStrings).toContain("swarm/state/*");
            expect(patternStrings).toContain("swarm/goal/*");
            
            // Run events
            expect(patternStrings).toContain("run/*");
            
            // User events
            expect(patternStrings).toContain("user/*");
            expect(patternStrings).toContain("user/credits/*");
            expect(patternStrings).toContain("user/permissions/*");
            
            // Safety/security events
            expect(patternStrings).toContain("safety/*");
            expect(patternStrings).toContain("security/*");
        });
    });

    describe("Integration Scenarios", () => {
        test("should handle complete tool approval and execution flow", async () => {
            await stateMachine.start(createMockSwarmExecutionTask());
            
            // Step 1: Tool approval granted
            const approvalEvent = createServiceEvent(EventTypes.TOOL.APPROVAL_GRANTED, {
                toolName: "spawn_swarm",
                toolCallId: "call-123",
                pendingId: "pending-123",
                callerBotId: "bot-123",
                chatId: "test-swarm-123",
            });
            
            approvalEvent.execution = {
                originalToolCall: {
                    name: "spawn_swarm",
                    arguments: { goal: "Sub-task goal" },
                },
            };
            
            // Mock successful tool execution
            vi.mocked(mockConversationEngine.orchestrateConversation).mockResolvedValueOnce(
                createMockConversationResult({
                    success: true,
                    messages: [{
                        role: "assistant",
                        content: "Created child swarm",
                    }],
                }),
            );
            
            await (stateMachine as any).processEvent(approvalEvent);
            
            // Verify complete flow
            expect(mockEventInterceptor.checkInterception).toHaveBeenCalled();
            expect(mockConversationEngine.orchestrateConversation).toHaveBeenCalled();
        });

        test("should handle security event leading to emergency stop", async () => {
            await stateMachine.start(createMockSwarmExecutionTask());
            
            const securityEvent = createServiceEvent(EventTypes.SECURITY.EMERGENCY_STOP, {
                reason: "Critical security threat",
                severity: "critical",
            });
            
            await (stateMachine as any).processEvent(securityEvent);
            
            expect(stateMachine.getState()).toBe(RunState.CANCELLED);
        });
    });
});
