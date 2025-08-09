import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { ModelStrategy } from "@vrooli/shared";
import { NetworkMonitor } from "./NetworkMonitor.js";
import { AIServiceRegistry } from "./registry.js";
import { ModelSelectionStrategyFactory } from "./ModelSelectionStrategy.js";
import type { ResponseContext } from "@vrooli/shared";

// Mock external dependencies
vi.mock("node-fetch", () => ({
    default: vi.fn(),
}));

vi.mock("dns", () => ({
    default: {
        resolve4: vi.fn(),
    },
}));

vi.mock("../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        error: vi.fn(),
        debug: vi.fn(),
    },
}));

vi.mock("@vrooli/shared", async () => {
    const actual = await vi.importActual("@vrooli/shared");
    return {
        ...actual,
        ModelStrategy: {
            FIXED: "fixed",
            FALLBACK: "fallback",
            COST_OPTIMIZED: "cost",
            QUALITY_FIRST: "quality",
            LOCAL_FIRST: "local",
        },
        aiServicesInfo: {
            services: {
                LocalOllama: {
                    models: {
                        "llama3.1:8b": {
                            enabled: true,
                            inputCost: 0.001,
                            outputCost: 0.001,
                            contextWindow: 128000,
                            maxOutputTokens: 4096,
                            features: {},
                            supportsReasoning: false,
                        },
                    },
                },
                CloudflareGateway: {
                    models: {
                        "gpt-4o": {
                            enabled: true,
                            inputCost: 250,
                            outputCost: 1000,
                            contextWindow: 128000,
                            maxOutputTokens: 4096,
                            features: { vision: true },
                            supportsReasoning: false,
                        },
                    },
                },
                OpenRouter: {
                    models: {
                        "openai/gpt-4o": {
                            enabled: true,
                            inputCost: 250,
                            outputCost: 1000,
                            contextWindow: 128000,
                            maxOutputTokens: 4096,
                            features: { vision: true },
                            supportsReasoning: false,
                        },
                    },
                },
            },
            fallbacks: {
                "llama3.1:8b": ["gpt-4o", "openai/gpt-4o"],
                "gpt-4o": ["openai/gpt-4o"],
            },
        },
    };
});

