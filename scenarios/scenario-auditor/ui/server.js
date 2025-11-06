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
      `${name} environment variable is required. Run the scenario through the lifecycle system.`,
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

const PORT = parsePort(requireEnv('UI_PORT'), 'UI_PORT');
const API_PORT = parsePort(requireEnv('API_PORT'), 'API_PORT');
const API_HOST = process.env.API_HOST || '127.0.0.1';
const API_TARGET = `http://${API_HOST}:${API_PORT}`;

// Health check endpoint
app.get('/health', (_req, res) => {
  res.json({
    status: 'healthy',
    service: 'scenario-auditor-ui',
    timestamp: new Date().toISOString(),
  });
});

// API proxy for frontend requests
app.use('/api', async (req, res) => {
  const apiUrl = `${API_TARGET}${req.url}`;
  try {
    const fetch = (await import('node-fetch')).default;
    const response = await fetch(apiUrl, {
      method: req.method,
      headers: req.headers,
      body: req.method !== 'GET' && req.method !== 'HEAD' ? JSON.stringify(req.body) : undefined,
    });
    const data = await response.text();
    res.status(response.status).send(data);
  } catch (error) {
    console.error('[Proxy] API error:', error.message);
    res.status(502).json({ error: 'API server unavailable', details: error.message });
  }
});

// Serve static files from dist
const distDir = path.join(__dirname, 'dist');
app.use(express.static(distDir));

// SPA fallback - catch all routes not matched above
app.use((_req, res) => {
  res.sendFile(path.join(distDir, 'index.html'));
});

const server = http.createServer(app);

server.listen(PORT, () => {
  console.log(`Scenario Auditor UI available at http://localhost:${PORT}`);
  console.log(`Proxying API requests to ${API_TARGET}`);
});
