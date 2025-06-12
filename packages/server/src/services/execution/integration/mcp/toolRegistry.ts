import { McpToolName, McpSwarmToolName, type SessionUser } from "@vrooli/shared";
import { type Logger } from "winston";
import { ToolRegistry as MCPToolRegistry } from "../../../mcp/registry.js";
import { BuiltInTools, SwarmTools } from "../../../mcp/tools.js";
import { type Tool, type ToolResponse } from "../../../mcp/types.js";
import { CompositeToolRunner, McpToolRunner, OpenAIToolRunner } from "../../../conversation/toolRunner.js";
import { type ConversationStateStore } from "../../../conversation/chatStore.js";
import { type ToolMeta } from "../../../conversation/types.js";
import type { 
    ToolResource, 
    ToolExecutionRequest, 
    ToolExecutionResult,
} from "@vrooli/shared";
import { MONITORING_TOOL_DEFINITIONS, createMonitoringToolInstances } from "../tools/index.js";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { type RollingHistoryAdapter as RollingHistory } from "../../monitoring/adapters/RollingHistoryAdapter.js";

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
 * Tool execution context for the integrated system
 */
export interface IntegratedToolContext {
    stepId: string;
    runId?: string;
    swarmId?: string;
    conversationId?: string;
    user: SessionUser;
    logger: Logger;
    metadata?: Record<string, unknown>;
}

/**
 * IntegratedToolRegistry - Bridges the MCP tool system with the execution architecture
 * 
 * This registry serves as the single source of truth for tool discovery and execution,
 * integrating:
 * - MCP built-in tools (defineTool, resourceManage, sendMessage, etc.)
 * - Swarm-specific tools (updateSwarmSharedState, endSwarm)
 * - OpenAI tools (web_search, file_search)
 * - Dynamic tool registration from routines and swarms
 * 
 * Key features:
 * - Unified tool discovery across all sources
 * - Consistent execution interface for all tool types
 * - Approval system for high-risk operations
 * - Resource tracking and quota management
 * - Schema compression via defineTool
 */
export class IntegratedToolRegistry {
    private static instance: IntegratedToolRegistry;
    
    private readonly mcpRegistry: MCPToolRegistry;
    private readonly compositeRunner: CompositeToolRunner;
    private readonly logger: Logger;
    
    // Dynamic tool registry for runtime additions
    private readonly dynamicTools: Map<string, Tool> = new Map();
    
    // Approval tracking
    private readonly pendingApprovals: Map<string, ToolApprovalStatus> = new Map();
    
    // Tool usage metrics
    private readonly usageMetrics: Map<string, {
        calls: number;
        totalDuration: number;
        errors: number;
        lastUsed: Date;
    }> = new Map();
    
    // Monitoring tool instances
    private monitoringToolInstances?: Map<string, (params: any) => Promise<ToolResponse>>;

    private constructor(logger: Logger, conversationStore?: ConversationStateStore) {
        this.logger = logger;
        this.mcpRegistry = new MCPToolRegistry(logger);
        
        // Initialize tool runners
        const swarmTools = conversationStore ? new SwarmTools(logger, conversationStore) : undefined;
        const mcpRunner = new McpToolRunner(logger, swarmTools);
        const openaiRunner = new OpenAIToolRunner(null); // Can be initialized with OpenAI client later
        
        this.compositeRunner = new CompositeToolRunner(mcpRunner, openaiRunner);
    }

    /**
     * Gets the singleton instance of the integrated tool registry
     */
    static getInstance(logger: Logger, conversationStore?: ConversationStateStore): IntegratedToolRegistry {
        if (!IntegratedToolRegistry.instance) {
            IntegratedToolRegistry.instance = new IntegratedToolRegistry(logger, conversationStore);
        }
        return IntegratedToolRegistry.instance;
    }

