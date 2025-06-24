/**
 * Contract Validator
 * 
 * Ensures mock responses comply with the actual AI service interfaces.
 */

import type { LLMRequest, LLMResponse, LLMStreamEvent } from "@vrooli/shared";
import type { ResponseStreamOptions } from "../../../../../services/conversation/types.js";
import type { MockValidationResult } from "../types.js";

/**
 * Validate that a mock response matches the LLMResponse interface
 */
export function validateLLMResponseContract(response: unknown): response is LLMResponse {
    if (!response || typeof response !== "object") return false;
    
    const r = response as any;
    
    return (
        typeof r.content === "string" &&
        typeof r.confidence === "number" &&
        typeof r.tokensUsed === "number" &&
        typeof r.model === "string" &&
        typeof r.finishReason === "string" &&
        ["stop", "length", "tool_calls", "content_filter", "error"].includes(r.finishReason) &&
        (r.reasoning === undefined || typeof r.reasoning === "string") &&
        (r.toolCalls === undefined || Array.isArray(r.toolCalls)) &&
        (r.metadata === undefined || typeof r.metadata === "object")
    );
}

/**
 * Validate that a request matches the LLMRequest interface
 */
export function validateLLMRequestContract(request: unknown): request is LLMRequest {
    if (!request || typeof request !== "object") return false;
    
    const r = request as any;
    
    return (
        typeof r.model === "string" &&
        Array.isArray(r.messages) &&
        r.messages.every((m: any) => 
            typeof m === "object" &&
            ["system", "user", "assistant", "tool"].includes(m.role) &&
            typeof m.content === "string"
        ) &&
        (r.systemMessage === undefined || typeof r.systemMessage === "string") &&
        (r.tools === undefined || Array.isArray(r.tools)) &&
        (r.maxTokens === undefined || typeof r.maxTokens === "number") &&
        (r.temperature === undefined || typeof r.temperature === "number")
    );
}

/**
 * Validate that stream options match the ResponseStreamOptions interface
 */
export function validateStreamOptionsContract(options: unknown): options is Partial<ResponseStreamOptions> {
    if (!options || typeof options !== "object") return false;
    
    const o = options as any;
    
    return (
        (o.model === undefined || typeof o.model === "string") &&
        (o.input === undefined || Array.isArray(o.input)) &&
        (o.tools === undefined || Array.isArray(o.tools)) &&
        (o.systemMessage === undefined || typeof o.systemMessage === "string") &&
        (o.maxTokens === undefined || typeof o.maxTokens === "number") &&
        (o.temperature === undefined || typeof o.temperature === "number") &&
        (o.userData === undefined || (typeof o.userData === "object" && typeof o.userData.id === "string"))
    );
}

/**
 * Validate contract compliance for mock/real service compatibility
 */
export function validateServiceCompatibility(
    mockResponse: unknown,
    realServiceInterface: "LLMResponse" | "StreamEvent"
): MockValidationResult {
    const errors: string[] = [];
    const warnings: string[] = [];
    const suggestions: string[] = [];
    
    switch (realServiceInterface) {
        case "LLMResponse":
            if (!validateLLMResponseContract(mockResponse)) {
                errors.push("Mock response does not match LLMResponse interface");
                
                // Detailed field checking for better error messages
                const r = mockResponse as any;
                if (!r) {
                    errors.push("Response is null or undefined");
                } else {
                    if (typeof r.content !== "string") errors.push("Missing or invalid 'content' field");
                    if (typeof r.confidence !== "number") errors.push("Missing or invalid 'confidence' field");
                    if (typeof r.tokensUsed !== "number") errors.push("Missing or invalid 'tokensUsed' field");
                    if (typeof r.model !== "string") errors.push("Missing or invalid 'model' field");
                    if (typeof r.finishReason !== "string") errors.push("Missing or invalid 'finishReason' field");
                }
            }
            break;
            
        case "StreamEvent":
            if (!validateStreamEventContract(mockResponse)) {
                errors.push("Mock event does not match LLMStreamEvent interface");
            }
            break;
    }
    
    return {
        valid: errors.length === 0,
        errors: errors.length > 0 ? errors : undefined,
        warnings: warnings.length > 0 ? warnings : undefined,
        suggestions: suggestions.length > 0 ? suggestions : undefined
    };
}

/**
 * Validate stream event contract
 */
function validateStreamEventContract(event: unknown): event is LLMStreamEvent {
    if (!event || typeof event !== "object") return false;
    
    const e = event as any;
    
    if (!e.type || typeof e.type !== "string") return false;
    
    switch (e.type) {
        case "start":
            return e.metadata === undefined || typeof e.metadata === "object";
            
        case "text":
            return typeof e.text === "string";
            
        case "reasoning":
            return typeof e.reasoning === "string";
            
        case "tool_call":
            return e.toolCall && typeof e.toolCall === "object";
            
        case "tool_result":
            return typeof e.toolCallId === "string";
            
        case "done":
            return e.usage && typeof e.usage === "object" && 
                   typeof e.finishReason === "string";
            
        case "error":
            return typeof e.error === "string";
            
        default:
            return false;
    }
}

