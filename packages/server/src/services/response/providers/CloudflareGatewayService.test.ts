import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { LlmServiceId, LATEST_CONFIG_VERSION, type MessageState } from "@vrooli/shared";
import { CloudflareGatewayService } from "./CloudflareGatewayService.js";
import { AIServiceErrorType } from "../registry.js";
import { TokenEstimatorType } from "../tokenTypes.js";

// Mock the logger
vi.mock("../../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
    },
}));

// Mock fetch globally
const mockFetch = vi.fn();
global.fetch = mockFetch;

describe("CloudflareGatewayService", () => {
    let service: CloudflareGatewayService;
    let mockAbortController: AbortController;

    beforeEach(() => {
        vi.clearAllMocks();
        mockAbortController = new AbortController();
        service = new CloudflareGatewayService({
            apiToken: "test-token",
            gatewayUrl: "https://test-gateway.com/v1",
            accountId: "test-account-id",
            defaultModel: "@cf/openai/gpt-4o-mini",
        });
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("constructor", () => {
        it("should initialize with provided options", () => {
            expect(service.defaultModel).toBe("@cf/openai/gpt-4o-mini");
            expect(service.__id).toBe(LlmServiceId.CloudflareGateway);
            expect(service.featureFlags.supportsStatefulConversations).toBe(false);
        });

        it("should initialize with default values", () => {
            const defaultService = new CloudflareGatewayService();
            expect(defaultService.defaultModel).toBe("@cf/openai/gpt-4o-mini");
        });

        it("should use environment variables", () => {
            process.env.CLOUDFLARE_GATEWAY_TOKEN = "env-token";
            process.env.CLOUDFLARE_GATEWAY_URL = "https://env-gateway.com/v1";
            process.env.CLOUDFLARE_ACCOUNT_ID = "env-account";

            const envService = new CloudflareGatewayService();
            expect(envService.defaultModel).toBe("@cf/openai/gpt-4o-mini");

            delete process.env.CLOUDFLARE_GATEWAY_TOKEN;
            delete process.env.CLOUDFLARE_GATEWAY_URL;
            delete process.env.CLOUDFLARE_ACCOUNT_ID;
        });

        it("should warn when required configuration is missing", async () => {
            const { logger } = await import("../../../events/logger.js");
            new CloudflareGatewayService({ apiToken: "", accountId: "" });
            
            expect(logger.warn).toHaveBeenCalledWith(
                "[CloudflareGatewayService] No API token configured - service will not function",
            );
            expect(logger.warn).toHaveBeenCalledWith(
                "[CloudflareGatewayService] No account ID configured - service will not function",
            );
        });
    });

    describe("supportsModel", () => {
        it("should return true for Cloudflare Gateway models", async () => {
            const result = await service.supportsModel("@cf/openai/gpt-4o");
            expect(result).toBe(true);
        });

        it("should return false for non-Cloudflare Gateway models", async () => {
            const result = await service.supportsModel("gpt-4o");
            expect(result).toBe(false);
        });

        it("should return false for empty model", async () => {
            const result = await service.supportsModel("");
            expect(result).toBe(false);
        });
    });

    describe("estimateTokens", () => {
        it("should estimate tokens correctly", () => {
            const result = service.estimateTokens({ aiModel: service.defaultModel, text: "Hello world" });
            
            expect(result.tokens).toBe(6); // DefaultTokenEstimator: ceil(11 bytes / 2) = 6
            expect(result.estimationModel).toBe(TokenEstimatorType.Default);
            expect(result.encoding).toBe("default");
        });

        it("should handle empty text", () => {
            const result = service.estimateTokens({ aiModel: service.defaultModel, text: "" });
            expect(result.tokens).toBe(0);
        });

        it("should handle long text", () => {
            const longText = "A".repeat(1000);
            const result = service.estimateTokens({ aiModel: service.defaultModel, text: longText });
            expect(result.tokens).toBe(500); // DefaultTokenEstimator: ceil(1000 bytes / 2) = 500
        });
    });

    describe("generateContext", () => {
        it("should generate context with system message", () => {
            const messages = [
                { id: "msg-1", text: "Hello", config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, language: "en", createdAt: new Date("2024-01-01T00:00:00Z") },
                { id: "msg-2", text: "Hi there", config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "assistant" }, language: "en", createdAt: new Date("2024-01-01T00:01:00Z") },
            ] as unknown as MessageState[];

            const context = service.generateContext(messages, "You are a helpful assistant");

            expect(context).toHaveLength(3);
            expect(context[0]).toEqual({
                role: "system",
                content: "You are a helpful assistant",
            });
            expect(context[1]).toEqual({
                role: "user",
                content: "Hello",
            });
            expect(context[2]).toEqual({
                role: "assistant",
                content: "Hi there",
            });
        });

        it("should generate context without system message", () => {
            const messages = [
                { id: "msg-1", text: "Hello", config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, language: "en", createdAt: new Date("2024-01-01T00:00:00Z") },
            ] as unknown as MessageState[];

            const context = service.generateContext(messages);

            expect(context).toHaveLength(1);
            expect(context[0]).toEqual({
                role: "user",
                content: "Hello",
            });
        });

        it("should filter out invalid roles", () => {
            const messages = [
                { id: "msg-1", text: "Hello", config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, language: "en", createdAt: new Date("2024-01-01T00:00:00Z") },
                { id: "msg-2", text: "Invalid", config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "invalid" as any }, language: "en", createdAt: new Date("2024-01-01T00:01:00Z") },
                { id: "msg-3", text: "Hi there", config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "assistant" }, language: "en", createdAt: new Date("2024-01-01T00:02:00Z") },
            ] as unknown as MessageState[];

            const context = service.generateContext(messages);

            expect(context).toHaveLength(2);
            expect(context[0].role).toBe("user");
            expect(context[1].role).toBe("assistant");
        });

        it("should handle messages with missing text", () => {
            const messages = [
                { id: "msg-1", text: undefined, config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, language: "en", createdAt: new Date("2024-01-01T00:00:00Z") },
                { id: "msg-2", text: "Hi there", config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "assistant" }, language: "en", createdAt: new Date("2024-01-01T00:01:00Z") },
            ] as unknown as MessageState[];

            const context = service.generateContext(messages);

            expect(context).toHaveLength(2);
            expect(context[0].content).toBe("");
            expect(context[1].content).toBe("Hi there");
        });
    });

    describe("generateResponseStreaming", () => {
        it("should generate streaming response successfully", async () => {
            // Mock streaming response
            const mockResponseBody = 
                "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n" +
                "data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n" +
                "data: {\"choices\":[{\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5}}\n";

            const mockReader = {
                read: vi.fn()
                    .mockResolvedValueOnce({
                        done: false,
                        value: new TextEncoder().encode(mockResponseBody),
                    })
                    .mockResolvedValueOnce({ done: true }),
                releaseLock: vi.fn(),
            };

            mockFetch.mockResolvedValueOnce({
                ok: true,
                body: { getReader: () => mockReader },
                headers: new Headers(),
            });

            const options = {
                model: "@cf/openai/gpt-4o",
                input: [{ id: "msg-1", text: "Hello", config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, language: "en", createdAt: new Date("2024-01-01T00:00:00Z") }] as unknown as MessageState[],
                systemMessage: "You are helpful",
                maxTokens: 100,
                signal: mockAbortController.signal,
                tools: [],
            };

            const events = [];
            for await (const event of service.generateResponseStreaming(options)) {
                events.push(event);
            }

            expect(events).toHaveLength(3);
            expect(events[0]).toEqual({ type: "text", content: "Hello" });
            expect(events[1]).toEqual({ type: "text", content: " world" });
            expect(events[2]).toEqual({ type: "done", cost: expect.any(Number) });
            expect(events[2].cost).toBeGreaterThan(0);
        });

        it("should handle [DONE] message", async () => {
            const mockResponseBody = 
                "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n" +
                "data: {\"choices\":[{\"finish_reason\":\"stop\"}]}\n";

            const mockReader = {
                read: vi.fn()
                    .mockResolvedValueOnce({
                        done: false,
                        value: new TextEncoder().encode(mockResponseBody),
                    })
                    .mockResolvedValueOnce({ done: true }),
                releaseLock: vi.fn(),
            };

            mockFetch.mockResolvedValueOnce({
                ok: true,
                body: { getReader: () => mockReader },
                headers: new Headers(),
            });

            const options = {
                model: "@cf/openai/gpt-4o",
                input: [{ id: "msg-1", text: "Hello", config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, language: "en", createdAt: new Date("2024-01-01T00:00:00Z") }] as unknown as MessageState[],
                tools: [],
            };

            const events = [];
            for await (const event of service.generateResponseStreaming(options)) {
                events.push(event);
            }

            expect(events).toHaveLength(2);
            expect(events[0]).toEqual({ type: "text", content: "Hello" });
            expect(events[1]).toEqual({ type: "done", cost: expect.any(Number) });
        });

        it("should throw error for unsupported model", async () => {
            const options = {
                model: "gpt-4o", // Not a Cloudflare Gateway model
                input: [{ id: "msg-1", text: "Hello", config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, language: "en", createdAt: new Date("2024-01-01T00:00:00Z") }] as unknown as MessageState[],
                tools: [],
            };

            await expect(async () => {
                for await (const _ of service.generateResponseStreaming(options)) {
                    // Should not reach here
                }
            }).rejects.toThrow("Model gpt-4o not supported by Cloudflare Gateway");
        });

        it("should handle HTTP errors", async () => {
            mockFetch.mockResolvedValueOnce({
                ok: false,
                status: 401,
                statusText: "Unauthorized",
            });

            const options = {
                model: "@cf/openai/gpt-4o",
                input: [{ id: "msg-1", text: "Hello", config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, language: "en", createdAt: new Date("2024-01-01T00:00:00Z") }] as unknown as MessageState[],
                tools: [],
            };

            await expect(async () => {
                for await (const _ of service.generateResponseStreaming(options)) {
                    // Should not reach here
                }
            }).rejects.toThrow("HTTP 401: Unauthorized");
        });

        it("should handle missing response body", async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                body: null,
            });

            const options = {
                model: "@cf/openai/gpt-4o",
                input: [{ id: "msg-1", text: "Hello", config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, language: "en", createdAt: new Date("2024-01-01T00:00:00Z") }] as unknown as MessageState[],
                tools: [],
            };

            await expect(async () => {
                for await (const _ of service.generateResponseStreaming(options)) {
                    // Should not reach here
                }
            }).rejects.toThrow("No response body available");
        });

        it("should handle malformed JSON gracefully", async () => {
            const mockResponseBody = 
                "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n" +
                "data: {invalid json}\n" +
                "data: {\"choices\":[{\"finish_reason\":\"stop\"}]}\n";

            const mockReader = {
                read: vi.fn()
                    .mockResolvedValueOnce({
                        done: false,
                        value: new TextEncoder().encode(mockResponseBody),
                    })
                    .mockResolvedValueOnce({ done: true }),
                releaseLock: vi.fn(),
            };

            mockFetch.mockResolvedValueOnce({
                ok: true,
                body: { getReader: () => mockReader },
                headers: new Headers(),
            });

            const options = {
                model: "@cf/openai/gpt-4o",
                input: [{ id: "msg-1", text: "Hello", config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, language: "en", createdAt: new Date("2024-01-01T00:00:00Z") }] as unknown as MessageState[],
                tools: [],
            };

            const events = [];
            for await (const event of service.generateResponseStreaming(options)) {
                events.push(event);
            }

            expect(events).toHaveLength(2); // Should skip malformed JSON
            expect(events[0]).toEqual({ type: "text", content: "Hello" });
            expect(events[1]).toEqual({ type: "done", cost: expect.any(Number) });
        });

        it("should handle tool calls in streaming response", async () => {
            const mockResponseBody = 
                "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"id\":\"call_123\",\"function\":{\"name\":\"search\",\"arguments\":\"{\\\"query\\\":\\\"test\\\"}\"}}]}}]}\n" +
                "data: {\"choices\":[{\"finish_reason\":\"tool_calls\"}],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5}}\n";

            const mockReader = {
                read: vi.fn()
                    .mockResolvedValueOnce({
                        done: false,
                        value: new TextEncoder().encode(mockResponseBody),
                    })
                    .mockResolvedValueOnce({ done: true }),
                releaseLock: vi.fn(),
            };

            mockFetch.mockResolvedValueOnce({
                ok: true,
                body: { getReader: () => mockReader },
                headers: new Headers(),
            });

            const options = {
                model: "@cf/openai/gpt-4o",
                input: [{ id: "msg-1", text: "Hello", config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, language: "en", createdAt: new Date("2024-01-01T00:00:00Z") }] as unknown as MessageState[],
                tools: [{
                    type: "function" as const,
                    function: {
                        name: "search",
                        description: "Search function",
                    },
                }],
            };

            const events = [];
            for await (const event of service.generateResponseStreaming(options)) {
                events.push(event);
            }

            expect(events).toHaveLength(2);
            expect(events[0]).toEqual({
                type: "function_call",
                name: "search",
                arguments: { query: "test" },
                callId: "call_123",
            });
            expect(events[1]).toEqual({ type: "done", cost: expect.any(Number) });
        });

        it("should handle malformed tool call arguments gracefully", async () => {
            const mockResponseBody = 
                "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"id\":\"call_123\",\"function\":{\"name\":\"search\",\"arguments\":\"{invalid json}\"}}]}}]}\n" +
                "data: {\"choices\":[{\"finish_reason\":\"tool_calls\"}]}\n";

            const mockReader = {
                read: vi.fn()
                    .mockResolvedValueOnce({
                        done: false,
                        value: new TextEncoder().encode(mockResponseBody),
                    })
                    .mockResolvedValueOnce({ done: true }),
                releaseLock: vi.fn(),
            };

            mockFetch.mockResolvedValueOnce({
                ok: true,
                body: { getReader: () => mockReader },
                headers: new Headers(),
            });

            const options = {
                model: "@cf/openai/gpt-4o",
                input: [{ id: "msg-1", text: "Hello", config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, language: "en", createdAt: new Date("2024-01-01T00:00:00Z") }] as unknown as MessageState[],
                tools: [{
                    type: "function" as const,
                    function: {
                        name: "search",
                        description: "Search function",
                    },
                }],
            };

            const events = [];
            for await (const event of service.generateResponseStreaming(options)) {
                events.push(event);
            }

            expect(events).toHaveLength(2);
            expect(events[0]).toEqual({
                type: "function_call",
                name: "search",
                arguments: {}, // Should fallback to empty object for malformed JSON
                callId: "call_123",
            });
            expect(events[1]).toEqual({ type: "done", cost: expect.any(Number) });
        });

        it("should handle AI Gateway cache hit (zero cost)", async () => {
            const mockResponseBody = "data: [DONE]\n";

            const mockReader = {
                read: vi.fn()
                    .mockResolvedValueOnce({
                        done: false,
                        value: new TextEncoder().encode(mockResponseBody),
                    })
                    .mockResolvedValueOnce({ done: true }),
                releaseLock: vi.fn(),
            };

            const mockHeaders = new Headers();
            mockHeaders.set("cf-aig-log-id", "12345");
            mockHeaders.set("cf-aig-cache-status", "HIT");

            mockFetch.mockResolvedValueOnce({
                ok: true,
                body: { getReader: () => mockReader },
                headers: mockHeaders,
            });

            const options = {
                model: "@cf/openai/gpt-4o",
                input: [{ id: "msg-1", text: "Hello", config: { __version: LATEST_CONFIG_VERSION, resources: [], role: "user" }, language: "en", createdAt: new Date("2024-01-01T00:00:00Z") }] as unknown as MessageState[],
                tools: [],
            };

            const events = [];
            for await (const event of service.generateResponseStreaming(options)) {
                events.push(event);
            }

            expect(events).toHaveLength(1);
            expect(events[0]).toEqual({ type: "done", cost: 0 }); // Cache hit = zero cost
        });
    });

    describe("getContextSize", () => {
        it("should return context size for known model", () => {
            const size = service.getContextSize("@cf/openai/gpt-4o");
            expect(size).toBe(128000);
        });

        it("should return default context size for unknown model", () => {
            const size = service.getContextSize("@cf/unknown/model");
            expect(size).toBe(8192);
        });

        it("should return default context size when no model specified", () => {
            const size = service.getContextSize();
            expect(size).toBe(128000); // Default model is @cf/openai/gpt-4o-mini
        });
    });

    describe("getModelInfo", () => {
        it("should return model info for supported models", () => {
            const modelInfo = service.getModelInfo();
            
            expect(modelInfo).toHaveProperty("@cf/openai/gpt-4o");
            expect(modelInfo).toHaveProperty("@cf/openai/gpt-4o-mini");
            expect(modelInfo).toHaveProperty("@cf/anthropic/claude-3-5-sonnet");
            expect(modelInfo).toHaveProperty("@cf/meta/llama-3-8b-instruct");

            const gpt4oInfo = modelInfo["@cf/openai/gpt-4o"];
            expect(gpt4oInfo).toMatchObject({
                enabled: true,
                name: "CF_GPT4o",
                contextWindow: 128000,
                maxOutputTokens: 4096,
                inputCost: 250,
                outputCost: 1000,
                supportsReasoning: false,
            });
        });

        it("should include all required model properties", () => {
            const modelInfo = service.getModelInfo();
            const firstModel = Object.values(modelInfo)[0];
            
            expect(firstModel).toHaveProperty("enabled");
            expect(firstModel).toHaveProperty("name");
            expect(firstModel).toHaveProperty("descriptionShort");
            expect(firstModel).toHaveProperty("inputCost");
            expect(firstModel).toHaveProperty("outputCost");
            expect(firstModel).toHaveProperty("contextWindow");
            expect(firstModel).toHaveProperty("maxOutputTokens");
            expect(firstModel).toHaveProperty("features");
            expect(firstModel).toHaveProperty("supportsReasoning");
        });
    });

    describe("getMaxOutputTokens", () => {
        it("should return max output tokens for known model", () => {
            const maxTokens = service.getMaxOutputTokens("@cf/openai/gpt-4o-mini");
            expect(maxTokens).toBe(16384);
        });

        it("should return default max output tokens for unknown model", () => {
            const maxTokens = service.getMaxOutputTokens("@cf/unknown/model");
            expect(maxTokens).toBe(4096);
        });

        it("should return default max output tokens when no model specified", () => {
            const maxTokens = service.getMaxOutputTokens();
            expect(maxTokens).toBe(16384); // Default model is @cf/openai/gpt-4o-mini
        });
    });

    describe("getMaxOutputTokensRestrained", () => {
        it("should calculate restrained output tokens for known model", () => {
            const params = {
                maxCredits: BigInt(100000),
                model: "@cf/openai/gpt-4o-mini",
                inputTokens: 100,
            };

            const maxTokens = service.getMaxOutputTokensRestrained(params);
            expect(maxTokens).toBeGreaterThan(0);
            expect(maxTokens).toBeLessThanOrEqual(16384);
        });

        it("should return 0 when input cost exceeds max credits", () => {
            const params = {
                maxCredits: BigInt(1),
                model: "@cf/openai/gpt-4o",
                inputTokens: 10000,
            };

            const maxTokens = service.getMaxOutputTokensRestrained(params);
            expect(maxTokens).toBe(0);
        });

        it("should return default when model info not available", () => {
            const params = {
                maxCredits: BigInt(100000),
                model: "@cf/unknown/model",
                inputTokens: 100,
            };

            const maxTokens = service.getMaxOutputTokensRestrained(params);
            expect(maxTokens).toBe(4096); // Default max output tokens
        });
    });

    describe("getResponseCost", () => {
        it("should calculate response cost for known model", () => {
            const params = {
                model: "@cf/openai/gpt-4o",
                usage: { input: 100, output: 50 },
            };

            const cost = service.getResponseCost(params);
            expect(cost).toBe(250 * 100 + 1000 * 50); // inputCost * input + outputCost * output
        });

        it("should use fallback cost for unknown model", () => {
            const params = {
                model: "@cf/unknown/model",
                usage: { input: 100, output: 50 },
            };

            const cost = service.getResponseCost(params);
            expect(cost).toBe(100 * 50 + 50 * 150); // Fallback pricing
        });
    });

    describe("getEstimationInfo", () => {
        it("should return estimation info", () => {
            const info = service.getEstimationInfo();
            expect(info).toEqual({
                estimationModel: TokenEstimatorType.Default,
                encoding: "cl100k_base",
            });
        });

        it("should return same info regardless of model", () => {
            const info1 = service.getEstimationInfo("@cf/openai/gpt-4o");
            const info2 = service.getEstimationInfo("@cf/anthropic/claude-3-haiku");
            expect(info1).toEqual(info2);
        });
    });

    describe("getModel", () => {
        it("should return Cloudflare Gateway model as is", () => {
            const model = service.getModel("@cf/openai/gpt-4o");
            expect(model).toBe("@cf/openai/gpt-4o");
        });

        it("should map common model names to Cloudflare Gateway equivalents", () => {
            expect(service.getModel("gpt-4o")).toBe("@cf/openai/gpt-4o");
            expect(service.getModel("gpt-4o-mini")).toBe("@cf/openai/gpt-4o-mini");
            expect(service.getModel("claude-3-5-sonnet")).toBe("@cf/anthropic/claude-3-5-sonnet");
            expect(service.getModel("llama-3-8b")).toBe("@cf/meta/llama-3-8b-instruct");
        });

        it("should return default model for unknown model", () => {
            const model = service.getModel("unknown-model");
            expect(model).toBe("@cf/openai/gpt-4o-mini");
        });

        it("should return default model when no model provided", () => {
            const model = service.getModel();
            expect(model).toBe("@cf/openai/gpt-4o-mini");
        });
    });

    describe("getErrorType", () => {
        it("should classify authentication errors", () => {
            expect(service.getErrorType(new Error("unauthorized"))).toBe(AIServiceErrorType.Authentication);
            expect(service.getErrorType(new Error("authentication failed"))).toBe(AIServiceErrorType.Authentication);
            expect(service.getErrorType(new Error("HTTP 401"))).toBe(AIServiceErrorType.Authentication);
        });

        it("should classify rate limit errors", () => {
            expect(service.getErrorType(new Error("rate limit exceeded"))).toBe(AIServiceErrorType.RateLimit);
            expect(service.getErrorType(new Error("too many requests"))).toBe(AIServiceErrorType.RateLimit);
            expect(service.getErrorType(new Error("HTTP 429"))).toBe(AIServiceErrorType.RateLimit);
        });

        it("should classify overload errors", () => {
            expect(service.getErrorType(new Error("server overloaded"))).toBe(AIServiceErrorType.Overloaded);
            expect(service.getErrorType(new Error("at capacity"))).toBe(AIServiceErrorType.Overloaded);
            expect(service.getErrorType(new Error("HTTP 503"))).toBe(AIServiceErrorType.Overloaded);
        });

        it("should classify invalid request errors", () => {
            expect(service.getErrorType(new Error("bad request"))).toBe(AIServiceErrorType.InvalidRequest);
            expect(service.getErrorType(new Error("invalid parameters"))).toBe(AIServiceErrorType.InvalidRequest);
            expect(service.getErrorType(new Error("HTTP 400"))).toBe(AIServiceErrorType.InvalidRequest);
        });

        it("should default to API error for unknown errors", () => {
            expect(service.getErrorType(new Error("unknown error"))).toBe(AIServiceErrorType.ApiError);
            expect(service.getErrorType("string error")).toBe(AIServiceErrorType.ApiError);
            expect(service.getErrorType(null)).toBe(AIServiceErrorType.ApiError);
        });
    });

    describe("safeInputCheck", () => {
        it("should consider safe input safe", async () => {
            const result = await service.safeInputCheck("normal conversation text");
            expect(result).toEqual({ cost: 0, isSafe: true });
        });

        it("should flag harmful content as unsafe", async () => {
            const result = await service.safeInputCheck("how to kill someone");
            expect(result).toEqual({ cost: 0, isSafe: false });
        });

        it("should flag excessive input length as unsafe", async () => {
            const longInput = "a".repeat(100000); // Exceeds MAX_INPUT_LENGTH
            const result = await service.safeInputCheck(longInput);
            expect(result).toEqual({ cost: 0, isSafe: false });
        });

        it("should allow reasonable content", async () => {
            const result = await service.safeInputCheck("How do I create a new user account?");
            expect(result).toEqual({ cost: 0, isSafe: true });
        });
    });

    describe("getNativeToolCapabilities", () => {
        it("should return function calling capabilities", () => {
            const tools = service.getNativeToolCapabilities();
            expect(tools).toEqual([
                { 
                    name: "function", 
                    description: "Execute custom functions with embedded execution in Workers AI", 
                },
            ]);
        });
    });

    describe("calculateCloudflareGatewayCost", () => {
        it("should calculate cost using model info", () => {
            // This is a private method, so we'll test it through getResponseCost
            const params = {
                model: "@cf/openai/gpt-4o",
                usage: { input: 1000, output: 500 },
            };

            const cost = service.getResponseCost(params);
            expect(cost).toBe(250 * 1000 + 1000 * 500); // Based on model info
        });
    });
});
