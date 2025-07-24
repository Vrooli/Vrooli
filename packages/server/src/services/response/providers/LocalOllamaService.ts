import { LlmServiceId, type MessageState, type ThirdPartyModelInfo, ModelFeature, type Tool } from "@vrooli/shared";
import type { ChatCompletionMessageParam } from "openai/resources/chat/completions.js";
import { logger } from "../../../events/logger.js";
import { AIServiceErrorType } from "../registry.js";
import { TokenEstimatorType, type EstimateTokensResult } from "../tokenTypes.js";
import { AIService, type ResponseStreamOptions, type ServiceStreamEvent, type GetOutputTokenLimitParams, type GetOutputTokenLimitResult, type GetResponseCostParams } from "../services.js";
import { generateContextFromMessages } from "../contextGeneration.js";
import { hasHarmfulContent, hasPromptInjection } from "../messageValidation.js";
import { withStreamTimeout } from "../streamTimeout.js";
// Type-safe imports for resource system integration (optional)
import type { ResourceRegistry } from "../../resources/ResourceRegistry.js";
import { ResourceHealth, DiscoveryStatus } from "../../resources/types.js";

/**
 * COST CALCULATION DOCUMENTATION
 * 
 * The Vrooli platform uses a credit system where:
 * - 1 US cent = 1,000,000 credits (API_CREDITS_MULTIPLIER)
 * - Model costs are stored as cents per million tokens
 * - Credit calculation: credits = tokens * cost_per_million_tokens_in_cents
 * 
 * For local models, we use a nominal cost of $0.001 per 1M tokens (0.1 cents)
 * This is for tracking purposes only - local models don't have actual costs.
 */

// Constants
const DEFAULT_CONTEXT_WINDOW = 8192;
const DEFAULT_MAX_OUTPUT_TOKENS = 4096;
const DEFAULT_MAX_TOKENS = 4096;
const TIMEOUT_MINUTES = 10;
const SECONDS_PER_MINUTE = 60;
const MS_PER_SECOND = 1000;
// eslint-disable-next-line no-magic-numbers
const MODEL_REFRESH_INTERVAL_MS = 5 * 60 * 1000; // 5 minutes
const MAX_INPUT_LENGTH = 100000; // 100KB limit for local models
// Local models have nominal cost for tracking purposes
// Using 0.1 cents per million tokens (very low but non-zero)
const NOMINAL_COST_CENTS_PER_MILLION = 0.1; // $0.001 per 1M tokens
const NOMINAL_COST_CREDITS = 1000; // 0.001 cents = 1,000 credits
const BASIC_HEALTH_CHECK_TIMEOUT_MS = 5000; // 5 second timeout for basic health checks

/**
 * Options for LocalOllamaService configuration
 */
export interface LocalOllamaServiceOptions {
    baseUrl?: string;
    defaultModel?: string;
    resourceRegistry?: ResourceRegistry; // Optional dependency injection
}

/**
 * LocalOllamaService - Service for communicating with local Ollama instances
 * Enhanced with optional resource system integration for better health checking
 */
export class LocalOllamaService extends AIService<string, ThirdPartyModelInfo> {
    private readonly baseUrl: string;
    private availableModels: Map<string, any> = new Map();
    private lastModelRefresh = 0;
    private readonly REFRESH_INTERVAL = MODEL_REFRESH_INTERVAL_MS;
    private resourceRegistry?: ResourceRegistry;

    __id: LlmServiceId = LlmServiceId.LocalOllama;
    featureFlags = { supportsStatefulConversations: false };
    defaultModel = "llama3.1:8b";

    constructor(options?: LocalOllamaServiceOptions) {
        super();
        this.baseUrl = options?.baseUrl ?? process.env.OLLAMA_BASE_URL ?? "http://localhost:11434";
        this.defaultModel = options?.defaultModel ?? "llama3.1:8b";
        this.resourceRegistry = options?.resourceRegistry;
        
        // Log configuration (no API key needed for local Ollama)
        logger.info(`[LocalOllamaService] Initialized with base URL: ${this.baseUrl}`, {
            hasResourceIntegration: !!this.resourceRegistry,
        });
    }

