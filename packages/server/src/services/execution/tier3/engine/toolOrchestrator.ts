import {
    nanoid,
    type ExecutionEvent,
    type ToolExecutionRequest,
    type ToolExecutionResult,
    type ToolResource
} from "@vrooli/shared";
import { type Logger } from "winston";
import { EventTypes, EventUtils, type IEventBus } from "../../../events/index.js";
import { getUnifiedEventSystem } from "../../../events/initialization/eventSystemService.js";
import { type Tool } from "../../../mcp/types.js";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import {
    IntegratedToolRegistry,
    convertToolResourceToTool,
    type IntegratedToolContext,
} from "../../integration/mcp/toolRegistry.js";
import { ErrorHandler, type ComponentErrorHandler } from "../../shared/ErrorHandler.js";
import { type ResourceManager } from "./resourceManager.js";

/**
 * Tool approval status
 */
export interface ToolApprovalStatus {
    toolName: string;
    approved: boolean;
    reason?: string;
    approvedBy?: string;
    approvedAt?: Date;
    expiresAt?: Date;
}


/**
 * ToolOrchestrator - MCP-based tool execution and coordination
 * 
 * This component provides a unified tool execution system built around
 * the Model Context Protocol (MCP) that serves both external AI agents
 * and internal swarms through a centralized tool registry.
 * 
 * Key features:
 * - MCP protocol compliance
 * - Built-in and dynamic tool support
 * - Tool approval and scheduling
 * - Resource tracking per tool
 * - Retry and error handling
 * - Schema compression via define_tool
 * 
 * The orchestrator maintains loose coupling with actual tool implementations
 * while providing consistent execution patterns and resource management.
 */
export class ToolOrchestrator {
    private readonly eventBus: EventBus;
    private readonly logger: Logger;
    private readonly errorHandler: ComponentErrorHandler;
    private readonly unifiedEventBus: IEventBus | null;

    // MCP tool infrastructure
    private readonly toolRegistry: IntegratedToolRegistry;

    // Current execution context
    private currentStepId?: string;
    private currentRunId?: string;
    private currentSwarmId?: string;
    private currentConversationId?: string;
    private currentTools?: ToolResource[];
    private resourceManager?: ResourceManager;
    private currentUser?: any;

    // Event subscription tracking for cleanup
    private toolExecutionSubscriptionId?: string;

    constructor(eventBus: EventBus, logger: Logger, toolRegistry?: IntegratedToolRegistry) {
        this.eventBus = eventBus;
        this.logger = logger;
        this.errorHandler = new ErrorHandler(logger).createComponentHandler("ToolOrchestrator");

        // Get unified event system for modern event publishing
        this.unifiedEventBus = getUnifiedEventSystem();

        // Use provided registry or get default instance
        this.toolRegistry = toolRegistry || IntegratedToolRegistry.getInstance(logger);

        // Initialize event subscriptions for tool execution requests (async)
        this.initializeEventSubscriptions().catch(error => {
            this.logger.error("[ToolOrchestrator] Failed to initialize event subscriptions", {
                error: error instanceof Error ? error.message : String(error),
            });
        });
    }

    /**
     * Cleanup method to remove event subscriptions and prevent memory leaks.
     * Should be called when the ToolOrchestrator is being destroyed.
     */
    async cleanup(): Promise<void> {
        if (this.toolExecutionSubscriptionId) {
            await this.eventBus.unsubscribe(this.toolExecutionSubscriptionId);
        }
        this.logger.info("[ToolOrchestrator] Cleanup completed");
    }

    /**
     * Initializes event subscriptions for tool execution requests.
     * 
     * This method sets up the ToolOrchestrator to respond to tool execution
     * requests from the LLMIntegrationService, enabling the event-driven
     * tool execution pattern that allows agents to monitor and enhance
     * tool calls throughout the execution architecture.
     */
    private async initializeEventSubscriptions(): Promise<void> {
        // Subscribe to tool execution requests from LLMIntegrationService
        this.toolExecutionSubscriptionId = `tool-orchestrator-${nanoid()}`;

        await this.eventBus.subscribe({
            id: this.toolExecutionSubscriptionId,
            pattern: "tool/execution/requested",
            handler: this.handleToolExecutionRequest.bind(this),
        });

        this.logger.info("[ToolOrchestrator] Event subscriptions initialized for tool execution", {
            subscriptionId: this.toolExecutionSubscriptionId,
        });
    }

