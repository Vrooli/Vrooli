import express from 'express';

const app = express();

if (!process.env.UI_PORT) {
  console.error('UI_PORT environment variable must be set');
  process.exit(1);
}
if (!process.env.API_PORT) {
  console.error('API_PORT environment variable must be set');
  process.exit(1);
}

const PORT = process.env.UI_PORT;
const API_PORT = process.env.API_PORT;
const API_URL = `http://localhost:${API_PORT}`;

const startTime = Date.now();

app.get('/health', async (req, res) => {
  const health = {
    status: 'healthy',
    service: 'prd-control-tower-ui',
    timestamp: new Date().toISOString(),
    uptime: (Date.now() - startTime) / 1000,
    readiness: true,
    api_connectivity: {
      connected: false,
      api_url: `${API_URL}/api/v1`,
      last_check: new Date().toISOString(),
      error: null,
      latency_ms: null
    }
  };

  try {
    const start = Date.now();
    const response = await fetch(`${API_URL}/api/v1/health`, {
      signal: AbortSignal.timeout(3000)
    });
    const latency = Date.now() - start;

    if (response.ok) {
      health.api_connectivity.connected = true;
      health.api_connectivity.latency_ms = latency;
    } else {
      health.status = 'degraded';
      health.api_connectivity.error = `API returned status ${response.status}`;
      health.readiness = false;
    }
  } catch (error) {
    health.status = 'degraded';
    health.api_connectivity.error = error.message;
    health.readiness = false;
  }

  res.status(health.status === 'healthy' ? 200 : 503).json(health);
});

app.listen(PORT, () => {
  console.log(`PRD Control Tower UI health server listening on port ${PORT}`);
  console.log(`Health endpoint: http://localhost:${PORT}/health`);
  console.log(`API URL: ${API_URL}`);
});
