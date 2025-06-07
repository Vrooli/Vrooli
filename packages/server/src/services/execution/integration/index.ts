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

// Export resource management
export { UnifiedResourceManager } from './resources/index.js';
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
} from './resources/index.js';

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