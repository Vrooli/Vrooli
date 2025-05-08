import { ModelType, ResourceSubType, ResourceType } from "@local/shared";
import { Logger, Tool, ToolResponse } from "./types.js";

type ToolHandler = (args: any, logger: Logger) => Promise<ToolResponse>;

/**
 * Available root-level tools, ordered by (likely) frequency of use.
 */
export enum McpToolName {
    /** Get detailed parameters for other tools based on a variant */
    DefineTool = "define_tool",
    /** Find notes, reminders, routines, users, etc. */
    FindResources = "find_resources",
    /** Run a routine (dynamic tool) */
    RunRoutine = "run_routine",
    /** Create a note, reminder, routine, user, etc. */
    AddResource = "add_resource",
    /** Update a note, reminder, routine, user, etc. */
    UpdateResource = "update_resource",
    /** Delete a note, reminder, routine, user, etc. */
    DeleteResource = "delete_resource",
}

/**
 * Available routine-level tools, ordered by (likely) frequency of use.
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
    [ResourceSubType.RoutineAction, "Perform an action within Vrooli (e.g. 'create a new note')"],
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
    [ResourceSubType.CodeDataConverter, "Sandboxed JavaScript code for simple data conversions"],
    [ResourceSubType.CodeSmartContract, "Smart contract"],
    [ResourceType.Api],
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

/**
 * Parameters for running a routine.
 */
export interface RunRoutineParams {
    /** The ID of an existing run to resume. If provided, 'routine_id' should usually not be set. */
    run_id?: string;
    /** The resource ID of the routine to start for a new run. If provided, 'run_id' should usually not be set. */
    routine_id?: string;
    /** Indicates if the run is time-sensitive. Defaults to false. */
    time_sensitive?: boolean;
    /** Optional configuration for the bot executing the routine. */
    bot_config?: {
        /** An optional initial prompt for the bot. */
        starting_prompt?: string;
        /** The ID of a specific bot to respond/execute. */
        responding_bot_id?: string;
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
                    description: "The specific type/variant of resource (e.g., Note, RoutineAction, Api).",
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
    [McpToolName.RunRoutine, {
        name: McpToolName.RunRoutine,
        description: "Starts a new routine run or resumes an existing one.",
        inputSchema: {
            type: "object",
            properties: {
                run_id: {
                    type: "string",
                    description: "The ID of an existing run to resume. If provided, 'routine_id' should usually not be set.",
                },
                routine_id: {
                    type: "string",
                    description: "The resource ID of the routine to start for a new run. If provided, 'run_id' should usually not be set.",
                },
                time_sensitive: {
                    type: "boolean",
                    description: "Indicates if the run is time-sensitive. Defaults to false.",
                    default: false,
                },
                bot_config: {
                    type: "object",
                    description: "Optional configuration for the bot executing the routine.",
                    properties: {
                        starting_prompt: {
                            type: "string",
                            description: "An optional initial prompt for the bot.",
                        },
                        responding_bot_id: {
                            type: "string",
                            description: "The ID of a specific bot to respond/execute.",
                        },
                    },
                    additionalProperties: false,
                },
            },
            // 'required' could be dynamic based on whether run_id or routine_id is provided.
            // For now, the handler will need to validate that at least one is present.
            // Example: required: ["routine_id"] if it's always new, or handle complex logic in the tool handler.
        },
        annotations: {
            title: "Run Routine",
            readOnlyHint: false, // Modifies state
            openWorldHint: true, // May interact with the real world
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
                    description: "The type of resource to add (e.g., Note, RoutineAction). This determines the expected structure of the 'attributes' object.",
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
                    description: "The type of resource being updated (e.g., Note, RoutineAction). This determines the expected structure of the 'attributes' object for the update.",
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
                    description: "The type of the resource to delete (e.g., Note, RoutineAction). This helps ensure the correct resource is targeted.",
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
 * Registry for managing MCP tools.
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
