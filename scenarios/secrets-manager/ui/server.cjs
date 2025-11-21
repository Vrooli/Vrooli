#!/usr/bin/env node
/**
 * Production server for secrets-manager UI
 * Serves the built dist/ bundle with proper health checks
 */

const express = require('express');
const path = require('path');
const fs = require('fs');
const { createProxyMiddleware } = require('http-proxy-middleware');

const app = express();
const PORT = process.env.UI_PORT || 37153;
const API_PORT = process.env.API_PORT || 16739;
const API_INTERNAL_BASE = process.env.API_INTERNAL_BASE || `http://127.0.0.1:${API_PORT}`;
const API_PROXY_PREFIX = process.env.API_PROXY_PREFIX || '/api';
const API_HEALTH_URL = new URL('/api/v1/health', API_INTERNAL_BASE).toString();

// Health check endpoint - standardized /health for lifecycle compatibility
app.get('/health', async (req, res) => {
  const distPath = path.join(__dirname, 'dist', 'index.html');
  const bundleExists = fs.existsSync(distPath);

  // Check API connectivity asynchronously
  let apiConnected = false;
  let apiLatency = null;
  let apiError = null;
  const lastCheck = new Date().toISOString();

  try {
    const http = require('http');
    const url = new URL(API_HEALTH_URL);
    const startTime = Date.now();

    const result = await new Promise((resolve) => {
      const options = {
        host: url.hostname,
        port: url.port,
        path: url.pathname,
        timeout: 2000,
        method: 'GET'
      };

      const request = http.request(options, (response) => {
        const latency = Date.now() - startTime;
        resolve({
          connected: response.statusCode === 200,
          latency,
          error: response.statusCode === 200 ? null : {
            code: 'HTTP_ERROR',
            message: `API returned status ${response.statusCode}`,
            category: 'network',
            retryable: true
          }
        });
      });

      request.on('error', (err) => {
        resolve({
          connected: false,
          latency: null,
          error: {
            code: 'CONNECTION_REFUSED',
            message: err.message || 'Failed to connect to API',
            category: 'network',
            retryable: true
          }
        });
      });

      request.on('timeout', () => {
        request.destroy();
        resolve({
          connected: false,
          latency: null,
          error: {
            code: 'TIMEOUT',
            message: 'API health check timed out after 2000ms',
            category: 'network',
            retryable: true
          }
        });
      });

      request.end();
    });

    apiConnected = result.connected;
    apiLatency = result.latency;
    apiError = result.error;
  } catch (error) {
    apiError = {
      code: 'UNKNOWN_ERROR',
      message: error.message || 'Unknown error during API connectivity check',
      category: 'internal',
      retryable: true
    };
  }

  const status = (bundleExists && apiConnected) ? 'healthy' :
                 bundleExists ? 'degraded' : 'unhealthy';
  const statusCode = status === 'healthy' ? 200 : status === 'degraded' ? 200 : 503;

  const statusNotes = [];
  if (!bundleExists) statusNotes.push('UI bundle not built');
  if (!apiConnected) statusNotes.push('API connectivity unavailable');

  res.status(statusCode).json({
    status,
    service: 'secrets-manager-ui',
    timestamp: new Date().toISOString(),
    readiness: bundleExists,
    status_notes: statusNotes.length > 0 ? statusNotes : undefined,
    api_connectivity: {
      connected: apiConnected,
      api_url: API_HEALTH_URL,
      last_check: lastCheck,
      error: apiError,
      latency_ms: apiLatency
    }
  });
});

// Proxy API requests to the Go backend
// Mount at /api path to avoid catching static files
const proxyTarget = `${API_INTERNAL_BASE}/api`;
console.log(`🔄 Configuring API proxy: ${API_PROXY_PREFIX}/* → ${proxyTarget}`);
app.use(
  API_PROXY_PREFIX,
  createProxyMiddleware({
    target: proxyTarget,
    changeOrigin: true,
    logLevel: 'warn',
    pathRewrite: {
      [`^${API_PROXY_PREFIX}`]: ''
    },
    onError: (err, req, res) => {
      console.error('Proxy error:', err.message);
      res.status(502).json({
        status: 'error',
        message: 'API proxy error',
        error: err.message
      });
    }
  })
);

// Serve static files from dist directory
app.use(express.static(path.join(__dirname, 'dist'), {
  maxAge: '1d',
  etag: true,
  lastModified: true
}));

// SPA fallback - serve index.html for all routes EXCEPT /api/* and /health
app.get('*', (req, res, next) => {
  // Skip SPA fallback for API routes and health check
  if (req.path.startsWith('/api/') || req.path === '/health') {
    return next();
  }

  const indexPath = path.join(__dirname, 'dist', 'index.html');

  if (fs.existsSync(indexPath)) {
    res.sendFile(indexPath);
  } else {
    res.status(503).send('UI bundle not built. Run setup first.');
  }
});

// Error handler
app.use((err, req, res, next) => {
  console.error('Server error:', err);
  res.status(500).json({
    status: 'error',
    message: 'Internal server error'
  });
});

app.listen(PORT, '0.0.0.0', () => {
  console.log(`🎨 Secrets Manager UI serving production bundle on http://localhost:${PORT}`);
  console.log(`📡 API health target: ${API_HEALTH_URL}`);
  console.log(`📨 Client requests should hit ${API_PROXY_PREFIX}/v1/* (auto-resolved in UI)`);
});
