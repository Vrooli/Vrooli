import { API_CREDITS_MULTIPLIER, LlmServiceId, type MessageState, type ThirdPartyModelInfo, type Tool } from "@vrooli/shared";
import type { ChatCompletionMessageParam } from "openai/resources/chat/completions.js";
import { logger } from "../../../events/logger.js";
import { AIServiceErrorType } from "../registry.js";
import { TokenEstimatorType, type EstimateTokensResult } from "../tokenTypes.js";
import { AIService, type ResponseStreamOptions, type ServiceStreamEvent, type GetOutputTokenLimitParams, type GetOutputTokenLimitResult, type GetResponseCostParams } from "../services.js";
import { generateContextFromMessages } from "../contextGeneration.js";
import { hasHarmfulContent, hasPromptInjection } from "../messageValidation.js";
import { withStreamTimeout } from "../streamTimeout.js";

/**
 * COST CALCULATION DOCUMENTATION
 * 
 * The Vrooli platform uses a credit system where:
 * - 1 US cent = 1,000,000 credits (API_CREDITS_MULTIPLIER)
 * - Model costs are stored as cents per million tokens
 * - Credit calculation: credits = tokens * cost_per_million_tokens_in_cents
 * 
 * Example: GPT-4o costs $2.50 per 1M input tokens
 * - Cost in cents: 250 cents per 1M tokens
 * - For 1000 tokens: 1000 * 250 = 250,000 credits
 * - In dollars: 250,000 / 1,000,000 = $0.0025
 */

// Constants
const DEFAULT_CONTEXT_WINDOW = 8192;
const DEFAULT_MAX_OUTPUT_TOKENS = 4096;
const DEFAULT_MAX_TOKENS = 4096;
const MAX_INPUT_LENGTH = 50000; // 50KB limit for OpenRouter API
const TIMEOUT_MINUTES = 5;
const SECONDS_PER_MINUTE = 60;
const MS_PER_SECOND = 1000;
// Model costs are in cents per million tokens
// API credits = tokens * cost_per_million_tokens_in_cents
const FALLBACK_INPUT_COST_CENTS_PER_MILLION = 15;  // $0.15 per 1M tokens
const FALLBACK_OUTPUT_COST_CENTS_PER_MILLION = 60; // $0.60 per 1M tokens

/**
 * OpenRouterService - Service for communicating with OpenRouter API
 */
export class OpenRouterService extends AIService<string, ThirdPartyModelInfo> {
    private readonly apiKey: string;
    private readonly baseUrl = "https://openrouter.ai/api/v1";
    private readonly appName: string;
    private readonly siteUrl: string;

    __id: LlmServiceId = LlmServiceId.OpenRouter;
    featureFlags = { supportsStatefulConversations: false };
    defaultModel = "openai/gpt-4o-mini";

    constructor(options?: { apiKey?: string; appName?: string; siteUrl?: string; defaultModel?: string }) {
        super();
        this.apiKey = options?.apiKey ?? process.env.OPENROUTER_API_KEY ?? "";
        this.appName = options?.appName ?? process.env.OPENROUTER_APP_NAME ?? "Vrooli";
        this.siteUrl = options?.siteUrl ?? process.env.OPENROUTER_SITE_URL ?? "https://vrooli.com";
        this.defaultModel = options?.defaultModel ?? "openai/gpt-4o-mini";
        
        // Validate required configuration
        if (!this.apiKey) {
            logger.warn("[OpenRouterService] No API key configured - service will not function");
        }
    }

    /**
     * Check if we support a specific model
     */
    async supportsModel(model: string): Promise<boolean> {
        // OpenRouter supports 50+ models, very permissive
        // Models typically follow the pattern: provider/model-name
        return model.includes("/") || model.startsWith("openai/") || model.startsWith("anthropic/") || 
               model.startsWith("meta-llama/") || model.startsWith("google/") || model.startsWith("mistralai/");
    }

    generateContext(messages: MessageState[], systemMessage?: string): ChatCompletionMessageParam[] {
        return generateContextFromMessages(messages, systemMessage, "OpenRouter");
    }

    async *generateResponseStreaming(options: ResponseStreamOptions): AsyncGenerator<ServiceStreamEvent> {
        // Create inner generator for timeout wrapping
        const innerGenerator = this.generateResponseStreamingInternal(options);
        
        // Wrap with timeout protection
        yield* withStreamTimeout(innerGenerator, {
            serviceName: "OpenRouter",
            modelName: options.model,
            signal: options.signal,
            timeoutMs: TIMEOUT_MINUTES * SECONDS_PER_MINUTE * MS_PER_SECOND, // 5 minute timeout
        });
    }
    
