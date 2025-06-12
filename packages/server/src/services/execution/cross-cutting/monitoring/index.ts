/**
 * Cross-cutting monitoring services
 * 
 * This module exports monitoring capabilities that can be used across all tiers.
 * NOTE: RollingHistory now points to the adapter for unified monitoring.
 */

export { RollingHistoryAdapter as RollingHistory } from "../../monitoring/adapters/RollingHistoryAdapter.js";
export type { HistoryEvent, RollingHistoryConfig, PatternDetectionResult, HistoryStats } from "../../monitoring/adapters/RollingHistoryAdapter.js";