import { McpToolName } from "@vrooli/shared";
import type { Logger } from "winston";
import { RequestService } from "../../auth/request.js";
import defineToolSchemaJson from "../schemas/DefineTool/schema.json" with { type: "json" };
import endSwarmSchemaJson from "../schemas/EndSwarm/schema.json" with { type: "json" };
import resourceManageSchemaJson from "../schemas/ResourceManage/schema.json" with { type: "json" };
import runRoutineSchemaJson from "../schemas/RunRoutine/schema.json" with { type: "json" };
import sendMessageSchemaJson from "../schemas/SendMessage/schema.json" with { type: "json" };
import spawnSwarmSchemaJson from "../schemas/SpawnSwarm/schema.json" with { type: "json" };
import updateSwarmStateSchemaJson from "../schemas/UpdateSwarmSharedState/schema.json" with { type: "json" };
import type { ResourceManageParams, RunRoutineParams } from "../types/tools.js";
import { type DefineToolParams, type SendMessageParams, type SpawnSwarmParams } from "../types/tools.js";
import { getCurrentMcpContext } from "./context.js";
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
    private logger: Logger;

    constructor(logger: Logger) {
        this.logger = logger;
    }

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
        this.logger.info(`Executing tool: ${name} with args: ${JSON.stringify(args)}`);

        try {
            const { req } = getCurrentMcpContext();
            await RequestService.get().rateLimit({ maxUser: 5_000, req });

            switch (name) {
                case McpToolName.DefineTool: {
                    const user = RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true, isUser: true });
                    return (new BuiltInTools(user, this.logger, req)).defineTool(args as DefineToolParams);
                }
                case McpToolName.SendMessage: {
                    const user = RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
                    return (new BuiltInTools(user, this.logger, req)).sendMessage(args as SendMessageParams);
                }
                case McpToolName.ResourceManage: {
                    const user = RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
                    return (new BuiltInTools(user, this.logger, req)).resourceManage(args as ResourceManageParams);
                }
                case McpToolName.RunRoutine: {
                    const user = RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
                    return (new BuiltInTools(user, this.logger, req)).runRoutine(args as RunRoutineParams);
                }
                case McpToolName.SpawnSwarm: {
                    const user = RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
                    return (new BuiltInTools(user, this.logger, req)).spawnSwarm(args as SpawnSwarmParams);
                }
                default: {
                    const exhaustiveCheck: never = name;
                    this.logger.error(`Tool handler not found or not implemented in BuiltInTools: ${exhaustiveCheck}`);
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
            this.logger.error(`Error executing tool ${name}:`, error);
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
