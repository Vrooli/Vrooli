/**
 * Integration module for the three-tier execution architecture
 * 
 * This module provides the central ExecutionArchitecture factory
 * that wires together all three tiers of Vrooli's execution system.
 */

export {
    ExecutionArchitecture,
    getExecutionArchitecture,
    resetExecutionArchitecture,
    type ExecutionArchitectureOptions,
} from "./executionArchitecture.js";

// Re-export resource types from shared (no longer from deprecated resources folder)
export type {
    AllocationResult, Resource, ResourceAccounting, ResourceLimitConfig, ResourceType,
    ResourceUnit,
} from "@vrooli/shared";

