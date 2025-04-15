import { Logger, ToolResponse } from './types.js';

/**
 * Defines the allowed resource types.
 */
enum ResourceType {
    Note = "Note",
    Reminder = "Reminder",
    Routine = "Routine",
    User = "User"
}

type Routine = {
    id: string;
    content: string;
    name: string;
    description?: string;
}

/**
 * Centralized store for resources (notes and reminders).
 * The outer map uses ResourceType as keys.
 * The inner map uses the resource name (string) as keys and content (string) as values.
 */
const resourceStore = new Map<ResourceType, Map<string, unknown>>([
    [ResourceType.Note, new Map<string, string>([
        ["Meeting Notes - Project Alpha", "Discussed roadmap milestones. Action items assigned."],
        ["Brainstorming Ideas", "- Integrate new AI model\n- Refactor UI components"]
    ])],
    [ResourceType.Reminder, new Map<string, string>([
        ["Reminder: Submit Report", "Submit the quarterly report by EOD Friday."],
        ["Reminder: Call John", "Call John Doe regarding the project update tomorrow at 10 AM."]
    ])],
    [ResourceType.Routine, new Map<string, Routine>([
        ["c9dd779d-ebf2-4e65-8429-4eef5c40aa4a", {
            id: "c9dd779d-ebf2-4e65-8429-4eef5c40aa4a",
            content: "Daily standup meeting with {arg}.",
            name: "Daily Standup",
            description: "Perform a daily standup meeting with the team."
        }],
        ["25020e82-ecfb-4ca3-afc9-f3a88ba77e76", {
            id: "25020e82-ecfb-4ca3-afc9-f3a88ba77e76",
            content: "Weekly review of the {arg} status.",
            name: "Weekly Review"
        }]
    ])],
    [ResourceType.User, new Map<string, string>([
        ["John Doe", "john.doe@example.com"]
    ])]
]);

/**
 * Finds an existing Routine by name.
 *
 * @param args - The arguments for the tool, containing the search query.
 * @param logger - The logger instance for logging operations.
 * @returns Routines that match the search query.
 */
export async function findRoutine(args: { id?: string, query?: string }, logger: Logger): Promise<Routine[]> {
    const { id, query } = args;

    const safeId = typeof id === 'string' ? id : "";
    const safeQuery = typeof query === 'string' ? query : "";
    const normalizedQuery = safeQuery.toLowerCase().trim();

    logger.info(`Searching for tools with id: "${safeId}" or query: "${normalizedQuery}"`);

    // Search resource store for matching routines
    const foundRoutines = (Array.from(resourceStore.get(ResourceType.Routine)?.values() || []) as Routine[]).filter((routine) => {
        if (safeId.length > 0) {
            return routine.id === safeId;
        }
        return routine.name.toLowerCase().includes(normalizedQuery) ||
            routine.content.toLowerCase().includes(normalizedQuery);
    });

    return foundRoutines;
}

/**
 * Interface for the arguments accepted by the run_routine tool.
 */
interface RunRoutineParams {
    /** The ID of the routine to run. */
    id: string;
    /** A string argument to pass to the routine. */
    replacement: string;
}

/**
 * Performs a mock run routine.
 * 
 * @param args - The arguments for the tool, containing the routine name and replacement string.
 * @param logger - The logger instance for logging operations.
 * @returns A ToolResponse object containing the result of the routine.
 */
export async function runRoutine(args: RunRoutineParams, logger: Logger): Promise<ToolResponse> {
    const { id, replacement } = args;

    logger.info(`Running routine: "${id}" with replacement: "${replacement}"`);

    // Find the routine
    const routine = await findRoutine({ id }, logger);

    if (routine.length === 0) {
        logger.warn(`No routine found matching id: "${id}"`);
        return {
            content: [{ type: "text", text: `No routine found matching id: "${id}".` }]
        };
    }

    const { content } = routine[0];

    const result = content.replace('{arg}', replacement);

    return {
        content: [{ type: "text", text: result }]
    };
}

// --- Resource Management Tools ---

/**
 * Interface for the arguments accepted by the fetch_resource tool.
 */
interface FindResourceParams {
    /** The exact name of the resource to fetch. */
    name: string;
    /** The type of resource to fetch (notes or reminders). */
    resource_type: ResourceType;
}

