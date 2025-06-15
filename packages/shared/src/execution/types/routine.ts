/**
 * Core type definitions for Tier 2: Process Intelligence
 * These types define the universal routine orchestration capabilities
 */

import type { ContextScope } from "./context.js";
import type { Location, RunConfig, RunProgress } from "../../run/types.js";

/**
 * Run lifecycle states
 * Maintains compatibility with existing RunStateMachine
 */
export enum RunState {
    UNINITIALIZED = "UNINITIALIZED",
    LOADING = "LOADING",
    CONFIGURING = "CONFIGURING",
    READY = "READY",
    RUNNING = "RUNNING",
    PAUSED = "PAUSED",
    SUSPENDED = "SUSPENDED",
    COMPLETED = "COMPLETED",
    FAILED = "FAILED",
    CANCELLED = "CANCELLED"
}

/**
 * Run event types for event-driven orchestration
 * Using enum for structured event names - renamed to avoid conflict with events.ts
 */
export enum RunEventTypeEnum {
    // Lifecycle events
    RUN_STARTED = "RUN_STARTED",
    RUN_PAUSED = "RUN_PAUSED",
    RUN_RESUMED = "RUN_RESUMED",
    RUN_COMPLETED = "RUN_COMPLETED",
    RUN_FAILED = "RUN_FAILED",
    RUN_CANCELLED = "RUN_CANCELLED",
    
    // Step events
    STEP_STARTED = "STEP_STARTED",
    STEP_COMPLETED = "STEP_COMPLETED",
    STEP_FAILED = "STEP_FAILED",
    STEP_SKIPPED = "STEP_SKIPPED",
    
    // Branch events
    BRANCH_CREATED = "BRANCH_CREATED",
    BRANCH_COMPLETED = "BRANCH_COMPLETED",
    BRANCH_FAILED = "BRANCH_FAILED",
    
    // Context events
    CONTEXT_UPDATED = "CONTEXT_UPDATED",
    VARIABLE_SET = "VARIABLE_SET",
    CHECKPOINT_CREATED = "CHECKPOINT_CREATED",
    
    // Performance events
    BOTTLENECK_DETECTED = "BOTTLENECK_DETECTED",
    OPTIMIZATION_APPLIED = "OPTIMIZATION_APPLIED"
}

/**
 * Base event interface for all run events
 */
export interface RunEvent {
    type: RunEventTypeEnum;
    timestamp: Date;
    runId: string;
    stepId?: string;
    metadata?: Record<string, unknown>;
}


/**
 * Step execution status
 */
export interface StepStatus {
    id: string;
    state: "pending" | "ready" | "running" | "completed" | "failed" | "skipped";
    startedAt?: Date;
    completedAt?: Date;
    error?: string;
    result?: unknown;
}

/**
 * Branch execution tracking
 */
export interface BranchExecution {
    id: string;
    parentStepId: string;
    steps: StepStatus[];
    state: "pending" | "running" | "completed" | "failed";
    parallel: boolean;
}


/**
 * Navigator interface for universal workflow support
 */
export interface Navigator {
    type: string; // "native" | "bpmn" | "langchain" | "temporal" | "custom"
    version: string;
    
    // Core navigation methods
    canNavigate(routine: unknown): boolean;
    getStartLocation(routine: unknown): Location;
    getNextLocations(current: Location, context: Record<string, unknown>): Location[];
    isEndLocation(location: Location): boolean;
    
    // Metadata extraction
    getStepInfo(location: Location): StepInfo;
    getDependencies(location: Location): string[];
    getParallelBranches(location: Location): Location[][];
}

/**
 * Step information
 */
export interface StepInfo {
    id: string;
    name: string;
    type: string;
    description?: string;
    inputs?: Record<string, unknown>;
    outputs?: Record<string, unknown>;
    config?: Record<string, unknown>;
}

/**
 * Routine definition (navigator-agnostic)
 */
export interface Routine {
    id: string;
    type: string; // Navigator type
    version: string;
    name: string;
    description?: string;
    definition: unknown; // Navigator-specific definition
    metadata: RoutineMetadata;
}

/**
 * Routine metadata
 */
export interface RoutineMetadata {
    author: string;
    createdAt: Date;
    updatedAt: Date;
    tags: string[];
    complexity: "simple" | "moderate" | "complex";
    estimatedDuration?: number;
}

/**
 * Execution run instance for routine orchestration
 * Represents internal execution state, not database records
 */
export interface ExecutionRun {
    id: string;
    routineId: string;
    state: RunState;
    config: RunConfig;
    progress: RunProgress;
    context: RunContext;
    startedAt?: Date;
    completedAt?: Date;
    error?: string;
}

/**
 * Run context for variable management
 */
export interface RunContext {
    variables: Record<string, unknown>;
    blackboard: Record<string, unknown>;
    scopes: ContextScope[];
}


/**
 * Checkpoint for recovery
 */
export interface Checkpoint {
    id: string;
    runId: string;
    timestamp: Date;
    state: RunState;
    progress: RunProgress;
    context: RunContext;
    size: number;
}

/**
 * Path optimization suggestion
 */
export interface RoutineOptimizationSuggestion {
    type: "parallelize" | "cache" | "skip" | "reorder";
    targetSteps: string[];
    expectedImprovement: number;
    rationale: string;
    risk: "low" | "medium" | "high";
}

/**
 * Performance analysis result
 */
export interface PerformanceAnalysis {
    bottlenecks: Array<{
        stepId: string;
        duration: number;
        resourceUsage: Record<string, number>;
    }>;
    suggestions: RoutineOptimizationSuggestion[];
    overallEfficiency: number;
}
