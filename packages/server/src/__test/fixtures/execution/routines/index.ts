/**
 * Routine Fixtures Index
 * 
 * Provides centralized access to all routine fixtures across the execution architecture
 * with proper typing and validation integration.
 */

import { RoutineFixture } from "../types.js";
import type { RoutineConfigObject } from "@vrooli/shared";
// Import evolution fixtures
import {
    TECHNICAL_SUPPORT_SPECIALIST_V3,
    BILLING_SPECIALIST_V3,
    ACCOUNT_SPECIALIST_V3,
    ESCALATION_HANDLER_V2,
    RESPONSE_SYNTHESIZER_V1,
} from "./evolutionFixtures.js";

// Import domain-specific routines (when they exist)
// import { medicalRoutines } from "./by-domain/medicalRoutines.js";
// import { securityRoutines } from "./by-domain/securityRoutines.js";

/**
 * Mock routine structure for testing until full implementation
 */
interface MockRoutine {
    id: string;
    name: string;
    description: string;
    version: string;
    resourceSubType: string;
    config: RoutineConfigObject;
}

/**
 * Agent to routine mapping for execution architecture
 */
export const AGENT_ROUTINE_MAP = {
    "technical_support": TECHNICAL_SUPPORT_SPECIALIST_V3,
    "billing_support": BILLING_SPECIALIST_V3,
    "account_support": ACCOUNT_SPECIALIST_V3,
    "escalation_handler": ESCALATION_HANDLER_V2,
    "response_synthesizer": RESPONSE_SYNTHESIZER_V1,
} as const;

/**
 * Get all available routines
 */
export function getAllRoutines(): MockRoutine[] {
    const evolutionRoutines = Object.values(AGENT_ROUTINE_MAP);
    
    return evolutionRoutines.map((routine, index) => ({
        id: `routine_${index + 1}`,
        name: routine.name || `Routine ${index + 1}`,
        description: routine.description || `Evolution routine ${index + 1}`,
        version: routine.version || "1.0.0",
        resourceSubType: "conversational", // Default for now
        config: routine as RoutineConfigObject,
    }));
}

/**
 * Get routine by ID
 */
export function getRoutineById(id: string): MockRoutine | undefined {
    return getAllRoutines().find(routine => routine.id === id);
}

/**
 * Get routines by evolution stage
 */
export function getRoutinesByEvolutionStage() {
    return {
        conversational: [TECHNICAL_SUPPORT_SPECIALIST_V3],
        reasoning: [BILLING_SPECIALIST_V3, ESCALATION_HANDLER_V2], 
        deterministic: [ACCOUNT_SPECIALIST_V3],
        routing: [RESPONSE_SYNTHESIZER_V1],
    };
}
