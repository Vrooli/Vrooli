import { McpToolName } from "@vrooli/shared";
import { RequestService } from "../../auth/request.js";
import { logger } from "../../events/logger.js";
import type { ResourceManageParams, RunRoutineParams } from "../types/tools.js";
import { type DefineToolParams, type SendMessageParams, type SpawnSwarmParams } from "../types/tools.js";
import { getCurrentMcpContext } from "./context.js";
import { loadSchema } from "./schemaLoader.js";
import { BuiltInTools } from "./tools.js";
import { type ToolResponse } from "./types.js";

// Define a mapped type for tool arguments
type ToolArgsMap = {
    [McpToolName.DefineTool]: DefineToolParams;
    [McpToolName.SendMessage]: SendMessageParams;
    [McpToolName.ResourceManage]: ResourceManageParams;
    [McpToolName.RunRoutine]: RunRoutineParams;
    [McpToolName.SpawnSwarm]: SpawnSwarmParams;
};

// Load schemas using the schema loader utility
const defineToolSchemaJson = loadSchema("DefineTool/schema.json");
const endSwarmSchemaJson = loadSchema("EndSwarm/schema.json");
const resourceManageSchemaJson = loadSchema("ResourceManage/schema.json");
const runRoutineSchemaJson = loadSchema("RunRoutine/schema.json");
const sendMessageSchemaJson = loadSchema("SendMessage/schema.json");
const spawnSwarmSchemaJson = loadSchema("SpawnSwarm/schema.json");
const updateSwarmStateSchemaJson = loadSchema("UpdateSwarmSharedState/schema.json");

const BUILT_IN_TOOL_DEFINITIONS = [
    defineToolSchemaJson,
    sendMessageSchemaJson,
    resourceManageSchemaJson,
    runRoutineSchemaJson,
    spawnSwarmSchemaJson,
];

const SWARM_TOOL_DEFINITIONS = [
    updateSwarmStateSchemaJson,
    endSwarmSchemaJson,
];

/**
 * ---------------------------------------------------------------------------
 *  TOOL REGISTRY – *single source of truth* for the MCP root‑level interface
 * ---------------------------------------------------------------------------
 */
export class ToolRegistry {

    /**
     * Get built-in tool definitions as an array of JSON objects.
     */
    getBuiltInDefinitions() {
        return BUILT_IN_TOOL_DEFINITIONS;
    }

    /**
     * Get swarm tool definitions as an array of JSON objects.
     */
    getSwarmToolDefinitions() {
        return SWARM_TOOL_DEFINITIONS;
    }

    /**
     * Get a specific tool definition by its name.
     * @param name The name of the tool to find.
     * @returns The tool definition object if found, otherwise undefined.
     */
    getToolDefinition(name: string) {
        const allTools = [...BUILT_IN_TOOL_DEFINITIONS, ...SWARM_TOOL_DEFINITIONS];
        // Assuming each tool definition object has a 'name' property matching the Tool interface
        return allTools.find(tool => tool.name === name);
    }

    /**
     * Execute a built-in tool by name
     * @param name Tool name
     * @param args Tool arguments
     * @returns Tool execution response
     */
    async execute<T extends McpToolName>(name: T, args: ToolArgsMap[T]): Promise<ToolResponse> {
        logger.info(`Executing tool: ${name} with args: ${JSON.stringify(args)}`);

        try {
            const { req } = getCurrentMcpContext();
            await RequestService.get().rateLimit({ maxUser: 5_000, req });

            switch (name) {
                case McpToolName.DefineTool: {
                    const user = RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true, isUser: true });
                    return (new BuiltInTools(user, req)).defineTool(args as DefineToolParams);
                }
                case McpToolName.SendMessage: {
                    const user = RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
                    return (new BuiltInTools(user, req)).sendMessage(args as SendMessageParams);
                }
                case McpToolName.ResourceManage: {
                    const user = RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
                    return (new BuiltInTools(user, req)).resourceManage(args as ResourceManageParams);
                }
                case McpToolName.RunRoutine: {
                    const user = RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
                    return (new BuiltInTools(user, req)).runRoutine(args as RunRoutineParams);
                }
                case McpToolName.SpawnSwarm: {
                    const user = RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
                    return (new BuiltInTools(user, req)).spawnSwarm(args as SpawnSwarmParams);
                }
                default: {
                    const exhaustiveCheck: never = name;
                    logger.error(`Tool handler not found or not implemented in BuiltInTools: ${exhaustiveCheck}`);
                    return {
                        isError: true,
                        content: [
                            {
                                type: "text",
                                text: `Error: Tool handler for '${exhaustiveCheck}' not found or not implemented.`,
                            },
                        ],
                    };
                }
            }
        } catch (error) {
            logger.error(`Error executing tool ${name}:`, error);
            // Check if the error is from context retrieval (e.g., outside runWithMcpContext)
            if (error instanceof Error && error.message.includes("No MCP HTTP context is set")) {
                return {
                    isError: true,
                    content: [
                        {
                            type: "text",
                            text: `Error executing tool '${name}': MCP context not available. This tool might have been called from a non-request path. Original error: ${(error as Error).message}`,
                        },
                    ],
                };
            }
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
