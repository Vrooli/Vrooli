/**
 * Example implementation of an Ollama resource provider.
 * Demonstrates how to implement a resource for AI model inference.
 */

import { ResourceProvider, RegisterResource } from "../ResourceProvider.js";
import { ResourceCategory, DeploymentType, ResourceHealth, DiscoveryStatus } from "../types.js";
import type { HealthCheckResult, IAIResource, BaseResourceConfig } from "../types.js";
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
export class OllamaResource extends ResourceProvider<OllamaConfig> implements IAIResource {
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
            const response = await this.fetchWithTimeout(
                `${this.config!.baseUrl}/api/tags`,
                {},
                5000,
            );
            
            if (response.ok) {
                // Cache the models while we're at it
                const data = await response.json();
                this.models = data.models?.map((m: any) => m.name) || [];
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
            const response = await this.fetchWithTimeout(
                `${this.config!.baseUrl}/api/tags`,
                {},
                this.config!.healthCheck?.timeoutMs || 5000,
            );
            
            if (response.ok) {
                const data = await response.json();
                const modelCount = data.models?.length || 0;
                
                return {
                    healthy: true,
                    message: `Ollama is running with ${modelCount} models`,
                    details: {
                        modelCount,
                        version: data.version,
                    },
                    timestamp: new Date(),
                };
            }
            
            return {
                healthy: false,
                message: `Ollama returned status ${response.status}`,
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
            const response = await this.fetchWithTimeout(
                `${this.config!.baseUrl}/api/tags`,
            );
            
            if (response.ok) {
                const data = await response.json();
                this.models = data.models?.map((m: any) => m.name) || [];
                return this.models;
            }
            
            return [];
        } catch (error) {
            logger.error(`[Ollama] Failed to list models:`, error);
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
    protected getMetadata(): Record<string, any> {
        return {
            version: "0.1.0",
            capabilities: [
                "text-generation",
                "embeddings",
                "completion",
                "chat",
            ],
            supportedModels: this.models,
            baseUrl: this.config?.baseUrl,
        };
    }
    
    /**
     * Pull a model from Ollama
     */
    async pullModel(modelName: string): Promise<void> {
        if (this._status !== DiscoveryStatus.Available) {
            throw new Error("Ollama is not available");
        }
        
        const response = await this.fetchWithTimeout(
            `${this.config!.baseUrl}/api/pull`,
            {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ name: modelName }),
            },
            // Pulling can take a long time
            600000, // 10 minutes
        );
        
        if (!response.ok) {
            throw new Error(`Failed to pull model ${modelName}: ${response.statusText}`);
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
        
        const response = await this.fetchWithTimeout(
            `${this.config!.baseUrl}/api/generate`,
            {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    model,
                    prompt,
                    stream: false,
                    options,
                }),
            },
            this.config!.timeout || 30000,
        );
        
        if (!response.ok) {
            throw new Error(`Generation failed: ${response.statusText}`);
        }
        
        const data = await response.json();
        return data.response;
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