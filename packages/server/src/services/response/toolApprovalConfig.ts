/**
 * Tool Approval Configuration
 * 
 * Defines which tools require approval before execution.
 * This can be configured at runtime based on user preferences,
 * security policies, or tool risk levels.
 */

import { McpSwarmToolName, McpToolName } from "@vrooli/shared";

/**
 * Risk levels for tools
 */
export enum ToolRiskLevel {
    /** No risk - informational only */
    NONE = "none",
    /** Low risk - read-only operations */
    LOW = "low",
    /** Medium risk - modifies data or costs credits */
    MEDIUM = "medium",
    /** High risk - executes code, spawns processes, or high cost */
    HIGH = "high",
    /** Critical risk - system-level changes or security-sensitive */
    CRITICAL = "critical",
}

/**
 * Tool approval policy
 */
export interface ToolApprovalPolicy {
    /** Default approval requirement for unlisted tools */
    defaultRequiresApproval: boolean;
    /** Risk threshold - tools at or above this level require approval */
    riskThreshold: ToolRiskLevel;
    /** Specific tool overrides */
    toolOverrides: Map<string, boolean>;
    /** Auto-approve for trusted bots */
    trustedBots: Set<string>;
    /** Maximum auto-approval credit cost */
    maxAutoApprovalCredits: bigint;
}

/**
 * Default tool risk assessments
 */
const TOOL_RISK_LEVELS = new Map<string, ToolRiskLevel>([
    // MCP Tools
    [McpToolName.DefineTool, ToolRiskLevel.NONE], // Just returns info
    [McpToolName.SendMessage, ToolRiskLevel.LOW], // Sends messages but controlled
    [McpToolName.ResourceManage, ToolRiskLevel.MEDIUM], // Creates/modifies resources
    [McpToolName.RunRoutine, ToolRiskLevel.HIGH], // Executes arbitrary routines
    [McpToolName.SpawnSwarm, ToolRiskLevel.HIGH], // Spawns new swarms (high cost)

    // Swarm Tools
    [McpSwarmToolName.UpdateSwarmSharedState, ToolRiskLevel.LOW], // Updates state
    [McpSwarmToolName.EndSwarm, ToolRiskLevel.MEDIUM], // Ends swarm execution

    // OpenAI Tools
    ["web_search", ToolRiskLevel.LOW], // External API call
    ["file_search", ToolRiskLevel.LOW], // Searches files
    ["code_interpreter", ToolRiskLevel.CRITICAL], // Executes code
    ["image_generation", ToolRiskLevel.MEDIUM], // Costs credits
    ["computer-preview", ToolRiskLevel.CRITICAL], // GUI automation
]);

/**
 * Default approval policy
 */
export const DEFAULT_APPROVAL_POLICY: ToolApprovalPolicy = {
    defaultRequiresApproval: true,
    riskThreshold: ToolRiskLevel.MEDIUM,
    toolOverrides: new Map(),
    trustedBots: new Set(),
    maxAutoApprovalCredits: BigInt(100),
};

/**
 * Tool approval configuration service
 */
export class ToolApprovalConfig {
    private policy: ToolApprovalPolicy;

    constructor(policy?: Partial<ToolApprovalPolicy>) {
        this.policy = {
            ...DEFAULT_APPROVAL_POLICY,
            ...policy,
            toolOverrides: new Map([
                ...DEFAULT_APPROVAL_POLICY.toolOverrides,
                ...(policy?.toolOverrides || []),
            ]),
            trustedBots: new Set([
                ...DEFAULT_APPROVAL_POLICY.trustedBots,
                ...(policy?.trustedBots || []),
            ]),
        };
    }

    /**
     * Check if a tool requires approval
     */
    requiresApproval(
        toolName: string,
        callerBotId?: string,
        estimatedCredits?: bigint,
    ): boolean {
        // Check tool-specific override first
        const toolOverride = this.policy.toolOverrides.get(toolName);
        if (toolOverride !== undefined) {
            return toolOverride;
        }

        // Check if caller is trusted
        if (callerBotId && this.policy.trustedBots.has(callerBotId)) {
            // Still check credit limit for trusted bots
            if (estimatedCredits && estimatedCredits > this.policy.maxAutoApprovalCredits) {
                return true;
            }
            return false;
        }

        // Check risk level
        const toolRisk = TOOL_RISK_LEVELS.get(toolName);
        if (toolRisk) {
            return this.compareRiskLevels(toolRisk, this.policy.riskThreshold) >= 0;
        }

        // Use default policy
        return this.policy.defaultRequiresApproval;
    }

    /**
     * Get risk level for a tool
     */
    getRiskLevel(toolName: string): ToolRiskLevel {
        return TOOL_RISK_LEVELS.get(toolName) || ToolRiskLevel.MEDIUM;
    }

    /**
     * Update policy at runtime
     */
    updatePolicy(updates: Partial<ToolApprovalPolicy>): void {
        if (updates.defaultRequiresApproval !== undefined) {
            this.policy.defaultRequiresApproval = updates.defaultRequiresApproval;
        }
        if (updates.riskThreshold !== undefined) {
            this.policy.riskThreshold = updates.riskThreshold;
        }
        if (updates.maxAutoApprovalCredits !== undefined) {
            this.policy.maxAutoApprovalCredits = updates.maxAutoApprovalCredits;
        }
        if (updates.toolOverrides) {
            for (const [tool, requires] of updates.toolOverrides) {
                this.policy.toolOverrides.set(tool, requires);
            }
        }
        if (updates.trustedBots) {
            for (const bot of updates.trustedBots) {
                this.policy.trustedBots.add(bot);
            }
        }
    }

    /**
     * Compare risk levels
     */
    private compareRiskLevels(a: ToolRiskLevel, b: ToolRiskLevel): number {
        const levels = [
            ToolRiskLevel.NONE,
            ToolRiskLevel.LOW,
            ToolRiskLevel.MEDIUM,
            ToolRiskLevel.HIGH,
            ToolRiskLevel.CRITICAL,
        ];
        return levels.indexOf(a) - levels.indexOf(b);
    }
}

/**
 * Global instance for easy access
 */
export const toolApprovalConfig = new ToolApprovalConfig();
