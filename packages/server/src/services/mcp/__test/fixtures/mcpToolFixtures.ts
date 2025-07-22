/**
 * Test fixtures for MCP tools and related operations
 * Provides mock data and helper functions for testing MCP tool functionality
 */

import { generatePK, DEFAULT_LANGUAGE, ResourceType, ResourceSubType, McpToolName } from "@vrooli/shared";
import type { 
    SessionUser, 
    ChatMessageCreateInput,
    ResourceVersionCreateInput,
    TeamCreateInput,
    BotCreateInput,
    MessageConfigObject,
    TaskContextInfo,
} from "@vrooli/shared";
import type { 
    DefineToolParams, 
    ResourceManageParams, 
    SendMessageParams, 
    RunRoutineParams, 
    SpawnSwarmParams,
    EndSwarmParams,
    UpdateSwarmSharedStateParams,
    Recipient, 
} from "../types/tools.js";

/**
 * Test user fixtures for MCP operations
 */
export const mcpTestUser: SessionUser = {
    id: generatePK().toString(),
    name: "MCP Test User",
    handle: "mcp_tester",
    email: "mcp.test@example.com",
    languages: [DEFAULT_LANGUAGE],
    isLoggedIn: true,
    roles: [],
    wallets: [],
    walletIdCurrent: null,
    isAdmin: false,
    theme: "light",
};

export const mcpAdminUser: SessionUser = {
    ...mcpTestUser,
    id: generatePK().toString(),
    name: "MCP Admin User",
    handle: "mcp_admin",
    email: "mcp.admin@example.com",
    isAdmin: true,
};

/**
 * Mock schema data for DefineTool tests
 */
export const mockResourceManageSchema = {
    name: McpToolName.ResourceManage,
    description: "Manage resources through CRUD operations",
    inputSchema: {
        anyOf: [
            {
                description: "Find resources",
                type: "object",
                properties: {
                    op: { type: "string", const: "find" },
                    resource_type: { type: "string" },
                    filters: { type: "object" },
                },
                required: ["op", "resource_type"],
                additionalProperties: false,
            },
            {
                description: "Add a new resource",
                type: "object", 
                properties: {
                    op: { type: "string", const: "add" },
                    resource_type: { type: "string" },
                    attributes: { type: "object" },
                },
                required: ["op", "resource_type", "attributes"],
                additionalProperties: false,
            },
            {
                description: "Update a resource",
                type: "object",
                properties: {
                    op: { type: "string", const: "update" },
                    id: { type: "string" },
                    resource_type: { type: "string" },
                    attributes: { type: "object" },
                },
                required: ["op", "id", "resource_type", "attributes"],
                additionalProperties: false,
            },
            {
                description: "Delete a resource",
                type: "object",
                properties: {
                    op: { type: "string", const: "delete" },
                    id: { type: "string" },
                },
                required: ["op", "id"],
                additionalProperties: false,
            },
        ],
    },
    annotations: {
        title: "Resource Management Tool",
        readOnlyHint: false,
        openWorldHint: true,
    },
};

/**
 * DefineTool parameter fixtures
 */
export const defineToolFixtures = {
    resourceManageFind: {
        toolName: McpToolName.ResourceManage,
        variant: "Note",
        op: "find",
    } as DefineToolParams,

    resourceManageAdd: {
        toolName: McpToolName.ResourceManage,
        variant: "Project", 
        op: "add",
    } as DefineToolParams,

    resourceManageUpdate: {
        toolName: McpToolName.ResourceManage,
        variant: "Team",
        op: "update", 
    } as DefineToolParams,

    resourceManageDelete: {
        toolName: McpToolName.ResourceManage,
        variant: "Bot",
        op: "delete",
    } as DefineToolParams,

    unsupportedTool: {
        toolName: "UnsupportedTool" as any,
        variant: "Note",
        op: "find",
    } as DefineToolParams,
};

/**
 * SendMessage parameter fixtures
 */
