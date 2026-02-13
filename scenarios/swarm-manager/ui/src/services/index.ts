/**
 * Services Layer - Data access seams for Swarm Manager
 *
 * This module provides service objects that encapsulate API operations.
 * Each service is a seam - it can be substituted for testing without
 * mocking the entire HTTP layer.
 *
 * Usage:
 * - Import services for direct use in components or hooks
 * - Use factory functions (createXxxService) to inject mock clients in tests
 */

// Backlog
export { backlogService, createBacklogService } from "./backlog-service";
export type { IBacklogService } from "./backlog-service";

// Scenarios
export { scenariosService, createScenariosService } from "./scenarios-service";
export type { IScenariosService } from "./scenarios-service";

// Settings
export { settingsService, createSettingsService } from "./settings-service";
export type { ISettingsService } from "./settings-service";

// Agent Manager
export { agentManagerService, createAgentManagerService } from "./agent-manager-service";
export type { IAgentManagerService } from "./agent-manager-service";

// Execution
export { executionService, createExecutionService } from "./execution-service";
export type { IExecutionService, CreateExecutionRequest, ListExecutionFilters } from "./execution-service";
