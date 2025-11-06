import http from 'http';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const PORT = Number(process.env.UI_PORT || process.env.PORT || 35200);
const API_PORT = process.env.API_PORT || 15000;
const distDir = path.resolve(__dirname, 'dist');
const indexFile = path.join(distDir, 'index.html');

const mimeTypes = {
  '.html': 'text/html; charset=utf-8',
  '.js': 'application/javascript; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.svg': 'image/svg+xml',
  '.ico': 'image/x-icon'
};

function sendJSON(res, status, payload) {
  res.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8' });
  res.end(JSON.stringify(payload));
}

function serveStatic(filePath, res) {
  fs.stat(filePath, (err, stats) => {
    if (err || !stats.isFile()) {
      serveIndex(res);
      return;
    }

    const ext = path.extname(filePath).toLowerCase();
    res.writeHead(200, { 'Content-Type': mimeTypes[ext] || 'application/octet-stream' });
    fs.createReadStream(filePath).pipe(res);
  });
}

function serveIndex(res) {
  fs.readFile(indexFile, (err, data) => {
    if (err) {
      sendJSON(res, 500, { status: 'error', message: 'UI bundle missing. Run npm run build.' });
      return;
    }
    res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
    res.end(data);
  });
}

if (!fs.existsSync(indexFile)) {
  console.error('[accessibility-compliance-hub][ui] Missing dist/index.html. Run \"npm run build\" first.');
  process.exit(1);
}

const server = http.createServer((req, res) => {
  if (req.url === '/health') {
    sendJSON(res, 200, {
      status: 'healthy',
      service: 'accessibility-compliance-hub-ui',
      timestamp: new Date().toISOString()
    });
    return;
  }

  if (req.url === '/config') {
    sendJSON(res, 200, {
      apiPort: Number(API_PORT),
      uiPort: PORT,
      version: '1.0.0'
    });
    return;
  }

  const cleanPath = req.url.split('?')[0].split('#')[0];
  const requestedPath = path.join(distDir, cleanPath);
  const normalizedPath = path.normalize(requestedPath);

  if (!normalizedPath.startsWith(distDir)) {
    sendJSON(res, 400, { status: 'error', message: 'Invalid path' });
    return;
  }

  if (cleanPath === '/' || cleanPath === '') {
    serveIndex(res);
    return;
  }

  serveStatic(normalizedPath, res);
});

server.listen(PORT, () => {
  console.log(`[accessibility-compliance-hub][ui] Serving dist on http://localhost:${PORT}`);
});
