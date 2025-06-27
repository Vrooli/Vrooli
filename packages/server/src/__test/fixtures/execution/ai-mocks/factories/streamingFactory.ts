/**
 * Streaming Factory
 * 
 * Factory for creating streaming AI responses with various patterns and behaviors.
 */

import type { LLMStreamEvent, LLMUsage } from "@vrooli/shared";
import type { 
    StreamingConfig, 
    StreamChunk,
    ErrorConfig,
    MockFactoryResult, 
} from "../types.js";
import { createAIMockResponse } from "./responseFactory.js";
import { createErrorFromConfig } from "./errorFactory.js";

/**
 * Create a streaming mock response
 */
export async function* createStreamingMock(
    config: StreamingConfig,
): AsyncGenerator<LLMStreamEvent> {
    // Start event
    yield {
        type: "start",
        metadata: {
            model: "gpt-4o-mini",
            timestamp: new Date().toISOString(),
        },
    };
    
    const chunks = normalizeChunks(config.chunks);
    const delays = normalizeDelays(config.chunkDelay, chunks.length);
    
    let totalText = "";
    let totalReasoning = "";
    
    try {
        // Stream text chunks
        for (let i = 0; i < chunks.length; i++) {
            const chunk = chunks[i];
            
            // Check for interruption
            if (config.interruptAt && i >= config.interruptAt) {
                throw new Error("Stream interrupted");
            }
            
            // Apply delay
            if (delays[i] > 0) {
                await sleep(delays[i]);
            }
            
            // Check for error injection
            if (config.error?.occurAfter && i >= config.error.occurAfter) {
                const error = createErrorFromConfig(config.error);
                yield {
                    type: "error",
                    error: error.message,
                };
                return;
            }
            
            // Yield appropriate event based on chunk type
            if (typeof chunk === "string") {
                totalText += chunk;
                yield {
                    type: "text",
                    text: chunk,
                };
            } else {
                switch (chunk.type) {
                    case "text":
                        totalText += chunk.content;
                        yield {
                            type: "text",
                            text: chunk.content,
                        };
                        break;
                        
                    case "reasoning":
                        totalReasoning += chunk.content;
                        yield {
                            type: "reasoning",
                            reasoning: chunk.content,
                        };
                        break;
                        
                    case "tool_call":
                        const toolCall = JSON.parse(chunk.content);
                        yield {
                            type: "tool_call",
                            toolCall: {
                                id: `call_${Date.now()}`,
                                type: "function",
                                function: {
                                    name: toolCall.name,
                                    arguments: JSON.stringify(toolCall.arguments),
                                },
                            },
                        };
                        break;
                }
            }
        }
        
        // Stream reasoning if configured
        if (config.includeReasoning && config.reasoningChunks) {
            for (const reasoningChunk of config.reasoningChunks) {
                await sleep(50); // Small delay between reasoning chunks
                totalReasoning += reasoningChunk;
                yield {
                    type: "reasoning",
                    reasoning: reasoningChunk,
                };
            }
        }
        
        // Calculate usage
        const usage: LLMUsage = {
            promptTokens: 50, // Mock value
            completionTokens: Math.ceil((totalText.length + totalReasoning.length) / 4),
            totalTokens: 0,
            cost: 0,
        };
        usage.totalTokens = usage.promptTokens + usage.completionTokens;
        usage.cost = (usage.totalTokens / 1000) * 0.0006; // Mock pricing
        
        // Done event
        yield {
            type: "done",
            usage,
            finishReason: "stop",
        };
        
    } catch (error) {
        yield {
            type: "error",
            error: error instanceof Error ? error.message : "Unknown streaming error",
        };
    }
}

/**
 * Create a typing simulation stream
 */
export async function* createTypingStream(
    text: string,
    wordsPerMinute = 150,
): AsyncGenerator<LLMStreamEvent> {
    yield { type: "start", metadata: {} };
    
    const words = text.split(" ");
    const delayPerWord = (60 / wordsPerMinute) * 1000; // Convert to milliseconds
    
    let accumulated = "";
    
    for (let i = 0; i < words.length; i++) {
        const word = words[i];
        const isLastWord = i === words.length - 1;
        
        // Type out each character with small random delays
        for (const char of word) {
            accumulated += char;
            yield { type: "text", text: char };
            await sleep(Math.random() * 50 + 10); // 10-60ms per character
        }
        
        // Add space after word (unless it's the last word)
        if (!isLastWord) {
            accumulated += " ";
            yield { type: "text", text: " " };
        }
        
        // Word pause
        await sleep(delayPerWord * (0.5 + Math.random()));
    }
    
    const usage: LLMUsage = {
        promptTokens: 20,
        completionTokens: Math.ceil(accumulated.length / 4),
        totalTokens: 0,
        cost: 0,
    };
    usage.totalTokens = usage.promptTokens + usage.completionTokens;
    
    yield { type: "done", usage, finishReason: "stop" };
}

