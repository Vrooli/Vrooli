/* eslint-disable @typescript-eslint/consistent-type-imports */
/**
 * Central Type System Export Index
 * 
 * This file provides the main export interface for Vrooli's centralized type system.
 * All types are defined in core-types.ts and re-exported here for easy importing.
 * 
 * **Usage**:
 * ```typescript
 * import type { 
 *   ExecutionError, 
 *   RunContext, 
 *   ResourceLimits,
 *   SecurityContext 
 * } from "../types/index.js";
 * ```
 * 
 * **Related Documentation**:
 * - [Core Types](./core-types.ts) - Complete type definitions
 * - [Communication Patterns](../communication/communication-patterns.md) - Type usage patterns
 * - [Integration Map](../communication/integration-map.md) - Type validation procedures
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
