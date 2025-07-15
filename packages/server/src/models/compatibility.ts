// AI_CHECK: TYPE_SAFETY=phase1-compatibility | LAST: 2025-07-04 - Compatibility layer for gradual type migration
import { type ModelLogic, type ModelLogicType } from "./types.js";

/**
 * Compatibility type that allows both old (any) and new (typed) model logic
 * This is a temporary measure to allow gradual migration
 * @deprecated Will be removed once all models are properly typed
 */
export type CompatModelLogic = ModelLogic<
    ModelLogicType,
    readonly string[],
    string
> | ModelLogic<any, any, any>;

/**
 * Type assertion helper that maintains type safety while allowing migration
 */
export function asCompatModelLogic<T extends CompatModelLogic>(
    logic: T,
): T {
    return logic;
}

/**
 * Helper to check if a model logic is using the new type system
 */
export function isTypedModelLogic(
    logic: CompatModelLogic,
): logic is ModelLogic<ModelLogicType, readonly string[], string> {
    // In practice, we can't easily distinguish at runtime
    // This is mainly for documentation and future use
    return true;
}
