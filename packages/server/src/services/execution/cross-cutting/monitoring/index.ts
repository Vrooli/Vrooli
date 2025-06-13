/**
 * Cross-cutting monitoring services
 * 
 * This module exports MINIMAL monitoring infrastructure.
 * All monitoring intelligence emerges from agents, not from hardcoded logic.
 * 
 * DEPRECATED: RollingHistory and related monitoring logic will be removed.
 * Use ExecutionEventEmitter to emit raw events for monitoring agents.
 */

// New minimal monitoring approach
export { ExecutionEventEmitter, ComponentEventEmitter } from "./ExecutionEventEmitter.js";
export type { RawMetricEvent, RawExecutionEvent } from "./ExecutionEventEmitter.js";

// DEPRECATED - to be removed
export { RollingHistoryAdapter as RollingHistory } from "../../monitoring/adapters/RollingHistoryAdapter.js";
export type { HistoryEvent, RollingHistoryConfig, PatternDetectionResult, HistoryStats } from "../../monitoring/adapters/RollingHistoryAdapter.js";