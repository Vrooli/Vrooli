/**
 * Integration module for the three-tier execution architecture
 * 
 * This module provides the central ExecutionArchitecture factory
 * that wires together all three tiers of Vrooli's execution system.
 */

export {
    ExecutionArchitecture,
    getExecutionArchitecture,
    resetExecutionArchitecture,
    type ExecutionArchitectureOptions,
} from './executionArchitecture.js';

// Export resource monitoring
export { ResourceMonitor } from './monitoring/resourceMonitor.js';

// Re-export resource types from shared (no longer from deprecated resources folder)
export type {
    ResourceType,
    ResourceUnit,
    Resource,
    ResourceAllocation,
    ResourceUsage,
    ResourceLimitConfig,
    AllocationResult,
    OptimizationSuggestion,
    ResourceAccounting,
} from '@vrooli/shared';

// Export monitoring tools
export {
    MonitoringTools,
    MonitoringUtils,
    MONITORING_TOOL_DEFINITIONS,
    createMonitoringToolInstances,
    registerMonitoringTools,
} from './tools/index.js';
export type {
    QueryMetricsParams,
    AnalyzeHistoryParams,
    AggregateDataParams,
    PublishReportParams,
    DetectAnomaliesParams,
    CalculateSLOParams,
    TimeSeriesPoint,
    StatisticalSummary,
    AnomalyResult,
    PatternResult,
} from './tools/index.js';