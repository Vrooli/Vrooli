import { type Logger } from "winston";
import { EventBus } from "../../cross-cutting/eventBus.js";
import { type ResourceManager } from "./resourceManager.js";
import {
    type ToolResource,
    type ToolExecutionRequest,
    type ToolExecutionResult,
    type RetryPolicy,
} from "@vrooli/shared";

/**
 * MCP tool definition
 */
export interface MCPTool {
    name: string;
    description: string;
    inputSchema: {
        type: "object";
        properties: Record<string, unknown>;
        required?: string[];
    };
    handler?: (args: Record<string, unknown>) => Promise<unknown>;
}

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
 * Tool execution context
 */
interface ToolExecutionContext {
    stepId: string;
    toolName: string;
    parameters: Record<string, unknown>;
    resourceManager: ResourceManager;
    retryPolicy?: RetryPolicy;
    approvalStatus?: ToolApprovalStatus;
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
    
    // Tool registry
    private readonly tools: Map<string, MCPTool> = new Map();
    private readonly pendingApprovals: Map<string, ToolApprovalStatus> = new Map();
    
    // Current execution context
    private currentStepId?: string;
    private currentTools?: ToolResource[];
    private resourceManager?: ResourceManager;

    constructor(eventBus: EventBus, logger: Logger) {
        this.eventBus = eventBus;
        this.logger = logger;
        
        // Initialize built-in tools
        this.initializeBuiltInTools();
    }

