import { ModelType, ResourceSubType, ResourceType } from "@local/shared";
import { Logger, Tool, ToolResponse } from "./types.js";

type ToolHandler = (args: any, logger: Logger) => Promise<ToolResponse>;
type JSONValue = null | boolean | number | string | JSONValue[] | { [key: string]: JSONValue };

/**
 * Available root-level tools, ordered by (likely) frequency of use.
 */
export enum McpToolName {
    /** Get detailed parameters for other tools based on a variant */
    DefineTool = "define_tool",
    /** Sends a message to a new or existing chat. Doubles as event bus for agents to react to events. */
    SendMessage = "send_message",
    /** CRUD for *any* resource */
    ResourceManage = "resource_manage",
    /** Run a routine (dynamic tool) inline (synchronous, no run object created) or as a job (asynchronous, run object created) */
    RunRoutine = "run_routine",
    /** Starts a session with a bot or team of bots */
    StartSession = "start_session",
}

/**
 * Available session-level tools, ordered by (likely) frequency of use.
 */
export enum McpSessionToolName {
    /** Check session infomration/status/policy */
    CheckSession = "check_session",
    /** Update session status/logs */
    UpdateSession = "update_session",
    /** End the session */
    EndSession = "end_session",
}

/**
 * Available routine-level tools, ordered by (likely) frequency of use.
 * 
 * These are only available when the MCP server is set up for a specific routine, 
 * rather than the site as a whole
 */
export enum McpRoutineToolName {
    /** Start a routine */
    StartRoutine = "start_routine",
    /** Stop a routine */
    StopRoutine = "stop_routine",
}

/**
 * Available resource types, ordered by (likely) frequency of use.
 */
const resourceTypes = [
    // Workflows
    [ResourceSubType.RoutineMultiStep, "A graph or sequence of routines"],
    [ResourceSubType.RoutineInternalAction, "Perform an action within Vrooli (e.g. 'create a new note')"],
    [ResourceSubType.RoutineApi, "Call an API"],
    [ResourceSubType.RoutineCode, "Run data conversion code"],
    [ResourceSubType.RoutineData, "Data (for passing into other routines)"],
    [ResourceSubType.RoutineGenerate, "Generative AI (e.g. LLM, image generation)"],
    [ResourceSubType.RoutineInformational, "Contains only context information/instructions/resources"],
    [ResourceSubType.RoutineSmartContract, "Run a smart contract"],
    [ResourceSubType.RoutineWeb, "Perform a web search"],
    // Agency
    ["Bot", "An autonomous agent"],
    [ModelType.Team, "Team of bots and/or humans"],
    // Standards
    [ResourceSubType.StandardPrompt, "AI prompt"],
    [ResourceSubType.StandardDataStructure, "Data structure"],
    // Organization
    [ResourceType.Project, "Group resources together"],
    // Other resources
    [ResourceType.Note, "Store short memos/thoughts/etc."],
    ["ExternalData", "KV bucket, table, S3 object, vector index, etc."]
]

/**
 * Parameters for defining how to use a tool.
 */
export interface DefineToolParams {
    toolName: McpToolName.ResourceManage;
    /** Resource subtype (e.g. "Note", "RoutineApi") */
    variant: string;
    /** Which CRUD op's parameter schema? */
    op: "add" | "find" | "update" | "delete";
}

/**
 * Parameters for sending a message or broadcasting an event.
 */
export interface SendMessageParams {
    /** Recipient – single agent / team / chat thread or the literal string "broadcast" */
    to: string;
    /** Mode of the message (defaults to "chat") */
    type?: "chat" | "event";
    /** Optional topic / routing key for pub‑sub style delivery */
    topic?: string;
    /** Free‑form payload – plain text or structured JSON */
    content: string | JSONValue;
    /** Additional opaque metadata */
    metadata?: Record<string, JSONValue>;
}

/**
 * Parameters for managing a resource.
 */
export interface ResourceManageParams {
    op: "add" | "find" | "update" | "delete";
    id?: string; // required when op in update|delete
    query?: string; // used when op=find
    resource_type: string;
    attributes?: Record<string, any>; // add/update payload
    filters?: Record<string, any>; // find filters
}

