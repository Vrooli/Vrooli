import { LlmServiceId, ModelInfo, OpenAIModel, SessionUser, openAIServiceInfo } from "@local/shared";
import { ToolFunctionCall } from "@local/shared/src/shape/configs/message.js";
import OpenAI from "openai";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { AIServiceErrorType, AIServiceRegistry } from "./registry.js";
import { TokenEstimationRegistry } from "./tokens.js";
import { MessageState } from "./types.js";
import { WorldModel, WorldModelConfig } from "./worldModel.js";

export type ResponseStreamOptions = {
    /** 
     * Model name (provider‑specific). 
     * 
     * This does not have to be a valid name yet. The service provider will convert it to a valid name later.
     */
    model: string;
    /** 
     * Optional reference to continue a threaded conversation on provider side. 
     * 
     * NOTE: This is not recommended. Building our own context allows us to be smart 
     * with what's being sent to the model. For example, we could filter out tool call results 
     * like old DefineTool results, which are very long and only needed for the immediate next response.
     */
    previous_response_id?: string;
    /** 
     * Messages to include as context. These will be converted to the correct format by the service provider.
     * 
     * This shape mirrors how we fetch messages from the database/cache.
     * 
     * NOTE: If you're relying on previous_response_id, you'll still need to provide at least the last user message.
     */
    input: MessageState[];
    /** 
     * OpenAI Tool definitions. Other services will need to adapt from this shape.
     * 
     * For our custom tools, we use the FunctionTool type.
     */
    tools: OpenAI.Responses.Tool[];
    /** 
     * Allow model to emit several tool calls at once, if supported.
     */
    parallel_tool_calls?: boolean;
    /**
     * If reasoning is supported, the effort level to use.
     */
    reasoningEffort?: "low" | "medium" | "high";
    /** 
     * User profile => plan, credit, localisation.
     */
    userData: SessionUser;
    /** 
     * Maximum reasoning + output tokens allowed per call.
     */
    maxTokens?: number;
    /**
     * Information about the conversation/world.
     */
    world: WorldModelConfig;
}

/**
 * Messages provided to the model as context, 
 * including the system prompt, previous messages, tool calls, etc.
 */
type ContextMessage = OpenAI.Responses.ResponseInputItem;

/**
 * Tool definition for function calling capabilities
 */
interface ExecutableTool {
    name: string;
    description: string;
    parameters: Record<string, unknown>; // JSON schema for parameters
    execute: (args: Record<string, unknown>) => Promise<unknown>;
}

type StreamEvent =
    | { type: "text"; content: string }
    | { type: "done"; cost: number };

/**
 * Method for token estimation
 */
export enum TokenEstimatorType {
    Default = "Default",
    Tiktoken = "Tiktoken",
}

export type EstimateTokensParams = {
    /** The requested model to base token logic on */
    aiModel: string;
    /** The text to estimate tokens for */
    text: string;
}

export type EstimateTokensResult = {
    /** The encoding used for the token estimation */
    encoding: string;
    /** The name of the token estimation model used (if requested one was invalid/incomplete) */
    estimationModel: TokenEstimatorType;
    /** The estimated amount of tokens calculated by this method/encoding pair */
    tokens: number;
}

type GetOutputTokenLimitParams = {
    /** The maximum cost (in cents * API_CREDITS_MULTIPLIER) that the response can have */
    maxCredits: bigint;
    /** The model to use for the calculation */
    model: string;
    /** The number of tokens in the input */
    inputTokens: number;
}

type GetOutputTokenLimitResult = {
    /** The cost of the output tokens */
    cost: number;
    /** Whether the cost is too low */
    isSafe: boolean;
}

type GetResponseCostParams = {
    model: string;
    usage: { input: number, output: number };
}

type AIServiceFeatureFlags = {
    supportsStatefulConversations: boolean;
}

/**
 * Abstract class for an AI service. 
 * 
 * This allows us to use different AI services interchangeably.
 */
export abstract class AIService<GenerateNameType extends string> {
    /** 
     * Identifier for the service.
     * 
     * Useful for our router (read: service availability/registry) to 
     * identify the service.
     */
    abstract __id: LlmServiceId;
    /** 
     * Feature flags for the service.
     * 
     * This might only be used internally by the service, but it 
     * is a nice way to standardize expected features.
     */
    abstract featureFlags: AIServiceFeatureFlags;
    /**
     * The default model to use if not specified.
     */
    abstract defaultModel: GenerateNameType;

