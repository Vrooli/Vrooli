import { type LlmServiceId, type MessageState, type ModelInfo, type SessionUser, type ThirdPartyModelInfo, type Tool } from "@vrooli/shared";
import type { ChatCompletionMessageParam, ChatCompletionTool } from "openai/resources/chat/completions.js";
import { type AIServiceErrorType } from "./registry.js";
import { TokenEstimatorType, type EstimateTokensParams, type EstimateTokensResult } from "./tokenTypes.js";
import { TokenEstimationRegistry } from "./tokens.js";

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
    tools: ChatCompletionTool[];
    /** 
     * Allow model to emit several tool calls at once, if supported.
     */
    parallel_tool_calls?: boolean;
    /**
     * If reasoning is supported, the effort level to use.
     */
    reasoningEffort?: "low" | "medium" | "high";
    /** 
     * Maximum reasoning + output tokens allowed per call.
     */
    maxTokens?: number;
    /**
     * The system message to use for the LLM call.
     */
    systemMessage?: string;
    /**
     * User data, e.g. for credit calculation.
     */
    userData?: SessionUser;
    /**
     * Optional AbortSignal to allow cancellation of the stream.
     */
    signal?: AbortSignal;
    /**
     * Service-specific configuration options.
     * This allows passing provider-specific settings without modifying the base interface.
     * Each service can define its own configuration structure.
     */
    serviceConfig?: Record<string, unknown>;
}

/**
 * Messages provided to the model as context, 
 * including the system prompt, previous messages, tool calls, etc.
 * 
 * This type represents the standardized message format used by OpenAI's 
 * Chat Completions API, which is also supported by most other providers
 * (Anthropic, Google, etc.) for compatibility.
 */
export type ContextMessage = ChatCompletionMessageParam;

/**
 * Events yielded by the AIService's generateResponseStreaming method.
 * These are more granular than the router's StreamEvent and lack responseId.
 */
export type ServiceStreamEvent =
    | { type: "text"; content: string }
    | { type: "function_call"; name: string; arguments: unknown; callId: string }
    | { type: "reasoning"; content: string }
    | { type: "done"; cost: number };


export type GetOutputTokenLimitParams = {
    /** The maximum cost (in cents * API_CREDITS_MULTIPLIER) that the response can have */
    maxCredits: bigint;
    /** The model to use for the calculation */
    model: string;
    /** The number of tokens in the input */
    inputTokens: number;
}

export type GetOutputTokenLimitResult = {
    /** The cost of the output tokens */
    cost: number;
    /** Whether the cost is too low */
    isSafe: boolean;
}

export type GetResponseCostParams = {
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
export abstract class AIService<GenerateNameType extends string, ModelInfoType extends ModelInfo | ThirdPartyModelInfo = ModelInfo> {
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
     * Check if the service supports a specific model.
     * This is an optional method that services can implement to filter models.
     * 
     * @param model - The model to check support for
     * @returns Promise<boolean> indicating if the model is supported
     */
    async supportsModel?(model: string): Promise<boolean>;

    /** 
     * Estimate the amount of tokens a string will take up in the model.
     * 
     * This is important for limiting cost and building properly-sized 
     * context windows.
     * 
     * Default implementation uses the TokenEstimationRegistry with tiktoken for accuracy.
     * Services can override this method if they have provider-specific token estimation needs.
     * 
     * @param params - The parameters for the token estimation.
     * @returns The estimated amount of tokens, as well as information 
     * about the encoding and model used.
     */
    estimateTokens(params: EstimateTokensParams): EstimateTokensResult {
        const tokenRegistry = TokenEstimationRegistry.get();
        // Try tiktoken first for accuracy, fallback to default if unavailable
        return tokenRegistry.estimateTokens(TokenEstimatorType.Tiktoken, params);
    }

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
     * @param systemMessage - Optional system message string to prepend.
     * @returns The generated context.
     */
    abstract generateContext(messages: MessageState[], systemMessage?: string): ContextMessage[];
    /**
     * Generates a stream of events from the language model.
     * @param options - The options for generating the response.
     * @returns An async generator of service stream events.
     */
    abstract generateResponseStreaming(options: ResponseStreamOptions): AsyncGenerator<ServiceStreamEvent>;
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
    abstract getModelInfo(): Record<GenerateNameType, ModelInfoType> | Record<string, ModelInfoType>;
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
    /**
     * Gets the names and descriptions of native tools supported by the AI service.
     * These are tools like "file_search", "web_search_preview", etc., not custom functions.
     * The 'name' returned should be the exact type string the AI provider expects for that tool.
     * 
     * @returns An array of objects, each with 'name' and an optional 'description'.
     */
    abstract getNativeToolCapabilities(): Pick<Tool, "name" | "description">[];
    /**
     * Checks if the AI service is currently healthy and available.
     * 
     * @returns true if the service is healthy and can accept requests, false otherwise
     */
    abstract isHealthy(): boolean | Promise<boolean>;
}