    /**
     * Configures orchestrator for a specific execution
     */
    configureForExecution(
        stepId: string,
        availableTools: ToolResource[],
        resourceManager: ResourceManager,
    ): void {
        this.currentStepId = stepId;
        this.currentTools = availableTools;
        this.resourceManager = resourceManager;

        this.logger.debug("[ToolOrchestrator] Configured for execution", {
            stepId,
            toolCount: availableTools.length,
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
            // Validate tool availability
            const tool = await this.validateToolAvailability(toolName);
            
            // Check approval requirements
            const approvalStatus = await this.checkApprovalRequirements(toolName);
            if (!approvalStatus.approved) {
                return this.createErrorResult(
                    `Tool ${toolName} requires approval: ${approvalStatus.reason}`,
                    Date.now() - startTime,
                );
            }

            // Create execution context
            const context: ToolExecutionContext = {
                stepId: this.currentStepId!,
                toolName,
                parameters,
                resourceManager: this.resourceManager!,
                retryPolicy,
                approvalStatus,
            };

            // Execute with retry logic
            const result = await this.executeWithRetry(tool, context);

            // Track resource usage
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
    async listTools(): Promise<MCPTool[]> {
        const availableTools: MCPTool[] = [];

        // Add registered tools
        for (const [name, tool] of this.tools) {
            if (this.isToolAvailable(name)) {
                availableTools.push(tool);
            }
        }

        // Add dynamic tools from current context
        if (this.currentTools) {
            for (const toolResource of this.currentTools) {
                if (!this.tools.has(toolResource.name)) {
                    availableTools.push(this.convertToMCPTool(toolResource));
                }
            }
        }

        return availableTools;
    }

    /**
     * Implements define_tool for schema compression
     */
    async defineTool(params: {
        toolName: string;
        variant?: string;
        operation?: string;
    }): Promise<MCPTool> {
        const { toolName, variant, operation } = params;

        this.logger.debug("[ToolOrchestrator] Defining tool schema", {
            toolName,
            variant,
            operation,
        });

        // Generate compressed schema based on parameters
        const compressedSchema = await this.generateCompressedSchema(
            toolName,
            variant,
            operation,
        );

        return compressedSchema;
    }

    /**
     * Registers a pending tool approval
     */
    async registerPendingApproval(
        toolName: string,
        parameters: Record<string, unknown>,
        timeout: number = 600000, // 10 minutes default
    ): Promise<string> {
        const approvalId = `approval_${Date.now()}_${toolName}`;
        
        const approval: ToolApprovalStatus = {
            toolName,
            approved: false,
            reason: "Awaiting user approval",
            expiresAt: new Date(Date.now() + timeout),
        };

        this.pendingApprovals.set(approvalId, approval);

        // Emit approval request event
        await this.eventBus.publish("tool.approval_required", {
            approvalId,
            toolName,
            parameters,
            stepId: this.currentStepId,
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
        const approval = this.pendingApprovals.get(approvalId);
        if (!approval) {
            throw new Error(`Approval ${approvalId} not found`);
        }

        approval.approved = approved;
        approval.approvedBy = approvedBy;
        approval.approvedAt = new Date();
        approval.reason = approved ? "User approved" : "User rejected";

        // Emit approval processed event
        await this.eventBus.publish("tool.approval_processed", {
            approvalId,
            approved,
            toolName: approval.toolName,
            stepId: this.currentStepId,
        });
    }

    /**
     * Private helper methods
     */
    private initializeBuiltInTools(): void {
        // Core MCP tools
        this.registerTool({
            name: "define_tool",
            description: "Dynamically define tool schemas for specific resource operations",
            inputSchema: {
                type: "object",
                properties: {
                    toolName: { type: "string", description: "Base tool name" },
                    variant: { type: "string", description: "Resource variant" },
                    operation: { type: "string", enum: ["find", "add", "update", "delete"] },
                },
                required: ["toolName"],
            },
            handler: async (args) => this.defineTool(args as any),
        });

        this.registerTool({
            name: "resource_manage",
            description: "Universal CRUD operations for Vrooli resources",
            inputSchema: {
                type: "object",
                properties: {
                    operation: { type: "string", enum: ["find", "add", "update", "delete"] },
                    resourceType: { type: "string" },
                    data: { type: "object" },
                },
                required: ["operation", "resourceType"],
            },
        });

        this.registerTool({
            name: "send_message",
            description: "Send messages to team members or users",
            inputSchema: {
                type: "object",
                properties: {
                    recipients: { type: "array", items: { type: "string" } },
                    message: { type: "string" },
                    metadata: { type: "object" },
                },
                required: ["recipients", "message"],
            },
        });

        this.registerTool({
            name: "run_routine",
            description: "Execute a Vrooli routine",
            inputSchema: {
                type: "object",
                properties: {
                    routineId: { type: "string" },
                    inputs: { type: "object" },
                    mode: { type: "string", enum: ["sync", "async"] },
                },
                required: ["routineId"],
            },
        });
    }

    private registerTool(tool: MCPTool): void {
        this.tools.set(tool.name, tool);
        this.logger.debug(`[ToolOrchestrator] Registered tool: ${tool.name}`);
    }

    private async validateToolAvailability(toolName: string): Promise<MCPTool> {
        // Check registered tools
        let tool = this.tools.get(toolName);
        if (tool) return tool;

        // Check current context tools
        if (this.currentTools) {
            const toolResource = this.currentTools.find(t => t.name === toolName);
            if (toolResource) {
                tool = this.convertToMCPTool(toolResource);
                this.tools.set(toolName, tool); // Cache for this execution
                return tool;
            }
        }

        throw new Error(`Tool ${toolName} not available`);
    }

    private convertToMCPTool(resource: ToolResource): MCPTool {
        return {
            name: resource.name,
            description: resource.description,
            inputSchema: {
                type: "object",
                properties: resource.parameters || {},
            },
        };
    }

    private async checkApprovalRequirements(
        toolName: string,
    ): Promise<ToolApprovalStatus> {
        // Check if tool requires approval
        const requiresApproval = this.toolRequiresApproval(toolName);
        
        if (!requiresApproval) {
            return {
                toolName,
                approved: true,
                reason: "No approval required",
            };
        }

        // Check for existing approval
        for (const approval of this.pendingApprovals.values()) {
            if (approval.toolName === toolName && approval.approved) {
                // Check if approval hasn't expired
                if (!approval.expiresAt || approval.expiresAt > new Date()) {
                    return approval;
                }
            }
        }

        return {
            toolName,
            approved: false,
            reason: "Tool requires user approval",
        };
    }

    private toolRequiresApproval(toolName: string): boolean {
        // High-risk tools that always require approval
        const highRiskTools = [
            "run_routine",
            "spawn_swarm",
            "resource_manage",
            "execute_code",
            "access_file",
        ];

        return highRiskTools.includes(toolName);
    }

    private async executeWithRetry(
        tool: MCPTool,
        context: ToolExecutionContext,
    ): Promise<ToolExecutionResult> {
        const { retryPolicy } = context;
        let lastError: Error | undefined;
        let retries = 0;

        const maxRetries = retryPolicy?.maxRetries || 0;
        const backoffStrategy = retryPolicy?.backoffStrategy || "exponential";
        const initialDelay = retryPolicy?.initialDelay || 1000;
        const maxDelay = retryPolicy?.maxDelay || 30000;

        while (retries <= maxRetries) {
            try {
                // Check resource quota before execution
                const quotaCheck = await context.resourceManager.checkQuota(
                    context.toolName,
                    this.estimateToolCost(context.toolName),
                );

                if (!quotaCheck.allowed) {
                    throw new Error(`Resource quota exceeded: ${quotaCheck.reason}`);
                }

                // Execute tool
                const result = await this.executeToolHandler(tool, context);
                
                return {
                    success: true,
                    output: result,
                    duration: Date.now() - Date.now(), // Will be calculated properly
                    retries,
                };

            } catch (error) {
                lastError = error instanceof Error ? error : new Error(String(error));
                
                if (retries < maxRetries) {
                    const delay = this.calculateRetryDelay(
                        retries,
                        backoffStrategy,
                        initialDelay,
                        maxDelay,
                    );
                    
                    this.logger.warn(`[ToolOrchestrator] Tool execution failed, retrying`, {
                        toolName: context.toolName,
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

    private async executeToolHandler(
        tool: MCPTool,
        context: ToolExecutionContext,
    ): Promise<unknown> {
        if (tool.handler) {
            // Execute built-in handler
            return await tool.handler(context.parameters);
        }

        // Execute via MCP protocol
        // TODO: Implement actual MCP execution
        this.logger.info(`[ToolOrchestrator] Executing MCP tool: ${tool.name}`, {
            parameters: context.parameters,
        });

        // Simulate execution for now
        return {
            status: "success",
            result: `Executed ${tool.name} with MCP protocol`,
        };
    }

    private isToolAvailable(toolName: string): boolean {
        // Check if tool is in current context
        if (this.currentTools) {
            return this.currentTools.some(t => t.name === toolName);
        }
        
        // Built-in tools are always available
        return this.tools.has(toolName);
    }

    private async generateCompressedSchema(
        toolName: string,
        variant?: string,
        operation?: string,
    ): Promise<MCPTool> {
        // Generate schema based on tool, variant, and operation
        const baseSchema: MCPTool = {
            name: `${toolName}_${variant}_${operation}`.toLowerCase(),
            description: `${operation} operation for ${variant} resources`,
            inputSchema: {
                type: "object",
                properties: {},
                required: [],
            },
        };

        // Add operation-specific properties
        switch (operation) {
            case "find":
                baseSchema.inputSchema.properties = {
                    filters: { type: "object", description: "Search filters" },
                    limit: { type: "number", description: "Maximum results" },
                    offset: { type: "number", description: "Result offset" },
                };
                break;
                
            case "add":
                baseSchema.inputSchema.properties = {
                    data: { type: "object", description: `${variant} data to create` },
                };
                baseSchema.inputSchema.required = ["data"];
                break;
                
            case "update":
                baseSchema.inputSchema.properties = {
                    id: { type: "string", description: `${variant} ID` },
                    updates: { type: "object", description: "Fields to update" },
                };
                baseSchema.inputSchema.required = ["id", "updates"];
                break;
                
            case "delete":
                baseSchema.inputSchema.properties = {
                    id: { type: "string", description: `${variant} ID to delete` },
                };
                baseSchema.inputSchema.required = ["id"];
                break;
        }

        return baseSchema;
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