    /** 
     * Estimate the amount of tokens a string will take up in the model.
     * 
     * This is important for limiting cost and building properly-sized 
     * context windows.
     * 
     * @param params - The parameters for the token estimation.
     * @returns The estimated amount of tokens, as well as information 
     * about the encoding and model used.
     */
    abstract estimateTokens(params: EstimateTokensParams): EstimateTokensResult;

    /** 
     * Generates context prompt from previous messages. 
     * 
     * We group tool use with messages, but the output expects them to be separate. 
     * For example, our history might look something like this (not the exact types):
     * 
     * [
     *    { role: "user", content: "What's the weather in Tokyo?" },
     *    { role: "assistant", content: "The weather in Tokyo is sunny.", tool_calls: [{ id: "1", type: "function", function: { name: "get_weather", arguments: "Tokyo" }, result: "Sunny" }] },
     * ]
     * 
     * The output will be:
     * [
     *    // System prompt added if not present
     *    { role: "system", content: "You are a helpful assistant." },
     *    // User message
     *    { role: "user", content: "What's the weather in Tokyo?" },
     *    // Assistant message without tool calls
     *    { role: "assistant", content: "The weather in Tokyo is sunny." },
     *    // Tool message
     *    { role: "tool", function: { name: "get_weather", arguments: "Tokyo" } },
     *    // Tool result
     *    { role: "tool", content: "Sunny" },
     * ]
     * 
     * Because of this, do not expect the output length to match the input length.
     * 
     * @param messages - The input messages to generate context for.
     * @param globalContext - The world model to use for the context.
     * @returns The generated context.
     */
    abstract generateContext(messages: MessageState[], globalContext?: WorldModelConfig): ContextMessage[];
    /**
     * Generate a message response, streaming.
     * The service is responsible for handling tool calls and yielding appropriate events.
     * The router will add 'responseId' to the events.
     * 'callId' for function calls must be generated by the service.
     * 
     * @param options - The options for the response generation.
     * @returns The generated response
     */
    abstract generateResponseStreaming(options: ResponseStreamOptions): AsyncGenerator<Omit<StreamEvent, "responseId">>;
    /** 
     * Get the context size (read: how much input data can be stored in the model).
     * 
     * @param requestedModel - The model to get the context size for.
     * @returns the context size of the model 
     */
    abstract getContextSize(requestedModel?: string | null): number;
    /**
     * Get information about the available models.
     * 
     * @returns Information about the available models 
     */
    abstract getModelInfo(): Record<GenerateNameType, ModelInfo>;
    /** 
     * Get the maximum number of output tokens for a model.
     * 
     * @param requestedModel - The model to get the max output tokens for.
     * @returns the max output window for the model 
     */
    abstract getMaxOutputTokens(requestedModel?: string | null): number;
    /**
     * Calculates the maximum number of tokens that can be output by the model 
     * to stay under the cost limit.
     * 
     * @param maxCredits The maximum cost (in cents * 1_000_000) that the response can have
     * @param model The model to use for the calculation
     * @param inputTokens The number of tokens in the input
     * @returns The maximum number of tokens that can be output, or 0 if the cost is too low 
     * (i.e. the input already exceeds the cost limit)
     */
    abstract getMaxOutputTokensRestrained(params: GetOutputTokenLimitParams): number;
    /** 
     * Calculates the api credits used by the generation of an LLM response, 
     * based on the model's input token and output token costs (in cents per 1_000_000 tokens).
     * 
     * NOTE: Instead of using decimals, the cost is in cents multiplied by 1_000_000. This 
     * can be normalized in the UI so the number isn't overwhelming. 
     * 
     * @returns the total cost of the response
     */
    abstract getResponseCost(params: GetResponseCostParams): number;
    /** 
     * @returns the estimation model and encoding for the model 
     */
    abstract getEstimationInfo(model?: string | null): Pick<EstimateTokensResult, "estimationModel" | "encoding">;
    /**
     * Convert a preferred model to an available one.
     * 
     * @param model - The model to convert.
     * @returns The converted model.
     */
    abstract getModel(model?: string | null): GenerateNameType;
    /**
     *  Converts error received by service to a standardized type
     * 
     * @param error - The error to convert.
     * @returns The converted error.
     */
    abstract getErrorType(error: unknown): AIServiceErrorType;
    /** 
     * Checks if the input contains potentially harmful content 
     * 
     * @param input - The input to check.
     * @returns true if the input is safe, false otherwise
     */
    abstract safeInputCheck(input: string): Promise<GetOutputTokenLimitResult>;
}

