import { vi } from "vitest";
import { type MessageState } from "@vrooli/shared";

// Use constant directly since LATEST_CONFIG_VERSION is not exported from @vrooli/shared
const LATEST_CONFIG_VERSION = "1.0";

/**
 * Common test utilities for AI service testing
 */

/**
 * Mock fetch response for streaming API calls
 */
export const createMockStreamingResponse = (chunks: string[], headers?: Record<string, string>) => {
    const mockHeaders = new Headers();
    if (headers) {
        Object.entries(headers).forEach(([key, value]) => {
            mockHeaders.set(key, value);
        });
    }

    const mockReader = {
        read: vi.fn(),
        releaseLock: vi.fn(),
    };

    // Set up read mock to return chunks sequentially
    chunks.forEach((chunk, index) => {
        mockReader.read.mockResolvedValueOnce({
            done: false,
            value: new TextEncoder().encode(chunk),
        });
    });

    // Final read call returns done: true
    mockReader.read.mockResolvedValueOnce({ done: true });

    return {
        ok: true,
        body: { getReader: () => mockReader },
        headers: mockHeaders,
    };
};

/**
 * Mock fetch response for non-streaming API calls
 */
export const createMockResponse = (data: any, status = 200, headers?: Record<string, string>) => {
    const mockHeaders = new Headers();
    if (headers) {
        Object.entries(headers).forEach(([key, value]) => {
            mockHeaders.set(key, value);
        });
    }

    return {
        ok: status >= 200 && status < 300,
        status,
        statusText: status === 200 ? "OK" : "Error",
        headers: mockHeaders,
        json: vi.fn().mockResolvedValue(data),
    };
};

/**
 * Mock fetch error response
 */
export const createMockErrorResponse = (status: number, statusText: string) => {
    return {
        ok: false,
        status,
        statusText,
        headers: new Headers(),
    };
};

/**
 * Create sample message states for testing
 */
export const createSampleMessages = (count = 3): MessageState[] => {
    const messages: MessageState[] = [];
    
    for (let i = 0; i < count; i++) {
        const isUser = i % 2 === 0;
        messages.push({
            id: `msg-${i + 1}`,
            createdAt: new Date().toISOString(),
            language: "en",
            text: isUser ? `User message ${i + 1}` : `Assistant message ${i + 1}`,
            config: {
                __version: LATEST_CONFIG_VERSION,
                resources: [],
                role: isUser ? "user" : "assistant",
            },
        });
    }
    
    return messages;
};

/**
 * Create sample message with tool calls
 */
export const createMessageWithToolCalls = (toolCalls: any[]): MessageState => {
    return {
        id: "msg-tool-call",
        createdAt: new Date().toISOString(),
        language: "en",
        text: "I'll use some tools to help you.",
        config: {
            __version: LATEST_CONFIG_VERSION,
            resources: [],
            role: "assistant",
            toolCalls,
        },
    };
};

/**
 * Create sample tool call
 */
export const createSampleToolCall = (name: string, args: Record<string, any>, result?: any) => {
    return {
        id: `tool-${Date.now()}-${Math.random()}`,
        function: {
            name,
            arguments: JSON.stringify(args),
        },
        result,
    };
};

/**
 * Mock AbortController for testing
 */
export const createMockAbortController = () => {
    const controller = {
        signal: {
            aborted: false,
            addEventListener: vi.fn(),
            removeEventListener: vi.fn(),
        },
        abort: vi.fn(),
    };
    
    return controller;
};

/**
 * Mock environment variables
 */
export const mockEnvVars = (vars: Record<string, string>) => {
    Object.entries(vars).forEach(([key, value]) => {
        process.env[key] = value;
    });
};

/**
 * Clean up environment variables
 */
export const cleanupEnvVars = (keys: string[]) => {
    keys.forEach(key => {
        delete process.env[key];
    });
};

/**
 * Create mock logger
 */
export const createMockLogger = () => {
    return {
        info: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
        debug: vi.fn(),
    };
};

/**
 * SSE (Server-Sent Events) response helpers
 */
export const createSSEChunk = (data: any, event?: string) => {
    let chunk = `data: ${JSON.stringify(data)}\n`;
    if (event) {
        chunk = `event: ${event}\n${chunk}`;
    }
    return chunk;
};

/**
 * Create OpenAI-style streaming response chunks
 */
export const createOpenAIStreamingChunks = (contents: string[], finishReason?: string) => {
    const chunks: string[] = [];
    
    // Add content chunks
    contents.forEach(content => {
        chunks.push(createSSEChunk({
            choices: [{
                delta: { content },
                index: 0,
            }],
        }));
    });
    
    // Add finish chunk
    if (finishReason) {
        chunks.push(createSSEChunk({
            choices: [{
                delta: {},
                index: 0,
                finish_reason: finishReason,
            }],
            usage: {
                prompt_tokens: 10,
                completion_tokens: contents.length,
                total_tokens: 10 + contents.length,
            },
        }));
    }
    
    return chunks;
};

/**
 * Create Ollama-style streaming response chunks
 */
export const createOllamaStreamingChunks = (contents: string[]) => {
    const chunks: string[] = [];
    
    // Add content chunks
    contents.forEach(content => {
        chunks.push(JSON.stringify({
            message: { content },
            done: false,
        }) + "\n");
    });
    
    // Add done chunk
    chunks.push(JSON.stringify({
        message: { content: "" },
        done: true,
    }) + "\n");
    
    return chunks;
};

