/**
 * Shared execution utilities
 * 
 * This module provides utilities that are shared across all three tiers
 * of the execution architecture, ensuring consistency and proper data flow.
 */

// Base classes for common patterns
export {
    BaseStateMachine,
    BaseStates, type BaseEvent, type BaseState, type ManagedTaskStateMachine
} from "./BaseStateMachine.js";

export {
    BaseTierExecutor,
    TierResourceUtils, type ExecutionMetrics, type TierErrorContext
} from "./BaseTierExecutor.js";

export {
    BaseResourceManager,
    SimpleResourceManager, type AllocationResult, type ResourceAllocation, type ResourceLimits, type ResourceRequest
} from "./BaseResourceManager.js";

// Context validation utilities
export {
    ContextValidationError, ContextValidator,
    ContextValidatorFactory, type ValidationError, type ValidationResult, type ValidationWarning
} from "./contextValidator.js";

// Socket events now handled through unified event system
// See: /packages/server/src/services/events/adapters/socketEventAdapter.ts

// SwarmContextManager - Unified swarm state management
export {
    SwarmContextManager,
    type ISwarmContextManager
} from "./SwarmContextManager.js";

export {
    isContextUpdateEvent, isResourcePool, isUnifiedSwarmContext, type BlackboardItem, type BlackboardState, type ContextQuery, type ContextSubscription, type ContextUpdateEvent, type ContextValidationResult, type MOISEPolicy, type ResourcePolicy, type ResourcePool, type SecurityPolicy, type SwarmConfiguration, type SwarmExecutionState, type ResourceAllocation as SwarmResourceAllocation, type UnifiedSwarmContext
} from "./UnifiedSwarmContext.js";

export {
    ResourceFlowProtocol,
    type ResourceFlowConfig
} from "./ResourceFlowProtocol.js";

export {
    ContextSubscriptionManager,
    type IContextSubscriptionManager, type NotificationBatch, type SubscriptionFilter
} from "./ContextSubscriptionManager.js";

// Note: Tier-specific context types should be imported directly from their respective tiers
// to avoid circular dependencies. Do not re-export them here.
