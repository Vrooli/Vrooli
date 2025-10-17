const express = require('express');
const path = require('path');
const { createProxyMiddleware } = require('http-proxy-middleware');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_PORT = process.env.API_PORT || '16917';
const API_URL = process.env.API_URL || `http://localhost:${API_PORT}`;

// Serve static files

// Health check endpoint for orchestrator
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy',
        scenario: 'palette-gen',
        port: PORT,
        timestamp: new Date().toISOString()
    });
});

app.use(express.static(__dirname));

// Proxy API requests
app.use('/api', createProxyMiddleware({
    target: API_URL,
    changeOrigin: true,
    pathRewrite: {
        '^/api': ''
    }
}));

// Serve index.html for all routes
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`🎨 Palette Gen UI running on http://localhost:${PORT}`);
});