    /**
     * Updates the ResourceRegistry after initialization.
     * This allows late-binding of the ResourceRegistry when it becomes available
     * after the AI services have already been initialized.
     */
    public setResourceRegistry(resourceRegistry: ResourceRegistry): void {
        this.resourceRegistry = resourceRegistry;
        logger.info("[LocalOllamaService] ResourceRegistry updated - enhanced health checking now available");
    }

    /**
     * Auto-detect available models from Ollama
     */
    async refreshAvailableModels(): Promise<void> {
        try {
            const response = await fetch(`${this.baseUrl}/api/tags`);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            const data = await response.json();
            
            this.availableModels.clear();
            for (const model of data.models || []) {
                this.availableModels.set(model.name, model);
            }
            
            this.lastModelRefresh = Date.now();
            logger.info(`[LocalOllamaService] Refreshed models: ${Array.from(this.availableModels.keys()).join(", ")}`);
        } catch (error) {
            logger.warn(`[LocalOllamaService] Failed to refresh models: ${error}`);
            // If we can't refresh, keep existing models or set defaults
            if (this.availableModels.size === 0) {
                this.availableModels.set(this.defaultModel, { name: this.defaultModel });
            }
        }
    }

    /**
     * Check if we support a specific model
     */
    async supportsModel(model: string): Promise<boolean> {
        // Refresh models if stale
        if (Date.now() - this.lastModelRefresh > this.REFRESH_INTERVAL) {
            await this.refreshAvailableModels();
        }
        
        return this.availableModels.has(model);
    }

    generateContext(messages: MessageState[], systemMessage?: string): ChatCompletionMessageParam[] {
        return generateContextFromMessages(messages, systemMessage, "LocalOllama");
    }

    async *generateResponseStreaming(options: ResponseStreamOptions): AsyncGenerator<ServiceStreamEvent> {
        // Create inner generator for timeout wrapping
        const innerGenerator = this.generateResponseStreamingInternal(options);
        
        // Wrap with timeout protection (longer timeout for local models)
        yield* withStreamTimeout(innerGenerator, {
            serviceName: "LocalOllama",
            modelName: options.model,
            signal: options.signal,
            timeoutMs: TIMEOUT_MINUTES * SECONDS_PER_MINUTE * MS_PER_SECOND, // 10 minute timeout for local models
        });
    }
    
    private async *generateResponseStreamingInternal(options: ResponseStreamOptions): AsyncGenerator<ServiceStreamEvent> {
        // First check if we support this model
        if (!(await this.supportsModel(options.model))) {
            throw new Error(`Model ${options.model} not available locally`);
        }

        // Convert to Ollama chat format
        const messages = this.generateContext(options.input, options.systemMessage);
        
        const requestBody: any = {
            model: options.model,
            messages,
            stream: true,
            options: {
                temperature: 0.7,
                num_predict: options.maxTokens || DEFAULT_MAX_TOKENS,
            },
        };

        // Add tools if provided (Ollama uses OpenAI-compatible format)
        if (options.tools && options.tools.length > 0) {
            requestBody.tools = options.tools;
        }

        const response = await fetch(`${this.baseUrl}/api/chat`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(requestBody),
            signal: options.signal,
        });

        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const reader = response.body?.getReader();
        if (!reader) {
            throw new Error("No response body available");
        }

        const decoder = new TextDecoder();

