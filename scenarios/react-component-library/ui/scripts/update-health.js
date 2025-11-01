#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { performance } = require('perf_hooks');

const API_URL = process.env.API_URL || 'http://localhost:15092';
const HEALTH_URL = `${API_URL.replace(/\/$/, '')}/health`;
const outputPath = path.join(__dirname, '..', 'public', 'health.json');

const controller = new AbortController();
const timeout = setTimeout(() => controller.abort(), 3000);

async function checkApiHealth() {
  const started = performance.now();
  try {
    const response = await fetch(HEALTH_URL, {
      method: 'GET',
      headers: { Accept: 'application/json' },
      signal: controller.signal,
    });
    clearTimeout(timeout);

    const latencyMs = Math.round(performance.now() - started);

    if (!response.ok) {
      return {
        connected: false,
        latency_ms: null,
        error: {
          code: `HTTP_${response.status}`,
          message: `API health responded with status ${response.status}`,
          category: 'network',
          retryable: response.status >= 500,
        },
      };
    }

    return {
      connected: true,
      latency_ms: latencyMs,
      error: null,
    };
  } catch (error) {
    const latencyMs = Math.round(performance.now() - started);
    const code = error.name === 'AbortError' ? 'TIMEOUT' : (error.code || 'CONNECTION_ERROR');
    return {
      connected: false,
      latency_ms: null,
      error: {
        code,
        message: error.message || 'Failed to reach API health endpoint',
        category: 'network',
        retryable: true,
        details: { latency_ms: latencyMs },
      },
    };
  }
}

(async function generateHealthPayload() {
  const nowIso = new Date().toISOString();
  const connectivity = await checkApiHealth();

  const payload = {
    status: connectivity.connected ? 'healthy' : 'degraded',
    service: 'react-component-library-ui',
    timestamp: nowIso,
    readiness: true,
    api_connectivity: {
      connected: connectivity.connected,
      api_url: API_URL,
      last_check: nowIso,
      latency_ms: connectivity.latency_ms,
      error: connectivity.error,
    },
  };

  fs.writeFileSync(outputPath, `${JSON.stringify(payload, null, 2)}\n`, 'utf8');
})();
