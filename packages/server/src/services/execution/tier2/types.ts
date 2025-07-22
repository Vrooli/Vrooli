/**
 * Types for RunContextManager - Tier 2 Resource & Context Management
 * 
 * This module defines the types used by RunContextManager for proper
 * resource hierarchy management (Swarm → Run → Step).
 */

import { type BranchExecution } from "@vrooli/shared";

/**
 * Resource limits for run/step allocations
 */
export interface RunResourceLimits {
    credits: string;
    timeoutMs: number;
    memoryMB: number;
    concurrentExecutions: number;
}

/**
 * Run-level resource allocation request
 * 
 * Represents a request to allocate resources from a swarm for a specific run.
 */
export interface RunAllocationRequest {
    runId: string;
    routineId: string;
    estimatedRequirements: {
        credits: string;
        durationMs: number;
        memoryMB: number;
        maxSteps: number;
    };
    priority: "low" | "medium" | "high";
    purpose: string;
}

/**
 * Allocated resources for a run
 * 
 * Represents the actual resource allocation for a run, including tracking
 * of remaining resources as they are consumed.
 */
export interface RunAllocation {
    allocationId: string;
    runId: string;
    swarmId: string;
    allocated: RunResourceLimits;
    remaining: RunResourceLimits;
    allocatedAt: Date;
    expiresAt: Date;
}

/**
 * Step-level resource allocation request
 * 
 * Represents a request to sub-allocate resources from a run for a specific step.
 */
export interface StepAllocationRequest {
    stepId: string;
    stepType: "llm_call" | "tool_execution" | "data_processing";
    estimatedRequirements: {
        credits: string;
        durationMs: number;
        memoryMB: number;
    };
}

/**
 * Allocated resources for a step
 * 
 * Represents the actual resource allocation for a step execution.
 */
export interface StepAllocation {
    allocationId: string;
    stepId: string;
    runId: string;
    allocated: RunResourceLimits;
    allocatedAt: Date;
    expiresAt: Date;
}

/**
 * INavigator - Pure navigation interface
 * 
 * Navigators are pure functions that take routine structure + current position
 * and return navigation information. They do NOT manage state, cache data,
 * or handle events - that's the responsibility of other components.
 */
export interface INavigator {
    readonly type: string;
    readonly version: string;

    /**
     * Check if this navigator can handle the routine structure
     */
    canNavigate(routine: unknown): boolean;

    /**
     * Get the starting location(s) for this routine
     */
    getStartLocation(routine: unknown): Location;

    /**
     * Get next possible locations from current position
     * Pure function - no side effects or state changes
     */
    getNextLocations(
        routine: unknown,
        current: Location,
        context?: Record<string, unknown>
    ): Location[];

    /**
     * Check if location is a terminal/end location
     */
    isEndLocation(routine: unknown, location: Location): boolean;

    /**
     * Get step information for execution at this location
     */
    getStepInfo(routine: unknown, location: Location): StepInfo;
}

/**
 * Supporting types for simplified navigation
 */
export interface Location {
    id: string;
    routineId: string;
    nodeId: string;
}

export interface StepInfo {
    id: string;
    name: string;
    type: string;
    description?: string;
    config?: Record<string, unknown>;
}

/**
 * Run configuration
 */
export interface RunConfig {
    id: string;
    routineId: string;
    userId: string;
    inputs: Record<string, unknown>;
    metadata: Record<string, unknown>;
    createdAt: Date;
    updatedAt: Date;
}

/**
 * Enhanced execution context for BPMN and complex workflow support
 * 
 * Extends basic variable tracking to include comprehensive execution state:
 * events, parallel branches, subprocesses, gateways, and external integrations.
 */
export interface EnhancedExecutionContext {
    // Current variables (unchanged from basic context)
    variables: Record<string, unknown>;
    
    // Event management system
    events: {
        // Currently monitoring boundary events (timers, errors, signals, etc.)
        active: BoundaryEvent[];
        // Intermediate events waiting to be triggered
        pending: IntermediateEvent[];
        // Recently fired events with their payloads
        fired: EventInstance[];
        // Active timer events with expiration tracking
        timers: TimerEvent[];
    };
    
    // Parallel execution tracking
    parallelExecution: {
        // Currently executing parallel branches
        activeBranches: ParallelBranch[];
        // Completed branch IDs for join synchronization
        completedBranches: string[];
        // Gateway join points waiting for synchronization
        joinPoints: JoinPoint[];
    };
    
    // Subprocess management
    subprocesses: {
        // Stack of nested subprocess contexts
        stack: SubprocessContext[];
        // Active event subprocesses (interrupting/non-interrupting)
        eventSubprocesses: EventSubprocess[];
    };
    
    // External integration events
    external: {
        // Pending message event correlations
        messageEvents: MessageEvent[];
        // External webhook trigger events
        webhookEvents: WebhookEvent[];
        // Signal propagation events
        signalEvents: SignalEvent[];
    };
    
    // Gateway execution state
    gateways: {
        // Inclusive gateway multi-path activation states
        inclusiveStates: InclusiveGatewayState[];
        // Complex gateway custom logic states
        complexConditions: ComplexGatewayState[];
    };
}

// Supporting types for enhanced context

export interface BoundaryEvent {
    id: string;
    type: "timer" | "error" | "message" | "signal" | "compensation";
    attachedToRef: string; // ID of the activity this event is attached to
    interrupting: boolean;
    config: Record<string, unknown>;
    activatedAt: Date;
}

export interface IntermediateEvent {
    id: string;
    type: "message" | "timer" | "signal" | "link" | "conditional";
    eventDefinition: Record<string, unknown>;
    waiting: boolean;
    correlationKey?: string;
}