/**
 * Parameters for running a routine.
 */
export interface RunRoutineParams {
    /** start | pause | resume | cancel | status (default: start) */
    op?: "start" | "pause" | "resume" | "cancel" | "status";
    /** Existing run identifier (required for non‑start ops) */
    run_id?: string;
    /** Routine to execute when op === "start" */
    routine_id?: string;
    /** sync (blocking) | async (background) – default is sync */
    mode?: "sync" | "async";
    /** ISO‑8601 timestamp OR cron expression (when mode === "async") */
    schedule?: {
        /** RFC 3339 instant, e.g. "2025-05-09T14:30:00Z" */
        at?: string;
        /** Cron pattern interpreted with server TZ unless timezone is given */
        cron?: string;
        /** Optional IANA TZ, e.g. "America/New_York" */
        timezone?: string;
    };
    /** Mark run as time‑critical (may influence queue priority) */
    time_sensitive?: boolean;
    /** Fine‑tune the executing bot */
    bot_config?: {
        starting_prompt?: string;
        responding_bot_id?: string;
    };
}

/**
 * Parameters for starting a session.
 */
export interface StartSessionParams {
    /** Either explicit bot IDs or a single team ID (one must be provided) */
    bot_ids?: string[];
    team_id?: string;
    /** High‑level goal / intent for this session */
    purpose?: string;
    /** Arbitrary initial context shared with participants */
    context?: string | JSONValue;
    /** Governance or ACL settings */
    policy?: {
        type?: "open" | "restricted" | "private";
        acl?: string[]; // allowed agent IDs
    };
    /** Usage caps & auto‑expiry */
    limits?: {
        max_messages?: number;
        max_duration_minutes?: number;
        max_credits?: number;
        expires_at?: string; // ISO‑8601 timestamp
    };
}

// Create a structure for JSON Schema oneOf with descriptions
const resourceVariantSchemaItems = resourceTypes.map(item => ({
    const: String(item[0]),
    description: item[1] || String(item[0]) // Use provided description or the name itself
}));

/**
 * How the user of the MCP sees what tools are available, what they do, and how to use them.
 */
