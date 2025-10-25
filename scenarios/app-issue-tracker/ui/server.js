import express from 'express';
import path from 'path';
import { fileURLToPath } from 'url';
import http from 'http';
import httpProxy from 'http-proxy';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const app = express();

function requireEnv(name) {
  const value = process.env[name];
  if (!value) {
    throw new Error(
      `${name} environment variable is required. Run the scenario through the lifecycle system so configuration is injected automatically.`,
    );
  }
  return value;
}

function parsePort(value, label) {
  const parsed = Number.parseInt(value, 10);
  if (Number.isNaN(parsed) || parsed <= 0) {
    throw new Error(`${label} must be a positive integer (received: ${value})`);
  }
  return parsed;
}

function resolveUiPort() {
  const raw = process.env.UI_PORT;
  if (!raw) {
    throw new Error('UI_PORT must be provided for the UI server.');
  }
  return parsePort(raw, 'UI_PORT');
}

const PORT = resolveUiPort();
const API_PORT = parsePort(requireEnv('API_PORT'), 'API_PORT');
const API_PROTOCOL = (process.env.API_PROTOCOL || process.env.APP_ISSUE_TRACKER_API_PROTOCOL || 'http').toLowerCase() === 'https' ? 'https' : 'http';
const API_HOST = process.env.API_HOST || process.env.APP_ISSUE_TRACKER_API_HOST || '127.0.0.1';
const API_TARGET = `${API_PROTOCOL}://${API_HOST}:${API_PORT}`;

const apiProxy = httpProxy.createProxyServer({
  target: API_TARGET,
  ws: true,
  changeOrigin: true,
  proxyTimeout: 30_000,
  secure: API_PROTOCOL === 'https',
});

function normalizeUpstreamPath(pathInput) {
  if (!pathInput) {
    return '/';
  }
  return pathInput.startsWith('/') ? pathInput : `/${pathInput}`;
}

function proxyToApi(req, res, upstreamPath) {
  const normalizedPath = normalizeUpstreamPath(upstreamPath ?? req.url ?? '/');
  const upstreamTarget = new URL(normalizedPath, API_TARGET).toString();

  logProxyEvent('request', req);
  apiProxy.web(req, res, { target: upstreamTarget, ignorePath: true });
}

function proxyWebSocketToApi(req, socket, head, upstreamPath) {
  const normalizedPath = normalizeUpstreamPath(upstreamPath ?? req.url ?? '/');
  const upstreamTarget = new URL(normalizedPath, API_TARGET).toString();

  logProxyEvent('upgrade', req);
  apiProxy.ws(req, socket, head, { target: upstreamTarget, ignorePath: true });
}

function logProxyEvent(phase, req) {
  const method = req.method ?? 'GET';
  const url = req.url ?? 'unknown';
  const origin = req.headers?.origin ?? 'n/a';
  const forwardedFor = req.headers?.['x-forwarded-for'];
  const host = req.headers?.host ?? 'n/a';
  console.log(
    `[Proxy] ${phase} ${method} ${url} (host=${host}, origin=${origin}, forwardedFor=${forwardedFor ?? 'n/a'})`,
  );
}

apiProxy.on('error', (err, req, res) => {
  const message = err instanceof Error ? err.message : String(err);

  if (req) {
    logProxyEvent('error', req);
  }

  if (res && 'writeHead' in res) {
    const response = res;
    if (!response.headersSent) {
      response.writeHead(502, { 'Content-Type': 'application/json' });
    }
    response.end(
      JSON.stringify({
        error: 'API server unavailable',
        details: message,
      }),
    );
  } else if (res && 'end' in res) {
    res.end();
  }

  console.error('API proxy error:', message);
});

apiProxy.on('proxyReq', (proxyReq, req) => {
  if (req) {
    logProxyEvent('forward:http', req);
  }
  proxyReq.setHeader('origin', API_TARGET);
  proxyReq.setHeader('host', `${API_HOST}:${API_PORT}`);
});

apiProxy.on('proxyReqWs', (proxyReq, req) => {
  if (req) {
    logProxyEvent('forward:ws', req);
  }
  proxyReq.setHeader('origin', API_TARGET);
  proxyReq.setHeader('host', `${API_HOST}:${API_PORT}`);
});

apiProxy.on('proxyRes', (proxyRes, req, res) => {
  console.log(
    '[Proxy] response',
    proxyRes.statusCode,
    req?.url,
    'headers',
    proxyRes.headers,
    'ws',
    proxyRes.headers?.upgrade,
  );
});

apiProxy.on('start', (req, res, target) => {
  console.log('[Proxy] start', req?.method, req?.url, '->', target?.href ?? target);
});

apiProxy.on('open', (proxySocket) => {
  console.log('[Proxy] WebSocket tunnel established');
  proxySocket.on('error', (err) => {
    console.error('[Proxy] Upstream socket error:', err.message);
  });
});

apiProxy.on('close', (_res, socket) => {
  console.log('[Proxy] WebSocket tunnel closed');
  if (socket && !socket.destroyed) {
    socket.destroy();
  }
});

app.get('/health', (req, res) => {
  proxyToApi(req, res, '/health');
});

app.use('/api', (req, res) => {
  proxyToApi(req, res);
});

const distDir = path.join(__dirname, 'dist');
app.use(express.static(distDir));

app.get('*', (_req, res) => {
  res.sendFile(path.join(distDir, 'index.html'));
});

const server = http.createServer(app);

server.on('upgrade', (req, socket, head) => {
  if (!req.url?.startsWith('/api')) {
    socket.destroy();
    return;
  }

  proxyWebSocketToApi(req, socket, head);
});

server.listen(PORT, () => {
  const displayHost = process.env.UI_HOST || '127.0.0.1';
  console.log(`App Issue Tracker UI available at http://${displayHost}:${PORT}`);
  console.log(`Proxying API requests to ${API_TARGET}`);
});

export { proxyToApi };
