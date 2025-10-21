#!/usr/bin/env node

/**
 * Simple HTTP server for scenario-to-extension UI
 * Serves static files and provides health check endpoint
 */

const http = require('http');
const fs = require('fs');
const path = require('path');
const url = require('url');

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

const server = http.createServer((req, res) => {
  const parsedUrl = url.parse(req.url);
  const pathname = parsedUrl.pathname;

  // Health check endpoint
  if (pathname === '/health') {
    // Validate API_PORT if used
    if (!process.env.API_PORT) {
      console.error('WARNING: API_PORT environment variable not set');
    }
    const apiPort = process.env.API_PORT ? parseInt(process.env.API_PORT, 10) : null;
    const apiUrl = apiPort ? `http://localhost:${apiPort}` : 'unknown';

    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({
      status: 'healthy',
      service: 'scenario-to-extension-ui',
      timestamp: new Date().toISOString(),
      readiness: true,
      api_connectivity: {
        connected: false,
        api_url: apiUrl,
        last_check: new Date().toISOString(),
        error: null,
        latency_ms: null
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
process.on('SIGTERM', () => {
  console.log('SIGTERM received, shutting down gracefully...');
  server.close(() => {
    console.log('Server closed');
    process.exit(0);
  });
});

process.on('SIGINT', () => {
  console.log('SIGINT received, shutting down gracefully...');
  server.close(() => {
    console.log('Server closed');
    process.exit(0);
  });
});