const builtInToolDefinitions: Map<McpToolName, Tool> = new Map([
    [McpToolName.DefineTool, {
        name: McpToolName.DefineTool,
        description: "Return the detailed input schema for ResourceManage given a resource variant *and* CRUD op.",
        inputSchema: {
            type: "object",
            properties: {
                toolName: {
                    type: "string",
                    enum: [McpToolName.ResourceManage],
                },
                variant: {
                    type: "string",
                    oneOf: resourceVariantSchemaItems,
                },
                op: {
                    type: "string",
                    enum: ["add", "find", "update", "delete"],
                },
            },
            required: ["toolName", "variant", "op"],
            additionalProperties: false,
        },
        annotations: {
            title: "Define Tool Parameters",
            readOnlyHint: true,
            openWorldHint: false,
        },
    }],
    [McpToolName.SendMessage, {
        name: McpToolName.SendMessage,
        description: "Send a chat message or publish an event for agents to react to.",
        inputSchema: {
            type: "object",
            properties: {
                to: { type: "string", description: "Recipient agent/team/thread or 'broadcast'." },
                type: { type: "string", enum: ["chat", "event"], default: "chat" },
                topic: { type: "string", description: "Topic / routing key (for events)." },
                content: { oneOf: [{ type: "string" }, { type: "object" }], description: "Text or structured payload." },
                metadata: { type: "object", additionalProperties: true, description: "Opaque extra data." }
            },
            required: ["to", "content"]
        },
        annotations: {
            title: "Send Message",
            readOnlyHint: false, // Modifies state
            openWorldHint: false, // Does not interact with the real world
        },
    }],
    [McpToolName.ResourceManage, {
        name: McpToolName.ResourceManage,
        description: "Find, add, update or delete a resource depending on 'op'.",
        inputSchema: {
            oneOf: [
                {
                    title: "Find Resource",
                    type: "object",
                    properties: {
                        op: { const: "find" },
                        resource_type: { type: "string", oneOf: resourceVariantSchemaItems },
                        id: { type: "string" },
                        query: { type: "string" },
                        filters: { type: "object", additionalProperties: true },
                    },
                    required: ["op", "resource_type"],
                    additionalProperties: false,
                },
                {
                    title: "Add Resource",
                    type: "object",
                    properties: {
                        op: { const: "add" },
                        resource_type: { type: "string", oneOf: resourceVariantSchemaItems },
                        attributes: { type: "object", additionalProperties: true },
                    },
                    required: ["op", "resource_type", "attributes"],
                    additionalProperties: false,
                },
                {
                    title: "Update Resource",
                    type: "object",
                    properties: {
                        op: { const: "update" },
                        resource_type: { type: "string", oneOf: resourceVariantSchemaItems },
                        id: { type: "string" },
                        attributes: { type: "object", additionalProperties: true, minProperties: 1 },
                    },
                    required: ["op", "resource_type", "id", "attributes"],
                    additionalProperties: false,
                },
                {
                    title: "Delete Resource",
                    type: "object",
                    properties: {
                        op: { const: "delete" },
                        resource_type: { type: "string", oneOf: resourceVariantSchemaItems },
                        id: { type: "string" },
                    },
                    required: ["op", "resource_type", "id"],
                    additionalProperties: false,
                }
            ]
        },
        annotations: {
            title: "Resource Manage",
            readOnlyHint: false,
            openWorldHint: false,
        },
    }],
    [McpToolName.RunRoutine, {
        name: McpToolName.RunRoutine,
        description: "Start or manage a routine run. Supports scheduling when mode = async & op = start.",
        inputSchema: {
            oneOf: [
                {
                    title: "Start Routine Run",
                    type: "object",
                    properties: {
                        op: { const: "start" },
                        routine_id: { type: "string" },
                        mode: { type: "string", enum: ["sync", "async"], default: "sync" },
                        schedule: {
                            type: "object",
                            properties: {
                                at: { type: "string", format: "date-time" },
                                cron: { type: "string" },
                                timezone: { type: "string" },
                            },
                            additionalProperties: false,
                        },
                        time_sensitive: { type: "boolean", default: false },
                        bot_config: {
                            type: "object",
                            properties: {
                                starting_prompt: { type: "string" },
                                responding_bot_id: { type: "string" },
                            },
                            additionalProperties: false,
                        },
                    },
                    required: ["op", "routine_id"],
                    additionalProperties: false,
                },
                {
                    title: "Manage Existing Run",
                    type: "object",
                    properties: {
                        op: { type: "string", enum: ["pause", "resume", "cancel", "status"] },
                        run_id: { type: "string" },
                    },
                    required: ["op", "run_id"],
                    additionalProperties: false,
                }
            ]
        },
        annotations: {
            title: "Run Routine",
            readOnlyHint: false, // Modifies state
            openWorldHint: true, // May interact with the real world
        },
    }],
    [McpToolName.StartSession, {
        name: McpToolName.StartSession,
        description: "Create a new session with selected bots or an existing team. Supports governance policy & usage limits.",
        inputSchema: {
            type: "object",
            properties: {
                bot_ids: {
                    type: "array",
                    items: { type: "string" },
                    minItems: 1,
                    description: "Explicit list of bot IDs to include",
                },
                team_id: { type: "string", description: "Team ID (exclusive with bot_ids)" },
                purpose: { type: "string", description: "High‑level goal for this session" },
                context: { oneOf: [{ type: "string" }, { type: "object" }], description: "Initial shared context" },
                policy: {
                    type: "object",
                    description: "Governance/ACL rules",
                    properties: {
                        type: { type: "string", enum: ["open", "restricted", "private"] },
                        acl: {
                            type: "array",
                            items: { type: "string" },
                            description: "Allowed agent IDs when type=restricted/private",
                        },
                    },
                    additionalProperties: false,
                },
                limits: {
                    type: "object",
                    properties: {
                        max_messages: { type: "integer", minimum: 1 },
                        max_duration_minutes: { type: "integer", minimum: 1 },
                        max_credits: { type: "integer", minimum: 1 },
                        expires_at: { type: "string", format: "date-time" },
                    },
                    additionalProperties: false,
                },
            },
            oneOf: [
                { required: ["bot_ids"] },
                { required: ["team_id"] },
            ],
            additionalProperties: false,
        },
        annotations: {
            title: "Start Session",
            readOnlyHint: false, // Modifies state
            openWorldHint: true // Interacts with the real world
        },
    }],
]);

