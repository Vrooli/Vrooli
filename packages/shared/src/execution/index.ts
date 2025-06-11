/**
 * Main execution architecture exports
 * 
 * This module provides access to all execution-related types,
 * interfaces, and utilities for the three-tier architecture.
 */

// Export all execution types
export * from "./types/index.js";

// Export event system selectively to avoid conflicts
export { EventBuilder, EventCorrelator, EventFilterUtils, EventReplayer, type IEventBus } from "./events/eventBus.js";
export { EventTypeRegistry, type EventTypeMetadata } from "./events/eventTypes.js";
export { 
    EventValidator, 
    EventSanitizer,
    type ValidationResult as EventValidationResult,
    type ValidationError as EventValidationError,
    type ValidationWarning as EventValidationWarning
} from "./events/eventValidation.js";