export const sendMessageFixtures = {
    toChatId: {
        recipient: { kind: "chat", chatId: generatePK().toString() } as Recipient,
        content: "Hello from MCP tool test",
        metadata: {
            mcpLlmModel: "claude-3-sonnet-20240229",
            mcpLlmTaskContexts: [
                {
                    id: "test-context",
                    priority: "high",
                    description: "Test task context",
                } as TaskContextInfo,
            ] as TaskContextInfo[],
            messageConfig: {
                __version: "1.0",
                resources: [],
                role: "assistant",
            } as MessageConfigObject,
        },
    } as SendMessageParams,

    toBotId: {
        recipient: { kind: "bot", botId: generatePK().toString() } as Recipient,
        content: "Message to bot",
    } as SendMessageParams,

    toUserId: {
        recipient: { kind: "user", userId: generatePK().toString() } as Recipient,
        content: "Direct message to user",
    } as SendMessageParams,

    toTopic: {
        recipient: { kind: "topic", topic: "general_updates" } as Recipient,
        content: "Message to topic",
    } as SendMessageParams,

    withComplexContent: {
        recipient: { kind: "chat", chatId: generatePK().toString() } as Recipient,
        content: {
            type: "structured",
            data: { analysis: "Complex data object", score: 95 },
        },
        metadata: {
            mcpLlmModel: "gpt-4-turbo",
        },
    } as SendMessageParams,

    invalidRecipient: {
        recipient: { kind: "unknown" } as any,
        content: "Test message",
    } as SendMessageParams,
};

/**
 * ResourceManage parameter fixtures
 */
export const resourceManageFixtures = {
    findNotes: {
        op: "find",
        resource_type: "Note",
        filters: {
            isPrivate: false,
            createdTimeFrame: { after: "2024-01-01" },
        },
    } as ResourceManageParams,

    findProjects: {
        op: "find", 
        resource_type: "Project",
        filters: {},
    } as ResourceManageParams,

    addNote: {
        op: "add",
        resource_type: "Note",
        attributes: {
            name: "Test Note from MCP",
            content: "This is a test note created via MCP tool",
            tagsConnect: ["test", "mcp"],
        },
    } as ResourceManageParams,

    addProject: {
        op: "add",
        resource_type: "Project", 
        attributes: {
            name: "MCP Test Project",
            handle: "mcp-test-project",
            isPrivate: false,
            config: {
                theme: "default",
                features: ["collaboration", "version_control"],
            },
            tagsConnect: ["mcp", "testing"],
        },
    } as ResourceManageParams,

    addTeam: {
        op: "add",
        resource_type: "Team",
        attributes: {
            handle: "mcp-test-team",
            isPrivate: false,
            config: {
                __version: "1.0",
                permissions: {
                    memberPermissions: ["read", "write"],
                    adminPermissions: ["read", "write", "admin"],
                },
            },
            memberInvitesCreate: [
                {
                    userConnect: generatePK().toString(),
                    message: "Welcome to MCP test team",
                    willBeAdmin: false,
                },
            ],
        },
    } as ResourceManageParams,

    addBot: {
        op: "add",
        resource_type: "Bot",
        attributes: {
            handle: "mcp-test-bot",
            isPrivate: false,
            config: {
                model: "claude-3-sonnet-20240229",
                temperature: 0.7,
                systemPrompt: "You are a helpful test bot created via MCP",
            },
        },
    } as ResourceManageParams,

    updateNote: {
        op: "update",
        id: generatePK().toString(),
        resource_type: "Note",
        attributes: {
            name: "Updated Test Note",
            content: "Updated content via MCP",
            tagsConnect: ["updated"],
        },
    } as ResourceManageParams,

    updateTeam: {
        op: "update", 
        id: generatePK().toString(),
        resource_type: "Team",
        attributes: {
            config: {
                __version: "1.0",
                permissions: {
                    memberPermissions: ["read"],
                    adminPermissions: ["read", "write", "admin"],
                },
            },
            memberInvitesCreate: [
                {
                    userConnect: generatePK().toString(),
                    message: "Updated team invitation",
                    willBeAdmin: true,
                },
            ],
        },
    } as ResourceManageParams,

    deleteResource: {
        op: "delete",
        id: generatePK().toString(),
    } as ResourceManageParams,

    invalidOperation: {
        op: "invalid" as any,
        resource_type: "Note",
    } as ResourceManageParams,
};

/**
 * RunRoutine parameter fixtures
 */
export const runRoutineFixtures = {
    startBasic: {
        action: "start",
        routineId: generatePK().toString(),
        inputs: {},
        mode: "sync",
        priority: "normal",
    } as RunRoutineParams,

    startWithInputs: {
        action: "start",
        routineId: generatePK().toString(),
        inputs: {
            userInput: "Process this data",
            options: { format: "json", includeMetadata: true },
        },
        mode: "async",
        priority: "high",
    } as RunRoutineParams,

    pauseRun: {
        action: "pause",
        runId: generatePK().toString(),
    } as RunRoutineParams,

    resumeRun: {
        action: "resume",
        runId: generatePK().toString(),
    } as RunRoutineParams,

    cancelRun: {
        action: "cancel",
        runId: generatePK().toString(),
    } as RunRoutineParams,

    statusRun: {
        action: "status",
        runId: generatePK().toString(),
    } as RunRoutineParams,
};