/**
 * Default output token calculator to determine available output tokens based on cost limit.
 * 
 * NOTE: input and output costs should be in cents per API_CREDITS_MULTIPLIER tokens. 
 * Since credits are stored with this multiplier already, it should cancel out (meaning 
 * we won't be using API_CREDITS_MULTIPLIER in the calculation).
 */
function getDefaultMaxOutputTokensRestrained<GenerateNameType extends string>(
    { maxCredits, model, inputTokens }: GetOutputTokenLimitParams,
    service: AIService<GenerateNameType>,
) {
    const modelToUse = service.getModel(model);
    const modelInfo = service.getModelInfo()[modelToUse];

    if (!modelInfo || !modelInfo.inputCost || !modelInfo.outputCost) {
        throw new Error(`Model "${model}" (converted to ${modelToUse}) not found in cost records`);
    }

    const inputCost = BigInt(modelInfo.inputCost * inputTokens);
    const remainingCredits = BigInt(maxCredits) - BigInt(inputCost);

    if (remainingCredits <= 0) {
        return 0;
    }

    const maxOutputTokensRestrained = remainingCredits / BigInt(Math.floor(modelInfo.outputCost));
    return Math.min(Math.max(0, Number(maxOutputTokensRestrained)), service.getMaxOutputTokens(model) - inputTokens);
}

export class OpenAIService extends AIService<OpenAIModel> {
    private readonly client: OpenAI;
    readonly defaultModel: OpenAIModel;
    /** Identifier for the service */
    __id: LlmServiceId = LlmServiceId.OpenAI;

    /** Feature flags for the service */
    featureFlags = {
        supportsStatefulConversations: true,
    };

    constructor(options?: {
        apiKey?: string,
        defaultModel?: OpenAIModel,
    }) {
        super();
        this.client = new OpenAI({ apiKey: options?.apiKey ?? process.env.OPENAI_API_KEY ?? "" });
        this.defaultModel = options?.defaultModel ?? OpenAIModel.Gpt4o_Mini;
    }

    /** Helper method to process tool calls and add them to the context */
    private processToolCall(contextMessages: ContextMessage[], toolCall: ToolFunctionCall): void {
        // Add function call from the assistant
        if (toolCall.function) {
            // Create a ResponseFunctionToolCall item to represent the assistant's decision to call the function.
            // This 'id' and 'call_id' will be the same, representing the unique ID for this specific tool invocation.
            const functionCallItem: OpenAI.Responses.ResponseFunctionToolCall = {
                id: toolCall.id, // Unique ID for this specific function call item in the input sequence
                call_id: toolCall.id, // The ID for this specific call, to be referenced by the output
                name: toolCall.function.name,
                arguments: toolCall.function.arguments, // This is a string
                type: "function_call", // Specifies the type of this item
            };
            contextMessages.push(functionCallItem);

            // Add tool response if available
            if (toolCall.result !== undefined) {
                // Create a ResponseFunctionToolCallOutputItem to provide the result of the function.
                const functionOutputItem: OpenAI.Responses.ResponseFunctionToolCallOutputItem = {
                    id: `${toolCall.id}_output`, // A unique ID for this output item
                    call_id: toolCall.id, // This references the call_id of the ResponseFunctionToolCall above
                    output: String(toolCall.result),
                    type: "function_call_output", // Specifies the type of this item
                };
                contextMessages.push(functionOutputItem);
            }
        }
    }

