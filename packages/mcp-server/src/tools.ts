import { Logger, ToolResponse } from './types.js';

/**
 * Defines the allowed resource types.
 */
type ResourceType = "notes" | "reminders";

/**
 * Centralized store for resources (notes and reminders).
 * The outer map uses ResourceType as keys.
 * The inner map uses the resource name (string) as keys and content (string) as values.
 */
const resourceStore: Map<ResourceType, Map<string, string>> = new Map([
    ["notes", new Map<string, string>([
        ["Meeting Notes - Project Alpha", "Discussed roadmap milestones. Action items assigned."],
        ["Brainstorming Ideas", "- Integrate new AI model\n- Refactor UI components"]
    ])],
    ["reminders", new Map<string, string>([
        ["Reminder: Submit Report", "Submit the quarterly report by EOD Friday."],
        ["Reminder: Call John", "Call John Doe regarding the project update tomorrow at 10 AM."]
    ])]
]);

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
        description: "Fetch a resource by name and type (notes or reminders)",
        endpoint: "http://localhost:8080/api/tools/fetch_resource" // Example endpoint
    },
    {
        name: "mcp_vrooli_mcp_server_add_resource",
        description: "Add or update a resource (note or reminder)",
        endpoint: "http://localhost:8080/api/tools/add_resource" // Example endpoint
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

// --- Resource Management Tools ---

/**
 * Interface for the arguments accepted by the fetch_resource tool.
 */
interface FetchResourceArgs {
    /** The exact name of the resource to fetch. */
    name: string;
    /** The type of resource to fetch (notes or reminders). */
    resource_type: ResourceType;
}

/**
 * Implementation of the fetch_resource tool.
 * Fetches resources like notes or reminders by exact name and type from the resourceStore.
 *
 * @param args - The arguments for the tool, including name and resource_type.
 * @param logger - The logger instance for logging operations.
 * @returns A ToolResponse object containing the found resource or an error message.
 */
export function fetchResource(args: FetchResourceArgs, logger: Logger): ToolResponse {
    const { name, resource_type } = args;

    logger.info(`Attempting to fetch resource of type "${resource_type}" with name: "${name}"`);

    // Validate resource type
    if (resource_type !== "notes" && resource_type !== "reminders") {
        logger.warn(`Invalid resource_type specified: "${resource_type}"`);
        return {
            content: [
                {
                    type: "text",
                    text: `Error: Invalid resource_type "${resource_type}". Allowed types are "notes" or "reminders".`
                }
            ]
        };
    }

    // Validate name
    if (!name || typeof name !== 'string' || name.trim() === "") {
        logger.warn(`Fetch resource called with empty or invalid name for type "${resource_type}".`);
        return {
            content: [
                {
                    type: "text",
                    text: `Please provide a non-empty name for resource type "${resource_type}".`
                }
            ]
        };
    }

    const typeStore = resourceStore.get(resource_type);
    if (!typeStore) {
        // This case should ideally not happen if validation is correct, but good for robustness
        logger.error(`Internal error: No store found for valid resource type "${resource_type}".`);
        return { content: [{ type: "text", text: `Internal error fetching resource type "${resource_type}".` }] };
    }

    const resourceContent = typeStore.get(name);

    if (resourceContent !== undefined) {
        logger.info(`Successfully fetched resource: "${name}" (Type: ${resource_type})`);
        return {
            content: [
                {
                    type: "text",
                    text: `Resource Type: ${resource_type}
Name: ${name}

${resourceContent}`
                }
            ]
        };
    } else {
        logger.info(`Resource not found: "${name}" (Type: ${resource_type})`);
        return {
            content: [
                {
                    type: "text",
                    text: `Resource not found with name "${name}" for type "${resource_type}".`
                }
            ]
        };
    }
}

/**
 * Interface for the arguments accepted by the add_resource tool.
 */
interface AddResourceArgs {
    /** The name of the resource to add or update. */
    name: string;
    /** The type of the resource (notes or reminders). */
    resource_type: ResourceType;
    /** The content of the resource. */
    content: string;
}

/**
 * Implementation of the add_resource tool.
 * Adds or updates a resource (note or reminder) in the resourceStore.
 *
 * @param args - The arguments for the tool, including name, resource_type, and content.
 * @param logger - The logger instance for logging operations.
 * @returns A ToolResponse object confirming the addition/update or reporting an error.
 */
export function addResource(args: AddResourceArgs, logger: Logger): ToolResponse {
    const { name, resource_type, content } = args;

    logger.info(`Attempting to add/update resource "${name}" of type "${resource_type}"`);

    // Validate resource type
    if (resource_type !== "notes" && resource_type !== "reminders") {
        logger.warn(`Invalid resource_type specified: "${resource_type}"`);
        return {
            content: [{ type: "text", text: `Error: Invalid resource_type "${resource_type}". Allowed types are "notes" or "reminders".` }]
        };
    }

    // Validate name
    if (!name || typeof name !== 'string' || name.trim() === "") {
        logger.warn(`Add resource called with empty or invalid name for type "${resource_type}".`);
        return {
            content: [{ type: "text", text: `Please provide a non-empty name for resource type "${resource_type}".` }]
        };
    }

    // Validate content (basic check for non-empty string)
    if (typeof content !== 'string' || content.trim() === "") {
        logger.warn(`Add resource called with empty or invalid content for "${name}" (Type: ${resource_type}).`);
        return {
            content: [{ type: "text", text: `Please provide non-empty content for resource "${name}".` }]
        };
    }

    // Get the specific store for the type, creating if it doesn't exist (though it should based on initial setup)
    let typeStore = resourceStore.get(resource_type);
    if (!typeStore) {
        logger.warn(`Creating new store for resource type "${resource_type}" as it did not exist.`);
        typeStore = new Map<string, string>();
        resourceStore.set(resource_type, typeStore);
    }

    // Add or update the resource
    typeStore.set(name, content);

    logger.info(`Successfully added/updated resource: "${name}" (Type: ${resource_type})`);
    return {
        content: [
            {
                type: "text",
                text: `Successfully added/updated resource "${name}" of type "${resource_type}".`
            }
        ]
    };
} 