/**
 * SpawnSwarm parameter fixtures
 */
export const spawnSwarmFixtures = {
    simpleSwarm: {
        kind: "simple",
        goal: "Complete data analysis task",
        swarmLeader: "analyst-bot",
        reservation: {
            creditsPct: 30,
            messagesPct: 25,
            durationPct: 40,
        },
    } as SpawnSwarmParams,

    richSwarm: {
        kind: "rich",
        goal: "Execute complex multi-step project",
        teamId: generatePK().toString(),
        subtasks: [
            {
                id: generatePK().toString(),
                title: "Research phase",
                description: "Conduct initial research",
                status: "pending",
                priority: "high",
                assignedTo: "researcher-bot",
                created_at: new Date().toISOString(),
            },
            {
                id: generatePK().toString(),
                title: "Development phase", 
                description: "Build the solution",
                status: "pending",
                priority: "medium",
                assignedTo: "developer-bot",
                created_at: new Date().toISOString(),
            },
        ],
        reservation: {
            creditsPct: 60,
            messagesPct: 50,
            durationPct: 70,
        },
        policy: {
            maxConcurrentTasks: 3,
            timeoutMinutes: 120,
            autoRetry: true,
        },
        eventSubscriptions: ["task_completed", "error_occurred"],
    } as SpawnSwarmParams,
};

/**
 * SwarmTools parameter fixtures
 */
export const swarmToolsFixtures = {
    updateSubTasks: {
        subTasks: {
            set: [
                {
                    id: generatePK().toString(),
                    title: "Updated task",
                    description: "Task updated via MCP",
                    status: "in_progress",
                    priority: "high",
                    assignedTo: "test-bot",
                    created_at: new Date().toISOString(),
                },
            ],
            delete: [generatePK().toString()],
        },
    } as UpdateSwarmSharedStateParams,

    updateBlackboard: {
        blackboard: {
            set: [
                { id: "analysis_results", value: { score: 85, confidence: 0.92 } },
                { id: "user_preferences", value: { theme: "dark", language: "en" } },
            ],
            delete: ["old_data", "temp_cache"],
        },
    } as UpdateSwarmSharedStateParams,

    updateTeamConfig: {
        teamConfig: {
            __version: "1.0",
            permissions: {
                memberPermissions: ["read", "write"],
                adminPermissions: ["read", "write", "admin", "delete"],
            },
            settings: {
                allowPublicJoin: false,
                requireApproval: true,
            },
        },
    } as UpdateSwarmSharedStateParams,

    endSwarmGraceful: {
        mode: "graceful",
        reason: "Goal completed successfully",
    } as EndSwarmParams,

    endSwarmForce: {
        mode: "force",
        reason: "Critical error - force termination",
    } as EndSwarmParams,
};

/**
 * Mock conversation state for SwarmTools tests
 */
export const mockConversationState = {
    conversationId: generatePK().toString(),
    userId: mcpTestUser.id,
    config: {
        teamId: generatePK().toString(),
        subtasks: [
            {
                id: generatePK().toString(),
                title: "Initial task",
                description: "Starting task",
                status: "pending",
                priority: "medium",
                assignedTo: "default-bot",
                created_at: new Date().toISOString(),
            },
        ],
        blackboard: [
            { id: "init_data", value: { initialized: true } },
        ],
    },
};

/**
 * Mock active swarm registry entries
 */
export const mockSwarmRegistry = {
    getOrderedRecords: () => [
        {
            id: generatePK().toString(),
            createdAt: new Date(),
            userId: mcpTestUser.id,
        },
    ],
    get: (id: string) => ({
        getAssociatedUserId: () => mcpTestUser.id,
        getState: () => "running",
        pause: () => Promise.resolve(),
        stop: () => Promise.resolve({ 
            success: true,
            message: "Swarm stopped successfully",
            finalState: {
                endedAt: new Date().toISOString(),
                reason: "User requested",
                mode: "graceful" as const,
                totalSubTasks: 3,
                completedSubTasks: 2,
                totalCreditsUsed: "150",
                totalToolCalls: 12,
            },
        }),
    }),
};

/**
 * Mock queue processing results
 */
export const mockQueueResults = {
    runSuccess: { success: true, message: "Run queued successfully" },
    runFailure: { success: false, error: "Failed to queue run" },
    swarmSuccess: { success: true, message: "Swarm spawned successfully" },
    swarmFailure: { success: false, error: "Failed to spawn swarm" },
};