export interface EventInstance {
    id: string;
    eventId: string;
    type: string;
    payload: Record<string, unknown>;
    firedAt: Date;
    source?: string;
}

export interface TimerEvent {
    id: string;
    eventId: string;
    duration?: number; // milliseconds
    dueDate?: Date;
    cycle?: string; // ISO 8601 duration for repeating timers
    expiresAt: Date;
    attachedToRef?: string; // For boundary events
}

export interface ParallelBranch {
    id: string;
    branchId: string;
    currentLocation: Location;
    status: "pending" | "running" | "completed" | "failed";
    startedAt: Date;
    completedAt?: Date;
    result?: unknown;
}

export interface JoinPoint {
    id: string;
    gatewayId: string;
    requiredBranches: string[];
    completedBranches: string[];
    isReady: boolean;
}

export interface SubprocessContext {
    id: string;
    subprocessId: string;
    parentLocation: Location;
    variables: Record<string, unknown>;
    startedAt: Date;
    status: "running" | "completed" | "failed";
}

export interface EventSubprocess {
    id: string;
    subprocessId: string;
    triggerEvent: string;
    interrupting: boolean;
    status: "monitoring" | "active" | "completed";
    startedAt?: Date;
}

export interface MessageEvent {
    id: string;
    messageRef: string;
    correlationKey: string;
    expectedPayload?: Record<string, unknown>;
    receivedAt?: Date;
    payload?: Record<string, unknown>;
}

export interface WebhookEvent {
    id: string;
    webhookId: string;
    url: string;
    method: string;
    payload: Record<string, unknown>;
    receivedAt: Date;
}

export interface SignalEvent {
    id: string;
    signalRef: string;
    scope: "global" | "process" | "local";
    payload?: Record<string, unknown>;
    propagatedAt: Date;
}

export interface InclusiveGatewayState {
    id: string;
    gatewayId: string;
    evaluatedConditions: Array<{
        conditionId: string;
        expression: string;
        result: boolean;
        evaluatedAt: Date;
    }>;
    activatedPaths: string[];
}

export interface ComplexGatewayState {
    id: string;
    gatewayId: string;
    customLogic: string;
    state: Record<string, unknown>;
    lastEvaluatedAt: Date;
}

/**
 * Abstract location types for complex execution states
 */
export type LocationType = 
    | "node"                    // Simple BPMN node
    | "boundary_event_monitor"  // Monitoring a boundary event
    | "parallel_branch"         // Executing in parallel branch
    | "subprocess_context"      // Inside a subprocess
    | "event_waiting"          // Waiting for intermediate event
    | "gateway_evaluation"     // Evaluating gateway conditions
    | "timer_waiting"          // Waiting for timer event
    | "message_waiting"        // Waiting for message correlation
    | "signal_waiting"         // Waiting for signal event
    | "compensation_active"    // Compensation handler active
    | "multi_instance_execution" // Executing multi-instance activity
    | "multi_instance_waiting"   // Waiting for multi-instance completion
    | "loop_execution";         // Executing loop iteration

export interface AbstractLocation extends Location {
    locationType: LocationType;
    parentNodeId?: string;      // Original BPMN node this state derives from
    branchId?: string;          // For parallel execution
    subprocessId?: string;      // For subprocess execution
    eventId?: string;           // For event-related states
    metadata?: Record<string, unknown>; // Additional state-specific data
}

/**
 * Context transformation utilities
 */
export interface ContextTransformer {
    /**
     * Convert basic context to enhanced context
     */
    enhance(basicContext: Record<string, unknown>): EnhancedExecutionContext;
    
    /**
     * Extract basic context from enhanced context
     */
    simplify(enhancedContext: EnhancedExecutionContext): Record<string, unknown>;
    
    /**
     * Merge contexts while preserving enhanced state
     */
    merge(base: EnhancedExecutionContext, updates: Partial<EnhancedExecutionContext>): EnhancedExecutionContext;
    
    /**
     * Validate context structure and data
     */
    validate(context: EnhancedExecutionContext): boolean;
    
    /**
     * Prune completed/expired state from context
     */
    prune(context: EnhancedExecutionContext): EnhancedExecutionContext;
}

/**
 * RunExecutionContext - Comprehensive execution state for Tier 2
 * 
 * This type represents the complete state of a routine execution,
 * including navigation, variables, resource tracking, and progress.
 */
export interface RunExecutionContext {
    // Core identification
    runId: string;
    routineId: string;
    swarmId?: string;

    // Navigation state
    navigator: any; // Dynamic navigator instance based on routine type
    currentLocation: Location;
    visitedLocations: Location[];

    // Execution state
    variables: Record<string, unknown>;
    outputs: Record<string, unknown>;
    completedSteps: string[];
    parallelBranches: BranchExecution[];

    // Swarm inheritance (optional - populated when run is part of a swarm)
    parentContext?: any; // Reference to parent swarm context
    availableResources?: any[]; // Resources allocated from swarm pool
    sharedKnowledge?: Record<string, unknown>; // Shared swarm state

    // Resource tracking
    resourceLimits: {
        maxCredits: string; // stringified BigInt
        maxDurationMs: number;
        maxMemoryMB: number;
        maxSteps: number;
    };
    resourceUsage: {
        creditsUsed: string; // stringified BigInt
        durationMs: number;
        memoryUsedMB: number;
        stepsExecuted: number;
        startTime: Date;
    };

    // Progress tracking
    progress: {
        currentStepId: string | null;
        completedSteps: string[];
        totalSteps: number;
        percentComplete: number;
    };

    // Error handling
    retryCount: number;
    lastError?: Error | unknown;
}
