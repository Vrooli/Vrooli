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

import { createScenarioServer, injectBaseTag } from '@vrooli/api-base/server';

const app = createScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: '{{SCENARIO_ID}}',
  version: '1.0.0',
  corsOrigins: '*',
  setupRoutes: (expressApp) => {
    // Ensure SPA assets always resolve from the root (per api-base host docs)
    expressApp.use((req, res, next) => {
      const originalSend = res.send;
      res.send = function sendWithBaseInjection(body) {
        const contentType = res.getHeader('content-type');
        const isHtml = contentType && typeof contentType === 'string' && contentType.includes('text/html');
        if (isHtml && typeof body === 'string') {
          const modified = injectBaseTag(body, '/', { skipIfExists: true });
          return originalSend.call(this, modified);
        }
        return originalSend.call(this, body);
      };
      next();
    });
  }
});

app.listen(process.env.UI_PORT);
