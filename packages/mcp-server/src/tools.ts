import { Logger, Tool, ToolResponse } from './types.js';

/**
 * Defines the available MCP tools.
 * @param logger - The logger instance for logging operations
 * @returns Array of tool definitions
 */
export function getToolDefinitions(logger: Logger): Tool[] {
    logger.info("Retrieving tool definitions...");
    return [
        {
            name: "calculate_sum",
            description: "Add two numbers together",
            inputSchema: {
                type: "object",
                properties: {
                    a: { type: "number" },
                    b: { type: "number" }
                },
                required: ["a", "b"]
            },
            annotations: {
                title: "Calculate Sum",
                readOnlyHint: true,
                openWorldHint: false
            }
        },
        {
            name: "fetch_resource",
            description: "Fetch a resource by name or search string",
            inputSchema: {
                type: "object",
                properties: {
                    query: {
                        type: "string",
                        description: "Resource name or search string to find the resource"
                    }
                },
                required: ["query"]
            },
            annotations: {
                title: "Fetch Resource",
                readOnlyHint: true,
                openWorldHint: false
            }
        }
    ];
}

/**
 * Implementation of the calculate_sum tool.
 * @param args - The arguments for the tool
 * @param logger - The logger instance for logging operations
 * @returns The result of the calculation
 */
export function calculateSum(args: { a: number, b: number }, logger: Logger): ToolResponse {
    const { a, b } = args;
    const sum = a + b;

    logger.info(`Calculated sum: ${sum}`);
    return {
        content: [
            {
                type: "text",
                text: String(sum)
            }
        ]
    };
}

/**
 * Implementation of the fetch_resource tool.
 * @param args - The arguments for the tool
 * @param logger - The logger instance for logging operations
 * @returns Resource content
 */
export function fetchResource(args: { query: string }, logger: Logger): ToolResponse {
    const { query } = args;

    logger.info(`Fetching resource with query: ${query}`);

    // Simple resource matching logic
    let resourceContent = "";
    let resourceName = "";

    // Mock resources
    if (query.toLowerCase().includes("greeting") || query.toLowerCase().includes("hello")) {
        resourceName = "Greeting";
        resourceContent = "Hello, world!";
    } else if (query.toLowerCase().includes("readme") || query.toLowerCase().includes("documentation")) {
        resourceName = "README Document";
        resourceContent = "# Project README\n\nWelcome to the Vrooli MCP Server project.\n\nThis is a mock README document served through MCP.";
    } else if (query.toLowerCase().includes("status") || query.toLowerCase().includes("system")) {
        resourceName = "System Status";
        const statusData = {
            status: "operational",
            uptime: "3 days, 4 hours",
            memory: {
                used: "1.2 GB",
                available: "4.8 GB"
            }
        };
        resourceContent = JSON.stringify(statusData, null, 2);
    } else {
        resourceName = "Resource Not Found";
        resourceContent = `No resource found matching query: "${query}"`;
    }

    logger.info(`Returning resource: ${resourceName}`);
    return {
        content: [
            {
                type: "text",
                text: `Resource: ${resourceName}\n\n${resourceContent}`
            }
        ]
    };
} 