import { type AIServiceName, type LlmServiceModel, type ModelConfig, ModelStrategy, aiServicesInfo } from "@vrooli/shared";
import { type NetworkState } from "./NetworkMonitor.js";
import { type AIServiceRegistry } from "./registry.js";

interface ModelSelectionContext {
    modelConfig: ModelConfig;
    networkState: NetworkState;
    registry: AIServiceRegistry;
    userCredits?: number;
}

interface ModelScore {
    model: string;
    score: number;
    cost: number;
    isLocal: boolean;
    available: boolean;
}

/**
 * Abstract base class for model selection strategies
 */
export abstract class ModelSelectionStrategy {
    abstract selectModel(context: ModelSelectionContext): Promise<string>;

    /**
     * Get all available models with their metadata
     */
    protected async getAvailableModels(
        context: ModelSelectionContext,
    ): Promise<ModelScore[]> {
        const models: ModelScore[] = [];
        const servicesInfo = aiServicesInfo;

        // Check each service
        for (const [serviceName, serviceInfo] of Object.entries(servicesInfo.services)) {
            const isLocal = serviceName === "LocalOllama";

            // Skip cloud services if offline or offline-only mode
            if (!isLocal && (!context.networkState.cloudServicesReachable || context.modelConfig.offlineOnly)) {
                continue;
            }

            // Skip local services if not reachable
            if (isLocal && !context.networkState.localServicesReachable) {
                continue;
            }

            // Check if service is available
            const service = context.registry.getService(serviceName as any);
            if (!service || !(await service.isHealthy())) {
                continue;
            }

            // Add models from this service
            for (const [modelId, modelInfo] of Object.entries(serviceInfo.models)) {
                if (!modelInfo.enabled) continue;

                models.push({
                    model: modelId,
                    score: this.calculateModelScore(modelInfo),
                    cost: (modelInfo.inputCost + modelInfo.outputCost) / 2, // Average cost
                    isLocal,
                    available: true,
                });
            }
        }

        return models;
    }

    /**
     * Calculate a quality score for a model (0-100)
     */
    protected calculateModelScore(modelInfo: any): number {
        let score = 50; // Base score

        // Context window size (normalized to 0-20 points)
        score += Math.min(20, (modelInfo.contextWindow / 200_000) * 20);

        // Output tokens (normalized to 0-15 points)
        score += Math.min(15, (modelInfo.maxOutputTokens / 65_000) * 15);

        // Features (5 points per feature, max 15)
        const featureCount = Object.keys(modelInfo.features || {}).length;
        score += Math.min(15, featureCount * 5);

        // Reasoning support (10 points)
        if (modelInfo.supportsReasoning) {
            score += 10;
        }

        return Math.min(100, score);
    }

    /**
     * Check if user has enough credits for a model
     */
    protected canAffordModel(modelScore: ModelScore, userCredits?: number): boolean {
        if (userCredits === undefined) return true; // No credit limit
        // Assume minimum of 1000 tokens for a basic interaction
        const estimatedCost = modelScore.cost * 1000 / 1_000_000; // Convert to cents
        return userCredits >= estimatedCost;
    }
}

/**
 * Fixed strategy - always use the preferred model or fail
 */
export class FixedModelStrategy extends ModelSelectionStrategy {
    async selectModel(context: ModelSelectionContext): Promise<string> {
        if (!context.modelConfig.preferredModel) {
            throw new Error("No preferred model specified for FIXED strategy");
        }

        // Check if we're offline and the model is not local
        if (context.modelConfig.offlineOnly || !context.networkState.isOnline) {
            const isLocalModel = this.isLocalModel(context.modelConfig.preferredModel);
            if (!isLocalModel) {
                throw new Error(`Model ${context.modelConfig.preferredModel} is not available offline`);
            }
        }

        // Verify the model is available
        const models = await this.getAvailableModels(context);
        const found = models.find(m => m.model === context.modelConfig.preferredModel);

        if (!found || !found.available) {
            throw new Error(`Model ${context.modelConfig.preferredModel} is not available`);
        }

        return context.modelConfig.preferredModel;
    }

    private isLocalModel(model: string): boolean {
        // Check if model is in LocalOllama service
        const localModels = Object.keys(aiServicesInfo.services.LocalOllama.models);
        return localModels.includes(model);
    }
}

/**
 * Fallback strategy - try preferred model, then use fallback chain
 */
