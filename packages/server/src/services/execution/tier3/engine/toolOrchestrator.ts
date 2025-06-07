import { type Logger } from "winston";
import { type EventBus } from "../../cross-cutting/eventBus.js";
import { type ResourceManager } from "./resourceManager.js";
import {
    type ToolResource,
    type ToolExecutionRequest,
    type ToolExecutionResult,
    type RetryPolicy,
} from "@vrooli/shared";
import { 
    IntegratedToolRegistry, 
    convertToolResourceToTool,
    type IntegratedToolContext,
} from "../../integration/mcp/toolRegistry.js";
import { type Tool } from "../../../mcp/types.js";

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

    constructor(eventBus: EventBus, logger: Logger, toolRegistry?: IntegratedToolRegistry) {
        this.eventBus = eventBus;
        this.logger = logger;
        
        // Use provided registry or get default instance
        this.toolRegistry = toolRegistry || IntegratedToolRegistry.getInstance(logger);
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

        try {
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

        } catch (error) {
            this.logger.error(`[ToolOrchestrator] Tool execution failed: ${toolName}`, {
                error: error instanceof Error ? error.message : String(error),
                stepId: this.currentStepId,
            });

            return this.createErrorResult(
                error instanceof Error ? error.message : "Unknown error",
                Date.now() - startTime,
            );
        }
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

        // Emit approval request event
        await this.eventBus.publish("tool.approval_required", {
            approvalId,
            toolName,
            parameters,
            stepId: this.currentStepId,
            runId: this.currentRunId,
            timeout,
        });

        return approvalId;
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

        // Emit approval processed event
        await this.eventBus.publish("tool.approval_processed", {
            approvalId,
            approved,
            approvedBy,
            stepId: this.currentStepId,
            runId: this.currentRunId,
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
            error.toLowerCase().includes(pattern)
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
        // Emit telemetry event
        await this.eventBus.publish("telemetry.perf", {
            type: "perf.tool_call",
            timestamp: new Date(),
            stepId: this.currentStepId,
            metadata: {
                toolName,
                duration,
                success: result.success,
                retries: result.retries,
            },
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
}
