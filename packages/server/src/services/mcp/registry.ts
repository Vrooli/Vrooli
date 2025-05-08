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
    /** Run a routine (dynamic tool) inline (synchronous, no run object created) or as a job (asynchronous, run object created) */
    RunRoutine = "run_routine",
    /** Starts a session with a bot or team of bots */
    StartSession = "start_session",
    /** Find notes, reminders, routines, users, etc. */
    FindResources = "find_resources",
    /** Create a note, reminder, routine, user, etc. */
    AddResource = "add_resource",
    /** Update a note, reminder, routine, user, etc. */
    UpdateResource = "update_resource",
    /** Delete a note, reminder, routine, user, etc. */
    DeleteResource = "delete_resource",
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
    [ResourceType.Project],
    // Other resources
    [ResourceType.Note],
    ["ExternalData", "KV bucket, table, S3 object, vector index, etc."]
]

/**
 * Parameters for defining how to use a tool.
 * 
 * Useful for tools like AddResource, which can have multiple variants 
 * whose parameters are complex and would not be useful to provide until 
 * the user decided to add a resource od the given type.
 */
export interface DefineToolParams {
    toolName: McpToolName.AddResource | McpToolName.FindResources | McpToolName.UpdateResource;
    variant?: string;
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
 * Parameters for finding resources (notes or routines).
 */
export interface FindResourcesParams {
    /** Optional resource ID to fetch */
    id?: string;
    /** Optional search query to match resource names or content */
    query?: string;
    /** Type of resource: e.g. "Note" or "Routine" */
    resource_type: string;
    /** Optional resource-specific filter parameters. Use DefineTool for schema. */
    filters?: Record<string, any>;
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
 * Parameters for adding or updating a resource.
 */
export interface AddResourceParams {
    /** The type of resource to add */
    resource_type: string;
    /** Resource-specific attributes */
    attributes: Record<string, any>;
}

/**
 * Parameters for updating a resource.
 */
export interface UpdateResourceParams {
    /** The public ID of the resource to update. */
    id: string;
    /** The type of resource being updated. This helps in applying the correct schema for attributes. */
    resource_type: string;
    /** Resource-specific attributes to update. Only provided fields will be updated. */
    attributes: Record<string, any>;
}

/**
 * Parameters for deleting a resource.
 */
export interface DeleteResourceParams {
    /** The public ID of the resource to delete. */
    id: string;
    /** The type of resource to delete. */
    resource_type: string;
}

// Create a structure for JSON Schema oneOf with descriptions
const resourceVariantSchemaItems = resourceTypes.map(item => ({
    const: String(item[0]),
    description: item[1] || String(item[0]) // Use provided description or the name itself
}));

/**
 * How the user of the MCP sees what tools are available, what they do, and how to use them.
 */
const toolDefinitions: Map<McpToolName, Tool> = new Map([
    [McpToolName.DefineTool, {
        name: McpToolName.DefineTool,
        description: "Get the detailed input schema for adding, updating, or finding a specific type of resource variant.",
        inputSchema: {
            type: "object",
            properties: {
                toolName: {
                    type: "string",
                    description: "The core tool (AddResource, UpdateResource, FindResources) for which to get a detailed schema.",
                    enum: [McpToolName.FindResources, McpToolName.AddResource, McpToolName.UpdateResource],
                },
                variant: {
                    type: "string",
                    description: "The specific type/variant of resource (e.g., Note, RoutineInternalAction, RoutineApi).",
                    oneOf: resourceVariantSchemaItems,
                },
            },
            required: ["toolName", "variant"],
        },
        annotations: {
            title: "Define Tool Parameters",
            readOnlyHint: true, // Does not modify state
            openWorldHint: false, // Does not interact with the real world
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
    [McpToolName.RunRoutine, {
        name: McpToolName.RunRoutine,
        description: "Start or manage a routine run. Supports scheduling when mode = async & op = start.",
        inputSchema: {
            type: "object",
            properties: {
                op: { type: "string", enum: ["start", "pause", "resume", "cancel", "status"], default: "start" },
                run_id: { type: "string", description: "Existing run identifier (required for non-start ops)." },
                routine_id: { type: "string", description: "Routine resource id (required when op = start)." },
                mode: { type: "string", enum: ["sync", "async"], default: "sync" },
                schedule: {
                    type: "object",
                    description: "When mode=async & op=start: ISO date or cron pattern.",
                    properties: {
                        at: { type: "string", format: "date-time" },
                        cron: { type: "string" },
                        timezone: { type: "string" }
                    },
                    additionalProperties: false
                },
                time_sensitive: { type: "boolean", default: false },
                bot_config: {
                    type: "object",
                    properties: {
                        starting_prompt: { type: "string" },
                        responding_bot_id: { type: "string" }
                    },
                    additionalProperties: false
                }
            },
            // Validation rule (enforced in handler):
            //   – if op === "start"  ⇒ routine_id required
            //   – else                ⇒ run_id required
        },
        annotations: {
            title: "Run Routine",
            readOnlyHint: false, // Modifies state
            openWorldHint: true, // May interact with the real world
        },
    }],
    [McpToolName.FindResources, {
        name: McpToolName.FindResources,
        description: "Find or search for resources by ID, query, and type.",
        inputSchema: {
            type: "object",
            properties: {
                id: {
                    type: "string",
                    description: "The public ID of the resource to fetch, if not using a name.",
                },
                query: {
                    type: "string",
                    description: "The name or search query to find resources by, if not using an ID.",
                },
                resource_type: {
                    type: "string",
                    description: "The type of resource to fetch.",
                    oneOf: resourceVariantSchemaItems,
                },
                filters: {
                    type: "object",
                    description: "Optional resource-specific filter parameters. Use the 'DefineTool' with toolName 'FindResources' and the desired 'variant' to get the specific schema for these filters.",
                    additionalProperties: true,
                },
            },
            required: ["resource_type"],
        },
        annotations: {
            title: "Find Resource",
            readOnlyHint: true, // Does not modify state
            openWorldHint: false, // Does not interact with the real world
        },
    }],
    [McpToolName.AddResource, {
        name: McpToolName.AddResource,
        description: "Add a new resource. Use the 'DefineTool' with toolName 'AddResource' and the desired 'variant' to get the specific schema for the 'attributes' object based on the resource type.",
        inputSchema: {
            type: "object",
            properties: {
                resource_type: {
                    type: "string",
                    description: "The type of resource to add (e.g., Note, RoutineInternalAction). This determines the expected structure of the 'attributes' object.",
                    oneOf: resourceVariantSchemaItems,
                },
                attributes: {
                    type: "object",
                    description: "Resource-specific attributes. The exact schema for this object depends on the 'resource_type'. Use 'DefineTool' to get this schema.",
                    additionalProperties: true, // Allows any properties; DefineTool will provide the specific schema.
                },
            },
            required: ["resource_type", "attributes"],
        },
        annotations: {
            title: "Add Resource",
            readOnlyHint: false, // Modifies state
            openWorldHint: false, // Does not interact with the real world
        },
    }],
    [McpToolName.UpdateResource, {
        name: McpToolName.UpdateResource,
        description: "Update an existing resource. Use the 'DefineTool' with toolName 'UpdateResource' and the desired 'variant' to get the specific schema for the 'attributes' object based on the resource type. Only provided attributes will be updated.",
        inputSchema: {
            type: "object",
            properties: {
                id: {
                    type: "string",
                    description: "The public ID of the resource to update.",
                },
                resource_type: {
                    type: "string",
                    description: "The type of resource being updated (e.g., Note, RoutineInternalAction). This determines the expected structure of the 'attributes' object for the update.",
                    oneOf: resourceVariantSchemaItems, // Ensures it's one of the known types
                },
                attributes: {
                    type: "object",
                    description: "Resource-specific attributes to update. The exact schema for this object (and which fields are updatable) depends on the 'resource_type'. Use 'DefineTool' to get this schema. Only fields present in this object will be considered for update.",
                    additionalProperties: true, // Allows any properties; DefineTool will provide the specific schema.
                    minProperties: 1, // Expect at least one attribute to update
                },
            },
            required: ["id", "resource_type", "attributes"],
        },
        annotations: {
            title: "Update Resource",
            readOnlyHint: false, // Modifies state
            openWorldHint: false, // Does not interact with the real world
        },
    }],
    [McpToolName.DeleteResource, {
        name: McpToolName.DeleteResource,
        description: "Delete a resource by its public ID and type.",
        inputSchema: {
            type: "object",
            properties: {
                id: {
                    type: "string",
                    description: "The public ID of the resource to delete.",
                },
                resource_type: {
                    type: "string",
                    description: "The type of the resource to delete (e.g., Note, RoutineInternalAction). This helps ensure the correct resource is targeted.",
                    oneOf: resourceVariantSchemaItems, // Ensures it's one of the known types
                },
            },
            required: ["id", "resource_type"],
        },
        annotations: {
            title: "Delete Resource",
            readOnlyHint: false, // Modifies state
            openWorldHint: false, // Does not interact with the real world
        },
    }],
]);

/**
 * ---------------------------------------------------------------------------
 *  TOOL REGISTRY – *single source of truth* for the MCP root‑level interface
 * ---------------------------------------------------------------------------
 * ▪   Follows the “micro‑kernel” philosophy:
 *     – minimal verb set (communicate · discover/CRUD · act · introspect)
 *     – all long‑running work handled via a uniform Run lifecycle
 * ▪   Adds basic *scheduling* for asynchronous runs (ISO timestamp or cron)
 * ▪   Message verb doubles as a lightweight pub‑sub / event bus
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
        return Array.from(toolDefinitions.values());
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
