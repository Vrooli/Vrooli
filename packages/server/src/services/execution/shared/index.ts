/**
 * Shared execution utilities
 * 
 * This module provides utilities that are shared across all three tiers
 * of the execution architecture, ensuring consistency and proper data flow.
 */

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

// Re-export tier-specific context types for convenience
export { type ProcessRunContext } from "../tier2/context/contextManager.js";
export { 
    ExecutionRunContext,
    ExecutionRunContextFactory,
    type ExecutionRunContextConfig,
    type UserData,
    type StepConfig,
    type UsageHints,
} from "../tier3/context/runContext.js";