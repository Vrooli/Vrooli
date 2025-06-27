/**
 * Shared utilities for execution strategies
 * Extracted common patterns to reduce code duplication
 */

// Note: StrategyBase is actually MinimalStrategyBase - it has NO performance tracking
export { MinimalStrategyBase, MinimalStrategyBase as StrategyBase } from "./strategyBase.js";
export type { 
    MinimalStrategyConfig as StrategyConfig,
    MinimalExecutionMetadata as ExecutionMetadata, 
} from "./strategyBase.js";
