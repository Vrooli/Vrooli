import { McpSwarmToolName, McpToolName } from "@vrooli/shared";
import type OpenAI from "openai";
import { logger } from "../../events/logger.js";
import { type OkErr, type ToolMeta } from "../conversation/types.js";
import { BuiltInTools, type SwarmTools } from "../mcp/tools.js";
import { type ToolResponse } from "../mcp/types.js";
import { type DefineToolParams, type EndSwarmParams, type ResourceManageParams, type RunRoutineParams, type SendMessageParams, type SpawnSwarmParams, type UpdateSwarmSharedStateParams } from "../types/tools.js";

/**
 * The result of a tool call, including the output and the number of credits used.
 */
interface ToolCallResult {
    output: unknown;
    creditsUsed: string; // stringified BigInt
}

/**
 * Abstract base class for all tool runners.
 * Implementations must validate arguments, execute the tool, and return a structured result.
 */
export abstract class ToolRunner {
    /**
     * Dispatch a tool call – must validate args against schema, call the appropriate service, and return a structured result.
     * @param name - The name of the tool to run.
     * @param args - The arguments for the tool.
     * @param meta - Metadata about the tool call (e.g., conversation, bot, etc.).
     * @returns A promise resolving to a ToolCallResult or an error.
     */
    abstract run(
        name: string,
        args: unknown,
        meta: ToolMeta
    ): Promise<OkErr<ToolCallResult>>;
}

/**
 * Handles built-in tools provided by the OpenAI Responses API (e.g., web_search, file_search, function, etc.).
 * This class should be extended to support additional OpenAI tools as needed.
 */
export class OpenAIToolRunner extends ToolRunner {
    private readonly openaiClient: OpenAI | null;

    /**
     * @param openaiClient - The OpenAI client instance.
     */
    constructor(openaiClient: OpenAI | null) {
        super();
        this.openaiClient = openaiClient;
    }

    /**
     * Executes an OpenAI built-in tool call.
     * @param name - The tool name (e.g., "web_search", "file_search", "function").
     * @param args - The arguments for the tool.
     * @param _meta - Metadata about the tool call (prefixed with underscore as it's not used yet).
     * @returns A promise resolving to a ToolCallResult or an error.
     */
    async run(name: string, args: unknown, _meta: ToolMeta): Promise<OkErr<ToolCallResult>> {
        // TODO: Replace placeholder with actual OpenAI API client initialization/access if this.openaiClient is not null.
        // This client might be injected via the constructor or accessed from the 'meta' object.

        try {
            let toolOutput: unknown;
            let creditsUsed = "0";

            switch (name) {
                case "web_search":
                    toolOutput = `Simulated web search for tool "${name}" with args: ${JSON.stringify(args)}`;
                    creditsUsed = "10"; // Example cost
                    break;

                case "file_search":
                    toolOutput = `Simulated file search for tool "${name}" with args: ${JSON.stringify(args)}`;
                    creditsUsed = "5"; // Example cost
                    break;

                default:
                    return {
                        ok: false,
                        error: {
                            code: "UNKNOWN_OPENAI_TOOL",
                            message: `OpenAI tool named "${name}" is not recognized or supported by this runner.`,
                            creditsUsed: "1", // Example: small cost for attempting an unknown tool
                        },
                    };
            }

            return {
                ok: true,
                data: {
                    output: toolOutput,
                    creditsUsed,
                },
            };
        } catch (error) {
            return {
                ok: false,
                error: {
                    code: "OPENAI_API_ERROR",
                    message: `An error occurred while executing OpenAI tool "${name}": ${(error instanceof Error) ? error.message : String(error)}`,
                    creditsUsed: "1", // Example: small cost for an API error
                },
            };
        }
    }
}

/**
 * Handles custom MCP tools as defined in the MCP registry (e.g., resource management, routines, etc.).
 * Validates arguments, dispatches to the correct handler, and returns a structured result.
 */
export class McpToolRunner extends ToolRunner {
    private readonly swarmTools?: SwarmTools;

