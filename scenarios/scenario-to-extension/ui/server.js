#!/usr/bin/env node

/**
 * Simple HTTP server for scenario-to-extension UI
 * Serves static files and provides health check endpoint
 */

import http from 'node:http';
import https from 'node:https';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath, parse as parseUrl } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Validate required environment variables
if (!process.env.UI_PORT) {
  console.error('ERROR: UI_PORT environment variable is required');
  process.exit(1);
}

const PORT = parseInt(process.env.UI_PORT, 10);
const HOST = process.env.UI_HOST || '0.0.0.0';

// Validate port is a valid number
if (isNaN(PORT) || PORT < 1 || PORT > 65535) {
  console.error(`ERROR: UI_PORT must be a valid port number (1-65535), got: ${process.env.UI_PORT}`);
  process.exit(1);
}

// MIME type mapping
const MIME_TYPES = {
  '.html': 'text/html',
  '.js': 'text/javascript',
  '.css': 'text/css',
  '.json': 'application/json',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.gif': 'image/gif',
  '.svg': 'image/svg+xml',
  '.ico': 'image/x-icon'
};

const API_HEALTH_PATH = '/api/v1/health';
const API_LOOPBACK_HOST = '127.0.0.1';
const API_PROBE_TIMEOUT_MS = 1500;

async function probeApiHealth(baseUrl) {
  if (!baseUrl) {
    return { connected: false, error: 'API_PORT not configured', latency: null, url: null };
  }

  try {
    const normalizedBase = baseUrl.replace(/\/$/, '');
    const healthUrl = new URL(`${normalizedBase}${API_HEALTH_PATH}`);
    const transport = healthUrl.protocol === 'https:' ? https : http;
    const start = Date.now();

    return await new Promise((resolve) => {
      const request = transport.request(
        {
          hostname: healthUrl.hostname,
          port: healthUrl.port || (healthUrl.protocol === 'https:' ? 443 : 80),
          path: `${healthUrl.pathname}${healthUrl.search}`,
          method: 'GET',
          headers: { Accept: 'application/json' },
          timeout: API_PROBE_TIMEOUT_MS
        },
        (response) => {
          const latency = Date.now() - start;
          response.resume();
          const statusCode = response.statusCode ?? 0;
          const ok = statusCode >= 200 && statusCode < 400;
          resolve({
            connected: ok,
            error: ok ? null : `HTTP ${statusCode}`,
            latency,
            url: normalizedBase
          });
        }
      );

      request.on('timeout', () => {
        request.destroy(new Error('timeout'));
      });

      request.on('error', (error) => {
        const latency = Date.now() - start;
        resolve({
          connected: false,
          error: error instanceof Error ? error.message : String(error),
          latency,
          url: normalizedBase
        });
      });

      request.end();
    });
  } catch (error) {
    return {
      connected: false,
      error: error instanceof Error ? error.message : String(error),
      latency: null,
      url: baseUrl
    };
  }
}

const server = http.createServer(async (req, res) => {
  const parsedUrl = parseUrl(req.url);
  const pathname = parsedUrl.pathname;

  // Health check endpoint
  if (pathname === '/health') {
    const timestamp = new Date().toISOString();
    const host = process.env.API_HOST || API_LOOPBACK_HOST;
    const rawPort = process.env.API_PORT;

    if (!rawPort) {
      console.warn('WARNING: API_PORT environment variable not set');
    }

    const parsedPort = rawPort ? Number.parseInt(rawPort, 10) : NaN;
    const hasValidPort = Number.isInteger(parsedPort) && parsedPort > 0;
    const probeHost = !host || host === 'localhost' || host === '0.0.0.0'
      ? API_LOOPBACK_HOST
      : host;
    const probeBaseUrl = hasValidPort ? `http://${probeHost}:${parsedPort}` : null;
    const reportedApiUrl = hasValidPort ? `http://${host}:${parsedPort}` : null;

    const connectivity = await probeApiHealth(probeBaseUrl ?? undefined).catch((error) => {
      console.warn('[scenario-to-extension] API connectivity probe failed:', error);
      return {
        connected: false,
        error: error instanceof Error ? error.message : String(error),
        latency: null,
        url: probeBaseUrl
      };
    });

    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({
      status: 'healthy',
      service: 'scenario-to-extension-ui',
      timestamp,
      readiness: true,
      api_connectivity: {
        connected: Boolean(connectivity.connected),
        api_url: reportedApiUrl || connectivity.url || 'unknown',
        last_check: timestamp,
        error: connectivity.error,
        latency_ms: typeof connectivity.latency === 'number' ? connectivity.latency : null
      }
    }));
    return;
  }

  // Serve index.html for root path
  let filePath = pathname === '/' ? '/index.html' : pathname;
  filePath = path.join(__dirname, filePath);

  // Security: prevent directory traversal
  const resolvedPath = path.resolve(filePath);
  const rootDir = path.resolve(__dirname);
  if (!resolvedPath.startsWith(rootDir)) {
    res.writeHead(403, { 'Content-Type': 'text/plain' });
    res.end('Forbidden');
    return;
  }

  // Check if file exists
  fs.stat(filePath, (err, stats) => {
    if (err || !stats.isFile()) {
      res.writeHead(404, { 'Content-Type': 'text/plain' });
      res.end('Not Found');
      return;
    }

    // Get MIME type
    const ext = path.extname(filePath);
    const mimeType = MIME_TYPES[ext] || 'application/octet-stream';

    // Read and serve file
    fs.readFile(filePath, (err, data) => {
      if (err) {
        res.writeHead(500, { 'Content-Type': 'text/plain' });
        res.end('Internal Server Error');
        return;
      }

      res.writeHead(200, { 'Content-Type': mimeType });
      res.end(data);
    });
  });
});

server.listen(PORT, HOST, () => {
  console.log(`scenario-to-extension UI server running at http://${HOST}:${PORT}`);
  console.log(`Health check: http://${HOST}:${PORT}/health`);
});

// Graceful shutdown
const gracefulShutdown = (signal) => {
  console.log(`${signal} received, shutting down gracefully...`);
  server.close(() => {
    console.log('Server closed');
    process.exit(0);
  });
};

process.on('SIGTERM', () => gracefulShutdown('SIGTERM'));
process.on('SIGINT', () => gracefulShutdown('SIGINT'));
