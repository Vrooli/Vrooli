import { addResource, calculateSum, fetchResource, fetchTool } from '../tools.js';
import { Logger, Tool, ToolResponse } from '../types.js';

type ToolHandler = (args: any, logger: Logger) => ToolResponse;

/**
 * Registry for managing MCP tools
 */
export class ToolRegistry {
    private tools: Map<string, ToolHandler> = new Map();
    private builtInDefinitions: Tool[] = [];
    private logger: Logger;

    constructor(logger: Logger) {
        this.logger = logger;
        this.registerBuiltInTools();
        this.registerKnownHandlers();
    }

    /**
     * Register built-in tools
     */
    private registerBuiltInTools(): void {
        this.registerBuiltInTool('fetch_resource', fetchResource, {
            name: 'fetch_resource',
            description: 'Fetch a resource (note or reminder) by its exact name and type.',
            inputSchema: {
                type: 'object',
                properties: {
                    name: {
                        type: 'string',
                        description: 'The exact name of the resource to fetch.'
                    },
                    resource_type: {
                        type: 'string',
                        description: 'The type of resource to fetch.',
                        enum: ['notes', 'reminders']
                    }
                },
                required: ['name', 'resource_type']
            },
            annotations: {
                title: 'Fetch Resource',
                readOnlyHint: true,
                openWorldHint: false
            }
        });

        this.registerBuiltInTool('add_resource', addResource, {
            name: 'add_resource',
            description: 'Add or update a resource (note or reminder).',
            inputSchema: {
                type: 'object',
                properties: {
                    name: {
                        type: 'string',
                        description: 'The name of the resource to add or update.'
                    },
                    resource_type: {
                        type: 'string',
                        description: 'The type of the resource.',
                        enum: ['notes', 'reminders']
                    },
                    content: {
                        type: 'string',
                        description: 'The content of the resource.'
                    }
                },
                required: ['name', 'resource_type', 'content']
            },
            annotations: {
                title: 'Add Resource',
                readOnlyHint: false,
                openWorldHint: false
            }
        });

        this.registerBuiltInTool('fetch_tool', fetchTool, {
            name: 'fetch_tool',
            description: 'Fetch/search for available MCP tools',
            inputSchema: {
                type: 'object',
                properties: {
                    query: {
                        type: 'string',
                        description: 'Search string for MCP tools'
                    }
                },
                required: ['query']
            },
            annotations: {
                title: 'Fetch Tool',
                readOnlyHint: true,
                openWorldHint: false
            }
        });
    }

    /**
     * Register handlers for known tools that might be fetched dynamically later.
     * This ensures the execute function can find the handler.
     */
    private registerKnownHandlers(): void {
        this.logger.info(`Registering known handler: calculate_sum`);
        this.tools.set('calculate_sum', calculateSum);
    }

    /**
     * Register a new built-in tool (definition and handler)
     * @param name Tool name
     * @param handler Tool handler function
     * @param definition Tool definition
     */
    registerBuiltInTool(name: string, handler: ToolHandler, definition: Tool): void {
        this.logger.info(`Registering built-in tool: ${name}`);
        this.tools.set(name, handler);
        this.builtInDefinitions.push(definition);
    }

    /**
     * Get built-in tool definitions
     * @returns Array of built-in tool definitions
     */
    getBuiltInDefinitions(): Tool[] {
        return this.builtInDefinitions;
    }

    /**
     * Get dynamic tool definitions (simulates fetching from a source like a DB)
     * @returns Promise resolving to an array of dynamic tool definitions
     */
    async getDynamicDefinitions(): Promise<Tool[]> {
        this.logger.info('Simulating fetch of dynamic tool definitions...');
        await new Promise(resolve => setTimeout(resolve, 50));

        return [
            {
                name: 'calculate_sum',
                description: 'Add two numbers together',
                inputSchema: {
                    type: 'object',
                    properties: {
                        a: { type: 'number' },
                        b: { type: 'number' }
                    },
                    required: ['a', 'b']
                },
                annotations: {
                    title: 'Calculate Sum',
                    readOnlyHint: true,
                    openWorldHint: false
                },
            }
        ];
    }

    /**
     * Get the handler function for a specific tool.
     * @param name The name of the tool.
     * @returns The handler function or undefined if not found.
     */
    getHandler(name: string): ToolHandler | undefined {
        return this.tools.get(name);
    }

    /**
     * Execute a tool by name
     * @param name Tool name
     * @param args Tool arguments
     * @returns Tool execution response
     */
    async execute(name: string, args: any): Promise<ToolResponse> {
        this.logger.info(`Executing tool: ${name}`);

        const handler = this.tools.get(name);
        if (!handler) {
            this.logger.error(`Tool handler not found or not registered: ${name}`);
            return {
                isError: true,
                content: [
                    {
                        type: 'text',
                        text: `Error: Tool handler for '${name}' not found. It might be a dynamic tool whose handler hasn't been registered.`
                    }
                ]
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
                        type: 'text',
                        text: `Error executing tool '${name}': ${(error as Error).message}`
                    }
                ]
            };
        }
    }
} 