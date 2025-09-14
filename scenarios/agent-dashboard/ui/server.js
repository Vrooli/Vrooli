const express = require('express');
const path = require('path');
const cors = require('cors');
const { createProxyMiddleware } = require('http-proxy-middleware');

const app = express();
const UI_PORT = process.env.UI_PORT || process.env.PORT;
const API_PORT = process.env.API_PORT;

// Middleware
app.use(cors());
app.use(express.json());

// Proxy API requests to the API server
if (API_PORT) {
    app.use('/api', createProxyMiddleware({
        target: `http://localhost:${API_PORT}`,
        changeOrigin: true,
        pathRewrite: {
            '^/api': '/api'
        }
    }));
}

app.use(express.static(__dirname));

// Serve the main page
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'agent-dashboard-ui' });
});

// Start server
app.listen(UI_PORT, () => {
    console.log(`Agent Dashboard UI running on http://localhost:${UI_PORT}`);
    console.log(`Environment: ${process.env.NODE_ENV || 'development'}`);
    if (API_PORT) {
        console.log(`Proxying /api requests to http://localhost:${API_PORT}`);
    }
});