/**
 * Implementation of the find_resource tool.
 * Fetches resources like notes or reminders by exact name and type from the resourceStore.
 *
 * @param args - The arguments for the tool, including name and resource_type.
 * @param logger - The logger instance for logging operations.
 * @returns A ToolResponse object containing the found resource or an error message.
 */
export async function findResource(args: FindResourceParams, logger: Logger): Promise<ToolResponse> {
    const { name, resource_type } = args;

    logger.info(`Attempting to fetch resource of type "${resource_type}" with name: "${name}"`);

    // Validate resource type
    if (!Object.values(ResourceType).includes(resource_type)) {
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
export async function addResource(args: AddResourceArgs, logger: Logger): Promise<ToolResponse> {
    const { name, resource_type, content } = args;

    logger.info(`Attempting to add/update resource "${name}" of type "${resource_type}"`);

    // Validate resource type
    if (!Object.values(ResourceType).includes(resource_type)) {
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

/**
 * Interface for the arguments accepted by the delete_resource tool.
 */
interface DeleteResourceParams {
    /** The name of the resource to delete. */
    resource_type: ResourceType;
    /** The name of the resource to delete. */
    name: string;
}

/**
 * Implementation of the delete_resource tool.
 * Deletes a resource (note or reminder) from the resourceStore.
 *
 * @param args - The arguments for the tool, including resource_type.
 * @param logger - The logger instance for logging operations.
 * @returns A ToolResponse object confirming the deletion or reporting an error.
 */
export async function deleteResource(args: DeleteResourceParams, logger: Logger): Promise<ToolResponse> {
    const { resource_type, name } = args;

    logger.info(`Attempting to delete resource of type "${resource_type}"`);

    // Validate resource type
    if (!Object.values(ResourceType).includes(resource_type)) {
        logger.warn(`Invalid resource_type specified: "${resource_type}"`);
        return {
            content: [{ type: "text", text: `Error: Invalid resource_type "${resource_type}". Allowed types are "notes" or "reminders".` }]
        };
    }

    // Get the specific store for the type
    let typeStore = resourceStore.get(resource_type);
    if (!typeStore) {
        logger.warn(`No store found for resource type "${resource_type}".`);
        return { content: [{ type: "text", text: `No store found for resource type "${resource_type}".` }] };
    }

    // Delete the resource
    typeStore.delete(name);

    logger.info(`Successfully deleted resource: "${name}" (Type: ${resource_type})`);
    return {
        content: [
            {
                type: "text",
                text: `Successfully deleted resource "${name}" of type "${resource_type}".`
            }
        ]
    };
}

/**
 * Interface for the arguments accepted by the update_resource tool.
 */
interface UpdateResourceParams {
    /** The name of the resource to update. */
    name: string;
    /** The type of the resource (notes or reminders). */
    resource_type: ResourceType;
    /** The new content of the resource. */
    content: string;
}


/**
 * Implementation of the update_resource tool.
 * Updates the content of a resource (note or reminder) in the resourceStore.
 *
 * @param args - The arguments for the tool, including name, resource_type, and content.
 * @param logger - The logger instance for logging operations.
 * @returns A ToolResponse object confirming the update or reporting an error.
 */
export async function updateResource(args: UpdateResourceParams, logger: Logger): Promise<ToolResponse> {
    const { name, resource_type, content } = args;

    logger.info(`Attempting to update resource "${name}" of type "${resource_type}"`);

    // Validate resource type
    if (!Object.values(ResourceType).includes(resource_type)) {
        logger.warn(`Invalid resource_type specified: "${resource_type}"`);
        return {
            content: [{ type: "text", text: `Error: Invalid resource_type "${resource_type}". Allowed types are "notes" or "reminders".` }]
        };
    }

    // Get the specific store for the type
    let typeStore = resourceStore.get(resource_type);
    if (!typeStore) {
        logger.warn(`No store found for resource type "${resource_type}".`);
        return { content: [{ type: "text", text: `No store found for resource type "${resource_type}".` }] };
    }

    // Update the resource
    typeStore.set(name, content);

    logger.info(`Successfully updated resource: "${name}" (Type: ${resource_type})`);
    return {
        content: [
            {
                type: "text",
                text: `Successfully updated resource "${name}" of type "${resource_type}".`
            }
        ]
    };
}