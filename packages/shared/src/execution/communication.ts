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

import type { SessionUser, TaskStatus } from "../api/types.js";
import type { RunConfig, RunTriggeredFrom } from "../run/types.js";
import {
    type CoreResourceAllocation,
    type ExecutionOptions,
    type ExecutionStrategy,
} from "./core.js";
import type { SwarmId } from "./ids.js";

export interface BaseTierExecutionRequest {
    /** Resource allocation limits to constrain execution and prevent runaway behaviors */
    allocation: CoreResourceAllocation;
    /** Optional execution parameters and strategy hints */
    options?: ExecutionOptions;
}

/**
 * Standard request structure for cross-tier execution delegation
 * 
 * This is the unified request format used for all tier-to-tier communication.
 * It encapsulates everything needed for safe, tracked execution delegation.
 * 
 * @template T - Tier-specific input type (SwarmExecutionInput, RoutineExecutionInput, or StepExecutionInput)
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
export type TierExecutionRequest<T = unknown> = BaseTierExecutionRequest & {
    /** Execution metadata, tracing, and environment information */
    context: {
        swarmId: string;
        userData: SessionUser;
        parentSwarmId?: string;
        timestamp?: Date;
    };
    input: T;
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
export type SwarmExecutionInput = TierExecutionRequest<{
    /** 
     * The ID of the swarm (also the chat ID) if it exists.
     * Leave empty if the swarm is not yet created.
     */
    swarmId?: SwarmId;
    /**
     * The ID of the chat to load configuration from.
     * If not provided, a new chat configuration will be created.
     */
    chatId?: string;
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
    /**
     * The user data of the user who triggered the bot response
     */
    userData: SessionUser;
}>

/**
 * Tier 2 specific input types
 */
export interface RoutineExecutionInput {
    /** The ID of the resource version to execute (what actually contains the executable routine) */
    resourceVersionId: string;
    /** Optional run ID for resuming existing executions */
    runId?: string;
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
 * 
 * Enhanced to support the full range of step execution capabilities including
 * LLM calls, tool execution, code execution, and API calls. The parameters
 * field carries step-specific configuration that varies by step type.
 */
export interface StepExecutionInput {
    /** Unique identifier for this step execution */
    stepId: string;
    
    /** Type of step from the routine definition (e.g., "decision", "action", "subprocess") */
    stepType: string;
    
    /** Optional execution type hint for stepExecutor's type inference */
    type?: "llm_call" | "tool_call" | "code_execution" | "api_call";
    
    /** Tool name for tool_call steps */
    toolName?: string;
    
    /** 
     * Step parameters containing all execution configuration.
     * The structure varies based on step type:
     * - For LLM calls: may contain messages, prompt, instructions
     * - For tool calls: contains tool-specific arguments
     * - For code execution: contains code, language, inputs
     * - For API calls: contains url, method, headers, body
     */
    parameters: Record<string, unknown>;
    
    /** Execution strategy to use for this step */
    strategy: ExecutionStrategy;
    
    // Optional fields that may be present in parameters but are called out
    // for type inference in stepExecutor
    messages?: any[];
    prompt?: string;
    code?: string;
    routineConfig?: any; // RoutineVersionConfigObject - avoiding circular dependency
    ioMapping?: any; // SubroutineIOMapping - avoiding circular dependency, will be properly typed when imported
}

/**
 * Tier 2 specific input for routine/run execution via queue
 * 
 * This type is used for queueing routine executions through the task queue system.
 * It provides all necessary information for Tier 2 to execute a routine, including
 * configuration, form values, and execution metadata.
 * 
 * ## Usage Pattern:
 * ```typescript
 * const runTask: RunExecutionInput = {
 *   context: { swarmId, userData, timestamp, ... },
 *   input: {
 *     runId: "run-123",
 *     resourceVersionId: "routine-456", 
 *     config: { botConfig, decisionConfig, limits, ... },
 *     formValues: { input1: "value1", ... },
 *     isNewRun: true,
 *     runFrom: RunTriggeredFrom.RunView,
 *     startedById: "user-789",
 *     status: TaskStatus.Idle
 *   },
 *   allocation: { maxCredits: "1000", maxDurationMs: 300000, ... }
 * };
 * ```
 */
export type RunExecutionInput = TierExecutionRequest<{
    /** The run ID */
    runId: string;
    /** Resource version ID of the routine */
    resourceVersionId: string;
    /** Run configuration (if new run) */
    config?: RunConfig;
    /** Form values for current step */
    formValues?: Record<string, unknown>;
    /** Whether this is a new run */
    isNewRun: boolean;
    /** What triggered the run */
    runFrom: RunTriggeredFrom;
    /** ID of user/bot who started the run */
    startedById: string;
    /** Current task status */
    status: TaskStatus | `${TaskStatus}`;
}>
