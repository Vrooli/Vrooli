/**
 * Production Ollama Resource Implementation
 * 
 * Provides infrastructure-level management for Ollama LLM service including:
 * - Service discovery and health monitoring
 * - Model management and availability checking
 * - Circuit breaker protection and resilience
 * - Integration with unified configuration system
 */

import { ResourceProvider } from "../ResourceProvider.js";
import { RegisterResource } from "../ResourceRegistry.js";
import { ResourceCategory, DeploymentType, ResourceHealth, DiscoveryStatus, ResourceEvent } from "../types.js";
import type { HealthCheckResult, IAIResource, ResourceEventData } from "../types.js";
import type { OllamaConfig, AIMetadata } from "../typeRegistry.js";
import { logger } from "../../../events/logger.js";

/**
 * Model information from Ollama API
 */
interface OllamaModel {
    name: string;
    size: number;
    digest: string;
    details?: {
        format: string;
        family: string;
        families?: string[];
        parameter_size: string;
        quantization_level: string;
    };
    modified_at: string;
}

/**
 * Ollama API response for /api/tags
 */
interface OllamaTagsResponse {
    models: OllamaModel[];
}

/**
 * Ollama version information
 */
interface OllamaVersionResponse {
    version: string;
}

/**
 * Production Ollama resource implementation with enhanced monitoring and resilience
 */
@RegisterResource
export class OllamaResource extends ResourceProvider<"ollama", OllamaConfig> implements IAIResource {
    // Resource identification
    readonly id = "ollama";
    readonly category = ResourceCategory.AI;
    readonly displayName = "Ollama";
    readonly description = "Local LLM inference engine with model management";
    readonly isSupported = true;
    readonly deploymentType = DeploymentType.Local;
    
    // Enhanced model caching with metadata
    private models: Map<string, OllamaModel> = new Map();
    private lastModelRefresh = 0;
    private ollamaVersion?: string;
    private readonly MODEL_CACHE_TTL_MS = 5 * 60 * 1000; // 5 minutes
    
    /**
     * Enhanced discovery with detailed model information
     */
    protected async performDiscovery(): Promise<boolean> {
        try {
            logger.debug("[OllamaResource] Starting service discovery");
            
            // First check if Ollama is responding
            const tagsResult = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/api/tags`,
                method: "GET",
                timeout: 5000,
            });
            
            if (!tagsResult.success) {
                logger.debug(`[OllamaResource] Tags endpoint failed: ${tagsResult.error}`);
                return false;
            }

            // Try to get version information for better discovery data
            try {
                const versionResult = await this.httpClient!.makeRequest({
                    url: `${this.config!.baseUrl}/api/version`,
                    method: "GET",
                    timeout: 3000,
                });
                
                if (versionResult.success) {
                    this.ollamaVersion = versionResult.data.version;
                }
            } catch (error) {
                logger.debug("[OllamaResource] Version endpoint not available (older Ollama version)");
            }

            // Cache models and emit discovery event
            await this.refreshModelCache(tagsResult.data);
            
            logger.info(`[OllamaResource] Discovered Ollama service with ${this.models.size} models`, {
                version: this.ollamaVersion,
                baseUrl: this.config!.baseUrl,
                models: Array.from(this.models.keys()),
            });

            // Emit discovery event
            this.emitResourceEvent(ResourceEvent.Discovered, {
                version: this.ollamaVersion,
                modelCount: this.models.size,
                models: Array.from(this.models.keys()),
            });
            
            return true;
        } catch (error) {
            logger.debug(`[OllamaResource] Discovery failed: ${error}`);
            return false;
        }
    }
    
    /**
     * Enhanced health check with detailed metrics
     */
    protected async performHealthCheck(): Promise<HealthCheckResult> {
        try {
            const startTime = Date.now();
            
            // Check basic connectivity and get models
            const result = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/api/tags`,
                method: "GET",
                timeout: this.config!.healthCheck?.timeoutMs || 5000,
            });
            
