/**
 * MSW server setup for Node.js test environment.
 * 
 * This configures the mock service worker for intercepting HTTP requests
 * during testing, providing a reliable way to mock API responses.
 */

import { setupServer } from "msw/node";
import { handlers } from "./handlers.js";

/**
 * Create MSW server instance with default handlers.
 * Individual tests can add specific handlers using server.use().
 */
export const server = setupServer(...handlers);

/**
 * Helper to reset all handlers to their defaults.
 * Useful in test cleanup or when you want a fresh slate.
 */
export const resetHandlers = () => {
  server.resetHandlers(...handlers);
};

/**
 * Helper to get the base API URL for tests.
 * This should match the URL pattern used in the actual application.
 */
export const getApiUrl = (endpoint: string, includeRestBase = true) => {
  const baseUrl = process.env.VITE_API_URL || "http://localhost:3000/api";
  const restBase = includeRestBase ? "/rest" : "";
  return `${baseUrl}${restBase}${endpoint}`;
};
