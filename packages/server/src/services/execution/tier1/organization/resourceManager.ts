import {
    type ResourceAllocation,
    type ResourceType,
} from "@vrooli/shared";
import { type Logger } from "winston";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { SwarmResourceAdapter } from "../../cross-cutting/resources/adapters.js";
import { type ResourceAmount } from "../../cross-cutting/resources/resourceManager.js";
import { BaseTierResourceManager } from "../../shared/BaseTierResourceManager.js";

/**
 * Resource pool configuration
 */
export interface ResourcePool {
    type: ResourceType;
    total: number;
    available: number;
    reserved: number;
    unit: string;
}

/**
 * Resource request
 */
export interface ResourceRequest {
    swarmId: string;
    agentId?: string;
    type: ResourceType;
    amount: number;
    purpose: string;
    priority?: "low" | "medium" | "high";
    duration?: number; // milliseconds
}

/**
 * Resource status
 */
export interface ResourceStatus {
    pools: ResourcePool[];
    allocations: ResourceAllocation[];
    utilizationRate: number;
    projectedExhaustion?: Date;
}

/**
 * Allocation strategy
 */
export type AllocationStrategy = "fair" | "priority" | "performance" | "adaptive";


/**
 * ResourceManager - Tier 1 Swarm Resource Management
 * 
 * This component provides the foundational resource allocation and tracking infrastructure
 * for swarm-level operations in Vrooli's three-tier execution architecture. It serves as
 * the bridge between high-level resource requests and the unified resource management core.
 * 
 * ## Architecture Philosophy
 * 
 * The ResourceManager follows Vrooli's emergent AI philosophy by providing:
 * - **Deterministic Infrastructure**: Reliable resource allocation, tracking, and cleanup
 * - **Rich Event Streams**: Comprehensive resource events for AI agent analysis
 * - **Minimal Intelligence**: No hard-coded optimization algorithms
 * - **Emergent Optimization**: Complex resource optimization emerges from AI agents
 * 
 * ## Resource Types Managed
 * - **Credits**: Primary cost tracking mechanism across all operations
 * - **Time**: Execution duration limits and tracking (milliseconds)
 * - **Memory**: Peak usage tracking and allocation limits (bytes)
 * - **Tokens**: LLM token consumption tracking and rate limiting
 * - **API Calls**: External service rate limiting and usage tracking
 * 
 * ## Event-Driven Intelligence
 * 
 * The manager emits resource events that AI agents analyze to provide:
 * - Pattern recognition in resource usage
 * - Predictive allocation strategies
 * - Cost optimization recommendations
 * - Performance bottleneck detection
 * - Automated scaling decisions
 * 
 * ## Resource Flow Hierarchy
 * ```
 * Tier 1 (Swarm) → Allocates to swarms and agent teams
 * Tier 2 (Run)   → Distributes across routine executions
 * Tier 3 (Step)  → Manages individual step execution
 * ```
 * 
 * ## Emergent Optimization by Design
 * 
 * Resource optimization intelligence emerges from deployed AI agents that:
 * 1. Analyze resource event streams in real-time
 * 2. Learn usage patterns across swarms and teams
 * 3. Propose optimizations through routine execution
 * 4. Continuously improve recommendations based on outcomes
 * 
 * This approach ensures the system remains adaptable and can evolve optimization
 * strategies without requiring code changes to the core infrastructure.
 */
export class ResourceManager extends BaseTierResourceManager<SwarmResourceAdapter> {
    // Configuration constants remain for backward compatibility
    private readonly DEFAULT_TIME_LIMIT_MS = 3600000; // 1 hour
    private readonly EFFICIENCY_THRESHOLD = 0.2;
    private readonly ALLOCATION_RATIO = 0.3;
    private readonly HIGH_UTILIZATION_THRESHOLD = 0.8;
    private readonly UTILIZATION_PENALTY = 0.7;

    constructor(logger: Logger, eventBus: EventBus) {
        super(logger, eventBus, 1);
    }

    /**
     * Create the swarm resource adapter
     */
    protected createAdapter(): SwarmResourceAdapter {
        return new SwarmResourceAdapter(this.unifiedManager);
    }

