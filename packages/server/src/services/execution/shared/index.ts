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

// Context transformation utilities
export {
    ContextTransformer,
    ContextTransformerFactory,
    type ContextTransformationOptions,
} from "./contextTransformation.js";

// Context validation utilities
export {
    ContextValidator,
    ContextValidatorFactory,
    ContextValidationError,
    type ValidationResult,
    type ValidationError,
    type ValidationWarning,
} from "./contextValidator.js";

// Socket event emission
export {
    ExecutionSocketEventEmitter,
} from "./SocketEventEmitter.js";

// Note: Tier-specific context types should be imported directly from their respective tiers
// to avoid circular dependencies. Do not re-export them here.