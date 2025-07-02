/**
 * Cross-Tier Communication Interface System
 * 
 * This module defines the standardized synchronous communication protocol that enables
 * direct execution delegation between the three intelligence tiers in Vrooli's execution
 * architecture. This is fundamentally different from the asynchronous event bus system.
 * 
 * ## Architecture Overview
 * 
 * The three-tier execution architecture follows this delegation pattern:
 * 
 * ```
 * Tier 1 (Coordination Intelligence) - Swarm management, strategic planning
 *    ↓ TierExecutionRequest<RoutineExecutionInput>
 * Tier 2 (Process Intelligence) - Routine orchestration, workflow navigation  
 *    ↓ TierExecutionRequest<StepExecutionInput>
 * Tier 3 (Execution Intelligence) - Direct tool execution, strategy implementation
 * ```
 * 
 * ## Synchronous vs Asynchronous Communication
 * 
 * ### Synchronous (This Interface)
 * - **Purpose**: Direct execution delegation with immediate results
 * - **Pattern**: Request-response with resource allocation and context propagation
 * - **Use Cases**: 
 *   - Tier 1 requesting routine execution from Tier 2
 *   - Tier 2 requesting step execution from Tier 3
 *   - Resource allocation and constraint enforcement
 *   - Error propagation and status monitoring
 * - **Characteristics**: Blocking, type-safe, resource-tracked
 * 
 * ### Asynchronous (Event Bus)
 * - **Purpose**: Emergent behaviors, coordination signals, optimization insights
 * - **Pattern**: Publish-subscribe with event-driven agent responses
 * - **Use Cases**:
 *   - Performance optimization agents analyzing execution patterns
 *   - Security agents responding to threat events
 *   - Resource monitoring agents sending alerts
 *   - Strategy evolution agents proposing improvements
 * - **Characteristics**: Non-blocking, event-driven, agent-subscribed
 * 
 * ## Data-Driven Emergent Capabilities
 * 
 * This interface supports emergent AI capabilities by:
 * 
 * 1. **Configuration-Driven Behavior**: Input types carry complete behavioral configuration
 * 2. **Resource Constraint Enforcement**: Prevents runaway emergent behaviors
 * 3. **Context Preservation**: Maintains execution hierarchy for learning and optimization
 * 4. **Type Safety**: Ensures reliable communication while agents evolve autonomously
 * 
 * ## Resource Flow Pattern
 * 
 * Resources flow down the hierarchy with constraints:
 * ```
 * Swarm (1000 credits) → Routine (500 credits) → Step (50 credits)
 * ```
 * 
 * Usage aggregates back up for tracking and billing:
 * ```
 * Step (45 used) → Routine (145 used) → Swarm (245 used)
 * ```
 * 
 * ## Error Propagation
 * 
 * Errors include tier identification for debugging and recovery:
 * - **Tier 3 errors**: Tool failures, validation errors, strategy failures
 * - **Tier 2 errors**: Workflow navigation errors, routine failures, resource exhaustion
 * - **Tier 1 errors**: Swarm coordination failures, organizational constraint violations
 * 
 * @see {@link https://github.com/Vrooli/Vrooli/blob/main/docs/architecture/execution/} - Complete execution architecture
 * @see EventBus - For asynchronous emergent coordination
 * @see ExecutionContext - For context propagation patterns
 */

import {
    type CoreResourceAllocation,
    type ExecutionContext,
    type ExecutionOptions,
    type ExecutionResult,
    type ExecutionStatus,
} from "./core.js";
import type { SwarmId } from "./ids.js";