        try {
            while (true) {
                const { done, value } = await reader.read();
                if (done) break;

                const chunk = decoder.decode(value);
                const lines = chunk.split("\n").filter(line => line.trim());

                for (const line of lines) {
                    try {
                        const data = JSON.parse(line);
                        
                        if (data.message?.content) {
                            yield { type: "text", content: data.message.content };
                        }

                        // Handle tool calls (Ollama uses OpenAI-compatible format)
                        if (data.message?.tool_calls) {
                            for (const toolCall of data.message.tool_calls) {
                                if (toolCall.function) {
                                    let parsedArgs: Record<string, unknown> = {};
                                    try {
                                        parsedArgs = typeof toolCall.function.arguments === "string" 
                                            ? JSON.parse(toolCall.function.arguments) 
                                            : toolCall.function.arguments;
                                    } catch (e) {
                                        logger.warn(`Failed to parse tool call arguments for ${toolCall.function.name}`, { error: e });
                                    }
                                    
                                    yield {
                                        type: "function_call",
                                        name: toolCall.function.name,
                                        arguments: parsedArgs,
                                        callId: toolCall.id,
                                    };
                                }
                            }
                        }

                        if (data.done) {
                            yield { type: "done", cost: NOMINAL_COST_CREDITS }; // Nominal cost for tracking
                            return;
                        }
                    } catch (e) {
                        // Skip malformed JSON
                    }
                }
            }
        } finally {
            reader.releaseLock();
        }
    }

    getContextSize(_requestedModel?: string | null): number {
        // Conservative default for most local models
        return DEFAULT_CONTEXT_WINDOW;
    }

    getModelInfo(): Record<string, ThirdPartyModelInfo> {
        const modelInfo: Record<string, ThirdPartyModelInfo> = {};
        
        // Build dynamic model info based on available models
        for (const modelName of Array.from(this.availableModels.keys())) {
            modelInfo[modelName] = {
                enabled: true,
                name: `Ollama_${modelName.replace(/[^a-zA-Z0-9]/g, "_")}`,
                descriptionShort: `Local Ollama model: ${modelName}`,
                inputCost: NOMINAL_COST_CENTS_PER_MILLION,  // $0.001 per 1M tokens
                outputCost: NOMINAL_COST_CENTS_PER_MILLION,
                contextWindow: DEFAULT_CONTEXT_WINDOW,  // Conservative default
                maxOutputTokens: DEFAULT_MAX_OUTPUT_TOKENS,
                features: {
                    [ModelFeature.FunctionCalling]: { type: "generic", notes: "Basic function calling support" },
                },
                supportsReasoning: false,
            };
        }
        
        // If no models available, provide default
        if (Object.keys(modelInfo).length === 0) {
            modelInfo[this.defaultModel] = {
                enabled: true,
                name: `Ollama_${this.defaultModel.replace(/[^a-zA-Z0-9]/g, "_")}`,
                descriptionShort: `Local Ollama model: ${this.defaultModel}`,
                inputCost: NOMINAL_COST_CENTS_PER_MILLION,
                outputCost: NOMINAL_COST_CENTS_PER_MILLION,
                contextWindow: DEFAULT_CONTEXT_WINDOW,
                maxOutputTokens: DEFAULT_MAX_OUTPUT_TOKENS,
                features: {},
                supportsReasoning: false,
            };
        }
        
        return modelInfo;
    }

    getMaxOutputTokens(_requestedModel?: string | null): number {
        // Conservative default for most local models
        return DEFAULT_MAX_OUTPUT_TOKENS;
    }

    getMaxOutputTokensRestrained(params: GetOutputTokenLimitParams): number {
        // For local models, we don't have strict cost constraints
        return Math.min(this.getMaxOutputTokens(params.model), DEFAULT_MAX_OUTPUT_TOKENS);
    }

    getResponseCost(params: GetResponseCostParams): number {
        // Calculate nominal cost for local models (tracking purposes)
        // Credits = tokens * cost_per_million_tokens_in_cents
        const inputCredits = params.usage.input * NOMINAL_COST_CENTS_PER_MILLION;
        const outputCredits = params.usage.output * NOMINAL_COST_CENTS_PER_MILLION;
        return inputCredits + outputCredits;
    }

    getEstimationInfo(_model?: string | null): Pick<EstimateTokensResult, "estimationModel" | "encoding"> {
        return {
            estimationModel: TokenEstimatorType.Default,
            encoding: "cl100k_base",
        };
    }

    getModel(model?: string | null): string {
        if (!model) return this.defaultModel;
        
        // Check if requested model is available
        if (this.availableModels.has(model)) {
            return model;
        }
        
        // Return default if not available
        return this.defaultModel;
    }

    getErrorType(error: unknown): AIServiceErrorType {
        if (error instanceof Error) {
            const message = error.message.toLowerCase();
            
            if (message.includes("unauthorized") || message.includes("authentication")) {
                return AIServiceErrorType.Authentication;
            }
            
            if (message.includes("rate limit") || message.includes("too many requests")) {
                return AIServiceErrorType.RateLimit;
            }
            
            if (message.includes("overload") || message.includes("capacity")) {
                return AIServiceErrorType.Overloaded;
            }
            
            if (message.includes("bad request") || message.includes("invalid")) {
                return AIServiceErrorType.InvalidRequest;
            }
        }
        
        return AIServiceErrorType.ApiError;
    }

    async safeInputCheck(input: string): Promise<GetOutputTokenLimitResult> {
        // Check for harmful content
        const harmfulCheck = hasHarmfulContent(input);
        if (harmfulCheck.isHarmful) {
            logger.warn("Local Ollama safety check flagged harmful content", { 
                pattern: harmfulCheck.pattern,
                inputLength: input.length, 
            });
            return { cost: 0, isSafe: false };
        }

        // Check for prompt injection attempts
        const injectionCheck = hasPromptInjection(input);
        if (injectionCheck.isInjection) {
            logger.warn("Local Ollama safety check flagged potential prompt injection", { 
                pattern: injectionCheck.pattern,
                inputLength: input.length, 
            });
            return { cost: 0, isSafe: false };
        }

        // Check for excessive length (potential abuse)
        if (input.length > MAX_INPUT_LENGTH) {
            logger.warn("Local Ollama safety check flagged excessive input length", { 
                inputLength: input.length,
                maxLength: MAX_INPUT_LENGTH,
            });
            return { cost: 0, isSafe: false };
        }

        // For production, consider integrating with external moderation services
        return { cost: 0, isSafe: true };
    }

    getNativeToolCapabilities(): Array<Pick<Tool, "name" | "description">> {
        // Ollama supports function calling with compatible models (Llama 3.1+, Qwen2.5, etc.)
        return [
            { name: "function", description: "Execute custom functions with structured parameters" },
        ];
    }

    /**
     * Checks if the LocalOllama service is healthy and available.
     * Enhanced with optional resource system integration for more sophisticated health checking.
     * @returns true if the service is healthy and can accept requests, false otherwise
     */
    async isHealthy(): Promise<boolean> {
        try {
            // Enhanced health checking: try resource system first if available
            if (this.resourceRegistry) {
                try {
                    const ollamaResource = this.resourceRegistry.getResource("ollama");
                    
                    if (ollamaResource) {
                        const resourceInfo = ollamaResource.getPublicInfo();
                        const isResourceHealthy = resourceInfo.health === ResourceHealth.Healthy && 
                                                resourceInfo.status === DiscoveryStatus.Available;
                        
                        if (isResourceHealthy) {
                            logger.debug("[LocalOllamaService] Resource system reports Ollama as healthy");
                            return true;
                        } else {
                            logger.warn(`[LocalOllamaService] Resource system reports Ollama as unhealthy: ${resourceInfo.health}/${resourceInfo.status}`);
                            return false;
                        }
                    } else {
                        logger.debug("[LocalOllamaService] Ollama resource not found in registry, falling back to basic health check");
                    }
                } catch (resourceError) {
                    logger.debug("[LocalOllamaService] Resource system health check failed, falling back to basic check", resourceError);
                }
            }

            // Fallback to basic health checking if resource system unavailable or failed
            logger.debug("[LocalOllamaService] Performing basic health check");
            
            return await this.performBasicHealthCheck();
        } catch (error) {
            logger.error(`[LocalOllamaService] Health check error: ${error}`);
            return false;
        }
    }

    /**
     * Performs basic health check by calling Ollama API directly
     * @private
     */
    private async performBasicHealthCheck(): Promise<boolean> {
        try {
            // Attempt to fetch the tags endpoint which should be available if Ollama is running
            const response = await fetch(`${this.baseUrl}/api/tags`, {
                method: "GET",
                headers: {
                    "Content-Type": "application/json",
                },
                // Add a timeout to prevent hanging
                signal: AbortSignal.timeout(BASIC_HEALTH_CHECK_TIMEOUT_MS), // 5 second timeout
            });

            // Check if the response is successful
            if (!response.ok) {
                logger.warn(`[LocalOllamaService] Basic health check failed: HTTP ${response.status}`);
                return false;
            }

            // Verify we can parse the response
            const data = await response.json();
            
            // Check if we have at least one model available
            const hasModels = data.models && Array.isArray(data.models) && data.models.length > 0;
            
            if (!hasModels) {
                logger.warn("[LocalOllamaService] Basic health check: No models available");
                return false;
            }

            logger.debug("[LocalOllamaService] Basic health check passed");
            return true;
        } catch (error) {
            logger.error("[LocalOllamaService] Basic health check failed:", error);
            return false;
        }
    }
}
