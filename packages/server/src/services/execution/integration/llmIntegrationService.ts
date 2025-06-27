import { type Logger } from "winston";
import {
    type LLMRequest,
    type LLMResponse,
    type AvailableResources,
    type ResourceUsage,
    type ExecutionEvent,
    type ToolExecutionResult,
    nanoid,
} from "@vrooli/shared";
import { AIServiceRegistry } from "../../conversation/registry.js";
import type { ResponseStreamOptions } from "../../conversation/types.js";
import { type EventBus } from "../cross-cutting/events/eventBus.js";

// Constants for configuration
const DEFAULT_MAX_TOKENS = 2000;
const DEFAULT_TEMPERATURE = 0.7;
const DEFAULT_TOOL_TIMEOUT_MS = 30000;
const TOKENS_PER_CHAR_ESTIMATE = 4;
const OUTPUT_TOKENS_ESTIMATE = 1000;
const CONSERVATIVE_COST_ESTIMATE = 0.01;

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
 * 
 * Tool execution is handled through an event-driven architecture where
 * the service publishes tool execution requests and waits for responses
 * from the ToolOrchestrator, enabling emergent capabilities through
 * agent monitoring and intervention.
 */
export class LLMIntegrationService {
    private readonly logger: Logger;
    private readonly registry: AIServiceRegistry;
    private readonly eventBus: EventBus;
    
    // Current execution context for tool calls
    private currentStepId?: string;
    private currentRunId?: string;
    private currentSwarmId?: string;
    private currentConversationId?: string;
    private currentUser?: { id: string; name?: string };

    constructor(logger: Logger, eventBus: EventBus) {
        this.logger = logger;
        this.registry = AIServiceRegistry.get();
        this.eventBus = eventBus;
    }

    /**
     * Configures the service with current execution context.
     * This context is used to enrich tool execution requests with
     * metadata for proper tracking and agent intervention.
     * 
     * @param context - Current execution context
     */
    configureExecutionContext(context: {
        stepId?: string;
        runId?: string;
        swarmId?: string;
        conversationId?: string;
        user?: { id: string; name?: string };
    }): void {
        this.currentStepId = context.stepId;
        this.currentRunId = context.runId;
        this.currentSwarmId = context.swarmId;
        this.currentConversationId = context.conversationId;
        this.currentUser = context.user;
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
            const _serviceId = this.registry.getBestService(request.model);
            const llmService = this.registry.getService(_serviceId);

            if (!llmService) {
                throw new Error(`No available LLM service for model: ${request.model}`);
            }

            // Convert to registry format
            const streamOptions: ResponseStreamOptions = {
                model: request.model,
                input: this.convertToMessageStates(request.messages),
                tools: request.tools ? this.convertToTools(request.tools) : undefined,
                systemMessage: request.systemMessage,
                maxTokens: request.maxTokens || DEFAULT_MAX_TOKENS,
                temperature: request.temperature || DEFAULT_TEMPERATURE,
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
            const toolCalls: ToolCallResult[] = [];
            const confidence = 0.8; // Default confidence

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
                service: _serviceId,
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
            const _serviceId = this.registry.getBestService(request.model);
            
            // Estimate tokens (rough approximation)
            const inputTokens = this.estimateTokens(request.messages, request.systemMessage);
            const outputTokens = request.maxTokens || OUTPUT_TOKENS_ESTIMATE;
            
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
            return CONSERVATIVE_COST_ESTIMATE; // Conservative estimate
        }
    }