export class FallbackModelStrategy extends ModelSelectionStrategy {
    async selectModel(context: ModelSelectionContext): Promise<string> {
        const preferredModel = context.modelConfig.preferredModel;

        // If we have a preferred model, try it first
        if (preferredModel) {
            const models = await this.getAvailableModels(context);
            const preferred = models.find(m => m.model === preferredModel);

            if (preferred?.available && this.canAffordModel(preferred, context.userCredits)) {
                return preferredModel;
            }

            // Try fallback chain for this specific model
            const fallbacks = aiServicesInfo.fallbacks[preferredModel as LlmServiceModel] || [];
            for (const fallbackModel of fallbacks) {
                const fallback = models.find(m => m.model === fallbackModel);
                if (fallback?.available && this.canAffordModel(fallback, context.userCredits)) {
                    return fallbackModel;
                }
            }
        }

        // No preferred model or all fallbacks failed, use default
        const defaultModel = this.getDefaultModel(context);
        if (defaultModel) return defaultModel;

        throw new Error("No available models found");
    }

    private getDefaultModel(context: ModelSelectionContext): string | null {
        // If offline or offline-only, use local default
        if (context.modelConfig.offlineOnly || !context.networkState.cloudServicesReachable) {
            return aiServicesInfo.services.LocalOllama.defaultModel;
        }

        // Use global default
        const defaultService = aiServicesInfo.defaultService;
        const service = aiServicesInfo.services[defaultService as unknown as AIServiceName];
        return service?.defaultModel || null;
    }
}

/**
 * Cost-optimized strategy - select cheapest available model
 */
export class CostOptimizedModelStrategy extends ModelSelectionStrategy {
    async selectModel(context: ModelSelectionContext): Promise<string> {
        const models = await this.getAvailableModels(context);

        // Filter by affordability
        const affordableModels = models.filter(m =>
            this.canAffordModel(m, context.userCredits),
        );

        if (affordableModels.length === 0) {
            throw new Error("No affordable models available");
        }

        // Sort by cost (ascending)
        affordableModels.sort((a, b) => a.cost - b.cost);

        // Return cheapest model
        return affordableModels[0].model;
    }
}

/**
 * Quality-first strategy - select best available model
 */
export class QualityFirstModelStrategy extends ModelSelectionStrategy {
    async selectModel(context: ModelSelectionContext): Promise<string> {
        const models = await this.getAvailableModels(context);

        // Filter by affordability
        const affordableModels = models.filter(m =>
            this.canAffordModel(m, context.userCredits),
        );

        if (affordableModels.length === 0) {
            throw new Error("No affordable models available");
        }

        // Sort by quality score (descending)
        affordableModels.sort((a, b) => b.score - a.score);

        // Return best model
        return affordableModels[0].model;
    }
}

/**
 * Local-first strategy - prefer local models, fallback to cloud
 */
export class LocalFirstModelStrategy extends ModelSelectionStrategy {
    async selectModel(context: ModelSelectionContext): Promise<string> {
        const models = await this.getAvailableModels(context);

        // Try local models first
        const localModels = models.filter(m => m.isLocal && this.canAffordModel(m, context.userCredits));

        if (localModels.length > 0) {
            // If we have a preferred model and it's local, use it
            if (context.modelConfig.preferredModel) {
                const preferred = localModels.find(m => m.model === context.modelConfig.preferredModel);
                if (preferred) return preferred.model;
            }

            // Otherwise, use the best local model
            localModels.sort((a, b) => b.score - a.score);
            return localModels[0].model;
        }

        // No local models available, try cloud if allowed
        if (!context.modelConfig.offlineOnly && context.networkState.cloudServicesReachable) {
            const cloudModels = models.filter(m => !m.isLocal && this.canAffordModel(m, context.userCredits));

            if (cloudModels.length > 0) {
                // Sort by quality
                cloudModels.sort((a, b) => b.score - a.score);
                return cloudModels[0].model;
            }
        }

        throw new Error("No available models found");
    }
}

/**
 * Factory for creating model selection strategies
 */
export class ModelSelectionStrategyFactory {
    private static strategies = new Map<ModelStrategy, ModelSelectionStrategy>([
        [ModelStrategy.FIXED, new FixedModelStrategy()],
        [ModelStrategy.FALLBACK, new FallbackModelStrategy()],
        [ModelStrategy.COST_OPTIMIZED, new CostOptimizedModelStrategy()],
        [ModelStrategy.QUALITY_FIRST, new QualityFirstModelStrategy()],
        [ModelStrategy.LOCAL_FIRST, new LocalFirstModelStrategy()],
    ]);

    static getStrategy(strategy: ModelStrategy): ModelSelectionStrategy {
        const impl = this.strategies.get(strategy);
        if (!impl) {
            throw new Error(`Unknown model strategy: ${strategy}`);
        }
        return impl;
    }
}
