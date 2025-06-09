/**
 * LLM Service - Interface for natural language reasoning in Tier 1
 * 
 * Provides LLM integration for strategic decision making and analysis.
 * This enables true natural language reasoning as documented in the architecture.
 */

import { type Logger } from "winston";
import { EventBus } from "../../cross-cutting/eventBus.js";

/**
 * LLM completion request
 */
export interface CompletionRequest {
    prompt: string;
    maxTokens?: number;
    temperature?: number;
    model?: string;
    systemPrompt?: string;
}

/**
 * LLM completion response
 */
export interface CompletionResponse {
    text: string;
    usage: {
        promptTokens: number;
        completionTokens: number;
        totalTokens: number;
    };
    model: string;
    finishReason: "stop" | "length" | "error";
}

/**
 * LLM provider interface
 */
export interface LLMProvider {
    name: string;
    complete(request: CompletionRequest): Promise<CompletionResponse>;
    isAvailable(): Promise<boolean>;
}

/**
 * LLM Service
 * 
 * Manages LLM providers and provides natural language reasoning capabilities
 * for Tier 1 strategic coordination.
 */
export class LLMService {
    private readonly logger: Logger;
    private readonly eventBus: EventBus;
    private readonly providers: Map<string, LLMProvider> = new Map();
    private defaultProvider?: string;

    constructor(logger: Logger, eventBus: EventBus) {
        this.logger = logger;
        this.eventBus = eventBus;
    }

    /**
     * Registers an LLM provider
     */
    registerProvider(provider: LLMProvider): void {
        this.providers.set(provider.name, provider);
        
        if (!this.defaultProvider) {
            this.defaultProvider = provider.name;
        }

        this.logger.info("[LLMService] Registered provider", {
            provider: provider.name,
            isDefault: this.defaultProvider === provider.name,
        });
    }

    /**
     * Sets the default provider
     */
    setDefaultProvider(providerName: string): void {
        if (!this.providers.has(providerName)) {
            throw new Error(`Provider not found: ${providerName}`);
        }
        
        this.defaultProvider = providerName;
        this.logger.info("[LLMService] Set default provider", { provider: providerName });
    }

    /**
     * Completes a prompt using natural language reasoning
     */
    async complete(
        prompt: string,
        options?: {
            provider?: string;
            maxTokens?: number;
            temperature?: number;
            systemPrompt?: string;
        },
    ): Promise<string> {
        const providerName = options?.provider || this.defaultProvider;
        
        if (!providerName) {
            throw new Error("No LLM provider available");
        }

        const provider = this.providers.get(providerName);
        if (!provider) {
            throw new Error(`Provider not found: ${providerName}`);
        }

        this.logger.debug("[LLMService] Requesting completion", {
            provider: providerName,
            promptLength: prompt.length,
            maxTokens: options?.maxTokens,
        });

        try {
            // Check provider availability
            const isAvailable = await provider.isAvailable();
            if (!isAvailable) {
                throw new Error(`Provider ${providerName} is not available`);
            }

            // Make completion request
            const request: CompletionRequest = {
                prompt,
                maxTokens: options?.maxTokens || 1000,
                temperature: options?.temperature || 0.7,
                systemPrompt: options?.systemPrompt,
            };

            const response = await provider.complete(request);

            // Emit completion event for monitoring
            await this.eventBus.publish("llm.events", {
                type: "COMPLETION_REQUESTED",
                timestamp: new Date(),
                metadata: {
                    provider: providerName,
                    promptTokens: response.usage.promptTokens,
                    completionTokens: response.usage.completionTokens,
                    totalTokens: response.usage.totalTokens,
                    model: response.model,
                    finishReason: response.finishReason,
                },
            });

            this.logger.debug("[LLMService] Completion successful", {
                provider: providerName,
                usage: response.usage,
                finishReason: response.finishReason,
            });

            return response.text;

        } catch (error) {
            this.logger.error("[LLMService] Completion failed", {
                provider: providerName,
                error: error instanceof Error ? error.message : String(error),
            });

            // Emit error event
            await this.eventBus.publish("llm.events", {
                type: "COMPLETION_FAILED",
                timestamp: new Date(),
                metadata: {
                    provider: providerName,
                    error: error instanceof Error ? error.message : String(error),
                },
            });

            throw error;
        }
    }

    /**
     * Strategic analysis using natural language reasoning
     */
    async analyzeStrategically(
        context: string,
        question: string,
    ): Promise<string> {
        const systemPrompt = `You are a strategic AI coordinator with deep expertise in:
- Complex problem decomposition
- Resource optimization
- Risk assessment
- Strategic planning
- System coordination

Provide clear, actionable strategic analysis.`;

        const prompt = `
Context:
${context}

Strategic Question:
${question}

Provide a strategic analysis addressing the question in the given context.`;

        return await this.complete(prompt, {
            systemPrompt,
            temperature: 0.3, // Lower temperature for strategic analysis
            maxTokens: 1500,
        });
    }

    /**
     * Decision generation using natural language reasoning
     */
    async generateDecision(
        goal: string,
        constraints: Record<string, unknown>,
        options: string[],
    ): Promise<string> {
        const systemPrompt = `You are a strategic decision maker. Generate specific, actionable decisions in the format:
decision_name(parameters)

Consider constraints carefully and provide rationale.`;

        const prompt = `
Goal: ${goal}

Constraints:
${JSON.stringify(constraints, null, 2)}

Available Options:
${options.map((opt, i) => `${i + 1}. ${opt}`).join('\n')}

Generate the best strategic decision with brief rationale.`;

        return await this.complete(prompt, {
            systemPrompt,
            temperature: 0.5,
            maxTokens: 500,
        });
    }

    /**
     * Gets available providers
     */
    getProviders(): string[] {
        return Array.from(this.providers.keys());
    }

    /**
     * Gets provider status
     */
    async getProviderStatus(): Promise<Record<string, { available: boolean; isDefault: boolean }>> {
        const status: Record<string, { available: boolean; isDefault: boolean }> = {};

        for (const [name, provider] of this.providers) {
            try {
                const available = await provider.isAvailable();
                status[name] = {
                    available,
                    isDefault: name === this.defaultProvider,
                };
            } catch (error) {
                status[name] = {
                    available: false,
                    isDefault: name === this.defaultProvider,
                };
            }
        }

        return status;
    }
}