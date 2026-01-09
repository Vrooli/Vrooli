const express = require('express');
const path = require('path');
const fs = require('fs');
const { Readable } = require('stream');
const { spawnSync } = require('child_process');

const DEFAULT_API_PATH = '/api';
const DIST_DIR = path.join(__dirname, 'dist');
const DIST_INDEX = path.join(DIST_DIR, 'index.html');

const app = express();

ensureUiBundle();

function requireEnv(name) {
  const value = process.env[name];
  if (!value) {
    throw new Error(
      `${name} environment variable is required. Run the scenario through the lifecycle system so configuration is injected automatically.`
    );
  }
  return value;
}

function parsePort(value, label) {
  const parsed = Number.parseInt(value, 10);
  if (Number.isNaN(parsed) || parsed <= 0) {
    throw new Error(`${label} must be a positive integer (received ${value}).`);
  }
  return parsed;
}

const PORT = parsePort(requireEnv('UI_PORT'), 'UI_PORT');
const API_PORT = parsePort(requireEnv('API_PORT'), 'API_PORT');
const API_PROTOCOL = (process.env.API_PROTOCOL || 'http').toLowerCase() === 'https' ? 'https' : 'http';
const API_HOST = (process.env.API_HOST || '127.0.0.1').trim();
const API_TARGET = `${API_PROTOCOL}://${API_HOST}:${API_PORT}`;

app.disable('x-powered-by');

app.use('/api', (req, res) => proxyToApi(req, res, req.originalUrl || req.url));

app.get('/health', (req, res) => proxyToApi(req, res, '/health'));

app.get('/config', (_req, res) => {
  res.setHeader('Cache-Control', 'no-store');
  res.json({
    apiBase: DEFAULT_API_PATH,
    apiPath: DEFAULT_API_PATH,
    apiTarget: API_TARGET,
    apiHost: API_HOST,
    apiPort: API_PORT,
    timestamp: new Date().toISOString()
  });
});

if (fs.existsSync(DIST_DIR)) {
  app.use(express.static(DIST_DIR, { index: false, fallthrough: true }));
}

app.use(express.static(__dirname, { index: false, fallthrough: true }));

app.get('*', (_req, res, next) => {
  const fallbackCandidates = [
    path.join(DIST_DIR, 'index.html'),
    path.join(__dirname, 'index.html')
  ];

  const existing = fallbackCandidates.find((file) => fs.existsSync(file));

  if (!existing) {
    return next();
  }

  res.sendFile(existing);
});

app.use((_req, res) => {
  res.status(503).json({
    error: 'UI build missing',
    hint: 'Run `npm install` and `npm run build` before starting the UI service.'
  });
});

function ensureUiBundle() {
  if (fs.existsSync(DIST_INDEX)) {
    return;
  }

  console.warn('[qr-code-generator-ui] Missing production bundle, building UI...');
  const result = spawnSync('npm', ['run', 'build'], {
    cwd: __dirname,
    stdio: 'inherit',
    env: process.env
  });

  if (result.status !== 0) {
    throw new Error('Failed to build QR Code Generator UI bundle');
  }
}

async function proxyToApi(req, res, upstreamPath) {
  const normalizedPath = normalizeUpstreamPath(upstreamPath ?? req.originalUrl ?? req.url ?? '/');
  const targetUrl = new URL(normalizedPath, API_TARGET);

  try {
    const method = (req.method || 'GET').toUpperCase();
    const headers = buildProxyHeaders(req.headers);
    const init = { method, headers, redirect: 'manual' };

    if (shouldForwardBody(method) && req.readable !== false) {
      const bodyBuffer = await readRequestBody(req);
      if (bodyBuffer.length > 0) {
        init.body = bodyBuffer;
      }
    }

    const upstreamResponse = await fetch(targetUrl, init);

    upstreamResponse.headers.forEach((value, key) => {
      const lower = key.toLowerCase();
      if (lower === 'content-length' || lower === 'transfer-encoding') {
        return;
      }
      res.setHeader(key, value);
    });

    res.status(upstreamResponse.status);

    if (upstreamResponse.body) {
      if (typeof Readable.fromWeb === 'function') {
        Readable.fromWeb(upstreamResponse.body).pipe(res);
        return;
      }

      const buffer = Buffer.from(await upstreamResponse.arrayBuffer());
      res.send(buffer);
      return;
    }

    res.end();
  } catch (error) {
    console.error('[qr-code-generator-ui] API proxy error', error);
    if (!res.headersSent) {
      res.status(502).json({
        error: 'API proxy error',
        details: error.message,
        target: targetUrl.toString()
      });
    } else {
      res.end();
    }
  }
}

function buildProxyHeaders(source = {}) {
  const headers = new Headers();
  Object.entries(source).forEach(([key, value]) => {
    if (!value) {
      return;
    }

    const lower = key.toLowerCase();
    if (lower === 'host' || lower === 'content-length' || lower === 'connection') {
      return;
    }

    if (Array.isArray(value)) {
      value.filter(Boolean).forEach((entry) => headers.append(key, entry));
      return;
    }

    headers.append(key, value);
  });

  headers.set('x-forwarded-host', process.env.UI_HOST || `${API_HOST}:${PORT}`);
  headers.set('x-forwarded-proto', API_PROTOCOL);
  headers.set('x-forwarded-port', String(API_PORT));
  return headers;
}

function shouldForwardBody(method) {
  return !['GET', 'HEAD'].includes(method);
}

function normalizeUpstreamPath(value) {
  if (!value || typeof value !== 'string') {
    return '/';
  }

  if (value.startsWith('http://') || value.startsWith('https://')) {
    try {
      const url = new URL(value);
      return `${url.pathname}${url.search}` || '/';
    } catch (error) {
      console.warn('[qr-code-generator-ui] Unable to normalize upstream URL', error.message);
      return '/';
    }
  }

  if (value.startsWith('/')) {
    return value;
  }

  return `/${value}`;
}

function readRequestBody(req) {
  return new Promise((resolve, reject) => {
    const chunks = [];
    req.on('data', (chunk) => chunks.push(chunk));
    req.on('end', () => resolve(Buffer.concat(chunks)));
    req.on('error', reject);
  });
}

let server;

if (require.main === module) {
  server = app.listen(PORT, () => {
    const displayHost = process.env.UI_HOST || API_HOST;
    console.log(`✨ QR Code Generator UI running on http://${displayHost}:${PORT}`);
    console.log(`🔀 Proxying API traffic to ${API_TARGET}`);
    if (!fs.existsSync(path.join(DIST_DIR, 'index.html'))) {
      console.warn('⚠️  No production build found in ./dist. Run `npm run build` to generate the UI bundle.');
    }
  });

  process.on('SIGTERM', () => {
    if (server) {
      server.close(() => {
        console.log('[qr-code-generator-ui] UI server stopped');
      });
    }
  });
}

app.proxyToApi = proxyToApi;

module.exports = { app, proxyToApi };
