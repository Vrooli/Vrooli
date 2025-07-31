import { API_CREDITS_MULTIPLIER, LlmServiceId, type MessageState, type ThirdPartyModelInfo, type Tool } from "@vrooli/shared";
import type { ChatCompletionMessageParam } from "openai/resources/chat/completions.js";
import { logger } from "../../../events/logger.js";
import type { ResourceRegistry } from "../../resources/ResourceRegistry.js";
import { DiscoveryStatus, ResourceHealth } from "../../resources/types.js";
import { generateContextFromMessages } from "../contextGeneration.js";
import { hasHarmfulContent, hasPromptInjection } from "../messageValidation.js";
import { AIServiceErrorType } from "../registry.js";
import { AIService, type GetOutputTokenLimitParams, type GetOutputTokenLimitResult, type GetResponseCostParams, type ResponseStreamOptions, type ServiceStreamEvent } from "../services.js";
import { withStreamTimeout } from "../streamTimeout.js";
import { TokenEstimatorType, type EstimateTokensResult } from "../tokenTypes.js";

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
const TIMEOUT_MINUTES = 5;
const SECONDS_PER_MINUTE = 60;
const MS_PER_SECOND = 1000;
// Model costs are in cents per million tokens
// API credits = tokens * cost_per_million_tokens_in_cents
const FALLBACK_INPUT_COST_CENTS_PER_MILLION = 50;  // $0.50 per 1M tokens
const FALLBACK_OUTPUT_COST_CENTS_PER_MILLION = 150; // $1.50 per 1M tokens
const MAX_INPUT_LENGTH = 50000; // 50KB limit for Cloudflare Gateway

/**
 * Options for CloudflareGatewayService configuration
 */
export interface CloudflareGatewayServiceOptions {
    apiToken?: string;
    gatewayUrl?: string;
    accountId?: string;
    defaultModel?: string;
    resourceRegistry?: ResourceRegistry; // Optional dependency injection
}

/**
 * CloudflareGatewayService - Service for communicating with Cloudflare AI Gateway
 */
export class CloudflareGatewayService extends AIService<string, ThirdPartyModelInfo> {
    private readonly apiToken: string;
    private readonly gatewayUrl: string;
    private readonly accountId: string;
    private resourceRegistry?: ResourceRegistry;

    __id: LlmServiceId = LlmServiceId.CloudflareGateway;
    featureFlags = { supportsStatefulConversations: false };
    defaultModel = "@cf/openai/gpt-4o-mini";