    /**
     * Configures orchestrator for a specific execution
     */
    configureForExecution(
        stepId: string,
        availableTools: ToolResource[],
        resourceManager: ResourceManager,
        context: {
            runId?: string;
            swarmId?: string;
            conversationId?: string;
            user?: any;
        },
    ): void {
        this.currentStepId = stepId;
        this.currentRunId = context.runId;
        this.currentSwarmId = context.swarmId;
        this.currentConversationId = context.conversationId;
        this.currentTools = availableTools;
        this.resourceManager = resourceManager;
        this.currentUser = context.user;

        // Register dynamic tools from available resources
        for (const toolResource of availableTools) {
            const tool = convertToolResourceToTool(toolResource);
            this.toolRegistry.registerDynamicTool(tool, {
                runId: context.runId,
                swarmId: context.swarmId,
                scope: context.runId ? "run" : context.swarmId ? "swarm" : "global",
            });
        }

        this.logger.debug("[ToolOrchestrator] Configured for execution", {
            stepId,
            toolCount: availableTools.length,
            runId: context.runId,
            swarmId: context.swarmId,
            hasUser: !!context.user,
        });
    }

    /**
     * Executes a tool with MCP protocol
     */
    async executeTool(request: ToolExecutionRequest): Promise<ToolExecutionResult> {
        const startTime = Date.now();
        const { toolName, parameters, timeout, retryPolicy } = request;

        this.logger.info(`[ToolOrchestrator] Executing tool: ${toolName}`, {
            stepId: this.currentStepId,
            hasRetryPolicy: !!retryPolicy,
        });

        const result = await this.errorHandler.wrap(
            async () => {
                // Create integrated tool context
                const context: IntegratedToolContext = {
                    stepId: this.currentStepId!,
                    runId: this.currentRunId,
                    swarmId: this.currentSwarmId,
                    conversationId: this.currentConversationId,
                    user: this.currentUser || { id: "system", languages: ["en"] },
                    logger: this.logger,
                    metadata: {
                        timeout,
                        retryPolicy,
                    },
                };

                // Check resource quota before execution
                if (this.resourceManager) {
                    const quotaCheck = await this.resourceManager.checkQuota(
                        toolName,
                        this.estimateToolCost(toolName),
                    );

                    if (!quotaCheck.allowed) {
                        return this.createErrorResult(
                            `Resource quota exceeded: ${quotaCheck.reason}`,
                            Date.now() - startTime,
                        );
                    }
                }

                // Execute through integrated registry with retry support
                let result: ToolExecutionResult;
                if (retryPolicy) {
                    result = await this.executeWithRetry(request, context);
                } else {
                    result = await this.toolRegistry.executeTool(request, context);
                }

                // Track telemetry
                await this.trackToolUsage(toolName, result, Date.now() - startTime);

                return result;
            },
            "executeTool",
            { toolName, stepId: this.currentStepId },
        );

        if (!result.success) {
            const errorResult = result as { success: false; error: Error };
            return this.createErrorResult(
                errorResult.error.message,
                Date.now() - startTime,
            );
        }

        return result.data;
    }

    /**
     * Lists available tools with MCP format
     */
    async listTools(): Promise<Tool[]> {
        const context: IntegratedToolContext = {
            stepId: this.currentStepId || "list",
            runId: this.currentRunId,
            swarmId: this.currentSwarmId,
            conversationId: this.currentConversationId,
            user: this.currentUser || { id: "system", languages: ["en"] },
            logger: this.logger,
        };

        return await this.toolRegistry.listAvailableTools(context);
    }