/**
 * Standard request structure for cross-tier execution delegation
 * 
 * This is the unified request format used for all tier-to-tier communication.
 * It encapsulates everything needed for safe, tracked execution delegation.
 * 
 * @template T - Tier-specific input type (SwarmCoordinationInput, RoutineExecutionInput, or StepExecutionInput)
 * 
 * ## Usage Patterns
 * 
 * **Tier 1 → Tier 2 Delegation:**
 * ```typescript
 * const request: TierExecutionRequest<RoutineExecutionInput> = {
 *   context: { swarmId, userData, ... },
 *   input: { routineId, parameters, workflow },
 *   allocation: { maxCredits: "500", maxDurationMs: 300000, ... },
 *   options: { strategy: "conversational", timeout: 30000 }
 * };
 * ```
 * 
 * **Tier 2 → Tier 3 Delegation:**
 * ```typescript
 * const request: TierExecutionRequest<StepExecutionInput> = {
 *   context: { swarmId, routineId, stepId, ... },
 *   input: { stepId, stepType, toolName, parameters, strategy },
 *   allocation: { maxCredits: "50", maxDurationMs: 30000, ... },
 *   options: { priority: "high", retryCount: 3 }
 * };
 * ```
 * 
 * ## Field Responsibilities
 * 
 * - **context**: Execution metadata, tracing, and environment information
 * - **input**: Tier-specific execution instructions and configuration
 * - **allocation**: Resource limits and constraints to prevent runaway execution
 * - **options**: Optional execution parameters and hints
 */
export interface TierExecutionRequest<T = unknown> {
    /** Execution context with tracing, environment, and hierarchy information */
    context: ExecutionContext;
    /** Tier-specific input containing all execution instructions and configuration */
    input: T;

    /** Resource allocation limits to constrain execution and prevent runaway behaviors */
    allocation: CoreResourceAllocation;

    /** Optional execution parameters and strategy hints */
    options?: ExecutionOptions;
}

/**
 * Standard Cross-Tier Communication Interface
 * 
 * This interface defines the contract that all three execution tiers must implement
 * to enable standardized, type-safe communication throughout the execution hierarchy.
 * Each tier acts as both a client (making requests) and server (handling requests).
 * 
 * ## Implementation Requirements
 * 
 * Each tier must implement this interface with tier-specific behavior:
 * 
 * **Tier 1 (TierOneCoordinator)**:
 * - Accepts: `SwarmCoordinationInput` for swarm creation and management
 * - Delegates: `RoutineExecutionInput` to Tier 2 for routine execution
 * - Responsibilities: Swarm lifecycle, resource hierarchy, organizational constraints
 * 
 * **Tier 2 (TierTwoOrchestrator)**:
 * - Accepts: `RoutineExecutionInput` for complete routine execution
 * - Delegates: `StepExecutionInput` to Tier 3 for individual step execution
 * - Responsibilities: Workflow navigation, routine orchestration, step coordination
 * 
 * **Tier 3 (TierThreeExecutor)**:
 * - Accepts: `StepExecutionInput` for direct tool execution
 * - No delegation: Executes tools directly and returns results
 * - Responsibilities: Tool orchestration, strategy execution, direct execution
 * 
 * ## Type Safety Guarantees
 * 
 * The interface uses TypeScript generics to ensure type safety across tier boundaries:
 * - Input types are tier-specific and validated at compile time
 * - Output types maintain consistency through the execution chain
 * - Resource allocation is enforced structurally
 * - Context propagation preserves execution hierarchy
 * 
 * ## Resource Management
 * 
 * All implementations must:
 * 1. **Respect allocation limits**: Never exceed provided resource constraints
 * 2. **Track usage accurately**: Report actual resource consumption
 * 3. **Handle exhaustion gracefully**: Fail fast when resources are depleted
 * 4. **Aggregate child usage**: Roll up resource consumption from delegated executions
 * 
 * ## Error Handling Requirements
 * 
 * All implementations must:
 * 1. **Include tier identification**: Mark errors with originating tier
 * 2. **Preserve error chains**: Maintain causal error relationships
 * 3. **Provide actionable messages**: Include debugging context
 * 4. **Handle cleanup**: Release resources on failure
 * 
 * @see TierExecutionRequest - For request structure and usage patterns
 * @see ExecutionResult - For response structure and resource tracking
 * @see ExecutionContext - For context propagation and tracing
 */
