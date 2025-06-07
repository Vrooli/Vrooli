/**
 * Communication interfaces for tier-to-tier interaction
 * 
 * This module defines the standardized communication interfaces that
 * all three tiers must implement for consistent interaction.
 */

import {
    type ExecutionId,
    type ExecutionContext,
    type ExecutionResult,
    type ExecutionStatus,
    type ResourceAllocation,
    type ExecutionOptions,
} from "./core.js";

/**
 * Request structure for tier execution
 */
export interface TierExecutionRequest<T = unknown> {
    context: ExecutionContext;
    input: T;
    allocation: ResourceAllocation;
    options?: ExecutionOptions;
}

/**
 * Standard interface for tier-to-tier communication
 * All three tiers must implement this interface for consistent interaction
 */
export interface TierCommunicationInterface {
    /**
     * Execute a request and return results
     * This is the primary method for tier-to-tier delegation
     */
    execute<TInput, TOutput>(
        request: TierExecutionRequest<TInput>
    ): Promise<ExecutionResult<TOutput>>;

    /**
     * Get execution status for monitoring
     * Used for tracking long-running executions
     */
    getExecutionStatus(executionId: ExecutionId): Promise<ExecutionStatus>;

    /**
     * Cancel a running execution
     * Used for graceful execution termination
     */
    cancelExecution(executionId: ExecutionId): Promise<void>;

    /**
     * Get tier-specific capabilities
     * Used for capability discovery and routing decisions
     */
    getCapabilities?(): Promise<TierCapabilities>;
}

/**
 * Tier capability description
 */
export interface TierCapabilities {
    tier: "tier1" | "tier2" | "tier3";
    supportedInputTypes: string[];
    supportedStrategies?: string[];
    maxConcurrency: number;
    estimatedLatency: {
        p50: number;
        p95: number;
        p99: number;
    };
    resourceLimits: {
        maxCredits: string;
        maxDurationMs: number;
        maxMemoryMB: number;
    };
}

/**
 * Tier 1 specific input types
 */
export interface SwarmCoordinationInput {
    goal: string;
    availableAgents: Agent[];
    constraints?: MOISEConstraints;
    teamConfiguration?: TeamConfiguration;
}

export interface Agent {
    id: string;
    name: string;
    capabilities: string[];
    currentLoad: number;
    maxConcurrentTasks: number;
}

export interface MOISEConstraints {
    organizationalStructure?: OrganizationalStructure;
    roles?: Role[];
    missions?: Mission[];
    norms?: Norm[];
}

export interface OrganizationalStructure {
    hierarchy: HierarchyLevel[];
    groups: Group[];
    dependencies: Dependency[];
}

export interface HierarchyLevel {
    level: number;
    roles: string[];
}

export interface Group {
    id: string;
    name: string;
    members: string[];
    authority: string[];
}

export interface Dependency {
    from: string;
    to: string;
    type: "informational" | "authoritative" | "collaborative";
}

export interface Role {
    id: string;
    name: string;
    permissions: string[];
    responsibilities: string[];
}

export interface Mission {
    id: string;
    name: string;
    objectives: string[];
    assignedRoles: string[];
}

export interface Norm {
    id: string;
    name: string;
    condition: string;
    obligation: string;
    prohibition?: string;
}

export interface TeamConfiguration {
    preferredTeamSize: number;
    requiredSkills: string[];
    collaborationStyle: "hierarchical" | "collaborative" | "autonomous";
}

/**
 * Tier 2 specific input types
 */
export interface RoutineExecutionInput {
    routineId: string;
    parameters: Record<string, unknown>;
    workflow: WorkflowDefinition;
}

export interface WorkflowDefinition {
    steps: WorkflowStep[];
    dependencies: StepDependency[];
    parallelBranches?: ParallelBranch[];
}

export interface WorkflowStep {
    id: string;
    name: string;
    toolName: string;
    parameters: Record<string, unknown>;
    strategy: string;
    timeout?: number;
}

export interface StepDependency {
    stepId: string;
    dependsOn: string[];
    condition?: string;
}

export interface ParallelBranch {
    id: string;
    steps: string[];
    mergeStrategy: "all" | "first" | "majority";
}

/**
 * Tier 3 specific input types
 */
export interface StepExecutionInput {
    stepId: string;
    stepType: string;
    toolName?: string;
    parameters: Record<string, unknown>;
    strategy: string;
}

/**
 * Common delegation patterns
 */
export type Tier1ToTier2Request = TierExecutionRequest<RoutineExecutionInput>;
export type Tier2ToTier3Request = TierExecutionRequest<StepExecutionInput>;
export type SwarmCoordinationRequest = TierExecutionRequest<SwarmCoordinationInput>;

/**
 * Helper function to create execution requests
 */
export function createTierRequest<T>(
    context: ExecutionContext,
    input: T,
    allocation: ResourceAllocation,
    options?: ExecutionOptions,
): TierExecutionRequest<T> {
    return {
        context,
        input,
        allocation,
        options,
    };
}
