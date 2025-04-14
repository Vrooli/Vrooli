import { Logger, ToolResponse } from '../types.js';

/**
 * Mock data for available MCP tools.
 * In a real implementation, this might come from a database or service discovery.
 */
const availableTools = [
    {
        name: "mcp_vrooli_mcp_server_calculate_sum",
        description: "Add two numbers together",
        endpoint: "http://localhost:8080/api/tools/calculate_sum" // Example endpoint
    },
    {
        name: "mcp_vrooli_mcp_server_fetch_resource",
        description: "Fetch a resource by name or search string",
        endpoint: "http://localhost:8080/api/tools/fetch_resource" // Example endpoint
    },
    {
        name: "mcp_vrooli_mcp_server_fetch_tool",
        description: "Fetch/search for available MCP tools",
        endpoint: "http://localhost:8080/api/tools/fetch_tool" // Example endpoint for the tool itself
    },
    {
        name: "external_weather_tool",
        description: "Get the current weather for a location",
        endpoint: "http://weather-service.example.com/api/get_weather"
    },
    {
        name: "image_generation_tool",
        description: "Generate an image based on a text prompt",
        endpoint: "http://image-gen.example.com/api/generate"
    }
];

/**
 * Implementation of the fetch_tool tool. Searches for available MCP tools based on a query.
 *
 * @param args - The arguments for the tool, containing the search query.
 * @param logger - The logger instance for logging operations.
 * @returns A ToolResponse object containing a list of matching tools or a message if none found.
 */
export function fetchTool(args: { query: string }, logger: Logger): ToolResponse {
    const { query } = args;

    // Validate query
    if (!query || typeof query !== 'string' || query.trim() === "") {
        logger.warn("Fetch tool called with empty or invalid query.");
        return {
            content: [
                {
                    type: "text",
                    text: "Please provide a non-empty search query to find tools."
                }
            ]
        };
    }

    const normalizedQuery = query.toLowerCase().trim();
    logger.info(`Searching for tools with query: "${normalizedQuery}"`);

    const foundTools = availableTools.filter(tool =>
        tool.name.toLowerCase().includes(normalizedQuery) ||
        tool.description.toLowerCase().includes(normalizedQuery)
    );

    let responseText = "";
    if (foundTools.length > 0) {
        logger.info(`Found ${foundTools.length} tools matching query "${query}".`);
        responseText = `Found the following tools matching "${query}":\n\n`;
        foundTools.forEach((tool, index) => {
            responseText += `Name: ${tool.name}\nDescription: ${tool.description}\nEndpoint: ${tool.endpoint}`;
            if (index < foundTools.length - 1) {
                responseText += '\n\n---\n\n';
            }
        });
    } else {
        logger.info(`No tools found matching query: "${query}"`);
        responseText = `No tools found matching query: "${query}"`;
    }

    return {
        content: [
            {
                type: "text",
                text: responseText
            }
        ]
    };
} 