export interface CheckSessionParams {
    // No parameters
}

/**
 * Parameters for mutating an existing session’s metadata, lifecycle, or routing.
 */
export interface UpdateSessionParams {
    /** Session being updated. */
    session_id: string;

    /* ---------- high-level metadata ---------- */

    /** Change the human-readable goal / purpose. */
    purpose?: string;

    /** Replace or extend the shared context blob. */
    context?: string | JSONValue;

    /** Patch governance rules (same shape as in `StartSessionParams`). */
    policy?: {
        type?: "open" | "restricted" | "private";
        acl?: string[];          // full replacement of the ACL list
    };

    /** Adjust caps or expiry. Any field omitted leaves the old value intact. */
    limits?: {
        max_messages?: number;
        max_duration_minutes?: number;
        max_credits?: number;
        expires_at?: string;     // ISO-8601 instant
    };

    /* ---------- event-routing table ---------- */

    /** Add / replace routes that forward session events to teammates. */
    event_routes?: {
        /** Topic pattern (literal or glob, e.g. `"build/*"` or `"sensor/#"`). */
        topic: string;
        /** Who should receive matching events. */
        recipients: "all" | string[];
        /** Delivery style. `"buffer"` queues until the bot’s next turn. */
        mode?: "push" | "buffer" | "ignore";
        /** Auto-expire after N seconds. 0 or undefined = no TTL. */
        ttl_seconds?: number;
    }[];

    /** IDs (or `"*"` wildcard) of previously created routes to delete. */
    remove_routes?: string[];

    /* ---------- lifecycle state ---------- */

    /** Soft transition of the session’s state. */
    state?: "active" | "paused" | "ended";
    /** Reason for the state change; recorded in the audit log. */
    state_reason?: string;
}

/**
 * Parameters for gracefully or force-ending a session.
 */
export interface EndSessionParams {
    /**
     * How to end the session:
     *  • "graceful" (default) – allow bots to finish their current turn  
     *  • "force"             – cancel immediately, may interrupt routines
     */
    mode?: "graceful" | "force";
    /** Reason for ending; sent to participants & saved to the audit trail. */
    reason?: string;
    /**
     * When true (default) the server archives chat history & artifacts.
     * When false transient data may be discarded permanently.
     */
    archive?: boolean;
}

export const sessionToolDefinitions: Map<McpSessionToolName, Tool> = new Map([
    [
        McpSessionToolName.CheckSession,
        {
            name: McpSessionToolName.CheckSession,
            description: "Return the current metadata, limits, routing rules, and lifecycle state for this session.",
            inputSchema: {
                type: "object",
                properties: {},
                required: [],
            },
            annotations: {
                title: "Check Session",
                readOnlyHint: true,                 // pure read
                openWorldHint: false
            }
        }
    ],
    [
        McpSessionToolName.UpdateSession,
        {
            name: McpSessionToolName.UpdateSession,
            description: "Mutate session metadata, limits, event-routing rules, lifecycle state, or append to the log.",
            inputSchema: {
                type: "object",
                properties: {
                    session_id: { type: "string" },

                    /* metadata ------------------------------------------------------ */
                    purpose: { type: "string" },
                    context: {
                        oneOf: [
                            { type: "string" },
                            { type: "object", additionalProperties: true }
                        ]
                    },
                    policy: {
                        type: "object",
                        properties: {
                            type: { type: "string", enum: ["open", "restricted", "private"] },
                            acl: {
                                type: "array",
                                items: { type: "string" }
                            }
                        },
                        additionalProperties: false
                    },
                    limits: {
                        type: "object",
                        properties: {
                            max_messages: { type: "integer", minimum: 1 },
                            max_duration_minutes: { type: "integer", minimum: 1 },
                            max_credits: { type: "integer", minimum: 1 },
                            expires_at: { type: "string", format: "date-time" }
                        },
                        additionalProperties: false
                    },

                    /* event routing -------------------------------------------------- */
                    event_routes: {
                        type: "array",
                        items: {
                            type: "object",
                            properties: {
                                topic: { type: "string" },
                                recipients: {
                                    oneOf: [
                                        { const: "all" },
                                        {
                                            type: "array",
                                            items: { type: "string" },
                                            minItems: 1
                                        }
                                    ]
                                },
                                mode: { type: "string", enum: ["push", "buffer", "ignore"] },
                                ttl_seconds: { type: "integer", minimum: 0 }
                            },
                            required: ["topic", "recipients"],
                            additionalProperties: false
                        }
                    },
                    remove_routes: {
                        type: "array",
                        items: { type: "string" }
                    },

                    /* lifecycle ------------------------------------------------------ */
                    state: { type: "string", enum: ["active", "paused", "ended"] },
                    state_reason: { type: "string" },
                },
                required: ["session_id"],
                additionalProperties: false
            },
            annotations: {
                title: "Update Session",
                readOnlyHint: false,                // mutates state
                openWorldHint: false
            }
        }
    ],
    [
        McpSessionToolName.EndSession,
        {
            name: McpSessionToolName.EndSession,
            description: "Gracefully or forcibly terminate a session and (optionally) archive its artifacts.",
            inputSchema: {
                type: "object",
                properties: {
                    mode: { type: "string", enum: ["graceful", "force"], default: "graceful" },
                    reason: { type: "string" },
                    archive: { type: "boolean", default: true }
                },
                required: [],
                additionalProperties: false
            },
            annotations: {
                title: "End Session",
                readOnlyHint: false,
                openWorldHint: false
            }
        }
    ]
]);

