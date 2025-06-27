/**
 * Error Factory
 * 
 * Factory for creating AI error responses and simulating various failure scenarios.
 */

import { LLMErrorType } from "@vrooli/shared";
import type { LLMError } from "@vrooli/shared";
import type { ErrorConfig, MockFactoryResult } from "../types.js";
import { createAIMockResponse } from "./responseFactory.js";

/**
 * Create an error response
 */
export function createErrorResponse(
    errorType: LLMErrorType,
    customMessage?: string,
    details?: Record<string, unknown>,
): MockFactoryResult {
    const error = createLLMError(errorType, customMessage, details);
    
    return createAIMockResponse({
        content: "",
        error: {
            type: errorType,
            message: error.message,
            code: error.code,
            details: error.details,
            retryable: error.retryable,
            retryAfter: error.retryAfter,
        },
        finishReason: "error",
        metadata: {
            errorType,
            timestamp: new Date().toISOString(),
        },
    });
}

/**
 * Create specific error types
 */
export const errorFactories = {
    rateLimit: (retryAfter = 60) => createErrorResponse(
        LLMErrorType.RATE_LIMIT,
        "Rate limit exceeded. Please try again later.",
        { retryAfter, limit: 100, used: 100 },
    ),
    
    authentication: (reason?: string) => createErrorResponse(
        LLMErrorType.AUTHENTICATION,
        reason || "Invalid API key or authentication credentials.",
        { authMethod: "api_key" },
    ),
    
    quotaExceeded: (quotaType = "monthly") => createErrorResponse(
        LLMErrorType.QUOTA_EXCEEDED,
        `Your ${quotaType} quota has been exceeded.`,
        { quotaType, limit: 1000000, used: 1000000 },
    ),
    
    modelNotFound: (model: string) => createErrorResponse(
        LLMErrorType.MODEL_NOT_FOUND,
        `Model '${model}' not found or not accessible.`,
        { requestedModel: model, availableModels: ["gpt-4o", "gpt-4o-mini"] },
    ),
    
    invalidRequest: (field?: string, reason?: string) => createErrorResponse(
        LLMErrorType.INVALID_REQUEST,
        reason || `Invalid request${field ? `: ${field} is invalid` : ""}.`,
        { field, validation: "failed" },
    ),
    
    contentFilter: (reason = "potentially harmful content") => createErrorResponse(
        LLMErrorType.CONTENT_FILTER,
        `Content was filtered due to ${reason}.`,
        { filterType: "content_policy", triggered: true },
    ),
    
    timeout: (duration = 30000) => createErrorResponse(
        LLMErrorType.TIMEOUT,
        `Request timed out after ${duration}ms.`,
        { timeoutMs: duration, elapsed: duration + 100 },
    ),
    
    networkError: (code?: string) => createErrorResponse(
        LLMErrorType.NETWORK_ERROR,
        "Network error occurred while communicating with AI service.",
        { networkCode: code || "ECONNREFUSED" },
    ),
    
    internalError: (details?: string) => createErrorResponse(
        LLMErrorType.INTERNAL_ERROR,
        "An internal error occurred.",
        { details: details || "Unexpected error in AI service" },
    ),
};

/**
 * Create an error from config
 */
export function createErrorFromConfig(config: ErrorConfig): LLMError {
    return createLLMError(
        config.type,
        config.message,
        config.details,
        config.retryable,
        config.retryAfter,
    );
}

/**
 * Create a flaky service simulation
 */
export function createFlakyServiceResponse(
    successRate = 0.7,
    attempt = 1,
): MockFactoryResult {
    const shouldSucceed = Math.random() < successRate;
    
    if (shouldSucceed) {
        return createAIMockResponse({
            content: `Response succeeded on attempt ${attempt}.`,
            metadata: {
                attempt,
                simulatedFailureRate: 1 - successRate,
            },
        });
    } else {
        const errorTypes = [
            LLMErrorType.NETWORK_ERROR,
            LLMErrorType.TIMEOUT,
            LLMErrorType.INTERNAL_ERROR,
        ];
        const errorType = errorTypes[Math.floor(Math.random() * errorTypes.length)];
        
        return createErrorResponse(
            errorType,
            `Simulated failure on attempt ${attempt}`,
            { attempt, successRate },
        );
    }
}

/**
 * Create a degraded service response
 */