    private async *generateResponseStreamingInternal(options: ResponseStreamOptions): AsyncGenerator<ServiceStreamEvent> {
        if (!(await this.supportsModel(options.model))) {
            throw new Error(`Model ${options.model} not supported by OpenRouter`);
        }

        // Track token usage for fallback cost calculation
        let inputTokens = 0;
        let outputTokens = 0;
        
        const messages = this.generateContext(options.input, options.systemMessage);
        // Estimate input tokens (rough: 4 chars = 1 token)
        inputTokens = Math.ceil(messages.reduce((sum, msg) => sum + (msg.content?.toString().length || 0), 0) / 4);
        
        const requestBody: any = {
            model: options.model,
            messages,
            stream: true,
            max_tokens: options.maxTokens || DEFAULT_MAX_TOKENS,
            temperature: 0.7,
        };

        // Add tools if provided (OpenRouter uses OpenAI-compatible format)
        if (options.tools && options.tools.length > 0) {
            requestBody.tools = options.tools;
        }

        const response = await fetch(`${this.baseUrl}/chat/completions`, {
            method: "POST",
            headers: {
                "Authorization": `Bearer ${this.apiKey}`,
                "Content-Type": "application/json",
                "HTTP-Referer": this.siteUrl,
                "X-Title": this.appName,
            },
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
                const lines = chunk.split("\n").filter(line => line.startsWith("data: "));

                for (const line of lines) {
                    try {
                        // Remove 'data: ' prefix
                        const dataContent = line.substring("data: ".length);
                        
                        if (dataContent.trim() === "[DONE]") {
                            // OpenRouter provides cost in response headers
                            const cost = this.calculateOpenRouterCost(response.headers, options.model, inputTokens, outputTokens);
                            yield { type: "done", cost };
                            return;
                        }
                        
                        const data = JSON.parse(dataContent);
                        
                        if (data.choices?.[0]?.delta?.content) {
                            const content = data.choices[0].delta.content;
                            // Rough estimate: 4 chars = 1 token
                            outputTokens += Math.ceil(content.length / 4);
                            yield { type: "text", content };
                        }

                        // Handle tool calls (OpenRouter uses OpenAI-compatible format)
                        if (data.choices?.[0]?.delta?.tool_calls) {
                            for (const toolCall of data.choices[0].delta.tool_calls) {
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

                        if (data.choices?.[0]?.finish_reason) {
                            // Extract usage from response if available
                            const usage = data.usage;
                            if (usage) {
                                const cost = this.calculateOpenRouterCostFromUsage(options.model, usage.prompt_tokens, usage.completion_tokens);
                                yield { type: "done", cost };
                            } else {
                                // Use accumulated token counts
                                const cost = this.calculateOpenRouterCost(response.headers, options.model, inputTokens, outputTokens);
                                yield { type: "done", cost };
                            }
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

    private calculateOpenRouterCost(headers: Headers, model?: string, inputTokens?: number, outputTokens?: number): number {
        // OpenRouter provides cost in response headers
        const totalCost = headers.get("x-ratelimit-cost") || headers.get("x-cost");
        if (!totalCost) {
            // Use model-based fallback pricing when headers are missing
            logger.warn("OpenRouter cost headers not found - using model-based fallback pricing", { 
                model,
                inputTokens,
                outputTokens,
            });
            
            // If we have token counts, use model-specific pricing
            if (model && inputTokens !== undefined && outputTokens !== undefined) {
                return this.calculateOpenRouterCostFromUsage(model, inputTokens, outputTokens);
            }
            
            // Conservative fallback: assume 1000 tokens each way at default pricing
            const fallbackInputTokens = 1000;
            const fallbackOutputTokens = 1000;
            return this.calculateOpenRouterCostFromUsage(model || this.defaultModel, fallbackInputTokens, fallbackOutputTokens);
        }
        // Cost is in dollars, convert to credits
        const costInDollars = parseFloat(totalCost);
        const costInCents = costInDollars * 100;
        return Math.floor(costInCents * Number(API_CREDITS_MULTIPLIER));
    }

    private calculateOpenRouterCostFromUsage(model: string, inputTokens: number, outputTokens: number): number {
        // Calculate cost in API credits
        // Model costs are in cents per million tokens
        // Credits = tokens * cost_per_million_tokens_in_cents
        const modelInfo = this.getModelInfo()[model];
        if (modelInfo && modelInfo.inputCost && modelInfo.outputCost) {
            const inputCredits = inputTokens * modelInfo.inputCost;
            const outputCredits = outputTokens * modelInfo.outputCost;
            return inputCredits + outputCredits;
        }
        
        // Fallback pricing using default costs
        const fallbackInputCredits = inputTokens * FALLBACK_INPUT_COST_CENTS_PER_MILLION;
        const fallbackOutputCredits = outputTokens * FALLBACK_OUTPUT_COST_CENTS_PER_MILLION;
        return fallbackInputCredits + fallbackOutputCredits;
    }

    getContextSize(requestedModel?: string | null): number {
        const model = this.getModel(requestedModel);
        const modelInfo = this.getModelInfo()[model];
        return modelInfo?.contextWindow || DEFAULT_CONTEXT_WINDOW;
    }

    getModelInfo(): Record<string, ThirdPartyModelInfo> {
        // Return model info for popular OpenRouter models
        // IMPORTANT: inputCost and outputCost are in cents per million tokens
        return {
            "openai/gpt-4o": {
                enabled: true,
                name: "GPT-4o",
                descriptionShort: "GPT-4o via OpenRouter",
                inputCost: 250,    // $2.50 per 1M input tokens
                outputCost: 1000,  // $10.00 per 1M output tokens
                contextWindow: 128000,
                maxOutputTokens: 4096,
                features: {},
                supportsReasoning: false,
            },
            "openai/gpt-4o-mini": {
                enabled: true,
                name: "GPT-4o Mini",
                descriptionShort: "GPT-4o Mini via OpenRouter",
                inputCost: 15,     // $0.15 per 1M input tokens
                outputCost: 60,    // $0.60 per 1M output tokens
                contextWindow: 128000,
                maxOutputTokens: 16384,
                features: {},
                supportsReasoning: false,
            },
            "anthropic/claude-3.5-sonnet": {
                enabled: true,
                name: "Claude 3.5 Sonnet",
                descriptionShort: "Claude 3.5 Sonnet via OpenRouter",
                inputCost: 300,    // $3.00 per 1M input tokens
                outputCost: 1500,  // $15.00 per 1M output tokens
                contextWindow: 200000,
                maxOutputTokens: 8192,
                features: {},
                supportsReasoning: false,
            },
            "anthropic/claude-3-haiku": {
                enabled: true,
                name: "Claude 3 Haiku",
                descriptionShort: "Claude 3 Haiku via OpenRouter",
                inputCost: 25,     // $0.25 per 1M input tokens
                outputCost: 125,   // $1.25 per 1M output tokens
                contextWindow: 200000,
                maxOutputTokens: 4096,
                features: {},
                supportsReasoning: false,
            },
            "meta-llama/llama-3.1-8b-instruct": {
                enabled: true,
                name: "Llama 3.1 8B",
                descriptionShort: "Llama 3.1 8B via OpenRouter",
                inputCost: 6,      // $0.06 per 1M tokens
                outputCost: 6,     // $0.06 per 1M tokens
                contextWindow: 131072,
                maxOutputTokens: 4096,
                features: {},
                supportsReasoning: false,
            },
            "meta-llama/llama-3.1-70b-instruct": {
                enabled: true,
                name: "Llama 3.1 70B",
                descriptionShort: "Llama 3.1 70B via OpenRouter",
                inputCost: 40,     // $0.40 per 1M tokens
                outputCost: 40,    // $0.40 per 1M tokens
                contextWindow: 131072,
                maxOutputTokens: 4096,
                features: {},
                supportsReasoning: false,
            },
            "meta-llama/llama-3.1-405b-instruct": {
                enabled: true,
                name: "Llama 3.1 405B",
                descriptionShort: "Llama 3.1 405B via OpenRouter",
                inputCost: 300,    // $3.00 per 1M tokens
                outputCost: 300,   // $3.00 per 1M tokens
                contextWindow: 131072,
                maxOutputTokens: 4096,
                features: {},
                supportsReasoning: false,
            },
            "google/gemini-pro-1.5": {
                enabled: true,
                name: "Gemini 1.5 Pro",
                descriptionShort: "Gemini 1.5 Pro via OpenRouter",
                inputCost: 125,    // $1.25 per 1M input tokens
                outputCost: 375,   // $3.75 per 1M output tokens
                contextWindow: 2097152,
                maxOutputTokens: 8192,
                features: {},
                supportsReasoning: false,
            },
            "google/gemini-flash-1.5": {
                enabled: true,
                name: "Gemini 1.5 Flash",
                descriptionShort: "Gemini 1.5 Flash via OpenRouter",
                inputCost: 7,      // $0.07 per 1M input tokens
                outputCost: 21,    // $0.21 per 1M output tokens
                contextWindow: 1048576,
                maxOutputTokens: 8192,
                features: {},
                supportsReasoning: false,
            },
            "mistralai/mistral-large-2407": {
                enabled: true,
                name: "Mistral Large 2",
                descriptionShort: "Mistral Large 2 via OpenRouter",
                inputCost: 300,    // $3.00 per 1M input tokens
                outputCost: 900,   // $9.00 per 1M output tokens
                contextWindow: 128000,
                maxOutputTokens: 4096,
                features: {},
                supportsReasoning: false,
            },
            "mistralai/codestral-mamba": {
                enabled: true,
                name: "Codestral",
                descriptionShort: "Codestral via OpenRouter",
                inputCost: 25,     // $0.25 per 1M tokens
                outputCost: 25,    // $0.25 per 1M tokens
                contextWindow: 32768,
                maxOutputTokens: 4096,
                features: {},
                supportsReasoning: false,
            },
        };
    }

    getMaxOutputTokens(requestedModel?: string | null): number {
        const model = this.getModel(requestedModel);
        const modelInfo = this.getModelInfo()[model];
        return modelInfo?.maxOutputTokens || DEFAULT_MAX_OUTPUT_TOKENS;
    }

    getMaxOutputTokensRestrained(params: GetOutputTokenLimitParams): number {
        const model = this.getModel(params.model);
        const modelInfo = this.getModelInfo()[model];

        if (!modelInfo || !modelInfo.inputCost || !modelInfo.outputCost) {
            return this.getMaxOutputTokens(model);
        }

        // Calculate input cost in credits
        // Credits = tokens * cost_per_million_tokens_in_cents
        const inputCredits = BigInt(params.inputTokens) * BigInt(modelInfo.inputCost);
        const remainingCredits = params.maxCredits - inputCredits;

        if (remainingCredits <= BigInt(0)) {
            return 0;
        }

        // Calculate max output tokens based on remaining credits
        // tokens = credits / cost_per_million_tokens_in_cents
        const maxOutputTokensFromCredits = remainingCredits / BigInt(modelInfo.outputCost);
        const modelMaxTokens = this.getMaxOutputTokens(model);
        
        return Math.min(
            Number(maxOutputTokensFromCredits),
            modelMaxTokens,
        );
    }

    getResponseCost(params: GetResponseCostParams): number {
        const model = this.getModel(params.model);
        const modelInfo = this.getModelInfo()[model];

        // Calculate cost using model pricing
        // Credits = tokens * cost_per_million_tokens_in_cents
        if (modelInfo && modelInfo.inputCost && modelInfo.outputCost) {
            const inputCredits = params.usage.input * modelInfo.inputCost;
            const outputCredits = params.usage.output * modelInfo.outputCost;
            return inputCredits + outputCredits;
        }

        // Fallback cost calculation
        const fallbackInputCredits = params.usage.input * FALLBACK_INPUT_COST_CENTS_PER_MILLION;
        const fallbackOutputCredits = params.usage.output * FALLBACK_OUTPUT_COST_CENTS_PER_MILLION;
        return fallbackInputCredits + fallbackOutputCredits;
    }

    getEstimationInfo(_model?: string | null): Pick<EstimateTokensResult, "estimationModel" | "encoding"> {
        return {
            estimationModel: TokenEstimatorType.Default,
            encoding: "cl100k_base",
        };
    }

    getModel(model?: string | null): string {
        if (!model) return this.defaultModel;
        
        // If it's already in OpenRouter format, return as is
        if (model.includes("/")) {
            return model;
        }
        
        // Try to map common model names to OpenRouter equivalents
        const modelMapping: Record<string, string> = {
            "gpt-4o": "openai/gpt-4o",
            "gpt-4o-mini": "openai/gpt-4o-mini",
            "gpt-4-turbo": "openai/gpt-4-turbo",
            "gpt-4": "openai/gpt-4",
            "gpt-3.5-turbo": "openai/gpt-3.5-turbo",
            "claude-3.5-sonnet": "anthropic/claude-3.5-sonnet",
            "claude-3-opus": "anthropic/claude-3-opus",
            "claude-3-sonnet": "anthropic/claude-3-sonnet",
            "claude-3-haiku": "anthropic/claude-3-haiku",
            "llama-3.1-8b": "meta-llama/llama-3.1-8b-instruct",
            "llama-3.1-70b": "meta-llama/llama-3.1-70b-instruct",
            "llama-3.1-405b": "meta-llama/llama-3.1-405b-instruct",
            "gemini-pro-1.5": "google/gemini-pro-1.5",
            "gemini-flash-1.5": "google/gemini-flash-1.5",
            "mistral-large-2": "mistralai/mistral-large-2407",
            "codestral": "mistralai/codestral-mamba",
        };
        
        return modelMapping[model] || this.defaultModel;
    }

    getErrorType(error: unknown): AIServiceErrorType {
        if (error instanceof Error) {
            const message = error.message.toLowerCase();
            
            if (message.includes("unauthorized") || message.includes("authentication") || message.includes("401")) {
                return AIServiceErrorType.Authentication;
            }
            
            if (message.includes("rate limit") || message.includes("too many requests") || message.includes("429")) {
                return AIServiceErrorType.RateLimit;
            }
            
            if (message.includes("overload") || message.includes("capacity") || message.includes("503")) {
                return AIServiceErrorType.Overloaded;
            }
            
            if (message.includes("bad request") || message.includes("invalid") || message.includes("400")) {
                return AIServiceErrorType.InvalidRequest;
            }
        }
        
        return AIServiceErrorType.ApiError;
    }

    async safeInputCheck(input: string): Promise<GetOutputTokenLimitResult> {
        // Check for harmful content
        const harmfulCheck = hasHarmfulContent(input);
        if (harmfulCheck.isHarmful) {
            logger.warn("OpenRouter safety check flagged harmful content", { 
                pattern: harmfulCheck.pattern,
                inputLength: input.length, 
            });
            return { cost: 0, isSafe: false };
        }

        // Check for prompt injection attempts
        const injectionCheck = hasPromptInjection(input);
        if (injectionCheck.isInjection) {
            logger.warn("OpenRouter safety check flagged potential prompt injection", { 
                pattern: injectionCheck.pattern,
                inputLength: input.length, 
            });
            return { cost: 0, isSafe: false };
        }

        // Check for excessive length (potential abuse)
        if (input.length > MAX_INPUT_LENGTH) {
            logger.warn("OpenRouter safety check flagged excessive input length", { 
                inputLength: input.length,
                maxLength: MAX_INPUT_LENGTH,
            });
            return { cost: 0, isSafe: false };
        }

        // For production, consider integrating with external moderation services
        return { cost: 0, isSafe: true };
    }

    getNativeToolCapabilities(): Array<Pick<Tool, "name" | "description">> {
        // OpenRouter supports function calling with unified API across providers
        return [
            { name: "function", description: "Execute custom functions with normalized schema across all providers" },
        ];
    }

    /**
     * Checks if the OpenRouter service is healthy and available.
     * @returns true if the service is healthy and can accept requests, false otherwise
     */
    async isHealthy(): Promise<boolean> {
        // Check if API key is configured
        if (!this.apiKey) {
            logger.warn("[OpenRouterService] Health check failed: No API key configured");
            return false;
        }

        try {
            // OpenRouter provides a models endpoint that doesn't cost credits
            // This is a good way to check if the service is available
            const response = await fetch(`${this.baseUrl}/models`, {
                method: "GET",
                headers: {
                    "Authorization": `Bearer ${this.apiKey}`,
                    "HTTP-Referer": this.siteUrl,
                    "X-Title": this.appName,
                },
                // Add a timeout to prevent hanging
                signal: AbortSignal.timeout(5000), // 5 second timeout
            });

            // Check if the response is successful
            if (!response.ok) {
                logger.warn(`[OpenRouterService] Health check failed: HTTP ${response.status}`);
                return false;
            }

            // Verify we can parse the response
            const data = await response.json();
            
            // Check if we have a valid response structure
            const hasModels = data && data.data && Array.isArray(data.data) && data.data.length > 0;
            
            if (!hasModels) {
                logger.warn("[OpenRouterService] Health check: Invalid response or no models available");
                return false;
            }

            return true;
        } catch (error) {
            logger.error(`[OpenRouterService] Health check error: ${error}`);
            return false;
        }
    }
}
