/**
 * LLM (Large Language Model) integration types
 * These types define the interface for AI model interactions within the execution architecture
 */

/**
 * LLM request configuration
 */
export interface LLMRequest {
    model: string;
    messages: LLMMessage[];
    systemMessage?: string;
    tools?: LLMTool[];
    maxTokens?: number;
    temperature?: number;
    topP?: number;
    stopSequences?: string[];
    presencePenalty?: number;
    frequencyPenalty?: number;
    metadata?: Record<string, unknown>;
}

/**
 * LLM message
 */
export interface LLMMessage {
    role: "system" | "user" | "assistant" | "tool";
    content: string;
    name?: string;
    toolCallId?: string;
}

/**
 * LLM tool definition
 */
export interface LLMTool {
    type: "function";
    function: {
        name: string;
        description: string;
        parameters: Record<string, unknown>;
    };
}

/**
 * LLM response
 */
export interface LLMResponse {
    content: string;
    reasoning?: string;
    confidence: number;
    tokensUsed: number;
    toolCalls?: LLMToolCall[];
    model: string;
    finishReason: "stop" | "length" | "tool_calls" | "content_filter" | "error";
    metadata?: Record<string, unknown>;
}

/**
 * LLM tool call
 */
export interface LLMToolCall {
    id: string;
    type: "function";
    function: {
        name: string;
        arguments: string; // JSON string
    };
}

/**
 * LLM usage statistics
 */
export interface LLMUsage {
    promptTokens: number;
    completionTokens: number;
    totalTokens: number;
    cost?: number;
}

/**
 * LLM provider configuration
 */
export interface LLMProviderConfig {
    provider: string;
    apiKey?: string;
    baseUrl?: string;
    model: string;
    maxTokens?: number;
    temperature?: number;
    timeout?: number;
    rateLimits?: {
        requestsPerMinute?: number;
        tokensPerMinute?: number;
    };
}

/**
 * LLM streaming event types
 */
export type LLMStreamEvent =
    | { type: "start"; metadata?: Record<string, unknown> }
    | { type: "text"; text: string }
    | { type: "reasoning"; reasoning: string }
    | { type: "tool_call"; toolCall: LLMToolCall }
    | { type: "tool_result"; toolCallId: string; result: unknown }
    | { type: "done"; usage: LLMUsage; finishReason: string }
    | { type: "error"; error: string };

/**
 * LLM conversation context
 */
export interface LLMConversationContext {
    conversationId: string;
    userId?: string;
    sessionId?: string;
    history: LLMMessage[];
    maxHistoryLength?: number;
    contextWindow?: number;
    systemPrompt?: string;
    metadata?: Record<string, unknown>;
}

/**
 * LLM model capabilities
 */
export interface LLMModelCapabilities {
    name: string;
    provider: string;
    maxTokens: number;
    supportsToolCalls: boolean;
    supportsStreaming: boolean;
    supportsSystemPrompt: boolean;
    inputCostPer1kTokens: number;
    outputCostPer1kTokens: number;
    contextWindow: number;
    trainingCutoff?: Date;
    languages: string[];
    features: string[];
}

/**
 * LLM error types
 */
export enum LLMErrorType {
    AUTHENTICATION = "AUTHENTICATION",
    RATE_LIMIT = "RATE_LIMIT",
    QUOTA_EXCEEDED = "QUOTA_EXCEEDED",
    MODEL_NOT_FOUND = "MODEL_NOT_FOUND",
    INVALID_REQUEST = "INVALID_REQUEST",
    CONTENT_FILTER = "CONTENT_FILTER",
    TIMEOUT = "TIMEOUT",
    NETWORK_ERROR = "NETWORK_ERROR",
    INTERNAL_ERROR = "INTERNAL_ERROR"
}

/**
 * LLM error
 */
export interface LLMError {
    type: LLMErrorType;
    message: string;
    code?: string;
    details?: Record<string, unknown>;
    retryable: boolean;
    retryAfter?: number; // seconds
}

/**
 * LLM execution options
 */
export interface LLMExecutionOptions {
    timeout?: number;
    retryPolicy?: {
        maxRetries: number;
        backoffStrategy: "linear" | "exponential";
        initialDelay: number;
        maxDelay: number;
    };
    fallbackModels?: string[];
    costLimit?: number;
    qualityThreshold?: number;
}
