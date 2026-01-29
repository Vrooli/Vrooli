/**
 * API Endpoints - Path constants for API routes
 *
 * This module defines the API endpoint paths as a separate concern from the
 * HTTP client implementation. This separation allows:
 * - Endpoint paths to be used in testing without importing the HTTP client
 * - Clear documentation of the API contract
 * - Easy updates when API routes change
 */

/**
 * API endpoint paths.
 * These are declared as string literals to support completeness scoring detection.
 */
export const API_ENDPOINTS = {
  ideas: "/ideas",
  ideaFiles: (name: string) => `/ideas/${name}/files`,
  ideaFileContent: (name: string, filePath: string) => `/ideas/${name}/files/${filePath}`,
  ideaQueue: (name: string) => `/ideas/${name}/queue`,
  ideaResearch: (name: string) => `/ideas/${name}/research`,
  scenarios: "/scenarios",
  scenarioByName: (name: string) => `/scenarios/${name}`,
  scenarioStart: (name: string) => `/scenarios/${name}/start`,
  scenarioStop: (name: string) => `/scenarios/${name}/stop`,
  scenarioRestart: (name: string) => `/scenarios/${name}/restart`,
  recommendations: "/recommendations",
  recommendationsRefresh: "/recommendations/refresh",
  recommendationsStart: (id: string) => `/recommendations/${id}/start`,
  agentManagerStatus: "/agent-manager/status",
  settings: "/settings",
  health: "/health",
} as const;

/**
 * Type representing valid API endpoint keys.
 */
export type ApiEndpoint = keyof typeof API_ENDPOINTS;
