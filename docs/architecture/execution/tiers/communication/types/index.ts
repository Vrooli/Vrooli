/* eslint-disable @typescript-eslint/consistent-type-imports */
/**
 * Master Type Index for Inter-Tier Communication
 * 
 * This file exports all types used across the Vrooli execution architecture
 * communication system, providing a single import point for type consistency.
 * 
 * All types are now consolidated in core-types.ts to eliminate redundancy.
 */

// All types consolidated in core-types.ts
export * from "./core-types.js";

/**
 * Version information for type compatibility checking
 */
export const TYPE_SYSTEM_VERSION = "1.0.0";

/**
 * Type guard utilities for runtime type checking
 */
export function isExecutionEvent(obj: unknown): obj is { id: string; type: string; timestamp: Date } {
    return typeof obj === "object" &&
        obj !== null &&
        "id" in obj &&
        "type" in obj &&
        "timestamp" in obj;
}

export function isResourceLimits(obj: unknown): obj is import("./core-types.js").ResourceLimits {
    return typeof obj === "object" &&
        obj !== null &&
        "maxCredits" in obj &&
        "maxDurationMs" in obj &&
        "maxMemoryMB" in obj;
}

export function isSecurityContext(obj: unknown): obj is import("./core-types.js").SecurityContext {
    return typeof obj === "object" &&
        obj !== null &&
        "requesterTier" in obj &&
        "targetTier" in obj &&
        "permissions" in obj;
} 
