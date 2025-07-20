import { describe, it, expect, beforeEach, vi } from "vitest";
import { ModelStrategy } from "@vrooli/shared";
import {
    ModelSelectionStrategyFactory,
    FixedModelStrategy,
    FallbackModelStrategy,
    CostOptimizedModelStrategy,
    QualityFirstModelStrategy,
    LocalFirstModelStrategy,
} from "./ModelSelectionStrategy.js";
import type { NetworkState } from "./NetworkMonitor.js";
import type { AIServiceRegistry } from "./registry.js";

// Mock the shared module
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

describe("ModelSelectionStrategy", () => {
    let mockNetworkState: NetworkState;
    let mockRegistry: AIServiceRegistry;

    beforeEach(() => {
        mockNetworkState = {
            isOnline: true,
            connectivity: "online",
            lastChecked: new Date(),
            cloudServicesReachable: true,
            localServicesReachable: true,
        };

        mockRegistry = {
            getService: vi.fn().mockReturnValue({
                isHealthy: () => true,
            }),
        } as any;
    });

    describe("ModelSelectionStrategyFactory", () => {
        it("should return correct strategy instances", () => {
            expect(ModelSelectionStrategyFactory.getStrategy(ModelStrategy.FIXED)).toBeInstanceOf(FixedModelStrategy);
            expect(ModelSelectionStrategyFactory.getStrategy(ModelStrategy.FALLBACK)).toBeInstanceOf(FallbackModelStrategy);
            expect(ModelSelectionStrategyFactory.getStrategy(ModelStrategy.COST_OPTIMIZED)).toBeInstanceOf(CostOptimizedModelStrategy);
            expect(ModelSelectionStrategyFactory.getStrategy(ModelStrategy.QUALITY_FIRST)).toBeInstanceOf(QualityFirstModelStrategy);
            expect(ModelSelectionStrategyFactory.getStrategy(ModelStrategy.LOCAL_FIRST)).toBeInstanceOf(LocalFirstModelStrategy);
        });

        it("should throw error for unknown strategy", () => {
            expect(() => ModelSelectionStrategyFactory.getStrategy("unknown" as ModelStrategy)).toThrow("Unknown model strategy: unknown");
        });
    });

    describe("FixedModelStrategy", () => {
        let strategy: FixedModelStrategy;

        beforeEach(() => {
            strategy = new FixedModelStrategy();
        });

        it("should return preferred model when available", async () => {
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.FIXED,
                    preferredModel: "llama3.1:8b",
                    offlineOnly: false,
                },
                networkState: mockNetworkState,
                registry: mockRegistry,
            };

            const result = await strategy.selectModel(context);
            expect(result).toBe("llama3.1:8b");
        });

        it("should throw error when no preferred model specified", async () => {
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.FIXED,
                    offlineOnly: false,
                },
                networkState: mockNetworkState,
                registry: mockRegistry,
            };

            await expect(strategy.selectModel(context)).rejects.toThrow("No preferred model specified for FIXED strategy");
        });

        it("should throw error when offline and model is not local", async () => {
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.FIXED,
                    preferredModel: "gpt-4o",
                    offlineOnly: true,
                },
                networkState: mockNetworkState,
                registry: mockRegistry,
            };

            await expect(strategy.selectModel(context)).rejects.toThrow("Model gpt-4o is not available offline");
        });
    });

    describe("FallbackModelStrategy", () => {
        let strategy: FallbackModelStrategy;

        beforeEach(() => {
            strategy = new FallbackModelStrategy();
        });

        it("should return preferred model when available", async () => {
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.FALLBACK,
                    preferredModel: "llama3.1:8b",
                    offlineOnly: false,
                },
                networkState: mockNetworkState,
                registry: mockRegistry,
            };

            const result = await strategy.selectModel(context);
            expect(result).toBe("llama3.1:8b");
        });

        it("should fallback to default when no preferred model", async () => {
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.FALLBACK,
                    offlineOnly: false,
                },
                networkState: mockNetworkState,
                registry: mockRegistry,
            };

            const result = await strategy.selectModel(context);
            expect(result).toBe("llama3.1:8b"); // LocalOllama default
        });

        it("should use cloud default when local services unavailable", async () => {
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.FALLBACK,
                    offlineOnly: false,
                },
                networkState: {
                    ...mockNetworkState,
                    localServicesReachable: false,
                },
                registry: mockRegistry,
            };

            const result = await strategy.selectModel(context);
            expect(result).toBe("gpt-4o"); // CloudflareGateway default
        });
    });

    describe("CostOptimizedModelStrategy", () => {
        let strategy: CostOptimizedModelStrategy;

        beforeEach(() => {
            strategy = new CostOptimizedModelStrategy();
        });

        it("should return cheapest available model", async () => {
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.COST_OPTIMIZED,
                    offlineOnly: false,
                },
                networkState: mockNetworkState,
                registry: mockRegistry,
            };

            const result = await strategy.selectModel(context);
            expect(result).toBe("llama3.1:8b"); // Cheapest model
        });

        it("should throw error when no affordable models", async () => {
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.COST_OPTIMIZED,
                    offlineOnly: false,
                },
                networkState: mockNetworkState,
                registry: mockRegistry,
                userCredits: 0.0001, // Very low credits
            };

            await expect(strategy.selectModel(context)).rejects.toThrow("No affordable models available");
        });
    });

    describe("QualityFirstModelStrategy", () => {
        let strategy: QualityFirstModelStrategy;

        beforeEach(() => {
            strategy = new QualityFirstModelStrategy();
        });

        it("should return highest quality available model", async () => {
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.QUALITY_FIRST,
                    offlineOnly: false,
                },
                networkState: mockNetworkState,
                registry: mockRegistry,
            };

            const result = await strategy.selectModel(context);
            // Should return one of the GPT-4o models as they have higher quality scores
            expect(["gpt-4o", "openai/gpt-4o"]).toContain(result);
        });

        it("should throw error when no affordable models", async () => {
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.QUALITY_FIRST,
                    offlineOnly: false,
                },
                networkState: mockNetworkState,
                registry: mockRegistry,
                userCredits: 0.0001, // Very low credits
            };

            await expect(strategy.selectModel(context)).rejects.toThrow("No affordable models available");
        });
    });

    describe("LocalFirstModelStrategy", () => {
        let strategy: LocalFirstModelStrategy;

        beforeEach(() => {
            strategy = new LocalFirstModelStrategy();
        });

        it("should return preferred local model when available", async () => {
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.LOCAL_FIRST,
                    preferredModel: "llama3.1:8b",
                    offlineOnly: false,
                },
                networkState: mockNetworkState,
                registry: mockRegistry,
            };

            const result = await strategy.selectModel(context);
            expect(result).toBe("llama3.1:8b");
        });

        it("should fallback to cloud when no local models available", async () => {
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.LOCAL_FIRST,
                    offlineOnly: false,
                },
                networkState: {
                    ...mockNetworkState,
                    localServicesReachable: false,
                },
                registry: mockRegistry,
            };

            const result = await strategy.selectModel(context);
            expect(["gpt-4o", "openai/gpt-4o"]).toContain(result);
        });

        it("should not fallback to cloud when offline-only mode", async () => {
            const context = {
                modelConfig: {
                    strategy: ModelStrategy.LOCAL_FIRST,
                    offlineOnly: true,
                },
                networkState: {
                    ...mockNetworkState,
                    localServicesReachable: false,
                },
                registry: mockRegistry,
            };

            await expect(strategy.selectModel(context)).rejects.toThrow("No available models found");
        });
    });

    describe("Base ModelSelectionStrategy", () => {
        let strategy: FixedModelStrategy;

        beforeEach(() => {
            strategy = new FixedModelStrategy();
        });

        it("should calculate model score correctly", () => {
            const modelInfo = {
                contextWindow: 128000,
                maxOutputTokens: 4096,
                features: { vision: true, functionCalling: true },
                supportsReasoning: true,
            };

            const score = strategy["calculateModelScore"](modelInfo);
            expect(score).toBe(85); // 50 + 12.8 + 2.5 + 10 + 10
        });

        it("should check affordability correctly", () => {
            const modelScore = {
                model: "test",
                score: 75,
                cost: 100, // 100 cents per 1M tokens
                isLocal: false,
                available: true,
            };

            expect(strategy["canAffordModel"](modelScore, 1000)).toBe(true); // 1000 > 0.1
            expect(strategy["canAffordModel"](modelScore, 0.05)).toBe(false); // 0.05 < 0.1
            expect(strategy["canAffordModel"](modelScore, undefined)).toBe(true); // No limit
        });
    });
});
