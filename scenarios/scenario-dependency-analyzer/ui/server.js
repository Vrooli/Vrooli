#!/usr/bin/env node
/**
 * Simple HTTP server for scenario-dependency-analyzer UI with health endpoint
 */
const express = require('express');
const path = require('path');
const fs = require('fs');

const PORT = parseInt(process.env.UI_PORT || '36897', 10);
const API_PORT = parseInt(process.env.API_PORT || '15533', 10);

const app = express();

// Determine serve directory (prefer dist if it exists)
const serveRoot = __dirname;
const distDir = path.join(serveRoot, 'dist');
const serveDir = fs.existsSync(distDir) ? distDir : serveRoot;

// Health check endpoint
app.get('/health', async (req, res) => {
  // Check if API is accessible
  let apiConnected = false;
  try {
    const response = await fetch(`http://localhost:${API_PORT}/health`, {
      signal: AbortSignal.timeout(2000)
    });
    apiConnected = response.ok;
  } catch (error) {
    apiConnected = false;
  }

  const status = apiConnected ? 'healthy' : 'degraded';

  const healthData = {
    status,
    service: 'scenario-dependency-analyzer-ui',
    timestamp: new Date().toISOString(),
    readiness: true, // UI is always ready once started
    api_connectivity: {
      connected: apiConnected,
      api_url: `http://localhost:${API_PORT}`,
      error: apiConnected ? null : {
        code: 'CONNECTION_REFUSED',
        message: 'Unable to connect to API service',
        category: 'network',
        retryable: true
      }
    }
  };

  res.json(healthData);
});

// Serve static files
app.use(express.static(serveDir));

// SPA fallback - serve index.html for all other routes
app.get('*', (req, res) => {
  const indexPath = path.join(serveDir, 'index.html');
  if (fs.existsSync(indexPath)) {
    res.sendFile(indexPath);
  } else {
    res.status(404).send('Not found');
  }
});

app.listen(PORT, () => {
  console.log(`🌐 UI Server running at http://localhost:${PORT}`);
  console.log(`📊 API endpoint: http://localhost:${API_PORT}`);
});