            const responseTime = Date.now() - startTime;
            
            if (result.success) {
                // Refresh model cache if stale
                if (Date.now() - this.lastModelRefresh > this.MODEL_CACHE_TTL_MS) {
                    await this.refreshModelCache(result.data);
                }
                
                const modelCount = this.models.size;
                const details = {
                    modelCount,
                    availableModels: Array.from(this.models.keys()),
                    version: this.ollamaVersion,
                    responseTime,
                    baseUrl: this.config!.baseUrl,
                    lastModelRefresh: new Date(this.lastModelRefresh),
                };

                logger.debug("[OllamaResource] Health check passed", details);

                return {
                    healthy: true,
                    message: `Ollama is healthy with ${modelCount} models (${responseTime}ms response)`,
                    details,
                    timestamp: new Date(),
                };
            }
            
            const errorDetails = {
                status: result.status,
                error: result.error,
                responseTime,
                baseUrl: this.config!.baseUrl,
            };

            logger.warn("[OllamaResource] Health check failed", errorDetails);
            
            return {
                healthy: false,
                message: `Ollama health check failed: ${result.error}`,
                details: errorDetails,
                timestamp: new Date(),
            };
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : "Unknown health check error";
            
            logger.error(`[OllamaResource] Health check error: ${errorMessage}`, error);
            
