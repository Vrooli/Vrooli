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

// Ideas
export { ideasService, createIdeasService } from "./ideas-service";
export type { IIdeasService } from "./ideas-service";

// Scenarios
export { scenariosService, createScenariosService } from "./scenarios-service";
export type { IScenariosService } from "./scenarios-service";