    /**
     * Initialize monitoring tools with required dependencies
     */
    initializeMonitoringTools(
        user: SessionUser,
        eventBus: EventBus,
        rollingHistory?: RollingHistory,
    ): void {
        this.monitoringToolInstances = createMonitoringToolInstances(
            user,
            this.logger,
            eventBus,
            rollingHistory,
        );

        // Register monitoring tool definitions
        for (const toolDef of MONITORING_TOOL_DEFINITIONS) {
            this.registerDynamicTool(toolDef, {
                scope: "global",
            });
        }

        this.logger.info("[IntegratedToolRegistry] Monitoring tools initialized", {
            toolCount: MONITORING_TOOL_DEFINITIONS.length,
            tools: MONITORING_TOOL_DEFINITIONS.map(t => t.name),
        });
    }

    /**
     * Lists all available tools from all sources
     */
    async listAvailableTools(context: IntegratedToolContext): Promise<Tool[]> {
        const tools: Tool[] = [];
        
        // Add MCP built-in tools
        const builtInDefs = this.mcpRegistry.getBuiltInDefinitions();
        for (const def of builtInDefs) {
            tools.push(this.convertToTool(def));
        }
        
        // Add swarm tools if in swarm context
        if (context.swarmId || context.conversationId) {
            const swarmDefs = this.mcpRegistry.getSwarmToolDefinitions();
            for (const def of swarmDefs) {
                tools.push(this.convertToTool(def));
            }
        }
        
        // Add dynamic tools
        for (const [name, tool] of this.dynamicTools) {
            if (this.isToolAvailableInContext(name, context)) {
                tools.push(tool);
            }
        }
        
        // Add OpenAI tools if configured
        const openAITools: Tool[] = [
            {
                name: "web_search",
                description: "Search the web for information",
                inputSchema: {
                    type: "object",
                    properties: {
                        query: { type: "string", description: "Search query" },
                        max_results: { type: "number", description: "Maximum number of results" },
                    },
                    required: ["query"],
                },
                annotations: {
                    title: "Web Search",
                    openWorldHint: true,
                },
            },
            {
                name: "file_search", 
                description: "Search through uploaded files",
                inputSchema: {
                    type: "object",
                    properties: {
                        query: { type: "string", description: "Search query" },
                        file_ids: { type: "array", items: { type: "string" } },
                    },
                    required: ["query"],
                },
                annotations: {
                    title: "File Search",
                    readOnlyHint: true,
                },
            },
        ];
        
        tools.push(...openAITools);
        
        return tools;
    }

    /**
     * Executes a tool with proper context and error handling
     */
    async executeTool(
        request: ToolExecutionRequest,
        context: IntegratedToolContext,
    ): Promise<ToolExecutionResult> {
        const startTime = Date.now();
        const { toolName, parameters } = request;
        
        this.logger.info(`[IntegratedToolRegistry] Executing tool: ${toolName}`, {
            stepId: context.stepId,
            runId: context.runId,
            swarmId: context.swarmId,
        });
        
        try {
            // Check if this is a monitoring tool
            if (this.monitoringToolInstances?.has(toolName)) {
                const toolExecutor = this.monitoringToolInstances.get(toolName)!;
                
                try {
                    const toolResponse = await toolExecutor(parameters);
                    
                    // Track metrics
                    this.updateUsageMetrics(toolName, Date.now() - startTime, !!toolResponse.isError);
                    
                    return {
                        success: !toolResponse.isError,
                        output: toolResponse.content?.[0]?.text || JSON.stringify(toolResponse),
                        duration: Date.now() - startTime,
                        retries: 0,
                        metadata: {
                            toolType: "monitoring",
                        },
                    };
                } catch (error) {
                    this.updateUsageMetrics(toolName, Date.now() - startTime, true);
                    return {
                        success: false,
                        error: error instanceof Error ? error.message : "Monitoring tool execution failed",
                        duration: Date.now() - startTime,
                        retries: 0,
                    };
                }
            }
            
            // Check if tool requires approval
            const approvalStatus = await this.checkApprovalRequirements(toolName, parameters, context);
            if (!approvalStatus.approved) {
                return {
                    success: false,
                    error: `Tool ${toolName} requires approval: ${approvalStatus.reason}`,
                    duration: Date.now() - startTime,
                    retries: 0,
                };
            }
            
            // Build tool metadata for execution
            const toolMeta: ToolMeta = {
                conversationId: context.conversationId,
                sessionUser: context.user,
                callerBotId: context.metadata?.botId as string | undefined,
            };
            
            // Execute through composite runner
            const result = await this.compositeRunner.run(toolName, parameters, toolMeta);
            
            // Track metrics
            this.updateUsageMetrics(toolName, Date.now() - startTime, !result.ok);
            
            if (result.ok) {
                return {
                    success: true,
                    output: result.data.output,
                    duration: Date.now() - startTime,
                    retries: 0,
                    metadata: {
                        creditsUsed: result.data.creditsUsed,
                    },
                };
            } else {
                return {
                    success: false,
                    error: result.error.message,
                    duration: Date.now() - startTime,
                    retries: 0,
                    metadata: {
                        errorCode: result.error.code,
                        creditsUsed: result.error.creditsUsed,
                    },
                };
            }
            
        } catch (error) {
            this.logger.error(`[IntegratedToolRegistry] Tool execution failed: ${toolName}`, {
                error: error instanceof Error ? error.message : String(error),
                context,
            });
            
            this.updateUsageMetrics(toolName, Date.now() - startTime, true);
            
            return {
                success: false,
                error: error instanceof Error ? error.message : "Unknown error",
                duration: Date.now() - startTime,
                retries: 0,
            };
        }
    }

