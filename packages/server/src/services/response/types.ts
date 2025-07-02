/**
 * Response Service Types
 * 
 * Types specific to the ResponseService that bridge between the unified
 * execution types and the existing conversation types.
 */

import type {
    ChatMessage, ExecutionError,
    ExecutionResourceUsage,
    ResponseContext,
    ToolCall,
} from "@vrooli/shared";
import type { Tool } from "../mcp/types.js";
import type { ContextBuilder as ConversationContextBuilder } from "./contextBuilder.js";
import type { LlmRouter } from "./router.js";
import type { ToolRunner } from "./toolRunner.js";

/**
 * Configuration for ResponseService
 */
export interface ResponseServiceConfig {
    /** Default timeout for response generation in milliseconds */
    defaultTimeoutMs: number;

    /** Default maximum tokens per response */
    defaultMaxTokens: number;

    /** Default temperature for LLM sampling */
    defaultTemperature: number;

    /** Whether to enable detailed logging */
    enableDetailedLogging: boolean;
}

/**
 * Parameters for generating a single bot response
 * This is the main input interface for ResponseService
 */
export interface ResponseGenerationParams {
    /** Unified context containing all necessary information */
    context: ResponseContext;

    /** Abort signal for cancellation */
    abortSignal?: AbortSignal;

    /** Override default service configuration */
    overrides?: Partial<ResponseServiceConfig>;
}

/**
 * Result from executing a single tool call
 */
export interface ToolExecutionResult {
    /** Whether the tool execution succeeded */
    success: boolean;

    /** The output from the tool (if successful) */
    output?: unknown;

    /** Error information (if failed) */
    error?: ExecutionError;

    /** Resources consumed by this tool call */
    resourcesUsed: ExecutionResourceUsage;

    /** Duration of tool execution in milliseconds */
    duration: number;

    /** The original tool call that was executed */
    originalCall: ToolCall;
}

/**
 * Result from a single LLM API call
 */
export interface LlmCallResult {
    /** Whether the LLM call succeeded */
    success: boolean;

    /** The generated message (if successful) */
    message?: ChatMessage;

    /** Tool calls requested by the LLM (if any) */
    toolCalls?: ToolCall[];

    /** Error information (if failed) */
    error?: ExecutionError;

    /** Resources consumed by this LLM call */
    resourcesUsed: ExecutionResourceUsage;

    /** Duration of LLM call in milliseconds */
    duration: number;

    /** LLM-specific metadata */
    metadata?: {
        /** Model used for generation */
        model?: string;

        /** Token counts */
        tokens?: {
            input?: number;
            output?: number;
            total?: number;
        };

        /** AI confidence score */
        confidence?: number;

        /** Finish reason */
        finishReason?: string;
    };
}

/**
 * Internal state during response generation
 * Used to track progress and accumulate results
 */
export interface ResponseGenerationState {
    /** All messages generated so far */
    messages: ChatMessage[];

    /** All tool calls made so far */
    toolCalls: ToolCall[];

    /** Accumulated resource usage */
    totalResourcesUsed: ExecutionResourceUsage;

    /** Whether the response loop should continue */
    shouldContinue: boolean;

    /** Current loop iteration count */
    iterationCount: number;

    /** Start time for duration tracking */
    startTime: number;

    /** Any error that occurred */
    error?: ExecutionError;
}

/**
 * Bridge types for compatibility with existing conversation system
 */

/**
 * Convert unified ResponseContext to existing conversation types
 */
export interface ConversationContextBridge {
    /** Convert ResponseContext to parameters for existing ContextBuilder */
    toContextBuilderParams(context: ResponseContext): {
        conversationHistory: ChatMessage[];
        botConfig: unknown;
        maxTokens?: number;
    };

    /** Convert ResponseContext to parameters for existing LlmRouter */
    toLlmRouterParams(context: ResponseContext, systemMessage: string, tools: Tool[]): {
        model: string;
        messages: ChatMessage[];
        tools: Tool[];
        temperature?: number;
        userId: string;
    };

    /** Convert ResponseContext to parameters for existing ToolRunner */
    toToolRunnerMeta(context: ResponseContext): {
        conversationId?: string;
        callerBotId?: string;
        sessionUser?: unknown;
    };
}

/**
 * Factory interface for creating ResponseService with proper dependencies
 */
export interface ResponseServiceDependencies {
    /** LLM router for making API calls */
    llmRouter: LlmRouter;

    /** Tool runner for executing tools */
    toolRunner: ToolRunner;

    /** Context builder for conversation history */
    contextBuilder: ConversationContextBuilder;

    /** Service configuration */
    config?: Partial<ResponseServiceConfig>;
}
