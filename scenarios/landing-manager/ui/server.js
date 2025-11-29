/**
 * Production server for Landing Manager UI
 *
 * Uses @vrooli/api-base to automatically handle:
 * - /health endpoint with API connectivity checks
 * - /config endpoint with runtime configuration
 * - /api/* proxy to API server
 * - /landing/{scenarioId}/proxy/* proxy to generated scenarios
 * - Static file serving from ./dist
 * - SPA fallback routing
 */

import {
  createScenarioProxyHost,
  createScenarioServer,
  injectBaseTag,
} from '@vrooli/api-base/server';
import axios from 'axios';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const LOOPBACK_HOST = '127.0.0.1';
const LOOPBACK_HOSTS = ['127.0.0.1', 'localhost', '::1', '[::1]', '0.0.0.0'];
const HOST_SCENARIO = 'landing-manager';
const CACHE_TTL_MS = 30_000;
const DEFAULT_TIMEOUT_MS = 30_000;

if (!process.env.UI_PORT) {
  console.error('Error: UI_PORT environment variable is required');
  process.exit(1);
}
if (!process.env.API_PORT) {
  console.error('Error: API_PORT environment variable is required');
  process.exit(1);
}

const PORT = process.env.UI_PORT;
const API_PORT = process.env.API_PORT;
const API_BASE = `http://${LOOPBACK_HOST}:${API_PORT}`;

// Proxy host for viewing generated landing pages in iframes
const proxyHost = createScenarioProxyHost({
  hostScenario: HOST_SCENARIO,
  loopbackHosts: LOOPBACK_HOSTS,
  cacheTtlMs: CACHE_TTL_MS,
  timeoutMs: DEFAULT_TIMEOUT_MS,
  proxiedAppHeader: 'X-Landing-Manager-Scenario',
  childBaseTagAttribute: 'data-landing-manager',
  patchFetch: true,
  verbose: true,
  // Custom prefix for landing page previews (e.g., /landing/{scenarioId}/proxy/*)
  appsPathPrefix: '/landing',
  fetchAppMetadata: async (scenarioId) => {
    try {
      // Fetch preview links from the landing-manager API
      const response = await axios.get(
        `${API_BASE}/api/v1/preview/${encodeURIComponent(scenarioId)}`,
        { timeout: DEFAULT_TIMEOUT_MS }
      );
      const data = response.data;

      // Extract ports from the response (API now returns ui_port and api_port directly)
      const uiPort = data.ui_port ? parseInt(data.ui_port, 10) : null;
      const apiPort = data.api_port ? parseInt(data.api_port, 10) : null;

      // Return metadata in the format expected by createScenarioProxyHost
      // Note: config.api_port is used by pickApiPort() in host.ts
      return {
        id: scenarioId,
        name: data.scenario_id || scenarioId,
        port: uiPort,
        port_mappings: {
          UI_PORT: uiPort,
          API_PORT: apiPort,
        },
        config: {
          ui_url: data.base_url,
          api_port: apiPort,
        },
      };
    } catch (error) {
      console.error(`[landing-manager] Failed to fetch metadata for scenario ${scenarioId}:`, error.message);
      throw error;
    }
  },
});

const app = createScenarioServer({
  uiPort: PORT,
  apiPort: API_PORT,
  apiHost: LOOPBACK_HOST,
  distDir: path.join(__dirname, 'dist'),
  serviceName: 'landing-manager',
  version: '1.0.0',
  corsOrigins: '*',
  verbose: true,
  setupRoutes: (expressApp) => {
    // Ensure host HTML always has base href="/" while leaving proxied apps untouched
    expressApp.use((req, res, next) => {
      if (req.path.startsWith('/landing/') && req.path.includes('/proxy')) {
        return next();
      }

      const originalSend = res.send;
      res.send = function sendWithInjectedBase(body) {
        const contentType = res.getHeader('content-type');
        const isHtml = contentType && typeof contentType === 'string' && contentType.includes('text/html');

        if (isHtml && typeof body === 'string') {
          const modified = injectBaseTag(body, '/', {
            skipIfExists: true,
            dataAttribute: 'data-landing-manager-self',
          });
          return originalSend.call(this, modified);
        }

        return originalSend.call(this, body);
      };

      next();
    });

    // Mount the proxy router for generated scenario previews
    expressApp.use(proxyHost.router);
  },
});

app.listen(PORT, () => {
  console.log(`
╔════════════════════════════════════════════╗
║     LANDING MANAGER - FACTORY UI           ║
║                                             ║
║  UI Server running on port ${PORT}            ║
║  API proxy to port ${API_PORT}                 ║
║  Landing preview proxy active               ║
║                                             ║
║  Access dashboard at:                       ║
║  http://localhost:${PORT}                      ║
╚════════════════════════════════════════════╝
  `);
});

process.on('SIGTERM', () => {
  console.log('SIGTERM received, closing server...');
  process.exit(0);
});
