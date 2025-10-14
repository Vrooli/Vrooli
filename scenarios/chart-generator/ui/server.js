#!/usr/bin/env node

const http = require('http');
const fs = require('fs');
const path = require('path');
const url = require('url');

// Ports must be provided via environment variables (set by lifecycle system)
if (!process.env.UI_PORT) {
  console.error('❌ ERROR: UI_PORT environment variable is required');
  process.exit(1);
}
if (!process.env.API_PORT) {
  console.error('❌ ERROR: API_PORT environment variable is required');
  process.exit(1);
}

const port = parseInt(process.env.UI_PORT, 10);
const apiPort = parseInt(process.env.API_PORT, 10);
const apiUrl = `http://localhost:${apiPort}`;

// MIME types for serving static files
const mimeTypes = {
  '.html': 'text/html',
  '.js': 'text/javascript',
  '.css': 'text/css',
  '.json': 'application/json',
  '.png': 'image/png',
  '.jpg': 'image/jpg',
  '.svg': 'image/svg+xml',
  '.ico': 'image/x-icon'
};

const server = http.createServer((req, res) => {
  const parsedUrl = url.parse(req.url);
  const pathname = parsedUrl.pathname;

  // Health endpoint with proper schema compliance
  if (pathname === '/health') {
    // Check API connectivity
    const healthCheckStart = Date.now();
    http.get(`${apiUrl}/health`, (apiRes) => {
      const latency = Date.now() - healthCheckStart;
      const connected = apiRes.statusCode === 200;

      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({
        status: connected ? 'healthy' : 'degraded',
        service: 'chart-generator-ui',
        timestamp: new Date().toISOString(),
        readiness: true,
        api_connectivity: {
          connected: connected,
          api_url: apiUrl,
          last_check: new Date().toISOString(),
          latency_ms: connected ? latency : null,
          error: connected ? null : {
            code: 'API_UNREACHABLE',
            message: `API health check returned status ${apiRes.statusCode}`,
            category: 'network',
            retryable: true
          }
        }
      }, null, 2));
    }).on('error', (err) => {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({
        status: 'degraded',
        service: 'chart-generator-ui',
        timestamp: new Date().toISOString(),
        readiness: true,
        api_connectivity: {
          connected: false,
          api_url: apiUrl,
          last_check: new Date().toISOString(),
          latency_ms: null,
          error: {
            code: 'CONNECTION_REFUSED',
            message: err.message,
            category: 'network',
            retryable: true
          }
        }
      }, null, 2));
    });
    return;
  }

  // Serve static files
  let filePath = '.' + pathname;
  if (filePath === './') {
    filePath = './index.html';
  }

  const extname = String(path.extname(filePath)).toLowerCase();
  const contentType = mimeTypes[extname] || 'application/octet-stream';

  fs.readFile(filePath, (error, content) => {
    if (error) {
      if (error.code === 'ENOENT') {
        res.writeHead(404, { 'Content-Type': 'text/html' });
        res.end('<h1>404 Not Found</h1>', 'utf-8');
      } else {
        res.writeHead(500);
        res.end(`Server Error: ${error.code}`, 'utf-8');
      }
    } else {
      res.writeHead(200, { 'Content-Type': contentType });
      res.end(content, 'utf-8');
    }
  });
});

server.listen(port, () => {
  console.log(`🎨 Chart Generator UI running on http://localhost:${port}`);
  console.log(`📊 Connected to API at ${apiUrl}`);
  console.log(`🏥 Health endpoint: http://localhost:${port}/health`);
});
