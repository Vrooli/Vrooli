const express = require('express');
const path = require('path');
const fs = require('fs');
const cors = require('cors');
const { initIframeBridgeChild } = require('@vrooli/iframe-bridge/child');
require('dotenv').config();

if (typeof window !== 'undefined' && window.parent !== window) {
  initIframeBridgeChild({ appId: 'token-economy-ui' });
}

const app = express();
const PORT = process.env.PORT_TOKEN_ECONOMY_UI || 11081;

// Middleware
app.use(cors());

const staticRoot = (() => {
  const distPath = path.join(__dirname, 'dist');
  return fs.existsSync(distPath) ? distPath : __dirname;
})();

app.use(express.static(staticRoot));

// API proxy configuration (if needed)
const API_URL = process.env.TOKEN_ECONOMY_API_URL || 'http://localhost:11080';

// Serve index.html for all routes (SPA)
app.get('*', (req, res) => {
    res.sendFile(path.join(staticRoot, 'index.html'));
});

// Start server
app.listen(PORT, '0.0.0.0', () => {
    console.log(`Token Economy UI running on http://localhost:${PORT}`);
    console.log(`Serving static assets from ${staticRoot}`);
    console.log(`API URL: ${API_URL}`);
});
