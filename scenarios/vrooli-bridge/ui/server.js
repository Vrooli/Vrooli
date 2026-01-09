#!/usr/bin/env node
/**
 * Lightweight static file server with /health endpoint for Vrooli Bridge UI.
 * Serves production bundle from dist/ directory.
 */

import { createServer } from 'http';
import { readFileSync, existsSync, statSync } from 'fs';
import { join, extname, dirname } from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const BASE_DIR = join(__dirname, 'dist');
const DEFAULT_PORT = 8080;

const MIME_TYPES = {
  '.html': 'text/html',
  '.css': 'text/css',
  '.js': 'application/javascript',
  '.json': 'application/json',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.gif': 'image/gif',
  '.svg': 'image/svg+xml',
  '.ico': 'image/x-icon',
};

function serveFile(res, filePath) {
  try {
    const content = readFileSync(filePath);
    const ext = extname(filePath);
    const mimeType = MIME_TYPES[ext] || 'application/octet-stream';

    res.writeHead(200, {
      'Content-Type': mimeType,
      'Content-Length': content.length,
    });
    res.end(content);
  } catch (err) {
    res.writeHead(500, { 'Content-Type': 'text/plain' });
    res.end('Internal Server Error');
  }
}

function serveHealth(res) {
  const payload = {
    status: 'healthy',
    service: 'vrooli-bridge-ui',
    timestamp: new Date().toISOString(),
  };
  const body = JSON.stringify(payload);

  res.writeHead(200, {
    'Content-Type': 'application/json',
    'Content-Length': Buffer.byteLength(body),
  });
  res.end(body);
}

const server = createServer((req, res) => {
  let url = req.url;

  // Health check endpoint
  if (url === '/health') {
    serveHealth(res);
    return;
  }

  // Default to index.html for root
  if (url === '/' || url === '') {
    url = '/index.html';
  }

  // Remove query string
  url = url.split('?')[0];

  const filePath = join(BASE_DIR, url);

  // Security: ensure file is within BASE_DIR
  if (!filePath.startsWith(BASE_DIR)) {
    res.writeHead(403, { 'Content-Type': 'text/plain' });
    res.end('Forbidden');
    return;
  }

  // Check if file exists
  if (!existsSync(filePath)) {
    res.writeHead(404, { 'Content-Type': 'text/plain' });
    res.end('Not Found');
    return;
  }

  // Check if it's a file (not directory)
  const stat = statSync(filePath);
  if (stat.isDirectory()) {
    const indexPath = join(filePath, 'index.html');
    if (existsSync(indexPath)) {
      serveFile(res, indexPath);
    } else {
      res.writeHead(403, { 'Content-Type': 'text/plain' });
      res.end('Forbidden');
    }
    return;
  }

  serveFile(res, filePath);
});

const PORT = parseInt(process.env.UI_PORT || process.env.PORT || DEFAULT_PORT);

server.listen(PORT, '0.0.0.0', () => {
  console.log(`[UI] Serving Vrooli Bridge UI on port ${PORT} (http://localhost:${PORT})`);
});
