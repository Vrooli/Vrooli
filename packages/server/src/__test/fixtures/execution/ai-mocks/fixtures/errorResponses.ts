/**
 * Error Response Fixtures
 * 
 * Pre-defined error response patterns for testing error handling.
 */

import { LLMErrorType } from "@vrooli/shared";
import type { ErrorConfig } from "../types.js";

/**
 * Rate limit error
 */
export const rateLimit = (retryAfter = 60): ErrorConfig => ({
    type: LLMErrorType.RATE_LIMIT,
    message: "Rate limit exceeded. Please try again later.",
    code: "rate_limit_exceeded",
    details: {
        limit: 100,
        used: 100,
        resetTime: new Date(Date.now() + retryAfter * 1000).toISOString(),
    },
    retryable: true,
    retryAfter,
});

/**
 * Authentication error
 */
export const authenticationError = (): ErrorConfig => ({
    type: LLMErrorType.AUTHENTICATION,
    message: "Invalid API key provided.",
    code: "invalid_api_key",
    details: {
        provided: "sk-...****",
        hint: "Check your API key configuration",
    },
    retryable: false,
});

/**
 * Quota exceeded error
 */
export const quotaExceeded = (): ErrorConfig => ({
    type: LLMErrorType.QUOTA_EXCEEDED,
    message: "Monthly quota exceeded.",
    code: "quota_exceeded",
    details: {
        quotaType: "monthly",
        limit: 1000000,
        used: 1000000,
        resetDate: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(),
    },
    retryable: false,
});

/**
 * Model not found error
 */
export const modelNotFound = (modelName = "gpt-5"): ErrorConfig => ({
    type: LLMErrorType.MODEL_NOT_FOUND,
    message: `The model '${modelName}' does not exist or you do not have access to it.`,
    code: "model_not_found",
    details: {
        requestedModel: modelName,
        availableModels: ["gpt-4o", "gpt-4o-mini", "gpt-4-turbo"],
    },
    retryable: false,
});

/**
 * Invalid request error
 */
export const invalidRequest = (field = "messages"): ErrorConfig => ({
    type: LLMErrorType.INVALID_REQUEST,
    message: `Invalid value for '${field}'`,
    code: "invalid_request_error",
    details: {
        field,
        reason: "Value must not be empty",
        example: field === "messages" ? [{ role: "user", content: "Hello" }] : undefined,
    },
    retryable: false,
});

/**
 * Content filter error
 */
export const contentFilter = (): ErrorConfig => ({
    type: LLMErrorType.CONTENT_FILTER,
    message: "The content was filtered due to policy violations.",
    code: "content_policy_violation",
    details: {
        categories: ["violence", "harmful_content"],
        severity: "high",
    },
    retryable: false,
});

/**
 * Timeout error
 */
export const timeout = (duration = 30000): ErrorConfig => ({
    type: LLMErrorType.TIMEOUT,
    message: `Request timed out after ${duration}ms`,
    code: "request_timeout",
    details: {
        timeoutMs: duration,
        elapsed: duration + Math.floor(Math.random() * 1000),
    },
    retryable: true,
});

/**
 * Network error
 */
export const networkError = (): ErrorConfig => ({
    type: LLMErrorType.NETWORK_ERROR,
    message: "Failed to connect to AI service",
    code: "ECONNREFUSED",
    details: {
        errno: -111,
        syscall: "connect",
        address: "api.openai.com",
        port: 443,
    },
    retryable: true,
});

/**
 * Internal server error
 */
export const internalError = (): ErrorConfig => ({
    type: LLMErrorType.INTERNAL_ERROR,
    message: "The server had an error while processing your request.",
    code: "internal_server_error",
    details: {
        requestId: `req_${Math.random().toString(36).substr(2, 9)}`,
        timestamp: new Date().toISOString(),
    },
    retryable: true,
});

/**
 * Context length exceeded error
 */
export const contextLengthExceeded = (limit = 128000): ErrorConfig => ({
    type: LLMErrorType.INVALID_REQUEST,
    message: `The request exceeds the context length limit of ${limit} tokens.`,
    code: "context_length_exceeded",
    details: {
        requestedTokens: limit + 1000,
        maxTokens: limit,
        suggestion: "Reduce the size of your messages or use a model with a larger context window.",
    },
    retryable: false,
});

/**
 * Service unavailable error
 */
export const serviceUnavailable = (): ErrorConfig => ({
    type: LLMErrorType.INTERNAL_ERROR,
    message: "The AI service is temporarily unavailable.",
    code: "service_unavailable",
    details: {
        status: 503,
        retryAfter: 300,
        maintenanceWindow: false,
    },
    retryable: true,
    retryAfter: 300,
});

/**
 * Malformed response error
 */
export const malformedResponse = (): ErrorConfig => ({
    type: LLMErrorType.INTERNAL_ERROR,
    message: "Received malformed response from AI service",
    code: "malformed_response",
    details: {
        parseError: "Unexpected token in JSON",
        rawResponse: "{\"content\": \"Hello\", \"tool_calls\": [{",
    },
    retryable: true,
});

/**
 * Tool execution error
 */
export const toolExecutionError = (toolName = "search"): ErrorConfig => ({
    type: LLMErrorType.INTERNAL_ERROR,
    message: `Failed to execute tool '${toolName}'`,
    code: "tool_execution_error",
    details: {
        toolName,
        error: "Tool not found in registry",
        availableTools: ["calculate", "fetch", "process"],
    },
    retryable: false,
});

/**
 * Partial failure error
 */
export const partialFailure = (): ErrorConfig => ({
    type: LLMErrorType.INTERNAL_ERROR,
    message: "Request partially failed",
    code: "partial_failure",
    details: {
        successful: ["text_generation"],
        failed: ["tool_execution", "reasoning"],
        successRate: 0.33,
    },
    retryable: true,
});

/**
 * Chain of errors (for cascading failures)
 */
export const errorChain = (): ErrorConfig[] => [
    networkError(),
    timeout(5000),
    internalError(),
    serviceUnavailable(),
];

/**
 * Random error generator for chaos testing
 */
export const randomError = (): ErrorConfig => {
    const errors = [
        rateLimit(),
        authenticationError(),
        quotaExceeded(),
        modelNotFound(),
        invalidRequest(),
        contentFilter(),
        timeout(),
        networkError(),
        internalError(),
    ];
    
    return errors[Math.floor(Math.random() * errors.length)];
};
