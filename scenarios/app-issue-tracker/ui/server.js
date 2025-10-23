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
const API_TARGET = `http://localhost:${API_PORT}`;

const apiProxy = httpProxy.createProxyServer({
  target: API_TARGET,
  ws: true,
  changeOrigin: true,
  proxyTimeout: 30_000,
});

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
  proxyReq.setHeader('host', `localhost:${API_PORT}`);
});

apiProxy.on('proxyReqWs', (proxyReq, req) => {
  if (req) {
    logProxyEvent('forward:ws', req);
  }
  proxyReq.setHeader('origin', API_TARGET);
  proxyReq.setHeader('host', `localhost:${API_PORT}`);
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
  logProxyEvent('request', req);
  apiProxy.web(req, res);
});

app.use('/api', (req, res) => {
  logProxyEvent('request', req);
  apiProxy.web(req, res);
});

const distDir = path.join(__dirname, 'dist');
app.use(express.static(distDir));

app.get('*', (_req, res) => {
  res.sendFile(path.join(distDir, 'index.html'));
});

const server = http.createServer(app);

server.on('upgrade', (req, socket, head) => {
  logProxyEvent('upgrade', req);
  if (!req.url?.startsWith('/api')) {
    socket.destroy();
    return;
  }

  apiProxy.ws(req, socket, head);
});

server.listen(PORT, () => {
  console.log(`App Issue Tracker UI available at http://localhost:${PORT}`);
  console.log(`Proxying API requests to ${API_TARGET}`);
});
