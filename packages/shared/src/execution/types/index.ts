/**
 * Central export point for all execution types
 * Re-exports all type definitions for easy importing
 */

// Core execution types (canonical source for shared interfaces)
export * from "./core.js";
export * from "./communication.js";

// Tier 1: Coordination Intelligence (excluding conflicting ResourceAllocation)
export {
    SwarmState,
    SwarmEventType,
    type SwarmEvent,
    type AgentRole,
    type AgentNorm,
    type SwarmAgent,
    type AgentPerformance,
    type SwarmTeam,
    type TeamHierarchy,
    type SwarmConfiguration,
    type SwarmMetadata,
    type ReasoningResult,
    type StrategySelectionResult,
    type SwarmConfig,
    type AgentCapability,
    type TeamConstraints,
    type ConsensusRequest,
    type ConsensusResult,
    type SwarmDecision,
    type SwarmPerformance,
    type MetacognitiveReflection,
    type TeamFormation,
    type SwarmResourceAllocation,
    type ChildSwarmReservation,
    type Swarm
} from "./swarm.js";

// Tier 2: Process Intelligence
export * from "./routine.js";

// Tier 3: Execution Intelligence (excluding core conflicts)
export {
    StrategyType,
    type StrategyExecutionResult,
    type StrategyMetadata,
    type StrategyResourceUsage,
    type StrategyFeedback,
    type ExecutionStrategy,
    type StrategyPerformance,
    type StrategyEvolution,
    type ToolExecutionRequest,
    type ToolExecutionResult,
    type StrategyFactoryConfig,
    type UnifiedExecutorConfig
} from "./strategies.js";

// Shared types (excluding ExecutionContext which conflicts with core)
export {
    type ContextScope,
    type BaseContext,
    type ContextMetadata,
    type CoordinationContext,
    type SharedMemory,
    type DecisionRecord,
    type ConsensusRecord,
    type ConflictRecord,
    type CoordinationState,
    type ProcessContext,
    type NavigationState,
    type BranchState,
    type ProcessMemory,
    type PerformanceData,
    type OrchestrationState,
    type ExecutionMemory,
    type ToolCallRecord,
    type LearningData,
    type Pattern,
    type FeedbackRecord,
    type Adaptation,
    type AdaptationState,
    type CrossTierContext,
    type ContextConstraints,
    type ContextSnapshot
} from "./context.js";

export * from "./events.js";
export * from "./security.js";
export * from "./resilience.js";

export {
    ResourceType as ResourceTypeEnum,
    ResourceUnit,
    type Resource,
    type ResourceCost,
    type ResourceRequest,
    AllocationPriority,
    type AllocationResult,
    type AllocatedResource,
    type DeniedResource,
    type ResourceAllocationUsage,
    type ResourceLimitConfig,
    LimitScope,
    type ResourceLimit,
    type LimitPeriod,
    type LimitEnforcement,
    type LimitNotification,
    type OptimizationSuggestion,
    type ResourcePool,
    type SharingPolicy,
    type PoolMember,
    type PoolStatistics,
    type ResourceConflict,
    type ConflictResolution,
    type ResourceAccounting,
    type ResourceUsageSummary,
    type ResourceCostSummary,
    type EfficiencyMetrics,
    type BudgetReservation,
    type ResourceReservation,
    type ResourceReturn
} from "./resources.js";

export * from "./llm.js";

// Monitoring types (excluding TimeRange which conflicts with events)
export {
    MonitoringEventPrefix,
    type MonitoringEvent,
    type MonitoringMetrics,
    type PercentileMetrics,
    type ResourceMetrics,
    type SamplingConfig,
    type SamplingCondition,
    type PerformanceEvent,
    type PerformancePayload,
    type ExecutionTimingPayload,
    type ResourceUtilizationPayload,
    type ThroughputPayload,
    type LatencyPayload,
    type CachePerformancePayload,
    type HealthEvent,
    type HealthPayload,
    type ComponentHealthPayload,
    type HealthCheck,
    type SystemHealthPayload,
    type ComponentStatus,
    type HealthAlert,
    type DependencyHealthPayload,
    type CircuitBreakerPayload,
    type BusinessEvent,
    type BusinessPayload,
    type TaskCompletionPayload,
    type GoalProgressPayload,
    type MilestoneStatus,
    type StrategyEffectivenessPayload,
    type UserInteractionPayload,
    type CostTrackingPayload,
    type SafetyEvent,
    type SafetyPayload,
    type ValidationErrorPayload,
    type ValidationError as MonitoringValidationError,
    type SecurityIncidentPayload,
    type PIIDetectionPayload,
    type ErrorOccurredPayload,
    type AnomalyDetectedPayload,
    type TierMetrics,
    type TierSpecificMetrics,
    type Tier1Metrics,
    type Tier2Metrics,
    type Tier3Metrics,
    type TelemetryConfig,
    type PerformanceGuards,
    type RollingHistoryConfig,
    type HistoryEntry,
    type HistoryQuery,
    type TimeRange as MonitoringTimeRange,
    type HistoryAggregation,
    type AggregationStats
} from "./monitoring.js";
