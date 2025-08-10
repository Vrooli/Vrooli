import chalk from "chalk";
import inquirer from "inquirer";
import { type ApiClient } from "./client.js";
import { type ConfigManager } from "./config.js";
import { type RespondToToolApprovalInput } from "@vrooli/shared";
import { LIMITS, TIMEOUTS, TOOL_APPROVAL } from "./constants.js";

// Constants for conversion
const SECONDS_IN_MS = TIMEOUTS.TOOL_APPROVAL_DEFAULT;

export interface ToolApprovalRequest {
    pendingId: string;
    toolCallId: string;
    toolName: string;
    toolArguments: Record<string, unknown>;
    callerBotId: string;
    callerBotName?: string;
    approvalTimeoutAt?: number;
    estimatedCost?: string;
    conversationId: string;
}

export interface ToolExecutionInfo {
    toolCallId: string;
    toolName: string;
    callerBotId: string;
    status: "pending" | "executing" | "completed" | "failed";
    startTime?: Date;
    endTime?: Date;
    result?: unknown;
    error?: string;
}

// Type guard for tool approval data
function isToolApprovalData(data: unknown): data is {
    pendingId: string;
    toolCallId: string;
    toolName: string;
    toolArguments?: Record<string, unknown>;
    callerBotId: string;
    callerBotName?: string;
    approvalTimeoutAt?: number;
    estimatedCost?: string;
} {
    return (
        typeof data === "object" &&
        data !== null &&
        "pendingId" in data &&
        "toolCallId" in data &&
        "toolName" in data &&
        "callerBotId" in data &&
        typeof (data as Record<string, unknown>).pendingId === "string" &&
        typeof (data as Record<string, unknown>).toolCallId === "string" &&
        typeof (data as Record<string, unknown>).toolName === "string" &&
        typeof (data as Record<string, unknown>).callerBotId === "string"
    );
}

// Type guard for tool execution data
function isToolExecutionData(data: unknown): data is {
    toolCallId: string;
    result?: unknown;
    error?: string;
} {
    return (
        typeof data === "object" &&
        data !== null &&
        "toolCallId" in data &&
        typeof (data as Record<string, unknown>).toolCallId === "string"
    );
}

export class ToolApprovalHandler {
    private pendingApprovals = new Map<string, ToolApprovalRequest>();
    private executingTools = new Map<string, ToolExecutionInfo>();
    private approvalInProgress = false;

    constructor(
        private client: ApiClient,
        private config: ConfigManager,
        private onToolStatusUpdate?: (info: ToolExecutionInfo) => void,
    ) {}

    /**
     * Handle a tool approval request from WebSocket
     */
    async handleApprovalRequest(data: unknown, conversationId: string): Promise<void> {
        if (!isToolApprovalData(data)) {
            throw new Error("Invalid tool approval data");
        }

        const request: ToolApprovalRequest = {
            pendingId: data.pendingId,
            toolCallId: data.toolCallId,
            toolName: data.toolName,
            toolArguments: data.toolArguments || {},
            callerBotId: data.callerBotId,
            callerBotName: data.callerBotName,
            approvalTimeoutAt: data.approvalTimeoutAt,
            estimatedCost: data.estimatedCost,
            conversationId,
        };

        this.pendingApprovals.set(request.pendingId, request);
        await this.promptUserForApproval(request);
    }

    /**
     * Prompt user for tool approval decision
     */
    private async promptUserForApproval(request: ToolApprovalRequest): Promise<void> {
        // If another approval is in progress, queue this one
        if (this.approvalInProgress) {
            setTimeout(() => this.promptUserForApproval(request), TOOL_APPROVAL.POLLING_INTERVAL_MS);
            return;
        }

        this.approvalInProgress = true;

        try {
            // Display tool request information
            this.displayToolRequest(request);

            // Get user decision
            const decision = await this.getUserApprovalDecision(request);

            // Send approval response
            await this.sendApprovalResponse(request, decision.approved, decision.reason);

            // Update tracking
            if (decision.approved) {
                this.executingTools.set(request.toolCallId, {
                    toolCallId: request.toolCallId,
                    toolName: request.toolName,
                    callerBotId: request.callerBotId,
                    status: "pending",
                    startTime: new Date(),
                });
            }

        } catch (error) {
            console.log(chalk.red(`Error handling tool approval: ${error instanceof Error ? error.message : String(error)}`));
        } finally {
            this.approvalInProgress = false;
            this.pendingApprovals.delete(request.pendingId);
        }
    }

