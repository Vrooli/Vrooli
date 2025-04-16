import { addResource, deleteResource, findResources, runRoutine, updateResource } from "./tools.js";
import { Logger, Tool, ToolResponse } from "./types.js";

type ToolHandler = (args: any, logger: Logger) => Promise<ToolResponse>;

export enum McpToolName {
    /** Create a note, reminder, routine, user, etc. */
    AddResource = "add_resource",
    /** Delete a note, reminder, routine, user, etc. */
    DeleteResources = "delete_resource",
    /** Find notes, reminders, routines, users, etc. */
    FindResources = "find_resources",
    /** Run a routine (dynamic tool) */
    RunRoutine = "run_routine",
    /** Update a note, reminder, routine, user, etc. */
    UpdateResource = "update_resource",
}

export enum McpRoutineToolName {
    /** Start a routine */
    StartRoutine = "start_routine",
    /** Stop a routine */
    StopRoutine = "stop_routine",
}

const toolDefinitions: Map<McpToolName, Tool> = new Map([
    [McpToolName.AddResource, {
        name: McpToolName.AddResource,
        description: "Add or update a resource",
        inputSchema: {
            type: "object",
            properties: {
                name: {
                    type: "string",
                    description: "The name of the resource to add or update.",
                },
                resource_type: {
                    type: "string",
                    description: "The type of the resource.",
                    enum: ["Note", "Routine"],
                },
                content: {
                    type: "string",
                    description: "The content of the resource.",
                },
            },
            required: ["name", "resource_type", "content"],
        },
        annotations: {
            title: "Add Resource",
            readOnlyHint: false,
            openWorldHint: false,
        },
    }],
    [McpToolName.DeleteResources, {
        name: McpToolName.DeleteResources,
        description: "Delete a resource",
        inputSchema: {
            type: "object",
            properties: {
                name: {
                    type: "string",
                    description: "The ID of the resource to delete.",
                },
                resource_type: {
                    type: "string",
                    description: "The type of the resource to delete.",
                    enum: ["Note", "Routine"],
                },
            },
            required: ["name", "resource_type"],
        },
        annotations: {
            title: "Delete Resource",
            readOnlyHint: false,
            openWorldHint: false,
        },
    }],
    [McpToolName.FindResources, {
        name: McpToolName.FindResources,
        description: "Search for a resource by its exact name and type.",
        inputSchema: {
            type: "object",
            properties: {
                id: {
                    type: "string",
                    description: "The ID of the resource to fetch, if not using a name.",
                },
                query: {
                    type: "string",
                    description: "The name or search query to find resources by, if not using an ID.",
                },
                resource_type: {
                    type: "string",
                    description: "The type of resource to fetch.",
                    enum: ["Note", "Routine"],
                },
            },
            required: ["resource_type"],
        },
        annotations: {
            title: "Find Resource",
            readOnlyHint: true,
            openWorldHint: false,
        },
    }],
    [McpToolName.RunRoutine, {
        name: McpToolName.RunRoutine,
        description: "Run a routine",
        inputSchema: {
            type: "object",
            properties: {
                name: {
                    type: "string",
                    description: "The name of the routine to run.",
                },
            },
            required: ["name"],
        },
        annotations: {
            title: "Run Routine",
            readOnlyHint: false,
            openWorldHint: false,
        },
    }],
    [McpToolName.UpdateResource, {
        name: McpToolName.UpdateResource,
        description: "Update a resource",
        inputSchema: {
            type: "object",
            properties: {
                name: {
                    type: "string",
                    description: "The name of the resource to update.",
                },

            },
            required: ["name"],
        },
        annotations: {
            title: "Update Resource",
            readOnlyHint: false,
            openWorldHint: false,
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
        this.registerBuiltInTool(McpToolName.AddResource, addResource);
        this.registerBuiltInTool(McpToolName.DeleteResources, deleteResource);
        this.registerBuiltInTool(McpToolName.FindResources, findResources);
        this.registerBuiltInTool(McpToolName.RunRoutine, runRoutine);
        this.registerBuiltInTool(McpToolName.UpdateResource, updateResource);
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
