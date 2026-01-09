import { createReadStream } from 'node:fs';
import { promises as fs } from 'node:fs';
import http from 'node:http';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const rootDir = __dirname;
const distDir = path.join(rootDir, 'dist');

const moduleRoots = new Map([
  ['/node_modules/@vrooli/iframe-bridge/', path.join(rootDir, 'node_modules/@vrooli/iframe-bridge')],
  ['/node_modules/@vrooli/api-base/', path.join(rootDir, 'node_modules/@vrooli/api-base')],
]);

function requireEnv(name, fallback) {
  const value = process.env[name] || fallback;
  if (typeof value === 'undefined' || value === null || String(value).trim() === '') {
    throw new Error(`${name} environment variable is required for the UI server.`);
  }
  return String(value);
}

function parsePort(value, label) {
  const parsed = Number.parseInt(String(value), 10);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    throw new Error(`${label} must be a positive integer (received: ${value})`);
  }
  return parsed;
}

const uiPort = parsePort(requireEnv('UI_PORT'), 'UI_PORT');
const apiPort = parsePort(requireEnv('API_PORT', process.env.FILE_TOOLS_DEFAULT_API_PORT || '15458'), 'API_PORT');
const explicitApiBase = process.env.FILE_TOOLS_API_BASE || '';
const apiToken = process.env.FILE_TOOLS_API_TOKEN || process.env.API_TOKEN || 'API_TOKEN_PLACEHOLDER';

const bootstrapPayload = {
  token: apiToken,
  explicitApiBase,
  defaultApiPort: String(apiPort),
};

const contentTypeMap = new Map([
  ['.html', 'text/html; charset=utf-8'],
  ['.css', 'text/css; charset=utf-8'],
  ['.js', 'application/javascript; charset=utf-8'],
  ['.json', 'application/json; charset=utf-8'],
  ['.png', 'image/png'],
  ['.svg', 'image/svg+xml'],
  ['.jpg', 'image/jpeg'],
  ['.jpeg', 'image/jpeg'],
  ['.ico', 'image/x-icon'],
  ['.woff2', 'font/woff2'],
]);

async function ensureDistPrepared() {
  try {
    const info = await fs.stat(distDir);
    if (!info.isDirectory()) {
      throw new Error(`${distDir} exists but is not a directory.`);
    }
  } catch (error) {
    console.log('[file-tools-ui] dist directory missing. Running build script…');
    await import('./scripts/build.mjs');
  }
}

function resolveModulePath(requestPath) {
  for (const [prefix, root] of moduleRoots.entries()) {
    if (!requestPath.startsWith(prefix)) {
      continue;
    }

    const suffix = requestPath.slice(prefix.length);
    const normalized = path.normalize(suffix);
    const candidate = path.join(root, normalized);

    if (candidate.startsWith(root)) {
      return candidate;
    }
  }

  return null;
}

function resolveStaticPath(requestPath) {
  const trimmed = requestPath.replace(/^\/+/, '');
  const normalized = path.normalize(trimmed);
  const candidate = path.join(distDir, normalized);

  if (!candidate.startsWith(distDir)) {
    return null;
  }

  return candidate;
}

function sendNotFound(res) {
  if (!res.headersSent) {
    res.writeHead(404, { 'Content-Type': 'text/plain; charset=utf-8' });
  }
  res.end('Not Found');
}

function sendServerError(res, error) {
  console.error('[file-tools-ui] Error handling request:', error);
  if (!res.headersSent) {
    res.writeHead(500, { 'Content-Type': 'text/plain; charset=utf-8' });
  }
  res.end('Internal Server Error');
}

function streamFile(res, filePath) {
  const extension = path.extname(filePath).toLowerCase();
  const contentType = contentTypeMap.get(extension) || 'application/octet-stream';
  res.writeHead(200, { 'Content-Type': contentType });

  const stream = createReadStream(filePath);
  stream.on('error', (error) => sendServerError(res, error));
  stream.pipe(res);
}

async function serveIndex(res) {
  try {
    const indexPath = path.join(distDir, 'index.html');
    let html = await fs.readFile(indexPath, 'utf8');
    html = html.replace('__FILE_TOOLS_BOOTSTRAP__', JSON.stringify(bootstrapPayload));
    res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
    res.end(html);
  } catch (error) {
    sendServerError(res, error);
  }
}

function proxyToApi(req, res, apiPath) {
  const normalizedPath = apiPath.startsWith('/') ? apiPath : `/${apiPath}`;
  const options = {
    hostname: '127.0.0.1',
    port: apiPort,
    path: normalizedPath,
    method: req.method,
    headers: {
      ...req.headers,
      host: `localhost:${apiPort}`,
    },
  };

  const proxyReq = http.request(options, (proxyRes) => {
    res.writeHead(proxyRes.statusCode || 502, proxyRes.headers);
    proxyRes.pipe(res);
  });

  proxyReq.on('error', (error) => {
    console.error('[file-tools-ui] API proxy error:', error);
    if (!res.headersSent) {
      res.writeHead(502, { 'Content-Type': 'application/json; charset=utf-8' });
    }
    res.end(JSON.stringify({ error: 'API server unavailable', details: error.message, target: `http://127.0.0.1:${apiPort}${normalizedPath}` }));
  });

  if (req.method === 'GET' || req.method === 'HEAD') {
    proxyReq.end();
  } else {
    req.pipe(proxyReq);
  }
}

async function handleRequest(req, res) {
  const requestUrl = req.url || '/';
  const parsedUrl = new URL(requestUrl, `http://localhost:${uiPort}`);
  const { pathname } = parsedUrl;

  if (pathname === '/health') {
    proxyToApi(req, res, '/health');
    return;
  }

  if (pathname.startsWith('/api/')) {
    proxyToApi(req, res, pathname);
    return;
  }

  if (pathname.startsWith('/node_modules/@vrooli/')) {
    const modulePath = resolveModulePath(pathname);
    if (!modulePath) {
      sendNotFound(res);
      return;
    }

    try {
      const stats = await fs.stat(modulePath);
      if (stats.isDirectory()) {
        sendNotFound(res);
        return;
      }

      streamFile(res, modulePath);
    } catch (error) {
      sendServerError(res, error);
    }
    return;
  }

  if (!['GET', 'HEAD'].includes(req.method || 'GET')) {
    res.writeHead(405, { 'Content-Type': 'text/plain; charset=utf-8' });
    res.end('Method Not Allowed');
    return;
  }

  if (pathname === '/' || pathname === '') {
    await serveIndex(res);
    return;
  }

  const staticPath = resolveStaticPath(pathname);
  if (staticPath) {
    try {
      const stats = await fs.stat(staticPath);
      if (stats.isDirectory()) {
        await serveIndex(res);
      } else {
        streamFile(res, staticPath);
      }
      return;
    } catch (error) {
      if (error.code !== 'ENOENT') {
        sendServerError(res, error);
        return;
      }
    }
  }

  await serveIndex(res);
}

async function main() {
  await ensureDistPrepared();

  const server = http.createServer((req, res) => {
    handleRequest(req, res).catch((error) => sendServerError(res, error));
  });

  server.listen(uiPort, '0.0.0.0', () => {
    console.log(`[file-tools-ui] UI server available at http://localhost:${uiPort}`);
    console.log(`[file-tools-ui] Proxying API requests to http://127.0.0.1:${apiPort}`);
  });
}

main().catch((error) => {
  console.error('[file-tools-ui] Failed to start UI server', error);
  process.exitCode = 1;
});