            return {
                healthy: false,
                message: errorMessage,
                details: { error: errorMessage },
                timestamp: new Date(),
            };
        }
    }
    
    /**
     * Refresh model cache with enhanced model information
     */
    private async refreshModelCache(tagsData: OllamaTagsResponse): Promise<void> {
        try {
            this.models.clear();
            
            for (const model of tagsData.models || []) {
                this.models.set(model.name, model);
            }
            
            this.lastModelRefresh = Date.now();
            
            logger.debug(`[OllamaResource] Refreshed model cache with ${this.models.size} models`);
        } catch (error) {
            logger.error("[OllamaResource] Failed to refresh model cache", error);
        }
    }
    
    /**
     * List available models - required by IAIResource
     */
    async listModels(): Promise<string[]> {
        if (this._status !== DiscoveryStatus.Available) {
            logger.debug("[OllamaResource] Service not available, returning empty model list");
            return [];
        }
        
        // Refresh cache if stale
        if (Date.now() - this.lastModelRefresh > this.MODEL_CACHE_TTL_MS) {
            try {
                const result = await this.httpClient!.makeRequest({
                    url: `${this.config!.baseUrl}/api/tags`,
                    method: "GET",
                    timeout: 10000,
                });
                
                if (result.success) {
                    await this.refreshModelCache(result.data);
                }
            } catch (error) {
                logger.warn("[OllamaResource] Failed to refresh models for listModels call", error);
            }
        }
        
        return Array.from(this.models.keys());
    }
    
    /**
     * Check if a specific model is available - required by IAIResource
     */
    async hasModel(modelId: string): Promise<boolean> {
        const models = await this.listModels();
        const hasModel = models.includes(modelId);
        
        logger.debug(`[OllamaResource] Model ${modelId} availability: ${hasModel}`);
        return hasModel;
    }
    
    /**
     * Get detailed model information
     */
    async getModelInfo(modelId: string): Promise<OllamaModel | null> {
        await this.listModels(); // Ensure cache is fresh
        return this.models.get(modelId) || null;
    }
    
    /**
     * Pull a model from Ollama with progress tracking
     */
    async pullModel(modelName: string, timeout = 600000): Promise<void> {
        if (this._status !== DiscoveryStatus.Available) {
            throw new Error("Ollama is not available");
        }
        
        logger.info(`[OllamaResource] Starting model pull: ${modelName}`);
        
        try {
            const result = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/api/pull`,
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: { name: modelName },
                timeout,
            });
            
            if (!result.success) {
                throw new Error(`Failed to pull model ${modelName}: ${result.error}`);
            }
            
            // Refresh model list after successful pull
            await this.listModels();
            
            logger.info(`[OllamaResource] Successfully pulled model: ${modelName}`);
            
            // Emit event for model addition
            this.emitResourceEvent(ResourceEvent.ConfigUpdated, {
                action: "model_pulled",
                modelName,
                totalModels: this.models.size,
            });
            
        } catch (error) {
            logger.error(`[OllamaResource] Failed to pull model ${modelName}`, error);
            throw error;
        }
    }
    
    /**
     * Delete a model from Ollama
     */
    async deleteModel(modelName: string): Promise<void> {
        if (this._status !== DiscoveryStatus.Available) {
            throw new Error("Ollama is not available");
        }
        
        logger.info(`[OllamaResource] Deleting model: ${modelName}`);
        
        try {
            const result = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/api/delete`,
                method: "DELETE",
                headers: { "Content-Type": "application/json" },
                body: { name: modelName },
                timeout: 30000,
            });
            
            if (!result.success) {
                throw new Error(`Failed to delete model ${modelName}: ${result.error}`);
            }
            
            // Refresh model list after successful deletion
            await this.listModels();
            
            logger.info(`[OllamaResource] Successfully deleted model: ${modelName}`);
            
            // Emit event for model removal
            this.emitResourceEvent(ResourceEvent.ConfigUpdated, {
                action: "model_deleted",
                modelName,
                totalModels: this.models.size,
            });
            
        } catch (error) {
            logger.error(`[OllamaResource] Failed to delete model ${modelName}`, error);
            throw error;
        }
    }
    
    /**
     * Test model generation capability
     */
    async testModelGeneration(modelName: string, prompt = "Hello"): Promise<boolean> {
        if (!await this.hasModel(modelName)) {
            return false;
        }
        
        try {
            const result = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/api/generate`,
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: {
                    model: modelName,
                    prompt,
                    stream: false,
                    options: { num_predict: 1 }, // Minimal generation for testing
                },
                timeout: 30000,
            });
            
            return result.success;
        } catch (error) {
            logger.debug(`[OllamaResource] Model generation test failed for ${modelName}`, error);
            return false;
        }
    }
    
    /**
     * Provide enhanced metadata about the resource
     */
    protected getMetadata(): AIMetadata {
        const modelList = Array.from(this.models.keys());
        
        return {
            version: this.ollamaVersion || "unknown",
            capabilities: [
                "text-generation",
                "chat-completion",
                "embeddings",
                "model-management",
                "streaming",
            ],
            supportedModels: modelList,
            lastUpdated: new Date(),
            discoveredAt: this._lastHealthCheck,
            maxTokens: 8192, // Common context window, varies by model
            contextWindow: 8192,
            costPerToken: 0, // Local models have no cost
            // Additional Ollama-specific metadata
            totalModels: this.models.size,
            baseUrl: this.config?.baseUrl,
            modelSizes: Array.from(this.models.values()).map(m => ({
                name: m.name,
                sizeBytes: m.size,
                family: m.details?.family,
                parameterSize: m.details?.parameter_size,
            })),
        };
    }
    
    /**
     * Emit resource events with proper typing
     */
    private emitResourceEvent(event: ResourceEvent, details?: Record<string, any>): void {
        const eventData: ResourceEventData = {
            resourceId: this.id,
            category: this.category,
            event,
            timestamp: new Date(),
            details,
        };
        
        this.emit(event, eventData);
    }
    
    /**
     * Enhanced shutdown with cleanup
     */
    async shutdown(): Promise<void> {
        logger.info("[OllamaResource] Shutting down Ollama resource");
        
        // Clear caches
        this.models.clear();
        this.lastModelRefresh = 0;
        this.ollamaVersion = undefined;
        
        // Call parent shutdown
        await super.shutdown();
        
        logger.info("[OllamaResource] Ollama resource shutdown complete");
    }
}
