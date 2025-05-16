import { McpToolName, ToolRegistry } from "../mcp/registry.js";
import { OkErr, ToolMeta } from "./types.js";

/**
 * The result of a tool call, including the output and the number of credits used.
 */
export interface ToolCallResult {
    output: unknown;
    creditsUsed: bigint;
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
    /**
     * Executes an OpenAI built-in tool call.
     * @param name - The tool name (e.g., 'web_search', 'file_search', 'function').
     * @param _args - The arguments for the tool.
     * @param _meta - Metadata about the tool call.
     * @returns A promise resolving to a ToolCallResult or an error.
     */
    async run(name: string, _args: unknown, _meta: ToolMeta): Promise<OkErr<ToolCallResult>> {
        // TODO: Integrate with OpenAI Responses API for built-in tools.
        // This is a stub implementation for now.
        // See: https://medium.com/@odhitom09/openai-responses-api-a-comprehensive-guide-ad546132b2ed
        return {
            ok: false,
            error: {
                code: "NOT_IMPLEMENTED",
                message: `OpenAIToolRunner is not yet implemented for tool: ${name}`,
            },
        };
    }
}

/**
 * Handles custom MCP tools as defined in the MCP registry (e.g., resource management, routines, etc.).
 * Validates arguments, dispatches to the correct handler, and returns a structured result.
 */
export class McpToolRunner extends ToolRunner {
    private readonly registry: ToolRegistry;

    /**
     * @param registry - The MCP tool registry instance.
     */
    constructor(registry: ToolRegistry) {
        super();
        this.registry = registry;
    }

    /**
     * Executes a custom MCP tool call.
     * @param name - The MCP tool name (must be a valid McpToolName).
     * @param args - The arguments for the tool.
     * @param _meta - Metadata about the tool call.
     * @returns A promise resolving to a ToolCallResult or an error.
     */
    async run(name: string, args: unknown, _meta: ToolMeta): Promise<OkErr<ToolCallResult>> {
        // Validate tool name
        if (!Object.values(McpToolName).includes(name as McpToolName)) {
            return {
                ok: false,
                error: {
                    code: "UNKNOWN_MCP_TOOL",
                    message: `Unknown MCP tool: ${name}`,
                },
            };
        }
        // Execute the tool using the registry
        try {
            const response = await this.registry.execute(name as McpToolName, args);
            if (response.isError) {
                return {
                    ok: false,
                    error: {
                        code: "MCP_TOOL_EXECUTION_FAILED",
                        message: response.content?.[0]?.text || `MCP tool '${name}' execution failed`,
                    },
                };
            }
            // TODO: Extract creditsUsed from response if available (default to 0)
            return {
                ok: true,
                data: {
                    output: response,
                    creditsUsed: BigInt(0),
                },
            };
        } catch (error) {
            return {
                ok: false,
                error: {
                    code: "MCP_TOOL_EXECUTION_ERROR",
                    message: `MCP tool '${name}' execution error: ${(error as Error).message}`,
                },
            };
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
     * @param openaiRunner - The OpenAI tool runner instance (optional).
     * @param mcpRunner - The MCP tool runner instance (optional).
     */
    constructor(
        private readonly openaiRunner: OpenAIToolRunner = new OpenAIToolRunner(),
        private readonly mcpRunner: McpToolRunner = new McpToolRunner(new ToolRegistry(console)),
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
        // Heuristic: If the tool name matches an MCP tool, use MCP runner; otherwise, use OpenAI runner.
        if (Object.values(McpToolName).includes(name as McpToolName)) {
            return this.mcpRunner.run(name, args, meta);
        }
        // Otherwise, assume it's an OpenAI built-in tool
        return this.openaiRunner.run(name, args, meta);
    }
}