/**
 * Assert that events match expected pattern
 */
export const assertStreamingEvents = (events: any[], expectedTypes: string[]) => {
    expect(events).toHaveLength(expectedTypes.length);
    
    events.forEach((event, index) => {
        expect(event.type).toBe(expectedTypes[index]);
        
        if (event.type === "text") {
            expect(event.content).toBeDefined();
            expect(typeof event.content).toBe("string");
        } else if (event.type === "done") {
            expect(event.cost).toBeDefined();
            expect(typeof event.cost).toBe("number");
            expect(event.cost).toBeGreaterThanOrEqual(0);
        } else if (event.type === "function_call") {
            expect(event.name).toBeDefined();
            expect(event.arguments).toBeDefined();
            expect(event.callId).toBeDefined();
        }
    });
};

/**
 * Test helper for model support validation
 */
export const testModelSupport = async (service: any, supportedModels: string[], unsupportedModels: string[]) => {
    // Test supported models
    for (const model of supportedModels) {
        const result = await service.supportsModel(model);
        expect(result).toBe(true);
    }
    
    // Test unsupported models
    for (const model of unsupportedModels) {
        const result = await service.supportsModel(model);
        expect(result).toBe(false);
    }
};

/**
 * Test helper for model info validation
 */
export const validateModelInfo = (modelInfo: any) => {
    expect(modelInfo).toBeDefined();
    expect(typeof modelInfo).toBe("object");
    
    Object.values(modelInfo).forEach((info: any) => {
        expect(info).toHaveProperty("enabled");
        expect(info).toHaveProperty("name");
        expect(info).toHaveProperty("descriptionShort");
        expect(info).toHaveProperty("inputCost");
        expect(info).toHaveProperty("outputCost");
        expect(info).toHaveProperty("contextWindow");
        expect(info).toHaveProperty("maxOutputTokens");
        expect(info).toHaveProperty("features");
        expect(info).toHaveProperty("supportsReasoning");
        
        expect(typeof info.enabled).toBe("boolean");
        expect(typeof info.name).toBe("string");
        expect(typeof info.descriptionShort).toBe("string");
        expect(typeof info.inputCost).toBe("number");
        expect(typeof info.outputCost).toBe("number");
        expect(typeof info.contextWindow).toBe("number");
        expect(typeof info.maxOutputTokens).toBe("number");
        expect(typeof info.features).toBe("object");
        expect(typeof info.supportsReasoning).toBe("boolean");
        
        expect(info.inputCost).toBeGreaterThanOrEqual(0);
        expect(info.outputCost).toBeGreaterThanOrEqual(0);
        expect(info.contextWindow).toBeGreaterThan(0);
        expect(info.maxOutputTokens).toBeGreaterThan(0);
    });
};

/**
 * Test helper for error type classification
 */
export const testErrorClassification = (service: any, errorMappings: Record<string, string>) => {
    Object.entries(errorMappings).forEach(([errorMessage, expectedType]) => {
        const error = new Error(errorMessage);
        const result = service.getErrorType(error);
        expect(result).toBe(expectedType);
    });
};

/**
 * Common test patterns for AI services
 */
export const commonServiceTests = {
    /**
     * Test basic service properties
     */
    testBasicProperties: (service: any, expectedId: string, expectedDefaultModel: string) => {
        expect(service.__id).toBe(expectedId);
        expect(service.defaultModel).toBe(expectedDefaultModel);
        expect(service.featureFlags).toBeDefined();
        expect(typeof service.featureFlags).toBe("object");
    },
    
    /**
     * Test token estimation
     */
    testTokenEstimation: (service: any) => {
        const result = service.estimateTokens({ aiModel: service.defaultModel, text: "Hello world" });
        expect(result).toHaveProperty("tokens");
        expect(result).toHaveProperty("estimationModel");
        expect(result).toHaveProperty("encoding");
        expect(typeof result.tokens).toBe("number");
        expect(result.tokens).toBeGreaterThan(0);
    },
    
    /**
     * Test context generation
     */
    testContextGeneration: (service: any) => {
        const messages = createSampleMessages(2);
        const context = service.generateContext(messages, "System prompt");
        
        expect(Array.isArray(context)).toBe(true);
        expect(context.length).toBeGreaterThan(0);
        
        // Should have system message as first item
        expect(context[0]).toHaveProperty("role", "system");
        expect(context[0]).toHaveProperty("content", "System prompt");
    },
    
    /**
     * Test cost calculation
     */
    testCostCalculation: (service: any, testModel: string) => {
        const params = {
            model: testModel,
            usage: { input: 100, output: 50 },
        };
        
        const cost = service.getResponseCost(params);
        expect(typeof cost).toBe("number");
        expect(cost).toBeGreaterThanOrEqual(0);
    },
    
    /**
     * Test safety check
     */
    testSafetyCheck: async (service: any) => {
        const result = await service.safeInputCheck("test input");
        expect(result).toHaveProperty("cost");
        expect(result).toHaveProperty("isSafe");
        expect(typeof result.cost).toBe("number");
        expect(typeof result.isSafe).toBe("boolean");
    },
};
