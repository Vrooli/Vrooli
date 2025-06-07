import { type Logger } from "winston";
import {
    type LLMRequest,
    type LLMResponse,
    type AvailableResources,
    type ResourceUsage,
} from "@vrooli/shared";
import { AIServiceRegistry } from "../../conversation/registry.js";
import type { ResponseStreamOptions } from "../../conversation/types.js";

/**
 * Tool call result for LLM integration
 */
export interface ToolCallResult {
    toolName: string;
    input: Record<string, unknown>;
    output: Record<string, unknown>;
    success: boolean;
    error?: string;
}

/**
 * LLM Integration Service
 * 
 * This service bridges the execution architecture with Vrooli's existing
 * LLM infrastructure, providing a clean interface for AI interactions
 * within the three-tier execution system.
 */
export class LLMIntegrationService {
    private readonly logger: Logger;
    private readonly registry: AIServiceRegistry;

    constructor(logger: Logger) {
        this.logger = logger;
        this.registry = AIServiceRegistry.get();
    }

    /**
     * Executes an LLM request using the best available service
     */
    async executeRequest(
        request: LLMRequest,
        resources: AvailableResources,
        userData?: { id: string; name?: string },
    ): Promise<LLMResponse & { resourceUsage: ResourceUsage }> {
        this.logger.debug("[LLMIntegrationService] Executing LLM request", {
            model: request.model,
            messageCount: request.messages.length,
            hasTools: !!request.tools?.length,
        });

        try {
            // Find the best service for the requested model
            const serviceId = this.registry.getBestService(request.model);
            const llmService = this.registry.getService(serviceId);

            if (!llmService) {
                throw new Error(`No available LLM service for model: ${request.model}`);
            }

            // Convert to registry format
            const streamOptions: ResponseStreamOptions = {
                model: request.model,
                input: this.convertToMessageStates(request.messages),
                tools: request.tools ? this.convertToTools(request.tools) : undefined,
                systemMessage: request.systemMessage,
                maxTokens: request.maxTokens || 2000,
                temperature: request.temperature || 0.7,
                userData: userData ? {
                    id: userData.id,
                    name: userData.name,
                } : undefined,
                // Add abort signal if needed
                signal: undefined,
            };

            // Track resource usage
            const resourceUsage: ResourceUsage = {
                tokens: 0,
                apiCalls: 1,
                computeTime: 0,
                cost: 0,
            };

            const startTime = Date.now();
            let responseContent = "";
            let reasoning = "";
            let toolCalls: ToolCallResult[] = [];
            let confidence = 0.8; // Default confidence

            // Stream the response
            for await (const event of llmService.generateResponseStreaming(streamOptions)) {
                switch (event.type) {
                    case "text":
                        responseContent += event.text;
                        break;

                    case "reasoning":
                        reasoning += event.reasoning;
                        break;

                    case "function_call":
                        // Handle tool calls
                        if (event.function) {
                            const toolResult = await this.executeToolCall(
                                event.function.name,
                                event.function.arguments,
                                resources,
                            );
                            toolCalls.push(toolResult);
                        }
                        break;

                    case "done":
                        // Final metrics
                        resourceUsage.tokens = event.usage?.total_tokens || 0;
                        resourceUsage.cost = event.cost || 0;
                        break;

                    case "error":
                        throw new Error(`LLM service error: ${event.error}`);
                }
            }

            resourceUsage.computeTime = Date.now() - startTime;

            // Build response
            const response: LLMResponse = {
                content: responseContent,
                reasoning: reasoning || undefined,
                confidence,
                tokensUsed: resourceUsage.tokens,
                toolCalls: toolCalls.length > 0 ? toolCalls : undefined,
                model: request.model,
                finishReason: "stop", // Would need to track actual finish reason
            };

            this.logger.info("[LLMIntegrationService] LLM request completed", {
                model: request.model,
                service: serviceId,
                tokensUsed: resourceUsage.tokens,
                cost: resourceUsage.cost,
                duration: resourceUsage.computeTime,
                toolCallCount: toolCalls.length,
            });

            return { ...response, resourceUsage };

        } catch (error) {
            this.logger.error("[LLMIntegrationService] LLM request failed", {
                model: request.model,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Checks if a model is available
     */
    async isModelAvailable(modelName: string): Promise<boolean> {
        try {
            const serviceId = this.registry.getBestService(modelName);
            const service = this.registry.getService(serviceId);
            return !!service;
        } catch (error) {
            this.logger.debug("[LLMIntegrationService] Model availability check failed", {
                model: modelName,
                error: error instanceof Error ? error.message : String(error),
            });
            return false;
        }
    }

    /**
     * Gets available models from the registry
     */
    getAvailableModels(): Array<{
        model: string;
        provider: string;
        available: boolean;
        pricing?: {
            input: number;
            output: number;
        };
    }> {
        try {
            const services = this.registry.getAllServices();
            const models: Array<{
                model: string;
                provider: string;
                available: boolean;
                pricing?: { input: number; output: number };
            }> = [];

            for (const [serviceId, service] of services) {
                // Get models from service (would need to implement in registry)
                // For now, return some common models
                const commonModels = this.getCommonModelsForService(serviceId);
                for (const model of commonModels) {
                    models.push({
                        model: model.name,
                        provider: serviceId,
                        available: true,
                        pricing: model.pricing,
                    });
                }
            }

            return models;
        } catch (error) {
            this.logger.error("[LLMIntegrationService] Failed to get available models", {
                error: error instanceof Error ? error.message : String(error),
            });
            return [];
        }
    }

    /**
     * Estimates cost for a request
     */
    estimateCost(request: LLMRequest): number {
        try {
            // Get service for model
            const serviceId = this.registry.getBestService(request.model);
            
            // Estimate tokens (rough approximation)
            const inputTokens = this.estimateTokens(request.messages, request.systemMessage);
            const outputTokens = request.maxTokens || 1000;
            
            // Get pricing (would need to be implemented in registry)
            const pricing = this.getPricingForModel(request.model);
            
            const inputCost = (inputTokens / 1000) * pricing.input;
            const outputCost = (outputTokens / 1000) * pricing.output;
            
            return inputCost + outputCost;
        } catch (error) {
            this.logger.warn("[LLMIntegrationService] Cost estimation failed", {
                model: request.model,
                error: error instanceof Error ? error.message : String(error),
            });
            return 0.01; // Conservative estimate
        }
    }

    /**
     * Private helper methods
     */
    private convertToMessageStates(messages: Array<{
        role: "system" | "user" | "assistant";
        content: string;
    }>): any[] {
        return messages.map(msg => ({
            message: {
                role: msg.role,
                content: msg.content,
            },
            // Add other required fields for MessageState
        }));
    }

    private convertToTools(tools: Array<{
        name: string;
        description: string;
        parameters: Record<string, unknown>;
    }>): any[] {
        return tools.map(tool => ({
            type: "function",
            function: {
                name: tool.name,
                description: tool.description,
                parameters: tool.parameters,
            },
        }));
    }

    private async executeToolCall(
        toolName: string,
        args: Record<string, unknown>,
        resources: AvailableResources,
    ): Promise<ToolCallResult> {
        this.logger.debug("[LLMIntegrationService] Executing tool call", {
            toolName,
            args,
        });

        try {
            // Check if tool is available
            const availableTool = resources.tools.find(t => t.name === toolName);
            if (!availableTool) {
                return {
                    toolName,
                    input: args,
                    output: {},
                    success: false,
                    error: `Tool ${toolName} not available`,
                };
            }

            // Note: Tool execution should be handled by the ToolOrchestrator in Tier 3
            // The LLMIntegrationService should not execute tools directly
            // Instead, it should return tool calls to be executed by the strategy
            this.logger.warn("[LLMIntegrationService] Direct tool execution requested - should be handled by ToolOrchestrator", {
                toolName,
            });

            // For now, return a placeholder indicating the tool should be executed
            return {
                toolName,
                input: args,
                output: {
                    pending: true,
                    message: "Tool execution should be handled by ToolOrchestrator in Tier 3",
                },
                success: true,
            };

        } catch (error) {
            this.logger.error("[LLMIntegrationService] Tool execution failed", {
                toolName,
                error: error instanceof Error ? error.message : String(error),
            });

            return {
                toolName,
                input: args,
                output: {},
                success: false,
                error: error instanceof Error ? error.message : String(error),
            };
        }
    }

    private estimateTokens(
        messages: Array<{ role: string; content: string }>,
        systemMessage?: string,
    ): number {
        let totalContent = systemMessage || "";
        for (const msg of messages) {
            totalContent += msg.content;
        }
        
        // Rough approximation: 1 token ≈ 4 characters for English
        return Math.ceil(totalContent.length / 4);
    }

    private getPricingForModel(model: string): { input: number; output: number } {
        // Default pricing in dollars per 1K tokens
        const pricing: Record<string, { input: number; output: number }> = {
            "gpt-4o": { input: 0.0025, output: 0.01 },
            "gpt-4o-mini": { input: 0.00015, output: 0.0006 },
            "gpt-4-turbo": { input: 0.01, output: 0.03 },
            "gpt-4": { input: 0.03, output: 0.06 },
            "o1-mini": { input: 0.003, output: 0.012 },
            "o1-preview": { input: 0.015, output: 0.06 },
        };

        return pricing[model] || { input: 0.001, output: 0.002 };
    }

    private getCommonModelsForService(serviceId: string): Array<{
        name: string;
        pricing: { input: number; output: number };
    }> {
        if (serviceId.includes("openai")) {
            return [
                { name: "gpt-4o", pricing: { input: 0.0025, output: 0.01 } },
                { name: "gpt-4o-mini", pricing: { input: 0.00015, output: 0.0006 } },
                { name: "gpt-4-turbo", pricing: { input: 0.01, output: 0.03 } },
                { name: "o1-mini", pricing: { input: 0.003, output: 0.012 } },
            ];
        }
        
        return [];
    }
}