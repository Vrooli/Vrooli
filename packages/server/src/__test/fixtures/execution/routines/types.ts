/**
 * Type definitions for routine test fixtures
 * 
 * Provides type-safe interfaces for all routine fixture configurations
 */

import type { 
    RoutineVersionConfigObject,
    ResourceSubType 
} from "@vrooli/shared";

/**
 * Base routine fixture interface
 */
export interface RoutineFixture {
    /** Unique identifier for the routine */
    id: string;
    /** Human-readable name */
    name: string;
    /** Description of what the routine does */
    description: string;
    /** Version string */
    version: string;
    /** Resource subtype determining routine category */
    resourceSubType: ResourceSubType;
    /** Routine configuration object */
    config: RoutineVersionConfigObject;
}

/**
 * Type guard to check if an object is a valid routine fixture
 */
export function isRoutineFixture(obj: unknown): obj is RoutineFixture {
    if (typeof obj !== 'object' || obj === null) return false;
    
    const fixture = obj as Record<string, unknown>;
    
    return (
        typeof fixture.id === 'string' &&
        typeof fixture.name === 'string' &&
        typeof fixture.description === 'string' &&
        typeof fixture.version === 'string' &&
        typeof fixture.resourceSubType === 'string' &&
        typeof fixture.config === 'object' &&
        fixture.config !== null
    );
}

/**
 * Collection of routine fixtures
 */
export type RoutineFixtureCollection<T extends string = string> = Record<T, RoutineFixture>;

/**
 * Agent to routine mapping entry
 */
export type AgentRoutineMapping = Record<string, string[]>;

/**
 * Routine statistics interface
 */
export interface RoutineStats {
    total: number;
    sequential: number;
    bpmn: number;
    byCategory: {
        security: number;
        medical: number;
        performance: number;
        system: number;
    };
    byStrategy: {
        reasoning: number;
        deterministic: number;
        conversational: number;
        auto: number;
    };
    byResourceType: {
        action: number;
        generate: number;
        code: number;
        web: number;
        multiStep: number;
    };
}

/**
 * Category type for routine filtering
 */
export type RoutineCategory = "security" | "medical" | "performance" | "system" | "bpmn";

/**
 * Execution strategy type
 */
export type ExecutionStrategy = "reasoning" | "deterministic" | "conversational" | "auto";