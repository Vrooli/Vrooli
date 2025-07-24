/**
 * Example implementation of an Ollama resource provider.
 * Demonstrates how to implement a resource for AI model inference.
 */

import { ResourceProvider } from "../ResourceProvider.js";
import { RegisterResource } from "../ResourceRegistry.js";
import { ResourceCategory, DeploymentType, ResourceHealth, DiscoveryStatus } from "../types.js";
import type { HealthCheckResult, IAIResource, BaseResourceConfig } from "../types.js";
import type { AIMetadata } from "../typeRegistry.js";
import { logger } from "../../../events/logger.js";

/**
 * Configuration for Ollama resource
 */
interface OllamaConfig extends BaseResourceConfig {
    baseUrl: string;
    models?: string[];
    keepAlive?: string;
    numThread?: number;
    timeout?: number;
}

/**
 * Ollama resource implementation.
 * Provides access to locally running Ollama LLM inference engine.
 */
@RegisterResource
export class OllamaResource extends ResourceProvider<"ollama", OllamaConfig> implements IAIResource {
    // Resource identification
    readonly id = "ollama";
    readonly category = ResourceCategory.AI;
    readonly displayName = "Ollama";
    readonly description = "Local LLM inference engine";
    readonly isSupported = true;
    readonly deploymentType = DeploymentType.Local;
    
    // Cache for discovered models
    private models: string[] = [];
    
    /**
     * Discover if Ollama is running locally
     */
    protected async performDiscovery(): Promise<boolean> {
        try {
            const result = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/api/tags`,
                method: "GET",
                timeout: 5000,
            });
            
            if (result.success) {
                // Cache the models while we're at it
                this.models = result.data.models?.map((m: any) => m.name) || [];
                return true;
            }
            
            return false;
        } catch (error) {
            logger.debug(`[Ollama] Discovery failed: ${error}`);
            return false;
        }
    }
    
    /**
     * Perform health check on Ollama
     */
    protected async performHealthCheck(): Promise<HealthCheckResult> {
        try {
            const result = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/api/tags`,
                method: "GET",
                timeout: this.config!.healthCheck?.timeoutMs || 5000,
            });
            
            if (result.success) {
                const modelCount = result.data.models?.length || 0;
                
                return {
                    healthy: true,
                    message: `Ollama is running with ${modelCount} models`,
                    details: {
                        modelCount,
                        version: result.data.version,
                    },
                    timestamp: new Date(),
                };
            }
            
            return {
                healthy: false,
                message: `Ollama returned status ${result.status}`,
                timestamp: new Date(),
            };
        } catch (error) {
            return {
                healthy: false,
                message: error instanceof Error ? error.message : "Health check failed",
                timestamp: new Date(),
            };
        }
    }
    
    /**
     * List available models - required by IAIResource
     */
    async listModels(): Promise<string[]> {
        if (this._status !== DiscoveryStatus.Available) {
            return [];
        }
        
        try {
            const result = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/api/tags`,
                method: "GET",
            });
            
            if (result.success) {
                this.models = result.data.models?.map((m: any) => m.name) || [];
                return this.models;
            }
            
            return [];
        } catch (error) {
            logger.error("[Ollama] Failed to list models:", error);
            return [];
        }
    }
    
    /**
     * Check if a specific model is available - required by IAIResource
     */
    async hasModel(modelId: string): Promise<boolean> {
        const models = await this.listModels();
        return models.includes(modelId);
    }
    
    /**
     * Provide metadata about the resource
     */
    protected getMetadata(): AIMetadata {
        return {
            version: "0.1.0",
            capabilities: [
                "text-generation",
                "embeddings",
                "completion",
                "chat",
            ],
            supportedModels: this.models,
            lastUpdated: new Date(),
            discoveredAt: this._lastHealthCheck,
            maxTokens: 4096, // Default context window for most Ollama models
            contextWindow: 4096,
        };
    }
    
    /**
     * Pull a model from Ollama
     */
    async pullModel(modelName: string): Promise<void> {
        if (this._status !== DiscoveryStatus.Available) {
            throw new Error("Ollama is not available");
        }
        
        const result = await this.httpClient!.makeRequest({
            url: `${this.config!.baseUrl}/api/pull`,
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: { name: modelName },
            // Pulling can take a long time
            timeout: 600000, // 10 minutes
        });
        
        if (!result.success) {
            throw new Error(`Failed to pull model ${modelName}: ${result.error}`);
        }
        
        // Refresh model list
        await this.listModels();
    }
    
    /**
     * Generate a completion using Ollama
     */
    async generate(
        model: string,
        prompt: string,
        options?: {
            temperature?: number;
            top_p?: number;
            seed?: number;
        },
    ): Promise<string> {
        if (this._health !== ResourceHealth.Healthy) {
            throw new Error("Ollama is not healthy");
        }
        
        // Validate model exists
        if (!await this.hasModel(model)) {
            throw new Error(`Model ${model} not available`);
        }
        
        const result = await this.httpClient!.makeRequest({
            url: `${this.config!.baseUrl}/api/generate`,
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: {
                model,
                prompt,
                stream: false,
                options,
            },
            timeout: this.config!.timeout || 30000,
        });
        
        if (!result.success) {
            throw new Error(`Generation failed: ${result.error}`);
        }
        
        return result.data.response;
    }
}

// ============================================================================
// Usage Examples
// ============================================================================

/**
 * Example: Using the Ollama resource
 */
export async function exampleUsage(): Promise<void> {
    const { ResourceRegistry } = await import("../ResourceRegistry.js");
    const registry = ResourceRegistry.getInstance();
    
    // Get the Ollama resource
    const ollama = registry.getResource<OllamaResource>("ollama");
    
    if (ollama && await ollama.hasModel("llama2")) {
        const response = await ollama.generate(
            "llama2",
            "What is the meaning of life?",
            { temperature: 0.7 },
        );
        console.log(response);
    }
}

/**
 * Example: Configuration in .vrooli/resources.local.json
 */
export const configExample = {
    version: "1.0.0",
    enabled: true,
    services: {
        ai: {
            ollama: {
                enabled: true,
                baseUrl: "http://localhost:11434",
                models: ["llama2", "codellama"],
                keepAlive: "5m",
                numThread: 8,
                healthCheck: {
                    intervalMs: 60000,
                    timeoutMs: 5000,
                },
            },
        },
    },
};
