/**
 * Shared utilities for execution strategies
 * Extracted common patterns to reduce code duplication
 */

export { PerformanceTrackerAdapter as PerformanceTracker } from '../../../monitoring/adapters/PerformanceTrackerAdapter.js';
export type { 
    PerformanceEntry, 
    PerformanceMetrics, 
    PerformanceFeedback 
} from '../../../monitoring/adapters/PerformanceTrackerAdapter.js';

export { StrategyBase } from './strategyBase.js';
export type { 
    StrategyConfig, 
    ExecutionMetadata 
} from './strategyBase.js';