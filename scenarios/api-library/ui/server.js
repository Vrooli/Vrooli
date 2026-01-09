const http = require('http');
const fs = require('fs');
const path = require('path');

const PORT = Number(process.env.UI_PORT || process.env.PORT || 35100);
const API_PORT = process.env.API_PORT || 15100;
const distDir = path.join(__dirname, 'dist');
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

function serveFile(filePath, res) {
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
  console.error('[api-library][ui] Missing dist/index.html. Run "npm run build" first.');
  process.exit(1);
}

const server = http.createServer((req, res) => {
  if (req.url === '/health') {
    sendJSON(res, 200, {
      status: 'healthy',
      service: 'api-library-ui',
      timestamp: new Date().toISOString(),
      api: `http://localhost:${API_PORT}`
    });
    return;
  }

  const cleanPath = req.url.split('?')[0].split('#')[0];
  const candidate = path.join(distDir, cleanPath);
  const normalized = path.normalize(candidate);

  if (!normalized.startsWith(distDir)) {
    sendJSON(res, 400, { status: 'error', message: 'Invalid path' });
    return;
  }

  if (cleanPath === '/' || cleanPath === '') {
    serveIndex(res);
    return;
  }

  serveFile(normalized, res);
});

server.listen(PORT, () => {
  console.log(`[api-library][ui] Serving dist bundle at http://localhost:${PORT}`);
});
