/**
 * Monitoring adapters for migrating legacy components to UnifiedMonitoringService
 */

// Base adapter
export * from "./BaseMonitoringAdapter.js";

// Tier-specific adapters
export * from "./MetacognitiveMonitorAdapter.js";  // Tier 1
export * from "./PerformanceMonitorAdapter.js";     // Tier 2
export * from "./PerformanceTrackerAdapter.js";     // Tier 3
export * from "./StrategyMetricsStoreAdapter.js";   // Tier 3

// Cross-cutting adapters
export * from "./ResourceMetricsAdapter.js";
export * from "./ResourceMonitorAdapter.js";
export * from "./RollingHistoryAdapter.js";
export * from "./TelemetryShimAdapter.js";