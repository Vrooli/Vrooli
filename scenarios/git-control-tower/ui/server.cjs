const http = require('http');
const fs = require('fs');
const path = require('path');

// Initialize iframe bridge if this file executes in a browser context (safety net
// for preview if injected into a <script> tag).
if (typeof window !== 'undefined' && window.parent !== window && !window.__GIT_CONTROL_TOWER_BRIDGE_INITIALIZED) {
  try {
    const { initIframeBridgeChild } = require('@vrooli/iframe-bridge/child');

    let parentOrigin;
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }

    initIframeBridgeChild({ parentOrigin, appId: 'git-control-tower' });
    window.__GIT_CONTROL_TOWER_BRIDGE_INITIALIZED = true;
  } catch (error) {
    console.warn('[git-control-tower] iframe bridge bootstrap skipped', error);
  }
}

const PORT = Number(process.env.UI_PORT || process.env.PORT || 35700);
const API_PORT = process.env.API_PORT || PORT;
const ROOT_DIR = __dirname;
const DIST_DIR = path.join(ROOT_DIR, 'dist');
const SRC_DIR = path.join(ROOT_DIR, 'src');

const MIME_TYPES = {
  '.html': 'text/html',
  '.js': 'text/javascript',
  '.css': 'text/css',
  '.json': 'application/json',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.svg': 'image/svg+xml',
  '.ico': 'image/x-icon'
};

function getStaticRoot() {
  if (fs.existsSync(DIST_DIR)) {
    return DIST_DIR;
  }
  return ROOT_DIR;
}

function normalizePath(requestUrl = '/') {
  const urlPath = decodeURIComponent((requestUrl || '/').split('?')[0]);
  let relativePath = urlPath;
  if (!relativePath || relativePath === '/' || relativePath === './') {
    relativePath = '/index.html';
  }

  const trimmed = relativePath.replace(/^\/+/, '');
  const normalized = path
    .normalize(trimmed)
    .replace(/^(\.\.([\\/]|$))+/, '');

  const staticRoot = getStaticRoot();
  const candidate = path.join(staticRoot, normalized);
  if (!candidate.startsWith(staticRoot)) {
    return path.join(staticRoot, 'index.html');
  }

  return candidate;
}

function serveBridge(req, res) {
  const bridgePath = path.join(ROOT_DIR, 'node_modules', '@vrooli', 'iframe-bridge', 'dist', 'iframeBridgeChild.js');
  fs.readFile(bridgePath, (err, data) => {
    if (err) {
      res.writeHead(500, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'Bridge asset unavailable', detail: err.message }));
      return;
    }
    res.writeHead(200, { 'Content-Type': 'application/javascript' });
    res.end(data, 'utf-8');
  });
}

const server = http.createServer((req, res) => {
  if (!req || !res) {
    return;
  }

  if (req.url === '/health') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(
      JSON.stringify({
        status: 'healthy',
        service: 'git-control-tower-ui',
        port: PORT,
        apiPort: API_PORT,
        timestamp: new Date().toISOString()
      })
    );
    return;
  }

  if (req.url === '/bridge/iframeBridgeChild.js') {
    serveBridge(req, res);
    return;
  }

  const filePath = normalizePath(req.url);
  const ext = path.extname(filePath).toLowerCase();
  const contentType = MIME_TYPES[ext] || 'application/octet-stream';

  fs.readFile(filePath, (err, content) => {
    if (err) {
      if (err.code === 'ENOENT') {
        res.writeHead(404, { 'Content-Type': 'text/html' });
        res.end('<h1>404 - File Not Found</h1>', 'utf-8');
      } else {
        res.writeHead(500, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ error: 'Server error', code: err.code }));
      }
      return;
    }

    res.writeHead(200, { 'Content-Type': contentType });
    res.end(content, 'utf-8');
  });
});

server.listen(PORT, () => {
  console.log(`[git-control-tower] UI running at http://localhost:${PORT}`);
});
