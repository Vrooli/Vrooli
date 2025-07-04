/**
 * Types for RunContextManager - Tier 2 Resource & Context Management
 * 
 * This module defines the types used by RunContextManager for proper
 * resource hierarchy management (Swarm → Run → Step).
 */

import { type ResourcePool } from "@vrooli/shared";

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
    allocated: ResourcePool;
    remaining: ResourcePool;
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
    allocated: ResourcePool;
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