    /**
     * Implements define_tool for schema compression
     */
    async defineTool(params: {
        toolName: string;
        variant?: string;
        operation?: string;
    }): Promise<Tool> {
        const { toolName, variant, operation } = params;

        this.logger.debug("[ToolOrchestrator] Defining tool schema", {
            toolName,
            variant,
            operation,
        });

        // Use the integrated registry's defineTool functionality
        const context: IntegratedToolContext = {
            stepId: this.currentStepId || "define",
            runId: this.currentRunId,
            swarmId: this.currentSwarmId,
            conversationId: this.currentConversationId,
            user: this.currentUser || { id: "system", languages: ["en"] },
            logger: this.logger,
        };

        const result = await this.toolRegistry.executeTool({
            toolName: "define_tool",
            parameters: {
                toolName,
                variant,
                op: operation,
            },
        }, context);

        if (result.success && result.output) {
            // Parse the schema from the result
            try {
                const schema = JSON.parse(result.output as string);
                return {
                    name: `${toolName}_${variant}_${operation}`.toLowerCase(),
                    description: schema.description,
                    inputSchema: schema,
                };
            } catch (error) {
                this.logger.error("[ToolOrchestrator] Failed to parse define_tool result", {
                    error,
                    output: result.output,
                });
                throw new Error("Failed to parse tool schema");
            }
        }

        throw new Error(result.error || "Failed to define tool");
    }

    /**
     * Registers a pending tool approval
     */
    async registerPendingApproval(
        toolName: string,
        parameters: Record<string, unknown>,
        timeout = 600000, // 10 minutes default
    ): Promise<string> {
        const context: IntegratedToolContext = {
            stepId: this.currentStepId!,
            runId: this.currentRunId,
            swarmId: this.currentSwarmId,
            conversationId: this.currentConversationId,
            user: this.currentUser || { id: "system", languages: ["en"] },
            logger: this.logger,
        };

        const approvalId = await this.toolRegistry.registerPendingApproval(
            toolName,
            parameters,
            context,
            timeout,
        );

        // Emit approval request event using unified event system
        await this.publishUnifiedEvent(EventTypes.TOOL_APPROVAL_REQUIRED, {
            approvalId,
            toolName,
            parameters,
            stepId: this.currentStepId,
            runId: this.currentRunId,
            timeout,
        }, {
            deliveryGuarantee: "reliable",
            priority: "high",
            tags: ["approval", "tool", "tier3"],
        });

        return approvalId;
    }

    /**
     * Helper method for publishing events using unified event system
     */
    private async publishUnifiedEvent(
        eventType: string,
        data: any,
        options?: {
            deliveryGuarantee?: "fire-and-forget" | "reliable" | "barrier-sync";
            priority?: "low" | "medium" | "high" | "critical";
            tags?: string[];
        },
    ): Promise<void> {
        if (!this.unifiedEventBus) {
            this.logger.debug("[ToolOrchestrator] Unified event bus not available, skipping event publication");
            return;
        }

        try {
            const event = EventUtils.createBaseEvent(
                eventType,
                data,
                EventUtils.createEventSource(3, "ToolOrchestrator", nanoid()),
                EventUtils.createEventMetadata(
                    options?.deliveryGuarantee || "fire-and-forget",
                    options?.priority || "medium",
                    {
                        tags: options?.tags || ["tool", "tier3"],
                        conversationId: this.currentSwarmId,
                    },
                ),
            );

            await this.unifiedEventBus.publish(event);

            this.logger.debug("[ToolOrchestrator] Published unified event", {
                eventType,
                deliveryGuarantee: options?.deliveryGuarantee,
                priority: options?.priority,
            });

        } catch (eventError) {
            this.logger.error("[ToolOrchestrator] Failed to publish unified event", {
                eventType,
                error: eventError instanceof Error ? eventError.message : String(eventError),
            });
        }
    }

