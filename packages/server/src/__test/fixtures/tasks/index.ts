/**
 * Task fixture factories for testing
 * 
 * This module provides factory functions for creating test data for various task types
 * used in the queue system. All factories create valid task objects with proper type safety.
 */

export { createRunTask, runTaskScenarios } from "./runTaskFactory.js";
export { createSwarmTask, swarmTaskScenarios } from "./swarmTaskFactory.js";
