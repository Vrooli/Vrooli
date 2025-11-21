/**
 * Production server for Landing Manager UI
 *
 * Uses @vrooli/api-base to automatically handle:
 * - /health endpoint with API connectivity checks
 * - /config endpoint with runtime configuration
 * - /api/* proxy to API server
 * - Static file serving from ./dist
 * - SPA fallback routing
 */

import { startScenarioServer } from '@vrooli/api-base/server';

startScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'landing-manager',
  version: '1.0.0',
  corsOrigins: '*'
});
