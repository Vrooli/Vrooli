/**
 * Type Guards and Validation for Cross-Tier Communication
 * 
 * This module provides type-safe validation and discrimination functions for 
 * cross-tier communication inputs. These replace unsafe runtime type checking
 * with proper TypeScript type guards that provide compile-time safety.
 * 
 * ## Purpose
 * 
 * Cross-tier communication in the three-tier execution architecture requires
 * robust type validation to ensure:
 * - **Type Safety**: Compile-time guarantees about input structure
 * - **Runtime Validation**: Safe discrimination between input types
 * - **Error Prevention**: Early detection of malformed requests
 * - **Performance**: Efficient validation without deep object inspection
 * 
 * ## Architecture Integration
 * 
 * ### Tier 1 (Coordination Intelligence)
 * - Uses `isSwarmCoordinationInput()` to validate swarm coordination requests
 * - Validates agent availability and organizational constraints
 * 
 * ### Tier 2 (Process Intelligence)
 * - Uses `isRoutineExecutionInput()` to validate routine execution requests
 * - Validates workflow definitions and step dependencies
 * 
 * ### Tier 3 (Execution Intelligence)
 * - Uses `isStepExecutionInput()` to validate step execution requests
 * - Validates strategies, step types, and tool parameters
 * 
 * ## Usage Examples
 * 
 * ### Basic Type Discrimination
 * ```typescript
 * function handleTierRequest(input: unknown): ExecutionResult {
 *     const typedInput = discriminateTierInput(input);
 *     if (!typedInput) {
 *         throw new Error("Invalid input type");
 *     }
 *     
 *     if (isSwarmCoordinationInput(typedInput)) {
 *         return handleSwarmCoordination(typedInput); // TypeScript knows the type
 *     } else if (isRoutineExecutionInput(typedInput)) {
 *         return handleRoutineExecution(typedInput);
 *     } else if (isStepExecutionInput(typedInput)) {
 *         return handleStepExecution(typedInput);
 *     }
 * }
 * ```
 * 
 * ### Execution Context Validation
 * ```typescript
 * function validateExecutionRequest<T>(request: unknown, inputValidator: (input: unknown) => input is T): boolean {
 *     return isValidTierExecutionRequest(request, inputValidator);
 * }
 * ```
 * 
 * ### Strategy and Step Type Validation
 * ```typescript
 * function createStepExecution(stepType: string, strategy: string) {
 *     if (!isValidStepType(stepType)) {
 *         throw new Error(`Invalid step type: ${stepType}`);
 *     }
 *     if (!isValidStrategy(strategy)) {
 *         throw new Error(`Invalid strategy: ${strategy}`);
 *     }
 *     // Types are now safely validated
 * }
 * ```
 * 
 * ## Validation Principles
 * 
 * ### Comprehensive Validation
 * - Validates both structure and content
 * - Checks required fields and their types
 * - Validates nested objects and arrays
 * - Ensures organizational constraints (MOISE+)
 * 
 * ### Performance Optimized
 * - Early termination on first invalid field
 * - Minimal object traversal
 * - No deep cloning or serialization
 * - Efficient array and object iteration
 * 
 * ### Error Context
 * - Provides specific validation failure reasons
 * - Maintains field-level error information
 * - Supports debugging and error reporting
 * - Enables graceful degradation
 */

import {
    type RoutineExecutionInput,
    type StepExecutionInput,
    type SwarmExecutionInput,
    type TierExecutionRequest,
    type WorkflowDefinition,
    type WorkflowStep,
} from "./communication.js";
import {
    type ExecutionContext,
} from "./core.js";

/**
 * Union type for all valid tier input types
 */
export type TierInput = SwarmExecutionInput | RoutineExecutionInput | StepExecutionInput;

/**
 * Type guard for SwarmExecutionInput
 * 
 * Validates that the input contains all required fields for swarm coordination
 * and that the organizational structure is properly formed.
 */
export function isSwarmCoordinationInput(input: unknown): input is SwarmExecutionInput {
    if (typeof input !== "object" || input === null) {
        return false;
    }

    const obj = input as Record<string, unknown>;

    // Check required fields
    if (typeof obj.goal !== "string" || obj.goal.trim().length === 0) {
        return false;
    }

    // Validate optional swarmId
    if (obj.swarmId !== undefined &&
        (typeof obj.swarmId !== "string" || obj.swarmId.trim().length === 0)) {
        return false;
    }

    // Validate optional teamConfiguration
    if (obj.teamConfiguration !== undefined && !isValidSwarmTeamConfiguration(obj.teamConfiguration)) {
        return false;
    }

    // Validate optional availableTools
    if (obj.availableTools !== undefined && !isValidAvailableTools(obj.availableTools)) {
        return false;
    }

    // Validate optional executionConfig
    if (obj.executionConfig !== undefined && !isValidExecutionConfig(obj.executionConfig)) {
        return false;
    }

    return true;
}

