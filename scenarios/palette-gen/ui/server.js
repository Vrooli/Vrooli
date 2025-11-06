const express = require('express');
const path = require('path');
const fs = require('fs');
const { createProxyMiddleware } = require('http-proxy-middleware');

const app = express();

// Fail fast if critical environment variables are missing
if (!process.env.UI_PORT) {
    console.error('ERROR: UI_PORT environment variable is required');
    process.exit(1);
}
if (!process.env.API_PORT) {
    console.error('ERROR: API_PORT environment variable is required');
    process.exit(1);
}

const PORT = process.env.UI_PORT;
const API_PORT = process.env.API_PORT;
const API_URL = process.env.API_URL || `http://localhost:${API_PORT}`;
const DIST_DIR = path.join(__dirname, 'dist');
const STATIC_ROOT = fs.existsSync(DIST_DIR) ? DIST_DIR : __dirname;
if (STATIC_ROOT !== DIST_DIR) {
    console.warn('[Palette Gen UI] dist bundle missing or stale. Run `npm run build` so lifecycle.setup.condition can verify ui/dist/index.html.');
}

// Serve static files

// Health check endpoint for orchestrator
app.get('/health', async (req, res) => {
    const timestamp = new Date().toISOString();

    // Check API connectivity
    let apiConnectivity = {
        connected: false,
        api_url: API_URL,
        last_check: timestamp,
        error: null,
        latency_ms: null
    };

    try {
        const startTime = Date.now();
        const response = await fetch(`${API_URL}/health`);
        const latency = Date.now() - startTime;

        if (response.ok) {
            apiConnectivity.connected = true;
            apiConnectivity.latency_ms = latency;
        } else {
            apiConnectivity.error = {
                code: 'API_UNHEALTHY',
                message: `API returned status ${response.status}`,
                category: 'resource',
                retryable: true
            };
        }
    } catch (error) {
        apiConnectivity.error = {
            code: 'CONNECTION_FAILED',
            message: error.message,
            category: 'network',
            retryable: true
        };
    }

    res.json({
        status: apiConnectivity.connected ? 'healthy' : 'degraded',
        service: 'palette-gen-ui',
        timestamp,
        readiness: true,
        api_connectivity: apiConnectivity
    });
});

app.use(express.static(STATIC_ROOT));

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
    res.sendFile(path.join(STATIC_ROOT, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`🎨 Palette Gen UI running on http://localhost:${PORT}`);
});