    /**
     * Allocates resources for a swarm or agent
     * Maps legacy interface to unified manager
     */
    async allocateResources(
        swarmId: string,
        purpose: string,
        amount: number,
        type: ResourceType = "credits",
        agentId?: string,
    ): Promise<ResourceAllocation> {
        return this.withErrorHandling("allocate resources", async () => {
            const entityId = agentId || swarmId;
            const resourceAmount: ResourceAmount = {};

            // Map ResourceType to ResourceAmount
            switch (type) {
                case "credits":
                    resourceAmount.credits = amount;
                    break;
                case "time":
                    resourceAmount.time = amount;
                    break;
                case "memory":
                    resourceAmount.memory = amount;
                    break;
                case "tokens":
                    resourceAmount.tokens = amount;
                    break;
                case "api_calls":
                    resourceAmount.apiCalls = amount;
                    break;
                default:
                    throw new Error(`Unknown resource type: ${type}`);
            }

            // Use unified manager to allocate
            const allocation = await this.unifiedManager.allocate(
                entityId,
                agentId ? "team" : "swarm",
                resourceAmount,
            );

            // Map to legacy ResourceAllocation format
            return {
                id: allocation.id,
                swarmId,
                agentId: agentId || swarmId,
                resourceType: type,
                amount,
                purpose,
                expiresAt: allocation.expires || new Date(Date.now() + 3600000),
            };
        });
    }

    /**
     * Releases allocated resources
     * Maps legacy interface to unified manager
     */
    async releaseResources(allocationId: string): Promise<void> {
        await this.unifiedManager.release(allocationId);
    }

    /**
     * Gets resource status for a swarm
     * Maps legacy interface to unified manager
     */
    async getResourceStatus(swarmId: string): Promise<ResourceStatus> {
        // Get swarm summary using adapter
        const summary = await this.adapter.getSwarmSummary(swarmId);

        // Convert to legacy ResourcePool format
        const pools: ResourcePool[] = [
            {
                type: "credits",
                total: summary.allocated.credits || 0,
                available: summary.available.credits || 0,
                reserved: summary.used.credits || 0,
                unit: "credits",
            },
            {
                type: "time",
                total: summary.allocated.time || 0,
                available: summary.available.time || 0,
                reserved: summary.used.time || 0,
                unit: "milliseconds",
            },
            {
                type: "memory",
                total: summary.allocated.memory || 0,
                available: summary.available.memory || 0,
                reserved: summary.used.memory || 0,
                unit: "bytes",
            },
            {
                type: "tokens",
                total: summary.allocated.tokens || 0,
                available: summary.available.tokens || 0,
                reserved: summary.used.tokens || 0,
                unit: "tokens",
            },
            {
                type: "api_calls",
                total: summary.allocated.apiCalls || 0,
                available: summary.available.apiCalls || 0,
                reserved: summary.used.apiCalls || 0,
                unit: "calls",
            },
        ];

        // Get allocations and convert to legacy format
        const allocations = this.unifiedManager.getAllocations(swarmId).map(a => ({
            id: a.id,
            swarmId,
            agentId: a.entityId,
            resourceType: this.getResourceTypeFromAllocation(a),
            amount: this.getAmountFromAllocation(a),
            purpose: "Resource allocation",
            expiresAt: a.expires || new Date(Date.now() + 3600000),
        }));

        // Calculate utilization rate
        const totalAllocated = summary.allocated.credits || 1;
        const totalUsed = summary.used.credits || 0;
        const utilizationRate = totalUsed / totalAllocated;

        // Projected exhaustion emerges from monitoring agents
        return {
            pools,
            allocations,
            utilizationRate,
            projectedExhaustion: undefined, // Now determined by AI agents
        };
    }

    /**
     * Helper method to extract resource type from unified allocation
     */
    private getResourceTypeFromAllocation(allocation: any): ResourceType {
        const resources = allocation.resources;
        if (resources.credits > 0) return "credits";
        if (resources.time > 0) return "time";
        if (resources.memory > 0) return "memory";
        if (resources.tokens > 0) return "tokens";
        if (resources.apiCalls > 0) return "api_calls";
        return "credits"; // default
    }

    /**
     * Helper method to extract amount from unified allocation
     */
    private getAmountFromAllocation(allocation: any): number {
        const resources = allocation.resources;
        return resources.credits || resources.time || resources.memory ||
            resources.tokens || resources.apiCalls || 0;
    }

    /**
     * Requests resources with priority handling.
     * 
     * Priority optimization emerges from AI agents analyzing resource events
     * and proposing improved allocation strategies through routine execution.
     */
    async requestResources(
        request: ResourceRequest,
    ): Promise<ResourceAllocation> {
        // Priority optimization emerges from resource agents analyzing
        // allocation patterns and proposing improvements
        return this.allocateResources(
            request.swarmId,
            request.purpose,
            request.amount,
            request.type,
            request.agentId,
        );
    }

    /**
     * Track resource usage across swarms and agent teams.
     * 
     * Emits usage events that AI optimization agents analyze to identify
     * patterns, inefficiencies, and opportunities for improvement.
     */
    async trackResourceUsage(
        swarmId: string,
        usage: Partial<ResourceAmount>,
    ): Promise<void> {
        await this.unifiedManager.trackUsage(swarmId, usage);
    }

    /**
     * Clean up and shutdown
     */
    async shutdown(): Promise<void> {
        await this.cleanup();
    }

}
