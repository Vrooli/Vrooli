#!/usr/bin/env node

const http = require('http');
const fs = require('fs');
const path = require('path');

const PORT = process.env.UI_PORT || process.env.PORT || 35000;
const API_PORT = process.env.API_PORT || 8080;

const MIME_TYPES = {
  '.html': 'text/html',
  '.css': 'text/css',
  '.js': 'application/javascript',
  '.json': 'application/json',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.gif': 'image/gif',
  '.svg': 'image/svg+xml',
  '.ico': 'image/x-icon'
};

const server = http.createServer((req, res) => {
  let filePath = path.join(__dirname, req.url === '/' ? 'index.html' : req.url);
  
  // Security: prevent directory traversal
  const safePath = path.normalize(filePath);
  if (!safePath.startsWith(__dirname)) {
    res.writeHead(403);
    res.end('Forbidden');
    return;
  }

  // Handle health check
  if (req.url === '/health' || req.url === '/api/health') {
    const apiUrl = `http://localhost:${API_PORT}`;
    const checkStart = Date.now();

    // Check API connectivity
    http.get(`${apiUrl}/api/v1/health`, (apiRes) => {
      const latency = Date.now() - checkStart;
      const healthResponse = {
        status: 'healthy',
        service: 'image-tools-ui',
        timestamp: new Date().toISOString(),
        readiness: true,
        api_connectivity: {
          connected: apiRes.statusCode === 200,
          api_url: apiUrl,
          last_check: new Date().toISOString(),
          error: apiRes.statusCode !== 200 ? {
            code: `HTTP_${apiRes.statusCode}`,
            message: `API returned status ${apiRes.statusCode}`,
            category: 'network',
            retryable: true
          } : null,
          latency_ms: latency
        }
      };

      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify(healthResponse));
    }).on('error', (err) => {
      const latency = Date.now() - checkStart;
      const healthResponse = {
        status: 'degraded',
        service: 'image-tools-ui',
        timestamp: new Date().toISOString(),
        readiness: true,
        api_connectivity: {
          connected: false,
          api_url: apiUrl,
          last_check: new Date().toISOString(),
          error: {
            code: err.code || 'CONNECTION_ERROR',
            message: err.message || 'Failed to connect to API',
            category: 'network',
            retryable: true,
            details: {
              errno: err.errno,
              syscall: err.syscall
            }
          },
          latency_ms: null
        }
      };

      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify(healthResponse));
    });
    return;
  }

  // Serve static files
  fs.readFile(filePath, (err, content) => {
    if (err) {
      if (err.code === 'ENOENT') {
        res.writeHead(404);
        res.end('File not found');
      } else {
        res.writeHead(500);
        res.end(`Server error: ${err.code}`);
      }
    } else {
      const ext = path.extname(filePath);
      const mimeType = MIME_TYPES[ext] || 'application/octet-stream';
      res.writeHead(200, { 'Content-Type': mimeType });
      res.end(content);
    }
  });
});

server.listen(PORT, () => {
  console.log(`🖼️  Image Tools UI Server`);
  console.log(`   Running at: http://localhost:${PORT}`);
  console.log(`   API endpoint: http://localhost:${API_PORT}`);
  console.log(`   Environment: ${process.env.NODE_ENV || 'development'}`);
  console.log(`\n   Ready for digital darkroom magic! 📸`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('\nShutting down UI server...');
  server.close(() => {
    console.log('UI server stopped.');
    process.exit(0);
  });
});