describe("Model Selection - Integration Tests", () => {
    let networkMonitor: NetworkMonitor;
    let registry: AIServiceRegistry;

    beforeEach(() => {
        // Reset singletons
        (NetworkMonitor as any).instance = undefined;
        (AIServiceRegistry as any).instance = undefined;

        networkMonitor = NetworkMonitor.getInstance();
        registry = AIServiceRegistry.get();

        // Mock registry methods
        vi.spyOn(registry, "getService").mockReturnValue({
            isHealthy: () => true,
        } as any);
    });

    afterEach(() => {
        networkMonitor.stop();
        vi.clearAllMocks();
    });

    describe("End-to-End Model Selection Scenarios", () => {
        it("should select local model when offline", async () => {
            // Mock offline state
            vi.spyOn(networkMonitor, "getState").mockResolvedValue({
                isOnline: false,
                connectivity: "offline",
                lastChecked: new Date(),
                cloudServicesReachable: false,
                localServicesReachable: true,
            });

            const strategy = ModelSelectionStrategyFactory.getStrategy(ModelStrategy.FALLBACK);
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.FALLBACK,
                    preferredModel: "gpt-4o",
                    offlineOnly: false,
                },
                networkState: await networkMonitor.getState(),
                registry,
            };

            const result = await strategy.selectModel(context);
            expect(result).toBe("llama3.1:8b"); // Should fallback to local
        });

        it("should respect offline-only mode", async () => {
            // Mock online state
            vi.spyOn(networkMonitor, "getState").mockResolvedValue({
                isOnline: true,
                connectivity: "online",
                lastChecked: new Date(),
                cloudServicesReachable: true,
                localServicesReachable: true,
            });

            const strategy = ModelSelectionStrategyFactory.getStrategy(ModelStrategy.LOCAL_FIRST);
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.LOCAL_FIRST,
                    preferredModel: "gpt-4o",
                    offlineOnly: true,
                },
                networkState: await networkMonitor.getState(),
                registry,
            };

            const result = await strategy.selectModel(context);
            expect(result).toBe("llama3.1:8b"); // Should use local despite preference
        });

        it("should handle degraded connectivity gracefully", async () => {
            // Mock degraded state (only local available)
            vi.spyOn(networkMonitor, "getState").mockResolvedValue({
                isOnline: true,
                connectivity: "degraded",
                lastChecked: new Date(),
                cloudServicesReachable: false,
                localServicesReachable: true,
            });

            const strategy = ModelSelectionStrategyFactory.getStrategy(ModelStrategy.QUALITY_FIRST);
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.QUALITY_FIRST,
                    preferredModel: "gpt-4o",
                    offlineOnly: false,
                },
                networkState: await networkMonitor.getState(),
                registry,
            };

            const result = await strategy.selectModel(context);
            expect(result).toBe("llama3.1:8b"); // Should use available local model
        });

        it("should select cost-optimized model when budget constrained", async () => {
            vi.spyOn(networkMonitor, "getState").mockResolvedValue({
                isOnline: true,
                connectivity: "online",
                lastChecked: new Date(),
                cloudServicesReachable: true,
                localServicesReachable: true,
            });

            const strategy = ModelSelectionStrategyFactory.getStrategy(ModelStrategy.COST_OPTIMIZED);
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.COST_OPTIMIZED,
                    offlineOnly: false,
                },
                networkState: await networkMonitor.getState(),
                registry,
                userCredits: 100, // Limited credits
            };

            const result = await strategy.selectModel(context);
            expect(result).toBe("llama3.1:8b"); // Should select cheapest model
        });

        it("should throw error when no affordable models available", async () => {
            vi.spyOn(networkMonitor, "getState").mockResolvedValue({
                isOnline: true,
                connectivity: "online",
                lastChecked: new Date(),
                cloudServicesReachable: true,
                localServicesReachable: true,
            });

            const strategy = ModelSelectionStrategyFactory.getStrategy(ModelStrategy.COST_OPTIMIZED);
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.COST_OPTIMIZED,
                    offlineOnly: false,
                },
                networkState: await networkMonitor.getState(),
                registry,
                userCredits: 0.0001, // Very limited credits
            };

            await expect(strategy.selectModel(context)).rejects.toThrow("No affordable models available");
        });
    });

    describe("Network State Integration", () => {
        it("should adapt to network state changes", async () => {
            // Start with online state
            const mockGetState = vi.spyOn(networkMonitor, "getState");
            mockGetState.mockResolvedValueOnce({
                isOnline: true,
                connectivity: "online",
                lastChecked: new Date(),
                cloudServicesReachable: true,
                localServicesReachable: true,
            });

            const strategy = ModelSelectionStrategyFactory.getStrategy(ModelStrategy.FALLBACK);
            let context = {
                modelConfig: {
                    strategy: ModelStrategy.FALLBACK,
                    preferredModel: "gpt-4o",
                    offlineOnly: false,
                },
                networkState: await networkMonitor.getState(),
                registry,
            };

            let result = await strategy.selectModel(context);
            expect(result).toBe("gpt-4o"); // Should use preferred cloud model

            // Switch to offline state
            mockGetState.mockResolvedValueOnce({
                isOnline: false,
                connectivity: "offline",
                lastChecked: new Date(),
                cloudServicesReachable: false,
                localServicesReachable: true,
            });

            context = {
                ...context,
                networkState: await networkMonitor.getState(),
            };

            result = await strategy.selectModel(context);
            expect(result).toBe("llama3.1:8b"); // Should fallback to local
        });
    });

    describe("Service Registry Integration", () => {
        it("should respect service availability from registry", async () => {
            vi.spyOn(networkMonitor, "getState").mockResolvedValue({
                isOnline: true,
                connectivity: "online",
                lastChecked: new Date(),
                cloudServicesReachable: true,
                localServicesReachable: true,
            });

            // Mock registry to return unavailable service
            vi.spyOn(registry, "getService").mockImplementation((serviceId) => {
                if (serviceId === "LocalOllama") {
                    throw new Error("Service not available");
                }
                return { isHealthy: () => true } as any;
            });

            const strategy = ModelSelectionStrategyFactory.getStrategy(ModelStrategy.LOCAL_FIRST);
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.LOCAL_FIRST,
                    preferredModel: "llama3.1:8b",
                    offlineOnly: false,
                },
                networkState: await networkMonitor.getState(),
                registry,
            };

            const result = await strategy.selectModel(context);
            expect(["gpt-4o", "openai/gpt-4o"]).toContain(result); // Should fallback to cloud
        });
    });

    describe("Comprehensive Fallback Scenarios", () => {
        it("should handle complex fallback chain", async () => {
            vi.spyOn(networkMonitor, "getState").mockResolvedValue({
                isOnline: true,
                connectivity: "degraded",
                lastChecked: new Date(),
                cloudServicesReachable: false,
                localServicesReachable: true,
            });

            const strategy = ModelSelectionStrategyFactory.getStrategy(ModelStrategy.FALLBACK);
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.FALLBACK,
                    preferredModel: "gpt-4o", // Preferred but unavailable
                    offlineOnly: false,
                },
                networkState: await networkMonitor.getState(),
                registry,
            };

            const result = await strategy.selectModel(context);
            expect(result).toBe("llama3.1:8b"); // Should use available local model
        });

        it("should handle all services unavailable", async () => {
            vi.spyOn(networkMonitor, "getState").mockResolvedValue({
                isOnline: false,
                connectivity: "offline",
                lastChecked: new Date(),
                cloudServicesReachable: false,
                localServicesReachable: false,
            });

            const strategy = ModelSelectionStrategyFactory.getStrategy(ModelStrategy.FALLBACK);
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.FALLBACK,
                    preferredModel: "gpt-4o",
                    offlineOnly: false,
                },
                networkState: await networkMonitor.getState(),
                registry,
            };

            await expect(strategy.selectModel(context)).rejects.toThrow();
        });
    });
});
