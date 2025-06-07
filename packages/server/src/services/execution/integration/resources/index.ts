/**
 * Resource management integration module
 * 
 * Provides unified resource management across all execution tiers
 */

export { UnifiedResourceManager } from "./unifiedResourceManager.js";

// Re-export resource types from shared
export type {
    ResourceType,
    ResourceUnit,
    Resource,
    ResourceCost,
    ResourceAllocation,
    ResourceRequest,
    AllocationPriority,
    AllocationResult,
    AllocatedResource,
    DeniedResource,
    ResourceUsage,
    ResourceLimitConfig,
    LimitScope,
    ResourceLimit,
    LimitPeriod,
    LimitEnforcement,
    LimitNotification,
    OptimizationSuggestion,
    ResourcePool,
    SharingPolicy,
    PoolMember,
    PoolStatistics,
    ResourceConflict,
    ConflictResolution,
    ResourceAccounting,
    ResourceUsageSummary,
    ResourceCostSummary,
    EfficiencyMetrics,
} from "@vrooli/shared";