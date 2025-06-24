/**
 * Response Validator
 * 
 * Validates AI mock responses to ensure they conform to expected structures.
 */

import type { LLMResponse, LLMStreamEvent, LLMToolCall } from "@vrooli/shared";
import type { MockValidationResult, AIMockConfig } from "../types.js";

/**
 * Validate an LLM response
 */
export function validateAIResponse(response: unknown): MockValidationResult {
    const errors: string[] = [];
    const warnings: string[] = [];
    const suggestions: string[] = [];
    
    if (!response || typeof response !== "object") {
        return {
            valid: false,
            errors: ["Response must be a non-null object"]
        };
    }
    
    const r = response as any;
    
    // Required fields
    if (typeof r.content !== "string") {
        errors.push("Response must have a 'content' string property");
    }
    
    if (typeof r.confidence !== "number") {
        errors.push("Response must have a 'confidence' number property");
    } else if (r.confidence < 0 || r.confidence > 1) {
        errors.push("Confidence must be between 0 and 1");
    }
    
    if (typeof r.tokensUsed !== "number") {
        errors.push("Response must have a 'tokensUsed' number property");
    } else if (r.tokensUsed < 0) {
        errors.push("tokensUsed must be non-negative");
    }
    
    if (typeof r.model !== "string") {
        errors.push("Response must have a 'model' string property");
    }
    
    if (!r.finishReason || !["stop", "length", "tool_calls", "content_filter", "error"].includes(r.finishReason)) {
        errors.push("Response must have a valid 'finishReason' property");
    }
    
    // Optional fields
    if (r.reasoning !== undefined && typeof r.reasoning !== "string") {
        errors.push("If present, 'reasoning' must be a string");
    }
    
    if (r.toolCalls !== undefined) {
        const toolCallResult = validateToolCalls(r.toolCalls);
        errors.push(...toolCallResult.errors);
        warnings.push(...toolCallResult.warnings);
    }
    
    if (r.metadata !== undefined && (typeof r.metadata !== "object" || r.metadata === null)) {
        errors.push("If present, 'metadata' must be an object");
    }
    
    // Warnings
    if (r.confidence > 0.95) {
        warnings.push("Very high confidence (>0.95) - ensure this is intentional");
    }
    
    if (r.content.length === 0 && !r.toolCalls?.length) {
        warnings.push("Empty content with no tool calls");
    }
    
    if (r.tokensUsed > 100000) {
        warnings.push("Unusually high token usage - verify this is correct");
    }
    
    // Suggestions
    if (!r.reasoning && r.model?.includes("o1")) {
        suggestions.push("O1 models typically include reasoning");
    }
    
    if (r.finishReason === "length" && !r.content.endsWith("...")) {
        suggestions.push("Content truncated due to length - consider adding ellipsis");
    }
    
    return {
        valid: errors.length === 0,
        errors: errors.length > 0 ? errors : undefined,
        warnings: warnings.length > 0 ? warnings : undefined,
        suggestions: suggestions.length > 0 ? suggestions : undefined
    };
}

/**
 * Validate tool calls
 */
function validateToolCalls(toolCalls: unknown): { errors: string[]; warnings: string[] } {
    const errors: string[] = [];
    const warnings: string[] = [];
    
    if (!Array.isArray(toolCalls)) {
        errors.push("toolCalls must be an array");
        return { errors, warnings };
    }
    
    toolCalls.forEach((toolCall, index) => {
        if (!toolCall || typeof toolCall !== "object") {
            errors.push(`toolCall[${index}] must be an object`);
            return;
        }
        
        const tc = toolCall as any;
        
        if (typeof tc.id !== "string") {
            errors.push(`toolCall[${index}] must have an 'id' string property`);
        }
        
        if (tc.type !== "function") {
            errors.push(`toolCall[${index}] must have type 'function'`);
        }
        
        if (!tc.function || typeof tc.function !== "object") {
            errors.push(`toolCall[${index}] must have a 'function' object property`);
        } else {
            if (typeof tc.function.name !== "string") {
                errors.push(`toolCall[${index}].function must have a 'name' string property`);
            }
            
            if (typeof tc.function.arguments !== "string") {
                errors.push(`toolCall[${index}].function must have an 'arguments' string property`);
            } else {
                try {
                    JSON.parse(tc.function.arguments);
                } catch {
                    errors.push(`toolCall[${index}].function.arguments must be valid JSON`);
                }
            }
        }
    });
    
    if (toolCalls.length > 10) {
        warnings.push("Large number of tool calls (>10) may impact performance");
    }
    
    return { errors, warnings };
}

/**
 * Validate streaming event
 */
