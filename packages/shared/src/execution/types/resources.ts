/**
 * Core type definitions for resource management
 * These types define resource allocation and tracking across all tiers
 */

/**
 * Resource types
 */
export enum ResourceType {
    COMPUTE = "COMPUTE",
    MEMORY = "MEMORY",
    STORAGE = "STORAGE",
    NETWORK = "NETWORK",
    API_CALLS = "API_CALLS",
    TOKENS = "TOKENS",
    CREDITS = "CREDITS",
    TIME = "TIME"
}

/**
 * Resource unit definitions
 */
export enum ResourceUnit {
    // Compute
    CPU_SECONDS = "CPU_SECONDS",
    GPU_SECONDS = "GPU_SECONDS",
    
    // Memory
    BYTES = "BYTES",
    KILOBYTES = "KILOBYTES",
    MEGABYTES = "MEGABYTES",
    GIGABYTES = "GIGABYTES",
    
    // API & Tokens
    CALLS = "CALLS",
    TOKENS = "TOKENS",
    CREDITS = "CREDITS",
    
    // Time
    MILLISECONDS = "MILLISECONDS",
    SECONDS = "SECONDS",
    MINUTES = "MINUTES"
}

/**
 * Resource definition
 */
export interface Resource {
    id: string;
    type: ResourceType;
    name: string;
    description?: string;
    unit: ResourceUnit;
    available: number;
    allocated: number;
    reserved: number;
    limit?: number;
    cost?: ResourceCost;
}

/**
 * Resource cost configuration
 */
export interface ResourceCost {
    amount: number;
    currency: string;
    per: number; // Per X units
    billingMode: "prepaid" | "postpaid" | "credits";
}

/**
 * Resource allocation request
 */
export interface ResourceAllocation {
    id: string;
    requesterId: string;
    requesterType: "user" | "swarm" | "run" | "step";
    resources: ResourceRequest[];
    priority: AllocationPriority;
    duration?: number; // Expected duration in seconds
    metadata: Record<string, unknown>;
}

/**
 * Resource request
 */
export interface ResourceRequest {
    resourceId: string;
    amount: number;
    required: boolean; // If false, this is a "nice to have"
    alternatives?: string[]; // Alternative resource IDs
}

/**
 * Allocation priority
 */
export enum AllocationPriority {
    LOW = "LOW",
    NORMAL = "NORMAL",
    HIGH = "HIGH",
    CRITICAL = "CRITICAL"
}

/**
 * Resource allocation result
 */
export interface AllocationResult {
    allocationId: string;
    status: "allocated" | "partial" | "denied" | "queued";
    allocatedResources: AllocatedResource[];
    deniedResources?: DeniedResource[];
    expiresAt?: Date;
    token?: string; // Token for releasing resources
}

/**
 * Allocated resource
 */
export interface AllocatedResource {
    resourceId: string;
    amount: number;
    actualCost?: number;
    metadata?: Record<string, unknown>;
}

/**
 * Denied resource
 */
export interface DeniedResource {
    resourceId: string;
    requestedAmount: number;
    availableAmount: number;
    reason: string;
    alternativeSuggested?: string;
}

/**
 * Resource usage tracking
 */
export interface ResourceUsage {
    id: string;
    allocationId: string;
    resourceId: string;
    startTime: Date;
    endTime?: Date;
    consumed: number;
    remaining: number;
    cost: number;
    metadata: Record<string, unknown>;
}

/**
 * Resource limit configuration
 */
export interface ResourceLimitConfig {
    id: string;
    scope: LimitScope;
    scopeId: string;
    limits: ResourceLimit[];
    enforcement: LimitEnforcement;
    notifications: LimitNotification[];
}

/**
 * Limit scope
 */
export enum LimitScope {
    USER = "USER",
    SESSION = "SESSION",
    SWARM = "SWARM",
    RUN = "RUN",
    STEP = "STEP",
    GLOBAL = "GLOBAL"
}

/**
 * Resource limit
 */
export interface ResourceLimit {
    resourceType: ResourceType;
    limit: number;
    period?: LimitPeriod;
    rollover: boolean;
}

/**
 * Limit period
 */
export interface LimitPeriod {
    unit: "minute" | "hour" | "day" | "week" | "month";
    value: number;
}

/**
 * Limit enforcement configuration
 */
export interface LimitEnforcement {
    mode: "soft" | "hard";
    gracePeriod?: number; // Seconds
    overageAllowed: boolean;
    overageMultiplier?: number; // Cost multiplier for overage
}

/**
 * Limit notification configuration
 */
export interface LimitNotification {
    threshold: number; // Percentage of limit
    channel: "email" | "webhook" | "event";
    recipients: string[];
    message?: string;
}

/**
 * Resource optimization suggestion
 */
export interface OptimizationSuggestion {
    id: string;
    type: "reduce" | "substitute" | "batch" | "cache" | "schedule";
    targetResource: string;
    currentUsage: number;
    projectedSavings: number;
    implementation: string;
    risk: "low" | "medium" | "high";
}

/**
 * Resource pool for shared resources
 */
export interface ResourcePool {
    id: string;
    name: string;
    resources: string[]; // Resource IDs
    sharingPolicy: SharingPolicy;
    members: PoolMember[];
    statistics: PoolStatistics;
}

/**
 * Sharing policy
 */
export interface SharingPolicy {
    type: "equal" | "priority" | "quota" | "fair";
    config: Record<string, unknown>;
}

/**
 * Pool member
 */
export interface PoolMember {
    id: string;
    type: "user" | "team" | "organization";
    quota?: number;
    priority?: number;
    usage: number;
}

/**
 * Pool statistics
 */
export interface PoolStatistics {
    totalCapacity: number;
    currentUsage: number;
    peakUsage: number;
    averageUtilization: number;
    contentionRate: number;
}

/**
 * Resource conflict
 */
export interface ResourceConflict {
    id: string;
    timestamp: Date;
    type: "contention" | "deadlock" | "starvation";
    resources: string[];
    requesters: string[];
    resolution?: ConflictResolution;
    resolved: boolean;
}

/**
 * Conflict resolution
 */
export interface ConflictResolution {
    type: "priority" | "preemption" | "sharing" | "queuing" | "negotiation";
    winner?: string;
    compromise?: Record<string, number>;
    reason: string;
}

/**
 * Resource accounting
 */
export interface ResourceAccounting {
    period: {
        start: Date;
        end: Date;
    };
    usage: ResourceUsageSummary[];
    costs: ResourceCostSummary[];
    efficiency: EfficiencyMetrics;
}

/**
 * Resource usage summary
 */
export interface ResourceUsageSummary {
    resourceType: ResourceType;
    totalConsumed: number;
    peakUsage: number;
    averageUsage: number;
    utilizationRate: number;
}

/**
 * Resource cost summary
 */
export interface ResourceCostSummary {
    resourceType: ResourceType;
    totalCost: number;
    breakdown: Array<{
        category: string;
        amount: number;
        percentage: number;
    }>;
}

/**
 * Efficiency metrics
 */
export interface EfficiencyMetrics {
    overallEfficiency: number; // 0-1
    wastedResources: number; // Percentage
    optimizationPotential: number; // Percentage
    recommendations: OptimizationSuggestion[];
}
