/**
 * AI Mock Types
 * 
 * Comprehensive type definitions for AI/LLM mock fixtures.
 * These types ensure type safety and consistency across all mock implementations.
 */

import {
    LLMErrorType} from "@vrooli/shared";
import type { 
    LLMRequest, 
    LLMResponse, 
    LLMMessage, 
    LLMToolCall,
    LLMStreamEvent,
    LLMError,
    LLMUsage,
    LLMModelCapabilities,
} from "@vrooli/shared";
import type { ToolCallResult } from "../../../../services/execution/integration/llmIntegrationService.js";

/**
 * Mock response configuration
 */
export interface AIMockConfig {
    /** Base response properties */
    content?: string;
    reasoning?: string;
    confidence?: number;
    model?: string;
    finishReason?: LLMResponse["finishReason"];
    tokensUsed?: number;
    
    /** Tool calling */
    toolCalls?: AIMockToolCall[];
    
    /** Streaming configuration */
    streaming?: StreamingConfig;
    
    /** Error simulation */
    error?: ErrorConfig;
    
    /** Behavior modifiers */
    delay?: number;
    metadata?: Record<string, unknown>;
}

/**
 * Mock tool call definition
 */
export interface AIMockToolCall {
    name: string;
    arguments: Record<string, unknown>;
    result?: unknown;
    error?: string;
    delay?: number;
}

/**
 * Streaming response configuration
 */
export interface StreamingConfig {
    chunks: string[] | StreamChunk[];
    chunkDelay?: number | number[];
    includeReasoning?: boolean;
    reasoningChunks?: string[];
    simulateTyping?: boolean;
    interruptAt?: number;
    error?: ErrorConfig;
}

/**
 * Individual stream chunk
 */
export interface StreamChunk {
    type: "text" | "reasoning" | "tool_call";
    content: string;
    delay?: number;
}

/**
 * Error simulation configuration
 */
export interface ErrorConfig {
    type: LLMErrorType;
    message?: string;
    code?: string;
    details?: Record<string, unknown>;
    retryable?: boolean;
    retryAfter?: number;
    occurAfter?: number; // Fail after N chunks/characters
}

/**
 * Mock behavior definition
 */
export interface MockBehavior {
    /** Pattern to match against input */
    pattern?: RegExp | ((request: LLMRequest) => boolean);
    
    /** Response generator */
    response: AIMockConfig | ((request: LLMRequest) => AIMockConfig);
    
    /** Priority for pattern matching */
    priority?: number;
    
    /** Maximum uses before exhaustion */
    maxUses?: number;
    
    /** Behavior metadata */
    metadata?: {
        name?: string;
        description?: string;
        tags?: string[];
    };
}

/**
 * Stateful mock configuration
 */
export interface StatefulMockConfig<TState = any> {
    initialState: TState;
    behavior: (request: LLMRequest, state: TState) => AIMockConfig;
    stateUpdater?: (request: LLMRequest, response: LLMResponse, state: TState) => TState;
}

/**
 * Dynamic mock configuration
 */
export interface DynamicMockConfig {
    matcher: (request: LLMRequest) => AIMockConfig | null;
    fallback?: AIMockConfig;
}

/**
 * Mock response factory result
 */
export interface MockFactoryResult {
    response: LLMResponse;
    stream?: AsyncGenerator<LLMStreamEvent>;
    usage?: LLMUsage;
    raw?: AIMockConfig;
}

/**
 * Validation result for mock responses
 */
export interface MockValidationResult {
    valid: boolean;
    errors?: string[];
    warnings?: string[];
    suggestions?: string[];
}

/**
 * Mock registry entry
 */
export interface MockRegistryEntry {
    id: string;
    behavior: MockBehavior;
    uses: number;
    created: Date;
    lastUsed?: Date;
}

/**
 * Cost tracking for mock responses
 */
export interface MockCostTracking {
    totalCost: number;
    inputTokens: number;
    outputTokens: number;
    requests: number;
    breakdown: Array<{
        model: string;
        inputTokens: number;
        outputTokens: number;
        cost: number;
        timestamp: Date;
    }>;
}

