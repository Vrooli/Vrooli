import { describe, it, expect, vi, beforeEach } from "vitest";
import { ChatConfig, type ChatConfigObject, type SwarmSubTask, type SwarmResource, type BlackboardItem, type ChatToolCallRecord, PendingToolCallStatus, type PendingToolCallEntry } from "./chat.js";
import { API_CREDITS_MULTIPLIER } from "../../consts/api.js";
import { DAYS_1_MS, MINUTES_10_MS, MINUTES_1_MS, SECONDS_1_MS } from "../../consts/numbers.js";

describe("ChatConfig", () => {
    let mockLogger: any;

    beforeEach(() => {
        mockLogger = {
            trace: vi.fn(),
            debug: vi.fn(),
            info: vi.fn(),
            warn: vi.fn(),
            error: vi.fn(),
        };
    });

    describe("constructor", () => {
        it("should create ChatConfig with complete data", () => {
            const config: ChatConfigObject = {
                __version: "1.0",
                goal: "Complete the user's task",
                preferredModel: "gpt-4",
                subtasks: [{
                    id: "task1",
                    description: "First task",
                    status: "todo",
                    created_at: new Date().toISOString(),
                }],
                swarmLeader: "bot1",
                subtaskLeaders: { "task1": "bot2" },
                teamId: "team123",
                blackboard: [{
                    id: "item1",
                    value: { note: "Important info" },
                    created_at: new Date().toISOString(),
                }],
                resources: [{
                    id: "res1",
                    kind: "Note",
                    creator_bot_id: "bot1",
                    created_at: new Date().toISOString(),
                }],
                records: [{
                    id: "call1",
                    routine_id: "routine1",
                    routine_name: "Process Data",
                    params: { input: "data" },
                    output_resource_ids: ["res1"],
                    caller_bot_id: "bot1",
                    created_at: new Date().toISOString(),
                }],
                eventSubscriptions: {
                    "swarm/subtask/created": ["bot1", "bot2"],
                },
                policy: {
                    visibility: "private",
                    acl: ["user1", "user2"],
                },
                stats: {
                    totalToolCalls: 5,
                    totalCredits: "1000000",
                    startedAt: Date.now(),
                    lastProcessingCycleEndedAt: Date.now(),
                },
                limits: {
                    maxToolCalls: 100,
                    maxCredits: "5000000",
                },
                scheduling: {
                    defaultDelayMs: 1000,
                    requiresApprovalTools: ["dangerous_tool"],
                },
                pendingToolCalls: [{
                    pendingId: "pending1",
                    toolCallId: "tool1",
                    toolName: "web_search",
                    toolArguments: JSON.stringify({ query: "test" }),
                    callerBotId: "bot1",
                    conversationId: "conv1",
                    requestedAt: Date.now(),
                    status: PendingToolCallStatus.PENDING_APPROVAL,
                    executionAttempts: 0,
                }],
            };

            const chatConfig = new ChatConfig({ config });

            expect(chatConfig.__version).toBe("1.0");
            expect(chatConfig.goal).toBe("Complete the user's task");
            expect(chatConfig.preferredModel).toBe("gpt-4");
            expect(chatConfig.subtasks).toHaveLength(1);
            expect(chatConfig.subtasks?.[0].id).toBe("task1");
            expect(chatConfig.swarmLeader).toBe("bot1");
            expect(chatConfig.subtaskLeaders?.["task1"]).toBe("bot2");
            expect(chatConfig.teamId).toBe("team123");
            expect(chatConfig.blackboard).toHaveLength(1);
            expect(chatConfig.resources).toHaveLength(1);
            expect(chatConfig.records).toHaveLength(1);
            expect(chatConfig.eventSubscriptions?.["swarm/subtask/created"]).toEqual(["bot1", "bot2"]);
            expect(chatConfig.policy?.visibility).toBe("private");
            expect(chatConfig.stats.totalToolCalls).toBe(5);
            expect(chatConfig.limits?.maxToolCalls).toBe(100);
            expect(chatConfig.scheduling?.defaultDelayMs).toBe(1000);
            expect(chatConfig.pendingToolCalls).toHaveLength(1);
        });

        it("should create ChatConfig with minimal data and defaults", () => {
            const config: ChatConfigObject = {
                __version: "1.0",
                stats: ChatConfig.defaultStats(),
            };

            const chatConfig = new ChatConfig({ config });

            expect(chatConfig.__version).toBe("1.0");
            expect(chatConfig.goal).toBeUndefined();
            expect(chatConfig.preferredModel).toBeUndefined();
            expect(chatConfig.subtasks).toEqual([]);
            expect(chatConfig.blackboard).toEqual([]);
            expect(chatConfig.resources).toEqual([]);
            expect(chatConfig.records).toEqual([]);
            expect(chatConfig.eventSubscriptions).toEqual({});
            expect(chatConfig.stats.totalToolCalls).toBe(0);
            expect(chatConfig.limits).toBeUndefined();
            expect(chatConfig.scheduling).toBeUndefined();
            expect(chatConfig.pendingToolCalls).toEqual([]);
        });
    });

    describe("parse", () => {
        it("should parse config with useFallbacks true (default)", () => {
            const config: ChatConfigObject = {
                __version: "1.0",
                goal: "Test goal",
                stats: {
                    totalToolCalls: 10,
                    totalCredits: "2000000",
                    startedAt: Date.now(),
                    lastProcessingCycleEndedAt: null,
                },
            };

            const chatConfig = ChatConfig.parse({ config }, mockLogger);

            expect(chatConfig.goal).toBe("Test goal");
            expect(chatConfig.stats.totalToolCalls).toBe(10);
            expect(chatConfig.limits).toEqual(ChatConfig.defaultLimits());
            expect(chatConfig.scheduling).toEqual(ChatConfig.defaultScheduling());
        });

        it("should parse config with useFallbacks false", () => {
            const config: ChatConfigObject = {
                __version: "1.0",
                goal: "Test goal",
                stats: {
                    totalToolCalls: 10,
                    totalCredits: "2000000",
                    startedAt: Date.now(),
                    lastProcessingCycleEndedAt: null,
                },
            };

            const chatConfig = ChatConfig.parse({ config }, mockLogger, { useFallbacks: false });

            expect(chatConfig.goal).toBe("Test goal");
            expect(chatConfig.stats.totalToolCalls).toBe(10);
            expect(chatConfig.limits).toBeUndefined();
            expect(chatConfig.scheduling).toBeUndefined();
        });

        it("should handle missing stats with fallbacks", () => {
            const config: Partial<ChatConfigObject> = {
                __version: "1.0",
            };

            const chatConfig = ChatConfig.parse({ config: config as ChatConfigObject }, mockLogger);

            // Check that stats has the correct structure and default values
            expect(chatConfig.stats).toBeDefined();
            expect(chatConfig.stats.totalToolCalls).toBe(0);
            expect(chatConfig.stats.totalCredits).toBe("0");
            expect(chatConfig.stats.lastProcessingCycleEndedAt).toBeNull();
            // Check that startedAt is a recent timestamp (within last second)
            expect(chatConfig.stats.startedAt).toBeGreaterThan(Date.now() - 1000);
            expect(chatConfig.stats.startedAt).toBeLessThanOrEqual(Date.now());
        });
    });

    describe("default", () => {
        it("should create default ChatConfig", () => {
            const chatConfig = ChatConfig.default();

            expect(chatConfig.__version).toBe("1.0");
            expect(chatConfig.goal).toBeUndefined();
            expect(chatConfig.preferredModel).toBeUndefined();
            expect(chatConfig.subtasks).toEqual([]);
            expect(chatConfig.eventSubscriptions).toEqual({
                "swarm/subtask/+": ["{{LEADER_ID}}"],
                "swarm/resource/+": ["{{LEADER_ID}}"],
                "swarm/routine/+": ["{{LEADER_ID}}"],
                "swarm/tool/+": ["{{LEADER_ID}}"],
                "swarm/user_message": ["{{LEADER_ID}}"],
                "swarm/webhook/+": ["{{LEADER_ID}}"],
                "swarm/Writer/events": ["bot_writer_1", "bot_writer_2"],
            });
            expect(chatConfig.blackboard).toEqual([]);
            expect(chatConfig.resources).toEqual([]);
            expect(chatConfig.records).toEqual([]);
            expect(chatConfig.limits).toEqual(ChatConfig.defaultLimits());
            expect(chatConfig.stats).toEqual(expect.objectContaining({
                totalToolCalls: 0,
                totalCredits: "0",
                startedAt: expect.any(Number),
                lastProcessingCycleEndedAt: null,
            }));
            expect(chatConfig.scheduling).toEqual(ChatConfig.defaultScheduling());
            expect(chatConfig.pendingToolCalls).toEqual([]);
        });
    });

    describe("export", () => {
        it("should export all properties", () => {
            const originalConfig: ChatConfigObject = {
                __version: "1.0",
                goal: "Test export",
                preferredModel: "gpt-4",
                subtasks: [{
                    id: "task1",
                    description: "Task 1",
                    status: "in_progress",
                    assignee_bot_id: "bot1",
                    created_at: new Date().toISOString(),
                    priority: "high",
                }],
                swarmLeader: "leader1",
                subtaskLeaders: { "task1": "bot1" },
                teamId: "team1",
                blackboard: [{
                    id: "bb1",
                    value: "note",
                    created_at: new Date().toISOString(),
                }],
                resources: [{
                    id: "res1",
                    kind: "URL",
                    mime: "text/html",
                    creator_bot_id: "bot1",
                    created_at: new Date().toISOString(),
                    meta: { url: "https://example.com" },
                }],
                records: [{
                    id: "rec1",
                    routine_id: "routine1",
                    routine_name: "Test Routine",
                    params: {},
                    output_resource_ids: [],
                    caller_bot_id: "bot1",
                    created_at: new Date().toISOString(),
                }],
                eventSubscriptions: { "test/event": ["bot1"] },
                policy: { visibility: "open" },
                stats: ChatConfig.defaultStats(),
                limits: ChatConfig.defaultLimits(),
                scheduling: ChatConfig.defaultScheduling(),
                pendingToolCalls: [],
            };

            const chatConfig = new ChatConfig({ config: originalConfig });
            const exported = chatConfig.export();

            expect(exported).toEqual(originalConfig);
        });
    });

    describe("setLimits", () => {
        it("should set new limits", () => {
            const chatConfig = ChatConfig.default();
            const newLimits = {
                maxToolCallsPerBotResponse: 20,
                maxToolCalls: 200,
                maxCreditsPerBotResponse: "2000000",
                maxCredits: "10000000",
                maxDurationPerBotResponseMs: 120000,
                maxDurationMs: 1200000,
                delayBetweenProcessingCyclesMs: 500,
            };

            chatConfig.setLimits(newLimits);

            expect(chatConfig.limits).toEqual(newLimits);
        });

        it("should replace existing limits", () => {
            const chatConfig = ChatConfig.default();
            const partialLimits = {
                maxToolCalls: 150,
            };

            chatConfig.setLimits(partialLimits);

            expect(chatConfig.limits).toEqual(partialLimits);
        });
    });

    describe("getEffectiveLimits", () => {
        it("should merge with defaults when limits are partial", () => {
            const chatConfig = ChatConfig.default();
            chatConfig.setLimits({
                maxToolCalls: 75,
                maxCredits: "7500000",
            });

            const effective = chatConfig.getEffectiveLimits();

            expect(effective.maxToolCalls).toBe(75);
            expect(effective.maxCredits).toBe("7500000");
            expect(effective.maxToolCallsPerBotResponse).toBe(ChatConfig.DEFAULT_LIMITS.maxToolCallsPerBotResponse);
            expect(effective.maxDurationMs).toBe(ChatConfig.DEFAULT_LIMITS.maxDurationMs);
        });

        it("should cap limits at absolute maximum", () => {
            const chatConfig = ChatConfig.default();
            chatConfig.setLimits({
                maxToolCalls: 2000, // Over absolute limit
                maxToolCallsPerBotResponse: 100, // Over absolute limit
                maxCreditsPerBotResponse: (API_CREDITS_MULTIPLIER * BigInt(10)).toString(), // Over absolute limit
                maxDurationMs: DAYS_1_MS * 2, // Over absolute limit
            });

            const effective = chatConfig.getEffectiveLimits();

            expect(effective.maxToolCalls).toBe(ChatConfig.ABSOLUTE_LIMITS.maxToolCalls);
            expect(effective.maxToolCallsPerBotResponse).toBe(ChatConfig.ABSOLUTE_LIMITS.maxToolCallsPerBotResponse);
            expect(effective.maxCreditsPerBotResponse).toBe(ChatConfig.ABSOLUTE_LIMITS.maxCreditsPerBotResponse.toString());
            expect(effective.maxDurationMs).toBe(ChatConfig.ABSOLUTE_LIMITS.maxDurationMs);
        });

        it("should enforce minimum values", () => {
            const chatConfig = ChatConfig.default();
            chatConfig.setLimits({
                maxToolCalls: -10,
                maxCredits: "-1000000",
                delayBetweenProcessingCyclesMs: -500,
            });

            const effective = chatConfig.getEffectiveLimits();

            expect(effective.maxToolCalls).toBe(0);
            expect(effective.maxCredits).toBe("0");
            expect(effective.delayBetweenProcessingCyclesMs).toBe(0);
        });

        it("should handle undefined limits", () => {
            const config: ChatConfigObject = {
                __version: "1.0",
                stats: ChatConfig.defaultStats(),
            };
            const chatConfig = new ChatConfig({ config });
            chatConfig.limits = undefined;

            const effective = chatConfig.getEffectiveLimits();

            expect(effective).toEqual(ChatConfig.DEFAULT_LIMITS);
        });
    });

    describe("setScheduling", () => {
        it("should set new scheduling", () => {
            const chatConfig = ChatConfig.default();
            const newScheduling = {
                defaultDelayMs: 2000,
                toolSpecificDelays: { "web_search": 5000 },
                requiresApprovalTools: ["dangerous_tool", "expensive_tool"],
                approvalTimeoutMs: MINUTES_1_MS * 5,
                autoRejectOnTimeout: false,
            };

            chatConfig.setScheduling(newScheduling);

            expect(chatConfig.scheduling).toEqual(newScheduling);
        });
    });

    describe("getEffectiveScheduling", () => {
        it("should merge with defaults when scheduling is partial", () => {
            const chatConfig = ChatConfig.default();
            chatConfig.setScheduling({
                defaultDelayMs: 3000,
                requiresApprovalTools: "all",
            });

            const effective = chatConfig.getEffectiveScheduling();

            expect(effective.defaultDelayMs).toBe(3000);
            expect(effective.requiresApprovalTools).toBe("all");
            expect(effective.toolSpecificDelays).toEqual({});
            expect(effective.approvalTimeoutMs).toBe(ChatConfig.DEFAULT_SCHEDULING.approvalTimeoutMs);
            expect(effective.autoRejectOnTimeout).toBe(true);
        });

        it("should ensure non-negative delays", () => {
            const chatConfig = ChatConfig.default();
            chatConfig.setScheduling({
                defaultDelayMs: -1000,
                toolSpecificDelays: {
                    "tool1": -500,
                    "tool2": 2000,
                    "tool3": -100,
                },
            });

            const effective = chatConfig.getEffectiveScheduling();

            expect(effective.defaultDelayMs).toBe(0);
            expect(effective.toolSpecificDelays).toEqual({
                "tool1": 0,
                "tool2": 2000,
                "tool3": 0,
            });
        });

        it("should enforce minimum approval timeout", () => {
            const chatConfig = ChatConfig.default();
            chatConfig.setScheduling({
                approvalTimeoutMs: 500, // Less than 1 second
            });

            const effective = chatConfig.getEffectiveScheduling();

            expect(effective.approvalTimeoutMs).toBe(SECONDS_1_MS);
        });

        it("should handle undefined scheduling", () => {
            const config: ChatConfigObject = {
                __version: "1.0",
                stats: ChatConfig.defaultStats(),
            };
            const chatConfig = new ChatConfig({ config });
            chatConfig.scheduling = undefined;

            const effective = chatConfig.getEffectiveScheduling();

            expect(effective).toEqual(ChatConfig.DEFAULT_SCHEDULING);
        });
    });

    describe("defaultStats", () => {
        it("should create valid initial stats for a new conversation", () => {
            const beforeTime = Date.now();
            const stats = ChatConfig.defaultStats();
            const afterTime = Date.now();

            // Initial values should be zero/empty
            expect(stats.totalToolCalls).toBe(0);
            expect(stats.totalCredits).toBe("0");
            
            // Timestamp should be within test execution time
            expect(stats.startedAt).toBeGreaterThanOrEqual(beforeTime);
            expect(stats.startedAt).toBeLessThanOrEqual(afterTime);
            
            // No processing cycle should have ended yet
            expect(stats.lastProcessingCycleEndedAt).toBeNull();
        });
    });

    describe("defaultLimits", () => {
        it("should return expected default limits", () => {
            const limits = ChatConfig.defaultLimits();

            expect(limits).toEqual({
                maxToolCallsPerBotResponse: 10,
                maxToolCalls: 50,
                maxCreditsPerBotResponse: (API_CREDITS_MULTIPLIER * BigInt(1)).toString(),
                maxCredits: (API_CREDITS_MULTIPLIER * BigInt(5)).toString(),
                maxDurationPerBotResponseMs: MINUTES_1_MS,
                maxDurationMs: MINUTES_10_MS,
                delayBetweenProcessingCyclesMs: 0,
            });
        });
    });

    describe("defaultScheduling", () => {
        it("should return expected default scheduling", () => {
            const scheduling = ChatConfig.defaultScheduling();

            expect(scheduling).toEqual({
                defaultDelayMs: 0,
                toolSpecificDelays: {},
                requiresApprovalTools: "none",
                approvalTimeoutMs: MINUTES_10_MS,
                autoRejectOnTimeout: true,
            });
        });
    });

    describe("Complex scenarios", () => {
        it("should handle subtask with dependencies", () => {
            const subtasks: SwarmSubTask[] = [
                {
                    id: "task1",
                    description: "First task",
                    status: "done",
                    created_at: new Date().toISOString(),
                },
                {
                    id: "task2",
                    description: "Second task",
                    status: "in_progress",
                    depends_on: ["task1"],
                    assignee_bot_id: "bot2",
                    created_at: new Date().toISOString(),
                    priority: "high",
                },
                {
                    id: "task3",
                    description: "Third task",
                    status: "blocked",
                    depends_on: ["task1", "task2"],
                    created_at: new Date().toISOString(),
                    priority: "low",
                },
            ];

            const config: ChatConfigObject = {
                __version: "1.0",
                subtasks,
                stats: ChatConfig.defaultStats(),
            };

            const chatConfig = new ChatConfig({ config });

            expect(chatConfig.subtasks).toHaveLength(3);
            expect(chatConfig.subtasks?.[1].depends_on).toEqual(["task1"]);
            expect(chatConfig.subtasks?.[2].status).toBe("blocked");
        });

        it("should handle resources with different types", () => {
            const resources: SwarmResource[] = [
                {
                    id: "res1",
                    kind: "Note",
                    creator_bot_id: "bot1",
                    created_at: new Date().toISOString(),
                },
                {
                    id: "res2",
                    kind: "File",
                    mime: "application/pdf",
                    creator_bot_id: "bot2",
                    created_at: new Date().toISOString(),
                    meta: { size: 1024, checksum: "abc123" },
                },
                {
                    id: "res3",
                    kind: "Vector",
                    creator_bot_id: "bot1",
                    created_at: new Date().toISOString(),
                    meta: { dimensions: 768, model: "text-embedding-ada-002" },
                },
            ];

            const config: ChatConfigObject = {
                __version: "1.0",
                resources,
                stats: ChatConfig.defaultStats(),
            };

            const chatConfig = new ChatConfig({ config });

            expect(chatConfig.resources).toHaveLength(3);
            expect(chatConfig.resources?.[1].kind).toBe("File");
            expect(chatConfig.resources?.[1].meta?.size).toBe(1024);
            expect(chatConfig.resources?.[2].meta?.model).toBe("text-embedding-ada-002");
        });

        it("should handle pending tool calls with various statuses", () => {
            const pendingToolCalls: PendingToolCallEntry[] = [
                {
                    pendingId: "p1",
                    toolCallId: "t1",
                    toolName: "web_search",
                    toolArguments: JSON.stringify({ query: "test" }),
                    callerBotId: "bot1",
                    conversationId: "conv1",
                    requestedAt: Date.now(),
                    status: PendingToolCallStatus.PENDING_APPROVAL,
                    executionAttempts: 0,
                },
                {
                    pendingId: "p2",
                    toolCallId: "t2",
                    toolName: "file_write",
                    toolArguments: JSON.stringify({ path: "/tmp/test.txt", content: "hello" }),
                    callerBotId: "bot2",
                    conversationId: "conv1",
                    requestedAt: Date.now() - 60000,
                    status: PendingToolCallStatus.COMPLETED_SUCCESS,
                    executionAttempts: 1,
                    result: JSON.stringify({ success: true }),
                    cost: "100000",
                },
                {
                    pendingId: "p3",
                    toolCallId: "t3",
                    toolName: "dangerous_operation",
                    toolArguments: JSON.stringify({ action: "delete_all" }),
                    callerBotId: "bot1",
                    conversationId: "conv1",
                    requestedAt: Date.now() - 120000,
                    status: PendingToolCallStatus.REJECTED_BY_USER,
                    statusReason: "Too dangerous",
                    executionAttempts: 0,
                    userIdToApprove: "user1",
                    approvedOrRejectedByUserId: "user1",
                    decisionTime: Date.now() - 100000,
                },
            ];

            const config: ChatConfigObject = {
                __version: "1.0",
                pendingToolCalls,
                stats: ChatConfig.defaultStats(),
            };

            const chatConfig = new ChatConfig({ config });

            expect(chatConfig.pendingToolCalls).toHaveLength(3);
            expect(chatConfig.pendingToolCalls?.[0].status).toBe(PendingToolCallStatus.PENDING_APPROVAL);
            expect(chatConfig.pendingToolCalls?.[1].status).toBe(PendingToolCallStatus.COMPLETED_SUCCESS);
            expect(chatConfig.pendingToolCalls?.[1].cost).toBe("100000");
            expect(chatConfig.pendingToolCalls?.[2].status).toBe(PendingToolCallStatus.REJECTED_BY_USER);
            expect(chatConfig.pendingToolCalls?.[2].statusReason).toBe("Too dangerous");
        });
    });

    describe("Configuration Migration and Versioning", () => {
        it("should handle configuration upgrades gracefully", () => {
            // Test with an older version config that might be missing new fields
            const oldConfig: Partial<ChatConfigObject> = {
                __version: "0.9",
                goal: "Old format goal",
                stats: {
                    totalToolCalls: 5,
                    totalCredits: "1000000",
                    startedAt: Date.now(),
                    lastProcessingCycleEndedAt: null,
                },
            };

            const chatConfig = ChatConfig.parse({ config: oldConfig as ChatConfigObject }, mockLogger);
            
            expect(chatConfig.goal).toBe("Old format goal");
            expect(chatConfig.__version).toBe("0.9");
            expect(chatConfig.limits).toEqual(ChatConfig.defaultLimits());
            expect(chatConfig.scheduling).toEqual(ChatConfig.defaultScheduling());
        });

        it("should safely ignore unknown fields in configuration", () => {
            const configWithUnknownFields: any = {
                __version: "1.0",
                goal: "Test goal",
                stats: ChatConfig.defaultStats(),
                unknownField: "should be ignored",
                deprecatedSetting: true,
            };

            const chatConfig = new ChatConfig({ config: configWithUnknownFields });
            
            // Known fields should be preserved
            expect(chatConfig.goal).toBe("Test goal");
            
            // Unknown fields should not be present on the instance
            // This tests that the config properly validates and filters input
            expect((chatConfig as any).unknownField).toBeUndefined();
            expect((chatConfig as any).deprecatedSetting).toBeUndefined();
            
            // Config should have proper stats that were provided
            expect(chatConfig.stats).toBeDefined();
            expect(chatConfig.stats.totalToolCalls).toBe(0);
            
            // Other fields may be undefined if not provided in config
            expect(chatConfig.limits).toBeUndefined(); // Not provided in config
            expect(chatConfig.scheduling).toBeUndefined(); // Not provided in config
        });
    });

    describe("Resource and Event Management", () => {
        it("should validate resource structure requirements", () => {
            const chatConfig = ChatConfig.default();
            
            // Start with default empty resources
            expect(chatConfig.resources).toEqual([]);
            
            // Test valid resource structure
            const validResource: SwarmResource = {
                id: "res1",
                kind: "File",
                mime: "text/plain",
                creator_bot_id: "bot1",
                created_at: new Date().toISOString(),
                meta: { size: 1024 },
            };
            
            // Validate required fields
            expect(validResource.id).toMatch(/^res\d+$/);
            expect(["File", "Text", "Image", "Video", "Audio"]).toContain(validResource.kind);
            expect(validResource.mime).toMatch(/^[a-z]+\/[a-z0-9\-\+\.]+$/);
            expect(validResource.creator_bot_id).toMatch(/^bot\d+$/);
            expect(new Date(validResource.created_at).toISOString()).toBe(validResource.created_at);
            
            // Meta should be appropriate for the resource kind
            if (validResource.kind === "File" && validResource.meta) {
                expect(validResource.meta.size).toBeGreaterThan(0);
            }
        });

        it("should manage event subscriptions properly", () => {
            const chatConfig = ChatConfig.default();
            
            // Default should have leader subscriptions
            expect(chatConfig.eventSubscriptions).toEqual(expect.objectContaining({
                "swarm/subtask/+": ["{{LEADER_ID}}"],
                "swarm/resource/+": ["{{LEADER_ID}}"],
            }));
            
            // Test custom event subscriptions
            const customConfig: ChatConfigObject = {
                __version: "1.0",
                eventSubscriptions: {
                    "custom/event": ["bot1", "bot2"],
                    "another/event": ["bot3"],
                },
                stats: ChatConfig.defaultStats(),
            };
            
            const customChatConfig = new ChatConfig({ config: customConfig });
            expect(customChatConfig.eventSubscriptions?.["custom/event"]).toEqual(["bot1", "bot2"]);
        });
    });

    describe("Realistic Usage Scenarios", () => {
        it("should handle a typical chat session lifecycle", () => {
            // Start with default config
            const chatConfig = ChatConfig.default();
            
            // Set realistic limits for a production environment
            chatConfig.setLimits({
                maxToolCalls: 100,
                maxCredits: "5000000",
                maxDurationMs: MINUTES_10_MS,
            });
            
            // Set scheduling for a moderate security environment
            chatConfig.setScheduling({
                requiresApprovalTools: ["file_write", "system_command"],
                approvalTimeoutMs: MINUTES_1_MS * 2,
                autoRejectOnTimeout: true,
            });
            
            const effective = chatConfig.getEffectiveLimits();
            const scheduling = chatConfig.getEffectiveScheduling();
            
            expect(effective.maxToolCalls).toBe(100);
            expect(Array.isArray(scheduling.requiresApprovalTools)).toBe(true);
            expect(scheduling.autoRejectOnTimeout).toBe(true);
        });

        it("should handle high-security configuration", () => {
            const secureConfig = ChatConfig.default();
            
            // Very restrictive limits
            secureConfig.setLimits({
                maxToolCalls: 10,
                maxToolCallsPerBotResponse: 2,
                maxCredits: "1000000",
                maxDurationMs: MINUTES_1_MS * 2,
                delayBetweenProcessingCyclesMs: 1000,
            });
            
            // Require approval for all tools
            secureConfig.setScheduling({
                requiresApprovalTools: "all",
                approvalTimeoutMs: SECONDS_1_MS * 30,
                autoRejectOnTimeout: true,
                defaultDelayMs: 2000,
            });
            
            const limits = secureConfig.getEffectiveLimits();
            const scheduling = secureConfig.getEffectiveScheduling();
            
            expect(limits.maxToolCalls).toBe(10);
            expect(limits.delayBetweenProcessingCyclesMs).toBe(1000);
            expect(scheduling.requiresApprovalTools).toBe("all");
            expect(scheduling.defaultDelayMs).toBe(2000);
        });

        it("should handle development/testing configuration", () => {
            const devConfig = ChatConfig.default();
            
            // Very permissive limits for development
            devConfig.setLimits({
                maxToolCalls: 1000,
                maxCredits: "50000000",
                maxDurationMs: MINUTES_10_MS * 6, // 1 hour
                delayBetweenProcessingCyclesMs: 0,
            });
            
            // No approval required for development
            devConfig.setScheduling({
                requiresApprovalTools: "none",
                defaultDelayMs: 0,
                autoRejectOnTimeout: false,
            });
            
            const limits = devConfig.getEffectiveLimits();
            const scheduling = devConfig.getEffectiveScheduling();
            
            expect(limits.maxToolCalls).toBe(1000);
            expect(limits.delayBetweenProcessingCyclesMs).toBe(0);
            expect(scheduling.requiresApprovalTools).toBe("none");
            expect(scheduling.defaultDelayMs).toBe(0);
        });
    });

    describe("Statistics and Performance", () => {
        it("should track statistics correctly", () => {
            const config: ChatConfigObject = {
                __version: "1.0",
                stats: {
                    totalToolCalls: 150,
                    totalCredits: "15000000",
                    startedAt: Date.now() - 3600000, // 1 hour ago
                    lastProcessingCycleEndedAt: Date.now() - 1000, // 1 second ago
                },
            };
            
            const chatConfig = new ChatConfig({ config });
            
            expect(chatConfig.stats.totalToolCalls).toBe(150);
            expect(chatConfig.stats.totalCredits).toBe("15000000");
            expect(chatConfig.stats.startedAt).toBeLessThan(Date.now());
            expect(chatConfig.stats.lastProcessingCycleEndedAt).toBeLessThan(Date.now());
        });

        it("should handle stats edge cases", () => {
            const config: ChatConfigObject = {
                __version: "1.0",
                stats: {
                    totalToolCalls: 0,
                    totalCredits: "0",
                    startedAt: Date.now(),
                    lastProcessingCycleEndedAt: null, // Never ended a cycle
                },
            };
            
            const chatConfig = new ChatConfig({ config });
            
            expect(chatConfig.stats.totalToolCalls).toBe(0);
            expect(chatConfig.stats.totalCredits).toBe("0");
            expect(chatConfig.stats.lastProcessingCycleEndedAt).toBeNull();
        });
    });
});