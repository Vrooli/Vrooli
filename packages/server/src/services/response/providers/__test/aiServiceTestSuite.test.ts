import { LlmServiceId } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AIServiceErrorType } from "../../registry.js";
import { CloudflareGatewayService } from "../CloudflareGatewayService.js";
import { LocalOllamaService } from "../LocalOllamaService.js";
import { OpenRouterService } from "../OpenRouterService.js";
import {
    assertStreamingEvents,
    cleanupEnvVars,
    commonServiceTests,
    createMockErrorResponse,
    createMockResponse,
    createMockStreamingResponse,
    createOllamaStreamingChunks,
    createOpenAIStreamingChunks,
    createSampleMessages,
    mockEnvVars,
    testErrorClassification,
    testModelSupport,
    validateModelInfo,
} from "./testUtils.js";

// Mock the logger
vi.mock("../../../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
    },
}));

// Mock fetch globally
const mockFetch = vi.fn();
global.fetch = mockFetch;

describe("AI Service Test Suite", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("LocalOllamaService Integration", () => {
        let service: LocalOllamaService;

        beforeEach(() => {
            service = new LocalOllamaService({
                baseUrl: "http://localhost:11434",
                defaultModel: "llama3.1:8b",
            });
        });

        it("should pass common service tests", () => {
            commonServiceTests.testBasicProperties(service, LlmServiceId.LocalOllama, "llama3.1:8b");
            commonServiceTests.testTokenEstimation(service);
            commonServiceTests.testContextGeneration(service);
            commonServiceTests.testCostCalculation(service, "llama3.1:8b");
        });

        it("should handle model support correctly", async () => {
            // Mock model refresh
            mockFetch.mockResolvedValueOnce(createMockResponse({
                models: [
                    { name: "llama3.1:8b" },
                    { name: "codellama:13b" },
                    { name: "mistral:7b" },
                ],
            }));

            await testModelSupport(
                service,
                ["llama3.1:8b", "codellama:13b", "mistral:7b"],
                ["gpt-4", "claude-3", "unknown:model"],
            );
        });

        it("should handle streaming responses", async () => {
            // Mock model support check
            mockFetch.mockResolvedValueOnce(createMockResponse({
                models: [{ name: "llama3.1:8b" }],
            }));

            // Mock streaming response
            const chunks = createOllamaStreamingChunks(["Hello", " world", "!"]);
            mockFetch.mockResolvedValueOnce(createMockStreamingResponse(chunks));

            const options = {
                model: "llama3.1:8b",
                input: createSampleMessages(1),
                tools: [],
            };

            const events = [];
            for await (const event of service.generateResponseStreaming(options)) {
                events.push(event);
            }

            assertStreamingEvents(events, ["text", "text", "text", "done"]);
        });

        it("should validate model info", async () => {
            // Mock model refresh
            mockFetch.mockResolvedValueOnce(createMockResponse({
                models: [{ name: "llama3.1:8b" }],
            }));

            await service.refreshAvailableModels();
            const modelInfo = service.getModelInfo();
            validateModelInfo(modelInfo);
        });

        it("should classify errors correctly", () => {
            testErrorClassification(service, {
                "unauthorized": AIServiceErrorType.Authentication,
                "rate limit": AIServiceErrorType.RateLimit,
                "overloaded": AIServiceErrorType.Overloaded,
                "bad request": AIServiceErrorType.InvalidRequest,
                "unknown error": AIServiceErrorType.ApiError,
            });
        });
    });

    describe("CloudflareGatewayService Integration", () => {
        let service: CloudflareGatewayService;

        beforeEach(() => {
            service = new CloudflareGatewayService({
                apiToken: "test-token",
                gatewayUrl: "https://test-gateway.com/v1",
                accountId: "test-account",
                defaultModel: "@cf/openai/gpt-4o-mini",
            });
        });

        it("should pass common service tests", () => {
            commonServiceTests.testBasicProperties(service, LlmServiceId.CloudflareGateway, "@cf/openai/gpt-4o-mini");
            commonServiceTests.testTokenEstimation(service);
            commonServiceTests.testContextGeneration(service);
            commonServiceTests.testCostCalculation(service, "@cf/openai/gpt-4o");
        });

        it("should handle model support correctly", async () => {
            await testModelSupport(
                service,
                ["@cf/openai/gpt-4o", "@cf/anthropic/claude-3-5-sonnet", "@cf/meta/llama-3-8b-instruct"],
                ["gpt-4o", "claude-3-5-sonnet", "unknown-model"],
            );
        });

        it("should handle streaming responses", async () => {
            const chunks = createOpenAIStreamingChunks(["Hello", " world", "!"], "stop");
            mockFetch.mockResolvedValueOnce(createMockStreamingResponse(chunks));

            const options = {
                model: "@cf/openai/gpt-4o",
                input: createSampleMessages(1),
                tools: [],
            };

            const events = [];
            for await (const event of service.generateResponseStreaming(options)) {
                events.push(event);
            }

            assertStreamingEvents(events, ["text", "text", "text", "done"]);
        });

        it("should validate model info", () => {
            const modelInfo = service.getModelInfo();
            validateModelInfo(modelInfo);
        });

        it("should classify errors correctly", () => {
            testErrorClassification(service, {
                "unauthorized": AIServiceErrorType.Authentication,
                "rate limit": AIServiceErrorType.RateLimit,
                "overloaded": AIServiceErrorType.Overloaded,
                "bad request": AIServiceErrorType.InvalidRequest,
                "unknown error": AIServiceErrorType.ApiError,
            });
        });
    });

    describe("OpenRouterService Integration", () => {
        let service: OpenRouterService;

        beforeEach(() => {
            service = new OpenRouterService({
                apiKey: "test-key",
                appName: "Test App",
                siteUrl: "https://test.com",
                defaultModel: "openai/gpt-4o-mini",
            });
        });

        it("should pass common service tests", () => {
            commonServiceTests.testBasicProperties(service, LlmServiceId.OpenRouter, "openai/gpt-4o-mini");
            commonServiceTests.testTokenEstimation(service);
            commonServiceTests.testContextGeneration(service);
            commonServiceTests.testCostCalculation(service, "openai/gpt-4o");
        });

        it("should handle model support correctly", async () => {
            await testModelSupport(
                service,
                ["openai/gpt-4o", "anthropic/claude-3.5-sonnet", "meta-llama/llama-3.1-8b-instruct"],
                ["gpt-4o", "claude-3.5-sonnet", "unknown-model"],
            );
        });

        it("should handle streaming responses", async () => {
            const chunks = createOpenAIStreamingChunks(["Hello", " world", "!"], "stop");
            const headers = { "x-ratelimit-cost": "0.001" };
            mockFetch.mockResolvedValueOnce(createMockStreamingResponse(chunks, headers));

            const options = {
                model: "openai/gpt-4o",
                input: createSampleMessages(1),
                tools: [],
            };

            const events = [];
            for await (const event of service.generateResponseStreaming(options)) {
                events.push(event);
            }

            assertStreamingEvents(events, ["text", "text", "text", "done"]);
        });

        it("should validate model info", () => {
            const modelInfo = service.getModelInfo();
            validateModelInfo(modelInfo);
        });

        it("should classify errors correctly", () => {
            testErrorClassification(service, {
                "unauthorized": AIServiceErrorType.Authentication,
                "rate limit": AIServiceErrorType.RateLimit,
                "overloaded": AIServiceErrorType.Overloaded,
                "bad request": AIServiceErrorType.InvalidRequest,
                "unknown error": AIServiceErrorType.ApiError,
            });
        });
    });

    describe("Cross-Service Compatibility Tests", () => {
        let localOllama: LocalOllamaService;
        let cloudflareGateway: CloudflareGatewayService;
        let openRouter: OpenRouterService;

        beforeEach(() => {
            localOllama = new LocalOllamaService();
            cloudflareGateway = new CloudflareGatewayService({
                apiToken: "test-token",
                accountId: "test-account",
            });
            openRouter = new OpenRouterService({
                apiKey: "test-key",
            });
        });

        it("should all implement required AIService methods", () => {
            const services = [localOllama, cloudflareGateway, openRouter];

            services.forEach(service => {
                expect(service.supportsModel).toBeDefined();
                expect(service.estimateTokens).toBeDefined();
                expect(service.generateContext).toBeDefined();
                expect(service.generateResponseStreaming).toBeDefined();
                expect(service.getContextSize).toBeDefined();
                expect(service.getModelInfo).toBeDefined();
                expect(service.getMaxOutputTokens).toBeDefined();
                expect(service.getMaxOutputTokensRestrained).toBeDefined();
                expect(service.getResponseCost).toBeDefined();
                expect(service.getEstimationInfo).toBeDefined();
                expect(service.getModel).toBeDefined();
                expect(service.getErrorType).toBeDefined();
                expect(service.safeInputCheck).toBeDefined();
                expect(service.getNativeToolCapabilities).toBeDefined();
            });
        });

        it("should have consistent method signatures", async () => {
            const services = [localOllama, cloudflareGateway, openRouter];

            for (const service of services) {
                // Test that safety check returns consistent structure
                await expect(service.safeInputCheck("test")).resolves.toMatchObject({
                    cost: expect.any(Number),
                    isSafe: expect.any(Boolean),
                });

                // Test that native tool capabilities returns array
                expect(service.getNativeToolCapabilities()).toBeInstanceOf(Array);
            }
        });

        it("should handle environment variables consistently", () => {
            const envVars = {
                OLLAMA_BASE_URL: "http://custom:11434",
                CLOUDFLARE_GATEWAY_TOKEN: "env-token",
                CLOUDFLARE_ACCOUNT_ID: "env-account",
                OPENROUTER_API_KEY: "env-key",
            };

            mockEnvVars(envVars);

            const envLocalOllama = new LocalOllamaService();
            const envCloudflareGateway = new CloudflareGatewayService();
            const envOpenRouter = new OpenRouterService();

            expect(envLocalOllama.defaultModel).toBe("llama3.1:8b");
            expect(envCloudflareGateway.defaultModel).toBe("@cf/openai/gpt-4o-mini");
            expect(envOpenRouter.defaultModel).toBe("openai/gpt-4o-mini");

            cleanupEnvVars(Object.keys(envVars));
        });
    });

    describe("Error Handling Tests", () => {
        let service: CloudflareGatewayService;

        beforeEach(() => {
            service = new CloudflareGatewayService({
                apiToken: "test-token",
                accountId: "test-account",
            });
        });

        it("should handle HTTP errors gracefully", async () => {
            mockFetch.mockResolvedValueOnce(createMockErrorResponse(401, "Unauthorized"));

            const options = {
                model: "@cf/openai/gpt-4o",
                input: createSampleMessages(1),
                tools: [],
            };

            await expect(async () => {
                for await (const _ of service.generateResponseStreaming(options)) {
                    // Should not reach here
                }
            }).rejects.toThrow("HTTP 401: Unauthorized");
        });

        it("should handle network errors gracefully", async () => {
            mockFetch.mockRejectedValueOnce(new Error("Network error"));

            const options = {
                model: "@cf/openai/gpt-4o",
                input: createSampleMessages(1),
                tools: [],
            };

            await expect(async () => {
                for await (const _ of service.generateResponseStreaming(options)) {
                    // Should not reach here
                }
            }).rejects.toThrow("Network error");
        });

        it("should handle malformed responses gracefully", async () => {
            const malformedChunks = [
                "data: {invalid json}\n",
                "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n",
                "data: {\"choices\":[{\"finish_reason\":\"stop\"}]}\n",
            ];

            mockFetch.mockResolvedValueOnce(createMockStreamingResponse(malformedChunks));

            const options = {
                model: "@cf/openai/gpt-4o",
                input: createSampleMessages(1),
                tools: [],
            };

            const events = [];
            for await (const event of service.generateResponseStreaming(options)) {
                events.push(event);
            }

            // Should skip malformed JSON and continue
            expect(events.length).toBeGreaterThan(0);
            expect(events[events.length - 1].type).toBe("done");
        });
    });
});