    /**
     * @param logger - A logger instance.
     * @param swarmTools - An optional instance of SwarmTools for handling swarm-specific MCP tools.
     */
    constructor(swarmTools?: SwarmTools) {
        super();
        this.swarmTools = swarmTools;
    }

    /**
     * Executes a custom MCP tool call.
     * @param name - The MCP tool name (must be a valid McpToolName or McpSwarmToolName).
     * @param args - The arguments for the tool.
     * @param meta - Metadata about the tool call (includes conversationId, callerBotId, sessionUser).
     * @returns A promise resolving to a ToolCallResult or an error.
     */
    async run(name: string, args: unknown, meta: ToolMeta): Promise<OkErr<ToolCallResult>> {
        let toolExecuteResponse: ToolResponse | null = null;
        let swarmToolInternalResult: { success: boolean; data?: any; error?: any; message?: string } | null = null;

        try {
            if (Object.values(McpToolName).includes(name as McpToolName)) {
                if (!meta.sessionUser) {
                    logger.error(`McpToolRunner: sessionUser is required in meta for McpToolName: ${name}`);
                    return { ok: false, error: { code: "MISSING_SESSION_USER_FOR_MCP_TOOL", message: `SessionUser missing for MCP tool ${name}.`, creditsUsed: "0" } };
                }
                const builtInTools = new BuiltInTools(meta.sessionUser, logger, undefined /* req */);
                switch (name as McpToolName) {
                    case McpToolName.DefineTool: {
                        toolExecuteResponse = await builtInTools.defineTool(args as DefineToolParams);
                        break;
                    }
                    case McpToolName.SendMessage: {
                        toolExecuteResponse = await builtInTools.sendMessage(args as SendMessageParams);
                        break;
                    }
                    case McpToolName.ResourceManage: {
                        toolExecuteResponse = await builtInTools.resourceManage(args as ResourceManageParams);
                        break;
                    }
                    case McpToolName.RunRoutine: {
                        toolExecuteResponse = await builtInTools.runRoutine(args as RunRoutineParams);
                        break;
                    }
                    case McpToolName.SpawnSwarm: {
                        toolExecuteResponse = await builtInTools.spawnSwarm(args as SpawnSwarmParams);
                        break;
                    }
                    default:
                        logger.error(`McpToolRunner: Unhandled McpToolName: ${name}`);
                        return { ok: false, error: { code: "UNHANDLED_MCP_TOOL", message: `Tool ${name} not handled.`, creditsUsed: "0" } };
                }
            } else if (this.swarmTools && Object.values(McpSwarmToolName).includes(name as McpSwarmToolName)) {
                if (!meta.conversationId) {
                    logger.error(`McpToolRunner: conversationId is required for SwarmTool: ${name}`);
                    return { ok: false, error: { code: "MISSING_CONVERSATION_ID_FOR_SWARM_TOOL", message: `ConversationId missing for swarm tool ${name}.`, creditsUsed: "0" } };
                }
                if (!meta.sessionUser) {
                    logger.error(`McpToolRunner: sessionUser is required for SwarmTool: ${name}`);
                    return { ok: false, error: { code: "MISSING_SESSION_USER_FOR_SWARM_TOOL", message: `SessionUser missing for swarm tool ${name}.`, creditsUsed: "0" } };
                }
                switch (name as McpSwarmToolName) {
                    case McpSwarmToolName.UpdateSwarmSharedState: {
                        const { success, error, message, ...rest } = await this.swarmTools.updateSwarmSharedState(meta.conversationId, args as UpdateSwarmSharedStateParams, meta.sessionUser);
                        swarmToolInternalResult = {
                            success,
                            data: rest,
                            error,
                            message,
                        };
                        break;
                    }
                    case McpSwarmToolName.EndSwarm: {
                        const endResult = await this.swarmTools.endSwarm(meta.conversationId, args as EndSwarmParams, meta.sessionUser);
                        swarmToolInternalResult = { success: endResult.success, data: { finalState: endResult.finalState }, error: endResult.error, message: endResult.message };
                        break;
                    }
                    default:
                        logger.error(`McpToolRunner: Unhandled McpSwarmToolName: ${name}`);
                        return { ok: false, error: { code: "UNHANDLED_SWARM_TOOL", message: `Swarm tool ${name} not handled.`, creditsUsed: "0" } };
                }
                // Convert swarmToolInternalResult to toolExecuteResponse structure
                if (swarmToolInternalResult) {
                    if (swarmToolInternalResult.success) {
                        toolExecuteResponse = { isError: false, content: [{ type: "text", text: JSON.stringify(swarmToolInternalResult.data) }], creditsUsed: "0" };
                    } else {
                        toolExecuteResponse = { isError: true, content: [{ type: "text", text: swarmToolInternalResult.message || JSON.stringify(swarmToolInternalResult.error) }], creditsUsed: "0" };
                    }
                }

            } else {
                return { ok: false, error: { code: "UNKNOWN_TOOL_CATEGORY", message: `Tool "${name}" not recognized as MCP or Swarm tool.`, creditsUsed: "0" } };
            }

            if (!toolExecuteResponse) {
                logger.error(`McpToolRunner: toolExecuteResponse is null after processing tool ${name}`);
                return { ok: false, error: { code: "INTERNAL_TOOL_RUNNER_ERROR", message: "Tool response was not generated.", creditsUsed: "0" } };
            }

            const actualCreditsUsed = toolExecuteResponse.creditsUsed || BigInt(0).toString();

            if (toolExecuteResponse.isError) {
                return { ok: false, error: { code: "TOOL_EXECUTION_FAILED", message: toolExecuteResponse.content?.[0]?.text || `Tool "${name}" failed.`, creditsUsed: actualCreditsUsed } };
            }

            // Determine the output for the LLM
            let outputForLlm = toolExecuteResponse.content?.[0]?.text;
            if (name === McpSwarmToolName.UpdateSwarmSharedState && swarmToolInternalResult?.success) {
                outputForLlm = swarmToolInternalResult.data; // Return the structured data for successful UpdateSwarmSharedState
            }

            return { ok: true, data: { output: outputForLlm, creditsUsed: actualCreditsUsed } };

        } catch (error) {
            logger.error(`McpToolRunner: Exception executing tool ${name}:`, error);
            return { ok: false, error: { code: "TOOL_RUNNER_EXCEPTION", message: `Exception for tool "${name}": ${(error as Error).message}`, creditsUsed: "0" } };
        }
    }
}

