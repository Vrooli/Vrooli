/**
 * Tier 3: Execution Intelligence
 * 
 * This tier represents the culmination of Vrooli's execution intelligence,
 * where individual routine steps are executed with strategy-aware adaptation
 * that evolves based on routine characteristics, usage patterns, and performance metrics.
 * 
 * Key Components:
 * - UnifiedExecutor: Central execution coordinator with strategy selection
 * - Strategy Framework: Conversational, Reasoning, and Deterministic strategies
 * - Tool Orchestration: MCP-based tool integration and management
 * - Resource Management: Credit tracking and resource allocation
 * - Context Management: Runtime environment and cross-tier synchronization
 * - Validation Engine: Output validation and security enforcement
 * - Telemetry: Performance and safety event emission
 */

// Main executor
export { TierThreeExecutor } from "./TierThreeExecutor.js";

// Core executor
export { UnifiedExecutor } from "./engine/unifiedExecutor.js";
export type { UnifiedExecutorConfig } from "@vrooli/shared";

// Strategies
export { ConversationalStrategy } from "./strategies/conversationalStrategy.js";
export { ReasoningStrategy } from "./strategies/reasoningStrategy.js";
export { DeterministicStrategy } from "./strategies/deterministicStrategy.js";

// Engine components
export { StrategySelector, type UsageHints } from "./engine/strategySelector.js";
export { ResourceManager, type BudgetReservation } from "./engine/resourceManager.js";
export { IOProcessor } from "./engine/ioProcessor.js";
export { ToolOrchestrator, type MCPTool, type ToolApprovalStatus } from "./engine/toolOrchestrator.js";
export { ValidationEngine, type ValidationResult } from "./engine/validationEngine.js";
export { TelemetryShimAdapter as TelemetryShim } from "../monitoring/adapters/TelemetryShimAdapter.js";

// Context management
export { RunContext, RunContextFactory, type RunContextConfig, type UserData, type StepConfig } from "./context/runContext.js";
export { ContextExporter, type ExportedContext } from "./context/contextExporter.js";

// Re-export types from shared
export type {
    // Core execution types
    ExecutionContext,
    ExecutionStrategy,
    StrategyExecutionResult,
    StrategyType,
    ResourceUsage,
    AvailableResources,
    ExecutionConstraints,
    
    // Tool types
    ToolResource,
    ToolExecutionRequest,
    ToolExecutionResult,
    RetryPolicy,
    
    // Strategy types
    StrategyMetadata,
    StrategyFeedback,
    StrategyPerformance,
    StrategyEvolution,
} from "@vrooli/shared";
