/**
 * Shared execution utilities
 * 
 * This module provides utilities that are shared across all three tiers
 * of the execution architecture, ensuring consistency and proper data flow.
 */

// Base classes for common patterns
export { 
    BaseStateMachine, 
    BaseStates, 
    type BaseState, 
    type BaseEvent,
    type ManagedTaskStateMachine,
} from "./BaseStateMachine.js";

export {
    BaseTierExecutor,
    TierResourceUtils,
    type TierErrorContext,
    type ExecutionMetrics,
} from "./BaseTierExecutor.js";

export { 
    type IStore,
    InMemoryStore,
    RedisStore,
    CachedStore,
} from "./BaseStore.js";

export {
    BaseResourceManager,
    SimpleResourceManager,
    type ResourceLimits,
    type ResourceAllocation,
    type ResourceRequest,
    type AllocationResult,
} from "./BaseResourceManager.js";

// Context validation utilities
export {
    ContextValidator,
    ContextValidatorFactory,
    ContextValidationError,
    type ValidationResult,
    type ValidationError,
    type ValidationWarning,
} from "./contextValidator.js";

// Socket events now handled through unified event system
// See: /packages/server/src/services/events/adapters/socketEventAdapter.ts

// Note: Tier-specific context types should be imported directly from their respective tiers
// to avoid circular dependencies. Do not re-export them here.
