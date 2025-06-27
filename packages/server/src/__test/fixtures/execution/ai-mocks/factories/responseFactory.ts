/**
 * Response Factory
 * 
 * Main factory for creating AI mock responses with full type safety and validation.
 */

import type { LLMResponse, LLMUsage } from "@vrooli/shared";
import {
    DEFAULT_MOCK_CONFIG, 
} from "../types.js";
import type { 
    AIMockConfig, 
    MockFactoryResult, 
} from "../types.js";

/**
 * Create a basic AI mock response
 */
export function createAIMockResponse(config: AIMockConfig = {}): MockFactoryResult {
    const mergedConfig = {
        ...DEFAULT_MOCK_CONFIG,
        ...config,
    };
    
    // Calculate token usage
    const usage: LLMUsage = {
        promptTokens: Math.floor((mergedConfig.content?.length || 0) / 4),
        completionTokens: mergedConfig.tokensUsed || 100,
        totalTokens: 0,
        cost: 0,
    };
    usage.totalTokens = usage.promptTokens + usage.completionTokens;
    
    // Calculate cost based on model
    const modelPricing = getModelPricing(mergedConfig.model || "gpt-4o-mini");
    usage.cost = (usage.promptTokens / 1000) * modelPricing.input + 
                 (usage.completionTokens / 1000) * modelPricing.output;
    
    // Build response
    const response: LLMResponse = {
        content: mergedConfig.content || "Mock response",
        reasoning: mergedConfig.reasoning,
        confidence: mergedConfig.confidence || 0.85,
        tokensUsed: usage.totalTokens,
        toolCalls: mergedConfig.toolCalls?.map((tc, index) => ({
            id: `call_${Date.now()}_${index}`,
            type: "function" as const,
            function: {
                name: tc.name,
                arguments: JSON.stringify(tc.arguments),
            },
        })),
        model: mergedConfig.model || "gpt-4o-mini",
        finishReason: mergedConfig.finishReason || "stop",
        metadata: mergedConfig.metadata,
    };
    
    return {
        response,
        usage,
        raw: mergedConfig,
    };
}

/**
 * Create a response with specific characteristics
 */
export function createCharacteristicResponse(
    characteristics: {
        length?: "short" | "medium" | "long";
        tone?: "formal" | "casual" | "technical";
        confidence?: "low" | "medium" | "high";
        includeReasoning?: boolean;
    },
): MockFactoryResult {
    const config: AIMockConfig = {
        content: generateContent(characteristics),
        confidence: getConfidenceValue(characteristics.confidence),
        reasoning: characteristics.includeReasoning ? generateReasoning(characteristics) : undefined,
    };
    
    return createAIMockResponse(config);
}

/**
 * Create a multi-turn conversation response
 */
export function createConversationResponse(
    context: {
        previousMessages: number;
        topic?: string;
        shouldReference?: boolean;
    },
): MockFactoryResult {
    const config: AIMockConfig = {
        content: generateConversationContent(context),
        confidence: 0.9,
        metadata: {
            conversationDepth: context.previousMessages,
            topic: context.topic,
        },
    };
    
    return createAIMockResponse(config);
}

/**
 * Create a response with specific token usage
 */
export function createTokenLimitedResponse(
    maxTokens: number,
    exceedLimit = false,
): MockFactoryResult {
    const actualTokens = exceedLimit ? maxTokens + 50 : Math.floor(maxTokens * 0.9);
    const content = generateContentForTokens(actualTokens);
    
    const config: AIMockConfig = {
        content,
        tokensUsed: actualTokens,
        finishReason: exceedLimit ? "length" : "stop",
    };
    
    return createAIMockResponse(config);
}

/**
 * Helper functions
 */
function getModelPricing(model: string): { input: number; output: number } {
    const pricing: Record<string, { input: number; output: number }> = {
        "gpt-4o": { input: 0.0025, output: 0.01 },
        "gpt-4o-mini": { input: 0.00015, output: 0.0006 },
        "gpt-4-turbo": { input: 0.01, output: 0.03 },
        "o1-mini": { input: 0.003, output: 0.012 },
        "o1-preview": { input: 0.015, output: 0.06 },
    };
    
    return pricing[model] || { input: 0.001, output: 0.002 };
}

function generateContent(characteristics: any): string {
    const lengths = {
        short: "This is a brief response.",
        medium: "This is a medium-length response that provides more detail and context to the user's query.",
        long: "This is a comprehensive response that thoroughly addresses the user's question with multiple points of consideration, examples, and detailed explanations that ensure complete understanding of the topic at hand.",
    };
    
    const tones = {
        formal: "I shall provide you with the requested information.",
        casual: "Sure thing! Here's what you need to know.",
        technical: "The implementation utilizes a factory pattern for response generation.",
    };
    
    const length = characteristics.length || "medium";
    const tone = characteristics.tone || "casual";
    
    return `${tones[tone]} ${lengths[length]}`;
}

function generateReasoning(characteristics: any): string {
    const reasoningTemplates = {
        formal: "Upon careful consideration of the query, I have determined that...",
        casual: "Let me think about this... I believe the best approach is...",
        technical: "Analyzing the requirements: 1) Parse input parameters, 2) Validate constraints, 3) Generate appropriate response...",
    };
    
    return reasoningTemplates[characteristics.tone || "casual"];
}

function getConfidenceValue(level?: "low" | "medium" | "high"): number {
    const values = {
        low: 0.6,
        medium: 0.85,
        high: 0.95,
    };
    
    return values[level || "medium"];
}

function generateConversationContent(context: any): string {
    const { previousMessages, topic, shouldReference } = context;
    
    let content = "Continuing our discussion";
    if (topic) {
        content += ` about ${topic}`;
    }
    content += ".";
    
    if (shouldReference && previousMessages > 0) {
        content += " As we discussed earlier,";
    }
    
    content += " here's my response based on the context of our conversation.";
    
    return content;
}

function generateContentForTokens(tokens: number): string {
    // Rough approximation: 1 token ≈ 4 characters
    const targetChars = tokens * 4;
    const words = [
        "The", "quick", "brown", "fox", "jumps", "over", "the", "lazy", "dog",
        "in", "a", "demonstration", "of", "agility", "and", "grace", "while",
        "showcasing", "natural", "movement", "patterns", "commonly", "observed",
    ];
    
    let content = "";
    while (content.length < targetChars) {
        content += words[Math.floor(Math.random() * words.length)] + " ";
    }
    
    return content.trim().substring(0, targetChars);
}

// Export the DEFAULT_MOCK_CONFIG for use in other modules
export const DEFAULT_MOCK_CONFIG = {
    model: "gpt-4o-mini",
    confidence: 0.85,
    finishReason: "stop" as const,
    tokensUsed: 100,
};