    /**
     * Display tool request information to user
     */
    private displayToolRequest(request: ToolApprovalRequest): void {
        console.log();
        console.log(chalk.bold.yellow("🔧 Tool Execution Request"));
        console.log("─".repeat(TOOL_APPROVAL.SEPARATOR_LENGTH));
        
        console.log(`${chalk.bold("Bot:")} ${request.callerBotName || request.callerBotId}`);
        console.log(`${chalk.bold("Tool:")} ${chalk.cyan(request.toolName)}`);
        
        if (request.estimatedCost) {
            console.log(`${chalk.bold("Estimated Cost:")} ${chalk.yellow(request.estimatedCost)} credits`);
        }

        // Display tool arguments in a readable format
        if (Object.keys(request.toolArguments).length > 0) {
            console.log(`${chalk.bold("Arguments:")}`);
            for (const [key, value] of Object.entries(request.toolArguments)) {
                const displayValue = this.formatArgumentValue(value);
                console.log(`  ${chalk.gray(key)}: ${displayValue}`);
            }
        }

        // Show timeout if present
        if (request.approvalTimeoutAt) {
            const timeoutDate = new Date(request.approvalTimeoutAt);
            const timeLeft = Math.max(0, Math.ceil((timeoutDate.getTime() - Date.now()) / SECONDS_IN_MS));
            console.log(`${chalk.bold("Timeout:")} ${timeLeft} seconds`);
        }

        console.log("─".repeat(TOOL_APPROVAL.SEPARATOR_LENGTH));
    }

    /**
     * Get user approval decision via interactive prompt
     */
    private async getUserApprovalDecision(request: ToolApprovalRequest): Promise<{
        approved: boolean;
        reason?: string;
    }> {
        const choices = [
            { name: `${chalk.green("✓ Approve")} - Allow tool execution`, value: "approve" },
            { name: `${chalk.red("✗ Deny")} - Block tool execution`, value: "deny" },
            { name: `${chalk.blue("ℹ Details")} - View detailed tool information`, value: "details" },
        ];

        let shouldContinue = true;
        while (shouldContinue) {
            const { action } = await inquirer.prompt([
                {
                    type: "list",
                    name: "action",
                    message: "Do you want to allow this tool execution?",
                    choices,
                    pageSize: 10,
                },
            ]);

            switch (action) {
                case "approve":
                    return { approved: true };
                
                case "deny": {
                    const { reason } = await inquirer.prompt([
                        {
                            type: "input",
                            name: "reason",
                            message: "Reason for denial (optional):",
                        },
                    ]);
                    return { approved: false, reason: reason || undefined };
                }
                
                case "details":
                    this.displayDetailedToolInfo(request);
                    break; // Loop back to show choices again
                
                default:
                    shouldContinue = false;
                    return { approved: false };
            }
        }
        // This should never be reached due to the while loop structure,
        // but TypeScript needs it for type safety
        return { approved: false };
    }

    /**
     * Display detailed tool information
     */
    private displayDetailedToolInfo(request: ToolApprovalRequest): void {
        console.log();
        console.log(chalk.bold.blue("📋 Detailed Tool Information"));
        console.log("─".repeat(TOOL_APPROVAL.SEPARATOR_LENGTH));
        
        console.log(`${chalk.bold("Tool Call ID:")} ${request.toolCallId}`);
        console.log(`${chalk.bold("Pending ID:")} ${request.pendingId}`);
        console.log(`${chalk.bold("Caller Bot ID:")} ${request.callerBotId}`);
        console.log(`${chalk.bold("Conversation ID:")} ${request.conversationId}`);
        
        if (Object.keys(request.toolArguments).length > 0) {
            console.log(`${chalk.bold("Full Arguments JSON:")}`);
            console.log(JSON.stringify(request.toolArguments, null, 2));
        }
        
        console.log("─".repeat(TOOL_APPROVAL.SEPARATOR_LENGTH));
        console.log();
    }

    /**
     * Send approval response to API
     */
    private async sendApprovalResponse(
        request: ToolApprovalRequest,
        approved: boolean,
        reason?: string,
    ): Promise<void> {
        try {
            const input: RespondToToolApprovalInput = {
                conversationId: request.conversationId,
                pendingId: request.pendingId,
                approved,
                reason,
            };

            await this.client.post("/task/respondToToolApproval", input);

            // Display confirmation
            if (approved) {
                console.log(chalk.green(`✓ Tool execution approved: ${request.toolName}`));
            } else {
                console.log(chalk.red(`✗ Tool execution denied: ${request.toolName}`));
                if (reason) {
                    console.log(chalk.gray(`  Reason: ${reason}`));
                }
            }

        } catch (error) {
            console.log(chalk.red(`Failed to send approval response: ${error instanceof Error ? error.message : String(error)}`));
        }
    }

    /**
     * Handle tool execution status updates
     */
    handleToolExecution(event: string, data: unknown): void {
        if (!isToolExecutionData(data)) {
            console.error("Invalid tool execution data");
            return;
        }

        const toolInfo = this.executingTools.get(data.toolCallId);
        
        if (!toolInfo) return;

        switch (event) {
            case "TOOL_CALLED":
                toolInfo.status = "executing";
                console.log(chalk.cyan(`🔧 Executing: ${toolInfo.toolName}`));
                break;
            
            case "TOOL_COMPLETED":
                toolInfo.status = "completed";
                toolInfo.endTime = new Date();
                toolInfo.result = data.result;
                console.log(chalk.green(`✅ Tool completed: ${toolInfo.toolName}`));
                this.displayToolResult(toolInfo);
                this.executingTools.delete(data.toolCallId);
                break;
            
            case "TOOL_FAILED":
                toolInfo.status = "failed";
                toolInfo.endTime = new Date();
                toolInfo.error = data.error || "Unknown error";
                console.log(chalk.red(`❌ Tool failed: ${toolInfo.toolName}`));
                console.log(chalk.red(`   Error: ${toolInfo.error}`));
                this.executingTools.delete(data.toolCallId);
                break;
        }

        // Notify callback if provided
        if (this.onToolStatusUpdate) {
            this.onToolStatusUpdate(toolInfo);
        }
    }