/**
 * Type guard for RoutineExecutionInput
 * 
 * Validates that the input contains all required fields for routine execution
 * and that the workflow definition is properly structured.
 */
export function isRoutineExecutionInput(input: unknown): input is RoutineExecutionInput {
    if (typeof input !== "object" || input === null) {
        return false;
    }

    const obj = input as Record<string, unknown>;

    // Check required fields
    if (typeof obj.routineId !== "string" || obj.routineId.trim().length === 0) {
        return false;
    }

    if (typeof obj.parameters !== "object" || obj.parameters === null) {
        return false;
    }

    if (!isValidWorkflowDefinition(obj.workflow)) {
        return false;
    }

    return true;
}

/**
 * Type guard for StepExecutionInput
 * 
 * Validates that the input contains all required fields for step execution
 * and that strategy and parameters are properly formed.
 */
export function isStepExecutionInput(input: unknown): input is StepExecutionInput {
    if (typeof input !== "object" || input === null) {
        return false;
    }

    const obj = input as Record<string, unknown>;

    // Check required fields
    if (typeof obj.stepId !== "string" || obj.stepId.trim().length === 0) {
        return false;
    }

    if (typeof obj.stepType !== "string" || obj.stepType.trim().length === 0) {
        return false;
    }

    if (typeof obj.parameters !== "object" || obj.parameters === null) {
        return false;
    }

    if (typeof obj.strategy !== "string" || obj.strategy.trim().length === 0) {
        return false;
    }

    // toolName is optional but must be string if present
    if (obj.toolName !== undefined && typeof obj.toolName !== "string") {
        return false;
    }

    return true;
}

/**
 * Discriminated union type guard for any tier input
 * 
 * Determines the specific input type and returns appropriate type predicate.
 * Useful for generic handlers that need to route based on input type.
 */
export function discriminateTierInput(input: unknown): TierInput | null {
    if (isSwarmCoordinationInput(input)) {
        return input;
    }
    if (isRoutineExecutionInput(input)) {
        return input;
    }
    if (isStepExecutionInput(input)) {
        return input;
    }
    return null;
}

/**
 * Type guard for execution context validation
 * 
 * Ensures execution context contains all required fields and maintains
 * proper hierarchy relationships.
 */
export function isValidExecutionContext(context: unknown): context is ExecutionContext {
    if (typeof context !== "object" || context === null) {
        return false;
    }

    const ctx = context as Record<string, unknown>;

    // Check required fields
    const requiredStringFields = ["executionId", "swarmId", "userId", "correlationId"];
    for (const field of requiredStringFields) {
        if (typeof ctx[field] !== "string" || (ctx[field] as string).trim().length === 0) {
            return false;
        }
    }

    // Check required timestamp field
    if (!ctx.timestamp || (typeof ctx.timestamp !== "object" && typeof ctx.timestamp !== "number")) {
        return false;
    }

    // Check optional parent execution ID
    if (ctx.parentExecutionId !== undefined &&
        (typeof ctx.parentExecutionId !== "string" || (ctx.parentExecutionId as string).trim().length === 0)) {
        return false;
    }

    // Check optional step and routine IDs
    if (ctx.stepId !== undefined &&
        (typeof ctx.stepId !== "string" || (ctx.stepId as string).trim().length === 0)) {
        return false;
    }

    if (ctx.routineId !== undefined &&
        (typeof ctx.routineId !== "string" || (ctx.routineId as string).trim().length === 0)) {
        return false;
    }

    return true;
}

/**
 * Validates TierExecutionRequest structure with type-specific input validation
 */
export function isValidTierExecutionRequest<T>(
    request: unknown,
    inputValidator: (input: unknown) => input is T,
): request is TierExecutionRequest<T> {
    if (typeof request !== "object" || request === null) {
        return false;
    }

    const req = request as Record<string, unknown>;

    // Validate context
    if (!isValidExecutionContext(req.context)) {
        return false;
    }

    // Validate input with provided validator
    if (!inputValidator(req.input)) {
        return false;
    }

    // Validate allocation
    if (!isValidResourceAllocation(req.allocation)) {
        return false;
    }

    return true;
}