/**
 * Composite tool runner that delegates tool calls to the appropriate runner (OpenAI or MCP).
 * This class provides a single interface for tool execution in the conversation loop and related services.
 *
 * By default, this will create a ToolRegistry and use it to initialize the McpToolRunner.
 * If you need to provide a custom registry or runners, you can pass them to the constructor.
 */
export class CompositeToolRunner extends ToolRunner {
    /**
     * @param mcpRunner - The MCP tool runner instance.
     * @param openaiRunner - The OpenAI tool runner instance (optional).
     */
    constructor(
        private readonly mcpRunner: McpToolRunner,
        private readonly openaiRunner: OpenAIToolRunner = new OpenAIToolRunner(null),
    ) {
        super();
    }

    /**
     * Dispatches the tool call to the correct runner based on the tool name/type.
     * @param name - The tool name.
     * @param args - The arguments for the tool.
     * @param meta - Metadata about the tool call.
     * @returns A promise resolving to a ToolCallResult or an error.
     */
    async run(name: string, args: unknown, meta: ToolMeta): Promise<OkErr<ToolCallResult>> {
        if (
            Object.values(McpToolName).includes(name as McpToolName) ||
            Object.values(McpSwarmToolName).includes(name as McpSwarmToolName)
        ) {
            return this.mcpRunner.run(name, args, meta);
        }
        return this.openaiRunner.run(name, args, meta);
    }
}
