import { type Team } from "../../api/types.js";
import { type PassableLogger } from "../../consts/commonTypes.js";
import { BaseConfig, type BaseConfigObject, type ModelConfig } from "./base.js";
import { type BlackboardItem } from "./chat.js";

/**
 * Direct resource allocation quota for team instances
 */
export interface ResourceQuota {
    /** GPU allocation percentage (0-100) */
    gpuPercentage: number;
    /** RAM allocation in gigabytes */
    ramGB: number;
    /** CPU core allocation */
    cpuCores: number;
    /** Storage allocation in gigabytes */
    storageGB: number;
}

/**
 * Predefined resource quota configurations for common team types
 */
export const STANDARD_RESOURCE_QUOTAS = {
    /** Edge deployment, basic automation */
    light: { gpuPercentage: 10, ramGB: 8, cpuCores: 2, storageGB: 50 } as ResourceQuota,
    /** Typical profit swarms, balanced workloads */
    standard: { gpuPercentage: 20, ramGB: 16, cpuCores: 4, storageGB: 100 } as ResourceQuota,
    /** Director swarms, complex coordination */
    heavy: { gpuPercentage: 35, ramGB: 32, cpuCores: 8, storageGB: 200 } as ResourceQuota,
    /** Ultra-heavy computational workloads */
    ultra: { gpuPercentage: 50, ramGB: 64, cpuCores: 16, storageGB: 500 } as ResourceQuota,
} as const;

/**
 * Economic tracking for team instances - runtime performance data
 */
export interface TeamEconomicTracking {
    /** Total number of chat instances spawned from this team */
    totalInstances: number;
    /** Total profit generated across all instances (stringified bigint) */
    totalProfit: string;
    /** Total costs incurred across all instances (stringified bigint) */
    totalCosts: string;
    /** Average KPI performance across all instances */
    averageKPIs: Record<string, number>;
    /** Current number of active instances */
    activeInstances: number;
    /** Last economic update timestamp */
    lastUpdated: number;
}

/**
 * Vertical package definition for industry-specific configurations
 */
export interface VerticalPackageConfig {
    /** Industry vertical identifier */
    industry: string;
    /** Compliance requirements for this vertical */
    complianceRequirements?: string[];
    /** Default workflows to enable for this vertical */
    defaultWorkflows?: string[];
    /** Data privacy level required */
    dataPrivacyLevel: "standard" | "hipaa" | "pci" | "sox" | "gdpr";
    /** Industry-specific terminology mappings */
    terminology?: Record<string, string>;
    /** Regulatory bodies to consider */
    regulatoryBodies?: string[];
}

/**
 * Security and resource isolation configuration
 */
export interface IsolationConfig {
    /** Level of process isolation */
    sandboxLevel: "none" | "user-namespace" | "full-container" | "vm";
    /** Network access policy */
    networkPolicy: "open" | "restricted" | "isolated" | "air-gapped";
    /** Allowed secret paths this team can access */
    secretsAccess: string[];
    /** Whether to enable audit logging for all actions */
    auditLogging?: boolean;
    /** Custom security policies */
    securityPolicies?: string[];
}

/**
 * Team organizational structure definition
 */
export interface TeamStructure {
    /** Type of organizational structure (e.g., "MOISE+", "FIPA ACL") */
    type: string;
    /** Version of the structure specification */
    version: string;
    /** The actual structure definition content */
    content: string;
}

/**
 * Represents all data that can be stored in a team's stringified config.
 * 
 * Teams serve as swarm configuration templates that can spawn multiple chat instances.
 * Following the minimal config pattern of bot.ts and chat.ts:
 * - Core identity and behavior defined through prompts
 * - Runtime intelligence through economic tracking and blackboard
 * - Algorithmic resource allocation instead of rigid quotas
 * - Emergent decision-making through AI rather than predefined rules
 */
export type TeamConfigObject = BaseConfigObject & {
    /**
     * Deployment context for the team
     * 
     * - development: Local development environment
     * - saas: SaaS deployment
     * - appliance: Appliance deployment
     */
    deploymentType: "development" | "saas" | "appliance";
    /** Primary objective of this team template */
    goal: string;
    /** Business behavior and economic guidance (prompt-driven like bots) */
    businessPrompt: string;
    /** Direct resource allocation for instances spawned from this team */
    resourceQuota: ResourceQuota;
    /** Target monthly profit across all team instances in micro-dollars (stringified bigint) */
    targetProfitPerMonth: string;
    /** Maximum monthly cost safety boundary in micro-dollars (stringified bigint) */
    costLimit?: string;
    /** Industry vertical package configuration (if applicable) */
    verticalPackage?: VerticalPackageConfig;
    /** Market segment targeting */
    marketSegment?: "enterprise" | "smb" | "consumer";
    /** Security and resource isolation configuration */
    isolation?: IsolationConfig;
    /** Model configuration for AI selection */
    modelConfig?: ModelConfig;
    /** Default limits for chat instances spawned from this team */
    defaultLimits?: {
        maxToolCallsPerBotResponse: number;
        maxToolCalls: number;
        maxCreditsPerBotResponse: string;
        maxCredits: string;
        maxDurationPerBotResponseMs: number;
        maxDurationMs: number;
        delayBetweenProcessingCyclesMs: number;
    };
    /** Default scheduling configuration for spawned instances */
    defaultScheduling?: {
        defaultDelayMs: number;
        toolSpecificDelays: Record<string, number>;
        requiresApprovalTools: string[] | "all" | "none";
        approvalTimeoutMs: number;
        autoRejectOnTimeout: boolean;
    };
    /** Default policy configuration for spawned instances */
    defaultPolicy?: {
        visibility: "open" | "restricted" | "private";
        acl?: string[];
    };
    /** Runtime performance data (like chat stats) */
    stats: TeamEconomicTracking;
    /** Team-specific blackboard for economic coordination and learning */
    blackboard?: BlackboardItem[];
    /** Organizational structure definition (MOISE+, FIPA ACL, etc.) */
    structure?: TeamStructure;
}