/**
 * ---------------------------------------------------------------------------
 *  TOOL REGISTRY – *single source of truth* for the MCP root‑level interface
 * ---------------------------------------------------------------------------
 */
export class ToolRegistry {
    private toolbox: Map<McpToolName, ToolHandler> = new Map();
    private logger: Logger;

    constructor(logger: Logger) {
        this.logger = logger;
        this.registerTools();
    }

    /**
     * Register built-in tools
     */
    private registerTools(): void {
        // this.registerBuiltInTool(McpToolName.AddResource, addResource);
        // this.registerBuiltInTool(McpToolName.DeleteResource, deleteResource);
        // this.registerBuiltInTool(McpToolName.FindResources, findResources);
        // this.registerBuiltInTool(McpToolName.RunRoutine, runRoutine);
        // this.registerBuiltInTool(McpToolName.UpdateResource, updateResource);
    }

    /**
     * Register a new built-in tool (definition and handler)
     * @param name Tool name
     * @param handler Tool handler function
     */
    registerBuiltInTool(name: McpToolName, handler: ToolHandler): void {
        this.logger.info(`Registering built-in tool: ${name}`);
        this.toolbox.set(name, handler);
    }

    /**
     * Get built-in tool definitions
     * @returns Array of built-in tool definitions
     */
    getBuiltInDefinitions(): Tool[] {
        return Array.from(builtInToolDefinitions.values());
    }

    /**
     * Get the handler function for a specific tool.
     * @param name The name of the tool.
     * @returns The handler function or undefined if not found.
     */
    getBuiltInTool(name: McpToolName): ToolHandler | undefined {
        return this.toolbox.get(name);
    }

    /**
     * Execute a tool by name
     * @param name Tool name
     * @param args Tool arguments
     * @returns Tool execution response
     */
    async execute(name: McpToolName, args: any): Promise<ToolResponse> {
        this.logger.info(`Executing tool: ${name}`);

        const handler = this.toolbox.get(name);
        if (!handler) {
            this.logger.error(`Tool handler not found or not registered: ${name}`);
            return {
                isError: true,
                content: [
                    {
                        type: "text",
                        text: `Error: Tool handler for '${name}' not found. It might be a dynamic tool whose handler hasn't been registered.`,
                    },
                ],
            };
        }

        try {
            const result = await handler(args, this.logger);
            return result;
        } catch (error) {
            this.logger.error(`Error executing tool ${name}:`, error);
            return {
                isError: true,
                content: [
                    {
                        type: "text",
                        text: `Error executing tool '${name}': ${(error as Error).message}`,
                    },
                ],
            };
        }
    }
} 