/**
 * Create a stream with tool calls
 */
export async function* createToolCallStream(
    preText: string,
    toolCalls: Array<{ name: string; arguments: any; result?: any }>,
    postText: string,
): AsyncGenerator<LLMStreamEvent> {
    yield { type: "start", metadata: {} };
    
    // Stream pre-text
    for (const chunk of chunkText(preText, 5)) {
        yield { type: "text", text: chunk };
        await sleep(50);
    }
    
    // Stream tool calls
    for (const toolCall of toolCalls) {
        yield {
            type: "tool_call",
            toolCall: {
                id: `call_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
                type: "function",
                function: {
                    name: toolCall.name,
                    arguments: JSON.stringify(toolCall.arguments),
                },
            },
        };
        await sleep(100);
        
        // Simulate tool result if provided
        if (toolCall.result !== undefined) {
            yield {
                type: "tool_result",
                toolCallId: `call_${Date.now()}`,
                result: toolCall.result,
            };
            await sleep(100);
        }
    }
    
    // Stream post-text
    for (const chunk of chunkText(postText, 5)) {
        yield { type: "text", text: chunk };
        await sleep(50);
    }
    
    yield {
        type: "done",
        usage: {
            promptTokens: 30,
            completionTokens: Math.ceil((preText.length + postText.length) / 4) + toolCalls.length * 20,
            totalTokens: 0,
            cost: 0,
        },
        finishReason: "stop",
    };
}

/**
 * Create streaming response from a mock response
 */
export async function* streamFromResponse(
    mockResult: MockFactoryResult,
    chunkSize = 5,
): AsyncGenerator<LLMStreamEvent> {
    yield { type: "start", metadata: { model: mockResult.response.model } };
    
    // Stream content
    if (mockResult.response.content) {
        for (const chunk of chunkText(mockResult.response.content, chunkSize)) {
            yield { type: "text", text: chunk };
            await sleep(30);
        }
    }
    
    // Stream reasoning
    if (mockResult.response.reasoning) {
        for (const chunk of chunkText(mockResult.response.reasoning, chunkSize)) {
            yield { type: "reasoning", reasoning: chunk };
            await sleep(30);
        }
    }
    
    // Stream tool calls
    if (mockResult.response.toolCalls) {
        for (const toolCall of mockResult.response.toolCalls) {
            yield { type: "tool_call", toolCall };
            await sleep(100);
        }
    }
    
    yield {
        type: "done",
        usage: mockResult.usage || {
            promptTokens: 0,
            completionTokens: mockResult.response.tokensUsed,
            totalTokens: mockResult.response.tokensUsed,
            cost: 0,
        },
        finishReason: mockResult.response.finishReason,
    };
}

/**
 * Helper functions
 */
function normalizeChunks(chunks: string[] | StreamChunk[]): StreamChunk[] {
    if (chunks.length === 0) return [];
    
    if (typeof chunks[0] === "string") {
        return (chunks as string[]).map(c => ({
            type: "text",
            content: c,
        }));
    }
    
    return chunks as StreamChunk[];
}

function normalizeDelays(
    delay: number | number[] | undefined,
    chunkCount: number,
): number[] {
    if (!delay) return new Array(chunkCount).fill(0);
    
    if (typeof delay === "number") {
        return new Array(chunkCount).fill(delay);
    }
    
    // Extend array if needed
    const result = [...delay];
    while (result.length < chunkCount) {
        result.push(delay[delay.length - 1] || 0);
    }
    
    return result;
}

function chunkText(text: string, chunkSize: number): string[] {
    const chunks: string[] = [];
    const words = text.split(" ");
    
    for (let i = 0; i < words.length; i += chunkSize) {
        const chunk = words.slice(i, i + chunkSize).join(" ");
        if (i + chunkSize < words.length) {
            chunks.push(chunk + " ");
        } else {
            chunks.push(chunk);
        }
    }
    
    return chunks;
}

function sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
}