    /**
     * Generates an array of context messages suitable for the OpenAI Responses API.
     *
     * This method translates an internal 'MessageState[]' into the 'ContextMessage[]'
     * (OpenAI.Responses.ResponseInputItem[]) format required by 'client.responses.create()'.
     *
     * Key behaviors and differences from older Chat Completions API conventions:
     * - **System Prompt**: Ensures a system prompt is always the first message. If not present
     *   in the input 'messages', it generates one using the 'WorldModel'.
     * - **Tool Call Handling**: Assistant messages with tool calls are expanded.
     *   An assistant message with text and a tool call like:
     *     { role: "assistant", text: "Okay, I will call the tool.", config: { toolCalls: [{ id: "t1", function: { name: "X", args: "{}"}, result: "Y" }] } }
     *   is translated into a sequence:
     *     1. { role: "assistant", content: "Okay, I will call the tool." }
     *     2. { type: "function_call", id: "t1", call_id: "t1", name: "X", arguments: "{}" }
     *     3. { type: "function_call_output", id: "t1_output", call_id: "t1", output: "Y" }
     *   This allows the Responses API to understand the full lifecycle of an assistant's turn,
     *   including its decision to use tools and the results of those tools.
     * - **Role Alternation**: Unlike the stricter user/assistant/user/assistant alternation
     *   often enforced by the Chat Completions API, the Responses API, when used with tools,
     *   accommodates sequences like 'assistant' (text) -> 'function_call' -> 'function_call_output'
     *   as part of a single logical assistant turn. The model can then continue generating
     *   assistant output based on the tool results.
     * - **Last Message**: The context doesn't strictly need to end with a 'user' message if the
     *   assistant is in the middle of a turn (e.g., after a tool call output). The API
     *   understands this and will prompt the model to continue the assistant's response.
     *
     * The primary responsibility for ensuring a logically sound conversational history (e.g.,
     * for agent-to-agent communication where one agent's final output becomes the 'user'
     * input for another) lies in how the input 'MessageState[]' is constructed before
     * being passed to this method. This method focuses on faithful translation to the
     * 'OpenAI.Responses.ResponseInputItem[]' format.
     *
     * @param messages - The input messages to generate context for.
     * @param globalContext - The world model to use for the context if a system prompt is needed.
     * @returns The generated context messages for the OpenAI Responses API.
     */
    generateContext(messages: MessageState[], globalContext?: WorldModelConfig): ContextMessage[] {
        const contextMessages: ContextMessage[] = [];

        // Check if system message exists
        const hasSystemMessage = messages.some(msg => msg.config.role === "system");

        // Add system message at the beginning if not present
        if (!hasSystemMessage) {
            contextMessages.push({
                role: "system",
                content: (new WorldModel(globalContext)).serialize(),
            });
        }

        // Process each message in order
        for (const message of messages) {
            const role = message.config.role || "user";

            // Skip if role is not valid
            if (role !== "system" && role !== "user" && role !== "assistant") {
                continue;
            }

            // Add the base message content
            contextMessages.push({
                role: role as "system" | "user" | "assistant",
                content: message.text,
            });

            // Handle tool calls if present
            if (role === "assistant" && message.config.toolCalls?.length) {
                for (const toolCall of message.config.toolCalls) {
                    // Process tool call
                    this.processToolCall(contextMessages, toolCall);
                }
            }
        }

        return contextMessages;
    }

    estimateTokens(params: EstimateTokensParams) {
        return TokenEstimationRegistry.get().estimateTokens(TokenEstimatorType.Tiktoken, params);
    }

    getContextSize(requestedModel?: string | null) {
        const model = this.getModel(requestedModel);
        return openAIServiceInfo.models[model].contextWindow;
    }

    getModelInfo() {
        return openAIServiceInfo.models;
    }

    getMaxOutputTokens(requestedModel?: string | null | undefined): number {
        const model = this.getModel(requestedModel);
        return openAIServiceInfo.models[model].maxOutputTokens;
    }

    getMaxOutputTokensRestrained(params: GetOutputTokenLimitParams): number {
        return getDefaultMaxOutputTokensRestrained(params, this);
    }

    getResponseCost(params: GetResponseCostParams) {
        return getDefaultResponseCost(params, this);
    }

