// Simple health check server for UI
import http from 'http';
import axios from 'axios';

const UI_PORT = process.env.UI_PORT || 3400;
const API_PORT = process.env.API_PORT || 3300;
const API_URL = `http://localhost:${API_PORT}`;

const server = http.createServer(async (req, res) => {
  // Only handle /health endpoint
  if (req.url === '/health' || req.url === '/ui/health') {
    res.setHeader('Content-Type', 'application/json');

    // Check API connectivity
    let apiConnected = false;
    let apiLatency = null;
    let apiError = null;
    const apiCheckStart = Date.now();

    try {
      const response = await axios.get(`${API_URL}/health`, { timeout: 5000 });
      apiLatency = Date.now() - apiCheckStart;
      apiConnected = response.status === 200;
    } catch (error) {
      apiLatency = Date.now() - apiCheckStart;
      apiError = {
        code: error.code || 'CONNECTION_ERROR',
        message: error.message,
        category: 'network',
        retryable: true
      };
    }

    const healthResponse = {
      status: apiConnected ? 'healthy' : 'degraded',
      service: 'smart-shopping-assistant-ui',
      timestamp: new Date().toISOString(),
      readiness: true,
      api_connectivity: {
        connected: apiConnected,
        api_url: `${API_URL}/health`,
        last_check: new Date().toISOString(),
        latency_ms: apiLatency,
        error: apiError
      }
    };

    res.writeHead(200);
    res.end(JSON.stringify(healthResponse, null, 2));
  } else {
    // Return 404 for any other endpoint
    res.writeHead(404);
    res.end(JSON.stringify({ error: 'Not found' }));
  }
});

// Start health check server on a different port
const HEALTH_PORT = parseInt(UI_PORT) + 1;
server.listen(HEALTH_PORT, () => {
  console.log(`UI Health server listening on port ${HEALTH_PORT}`);
});