export interface TierCommunicationInterface {
    /**
     * Execute a tier-specific request and return structured results
     * 
     * This is the primary method for cross-tier execution delegation. Each tier
     * processes the request according to its responsibilities and either executes
     * directly or delegates to the next tier in the hierarchy.
     * 
     * ## Type Safety
     * 
     * The method uses TypeScript generics to ensure type safety:
     * - `TInput` must match the tier's expected input type
     * - `TOutput` represents the expected result structure
     * - Type mismatches are caught at compile time
     * 
     * ## Resource Enforcement
     * 
     * Implementations must:
     * - Never exceed `request.allocation` limits
     * - Track and report actual usage in `ExecutionResult.resourcesUsed`
     * - Fail gracefully when resources are exhausted
     * 
     * ## Error Handling
     * 
     * On failure, implementations must:
     * - Return `ExecutionResult` with `success: false`
     * - Include tier identification in error details
     * - Preserve error causality for debugging
     * - Release any allocated resources
     * 
     * @template TInput - Tier-specific input type (SwarmCoordinationInput | RoutineExecutionInput | StepExecutionInput)
     * @template TOutput - Expected result type from execution
     * @param request - Complete execution request with context, input, and resource allocation
     * @returns Promise resolving to structured execution result with resource usage
     * 
     * @throws Never throws - All errors must be returned in ExecutionResult structure
     */
    execute<TInput, TOutput>(
        request: TierExecutionRequest<TInput>
    ): Promise<ExecutionResult<TOutput>>;

    /**
     * Get current execution status for monitoring and coordination
     * 
     * Used for tracking long-running executions, particularly important for:
     * - Tier 1: Swarm lifecycle and coordination status
     * - Tier 2: Routine progress and step completion
     * - Tier 3: Tool execution and resource consumption
     * 
     * @param swarmId - Unique identifier for the execution to monitor
     * @returns Promise resolving to current execution status and metadata
     */
    getExecutionStatus(swarmId: SwarmId): Promise<ExecutionStatus>;

    /**
     * Cancel a running execution gracefully
     * 
     * Implementations must:
     * - Stop ongoing execution as quickly as possible
     * - Release allocated resources
     * - Clean up any side effects
     * - Propagate cancellation to delegated executions
     * 
     * @param swarmId - Unique identifier for the execution to cancel
     */
    cancelExecution(swarmId: SwarmId): Promise<void>;

    /**
     * Get tier-specific capabilities for discovery and routing
     * 
     * Used by coordination systems to:
     * - Discover available execution capabilities
     * - Make routing decisions based on tier characteristics
     * - Estimate resource requirements and latency
     * - Validate input type compatibility
     * 
     * @returns Promise resolving to tier capability description
     */
    getCapabilities?(): Promise<TierCapabilities>;
}

/**
 * Tier capability description
 */
export interface TierCapabilities {
    tier: "tier1" | "tier2" | "tier3";
    maxConcurrency: number;
    resourceLimits: {
        /** Maximum number of credits that can be used by the tier, as a stringified BigInt */
        maxCredits: string;
        /** Maximum duration of the execution in milliseconds */
        maxDurationMs: number;
        /** Maximum memory usage in megabytes */
        maxMemoryMB: number;
    };
}

/**
 * Tier 1 specific input types
 */
export type SwarmCoordinationInput = TierExecutionRequest<{
    /** 
     * The ID of the swarm (also the chat ID) if it exists.
     * Leave empty if the swarm is not yet created.
     */
    swarmId?: string;
    /**
     * The goal of the swarm.
     */
    goal: string;
    /**
     * Optional team configuration to influence the swarm's behavior.
     */
    teamConfiguration?: {
        /** 
         * What agent should be the leader of the swarm.
         * This will be the sole agent when the swarm is created, 
         * and is responsible for building the team.
         */
        leaderAgentId: string;
        /** The preferred team size for the swarm */
        preferredTeamSize: number;
        /** The skills required for the swarm. Influences the team building process. */
        requiredSkills: string[];
    };
    /**
     * Optional list of tools to limit the swarm to.
     */
    availableTools?: {
        name: string;
        description: string;
        // Add more tool metadata if needed
    }[];
    /**
     * Optional execution configuration for the swarm.
     */
    executionConfig?: {
        /** The preferred model when not specified by an agent/routine. */
        model?: string;
        /** The temperature for the model. */
        temperature?: number;
        /** The maximum number of parallel branches across all active runs. */
        parallelExecutionLimit?: number;
    };
}>

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
    allocation: CoreResourceAllocation,
    options?: ExecutionOptions,
): TierExecutionRequest<T> {
    return {
        context,
        input,
        allocation,
        options,
    };
}