    /**
     * Main streaming generator.
     * 
     * Uses the Responses API to stream the response. 
     * DO NOT use the ChatCompletions API, as it is outdated!
     */
    async *generateResponseStreaming(opts: ResponseStreamOptions): AsyncGenerator<StreamEvent> {
        const {
            model,
            input,
            tools = [],
            userData,
            world,
            maxTokens,
            previous_response_id,
            reasoningEffort,
        } = opts;

        // Get the model info
        const modelInfo = this.getModelInfo()[this.getModel(model)];
        const canReason = modelInfo.supportsReasoning === true;

        /* ------------- 1  Build create() params ------------- */
        const createParams: OpenAI.Responses.ResponseCreateParamsStreaming = {
            model: modelInfo.name,
            reasoning: canReason ? {
                effort: reasoningEffort,
                summary: "concise",
            } : undefined,
            max_output_tokens: maxTokens, // Includes reasoning tokens
            stream: true,
            input: [], // Set later
        };

        // Server-side state or full context
        if (previous_response_id) {
            createParams.previous_response_id = previous_response_id;
            createParams.input = [{ role: "user", content: input[input.length - 1]?.text ?? "" }];
        } else {
            createParams.input = this.generateContext(input, world);
        }

        // Optional custom tools
        if (tools.length) {
            createParams.tools = tools;
        }

        /* ------------- 2  Issue request & iterate events ------------- */
        const stream = await this.client.responses.create(createParams);

        let currentText = "";
        let totalCost = 0;
        let responseId: string | undefined;
        const pendingToolCalls: Array<{ name: string; callId: string; arguments: Record<string, unknown> }> = [];

        for await (const ev of stream) {
            switch (ev.type) {
                case "response.created":
                    responseId = ev.response.id;
                    break;

                case "response.output_text.delta":
                    currentText += ev.delta;
                    yield { type: "text", content: ev.delta };
                    break;

                case "response.output_item.added":
                    if (ev.item.type === "function_call") {
                        // ev.item is OpenAI.Responses.ResponseFunctionToolCallItem
                        // ev.item.arguments is a string, needs parsing
                        let parsedArgs: Record<string, unknown> = {};
                        try {
                            if (ev.item.arguments) {
                                parsedArgs = JSON.parse(ev.item.arguments);
                            }
                        } catch (e) {
                            logger.error("Failed to parse tool call arguments", { name: ev.item.name, args: ev.item.arguments, error: e });
                            // Decide how to handle: skip tool, error, or call with empty args?
                            // For now, will proceed with empty args if parsing fails.
                        }
                        pendingToolCalls.push({
                            name: ev.item.name!,
                            callId: ev.item.id!,
                            arguments: parsedArgs,
                        });
                    }
                    break;

                case "response.completed":
                    totalCost += this.estimateCost(ev.response.usage);
                    break;

                case "error":
                    throw new Error(ev.message);

                default:
                    // ignore other rare events (rate_limits.updated, etc.)
                    break;
            }
        }

        /* ------------- 3  If tools were requested, recurse ------------- */
        if (pendingToolCalls.length) {
            // Assume 'this.getRegisteredExecutableTools()' provides the List of our internal 'ExecutableTool' type
            // This method needs to be implemented or these tools need to be passed via constructor.
            const executableTools = this.getRegisteredExecutableTools();
            const outputs = await this.handleToolCalls(pendingToolCalls, executableTools);

            const followUp: ResponseStreamOptions = {
                model,
                input: input, // Simplified for now, TODO: Construct proper followup input with assistant response and tool results
                tools: opts.tools,
                userData,
                world,
                maxTokens,
                // Only pass previous_response_id if we initially used it. 
                // This is because previous_response_id stores the conversation with OpenAI, 
                // but we typically want to generate the context ourselves.
                previous_response_id: previous_response_id ? responseId : undefined,
            };
            yield* this.generateResponseStreaming(followUp);
            return;
        }

        /* ------------- 4  Finish ------------- */
        yield { type: "done", cost: totalCost };
    }

    // Placeholder for fetching executable tools.
    // This needs to be implemented, e.g., by injecting a tool registry/service.
    private getRegisteredExecutableTools(): ExecutableTool[] {
        // TODO: Implement this method to return the actual list of executable tools.
        // This might involve fetching from a registry or a class member populated during construction.
        // Example: return this.toolRegistry.getAll();
        logger.warn("getRegisteredExecutableTools is a placeholder and needs implementation.");
        return [];
    }