/**
 * Validate that mock behavior matches real service behavior
 */
export function validateBehaviorContract(config: {
    mockBehavior: any;
    serviceType: "openai" | "anthropic" | "custom";
}): MockValidationResult {
    const errors: string[] = [];
    const warnings: string[] = [];
    const suggestions: string[] = [];
    
    const { mockBehavior, serviceType } = config;
    
    // Service-specific validations
    switch (serviceType) {
        case "openai":
            // OpenAI specific constraints
            if (mockBehavior.model?.startsWith("o1") && mockBehavior.tools?.length > 0) {
                errors.push("O1 models do not support tool calls");
            }
            if (mockBehavior.model?.startsWith("o1") && mockBehavior.systemMessage) {
                warnings.push("O1 models ignore system messages");
            }
            if (mockBehavior.streaming && mockBehavior.model?.startsWith("o1")) {
                errors.push("O1 models do not support streaming");
            }
            break;
            
        case "anthropic":
            // Anthropic specific constraints
            if (mockBehavior.maxTokens > 200000) {
                errors.push("Anthropic models have a maximum of 200k output tokens");
            }
            break;
    }
    
    // General validations
    if (mockBehavior.temperature !== undefined) {
        if (mockBehavior.temperature < 0 || mockBehavior.temperature > 2) {
            errors.push("Temperature must be between 0 and 2");
        }
    }
    
    if (mockBehavior.topP !== undefined) {
        if (mockBehavior.topP < 0 || mockBehavior.topP > 1) {
            errors.push("Top-p must be between 0 and 1");
        }
    }
    
    if (mockBehavior.presencePenalty !== undefined) {
        if (mockBehavior.presencePenalty < -2 || mockBehavior.presencePenalty > 2) {
            errors.push("Presence penalty must be between -2 and 2");
        }
    }
    
    if (mockBehavior.frequencyPenalty !== undefined) {
        if (mockBehavior.frequencyPenalty < -2 || mockBehavior.frequencyPenalty > 2) {
            errors.push("Frequency penalty must be between -2 and 2");
        }
    }
    
    // Behavioral suggestions
    if (mockBehavior.temperature > 1.5) {
        warnings.push("High temperature (>1.5) may produce very random outputs");
    }
    
    if (mockBehavior.maxTokens && mockBehavior.maxTokens < 10) {
        warnings.push("Very low max tokens (<10) may result in incomplete responses");
    }
    
    if (mockBehavior.tools?.length > 20) {
        warnings.push("Large number of tools (>20) may impact performance and accuracy");
    }
    
    return {
        valid: errors.length === 0,
        errors: errors.length > 0 ? errors : undefined,
        warnings: warnings.length > 0 ? warnings : undefined,
        suggestions: suggestions.length > 0 ? suggestions : undefined
    };
}

/**
 * Comprehensive contract validation
 */
export function validateFullContract(mock: {
    request?: unknown;
    response?: unknown;
    streamEvents?: unknown[];
    behavior?: any;
    serviceType?: "openai" | "anthropic" | "custom";
}): MockValidationResult {
    const allErrors: string[] = [];
    const allWarnings: string[] = [];
    const allSuggestions: string[] = [];
    
    // Validate request if provided
    if (mock.request !== undefined) {
        if (!validateLLMRequestContract(mock.request)) {
            allErrors.push("Request does not match LLMRequest contract");
        }
    }
    
    // Validate response if provided
    if (mock.response !== undefined) {
        const responseValidation = validateServiceCompatibility(mock.response, "LLMResponse");
        if (responseValidation.errors) allErrors.push(...responseValidation.errors);
        if (responseValidation.warnings) allWarnings.push(...responseValidation.warnings);
    }
    
    // Validate stream events if provided
    if (mock.streamEvents !== undefined) {
        mock.streamEvents.forEach((event, index) => {
            const eventValidation = validateServiceCompatibility(event, "StreamEvent");
            if (eventValidation.errors) {
                allErrors.push(`Stream event ${index}: ${eventValidation.errors.join(", ")}`);
            }
        });
    }
    
    // Validate behavior if provided
    if (mock.behavior !== undefined && mock.serviceType) {
        const behaviorValidation = validateBehaviorContract({
            mockBehavior: mock.behavior,
            serviceType: mock.serviceType
        });
        if (behaviorValidation.errors) allErrors.push(...behaviorValidation.errors);
        if (behaviorValidation.warnings) allWarnings.push(...behaviorValidation.warnings);
        if (behaviorValidation.suggestions) allSuggestions.push(...behaviorValidation.suggestions);
    }
    
    return {
        valid: allErrors.length === 0,
        errors: allErrors.length > 0 ? allErrors : undefined,
        warnings: allWarnings.length > 0 ? allWarnings : undefined,
        suggestions: allSuggestions.length > 0 ? allSuggestions : undefined
    };
}