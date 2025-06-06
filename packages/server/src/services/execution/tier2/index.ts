/**
 * Tier 2: Process Intelligence
 * 
 * This tier handles routine execution and workflow management. It provides
 * universal navigation across different workflow formats, intelligent
 * branch coordination, and performance optimization.
 */

// Core orchestration
export { RunStateMachine } from "./orchestration/runStateMachine.js";
export { BranchCoordinator } from "./orchestration/branchCoordinator.js";
export { StepExecutor } from "./orchestration/stepExecutor.js";

// Context management
export { ContextManager } from "./context/contextManager.js";

// Navigation
export { NavigatorRegistry } from "./navigation/navigatorRegistry.js";
export { NativeNavigator } from "./navigation/navigators/nativeNavigator.js";

// Intelligence components
export { PerformanceMonitor } from "./intelligence/performanceMonitor.js";
export { PathOptimizer } from "./intelligence/pathOptimizer.js";

// Persistence and validation
export { CheckpointManager } from "./persistence/checkpointManager.js";
export { MOISEGate } from "./validation/moiseGate.js";

// State management
export { type IRunStateStore } from "./state/runStateStore.js";

// Re-export types
export type {
    RunInitParams,
    RunState,
    RunPhase,
    NavigationResult,
    BranchState,
    PerformanceMetrics,
    OptimizationSuggestion,
    CheckpointData,
    MOISERole,
    MOISENorm,
} from "@vrooli/shared";