    constructor(options?: CloudflareGatewayServiceOptions) {
        super();
        this.apiToken = options?.apiToken ?? process.env.CLOUDFLARE_GATEWAY_TOKEN ?? "";
        this.gatewayUrl = options?.gatewayUrl ?? process.env.CLOUDFLARE_GATEWAY_URL ??
            "https://gateway.ai.cloudflare.com/v1";
        this.accountId = options?.accountId ?? process.env.CLOUDFLARE_ACCOUNT_ID ?? "";
        this.defaultModel = options?.defaultModel ?? "@cf/openai/gpt-4o-mini";
        this.resourceRegistry = options?.resourceRegistry;

        // Validate required configuration
        if (!this.apiToken) {
            logger.warn("[CloudflareGatewayService] No API token configured - service will not function");
        }
        if (!this.accountId) {
            logger.warn("[CloudflareGatewayService] No account ID configured - service will not function");
        }

        // Log ResourceRegistry integration status
        logger.info("[CloudflareGatewayService] Initialized", {
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
        logger.info("[CloudflareGatewayService] ResourceRegistry updated - enhanced health checking now available");
    }

    /**
     * Check if we support a specific model
     */
    async supportsModel(model: string): Promise<boolean> {
        // Support all Cloudflare Gateway models (models starting with @cf/)
        return model.startsWith("@cf/");
    }

    generateContext(messages: MessageState[], systemMessage?: string): ChatCompletionMessageParam[] {
        return generateContextFromMessages(messages, systemMessage, "CloudflareGateway");
    }

    async *generateResponseStreaming(options: ResponseStreamOptions): AsyncGenerator<ServiceStreamEvent> {
        // Create inner generator for timeout wrapping
        const innerGenerator = this.generateResponseStreamingInternal(options);

        // Wrap with timeout protection
        yield* withStreamTimeout(innerGenerator, {
            serviceName: "CloudflareGateway",
            modelName: options.model,
            signal: options.signal,
            timeoutMs: TIMEOUT_MINUTES * SECONDS_PER_MINUTE * MS_PER_SECOND, // 5 minute timeout
        });
    }

    private async *generateResponseStreamingInternal(options: ResponseStreamOptions): AsyncGenerator<ServiceStreamEvent> {
        if (!(await this.supportsModel(options.model))) {
            throw new Error(`Model ${options.model} not supported by Cloudflare Gateway`);
        }

        const url = `${this.gatewayUrl}/${this.accountId}/ai/run/${options.model}`;
        const messages = this.generateContext(options.input, options.systemMessage);

        const requestBody: any = {
            messages,
            stream: true,
            max_tokens: options.maxTokens || DEFAULT_MAX_TOKENS,
            temperature: 0.7,
        };

        // Add tools if provided (Cloudflare Gateway uses OpenAI-compatible format)
        if (options.tools && options.tools.length > 0) {
            requestBody.tools = options.tools;
        }

        const response = await fetch(url, {
            method: "POST",
            headers: {
                "Authorization": `Bearer ${this.apiToken}`,
                "Content-Type": "application/json",
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
                            // Try to extract actual cost from AI Gateway headers, fallback to 0
                            const headerCost = this.extractAIGatewayCost(response.headers);
                            const cost = headerCost !== null ? headerCost : 0;
                            yield { type: "done", cost };
                            return;
                        }

                        const data = JSON.parse(dataContent);

                        if (data.choices?.[0]?.delta?.content) {
                            yield { type: "text", content: data.choices[0].delta.content };
                        }

                        // Handle tool calls (Cloudflare Gateway uses OpenAI-compatible format)
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
                            const usage = data.usage || { prompt_tokens: 0, completion_tokens: 0 };

                            // First try to get actual cost from AI Gateway headers
                            const headerCost = this.extractAIGatewayCost(response.headers);
                            const cost = headerCost !== null ? headerCost :
                                this.calculateCloudflareGatewayCost(options.model, usage.prompt_tokens, usage.completion_tokens);

                            yield { type: "done", cost };
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

    private calculateCloudflareGatewayCost(model: string, inputTokens: number, outputTokens: number): number {
        // Calculate cost in API credits
        // Model costs are in cents per million tokens
        // Credits = tokens * cost_per_million_tokens_in_cents
        const modelInfo = this.getModelInfo()[model];
        if (modelInfo && modelInfo.inputCost && modelInfo.outputCost) {
            // inputCost and outputCost are in cents per million tokens
            const inputCredits = inputTokens * modelInfo.inputCost;
            const outputCredits = outputTokens * modelInfo.outputCost;
            return inputCredits + outputCredits;
        }

        // Fallback pricing using default costs
        const fallbackInputCredits = inputTokens * FALLBACK_INPUT_COST_CENTS_PER_MILLION;
        const fallbackOutputCredits = outputTokens * FALLBACK_OUTPUT_COST_CENTS_PER_MILLION;
        return fallbackInputCredits + fallbackOutputCredits;
    }

    /**
     * Extract cost information from AI Gateway response headers
     * AI Gateway may provide actual cost data in response headers
     * 
     * Note: For historical cost data and analytics, you can also query the AI Gateway
     * analytics API using GraphQL at https://api.cloudflare.com/client/v4/graphql
     * with aiGatewayRequestsAdaptiveGroups to get aggregated cost metrics.
     */
    private extractAIGatewayCost(headers: Headers): number | null {
        // Check for AI Gateway cost headers
        // cf-aig-log-id indicates the request went through AI Gateway
        const logId = headers.get("cf-aig-log-id");
        if (!logId) {
            return null; // Not an AI Gateway response
        }

        // Look for cost-related headers that AI Gateway might provide
        // Note: These headers may not be available in all AI Gateway responses
        const costHeader = headers.get("cf-aig-cost") ||
            headers.get("x-cost") ||
            headers.get("cost");

        if (costHeader) {
            const costInDollars = parseFloat(costHeader);
            if (!isNaN(costInDollars)) {
                // Convert from dollars to credits
                // $1 = 100 cents = 100 * 1,000,000 credits
                const costInCents = costInDollars * 100;
                return Math.floor(costInCents * Number(API_CREDITS_MULTIPLIER));
            }
        }

        // Check for cache status - cached responses should have zero cost
        const cacheStatus = headers.get("cf-aig-cache-status");
        if (cacheStatus === "HIT") {
            return 0; // Cached responses have no cost
        }

        return null; // No cost information available in headers
    }

    getContextSize(requestedModel?: string | null): number {
        const model = this.getModel(requestedModel);
        const modelInfo = this.getModelInfo()[model];
        return modelInfo?.contextWindow || DEFAULT_CONTEXT_WINDOW;
    }

    getModelInfo(): Record<string, ThirdPartyModelInfo> {
        // Return simplified model info for common Cloudflare Gateway models
        // IMPORTANT: inputCost and outputCost are in cents per million tokens
        return {
            "@cf/openai/gpt-4o": {
                enabled: true,
                name: "GPT-4o",
                descriptionShort: "GPT-4o via Cloudflare Gateway",
                inputCost: 250,    // $2.50 per 1M input tokens
                outputCost: 1000,  // $10.00 per 1M output tokens
                contextWindow: 128000,
                maxOutputTokens: 4096,
                features: {},
                supportsReasoning: false,
            },
            "@cf/openai/gpt-4o-mini": {
                enabled: true,
                name: "GPT-4o Mini",
                descriptionShort: "GPT-4o Mini via Cloudflare Gateway",
                inputCost: 15,     // $0.15 per 1M input tokens
                outputCost: 60,    // $0.60 per 1M output tokens
                contextWindow: 128000,
                maxOutputTokens: 16384,
                features: {},
                supportsReasoning: false,
            },
            "@cf/anthropic/claude-3-5-sonnet": {
                enabled: true,
                name: "Claude 3.5 Sonnet",
                descriptionShort: "Claude 3.5 Sonnet via Cloudflare Gateway",
                inputCost: 300,    // $3.00 per 1M input tokens
                outputCost: 1500,  // $15.00 per 1M output tokens
                contextWindow: 200000,
                maxOutputTokens: 8192,
                features: {},
                supportsReasoning: false,
            },
            "@cf/anthropic/claude-3-haiku": {
                enabled: true,
                name: "Claude 3 Haiku",
                descriptionShort: "Claude 3 Haiku via Cloudflare Gateway",
                inputCost: 25,     // $0.25 per 1M input tokens
                outputCost: 125,   // $1.25 per 1M output tokens
                contextWindow: 200000,
                maxOutputTokens: 4096,
                features: {},
                supportsReasoning: false,
            },
            "@cf/meta/llama-3-8b-instruct": {
                enabled: true,
                name: "Llama 3 8B",
                descriptionShort: "Llama 3 8B via Cloudflare Gateway",
                inputCost: 10,     // $0.10 per 1M tokens
                outputCost: 10,    // $0.10 per 1M tokens
                contextWindow: 8192,
                maxOutputTokens: 4096,
                features: {},
                supportsReasoning: false,
            },
            "@cf/meta/llama-3-70b-instruct": {
                enabled: true,
                name: "Llama 3 70B",
                descriptionShort: "Llama 3 70B via Cloudflare Gateway",
                inputCost: 50,     // $0.50 per 1M tokens
                outputCost: 50,    // $0.50 per 1M tokens
                contextWindow: 8192,
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

        // First try to use AI Gateway's actual cost if available
        if (params.usage && "actualCost" in params.usage && typeof params.usage.actualCost === "number") {
            // actualCost is in dollars, convert to credits
            const costInCents = params.usage.actualCost * 100;
            return Math.floor(costInCents * Number(API_CREDITS_MULTIPLIER));
        }

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

        // If it's a Cloudflare Gateway model, return as is
        if (model.startsWith("@cf/")) {
            return model;
        }

        // Try to map common model names to Cloudflare Gateway equivalents
        const modelMapping: Record<string, string> = {
            "gpt-4o": "@cf/openai/gpt-4o",
            "gpt-4o-mini": "@cf/openai/gpt-4o-mini",
            "gpt-4-turbo": "@cf/openai/gpt-4-turbo",
            "gpt-3.5-turbo": "@cf/openai/gpt-3.5-turbo",
            "claude-3-5-sonnet": "@cf/anthropic/claude-3-5-sonnet",
            "claude-3-haiku": "@cf/anthropic/claude-3-haiku",
            "claude-3-sonnet": "@cf/anthropic/claude-3-sonnet",
            "llama-3-8b": "@cf/meta/llama-3-8b-instruct",
            "llama-3-70b": "@cf/meta/llama-3-70b-instruct",
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
            logger.warn("Cloudflare Gateway safety check flagged harmful content", {
                pattern: harmfulCheck.pattern,
                inputLength: input.length,
            });
            return { cost: 0, isSafe: false };
        }

        // Check for prompt injection attempts
        const injectionCheck = hasPromptInjection(input);
        if (injectionCheck.isInjection) {
            logger.warn("Cloudflare Gateway safety check flagged potential prompt injection", {
                pattern: injectionCheck.pattern,
                inputLength: input.length,
            });
            return { cost: 0, isSafe: false };
        }

        // Check for excessive length (potential abuse)
        if (input.length > MAX_INPUT_LENGTH) {
            logger.warn("Cloudflare Gateway safety check flagged excessive input length", {
                inputLength: input.length,
                maxLength: MAX_INPUT_LENGTH,
            });
            return { cost: 0, isSafe: false };
        }

        // For production, consider integrating with Cloudflare's AI safety features
        // or other third-party moderation services
        return { cost: 0, isSafe: true };
    }

    getNativeToolCapabilities(): Array<Pick<Tool, "name" | "description">> {
        // Cloudflare Gateway supports function calling with embedded execution
        return [
            { name: "function", description: "Execute custom functions with embedded execution in Workers AI" },
        ];
    }

    /**
     * Checks if the CloudflareGateway service is healthy and available.
     * Enhanced with optional ResourceRegistry integration for better monitoring.
     * @returns true if the service is healthy and can accept requests, false otherwise
     */
    async isHealthy(): Promise<boolean> {
        // Check if required configuration is present
        const hasRequiredConfig = Boolean(this.apiToken && this.accountId && this.gatewayUrl);

        if (!hasRequiredConfig) {
            logger.warn("[CloudflareGatewayService] Health check failed: Missing required configuration");
            return false;
        }

        // Enhanced check with ResourceRegistry if available
        if (this.resourceRegistry) {
            try {
                // Try to get CloudflareGateway resource from registry
                const resourceId = "cloudflare-gateway";
                const resource = this.resourceRegistry.getResource(resourceId);

                if (resource) {
                    const resourceInfo = resource.getPublicInfo();
                    const isResourceHealthy = resourceInfo.health === ResourceHealth.Healthy &&
                        resourceInfo.status === DiscoveryStatus.Available;

                    logger.debug("[CloudflareGatewayService] ResourceRegistry health check", {
                        health: resourceInfo.health,
                        status: resourceInfo.status,
                        isHealthy: isResourceHealthy,
                    });

                    if (isResourceHealthy) {
                        logger.debug("[CloudflareGatewayService] Resource system reports CloudflareGateway as healthy");
                        return true;
                    } else {
                        logger.warn(`[CloudflareGatewayService] Resource system reports CloudflareGateway as unhealthy: ${resourceInfo.health}/${resourceInfo.status}`);
                        return false;
                    }
                }

                logger.debug("[CloudflareGatewayService] No CloudflareGateway resource found in registry, falling back to basic check");
            } catch (error) {
                logger.warn("[CloudflareGatewayService] ResourceRegistry health check failed, falling back to basic check", error);
            }
        }

        // Basic health check - configuration validation only
        // For Cloudflare Gateway, we consider it healthy if the configuration is valid
        // since the actual API endpoint validation would require making requests with credits
        // The service will fail gracefully on actual requests if the endpoint is down
        return true;
    }
}