    /**
     * Processes tool approval
     */
    async processApproval(
        approvalId: string,
        approved: boolean,
        approvedBy?: string,
    ): Promise<void> {
        this.toolRegistry.processApproval(approvalId, approved, approvedBy);

        // Emit approval processed event using unified event system
        const eventType = approved ? EventTypes.TOOL_APPROVAL_GRANTED : EventTypes.TOOL_APPROVAL_REJECTED;
        await this.publishUnifiedEvent(eventType, {
            approvalId,
            approved,
            approvedBy,
            stepId: this.currentStepId,
            runId: this.currentRunId,
        }, {
            deliveryGuarantee: "reliable",
            priority: "high",
            tags: ["approval", "tool", "tier3"],
        });
    }

    /**
     * Private helper methods
     */
    private async executeWithRetry(
        request: ToolExecutionRequest,
        context: IntegratedToolContext,
    ): Promise<ToolExecutionResult> {
        const { retryPolicy } = request;
        let lastError: Error | undefined;
        let retries = 0;

        const maxRetries = retryPolicy?.maxRetries || 0;
        const backoffStrategy = retryPolicy?.backoffStrategy || "exponential";
        const initialDelay = retryPolicy?.initialDelay || 1000;
        const maxDelay = retryPolicy?.maxDelay || 30000;

        while (retries <= maxRetries) {
            try {
                const result = await this.toolRegistry.executeTool(request, context);

                if (result.success) {
                    return {
                        ...result,
                        retries,
                    };
                }

                // If it's a non-retryable error, return immediately
                if (this.isNonRetryableError(result.error)) {
                    return result;
                }

                throw new Error(result.error);

            } catch (error) {
                lastError = error instanceof Error ? error : new Error(String(error));

                if (retries < maxRetries) {
                    const delay = this.calculateRetryDelay(
                        retries,
                        backoffStrategy,
                        initialDelay,
                        maxDelay,
                    );

                    this.logger.warn("[ToolOrchestrator] Tool execution failed, retrying", {
                        toolName: request.toolName,
                        retry: retries + 1,
                        delay,
                        error: lastError.message,
                    });

                    await this.sleep(delay);
                    retries++;
                } else {
                    break;
                }
            }
        }

        return {
            success: false,
            error: lastError?.message || "Unknown error",
            duration: 0,
            retries,
        };
    }

    private isNonRetryableError(error?: string): boolean {
        if (!error) return false;

        const nonRetryablePatterns = [
            "requires approval",
            "not found",
            "invalid parameters",
            "insufficient permissions",
            "quota exceeded",
        ];

        return nonRetryablePatterns.some(pattern =>
            error.toLowerCase().includes(pattern),
        );
    }

    private estimateToolCost(toolName: string): number {
        // Estimate cost based on tool type
        const costMap: Record<string, number> = {
            define_tool: 0.001,
            resource_manage: 0.01,
            send_message: 0.005,
            run_routine: 0.1,
            spawn_swarm: 0.5,
        };

        return costMap[toolName] || 0.01;
    }

    private calculateRetryDelay(
        retry: number,
        strategy: string,
        initialDelay: number,
        maxDelay: number,
    ): number {
        let delay: number;

        if (strategy === "exponential") {
            delay = initialDelay * Math.pow(2, retry);
        } else {
            delay = initialDelay * (retry + 1);
        }

        return Math.min(delay, maxDelay);
    }

    private async trackToolUsage(
        toolName: string,
        result: ToolExecutionResult,
        duration: number,
    ): Promise<void> {
        // Emit telemetry event using unified event system
        const eventType = result.success ? EventTypes.TOOL_COMPLETED : EventTypes.TOOL_FAILED;
        await this.publishUnifiedEvent(eventType, {
            toolName,
            duration,
            stepId: this.currentStepId || "unknown",
            success: result.success,
            retries: result.retries || 0,
            error: result.success ? undefined : result.error,
        }, {
            deliveryGuarantee: "fire-and-forget",
            priority: "low",
            tags: ["telemetry", "tool", "tier3"],
        });
    }

