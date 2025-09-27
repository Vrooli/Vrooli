import express from 'express';
import path from 'path';
import { fileURLToPath } from 'url';
import http from 'http';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT || 4173;
const API_PORT = process.env.API_PORT || 8090;

function proxyToApi(req, res, upstreamPath) {
  const options = {
    hostname: 'localhost',
    port: Number(API_PORT),
    path: upstreamPath || req.url,
    method: req.method,
    headers: {
      ...req.headers,
      host: `localhost:${API_PORT}`,
    },
  };

  const proxyReq = http.request(options, (proxyRes) => {
    res.status(proxyRes.statusCode || 500);
    Object.entries(proxyRes.headers).forEach(([key, value]) => {
      if (value !== undefined) {
        res.setHeader(key, value);
      }
    });
    proxyRes.pipe(res);
  });

  proxyReq.on('error', (err) => {
    console.error('API proxy error:', err.message);
    res.status(502).json({
      error: 'API server unavailable',
      details: err.message,
      target: `http://localhost:${API_PORT}${upstreamPath}`,
    });
  });

  if (req.method !== 'GET' && req.method !== 'HEAD') {
    req.pipe(proxyReq);
  } else {
    proxyReq.end();
  }
}

app.get('/health', (req, res) => {
  proxyToApi(req, res, '/health');
});

app.use('/api', (req, res) => {
  const fullApiPath = req.url.startsWith('/api') ? req.url : `/api${req.url}`;
  proxyToApi(req, res, fullApiPath);
});

const distDir = path.join(__dirname, 'dist');
app.use(express.static(distDir));

app.get('*', (_req, res) => {
  res.sendFile(path.join(distDir, 'index.html'));
});

app.listen(PORT, () => {
  console.log(`App Issue Tracker UI available at http://localhost:${PORT}`);
  console.log(`Proxying API requests to http://localhost:${API_PORT}`);
});
