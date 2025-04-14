import { Logger, Tool, ToolResponse } from '../types.js';
import { calculateSum } from './calculate-sum.js';
import { fetchResource } from './fetch-resource.js';

type ToolHandler = (args: any, logger: Logger) => ToolResponse;

/**
 * Registry for managing MCP tools
 */
export class ToolRegistry {
    private tools: Map<string, ToolHandler> = new Map();
    private toolDefinitions: Tool[] = [];
    private logger: Logger;

    constructor(logger: Logger) {
        this.logger = logger;
        this.registerBuiltInTools();
    }

    /**
     * Register built-in tools
     */
    private registerBuiltInTools(): void {
        this.register('calculate_sum', calculateSum, {
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
            }
        });

        this.register('fetch_resource', fetchResource, {
            name: 'fetch_resource',
            description: 'Fetch a resource by name or search string',
            inputSchema: {
                type: 'object',
                properties: {
                    query: {
                        type: 'string',
                        description: 'Resource name or search string to find the resource'
                    }
                },
                required: ['query']
            },
            annotations: {
                title: 'Fetch Resource',
                readOnlyHint: true,
                openWorldHint: false
            }
        });
    }

    /**
     * Register a new tool
     * @param name Tool name
     * @param handler Tool handler function
     * @param definition Tool definition
     */
    register(name: string, handler: ToolHandler, definition: Tool): void {
        this.logger.info(`Registering tool: ${name}`);
        this.tools.set(name, handler);
        this.toolDefinitions.push(definition);
    }

    /**
     * Get all tool definitions
     * @returns Array of tool definitions
     */
    getDefinitions(): Tool[] {
        return this.toolDefinitions;
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
            this.logger.error(`Tool not implemented: ${name}`);
            return {
                isError: true,
                content: [
                    {
                        type: 'text',
                        text: `Error: Tool '${name}' not implemented`
                    }
                ]
            };
        }

        try {
            return handler(args, this.logger);
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