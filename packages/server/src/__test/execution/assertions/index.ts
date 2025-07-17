/**
 * Execution Test Assertions
 * 
 * Central export for all custom test assertions
 */

export { extendBlackboardAssertions } from "./blackboard.js";
export { extendEventAssertions } from "./events.js";
export { extendCoordinationAssertions } from "./coordination.js";
export { extendOutcomeAssertions } from "./outcomes.js";

// Initialize all custom assertions
import { extendBlackboardAssertions } from "./blackboard.js";
import { extendEventAssertions } from "./events.js";
import { extendCoordinationAssertions } from "./coordination.js";
import { extendOutcomeAssertions } from "./outcomes.js";

// Auto-initialize when this module is imported
extendBlackboardAssertions();
extendEventAssertions();
extendCoordinationAssertions();
extendOutcomeAssertions();

export function initializeExecutionAssertions() {
    // Already initialized above, kept for backward compatibility
}
