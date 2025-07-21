/**
 * Core type definitions for Tier 2: Process Intelligence
 * These types define the universal routine orchestration capabilities
 */

import type { Location, RunConfig, RunProgress } from "../run/types.js";

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
    type: "sequential" | "bpmn";
    version: string;

    // Core navigation methods
    canNavigate(routine: unknown): boolean;
    getStartLocation(routine: unknown): Location;
    getAllStartLocations(routine: unknown): Location[]; // Support for multiple start locations
    getNextLocations(current: Location, context: Record<string, unknown>): Location[];
    isEndLocation(location: Location): boolean;

    // Event-driven navigation
    getLocationTriggers(location: Location): NavigationTrigger[];
    getLocationTimeouts(location: Location): NavigationTimeout[];
    canTriggerEvent(location: Location, event: NavigationEvent): boolean;

    // Metadata extraction
    getStepInfo(location: Location): StepInfo;
    getDependencies(location: Location): string[];
    getParallelBranches(location: Location): Location[][];
}

/**
 * Navigation trigger for event-driven execution
 */
export interface NavigationTrigger {
    id: string;
    type: "message" | "timer" | "signal" | "condition" | "webhook" | "custom";
    name?: string;
    config: Record<string, unknown>;
    targetLocation?: Location;
}

/**
 * Navigation timeout for time-based execution control
 */
export interface NavigationTimeout {
    id: string;
    duration: number; // milliseconds
    onTimeout: "continue" | "fail" | "retry" | "custom";
    targetLocation?: Location;
    config?: Record<string, unknown>;
}

/**
 * Navigation event for triggering flow transitions
 */
export interface NavigationEvent {
    id: string;
    type: "message" | "timer" | "signal" | "condition" | "webhook" | "custom";
    payload?: Record<string, unknown>;
    timestamp: Date;
    source?: string;
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
