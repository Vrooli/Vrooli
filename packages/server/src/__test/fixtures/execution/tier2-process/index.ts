/**
 * Tier 2: Process Intelligence Fixtures
 * 
 * Universal workflow execution supporting multiple formats.
 * Plugin architecture supporting Native Vrooli, BPMN, and future formats.
 */

// Routine fixtures organized by domain
export * from "./routines/by-domain/index.js";
export * from "./routines/by-domain/types.js";

// Run state management
export * from "./run-states/runFixtures.js";

// Navigator patterns for different workflow formats
export * from "./navigators/configDrivenWorkflows.js";
export * from "./navigators/zeroCodeRoutines.js";

/**
 * Key Tier 2 Components:
 * - RunStateMachine: Orchestrating routine execution
 * - Navigator System: Pluggable components translating between universal execution model and platform-specific formats
 * - Evolution Support: Routines evolve through Conversational → Reasoning → Deterministic → Routing strategies
 */

// Helper to categorize routines by evolution stage
import { getAllRoutines } from "./routines/by-domain/index.js";
import type { RoutineFixture } from "./routines/by-domain/types.js";

export function getRoutinesByEvolutionStage() {
    const routines = getAllRoutines();
    
    return {
        conversational: routines.filter(r => r.config.executionStrategy === "conversational"),
        reasoning: routines.filter(r => r.config.executionStrategy === "reasoning"),
        deterministic: routines.filter(r => r.config.executionStrategy === "deterministic"),
        routing: routines.filter(r => r.config.executionStrategy === "routing"),
    };
}