/**
 * Default economic tracking state for new teams
 */
function defaultEconomicTracking(): TeamEconomicTracking {
    return {
        totalInstances: 0,
        totalProfit: "0",
        totalCosts: "0",
        averageKPIs: {},
        activeInstances: 0,
        lastUpdated: Date.now(),
    };
}

/**
 * Default swarm configuration limits
 */
function defaultSwarmLimits() {
    return {
        maxToolCallsPerBotResponse: 10,
        maxToolCalls: 50,
        maxCreditsPerBotResponse: "1000000",
        maxCredits: "5000000",
        maxDurationPerBotResponseMs: 300000,
        maxDurationMs: 3600000,
        delayBetweenProcessingCyclesMs: 0,
    };
}

/**
 * Default swarm scheduling configuration
 */
function defaultSwarmScheduling() {
    return {
        defaultDelayMs: 0,
        toolSpecificDelays: {},
        requiresApprovalTools: "none" as const,
        approvalTimeoutMs: 600000,
        autoRejectOnTimeout: true,
    };
}

/**
 * Top-level Team config that encapsulates swarm template configuration.
 * 
 * Following the minimal config pattern of bot.ts and chat.ts:
 * - Behavior defined through prompts, not rigid schemas
 * - Runtime intelligence through stats and blackboard
 * - Algorithmic resource allocation
 * - Emergent decision-making through AI
 */
export class TeamConfig extends BaseConfig<TeamConfigObject> {
    deploymentType: TeamConfigObject["deploymentType"];
    goal: string;
    businessPrompt: string;
    resourceQuota: ResourceQuota;
    targetProfitPerMonth: string;
    costLimit?: string;
    verticalPackage?: VerticalPackageConfig;
    marketSegment?: TeamConfigObject["marketSegment"];
    isolation?: IsolationConfig;
    modelConfig?: ModelConfig;
    defaultLimits?: TeamConfigObject["defaultLimits"];
    defaultScheduling?: TeamConfigObject["defaultScheduling"];
    defaultPolicy?: TeamConfigObject["defaultPolicy"];
    stats: TeamEconomicTracking;
    blackboard?: BlackboardItem[];
    structure?: TeamStructure;

    constructor({ config }: { config: TeamConfigObject }) {
        super({ config });
        this.deploymentType = config.deploymentType;
        this.goal = config.goal;
        this.businessPrompt = config.businessPrompt;
        this.resourceQuota = config.resourceQuota;
        this.targetProfitPerMonth = config.targetProfitPerMonth;
        this.costLimit = config.costLimit;
        this.verticalPackage = config.verticalPackage;
        this.marketSegment = config.marketSegment;
        this.isolation = config.isolation;
        this.modelConfig = config.modelConfig;
        this.defaultLimits = config.defaultLimits;
        this.defaultScheduling = config.defaultScheduling;
        this.defaultPolicy = config.defaultPolicy;
        this.stats = config.stats!; // Will be set by parse method when needed
        this.blackboard = config.blackboard;
        this.structure = config.structure;
    }

    static parse(
        version: Pick<Team, "config">,
        logger: PassableLogger,
        opts?: { useFallbacks?: boolean },
    ): TeamConfig {
        return super.parseBase<TeamConfigObject, TeamConfig>(
            version.config,
            logger,
            ({ config }) => {
                if (opts?.useFallbacks ?? true) {
                    config.stats ??= defaultEconomicTracking();
                    config.defaultLimits ??= defaultSwarmLimits();
                    config.defaultScheduling ??= defaultSwarmScheduling();
                    config.blackboard ??= [];
                    config.structure ??= {
                        type: "MOISE+",
                        version: "1.0",
                        content: "",
                    };
                }
                return new TeamConfig({ config });
            },
        );
    }