export function validateStreamEvent(event: unknown): MockValidationResult {
    const errors: string[] = [];
    const warnings: string[] = [];
    
    if (!event || typeof event !== "object") {
        return {
            valid: false,
            errors: ["Stream event must be a non-null object"]
        };
    }
    
    const e = event as any;
    
    if (!e.type) {
        errors.push("Stream event must have a 'type' property");
    } else {
        const validTypes = ["start", "text", "reasoning", "tool_call", "tool_result", "done", "error"];
        if (!validTypes.includes(e.type)) {
            errors.push(`Invalid stream event type: ${e.type}`);
        }
    }
    
    // Type-specific validation
    switch (e.type) {
        case "text":
            if (typeof e.text !== "string") {
                errors.push("Text event must have a 'text' string property");
            }
            break;
            
        case "reasoning":
            if (typeof e.reasoning !== "string") {
                errors.push("Reasoning event must have a 'reasoning' string property");
            }
            break;
            
        case "tool_call":
            if (!e.toolCall || typeof e.toolCall !== "object") {
                errors.push("Tool call event must have a 'toolCall' object property");
            }
            break;
            
        case "tool_result":
            if (typeof e.toolCallId !== "string") {
                errors.push("Tool result event must have a 'toolCallId' string property");
            }
            break;
            
        case "done":
            if (!e.usage || typeof e.usage !== "object") {
                errors.push("Done event must have a 'usage' object property");
            }
            if (typeof e.finishReason !== "string") {
                errors.push("Done event must have a 'finishReason' string property");
            }
            break;
            
        case "error":
            if (typeof e.error !== "string") {
                errors.push("Error event must have an 'error' string property");
            }
            break;
    }
    
    return {
        valid: errors.length === 0,
        errors: errors.length > 0 ? errors : undefined,
        warnings: warnings.length > 0 ? warnings : undefined
    };
}

/**
 * Validate mock configuration
 */
export function validateMockConfig(config: unknown): MockValidationResult {
    const errors: string[] = [];
    const warnings: string[] = [];
    const suggestions: string[] = [];
    
    if (!config || typeof config !== "object") {
        return {
            valid: false,
            errors: ["Mock config must be a non-null object"]
        };
    }
    
    const c = config as AIMockConfig;
    
    // All fields are optional, but if present must be correct type
    if (c.content !== undefined && typeof c.content !== "string") {
        errors.push("If present, 'content' must be a string");
    }
    
    if (c.reasoning !== undefined && typeof c.reasoning !== "string") {
        errors.push("If present, 'reasoning' must be a string");
    }
    
    if (c.confidence !== undefined) {
        if (typeof c.confidence !== "number") {
            errors.push("If present, 'confidence' must be a number");
        } else if (c.confidence < 0 || c.confidence > 1) {
            errors.push("Confidence must be between 0 and 1");
        }
    }
    
    if (c.model !== undefined && typeof c.model !== "string") {
        errors.push("If present, 'model' must be a string");
    }
    
    if (c.delay !== undefined) {
        if (typeof c.delay !== "number") {
            errors.push("If present, 'delay' must be a number");
        } else if (c.delay < 0) {
            errors.push("Delay must be non-negative");
        } else if (c.delay > 30000) {
            warnings.push("Very long delay (>30s) may cause timeouts");
        }
    }
    
    // Validate nested structures
    if (c.toolCalls !== undefined) {
        if (!Array.isArray(c.toolCalls)) {
            errors.push("If present, 'toolCalls' must be an array");
        } else {
            c.toolCalls.forEach((tc, i) => {
                if (!tc.name || typeof tc.name !== "string") {
                    errors.push(`toolCalls[${i}] must have a 'name' string property`);
                }
                if (!tc.arguments || typeof tc.arguments !== "object") {
                    errors.push(`toolCalls[${i}] must have an 'arguments' object property`);
                }
            });
        }
    }
    
    if (c.streaming !== undefined) {
        if (!c.streaming || typeof c.streaming !== "object") {
            errors.push("If present, 'streaming' must be an object");
        } else if (!Array.isArray(c.streaming.chunks)) {
            errors.push("streaming.chunks must be an array");
        }
    }
    
    if (c.error !== undefined) {
        if (!c.error || typeof c.error !== "object") {
            errors.push("If present, 'error' must be an object");
        } else if (!c.error.type) {
            errors.push("error must have a 'type' property");
        }
    }
    
    return {
        valid: errors.length === 0,
        errors: errors.length > 0 ? errors : undefined,
        warnings: warnings.length > 0 ? warnings : undefined,
        suggestions: suggestions.length > 0 ? suggestions : undefined
    };
}