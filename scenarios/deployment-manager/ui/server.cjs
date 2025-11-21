#!/usr/bin/env node
'use strict';

/**
 * Production static file server for deployment-manager UI
 * Serves built assets from ui/dist/ with health check support
 */

const express = require('express');
const path = require('path');
const fs = require('fs');

const PORT = process.env.UI_PORT || 3000;
const DIST_DIR = path.join(__dirname, 'dist');
const INDEX_PATH = path.join(DIST_DIR, 'index.html');

// Validate dist directory exists
if (!fs.existsSync(DIST_DIR)) {
  console.error('❌ ui/dist/ directory not found. Run setup lifecycle first to build UI.');
  process.exit(1);
}

if (!fs.existsSync(INDEX_PATH)) {
  console.error('❌ ui/dist/index.html not found. Run setup lifecycle first to build UI.');
  process.exit(1);
}

const app = express();

// Health check endpoint (required by lifecycle)
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    service: 'Deployment Manager UI',
    version: '1.0.0',
    readiness: true,
    timestamp: new Date().toISOString()
  });
});

// Serve static assets with caching
app.use(express.static(DIST_DIR, {
  maxAge: '1h',
  etag: true,
  lastModified: true
}));

// SPA fallback - all non-file requests go to index.html
app.get('*', (req, res) => {
  res.sendFile(INDEX_PATH);
});

app.listen(PORT, '0.0.0.0', () => {
  console.log(`✅ Deployment Manager UI serving production bundle on http://localhost:${PORT}`);
  console.log(`📁 Serving from: ${DIST_DIR}`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('Received SIGTERM, shutting down gracefully');
  process.exit(0);
});

process.on('SIGINT', () => {
  console.log('Received SIGINT, shutting down gracefully');
  process.exit(0);
});