    /* ------------------------------------------------------------ */
    private async handleToolCalls(
        calls: Array<{ name: string; callId: string; arguments: Record<string, unknown> }>,
        executableTools: ExecutableTool[],
    ): Promise<unknown[]> {
        return Promise.all(
            calls.map(async ({ name, callId, arguments: args }) => {
                const tool = executableTools.find((t) => t.name === name);
                if (!tool) {
                    // It's important to provide a result for each callId back to OpenAI
                    // So, instead of just a string, we might need to conform to OpenAI's tool result format.
                    // For now, returning a string indicating the error.
                    logger.error("Executable tool not found", { name, callId });
                    return `Tool ${name} not found`;
                }
                try {
                    return await tool.execute(args);
                } catch (error) {
                    logger.error("Tool execution failed", { name, callId, error });
                    return `Error executing tool ${name}: ${error instanceof Error ? error.message : String(error)}`;
                }
            }),
        );
    }

    private estimateCost(metrics?: OpenAI.Responses.ResponseUsage): number {
        if (!metrics) return 0;
        const INPUT_TOKEN_COST_PER_1K = 0.01;  // Cost per 1,000 input tokens
        const OUTPUT_TOKEN_COST_PER_1K = 0.03;  // Cost per 1,000 output tokens
        const TOKENS_PER_1K = 1000;
        return (
            (metrics.input_tokens / TOKENS_PER_1K) * INPUT_TOKEN_COST_PER_1K +
            (metrics.output_tokens / TOKENS_PER_1K) * OUTPUT_TOKEN_COST_PER_1K
        );
    }

    getEstimationInfo(model?: string | null): Pick<EstimateTokensResult, "estimationModel" | "encoding"> {
        return TokenEstimationRegistry.get().getEstimationInfo(TokenEstimatorType.Tiktoken, model);
    }

    getModel(model?: string | null) {
        if (typeof model !== "string") return this.defaultModel;
        if (model.startsWith("gpt-4o-mini")) return OpenAIModel.Gpt4o_Mini;
        if (model.startsWith("gpt-4o")) return OpenAIModel.Gpt4o;
        if (model.startsWith("gpt-4-turbo")) return OpenAIModel.Gpt4_Turbo;
        if (model.startsWith("gpt-4")) return OpenAIModel.Gpt4;
        if (model.startsWith("o1-mini")) return OpenAIModel.o1_Mini;
        if (model.startsWith("o1-preview")) return OpenAIModel.o1_Preview;
        return this.defaultModel;
    }

    getErrorType(error: unknown): AIServiceErrorType {
        if (!error || typeof error !== "object" || typeof (error as { error?: unknown }).error !== "object") return AIServiceErrorType.ApiError;
        const { type, code } = (error as { error: { type?: string, code?: string } }).error;

        if (code === "invalid_api_key") {
            return AIServiceErrorType.Authentication;
        }

        switch (type) {
            case "invalid_request_error":
            case "not_found_error":
            case "tokens_exceeded_error":
                return AIServiceErrorType.InvalidRequest;
            case "authentication_error":
            case "permission_error":
                return AIServiceErrorType.Authentication;
            case "rate_limit_error":
                return AIServiceErrorType.RateLimit;
            case "server_error":
                return AIServiceErrorType.ApiError;
            default:
                return AIServiceErrorType.ApiError;
        }
    }

    async safeInputCheck(input: string): Promise<GetOutputTokenLimitResult> {
        try {
            const response = await this.client.moderations.create({
                input,
            });

            // The response contains an array of results, but we're only checking one input
            const result = response.results[0];
            // If the content is flagged, it's not safe
            const isSafe = !result.flagged;
            // The moderation check for OpenAI is free
            const cost = 0;
            return { cost, isSafe };
        } catch (error) {
            const trace = "0606";
            const errorType = this.getErrorType(error);
            AIServiceRegistry.get().updateServiceState(this.__id, errorType);
            logger.error("Failed to call OpenAI moderation", { trace, error, errorType });
            // Instead of treating service errors as unsafe content,
            // throw the error to allow fallback mechanisms to work
            throw new CustomError(trace, "InternalError", { error, errorType });
        }
    }
}