    /**
     * Gets a specific tool definition by name
     */
    async getToolDefinition(toolName: string, context: IntegratedToolContext): Promise<Tool | undefined> {
        // Check MCP registry first
        const mcpDef = this.mcpRegistry.getToolDefinition(toolName);
        if (mcpDef) {
            return this.convertToTool(mcpDef);
        }
        
        // Check dynamic tools
        const dynamicTool = this.dynamicTools.get(toolName);
        if (dynamicTool && this.isToolAvailableInContext(toolName, context)) {
            return dynamicTool;
        }
        
        // Check OpenAI tools
        if (toolName === "web_search" || toolName === "file_search") {
            const tools = await this.listAvailableTools(context);
            return tools.find(t => t.name === toolName);
        }
        
        return undefined;
    }

    /**
     * Registers a dynamic tool at runtime
     */
    registerDynamicTool(tool: Tool, context?: {
        runId?: string;
        swarmId?: string;
        scope?: "global" | "run" | "swarm";
    }): void {
        const toolKey = context?.scope === "global" 
            ? tool.name 
            : `${tool.name}_${context?.runId || context?.swarmId || "temp"}`;
            
        this.dynamicTools.set(toolKey, {
            ...tool,
            annotations: {
                ...tool.annotations,
                scope: context?.scope || "global",
                runId: context?.runId,
                swarmId: context?.swarmId,
            },
        });
        
        this.logger.info(`[IntegratedToolRegistry] Registered dynamic tool: ${tool.name}`, {
            scope: context?.scope || "global",
            runId: context?.runId,
            swarmId: context?.swarmId,
        });
    }

    /**
     * Removes a dynamic tool
     */
    unregisterDynamicTool(toolName: string, context?: {
        runId?: string;
        swarmId?: string;
    }): void {
        const toolKey = context?.runId || context?.swarmId 
            ? `${toolName}_${context.runId || context.swarmId}`
            : toolName;
            
        if (this.dynamicTools.delete(toolKey)) {
            this.logger.info(`[IntegratedToolRegistry] Unregistered dynamic tool: ${toolName}`);
        }
    }

    /**
     * Registers a pending tool approval
     */
    async registerPendingApproval(
        toolName: string,
        parameters: Record<string, unknown>,
        context: IntegratedToolContext,
        timeout = 600000, // 10 minutes default
    ): Promise<string> {
        const approvalId = `approval_${Date.now()}_${toolName}_${context.stepId}`;
        
        const approval: ToolApprovalStatus = {
            toolName,
            approved: false,
            reason: "Awaiting user approval",
            expiresAt: new Date(Date.now() + timeout),
        };
        
        this.pendingApprovals.set(approvalId, approval);
        
        this.logger.info(`[IntegratedToolRegistry] Registered pending approval`, {
            approvalId,
            toolName,
            stepId: context.stepId,
        });
        
        return approvalId;
    }

