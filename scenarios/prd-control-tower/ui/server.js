import express from 'express';
import http from 'http';
import path from 'path';
import fs from 'fs';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const app = express();

function requireEnv(name) {
  const value = process.env[name];
  if (!value) {
    throw new Error(`${name} environment variable must be set by the lifecycle system.`);
  }
  return value;
}

function parsePort(raw, label) {
  const parsed = Number.parseInt(raw, 10);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    throw new Error(`${label} must be a positive integer (received: ${raw}).`);
  }
  return parsed;
}

const PORT = parsePort(requireEnv('UI_PORT'), 'UI_PORT');
const API_PORT = parsePort(requireEnv('API_PORT'), 'API_PORT');

function proxyToApi(req, res, upstreamPath) {
  const targetPath = upstreamPath || req.originalUrl || req.url;
  const options = {
    hostname: '127.0.0.1',
    port: API_PORT,
    path: targetPath,
    method: req.method,
    headers: {
      ...req.headers,
      host: `127.0.0.1:${API_PORT}`
    }
  };

  const proxyReq = http.request(options, (proxyRes) => {
    res.status(proxyRes.statusCode ?? 500);
    Object.entries(proxyRes.headers).forEach(([key, value]) => {
      if (value !== undefined) {
        res.setHeader(key, value);
      }
    });
    proxyRes.pipe(res);
  });

  proxyReq.on('error', (error) => {
    console.error('proxyToApi error:', error.message);
    res.status(502).json({
      error: 'API server unavailable',
      details: error.message,
      target: `http://127.0.0.1:${API_PORT}${targetPath}`
    });
  });

  if (req.method === 'GET' || req.method === 'HEAD') {
    proxyReq.end();
  } else {
    req.pipe(proxyReq);
  }
}

export { proxyToApi };

app.get('/health', (req, res) => {
  const startTime = Date.now();
  const apiUrl = `http://127.0.0.1:${API_PORT}/api/v1/health`;
  let responseSent = false;

  const sendResponse = (responseData) => {
    if (!responseSent) {
      responseSent = true;
      res.json(responseData);
    }
  };

  const options = {
    hostname: '127.0.0.1',
    port: API_PORT,
    path: '/api/v1/health',
    method: 'GET',
    timeout: 3000
  };

  const healthReq = http.request(options, (healthRes) => {
    const latency = Date.now() - startTime;
    const connected = healthRes.statusCode === 200;

    sendResponse({
      status: connected ? 'healthy' : 'degraded',
      service: 'prd-control-tower-ui',
      timestamp: new Date().toISOString(),
      readiness: true,
      api_connectivity: {
        connected: connected,
        api_url: apiUrl,
        last_check: new Date().toISOString(),
        latency_ms: latency,
        error: connected ? null : {
          code: 'API_UNHEALTHY',
          message: `API returned status ${healthRes.statusCode}`,
          category: 'internal',
          retryable: true
        }
      }
    });
  });

  healthReq.on('error', (error) => {
    sendResponse({
      status: 'degraded',
      service: 'prd-control-tower-ui',
      timestamp: new Date().toISOString(),
      readiness: true,
      api_connectivity: {
        connected: false,
        api_url: apiUrl,
        last_check: new Date().toISOString(),
        latency_ms: null,
        error: {
          code: error.code || 'CONNECTION_FAILED',
          message: error.message,
          category: 'network',
          retryable: true
        }
      }
    });
  });

  healthReq.on('timeout', () => {
    healthReq.destroy();
    sendResponse({
      status: 'degraded',
      service: 'prd-control-tower-ui',
      timestamp: new Date().toISOString(),
      readiness: true,
      api_connectivity: {
        connected: false,
        api_url: apiUrl,
        last_check: new Date().toISOString(),
        latency_ms: null,
        error: {
          code: 'TIMEOUT',
          message: 'API health check timed out after 3000ms',
          category: 'network',
          retryable: true
        }
      }
    });
  });

  healthReq.end();
});

app.use('/api', (req, res) => {
  proxyToApi(req, res, req.originalUrl || req.url);
});

const distDir = path.join(__dirname, 'dist');
const indexHtml = path.join(distDir, 'index.html');

if (!fs.existsSync(indexHtml)) {
  console.warn('UI build not found at', indexHtml);
  console.warn('Run `npm run build` inside scenarios/prd-control-tower/ui to generate the production bundle.');
}

app.use(express.static(distDir, { index: false }));

app.get('*', (_req, res) => {
  if (fs.existsSync(indexHtml)) {
    res.sendFile(indexHtml);
    return;
  }
  res.status(500).send('UI assets not built. Please run the scenario build process.');
});

app.listen(PORT, () => {
  console.log(`PRD Control Tower UI available at http://localhost:${PORT}`);
  console.log(`Proxying API requests to http://127.0.0.1:${API_PORT}`);
});