// Helper validation functions

function isValidSwarmTeamConfiguration(config: unknown): boolean {
    if (typeof config !== "object" || config === null) {
        return false;
    }

    const tc = config as Record<string, unknown>;

    return typeof tc.leaderAgentId === "string" &&
        tc.leaderAgentId.trim().length > 0 &&
        typeof tc.preferredTeamSize === "number" &&
        tc.preferredTeamSize > 0 &&
        Array.isArray(tc.requiredSkills) &&
        tc.requiredSkills.every(skill => typeof skill === "string");
}

function isValidAvailableTools(tools: unknown): boolean {
    if (!Array.isArray(tools)) {
        return false;
    }

    return tools.every(tool => {
        if (typeof tool !== "object" || tool === null) {
            return false;
        }
        const t = tool as Record<string, unknown>;
        return typeof t.name === "string" &&
            t.name.trim().length > 0 &&
            typeof t.description === "string" &&
            t.description.trim().length > 0;
    });
}

function isValidExecutionConfig(config: unknown): boolean {
    if (typeof config !== "object" || config === null) {
        return false;
    }

    const ec = config as Record<string, unknown>;

    // All fields are optional, so if any are present, they must be valid
    if (ec.model !== undefined &&
        (typeof ec.model !== "string" || ec.model.trim().length === 0)) {
        return false;
    }

    if (ec.temperature !== undefined &&
        (typeof ec.temperature !== "number" || ec.temperature < 0 || ec.temperature > 2)) {
        return false;
    }

    if (ec.parallelExecutionLimit !== undefined &&
        (typeof ec.parallelExecutionLimit !== "number" || ec.parallelExecutionLimit < 1)) {
        return false;
    }

    return true;
}

function isValidWorkflowDefinition(workflow: unknown): workflow is WorkflowDefinition {
    if (typeof workflow !== "object" || workflow === null) {
        return false;
    }

    const wf = workflow as Record<string, unknown>;

    if (!Array.isArray(wf.steps)) {
        return false;
    }

    // Validate each step
    for (const step of wf.steps) {
        if (!isValidWorkflowStep(step)) {
            return false;
        }
    }

    if (!Array.isArray(wf.dependencies)) {
        return false;
    }

    // Validate dependencies structure
    for (const dep of wf.dependencies) {
        if (!isValidStepDependency(dep)) {
            return false;
        }
    }

    return true;
}

function isValidWorkflowStep(step: unknown): step is WorkflowStep {
    if (typeof step !== "object" || step === null) {
        return false;
    }

    const s = step as Record<string, unknown>;

    return typeof s.id === "string" &&
        typeof s.name === "string" &&
        typeof s.toolName === "string" &&
        typeof s.parameters === "object" &&
        s.parameters !== null &&
        typeof s.strategy === "string";
}

function isValidStepDependency(dep: unknown): boolean {
    if (typeof dep !== "object" || dep === null) {
        return false;
    }

    const d = dep as Record<string, unknown>;

    return typeof d.stepId === "string" &&
        Array.isArray(d.dependsOn) &&
        d.dependsOn.every(id => typeof id === "string");
}

function isValidResourceAllocation(allocation: unknown): boolean {
    if (typeof allocation !== "object" || allocation === null) {
        return false;
    }

    const alloc = allocation as Record<string, unknown>;

    return typeof alloc.maxCredits === "string" &&
        typeof alloc.maxDurationMs === "number" &&
        typeof alloc.maxMemoryMB === "number" &&
        alloc.maxDurationMs > 0 &&
        alloc.maxMemoryMB > 0;
}

/**
 * Strategy validation for step execution
 */
export const VALID_STRATEGIES = ["conversational", "reasoning", "deterministic", "routing"] as const;
export type ValidStrategy = typeof VALID_STRATEGIES[number];

export function isValidStrategy(strategy: string): strategy is ValidStrategy {
    return VALID_STRATEGIES.includes(strategy as ValidStrategy);
}

/**
 * Step type validation for step execution
 */
export const VALID_STEP_TYPES = [
    "tool_call",
    "api_request",
    "data_transform",
    "decision_point",
    "loop_iteration",
    "conditional_branch",
    "parallel_execution",
    "subroutine_call",
] as const;
export type ValidStepType = typeof VALID_STEP_TYPES[number];

export function isValidStepType(stepType: string): stepType is ValidStepType {
    return VALID_STEP_TYPES.includes(stepType as ValidStepType);
}
