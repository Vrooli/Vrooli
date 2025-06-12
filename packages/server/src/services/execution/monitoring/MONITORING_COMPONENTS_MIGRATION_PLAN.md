# Monitoring Components Migration Plan

## Overview
This document outlines the migration plan for the remaining monitoring components to the UnifiedMonitoringService architecture.

## Components to Migrate

### 1. MetacognitiveMonitor (Tier 1)
- **Current Location**: `tier1/intelligence/metacognitiveMonitor.ts`
- **Adapter**: `MetacognitiveMonitorAdapter`
- **Used By**: tierOneFactory, tierOneCoordinator
- **Key Methods**:
  - `collectPerformanceData()`
  - `getPerformanceSnapshots()`
  - `recordDecisionOutcome()`

### 2. PerformanceMonitor (Tier 2)
- **Current Location**: `tier2/intelligence/performanceMonitor.ts`
- **Adapter**: `PerformanceMonitorAdapter`
- **Used By**: tierTwoOrchestrator
- **Key Methods**:
  - `recordMetric()`
  - `recordStepTiming()`
  - `recordResourceUsage()`
  - `getRunMetrics()`
  - `requestPerformanceAnalysis()`

### 3. PerformanceTracker (Tier 3)
- **Current Location**: `tier3/strategies/shared/performanceTracker.ts`
- **Adapter**: `PerformanceTrackerAdapter`
- **Used By**: strategyBase
- **Key Methods**:
  - `recordPerformance()`
  - `getMetrics()`
  - `generateFeedback()`
  - `getRecentTrend()`

### 4. StrategyMetricsStore (Tier 3)
- **Current Location**: `tier3/metrics/strategyMetricsStore.ts`
- **Adapter**: `StrategyMetricsStoreAdapter`
- **Used By**: reasoningStrategy, strategyMetricsExample
- **Key Methods**:
  - `recordExecution()`
  - `getAggregatedMetrics()`
  - `getExecutionsByStepType()`
  - `getRecentExecutions()`
  - `exportMetrics() / importMetrics()`

### 5. ResourceMetrics (Cross-cutting)
- **Current Location**: `cross-cutting/resources/resourceMetrics.ts`
- **Adapter**: `ResourceMetricsAdapter`
- **Used By**: ResourceMonitor
- **Key Methods**:
  - `recordMetric()`
  - `recordEfficiency()`
  - `recordHealth()`
  - `recordExecution()`
  - `getCostAnalysis()`
  - `detectBottlenecks()`

### 6. ResourceMonitor (Integration)
- **Current Location**: `integration/monitoring/resourceMonitor.ts`
- **Adapter**: `ResourceMonitorAdapter`
- **Used By**: executionArchitecture
- **Key Methods**:
  - `getUsageReport()`
  - `getOptimizationSuggestions()`
  - `getUtilizationSummary()`

## Migration Strategy

### Phase 1: Create Base Adapter Class
Create a shared base class for all monitoring adapters to reduce code duplication.

### Phase 2: Implement Individual Adapters
Create adapter for each component in order of dependency:
1. ResourceMetrics (no dependencies)
2. PerformanceTracker (no dependencies)
3. StrategyMetricsStore (no dependencies)
4. MetacognitiveMonitor (depends on EventBus)
5. PerformanceMonitor (depends on EventBus)
6. ResourceMonitor (depends on ResourceMetrics)

### Phase 3: Update Imports
Update all files that import these components to use the adapters.

### Phase 4: Testing
Create comprehensive tests for each adapter to ensure backward compatibility.

### Phase 5: Cleanup
Mark original files as legacy with deprecation notices.

## Benefits
- **Unified Storage**: All monitoring data in one place
- **Consistent Schema**: UnifiedMetric format across all components
- **Cross-Tier Analytics**: Enable holistic system analysis
- **MCP Tool Integration**: Ready for AI agent monitoring
- **Performance Optimization**: Shared infrastructure reduces overhead

## Success Criteria
- All adapters maintain 100% backward compatibility
- All existing tests continue to pass
- Performance overhead remains <5ms
- Event bus integration preserved
- Zero downtime migration