    /**
     * Processes a tool approval
     */
    processApproval(
        approvalId: string,
        approved: boolean,
        approvedBy?: string,
    ): void {
        const approval = this.pendingApprovals.get(approvalId);
        if (!approval) {
            throw new Error(`Approval ${approvalId} not found`);
        }
        
        approval.approved = approved;
        approval.approvedBy = approvedBy;
        approval.approvedAt = new Date();
        approval.reason = approved ? "User approved" : "User rejected";
        
        this.logger.info(`[IntegratedToolRegistry] Processed approval`, {
            approvalId,
            approved,
            toolName: approval.toolName,
        });
    }

    /**
     * Gets usage metrics for a tool
     */
    getUsageMetrics(toolName: string) {
        return this.usageMetrics.get(toolName);
    }

    /**
     * Clears expired approvals
     */
    cleanupExpiredApprovals(): void {
        const now = new Date();
        let cleaned = 0;
        
        for (const [id, approval] of this.pendingApprovals) {
            if (approval.expiresAt && approval.expiresAt < now) {
                this.pendingApprovals.delete(id);
                cleaned++;
            }
        }
        
        if (cleaned > 0) {
            this.logger.debug(`[IntegratedToolRegistry] Cleaned up ${cleaned} expired approvals`);
        }
    }

    /**
     * Private helper methods
     */
    private convertToTool(def: any): Tool {
        return {
            name: def.name,
            description: def.description,
            inputSchema: def.inputSchema || {},
            annotations: def.annotations || {},
        };
    }

    private isToolAvailableInContext(toolName: string, context: IntegratedToolContext): boolean {
        const tool = this.dynamicTools.get(toolName);
        if (!tool) return false;
        
        const annotations = tool.annotations || {};
        
        // Global tools are always available
        if (annotations.scope === "global") return true;
        
        // Run-scoped tools
        if (annotations.scope === "run" && annotations.runId === context.runId) return true;
        
        // Swarm-scoped tools
        if (annotations.scope === "swarm" && annotations.swarmId === context.swarmId) return true;
        
        return false;
    }

    private async checkApprovalRequirements(
        toolName: string,
        parameters: Record<string, unknown>,
        context: IntegratedToolContext,
    ): Promise<ToolApprovalStatus> {
        // Tools that always require approval
        const highRiskTools = [
            McpToolName.RunRoutine,
            McpToolName.SpawnSwarm,
            McpToolName.ResourceManage,
            McpSwarmToolName.EndSwarm,
        ];
        
        // Check if tool is high-risk
        const requiresApproval = highRiskTools.includes(toolName as any) ||
            this.isDestructiveOperation(toolName, parameters);
            
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
            reason: "Tool requires user approval due to high-risk operation",
        };
    }

    private isDestructiveOperation(toolName: string, parameters: Record<string, unknown>): boolean {
        // Check for destructive resource_manage operations
        if (toolName === McpToolName.ResourceManage) {
            const op = parameters.op || parameters.operation;
            return op === "delete";
        }
        
        return false;
    }

    private updateUsageMetrics(toolName: string, duration: number, isError: boolean): void {
        const existing = this.usageMetrics.get(toolName) || {
            calls: 0,
            totalDuration: 0,
            errors: 0,
            lastUsed: new Date(),
        };
        
        this.usageMetrics.set(toolName, {
            calls: existing.calls + 1,
            totalDuration: existing.totalDuration + duration,
            errors: existing.errors + (isError ? 1 : 0),
            lastUsed: new Date(),
        });
    }
}

/**
 * Helper function to convert ToolResource to Tool format
 */
export function convertToolResourceToTool(resource: ToolResource): Tool {
    return {
        name: resource.name,
        description: resource.description,
        inputSchema: {
            type: "object",
            properties: resource.parameters || {},
            required: resource.required || [],
        },
        estimatedCost: resource.estimatedCost?.toString(),
        annotations: {
            title: resource.displayName,
            readOnlyHint: resource.readOnly,
            openWorldHint: resource.externalAccess,
        },
    };
}