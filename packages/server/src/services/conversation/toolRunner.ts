import OpenAI from "openai";
import { McpToolName, ToolRegistry } from "../mcp/registry.js";
import { OkErr, ToolMeta } from "./types.js";

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
    private readonly openaiClient: OpenAI;

    /**
     * @param openaiClient - The OpenAI client instance.
     */
    constructor(openaiClient: OpenAI) {
        super();
        this.openaiClient = openaiClient;
    }

    /**
     * Executes an OpenAI built-in tool call.
     * @param name - The tool name (e.g., "web_search", "file_search", "function").
     * @param args - The arguments for the tool.
     * @param meta - Metadata about the tool call.
     * @returns A promise resolving to a ToolCallResult or an error.
     */
    async run(name: string, args: unknown, meta: ToolMeta): Promise<OkErr<ToolCallResult>> {
        // TODO: Replace placeholder with actual OpenAI API client initialization/access.
        // This client might be injected via the constructor or accessed from the 'meta' object.
        // const openaiClient = meta.openaiClient || new SomeOpenAIClient();

        try {
            let toolOutput: unknown;
            let creditsUsed: string = "0"; // Default, should be updated from API response if possible.

            // Hypothetical structure for calling OpenAI tools.
            // The actual implementation depends on the specific "OpenAI Responses API" SDK/client.
            switch (name) {
                case "web_search":
                    // Example: const searchArgs = args as YourWebSearchArgsType;
                    // const response = await openaiClient.performWebSearch(searchArgs, meta.userContext);
                    // toolOutput = response.results;
                    // creditsUsed = response.usageInfo.credits.toString();
                    toolOutput = `Simulated web search for tool "${name}" with args: ${JSON.stringify(args)}`;
                    // Simulate credits used, e.g. based on args complexity or a fixed value
                    creditsUsed = "10";
                    break;

                case "file_search":
                    // Example: const fileArgs = args as YourFileSearchArgsType;
                    // const response = await openaiClient.performFileSearch(fileArgs, meta.userContext);
                    // toolOutput = response.results;
                    // creditsUsed = response.usageInfo.credits.toString();
                    toolOutput = `Simulated file search for tool "${name}" with args: ${JSON.stringify(args)}`;
                    creditsUsed = "5";
                    break;

                // TODO: Add cases for other supported OpenAI tools (e.g., "function", "code_interpreter")
                // case "function":
                //     const functionArgs = args as YourFunctionArgsType;
                //     // ... logic to call the function tool ...
                //     toolOutput = resultOfFunctionCall;
                //     creditsUsed = ...;
                //     break;

                default:
                    return {
                        ok: false,
                        error: {
                            code: "UNKNOWN_OPENAI_TOOL",
                            message: `OpenAI tool named "${name}" is not recognized or supported by this runner.`,
                        },
                    };
            }

            return {
                ok: true,
                data: {
                    output: toolOutput,
                    creditsUsed: creditsUsed,
                },
            };
        } catch (error) {
            // Log the error for debugging purposes
            // console.error(`Error executing OpenAI tool "${name}":`, error);
            return {
                ok: false,
                error: {
                    code: "OPENAI_API_ERROR",
                    message: `An error occurred while executing OpenAI tool "${name}": ${(error instanceof Error) ? error.message : String(error)}`,
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
                        message: response.content?.[0]?.text || `MCP tool "${name}" execution failed`,
                    },
                };
            }
            // TODO: Extract creditsUsed from response if available (default to 0)
            return {
                ok: true,
                data: {
                    output: response,
                    creditsUsed: BigInt(0).toString(),
                },
            };
        } catch (error) {
            return {
                ok: false,
                error: {
                    code: "MCP_TOOL_EXECUTION_ERROR",
                    message: `MCP tool "${name}" execution error: ${(error as Error).message}`,
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