    /**
     * Private helper methods
     */
    private convertToMessageStates(messages: Array<{
        role: "system" | "user" | "assistant";
        content: string;
    }>): Array<{ message: { role: string; content: string } }> {
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
    }>): Array<{
        type: string;
        function: {
            name: string;
            description: string;
            parameters: Record<string, unknown>;
        };
    }> {
        return tools.map(tool => ({
            type: "function",
            function: {
                name: tool.name,
                description: tool.description,
                parameters: tool.parameters,
            },
        }));
    }

    /**
     * Executes a tool call through the event-driven architecture.
     * 
     * This method publishes a tool execution request event and waits for the
     * ToolOrchestrator to handle it, enabling emergent capabilities through
     * agent monitoring and intervention. The event-driven pattern allows
     * security agents, optimization agents, and approval systems to intercept
     * and enhance tool execution.
     * 
     * @param toolName - Name of the tool to execute
     * @param args - Tool parameters from the LLM response
     * @param resources - Available resources and tools for this execution
     * @returns Promise resolving to tool execution result
     */
    private async executeToolCall(
        toolName: string,
        args: Record<string, unknown>,
        resources: AvailableResources,
    ): Promise<ToolCallResult> {
        // Generate unique request ID for correlation across events
        const requestId = nanoid();
        
        this.logger.debug("[LLMIntegrationService] Executing tool call via event bus", {
            toolName,
            requestId,
            hasContext: !!this.currentStepId,
        });

        try {
            // Check if tool is available in resources
            const availableTool = resources.tools.find(t => t.name === toolName);
            if (!availableTool) {
                return this.createToolErrorResult(toolName, args, `Tool ${toolName} not available`);
            }

            // Create tool execution request event
            const requestEvent: ExecutionEvent = {
                id: nanoid(),
                timestamp: new Date(),
                type: "tool/execution/requested",
                source: {
                    tier: "integration",
                    component: "LLMIntegrationService",
                    instanceId: "llm-integration-service",
                },
                correlationId: requestId,
                data: {
                    requestId,
                    toolName,
                    parameters: args,
                    context: {
                        stepId: this.currentStepId,
                        runId: this.currentRunId,
                        swarmId: this.currentSwarmId,
                        conversationId: this.currentConversationId,
                        user: this.currentUser,
                    },
                    timeout: this.getToolTimeout(toolName),
                    requiresApproval: this.requiresApproval(toolName),
                    availableTools: resources.tools.map(t => t.name),
                },
            };

            // Publish the tool execution request
            await this.eventBus.publish(requestEvent);

            // Wait for tool execution completion with timeout
            return await this.waitForToolCompletion(requestId, requestEvent.data.timeout);

        } catch (error) {
            this.logger.error("[LLMIntegrationService] Tool execution request failed", {
                toolName,
                requestId,
                error: error instanceof Error ? error.message : String(error),
            });

            return this.createToolErrorResult(
                toolName, 
                args, 
                error instanceof Error ? error.message : String(error),
            );
        }
    }

    /**
     * Waits for tool execution completion by subscribing to completion events.
     * 
     * This enables the event-driven tool execution pattern while maintaining
     * the synchronous interface expected by LLM response processing. The
     * promise-based approach allows the LLM integration to continue processing
     * while tools are executed asynchronously by the ToolOrchestrator.
     * 
     * @param requestId - Unique request ID for correlation
     * @param timeout - Maximum time to wait for completion in milliseconds
     * @returns Promise resolving to tool execution result
     */
    private async waitForToolCompletion(
        requestId: string, 
        timeout = DEFAULT_TOOL_TIMEOUT_MS,
    ): Promise<ToolCallResult> {
        return new Promise((resolve) => {
            let completed = false;
            
            // Clean up function to remove event listeners
            const cleanup = () => {
                this.eventBus.off("tool/execution/completed", completionHandler);
                this.eventBus.off("tool/execution/failed", failureHandler);
            };

            // Set up timeout to prevent hanging
            const timeoutId = setTimeout(() => {
                if (!completed) {
                    completed = true;
                    cleanup();
                    
                    this.logger.warn("[LLMIntegrationService] Tool execution timeout", {
                        requestId,
                        timeout,
                    });
                    
                    resolve(this.createToolErrorResult("unknown", {}, "Tool execution timeout"));
                }
            }, timeout);

            // Handle successful completion
            const completionHandler = (event: ExecutionEvent) => {
                if (event.correlationId === requestId && !completed) {
                    completed = true;
                    clearTimeout(timeoutId);
                    cleanup();
                    
                    const result = event.data.result as ToolExecutionResult;
                    resolve({
                        toolName: result.toolName,
                        input: result.input,
                        output: result.output || {},
                        success: result.success,
                        error: result.error,
                    });
                }
            };

            // Handle execution failure
            const failureHandler = (event: ExecutionEvent) => {
                if (event.correlationId === requestId && !completed) {
                    completed = true;
                    clearTimeout(timeoutId);
                    cleanup();
                    
                    resolve(this.createToolErrorResult(
                        event.data.toolName || "unknown",
                        event.data.parameters || {},
                        event.data.error || "Tool execution failed",
                    ));
                }
            };

            // Subscribe to completion events
            this.eventBus.on("tool/execution/completed", completionHandler);
            this.eventBus.on("tool/execution/failed", failureHandler);
        });
    }

    /**
     * Determines the appropriate timeout for a tool based on its type.
     * Different tools have different expected execution times, and this
     * method provides reasonable defaults that can be enhanced by agents
     * monitoring tool performance.
     * 
     * @param toolName - Name of the tool to get timeout for
     * @returns Timeout in milliseconds
     */
    private getToolTimeout(toolName: string): number {
        // Tool-specific timeouts based on expected execution patterns
        const timeouts: Record<string, number> = {
            "send_message": 10000,      // Fast communication
            "resource_manage": 15000,   // Database operations
            "run_routine": 60000,       // Complex workflows
            "define_tool": 20000,       // Tool generation
            "spawn_swarm": 30000,       // Agent coordination
        };
        return timeouts[toolName] || DEFAULT_TOOL_TIMEOUT_MS; // Default timeout for unknown tools
    }

    /**
     * Determines if a tool requires approval before execution.
     * This supports the approval workflow where certain sensitive
     * tools require human or agent approval before execution.
     * 
     * @param toolName - Name of the tool to check
     * @returns True if approval is required
     */
    private requiresApproval(toolName: string): boolean {
        // Tools that require approval for security/safety reasons
        const approvalRequired = [
            "spawn_swarm",           // Creating new agents
            "resource_manage",       // When performing write operations
            "send_message",          // For external communications
        ];
        return approvalRequired.includes(toolName);
    }

    /**
     * Creates a standardized tool error result.
     * 
     * @param toolName - Name of the tool that failed
     * @param input - Input parameters that were provided
     * @param error - Error message describing the failure
     * @returns Standardized tool error result
     */
    private createToolErrorResult(
        toolName: string, 
        input: Record<string, unknown>, 
        error: string,
    ): ToolCallResult {
        return {
            toolName,
            input,
            output: {},
            success: false,
            error,
        };
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
        return Math.ceil(totalContent.length / TOKENS_PER_CHAR_ESTIMATE);
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
