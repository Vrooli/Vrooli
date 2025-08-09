/**
 * Schema Registry Index
 * 
 * Central exports for all test schema types and registries
 */

// Export routine schema types and registry
export type { RoutineSchema } from "./routines/index.js";
export { RoutineSchemaRegistry } from "./routines/index.js";

// Export agent schema types and registry
export type { AgentSchema } from "./agents/index.js";
export { AgentSchemaRegistry } from "./agents/index.js";

// Export swarm schema types and registry
export type { SwarmSchema } from "./swarms/index.js";
export { SwarmSchemaRegistry } from "./swarms/index.js";
