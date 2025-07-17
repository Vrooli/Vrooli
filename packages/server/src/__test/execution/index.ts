/**
 * Execution Test Framework
 * 
 * Main entry point for the execution service test infrastructure
 */

// Schema registries
export * from "./schemas/routines/index.js";
export * from "./schemas/agents/index.js";
export * from "./schemas/swarms/index.js";

// Factories
export { RoutineFactory } from "./factories/routine/RoutineFactory.js";
export { AgentFactory } from "./factories/agent/AgentFactory.js";
export { SwarmFactory } from "./factories/swarm/SwarmFactory.js";
export { ScenarioFactory } from "./factories/scenario/ScenarioFactory.js";
export { ScenarioRunner } from "./factories/scenario/ScenarioRunner.js";
export { ScenarioValidator } from "./factories/scenario/ScenarioValidator.js";
export { MockController } from "./factories/scenario/MockController.js";

// Types
export type * from "./factories/scenario/types.js";

// Assertions
export * from "./assertions/index.js";

// Mock utilities
export { RoutineResponseMocker } from "./factories/routine/RoutineResponseMocker.js";
export { AgentBehaviorMocker } from "./factories/agent/AgentBehaviorMocker.js";
export { SwarmStateMocker } from "./factories/swarm/SwarmStateMocker.js";