    private createErrorResult(error: string, duration: number): ToolExecutionResult {
        return {
            success: false,
            error,
            duration,
            retries: 0,
        };
    }

    private sleep(ms: number): Promise<void> {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    /**
     * Handles tool execution requests from the event bus.
     * 
     * This method receives tool execution requests from the LLMIntegrationService,
     * validates them, executes the tool using the existing infrastructure, and
     * publishes the result back to the event bus. This enables the event-driven
     * tool execution pattern while leveraging all existing features like approval
     * workflows, retry policies, and resource management.
     * 
     * @param event - Tool execution request event from LLMIntegrationService
     */
    private async handleToolExecutionRequest(event: ExecutionEvent): Promise<void> {
        // Validate event data structure
        if (!event.data || typeof event.data !== "object") {
            this.logger.error("[ToolOrchestrator] Invalid event data structure", {
                eventId: event.id,
                eventType: event.type,
            });
            return;
        }

        const { requestId, toolName, parameters, context, timeout, requiresApproval } = event.data;

        // Validate required fields
        if (!requestId || !toolName || !parameters) {
            this.logger.error("[ToolOrchestrator] Missing required fields in tool execution request", {
                eventId: event.id,
                hasRequestId: !!requestId,
                hasToolName: !!toolName,
                hasParameters: !!parameters,
            });

            // Publish failure result for invalid request
            if (requestId) {
                await this.publishToolExecutionResult(requestId, {
                    toolName: toolName || "unknown",
                    input: parameters || {},
                    output: {},
                    success: false,
                    error: "Invalid tool execution request: missing required fields",
                    duration: 0,
                }, "tool/execution/failed");
            }
            return;
        }

        this.logger.debug("[ToolOrchestrator] Handling tool execution request", {
            requestId,
            toolName,
            requiresApproval,
            hasContext: !!context?.stepId,
        });

        try {
            // Configure execution context if provided
            if (context && context.stepId) {
                // Only configure if we have a resource manager
                if (this.resourceManager) {
                    this.configureForExecution(
                        context.stepId,
                        this.currentTools || [],
                        this.resourceManager,
                        {
                            runId: context.runId,
                            swarmId: context.swarmId,
                            conversationId: context.conversationId,
                            user: context.user,
                        },
                    );
                } else {
                    this.logger.warn("[ToolOrchestrator] No resource manager available for context configuration", {
                        requestId,
                        stepId: context.stepId,
                    });
                    // Set minimal context
                    this.currentStepId = context.stepId;
                    this.currentRunId = context.runId;
                    this.currentSwarmId = context.swarmId;
                    this.currentConversationId = context.conversationId;
                    this.currentUser = context.user;
                }
            }

            // Handle approval workflow if required
            if (requiresApproval) {
                const approved = await this.handleToolApproval(
                    toolName,
                    parameters,
                    timeout || 600000,
                );

                if (!approved) {
                    await this.publishToolExecutionResult(requestId, {
                        toolName,
                        input: parameters,
                        output: {},
                        success: false,
                        error: "Tool execution not approved",
                        duration: 0,
                    }, "tool/execution/failed");
                    return;
                }
            }

            // Create tool execution request
            const toolRequest: ToolExecutionRequest = {
                toolName,
                parameters,
                retryPolicy: {
                    maxRetries: 2,
                    backoffStrategy: "exponential",
                    initialDelay: 1000,
                    maxDelay: 5000,
                },
            };

            // Execute the tool using existing infrastructure
            const result = await this.executeTool(toolRequest);

            // Publish success result
            await this.publishToolExecutionResult(
                requestId,
                result,
                "tool/execution/completed",
            );

        } catch (error) {
            this.logger.error("[ToolOrchestrator] Tool execution failed", {
                requestId,
                toolName,
                error: error instanceof Error ? error.message : String(error),
            });

            // Publish failure result
            await this.publishToolExecutionResult(requestId, {
                toolName,
                input: parameters,
                output: {},
                success: false,
                error: error instanceof Error ? error.message : String(error),
                duration: 0,
            }, "tool/execution/failed");
        }
    }

    /**
     * Handles tool approval workflow for event-driven execution.
     * 
     * This method integrates with the existing approval system to handle
     * tools that require approval before execution. It registers the approval
     * request and waits for the approval response through events.
     * 
     * @param toolName - Name of the tool requiring approval
     * @param parameters - Tool parameters for approval review
     * @param timeout - Maximum time to wait for approval
     * @returns Promise resolving to approval status
     */
    private async handleToolApproval(
        toolName: string,
        parameters: Record<string, unknown>,
        timeout: number,
    ): Promise<boolean> {
        try {
            // Register pending approval using existing infrastructure
            const approvalId = await this.registerPendingApproval(
                toolName,
                parameters,
                timeout,
            );

            // Wait for approval through events
            return await this.waitForApproval(approvalId, timeout);

        } catch (error) {
            this.logger.error("[ToolOrchestrator] Approval handling failed", {
                toolName,
                error: error instanceof Error ? error.message : String(error),
            });
            return false;
        }
    }

    /**
     * Publishes tool execution results to the event bus.
     * 
     * This method creates and publishes the appropriate event (completion or failure)
     * with the tool execution result, enabling the LLMIntegrationService to receive
     * the response and continue processing. The event includes correlation data to
     * match requests with responses.
     * 
     * @param requestId - Original request ID for correlation
     * @param result - Tool execution result from ToolOrchestrator
     * @param eventType - Type of result event to publish
     */
    private async publishToolExecutionResult(
        requestId: string,
        result: ToolExecutionResult,
        eventType: "tool/execution/completed" | "tool/execution/failed",
    ): Promise<void> {
        const resultEvent: ExecutionEvent = {
            id: nanoid(),
            timestamp: new Date(),
            type: eventType,
            source: {
                tier: "tier3",
                component: "ToolOrchestrator",
                instanceId: "tool-orchestrator",
            },
            correlationId: requestId,
            data: {
                requestId,
                result,
                toolName: result.toolName,
                success: result.success,
                error: result.error,
                duration: result.duration,
                metadata: {
                    stepId: this.currentStepId,
                    runId: this.currentRunId,
                    swarmId: this.currentSwarmId,
                },
            },
        };

        await this.eventBus.publish(resultEvent);

        this.logger.debug("[ToolOrchestrator] Published tool execution result", {
            requestId,
            eventType,
            toolName: result.toolName,
            success: result.success,
        });
    }

    /**
     * Waits for tool approval with timeout using event subscription.
     * 
     * This method subscribes to approval events and waits for the specific
     * approval response, with a timeout to prevent hanging. It integrates
     * with the existing approval workflow events.
     * 
     * @param approvalId - Approval request ID to wait for
     * @param timeout - Timeout in milliseconds
     * @returns Promise resolving to approval status
     */
    private async waitForApproval(approvalId: string, timeout: number): Promise<boolean> {
        return new Promise((resolve) => {
            let resolved = false;

            // Set up timeout
            const timeoutId = setTimeout(() => {
                if (!resolved) {
                    resolved = true;
                    this.eventBus.off("tool.approval_processed", approvalHandler);
                    this.logger.warn("[ToolOrchestrator] Tool approval timeout", {
                        approvalId,
                        timeout,
                    });
                    resolve(false);
                }
            }, timeout);

            // Handle approval response
            const approvalHandler = (event: ExecutionEvent) => {
                if (event.data.approvalId === approvalId && !resolved) {
                    resolved = true;
                    clearTimeout(timeoutId);
                    this.eventBus.off("tool.approval_processed", approvalHandler);

                    const approved = event.data.approved === true;
                    this.logger.debug("[ToolOrchestrator] Tool approval received", {
                        approvalId,
                        approved,
                        approvedBy: event.data.approvedBy,
                    });

                    resolve(approved);
                }
            };

            // Subscribe to approval events
            this.eventBus.on("tool.approval_processed", approvalHandler);
        });
    }
}
