import { AnthropicModel, MistralModel, OpenAIModel } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { NetworkMonitor } from "./NetworkMonitor.js";
import { AIServiceErrorType, AIServiceRegistry, AIServiceState, LlmServiceId } from "./registry.js";

// Mock NetworkMonitor
vi.mock("./NetworkMonitor.js", () => ({
    NetworkMonitor: {
        getInstance: vi.fn(() => ({
            start: vi.fn(),
            getState: vi.fn(),
        })),
    },
}));

// Mock logger
vi.mock("../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        warn: vi.fn(),
        warning: vi.fn(),
        error: vi.fn(),
        debug: vi.fn(),
    },
}));

describe("AIServiceRegistry", () => {
    let registry: AIServiceRegistry;

    beforeEach(async () => {
        vi.useFakeTimers();
        AIServiceRegistry["instance"] = undefined;
        await AIServiceRegistry.init();
        registry = AIServiceRegistry.get();
    });

    afterEach(() => {
        vi.runAllTimers();
        vi.restoreAllMocks();
        vi.useRealTimers();
    });

    it("returns the preferred service when active", () => {
        expect(registry.getBestService(AnthropicModel.Opus3)).toBe(LlmServiceId.Anthropic);
        expect(registry.getBestService(OpenAIModel.Gpt4o_Mini)).toBe(LlmServiceId.OpenAI);
        expect(registry.getBestService(MistralModel.Codestral)).toBe(LlmServiceId.Mistral);
    });

    it("returns the first active fallback service when preferred is on cooldown", () => {
        registry.updateServiceState("Anthropic", AIServiceErrorType.Overloaded);
        expect(registry.getBestService(AnthropicModel.Opus3)).not.toBe(LlmServiceId.Anthropic);
    });

    it("returns the next active fallback service when preferred and first fallback are on cooldown", () => {
        registry.updateServiceState("Anthropic", AIServiceErrorType.Overloaded); // Set Anthropic to cooldown
        registry.updateServiceState("Mistral", AIServiceErrorType.Overloaded); // Set Mistral to cooldown

        const bestServiceId = registry.getBestService(AnthropicModel.Sonnet3_5);
        expect(bestServiceId).not.toBe(LlmServiceId.Anthropic);
        expect(bestServiceId).not.toBe(LlmServiceId.Mistral);
    });

    it("returns null when all services are on cooldown or disabled", () => {
        Object.values(LlmServiceId).forEach(serviceId => {
            registry.updateServiceState(serviceId, AIServiceErrorType.Overloaded);
        });

        expect(registry.getBestService(AnthropicModel.Opus3)).toBeNull();
    });

    it("registers and retrieves service states correctly", () => {
        registry.registerService("testService");
        registry.registerService("testService2");
        expect(registry.getServiceState("testService")).toBe(AIServiceState.Active);
        expect(registry.getServiceState("testService2")).toBe(AIServiceState.Active);
    });

    it("automatically registers and activates an unregistered service when its state is requested", () => {
        const serviceState = registry.getServiceState("autoRegisteredService");
        expect(serviceState).toBe(AIServiceState.Active);
    });

    it("handles critical error by disabling service", () => {
        registry.registerService("criticalErrorService");
        registry.registerService("fineService");
        registry.updateServiceState("criticalErrorService", AIServiceErrorType.Authentication);
        expect(registry.getServiceState("criticalErrorService")).toBe(AIServiceState.Disabled);
        expect(registry.getServiceState("fineService")).toBe(AIServiceState.Active);

        // Fast-forward to make sure it stays disabled
        vi.advanceTimersByTime(1_000_000);
        expect(registry.getServiceState("criticalErrorService")).toBe(AIServiceState.Disabled);
    });

    it("handles cooldown error by setting service into cooldown", () => {
        registry.registerService("cooldownService");
        registry.updateServiceState("cooldownService", AIServiceErrorType.Overloaded);
        expect(registry.getServiceState("cooldownService")).toBe(AIServiceState.Cooldown);
    });

    it("resets service from cooldown to active after cooldown period", async () => {
        registry.registerService("resetService");
        registry.updateServiceState("resetService", AIServiceErrorType.Overloaded);

        // Fast-forward a few seconds to make sure it's still in cooldown
        vi.advanceTimersByTime(10_000);
        expect(registry.getServiceState("resetService")).toBe(AIServiceState.Cooldown);

        // Fast-forward to the end of the cooldown period (15 minutes)
        vi.advanceTimersByTime(15 * 60 * 1000);
        expect(registry.getServiceState("resetService")).toBe(AIServiceState.Active);
    });

    describe("Network Awareness", () => {
        let mockNetworkMonitor: any;

        beforeEach(() => {
            mockNetworkMonitor = {
                start: vi.fn(),
                getState: vi.fn(),
            };
            vi.mocked(NetworkMonitor.getInstance).mockReturnValue(mockNetworkMonitor);
            // Reset singleton to ensure it uses the mocked NetworkMonitor
            AIServiceRegistry["instance"] = undefined;
        });

        afterEach(() => {
            vi.clearAllMocks();
        });

        describe("initialization", () => {
            it("should start network monitoring on init", async () => {
                await AIServiceRegistry.init();
                expect(mockNetworkMonitor.start).toHaveBeenCalled();
            });
        });

        describe("getBestService with network awareness", () => {
            beforeEach(() => {
                // Setup services as active
                registry.registerService(LlmServiceId.LocalOllama, AIServiceState.Active);
                registry.registerService(LlmServiceId.CloudflareGateway, AIServiceState.Active);
                registry.registerService(LlmServiceId.OpenRouter, AIServiceState.Active);
            });

            it("should prefer local services when offline", async () => {
                mockNetworkMonitor.getState.mockResolvedValue({
                    isOnline: false,
                    cloudServicesReachable: false,
                    localServicesReachable: true,
                });

                const result = await registry.getBestService("llama3.1:8b");
                expect(result).toBe(LlmServiceId.LocalOllama);
            });

            it("should return null when offline-only and no local services", async () => {
                mockNetworkMonitor.getState.mockResolvedValue({
                    isOnline: false,
                    cloudServicesReachable: false,
                    localServicesReachable: false,
                });

                const result = await registry.getBestService("llama3.1:8b", true);
                expect(result).toBeNull();
            });

            it("should use cloud services when online", async () => {
                mockNetworkMonitor.getState.mockResolvedValue({
                    isOnline: true,
                    cloudServicesReachable: true,
                    localServicesReachable: true,
                });

                const result = await registry.getBestService("@cf/openai/gpt-4o");
                expect(result).toBe(LlmServiceId.CloudflareGateway);
            });

            it("should fallback to available services when preferred is unreachable", async () => {
                mockNetworkMonitor.getState.mockResolvedValue({
                    isOnline: true,
                    cloudServicesReachable: false,
                    localServicesReachable: true,
                });

                const result = await registry.getBestService("@cf/openai/gpt-4o");
                expect(result).toBe(LlmServiceId.LocalOllama);
            });

            it("should respect offline-only mode", async () => {
                mockNetworkMonitor.getState.mockResolvedValue({
                    isOnline: true,
                    cloudServicesReachable: true,
                    localServicesReachable: true,
                });

                const result = await registry.getBestService("@cf/openai/gpt-4o", true);
                expect(result).toBe(LlmServiceId.LocalOllama);
            });
        });

        describe("isServiceAvailable", () => {
            beforeEach(() => {
                registry.registerService(LlmServiceId.LocalOllama, AIServiceState.Active);
                registry.registerService(LlmServiceId.CloudflareGateway, AIServiceState.Active);
            });

            it("should return false for inactive services", () => {
                registry.registerService(LlmServiceId.OpenRouter, AIServiceState.Disabled);

                const networkState = {
                    cloudServicesReachable: true,
                    localServicesReachable: true,
                };

                const result = registry["isServiceAvailable"](LlmServiceId.OpenRouter, networkState);
                expect(result).toBe(false);
            });

            it("should return false for local services when local not reachable", () => {
                const networkState = {
                    cloudServicesReachable: true,
                    localServicesReachable: false,
                };

                const result = registry["isServiceAvailable"](LlmServiceId.LocalOllama, networkState);
                expect(result).toBe(false);
            });

            it("should return false for cloud services when cloud not reachable", () => {
                const networkState = {
                    cloudServicesReachable: false,
                    localServicesReachable: true,
                };

                const result = registry["isServiceAvailable"](LlmServiceId.CloudflareGateway, networkState);
                expect(result).toBe(false);
            });

            it("should return true for local services when local reachable", () => {
                const networkState = {
                    cloudServicesReachable: false,
                    localServicesReachable: true,
                };

                const result = registry["isServiceAvailable"](LlmServiceId.LocalOllama, networkState);
                expect(result).toBe(true);
            });

            it("should return true for cloud services when cloud reachable", () => {
                const networkState = {
                    cloudServicesReachable: true,
                    localServicesReachable: false,
                };

                const result = registry["isServiceAvailable"](LlmServiceId.CloudflareGateway, networkState);
                expect(result).toBe(true);
            });
        });

        describe("model identification", () => {
            it("should identify Cloudflare models", () => {
                const result = registry.getServiceId("@cf/openai/gpt-4o");
                expect(result).toBe(LlmServiceId.CloudflareGateway);
            });

            it("should identify OpenRouter models", () => {
                const result = registry.getServiceId("openai/gpt-4o");
                expect(result).toBe(LlmServiceId.OpenRouter);
            });

            it("should identify local Ollama models", () => {
                const result = registry.getServiceId("llama3.1:8b");
                expect(result).toBe(LlmServiceId.LocalOllama);
            });

            it("should return default for unknown models", () => {
                const result = registry.getServiceId("unknown-model");
                expect(result).toBe(LlmServiceId.LocalOllama); // Default service
            });
        });

        describe("universal fallback chain", () => {
            it("should try universal fallback when specific fallbacks fail", async () => {
                mockNetworkMonitor.getState.mockResolvedValue({
                    isOnline: true,
                    cloudServicesReachable: true,
                    localServicesReachable: false,
                });

                // Set LocalOllama as disabled to force fallback
                registry.registerService(LlmServiceId.LocalOllama, AIServiceState.Disabled);
                registry.registerService(LlmServiceId.CloudflareGateway, AIServiceState.Active);
                registry.registerService(LlmServiceId.OpenRouter, AIServiceState.Active);

                const result = await registry.getBestService("unknown-model");
                expect(result).toBe(LlmServiceId.CloudflareGateway);
            });

            it("should return null when no services are available", async () => {
                mockNetworkMonitor.getState.mockResolvedValue({
                    isOnline: false,
                    cloudServicesReachable: false,
                    localServicesReachable: false,
                });

                registry.registerService(LlmServiceId.LocalOllama, AIServiceState.Disabled);
                registry.registerService(LlmServiceId.CloudflareGateway, AIServiceState.Disabled);
                registry.registerService(LlmServiceId.OpenRouter, AIServiceState.Disabled);

                const result = await registry.getBestService("any-model");
                expect(result).toBeNull();
            });
        });
    });
});