/**
 * Mock service interface
 */
export interface AIMockService {
    /** Execute a mock request */
    execute(request: LLMRequest): Promise<MockFactoryResult>;
    
    /** Register a mock behavior */
    registerBehavior(id: string, behavior: MockBehavior): void;
    
    /** Clear all behaviors */
    clearBehaviors(): void;
    
    /** Get usage statistics */
    getStats(): {
        totalRequests: number;
        behaviorHits: Record<string, number>;
        averageLatency: number;
    };
}

/**
 * Emergent behavior mock configuration
 */
export interface EmergentMockBehavior {
    capability: string;
    evolution: Array<{
        iteration: number;
        response: AIMockConfig;
        threshold?: number;
    }>;
    regressionChance?: number;
}

/**
 * Test helper options
 */
export interface MockTestOptions {
    /** Enable debug logging */
    debug?: boolean;
    
    /** Track costs */
    trackCosts?: boolean;
    
    /** Validate all responses */
    validateResponses?: boolean;
    
    /** Default model to use */
    defaultModel?: string;
    
    /** Global delay for all responses */
    globalDelay?: number;
    
    /** Capture all requests/responses */
    captureInteractions?: boolean;
}

/**
 * Conversation mock state
 */
export interface ConversationMockState {
    messages: LLMMessage[];
    context: Record<string, unknown>;
    turnCount: number;
    totalTokens: number;
}

/**
 * Mock scenario definition
 */
export interface MockScenario {
    name: string;
    description: string;
    steps: Array<{
        input: Partial<LLMRequest>;
        expectedBehavior: string;
        mockConfig: AIMockConfig;
        assertions?: (response: LLMResponse) => void;
    }>;
}

/**
 * Type guards
 */
export const isStreamingConfig = (config: any): config is StreamingConfig => {
    return config && Array.isArray(config.chunks);
};

export const isErrorConfig = (config: any): config is ErrorConfig => {
    return config && config.type && Object.values(LLMErrorType).includes(config.type);
};

export const isToolCallResponse = (response: LLMResponse): response is LLMResponse & { toolCalls: NonNullable<LLMResponse["toolCalls"]> } => {
    return response.toolCalls !== undefined && response.toolCalls.length > 0;
};

/**
 * Constants
 */
export const DEFAULT_MOCK_CONFIG: Partial<AIMockConfig> = {
    model: "gpt-4o-mini",
    confidence: 0.85,
    finishReason: "stop",
    tokensUsed: 100,
};

export const MOCK_MODELS: LLMModelCapabilities[] = [
    {
        name: "gpt-4o",
        provider: "openai",
        maxTokens: 128000,
        supportsToolCalls: true,
        supportsStreaming: true,
        supportsSystemPrompt: true,
        inputCostPer1kTokens: 0.0025,
        outputCostPer1kTokens: 0.01,
        contextWindow: 128000,
        languages: ["en", "es", "fr", "de", "ja", "zh"],
        features: ["reasoning", "tool-use", "vision"],
    },
    {
        name: "gpt-4o-mini",
        provider: "openai",
        maxTokens: 128000,
        supportsToolCalls: true,
        supportsStreaming: true,
        supportsSystemPrompt: true,
        inputCostPer1kTokens: 0.00015,
        outputCostPer1kTokens: 0.0006,
        contextWindow: 128000,
        languages: ["en", "es", "fr", "de", "ja", "zh"],
        features: ["reasoning", "tool-use"],
    },
    {
        name: "o1-mini",
        provider: "openai",
        maxTokens: 65536,
        supportsToolCalls: false,
        supportsStreaming: false,
        supportsSystemPrompt: false,
        inputCostPer1kTokens: 0.003,
        outputCostPer1kTokens: 0.012,
        contextWindow: 128000,
        languages: ["en"],
        features: ["deep-reasoning", "math", "coding"],
    },
];
