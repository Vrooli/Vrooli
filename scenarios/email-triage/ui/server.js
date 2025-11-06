#!/usr/bin/env node

// Lightweight static server for the Email Triage UI production bundle
const http = require('http');
const fs = require('fs');
const path = require('path');

const UI_PORT = parseInt(process.env.UI_PORT || process.env.PORT || '3201', 10);
if (!Number.isFinite(UI_PORT)) {
  console.error('❌ UI_PORT environment variable is required to start the UI server');
  process.exit(1);
}

const API_PORT = parseInt(process.env.API_PORT || process.env.EMAIL_TRIAGE_API_PORT || '0', 10);
const API_BASE_URL = process.env.API_BASE_URL || (Number.isFinite(API_PORT) && API_PORT > 0 ? `http://localhost:${API_PORT}` : '');
const DIST_DIR = path.join(__dirname, 'dist');
const INDEX_FILE = path.join(DIST_DIR, 'index.html');
const HEALTH_FILE = path.join(__dirname, 'health');

if (!fs.existsSync(INDEX_FILE)) {
  console.error('❌ Missing ui/dist/index.html. Run `npm run build` inside ./ui before starting develop lifecycle.');
  process.exit(1);
}

const MIME_TYPES = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.js': 'application/javascript; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.svg': 'image/svg+xml',
  '.ico': 'image/x-icon',
  '.txt': 'text/plain; charset=utf-8'
};

function sendJSON(res, status, payload) {
  const body = JSON.stringify(payload, null, 2);
  res.writeHead(status, {
    'Content-Type': 'application/json; charset=utf-8',
    'Content-Length': Buffer.byteLength(body)
  });
  res.end(body);
}

function readHealthTemplate() {
  try {
    const raw = fs.readFileSync(HEALTH_FILE, 'utf-8');
    return JSON.parse(raw);
  } catch (error) {
    return {
      status: 'healthy',
      service: 'email-triage-ui',
      readiness: true
    };
  }
}

function serveStaticFile(res, filePath) {
  const ext = path.extname(filePath).toLowerCase();
  const contentType = MIME_TYPES[ext] || 'application/octet-stream';
  const stream = fs.createReadStream(filePath);
  stream.on('open', () => {
    res.writeHead(200, { 'Content-Type': contentType });
    stream.pipe(res);
  });
  stream.on('error', () => {
    sendJSON(res, 404, { error: 'NOT_FOUND', file: path.relative(__dirname, filePath) });
  });
}

function resolveFilePath(requestPath) {
  const normalized = path.normalize(decodeURIComponent(requestPath)).replace(/^\/+/ , '/');
  const target = path.join(DIST_DIR, normalized);
  if (!target.startsWith(DIST_DIR)) {
    return INDEX_FILE;
  }
  if (fs.existsSync(target) && fs.statSync(target).isDirectory()) {
    return path.join(target, 'index.html');
  }
  if (fs.existsSync(target)) {
    return target;
  }
  return INDEX_FILE;
}

const server = http.createServer((req, res) => {
  const { url = '/', method } = req;
  if (method !== 'GET' && method !== 'HEAD') {
    sendJSON(res, 405, { error: 'METHOD_NOT_ALLOWED' });
    return;
  }

  if (url.startsWith('/health')) {
    const base = readHealthTemplate();
    base.timestamp = new Date().toISOString();
    base.api_connectivity = base.api_connectivity || {};
    if (API_BASE_URL) {
      base.api_connectivity.api_url = `${API_BASE_URL}/health`;
    }
    sendJSON(res, 200, base);
    return;
  }

  if (url.startsWith('/config')) {
    sendJSON(res, 200, {
      service: 'email-triage-ui',
      apiUrl: API_BASE_URL,
      timestamp: new Date().toISOString()
    });
    return;
  }

  const filePath = resolveFilePath(url.split('?')[0]);
  serveStaticFile(res, filePath);
});

server.listen(UI_PORT, () => {
  console.log(`📄 Email Triage UI serving dist bundle on http://localhost:${UI_PORT}`);
  if (API_BASE_URL) {
    console.log(`↪ Proxying API requests to ${API_BASE_URL}`);
  }
});
