/**
 * Type declarations for routine fixtures
 * 
 * This file provides enhanced IDE support and type information
 */

import type { ResourceSubType } from "@vrooli/shared";
import type { 
    RoutineFixture, 
    RoutineFixtureCollection, 
    RoutineCategory, 
    ExecutionStrategy,
    RoutineStats,
    AgentRoutineMapping
} from "./types";

// Re-export types for convenience
export type {
    RoutineFixture,
    RoutineFixtureCollection,
    RoutineCategory,
    ExecutionStrategy,
    RoutineStats,
    AgentRoutineMapping
} from "./types";

// Specific routine collection types
export declare const SECURITY_ROUTINES: RoutineFixtureCollection<
    | "HIPAA_COMPLIANCE_CHECK"
    | "API_SECURITY_SCAN"
    | "GDPR_DATA_AUDIT"
    | "TRADING_PATTERN_ANALYSIS"
>;

export declare const MEDICAL_ROUTINES: RoutineFixtureCollection<
    "MEDICAL_DIAGNOSIS_VALIDATION"
>;

export declare const PERFORMANCE_ROUTINES: RoutineFixtureCollection<
    | "PERFORMANCE_BOTTLENECK_DETECTION"
    | "COST_ANALYSIS"
    | "OUTPUT_QUALITY_ASSESSMENT"
>;

export declare const SYSTEM_ROUTINES: RoutineFixtureCollection<
    | "SYSTEM_FAILURE_ANALYSIS"
    | "SYSTEM_HEALTH_CHECK"
>;

export declare const BPMN_WORKFLOWS: RoutineFixtureCollection<
    | "COMPREHENSIVE_SECURITY_AUDIT"
    | "MEDICAL_TREATMENT_VALIDATION"
    | "RESILIENCE_OPTIMIZATION_WORKFLOW"
>;

// Aggregated collections
export declare const SEQUENTIAL_ROUTINES: RoutineFixtureCollection;
export declare const BPMN_ROUTINES: RoutineFixtureCollection;

// Helper functions with precise return types
export declare function getAllRoutines(): RoutineFixture[];
export declare function getRoutineById(routineId: string): RoutineFixture | undefined;
export declare function getRoutinesByStrategy(strategy: ExecutionStrategy): RoutineFixture[];
export declare function getRoutinesByResourceSubType(subType: ResourceSubType): RoutineFixture[];
export declare function getRoutinesByCategory(category: RoutineCategory): RoutineFixture[];
export declare function getRoutinesForAgent(agentId: string): RoutineFixture[];
export declare function getRoutineStats(): RoutineStats;

// Agent mapping
export declare const AGENT_ROUTINE_MAP: AgentRoutineMapping;