    /**
     * Exports the config to a plain object
     */
    override export(): TeamConfigObject {
        const base = super.export();
        const result: TeamConfigObject = { ...base } as TeamConfigObject;
        
        // Only include defined fields
        if (this.deploymentType !== undefined) result.deploymentType = this.deploymentType;
        if (this.goal !== undefined) result.goal = this.goal;
        if (this.businessPrompt !== undefined) result.businessPrompt = this.businessPrompt;
        if (this.resourceQuota !== undefined) result.resourceQuota = this.resourceQuota;
        if (this.targetProfitPerMonth !== undefined) result.targetProfitPerMonth = this.targetProfitPerMonth;
        if (this.costLimit !== undefined) result.costLimit = this.costLimit;
        if (this.verticalPackage !== undefined) result.verticalPackage = this.verticalPackage;
        if (this.marketSegment !== undefined) result.marketSegment = this.marketSegment;
        if (this.isolation !== undefined) result.isolation = this.isolation;
        if (this.modelConfig !== undefined) result.modelConfig = this.modelConfig;
        if (this.defaultLimits !== undefined) result.defaultLimits = this.defaultLimits;
        if (this.defaultScheduling !== undefined) result.defaultScheduling = this.defaultScheduling;
        if (this.defaultPolicy !== undefined) result.defaultPolicy = this.defaultPolicy;
        if (this.stats !== undefined) result.stats = this.stats;
        if (this.blackboard !== undefined) result.blackboard = this.blackboard;
        if (this.structure !== undefined) result.structure = this.structure;
        
        return result;
    }

    /**
     * Sets the organizational structure for the team
     */
    setStructure(structure: TeamStructure | undefined): void {
        this.structure = structure;
    }

    /**
     * Updates economic tracking with new instance data
     */
    updateEconomicTracking(instanceData: {
        profit: string; // bigint as string
        costs: string;  // bigint as string
        kpis: Record<string, number>;
        isNewInstance?: boolean;
        isInstanceClosed?: boolean;
    }): void {
        const stats = this.stats;

        if (instanceData.isNewInstance) {
            stats.totalInstances += 1;
            stats.activeInstances += 1;
        }

        if (instanceData.isInstanceClosed) {
            stats.activeInstances = Math.max(0, stats.activeInstances - 1);
        }

        // Update profit and costs (bigint arithmetic)
        const currentProfit = BigInt(stats.totalProfit);
        const currentCosts = BigInt(stats.totalCosts);
        stats.totalProfit = (currentProfit + BigInt(instanceData.profit)).toString();
        stats.totalCosts = (currentCosts + BigInt(instanceData.costs)).toString();

        // Update average KPIs
        const totalInstances = stats.totalInstances;
        if (totalInstances > 0) {
            for (const [kpi, value] of Object.entries(instanceData.kpis)) {
                const currentAvg = stats.averageKPIs[kpi] || 0;
                stats.averageKPIs[kpi] = ((currentAvg * (totalInstances - 1)) + value) / totalInstances;
            }
        }

        stats.lastUpdated = Date.now();
    }

    /**
     * Adds insight to team blackboard for shared learning
     */
    addInsight(type: string, data: unknown, confidence: number): void {
        this.blackboard = this.blackboard || [];
        this.blackboard.push({
            id: `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
            value: { type, data, confidence },
            created_at: new Date().toISOString(),
        });

        // Keep blackboard manageable size (last 100 items)
        if (this.blackboard.length > 100) {
            this.blackboard = this.blackboard.slice(-100);
        }
    }

    /**
     * Gets insights from team blackboard of a specific type
     */
    getInsights(type?: string): BlackboardItem[] {
        if (!this.blackboard) return [];

        if (type) {
            return this.blackboard.filter(item =>
                typeof item.value === "object" &&
                item.value !== null &&
                "type" in item.value &&
                item.value.type === type,
            );
        }

        return this.blackboard;
    }

    /**
     * Gets current profit performance for decision making
     */
    getProfitPerformance(): {
        totalProfitUSD: number;
        totalCostsUSD: number;
        netProfitUSD: number;
        profitPerInstance: number;
        isAboveTarget: boolean;
        isWithinCostLimit: boolean;
        targetAchievementRatio: number;
    } {
        const totalProfit = Number(BigInt(this.stats.totalProfit)) / 1000000; // Convert from micro-dollars
        const totalCosts = Number(BigInt(this.stats.totalCosts)) / 1000000;
        const netProfit = totalProfit - totalCosts;
        const instances = this.stats.totalInstances;
        const targetProfitUSD = Number(BigInt(this.targetProfitPerMonth)) / 1000000;
        const costLimitUSD = this.costLimit ? Number(BigInt(this.costLimit)) / 1000000 : null;

        return {
            totalProfitUSD: totalProfit,
            totalCostsUSD: totalCosts,
            netProfitUSD: netProfit,
            profitPerInstance: instances > 0 ? netProfit / instances : 0,
            isAboveTarget: netProfit >= targetProfitUSD,
            isWithinCostLimit: costLimitUSD ? totalCosts <= costLimitUSD : true,
            targetAchievementRatio: targetProfitUSD > 0 ? netProfit / targetProfitUSD : 0,
        };
    }
}