export function createDegradedServiceResponse(
    degradationLevel: "mild" | "moderate" | "severe",
): MockFactoryResult {
    const configs = {
        mild: {
            delay: 2000,
            confidence: 0.8,
            content: "Response generated with slight delays.",
        },
        moderate: {
            delay: 5000,
            confidence: 0.6,
            content: "Service is experiencing issues. Response may be less accurate.",
        },
        severe: {
            delay: 10000,
            confidence: 0.4,
            content: "Service severely degraded. Limited functionality available.",
        },
    };
    
    const config = configs[degradationLevel];
    
    return createAIMockResponse({
        content: config.content,
        confidence: config.confidence,
        delay: config.delay,
        metadata: {
            serviceStatus: "degraded",
            degradationLevel,
            responseTime: config.delay,
        },
    });
}

/**
 * Create cascading error scenario
 */
export function createCascadingErrors(
    stages: Array<{ stage: string; errorType: LLMErrorType; duration?: number }>,
): MockFactoryResult[] {
    return stages.map((stage, index) => {
        const isRecovery = index === stages.length - 1;
        
        if (isRecovery) {
            return createAIMockResponse({
                content: "Service recovered. Operations resuming normally.",
                metadata: {
                    stage: "recovery",
                    previousErrors: stages.length - 1,
                },
            });
        }
        
        return createErrorResponse(
            stage.errorType,
            `Error in ${stage.stage} stage`,
            {
                stage: stage.stage,
                cascadeIndex: index,
                duration: stage.duration,
            },
        );
    });
}

/**
 * Helper to create LLM error object
 */
function createLLMError(
    type: LLMErrorType,
    customMessage?: string,
    details?: Record<string, unknown>,
    retryable?: boolean,
    retryAfter?: number,
): LLMError {
    const errorConfigs: Record<LLMErrorType, Partial<LLMError>> = {
        [LLMErrorType.AUTHENTICATION]: {
            message: "Authentication failed",
            code: "AUTH_FAILED",
            retryable: false,
        },
        [LLMErrorType.RATE_LIMIT]: {
            message: "Rate limit exceeded",
            code: "RATE_LIMIT_EXCEEDED",
            retryable: true,
            retryAfter: 60,
        },
        [LLMErrorType.QUOTA_EXCEEDED]: {
            message: "Quota exceeded",
            code: "QUOTA_EXCEEDED",
            retryable: false,
        },
        [LLMErrorType.MODEL_NOT_FOUND]: {
            message: "Model not found",
            code: "MODEL_NOT_FOUND",
            retryable: false,
        },
        [LLMErrorType.INVALID_REQUEST]: {
            message: "Invalid request",
            code: "INVALID_REQUEST",
            retryable: false,
        },
        [LLMErrorType.CONTENT_FILTER]: {
            message: "Content filtered",
            code: "CONTENT_FILTERED",
            retryable: false,
        },
        [LLMErrorType.TIMEOUT]: {
            message: "Request timeout",
            code: "TIMEOUT",
            retryable: true,
        },
        [LLMErrorType.NETWORK_ERROR]: {
            message: "Network error",
            code: "NETWORK_ERROR",
            retryable: true,
        },
        [LLMErrorType.INTERNAL_ERROR]: {
            message: "Internal server error",
            code: "INTERNAL_ERROR",
            retryable: true,
        },
    };
    
    const baseConfig = errorConfigs[type];
    
    return {
        type,
        message: customMessage || baseConfig.message || "Unknown error",
        code: baseConfig.code,
        details: details || {},
        retryable: retryable !== undefined ? retryable : baseConfig.retryable || false,
        retryAfter: retryAfter || baseConfig.retryAfter,
    };
}

/**
 * Create partial failure scenario
 */
export function createPartialFailureResponse(
    successfulParts: string[],
    failedParts: Array<{ part: string; reason: string }>,
): MockFactoryResult {
    const content = `Partially completed. Successful: ${successfulParts.join(", ")}. Failed: ${failedParts.map(f => f.part).join(", ")}.`;
    
    return createAIMockResponse({
        content,
        confidence: 0.6,
        finishReason: "stop",
        metadata: {
            partial: true,
            successful: successfulParts,
            failed: failedParts,
            successRate: successfulParts.length / (successfulParts.length + failedParts.length),
        },
    });
}
