import express from 'express';
import path from 'path';
import { fileURLToPath } from 'url';
import http from 'http';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const app = express();

function requireEnv(name) {
  const value = process.env[name];
  if (!value) {
    throw new Error(
      `${name} environment variable is required. Run the scenario via the lifecycle system so configuration is injected automatically.`,
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

function proxyToApi(req, res, upstreamPath) {
  const options = {
    hostname: 'localhost',
    port: API_PORT,
    path: upstreamPath,
    method: req.method,
    headers: {
      ...req.headers,
      host: `localhost:${API_PORT}`,
    },
  };

  const proxyReq = http.request(options, (proxyRes) => {
    res.status(proxyRes.statusCode || 502);
    for (const [key, value] of Object.entries(proxyRes.headers)) {
      if (value !== undefined) {
        res.setHeader(key, value);
      }
    }
    proxyRes.pipe(res);
  });

  proxyReq.on('error', (err) => {
    console.error('[chart-generator-ui] API proxy error:', err.message);
    res.status(502).json({
      error: 'API server unavailable',
      details: err.message,
      target: `http://localhost:${API_PORT}${upstreamPath}`,
    });
  });

  if (req.method === 'GET' || req.method === 'HEAD') {
    proxyReq.end();
  } else {
    req.pipe(proxyReq);
  }
}

app.get('/health', (req, res) => {
  const start = Date.now();
  const apiUrl = `http://localhost:${API_PORT}/health`;
  const upstream = http.request(
    {
      hostname: 'localhost',
      port: API_PORT,
      path: '/health',
      method: 'GET',
    },
    (proxyRes) => {
      const latency = Date.now() - start;
      const healthy = proxyRes.statusCode === 200;
      res.status(200).json({
        status: healthy ? 'healthy' : 'degraded',
        service: 'chart-generator-ui',
        readiness: true,
        timestamp: new Date().toISOString(),
        api_connectivity: {
          connected: healthy,
          api_url: apiUrl,
          last_check: new Date().toISOString(),
          latency_ms: healthy ? latency : null,
          error: null,
        },
      });
    },
  );

  upstream.on('error', (err) => {
    res.status(200).json({
      status: 'degraded',
      service: 'chart-generator-ui',
      readiness: true,
      timestamp: new Date().toISOString(),
      api_connectivity: {
        connected: false,
        api_url: apiUrl,
        last_check: new Date().toISOString(),
        latency_ms: null,
        error: {
          code: 'CONNECTION_FAILED',
          message: err.message,
          category: 'network',
          retryable: true,
        },
      },
    });
  });

  upstream.end();
});

app.use('/api', (req, res) => {
  const fullPath = req.url.startsWith('/api') ? req.url : `/api${req.url}`;
  proxyToApi(req, res, fullPath);
});

const distDir = path.join(__dirname, 'dist');
app.use(express.static(distDir));

app.get('*', (_req, res) => {
  res.sendFile(path.join(distDir, 'index.html'));
});

app.listen(PORT, () => {
  console.log(`Chart Generator UI ready at http://localhost:${PORT}`);
  console.log(`Proxying API traffic to http://localhost:${API_PORT}`);
});
