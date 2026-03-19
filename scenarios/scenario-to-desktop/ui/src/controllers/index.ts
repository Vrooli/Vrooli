/**
 * Controller layer - page-specific orchestration functions.
 *
 * Controllers combine:
 * - API calls for data fetching/mutation
 * - Service functions for data transformation
 * - Error handling and result packaging
 *
 * Controllers are called by hooks and provide a clean interface
 * for page-level operations.
 */

export * from "./generatorController";
export * from "./pipelineController";
export * from "./preflightController";
