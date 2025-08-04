/**
 * Whisper Speech-to-Text Resource Implementation
 * 
 * Provides infrastructure-level management for Whisper.cpp service including:
 * - Service discovery and health monitoring
 * - Model availability checking
 * - Transcription endpoint validation
 * - Integration with unified configuration system
 */

import { MINUTES_5_MS } from "@vrooli/shared";
import { logger } from "../../../events/logger.js";
import { ResourceProvider } from "../ResourceProvider.js";
import { RegisterResource } from "../ResourceRegistry.js";
import type { AIMetadata, WhisperConfig } from "../typeRegistry.js";
import type { HealthCheckResult, IAIResource } from "../types.js";
import { DeploymentType, ResourceCategory } from "../types.js";

/**
 * Whisper model information
 */
interface WhisperModel {
    name: string;
    size: string;
    loaded: boolean;
}

/**
 * Production Whisper resource implementation with enhanced monitoring
 */
@RegisterResource
export class WhisperResource extends ResourceProvider<"whisper", WhisperConfig> implements IAIResource {
    // Resource identification
    readonly id = "whisper";
    readonly category = ResourceCategory.AI;
    readonly displayName = "Whisper.cpp";
    readonly description = "Speech-to-text transcription service";
    readonly isSupported = true;
    readonly deploymentType = DeploymentType.Local;

    // Model caching
    private availableModels: Map<string, WhisperModel> = new Map();
    private lastModelRefresh = 0;
    private readonly CACHE_TTL_MS = MINUTES_5_MS;

    // Default Whisper models
    private readonly DEFAULT_MODELS = [
        "tiny",
        "tiny.en",
        "base",
        "base.en",
        "small",
        "small.en",
        "medium",
        "medium.en",
        "large",
        "large-v1",
        "large-v2",
        "large-v3",
    ];

    /**
     * Perform service discovery by checking Whisper API availability
     */
    protected async performDiscovery(): Promise<boolean> {
        if (!this.config?.baseUrl) {
            logger.debug(`[${this.id}] No baseUrl configured`);
            return false;
        }

        try {
            // Try to reach the root endpoint (whisper redirects to /docs)
            const result = await this.httpClient!.makeRequest({
                url: `${this.config.baseUrl}/`,
                method: "GET",
                timeout: this.config.healthCheck?.timeoutMs || 5000,
            });

            // Accept both 200 OK and 307 redirect as success
            return result.success || result.status === 307;
        } catch (error) {
            logger.debug(`[${this.id}] Discovery failed:`, error);
            return false;
        }
    }

    /**
     * Perform comprehensive health check
     */
    protected async performHealthCheck(): Promise<HealthCheckResult> {
        if (!this.config?.baseUrl) {
            return {
                healthy: false,
                message: "No baseUrl configured",
                timestamp: new Date(),
            };
        }

        try {
            // Check root endpoint (whisper redirects to /docs when healthy)
            const healthResult = await this.httpClient!.makeRequest({
                url: `${this.config.baseUrl}/`,
                method: "GET",
                timeout: this.config.healthCheck?.timeoutMs || 5000,
            });

            // Accept both 200 OK and 307 redirect as success
            const isHealthy = healthResult.success || healthResult.status === 307;

            if (!isHealthy) {
                return {
                    healthy: false,
                    message: "Service not responding",
                    timestamp: new Date(),
                    details: { error: healthResult.error },
                };
            }

            // Use configured model size from config
            const modelSize = this.config.modelSize || "large";
            const modelInfo = `Model: ${modelSize}`;

            return {
                healthy: true,
                message: `Whisper service is healthy - ${modelInfo}`,
                timestamp: new Date(),
                details: {
                    baseUrl: this.config.baseUrl,
                    modelSize: this.config.modelSize || "auto",
                },
            };

        } catch (error) {
            logger.error(`[${this.id}] Health check failed:`, error);
            return {
                healthy: false,
                message: error instanceof Error ? error.message : "Health check failed",
                timestamp: new Date(),
            };
        }
    }

    /**
     * Get resource-specific metadata
     */
    protected getMetadata(): AIMetadata {
        return {
            version: "whisper.cpp",
            capabilities: ["transcription", "translation", "speech-to-text", "language-detection"],
            lastUpdated: new Date(),
            discoveredAt: this._lastHealthCheck,
            supportedModels: this.availableModels.size > 0 ? Array.from(this.availableModels.keys()) : this.DEFAULT_MODELS,
            contextWindow: undefined, // Not applicable for Whisper
            maxTokens: undefined, // Not applicable for Whisper
            costPerToken: undefined, // Local deployment - no cost
        };
    }

    /**
     * List available models
     */
    async listModels(): Promise<string[]> {
        await this.refreshModels();
        // Return default models if none were discovered
        return this.availableModels.size > 0 ? Array.from(this.availableModels.keys()) : this.DEFAULT_MODELS;
    }

    /**
     * Check if a specific model is available
     */
    async hasModel(modelId: string): Promise<boolean> {
        await this.refreshModels();
        // If no models discovered, check against default models
        return this.availableModels.size > 0
            ? this.availableModels.has(modelId)
            : this.DEFAULT_MODELS.includes(modelId);
    }

    /**
     * Refresh model information (if endpoint exists)
     */
    private async refreshModels(): Promise<void> {
        const now = Date.now();
        if (now - this.lastModelRefresh < this.CACHE_TTL_MS) {
            return; // Use cached data
        }

        if (!this.config?.baseUrl) {
            return;
        }

        try {
            const result = await this.httpClient!.makeRequest({
                url: `${this.config.baseUrl}/models`,
                method: "GET",
                timeout: 3000,
            });

            if (result.success && result.data) {
                this.availableModels.clear();
                const models = Array.isArray(result.data) ? result.data : [result.data];

                for (const model of models) {
                    if (typeof model === "object" && model.name) {
                        this.availableModels.set(model.name, {
                            name: model.name,
                            size: model.size || "unknown",
                            loaded: model.loaded !== false,
                        });
                    }
                }

                this.lastModelRefresh = now;
                logger.debug(`[${this.id}] Refreshed ${this.availableModels.size} models`);
            }
        } catch (error) {
            logger.debug(`[${this.id}] Models endpoint not available, using default models`);
            // Don't throw - models endpoint is optional
            // The service works fine without it
        }
    }
}