    /**
     * Display tool execution result
     */
    private displayToolResult(toolInfo: ToolExecutionInfo): void {
        if (!toolInfo.result) return;

        console.log(chalk.bold("📊 Tool Result:"));
        
        // Calculate execution time
        if (toolInfo.startTime && toolInfo.endTime) {
            const duration = toolInfo.endTime.getTime() - toolInfo.startTime.getTime();
            console.log(chalk.gray(`   Duration: ${duration}ms`));
        }

        // Display result based on type
        if (typeof toolInfo.result === "string") {
            console.log(`   ${toolInfo.result}`);
        } else if (typeof toolInfo.result === "object") {
            const resultStr = JSON.stringify(toolInfo.result, null, 2);
            // Truncate very long results
            if (resultStr.length > LIMITS.MAX_RESULT_DISPLAY_LENGTH) {
                console.log(`   ${resultStr.substring(0, LIMITS.MAX_RESULT_DISPLAY_LENGTH)}...`);
                console.log(chalk.gray("   (result truncated)"));
            } else {
                console.log(`   ${resultStr}`);
            }
        } else {
            console.log(`   ${String(toolInfo.result)}`);
        }
    }

    /**
     * Format argument value for display
     */
    private formatArgumentValue(value: unknown): string {
        if (typeof value === "string") {
            // Truncate long strings
            return value.length > 100 ? `"${value.substring(0, 100)}..."` : `"${value}"`;
        } else if (typeof value === "object") {
            const jsonStr = JSON.stringify(value);
            return jsonStr.length > 100 ? `${jsonStr.substring(0, 100)}...` : jsonStr;
        }
        return String(value);
    }

    /**
     * Handle tool execution start event
     */
    handleExecutionStart(data: unknown): void {
        if (!data || typeof data !== "object" || !("toolCallId" in data) || !("toolName" in data) || !("callerBotId" in data)) {
            throw new Error("Invalid tool execution start data");
        }

        const executionData = data as { toolCallId: string; toolName: string; callerBotId: string };
        
        const toolInfo: ToolExecutionInfo = {
            toolCallId: executionData.toolCallId,
            toolName: executionData.toolName,
            callerBotId: executionData.callerBotId,
            status: "executing",
            startTime: new Date(),
        };

        this.executingTools.set(executionData.toolCallId, toolInfo);
        
        if (this.onToolStatusUpdate) {
            this.onToolStatusUpdate(toolInfo);
        }
    }

    /**
     * Handle tool execution completion event
     */
    handleExecutionComplete(data: unknown): void {
        if (!data || typeof data !== "object" || !("toolCallId" in data)) {
            throw new Error("Invalid tool execution complete data");
        }

        const completionData = data as { toolCallId: string; result?: unknown; error?: string };
        const toolInfo = this.executingTools.get(completionData.toolCallId);

        if (!toolInfo) {
            console.log(chalk.yellow(`Received completion for unknown tool: ${completionData.toolCallId}`));
            return;
        }

        toolInfo.endTime = new Date();
        if (completionData.error) {
            toolInfo.status = "failed";
            toolInfo.error = completionData.error;
        } else {
            toolInfo.status = "completed";
            toolInfo.result = completionData.result;
        }

        if (this.onToolStatusUpdate) {
            this.onToolStatusUpdate(toolInfo);
        }

        this.executingTools.delete(completionData.toolCallId);
    }

    /**
     * Get list of currently executing tools
     */
    getExecutingTools(): ToolExecutionInfo[] {
        return Array.from(this.executingTools.values());
    }

    /**
     * Display currently executing tools
     */
    displayExecutingTools(): void {
        const executing = this.getExecutingTools();
        
        if (executing.length === 0) {
            console.log(chalk.gray("No tools currently executing"));
            return;
        }

        console.log(chalk.bold("🔧 Currently Executing Tools:"));
        executing.forEach(tool => {
            const duration = tool.startTime ? Date.now() - tool.startTime.getTime() : 0;
            const MS_PER_SECOND = 1000;
            console.log(`  ${chalk.cyan(tool.toolName)} (${Math.floor(duration / MS_PER_SECOND)}s)`);
        });
    }

    /**
     * Get current status of all tools
     */
    getToolStatus(): {
        pending: ToolApprovalRequest[];
        executing: ToolExecutionInfo[];
    } {
        return {
            pending: Array.from(this.pendingApprovals.values()),
            executing: Array.from(this.executingTools.values()),
        };
    }

    /**
     * Clean up resources
     */
    cleanup(): void {
        this.pendingApprovals.clear();
        this.executingTools.clear();
        this.approvalInProgress = false;
    }
}
